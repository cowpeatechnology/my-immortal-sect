package sect

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

func TestApplyEventResourceChangedMutatesUnifiedResourceState(t *testing.T) {
	state := NewInitialSectState(SectID("sect-preview"), UserID("player-preview"), "青崖宗")

	eventBody, err := json.Marshal(ResourceChangedPayload{
		Changes: map[ResourceKind]int64{
			ResourceKindSpiritStone: -20,
			ResourceKindOre:         +5,
		},
		Reason: "test_adjustment",
	})
	if err != nil {
		t.Fatalf("marshal resource event: %v", err)
	}

	if err := ApplyEvent(&state, DomainEvent{
		EventID: "evt-resource-1",
		SectID:  state.Meta.SectID,
		Version: 1,
		Type:    DomainEventTypeResourceChanged,
		Payload: eventBody,
	}); err != nil {
		t.Fatalf("apply resource changed event: %v", err)
	}

	if state.Resources.Stock[ResourceKindSpiritStone] != 100 {
		t.Fatalf("expected spirit stone 100 after delta, got %d", state.Resources.Stock[ResourceKindSpiritStone])
	}
	if state.Resources.Stock[ResourceKindOre] != 45 {
		t.Fatalf("expected ore 45 after delta, got %d", state.Resources.Stock[ResourceKindOre])
	}
}

func TestApplyResourceDeltaRejectsNegativeResourceState(t *testing.T) {
	state := NewInitialSectState(SectID("sect-preview"), UserID("player-preview"), "青崖宗")
	err := ApplyResourceDelta(&state.Resources, map[ResourceKind]int64{
		ResourceKindSpiritStone: -1000,
	})
	if err == nil {
		t.Fatalf("expected negative resource delta to fail")
	}
}

func TestBuildBuildingProducesResourceChangedAndBuildingBuilt(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-builder"), UserID("player-builder"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	payload, err := json.Marshal(BuildBuildingPayload{
		DefinitionKey: "main_hall",
		Origin:        TileCoord{Col: 4, Row: 6},
	})
	if err != nil {
		t.Fatalf("marshal build payload: %v", err)
	}

	response := receiver.executeCommand(SubmitCommand{
		SessionID: "session-builder",
		Command: ClientCommand{
			CmdID:       "cmd-build-1",
			UserID:      "player-builder",
			SectID:      "sect-builder",
			Type:        CommandTypeBuildBuilding,
			Payload:     payload,
			BaseVersion: 0,
		},
	})

	if response.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected accepted result, got %+v", response.Result)
	}
	if len(response.Result.Events) != 2 {
		t.Fatalf("expected resource and building events, got %+v", response.Result.Events)
	}
	if response.Result.Events[0].Type != ClientEventTypeResourceChanged {
		t.Fatalf("expected first event resource changed, got %+v", response.Result.Events)
	}
	if response.Result.Events[1].Type != ClientEventTypeBuildingChanged {
		t.Fatalf("expected second event building changed, got %+v", response.Result.Events)
	}
	if response.Result.SceneVersion != 2 {
		t.Fatalf("expected scene version 2 after two events, got %d", response.Result.SceneVersion)
	}
	if response.Snapshot.State.Resources.Stock[ResourceKindSpiritStone] != 90 {
		t.Fatalf("expected spirit stone 90 after building cost, got %d", response.Snapshot.State.Resources.Stock[ResourceKindSpiritStone])
	}
	if response.Snapshot.State.Resources.Stock[ResourceKindOre] != 30 {
		t.Fatalf("expected ore 30 after building cost, got %d", response.Snapshot.State.Resources.Stock[ResourceKindOre])
	}
	if _, ok := response.Snapshot.State.Buildings[BuildingID("building-1")]; !ok {
		t.Fatalf("expected building-1 to exist after build command")
	}
}

func TestBuildBuildingRejectsWhenResourcesAreInsufficient(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-builder"), UserID("player-builder"), "青崖宗")
	initial.Resources.Stock[ResourceKindSpiritStone] = 5
	initial.Resources.Stock[ResourceKindOre] = 1

	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	payload, err := json.Marshal(BuildBuildingPayload{
		DefinitionKey: "main_hall",
		Origin:        TileCoord{Col: 4, Row: 6},
	})
	if err != nil {
		t.Fatalf("marshal build payload: %v", err)
	}

	response := receiver.executeCommand(SubmitCommand{
		SessionID: "session-builder",
		Command: ClientCommand{
			CmdID:       "cmd-build-reject-1",
			UserID:      "player-builder",
			SectID:      "sect-builder",
			Type:        CommandTypeBuildBuilding,
			Payload:     payload,
			BaseVersion: 0,
		},
	})

	if response.Result.Status != CommandResultStatusRejected {
		t.Fatalf("expected rejected result, got %+v", response.Result)
	}
	if response.Result.Error == nil || response.Result.Error.Code != CommandErrorCodeInsufficientResource {
		t.Fatalf("expected insufficient-resource error, got %+v", response.Result.Error)
	}
	if response.Result.SceneVersion != 0 {
		t.Fatalf("expected version to stay at 0, got %d", response.Result.SceneVersion)
	}
	if len(response.Result.Events) != 0 {
		t.Fatalf("expected no emitted events on rejection, got %+v", response.Result.Events)
	}
	if len(response.Snapshot.State.Buildings) != 0 {
		t.Fatalf("expected no buildings to be created on rejection, got %+v", response.Snapshot.State.Buildings)
	}
	if response.Snapshot.State.Resources.Stock[ResourceKindSpiritStone] != 5 {
		t.Fatalf("expected spirit stone to remain unchanged on rejection, got %d", response.Snapshot.State.Resources.Stock[ResourceKindSpiritStone])
	}
}

func TestUpgradeBuildingSharesAuthorityEventAndSnapshotRules(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-builder"), UserID("player-builder"), "青崖宗")
	initial.Buildings[BuildingID("building-1")] = newBuildingState(BuildingID("building-1"), "main_hall", 1, TileCoord{Col: 4, Row: 6})

	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	payload, err := json.Marshal(UpgradeBuildingPayload{BuildingID: BuildingID("building-1")})
	if err != nil {
		t.Fatalf("marshal upgrade payload: %v", err)
	}

	response := receiver.executeCommand(SubmitCommand{
		SessionID: "session-builder",
		Command: ClientCommand{
			CmdID:       "cmd-upgrade-1",
			UserID:      "player-builder",
			SectID:      "sect-builder",
			Type:        CommandTypeUpgradeBuilding,
			Payload:     payload,
			BaseVersion: 0,
		},
	})

	if response.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected accepted result, got %+v", response.Result)
	}
	if len(response.Result.Events) != 2 || response.Result.Events[0].Type != ClientEventTypeResourceChanged || response.Result.Events[1].Type != ClientEventTypeBuildingChanged {
		t.Fatalf("expected resource+building client events, got %+v", response.Result.Events)
	}
	if response.Result.SceneVersion != 2 {
		t.Fatalf("expected scene version 2 after upgrade events, got %d", response.Result.SceneVersion)
	}
	upgraded := response.Snapshot.State.Buildings[BuildingID("building-1")]
	if upgraded.Level != 2 {
		t.Fatalf("expected building level 2 after upgrade, got %+v", upgraded)
	}
	if upgraded.MaxHP != 150 || upgraded.HP != 150 {
		t.Fatalf("expected level-based hp scaling, got %+v", upgraded)
	}
	if response.Snapshot.State.Resources.Stock[ResourceKindSpiritStone] != 75 {
		t.Fatalf("expected spirit stone 75 after upgrade cost, got %d", response.Snapshot.State.Resources.Stock[ResourceKindSpiritStone])
	}
	if response.Snapshot.State.Resources.Stock[ResourceKindOre] != 22 {
		t.Fatalf("expected ore 22 after upgrade cost, got %d", response.Snapshot.State.Resources.Stock[ResourceKindOre])
	}
}

func TestRestoreSnapshotReplayCanContinueWithUpgradeBuilding(t *testing.T) {
	base := NewInitialSectState(SectID("sect-builder"), UserID("player-builder"), "青崖宗")
	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	buildPayload, err := json.Marshal(BuildBuildingPayload{
		DefinitionKey: "main_hall",
		Origin:        TileCoord{Col: 4, Row: 6},
	})
	if err != nil {
		t.Fatalf("marshal build payload: %v", err)
	}
	buildResponse := receiver.executeCommand(SubmitCommand{
		SessionID: "session-builder",
		Command: ClientCommand{
			CmdID:       "cmd-build-1",
			UserID:      "player-builder",
			SectID:      "sect-builder",
			Type:        CommandTypeBuildBuilding,
			Payload:     buildPayload,
			BaseVersion: 0,
		},
	})
	savepoint := NewSnapshotReplaySavepoint(base)
	savepoint.AppendReplay(buildResponse.DomainEvents)
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore snapshot replay: %v", err)
	}
	if restored.Runtime.Version != 2 {
		t.Fatalf("expected restored version 2, got %d", restored.Runtime.Version)
	}

	restoredReceiver, ok := NewSectActor(restored).(*SectActor)
	if !ok {
		t.Fatalf("expected restored concrete sect actor receiver")
	}
	upgradePayload, err := json.Marshal(UpgradeBuildingPayload{BuildingID: BuildingID("building-1")})
	if err != nil {
		t.Fatalf("marshal upgrade payload: %v", err)
	}
	upgradeResponse := restoredReceiver.executeCommand(SubmitCommand{
		SessionID: "session-builder-restored",
		Command: ClientCommand{
			CmdID:       "cmd-upgrade-2",
			UserID:      "player-builder",
			SectID:      "sect-builder",
			Type:        CommandTypeUpgradeBuilding,
			Payload:     upgradePayload,
			BaseVersion: restored.Runtime.Version,
		},
	})
	if upgradeResponse.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected accepted upgrade after restore, got %+v", upgradeResponse.Result)
	}
	if upgradeResponse.Result.SceneVersion != 4 {
		t.Fatalf("expected version 4 after build replay + upgrade, got %d", upgradeResponse.Result.SceneVersion)
	}
	if upgradeResponse.Snapshot.State.Buildings[BuildingID("building-1")].Level != 2 {
		t.Fatalf("expected restored replay path to continue with level 2 building, got %+v", upgradeResponse.Snapshot.State.Buildings)
	}
}

func TestJoinSectBackfillsStarterDiscipleForRestoredEmptyState(t *testing.T) {
	legacy := NewInitialSectState(SectID("sect-legacy"), UserID("player-legacy"), "青崖宗")
	legacy.Disciples = map[DiscipleID]DiscipleState{}

	receiver, ok := NewSectActor(legacy).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	snapshot := receiver.snapshot(UserID("player-legacy"), "session-legacy")
	if len(snapshot.State.Disciples) != 1 {
		t.Fatalf("expected starter disciple backfill in snapshot, got %+v", snapshot.State.Disciples)
	}
	if _, ok := snapshot.State.Disciples[starterDiscipleID]; !ok {
		t.Fatalf("expected starter disciple %q in join snapshot, got %+v", starterDiscipleID, snapshot.State.Disciples)
	}
}

func TestRecruitmentCommandsGenerateRosterInsideSectState(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-admission"), UserID("player-admission"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	start := startRecruitmentCommand(t, receiver, StartRecruitmentPayload{
		CandidateCount:        2,
		InvestmentSpiritStone: 20,
		DurationDays:          5,
	}, 0)
	if start.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected recruitment start accepted, got %+v", start.Result)
	}
	if len(start.Result.Events) != 1 || start.Result.Events[0].Type != ClientEventTypeAdmissionChanged {
		t.Fatalf("expected admission client event for recruitment start, got %+v", start.Result.Events)
	}
	if len(start.Snapshot.State.Admissions.Candidates) != 2 {
		t.Fatalf("expected two authority-owned candidates, got %+v", start.Snapshot.State.Admissions)
	}
	if start.Snapshot.State.Admissions.CurrentRecruitment == nil {
		t.Fatalf("expected current recruitment session in snapshot, got %+v", start.Snapshot.State.Admissions)
	}
	if len(start.Snapshot.State.Disciples) != 1 {
		t.Fatalf("candidate generation must not create disciples before acceptance, got %+v", start.Snapshot.State.Disciples)
	}

	candidateIDs := sortedCandidateIDs(start.Snapshot.State.Admissions.Candidates)
	firstCandidate := start.Snapshot.State.Admissions.Candidates[candidateIDs[0]]
	secondCandidate := start.Snapshot.State.Admissions.Candidates[candidateIDs[1]]
	if firstCandidate.Identity != IdentityRankOuter || firstCandidate.Aptitude.SpiritRoot <= 0 {
		t.Fatalf("expected candidate identity and aptitude in authority snapshot, got %+v", firstCandidate)
	}
	if firstCandidate.Needs.DailySpiritGrain <= 0 || !firstCandidate.Support.FoodSatisfied {
		t.Fatalf("expected candidate basic needs/support in authority snapshot, got %+v", firstCandidate)
	}

	accept := acceptCandidateCommand(t, receiver, AcceptCandidatePayload{
		CandidateID: firstCandidate.CandidateID,
	}, start.Result.SceneVersion)
	if accept.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected candidate acceptance accepted, got %+v", accept.Result)
	}
	if !hasDomainEventType(accept.DomainEvents, DomainEventTypeCandidateAccepted) {
		t.Fatalf("expected candidate accepted domain event, got %+v", accept.DomainEvents)
	}
	if !hasPatchPath(accept.Result.Patch.Ops, "/admissions") {
		t.Fatalf("expected acceptance patch to update admissions, got %+v", accept.Result.Patch.Ops)
	}
	if len(accept.Snapshot.State.Disciples) != 2 {
		t.Fatalf("expected accepted candidate to enter disciple roster, got %+v", accept.Snapshot.State.Disciples)
	}
	if _, exists := accept.Snapshot.State.Admissions.Candidates[firstCandidate.CandidateID]; exists {
		t.Fatalf("accepted candidate must be removed from candidate pool, got %+v", accept.Snapshot.State.Admissions.Candidates)
	}

	newDiscipleID := DiscipleID("")
	for discipleID, disciple := range accept.Snapshot.State.Disciples {
		if disciple.Name == firstCandidate.Name {
			newDiscipleID = discipleID
			break
		}
	}
	if newDiscipleID == "" {
		t.Fatalf("expected accepted candidate name to appear in disciple roster, got %+v", accept.Snapshot.State.Disciples)
	}
	newDisciple := accept.Snapshot.State.Disciples[newDiscipleID]
	if newDisciple.Identity != firstCandidate.Identity || newDisciple.Aptitude != firstCandidate.Aptitude {
		t.Fatalf("expected accepted disciple identity and aptitude to come from candidate, got %+v from %+v", newDisciple, firstCandidate)
	}
	if newDisciple.AssignmentKind != DiscipleAssignmentIdle || newDisciple.WorkTarget.Description != "sect_support" {
		t.Fatalf("expected accepted disciple assignment to be authority-owned idle support, got %+v", newDisciple)
	}
	if newDisciple.Needs != firstCandidate.Needs || newDisciple.Support != firstCandidate.Support {
		t.Fatalf("expected accepted disciple needs/support to come from authority candidate, got %+v from %+v", newDisciple, firstCandidate)
	}
	account, ok := accept.Snapshot.State.Contribution.Accounts[newDiscipleID]
	if !ok || account.DiscipleID != newDiscipleID {
		t.Fatalf("expected contribution account for accepted disciple, got %+v", accept.Snapshot.State.Contribution.Accounts)
	}
	if !hasPatchPath(accept.Result.Patch.Ops, "/disciples/"+string(newDiscipleID)) {
		t.Fatalf("expected acceptance patch to include new disciple, got %+v", accept.Result.Patch.Ops)
	}
	if !hasPatchPath(accept.Result.Patch.Ops, "/contribution/accounts/"+string(newDiscipleID)) {
		t.Fatalf("expected acceptance patch to include contribution account, got %+v", accept.Result.Patch.Ops)
	}

	reject := rejectCandidateCommand(t, receiver, RejectCandidatePayload{
		CandidateID: secondCandidate.CandidateID,
	}, accept.Result.SceneVersion)
	if reject.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected candidate rejection accepted, got %+v", reject.Result)
	}
	if !hasDomainEventType(reject.DomainEvents, DomainEventTypeCandidateRejected) {
		t.Fatalf("expected candidate rejected domain event, got %+v", reject.DomainEvents)
	}
	if len(reject.Snapshot.State.Admissions.Candidates) != 0 || reject.Snapshot.State.Admissions.CurrentRecruitment != nil {
		t.Fatalf("expected rejection to close empty recruitment pool, got %+v", reject.Snapshot.State.Admissions)
	}
	if len(reject.Snapshot.State.Disciples) != 2 {
		t.Fatalf("rejected candidate must not enter disciple roster, got %+v", reject.Snapshot.State.Disciples)
	}
	for _, disciple := range reject.Snapshot.State.Disciples {
		if disciple.Name == secondCandidate.Name {
			t.Fatalf("rejected candidate appeared in disciple roster: %+v", reject.Snapshot.State.Disciples)
		}
	}

	savepoint := NewSnapshotReplaySavepoint(initial)
	savepoint.AppendReplay(start.DomainEvents)
	savepoint.AppendReplay(accept.DomainEvents)
	savepoint.AppendReplay(reject.DomainEvents)
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore recruitment savepoint: %v", err)
	}
	if restored.Runtime.Version != reject.Result.SceneVersion {
		t.Fatalf("expected restored recruitment version %d, got %d", reject.Result.SceneVersion, restored.Runtime.Version)
	}
	if len(restored.Disciples) != 2 || len(restored.Admissions.Candidates) != 0 {
		t.Fatalf("expected replay restore to preserve roster and empty candidate pool, got disciples=%+v admissions=%+v", restored.Disciples, restored.Admissions)
	}
	if _, ok := restored.Contribution.Accounts[newDiscipleID]; !ok {
		t.Fatalf("expected replay restore to preserve accepted disciple contribution account, got %+v", restored.Contribution.Accounts)
	}
}

func TestPublishAndAssignTaskStayInsideAuthorityState(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-taskhall"), UserID("player-taskhall"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	publish := publishTaskCommand(t, receiver, PublishTaskPayload{
		Kind:                 "sect_patrol",
		Title:                "巡山清障",
		RequiredProgressDays: 2,
		RewardResources: map[ResourceKind]int64{
			ResourceKindSpiritStone: 8,
			ResourceKindHerb:        2,
		},
	})
	if publish.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected publish task accepted, got %+v", publish.Result)
	}
	if len(publish.Result.Events) != 1 || publish.Result.Events[0].Type != ClientEventTypeTaskChanged {
		t.Fatalf("expected publish task to emit one task event, got %+v", publish.Result.Events)
	}

	task, ok := publish.Snapshot.State.Tasks[TaskID("task-1")]
	if !ok {
		t.Fatalf("expected task-1 to exist in authority snapshot, got %+v", publish.Snapshot.State.Tasks)
	}
	if task.Status != "published" || task.RequiredProgressDays != 2 {
		t.Fatalf("expected published task state, got %+v", task)
	}

	assign := assignTaskCommand(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("task-1"),
		DiscipleID: starterDiscipleID,
	}, publish.Result.SceneVersion)
	if assign.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected assign task accepted, got %+v", assign.Result)
	}
	if len(assign.Result.Events) != 2 || assign.Result.Events[0].Type != ClientEventTypeTaskChanged || assign.Result.Events[1].Type != ClientEventTypeDiscipleChanged {
		t.Fatalf("expected task+disciple authority events on assign, got %+v", assign.Result.Events)
	}

	assignedTask := assign.Snapshot.State.Tasks[TaskID("task-1")]
	if assignedTask.Status != "accepted" || len(assignedTask.AssignedDiscipleIDs) != 1 || assignedTask.AssignedDiscipleIDs[0] != starterDiscipleID {
		t.Fatalf("expected authority-owned accepted task, got %+v", assignedTask)
	}

	disciple := assign.Snapshot.State.Disciples[starterDiscipleID]
	if disciple.AssignmentKind != DiscipleAssignmentTask {
		t.Fatalf("expected disciple assignment to come from authority task state, got %+v", disciple)
	}
	if disciple.AssignmentTask == nil || *disciple.AssignmentTask != TaskID("task-1") {
		t.Fatalf("expected disciple assignment task to round-trip through snapshot, got %+v", disciple)
	}
	if disciple.WorkTarget.TaskID == nil || *disciple.WorkTarget.TaskID != TaskID("task-1") {
		t.Fatalf("expected disciple work target to point at task-1, got %+v", disciple.WorkTarget)
	}
}

func TestInitialTaskPoolExposesAuthorityOptions(t *testing.T) {
	state := NewInitialSectState(SectID("sect-taskpool"), UserID("player-taskpool"), "青崖宗")

	if len(state.Tasks) != 6 {
		t.Fatalf("expected authority snapshot to expose six authority task pool options, got %+v", state.Tasks)
	}
	taskTypes := map[TaskType]bool{}
	for taskID, task := range state.Tasks {
		if task.Status != TaskStatusPublished {
			t.Fatalf("expected task pool option %s to be published, got %+v", taskID, task)
		}
		if task.Type == "" || task.Grade == "" {
			t.Fatalf("expected task pool option %s to expose type and grade, got %+v", taskID, task)
		}
		taskTypes[task.Type] = true
		if task.Priority <= 0 || task.MaxAssignees <= 0 || task.ContributionReward <= 0 {
			t.Fatalf("expected task %s to expose priority/reward/dispatch shape, got %+v", taskID, task)
		}
		if task.Risk < 0 || task.Risk > 100 {
			t.Fatalf("expected task %s risk to stay bounded, got %+v", taskID, task)
		}
	}
	for _, taskType := range []TaskType{TaskTypeExternal, TaskTypeExplore, TaskTypeCombat} {
		if !taskTypes[taskType] {
			t.Fatalf("expected task pool to include %s tasks, got %+v", taskType, taskTypes)
		}
	}
}

func TestTaskPoolDispatchProgressAndFailureStayAuthorityOwned(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-taskpool-dispatch"), UserID("player-taskpool-dispatch"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	start := startRecruitmentCommandWithID(t, receiver, StartRecruitmentPayload{CandidateCount: 1}, receiver.state.Runtime.Version, "cmd-taskpool-recruit")
	candidateID := sortedCandidateIDs(start.Snapshot.State.Admissions.Candidates)[0]
	accept := acceptCandidateCommandWithID(t, receiver, AcceptCandidatePayload{CandidateID: candidateID}, start.Result.SceneVersion, "cmd-taskpool-accept")
	secondDiscipleID := discipleIDByName(t, accept.Snapshot.State, start.Snapshot.State.Admissions.Candidates[candidateID].Name)
	beforeSpiritStone := accept.Snapshot.State.Resources.Stock[ResourceKindSpiritStone]

	dispatch := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:      TaskID("pool-1"),
		DiscipleIDs: []DiscipleID{starterDiscipleID, secondDiscipleID},
	}, accept.Result.SceneVersion, "cmd-taskpool-dispatch")
	if dispatch.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected multi-disciple task dispatch accepted, got %+v", dispatch.Result)
	}
	if dispatch.Snapshot.State.Tasks[TaskID("pool-1")].Status != TaskStatusAccepted {
		t.Fatalf("expected pool-1 accepted after dispatch, got %+v", dispatch.Snapshot.State.Tasks[TaskID("pool-1")])
	}
	if len(dispatch.Snapshot.State.Tasks[TaskID("pool-1")].AssignedDiscipleIDs) != 2 {
		t.Fatalf("expected both disciples assigned by authority, got %+v", dispatch.Snapshot.State.Tasks[TaskID("pool-1")])
	}

	duplicateDispatch := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:      TaskID("pool-1"),
		DiscipleIDs: []DiscipleID{starterDiscipleID, secondDiscipleID},
	}, accept.Result.SceneVersion, "cmd-taskpool-dispatch")
	if duplicateDispatch.Result.Status != CommandResultStatusAccepted || receiver.state.Runtime.Version != dispatch.Result.SceneVersion {
		t.Fatalf("expected duplicate dispatch cmd_id to return cached result without side effects, got duplicate=%+v version=%d", duplicateDispatch.Result, receiver.state.Runtime.Version)
	}

	busy := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("pool-2"),
		DiscipleID: starterDiscipleID,
	}, receiver.state.Runtime.Version, "cmd-taskpool-busy")
	if busy.Result.Status != CommandResultStatusRejected || busy.Result.Error == nil || busy.Result.Error.Code != CommandErrorCodeDiscipleBusy {
		t.Fatalf("expected busy disciple dispatch rejection, got %+v", busy.Result)
	}

	advance := receiver.advanceTasksOneDay("session-taskpool-dispatch")
	if advance.Snapshot.State.Tasks[TaskID("pool-1")].Status != TaskStatusCompleted {
		t.Fatalf("expected authority day advance to complete pool-1, got %+v", advance.Snapshot.State.Tasks[TaskID("pool-1")])
	}
	for _, discipleID := range []DiscipleID{starterDiscipleID, secondDiscipleID} {
		disciple := advance.Snapshot.State.Disciples[discipleID]
		if disciple.AssignmentKind != DiscipleAssignmentIdle {
			t.Fatalf("expected completed task to release disciple %s, got %+v", discipleID, disciple)
		}
		if advance.Snapshot.State.Contribution.Accounts[discipleID].Balance != 8 {
			t.Fatalf("expected contribution reward for disciple %s, got %+v", discipleID, advance.Snapshot.State.Contribution.Accounts[discipleID])
		}
	}
	if got := advance.Snapshot.State.Resources.Stock[ResourceKindSpiritStone]; got != beforeSpiritStone+4 {
		t.Fatalf("expected task reward resource settlement from authority, got %d want %d", got, beforeSpiritStone+4)
	}

	failing, ok := NewSectActor(NewInitialSectState(SectID("sect-taskpool-fail"), UserID("player-taskpool-fail"), "青崖宗")).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete failing sect actor receiver")
	}
	failDispatch := assignTaskCommandWithID(t, failing, AssignDiscipleTaskPayload{
		TaskID:     TaskID("pool-4"),
		DiscipleID: starterDiscipleID,
	}, failing.state.Runtime.Version, "cmd-taskpool-fail-dispatch")
	if failDispatch.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected high-risk task dispatch accepted before authority settlement, got %+v", failDispatch.Result)
	}
	failAdvance := failing.advanceTasksOneDay("session-taskpool-fail")
	if failAdvance.Snapshot.State.Tasks[TaskID("pool-4")].Status != TaskStatusFailed {
		t.Fatalf("expected high-risk task to fail by authority calculation, got %+v", failAdvance.Snapshot.State.Tasks[TaskID("pool-4")])
	}
	if failAdvance.Snapshot.State.Contribution.Accounts[starterDiscipleID].Balance != 0 {
		t.Fatalf("failed task must not grant contribution, got %+v", failAdvance.Snapshot.State.Contribution.Accounts[starterDiscipleID])
	}
}

func TestTaskPoolIntentValidationHandlesRequirementsResourcesCancelAndPriority(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-taskpool-validation"), UserID("player-taskpool-validation"), "青崖宗")
	initial.Resources.Stock[ResourceKindSpiritStone] = 0
	resourceReceiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete resource sect actor receiver")
	}
	resourceRejected := assignTaskCommandWithID(t, resourceReceiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("pool-3"),
		DiscipleID: starterDiscipleID,
	}, resourceReceiver.state.Runtime.Version, "cmd-taskpool-resource-reject")
	if resourceRejected.Result.Status != CommandResultStatusRejected || resourceRejected.Result.Error == nil || resourceRejected.Result.Error.Code != CommandErrorCodeInsufficientResource {
		t.Fatalf("expected dispatch cost resource rejection, got %+v", resourceRejected.Result)
	}

	requirementReceiver, ok := NewSectActor(NewInitialSectState(SectID("sect-taskpool-requirement"), UserID("player-taskpool-requirement"), "青崖宗")).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete requirement sect actor receiver")
	}
	requirementRejected := assignTaskCommandWithID(t, requirementReceiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("pool-5"),
		DiscipleID: starterDiscipleID,
	}, requirementReceiver.state.Runtime.Version, "cmd-taskpool-requirement-reject")
	if requirementRejected.Result.Status != CommandResultStatusRejected || requirementRejected.Result.Error == nil || requirementRejected.Result.Error.Code != CommandErrorCodeTaskRequirementNotMet {
		t.Fatalf("expected aptitude mismatch rejection, got %+v", requirementRejected.Result)
	}

	intentReceiver, ok := NewSectActor(NewInitialSectState(SectID("sect-taskpool-intent"), UserID("player-taskpool-intent"), "青崖宗")).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete intent sect actor receiver")
	}
	priority := setTaskPriorityCommand(t, intentReceiver, SetTaskPriorityPayload{TaskID: TaskID("pool-2"), Priority: 95}, intentReceiver.state.Runtime.Version)
	if priority.Result.Status != CommandResultStatusAccepted || priority.Snapshot.State.Tasks[TaskID("pool-2")].Priority != 95 {
		t.Fatalf("expected priority intent to update authority task state, got %+v", priority.Result)
	}
	cancel := cancelTaskCommand(t, intentReceiver, CancelTaskPayload{TaskID: TaskID("pool-2")}, priority.Result.SceneVersion)
	if cancel.Result.Status != CommandResultStatusAccepted || cancel.Snapshot.State.Tasks[TaskID("pool-2")].Status != TaskStatusCancelled {
		t.Fatalf("expected cancel intent to close task by authority event, got %+v", cancel.Result)
	}
}

func TestAdvanceTasksOneDaySettlesTaskWithoutClientCompletionCommand(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-taskhall"), UserID("player-taskhall"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	publish := publishTaskCommand(t, receiver, PublishTaskPayload{
		Kind:                 "herb_delivery",
		Title:                "药圃采买",
		RequiredProgressDays: 1,
		ContributionReward:   9,
		RewardResources: map[ResourceKind]int64{
			ResourceKindSpiritStone: 12,
			ResourceKindHerb:        3,
		},
	})
	assign := assignTaskCommand(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("task-1"),
		DiscipleID: starterDiscipleID,
	}, publish.Result.SceneVersion)

	beforeRewardStone := assign.Snapshot.State.Resources.Stock[ResourceKindSpiritStone]
	beforeRewardHerb := assign.Snapshot.State.Resources.Stock[ResourceKindHerb]
	beforeContribution := assign.Snapshot.State.Contribution.Accounts[starterDiscipleID].Balance

	advance := receiver.advanceTasksOneDay("session-taskhall")
	if advance.FromVersion != assign.Result.SceneVersion {
		t.Fatalf("expected authority day-advance from version %d, got %+v", assign.Result.SceneVersion, advance)
	}
	if advance.ToVersion <= assign.Result.SceneVersion {
		t.Fatalf("expected authority day-advance to move runtime version forward, got %+v", advance)
	}
	if !hasDomainEventType(advance.DomainEvents, DomainEventTypeTimeAdvanced) ||
		!hasDomainEventType(advance.DomainEvents, DomainEventTypeTaskCompleted) ||
		!hasDomainEventType(advance.DomainEvents, DomainEventTypeContributionEarned) ||
		!hasDomainEventType(advance.DomainEvents, DomainEventTypeSectGoalResolved) {
		t.Fatalf("expected authority day-advance to settle time/task/contribution/goal events, got %+v", advance.DomainEvents)
	}

	task := advance.Snapshot.State.Tasks[TaskID("task-1")]
	if task.Status != "completed" || task.CompletedProgressDays != 1 {
		t.Fatalf("expected task completion to be settled by authority, got %+v", task)
	}

	disciple := advance.Snapshot.State.Disciples[starterDiscipleID]
	if disciple.AssignmentKind != DiscipleAssignmentIdle || disciple.AssignmentTask != nil {
		t.Fatalf("expected authority completion to release disciple, got %+v", disciple)
	}
	if disciple.WorkTarget.TaskID != nil || disciple.WorkTarget.Description != "sect_support" {
		t.Fatalf("expected authority completion to reset work target, got %+v", disciple.WorkTarget)
	}

	if got := advance.Snapshot.State.Resources.Stock[ResourceKindSpiritStone]; got != beforeRewardStone+12 {
		t.Fatalf("expected spirit stone reward settlement from authority, got %d", got)
	}
	if got := advance.Snapshot.State.Resources.Stock[ResourceKindHerb]; got != beforeRewardHerb+3 {
		t.Fatalf("expected herb reward settlement from authority, got %d", got)
	}
	account := advance.Snapshot.State.Contribution.Accounts[starterDiscipleID]
	if account.Balance != beforeContribution+9 || account.EarnedTotal != 9 {
		t.Fatalf("expected contribution ledger reward settlement from authority, got %+v", account)
	}
	if advance.Snapshot.State.Contribution.OutstandingContribution != account.Balance {
		t.Fatalf("expected contribution metrics to track outstanding balance, got %+v", advance.Snapshot.State.Contribution)
	}
}

func TestTaskReplayRestoreCanContinueAuthorityProgression(t *testing.T) {
	base := NewInitialSectState(SectID("sect-taskhall"), UserID("player-taskhall"), "青崖宗")
	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	publish := publishTaskCommand(t, receiver, PublishTaskPayload{
		Kind:                 "library_copying",
		Title:                "抄录典籍",
		RequiredProgressDays: 2,
		RewardResources: map[ResourceKind]int64{
			ResourceKindSpiritStone: 6,
		},
	})
	assign := assignTaskCommand(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("task-1"),
		DiscipleID: starterDiscipleID,
	}, publish.Result.SceneVersion)
	dayOne := receiver.advanceTasksOneDay("session-taskhall")

	savepoint := NewSnapshotReplaySavepoint(base)
	savepoint.AppendReplay(publish.DomainEvents)
	savepoint.AppendReplay(assign.DomainEvents)
	savepoint.AppendReplay(dayOne.DomainEvents)

	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore task savepoint: %v", err)
	}
	restoredTask := restored.Tasks[TaskID("task-1")]
	if restoredTask.Status != "accepted" || restoredTask.CompletedProgressDays != 1 {
		t.Fatalf("expected replay restore to preserve in-progress task state, got %+v", restoredTask)
	}
	restoredDisciple := restored.Disciples[starterDiscipleID]
	if restoredDisciple.AssignmentKind != DiscipleAssignmentTask || restoredDisciple.AssignmentTask == nil {
		t.Fatalf("expected replay restore to preserve assigned disciple, got %+v", restoredDisciple)
	}

	restoredReceiver, ok := NewSectActor(restored).(*SectActor)
	if !ok {
		t.Fatalf("expected restored concrete sect actor receiver")
	}
	dayTwo := restoredReceiver.advanceTasksOneDay("session-taskhall-restored")
	if !hasDomainEventType(dayTwo.DomainEvents, DomainEventTypeTimeAdvanced) ||
		!hasDomainEventType(dayTwo.DomainEvents, DomainEventTypeTaskCompleted) ||
		!hasDomainEventType(dayTwo.DomainEvents, DomainEventTypeDiscipleAssignmentChanged) ||
		!hasDomainEventType(dayTwo.DomainEvents, DomainEventTypeResourceChanged) ||
		!hasDomainEventType(dayTwo.DomainEvents, DomainEventTypeProductionChanged) {
		t.Fatalf("expected restored authority day-advance to settle time, task, disciple release, and production progress, got %+v", dayTwo.DomainEvents)
	}
	if dayTwo.Snapshot.State.Tasks[TaskID("task-1")].Status != "completed" {
		t.Fatalf("expected restored authority progression to complete task, got %+v", dayTwo.Snapshot.State.Tasks)
	}
	if dayTwo.Snapshot.State.Disciples[starterDiscipleID].AssignmentKind != DiscipleAssignmentIdle {
		t.Fatalf("expected restored authority progression to release disciple, got %+v", dayTwo.Snapshot.State.Disciples[starterDiscipleID])
	}
}

func TestExchangeContributionItemUsesAuthorityLedgerAndSectStock(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-contribution"), UserID("player-contribution"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	publish := publishTaskCommand(t, receiver, PublishTaskPayload{
		Kind:                 "grain_allocation",
		Title:                "仓廪盘点",
		RequiredProgressDays: 1,
		ContributionReward:   15,
	})
	assignTaskCommand(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("task-1"),
		DiscipleID: starterDiscipleID,
	}, publish.Result.SceneVersion)
	advance := receiver.advanceTasksOneDay("session-contribution")

	beforeGrain := advance.Snapshot.State.Resources.Stock[ResourceKindSpiritGrain]
	beforeContribution := advance.Snapshot.State.Contribution.Accounts[starterDiscipleID]

	exchange := exchangeContributionCommand(t, receiver, ExchangeContributionItemPayload{
		DiscipleID:     starterDiscipleID,
		ExchangeItemID: ExchangeItemID("treasury-spirit-grain"),
		Quantity:       2,
	}, advance.ToVersion)
	if exchange.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected authority exchange accepted, got %+v", exchange.Result)
	}
	if len(exchange.Result.Events) != 2 || exchange.Result.Events[0].Type != ClientEventTypeContributionChanged || exchange.Result.Events[1].Type != ClientEventTypeResourceChanged {
		t.Fatalf("expected contribution+resource events for exchange, got %+v", exchange.Result.Events)
	}

	account := exchange.Snapshot.State.Contribution.Accounts[starterDiscipleID]
	if account.Balance != beforeContribution.Balance-10 || account.SpentTotal != 10 {
		t.Fatalf("expected authority exchange to spend contribution ledger, got %+v", account)
	}
	if got := exchange.Snapshot.State.Resources.Stock[ResourceKindSpiritGrain]; got != beforeGrain-2 {
		t.Fatalf("expected authority exchange to consume sect stock, got %d", got)
	}
	if exchange.Snapshot.State.Contribution.MonthlyPurchases[starterDiscipleID][ExchangeItemID("treasury-spirit-grain")] != 2 {
		t.Fatalf("expected authority exchange ledger to record monthly purchase, got %+v", exchange.Snapshot.State.Contribution.MonthlyPurchases)
	}
}

func TestMonthlySettlementDelaysStipendChecksDutyAndPenalizesLowRedeemability(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-monthly-pressure"), UserID("player-monthly"), "青崖宗")
	initial.Productions = map[ProductionID]ProductionJob{}
	initial.Resources.Stock = map[ResourceKind]int64{
		ResourceKindSpiritStone:  0,
		ResourceKindSpiritGrain:  0,
		ResourceKindHerb:         0,
		ResourceKindOre:          0,
		ResourceKindBeastMat:     0,
		ResourceKindFormationMat: 0,
	}
	account := initial.Contribution.Accounts[starterDiscipleID]
	account.Balance = 100
	account.EarnedTotal = 100
	initial.Contribution.Accounts[starterDiscipleID] = account
	recalculateContributionMetrics(&initial)

	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	dayThirty := advanceSectDays(receiver, 30, "session-monthly-pressure")
	if !hasDomainEventType(dayThirty.DomainEvents, DomainEventTypeTimeAdvanced) ||
		!hasDomainEventType(dayThirty.DomainEvents, DomainEventTypePayrollDelayed) ||
		!hasDomainEventType(dayThirty.DomainEvents, DomainEventTypeMonthlyObligationChecked) ||
		!hasDomainEventType(dayThirty.DomainEvents, DomainEventTypeDiscipleSatisfactionChanged) ||
		!hasDomainEventType(dayThirty.DomainEvents, DomainEventTypeDiscipleLoyaltyChanged) ||
		!hasDomainEventType(dayThirty.DomainEvents, DomainEventTypeMonthAdvanced) {
		t.Fatalf("expected monthly pressure events in event log, got %+v", dayThirty.DomainEvents)
	}

	state := dayThirty.Snapshot.State
	summary := state.Monthly.LastSettlement
	if summary.MonthIndex != 1 || !summary.ResourceShortage || !summary.ContributionShortage || summary.DutyViolations != 1 {
		t.Fatalf("expected shortage, low redeemability, and duty violation summary, got %+v", summary)
	}
	if got := state.Monthly.Payroll.Arrears[starterDiscipleID]; got != 1 {
		t.Fatalf("expected one month stipend arrears, got %+v", state.Monthly.Payroll.Arrears)
	}
	disciple := state.Disciples[starterDiscipleID]
	if disciple.Satisfaction >= 70 || disciple.Loyalty >= 70 {
		t.Fatalf("expected satisfaction and loyalty to fall under monthly pressure, got %+v", disciple)
	}
}

func TestMonthlySettlementPaysStipendResetsPurchasesAndRewardsFulfilledDuty(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-monthly-paid"), UserID("player-monthly"), "青崖宗")
	initial.Productions = map[ProductionID]ProductionJob{}
	initial.Monthly.Obligations.CompletedDays[starterDiscipleID] = monthlyDutyRequiredDays(initial.Disciples[starterDiscipleID])
	initial.Contribution.MonthlyPurchases[starterDiscipleID] = map[ExchangeItemID]int64{
		ExchangeItemID("treasury-spirit-grain"): 2,
	}

	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}
	beforeSpiritStone := receiver.state.Resources.Stock[ResourceKindSpiritStone]

	dayThirty := advanceSectDays(receiver, 30, "session-monthly-paid")
	if !hasDomainEventType(dayThirty.DomainEvents, DomainEventTypePayrollPaid) || !hasDomainEventType(dayThirty.DomainEvents, DomainEventTypeMonthAdvanced) {
		t.Fatalf("expected paid monthly settlement events, got %+v", dayThirty.DomainEvents)
	}

	state := dayThirty.Snapshot.State
	summary := state.Monthly.LastSettlement
	stipend := monthlyStipendFor(state.Disciples[starterDiscipleID])
	if summary.StipendPaid != stipend || summary.StipendDelayed != 0 || summary.DutyViolations != 0 || summary.ResourceShortage {
		t.Fatalf("expected clean monthly settlement, got %+v", summary)
	}
	stableGoal := state.Goals.ByID[SectGoalID("goal-stable-monthly")]
	if stableGoal.Status != SectGoalStatusCompleted {
		t.Fatalf("expected stable monthly goal to complete on clean settlement, got %+v", stableGoal)
	}
	expectedStone := beforeSpiritStone - stipend + stableGoal.RewardResources[ResourceKindSpiritStone]
	if got := state.Resources.Stock[ResourceKindSpiritStone]; got != expectedStone {
		t.Fatalf("expected monthly stipend and stable-goal reward to net %d spirit stones, got %d", expectedStone, got)
	}
	if state.Monthly.Payroll.LastPaidMonth != 1 || state.Monthly.LastSettledMonth != 1 {
		t.Fatalf("expected month one settlement markers, got %+v", state.Monthly)
	}
	if len(state.Contribution.MonthlyPurchases) != 0 {
		t.Fatalf("expected monthly contribution purchases to reset, got %+v", state.Contribution.MonthlyPurchases)
	}
	if state.Monthly.Obligations.MonthIndex != 2 || state.Monthly.Obligations.CompletedDays[starterDiscipleID] != 0 {
		t.Fatalf("expected next month obligation progress to reset, got %+v", state.Monthly.Obligations)
	}
	disciple := state.Disciples[starterDiscipleID]
	if disciple.Satisfaction <= 70 || disciple.Loyalty <= 70 {
		t.Fatalf("expected fulfilled duty and paid stipend to improve mood, got %+v", disciple)
	}
}

func TestMonthlySettlementRestoreReplayMatchesContinuousAuthorityPath(t *testing.T) {
	base := NewInitialSectState(SectID("sect-monthly-restore"), UserID("player-monthly"), "青崖宗")
	base.Productions = map[ProductionID]ProductionJob{}

	continuous, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete continuous sect actor receiver")
	}
	continuousFinal := advanceSectDays(continuous, 30, "session-monthly-restore")

	restoreSeed, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete restore seed sect actor receiver")
	}
	savepoint := NewSnapshotReplaySavepoint(base)
	for day := 0; day < 15; day++ {
		advance := restoreSeed.advanceTasksOneDay("session-monthly-restore")
		savepoint.AppendReplay(advance.DomainEvents)
	}

	restoredState, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore monthly savepoint: %v", err)
	}
	restoredReceiver, ok := NewSectActor(restoredState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete restored sect actor receiver")
	}
	restoredFinal := advanceSectDays(restoredReceiver, 15, "session-monthly-restore")

	continuousJSON, err := json.Marshal(continuousFinal.Snapshot.State)
	if err != nil {
		t.Fatalf("marshal continuous monthly state: %v", err)
	}
	restoredJSON, err := json.Marshal(restoredFinal.Snapshot.State)
	if err != nil {
		t.Fatalf("marshal restored monthly state: %v", err)
	}
	if string(continuousJSON) != string(restoredJSON) {
		t.Fatalf("expected restored monthly path to match continuous authority state\ncontinuous: %s\nrestored: %s", continuousJSON, restoredJSON)
	}
	if got := len(savepoint.EventLogAfter(0)); got == 0 {
		t.Fatalf("expected monthly restore evidence to use replay event log")
	}
}

func TestAdvanceTasksOneDaySettlesProductionAndRefinementInsideAuthorityState(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-production"), UserID("player-production"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	var dayFive AdvanceTasksOneDayResponse
	for day := 0; day < 5; day++ {
		dayFive = receiver.advanceTasksOneDay("session-production")
	}
	if !hasClientEventType(dayFive.Events, ClientEventTypeProductionChanged) {
		t.Fatalf("expected production change events on day five, got %+v", dayFive.Events)
	}
	if !hasClientEventType(dayFive.Events, ClientEventTypeResourceChanged) {
		t.Fatalf("expected refinement resource delta on day five, got %+v", dayFive.Events)
	}

	refineDayFive := dayFive.Snapshot.State.Productions[ProductionID("prod-4-formation-refine")]
	if refineDayFive.CompletedCycles != 1 || refineDayFive.ProgressDays != 0 {
		t.Fatalf("expected first refinement cycle to close on day five, got %+v", refineDayFive)
	}
	if got := dayFive.Snapshot.State.Resources.Stock[ResourceKindSpiritStone]; got != 116 {
		t.Fatalf("expected spirit stone 116 after first refinement cycle, got %d", got)
	}
	if got := dayFive.Snapshot.State.Resources.Stock[ResourceKindHerb]; got != 18 {
		t.Fatalf("expected herb 18 after first refinement cycle, got %d", got)
	}
	if got := dayFive.Snapshot.State.Resources.Stock[ResourceKindOre]; got != 34 {
		t.Fatalf("expected ore 34 after first refinement cycle, got %d", got)
	}
	if got := dayFive.Snapshot.State.Resources.Stock[ResourceKindFormationMat]; got != 11 {
		t.Fatalf("expected formation mat 11 after first refinement cycle, got %d", got)
	}

	var dayTen AdvanceTasksOneDayResponse
	for day := 0; day < 5; day++ {
		dayTen = receiver.advanceTasksOneDay("session-production")
	}
	if !hasClientEventType(dayTen.Events, ClientEventTypeProductionChanged) {
		t.Fatalf("expected production change events on day ten, got %+v", dayTen.Events)
	}
	if !hasClientEventType(dayTen.Events, ClientEventTypeResourceChanged) {
		t.Fatalf("expected resource change events on day ten, got %+v", dayTen.Events)
	}

	final := dayTen.Snapshot.State
	if got := final.Resources.Stock[ResourceKindSpiritStone]; got != 112 {
		t.Fatalf("expected spirit stone 112 after ten-day authority production, got %d", got)
	}
	if got := final.Resources.Stock[ResourceKindSpiritGrain]; got != 120 {
		t.Fatalf("expected spirit grain 120 after farm output, got %d", got)
	}
	if got := final.Resources.Stock[ResourceKindHerb]; got != 24 {
		t.Fatalf("expected herb 24 after herb + refinement loops, got %d", got)
	}
	if got := final.Resources.Stock[ResourceKindOre]; got != 40 {
		t.Fatalf("expected ore 40 after mining + refinement loops, got %d", got)
	}
	if got := final.Resources.Stock[ResourceKindFormationMat]; got != 14 {
		t.Fatalf("expected formation mat 14 after two refinement cycles, got %d", got)
	}
	if final.Productions[ProductionID("prod-1-farm-grain")].CompletedCycles != 1 {
		t.Fatalf("expected grain farm to complete one cycle, got %+v", final.Productions[ProductionID("prod-1-farm-grain")])
	}
	if final.Productions[ProductionID("prod-2-herb-garden")].CompletedCycles != 1 {
		t.Fatalf("expected herb garden to complete one cycle, got %+v", final.Productions[ProductionID("prod-2-herb-garden")])
	}
	if final.Productions[ProductionID("prod-3-ore-mine")].CompletedCycles != 1 {
		t.Fatalf("expected ore mine to complete one cycle, got %+v", final.Productions[ProductionID("prod-3-ore-mine")])
	}
	if final.Productions[ProductionID("prod-4-formation-refine")].CompletedCycles != 2 {
		t.Fatalf("expected refinement loop to complete two cycles, got %+v", final.Productions[ProductionID("prod-4-formation-refine")])
	}
}

func TestProductionReplayRestoreContinuesAuthorityCycle(t *testing.T) {
	base := NewInitialSectState(SectID("sect-production"), UserID("player-production"), "青崖宗")
	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	savepoint := NewSnapshotReplaySavepoint(base)
	for day := 0; day < 4; day++ {
		advance := receiver.advanceTasksOneDay("session-production")
		savepoint.AppendReplay(advance.DomainEvents)
	}

	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore production savepoint: %v", err)
	}
	if restored.Productions[ProductionID("prod-4-formation-refine")].ProgressDays != 4 {
		t.Fatalf("expected restored refinement progress day 4, got %+v", restored.Productions[ProductionID("prod-4-formation-refine")])
	}

	restoredReceiver, ok := NewSectActor(restored).(*SectActor)
	if !ok {
		t.Fatalf("expected restored concrete sect actor receiver")
	}
	dayFive := restoredReceiver.advanceTasksOneDay("session-production-restored")
	if !hasClientEventType(dayFive.Events, ClientEventTypeResourceChanged) {
		t.Fatalf("expected restored replay to emit refinement resource delta, got %+v", dayFive.Events)
	}

	refine := dayFive.Snapshot.State.Productions[ProductionID("prod-4-formation-refine")]
	if refine.CompletedCycles != 1 || refine.ProgressDays != 0 {
		t.Fatalf("expected restored replay to preserve and settle refinement cycle, got %+v", refine)
	}
	if got := dayFive.Snapshot.State.Resources.Stock[ResourceKindFormationMat]; got != 11 {
		t.Fatalf("expected restored replay to produce formation mat 11, got %d", got)
	}
	if got := dayFive.Snapshot.State.Resources.Stock[ResourceKindOre]; got != 34 {
		t.Fatalf("expected restored replay to preserve ore settlement, got %d", got)
	}
}

func TestProductionQueueCommandsAndBottleneckStayAuthorityOwned(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-production-queue"), UserID("player-production-queue"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	start := startProductionCommandWithID(t, receiver, StartProductionPayload{
		RecipeID:     RecipeID("formation_refine_mvp"),
		Priority:     72,
		TargetCycles: 1,
	}, receiver.state.Runtime.Version, "cmd-production-start-queue")
	if start.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected start production accepted, got %+v", start.Result)
	}
	if !hasClientEventType(start.Result.Events, ClientEventTypeProductionChanged) {
		t.Fatalf("expected production changed event for start, got %+v", start.Result.Events)
	}
	started := start.Snapshot.State.Productions[ProductionID("prod-player-1")]
	if started.RecipeID != RecipeID("formation_refine_mvp") || started.Priority != 72 || started.TargetCycles != 1 {
		t.Fatalf("expected authority-created production job from recipe, got %+v", started)
	}

	adjust := adjustProductionCommand(t, receiver, AdjustProductionPayload{
		ProductionID: started.ProductionID,
		Priority:     90,
		TargetCycles: 2,
	}, receiver.state.Runtime.Version)
	adjusted := adjust.Snapshot.State.Productions[started.ProductionID]
	if adjusted.Priority != 90 || adjusted.TargetCycles != 2 {
		t.Fatalf("expected authority-adjusted production priority and target, got %+v", adjusted)
	}

	cancel := cancelProductionCommand(t, receiver, CancelProductionPayload{ProductionID: started.ProductionID}, receiver.state.Runtime.Version)
	cancelled := cancel.Snapshot.State.Productions[started.ProductionID]
	if cancelled.Status != ProductionStatusCancelled {
		t.Fatalf("expected authority-cancelled production job, got %+v", cancelled)
	}
	for day := 0; day < 3; day++ {
		receiver.advanceTasksOneDay("session-production-queue")
	}
	if progressed := receiver.state.Productions[started.ProductionID]; progressed.ProgressDays != 0 || progressed.Status != ProductionStatusCancelled {
		t.Fatalf("expected cancelled production not to advance locally or by tick, got %+v", progressed)
	}

	bottleneckState := NewInitialSectState(SectID("sect-production-bottleneck"), UserID("player-production-bottleneck"), "青崖宗")
	bottleneckState.Resources.Stock[ResourceKindOre] = 0
	bottleneckReceiver, ok := NewSectActor(bottleneckState).(*SectActor)
	if !ok {
		t.Fatalf("expected bottleneck concrete sect actor receiver")
	}
	bottleneckStart := startProductionCommandWithID(t, bottleneckReceiver, StartProductionPayload{
		RecipeID:     RecipeID("formation_refine_mvp"),
		Priority:     55,
		TargetCycles: 1,
	}, bottleneckReceiver.state.Runtime.Version, "cmd-production-start-bottleneck")
	bottleneckID := bottleneckStart.Snapshot.State.Productions[ProductionID("prod-player-1")].ProductionID

	var dayFive AdvanceTasksOneDayResponse
	for day := 0; day < 5; day++ {
		dayFive = bottleneckReceiver.advanceTasksOneDay("session-production-bottleneck")
	}
	blocked := dayFive.Snapshot.State.Productions[bottleneckID]
	if blocked.Status != ProductionStatusBlocked || blocked.BlockedReason != "input_shortage" || blocked.Shortage[ResourceKindOre] <= 0 {
		t.Fatalf("expected authority bottleneck shortage in snapshot, got %+v", blocked)
	}
	if got := dayFive.Snapshot.State.Resources.Stock[ResourceKindOre]; got != 0 {
		t.Fatalf("expected bottleneck to avoid negative ore, got %d", got)
	}
}

func TestProductionCompletionUsesResourceChangedAndApplyEventPath(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-production-completion"), UserID("player-production-completion"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	start := startProductionCommandWithID(t, receiver, StartProductionPayload{
		RecipeID:     RecipeID("formation_refine_mvp"),
		Priority:     65,
		TargetCycles: 1,
	}, receiver.state.Runtime.Version, "cmd-production-start-completion")
	productionID := start.Snapshot.State.Productions[ProductionID("prod-player-1")].ProductionID

	var dayFive AdvanceTasksOneDayResponse
	for day := 0; day < 5; day++ {
		dayFive = receiver.advanceTasksOneDay("session-production-completion")
	}
	if !hasDomainEventType(dayFive.DomainEvents, DomainEventTypeResourceChanged) {
		t.Fatalf("expected production completion to write ResourceChanged domain event, got %+v", dayFive.DomainEvents)
	}
	if !hasClientEventType(dayFive.Events, ClientEventTypeResourceChanged) {
		t.Fatalf("expected production completion resource event for thin client, got %+v", dayFive.Events)
	}
	completed := dayFive.Snapshot.State.Productions[productionID]
	if completed.Status != ProductionStatusCompleted || completed.CompletedCycles != 1 || completed.ProgressDays != 0 {
		t.Fatalf("expected target-cycle production to close through ApplyEvent, got %+v", completed)
	}
	if got := dayFive.Snapshot.State.Resources.Stock[ResourceKindFormationMat]; got != 14 {
		t.Fatalf("expected default + queued refinement outputs through authority resources, got %d", got)
	}
	if got := dayFive.Snapshot.State.Resources.Stock[ResourceKindOre]; got != 28 {
		t.Fatalf("expected input costs applied through ResourceChanged, got ore %d", got)
	}
}

func TestStartCultivationAndDailyAdvanceStayAuthorityOwned(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-cultivation"), UserID("player-cultivation"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	start := startCultivationCommand(t, receiver, StartCultivationPayload{DiscipleID: starterDiscipleID}, receiver.state.Runtime.Version)
	if start.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected start cultivation accepted, got %+v", start.Result)
	}
	startedDisciple := start.Snapshot.State.Disciples[starterDiscipleID]
	if startedDisciple.AssignmentKind != DiscipleAssignmentCultivation {
		t.Fatalf("expected authority to own cultivation assignment, got %+v", startedDisciple)
	}

	dayOne := receiver.advanceTasksOneDay("session-cultivation")
	if !hasClientEventType(dayOne.Events, ClientEventTypeDiscipleChanged) {
		t.Fatalf("expected cultivation day to emit disciple change, got %+v", dayOne.Events)
	}
	if !hasClientEventType(dayOne.Events, ClientEventTypeResourceChanged) {
		t.Fatalf("expected cultivation day to consume spirit stone, got %+v", dayOne.Events)
	}

	disciple := dayOne.Snapshot.State.Disciples[starterDiscipleID]
	if disciple.Realm.CultivationPoints != 15 {
		t.Fatalf("expected daily cultivation points 15, got %+v", disciple.Realm)
	}
	if disciple.Realm.ReadyForBreakthrough {
		t.Fatalf("expected disciple to need another day before breakthrough, got %+v", disciple.Realm)
	}
	if dayOne.Snapshot.State.Resources.Stock[ResourceKindSpiritStone] != 119 {
		t.Fatalf("expected daily cultivation to consume one spirit stone, got %d", dayOne.Snapshot.State.Resources.Stock[ResourceKindSpiritStone])
	}
}

func TestCultivationDecisionChainPillAndCaveStayAuthorityOwned(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-cultivation-decision"), UserID("player-cultivation-decision"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	joinSnapshot := receiver.snapshot(UserID("player-cultivation-decision"), "session-cultivation-decision")
	initialDecision := joinSnapshot.State.Disciples[starterDiscipleID].Cultivation
	if initialDecision.DailyGain != 15 || initialDecision.RequiredPoints != 30 || initialDecision.BreakthroughSuccessRate != 100 {
		t.Fatalf("expected authority cultivation decision fields on snapshot, got %+v", initialDecision)
	}
	if initialDecision.CultivationPillAvailable != 2 || initialDecision.BreakthroughSpiritStoneCost != 6 {
		t.Fatalf("expected authority inventory and breakthrough cost visibility, got %+v", initialDecision)
	}

	usePill := usePillForCultivationCommand(t, receiver, UsePillForCultivationPayload{
		DiscipleID: starterDiscipleID,
		PillType:   PillCultivation,
		Quantity:   1,
	}, receiver.state.Runtime.Version)
	if usePill.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected pill command accepted, got %+v", usePill.Result)
	}
	if !hasDomainEventType(usePill.DomainEvents, DomainEventTypeInventoryChanged) || !hasDomainEventType(usePill.DomainEvents, DomainEventTypeCultivationAdvanced) {
		t.Fatalf("expected pill use to write inventory and cultivation events, got %+v", usePill.DomainEvents)
	}
	pillDisciple := usePill.Snapshot.State.Disciples[starterDiscipleID]
	if pillDisciple.Realm.CultivationPoints != cultivationPillPointGain() || pillDisciple.Pressure != 3 {
		t.Fatalf("expected authority pill result on disciple state, got %+v", pillDisciple)
	}
	if got := usePill.Snapshot.State.Inventory.Items[pillInventoryItemID(PillCultivation)].Quantity; got != 1 {
		t.Fatalf("expected authority pill inventory decrement, got %d", got)
	}
	if pillDisciple.Cultivation.CultivationPillAvailable != 1 {
		t.Fatalf("expected snapshot decision to reflect remaining pills, got %+v", pillDisciple.Cultivation)
	}

	reserve := reserveCaveCommand(t, receiver, ReserveCavePayload{DiscipleID: starterDiscipleID, DurationDays: 1}, receiver.state.Runtime.Version)
	if reserve.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected cave reserve accepted, got %+v", reserve.Result)
	}
	if !hasDomainEventType(reserve.DomainEvents, DomainEventTypeResourceChanged) || !hasDomainEventType(reserve.DomainEvents, DomainEventTypeDiscipleAssignmentChanged) {
		t.Fatalf("expected cave reserve to consume resource and update disciple support, got %+v", reserve.DomainEvents)
	}
	caveDisciple := reserve.Snapshot.State.Disciples[starterDiscipleID]
	if !strings.HasPrefix(caveDisciple.WorkTarget.Description, "cultivation:cave") {
		t.Fatalf("expected cave support in authority work target, got %+v", caveDisciple.WorkTarget)
	}
	if caveDisciple.Cultivation.EnvironmentBonus <= initialDecision.EnvironmentBonus || caveDisciple.Cultivation.DailyGain <= initialDecision.DailyGain {
		t.Fatalf("expected cave support to improve authority cultivation decision, before %+v after %+v", initialDecision, caveDisciple.Cultivation)
	}
	if got := reserve.Snapshot.State.Resources.Stock[ResourceKindSpiritStone]; got != 115 {
		t.Fatalf("expected cave reservation to consume spirit stone, got %d", got)
	}
}

func TestAttemptBreakthroughSucceedsThroughAuthorityPath(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-cultivation"), UserID("player-cultivation"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	startCultivationCommand(t, receiver, StartCultivationPayload{DiscipleID: starterDiscipleID}, receiver.state.Runtime.Version)
	receiver.advanceTasksOneDay("session-cultivation")
	dayTwo := receiver.advanceTasksOneDay("session-cultivation")
	readyDisciple := dayTwo.Snapshot.State.Disciples[starterDiscipleID]
	if !readyDisciple.Realm.ReadyForBreakthrough {
		t.Fatalf("expected disciple ready for breakthrough after two days, got %+v", readyDisciple.Realm)
	}
	if !hasClientEventType(dayTwo.Events, ClientEventTypeSectEventChanged) {
		t.Fatalf("expected breakthrough readiness to emit sect event seed, got %+v", dayTwo.Events)
	}
	seededEvent := activeBreakthroughPortentForDisciple(dayTwo.Snapshot.State.Events, starterDiscipleID)
	if seededEvent == nil || seededEvent.Status != SectEventStatusSeeded {
		t.Fatalf("expected authority-owned breakthrough seed after day two, got %+v", dayTwo.Snapshot.State.Events)
	}

	attempt := attemptBreakthroughCommand(t, receiver, AttemptBreakthroughPayload{DiscipleID: starterDiscipleID}, dayTwo.ToVersion)
	if attempt.Result.Status != CommandResultStatusRejected {
		t.Fatalf("expected breakthrough to wait for authority omen, got %+v", attempt.Result)
	}
	if attempt.Result.Error == nil || attempt.Result.Error.Code != CommandErrorCodeCultivationNotReady {
		t.Fatalf("expected cultivation-not-ready before omen reveal, got %+v", attempt.Result.Error)
	}

	dayThree := receiver.advanceTasksOneDay("session-cultivation")
	if !hasClientEventType(dayThree.Events, ClientEventTypeSectEventChanged) {
		t.Fatalf("expected day three to reveal authority omen, got %+v", dayThree.Events)
	}
	foreshadowed := activeBreakthroughPortentForDisciple(dayThree.Snapshot.State.Events, starterDiscipleID)
	if foreshadowed == nil || foreshadowed.Status != SectEventStatusForeshadowed || foreshadowed.OmenText == "" {
		t.Fatalf("expected foreshadowed breakthrough omen, got %+v", dayThree.Snapshot.State.Events)
	}
	decision := dayThree.Snapshot.State.Disciples[starterDiscipleID].Cultivation
	if decision.OmenStatus != string(SectEventStatusForeshadowed) || decision.OmenText == "" || decision.BreakthroughRisk > decision.BreakthroughRiskLimit || decision.BreakthroughSuccessRate != 100 {
		t.Fatalf("expected authority-readable breakthrough decision after omen reveal, got %+v", decision)
	}

	attempt = attemptBreakthroughCommand(t, receiver, AttemptBreakthroughPayload{DiscipleID: starterDiscipleID}, dayThree.ToVersion)
	if attempt.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected breakthrough accepted after omen, got %+v", attempt.Result)
	}
	if !hasDomainEventType(attempt.DomainEvents, DomainEventTypeBreakthroughSucceeded) || !hasDomainEventType(attempt.DomainEvents, DomainEventTypeSectEventResolved) {
		t.Fatalf("expected authority breakthrough success plus event resolution, got %+v", attempt.DomainEvents)
	}

	disciple := attempt.Snapshot.State.Disciples[starterDiscipleID]
	if disciple.Realm.Stage != RealmQiEntry {
		t.Fatalf("expected disciple to reach qi_entry, got %+v", disciple.Realm)
	}
	if disciple.Realm.CultivationPoints != 0 || disciple.Realm.ReadyForBreakthrough {
		t.Fatalf("expected cultivation state reset after success, got %+v", disciple.Realm)
	}
	if disciple.Pressure != 0 || disciple.InjuryLevel != 0 {
		t.Fatalf("expected no lingering risk penalties on success, got %+v", disciple)
	}
	if attempt.Snapshot.State.Resources.Stock[ResourceKindSpiritStone] != 112 {
		t.Fatalf("expected cultivation and breakthrough to consume 8 spirit stone, got %d", attempt.Snapshot.State.Resources.Stock[ResourceKindSpiritStone])
	}
	if activeBreakthroughPortentForDisciple(attempt.Snapshot.State.Events, starterDiscipleID) != nil {
		t.Fatalf("expected breakthrough omen to resolve out of active state, got %+v", attempt.Snapshot.State.Events)
	}
	if len(attempt.Snapshot.State.Events.ResolvedEvents) == 0 {
		t.Fatalf("expected resolved event history to record breakthrough outcome")
	}
}

func TestAttemptBreakthroughFailureWritesRiskIntoAuthorityState(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-cultivation-risk"), UserID("player-cultivation"), "青崖宗")
	starter := initial.Disciples[starterDiscipleID]
	starter.Pressure = 35
	starter.InjuryLevel = 1
	initial.Disciples[starterDiscipleID] = starter

	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	startCultivationCommand(t, receiver, StartCultivationPayload{DiscipleID: starterDiscipleID}, receiver.state.Runtime.Version)
	receiver.advanceTasksOneDay("session-cultivation-risk")
	receiver.advanceTasksOneDay("session-cultivation-risk")
	dayThree := receiver.advanceTasksOneDay("session-cultivation-risk")
	if !dayThree.Snapshot.State.Disciples[starterDiscipleID].Realm.ReadyForBreakthrough {
		t.Fatalf("expected disciple ready for breakthrough after three days under risk, got %+v", dayThree.Snapshot.State.Disciples[starterDiscipleID].Realm)
	}
	seededEvent := activeBreakthroughPortentForDisciple(dayThree.Snapshot.State.Events, starterDiscipleID)
	if seededEvent == nil || seededEvent.Status != SectEventStatusSeeded {
		t.Fatalf("expected authority seed before risky breakthrough, got %+v", dayThree.Snapshot.State.Events)
	}

	dayFour := receiver.advanceTasksOneDay("session-cultivation-risk")
	foreshadowed := activeBreakthroughPortentForDisciple(dayFour.Snapshot.State.Events, starterDiscipleID)
	if foreshadowed == nil || foreshadowed.Status != SectEventStatusForeshadowed {
		t.Fatalf("expected omen reveal before risky breakthrough, got %+v", dayFour.Snapshot.State.Events)
	}

	attempt := attemptBreakthroughCommand(t, receiver, AttemptBreakthroughPayload{DiscipleID: starterDiscipleID}, dayFour.ToVersion)
	if attempt.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected authority breakthrough attempt to settle, got %+v", attempt.Result)
	}
	if !hasDomainEventType(attempt.DomainEvents, DomainEventTypeBreakthroughFailed) || !hasDomainEventType(attempt.DomainEvents, DomainEventTypeSectEventResolved) {
		t.Fatalf("expected authority breakthrough failure plus event resolution, got %+v", attempt.DomainEvents)
	}

	disciple := attempt.Snapshot.State.Disciples[starterDiscipleID]
	if disciple.Realm.Stage != RealmMortal {
		t.Fatalf("expected failed breakthrough to keep mortal stage, got %+v", disciple.Realm)
	}
	if disciple.Realm.FailedBreakthroughCount != 1 {
		t.Fatalf("expected failed breakthrough count 1, got %+v", disciple.Realm)
	}
	if disciple.InjuryLevel != 2 || disciple.Pressure != 53 {
		t.Fatalf("expected failure to add injury/pressure risk, got %+v", disciple)
	}
	if disciple.HP != starterDiscipleMaxHP-15 {
		t.Fatalf("expected failure to reduce hp by 15, got %d", disciple.HP)
	}
	if len(attempt.Snapshot.State.Events.ResolvedEvents) == 0 {
		t.Fatalf("expected failure path to append resolved portent summary")
	}
}

func TestCultivationReplayRestoreContinuesAuthorityGrowth(t *testing.T) {
	base := NewInitialSectState(SectID("sect-cultivation-restore"), UserID("player-cultivation"), "青崖宗")
	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	start := startCultivationCommand(t, receiver, StartCultivationPayload{DiscipleID: starterDiscipleID}, receiver.state.Runtime.Version)
	dayOne := receiver.advanceTasksOneDay("session-cultivation-restore")

	savepoint := NewSnapshotReplaySavepoint(base)
	savepoint.AppendReplay(start.DomainEvents)
	savepoint.AppendReplay(dayOne.DomainEvents)

	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore cultivation savepoint: %v", err)
	}
	restoredDisciple := restored.Disciples[starterDiscipleID]
	if restoredDisciple.AssignmentKind != DiscipleAssignmentCultivation || restoredDisciple.Realm.CultivationPoints != 15 {
		t.Fatalf("expected restored cultivation state after one day, got %+v", restoredDisciple)
	}

	restoredReceiver, ok := NewSectActor(restored).(*SectActor)
	if !ok {
		t.Fatalf("expected restored concrete sect actor receiver")
	}
	dayTwo := restoredReceiver.advanceTasksOneDay("session-cultivation-restore")
	seededEvent := activeBreakthroughPortentForDisciple(dayTwo.Snapshot.State.Events, starterDiscipleID)
	if seededEvent == nil || seededEvent.Status != SectEventStatusSeeded {
		t.Fatalf("expected restored replay to seed breakthrough portent, got %+v", dayTwo.Snapshot.State.Events)
	}
	dayThree := restoredReceiver.advanceTasksOneDay("session-cultivation-restore")
	foreshadowed := activeBreakthroughPortentForDisciple(dayThree.Snapshot.State.Events, starterDiscipleID)
	if foreshadowed == nil || foreshadowed.Status != SectEventStatusForeshadowed {
		t.Fatalf("expected restored replay to reveal authority omen, got %+v", dayThree.Snapshot.State.Events)
	}
	attempt := attemptBreakthroughCommand(t, restoredReceiver, AttemptBreakthroughPayload{DiscipleID: starterDiscipleID}, dayThree.ToVersion)
	if attempt.Snapshot.State.Disciples[starterDiscipleID].Realm.Stage != RealmQiEntry {
		t.Fatalf("expected restored authority growth path to reach qi_entry, got %+v", attempt.Snapshot.State.Disciples[starterDiscipleID].Realm)
	}
}

func TestBreakthroughFateForeshadowingRemainsAuthorityOwned(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-fate"), UserID("player-fate"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	startCultivationCommand(t, receiver, StartCultivationPayload{DiscipleID: starterDiscipleID}, receiver.state.Runtime.Version)
	receiver.advanceTasksOneDay("session-fate")
	dayTwo := receiver.advanceTasksOneDay("session-fate")

	seeded := activeBreakthroughPortentForDisciple(dayTwo.Snapshot.State.Events, starterDiscipleID)
	if seeded == nil || seeded.Status != SectEventStatusSeeded {
		t.Fatalf("expected authority event seed on breakthrough readiness, got %+v", dayTwo.Snapshot.State.Events)
	}
	if seeded.OmenText == "" {
		t.Fatalf("expected seeded event to carry omen copy, got %+v", seeded)
	}

	dayThree := receiver.advanceTasksOneDay("session-fate")
	foreshadowed := activeBreakthroughPortentForDisciple(dayThree.Snapshot.State.Events, starterDiscipleID)
	if foreshadowed == nil || foreshadowed.Status != SectEventStatusForeshadowed {
		t.Fatalf("expected authority-only omen reveal, got %+v", dayThree.Snapshot.State.Events)
	}
	if !hasClientEventType(dayThree.Events, ClientEventTypeSectEventChanged) {
		t.Fatalf("expected client to consume omen change event, got %+v", dayThree.Events)
	}
}

func publishTaskCommand(t *testing.T, receiver *SectActor, payload PublishTaskPayload) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal publish task payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-taskhall",
		Command: ClientCommand{
			CmdID:       "cmd-task-publish-1",
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypePublishTask,
			Payload:     body,
			BaseVersion: receiver.state.Runtime.Version,
		},
	})
}

func assignTaskCommand(t *testing.T, receiver *SectActor, payload AssignDiscipleTaskPayload, baseVersion Version) SubmitCommandResponse {
	t.Helper()
	return assignTaskCommandWithID(t, receiver, payload, baseVersion, "cmd-task-assign-1")
}

func assignTaskCommandWithID(t *testing.T, receiver *SectActor, payload AssignDiscipleTaskPayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal assign task payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-taskhall",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeAssignDiscipleTask,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func cancelTaskCommand(t *testing.T, receiver *SectActor, payload CancelTaskPayload, baseVersion Version) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal cancel task payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-taskhall",
		Command: ClientCommand{
			CmdID:       "cmd-task-cancel-1",
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeCancelTask,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func setTaskPriorityCommand(t *testing.T, receiver *SectActor, payload SetTaskPriorityPayload, baseVersion Version) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal set task priority payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-taskhall",
		Command: ClientCommand{
			CmdID:       "cmd-task-priority-1",
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeSetTaskPriority,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func exchangeContributionCommand(t *testing.T, receiver *SectActor, payload ExchangeContributionItemPayload, baseVersion Version) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal exchange contribution payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-taskhall",
		Command: ClientCommand{
			CmdID:       "cmd-contribution-exchange-1",
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeExchangeContributionItem,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func startProductionCommandWithID(t *testing.T, receiver *SectActor, payload StartProductionPayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal start production payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-production",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeStartProduction,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func cancelProductionCommand(t *testing.T, receiver *SectActor, payload CancelProductionPayload, baseVersion Version) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal cancel production payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-production",
		Command: ClientCommand{
			CmdID:       "cmd-production-cancel-1",
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeCancelProduction,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func adjustProductionCommand(t *testing.T, receiver *SectActor, payload AdjustProductionPayload, baseVersion Version) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal adjust production payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-production",
		Command: ClientCommand{
			CmdID:       "cmd-production-adjust-1",
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeAdjustProduction,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func startCultivationCommand(t *testing.T, receiver *SectActor, payload StartCultivationPayload, baseVersion Version) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal start cultivation payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-cultivation",
		Command: ClientCommand{
			CmdID:       "cmd-cultivation-start-1",
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeStartCultivation,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func usePillForCultivationCommand(t *testing.T, receiver *SectActor, payload UsePillForCultivationPayload, baseVersion Version) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal use pill cultivation payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-cultivation",
		Command: ClientCommand{
			CmdID:       "cmd-cultivation-pill-1",
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeUsePillForCultivation,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func reserveCaveCommand(t *testing.T, receiver *SectActor, payload ReserveCavePayload, baseVersion Version) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal reserve cave payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-cultivation",
		Command: ClientCommand{
			CmdID:       "cmd-cultivation-cave-1",
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeReserveCave,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func advanceSectDays(receiver *SectActor, days int, sessionID string) AdvanceTasksOneDayResponse {
	var response AdvanceTasksOneDayResponse
	for day := 0; day < days; day++ {
		response = receiver.advanceTasksOneDay(sessionID)
	}
	return response
}

func attemptBreakthroughCommand(t *testing.T, receiver *SectActor, payload AttemptBreakthroughPayload, baseVersion Version) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal breakthrough payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-cultivation",
		Command: ClientCommand{
			CmdID:       "cmd-breakthrough-1",
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeAttemptBreakthrough,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func startRecruitmentCommand(t *testing.T, receiver *SectActor, payload StartRecruitmentPayload, baseVersion Version) SubmitCommandResponse {
	t.Helper()
	return startRecruitmentCommandWithID(t, receiver, payload, baseVersion, "cmd-recruitment-start-1")
}

func startRecruitmentCommandWithID(t *testing.T, receiver *SectActor, payload StartRecruitmentPayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal start recruitment payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-admission",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeStartRecruitment,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func acceptCandidateCommand(t *testing.T, receiver *SectActor, payload AcceptCandidatePayload, baseVersion Version) SubmitCommandResponse {
	t.Helper()
	return acceptCandidateCommandWithID(t, receiver, payload, baseVersion, "cmd-candidate-accept-1")
}

func acceptCandidateCommandWithID(t *testing.T, receiver *SectActor, payload AcceptCandidatePayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal accept candidate payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-admission",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeAcceptCandidate,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func rejectCandidateCommand(t *testing.T, receiver *SectActor, payload RejectCandidatePayload, baseVersion Version) SubmitCommandResponse {
	t.Helper()
	return rejectCandidateCommandWithID(t, receiver, payload, baseVersion, "cmd-candidate-reject-1")
}

func rejectCandidateCommandWithID(t *testing.T, receiver *SectActor, payload RejectCandidatePayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal reject candidate payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-admission",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeRejectCandidate,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func sortedCandidateIDs(candidates map[CandidateID]CandidateState) []CandidateID {
	ids := make([]CandidateID, 0, len(candidates))
	for id := range candidates {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func hasClientEventType(events []ClientEvent, want ClientEventType) bool {
	for _, event := range events {
		if event.Type == want {
			return true
		}
	}
	return false
}

func hasPatchPath(ops []PatchOp, want string) bool {
	for _, op := range ops {
		if op.Path == want {
			return true
		}
	}
	return false
}

func hasDomainEventType(events []DomainEvent, want DomainEventType) bool {
	for _, event := range events {
		if event.Type == want {
			return true
		}
	}
	return false
}
