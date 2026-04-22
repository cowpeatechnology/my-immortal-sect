package authority

import (
	"reflect"
	"testing"
)

type mainlineReplayHarness struct {
	t                  *testing.T
	state              *sessionState
	upgradeRequested   bool
	raidDamageReported bool
	restoreApplied     bool
	checkpoints        map[string]SessionSnapshot
}

func newMainlineReplayHarness(t *testing.T) *mainlineReplayHarness {
	t.Helper()
	return &mainlineReplayHarness{
		t:           t,
		state:       newSessionState("test-session"),
		checkpoints: map[string]SessionSnapshot{},
	}
}

func (h *mainlineReplayHarness) runToSecondCycleReady(restoreAt func(SessionSnapshot) bool) {
	h.t.Helper()

	var lastSnapshot SessionSnapshot
	for step := 0; step < 512; step++ {
		snapshot := h.state.snapshot()
		lastSnapshot = snapshot
		h.captureCheckpoint(snapshot)

		if snapshot.Session.Phase == SessionPhaseSecondCycleReady {
			return
		}

		if restoreAt != nil && !h.restoreApplied && restoreAt(snapshot) {
			h.restoreRoundTrip(snapshot)
			continue
		}

		h.advance(snapshot)
	}

	disciple := requireSingleDiscipleSnapshot(h.t, lastSnapshot)
	h.t.Fatalf(
		"mainline replay did not reach second_cycle_ready within step budget; phase=%s assignment=%s guardTower=%v damaged=%d regenerating=%d stockpile=%+v",
		lastSnapshot.Session.Phase,
		disciple.AssignmentKind,
		lastSnapshot.Session.GuardTowerID,
		lastSnapshot.Session.DamagedBuildingCount,
		lastSnapshot.Session.RegeneratingNodeCount,
		lastSnapshot.Stockpile,
	)
}

func (h *mainlineReplayHarness) captureCheckpoint(snapshot SessionSnapshot) {
	h.captureOnce("phase:"+string(snapshot.Session.Phase), snapshot)

	guardTowerID := snapshot.Session.GuardTowerID
	if guardTowerID == nil {
		return
	}

	for _, building := range snapshot.Buildings {
		if building.ID != *guardTowerID {
			continue
		}
		if building.Type == BuildingGuardTower && building.State == BuildingStateActive && building.Level >= 2 {
			h.captureOnce("guard_tower:lv2_active", snapshot)
		}
	}

	disciple := requireSingleDiscipleSnapshot(h.t, snapshot)
	if snapshot.Session.Phase == SessionPhaseRecover && disciple.AssignmentKind == DiscipleAssignmentRepair {
		h.captureOnce("recover:repair_assignment", snapshot)
		if disciple.WorkProgressTicks > 0 {
			h.captureOnce("recover:repair_in_progress", snapshot)
		}
	}
}

func (h *mainlineReplayHarness) captureOnce(label string, snapshot SessionSnapshot) {
	if _, ok := h.checkpoints[label]; ok {
		return
	}
	h.checkpoints[label] = snapshot
}

func (h *mainlineReplayHarness) restoreRoundTrip(before SessionSnapshot) {
	h.t.Helper()

	save, err := h.state.exportSaveEnvelope()
	if err != nil {
		h.t.Fatalf("export save envelope: %v", err)
	}

	restored, err := restoreSessionStateFromSaveEnvelope(save)
	if err != nil {
		h.t.Fatalf("restore session state: %v", err)
	}

	after := restored.snapshot()
	if !reflect.DeepEqual(before, after) {
		h.t.Fatalf("expected replay restore snapshot to match pre-restore checkpoint")
	}

	h.state = restored
	h.restoreApplied = true
}

func (h *mainlineReplayHarness) advance(snapshot SessionSnapshot) {
	h.t.Helper()

	switch snapshot.Session.Phase {
	case SessionPhaseClearRuin:
		ruinID := derefString(h.t, snapshot.Session.RuinBuildingID)
		if !h.state.Buildings[ruinID].MarkedForDemolition {
			executeAuthorityCommandForTest(h.t, h.state, "toggle_demolition", map[string]any{
				"buildingId": ruinID,
			})
			return
		}
		h.state.advanceResourceNodes(1)
	case SessionPhasePlaceGuardTower:
		executeAuthorityCommandForTest(h.t, h.state, "place_building", map[string]any{
			"buildingType": BuildingGuardTower,
			"origin": map[string]any{
				"col": sharedAuthorityConfig.initialRuinTile.Col,
				"row": sharedAuthorityConfig.initialRuinTile.Row,
			},
		})
	case SessionPhaseUpgradeGuardTower:
		h.advanceUpgrade(snapshot)
	case SessionPhaseRaidCountdown:
		h.state.advanceResourceNodes(1)
	case SessionPhaseDefend:
		h.advanceDefend(snapshot)
	case SessionPhaseRecover:
		if snapshot.Session.DamagedBuildingCount == 0 {
			h.state.advanceResourceNodes(1)
			return
		}
		h.advanceWorkerDriven(snapshot)
	default:
		h.t.Fatalf("unexpected phase during replay: %s", snapshot.Session.Phase)
	}
}

func (h *mainlineReplayHarness) advanceUpgrade(snapshot SessionSnapshot) {
	h.t.Helper()

	guardTowerID := derefString(h.t, snapshot.Session.GuardTowerID)
	guardTower := h.state.Buildings[guardTowerID]
	if guardTower == nil {
		h.t.Fatalf("expected current guard tower %s", guardTowerID)
	}

	if guardTower.State == BuildingStateActive && guardTower.Level == 1 && !h.upgradeRequested {
		executeAuthorityCommandForTest(h.t, h.state, "request_upgrade", map[string]any{
			"buildingId": guardTowerID,
		})
		h.upgradeRequested = true
		return
	}

	h.advanceWorkerDriven(snapshot)
}

func (h *mainlineReplayHarness) advanceDefend(snapshot SessionSnapshot) {
	h.t.Helper()

	if !h.raidDamageReported && snapshot.Session.DamagedBuildingCount > 0 {
		h.raidDamageReported = true
	}
	h.state.advanceResourceNodes(1)
}

func (h *mainlineReplayHarness) advanceWorkerDriven(snapshot SessionSnapshot) {
	h.t.Helper()

	disciple := requireSingleDiscipleSnapshot(h.t, snapshot)
	switch disciple.AssignmentKind {
	case DiscipleAssignmentGather:
		if disciple.TargetResourceKind == nil || disciple.TargetTile == nil {
			h.t.Fatalf("expected gather assignment targets, got %+v", disciple)
		}
		executeAuthorityCommandForTest(h.t, h.state, "collect_stockpile", map[string]any{
			"resourceKind": *disciple.TargetResourceKind,
			"amount":       1,
			"resourceTile": map[string]any{
				"col": disciple.TargetTile.Col,
				"row": disciple.TargetTile.Row,
			},
		})
	case DiscipleAssignmentHaul:
		h.state.advanceResourceNodes(1)
	case DiscipleAssignmentBuild, DiscipleAssignmentRepair, DiscipleAssignmentGuard:
		h.state.advanceResourceNodes(1)
	case DiscipleAssignmentDemolish:
		h.state.advanceResourceNodes(1)
	case DiscipleAssignmentIdle:
		if snapshot.Session.Phase != SessionPhaseRecover {
			h.t.Fatalf("unexpected idle assignment during replay phase %s", snapshot.Session.Phase)
		}
		h.state.advanceResourceNodes(1)
	default:
		h.t.Fatalf("unexpected worker assignment during replay phase %s: %s", snapshot.Session.Phase, disciple.AssignmentKind)
	}
}

func requireReplayCheckpoint(t *testing.T, checkpoints map[string]SessionSnapshot, label string) SessionSnapshot {
	t.Helper()
	snapshot, ok := checkpoints[label]
	if !ok {
		t.Fatalf("expected replay checkpoint %q", label)
	}
	return snapshot
}

func TestAuthorityMainlineReplayHarnessReachesSecondCycleReady(t *testing.T) {
	harness := newMainlineReplayHarness(t)

	harness.runToSecondCycleReady(nil)

	required := []string{
		"phase:clear_ruin",
		"phase:place_guard_tower",
		"phase:upgrade_guard_tower",
		"guard_tower:lv2_active",
		"phase:raid_countdown",
		"phase:defend",
		"phase:recover",
		"recover:repair_assignment",
		"recover:repair_in_progress",
		"phase:second_cycle_ready",
	}
	for _, label := range required {
		requireReplayCheckpoint(t, harness.checkpoints, label)
	}

	finalSnapshot := harness.state.snapshot()
	if finalSnapshot.Session.Phase != SessionPhaseSecondCycleReady {
		t.Fatalf("expected second_cycle_ready, got %s", finalSnapshot.Session.Phase)
	}
	if finalSnapshot.Session.RecoverReason != SessionRecoverReasonNone {
		t.Fatalf("expected recover reason to clear in second_cycle_ready, got %s", finalSnapshot.Session.RecoverReason)
	}
	if finalSnapshot.Session.DamagedBuildingCount != 0 {
		t.Fatalf("expected no damaged buildings in second_cycle_ready, got %d", finalSnapshot.Session.DamagedBuildingCount)
	}
}

func TestAuthorityMainlineReplayHarnessRestoreContinuity(t *testing.T) {
	harness := newMainlineReplayHarness(t)

	harness.runToSecondCycleReady(func(snapshot SessionSnapshot) bool {
		disciple := requireSingleDiscipleSnapshot(t, snapshot)
		return snapshot.Session.Phase == SessionPhaseRecover &&
			disciple.AssignmentKind == DiscipleAssignmentRepair &&
			disciple.WorkProgressTicks == 1
	})

	if !harness.restoreApplied {
		t.Fatalf("expected replay harness to apply a mid-session restore")
	}

	recoverSnapshot := requireReplayCheckpoint(t, harness.checkpoints, "recover:repair_in_progress")
	disciple := requireSingleDiscipleSnapshot(t, recoverSnapshot)
	if disciple.WorkProgressTicks != 1 {
		t.Fatalf("expected saved repair checkpoint progress 1, got %d", disciple.WorkProgressTicks)
	}

	finalSnapshot := harness.state.snapshot()
	if finalSnapshot.Session.Phase != SessionPhaseSecondCycleReady {
		t.Fatalf("expected second_cycle_ready after restored replay, got %s", finalSnapshot.Session.Phase)
	}
	if finalSnapshot.Session.DamagedBuildingCount != 0 {
		t.Fatalf("expected no damaged buildings after restored replay, got %d", finalSnapshot.Session.DamagedBuildingCount)
	}
}
