package authority

import (
	"errors"
	"fmt"
	"sort"
)

const (
	authoritySessionSchemaVersion     = 2
	authoritySessionSchemaVersionV1   = 1
	authoritySessionSimulationVersion = 1
	authoritySessionConfigVersion     = 1
)

type persistedReplayEntryKind int

const (
	persistedReplayEntryKindUnspecified persistedReplayEntryKind = 0
	persistedReplayEntryKindCommand     persistedReplayEntryKind = 1
	persistedReplayEntryKindAdvanceTick persistedReplayEntryKind = 2
	persistedReplayEntryKindDefenseSync persistedReplayEntryKind = 3
)

type persistedReplayEntry struct {
	Kind           persistedReplayEntryKind
	CommandID      string
	CommandName    string
	CommandPayload []byte
	AdvanceSeconds int
	DefenseContext ExternalDefenseContext
}

type persistedCommandLogEntry struct {
	CommandID      string
	CommandName    string
	CommandPayload []byte
	Result         CommandResult
	ErrorMessage   string
}

func newPersistedCommandLogEntry(envelope CommandEnvelope, result CommandResult, err error) persistedCommandLogEntry {
	entry := persistedCommandLogEntry{
		CommandID:      envelope.CommandID,
		CommandName:    envelope.Name,
		CommandPayload: append([]byte(nil), envelope.Payload...),
		Result:         result,
	}
	if entry.Result.CommandID == "" {
		entry.Result.CommandID = envelope.CommandID
	}
	if err != nil {
		entry.ErrorMessage = err.Error()
	}
	return entry
}

func (e persistedCommandLogEntry) outcome() (CommandResult, error) {
	if e.ErrorMessage != "" {
		return CommandResult{}, errors.New(e.ErrorMessage)
	}
	return e.Result, nil
}

type persistedBuildingEntity struct {
	ID                  string            `json:"id"`
	Type                BuildingType      `json:"type"`
	Origin              TileCoord         `json:"origin"`
	State               BuildingState     `json:"state"`
	Level               int               `json:"level"`
	HP                  int               `json:"hp"`
	Durability          int               `json:"durability"`
	DamagedReason       string            `json:"damaged_reason,omitempty"`
	RepairPressure      int               `json:"repair_pressure,omitempty"`
	MarkedForDemolition bool              `json:"marked_for_demolition"`
	PendingAction       *BuildingWorkKind `json:"pending_action,omitempty"`
	PendingLevel        *int              `json:"pending_level,omitempty"`
	Supplied            Stockpile         `json:"supplied"`
	WorkProgressTicks   int64             `json:"work_progress_ticks"`
	AttackCooldown      int               `json:"attack_cooldown"`
}

type persistedResourceNodeEntity struct {
	Kind              ResourceKind      `json:"kind"`
	Tile              TileCoord         `json:"tile"`
	Designated        bool              `json:"designated"`
	State             ResourceNodeState `json:"state"`
	RemainingCharges  int               `json:"remaining_charges"`
	MaxCharges        int               `json:"max_charges"`
	RegenSeconds      int               `json:"regen_seconds"`
	RegenTimerSeconds int               `json:"regen_timer_seconds"`
}

type persistedSessionState struct {
	SessionID              string                        `json:"session_id"`
	GameTick               int64                         `json:"game_tick"`
	NextBuildingSeq        int                           `json:"next_building_seq"`
	Stockpile              Stockpile                     `json:"stockpile"`
	ResourceNodes          []persistedResourceNodeEntity `json:"resource_nodes"`
	Buildings              []persistedBuildingEntity     `json:"buildings"`
	DiscipleHP             int                           `json:"disciple_hp"`
	DiscipleAttackCooldown int                           `json:"disciple_attack_cooldown"`
	Hostile                *persistedHostileEntity       `json:"hostile,omitempty"`
	Phase                  SessionPhase                  `json:"phase"`
	Outcome                SessionOutcome                `json:"outcome"`
	Objective              string                        `json:"objective"`
	GuardTowerID           *string                       `json:"guard_tower_id,omitempty"`
	RuinBuildingID         *string                       `json:"ruin_building_id,omitempty"`
	FirstRaidTriggered     bool                          `json:"first_raid_triggered"`
	FirstRaidResolved      bool                          `json:"first_raid_resolved"`
	RaidCountdownSeconds   int                           `json:"raid_countdown_seconds"`
	DefendRemainingSeconds int                           `json:"defend_remaining_seconds"`
	ExternalDefense        ExternalDefenseContext        `json:"external_defense,omitempty"`
	LastDefenseSummary     string                        `json:"last_defense_summary,omitempty"`
	LastDamageSummary      string                        `json:"last_damage_summary,omitempty"`
}

type persistedHostileEntity struct {
	ID               string          `json:"id"`
	ArchetypeID      UnitArchetypeID `json:"archetype_id"`
	Name             string          `json:"name"`
	Tile             TileCoord       `json:"tile"`
	HP               int             `json:"hp"`
	MaxHP            int             `json:"max_hp"`
	AttackPower      int             `json:"attack_power"`
	Defense          int             `json:"defense"`
	VisualState      UnitVisualState `json:"visual_state"`
	TargetBuildingID *string         `json:"target_building_id,omitempty"`
	AttackCooldown   int             `json:"attack_cooldown"`
	Active           bool            `json:"active"`
}

func (s *sessionState) exportSaveEnvelope() (AuthoritySessionSaveEnvelope, error) {
	s.ensurePersistenceState()

	snapshotBase := s.snapshotBase
	if len(s.postSnapshotReplay) == 0 {
		snapshotBase = s.persistedState()
	}

	stateBlob, err := encodePersistedSessionStateProto(snapshotBase)
	if err != nil {
		return AuthoritySessionSaveEnvelope{}, err
	}
	replayLogBlob, err := encodePersistedReplayLogProto(s.postSnapshotReplay)
	if err != nil {
		return AuthoritySessionSaveEnvelope{}, err
	}
	commandLogBlob, err := encodePersistedCommandLogProto(s.sortedCommandLogEntries())
	if err != nil {
		return AuthoritySessionSaveEnvelope{}, err
	}

	return AuthoritySessionSaveEnvelope{
		SchemaVersion:     authoritySessionSchemaVersion,
		SimulationVersion: authoritySessionSimulationVersion,
		ConfigVersion:     authoritySessionConfigVersion,
		SessionID:         s.SessionID,
		GameTick:          s.GameTick,
		SnapshotGameTick:  snapshotBase.GameTick,
		StateBlob:         stateBlob,
		ReplayLogBlob:     replayLogBlob,
		CommandLogBlob:    commandLogBlob,
	}, nil
}

func restoreSessionStateFromSaveEnvelope(save AuthoritySessionSaveEnvelope) (*sessionState, error) {
	if save.SchemaVersion != authoritySessionSchemaVersion && save.SchemaVersion != authoritySessionSchemaVersionV1 {
		return nil, fmt.Errorf("unsupported authority session schema version: %d", save.SchemaVersion)
	}
	if save.SimulationVersion != authoritySessionSimulationVersion {
		return nil, fmt.Errorf("unsupported authority session simulation version: %d", save.SimulationVersion)
	}
	if save.ConfigVersion != authoritySessionConfigVersion {
		return nil, fmt.Errorf("unsupported authority session config version: %d", save.ConfigVersion)
	}
	if len(save.StateBlob) == 0 {
		return nil, errors.New("authority session state blob is empty")
	}

	persisted, err := decodePersistedSessionStateProto(save.StateBlob)
	if err != nil {
		return nil, err
	}

	if persisted.SessionID == "" {
		return nil, errors.New("authority session state is missing session_id")
	}
	if normalizedSessionID(save.SessionID) != normalizedSessionID(persisted.SessionID) {
		return nil, fmt.Errorf("authority session envelope session_id %q does not match state blob session_id %q", save.SessionID, persisted.SessionID)
	}
	expectedSnapshotTick := save.GameTick
	if save.SchemaVersion >= authoritySessionSchemaVersion {
		expectedSnapshotTick = save.SnapshotGameTick
	}
	if expectedSnapshotTick != persisted.GameTick {
		return nil, fmt.Errorf("authority session envelope snapshot game_tick %d does not match state blob game_tick %d", expectedSnapshotTick, persisted.GameTick)
	}

	state := &sessionState{
		SessionID:              normalizedSessionID(persisted.SessionID),
		GameTick:               persisted.GameTick,
		nextBuildingSeq:        persisted.NextBuildingSeq,
		Stockpile:              persisted.Stockpile,
		ResourceNodes:          map[string]*resourceNodeEntity{},
		Buildings:              map[string]*buildingEntity{},
		DiscipleHP:             persisted.DiscipleHP,
		DiscipleAttackCooldown: persisted.DiscipleAttackCooldown,
		Phase:                  persisted.Phase,
		Outcome:                persisted.Outcome,
		Objective:              persisted.Objective,
		GuardTowerID:           persisted.GuardTowerID,
		RuinBuildingID:         persisted.RuinBuildingID,
		FirstRaidTriggered:     persisted.FirstRaidTriggered,
		FirstRaidResolved:      persisted.FirstRaidResolved,
		RaidCountdownSeconds:   persisted.RaidCountdownSeconds,
		DefendRemainingSeconds: persisted.DefendRemainingSeconds,
		ExternalDefense:        normalizeExternalDefenseContext(persisted.ExternalDefense),
		LastDefenseSummary:     persisted.LastDefenseSummary,
		LastDamageSummary:      persisted.LastDamageSummary,
		snapshotBase:           persisted,
		postSnapshotReplay:     nil,
		commandLog:             map[string]persistedCommandLogEntry{},
	}

	for _, node := range persisted.ResourceNodes {
		copied := node
		state.ResourceNodes[resourceTileKey(copied.Tile)] = &resourceNodeEntity{
			Kind:              copied.Kind,
			Tile:              copied.Tile,
			Designated:        copied.Designated,
			State:             copied.State,
			RemainingCharges:  copied.RemainingCharges,
			MaxCharges:        copied.MaxCharges,
			RegenSeconds:      copied.RegenSeconds,
			RegenTimerSeconds: copied.RegenTimerSeconds,
		}
	}

	for _, building := range persisted.Buildings {
		copied := building
		durability := copied.Durability
		if durability <= 0 && copied.HP > 0 {
			durability = clampInt(int(float64(copied.HP)/float64(maxInt(1, getBuildingMaxHP(copied.Type, copied.Level)))*100), 1, 100)
		}
		state.Buildings[copied.ID] = &buildingEntity{
			ID:                  copied.ID,
			Type:                copied.Type,
			Origin:              copied.Origin,
			State:               copied.State,
			Level:               copied.Level,
			HP:                  copied.HP,
			Durability:          durability,
			DamagedReason:       copied.DamagedReason,
			RepairPressure:      copied.RepairPressure,
			MarkedForDemolition: copied.MarkedForDemolition,
			PendingAction:       copied.PendingAction,
			PendingLevel:        copied.PendingLevel,
			Supplied:            copied.Supplied,
			WorkProgressTicks:   copied.WorkProgressTicks,
			AttackCooldown:      copied.AttackCooldown,
		}
	}

	if persisted.Hostile != nil {
		hostileMaxHP := choosePositiveInt(persisted.Hostile.MaxHP, mustUnitArchetype(persisted.Hostile.ArchetypeID).Stats.MaxHP)
		hostileAttackPower := choosePositiveInt(persisted.Hostile.AttackPower, mustUnitArchetype(persisted.Hostile.ArchetypeID).Stats.AttackPower)
		hostileDefense := choosePositiveInt(persisted.Hostile.Defense, mustUnitArchetype(persisted.Hostile.ArchetypeID).Stats.Defense)
		state.Hostile = &hostileEntity{
			ID:               persisted.Hostile.ID,
			ArchetypeID:      persisted.Hostile.ArchetypeID,
			Name:             persisted.Hostile.Name,
			Tile:             persisted.Hostile.Tile,
			HP:               persisted.Hostile.HP,
			MaxHP:            hostileMaxHP,
			AttackPower:      hostileAttackPower,
			Defense:          hostileDefense,
			VisualState:      persisted.Hostile.VisualState,
			TargetBuildingID: persisted.Hostile.TargetBuildingID,
			AttackCooldown:   persisted.Hostile.AttackCooldown,
			Active:           persisted.Hostile.Active,
		}
	}

	if state.nextBuildingSeq < 1 {
		state.nextBuildingSeq = 1
	}
	if state.DiscipleHP <= 0 {
		state.DiscipleHP = mustUnitArchetype(UnitArchetypeSectDisciple).Stats.MaxHP
	}

	if len(save.ReplayLogBlob) > 0 {
		replayLog, err := decodePersistedReplayLogProto(save.ReplayLogBlob)
		if err != nil {
			return nil, err
		}
		for _, entry := range replayLog {
			if err := state.applyReplayEntry(entry); err != nil {
				return nil, err
			}
		}
		state.postSnapshotReplay = replayLog
	}

	if len(save.CommandLogBlob) > 0 {
		commandEntries, err := decodePersistedCommandLogProto(save.CommandLogBlob)
		if err != nil {
			return nil, err
		}
		for _, entry := range commandEntries {
			state.commandLog[entry.CommandID] = entry
		}
	}

	if save.GameTick != state.GameTick {
		return nil, fmt.Errorf("authority session envelope game_tick %d does not match restored game_tick %d", save.GameTick, state.GameTick)
	}
	state.syncDerivedProgress()
	return state, nil
}

func (s *sessionState) ensurePersistenceState() {
	if s.commandLog == nil {
		s.commandLog = map[string]persistedCommandLogEntry{}
	}
	if s.snapshotBase.SessionID == "" {
		s.snapshotBase = s.persistedState()
	}
}

func (s *sessionState) resetPersistenceBaseline() {
	s.snapshotBase = s.persistedState()
	s.postSnapshotReplay = nil
	s.commandLog = map[string]persistedCommandLogEntry{}
}

func (s *sessionState) recordReplayCommand(envelope CommandEnvelope) {
	s.postSnapshotReplay = append(s.postSnapshotReplay, persistedReplayEntry{
		Kind:           persistedReplayEntryKindCommand,
		CommandID:      envelope.CommandID,
		CommandName:    envelope.Name,
		CommandPayload: append([]byte(nil), envelope.Payload...),
	})
}

func (s *sessionState) recordReplayAdvance(seconds int) {
	s.postSnapshotReplay = append(s.postSnapshotReplay, persistedReplayEntry{
		Kind:           persistedReplayEntryKindAdvanceTick,
		AdvanceSeconds: seconds,
	})
}

func (s *sessionState) recordReplayDefenseContext(context ExternalDefenseContext) {
	s.postSnapshotReplay = append(s.postSnapshotReplay, persistedReplayEntry{
		Kind:           persistedReplayEntryKindDefenseSync,
		DefenseContext: normalizeExternalDefenseContext(context),
	})
}

func (s *sessionState) applyReplayEntry(entry persistedReplayEntry) error {
	switch entry.Kind {
	case persistedReplayEntryKindCommand:
		_, err := s.executeCommandOnce(CommandEnvelope{
			CommandID: entry.CommandID,
			Name:      entry.CommandName,
			Payload:   append([]byte(nil), entry.CommandPayload...),
		})
		return err
	case persistedReplayEntryKindAdvanceTick:
		s.advanceResourceNodes(entry.AdvanceSeconds)
		if len(s.postSnapshotReplay) > 0 {
			s.postSnapshotReplay = s.postSnapshotReplay[:len(s.postSnapshotReplay)-1]
		}
		return nil
	case persistedReplayEntryKindDefenseSync:
		beforeLen := len(s.postSnapshotReplay)
		s.syncExternalDefenseContext(entry.DefenseContext)
		if len(s.postSnapshotReplay) > beforeLen {
			s.postSnapshotReplay = s.postSnapshotReplay[:len(s.postSnapshotReplay)-1]
		}
		return nil
	default:
		return fmt.Errorf("unsupported replay entry kind: %d", entry.Kind)
	}
}

func (s *sessionState) sortedCommandLogEntries() []persistedCommandLogEntry {
	entries := make([]persistedCommandLogEntry, 0, len(s.commandLog))
	for _, entry := range s.commandLog {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CommandID < entries[j].CommandID
	})
	return entries
}

func (s *sessionState) persistedState() persistedSessionState {
	resourceNodes := make([]persistedResourceNodeEntity, 0, len(s.ResourceNodes))
	for _, node := range s.ResourceNodes {
		resourceNodes = append(resourceNodes, persistedResourceNodeEntity{
			Kind:              node.Kind,
			Tile:              node.Tile,
			Designated:        node.Designated,
			State:             node.State,
			RemainingCharges:  node.RemainingCharges,
			MaxCharges:        node.MaxCharges,
			RegenSeconds:      node.RegenSeconds,
			RegenTimerSeconds: node.RegenTimerSeconds,
		})
	}
	sort.Slice(resourceNodes, func(i, j int) bool {
		return resourceTileKey(resourceNodes[i].Tile) < resourceTileKey(resourceNodes[j].Tile)
	})

	buildings := make([]persistedBuildingEntity, 0, len(s.Buildings))
	for _, building := range s.Buildings {
		buildings = append(buildings, persistedBuildingEntity{
			ID:                  building.ID,
			Type:                building.Type,
			Origin:              building.Origin,
			State:               building.State,
			Level:               building.Level,
			HP:                  building.HP,
			Durability:          building.Durability,
			DamagedReason:       building.DamagedReason,
			RepairPressure:      building.RepairPressure,
			MarkedForDemolition: building.MarkedForDemolition,
			PendingAction:       building.PendingAction,
			PendingLevel:        building.PendingLevel,
			Supplied:            building.Supplied,
			WorkProgressTicks:   building.WorkProgressTicks,
			AttackCooldown:      building.AttackCooldown,
		})
	}
	sort.Slice(buildings, func(i, j int) bool {
		return buildings[i].ID < buildings[j].ID
	})

	return persistedSessionState{
		SessionID:              s.SessionID,
		GameTick:               s.GameTick,
		NextBuildingSeq:        s.nextBuildingSeq,
		Stockpile:              s.Stockpile,
		ResourceNodes:          resourceNodes,
		Buildings:              buildings,
		DiscipleHP:             s.DiscipleHP,
		DiscipleAttackCooldown: s.DiscipleAttackCooldown,
		Hostile:                s.persistedHostile(),
		Phase:                  s.Phase,
		Outcome:                s.Outcome,
		Objective:              s.Objective,
		GuardTowerID:           s.GuardTowerID,
		RuinBuildingID:         s.RuinBuildingID,
		FirstRaidTriggered:     s.FirstRaidTriggered,
		FirstRaidResolved:      s.FirstRaidResolved,
		RaidCountdownSeconds:   s.RaidCountdownSeconds,
		DefendRemainingSeconds: s.DefendRemainingSeconds,
		ExternalDefense:        normalizeExternalDefenseContext(s.ExternalDefense),
		LastDefenseSummary:     s.LastDefenseSummary,
		LastDamageSummary:      s.LastDamageSummary,
	}
}

func (s *sessionState) persistedHostile() *persistedHostileEntity {
	if s.Hostile == nil {
		return nil
	}
	return &persistedHostileEntity{
		ID:               s.Hostile.ID,
		ArchetypeID:      s.Hostile.ArchetypeID,
		Name:             s.Hostile.Name,
		Tile:             s.Hostile.Tile,
		HP:               s.Hostile.HP,
		MaxHP:            s.Hostile.MaxHP,
		AttackPower:      s.Hostile.AttackPower,
		Defense:          s.Hostile.Defense,
		VisualState:      s.Hostile.VisualState,
		TargetBuildingID: s.Hostile.TargetBuildingID,
		AttackCooldown:   s.Hostile.AttackCooldown,
		Active:           s.Hostile.Active,
	}
}
