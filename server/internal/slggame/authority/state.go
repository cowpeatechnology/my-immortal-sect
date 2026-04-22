package authority

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

type resourceNodeRule struct {
	MaxCharges   int
	RegenSeconds int
}

type resourceNodeSeed struct {
	Kind ResourceKind
	Tile TileCoord
}

type buildingDefinition struct {
	ID         BuildingType
	Width      int
	Height     int
	BaseCost   Stockpile
	BaseMaxHP  int
	RepairCost Stockpile
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
	WorkProgressTicks   int64
	AttackCooldown      int
}

type hostileEntity struct {
	ID               string
	ArchetypeID      UnitArchetypeID
	Name             string
	Tile             TileCoord
	HP               int
	VisualState      UnitVisualState
	TargetBuildingID *string
	AttackCooldown   int
	Active           bool
}

type resourceNodeEntity struct {
	Kind              ResourceKind
	Tile              TileCoord
	State             ResourceNodeState
	RemainingCharges  int
	MaxCharges        int
	RegenSeconds      int
	RegenTimerSeconds int
}

type sessionState struct {
	SessionID              string
	GameTick               int64
	nextBuildingSeq        int
	Stockpile              Stockpile
	ResourceNodes          map[string]*resourceNodeEntity
	Buildings              map[string]*buildingEntity
	DiscipleHP             int
	DiscipleAttackCooldown int
	Hostile                *hostileEntity
	Phase                  SessionPhase
	Outcome                SessionOutcome
	Objective              string
	GuardTowerID           *string
	RuinBuildingID         *string
	FirstRaidTriggered     bool
	FirstRaidResolved      bool
	RaidCountdownSeconds   int
	DefendRemainingSeconds int
}

const (
	authorityDiscipleID      = "disciple-01"
	authorityDiscipleName    = "玄"
	authorityHostileID       = "hostile-01"
	authorityBuildWorkTicks  = 2
	authorityDemolitionTicks = 2
	authorityRepairWorkTicks = 2
)

func newSessionState(sessionID string) *sessionState {
	state := &sessionState{
		SessionID:       sessionID,
		GameTick:        0,
		nextBuildingSeq: 1,
		Stockpile:       zeroStockpile(),
		ResourceNodes:   map[string]*resourceNodeEntity{},
		Buildings:       map[string]*buildingEntity{},
		DiscipleHP:      mustUnitArchetype(UnitArchetypeSectDisciple).Stats.MaxHP,
		Phase:           SessionPhaseClearRuin,
		Outcome:         SessionOutcomeInProgress,
		Objective:       "长按废弃仓房并拆除，为护山台腾出位置",
	}

	for _, seed := range sharedAuthorityConfig.initialResourceSeeds {
		rule := mustResourceNodeRule(seed.Kind)
		state.ResourceNodes[resourceTileKey(seed.Tile)] = &resourceNodeEntity{
			Kind:              seed.Kind,
			Tile:              seed.Tile,
			State:             ResourceNodeStateAvailable,
			RemainingCharges:  rule.MaxCharges,
			MaxCharges:        rule.MaxCharges,
			RegenSeconds:      rule.RegenSeconds,
			RegenTimerSeconds: 0,
		}
	}

	mainHall := state.addBuilding(BuildingMainHall, sharedAuthorityConfig.initialMainHallTile, BuildingStateActive)
	mainHall.Level = 1
	mainHall.HP = getBuildingMaxHP(mainHall.Type, mainHall.Level)
	mainHall.PendingAction = nil
	mainHall.PendingLevel = nil

	ruin := state.addBuilding(BuildingWarehouse, sharedAuthorityConfig.initialRuinTile, BuildingStateDamaged)
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

	resourceNodes := make([]ResourceNodeSnapshot, 0, len(s.ResourceNodes))
	for _, node := range s.ResourceNodes {
		resourceNodes = append(resourceNodes, ResourceNodeSnapshot{
			Tile:              node.Tile,
			Kind:              node.Kind,
			State:             node.State,
			RemainingCharges:  node.RemainingCharges,
			MaxCharges:        node.MaxCharges,
			RegenSeconds:      node.RegenSeconds,
			RegenTimerSeconds: node.RegenTimerSeconds,
		})
	}

	sort.Slice(resourceNodes, func(i, j int) bool {
		left := resourceTileKey(resourceNodes[i].Tile)
		right := resourceTileKey(resourceNodes[j].Tile)
		return left < right
	})

	disciples := []DiscipleSnapshot{s.currentDiscipleSnapshot()}
	hostiles := s.currentHostileSnapshots()
	recoverReason, damagedBuildingCount, regeneratingNodeCount := s.sessionRecoveryStatus()

	return SessionSnapshot{
		SessionID:     s.SessionID,
		GameTick:      s.GameTick,
		Stockpile:     s.Stockpile,
		ResourceNodes: resourceNodes,
		Buildings:     buildings,
		Disciples:     disciples,
		Hostiles:      hostiles,
		Session: SessionProgressSnapshot{
			Phase:                  s.Phase,
			Outcome:                s.Outcome,
			Objective:              s.Objective,
			GuardTowerID:           s.GuardTowerID,
			RuinBuildingID:         s.RuinBuildingID,
			FirstRaidTriggered:     s.FirstRaidTriggered,
			FirstRaidResolved:      s.FirstRaidResolved,
			RaidCountdownSeconds:   s.RaidCountdownSeconds,
			DefendRemainingSeconds: s.DefendRemainingSeconds,
			RecoverReason:          recoverReason,
			DamagedBuildingCount:   damagedBuildingCount,
			RegeneratingNodeCount:  regeneratingNodeCount,
		},
	}
}

type discipleAssignmentPlan struct {
	kind                   DiscipleAssignmentKind
	targetBuildingID       *string
	targetResourceKind     *ResourceKind
	targetTile             *TileCoord
	carryingKind           *ResourceKind
	carryingAmount         int
	expectedNextTransition *string
}

func (s *sessionState) currentDiscipleSnapshot() DiscipleSnapshot {
	plan := s.currentDiscipleAssignment()
	disciple := mustUnitArchetype(UnitArchetypeSectDisciple)
	return DiscipleSnapshot{
		ArchetypeID:         disciple.ID,
		ID:                 authorityDiscipleID,
		Name:               authorityDiscipleName,
		AssignmentKind:     plan.kind,
		TargetBuildingID:   plan.targetBuildingID,
		TargetResourceKind: plan.targetResourceKind,
		TargetTile:         plan.targetTile,
		Carrying: DiscipleCarryingSnapshot{
			Kind:   plan.carryingKind,
			Amount: plan.carryingAmount,
		},
		HP:                     s.DiscipleHP,
		MaxHP:                  disciple.Stats.MaxHP,
		VisualState:            s.currentDiscipleVisualState(plan),
		WorkProgressTicks:      s.assignmentWorkProgressTicks(plan),
		ExpectedNextTransition: plan.expectedNextTransition,
	}
}

func (s *sessionState) currentHostileSnapshots() []HostileSnapshot {
	if s.Hostile == nil || !s.Hostile.Active {
		return []HostileSnapshot{}
	}

	archetype := mustUnitArchetype(s.Hostile.ArchetypeID)
	return []HostileSnapshot{{
		ID:               s.Hostile.ID,
		ArchetypeID:      s.Hostile.ArchetypeID,
		Name:             s.Hostile.Name,
		Tile:             s.Hostile.Tile,
		HP:               s.Hostile.HP,
		MaxHP:            archetype.Stats.MaxHP,
		VisualState:      s.Hostile.VisualState,
		Active:           s.Hostile.Active,
		TargetBuildingID: s.Hostile.TargetBuildingID,
	}}
}

func (s *sessionState) assignmentWorkProgressTicks(plan discipleAssignmentPlan) int64 {
	if plan.targetBuildingID == nil {
		return 0
	}

	building := s.Buildings[*plan.targetBuildingID]
	if building == nil {
		return 0
	}

	switch plan.kind {
	case DiscipleAssignmentBuild, DiscipleAssignmentDemolish, DiscipleAssignmentRepair:
		return building.WorkProgressTicks
	default:
		return 0
	}
}

func (s *sessionState) currentDiscipleAssignment() discipleAssignmentPlan {
	if s.Outcome != SessionOutcomeInProgress {
		return discipleAssignmentPlan{kind: DiscipleAssignmentIdle}
	}

	if target := s.currentGuardAssignment(); target != nil {
		next := "hold_guard_line"
		targetTile := tilePtr(target.Origin)
		if attackTile := s.currentGuardLineTile(); attackTile != nil {
			targetTile = attackTile
		}
		return discipleAssignmentPlan{
			kind:                   DiscipleAssignmentGuard,
			targetBuildingID:       stringPtr(target.ID),
			targetTile:             targetTile,
			expectedNextTransition: stringPtr(next),
		}
	}

	if target := s.currentRepairTarget(); target != nil {
		next := "move_to_repair_site"
		return discipleAssignmentPlan{
			kind:                   DiscipleAssignmentRepair,
			targetBuildingID:       stringPtr(target.ID),
			targetTile:             tilePtr(target.Origin),
			expectedNextTransition: stringPtr(next),
		}
	}

	if target := s.currentDemolitionTarget(); target != nil {
		next := "move_to_demolition_site"
		return discipleAssignmentPlan{
			kind:                   DiscipleAssignmentDemolish,
			targetBuildingID:       stringPtr(target.ID),
			targetTile:             tilePtr(target.Origin),
			expectedNextTransition: stringPtr(next),
		}
	}

	if target := s.currentBuildTarget(); target != nil {
		next := "move_to_build_site"
		return discipleAssignmentPlan{
			kind:                   DiscipleAssignmentBuild,
			targetBuildingID:       stringPtr(target.ID),
			targetTile:             tilePtr(target.Origin),
			expectedNextTransition: stringPtr(next),
		}
	}

	if building, kind := s.currentHaulTarget(); building != nil && kind != nil {
		next := "deliver_build_resource"
		return discipleAssignmentPlan{
			kind:                   DiscipleAssignmentHaul,
			targetBuildingID:       stringPtr(building.ID),
			targetResourceKind:     kind,
			targetTile:             tilePtr(building.Origin),
			carryingKind:           kind,
			carryingAmount:         1,
			expectedNextTransition: stringPtr(next),
		}
	}

	if resource := s.currentGatherTarget(); resource != nil {
		next := "collect_stockpile"
		return discipleAssignmentPlan{
			kind:                   DiscipleAssignmentGather,
			targetResourceKind:     resourceKindPtr(resource.Kind),
			targetTile:             tilePtr(resource.Tile),
			expectedNextTransition: stringPtr(next),
		}
	}

	return discipleAssignmentPlan{kind: DiscipleAssignmentIdle}
}

func (s *sessionState) currentDiscipleVisualState(plan discipleAssignmentPlan) UnitVisualState {
	switch plan.kind {
	case DiscipleAssignmentGather:
		return UnitVisualStateWorking
	case DiscipleAssignmentHaul:
		return UnitVisualStateCarrying
	case DiscipleAssignmentBuild, DiscipleAssignmentDemolish, DiscipleAssignmentRepair:
		if s.assignmentWorkProgressTicks(plan) > 0 {
			return UnitVisualStateWorking
		}
		return UnitVisualStateMoving
	case DiscipleAssignmentGuard:
		if s.discipleCanAttackHostile() {
			return UnitVisualStateAttacking
		}
		return UnitVisualStateGuarding
	default:
		return UnitVisualStateIdle
	}
}

func (s *sessionState) currentGuardAssignment() *buildingEntity {
	if !s.FirstRaidTriggered || s.FirstRaidResolved {
		return nil
	}
	return s.currentGuardTower()
}

func (s *sessionState) currentGuardLineTile() *TileCoord {
	if s.Hostile == nil || !s.Hostile.Active {
		return nil
	}
	tile := TileCoord{Col: s.Hostile.Tile.Col, Row: maxInt(0, s.Hostile.Tile.Row-1)}
	return &tile
}

func (s *sessionState) discipleCanAttackHostile() bool {
	guardTile := s.currentGuardLineTile()
	if guardTile == nil || s.Hostile == nil || !s.Hostile.Active {
		return false
	}
	return manhattanDistance(*guardTile, s.Hostile.Tile) <= mustUnitArchetype(UnitArchetypeSectDisciple).Stats.AttackRangeTiles
}

func (s *sessionState) currentRepairTarget() *buildingEntity {
	for _, building := range s.sortedBuildings() {
		if s.canRepairBuilding(building) {
			return building
		}
	}
	return nil
}

func (s *sessionState) currentDemolitionTarget() *buildingEntity {
	for _, building := range s.sortedBuildings() {
		if building.MarkedForDemolition {
			return building
		}
	}
	return nil
}

func (s *sessionState) currentBuildTarget() *buildingEntity {
	if priority := s.currentPriorityBuilding(); priority != nil && (priority.State == BuildingStateSupplied || priority.State == BuildingStateConstructing) {
		return priority
	}

	for _, building := range s.sortedBuildings() {
		if building.State == BuildingStateSupplied || building.State == BuildingStateConstructing {
			return building
		}
	}
	return nil
}

func (s *sessionState) currentHaulTarget() (*buildingEntity, *ResourceKind) {
	if priority := s.currentPriorityBuilding(); priority != nil && priority.State == BuildingStatePlanned {
		if kind := firstMissingResourceWithStockpile(activeCostForBuilding(priority), priority.Supplied, s.Stockpile); kind != nil {
			return priority, kind
		}
	}

	for _, building := range s.sortedBuildings() {
		if building.State != BuildingStatePlanned {
			continue
		}
		if kind := firstMissingResourceWithStockpile(activeCostForBuilding(building), building.Supplied, s.Stockpile); kind != nil {
			return building, kind
		}
	}

	return nil, nil
}

func (s *sessionState) currentRaidTargetBuilding() *buildingEntity {
	preferred := make([]*buildingEntity, 0)
	for _, building := range s.sortedBuildings() {
		if building.HP <= 0 {
			continue
		}
		if building.Type == BuildingMainHall {
			continue
		}
		if building.State != BuildingStateActive && building.State != BuildingStateDamaged {
			continue
		}
		preferred = append(preferred, building)
	}
	if len(preferred) > 0 {
		sort.Slice(preferred, func(i, j int) bool {
			if preferred[i].HP != preferred[j].HP {
				return preferred[i].HP < preferred[j].HP
			}
			return preferred[i].ID < preferred[j].ID
		})
		return preferred[0]
	}

	for _, building := range s.sortedBuildings() {
		if building.Type == BuildingMainHall && building.HP > 0 {
			return building
		}
	}
	return nil
}

func (s *sessionState) hostileAttackTile(building *buildingEntity) TileCoord {
	def := mustDefinition(building.Type)
	return TileCoord{
		Col: building.Origin.Col + def.Width,
		Row: building.Origin.Row + maxInt(0, def.Height-1),
	}
}

func (s *sessionState) currentGatherTarget() *resourceNodeEntity {
	neededKinds := s.gatherPriorityKinds()
	if len(neededKinds) == 0 {
		return nil
	}

	for _, kind := range neededKinds {
		if target := s.firstAvailableResourceNode(kind); target != nil {
			return target
		}
	}

	return nil
}

func (s *sessionState) gatherPriorityKinds() []ResourceKind {
	if deficit := s.repairResourceDeficit(); deficit != nil {
		if kinds := stockpileKindsWithPositiveValue(*deficit); len(kinds) > 0 {
			return kinds
		}
	}

	if priority := s.currentPriorityBuilding(); priority != nil && priority.State == BuildingStatePlanned {
		if kinds := stockpileKindsWithPositiveValue(missingResources(activeCostForBuilding(priority), priority.Supplied)); len(kinds) > 0 {
			return kinds
		}
	}

	for _, building := range s.sortedBuildings() {
		if building.State != BuildingStatePlanned {
			continue
		}
		if kinds := stockpileKindsWithPositiveValue(missingResources(activeCostForBuilding(building), building.Supplied)); len(kinds) > 0 {
			return kinds
		}
	}

	return resourceKindOrder()
}

func (s *sessionState) currentPriorityBuilding() *buildingEntity {
	if s.Phase != SessionPhasePlaceGuardTower && s.Phase != SessionPhaseUpgradeGuardTower {
		return nil
	}
	return s.currentGuardTower()
}

func (s *sessionState) canRepairBuilding(building *buildingEntity) bool {
	if building.State != BuildingStateDamaged {
		return false
	}
	if s.RuinBuildingID != nil && *s.RuinBuildingID == building.ID {
		return false
	}
	if building.HP >= getBuildingMaxHP(building.Type, building.Level) {
		return false
	}
	if building.WorkProgressTicks > 0 {
		return true
	}
	return hasStockpile(s.Stockpile, repairCostForBuilding(building.Type, building.Level))
}

func (s *sessionState) repairResourceDeficit() *Stockpile {
	deficit := zeroStockpile()
	hasDeficit := false
	for _, building := range s.sortedBuildings() {
		if building.State != BuildingStateDamaged {
			continue
		}
		if s.RuinBuildingID != nil && *s.RuinBuildingID == building.ID {
			continue
		}
		if building.HP >= getBuildingMaxHP(building.Type, building.Level) {
			continue
		}
		if building.WorkProgressTicks > 0 {
			continue
		}
		repairCost := repairCostForBuilding(building.Type, building.Level)
		for _, kind := range resourceKindOrder() {
			missing := maxInt(0, stockpileValue(repairCost, kind)-stockpileValue(s.Stockpile, kind))
			if missing <= stockpileValue(deficit, kind) {
				continue
			}
			setStockpileValue(&deficit, kind, missing)
			hasDeficit = true
		}
	}
	if !hasDeficit {
		return nil
	}
	return &deficit
}

func (s *sessionState) firstAvailableResourceNode(kind ResourceKind) *resourceNodeEntity {
	nodes := make([]*resourceNodeEntity, 0, len(s.ResourceNodes))
	for _, node := range s.ResourceNodes {
		if node.Kind != kind {
			continue
		}
		if node.State != ResourceNodeStateAvailable || node.RemainingCharges <= 0 {
			continue
		}
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return resourceTileKey(nodes[i].Tile) < resourceTileKey(nodes[j].Tile)
	})
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0]
}

func (s *sessionState) sortedBuildings() []*buildingEntity {
	buildings := make([]*buildingEntity, 0, len(s.Buildings))
	for _, building := range s.Buildings {
		buildings = append(buildings, building)
	}
	sort.Slice(buildings, func(i, j int) bool {
		return buildings[i].ID < buildings[j].ID
	})
	return buildings
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
			ResourceTile *TileCoord   `json:"resourceTile"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return CommandResult{}, err
		}
		return s.collectStockpile(payload.ResourceKind, payload.Amount, payload.ResourceTile)
	case "deliver_build_resource":
		var payload struct {
			BuildingID   string       `json:"buildingId"`
			ResourceKind ResourceKind `json:"resourceKind"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return CommandResult{}, err
		}
		return s.deliverBuildResource(payload.BuildingID, payload.ResourceKind)
	case "complete_demolition":
		var payload struct {
			BuildingID string `json:"buildingId"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return CommandResult{}, err
		}
		return s.completeDemolition(payload.BuildingID)
	case "trigger_first_raid":
		return s.triggerFirstRaid()
	case "resolve_first_raid":
		return s.resolveFirstRaid()
	case "expire_session":
		return s.expireSession()
	case "report_building_damage":
		var payload struct {
			BuildingID string `json:"buildingId"`
			HP         int    `json:"hp"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return CommandResult{}, err
		}
		return s.reportBuildingDamage(payload.BuildingID, payload.HP)
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
	if _, ok := sharedAuthorityConfig.buildingDefinitions[buildingType]; !ok || buildingType == BuildingMainHall {
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
	building.WorkProgressTicks = 0
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

func (s *sessionState) collectStockpile(kind ResourceKind, amount int, resourceTile *TileCoord) (CommandResult, error) {
	if amount <= 0 {
		return CommandResult{}, errors.New("collect amount must be positive")
	}
	if !isValidResourceKind(kind) {
		return CommandResult{}, fmt.Errorf("invalid resource kind: %s", kind)
	}

	depleted := false
	if resourceTile != nil {
		node, err := s.getResourceNode(*resourceTile)
		if err != nil {
			return CommandResult{}, err
		}
		if node.Kind != kind {
			return CommandResult{}, fmt.Errorf("resource node kind mismatch at %d,%d", resourceTile.Col, resourceTile.Row)
		}
		if node.State != ResourceNodeStateAvailable || node.RemainingCharges <= 0 {
			return CommandResult{}, errors.New("resource node is not harvestable")
		}
		if amount > node.RemainingCharges {
			return CommandResult{}, errors.New("resource node does not contain enough charges")
		}

		node.RemainingCharges -= amount
		if node.RemainingCharges == 0 {
			node.State = ResourceNodeStateRegenerating
			node.RegenTimerSeconds = node.RegenSeconds
			depleted = true
		}
	}

	s.Stockpile = addStockpile(s.Stockpile, stockpileWith(kind, amount))
	s.syncDerivedProgress()
	message := fmt.Sprintf("%s 已入库", labelForResource(kind))
	if depleted {
		message = fmt.Sprintf("%s 已入库，对应资源点进入刷新", labelForResource(kind))
	}
	return CommandResult{
		Accepted: true,
		Event:    "resource.stockpile_gain",
		Message:  message,
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
		building.WorkProgressTicks = 0
	}
	s.syncDerivedProgress()

	return CommandResult{
		Accepted: true,
		Event:    "resource.stockpile_deliver",
		Message:  fmt.Sprintf("%s 已收到 %s", labelForBuilding(building.Type), labelForResource(kind)),
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

func (s *sessionState) triggerFirstRaid() (CommandResult, error) {
	if s.Outcome != SessionOutcomeInProgress {
		return CommandResult{}, errors.New("session already resolved")
	}
	if s.FirstRaidTriggered {
		return CommandResult{
			Accepted: true,
			Event:    "session.raid_triggered",
			Message:  "首波敌袭已进入 authority 守御阶段",
		}, nil
	}
	if s.Phase != SessionPhaseRaidCountdown {
		return CommandResult{}, fmt.Errorf("cannot trigger first raid from phase %s", s.Phase)
	}

	s.FirstRaidTriggered = true
	s.RaidCountdownSeconds = 0
	s.DefendRemainingSeconds = sharedAuthorityConfig.firstRaidPrepSeconds
	s.syncDerivedProgress()
	return CommandResult{
		Accepted: true,
		Event:    "session.raid_triggered",
		Message:  "首波敌袭已进入 authority 守御阶段",
	}, nil
}

func (s *sessionState) resolveFirstRaid() (CommandResult, error) {
	if s.Outcome != SessionOutcomeInProgress {
		return CommandResult{}, errors.New("session already resolved")
	}
	if !s.FirstRaidTriggered {
		return CommandResult{}, errors.New("first raid has not been triggered")
	}
	if s.FirstRaidResolved {
		return CommandResult{
			Accepted: true,
			Event:    "session.raid_resolved",
			Message:  "首波敌袭已由 authority 记为击退",
		}, nil
	}

	s.FirstRaidResolved = true
	s.DefendRemainingSeconds = 0
	s.syncDerivedProgress()
	message := "首波敌袭已由 authority 记为击退，进入修复收口"
	if s.Outcome == SessionOutcomeVictory {
		message = "首波敌袭已守住且宗门恢复完毕，authority 判定短会话达成"
	}

	return CommandResult{
		Accepted: true,
		Event:    "session.raid_resolved",
		Message:  message,
	}, nil
}

func (s *sessionState) expireSession() (CommandResult, error) {
	if s.Outcome != SessionOutcomeInProgress {
		return CommandResult{
			Accepted: true,
			Event:    "session.expired",
			Message:  "短会话已是终局，无需重复收口",
		}, nil
	}

	s.Outcome = SessionOutcomeDefeat
	s.syncDerivedProgress()
	return CommandResult{
		Accepted: true,
		Event:    "session.expired",
		Message:  "authority 判定短会话超时失败",
	}, nil
}

func (s *sessionState) reportBuildingDamage(buildingID string, hp int) (CommandResult, error) {
	_ = buildingID
	_ = hp
	return CommandResult{}, errors.New("report_building_damage is no longer allowed; authority owns combat damage")
}

func (s *sessionState) advanceResourceNodes(seconds int) {
	if seconds <= 0 {
		return
	}

	changed := false
	for step := 0; step < seconds; step++ {
		s.GameTick++
		if s.advanceRaidClock(1) {
			changed = true
		}
		if s.advanceCombatStep() {
			changed = true
		}
		if s.advanceResourceNodeRefreshStep() {
			changed = true
		}
		if s.advanceAssignedWork(1) {
			changed = true
		}
	}

	if changed {
		s.syncDerivedProgress()
	}
}

func (s *sessionState) advanceResourceNodeRefreshStep() bool {
	changed := false
	for _, node := range s.ResourceNodes {
		if node.State != ResourceNodeStateRegenerating {
			continue
		}
		nextTimer := maxInt(0, node.RegenTimerSeconds-1)
		if nextTimer != node.RegenTimerSeconds {
			node.RegenTimerSeconds = nextTimer
			changed = true
		}
		if node.RegenTimerSeconds > 0 {
			continue
		}
		node.State = ResourceNodeStateAvailable
		node.RemainingCharges = node.MaxCharges
		changed = true
	}
	return changed
}

func (s *sessionState) advanceRaidClock(seconds int) bool {
	changed := false
	if s.Phase == SessionPhaseRaidCountdown && !s.FirstRaidTriggered {
		if s.RaidCountdownSeconds <= 0 {
			s.RaidCountdownSeconds = sharedAuthorityConfig.firstRaidPrepSeconds
			changed = true
		}
		nextCountdown := maxInt(0, s.RaidCountdownSeconds-seconds)
		if nextCountdown != s.RaidCountdownSeconds {
			s.RaidCountdownSeconds = nextCountdown
			changed = true
		}
		if s.RaidCountdownSeconds == 0 {
			s.FirstRaidTriggered = true
			s.DefendRemainingSeconds = sharedAuthorityConfig.firstRaidPrepSeconds
			changed = true
		}
	}

	if s.Phase == SessionPhaseDefend && s.FirstRaidTriggered && !s.FirstRaidResolved {
		if s.DefendRemainingSeconds <= 0 {
			s.DefendRemainingSeconds = sharedAuthorityConfig.firstRaidPrepSeconds
			changed = true
		}
		nextDefend := maxInt(0, s.DefendRemainingSeconds-seconds)
		if nextDefend != s.DefendRemainingSeconds {
			s.DefendRemainingSeconds = nextDefend
			changed = true
		}
		if s.DefendRemainingSeconds == 0 {
			s.FirstRaidResolved = true
			changed = true
		}
	}

	return changed
}

func (s *sessionState) advanceCombatStep() bool {
	if s.Outcome != SessionOutcomeInProgress {
		s.clearHostile()
		return false
	}
	if s.Phase != SessionPhaseDefend || !s.FirstRaidTriggered || s.FirstRaidResolved {
		if s.clearHostile() {
			return true
		}
		return false
	}

	changed := false
	if s.Hostile == nil || !s.Hostile.Active {
		s.spawnHostile()
		changed = true
	}
	if s.Hostile == nil || !s.Hostile.Active {
		return changed
	}

	if s.advanceHostileMovement() {
		changed = true
	}
	if s.advanceHostileAttack() {
		changed = true
	}
	if s.advanceGuardTowerFire() {
		changed = true
	}
	if s.advanceDiscipleGuardFire() {
		changed = true
	}

	if s.Hostile != nil && s.Hostile.Active && s.Hostile.HP <= 0 {
		s.FirstRaidResolved = true
		s.DefendRemainingSeconds = 0
		s.clearHostile()
		return true
	}

	return changed
}

func (s *sessionState) spawnHostile() {
	target := s.currentRaidTargetBuilding()
	if target == nil {
		return
	}

	archetype := mustUnitArchetype(UnitArchetypeBanditScout)
	spawnTile := TileCoord{
		Col: target.Origin.Col + mustDefinition(target.Type).Width + hostileSpawnOffsetCols,
		Row: target.Origin.Row,
	}
	s.Hostile = &hostileEntity{
		ID:               authorityHostileID,
		ArchetypeID:      archetype.ID,
		Name:             archetype.DisplayName,
		Tile:             spawnTile,
		HP:               archetype.Stats.MaxHP,
		VisualState:      UnitVisualStateMoving,
		TargetBuildingID: stringPtr(target.ID),
		Active:           true,
	}
}

func (s *sessionState) clearHostile() bool {
	if s.Hostile == nil || !s.Hostile.Active {
		s.Hostile = nil
		return false
	}
	s.Hostile = nil
	return true
}

func (s *sessionState) advanceHostileMovement() bool {
	if s.Hostile == nil || !s.Hostile.Active {
		return false
	}
	target := s.currentRaidTargetBuilding()
	if target == nil {
		return false
	}
	s.Hostile.TargetBuildingID = stringPtr(target.ID)
	attackTile := s.hostileAttackTile(target)
	if s.Hostile.Tile == attackTile {
		s.Hostile.VisualState = UnitVisualStateAttacking
		return true
	}

	if s.Hostile.Tile.Col < attackTile.Col {
		s.Hostile.Tile.Col++
	} else if s.Hostile.Tile.Col > attackTile.Col {
		s.Hostile.Tile.Col--
	} else if s.Hostile.Tile.Row < attackTile.Row {
		s.Hostile.Tile.Row++
	} else if s.Hostile.Tile.Row > attackTile.Row {
		s.Hostile.Tile.Row--
	}
	s.Hostile.VisualState = UnitVisualStateMoving
	return true
}

func (s *sessionState) advanceHostileAttack() bool {
	if s.Hostile == nil || !s.Hostile.Active {
		return false
	}
	target := s.currentRaidTargetBuilding()
	if target == nil {
		return false
	}
	if manhattanDistance(s.Hostile.Tile, s.hostileAttackTile(target)) > 0 {
		return false
	}
	if s.Hostile.AttackCooldown > 0 {
		s.Hostile.AttackCooldown--
		return true
	}

	archetype := mustUnitArchetype(s.Hostile.ArchetypeID)
	damage := getMitigatedDamage(archetype.Stats.AttackPower, getBuildingStructureDefense(target))
	if damage <= 0 {
		return false
	}

	target.HP = maxInt(0, target.HP-damage)
	target.WorkProgressTicks = 0
	if target.HP < getBuildingMaxHP(target.Type, target.Level) {
		target.State = BuildingStateDamaged
	}
	if target.HP == 0 && target.Type == BuildingMainHall {
		s.Outcome = SessionOutcomeDefeat
	}
	s.Hostile.VisualState = UnitVisualStateAttacking
	s.Hostile.AttackCooldown = maxInt(0, archetype.Stats.AttackIntervalTicks-1)
	return true
}

func (s *sessionState) advanceGuardTowerFire() bool {
	if s.Hostile == nil || !s.Hostile.Active {
		return false
	}

	changed := false
	for _, building := range s.sortedBuildings() {
		profile := getBuildingGuardProfile(building)
		if profile == nil || building.HP <= 0 {
			continue
		}
		if building.State != BuildingStateActive && building.State != BuildingStateDamaged {
			continue
		}
		if manhattanDistance(building.Origin, s.Hostile.Tile) > profile.RangeTiles {
			continue
		}
		if building.AttackCooldown > 0 {
			building.AttackCooldown--
			changed = true
			continue
		}

		damage := getMitigatedDamage(profile.AttackPower, mustUnitArchetype(s.Hostile.ArchetypeID).Stats.Defense)
		s.Hostile.HP = maxInt(0, s.Hostile.HP-damage)
		if s.Hostile.HP < mustUnitArchetype(s.Hostile.ArchetypeID).Stats.MaxHP {
			s.Hostile.VisualState = UnitVisualStateInjured
		}
		changed = true
		if s.Hostile.HP <= 0 {
			return true
		}
		building.AttackCooldown = maxInt(0, profile.AttackIntervalTicks-1)
	}
	return changed
}

func (s *sessionState) advanceDiscipleGuardFire() bool {
	if s.Hostile == nil || !s.Hostile.Active {
		return false
	}
	if s.DiscipleHP <= 0 {
		return false
	}
	plan := s.currentDiscipleAssignment()
	if plan.kind != DiscipleAssignmentGuard {
		return false
	}
	guardTile := s.currentGuardLineTile()
	if guardTile == nil {
		return false
	}
	disciple := mustUnitArchetype(UnitArchetypeSectDisciple)
	if manhattanDistance(*guardTile, s.Hostile.Tile) > disciple.Stats.AttackRangeTiles {
		return false
	}
	if s.DiscipleAttackCooldown > 0 {
		s.DiscipleAttackCooldown--
		return true
	}

	damage := getMitigatedDamage(disciple.Stats.AttackPower, mustUnitArchetype(s.Hostile.ArchetypeID).Stats.Defense)
	s.Hostile.HP = maxInt(0, s.Hostile.HP-damage)
	if s.Hostile.HP < mustUnitArchetype(s.Hostile.ArchetypeID).Stats.MaxHP {
		s.Hostile.VisualState = UnitVisualStateInjured
	}
	s.DiscipleAttackCooldown = maxInt(0, disciple.Stats.AttackIntervalTicks-1)
	return true
}

func (s *sessionState) advanceAssignedWork(seconds int) bool {
	plan := s.currentDiscipleAssignment()
	if plan.targetBuildingID == nil {
		return false
	}

	building := s.Buildings[*plan.targetBuildingID]
	if building == nil {
		return false
	}

	switch plan.kind {
	case DiscipleAssignmentHaul:
		return s.advanceHaulWork(building, plan, seconds)
	case DiscipleAssignmentBuild:
		return s.advanceBuildingWork(building, seconds)
	case DiscipleAssignmentDemolish:
		return s.advanceDemolitionWork(building, seconds)
	case DiscipleAssignmentRepair:
		return s.advanceRepairWork(building, seconds)
	default:
		return false
	}
}

func (s *sessionState) advanceHaulWork(building *buildingEntity, plan discipleAssignmentPlan, seconds int) bool {
	if seconds <= 0 || plan.targetResourceKind == nil {
		return false
	}

	changed := false
	for step := 0; step < seconds; step++ {
		kind := plan.targetResourceKind
		if stockpileValue(s.Stockpile, *kind) <= 0 {
			break
		}
		cost := activeCostForBuilding(building)
		missing := missingResources(cost, building.Supplied)
		if stockpileValue(missing, *kind) <= 0 {
			break
		}

		s.Stockpile = spendStockpile(s.Stockpile, stockpileWith(*kind, 1))
		building.Supplied = addStockpile(building.Supplied, stockpileWith(*kind, 1))
		if isZeroStockpile(missingResources(cost, building.Supplied)) {
			building.State = BuildingStateSupplied
			building.WorkProgressTicks = 0
		}
		changed = true

		plan = s.currentDiscipleAssignment()
		if plan.targetBuildingID == nil || *plan.targetBuildingID != building.ID || plan.targetResourceKind == nil {
			break
		}
	}

	return changed
}

func (s *sessionState) advanceBuildingWork(building *buildingEntity, seconds int) bool {
	if building.State != BuildingStateSupplied && building.State != BuildingStateConstructing {
		return false
	}

	previousState := building.State
	previousProgress := building.WorkProgressTicks

	building.WorkProgressTicks += int64(seconds)
	if building.State == BuildingStateSupplied && building.WorkProgressTicks > 0 {
		building.State = BuildingStateConstructing
	}

	if building.WorkProgressTicks < authorityBuildWorkTicks {
		return previousState != building.State || previousProgress != building.WorkProgressTicks
	}

	if building.PendingAction != nil && *building.PendingAction == BuildingWorkUpgrade && building.PendingLevel != nil {
		building.Level = *building.PendingLevel
	}
	building.PendingAction = nil
	building.PendingLevel = nil
	building.State = BuildingStateActive
	building.HP = getBuildingMaxHP(building.Type, building.Level)
	building.MarkedForDemolition = false
	building.WorkProgressTicks = 0
	return true
}

func (s *sessionState) advanceDemolitionWork(building *buildingEntity, seconds int) bool {
	if !building.MarkedForDemolition || building.Type == BuildingMainHall {
		return false
	}

	previousProgress := building.WorkProgressTicks
	building.WorkProgressTicks += int64(seconds)
	if building.WorkProgressTicks < authorityDemolitionTicks {
		return previousProgress != building.WorkProgressTicks
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
	return true
}

func (s *sessionState) advanceRepairWork(building *buildingEntity, seconds int) bool {
	if building.State != BuildingStateDamaged {
		return false
	}

	maxHP := getBuildingMaxHP(building.Type, building.Level)
	if building.HP >= maxHP {
		return false
	}

	changed := false
	if building.WorkProgressTicks == 0 {
		repairCost := repairCostForBuilding(building.Type, building.Level)
		if !hasStockpile(s.Stockpile, repairCost) {
			return false
		}
		s.Stockpile = spendStockpile(s.Stockpile, repairCost)
		changed = true
	}

	previousProgress := building.WorkProgressTicks
	building.WorkProgressTicks += int64(seconds)
	if building.WorkProgressTicks < authorityRepairWorkTicks {
		return changed || previousProgress != building.WorkProgressTicks
	}

	building.HP = maxHP
	building.State = BuildingStateActive
	building.WorkProgressTicks = 0
	return true
}

func (s *sessionState) getBuilding(buildingID string) (*buildingEntity, error) {
	building := s.Buildings[buildingID]
	if building == nil {
		return nil, fmt.Errorf("building not found: %s", buildingID)
	}
	return building, nil
}

func (s *sessionState) getResourceNode(tile TileCoord) (*resourceNodeEntity, error) {
	node := s.ResourceNodes[resourceTileKey(tile)]
	if node == nil {
		return nil, fmt.Errorf("resource node not found: %d,%d", tile.Col, tile.Row)
	}
	return node, nil
}

func (s *sessionState) syncDerivedProgress() {
	if s.Outcome != SessionOutcomeInProgress {
		s.Hostile = nil
		s.RaidCountdownSeconds = 0
		s.DefendRemainingSeconds = 0
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
			s.RaidCountdownSeconds = 0
			s.DefendRemainingSeconds = 0
			s.Phase = SessionPhaseClearRuin
			s.Objective = "长按废弃仓房并拆除，为护山台腾出位置"
			return
		}
		s.RuinBuildingID = nil
	}

	guardTower := s.currentGuardTower()
	if guardTower == nil {
		s.RaidCountdownSeconds = 0
		s.DefendRemainingSeconds = 0
		s.Phase = SessionPhasePlaceGuardTower
		s.Objective = "在空地上放下第一座护山台"
		return
	}

	if s.FirstRaidTriggered && !s.FirstRaidResolved {
		s.RaidCountdownSeconds = 0
		if s.DefendRemainingSeconds <= 0 {
			s.DefendRemainingSeconds = sharedAuthorityConfig.firstRaidPrepSeconds
		}
		s.Phase = SessionPhaseDefend
		s.Objective = "守住首波敌袭，不要让主殿瘫痪"
		return
	}

	if s.FirstRaidResolved {
		s.Hostile = nil
		s.RaidCountdownSeconds = 0
		s.DefendRemainingSeconds = 0
		recoverReason, damagedBuildingCount, regeneratingNodeCount := s.sessionRecoveryStatus()
		if recoverReason != SessionRecoverReasonNone {
			s.Phase = SessionPhaseRecover
			switch recoverReason {
			case SessionRecoverReasonDamagedBuildingsAndRegen:
				s.Objective = "首波敌袭已退去，修复受损建筑并等待资源点恢复"
			case SessionRecoverReasonDamagedBuildings:
				s.Objective = "首波敌袭已退去，修复受损建筑并恢复运转"
			case SessionRecoverReasonResourceRegeneration:
				s.Objective = "首波敌袭已守住，等待采空资源点恢复后进入下一轮筹备"
			default:
				s.Objective = "首波敌袭已退去，等待 authority 完成战后收口"
			}
			return
		}

		s.Phase = SessionPhaseSecondCycleReady
		if damagedBuildingCount == 0 && regeneratingNodeCount == 0 {
			s.Objective = "战后修复与资源刷新已完成，可继续当前 authority 下一轮筹备"
		} else {
			s.Objective = "资源点与库存保持当前 authority 状态，可继续准备下一轮"
		}
		return
	}

	if guardTower.Level < 2 || guardTower.State != BuildingStateActive {
		s.RaidCountdownSeconds = 0
		s.DefendRemainingSeconds = 0
		s.Phase = SessionPhaseUpgradeGuardTower
		s.Objective = "把护山台升到 Lv.2，为首波敌袭做准备"
		return
	}

	if !s.FirstRaidTriggered {
		if s.RaidCountdownSeconds <= 0 {
			s.RaidCountdownSeconds = sharedAuthorityConfig.firstRaidPrepSeconds
		}
		s.DefendRemainingSeconds = 0
		s.Phase = SessionPhaseRaidCountdown
		s.Objective = "护山台已就位，准备迎接第一波敌袭"
		return
	}
}

func (s *sessionState) sessionRecoveryStatus() (SessionRecoverReason, int, int) {
	damagedBuildingCount := s.damagedBuildingCount()
	regeneratingNodeCount := s.regeneratingNodeCount()
	switch {
	case damagedBuildingCount > 0 && regeneratingNodeCount > 0:
		return SessionRecoverReasonDamagedBuildingsAndRegen, damagedBuildingCount, regeneratingNodeCount
	case damagedBuildingCount > 0:
		return SessionRecoverReasonDamagedBuildings, damagedBuildingCount, regeneratingNodeCount
	case regeneratingNodeCount > 0:
		return SessionRecoverReasonResourceRegeneration, damagedBuildingCount, regeneratingNodeCount
	default:
		return SessionRecoverReasonNone, damagedBuildingCount, regeneratingNodeCount
	}
}

func (s *sessionState) damagedBuildingCount() int {
	count := 0
	for _, building := range s.Buildings {
		if building.State == BuildingStateDamaged && building.HP < getBuildingMaxHP(building.Type, building.Level) {
			count++
		}
	}
	return count
}

func (s *sessionState) regeneratingNodeCount() int {
	count := 0
	for _, node := range s.ResourceNodes {
		if node.State == ResourceNodeStateRegenerating {
			count++
		}
	}
	return count
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

func setStockpileValue(pile *Stockpile, kind ResourceKind, amount int) {
	switch kind {
	case ResourceSpiritWood:
		pile.SpiritWood = amount
	case ResourceSpiritStone:
		pile.SpiritStone = amount
	case ResourceHerb:
		pile.Herb = amount
	}
}

func resourceKindOrder() []ResourceKind {
	return []ResourceKind{
		ResourceSpiritWood,
		ResourceSpiritStone,
		ResourceHerb,
	}
}

func stockpileKindsWithPositiveValue(pile Stockpile) []ResourceKind {
	kinds := make([]ResourceKind, 0, 3)
	for _, kind := range resourceKindOrder() {
		if stockpileValue(pile, kind) > 0 {
			kinds = append(kinds, kind)
		}
	}
	return kinds
}

func firstMissingResourceWithStockpile(cost, supplied, stockpile Stockpile) *ResourceKind {
	missing := missingResources(cost, supplied)
	for _, kind := range resourceKindOrder() {
		if stockpileValue(missing, kind) <= 0 {
			continue
		}
		if stockpileValue(stockpile, kind) <= 0 {
			continue
		}
		return resourceKindPtr(kind)
	}
	return nil
}

func mustDefinition(buildingType BuildingType) buildingDefinition {
	def, ok := sharedAuthorityConfig.buildingDefinitions[buildingType]
	if !ok {
		panic("missing building definition")
	}
	return def
}

func mustResourceNodeRule(kind ResourceKind) resourceNodeRule {
	rule, ok := sharedAuthorityConfig.resourceNodeRules[kind]
	if !ok {
		panic("missing resource node rule")
	}
	return rule
}

func labelForBuilding(buildingType BuildingType) string {
	label, ok := sharedAuthorityConfig.buildingLabels[buildingType]
	if !ok {
		return "建筑"
	}
	return label
}

func labelForResource(kind ResourceKind) string {
	label, ok := sharedAuthorityConfig.resourceLabels[kind]
	if !ok {
		return "资源"
	}
	return label
}

func isValidResourceKind(kind ResourceKind) bool {
	_, ok := sharedAuthorityConfig.resourceNodeRules[kind]
	return ok
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

func resourceTileKey(tile TileCoord) string {
	return fmt.Sprintf("%d,%d", tile.Col, tile.Row)
}

func intPtr(v int) *int {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func resourceKindPtr(v ResourceKind) *ResourceKind {
	return &v
}

func tilePtr(v TileCoord) *TileCoord {
	return &v
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func manhattanDistance(left, right TileCoord) int {
	deltaCol := left.Col - right.Col
	if deltaCol < 0 {
		deltaCol = -deltaCol
	}
	deltaRow := left.Row - right.Row
	if deltaRow < 0 {
		deltaRow = -deltaRow
	}
	return deltaCol + deltaRow
}
