package authority

import (
	"errors"
	"fmt"
	"sort"
)

const (
	authoritySessionSchemaVersion     = 1
	authoritySessionSimulationVersion = 1
	authoritySessionConfigVersion     = 1
)

type persistedBuildingEntity struct {
	ID                  string            `json:"id"`
	Type                BuildingType      `json:"type"`
	Origin              TileCoord         `json:"origin"`
	State               BuildingState     `json:"state"`
	Level               int               `json:"level"`
	HP                  int               `json:"hp"`
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
}

type persistedHostileEntity struct {
	ID               string          `json:"id"`
	ArchetypeID      UnitArchetypeID `json:"archetype_id"`
	Name             string          `json:"name"`
	Tile             TileCoord       `json:"tile"`
	HP               int             `json:"hp"`
	VisualState      UnitVisualState `json:"visual_state"`
	TargetBuildingID *string         `json:"target_building_id,omitempty"`
	AttackCooldown   int             `json:"attack_cooldown"`
	Active           bool            `json:"active"`
}

func (s *sessionState) exportSaveEnvelope() (AuthoritySessionSaveEnvelope, error) {
	stateBlob, err := encodePersistedSessionStateProto(s.persistedState())
	if err != nil {
		return AuthoritySessionSaveEnvelope{}, err
	}

	return AuthoritySessionSaveEnvelope{
		SchemaVersion:     authoritySessionSchemaVersion,
		SimulationVersion: authoritySessionSimulationVersion,
		ConfigVersion:     authoritySessionConfigVersion,
		SessionID:         s.SessionID,
		GameTick:          s.GameTick,
		StateBlob:         stateBlob,
	}, nil
}

func restoreSessionStateFromSaveEnvelope(save AuthoritySessionSaveEnvelope) (*sessionState, error) {
	if save.SchemaVersion != authoritySessionSchemaVersion {
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
	if save.GameTick != persisted.GameTick {
		return nil, fmt.Errorf("authority session envelope game_tick %d does not match state blob game_tick %d", save.GameTick, persisted.GameTick)
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
	}

	for _, node := range persisted.ResourceNodes {
		copied := node
		state.ResourceNodes[resourceTileKey(copied.Tile)] = &resourceNodeEntity{
			Kind:              copied.Kind,
			Tile:              copied.Tile,
			State:             copied.State,
			RemainingCharges:  copied.RemainingCharges,
			MaxCharges:        copied.MaxCharges,
			RegenSeconds:      copied.RegenSeconds,
			RegenTimerSeconds: copied.RegenTimerSeconds,
		}
	}

	for _, building := range persisted.Buildings {
		copied := building
		state.Buildings[copied.ID] = &buildingEntity{
			ID:                  copied.ID,
			Type:                copied.Type,
			Origin:              copied.Origin,
			State:               copied.State,
			Level:               copied.Level,
			HP:                  copied.HP,
			MarkedForDemolition: copied.MarkedForDemolition,
			PendingAction:       copied.PendingAction,
			PendingLevel:        copied.PendingLevel,
			Supplied:            copied.Supplied,
			WorkProgressTicks:   copied.WorkProgressTicks,
			AttackCooldown:      copied.AttackCooldown,
		}
	}

	if persisted.Hostile != nil {
		state.Hostile = &hostileEntity{
			ID:               persisted.Hostile.ID,
			ArchetypeID:      persisted.Hostile.ArchetypeID,
			Name:             persisted.Hostile.Name,
			Tile:             persisted.Hostile.Tile,
			HP:               persisted.Hostile.HP,
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
	state.syncDerivedProgress()
	return state, nil
}

func (s *sessionState) persistedState() persistedSessionState {
	resourceNodes := make([]persistedResourceNodeEntity, 0, len(s.ResourceNodes))
	for _, node := range s.ResourceNodes {
		resourceNodes = append(resourceNodes, persistedResourceNodeEntity{
			Kind:              node.Kind,
			Tile:              node.Tile,
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
		VisualState:      s.Hostile.VisualState,
		TargetBuildingID: s.Hostile.TargetBuildingID,
		AttackCooldown:   s.Hostile.AttackCooldown,
		Active:           s.Hostile.Active,
	}
}
