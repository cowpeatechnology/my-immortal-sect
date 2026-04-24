package sect

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDiscipleAssessmentAndPromotionAuthorityLoop(t *testing.T) {
	blockedState := promotionReadyState(false)
	blockedReceiver, ok := NewSectActor(blockedState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete blocked promotion sect actor receiver")
	}
	blockedAssessment := startAssessmentCommandWithID(t, blockedReceiver, StartAssessmentPayload{
		DiscipleID: starterDiscipleID,
		TargetRank: IdentityRankInner,
	}, blockedReceiver.state.Runtime.Version, "cmd-assessment-missing-building")
	if blockedAssessment.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected blocked assessment to resolve through authority, got %+v", blockedAssessment.Result)
	}
	assessment := blockedAssessment.Snapshot.State.Disciples[starterDiscipleID].Assessment
	if assessment.Passed || !strings.Contains(assessment.Reason, "building") {
		t.Fatalf("expected missing building condition to fail assessment, got %+v", assessment)
	}
	blockedPromotion := promoteDiscipleCommandWithID(t, blockedReceiver, PromoteDisciplePayload{
		DiscipleID: starterDiscipleID,
		TargetRank: IdentityRankInner,
	}, blockedAssessment.Result.SceneVersion, "cmd-promote-blocked")
	if blockedPromotion.Result.Status != CommandResultStatusRejected ||
		blockedPromotion.Result.Error == nil ||
		blockedPromotion.Result.Error.Code != CommandErrorCodeTaskRequirementNotMet {
		t.Fatalf("expected promotion blocked by failed assessment, got %+v", blockedPromotion.Result)
	}

	state := promotionReadyState(true)
	receiver, ok := NewSectActor(state).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete promotion sect actor receiver")
	}
	before := receiver.state.Disciples[starterDiscipleID]
	beforeStipend := monthlyStipendFor(before)
	beforeDuty := monthlyDutyRequiredDays(before)

	passedAssessment := startAssessmentCommandWithID(t, receiver, StartAssessmentPayload{
		DiscipleID: starterDiscipleID,
		TargetRank: IdentityRankInner,
	}, receiver.state.Runtime.Version, "cmd-assessment-pass")
	if passedAssessment.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected assessment accepted, got %+v", passedAssessment.Result)
	}
	if !hasDomainEventType(passedAssessment.DomainEvents, DomainEventTypeAssessmentResolved) ||
		!hasClientEventType(passedAssessment.Result.Events, ClientEventTypeDiscipleChanged) {
		t.Fatalf("expected assessment resolved disciple delta, got domain=%+v client=%+v", passedAssessment.DomainEvents, passedAssessment.Result.Events)
	}
	passed := passedAssessment.Snapshot.State.Disciples[starterDiscipleID].Assessment
	if !passed.Passed || passed.TargetRank != IdentityRankInner {
		t.Fatalf("expected authority assessment pass for inner promotion, got %+v", passed)
	}

	promote := promoteDiscipleCommandWithID(t, receiver, PromoteDisciplePayload{
		DiscipleID: starterDiscipleID,
		TargetRank: IdentityRankInner,
	}, passedAssessment.Result.SceneVersion, "cmd-promote-inner")
	if promote.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected promotion accepted, got %+v", promote.Result)
	}
	if !hasDomainEventType(promote.DomainEvents, DomainEventTypeContributionSpent) ||
		!hasDomainEventType(promote.DomainEvents, DomainEventTypeDisciplePromoted) ||
		!hasDomainEventType(promote.DomainEvents, DomainEventTypeDiscipleSatisfactionChanged) {
		t.Fatalf("expected contribution spend, promotion, and mood events, got %+v", promote.DomainEvents)
	}

	promoted := promote.Snapshot.State.Disciples[starterDiscipleID]
	if promoted.Identity != IdentityRankInner {
		t.Fatalf("expected disciple promoted to inner, got %+v", promoted)
	}
	account := promote.Snapshot.State.Contribution.Accounts[starterDiscipleID]
	if account.Balance != 20 || account.SpentTotal != 10 {
		t.Fatalf("expected promotion contribution cost spent once, got %+v", account)
	}
	if promoted.Satisfaction <= before.Satisfaction || promoted.Loyalty <= before.Loyalty {
		t.Fatalf("expected promotion to improve satisfaction and loyalty, before=%+v after=%+v", before, promoted)
	}
	if got := monthlyStipendFor(promoted); got <= beforeStipend {
		t.Fatalf("expected promoted identity to increase monthly stipend above %d, got %d", beforeStipend, got)
	}
	if got := monthlyDutyRequiredDays(promoted); got >= beforeDuty {
		t.Fatalf("expected promoted identity to reduce monthly duty below %d, got %d", beforeDuty, got)
	}
	if got := promote.Snapshot.State.Monthly.Obligations.RequiredDays[starterDiscipleID]; got != monthlyDutyRequiredDays(promoted) {
		t.Fatalf("expected promotion event to update monthly duty requirement, got %+v", promote.Snapshot.State.Monthly.Obligations.RequiredDays)
	}

	duplicate := promoteDiscipleCommandWithID(t, receiver, PromoteDisciplePayload{
		DiscipleID: starterDiscipleID,
		TargetRank: IdentityRankInner,
	}, 0, "cmd-promote-inner")
	if duplicate.Result.Status != CommandResultStatusAccepted || duplicate.Snapshot.SceneVersion != promote.Snapshot.SceneVersion {
		t.Fatalf("expected duplicate promote cmd_id to return cached result, first=%+v duplicate=%+v", promote.Result, duplicate.Result)
	}
	if duplicate.Snapshot.State.Contribution.Accounts[starterDiscipleID].SpentTotal != 10 {
		t.Fatalf("duplicate promotion must not spend contribution twice, got %+v", duplicate.Snapshot.State.Contribution.Accounts[starterDiscipleID])
	}

	innerTask := publishTaskCommand(t, receiver, PublishTaskPayload{
		Kind:                 "inner_archive",
		Title:                "内门经卷校录",
		RequiredProgressDays: 1,
		MinIdentity:          IdentityRankInner,
		ContributionReward:   5,
	})
	assignInner := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("task-1"),
		DiscipleID: starterDiscipleID,
	}, innerTask.Result.SceneVersion, "cmd-assign-inner-task")
	if assignInner.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected promoted disciple to meet inner task permission, got %+v", assignInner.Result)
	}

	savepoint := NewSnapshotReplaySavepoint(state)
	savepoint.AppendReplay(passedAssessment.DomainEvents)
	savepoint.AppendReplay(promote.DomainEvents)
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore promotion savepoint: %v", err)
	}
	if canonicalStateJSON(t, restored) != canonicalStateJSON(t, promote.Snapshot.State) {
		t.Fatalf("expected promotion replay restore to match command snapshot\nrestored=%+v\nsnapshot=%+v", restored, promote.Snapshot.State)
	}
	if !feedbackHasCategory(BuildSectEventFeedbackFromEventLog(append(passedAssessment.DomainEvents, promote.DomainEvents...)), "promotion") {
		t.Fatalf("expected promotion events to produce bounded promotion feedback")
	}
}

func TestInnerTaskRejectsOuterDiscipleBeforePromotion(t *testing.T) {
	state := promotionReadyState(true)
	receiver, ok := NewSectActor(state).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete outer permission sect actor receiver")
	}
	innerTask := publishTaskCommand(t, receiver, PublishTaskPayload{
		Kind:                 "inner_archive",
		Title:                "内门经卷校录",
		RequiredProgressDays: 1,
		MinIdentity:          IdentityRankInner,
		ContributionReward:   5,
	})
	assign := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("task-1"),
		DiscipleID: starterDiscipleID,
	}, innerTask.Result.SceneVersion, "cmd-assign-inner-task-before-promotion")
	if assign.Result.Status != CommandResultStatusRejected ||
		assign.Result.Error == nil ||
		assign.Result.Error.Code != CommandErrorCodeTaskRequirementNotMet {
		t.Fatalf("expected outer disciple rejected from inner task, got %+v", assign.Result)
	}
}

func promotionReadyState(withRequiredBuilding bool) SectState {
	state := NewInitialSectState(SectID("sect-promotion"), UserID("player-promotion"), "青崖宗")
	disciple := state.Disciples[starterDiscipleID]
	disciple.Realm.Stage = RealmQiEntry
	disciple.Realm.CultivationPoints = cultivationThresholdForStage(RealmMortal)
	disciple.Satisfaction = 70
	disciple.Loyalty = 70
	disciple.InjuryLevel = 0
	disciple.HP = disciple.MaxHP
	state.Disciples[starterDiscipleID] = disciple
	account := state.Contribution.Accounts[starterDiscipleID]
	account.Balance = 30
	account.EarnedTotal = 30
	state.Contribution.Accounts[starterDiscipleID] = account
	state.Monthly.Obligations.CompletedDays[starterDiscipleID] = 1
	state.Monthly.Obligations.RequiredDays[starterDiscipleID] = monthlyDutyRequiredDays(disciple)
	if withRequiredBuilding {
		state.Buildings[BuildingID("building-main-hall")] = newBuildingState(BuildingID("building-main-hall"), "main_hall", 2, TileCoord{})
	}
	recalculateContributionMetrics(&state)
	refreshCultivationDecisionState(&state)
	return state
}

func startAssessmentCommandWithID(t *testing.T, receiver *SectActor, payload StartAssessmentPayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal assessment payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-promotion",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeStartAssessment,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func promoteDiscipleCommandWithID(t *testing.T, receiver *SectActor, payload PromoteDisciplePayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal promote disciple payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-promotion",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypePromoteDisciple,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}
