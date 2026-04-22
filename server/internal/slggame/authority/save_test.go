package authority

import (
	"encoding/json"
	"reflect"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func TestAuthoritySessionSaveEnvelopeRoundTrip(t *testing.T) {
	state := newReadyForRaidSession()
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
	state.advanceResourceNodes(4)

	save, err := state.exportSaveEnvelope()
	if err != nil {
		t.Fatalf("export save envelope: %v", err)
	}

	if save.SchemaVersion != authoritySessionSchemaVersion {
		t.Fatalf("expected schema version %d, got %d", authoritySessionSchemaVersion, save.SchemaVersion)
	}
	if save.SimulationVersion != authoritySessionSimulationVersion {
		t.Fatalf("expected simulation version %d, got %d", authoritySessionSimulationVersion, save.SimulationVersion)
	}
	if save.ConfigVersion != authoritySessionConfigVersion {
		t.Fatalf("expected config version %d, got %d", authoritySessionConfigVersion, save.ConfigVersion)
	}
	if save.GameTick != state.GameTick {
		t.Fatalf("expected saved game tick %d, got %d", state.GameTick, save.GameTick)
	}
	if len(save.StateBlob) == 0 {
		t.Fatalf("expected non-empty state blob")
	}

	restored, err := restoreSessionStateFromSaveEnvelope(save)
	if err != nil {
		t.Fatalf("restore session state: %v", err)
	}

	if !reflect.DeepEqual(state.snapshot(), restored.snapshot()) {
		t.Fatalf("expected restored snapshot to match saved snapshot")
	}
	if restored.nextBuildingSeq != state.nextBuildingSeq {
		t.Fatalf("expected next building seq %d, got %d", state.nextBuildingSeq, restored.nextBuildingSeq)
	}
}

func TestAuthoritySessionRestoreContinuesProgression(t *testing.T) {
	state := newReadyForRaidSession()
	state.advanceResourceNodes(3)

	save, err := state.exportSaveEnvelope()
	if err != nil {
		t.Fatalf("export save envelope: %v", err)
	}

	restored, err := restoreSessionStateFromSaveEnvelope(save)
	if err != nil {
		t.Fatalf("restore session state: %v", err)
	}

	beforeTick := restored.GameTick
	restored.advanceResourceNodes(2)
	if restored.GameTick != beforeTick+2 {
		t.Fatalf("expected restored game tick to continue from %d to %d, got %d", beforeTick, beforeTick+2, restored.GameTick)
	}

	executeAuthorityCommandForTest(t, restored, "place_building", map[string]any{
		"buildingType": "herb_garden",
		"origin": map[string]any{
			"col": 1,
			"row": 1,
		},
	})

	if restored.Buildings["building-4"] == nil {
		t.Fatalf("expected restored session to continue building sequence and create building-4")
	}
}

func TestAuthoritySessionSaveStateBlobUsesProtobufWireFormat(t *testing.T) {
	state := newReadyForRaidSession()

	save, err := state.exportSaveEnvelope()
	if err != nil {
		t.Fatalf("export save envelope: %v", err)
	}

	if json.Valid(save.StateBlob) {
		t.Fatalf("expected binary protobuf state blob, got JSON payload")
	}

	fieldNum, fieldType, n := protowire.ConsumeTag(save.StateBlob)
	if n < 0 {
		t.Fatalf("expected protobuf wire tag, got parse error: %v", protowire.ParseError(n))
	}
	if fieldNum != 1 || fieldType != protowire.BytesType {
		t.Fatalf("expected protobuf field 1 bytes header, got field %d type %d", fieldNum, fieldType)
	}
}

func TestAuthoritySessionSaveRestoresInProgressAuthorityWork(t *testing.T) {
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

	state.advanceResourceNodes(1)
	if guardTower.WorkProgressTicks != 1 {
		t.Fatalf("expected build work progress 1 before save, got %d", guardTower.WorkProgressTicks)
	}

	save, err := state.exportSaveEnvelope()
	if err != nil {
		t.Fatalf("export save envelope: %v", err)
	}

	restored, err := restoreSessionStateFromSaveEnvelope(save)
	if err != nil {
		t.Fatalf("restore session state: %v", err)
	}

	restoredGuardTower := restored.Buildings[guardTower.ID]
	if restoredGuardTower == nil {
		t.Fatalf("expected restored guard tower %s", guardTower.ID)
	}
	if restoredGuardTower.WorkProgressTicks != 1 {
		t.Fatalf("expected restored work progress 1, got %d", restoredGuardTower.WorkProgressTicks)
	}

	restored.advanceResourceNodes(1)
	if restoredGuardTower.State != BuildingStateActive {
		t.Fatalf("expected restored authority work to complete building, got %s", restoredGuardTower.State)
	}
}

func TestAuthoritySessionSaveRestoresAuthorityOwnedWorkerSnapshots(t *testing.T) {
	makePlannedGuardTowerState := func() (*sessionState, *buildingEntity) {
		state := newSessionState("test-session")
		if state.RuinBuildingID != nil {
			delete(state.Buildings, *state.RuinBuildingID)
			state.RuinBuildingID = nil
		}

		guardTower := state.addBuilding(BuildingGuardTower, sharedAuthorityConfig.initialRuinTile, BuildingStatePlanned)
		state.GuardTowerID = stringPtr(guardTower.ID)
		state.syncDerivedProgress()
		return state, guardTower
	}

	testCases := []struct {
		name       string
		setup      func(t *testing.T) *sessionState
		wantWorker DiscipleAssignmentKind
	}{
		{
			name: "gather assignment",
			setup: func(t *testing.T) *sessionState {
				t.Helper()
				state, _ := makePlannedGuardTowerState()
				return state
			},
			wantWorker: DiscipleAssignmentGather,
		},
		{
			name: "haul assignment",
			setup: func(t *testing.T) *sessionState {
				t.Helper()
				state, _ := makePlannedGuardTowerState()
				state.Stockpile = Stockpile{SpiritWood: 1}
				state.syncDerivedProgress()
				return state
			},
			wantWorker: DiscipleAssignmentHaul,
		},
		{
			name: "build assignment",
			setup: func(t *testing.T) *sessionState {
				t.Helper()
				state, guardTower := makePlannedGuardTowerState()
				guardTower.Supplied = activeCostForBuilding(guardTower)
				guardTower.State = BuildingStateSupplied
				state.syncDerivedProgress()
				return state
			},
			wantWorker: DiscipleAssignmentBuild,
		},
		{
			name: "guard assignment",
			setup: func(t *testing.T) *sessionState {
				t.Helper()
				state := newReadyForRaidSession()
				state.advanceResourceNodes(sharedAuthorityConfig.firstRaidPrepSeconds)
				return state
			},
			wantWorker: DiscipleAssignmentGuard,
		},
		{
			name: "repair assignment",
			setup: func(t *testing.T) *sessionState {
				t.Helper()
				state := newReadyForRaidSession()
				advanceIntoRecoverFromAuthorityRaid(t, state)
				return state
			},
			wantWorker: DiscipleAssignmentRepair,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := tc.setup(t)
			before := state.snapshot()
			disciple := requireSingleDiscipleSnapshot(t, before)
			if disciple.AssignmentKind != tc.wantWorker {
				t.Fatalf("expected worker assignment %s before save, got %s", tc.wantWorker, disciple.AssignmentKind)
			}

			save, err := state.exportSaveEnvelope()
			if err != nil {
				t.Fatalf("export save envelope: %v", err)
			}

			restored, err := restoreSessionStateFromSaveEnvelope(save)
			if err != nil {
				t.Fatalf("restore session state: %v", err)
			}

			after := restored.snapshot()
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("expected restored snapshot to preserve authority-owned worker state")
			}
		})
	}
}

func TestAuthoritySessionRestoreContinuesRecoverClosureFromAuthorityWorkerState(t *testing.T) {
	state := newReadyForRaidSession()
	guardTowerID := advanceIntoRecoverFromAuthorityRaid(t, state)

	before := state.snapshot()
	if before.Session.Phase != SessionPhaseRecover {
		t.Fatalf("expected recover phase before save, got %s", before.Session.Phase)
	}
	disciple := requireSingleDiscipleSnapshot(t, before)
	if disciple.AssignmentKind != DiscipleAssignmentRepair {
		t.Fatalf("expected repair assignment before save, got %s", disciple.AssignmentKind)
	}
	if disciple.WorkProgressTicks != 1 {
		t.Fatalf("expected repair work progress 1 before save, got %d", disciple.WorkProgressTicks)
	}

	save, err := state.exportSaveEnvelope()
	if err != nil {
		t.Fatalf("export save envelope: %v", err)
	}

	restored, err := restoreSessionStateFromSaveEnvelope(save)
	if err != nil {
		t.Fatalf("restore session state: %v", err)
	}

	afterRestore := restored.snapshot()
	if !reflect.DeepEqual(before, afterRestore) {
		t.Fatalf("expected restored snapshot to preserve mid-recover authority state")
	}

	restored.advanceResourceNodes(1)

	guardTower := restored.Buildings[guardTowerID]
	if guardTower == nil {
		t.Fatalf("expected restored guard tower %s", guardTowerID)
	}
	if guardTower.State != BuildingStateActive {
		t.Fatalf("expected restored repair closure to activate guard tower, got %s", guardTower.State)
	}
	if guardTower.WorkProgressTicks != 0 {
		t.Fatalf("expected repair work progress reset after restored closure, got %d", guardTower.WorkProgressTicks)
	}
	if restored.Phase != SessionPhaseSecondCycleReady {
		t.Fatalf("expected second_cycle_ready after restored repair closure, got %s", restored.Phase)
	}

	finalSnapshot := restored.snapshot()
	if finalSnapshot.Session.RecoverReason != SessionRecoverReasonNone {
		t.Fatalf("expected cleared recover reason after restored closure, got %s", finalSnapshot.Session.RecoverReason)
	}
	if finalSnapshot.Session.DamagedBuildingCount != 0 {
		t.Fatalf("expected no damaged buildings after restored closure, got %d", finalSnapshot.Session.DamagedBuildingCount)
	}
}
