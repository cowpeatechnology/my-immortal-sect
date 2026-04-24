package sect

// SectState owns every authoritative gameplay module for one sect.
// Disciple, building, task, institution, and active storylet instances remain
// internal entries in this big-state and must not become standalone V1 actors.

type Version uint64

const (
	SectStateSchemaVersion     uint32 = 1
	SectStateSimulationVersion uint32 = 1
	SectStateConfigVersion     uint32 = 1
)

type SectID string
type UserID string
type DiscipleID string
type BuildingID string
type TaskID string
type ProductionID string
type RecipeID string
type StoryletID string
type EventID string
type SectGoalID string
type InstitutionID string
type ResourceNodeID string
type ItemID string
type CandidateID string
type ResourceKind string
type PillType string
type DiscipleAssignmentKind string
type IdentityRank string
type ExchangeItemID string
type ExchangeItemKind string
type ProductionKind string
type ProductionStatus string
type RealmStage string
type SectEventStatus string
type RecruitmentType string
type TaskStatus string
type TaskType string
type TaskGrade string
type TaskEvaluation string
type ArtifactType string
type ArtifactSlot string
type FormationKind string
type PolicyCategory string
type TaskPolicy string
type ResourcePolicy string
type RecruitmentPolicy string
type CultivationPolicy string
type SectGoalStatus string

const (
	ResourceKindSpiritStone  ResourceKind = "spirit_stone"
	ResourceKindSpiritGrain  ResourceKind = "spirit_grain"
	ResourceKindHerb         ResourceKind = "herb"
	ResourceKindOre          ResourceKind = "ore"
	ResourceKindBeastMat     ResourceKind = "beast_mat"
	ResourceKindFormationMat ResourceKind = "formation_mat"

	PillCultivation  PillType = "cultivation_pill"
	PillBreakthrough PillType = "breakthrough_pill"
	PillCalmMind     PillType = "calm_mind_pill"

	DiscipleAssignmentIdle        DiscipleAssignmentKind = "idle"
	DiscipleAssignmentTask        DiscipleAssignmentKind = "task_hall"
	DiscipleAssignmentCultivation DiscipleAssignmentKind = "cultivation"

	IdentityRankOuter IdentityRank = "outer_disciple"
	IdentityRankInner IdentityRank = "inner_disciple"

	RecruitmentTypeOpenMountain RecruitmentType = "open_mountain"

	ProductionKindFarm       ProductionKind = "farm"
	ProductionKindMining     ProductionKind = "mining"
	ProductionKindRefinement ProductionKind = "refinement"

	ProductionStatusRunning   ProductionStatus = "running"
	ProductionStatusBlocked   ProductionStatus = "blocked"
	ProductionStatusCompleted ProductionStatus = "completed"
	ProductionStatusCancelled ProductionStatus = "cancelled"

	RealmMortal     RealmStage = "mortal"
	RealmQiEntry    RealmStage = "qi_entry"
	RealmQiEarly    RealmStage = "qi_early"
	RealmQiMiddle   RealmStage = "qi_middle"
	RealmQiLate     RealmStage = "qi_late"
	RealmFoundation RealmStage = "foundation"
	RealmGoldenCore RealmStage = "golden_core"

	SectEventStatusSeeded       SectEventStatus = "seeded"
	SectEventStatusForeshadowed SectEventStatus = "foreshadowed"
	SectEventStatusResolved     SectEventStatus = "resolved"
	SectEventStatusExpired      SectEventStatus = "expired"

	SectGoalStatusActive    SectGoalStatus = "active"
	SectGoalStatusCompleted SectGoalStatus = "completed"
	SectGoalStatusFailed    SectGoalStatus = "failed"

	TaskStatusPublished TaskStatus = "published"
	TaskStatusAccepted  TaskStatus = "accepted"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"

	TaskTypeInternal   TaskType = "internal"
	TaskTypeExternal   TaskType = "external"
	TaskTypeExplore    TaskType = "explore"
	TaskTypeCombat     TaskType = "combat"
	TaskTypeProduction TaskType = "production"

	TaskGradeDing    TaskGrade = "ding"
	TaskGradeBing    TaskGrade = "bing"
	TaskGradeYi      TaskGrade = "yi"
	TaskGradeJia     TaskGrade = "jia"
	TaskGradeSpecial TaskGrade = "special"

	TaskEvaluationExcellent TaskEvaluation = "excellent"
	TaskEvaluationGood      TaskEvaluation = "good"
	TaskEvaluationNormal    TaskEvaluation = "normal"
	TaskEvaluationPoor      TaskEvaluation = "poor"
	TaskEvaluationFailed    TaskEvaluation = "failed"

	ArtifactTypeSword          ArtifactType = "sword"
	ArtifactTypeRobe           ArtifactType = "robe"
	ArtifactTypeFarmTool       ArtifactType = "farm_tool"
	ArtifactTypeAlchemyFurnace ArtifactType = "alchemy_furnace"
	ArtifactTypeFormationDisk  ArtifactType = "formation_disk"

	ArtifactSlotWeapon  ArtifactSlot = "weapon"
	ArtifactSlotRobe    ArtifactSlot = "robe"
	ArtifactSlotTool    ArtifactSlot = "tool"
	ArtifactSlotSpecial ArtifactSlot = "special"

	FormationKindGatherSpirit FormationKind = "gather_spirit"
	FormationKindFieldGuard   FormationKind = "field_guard"
	FormationKindDefense      FormationKind = "defense"
	FormationKindCalmMind     FormationKind = "calm_mind"

	PolicyCategoryTask        PolicyCategory = "task"
	PolicyCategoryResource    PolicyCategory = "resource"
	PolicyCategoryRecruitment PolicyCategory = "recruitment"
	PolicyCategoryCultivation PolicyCategory = "cultivation"

	TaskPolicyStable            TaskPolicy = "stable"
	TaskPolicyRewardExternal    TaskPolicy = "reward_external"
	TaskPolicyProduction        TaskPolicy = "production"
	TaskPolicyCombat            TaskPolicy = "combat"
	TaskPolicyClosedCultivation TaskPolicy = "closed_cultivation"

	ResourcePolicySaving         ResourcePolicy = "saving"
	ResourcePolicyGenerous       ResourcePolicy = "generous"
	ResourcePolicyPillLimited    ResourcePolicy = "pill_limited"
	ResourcePolicyOpenExchange   ResourcePolicy = "open_exchange"
	ResourcePolicyWarPreparation ResourcePolicy = "war_preparation"

	RecruitmentPolicyBroad      RecruitmentPolicy = "broad"
	RecruitmentPolicySelective  RecruitmentPolicy = "selective"
	RecruitmentPolicyWandering  RecruitmentPolicy = "wandering"
	RecruitmentPolicyAffiliated RecruitmentPolicy = "affiliated"

	CultivationPolicyBalanced          CultivationPolicy = "balanced"
	CultivationPolicyClosedCultivation CultivationPolicy = "closed_cultivation"
	CultivationPolicyBreakthroughSafe  CultivationPolicy = "breakthrough_safe"
)

type AuthorityBoundaryVersion struct {
	SchemaVersion     uint32 `json:"schema_version"`
	SimulationVersion uint32 `json:"simulation_version"`
	ConfigVersion     uint32 `json:"config_version"`
}

func DefaultAuthorityBoundaryVersion() AuthorityBoundaryVersion {
	return AuthorityBoundaryVersion{
		SchemaVersion:     SectStateSchemaVersion,
		SimulationVersion: SectStateSimulationVersion,
		ConfigVersion:     SectStateConfigVersion,
	}
}

type SectState struct {
	Meta              SectMetaState                  `json:"meta"`
	Runtime           SectRuntimeState               `json:"runtime"`
	Time              SectTimeState                  `json:"time"`
	Resources         ResourceState                  `json:"resources"`
	Contribution      ContributionState              `json:"contribution"`
	Inventory         InventoryState                 `json:"inventory"`
	Monthly           MonthlyState                   `json:"monthly"`
	Policies          PolicyState                    `json:"policies"`
	Disciples         map[DiscipleID]DiscipleState   `json:"disciples"`
	Admissions        AdmissionState                 `json:"admissions"`
	Buildings         map[BuildingID]BuildingState   `json:"buildings"`
	Formations        map[BuildingID]FormationState  `json:"formations"`
	Tasks             map[TaskID]TaskState           `json:"tasks"`
	Productions       map[ProductionID]ProductionJob `json:"productions"`
	Institutions      InstitutionState               `json:"institutions"`
	Order             SectOrderState                 `json:"order"`
	Goals             SectGoalsState                 `json:"goals"`
	Events            SectEventState                 `json:"events"`
	MonthlyAssessment MonthlyAssessmentState         `json:"monthly_assessment"`
	Storylets         map[StoryletID]StoryletState   `json:"storylets"`
	Flags             map[string]bool                `json:"flags"`
}

type SectMetaState struct {
	SectID               SectID         `json:"sect_id"`
	OwnerUserID          UserID         `json:"owner_user_id"`
	Name                 string         `json:"name"`
	CreatedAtUnixSeconds int64          `json:"created_at_unix_seconds"`
	Level                int32          `json:"level"`
	Expansion            int32          `json:"expansion"`
	BuildingLimit        int32          `json:"building_limit"`
	Reputation           int            `json:"reputation"`
	Relations            map[string]int `json:"relations,omitempty"`
}

type SectRuntimeState struct {
	Version                 Version                  `json:"version"`
	LastAppliedEventVersion Version                  `json:"last_applied_event_version"`
	LastSnapshotVersion     Version                  `json:"last_snapshot_version"`
	AuthorityBoundary       AuthorityBoundaryVersion `json:"authority_boundary"`
	RNG                     RNGState                 `json:"rng"`
	Dirty                   bool                     `json:"dirty"`
}

type SectTimeState struct {
	GameTick    int64 `json:"game_tick"`
	CalendarDay int32 `json:"calendar_day"`
	DayTick     int32 `json:"day_tick"`
	SeasonIndex int32 `json:"season_index"`
}

type RNGState struct {
	Seed   uint64 `json:"seed"`
	Cursor uint64 `json:"cursor"`
}

type ResourceState struct {
	Stock map[ResourceKind]int64               `json:"stock"`
	Nodes map[ResourceNodeID]ResourceNodeState `json:"nodes"`
}

type ResourceNodeState struct {
	NodeID               ResourceNodeID `json:"node_id"`
	Kind                 ResourceKind   `json:"kind"`
	Tile                 TileCoord      `json:"tile"`
	RemainingCharges     int64          `json:"remaining_charges"`
	MaxCharges           int64          `json:"max_charges"`
	RegenerateAtGameTick int64          `json:"regenerate_at_game_tick"`
}

type InventoryState struct {
	Items     map[ItemID]InventoryEntry `json:"items"`
	Artifacts map[ItemID]ArtifactState  `json:"artifacts"`
}

type InventoryEntry struct {
	ItemID    ItemID `json:"item_id"`
	Kind      string `json:"kind"`
	Quantity  int64  `json:"quantity"`
	Bound     bool   `json:"bound"`
	SourceTag string `json:"source_tag"`
}

type ArtifactState struct {
	ItemID             ItemID         `json:"item_id"`
	Type               ArtifactType   `json:"type"`
	Quality            int            `json:"quality"`
	Durability         int            `json:"durability"`
	MaxDurability      int            `json:"max_durability"`
	BoundDiscipleID    DiscipleID     `json:"bound_disciple_id,omitempty"`
	AttachedBuildingID BuildingID     `json:"attached_building_id,omitempty"`
	Stats              map[string]int `json:"stats,omitempty"`
	SourceTag          string         `json:"source_tag,omitempty"`
}

type EquipmentState struct {
	Weapon  ItemID `json:"weapon,omitempty"`
	Robe    ItemID `json:"robe,omitempty"`
	Tool    ItemID `json:"tool,omitempty"`
	Special ItemID `json:"special,omitempty"`
}

type ContributionState struct {
	Accounts                map[DiscipleID]ContributionAccount      `json:"accounts"`
	TreasuryRules           map[ExchangeItemID]ExchangeRule         `json:"treasury_rules"`
	MonthlyPurchases        map[DiscipleID]map[ExchangeItemID]int64 `json:"monthly_purchases"`
	RedeemabilityRatio      float64                                 `json:"redeemability_ratio"`
	OutstandingContribution int64                                   `json:"outstanding_contribution"`
	TreasuryValue           int64                                   `json:"treasury_value"`
}

type ContributionAccount struct {
	DiscipleID  DiscipleID `json:"disciple_id"`
	Balance     int64      `json:"balance"`
	EarnedTotal int64      `json:"earned_total"`
	SpentTotal  int64      `json:"spent_total"`
}

type MonthlyState struct {
	LastSettledMonth int64                    `json:"last_settled_month"`
	Payroll          PayrollState             `json:"payroll"`
	Obligations      MonthlyObligationState   `json:"obligations"`
	LastSettlement   MonthlySettlementSummary `json:"last_settlement"`
}

type PayrollState struct {
	LastPaidMonth int64              `json:"last_paid_month"`
	Arrears       map[DiscipleID]int `json:"arrears"`
}

type MonthlyObligationState struct {
	MonthIndex    int64              `json:"month_index"`
	CompletedDays map[DiscipleID]int `json:"completed_days"`
	RequiredDays  map[DiscipleID]int `json:"required_days"`
	Violations    map[DiscipleID]int `json:"violations"`
}

type MonthlySettlementSummary struct {
	MonthIndex             int64   `json:"month_index"`
	StipendPaid            int64   `json:"stipend_paid"`
	StipendDelayed         int64   `json:"stipend_delayed"`
	PayrollPaidCount       int     `json:"payroll_paid_count"`
	PayrollDelayedCount    int     `json:"payroll_delayed_count"`
	DutyRequiredDays       int     `json:"duty_required_days"`
	DutyCompletedDays      int     `json:"duty_completed_days"`
	DutyViolations         int     `json:"duty_violations"`
	RedeemabilityRatio     float64 `json:"redeemability_ratio"`
	ResourceShortage       bool    `json:"resource_shortage"`
	ContributionShortage   bool    `json:"contribution_shortage"`
	SatisfactionDeltaTotal int     `json:"satisfaction_delta_total"`
	LoyaltyDeltaTotal      int     `json:"loyalty_delta_total"`
}

type PolicyState struct {
	TaskPolicy        TaskPolicy              `json:"task_policy"`
	ResourcePolicy    ResourcePolicy          `json:"resource_policy"`
	RecruitmentPolicy RecruitmentPolicy       `json:"recruitment_policy"`
	CultivationPolicy CultivationPolicy       `json:"cultivation_policy"`
	CustomFlags       map[string]bool         `json:"custom_flags,omitempty"`
	Presentation      PolicyPresentationState `json:"presentation"`
}

type PolicyPresentationState struct {
	Categories map[PolicyCategory]PolicyPresentationCategory `json:"categories"`
}

type PolicyPresentationCategory struct {
	Category      PolicyCategory        `json:"category"`
	CurrentValue  string                `json:"current_value"`
	CurrentLabel  string                `json:"current_label"`
	Explanation   string                `json:"explanation"`
	ImpactSummary []string              `json:"impact_summary"`
	Options       []PolicyOptionSummary `json:"options"`
}

type PolicyOptionSummary struct {
	Value         string   `json:"value"`
	Label         string   `json:"label"`
	Explanation   string   `json:"explanation"`
	ImpactSummary []string `json:"impact_summary"`
}

type ExchangeRule struct {
	ExchangeItemID   ExchangeItemID   `json:"exchange_item_id"`
	Name             string           `json:"name"`
	ItemKind         ExchangeItemKind `json:"item_kind"`
	ItemRef          string           `json:"item_ref"`
	ContributionCost int64            `json:"contribution_cost"`
	MonthlyLimit     int64            `json:"monthly_limit"`
	StockLimit       int64            `json:"stock_limit"`
	Enabled          bool             `json:"enabled"`
}

type DiscipleState struct {
	DiscipleID     DiscipleID               `json:"disciple_id"`
	Name           string                   `json:"name"`
	Location       TileCoord                `json:"location"`
	Identity       IdentityRank             `json:"identity"`
	Aptitude       DiscipleAptitudeState    `json:"aptitude"`
	AssignmentKind DiscipleAssignmentKind   `json:"assignment_kind"`
	AssignmentTask *TaskID                  `json:"assignment_task,omitempty"`
	WorkTarget     DiscipleWorkTargetState  `json:"work_target"`
	Realm          RealmState               `json:"realm"`
	Needs          DiscipleNeedsState       `json:"needs"`
	Support        DiscipleSupportState     `json:"support"`
	Equipment      EquipmentState           `json:"equipment"`
	Satisfaction   int                      `json:"satisfaction"`
	Loyalty        int                      `json:"loyalty"`
	Carrying       *ResourceStack           `json:"carrying,omitempty"`
	Pressure       int                      `json:"pressure"`
	InjuryLevel    int                      `json:"injury_level"`
	HP             int64                    `json:"hp"`
	MaxHP          int64                    `json:"max_hp"`
	Cultivation    CultivationDecisionState `json:"cultivation_decision"`
	Assessment     DiscipleAssessmentState  `json:"assessment"`
	Memories       []DiscipleMemoryEntry    `json:"memories"`
	Relationship   []string                 `json:"relationship_tags"`
	Emotion        []string                 `json:"emotion_tags"`
	RecentSummary  []string                 `json:"recent_experience_summary"`
}

type DiscipleAssessmentState struct {
	TargetRank        IdentityRank `json:"target_rank,omitempty"`
	Passed            bool         `json:"passed"`
	Score             int          `json:"score"`
	Reason            string       `json:"reason,omitempty"`
	ResolvedAtVersion Version      `json:"resolved_at_version,omitempty"`
}

type DiscipleMemoryEntry struct {
	Kind              string          `json:"kind"`
	Summary           string          `json:"summary"`
	SourceEventType   DomainEventType `json:"source_event_type"`
	RecordedAtVersion Version         `json:"recorded_at_version"`
	RecordedAtDay     int32           `json:"recorded_at_day"`
	Intensity         int             `json:"intensity"`
	Tags              []string        `json:"tags"`
}

type CultivationDecisionState struct {
	DailyGain                   int64  `json:"daily_gain"`
	RequiredPoints              int64  `json:"required_points"`
	ProgressPercent             int    `json:"progress_percent"`
	EnvironmentBonus            int    `json:"environment_bonus"`
	CultivationPillAvailable    int64  `json:"cultivation_pill_available"`
	BreakthroughPillAvailable   int64  `json:"breakthrough_pill_available"`
	BreakthroughSpiritStoneCost int64  `json:"breakthrough_spirit_stone_cost"`
	BreakthroughSuccessRate     int    `json:"breakthrough_success_rate"`
	BreakthroughRisk            int    `json:"breakthrough_risk"`
	BreakthroughRiskLimit       int    `json:"breakthrough_risk_limit"`
	OmenStatus                  string `json:"omen_status"`
	OmenText                    string `json:"omen_text,omitempty"`
}

type DiscipleAptitudeState struct {
	SpiritRoot    int `json:"spirit_root"`
	Comprehension int `json:"comprehension"`
	Physique      int `json:"physique"`
	Mind          int `json:"mind"`
	Luck          int `json:"luck"`
}

type RealmState struct {
	Stage                   RealmStage `json:"stage"`
	CultivationPoints       int64      `json:"cultivation_points"`
	ReadyForBreakthrough    bool       `json:"ready_for_breakthrough"`
	FailedBreakthroughCount int        `json:"failed_breakthrough_count"`
}

type DiscipleWorkTargetState struct {
	TaskID      *TaskID     `json:"task_id,omitempty"`
	BuildingID  *BuildingID `json:"building_id,omitempty"`
	StoryletID  *StoryletID `json:"storylet_id,omitempty"`
	Description string      `json:"description,omitempty"`
}

type DiscipleNeedsState struct {
	DailySpiritGrain int64 `json:"daily_spirit_grain"`
	DailyRestTicks   int64 `json:"daily_rest_ticks"`
}

type DiscipleSupportState struct {
	FoodSatisfied    bool `json:"food_satisfied"`
	HousingSatisfied bool `json:"housing_satisfied"`
	MedicalSupported bool `json:"medical_supported"`
}

type AdmissionState struct {
	CurrentRecruitment        *RecruitmentSession            `json:"current_recruitment,omitempty"`
	Candidates                map[CandidateID]CandidateState `json:"candidates"`
	LastAnnualRecruitmentYear int32                          `json:"last_annual_recruitment_year"`
}

type RecruitmentSession struct {
	RecruitmentID         string          `json:"recruitment_id"`
	Type                  RecruitmentType `json:"type"`
	StartedAtCalendarDay  int32           `json:"started_at_calendar_day"`
	EndsAtCalendarDay     int32           `json:"ends_at_calendar_day"`
	InvestmentSpiritStone int64           `json:"investment_spirit_stone"`
	CandidateCount        int             `json:"candidate_count"`
}

type CandidateState struct {
	CandidateID CandidateID           `json:"candidate_id"`
	Name        string                `json:"name"`
	Source      RecruitmentType       `json:"source"`
	Identity    IdentityRank          `json:"identity"`
	Aptitude    DiscipleAptitudeState `json:"aptitude"`
	Realm       RealmState            `json:"realm"`
	Needs       DiscipleNeedsState    `json:"needs"`
	Support     DiscipleSupportState  `json:"support"`
	Pressure    int                   `json:"pressure"`
	InjuryLevel int                   `json:"injury_level"`
	HP          int64                 `json:"hp"`
	MaxHP       int64                 `json:"max_hp"`
}

type BuildingState struct {
	BuildingID      BuildingID     `json:"building_id"`
	DefinitionKey   string         `json:"definition_key"`
	Level           int32          `json:"level"`
	Phase           string         `json:"phase"`
	Origin          TileCoord      `json:"origin"`
	HP              int64          `json:"hp"`
	MaxHP           int64          `json:"max_hp"`
	Durability      int            `json:"durability"`
	Efficiency      int            `json:"efficiency"`
	MaintenanceDebt int            `json:"maintenance_debt"`
	DamagedReason   string         `json:"damaged_reason,omitempty"`
	InstitutionID   *InstitutionID `json:"institution_id,omitempty"`
	ActiveTaskIDs   []TaskID       `json:"active_task_ids,omitempty"`
	AssignedWorkers []DiscipleID   `json:"assigned_workers,omitempty"`
}

type FormationState struct {
	FormationID       string                 `json:"formation_id"`
	Kind              FormationKind          `json:"kind"`
	BuildingID        BuildingID             `json:"building_id"`
	ArtifactItemID    ItemID                 `json:"artifact_item_id"`
	Level             int                    `json:"level"`
	Stability         int                    `json:"stability"`
	MaintenanceDebt   int                    `json:"maintenance_debt"`
	Active            bool                   `json:"active"`
	AttachedAtVersion Version                `json:"attached_at_version"`
	MaintenanceCost   map[ResourceKind]int64 `json:"maintenance_cost,omitempty"`
	EffectSummary     []string               `json:"effect_summary,omitempty"`
}

type TaskState struct {
	TaskID                TaskID                 `json:"task_id"`
	Kind                  string                 `json:"kind"`
	Type                  TaskType               `json:"type,omitempty"`
	Grade                 TaskGrade              `json:"grade,omitempty"`
	Title                 string                 `json:"title"`
	Description           string                 `json:"description,omitempty"`
	Status                TaskStatus             `json:"status"`
	Priority              int                    `json:"priority"`
	AssignedDiscipleIDs   []DiscipleID           `json:"assigned_disciple_ids,omitempty"`
	TargetBuildingID      *BuildingID            `json:"target_building_id,omitempty"`
	TargetStoryletID      *StoryletID            `json:"target_storylet_id,omitempty"`
	ProgressTicks         int64                  `json:"progress_ticks"`
	RequiredProgressDays  int32                  `json:"required_progress_days"`
	CompletedProgressDays int32                  `json:"completed_progress_days"`
	Risk                  int                    `json:"risk"`
	MaxAssignees          int                    `json:"max_assignees"`
	MinIdentity           IdentityRank           `json:"min_identity,omitempty"`
	MinRealm              RealmStage             `json:"min_realm,omitempty"`
	RequiredAptitude      DiscipleAptitudeState  `json:"required_aptitude"`
	DispatchCost          map[ResourceKind]int64 `json:"dispatch_cost,omitempty"`
	ContributionReward    int64                  `json:"contribution_reward"`
	RewardResources       map[ResourceKind]int64 `json:"reward_resources,omitempty"`
	ReputationReward      int                    `json:"reputation_reward,omitempty"`
	RelationReward        map[string]int         `json:"relation_reward,omitempty"`
	CrisisClue            string                 `json:"crisis_clue,omitempty"`
	SuccessRate           int                    `json:"success_rate,omitempty"`
	Evaluation            TaskEvaluation         `json:"evaluation,omitempty"`
}

type ProductionJob struct {
	ProductionID         ProductionID           `json:"production_id"`
	Kind                 ProductionKind         `json:"kind"`
	BuildingID           BuildingID             `json:"building_id"`
	RecipeID             RecipeID               `json:"recipe_id"`
	Status               ProductionStatus       `json:"status"`
	Priority             int                    `json:"priority"`
	TargetCycles         int32                  `json:"target_cycles,omitempty"`
	AssignedDisciples    []DiscipleID           `json:"assigned_disciples,omitempty"`
	InputCost            map[ResourceKind]int64 `json:"input_cost,omitempty"`
	OutputReward         map[ResourceKind]int64 `json:"output_reward,omitempty"`
	ProgressDays         int32                  `json:"progress_days"`
	RequiredProgressDays int32                  `json:"required_progress_days"`
	CompletedCycles      int32                  `json:"completed_cycles"`
	BlockedReason        string                 `json:"blocked_reason,omitempty"`
	Shortage             map[ResourceKind]int64 `json:"shortage,omitempty"`
}

type ProductionRecipe struct {
	RecipeID             RecipeID
	Kind                 ProductionKind
	BuildingID           BuildingID
	DefaultPriority      int
	InputCost            map[ResourceKind]int64
	OutputReward         map[ResourceKind]int64
	RequiredProgressDays int32
}

type InstitutionState struct {
	ByID map[InstitutionID]InstitutionLoopState `json:"by_id"`
}

type InstitutionLoopState struct {
	InstitutionID       InstitutionID                 `json:"institution_id"`
	Kind                string                        `json:"kind"`
	Level               int32                         `json:"level"`
	Enabled             bool                          `json:"enabled"`
	ManagerDiscipleID   *DiscipleID                   `json:"manager_disciple_id,omitempty"`
	ManagerEffect       InstitutionManagerEffectState `json:"manager_effect"`
	Capacity            int                           `json:"capacity"`
	Comfort             int                           `json:"comfort"`
	HealingPower        int                           `json:"healing_power"`
	CultivationSupport  int                           `json:"cultivation_support"`
	TaskCapacityBonus   int                           `json:"task_capacity_bonus"`
	ExchangePressure    int                           `json:"exchange_pressure"`
	Efficiency          int                           `json:"efficiency"`
	EffectSummary       []string                      `json:"effect_summary,omitempty"`
	GatePolicy          GatePolicyState               `json:"gate_policy,omitempty"`
	PublicExchange      bool                          `json:"public_exchange_enabled,omitempty"`
	CaveSlots           []CaveSlotState               `json:"cave_slots,omitempty"`
	AssignedBuildingIDs []BuildingID                  `json:"assigned_building_ids,omitempty"`
	ActiveTaskIDs       []TaskID                      `json:"active_task_ids,omitempty"`
}

type InstitutionManagerEffectState struct {
	ManagerScore    int `json:"manager_score"`
	IdentityBonus   int `json:"identity_bonus"`
	AptitudeBonus   int `json:"aptitude_bonus"`
	LoyaltyModifier int `json:"loyalty_modifier"`
	InjuryPenalty   int `json:"injury_penalty"`
	EfficiencyBonus int `json:"efficiency_bonus"`
}

type GatePolicyState struct {
	OpenToVisitors            bool         `json:"open_to_visitors"`
	AllowWanderingCultivators bool         `json:"allow_wandering_cultivators"`
	EnforcementStrictness     int          `json:"enforcement_strictness"`
	GuardDiscipleIDs          []DiscipleID `json:"guard_disciple_ids,omitempty"`
}

type SectOrderState struct {
	Safety             int      `json:"safety"`
	Discipline         int      `json:"discipline"`
	InternalStrifeRisk int      `json:"internal_strife_risk"`
	Summary            []string `json:"summary,omitempty"`
	LastUpdatedVersion Version  `json:"last_updated_version,omitempty"`
}

type MonthlyAssessmentState struct {
	LastMonthIndex int64                     `json:"last_month_index"`
	Latest         *MonthlyAssessmentResult  `json:"latest,omitempty"`
	History        []MonthlyAssessmentResult `json:"history,omitempty"`
}

type MonthlyAssessmentResult struct {
	MonthIndex         int64       `json:"month_index"`
	ChampionDiscipleID *DiscipleID `json:"champion_disciple_id,omitempty"`
	ChampionName       string      `json:"champion_name,omitempty"`
	Score              int         `json:"score"`
	RewardContribution int64       `json:"reward_contribution,omitempty"`
	RewardReputation   int         `json:"reward_reputation,omitempty"`
	PromotionMomentum  int         `json:"promotion_momentum,omitempty"`
	Summary            string      `json:"summary"`
	ResolvedAtVersion  Version     `json:"resolved_at_version"`
}

type CaveSlotState struct {
	SlotID           string      `json:"slot_id"`
	OccupiedBy       *DiscipleID `json:"occupied_by,omitempty"`
	ReservedUntilDay int32       `json:"reserved_until_day,omitempty"`
	EnvironmentBonus int         `json:"environment_bonus"`
}

type SectGoalsState struct {
	ByID     map[SectGoalID]SectGoal   `json:"by_id"`
	Resolved []ResolvedSectGoalSummary `json:"resolved"`
}

type SectGoal struct {
	GoalID              SectGoalID             `json:"goal_id"`
	Kind                string                 `json:"kind"`
	Title               string                 `json:"title"`
	Description         string                 `json:"description,omitempty"`
	Status              SectGoalStatus         `json:"status"`
	CurrentProgress     int32                  `json:"current_progress"`
	TargetProgress      int32                  `json:"target_progress"`
	ProgressText        string                 `json:"progress_text,omitempty"`
	RewardResources     map[ResourceKind]int64 `json:"reward_resources,omitempty"`
	RewardReputation    int                    `json:"reward_reputation,omitempty"`
	RewardSatisfaction  int                    `json:"reward_satisfaction,omitempty"`
	FailureReputation   int                    `json:"failure_reputation,omitempty"`
	FailureSatisfaction int                    `json:"failure_satisfaction,omitempty"`
	RewardSummary       []string               `json:"reward_summary,omitempty"`
	FocusDiscipleID     *DiscipleID            `json:"focus_disciple_id,omitempty"`
	OutcomeSummary      string                 `json:"outcome_summary,omitempty"`
	StartedAtVersion    Version                `json:"started_at_version,omitempty"`
	ResolvedAtVersion   Version                `json:"resolved_at_version,omitempty"`
	Tags                []string               `json:"tags,omitempty"`
}

type ResolvedSectGoalSummary struct {
	GoalID            SectGoalID `json:"goal_id"`
	Kind              string     `json:"kind"`
	Outcome           string     `json:"outcome"`
	Summary           string     `json:"summary"`
	ResolvedAtVersion Version    `json:"resolved_at_version"`
}

type SectEventState struct {
	ActiveEvents          map[EventID]SectEvent  `json:"active_events"`
	ResolvedEvents        []ResolvedEventSummary `json:"resolved_events"`
	Tension               int                    `json:"tension"`
	LastMajorEventVersion Version                `json:"last_major_event_version"`
}

type SectEvent struct {
	EventID           EventID                `json:"event_id"`
	Kind              string                 `json:"kind"`
	Status            SectEventStatus        `json:"status"`
	Severity          int                    `json:"severity"`
	Title             string                 `json:"title"`
	Description       string                 `json:"description"`
	OmenText          string                 `json:"omen_text,omitempty"`
	ChainID           string                 `json:"chain_id,omitempty"`
	ChainStage        string                 `json:"chain_stage,omitempty"`
	SourceDiscipleID  *DiscipleID            `json:"source_disciple_id,omitempty"`
	SeededAtVersion   Version                `json:"seeded_at_version"`
	RevealAtVersion   Version                `json:"reveal_at_version"`
	RevealedAtVersion Version                `json:"revealed_at_version"`
	ResolvedAtVersion Version                `json:"resolved_at_version"`
	ExpiresAtDay      int32                  `json:"expires_at_day,omitempty"`
	Requirements      SectEventRequirement   `json:"requirements,omitempty"`
	Options           []SectEventOption      `json:"options,omitempty"`
	ResultPreview     SectEventResultPreview `json:"result_preview,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
}

type SectEventRequirement struct {
	MinResources       map[ResourceKind]int64 `json:"min_resources,omitempty"`
	RequiredDiscipleID *DiscipleID            `json:"required_disciple_id,omitempty"`
}

type SectEventOption struct {
	OptionID      string                 `json:"option_id"`
	Label         string                 `json:"label"`
	Description   string                 `json:"description,omitempty"`
	Requirements  SectEventRequirement   `json:"requirements,omitempty"`
	ResultPreview SectEventResultPreview `json:"result_preview"`
}

type SectEventResultPreview struct {
	ResourceDelta             map[ResourceKind]int64 `json:"resource_delta,omitempty"`
	DisciplePressureDelta     int                    `json:"disciple_pressure_delta,omitempty"`
	DiscipleSatisfactionDelta int                    `json:"disciple_satisfaction_delta,omitempty"`
	TaskID                    TaskID                 `json:"task_id,omitempty"`
	TaskTitle                 string                 `json:"task_title,omitempty"`
	FameDelta                 int                    `json:"fame_delta,omitempty"`
	TensionDelta              int                    `json:"tension_delta,omitempty"`
	Summary                   string                 `json:"summary,omitempty"`
}

type ResolvedEventSummary struct {
	EventID           EventID `json:"event_id"`
	Kind              string  `json:"kind"`
	Outcome           string  `json:"outcome"`
	Summary           string  `json:"summary"`
	ResolvedAtVersion Version `json:"resolved_at_version"`
}

type StoryletState struct {
	StoryletID       StoryletID   `json:"storylet_id"`
	DefinitionKey    string       `json:"definition_key"`
	Status           string       `json:"status"`
	Phase            string       `json:"phase"`
	BoundDiscipleIDs []DiscipleID `json:"bound_disciple_ids,omitempty"`
	BoundBuildingIDs []BuildingID `json:"bound_building_ids,omitempty"`
}

type ResourceStack struct {
	Kind   ResourceKind `json:"kind"`
	Amount int64        `json:"amount"`
}

type TileCoord struct {
	Col int `json:"col"`
	Row int `json:"row"`
}

const (
	starterDiscipleID    DiscipleID = "disciple-1"
	starterDiscipleName             = "starter_disciple"
	starterDiscipleMaxHP int64      = 100
)

func NewInitialSectState(sectID SectID, ownerUserID UserID, name string) SectState {
	state := SectState{
		Meta: SectMetaState{
			SectID:        sectID,
			OwnerUserID:   ownerUserID,
			Name:          name,
			Level:         1,
			Expansion:     0,
			BuildingLimit: sectBuildingLimitForLevel(1),
			Relations:     map[string]int{},
		},
		Runtime: SectRuntimeState{
			Version:                 0,
			LastAppliedEventVersion: 0,
			LastSnapshotVersion:     0,
			AuthorityBoundary:       DefaultAuthorityBoundaryVersion(),
		},
		Time: SectTimeState{},
		Resources: ResourceState{
			Stock: map[ResourceKind]int64{
				ResourceKindSpiritStone:  120,
				ResourceKindSpiritGrain:  60,
				ResourceKindHerb:         20,
				ResourceKindOre:          40,
				ResourceKindBeastMat:     10,
				ResourceKindFormationMat: 8,
			},
			Nodes: map[ResourceNodeID]ResourceNodeState{},
		},
		Contribution:      newInitialContributionState(),
		Inventory:         newInitialInventoryState(),
		Monthly:           newInitialMonthlyState(),
		Policies:          newInitialPolicyState(),
		Disciples:         map[DiscipleID]DiscipleState{},
		Admissions:        newInitialAdmissionState(),
		Buildings:         map[BuildingID]BuildingState{},
		Formations:        map[BuildingID]FormationState{},
		Tasks:             defaultTaskPool(),
		Productions:       defaultProductionJobs(),
		Institutions:      newInitialInstitutionState(),
		Order:             SectOrderState{},
		Goals:             defaultSectGoalState(),
		Events:            newInitialSectEventState(),
		MonthlyAssessment: MonthlyAssessmentState{},
		Storylets:         map[StoryletID]StoryletState{},
		Flags:             map[string]bool{},
	}
	ensureInventoryState(&state)
	ensureStarterDisciple(&state)
	ensureAdmissionState(&state)
	ensureContributionAccounts(&state)
	ensureMonthlyState(&state)
	ensureSectGoalState(&state)
	ensurePolicyState(&state)
	ensureInstitutionState(&state)
	applyPolicyEffectsToState(&state)
	ensureSectOrderState(&state)
	ensureMonthlyAssessmentState(&state)
	recalculateContributionMetrics(&state)
	return state
}

func newInitialInstitutionState() InstitutionState {
	return InstitutionState{ByID: defaultInstitutionLoops(nil)}
}

func newInitialPolicyState() PolicyState {
	policies := PolicyState{
		TaskPolicy:        TaskPolicyStable,
		ResourcePolicy:    ResourcePolicyOpenExchange,
		RecruitmentPolicy: RecruitmentPolicyBroad,
		CultivationPolicy: CultivationPolicyBalanced,
		CustomFlags:       map[string]bool{},
	}
	policies.Presentation = buildPolicyPresentation(policies)
	return policies
}

func defaultSectGoalState() SectGoalsState {
	return SectGoalsState{
		ByID: map[SectGoalID]SectGoal{
			SectGoalID("goal-cave-routine"): {
				GoalID:             SectGoalID("goal-cave-routine"),
				Kind:               "cave_routine",
				Title:              "洞府成形",
				Description:        "让首名弟子进入洞府闭关，证明宗门修炼设施开始真正运转。",
				Status:             SectGoalStatusActive,
				TargetProgress:     1,
				ProgressText:       "等待首名弟子入洞闭关。",
				RewardResources:    map[ResourceKind]int64{ResourceKindSpiritStone: 6},
				RewardReputation:   1,
				RewardSatisfaction: 3,
				RewardSummary:      []string{"奖励灵石 6", "名望 +1", "相关弟子满意 +3"},
				Tags:               []string{"cultivation", "cave"},
			},
			SectGoalID("goal-inner-disciple"): {
				GoalID:             SectGoalID("goal-inner-disciple"),
				Kind:               "inner_disciple",
				Title:              "培养内门",
				Description:        "完成一名弟子的考核与晋升，使宗门出现第一名内门弟子。",
				Status:             SectGoalStatusActive,
				TargetProgress:     1,
				ProgressText:       "尚未出现内门弟子。",
				RewardResources:    map[ResourceKind]int64{ResourceKindHerb: 3, ResourceKindSpiritStone: 8},
				RewardReputation:   3,
				RewardSatisfaction: 4,
				RewardSummary:      []string{"奖励灵植 3、灵石 8", "名望 +3", "晋升弟子满意 +4"},
				Tags:               []string{"promotion", "disciple_growth"},
			},
			SectGoalID("goal-external-affairs"): {
				GoalID:             SectGoalID("goal-external-affairs"),
				Kind:               "external_affairs",
				Title:              "完成外务",
				Description:        "完成至少一次外务、探索或战斗任务，证明宗门开始向山外伸手。",
				Status:             SectGoalStatusActive,
				TargetProgress:     1,
				ProgressText:       "尚未完成外务/探索/战斗。",
				RewardResources:    map[ResourceKind]int64{ResourceKindSpiritGrain: 6},
				RewardReputation:   2,
				RewardSatisfaction: 2,
				RewardSummary:      []string{"奖励灵粮 6", "名望 +2", "外务弟子满意 +2"},
				Tags:               []string{"task", "external"},
			},
			SectGoalID("goal-stable-monthly"): {
				GoalID:              SectGoalID("goal-stable-monthly"),
				Kind:                "stable_monthly",
				Title:               "稳定月结",
				Description:         "完成一次无欠发、无义务违约、无明显资源短缺的月结。",
				Status:              SectGoalStatusActive,
				TargetProgress:      1,
				ProgressText:        "等待首个完整月结。",
				RewardResources:     map[ResourceKind]int64{ResourceKindSpiritStone: 12, ResourceKindSpiritGrain: 8},
				RewardReputation:    4,
				RewardSatisfaction:  5,
				FailureReputation:   -3,
				FailureSatisfaction: -4,
				RewardSummary:       []string{"稳定成功：灵石 12、灵粮 8、名望 +4", "若月结失衡：名望 -3，弟子满意 -4"},
				Tags:                []string{"monthly", "stability"},
			},
		},
		Resolved: []ResolvedSectGoalSummary{},
	}
}

func ensureSectGoalState(state *SectState) {
	if state == nil {
		return
	}
	defaults := defaultSectGoalState()
	if state.Goals.ByID == nil {
		state.Goals.ByID = map[SectGoalID]SectGoal{}
	}
	for goalID, goal := range defaults.ByID {
		if _, exists := state.Goals.ByID[goalID]; !exists {
			state.Goals.ByID[goalID] = goal
		}
	}
	if state.Goals.Resolved == nil {
		state.Goals.Resolved = []ResolvedSectGoalSummary{}
	}
}

func ensureSectOrderState(state *SectState) {
	if state == nil {
		return
	}
	state.Order.Safety = clampInt(state.Order.Safety, 0, 100)
	state.Order.Discipline = clampInt(state.Order.Discipline, 0, 100)
	state.Order.InternalStrifeRisk = clampInt(state.Order.InternalStrifeRisk, 0, 100)
	if state.Order.Summary == nil {
		state.Order.Summary = []string{}
	}
	if state.Order.LastUpdatedVersion == 0 && len(state.Order.Summary) == 0 {
		state.Order = buildSectOrderProjection(*state, state.Runtime.Version)
	}
}

func ensureMonthlyAssessmentState(state *SectState) {
	if state == nil {
		return
	}
	if state.MonthlyAssessment.History == nil {
		state.MonthlyAssessment.History = []MonthlyAssessmentResult{}
	}
}

func newInitialInventoryState() InventoryState {
	return InventoryState{
		Artifacts: map[ItemID]ArtifactState{},
		Items: map[ItemID]InventoryEntry{
			ItemID("pill-cultivation"): {
				ItemID:    ItemID("pill-cultivation"),
				Kind:      string(PillCultivation),
				Quantity:  2,
				SourceTag: "starter",
			},
			ItemID("pill-breakthrough"): {
				ItemID:    ItemID("pill-breakthrough"),
				Kind:      string(PillBreakthrough),
				Quantity:  1,
				SourceTag: "starter",
			},
		},
	}
}

func newInitialMonthlyState() MonthlyState {
	return MonthlyState{
		Payroll: PayrollState{
			Arrears: map[DiscipleID]int{},
		},
		Obligations: MonthlyObligationState{
			MonthIndex:    1,
			CompletedDays: map[DiscipleID]int{},
			RequiredDays:  map[DiscipleID]int{},
			Violations:    map[DiscipleID]int{},
		},
	}
}

func newInitialAdmissionState() AdmissionState {
	return AdmissionState{
		Candidates: map[CandidateID]CandidateState{},
	}
}

func newInitialSectEventState() SectEventState {
	return SectEventState{
		ActiveEvents:   map[EventID]SectEvent{},
		ResolvedEvents: []ResolvedEventSummary{},
	}
}

func defaultTaskPool() map[TaskID]TaskState {
	return map[TaskID]TaskState{
		TaskID("pool-1"): {
			TaskID:               TaskID("pool-1"),
			Kind:                 "sect_patrol",
			Type:                 TaskTypeCombat,
			Grade:                TaskGradeDing,
			Title:                "巡山清障",
			Description:          "清理山门近处的杂妖痕迹，适合新入门弟子结队执行。",
			Status:               TaskStatusPublished,
			Priority:             60,
			RequiredProgressDays: 1,
			Risk:                 12,
			MaxAssignees:         2,
			MinIdentity:          IdentityRankOuter,
			MinRealm:             RealmMortal,
			RequiredAptitude:     DiscipleAptitudeState{Physique: 5},
			ContributionReward:   8,
			RewardResources:      map[ResourceKind]int64{ResourceKindSpiritStone: 4},
			ReputationReward:     1,
		},
		TaskID("pool-2"): {
			TaskID:               TaskID("pool-2"),
			Kind:                 "merchant_commission",
			Type:                 TaskTypeExternal,
			Grade:                TaskGradeDing,
			Title:                "商会委托",
			Description:          "替山下商会护送短程货物，结算灵石、名望与商会关系。",
			Status:               TaskStatusPublished,
			Priority:             48,
			RequiredProgressDays: 1,
			Risk:                 8,
			MaxAssignees:         1,
			MinIdentity:          IdentityRankOuter,
			MinRealm:             RealmMortal,
			RequiredAptitude:     DiscipleAptitudeState{Mind: 5},
			ContributionReward:   6,
			RewardResources:      map[ResourceKind]int64{ResourceKindSpiritStone: 10},
			ReputationReward:     2,
			RelationReward:       map[string]int{"merchant_guild": 4},
		},
		TaskID("pool-3"): {
			TaskID:               TaskID("pool-3"),
			Kind:                 "village_aid",
			Type:                 TaskTypeExternal,
			Grade:                TaskGradeDing,
			Title:                "山下村寨",
			Description:          "消耗少量灵石协助村寨修葺灵渠，换取民望与供给。",
			Status:               TaskStatusPublished,
			Priority:             52,
			RequiredProgressDays: 2,
			Risk:                 18,
			MaxAssignees:         2,
			MinIdentity:          IdentityRankOuter,
			MinRealm:             RealmMortal,
			RequiredAptitude:     DiscipleAptitudeState{SpiritRoot: 5},
			DispatchCost:         map[ResourceKind]int64{ResourceKindSpiritStone: 8},
			ContributionReward:   12,
			RewardResources:      map[ResourceKind]int64{ResourceKindSpiritGrain: 8, ResourceKindHerb: 3},
			ReputationReward:     3,
			RelationReward:       map[string]int{"mountain_village": 6},
		},
		TaskID("pool-4"): {
			TaskID:               TaskID("pool-4"),
			Kind:                 "demon_scout",
			Type:                 TaskTypeCombat,
			Grade:                TaskGradeBing,
			Title:                "妖踪探查",
			Description:          "高风险外勤，若队伍能力不足会由 authority 判定失败。",
			Status:               TaskStatusPublished,
			Priority:             40,
			RequiredProgressDays: 1,
			Risk:                 95,
			MaxAssignees:         1,
			MinIdentity:          IdentityRankOuter,
			MinRealm:             RealmMortal,
			RequiredAptitude:     DiscipleAptitudeState{Physique: 6},
			ContributionReward:   20,
			RewardResources:      map[ResourceKind]int64{ResourceKindBeastMat: 4},
			ReputationReward:     4,
			RelationReward:       map[string]int{"mountain_village": 2},
			CrisisClue:           "demon_trail",
		},
		TaskID("pool-5"): {
			TaskID:               TaskID("pool-5"),
			Kind:                 "ancient_road_explore",
			Type:                 TaskTypeExplore,
			Grade:                TaskGradeBing,
			Title:                "古道探索",
			Description:          "沿废弃古道查找遗落阵材，需要较高灵根与悟性。",
			Status:               TaskStatusPublished,
			Priority:             35,
			RequiredProgressDays: 2,
			Risk:                 35,
			MaxAssignees:         1,
			MinIdentity:          IdentityRankOuter,
			MinRealm:             RealmMortal,
			RequiredAptitude:     DiscipleAptitudeState{SpiritRoot: 10, Comprehension: 9},
			ContributionReward:   10,
			RewardResources:      map[ResourceKind]int64{ResourceKindFormationMat: 3, ResourceKindOre: 4},
			ReputationReward:     2,
			RelationReward:       map[string]int{"wandering_traders": 2},
			CrisisClue:           "ancient_road_omen",
		},
		TaskID("pool-6"): {
			TaskID:               TaskID("pool-6"),
			Kind:                 "combat_training",
			Type:                 TaskTypeCombat,
			Grade:                TaskGradeDing,
			Title:                "战斗历练",
			Description:          "组织短程战斗历练，带来兽材、经验与受伤风险。",
			Status:               TaskStatusPublished,
			Priority:             45,
			RequiredProgressDays: 1,
			Risk:                 42,
			MaxAssignees:         2,
			MinIdentity:          IdentityRankOuter,
			MinRealm:             RealmMortal,
			RequiredAptitude:     DiscipleAptitudeState{Physique: 6},
			ContributionReward:   14,
			RewardResources:      map[ResourceKind]int64{ResourceKindBeastMat: 2, ResourceKindSpiritStone: 3},
			ReputationReward:     2,
			CrisisClue:           "combat_wound_report",
		},
	}
}

func ensureAdmissionState(state *SectState) {
	if state == nil {
		return
	}
	if state.Admissions.Candidates == nil {
		state.Admissions.Candidates = map[CandidateID]CandidateState{}
	}
}

func ensurePolicyState(state *SectState) {
	if state == nil {
		return
	}
	if state.Policies.TaskPolicy == "" {
		state.Policies.TaskPolicy = TaskPolicyStable
	}
	if state.Policies.ResourcePolicy == "" {
		state.Policies.ResourcePolicy = ResourcePolicyOpenExchange
	}
	if state.Policies.RecruitmentPolicy == "" {
		state.Policies.RecruitmentPolicy = RecruitmentPolicyBroad
	}
	if state.Policies.CultivationPolicy == "" {
		state.Policies.CultivationPolicy = CultivationPolicyBalanced
	}
	if state.Policies.CustomFlags == nil {
		state.Policies.CustomFlags = map[string]bool{}
	}
	state.Policies.Presentation = buildPolicyPresentation(state.Policies)
}

func buildPolicyPresentation(policies PolicyState) PolicyPresentationState {
	return PolicyPresentationState{
		Categories: map[PolicyCategory]PolicyPresentationCategory{
			PolicyCategoryTask:        buildPolicyPresentationCategory(PolicyCategoryTask, string(policies.TaskPolicy)),
			PolicyCategoryResource:    buildPolicyPresentationCategory(PolicyCategoryResource, string(policies.ResourcePolicy)),
			PolicyCategoryRecruitment: buildPolicyPresentationCategory(PolicyCategoryRecruitment, string(policies.RecruitmentPolicy)),
			PolicyCategoryCultivation: buildPolicyPresentationCategory(PolicyCategoryCultivation, string(policies.CultivationPolicy)),
		},
	}
}

func buildPolicyPresentationCategory(category PolicyCategory, currentValue string) PolicyPresentationCategory {
	options := policyOptionSummaries(category)
	current := PolicyOptionSummary{Value: currentValue, Label: currentValue, Explanation: "authority 尚未提供该政策说明。"}
	for _, option := range options {
		if option.Value == currentValue {
			current = option
			break
		}
	}
	return PolicyPresentationCategory{
		Category:      category,
		CurrentValue:  current.Value,
		CurrentLabel:  current.Label,
		Explanation:   current.Explanation,
		ImpactSummary: append([]string{}, current.ImpactSummary...),
		Options:       options,
	}
}

func policyOptionSummaries(category PolicyCategory) []PolicyOptionSummary {
	switch category {
	case PolicyCategoryTask:
		return []PolicyOptionSummary{
			{Value: string(TaskPolicyStable), Label: "稳健经营", Explanation: "保持低风险宗务权重，压低危险外勤。", ImpactSummary: []string{"高风险任务优先级下降", "基础巡山与采买保持稳定"}},
			{Value: string(TaskPolicyRewardExternal), Label: "重赏外务", Explanation: "提高有奖励或有资源产出的外务任务权重。", ImpactSummary: []string{"奖励任务优先级上升", "灵石与外勤压力增加"}},
			{Value: string(TaskPolicyProduction), Label: "重产炼制", Explanation: "优先支持灵植、阵材和生产相关任务。", ImpactSummary: []string{"生产任务优先级上升", "高风险探查权重下降"}},
			{Value: string(TaskPolicyCombat), Label: "备战巡防", Explanation: "提高巡山和高风险战斗外勤权重。", ImpactSummary: []string{"巡山与战斗任务优先级上升", "非战斗任务权重下降"}},
			{Value: string(TaskPolicyClosedCultivation), Label: "闭门修持", Explanation: "让宗门任务偏向藏经与修持准备。", ImpactSummary: []string{"校录与修持任务优先级上升", "高危外勤权重下降"}},
		}
	case PolicyCategoryResource:
		return []PolicyOptionSummary{
			{Value: string(ResourcePolicySaving), Label: "节用蓄积", Explanation: "收紧月例和兑付预期以保存库存。", ImpactSummary: []string{"月结满意压力增加", "保留灵石与粮食库存"}},
			{Value: string(ResourcePolicyGenerous), Label: "厚给安众", Explanation: "提高供养慷慨度，缓和月结满意压力。", ImpactSummary: []string{"月结满意压力降低", "资源消耗预期增加"}},
			{Value: string(ResourcePolicyPillLimited), Label: "限丹管制", Explanation: "限制丹药消耗，避免成长线挤占库存。", ImpactSummary: []string{"丹药相关生产优先级下降", "突破支持更保守"}},
			{Value: string(ResourcePolicyOpenExchange), Label: "开放兑付", Explanation: "维持贡献兑付和日常生产的默认平衡。", ImpactSummary: []string{"兑付规则保持开放", "生产优先级不额外偏斜"}},
			{Value: string(ResourcePolicyWarPreparation), Label: "备战储材", Explanation: "提高阵材、矿材和防务生产优先级。", ImpactSummary: []string{"阵材炼制优先级上升", "战备库存倾向增强"}},
		}
	case PolicyCategoryRecruitment:
		return []PolicyOptionSummary{
			{Value: string(RecruitmentPolicyBroad), Label: "广开山门", Explanation: "默认招收规模，保持弟子来源稳定。", ImpactSummary: []string{"候选数量保持默认", "资质不额外筛选"}},
			{Value: string(RecruitmentPolicySelective), Label: "严筛根骨", Explanation: "减少候选数量，换取更高基础资质。", ImpactSummary: []string{"候选数量减少", "候选资质提高"}},
			{Value: string(RecruitmentPolicyWandering), Label: "寻访散修", Explanation: "偏向外出寻访，扩大不稳定来源。", ImpactSummary: []string{"候选数量略增", "来源更偏游历"}},
			{Value: string(RecruitmentPolicyAffiliated), Label: "附庸举荐", Explanation: "偏向关系来源，保持稳定但规模较小。", ImpactSummary: []string{"候选数量较少", "身份来源更稳定"}},
		}
	case PolicyCategoryCultivation:
		return []PolicyOptionSummary{
			{Value: string(CultivationPolicyBalanced), Label: "均衡修行", Explanation: "保持任务、生产与修炼之间的默认平衡。", ImpactSummary: []string{"每日修炼收益保持默认", "风险控制中性"}},
			{Value: string(CultivationPolicyClosedCultivation), Label: "闭关精进", Explanation: "提高修炼推进速度，并让宗门更偏向修持。", ImpactSummary: []string{"每日修炼收益提高", "外务倾向下降"}},
			{Value: string(CultivationPolicyBreakthroughSafe), Label: "稳妥破境", Explanation: "偏向安全突破和风险收束。", ImpactSummary: []string{"突破风险阈值更保守", "成长节奏更稳"}},
		}
	default:
		return []PolicyOptionSummary{}
	}
}

func ensureStarterDisciple(state *SectState) {
	if state == nil {
		return
	}
	if state.Disciples == nil {
		state.Disciples = map[DiscipleID]DiscipleState{}
	}
	if len(state.Disciples) > 0 {
		return
	}
	state.Disciples[starterDiscipleID] = newStarterDisciple()
}

func ensureContributionAccounts(state *SectState) {
	if state == nil {
		return
	}
	if state.Contribution.Accounts == nil {
		state.Contribution.Accounts = map[DiscipleID]ContributionAccount{}
	}
	if state.Contribution.TreasuryRules == nil {
		state.Contribution.TreasuryRules = defaultExchangeRules()
	}
	if state.Contribution.MonthlyPurchases == nil {
		state.Contribution.MonthlyPurchases = map[DiscipleID]map[ExchangeItemID]int64{}
	}
	for discipleID := range state.Disciples {
		if _, ok := state.Contribution.Accounts[discipleID]; ok {
			continue
		}
		state.Contribution.Accounts[discipleID] = ContributionAccount{DiscipleID: discipleID}
	}
}

func newStarterDisciple() DiscipleState {
	return DiscipleState{
		DiscipleID:     starterDiscipleID,
		Name:           starterDiscipleName,
		Location:       TileCoord{Col: 0, Row: 0},
		Identity:       IdentityRankOuter,
		Aptitude:       DiscipleAptitudeState{SpiritRoot: 7, Comprehension: 7, Physique: 7, Mind: 7, Luck: 6},
		AssignmentKind: DiscipleAssignmentIdle,
		WorkTarget: DiscipleWorkTargetState{
			Description: "sect_support",
		},
		Realm: RealmState{
			Stage: RealmMortal,
		},
		Needs: DiscipleNeedsState{
			DailySpiritGrain: 1,
			DailyRestTicks:   1,
		},
		Support: DiscipleSupportState{
			FoodSatisfied:    true,
			HousingSatisfied: true,
			MedicalSupported: false,
		},
		Satisfaction:  70,
		Loyalty:       70,
		Pressure:      0,
		InjuryLevel:   0,
		HP:            starterDiscipleMaxHP,
		MaxHP:         starterDiscipleMaxHP,
		Memories:      []DiscipleMemoryEntry{},
		Relationship:  []string{},
		Emotion:       []string{},
		RecentSummary: []string{},
	}
}

func newInitialContributionState() ContributionState {
	return ContributionState{
		Accounts:         map[DiscipleID]ContributionAccount{},
		TreasuryRules:    defaultExchangeRules(),
		MonthlyPurchases: map[DiscipleID]map[ExchangeItemID]int64{},
	}
}

func defaultExchangeRules() map[ExchangeItemID]ExchangeRule {
	return map[ExchangeItemID]ExchangeRule{
		ExchangeItemID("treasury-spirit-grain"): {
			ExchangeItemID:   ExchangeItemID("treasury-spirit-grain"),
			Name:             "灵谷配给",
			ItemKind:         ExchangeItemKind("resource"),
			ItemRef:          string(ResourceKindSpiritGrain),
			ContributionCost: 5,
			MonthlyLimit:     10,
			StockLimit:       -1,
			Enabled:          true,
		},
		ExchangeItemID("treasury-herb"): {
			ExchangeItemID:   ExchangeItemID("treasury-herb"),
			Name:             "灵植药材",
			ItemKind:         ExchangeItemKind("resource"),
			ItemRef:          string(ResourceKindHerb),
			ContributionCost: 8,
			MonthlyLimit:     6,
			StockLimit:       -1,
			Enabled:          true,
		},
	}
}

func defaultProductionJobs() map[ProductionID]ProductionJob {
	recipes := defaultProductionRecipes()
	return map[ProductionID]ProductionJob{
		ProductionID("prod-1-farm-grain"):       newProductionJobFromRecipe(recipes[RecipeID("farm_grain_mvp")], ProductionID("prod-1-farm-grain"), 0, 0),
		ProductionID("prod-2-herb-garden"):      newProductionJobFromRecipe(recipes[RecipeID("farm_herb_mvp")], ProductionID("prod-2-herb-garden"), 0, 0),
		ProductionID("prod-3-ore-mine"):         newProductionJobFromRecipe(recipes[RecipeID("mine_ore_mvp")], ProductionID("prod-3-ore-mine"), 0, 0),
		ProductionID("prod-4-formation-refine"): newProductionJobFromRecipe(recipes[RecipeID("formation_refine_mvp")], ProductionID("prod-4-formation-refine"), 0, 0),
	}
}

func defaultProductionRecipes() map[RecipeID]ProductionRecipe {
	return map[RecipeID]ProductionRecipe{
		RecipeID("farm_grain_mvp"): {
			RecipeID:             RecipeID("farm_grain_mvp"),
			Kind:                 ProductionKindFarm,
			BuildingID:           BuildingID("building-farm-1"),
			DefaultPriority:      40,
			OutputReward:         map[ResourceKind]int64{ResourceKindSpiritGrain: 60},
			RequiredProgressDays: 10,
		},
		RecipeID("farm_herb_mvp"): {
			RecipeID:             RecipeID("farm_herb_mvp"),
			Kind:                 ProductionKindFarm,
			BuildingID:           BuildingID("building-herb-1"),
			DefaultPriority:      45,
			OutputReward:         map[ResourceKind]int64{ResourceKindHerb: 8},
			RequiredProgressDays: 10,
		},
		RecipeID("mine_ore_mvp"): {
			RecipeID:             RecipeID("mine_ore_mvp"),
			Kind:                 ProductionKindMining,
			BuildingID:           BuildingID("building-mine-1"),
			DefaultPriority:      50,
			OutputReward:         map[ResourceKind]int64{ResourceKindOre: 12},
			RequiredProgressDays: 10,
		},
		RecipeID("formation_refine_mvp"): {
			RecipeID:        RecipeID("formation_refine_mvp"),
			Kind:            ProductionKindRefinement,
			BuildingID:      BuildingID("building-refinery-1"),
			DefaultPriority: 60,
			InputCost: map[ResourceKind]int64{
				ResourceKindSpiritStone: 4,
				ResourceKindHerb:        2,
				ResourceKindOre:         6,
			},
			OutputReward: map[ResourceKind]int64{
				ResourceKindFormationMat: 3,
			},
			RequiredProgressDays: 5,
		},
	}
}

func newProductionJobFromRecipe(recipe ProductionRecipe, productionID ProductionID, priority int, targetCycles int32) ProductionJob {
	if priority <= 0 {
		priority = recipe.DefaultPriority
	}
	return ProductionJob{
		ProductionID:         productionID,
		Kind:                 recipe.Kind,
		BuildingID:           recipe.BuildingID,
		RecipeID:             recipe.RecipeID,
		Status:               ProductionStatusRunning,
		Priority:             clampInt(priority, 0, 100),
		TargetCycles:         targetCycles,
		InputCost:            cloneResourceAmounts(recipe.InputCost),
		OutputReward:         cloneResourceAmounts(recipe.OutputReward),
		RequiredProgressDays: recipe.RequiredProgressDays,
	}
}
