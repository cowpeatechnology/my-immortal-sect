package sect

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildingUnlocksSectLevelPrerequisitesAndLimits(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-building-unlock"), UserID("player-building-unlock"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	lockedGate := buildBuildingCommandWithID(t, receiver, BuildBuildingPayload{
		DefinitionKey: "gate",
		Origin:        TileCoord{Col: 7, Row: 6},
	}, receiver.state.Runtime.Version, "cmd-build-locked-gate")
	if lockedGate.Result.Status != CommandResultStatusRejected ||
		lockedGate.Result.Error == nil ||
		!strings.Contains(lockedGate.Result.Error.Message, "sect level") {
		t.Fatalf("expected gate blocked by sect level/main hall unlocks, got %+v", lockedGate.Result)
	}

	mainHall := buildBuildingCommandWithID(t, receiver, BuildBuildingPayload{
		DefinitionKey: "main_hall",
		Origin:        TileCoord{Col: 4, Row: 6},
	}, receiver.state.Runtime.Version, "cmd-build-main-hall-unlock")
	if mainHall.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected main hall build accepted, got %+v", mainHall.Result)
	}
	if got := mainHall.Snapshot.State.Meta.Level; got != 1 {
		t.Fatalf("expected sect level 1 after level-one main hall, got %d", got)
	}
	warehouseCatalog := requireBuildingCatalogEntry(t, mainHall.Snapshot.BuildingCatalog, "warehouse")
	if !warehouseCatalog.Unlocked || warehouseCatalog.MaintenanceByLevel[1][ResourceKindSpiritStone] != 1 {
		t.Fatalf("expected warehouse catalog to expose unlock and level-one maintenance, got %+v", warehouseCatalog)
	}
	lockedGateCatalog := requireBuildingCatalogEntry(t, mainHall.Snapshot.BuildingCatalog, "gate")
	if lockedGateCatalog.Unlocked || lockedGateCatalog.CanBuild || len(lockedGateCatalog.Blockers) == 0 {
		t.Fatalf("expected gate catalog to expose locked blockers before main hall level two, got %+v", lockedGateCatalog)
	}

	duplicateMainHall := buildBuildingCommandWithID(t, receiver, BuildBuildingPayload{
		DefinitionKey: "main_hall",
		Origin:        TileCoord{Col: 5, Row: 6},
	}, mainHall.Result.SceneVersion, "cmd-build-duplicate-main-hall")
	if duplicateMainHall.Result.Status != CommandResultStatusRejected ||
		duplicateMainHall.Result.Error == nil ||
		!strings.Contains(duplicateMainHall.Result.Error.Message, "count limit") {
		t.Fatalf("expected duplicate main hall blocked by building count limit, got %+v", duplicateMainHall.Result)
	}

	upgradedMainHall := upgradeBuildingCommandWithID(t, receiver, UpgradeBuildingPayload{
		BuildingID: BuildingID("building-1"),
	}, mainHall.Result.SceneVersion, "cmd-upgrade-main-hall-unlock")
	if upgradedMainHall.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected main hall upgrade accepted, got %+v", upgradedMainHall.Result)
	}
	if got := upgradedMainHall.Snapshot.State.Meta.Level; got != 2 {
		t.Fatalf("expected main hall upgrade to raise sect level to 2, got %d", got)
	}
	unlockedGateCatalog := requireBuildingCatalogEntry(t, upgradedMainHall.Snapshot.BuildingCatalog, "gate")
	if !unlockedGateCatalog.Unlocked || !unlockedGateCatalog.CanBuild || unlockedGateCatalog.MaintenanceByLevel[2][ResourceKindSpiritStone] != 2 {
		t.Fatalf("expected gate catalog to expose unlocked maintenance after main hall level two, got %+v", unlockedGateCatalog)
	}

	gate := buildBuildingCommandWithID(t, receiver, BuildBuildingPayload{
		DefinitionKey: "gate",
		Origin:        TileCoord{Col: 7, Row: 6},
	}, upgradedMainHall.Result.SceneVersion, "cmd-build-unlocked-gate")
	if gate.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected gate build accepted after main hall level 2, got %+v", gate.Result)
	}

	limited := NewInitialSectState(SectID("sect-building-limit"), UserID("player-building-limit"), "青崖宗")
	limited.Meta.BuildingLimit = 1
	limited.Buildings[BuildingID("building-main-hall")] = newBuildingState(BuildingID("building-main-hall"), "main_hall", 1, TileCoord{Col: 4, Row: 6})
	limitedReceiver, ok := NewSectActor(limited).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete limited sect actor receiver")
	}
	overLimit := buildBuildingCommandWithID(t, limitedReceiver, BuildBuildingPayload{
		DefinitionKey: "warehouse",
		Origin:        TileCoord{Col: 5, Row: 6},
	}, limitedReceiver.state.Runtime.Version, "cmd-build-over-global-limit")
	if overLimit.Result.Status != CommandResultStatusRejected ||
		overLimit.Result.Error == nil ||
		!strings.Contains(overLimit.Result.Error.Message, "building limit") {
		t.Fatalf("expected global building limit rejection, got %+v", overLimit.Result)
	}
}

func requireBuildingCatalogEntry(t *testing.T, catalog []BuildingCatalogEntry, definitionKey string) BuildingCatalogEntry {
	t.Helper()
	for _, entry := range catalog {
		if entry.DefinitionKey == definitionKey {
			return entry
		}
	}
	t.Fatalf("missing building catalog entry %s in %+v", definitionKey, catalog)
	return BuildingCatalogEntry{}
}

func TestDailyBuildingMaintenanceEventsRestoreAndOfflineContinuity(t *testing.T) {
	paidState := NewInitialSectState(SectID("sect-building-maintenance"), UserID("player-building-maintenance"), "青崖宗")
	paidState.Buildings[BuildingID("building-warehouse")] = newBuildingState(BuildingID("building-warehouse"), "warehouse", 1, TileCoord{Col: 5, Row: 6})
	refreshSectBuildingMeta(&paidState)
	paidReceiver, ok := NewSectActor(paidState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete maintenance receiver")
	}

	paidDay := paidReceiver.advanceTasksOneDay("session-building-maintenance")
	if !hasDomainEventType(paidDay.DomainEvents, DomainEventTypeResourceChanged) ||
		!hasDomainEventType(paidDay.DomainEvents, DomainEventTypeBuildingMaintained) {
		t.Fatalf("expected daily maintenance resource and building events, got %+v", paidDay.DomainEvents)
	}
	if got := paidDay.Snapshot.State.Resources.Stock[ResourceKindSpiritStone]; got != paidState.Resources.Stock[ResourceKindSpiritStone]-1 {
		t.Fatalf("expected maintenance to deduct one spirit stone, got %d", got)
	}
	maintained := paidDay.Snapshot.State.Buildings[BuildingID("building-warehouse")]
	if maintained.Phase != "active" || maintained.Efficiency != 100 || maintained.MaintenanceDebt != 0 || maintained.Durability != 100 {
		t.Fatalf("expected paid maintenance to keep active/full-efficiency building, got %+v", maintained)
	}
	if !maintenanceReasonPresent(t, paidDay.DomainEvents, "building_maintenance:building-warehouse") {
		t.Fatalf("expected resource event reason to identify maintained building, got %+v", paidDay.DomainEvents)
	}

	paidSavepoint := NewSnapshotReplaySavepoint(paidState)
	paidSavepoint.AppendReplay(paidDay.DomainEvents)
	paidRestored, err := paidSavepoint.Restore()
	if err != nil {
		t.Fatalf("restore paid maintenance savepoint: %v", err)
	}
	if canonicalStateJSON(t, paidRestored) != canonicalStateJSON(t, paidDay.Snapshot.State) {
		t.Fatalf("expected paid maintenance replay to match snapshot\nrestored=%+v\nsnapshot=%+v", paidRestored, paidDay.Snapshot.State)
	}

	shortageState := NewInitialSectState(SectID("sect-building-shortage"), UserID("player-building-shortage"), "青崖宗")
	shortageState.Resources.Stock[ResourceKindSpiritStone] = 0
	shortageState.Buildings[BuildingID("building-warehouse")] = newBuildingState(BuildingID("building-warehouse"), "warehouse", 1, TileCoord{Col: 5, Row: 6})
	refreshSectBuildingMeta(&shortageState)
	shortageReceiver, ok := NewSectActor(shortageState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete shortage receiver")
	}

	shortageDay := shortageReceiver.advanceTasksOneDay("session-building-shortage")
	if !hasDomainEventType(shortageDay.DomainEvents, DomainEventTypeBuildingDamaged) {
		t.Fatalf("expected daily maintenance shortage to damage building, got %+v", shortageDay.DomainEvents)
	}
	damaged := shortageDay.Snapshot.State.Buildings[BuildingID("building-warehouse")]
	if damaged.Phase != "damaged" || damaged.DamagedReason != "maintenance_shortage" || damaged.MaintenanceDebt != 1 || damaged.Efficiency >= 100 || damaged.Durability >= 100 {
		t.Fatalf("expected shortage to lower efficiency/durability and mark damaged, got %+v", damaged)
	}

	shortageSavepoint := NewSnapshotReplaySavepoint(shortageState)
	shortageSavepoint.AppendReplay(shortageDay.DomainEvents)
	shortageRestored, err := shortageSavepoint.Restore()
	if err != nil {
		t.Fatalf("restore shortage maintenance savepoint: %v", err)
	}
	if canonicalStateJSON(t, shortageRestored) != canonicalStateJSON(t, shortageDay.Snapshot.State) {
		t.Fatalf("expected shortage maintenance replay to match snapshot\nrestored=%+v\nsnapshot=%+v", shortageRestored, shortageDay.Snapshot.State)
	}

	offlineReceiver, ok := NewSectActor(shortageState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete offline shortage receiver")
	}
	offlineDay := offlineReceiver.advanceTasksOneDay("session-building-shortage-offline")
	if canonicalStateJSON(t, offlineDay.Snapshot.State) != canonicalStateJSON(t, shortageDay.Snapshot.State) {
		t.Fatalf("expected offline catch-up to match continuous daily maintenance\ncontinuous=%+v\noffline=%+v", shortageDay.Snapshot.State, offlineDay.Snapshot.State)
	}
}

func TestRepairBuildingCommandRestoresDamagedBuildingThroughAuthorityEvents(t *testing.T) {
	state := NewInitialSectState(SectID("sect-building-repair"), UserID("player-building-repair"), "青崖宗")
	damaged := newBuildingState(BuildingID("building-warehouse"), "warehouse", 1, TileCoord{Col: 5, Row: 6})
	damaged.Phase = "damaged"
	damaged.HP = 40
	damaged.Efficiency = 50
	damaged.Durability = 40
	damaged.MaintenanceDebt = 2
	damaged.DamagedReason = "maintenance_shortage"
	state.Buildings[damaged.BuildingID] = damaged
	refreshSectBuildingMeta(&state)

	receiver, ok := NewSectActor(state).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete repair receiver")
	}

	repaired := repairBuildingCommandWithID(t, receiver, RepairBuildingPayload{
		BuildingID: damaged.BuildingID,
	}, receiver.state.Runtime.Version, "cmd-repair-warehouse")
	if repaired.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected repair accepted, got %+v", repaired.Result)
	}
	if !hasDomainEventType(repaired.DomainEvents, DomainEventTypeResourceChanged) ||
		!hasDomainEventType(repaired.DomainEvents, DomainEventTypeBuildingMaintained) {
		t.Fatalf("expected repair to emit resource and building events, got %+v", repaired.DomainEvents)
	}
	if got := repaired.Snapshot.State.Resources.Stock[ResourceKindSpiritStone]; got != state.Resources.Stock[ResourceKindSpiritStone]-2 {
		t.Fatalf("expected repair to pay maintenance-debt cost, got %d", got)
	}
	building := repaired.Snapshot.State.Buildings[damaged.BuildingID]
	if building.Phase != "active" || building.HP != building.MaxHP || building.Efficiency != 100 || building.Durability != 100 || building.MaintenanceDebt != 0 || building.DamagedReason != "" {
		t.Fatalf("expected repair to restore authority building state, got %+v", building)
	}

	duplicate := repairBuildingCommandWithID(t, receiver, RepairBuildingPayload{
		BuildingID: damaged.BuildingID,
	}, state.Runtime.Version, "cmd-repair-warehouse")
	if duplicate.Result.Status != CommandResultStatusAccepted ||
		duplicate.Result.SceneVersion != repaired.Result.SceneVersion ||
		duplicate.Snapshot.State.Resources.Stock[ResourceKindSpiritStone] != repaired.Snapshot.State.Resources.Stock[ResourceKindSpiritStone] {
		t.Fatalf("expected duplicate repair cmd_id to return cached result without extra spend, got %+v", duplicate.Result)
	}

	savepoint := NewSnapshotReplaySavepoint(state)
	savepoint.AppendReplay(repaired.DomainEvents)
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore repaired savepoint: %v", err)
	}
	if canonicalStateJSON(t, restored) != canonicalStateJSON(t, repaired.Snapshot.State) {
		t.Fatalf("expected repaired replay to match snapshot\nrestored=%+v\nsnapshot=%+v", restored, repaired.Snapshot.State)
	}
}

func TestBuildingMaintenanceDamageRepairUpgradeSixtyDayRestoreOfflineContinuity(t *testing.T) {
	base := NewInitialSectState(SectID("sect-building-maintenance-60d"), UserID("player-building-maintenance-60d"), "青崖宗")
	base.Runtime.RNG = RNGState{Seed: 20_260_423, Cursor: 0}
	base.Tasks = map[TaskID]TaskState{}
	base.Productions = map[ProductionID]ProductionJob{}
	base.Events = SectEventState{
		ActiveEvents:   map[EventID]SectEvent{},
		ResolvedEvents: []ResolvedEventSummary{},
	}

	receiver, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete building maintenance receiver")
	}
	var eventLog []DomainEvent
	appendEvents := func(events []DomainEvent) {
		eventLog = append(eventLog, events...)
	}

	mainHall := buildBuildingCommandWithID(t, receiver, BuildBuildingPayload{
		DefinitionKey: "main_hall",
		Origin:        TileCoord{Col: 4, Row: 6},
	}, receiver.state.Runtime.Version, "cmd-building-maintenance-60d-main-hall")
	if mainHall.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected main hall build accepted, got %+v", mainHall.Result)
	}
	appendEvents(mainHall.DomainEvents)

	upgradeMainHall := upgradeBuildingCommandWithID(t, receiver, UpgradeBuildingPayload{
		BuildingID: BuildingID("building-1"),
	}, mainHall.Result.SceneVersion, "cmd-building-maintenance-60d-main-hall-upgrade")
	if upgradeMainHall.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected main hall upgrade accepted, got %+v", upgradeMainHall.Result)
	}
	appendEvents(upgradeMainHall.DomainEvents)

	warehouse := buildBuildingCommandWithID(t, receiver, BuildBuildingPayload{
		DefinitionKey: "warehouse",
		Origin:        TileCoord{Col: 5, Row: 6},
	}, upgradeMainHall.Result.SceneVersion, "cmd-building-maintenance-60d-warehouse")
	if warehouse.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected warehouse build accepted, got %+v", warehouse.Result)
	}
	appendEvents(warehouse.DomainEvents)
	warehouseID := BuildingID("building-2")
	if receiver.state.Buildings[warehouseID].DefinitionKey != "warehouse" {
		t.Fatalf("expected deterministic warehouse building-2, got %+v", receiver.state.Buildings)
	}

	starveDays := advanceSectDaysWithLog(receiver, 30, "session-building-maintenance-60d-starve", appendEvents)
	if !hasDomainEventType(starveDays.DomainEvents, DomainEventTypeBuildingDamaged) {
		t.Fatalf("expected sixty-day proof starve window to damage building, got %+v", starveDays.DomainEvents)
	}
	damaged := receiver.state.Buildings[warehouseID]
	if damaged.Phase != "damaged" ||
		damaged.DamagedReason != "maintenance_shortage" ||
		damaged.MaintenanceDebt == 0 ||
		damaged.Efficiency >= 100 ||
		damaged.Durability >= 100 {
		t.Fatalf("expected maintenance shortage to lower efficiency/durability and mark damaged, got %+v", damaged)
	}

	publishFundingTask := publishTaskCommand(t, receiver, PublishTaskPayload{
		Kind:                 "maintenance_repair_funding",
		Title:                "建筑维护修复资金",
		Description:          "固定种子建筑维护验收的 authority 资源补充任务。",
		Priority:             90,
		RequiredProgressDays: 1,
		Risk:                 0,
		MaxAssignees:         1,
		MinIdentity:          IdentityRankOuter,
		MinRealm:             RealmMortal,
		RequiredAptitude:     DiscipleAptitudeState{Physique: 1},
		ContributionReward:   0,
		RewardResources:      map[ResourceKind]int64{ResourceKindSpiritStone: 20},
	})
	if publishFundingTask.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected repair funding task publish accepted, got %+v", publishFundingTask.Result)
	}
	appendEvents(publishFundingTask.DomainEvents)
	assignFundingTask := assignTaskCommandWithID(t, receiver, AssignDiscipleTaskPayload{
		TaskID:     TaskID("task-1"),
		DiscipleID: starterDiscipleID,
		DiscipleIDs: []DiscipleID{
			starterDiscipleID,
		},
	}, publishFundingTask.Result.SceneVersion, "cmd-building-maintenance-60d-funding-assign")
	if assignFundingTask.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected repair funding task assignment accepted, got %+v", assignFundingTask.Result)
	}
	appendEvents(assignFundingTask.DomainEvents)

	fundingDay := advanceSectDaysWithLog(receiver, 1, "session-building-maintenance-60d-funding", appendEvents)
	if !hasDomainEventType(fundingDay.DomainEvents, DomainEventTypeTaskCompleted) ||
		!hasDomainEventType(fundingDay.DomainEvents, DomainEventTypeResourceChanged) {
		t.Fatalf("expected funding task to complete and reward resources, got %+v", fundingDay.DomainEvents)
	}

	beforeRepair := receiver.state.Buildings[warehouseID]
	repair := repairBuildingCommandWithID(t, receiver, RepairBuildingPayload{
		BuildingID: warehouseID,
	}, receiver.state.Runtime.Version, "cmd-building-maintenance-60d-repair-warehouse")
	if repair.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected warehouse repair accepted after authority resource reward, got %+v", repair.Result)
	}
	appendEvents(repair.DomainEvents)
	repaired := repair.Snapshot.State.Buildings[warehouseID]
	if repaired.Phase != "active" ||
		repaired.HP != repaired.MaxHP ||
		repaired.Efficiency != 100 ||
		repaired.Durability != 100 ||
		repaired.MaintenanceDebt != 0 ||
		repaired.DamagedReason != "" {
		t.Fatalf("expected repair to restore building before long continuation, before=%+v after=%+v", beforeRepair, repaired)
	}

	midState := receiver.state.Clone()
	midEventCut := len(eventLog)
	finalAdvance := advanceSectDaysWithLog(receiver, 60, "session-building-maintenance-60d-continuous", appendEvents)
	finalState := finalAdvance.Snapshot.State
	if finalState.Time.CalendarDay < 91 {
		t.Fatalf("expected at least 91 authority days after 30+1+60 advances, got %+v", finalState.Time)
	}
	finalWarehouse := finalState.Buildings[warehouseID]
	if finalWarehouse.MaintenanceDebt == 0 || finalWarehouse.Efficiency >= 100 || finalWarehouse.Durability >= 100 {
		t.Fatalf("expected continued 60-day maintenance pressure to remain visible, got %+v", finalWarehouse)
	}

	seenEvents := domainEventTypes(eventLog)
	for _, eventType := range []DomainEventType{
		DomainEventTypeBuildingBuilt,
		DomainEventTypeBuildingUpgraded,
		DomainEventTypeBuildingDamaged,
		DomainEventTypeBuildingMaintained,
		DomainEventTypeTaskCompleted,
		DomainEventTypeResourceChanged,
	} {
		if !seenEvents[eventType] {
			t.Fatalf("expected 60-day building maintenance proof to include %s, got %+v", eventType, seenEvents)
		}
	}

	savepoint := NewSnapshotReplaySavepoint(midState)
	savepoint.AppendReplay(eventLog[midEventCut:])
	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore building maintenance day-31 savepoint: %v", err)
	}
	continuousJSON := canonicalStateJSON(t, finalState)
	if restoredJSON := canonicalStateJSON(t, restored); continuousJSON != restoredJSON {
		t.Fatalf("expected building maintenance snapshot replay to match continuous branch\ncontinuous: %s\nrestored: %s", continuousJSON, restoredJSON)
	}

	offlineReceiver, ok := NewSectActor(midState).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete offline building maintenance receiver")
	}
	advanceSectDays(offlineReceiver, 60, "session-building-maintenance-60d-offline")
	if offlineJSON := canonicalStateJSON(t, offlineReceiver.state); continuousJSON != offlineJSON {
		t.Fatalf("expected building maintenance offline catch-up to match continuous branch\ncontinuous: %s\noffline: %s", continuousJSON, offlineJSON)
	}
}

func advanceSectDaysWithLog(receiver *SectActor, days int, sessionID string, appendEvents func([]DomainEvent)) AdvanceTasksOneDayResponse {
	var response AdvanceTasksOneDayResponse
	for day := 0; day < days; day++ {
		response = receiver.advanceTasksOneDay(sessionID)
		appendEvents(response.DomainEvents)
	}
	return response
}

func repairBuildingCommandWithID(t *testing.T, receiver *SectActor, payload RepairBuildingPayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal repair building payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-building-repair",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeRepairBuilding,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func maintenanceReasonPresent(t *testing.T, events []DomainEvent, reason string) bool {
	t.Helper()
	for _, event := range events {
		if event.Type != DomainEventTypeResourceChanged {
			continue
		}
		var payload ResourceChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode resource changed payload: %v", err)
		}
		if payload.Reason == reason {
			return true
		}
	}
	return false
}
