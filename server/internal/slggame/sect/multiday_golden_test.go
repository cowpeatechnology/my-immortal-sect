package sect

import (
	"encoding/json"
	"testing"
)

func TestGoldenGovernancePromotionCoreSectLoopDeterministicSixtyDays(t *testing.T) {
	base := NewInitialSectState(SectID("sect-golden-60d"), UserID("player-golden-60d"), "青崖宗")
	base.Runtime.RNG = RNGState{Seed: 20_260_423, Cursor: 0}
	initialResources := cloneResourceStock(base.Resources.Stock)

	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	var eventLog []DomainEvent
	appendEvents := func(events []DomainEvent) {
		eventLog = append(eventLog, events...)
	}

	for index, policy := range []SetPolicyPayload{
		{PolicyCategory: "task", PolicyValue: "production"},
		{PolicyCategory: "resource", PolicyValue: "war_preparation"},
		{PolicyCategory: "recruitment", PolicyValue: "selective"},
		{PolicyCategory: "cultivation", PolicyValue: "closed_cultivation"},
	} {
		change := setPolicyCommandWithID(t, receiver, policy, receiver.state.Runtime.Version, "cmd-golden-policy-"+policy.PolicyCategory)
		if change.Result.Status != CommandResultStatusAccepted {
			t.Fatalf("expected policy %d accepted, got %+v", index, change.Result)
		}
		appendEvents(change.DomainEvents)
	}

	buildMainHall := buildBuildingCommandWithID(t, receiver, BuildBuildingPayload{
		DefinitionKey: "main_hall",
		Origin:        TileCoord{Col: 4, Row: 6},
	}, receiver.state.Runtime.Version, "cmd-golden-build-main-hall")
	if buildMainHall.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected main hall build accepted, got %+v", buildMainHall.Result)
	}
	appendEvents(buildMainHall.DomainEvents)

	upgradeMainHall := upgradeBuildingCommandWithID(t, receiver, UpgradeBuildingPayload{BuildingID: BuildingID("building-1")}, buildMainHall.Result.SceneVersion, "cmd-golden-upgrade-main-hall")
	if upgradeMainHall.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected main hall upgrade accepted, got %+v", upgradeMainHall.Result)
	}
	appendEvents(upgradeMainHall.DomainEvents)

	recruit := startRecruitmentCommandWithID(t, receiver, StartRecruitmentPayload{
		CandidateCount:        2,
		InvestmentSpiritStone: 20,
		DurationDays:          5,
	}, receiver.state.Runtime.Version, "cmd-golden-recruit")
	if recruit.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected recruitment accepted, got %+v", recruit.Result)
	}
	appendEvents(recruit.DomainEvents)
	candidateIDs := sortedCandidateIDs(recruit.Snapshot.State.Admissions.Candidates)
	if len(candidateIDs) != 2 {
		t.Fatalf("expected deterministic two-candidate pool, got %+v", recruit.Snapshot.State.Admissions.Candidates)
	}
	acceptedCandidate := recruit.Snapshot.State.Admissions.Candidates[candidateIDs[0]]
	rejectedCandidate := recruit.Snapshot.State.Admissions.Candidates[candidateIDs[1]]

	accept := acceptCandidateCommandWithID(t, receiver, AcceptCandidatePayload{
		CandidateID: acceptedCandidate.CandidateID,
	}, recruit.Result.SceneVersion, "cmd-golden-accept")
	if accept.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected candidate accept accepted, got %+v", accept.Result)
	}
	appendEvents(accept.DomainEvents)
	acceptedDiscipleID := discipleIDByName(t, accept.Snapshot.State, acceptedCandidate.Name)

	reject := rejectCandidateCommandWithID(t, receiver, RejectCandidatePayload{
		CandidateID: rejectedCandidate.CandidateID,
	}, accept.Result.SceneVersion, "cmd-golden-reject")
	if reject.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected candidate reject accepted, got %+v", reject.Result)
	}
	appendEvents(reject.DomainEvents)

	publish := publishTaskCommand(t, receiver, PublishTaskPayload{
		Kind:                 "golden_patrol",
		Title:                "六十日治理验收",
		Description:          "固定种子 golden simulation 的 authority 任务。",
		Priority:             80,
		RequiredProgressDays: 1,
		Risk:                 5,
		MaxAssignees:         1,
		MinIdentity:          IdentityRankOuter,
		MinRealm:             RealmMortal,
		RequiredAptitude:     DiscipleAptitudeState{Physique: 5},
		ContributionReward:   30,
		RewardResources:      map[ResourceKind]int64{ResourceKindSpiritStone: 5, ResourceKindHerb: 2},
	})
	if publish.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected task publish accepted, got %+v", publish.Result)
	}
	appendEvents(publish.DomainEvents)

	assign := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("task-1"),
		DiscipleID: starterDiscipleID,
	}, publish.Result.SceneVersion, "cmd-golden-assign")
	if assign.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected task assignment accepted, got %+v", assign.Result)
	}
	appendEvents(assign.DomainEvents)

	production := startProductionCommandWithID(t, receiver, StartProductionPayload{
		RecipeID:     RecipeID("formation_refine_mvp"),
		Priority:     75,
		TargetCycles: 1,
	}, receiver.state.Runtime.Version, "cmd-golden-production")
	if production.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected production start accepted, got %+v", production.Result)
	}
	appendEvents(production.DomainEvents)
	productionID := production.Snapshot.State.Productions[ProductionID("prod-player-1")].ProductionID

	firstDay := receiver.advanceTasksOneDay("session-golden-60d")
	appendEvents(firstDay.DomainEvents)

	cultivation := startCultivationCommand(t, receiver, StartCultivationPayload{DiscipleID: starterDiscipleID}, receiver.state.Runtime.Version)
	if cultivation.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected cultivation start accepted, got %+v", cultivation.Result)
	}
	appendEvents(cultivation.DomainEvents)

	for day := 2; day <= 4; day++ {
		advance := receiver.advanceTasksOneDay("session-golden-60d")
		appendEvents(advance.DomainEvents)
	}
	event := activeGovernanceChoiceEvent(receiver.state)
	if event == nil {
		t.Fatalf("expected authority governance choice event before player choice, got %+v", receiver.state.Events)
	}
	choice := chooseEventOptionCommand(t, receiver, ChooseEventOptionPayload{
		EventID:  event.EventID,
		OptionID: "send_aid",
	}, receiver.state.Runtime.Version, "cmd-golden-event-choice-send-aid")
	if choice.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected governance event choice accepted, got %+v", choice.Result)
	}
	appendEvents(choice.DomainEvents)

	breakthrough := attemptBreakthroughCommand(t, receiver, AttemptBreakthroughPayload{DiscipleID: starterDiscipleID}, receiver.state.Runtime.Version)
	if breakthrough.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected breakthrough accepted after omen reveal, got %+v", breakthrough.Result)
	}
	appendEvents(breakthrough.DomainEvents)

	assessment := startAssessmentCommandWithID(t, receiver, StartAssessmentPayload{
		DiscipleID: starterDiscipleID,
		TargetRank: IdentityRankInner,
	}, receiver.state.Runtime.Version, "cmd-golden-assessment-inner")
	if assessment.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected assessment accepted, got %+v", assessment.Result)
	}
	appendEvents(assessment.DomainEvents)
	if got := assessment.Snapshot.State.Disciples[starterDiscipleID].Assessment; !got.Passed || got.TargetRank != IdentityRankInner {
		t.Fatalf("expected starter to pass inner assessment, got %+v", got)
	}

	promotion := promoteDiscipleCommandWithID(t, receiver, PromoteDisciplePayload{
		DiscipleID: starterDiscipleID,
		TargetRank: IdentityRankInner,
	}, assessment.Result.SceneVersion, "cmd-golden-promote-inner")
	if promotion.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected promotion accepted, got %+v", promotion.Result)
	}
	appendEvents(promotion.DomainEvents)

	var midState SectState
	midEventCut := 0
	for day := 5; day <= 60; day++ {
		advance := receiver.advanceTasksOneDay("session-golden-60d")
		appendEvents(advance.DomainEvents)
		if day == 30 {
			midState = receiver.state.Clone()
			midEventCut = len(eventLog)
		}
	}

	finalSnapshot := receiver.snapshot(UserID("player-golden-60d"), "session-golden-60d")
	finalState := finalSnapshot.State
	if finalState.Time.CalendarDay < 60 {
		t.Fatalf("expected at least day 60, got %+v", finalState.Time)
	}
	if len(finalState.Disciples) < 2 {
		t.Fatalf("expected accepted disciple in authority roster, got %+v", finalState.Disciples)
	}
	if _, exists := finalState.Admissions.Candidates[rejectedCandidate.CandidateID]; exists {
		t.Fatalf("rejected candidate must not remain in authority candidate pool, got %+v", finalState.Admissions.Candidates)
	}
	task := finalState.Tasks[TaskID("task-1")]
	if task.Status != TaskStatusCompleted {
		t.Fatalf("expected golden task completed by authority tick, got %+v", task)
	}
	if _, ok := finalState.Contribution.Accounts[acceptedDiscipleID]; !ok {
		t.Fatalf("expected accepted disciple account to exist in ledger, got %+v", finalState.Contribution.Accounts)
	}
	starterAccount := finalState.Contribution.Accounts[starterDiscipleID]
	if starterAccount.EarnedTotal < task.ContributionReward || starterAccount.SpentTotal < promotionContributionCost(IdentityRankOuter, IdentityRankInner) {
		t.Fatalf("expected starter promotion contribution earn/spend in ledger, got %+v", starterAccount)
	}
	completedProduction := finalState.Productions[productionID]
	if completedProduction.CompletedCycles < 1 {
		t.Fatalf("expected player production to complete at least one cycle, got %+v", completedProduction)
	}
	starter := finalState.Disciples[starterDiscipleID]
	if starter.Realm.Stage != RealmQiEntry || starter.Identity != IdentityRankInner {
		t.Fatalf("expected starter breakthrough and promotion by day 60, got %+v", starter)
	}
	if got := finalState.Monthly.Obligations.RequiredDays[starterDiscipleID]; got != monthlyDutyRequiredDays(starter) {
		t.Fatalf("expected promoted monthly duty requirement to persist, got %+v", finalState.Monthly.Obligations.RequiredDays)
	}
	if finalState.Monthly.LastSettledMonth < 2 || finalState.Monthly.LastSettlement.MonthIndex < 2 {
		t.Fatalf("expected at least two monthly settlements in golden state, got %+v", finalState.Monthly)
	}
	if !resourceStockChanged(initialResources, finalState.Resources.Stock) {
		t.Fatalf("expected resources to change through task, production, cultivation, and monthly events; initial=%+v final=%+v", initialResources, finalState.Resources.Stock)
	}

	seenEvents := domainEventTypes(eventLog)
	for _, eventType := range []DomainEventType{
		DomainEventTypePolicyChanged,
		DomainEventTypeBuildingBuilt,
		DomainEventTypeBuildingUpgraded,
		DomainEventTypeRecruitmentStarted,
		DomainEventTypeCandidateAccepted,
		DomainEventTypeCandidateRejected,
		DomainEventTypeTaskCompleted,
		DomainEventTypeProductionChanged,
		DomainEventTypeCultivationAdvanced,
		DomainEventTypeBreakthroughSucceeded,
		DomainEventTypeSectEventResolved,
		DomainEventTypeAssessmentResolved,
		DomainEventTypeDisciplePromoted,
		DomainEventTypeSectEventForeshadowed,
		DomainEventTypeMonthAdvanced,
		DomainEventTypeResourceChanged,
	} {
		if !seenEvents[eventType] {
			t.Fatalf("expected golden event log to include %s, got %+v", eventType, seenEvents)
		}
	}
	if !seenEvents[DomainEventTypePayrollPaid] && !seenEvents[DomainEventTypePayrollDelayed] {
		t.Fatalf("expected golden event log to include payroll monthly settlement, got %+v", seenEvents)
	}
	seenFeedback := eventFeedbackCategories(BuildSectEventFeedbackFromEventLog(eventLog))
	for _, category := range []string{"policy", "task_result", "resource_change", "production", "cultivation", "breakthrough", "monthly", "omen", "promotion"} {
		if !seenFeedback[category] {
			t.Fatalf("expected golden player feedback category %q, got %+v", category, seenFeedback)
		}
	}
	if diary := BuildDiscipleDiaryFromEventLog(eventLog, starterDiscipleID); len(diary) == 0 {
		t.Fatalf("expected event_log-derived disciple diary for starter")
	}

	savepoint := NewSnapshotReplaySavepoint(midState)
	savepoint.AppendReplay(eventLog[midEventCut:])
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore golden day-30 savepoint: %v", err)
	}
	continuousJSON := canonicalStateJSON(t, finalState)
	if restoredJSON := canonicalStateJSON(t, restored); continuousJSON != restoredJSON {
		t.Fatalf("expected day-30 restore replay to match continuous 60-day authority path\ncontinuous: %s\nrestored: %s", continuousJSON, restoredJSON)
	}

	offlineReceiver, ok := NewSectActor(midState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete offline catch-up receiver")
	}
	advanceSectDays(offlineReceiver, 30, "session-golden-60d-offline")
	if offlineJSON := canonicalStateJSON(t, offlineReceiver.state); continuousJSON != offlineJSON {
		t.Fatalf("expected day-30 offline catch-up to match continuous 60-day authority path\ncontinuous: %s\noffline: %s", continuousJSON, offlineJSON)
	}
}

func buildBuildingCommandWithID(t *testing.T, receiver *SectActor, payload BuildBuildingPayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal build building payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-golden-build",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeBuildBuilding,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func upgradeBuildingCommandWithID(t *testing.T, receiver *SectActor, payload UpgradeBuildingPayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal upgrade building payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-golden-build",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeUpgradeBuilding,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func cloneResourceStock(stock map[ResourceKind]int64) map[ResourceKind]int64 {
	cloned := make(map[ResourceKind]int64, len(stock))
	for kind, amount := range stock {
		cloned[kind] = amount
	}
	return cloned
}

func resourceStockChanged(initial, final map[ResourceKind]int64) bool {
	if len(initial) != len(final) {
		return true
	}
	for kind, amount := range initial {
		if final[kind] != amount {
			return true
		}
	}
	return false
}

func domainEventTypes(events []DomainEvent) map[DomainEventType]bool {
	seen := map[DomainEventType]bool{}
	for _, event := range events {
		seen[event.Type] = true
	}
	return seen
}

func eventFeedbackCategories(entries []SectEventFeedbackEntry) map[string]bool {
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Category] = true
	}
	return seen
}
