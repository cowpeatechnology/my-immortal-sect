package sect

import (
	"encoding/json"
	"sort"
	"testing"
)

func TestFormationAttachDetachMaintainAuthorityLoop(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-formation"), UserID("player-formation"), "青崖宗")
	initial.Resources.Stock[ResourceKindSpiritStone] = 240
	initial.Resources.Stock[ResourceKindFormationMat] = 60
	initial.Resources.Stock[ResourceKindOre] = 120
	initial.Buildings[BuildingID("building-defense-core")] = newBuildingState(BuildingID("building-defense-core"), "guard_tower", 2, TileCoord{Col: 3, Row: 3})

	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	eventLog := []DomainEvent{}
	submit := func(cmdID string, commandType CommandType, payload any) SubmitCommandResponse {
		t.Helper()
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s payload: %v", commandType, err)
		}
		response := receiver.executeCommand(SubmitCommand{
			SessionID: "session-formation",
			Command: ClientCommand{
				CmdID:       cmdID,
				UserID:      string(initial.Meta.OwnerUserID),
				SectID:      string(initial.Meta.SectID),
				Type:        commandType,
				BaseVersion: receiver.state.Runtime.Version,
				Payload:     body,
			},
		})
		if response.Result.Status != CommandResultStatusAccepted {
			t.Fatalf("expected %s accepted, got %+v", commandType, response.Result)
		}
		eventLog = append(eventLog, response.DomainEvents...)
		return response
	}

	submit("cmd-formation-craft-1", CommandTypeCraftArtifact, CraftArtifactPayload{ArtifactType: ArtifactTypeFormationDisk})
	artifactIDs := formationDiskIDs(receiver.state)
	if len(artifactIDs) != 1 {
		t.Fatalf("expected one crafted formation disk, got %+v", receiver.state.Inventory.Artifacts)
	}

	attached := submit("cmd-formation-attach-1", CommandTypeAttachFormationToBuilding, AttachFormationToBuildingPayload{
		BuildingID:     BuildingID("building-defense-core"),
		ArtifactItemID: artifactIDs[0],
		FormationKind:  FormationKindDefense,
	})
	formation, ok := attached.Snapshot.State.Formations[BuildingID("building-defense-core")]
	if !ok {
		t.Fatalf("expected formation attached to building-defense-core")
	}
	if formation.Kind != FormationKindDefense || !formation.Active {
		t.Fatalf("expected active defense formation, got %+v", formation)
	}
	if attached.Snapshot.State.Inventory.Artifacts[artifactIDs[0]].AttachedBuildingID != BuildingID("building-defense-core") {
		t.Fatalf("expected formation disk attached to building, got %+v", attached.Snapshot.State.Inventory.Artifacts[artifactIDs[0]])
	}

	degraded := receiver.state.Formations[BuildingID("building-defense-core")]
	degraded.MaintenanceDebt = 2
	degraded.Stability = 35
	degraded.Active = false
	receiver.state.Formations[BuildingID("building-defense-core")] = degraded

	maintained := submit("cmd-formation-maintain-1", CommandTypeMaintainFormation, MaintainFormationPayload{
		BuildingID: BuildingID("building-defense-core"),
	})
	maintainedFormation := maintained.Snapshot.State.Formations[BuildingID("building-defense-core")]
	if maintainedFormation.MaintenanceDebt != 0 || maintainedFormation.Stability != 100 || !maintainedFormation.Active {
		t.Fatalf("expected maintenance to restore formation stability, got %+v", maintainedFormation)
	}

	savepoint := NewSnapshotReplaySavepoint(initial)
	savepoint.AppendReplay(eventLog)
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore formation replay: %v", err)
	}
	if canonicalStateJSON(t, restored) != canonicalStateJSON(t, maintained.Snapshot.State) {
		t.Fatalf("expected restored maintained formation state to match authority snapshot")
	}

	detached := submit("cmd-formation-detach-1", CommandTypeDetachFormation, DetachFormationPayload{
		BuildingID: BuildingID("building-defense-core"),
	})
	if _, exists := detached.Snapshot.State.Formations[BuildingID("building-defense-core")]; exists {
		t.Fatalf("expected detached formation removed from state, got %+v", detached.Snapshot.State.Formations)
	}
	if detached.Snapshot.State.Inventory.Artifacts[artifactIDs[0]].AttachedBuildingID != "" {
		t.Fatalf("expected detached formation disk to clear attached building, got %+v", detached.Snapshot.State.Inventory.Artifacts[artifactIDs[0]])
	}
}

func TestFormationEffectsAndDefenseRiskRestoreOfflineContinuity(t *testing.T) {
	base := NewInitialSectState(SectID("sect-formation-proof"), UserID("player-formation-proof"), "青崖宗")
	base.Runtime.RNG = RNGState{Seed: 20260424, Cursor: 0}
	base.Resources.Stock[ResourceKindSpiritStone] = 320
	base.Resources.Stock[ResourceKindFormationMat] = 80
	base.Resources.Stock[ResourceKindOre] = 140
	base.Resources.Stock[ResourceKindBeastMat] = 80
	base.Resources.Stock[ResourceKindHerb] = 60
	base.Meta.Reputation = 96
	base.Policies.TaskPolicy = TaskPolicyCombat
	base.Policies.ResourcePolicy = ResourcePolicyWarPreparation
	base.Policies.RecruitmentPolicy = RecruitmentPolicyAffiliated
	base.Policies.CultivationPolicy = CultivationPolicyClosedCultivation
	base.Policies.Presentation = buildPolicyPresentation(base.Policies)
	applyPolicyEffectsToState(&base)
	base.Events.Tension = 4
	base.Buildings[BuildingID("building-gather-hall")] = newBuildingState(BuildingID("building-gather-hall"), "cave", 1, TileCoord{Col: 1, Row: 2})
	base.Buildings[BuildingID("building-calm-court")] = newBuildingState(BuildingID("building-calm-court"), "dormitory", 1, TileCoord{Col: 2, Row: 2})
	base.Buildings[BuildingID("building-farm-1")] = newBuildingState(BuildingID("building-farm-1"), "spirit_field", 2, TileCoord{Col: 3, Row: 2})
	base.Buildings[BuildingID("building-defense-gate")] = newBuildingState(BuildingID("building-defense-gate"), "guard_tower", 2, TileCoord{Col: 4, Row: 2})
	disciple := base.Disciples[starterDiscipleID]
	disciple.Realm.Stage = RealmFoundation
	disciple.Pressure = 36
	base.Disciples[starterDiscipleID] = disciple
	base.Tasks[TaskID("task-risk-1")] = TaskState{
		TaskID:                TaskID("task-risk-1"),
		Kind:                  "demon_scout",
		Type:                  TaskTypeCombat,
		Status:                TaskStatusAccepted,
		Risk:                  75,
		AssignedDiscipleIDs:   []DiscipleID{starterDiscipleID},
		RequiredProgressDays:  4,
		CompletedProgressDays: 0,
	}

	baselineMetrics := formationImpactMetricsForState(base)

	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}
	eventLog := []DomainEvent{}
	submit := func(cmdID string, commandType CommandType, payload any) SubmitCommandResponse {
		t.Helper()
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s payload: %v", commandType, err)
		}
		response := receiver.executeCommand(SubmitCommand{
			SessionID: "session-formation-proof",
			Command: ClientCommand{
				CmdID:       cmdID,
				UserID:      string(base.Meta.OwnerUserID),
				SectID:      string(base.Meta.SectID),
				Type:        commandType,
				BaseVersion: receiver.state.Runtime.Version,
				Payload:     body,
			},
		})
		if response.Result.Status != CommandResultStatusAccepted {
			t.Fatalf("expected %s accepted, got %+v", commandType, response.Result)
		}
		eventLog = append(eventLog, response.DomainEvents...)
		return response
	}

	for index := 0; index < 4; index++ {
		submit(
			"cmd-formation-proof-craft-"+string(rune('1'+index)),
			CommandTypeCraftArtifact,
			CraftArtifactPayload{ArtifactType: ArtifactTypeFormationDisk},
		)
	}
	artifactIDs := formationDiskIDs(receiver.state)
	if len(artifactIDs) != 4 {
		t.Fatalf("expected four crafted formation disks, got %+v", receiver.state.Inventory.Artifacts)
	}

	submit("cmd-formation-proof-attach-gather", CommandTypeAttachFormationToBuilding, AttachFormationToBuildingPayload{
		BuildingID:     BuildingID("building-gather-hall"),
		ArtifactItemID: artifactIDs[0],
		FormationKind:  FormationKindGatherSpirit,
	})
	submit("cmd-formation-proof-attach-calm", CommandTypeAttachFormationToBuilding, AttachFormationToBuildingPayload{
		BuildingID:     BuildingID("building-calm-court"),
		ArtifactItemID: artifactIDs[1],
		FormationKind:  FormationKindCalmMind,
	})
	submit("cmd-formation-proof-attach-field", CommandTypeAttachFormationToBuilding, AttachFormationToBuildingPayload{
		BuildingID:     BuildingID("building-farm-1"),
		ArtifactItemID: artifactIDs[2],
		FormationKind:  FormationKindFieldGuard,
	})
	submit("cmd-formation-proof-attach-defense", CommandTypeAttachFormationToBuilding, AttachFormationToBuildingPayload{
		BuildingID:     BuildingID("building-defense-gate"),
		ArtifactItemID: artifactIDs[3],
		FormationKind:  FormationKindDefense,
	})

	midState := receiver.state.Clone()
	midMetrics := formationImpactMetricsForState(midState)
	if midMetrics.CultivationPoints <= baselineMetrics.CultivationPoints {
		t.Fatalf("expected formations to improve cultivation, before=%d after=%d", baselineMetrics.CultivationPoints, midMetrics.CultivationPoints)
	}
	if midMetrics.FarmOutputSpiritGrain <= baselineMetrics.FarmOutputSpiritGrain {
		t.Fatalf("expected field guard formation to improve farm output, before=%d after=%d", baselineMetrics.FarmOutputSpiritGrain, midMetrics.FarmOutputSpiritGrain)
	}
	if midMetrics.DefenseRiskMitigation <= baselineMetrics.DefenseRiskMitigation {
		t.Fatalf("expected defense formation to add mitigation, before=%d after=%d", baselineMetrics.DefenseRiskMitigation, midMetrics.DefenseRiskMitigation)
	}
	if midMetrics.DefenseRiskIntensity >= baselineMetrics.DefenseRiskIntensity {
		t.Fatalf("expected defense formation to reduce risk intensity, before=%d after=%d", baselineMetrics.DefenseRiskIntensity, midMetrics.DefenseRiskIntensity)
	}
	if !midMetrics.HasAllRiskSources {
		t.Fatalf("expected defense risk summary to expose all configured source categories, got %s", midMetrics.RiskSourceKey)
	}

	basePressureActor := NewSectActor(base).(*SectActor)
	formedPressureActor := NewSectActor(midState).(*SectActor)
	basePressure := advanceSectDays(basePressureActor, 1, "session-formation-pressure-base").Snapshot.State.Disciples[starterDiscipleID].Pressure
	formedPressure := advanceSectDays(formedPressureActor, 1, "session-formation-pressure-formed").Snapshot.State.Disciples[starterDiscipleID].Pressure
	if formedPressure >= basePressure {
		t.Fatalf("expected calm-mind formation to reduce pressure faster, base=%d formed=%d", basePressure, formedPressure)
	}

	savepoint := NewSnapshotReplaySavepoint(base)
	savepoint.AppendReplay(eventLog)
	restoredMid, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore formation proof savepoint: %v", err)
	}
	if restoredJSON, midJSON := canonicalStateJSON(t, restoredMid), canonicalStateJSON(t, midState); restoredJSON != midJSON {
		t.Fatalf("expected restored formation mid-state to match attached authority state")
	}
	if restoredMetrics := formationImpactMetricsForState(restoredMid); restoredMetrics != midMetrics {
		t.Fatalf("expected restored formation metrics to match mid-state, restored=%+v mid=%+v", restoredMetrics, midMetrics)
	}

	continuousReceiver := NewSectActor(midState).(*SectActor)
	continuousFinal := advanceSectDays(continuousReceiver, 20, "session-formation-proof-continuous").Snapshot.State
	continuousMetrics := formationImpactMetricsForState(continuousFinal)

	restoredReceiver := NewSectActor(restoredMid).(*SectActor)
	restoredFinal := advanceSectDays(restoredReceiver, 20, "session-formation-proof-restored").Snapshot.State
	if continuousJSON, restoredJSON := canonicalStateJSON(t, continuousFinal), canonicalStateJSON(t, restoredFinal); continuousJSON != restoredJSON {
		t.Fatalf("expected restored formation branch to match continuous authority state")
	}
	if restoredMetrics := formationImpactMetricsForState(restoredFinal); restoredMetrics != continuousMetrics {
		t.Fatalf("expected restored final formation metrics to match continuous, restored=%+v continuous=%+v", restoredMetrics, continuousMetrics)
	}

	offlineReceiver := NewSectActor(midState).(*SectActor)
	advanceSectDays(offlineReceiver, 20, "session-formation-proof-offline")
	if continuousJSON, offlineJSON := canonicalStateJSON(t, continuousFinal), canonicalStateJSON(t, offlineReceiver.state); continuousJSON != offlineJSON {
		t.Fatalf("expected offline formation continuation to match continuous authority state")
	}
	if offlineMetrics := formationImpactMetricsForState(offlineReceiver.state); offlineMetrics != continuousMetrics {
		t.Fatalf("expected offline formation metrics to match continuous, offline=%+v continuous=%+v", offlineMetrics, continuousMetrics)
	}
}

type formationImpactMetrics struct {
	CultivationPoints      int64
	FarmOutputSpiritGrain  int64
	DefenseRiskIntensity   int
	DefenseRiskMitigation  int
	HasAllRiskSources      bool
	RiskSourceKey          string
	AttachedFormationCount int
}

func formationImpactMetricsForState(state SectState) formationImpactMetrics {
	farmDelta := productionCompletionDeltaForState(state, ProductionJob{
		ProductionID: ProductionID("formation-proof-farm"),
		Kind:         ProductionKindFarm,
		BuildingID:   BuildingID("building-farm-1"),
		OutputReward: map[ResourceKind]int64{ResourceKindSpiritGrain: 60},
	})
	projection := buildDefenseRiskProjection(state)
	sources := make([]string, 0, len(projection.SourceSummary))
	for _, source := range projection.SourceSummary {
		sources = append(sources, source.Source)
	}
	sort.Strings(sources)
	return formationImpactMetrics{
		CultivationPoints:      dailyCultivationPointsForPolicy(state, state.Disciples[starterDiscipleID]),
		FarmOutputSpiritGrain:  farmDelta[ResourceKindSpiritGrain],
		DefenseRiskIntensity:   projection.Intensity,
		DefenseRiskMitigation:  projection.Mitigation,
		HasAllRiskSources:      hasDefenseRiskSources(projection, []string{"reputation", "wealth", "high_tier_disciples", "external_tasks", "policy", "event_tension", "defense_formation"}),
		RiskSourceKey:          joinStrings(sources),
		AttachedFormationCount: len(state.Formations),
	}
}

func formationDiskIDs(state SectState) []ItemID {
	ids := []ItemID{}
	for itemID, artifact := range state.Inventory.Artifacts {
		if artifact.Type != ArtifactTypeFormationDisk {
			continue
		}
		ids = append(ids, itemID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func hasDefenseRiskSources(projection DefenseRiskProjection, expected []string) bool {
	seen := map[string]bool{}
	for _, source := range projection.SourceSummary {
		seen[source.Source] = true
	}
	for _, source := range expected {
		if !seen[source] {
			return false
		}
	}
	return true
}

func joinStrings(values []string) string {
	if len(values) == 0 {
		return ""
	}
	joined := values[0]
	for index := 1; index < len(values); index++ {
		joined += "|" + values[index]
	}
	return joined
}

type formationDefenseRiskBranch struct {
	receiver *SectActor
	gateID   BuildingID
	eventLog []DomainEvent
	label    string
}

func (b *formationDefenseRiskBranch) appendEvents(events []DomainEvent) {
	b.eventLog = append(b.eventLog, events...)
}

func (b *formationDefenseRiskBranch) submit(t *testing.T, cmdID string, commandType CommandType, payload any) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s payload: %v", commandType, err)
	}
	response := b.receiver.executeCommand(SubmitCommand{
		SessionID: "session-formation-defense-risk-" + b.label,
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(b.receiver.state.Meta.OwnerUserID),
			SectID:      string(b.receiver.state.Meta.SectID),
			Type:        commandType,
			BaseVersion: b.receiver.state.Runtime.Version,
			Payload:     body,
		},
	})
	if response.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected %s accepted, got %+v", commandType, response.Result)
	}
	b.appendEvents(response.DomainEvents)
	return response
}

func newFormationDefenseRiskBranch(t *testing.T, label string, attachDefenseFormation bool) *formationDefenseRiskBranch {
	t.Helper()

	base := NewInitialSectState(
		SectID("sect-formation-defense-risk-"+label),
		UserID("player-formation-defense-risk-"+label),
		"青崖宗",
	)
	base.Runtime.RNG = RNGState{Seed: 20260424, Cursor: 0}
	base.Resources.Stock[ResourceKindSpiritStone] = 150
	base.Resources.Stock[ResourceKindFormationMat] = 20
	base.Resources.Stock[ResourceKindOre] = 80
	base.Resources.Stock[ResourceKindBeastMat] = 60
	base.Resources.Stock[ResourceKindHerb] = 40
	base.Meta.Reputation = 96
	base.Policies.TaskPolicy = TaskPolicyCombat
	base.Policies.ResourcePolicy = ResourcePolicyWarPreparation
	base.Policies.RecruitmentPolicy = RecruitmentPolicyAffiliated
	base.Policies.CultivationPolicy = CultivationPolicyClosedCultivation
	base.Policies.Presentation = buildPolicyPresentation(base.Policies)
	applyPolicyEffectsToState(&base)
	base.Events.Tension = 4
	base.Tasks[TaskID("task-risk-1")] = TaskState{
		TaskID:               TaskID("task-risk-1"),
		Kind:                 "demon_scout",
		Type:                 TaskTypeCombat,
		Status:               TaskStatusAccepted,
		Risk:                 75,
		RequiredProgressDays: 4,
	}

	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete formation defense risk receiver")
	}
	branch := &formationDefenseRiskBranch{
		receiver: receiver,
		label:    label,
	}

	mainHall := buildBuildingCommandWithID(t, receiver, BuildBuildingPayload{
		DefinitionKey: "main_hall",
		Origin:        TileCoord{Col: 4, Row: 6},
	}, receiver.state.Runtime.Version, "cmd-formation-defense-risk-"+label+"-main-hall")
	if mainHall.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected main hall build accepted, got %+v", mainHall.Result)
	}
	branch.appendEvents(mainHall.DomainEvents)

	upgradeMainHall := upgradeBuildingCommandWithID(t, receiver, UpgradeBuildingPayload{
		BuildingID: BuildingID("building-1"),
	}, mainHall.Result.SceneVersion, "cmd-formation-defense-risk-"+label+"-main-hall-upgrade")
	if upgradeMainHall.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected main hall upgrade accepted, got %+v", upgradeMainHall.Result)
	}
	branch.appendEvents(upgradeMainHall.DomainEvents)

		gate := buildBuildingCommandWithID(t, receiver, BuildBuildingPayload{
			DefinitionKey: "gate",
			Origin:        TileCoord{Col: 7, Row: 6},
		}, upgradeMainHall.Result.SceneVersion, "cmd-formation-defense-risk-"+label+"-gate")
		if gate.Result.Status != CommandResultStatusAccepted {
			t.Fatalf("expected gate build accepted, got %+v", gate.Result)
		}
		branch.appendEvents(gate.DomainEvents)
		branch.gateID = buildingIDByDefinition(receiver.state, "gate")
		if branch.gateID == "" {
			t.Fatalf("expected authority branch to create gate building, got %+v", receiver.state.Buildings)
		}
		upgradeGate := upgradeBuildingCommandWithID(t, receiver, UpgradeBuildingPayload{
			BuildingID: branch.gateID,
		}, gate.Result.SceneVersion, "cmd-formation-defense-risk-"+label+"-gate-upgrade")
		if upgradeGate.Result.Status != CommandResultStatusAccepted {
			t.Fatalf("expected gate upgrade accepted, got %+v", upgradeGate.Result)
		}
		branch.appendEvents(upgradeGate.DomainEvents)

		if attachDefenseFormation {
			branch.submit(t, "cmd-formation-defense-risk-"+label+"-craft-disk", CommandTypeCraftArtifact, CraftArtifactPayload{
				ArtifactType: ArtifactTypeFormationDisk,
			})
		artifactIDs := formationDiskIDs(receiver.state)
		if len(artifactIDs) == 0 {
			t.Fatalf("expected crafted formation disk for %s branch", label)
		}
		branch.submit(t, "cmd-formation-defense-risk-"+label+"-attach-defense", CommandTypeAttachFormationToBuilding, AttachFormationToBuildingPayload{
			BuildingID:     branch.gateID,
			ArtifactItemID: artifactIDs[0],
			FormationKind:  FormationKindDefense,
		})
	}

	return branch
}

type formationDefenseLongtermMetrics struct {
	CalendarDay               int64
	RiskIntensity             int
	RiskMitigation            int
	HasDefenseFormationSource bool
	SourceSummaryKey          string
	SpiritStone               int64
	FormationMat              int64
	GatePhase                 string
	GateDurability            int
	GateMaintenanceDebt       int
	FormationCount            int
	FormationActive           bool
	FormationMaintenanceDebt  int
}

func formationDefenseLongtermMetricsForState(state SectState, gateID BuildingID) formationDefenseLongtermMetrics {
	projection := buildDefenseRiskProjection(state)
	sources := make([]string, 0, len(projection.SourceSummary))
	hasDefenseFormationSource := false
	for _, source := range projection.SourceSummary {
		sources = append(sources, source.Source)
		if source.Source == "defense_formation" {
			hasDefenseFormationSource = true
		}
	}
	sort.Strings(sources)

	gate := state.Buildings[gateID]
	formation, hasFormation := state.Formations[gateID]
	return formationDefenseLongtermMetrics{
		CalendarDay:               int64(state.Time.CalendarDay),
		RiskIntensity:             projection.Intensity,
		RiskMitigation:            projection.Mitigation,
		HasDefenseFormationSource: hasDefenseFormationSource,
		SourceSummaryKey:          joinStrings(sources),
		SpiritStone:               state.Resources.Stock[ResourceKindSpiritStone],
		FormationMat:              state.Resources.Stock[ResourceKindFormationMat],
		GatePhase:                 gate.Phase,
		GateDurability:            gate.Durability,
		GateMaintenanceDebt:       gate.MaintenanceDebt,
		FormationCount:            len(state.Formations),
		FormationActive:           hasFormation && formation.Active,
		FormationMaintenanceDebt:  formation.MaintenanceDebt,
	}
}

func buildingIDByDefinition(state SectState, definitionKey string) BuildingID {
	for _, buildingID := range sortedBuildingIDs(state.Buildings) {
		if state.Buildings[buildingID].DefinitionKey == definitionKey {
			return buildingID
		}
	}
	return ""
}

func countDomainEventType(events []DomainEvent, want DomainEventType) int {
	count := 0
	for _, event := range events {
		if event.Type == want {
			count++
		}
	}
	return count
}

func buildingMaintainedReasonPresent(events []DomainEvent, reason string) bool {
	for _, event := range events {
		if event.Type != DomainEventTypeBuildingMaintained {
			continue
		}
		var payload BuildingMaintainedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return false
		}
		if payload.Reason == reason {
			return true
		}
	}
	return false
}

func TestFormationDefenseRiskNinetyDayGoldenSimulation(t *testing.T) {
	weak := newFormationDefenseRiskBranch(t, "weak", false)
	fortified := newFormationDefenseRiskBranch(t, "fortified", true)

	weakSetup := formationDefenseLongtermMetricsForState(weak.receiver.state, weak.gateID)
	fortifiedSetup := formationDefenseLongtermMetricsForState(fortified.receiver.state, fortified.gateID)
	if fortifiedSetup.FormationCount != 1 || weakSetup.FormationCount != 0 {
		t.Fatalf("expected only fortified branch to attach one defense formation, weak=%+v fortified=%+v", weakSetup, fortifiedSetup)
	}
	if fortifiedSetup.RiskMitigation <= weakSetup.RiskMitigation || fortifiedSetup.RiskIntensity >= weakSetup.RiskIntensity {
		t.Fatalf("expected fortified setup to reduce risk before long replay, weak=%+v fortified=%+v", weakSetup, fortifiedSetup)
	}
	if !fortifiedSetup.HasDefenseFormationSource {
		t.Fatalf("expected fortified setup risk summary to expose defense_formation source, got %+v", fortifiedSetup)
	}

	advanceSectDaysWithLog(weak.receiver, 7, "session-formation-defense-risk-weak-early", weak.appendEvents)
	advanceSectDaysWithLog(fortified.receiver, 7, "session-formation-defense-risk-fortified-early", fortified.appendEvents)

	weakEarly := formationDefenseLongtermMetricsForState(weak.receiver.state, weak.gateID)
	fortifiedEarly := formationDefenseLongtermMetricsForState(fortified.receiver.state, fortified.gateID)
	if fortifiedEarly.RiskMitigation <= weakEarly.RiskMitigation || fortifiedEarly.RiskIntensity >= weakEarly.RiskIntensity {
		t.Fatalf("expected fortified early replay window to preserve lower risk profile, weak=%+v fortified=%+v", weakEarly, fortifiedEarly)
	}
	if !fortifiedEarly.HasDefenseFormationSource {
		t.Fatalf("expected fortified early replay window to retain defense_formation source, got %+v", fortifiedEarly)
	}

	advanceSectDaysWithLog(weak.receiver, 83, "session-formation-defense-risk-weak", weak.appendEvents)
	advanceSectDaysWithLog(fortified.receiver, 83, "session-formation-defense-risk-fortified", fortified.appendEvents)

	weakFinal := formationDefenseLongtermMetricsForState(weak.receiver.state, weak.gateID)
	fortifiedFinal := formationDefenseLongtermMetricsForState(fortified.receiver.state, fortified.gateID)
	if weakFinal.CalendarDay < 90 || fortifiedFinal.CalendarDay < 90 {
		t.Fatalf("expected ninety-day replay window, weak=%+v fortified=%+v", weakFinal, fortifiedFinal)
	}
	if fortifiedFinal.GateMaintenanceDebt <= weakFinal.GateMaintenanceDebt {
		t.Fatalf("expected fortified branch to accumulate higher upkeep pressure from defense formation, weak=%+v fortified=%+v", weakFinal, fortifiedFinal)
	}
	if fortifiedFinal.FormationMaintenanceDebt == 0 {
		t.Fatalf("expected fortified ninety-day branch to expose formation maintenance pressure, got %+v", fortifiedFinal)
	}
	if countDomainEventType(fortified.eventLog, DomainEventTypeBuildingDamaged) == 0 {
		t.Fatalf("expected fortified ninety-day replay to include authority building damage, got %+v", domainEventTypes(fortified.eventLog))
	}
}

func TestFormationDefenseRiskRepairRestoreOfflineContinuity(t *testing.T) {
	branch := newFormationDefenseRiskBranch(t, "repair", true)

	advanceSectDaysWithLog(branch.receiver, 30, "session-formation-defense-risk-repair-starve", branch.appendEvents)
	damagedBeforeRepair := formationDefenseLongtermMetricsForState(branch.receiver.state, branch.gateID)
	if damagedBeforeRepair.GatePhase != "damaged" || damagedBeforeRepair.GateMaintenanceDebt == 0 || damagedBeforeRepair.FormationMaintenanceDebt == 0 {
		t.Fatalf("expected defense branch to enter damaged/high-pressure state before repair, got %+v", damagedBeforeRepair)
	}

	publishFundingTask := publishTaskCommand(t, branch.receiver, PublishTaskPayload{
		Kind:                 "formation_repair_funding",
		Title:                "阵法守御修复资金",
		Description:          "固定种子阵法防御验收的 authority 修复补给任务。",
		Priority:             90,
		RequiredProgressDays: 1,
		Risk:                 0,
		MaxAssignees:         1,
		MinIdentity:          IdentityRankOuter,
		MinRealm:             RealmMortal,
		RequiredAptitude:     DiscipleAptitudeState{Physique: 1},
		ContributionReward:   0,
		RewardResources:      map[ResourceKind]int64{ResourceKindSpiritStone: 220},
	})
	if publishFundingTask.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected formation repair funding task publish accepted, got %+v", publishFundingTask.Result)
	}
	branch.appendEvents(publishFundingTask.DomainEvents)

	assignFundingTask := assignTaskCommandWithID(t, branch.receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("task-1"),
		DiscipleID: starterDiscipleID,
		DiscipleIDs: []DiscipleID{
			starterDiscipleID,
		},
	}, publishFundingTask.Result.SceneVersion, "cmd-formation-defense-risk-repair-funding-assign")
	if assignFundingTask.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected formation repair funding assignment accepted, got %+v", assignFundingTask.Result)
	}
	branch.appendEvents(assignFundingTask.DomainEvents)

	advanceSectDaysWithLog(branch.receiver, 1, "session-formation-defense-risk-repair-funding", branch.appendEvents)

	repairedGate := repairBuildingCommandWithID(t, branch.receiver, RepairBuildingPayload{
		BuildingID: branch.gateID,
	}, branch.receiver.state.Runtime.Version, "cmd-formation-defense-risk-repair-gate-repair")
	if repairedGate.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected formation defense gate repair accepted, got %+v", repairedGate.Result)
	}
	if !buildingMaintainedReasonPresent(repairedGate.DomainEvents, "repair_building") {
		t.Fatalf("expected repair command to emit repair_building maintained event, got %+v", repairedGate.DomainEvents)
	}
	branch.appendEvents(repairedGate.DomainEvents)

	maintainedDefense := branch.submit(t, "cmd-formation-defense-risk-maintain-defense", CommandTypeMaintainFormation, MaintainFormationPayload{
		BuildingID: branch.gateID,
	})

	repairedMidState := maintainedDefense.Snapshot.State.Clone()
	repairedMid := formationDefenseLongtermMetricsForState(repairedMidState, branch.gateID)
	if repairedMid.FormationMaintenanceDebt != 0 || !repairedMid.FormationActive || repairedMid.RiskMitigation <= damagedBeforeRepair.RiskMitigation {
		t.Fatalf("expected repair + maintain commands to restore formation mitigation at mid-state, before=%+v after=%+v", damagedBeforeRepair, repairedMid)
	}
	if countDomainEventType(branch.eventLog, DomainEventTypeBuildingMaintained) == 0 || countDomainEventType(branch.eventLog, DomainEventTypeFormationChanged) == 0 {
		t.Fatalf("expected repair branch event log to include building + formation maintenance, got %+v", domainEventTypes(branch.eventLog))
	}

	midState := repairedMidState.Clone()
	continuousReceiver, ok := NewSectActor(midState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete continuous formation defense receiver")
	}
	continuousEventLog := []DomainEvent{}
	advanceSectDaysWithLog(continuousReceiver, 60, "session-formation-defense-risk-repair-continuous", func(events []DomainEvent) {
		continuousEventLog = append(continuousEventLog, events...)
	})
	continuousFinal := continuousReceiver.state
	continuousMetrics := formationDefenseLongtermMetricsForState(continuousFinal, branch.gateID)
	if continuousMetrics.CalendarDay < 91 {
		t.Fatalf("expected repair branch to continue through day 91, got %+v", continuousMetrics)
	}

	savepoint := NewSnapshotReplaySavepoint(midState)
	savepoint.AppendReplay(continuousEventLog)
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore formation defense risk mid-state: %v", err)
	}
	continuousJSON := canonicalStateJSON(t, continuousFinal)
	if restoredJSON := canonicalStateJSON(t, restored); continuousJSON != restoredJSON {
		t.Fatalf("expected repaired formation restore replay to match continuous branch\ncontinuous: %s\nrestored: %s", continuousJSON, restoredJSON)
	}
	if restoredMetrics := formationDefenseLongtermMetricsForState(restored, branch.gateID); restoredMetrics != continuousMetrics {
		t.Fatalf("expected restored formation metrics to match continuous, restored=%+v continuous=%+v", restoredMetrics, continuousMetrics)
	}

	offlineReceiver, ok := NewSectActor(midState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete offline formation defense receiver")
	}
	advanceSectDays(offlineReceiver, 60, "session-formation-defense-risk-repair-offline")
	if offlineJSON := canonicalStateJSON(t, offlineReceiver.state); continuousJSON != offlineJSON {
		t.Fatalf("expected formation defense offline catch-up to match continuous branch\ncontinuous: %s\noffline: %s", continuousJSON, offlineJSON)
	}
	if offlineMetrics := formationDefenseLongtermMetricsForState(offlineReceiver.state, branch.gateID); offlineMetrics != continuousMetrics {
		t.Fatalf("expected offline formation metrics to match continuous, offline=%+v continuous=%+v", offlineMetrics, continuousMetrics)
	}
}
