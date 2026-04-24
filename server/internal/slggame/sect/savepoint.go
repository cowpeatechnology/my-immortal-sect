package sect

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type SnapshotReplaySavepoint struct {
	Snapshot           SectState
	ReplayEvents       []DomainEvent
	EventLog           []DomainEvent
	LastActiveWallUnix int64
}

type DiscipleDiaryEntry struct {
	EventID      string  `json:"event_id"`
	Version      Version `json:"version"`
	EventType    string  `json:"event_type"`
	Summary      string  `json:"summary"`
	RelatedDay   int32   `json:"related_day"`
	RelatedTick  int64   `json:"related_tick"`
	CommandID    string  `json:"command_id,omitempty"`
	ReplaySource string  `json:"replay_source"`
}

type EventLogEntry struct {
	EventID      string  `json:"event_id"`
	Version      Version `json:"version"`
	EventType    string  `json:"event_type"`
	Summary      string  `json:"summary"`
	RelatedDay   int32   `json:"related_day"`
	RelatedTick  int64   `json:"related_tick"`
	CommandID    string  `json:"command_id,omitempty"`
	ReplaySource string  `json:"replay_source"`
}

type SectEventFeedbackEntry struct {
	EventID      string  `json:"event_id"`
	Version      Version `json:"version"`
	EventType    string  `json:"event_type"`
	Category     string  `json:"category"`
	Summary      string  `json:"summary"`
	RelatedDay   int32   `json:"related_day"`
	RelatedTick  int64   `json:"related_tick"`
	CommandID    string  `json:"command_id,omitempty"`
	ReplaySource string  `json:"replay_source"`
}

func NewSnapshotReplaySavepoint(snapshot SectState) SnapshotReplaySavepoint {
	return SnapshotReplaySavepoint{
		Snapshot:     snapshot.Clone(),
		ReplayEvents: []DomainEvent{},
		EventLog:     []DomainEvent{},
	}
}

func (s SnapshotReplaySavepoint) Restore() (SectState, error) {
	state := s.Snapshot.Clone()
	for _, event := range cloneDomainEvents(s.ReplayEvents) {
		if event.Version <= state.Runtime.Version {
			return SectState{}, fmt.Errorf("replay event version %d must be greater than snapshot version %d", event.Version, state.Runtime.Version)
		}
		if err := ApplyEvent(&state, event); err != nil {
			return SectState{}, err
		}
	}
	return state, nil
}

func (s *SnapshotReplaySavepoint) AppendReplay(events []DomainEvent) {
	cloned := cloneDomainEvents(events)
	s.ReplayEvents = append(s.ReplayEvents, cloned...)
	s.EventLog = append(s.EventLog, cloned...)
}

func (s SnapshotReplaySavepoint) EventLogAfter(version Version) []DomainEvent {
	if len(s.EventLog) == 0 {
		return []DomainEvent{}
	}
	filtered := make([]DomainEvent, 0, len(s.EventLog))
	for _, event := range s.EventLog {
		if event.Version > version {
			filtered = append(filtered, event)
		}
	}
	return cloneDomainEvents(filtered)
}

func (s SnapshotReplaySavepoint) DiscipleDiary(discipleID DiscipleID) []DiscipleDiaryEntry {
	return BuildDiscipleDiaryFromEventLog(s.EventLog, discipleID)
}

func (s SnapshotReplaySavepoint) EventFeedback() []SectEventFeedbackEntry {
	return BuildSectEventFeedbackFromEventLog(s.EventLog)
}

func cloneDomainEvents(events []DomainEvent) []DomainEvent {
	if len(events) == 0 {
		return []DomainEvent{}
	}
	cloned := make([]DomainEvent, len(events))
	for index, event := range events {
		entry := event
		entry.Payload = append([]byte(nil), event.Payload...)
		cloned[index] = entry
	}
	return cloned
}

func BuildDiscipleDiaryFromEventLog(events []DomainEvent, discipleID DiscipleID) []DiscipleDiaryEntry {
	if len(events) == 0 {
		return []DiscipleDiaryEntry{}
	}

	sourceByEventID := map[EventID]DiscipleID{}
	entries := make([]DiscipleDiaryEntry, 0, len(events))
	for _, event := range cloneDomainEvents(events) {
		if entry, sourceDiscipleID, ok := diaryEntryForEvent(event, discipleID); ok {
			if sourceDiscipleID != nil {
				sourceByEventID[EventID(event.EventID)] = *sourceDiscipleID
			}
			entries = append(entries, entry)
			continue
		}

		if entry, ok := diaryEntryForResolvedEvent(event, discipleID, sourceByEventID); ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

func BuildEventLogEntriesFromEventLog(events []DomainEvent) []EventLogEntry {
	if len(events) == 0 {
		return []EventLogEntry{}
	}
	entries := make([]EventLogEntry, 0, len(events))
	for _, event := range cloneDomainEvents(events) {
		summary := string(event.Type)
		if feedback, ok := feedbackEntryForEvent(event); ok && feedback.Summary != "" {
			summary = feedback.Summary
		}
		entries = append(entries, EventLogEntry{
			EventID:      event.EventID,
			Version:      event.Version,
			EventType:    string(event.Type),
			Summary:      summary,
			RelatedTick:  event.GameTick,
			CommandID:    event.CommandID,
			ReplaySource: "event_log",
		})
	}
	return entries
}

func BuildSectEventFeedbackFromEventLog(events []DomainEvent) []SectEventFeedbackEntry {
	if len(events) == 0 {
		return []SectEventFeedbackEntry{}
	}
	entries := make([]SectEventFeedbackEntry, 0, len(events))
	for _, event := range cloneDomainEvents(events) {
		if entry, ok := feedbackEntryForEvent(event); ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

func diaryEntryForEvent(event DomainEvent, discipleID DiscipleID) (DiscipleDiaryEntry, *DiscipleID, bool) {
	switch event.Type {
	case DomainEventTypeTaskAccepted, DomainEventTypeTaskProgressed, DomainEventTypeTaskCompleted, DomainEventTypeTaskFailed, DomainEventTypeTaskCancelled:
		var payload TaskChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil || !slices.Contains(payload.Task.AssignedDiscipleIDs, discipleID) {
			return DiscipleDiaryEntry{}, nil, false
		}
		statusText := "接取"
		if event.Type == DomainEventTypeTaskCompleted {
			statusText = "完成"
		} else if event.Type == DomainEventTypeTaskProgressed {
			statusText = "推进"
		} else if event.Type == DomainEventTypeTaskFailed {
			statusText = "失败"
		} else if event.Type == DomainEventTypeTaskCancelled {
			statusText = "取消"
		}
		return newDiaryEntry(event, string(event.Type), fmt.Sprintf("弟子%s任务《%s》。", statusText, payload.Task.Title)), nil, true
	case DomainEventTypeCultivationAdvanced, DomainEventTypeBreakthroughSucceeded, DomainEventTypeBreakthroughFailed, DomainEventTypeDiscipleAssignmentChanged, DomainEventTypeDiscipleSatisfactionChanged, DomainEventTypeDiscipleLoyaltyChanged:
		var payload DiscipleChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil || payload.Disciple.DiscipleID != discipleID {
			return DiscipleDiaryEntry{}, nil, false
		}
		switch event.Type {
		case DomainEventTypeCultivationAdvanced:
			return newDiaryEntry(event, "cultivation", fmt.Sprintf("%s 修炼至 %d 点。", payload.Disciple.Name, payload.Disciple.Realm.CultivationPoints)), nil, true
		case DomainEventTypeBreakthroughSucceeded:
			return newDiaryEntry(event, "breakthrough", fmt.Sprintf("%s 成功突破至 %s。", payload.Disciple.Name, formatDiaryRealmStage(payload.Disciple.Realm.Stage))), nil, true
		case DomainEventTypeBreakthroughFailed:
			return newDiaryEntry(event, "injury", fmt.Sprintf("%s 突破受挫，伤势 %d，压强 %d，HP %d/%d。", payload.Disciple.Name, payload.Disciple.InjuryLevel, payload.Disciple.Pressure, payload.Disciple.HP, payload.Disciple.MaxHP)), nil, true
		case DomainEventTypeDiscipleAssignmentChanged:
			if payload.Disciple.AssignmentKind == DiscipleAssignmentCultivation {
				return newDiaryEntry(event, "assignment", fmt.Sprintf("%s 进入闭关修炼。", payload.Disciple.Name)), nil, true
			}
		case DomainEventTypeDiscipleSatisfactionChanged:
			return newDiaryEntry(event, "monthly", fmt.Sprintf("%s 满意度变为 %d。", payload.Disciple.Name, payload.Disciple.Satisfaction)), nil, true
		case DomainEventTypeDiscipleLoyaltyChanged:
			return newDiaryEntry(event, "monthly", fmt.Sprintf("%s 忠诚度变为 %d。", payload.Disciple.Name, payload.Disciple.Loyalty)), nil, true
		}
	case DomainEventTypePayrollPaid, DomainEventTypePayrollDelayed:
		var payload PayrollSettlementPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil || payload.DiscipleID != discipleID {
			return DiscipleDiaryEntry{}, nil, false
		}
		if event.Type == DomainEventTypePayrollPaid {
			return newDiaryEntry(event, "monthly", fmt.Sprintf("月例发放 %d 灵石，欠例 %d。", payload.Amount, payload.ArrearsAfter)), nil, true
		}
		return newDiaryEntry(event, "monthly", fmt.Sprintf("月例延迟 %d 灵石，欠例 %d。", payload.Amount, payload.ArrearsAfter)), nil, true
	case DomainEventTypeMonthlyObligationChecked:
		var payload MonthlyObligationCheckedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil || payload.DiscipleID != discipleID {
			return DiscipleDiaryEntry{}, nil, false
		}
		statusText := "达成"
		if payload.ViolationAdded {
			statusText = "未达成"
		}
		return newDiaryEntry(event, "monthly", fmt.Sprintf("本月义务%s：%d/%d 天。", statusText, payload.CompletedDays, payload.RequiredDays)), nil, true
	case DomainEventTypeSectEventSeeded, DomainEventTypeSectEventForeshadowed:
		var payload SectEventChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil || payload.Event.SourceDiscipleID == nil || *payload.Event.SourceDiscipleID != discipleID {
			return DiscipleDiaryEntry{}, nil, false
		}
		source := *payload.Event.SourceDiscipleID
		summary := payload.Event.Description
		if event.Type == DomainEventTypeSectEventForeshadowed && payload.Event.OmenText != "" {
			summary = payload.Event.OmenText
		}
		return newDiaryEntry(event, "omen", summary), &source, true
	}

	return DiscipleDiaryEntry{}, nil, false
}

func feedbackEntryForEvent(event DomainEvent) (SectEventFeedbackEntry, bool) {
	switch event.Type {
	case DomainEventTypeResourceChanged:
		var payload ResourceChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		return newFeedbackEntry(event, "resource_change", fmt.Sprintf("资源变化：%s（%s）。", formatResourceChanges(payload.Changes), payload.Reason)), true
	case DomainEventTypeSectMetaChanged:
		var payload SectMetaChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		return newFeedbackEntry(event, "sect_meta", fmt.Sprintf("宗门名望 %d，关系：%s（%s）。", payload.Meta.Reputation, formatRelationChanges(payload.Meta.Relations), payload.Reason)), true
	case DomainEventTypeTaskPublished, DomainEventTypeTaskAccepted, DomainEventTypeTaskCancelled, DomainEventTypeTaskPriorityChanged, DomainEventTypeTaskProgressed, DomainEventTypeTaskCompleted, DomainEventTypeTaskFailed:
		var payload TaskChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		return newFeedbackEntry(event, "task_result", fmt.Sprintf("任务《%s》状态：%s，进度 %d/%d。", payload.Task.Title, payload.Task.Status, payload.Task.CompletedProgressDays, payload.Task.RequiredProgressDays)), true
	case DomainEventTypeContributionEarned, DomainEventTypeContributionSpent:
		var payload ContributionChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		return newFeedbackEntry(event, "contribution", fmt.Sprintf("弟子 %s 贡献变化 %+d（%s）。", payload.DiscipleID, payload.Delta, payload.Reason)), true
	case DomainEventTypeProductionChanged:
		var payload ProductionChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		return newFeedbackEntry(event, "production", fmt.Sprintf("生产 %s/%s 状态 %s，轮次 %d。", payload.Production.Kind, payload.Production.RecipeID, payload.Production.Status, payload.Production.CompletedCycles)), true
	case DomainEventTypeInventoryChanged:
		var payload InventoryChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		return newFeedbackEntry(event, "resource_change", fmt.Sprintf("库存 %s 变为 %d。", payload.Item.Kind, payload.Item.Quantity)), true
	case DomainEventTypeCultivationAdvanced:
		var payload DiscipleChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		return newFeedbackEntry(event, "cultivation", fmt.Sprintf("%s 修炼推进至 %d 点。", payload.Disciple.Name, payload.Disciple.Realm.CultivationPoints)), true
	case DomainEventTypeBreakthroughSucceeded:
		var payload DiscipleChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		return newFeedbackEntry(event, "breakthrough", fmt.Sprintf("%s 突破成功，境界 %s。", payload.Disciple.Name, formatDiaryRealmStage(payload.Disciple.Realm.Stage))), true
	case DomainEventTypeBreakthroughFailed:
		var payload DiscipleChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		return newFeedbackEntry(event, "injury", fmt.Sprintf("%s 突破失败，伤势 %d，压强 %d，HP %d/%d。", payload.Disciple.Name, payload.Disciple.InjuryLevel, payload.Disciple.Pressure, payload.Disciple.HP, payload.Disciple.MaxHP)), true
	case DomainEventTypePayrollPaid, DomainEventTypePayrollDelayed:
		var payload PayrollSettlementPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		status := "发放"
		if event.Type == DomainEventTypePayrollDelayed {
			status = "延迟"
		}
		return newFeedbackEntry(event, "monthly", fmt.Sprintf("月例%s：弟子 %s，金额 %d，欠例 %d。", status, payload.DiscipleID, payload.Amount, payload.ArrearsAfter)), true
	case DomainEventTypeMonthlyObligationChecked:
		var payload MonthlyObligationCheckedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		return newFeedbackEntry(event, "monthly", fmt.Sprintf("月义务：弟子 %s 完成 %d/%d，违规累计 %d。", payload.DiscipleID, payload.CompletedDays, payload.RequiredDays, payload.ViolationsAfter)), true
	case DomainEventTypeDiscipleSatisfactionChanged, DomainEventTypeDiscipleLoyaltyChanged:
		var payload DiscipleChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		return newFeedbackEntry(event, "monthly", fmt.Sprintf("%s 满意/忠诚：%d/%d。", payload.Disciple.Name, payload.Disciple.Satisfaction, payload.Disciple.Loyalty)), true
	case DomainEventTypeAssessmentResolved:
		var payload AssessmentResolvedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		status := "未通过"
		if payload.Assessment.Passed {
			status = "通过"
		}
		return newFeedbackEntry(event, "promotion", fmt.Sprintf("%s 晋升考核%s：目标 %s，评分 %d，原因 %s。", payload.Disciple.Name, status, payload.Assessment.TargetRank, payload.Assessment.Score, payload.Assessment.Reason)), true
	case DomainEventTypeDisciplePromoted:
		var payload DiscipleChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		return newFeedbackEntry(event, "promotion", fmt.Sprintf("%s 晋升为 %s。", payload.Disciple.Name, payload.Disciple.Identity)), true
	case DomainEventTypeMonthAdvanced:
		var payload MonthAdvancedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		return newFeedbackEntry(event, "monthly", fmt.Sprintf("第 %d 月结算：月例发放 %d，延迟 %d，义务违规 %d。", payload.MonthIndex, payload.Summary.StipendPaid, payload.Summary.StipendDelayed, payload.Summary.DutyViolations)), true
	case DomainEventTypeSectEventSeeded, DomainEventTypeSectEventForeshadowed:
		var payload SectEventChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		summary := payload.Event.Description
		if payload.Event.OmenText != "" {
			summary = payload.Event.OmenText
		}
		return newFeedbackEntry(event, "omen", fmt.Sprintf("%s：%s", payload.Event.Title, summary)), true
	case DomainEventTypeSectEventResolved, DomainEventTypeSectEventExpired:
		var payload SectEventResolvedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		prefix := "天机收束"
		if event.Type == DomainEventTypeSectEventExpired {
			prefix = "天机错失"
		}
		return newFeedbackEntry(event, "omen", fmt.Sprintf("%s：%s", prefix, payload.Summary.Summary)), true
	case DomainEventTypePolicyChanged:
		var payload PolicyChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return SectEventFeedbackEntry{}, false
		}
		return newFeedbackEntry(event, "policy", fmt.Sprintf("宗门政策调整：任务=%s，资源=%s，招收=%s，修炼=%s。",
			payload.Policies.TaskPolicy,
			payload.Policies.ResourcePolicy,
			payload.Policies.RecruitmentPolicy,
			payload.Policies.CultivationPolicy,
		)), true
	}
	return SectEventFeedbackEntry{}, false
}

func diaryEntryForResolvedEvent(event DomainEvent, discipleID DiscipleID, sourceByEventID map[EventID]DiscipleID) (DiscipleDiaryEntry, bool) {
	if event.Type != DomainEventTypeSectEventResolved && event.Type != DomainEventTypeSectEventExpired {
		return DiscipleDiaryEntry{}, false
	}
	var payload SectEventResolvedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return DiscipleDiaryEntry{}, false
	}
	if sourceByEventID[payload.EventID] != discipleID {
		return DiscipleDiaryEntry{}, false
	}
	return newDiaryEntry(event, "omen_resolved", payload.Summary.Summary), true
}

func newDiaryEntry(event DomainEvent, eventType string, summary string) DiscipleDiaryEntry {
	return DiscipleDiaryEntry{
		EventID:      event.EventID,
		Version:      event.Version,
		EventType:    eventType,
		Summary:      summary,
		RelatedTick:  event.GameTick,
		CommandID:    event.CommandID,
		ReplaySource: "event_log",
	}
}

func newFeedbackEntry(event DomainEvent, category string, summary string) SectEventFeedbackEntry {
	return SectEventFeedbackEntry{
		EventID:      event.EventID,
		Version:      event.Version,
		EventType:    string(event.Type),
		Category:     category,
		Summary:      summary,
		RelatedTick:  event.GameTick,
		CommandID:    event.CommandID,
		ReplaySource: "event_log",
	}
}

func formatDiaryRealmStage(stage RealmStage) string {
	return strings.ReplaceAll(string(stage), "_", " ")
}

func formatResourceChanges(changes map[ResourceKind]int64) string {
	if len(changes) == 0 {
		return "无变化"
	}
	kinds := make([]ResourceKind, 0, len(changes))
	for kind := range changes {
		kinds = append(kinds, kind)
	}
	slices.SortFunc(kinds, func(left ResourceKind, right ResourceKind) int {
		return strings.Compare(string(left), string(right))
	})
	parts := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		parts = append(parts, fmt.Sprintf("%s%+d", kind, changes[kind]))
	}
	return strings.Join(parts, " / ")
}

func formatRelationChanges(relations map[string]int) string {
	if len(relations) == 0 {
		return "无"
	}
	keys := make([]string, 0, len(relations))
	for key := range relations {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, relations[key]))
	}
	return strings.Join(parts, " / ")
}
