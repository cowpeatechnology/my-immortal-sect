package sect

import (
	"encoding/json"
	"testing"
)

func TestInitialSectStateIncludesCoreInstitutions(t *testing.T) {
	state := NewInitialSectState(SectID("sect-institution"), UserID("player-institution"), "青崖宗")

	for _, institutionID := range []InstitutionID{
		InstitutionIDGate,
		InstitutionIDMainHall,
		InstitutionIDTaskHall,
		InstitutionIDTreasury,
		InstitutionIDDormitory,
		InstitutionIDCanteen,
		InstitutionIDMedicineHut,
		InstitutionIDCave,
	} {
		institution, ok := state.Institutions.ByID[institutionID]
		if !ok || !institution.Enabled {
			t.Fatalf("expected enabled institution %s in SectState, got %+v", institutionID, state.Institutions.ByID)
		}
		if len(institution.EffectSummary) == 0 {
			t.Fatalf("expected institution %s to expose authority effect summary, got %+v", institutionID, institution)
		}
	}
	if got := state.Institutions.ByID[InstitutionIDDormitory].Capacity; got < mvpDiscipleRosterCapacity {
		t.Fatalf("expected dormitory to own roster capacity, got %d", got)
	}
	if got := len(state.Institutions.ByID[InstitutionIDCave].CaveSlots); got != 1 {
		t.Fatalf("expected first cave seat in institution state, got %d", got)
	}
}

func TestInstitutionManagerCommandsAffectAuthorityLoops(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-institution-manager"), UserID("player-institution"), "青崖宗")
	starter := initial.Disciples[starterDiscipleID]
	starter.Identity = IdentityRankInner
	starter.Aptitude = DiscipleAptitudeState{SpiritRoot: 12, Comprehension: 13, Physique: 11, Mind: 13, Luck: 10}
	starter.Loyalty = 90
	initial.Disciples[starterDiscipleID] = starter

	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	baseTaskLimit := taskHallOpenTaskLimit(receiver.state)
	taskHall := assignInstitutionManagerCommand(t, receiver, AssignInstitutionManagerPayload{
		InstitutionID: InstitutionIDTaskHall,
		DiscipleID:    starterDiscipleID,
	}, receiver.state.Runtime.Version, "cmd-institution-task-manager")
	if taskHall.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected task hall manager accepted, got %+v", taskHall.Result)
	}
	if !hasDomainEventType(taskHall.DomainEvents, DomainEventTypeInstitutionChanged) || !hasClientEventType(taskHall.Result.Events, ClientEventTypeInstitutionChanged) {
		t.Fatalf("expected manager assignment to emit institution event, got domain=%+v client=%+v", taskHall.DomainEvents, taskHall.Result.Events)
	}
	managedTaskHall := taskHall.Snapshot.State.Institutions.ByID[InstitutionIDTaskHall]
	if managedTaskHall.ManagerDiscipleID == nil || *managedTaskHall.ManagerDiscipleID != starterDiscipleID {
		t.Fatalf("expected task hall manager in snapshot, got %+v", managedTaskHall)
	}
	if managedTaskHall.ManagerEffect.EfficiencyBonus <= 0 || taskHallOpenTaskLimit(taskHall.Snapshot.State) <= baseTaskLimit {
		t.Fatalf("expected manager identity/aptitude/loyalty to improve task hall, before limit=%d after=%+v", baseTaskLimit, managedTaskHall)
	}

	openToVisitors := false
	allowWandering := false
	gate := setGatePolicyCommand(t, receiver, SetGatePolicyPayload{
		OpenToVisitors:            &openToVisitors,
		AllowWanderingCultivators: &allowWandering,
		GuardDiscipleIDs:          []DiscipleID{starterDiscipleID},
	}, taskHall.Result.SceneVersion, "cmd-institution-gate-policy")
	if gate.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected gate policy accepted, got %+v", gate.Result)
	}
	gateState := gate.Snapshot.State.Institutions.ByID[InstitutionIDGate]
	if gateState.GatePolicy.OpenToVisitors || gateState.GatePolicy.AllowWanderingCultivators || len(gateState.GatePolicy.GuardDiscipleIDs) != 1 {
		t.Fatalf("expected gate policy to round-trip through institution state, got %+v", gateState)
	}

	cost := int64(4)
	limit := int64(12)
	enabled := true
	treasury := setExchangeRuleCommand(t, receiver, SetExchangeRulePayload{
		ExchangeItemID:   ExchangeItemID("treasury-spirit-grain"),
		ContributionCost: &cost,
		MonthlyLimit:     &limit,
		Enabled:          &enabled,
	}, gate.Result.SceneVersion, "cmd-institution-exchange-rule")
	if treasury.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected exchange rule accepted, got %+v", treasury.Result)
	}
	rule := treasury.Snapshot.State.Contribution.TreasuryRules[ExchangeItemID("treasury-spirit-grain")]
	if rule.ContributionCost != cost || rule.MonthlyLimit != limit || !rule.Enabled {
		t.Fatalf("expected exchange rule update through institution event, got %+v", rule)
	}
	if !hasPatchPath(treasury.Result.Patch.Ops, "/institutions/by_id/treasury") || !hasPatchPath(treasury.Result.Patch.Ops, "/contribution/treasury_rules/treasury-spirit-grain") {
		t.Fatalf("expected institution and exchange patch ops, got %+v", treasury.Result.Patch.Ops)
	}

	cave := reserveCaveCommand(t, receiver, ReserveCavePayload{DiscipleID: starterDiscipleID, DurationDays: 2}, treasury.Result.SceneVersion)
	if cave.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected cave reserve accepted, got %+v", cave.Result)
	}
	if !hasDomainEventType(cave.DomainEvents, DomainEventTypeInstitutionChanged) {
		t.Fatalf("expected cave reservation to modify institution state, got %+v", cave.DomainEvents)
	}
	caveState := cave.Snapshot.State.Institutions.ByID[InstitutionIDCave]
	if len(caveState.CaveSlots) == 0 || caveState.CaveSlots[0].OccupiedBy == nil || *caveState.CaveSlots[0].OccupiedBy != starterDiscipleID {
		t.Fatalf("expected cave slot reservation in authority institution state, got %+v", caveState)
	}
}

func TestInstitutionDailyEffectsApplyThroughDomainEvents(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-institution-daily"), UserID("player-institution"), "青崖宗")
	starter := initial.Disciples[starterDiscipleID]
	starter.InjuryLevel = 2
	starter.HP = 40
	starter.Pressure = 50
	starter.Satisfaction = 50
	initial.Disciples[starterDiscipleID] = starter

	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	medicine := assignInstitutionManagerCommand(t, receiver, AssignInstitutionManagerPayload{
		InstitutionID: InstitutionIDMedicineHut,
		DiscipleID:    starterDiscipleID,
	}, receiver.state.Runtime.Version, "cmd-institution-medicine-manager")
	if medicine.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected medicine manager accepted, got %+v", medicine.Result)
	}
	canteen := assignInstitutionManagerCommand(t, receiver, AssignInstitutionManagerPayload{
		InstitutionID: InstitutionIDCanteen,
		DiscipleID:    starterDiscipleID,
	}, medicine.Result.SceneVersion, "cmd-institution-canteen-manager")
	if canteen.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected canteen manager accepted, got %+v", canteen.Result)
	}

	dayOne := receiver.advanceTasksOneDay("session-institution-daily")
	if !hasDomainEventType(dayOne.DomainEvents, DomainEventTypeDiscipleSatisfactionChanged) {
		t.Fatalf("expected daily institution effects to emit disciple event, got %+v", dayOne.DomainEvents)
	}
	after := dayOne.Snapshot.State.Disciples[starterDiscipleID]
	if after.InjuryLevel >= starter.InjuryLevel || after.HP <= starter.HP || after.Pressure >= starter.Pressure || after.Satisfaction <= starter.Satisfaction {
		t.Fatalf("expected medicine/canteen manager effects to recover and support disciple, before=%+v after=%+v", starter, after)
	}
}

func TestInstitutionManagementMultidayImpactRestoreOffline(t *testing.T) {
	unmanaged := runInstitutionManagementMultidayBranch(t, "unmanaged", false)
	managed := runInstitutionManagementMultidayBranch(t, "managed", true)

	if managed.TaskLimit <= unmanaged.TaskLimit {
		t.Fatalf("expected task hall manager to raise task limit, unmanaged=%+v managed=%+v", unmanaged, managed)
	}
	if managed.RosterCapacity <= unmanaged.RosterCapacity {
		t.Fatalf("expected dormitory manager to raise roster capacity, unmanaged=%+v managed=%+v", unmanaged, managed)
	}
	if managed.ExchangePressure >= unmanaged.ExchangePressure {
		t.Fatalf("expected treasury manager to lower exchange pressure, unmanaged=%+v managed=%+v", unmanaged, managed)
	}
	if managed.RecoveryPower <= unmanaged.RecoveryPower || managed.FinalInjury >= unmanaged.FinalInjury || managed.FinalHP <= unmanaged.FinalHP {
		t.Fatalf("expected medicine/dormitory managers to improve recovery, unmanaged=%+v managed=%+v", unmanaged, managed)
	}
	if managed.CultivationSupport <= unmanaged.CultivationSupport || managed.CultivationPoints <= unmanaged.CultivationPoints {
		t.Fatalf("expected cave manager/reservation to improve cultivation support, unmanaged=%+v managed=%+v", unmanaged, managed)
	}
	if managed.SatisfactionSupport <= unmanaged.SatisfactionSupport || managed.FinalSatisfaction <= unmanaged.FinalSatisfaction {
		t.Fatalf("expected canteen/dormitory managers to improve satisfaction, unmanaged=%+v managed=%+v", unmanaged, managed)
	}
	if managed.ContributionSpent >= unmanaged.ContributionSpent {
		t.Fatalf("expected treasury manager to lower contribution exchange cost, unmanaged=%+v managed=%+v", unmanaged, managed)
	}
	if managed.FinalSpiritStone >= unmanaged.FinalSpiritStone {
		t.Fatalf("expected cave reservation/resource effects to produce distinct resource metrics, unmanaged=%+v managed=%+v", unmanaged, managed)
	}
	if managed.ManagedInstitutionCount < 5 || managed.InstitutionEventCount < 6 {
		t.Fatalf("expected managed branch to record institution assignments and recent institution events, got %+v", managed)
	}
	if !managed.CaveReserved {
		t.Fatalf("expected managed branch to preserve cave reservation state, got %+v", managed)
	}
	if !managed.RestoreMatchesContinuous || !managed.OfflineMatchesContinuous {
		t.Fatalf("expected managed institution state to match restore/offline branches, got %+v", managed)
	}
}

type institutionManagementMultidayMetrics struct {
	TaskLimit                int
	RosterCapacity           int
	ExchangePressure         int
	RecoveryPower            int
	CultivationSupport       int64
	SatisfactionSupport      int
	CultivationPoints        int64
	FinalHP                  int64
	FinalInjury              int
	FinalSatisfaction        int
	FinalSpiritStone         int64
	FinalSpiritGrain         int64
	ContributionBalance      int64
	ContributionSpent        int64
	ManagedInstitutionCount  int
	InstitutionEventCount    int
	CaveReserved             bool
	RestoreMatchesContinuous bool
	OfflineMatchesContinuous bool
}

func runInstitutionManagementMultidayBranch(t *testing.T, label string, managed bool) institutionManagementMultidayMetrics {
	t.Helper()
	initial := NewInitialSectState(SectID("sect-institution-proof-"+label), UserID("player-institution-proof"), "青崖宗")
	initial.Runtime.RNG = RNGState{Seed: 20_260_423, Cursor: 0}
	starter := initial.Disciples[starterDiscipleID]
	starter.Identity = IdentityRankInner
	starter.Aptitude = DiscipleAptitudeState{SpiritRoot: 50, Comprehension: 50, Physique: 50, Mind: 50, Luck: 20}
	starter.Loyalty = 95
	starter.InjuryLevel = 2
	starter.HP = 40
	starter.Pressure = 55
	starter.Satisfaction = 45
	initial.Disciples[starterDiscipleID] = starter
	account := initial.Contribution.Accounts[starterDiscipleID]
	account.Balance = 100
	account.EarnedTotal = 100
	initial.Contribution.Accounts[starterDiscipleID] = account
	recalculateContributionMetrics(&initial)

	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	var eventLog []DomainEvent
	appendEvents := func(events []DomainEvent) {
		eventLog = append(eventLog, events...)
	}
	accept := func(response SubmitCommandResponse, action string) {
		t.Helper()
		if response.Result.Status != CommandResultStatusAccepted {
			t.Fatalf("expected %s accepted, got %+v", action, response.Result)
		}
		appendEvents(response.DomainEvents)
	}

	if managed {
		for _, institutionID := range []InstitutionID{
			InstitutionIDTaskHall,
			InstitutionIDDormitory,
			InstitutionIDCanteen,
			InstitutionIDMedicineHut,
			InstitutionIDCave,
			InstitutionIDTreasury,
		} {
			accept(assignInstitutionManagerCommand(t, receiver, AssignInstitutionManagerPayload{
				InstitutionID: institutionID,
				DiscipleID:    starterDiscipleID,
			}, receiver.state.Runtime.Version, "cmd-institution-proof-"+label+"-"+string(institutionID)), "assign "+string(institutionID))
		}
	}

	cost := int64(5)
	limit := int64(10)
	enabled := true
	accept(setExchangeRuleCommand(t, receiver, SetExchangeRulePayload{
		ExchangeItemID:   ExchangeItemID("treasury-spirit-grain"),
		ContributionCost: &cost,
		MonthlyLimit:     &limit,
		Enabled:          &enabled,
	}, receiver.state.Runtime.Version, "cmd-institution-proof-"+label+"-exchange-rule"), "set exchange rule")

	accept(startCultivationCommand(t, receiver, StartCultivationPayload{DiscipleID: starterDiscipleID}, receiver.state.Runtime.Version), "start cultivation")
	if managed {
		accept(reserveCaveCommand(t, receiver, ReserveCavePayload{DiscipleID: starterDiscipleID, DurationDays: 2}, receiver.state.Runtime.Version), "reserve cave")
	}
	accept(exchangeContributionCommand(t, receiver, ExchangeContributionItemPayload{
		DiscipleID:     starterDiscipleID,
		ExchangeItemID: ExchangeItemID("treasury-spirit-grain"),
		Quantity:       2,
	}, receiver.state.Runtime.Version), "exchange contribution")

	midState := receiver.state.Clone()
	midEventCut := len(eventLog)
	advanceSectDaysWithLog(receiver, 30, "session-institution-proof-"+label, appendEvents)
	finalState := receiver.state.Clone()

	savepoint := NewSnapshotReplaySavepoint(midState)
	savepoint.AppendReplay(eventLog[midEventCut:])
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore institution proof savepoint: %v", err)
	}
	offlineReceiver, ok := NewSectActor(midState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete offline institution receiver")
	}
	advanceSectDays(offlineReceiver, 30, "session-institution-proof-"+label+"-offline")

	starterFinal := finalState.Disciples[starterDiscipleID]
	treasury := finalState.Institutions.ByID[InstitutionIDTreasury]
	cave := finalState.Institutions.ByID[InstitutionIDCave]
	managedInstitutionCount := 0
	for _, institution := range finalState.Institutions.ByID {
		if institution.ManagerDiscipleID != nil {
			managedInstitutionCount++
		}
	}
	institutionEventCount := 0
	for _, event := range eventLog {
		if event.Type == DomainEventTypeInstitutionChanged {
			institutionEventCount++
		}
	}

	return institutionManagementMultidayMetrics{
		TaskLimit:                taskHallOpenTaskLimit(finalState),
		RosterCapacity:           discipleRosterCapacity(finalState),
		ExchangePressure:         treasury.ExchangePressure,
		RecoveryPower:            institutionRecoveryPower(finalState),
		CultivationSupport:       institutionCultivationSupport(finalState, starterFinal),
		SatisfactionSupport:      institutionSatisfactionSupport(finalState),
		CultivationPoints:        starterFinal.Realm.CultivationPoints,
		FinalHP:                  starterFinal.HP,
		FinalInjury:              starterFinal.InjuryLevel,
		FinalSatisfaction:        starterFinal.Satisfaction,
		FinalSpiritStone:         finalState.Resources.Stock[ResourceKindSpiritStone],
		FinalSpiritGrain:         finalState.Resources.Stock[ResourceKindSpiritGrain],
		ContributionBalance:      finalState.Contribution.Accounts[starterDiscipleID].Balance,
		ContributionSpent:        finalState.Contribution.Accounts[starterDiscipleID].SpentTotal,
		ManagedInstitutionCount:  managedInstitutionCount,
		InstitutionEventCount:    institutionEventCount,
		CaveReserved:             len(cave.CaveSlots) > 0 && cave.CaveSlots[0].OccupiedBy != nil && *cave.CaveSlots[0].OccupiedBy == starterDiscipleID,
		RestoreMatchesContinuous: canonicalStateJSON(t, restored) == canonicalStateJSON(t, finalState),
		OfflineMatchesContinuous: canonicalStateJSON(t, offlineReceiver.state) == canonicalStateJSON(t, finalState),
	}
}

func assignInstitutionManagerCommand(t *testing.T, receiver *SectActor, payload AssignInstitutionManagerPayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal assign institution manager payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-institution",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeAssignInstitutionManager,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func setGatePolicyCommand(t *testing.T, receiver *SectActor, payload SetGatePolicyPayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal set gate policy payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-institution",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeSetGatePolicy,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func setExchangeRuleCommand(t *testing.T, receiver *SectActor, payload SetExchangeRulePayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal set exchange rule payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-institution",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeSetExchangeRule,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}
