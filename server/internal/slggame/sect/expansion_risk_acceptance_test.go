package sect

import (
	"encoding/json"
	"testing"
)

func TestGoldenExpansionRiskOneHundredTwentyDaysRestoreOfflineContinuity(t *testing.T) {
	base := NewInitialSectState(SectID("sect-expansion-risk-120d"), UserID("player-expansion-risk-120d"), "青崖宗")
	base.Runtime.RNG = RNGState{Seed: 20_260_424, Cursor: 0}
	base.Resources.Stock[ResourceKindSpiritStone] = 240
	base.Resources.Stock[ResourceKindOre] = 120
	base.Resources.Stock[ResourceKindBeastMat] = 20
	base.Resources.Stock[ResourceKindFormationMat] = 16
	base.Resources.Stock[ResourceKindHerb] = 32
	base.Meta.Reputation = 96
	base.Events.Tension = 4
	starterBefore := base.Disciples[starterDiscipleID]
	starterBefore.InjuryLevel = 1
	starterBefore.HP = 72
	starterBefore.Pressure = 26
	base.Disciples[starterDiscipleID] = starterBefore

	baselineArtifact := artifactImpactMetricsForState(base)
	baselineFormation := formationImpactMetricsForState(base)
	baselineTaskLimit := taskHallOpenTaskLimit(base)

	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete expansion-risk receiver")
	}

	var eventLog []DomainEvent
	appendEvents := func(events []DomainEvent) {
		eventLog = append(eventLog, cloneDomainEvents(events)...)
	}
	recordAccepted := func(label string, response SubmitCommandResponse) SubmitCommandResponse {
		t.Helper()
		if response.Result.Status != CommandResultStatusAccepted {
			t.Fatalf("expected %s accepted, got %+v", label, response.Result)
		}
		appendEvents(response.DomainEvents)
		return response
	}

	for _, policy := range []SetPolicyPayload{
		{PolicyCategory: "task", PolicyValue: "combat"},
		{PolicyCategory: "resource", PolicyValue: "war_preparation"},
		{PolicyCategory: "recruitment", PolicyValue: "affiliated"},
		{PolicyCategory: "cultivation", PolicyValue: "closed_cultivation"},
	} {
		recordAccepted("set-policy-"+policy.PolicyCategory, setPolicyCommandWithID(
			t,
			receiver,
			policy,
			receiver.state.Runtime.Version,
			"cmd-expansion-risk-policy-"+policy.PolicyCategory,
		))
	}

	mainHall := recordAccepted("build-main-hall", buildBuildingCommandWithID(
		t,
		receiver,
		BuildBuildingPayload{
			DefinitionKey: "main_hall",
			Origin:        TileCoord{Col: 4, Row: 6},
		},
		receiver.state.Runtime.Version,
		"cmd-expansion-risk-build-main-hall",
	))
	recordAccepted("upgrade-main-hall", upgradeBuildingCommandWithID(
		t,
		receiver,
		UpgradeBuildingPayload{BuildingID: BuildingID("building-1")},
		mainHall.Result.SceneVersion,
		"cmd-expansion-risk-upgrade-main-hall",
	))
	gate := recordAccepted("build-gate", buildBuildingCommandWithID(
		t,
		receiver,
		BuildBuildingPayload{
			DefinitionKey: "gate",
			Origin:        TileCoord{Col: 7, Row: 6},
		},
		receiver.state.Runtime.Version,
		"cmd-expansion-risk-build-gate",
	))
	recordAccepted("upgrade-gate", upgradeBuildingCommandWithID(
		t,
		receiver,
		UpgradeBuildingPayload{BuildingID: buildingIDByDefinition(receiver.state, "gate")},
		gate.Result.SceneVersion,
		"cmd-expansion-risk-upgrade-gate",
	))
	recordAccepted("build-warehouse", buildBuildingCommandWithID(
		t,
		receiver,
		BuildBuildingPayload{
			DefinitionKey: "warehouse",
			Origin:        TileCoord{Col: 5, Row: 6},
		},
		receiver.state.Runtime.Version,
		"cmd-expansion-risk-build-warehouse",
	))
	gateID := buildingIDByDefinition(receiver.state, "gate")
	warehouseID := buildingIDByDefinition(receiver.state, "warehouse")
	if gateID == "" || warehouseID == "" {
		t.Fatalf("expected deterministic gate/warehouse IDs, got buildings=%+v", receiver.state.Buildings)
	}

	recruit := recordAccepted("start-recruitment", startRecruitmentCommandWithID(
		t,
		receiver,
		StartRecruitmentPayload{
			CandidateCount:        2,
			InvestmentSpiritStone: 12,
			DurationDays:          5,
		},
		receiver.state.Runtime.Version,
		"cmd-expansion-risk-recruit",
	))
	candidateIDs := sortedCandidateIDs(recruit.Snapshot.State.Admissions.Candidates)
	if len(candidateIDs) != 2 {
		t.Fatalf("expected two deterministic candidates, got %+v", recruit.Snapshot.State.Admissions.Candidates)
	}
	acceptedCandidate := recruit.Snapshot.State.Admissions.Candidates[candidateIDs[0]]
	rejectedCandidate := recruit.Snapshot.State.Admissions.Candidates[candidateIDs[1]]
	accept := recordAccepted("accept-candidate", acceptCandidateCommandWithID(
		t,
		receiver,
		AcceptCandidatePayload{CandidateID: acceptedCandidate.CandidateID},
		recruit.Result.SceneVersion,
		"cmd-expansion-risk-accept-candidate",
	))
	recordAccepted("reject-candidate", rejectCandidateCommandWithID(
		t,
		receiver,
		RejectCandidatePayload{CandidateID: rejectedCandidate.CandidateID},
		accept.Result.SceneVersion,
		"cmd-expansion-risk-reject-candidate",
	))
	acceptedDiscipleID := discipleIDByName(t, receiver.state, acceptedCandidate.Name)

	recordAccepted("assign-task-manager", assignInstitutionManagerCommand(
		t,
		receiver,
		AssignInstitutionManagerPayload{InstitutionID: InstitutionIDTaskHall, DiscipleID: starterDiscipleID},
		receiver.state.Runtime.Version,
		"cmd-expansion-risk-manager-task-hall",
	))
	recordAccepted("assign-medicine-manager", assignInstitutionManagerCommand(
		t,
		receiver,
		AssignInstitutionManagerPayload{InstitutionID: InstitutionIDMedicineHut, DiscipleID: starterDiscipleID},
		receiver.state.Runtime.Version,
		"cmd-expansion-risk-manager-medicine",
	))
	recordAccepted("set-gate-policy", setGatePolicyCommand(
		t,
		receiver,
		SetGatePolicyPayload{EnforcementStrictness: intPtr(2), GuardDiscipleIDs: []DiscipleID{starterDiscipleID}},
		receiver.state.Runtime.Version,
		"cmd-expansion-risk-gate-policy",
	))
	recordAccepted("reserve-cave", reserveCaveCommand(
		t,
		receiver,
		ReserveCavePayload{DiscipleID: starterDiscipleID, DurationDays: 3},
		receiver.state.Runtime.Version,
	))

	for _, craft := range []CraftArtifactPayload{
		{ArtifactType: ArtifactTypeSword},
		{ArtifactType: ArtifactTypeRobe},
		{ArtifactType: ArtifactTypeFarmTool},
		{ArtifactType: ArtifactTypeFormationDisk},
	} {
		recordAccepted("craft-"+string(craft.ArtifactType), receiver.executeCommand(mustCraftArtifactCommand(receiver, craft)))
	}
	swordID := artifactIDForType(receiver.state, ArtifactTypeSword)
	robeID := artifactIDForType(receiver.state, ArtifactTypeRobe)
	toolID := artifactIDForType(receiver.state, ArtifactTypeFarmTool)
	formationDiskID := artifactIDForType(receiver.state, ArtifactTypeFormationDisk)
	if swordID == "" || robeID == "" || toolID == "" || formationDiskID == "" {
		t.Fatalf("expected crafted artifacts for expansion branch, got %+v", receiver.state.Inventory.Artifacts)
	}

	recordAccepted("equip-sword", receiver.executeCommand(mustEquipArtifactCommand(receiver, swordID, starterDiscipleID, "cmd-expansion-risk-equip-sword")))
	recordAccepted("equip-robe", receiver.executeCommand(mustEquipArtifactCommand(receiver, robeID, starterDiscipleID, "cmd-expansion-risk-equip-robe")))
	recordAccepted("equip-tool", receiver.executeCommand(mustEquipArtifactCommand(receiver, toolID, starterDiscipleID, "cmd-expansion-risk-equip-tool")))
	recordAccepted("attach-defense-formation", receiver.executeCommand(mustAttachFormationCommand(
		receiver,
		formationDiskID,
		gateID,
		FormationKindDefense,
		"cmd-expansion-risk-attach-defense-formation",
	)))
	setupDefense := formationDefenseLongtermMetricsForState(receiver.state, gateID)
	if !setupDefense.HasDefenseFormationSource ||
		setupDefense.FormationCount != 1 ||
		setupDefense.RiskMitigation <= baselineFormation.DefenseRiskMitigation ||
		setupDefense.RiskIntensity >= baselineFormation.DefenseRiskIntensity {
		t.Fatalf("expected attached defense formation to enter authority risk summary immediately, baseline=%+v setup=%+v", baselineFormation, setupDefense)
	}

	recordAccepted("start-production", startProductionCommandWithID(
		t,
		receiver,
		StartProductionPayload{
			RecipeID:     RecipeID("formation_refine_mvp"),
			Priority:     78,
			TargetCycles: 1,
		},
		receiver.state.Runtime.Version,
		"cmd-expansion-risk-production",
	))

	merchantTask := recordAccepted("publish-merchant-task", publishTaskCommandWithID(
		t,
		receiver,
		PublishTaskPayload{
			Kind:                 "merchant_commission",
			Title:                "扩张验收外务",
			Description:          "120 日扩张风险门禁的外务收益任务。",
			Priority:             88,
			RequiredProgressDays: 1,
			Risk:                 18,
			MaxAssignees:         1,
			MinIdentity:          IdentityRankOuter,
			MinRealm:             RealmMortal,
			RequiredAptitude:     DiscipleAptitudeState{Physique: 3},
			ContributionReward:   30,
			RewardResources:      map[ResourceKind]int64{ResourceKindSpiritStone: 8, ResourceKindHerb: 2},
		},
		"cmd-expansion-risk-publish-merchant-task",
	))
	merchantTaskID := taskIDByTitle(t, merchantTask.Snapshot.State, "扩张验收外务")
	recordAccepted("assign-merchant-task", assignTaskCommandWithID(
		t,
		receiver,
		AssignDiscipleTaskPayload{TaskID: merchantTaskID, DiscipleID: starterDiscipleID, DiscipleIDs: []DiscipleID{starterDiscipleID}},
		merchantTask.Result.SceneVersion,
		"cmd-expansion-risk-assign-merchant-task",
	))

	var midState SectState
	midEventCut := 0
	cultivationStarted := false
	combatTaskPublished := false
	breakthroughAttempted := false
	assessmentStarted := false
	promotionIssued := false
	fundingTaskPublished := false
	buildingRepaired := false
	governanceChoicesHandled := 0
	crisisChoicesHandled := 0

	for day := 1; day <= 60; day++ {
		advance := receiver.advanceTasksOneDay("session-expansion-risk-120d")
		appendEvents(advance.DomainEvents)

		if !combatTaskPublished && receiver.state.Tasks[merchantTaskID].Status == TaskStatusCompleted {
			combatTask := recordAccepted("publish-combat-task", publishTaskCommandWithID(
				t,
				receiver,
				PublishTaskPayload{
					Kind:                 "demon_scout",
					Type:                 TaskTypeCombat,
					Title:                "扩张验收战斗",
					Description:          "120 日扩张风险门禁的高危战斗任务。",
					Priority:             92,
					RequiredProgressDays: 10,
					Risk:                 88,
					MaxAssignees:         1,
					MinIdentity:          IdentityRankOuter,
					MinRealm:             RealmMortal,
					RequiredAptitude:     DiscipleAptitudeState{Physique: 2},
					ContributionReward:   6,
					RewardResources:      map[ResourceKind]int64{ResourceKindBeastMat: 4},
				},
				"cmd-expansion-risk-publish-combat-task",
			))
			combatTaskID := taskIDByTitle(t, combatTask.Snapshot.State, "扩张验收战斗")
			recordAccepted("assign-combat-task", assignTaskCommandWithID(
				t,
				receiver,
				AssignDiscipleTaskPayload{
					TaskID:      combatTaskID,
					DiscipleID:  acceptedDiscipleID,
					DiscipleIDs: []DiscipleID{acceptedDiscipleID},
				},
				combatTask.Result.SceneVersion,
				"cmd-expansion-risk-assign-combat-task",
			))
			combatTaskPublished = true
		}

		if !cultivationStarted && receiver.state.Tasks[merchantTaskID].Status == TaskStatusCompleted {
			recordAccepted("start-cultivation", startCultivationCommand(
				t,
				receiver,
				StartCultivationPayload{DiscipleID: starterDiscipleID},
				receiver.state.Runtime.Version,
			))
			cultivationStarted = true
		}

		if governanceChoicesHandled == 0 {
			if event := activeGovernanceChoiceEvent(receiver.state); event != nil && len(event.Options) > 0 {
				optionID, ok := firstSelectableSectEventOptionID(receiver.state, *event)
				if !ok {
					t.Fatalf("expected selectable governance event option, got %+v", event)
				}
				recordAccepted("choose-governance-event", chooseEventOptionCommand(
					t,
					receiver,
					ChooseEventOptionPayload{EventID: event.EventID, OptionID: optionID},
					receiver.state.Runtime.Version,
					"cmd-expansion-risk-governance-choice",
				))
				governanceChoicesHandled++
			}
		}

		if crisisChoicesHandled == 0 {
			if event := activeSectCrisisEventByStage(receiver.state.Events, "choice"); event != nil && len(event.Options) > 0 {
				optionID, ok := firstSelectableSectEventOptionID(receiver.state, *event)
				if !ok {
					t.Fatalf("expected selectable crisis event option, got %+v", event)
				}
				recordAccepted("choose-crisis-event", chooseEventOptionCommand(
					t,
					receiver,
					ChooseEventOptionPayload{EventID: event.EventID, OptionID: optionID},
					receiver.state.Runtime.Version,
					"cmd-expansion-risk-crisis-choice",
				))
				crisisChoicesHandled++
			}
		}

		starter := receiver.state.Disciples[starterDiscipleID]
		portent := activeBreakthroughPortentForDisciple(receiver.state.Events, starterDiscipleID)
		if cultivationStarted &&
			!breakthroughAttempted &&
			starter.Realm.ReadyForBreakthrough &&
			starter.InjuryLevel < 3 &&
			portent != nil &&
			portent.Status == SectEventStatusForeshadowed {
			recordAccepted("attempt-breakthrough", attemptBreakthroughCommand(
				t,
				receiver,
				AttemptBreakthroughPayload{DiscipleID: starterDiscipleID},
				receiver.state.Runtime.Version,
			))
			breakthroughAttempted = true
		}

		starter = receiver.state.Disciples[starterDiscipleID]
		if breakthroughAttempted &&
			!assessmentStarted &&
			starter.Identity == IdentityRankOuter &&
			starter.Realm.Stage == RealmQiEntry &&
			receiver.state.Contribution.Accounts[starterDiscipleID].Balance >= promotionContributionCost(IdentityRankOuter, IdentityRankInner) {
			assessment := recordAccepted("start-assessment", startAssessmentCommandWithID(
				t,
				receiver,
				StartAssessmentPayload{DiscipleID: starterDiscipleID, TargetRank: IdentityRankInner},
				receiver.state.Runtime.Version,
				"cmd-expansion-risk-start-assessment",
			))
			assessmentStarted = true
			if assessment.Snapshot.State.Disciples[starterDiscipleID].Assessment.Passed {
				recordAccepted("promote-disciple", promoteDiscipleCommandWithID(
					t,
					receiver,
					PromoteDisciplePayload{DiscipleID: starterDiscipleID, TargetRank: IdentityRankInner},
					assessment.Result.SceneVersion,
					"cmd-expansion-risk-promote-disciple",
				))
				promotionIssued = true
			}
		}

		if day == 30 {
			warehouse := receiver.state.Buildings[warehouseID]
			if warehouse.Phase != "damaged" {
				t.Fatalf("expected warehouse damaged under maintenance pressure by day 30, got %+v", warehouse)
			}
			funding := recordAccepted("publish-funding-task", publishTaskCommandWithID(
				t,
				receiver,
				PublishTaskPayload{
					Kind:                 "maintenance_repair_funding",
					Title:                "扩张验收修缮资金",
					Description:          "120 日扩张风险门禁的修缮资金任务。",
					Priority:             95,
					RequiredProgressDays: 1,
					Risk:                 0,
					MaxAssignees:         1,
					MinIdentity:          IdentityRankOuter,
					MinRealm:             RealmMortal,
					RequiredAptitude:     DiscipleAptitudeState{Physique: 1},
					ContributionReward:   0,
					RewardResources:      map[ResourceKind]int64{ResourceKindSpiritStone: 20},
				},
				"cmd-expansion-risk-publish-funding-task",
			))
			fundingTaskID := taskIDByTitle(t, funding.Snapshot.State, "扩张验收修缮资金")
			recordAccepted("assign-funding-task", assignTaskCommandWithID(
				t,
				receiver,
				AssignDiscipleTaskPayload{
					TaskID:      fundingTaskID,
					DiscipleID:  acceptedDiscipleID,
					DiscipleIDs: []DiscipleID{acceptedDiscipleID},
				},
				funding.Result.SceneVersion,
				"cmd-expansion-risk-assign-funding-task",
			))
			fundingTaskPublished = true
		}

		if fundingTaskPublished && !buildingRepaired {
			warehouse := receiver.state.Buildings[warehouseID]
			repairCost := buildingRepairCostFor(warehouse)
			if warehouse.Phase == "damaged" && CanAfford(receiver.state.Resources, repairCost) {
				recordAccepted("repair-warehouse", repairBuildingCommandWithID(
					t,
					receiver,
					RepairBuildingPayload{BuildingID: warehouseID},
					receiver.state.Runtime.Version,
					"cmd-expansion-risk-repair-warehouse",
				))
				buildingRepaired = true
			}
		}

		if day == 60 {
			midState = receiver.state.Clone()
			midEventCut = len(eventLog)
		}
	}

	if !cultivationStarted || !breakthroughAttempted || !assessmentStarted || !promotionIssued || !buildingRepaired {
		t.Fatalf(
			"expected expansion branch to complete cultivation/promotion/repair milestones before mid-state, cultivation=%t breakthrough=%t assessment=%t promotion=%t repair=%t",
			cultivationStarted,
			breakthroughAttempted,
			assessmentStarted,
			promotionIssued,
			buildingRepaired,
		)
	}
	if midState.Time.CalendarDay < 60 {
		t.Fatalf("expected mid-state at or after day 60, got %+v", midState.Time)
	}

	finalAdvance := advanceSectDaysWithLog(receiver, 60, "session-expansion-risk-120d-continuous", appendEvents)
	finalState := finalAdvance.Snapshot.State
	if finalState.Time.CalendarDay < 120 {
		t.Fatalf("expected at least day 120 after continuation, got %+v", finalState.Time)
	}

	finalArtifact := artifactImpactMetricsForState(finalState)
	finalFormation := formationImpactMetricsForState(finalState)
	finalWarehouse := finalState.Buildings[warehouseID]
	finalGate := finalState.Buildings[gateID]
	starter := finalState.Disciples[starterDiscipleID]
	accepted := finalState.Disciples[acceptedDiscipleID]
	order := finalState.Order
	feedbackCategories := eventFeedbackCategories(BuildSectEventFeedbackFromEventLog(eventLog))
	starterDiary := BuildDiscipleDiaryFromEventLog(eventLog, starterDiscipleID)

	if effectiveSectLevel(finalState) < 2 || finalState.Meta.BuildingLimit < base.Meta.BuildingLimit+1 {
		t.Fatalf("expected sect expansion to raise effective level/building limit, base=%+v final=%+v", base.Meta, finalState.Meta)
	}
	if taskHallOpenTaskLimit(finalState) <= baselineTaskLimit {
		t.Fatalf("expected institution/policy branch to raise open task limit, before=%d after=%d", baselineTaskLimit, taskHallOpenTaskLimit(finalState))
	}
	if finalArtifact.CombatSuccessRate <= baselineArtifact.CombatSuccessRate ||
		finalArtifact.ProductionSuccessRate <= baselineArtifact.ProductionSuccessRate ||
		finalArtifact.CultivationPoints <= baselineArtifact.CultivationPoints ||
		finalArtifact.Weapon == "" ||
		finalArtifact.Robe == "" ||
		finalArtifact.Tool == "" {
		t.Fatalf("expected equipped artifacts to improve authority task/production/cultivation metrics, before=%+v after=%+v", baselineArtifact, finalArtifact)
	}
	if finalFormation.AttachedFormationCount != 1 ||
		finalFormation.CultivationPoints <= baselineFormation.CultivationPoints ||
		finalFormation.DefenseRiskMitigation <= baselineFormation.DefenseRiskMitigation ||
		finalFormation.DefenseRiskIntensity >= baselineFormation.DefenseRiskIntensity {
		t.Fatalf("expected attached defense formation branch to improve risk/cultivation projection, before=%+v after=%+v", baselineFormation, finalFormation)
	}
	if finalWarehouse.MaintenanceDebt == 0 && finalWarehouse.Durability >= 100 && finalWarehouse.Efficiency >= 100 {
		t.Fatalf("expected continued maintenance pressure to remain visible on warehouse, got %+v", finalWarehouse)
	}
	if finalGate.Phase == "" || finalGate.MaxHP <= 0 || finalGate.HP <= 0 {
		t.Fatalf("expected defense gate to persist as authority defense asset, got %+v", finalGate)
	}
	if starter.Identity != IdentityRankInner || starter.Realm.Stage != RealmQiEntry {
		t.Fatalf("expected starter disciple promoted to inner / qi_entry, got %+v", starter)
	}
	if starter.InjuryLevel >= starterBefore.InjuryLevel || starter.HP <= starterBefore.HP {
		t.Fatalf("expected medicine treatment to recover injured starter, before=%+v after=%+v", starterBefore, starter)
	}
	if len(starter.Memories) == 0 || len(starter.Relationship) == 0 || len(starter.RecentSummary) == 0 {
		t.Fatalf("expected starter memory/relationship surface after 120-day loop, got %+v", starter)
	}
	if len(accepted.Memories) == 0 {
		t.Fatalf("expected accepted disciple to accumulate external/combat memories, got %+v", accepted)
	}
	if _, ok := resolvedGoalSummary(finalState, SectGoalID("goal-cave-routine")); !ok {
		t.Fatalf("expected cave goal resolved in final expansion state, got %+v", finalState.Goals)
	}
	if _, ok := resolvedGoalSummary(finalState, SectGoalID("goal-external-affairs")); !ok {
		t.Fatalf("expected external-affairs goal resolved in final expansion state, got %+v", finalState.Goals)
	}
	if _, ok := resolvedGoalSummary(finalState, SectGoalID("goal-inner-disciple")); !ok {
		t.Fatalf("expected inner-disciple goal resolved in final expansion state, got %+v", finalState.Goals)
	}
	if _, ok := resolvedGoalSummary(finalState, SectGoalID("goal-stable-monthly")); !ok {
		t.Fatalf("expected monthly goal to resolve by day 120, got %+v", finalState.Goals)
	}
	if finalState.MonthlyAssessment.LastMonthIndex < 4 || finalState.MonthlyAssessment.Latest == nil {
		t.Fatalf("expected four monthly assessments by day 120, got %+v", finalState.MonthlyAssessment)
	}
	if order.Safety <= 0 || order.Discipline <= 0 || order.InternalStrifeRisk >= 100 {
		t.Fatalf("expected bounded order projection in final state, got %+v", order)
	}
	if len(finalState.Events.ResolvedEvents) == 0 || governanceChoicesHandled == 0 || crisisChoicesHandled == 0 {
		t.Fatalf(
			"expected handled governance/crisis chain evidence in final state, governance=%d crisis=%d resolved=%+v",
			governanceChoicesHandled,
			crisisChoicesHandled,
			finalState.Events.ResolvedEvents,
		)
	}
	if len(starterDiary) == 0 {
		t.Fatalf("expected bounded starter diary from aggregate event log")
	}
	for _, category := range []string{"policy", "task_result", "resource_change", "production", "cultivation", "breakthrough", "monthly", "omen", "promotion"} {
		if !feedbackCategories[category] {
			t.Fatalf("expected expansion-risk feedback category %q, got %+v", category, feedbackCategories)
		}
	}

	seenEvents := domainEventTypes(eventLog)
	for _, eventType := range []DomainEventType{
		DomainEventTypePolicyChanged,
		DomainEventTypeBuildingBuilt,
		DomainEventTypeBuildingUpgraded,
		DomainEventTypeInstitutionChanged,
		DomainEventTypeArtifactChanged,
		DomainEventTypeFormationChanged,
		DomainEventTypeTaskCompleted,
		DomainEventTypeCultivationAdvanced,
		DomainEventTypeBreakthroughSucceeded,
		DomainEventTypeAssessmentResolved,
		DomainEventTypeDisciplePromoted,
		DomainEventTypeOrderChanged,
		DomainEventTypeMonthlyAssessmentResolved,
		DomainEventTypeBuildingDamaged,
		DomainEventTypeBuildingMaintained,
		DomainEventTypeSectEventForeshadowed,
		DomainEventTypeSectEventResolved,
		DomainEventTypeResourceChanged,
	} {
		if !seenEvents[eventType] {
			t.Fatalf("expected aggregate 120-day event log to include %s, got %+v", eventType, seenEvents)
		}
	}

	savepoint := NewSnapshotReplaySavepoint(midState)
	savepoint.AppendReplay(eventLog[midEventCut:])
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore expansion-risk day-60 savepoint: %v", err)
	}
	continuousJSON := canonicalStateJSON(t, finalState)
	if restoredJSON := canonicalStateJSON(t, restored); continuousJSON != restoredJSON {
		t.Fatalf("expected day-60 restore replay to match continuous 120-day authority path\ncontinuous: %s\nrestored: %s", continuousJSON, restoredJSON)
	}

	offlineReceiver, ok := NewSectActor(midState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete offline expansion-risk receiver")
	}
	advanceSectDays(offlineReceiver, 60, "session-expansion-risk-120d-offline")
	if offlineJSON := canonicalStateJSON(t, offlineReceiver.state); continuousJSON != offlineJSON {
		t.Fatalf("expected day-60 offline catch-up to match continuous 120-day authority path\ncontinuous: %s\noffline: %s", continuousJSON, offlineJSON)
	}
}

func mustCraftArtifactCommand(receiver *SectActor, payload CraftArtifactPayload) SubmitCommand {
	return SubmitCommand{
		SessionID: "session-expansion-risk-artifact",
		Command: ClientCommand{
			CmdID:       "cmd-expansion-risk-craft-" + string(payload.ArtifactType),
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeCraftArtifact,
			BaseVersion: receiver.state.Runtime.Version,
			Payload:     mustJSONPayload(payload),
		},
	}
}

func mustEquipArtifactCommand(receiver *SectActor, itemID ItemID, discipleID DiscipleID, cmdID string) SubmitCommand {
	return SubmitCommand{
		SessionID: "session-expansion-risk-artifact",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeEquipArtifact,
			BaseVersion: receiver.state.Runtime.Version,
			Payload: mustJSONPayload(EquipArtifactPayload{
				ItemID:     itemID,
				DiscipleID: discipleID,
			}),
		},
	}
}

func mustAttachFormationCommand(receiver *SectActor, itemID ItemID, buildingID BuildingID, kind FormationKind, cmdID string) SubmitCommand {
	return SubmitCommand{
		SessionID: "session-expansion-risk-formation",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeAttachFormationToBuilding,
			BaseVersion: receiver.state.Runtime.Version,
			Payload: mustJSONPayload(AttachFormationToBuildingPayload{
				BuildingID:     buildingID,
				ArtifactItemID: itemID,
				FormationKind:  kind,
			}),
		},
	}
}

func publishTaskCommandWithID(t *testing.T, receiver *SectActor, payload PublishTaskPayload, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal publish task payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-taskhall",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypePublishTask,
			Payload:     body,
			BaseVersion: receiver.state.Runtime.Version,
		},
	})
}

func firstSelectableSectEventOptionID(state SectState, event SectEvent) (string, bool) {
	if !sectEventRequirementMet(state, event.Requirements) {
		return "", false
	}
	for _, option := range event.Options {
		if !sectEventRequirementMet(state, option.Requirements) {
			continue
		}
		if !resourceDeltaAffordable(state.Resources, option.ResultPreview.ResourceDelta) {
			continue
		}
		return option.OptionID, true
	}
	return "", false
}

func mustJSONPayload(payload any) []byte {
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return data
}

func taskIDByTitle(t *testing.T, state SectState, title string) TaskID {
	t.Helper()
	for _, taskID := range sortedTaskIDs(state.Tasks) {
		if state.Tasks[taskID].Title == title {
			return taskID
		}
	}
	t.Fatalf("expected task title %q in authority state, got %+v", title, state.Tasks)
	return ""
}

func intPtr(value int) *int {
	return &value
}
