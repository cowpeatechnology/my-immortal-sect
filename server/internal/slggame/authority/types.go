package authority

import "encoding/json"

type ResourceKind string

const (
	ResourceSpiritWood  ResourceKind = "spirit_wood"
	ResourceSpiritStone ResourceKind = "spirit_stone"
	ResourceHerb        ResourceKind = "herb"
)

type ResourceNodeState string

const (
	ResourceNodeStateAvailable    ResourceNodeState = "available"
	ResourceNodeStateRegenerating ResourceNodeState = "regenerating"
)

type BuildingType string

const (
	BuildingMainHall         BuildingType = "main_hall"
	BuildingDiscipleQuarters BuildingType = "disciple_quarters"
	BuildingWarehouse        BuildingType = "warehouse"
	BuildingHerbGarden       BuildingType = "herb_garden"
	BuildingGuardTower       BuildingType = "guard_tower"
)

type BuildingState string

const (
	BuildingStatePlanned      BuildingState = "planned"
	BuildingStateSupplied     BuildingState = "supplied"
	BuildingStateConstructing BuildingState = "constructing"
	BuildingStateActive       BuildingState = "active"
	BuildingStateDamaged      BuildingState = "damaged"
)

type BuildingWorkKind string

const (
	BuildingWorkBuild   BuildingWorkKind = "build"
	BuildingWorkUpgrade BuildingWorkKind = "upgrade"
)

type DiscipleAssignmentKind string

const (
	DiscipleAssignmentIdle     DiscipleAssignmentKind = "idle"
	DiscipleAssignmentGather   DiscipleAssignmentKind = "gather"
	DiscipleAssignmentHaul     DiscipleAssignmentKind = "haul"
	DiscipleAssignmentBuild    DiscipleAssignmentKind = "build"
	DiscipleAssignmentRepair   DiscipleAssignmentKind = "repair"
	DiscipleAssignmentGuard    DiscipleAssignmentKind = "guard"
	DiscipleAssignmentDemolish DiscipleAssignmentKind = "demolish"
)

type SessionPhase string

const (
	SessionPhaseClearRuin         SessionPhase = "clear_ruin"
	SessionPhasePlaceGuardTower   SessionPhase = "place_guard_tower"
	SessionPhaseUpgradeGuardTower SessionPhase = "upgrade_guard_tower"
	SessionPhaseRaidCountdown     SessionPhase = "raid_countdown"
	SessionPhaseDefend            SessionPhase = "defend"
	SessionPhaseRecover           SessionPhase = "recover"
	SessionPhaseSecondCycleReady  SessionPhase = "second_cycle_ready"
	SessionPhaseVictory           SessionPhase = "victory"
	SessionPhaseDefeat            SessionPhase = "defeat"
)

type SessionOutcome string

const (
	SessionOutcomeInProgress SessionOutcome = "in_progress"
	SessionOutcomeVictory    SessionOutcome = "victory"
	SessionOutcomeDefeat     SessionOutcome = "defeat"
)

type SessionRecoverReason string

const (
	SessionRecoverReasonNone                     SessionRecoverReason = "none"
	SessionRecoverReasonDamagedBuildings         SessionRecoverReason = "damaged_buildings"
	SessionRecoverReasonResourceRegeneration     SessionRecoverReason = "resource_regeneration"
	SessionRecoverReasonDamagedBuildingsAndRegen SessionRecoverReason = "damaged_buildings_and_resource_regeneration"
)

type Stockpile struct {
	SpiritWood  int `json:"spirit_wood"`
	SpiritStone int `json:"spirit_stone"`
	Herb        int `json:"herb"`
}

type UnitArchetypeID string

const (
	UnitArchetypeSectDisciple UnitArchetypeID = "sect_disciple"
	UnitArchetypeBanditScout  UnitArchetypeID = "bandit_scout"
)

type UnitVisualState string

const (
	UnitVisualStateIdle      UnitVisualState = "idle"
	UnitVisualStateMoving    UnitVisualState = "moving"
	UnitVisualStateWorking   UnitVisualState = "working"
	UnitVisualStateCarrying  UnitVisualState = "carrying"
	UnitVisualStateGuarding  UnitVisualState = "guarding"
	UnitVisualStateAttacking UnitVisualState = "attacking"
	UnitVisualStateInjured   UnitVisualState = "injured"
)

type TileCoord struct {
	Col int `json:"col"`
	Row int `json:"row"`
}

type ResourceNodeSnapshot struct {
	Tile              TileCoord         `json:"tile"`
	Kind              ResourceKind      `json:"kind"`
	State             ResourceNodeState `json:"state"`
	RemainingCharges  int               `json:"remainingCharges"`
	MaxCharges        int               `json:"maxCharges"`
	RegenSeconds      int               `json:"regenSeconds"`
	RegenTimerSeconds int               `json:"regenTimerSeconds"`
}

type BuildingSnapshot struct {
	ID                  string            `json:"id"`
	Type                BuildingType      `json:"type"`
	Origin              TileCoord         `json:"origin"`
	State               BuildingState     `json:"state"`
	Level               int               `json:"level"`
	HP                  int               `json:"hp"`
	MaxHP               int               `json:"maxHp"`
	MarkedForDemolition bool              `json:"markedForDemolition"`
	PendingAction       *BuildingWorkKind `json:"pendingAction"`
	PendingLevel        *int              `json:"pendingLevel"`
	Supplied            Stockpile         `json:"supplied"`
}

type DiscipleCarryingSnapshot struct {
	Kind   *ResourceKind `json:"kind"`
	Amount int           `json:"amount"`
}

type DiscipleSnapshot struct {
	ArchetypeID             UnitArchetypeID           `json:"archetypeId"`
	ID                     string                   `json:"id"`
	Name                   string                   `json:"name"`
	AssignmentKind         DiscipleAssignmentKind   `json:"assignmentKind"`
	TargetBuildingID       *string                  `json:"targetBuildingId"`
	TargetResourceKind     *ResourceKind            `json:"targetResourceKind"`
	TargetTile             *TileCoord               `json:"targetTile"`
	Carrying               DiscipleCarryingSnapshot `json:"carrying"`
	HP                     int                      `json:"hp"`
	MaxHP                  int                      `json:"maxHp"`
	VisualState            UnitVisualState          `json:"visualState"`
	WorkProgressTicks      int64                    `json:"workProgressTicks"`
	ExpectedNextTransition *string                  `json:"expectedNextTransition"`
}

type HostileSnapshot struct {
	ID               string          `json:"id"`
	ArchetypeID      UnitArchetypeID `json:"archetypeId"`
	Name             string          `json:"name"`
	Tile             TileCoord       `json:"tile"`
	HP               int             `json:"hp"`
	MaxHP            int             `json:"maxHp"`
	VisualState      UnitVisualState `json:"visualState"`
	Active           bool            `json:"active"`
	TargetBuildingID *string         `json:"targetBuildingId"`
}

type SessionProgressSnapshot struct {
	Phase                  SessionPhase         `json:"phase"`
	Outcome                SessionOutcome       `json:"outcome"`
	Objective              string               `json:"objective"`
	GuardTowerID           *string              `json:"guardTowerId"`
	RuinBuildingID         *string              `json:"ruinBuildingId"`
	FirstRaidTriggered     bool                 `json:"firstRaidTriggered"`
	FirstRaidResolved      bool                 `json:"firstRaidResolved"`
	RaidCountdownSeconds   int                  `json:"raidCountdownSeconds"`
	DefendRemainingSeconds int                  `json:"defendRemainingSeconds"`
	RecoverReason          SessionRecoverReason `json:"recoverReason"`
	DamagedBuildingCount   int                  `json:"damagedBuildingCount"`
	RegeneratingNodeCount  int                  `json:"regeneratingNodeCount"`
}

type SessionSnapshot struct {
	SessionID     string                  `json:"sessionId"`
	GameTick      int64                   `json:"gameTick"`
	Stockpile     Stockpile               `json:"stockpile"`
	ResourceNodes []ResourceNodeSnapshot  `json:"resourceNodes"`
	Buildings     []BuildingSnapshot      `json:"buildings"`
	Disciples     []DiscipleSnapshot      `json:"disciples"`
	Hostiles      []HostileSnapshot       `json:"hostiles"`
	Session       SessionProgressSnapshot `json:"session"`
}

type CommandEnvelope struct {
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload"`
}

type CommandResult struct {
	Accepted bool   `json:"accepted"`
	Event    string `json:"event"`
	Message  string `json:"message"`
}

type SessionBootstrapMode string

const (
	SessionBootstrapModeRestoreLatest SessionBootstrapMode = "restore_latest"
	SessionBootstrapModeReset         SessionBootstrapMode = "reset"
)

type BootstrapSession struct {
	SessionID   string
	PlayerID    string
	PlayerToken string
	Mode        SessionBootstrapMode
}

type GetSessionSnapshot struct {
	SessionID   string
	PlayerID    string
	PlayerToken string
}

type ExportSessionSave struct {
	SessionID string
}

type ImportSessionSave struct {
	Save AuthoritySessionSaveEnvelope
}

type ExecuteCommand struct {
	SessionID   string
	PlayerID    string
	PlayerToken string
	Command     CommandEnvelope
}

type PlayerIdentity struct {
	PlayerID        string `json:"playerId"`
	PlayerToken     string `json:"playerToken"`
	PlayerSessionID string `json:"playerSessionId"`
}

type SessionResponse struct {
	Identity PlayerIdentity  `json:"identity"`
	Snapshot SessionSnapshot `json:"snapshot"`
	Result   *CommandResult  `json:"result,omitempty"`
}

type ErrorBody struct {
	Error string `json:"error"`
}

type AuthoritySessionSaveEnvelope struct {
	SchemaVersion     int    `json:"schema_version"`
	SimulationVersion int    `json:"simulation_version"`
	ConfigVersion     int    `json:"config_version"`
	SessionID         string `json:"session_id"`
	GameTick          int64  `json:"game_tick"`
	StateBlob         []byte `json:"state_blob"`
}
