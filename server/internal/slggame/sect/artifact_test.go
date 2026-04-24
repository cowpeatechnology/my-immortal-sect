package sect

import (
	"encoding/json"
	"testing"
)

func TestArtifactCraftEquipUnequipRepairAuthorityLoop(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-artifact"), UserID("player-artifact"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	craftPayload, err := json.Marshal(CraftArtifactPayload{ArtifactType: ArtifactTypeSword})
	if err != nil {
		t.Fatalf("marshal craft payload: %v", err)
	}
	craftCommand := SubmitCommand{
		SessionID: "session-artifact",
		Command: ClientCommand{
			CmdID:   "cmd-artifact-craft-1",
			UserID:  "player-artifact",
			SectID:  "sect-artifact",
			Type:    CommandTypeCraftArtifact,
			Payload: craftPayload,
		},
	}
	crafted := receiver.executeCommand(craftCommand)
	if crafted.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected craft accepted, got %+v", crafted.Result)
	}
	if crafted.Result.SceneVersion != 2 {
		t.Fatalf("expected craft to emit resource+artifact events, got version %d", crafted.Result.SceneVersion)
	}
	if crafted.Snapshot.State.Resources.Stock[ResourceKindOre] != 30 {
		t.Fatalf("expected ore cost applied, got %d", crafted.Snapshot.State.Resources.Stock[ResourceKindOre])
	}
	artifact := crafted.Snapshot.State.Inventory.Artifacts[ItemID("artifact-1")]
	if artifact.Type != ArtifactTypeSword || artifact.Quality != 1 || artifact.Durability != 100 || artifact.Stats["combat"] <= 0 {
		t.Fatalf("expected fixed sword artifact state, got %+v", artifact)
	}

	duplicate := receiver.executeCommand(craftCommand)
	if duplicate.Result.SceneVersion != crafted.Result.SceneVersion {
		t.Fatalf("expected duplicate cmd_id to return cached version %d, got %d", crafted.Result.SceneVersion, duplicate.Result.SceneVersion)
	}
	if len(duplicate.Snapshot.State.Inventory.Artifacts) != 1 {
		t.Fatalf("expected duplicate craft to remain idempotent, got artifacts=%d", len(duplicate.Snapshot.State.Inventory.Artifacts))
	}

	equipPayload, err := json.Marshal(EquipArtifactPayload{ItemID: artifact.ItemID, DiscipleID: starterDiscipleID})
	if err != nil {
		t.Fatalf("marshal equip payload: %v", err)
	}
	equipped := receiver.executeCommand(SubmitCommand{
		SessionID: "session-artifact",
		Command: ClientCommand{
			CmdID:   "cmd-artifact-equip-1",
			UserID:  "player-artifact",
			SectID:  "sect-artifact",
			Type:    CommandTypeEquipArtifact,
			Payload: equipPayload,
		},
	})
	if equipped.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected equip accepted, got %+v", equipped.Result)
	}
	equippedArtifact := equipped.Snapshot.State.Inventory.Artifacts[artifact.ItemID]
	if equippedArtifact.BoundDiscipleID != starterDiscipleID {
		t.Fatalf("expected artifact bound to starter, got %+v", equippedArtifact)
	}
	if equipped.Snapshot.State.Disciples[starterDiscipleID].Equipment.Weapon != artifact.ItemID {
		t.Fatalf("expected disciple weapon slot to point at artifact, got %+v", equipped.Snapshot.State.Disciples[starterDiscipleID].Equipment)
	}

	unequipPayload, err := json.Marshal(UnequipArtifactPayload{ItemID: artifact.ItemID})
	if err != nil {
		t.Fatalf("marshal unequip payload: %v", err)
	}
	unequipped := receiver.executeCommand(SubmitCommand{
		SessionID: "session-artifact",
		Command: ClientCommand{
			CmdID:   "cmd-artifact-unequip-1",
			UserID:  "player-artifact",
			SectID:  "sect-artifact",
			Type:    CommandTypeUnequipArtifact,
			Payload: unequipPayload,
		},
	})
	if unequipped.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected unequip accepted, got %+v", unequipped.Result)
	}
	if unequipped.Snapshot.State.Inventory.Artifacts[artifact.ItemID].BoundDiscipleID != "" {
		t.Fatalf("expected artifact unbound after unequip, got %+v", unequipped.Snapshot.State.Inventory.Artifacts[artifact.ItemID])
	}
	if unequipped.Snapshot.State.Disciples[starterDiscipleID].Equipment.Weapon != "" {
		t.Fatalf("expected weapon slot cleared, got %+v", unequipped.Snapshot.State.Disciples[starterDiscipleID].Equipment)
	}

	damaged := receiver.state.Inventory.Artifacts[artifact.ItemID]
	damaged.Durability = 35
	receiver.state.Inventory.Artifacts[artifact.ItemID] = damaged
	repairPayload, err := json.Marshal(RepairArtifactPayload{ItemID: artifact.ItemID})
	if err != nil {
		t.Fatalf("marshal repair payload: %v", err)
	}
	repaired := receiver.executeCommand(SubmitCommand{
		SessionID: "session-artifact",
		Command: ClientCommand{
			CmdID:   "cmd-artifact-repair-1",
			UserID:  "player-artifact",
			SectID:  "sect-artifact",
			Type:    CommandTypeRepairArtifact,
			Payload: repairPayload,
		},
	})
	if repaired.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected repair accepted, got %+v", repaired.Result)
	}
	if repaired.Snapshot.State.Inventory.Artifacts[artifact.ItemID].Durability != repaired.Snapshot.State.Inventory.Artifacts[artifact.ItemID].MaxDurability {
		t.Fatalf("expected artifact durability restored, got %+v", repaired.Snapshot.State.Inventory.Artifacts[artifact.ItemID])
	}

	savepoint := NewSnapshotReplaySavepoint(initial)
	savepoint.AppendReplay(crafted.DomainEvents)
	savepoint.AppendReplay(equipped.DomainEvents)
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore artifact replay: %v", err)
	}
	if restored.Inventory.Artifacts[artifact.ItemID].BoundDiscipleID != starterDiscipleID {
		t.Fatalf("expected replay restore to preserve equipped artifact, got %+v", restored.Inventory.Artifacts[artifact.ItemID])
	}
	if restored.Disciples[starterDiscipleID].Equipment.Weapon != artifact.ItemID {
		t.Fatalf("expected replay restore to preserve disciple equipment, got %+v", restored.Disciples[starterDiscipleID].Equipment)
	}
}

func TestArtifactCommandsValidateBuildingsResourcesDisciplesAndDurability(t *testing.T) {
	state := NewInitialSectState(SectID("sect-artifact-rules"), UserID("player-artifact"), "青崖宗")
	state.Institutions.ByID[InstitutionIDMedicineHut] = InstitutionLoopState{InstitutionID: InstitutionIDMedicineHut, Enabled: false}
	receiver := NewSectActor(state).(*SectActor)

	furnacePayload, err := json.Marshal(CraftArtifactPayload{ArtifactType: ArtifactTypeAlchemyFurnace})
	if err != nil {
		t.Fatalf("marshal furnace payload: %v", err)
	}
	rejectedBuilding := receiver.executeCommand(SubmitCommand{
		SessionID: "session-artifact",
		Command:   ClientCommand{CmdID: "cmd-artifact-craft-blocked", UserID: "player-artifact", SectID: "sect-artifact-rules", Type: CommandTypeCraftArtifact, Payload: furnacePayload},
	})
	if rejectedBuilding.Result.Status != CommandResultStatusRejected || rejectedBuilding.Result.Error == nil || rejectedBuilding.Result.Error.Code != CommandErrorCodeTaskRequirementNotMet {
		t.Fatalf("expected craft to require enabled institution, got %+v", rejectedBuilding.Result)
	}

	state = NewInitialSectState(SectID("sect-artifact-poor"), UserID("player-artifact"), "青崖宗")
	state.Resources.Stock[ResourceKindOre] = 0
	receiver = NewSectActor(state).(*SectActor)
	swordPayload, err := json.Marshal(CraftArtifactPayload{ArtifactType: ArtifactTypeSword})
	if err != nil {
		t.Fatalf("marshal sword payload: %v", err)
	}
	rejectedResources := receiver.executeCommand(SubmitCommand{
		SessionID: "session-artifact",
		Command:   ClientCommand{CmdID: "cmd-artifact-craft-poor", UserID: "player-artifact", SectID: "sect-artifact-poor", Type: CommandTypeCraftArtifact, Payload: swordPayload},
	})
	if rejectedResources.Result.Status != CommandResultStatusRejected || rejectedResources.Result.Error == nil || rejectedResources.Result.Error.Code != CommandErrorCodeInsufficientResource {
		t.Fatalf("expected craft to require resources, got %+v", rejectedResources.Result)
	}

	state = NewInitialSectState(SectID("sect-artifact-equip"), UserID("player-artifact"), "青崖宗")
	state.Inventory.Artifacts[ItemID("artifact-broken")] = ArtifactState{
		ItemID:        ItemID("artifact-broken"),
		Type:          ArtifactTypeSword,
		Quality:       1,
		Durability:    0,
		MaxDurability: 100,
		Stats:         map[string]int{"combat": 10},
	}
	receiver = NewSectActor(state).(*SectActor)
	equipPayload, err := json.Marshal(EquipArtifactPayload{ItemID: ItemID("artifact-broken"), DiscipleID: DiscipleID("missing-disciple")})
	if err != nil {
		t.Fatalf("marshal equip payload: %v", err)
	}
	rejectedDurability := receiver.executeCommand(SubmitCommand{
		SessionID: "session-artifact",
		Command:   ClientCommand{CmdID: "cmd-artifact-equip-broken", UserID: "player-artifact", SectID: "sect-artifact-equip", Type: CommandTypeEquipArtifact, Payload: equipPayload},
	})
	if rejectedDurability.Result.Status != CommandResultStatusRejected || rejectedDurability.Result.Error == nil || rejectedDurability.Result.Error.Code != CommandErrorCodeTaskRequirementNotMet {
		t.Fatalf("expected equip to reject zero durability before disciple binding, got %+v", rejectedDurability.Result)
	}
}

func TestEquippedArtifactsAffectTaskSuccessRiskAndCultivation(t *testing.T) {
	state := NewInitialSectState(SectID("sect-artifact-effects"), UserID("player-artifact"), "青崖宗")
	task := TaskState{
		TaskID:              TaskID("artifact-combat"),
		Kind:                "artifact_combat",
		Type:                TaskTypeCombat,
		Status:              TaskStatusAccepted,
		AssignedDiscipleIDs: []DiscipleID{starterDiscipleID},
		Risk:                45,
	}
	beforeCombat := taskSuccessRate(state, task)
	beforeSeverity := taskConsequenceSeverity(task, beforeCombat, true)
	beforeCultivation := dailyCultivationPointsForPolicy(state, state.Disciples[starterDiscipleID])

	sword := newArtifactState(ItemID("artifact-sword"), ArtifactTypeSword, 1, "test")
	sword.BoundDiscipleID = starterDiscipleID
	robe := newArtifactState(ItemID("artifact-robe"), ArtifactTypeRobe, 1, "test")
	robe.BoundDiscipleID = starterDiscipleID
	state.Inventory.Artifacts[sword.ItemID] = sword
	state.Inventory.Artifacts[robe.ItemID] = robe
	disciple := state.Disciples[starterDiscipleID]
	disciple.Equipment.Weapon = sword.ItemID
	disciple.Equipment.Robe = robe.ItemID
	state.Disciples[starterDiscipleID] = disciple

	afterCombat := taskSuccessRate(state, task)
	afterSeverity := maxInt(1, taskConsequenceSeverity(task, afterCombat, true)-taskTeamArtifactMitigation(state, task))
	afterCultivation := dailyCultivationPointsForPolicy(state, state.Disciples[starterDiscipleID])
	if afterCombat <= beforeCombat {
		t.Fatalf("expected equipped artifact to improve combat task success, before=%d after=%d", beforeCombat, afterCombat)
	}
	if afterSeverity >= beforeSeverity {
		t.Fatalf("expected robe mitigation to reduce risk severity, before=%d after=%d", beforeSeverity, afterSeverity)
	}
	if afterCultivation <= beforeCultivation {
		t.Fatalf("expected robe cultivation stat to increase daily cultivation, before=%d after=%d", beforeCultivation, afterCultivation)
	}

	productionTask := TaskState{
		TaskID:              TaskID("artifact-production"),
		Kind:                "ore_refinement",
		Type:                TaskTypeProduction,
		Status:              TaskStatusAccepted,
		AssignedDiscipleIDs: []DiscipleID{starterDiscipleID},
		Risk:                10,
	}
	beforeProduction := taskSuccessRate(NewInitialSectState(SectID("sect-artifact-effects-base"), UserID("player-artifact"), "青崖宗"), productionTask)
	tool := newArtifactState(ItemID("artifact-tool"), ArtifactTypeFarmTool, 1, "test")
	tool.BoundDiscipleID = starterDiscipleID
	state.Inventory.Artifacts[tool.ItemID] = tool
	disciple = state.Disciples[starterDiscipleID]
	disciple.Equipment.Tool = tool.ItemID
	state.Disciples[starterDiscipleID] = disciple
	afterProduction := taskSuccessRate(state, productionTask)
	if afterProduction <= beforeProduction {
		t.Fatalf("expected tool artifact to improve production task success, before=%d after=%d", beforeProduction, afterProduction)
	}
}

func TestArtifactImpactRepairRestoreAndOfflineContinuity(t *testing.T) {
	base := NewInitialSectState(SectID("sect-artifact-proof"), UserID("player-artifact-proof"), "青崖宗")
	base.Runtime.RNG = RNGState{Seed: 20260424, Cursor: 0}
	baselineMetrics := artifactImpactMetricsForState(base)

	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}
	var eventLog []DomainEvent

	submit := func(cmdID string, commandType CommandType, payload any) SubmitCommandResponse {
		payloadBlob, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s payload: %v", commandType, err)
		}
		response := receiver.executeCommand(SubmitCommand{
			SessionID: "session-artifact-proof",
			Command: ClientCommand{
				CmdID:       cmdID,
				UserID:      string(base.Meta.OwnerUserID),
				SectID:      string(base.Meta.SectID),
				Type:        commandType,
				BaseVersion: receiver.state.Runtime.Version,
				Payload:     payloadBlob,
			},
		})
		if response.Result.Status != CommandResultStatusAccepted {
			t.Fatalf("expected %s accepted, got %+v", commandType, response.Result)
		}
		eventLog = append(eventLog, response.DomainEvents...)
		return response
	}

	submit("cmd-artifact-proof-craft-sword", CommandTypeCraftArtifact, CraftArtifactPayload{ArtifactType: ArtifactTypeSword})
	submit("cmd-artifact-proof-craft-robe", CommandTypeCraftArtifact, CraftArtifactPayload{ArtifactType: ArtifactTypeRobe})
	submit("cmd-artifact-proof-craft-tool", CommandTypeCraftArtifact, CraftArtifactPayload{ArtifactType: ArtifactTypeFarmTool})

	swordID := artifactIDForType(receiver.state, ArtifactTypeSword)
	robeID := artifactIDForType(receiver.state, ArtifactTypeRobe)
	toolID := artifactIDForType(receiver.state, ArtifactTypeFarmTool)
	if swordID == "" || robeID == "" || toolID == "" {
		t.Fatalf("expected crafted sword, robe, and tool artifacts, got %+v", receiver.state.Inventory.Artifacts)
	}

	submit("cmd-artifact-proof-equip-sword", CommandTypeEquipArtifact, EquipArtifactPayload{ItemID: swordID, DiscipleID: starterDiscipleID})
	submit("cmd-artifact-proof-equip-robe", CommandTypeEquipArtifact, EquipArtifactPayload{ItemID: robeID, DiscipleID: starterDiscipleID})
	submit("cmd-artifact-proof-equip-tool", CommandTypeEquipArtifact, EquipArtifactPayload{ItemID: toolID, DiscipleID: starterDiscipleID})

	damagedRobe := receiver.state.Inventory.Artifacts[robeID]
	damagedRobe.Durability = 34
	receiver.state.Inventory.Artifacts[robeID] = damagedRobe
	repaired := submit("cmd-artifact-proof-repair-robe", CommandTypeRepairArtifact, RepairArtifactPayload{ItemID: robeID})
	if repaired.Snapshot.State.Inventory.Artifacts[robeID].Durability != repaired.Snapshot.State.Inventory.Artifacts[robeID].MaxDurability {
		t.Fatalf("expected robe repair to restore durability, got %+v", repaired.Snapshot.State.Inventory.Artifacts[robeID])
	}

	midState := receiver.state.Clone()
	midMetrics := artifactImpactMetricsForState(midState)
	if midMetrics.CombatSuccessRate <= baselineMetrics.CombatSuccessRate {
		t.Fatalf("expected repaired equipped artifacts to improve combat success, before=%d after=%d", baselineMetrics.CombatSuccessRate, midMetrics.CombatSuccessRate)
	}
	if midMetrics.CombatSeverity >= baselineMetrics.CombatSeverity {
		t.Fatalf("expected repaired equipped artifacts to reduce combat severity, before=%d after=%d", baselineMetrics.CombatSeverity, midMetrics.CombatSeverity)
	}
	if midMetrics.ProductionSuccessRate <= baselineMetrics.ProductionSuccessRate {
		t.Fatalf("expected repaired equipped artifacts to improve production success, before=%d after=%d", baselineMetrics.ProductionSuccessRate, midMetrics.ProductionSuccessRate)
	}
	if midMetrics.CultivationPoints <= baselineMetrics.CultivationPoints {
		t.Fatalf("expected repaired equipped artifacts to improve cultivation support, before=%d after=%d", baselineMetrics.CultivationPoints, midMetrics.CultivationPoints)
	}
	if midMetrics.RobeDurability != midMetrics.RobeMaxDurability {
		t.Fatalf("expected repaired robe durability to remain full in mid-state metrics, got %+v", midMetrics)
	}

	savepoint := NewSnapshotReplaySavepoint(base)
	savepoint.AppendReplay(eventLog)
	restoredMid, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore artifact proof savepoint: %v", err)
	}
	if canonicalStateJSON(t, restoredMid) != canonicalStateJSON(t, midState) {
		t.Fatalf("expected artifact proof replay to restore equipped repaired state\nrestored=%+v\nmid=%+v", restoredMid, midState)
	}
	if restoredMidMetrics := artifactImpactMetricsForState(restoredMid); restoredMidMetrics != midMetrics {
		t.Fatalf("expected restored artifact metrics to match mid-state, restored=%+v mid=%+v", restoredMidMetrics, midMetrics)
	}

	continuousReceiver, ok := NewSectActor(midState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete continuous artifact receiver")
	}
	continuousFinal := advanceSectDays(continuousReceiver, 30, "session-artifact-proof-continuous").Snapshot.State
	continuousMetrics := artifactImpactMetricsForState(continuousFinal)

	restoredReceiver, ok := NewSectActor(restoredMid).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete restored artifact receiver")
	}
	restoredFinal := advanceSectDays(restoredReceiver, 30, "session-artifact-proof-restored").Snapshot.State
	if continuousJSON, restoredJSON := canonicalStateJSON(t, continuousFinal), canonicalStateJSON(t, restoredFinal); continuousJSON != restoredJSON {
		t.Fatalf("expected restored artifact branch to match continuous authority state\ncontinuous=%s\nrestored=%s", continuousJSON, restoredJSON)
	}
	if restoredFinalMetrics := artifactImpactMetricsForState(restoredFinal); restoredFinalMetrics != continuousMetrics {
		t.Fatalf("expected restored final artifact metrics to match continuous, restored=%+v continuous=%+v", restoredFinalMetrics, continuousMetrics)
	}

	offlineReceiver, ok := NewSectActor(midState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete offline artifact receiver")
	}
	advanceSectDays(offlineReceiver, 30, "session-artifact-proof-offline")
	if continuousJSON, offlineJSON := canonicalStateJSON(t, continuousFinal), canonicalStateJSON(t, offlineReceiver.state); continuousJSON != offlineJSON {
		t.Fatalf("expected offline artifact continuation to match continuous authority state\ncontinuous=%s\noffline=%s", continuousJSON, offlineJSON)
	}
	if offlineMetrics := artifactImpactMetricsForState(offlineReceiver.state); offlineMetrics != continuousMetrics {
		t.Fatalf("expected offline artifact metrics to match continuous, offline=%+v continuous=%+v", offlineMetrics, continuousMetrics)
	}
}

type artifactImpactMetrics struct {
	CombatSuccessRate     int
	CombatSeverity        int
	ProductionSuccessRate int
	CultivationPoints     int64
	Weapon                ItemID
	Robe                  ItemID
	Tool                  ItemID
	RobeDurability        int
	RobeMaxDurability     int
}

func artifactImpactMetricsForState(state SectState) artifactImpactMetrics {
	combatTask := TaskState{
		TaskID:              TaskID("artifact-proof-combat"),
		Kind:                "artifact_proof_combat",
		Type:                TaskTypeCombat,
		Status:              TaskStatusAccepted,
		AssignedDiscipleIDs: []DiscipleID{starterDiscipleID},
		Risk:                45,
	}
	productionTask := TaskState{
		TaskID:              TaskID("artifact-proof-production"),
		Kind:                "artifact_proof_production",
		Type:                TaskTypeProduction,
		Status:              TaskStatusAccepted,
		AssignedDiscipleIDs: []DiscipleID{starterDiscipleID},
		Risk:                10,
	}
	disciple := state.Disciples[starterDiscipleID]
	robe := state.Inventory.Artifacts[disciple.Equipment.Robe]
	combatSuccessRate := taskSuccessRate(state, combatTask)
	return artifactImpactMetrics{
		CombatSuccessRate:     combatSuccessRate,
		CombatSeverity:        maxInt(1, taskConsequenceSeverity(combatTask, combatSuccessRate, true)-taskTeamArtifactMitigation(state, combatTask)),
		ProductionSuccessRate: taskSuccessRate(state, productionTask),
		CultivationPoints:     dailyCultivationPointsForPolicy(state, disciple),
		Weapon:                disciple.Equipment.Weapon,
		Robe:                  disciple.Equipment.Robe,
		Tool:                  disciple.Equipment.Tool,
		RobeDurability:        robe.Durability,
		RobeMaxDurability:     robe.MaxDurability,
	}
}

func artifactIDForType(state SectState, artifactType ArtifactType) ItemID {
	for itemID, artifact := range state.Inventory.Artifacts {
		if artifact.Type == artifactType {
			return itemID
		}
	}
	return ""
}
