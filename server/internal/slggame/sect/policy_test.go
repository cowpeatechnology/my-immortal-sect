package sect

import (
	"encoding/json"
	"testing"
)

func TestSetPolicyChangesAuthorityStateAndEventLogReplay(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-policy"), UserID("player-policy"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	beforeProductionPriority := initial.Tasks[TaskID("pool-3")].Priority
	response := setPolicyCommandWithID(t, receiver, SetPolicyPayload{
		PolicyCategory: string(PolicyCategoryTask),
		PolicyValue:    string(TaskPolicyProduction),
	}, receiver.state.Runtime.Version, "cmd-policy-task-production")
	if response.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected policy change accepted, got %+v", response.Result)
	}
	if !hasDomainEventType(response.DomainEvents, DomainEventTypePolicyChanged) {
		t.Fatalf("expected policy changed domain event, got %+v", response.DomainEvents)
	}
	if !hasClientEventType(response.Result.Events, ClientEventTypePolicyChanged) {
		t.Fatalf("expected policy changed client event, got %+v", response.Result.Events)
	}
	if !hasPatchPath(response.Result.Patch.Ops, "/policies") || !hasPatchPath(response.Result.Patch.Ops, "/tasks/pool-3") {
		t.Fatalf("expected policy and task effect patch ops, got %+v", response.Result.Patch.Ops)
	}
	if got := response.Snapshot.State.Policies.TaskPolicy; got != TaskPolicyProduction {
		t.Fatalf("expected task policy %s, got %s", TaskPolicyProduction, got)
	}
	taskPresentation := response.Snapshot.State.Policies.Presentation.Categories[PolicyCategoryTask]
	if taskPresentation.CurrentValue != string(TaskPolicyProduction) || taskPresentation.Explanation == "" || len(taskPresentation.ImpactSummary) == 0 || len(taskPresentation.Options) < 3 {
		t.Fatalf("expected authority policy presentation for task policy, got %+v", taskPresentation)
	}
	if got := response.Snapshot.State.Tasks[TaskID("pool-3")].Priority; got <= beforeProductionPriority {
		t.Fatalf("expected production policy to raise formation task priority above %d, got %d", beforeProductionPriority, got)
	}

	duplicate := setPolicyCommandWithID(t, receiver, SetPolicyPayload{
		PolicyCategory: string(PolicyCategoryTask),
		PolicyValue:    string(TaskPolicyProduction),
	}, 0, "cmd-policy-task-production")
	if duplicate.Result.Status != CommandResultStatusAccepted || duplicate.Snapshot.SceneVersion != response.Snapshot.SceneVersion {
		t.Fatalf("expected duplicate policy cmd_id to return cached result, first=%+v duplicate=%+v", response.Result, duplicate.Result)
	}

	savepoint := NewSnapshotReplaySavepoint(initial)
	savepoint.AppendReplay(response.DomainEvents)
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore policy savepoint: %v", err)
	}
	if canonicalStateJSON(t, restored) != canonicalStateJSON(t, response.Snapshot.State) {
		t.Fatalf("expected policy replay restore to match command snapshot\nrestored=%+v\nsnapshot=%+v", restored, response.Snapshot.State)
	}
	if !feedbackHasCategory(BuildSectEventFeedbackFromEventLog(response.DomainEvents), "policy") {
		t.Fatalf("expected policy change to enter event feedback, got %+v", BuildSectEventFeedbackFromEventLog(response.DomainEvents))
	}
}

func TestPolicyEffectsCoverRecruitmentProductionCultivationAndMonthlyPressure(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-policy-effects"), UserID("player-policy-effects"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	selective := setPolicyCommandWithID(t, receiver, SetPolicyPayload{
		PolicyCategory: string(PolicyCategoryRecruitment),
		PolicyValue:    string(RecruitmentPolicySelective),
	}, receiver.state.Runtime.Version, "cmd-policy-recruit-selective")
	recruitment := startRecruitmentCommandWithID(t, receiver, StartRecruitmentPayload{}, selective.Result.SceneVersion, "cmd-policy-recruit-start")
	if recruitment.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected recruitment after policy accepted, got %+v", recruitment.Result)
	}
	if got := len(recruitment.Snapshot.State.Admissions.Candidates); got != 2 {
		t.Fatalf("expected selective recruitment to default to two candidates, got %d", got)
	}
	for _, candidate := range recruitment.Snapshot.State.Admissions.Candidates {
		if candidate.Aptitude.SpiritRoot < 10 {
			t.Fatalf("expected selective recruitment quality boost, got %+v", candidate)
		}
	}

	productionActor, ok := NewSectActor(NewInitialSectState(SectID("sect-policy-production"), UserID("player-policy-production"), "青崖宗")).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete production sect actor receiver")
	}
	war := setPolicyCommandWithID(t, productionActor, SetPolicyPayload{
		PolicyCategory: string(PolicyCategoryResource),
		PolicyValue:    string(ResourcePolicyWarPreparation),
	}, productionActor.state.Runtime.Version, "cmd-policy-war")
	if got := war.Snapshot.State.Productions[ProductionID("prod-4-formation-refine")].Priority; got <= initial.Productions[ProductionID("prod-4-formation-refine")].Priority {
		t.Fatalf("expected war preparation policy to raise formation production priority, got %d", got)
	}

	cultivationActor, ok := NewSectActor(NewInitialSectState(SectID("sect-policy-cultivation"), UserID("player-policy-cultivation"), "青崖宗")).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete cultivation sect actor receiver")
	}
	closed := setPolicyCommandWithID(t, cultivationActor, SetPolicyPayload{
		PolicyCategory: string(PolicyCategoryCultivation),
		PolicyValue:    string(CultivationPolicyClosedCultivation),
	}, cultivationActor.state.Runtime.Version, "cmd-policy-cultivation")
	start := startCultivationCommand(t, cultivationActor, StartCultivationPayload{DiscipleID: starterDiscipleID}, closed.Result.SceneVersion)
	if start.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected cultivation start accepted, got %+v", start.Result)
	}
	dayOne := advanceSectDays(cultivationActor, 1, "session-policy-cultivation")
	cultivating := dayOne.Snapshot.State.Disciples[starterDiscipleID]
	if cultivating.Realm.CultivationPoints <= dailyCultivationPoints(cultivating) {
		t.Fatalf("expected closed cultivation policy to increase daily gain, got disciple %+v", cultivating)
	}

	savingSummary := monthlySummaryAfterResourcePolicy(t, ResourcePolicySaving)
	generousSummary := monthlySummaryAfterResourcePolicy(t, ResourcePolicyGenerous)
	if savingSummary.SatisfactionDeltaTotal >= generousSummary.SatisfactionDeltaTotal {
		t.Fatalf("expected saving policy to create stronger monthly satisfaction pressure than generous policy, saving=%+v generous=%+v", savingSummary, generousSummary)
	}
}

func TestPolicyMultidayDeterministicReplayShowsDifferentAuthorityOutcomes(t *testing.T) {
	stable := runPolicyMultidayMetrics(t, "stable", nil)
	focused := runPolicyMultidayMetrics(t, "focused", []SetPolicyPayload{
		{PolicyCategory: string(PolicyCategoryTask), PolicyValue: string(TaskPolicyProduction)},
		{PolicyCategory: string(PolicyCategoryResource), PolicyValue: string(ResourcePolicyWarPreparation)},
		{PolicyCategory: string(PolicyCategoryRecruitment), PolicyValue: string(RecruitmentPolicySelective)},
		{PolicyCategory: string(PolicyCategoryCultivation), PolicyValue: string(CultivationPolicyClosedCultivation)},
	})

	if stable.FormationTaskPriority >= focused.FormationTaskPriority {
		t.Fatalf("expected production task policy to raise formation task priority, stable=%+v focused=%+v", stable, focused)
	}
	if stable.FormationProductionPriority >= focused.FormationProductionPriority {
		t.Fatalf("expected war preparation policy to raise formation production priority, stable=%+v focused=%+v", stable, focused)
	}
	if stable.CandidateCount <= focused.CandidateCount {
		t.Fatalf("expected selective recruitment to reduce candidate count, stable=%+v focused=%+v", stable, focused)
	}
	if stable.AverageCandidateSpiritRoot >= focused.AverageCandidateSpiritRoot {
		t.Fatalf("expected selective recruitment to improve candidate quality, stable=%+v focused=%+v", stable, focused)
	}
	if stable.CultivationPointsAfterThirtyDays >= focused.CultivationPointsAfterThirtyDays {
		t.Fatalf("expected closed cultivation policy to increase multiday cultivation points, stable=%+v focused=%+v", stable, focused)
	}
}

type policyMultidayMetrics struct {
	FormationTaskPriority            int
	FormationProductionPriority      int
	CandidateCount                   int
	AverageCandidateSpiritRoot       int
	CultivationPointsAfterThirtyDays int64
	FinalVersion                     Version
}

func runPolicyMultidayMetrics(t *testing.T, label string, policies []SetPolicyPayload) policyMultidayMetrics {
	t.Helper()
	state := NewInitialSectState(SectID("sect-policy-multiday-"+label), UserID("player-policy-multiday"), "青崖宗")
	receiver, ok := NewSectActor(state).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete policy multiday sect actor receiver")
	}

	for index, policy := range policies {
		response := setPolicyCommandWithID(t, receiver, policy, receiver.state.Runtime.Version, "cmd-policy-multiday-"+label+"-"+string(rune('a'+index)))
		if response.Result.Status != CommandResultStatusAccepted {
			t.Fatalf("expected policy %s accepted, got %+v", policy.PolicyValue, response.Result)
		}
	}

	recruitment := startRecruitmentCommandWithID(t, receiver, StartRecruitmentPayload{}, receiver.state.Runtime.Version, "cmd-policy-multiday-recruit-"+label)
	if recruitment.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected multiday recruitment accepted, got %+v", recruitment.Result)
	}
	start := startCultivationCommand(t, receiver, StartCultivationPayload{DiscipleID: starterDiscipleID}, recruitment.Result.SceneVersion)
	if start.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected multiday cultivation start accepted, got %+v", start.Result)
	}
	final := advanceSectDays(receiver, 30, "session-policy-multiday-"+label)
	finalState := final.Snapshot.State
	return policyMultidayMetrics{
		FormationTaskPriority:            finalState.Tasks[TaskID("pool-3")].Priority,
		FormationProductionPriority:      finalState.Productions[ProductionID("prod-4-formation-refine")].Priority,
		CandidateCount:                   len(finalState.Admissions.Candidates),
		AverageCandidateSpiritRoot:       averageCandidateSpiritRoot(finalState.Admissions.Candidates),
		CultivationPointsAfterThirtyDays: finalState.Disciples[starterDiscipleID].Realm.CultivationPoints,
		FinalVersion:                     final.Snapshot.SceneVersion,
	}
}

func averageCandidateSpiritRoot(candidates map[CandidateID]CandidateState) int {
	if len(candidates) == 0 {
		return 0
	}
	total := 0
	for _, candidate := range candidates {
		total += candidate.Aptitude.SpiritRoot
	}
	return total / len(candidates)
}

func setPolicyCommandWithID(t *testing.T, receiver *SectActor, payload SetPolicyPayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal set policy payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-policy",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeSetPolicy,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func monthlySummaryAfterResourcePolicy(t *testing.T, policy ResourcePolicy) MonthlySettlementSummary {
	t.Helper()
	state := NewInitialSectState(SectID("sect-monthly-policy-"+string(policy)), UserID("player-monthly-policy"), "青崖宗")
	state.Productions = map[ProductionID]ProductionJob{}
	state.Resources.Stock = map[ResourceKind]int64{
		ResourceKindSpiritStone:  0,
		ResourceKindSpiritGrain:  0,
		ResourceKindHerb:         0,
		ResourceKindOre:          0,
		ResourceKindBeastMat:     0,
		ResourceKindFormationMat: 0,
	}
	receiver, ok := NewSectActor(state).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete monthly policy sect actor receiver")
	}
	change := setPolicyCommandWithID(t, receiver, SetPolicyPayload{
		PolicyCategory: string(PolicyCategoryResource),
		PolicyValue:    string(policy),
	}, receiver.state.Runtime.Version, "cmd-policy-monthly-"+string(policy))
	if change.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected resource policy change accepted, got %+v", change.Result)
	}
	return advanceSectDays(receiver, 30, "session-monthly-policy").Snapshot.State.Monthly.LastSettlement
}

func feedbackHasCategory(entries []SectEventFeedbackEntry, category string) bool {
	for _, entry := range entries {
		if entry.Category == category {
			return true
		}
	}
	return false
}
