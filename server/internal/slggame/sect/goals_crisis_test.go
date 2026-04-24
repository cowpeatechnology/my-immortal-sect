package sect

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSectGoalsResolveThroughAuthorityAdvance(t *testing.T) {
	initial := promotionReadyState(true)
	initial.Meta.SectID = SectID("sect-goals")
	initial.Meta.OwnerUserID = UserID("player-goals")

	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	reserve := reserveCaveCommand(t, receiver, ReserveCavePayload{
		DiscipleID:   starterDiscipleID,
		DurationDays: 2,
	}, receiver.state.Runtime.Version)
	if reserve.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected reserve cave accepted, got %+v", reserve.Result)
	}

	publish := publishTaskCommand(t, receiver, PublishTaskPayload{
		Kind:                 "herb_delivery",
		Type:                 TaskTypeExternal,
		Title:                "药圃采买",
		RequiredProgressDays: 1,
		ContributionReward:   6,
		RewardResources: map[ResourceKind]int64{
			ResourceKindSpiritStone: 4,
		},
	})
	assign := assignTaskCommand(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("task-1"),
		DiscipleID: starterDiscipleID,
	}, publish.Result.SceneVersion)
	if assign.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected assign task accepted, got %+v", assign.Result)
	}

	dayOne := receiver.advanceTasksOneDay("session-goals")
	if !hasDomainEventType(dayOne.DomainEvents, DomainEventTypeSectGoalResolved) {
		t.Fatalf("expected authority goal resolution on day one, got %+v", dayOne.DomainEvents)
	}

	assessment := startAssessmentCommandWithID(t, receiver, StartAssessmentPayload{
		DiscipleID: starterDiscipleID,
		TargetRank: IdentityRankInner,
	}, receiver.state.Runtime.Version, "cmd-goal-assessment")
	if assessment.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected assessment accepted, got %+v", assessment.Result)
	}
	promotion := promoteDiscipleCommandWithID(t, receiver, PromoteDisciplePayload{
		DiscipleID: starterDiscipleID,
		TargetRank: IdentityRankInner,
	}, assessment.Result.SceneVersion, "cmd-goal-promotion")
	if promotion.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected promotion accepted, got %+v", promotion.Result)
	}

	dayTwo := receiver.advanceTasksOneDay("session-goals")
	if !hasDomainEventType(dayTwo.DomainEvents, DomainEventTypeSectGoalResolved) {
		t.Fatalf("expected inner-disciple goal resolution after promotion, got %+v", dayTwo.DomainEvents)
	}

	state := dayTwo.Snapshot.State
	caveGoal := state.Goals.ByID[SectGoalID("goal-cave-routine")]
	externalGoal := state.Goals.ByID[SectGoalID("goal-external-affairs")]
	innerGoal := state.Goals.ByID[SectGoalID("goal-inner-disciple")]
	if caveGoal.Status != SectGoalStatusCompleted || externalGoal.Status != SectGoalStatusCompleted || innerGoal.Status != SectGoalStatusCompleted {
		t.Fatalf("expected cave/external/inner goals completed, got cave=%+v external=%+v inner=%+v", caveGoal, externalGoal, innerGoal)
	}

	caveSummary, ok := resolvedGoalSummary(state, SectGoalID("goal-cave-routine"))
	if !ok || caveSummary.Outcome != "completed" {
		t.Fatalf("expected resolved cave goal summary, got %+v", state.Goals.Resolved)
	}
	externalSummary, ok := resolvedGoalSummary(state, SectGoalID("goal-external-affairs"))
	if !ok || externalSummary.Outcome != "completed" {
		t.Fatalf("expected resolved external goal summary, got %+v", state.Goals.Resolved)
	}
	innerSummary, ok := resolvedGoalSummary(state, SectGoalID("goal-inner-disciple"))
	if !ok || innerSummary.Outcome != "completed" {
		t.Fatalf("expected resolved inner goal summary, got %+v", state.Goals.Resolved)
	}

	if state.Meta.Reputation <= initial.Meta.Reputation {
		t.Fatalf("expected sect goals to increase reputation, initial=%d final=%d", initial.Meta.Reputation, state.Meta.Reputation)
	}
	disciple := state.Disciples[starterDiscipleID]
	if disciple.Satisfaction <= initial.Disciples[starterDiscipleID].Satisfaction {
		t.Fatalf("expected goal rewards to raise disciple satisfaction, initial=%d final=%d", initial.Disciples[starterDiscipleID].Satisfaction, disciple.Satisfaction)
	}
	if !discipleMemoryKindPresent(disciple, "goal_completed") {
		t.Fatalf("expected disciple memory surface to record goal completion, got %+v", disciple.Memories)
	}
}

func TestStableMonthlyGoalFailsThroughAuthoritySettlement(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-goal-monthly-fail"), UserID("player-goal-monthly-fail"), "青崖宗")
	initial.Meta.Reputation = 5
	initial.Productions = map[ProductionID]ProductionJob{}

	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	dayThirty := advanceSectDays(receiver, 30, "session-goal-monthly-fail")
	state := dayThirty.Snapshot.State
	goal := state.Goals.ByID[SectGoalID("goal-stable-monthly")]
	if goal.Status != SectGoalStatusFailed {
		t.Fatalf("expected stable monthly goal to fail under shortage/arrears pressure, got %+v", goal)
	}
	summary, ok := resolvedGoalSummary(state, SectGoalID("goal-stable-monthly"))
	if !ok || summary.Outcome != "failed" {
		t.Fatalf("expected failed stable monthly goal summary, got %+v", state.Goals.Resolved)
	}
	if state.Meta.Reputation >= initial.Meta.Reputation {
		t.Fatalf("expected failed monthly goal to reduce reputation, initial=%d final=%d", initial.Meta.Reputation, state.Meta.Reputation)
	}
	if !discipleMemoryKindPresent(state.Disciples[starterDiscipleID], "goal_failed") {
		t.Fatalf("expected monthly failure to enter disciple memory surface, got %+v", state.Disciples[starterDiscipleID].Memories)
	}
}

func TestSectCrisisChainAdvancesFromOmenToChoiceWithoutSilentResolution(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-crisis-chain"), UserID("player-crisis-chain"), "青崖宗")
	initial.Meta.Reputation = 160
	initial.Resources.Stock[ResourceKindSpiritStone] = 320
	initial.Resources.Stock[ResourceKindFormationMat] = 24
	initial.Resources.Stock[ResourceKindOre] = 120
	initial.Events.Tension = 4
	initial.Productions = map[ProductionID]ProductionJob{}

	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	publish := publishTaskCommand(t, receiver, PublishTaskPayload{
		Kind:                 "perimeter_watch",
		Type:                 TaskTypeCombat,
		Title:                "山门长巡",
		RequiredProgressDays: 10,
		Risk:                 70,
		MaxAssignees:         1,
	})
	assign := assignTaskCommand(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("task-1"),
		DiscipleID: starterDiscipleID,
	}, publish.Result.SceneVersion)
	if assign.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected long-running risk task accepted, got %+v", assign.Result)
	}

	dayOne := receiver.advanceTasksOneDay("session-crisis-chain")
	omen := activeSectCrisisEventByStage(dayOne.Snapshot.State.Events, "omen")
	if omen == nil || omen.Kind != sectCrisisOmenEventKind {
		t.Fatalf("expected first crisis step to be omen, got %+v", dayOne.Snapshot.State.Events)
	}
	if omen.ChainID == "" || !strings.Contains(omen.OmenText, "下一日转为小危机") {
		t.Fatalf("expected omen metadata and foreshadow text, got %+v", omen)
	}

	dayTwo := receiver.advanceTasksOneDay("session-crisis-chain")
	minor := activeSectCrisisEventByStage(dayTwo.Snapshot.State.Events, "minor_crisis")
	if minor == nil || minor.Kind != sectCrisisMinorEventKind {
		t.Fatalf("expected omen to advance into minor crisis, got %+v", dayTwo.Snapshot.State.Events)
	}
	if _, ok := resolvedEventSummaryByKindOutcome(dayTwo.Snapshot.State.Events, sectCrisisOmenEventKind, "advanced:minor_crisis"); !ok {
		t.Fatalf("expected omen resolution summary before minor crisis, got %+v", dayTwo.Snapshot.State.Events.ResolvedEvents)
	}

	dayThree := receiver.advanceTasksOneDay("session-crisis-chain")
	choice := activeSectCrisisEventByStage(dayThree.Snapshot.State.Events, "choice")
	if choice == nil || choice.Kind != sectCrisisChoiceEventKind || len(choice.Options) < 2 {
		t.Fatalf("expected crisis choice stage with authority options, got %+v", dayThree.Snapshot.State.Events)
	}
	if _, ok := resolvedEventSummaryByKindOutcome(dayThree.Snapshot.State.Events, sectCrisisMinorEventKind, "advanced:choice"); !ok {
		t.Fatalf("expected minor crisis to advance into choice, got %+v", dayThree.Snapshot.State.Events.ResolvedEvents)
	}
	if !strings.Contains(choice.OmenText, "不会替你偷偷选项") {
		t.Fatalf("expected choice omen text to fail closed on player silence, got %+v", choice)
	}

	var expiry AdvanceTasksOneDayResponse
	for activeSectCrisisEventByStage(receiver.state.Events, "choice") != nil {
		expiry = receiver.advanceTasksOneDay("session-crisis-chain")
		if receiver.state.Time.CalendarDay > choice.ExpiresAtDay+1 {
			t.Fatalf("crisis choice did not expire by expected day %d", choice.ExpiresAtDay)
		}
	}
	if !hasDomainEventType(expiry.DomainEvents, DomainEventTypeSectEventExpired) {
		t.Fatalf("expected crisis choice expiry event, got %+v", expiry.DomainEvents)
	}
	for _, domainEvent := range expiry.DomainEvents {
		if domainEvent.Type != DomainEventTypeResourceChanged {
			continue
		}
		var payload ResourceChangedPayload
		if err := json.Unmarshal(domainEvent.Payload, &payload); err != nil {
			t.Fatalf("decode resource payload: %v", err)
		}
		if strings.HasPrefix(payload.Reason, "sect_event_option:") {
			t.Fatalf("crisis expiry must not silently apply option side effects, got %+v", payload)
		}
	}
	expiredSummary, ok := resolvedEventSummaryByKindOutcome(receiver.state.Events, sectCrisisChoiceEventKind, "expired")
	if !ok || expiredSummary.Kind != sectCrisisChoiceEventKind {
		t.Fatalf("expected crisis choice to expire without hidden resolution, got %+v", receiver.state.Events.ResolvedEvents)
	}
}

func resolvedGoalSummary(state SectState, goalID SectGoalID) (ResolvedSectGoalSummary, bool) {
	for _, summary := range state.Goals.Resolved {
		if summary.GoalID == goalID {
			return summary, true
		}
	}
	return ResolvedSectGoalSummary{}, false
}

func discipleMemoryKindPresent(disciple DiscipleState, kind string) bool {
	for _, memory := range disciple.Memories {
		if memory.Kind == kind {
			return true
		}
	}
	return false
}

func resolvedEventSummaryByKindOutcome(events SectEventState, kind string, outcome string) (ResolvedEventSummary, bool) {
	for _, summary := range events.ResolvedEvents {
		if summary.Kind != kind || summary.Outcome != outcome {
			continue
		}
		return summary, true
	}
	return ResolvedEventSummary{}, false
}
