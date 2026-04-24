export type SectAuthorityResourceKind =
    | 'spirit_stone'
    | 'spirit_grain'
    | 'herb'
    | 'ore'
    | 'beast_mat'
    | 'formation_mat';

export type SectAuthorityTaskStatus = 'published' | 'accepted' | 'completed' | 'failed' | 'cancelled';
export type SectAuthorityTaskType = 'internal' | 'external' | 'explore' | 'combat' | 'production' | string;
export type SectAuthorityTaskGrade = 'ding' | 'bing' | 'yi' | 'jia' | 'special' | string;
export type SectAuthorityTaskEvaluation = 'excellent' | 'good' | 'normal' | 'poor' | 'failed' | string;
export type SectAuthorityExchangeItemKind = 'resource' | string;
export type SectAuthorityArtifactType = 'sword' | 'robe' | 'farm_tool' | 'alchemy_furnace' | 'formation_disk' | string;
export type SectAuthorityArtifactSlot = 'weapon' | 'robe' | 'tool' | 'special' | string;
export type SectAuthorityCommandType =
    | 'COMMAND_TYPE_BUILD_BUILDING'
    | 'COMMAND_TYPE_UPGRADE_BUILDING'
    | 'COMMAND_TYPE_REPAIR_BUILDING'
    | 'COMMAND_TYPE_PUBLISH_TASK'
    | 'COMMAND_TYPE_CANCEL_TASK'
    | 'COMMAND_TYPE_ASSIGN_DISCIPLE_TASK'
    | 'COMMAND_TYPE_SET_TASK_PRIORITY'
    | 'COMMAND_TYPE_EXCHANGE_CONTRIBUTION_ITEM'
    | 'COMMAND_TYPE_START_PRODUCTION'
    | 'COMMAND_TYPE_CANCEL_PRODUCTION'
    | 'COMMAND_TYPE_ADJUST_PRODUCTION'
    | 'COMMAND_TYPE_START_CULTIVATION'
    | 'COMMAND_TYPE_USE_PILL_FOR_CULTIVATION'
    | 'COMMAND_TYPE_RESERVE_CAVE'
    | 'COMMAND_TYPE_ATTEMPT_BREAKTHROUGH'
    | 'COMMAND_TYPE_START_RECRUITMENT'
    | 'COMMAND_TYPE_ACCEPT_CANDIDATE'
    | 'COMMAND_TYPE_REJECT_CANDIDATE'
    | 'COMMAND_TYPE_CHOOSE_EVENT_OPTION'
    | 'COMMAND_TYPE_DISMISS_EVENT'
    | 'COMMAND_TYPE_START_ASSESSMENT'
    | 'COMMAND_TYPE_PROMOTE_DISCIPLE'
    | 'COMMAND_TYPE_SET_POLICY'
    | 'COMMAND_TYPE_ASSIGN_INSTITUTION_MANAGER'
    | 'COMMAND_TYPE_SET_GATE_POLICY'
    | 'COMMAND_TYPE_SET_EXCHANGE_RULE'
    | 'COMMAND_TYPE_CRAFT_ARTIFACT'
    | 'COMMAND_TYPE_EQUIP_ARTIFACT'
    | 'COMMAND_TYPE_UNEQUIP_ARTIFACT'
    | 'COMMAND_TYPE_REPAIR_ARTIFACT';

export type SectAuthorityCommandStatus = 'COMMAND_RESULT_STATUS_ACCEPTED' | 'COMMAND_RESULT_STATUS_REJECTED';
export type SectAuthorityProductionStatus = 'running' | 'blocked' | 'completed' | 'cancelled' | string;
export type SectAuthorityRealmStage =
    | 'mortal'
    | 'qi_entry'
    | 'qi_early'
    | 'qi_middle'
    | 'qi_late'
    | 'foundation'
    | 'golden_core'
    | string;
export type SectAuthorityEventStatus = 'seeded' | 'foreshadowed' | 'resolved' | 'expired' | string;
export type SectAuthorityGoalStatus = 'active' | 'completed' | 'failed' | string;

export type SectAuthorityResourceState = {
    stock: Partial<Record<SectAuthorityResourceKind, number>>;
};

export type SectAuthorityTileCoord = {
    col: number;
    row: number;
};

export type SectAuthorityMetaSnapshot = {
    sect_id: string;
    owner_user_id: string;
    name: string;
    created_at_unix_seconds: number;
    level: number;
    expansion: number;
    building_limit: number;
};

export type SectAuthorityInventoryItemSnapshot = {
    item_id: string;
    kind: string;
    quantity: number;
    bound: boolean;
    source_tag: string;
};

export type SectAuthorityArtifactSnapshot = {
    item_id: string;
    type: SectAuthorityArtifactType;
    quality: number;
    durability: number;
    max_durability: number;
    bound_disciple_id?: string | null;
    stats?: Record<string, number>;
    source_tag?: string;
};

export type SectAuthorityInventorySnapshot = {
    items: Record<string, SectAuthorityInventoryItemSnapshot>;
    artifacts?: Record<string, SectAuthorityArtifactSnapshot>;
};

export type SectAuthorityRealmSnapshot = {
    stage: SectAuthorityRealmStage;
    cultivation_points: number;
    ready_for_breakthrough: boolean;
    failed_breakthrough_count: number;
};

export type SectAuthorityDiscipleNeedsSnapshot = {
    daily_spirit_grain: number;
    daily_rest_ticks: number;
};

export type SectAuthorityDiscipleSupportSnapshot = {
    food_satisfied: boolean;
    housing_satisfied: boolean;
    medical_supported: boolean;
};

export type SectAuthorityCultivationDecisionSnapshot = {
    daily_gain: number;
    required_points: number;
    progress_percent: number;
    environment_bonus: number;
    cultivation_pill_available: number;
    breakthrough_pill_available: number;
    breakthrough_spirit_stone_cost: number;
    breakthrough_success_rate: number;
    breakthrough_risk: number;
    breakthrough_risk_limit: number;
    omen_status: string;
    omen_text?: string;
};

export type SectAuthorityDiscipleAssessmentSnapshot = {
    target_rank?: string;
    passed: boolean;
    score: number;
    reason?: string;
    resolved_at_version?: number;
};

export type SectAuthorityEquipmentSnapshot = {
    weapon?: string;
    robe?: string;
    tool?: string;
    special?: string;
};

export type SectAuthorityDiscipleAptitudeSnapshot = {
    spirit_root: number;
    comprehension: number;
    physique: number;
    mind: number;
    luck: number;
};

export type SectAuthorityDiscipleMemorySnapshot = {
    kind: string;
    summary: string;
    source_event_type: string;
    recorded_at_version: number;
    recorded_at_day: number;
    intensity: number;
    tags: string[];
};

export type SectAuthorityDiscipleSnapshot = {
    disciple_id: string;
    name: string;
    identity: string;
    aptitude: SectAuthorityDiscipleAptitudeSnapshot;
    assignment_kind: string;
    assignment_task?: string | null;
    work_target: {
        task_id?: string | null;
        building_id?: string | null;
        storylet_id?: string | null;
        description?: string;
    };
    realm: SectAuthorityRealmSnapshot;
    needs: SectAuthorityDiscipleNeedsSnapshot;
    support: SectAuthorityDiscipleSupportSnapshot;
    equipment?: SectAuthorityEquipmentSnapshot;
    pressure: number;
    injury_level: number;
    hp: number;
    max_hp: number;
    satisfaction: number;
    loyalty: number;
    cultivation_decision: SectAuthorityCultivationDecisionSnapshot;
    assessment?: SectAuthorityDiscipleAssessmentSnapshot;
    memories?: SectAuthorityDiscipleMemorySnapshot[];
    relationship_tags?: string[];
    emotion_tags?: string[];
    recent_experience_summary?: string[];
};

export type SectAuthorityBuildingSnapshot = {
    building_id: string;
    definition_key: string;
    level: number;
    phase: string;
    origin?: SectAuthorityTileCoord;
    hp: number;
    max_hp: number;
    durability?: number;
    efficiency?: number;
    maintenance_debt?: number;
    damaged_reason?: string;
    institution_id?: string | null;
    active_task_ids?: string[];
    assigned_workers?: string[];
};

export type SectAuthorityBuildingCatalogInstance = {
    building_id: string;
    level: number;
    phase: string;
    efficiency: number;
    durability: number;
    maintenance_debt: number;
    damaged_reason?: string;
};

export type SectAuthorityBuildingCatalogEntry = {
    definition_key: string;
    label: string;
    max_level: number;
    unlock_sect_level: number;
    required_main_hall_level?: number;
    max_count: number;
    current_count: number;
    build_cost: Partial<Record<SectAuthorityResourceKind, number>>;
    upgrade_cost_by_level?: Record<string, Partial<Record<SectAuthorityResourceKind, number>>>;
    maintenance_by_level?: Record<string, Partial<Record<SectAuthorityResourceKind, number>>>;
    unlocked: boolean;
    can_build: boolean;
    blockers?: string[];
    existing_buildings?: SectAuthorityBuildingCatalogInstance[];
};

export type SectAuthorityRecruitmentSessionSnapshot = {
    recruitment_id: string;
    type: string;
    started_at_calendar_day: number;
    ends_at_calendar_day: number;
    investment_spirit_stone: number;
    candidate_count: number;
};

export type SectAuthorityCandidateSnapshot = {
    candidate_id: string;
    name: string;
    source: string;
    identity: string;
    aptitude: SectAuthorityDiscipleAptitudeSnapshot;
    realm: SectAuthorityRealmSnapshot;
    needs: SectAuthorityDiscipleNeedsSnapshot;
    support: SectAuthorityDiscipleSupportSnapshot;
    pressure: number;
    injury_level: number;
    hp: number;
    max_hp: number;
};

export type SectAuthorityAdmissionSnapshot = {
    current_recruitment?: SectAuthorityRecruitmentSessionSnapshot | null;
    candidates: Record<string, SectAuthorityCandidateSnapshot>;
    last_annual_recruitment_year: number;
};

export type SectAuthorityTaskSnapshot = {
    task_id: string;
    kind: string;
    type?: SectAuthorityTaskType;
    grade?: SectAuthorityTaskGrade;
    title: string;
    description?: string;
    status: SectAuthorityTaskStatus;
    priority: number;
    assigned_disciple_ids?: string[];
    progress_ticks: number;
    required_progress_days: number;
    completed_progress_days: number;
    risk: number;
    max_assignees: number;
    min_identity?: string;
    min_realm?: string;
    required_aptitude: SectAuthorityDiscipleAptitudeSnapshot;
    dispatch_cost?: Partial<Record<SectAuthorityResourceKind, number>>;
    contribution_reward: number;
    reward_resources?: Partial<Record<SectAuthorityResourceKind, number>>;
    reputation_reward?: number;
    relation_reward?: Record<string, number>;
    crisis_clue?: string;
    success_rate?: number;
    evaluation?: SectAuthorityTaskEvaluation;
};

export type SectAuthorityTaskDispatchProjection = {
    task_id: string;
    recommended_disciple_ids?: string[];
    recommended_success_rate?: number;
    blocked_reason?: string;
};

export type SectAuthorityProductionSnapshot = {
    production_id: string;
    kind: string;
    building_id: string;
    recipe_id: string;
    status: SectAuthorityProductionStatus;
    priority: number;
    target_cycles?: number;
    assigned_disciples?: string[];
    input_cost?: Partial<Record<SectAuthorityResourceKind, number>>;
    output_reward?: Partial<Record<SectAuthorityResourceKind, number>>;
    progress_days: number;
    required_progress_days: number;
    completed_cycles: number;
    blocked_reason?: string;
    shortage?: Partial<Record<SectAuthorityResourceKind, number>>;
};

export type SectAuthorityInstitutionManagerEffectSnapshot = {
    manager_score: number;
    identity_bonus: number;
    aptitude_bonus: number;
    loyalty_modifier: number;
    injury_penalty: number;
    efficiency_bonus: number;
};

export type SectAuthorityGatePolicySnapshot = {
    open_to_visitors: boolean;
    allow_wandering_cultivators: boolean;
    guard_disciple_ids?: string[];
};

export type SectAuthorityCaveSlotSnapshot = {
    slot_id: string;
    occupied_by?: string | null;
    reserved_until_day?: number;
    environment_bonus: number;
};

export type SectAuthorityInstitutionSnapshot = {
    institution_id: string;
    kind: string;
    level: number;
    enabled: boolean;
    manager_disciple_id?: string | null;
    manager_effect?: SectAuthorityInstitutionManagerEffectSnapshot;
    capacity?: number;
    comfort?: number;
    healing_power?: number;
    cultivation_support?: number;
    task_capacity_bonus?: number;
    exchange_pressure?: number;
    efficiency?: number;
    effect_summary?: string[];
    gate_policy?: SectAuthorityGatePolicySnapshot;
    public_exchange_enabled?: boolean;
    cave_slots?: SectAuthorityCaveSlotSnapshot[];
    assigned_building_ids?: string[];
    active_task_ids?: string[];
};

export type SectAuthorityEventRequirementSnapshot = {
    min_resources?: Partial<Record<SectAuthorityResourceKind, number>>;
    required_disciple_id?: string | null;
};

export type SectAuthorityEventResultPreviewSnapshot = {
    resource_delta?: Partial<Record<SectAuthorityResourceKind, number>>;
    disciple_pressure_delta?: number;
    disciple_satisfaction_delta?: number;
    task_id?: string;
    task_title?: string;
    fame_delta?: number;
    tension_delta?: number;
    summary?: string;
};

export type SectAuthorityEventOptionSnapshot = {
    option_id: string;
    label: string;
    description?: string;
    requirements?: SectAuthorityEventRequirementSnapshot;
    result_preview: SectAuthorityEventResultPreviewSnapshot;
};

export type SectAuthorityEventSnapshot = {
    event_id: string;
    kind: string;
    status: SectAuthorityEventStatus;
    severity: number;
    title: string;
    description: string;
    omen_text?: string;
    chain_id?: string;
    chain_stage?: string;
    source_disciple_id?: string | null;
    seeded_at_version: number;
    reveal_at_version: number;
    revealed_at_version: number;
    resolved_at_version: number;
    expires_at_day?: number;
    requirements?: SectAuthorityEventRequirementSnapshot;
    options?: SectAuthorityEventOptionSnapshot[];
    result_preview?: SectAuthorityEventResultPreviewSnapshot;
    tags?: string[];
};

export type SectAuthorityResolvedEventSummary = {
    event_id: string;
    kind: string;
    outcome: string;
    summary: string;
    resolved_at_version: number;
};

export type SectAuthorityEventStateSnapshot = {
    active_events: Record<string, SectAuthorityEventSnapshot>;
    resolved_events: SectAuthorityResolvedEventSummary[];
    tension: number;
    last_major_event_version: number;
};

export type SectAuthorityOrderSnapshot = {
    safety: number;
    discipline: number;
    internal_strife_risk: number;
    summary?: string[];
    last_updated_version?: number;
};

export type SectAuthorityGoalSnapshot = {
    goal_id: string;
    kind: string;
    title: string;
    description?: string;
    status: SectAuthorityGoalStatus;
    current_progress: number;
    target_progress: number;
    progress_text?: string;
    reward_resources?: Partial<Record<SectAuthorityResourceKind, number>>;
    reward_reputation?: number;
    reward_satisfaction?: number;
    failure_reputation?: number;
    failure_satisfaction?: number;
    reward_summary?: string[];
    focus_disciple_id?: string | null;
    outcome_summary?: string;
    started_at_version?: number;
    resolved_at_version?: number;
    tags?: string[];
};

export type SectAuthorityResolvedGoalSummary = {
    goal_id: string;
    kind: string;
    outcome: string;
    summary: string;
    resolved_at_version: number;
};

export type SectAuthorityGoalsStateSnapshot = {
    by_id: Record<string, SectAuthorityGoalSnapshot>;
    resolved: SectAuthorityResolvedGoalSummary[];
};

export type SectAuthorityMonthlyAssessmentResultSnapshot = {
    month_index: number;
    champion_disciple_id?: string | null;
    champion_name?: string;
    score: number;
    reward_contribution?: number;
    reward_reputation?: number;
    promotion_momentum?: number;
    summary: string;
    resolved_at_version: number;
};

export type SectAuthorityMonthlyAssessmentStateSnapshot = {
    last_month_index: number;
    latest?: SectAuthorityMonthlyAssessmentResultSnapshot | null;
    history?: SectAuthorityMonthlyAssessmentResultSnapshot[];
};

export type SectAuthorityPolicyCategory = 'task' | 'resource' | 'recruitment' | 'cultivation' | string;

export type SectAuthorityPolicyOptionSummary = {
    value: string;
    label: string;
    explanation: string;
    impact_summary?: string[];
};

export type SectAuthorityPolicyPresentationCategory = {
    category: SectAuthorityPolicyCategory;
    current_value: string;
    current_label: string;
    explanation: string;
    impact_summary?: string[];
    options?: SectAuthorityPolicyOptionSummary[];
};

export type SectAuthorityPolicyPresentationState = {
    categories: Record<SectAuthorityPolicyCategory, SectAuthorityPolicyPresentationCategory>;
};

export type SectAuthorityPolicyStateSnapshot = {
    task_policy: string;
    resource_policy: string;
    recruitment_policy: string;
    cultivation_policy: string;
    custom_flags?: Record<string, boolean>;
    presentation?: SectAuthorityPolicyPresentationState;
};

export type SectAuthorityContributionAccountSnapshot = {
    disciple_id: string;
    balance: number;
    earned_total: number;
    spent_total: number;
};

export type SectAuthorityExchangeRuleSnapshot = {
    exchange_item_id: string;
    name: string;
    item_kind: SectAuthorityExchangeItemKind;
    item_ref: string;
    contribution_cost: number;
    monthly_limit: number;
    stock_limit: number;
    enabled: boolean;
};

export type SectAuthorityContributionSnapshot = {
    accounts: Record<string, SectAuthorityContributionAccountSnapshot>;
    treasury_rules: Record<string, SectAuthorityExchangeRuleSnapshot>;
    monthly_purchases: Record<string, Record<string, number>>;
    redeemability_ratio: number;
    outstanding_contribution: number;
    treasury_value: number;
};

export type SectAuthorityMonthlySnapshot = {
    last_settled_month: number;
    payroll: {
        last_paid_month: number;
        arrears: Record<string, number>;
    };
    obligations: {
        month_index: number;
        completed_days: Record<string, number>;
        required_days: Record<string, number>;
        violations: Record<string, number>;
    };
    last_settlement: {
        month_index: number;
        stipend_paid: number;
        stipend_delayed: number;
        payroll_paid_count: number;
        payroll_delayed_count: number;
        duty_required_days: number;
        duty_completed_days: number;
        duty_violations: number;
        redeemability_ratio: number;
        resource_shortage: boolean;
        contribution_shortage: boolean;
        satisfaction_delta_total: number;
        loyalty_delta_total: number;
    };
};

export type SectAuthorityRuntimeSnapshot = {
    version: number;
    last_applied_event_version: number;
    last_snapshot_version: number;
    authority_boundary: string;
    dirty: boolean;
};

export type SectAuthorityTimeSnapshot = {
    game_tick: number;
    calendar_day: number;
    day_tick: number;
    season_index: number;
};

export type SectAuthorityStateSnapshot = {
    meta: SectAuthorityMetaSnapshot;
    runtime?: SectAuthorityRuntimeSnapshot;
    time?: SectAuthorityTimeSnapshot;
    resources: SectAuthorityResourceState;
    inventory: SectAuthorityInventorySnapshot;
    contribution: SectAuthorityContributionSnapshot;
    monthly?: SectAuthorityMonthlySnapshot;
    monthly_assessment?: SectAuthorityMonthlyAssessmentStateSnapshot;
    disciples: Record<string, SectAuthorityDiscipleSnapshot>;
    admissions: SectAuthorityAdmissionSnapshot;
    buildings: Record<string, SectAuthorityBuildingSnapshot>;
    tasks: Record<string, SectAuthorityTaskSnapshot>;
    productions: Record<string, SectAuthorityProductionSnapshot>;
    institutions: {
        by_id: Record<string, SectAuthorityInstitutionSnapshot>;
    };
    policies: SectAuthorityPolicyStateSnapshot;
    order: SectAuthorityOrderSnapshot;
    goals: SectAuthorityGoalsStateSnapshot;
    events: SectAuthorityEventStateSnapshot;
};

export type SectAuthorityEventLogEntry = {
    event_id: string;
    version: number;
    event_type: string;
    summary: string;
    related_day: number;
    related_tick: number;
    command_id?: string;
    replay_source: 'event_log' | string;
};

export type SectAuthorityDiaryEntry = SectAuthorityEventLogEntry;

export type SectAuthorityEventFeedbackEntry = SectAuthorityEventLogEntry & {
    category: string;
};

export type SectAuthoritySnapshot = {
    sectId: string;
    userId: string;
    sessionId: string;
    sceneVersion: number;
    state: SectAuthorityStateSnapshot;
    buildingCatalog?: SectAuthorityBuildingCatalogEntry[];
    taskDispatch?: Record<string, SectAuthorityTaskDispatchProjection>;
    eventLog?: SectAuthorityEventLogEntry[];
    diary?: SectAuthorityDiaryEntry[];
    eventSummaries?: SectAuthorityEventFeedbackEntry[];
};

export type SectAuthorityJoinResponse = {
    snapshot: SectAuthoritySnapshot;
};

export type SectAuthorityCommandError = {
    code: string;
    message: string;
    retriable: boolean;
};

export type SectAuthorityPatchOp = {
    op: string;
    path: string;
    value: string;
    valueEncoding: string;
};

export type SectAuthorityCommandResult = {
    cmdId: string;
    status: SectAuthorityCommandStatus;
    error?: SectAuthorityCommandError;
    sectId: string;
    sceneVersion: number;
    patch: {
        sectId: string;
        fromVersion: number;
        toVersion: number;
        ops?: SectAuthorityPatchOp[];
    };
};

export type SectAuthorityCommandResponse = {
    result: SectAuthorityCommandResult;
    snapshot: SectAuthoritySnapshot;
};

export type SectAuthorityPublishTaskPayload = {
    kind: string;
    type?: SectAuthorityTaskType;
    grade?: SectAuthorityTaskGrade;
    title: string;
    description?: string;
    priority?: number;
    requiredProgressDays: number;
    risk?: number;
    maxAssignees?: number;
    minIdentity?: string;
    minRealm?: string;
    requiredAptitude?: Partial<SectAuthorityDiscipleAptitudeSnapshot>;
    dispatchCost?: Partial<Record<SectAuthorityResourceKind, number>>;
    contributionReward: number;
    rewardResources?: Partial<Record<SectAuthorityResourceKind, number>>;
    reputationReward?: number;
    relationReward?: Record<string, number>;
    crisisClue?: string;
};

export type SectAuthorityBuildBuildingPayload = {
    definitionKey: string;
    origin: SectAuthorityTileCoord;
};

export type SectAuthorityUpgradeBuildingPayload = {
    buildingId: string;
};

export type SectAuthorityRepairBuildingPayload = {
    buildingId: string;
};

export type SectAuthorityCancelTaskPayload = {
    taskId: string;
};

export type SectAuthorityAssignDiscipleTaskPayload = {
    taskId: string;
    discipleId?: string;
    discipleIds?: string[];
};

export type SectAuthoritySetTaskPriorityPayload = {
    taskId: string;
    priority: number;
};

export type SectAuthorityExchangeContributionItemPayload = {
    discipleId: string;
    exchangeItemId: string;
    quantity: number;
};

export type SectAuthorityStartProductionPayload = {
    recipeId: string;
    priority?: number;
    targetCycles?: number;
};

export type SectAuthorityCancelProductionPayload = {
    productionId: string;
};

export type SectAuthorityAdjustProductionPayload = {
    productionId: string;
    priority?: number;
    targetCycles?: number;
};

export type SectAuthorityStartCultivationPayload = {
    discipleId: string;
};

export type SectAuthorityUsePillForCultivationPayload = {
    discipleId: string;
    pillType: 'cultivation_pill' | 'breakthrough_pill' | 'calm_mind_pill' | string;
    quantity: number;
};

export type SectAuthorityReserveCavePayload = {
    discipleId: string;
    durationDays?: number;
};

export type SectAuthorityAttemptBreakthroughPayload = {
    discipleId: string;
    usePills?: Record<string, number>;
    useSpiritStone?: number;
    caveBuildingId?: string;
    protectorDiscipleId?: string;
};

export type SectAuthorityStartRecruitmentPayload = {
    candidateCount: number;
    investmentSpiritStone?: number;
    durationDays?: number;
};

export type SectAuthorityAcceptCandidatePayload = {
    candidateId: string;
};

export type SectAuthorityRejectCandidatePayload = {
    candidateId: string;
};

export type SectAuthorityChooseEventOptionPayload = {
    eventId: string;
    optionId: string;
};

export type SectAuthorityDismissEventPayload = {
    eventId: string;
};

export type SectAuthorityStartAssessmentPayload = {
    discipleId: string;
    targetRank?: string;
    target_rank?: string;
};

export type SectAuthorityPromoteDisciplePayload = {
    discipleId: string;
    targetRank?: string;
    target_rank?: string;
};

export type SectAuthoritySetPolicyPayload = {
    policyCategory: SectAuthorityPolicyCategory;
    policyValue: string;
    policy_category?: SectAuthorityPolicyCategory;
    policy_value?: string;
};

export type SectAuthorityAssignInstitutionManagerPayload = {
    institutionId: string;
    discipleId: string;
};

export type SectAuthoritySetGatePolicyPayload = {
    openToVisitors?: boolean;
    allowWanderingCultivators?: boolean;
    guardDiscipleIds?: string[];
};

export type SectAuthoritySetExchangeRulePayload = {
    exchangeItemId: string;
    contributionCost?: number;
    monthlyLimit?: number;
    enabled?: boolean;
};

export type SectAuthorityCraftArtifactPayload = {
    artifactType: SectAuthorityArtifactType;
    type?: SectAuthorityArtifactType;
    quality?: number;
};

export type SectAuthorityEquipArtifactPayload = {
    itemId: string;
    discipleId: string;
};

export type SectAuthorityUnequipArtifactPayload = {
    itemId: string;
    discipleId?: string;
};

export type SectAuthorityRepairArtifactPayload = {
    itemId: string;
};

export type SectAuthorityCommandEnvelope<TPayload> = {
    cmdId: string;
    type: SectAuthorityCommandType;
    baseVersion: number;
    payload: TPayload;
};
