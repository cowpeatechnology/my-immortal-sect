package sect

func (s SectState) Clone() SectState {
	clone := s
	clone.Meta.Relations = cloneStringIntMap(s.Meta.Relations)
	clone.Resources = ResourceState{
		Stock: cloneResourceAmounts(s.Resources.Stock),
		Nodes: cloneResourceNodes(s.Resources.Nodes),
	}
	clone.Contribution = ContributionState{
		Accounts:                cloneContributionAccounts(s.Contribution.Accounts),
		TreasuryRules:           cloneExchangeRules(s.Contribution.TreasuryRules),
		MonthlyPurchases:        cloneMonthlyPurchases(s.Contribution.MonthlyPurchases),
		RedeemabilityRatio:      s.Contribution.RedeemabilityRatio,
		OutstandingContribution: s.Contribution.OutstandingContribution,
		TreasuryValue:           s.Contribution.TreasuryValue,
	}
	clone.Inventory = InventoryState{
		Items:     cloneInventoryItems(s.Inventory.Items),
		Artifacts: cloneArtifacts(s.Inventory.Artifacts),
	}
	clone.Monthly = cloneMonthlyState(s.Monthly)
	clone.Policies = clonePolicyState(s.Policies)
	clone.Disciples = cloneDisciples(s.Disciples)
	clone.Admissions = cloneAdmissions(s.Admissions)
	clone.Buildings = cloneBuildings(s.Buildings)
	clone.Formations = cloneFormations(s.Formations)
	clone.Tasks = cloneTasks(s.Tasks)
	clone.Productions = cloneProductions(s.Productions)
	clone.Institutions = InstitutionState{ByID: cloneInstitutions(s.Institutions.ByID)}
	clone.Order = cloneSectOrder(s.Order)
	clone.Goals = cloneSectGoals(s.Goals)
	clone.Events = cloneSectEvents(s.Events)
	clone.MonthlyAssessment = cloneMonthlyAssessmentState(s.MonthlyAssessment)
	clone.Storylets = cloneStorylets(s.Storylets)
	clone.Flags = cloneFlags(s.Flags)
	return clone
}

func clonePolicyState(source PolicyState) PolicyState {
	clone := source
	clone.CustomFlags = cloneFlags(source.CustomFlags)
	clone.Presentation = clonePolicyPresentationState(source.Presentation)
	return clone
}

func clonePolicyPresentationState(source PolicyPresentationState) PolicyPresentationState {
	clone := PolicyPresentationState{Categories: map[PolicyCategory]PolicyPresentationCategory{}}
	for category, entry := range source.Categories {
		clonedEntry := entry
		clonedEntry.ImpactSummary = append([]string{}, entry.ImpactSummary...)
		clonedEntry.Options = make([]PolicyOptionSummary, 0, len(entry.Options))
		for _, option := range entry.Options {
			clonedOption := option
			clonedOption.ImpactSummary = append([]string{}, option.ImpactSummary...)
			clonedEntry.Options = append(clonedEntry.Options, clonedOption)
		}
		clone.Categories[category] = clonedEntry
	}
	return clone
}

func cloneMonthlyState(source MonthlyState) MonthlyState {
	return MonthlyState{
		LastSettledMonth: source.LastSettledMonth,
		Payroll: PayrollState{
			LastPaidMonth: source.Payroll.LastPaidMonth,
			Arrears:       cloneDiscipleIntMap(source.Payroll.Arrears),
		},
		Obligations: MonthlyObligationState{
			MonthIndex:    source.Obligations.MonthIndex,
			CompletedDays: cloneDiscipleIntMap(source.Obligations.CompletedDays),
			RequiredDays:  cloneDiscipleIntMap(source.Obligations.RequiredDays),
			Violations:    cloneDiscipleIntMap(source.Obligations.Violations),
		},
		LastSettlement: source.LastSettlement,
	}
}

func cloneDiscipleIntMap(source map[DiscipleID]int) map[DiscipleID]int {
	if source == nil {
		return map[DiscipleID]int{}
	}
	clone := make(map[DiscipleID]int, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneStringIntMap(source map[string]int) map[string]int {
	if source == nil {
		return map[string]int{}
	}
	clone := make(map[string]int, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneAdmissions(source AdmissionState) AdmissionState {
	clone := AdmissionState{
		Candidates:                cloneCandidates(source.Candidates),
		LastAnnualRecruitmentYear: source.LastAnnualRecruitmentYear,
	}
	if source.CurrentRecruitment != nil {
		session := *source.CurrentRecruitment
		clone.CurrentRecruitment = &session
	}
	return clone
}

func cloneCandidates(source map[CandidateID]CandidateState) map[CandidateID]CandidateState {
	if source == nil {
		return map[CandidateID]CandidateState{}
	}
	clone := make(map[CandidateID]CandidateState, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneContributionAccounts(source map[DiscipleID]ContributionAccount) map[DiscipleID]ContributionAccount {
	if source == nil {
		return map[DiscipleID]ContributionAccount{}
	}
	clone := make(map[DiscipleID]ContributionAccount, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneExchangeRules(source map[ExchangeItemID]ExchangeRule) map[ExchangeItemID]ExchangeRule {
	if source == nil {
		return map[ExchangeItemID]ExchangeRule{}
	}
	clone := make(map[ExchangeItemID]ExchangeRule, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneMonthlyPurchases(source map[DiscipleID]map[ExchangeItemID]int64) map[DiscipleID]map[ExchangeItemID]int64 {
	if source == nil {
		return map[DiscipleID]map[ExchangeItemID]int64{}
	}
	clone := make(map[DiscipleID]map[ExchangeItemID]int64, len(source))
	for discipleID, items := range source {
		itemClone := make(map[ExchangeItemID]int64, len(items))
		for itemID, count := range items {
			itemClone[itemID] = count
		}
		clone[discipleID] = itemClone
	}
	return clone
}

func cloneStringSlice(source []string) []string {
	if len(source) == 0 {
		return []string{}
	}
	return append([]string{}, source...)
}

func cloneSectOrder(source SectOrderState) SectOrderState {
	clone := source
	clone.Summary = cloneStringSlice(source.Summary)
	return clone
}

func cloneMonthlyAssessmentState(source MonthlyAssessmentState) MonthlyAssessmentState {
	clone := MonthlyAssessmentState{
		LastMonthIndex: source.LastMonthIndex,
		History:        cloneMonthlyAssessmentResults(source.History),
	}
	if source.Latest != nil {
		latest := cloneMonthlyAssessmentResult(*source.Latest)
		clone.Latest = &latest
	}
	return clone
}

func cloneMonthlyAssessmentResults(source []MonthlyAssessmentResult) []MonthlyAssessmentResult {
	if len(source) == 0 {
		return []MonthlyAssessmentResult{}
	}
	clone := make([]MonthlyAssessmentResult, len(source))
	for index, result := range source {
		clone[index] = cloneMonthlyAssessmentResult(result)
	}
	return clone
}

func cloneMonthlyAssessmentResult(source MonthlyAssessmentResult) MonthlyAssessmentResult {
	clone := source
	if source.ChampionDiscipleID != nil {
		discipleID := *source.ChampionDiscipleID
		clone.ChampionDiscipleID = &discipleID
	}
	return clone
}

func cloneResourceNodes(source map[ResourceNodeID]ResourceNodeState) map[ResourceNodeID]ResourceNodeState {
	if source == nil {
		return map[ResourceNodeID]ResourceNodeState{}
	}
	clone := make(map[ResourceNodeID]ResourceNodeState, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneResourceAmounts(source map[ResourceKind]int64) map[ResourceKind]int64 {
	if source == nil {
		return map[ResourceKind]int64{}
	}
	clone := make(map[ResourceKind]int64, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneInventoryItems(source map[ItemID]InventoryEntry) map[ItemID]InventoryEntry {
	if source == nil {
		return map[ItemID]InventoryEntry{}
	}
	clone := make(map[ItemID]InventoryEntry, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneArtifacts(source map[ItemID]ArtifactState) map[ItemID]ArtifactState {
	if source == nil {
		return map[ItemID]ArtifactState{}
	}
	clone := make(map[ItemID]ArtifactState, len(source))
	for key, value := range source {
		entry := value
		entry.Stats = cloneStringIntMap(value.Stats)
		clone[key] = entry
	}
	return clone
}

func cloneDisciples(source map[DiscipleID]DiscipleState) map[DiscipleID]DiscipleState {
	if source == nil {
		return map[DiscipleID]DiscipleState{}
	}
	clone := make(map[DiscipleID]DiscipleState, len(source))
	for key, value := range source {
		entry := value
		if value.AssignmentTask != nil {
			taskID := *value.AssignmentTask
			entry.AssignmentTask = &taskID
		}
		entry.WorkTarget = cloneDiscipleWorkTarget(value.WorkTarget)
		if value.Carrying != nil {
			stack := *value.Carrying
			entry.Carrying = &stack
		}
		entry.Memories = cloneDiscipleMemories(value.Memories)
		entry.Relationship = cloneStringSlice(value.Relationship)
		entry.Emotion = cloneStringSlice(value.Emotion)
		entry.RecentSummary = cloneStringSlice(value.RecentSummary)
		clone[key] = entry
	}
	return clone
}

func cloneDiscipleMemories(source []DiscipleMemoryEntry) []DiscipleMemoryEntry {
	if len(source) == 0 {
		return []DiscipleMemoryEntry{}
	}
	clone := make([]DiscipleMemoryEntry, len(source))
	for index, entry := range source {
		cloned := entry
		cloned.Tags = cloneStringSlice(entry.Tags)
		clone[index] = cloned
	}
	return clone
}

func cloneDiscipleWorkTarget(source DiscipleWorkTargetState) DiscipleWorkTargetState {
	clone := source
	if source.TaskID != nil {
		taskID := *source.TaskID
		clone.TaskID = &taskID
	}
	if source.BuildingID != nil {
		buildingID := *source.BuildingID
		clone.BuildingID = &buildingID
	}
	if source.StoryletID != nil {
		storyletID := *source.StoryletID
		clone.StoryletID = &storyletID
	}
	return clone
}

func cloneBuildings(source map[BuildingID]BuildingState) map[BuildingID]BuildingState {
	if source == nil {
		return map[BuildingID]BuildingState{}
	}
	clone := make(map[BuildingID]BuildingState, len(source))
	for key, value := range source {
		entry := value
		if value.InstitutionID != nil {
			institutionID := *value.InstitutionID
			entry.InstitutionID = &institutionID
		}
		entry.ActiveTaskIDs = append([]TaskID(nil), value.ActiveTaskIDs...)
		entry.AssignedWorkers = append([]DiscipleID(nil), value.AssignedWorkers...)
		clone[key] = entry
	}
	return clone
}

func cloneFormations(source map[BuildingID]FormationState) map[BuildingID]FormationState {
	if source == nil {
		return map[BuildingID]FormationState{}
	}
	clone := make(map[BuildingID]FormationState, len(source))
	for key, value := range source {
		entry := value
		entry.MaintenanceCost = cloneResourceAmounts(value.MaintenanceCost)
		entry.EffectSummary = append([]string(nil), value.EffectSummary...)
		clone[key] = entry
	}
	return clone
}

func cloneTasks(source map[TaskID]TaskState) map[TaskID]TaskState {
	if source == nil {
		return map[TaskID]TaskState{}
	}
	clone := make(map[TaskID]TaskState, len(source))
	for key, value := range source {
		entry := value
		entry.AssignedDiscipleIDs = append([]DiscipleID(nil), value.AssignedDiscipleIDs...)
		if value.TargetBuildingID != nil {
			buildingID := *value.TargetBuildingID
			entry.TargetBuildingID = &buildingID
		}
		if value.TargetStoryletID != nil {
			storyletID := *value.TargetStoryletID
			entry.TargetStoryletID = &storyletID
		}
		entry.DispatchCost = cloneResourceAmounts(value.DispatchCost)
		entry.RewardResources = cloneResourceAmounts(value.RewardResources)
		entry.RelationReward = cloneStringIntMap(value.RelationReward)
		clone[key] = entry
	}
	return clone
}

func cloneProductions(source map[ProductionID]ProductionJob) map[ProductionID]ProductionJob {
	if source == nil {
		return map[ProductionID]ProductionJob{}
	}
	clone := make(map[ProductionID]ProductionJob, len(source))
	for key, value := range source {
		entry := value
		entry.AssignedDisciples = append([]DiscipleID(nil), value.AssignedDisciples...)
		entry.InputCost = cloneResourceAmounts(value.InputCost)
		entry.OutputReward = cloneResourceAmounts(value.OutputReward)
		entry.Shortage = cloneResourceAmounts(value.Shortage)
		clone[key] = entry
	}
	return clone
}

func cloneInstitutions(source map[InstitutionID]InstitutionLoopState) map[InstitutionID]InstitutionLoopState {
	if source == nil {
		return map[InstitutionID]InstitutionLoopState{}
	}
	clone := make(map[InstitutionID]InstitutionLoopState, len(source))
	for key, value := range source {
		entry := value
		if value.ManagerDiscipleID != nil {
			managerID := *value.ManagerDiscipleID
			entry.ManagerDiscipleID = &managerID
		}
		entry.EffectSummary = append([]string(nil), value.EffectSummary...)
		entry.GatePolicy.GuardDiscipleIDs = append([]DiscipleID(nil), value.GatePolicy.GuardDiscipleIDs...)
		entry.CaveSlots = cloneCaveSlots(value.CaveSlots)
		entry.AssignedBuildingIDs = append([]BuildingID(nil), value.AssignedBuildingIDs...)
		entry.ActiveTaskIDs = append([]TaskID(nil), value.ActiveTaskIDs...)
		clone[key] = entry
	}
	return clone
}

func cloneSectEvents(source SectEventState) SectEventState {
	return SectEventState{
		ActiveEvents:          cloneActiveSectEvents(source.ActiveEvents),
		ResolvedEvents:        cloneResolvedSectEvents(source.ResolvedEvents),
		Tension:               source.Tension,
		LastMajorEventVersion: source.LastMajorEventVersion,
	}
}

func cloneSectGoals(source SectGoalsState) SectGoalsState {
	return SectGoalsState{
		ByID:     cloneGoalsByID(source.ByID),
		Resolved: cloneResolvedGoals(source.Resolved),
	}
}

func cloneGoalsByID(source map[SectGoalID]SectGoal) map[SectGoalID]SectGoal {
	if source == nil {
		return map[SectGoalID]SectGoal{}
	}
	clone := make(map[SectGoalID]SectGoal, len(source))
	for key, value := range source {
		entry := value
		entry.RewardResources = cloneResourceAmounts(value.RewardResources)
		entry.RewardSummary = append([]string(nil), value.RewardSummary...)
		entry.Tags = append([]string(nil), value.Tags...)
		if value.FocusDiscipleID != nil {
			discipleID := *value.FocusDiscipleID
			entry.FocusDiscipleID = &discipleID
		}
		clone[key] = entry
	}
	return clone
}

func cloneResolvedGoals(source []ResolvedSectGoalSummary) []ResolvedSectGoalSummary {
	if source == nil {
		return []ResolvedSectGoalSummary{}
	}
	clone := make([]ResolvedSectGoalSummary, len(source))
	copy(clone, source)
	return clone
}

func cloneActiveSectEvents(source map[EventID]SectEvent) map[EventID]SectEvent {
	if source == nil {
		return map[EventID]SectEvent{}
	}
	clone := make(map[EventID]SectEvent, len(source))
	for key, value := range source {
		entry := value
		if value.SourceDiscipleID != nil {
			discipleID := *value.SourceDiscipleID
			entry.SourceDiscipleID = &discipleID
		}
		entry.Requirements = cloneSectEventRequirement(value.Requirements)
		entry.Options = cloneSectEventOptions(value.Options)
		entry.ResultPreview = cloneSectEventResultPreview(value.ResultPreview)
		entry.Tags = append([]string(nil), value.Tags...)
		clone[key] = entry
	}
	return clone
}

func cloneSectEventOptions(source []SectEventOption) []SectEventOption {
	if source == nil {
		return nil
	}
	clone := make([]SectEventOption, len(source))
	for index, value := range source {
		clone[index] = value
		clone[index].Requirements = cloneSectEventRequirement(value.Requirements)
		clone[index].ResultPreview = cloneSectEventResultPreview(value.ResultPreview)
	}
	return clone
}

func cloneSectEventRequirement(source SectEventRequirement) SectEventRequirement {
	clone := source
	clone.MinResources = cloneResourceAmounts(source.MinResources)
	if source.RequiredDiscipleID != nil {
		discipleID := *source.RequiredDiscipleID
		clone.RequiredDiscipleID = &discipleID
	}
	return clone
}

func cloneSectEventResultPreview(source SectEventResultPreview) SectEventResultPreview {
	clone := source
	clone.ResourceDelta = cloneResourceAmounts(source.ResourceDelta)
	return clone
}

func cloneResolvedSectEvents(source []ResolvedEventSummary) []ResolvedEventSummary {
	if source == nil {
		return []ResolvedEventSummary{}
	}
	clone := make([]ResolvedEventSummary, len(source))
	copy(clone, source)
	return clone
}

func cloneStorylets(source map[StoryletID]StoryletState) map[StoryletID]StoryletState {
	if source == nil {
		return map[StoryletID]StoryletState{}
	}
	clone := make(map[StoryletID]StoryletState, len(source))
	for key, value := range source {
		entry := value
		entry.BoundDiscipleIDs = append([]DiscipleID(nil), value.BoundDiscipleIDs...)
		entry.BoundBuildingIDs = append([]BuildingID(nil), value.BoundBuildingIDs...)
		clone[key] = entry
	}
	return clone
}

func cloneFlags(source map[string]bool) map[string]bool {
	if source == nil {
		return map[string]bool{}
	}
	clone := make(map[string]bool, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
