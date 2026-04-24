package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/cowpeatechnology/my-immortal-sect/server/internal/slggame/authority"
	"github.com/cowpeatechnology/my-immortal-sect/server/internal/slggame/sect"
)

const requestTimeout = 3 * time.Second
const offlineCatchUpDay = 24 * time.Hour

type AuthorityHTTPServer struct {
	engine     *actor.Engine
	sessionPID *actor.PID
	mux        *http.ServeMux
	sectMu     sync.Mutex
	sectPIDs   map[string]*actor.PID
	savedSects map[string]sect.SnapshotReplaySavepoint
	now        func() time.Time
}

func NewAuthorityHTTPServer() (*AuthorityHTTPServer, error) {
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		return nil, err
	}

	sessionPID := engine.Spawn(authority.NewSessionActor, "m1-authority-session")
	server := &AuthorityHTTPServer{
		engine:     engine,
		sessionPID: sessionPID,
		mux:        http.NewServeMux(),
		sectPIDs:   map[string]*actor.PID{},
		savedSects: map[string]sect.SnapshotReplaySavepoint{},
		now:        time.Now,
	}
	server.routes()
	return server, nil
}

func (s *AuthorityHTTPServer) Handler() http.Handler {
	return s.withCORS(s.mux)
}

func (s *AuthorityHTTPServer) routes() {
	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	s.mux.HandleFunc("/v1/authority/m1/session/bootstrap", s.handleBootstrap)
	s.mux.HandleFunc("/v1/authority/m1/session/snapshot", s.handleSnapshot)
	s.mux.HandleFunc("/v1/authority/m1/session/command", s.handleCommand)
	s.mux.HandleFunc("/v1/authority/sect/join", s.handleJoinSect)
	s.mux.HandleFunc("/v1/authority/sect/command", s.handleSectCommand)
	s.mux.HandleFunc("/v1/authority/sect/debug/reset", s.handleSectDebugReset)
	s.mux.HandleFunc("/v1/authority/sect/debug/advance-days", s.handleSectDebugAdvanceDays)
}

func (s *AuthorityHTTPServer) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var body struct {
		SessionID   string `json:"sessionId"`
		PlayerID    string `json:"playerId"`
		PlayerToken string `json:"playerToken"`
		Mode        string `json:"mode"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	response, err := s.request(authority.BootstrapSession{
		SessionID:   body.SessionID,
		PlayerID:    body.PlayerID,
		PlayerToken: body.PlayerToken,
		Mode:        authority.SessionBootstrapMode(body.Mode),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	response, err = s.syncM1ResponseWithSectDefense(response)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *AuthorityHTTPServer) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	query := r.URL.Query()
	response, err := s.request(authority.GetSessionSnapshot{
		SessionID:   query.Get("sessionId"),
		PlayerID:    query.Get("playerId"),
		PlayerToken: query.Get("playerToken"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	response, err = s.syncM1ResponseWithSectDefense(response)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *AuthorityHTTPServer) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var body struct {
		SessionID   string                    `json:"sessionId"`
		PlayerID    string                    `json:"playerId"`
		PlayerToken string                    `json:"playerToken"`
		Command     authority.CommandEnvelope `json:"command"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	response, err := s.request(authority.ExecuteCommand{
		SessionID:   body.SessionID,
		PlayerID:    body.PlayerID,
		PlayerToken: body.PlayerToken,
		Command:     body.Command,
	})
	if err != nil {
		status := http.StatusBadRequest
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	response, err = s.syncM1ResponseWithSectDefense(response)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *AuthorityHTTPServer) request(message any) (authority.SessionResponse, error) {
	result, err := s.engine.Request(s.sessionPID, message, requestTimeout).Result()
	if err != nil {
		return authority.SessionResponse{}, err
	}
	if messageErr, ok := result.(error); ok {
		return authority.SessionResponse{}, messageErr
	}

	response, ok := result.(authority.SessionResponse)
	if !ok {
		return authority.SessionResponse{}, errors.New("unexpected actor response type")
	}
	return response, nil
}

func (s *AuthorityHTTPServer) syncM1ResponseWithSectDefense(response authority.SessionResponse) (authority.SessionResponse, error) {
	playerID := response.Identity.PlayerID
	if playerID == "" {
		playerID = "preview-player"
	}
	sessionID := response.Identity.PlayerSessionID
	if sessionID == "" {
		sessionID = response.Snapshot.SessionID
	}
	sectSnapshot, err := s.loadSectSnapshotForUser(playerID, sessionID)
	if err != nil {
		return authority.SessionResponse{}, err
	}
	synced, err := s.requestDefenseContextSync(authority.SyncExternalDefenseContext{
		SessionID: response.Snapshot.SessionID,
		Context:   buildAuthorityDefenseContextFromSectSnapshot(sectSnapshot),
	})
	if err != nil {
		return authority.SessionResponse{}, err
	}
	response.Snapshot = synced.Snapshot
	return response, nil
}

func (s *AuthorityHTTPServer) requestDefenseContextSync(message authority.SyncExternalDefenseContext) (authority.SessionResponse, error) {
	result, err := s.engine.Request(s.sessionPID, message, requestTimeout).Result()
	if err != nil {
		return authority.SessionResponse{}, err
	}
	if messageErr, ok := result.(error); ok {
		return authority.SessionResponse{}, messageErr
	}
	response, ok := result.(authority.SessionResponse)
	if !ok {
		return authority.SessionResponse{}, errors.New("unexpected authority defense sync response type")
	}
	return response, nil
}

func (s *AuthorityHTTPServer) loadSectSnapshotForUser(userID string, sessionID string) (sect.SectSnapshot, error) {
	if userID == "" {
		userID = "preview-player"
	}
	sectUserID := sect.UserID(userID)
	sectID := sect.DefaultSectIDForUser(sectUserID)
	pid := s.getOrCreateSectActor(sectID, sectUserID)
	if err := s.applySectOfflineCatchUp(string(sectID), pid, sessionID); err != nil {
		return sect.SectSnapshot{}, err
	}
	result, err := s.engine.Request(pid, sect.JoinSect{
		UserID:    sectUserID,
		SectID:    sectID,
		SessionID: sessionID,
	}, requestTimeout).Result()
	if err != nil {
		return sect.SectSnapshot{}, err
	}
	response, ok := result.(sect.JoinSectResponse)
	if !ok {
		return sect.SectSnapshot{}, errors.New("unexpected sect snapshot bridge response type")
	}
	s.touchSectActivity(string(sectID))
	return response.Snapshot, nil
}

func buildAuthorityDefenseContextFromSectSnapshot(snapshot sect.SectSnapshot) authority.ExternalDefenseContext {
	guards := authorityGuardDiscipleIDs(snapshot)
	formationStrength := authorityDefenseFormationStrength(snapshot)
	combatBonus := authorityArtifactBonus(snapshot, guards, "combat")
	injuryMitigation := authorityArtifactBonus(snapshot, guards, "injury_mitigation")
	policyDefenseBonus := authorityPolicyDefenseBonus(snapshot.State.Policies)
	omenStatus, omenText := authorityDefenseOmen(snapshot)
	sourceSummary := make([]authority.DefenseSourceSummary, 0, len(snapshot.DefenseRisk.SourceSummary))
	for _, source := range snapshot.DefenseRisk.SourceSummary {
		sourceSummary = append(sourceSummary, authority.DefenseSourceSummary{
			Source: source.Source,
			Label:  source.Label,
			Delta:  source.Delta,
		})
	}
	return authority.ExternalDefenseContext{
		RiskIntensity:         snapshot.DefenseRisk.Intensity,
		RiskMitigation:        snapshot.DefenseRisk.Mitigation,
		ThreatCurve:           authorityThreatCurve(snapshot),
		GuardDiscipleCount:    len(guards),
		DefenseFormationLevel: formationStrength,
		CombatEquipmentBonus:  combatBonus,
		InjuryMitigation:      injuryMitigation,
		PolicyDefenseBonus:    policyDefenseBonus,
		OmenStatus:            omenStatus,
		OmenText:              omenText,
		Summary: fmt.Sprintf(
			"宗门名望 %d，张力 %d，山门守卫 %d，守御阵强度 %d。",
			snapshot.State.Meta.Reputation,
			snapshot.State.Events.Tension,
			len(guards),
			formationStrength,
		),
		SourceSummary: sourceSummary,
	}
}

func authorityGuardDiscipleIDs(snapshot sect.SectSnapshot) []sect.DiscipleID {
	gate, ok := snapshot.State.Institutions.ByID[sect.InstitutionIDGate]
	if !ok {
		return nil
	}
	return append([]sect.DiscipleID(nil), gate.GatePolicy.GuardDiscipleIDs...)
}

func authorityDefenseFormationStrength(snapshot sect.SectSnapshot) int {
	total := 0
	for _, formation := range snapshot.State.Formations {
		if formation.Kind != sect.FormationKindDefense || !formation.Active || formation.Stability <= 0 {
			continue
		}
		strength := formation.Level*8 + formation.Stability/10 - formation.MaintenanceDebt*6
		if strength < 0 {
			strength = 0
		}
		if strength > 40 {
			strength = 40
		}
		total += strength
	}
	return total
}

func authorityArtifactBonus(snapshot sect.SectSnapshot, discipleIDs []sect.DiscipleID, stat string) int {
	total := 0
	for _, discipleID := range discipleIDs {
		disciple, ok := snapshot.State.Disciples[discipleID]
		if !ok {
			continue
		}
		for _, itemID := range []sect.ItemID{
			disciple.Equipment.Weapon,
			disciple.Equipment.Robe,
			disciple.Equipment.Tool,
			disciple.Equipment.Special,
		} {
			if itemID == "" {
				continue
			}
			artifact, ok := snapshot.State.Inventory.Artifacts[itemID]
			if !ok {
				continue
			}
			total += artifact.Stats[stat]
		}
	}
	return total
}

func authorityPolicyDefenseBonus(policies sect.PolicyState) int {
	bonus := 0
	switch policies.TaskPolicy {
	case sect.TaskPolicyCombat:
		bonus += 6
	case sect.TaskPolicyStable:
		bonus += 2
	case sect.TaskPolicyClosedCultivation:
		bonus -= 2
	}
	switch policies.ResourcePolicy {
	case sect.ResourcePolicyWarPreparation:
		bonus += 4
	case sect.ResourcePolicySaving:
		bonus -= 1
	}
	switch policies.CultivationPolicy {
	case sect.CultivationPolicyBreakthroughSafe:
		bonus += 1
	}
	if bonus < 0 {
		return 0
	}
	return bonus
}

func authorityThreatCurve(snapshot sect.SectSnapshot) int {
	curve := 1 + (snapshot.DefenseRisk.Intensity+snapshot.State.Events.Tension*4-snapshot.DefenseRisk.Mitigation)/24
	if curve < 1 {
		return 1
	}
	if curve > 5 {
		return 5
	}
	return curve
}

func authorityDefenseOmen(snapshot sect.SectSnapshot) (string, string) {
	var chosen *sect.SectEvent
	for _, event := range snapshot.State.Events.ActiveEvents {
		if event.Status != sect.SectEventStatusForeshadowed && event.Status != sect.SectEventStatusSeeded {
			continue
		}
		if chosen == nil || event.Severity > chosen.Severity {
			copied := event
			chosen = &copied
		}
	}
	if chosen != nil {
		if chosen.OmenText != "" {
			return string(chosen.Status), chosen.OmenText
		}
		return string(chosen.Status), chosen.Title
	}
	if snapshot.DefenseRisk.Intensity >= 70 {
		return string(sect.SectEventStatusForeshadowed), "护山灵机持续收缩，敌袭预兆已经逼近山门。"
	}
	if snapshot.DefenseRisk.Intensity >= 40 {
		return string(sect.SectEventStatusSeeded), "外界已注意到宗门动向，需提前巩固守备。"
	}
	return "steady", "山门气机尚稳，维持守卫与阵法即可。"
}

func (s *AuthorityHTTPServer) handleJoinSect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var body struct {
		UserID    string `json:"userId"`
		SectID    string `json:"sectId"`
		SessionID string `json:"sessionId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	sectID := sect.DefaultSectIDForUser(sect.UserID(body.UserID))
	if body.SectID != "" {
		sectID = sect.SectID(body.SectID)
	}
	pid := s.getOrCreateSectActor(sectID, sect.UserID(body.UserID))
	if err := s.applySectOfflineCatchUp(string(sectID), pid, body.SessionID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	result, err := s.engine.Request(pid, sect.JoinSect{
		UserID:    sect.UserID(body.UserID),
		SectID:    sectID,
		SessionID: body.SessionID,
	}, requestTimeout).Result()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	response, ok := result.(sect.JoinSectResponse)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("unexpected sect join response type"))
		return
	}
	s.touchSectActivity(string(sectID))
	writeJSON(w, http.StatusOK, response)
}

func (s *AuthorityHTTPServer) handleSectCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var body struct {
		UserID    string `json:"userId"`
		SectID    string `json:"sectId"`
		SessionID string `json:"sessionId"`
		Command   struct {
			CmdID        string          `json:"cmdId"`
			Type         string          `json:"type"`
			Payload      json.RawMessage `json:"payload"`
			ClientSeq    int64           `json:"clientSeq"`
			BaseVersion  uint64          `json:"baseVersion"`
			SentAtWallMS int64           `json:"sentAtWallMs"`
		} `json:"command"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	sectID := sect.DefaultSectIDForUser(sect.UserID(body.UserID))
	if body.SectID != "" {
		sectID = sect.SectID(body.SectID)
	}
	pid := s.getOrCreateSectActor(sectID, sect.UserID(body.UserID))
	result, err := s.engine.Request(pid, sect.SubmitCommand{
		SessionID: body.SessionID,
		Command: sect.ClientCommand{
			CmdID:        body.Command.CmdID,
			UserID:       body.UserID,
			SectID:       string(sectID),
			Type:         sect.CommandType(body.Command.Type),
			Payload:      append([]byte(nil), body.Command.Payload...),
			ClientSeq:    body.Command.ClientSeq,
			BaseVersion:  sect.Version(body.Command.BaseVersion),
			SentAtWallMS: body.Command.SentAtWallMS,
		},
	}, requestTimeout).Result()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	response, ok := result.(sect.SubmitCommandResponse)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("unexpected sect command response type"))
		return
	}
	s.appendSectReplay(string(sectID), response.DomainEvents)
	writeJSON(w, http.StatusOK, response)
}

func (s *AuthorityHTTPServer) handleSectDebugReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var body struct {
		UserID    string                      `json:"userId"`
		SectID    string                      `json:"sectId"`
		SessionID string                      `json:"sessionId"`
		Resources map[sect.ResourceKind]int64 `json:"resources"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	userID := sect.UserID(body.UserID)
	sectID := sect.DefaultSectIDForUser(userID)
	if body.SectID != "" {
		sectID = sect.SectID(body.SectID)
	}
	key := string(sectID)
	initialState := sect.NewInitialSectState(sectID, userID, "青崖宗")
	for kind, amount := range body.Resources {
		if amount < 0 {
			writeError(w, http.StatusBadRequest, errors.New("debug reset resources cannot be negative"))
			return
		}
		initialState.Resources.Stock[kind] = amount
	}
	savepoint := sect.NewSnapshotReplaySavepoint(initialState)
	savepoint.LastActiveWallUnix = s.now().Unix()

	s.sectMu.Lock()
	oldPID := s.sectPIDs[key]
	delete(s.sectPIDs, key)
	s.savedSects[key] = savepoint
	s.sectMu.Unlock()
	if oldPID != nil {
		<-s.engine.Poison(oldPID).Done()
	}

	pid := s.getOrCreateSectActor(sectID, userID)
	result, err := s.engine.Request(pid, sect.JoinSect{
		UserID:    userID,
		SectID:    sectID,
		SessionID: body.SessionID,
	}, requestTimeout).Result()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	response, ok := result.(sect.JoinSectResponse)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("unexpected sect reset join response type"))
		return
	}
	s.touchSectActivity(key)
	writeJSON(w, http.StatusOK, response)
}

func (s *AuthorityHTTPServer) handleSectDebugAdvanceDays(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var body struct {
		UserID    string `json:"userId"`
		SectID    string `json:"sectId"`
		SessionID string `json:"sessionId"`
		Days      int    `json:"days"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.Days < 1 || body.Days > 30 {
		writeError(w, http.StatusBadRequest, errors.New("days must be between 1 and 30"))
		return
	}

	sectID := sect.DefaultSectIDForUser(sect.UserID(body.UserID))
	if body.SectID != "" {
		sectID = sect.SectID(body.SectID)
	}
	pid := s.getOrCreateSectActor(sectID, sect.UserID(body.UserID))
	var lastResponse sect.AdvanceTasksOneDayResponse
	for day := 0; day < body.Days; day++ {
		result, err := s.engine.Request(pid, sect.AdvanceTasksOneDay{SessionID: body.SessionID}, requestTimeout).Result()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		response, ok := result.(sect.AdvanceTasksOneDayResponse)
		if !ok {
			writeError(w, http.StatusInternalServerError, errors.New("unexpected sect advance response type"))
			return
		}
		s.appendSectReplay(string(sectID), response.DomainEvents)
		lastResponse = response
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"daysAdvanced": body.Days,
		"sectId":       string(sectID),
		"fromVersion":  uint64(lastResponse.FromVersion),
		"toVersion":    uint64(lastResponse.ToVersion),
		"snapshot":     lastResponse.Snapshot,
	})
}

func (s *AuthorityHTTPServer) getOrCreateSectActor(sectID sect.SectID, ownerUserID sect.UserID) *actor.PID {
	s.sectMu.Lock()
	defer s.sectMu.Unlock()

	key := string(sectID)
	if pid := s.sectPIDs[key]; pid != nil {
		return pid
	}

	savepoint, ok := s.savedSects[key]
	if !ok {
		initialState := sect.NewInitialSectState(sectID, ownerUserID, "青崖宗")
		newSavepoint := sect.NewSnapshotReplaySavepoint(initialState)
		newSavepoint.LastActiveWallUnix = s.now().Unix()
		s.savedSects[key] = newSavepoint
		savepoint = s.savedSects[key]
	}
	initialState, err := savepoint.Restore()
	if err != nil {
		initialState = savepoint.Snapshot.Clone()
	}

	pid := s.engine.Spawn(func() actor.Receiver { return sect.NewSectActorWithEventLog(initialState, savepoint.EventLog) }, key)
	s.sectPIDs[key] = pid
	return pid
}

func (s *AuthorityHTTPServer) appendSectReplay(key string, events []sect.DomainEvent) {
	if len(events) == 0 {
		s.touchSectActivity(key)
		return
	}
	s.sectMu.Lock()
	savepoint, ok := s.savedSects[key]
	if !ok {
		s.sectMu.Unlock()
		return
	}
	knownEventIDs := make(map[string]bool, len(savepoint.EventLog)+len(savepoint.ReplayEvents))
	for _, event := range savepoint.EventLog {
		knownEventIDs[event.EventID] = true
	}
	for _, event := range savepoint.ReplayEvents {
		knownEventIDs[event.EventID] = true
	}
	freshEvents := make([]sect.DomainEvent, 0, len(events))
	for _, event := range events {
		if knownEventIDs[event.EventID] {
			continue
		}
		knownEventIDs[event.EventID] = true
		freshEvents = append(freshEvents, event)
	}
	if len(freshEvents) > 0 {
		savepoint.AppendReplay(freshEvents)
	}
	savepoint.LastActiveWallUnix = s.now().Unix()
	s.savedSects[key] = savepoint
	s.sectMu.Unlock()
}

func (s *AuthorityHTTPServer) applySectOfflineCatchUp(key string, pid *actor.PID, sessionID string) error {
	days := s.pendingOfflineDays(key)
	if days <= 0 {
		return nil
	}
	for index := 0; index < days; index++ {
		result, err := s.engine.Request(pid, sect.AdvanceTasksOneDay{SessionID: sessionID}, requestTimeout).Result()
		if err != nil {
			return err
		}
		response, ok := result.(sect.AdvanceTasksOneDayResponse)
		if !ok {
			return errors.New("unexpected offline catch-up response type")
		}
		s.appendSectReplay(key, response.DomainEvents)
	}
	return nil
}

func (s *AuthorityHTTPServer) pendingOfflineDays(key string) int {
	s.sectMu.Lock()
	defer s.sectMu.Unlock()

	savepoint, ok := s.savedSects[key]
	if !ok || savepoint.LastActiveWallUnix <= 0 {
		return 0
	}
	elapsed := s.now().Unix() - savepoint.LastActiveWallUnix
	if elapsed <= 0 {
		return 0
	}
	return int(elapsed / int64(offlineCatchUpDay/time.Second))
}

func (s *AuthorityHTTPServer) touchSectActivity(key string) {
	s.sectMu.Lock()
	defer s.sectMu.Unlock()

	savepoint, ok := s.savedSects[key]
	if !ok {
		return
	}
	savepoint.LastActiveWallUnix = s.now().Unix()
	s.savedSects[key] = savepoint
}

func (s *AuthorityHTTPServer) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func decodeJSON(r *http.Request, target any) error {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	if r.ContentLength == 0 {
		return nil
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, authority.ErrorBody{Error: "method_not_allowed"})
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, authority.ErrorBody{Error: err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
