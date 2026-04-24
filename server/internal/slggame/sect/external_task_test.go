package sect

import "testing"

func TestExternalExploreCombatTaskPoolSettlesAuthorityRewards(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-external-success"), UserID("player-external-success"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	task := receiver.state.Tasks[TaskID("pool-2")]
	if task.Kind != "merchant_commission" || task.Type != TaskTypeExternal {
		t.Fatalf("expected pool-2 to be merchant external task, got %+v", task)
	}
	beforeSpiritStone := receiver.state.Resources.Stock[ResourceKindSpiritStone]

	dispatch := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("pool-2"),
		DiscipleID: starterDiscipleID,
	}, receiver.state.Runtime.Version, "cmd-external-merchant-dispatch")
	if dispatch.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected external dispatch accepted, got %+v", dispatch.Result)
	}

	advance := receiver.advanceTasksOneDay("session-external-success")
	settled := advance.Snapshot.State.Tasks[TaskID("pool-2")]
	if settled.Status != TaskStatusCompleted || settled.SuccessRate <= 0 || settled.Evaluation == "" {
		t.Fatalf("expected authority to settle external task with success rate/evaluation, got %+v", settled)
	}
	if got := advance.Snapshot.State.Resources.Stock[ResourceKindSpiritStone]; got <= beforeSpiritStone {
		t.Fatalf("expected merchant commission to reward spirit stones, before=%d got=%d", beforeSpiritStone, got)
	}
	if got := advance.Snapshot.State.Meta.Reputation; got <= initial.Meta.Reputation {
		t.Fatalf("expected external task to raise authority reputation, got %d", got)
	}
	if got := advance.Snapshot.State.Meta.Relations["merchant_guild"]; got <= 0 {
		t.Fatalf("expected external task to raise merchant relation, got %+v", advance.Snapshot.State.Meta.Relations)
	}
	if !hasDomainEventType(advance.DomainEvents, DomainEventTypeSectMetaChanged) || !hasClientEventType(advance.Events, ClientEventTypeSectMetaChanged) {
		t.Fatalf("expected sect meta change to be emitted to client delta, domain=%+v client=%+v", advance.DomainEvents, advance.Events)
	}
}

func TestExternalCombatFailureAppliesAuthorityConsequences(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-external-failure"), UserID("player-external-failure"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}
	beforeDisciple := receiver.state.Disciples[starterDiscipleID]
	beforeBeastMat := receiver.state.Resources.Stock[ResourceKindBeastMat]

	dispatch := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("pool-4"),
		DiscipleID: starterDiscipleID,
	}, receiver.state.Runtime.Version, "cmd-external-demon-dispatch")
	if dispatch.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected high-risk combat dispatch accepted before settlement, got %+v", dispatch.Result)
	}

	advance := receiver.advanceTasksOneDay("session-external-failure")
	task := advance.Snapshot.State.Tasks[TaskID("pool-4")]
	if task.Status != TaskStatusFailed || task.Evaluation != TaskEvaluationFailed || task.SuccessRate >= 30 {
		t.Fatalf("expected authority failure with failed evaluation, got %+v", task)
	}
	disciple := advance.Snapshot.State.Disciples[starterDiscipleID]
	if disciple.InjuryLevel <= beforeDisciple.InjuryLevel || disciple.Pressure <= beforeDisciple.Pressure || disciple.HP >= beforeDisciple.HP || disciple.Loyalty >= beforeDisciple.Loyalty {
		t.Fatalf("expected failure to injure and pressure disciple through authority events, before=%+v after=%+v", beforeDisciple, disciple)
	}
	if disciple.AssignmentKind != DiscipleAssignmentIdle || disciple.AssignmentTask != nil {
		t.Fatalf("expected failed task to release disciple after authority consequences, got %+v", disciple)
	}
	if got := advance.Snapshot.State.Resources.Stock[ResourceKindBeastMat]; got >= beforeBeastMat {
		t.Fatalf("expected high-risk combat failure to lose beast materials, before=%d got=%d", beforeBeastMat, got)
	}
	if len(advance.Snapshot.State.Events.ActiveEvents) == 0 || advance.Snapshot.State.Events.Tension == 0 {
		t.Fatalf("expected failure to create crisis clue and tension, got %+v", advance.Snapshot.State.Events)
	}
}

func TestTaskSuccessRateUsesDisciplePolicyTeamAndInstitutionSupport(t *testing.T) {
	base := NewInitialSectState(SectID("sect-task-rate"), UserID("player-task-rate"), "青崖宗")
	task := base.Tasks[TaskID("pool-6")]
	task.AssignedDiscipleIDs = []DiscipleID{starterDiscipleID}
	baseRate := taskSuccessRate(base, task)

	boosted := base.Clone()
	adept := newStarterDisciple()
	adept.DiscipleID = DiscipleID("disciple-2")
	adept.Name = "trial_adept"
	adept.Aptitude = DiscipleAptitudeState{SpiritRoot: 9, Comprehension: 9, Physique: 12, Mind: 10, Luck: 8}
	adept.Realm.Stage = RealmQiMiddle
	boosted.Disciples[adept.DiscipleID] = adept
	boosted.Policies.TaskPolicy = TaskPolicyCombat
	task.AssignedDiscipleIDs = []DiscipleID{starterDiscipleID, adept.DiscipleID}
	taskHall := boosted.Institutions.ByID[InstitutionIDTaskHall]
	taskHall.ManagerEffect.EfficiencyBonus = 24
	boosted.Institutions.ByID[InstitutionIDTaskHall] = taskHall

	boostedRate := taskSuccessRate(boosted, task)
	if boostedRate <= baseRate {
		t.Fatalf("expected disciple ability, realm, team, policy, and institution support to improve rate, base=%d boosted=%d", baseRate, boostedRate)
	}
}

func TestExternalRiskRewardNinetyDayRestoreAndOfflineContinuity(t *testing.T) {
	base := NewInitialSectState(SectID("sect-external-risk-restore"), UserID("player-external-risk-restore"), "青崖宗")
	base.Runtime.RNG = RNGState{Seed: 20260424, Cursor: 0}
	base.Disciples[DiscipleID("disciple-explorer")] = externalProofDisciple(
		DiscipleID("disciple-explorer"),
		"proof_explorer",
		DiscipleAptitudeState{SpiritRoot: 14, Comprehension: 14, Physique: 8, Mind: 10, Luck: 12},
		RealmQiMiddle,
	)
	base.Disciples[DiscipleID("disciple-combat")] = externalProofDisciple(
		DiscipleID("disciple-combat"),
		"proof_combat",
		DiscipleAptitudeState{SpiritRoot: 5, Comprehension: 5, Physique: 6, Mind: 5, Luck: 5},
		RealmMortal,
	)
	ensureContributionAccounts(&base)

	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete external restore receiver")
	}
	var eventLog []DomainEvent
	appendEvents := func(events []DomainEvent) {
		eventLog = append(eventLog, events...)
	}

	beforeSpiritStone := receiver.state.Resources.Stock[ResourceKindSpiritStone]
	merchantDispatch := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("pool-2"),
		DiscipleID: starterDiscipleID,
	}, receiver.state.Runtime.Version, "cmd-external-proof-merchant")
	appendEvents(merchantDispatch.DomainEvents)
	if merchantDispatch.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected merchant external dispatch accepted, got %+v", merchantDispatch.Result)
	}
	versionAfterMerchantDispatch := receiver.state.Runtime.Version
	duplicateMerchant := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("pool-2"),
		DiscipleID: starterDiscipleID,
	}, base.Runtime.Version, "cmd-external-proof-merchant")
	if duplicateMerchant.Result.Status != CommandResultStatusAccepted || receiver.state.Runtime.Version != versionAfterMerchantDispatch {
		t.Fatalf("expected duplicate merchant cmd_id to return cached result without side effects, got duplicate=%+v version=%d want=%d", duplicateMerchant.Result, receiver.state.Runtime.Version, versionAfterMerchantDispatch)
	}

	merchantDay := receiver.advanceTasksOneDay("session-external-risk-restore")
	appendEvents(merchantDay.DomainEvents)
	merchantTask := receiver.state.Tasks[TaskID("pool-2")]
	if merchantTask.Status != TaskStatusCompleted || merchantTask.SuccessRate <= 0 || merchantTask.Evaluation == "" {
		t.Fatalf("expected merchant external task result to be authority-settled, got %+v", merchantTask)
	}
	if got := receiver.state.Resources.Stock[ResourceKindSpiritStone]; got != beforeSpiritStone+10 {
		t.Fatalf("expected duplicate dispatch not to duplicate merchant reward, got spirit_stone=%d want=%d", got, beforeSpiritStone+10)
	}
	if receiver.state.Meta.Reputation <= base.Meta.Reputation || receiver.state.Meta.Relations["merchant_guild"] <= 0 {
		t.Fatalf("expected external reward to raise reputation and relation, got meta=%+v", receiver.state.Meta)
	}

	beforeFormationMat := receiver.state.Resources.Stock[ResourceKindFormationMat]
	beforeOre := receiver.state.Resources.Stock[ResourceKindOre]
	exploreDispatch := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("pool-5"),
		DiscipleID: DiscipleID("disciple-explorer"),
	}, receiver.state.Runtime.Version, "cmd-external-proof-explore")
	appendEvents(exploreDispatch.DomainEvents)
	if exploreDispatch.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected exploration dispatch accepted, got %+v", exploreDispatch.Result)
	}
	exploreDays := advanceSectDaysWithLog(receiver, 2, "session-external-risk-restore", appendEvents)
	exploreTask := exploreDays.Snapshot.State.Tasks[TaskID("pool-5")]
	if exploreTask.Status != TaskStatusCompleted || exploreTask.Evaluation == TaskEvaluationFailed {
		t.Fatalf("expected exploration rare-material task to complete, got %+v", exploreTask)
	}
	if got := exploreDays.Snapshot.State.Resources.Stock[ResourceKindFormationMat]; got <= beforeFormationMat {
		t.Fatalf("expected exploration to reward formation materials, before=%d got=%d", beforeFormationMat, got)
	}
	if got := exploreDays.Snapshot.State.Resources.Stock[ResourceKindOre]; got <= beforeOre {
		t.Fatalf("expected exploration to reward ore, before=%d got=%d", beforeOre, got)
	}

	beforeCombatDisciple := receiver.state.Disciples[DiscipleID("disciple-combat")]
	beforeCombatReputation := receiver.state.Meta.Reputation
	beforeTension := receiver.state.Events.Tension
	combatDispatch := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("pool-4"),
		DiscipleID: DiscipleID("disciple-combat"),
	}, receiver.state.Runtime.Version, "cmd-external-proof-combat-risk")
	appendEvents(combatDispatch.DomainEvents)
	if combatDispatch.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected high-risk combat dispatch accepted, got %+v", combatDispatch.Result)
	}
	combatDay := receiver.advanceTasksOneDay("session-external-risk-restore")
	appendEvents(combatDay.DomainEvents)
	combatTask := combatDay.Snapshot.State.Tasks[TaskID("pool-4")]
	if combatTask.Status != TaskStatusFailed || combatTask.Evaluation != TaskEvaluationFailed {
		t.Fatalf("expected high-risk combat to settle failed through authority, got %+v", combatTask)
	}
	combatDisciple := combatDay.Snapshot.State.Disciples[DiscipleID("disciple-combat")]
	if combatDisciple.InjuryLevel <= beforeCombatDisciple.InjuryLevel ||
		combatDisciple.Pressure <= beforeCombatDisciple.Pressure ||
		combatDisciple.HP >= beforeCombatDisciple.HP {
		t.Fatalf("expected combat risk to injure and pressure disciple, before=%+v after=%+v", beforeCombatDisciple, combatDisciple)
	}
	if combatDay.Snapshot.State.Events.Tension <= beforeTension || len(combatDay.Snapshot.State.Events.ActiveEvents) == 0 {
		t.Fatalf("expected combat failure to raise event tension and create crisis clue, before=%d after=%+v", beforeTension, combatDay.Snapshot.State.Events)
	}
	if combatDay.Snapshot.State.Meta.Reputation != beforeCombatReputation {
		t.Fatalf("expected failed combat not to grant reputation, before=%d got=%d", beforeCombatReputation, combatDay.Snapshot.State.Meta.Reputation)
	}

	advanceSectDaysWithLog(receiver, 26, "session-external-risk-restore", appendEvents)
	midState := receiver.state.Clone()
	midEventCut := len(eventLog)
	if midState.Time.CalendarDay != 30 {
		t.Fatalf("expected day-30 mid-session savepoint, got day %d", midState.Time.CalendarDay)
	}

	advanceSectDaysWithLog(receiver, 60, "session-external-risk-restore", appendEvents)
	finalState := receiver.state.Clone()
	if finalState.Time.CalendarDay != 90 {
		t.Fatalf("expected 90-day continuous branch, got day %d", finalState.Time.CalendarDay)
	}
	if finalState.Tasks[TaskID("pool-2")].Status != TaskStatusCompleted ||
		finalState.Tasks[TaskID("pool-5")].Status != TaskStatusCompleted ||
		finalState.Tasks[TaskID("pool-4")].Status != TaskStatusFailed {
		t.Fatalf("expected external/explore/combat outcomes to survive 90-day branch, tasks=%+v", finalState.Tasks)
	}
	if finalState.Disciples[DiscipleID("disciple-combat")].InjuryLevel < combatDisciple.InjuryLevel ||
		finalState.Disciples[DiscipleID("disciple-combat")].Pressure < combatDisciple.Pressure {
		t.Fatalf("expected 90-day branch not to lose injury/pressure outcome, day1=%+v final=%+v", combatDisciple, finalState.Disciples[DiscipleID("disciple-combat")])
	}

	savepoint := NewSnapshotReplaySavepoint(midState)
	savepoint.AppendReplay(eventLog[midEventCut:])
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore external risk snapshot replay: %v", err)
	}
	continuousJSON := canonicalStateJSON(t, finalState)
	if restoredJSON := canonicalStateJSON(t, restored); restoredJSON != continuousJSON {
		t.Fatalf("expected snapshot-after replay to match continuous external branch\ncontinuous: %s\nrestored: %s", continuousJSON, restoredJSON)
	}

	offlineReceiver, ok := NewSectActor(midState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete offline external receiver")
	}
	advanceSectDays(offlineReceiver, 60, "session-external-risk-restore-offline")
	if offlineJSON := canonicalStateJSON(t, offlineReceiver.state); offlineJSON != continuousJSON {
		t.Fatalf("expected offline catch-up to match continuous external branch\ncontinuous: %s\noffline: %s", continuousJSON, offlineJSON)
	}
}

func externalProofDisciple(id DiscipleID, name string, aptitude DiscipleAptitudeState, realm RealmStage) DiscipleState {
	disciple := newStarterDisciple()
	disciple.DiscipleID = id
	disciple.Name = name
	disciple.Aptitude = aptitude
	disciple.Realm.Stage = realm
	return disciple
}
