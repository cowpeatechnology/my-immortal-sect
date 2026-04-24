package sect

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSnapshotReplaySavepointBuildsDiscipleDiaryFromEventLog(t *testing.T) {
	base := NewInitialSectState(SectID("sect-diary"), UserID("player-diary"), "青崖宗")
	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	start := startCultivationCommand(t, receiver, StartCultivationPayload{DiscipleID: starterDiscipleID}, receiver.state.Runtime.Version)
	dayOne := receiver.advanceTasksOneDay("session-diary")
	dayTwo := receiver.advanceTasksOneDay("session-diary")
	dayThree := receiver.advanceTasksOneDay("session-diary")
	attempt := attemptBreakthroughCommand(t, receiver, AttemptBreakthroughPayload{DiscipleID: starterDiscipleID}, dayThree.ToVersion)

	savepoint := NewSnapshotReplaySavepoint(base)
	savepoint.AppendReplay(start.DomainEvents)
	savepoint.AppendReplay(dayOne.DomainEvents)
	savepoint.AppendReplay(dayTwo.DomainEvents)
	savepoint.AppendReplay(dayThree.DomainEvents)
	savepoint.AppendReplay(attempt.DomainEvents)

	if len(savepoint.EventLogAfter(dayOne.ToVersion)) == 0 {
		t.Fatalf("expected bounded event log after first cultivation day")
	}

	diary := savepoint.DiscipleDiary(starterDiscipleID)
	if len(diary) == 0 {
		t.Fatalf("expected disciple diary entries to be derived from event log")
	}
	if diary[0].ReplaySource != "event_log" {
		t.Fatalf("expected diary to declare event_log replay source, got %+v", diary[0])
	}

	var sawOmen bool
	var sawBreakthrough bool
	for _, entry := range diary {
		if entry.EventType == "omen" && entry.Summary != "" {
			sawOmen = true
		}
		if entry.EventType == "breakthrough" {
			sawBreakthrough = true
		}
	}
	if !sawOmen {
		t.Fatalf("expected diary to include fate foreshadowing derived from event log, got %+v", diary)
	}
	if !sawBreakthrough {
		t.Fatalf("expected diary to include breakthrough outcome derived from event log, got %+v", diary)
	}
}

func TestEventLogFeedbackCoversPlayerFacingCategories(t *testing.T) {
	base := NewInitialSectState(SectID("sect-feedback"), UserID("player-feedback"), "青崖宗")
	disciple := base.Disciples[starterDiscipleID]
	cultivating := disciple
	cultivating.Realm.CultivationPoints = 30
	success := cultivating
	success.Realm.Stage = RealmQiEntry
	success.Realm.CultivationPoints = 0
	injured := disciple
	injured.InjuryLevel = 2
	injured.Pressure = 53
	injured.HP = 85
	sourceDiscipleID := starterDiscipleID

	events := []DomainEvent{
		testFeedbackEvent("evt-resource", 1, DomainEventTypeResourceChanged, ResourceChangedPayload{
			Changes: map[ResourceKind]int64{ResourceKindSpiritStone: -3, ResourceKindHerb: 2},
			Reason:  "test",
		}),
		testFeedbackEvent("evt-task", 2, DomainEventTypeTaskCompleted, TaskChangedPayload{Task: TaskState{
			TaskID:                TaskID("task-feedback"),
			Title:                 "巡山问路",
			Status:                TaskStatusCompleted,
			AssignedDiscipleIDs:   []DiscipleID{starterDiscipleID},
			CompletedProgressDays: 1,
			RequiredProgressDays:  1,
			ContributionReward:    8,
		}}),
		testFeedbackEvent("evt-production", 3, DomainEventTypeProductionChanged, ProductionChangedPayload{Production: ProductionJob{
			ProductionID:    ProductionID("prod-feedback"),
			Kind:            ProductionKindRefinement,
			RecipeID:        RecipeID("formation_refine_mvp"),
			Status:          ProductionStatusCompleted,
			CompletedCycles: 1,
		}}),
		testFeedbackEvent("evt-cultivation", 4, DomainEventTypeCultivationAdvanced, DiscipleChangedPayload{Disciple: cultivating}),
		testFeedbackEvent("evt-breakthrough", 5, DomainEventTypeBreakthroughSucceeded, DiscipleChangedPayload{Disciple: success}),
		testFeedbackEvent("evt-injury", 6, DomainEventTypeBreakthroughFailed, DiscipleChangedPayload{Disciple: injured}),
		testFeedbackEvent("evt-monthly", 7, DomainEventTypePayrollDelayed, PayrollSettlementPayload{
			MonthIndex:   1,
			DiscipleID:   starterDiscipleID,
			Amount:       2,
			ArrearsAfter: 1,
		}),
		testFeedbackEvent("evt-omen", 8, DomainEventTypeSectEventForeshadowed, SectEventChangedPayload{Event: SectEvent{
			EventID:          EventID("omen-feedback"),
			Kind:             "breakthrough_portent",
			Status:           SectEventStatusForeshadowed,
			Severity:         2,
			Title:            "心魔风声",
			Description:      "洞府风声不止。",
			OmenText:         "近日心神不宁，突破需谨慎。",
			SourceDiscipleID: &sourceDiscipleID,
		}}),
	}

	logEntries := BuildEventLogEntriesFromEventLog(events)
	if len(logEntries) != len(events) {
		t.Fatalf("expected bounded event log entries for every domain event, got %d vs %d", len(logEntries), len(events))
	}
	for _, entry := range logEntries {
		if entry.ReplaySource != "event_log" || entry.Summary == "" {
			t.Fatalf("event log entry must be readable and declare event_log source, got %+v", entry)
		}
	}

	feedback := BuildSectEventFeedbackFromEventLog(events)
	seen := map[string]bool{}
	for _, entry := range feedback {
		if entry.ReplaySource != "event_log" || entry.Summary == "" {
			t.Fatalf("event feedback must be readable and declare event_log source, got %+v", entry)
		}
		seen[entry.Category] = true
	}
	for _, category := range []string{"task_result", "resource_change", "production", "cultivation", "breakthrough", "injury", "monthly", "omen"} {
		if !seen[category] {
			t.Fatalf("expected event feedback category %q from event log, got %+v", category, feedback)
		}
	}

	diary := BuildDiscipleDiaryFromEventLog(events, starterDiscipleID)
	diarySeen := map[string]bool{}
	for _, entry := range diary {
		diarySeen[entry.EventType] = true
	}
	for _, eventType := range []string{"task.completed", "cultivation", "breakthrough", "injury", "monthly", "omen"} {
		if !diarySeen[eventType] {
			t.Fatalf("expected diary event type %q from event log, got %+v", eventType, diary)
		}
	}
}

func TestSectSnapshotSurfacesBoundedEventLogDiaryAndFeedback(t *testing.T) {
	base := NewInitialSectState(SectID("sect-feedback-snapshot"), UserID("player-feedback"), "青崖宗")
	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	start := startCultivationCommand(t, receiver, StartCultivationPayload{DiscipleID: starterDiscipleID}, receiver.state.Runtime.Version)
	advance := receiver.advanceTasksOneDay("session-feedback")
	if len(start.DomainEvents) == 0 || len(advance.DomainEvents) == 0 {
		t.Fatalf("expected commands to produce authority event log entries")
	}
	snapshot := receiver.snapshot(UserID("player-feedback"), "session-feedback")
	if len(snapshot.EventLog) == 0 || len(snapshot.Diary) == 0 || len(snapshot.EventSummaries) == 0 {
		t.Fatalf("expected snapshot to expose event_log-derived feedback, got log=%d diary=%d summary=%d", len(snapshot.EventLog), len(snapshot.Diary), len(snapshot.EventSummaries))
	}

	savepoint := NewSnapshotReplaySavepoint(base)
	savepoint.AppendReplay(start.DomainEvents)
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore feedback savepoint: %v", err)
	}
	restoredReceiver, ok := NewSectActorWithEventLog(restored, savepoint.EventLog).(*SectActor)
	if !ok {
		t.Fatalf("expected restored concrete sect actor receiver")
	}
	restoredSnapshot := restoredReceiver.snapshot(UserID("player-feedback"), "session-feedback")
	if len(restoredSnapshot.EventLog) != len(savepoint.EventLog) || len(restoredSnapshot.Diary) == 0 {
		t.Fatalf("expected restored actor to keep event_log-derived diary, got log=%d/%d diary=%d", len(restoredSnapshot.EventLog), len(savepoint.EventLog), len(restoredSnapshot.Diary))
	}
}

func TestMidSessionRestoreReplayMatchesContinuousAuthorityPath(t *testing.T) {
	base := NewInitialSectState(SectID("sect-restore-proof"), UserID("player-restore"), "青崖宗")

	continuous, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete continuous sect actor receiver")
	}
	startContinuous := startCultivationCommand(t, continuous, StartCultivationPayload{DiscipleID: starterDiscipleID}, continuous.state.Runtime.Version)
	_ = startContinuous
	continuous.advanceTasksOneDay("session-restore-proof")
	continuousDayTwo := continuous.advanceTasksOneDay("session-restore-proof")
	continuousDayThree := continuous.advanceTasksOneDay("session-restore-proof")
	continuousFinal := attemptBreakthroughCommand(t, continuous, AttemptBreakthroughPayload{DiscipleID: starterDiscipleID}, continuousDayThree.ToVersion)
	if continuousFinal.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected continuous breakthrough to settle, got %+v", continuousFinal.Result)
	}

	restoreSeed, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete restore sect actor receiver")
	}
	startRestore := startCultivationCommand(t, restoreSeed, StartCultivationPayload{DiscipleID: starterDiscipleID}, restoreSeed.state.Runtime.Version)
	restoreDayOne := restoreSeed.advanceTasksOneDay("session-restore-proof")

	savepoint := NewSnapshotReplaySavepoint(base)
	savepoint.AppendReplay(startRestore.DomainEvents)
	savepoint.AppendReplay(restoreDayOne.DomainEvents)

	restoredState, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore mid-session savepoint: %v", err)
	}
	restoredReceiver, ok := NewSectActor(restoredState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete restored sect actor receiver")
	}
	restoredDayTwo := restoredReceiver.advanceTasksOneDay("session-restore-proof")
	restoredDayThree := restoredReceiver.advanceTasksOneDay("session-restore-proof")
	restoredFinal := attemptBreakthroughCommand(t, restoredReceiver, AttemptBreakthroughPayload{DiscipleID: starterDiscipleID}, restoredDayThree.ToVersion)
	if restoredFinal.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected restored breakthrough to settle, got %+v", restoredFinal.Result)
	}

	if continuousFinal.Snapshot.SceneVersion != restoredFinal.Snapshot.SceneVersion {
		t.Fatalf("expected same final version after restore continuity, got %d vs %d", continuousFinal.Snapshot.SceneVersion, restoredFinal.Snapshot.SceneVersion)
	}
	if !reflect.DeepEqual(continuousFinal.Snapshot.State, restoredFinal.Snapshot.State) {
		t.Fatalf("expected restored path to match continuous authority state\ncontinuous: %+v\nrestored: %+v", continuousFinal.Snapshot.State, restoredFinal.Snapshot.State)
	}
	if !reflect.DeepEqual(continuousDayTwo.Snapshot.State.Events, restoredDayTwo.Snapshot.State.Events) {
		t.Fatalf("expected restored day-two event seed state to match continuous path")
	}
}

func testFeedbackEvent(id string, version Version, eventType DomainEventType, payload any) DomainEvent {
	return DomainEvent{
		EventID:  id,
		SectID:   SectID("sect-feedback"),
		Version:  version,
		Type:     eventType,
		Payload:  mustMarshalPayload(payload),
		GameTick: int64(version) * 10,
	}
}

func TestRecruitmentRosterSnapshotReplayMatchesContinuousAuthorityPath(t *testing.T) {
	base := NewInitialSectState(SectID("sect-roster-proof"), UserID("player-roster-proof"), "青崖宗")

	continuous, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete continuous sect actor receiver")
	}
	startContinuous := startRecruitmentCommandWithID(t, continuous, StartRecruitmentPayload{
		CandidateCount:        3,
		InvestmentSpiritStone: 30,
		DurationDays:          6,
	}, continuous.state.Runtime.Version, "cmd-roster-recruitment-start")
	candidateIDs := sortedCandidateIDs(startContinuous.Snapshot.State.Admissions.Candidates)
	acceptedCandidate := startContinuous.Snapshot.State.Admissions.Candidates[candidateIDs[0]]
	rejectedCandidate := startContinuous.Snapshot.State.Admissions.Candidates[candidateIDs[1]]

	acceptContinuous := acceptCandidateCommandWithID(t, continuous, AcceptCandidatePayload{
		CandidateID: acceptedCandidate.CandidateID,
	}, startContinuous.Result.SceneVersion, "cmd-roster-candidate-accept")
	acceptedDiscipleID := discipleIDByName(t, acceptContinuous.Snapshot.State, acceptedCandidate.Name)
	rejectCandidateCommandWithID(t, continuous, RejectCandidatePayload{
		CandidateID: rejectedCandidate.CandidateID,
	}, acceptContinuous.Result.SceneVersion, "cmd-roster-candidate-reject")
	publishContinuous := publishTaskCommand(t, continuous, PublishTaskPayload{
		Kind:                 "new_disciple_orientation",
		Title:                "新弟子入门执事",
		RequiredProgressDays: 2,
		ContributionReward:   5,
	})
	assignContinuous := assignTaskCommand(t, continuous, AssignDiscipleTaskPayload{
		TaskID:     TaskID("task-1"),
		DiscipleID: acceptedDiscipleID,
	}, publishContinuous.Result.SceneVersion)

	restoreSeed, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete restore sect actor receiver")
	}
	startRestore := startRecruitmentCommandWithID(t, restoreSeed, StartRecruitmentPayload{
		CandidateCount:        3,
		InvestmentSpiritStone: 30,
		DurationDays:          6,
	}, restoreSeed.state.Runtime.Version, "cmd-roster-recruitment-start")
	restoreCandidateIDs := sortedCandidateIDs(startRestore.Snapshot.State.Admissions.Candidates)
	if !reflect.DeepEqual(candidateIDs, restoreCandidateIDs) {
		t.Fatalf("expected deterministic candidate ids before restore branch, got %v vs %v", candidateIDs, restoreCandidateIDs)
	}
	acceptRestore := acceptCandidateCommandWithID(t, restoreSeed, AcceptCandidatePayload{
		CandidateID: acceptedCandidate.CandidateID,
	}, startRestore.Result.SceneVersion, "cmd-roster-candidate-accept")
	snapshotAfterAccept := acceptRestore.Snapshot.State.Clone()

	savepoint := NewSnapshotReplaySavepoint(snapshotAfterAccept)
	rejectRestore := rejectCandidateCommandWithID(t, restoreSeed, RejectCandidatePayload{
		CandidateID: rejectedCandidate.CandidateID,
	}, acceptRestore.Result.SceneVersion, "cmd-roster-candidate-reject")
	publishRestore := publishTaskCommand(t, restoreSeed, PublishTaskPayload{
		Kind:                 "new_disciple_orientation",
		Title:                "新弟子入门执事",
		RequiredProgressDays: 2,
		ContributionReward:   5,
	})
	assignRestore := assignTaskCommand(t, restoreSeed, AssignDiscipleTaskPayload{
		TaskID:     TaskID("task-1"),
		DiscipleID: acceptedDiscipleID,
	}, publishRestore.Result.SceneVersion)
	savepoint.AppendReplay(rejectRestore.DomainEvents)
	savepoint.AppendReplay(publishRestore.DomainEvents)
	savepoint.AppendReplay(assignRestore.DomainEvents)

	if got := savepoint.EventLogAfter(snapshotAfterAccept.Runtime.Version); len(got) != len(savepoint.ReplayEvents) {
		t.Fatalf("expected bounded event log after snapshot version %d, got events=%d replay=%d", snapshotAfterAccept.Runtime.Version, len(got), len(savepoint.ReplayEvents))
	}
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore roster savepoint: %v", err)
	}

	continuousFinal := assignContinuous.Snapshot.State
	if continuousJSON, restoredJSON := canonicalStateJSON(t, continuousFinal), canonicalStateJSON(t, restored); continuousJSON != restoredJSON {
		t.Fatalf("expected restored roster authority state to match continuous path\ncontinuous: %+v\nrestored: %+v", continuousFinal, restored)
	}
	if len(restored.Admissions.Candidates) != 1 || restored.Admissions.CurrentRecruitment == nil {
		t.Fatalf("expected one pending authority candidate after accept/reject replay, got %+v", restored.Admissions)
	}
	if _, ok := restored.Contribution.Accounts[acceptedDiscipleID]; !ok {
		t.Fatalf("expected accepted disciple contribution account after restore, got %+v", restored.Contribution.Accounts)
	}
	restoredDisciple := restored.Disciples[acceptedDiscipleID]
	if restoredDisciple.AssignmentKind != DiscipleAssignmentTask || restoredDisciple.AssignmentTask == nil || *restoredDisciple.AssignmentTask != TaskID("task-1") {
		t.Fatalf("expected accepted disciple assignment to survive snapshot replay, got %+v", restoredDisciple)
	}
	if _, exists := restored.Admissions.Candidates[rejectedCandidate.CandidateID]; exists {
		t.Fatalf("rejected candidate must not reappear after restore, got %+v", restored.Admissions.Candidates)
	}
}

func TestRecruitmentCommandIDReplayIsIdempotent(t *testing.T) {
	base := NewInitialSectState(SectID("sect-roster-idempotent"), UserID("player-roster-idempotent"), "青崖宗")
	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	startPayload := StartRecruitmentPayload{
		CandidateCount:        2,
		InvestmentSpiritStone: 25,
		DurationDays:          5,
	}
	firstStart := startRecruitmentCommandWithID(t, receiver, startPayload, receiver.state.Runtime.Version, "cmd-idempotent-recruitment-start")
	duplicateStart := startRecruitmentCommandWithID(t, receiver, startPayload, 0, "cmd-idempotent-recruitment-start")
	if firstStart.Result.Status != CommandResultStatusAccepted || duplicateStart.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected duplicate recruitment command to return accepted cached result, got first=%+v duplicate=%+v", firstStart.Result, duplicateStart.Result)
	}
	if receiver.state.Runtime.Version != firstStart.Result.SceneVersion {
		t.Fatalf("duplicate recruitment command must not advance authority version, got state=%d first=%d", receiver.state.Runtime.Version, firstStart.Result.SceneVersion)
	}
	if !reflect.DeepEqual(firstStart.Snapshot.State, duplicateStart.Snapshot.State) {
		t.Fatalf("duplicate recruitment command should return cached snapshot\nfirst: %+v\nduplicate: %+v", firstStart.Snapshot.State, duplicateStart.Snapshot.State)
	}
	if len(receiver.state.Admissions.Candidates) != 2 {
		t.Fatalf("duplicate recruitment command must not generate extra candidates, got %+v", receiver.state.Admissions.Candidates)
	}

	candidateID := sortedCandidateIDs(firstStart.Snapshot.State.Admissions.Candidates)[0]
	acceptPayload := AcceptCandidatePayload{CandidateID: candidateID}
	firstAccept := acceptCandidateCommandWithID(t, receiver, acceptPayload, firstStart.Result.SceneVersion, "cmd-idempotent-candidate-accept")
	duplicateAccept := acceptCandidateCommandWithID(t, receiver, acceptPayload, firstStart.Result.SceneVersion, "cmd-idempotent-candidate-accept")
	if firstAccept.Result.Status != CommandResultStatusAccepted || duplicateAccept.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected duplicate accept command to return accepted cached result, got first=%+v duplicate=%+v", firstAccept.Result, duplicateAccept.Result)
	}
	if receiver.state.Runtime.Version != firstAccept.Result.SceneVersion {
		t.Fatalf("duplicate accept command must not advance authority version, got state=%d first=%d", receiver.state.Runtime.Version, firstAccept.Result.SceneVersion)
	}
	if !reflect.DeepEqual(firstAccept.Snapshot.State, duplicateAccept.Snapshot.State) {
		t.Fatalf("duplicate accept command should return cached snapshot\nfirst: %+v\nduplicate: %+v", firstAccept.Snapshot.State, duplicateAccept.Snapshot.State)
	}
	if len(receiver.state.Disciples) != 2 || len(receiver.state.Contribution.Accounts) != 2 {
		t.Fatalf("duplicate accept command must not create duplicate disciple/account, got disciples=%+v accounts=%+v", receiver.state.Disciples, receiver.state.Contribution.Accounts)
	}
}

func discipleIDByName(t *testing.T, state SectState, name string) DiscipleID {
	t.Helper()
	for discipleID, disciple := range state.Disciples {
		if disciple.Name == name {
			return discipleID
		}
	}
	t.Fatalf("expected disciple named %q in roster, got %+v", name, state.Disciples)
	return ""
}

func canonicalStateJSON(t *testing.T, state SectState) string {
	t.Helper()
	body, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal canonical state: %v", err)
	}
	return string(body)
}
