package sect

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestSectEventChoiceCommandAppliesOptionThroughAuthorityEvents(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-event-choice"), UserID("player-event"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	dayTwo := advanceSectDays(receiver, 2, "session-event-choice")
	event := activeGovernanceChoiceEvent(dayTwo.Snapshot.State)
	if event == nil {
		t.Fatalf("expected authority-generated active choice event, got %+v", dayTwo.Snapshot.State.Events)
	}
	if event.Severity <= 0 || event.ExpiresAtDay <= dayTwo.Snapshot.State.Time.CalendarDay || len(event.Options) < 2 {
		t.Fatalf("expected severity, expiry, and options on choice event, got %+v", event)
	}
	if event.ResultPreview.Summary == "" || event.Options[0].ResultPreview.Summary == "" {
		t.Fatalf("expected event and option result previews, got %+v", event)
	}

	beforeStone := receiver.state.Resources.Stock[ResourceKindSpiritStone]
	beforeHerb := receiver.state.Resources.Stock[ResourceKindHerb]
	result := chooseEventOptionCommand(t, receiver, ChooseEventOptionPayload{
		EventID:  event.EventID,
		OptionID: "send_aid",
	}, receiver.state.Runtime.Version, "cmd-event-choice-1")

	if result.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected choice accepted, got %+v", result.Result)
	}
	if !hasDomainEventType(result.DomainEvents, DomainEventTypeResourceChanged) || !hasDomainEventType(result.DomainEvents, DomainEventTypeSectEventResolved) {
		t.Fatalf("expected resource and resolved events, got %+v", result.DomainEvents)
	}
	if got := result.Snapshot.State.Resources.Stock[ResourceKindSpiritStone]; got != beforeStone-10 {
		t.Fatalf("expected send_aid to consume 10 spirit stone, before %d got %d", beforeStone, got)
	}
	if got := result.Snapshot.State.Resources.Stock[ResourceKindHerb]; got != beforeHerb-3 {
		t.Fatalf("expected send_aid to consume 3 herb, before %d got %d", beforeHerb, got)
	}
	if activeGovernanceChoiceEvent(result.Snapshot.State) != nil {
		t.Fatalf("expected choice event to resolve out of active state, got %+v", result.Snapshot.State.Events.ActiveEvents)
	}
	if len(result.Snapshot.State.Events.ResolvedEvents) == 0 || result.Snapshot.State.Events.ResolvedEvents[len(result.Snapshot.State.Events.ResolvedEvents)-1].Outcome != "option:send_aid" {
		t.Fatalf("expected resolved option summary, got %+v", result.Snapshot.State.Events.ResolvedEvents)
	}
}

func TestSectEventChoiceRejectsExpiredAndUnmetRequirements(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-event-reject"), UserID("player-event"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	advanceSectDays(receiver, 2, "session-event-reject")
	event := activeGovernanceChoiceEvent(receiver.state)
	if event == nil {
		t.Fatalf("expected active choice event")
	}

	receiver.state.Resources.Stock[ResourceKindHerb] = 0
	rejected := chooseEventOptionCommand(t, receiver, ChooseEventOptionPayload{
		EventID:  event.EventID,
		OptionID: "send_aid",
	}, receiver.state.Runtime.Version, "cmd-event-choice-reject-requirement")
	if rejected.Result.Status != CommandResultStatusRejected || rejected.Result.Error == nil || rejected.Result.Error.Code != CommandErrorCodeEventRequirementNotMet {
		t.Fatalf("expected unmet requirement rejection, got %+v", rejected.Result)
	}

	receiver.state.Resources.Stock[ResourceKindHerb] = 20
	receiver.state.Time.CalendarDay = event.ExpiresAtDay
	expired := chooseEventOptionCommand(t, receiver, ChooseEventOptionPayload{
		EventID:  event.EventID,
		OptionID: "send_aid",
	}, receiver.state.Runtime.Version, "cmd-event-choice-reject-expired")
	if expired.Result.Status != CommandResultStatusRejected || expired.Result.Error == nil || expired.Result.Error.Code != CommandErrorCodeEventExpired {
		t.Fatalf("expected expired rejection, got %+v", expired.Result)
	}
}

func TestAdvanceExpiresSectChoiceEventsWithoutChoosingPlayerOutcome(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-event-expire"), UserID("player-event"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	advanceSectDays(receiver, 2, "session-event-expire")
	event := activeGovernanceChoiceEvent(receiver.state)
	if event == nil {
		t.Fatalf("expected active choice event")
	}

	var expiry AdvanceTasksOneDayResponse
	for activeGovernanceChoiceEvent(receiver.state) != nil {
		expiry = receiver.advanceTasksOneDay("session-event-expire")
		if receiver.state.Time.CalendarDay > event.ExpiresAtDay+1 {
			t.Fatalf("event did not expire at expected day %d, state %+v", event.ExpiresAtDay, receiver.state.Events)
		}
	}

	if !hasDomainEventType(expiry.DomainEvents, DomainEventTypeSectEventExpired) {
		t.Fatalf("expected expiry domain event, got %+v", expiry.DomainEvents)
	}
	for _, domainEvent := range expiry.DomainEvents {
		if domainEvent.Type == DomainEventTypeResourceChanged {
			var payload ResourceChangedPayload
			if err := json.Unmarshal(domainEvent.Payload, &payload); err != nil {
				t.Fatalf("decode resource payload: %v", err)
			}
			if strings.HasPrefix(payload.Reason, "sect_event_option:") {
				t.Fatalf("expiry must not silently choose option result, got %+v", payload)
			}
		}
		if domainEvent.Type == DomainEventTypeSectEventResolved {
			t.Fatalf("expected explicit expiry event instead of option resolution, got %+v", domainEvent)
		}
	}
	if len(receiver.state.Events.ResolvedEvents) == 0 || receiver.state.Events.ResolvedEvents[len(receiver.state.Events.ResolvedEvents)-1].Outcome != "expired" {
		t.Fatalf("expected expired resolved summary, got %+v", receiver.state.Events.ResolvedEvents)
	}
}

func TestDismissSectEventResolvesWithoutOptionSideEffects(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-event-dismiss"), UserID("player-event"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	advanceSectDays(receiver, 2, "session-event-dismiss")
	event := activeGovernanceChoiceEvent(receiver.state)
	if event == nil {
		t.Fatalf("expected active choice event")
	}
	beforeResources := cloneResourceAmounts(receiver.state.Resources.Stock)

	result := dismissEventCommand(t, receiver, DismissEventPayload{EventID: event.EventID}, receiver.state.Runtime.Version, "cmd-event-dismiss-1")
	if result.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected dismiss accepted, got %+v", result.Result)
	}
	if hasDomainEventType(result.DomainEvents, DomainEventTypeResourceChanged) {
		t.Fatalf("dismiss must not apply option resource side effects, got %+v", result.DomainEvents)
	}
	if activeGovernanceChoiceEvent(result.Snapshot.State) != nil {
		t.Fatalf("expected dismissed event removed from active events")
	}
	if got := result.Snapshot.State.Resources.Stock[ResourceKindSpiritStone]; got != beforeResources[ResourceKindSpiritStone] {
		t.Fatalf("expected dismiss to preserve resources, before %d got %d", beforeResources[ResourceKindSpiritStone], got)
	}
}

func TestSectEventChoiceCommandIDIsIdempotent(t *testing.T) {
	initial := NewInitialSectState(SectID("sect-event-idempotent"), UserID("player-event"), "青崖宗")
	receiver, ok := NewSectActor(initial).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete sect actor receiver")
	}

	advanceSectDays(receiver, 2, "session-event-idempotent")
	event := activeGovernanceChoiceEvent(receiver.state)
	if event == nil {
		t.Fatalf("expected active choice event")
	}

	first := chooseEventOptionCommand(t, receiver, ChooseEventOptionPayload{
		EventID:  event.EventID,
		OptionID: "send_aid",
	}, receiver.state.Runtime.Version, "cmd-event-choice-idempotent")
	duplicate := chooseEventOptionCommand(t, receiver, ChooseEventOptionPayload{
		EventID:  event.EventID,
		OptionID: "send_aid",
	}, 0, "cmd-event-choice-idempotent")

	if first.Result.Status != CommandResultStatusAccepted || duplicate.Result.Status != CommandResultStatusAccepted {
		t.Fatalf("expected duplicate event choice command to return cached accepted result, got first=%+v duplicate=%+v", first.Result, duplicate.Result)
	}
	if receiver.state.Runtime.Version != first.Result.SceneVersion {
		t.Fatalf("duplicate event choice command must not advance authority version, got state=%d first=%d", receiver.state.Runtime.Version, first.Result.SceneVersion)
	}
	if !reflect.DeepEqual(first.Snapshot.State, duplicate.Snapshot.State) {
		t.Fatalf("duplicate event choice command should return cached snapshot\nfirst: %+v\nduplicate: %+v", first.Snapshot.State, duplicate.Snapshot.State)
	}
	if len(receiver.state.Events.ResolvedEvents) != 1 || receiver.state.Events.ResolvedEvents[0].Outcome != "option:send_aid" {
		t.Fatalf("duplicate event choice command must not append extra resolved events, got %+v", receiver.state.Events.ResolvedEvents)
	}
}

func TestSectEventChoiceSnapshotReplayAndEventLogFeedbackMatchContinuousAuthorityPath(t *testing.T) {
	base := NewInitialSectState(SectID("sect-event-replay"), UserID("player-event-replay"), "青崖宗")

	continuous, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete continuous sect actor receiver")
	}
	startCultivationCommand(t, continuous, StartCultivationPayload{DiscipleID: starterDiscipleID}, continuous.state.Runtime.Version)
	continuous.advanceTasksOneDay("session-event-replay")
	continuousDayTwo := continuous.advanceTasksOneDay("session-event-replay")
	continuousEvent := activeGovernanceChoiceEvent(continuousDayTwo.Snapshot.State)
	if continuousEvent == nil {
		t.Fatalf("expected continuous path to reveal an active governance choice event")
	}
	continuousChoice := chooseEventOptionCommand(t, continuous, ChooseEventOptionPayload{
		EventID:  continuousEvent.EventID,
		OptionID: "send_aid",
	}, continuousDayTwo.ToVersion, "cmd-event-choice-replay")

	restoreSeed, ok := NewSectActor(base).(*SectActor)
	if !ok {
		t.Fatalf("expected concrete restore sect actor receiver")
	}
	startCultivationCommand(t, restoreSeed, StartCultivationPayload{DiscipleID: starterDiscipleID}, restoreSeed.state.Runtime.Version)
	restoreDayOne := restoreSeed.advanceTasksOneDay("session-event-replay")
	snapshotAfterDayOne := restoreDayOne.Snapshot.State.Clone()
	savepoint := NewSnapshotReplaySavepoint(snapshotAfterDayOne)

	restoreDayTwo := restoreSeed.advanceTasksOneDay("session-event-replay")
	restoreEvent := activeGovernanceChoiceEvent(restoreDayTwo.Snapshot.State)
	if restoreEvent == nil {
		t.Fatalf("expected restored branch to reveal an active governance choice event")
	}
	if restoreEvent.EventID != continuousEvent.EventID || restoreEvent.ExpiresAtDay != continuousEvent.ExpiresAtDay {
		t.Fatalf("expected deterministic event identity across restore branch, continuous=%+v restore=%+v", continuousEvent, restoreEvent)
	}
	restoreChoice := chooseEventOptionCommand(t, restoreSeed, ChooseEventOptionPayload{
		EventID:  restoreEvent.EventID,
		OptionID: "send_aid",
	}, restoreDayTwo.ToVersion, "cmd-event-choice-replay")
	savepoint.AppendReplay(restoreDayTwo.DomainEvents)
	savepoint.AppendReplay(restoreChoice.DomainEvents)

	restored, err := savepoint.Restore()
	if err != nil {
		t.Fatalf("restore event choice replay savepoint: %v", err)
	}
	if continuousJSON, restoredJSON := canonicalStateJSON(t, continuousChoice.Snapshot.State), canonicalStateJSON(t, restored); continuousJSON != restoredJSON {
		t.Fatalf("expected restored event choice authority state to match continuous path\ncontinuous: %s\nrestored: %s", continuousJSON, restoredJSON)
	}

	continuousReplayLog := appendEventChoiceLog(continuousDayTwo.DomainEvents, continuousChoice.DomainEvents)
	if len(savepoint.EventLogAfter(snapshotAfterDayOne.Runtime.Version)) != len(continuousReplayLog) {
		t.Fatalf("expected bounded event log after snapshot version %d, got %d want %d", snapshotAfterDayOne.Runtime.Version, len(savepoint.EventLogAfter(snapshotAfterDayOne.Runtime.Version)), len(continuousReplayLog))
	}
	if !reflect.DeepEqual(BuildSectEventFeedbackFromEventLog(continuousReplayLog), savepoint.EventFeedback()) {
		t.Fatalf("expected restored event feedback to match continuous event_log")
	}
	continuousDiary := BuildDiscipleDiaryFromEventLog(continuousReplayLog, starterDiscipleID)
	restoredDiary := savepoint.DiscipleDiary(starterDiscipleID)
	if len(continuousDiary) == 0 || !reflect.DeepEqual(continuousDiary, restoredDiary) {
		t.Fatalf("expected restored diary to match continuous event_log, continuous=%+v restored=%+v", continuousDiary, restoredDiary)
	}
	if activeGovernanceChoiceEvent(restored) != nil || len(restored.Events.ResolvedEvents) == 0 || restored.Events.ResolvedEvents[len(restored.Events.ResolvedEvents)-1].Outcome != "option:send_aid" {
		t.Fatalf("expected restored choice state to close governance event and preserve resolved option, got %+v", restored.Events)
	}
}

func activeGovernanceChoiceEvent(state SectState) *SectEvent {
	for _, eventID := range sortedEventIDs(state.Events.ActiveEvents) {
		event := state.Events.ActiveEvents[eventID]
		if event.Kind == sectGovernanceChoiceEventKind {
			entry := event
			return &entry
		}
	}
	return nil
}

func appendEventChoiceLog(groups ...[]DomainEvent) []DomainEvent {
	var events []DomainEvent
	for _, group := range groups {
		events = append(events, cloneDomainEvents(group)...)
	}
	return events
}

func chooseEventOptionCommand(t *testing.T, receiver *SectActor, payload ChooseEventOptionPayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal choose event option payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-event-choice",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeChooseEventOption,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}

func dismissEventCommand(t *testing.T, receiver *SectActor, payload DismissEventPayload, baseVersion Version, cmdID string) SubmitCommandResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal dismiss event payload: %v", err)
	}
	return receiver.executeCommand(SubmitCommand{
		SessionID: "session-event-choice",
		Command: ClientCommand{
			CmdID:       cmdID,
			UserID:      string(receiver.state.Meta.OwnerUserID),
			SectID:      string(receiver.state.Meta.SectID),
			Type:        CommandTypeDismissEvent,
			Payload:     body,
			BaseVersion: baseVersion,
		},
	})
}
