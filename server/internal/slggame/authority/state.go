package authority

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

var (
	initialMainHallTile = TileCoord{Col: 6, Row: 5}
	initialRuinTile     = TileCoord{Col: 8, Row: 8}
)

type buildingDefinition struct {
	ID         BuildingType
	Width      int
	Height     int
	BaseCost   Stockpile
	BaseMaxHP  int
	RepairCost Stockpile
}

var buildingDefinitions = map[BuildingType]buildingDefinition{
	BuildingMainHall: {
		ID:         BuildingMainHall,
		Width:      3,
		Height:     2,
		BaseCost:   zeroStockpile(),
		BaseMaxHP:  18,
		RepairCost: zeroStockpile(),
	},
	BuildingDiscipleQuarters: {
		ID:         BuildingDiscipleQuarters,
		Width:      2,
		Height:     2,
		BaseCost:   Stockpile{SpiritWood: 2, SpiritStone: 0, Herb: 1},
		BaseMaxHP:  12,
		RepairCost: Stockpile{SpiritWood: 1, SpiritStone: 0, Herb: 0},
	},
	BuildingWarehouse: {
		ID:         BuildingWarehouse,
		Width:      2,
		Height:     2,
		BaseCost:   Stockpile{SpiritWood: 2, SpiritStone: 1, Herb: 0},
		BaseMaxHP:  14,
		RepairCost: Stockpile{SpiritWood: 1, SpiritStone: 1, Herb: 0},
	},
	BuildingHerbGarden: {
		ID:         BuildingHerbGarden,
		Width:      2,
		Height:     1,
		BaseCost:   Stockpile{SpiritWood: 1, SpiritStone: 0, Herb: 1},
		BaseMaxHP:  10,
		RepairCost: Stockpile{SpiritWood: 1, SpiritStone: 0, Herb: 1},
	},
	BuildingGuardTower: {
		ID:         BuildingGuardTower,
		Width:      1,
		Height:     2,
		BaseCost:   Stockpile{SpiritWood: 2, SpiritStone: 1, Herb: 0},
		BaseMaxHP:  16,
		RepairCost: Stockpile{SpiritWood: 1, SpiritStone: 1, Herb: 0},
	},
}

type buildingEntity struct {
	ID                  string
	Type                BuildingType
	Origin              TileCoord
	State               BuildingState
	Level               int
	HP                  int
	MarkedForDemolition bool
	PendingAction       *BuildingWorkKind
	PendingLevel        *int
	Supplied            Stockpile
}

type sessionState struct {
	SessionID          string
	nextBuildingSeq    int
	Stockpile          Stockpile
	Buildings          map[string]*buildingEntity
	Phase              SessionPhase
	Outcome            SessionOutcome
	Objective          string
	GuardTowerID       *string
	RuinBuildingID     *string
	FirstRaidTriggered bool
	FirstRaidResolved  bool
}

func newSessionState(sessionID string) *sessionState {
	state := &sessionState{
		SessionID:       sessionID,
		nextBuildingSeq: 1,
		Stockpile:       zeroStockpile(),
		Buildings:       map[string]*buildingEntity{},
		Phase:           SessionPhaseClearRuin,
		Outcome:         SessionOutcomeInProgress,
		Objective:       "长按废弃仓房并拆除，为护山台腾出位置",
	}

	mainHall := state.addBuilding(BuildingMainHall, initialMainHallTile, BuildingStateActive)
	mainHall.Level = 1
	mainHall.HP = getBuildingMaxHP(mainHall.Type, mainHall.Level)
	mainHall.PendingAction = nil
	mainHall.PendingLevel = nil

	ruin := state.addBuilding(BuildingWarehouse, initialRuinTile, BuildingStateDamaged)
	ruin.Level = 1
	ruin.HP = 0
	ruin.PendingAction = nil
	ruin.PendingLevel = nil
	state.RuinBuildingID = stringPtr(ruin.ID)

	state.syncDerivedProgress()
	return state
}

func (s *sessionState) snapshot() SessionSnapshot {
	buildings := make([]BuildingSnapshot, 0, len(s.Buildings))
	for _, building := range s.Buildings {
		buildings = append(buildings, BuildingSnapshot{
			ID:                  building.ID,
			Type:                building.Type,
			Origin:              building.Origin,
			State:               building.State,
			Level:               building.Level,
			HP:                  building.HP,
			MaxHP:               getBuildingMaxHP(building.Type, building.Level),
			MarkedForDemolition: building.MarkedForDemolition,
			PendingAction:       building.PendingAction,
			PendingLevel:        building.PendingLevel,
			Supplied:            building.Supplied,
		})
	}

	sort.Slice(buildings, func(i, j int) bool {
		return buildings[i].ID < buildings[j].ID
	})

	return SessionSnapshot{
		SessionID: s.SessionID,
		Stockpile: s.Stockpile,
		Buildings: buildings,
		Session: SessionProgressSnapshot{
			Phase:              s.Phase,
			Outcome:            s.Outcome,
			Objective:          s.Objective,
			GuardTowerID:       s.GuardTowerID,
			RuinBuildingID:     s.RuinBuildingID,
			FirstRaidTriggered: s.FirstRaidTriggered,
			FirstRaidResolved:  s.FirstRaidResolved,
		},
	}
}

func (s *sessionState) executeCommand(envelope CommandEnvelope) (CommandResult, error) {
	switch envelope.Name {
	case "place_building":
		var payload struct {
			BuildingType BuildingType `json:"buildingType"`
			Origin       TileCoord    `json:"origin"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return CommandResult{}, err
		}
		return s.placeBuilding(payload.BuildingType, payload.Origin)
	case "request_upgrade":
		var payload struct {
			BuildingID string `json:"buildingId"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return CommandResult{}, err
		}
		return s.requestUpgrade(payload.BuildingID)
	case "toggle_demolition":
		var payload struct {
			BuildingID string `json:"buildingId"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return CommandResult{}, err
		}
		return s.toggleDemolition(payload.BuildingID)
	case "collect_stockpile":
		var payload struct {
			ResourceKind ResourceKind `json:"resourceKind"`
			Amount       int          `json:"amount"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return CommandResult{}, err
		}
		return s.collectStockpile(payload.ResourceKind, payload.Amount)
	case "deliver_build_resource":
		var payload struct {
			BuildingID   string       `json:"buildingId"`
			ResourceKind ResourceKind `json:"resourceKind"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return CommandResult{}, err
		}
		return s.deliverBuildResource(payload.BuildingID, payload.ResourceKind)
	case "start_building_work":
		var payload struct {
			BuildingID string `json:"buildingId"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return CommandResult{}, err
		}
		return s.startBuildingWork(payload.BuildingID)
	case "complete_building_work":
		var payload struct {
			BuildingID string `json:"buildingId"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return CommandResult{}, err
		}
		return s.completeBuildingWork(payload.BuildingID)
	case "complete_demolition":
		var payload struct {
			BuildingID string `json:"buildingId"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return CommandResult{}, err
		}
		return s.completeDemolition(payload.BuildingID)
	case "complete_repair":
		var payload struct {
			BuildingID string `json:"buildingId"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return CommandResult{}, err
		}
		return s.completeRepair(payload.BuildingID)
	case "sync_session_progress":
		var payload struct {
			Phase              SessionPhase   `json:"phase"`
			Outcome            SessionOutcome `json:"outcome"`
			Objective          string         `json:"objective"`
			FirstRaidTriggered bool           `json:"firstRaidTriggered"`
			FirstRaidResolved  bool           `json:"firstRaidResolved"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return CommandResult{}, err
		}
		return s.syncSessionProgress(payload.Phase, payload.Outcome, payload.Objective, payload.FirstRaidTriggered, payload.FirstRaidResolved)
	default:
		return CommandResult{}, fmt.Errorf("unsupported command: %s", envelope.Name)
	}
}

func (s *sessionState) addBuilding(buildingType BuildingType, origin TileCoord, state BuildingState) *buildingEntity {
	buildingID := fmt.Sprintf("building-%d", s.nextBuildingSeq)
	s.nextBuildingSeq++

	entity := &buildingEntity{
		ID:       buildingID,
		Type:     buildingType,
		Origin:   origin,
		State:    state,
		Level:    1,
		HP:       getBuildingMaxHP(buildingType, 1),
		Supplied: zeroStockpile(),
	}

	if state == BuildingStateActive {
		entity.PendingAction = nil
		entity.PendingLevel = nil
	} else {
		pending := BuildingWorkBuild
		entity.PendingAction = &pending
		entity.PendingLevel = intPtr(1)
	}

	s.Buildings[buildingID] = entity
	return entity
}

func (s *sessionState) placeBuilding(buildingType BuildingType, origin TileCoord) (CommandResult, error) {
	if _, ok := buildingDefinitions[buildingType]; !ok || buildingType == BuildingMainHall {
		return CommandResult{}, fmt.Errorf("invalid building type: %s", buildingType)
	}
	if s.isFootprintOccupied(buildingType, origin, "") {
		return CommandResult{}, errors.New("building footprint overlaps an existing structure")
	}

	building := s.addBuilding(buildingType, origin, BuildingStatePlanned)
	building.MarkedForDemolition = false
	if buildingType == BuildingGuardTower && s.GuardTowerID == nil {
		s.GuardTowerID = stringPtr(building.ID)
	}
	s.syncDerivedProgress()

	return CommandResult{
		Accepted: true,
		Event:    "build.placed",
		Message:  fmt.Sprintf("已下达%s蓝图，等待弟子备料和施工", labelForBuilding(buildingType)),
	}, nil
}

func (s *sessionState) requestUpgrade(buildingID string) (CommandResult, error) {
	building, err := s.getBuilding(buildingID)
	if err != nil {
		return CommandResult{}, err
	}
	if building.Type == BuildingMainHall || building.State != BuildingStateActive || building.MarkedForDemolition {
		return CommandResult{}, errors.New("building cannot be upgraded in its current state")
	}
	upgradeCost, ok := getUpgradeCost(building.Type, building.Level)
	if !ok {
		return CommandResult{}, errors.New("building already reached max level")
	}

	building.Supplied = zeroStockpile()
	building.MarkedForDemolition = false
	pending := BuildingWorkUpgrade
	nextLevel := building.Level + 1
	building.PendingAction = &pending
	building.PendingLevel = &nextLevel
	building.State = BuildingStatePlanned
	_ = upgradeCost
	s.syncDerivedProgress()

	return CommandResult{
		Accepted: true,
		Event:    "build.upgrade_requested",
		Message:  fmt.Sprintf("%s 已进入升级筹备，等待补足资源", labelForBuilding(building.Type)),
	}, nil
}

func (s *sessionState) toggleDemolition(buildingID string) (CommandResult, error) {
	building, err := s.getBuilding(buildingID)
	if err != nil {
		return CommandResult{}, err
	}
	if building.Type == BuildingMainHall {
		return CommandResult{}, errors.New("main hall cannot be demolished")
	}

	building.MarkedForDemolition = !building.MarkedForDemolition
	s.syncDerivedProgress()

	message := fmt.Sprintf("已标记拆除 %s", labelForBuilding(building.Type))
	if !building.MarkedForDemolition {
		message = fmt.Sprintf("已取消拆除 %s", labelForBuilding(building.Type))
	}

	return CommandResult{
		Accepted: true,
		Event:    "build.demolition_toggle",
		Message:  message,
	}, nil
}

func (s *sessionState) collectStockpile(kind ResourceKind, amount int) (CommandResult, error) {
	if amount <= 0 {
		return CommandResult{}, errors.New("collect amount must be positive")
	}
	if !isValidResourceKind(kind) {
		return CommandResult{}, fmt.Errorf("invalid resource kind: %s", kind)
	}

	s.Stockpile = addStockpile(s.Stockpile, stockpileWith(kind, amount))
	return CommandResult{
		Accepted: true,
		Event:    "resource.stockpile_gain",
		Message:  fmt.Sprintf("%s 已入库", labelForResource(kind)),
	}, nil
}

func (s *sessionState) deliverBuildResource(buildingID string, kind ResourceKind) (CommandResult, error) {
	building, err := s.getBuilding(buildingID)
	if err != nil {
		return CommandResult{}, err
	}
	if !isValidResourceKind(kind) {
		return CommandResult{}, fmt.Errorf("invalid resource kind: %s", kind)
	}
	if stockpileValue(s.Stockpile, kind) <= 0 {
		return CommandResult{}, errors.New("stockpile does not contain the requested resource")
	}

	cost := activeCostForBuilding(building)
	missing := missingResources(cost, building.Supplied)
	if stockpileValue(missing, kind) <= 0 {
		return CommandResult{}, errors.New("building no longer needs this resource")
	}

	s.Stockpile = spendStockpile(s.Stockpile, stockpileWith(kind, 1))
	building.Supplied = addStockpile(building.Supplied, stockpileWith(kind, 1))
	if isZeroStockpile(missingResources(cost, building.Supplied)) {
		building.State = BuildingStateSupplied
	}
	s.syncDerivedProgress()

	return CommandResult{
		Accepted: true,
		Event:    "resource.stockpile_deliver",
		Message:  fmt.Sprintf("%s 已收到 %s", labelForBuilding(building.Type), labelForResource(kind)),
	}, nil
}

func (s *sessionState) startBuildingWork(buildingID string) (CommandResult, error) {
	building, err := s.getBuilding(buildingID)
	if err != nil {
		return CommandResult{}, err
	}
	if building.State != BuildingStateSupplied {
		return CommandResult{}, errors.New("building is not ready for construction")
	}

	building.State = BuildingStateConstructing
	s.syncDerivedProgress()
	return CommandResult{
		Accepted: true,
		Event:    "build.constructing",
		Message:  fmt.Sprintf("%s 已开始施工", labelForBuilding(building.Type)),
	}, nil
}

func (s *sessionState) completeBuildingWork(buildingID string) (CommandResult, error) {
	building, err := s.getBuilding(buildingID)
	if err != nil {
		return CommandResult{}, err
	}
	if building.State != BuildingStateConstructing && building.State != BuildingStateSupplied {
		return CommandResult{}, errors.New("building is not under construction")
	}

	if building.PendingAction != nil && *building.PendingAction == BuildingWorkUpgrade && building.PendingLevel != nil {
		building.Level = *building.PendingLevel
	}
	building.PendingAction = nil
	building.PendingLevel = nil
	building.State = BuildingStateActive
	building.HP = getBuildingMaxHP(building.Type, building.Level)
	building.MarkedForDemolition = false
	s.syncDerivedProgress()

	message := fmt.Sprintf("%s 已完工", labelForBuilding(building.Type))
	if building.Level > 1 {
		message = fmt.Sprintf("%s 已升至 Lv.%d", labelForBuilding(building.Type), building.Level)
	}

	return CommandResult{
		Accepted: true,
		Event:    "build.completed",
		Message:  message,
	}, nil
}

func (s *sessionState) completeDemolition(buildingID string) (CommandResult, error) {
	building, err := s.getBuilding(buildingID)
	if err != nil {
		return CommandResult{}, err
	}
	if building.Type == BuildingMainHall {
		return CommandResult{}, errors.New("main hall cannot be demolished")
	}

	salvage := demolitionYield(building, s.RuinBuildingID)
	s.Stockpile = addStockpile(s.Stockpile, salvage)
	delete(s.Buildings, building.ID)
	if s.GuardTowerID != nil && *s.GuardTowerID == building.ID {
		s.GuardTowerID = nil
	}
	if s.RuinBuildingID != nil && *s.RuinBuildingID == building.ID {
		s.RuinBuildingID = nil
	}
	s.syncDerivedProgress()

	message := fmt.Sprintf("%s 已拆除", labelForBuilding(building.Type))
	if !isZeroStockpile(salvage) {
		message = fmt.Sprintf("%s 已拆除，返还 木%d 石%d 药%d", labelForBuilding(building.Type), salvage.SpiritWood, salvage.SpiritStone, salvage.Herb)
	}

	return CommandResult{
		Accepted: true,
		Event:    "build.demolished",
		Message:  message,
	}, nil
}

func (s *sessionState) completeRepair(buildingID string) (CommandResult, error) {
	building, err := s.getBuilding(buildingID)
	if err != nil {
		return CommandResult{}, err
	}
	if building.State != BuildingStateDamaged {
		return CommandResult{}, errors.New("building is not damaged")
	}

	repairCost := repairCostForBuilding(building.Type, building.Level)
	if !hasStockpile(s.Stockpile, repairCost) {
		return CommandResult{}, errors.New("stockpile is insufficient for repair")
	}

	s.Stockpile = spendStockpile(s.Stockpile, repairCost)
	building.HP = getBuildingMaxHP(building.Type, building.Level)
	building.State = BuildingStateActive
	s.syncDerivedProgress()

	return CommandResult{
		Accepted: true,
		Event:    "build.repaired",
		Message:  fmt.Sprintf("%s 已修复完毕", labelForBuilding(building.Type)),
	}, nil
}

func (s *sessionState) syncSessionProgress(phase SessionPhase, outcome SessionOutcome, objective string, firstRaidTriggered, firstRaidResolved bool) (CommandResult, error) {
	s.Phase = phase
	s.Outcome = outcome
	if objective != "" {
		s.Objective = objective
	}
	s.FirstRaidTriggered = firstRaidTriggered
	s.FirstRaidResolved = firstRaidResolved
	if outcome == SessionOutcomeVictory {
		s.Phase = SessionPhaseVictory
	}
	if outcome == SessionOutcomeDefeat {
		s.Phase = SessionPhaseDefeat
	}
	return CommandResult{
		Accepted: true,
		Event:    "session.progress_synced",
		Message:  "短会话关键快照已同步到 authority",
	}, nil
}

func (s *sessionState) getBuilding(buildingID string) (*buildingEntity, error) {
	building := s.Buildings[buildingID]
	if building == nil {
		return nil, fmt.Errorf("building not found: %s", buildingID)
	}
	return building, nil
}

func (s *sessionState) syncDerivedProgress() {
	if s.Outcome != SessionOutcomeInProgress {
		if s.Outcome == SessionOutcomeVictory {
			s.Phase = SessionPhaseVictory
			s.Objective = "本轮短会话已完成，可继续自由观察"
		} else {
			s.Phase = SessionPhaseDefeat
			s.Objective = "短会话失败，请重新开局再试"
		}
		return
	}

	if s.RuinBuildingID != nil {
		if _, ok := s.Buildings[*s.RuinBuildingID]; ok {
			s.Phase = SessionPhaseClearRuin
			s.Objective = "长按废弃仓房并拆除，为护山台腾出位置"
			return
		}
		s.RuinBuildingID = nil
	}

	guardTower := s.currentGuardTower()
	if guardTower == nil {
		s.Phase = SessionPhasePlaceGuardTower
		s.Objective = "在空地上放下第一座护山台"
		return
	}

	if guardTower.Level < 2 || guardTower.State != BuildingStateActive {
		s.Phase = SessionPhaseUpgradeGuardTower
		s.Objective = "把护山台升到 Lv.2，为首波敌袭做准备"
		return
	}

	if !s.FirstRaidTriggered {
		s.Phase = SessionPhaseRaidCountdown
		s.Objective = "护山台已就位，准备迎接第一波敌袭"
		return
	}

	if s.FirstRaidTriggered && !s.FirstRaidResolved {
		s.Phase = SessionPhaseDefend
		s.Objective = "守住首波敌袭，不要让主殿瘫痪"
		return
	}

	s.Phase = SessionPhaseRecover
	s.Objective = "首波敌袭已退去，修复受损建筑并恢复运转"
}

func (s *sessionState) currentGuardTower() *buildingEntity {
	if s.GuardTowerID != nil {
		if building, ok := s.Buildings[*s.GuardTowerID]; ok {
			return building
		}
		s.GuardTowerID = nil
	}
	for _, building := range s.Buildings {
		if building.Type == BuildingGuardTower {
			s.GuardTowerID = stringPtr(building.ID)
			return building
		}
	}
	return nil
}

func (s *sessionState) isFootprintOccupied(buildingType BuildingType, origin TileCoord, ignoreBuildingID string) bool {
	tiles := footprintFor(buildingType, origin)
	for _, building := range s.Buildings {
		if building.ID == ignoreBuildingID {
			continue
		}
		existing := footprintFor(building.Type, building.Origin)
		for _, tile := range tiles {
			for _, entry := range existing {
				if tile == entry {
					return true
				}
			}
		}
	}
	return false
}

func zeroStockpile() Stockpile {
	return Stockpile{}
}

func stockpileWith(kind ResourceKind, amount int) Stockpile {
	pile := zeroStockpile()
	switch kind {
	case ResourceSpiritWood:
		pile.SpiritWood = amount
	case ResourceSpiritStone:
		pile.SpiritStone = amount
	case ResourceHerb:
		pile.Herb = amount
	}
	return pile
}

func addStockpile(base, delta Stockpile) Stockpile {
	return Stockpile{
		SpiritWood:  base.SpiritWood + delta.SpiritWood,
		SpiritStone: base.SpiritStone + delta.SpiritStone,
		Herb:        base.Herb + delta.Herb,
	}
}

func spendStockpile(base, cost Stockpile) Stockpile {
	return Stockpile{
		SpiritWood:  maxInt(0, base.SpiritWood-cost.SpiritWood),
		SpiritStone: maxInt(0, base.SpiritStone-cost.SpiritStone),
		Herb:        maxInt(0, base.Herb-cost.Herb),
	}
}

func hasStockpile(base, cost Stockpile) bool {
	return base.SpiritWood >= cost.SpiritWood && base.SpiritStone >= cost.SpiritStone && base.Herb >= cost.Herb
}

func missingResources(cost, supplied Stockpile) Stockpile {
	return Stockpile{
		SpiritWood:  maxInt(0, cost.SpiritWood-supplied.SpiritWood),
		SpiritStone: maxInt(0, cost.SpiritStone-supplied.SpiritStone),
		Herb:        maxInt(0, cost.Herb-supplied.Herb),
	}
}

func isZeroStockpile(pile Stockpile) bool {
	return pile.SpiritWood == 0 && pile.SpiritStone == 0 && pile.Herb == 0
}

func stockpileValue(pile Stockpile, kind ResourceKind) int {
	switch kind {
	case ResourceSpiritWood:
		return pile.SpiritWood
	case ResourceSpiritStone:
		return pile.SpiritStone
	case ResourceHerb:
		return pile.Herb
	default:
		return 0
	}
}

func mustDefinition(buildingType BuildingType) buildingDefinition {
	def, ok := buildingDefinitions[buildingType]
	if !ok {
		panic("missing building definition")
	}
	return def
}

func labelForBuilding(buildingType BuildingType) string {
	switch buildingType {
	case BuildingDiscipleQuarters:
		return "弟子居"
	case BuildingWarehouse:
		return "仓房"
	case BuildingHerbGarden:
		return "药圃"
	case BuildingGuardTower:
		return "护山台"
	default:
		return "主殿"
	}
}

func labelForResource(kind ResourceKind) string {
	switch kind {
	case ResourceSpiritStone:
		return "灵石"
	case ResourceHerb:
		return "灵草"
	default:
		return "灵木"
	}
}

func isValidResourceKind(kind ResourceKind) bool {
	return kind == ResourceSpiritWood || kind == ResourceSpiritStone || kind == ResourceHerb
}

func getBuildingMaxHP(buildingType BuildingType, level int) int {
	base := mustDefinition(buildingType).BaseMaxHP
	if level <= 1 {
		return base
	}
	return int(float64(base) * (1 + float64(level-1)*0.45))
}

func activeCostForBuilding(building *buildingEntity) Stockpile {
	if building.PendingAction != nil && *building.PendingAction == BuildingWorkUpgrade {
		cost, ok := getUpgradeCost(building.Type, building.Level)
		if ok {
			return cost
		}
	}
	return mustDefinition(building.Type).BaseCost
}

func getUpgradeCost(buildingType BuildingType, level int) (Stockpile, bool) {
	if level >= 2 || buildingType == BuildingMainHall {
		return Stockpile{}, false
	}

	switch buildingType {
	case BuildingGuardTower:
		return Stockpile{SpiritWood: 1, SpiritStone: 1, Herb: 0}, true
	case BuildingDiscipleQuarters:
		return Stockpile{SpiritWood: 1, SpiritStone: 0, Herb: 1}, true
	case BuildingHerbGarden:
		return Stockpile{SpiritWood: 1, SpiritStone: 0, Herb: 1}, true
	case BuildingWarehouse:
		return Stockpile{SpiritWood: 1, SpiritStone: 1, Herb: 0}, true
	default:
		return Stockpile{}, false
	}
}

func repairCostForBuilding(buildingType BuildingType, level int) Stockpile {
	base := mustDefinition(buildingType).RepairCost
	if level <= 1 {
		return base
	}
	pile := base
	if buildingType == BuildingGuardTower {
		pile.SpiritWood++
		pile.SpiritStone++
	}
	if buildingType == BuildingWarehouse {
		pile.SpiritStone++
	}
	if buildingType == BuildingHerbGarden {
		pile.Herb++
	}
	return pile
}

func demolitionYield(building *buildingEntity, ruinBuildingID *string) Stockpile {
	if ruinBuildingID != nil && *ruinBuildingID == building.ID {
		return Stockpile{SpiritWood: 1, SpiritStone: 1, Herb: 0}
	}
	if building.PendingAction != nil && *building.PendingAction == BuildingWorkUpgrade {
		return zeroStockpile()
	}

	yield := zeroStockpile()
	if building.Level >= 2 && building.Type != BuildingHerbGarden {
		yield.SpiritWood = 1
	}
	if building.Level >= 2 && building.Type != BuildingDiscipleQuarters {
		yield.SpiritStone = 1
	}
	if building.Type == BuildingHerbGarden {
		yield.Herb = 1
	}
	return yield
}

func footprintFor(buildingType BuildingType, origin TileCoord) []TileCoord {
	def := mustDefinition(buildingType)
	tiles := make([]TileCoord, 0, def.Width*def.Height)
	for rowOffset := 0; rowOffset < def.Height; rowOffset++ {
		for colOffset := 0; colOffset < def.Width; colOffset++ {
			tiles = append(tiles, TileCoord{
				Col: origin.Col + colOffset,
				Row: origin.Row + rowOffset,
			})
		}
	}
	return tiles
}

func intPtr(v int) *int {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
