package sect

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestDiscipleMemoryAndRelationshipTagsGenerateFromAuthorityEvents(t *testing.T) {
	state := promotionReadyState(true)
	starter := state.Disciples[starterDiscipleID]

	completedTask := TaskState{
		TaskID:              TaskID("task-memory-merchant"),
		Kind:                "merchant_commission",
		Type:                TaskTypeExternal,
		Title:               "商会委托",
		Status:              TaskStatusCompleted,
		Evaluation:          TaskEvaluationExcellent,
		AssignedDiscipleIDs: []DiscipleID{starterDiscipleID},
		ContributionReward:  5,
		Risk:                45,
	}
	applyMemoryEvent(t, &state, DomainEventTypeTaskCompleted, 1, TaskChangedPayload{Task: completedTask})

	breakthrough := state.Disciples[starterDiscipleID]
	breakthrough.Realm.Stage = RealmQiMiddle
	breakthrough.Realm.ReadyForBreakthrough = false
	breakthrough.Realm.CultivationPoints = 0
	applyMemoryEvent(t, &state, DomainEventTypeBreakthroughSucceeded, 2, DiscipleChangedPayload{Disciple: breakthrough})

	applyMemoryEvent(t, &state, DomainEventTypePayrollDelayed, 3, PayrollSettlementPayload{
		MonthIndex:   1,
		DiscipleID:   starterDiscipleID,
		Amount:       monthlyStipendFor(starter),
		ArrearsAfter: 1,
	})
	applyMemoryEvent(t, &state, DomainEventTypeMonthlyObligationChecked, 4, MonthlyObligationCheckedPayload{
		MonthIndex:      1,
		DiscipleID:      starterDiscipleID,
		RequiredDays:    4,
		CompletedDays:   1,
		ViolationAdded:  true,
		ViolationsAfter: 1,
	})

	sourceDiscipleID := starterDiscipleID
	omen := SectEvent{
		EventID:           EventID("event-memory-crisis"),
		Kind:              "task_crisis_clue",
		Status:            SectEventStatusForeshadowed,
		Severity:          2,
		Title:             "巡山异兆",
		Description:       "巡山弟子带回了异动线索。",
		SourceDiscipleID:  &sourceDiscipleID,
		SeededAtVersion:   5,
		RevealAtVersion:   5,
		RevealedAtVersion: 5,
		ExpiresAtDay:      state.Time.CalendarDay + 3,
	}
	applyMemoryEvent(t, &state, DomainEventTypeSectEventForeshadowed, 5, SectEventChangedPayload{Event: omen})
	applyMemoryEvent(t, &state, DomainEventTypeSectEventExpired, 6, SectEventResolvedPayload{
		EventID: omen.EventID,
		Summary: ResolvedEventSummary{
			EventID:           omen.EventID,
			Kind:              omen.Kind,
			Outcome:           "expired",
			Summary:           "《巡山异兆》已过期，authority 未替玩家选择结果。",
			ResolvedAtVersion: 6,
		},
	})

	assessmentDisciple := state.Disciples[starterDiscipleID]
	assessmentDisciple.Assessment = DiscipleAssessmentState{
		TargetRank:        IdentityRankInner,
		Passed:            false,
		Score:             78,
		Reason:            "loyalty",
		ResolvedAtVersion: 7,
	}
	applyMemoryEvent(t, &state, DomainEventTypeAssessmentResolved, 7, AssessmentResolvedPayload{
		Disciple:   assessmentDisciple,
		Assessment: assessmentDisciple.Assessment,
	})

	promoted := state.Disciples[starterDiscipleID]
	promoted.Identity = IdentityRankInner
	applyMemoryEvent(t, &state, DomainEventTypeDisciplePromoted, 8, DiscipleChangedPayload{Disciple: promoted})

	disciple := state.Disciples[starterDiscipleID]
	if len(disciple.Memories) != maxDiscipleMemoryEntries {
		t.Fatalf("expected bounded memory list of %d entries, got %+v", maxDiscipleMemoryEntries, disciple.Memories)
	}
	if len(disciple.Relationship) == 0 || len(disciple.Emotion) == 0 {
		t.Fatalf("expected derived relationship and emotion tags, got relationship=%v emotion=%v", disciple.Relationship, disciple.Emotion)
	}
	if len(disciple.RecentSummary) != recentDiscipleExperienceEntries {
		t.Fatalf("expected recent experience summary window of %d, got %+v", recentDiscipleExperienceEntries, disciple.RecentSummary)
	}
	if disciple.RecentSummary[0] != "starter_disciple 晋升为 inner_disciple。" {
		t.Fatalf("expected latest summary to surface newest authority event, got %+v", disciple.RecentSummary)
	}
	if !hasString(disciple.Relationship, "promotion_driven") || !hasString(disciple.Relationship, "battle_proven") {
		t.Fatalf("expected task/cultivation/promotion memories to derive relationship tags, got %+v", disciple.Relationship)
	}
	if !hasString(disciple.Emotion, "ambitious") || !hasString(disciple.Emotion, "alert") {
		t.Fatalf("expected event and advancement memories to derive emotion tags, got %+v", disciple.Emotion)
	}
}

func TestDiscipleMemoryInfluencesMoodTaskAndPromotionAssessment(t *testing.T) {
	baseState := promotionReadyState(true)
	base := baseState.Disciples[starterDiscipleID]

	positive := base
	positive.Memories = []DiscipleMemoryEntry{
		{Kind: "battle_merit", Summary: "完成高风险外务。", Intensity: 3, Tags: []string{"battle_merit", "task_external", "risk_response"}},
		{Kind: "promotion", Summary: "获得宗门晋升。", Intensity: 3, Tags: []string{"promotion", "promotion_path", "thick_support"}},
		{Kind: "duty_fulfilled", Summary: "本月义务如期完成。", Intensity: 2, Tags: []string{"duty_fulfilled"}},
	}
	positive = normalizeDiscipleState(positive)

	negative := base
	negative.Memories = []DiscipleMemoryEntry{
		{Kind: "payroll_delayed", Summary: "月例拖欠。", Intensity: 3, Tags: []string{"payroll_delayed", "support_withheld"}},
		{Kind: "duty_breached", Summary: "义务未达标。", Intensity: 2, Tags: []string{"duty_breached", "setback"}},
		{Kind: "injury", Summary: "外务中负伤。", Intensity: 2, Tags: []string{"injury", "wound_shadow", "pressure"}},
	}
	negative = normalizeDiscipleState(negative)

	positiveMoodS, positiveMoodL := monthlyDiscipleMoodDelta(positive, 1.0, ResourcePolicySaving, true, false, false, 0)
	negativeMoodS, negativeMoodL := monthlyDiscipleMoodDelta(negative, 1.0, ResourcePolicySaving, true, false, false, 0)
	if positiveMoodS <= negativeMoodS || positiveMoodL <= negativeMoodL {
		t.Fatalf("expected positive memory surface to improve monthly mood over negative memories, positive=(%d,%d) negative=(%d,%d)", positiveMoodS, positiveMoodL, negativeMoodS, negativeMoodL)
	}

	combatTask := baseState.Tasks[TaskID("pool-4")]
	if taskRecommendationScore(positive, combatTask) <= taskRecommendationScore(negative, combatTask) {
		t.Fatalf("expected battle merit / wound shadow memories to shift combat willingness, positive=%d negative=%d", taskRecommendationScore(positive, combatTask), taskRecommendationScore(negative, combatTask))
	}

	positiveState := promotionReadyState(true)
	positiveState.Disciples[starterDiscipleID] = positive
	negativeState := promotionReadyState(true)
	negativeState.Disciples[starterDiscipleID] = negative

	positiveAssessment := evaluatePromotionAssessment(positiveState, positive, IdentityRankInner, 1)
	negativeAssessment := evaluatePromotionAssessment(negativeState, negative, IdentityRankInner, 1)
	if positiveAssessment.Score <= negativeAssessment.Score {
		t.Fatalf("expected memory tags to affect promotion assessment score, positive=%+v negative=%+v", positiveAssessment, negativeAssessment)
	}
	if !strings.Contains(negativeAssessment.Reason, "memory_") {
		t.Fatalf("expected negative memory surface to leave promotion reason trace, got %+v", negativeAssessment)
	}
}

func TestDiscipleMemoryReplayMatchesContinuousAuthorityState(t *testing.T) {
	initial := promotionReadyState(true)
	continuous := initial.Clone()
	eventLog := []DomainEvent{
		applyMemoryEvent(t, &continuous, DomainEventTypeTaskCompleted, 1, TaskChangedPayload{Task: TaskState{
			TaskID:              TaskID("task-memory-replay"),
			Kind:                "merchant_commission",
			Type:                TaskTypeExternal,
			Title:               "商会委托",
			Status:              TaskStatusCompleted,
			Evaluation:          TaskEvaluationGood,
			AssignedDiscipleIDs: []DiscipleID{starterDiscipleID},
			ContributionReward:  5,
			Risk:                35,
		}}),
		applyMemoryEvent(t, &continuous, DomainEventTypePayrollPaid, 2, PayrollSettlementPayload{
			MonthIndex:   1,
			DiscipleID:   starterDiscipleID,
			Amount:       monthlyStipendFor(initial.Disciples[starterDiscipleID]),
			ArrearsAfter: 0,
		}),
	}

	savepoint := NewSnapshotReplaySavepoint(initial)
	savepoint.AppendReplay(eventLog)
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore disciple memory replay savepoint: %v", err)
	}
	if canonicalStateJSON(t, restored) != canonicalStateJSON(t, continuous) {
		t.Fatalf("expected replayed disciple memory state to match continuous authority path\ncontinuous=%s\nrestored=%s", canonicalStateJSON(t, continuous), canonicalStateJSON(t, restored))
	}
}

func TestDiscipleMemoryRelationshipStoryFeedbackRestoreOfflineContinuity(t *testing.T) {
	base := promotionReadyState(true)
	base.Meta.SectID = SectID("sect-memory-story-proof")
	base.Meta.OwnerUserID = UserID("player-memory-story-proof")
	base.Runtime.RNG = RNGState{Seed: 20_260_424, Cursor: 0}

	companionID := DiscipleID("disciple-memory-companion")
	companion := externalProofDisciple(
		companionID,
		"memory_companion",
		DiscipleAptitudeState{SpiritRoot: 5, Comprehension: 5, Physique: 6, Mind: 5, Luck: 5},
		RealmMortal,
	)
	base.Disciples[companionID] = companion
	ensureContributionAccounts(&base)
	base.Monthly.Obligations.RequiredDays[companionID] = monthlyDutyRequiredDays(companion)
	recalculateContributionMetrics(&base)
	refreshCultivationDecisionState(&base)

	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete memory continuity receiver")
	}

	var eventLog []DomainEvent
	appendEvents := func(events []DomainEvent) {
		eventLog = append(eventLog, cloneDomainEvents(events)...)
	}

	merchantDispatch := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("pool-2"),
		DiscipleID: starterDiscipleID,
	}, receiver.state.Runtime.Version, "cmd-memory-proof-merchant")
	if merchantDispatch.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected merchant dispatch accepted, got %+v", merchantDispatch.Result)
	}
	appendEvents(merchantDispatch.DomainEvents)

	combatDispatch := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("pool-4"),
		DiscipleID: companionID,
	}, receiver.state.Runtime.Version, "cmd-memory-proof-combat")
	if combatDispatch.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected combat dispatch accepted, got %+v", combatDispatch.Result)
	}
	appendEvents(combatDispatch.DomainEvents)

	dayOne := receiver.advanceTasksOneDay("session-memory-story-proof")
	appendEvents(dayOne.DomainEvents)
	if got := dayOne.Snapshot.State.Tasks[TaskID("pool-2")]; got.Status != TaskStatusCompleted {
		t.Fatalf("expected merchant task completed on day one, got %+v", got)
	}
	if got := dayOne.Snapshot.State.Tasks[TaskID("pool-4")]; got.Status != TaskStatusFailed {
		t.Fatalf("expected combat task failed on day one, got %+v", got)
	}
	if got := dayOne.Snapshot.State.Disciples[companionID]; got.InjuryLevel == 0 || got.Pressure == 0 {
		t.Fatalf("expected combat disciple to gain injury and pressure memories, got %+v", got)
	}

	dayTwo := receiver.advanceTasksOneDay("session-memory-story-proof")
	appendEvents(dayTwo.DomainEvents)
	event := activeGovernanceChoiceEvent(dayTwo.Snapshot.State)
	if event == nil {
		t.Fatalf("expected authority-generated governance choice event by day two, got %+v", dayTwo.Snapshot.State.Events)
	}
	choice := chooseEventOptionCommand(t, receiver, ChooseEventOptionPayload{
		EventID:  event.EventID,
		OptionID: "send_aid",
	}, receiver.state.Runtime.Version, "cmd-memory-proof-event-choice")
	if choice.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected event choice accepted, got %+v", choice.Result)
	}
	appendEvents(choice.DomainEvents)

	assessment := startAssessmentCommandWithID(t, receiver, StartAssessmentPayload{
		DiscipleID: starterDiscipleID,
		TargetRank: IdentityRankInner,
	}, receiver.state.Runtime.Version, "cmd-memory-proof-assessment")
	if assessment.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected memory proof assessment accepted, got %+v", assessment.Result)
	}
	appendEvents(assessment.DomainEvents)

	promotion := promoteDiscipleCommandWithID(t, receiver, PromoteDisciplePayload{
		DiscipleID: starterDiscipleID,
		TargetRank: IdentityRankInner,
	}, assessment.Result.SceneVersion, "cmd-memory-proof-promotion")
	if promotion.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected memory proof promotion accepted, got %+v", promotion.Result)
	}
	appendEvents(promotion.DomainEvents)

	for day := int(receiver.state.Time.CalendarDay) + 1; day <= 30; day++ {
		advance := receiver.advanceTasksOneDay("session-memory-story-proof")
		appendEvents(advance.DomainEvents)
	}
	midState := receiver.state.Clone()
	midEventCut := len(eventLog)
	if midState.Time.CalendarDay != 30 {
		t.Fatalf("expected day-30 mid-session savepoint, got day %d", midState.Time.CalendarDay)
	}

	for day := 31; day <= 60; day++ {
		advance := receiver.advanceTasksOneDay("session-memory-story-proof")
		appendEvents(advance.DomainEvents)
	}

	finalState := receiver.state.Clone()
	continuousSnapshot := receiver.snapshot(base.Meta.OwnerUserID, "session-memory-story-proof")
	if finalState.Time.CalendarDay != 60 {
		t.Fatalf("expected 60-day continuous memory branch, got day %d", finalState.Time.CalendarDay)
	}

	starter := finalState.Disciples[starterDiscipleID]
	if len(starter.Memories) == 0 || len(starter.Relationship) == 0 || len(starter.Emotion) == 0 || len(starter.RecentSummary) == 0 {
		t.Fatalf("expected starter disciple memory surface after task/event/promotion/monthly loop, got %+v", starter)
	}
	if !hasString(starter.Relationship, "promotion_driven") {
		t.Fatalf("expected starter relationship tags to retain promotion path memory, got %+v", starter.Relationship)
	}

	companionFinal := finalState.Disciples[companionID]
	if len(companionFinal.Memories) == 0 || len(companionFinal.Relationship) == 0 || len(companionFinal.Emotion) == 0 || len(companionFinal.RecentSummary) == 0 {
		t.Fatalf("expected companion disciple memory surface after injury/monthly loop, got %+v", companionFinal)
	}
	if companionFinal.InjuryLevel == 0 || companionFinal.HP >= companionFinal.MaxHP {
		t.Fatalf("expected companion to retain authority injury consequences, got %+v", companionFinal)
	}

	feedback := BuildSectEventFeedbackFromEventLog(eventLog)
	feedbackCategories := eventFeedbackCategories(feedback)
	for _, category := range []string{"task_result", "monthly", "promotion", "omen"} {
		if !feedbackCategories[category] {
			t.Fatalf("expected memory proof feedback category %q, got %+v", category, feedbackCategories)
		}
	}

	starterDiary := BuildDiscipleDiaryFromEventLog(eventLog, starterDiscipleID)
	companionDiary := BuildDiscipleDiaryFromEventLog(eventLog, companionID)
	boundedCompanionDiary := BuildDiscipleDiaryFromEventLog(boundDomainEventLog(eventLog), companionID)
	if len(starterDiary) == 0 || len(companionDiary) == 0 {
		t.Fatalf("expected both disciples to retain authority diaries, starter=%+v companion=%+v", starterDiary, companionDiary)
	}
	if len(continuousSnapshot.Diary) == 0 || len(continuousSnapshot.EventSummaries) == 0 {
		t.Fatalf("expected continuous snapshot to surface story feedback, diary=%d feedback=%d", len(continuousSnapshot.Diary), len(continuousSnapshot.EventSummaries))
	}

	savepoint := NewSnapshotReplaySavepoint(midState)
	savepoint.EventLog = cloneDomainEvents(eventLog[:midEventCut])
	savepoint.AppendReplay(eventLog[midEventCut:])
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore memory proof savepoint: %v", err)
	}
	if restoredJSON := canonicalStateJSON(t, restored); restoredJSON != canonicalStateJSON(t, finalState) {
		t.Fatalf("expected restored memory branch to match continuous authority state\ncontinuous=%s\nrestored=%s", canonicalStateJSON(t, finalState), restoredJSON)
	}

	restoredReceiver, ok := NewSectActorWithEventLog(restored, savepoint.EventLog).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete restored memory receiver")
	}
	restoredSnapshot := restoredReceiver.snapshot(base.Meta.OwnerUserID, "session-memory-story-proof-restored")
	if !reflect.DeepEqual(continuousSnapshot.EventSummaries, restoredSnapshot.EventSummaries) {
		t.Fatalf("expected restored event feedback to match continuous branch\ncontinuous=%+v\nrestored=%+v", continuousSnapshot.EventSummaries, restoredSnapshot.EventSummaries)
	}
	if !reflect.DeepEqual(continuousSnapshot.Diary, restoredSnapshot.Diary) {
		t.Fatalf("expected restored starter diary to match continuous branch\ncontinuous=%+v\nrestored=%+v", continuousSnapshot.Diary, restoredSnapshot.Diary)
	}
	if restoredCompanionDiary := BuildDiscipleDiaryFromEventLog(boundDomainEventLog(savepoint.EventLog), companionID); !reflect.DeepEqual(boundedCompanionDiary, restoredCompanionDiary) {
		t.Fatalf("expected restored companion diary to match bounded continuous event log\ncontinuous=%+v\nrestored=%+v", boundedCompanionDiary, restoredCompanionDiary)
	}

	offlineReceiver, ok := NewSectActorWithEventLog(midState, cloneDomainEvents(eventLog[:midEventCut])).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete offline memory receiver")
	}
	advanceSectDays(offlineReceiver, 30, "session-memory-story-proof-offline")
	if offlineJSON := canonicalStateJSON(t, offlineReceiver.state); offlineJSON != canonicalStateJSON(t, finalState) {
		t.Fatalf("expected offline memory catch-up to match continuous authority state\ncontinuous=%s\noffline=%s", canonicalStateJSON(t, finalState), offlineJSON)
	}
	offlineSnapshot := offlineReceiver.snapshot(base.Meta.OwnerUserID, "session-memory-story-proof-offline")
	if !reflect.DeepEqual(continuousSnapshot.EventSummaries, offlineSnapshot.EventSummaries) {
		t.Fatalf("expected offline event feedback to match continuous branch\ncontinuous=%+v\noffline=%+v", continuousSnapshot.EventSummaries, offlineSnapshot.EventSummaries)
	}
	if !reflect.DeepEqual(continuousSnapshot.Diary, offlineSnapshot.Diary) {
		t.Fatalf("expected offline starter diary to match continuous branch\ncontinuous=%+v\noffline=%+v", continuousSnapshot.Diary, offlineSnapshot.Diary)
	}
	if offlineCompanionDiary := BuildDiscipleDiaryFromEventLog(boundDomainEventLog(offlineReceiver.eventLog), companionID); !reflect.DeepEqual(boundedCompanionDiary, offlineCompanionDiary) {
		t.Fatalf("expected offline companion diary to match bounded continuous event log\ncontinuous=%+v\noffline=%+v", boundedCompanionDiary, offlineCompanionDiary)
	}
}

func applyMemoryEvent(t *testing.T, state *SectState, eventType DomainEventType, version Version, payload any) DomainEvent {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s payload: %v", eventType, err)
	}
	event := DomainEvent{
		EventID:   fmt.Sprintf("%s-%d", eventType, version),
		SectID:    state.Meta.SectID,
		Version:   version,
		Type:      eventType,
		Payload:   body,
		CommandID: "cmd-memory-test",
		GameTick:  state.Time.GameTick,
	}
	if err := ApplyEvent(state, event); err != nil {
		t.Fatalf("apply %s: %v", eventType, err)
	}
	return event
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
