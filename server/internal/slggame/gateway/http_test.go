package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/cowpeatechnology/my-immortal-sect/server/internal/slggame/authority"
	"github.com/cowpeatechnology/my-immortal-sect/server/internal/slggame/sect"
)

func TestBootstrapCanRestoreLatestSavedSessionAndReset(t *testing.T) {
	server, err := NewAuthorityHTTPServer()
	if err != nil {
		t.Fatalf("create authority http server: %v", err)
	}

	preview := httptest.NewServer(server.Handler())
	defer preview.Close()

	resetResponse := postBootstrap(t, preview.URL, authority.SessionBootstrapModeReset)
	if resetResponse.Snapshot.SessionID != "preview-player/preview-local" {
		t.Fatalf("expected preview-player/preview-local session, got %q", resetResponse.Snapshot.SessionID)
	}
	if resetResponse.Identity.PlayerID != "preview-player" {
		t.Fatalf("expected preview-player identity, got %+v", resetResponse.Identity)
	}
	if resetResponse.Identity.PlayerToken == "" {
		t.Fatalf("expected bootstrap to return a player token")
	}
	if resetResponse.Snapshot.GameTick != 0 {
		t.Fatalf("expected clean reset to start at game tick 0, got %d", resetResponse.Snapshot.GameTick)
	}
	if resetResponse.Snapshot.Stockpile.SpiritWood != 0 {
		t.Fatalf("expected clean reset stockpile to start empty, got %+v", resetResponse.Snapshot.Stockpile)
	}

	commandResponse := postCommand(t, preview.URL, map[string]any{
		"name": "collect_stockpile",
		"payload": map[string]any{
			"resourceKind": "spirit_wood",
			"amount":       1,
			"resourceTile": map[string]any{
				"col": 2,
				"row": 4,
			},
		},
	})
	if commandResponse.Snapshot.Stockpile.SpiritWood != 1 {
		t.Fatalf("expected mutated authority stockpile after collect, got %+v", commandResponse.Snapshot.Stockpile)
	}

	restoreResponse := postBootstrap(t, preview.URL, authority.SessionBootstrapModeRestoreLatest)
	if restoreResponse.Snapshot.GameTick != commandResponse.Snapshot.GameTick {
		t.Fatalf("expected restored snapshot game tick %d, got %d", commandResponse.Snapshot.GameTick, restoreResponse.Snapshot.GameTick)
	}
	if restoreResponse.Snapshot.Stockpile != commandResponse.Snapshot.Stockpile {
		t.Fatalf("expected restored stockpile %+v, got %+v", commandResponse.Snapshot.Stockpile, restoreResponse.Snapshot.Stockpile)
	}
	if len(restoreResponse.Snapshot.ResourceNodes) != len(commandResponse.Snapshot.ResourceNodes) {
		t.Fatalf("expected restored resource-node shape length %d, got %d", len(commandResponse.Snapshot.ResourceNodes), len(restoreResponse.Snapshot.ResourceNodes))
	}
	if restoreResponse.Snapshot.ResourceNodes[0] != commandResponse.Snapshot.ResourceNodes[0] {
		t.Fatalf("expected restored first resource node %+v, got %+v", commandResponse.Snapshot.ResourceNodes[0], restoreResponse.Snapshot.ResourceNodes[0])
	}

	secondResetResponse := postBootstrap(t, preview.URL, authority.SessionBootstrapModeReset)
	if secondResetResponse.Snapshot.Stockpile.SpiritWood != 0 {
		t.Fatalf("expected reset stockpile to clear authority gather progress, got %+v", secondResetResponse.Snapshot.Stockpile)
	}
	if secondResetResponse.Snapshot.GameTick != 0 {
		t.Fatalf("expected reset game tick to return to 0, got %d", secondResetResponse.Snapshot.GameTick)
	}
}

func TestSnapshotAcceptsReturnedPlayerSessionIDWithoutDoublePrefix(t *testing.T) {
	server, err := NewAuthorityHTTPServer()
	if err != nil {
		t.Fatalf("create authority http server: %v", err)
	}

	preview := httptest.NewServer(server.Handler())
	defer preview.Close()

	bootstrap := postBootstrap(t, preview.URL, authority.SessionBootstrapModeReset)
	snapshot := getSnapshot(t, preview.URL, bootstrap.Identity.PlayerSessionID, bootstrap.Identity.PlayerID, bootstrap.Identity.PlayerToken)

	if snapshot.Identity.PlayerSessionID != bootstrap.Identity.PlayerSessionID {
		t.Fatalf("expected stable player session id %q, got %q", bootstrap.Identity.PlayerSessionID, snapshot.Identity.PlayerSessionID)
	}
	if snapshot.Snapshot.SessionID != bootstrap.Snapshot.SessionID {
		t.Fatalf("expected stable snapshot session id %q, got %q", bootstrap.Snapshot.SessionID, snapshot.Snapshot.SessionID)
	}
}

func TestCommandEndpointDeduplicatesRepeatedCmdID(t *testing.T) {
	server, err := NewAuthorityHTTPServer()
	if err != nil {
		t.Fatalf("create authority http server: %v", err)
	}

	preview := httptest.NewServer(server.Handler())
	defer preview.Close()

	postBootstrap(t, preview.URL, authority.SessionBootstrapModeReset)

	command := map[string]any{
		"cmdId": "cmd-http-collect-1",
		"name":  "collect_stockpile",
		"payload": map[string]any{
			"resourceKind": "spirit_wood",
			"amount":       1,
			"resourceTile": map[string]any{
				"col": 2,
				"row": 4,
			},
		},
	}

	first := postCommand(t, preview.URL, command)
	second := postCommand(t, preview.URL, command)

	if first.Result == nil || second.Result == nil {
		t.Fatalf("expected command responses to include results")
	}
	if first.Result.CommandID != "cmd-http-collect-1" || second.Result.CommandID != "cmd-http-collect-1" {
		t.Fatalf("expected command id to round-trip through gateway, got %+v / %+v", first.Result, second.Result)
	}
	if second.Snapshot.Stockpile.SpiritWood != 1 {
		t.Fatalf("expected duplicate HTTP cmd_id to avoid duplicate side effects, got %+v", second.Snapshot.Stockpile)
	}
}

func TestCommandEndpointAcceptsResourceDesignationIntent(t *testing.T) {
	server, err := NewAuthorityHTTPServer()
	if err != nil {
		t.Fatalf("create authority http server: %v", err)
	}

	preview := httptest.NewServer(server.Handler())
	defer preview.Close()

	postBootstrap(t, preview.URL, authority.SessionBootstrapModeReset)

	response := postCommand(t, preview.URL, map[string]any{
		"cmdId": "cmd-http-designate-1",
		"name":  "set_resource_designation",
		"payload": map[string]any{
			"resourceTile": map[string]any{
				"col": 2,
				"row": 4,
			},
			"designated": false,
		},
	})

	if response.Result == nil || !response.Result.Accepted {
		t.Fatalf("expected resource designation command to be accepted, got %+v", response.Result)
	}
	for _, node := range response.Snapshot.ResourceNodes {
		if node.Tile == (authority.TileCoord{Col: 2, Row: 4}) {
			if node.Designated {
				t.Fatalf("expected resource designation to be false in authority snapshot, got %+v", node)
			}
			return
		}
	}
	t.Fatalf("expected target resource node in snapshot, got %+v", response.Snapshot.ResourceNodes)
}

func TestBuildAuthorityDefenseContextFromSectSnapshotIncludesRiskAndDefenseSources(t *testing.T) {
	state := sect.NewInitialSectState(sect.SectID("sect-defense"), sect.UserID("player-defense"), "青崖宗")
	state.Meta.Reputation = 72
	state.Events.Tension = 3

	var starterDiscipleID sect.DiscipleID
	for discipleID := range state.Disciples {
		starterDiscipleID = discipleID
		break
	}
	if starterDiscipleID == "" {
		t.Fatalf("expected starter disciple in initial sect state")
	}

	disciple := state.Disciples[starterDiscipleID]
	disciple.Equipment.Weapon = sect.ItemID("artifact-guard-sabre")
	state.Disciples[starterDiscipleID] = disciple
	state.Inventory.Artifacts[sect.ItemID("artifact-guard-sabre")] = sect.ArtifactState{
		ItemID:        sect.ItemID("artifact-guard-sabre"),
		Type:          sect.ArtifactTypeSword,
		Quality:       1,
		Durability:    20,
		MaxDurability: 20,
		Stats: map[string]int{
			"combat":            9,
			"injury_mitigation": 4,
		},
	}

	gate := state.Institutions.ByID[sect.InstitutionIDGate]
	gate.GatePolicy.GuardDiscipleIDs = []sect.DiscipleID{starterDiscipleID}
	state.Institutions.ByID[sect.InstitutionIDGate] = gate
	state.Formations[sect.BuildingID("building-defense-core")] = sect.FormationState{
		FormationID:     "formation-defense-core",
		Kind:            sect.FormationKindDefense,
		BuildingID:      sect.BuildingID("building-defense-core"),
		ArtifactItemID:  sect.ItemID("artifact-disk-defense"),
		Level:           2,
		Stability:       80,
		MaintenanceDebt: 1,
		Active:          true,
	}
	state.Policies.TaskPolicy = sect.TaskPolicyCombat
	state.Policies.ResourcePolicy = sect.ResourcePolicyWarPreparation
	state.Policies.CultivationPolicy = sect.CultivationPolicyBreakthroughSafe
	state.Events.ActiveEvents[sect.EventID("event-defense-omen")] = sect.SectEvent{
		EventID:      sect.EventID("event-defense-omen"),
		Kind:         "raid_warning",
		Status:       sect.SectEventStatusForeshadowed,
		Severity:     3,
		Title:        "山门异响",
		OmenText:     "护山灵机示警，西侧山道有敌踪。",
		ExpiresAtDay: 5,
	}

	snapshot := sect.SectSnapshot{
		SectID:       string(state.Meta.SectID),
		UserID:       string(state.Meta.OwnerUserID),
		SessionID:    "session-defense-bridge",
		SceneVersion: state.Runtime.Version,
		State:        state,
		DefenseRisk: sect.DefenseRiskProjection{
			Intensity:  58,
			Mitigation: 12,
			SourceSummary: []sect.DefenseRiskSourceSummary{
				{Source: "reputation", Label: "名望 72", Delta: 9},
				{Source: "policy", Label: "战备政策", Delta: 4},
				{Source: "defense_formation", Label: "守御阵缓冲宗门风险", Delta: -12},
			},
		},
	}

	context := buildAuthorityDefenseContextFromSectSnapshot(snapshot)
	if context.RiskIntensity != 58 || context.RiskMitigation != 12 {
		t.Fatalf("expected bridge to keep sect risk projection, got %+v", context)
	}
	if context.GuardDiscipleCount != 1 {
		t.Fatalf("expected bridge to include guard disciple count, got %+v", context)
	}
	if context.DefenseFormationLevel <= 0 {
		t.Fatalf("expected bridge to include defense formation strength, got %+v", context)
	}
	if context.CombatEquipmentBonus <= 0 || context.InjuryMitigation <= 0 {
		t.Fatalf("expected bridge to include guard artifact bonuses, got %+v", context)
	}
	if context.PolicyDefenseBonus <= 0 || context.ThreatCurve < 1 {
		t.Fatalf("expected bridge to include policy/threat projection, got %+v", context)
	}
	if context.OmenStatus != string(sect.SectEventStatusForeshadowed) || context.OmenText == "" {
		t.Fatalf("expected bridge to surface foreshadow omen, got %+v", context)
	}
	if len(context.SourceSummary) != 3 || context.Summary == "" {
		t.Fatalf("expected bridge to keep bounded risk source summary, got %+v", context)
	}
}

func TestJoinSectReturnsAuthoritativeSnapshot(t *testing.T) {
	server, err := NewAuthorityHTTPServer()
	if err != nil {
		t.Fatalf("create authority http server: %v", err)
	}

	preview := httptest.NewServer(server.Handler())
	defer preview.Close()

	response := postSectJoin(t, preview.URL, "player-alpha", "", "session-alpha")

	if response.Snapshot.SectID != "sect-player-alpha" {
		t.Fatalf("expected default sect id, got %q", response.Snapshot.SectID)
	}
	if response.Snapshot.UserID != "player-alpha" {
		t.Fatalf("expected user id round-trip, got %q", response.Snapshot.UserID)
	}
	if response.Snapshot.SceneVersion != 0 {
		t.Fatalf("expected initial scene version 0, got %d", response.Snapshot.SceneVersion)
	}
	if response.Snapshot.State.Meta.OwnerUserID != sect.UserID("player-alpha") {
		t.Fatalf("expected snapshot owner player-alpha, got %+v", response.Snapshot.State.Meta)
	}
	if response.Snapshot.State.Resources.Stock[sect.ResourceKindSpiritStone] <= 0 {
		t.Fatalf("expected authority resource snapshot, got %+v", response.Snapshot.State.Resources.Stock)
	}
	if len(response.Snapshot.State.Disciples) != 1 {
		t.Fatalf("expected authoritative starter disciple snapshot, got %+v", response.Snapshot.State.Disciples)
	}
	starter, ok := response.Snapshot.State.Disciples[sect.DiscipleID("disciple-1")]
	if !ok {
		t.Fatalf("expected starter disciple in join snapshot, got %+v", response.Snapshot.State.Disciples)
	}
	if starter.Name == "" {
		t.Fatalf("expected starter disciple name to be populated")
	}
	if starter.AssignmentKind != sect.DiscipleAssignmentIdle {
		t.Fatalf("expected join snapshot to expose authority assignment, got %+v", starter)
	}
	if starter.WorkTarget.Description == "" {
		t.Fatalf("expected join snapshot to expose authority work target, got %+v", starter.WorkTarget)
	}
	if starter.Needs.DailySpiritGrain <= 0 || starter.Needs.DailyRestTicks <= 0 {
		t.Fatalf("expected join snapshot to expose authority disciple needs, got %+v", starter.Needs)
	}
	if !starter.Support.FoodSatisfied || !starter.Support.HousingSatisfied {
		t.Fatalf("expected join snapshot to expose authority support state, got %+v", starter.Support)
	}

	rejoin := postSectJoin(t, preview.URL, "player-alpha", response.Snapshot.SectID, "session-alpha-2")
	if rejoin.Snapshot.SceneVersion != response.Snapshot.SceneVersion {
		t.Fatalf("expected rejoin to preserve scene version %d, got %d", response.Snapshot.SceneVersion, rejoin.Snapshot.SceneVersion)
	}
	if !reflect.DeepEqual(rejoin.Snapshot.State.Disciples[sect.DiscipleID("disciple-1")], starter) {
		t.Fatalf("expected rejoin to restore the same starter disciple, got %+v", rejoin.Snapshot.State.Disciples)
	}
}

func TestJoinSectAppliesOfflineCatchUpThroughAuthorityAdvancePath(t *testing.T) {
	server, err := NewAuthorityHTTPServer()
	if err != nil {
		t.Fatalf("create authority http server: %v", err)
	}

	baseTime := time.Unix(1_700_000_000, 0)
	server.now = func() time.Time { return baseTime }

	preview := httptest.NewServer(server.Handler())
	defer preview.Close()

	join := postSectJoin(t, preview.URL, "player-offline", "", "session-offline-1")
	start := postSectCommand(t, preview.URL, "player-offline", join.Snapshot.SectID, "session-offline-1", map[string]any{
		"cmdId":       "cmd-cultivation-start-1",
		"type":        string(sect.CommandTypeStartCultivation),
		"baseVersion": join.Snapshot.SceneVersion,
		"payload": map[string]any{
			"discipleId": "disciple-1",
		},
	})
	if start.Result.Status != sect.CommandResultStatusAccepted {
		t.Fatalf("expected start cultivation accepted, got %+v", start.Result)
	}

	server.now = func() time.Time { return baseTime.Add(72 * time.Hour) }
	rejoin := postSectJoin(t, preview.URL, "player-offline", join.Snapshot.SectID, "session-offline-2")
	if rejoin.Snapshot.SceneVersion <= start.Result.SceneVersion {
		t.Fatalf("expected offline catch-up to advance authority version beyond %d, got %d", start.Result.SceneVersion, rejoin.Snapshot.SceneVersion)
	}
	disciple := rejoin.Snapshot.State.Disciples[sect.DiscipleID("disciple-1")]
	if !disciple.Realm.ReadyForBreakthrough {
		t.Fatalf("expected offline catch-up to advance cultivation readiness, got %+v", disciple.Realm)
	}
	if rejoin.Snapshot.State.Resources.Stock[sect.ResourceKindSpiritStone] != 118 {
		t.Fatalf("expected offline catch-up to stop consuming spirit stone once breakthrough is ready, got %+v", rejoin.Snapshot.State.Resources.Stock)
	}
	portentCount := len(rejoin.Snapshot.State.Events.ActiveEvents)
	if portentCount == 0 {
		t.Fatalf("expected offline catch-up to reveal a fate portent through sect events, got %+v", rejoin.Snapshot.State.Events)
	}

	server.sectMu.Lock()
	savepoint := server.savedSects[join.Snapshot.SectID]
	server.sectMu.Unlock()
	if len(savepoint.EventLogAfter(1)) == 0 {
		t.Fatalf("expected offline catch-up to append day-advance events into event_log")
	}
	diary := savepoint.DiscipleDiary(sect.DiscipleID("disciple-1"))
	if len(diary) == 0 {
		t.Fatalf("expected diary replay from shared event_log after offline catch-up")
	}
}

func TestRestoreAndOfflineCatchUpMatchManualAuthorityAdvance(t *testing.T) {
	server, err := NewAuthorityHTTPServer()
	if err != nil {
		t.Fatalf("create authority http server: %v", err)
	}

	baseTime := time.Unix(1_700_100_000, 0)
	server.now = func() time.Time { return baseTime }

	preview := httptest.NewServer(server.Handler())
	defer preview.Close()

	join := postSectJoin(t, preview.URL, "player-proof", "", "session-proof-1")
	start := postSectCommand(t, preview.URL, "player-proof", join.Snapshot.SectID, "session-proof-1", map[string]any{
		"cmdId":       "cmd-cultivation-start-proof",
		"type":        string(sect.CommandTypeStartCultivation),
		"baseVersion": join.Snapshot.SceneVersion,
		"payload": map[string]any{
			"discipleId": "disciple-1",
		},
	})
	if start.Result.Status != sect.CommandResultStatusAccepted {
		t.Fatalf("expected start cultivation accepted, got %+v", start.Result)
	}

	server.sectMu.Lock()
	savepoint := server.savedSects[join.Snapshot.SectID]
	pid := server.sectPIDs[join.Snapshot.SectID]
	delete(server.sectPIDs, join.Snapshot.SectID)
	server.sectMu.Unlock()
	if pid == nil {
		t.Fatalf("expected active sect actor pid before restart")
	}
	<-server.engine.Poison(pid).Done()

	restoredState, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore savepoint before offline proof: %v", err)
	}
	manualPID := server.engine.Spawn(func() actor.Receiver { return sect.NewSectActor(restoredState) }, "sect-manual-proof")
	var manualAdvance sect.AdvanceTasksOneDayResponse
	for day := 0; day < 30; day++ {
		result, err := server.engine.Request(manualPID, sect.AdvanceTasksOneDay{SessionID: "manual-proof"}, requestTimeout).Result()
		if err != nil {
			t.Fatalf("manual offline advance day %d: %v", day+1, err)
		}
		response, ok := result.(sect.AdvanceTasksOneDayResponse)
		if !ok {
			t.Fatalf("unexpected manual offline advance response type")
		}
		manualAdvance = response
	}

	server.now = func() time.Time { return baseTime.Add(30 * 24 * time.Hour) }
	rejoin := postSectJoin(t, preview.URL, "player-proof", join.Snapshot.SectID, "session-proof-2")

	if rejoin.Snapshot.SceneVersion != manualAdvance.Snapshot.SceneVersion {
		t.Fatalf("expected offline catch-up version %d to match manual authority advance, got %d", manualAdvance.Snapshot.SceneVersion, rejoin.Snapshot.SceneVersion)
	}
	rejoinStateBlob, err := json.Marshal(rejoin.Snapshot.State)
	if err != nil {
		t.Fatalf("marshal rejoin snapshot state: %v", err)
	}
	manualStateBlob, err := json.Marshal(manualAdvance.Snapshot.State)
	if err != nil {
		t.Fatalf("marshal manual snapshot state: %v", err)
	}
	if !bytes.Equal(rejoinStateBlob, manualStateBlob) {
		t.Fatalf("expected restore+offline catch-up state to match manual authority advance\nrejoin: %s\nmanual: %s", rejoinStateBlob, manualStateBlob)
	}
	if rejoin.Snapshot.State.Monthly.LastSettledMonth != 1 {
		t.Fatalf("expected 30-day offline catch-up to include month-one settlement, got %+v", rejoin.Snapshot.State.Monthly)
	}
}

func TestEventChoiceRestoreOfflineCatchUpAndDuplicateCmdIDStayAuthorityConsistent(t *testing.T) {
	server, err := NewAuthorityHTTPServer()
	if err != nil {
		t.Fatalf("create authority http server: %v", err)
	}

	baseTime := time.Unix(1_700_200_000, 0)
	server.now = func() time.Time { return baseTime }

	preview := httptest.NewServer(server.Handler())
	defer preview.Close()

	join := postSectJoin(t, preview.URL, "player-event-proof", "", "session-event-proof-1")
	type advanceDaysResponse struct {
		DaysAdvanced int               `json:"daysAdvanced"`
		FromVersion  uint64            `json:"fromVersion"`
		ToVersion    uint64            `json:"toVersion"`
		Snapshot     sect.SectSnapshot `json:"snapshot"`
	}
	advance := postJSON[advanceDaysResponse](t, preview.URL+"/v1/authority/sect/debug/advance-days", map[string]any{
		"userId":    "player-event-proof",
		"sectId":    join.Snapshot.SectID,
		"sessionId": "session-event-proof-advance",
		"days":      2,
	})
	if advance.DaysAdvanced != 2 {
		t.Fatalf("expected event proof advance to run two days, got %+v", advance)
	}
	event := firstAuthorityChoiceEvent(advance.Snapshot.State)
	if event == nil || len(event.Options) < 2 {
		t.Fatalf("expected authority choice event with options, got %+v", advance.Snapshot.State.Events)
	}
	beforeStone := advance.Snapshot.State.Resources.Stock[sect.ResourceKindSpiritStone]

	command := map[string]any{
		"cmdId":       "cmd-event-choice-http-idempotent",
		"type":        string(sect.CommandTypeChooseEventOption),
		"baseVersion": advance.Snapshot.SceneVersion,
		"payload": map[string]any{
			"eventId":  event.EventID,
			"optionId": "send_aid",
		},
	}
	choice := postSectCommand(t, preview.URL, "player-event-proof", join.Snapshot.SectID, "session-event-proof-1", command)
	if choice.Result.Status != sect.CommandResultStatusAccepted {
		t.Fatalf("expected event choice accepted, got %+v", choice.Result)
	}
	if got := choice.Snapshot.State.Resources.Stock[sect.ResourceKindSpiritStone]; got != beforeStone-10 {
		t.Fatalf("expected authority event choice to deduct 10 spirit stone, got %d from %d", got, beforeStone)
	}
	duplicate := postSectCommand(t, preview.URL, "player-event-proof", join.Snapshot.SectID, "session-event-proof-1", command)
	if duplicate.Result.Status != sect.CommandResultStatusAccepted || duplicate.Snapshot.SceneVersion != choice.Snapshot.SceneVersion {
		t.Fatalf("expected duplicate event choice cmd_id to return cached accepted result, first=%+v duplicate=%+v", choice.Result, duplicate.Result)
	}
	if got := duplicate.Snapshot.State.Resources.Stock[sect.ResourceKindSpiritStone]; got != beforeStone-10 {
		t.Fatalf("duplicate event choice must not reapply resource side effects, got %d from %d", got, beforeStone)
	}

	server.sectMu.Lock()
	savepoint := server.savedSects[join.Snapshot.SectID]
	pid := server.sectPIDs[join.Snapshot.SectID]
	delete(server.sectPIDs, join.Snapshot.SectID)
	server.sectMu.Unlock()
	if pid == nil {
		t.Fatalf("expected active sect actor pid before event proof restart")
	}
	<-server.engine.Poison(pid).Done()

	restoredState, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore event choice savepoint after duplicate cmd_id: %v", err)
	}
	if len(restoredState.Events.ActiveEvents) != 0 || len(restoredState.Events.ResolvedEvents) == 0 {
		t.Fatalf("expected restored event choice state to preserve resolved event and no active choice, got %+v", restoredState.Events)
	}
	manualPID := server.engine.Spawn(func() actor.Receiver {
		return sect.NewSectActorWithEventLog(restoredState, savepoint.EventLog)
	}, "sect-event-manual-proof")
	var manualAdvance sect.AdvanceTasksOneDayResponse
	for day := 0; day < 5; day++ {
		result, err := server.engine.Request(manualPID, sect.AdvanceTasksOneDay{SessionID: "session-event-manual-proof"}, requestTimeout).Result()
		if err != nil {
			t.Fatalf("manual event offline advance day %d: %v", day+1, err)
		}
		response, ok := result.(sect.AdvanceTasksOneDayResponse)
		if !ok {
			t.Fatalf("unexpected manual event offline advance response type")
		}
		manualAdvance = response
	}

	server.now = func() time.Time { return baseTime.Add(5 * 24 * time.Hour) }
	rejoin := postSectJoin(t, preview.URL, "player-event-proof", join.Snapshot.SectID, "session-event-proof-2")
	if rejoin.Snapshot.SceneVersion != manualAdvance.Snapshot.SceneVersion {
		t.Fatalf("expected event restore+offline version %d to match manual branch, got %d", manualAdvance.Snapshot.SceneVersion, rejoin.Snapshot.SceneVersion)
	}
	rejoinStateBlob, err := json.Marshal(rejoin.Snapshot.State)
	if err != nil {
		t.Fatalf("marshal event proof rejoin state: %v", err)
	}
	manualStateBlob, err := json.Marshal(manualAdvance.Snapshot.State)
	if err != nil {
		t.Fatalf("marshal event proof manual state: %v", err)
	}
	if !bytes.Equal(rejoinStateBlob, manualStateBlob) {
		t.Fatalf("expected event restore+offline state to match manual authority branch\nrejoin: %s\nmanual: %s", rejoinStateBlob, manualStateBlob)
	}
	assertJSONEqual(t, "event_log", rejoin.Snapshot.EventLog, manualAdvance.Snapshot.EventLog)
	assertJSONEqual(t, "event_summaries", rejoin.Snapshot.EventSummaries, manualAdvance.Snapshot.EventSummaries)
	assertJSONEqual(t, "diary", rejoin.Snapshot.Diary, manualAdvance.Snapshot.Diary)
	if len(rejoin.Snapshot.EventSummaries) == 0 {
		t.Fatalf("expected event choice/offline proof to expose event_log-derived feedback")
	}
}

func TestPolicyRestoreOfflineCatchUpMatchesManualAuthorityBranch(t *testing.T) {
	server, err := NewAuthorityHTTPServer()
	if err != nil {
		t.Fatalf("create authority http server: %v", err)
	}

	baseTime := time.Unix(1_700_300_000, 0)
	server.now = func() time.Time { return baseTime }

	preview := httptest.NewServer(server.Handler())
	defer preview.Close()

	join := postSectJoin(t, preview.URL, "player-policy-proof", "", "session-policy-proof-1")
	resourcePolicy := postSectCommand(t, preview.URL, "player-policy-proof", join.Snapshot.SectID, "session-policy-proof-1", map[string]any{
		"cmdId":       "cmd-policy-proof-resource",
		"type":        string(sect.CommandTypeSetPolicy),
		"baseVersion": join.Snapshot.SceneVersion,
		"payload": map[string]any{
			"policy_category": string(sect.PolicyCategoryResource),
			"policy_value":    string(sect.ResourcePolicySaving),
		},
	})
	if resourcePolicy.Result.Status != sect.CommandResultStatusAccepted {
		t.Fatalf("expected resource policy accepted, got %+v", resourcePolicy.Result)
	}
	duplicateResourcePolicy := postSectCommand(t, preview.URL, "player-policy-proof", join.Snapshot.SectID, "session-policy-proof-1", map[string]any{
		"cmdId":       "cmd-policy-proof-resource",
		"type":        string(sect.CommandTypeSetPolicy),
		"baseVersion": join.Snapshot.SceneVersion,
		"payload": map[string]any{
			"policy_category": string(sect.PolicyCategoryResource),
			"policy_value":    string(sect.ResourcePolicySaving),
		},
	})
	if duplicateResourcePolicy.Result.Status != sect.CommandResultStatusAccepted ||
		duplicateResourcePolicy.Snapshot.SceneVersion != resourcePolicy.Snapshot.SceneVersion {
		t.Fatalf("expected duplicate SetPolicy cmd_id to return cached result without extra side effects, first=%+v duplicate=%+v", resourcePolicy.Result, duplicateResourcePolicy.Result)
	}
	if !sectFeedbackHasCategory(resourcePolicy.Snapshot.EventSummaries, "policy") {
		t.Fatalf("expected policy feedback from event_log after policy command, got %+v", resourcePolicy.Snapshot.EventSummaries)
	}
	cultivationPolicy := postSectCommand(t, preview.URL, "player-policy-proof", join.Snapshot.SectID, "session-policy-proof-1", map[string]any{
		"cmdId":       "cmd-policy-proof-cultivation",
		"type":        string(sect.CommandTypeSetPolicy),
		"baseVersion": resourcePolicy.Snapshot.SceneVersion,
		"payload": map[string]any{
			"policy_category": string(sect.PolicyCategoryCultivation),
			"policy_value":    string(sect.CultivationPolicyClosedCultivation),
		},
	})
	if cultivationPolicy.Result.Status != sect.CommandResultStatusAccepted {
		t.Fatalf("expected cultivation policy accepted, got %+v", cultivationPolicy.Result)
	}

	server.sectMu.Lock()
	savepoint := server.savedSects[join.Snapshot.SectID]
	pid := server.sectPIDs[join.Snapshot.SectID]
	delete(server.sectPIDs, join.Snapshot.SectID)
	server.sectMu.Unlock()
	if pid == nil {
		t.Fatalf("expected active sect actor pid before policy proof restart")
	}
	<-server.engine.Poison(pid).Done()

	restoredState, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore policy savepoint after commands: %v", err)
	}
	manualPID := server.engine.Spawn(func() actor.Receiver {
		return sect.NewSectActorWithEventLog(restoredState, savepoint.EventLog)
	}, "sect-policy-manual-proof")
	var manualAdvance sect.AdvanceTasksOneDayResponse
	for day := 0; day < 30; day++ {
		result, err := server.engine.Request(manualPID, sect.AdvanceTasksOneDay{SessionID: "session-policy-manual-proof"}, requestTimeout).Result()
		if err != nil {
			t.Fatalf("manual policy offline advance day %d: %v", day+1, err)
		}
		response, ok := result.(sect.AdvanceTasksOneDayResponse)
		if !ok {
			t.Fatalf("unexpected manual policy offline advance response type")
		}
		manualAdvance = response
	}

	server.now = func() time.Time { return baseTime.Add(30 * 24 * time.Hour) }
	rejoin := postSectJoin(t, preview.URL, "player-policy-proof", join.Snapshot.SectID, "session-policy-proof-2")
	if rejoin.Snapshot.SceneVersion != manualAdvance.Snapshot.SceneVersion {
		t.Fatalf("expected policy restore+offline version %d to match manual branch, got %d", manualAdvance.Snapshot.SceneVersion, rejoin.Snapshot.SceneVersion)
	}
	rejoinStateBlob, err := json.Marshal(rejoin.Snapshot.State)
	if err != nil {
		t.Fatalf("marshal policy proof rejoin state: %v", err)
	}
	manualStateBlob, err := json.Marshal(manualAdvance.Snapshot.State)
	if err != nil {
		t.Fatalf("marshal policy proof manual state: %v", err)
	}
	if !bytes.Equal(rejoinStateBlob, manualStateBlob) {
		t.Fatalf("expected policy restore+offline state to match manual authority branch\nrejoin: %s\nmanual: %s", rejoinStateBlob, manualStateBlob)
	}
	if rejoin.Snapshot.State.Policies.ResourcePolicy != sect.ResourcePolicySaving ||
		rejoin.Snapshot.State.Policies.CultivationPolicy != sect.CultivationPolicyClosedCultivation {
		t.Fatalf("expected policy state to survive restore/offline, got %+v", rejoin.Snapshot.State.Policies)
	}
	if rejoin.Snapshot.State.Monthly.LastSettlement.SatisfactionDeltaTotal >= 0 {
		t.Fatalf("expected saving policy to contribute monthly satisfaction pressure, got %+v", rejoin.Snapshot.State.Monthly.LastSettlement)
	}
}

func TestSectDebugResetAndAdvanceDaysUseAuthorityReplayPath(t *testing.T) {
	server, err := NewAuthorityHTTPServer()
	if err != nil {
		t.Fatalf("create authority http server: %v", err)
	}

	preview := httptest.NewServer(server.Handler())
	defer preview.Close()

	join := postSectJoin(t, preview.URL, "player-gate", "", "session-gate-1")
	start := postSectCommand(t, preview.URL, "player-gate", join.Snapshot.SectID, "session-gate-1", map[string]any{
		"cmdId":       "cmd-gate-cultivation-start",
		"type":        string(sect.CommandTypeStartCultivation),
		"baseVersion": join.Snapshot.SceneVersion,
		"payload": map[string]any{
			"discipleId": "disciple-1",
		},
	})
	if start.Result.Status != sect.CommandResultStatusAccepted {
		t.Fatalf("expected start cultivation accepted, got %+v", start.Result)
	}

	var resetResponse sect.JoinSectResponse = postJSON[sect.JoinSectResponse](t, preview.URL+"/v1/authority/sect/debug/reset", map[string]any{
		"userId":    "player-gate",
		"sectId":    join.Snapshot.SectID,
		"sessionId": "session-gate-reset",
	})
	if resetResponse.Snapshot.SceneVersion != 0 {
		t.Fatalf("expected debug reset to return clean sect version 0, got %d", resetResponse.Snapshot.SceneVersion)
	}
	if resetResponse.Snapshot.State.Disciples[sect.DiscipleID("disciple-1")].AssignmentKind != sect.DiscipleAssignmentIdle {
		t.Fatalf("expected debug reset to clear cultivation assignment, got %+v", resetResponse.Snapshot.State.Disciples)
	}

	type advanceDaysResponse struct {
		DaysAdvanced int               `json:"daysAdvanced"`
		FromVersion  uint64            `json:"fromVersion"`
		ToVersion    uint64            `json:"toVersion"`
		Snapshot     sect.SectSnapshot `json:"snapshot"`
	}
	advance := postJSON[advanceDaysResponse](t, preview.URL+"/v1/authority/sect/debug/advance-days", map[string]any{
		"userId":    "player-gate",
		"sectId":    join.Snapshot.SectID,
		"sessionId": "session-gate-advance",
		"days":      30,
	})
	if advance.DaysAdvanced != 30 || advance.ToVersion <= advance.FromVersion {
		t.Fatalf("expected debug day advance to mutate authority versions, got %+v", advance)
	}
	if advance.Snapshot.State.Monthly.LastSettledMonth != 1 {
		t.Fatalf("expected debug day advance to use monthly authority settlement, got %+v", advance.Snapshot.State.Monthly)
	}

	server.sectMu.Lock()
	savepoint := server.savedSects[join.Snapshot.SectID]
	server.sectMu.Unlock()
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore debug advance savepoint: %v", err)
	}
	advanceStateBlob, err := json.Marshal(advance.Snapshot.State)
	if err != nil {
		t.Fatalf("marshal debug advance state: %v", err)
	}
	restoredStateBlob, err := json.Marshal(restored)
	if err != nil {
		t.Fatalf("marshal restored debug advance state: %v", err)
	}
	if !bytes.Equal(advanceStateBlob, restoredStateBlob) {
		t.Fatalf("expected debug day advance response to match saved replay state\nadvance: %s\nrestored: %s", advanceStateBlob, restoredStateBlob)
	}
}

func TestSectCommandBuildBuildingReturnsVersionedResultAndPersistsState(t *testing.T) {
	server, err := NewAuthorityHTTPServer()
	if err != nil {
		t.Fatalf("create authority http server: %v", err)
	}

	preview := httptest.NewServer(server.Handler())
	defer preview.Close()

	join := postSectJoin(t, preview.URL, "player-builder", "", "session-build")
	command := postSectCommand(t, preview.URL, "player-builder", join.Snapshot.SectID, "session-build", map[string]any{
		"cmdId":       "cmd-build-1",
		"type":        string(sect.CommandTypeBuildBuilding),
		"baseVersion": 0,
		"payload": map[string]any{
			"definitionKey": "main_hall",
			"origin": map[string]any{
				"col": 4,
				"row": 6,
			},
		},
	})

	if command.Result.Status != sect.CommandResultStatusAccepted {
		t.Fatalf("expected accepted result, got %+v", command.Result)
	}
	if command.Result.SceneVersion != 2 {
		t.Fatalf("expected scene version 2 after resource+building events, got %d", command.Result.SceneVersion)
	}
	if command.Result.Patch.FromVersion != 0 || command.Result.Patch.ToVersion != 2 {
		t.Fatalf("expected versioned patch 0->2, got %+v", command.Result.Patch)
	}
	if len(command.Result.Events) != 2 || command.Result.Events[0].Type != sect.ClientEventTypeResourceChanged {
		t.Fatalf("expected resource change event before building event, got %+v", command.Result.Events)
	}
	building, ok := command.Snapshot.State.Buildings[sect.BuildingID("building-1")]
	if !ok {
		t.Fatalf("expected first building in snapshot, got %+v", command.Snapshot.State.Buildings)
	}
	if building.DefinitionKey != "main_hall" || building.Phase != "active" {
		t.Fatalf("expected built building state, got %+v", building)
	}
	if command.Snapshot.State.Resources.Stock[sect.ResourceKindSpiritStone] != 90 {
		t.Fatalf("expected spirit stone 90 after authority resource change, got %+v", command.Snapshot.State.Resources.Stock)
	}
	if command.Snapshot.State.Resources.Stock[sect.ResourceKindOre] != 30 {
		t.Fatalf("expected ore 30 after authority resource change, got %+v", command.Snapshot.State.Resources.Stock)
	}
	server.sectMu.Lock()
	savepoint := server.savedSects[join.Snapshot.SectID]
	server.sectMu.Unlock()
	savedState, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore saved replay state: %v", err)
	}
	if savedState.Runtime.Version != 2 {
		t.Fatalf("expected saved sect version 2, got %d", savedState.Runtime.Version)
	}
	if _, ok := savedState.Buildings[sect.BuildingID("building-1")]; !ok {
		t.Fatalf("expected saved sect state to contain building-1, got %+v", savedState.Buildings)
	}
	if savedState.Resources.Stock[sect.ResourceKindSpiritStone] != 90 {
		t.Fatalf("expected saved sect resource state to persist spirit stone cost, got %+v", savedState.Resources.Stock)
	}

	rejoin := postSectJoin(t, preview.URL, "player-builder", join.Snapshot.SectID, "session-build-2")
	if rejoin.Snapshot.SceneVersion != 2 {
		t.Fatalf("expected saved scene version 2 on rejoin, got %d", rejoin.Snapshot.SceneVersion)
	}
	if _, ok := rejoin.Snapshot.State.Buildings[sect.BuildingID("building-1")]; !ok {
		t.Fatalf("expected saved building to survive rejoin, got %+v", rejoin.Snapshot.State.Buildings)
	}
}

func TestUpgradeBuildingRestoresFromSnapshotReplayAfterActorRestart(t *testing.T) {
	server, err := NewAuthorityHTTPServer()
	if err != nil {
		t.Fatalf("create authority http server: %v", err)
	}

	preview := httptest.NewServer(server.Handler())
	defer preview.Close()

	join := postSectJoin(t, preview.URL, "player-upgrader", "", "session-build")
	build := postSectCommand(t, preview.URL, "player-upgrader", join.Snapshot.SectID, "session-build", map[string]any{
		"cmdId":       "cmd-build-1",
		"type":        string(sect.CommandTypeBuildBuilding),
		"baseVersion": 0,
		"payload": map[string]any{
			"definitionKey": "main_hall",
			"origin": map[string]any{
				"col": 4,
				"row": 6,
			},
		},
	})
	if build.Result.SceneVersion != 2 {
		t.Fatalf("expected build version 2, got %d", build.Result.SceneVersion)
	}

	server.sectMu.Lock()
	pid := server.sectPIDs[join.Snapshot.SectID]
	delete(server.sectPIDs, join.Snapshot.SectID)
	server.sectMu.Unlock()
	if pid == nil {
		t.Fatalf("expected live sect actor pid")
	}
	<-server.engine.Poison(pid).Done()

	rejoined := postSectJoin(t, preview.URL, "player-upgrader", join.Snapshot.SectID, "session-rejoin")
	if rejoined.Snapshot.SceneVersion != 2 {
		t.Fatalf("expected restored replay version 2, got %d", rejoined.Snapshot.SceneVersion)
	}
	if rejoined.Snapshot.State.Buildings[sect.BuildingID("building-1")].Level != 1 {
		t.Fatalf("expected restored building level 1 before upgrade, got %+v", rejoined.Snapshot.State.Buildings)
	}

	upgrade := postSectCommand(t, preview.URL, "player-upgrader", join.Snapshot.SectID, "session-rejoin", map[string]any{
		"cmdId":       "cmd-upgrade-1",
		"type":        string(sect.CommandTypeUpgradeBuilding),
		"baseVersion": 2,
		"payload": map[string]any{
			"buildingId": "building-1",
		},
	})
	if upgrade.Result.Status != sect.CommandResultStatusAccepted {
		t.Fatalf("expected accepted upgrade result, got %+v", upgrade.Result)
	}
	if upgrade.Result.SceneVersion != 4 {
		t.Fatalf("expected version 4 after restored upgrade, got %d", upgrade.Result.SceneVersion)
	}
	if upgrade.Snapshot.State.Buildings[sect.BuildingID("building-1")].Level != 2 {
		t.Fatalf("expected building level 2 after restored upgrade, got %+v", upgrade.Snapshot.State.Buildings)
	}
	if upgrade.Snapshot.State.Resources.Stock[sect.ResourceKindSpiritStone] != 45 {
		t.Fatalf("expected spirit stone 45 after build+upgrade, got %+v", upgrade.Snapshot.State.Resources.Stock)
	}
}

func postBootstrap(t *testing.T, baseURL string, mode authority.SessionBootstrapMode) authority.SessionResponse {
	t.Helper()
	return postJSON[authority.SessionResponse](t, baseURL+"/v1/authority/m1/session/bootstrap", map[string]any{
		"sessionId": "preview-local",
		"playerId":  "preview-player",
		"mode":      mode,
	})
}

func postSectJoin(t *testing.T, baseURL string, userID string, sectID string, sessionID string) sect.JoinSectResponse {
	t.Helper()
	return postJSON[sect.JoinSectResponse](t, baseURL+"/v1/authority/sect/join", map[string]any{
		"userId":    userID,
		"sectId":    sectID,
		"sessionId": sessionID,
	})
}

func postSectCommand(t *testing.T, baseURL string, userID string, sectID string, sessionID string, command map[string]any) sect.SubmitCommandResponse {
	t.Helper()
	return postJSON[sect.SubmitCommandResponse](t, baseURL+"/v1/authority/sect/command", map[string]any{
		"userId":    userID,
		"sectId":    sectID,
		"sessionId": sessionID,
		"command":   command,
	})
}

func postCommand(t *testing.T, baseURL string, command map[string]any) authority.SessionResponse {
	t.Helper()
	return postJSON[authority.SessionResponse](t, baseURL+"/v1/authority/m1/session/command", map[string]any{
		"sessionId": "preview-local",
		"playerId":  "preview-player",
		"command":   command,
	})
}

func getSnapshot(t *testing.T, baseURL string, sessionID string, playerID string, playerToken string) authority.SessionResponse {
	t.Helper()

	request, err := http.NewRequest(
		http.MethodGet,
		baseURL+"/v1/authority/m1/session/snapshot?sessionId="+url.QueryEscape(sessionID)+"&playerId="+url.QueryEscape(playerID)+"&playerToken="+url.QueryEscape(playerToken),
		nil,
	)
	if err != nil {
		t.Fatalf("build snapshot request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("do snapshot request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var errorBody authority.ErrorBody
		_ = json.NewDecoder(response.Body).Decode(&errorBody)
		t.Fatalf("expected status 200, got %d (%s)", response.StatusCode, errorBody.Error)
	}

	var decoded authority.SessionResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode snapshot response: %v", err)
	}
	return decoded
}

func postJSON[T any](t *testing.T, url string, payload map[string]any) T {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var errorBody authority.ErrorBody
		_ = json.NewDecoder(response.Body).Decode(&errorBody)
		t.Fatalf("expected status 200, got %d (%s)", response.StatusCode, errorBody.Error)
	}

	var decoded T
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return decoded
}

func firstAuthorityChoiceEvent(state sect.SectState) *sect.SectEvent {
	for _, event := range state.Events.ActiveEvents {
		if len(event.Options) > 0 {
			entry := event
			return &entry
		}
	}
	return nil
}

func assertJSONEqual(t *testing.T, label string, left any, right any) {
	t.Helper()
	leftBlob, err := json.Marshal(left)
	if err != nil {
		t.Fatalf("marshal %s left: %v", label, err)
	}
	rightBlob, err := json.Marshal(right)
	if err != nil {
		t.Fatalf("marshal %s right: %v", label, err)
	}
	if !bytes.Equal(leftBlob, rightBlob) {
		t.Fatalf("expected %s to match\nleft: %s\nright: %s", label, leftBlob, rightBlob)
	}
}

func sectFeedbackHasCategory(entries []sect.SectEventFeedbackEntry, category string) bool {
	for _, entry := range entries {
		if entry.Category == category {
			return true
		}
	}
	return false
}
