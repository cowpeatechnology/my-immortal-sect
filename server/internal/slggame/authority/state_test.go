package authority

import (
	"encoding/json"
	"testing"
)

func TestAuthorityResourceNodesDepleteAndRefresh(t *testing.T) {
	state := newSessionState("test-session")
	resourceTile := TileCoord{Col: 2, Row: 4}

	executeAuthorityCommandForTest(t, state, "collect_stockpile", map[string]any{
		"resourceKind": "spirit_wood",
		"amount":       1,
		"resourceTile": map[string]any{
			"col": resourceTile.Col,
			"row": resourceTile.Row,
		},
	})
	executeAuthorityCommandForTest(t, state, "collect_stockpile", map[string]any{
		"resourceKind": "spirit_wood",
		"amount":       1,
		"resourceTile": map[string]any{
			"col": resourceTile.Col,
			"row": resourceTile.Row,
		},
	})
	executeAuthorityCommandForTest(t, state, "collect_stockpile", map[string]any{
		"resourceKind": "spirit_wood",
		"amount":       1,
		"resourceTile": map[string]any{
			"col": resourceTile.Col,
			"row": resourceTile.Row,
		},
	})

	node, err := state.getResourceNode(resourceTile)
	if err != nil {
		t.Fatalf("get resource node: %v", err)
	}
	if node.State != ResourceNodeStateRegenerating {
		t.Fatalf("expected regenerating node after depletion, got %s", node.State)
	}
	if node.RegenTimerSeconds != node.RegenSeconds {
		t.Fatalf("expected full regen timer after depletion, got %d", node.RegenTimerSeconds)
	}

	state.advanceResourceNodes(node.RegenSeconds - 1)
	if node.State != ResourceNodeStateRegenerating {
		t.Fatalf("expected node to remain regenerating before final tick, got %s", node.State)
	}
	if node.RegenTimerSeconds != 1 {
		t.Fatalf("expected one second remaining before refresh, got %d", node.RegenTimerSeconds)
	}

	state.advanceResourceNodes(1)
	if node.State != ResourceNodeStateAvailable {
		t.Fatalf("expected available node after refresh, got %s", node.State)
	}
	if node.RemainingCharges != node.MaxCharges {
		t.Fatalf("expected refreshed node to restore max charges, got %d", node.RemainingCharges)
	}
}

func TestAuthorityRaidTransitionsThroughRecoverToSecondCycleReady(t *testing.T) {
	state := newReadyForRaidSession()

	if state.Phase != SessionPhaseRaidCountdown {
		t.Fatalf("expected raid_countdown, got %s", state.Phase)
	}
	initialSnapshot := state.snapshot()
	if initialSnapshot.Session.RaidCountdownSeconds != sharedAuthorityConfig.firstRaidPrepSeconds {
		t.Fatalf("expected authority countdown seconds %d, got %d", sharedAuthorityConfig.firstRaidPrepSeconds, initialSnapshot.Session.RaidCountdownSeconds)
	}

	state.advanceResourceNodes(sharedAuthorityConfig.firstRaidPrepSeconds)
	if state.Phase != SessionPhaseDefend {
		t.Fatalf("expected defend after authority countdown, got %s", state.Phase)
	}
	if !state.FirstRaidTriggered {
		t.Fatalf("expected first raid to be marked triggered")
	}
	if state.snapshot().Session.DefendRemainingSeconds != sharedAuthorityConfig.firstRaidPrepSeconds {
		t.Fatalf("expected defend remaining seconds %d, got %d", sharedAuthorityConfig.firstRaidPrepSeconds, state.snapshot().Session.DefendRemainingSeconds)
	}

	guardTowerID := derefString(t, state.GuardTowerID)
	advanceUntilBuildingDamaged(t, state, guardTowerID, 12)
	if state.Buildings[guardTowerID].State != BuildingStateDamaged {
		t.Fatalf("expected guard tower to be damaged, got %s", state.Buildings[guardTowerID].State)
	}

	advanceUntilPhase(t, state, SessionPhaseRecover, sharedAuthorityConfig.firstRaidPrepSeconds+12)
	if state.Phase != SessionPhaseRecover {
		t.Fatalf("expected recover after authority defend window, got %s", state.Phase)
	}
	if state.Outcome != SessionOutcomeInProgress {
		t.Fatalf("expected in_progress outcome after resolving damaged raid, got %s", state.Outcome)
	}
	snapshot := state.snapshot()
	if snapshot.Session.RecoverReason != SessionRecoverReasonDamagedBuildings {
		t.Fatalf("expected damaged-building recover reason immediately after raid, got %s", snapshot.Session.RecoverReason)
	}
	if snapshot.Session.DamagedBuildingCount != 1 {
		t.Fatalf("expected one damaged building immediately after raid, got %d", snapshot.Session.DamagedBuildingCount)
	}
	if snapshot.Session.RegeneratingNodeCount != 0 {
		t.Fatalf("expected regenerated resource nodes before recover, got %d still regenerating", snapshot.Session.RegeneratingNodeCount)
	}

	state.advanceResourceNodes(1)
	if state.Phase != SessionPhaseSecondCycleReady {
		t.Fatalf("expected second_cycle_ready after authority repair closure, got %s", state.Phase)
	}
	if state.Outcome != SessionOutcomeInProgress {
		t.Fatalf("expected in_progress outcome in ready state, got %s", state.Outcome)
	}
	snapshot = state.snapshot()
	if snapshot.Session.RecoverReason != SessionRecoverReasonNone {
		t.Fatalf("expected no recover reason once second cycle is ready, got %s", snapshot.Session.RecoverReason)
	}
	if snapshot.Session.DamagedBuildingCount != 0 {
		t.Fatalf("expected zero damaged buildings in second_cycle_ready, got %d", snapshot.Session.DamagedBuildingCount)
	}
	if snapshot.Session.RegeneratingNodeCount != 0 {
		t.Fatalf("expected zero regenerating nodes in second_cycle_ready, got %d", snapshot.Session.RegeneratingNodeCount)
	}
}

func TestAuthorityBuildProgressAdvancesOnAuthorityTick(t *testing.T) {
	state := newSessionState("test-session")
	if state.RuinBuildingID != nil {
		delete(state.Buildings, *state.RuinBuildingID)
		state.RuinBuildingID = nil
	}

	guardTower := state.addBuilding(BuildingGuardTower, sharedAuthorityConfig.initialRuinTile, BuildingStatePlanned)
	guardTower.Supplied = activeCostForBuilding(guardTower)
	guardTower.State = BuildingStateSupplied
	state.GuardTowerID = stringPtr(guardTower.ID)
	state.syncDerivedProgress()

	disciple := requireSingleDiscipleSnapshot(t, state.snapshot())
	if disciple.AssignmentKind != DiscipleAssignmentBuild {
		t.Fatalf("expected build assignment before work starts, got %s", disciple.AssignmentKind)
	}
	if disciple.WorkProgressTicks != 0 {
		t.Fatalf("expected zero build work progress before authority tick, got %d", disciple.WorkProgressTicks)
	}

	state.advanceResourceNodes(1)
	if guardTower.State != BuildingStateConstructing {
		t.Fatalf("expected constructing after first authority work tick, got %s", guardTower.State)
	}
	disciple = requireSingleDiscipleSnapshot(t, state.snapshot())
	if disciple.AssignmentKind != DiscipleAssignmentBuild {
		t.Fatalf("expected build assignment while constructing, got %s", disciple.AssignmentKind)
	}
	if disciple.WorkProgressTicks != 1 {
		t.Fatalf("expected build work progress 1 after first tick, got %d", disciple.WorkProgressTicks)
	}

	state.advanceResourceNodes(1)
	if guardTower.State != BuildingStateActive {
		t.Fatalf("expected active after second authority work tick, got %s", guardTower.State)
	}
	if guardTower.WorkProgressTicks != 0 {
		t.Fatalf("expected build work progress reset after completion, got %d", guardTower.WorkProgressTicks)
	}
}

func TestAuthorityHaulProgressAdvancesOnAuthorityTick(t *testing.T) {
	state := newSessionState("test-session")
	if state.RuinBuildingID != nil {
		delete(state.Buildings, *state.RuinBuildingID)
		state.RuinBuildingID = nil
	}

	guardTower := state.addBuilding(BuildingGuardTower, sharedAuthorityConfig.initialRuinTile, BuildingStatePlanned)
	state.GuardTowerID = stringPtr(guardTower.ID)
	state.Stockpile = activeCostForBuilding(guardTower)
	state.syncDerivedProgress()

	disciple := requireSingleDiscipleSnapshot(t, state.snapshot())
	if disciple.AssignmentKind != DiscipleAssignmentHaul {
		t.Fatalf("expected haul assignment before authority delivery tick, got %s", disciple.AssignmentKind)
	}

	state.advanceResourceNodes(1)
	if guardTower.State != BuildingStatePlanned {
		t.Fatalf("expected guard tower to remain planned until all resources are delivered, got %s", guardTower.State)
	}
	if guardTower.Supplied.SpiritWood != 1 || guardTower.Supplied.SpiritStone != 0 {
		t.Fatalf("expected first authority haul tick to deliver one wood, got %+v", guardTower.Supplied)
	}
	disciple = requireSingleDiscipleSnapshot(t, state.snapshot())
	if disciple.AssignmentKind != DiscipleAssignmentHaul {
		t.Fatalf("expected haul assignment while more resources are still pending, got %s", disciple.AssignmentKind)
	}
	if disciple.TargetResourceKind == nil || *disciple.TargetResourceKind != ResourceSpiritWood {
		t.Fatalf("expected second haul target to remain spirit_wood, got %+v", disciple.TargetResourceKind)
	}

	state.advanceResourceNodes(1)
	if guardTower.State != BuildingStatePlanned {
		t.Fatalf("expected guard tower to remain planned until the final resource is delivered, got %s", guardTower.State)
	}
	if guardTower.Supplied.SpiritWood != 2 || guardTower.Supplied.SpiritStone != 0 {
		t.Fatalf("expected second authority haul tick to deliver the second wood, got %+v", guardTower.Supplied)
	}
	disciple = requireSingleDiscipleSnapshot(t, state.snapshot())
	if disciple.AssignmentKind != DiscipleAssignmentHaul {
		t.Fatalf("expected haul assignment while spirit_stone is still pending, got %s", disciple.AssignmentKind)
	}
	if disciple.TargetResourceKind == nil || *disciple.TargetResourceKind != ResourceSpiritStone {
		t.Fatalf("expected final haul target to switch to spirit_stone, got %+v", disciple.TargetResourceKind)
	}

	state.advanceResourceNodes(1)
	if guardTower.State != BuildingStateSupplied {
		t.Fatalf("expected guard tower to become supplied after final authority haul tick, got %s", guardTower.State)
	}
	if guardTower.Supplied != activeCostForBuilding(guardTower) {
		t.Fatalf("expected supplied stockpile to match build cost after authority haul ticks, got %+v", guardTower.Supplied)
	}
	if state.Stockpile != zeroStockpile() {
		t.Fatalf("expected authority haul ticks to spend the reserved stockpile, got %+v", state.Stockpile)
	}
}

func TestAuthorityRepairProgressAdvancesOnAuthorityTick(t *testing.T) {
	state := newReadyForRaidSession()
	guardTowerID := advanceIntoRecoverFromAuthorityRaid(t, state)

	repairCost := repairCostForBuilding(BuildingGuardTower, 2)
	beforeStockpile := state.Stockpile
	guardTower := state.Buildings[guardTowerID]
	if guardTower == nil {
		t.Fatalf("expected guard tower %s", guardTowerID)
	}
	startingProgress := guardTower.WorkProgressTicks

	state.advanceResourceNodes(1)

	if startingProgress == 0 {
		if guardTower.State != BuildingStateDamaged {
			t.Fatalf("expected damaged guard tower after first repair tick, got %s", guardTower.State)
		}
		if guardTower.WorkProgressTicks != 1 {
			t.Fatalf("expected repair work progress 1 after first tick, got %d", guardTower.WorkProgressTicks)
		}
		expectedStockpile := spendStockpile(beforeStockpile, repairCost)
		if state.Stockpile != expectedStockpile {
			t.Fatalf("expected repair cost to be reserved on first authority repair tick, got %+v want %+v", state.Stockpile, expectedStockpile)
		}

		disciple := requireSingleDiscipleSnapshot(t, state.snapshot())
		if disciple.AssignmentKind != DiscipleAssignmentRepair {
			t.Fatalf("expected repair assignment while repair is in progress, got %s", disciple.AssignmentKind)
		}
		if disciple.WorkProgressTicks != 1 {
			t.Fatalf("expected repair work progress 1 in snapshot, got %d", disciple.WorkProgressTicks)
		}

		state.advanceResourceNodes(1)
	}

	if guardTower.State != BuildingStateActive {
		t.Fatalf("expected active guard tower after repair closure, got %s", guardTower.State)
	}
	if guardTower.WorkProgressTicks != 0 {
		t.Fatalf("expected repair work progress reset after completion, got %d", guardTower.WorkProgressTicks)
	}
	if guardTower.HP != getBuildingMaxHP(guardTower.Type, guardTower.Level) {
		t.Fatalf("expected guard tower HP restored after repair, got %d", guardTower.HP)
	}
}

func TestAuthorityRejectsLegacyClientOwnedDamageReports(t *testing.T) {
	state := newReadyForRaidSession()

	result, err := state.executeCommand(CommandEnvelope{
		Name:    "report_building_damage",
		Payload: json.RawMessage(`{"buildingId":"building-3","hp":12}`),
	})
	if err == nil {
		t.Fatalf("expected legacy damage report to be rejected, got %+v", result)
	}
	if err.Error() != "report_building_damage is no longer allowed; authority owns combat damage" {
		t.Fatalf("unexpected damage-report rejection: %v", err)
	}
}

func TestAuthorityRejectsLegacyClientOwnedBuildAndRepairCommands(t *testing.T) {
	state := newReadyForRaidSession()

	legacyCommands := []string{
		"start_building_work",
		"complete_building_work",
		"complete_repair",
	}

	for _, name := range legacyCommands {
		result, err := state.executeCommand(CommandEnvelope{
			Name:    name,
			Payload: json.RawMessage(`{"buildingId":"building-3"}`),
		})
		if err == nil {
			t.Fatalf("expected legacy command %s to be rejected, got result %+v", name, result)
		}
	}
}

func TestAuthorityMainHallDamageTriggersDefeat(t *testing.T) {
	state := newReadyForRaidSession()
	state.advanceResourceNodes(sharedAuthorityConfig.firstRaidPrepSeconds)
	guardTowerID := derefString(t, state.GuardTowerID)
	state.Buildings[guardTowerID].HP = 0
	state.Buildings[guardTowerID].State = BuildingStateDamaged
	state.DiscipleHP = 0
	mainHall := state.Buildings["building-1"]
	if mainHall == nil {
		t.Fatalf("expected main hall building-1")
	}
	state.Hostile = &hostileEntity{
		ID:               authorityHostileID,
		ArchetypeID:      UnitArchetypeBanditScout,
		Name:             mustUnitArchetype(UnitArchetypeBanditScout).DisplayName,
		Tile:             state.hostileAttackTile(mainHall),
		HP:               99,
		VisualState:      UnitVisualStateAttacking,
		TargetBuildingID: stringPtr(mainHall.ID),
		Active:           true,
	}
	advanceUntilOutcome(t, state, SessionOutcomeDefeat, 12)

	if state.Phase != SessionPhaseDefeat {
		t.Fatalf("expected defeat phase after main hall loss, got %s", state.Phase)
	}
	if state.Outcome != SessionOutcomeDefeat {
		t.Fatalf("expected defeat outcome after main hall loss, got %s", state.Outcome)
	}
}

func TestAuthorityExpireSessionTriggersDefeat(t *testing.T) {
	state := newReadyForRaidSession()

	executeAuthorityCommandForTest(t, state, "expire_session", map[string]any{})

	if state.Phase != SessionPhaseDefeat {
		t.Fatalf("expected defeat phase after expiration, got %s", state.Phase)
	}
	if state.Outcome != SessionOutcomeDefeat {
		t.Fatalf("expected defeat outcome after expiration, got %s", state.Outcome)
	}
}

func TestAuthoritySnapshotIncludesDemolitionAssignment(t *testing.T) {
	state := newSessionState("test-session")
	ruinID := derefString(t, state.RuinBuildingID)

	executeAuthorityCommandForTest(t, state, "toggle_demolition", map[string]any{
		"buildingId": ruinID,
	})

	disciple := requireSingleDiscipleSnapshot(t, state.snapshot())
	if disciple.AssignmentKind != DiscipleAssignmentDemolish {
		t.Fatalf("expected demolish assignment, got %s", disciple.AssignmentKind)
	}
	if disciple.TargetBuildingID == nil || *disciple.TargetBuildingID != ruinID {
		t.Fatalf("expected ruin demolition target, got %+v", disciple.TargetBuildingID)
	}
}

func TestAuthorityDemolitionProgressAdvancesOnAuthorityTick(t *testing.T) {
	state := newSessionState("test-session")
	ruinID := derefString(t, state.RuinBuildingID)

	executeAuthorityCommandForTest(t, state, "toggle_demolition", map[string]any{
		"buildingId": ruinID,
	})

	disciple := requireSingleDiscipleSnapshot(t, state.snapshot())
	if disciple.AssignmentKind != DiscipleAssignmentDemolish {
		t.Fatalf("expected demolish assignment before work starts, got %s", disciple.AssignmentKind)
	}
	if disciple.WorkProgressTicks != 0 {
		t.Fatalf("expected zero demolish work progress before authority tick, got %d", disciple.WorkProgressTicks)
	}

	state.advanceResourceNodes(1)
	ruin := state.Buildings[ruinID]
	if ruin == nil {
		t.Fatalf("expected ruin to still exist after first demolition tick")
	}
	if ruin.WorkProgressTicks != 1 {
		t.Fatalf("expected demolish work progress 1 after first tick, got %d", ruin.WorkProgressTicks)
	}
	disciple = requireSingleDiscipleSnapshot(t, state.snapshot())
	if disciple.AssignmentKind != DiscipleAssignmentDemolish {
		t.Fatalf("expected demolish assignment while demolition is in progress, got %s", disciple.AssignmentKind)
	}
	if disciple.WorkProgressTicks != 1 {
		t.Fatalf("expected demolish work progress 1 in snapshot, got %d", disciple.WorkProgressTicks)
	}

	state.advanceResourceNodes(1)
	if _, ok := state.Buildings[ruinID]; ok {
		t.Fatalf("expected ruin to be removed after second demolition tick")
	}
	if state.RuinBuildingID != nil {
		t.Fatalf("expected ruin building id cleared after demolition, got %v", *state.RuinBuildingID)
	}
	if state.Phase != SessionPhasePlaceGuardTower {
		t.Fatalf("expected phase to advance to place_guard_tower after demolition, got %s", state.Phase)
	}
	if state.Stockpile.SpiritWood != 1 || state.Stockpile.SpiritStone != 1 || state.Stockpile.Herb != 0 {
		t.Fatalf("expected demolition salvage stockpile wood=1 stone=1 herb=0, got %+v", state.Stockpile)
	}
}

func TestAuthoritySnapshotIncludesGatherHaulAndBuildAssignments(t *testing.T) {
	state := newSessionState("test-session")
	if state.RuinBuildingID != nil {
		delete(state.Buildings, *state.RuinBuildingID)
		state.RuinBuildingID = nil
	}

	guardTower := state.addBuilding(BuildingGuardTower, sharedAuthorityConfig.initialRuinTile, BuildingStatePlanned)
	state.GuardTowerID = stringPtr(guardTower.ID)
	state.syncDerivedProgress()

	disciple := requireSingleDiscipleSnapshot(t, state.snapshot())
	if disciple.AssignmentKind != DiscipleAssignmentGather {
		t.Fatalf("expected gather assignment for unsupplied guard tower, got %s", disciple.AssignmentKind)
	}
	if disciple.TargetResourceKind == nil || *disciple.TargetResourceKind != ResourceSpiritWood {
		t.Fatalf("expected spirit_wood gather target, got %+v", disciple.TargetResourceKind)
	}

	state.Stockpile = Stockpile{SpiritWood: 1}
	disciple = requireSingleDiscipleSnapshot(t, state.snapshot())
	if disciple.AssignmentKind != DiscipleAssignmentHaul {
		t.Fatalf("expected haul assignment once stockpile is available, got %s", disciple.AssignmentKind)
	}
	if disciple.TargetBuildingID == nil || *disciple.TargetBuildingID != guardTower.ID {
		t.Fatalf("expected haul target %s, got %+v", guardTower.ID, disciple.TargetBuildingID)
	}
	if disciple.TargetResourceKind == nil || *disciple.TargetResourceKind != ResourceSpiritWood {
		t.Fatalf("expected spirit_wood haul target, got %+v", disciple.TargetResourceKind)
	}

	guardTower.Supplied = activeCostForBuilding(guardTower)
	guardTower.State = BuildingStateSupplied
	state.syncDerivedProgress()
	disciple = requireSingleDiscipleSnapshot(t, state.snapshot())
	if disciple.AssignmentKind != DiscipleAssignmentBuild {
		t.Fatalf("expected build assignment for supplied guard tower, got %s", disciple.AssignmentKind)
	}
	if disciple.TargetBuildingID == nil || *disciple.TargetBuildingID != guardTower.ID {
		t.Fatalf("expected build target %s, got %+v", guardTower.ID, disciple.TargetBuildingID)
	}
}

func TestAuthoritySnapshotIncludesGuardAssignmentDuringDefend(t *testing.T) {
	state := newReadyForRaidSession()
	executeAuthorityCommandForTest(t, state, "trigger_first_raid", map[string]any{})

	disciple := requireSingleDiscipleSnapshot(t, state.snapshot())
	if disciple.AssignmentKind != DiscipleAssignmentGuard {
		t.Fatalf("expected guard assignment during defend, got %s", disciple.AssignmentKind)
	}
	if disciple.TargetBuildingID == nil || *disciple.TargetBuildingID != derefString(t, state.GuardTowerID) {
		t.Fatalf("expected guard target to be current guard tower, got %+v", disciple.TargetBuildingID)
	}
}

func TestAuthoritySnapshotIncludesRepairAssignmentDuringRecover(t *testing.T) {
	state := newReadyForRaidSession()
	guardTowerID := advanceIntoRecoverFromAuthorityRaid(t, state)

	disciple := requireSingleDiscipleSnapshot(t, state.snapshot())
	if disciple.AssignmentKind != DiscipleAssignmentRepair {
		t.Fatalf("expected repair assignment during recover, got %s", disciple.AssignmentKind)
	}
	if disciple.TargetBuildingID == nil || *disciple.TargetBuildingID != guardTowerID {
		t.Fatalf("expected repair target %s, got %+v", guardTowerID, disciple.TargetBuildingID)
	}
}

func newReadyForRaidSession() *sessionState {
	state := newSessionState("test-session")
	if state.RuinBuildingID != nil {
		delete(state.Buildings, *state.RuinBuildingID)
		state.RuinBuildingID = nil
	}

	guardTower := state.addBuilding(BuildingGuardTower, sharedAuthorityConfig.initialRuinTile, BuildingStateActive)
	guardTower.Level = 2
	guardTower.HP = getBuildingMaxHP(guardTower.Type, guardTower.Level)
	guardTower.PendingAction = nil
	guardTower.PendingLevel = nil
	state.GuardTowerID = stringPtr(guardTower.ID)
	state.Stockpile = Stockpile{
		SpiritWood:  10,
		SpiritStone: 10,
		Herb:        10,
	}
	state.syncDerivedProgress()
	return state
}

func executeAuthorityCommandForTest(t *testing.T, state *sessionState, name string, payload any) CommandResult {
	t.Helper()

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	result, err := state.executeCommand(CommandEnvelope{
		Name:    name,
		Payload: rawPayload,
	})
	if err != nil {
		t.Fatalf("execute command %s: %v", name, err)
	}
	return result
}

func derefString(t *testing.T, value *string) string {
	t.Helper()
	if value == nil {
		t.Fatalf("expected non-nil string pointer")
	}
	return *value
}

func requireSingleDiscipleSnapshot(t *testing.T, snapshot SessionSnapshot) DiscipleSnapshot {
	t.Helper()
	if len(snapshot.Disciples) != 1 {
		t.Fatalf("expected exactly one disciple snapshot, got %d", len(snapshot.Disciples))
	}
	return snapshot.Disciples[0]
}

func advanceUntilPhase(t *testing.T, state *sessionState, phase SessionPhase, limit int) {
	t.Helper()
	for step := 0; step < limit; step++ {
		if state.Phase == phase {
			return
		}
		state.advanceResourceNodes(1)
	}
	t.Fatalf("expected phase %s within %d ticks, got %s", phase, limit, state.Phase)
}

func advanceUntilOutcome(t *testing.T, state *sessionState, outcome SessionOutcome, limit int) {
	t.Helper()
	for step := 0; step < limit; step++ {
		if state.Outcome == outcome {
			return
		}
		state.advanceResourceNodes(1)
	}
	t.Fatalf("expected outcome %s within %d ticks, got %s", outcome, limit, state.Outcome)
}

func advanceUntilBuildingDamaged(t *testing.T, state *sessionState, buildingID string, limit int) {
	t.Helper()
	for step := 0; step < limit; step++ {
		building := state.Buildings[buildingID]
		if building != nil && building.State == BuildingStateDamaged {
			return
		}
		state.advanceResourceNodes(1)
	}
	building := state.Buildings[buildingID]
	currentState := BuildingState("")
	if building != nil {
		currentState = building.State
	}
	t.Fatalf("expected building %s to be damaged within %d ticks, got %s", buildingID, limit, currentState)
}

func advanceIntoRecoverFromAuthorityRaid(t *testing.T, state *sessionState) string {
	t.Helper()
	advanceUntilPhase(t, state, SessionPhaseDefend, sharedAuthorityConfig.firstRaidPrepSeconds+1)
	guardTowerID := derefString(t, state.GuardTowerID)
	advanceUntilBuildingDamaged(t, state, guardTowerID, 12)
	advanceUntilPhase(t, state, SessionPhaseRecover, sharedAuthorityConfig.firstRaidPrepSeconds+12)
	return guardTowerID
}
