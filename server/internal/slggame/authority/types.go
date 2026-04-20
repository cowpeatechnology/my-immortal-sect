package authority

import "encoding/json"

type ResourceKind string

const (
	ResourceSpiritWood  ResourceKind = "spirit_wood"
	ResourceSpiritStone ResourceKind = "spirit_stone"
	ResourceHerb        ResourceKind = "herb"
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

type SessionPhase string

const (
	SessionPhaseClearRuin         SessionPhase = "clear_ruin"
	SessionPhasePlaceGuardTower   SessionPhase = "place_guard_tower"
	SessionPhaseUpgradeGuardTower SessionPhase = "upgrade_guard_tower"
	SessionPhaseRaidCountdown     SessionPhase = "raid_countdown"
	SessionPhaseDefend            SessionPhase = "defend"
	SessionPhaseRecover           SessionPhase = "recover"
	SessionPhaseVictory           SessionPhase = "victory"
	SessionPhaseDefeat            SessionPhase = "defeat"
)

type SessionOutcome string

const (
	SessionOutcomeInProgress SessionOutcome = "in_progress"
	SessionOutcomeVictory    SessionOutcome = "victory"
	SessionOutcomeDefeat     SessionOutcome = "defeat"
)

type Stockpile struct {
	SpiritWood  int `json:"spirit_wood"`
	SpiritStone int `json:"spirit_stone"`
	Herb        int `json:"herb"`
}

type TileCoord struct {
	Col int `json:"col"`
	Row int `json:"row"`
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

type SessionProgressSnapshot struct {
	Phase              SessionPhase   `json:"phase"`
	Outcome            SessionOutcome `json:"outcome"`
	Objective          string         `json:"objective"`
	GuardTowerID       *string        `json:"guardTowerId"`
	RuinBuildingID     *string        `json:"ruinBuildingId"`
	FirstRaidTriggered bool           `json:"firstRaidTriggered"`
	FirstRaidResolved  bool           `json:"firstRaidResolved"`
}

type SessionSnapshot struct {
	SessionID string                  `json:"sessionId"`
	Stockpile Stockpile               `json:"stockpile"`
	Buildings []BuildingSnapshot      `json:"buildings"`
	Session   SessionProgressSnapshot `json:"session"`
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

type BootstrapSession struct {
	SessionID string
}

type GetSessionSnapshot struct {
	SessionID string
}

type ExecuteCommand struct {
	SessionID string
	Command   CommandEnvelope
}

type SessionResponse struct {
	Snapshot SessionSnapshot `json:"snapshot"`
	Result   *CommandResult  `json:"result,omitempty"`
}

type ErrorBody struct {
	Error string `json:"error"`
}
