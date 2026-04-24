package sect

import (
	"encoding/json"
	"testing"
)

func TestTreatmentConsumesHerbsAndMedicineCapacity(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-treatment-capacity"), UserID("player-treatment"), "青崖宗")
	initial.Resources.Stock[ResourceKindHerb] = 2

	starter := initial.Disciples[starterDiscipleID]
	starter.InjuryLevel = 2
	starter.HP = 40
	starter.Pressure = 30
	initial.Disciples[starterDiscipleID] = starter

	companionID := DiscipleID("disciple-2")
	companion := externalProofDisciple(
		companionID,
		"disciple_companion",
		DiscipleAptitudeState{SpiritRoot: 6, Comprehension: 6, Physique: 6, Mind: 6, Luck: 6},
		RealmMortal,
	)
	companion.InjuryLevel = 1
	companion.HP = 50
	companion.Pressure = 20
	initial.Disciples[companionID] = companion

	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	assigned := assignInstitutionManagerCommand(t, receiver, AssignInstitutionManagerPayload{
		InstitutionID: InstitutionIDMedicineHut,
		DiscipleID:    starterDiscipleID,
	}, receiver.state.Runtime.Version, "cmd-treatment-manager")
	if assigned.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected medicine hut manager assignment accepted, got %+v", assigned.Result)
	}

	dayOne := receiver.advanceTasksOneDay("session-treatment-capacity")
	if !hasDomainEventType(dayOne.DomainEvents, DomainEventTypeResourceChanged) ||
		!hasDomainEventType(dayOne.DomainEvents, DomainEventTypeDiscipleSatisfactionChanged) {
		t.Fatalf("expected treatment day to emit resource + disciple authority events, got %+v", dayOne.DomainEvents)
	}

	afterStarter := dayOne.Snapshot.State.Disciples[starterDiscipleID]
	if afterStarter.InjuryLevel >= starter.InjuryLevel || afterStarter.HP <= starter.HP || afterStarter.Pressure >= starter.Pressure || !afterStarter.Support.MedicalSupported {
		t.Fatalf("expected treated disciple to recover through authority event path, before=%+v after=%+v", starter, afterStarter)
	}

	afterCompanion := dayOne.Snapshot.State.Disciples[companionID]
	if afterCompanion.InjuryLevel != companion.InjuryLevel || afterCompanion.HP != companion.HP || afterCompanion.Pressure != companion.Pressure {
		t.Fatalf("expected medicine hut capacity 1 to leave second disciple untreated, before=%+v after=%+v", companion, afterCompanion)
	}

	if got := dayOne.Snapshot.State.Resources.Stock[ResourceKindHerb]; got != 0 {
		t.Fatalf("expected severe treatment to consume both herbs, got %d", got)
	}
}

func TestSectOrderRespondsToGateDisciplineAndMitigatesRisk(t *testing.T) {
	runBranch := func(label string, strictness int, guards []DiscipleID) AdvanceTasksOneDayResponse {
		initial := NewInitialSectState(SectID("sect-order-"+label), UserID("player-order"), "青崖宗")
		initial.Meta.Reputation = 48
		initial.Events.Tension = 2
		starter := initial.Disciples[starterDiscipleID]
		starter.Loyalty = 82
		starter.Satisfaction = 44
		initial.Disciples[starterDiscipleID] = starter

		companionID := DiscipleID("disciple-order-guard")
		companion := externalProofDisciple(
			companionID,
			"order_guard",
			DiscipleAptitudeState{SpiritRoot: 7, Comprehension: 6, Physique: 8, Mind: 7, Luck: 6},
			RealmQiEntry,
		)
		companion.Loyalty = 86
		companion.Satisfaction = 52
		initial.Disciples[companionID] = companion
		ensureContributionAccounts(&initial)
		refreshCultivationDecisionState(&initial)

		receiver, ok := NewSectActor(initial).(*SectActor)
		if !ok {
			t.Fatalf("expected concrete sect actor receiver")
		}

		response := setGatePolicyCommand(t, receiver, SetGatePolicyPayload{
			EnforcementStrictness: &strictness,
			GuardDiscipleIDs:      guards,
		}, receiver.state.Runtime.Version, "cmd-order-policy-"+label)
		if response.Result.Status != CommandResultStatusAccepted {
			t.Fatalf("expected gate policy accepted for %s, got %+v", label, response.Result)
		}
		return receiver.advanceTasksOneDay("session-order-" + label)
	}

	weak := runBranch("weak", 0, nil)
	strong := runBranch("strong", 2, []DiscipleID{starterDiscipleID})

	if !hasDomainEventType(strong.DomainEvents, DomainEventTypeOrderChanged) {
		t.Fatalf("expected stronger gate branch to emit order-changed authority event, got %+v", strong.DomainEvents)
	}

	weakOrder := weak.Snapshot.State.Order
	strongOrder := strong.Snapshot.State.Order
	if strongOrder.Safety <= weakOrder.Safety || strongOrder.Discipline <= weakOrder.Discipline {
		t.Fatalf("expected guards + strictness to raise safety and discipline, weak=%+v strong=%+v", weakOrder, strongOrder)
	}
	if strongOrder.InternalStrifeRisk >= weakOrder.InternalStrifeRisk {
		t.Fatalf("expected guards + strictness to lower internal strife risk, weak=%+v strong=%+v", weakOrder, strongOrder)
	}

	weakDefense := buildDefenseRiskProjection(weak.Snapshot.State)
	strongDefense := buildDefenseRiskProjection(strong.Snapshot.State)
	if strongDefense.Intensity >= weakDefense.Intensity {
		t.Fatalf("expected stronger order to mitigate authority defense risk, weak=%+v strong=%+v", weakDefense, strongDefense)
	}
}

func TestMonthlyAssessmentRewardsMomentumAndMemory(t *testing.T) {
	initial := promotionReadyState(true)
	starter := initial.Disciples[starterDiscipleID]
	starter.Realm.Stage = RealmQiMiddle
	starter.Satisfaction = 82
	starter.Loyalty = 88
	initial.Disciples[starterDiscipleID] = starter
	beforeAccount := initial.Contribution.Accounts[starterDiscipleID]
	beforeReputation := initial.Meta.Reputation

	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	dayThirty := advanceSectDays(receiver, 30, "session-monthly-assessment")
	if !hasDomainEventType(dayThirty.DomainEvents, DomainEventTypeMonthlyAssessmentResolved) {
		t.Fatalf("expected monthly assessment resolution event after month advance, got %+v", dayThirty.DomainEvents)
	}

	state := dayThirty.Snapshot.State
	if state.MonthlyAssessment.LastMonthIndex != 1 || state.MonthlyAssessment.Latest == nil {
		t.Fatalf("expected month-one assessment snapshot, got %+v", state.MonthlyAssessment)
	}
	latest := *state.MonthlyAssessment.Latest
	if latest.ChampionDiscipleID == nil || *latest.ChampionDiscipleID != starterDiscipleID {
		t.Fatalf("expected starter disciple to lead first monthly assessment, got %+v", latest)
	}
	if latest.PromotionMomentum <= 0 || latest.RewardContribution <= 0 || latest.RewardReputation <= 0 {
		t.Fatalf("expected monthly assessment to expose contribution/reputation/momentum rewards, got %+v", latest)
	}

	account := state.Contribution.Accounts[starterDiscipleID]
	if account.EarnedTotal <= beforeAccount.EarnedTotal || account.Balance <= beforeAccount.Balance {
		t.Fatalf("expected monthly assessment reward to increase contribution account, before=%+v after=%+v", beforeAccount, account)
	}

	foundReputationReward := false
	for _, event := range dayThirty.DomainEvents {
		if event.Type != DomainEventTypeSectMetaChanged {
			continue
		}
		var payload SectMetaChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode sect meta payload: %v", err)
		}
		if payload.Reason != "monthly_assessment:1" {
			continue
		}
		foundReputationReward = true
		if payload.Meta.Reputation < beforeReputation+latest.RewardReputation {
			t.Fatalf("expected monthly assessment sect-meta reward to raise reputation by at least %d, payload=%+v", latest.RewardReputation, payload)
		}
	}
	if !foundReputationReward {
		t.Fatalf("expected monthly assessment to emit sect-meta reputation reward event, got %+v", dayThirty.DomainEvents)
	}

	disciple := state.Disciples[starterDiscipleID]
	if !hasString(disciple.Relationship, "promotion_driven") || !hasString(disciple.Relationship, "sect_grateful") {
		t.Fatalf("expected monthly assessment memory to influence relationship tags, got %+v", disciple.Relationship)
	}

	foundMemory := false
	for _, memory := range disciple.Memories {
		if memory.Kind != "monthly_assessment_champion" {
			continue
		}
		foundMemory = true
		if memory.SourceEventType != DomainEventTypeMonthlyAssessmentResolved || !hasString(memory.Tags, "promotion_path") {
			t.Fatalf("expected monthly assessment memory to carry authority source + promotion path tags, got %+v", memory)
		}
	}
	if !foundMemory {
		t.Fatalf("expected champion disciple to receive monthly assessment memory, got %+v", disciple.Memories)
	}
}
