package sect

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/anthdm/hollywood/actor"
)

type CommandType string
type CommandResultStatus string
type CommandErrorCode string
type ClientEventType string
type PatchOpType string
type PatchValueEncoding string
type DomainEventType string

const (
	CommandTypeUnspecified               CommandType = "COMMAND_TYPE_UNSPECIFIED"
	CommandTypeBuildBuilding             CommandType = "COMMAND_TYPE_BUILD_BUILDING"
	CommandTypeUpgradeBuilding           CommandType = "COMMAND_TYPE_UPGRADE_BUILDING"
	CommandTypeRepairBuilding            CommandType = "COMMAND_TYPE_REPAIR_BUILDING"
	CommandTypeSetPolicy                 CommandType = "COMMAND_TYPE_SET_POLICY"
	CommandTypePublishTask               CommandType = "COMMAND_TYPE_PUBLISH_TASK"
	CommandTypeCancelTask                CommandType = "COMMAND_TYPE_CANCEL_TASK"
	CommandTypeAssignDiscipleTask        CommandType = "COMMAND_TYPE_ASSIGN_DISCIPLE_TASK"
	CommandTypeSetTaskPriority           CommandType = "COMMAND_TYPE_SET_TASK_PRIORITY"
	CommandTypeExchangeContributionItem  CommandType = "COMMAND_TYPE_EXCHANGE_CONTRIBUTION_ITEM"
	CommandTypeStartProduction           CommandType = "COMMAND_TYPE_START_PRODUCTION"
	CommandTypeCancelProduction          CommandType = "COMMAND_TYPE_CANCEL_PRODUCTION"
	CommandTypeAdjustProduction          CommandType = "COMMAND_TYPE_ADJUST_PRODUCTION"
	CommandTypeStartCultivation          CommandType = "COMMAND_TYPE_START_CULTIVATION"
	CommandTypeUsePillForCultivation     CommandType = "COMMAND_TYPE_USE_PILL_FOR_CULTIVATION"
	CommandTypeReserveCave               CommandType = "COMMAND_TYPE_RESERVE_CAVE"
	CommandTypeAttemptBreakthrough       CommandType = "COMMAND_TYPE_ATTEMPT_BREAKTHROUGH"
	CommandTypeStartRecruitment          CommandType = "COMMAND_TYPE_START_RECRUITMENT"
	CommandTypeAcceptCandidate           CommandType = "COMMAND_TYPE_ACCEPT_CANDIDATE"
	CommandTypeRejectCandidate           CommandType = "COMMAND_TYPE_REJECT_CANDIDATE"
	CommandTypeChooseEventOption         CommandType = "COMMAND_TYPE_CHOOSE_EVENT_OPTION"
	CommandTypeDismissEvent              CommandType = "COMMAND_TYPE_DISMISS_EVENT"
	CommandTypeStartAssessment           CommandType = "COMMAND_TYPE_START_ASSESSMENT"
	CommandTypePromoteDisciple           CommandType = "COMMAND_TYPE_PROMOTE_DISCIPLE"
	CommandTypeAssignInstitutionManager  CommandType = "COMMAND_TYPE_ASSIGN_INSTITUTION_MANAGER"
	CommandTypeSetGatePolicy             CommandType = "COMMAND_TYPE_SET_GATE_POLICY"
	CommandTypeSetExchangeRule           CommandType = "COMMAND_TYPE_SET_EXCHANGE_RULE"
	CommandTypeCraftArtifact             CommandType = "COMMAND_TYPE_CRAFT_ARTIFACT"
	CommandTypeEquipArtifact             CommandType = "COMMAND_TYPE_EQUIP_ARTIFACT"
	CommandTypeUnequipArtifact           CommandType = "COMMAND_TYPE_UNEQUIP_ARTIFACT"
	CommandTypeRepairArtifact            CommandType = "COMMAND_TYPE_REPAIR_ARTIFACT"
	CommandTypeAttachFormationToBuilding CommandType = "COMMAND_TYPE_ATTACH_FORMATION_TO_BUILDING"
	CommandTypeDetachFormation           CommandType = "COMMAND_TYPE_DETACH_FORMATION"
	CommandTypeMaintainFormation         CommandType = "COMMAND_TYPE_MAINTAIN_FORMATION"

	CommandResultStatusUnspecified CommandResultStatus = "COMMAND_RESULT_STATUS_UNSPECIFIED"
	CommandResultStatusAccepted    CommandResultStatus = "COMMAND_RESULT_STATUS_ACCEPTED"
	CommandResultStatusRejected    CommandResultStatus = "COMMAND_RESULT_STATUS_REJECTED"

	CommandErrorCodeInvalidCommand         CommandErrorCode = "COMMAND_ERROR_CODE_INVALID_COMMAND"
	CommandErrorCodeInsufficientResource   CommandErrorCode = "COMMAND_ERROR_CODE_INSUFFICIENT_RESOURCE"
	CommandErrorCodeStaleClientVersion     CommandErrorCode = "COMMAND_ERROR_CODE_STALE_CLIENT_VERSION"
	CommandErrorCodeInternal               CommandErrorCode = "COMMAND_ERROR_CODE_INTERNAL"
	CommandErrorCodeCommandNotImplemented  CommandErrorCode = "COMMAND_ERROR_CODE_COMMAND_NOT_IMPLEMENTED"
	CommandErrorCodeDiscipleNotFound       CommandErrorCode = "COMMAND_ERROR_CODE_DISCIPLE_NOT_FOUND"
	CommandErrorCodeDiscipleBusy           CommandErrorCode = "COMMAND_ERROR_CODE_DISCIPLE_BUSY"
	CommandErrorCodeTaskNotFound           CommandErrorCode = "COMMAND_ERROR_CODE_TASK_NOT_FOUND"
	CommandErrorCodeTaskRequirementNotMet  CommandErrorCode = "COMMAND_ERROR_CODE_TASK_REQUIREMENT_NOT_MET"
	CommandErrorCodeContributionNotEnough  CommandErrorCode = "COMMAND_ERROR_CODE_CONTRIBUTION_NOT_ENOUGH"
	CommandErrorCodeMonthlyLimitReached    CommandErrorCode = "COMMAND_ERROR_CODE_MONTHLY_LIMIT_REACHED"
	CommandErrorCodeCultivationNotReady    CommandErrorCode = "COMMAND_ERROR_CODE_CULTIVATION_NOT_READY"
	CommandErrorCodeRiskTooHigh            CommandErrorCode = "COMMAND_ERROR_CODE_RISK_TOO_HIGH"
	CommandErrorCodeCandidateNotFound      CommandErrorCode = "COMMAND_ERROR_CODE_CANDIDATE_NOT_FOUND"
	CommandErrorCodeEventNotFound          CommandErrorCode = "COMMAND_ERROR_CODE_EVENT_NOT_FOUND"
	CommandErrorCodeEventExpired           CommandErrorCode = "COMMAND_ERROR_CODE_EVENT_EXPIRED"
	CommandErrorCodeEventRequirementNotMet CommandErrorCode = "COMMAND_ERROR_CODE_EVENT_REQUIREMENT_NOT_MET"

	ClientEventTypeResourceChanged          ClientEventType = "CLIENT_EVENT_TYPE_RESOURCE_CHANGED"
	ClientEventTypeBuildingChanged          ClientEventType = "CLIENT_EVENT_TYPE_BUILDING_CHANGED"
	ClientEventTypeTaskChanged              ClientEventType = "CLIENT_EVENT_TYPE_TASK_CHANGED"
	ClientEventTypeDiscipleChanged          ClientEventType = "CLIENT_EVENT_TYPE_DISCIPLE_CHANGED"
	ClientEventTypeContributionChanged      ClientEventType = "CLIENT_EVENT_TYPE_CONTRIBUTION_CHANGED"
	ClientEventTypeProductionChanged        ClientEventType = "CLIENT_EVENT_TYPE_PRODUCTION_CHANGED"
	ClientEventTypeInventoryChanged         ClientEventType = "CLIENT_EVENT_TYPE_INVENTORY_CHANGED"
	ClientEventTypeSectEventChanged         ClientEventType = "CLIENT_EVENT_TYPE_SECT_EVENT_CHANGED"
	ClientEventTypeAdmissionChanged         ClientEventType = "CLIENT_EVENT_TYPE_ADMISSION_CHANGED"
	ClientEventTypeTimeChanged              ClientEventType = "CLIENT_EVENT_TYPE_TIME_CHANGED"
	ClientEventTypeMonthlyChanged           ClientEventType = "CLIENT_EVENT_TYPE_MONTHLY_CHANGED"
	ClientEventTypePolicyChanged            ClientEventType = "CLIENT_EVENT_TYPE_POLICY_CHANGED"
	ClientEventTypeInstitutionChanged       ClientEventType = "CLIENT_EVENT_TYPE_INSTITUTION_CHANGED"
	ClientEventTypeSectMetaChanged          ClientEventType = "CLIENT_EVENT_TYPE_SECT_META_CHANGED"
	ClientEventTypeFormationChanged         ClientEventType = "CLIENT_EVENT_TYPE_FORMATION_CHANGED"
	ClientEventTypeSectGoalChanged          ClientEventType = "CLIENT_EVENT_TYPE_SECT_GOAL_CHANGED"
	ClientEventTypeOrderChanged             ClientEventType = "CLIENT_EVENT_TYPE_ORDER_CHANGED"
	ClientEventTypeMonthlyAssessmentChanged ClientEventType = "CLIENT_EVENT_TYPE_MONTHLY_ASSESSMENT_CHANGED"

	PatchOpTypeIncrement PatchOpType = "PATCH_OP_TYPE_INCREMENT"
	PatchOpTypeSet       PatchOpType = "PATCH_OP_TYPE_SET"

	PatchValueEncodingBytes  PatchValueEncoding = "PATCH_VALUE_ENCODING_BYTES"
	PatchValueEncodingVarint PatchValueEncoding = "PATCH_VALUE_ENCODING_VARINT"

	DomainEventTypeResourceChanged             DomainEventType = "resource_changed"
	DomainEventTypeSectMetaChanged             DomainEventType = "sect.meta_changed"
	DomainEventTypeBuildingBuilt               DomainEventType = "building.built"
	DomainEventTypeBuildingUpgraded            DomainEventType = "building.upgraded"
	DomainEventTypeBuildingMaintained          DomainEventType = "building.maintained"
	DomainEventTypeBuildingDamaged             DomainEventType = "building.damaged"
	DomainEventTypeTaskPublished               DomainEventType = "task.published"
	DomainEventTypeTaskAccepted                DomainEventType = "task.accepted"
	DomainEventTypeTaskCancelled               DomainEventType = "task.cancelled"
	DomainEventTypeTaskPriorityChanged         DomainEventType = "task.priority_changed"
	DomainEventTypeTaskProgressed              DomainEventType = "task.progressed"
	DomainEventTypeTaskCompleted               DomainEventType = "task.completed"
	DomainEventTypeTaskFailed                  DomainEventType = "task.failed"
	DomainEventTypeDiscipleAssignmentChanged   DomainEventType = "disciple.assignment_changed"
	DomainEventTypeContributionEarned          DomainEventType = "contribution.earned"
	DomainEventTypeContributionSpent           DomainEventType = "contribution.spent"
	DomainEventTypeProductionChanged           DomainEventType = "production.changed"
	DomainEventTypeInventoryChanged            DomainEventType = "inventory.changed"
	DomainEventTypeCultivationAdvanced         DomainEventType = "cultivation.advanced"
	DomainEventTypeBreakthroughSucceeded       DomainEventType = "breakthrough.succeeded"
	DomainEventTypeBreakthroughFailed          DomainEventType = "breakthrough.failed"
	DomainEventTypeTimeAdvanced                DomainEventType = "time.day_advanced"
	DomainEventTypePayrollPaid                 DomainEventType = "monthly.payroll_paid"
	DomainEventTypePayrollDelayed              DomainEventType = "monthly.payroll_delayed"
	DomainEventTypeMonthlyObligationChecked    DomainEventType = "monthly.obligation_checked"
	DomainEventTypeDiscipleSatisfactionChanged DomainEventType = "disciple.satisfaction_changed"
	DomainEventTypeDiscipleLoyaltyChanged      DomainEventType = "disciple.loyalty_changed"
	DomainEventTypeMonthAdvanced               DomainEventType = "monthly.month_advanced"
	DomainEventTypeSectEventSeeded             DomainEventType = "sect_event.seeded"
	DomainEventTypeSectEventForeshadowed       DomainEventType = "sect_event.foreshadowed"
	DomainEventTypeSectEventResolved           DomainEventType = "sect_event.resolved"
	DomainEventTypeSectEventExpired            DomainEventType = "sect_event.expired"
	DomainEventTypeSectGoalChanged             DomainEventType = "sect_goal.changed"
	DomainEventTypeSectGoalResolved            DomainEventType = "sect_goal.resolved"
	DomainEventTypeRecruitmentStarted          DomainEventType = "admission.recruitment_started"
	DomainEventTypeCandidateAccepted           DomainEventType = "admission.candidate_accepted"
	DomainEventTypeCandidateRejected           DomainEventType = "admission.candidate_rejected"
	DomainEventTypePolicyChanged               DomainEventType = "policy.changed"
	DomainEventTypeAssessmentResolved          DomainEventType = "disciple.assessment_resolved"
	DomainEventTypeDisciplePromoted            DomainEventType = "disciple.promoted"
	DomainEventTypeInstitutionChanged          DomainEventType = "institution.changed"
	DomainEventTypeArtifactChanged             DomainEventType = "artifact.changed"
	DomainEventTypeFormationChanged            DomainEventType = "formation.changed"
	DomainEventTypeOrderChanged                DomainEventType = "sect.order_changed"
	DomainEventTypeMonthlyAssessmentResolved   DomainEventType = "monthly.assessment_resolved"
)

const (
	InstitutionIDGate        InstitutionID = "gate"
	InstitutionIDMainHall    InstitutionID = "main_hall"
	InstitutionIDTaskHall    InstitutionID = "task_hall"
	InstitutionIDTreasury    InstitutionID = "treasury"
	InstitutionIDDormitory   InstitutionID = "dormitory"
	InstitutionIDCanteen     InstitutionID = "canteen"
	InstitutionIDMedicineHut InstitutionID = "medicine_hut"
	InstitutionIDCave        InstitutionID = "cave"
)

type ClientCommand struct {
	CmdID        string      `json:"cmdId"`
	UserID       string      `json:"userId"`
	SectID       string      `json:"sectId"`
	Type         CommandType `json:"type"`
	Payload      []byte      `json:"payload"`
	ClientSeq    int64       `json:"clientSeq,omitempty"`
	BaseVersion  Version     `json:"baseVersion,omitempty"`
	SentAtWallMS int64       `json:"sentAtWallMs,omitempty"`
}

type CommandError struct {
	Code      CommandErrorCode `json:"code"`
	Message   string           `json:"message"`
	Retriable bool             `json:"retriable"`
}

type CommandResult struct {
	CmdID        string              `json:"cmdId"`
	Status       CommandResultStatus `json:"status"`
	Error        *CommandError       `json:"error,omitempty"`
	SectID       string              `json:"sectId"`
	SceneVersion Version             `json:"sceneVersion"`
	Events       []ClientEvent       `json:"events,omitempty"`
	Patch        StatePatch          `json:"patch"`
	NeedSnapshot bool                `json:"needSnapshot"`
}

type ClientEvent struct {
	SceneVersion Version         `json:"sceneVersion"`
	Type         ClientEventType `json:"type"`
	Payload      []byte          `json:"payload"`
}

type StatePatch struct {
	SectID      string    `json:"sectId"`
	FromVersion Version   `json:"fromVersion"`
	ToVersion   Version   `json:"toVersion"`
	Ops         []PatchOp `json:"ops,omitempty"`
}

type PatchOp struct {
	Op            PatchOpType        `json:"op"`
	Path          string             `json:"path"`
	Value         []byte             `json:"value"`
	ValueEncoding PatchValueEncoding `json:"valueEncoding"`
}

type DomainEvent struct {
	EventID   string          `json:"eventId"`
	SectID    SectID          `json:"sectId"`
	Version   Version         `json:"version"`
	Type      DomainEventType `json:"type"`
	Payload   []byte          `json:"payload"`
	CommandID string          `json:"commandId"`
	GameTick  int64           `json:"gameTick"`
}

type BuildBuildingPayload struct {
	DefinitionKey string    `json:"definitionKey"`
	Origin        TileCoord `json:"origin"`
}

type UpgradeBuildingPayload struct {
	BuildingID BuildingID `json:"buildingId"`
}

type RepairBuildingPayload struct {
	BuildingID BuildingID `json:"buildingId"`
}

type SetPolicyPayload struct {
	PolicyCategory      string `json:"policyCategory"`
	PolicyValue         string `json:"policyValue"`
	PolicyCategorySnake string `json:"policy_category,omitempty"`
	PolicyValueSnake    string `json:"policy_value,omitempty"`
}

type PublishTaskPayload struct {
	Kind                 string                 `json:"kind"`
	Type                 TaskType               `json:"type,omitempty"`
	Grade                TaskGrade              `json:"grade,omitempty"`
	Title                string                 `json:"title"`
	Description          string                 `json:"description,omitempty"`
	Priority             int                    `json:"priority,omitempty"`
	RequiredProgressDays int32                  `json:"requiredProgressDays"`
	Risk                 int                    `json:"risk,omitempty"`
	MaxAssignees         int                    `json:"maxAssignees,omitempty"`
	MinIdentity          IdentityRank           `json:"minIdentity,omitempty"`
	MinRealm             RealmStage             `json:"minRealm,omitempty"`
	RequiredAptitude     DiscipleAptitudeState  `json:"requiredAptitude,omitempty"`
	DispatchCost         map[ResourceKind]int64 `json:"dispatchCost,omitempty"`
	ContributionReward   int64                  `json:"contributionReward"`
	RewardResources      map[ResourceKind]int64 `json:"rewardResources,omitempty"`
	ReputationReward     int                    `json:"reputationReward,omitempty"`
	RelationReward       map[string]int         `json:"relationReward,omitempty"`
	CrisisClue           string                 `json:"crisisClue,omitempty"`
}

type CancelTaskPayload struct {
	TaskID TaskID `json:"taskId"`
}

type AssignDiscipleTaskPayload struct {
	TaskID      TaskID       `json:"taskId"`
	DiscipleID  DiscipleID   `json:"discipleId,omitempty"`
	DiscipleIDs []DiscipleID `json:"discipleIds,omitempty"`
}

type SetTaskPriorityPayload struct {
	TaskID   TaskID `json:"taskId"`
	Priority int    `json:"priority"`
}

type ExchangeContributionItemPayload struct {
	DiscipleID     DiscipleID     `json:"discipleId"`
	ExchangeItemID ExchangeItemID `json:"exchangeItemId"`
	Quantity       int64          `json:"quantity"`
}

type StartProductionPayload struct {
	RecipeID     RecipeID `json:"recipeId"`
	Priority     int      `json:"priority,omitempty"`
	TargetCycles int32    `json:"targetCycles,omitempty"`
}

type CancelProductionPayload struct {
	ProductionID ProductionID `json:"productionId"`
}

type AdjustProductionPayload struct {
	ProductionID ProductionID `json:"productionId"`
	Priority     int          `json:"priority,omitempty"`
	TargetCycles int32        `json:"targetCycles,omitempty"`
}

type StartCultivationPayload struct {
	DiscipleID DiscipleID `json:"discipleId"`
}

type UsePillForCultivationPayload struct {
	DiscipleID DiscipleID `json:"discipleId"`
	PillType   PillType   `json:"pillType"`
	Quantity   int64      `json:"quantity"`
}

type ReserveCavePayload struct {
	DiscipleID   DiscipleID `json:"discipleId"`
	DurationDays int32      `json:"durationDays,omitempty"`
}

type AttemptBreakthroughPayload struct {
	DiscipleID          DiscipleID         `json:"discipleId"`
	UsePills            map[PillType]int64 `json:"usePills,omitempty"`
	UseSpiritStone      int64              `json:"useSpiritStone,omitempty"`
	CaveBuildingID      BuildingID         `json:"caveBuildingId,omitempty"`
	ProtectorDiscipleID DiscipleID         `json:"protectorDiscipleId,omitempty"`
}

type StartRecruitmentPayload struct {
	CandidateCount        int   `json:"candidateCount"`
	InvestmentSpiritStone int64 `json:"investmentSpiritStone,omitempty"`
	DurationDays          int32 `json:"durationDays,omitempty"`
}

type AcceptCandidatePayload struct {
	CandidateID CandidateID `json:"candidateId"`
}

type RejectCandidatePayload struct {
	CandidateID CandidateID `json:"candidateId"`
}

type ChooseEventOptionPayload struct {
	EventID  EventID `json:"eventId"`
	OptionID string  `json:"optionId"`
}

type DismissEventPayload struct {
	EventID EventID `json:"eventId"`
}

type StartAssessmentPayload struct {
	DiscipleID      DiscipleID   `json:"discipleId"`
	TargetRank      IdentityRank `json:"targetRank,omitempty"`
	TargetRankSnake IdentityRank `json:"target_rank,omitempty"`
}

type PromoteDisciplePayload struct {
	DiscipleID      DiscipleID   `json:"discipleId"`
	TargetRank      IdentityRank `json:"targetRank,omitempty"`
	TargetRankSnake IdentityRank `json:"target_rank,omitempty"`
}

type AssignInstitutionManagerPayload struct {
	InstitutionID InstitutionID `json:"institutionId"`
	DiscipleID    DiscipleID    `json:"discipleId"`
}

type SetGatePolicyPayload struct {
	OpenToVisitors            *bool        `json:"openToVisitors,omitempty"`
	AllowWanderingCultivators *bool        `json:"allowWanderingCultivators,omitempty"`
	EnforcementStrictness     *int         `json:"enforcementStrictness,omitempty"`
	GuardDiscipleIDs          []DiscipleID `json:"guardDiscipleIds,omitempty"`
}

type SetExchangeRulePayload struct {
	ExchangeItemID   ExchangeItemID `json:"exchangeItemId"`
	ContributionCost *int64         `json:"contributionCost,omitempty"`
	MonthlyLimit     *int64         `json:"monthlyLimit,omitempty"`
	Enabled          *bool          `json:"enabled,omitempty"`
}

type CraftArtifactPayload struct {
	ArtifactType ArtifactType `json:"artifactType"`
	Type         ArtifactType `json:"type,omitempty"`
	Quality      int          `json:"quality,omitempty"`
}

type EquipArtifactPayload struct {
	ItemID     ItemID     `json:"itemId"`
	DiscipleID DiscipleID `json:"discipleId"`
}

type UnequipArtifactPayload struct {
	ItemID     ItemID     `json:"itemId"`
	DiscipleID DiscipleID `json:"discipleId,omitempty"`
}

type RepairArtifactPayload struct {
	ItemID ItemID `json:"itemId"`
}

type AttachFormationToBuildingPayload struct {
	BuildingID     BuildingID    `json:"buildingId"`
	ArtifactItemID ItemID        `json:"artifactItemId"`
	FormationKind  FormationKind `json:"formationKind"`
}

type DetachFormationPayload struct {
	BuildingID BuildingID `json:"buildingId"`
}

type MaintainFormationPayload struct {
	BuildingID BuildingID `json:"buildingId"`
}

type BuildingBuiltPayload struct {
	Building BuildingState `json:"building"`
}

type BuildingUpgradedPayload struct {
	Building BuildingState `json:"building"`
}

type BuildingMaintainedPayload struct {
	Building BuildingState          `json:"building"`
	Cost     map[ResourceKind]int64 `json:"cost,omitempty"`
	Shortage map[ResourceKind]int64 `json:"shortage,omitempty"`
	Reason   string                 `json:"reason"`
}

type InstitutionChangedPayload struct {
	Institution  InstitutionLoopState `json:"institution"`
	ExchangeRule *ExchangeRule        `json:"exchange_rule,omitempty"`
	Reason       string               `json:"reason"`
}

type ResourceCost map[ResourceKind]int64

type ResourceChangedPayload struct {
	Changes map[ResourceKind]int64 `json:"changes"`
	Reason  string                 `json:"reason"`
}

type SectMetaChangedPayload struct {
	Meta   SectMetaState `json:"meta"`
	Reason string        `json:"reason"`
}

type TimeAdvancedPayload struct {
	Time SectTimeState `json:"time"`
}

type PayrollSettlementPayload struct {
	MonthIndex   int64      `json:"month_index"`
	DiscipleID   DiscipleID `json:"disciple_id"`
	Amount       int64      `json:"amount"`
	ArrearsAfter int        `json:"arrears_after"`
}

type MonthlyObligationCheckedPayload struct {
	MonthIndex      int64      `json:"month_index"`
	DiscipleID      DiscipleID `json:"disciple_id"`
	RequiredDays    int        `json:"required_days"`
	CompletedDays   int        `json:"completed_days"`
	ViolationAdded  bool       `json:"violation_added"`
	ViolationsAfter int        `json:"violations_after"`
}

type MonthAdvancedPayload struct {
	MonthIndex     int64                    `json:"month_index"`
	NextMonthIndex int64                    `json:"next_month_index"`
	Monthly        MonthlyState             `json:"monthly"`
	Summary        MonthlySettlementSummary `json:"summary"`
}

type OrderChangedPayload struct {
	Order SectOrderState `json:"order"`
}

type MonthlyAssessmentResolvedPayload struct {
	Assessment MonthlyAssessmentState  `json:"assessment"`
	Result     MonthlyAssessmentResult `json:"result"`
}

type TaskChangedPayload struct {
	Task TaskState `json:"task"`
}

type DiscipleChangedPayload struct {
	Disciple DiscipleState `json:"disciple"`
}

type AssessmentResolvedPayload struct {
	Disciple   DiscipleState           `json:"disciple"`
	Assessment DiscipleAssessmentState `json:"assessment"`
}

type ContributionChangedPayload struct {
	DiscipleID     DiscipleID      `json:"discipleId"`
	Delta          int64           `json:"delta"`
	Reason         string          `json:"reason"`
	ExchangeItemID *ExchangeItemID `json:"exchangeItemId,omitempty"`
	Quantity       int64           `json:"quantity,omitempty"`
}

type ProductionChangedPayload struct {
	Production ProductionJob `json:"production"`
}

type InventoryChangedPayload struct {
	Item InventoryEntry `json:"item"`
}

type ArtifactChangedPayload struct {
	Artifact ArtifactState  `json:"artifact"`
	Disciple *DiscipleState `json:"disciple,omitempty"`
	Reason   string         `json:"reason"`
}

type FormationChangedPayload struct {
	Formation FormationState `json:"formation"`
	Reason    string         `json:"reason"`
	Detached  bool           `json:"detached,omitempty"`
}

type SectEventChangedPayload struct {
	Event SectEvent `json:"event"`
}

type SectEventResolvedPayload struct {
	EventID EventID              `json:"event_id"`
	Summary ResolvedEventSummary `json:"summary"`
}

type SectGoalChangedPayload struct {
	Goal SectGoal `json:"goal"`
}

type SectGoalResolvedPayload struct {
	Goal    SectGoal                `json:"goal"`
	Summary ResolvedSectGoalSummary `json:"summary"`
}

type RecruitmentStartedPayload struct {
	Admissions AdmissionState `json:"admissions"`
}

type CandidateAcceptedPayload struct {
	CandidateID CandidateID    `json:"candidate_id"`
	Disciple    DiscipleState  `json:"disciple"`
	Admissions  AdmissionState `json:"admissions"`
}

type CandidateRejectedPayload struct {
	CandidateID CandidateID    `json:"candidate_id"`
	Admissions  AdmissionState `json:"admissions"`
}

type PolicyChangedPayload struct {
	Policies    PolicyState                    `json:"policies"`
	Tasks       map[TaskID]TaskState           `json:"tasks,omitempty"`
	Productions map[ProductionID]ProductionJob `json:"productions,omitempty"`
}

type DefenseRiskProjection struct {
	Intensity     int                        `json:"intensity"`
	Mitigation    int                        `json:"mitigation"`
	SourceSummary []DefenseRiskSourceSummary `json:"sourceSummary,omitempty"`
}

type DefenseRiskSourceSummary struct {
	Source string `json:"source"`
	Label  string `json:"label"`
	Delta  int    `json:"delta"`
}

type SectSnapshot struct {
	SectID          string                            `json:"sectId"`
	UserID          string                            `json:"userId"`
	SessionID       string                            `json:"sessionId"`
	SceneVersion    Version                           `json:"sceneVersion"`
	State           SectState                         `json:"state"`
	DefenseRisk     DefenseRiskProjection             `json:"defenseRisk"`
	BuildingCatalog []BuildingCatalogEntry            `json:"buildingCatalog"`
	TaskDispatch    map[TaskID]TaskDispatchProjection `json:"taskDispatch"`
	EventLog        []EventLogEntry                   `json:"eventLog"`
	Diary           []DiscipleDiaryEntry              `json:"diary"`
	EventSummaries  []SectEventFeedbackEntry          `json:"eventSummaries"`
}

type TaskDispatchProjection struct {
	TaskID                 TaskID       `json:"task_id"`
	RecommendedDiscipleIDs []DiscipleID `json:"recommended_disciple_ids,omitempty"`
	RecommendedSuccessRate int          `json:"recommended_success_rate,omitempty"`
	BlockedReason          string       `json:"blocked_reason,omitempty"`
}

type BuildingCatalogEntry struct {
	DefinitionKey         string                    `json:"definition_key"`
	Label                 string                    `json:"label"`
	MaxLevel              int32                     `json:"max_level"`
	UnlockSectLevel       int32                     `json:"unlock_sect_level"`
	RequiredMainHallLevel int32                     `json:"required_main_hall_level,omitempty"`
	MaxCount              int32                     `json:"max_count"`
	CurrentCount          int32                     `json:"current_count"`
	BuildCost             ResourceCost              `json:"build_cost"`
	UpgradeCostByLevel    map[int32]ResourceCost    `json:"upgrade_cost_by_level,omitempty"`
	MaintenanceByLevel    map[int32]ResourceCost    `json:"maintenance_by_level,omitempty"`
	Unlocked              bool                      `json:"unlocked"`
	CanBuild              bool                      `json:"can_build"`
	Blockers              []string                  `json:"blockers,omitempty"`
	ExistingBuildings     []BuildingCatalogInstance `json:"existing_buildings,omitempty"`
}

type BuildingCatalogInstance struct {
	BuildingID      BuildingID `json:"building_id"`
	Level           int32      `json:"level"`
	Phase           string     `json:"phase"`
	Efficiency      int        `json:"efficiency"`
	Durability      int        `json:"durability"`
	MaintenanceDebt int        `json:"maintenance_debt"`
	DamagedReason   string     `json:"damaged_reason,omitempty"`
}

type JoinSect struct {
	UserID    UserID
	SectID    SectID
	SessionID string
}

type SubmitCommand struct {
	SessionID string
	Command   ClientCommand
}

type JoinSectResponse struct {
	Snapshot SectSnapshot `json:"snapshot"`
}

type SubmitCommandResponse struct {
	Result       CommandResult `json:"result"`
	Snapshot     SectSnapshot  `json:"snapshot"`
	DomainEvents []DomainEvent `json:"-"`
}

type ExportState struct{}
type AdvanceTasksOneDay struct {
	SessionID string
}

type AdvanceTasksOneDayResponse struct {
	Snapshot     SectSnapshot  `json:"snapshot"`
	Events       []ClientEvent `json:"events,omitempty"`
	DomainEvents []DomainEvent `json:"-"`
	FromVersion  Version       `json:"fromVersion"`
	ToVersion    Version       `json:"toVersion"`
}

type SectActor struct {
	state         SectState
	eventLog      []DomainEvent
	commandResult map[string]SubmitCommandResponse
}

func NewSectActor(initialState SectState) actor.Receiver {
	return NewSectActorWithEventLog(initialState, nil)
}

func NewSectActorWithEventLog(initialState SectState, eventLog []DomainEvent) actor.Receiver {
	state := initialState.Clone()
	ensureInventoryState(&state)
	ensureStarterDisciple(&state)
	ensureAdmissionState(&state)
	ensureContributionAccounts(&state)
	ensureMonthlyState(&state)
	ensureSectGoalState(&state)
	ensurePolicyState(&state)
	applyPolicyEffectsToState(&state)
	recalculateContributionMetrics(&state)
	refreshCultivationDecisionState(&state)
	return &SectActor{
		state:         state,
		eventLog:      boundDomainEventLog(cloneDomainEvents(eventLog)),
		commandResult: map[string]SubmitCommandResponse{},
	}
}

func (a *SectActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case JoinSect:
		ctx.Respond(JoinSectResponse{Snapshot: a.snapshot(msg.UserID, msg.SessionID)})
	case SubmitCommand:
		ctx.Respond(a.executeCommand(msg))
	case AdvanceTasksOneDay:
		ctx.Respond(a.advanceTasksOneDay(msg.SessionID))
	case ExportState:
		ctx.Respond(a.state.Clone())
	}
}

func (a *SectActor) executeCommand(msg SubmitCommand) SubmitCommandResponse {
	if msg.Command.CmdID == "" {
		return a.rejected(msg, CommandErrorCodeInvalidCommand, "cmd_id is required", false)
	}
	if msg.Command.SectID == "" {
		msg.Command.SectID = string(a.state.Meta.SectID)
	}
	if SectID(msg.Command.SectID) != a.state.Meta.SectID {
		return a.rejected(msg, CommandErrorCodeInvalidCommand, "sect_id does not match actor state", false)
	}
	if msg.Command.UserID == "" {
		return a.rejected(msg, CommandErrorCodeInvalidCommand, "user_id is required", false)
	}
	if UserID(msg.Command.UserID) != a.state.Meta.OwnerUserID {
		return a.rejected(msg, CommandErrorCodeInvalidCommand, "user_id does not own this sect", false)
	}
	if cached, ok := a.cachedCommandResult(msg.Command.CmdID); ok {
		return cached
	}
	if msg.Command.BaseVersion != 0 && msg.Command.BaseVersion != a.state.Runtime.Version {
		return a.rejected(msg, CommandErrorCodeStaleClientVersion, "base_version does not match current sect version", true)
	}

	var (
		events []DomainEvent
		err    error
	)

	switch msg.Command.Type {
	case CommandTypeBuildBuilding:
		events, err = a.handleBuildBuilding(msg.Command)
	case CommandTypeUpgradeBuilding:
		events, err = a.handleUpgradeBuilding(msg.Command)
	case CommandTypeRepairBuilding:
		events, err = a.handleRepairBuilding(msg.Command)
	case CommandTypeSetPolicy:
		events, err = a.handleSetPolicy(msg.Command)
	case CommandTypePublishTask:
		events, err = a.handlePublishTask(msg.Command)
	case CommandTypeCancelTask:
		events, err = a.handleCancelTask(msg.Command)
	case CommandTypeAssignDiscipleTask:
		events, err = a.handleAssignDiscipleTask(msg.Command)
	case CommandTypeSetTaskPriority:
		events, err = a.handleSetTaskPriority(msg.Command)
	case CommandTypeExchangeContributionItem:
		events, err = a.handleExchangeContributionItem(msg.Command)
	case CommandTypeStartProduction:
		events, err = a.handleStartProduction(msg.Command)
	case CommandTypeCancelProduction:
		events, err = a.handleCancelProduction(msg.Command)
	case CommandTypeAdjustProduction:
		events, err = a.handleAdjustProduction(msg.Command)
	case CommandTypeStartCultivation:
		events, err = a.handleStartCultivation(msg.Command)
	case CommandTypeUsePillForCultivation:
		events, err = a.handleUsePillForCultivation(msg.Command)
	case CommandTypeReserveCave:
		events, err = a.handleReserveCave(msg.Command)
	case CommandTypeAttemptBreakthrough:
		events, err = a.handleAttemptBreakthrough(msg.Command)
	case CommandTypeStartRecruitment:
		events, err = a.handleStartRecruitment(msg.Command)
	case CommandTypeAcceptCandidate:
		events, err = a.handleAcceptCandidate(msg.Command)
	case CommandTypeRejectCandidate:
		events, err = a.handleRejectCandidate(msg.Command)
	case CommandTypeChooseEventOption:
		events, err = a.handleChooseEventOption(msg.Command)
	case CommandTypeDismissEvent:
		events, err = a.handleDismissEvent(msg.Command)
	case CommandTypeStartAssessment:
		events, err = a.handleStartAssessment(msg.Command)
	case CommandTypePromoteDisciple:
		events, err = a.handlePromoteDisciple(msg.Command)
	case CommandTypeAssignInstitutionManager:
		events, err = a.handleAssignInstitutionManager(msg.Command)
	case CommandTypeSetGatePolicy:
		events, err = a.handleSetGatePolicy(msg.Command)
	case CommandTypeSetExchangeRule:
		events, err = a.handleSetExchangeRule(msg.Command)
	case CommandTypeCraftArtifact:
		events, err = a.handleCraftArtifact(msg.Command)
	case CommandTypeEquipArtifact:
		events, err = a.handleEquipArtifact(msg.Command)
	case CommandTypeUnequipArtifact:
		events, err = a.handleUnequipArtifact(msg.Command)
	case CommandTypeRepairArtifact:
		events, err = a.handleRepairArtifact(msg.Command)
	case CommandTypeAttachFormationToBuilding:
		events, err = a.handleAttachFormationToBuilding(msg.Command)
	case CommandTypeDetachFormation:
		events, err = a.handleDetachFormation(msg.Command)
	case CommandTypeMaintainFormation:
		events, err = a.handleMaintainFormation(msg.Command)
	default:
		return a.rejected(msg, CommandErrorCodeCommandNotImplemented, fmt.Sprintf("command type %s is not implemented", msg.Command.Type), false)
	}
	if err != nil {
		if errors.Is(err, errInsufficientResources) {
			return a.rejected(msg, CommandErrorCodeInsufficientResource, err.Error(), false)
		}
		if errors.Is(err, errDiscipleNotFound) {
			return a.rejected(msg, CommandErrorCodeDiscipleNotFound, err.Error(), false)
		}
		if errors.Is(err, errDiscipleBusy) {
			return a.rejected(msg, CommandErrorCodeDiscipleBusy, err.Error(), false)
		}
		if errors.Is(err, errTaskNotFound) {
			return a.rejected(msg, CommandErrorCodeTaskNotFound, err.Error(), false)
		}
		if errors.Is(err, errTaskRequirementNotMet) {
			return a.rejected(msg, CommandErrorCodeTaskRequirementNotMet, err.Error(), false)
		}
		if errors.Is(err, errContributionNotEnough) {
			return a.rejected(msg, CommandErrorCodeContributionNotEnough, err.Error(), false)
		}
		if errors.Is(err, errMonthlyLimitReached) {
			return a.rejected(msg, CommandErrorCodeMonthlyLimitReached, err.Error(), false)
		}
		if errors.Is(err, errCultivationNotReady) {
			return a.rejected(msg, CommandErrorCodeCultivationNotReady, err.Error(), false)
		}
		if errors.Is(err, errRiskTooHigh) {
			return a.rejected(msg, CommandErrorCodeRiskTooHigh, err.Error(), false)
		}
		if errors.Is(err, errCandidateNotFound) {
			return a.rejected(msg, CommandErrorCodeCandidateNotFound, err.Error(), false)
		}
		if errors.Is(err, errEventNotFound) {
			return a.rejected(msg, CommandErrorCodeEventNotFound, err.Error(), false)
		}
		if errors.Is(err, errEventExpired) {
			return a.rejected(msg, CommandErrorCodeEventExpired, err.Error(), false)
		}
		if errors.Is(err, errEventRequirementNotMet) {
			return a.rejected(msg, CommandErrorCodeEventRequirementNotMet, err.Error(), false)
		}
		return a.rejected(msg, CommandErrorCodeInvalidCommand, err.Error(), false)
	}

	fromVersion := a.state.Runtime.Version
	if applyErr := a.applyDomainEvents(events); applyErr != nil {
		return a.rejected(msg, CommandErrorCodeInternal, applyErr.Error(), true)
	}

	clientEvents, patchOps, patchErr := clientDeltaForEvents(events)
	if patchErr != nil {
		return a.rejected(msg, CommandErrorCodeInternal, patchErr.Error(), true)
	}

	result := CommandResult{
		CmdID:        msg.Command.CmdID,
		Status:       CommandResultStatusAccepted,
		SectID:       string(a.state.Meta.SectID),
		SceneVersion: a.state.Runtime.Version,
		Events:       clientEvents,
		Patch: StatePatch{
			SectID:      string(a.state.Meta.SectID),
			FromVersion: fromVersion,
			ToVersion:   a.state.Runtime.Version,
			Ops:         patchOps,
		},
	}

	response := SubmitCommandResponse{
		Snapshot:     a.snapshot(UserID(msg.Command.UserID), msg.SessionID),
		Result:       result,
		DomainEvents: cloneDomainEvents(events),
	}
	a.rememberCommandResult(msg.Command.CmdID, response)
	return response
}

func (a *SectActor) cachedCommandResult(cmdID string) (SubmitCommandResponse, bool) {
	if a.commandResult == nil {
		a.commandResult = map[string]SubmitCommandResponse{}
	}
	response, ok := a.commandResult[cmdID]
	if !ok {
		return SubmitCommandResponse{}, false
	}
	return cloneSubmitCommandResponse(response), true
}

func (a *SectActor) rememberCommandResult(cmdID string, response SubmitCommandResponse) {
	if a.commandResult == nil {
		a.commandResult = map[string]SubmitCommandResponse{}
	}
	a.commandResult[cmdID] = cloneSubmitCommandResponse(response)
}

func cloneSubmitCommandResponse(response SubmitCommandResponse) SubmitCommandResponse {
	cloned := response
	cloned.Snapshot.State = response.Snapshot.State.Clone()
	cloned.Snapshot.DefenseRisk.SourceSummary = append([]DefenseRiskSourceSummary(nil), response.Snapshot.DefenseRisk.SourceSummary...)
	cloned.Result.Events = cloneClientEvents(response.Result.Events)
	cloned.Result.Patch.Ops = clonePatchOps(response.Result.Patch.Ops)
	cloned.DomainEvents = cloneDomainEvents(response.DomainEvents)
	if response.Result.Error != nil {
		errorCopy := *response.Result.Error
		cloned.Result.Error = &errorCopy
	}
	return cloned
}

func cloneClientEvents(events []ClientEvent) []ClientEvent {
	if len(events) == 0 {
		return []ClientEvent{}
	}
	cloned := make([]ClientEvent, len(events))
	for index, event := range events {
		cloned[index] = event
		cloned[index].Payload = append([]byte(nil), event.Payload...)
	}
	return cloned
}

func clonePatchOps(ops []PatchOp) []PatchOp {
	if len(ops) == 0 {
		return []PatchOp{}
	}
	cloned := make([]PatchOp, len(ops))
	for index, op := range ops {
		cloned[index] = op
		cloned[index].Value = append([]byte(nil), op.Value...)
	}
	return cloned
}

func (a *SectActor) handleBuildBuilding(command ClientCommand) ([]DomainEvent, error) {
	var payload BuildBuildingPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("build building payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode build building payload: %w", err)
	}
	payload.DefinitionKey = strings.TrimSpace(payload.DefinitionKey)
	if payload.DefinitionKey == "" {
		return nil, errors.New("definitionKey is required")
	}
	if err := validateBuildingBuild(a.state, payload.DefinitionKey); err != nil {
		return nil, err
	}
	cost, ok := buildingCostFor(payload.DefinitionKey)
	if !ok {
		return nil, fmt.Errorf("definitionKey %s is not buildable", payload.DefinitionKey)
	}
	if !CanAfford(a.state.Resources, cost) {
		return nil, fmt.Errorf("%w: %s", errInsufficientResources, payload.DefinitionKey)
	}
	for _, building := range a.state.Buildings {
		if building.Origin == payload.Origin {
			return nil, errors.New("building origin is already occupied")
		}
	}

	resourceEventVersion := a.state.Runtime.Version + 1
	buildingEventVersion := a.state.Runtime.Version + 2
	buildingID := BuildingID(fmt.Sprintf("building-%d", len(a.state.Buildings)+1))
	building := newBuildingState(buildingID, payload.DefinitionKey, 1, payload.Origin)
	resourcePayload, err := json.Marshal(ResourceChangedPayload{
		Changes: invertCost(cost),
		Reason:  "build_building",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal resource changed payload: %w", err)
	}

	return []DomainEvent{
		{
			EventID:   fmt.Sprintf("%s@%d", DomainEventTypeResourceChanged, resourceEventVersion),
			SectID:    a.state.Meta.SectID,
			Version:   resourceEventVersion,
			Type:      DomainEventTypeResourceChanged,
			Payload:   resourcePayload,
			CommandID: command.CmdID,
			GameTick:  a.state.Time.GameTick,
		},
		{
			EventID:   fmt.Sprintf("%s@%d", DomainEventTypeBuildingBuilt, buildingEventVersion),
			SectID:    a.state.Meta.SectID,
			Version:   buildingEventVersion,
			Type:      DomainEventTypeBuildingBuilt,
			Payload:   mustMarshalBuildingPayload(BuildingBuiltPayload{Building: building}),
			CommandID: command.CmdID,
			GameTick:  a.state.Time.GameTick,
		},
	}, nil
}

func (a *SectActor) handleUpgradeBuilding(command ClientCommand) ([]DomainEvent, error) {
	var payload UpgradeBuildingPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("upgrade building payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode upgrade building payload: %w", err)
	}
	building, ok := a.state.Buildings[payload.BuildingID]
	if !ok {
		return nil, fmt.Errorf("building %s does not exist", payload.BuildingID)
	}
	if err := validateBuildingUpgrade(a.state, building); err != nil {
		return nil, err
	}
	cost, ok := buildingUpgradeCostFor(building.DefinitionKey, building.Level)
	if !ok {
		return nil, fmt.Errorf("building %s cannot be upgraded from level %d", building.DefinitionKey, building.Level)
	}
	if !CanAfford(a.state.Resources, cost) {
		return nil, fmt.Errorf("%w: %s level %d", errInsufficientResources, building.DefinitionKey, building.Level+1)
	}

	resourceEventVersion := a.state.Runtime.Version + 1
	upgradeEventVersion := a.state.Runtime.Version + 2
	upgraded := newBuildingState(building.BuildingID, building.DefinitionKey, building.Level+1, building.Origin)

	resourcePayload, err := json.Marshal(ResourceChangedPayload{
		Changes: invertCost(cost),
		Reason:  "upgrade_building",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal resource changed payload: %w", err)
	}

	return []DomainEvent{
		{
			EventID:   fmt.Sprintf("%s@%d", DomainEventTypeResourceChanged, resourceEventVersion),
			SectID:    a.state.Meta.SectID,
			Version:   resourceEventVersion,
			Type:      DomainEventTypeResourceChanged,
			Payload:   resourcePayload,
			CommandID: command.CmdID,
			GameTick:  a.state.Time.GameTick,
		},
		{
			EventID:   fmt.Sprintf("%s@%d", DomainEventTypeBuildingUpgraded, upgradeEventVersion),
			SectID:    a.state.Meta.SectID,
			Version:   upgradeEventVersion,
			Type:      DomainEventTypeBuildingUpgraded,
			Payload:   mustMarshalBuildingPayload(BuildingUpgradedPayload{Building: upgraded}),
			CommandID: command.CmdID,
			GameTick:  a.state.Time.GameTick,
		},
	}, nil
}

func (a *SectActor) handleRepairBuilding(command ClientCommand) ([]DomainEvent, error) {
	var payload RepairBuildingPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("repair building payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode repair building payload: %w", err)
	}
	building, ok := a.state.Buildings[payload.BuildingID]
	if !ok {
		return nil, fmt.Errorf("building %s does not exist", payload.BuildingID)
	}
	building = normalizedBuildingRuntimeFields(building)
	if !buildingNeedsRepair(building) {
		return nil, fmt.Errorf("%w: building %s does not need repair", errTaskRequirementNotMet, payload.BuildingID)
	}
	cost := buildingRepairCostFor(building)
	if !CanAfford(a.state.Resources, cost) {
		return nil, fmt.Errorf("%w: repair %s", errInsufficientResources, payload.BuildingID)
	}

	resourceEventVersion := a.state.Runtime.Version + 1
	repairEventVersion := a.state.Runtime.Version + 2
	repaired := buildingAfterRepair(building)
	resourcePayload, err := json.Marshal(ResourceChangedPayload{
		Changes: invertCost(cost),
		Reason:  "repair_building",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal resource changed payload: %w", err)
	}

	return []DomainEvent{
		{
			EventID:   fmt.Sprintf("%s@%d", DomainEventTypeResourceChanged, resourceEventVersion),
			SectID:    a.state.Meta.SectID,
			Version:   resourceEventVersion,
			Type:      DomainEventTypeResourceChanged,
			Payload:   resourcePayload,
			CommandID: command.CmdID,
			GameTick:  a.state.Time.GameTick,
		},
		{
			EventID: fmt.Sprintf("%s@%d", DomainEventTypeBuildingMaintained, repairEventVersion),
			SectID:  a.state.Meta.SectID,
			Version: repairEventVersion,
			Type:    DomainEventTypeBuildingMaintained,
			Payload: mustMarshalBuildingPayload(BuildingMaintainedPayload{
				Building: repaired,
				Cost:     cost,
				Reason:   "repair_building",
			}),
			CommandID: command.CmdID,
			GameTick:  a.state.Time.GameTick,
		},
	}, nil
}

func (a *SectActor) handleSetPolicy(command ClientCommand) ([]DomainEvent, error) {
	var payload SetPolicyPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("set policy payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode set policy payload: %w", err)
	}

	category := strings.TrimSpace(payload.PolicyCategory)
	if category == "" {
		category = strings.TrimSpace(payload.PolicyCategorySnake)
	}
	value := strings.TrimSpace(payload.PolicyValue)
	if value == "" {
		value = strings.TrimSpace(payload.PolicyValueSnake)
	}
	if category == "" || value == "" {
		return nil, errors.New("policy category and value are required")
	}

	ensurePolicyState(&a.state)
	nextPolicies, err := setPolicyValue(a.state.Policies, category, value)
	if err != nil {
		return nil, err
	}
	changed := buildPolicyChangedPayload(a.state, nextPolicies)
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypePolicyChanged, changed, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handlePublishTask(command ClientCommand) ([]DomainEvent, error) {
	var payload PublishTaskPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("publish task payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode publish task payload: %w", err)
	}

	payload.Kind = strings.TrimSpace(payload.Kind)
	payload.Title = strings.TrimSpace(payload.Title)
	if payload.Kind == "" {
		return nil, errors.New("kind is required")
	}
	if payload.Title == "" {
		return nil, errors.New("title is required")
	}
	if payload.RequiredProgressDays <= 0 {
		return nil, errors.New("requiredProgressDays must be greater than zero")
	}
	if payload.ContributionReward < 0 {
		return nil, errors.New("contributionReward cannot be negative")
	}
	if payload.Risk < 0 || payload.Risk > 100 {
		return nil, errors.New("risk must be between 0 and 100")
	}

	ensureInstitutionState(&a.state)
	openTasks := 0
	for _, existing := range a.state.Tasks {
		if existing.Status == TaskStatusPublished || existing.Status == TaskStatusAccepted {
			openTasks++
		}
	}
	if limit := taskHallOpenTaskLimit(a.state); openTasks >= limit {
		return nil, fmt.Errorf("%w: task hall open task limit %d reached", errTaskRequirementNotMet, limit)
	}

	task := TaskState{
		TaskID:               nextPlayerTaskID(a.state.Tasks),
		Kind:                 payload.Kind,
		Type:                 normalizeTaskType(payload.Kind, payload.Type),
		Grade:                normalizeTaskGrade(payload.Risk, payload.Grade),
		Title:                payload.Title,
		Description:          strings.TrimSpace(payload.Description),
		Status:               TaskStatusPublished,
		Priority:             payload.Priority,
		RequiredProgressDays: payload.RequiredProgressDays,
		Risk:                 payload.Risk,
		MaxAssignees:         normalizeMaxAssignees(payload.MaxAssignees),
		MinIdentity:          normalizeMinIdentity(payload.MinIdentity),
		MinRealm:             normalizeMinRealm(payload.MinRealm),
		RequiredAptitude:     payload.RequiredAptitude,
		DispatchCost:         normalizePositiveResourceAmounts(payload.DispatchCost),
		ContributionReward:   payload.ContributionReward,
		RewardResources:      normalizePositiveResourceAmounts(payload.RewardResources),
		ReputationReward:     maxInt(payload.ReputationReward, 0),
		RelationReward:       normalizePositiveStringIntMap(payload.RelationReward),
		CrisisClue:           strings.TrimSpace(payload.CrisisClue),
	}

	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeTaskPublished, TaskChangedPayload{Task: task}, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handleCancelTask(command ClientCommand) ([]DomainEvent, error) {
	var payload CancelTaskPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("cancel task payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode cancel task payload: %w", err)
	}
	task, ok := a.state.Tasks[payload.TaskID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errTaskNotFound, payload.TaskID)
	}
	if task.Status == TaskStatusCompleted || task.Status == TaskStatusFailed || task.Status == TaskStatusCancelled {
		return nil, fmt.Errorf("%w: task %s is already closed", errTaskRequirementNotMet, payload.TaskID)
	}

	events := make([]DomainEvent, 0, len(task.AssignedDiscipleIDs)+1)
	nextTask := task
	nextTask.Status = TaskStatusCancelled
	nextTask.AssignedDiscipleIDs = nil
	events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeTaskCancelled, TaskChangedPayload{Task: nextTask}, a.state.Runtime.Version+1))

	versionCursor := a.state.Runtime.Version + 1
	for _, discipleID := range sortedDiscipleIDs(task.AssignedDiscipleIDs) {
		disciple, ok := a.state.Disciples[discipleID]
		if !ok {
			continue
		}
		cleared := clearDiscipleTaskAssignment(disciple)
		versionCursor++
		events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeDiscipleAssignmentChanged, DiscipleChangedPayload{Disciple: cleared}, versionCursor))
	}
	return events, nil
}

func (a *SectActor) handleAssignDiscipleTask(command ClientCommand) ([]DomainEvent, error) {
	var payload AssignDiscipleTaskPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("assign disciple task payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode assign disciple task payload: %w", err)
	}

	task, ok := a.state.Tasks[payload.TaskID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errTaskNotFound, payload.TaskID)
	}
	if task.Status != TaskStatusPublished {
		return nil, fmt.Errorf("%w: task %s is not publishable", errTaskRequirementNotMet, payload.TaskID)
	}
	if len(task.AssignedDiscipleIDs) > 0 {
		return nil, fmt.Errorf("%w: task %s already has assigned disciples", errTaskRequirementNotMet, payload.TaskID)
	}

	discipleIDs := normalizeAssignDiscipleIDs(payload)
	if len(discipleIDs) == 0 {
		return nil, errors.New("at least one disciple id is required")
	}
	if maxAssignees := taskMaxAssignees(task); len(discipleIDs) > maxAssignees {
		return nil, fmt.Errorf("%w: task %s accepts at most %d disciples", errTaskRequirementNotMet, payload.TaskID, maxAssignees)
	}
	if len(task.DispatchCost) > 0 && !CanAfford(a.state.Resources, ResourceCost(task.DispatchCost)) {
		return nil, fmt.Errorf("%w: task %s dispatch cost", errInsufficientResources, task.TaskID)
	}

	assignedTaskID := task.TaskID
	assignedTask := task
	assignedTask.Status = TaskStatusAccepted
	assignedTask.AssignedDiscipleIDs = append([]DiscipleID(nil), discipleIDs...)
	assignedTask.SuccessRate = taskSuccessRate(a.state, assignedTask)
	assignedTask.Evaluation = taskEvaluationForRate(assignedTask.SuccessRate)

	events := make([]DomainEvent, 0, len(discipleIDs)+2)
	versionCursor := a.state.Runtime.Version
	if len(task.DispatchCost) > 0 {
		versionCursor++
		events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeResourceChanged, ResourceChangedPayload{
			Changes: invertCost(ResourceCost(task.DispatchCost)),
			Reason:  "task_dispatch",
		}, versionCursor))
	}
	versionCursor++
	events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeTaskAccepted, TaskChangedPayload{Task: assignedTask}, versionCursor))

	for _, discipleID := range discipleIDs {
		disciple, ok := a.state.Disciples[discipleID]
		if !ok {
			return nil, fmt.Errorf("%w: %s", errDiscipleNotFound, discipleID)
		}
		if disciple.AssignmentKind != DiscipleAssignmentIdle || disciple.AssignmentTask != nil {
			return nil, fmt.Errorf("%w: %s is already assigned", errDiscipleBusy, discipleID)
		}
		if !discipleMeetsTaskRequirements(disciple, task) {
			return nil, fmt.Errorf("%w: disciple %s does not meet task requirements", errTaskRequirementNotMet, discipleID)
		}
		assignedDisciple := disciple
		assignedDisciple.AssignmentKind = DiscipleAssignmentTask
		assignedDisciple.AssignmentTask = &assignedTaskID
		assignedDisciple.WorkTarget = DiscipleWorkTargetState{
			TaskID:      &assignedTaskID,
			Description: fmt.Sprintf("task_hall:%s", task.Kind),
		}
		versionCursor++
		events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeDiscipleAssignmentChanged, DiscipleChangedPayload{Disciple: assignedDisciple}, versionCursor))
	}
	return events, nil
}

func (a *SectActor) handleSetTaskPriority(command ClientCommand) ([]DomainEvent, error) {
	var payload SetTaskPriorityPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("set task priority payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode set task priority payload: %w", err)
	}
	task, ok := a.state.Tasks[payload.TaskID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errTaskNotFound, payload.TaskID)
	}
	if task.Status == TaskStatusCompleted || task.Status == TaskStatusFailed || task.Status == TaskStatusCancelled {
		return nil, fmt.Errorf("%w: task %s is already closed", errTaskRequirementNotMet, payload.TaskID)
	}
	task.Priority = clampInt(payload.Priority, 0, 100)
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeTaskPriorityChanged, TaskChangedPayload{Task: task}, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handleExchangeContributionItem(command ClientCommand) ([]DomainEvent, error) {
	var payload ExchangeContributionItemPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("exchange contribution payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode exchange contribution payload: %w", err)
	}
	if payload.Quantity <= 0 {
		return nil, errors.New("quantity must be greater than zero")
	}
	if _, ok := a.state.Disciples[payload.DiscipleID]; !ok {
		return nil, fmt.Errorf("%w: %s", errDiscipleNotFound, payload.DiscipleID)
	}

	rule, ok := a.state.Contribution.TreasuryRules[payload.ExchangeItemID]
	if !ok || !rule.Enabled {
		return nil, fmt.Errorf("exchange rule %s is not available", payload.ExchangeItemID)
	}
	ensureInstitutionState(&a.state)
	rule = effectiveExchangeRuleForState(a.state, rule)
	if rule.ItemKind != ExchangeItemKind("resource") {
		return nil, fmt.Errorf("exchange rule %s is not a resource rule", payload.ExchangeItemID)
	}

	account := a.state.Contribution.Accounts[payload.DiscipleID]
	totalCost := rule.ContributionCost * payload.Quantity
	if account.Balance < totalCost {
		return nil, fmt.Errorf("%w: disciple %s balance %d < %d", errContributionNotEnough, payload.DiscipleID, account.Balance, totalCost)
	}

	purchased := a.state.Contribution.MonthlyPurchases[payload.DiscipleID][payload.ExchangeItemID]
	if rule.MonthlyLimit > 0 && purchased+payload.Quantity > rule.MonthlyLimit {
		return nil, fmt.Errorf("%w: exchange %s limit %d", errMonthlyLimitReached, payload.ExchangeItemID, rule.MonthlyLimit)
	}

	resourceKind := ResourceKind(rule.ItemRef)
	if a.state.Resources.Stock[resourceKind] < payload.Quantity {
		return nil, fmt.Errorf("%w: %s", errInsufficientResources, resourceKind)
	}

	exchangeItemID := payload.ExchangeItemID
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeContributionSpent, ContributionChangedPayload{
			DiscipleID:     payload.DiscipleID,
			Delta:          -totalCost,
			Reason:         "treasury_exchange",
			ExchangeItemID: &exchangeItemID,
			Quantity:       payload.Quantity,
		}, a.state.Runtime.Version+1),
		a.newDomainEvent(command.CmdID, DomainEventTypeResourceChanged, ResourceChangedPayload{
			Changes: map[ResourceKind]int64{resourceKind: -payload.Quantity},
			Reason:  "contribution_exchange",
		}, a.state.Runtime.Version+2),
	}, nil
}

func (a *SectActor) handleStartProduction(command ClientCommand) ([]DomainEvent, error) {
	var payload StartProductionPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("start production payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode start production payload: %w", err)
	}
	recipe, ok := defaultProductionRecipes()[payload.RecipeID]
	if !ok {
		return nil, fmt.Errorf("production recipe %s is not available", payload.RecipeID)
	}
	if payload.TargetCycles < 0 {
		return nil, errors.New("targetCycles cannot be negative")
	}

	priority := payload.Priority
	if priority <= 0 {
		priority = productionPriorityForPolicy(recipe, a.state.Policies.ResourcePolicy)
	}
	job := newProductionJobFromRecipe(recipe, nextPlayerProductionID(a.state.Productions), priority, payload.TargetCycles)
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeProductionChanged, ProductionChangedPayload{Production: job}, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handleCancelProduction(command ClientCommand) ([]DomainEvent, error) {
	var payload CancelProductionPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("cancel production payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode cancel production payload: %w", err)
	}
	job, ok := a.state.Productions[payload.ProductionID]
	if !ok {
		return nil, fmt.Errorf("production %s does not exist", payload.ProductionID)
	}
	if productionClosed(job.Status) {
		return nil, fmt.Errorf("production %s is already closed", payload.ProductionID)
	}
	job.Status = ProductionStatusCancelled
	job.BlockedReason = ""
	job.Shortage = nil
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeProductionChanged, ProductionChangedPayload{Production: job}, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handleAdjustProduction(command ClientCommand) ([]DomainEvent, error) {
	var payload AdjustProductionPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("adjust production payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode adjust production payload: %w", err)
	}
	if payload.TargetCycles < 0 {
		return nil, errors.New("targetCycles cannot be negative")
	}
	job, ok := a.state.Productions[payload.ProductionID]
	if !ok {
		return nil, fmt.Errorf("production %s does not exist", payload.ProductionID)
	}
	if productionClosed(job.Status) {
		return nil, fmt.Errorf("production %s is already closed", payload.ProductionID)
	}
	if payload.Priority > 0 {
		job.Priority = clampInt(payload.Priority, 0, 100)
	}
	job.TargetCycles = payload.TargetCycles
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeProductionChanged, ProductionChangedPayload{Production: job}, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handleStartCultivation(command ClientCommand) ([]DomainEvent, error) {
	var payload StartCultivationPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("start cultivation payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode start cultivation payload: %w", err)
	}

	disciple, ok := a.state.Disciples[payload.DiscipleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errDiscipleNotFound, payload.DiscipleID)
	}
	if disciple.AssignmentKind != DiscipleAssignmentIdle || disciple.AssignmentTask != nil {
		return nil, fmt.Errorf("%w: %s is already assigned", errDiscipleBusy, payload.DiscipleID)
	}
	if disciple.InjuryLevel >= 3 {
		return nil, fmt.Errorf("%w: disciple %s is too injured for cultivation", errRiskTooHigh, payload.DiscipleID)
	}
	if disciple.Pressure >= 80 {
		return nil, fmt.Errorf("%w: disciple %s pressure too high", errRiskTooHigh, payload.DiscipleID)
	}

	next := disciple
	next.AssignmentKind = DiscipleAssignmentCultivation
	next.WorkTarget = DiscipleWorkTargetState{Description: "cultivation:daily"}

	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeDiscipleAssignmentChanged, DiscipleChangedPayload{Disciple: next}, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handleUsePillForCultivation(command ClientCommand) ([]DomainEvent, error) {
	var payload UsePillForCultivationPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("use pill for cultivation payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode use pill for cultivation payload: %w", err)
	}
	if payload.Quantity <= 0 {
		return nil, errors.New("quantity must be greater than zero")
	}
	if payload.PillType != PillCultivation {
		return nil, fmt.Errorf("pill %s is not supported for cultivation growth", payload.PillType)
	}
	disciple, ok := a.state.Disciples[payload.DiscipleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errDiscipleNotFound, payload.DiscipleID)
	}
	itemID := pillInventoryItemID(payload.PillType)
	item := a.state.Inventory.Items[itemID]
	if item.Quantity < payload.Quantity {
		return nil, fmt.Errorf("%w: %s requires %d", errInsufficientResources, payload.PillType, payload.Quantity)
	}

	updatedItem := item
	updatedItem.Quantity -= payload.Quantity
	next := disciple
	next.Realm.CultivationPoints += payload.Quantity * cultivationPillPointGain()
	next.Realm.ReadyForBreakthrough = next.Realm.CultivationPoints >= cultivationThresholdForStage(next.Realm.Stage)
	next.Pressure = minInt(100, next.Pressure+int(payload.Quantity)*3)
	next.WorkTarget = DiscipleWorkTargetState{Description: fmt.Sprintf("cultivation:pill:%s", payload.PillType)}

	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeInventoryChanged, InventoryChangedPayload{Item: updatedItem}, a.state.Runtime.Version+1),
		a.newDomainEvent(command.CmdID, DomainEventTypeCultivationAdvanced, DiscipleChangedPayload{Disciple: next}, a.state.Runtime.Version+2),
	}, nil
}

func (a *SectActor) handleReserveCave(command ClientCommand) ([]DomainEvent, error) {
	var payload ReserveCavePayload
	if len(command.Payload) == 0 {
		return nil, errors.New("reserve cave payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode reserve cave payload: %w", err)
	}
	disciple, ok := a.state.Disciples[payload.DiscipleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errDiscipleNotFound, payload.DiscipleID)
	}
	durationDays := payload.DurationDays
	if durationDays <= 0 {
		durationDays = 1
	}
	cost := reserveCaveSpiritStoneCost(durationDays)
	if a.state.Resources.Stock[ResourceKindSpiritStone] < cost {
		return nil, fmt.Errorf("%w: cave reserve requires %d spirit_stone", errInsufficientResources, cost)
	}
	ensureInstitutionState(&a.state)
	cave := a.state.Institutions.ByID[InstitutionIDCave]
	if !cave.Enabled || len(cave.CaveSlots) == 0 {
		return nil, fmt.Errorf("%w: no cave seat available", errTaskRequirementNotMet)
	}
	slotIndex := -1
	for index, slot := range cave.CaveSlots {
		if slot.OccupiedBy == nil || slot.ReservedUntilDay <= a.state.Time.CalendarDay {
			slotIndex = index
			break
		}
	}
	if slotIndex < 0 {
		return nil, fmt.Errorf("%w: cave seats are occupied", errTaskRequirementNotMet)
	}

	next := disciple
	next.Support.HousingSatisfied = true
	next.Pressure = maxInt(0, next.Pressure-6)
	next.WorkTarget = DiscipleWorkTargetState{Description: fmt.Sprintf("cultivation:cave:%d_day", durationDays)}
	cave.CaveSlots = cloneCaveSlots(cave.CaveSlots)
	discipleID := payload.DiscipleID
	cave.CaveSlots[slotIndex].OccupiedBy = &discipleID
	cave.CaveSlots[slotIndex].ReservedUntilDay = a.state.Time.CalendarDay + durationDays
	cave.EffectSummary = institutionEffectSummary(a.state, cave)

	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeResourceChanged, ResourceChangedPayload{
			Changes: map[ResourceKind]int64{ResourceKindSpiritStone: -cost},
			Reason:  "reserve_cave",
		}, a.state.Runtime.Version+1),
		a.newDomainEvent(command.CmdID, DomainEventTypeDiscipleAssignmentChanged, DiscipleChangedPayload{Disciple: next}, a.state.Runtime.Version+2),
		a.newDomainEvent(command.CmdID, DomainEventTypeInstitutionChanged, InstitutionChangedPayload{
			Institution: cave,
			Reason:      "reserve_cave",
		}, a.state.Runtime.Version+3),
	}, nil
}

func (a *SectActor) handleAttemptBreakthrough(command ClientCommand) ([]DomainEvent, error) {
	var payload AttemptBreakthroughPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("attempt breakthrough payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode attempt breakthrough payload: %w", err)
	}

	disciple, ok := a.state.Disciples[payload.DiscipleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errDiscipleNotFound, payload.DiscipleID)
	}
	if !disciple.Realm.ReadyForBreakthrough {
		return nil, fmt.Errorf("%w: disciple %s is not ready for breakthrough", errCultivationNotReady, payload.DiscipleID)
	}
	pendingPortent := activeBreakthroughPortentForDisciple(a.state.Events, payload.DiscipleID)
	if pendingPortent == nil || pendingPortent.Status != SectEventStatusForeshadowed {
		return nil, fmt.Errorf("%w: disciple %s fate portent has not been revealed", errCultivationNotReady, payload.DiscipleID)
	}
	if disciple.InjuryLevel >= 3 {
		return nil, fmt.Errorf("%w: disciple %s injury too high", errRiskTooHigh, payload.DiscipleID)
	}

	stageCost := breakthroughSpiritStoneCost(disciple.Realm.Stage)
	if stageCost > 0 && a.state.Resources.Stock[ResourceKindSpiritStone] < stageCost {
		return nil, fmt.Errorf("%w: breakthrough requires %d spirit_stone", errInsufficientResources, stageCost)
	}

	events := make([]DomainEvent, 0, 2)
	versionCursor := a.state.Runtime.Version
	if stageCost > 0 {
		versionCursor++
		events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeResourceChanged, ResourceChangedPayload{
			Changes: map[ResourceKind]int64{
				ResourceKindSpiritStone: -stageCost,
			},
			Reason: "attempt_breakthrough",
		}, versionCursor))
	}

	next := disciple
	if breakthroughRiskScore(disciple) <= breakthroughRiskLimit(disciple.Realm.Stage) {
		next.Realm.Stage = nextRealmStage(disciple.Realm.Stage)
		next.Realm.CultivationPoints = 0
		next.Realm.ReadyForBreakthrough = false
		next.Realm.FailedBreakthroughCount = 0
		if next.Pressure >= 10 {
			next.Pressure -= 10
		} else {
			next.Pressure = 0
		}
		versionCursor++
		events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeBreakthroughSucceeded, DiscipleChangedPayload{Disciple: next}, versionCursor))
		if pending := activeBreakthroughPortentForDisciple(a.state.Events, payload.DiscipleID); pending != nil {
			versionCursor++
			events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeSectEventResolved, SectEventResolvedPayload{
				EventID: pending.EventID,
				Summary: ResolvedEventSummary{
					EventID:           pending.EventID,
					Kind:              pending.Kind,
					Outcome:           "breakthrough_succeeded",
					Summary:           fmt.Sprintf("%s 顺利突破至 %s。", disciple.Name, formatRealmStageLabel(next.Realm.Stage)),
					ResolvedAtVersion: versionCursor,
				},
			}, versionCursor))
		}
		return events, nil
	}

	next.Realm.FailedBreakthroughCount++
	next.Realm.CultivationPoints = breakthroughFailurePoints(disciple)
	next.Realm.ReadyForBreakthrough = next.Realm.CultivationPoints >= cultivationThresholdForStage(next.Realm.Stage)
	next.Pressure = minInt(100, next.Pressure+18)
	next.InjuryLevel = minInt(3, next.InjuryLevel+1)
	next.HP = maxInt64(1, next.HP-15)
	if next.InjuryLevel > 0 {
		next.Support.MedicalSupported = false
	}
	versionCursor++
	events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeBreakthroughFailed, DiscipleChangedPayload{Disciple: next}, versionCursor))
	if pending := activeBreakthroughPortentForDisciple(a.state.Events, payload.DiscipleID); pending != nil {
		versionCursor++
		events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeSectEventResolved, SectEventResolvedPayload{
			EventID: pending.EventID,
			Summary: ResolvedEventSummary{
				EventID:           pending.EventID,
				Kind:              pending.Kind,
				Outcome:           "breakthrough_failed",
				Summary:           fmt.Sprintf("%s 的突破气机反噬，需再稳固根基。", disciple.Name),
				ResolvedAtVersion: versionCursor,
			},
		}, versionCursor))
	}
	return events, nil
}

func (a *SectActor) handleStartRecruitment(command ClientCommand) ([]DomainEvent, error) {
	var payload StartRecruitmentPayload
	if len(command.Payload) > 0 {
		if err := json.Unmarshal(command.Payload, &payload); err != nil {
			return nil, fmt.Errorf("decode start recruitment payload: %w", err)
		}
	}
	if payload.InvestmentSpiritStone < 0 {
		return nil, errors.New("investmentSpiritStone cannot be negative")
	}
	candidateCount := payload.CandidateCount
	if candidateCount <= 0 {
		ensureInstitutionState(&a.state)
		candidateCount = recruitmentCandidateCountForPolicy(a.state.Policies.RecruitmentPolicy)
		candidateCount += institutionRecruitmentBonus(a.state)
	}
	if candidateCount > 5 {
		candidateCount = 5
	}
	durationDays := payload.DurationDays
	if durationDays <= 0 {
		durationDays = 7
	}

	ensureAdmissionState(&a.state)
	ensureInstitutionState(&a.state)
	if len(a.state.Admissions.Candidates) > 0 {
		return nil, fmt.Errorf("%w: recruitment candidates already pending", errTaskRequirementNotMet)
	}

	admissions := cloneAdmissions(a.state.Admissions)
	recruitmentID := fmt.Sprintf("recruitment-%d", a.state.Runtime.Version+1)
	admissions.CurrentRecruitment = &RecruitmentSession{
		RecruitmentID:         recruitmentID,
		Type:                  RecruitmentTypeOpenMountain,
		StartedAtCalendarDay:  a.state.Time.CalendarDay,
		EndsAtCalendarDay:     a.state.Time.CalendarDay + durationDays,
		InvestmentSpiritStone: payload.InvestmentSpiritStone,
		CandidateCount:        candidateCount,
	}
	admissions.Candidates = make(map[CandidateID]CandidateState, candidateCount)
	for index := 0; index < candidateCount; index++ {
		candidate := buildRecruitmentCandidate(a.state, RecruitmentTypeOpenMountain, index, payload.InvestmentSpiritStone)
		admissions.Candidates[candidate.CandidateID] = candidate
	}

	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeRecruitmentStarted, RecruitmentStartedPayload{Admissions: admissions}, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handleAcceptCandidate(command ClientCommand) ([]DomainEvent, error) {
	var payload AcceptCandidatePayload
	if len(command.Payload) == 0 {
		return nil, errors.New("accept candidate payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode accept candidate payload: %w", err)
	}
	ensureAdmissionState(&a.state)
	candidate, ok := a.state.Admissions.Candidates[payload.CandidateID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errCandidateNotFound, payload.CandidateID)
	}
	ensureInstitutionState(&a.state)
	capacity := discipleRosterCapacity(a.state)
	if len(a.state.Disciples) >= capacity {
		return nil, fmt.Errorf("%w: disciple roster capacity %d reached", errTaskRequirementNotMet, capacity)
	}

	admissions := cloneAdmissions(a.state.Admissions)
	delete(admissions.Candidates, payload.CandidateID)
	if admissions.CurrentRecruitment != nil {
		admissions.CurrentRecruitment.CandidateCount = len(admissions.Candidates)
		if len(admissions.Candidates) == 0 {
			admissions.CurrentRecruitment = nil
		}
	}
	disciple := newDiscipleFromCandidate(candidate, nextDiscipleID(a.state.Disciples))

	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeCandidateAccepted, CandidateAcceptedPayload{
			CandidateID: payload.CandidateID,
			Disciple:    disciple,
			Admissions:  admissions,
		}, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handleRejectCandidate(command ClientCommand) ([]DomainEvent, error) {
	var payload RejectCandidatePayload
	if len(command.Payload) == 0 {
		return nil, errors.New("reject candidate payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode reject candidate payload: %w", err)
	}
	ensureAdmissionState(&a.state)
	if _, ok := a.state.Admissions.Candidates[payload.CandidateID]; !ok {
		return nil, fmt.Errorf("%w: %s", errCandidateNotFound, payload.CandidateID)
	}

	admissions := cloneAdmissions(a.state.Admissions)
	delete(admissions.Candidates, payload.CandidateID)
	if admissions.CurrentRecruitment != nil {
		admissions.CurrentRecruitment.CandidateCount = len(admissions.Candidates)
		if len(admissions.Candidates) == 0 {
			admissions.CurrentRecruitment = nil
		}
	}

	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeCandidateRejected, CandidateRejectedPayload{
			CandidateID: payload.CandidateID,
			Admissions:  admissions,
		}, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handleChooseEventOption(command ClientCommand) ([]DomainEvent, error) {
	var payload ChooseEventOptionPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("choose event option payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode choose event option payload: %w", err)
	}
	event, ok := a.state.Events.ActiveEvents[payload.EventID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errEventNotFound, payload.EventID)
	}
	if eventExpiredAtDay(event, a.state.Time.CalendarDay) {
		return nil, fmt.Errorf("%w: %s expired at day %d", errEventExpired, payload.EventID, event.ExpiresAtDay)
	}
	if event.Status != SectEventStatusForeshadowed {
		return nil, fmt.Errorf("%w: event %s is not ready for player choice", errEventRequirementNotMet, payload.EventID)
	}
	if !sectEventRequirementMet(a.state, event.Requirements) {
		return nil, fmt.Errorf("%w: event %s requirements are not met", errEventRequirementNotMet, payload.EventID)
	}
	option, ok := findSectEventOption(event, payload.OptionID)
	if !ok {
		return nil, fmt.Errorf("%w: option %s", errEventRequirementNotMet, payload.OptionID)
	}
	if !sectEventRequirementMet(a.state, option.Requirements) {
		return nil, fmt.Errorf("%w: option %s requirements are not met", errEventRequirementNotMet, payload.OptionID)
	}
	if !resourceDeltaAffordable(a.state.Resources, option.ResultPreview.ResourceDelta) {
		return nil, fmt.Errorf("%w: event option %s resource delta", errInsufficientResources, payload.OptionID)
	}

	events := make([]DomainEvent, 0, 3)
	versionCursor := a.state.Runtime.Version
	if len(option.ResultPreview.ResourceDelta) > 0 {
		versionCursor++
		events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeResourceChanged, ResourceChangedPayload{
			Changes: option.ResultPreview.ResourceDelta,
			Reason:  fmt.Sprintf("sect_event_option:%s:%s", event.EventID, option.OptionID),
		}, versionCursor))
	}
	if event.SourceDiscipleID != nil && (option.ResultPreview.DisciplePressureDelta != 0 || option.ResultPreview.DiscipleSatisfactionDelta != 0) {
		if disciple, ok := a.state.Disciples[*event.SourceDiscipleID]; ok {
			next := disciple
			next.Pressure = clampInt(next.Pressure+option.ResultPreview.DisciplePressureDelta, 0, 100)
			next.Satisfaction = clampInt(next.Satisfaction+option.ResultPreview.DiscipleSatisfactionDelta, 0, 100)
			versionCursor++
			events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeDiscipleSatisfactionChanged, DiscipleChangedPayload{Disciple: next}, versionCursor))
		}
	}

	versionCursor++
	events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeSectEventResolved, SectEventResolvedPayload{
		EventID: event.EventID,
		Summary: ResolvedEventSummary{
			EventID:           event.EventID,
			Kind:              event.Kind,
			Outcome:           fmt.Sprintf("option:%s", option.OptionID),
			Summary:           sectEventOptionSummary(event, option),
			ResolvedAtVersion: versionCursor,
		},
	}, versionCursor))
	return events, nil
}

func (a *SectActor) handleDismissEvent(command ClientCommand) ([]DomainEvent, error) {
	var payload DismissEventPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("dismiss event payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode dismiss event payload: %w", err)
	}
	event, ok := a.state.Events.ActiveEvents[payload.EventID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errEventNotFound, payload.EventID)
	}
	if eventExpiredAtDay(event, a.state.Time.CalendarDay) {
		return nil, fmt.Errorf("%w: %s expired at day %d", errEventExpired, payload.EventID, event.ExpiresAtDay)
	}
	version := a.state.Runtime.Version + 1
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeSectEventResolved, SectEventResolvedPayload{
			EventID: event.EventID,
			Summary: ResolvedEventSummary{
				EventID:           event.EventID,
				Kind:              event.Kind,
				Outcome:           "dismissed",
				Summary:           fmt.Sprintf("宗门暂不处理《%s》，事件已由掌门搁置。", event.Title),
				ResolvedAtVersion: version,
			},
		}, version),
	}, nil
}

func (a *SectActor) handleStartAssessment(command ClientCommand) ([]DomainEvent, error) {
	var payload StartAssessmentPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("start assessment payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode start assessment payload: %w", err)
	}
	disciple, ok := a.state.Disciples[payload.DiscipleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errDiscipleNotFound, payload.DiscipleID)
	}
	targetRank, err := normalizePromotionTarget(disciple.Identity, payload.TargetRank, payload.TargetRankSnake)
	if err != nil {
		return nil, err
	}

	version := a.state.Runtime.Version + 1
	assessment := evaluatePromotionAssessment(a.state, disciple, targetRank, version)
	next := disciple
	next.Assessment = assessment
	if !assessment.Passed {
		next.Satisfaction = clampInt(next.Satisfaction-2, 0, 100)
	}
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeAssessmentResolved, AssessmentResolvedPayload{
			Disciple:   next,
			Assessment: assessment,
		}, version),
	}, nil
}

func (a *SectActor) handlePromoteDisciple(command ClientCommand) ([]DomainEvent, error) {
	var payload PromoteDisciplePayload
	if len(command.Payload) == 0 {
		return nil, errors.New("promote disciple payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode promote disciple payload: %w", err)
	}
	disciple, ok := a.state.Disciples[payload.DiscipleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errDiscipleNotFound, payload.DiscipleID)
	}
	targetRank, err := normalizePromotionTarget(disciple.Identity, payload.TargetRank, payload.TargetRankSnake)
	if err != nil {
		return nil, err
	}
	assessment := evaluatePromotionAssessment(a.state, disciple, targetRank, a.state.Runtime.Version)
	if !disciple.Assessment.Passed || disciple.Assessment.TargetRank != targetRank {
		return nil, fmt.Errorf("%w: disciple %s has not passed assessment for %s", errTaskRequirementNotMet, payload.DiscipleID, targetRank)
	}
	if !assessment.Passed {
		return nil, fmt.Errorf("%w: disciple %s no longer meets promotion requirements: %s", errTaskRequirementNotMet, payload.DiscipleID, assessment.Reason)
	}

	cost := promotionContributionCost(disciple.Identity, targetRank)
	account := a.state.Contribution.Accounts[payload.DiscipleID]
	if account.Balance < cost {
		return nil, fmt.Errorf("%w: disciple %s balance %d < promotion cost %d", errContributionNotEnough, payload.DiscipleID, account.Balance, cost)
	}

	versionCursor := a.state.Runtime.Version
	events := make([]DomainEvent, 0, 3)
	if cost > 0 {
		versionCursor++
		events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeContributionSpent, ContributionChangedPayload{
			DiscipleID: payload.DiscipleID,
			Delta:      -cost,
			Reason:     fmt.Sprintf("promotion:%s", targetRank),
			Quantity:   1,
		}, versionCursor))
	}

	promoted := disciple
	promoted.Identity = targetRank
	promoted.WorkTarget = DiscipleWorkTargetState{Description: fmt.Sprintf("promotion:%s", targetRank)}
	versionCursor++
	events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeDisciplePromoted, DiscipleChangedPayload{Disciple: promoted}, versionCursor))

	mood := promoted
	mood.Satisfaction = clampInt(mood.Satisfaction+3, 0, 100)
	mood.Loyalty = clampInt(mood.Loyalty+1, 0, 100)
	versionCursor++
	events = append(events, a.newDomainEvent(command.CmdID, DomainEventTypeDiscipleSatisfactionChanged, DiscipleChangedPayload{Disciple: mood}, versionCursor))
	return events, nil
}

func (a *SectActor) handleAssignInstitutionManager(command ClientCommand) ([]DomainEvent, error) {
	var payload AssignInstitutionManagerPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("assign institution manager payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode assign institution manager payload: %w", err)
	}
	ensureInstitutionState(&a.state)
	institution, ok := a.state.Institutions.ByID[payload.InstitutionID]
	if !ok {
		return nil, fmt.Errorf("%w: institution %s does not exist", errTaskRequirementNotMet, payload.InstitutionID)
	}
	disciple, ok := a.state.Disciples[payload.DiscipleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errDiscipleNotFound, payload.DiscipleID)
	}
	next := applyInstitutionManager(a.state, institution, disciple)
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeInstitutionChanged, InstitutionChangedPayload{
			Institution: next,
			Reason:      "assign_manager",
		}, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handleSetGatePolicy(command ClientCommand) ([]DomainEvent, error) {
	var payload SetGatePolicyPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("set gate policy payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode set gate policy payload: %w", err)
	}
	ensureInstitutionState(&a.state)
	gate := a.state.Institutions.ByID[InstitutionIDGate]
	if payload.OpenToVisitors != nil {
		gate.GatePolicy.OpenToVisitors = *payload.OpenToVisitors
	}
	if payload.AllowWanderingCultivators != nil {
		gate.GatePolicy.AllowWanderingCultivators = *payload.AllowWanderingCultivators
	}
	if payload.EnforcementStrictness != nil {
		gate.GatePolicy.EnforcementStrictness = clampInt(*payload.EnforcementStrictness, 0, 2)
	}
	if payload.GuardDiscipleIDs != nil {
		for _, discipleID := range payload.GuardDiscipleIDs {
			if _, ok := a.state.Disciples[discipleID]; !ok {
				return nil, fmt.Errorf("%w: %s", errDiscipleNotFound, discipleID)
			}
		}
		gate.GatePolicy.GuardDiscipleIDs = append([]DiscipleID(nil), payload.GuardDiscipleIDs...)
	}
	gate.EffectSummary = institutionEffectSummary(a.state, gate)
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeInstitutionChanged, InstitutionChangedPayload{
			Institution: gate,
			Reason:      "set_gate_policy",
		}, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handleSetExchangeRule(command ClientCommand) ([]DomainEvent, error) {
	var payload SetExchangeRulePayload
	if len(command.Payload) == 0 {
		return nil, errors.New("set exchange rule payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode set exchange rule payload: %w", err)
	}
	if payload.ExchangeItemID == "" {
		return nil, errors.New("exchangeItemId is required")
	}
	rule, ok := a.state.Contribution.TreasuryRules[payload.ExchangeItemID]
	if !ok {
		return nil, fmt.Errorf("exchange rule %s is not available", payload.ExchangeItemID)
	}
	if payload.ContributionCost != nil {
		if *payload.ContributionCost < 0 {
			return nil, errors.New("contributionCost cannot be negative")
		}
		rule.ContributionCost = *payload.ContributionCost
	}
	if payload.MonthlyLimit != nil {
		if *payload.MonthlyLimit < 0 {
			return nil, errors.New("monthlyLimit cannot be negative")
		}
		rule.MonthlyLimit = *payload.MonthlyLimit
	}
	if payload.Enabled != nil {
		rule.Enabled = *payload.Enabled
	}
	ensureInstitutionState(&a.state)
	treasury := a.state.Institutions.ByID[InstitutionIDTreasury]
	treasury.PublicExchange = rule.Enabled
	treasury.ExchangePressure = institutionExchangePressure(a.state, treasury)
	treasury.EffectSummary = institutionEffectSummary(a.state, treasury)
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeInstitutionChanged, InstitutionChangedPayload{
			Institution:  treasury,
			ExchangeRule: &rule,
			Reason:       "set_exchange_rule",
		}, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handleCraftArtifact(command ClientCommand) ([]DomainEvent, error) {
	var payload CraftArtifactPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("craft artifact payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode craft artifact payload: %w", err)
	}
	artifactType := normalizeCraftArtifactType(payload)
	if artifactType == "" {
		return nil, errors.New("artifactType is required")
	}
	if err := validateArtifactCraftBuilding(a.state, artifactType); err != nil {
		return nil, err
	}
	quality := clampArtifactQuality(payload.Quality)
	cost, err := artifactCraftCost(artifactType, quality)
	if err != nil {
		return nil, err
	}
	if !CanAfford(a.state.Resources, cost) {
		return nil, fmt.Errorf("%w: craft %s", errInsufficientResources, artifactType)
	}
	itemID := nextArtifactID(a.state)
	artifact := newArtifactState(itemID, artifactType, quality, "crafted")
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeResourceChanged, ResourceChangedPayload{
			Changes: invertCost(cost),
			Reason:  "craft_artifact",
		}, a.state.Runtime.Version+1),
		a.newDomainEvent(command.CmdID, DomainEventTypeArtifactChanged, ArtifactChangedPayload{
			Artifact: artifact,
			Reason:   "craft_artifact",
		}, a.state.Runtime.Version+2),
	}, nil
}

func (a *SectActor) handleEquipArtifact(command ClientCommand) ([]DomainEvent, error) {
	var payload EquipArtifactPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("equip artifact payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode equip artifact payload: %w", err)
	}
	if payload.ItemID == "" || payload.DiscipleID == "" {
		return nil, errors.New("itemId and discipleId are required")
	}
	ensureInventoryState(&a.state)
	artifact, ok := a.state.Inventory.Artifacts[payload.ItemID]
	if !ok {
		return nil, fmt.Errorf("%w: artifact %s", errTaskRequirementNotMet, payload.ItemID)
	}
	if artifact.Durability <= 0 {
		return nil, fmt.Errorf("%w: artifact %s has no durability", errTaskRequirementNotMet, payload.ItemID)
	}
	disciple, ok := a.state.Disciples[payload.DiscipleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errDiscipleNotFound, payload.DiscipleID)
	}
	if artifact.BoundDiscipleID != "" && artifact.BoundDiscipleID != payload.DiscipleID {
		return nil, fmt.Errorf("%w: artifact %s is bound to %s", errTaskRequirementNotMet, payload.ItemID, artifact.BoundDiscipleID)
	}
	slot := artifactSlotForType(artifact.Type)
	if equipped := equipmentItemForSlot(disciple.Equipment, slot); equipped != "" && equipped != payload.ItemID {
		return nil, fmt.Errorf("%w: disciple %s already has %s equipped", errTaskRequirementNotMet, payload.DiscipleID, slot)
	}
	artifact.BoundDiscipleID = payload.DiscipleID
	disciple.Equipment = setEquipmentItemForSlot(disciple.Equipment, slot, payload.ItemID)
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeArtifactChanged, ArtifactChangedPayload{
			Artifact: artifact,
			Disciple: &disciple,
			Reason:   "equip_artifact",
		}, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handleUnequipArtifact(command ClientCommand) ([]DomainEvent, error) {
	var payload UnequipArtifactPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("unequip artifact payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode unequip artifact payload: %w", err)
	}
	if payload.ItemID == "" {
		return nil, errors.New("itemId is required")
	}
	ensureInventoryState(&a.state)
	artifact, ok := a.state.Inventory.Artifacts[payload.ItemID]
	if !ok {
		return nil, fmt.Errorf("%w: artifact %s", errTaskRequirementNotMet, payload.ItemID)
	}
	discipleID := artifact.BoundDiscipleID
	if payload.DiscipleID != "" {
		discipleID = payload.DiscipleID
	}
	if discipleID == "" {
		return nil, fmt.Errorf("%w: artifact %s is not equipped", errTaskRequirementNotMet, payload.ItemID)
	}
	if artifact.BoundDiscipleID != "" && artifact.BoundDiscipleID != discipleID {
		return nil, fmt.Errorf("%w: artifact %s is bound to %s", errTaskRequirementNotMet, payload.ItemID, artifact.BoundDiscipleID)
	}
	disciple, ok := a.state.Disciples[discipleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errDiscipleNotFound, discipleID)
	}
	artifact.BoundDiscipleID = ""
	disciple.Equipment = clearEquipmentItem(disciple.Equipment, payload.ItemID)
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeArtifactChanged, ArtifactChangedPayload{
			Artifact: artifact,
			Disciple: &disciple,
			Reason:   "unequip_artifact",
		}, a.state.Runtime.Version+1),
	}, nil
}

func (a *SectActor) handleRepairArtifact(command ClientCommand) ([]DomainEvent, error) {
	var payload RepairArtifactPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("repair artifact payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode repair artifact payload: %w", err)
	}
	if payload.ItemID == "" {
		return nil, errors.New("itemId is required")
	}
	ensureInventoryState(&a.state)
	artifact, ok := a.state.Inventory.Artifacts[payload.ItemID]
	if !ok {
		return nil, fmt.Errorf("%w: artifact %s", errTaskRequirementNotMet, payload.ItemID)
	}
	if artifact.Durability >= artifact.MaxDurability {
		return nil, fmt.Errorf("%w: artifact %s does not need repair", errTaskRequirementNotMet, payload.ItemID)
	}
	if err := validateArtifactRepairBuilding(a.state, artifact.Type); err != nil {
		return nil, err
	}
	cost := artifactRepairCost(artifact)
	if !CanAfford(a.state.Resources, cost) {
		return nil, fmt.Errorf("%w: repair artifact %s", errInsufficientResources, payload.ItemID)
	}
	artifact.Durability = artifact.MaxDurability
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeResourceChanged, ResourceChangedPayload{
			Changes: invertCost(cost),
			Reason:  "repair_artifact",
		}, a.state.Runtime.Version+1),
		a.newDomainEvent(command.CmdID, DomainEventTypeArtifactChanged, ArtifactChangedPayload{
			Artifact: artifact,
			Reason:   "repair_artifact",
		}, a.state.Runtime.Version+2),
	}, nil
}

func (a *SectActor) handleAttachFormationToBuilding(command ClientCommand) ([]DomainEvent, error) {
	var payload AttachFormationToBuildingPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("attach formation payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode attach formation payload: %w", err)
	}
	if payload.BuildingID == "" || payload.ArtifactItemID == "" || payload.FormationKind == "" {
		return nil, errors.New("buildingId, artifactItemId, and formationKind are required")
	}
	building, ok := a.state.Buildings[payload.BuildingID]
	if !ok {
		return nil, fmt.Errorf("%w: building %s does not exist", errTaskRequirementNotMet, payload.BuildingID)
	}
	if building.Phase != "active" {
		return nil, fmt.Errorf("%w: building %s is not active", errTaskRequirementNotMet, payload.BuildingID)
	}
	ensureInventoryState(&a.state)
	ensureFormationState(&a.state)
	artifact, ok := a.state.Inventory.Artifacts[payload.ArtifactItemID]
	if !ok {
		return nil, fmt.Errorf("%w: artifact %s", errTaskRequirementNotMet, payload.ArtifactItemID)
	}
	if artifact.Type != ArtifactTypeFormationDisk {
		return nil, fmt.Errorf("%w: artifact %s is not a formation disk", errTaskRequirementNotMet, payload.ArtifactItemID)
	}
	if artifact.BoundDiscipleID != "" {
		return nil, fmt.Errorf("%w: formation disk %s is equipped by %s", errTaskRequirementNotMet, payload.ArtifactItemID, artifact.BoundDiscipleID)
	}
	if artifact.AttachedBuildingID != "" {
		return nil, fmt.Errorf("%w: formation disk %s is already attached to %s", errTaskRequirementNotMet, payload.ArtifactItemID, artifact.AttachedBuildingID)
	}
	if artifact.Durability <= 0 {
		return nil, fmt.Errorf("%w: formation disk %s has no durability", errTaskRequirementNotMet, payload.ArtifactItemID)
	}
	if existing, exists := a.state.Formations[payload.BuildingID]; exists {
		return nil, fmt.Errorf("%w: building %s already hosts %s", errTaskRequirementNotMet, payload.BuildingID, existing.Kind)
	}
	if err := validateFormationAttachment(a.state, building, payload.FormationKind); err != nil {
		return nil, err
	}
	artifact.AttachedBuildingID = payload.BuildingID
	formation := newFormationState(payload.BuildingID, artifact, payload.FormationKind, a.state.Runtime.Version+2)
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeArtifactChanged, ArtifactChangedPayload{
			Artifact: artifact,
			Reason:   "attach_formation",
		}, a.state.Runtime.Version+1),
		a.newDomainEvent(command.CmdID, DomainEventTypeFormationChanged, FormationChangedPayload{
			Formation: formation,
			Reason:    "attach_formation",
		}, a.state.Runtime.Version+2),
	}, nil
}

func (a *SectActor) handleDetachFormation(command ClientCommand) ([]DomainEvent, error) {
	var payload DetachFormationPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("detach formation payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode detach formation payload: %w", err)
	}
	if payload.BuildingID == "" {
		return nil, errors.New("buildingId is required")
	}
	ensureFormationState(&a.state)
	formation, ok := a.state.Formations[payload.BuildingID]
	if !ok {
		return nil, fmt.Errorf("%w: building %s has no attached formation", errTaskRequirementNotMet, payload.BuildingID)
	}
	artifact, ok := a.state.Inventory.Artifacts[formation.ArtifactItemID]
	if !ok {
		return nil, fmt.Errorf("%w: formation disk %s", errTaskRequirementNotMet, formation.ArtifactItemID)
	}
	artifact.AttachedBuildingID = ""
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeArtifactChanged, ArtifactChangedPayload{
			Artifact: artifact,
			Reason:   "detach_formation",
		}, a.state.Runtime.Version+1),
		a.newDomainEvent(command.CmdID, DomainEventTypeFormationChanged, FormationChangedPayload{
			Formation: formation,
			Reason:    "detach_formation",
			Detached:  true,
		}, a.state.Runtime.Version+2),
	}, nil
}

func (a *SectActor) handleMaintainFormation(command ClientCommand) ([]DomainEvent, error) {
	var payload MaintainFormationPayload
	if len(command.Payload) == 0 {
		return nil, errors.New("maintain formation payload is required")
	}
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode maintain formation payload: %w", err)
	}
	if payload.BuildingID == "" {
		return nil, errors.New("buildingId is required")
	}
	ensureFormationState(&a.state)
	formation, ok := a.state.Formations[payload.BuildingID]
	if !ok {
		return nil, fmt.Errorf("%w: building %s has no attached formation", errTaskRequirementNotMet, payload.BuildingID)
	}
	if formation.MaintenanceDebt == 0 && formation.Stability >= 100 && formation.Active {
		return nil, fmt.Errorf("%w: formation %s does not need maintenance", errTaskRequirementNotMet, formation.FormationID)
	}
	cost := formationManualMaintenanceCost(formation)
	if !CanAfford(a.state.Resources, cost) {
		return nil, fmt.Errorf("%w: maintain formation %s", errInsufficientResources, formation.FormationID)
	}
	maintained := formationAfterManualMaintenance(formation)
	return []DomainEvent{
		a.newDomainEvent(command.CmdID, DomainEventTypeResourceChanged, ResourceChangedPayload{
			Changes: invertCost(cost),
			Reason:  fmt.Sprintf("maintain_formation:%s", formation.FormationID),
		}, a.state.Runtime.Version+1),
		a.newDomainEvent(command.CmdID, DomainEventTypeFormationChanged, FormationChangedPayload{
			Formation: maintained,
			Reason:    "maintain_formation",
		}, a.state.Runtime.Version+2),
	}, nil
}

func (a *SectActor) advanceTasksOneDay(sessionID string) AdvanceTasksOneDayResponse {
	fromVersion := a.state.Runtime.Version
	events, err := a.buildAdvanceTasksOneDayEvents()
	if err != nil {
		return AdvanceTasksOneDayResponse{
			Snapshot:    a.snapshot(a.state.Meta.OwnerUserID, sessionID),
			FromVersion: fromVersion,
			ToVersion:   fromVersion,
		}
	}
	if err := a.applyDomainEvents(events); err != nil {
		return AdvanceTasksOneDayResponse{
			Snapshot:    a.snapshot(a.state.Meta.OwnerUserID, sessionID),
			FromVersion: fromVersion,
			ToVersion:   fromVersion,
		}
	}
	clientEvents, _, clientErr := clientDeltaForEvents(events)
	if clientErr != nil {
		return AdvanceTasksOneDayResponse{
			Snapshot:    a.snapshot(a.state.Meta.OwnerUserID, sessionID),
			FromVersion: fromVersion,
			ToVersion:   a.state.Runtime.Version,
		}
	}
	return AdvanceTasksOneDayResponse{
		Snapshot:     a.snapshot(a.state.Meta.OwnerUserID, sessionID),
		Events:       clientEvents,
		DomainEvents: cloneDomainEvents(events),
		FromVersion:  fromVersion,
		ToVersion:    a.state.Runtime.Version,
	}
}

func (a *SectActor) buildAdvanceTasksOneDayEvents() ([]DomainEvent, error) {
	simulated := a.state.Clone()
	taskIDs := sortedTaskIDs(simulated.Tasks)
	productionIDs := sortedProductionIDs(simulated.Productions)
	versionCursor := simulated.Runtime.Version
	events := make([]DomainEvent, 0, len(taskIDs)*3+len(productionIDs)*2+len(simulated.Disciples)*2)
	appendEvent := func(event DomainEvent) error {
		if err := ApplyEvent(&simulated, event); err != nil {
			return err
		}
		versionCursor = simulated.Runtime.Version
		events = append(events, event)
		return nil
	}

	if err := appendEvent(a.newDomainEvent("", DomainEventTypeTimeAdvanced, TimeAdvancedPayload{Time: nextDayTime(simulated.Time)}, versionCursor+1)); err != nil {
		return nil, err
	}

	for _, buildingID := range sortedBuildingIDs(simulated.Buildings) {
		building := simulated.Buildings[buildingID]
		if !buildingParticipatesInMaintenance(building) {
			continue
		}
		cost := buildingMaintenanceCostFor(building.DefinitionKey, building.Level)
		if len(cost) == 0 {
			continue
		}
		if CanAfford(simulated.Resources, cost) {
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeResourceChanged, ResourceChangedPayload{
				Changes: invertCost(cost),
				Reason:  fmt.Sprintf("building_maintenance:%s", building.BuildingID),
			}, versionCursor+1)); err != nil {
				return nil, err
			}
			maintained := buildingAfterMaintenancePaid(simulated.Buildings[buildingID])
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeBuildingMaintained, BuildingMaintainedPayload{
				Building: maintained,
				Cost:     cloneResourceAmounts(cost),
				Reason:   "daily_maintenance_paid",
			}, versionCursor+1)); err != nil {
				return nil, err
			}
			continue
		}

		damaged, shortage := buildingAfterMaintenanceShortage(building, cost, simulated.Resources)
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeBuildingDamaged, BuildingMaintainedPayload{
			Building: damaged,
			Cost:     cloneResourceAmounts(cost),
			Shortage: shortage,
			Reason:   "daily_maintenance_shortage",
		}, versionCursor+1)); err != nil {
			return nil, err
		}
	}

	for _, buildingID := range sortedFormationBuildingIDs(simulated.Formations) {
		formation := simulated.Formations[buildingID]
		cost := formationMaintenanceCostFor(formation)
		if len(cost) == 0 {
			continue
		}
		if CanAfford(simulated.Resources, cost) {
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeResourceChanged, ResourceChangedPayload{
				Changes: invertCost(cost),
				Reason:  fmt.Sprintf("formation_maintenance:%s", formation.FormationID),
			}, versionCursor+1)); err != nil {
				return nil, err
			}
			maintained := formationAfterDailyMaintenancePaid(simulated.Formations[buildingID])
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeFormationChanged, FormationChangedPayload{
				Formation: maintained,
				Reason:    "daily_formation_maintenance_paid",
			}, versionCursor+1)); err != nil {
				return nil, err
			}
			continue
		}

		degraded := formationAfterDailyMaintenanceShortage(formation)
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeFormationChanged, FormationChangedPayload{
			Formation: degraded,
			Reason:    "daily_formation_maintenance_shortage",
		}, versionCursor+1)); err != nil {
			return nil, err
		}
	}

	if err := a.appendInstitutionDailyEffectEvents(&simulated, &versionCursor, appendEvent); err != nil {
		return nil, err
	}
	if err := a.appendSectOrderEvents(&simulated, &versionCursor, appendEvent); err != nil {
		return nil, err
	}

	for _, eventID := range sortedEventIDs(simulated.Events.ActiveEvents) {
		current := simulated.Events.ActiveEvents[eventID]
		if current.Status != SectEventStatusSeeded || current.RevealAtVersion > versionCursor+1 {
			continue
		}
		next := current
		next.Status = SectEventStatusForeshadowed
		next.RevealedAtVersion = versionCursor + 1
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectEventForeshadowed, SectEventChangedPayload{Event: next}, versionCursor+1)); err != nil {
			return nil, err
		}
	}

	for _, eventID := range sortedEventIDs(simulated.Events.ActiveEvents) {
		current := simulated.Events.ActiveEvents[eventID]
		if !eventExpiredAtDay(current, simulated.Time.CalendarDay) {
			continue
		}
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectEventExpired, SectEventResolvedPayload{
			EventID: current.EventID,
			Summary: ResolvedEventSummary{
				EventID:           current.EventID,
				Kind:              current.Kind,
				Outcome:           "expired",
				Summary:           fmt.Sprintf("《%s》已过期，authority 未替玩家选择结果。", current.Title),
				ResolvedAtVersion: versionCursor + 1,
			},
		}, versionCursor+1)); err != nil {
			return nil, err
		}
	}

	if shouldSeedSectGovernanceChoiceEvent(simulated) {
		seededAtVersion := versionCursor + 1
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectEventForeshadowed, SectEventChangedPayload{
			Event: buildSectGovernanceChoiceEvent(simulated, seededAtVersion),
		}, seededAtVersion)); err != nil {
			return nil, err
		}
	}

	for _, taskID := range taskIDs {
		task := simulated.Tasks[taskID]
		if task.Status != TaskStatusAccepted || len(task.AssignedDiscipleIDs) == 0 {
			continue
		}
		if !allDisciplesExist(simulated.Disciples, task.AssignedDiscipleIDs) {
			continue
		}

		nextTask := task
		nextTask.SuccessRate = taskSuccessRate(simulated, task)
		nextTask.ProgressTicks++
		nextTask.CompletedProgressDays++

		if nextTask.CompletedProgressDays < nextTask.RequiredProgressDays {
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeTaskProgressed, TaskChangedPayload{Task: nextTask}, versionCursor+1)); err != nil {
				return nil, err
			}
			continue
		}

		nextTask.Evaluation = taskEvaluationForRate(nextTask.SuccessRate)
		if nextTask.Evaluation == TaskEvaluationFailed {
			nextTask.Status = TaskStatusFailed
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeTaskFailed, TaskChangedPayload{Task: nextTask}, versionCursor+1)); err != nil {
				return nil, err
			}
			if err := a.appendTaskRiskConsequenceEvents(&simulated, &versionCursor, appendEvent, nextTask, nextTask.SuccessRate, true); err != nil {
				return nil, err
			}
			for _, discipleID := range sortedDiscipleIDs(task.AssignedDiscipleIDs) {
				disciple := simulated.Disciples[discipleID]
				cleared := clearDiscipleTaskAssignment(disciple)
				if err := appendEvent(a.newDomainEvent("", DomainEventTypeDiscipleAssignmentChanged, DiscipleChangedPayload{Disciple: cleared}, versionCursor+1)); err != nil {
					return nil, err
				}
			}
			continue
		}

		nextTask.Status = TaskStatusCompleted
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeTaskCompleted, TaskChangedPayload{Task: nextTask}, versionCursor+1)); err != nil {
			return nil, err
		}

		if nextTask.Evaluation == TaskEvaluationPoor {
			if err := a.appendTaskRiskConsequenceEvents(&simulated, &versionCursor, appendEvent, nextTask, nextTask.SuccessRate, false); err != nil {
				return nil, err
			}
		}

		for _, discipleID := range sortedDiscipleIDs(task.AssignedDiscipleIDs) {
			disciple := simulated.Disciples[discipleID]
			cleared := clearDiscipleTaskAssignment(disciple)
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeDiscipleAssignmentChanged, DiscipleChangedPayload{Disciple: cleared}, versionCursor+1)); err != nil {
				return nil, err
			}

			if nextTask.ContributionReward > 0 {
				if err := appendEvent(a.newDomainEvent("", DomainEventTypeContributionEarned, ContributionChangedPayload{
					DiscipleID: discipleID,
					Delta:      nextTask.ContributionReward,
					Reason:     "task_completed",
					Quantity:   1,
				}, versionCursor+1)); err != nil {
					return nil, err
				}
			}
		}

		if metaChangedByTaskReward(nextTask) {
			nextMeta := applyTaskMetaRewards(simulated.Meta, nextTask)
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectMetaChanged, SectMetaChangedPayload{
				Meta:   nextMeta,
				Reason: fmt.Sprintf("task_completed:%s", nextTask.Kind),
			}, versionCursor+1)); err != nil {
				return nil, err
			}
		}

		if len(nextTask.RewardResources) > 0 {
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeResourceChanged, ResourceChangedPayload{
				Changes: nextTask.RewardResources,
				Reason:  "task_completed",
			}, versionCursor+1)); err != nil {
				return nil, err
			}
		}
	}

	for _, productionID := range productionIDs {
		job := simulated.Productions[productionID]
		nextJob := job
		if productionClosed(job.Status) {
			continue
		}

		if job.Status == ProductionStatusBlocked {
			shortage := productionShortage(simulated.Resources, job.InputCost)
			if len(shortage) > 0 {
				nextJob.BlockedReason = "input_shortage"
				nextJob.Shortage = shortage
				if err := appendEvent(a.newDomainEvent("", DomainEventTypeProductionChanged, ProductionChangedPayload{Production: nextJob}, versionCursor+1)); err != nil {
					return nil, err
				}
				continue
			}
			nextJob.Status = ProductionStatusRunning
			nextJob.BlockedReason = ""
			nextJob.Shortage = nil
		} else {
			nextJob.ProgressDays++
		}

		if nextJob.ProgressDays < nextJob.RequiredProgressDays {
			nextJob.Status = ProductionStatusRunning
			nextJob.BlockedReason = ""
			nextJob.Shortage = nil
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeProductionChanged, ProductionChangedPayload{Production: nextJob}, versionCursor+1)); err != nil {
				return nil, err
			}
			continue
		}

		if shortage := productionShortage(simulated.Resources, nextJob.InputCost); len(shortage) > 0 {
			nextJob.Status = ProductionStatusBlocked
			nextJob.ProgressDays = nextJob.RequiredProgressDays
			nextJob.BlockedReason = "input_shortage"
			nextJob.Shortage = shortage
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeProductionChanged, ProductionChangedPayload{Production: nextJob}, versionCursor+1)); err != nil {
				return nil, err
			}
			continue
		}

		delta := productionCompletionDeltaForState(simulated, nextJob)
		if len(delta) > 0 {
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeResourceChanged, ResourceChangedPayload{
				Changes: delta,
				Reason:  fmt.Sprintf("production_completed:%s", nextJob.RecipeID),
			}, versionCursor+1)); err != nil {
				return nil, err
			}
		}

		nextJob.Status = ProductionStatusRunning
		nextJob.ProgressDays = 0
		nextJob.CompletedCycles++
		nextJob.BlockedReason = ""
		nextJob.Shortage = nil
		if nextJob.TargetCycles > 0 && nextJob.CompletedCycles >= nextJob.TargetCycles {
			nextJob.Status = ProductionStatusCompleted
		}
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeProductionChanged, ProductionChangedPayload{Production: nextJob}, versionCursor+1)); err != nil {
			return nil, err
		}
	}

	for _, discipleID := range sortedDiscipleMapIDs(simulated.Disciples) {
		disciple := simulated.Disciples[discipleID]
		if disciple.AssignmentKind != DiscipleAssignmentCultivation || disciple.Realm.ReadyForBreakthrough {
			continue
		}
		if simulated.Resources.Stock[ResourceKindSpiritStone] <= 0 {
			continue
		}

		if err := appendEvent(a.newDomainEvent("", DomainEventTypeResourceChanged, ResourceChangedPayload{
			Changes: map[ResourceKind]int64{
				ResourceKindSpiritStone: -1,
			},
			Reason: "cultivation_daily",
		}, versionCursor+1)); err != nil {
			return nil, err
		}

		next := simulated.Disciples[discipleID]
		next.Realm.CultivationPoints += dailyCultivationPointsForPolicy(simulated, next)
		next.Realm.ReadyForBreakthrough = next.Realm.CultivationPoints >= cultivationThresholdForStage(next.Realm.Stage)
		next.WorkTarget = DiscipleWorkTargetState{Description: "cultivation:daily"}
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeCultivationAdvanced, DiscipleChangedPayload{Disciple: next}, versionCursor+1)); err != nil {
			return nil, err
		}
	}

	for _, discipleID := range sortedDiscipleMapIDs(simulated.Disciples) {
		disciple := simulated.Disciples[discipleID]
		if !disciple.Realm.ReadyForBreakthrough {
			continue
		}
		if activeBreakthroughPortentForDisciple(simulated.Events, discipleID) != nil {
			continue
		}
		seededAtVersion := versionCursor + 1
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectEventSeeded, SectEventChangedPayload{
			Event: buildBreakthroughPortentEvent(disciple, seededAtVersion),
		}, seededAtVersion)); err != nil {
			return nil, err
		}
	}

	if err := a.appendMonthlySettlementEvents(&simulated, &versionCursor, appendEvent); err != nil {
		return nil, err
	}
	if err := a.appendMonthlyAssessmentEvents(&simulated, &versionCursor, appendEvent); err != nil {
		return nil, err
	}
	if err := a.appendSectGoalProgressEvents(&simulated, &versionCursor, appendEvent); err != nil {
		return nil, err
	}
	if err := a.appendSectCrisisChainEvents(&simulated, &versionCursor, appendEvent); err != nil {
		return nil, err
	}

	return events, nil
}

func ApplyEvent(state *SectState, event DomainEvent) error {
	switch event.Type {
	case DomainEventTypeResourceChanged:
		var payload ResourceChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		if err := ApplyResourceDelta(&state.Resources, payload.Changes); err != nil {
			return err
		}
		recalculateContributionMetrics(state)
	case DomainEventTypeSectMetaChanged:
		var payload SectMetaChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		state.Meta = payload.Meta
		if state.Meta.Relations == nil {
			state.Meta.Relations = map[string]int{}
		}
	case DomainEventTypeTimeAdvanced:
		var payload TimeAdvancedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		state.Time = payload.Time
	case DomainEventTypeBuildingBuilt:
		var payload BuildingBuiltPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		state.Buildings[payload.Building.BuildingID] = payload.Building
		refreshSectBuildingMeta(state)
	case DomainEventTypeBuildingUpgraded:
		var payload BuildingUpgradedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		state.Buildings[payload.Building.BuildingID] = payload.Building
		refreshSectBuildingMeta(state)
	case DomainEventTypeBuildingMaintained, DomainEventTypeBuildingDamaged:
		var payload BuildingMaintainedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		if state.Buildings == nil {
			state.Buildings = map[BuildingID]BuildingState{}
		}
		state.Buildings[payload.Building.BuildingID] = payload.Building
		refreshSectBuildingMeta(state)
	case DomainEventTypeTaskPublished, DomainEventTypeTaskAccepted, DomainEventTypeTaskCancelled, DomainEventTypeTaskPriorityChanged, DomainEventTypeTaskProgressed, DomainEventTypeTaskCompleted, DomainEventTypeTaskFailed:
		var payload TaskChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		if state.Tasks == nil {
			state.Tasks = map[TaskID]TaskState{}
		}
		state.Tasks[payload.Task.TaskID] = payload.Task
		if event.Type == DomainEventTypeTaskProgressed || event.Type == DomainEventTypeTaskCompleted {
			applyMonthlyDutyProgress(state, payload.Task)
		}
		applyTaskMemoryFromEvent(state, event.Type, payload.Task, event.Version)
	case DomainEventTypeDiscipleAssignmentChanged, DomainEventTypeCultivationAdvanced, DomainEventTypeBreakthroughSucceeded, DomainEventTypeBreakthroughFailed, DomainEventTypeDiscipleSatisfactionChanged, DomainEventTypeDiscipleLoyaltyChanged, DomainEventTypeDisciplePromoted:
		var payload DiscipleChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		if state.Disciples == nil {
			state.Disciples = map[DiscipleID]DiscipleState{}
		}
		previous := normalizeDiscipleState(state.Disciples[payload.Disciple.DiscipleID])
		next := normalizeDiscipleState(payload.Disciple)
		state.Disciples[next.DiscipleID] = next
		if event.Type == DomainEventTypeDisciplePromoted {
			ensureMonthlyState(state)
			state.Monthly.Obligations.RequiredDays[next.DiscipleID] = monthlyDutyRequiredDays(next)
		}
		applyDiscipleMemoryFromEvent(state, event.Type, previous, next, event.Version)
	case DomainEventTypeAssessmentResolved:
		var payload AssessmentResolvedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		if state.Disciples == nil {
			state.Disciples = map[DiscipleID]DiscipleState{}
		}
		previous := normalizeDiscipleState(state.Disciples[payload.Disciple.DiscipleID])
		next := normalizeDiscipleState(payload.Disciple)
		state.Disciples[next.DiscipleID] = next
		applyAssessmentMemoryFromEvent(state, previous, payload.Assessment, next, event.Version)
	case DomainEventTypePayrollPaid:
		var payload PayrollSettlementPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		ensureMonthlyState(state)
		state.Monthly.Payroll.LastPaidMonth = maxInt64(state.Monthly.Payroll.LastPaidMonth, payload.MonthIndex)
		state.Monthly.Payroll.Arrears[payload.DiscipleID] = payload.ArrearsAfter
		applyPayrollMemoryFromEvent(state, event.Type, payload, event.Version)
	case DomainEventTypePayrollDelayed:
		var payload PayrollSettlementPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		ensureMonthlyState(state)
		state.Monthly.Payroll.Arrears[payload.DiscipleID] = payload.ArrearsAfter
		applyPayrollMemoryFromEvent(state, event.Type, payload, event.Version)
	case DomainEventTypeMonthlyObligationChecked:
		var payload MonthlyObligationCheckedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		ensureMonthlyState(state)
		state.Monthly.Obligations.MonthIndex = payload.MonthIndex
		state.Monthly.Obligations.RequiredDays[payload.DiscipleID] = payload.RequiredDays
		state.Monthly.Obligations.CompletedDays[payload.DiscipleID] = payload.CompletedDays
		state.Monthly.Obligations.Violations[payload.DiscipleID] = payload.ViolationsAfter
		applyMonthlyObligationMemoryFromEvent(state, payload, event.Version)
	case DomainEventTypeMonthAdvanced:
		var payload MonthAdvancedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		state.Monthly = cloneMonthlyState(payload.Monthly)
		state.Contribution.MonthlyPurchases = map[DiscipleID]map[ExchangeItemID]int64{}
		ensureMonthlyState(state)
	case DomainEventTypeContributionEarned, DomainEventTypeContributionSpent:
		var payload ContributionChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		if err := applyContributionDelta(state, payload); err != nil {
			return err
		}
	case DomainEventTypeInventoryChanged:
		var payload InventoryChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		ensureInventoryState(state)
		state.Inventory.Items[payload.Item.ItemID] = payload.Item
	case DomainEventTypeArtifactChanged:
		var payload ArtifactChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		ensureInventoryState(state)
		payload.Artifact.Stats = cloneStringIntMap(payload.Artifact.Stats)
		state.Inventory.Artifacts[payload.Artifact.ItemID] = payload.Artifact
		if payload.Disciple != nil {
			if state.Disciples == nil {
				state.Disciples = map[DiscipleID]DiscipleState{}
			}
			state.Disciples[payload.Disciple.DiscipleID] = *payload.Disciple
		}
	case DomainEventTypeFormationChanged:
		var payload FormationChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		ensureFormationState(state)
		if payload.Detached {
			delete(state.Formations, payload.Formation.BuildingID)
			break
		}
		state.Formations[payload.Formation.BuildingID] = normalizeFormationState(payload.Formation)
	case DomainEventTypeProductionChanged:
		var payload ProductionChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		if state.Productions == nil {
			state.Productions = map[ProductionID]ProductionJob{}
		}
		state.Productions[payload.Production.ProductionID] = payload.Production
	case DomainEventTypeSectGoalChanged:
		var payload SectGoalChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		ensureSectGoalState(state)
		state.Goals.ByID[payload.Goal.GoalID] = normalizeSectGoal(payload.Goal)
	case DomainEventTypeSectGoalResolved:
		var payload SectGoalResolvedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		ensureSectGoalState(state)
		resolved := normalizeSectGoal(payload.Goal)
		state.Goals.ByID[resolved.GoalID] = resolved
		state.Goals.Resolved = append(state.Goals.Resolved, payload.Summary)
		applySectGoalMemoryFromEvent(state, resolved, payload.Summary, event.Type)
	case DomainEventTypeSectEventSeeded, DomainEventTypeSectEventForeshadowed:
		var payload SectEventChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		ensureSectEventState(state)
		state.Events.ActiveEvents[payload.Event.EventID] = payload.Event
		state.Events.Tension = activeSectEventTension(state.Events.ActiveEvents)
		applySectEventForeshadowMemoryFromEvent(state, payload.Event, event.Type, event.Version)
	case DomainEventTypeSectEventResolved, DomainEventTypeSectEventExpired:
		var payload SectEventResolvedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		ensureSectEventState(state)
		sourceEvent, hadSource := state.Events.ActiveEvents[payload.EventID]
		delete(state.Events.ActiveEvents, payload.EventID)
		state.Events.ResolvedEvents = append(state.Events.ResolvedEvents, payload.Summary)
		state.Events.LastMajorEventVersion = payload.Summary.ResolvedAtVersion
		state.Events.Tension = activeSectEventTension(state.Events.ActiveEvents)
		if hadSource {
			applySectEventResolutionMemoryFromEvent(state, sourceEvent, payload.Summary, event.Type)
		}
	case DomainEventTypeRecruitmentStarted:
		var payload RecruitmentStartedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		state.Admissions = cloneAdmissions(payload.Admissions)
	case DomainEventTypeCandidateAccepted:
		var payload CandidateAcceptedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		state.Admissions = cloneAdmissions(payload.Admissions)
		if state.Disciples == nil {
			state.Disciples = map[DiscipleID]DiscipleState{}
		}
		state.Disciples[payload.Disciple.DiscipleID] = normalizeDiscipleState(payload.Disciple)
		ensureContributionAccounts(state)
		ensureMonthlyState(state)
		recalculateContributionMetrics(state)
	case DomainEventTypeCandidateRejected:
		var payload CandidateRejectedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		state.Admissions = cloneAdmissions(payload.Admissions)
	case DomainEventTypePolicyChanged:
		var payload PolicyChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		state.Policies = clonePolicyState(payload.Policies)
		ensurePolicyState(state)
		if len(payload.Tasks) > 0 {
			state.Tasks = cloneTasks(payload.Tasks)
		}
		if len(payload.Productions) > 0 {
			state.Productions = cloneProductions(payload.Productions)
		}
		if len(payload.Tasks) == 0 && len(payload.Productions) == 0 {
			applyPolicyEffectsToState(state)
		}
	case DomainEventTypeInstitutionChanged:
		var payload InstitutionChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		ensureInstitutionState(state)
		state.Institutions.ByID[payload.Institution.InstitutionID] = normalizeInstitutionLoop(payload.Institution)
		if payload.ExchangeRule != nil {
			if state.Contribution.TreasuryRules == nil {
				state.Contribution.TreasuryRules = map[ExchangeItemID]ExchangeRule{}
			}
			state.Contribution.TreasuryRules[payload.ExchangeRule.ExchangeItemID] = *payload.ExchangeRule
			recalculateContributionMetrics(state)
		}
	case DomainEventTypeOrderChanged:
		var payload OrderChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		state.Order = cloneSectOrder(payload.Order)
		ensureSectOrderState(state)
	case DomainEventTypeMonthlyAssessmentResolved:
		var payload MonthlyAssessmentResolvedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		state.MonthlyAssessment = cloneMonthlyAssessmentState(payload.Assessment)
		ensureMonthlyAssessmentState(state)
		applyMonthlyAssessmentMemoryFromEvent(state, payload.Result, event.Type)
	default:
		return fmt.Errorf("unsupported domain event: %s", event.Type)
	}

	ensureInstitutionState(state)
	ensureSectOrderState(state)
	ensureMonthlyAssessmentState(state)
	refreshCultivationDecisionState(state)
	state.Runtime.Version = event.Version
	state.Runtime.LastAppliedEventVersion = event.Version
	state.Runtime.Dirty = true
	return nil
}

const (
	maxDiscipleMemoryEntries        = 8
	recentDiscipleExperienceEntries = 3
)

func normalizeDiscipleState(disciple DiscipleState) DiscipleState {
	if disciple.Memories == nil {
		disciple.Memories = []DiscipleMemoryEntry{}
	}
	if disciple.Relationship == nil {
		disciple.Relationship = []string{}
	}
	if disciple.Emotion == nil {
		disciple.Emotion = []string{}
	}
	if disciple.RecentSummary == nil {
		disciple.RecentSummary = []string{}
	}
	if len(disciple.Memories) > maxDiscipleMemoryEntries {
		disciple.Memories = cloneDiscipleMemories(disciple.Memories[len(disciple.Memories)-maxDiscipleMemoryEntries:])
	}
	for index := range disciple.Memories {
		if disciple.Memories[index].Intensity <= 0 {
			disciple.Memories[index].Intensity = 1
		}
		disciple.Memories[index].Tags = sortedUniqueStrings(disciple.Memories[index].Tags)
	}
	return rebuildDiscipleMemorySurface(disciple)
}

func rebuildDiscipleMemorySurface(disciple DiscipleState) DiscipleState {
	disciple.Relationship = []string{}
	disciple.Emotion = []string{}
	disciple.RecentSummary = []string{}

	gratefulScore := 0
	resentfulScore := 0
	dutyScore := 0
	meritScore := 0
	advancementScore := 0
	woundScore := 0
	stressScore := 0
	crisisScore := 0

	for _, memory := range disciple.Memories {
		tagSet := makeStringSet(memory.Tags)
		if tagSet["payroll_paid"] || tagSet["thick_support"] || tagSet["promotion"] || tagSet["battle_merit"] || tagSet["duty_fulfilled"] {
			gratefulScore += memory.Intensity
		}
		if tagSet["payroll_delayed"] || tagSet["duty_breached"] || tagSet["assessment_failed"] || tagSet["ignored_omen"] || tagSet["task_failed"] {
			resentfulScore += memory.Intensity
		}
		if tagSet["duty_fulfilled"] {
			dutyScore += memory.Intensity
		}
		if tagSet["battle_merit"] {
			meritScore += memory.Intensity
		}
		if tagSet["advancement"] || tagSet["promotion_path"] || tagSet["cultivation_peak"] {
			advancementScore += memory.Intensity
		}
		if tagSet["injury"] || tagSet["battle_wound"] || tagSet["wound_shadow"] {
			woundScore += memory.Intensity
		}
		if tagSet["pressure"] || tagSet["setback"] || tagSet["risk_response"] {
			stressScore += memory.Intensity
		}
		if tagSet["crisis"] || tagSet["omen"] {
			crisisScore += memory.Intensity
		}
	}

	if gratefulScore > resentfulScore {
		disciple.Relationship = append(disciple.Relationship, "sect_grateful")
	}
	if resentfulScore >= gratefulScore+2 {
		disciple.Relationship = append(disciple.Relationship, "sect_resentful")
	}
	if dutyScore > 0 {
		disciple.Relationship = append(disciple.Relationship, "duty_trusted")
	}
	if meritScore > 0 {
		disciple.Relationship = append(disciple.Relationship, "battle_proven")
	}
	if advancementScore > 0 {
		disciple.Relationship = append(disciple.Relationship, "promotion_driven")
	}
	if woundScore > 0 {
		disciple.Relationship = append(disciple.Relationship, "wound_shadow")
	}
	if crisisScore > 0 {
		disciple.Relationship = append(disciple.Relationship, "risk_tempered")
	}

	if disciple.InjuryLevel > 0 || woundScore > 0 {
		disciple.Emotion = append(disciple.Emotion, "wounded")
	}
	if disciple.Pressure >= 45 || stressScore > 2 {
		disciple.Emotion = append(disciple.Emotion, "pressured")
	}
	if gratefulScore > resentfulScore {
		disciple.Emotion = append(disciple.Emotion, "grateful")
	}
	if resentfulScore > gratefulScore {
		disciple.Emotion = append(disciple.Emotion, "resentful")
	}
	if advancementScore > 0 && disciple.Pressure < 70 {
		disciple.Emotion = append(disciple.Emotion, "ambitious")
	}
	if meritScore > 0 && disciple.InjuryLevel == 0 {
		disciple.Emotion = append(disciple.Emotion, "confident")
	}
	if crisisScore > 0 {
		disciple.Emotion = append(disciple.Emotion, "alert")
	}

	for index := len(disciple.Memories) - 1; index >= 0 && len(disciple.RecentSummary) < recentDiscipleExperienceEntries; index-- {
		summary := strings.TrimSpace(disciple.Memories[index].Summary)
		if summary == "" {
			continue
		}
		disciple.RecentSummary = append(disciple.RecentSummary, summary)
	}
	disciple.Relationship = sortedUniqueStrings(disciple.Relationship)
	disciple.Emotion = sortedUniqueStrings(disciple.Emotion)
	return disciple
}

func applyTaskMemoryFromEvent(state *SectState, eventType DomainEventType, task TaskState, version Version) {
	if eventType != DomainEventTypeTaskCompleted && eventType != DomainEventTypeTaskFailed {
		return
	}
	for _, discipleID := range sortedDiscipleIDs(task.AssignedDiscipleIDs) {
		storeDiscipleMemory(state, discipleID, taskMemoryEntry(task, eventType, state.Time.CalendarDay, version))
	}
}

func applyDiscipleMemoryFromEvent(state *SectState, eventType DomainEventType, previous DiscipleState, next DiscipleState, version Version) {
	switch eventType {
	case DomainEventTypeCultivationAdvanced:
		if !previous.Realm.ReadyForBreakthrough && next.Realm.ReadyForBreakthrough {
			storeDiscipleMemory(state, next.DiscipleID, DiscipleMemoryEntry{
				Kind:              "cultivation_peak",
				Summary:           fmt.Sprintf("%s 修炼圆满，已可尝试突破。", next.Name),
				SourceEventType:   eventType,
				RecordedAtVersion: version,
				RecordedAtDay:     state.Time.CalendarDay,
				Intensity:         2,
				Tags:              []string{"advancement", "cultivation_peak", "promotion_path"},
			})
		}
	case DomainEventTypeBreakthroughSucceeded:
		storeDiscipleMemory(state, next.DiscipleID, DiscipleMemoryEntry{
			Kind:              "breakthrough_success",
			Summary:           fmt.Sprintf("%s 成功突破至 %s。", next.Name, formatRealmStageLabel(next.Realm.Stage)),
			SourceEventType:   eventType,
			RecordedAtVersion: version,
			RecordedAtDay:     state.Time.CalendarDay,
			Intensity:         4,
			Tags:              []string{"advancement", "breakthrough_success", "confidence"},
		})
	case DomainEventTypeBreakthroughFailed:
		storeDiscipleMemory(state, next.DiscipleID, DiscipleMemoryEntry{
			Kind:              "breakthrough_failed",
			Summary:           fmt.Sprintf("%s 的突破受挫，需重新稳固根基。", next.Name),
			SourceEventType:   eventType,
			RecordedAtVersion: version,
			RecordedAtDay:     state.Time.CalendarDay,
			Intensity:         3,
			Tags:              []string{"advancement", "breakthrough_failed", "injury", "pressure", "setback", "wound_shadow"},
		})
	case DomainEventTypeDisciplePromoted:
		storeDiscipleMemory(state, next.DiscipleID, DiscipleMemoryEntry{
			Kind:              "promotion",
			Summary:           fmt.Sprintf("%s 晋升为 %s。", next.Name, next.Identity),
			SourceEventType:   eventType,
			RecordedAtVersion: version,
			RecordedAtDay:     state.Time.CalendarDay,
			Intensity:         3,
			Tags:              []string{"advancement", "promotion", "promotion_path", "thick_support"},
		})
	case DomainEventTypeDiscipleSatisfactionChanged, DomainEventTypeDiscipleLoyaltyChanged:
		if next.InjuryLevel > previous.InjuryLevel || next.HP < previous.HP {
			tags := []string{"injury", "pressure", "wound_shadow"}
			if previous.AssignmentTask != nil || next.WorkTarget.TaskID != nil {
				tags = append(tags, "battle_wound", "risk_response")
			}
			storeDiscipleMemory(state, next.DiscipleID, DiscipleMemoryEntry{
				Kind:              "injury",
				Summary:           fmt.Sprintf("%s 在事务中受伤，伤势升至 %d 级。", next.Name, next.InjuryLevel),
				SourceEventType:   eventType,
				RecordedAtVersion: version,
				RecordedAtDay:     state.Time.CalendarDay,
				Intensity:         clampInt(maxInt(next.InjuryLevel-previous.InjuryLevel, 1)+1, 1, 4),
				Tags:              tags,
			})
			return
		}
		if next.Pressure >= previous.Pressure+20 {
			storeDiscipleMemory(state, next.DiscipleID, DiscipleMemoryEntry{
				Kind:              "pressure_spike",
				Summary:           fmt.Sprintf("%s 承受的事务压力明显上升。", next.Name),
				SourceEventType:   eventType,
				RecordedAtVersion: version,
				RecordedAtDay:     state.Time.CalendarDay,
				Intensity:         2,
				Tags:              []string{"pressure", "risk_response", "setback"},
			})
		}
	}
}

func applyAssessmentMemoryFromEvent(state *SectState, previous DiscipleState, assessment DiscipleAssessmentState, next DiscipleState, version Version) {
	_ = previous
	kind := "assessment_failed"
	summary := fmt.Sprintf("%s 的晋升考核未通过，原因：%s。", next.Name, assessment.Reason)
	tags := []string{"assessment_failed", "promotion_path", "setback"}
	intensity := 2
	if assessment.Passed {
		kind = "assessment_passed"
		summary = fmt.Sprintf("%s 通过了 %s 晋升考核。", next.Name, assessment.TargetRank)
		tags = []string{"advancement", "assessment_passed", "promotion_path"}
		intensity = 3
	}
	storeDiscipleMemory(state, next.DiscipleID, DiscipleMemoryEntry{
		Kind:              kind,
		Summary:           summary,
		SourceEventType:   DomainEventTypeAssessmentResolved,
		RecordedAtVersion: version,
		RecordedAtDay:     state.Time.CalendarDay,
		Intensity:         intensity,
		Tags:              tags,
	})
}

func applyPayrollMemoryFromEvent(state *SectState, eventType DomainEventType, payload PayrollSettlementPayload, version Version) {
	if _, ok := state.Disciples[payload.DiscipleID]; !ok {
		return
	}
	kind := "payroll_paid"
	summary := fmt.Sprintf("月例如期发放，账目结清至第 %d 月。", payload.MonthIndex)
	tags := []string{"monthly", "payroll_paid"}
	intensity := 2
	if state.Policies.ResourcePolicy == ResourcePolicyGenerous {
		summary = fmt.Sprintf("宗门按厚养政策发放第 %d 月月例。", payload.MonthIndex)
		tags = append(tags, "thick_support")
		intensity = 3
	}
	if eventType == DomainEventTypePayrollDelayed {
		kind = "payroll_delayed"
		summary = fmt.Sprintf("第 %d 月月例拖欠，累计欠发 %d 期。", payload.MonthIndex, payload.ArrearsAfter)
		tags = []string{"monthly", "payroll_delayed", "support_withheld", "setback"}
		intensity = clampInt(2+payload.ArrearsAfter, 2, 4)
	}
	storeDiscipleMemory(state, payload.DiscipleID, DiscipleMemoryEntry{
		Kind:              kind,
		Summary:           summary,
		SourceEventType:   eventType,
		RecordedAtVersion: version,
		RecordedAtDay:     state.Time.CalendarDay,
		Intensity:         intensity,
		Tags:              tags,
	})
}

func applyMonthlyObligationMemoryFromEvent(state *SectState, payload MonthlyObligationCheckedPayload, version Version) {
	if _, ok := state.Disciples[payload.DiscipleID]; !ok {
		return
	}
	kind := "duty_fulfilled"
	summary := fmt.Sprintf("本月宗务义务已完成 %d/%d 天。", payload.CompletedDays, payload.RequiredDays)
	tags := []string{"monthly", "duty_fulfilled"}
	intensity := 1
	if payload.ViolationAdded {
		kind = "duty_breached"
		summary = fmt.Sprintf("本月宗务义务未达标，仅完成 %d/%d 天。", payload.CompletedDays, payload.RequiredDays)
		tags = []string{"monthly", "duty_breached", "pressure", "setback"}
		intensity = 2
	}
	if payload.RequiredDays <= 0 {
		return
	}
	storeDiscipleMemory(state, payload.DiscipleID, DiscipleMemoryEntry{
		Kind:              kind,
		Summary:           summary,
		SourceEventType:   DomainEventTypeMonthlyObligationChecked,
		RecordedAtVersion: version,
		RecordedAtDay:     state.Time.CalendarDay,
		Intensity:         intensity,
		Tags:              tags,
	})
}

func applySectEventForeshadowMemoryFromEvent(state *SectState, event SectEvent, eventType DomainEventType, version Version) {
	if event.SourceDiscipleID == nil {
		return
	}
	tags := []string{"event", "omen"}
	if event.Kind == "task_crisis_clue" || event.Severity >= 2 {
		tags = append(tags, "crisis", "risk_response")
	}
	storeDiscipleMemory(state, *event.SourceDiscipleID, DiscipleMemoryEntry{
		Kind:              "event_omen",
		Summary:           fmt.Sprintf("%s 察觉到《%s》的征兆。", state.Disciples[*event.SourceDiscipleID].Name, event.Title),
		SourceEventType:   eventType,
		RecordedAtVersion: version,
		RecordedAtDay:     state.Time.CalendarDay,
		Intensity:         clampInt(maxInt(event.Severity, 1), 1, 4),
		Tags:              tags,
	})
}

func applySectEventResolutionMemoryFromEvent(state *SectState, sourceEvent SectEvent, summary ResolvedEventSummary, eventType DomainEventType) {
	if sourceEvent.SourceDiscipleID == nil {
		return
	}
	tags := []string{"event_resolution", "risk_response"}
	kind := "event_resolved"
	if eventType == DomainEventTypeSectEventExpired {
		kind = "event_expired"
		tags = append(tags, "ignored_omen", "setback")
	}
	storeDiscipleMemory(state, *sourceEvent.SourceDiscipleID, DiscipleMemoryEntry{
		Kind:              kind,
		Summary:           summary.Summary,
		SourceEventType:   eventType,
		RecordedAtVersion: summary.ResolvedAtVersion,
		RecordedAtDay:     state.Time.CalendarDay,
		Intensity:         clampInt(maxInt(sourceEvent.Severity, 1), 1, 4),
		Tags:              tags,
	})
}

func applySectGoalMemoryFromEvent(state *SectState, goal SectGoal, summary ResolvedSectGoalSummary, eventType DomainEventType) {
	if goal.FocusDiscipleID == nil {
		return
	}
	disciple, ok := state.Disciples[*goal.FocusDiscipleID]
	if !ok {
		return
	}
	kind := "goal_completed"
	tags := []string{"goal", goal.Kind}
	intensity := 2
	if goal.Status == SectGoalStatusCompleted {
		tags = append(tags, "sect_merit", "thick_support")
		switch goal.Kind {
		case "inner_disciple":
			tags = append(tags, "advancement", "promotion", "promotion_path")
			intensity = 3
		case "external_affairs":
			tags = append(tags, "battle_merit", "duty_fulfilled", "risk_response")
			intensity = 3
		case "cave_routine":
			tags = append(tags, "cultivation_peak")
		case "stable_monthly":
			tags = append(tags, "monthly", "payroll_paid", "duty_fulfilled")
		}
	} else {
		kind = "goal_failed"
		tags = append(tags, "setback", "pressure")
		switch goal.Kind {
		case "stable_monthly":
			tags = append(tags, "monthly", "payroll_delayed", "duty_breached", "support_withheld")
			intensity = 3
		}
	}
	storeDiscipleMemory(state, disciple.DiscipleID, DiscipleMemoryEntry{
		Kind:              kind,
		Summary:           summary.Summary,
		SourceEventType:   eventType,
		RecordedAtVersion: summary.ResolvedAtVersion,
		RecordedAtDay:     state.Time.CalendarDay,
		Intensity:         intensity,
		Tags:              tags,
	})
}

func applyMonthlyAssessmentMemoryFromEvent(state *SectState, result MonthlyAssessmentResult, eventType DomainEventType) {
	if result.ChampionDiscipleID == nil {
		return
	}
	disciple, ok := state.Disciples[*result.ChampionDiscipleID]
	if !ok {
		return
	}
	summary := strings.TrimSpace(result.Summary)
	if summary == "" {
		summary = fmt.Sprintf("第 %d 月小比由 %s 领先，获得宗门记功。", result.MonthIndex, disciple.Name)
	}
	storeDiscipleMemory(state, disciple.DiscipleID, DiscipleMemoryEntry{
		Kind:              "monthly_assessment_champion",
		Summary:           summary,
		SourceEventType:   eventType,
		RecordedAtVersion: result.ResolvedAtVersion,
		RecordedAtDay:     state.Time.CalendarDay,
		Intensity:         clampInt(maxInt(result.PromotionMomentum, 2), 2, 4),
		Tags: []string{
			"monthly",
			"sect_merit",
			"duty_fulfilled",
			"advancement",
			"promotion_path",
			"thick_support",
		},
	})
}

func storeDiscipleMemory(state *SectState, discipleID DiscipleID, entry DiscipleMemoryEntry) {
	if state == nil || discipleID == "" {
		return
	}
	disciple, ok := state.Disciples[discipleID]
	if !ok {
		return
	}
	summary := strings.TrimSpace(entry.Summary)
	if summary == "" {
		return
	}
	disciple = normalizeDiscipleState(disciple)
	entry.Summary = summary
	entry.Intensity = clampInt(maxInt(entry.Intensity, 1), 1, 4)
	entry.Tags = sortedUniqueStrings(entry.Tags)
	disciple.Memories = append(disciple.Memories, entry)
	if len(disciple.Memories) > maxDiscipleMemoryEntries {
		disciple.Memories = cloneDiscipleMemories(disciple.Memories[len(disciple.Memories)-maxDiscipleMemoryEntries:])
	}
	state.Disciples[discipleID] = rebuildDiscipleMemorySurface(disciple)
}

func taskMemoryEntry(task TaskState, eventType DomainEventType, recordedAtDay int32, version Version) DiscipleMemoryEntry {
	taskType := normalizeTaskType(task.Kind, task.Type)
	kind := "task_completed"
	summary := fmt.Sprintf("完成%s《%s》，评价 %s。", taskTypeLabel(taskType), task.Title, task.Evaluation)
	tags := []string{"task_result", "task_success", memoryTaskTypeTag(taskType)}
	intensity := 2
	if taskType == TaskTypeCombat || taskType == TaskTypeExternal {
		tags = append(tags, "battle_merit", "risk_response")
	}
	if task.Evaluation == TaskEvaluationExcellent {
		tags = append(tags, "sect_merit")
		intensity = 3
	}
	if task.ContributionReward > 0 {
		tags = append(tags, "duty_fulfilled")
	}
	if eventType == DomainEventTypeTaskFailed {
		kind = "task_failed"
		summary = fmt.Sprintf("%s《%s》失利，评价 %s。", taskTypeLabel(taskType), task.Title, task.Evaluation)
		tags = []string{"task_failed", "task_result", memoryTaskTypeTag(taskType), "setback"}
		if taskType == TaskTypeCombat || taskType == TaskTypeExternal {
			tags = append(tags, "battle_wound", "risk_response")
		}
		intensity = clampInt(2+task.Risk/40, 2, 4)
	}
	return DiscipleMemoryEntry{
		Kind:              kind,
		Summary:           summary,
		SourceEventType:   eventType,
		RecordedAtVersion: version,
		RecordedAtDay:     recordedAtDay,
		Intensity:         intensity,
		Tags:              tags,
	}
}

func taskTypeLabel(taskType TaskType) string {
	switch taskType {
	case TaskTypeCombat:
		return "战斗"
	case TaskTypeExplore:
		return "探索"
	case TaskTypeExternal:
		return "外务"
	case TaskTypeProduction:
		return "生产"
	default:
		return "任务"
	}
}

func memoryTaskTypeTag(taskType TaskType) string {
	switch taskType {
	case TaskTypeCombat:
		return "task_combat"
	case TaskTypeExplore:
		return "task_explore"
	case TaskTypeExternal:
		return "task_external"
	case TaskTypeProduction:
		return "task_production"
	default:
		return "task_general"
	}
}

func sortedUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func makeStringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func eventToClientDelta(event DomainEvent) (ClientEvent, []PatchOp, error) {
	switch event.Type {
	case DomainEventTypeResourceChanged:
		var payload ResourceChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		ops := make([]PatchOp, 0, len(payload.Changes))
		for kind, delta := range payload.Changes {
			ops = append(ops, PatchOp{
				Op:            PatchOpTypeIncrement,
				Path:          fmt.Sprintf("/resources/stock/%s", kind),
				Value:         encodeSignedVarint(delta),
				ValueEncoding: PatchValueEncodingVarint,
			})
		}
		return ClientEvent{
			SceneVersion: event.Version,
			Type:         ClientEventTypeResourceChanged,
			Payload:      event.Payload,
		}, ops, nil
	case DomainEventTypeSectMetaChanged:
		var payload SectMetaChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return sectMetaChangedDelta(event.Version, event.Payload, payload.Meta)
	case DomainEventTypeTimeAdvanced:
		var payload TimeAdvancedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return timeChangedDelta(event.Version, event.Payload, payload.Time)
	case DomainEventTypeBuildingBuilt:
		var payload BuildingBuiltPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return buildingChangedDelta(event.Version, event.Payload, payload.Building)
	case DomainEventTypeBuildingUpgraded:
		var payload BuildingUpgradedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return buildingChangedDelta(event.Version, event.Payload, payload.Building)
	case DomainEventTypeBuildingMaintained, DomainEventTypeBuildingDamaged:
		var payload BuildingMaintainedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return buildingChangedDelta(event.Version, event.Payload, payload.Building)
	case DomainEventTypeTaskPublished, DomainEventTypeTaskAccepted, DomainEventTypeTaskCancelled, DomainEventTypeTaskPriorityChanged, DomainEventTypeTaskProgressed, DomainEventTypeTaskCompleted, DomainEventTypeTaskFailed:
		var payload TaskChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return taskChangedDelta(event.Version, event.Payload, payload.Task)
	case DomainEventTypeDiscipleAssignmentChanged, DomainEventTypeCultivationAdvanced, DomainEventTypeBreakthroughSucceeded, DomainEventTypeBreakthroughFailed, DomainEventTypeDiscipleSatisfactionChanged, DomainEventTypeDiscipleLoyaltyChanged, DomainEventTypeDisciplePromoted:
		var payload DiscipleChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return discipleChangedDelta(event.Version, event.Payload, payload.Disciple)
	case DomainEventTypeAssessmentResolved:
		var payload AssessmentResolvedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return discipleChangedDelta(event.Version, event.Payload, payload.Disciple)
	case DomainEventTypeContributionEarned, DomainEventTypeContributionSpent:
		var payload ContributionChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return contributionChangedDelta(event.Version, event.Payload, payload.DiscipleID)
	case DomainEventTypeInventoryChanged:
		var payload InventoryChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return inventoryChangedDelta(event.Version, event.Payload, payload.Item)
	case DomainEventTypeArtifactChanged:
		var payload ArtifactChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return artifactChangedDelta(event.Version, event.Payload, payload.Artifact, payload.Disciple)
	case DomainEventTypeFormationChanged:
		var payload FormationChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return formationChangedDelta(event.Version, event.Payload, payload)
	case DomainEventTypeProductionChanged:
		var payload ProductionChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return productionChangedDelta(event.Version, event.Payload, payload.Production)
	case DomainEventTypeSectGoalChanged, DomainEventTypeSectGoalResolved:
		return ClientEvent{
			SceneVersion: event.Version,
			Type:         ClientEventTypeSectGoalChanged,
			Payload:      event.Payload,
		}, nil, nil
	case DomainEventTypeSectEventSeeded, DomainEventTypeSectEventForeshadowed, DomainEventTypeSectEventResolved, DomainEventTypeSectEventExpired:
		return ClientEvent{
			SceneVersion: event.Version,
			Type:         ClientEventTypeSectEventChanged,
			Payload:      event.Payload,
		}, nil, nil
	case DomainEventTypeRecruitmentStarted:
		var payload RecruitmentStartedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return admissionChangedDelta(event.Version, event.Payload, payload.Admissions, nil, nil)
	case DomainEventTypeCandidateAccepted:
		var payload CandidateAcceptedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		account := &ContributionAccount{DiscipleID: payload.Disciple.DiscipleID}
		return admissionChangedDelta(event.Version, event.Payload, payload.Admissions, &payload.Disciple, account)
	case DomainEventTypeCandidateRejected:
		var payload CandidateRejectedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return admissionChangedDelta(event.Version, event.Payload, payload.Admissions, nil, nil)
	case DomainEventTypePolicyChanged:
		var payload PolicyChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return policyChangedDelta(event.Version, event.Payload, payload.Policies)
	case DomainEventTypeInstitutionChanged:
		var payload InstitutionChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return institutionChangedDelta(event.Version, event.Payload, payload)
	case DomainEventTypePayrollPaid, DomainEventTypePayrollDelayed, DomainEventTypeMonthlyObligationChecked:
		return ClientEvent{
			SceneVersion: event.Version,
			Type:         ClientEventTypeMonthlyChanged,
			Payload:      event.Payload,
		}, nil, nil
	case DomainEventTypeMonthAdvanced:
		var payload MonthAdvancedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return monthlyChangedDelta(event.Version, event.Payload, payload.Monthly)
	case DomainEventTypeOrderChanged:
		var payload OrderChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return orderChangedDelta(event.Version, event.Payload, payload.Order)
	case DomainEventTypeMonthlyAssessmentResolved:
		var payload MonthlyAssessmentResolvedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return ClientEvent{}, nil, fmt.Errorf("decode %s payload: %w", event.Type, err)
		}
		return monthlyAssessmentChangedDelta(event.Version, event.Payload, payload.Assessment)
	default:
		return ClientEvent{}, nil, fmt.Errorf("unsupported client delta event: %s", event.Type)
	}
}

const mvpDiscipleRosterCapacity = 6

var (
	errInsufficientResources  = errors.New("insufficient resources")
	errDiscipleNotFound       = errors.New("disciple not found")
	errDiscipleBusy           = errors.New("disciple busy")
	errTaskNotFound           = errors.New("task not found")
	errTaskRequirementNotMet  = errors.New("task requirement not met")
	errContributionNotEnough  = errors.New("contribution not enough")
	errMonthlyLimitReached    = errors.New("monthly limit reached")
	errCultivationNotReady    = errors.New("cultivation not ready")
	errRiskTooHigh            = errors.New("risk too high")
	errCandidateNotFound      = errors.New("candidate not found")
	errEventNotFound          = errors.New("sect event not found")
	errEventExpired           = errors.New("sect event expired")
	errEventRequirementNotMet = errors.New("sect event requirement not met")
)

type artifactDefinition struct {
	ArtifactType        ArtifactType
	Slot                ArtifactSlot
	BaseCost            ResourceCost
	BaseStats           map[string]int
	RequiredInstitution InstitutionID
}

var artifactDefinitions = map[ArtifactType]artifactDefinition{
	ArtifactTypeSword: {
		ArtifactType:        ArtifactTypeSword,
		Slot:                ArtifactSlotWeapon,
		BaseCost:            ResourceCost{ResourceKindOre: 10, ResourceKindBeastMat: 2, ResourceKindSpiritStone: 8},
		BaseStats:           map[string]int{"combat": 10, "injury_mitigation": 1},
		RequiredInstitution: InstitutionIDMainHall,
	},
	ArtifactTypeRobe: {
		ArtifactType:        ArtifactTypeRobe,
		Slot:                ArtifactSlotRobe,
		BaseCost:            ResourceCost{ResourceKindOre: 6, ResourceKindBeastMat: 4, ResourceKindSpiritStone: 5},
		BaseStats:           map[string]int{"combat": 3, "cultivation": 3, "injury_mitigation": 2},
		RequiredInstitution: InstitutionIDMainHall,
	},
	ArtifactTypeFarmTool: {
		ArtifactType:        ArtifactTypeFarmTool,
		Slot:                ArtifactSlotTool,
		BaseCost:            ResourceCost{ResourceKindOre: 8, ResourceKindSpiritStone: 3},
		BaseStats:           map[string]int{"production": 8},
		RequiredInstitution: InstitutionIDMainHall,
	},
	ArtifactTypeAlchemyFurnace: {
		ArtifactType:        ArtifactTypeAlchemyFurnace,
		Slot:                ArtifactSlotTool,
		BaseCost:            ResourceCost{ResourceKindOre: 8, ResourceKindFormationMat: 2, ResourceKindSpiritStone: 6},
		BaseStats:           map[string]int{"production": 6, "cultivation": 2},
		RequiredInstitution: InstitutionIDMedicineHut,
	},
	ArtifactTypeFormationDisk: {
		ArtifactType:        ArtifactTypeFormationDisk,
		Slot:                ArtifactSlotSpecial,
		BaseCost:            ResourceCost{ResourceKindOre: 10, ResourceKindFormationMat: 5, ResourceKindSpiritStone: 10},
		BaseStats:           map[string]int{"combat": 4, "exploration": 6, "cultivation": 4},
		RequiredInstitution: InstitutionIDMainHall,
	},
}

func ensureInventoryState(state *SectState) {
	if state.Inventory.Items == nil {
		state.Inventory.Items = map[ItemID]InventoryEntry{}
	}
	if state.Inventory.Artifacts == nil {
		state.Inventory.Artifacts = map[ItemID]ArtifactState{}
	}
}

func normalizeCraftArtifactType(payload CraftArtifactPayload) ArtifactType {
	if payload.ArtifactType != "" {
		return payload.ArtifactType
	}
	return payload.Type
}

func clampArtifactQuality(quality int) int {
	if quality <= 0 {
		return 1
	}
	return clampInt(quality, 1, 2)
}

func artifactDefinitionFor(artifactType ArtifactType) (artifactDefinition, error) {
	definition, ok := artifactDefinitions[artifactType]
	if !ok {
		return artifactDefinition{}, fmt.Errorf("%w: unsupported artifact type %s", errTaskRequirementNotMet, artifactType)
	}
	return definition, nil
}

func artifactCraftCost(artifactType ArtifactType, quality int) (ResourceCost, error) {
	definition, err := artifactDefinitionFor(artifactType)
	if err != nil {
		return nil, err
	}
	cost := ResourceCost{}
	for kind, amount := range definition.BaseCost {
		cost[kind] = amount * int64(clampArtifactQuality(quality))
	}
	return cost, nil
}

func newArtifactState(itemID ItemID, artifactType ArtifactType, quality int, sourceTag string) ArtifactState {
	definition, _ := artifactDefinitionFor(artifactType)
	quality = clampArtifactQuality(quality)
	stats := map[string]int{}
	for key, value := range definition.BaseStats {
		stats[key] = value * quality
	}
	return ArtifactState{
		ItemID:        itemID,
		Type:          artifactType,
		Quality:       quality,
		Durability:    100,
		MaxDurability: 100,
		Stats:         stats,
		SourceTag:     sourceTag,
	}
}

func nextArtifactID(state SectState) ItemID {
	ensureInventoryState(&state)
	for index := len(state.Inventory.Artifacts) + 1; ; index++ {
		itemID := ItemID(fmt.Sprintf("artifact-%d", index))
		if _, exists := state.Inventory.Artifacts[itemID]; !exists {
			return itemID
		}
	}
}

func validateArtifactCraftBuilding(state SectState, artifactType ArtifactType) error {
	definition, err := artifactDefinitionFor(artifactType)
	if err != nil {
		return err
	}
	if institution, ok := state.Institutions.ByID[definition.RequiredInstitution]; !ok || !institution.Enabled {
		return fmt.Errorf("%w: crafting %s requires %s", errTaskRequirementNotMet, artifactType, definition.RequiredInstitution)
	}
	return nil
}

func validateArtifactRepairBuilding(state SectState, artifactType ArtifactType) error {
	return validateArtifactCraftBuilding(state, artifactType)
}

func artifactRepairCost(artifact ArtifactState) ResourceCost {
	missing := maxInt(0, artifact.MaxDurability-artifact.Durability)
	spiritStone := int64(maxInt(1, (missing+24)/25))
	cost := ResourceCost{ResourceKindSpiritStone: spiritStone}
	switch artifact.Type {
	case ArtifactTypeSword, ArtifactTypeFarmTool, ArtifactTypeAlchemyFurnace, ArtifactTypeFormationDisk:
		cost[ResourceKindOre] = 1
	case ArtifactTypeRobe:
		cost[ResourceKindBeastMat] = 1
	}
	return cost
}

func artifactSlotForType(artifactType ArtifactType) ArtifactSlot {
	definition, err := artifactDefinitionFor(artifactType)
	if err != nil {
		return ""
	}
	return definition.Slot
}

func equipmentItemForSlot(equipment EquipmentState, slot ArtifactSlot) ItemID {
	switch slot {
	case ArtifactSlotWeapon:
		return equipment.Weapon
	case ArtifactSlotRobe:
		return equipment.Robe
	case ArtifactSlotTool:
		return equipment.Tool
	case ArtifactSlotSpecial:
		return equipment.Special
	default:
		return ""
	}
}

func setEquipmentItemForSlot(equipment EquipmentState, slot ArtifactSlot, itemID ItemID) EquipmentState {
	switch slot {
	case ArtifactSlotWeapon:
		equipment.Weapon = itemID
	case ArtifactSlotRobe:
		equipment.Robe = itemID
	case ArtifactSlotTool:
		equipment.Tool = itemID
	case ArtifactSlotSpecial:
		equipment.Special = itemID
	}
	return equipment
}

func clearEquipmentItem(equipment EquipmentState, itemID ItemID) EquipmentState {
	if equipment.Weapon == itemID {
		equipment.Weapon = ""
	}
	if equipment.Robe == itemID {
		equipment.Robe = ""
	}
	if equipment.Tool == itemID {
		equipment.Tool = ""
	}
	if equipment.Special == itemID {
		equipment.Special = ""
	}
	return equipment
}

type formationDefinition struct {
	Kind              FormationKind
	DisplayName       string
	MinBuildingLevel  int32
	MaintenanceCost   ResourceCost
	CultivationBonus  int
	PressureRelief    int
	ProductionBonus   int64
	DefenseMitigation int
}

var formationDefinitions = map[FormationKind]formationDefinition{
	FormationKindGatherSpirit: {
		Kind:             FormationKindGatherSpirit,
		DisplayName:      "聚灵阵",
		MinBuildingLevel: 1,
		MaintenanceCost:  ResourceCost{ResourceKindSpiritStone: 1},
		CultivationBonus: 3,
	},
	FormationKindFieldGuard: {
		Kind:             FormationKindFieldGuard,
		DisplayName:      "护田阵",
		MinBuildingLevel: 2,
		MaintenanceCost:  ResourceCost{ResourceKindSpiritStone: 1, ResourceKindFormationMat: 1},
		ProductionBonus:  1,
	},
	FormationKindDefense: {
		Kind:              FormationKindDefense,
		DisplayName:       "守御阵",
		MinBuildingLevel:  2,
		MaintenanceCost:   ResourceCost{ResourceKindSpiritStone: 2},
		DefenseMitigation: 10,
	},
	FormationKindCalmMind: {
		Kind:             FormationKindCalmMind,
		DisplayName:      "静心阵",
		MinBuildingLevel: 1,
		MaintenanceCost:  ResourceCost{ResourceKindSpiritStone: 1, ResourceKindHerb: 1},
		CultivationBonus: 1,
		PressureRelief:   4,
	},
}

func ensureFormationState(state *SectState) {
	if state == nil {
		return
	}
	if state.Formations == nil {
		state.Formations = map[BuildingID]FormationState{}
	}
}

func formationDefinitionFor(kind FormationKind) (formationDefinition, error) {
	definition, ok := formationDefinitions[kind]
	if !ok {
		return formationDefinition{}, fmt.Errorf("%w: unsupported formation kind %s", errTaskRequirementNotMet, kind)
	}
	return definition, nil
}

func validateFormationAttachment(state SectState, building BuildingState, kind FormationKind) error {
	definition, err := formationDefinitionFor(kind)
	if err != nil {
		return err
	}
	if building.Level < definition.MinBuildingLevel {
		return fmt.Errorf("%w: %s requires building level %d", errTaskRequirementNotMet, definition.DisplayName, definition.MinBuildingLevel)
	}
	if building.Phase != "active" {
		return fmt.Errorf("%w: building %s is not active", errTaskRequirementNotMet, building.BuildingID)
	}
	return nil
}

func formationMaintenanceCostFor(formation FormationState) ResourceCost {
	definition, err := formationDefinitionFor(formation.Kind)
	if err != nil {
		return nil
	}
	return scaleResourceCost(definition.MaintenanceCost, int64(maxInt(1, formation.Level)))
}

func formationManualMaintenanceCost(formation FormationState) ResourceCost {
	multiplier := int64(maxInt(1, formation.Level+formation.MaintenanceDebt))
	return scaleResourceCost(formationMaintenanceCostFor(formation), multiplier)
}

func scaleResourceCost(cost ResourceCost, multiplier int64) ResourceCost {
	if len(cost) == 0 || multiplier <= 0 {
		return nil
	}
	scaled := ResourceCost{}
	for kind, amount := range cost {
		if amount <= 0 {
			continue
		}
		scaled[kind] = amount * multiplier
	}
	return scaled
}

func newFormationState(buildingID BuildingID, artifact ArtifactState, kind FormationKind, attachedAtVersion Version) FormationState {
	definition, _ := formationDefinitionFor(kind)
	formation := FormationState{
		FormationID:       fmt.Sprintf("formation-%s", buildingID),
		Kind:              kind,
		BuildingID:        buildingID,
		ArtifactItemID:    artifact.ItemID,
		Level:             maxInt(1, artifact.Quality),
		Stability:         100,
		MaintenanceDebt:   0,
		Active:            true,
		AttachedAtVersion: attachedAtVersion,
		MaintenanceCost:   formationMaintenanceCostFor(FormationState{Kind: kind, Level: maxInt(1, artifact.Quality)}),
		EffectSummary: []string{
			fmt.Sprintf("%s已挂接到 %s。", definition.DisplayName, buildingID),
		},
	}
	formation.EffectSummary = formationEffectSummary(formation)
	return formation
}

func formationAfterDailyMaintenancePaid(formation FormationState) FormationState {
	next := normalizeFormationState(formation)
	next.Stability = minInt(100, next.Stability+4)
	next.MaintenanceDebt = 0
	next.Active = true
	next.EffectSummary = formationEffectSummary(next)
	return next
}

func formationAfterDailyMaintenanceShortage(formation FormationState) FormationState {
	next := normalizeFormationState(formation)
	next.MaintenanceDebt++
	next.Stability = maxInt(0, next.Stability-20)
	next.Active = next.Stability > 0
	next.EffectSummary = formationEffectSummary(next)
	return next
}

func formationAfterManualMaintenance(formation FormationState) FormationState {
	next := normalizeFormationState(formation)
	next.Stability = 100
	next.MaintenanceDebt = 0
	next.Active = true
	next.EffectSummary = formationEffectSummary(next)
	return next
}

func normalizeFormationState(formation FormationState) FormationState {
	next := formation
	if next.FormationID == "" {
		next.FormationID = fmt.Sprintf("formation-%s", next.BuildingID)
	}
	if next.Level <= 0 {
		next.Level = 1
	}
	next.Stability = clampInt(next.Stability, 0, 100)
	if next.MaintenanceDebt < 0 {
		next.MaintenanceDebt = 0
	}
	if len(next.MaintenanceCost) == 0 {
		next.MaintenanceCost = formationMaintenanceCostFor(next)
	}
	next.Active = next.Active && next.Stability > 0
	if next.Stability > 0 && !next.Active && next.MaintenanceDebt == 0 {
		next.Active = true
	}
	next.EffectSummary = formationEffectSummary(next)
	return next
}

func formationEffectSummary(formation FormationState) []string {
	definition, err := formationDefinitionFor(formation.Kind)
	if err != nil {
		return []string{}
	}
	status := "激活"
	if !formation.Active || formation.Stability <= 0 {
		status = "失效"
	}
	summary := []string{
		fmt.Sprintf("%s挂接于 %s，稳定度 %d，状态=%s。", definition.DisplayName, formation.BuildingID, formation.Stability, status),
	}
	if formation.MaintenanceDebt > 0 {
		summary = append(summary, fmt.Sprintf("维护欠账 %d。", formation.MaintenanceDebt))
	}
	switch formation.Kind {
	case FormationKindGatherSpirit:
		summary = append(summary, "修炼日进度获得额外加成。")
	case FormationKindFieldGuard:
		summary = append(summary, "灵田与灵植产出获得额外加成。")
	case FormationKindDefense:
		summary = append(summary, "敌袭风险与防御压力获得缓冲。")
	case FormationKindCalmMind:
		summary = append(summary, "弟子压力恢复更稳，闭关更安定。")
	}
	return summary
}

func formationStrength(formation FormationState) int {
	formation = normalizeFormationState(formation)
	if !formation.Active || formation.Stability <= 0 {
		return 0
	}
	return clampInt(formation.Level*8+formation.Stability/10-formation.MaintenanceDebt*6, 0, 40)
}

func formationBonusByKind(state SectState, kind FormationKind) int {
	total := 0
	for _, buildingID := range sortedFormationBuildingIDs(state.Formations) {
		formation := state.Formations[buildingID]
		if formation.Kind != kind {
			continue
		}
		total += formationStrength(formation)
	}
	return total
}

func sortedFormationBuildingIDs(formations map[BuildingID]FormationState) []BuildingID {
	ids := make([]BuildingID, 0, len(formations))
	for buildingID := range formations {
		ids = append(ids, buildingID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func buildDefenseRiskProjection(state SectState) DefenseRiskProjection {
	summary := make([]DefenseRiskSourceSummary, 0, 6)
	appendSource := func(source, label string, delta int) {
		if delta == 0 {
			return
		}
		summary = append(summary, DefenseRiskSourceSummary{Source: source, Label: label, Delta: delta})
	}

	intensity := 0
	reputationDelta := clampInt(state.Meta.Reputation/8, 0, 20)
	intensity += reputationDelta
	appendSource("reputation", fmt.Sprintf("名望 %d", state.Meta.Reputation), reputationDelta)

	wealth := defenseRiskWealthScore(state.Resources)
	intensity += wealth
	appendSource("wealth", "宗门财富与库存外显", wealth)

	highTier := defenseRiskHighTierDiscipleScore(state.Disciples)
	intensity += highTier
	appendSource("high_tier_disciples", "高阶弟子与显眼战力", highTier)

	externalExposure := defenseRiskExternalTaskScore(state.Tasks)
	intensity += externalExposure
	appendSource("external_tasks", "外务、探索与战斗外露", externalExposure)

	policyPressure := defenseRiskPolicyScore(state.Policies)
	intensity += policyPressure
	appendSource("policy", "政策偏向带来的外部注意", policyPressure)

	tensionDelta := clampInt(state.Events.Tension*4, 0, 28)
	intensity += tensionDelta
	appendSource("event_tension", "事件张力与暗潮积累", tensionDelta)

	orderPressure := clampInt(state.Order.InternalStrifeRisk/8, 0, 12)
	intensity += orderPressure
	appendSource("order_strife", "宗门内部失序与不满外溢", orderPressure)

	orderMitigation := clampInt((state.Order.Safety+state.Order.Discipline)/16, 0, 12)
	appendSource("order_safety", "山门秩序与治安缓冲外部风险", -orderMitigation)

	mitigation := clampInt(formationBonusByKind(state, FormationKindDefense)/3, 0, 25)
	appendSource("defense_formation", "守御阵缓冲宗门风险", -mitigation)

	return DefenseRiskProjection{
		Intensity:     clampInt(intensity-mitigation-orderMitigation, 0, 100),
		Mitigation:    mitigation + orderMitigation,
		SourceSummary: summary,
	}
}

func defenseRiskWealthScore(resources ResourceState) int {
	score := 0
	score += clampInt(int(resources.Stock[ResourceKindSpiritStone]/15), 0, 14)
	score += clampInt(int(resources.Stock[ResourceKindFormationMat]/4), 0, 6)
	score += clampInt(int((resources.Stock[ResourceKindOre]+resources.Stock[ResourceKindBeastMat])/20), 0, 6)
	return clampInt(score, 0, 24)
}

func defenseRiskHighTierDiscipleScore(disciples map[DiscipleID]DiscipleState) int {
	score := 0
	for _, discipleID := range sortedDiscipleMapIDs(disciples) {
		disciple := disciples[discipleID]
		switch disciple.Realm.Stage {
		case RealmGoldenCore:
			score += 12
		case RealmFoundation:
			score += 8
		case RealmQiLate:
			score += 4
		}
	}
	return clampInt(score, 0, 24)
}

func defenseRiskExternalTaskScore(tasks map[TaskID]TaskState) int {
	score := 0
	for _, taskID := range sortedTaskIDs(tasks) {
		task := tasks[taskID]
		if task.Status != TaskStatusAccepted {
			continue
		}
		switch normalizeTaskType(task.Kind, task.Type) {
		case TaskTypeExternal, TaskTypeExplore:
			score += 4 + clampInt(task.Risk/20, 0, 4)
		case TaskTypeCombat:
			score += 6 + clampInt(task.Risk/15, 0, 6)
		}
	}
	return clampInt(score, 0, 24)
}

func defenseRiskPolicyScore(policies PolicyState) int {
	score := 0
	switch policies.TaskPolicy {
	case TaskPolicyCombat:
		score += 6
	case TaskPolicyRewardExternal:
		score += 4
	case TaskPolicyClosedCultivation:
		score -= 2
	}
	switch policies.ResourcePolicy {
	case ResourcePolicyWarPreparation:
		score += 5
	case ResourcePolicySaving:
		score -= 1
	}
	switch policies.CultivationPolicy {
	case CultivationPolicyClosedCultivation:
		score += 3
	case CultivationPolicyBreakthroughSafe:
		score -= 2
	}
	if policies.RecruitmentPolicy == RecruitmentPolicyAffiliated {
		score += 2
	}
	return clampInt(score, -6, 16)
}

func defaultInstitutionLoops(existing map[InstitutionID]InstitutionLoopState) map[InstitutionID]InstitutionLoopState {
	loops := map[InstitutionID]InstitutionLoopState{
		InstitutionIDGate: {
			InstitutionID: InstitutionIDGate,
			Kind:          "gate",
			Level:         1,
			Enabled:       true,
			Capacity:      2,
			Efficiency:    100,
			GatePolicy: GatePolicyState{
				OpenToVisitors:            true,
				AllowWanderingCultivators: true,
				EnforcementStrictness:     1,
			},
			EffectSummary: []string{"山门开放，候选弟子来源保持默认。"},
		},
		InstitutionIDMainHall: {
			InstitutionID: InstitutionIDMainHall,
			Kind:          "main_hall",
			Level:         1,
			Enabled:       true,
			Capacity:      1,
			Efficiency:    100,
			EffectSummary: []string{"主殿维持基础宗门等级与政策承载。"},
		},
		InstitutionIDTaskHall: {
			InstitutionID:     InstitutionIDTaskHall,
			Kind:              "task_hall",
			Level:             1,
			Enabled:           true,
			Capacity:          8,
			TaskCapacityBonus: 0,
			Efficiency:        100,
			EffectSummary:     []string{"任务堂允许基础任务池与派遣。"},
		},
		InstitutionIDTreasury: {
			InstitutionID:    InstitutionIDTreasury,
			Kind:             "treasury",
			Level:            1,
			Enabled:          true,
			Capacity:         6,
			Efficiency:       100,
			PublicExchange:   true,
			ExchangePressure: 0,
			EffectSummary:    []string{"功勋宝库开放基础贡献兑换。"},
		},
		InstitutionIDDormitory: {
			InstitutionID: InstitutionIDDormitory,
			Kind:          "dormitory",
			Level:         1,
			Enabled:       true,
			Capacity:      mvpDiscipleRosterCapacity,
			Comfort:       50,
			Efficiency:    100,
			EffectSummary: []string{"居舍提供基础名册容量与休息舒适度。"},
		},
		InstitutionIDCanteen: {
			InstitutionID: InstitutionIDCanteen,
			Kind:          "canteen",
			Level:         1,
			Enabled:       true,
			Capacity:      1,
			Comfort:       45,
			Efficiency:    100,
			EffectSummary: []string{"膳堂提供基础供养与满意支撑。"},
		},
		InstitutionIDMedicineHut: {
			InstitutionID: InstitutionIDMedicineHut,
			Kind:          "medicine_hut",
			Level:         1,
			Enabled:       true,
			Capacity:      1,
			HealingPower:  0,
			Efficiency:    100,
			EffectSummary: []string{"药庐待管事后提供治疗与伤病恢复。"},
		},
		InstitutionIDCave: {
			InstitutionID:      InstitutionIDCave,
			Kind:               "cave",
			Level:              1,
			Enabled:            true,
			Capacity:           1,
			CultivationSupport: 6,
			Efficiency:         100,
			CaveSlots: []CaveSlotState{
				{SlotID: "cave-slot-1", EnvironmentBonus: 25},
			},
			EffectSummary: []string{"洞府提供一个基础闭关席位。"},
		},
	}
	for id, current := range existing {
		current.InstitutionID = id
		loops[id] = normalizeInstitutionLoop(current)
	}
	for id, entry := range loops {
		loops[id] = normalizeInstitutionLoop(entry)
	}
	return loops
}

func ensureInstitutionState(state *SectState) {
	state.Institutions.ByID = defaultInstitutionLoops(state.Institutions.ByID)
	for id, institution := range state.Institutions.ByID {
		institution.EffectSummary = institutionEffectSummary(*state, institution)
		state.Institutions.ByID[id] = institution
	}
}

func normalizeInstitutionLoop(institution InstitutionLoopState) InstitutionLoopState {
	if institution.Level <= 0 {
		institution.Level = 1
	}
	if institution.Efficiency <= 0 {
		institution.Efficiency = 100
	}
	institution.Capacity = maxInt(0, institution.Capacity)
	institution.Comfort = clampInt(institution.Comfort, 0, 100)
	institution.HealingPower = maxInt(0, institution.HealingPower)
	institution.CultivationSupport = maxInt(0, institution.CultivationSupport)
	institution.TaskCapacityBonus = maxInt(0, institution.TaskCapacityBonus)
	institution.ExchangePressure = maxInt(0, institution.ExchangePressure)
	institution.AssignedBuildingIDs = append([]BuildingID(nil), institution.AssignedBuildingIDs...)
	institution.ActiveTaskIDs = append([]TaskID(nil), institution.ActiveTaskIDs...)
	institution.GatePolicy.EnforcementStrictness = clampInt(institution.GatePolicy.EnforcementStrictness, 0, 2)
	institution.GatePolicy.GuardDiscipleIDs = append([]DiscipleID(nil), institution.GatePolicy.GuardDiscipleIDs...)
	institution.CaveSlots = cloneCaveSlots(institution.CaveSlots)
	institution.EffectSummary = append([]string(nil), institution.EffectSummary...)
	return institution
}

func applyInstitutionManager(state SectState, institution InstitutionLoopState, manager DiscipleState) InstitutionLoopState {
	next := normalizeInstitutionLoop(institution)
	managerID := manager.DiscipleID
	next.ManagerDiscipleID = &managerID
	next.ManagerEffect = institutionManagerEffect(manager)
	next.Efficiency = 100 + next.ManagerEffect.EfficiencyBonus
	switch next.InstitutionID {
	case InstitutionIDGate:
		next.Capacity = 2 + next.ManagerEffect.EfficiencyBonus/15
	case InstitutionIDMainHall:
		next.Capacity = 1 + next.ManagerEffect.EfficiencyBonus/20
	case InstitutionIDTaskHall:
		next.TaskCapacityBonus = maxInt(1, next.ManagerEffect.EfficiencyBonus/10)
	case InstitutionIDTreasury:
		next.ExchangePressure = institutionExchangePressure(state, next)
	case InstitutionIDDormitory:
		next.Capacity = mvpDiscipleRosterCapacity + maxInt(1, next.ManagerEffect.EfficiencyBonus/10)
		next.Comfort = clampInt(50+next.ManagerEffect.EfficiencyBonus, 0, 100)
	case InstitutionIDCanteen:
		next.Comfort = clampInt(45+next.ManagerEffect.EfficiencyBonus, 0, 100)
	case InstitutionIDMedicineHut:
		next.HealingPower = maxInt(1, next.ManagerEffect.EfficiencyBonus/12)
	case InstitutionIDCave:
		next.CultivationSupport = 6 + maxInt(1, next.ManagerEffect.EfficiencyBonus/8)
		for index := range next.CaveSlots {
			next.CaveSlots[index].EnvironmentBonus = 25 + next.ManagerEffect.EfficiencyBonus/2
		}
	}
	next.EffectSummary = institutionEffectSummary(state, next)
	return next
}

func institutionManagerEffect(manager DiscipleState) InstitutionManagerEffectState {
	identityBonus := identityRankValue(manager.Identity) * 8
	aptitudeBonus := (manager.Aptitude.Comprehension + manager.Aptitude.Mind + manager.Aptitude.SpiritRoot) / 3
	loyaltyModifier := (manager.Loyalty - 50) / 5
	injuryPenalty := manager.InjuryLevel * 8
	managerScore := clampInt(aptitudeBonus+identityBonus+loyaltyModifier-injuryPenalty, 0, 100)
	return InstitutionManagerEffectState{
		ManagerScore:    managerScore,
		IdentityBonus:   identityBonus,
		AptitudeBonus:   aptitudeBonus,
		LoyaltyModifier: loyaltyModifier,
		InjuryPenalty:   injuryPenalty,
		EfficiencyBonus: managerScore / 5,
	}
}

func institutionEffectSummary(state SectState, institution InstitutionLoopState) []string {
	summary := []string{}
	if institution.ManagerDiscipleID != nil {
		summary = append(summary, fmt.Sprintf("管事 %s 效率加成 +%d。", *institution.ManagerDiscipleID, institution.ManagerEffect.EfficiencyBonus))
	}
	switch institution.InstitutionID {
	case InstitutionIDGate:
		summary = append(summary, fmt.Sprintf("山门开放=%t，游方修士=%t，候选加成=%d，执法严格度=%d。", institution.GatePolicy.OpenToVisitors, institution.GatePolicy.AllowWanderingCultivators, institutionRecruitmentBonus(state), institution.GatePolicy.EnforcementStrictness))
	case InstitutionIDMainHall:
		summary = append(summary, fmt.Sprintf("主殿承载宗门等级 %d。", state.Meta.Level))
	case InstitutionIDTaskHall:
		summary = append(summary, fmt.Sprintf("任务上限 %d。", taskHallOpenTaskLimit(state)))
	case InstitutionIDTreasury:
		summary = append(summary, fmt.Sprintf("兑换开放=%t，兑付压力修正=%d。", institution.PublicExchange, institution.ExchangePressure))
	case InstitutionIDDormitory:
		summary = append(summary, fmt.Sprintf("名册容量 %d，舒适 %d。", institution.Capacity, institution.Comfort))
	case InstitutionIDCanteen:
		summary = append(summary, fmt.Sprintf("供养舒适 %d。", institution.Comfort))
	case InstitutionIDMedicineHut:
		summary = append(summary, fmt.Sprintf("治疗能力 %d。", institution.HealingPower))
	case InstitutionIDCave:
		summary = append(summary, fmt.Sprintf("洞府席位 %d，修炼支持 +%d。", len(institution.CaveSlots), institution.CultivationSupport))
	}
	return summary
}

func cloneCaveSlots(source []CaveSlotState) []CaveSlotState {
	if len(source) == 0 {
		return []CaveSlotState{}
	}
	cloned := make([]CaveSlotState, len(source))
	for index, slot := range source {
		cloned[index] = slot
		if slot.OccupiedBy != nil {
			discipleID := *slot.OccupiedBy
			cloned[index].OccupiedBy = &discipleID
		}
	}
	return cloned
}

func institutionByID(state SectState, id InstitutionID) InstitutionLoopState {
	ensureInstitutionState(&state)
	return state.Institutions.ByID[id]
}

func institutionRecruitmentBonus(state SectState) int {
	gate := state.Institutions.ByID[InstitutionIDGate]
	if !gate.Enabled || !gate.GatePolicy.OpenToVisitors {
		return 0
	}
	bonus := 0
	if gate.ManagerDiscipleID != nil && gate.GatePolicy.AllowWanderingCultivators {
		bonus++
	}
	bonus += len(gate.GatePolicy.GuardDiscipleIDs) / 2
	bonus += gate.ManagerEffect.EfficiencyBonus / 10
	return bonus
}

func taskHallOpenTaskLimit(state SectState) int {
	taskHall := state.Institutions.ByID[InstitutionIDTaskHall]
	limit := 8
	if taskHall.Enabled {
		limit += taskHall.TaskCapacityBonus
	}
	return limit
}

func institutionTaskCapabilityBonus(state SectState) int {
	taskHall := state.Institutions.ByID[InstitutionIDTaskHall]
	if !taskHall.Enabled {
		return 0
	}
	return taskHall.ManagerEffect.EfficiencyBonus
}

func institutionCultivationSupport(state SectState, disciple DiscipleState) int64 {
	cave := state.Institutions.ByID[InstitutionIDCave]
	if !cave.Enabled {
		return 0
	}
	if strings.HasPrefix(disciple.WorkTarget.Description, "cultivation:cave") {
		return int64(cave.CultivationSupport)
	}
	return int64(cave.ManagerEffect.EfficiencyBonus / 10)
}

func institutionRecoveryPower(state SectState) int {
	medicine := state.Institutions.ByID[InstitutionIDMedicineHut]
	dormitory := state.Institutions.ByID[InstitutionIDDormitory]
	recovery := 0
	if medicine.Enabled {
		recovery += medicine.HealingPower
	}
	if dormitory.Enabled && dormitory.ManagerDiscipleID != nil {
		recovery += maxInt(1, dormitory.Comfort/50)
	}
	return recovery
}

func institutionSatisfactionSupport(state SectState) int {
	canteen := state.Institutions.ByID[InstitutionIDCanteen]
	dormitory := state.Institutions.ByID[InstitutionIDDormitory]
	support := 0
	if canteen.Enabled && canteen.ManagerDiscipleID != nil {
		support += maxInt(1, canteen.Comfort/50)
	}
	if dormitory.Enabled && dormitory.ManagerDiscipleID != nil {
		support += maxInt(1, dormitory.Comfort/60)
	}
	return support
}

func institutionExchangePressure(state SectState, treasury InstitutionLoopState) int {
	pressure := int(state.Contribution.OutstandingContribution / 20)
	if treasury.ManagerDiscipleID != nil {
		pressure = maxInt(0, pressure-treasury.ManagerEffect.EfficiencyBonus/8)
	}
	return pressure
}

func buildSectOrderProjection(state SectState, version Version) SectOrderState {
	gate := state.Institutions.ByID[InstitutionIDGate]
	guardScore := len(gate.GatePolicy.GuardDiscipleIDs) * 8
	if gate.ManagerDiscipleID != nil {
		guardScore += gate.ManagerEffect.EfficiencyBonus / 2
	}
	formationDefense := formationBonusByKind(state, FormationKindDefense)
	averageSatisfaction := 60
	if len(state.Disciples) > 0 {
		total := 0
		for _, discipleID := range sortedDiscipleMapIDs(state.Disciples) {
			total += state.Disciples[discipleID].Satisfaction
		}
		averageSatisfaction = total / len(state.Disciples)
	}
	averageLoyalty := 60
	if len(state.Disciples) > 0 {
		total := 0
		for _, discipleID := range sortedDiscipleMapIDs(state.Disciples) {
			total += state.Disciples[discipleID].Loyalty
		}
		averageLoyalty = total / len(state.Disciples)
	}
	strictness := clampInt(gate.GatePolicy.EnforcementStrictness, 0, 2)
	safety := clampInt(35+guardScore+formationDefense/8+strictness*8-state.Events.Tension*4+averageLoyalty/8, 0, 100)
	discipline := clampInt(40+guardScore/2+strictness*12+averageLoyalty/6-averageSatisfaction/12-state.Events.Tension*3, 0, 100)
	internalStrifeRisk := clampInt(55-discipline/6-safety/10+state.Events.Tension*7+(60-averageSatisfaction)/2-strictness*4, 0, 100)

	summary := []string{
		fmt.Sprintf("山门守卫与阵法提供安全值 %d。", safety),
		fmt.Sprintf("执法严格度 %d 与弟子忠诚共同形成秩序值 %d。", strictness, discipline),
		fmt.Sprintf("当前内斗/失序风险 %d。", internalStrifeRisk),
	}
	return SectOrderState{
		Safety:             safety,
		Discipline:         discipline,
		InternalStrifeRisk: internalStrifeRisk,
		Summary:            summary,
		LastUpdatedVersion: version,
	}
}

func effectiveExchangeRuleForState(state SectState, rule ExchangeRule) ExchangeRule {
	treasury := state.Institutions.ByID[InstitutionIDTreasury]
	if treasury.Enabled && treasury.ManagerDiscipleID != nil && rule.ContributionCost > 1 {
		rule.ContributionCost = maxInt64(1, rule.ContributionCost-int64(treasury.ManagerEffect.EfficiencyBonus/10))
	}
	if treasury.Enabled && treasury.ManagerDiscipleID != nil {
		rule.MonthlyLimit += int64(treasury.ManagerEffect.EfficiencyBonus / 10)
	}
	return rule
}

func discipleRosterCapacity(state SectState) int {
	dormitory := state.Institutions.ByID[InstitutionIDDormitory]
	if !dormitory.Enabled || dormitory.Capacity <= 0 {
		return mvpDiscipleRosterCapacity
	}
	return dormitory.Capacity
}

func CanAfford(resources ResourceState, cost ResourceCost) bool {
	for kind, amount := range cost {
		if amount < 0 {
			return false
		}
		if resources.Stock[kind] < amount {
			return false
		}
	}
	return true
}

func ApplyResourceDelta(resources *ResourceState, delta map[ResourceKind]int64) error {
	if resources.Stock == nil {
		resources.Stock = map[ResourceKind]int64{}
	}
	for kind, change := range delta {
		next := resources.Stock[kind] + change
		if next < 0 {
			return fmt.Errorf("resource %s would become negative", kind)
		}
		resources.Stock[kind] = next
	}
	return nil
}

func invertCost(cost ResourceCost) map[ResourceKind]int64 {
	delta := make(map[ResourceKind]int64, len(cost))
	for kind, amount := range cost {
		delta[kind] = -amount
	}
	return delta
}

type buildingAuthorityConfig struct {
	DefinitionKey         string
	MaxLevel              int32
	UnlockSectLevel       int32
	RequiredMainHallLevel int32
	MaxCount              int32
	BuildCost             ResourceCost
	UpgradeCostByLevel    map[int32]ResourceCost
	MaintenanceByLevel    map[int32]ResourceCost
}

func buildingConfigFor(definitionKey string) (buildingAuthorityConfig, bool) {
	switch definitionKey {
	case "main_hall":
		return buildingAuthorityConfig{
			DefinitionKey:   definitionKey,
			MaxLevel:        2,
			UnlockSectLevel: 1,
			MaxCount:        1,
			BuildCost: ResourceCost{
				ResourceKindSpiritStone: 30,
				ResourceKindOre:         10,
			},
			UpgradeCostByLevel: map[int32]ResourceCost{
				1: {
					ResourceKindSpiritStone: 45,
					ResourceKindOre:         18,
				},
			},
		}, true
	case "warehouse":
		return buildingAuthorityConfig{
			DefinitionKey:         definitionKey,
			MaxLevel:              2,
			UnlockSectLevel:       1,
			RequiredMainHallLevel: 1,
			MaxCount:              2,
			BuildCost: ResourceCost{
				ResourceKindSpiritStone: 16,
				ResourceKindOre:         6,
			},
			UpgradeCostByLevel: map[int32]ResourceCost{
				1: {
					ResourceKindSpiritStone: 24,
					ResourceKindOre:         10,
				},
			},
			MaintenanceByLevel: map[int32]ResourceCost{
				1: {ResourceKindSpiritStone: 1},
				2: {ResourceKindSpiritStone: 2},
			},
		}, true
	case "gate":
		return buildingAuthorityConfig{
			DefinitionKey:         definitionKey,
			MaxLevel:              2,
			UnlockSectLevel:       2,
			RequiredMainHallLevel: 2,
			MaxCount:              1,
			BuildCost: ResourceCost{
				ResourceKindSpiritStone: 12,
				ResourceKindOre:         8,
			},
			UpgradeCostByLevel: map[int32]ResourceCost{
				1: {
					ResourceKindSpiritStone: 18,
					ResourceKindOre:         12,
				},
			},
			MaintenanceByLevel: map[int32]ResourceCost{
				1: {ResourceKindSpiritStone: 1},
				2: {ResourceKindSpiritStone: 2},
			},
		}, true
	default:
		return buildingAuthorityConfig{}, false
	}
}

func buildingCostFor(definitionKey string) (ResourceCost, bool) {
	config, ok := buildingConfigFor(definitionKey)
	if !ok {
		return nil, false
	}
	return cloneResourceAmounts(config.BuildCost), true
}

func buildingUpgradeCostFor(definitionKey string, currentLevel int32) (ResourceCost, bool) {
	if currentLevel < 1 {
		return nil, false
	}
	config, ok := buildingConfigFor(definitionKey)
	if !ok || currentLevel >= config.MaxLevel {
		return nil, false
	}
	cost, ok := config.UpgradeCostByLevel[currentLevel]
	if !ok {
		return nil, false
	}
	return cloneResourceAmounts(cost), true
}

func buildingMaintenanceCostFor(definitionKey string, level int32) ResourceCost {
	config, ok := buildingConfigFor(definitionKey)
	if !ok || level < 1 {
		return nil
	}
	cost, ok := config.MaintenanceByLevel[level]
	if !ok {
		return nil
	}
	return cloneResourceAmounts(cost)
}

func validateBuildingBuild(state SectState, definitionKey string) error {
	config, ok := buildingConfigFor(definitionKey)
	if !ok {
		return fmt.Errorf("definitionKey %s is not buildable", definitionKey)
	}
	if effectiveSectLevel(state) < config.UnlockSectLevel {
		return fmt.Errorf("%w: sect level %d < unlock level %d for %s", errTaskRequirementNotMet, effectiveSectLevel(state), config.UnlockSectLevel, definitionKey)
	}
	if config.RequiredMainHallLevel > 0 && !stateHasActiveBuilding(state, "main_hall", config.RequiredMainHallLevel) {
		return fmt.Errorf("%w: %s requires main_hall level %d", errTaskRequirementNotMet, definitionKey, config.RequiredMainHallLevel)
	}
	if int32(len(state.Buildings)) >= effectiveBuildingLimit(state) {
		return fmt.Errorf("%w: building limit %d reached", errTaskRequirementNotMet, effectiveBuildingLimit(state))
	}
	if countBuildingsByDefinition(state, definitionKey) >= config.MaxCount {
		return fmt.Errorf("%w: building count limit for %s reached", errTaskRequirementNotMet, definitionKey)
	}
	return nil
}

func validateBuildingUpgrade(state SectState, building BuildingState) error {
	config, ok := buildingConfigFor(building.DefinitionKey)
	if !ok {
		return fmt.Errorf("building %s is not upgradeable", building.DefinitionKey)
	}
	if building.Phase != "active" {
		return fmt.Errorf("%w: building %s is not active", errTaskRequirementNotMet, building.BuildingID)
	}
	if building.Level >= config.MaxLevel {
		return fmt.Errorf("%w: building %s has reached max level %d", errTaskRequirementNotMet, building.DefinitionKey, config.MaxLevel)
	}
	nextLevel := building.Level + 1
	if building.DefinitionKey != "main_hall" && effectiveSectLevel(state) < nextLevel {
		return fmt.Errorf("%w: sect level %d < upgrade level %d for %s", errTaskRequirementNotMet, effectiveSectLevel(state), nextLevel, building.DefinitionKey)
	}
	if config.RequiredMainHallLevel > 0 && !stateHasActiveBuilding(state, "main_hall", config.RequiredMainHallLevel) {
		return fmt.Errorf("%w: %s requires main_hall level %d", errTaskRequirementNotMet, building.DefinitionKey, config.RequiredMainHallLevel)
	}
	return nil
}

func effectiveSectLevel(state SectState) int32 {
	level := state.Meta.Level
	if level < 1 {
		level = 1
	}
	for _, building := range state.Buildings {
		if building.DefinitionKey == "main_hall" && building.Level > level && building.HP > 0 && building.Phase == "active" {
			level = building.Level
		}
	}
	return level
}

func sectBuildingLimitForLevel(level int32) int32 {
	if level < 1 {
		level = 1
	}
	return 2 + level*2
}

func effectiveBuildingLimit(state SectState) int32 {
	if state.Meta.BuildingLimit > 0 {
		return state.Meta.BuildingLimit
	}
	return sectBuildingLimitForLevel(effectiveSectLevel(state))
}

func countBuildingsByDefinition(state SectState, definitionKey string) int32 {
	var count int32
	for _, building := range state.Buildings {
		if building.DefinitionKey == definitionKey {
			count++
		}
	}
	return count
}

func refreshSectBuildingMeta(state *SectState) {
	level := effectiveSectLevel(*state)
	state.Meta.Level = level
	state.Meta.Expansion = int32(len(state.Buildings))
	derivedLimit := sectBuildingLimitForLevel(level)
	if state.Meta.BuildingLimit < derivedLimit {
		state.Meta.BuildingLimit = derivedLimit
	}
}

func encodeSignedVarint(value int64) []byte {
	buffer := make([]byte, binary.MaxVarintLen64)
	length := binary.PutVarint(buffer, value)
	return buffer[:length]
}

func newBuildingState(buildingID BuildingID, definitionKey string, level int32, origin TileCoord) BuildingState {
	maxHP := int64(100 + (level-1)*50)
	return BuildingState{
		BuildingID:    buildingID,
		DefinitionKey: definitionKey,
		Level:         level,
		Phase:         "active",
		Origin:        origin,
		HP:            maxHP,
		MaxHP:         maxHP,
		Durability:    100,
		Efficiency:    100,
	}
}

func buildingParticipatesInMaintenance(building BuildingState) bool {
	return building.HP > 0 && (building.Phase == "active" || building.Phase == "damaged")
}

func normalizedBuildingRuntimeFields(building BuildingState) BuildingState {
	next := building
	if next.MaxHP <= 0 {
		next.MaxHP = 100
	}
	if next.HP <= 0 {
		next.HP = next.MaxHP
	}
	if next.Efficiency <= 0 {
		next.Efficiency = 100
	}
	if next.Durability <= 0 {
		next.Durability = buildingDurabilityPercent(next)
	}
	if next.Phase == "" {
		next.Phase = "active"
	}
	return next
}

func buildingAfterMaintenancePaid(building BuildingState) BuildingState {
	next := normalizedBuildingRuntimeFields(building)
	next.MaintenanceDebt = 0
	next.Efficiency = minInt(100, next.Efficiency+5)
	next.Durability = buildingDurabilityPercent(next)
	if next.Phase == "damaged" && next.Durability >= 60 {
		next.Phase = "active"
		next.DamagedReason = ""
	}
	return next
}

func buildingAfterMaintenanceShortage(building BuildingState, cost ResourceCost, resources ResourceState) (BuildingState, map[ResourceKind]int64) {
	next := normalizedBuildingRuntimeFields(building)
	next.MaintenanceDebt++
	next.Efficiency = maxInt(25, next.Efficiency-25)
	damage := maxInt64(5, next.MaxHP/5)
	next.HP = maxInt64(1, next.HP-damage)
	next.Durability = buildingDurabilityPercent(next)
	next.Phase = "damaged"
	next.DamagedReason = "maintenance_shortage"
	return next, maintenanceShortage(resources, cost)
}

func buildingNeedsRepair(building BuildingState) bool {
	return building.Phase == "damaged" ||
		building.HP < building.MaxHP ||
		building.Efficiency < 100 ||
		building.Durability < 100 ||
		building.MaintenanceDebt > 0 ||
		building.DamagedReason != ""
}

func buildingRepairCostFor(building BuildingState) ResourceCost {
	cost := ResourceCost{}
	maintenance := buildingMaintenanceCostFor(building.DefinitionKey, building.Level)
	for kind, amount := range maintenance {
		if amount > 0 {
			cost[kind] = amount * int64(maxInt(1, building.MaintenanceDebt))
		}
	}
	if len(cost) == 0 {
		fallback := int64(building.Level)
		if fallback < 1 {
			fallback = 1
		}
		cost[ResourceKindSpiritStone] = fallback
	}
	return cost
}

func buildingAfterRepair(building BuildingState) BuildingState {
	next := normalizedBuildingRuntimeFields(building)
	next.HP = next.MaxHP
	next.Efficiency = 100
	next.Durability = 100
	next.MaintenanceDebt = 0
	next.Phase = "active"
	next.DamagedReason = ""
	return next
}

func maintenanceShortage(resources ResourceState, cost ResourceCost) map[ResourceKind]int64 {
	shortage := map[ResourceKind]int64{}
	for kind, amount := range cost {
		if amount <= 0 {
			continue
		}
		if resources.Stock[kind] < amount {
			shortage[kind] = amount - resources.Stock[kind]
		}
	}
	return shortage
}

func buildingDurabilityPercent(building BuildingState) int {
	if building.MaxHP <= 0 {
		return 100
	}
	percent := int(building.HP * 100 / building.MaxHP)
	return clampInt(percent, 0, 100)
}

func mustMarshalBuildingPayload(payload any) []byte {
	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return body
}

func buildingChangedDelta(version Version, payload []byte, building BuildingState) (ClientEvent, []PatchOp, error) {
	buildingBlob, err := json.Marshal(building)
	if err != nil {
		return ClientEvent{}, nil, fmt.Errorf("marshal building patch payload: %w", err)
	}
	return ClientEvent{
			SceneVersion: version,
			Type:         ClientEventTypeBuildingChanged,
			Payload:      payload,
		}, []PatchOp{{
			Op:            PatchOpTypeSet,
			Path:          fmt.Sprintf("/buildings/%s", building.BuildingID),
			Value:         buildingBlob,
			ValueEncoding: PatchValueEncodingBytes,
		}}, nil
}

func taskChangedDelta(version Version, payload []byte, task TaskState) (ClientEvent, []PatchOp, error) {
	taskBlob, err := json.Marshal(task)
	if err != nil {
		return ClientEvent{}, nil, fmt.Errorf("marshal task patch payload: %w", err)
	}
	return ClientEvent{
			SceneVersion: version,
			Type:         ClientEventTypeTaskChanged,
			Payload:      payload,
		}, []PatchOp{{
			Op:            PatchOpTypeSet,
			Path:          fmt.Sprintf("/tasks/%s", task.TaskID),
			Value:         taskBlob,
			ValueEncoding: PatchValueEncodingBytes,
		}}, nil
}

func discipleChangedDelta(version Version, payload []byte, disciple DiscipleState) (ClientEvent, []PatchOp, error) {
	discipleBlob, err := json.Marshal(disciple)
	if err != nil {
		return ClientEvent{}, nil, fmt.Errorf("marshal disciple patch payload: %w", err)
	}
	return ClientEvent{
			SceneVersion: version,
			Type:         ClientEventTypeDiscipleChanged,
			Payload:      payload,
		}, []PatchOp{{
			Op:            PatchOpTypeSet,
			Path:          fmt.Sprintf("/disciples/%s", disciple.DiscipleID),
			Value:         discipleBlob,
			ValueEncoding: PatchValueEncodingBytes,
		}}, nil
}

func sectMetaChangedDelta(version Version, payload []byte, meta SectMetaState) (ClientEvent, []PatchOp, error) {
	metaBlob, err := json.Marshal(meta)
	if err != nil {
		return ClientEvent{}, nil, fmt.Errorf("marshal sect meta patch payload: %w", err)
	}
	return ClientEvent{
			SceneVersion: version,
			Type:         ClientEventTypeSectMetaChanged,
			Payload:      payload,
		}, []PatchOp{{
			Op:            PatchOpTypeSet,
			Path:          "/meta",
			Value:         metaBlob,
			ValueEncoding: PatchValueEncodingBytes,
		}}, nil
}

func contributionChangedDelta(version Version, payload []byte, _ DiscipleID) (ClientEvent, []PatchOp, error) {
	return ClientEvent{
		SceneVersion: version,
		Type:         ClientEventTypeContributionChanged,
		Payload:      payload,
	}, nil, nil
}

func timeChangedDelta(version Version, payload []byte, timeState SectTimeState) (ClientEvent, []PatchOp, error) {
	timeBlob, err := json.Marshal(timeState)
	if err != nil {
		return ClientEvent{}, nil, fmt.Errorf("marshal time patch payload: %w", err)
	}
	return ClientEvent{
			SceneVersion: version,
			Type:         ClientEventTypeTimeChanged,
			Payload:      payload,
		}, []PatchOp{{
			Op:            PatchOpTypeSet,
			Path:          "/time",
			Value:         timeBlob,
			ValueEncoding: PatchValueEncodingBytes,
		}}, nil
}

func monthlyChangedDelta(version Version, payload []byte, monthly MonthlyState) (ClientEvent, []PatchOp, error) {
	monthlyBlob, err := json.Marshal(monthly)
	if err != nil {
		return ClientEvent{}, nil, fmt.Errorf("marshal monthly patch payload: %w", err)
	}
	return ClientEvent{
			SceneVersion: version,
			Type:         ClientEventTypeMonthlyChanged,
			Payload:      payload,
		}, []PatchOp{{
			Op:            PatchOpTypeSet,
			Path:          "/monthly",
			Value:         monthlyBlob,
			ValueEncoding: PatchValueEncodingBytes,
		}}, nil
}

func orderChangedDelta(version Version, payload []byte, order SectOrderState) (ClientEvent, []PatchOp, error) {
	orderBlob, err := json.Marshal(order)
	if err != nil {
		return ClientEvent{}, nil, fmt.Errorf("marshal order patch payload: %w", err)
	}
	return ClientEvent{
			SceneVersion: version,
			Type:         ClientEventTypeOrderChanged,
			Payload:      payload,
		}, []PatchOp{{
			Op:            PatchOpTypeSet,
			Path:          "/order",
			Value:         orderBlob,
			ValueEncoding: PatchValueEncodingBytes,
		}}, nil
}

func monthlyAssessmentChangedDelta(version Version, payload []byte, assessment MonthlyAssessmentState) (ClientEvent, []PatchOp, error) {
	assessmentBlob, err := json.Marshal(assessment)
	if err != nil {
		return ClientEvent{}, nil, fmt.Errorf("marshal monthly assessment patch payload: %w", err)
	}
	return ClientEvent{
			SceneVersion: version,
			Type:         ClientEventTypeMonthlyAssessmentChanged,
			Payload:      payload,
		}, []PatchOp{{
			Op:            PatchOpTypeSet,
			Path:          "/monthly_assessment",
			Value:         assessmentBlob,
			ValueEncoding: PatchValueEncodingBytes,
		}}, nil
}

func policyChangedDelta(version Version, payload []byte, policies PolicyState) (ClientEvent, []PatchOp, error) {
	policyBlob, err := json.Marshal(policies)
	if err != nil {
		return ClientEvent{}, nil, fmt.Errorf("marshal policy patch payload: %w", err)
	}
	var changed PolicyChangedPayload
	if err := json.Unmarshal(payload, &changed); err != nil {
		return ClientEvent{}, nil, fmt.Errorf("decode policy changed payload: %w", err)
	}
	ops := []PatchOp{{
		Op:            PatchOpTypeSet,
		Path:          "/policies",
		Value:         policyBlob,
		ValueEncoding: PatchValueEncodingBytes,
	}}
	for _, taskID := range sortedTaskIDs(changed.Tasks) {
		task := changed.Tasks[taskID]
		taskBlob, err := json.Marshal(task)
		if err != nil {
			return ClientEvent{}, nil, fmt.Errorf("marshal policy task patch payload: %w", err)
		}
		ops = append(ops, PatchOp{
			Op:            PatchOpTypeSet,
			Path:          fmt.Sprintf("/tasks/%s", task.TaskID),
			Value:         taskBlob,
			ValueEncoding: PatchValueEncodingBytes,
		})
	}
	for _, productionID := range sortedProductionIDs(changed.Productions) {
		production := changed.Productions[productionID]
		productionBlob, err := json.Marshal(production)
		if err != nil {
			return ClientEvent{}, nil, fmt.Errorf("marshal policy production patch payload: %w", err)
		}
		ops = append(ops, PatchOp{
			Op:            PatchOpTypeSet,
			Path:          fmt.Sprintf("/productions/%s", production.ProductionID),
			Value:         productionBlob,
			ValueEncoding: PatchValueEncodingBytes,
		})
	}
	return ClientEvent{
		SceneVersion: version,
		Type:         ClientEventTypePolicyChanged,
		Payload:      payload,
	}, ops, nil
}

func institutionChangedDelta(version Version, payload []byte, changed InstitutionChangedPayload) (ClientEvent, []PatchOp, error) {
	institutionBlob, err := json.Marshal(changed.Institution)
	if err != nil {
		return ClientEvent{}, nil, fmt.Errorf("marshal institution patch payload: %w", err)
	}
	ops := []PatchOp{{
		Op:            PatchOpTypeSet,
		Path:          fmt.Sprintf("/institutions/by_id/%s", changed.Institution.InstitutionID),
		Value:         institutionBlob,
		ValueEncoding: PatchValueEncodingBytes,
	}}
	if changed.ExchangeRule != nil {
		ruleBlob, err := json.Marshal(changed.ExchangeRule)
		if err != nil {
			return ClientEvent{}, nil, fmt.Errorf("marshal exchange rule patch payload: %w", err)
		}
		ops = append(ops, PatchOp{
			Op:            PatchOpTypeSet,
			Path:          fmt.Sprintf("/contribution/treasury_rules/%s", changed.ExchangeRule.ExchangeItemID),
			Value:         ruleBlob,
			ValueEncoding: PatchValueEncodingBytes,
		})
	}
	return ClientEvent{
		SceneVersion: version,
		Type:         ClientEventTypeInstitutionChanged,
		Payload:      payload,
	}, ops, nil
}

func inventoryChangedDelta(version Version, payload []byte, item InventoryEntry) (ClientEvent, []PatchOp, error) {
	itemBlob, err := json.Marshal(item)
	if err != nil {
		return ClientEvent{}, nil, fmt.Errorf("marshal inventory patch payload: %w", err)
	}
	return ClientEvent{
			SceneVersion: version,
			Type:         ClientEventTypeInventoryChanged,
			Payload:      payload,
		}, []PatchOp{{
			Op:            PatchOpTypeSet,
			Path:          fmt.Sprintf("/inventory/items/%s", item.ItemID),
			Value:         itemBlob,
			ValueEncoding: PatchValueEncodingBytes,
		}}, nil
}

func artifactChangedDelta(version Version, payload []byte, artifact ArtifactState, disciple *DiscipleState) (ClientEvent, []PatchOp, error) {
	artifactBlob, err := json.Marshal(artifact)
	if err != nil {
		return ClientEvent{}, nil, fmt.Errorf("marshal artifact patch payload: %w", err)
	}
	ops := []PatchOp{{
		Op:            PatchOpTypeSet,
		Path:          fmt.Sprintf("/inventory/artifacts/%s", artifact.ItemID),
		Value:         artifactBlob,
		ValueEncoding: PatchValueEncodingBytes,
	}}
	if disciple != nil {
		discipleBlob, err := json.Marshal(disciple)
		if err != nil {
			return ClientEvent{}, nil, fmt.Errorf("marshal artifact disciple patch payload: %w", err)
		}
		ops = append(ops, PatchOp{
			Op:            PatchOpTypeSet,
			Path:          fmt.Sprintf("/disciples/%s", disciple.DiscipleID),
			Value:         discipleBlob,
			ValueEncoding: PatchValueEncodingBytes,
		})
	}
	return ClientEvent{
		SceneVersion: version,
		Type:         ClientEventTypeInventoryChanged,
		Payload:      payload,
	}, ops, nil
}

func formationChangedDelta(version Version, payload []byte, change FormationChangedPayload) (ClientEvent, []PatchOp, error) {
	formationBlob := []byte("null")
	if !change.Detached {
		var err error
		formationBlob, err = json.Marshal(normalizeFormationState(change.Formation))
		if err != nil {
			return ClientEvent{}, nil, fmt.Errorf("marshal formation patch payload: %w", err)
		}
	}
	return ClientEvent{
			SceneVersion: version,
			Type:         ClientEventTypeFormationChanged,
			Payload:      payload,
		}, []PatchOp{{
			Op:            PatchOpTypeSet,
			Path:          fmt.Sprintf("/formations/%s", change.Formation.BuildingID),
			Value:         formationBlob,
			ValueEncoding: PatchValueEncodingBytes,
		}}, nil
}

func productionChangedDelta(version Version, payload []byte, production ProductionJob) (ClientEvent, []PatchOp, error) {
	productionBlob, err := json.Marshal(production)
	if err != nil {
		return ClientEvent{}, nil, fmt.Errorf("marshal production patch payload: %w", err)
	}
	return ClientEvent{
			SceneVersion: version,
			Type:         ClientEventTypeProductionChanged,
			Payload:      payload,
		}, []PatchOp{{
			Op:            PatchOpTypeSet,
			Path:          fmt.Sprintf("/productions/%s", production.ProductionID),
			Value:         productionBlob,
			ValueEncoding: PatchValueEncodingBytes,
		}}, nil
}

func admissionChangedDelta(
	version Version,
	payload []byte,
	admissions AdmissionState,
	disciple *DiscipleState,
	account *ContributionAccount,
) (ClientEvent, []PatchOp, error) {
	admissionsBlob, err := json.Marshal(admissions)
	if err != nil {
		return ClientEvent{}, nil, fmt.Errorf("marshal admission patch payload: %w", err)
	}
	ops := []PatchOp{{
		Op:            PatchOpTypeSet,
		Path:          "/admissions",
		Value:         admissionsBlob,
		ValueEncoding: PatchValueEncodingBytes,
	}}
	if disciple != nil {
		discipleBlob, err := json.Marshal(disciple)
		if err != nil {
			return ClientEvent{}, nil, fmt.Errorf("marshal recruited disciple patch payload: %w", err)
		}
		ops = append(ops, PatchOp{
			Op:            PatchOpTypeSet,
			Path:          fmt.Sprintf("/disciples/%s", disciple.DiscipleID),
			Value:         discipleBlob,
			ValueEncoding: PatchValueEncodingBytes,
		})
	}
	if account != nil {
		accountBlob, err := json.Marshal(account)
		if err != nil {
			return ClientEvent{}, nil, fmt.Errorf("marshal contribution account patch payload: %w", err)
		}
		ops = append(ops, PatchOp{
			Op:            PatchOpTypeSet,
			Path:          fmt.Sprintf("/contribution/accounts/%s", account.DiscipleID),
			Value:         accountBlob,
			ValueEncoding: PatchValueEncodingBytes,
		})
	}
	return ClientEvent{
		SceneVersion: version,
		Type:         ClientEventTypeAdmissionChanged,
		Payload:      payload,
	}, ops, nil
}

func ensureSectEventState(state *SectState) {
	if state.Events.ActiveEvents == nil {
		state.Events.ActiveEvents = map[EventID]SectEvent{}
	}
	if state.Events.ResolvedEvents == nil {
		state.Events.ResolvedEvents = []ResolvedEventSummary{}
	}
}

func normalizeSectGoal(goal SectGoal) SectGoal {
	next := goal
	if next.TargetProgress <= 0 {
		next.TargetProgress = 1
	}
	if next.CurrentProgress < 0 {
		next.CurrentProgress = 0
	}
	if next.CurrentProgress > next.TargetProgress {
		next.CurrentProgress = next.TargetProgress
	}
	if next.Status == "" {
		next.Status = SectGoalStatusActive
	}
	next.RewardResources = cloneResourceAmounts(next.RewardResources)
	next.RewardSummary = append([]string(nil), next.RewardSummary...)
	next.Tags = sortedUniqueStrings(next.Tags)
	return next
}

func sortedGoalIDs(goals map[SectGoalID]SectGoal) []SectGoalID {
	ids := make([]SectGoalID, 0, len(goals))
	for goalID := range goals {
		ids = append(ids, goalID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func eventExpiredAtDay(event SectEvent, calendarDay int32) bool {
	return event.ExpiresAtDay > 0 && calendarDay >= event.ExpiresAtDay
}

func findSectEventOption(event SectEvent, optionID string) (SectEventOption, bool) {
	optionID = strings.TrimSpace(optionID)
	if optionID == "" {
		return SectEventOption{}, false
	}
	for _, option := range event.Options {
		if option.OptionID == optionID {
			return option, true
		}
	}
	return SectEventOption{}, false
}

func sectEventRequirementMet(state SectState, requirement SectEventRequirement) bool {
	for kind, amount := range requirement.MinResources {
		if amount < 0 {
			return false
		}
		if state.Resources.Stock[kind] < amount {
			return false
		}
	}
	if requirement.RequiredDiscipleID != nil {
		if _, ok := state.Disciples[*requirement.RequiredDiscipleID]; !ok {
			return false
		}
	}
	return true
}

func resourceDeltaAffordable(resources ResourceState, delta map[ResourceKind]int64) bool {
	for kind, change := range delta {
		if change >= 0 {
			continue
		}
		if resources.Stock[kind] < -change {
			return false
		}
	}
	return true
}

func sectEventOptionSummary(event SectEvent, option SectEventOption) string {
	if strings.TrimSpace(option.ResultPreview.Summary) != "" {
		return option.ResultPreview.Summary
	}
	return fmt.Sprintf("宗门事件《%s》选择了「%s」。", event.Title, option.Label)
}

func (a *SectActor) appendSectGoalProgressEvents(
	simulated *SectState,
	versionCursor *Version,
	appendEvent appendDomainEventFunc,
) error {
	if simulated == nil {
		return nil
	}
	ensureSectGoalState(simulated)
	for _, goalID := range sortedGoalIDs(simulated.Goals.ByID) {
		current := normalizeSectGoal(simulated.Goals.ByID[goalID])
		if current.Status != SectGoalStatusActive {
			continue
		}
		next := projectSectGoal(*simulated, current)
		if next.Status == SectGoalStatusCompleted || next.Status == SectGoalStatusFailed {
			if err := a.appendSectGoalOutcomeEvents(simulated, versionCursor, appendEvent, current, next); err != nil {
				return err
			}
			continue
		}
		if !sectGoalProgressChanged(current, next) {
			continue
		}
		next.StartedAtVersion = current.StartedAtVersion
		if next.StartedAtVersion == 0 {
			next.StartedAtVersion = *versionCursor + 1
		}
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectGoalChanged, SectGoalChangedPayload{
			Goal: next,
		}, *versionCursor+1)); err != nil {
			return err
		}
	}
	return nil
}

func projectSectGoal(state SectState, current SectGoal) SectGoal {
	next := normalizeSectGoal(current)
	switch next.Kind {
	case "cave_routine":
		next.CurrentProgress = 0
		next.ProgressText = "等待首名弟子入洞闭关。"
		if discipleID, ok := firstReservedCaveDisciple(state); ok {
			next.CurrentProgress = 1
			next.FocusDiscipleID = &discipleID
			next.ProgressText = fmt.Sprintf("%s 已进入洞府闭关。", state.Disciples[discipleID].Name)
			next.Status = SectGoalStatusCompleted
			next.OutcomeSummary = fmt.Sprintf("《%s》达成：%s 入洞闭关，宗门洞府体系正式运转。", next.Title, state.Disciples[discipleID].Name)
		}
	case "inner_disciple":
		next.CurrentProgress = 0
		next.ProgressText = "尚未出现内门弟子。"
		if discipleID, ok := firstInnerDisciple(state); ok {
			next.CurrentProgress = 1
			next.FocusDiscipleID = &discipleID
			next.ProgressText = fmt.Sprintf("%s 已晋升为内门弟子。", state.Disciples[discipleID].Name)
			next.Status = SectGoalStatusCompleted
			next.OutcomeSummary = fmt.Sprintf("《%s》达成：%s 完成晋升，宗门开始形成内门骨干。", next.Title, state.Disciples[discipleID].Name)
		}
	case "external_affairs":
		next.CurrentProgress = 0
		next.ProgressText = "尚未完成外务/探索/战斗。"
		if task, discipleID, ok := firstCompletedExternalGoalTask(state); ok {
			next.CurrentProgress = 1
			next.FocusDiscipleID = &discipleID
			next.ProgressText = fmt.Sprintf("《%s》已完成。", task.Title)
			next.Status = SectGoalStatusCompleted
			next.OutcomeSummary = fmt.Sprintf("《%s》达成：%s 完成《%s》，宗门已能稳定回应山外事务。", next.Title, state.Disciples[discipleID].Name, task.Title)
		}
	case "stable_monthly":
		next.CurrentProgress = 0
		if state.Monthly.LastSettlement.MonthIndex <= 0 {
			next.ProgressText = "等待首个完整月结。"
			break
		}
		next.CurrentProgress = 1
		focusDiscipleID := defaultGoalFocusDisciple(state, nil)
		if focusDiscipleID != "" {
			next.FocusDiscipleID = &focusDiscipleID
		}
		if monthlySettlementStable(state.Monthly.LastSettlement) {
			next.ProgressText = fmt.Sprintf("第 %d 月月结平稳完成。", state.Monthly.LastSettlement.MonthIndex)
			next.Status = SectGoalStatusCompleted
			next.OutcomeSummary = fmt.Sprintf("《%s》达成：第 %d 月无欠发、无违约、无明显资源短缺。", next.Title, state.Monthly.LastSettlement.MonthIndex)
		} else {
			next.ProgressText = fmt.Sprintf("第 %d 月月结失衡。", state.Monthly.LastSettlement.MonthIndex)
			next.Status = SectGoalStatusFailed
			next.OutcomeSummary = fmt.Sprintf("《%s》失败：月结出现欠发/违约/资源短缺，宗门治理压力上升。", next.Title)
		}
	}
	return normalizeSectGoal(next)
}

func sectGoalProgressChanged(current SectGoal, next SectGoal) bool {
	if current.Status != next.Status ||
		current.CurrentProgress != next.CurrentProgress ||
		current.TargetProgress != next.TargetProgress ||
		current.ProgressText != next.ProgressText ||
		current.OutcomeSummary != next.OutcomeSummary {
		return true
	}
	if current.FocusDiscipleID == nil && next.FocusDiscipleID == nil {
		return false
	}
	if current.FocusDiscipleID == nil || next.FocusDiscipleID == nil {
		return true
	}
	return *current.FocusDiscipleID != *next.FocusDiscipleID
}

func (a *SectActor) appendSectGoalOutcomeEvents(
	simulated *SectState,
	versionCursor *Version,
	appendEvent appendDomainEventFunc,
	current SectGoal,
	next SectGoal,
) error {
	next.StartedAtVersion = current.StartedAtVersion
	if next.StartedAtVersion == 0 {
		next.StartedAtVersion = *versionCursor + 1
	}
	focusDiscipleID := defaultGoalFocusDisciple(*simulated, next.FocusDiscipleID)
	if next.FocusDiscipleID == nil && focusDiscipleID != "" {
		next.FocusDiscipleID = &focusDiscipleID
	}
	satisfactionDelta := next.RewardSatisfaction
	if next.Status == SectGoalStatusFailed {
		satisfactionDelta = next.FailureSatisfaction
	}
	reputationDelta := next.RewardReputation
	if next.Status == SectGoalStatusFailed {
		reputationDelta = next.FailureReputation
	}
	resourceDelta := cloneResourceAmounts(next.RewardResources)
	if next.Status == SectGoalStatusFailed {
		resourceDelta = nil
	}
	if len(resourceDelta) > 0 {
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeResourceChanged, ResourceChangedPayload{
			Changes: resourceDelta,
			Reason:  fmt.Sprintf("sect_goal:%s:%s", next.Status, next.GoalID),
		}, *versionCursor+1)); err != nil {
			return err
		}
	}
	if reputationDelta != 0 {
		nextMeta := simulated.Meta
		nextMeta.Reputation = maxInt(0, nextMeta.Reputation+reputationDelta)
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectMetaChanged, SectMetaChangedPayload{
			Meta:   nextMeta,
			Reason: fmt.Sprintf("sect_goal:%s:%s", next.Status, next.GoalID),
		}, *versionCursor+1)); err != nil {
			return err
		}
	}
	if satisfactionDelta != 0 && focusDiscipleID != "" {
		if disciple, ok := simulated.Disciples[focusDiscipleID]; ok {
			nextDisciple := disciple
			nextDisciple.Satisfaction = clampInt(nextDisciple.Satisfaction+satisfactionDelta, 0, 100)
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeDiscipleSatisfactionChanged, DiscipleChangedPayload{
				Disciple: nextDisciple,
			}, *versionCursor+1)); err != nil {
				return err
			}
		}
	}
	next.ResolvedAtVersion = *versionCursor + 1
	outcome := "completed"
	if next.Status == SectGoalStatusFailed {
		outcome = "failed"
	}
	if strings.TrimSpace(next.OutcomeSummary) == "" {
		next.OutcomeSummary = fmt.Sprintf("宗门目标《%s》已%s。", next.Title, outcome)
	}
	if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectGoalResolved, SectGoalResolvedPayload{
		Goal: next,
		Summary: ResolvedSectGoalSummary{
			GoalID:            next.GoalID,
			Kind:              next.Kind,
			Outcome:           outcome,
			Summary:           next.OutcomeSummary,
			ResolvedAtVersion: next.ResolvedAtVersion,
		},
	}, *versionCursor+1)); err != nil {
		return err
	}
	return nil
}

func defaultGoalFocusDisciple(state SectState, explicit *DiscipleID) DiscipleID {
	if explicit != nil {
		if _, ok := state.Disciples[*explicit]; ok {
			return *explicit
		}
	}
	if _, ok := state.Disciples[starterDiscipleID]; ok {
		return starterDiscipleID
	}
	ids := sortedDiscipleMapIDs(state.Disciples)
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

func firstReservedCaveDisciple(state SectState) (DiscipleID, bool) {
	cave, ok := state.Institutions.ByID[InstitutionIDCave]
	if !ok {
		return "", false
	}
	for _, slot := range cave.CaveSlots {
		if slot.OccupiedBy != nil {
			return *slot.OccupiedBy, true
		}
	}
	for _, discipleID := range sortedDiscipleMapIDs(state.Disciples) {
		disciple := state.Disciples[discipleID]
		if strings.HasPrefix(disciple.WorkTarget.Description, "cultivation:cave") {
			return discipleID, true
		}
	}
	return "", false
}

func firstInnerDisciple(state SectState) (DiscipleID, bool) {
	for _, discipleID := range sortedDiscipleMapIDs(state.Disciples) {
		if state.Disciples[discipleID].Identity == IdentityRankInner {
			return discipleID, true
		}
	}
	return "", false
}

func firstCompletedExternalGoalTask(state SectState) (TaskState, DiscipleID, bool) {
	for _, taskID := range sortedTaskIDs(state.Tasks) {
		task := state.Tasks[taskID]
		if task.Status != TaskStatusCompleted {
			continue
		}
		taskType := normalizeTaskType(task.Kind, task.Type)
		if taskType != TaskTypeExternal && taskType != TaskTypeExplore && taskType != TaskTypeCombat {
			continue
		}
		if len(task.AssignedDiscipleIDs) == 0 {
			continue
		}
		discipleID := sortedDiscipleIDs(task.AssignedDiscipleIDs)[0]
		return task, discipleID, true
	}
	return TaskState{}, "", false
}

func monthlySettlementStable(summary MonthlySettlementSummary) bool {
	if summary.MonthIndex <= 0 {
		return false
	}
	return summary.PayrollDelayedCount == 0 &&
		summary.DutyViolations == 0 &&
		!summary.ResourceShortage &&
		!summary.ContributionShortage
}

func activeSectEventTension(events map[EventID]SectEvent) int {
	maxSeverity := 0
	for _, event := range events {
		if event.Severity > maxSeverity {
			maxSeverity = event.Severity
		}
	}
	return maxSeverity
}

func clientDeltaForEvents(events []DomainEvent) ([]ClientEvent, []PatchOp, error) {
	clientEvents := make([]ClientEvent, 0, len(events))
	ops := make([]PatchOp, 0, len(events))
	for _, event := range events {
		clientEvent, patchOps, err := eventToClientDelta(event)
		if err != nil {
			return nil, nil, err
		}
		clientEvents = append(clientEvents, clientEvent)
		ops = append(ops, patchOps...)
	}
	return clientEvents, ops, nil
}

func (a *SectActor) applyDomainEvents(events []DomainEvent) error {
	for _, event := range events {
		if err := ApplyEvent(&a.state, event); err != nil {
			return err
		}
		a.eventLog = boundDomainEventLog(append(a.eventLog, cloneDomainEvents([]DomainEvent{event})...))
	}
	return nil
}

func (a *SectActor) newDomainEvent(commandID string, eventType DomainEventType, payload any, version Version) DomainEvent {
	return DomainEvent{
		EventID:   fmt.Sprintf("%s@%d", eventType, version),
		SectID:    a.state.Meta.SectID,
		Version:   version,
		Type:      eventType,
		Payload:   mustMarshalPayload(payload),
		CommandID: commandID,
		GameTick:  a.state.Time.GameTick,
	}
}

func mustMarshalPayload(payload any) []byte {
	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return body
}

func normalizePositiveResourceAmounts(values map[ResourceKind]int64) map[ResourceKind]int64 {
	if len(values) == 0 {
		return nil
	}
	normalized := make(map[ResourceKind]int64, len(values))
	for kind, amount := range values {
		if amount > 0 {
			normalized[kind] = amount
		}
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizePositiveStringIntMap(values map[string]int) map[string]int {
	if len(values) == 0 {
		return nil
	}
	normalized := make(map[string]int, len(values))
	for key, amount := range values {
		key = strings.TrimSpace(key)
		if key != "" && amount > 0 {
			normalized[key] = amount
		}
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func setPolicyValue(current PolicyState, category string, value string) (PolicyState, error) {
	next := clonePolicyState(current)
	ensurePolicyStateForValue(&next)
	switch normalizePolicyCategory(category) {
	case PolicyCategoryTask:
		policy := TaskPolicy(value)
		if !validTaskPolicy(policy) {
			return PolicyState{}, fmt.Errorf("unsupported task policy: %s", value)
		}
		next.TaskPolicy = policy
	case PolicyCategoryResource:
		policy := ResourcePolicy(value)
		if !validResourcePolicy(policy) {
			return PolicyState{}, fmt.Errorf("unsupported resource policy: %s", value)
		}
		next.ResourcePolicy = policy
	case PolicyCategoryRecruitment:
		policy := RecruitmentPolicy(value)
		if !validRecruitmentPolicy(policy) {
			return PolicyState{}, fmt.Errorf("unsupported recruitment policy: %s", value)
		}
		next.RecruitmentPolicy = policy
	case PolicyCategoryCultivation:
		policy := CultivationPolicy(value)
		if !validCultivationPolicy(policy) {
			return PolicyState{}, fmt.Errorf("unsupported cultivation policy: %s", value)
		}
		next.CultivationPolicy = policy
	default:
		return PolicyState{}, fmt.Errorf("unsupported policy category: %s", category)
	}
	return next, nil
}

func ensurePolicyStateForValue(policy *PolicyState) {
	if policy == nil {
		return
	}
	if policy.TaskPolicy == "" {
		policy.TaskPolicy = TaskPolicyStable
	}
	if policy.ResourcePolicy == "" {
		policy.ResourcePolicy = ResourcePolicyOpenExchange
	}
	if policy.RecruitmentPolicy == "" {
		policy.RecruitmentPolicy = RecruitmentPolicyBroad
	}
	if policy.CultivationPolicy == "" {
		policy.CultivationPolicy = CultivationPolicyBalanced
	}
	if policy.CustomFlags == nil {
		policy.CustomFlags = map[string]bool{}
	}
}

func normalizePolicyCategory(category string) PolicyCategory {
	switch strings.TrimSpace(category) {
	case "task", "task_policy":
		return PolicyCategoryTask
	case "resource", "resource_policy":
		return PolicyCategoryResource
	case "recruitment", "recruitment_policy":
		return PolicyCategoryRecruitment
	case "cultivation", "cultivation_policy":
		return PolicyCategoryCultivation
	default:
		return PolicyCategory(category)
	}
}

func validTaskPolicy(policy TaskPolicy) bool {
	switch policy {
	case TaskPolicyStable, TaskPolicyRewardExternal, TaskPolicyProduction, TaskPolicyCombat, TaskPolicyClosedCultivation:
		return true
	default:
		return false
	}
}

func validResourcePolicy(policy ResourcePolicy) bool {
	switch policy {
	case ResourcePolicySaving, ResourcePolicyGenerous, ResourcePolicyPillLimited, ResourcePolicyOpenExchange, ResourcePolicyWarPreparation:
		return true
	default:
		return false
	}
}

func validRecruitmentPolicy(policy RecruitmentPolicy) bool {
	switch policy {
	case RecruitmentPolicyBroad, RecruitmentPolicySelective, RecruitmentPolicyWandering, RecruitmentPolicyAffiliated:
		return true
	default:
		return false
	}
}

func validCultivationPolicy(policy CultivationPolicy) bool {
	switch policy {
	case CultivationPolicyBalanced, CultivationPolicyClosedCultivation, CultivationPolicyBreakthroughSafe:
		return true
	default:
		return false
	}
}

func buildPolicyChangedPayload(state SectState, policies PolicyState) PolicyChangedPayload {
	projected := state.Clone()
	projected.Policies = clonePolicyState(policies)
	ensurePolicyState(&projected)
	applyPolicyEffectsToState(&projected)
	return PolicyChangedPayload{
		Policies:    clonePolicyState(projected.Policies),
		Tasks:       cloneTasks(projected.Tasks),
		Productions: cloneProductions(projected.Productions),
	}
}

func applyPolicyEffectsToState(state *SectState) {
	if state == nil {
		return
	}
	ensurePolicyState(state)
	applyTaskPolicyToTasks(state.Tasks, state.Policies.TaskPolicy)
	applyResourcePolicyToProductions(state.Productions, state.Policies.ResourcePolicy)
}

func applyTaskPolicyToTasks(tasks map[TaskID]TaskState, policy TaskPolicy) {
	for taskID, task := range tasks {
		if !strings.HasPrefix(string(taskID), "pool-") || task.Status != TaskStatusPublished {
			continue
		}
		task.Priority = taskPriorityForPolicy(task, policy)
		tasks[taskID] = task
	}
}

func taskPriorityForPolicy(task TaskState, policy TaskPolicy) int {
	base := defaultTaskPriority(task)
	switch policy {
	case TaskPolicyProduction:
		if task.Kind == "village_aid" || normalizeTaskType(task.Kind, task.Type) == TaskTypeProduction {
			return clampInt(base+25, 0, 100)
		}
		if task.Risk >= 60 {
			return clampInt(base-15, 0, 100)
		}
	case TaskPolicyCombat:
		if normalizeTaskType(task.Kind, task.Type) == TaskTypeCombat {
			return clampInt(base+25, 0, 100)
		}
		return clampInt(base-8, 0, 100)
	case TaskPolicyRewardExternal:
		if normalizeTaskType(task.Kind, task.Type) == TaskTypeExternal {
			return clampInt(base+28, 0, 100)
		}
		if task.ContributionReward >= 12 || task.ReputationReward > 0 || len(task.RelationReward) > 0 {
			return clampInt(base+18, 0, 100)
		}
	case TaskPolicyClosedCultivation:
		if task.Kind == "ancient_road_explore" {
			return clampInt(base+30, 0, 100)
		}
		if task.Risk >= 60 {
			return clampInt(base-20, 0, 100)
		}
	case TaskPolicyStable:
		if task.Risk >= 60 {
			return clampInt(base-20, 0, 100)
		}
		if task.Risk <= 12 {
			return clampInt(base+8, 0, 100)
		}
	}
	return clampInt(base, 0, 100)
}

func defaultTaskPriority(task TaskState) int {
	switch task.Kind {
	case "sect_patrol":
		return 60
	case "merchant_commission":
		return 48
	case "village_aid":
		return 52
	case "demon_scout":
		return 40
	case "ancient_road_explore":
		return 35
	case "combat_training":
		return 45
	default:
		return task.Priority
	}
}

func applyResourcePolicyToProductions(productions map[ProductionID]ProductionJob, policy ResourcePolicy) {
	for productionID, job := range productions {
		if productionClosed(job.Status) {
			continue
		}
		job.Priority = productionPriorityForJobPolicy(job, policy)
		productions[productionID] = job
	}
}

func productionPriorityForJobPolicy(job ProductionJob, policy ResourcePolicy) int {
	recipe, ok := defaultProductionRecipes()[job.RecipeID]
	if !ok {
		return clampInt(job.Priority, 0, 100)
	}
	return productionPriorityForPolicy(recipe, policy)
}

func productionPriorityForPolicy(recipe ProductionRecipe, policy ResourcePolicy) int {
	base := recipe.DefaultPriority
	switch policy {
	case ResourcePolicySaving:
		if recipe.InputCost[ResourceKindSpiritStone] > 0 {
			return clampInt(base-25, 0, 100)
		}
		if recipe.Kind == ProductionKindFarm {
			return clampInt(base+12, 0, 100)
		}
	case ResourcePolicyGenerous:
		if recipe.Kind == ProductionKindFarm {
			return clampInt(base+25, 0, 100)
		}
		return clampInt(base+8, 0, 100)
	case ResourcePolicyPillLimited:
		if recipe.RecipeID == RecipeID("farm_herb_mvp") {
			return clampInt(base+20, 0, 100)
		}
		if recipe.Kind == ProductionKindRefinement {
			return clampInt(base-10, 0, 100)
		}
	case ResourcePolicyWarPreparation:
		if recipe.Kind == ProductionKindMining || recipe.Kind == ProductionKindRefinement {
			return clampInt(base+25, 0, 100)
		}
		return clampInt(base-5, 0, 100)
	case ResourcePolicyOpenExchange:
		return base
	}
	return clampInt(base, 0, 100)
}

func recruitmentCandidateCountForPolicy(policy RecruitmentPolicy) int {
	switch policy {
	case RecruitmentPolicyBroad:
		return 5
	case RecruitmentPolicySelective:
		return 2
	case RecruitmentPolicyWandering, RecruitmentPolicyAffiliated:
		return 3
	default:
		return 3
	}
}

func recruitmentQualityBoostForPolicy(policy RecruitmentPolicy) int {
	switch policy {
	case RecruitmentPolicySelective:
		return 3
	case RecruitmentPolicyAffiliated:
		return 2
	case RecruitmentPolicyWandering:
		return 1
	default:
		return 0
	}
}

type appendDomainEventFunc func(DomainEvent) error

func (a *SectActor) appendMonthlySettlementEvents(
	simulated *SectState,
	versionCursor *Version,
	appendEvent appendDomainEventFunc,
) error {
	if simulated == nil {
		return nil
	}
	ensureMonthlyState(simulated)
	monthIndex := monthIndexForDay(simulated.Time.CalendarDay)
	if monthIndex <= 0 || simulated.Time.CalendarDay%30 != 0 || simulated.Monthly.LastSettledMonth >= monthIndex {
		return nil
	}

	recalculateContributionMetrics(simulated)
	summary := MonthlySettlementSummary{
		MonthIndex:           monthIndex,
		RedeemabilityRatio:   simulated.Contribution.RedeemabilityRatio,
		ContributionShortage: simulated.Contribution.RedeemabilityRatio < 0.8,
	}
	payrollPaid := map[DiscipleID]bool{}
	payrollDelayed := map[DiscipleID]bool{}
	dutyMissed := map[DiscipleID]bool{}

	for _, discipleID := range sortedDiscipleMapIDs(simulated.Disciples) {
		disciple := simulated.Disciples[discipleID]
		stipend := monthlyStipendFor(disciple)
		if stipend <= 0 {
			continue
		}
		currentArrears := simulated.Monthly.Payroll.Arrears[discipleID]
		if simulated.Resources.Stock[ResourceKindSpiritStone] >= stipend {
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeResourceChanged, ResourceChangedPayload{
				Changes: map[ResourceKind]int64{ResourceKindSpiritStone: -stipend},
				Reason:  fmt.Sprintf("monthly_stipend:%s:%d", discipleID, monthIndex),
			}, *versionCursor+1)); err != nil {
				return err
			}
			if err := appendEvent(a.newDomainEvent("", DomainEventTypePayrollPaid, PayrollSettlementPayload{
				MonthIndex:   monthIndex,
				DiscipleID:   discipleID,
				Amount:       stipend,
				ArrearsAfter: currentArrears,
			}, *versionCursor+1)); err != nil {
				return err
			}
			payrollPaid[discipleID] = true
			summary.StipendPaid += stipend
			summary.PayrollPaidCount++
			continue
		}

		nextArrears := currentArrears + 1
		if err := appendEvent(a.newDomainEvent("", DomainEventTypePayrollDelayed, PayrollSettlementPayload{
			MonthIndex:   monthIndex,
			DiscipleID:   discipleID,
			Amount:       stipend,
			ArrearsAfter: nextArrears,
		}, *versionCursor+1)); err != nil {
			return err
		}
		payrollDelayed[discipleID] = true
		summary.StipendDelayed += stipend
		summary.PayrollDelayedCount++
		summary.ResourceShortage = true
	}

	for _, discipleID := range sortedDiscipleMapIDs(simulated.Disciples) {
		disciple := simulated.Disciples[discipleID]
		requiredDays := monthlyDutyRequiredDays(disciple)
		completedDays := simulated.Monthly.Obligations.CompletedDays[discipleID]
		violationAdded := completedDays < requiredDays
		violationsAfter := simulated.Monthly.Obligations.Violations[discipleID]
		if violationAdded {
			violationsAfter++
			dutyMissed[discipleID] = true
			summary.DutyViolations++
		}
		summary.DutyRequiredDays += requiredDays
		summary.DutyCompletedDays += completedDays
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeMonthlyObligationChecked, MonthlyObligationCheckedPayload{
			MonthIndex:      monthIndex,
			DiscipleID:      discipleID,
			RequiredDays:    requiredDays,
			CompletedDays:   completedDays,
			ViolationAdded:  violationAdded,
			ViolationsAfter: violationsAfter,
		}, *versionCursor+1)); err != nil {
			return err
		}
	}

	for _, discipleID := range sortedDiscipleMapIDs(simulated.Disciples) {
		disciple := simulated.Disciples[discipleID]
		satisfactionDelta, loyaltyDelta := monthlyDiscipleMoodDelta(
			disciple,
			simulated.Contribution.RedeemabilityRatio,
			simulated.Policies.ResourcePolicy,
			payrollPaid[discipleID],
			payrollDelayed[discipleID],
			dutyMissed[discipleID],
			simulated.Monthly.Payroll.Arrears[discipleID],
		)

		if satisfactionDelta != 0 {
			next := simulated.Disciples[discipleID]
			next.Satisfaction = clampInt(next.Satisfaction+satisfactionDelta, 0, 100)
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeDiscipleSatisfactionChanged, DiscipleChangedPayload{Disciple: next}, *versionCursor+1)); err != nil {
				return err
			}
			summary.SatisfactionDeltaTotal += satisfactionDelta
		}
		if loyaltyDelta != 0 {
			next := simulated.Disciples[discipleID]
			next.Loyalty = clampInt(next.Loyalty+loyaltyDelta, 0, 100)
			if err := appendEvent(a.newDomainEvent("", DomainEventTypeDiscipleLoyaltyChanged, DiscipleChangedPayload{Disciple: next}, *versionCursor+1)); err != nil {
				return err
			}
			summary.LoyaltyDeltaTotal += loyaltyDelta
		}
	}

	nextMonthly := simulated.Monthly
	nextMonthly.LastSettledMonth = monthIndex
	nextMonthly.LastSettlement = summary
	nextMonthly.Obligations.MonthIndex = monthIndex + 1
	nextMonthly.Obligations.CompletedDays = map[DiscipleID]int{}
	nextMonthly.Obligations.RequiredDays = map[DiscipleID]int{}
	nextMonthly.Payroll.Arrears = cloneDiscipleIntMap(simulated.Monthly.Payroll.Arrears)
	nextMonthly.Obligations.Violations = cloneDiscipleIntMap(simulated.Monthly.Obligations.Violations)

	if err := appendEvent(a.newDomainEvent("", DomainEventTypeMonthAdvanced, MonthAdvancedPayload{
		MonthIndex:     monthIndex,
		NextMonthIndex: monthIndex + 1,
		Monthly:        nextMonthly,
		Summary:        summary,
	}, *versionCursor+1)); err != nil {
		return err
	}
	return nil
}

func (a *SectActor) appendInstitutionDailyEffectEvents(
	simulated *SectState,
	versionCursor *Version,
	appendEvent appendDomainEventFunc,
) error {
	if simulated == nil {
		return nil
	}
	ensureInstitutionState(simulated)
	medicine := simulated.Institutions.ByID[InstitutionIDMedicineHut]
	recovery := institutionRecoveryPower(*simulated)
	satisfactionSupport := institutionSatisfactionSupport(*simulated)
	formationRelief := formationPressureRelief(*simulated)
	if recovery <= 0 && satisfactionSupport <= 0 && formationRelief <= 0 {
		return nil
	}

	treatmentsRemaining := 0
	if medicine.Enabled && recovery > 0 {
		treatmentsRemaining = maxInt(1, medicine.Capacity)
	}

	for _, discipleID := range sortedDiscipleMapIDs(simulated.Disciples) {
		disciple := simulated.Disciples[discipleID]
		next := disciple
		treatedToday := false
		if recovery > 0 {
			needsTreatment := next.InjuryLevel > 0 || next.HP < next.MaxHP
			if needsTreatment && treatmentsRemaining > 0 {
				herbCost := int64(1)
				if next.InjuryLevel >= 2 {
					herbCost++
				}
				if simulated.Resources.Stock[ResourceKindHerb] >= herbCost {
					if err := appendEvent(a.newDomainEvent("", DomainEventTypeResourceChanged, ResourceChangedPayload{
						Changes: map[ResourceKind]int64{
							ResourceKindHerb: -herbCost,
						},
						Reason: fmt.Sprintf("institution_treatment:%s", discipleID),
					}, *versionCursor+1)); err != nil {
						return err
					}
					treatmentsRemaining--
					treatedToday = true
				}
			}
			if treatedToday && next.InjuryLevel > 0 {
				next.InjuryLevel = maxInt(0, next.InjuryLevel-recovery)
				next.Support.MedicalSupported = true
			}
			if treatedToday && next.HP < next.MaxHP {
				next.HP += int64(recovery * 10)
				if next.HP > next.MaxHP {
					next.HP = next.MaxHP
				}
				next.Support.MedicalSupported = true
			}
			if treatedToday && next.Pressure > 0 {
				next.Pressure = maxInt(0, next.Pressure-recovery*2)
			}
		}
		if satisfactionSupport > 0 {
			next.Satisfaction = clampInt(next.Satisfaction+satisfactionSupport, 0, 100)
			next.Support.FoodSatisfied = true
			next.Support.HousingSatisfied = true
		}
		if formationRelief > 0 && next.Pressure > 0 {
			next.Pressure = maxInt(0, next.Pressure-formationRelief)
		}
		if reflect.DeepEqual(next, disciple) {
			continue
		}
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeDiscipleSatisfactionChanged, DiscipleChangedPayload{Disciple: next}, *versionCursor+1)); err != nil {
			return err
		}
	}
	return nil
}

func (a *SectActor) appendSectOrderEvents(
	simulated *SectState,
	versionCursor *Version,
	appendEvent appendDomainEventFunc,
) error {
	if simulated == nil {
		return nil
	}
	ensureSectOrderState(simulated)
	next := buildSectOrderProjection(*simulated, *versionCursor+1)
	current := simulated.Order
	if reflect.DeepEqual(current, next) {
		return nil
	}
	return appendEvent(a.newDomainEvent("", DomainEventTypeOrderChanged, OrderChangedPayload{
		Order: next,
	}, *versionCursor+1))
}

func (a *SectActor) appendMonthlyAssessmentEvents(
	simulated *SectState,
	versionCursor *Version,
	appendEvent appendDomainEventFunc,
) error {
	if simulated == nil {
		return nil
	}
	ensureMonthlyAssessmentState(simulated)
	monthIndex := simulated.Monthly.LastSettlement.MonthIndex
	if monthIndex <= 0 || simulated.MonthlyAssessment.LastMonthIndex >= monthIndex {
		return nil
	}
	championID, championScore := monthlyAssessmentChampion(*simulated)
	if championID == "" {
		return nil
	}
	champion := simulated.Disciples[championID]
	contributionReward := int64(6 + championScore/20)
	reputationReward := 1
	if championScore >= 85 {
		reputationReward = 2
	}
	result := MonthlyAssessmentResult{
		MonthIndex:         monthIndex,
		ChampionDiscipleID: &championID,
		ChampionName:       champion.Name,
		Score:              championScore,
		RewardContribution: contributionReward,
		RewardReputation:   reputationReward,
		PromotionMomentum:  clampInt(championScore/20, 1, 5),
		Summary:            fmt.Sprintf("第 %d 月小比由 %s 领先，宗门为其记功并提振门内士气。", monthIndex, champion.Name),
		ResolvedAtVersion:  *versionCursor + 1,
	}
	if contributionReward > 0 {
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeContributionEarned, ContributionChangedPayload{
			DiscipleID: championID,
			Delta:      contributionReward,
			Reason:     fmt.Sprintf("monthly_assessment:%d", monthIndex),
			Quantity:   1,
		}, *versionCursor+1)); err != nil {
			return err
		}
	}
	if reputationReward != 0 {
		nextMeta := simulated.Meta
		nextMeta.Reputation = maxInt(0, nextMeta.Reputation+reputationReward)
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectMetaChanged, SectMetaChangedPayload{
			Meta:   nextMeta,
			Reason: fmt.Sprintf("monthly_assessment:%d", monthIndex),
		}, *versionCursor+1)); err != nil {
			return err
		}
	}
	nextDisciple := simulated.Disciples[championID]
	nextDisciple.Satisfaction = clampInt(nextDisciple.Satisfaction+2, 0, 100)
	if err := appendEvent(a.newDomainEvent("", DomainEventTypeDiscipleSatisfactionChanged, DiscipleChangedPayload{
		Disciple: nextDisciple,
	}, *versionCursor+1)); err != nil {
		return err
	}
	nextAssessment := cloneMonthlyAssessmentState(simulated.MonthlyAssessment)
	nextAssessment.LastMonthIndex = monthIndex
	nextAssessment.Latest = &result
	nextAssessment.History = append(nextAssessment.History, result)
	if len(nextAssessment.History) > 6 {
		nextAssessment.History = cloneMonthlyAssessmentResults(nextAssessment.History[len(nextAssessment.History)-6:])
	}
	return appendEvent(a.newDomainEvent("", DomainEventTypeMonthlyAssessmentResolved, MonthlyAssessmentResolvedPayload{
		Assessment: nextAssessment,
		Result:     result,
	}, *versionCursor+1))
}

func monthlyAssessmentChampion(state SectState) (DiscipleID, int) {
	bestID := DiscipleID("")
	bestScore := -1
	for _, discipleID := range sortedDiscipleMapIDs(state.Disciples) {
		disciple := state.Disciples[discipleID]
		score := monthlyAssessmentScore(state, disciple)
		if score > bestScore {
			bestID = discipleID
			bestScore = score
		}
	}
	return bestID, maxInt(bestScore, 0)
}

func monthlyAssessmentScore(state SectState, disciple DiscipleState) int {
	score := realmRankValue(disciple.Realm.Stage) * 18
	score += disciple.Loyalty / 4
	score += disciple.Satisfaction / 4
	score += int(state.Contribution.Accounts[disciple.DiscipleID].EarnedTotal / 5)
	score += monthlyAssessmentCompletedTaskBonus(state, disciple.DiscipleID)
	score -= disciple.InjuryLevel * 12
	score -= disciple.Pressure / 5
	score += len(disciple.Relationship) * 2
	return clampInt(score, 0, 100)
}

func monthlyAssessmentCompletedTaskBonus(state SectState, discipleID DiscipleID) int {
	total := 0
	for _, taskID := range sortedTaskIDs(state.Tasks) {
		task := state.Tasks[taskID]
		if task.Status != TaskStatusCompleted {
			continue
		}
		for _, assigned := range task.AssignedDiscipleIDs {
			if assigned != discipleID {
				continue
			}
			total += 6
			if normalizeTaskType(task.Kind, task.Type) == TaskTypeCombat {
				total += 3
			}
			break
		}
	}
	return total
}

func ensureMonthlyState(state *SectState) {
	if state == nil {
		return
	}
	if state.Monthly.Payroll.Arrears == nil {
		state.Monthly.Payroll.Arrears = map[DiscipleID]int{}
	}
	if state.Monthly.Obligations.MonthIndex == 0 {
		state.Monthly.Obligations.MonthIndex = currentMonthIndexForDay(state.Time.CalendarDay)
	}
	if state.Monthly.Obligations.CompletedDays == nil {
		state.Monthly.Obligations.CompletedDays = map[DiscipleID]int{}
	}
	if state.Monthly.Obligations.RequiredDays == nil {
		state.Monthly.Obligations.RequiredDays = map[DiscipleID]int{}
	}
	if state.Monthly.Obligations.Violations == nil {
		state.Monthly.Obligations.Violations = map[DiscipleID]int{}
	}
	for discipleID := range state.Disciples {
		if _, ok := state.Monthly.Payroll.Arrears[discipleID]; !ok {
			state.Monthly.Payroll.Arrears[discipleID] = 0
		}
		if _, ok := state.Monthly.Obligations.CompletedDays[discipleID]; !ok {
			state.Monthly.Obligations.CompletedDays[discipleID] = 0
		}
		if _, ok := state.Monthly.Obligations.RequiredDays[discipleID]; !ok {
			state.Monthly.Obligations.RequiredDays[discipleID] = monthlyDutyRequiredDays(state.Disciples[discipleID])
		}
		if _, ok := state.Monthly.Obligations.Violations[discipleID]; !ok {
			state.Monthly.Obligations.Violations[discipleID] = 0
		}
	}
}

func applyMonthlyDutyProgress(state *SectState, task TaskState) {
	if state == nil {
		return
	}
	ensureMonthlyState(state)
	for _, discipleID := range sortedDiscipleIDs(task.AssignedDiscipleIDs) {
		if _, exists := state.Disciples[discipleID]; !exists {
			continue
		}
		state.Monthly.Obligations.CompletedDays[discipleID]++
		state.Monthly.Obligations.RequiredDays[discipleID] = monthlyDutyRequiredDays(state.Disciples[discipleID])
	}
}

func nextDayTime(current SectTimeState) SectTimeState {
	next := current
	next.GameTick++
	next.CalendarDay++
	next.DayTick = 0
	if next.CalendarDay > 0 {
		next.SeasonIndex = ((next.CalendarDay - 1) / 30) % 4
	}
	return next
}

func monthIndexForDay(calendarDay int32) int64 {
	if calendarDay <= 0 {
		return 0
	}
	return int64((calendarDay-1)/30) + 1
}

func currentMonthIndexForDay(calendarDay int32) int64 {
	if calendarDay <= 0 {
		return 1
	}
	return monthIndexForDay(calendarDay)
}

func monthlyStipendFor(disciple DiscipleState) int64 {
	base := int64(4)
	switch disciple.Identity {
	case IdentityRankOuter:
		base = 4
	case IdentityRankInner:
		base = 12
	default:
		base = 4
	}
	return base + int64(realmRankValue(disciple.Realm.Stage))*2
}

func monthlyDutyRequiredDays(disciple DiscipleState) int {
	required := 4
	if disciple.Identity == IdentityRankInner {
		required = 3
	}
	if disciple.InjuryLevel >= 3 || disciple.HP <= 0 {
		return 0
	}
	if disciple.Realm.Stage != RealmMortal {
		required++
	}
	return required
}

func monthlyDiscipleMoodDelta(
	disciple DiscipleState,
	redeemabilityRatio float64,
	resourcePolicy ResourcePolicy,
	payrollPaid bool,
	payrollDelayed bool,
	dutyMissed bool,
	arrears int,
) (int, int) {
	satisfactionDelta := 0
	loyaltyDelta := 0
	switch {
	case redeemabilityRatio >= 1.2:
		satisfactionDelta += 2
	case redeemabilityRatio >= 0.8:
	case redeemabilityRatio >= 0.5:
		satisfactionDelta -= 3
	default:
		satisfactionDelta -= 7
		loyaltyDelta -= 3
	}
	if payrollPaid {
		loyaltyDelta++
	}
	if payrollDelayed {
		satisfactionDelta -= 8
		loyaltyDelta -= 5
	}
	if arrears >= 2 {
		satisfactionDelta -= 2
		loyaltyDelta -= 2
	}
	if dutyMissed {
		satisfactionDelta -= 5
		loyaltyDelta -= 3
	} else {
		satisfactionDelta++
	}
	if disciple.Pressure >= 60 {
		satisfactionDelta -= 2
	}
	policySatisfactionDelta, policyLoyaltyDelta := monthlyMoodPolicyDelta(resourcePolicy, payrollPaid, payrollDelayed, dutyMissed, redeemabilityRatio)
	satisfactionDelta += policySatisfactionDelta
	loyaltyDelta += policyLoyaltyDelta
	memorySatisfactionDelta, memoryLoyaltyDelta := discipleMemoryMoodDelta(disciple)
	satisfactionDelta += memorySatisfactionDelta
	loyaltyDelta += memoryLoyaltyDelta
	if disciple.Satisfaction+satisfactionDelta < 40 {
		loyaltyDelta -= 2
	}
	return satisfactionDelta, loyaltyDelta
}

func monthlyMoodPolicyDelta(
	policy ResourcePolicy,
	payrollPaid bool,
	payrollDelayed bool,
	dutyMissed bool,
	redeemabilityRatio float64,
) (int, int) {
	switch policy {
	case ResourcePolicySaving:
		satisfaction := -2
		loyalty := 0
		if payrollDelayed {
			satisfaction -= 2
			loyalty--
		}
		return satisfaction, loyalty
	case ResourcePolicyGenerous:
		satisfaction := 2
		loyalty := 1
		if payrollDelayed {
			satisfaction += 2
		}
		if !dutyMissed && payrollPaid {
			satisfaction++
		}
		return satisfaction, loyalty
	case ResourcePolicyOpenExchange:
		if redeemabilityRatio >= 1 {
			return 1, 0
		}
	case ResourcePolicyWarPreparation:
		if dutyMissed {
			return -1, -1
		}
	case ResourcePolicyPillLimited:
		return -1, 0
	}
	return 0, 0
}

func discipleMemoryMoodDelta(disciple DiscipleState) (int, int) {
	relationship := makeStringSet(disciple.Relationship)
	emotion := makeStringSet(disciple.Emotion)
	satisfactionDelta := 0
	loyaltyDelta := 0
	if relationship["sect_grateful"] {
		satisfactionDelta += 2
		loyaltyDelta += 3
	}
	if relationship["sect_resentful"] {
		satisfactionDelta -= 4
		loyaltyDelta -= 5
	}
	if relationship["duty_trusted"] {
		loyaltyDelta++
	}
	if relationship["battle_proven"] {
		loyaltyDelta++
	}
	if emotion["wounded"] {
		satisfactionDelta -= 2
	}
	if emotion["pressured"] {
		satisfactionDelta--
		loyaltyDelta--
	}
	if emotion["grateful"] {
		satisfactionDelta++
	}
	if emotion["resentful"] {
		satisfactionDelta--
		loyaltyDelta--
	}
	return satisfactionDelta, loyaltyDelta
}

func discipleTaskWillingnessModifier(disciple DiscipleState, task TaskState) int {
	relationship := makeStringSet(disciple.Relationship)
	emotion := makeStringSet(disciple.Emotion)
	modifier := 0
	taskType := normalizeTaskType(task.Kind, task.Type)
	if relationship["sect_resentful"] {
		modifier -= 6
	}
	if relationship["duty_trusted"] {
		modifier += 4
	}
	if emotion["pressured"] {
		modifier -= 4
	}
	if taskType == TaskTypeCombat || taskType == TaskTypeExternal {
		if relationship["battle_proven"] {
			modifier += 10
		}
		if relationship["wound_shadow"] {
			modifier -= 8
		}
	}
	if relationship["promotion_driven"] && (taskType == TaskTypeExternal || taskType == TaskTypeExplore) {
		modifier += 3
	}
	if emotion["ambitious"] && task.Risk <= 60 {
		modifier += 2
	}
	return modifier
}

func disciplePromotionMemoryModifier(disciple DiscipleState) (int, []string) {
	relationship := makeStringSet(disciple.Relationship)
	emotion := makeStringSet(disciple.Emotion)
	score := 0
	reasons := []string{}
	if relationship["sect_grateful"] {
		score += 6
	}
	if relationship["duty_trusted"] {
		score += 4
	}
	if relationship["promotion_driven"] {
		score += 5
	}
	if relationship["sect_resentful"] {
		score -= 10
		reasons = append(reasons, "memory_resentment")
	}
	if relationship["wound_shadow"] {
		score -= 6
		reasons = append(reasons, "memory_wound_shadow")
	}
	if emotion["ambitious"] {
		score += 2
	}
	if emotion["resentful"] {
		score -= 4
	}
	return score, sortedUniqueStrings(reasons)
}

func applyContributionDelta(state *SectState, payload ContributionChangedPayload) error {
	ensureContributionAccounts(state)
	account := state.Contribution.Accounts[payload.DiscipleID]
	nextBalance := account.Balance + payload.Delta
	if nextBalance < 0 {
		return fmt.Errorf("%w: disciple %s balance would become negative", errContributionNotEnough, payload.DiscipleID)
	}
	account.DiscipleID = payload.DiscipleID
	account.Balance = nextBalance
	if payload.Delta >= 0 {
		account.EarnedTotal += payload.Delta
	} else {
		account.SpentTotal += -payload.Delta
		if payload.ExchangeItemID != nil && payload.Quantity > 0 {
			if state.Contribution.MonthlyPurchases[payload.DiscipleID] == nil {
				state.Contribution.MonthlyPurchases[payload.DiscipleID] = map[ExchangeItemID]int64{}
			}
			state.Contribution.MonthlyPurchases[payload.DiscipleID][*payload.ExchangeItemID] += payload.Quantity
		}
	}
	state.Contribution.Accounts[payload.DiscipleID] = account
	recalculateContributionMetrics(state)
	return nil
}

func recalculateContributionMetrics(state *SectState) {
	if state == nil {
		return
	}
	ensureContributionAccounts(state)
	var outstanding int64
	for _, account := range state.Contribution.Accounts {
		outstanding += account.Balance
	}
	var treasuryValue int64
	for _, rule := range state.Contribution.TreasuryRules {
		if !rule.Enabled || rule.ItemKind != ExchangeItemKind("resource") || rule.ContributionCost <= 0 {
			continue
		}
		available := state.Resources.Stock[ResourceKind(rule.ItemRef)]
		if rule.StockLimit >= 0 && available > rule.StockLimit {
			available = rule.StockLimit
		}
		if available < 0 {
			available = 0
		}
		treasuryValue += available * rule.ContributionCost
	}
	state.Contribution.OutstandingContribution = outstanding
	state.Contribution.TreasuryValue = treasuryValue
	if outstanding <= 0 {
		state.Contribution.RedeemabilityRatio = 1
		return
	}
	state.Contribution.RedeemabilityRatio = float64(treasuryValue) / float64(outstanding)
}

func nextPlayerTaskID(tasks map[TaskID]TaskState) TaskID {
	for index := 1; ; index++ {
		taskID := TaskID(fmt.Sprintf("task-%d", index))
		if _, exists := tasks[taskID]; !exists {
			return taskID
		}
	}
}

func normalizeMaxAssignees(maxAssignees int) int {
	if maxAssignees <= 0 {
		return 1
	}
	return maxAssignees
}

func normalizeTaskType(kind string, taskType TaskType) TaskType {
	if taskType != "" {
		return taskType
	}
	switch strings.TrimSpace(kind) {
	case "merchant_commission", "village_aid", "herb_delivery", "grain_allocation":
		return TaskTypeExternal
	case "ancient_road_explore":
		return TaskTypeExplore
	case "sect_patrol", "demon_scout", "combat_training":
		return TaskTypeCombat
	case "formation_refine_support", "archive_sorting", "library_copying":
		return TaskTypeProduction
	default:
		return TaskTypeInternal
	}
}

func normalizeTaskGrade(risk int, grade TaskGrade) TaskGrade {
	if grade != "" {
		return grade
	}
	switch {
	case risk >= 90:
		return TaskGradeYi
	case risk >= 50:
		return TaskGradeBing
	default:
		return TaskGradeDing
	}
}

func normalizeMinIdentity(identity IdentityRank) IdentityRank {
	if identity == "" {
		return IdentityRankOuter
	}
	return identity
}

func normalizePromotionTarget(current IdentityRank, target IdentityRank, targetSnake IdentityRank) (IdentityRank, error) {
	if target == "" {
		target = targetSnake
	}
	if target == "" {
		next, ok := nextPromotionRank(current)
		if !ok {
			return "", fmt.Errorf("%w: disciple identity %s has no configured promotion path", errTaskRequirementNotMet, current)
		}
		return next, nil
	}
	if !legalPromotionPath(current, target) {
		return "", fmt.Errorf("%w: cannot promote %s to %s", errTaskRequirementNotMet, current, target)
	}
	return target, nil
}

func nextPromotionRank(current IdentityRank) (IdentityRank, bool) {
	switch current {
	case IdentityRankOuter:
		return IdentityRankInner, true
	default:
		return "", false
	}
}

func normalizeMinRealm(realm RealmStage) RealmStage {
	if realm == "" {
		return RealmMortal
	}
	return realm
}

func normalizeAssignDiscipleIDs(payload AssignDiscipleTaskPayload) []DiscipleID {
	seen := map[DiscipleID]bool{}
	ids := make([]DiscipleID, 0, len(payload.DiscipleIDs)+1)
	if payload.DiscipleID != "" {
		seen[payload.DiscipleID] = true
		ids = append(ids, payload.DiscipleID)
	}
	for _, discipleID := range payload.DiscipleIDs {
		if discipleID == "" || seen[discipleID] {
			continue
		}
		seen[discipleID] = true
		ids = append(ids, discipleID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func buildTaskDispatchProjections(state SectState) map[TaskID]TaskDispatchProjection {
	projections := map[TaskID]TaskDispatchProjection{}
	for taskID, task := range state.Tasks {
		if task.Status != TaskStatusPublished {
			continue
		}
		projection := TaskDispatchProjection{TaskID: taskID}
		if len(task.DispatchCost) > 0 && !CanAfford(state.Resources, ResourceCost(task.DispatchCost)) {
			projection.BlockedReason = "资源不足"
			projections[taskID] = projection
			continue
		}
		discipleIDs := recommendTaskDiscipleIDs(state, task)
		if len(discipleIDs) == 0 {
			projection.BlockedReason = "无可派遣弟子"
			projections[taskID] = projection
			continue
		}
		projectedTask := task
		projectedTask.AssignedDiscipleIDs = discipleIDs
		projection.RecommendedDiscipleIDs = discipleIDs
		projection.RecommendedSuccessRate = taskSuccessRate(state, projectedTask)
		projections[taskID] = projection
	}
	return projections
}

func recommendTaskDiscipleIDs(state SectState, task TaskState) []DiscipleID {
	type candidate struct {
		discipleID DiscipleID
		score      int
	}
	candidates := []candidate{}
	for _, discipleID := range sortedDiscipleMapIDs(state.Disciples) {
		disciple := state.Disciples[discipleID]
		if disciple.AssignmentKind != DiscipleAssignmentIdle || disciple.AssignmentTask != nil {
			continue
		}
		if !discipleMeetsTaskRequirements(disciple, task) {
			continue
		}
		candidates = append(candidates, candidate{
			discipleID: discipleID,
			score:      taskRecommendationScore(disciple, task),
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].discipleID < candidates[j].discipleID
	})
	limit := taskMaxAssignees(task)
	if limit <= 0 {
		limit = 1
	}
	if len(candidates) < limit {
		limit = len(candidates)
	}
	ids := make([]DiscipleID, 0, limit)
	for _, candidate := range candidates[:limit] {
		ids = append(ids, candidate.discipleID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func taskRecommendationScore(disciple DiscipleState, task TaskState) int {
	score := realmRankValue(disciple.Realm.Stage)*8 + disciple.Loyalty/5 - disciple.Pressure/10 - disciple.InjuryLevel*10
	switch normalizeTaskType(task.Kind, task.Type) {
	case TaskTypeCombat:
		score += disciple.Aptitude.Physique*3 + disciple.Aptitude.Mind + disciple.Aptitude.Luck
	case TaskTypeExplore:
		score += disciple.Aptitude.Luck*3 + disciple.Aptitude.Comprehension*2 + disciple.Aptitude.SpiritRoot
	case TaskTypeExternal:
		score += disciple.Aptitude.Mind*3 + disciple.Aptitude.Comprehension*2 + disciple.Aptitude.Luck
	case TaskTypeProduction:
		score += disciple.Aptitude.SpiritRoot*3 + disciple.Aptitude.Mind + disciple.Aptitude.Comprehension
	default:
		score += disciple.Aptitude.SpiritRoot + disciple.Aptitude.Comprehension + disciple.Aptitude.Physique + disciple.Aptitude.Mind + disciple.Aptitude.Luck
	}
	score += discipleTaskWillingnessModifier(disciple, task)
	return score
}

func taskMaxAssignees(task TaskState) int {
	return normalizeMaxAssignees(task.MaxAssignees)
}

func discipleMeetsTaskRequirements(disciple DiscipleState, task TaskState) bool {
	if disciple.HP <= 0 || disciple.InjuryLevel >= 3 {
		return false
	}
	if identityRankValue(disciple.Identity) < identityRankValue(normalizeMinIdentity(task.MinIdentity)) {
		return false
	}
	if realmRankValue(disciple.Realm.Stage) < realmRankValue(normalizeMinRealm(task.MinRealm)) {
		return false
	}
	return disciple.Aptitude.SpiritRoot >= task.RequiredAptitude.SpiritRoot &&
		disciple.Aptitude.Comprehension >= task.RequiredAptitude.Comprehension &&
		disciple.Aptitude.Physique >= task.RequiredAptitude.Physique &&
		disciple.Aptitude.Mind >= task.RequiredAptitude.Mind &&
		disciple.Aptitude.Luck >= task.RequiredAptitude.Luck
}

func taskTeamCapability(state SectState, task TaskState) int {
	capability := 0
	for _, discipleID := range task.AssignedDiscipleIDs {
		disciple := state.Disciples[discipleID]
		capability += disciple.Aptitude.SpiritRoot + disciple.Aptitude.Comprehension + disciple.Aptitude.Physique + disciple.Aptitude.Mind + disciple.Aptitude.Luck + 10
	}
	capability += institutionTaskCapabilityBonus(state)
	return capability
}

func taskSuccessRate(state SectState, task TaskState) int {
	score := 50
	score += taskTeamAbilityScore(state, task)
	score += taskTeamRealmBonus(state, task)
	score += taskTeamCooperationBonus(state, task)
	score += taskEquipmentBonus(state, task)
	score += taskPolicySuccessModifier(state, task)
	score += taskBuildingSupportBonus(state, task)
	score -= taskDifficultyPenalty(task)
	return clampInt(score, 5, 100)
}

func taskTeamAbilityScore(state SectState, task TaskState) int {
	if len(task.AssignedDiscipleIDs) == 0 {
		return 0
	}
	total := 0
	for _, discipleID := range task.AssignedDiscipleIDs {
		disciple := state.Disciples[discipleID]
		switch normalizeTaskType(task.Kind, task.Type) {
		case TaskTypeCombat:
			total += disciple.Aptitude.Physique*2 + disciple.Aptitude.Mind + disciple.Aptitude.Luck
		case TaskTypeExplore:
			total += disciple.Aptitude.Luck*2 + disciple.Aptitude.Comprehension + disciple.Aptitude.SpiritRoot
		case TaskTypeExternal:
			total += disciple.Aptitude.Mind*2 + disciple.Aptitude.Comprehension + disciple.Aptitude.Luck
		case TaskTypeProduction:
			total += disciple.Aptitude.SpiritRoot*2 + disciple.Aptitude.Mind + disciple.Aptitude.Comprehension
		default:
			total += disciple.Aptitude.SpiritRoot + disciple.Aptitude.Comprehension + disciple.Aptitude.Physique + disciple.Aptitude.Mind + disciple.Aptitude.Luck
		}
	}
	return clampInt(total/len(task.AssignedDiscipleIDs), 0, 45)
}

func taskTeamRealmBonus(state SectState, task TaskState) int {
	bonus := 0
	for _, discipleID := range task.AssignedDiscipleIDs {
		bonus += realmRankValue(state.Disciples[discipleID].Realm.Stage) * 4
	}
	return clampInt(bonus, 0, 20)
}

func taskTeamCooperationBonus(state SectState, task TaskState) int {
	count := len(task.AssignedDiscipleIDs)
	if count <= 1 {
		return 0
	}
	loyaltyTotal := 0
	for _, discipleID := range task.AssignedDiscipleIDs {
		loyaltyTotal += state.Disciples[discipleID].Loyalty
	}
	return clampInt(6+loyaltyTotal/(count*20), 0, 14)
}

func taskEquipmentBonus(state SectState, task TaskState) int {
	bonus := taskArtifactEquipmentBonus(state, task)
	switch normalizeTaskType(task.Kind, task.Type) {
	case TaskTypeCombat:
		if state.Resources.Stock[ResourceKindBeastMat] > 0 || inventoryPillQuantity(state, PillCalmMind) > 0 {
			bonus += 5
		}
	case TaskTypeExplore:
		if state.Resources.Stock[ResourceKindFormationMat] > 0 {
			bonus += 5
		}
	case TaskTypeExternal:
		if state.Resources.Stock[ResourceKindSpiritStone] >= 10 {
			bonus += 4
		}
	case TaskTypeProduction:
		if state.Resources.Stock[ResourceKindOre] > 0 || state.Resources.Stock[ResourceKindHerb] > 0 {
			bonus += 4
		}
	}
	return clampInt(bonus, 0, 24)
}

func taskArtifactEquipmentBonus(state SectState, task TaskState) int {
	if len(task.AssignedDiscipleIDs) == 0 || len(state.Inventory.Artifacts) == 0 {
		return 0
	}
	taskType := normalizeTaskType(task.Kind, task.Type)
	total := 0
	for _, discipleID := range task.AssignedDiscipleIDs {
		disciple := state.Disciples[discipleID]
		for _, itemID := range equippedArtifactIDs(disciple.Equipment) {
			artifact, ok := state.Inventory.Artifacts[itemID]
			if !ok || artifact.Durability <= 0 || artifact.BoundDiscipleID != discipleID {
				continue
			}
			switch taskType {
			case TaskTypeCombat:
				total += artifact.Stats["combat"] + artifact.Stats["injury_mitigation"]
			case TaskTypeExplore:
				total += artifact.Stats["exploration"] + artifact.Stats["combat"]/2
			case TaskTypeExternal:
				total += artifact.Stats["combat"]/2 + artifact.Stats["exploration"]/2
			case TaskTypeProduction:
				total += artifact.Stats["production"]
			default:
				total += artifact.Stats["combat"]/3 + artifact.Stats["production"]/3 + artifact.Stats["cultivation"]/3
			}
		}
	}
	return clampInt(total, 0, 18)
}

func equippedArtifactIDs(equipment EquipmentState) []ItemID {
	itemIDs := []ItemID{}
	if equipment.Weapon != "" {
		itemIDs = append(itemIDs, equipment.Weapon)
	}
	if equipment.Robe != "" {
		itemIDs = append(itemIDs, equipment.Robe)
	}
	if equipment.Tool != "" {
		itemIDs = append(itemIDs, equipment.Tool)
	}
	if equipment.Special != "" {
		itemIDs = append(itemIDs, equipment.Special)
	}
	return itemIDs
}

func discipleArtifactStat(state SectState, disciple DiscipleState, stat string) int {
	total := 0
	for _, itemID := range equippedArtifactIDs(disciple.Equipment) {
		artifact, ok := state.Inventory.Artifacts[itemID]
		if !ok || artifact.Durability <= 0 || artifact.BoundDiscipleID != disciple.DiscipleID {
			continue
		}
		total += artifact.Stats[stat]
	}
	return total
}

func taskTeamArtifactMitigation(state SectState, task TaskState) int {
	if len(task.AssignedDiscipleIDs) == 0 {
		return 0
	}
	total := 0
	for _, discipleID := range task.AssignedDiscipleIDs {
		total += discipleArtifactStat(state, state.Disciples[discipleID], "injury_mitigation")
	}
	return clampInt(total/len(task.AssignedDiscipleIDs), 0, 2)
}

func taskPolicySuccessModifier(state SectState, task TaskState) int {
	switch state.Policies.TaskPolicy {
	case TaskPolicyRewardExternal:
		if normalizeTaskType(task.Kind, task.Type) == TaskTypeExternal {
			return 12
		}
	case TaskPolicyCombat:
		if normalizeTaskType(task.Kind, task.Type) == TaskTypeCombat {
			return 12
		}
	case TaskPolicyProduction:
		if normalizeTaskType(task.Kind, task.Type) == TaskTypeProduction {
			return 10
		}
		if normalizeTaskType(task.Kind, task.Type) == TaskTypeExplore {
			return -4
		}
	case TaskPolicyClosedCultivation:
		if task.Risk >= 40 {
			return -8
		}
	case TaskPolicyStable:
		if task.Risk >= 60 {
			return -10
		}
	}
	return 0
}

func taskBuildingSupportBonus(state SectState, task TaskState) int {
	bonus := institutionTaskCapabilityBonus(state) / 2
	gate := state.Institutions.ByID[InstitutionIDGate]
	if gate.Enabled && (normalizeTaskType(task.Kind, task.Type) == TaskTypeExternal || normalizeTaskType(task.Kind, task.Type) == TaskTypeCombat) {
		bonus += gate.ManagerEffect.EfficiencyBonus / 3
		if len(gate.GatePolicy.GuardDiscipleIDs) > 0 {
			bonus += 3
		}
	}
	bonus += formationDefenseSupportBonus(state, task)
	return clampInt(bonus, 0, 18)
}

func formationDefenseSupportBonus(state SectState, task TaskState) int {
	taskType := normalizeTaskType(task.Kind, task.Type)
	if taskType != TaskTypeExternal && taskType != TaskTypeCombat {
		return 0
	}
	definition, err := formationDefinitionFor(FormationKindDefense)
	if err != nil {
		return 0
	}
	return clampInt((formationBonusByKind(state, FormationKindDefense)+definition.DefenseMitigation)/10, 0, 6)
}

func formationDefenseRiskMitigation(state SectState, task TaskState) int {
	taskType := normalizeTaskType(task.Kind, task.Type)
	if taskType != TaskTypeExternal && taskType != TaskTypeCombat {
		return 0
	}
	definition, err := formationDefinitionFor(FormationKindDefense)
	if err != nil {
		return 0
	}
	return clampInt((formationBonusByKind(state, FormationKindDefense)+definition.DefenseMitigation)/16, 0, 2)
}

func taskDifficultyPenalty(task TaskState) int {
	penalty := task.Risk
	switch normalizeTaskGrade(task.Risk, task.Grade) {
	case TaskGradeBing:
		penalty += 8
	case TaskGradeYi:
		penalty += 16
	case TaskGradeJia:
		penalty += 24
	case TaskGradeSpecial:
		penalty += 32
	}
	penalty += (task.RequiredAptitude.SpiritRoot + task.RequiredAptitude.Comprehension + task.RequiredAptitude.Physique + task.RequiredAptitude.Mind + task.RequiredAptitude.Luck) / 4
	return penalty
}

func taskEvaluationForRate(rate int) TaskEvaluation {
	switch {
	case rate >= 85:
		return TaskEvaluationExcellent
	case rate >= 70:
		return TaskEvaluationGood
	case rate >= 45:
		return TaskEvaluationNormal
	case rate >= 30:
		return TaskEvaluationPoor
	default:
		return TaskEvaluationFailed
	}
}

func (a *SectActor) appendTaskRiskConsequenceEvents(
	simulated *SectState,
	versionCursor *Version,
	appendEvent appendDomainEventFunc,
	task TaskState,
	successRate int,
	failed bool,
) error {
	if simulated == nil {
		return nil
	}
	severity := maxInt(1, taskConsequenceSeverity(task, successRate, failed)-taskTeamArtifactMitigation(*simulated, task)-formationDefenseRiskMitigation(*simulated, task))
	for _, discipleID := range sortedDiscipleIDs(task.AssignedDiscipleIDs) {
		current := simulated.Disciples[discipleID]
		next := current
		next.InjuryLevel = clampInt(next.InjuryLevel+severity, 0, 5)
		next.Pressure = clampInt(next.Pressure+8+severity*8, 0, 100)
		next.Satisfaction = clampInt(next.Satisfaction-severity*3, 0, 100)
		next.Loyalty = clampInt(next.Loyalty-severity*2, 0, 100)
		next.HP = maxInt64(0, next.HP-int64(8+severity*10))
		next.WorkTarget = DiscipleWorkTargetState{
			TaskID:      &task.TaskID,
			Description: fmt.Sprintf("task_risk:%s", task.Kind),
		}
		if reflect.DeepEqual(next, current) {
			continue
		}
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeDiscipleSatisfactionChanged, DiscipleChangedPayload{Disciple: next}, *versionCursor+1)); err != nil {
			return err
		}
	}

	loss := taskFailureResourceLoss(task, severity, failed)
	if len(loss) > 0 && resourceDeltaAffordable(simulated.Resources, loss) {
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeResourceChanged, ResourceChangedPayload{
			Changes: loss,
			Reason:  fmt.Sprintf("task_risk:%s", task.Kind),
		}, *versionCursor+1)); err != nil {
			return err
		}
	}

	if crisisEvent := buildTaskCrisisEvent(*simulated, task, *versionCursor+1, successRate, failed); crisisEvent.EventID != "" {
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectEventForeshadowed, SectEventChangedPayload{Event: crisisEvent}, *versionCursor+1)); err != nil {
			return err
		}
	}
	return nil
}

func taskConsequenceSeverity(task TaskState, successRate int, failed bool) int {
	severity := 1
	if failed {
		severity = 2
	}
	if task.Risk >= 80 || successRate < 20 {
		severity++
	}
	return clampInt(severity, 1, 3)
}

func taskFailureResourceLoss(task TaskState, severity int, failed bool) map[ResourceKind]int64 {
	if !failed && severity <= 1 && task.Risk < 40 {
		return nil
	}
	loss := map[ResourceKind]int64{}
	switch normalizeTaskType(task.Kind, task.Type) {
	case TaskTypeExternal:
		loss[ResourceKindSpiritStone] = -int64(2 * severity)
	case TaskTypeExplore:
		loss[ResourceKindOre] = -int64(severity)
	case TaskTypeCombat:
		loss[ResourceKindBeastMat] = -int64(severity)
	default:
		if task.Risk >= 40 {
			loss[ResourceKindSpiritStone] = -int64(severity)
		}
	}
	return loss
}

func buildTaskCrisisEvent(state SectState, task TaskState, seededAtVersion Version, successRate int, failed bool) SectEvent {
	clue := strings.TrimSpace(task.CrisisClue)
	if clue == "" && task.Risk < 40 {
		return SectEvent{}
	}
	if clue == "" {
		clue = fmt.Sprintf("%s_risk", task.Kind)
	}
	sourceDiscipleID := DiscipleID("")
	if len(task.AssignedDiscipleIDs) > 0 {
		sourceDiscipleID = sortedDiscipleIDs(task.AssignedDiscipleIDs)[0]
	}
	severity := 1
	if failed {
		severity = 2
	}
	if task.Risk >= 80 || successRate < 20 {
		severity++
	}
	event := SectEvent{
		EventID:           EventID(fmt.Sprintf("event-task-crisis-%s-%d", task.TaskID, seededAtVersion)),
		Kind:              "task_crisis_clue",
		Status:            SectEventStatusForeshadowed,
		Severity:          clampInt(severity, 1, 4),
		Title:             fmt.Sprintf("%s 后续线索", task.Title),
		Description:       fmt.Sprintf("任务《%s》评价为 %s，留下线索 %s。", task.Title, task.Evaluation, clue),
		OmenText:          fmt.Sprintf("外务回报显示：%s，成功率 %d%%。", clue, successRate),
		SeededAtVersion:   seededAtVersion,
		RevealAtVersion:   seededAtVersion,
		RevealedAtVersion: seededAtVersion,
		ExpiresAtDay:      state.Time.CalendarDay + 5,
		ResultPreview: SectEventResultPreview{
			TensionDelta: severity,
			Summary:      "该线索只进入待处理事件与 tension，不替玩家选择后续结果。",
		},
		Tags: []string{"task", string(normalizeTaskType(task.Kind, task.Type)), clue},
	}
	if sourceDiscipleID != "" {
		event.SourceDiscipleID = &sourceDiscipleID
	}
	return event
}

func metaChangedByTaskReward(task TaskState) bool {
	return task.ReputationReward != 0 || len(task.RelationReward) > 0
}

func applyTaskMetaRewards(meta SectMetaState, task TaskState) SectMetaState {
	next := meta
	next.Relations = cloneStringIntMap(meta.Relations)
	next.Reputation += task.ReputationReward
	for faction, delta := range task.RelationReward {
		if faction == "" || delta == 0 {
			continue
		}
		next.Relations[faction] += delta
	}
	return next
}

func clearDiscipleTaskAssignment(disciple DiscipleState) DiscipleState {
	cleared := disciple
	cleared.AssignmentKind = DiscipleAssignmentIdle
	cleared.AssignmentTask = nil
	cleared.WorkTarget = DiscipleWorkTargetState{Description: "sect_support"}
	return cleared
}

type promotionRequirement struct {
	FromIdentity        IdentityRank
	ToIdentity          IdentityRank
	MinRealm            RealmStage
	MinContribution     int64
	MinContributionCost int64
	MinDutyDays         int
	MinLoyalty          int
	MinSatisfaction     int
	MaxInjuryLevel      int
	RequiredBuilding    string
	RequiredBuildingLv  int32
}

func promotionRequirementFor(from IdentityRank, to IdentityRank) (promotionRequirement, bool) {
	if from == IdentityRankOuter && to == IdentityRankInner {
		return promotionRequirement{
			FromIdentity:        from,
			ToIdentity:          to,
			MinRealm:            RealmQiEntry,
			MinContribution:     20,
			MinContributionCost: 10,
			MinDutyDays:         1,
			MinLoyalty:          60,
			MinSatisfaction:     55,
			MaxInjuryLevel:      1,
			RequiredBuilding:    "main_hall",
			RequiredBuildingLv:  2,
		}, true
	}
	return promotionRequirement{}, false
}

func legalPromotionPath(from IdentityRank, to IdentityRank) bool {
	_, ok := promotionRequirementFor(from, to)
	return ok
}

func promotionContributionCost(from IdentityRank, to IdentityRank) int64 {
	requirement, ok := promotionRequirementFor(from, to)
	if !ok {
		return 0
	}
	return requirement.MinContributionCost
}

func evaluatePromotionAssessment(state SectState, disciple DiscipleState, target IdentityRank, resolvedAt Version) DiscipleAssessmentState {
	requirement, ok := promotionRequirementFor(disciple.Identity, target)
	if !ok {
		return DiscipleAssessmentState{
			TargetRank:        target,
			Passed:            false,
			Score:             0,
			Reason:            fmt.Sprintf("unsupported_promotion_path:%s->%s", disciple.Identity, target),
			ResolvedAtVersion: resolvedAt,
		}
	}

	account := state.Contribution.Accounts[disciple.DiscipleID]
	completedDuty := state.Monthly.Obligations.CompletedDays[disciple.DiscipleID]
	reasons := []string{}
	score := 100
	if realmRankValue(disciple.Realm.Stage) < realmRankValue(requirement.MinRealm) {
		reasons = append(reasons, "realm")
		score -= 25
	}
	if account.EarnedTotal < requirement.MinContribution {
		reasons = append(reasons, "contribution_total")
		score -= 20
	}
	if account.Balance < requirement.MinContributionCost {
		reasons = append(reasons, "contribution_balance")
		score -= 10
	}
	if completedDuty < requirement.MinDutyDays {
		reasons = append(reasons, "monthly_duty")
		score -= 15
	}
	if disciple.Loyalty < requirement.MinLoyalty {
		reasons = append(reasons, "loyalty")
		score -= 10
	}
	if disciple.Satisfaction < requirement.MinSatisfaction {
		reasons = append(reasons, "satisfaction")
		score -= 10
	}
	if disciple.InjuryLevel > requirement.MaxInjuryLevel || disciple.HP <= 0 {
		reasons = append(reasons, "injury")
		score -= 15
	}
	if !stateHasActiveBuilding(state, requirement.RequiredBuilding, requirement.RequiredBuildingLv) {
		reasons = append(reasons, "building")
		score -= 20
	}
	memoryScore, memoryReasons := disciplePromotionMemoryModifier(disciple)
	score += memoryScore
	reasons = append(reasons, memoryReasons...)
	if score < 0 {
		score = 0
	}
	reason := "passed"
	if len(reasons) > 0 {
		reason = strings.Join(reasons, ",")
	}
	return DiscipleAssessmentState{
		TargetRank:        target,
		Passed:            len(reasons) == 0,
		Score:             score,
		Reason:            reason,
		ResolvedAtVersion: resolvedAt,
	}
}

func stateHasActiveBuilding(state SectState, definitionKey string, minLevel int32) bool {
	for _, building := range state.Buildings {
		if building.DefinitionKey == definitionKey && building.Level >= minLevel && building.Phase == "active" && building.HP > 0 {
			return true
		}
	}
	return false
}

func identityRankValue(identity IdentityRank) int {
	switch identity {
	case IdentityRankOuter:
		return 1
	case IdentityRankInner:
		return 2
	default:
		return 0
	}
}

func realmRankValue(realm RealmStage) int {
	switch realm {
	case RealmMortal:
		return 0
	case RealmQiEntry:
		return 1
	case RealmQiEarly:
		return 2
	case RealmQiMiddle:
		return 3
	case RealmQiLate:
		return 4
	case RealmFoundation:
		return 5
	case RealmGoldenCore:
		return 6
	default:
		return 0
	}
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func sortedTaskIDs(tasks map[TaskID]TaskState) []TaskID {
	ids := make([]TaskID, 0, len(tasks))
	for id := range tasks {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func sortedProductionIDs(productions map[ProductionID]ProductionJob) []ProductionID {
	ids := make([]ProductionID, 0, len(productions))
	for id := range productions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func sortedBuildingIDs(buildings map[BuildingID]BuildingState) []BuildingID {
	ids := make([]BuildingID, 0, len(buildings))
	for id := range buildings {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func buildBuildingCatalog(state SectState) []BuildingCatalogEntry {
	definitionKeys := []string{"main_hall", "warehouse", "gate"}
	catalog := make([]BuildingCatalogEntry, 0, len(definitionKeys))
	for _, definitionKey := range definitionKeys {
		config, ok := buildingConfigFor(definitionKey)
		if !ok {
			continue
		}
		blockers := buildingCatalogBlockers(state, config)
		catalog = append(catalog, BuildingCatalogEntry{
			DefinitionKey:         config.DefinitionKey,
			Label:                 buildingCatalogLabel(config.DefinitionKey),
			MaxLevel:              config.MaxLevel,
			UnlockSectLevel:       config.UnlockSectLevel,
			RequiredMainHallLevel: config.RequiredMainHallLevel,
			MaxCount:              config.MaxCount,
			CurrentCount:          countBuildingsByDefinition(state, config.DefinitionKey),
			BuildCost:             cloneResourceAmounts(config.BuildCost),
			UpgradeCostByLevel:    cloneResourceCostByLevel(config.UpgradeCostByLevel),
			MaintenanceByLevel:    cloneResourceCostByLevel(config.MaintenanceByLevel),
			Unlocked:              buildingCatalogUnlocked(state, config),
			CanBuild:              len(blockers) == 0,
			Blockers:              blockers,
			ExistingBuildings:     buildingCatalogInstances(state, config.DefinitionKey),
		})
	}
	return catalog
}

func buildingCatalogLabel(definitionKey string) string {
	switch definitionKey {
	case "main_hall":
		return "主殿"
	case "warehouse":
		return "仓房"
	case "gate":
		return "护山台"
	default:
		return definitionKey
	}
}

func cloneResourceCostByLevel(source map[int32]ResourceCost) map[int32]ResourceCost {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[int32]ResourceCost, len(source))
	for level, cost := range source {
		cloned[level] = cloneResourceAmounts(cost)
	}
	return cloned
}

func buildingCatalogUnlocked(state SectState, config buildingAuthorityConfig) bool {
	if effectiveSectLevel(state) < config.UnlockSectLevel {
		return false
	}
	if config.RequiredMainHallLevel > 0 && !stateHasActiveBuilding(state, "main_hall", config.RequiredMainHallLevel) {
		return false
	}
	return true
}

func buildingCatalogBlockers(state SectState, config buildingAuthorityConfig) []string {
	blockers := []string{}
	if effectiveSectLevel(state) < config.UnlockSectLevel {
		blockers = append(blockers, fmt.Sprintf("sect_level_%d_required", config.UnlockSectLevel))
	}
	if config.RequiredMainHallLevel > 0 && !stateHasActiveBuilding(state, "main_hall", config.RequiredMainHallLevel) {
		blockers = append(blockers, fmt.Sprintf("main_hall_level_%d_required", config.RequiredMainHallLevel))
	}
	if int32(len(state.Buildings)) >= effectiveBuildingLimit(state) {
		blockers = append(blockers, fmt.Sprintf("building_limit_%d_reached", effectiveBuildingLimit(state)))
	}
	if countBuildingsByDefinition(state, config.DefinitionKey) >= config.MaxCount {
		blockers = append(blockers, fmt.Sprintf("definition_count_limit_%d_reached", config.MaxCount))
	}
	if !CanAfford(state.Resources, config.BuildCost) {
		blockers = append(blockers, "insufficient_resources")
	}
	return blockers
}

func buildingCatalogInstances(state SectState, definitionKey string) []BuildingCatalogInstance {
	ids := sortedBuildingIDs(state.Buildings)
	instances := []BuildingCatalogInstance{}
	for _, id := range ids {
		building := normalizedBuildingRuntimeFields(state.Buildings[id])
		if building.DefinitionKey != definitionKey {
			continue
		}
		instances = append(instances, BuildingCatalogInstance{
			BuildingID:      building.BuildingID,
			Level:           building.Level,
			Phase:           building.Phase,
			Efficiency:      building.Efficiency,
			Durability:      building.Durability,
			MaintenanceDebt: building.MaintenanceDebt,
			DamagedReason:   building.DamagedReason,
		})
	}
	return instances
}

func sortedEventIDs(events map[EventID]SectEvent) []EventID {
	ids := make([]EventID, 0, len(events))
	for id := range events {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func sortedDiscipleIDs(ids []DiscipleID) []DiscipleID {
	clone := append([]DiscipleID(nil), ids...)
	sort.Slice(clone, func(i, j int) bool { return clone[i] < clone[j] })
	return clone
}

func sortedDiscipleMapIDs(disciples map[DiscipleID]DiscipleState) []DiscipleID {
	ids := make([]DiscipleID, 0, len(disciples))
	for id := range disciples {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func nextDiscipleID(disciples map[DiscipleID]DiscipleState) DiscipleID {
	for index := len(disciples) + 1; ; index++ {
		id := DiscipleID(fmt.Sprintf("disciple-%d", index))
		if _, exists := disciples[id]; !exists {
			return id
		}
	}
}

func buildRecruitmentCandidate(state SectState, source RecruitmentType, index int, investmentSpiritStone int64) CandidateState {
	qualityBoost := minInt(3, int(investmentSpiritStone/20))
	qualityBoost += recruitmentQualityBoostForPolicy(state.Policies.RecruitmentPolicy)
	base := 6 + int((uint64(state.Runtime.Version)+uint64(index)*3+state.Runtime.RNG.Cursor)%3) + qualityBoost
	aptitude := DiscipleAptitudeState{
		SpiritRoot:    minInt(14, base+index%2),
		Comprehension: minInt(14, base+(index+1)%3),
		Physique:      minInt(14, base+(index+2)%3),
		Mind:          minInt(14, base),
		Luck:          minInt(14, base-1+((index+2)%2)),
	}
	maxHP := int64(80 + aptitude.Physique*3)
	candidateID := CandidateID(fmt.Sprintf("candidate-%d-%d", state.Runtime.Version+1, index+1))
	return CandidateState{
		CandidateID: candidateID,
		Name:        fmt.Sprintf("candidate_apprentice_%d", index+1),
		Source:      source,
		Identity:    IdentityRankOuter,
		Aptitude:    aptitude,
		Realm:       RealmState{Stage: RealmMortal},
		Needs: DiscipleNeedsState{
			DailySpiritGrain: 1,
			DailyRestTicks:   1,
		},
		Support: DiscipleSupportState{
			FoodSatisfied:    true,
			HousingSatisfied: true,
			MedicalSupported: false,
		},
		HP:    maxHP,
		MaxHP: maxHP,
	}
}

func newDiscipleFromCandidate(candidate CandidateState, discipleID DiscipleID) DiscipleState {
	return DiscipleState{
		DiscipleID:     discipleID,
		Name:           candidate.Name,
		Location:       TileCoord{Col: 0, Row: 0},
		Identity:       candidate.Identity,
		Aptitude:       candidate.Aptitude,
		AssignmentKind: DiscipleAssignmentIdle,
		WorkTarget: DiscipleWorkTargetState{
			Description: "sect_support",
		},
		Realm:        candidate.Realm,
		Needs:        candidate.Needs,
		Support:      candidate.Support,
		Satisfaction: 70,
		Loyalty:      70,
		Pressure:     candidate.Pressure,
		InjuryLevel:  candidate.InjuryLevel,
		HP:           candidate.HP,
		MaxHP:        candidate.MaxHP,
	}
}

func allDisciplesExist(disciples map[DiscipleID]DiscipleState, ids []DiscipleID) bool {
	for _, id := range ids {
		if _, ok := disciples[id]; !ok {
			return false
		}
	}
	return true
}

func productionCompletionDelta(job ProductionJob) map[ResourceKind]int64 {
	delta := map[ResourceKind]int64{}
	for kind, amount := range job.InputCost {
		if amount > 0 {
			delta[kind] -= amount
		}
	}
	for kind, amount := range job.OutputReward {
		if amount > 0 {
			delta[kind] += amount
		}
	}
	if len(delta) == 0 {
		return nil
	}
	return delta
}

func productionCompletionDeltaForState(state SectState, job ProductionJob) map[ResourceKind]int64 {
	delta := productionCompletionDelta(job)
	if len(delta) == 0 || job.Kind != ProductionKindFarm {
		return delta
	}
	formation, ok := state.Formations[job.BuildingID]
	if !ok || formation.Kind != FormationKindFieldGuard {
		return delta
	}
	definition, err := formationDefinitionFor(formation.Kind)
	if err != nil {
		return delta
	}
	bonus := definition.ProductionBonus + int64(formationStrength(formation)/18)
	if bonus <= 0 {
		return delta
	}
	for kind, amount := range job.OutputReward {
		if amount <= 0 {
			continue
		}
		delta[kind] += bonus
	}
	return delta
}

func productionShortage(resources ResourceState, cost map[ResourceKind]int64) map[ResourceKind]int64 {
	shortage := map[ResourceKind]int64{}
	for kind, amount := range cost {
		if amount <= 0 {
			continue
		}
		if missing := amount - resources.Stock[kind]; missing > 0 {
			shortage[kind] = missing
		}
	}
	if len(shortage) == 0 {
		return nil
	}
	return shortage
}

func productionClosed(status ProductionStatus) bool {
	return status == ProductionStatusCompleted || status == ProductionStatusCancelled
}

func nextPlayerProductionID(productions map[ProductionID]ProductionJob) ProductionID {
	for index := 1; ; index++ {
		id := ProductionID(fmt.Sprintf("prod-player-%d", index))
		if _, exists := productions[id]; !exists {
			return id
		}
	}
}

func activeBreakthroughPortentForDisciple(events SectEventState, discipleID DiscipleID) *SectEvent {
	for _, eventID := range sortedEventIDs(events.ActiveEvents) {
		event := events.ActiveEvents[eventID]
		if event.Kind != "breakthrough_portent" || event.SourceDiscipleID == nil || *event.SourceDiscipleID != discipleID {
			continue
		}
		entry := event
		return &entry
	}
	return nil
}

const sectGovernanceChoiceEventKind = "sect_governance_choice"
const (
	sectCrisisOmenEventKind   = "sect_crisis_omen"
	sectCrisisMinorEventKind  = "sect_crisis_minor"
	sectCrisisChoiceEventKind = "sect_crisis_choice"
)

func (a *SectActor) appendSectCrisisChainEvents(
	simulated *SectState,
	versionCursor *Version,
	appendEvent appendDomainEventFunc,
) error {
	if simulated == nil {
		return nil
	}
	ensureSectEventState(simulated)
	if choice := activeSectCrisisEventByStage(simulated.Events, "choice"); choice != nil {
		return nil
	}
	if minor := activeSectCrisisEventByStage(simulated.Events, "minor_crisis"); minor != nil && *versionCursor > minor.RevealedAtVersion {
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectEventResolved, SectEventResolvedPayload{
			EventID: minor.EventID,
			Summary: ResolvedEventSummary{
				EventID:           minor.EventID,
				Kind:              minor.Kind,
				Outcome:           "advanced:choice",
				Summary:           fmt.Sprintf("《%s》已升级为掌门必须处置的危机选择。", minor.Title),
				ResolvedAtVersion: *versionCursor + 1,
			},
		}, *versionCursor+1)); err != nil {
			return err
		}
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectEventForeshadowed, SectEventChangedPayload{
			Event: buildSectCrisisChoiceEvent(*simulated, *minor, *versionCursor+1),
		}, *versionCursor+1)); err != nil {
			return err
		}
		return nil
	}
	if omen := activeSectCrisisEventByStage(simulated.Events, "omen"); omen != nil && omen.Status == SectEventStatusForeshadowed && *versionCursor > omen.RevealedAtVersion {
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectEventResolved, SectEventResolvedPayload{
			EventID: omen.EventID,
			Summary: ResolvedEventSummary{
				EventID:           omen.EventID,
				Kind:              omen.Kind,
				Outcome:           "advanced:minor_crisis",
				Summary:           fmt.Sprintf("《%s》的征兆已转为可见的小危机。", omen.Title),
				ResolvedAtVersion: *versionCursor + 1,
			},
		}, *versionCursor+1)); err != nil {
			return err
		}
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectEventForeshadowed, SectEventChangedPayload{
			Event: buildSectCrisisMinorEvent(*simulated, *omen, *versionCursor+1),
		}, *versionCursor+1)); err != nil {
			return err
		}
		return nil
	}
	if shouldSeedSectCrisisOmen(*simulated) {
		projection := buildDefenseRiskProjection(*simulated)
		seededAtVersion := *versionCursor + 1
		if err := appendEvent(a.newDomainEvent("", DomainEventTypeSectEventSeeded, SectEventChangedPayload{
			Event: buildSectCrisisOmenEvent(*simulated, seededAtVersion, projection),
		}, seededAtVersion)); err != nil {
			return err
		}
	}
	return nil
}

func activeSectCrisisEventByStage(events SectEventState, stage string) *SectEvent {
	for _, eventID := range sortedEventIDs(events.ActiveEvents) {
		event := events.ActiveEvents[eventID]
		if event.ChainID == "" || event.ChainStage != stage {
			continue
		}
		entry := event
		return &entry
	}
	return nil
}

func activeCrisisChainEventExists(events SectEventState) bool {
	for _, event := range events.ActiveEvents {
		if event.ChainID != "" {
			return true
		}
	}
	return false
}

func shouldSeedSectCrisisOmen(state SectState) bool {
	if activeCrisisChainEventExists(state.Events) {
		return false
	}
	if state.Events.LastMajorEventVersion > 0 && state.Runtime.Version < state.Events.LastMajorEventVersion+20 {
		return false
	}
	return buildDefenseRiskProjection(state).Intensity >= 55
}

func shouldSeedSectGovernanceChoiceEvent(state SectState) bool {
	if state.Time.CalendarDay < 2 {
		return false
	}
	if hasAcceptedSectTask(state.Tasks) {
		return false
	}
	if sectEventKindExists(state.Events, sectGovernanceChoiceEventKind) {
		return false
	}
	return true
}

func buildSectCrisisOmenEvent(state SectState, seededAtVersion Version, projection DefenseRiskProjection) SectEvent {
	chainID := fmt.Sprintf("crisis-chain-%d", seededAtVersion)
	sourceDiscipleID := defaultGoalFocusDisciple(state, nil)
	omenText := fmt.Sprintf("风险升至 %d，主要来源：%s。若继续放任，征兆会在下一日转为小危机。", projection.Intensity, summarizeDefenseRiskSources(projection.SourceSummary))
	event := SectEvent{
		EventID:         EventID(fmt.Sprintf("event-sect-crisis-omen-%d", seededAtVersion)),
		Kind:            sectCrisisOmenEventKind,
		Status:          SectEventStatusSeeded,
		Severity:        clampInt(maxInt(projection.Intensity/20, 2), 2, 4),
		Title:           "宗门危机预兆",
		Description:     "宗门名望、财富、外务与暗潮开始叠加，危机正逐步逼近。",
		OmenText:        omenText,
		ChainID:         chainID,
		ChainStage:      "omen",
		SeededAtVersion: seededAtVersion,
		RevealAtVersion: seededAtVersion + 1,
		Tags:            []string{"crisis", "omen", "risk_curve"},
		ResultPreview: SectEventResultPreview{
			Summary: "该危机必须先经历预兆与显形阶段，authority 不会跳过掌门选择直接结算重大结果。",
		},
	}
	if sourceDiscipleID != "" {
		event.SourceDiscipleID = &sourceDiscipleID
	}
	return event
}

func buildSectCrisisMinorEvent(state SectState, omen SectEvent, version Version) SectEvent {
	return SectEvent{
		EventID:           EventID(fmt.Sprintf("event-sect-crisis-minor-%s-%d", omen.ChainID, version)),
		Kind:              sectCrisisMinorEventKind,
		Status:            SectEventStatusForeshadowed,
		Severity:          clampInt(omen.Severity, 2, 4),
		Title:             "危机显形",
		Description:       "山门外出现针对宗门的试探与骚动，小危机已经从预兆转为现实。",
		OmenText:          "危机已显形：下一步若不由掌门抉择，将只会以过期记录收场，不会静默替你承担。",
		ChainID:           omen.ChainID,
		ChainStage:        "minor_crisis",
		SourceDiscipleID:  omen.SourceDiscipleID,
		SeededAtVersion:   version,
		RevealAtVersion:   version,
		RevealedAtVersion: version,
		ExpiresAtDay:      state.Time.CalendarDay + 2,
		Tags:              []string{"crisis", "minor_crisis", "risk_curve"},
		ResultPreview: SectEventResultPreview{
			Summary: "小危机已被 authority 记录，若继续恶化将进入掌门必须选择的节点。",
		},
	}
}

func buildSectCrisisChoiceEvent(state SectState, minor SectEvent, version Version) SectEvent {
	fortifyDelta := map[ResourceKind]int64{
		ResourceKindSpiritStone:  -12,
		ResourceKindFormationMat: -1,
	}
	appeaseDelta := map[ResourceKind]int64{
		ResourceKindSpiritGrain: -6,
		ResourceKindHerb:        -2,
	}
	return SectEvent{
		EventID:           EventID(fmt.Sprintf("event-sect-crisis-choice-%s-%d", minor.ChainID, version)),
		Kind:              sectCrisisChoiceEventKind,
		Status:            SectEventStatusForeshadowed,
		Severity:          clampInt(minor.Severity+1, 2, 4),
		Title:             "危机抉择",
		Description:       "危机已越过试探阶段，掌门必须决定是加固防线还是出资安抚外部压力。",
		OmenText:          "若掌门迟疑，authority 只会记下错失与过期，不会替你偷偷选项。",
		ChainID:           minor.ChainID,
		ChainStage:        "choice",
		SourceDiscipleID:  minor.SourceDiscipleID,
		SeededAtVersion:   version,
		RevealAtVersion:   version,
		RevealedAtVersion: version,
		ExpiresAtDay:      state.Time.CalendarDay + 3,
		Tags:              []string{"crisis", "choice", "risk_curve"},
		Options: []SectEventOption{
			{
				OptionID:    "fortify_perimeter",
				Label:       "加固山门",
				Description: "消耗灵石与阵材，调集宗门防备。",
				Requirements: SectEventRequirement{
					MinResources: map[ResourceKind]int64{
						ResourceKindSpiritStone:  12,
						ResourceKindFormationMat: 1,
					},
				},
				ResultPreview: SectEventResultPreview{
					ResourceDelta:             fortifyDelta,
					DisciplePressureDelta:     6,
					DiscipleSatisfactionDelta: 2,
					Summary:                   "宗门付出阵材与灵石换取短期防线稳固，相关弟子压力上升但认可掌门决断。",
				},
			},
			{
				OptionID:    "appease_locals",
				Label:       "资粮安抚",
				Description: "消耗灵粮与灵植安抚山下人心，缓和危机外扩。",
				Requirements: SectEventRequirement{
					MinResources: map[ResourceKind]int64{
						ResourceKindSpiritGrain: 6,
						ResourceKindHerb:        2,
					},
				},
				ResultPreview: SectEventResultPreview{
					ResourceDelta:             appeaseDelta,
					DisciplePressureDelta:     -2,
					DiscipleSatisfactionDelta: 3,
					Summary:                   "宗门以资粮换来缓冲，相关弟子感到局势稍稳。",
				},
			},
		},
		ResultPreview: SectEventResultPreview{
			Summary: "危机已进入掌门选择阶段；若过期，只会保留错失记录，不会替玩家结算重大后果。",
		},
	}
}

func summarizeDefenseRiskSources(summary []DefenseRiskSourceSummary) string {
	if len(summary) == 0 {
		return "风险来源尚不明朗"
	}
	parts := make([]string, 0, minInt(len(summary), 3))
	for index, entry := range summary {
		if index >= 3 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s(%+d)", entry.Source, entry.Delta))
	}
	return strings.Join(parts, "、")
}

func hasAcceptedSectTask(tasks map[TaskID]TaskState) bool {
	for _, task := range tasks {
		if task.Status == TaskStatusAccepted {
			return true
		}
	}
	return false
}

func sectEventKindExists(events SectEventState, kind string) bool {
	for _, event := range events.ActiveEvents {
		if event.Kind == kind {
			return true
		}
	}
	for _, event := range events.ResolvedEvents {
		if event.Kind == kind {
			return true
		}
	}
	return false
}

func buildSectGovernanceChoiceEvent(state SectState, seededAtVersion Version) SectEvent {
	expiresAtDay := state.Time.CalendarDay + 3
	aidDelta := map[ResourceKind]int64{
		ResourceKindSpiritStone: -10,
		ResourceKindHerb:        -3,
	}
	sealDelta := map[ResourceKind]int64{
		ResourceKindFormationMat: -1,
	}
	return SectEvent{
		EventID:           EventID(fmt.Sprintf("event-sect-governance-%d", seededAtVersion)),
		Kind:              sectGovernanceChoiceEventKind,
		Status:            SectEventStatusForeshadowed,
		Severity:          2,
		Title:             "山下村寨求援",
		Description:       "山下村寨送来急信，请宗门在妖踪未散前决定是否出手。",
		OmenText:          "山门外民心浮动，若掌门不决，事态只会自然过期，不会被 authority 代选。",
		SeededAtVersion:   seededAtVersion,
		RevealAtVersion:   seededAtVersion,
		RevealedAtVersion: seededAtVersion,
		ExpiresAtDay:      expiresAtDay,
		Requirements: SectEventRequirement{
			MinResources: map[ResourceKind]int64{ResourceKindSpiritStone: 1},
		},
		Options: []SectEventOption{
			{
				OptionID:    "send_aid",
				Label:       "遣物救援",
				Description: "消耗灵石与灵植安抚村寨。",
				Requirements: SectEventRequirement{
					MinResources: map[ResourceKind]int64{
						ResourceKindSpiritStone: 10,
						ResourceKindHerb:        3,
					},
				},
				ResultPreview: SectEventResultPreview{
					ResourceDelta: aidDelta,
					TensionDelta:  -1,
					Summary:       "宗门拨出灵石与灵植，村寨暂得安稳。",
				},
			},
			{
				OptionID:    "seal_mountain",
				Label:       "封山自保",
				Description: "消耗阵材收束山门，不外派弟子。",
				Requirements: SectEventRequirement{
					MinResources: map[ResourceKind]int64{ResourceKindFormationMat: 1},
				},
				ResultPreview: SectEventResultPreview{
					ResourceDelta: sealDelta,
					TensionDelta:  1,
					Summary:       "宗门封山自保，山外怨气仍在暗处积累。",
				},
			},
		},
		ResultPreview: SectEventResultPreview{
			Summary: "需掌门选择救援或封山；过期只记录错失，不自动结算重大结果。",
		},
		Tags: []string{"sect_event", "choice", "governance"},
	}
}

func buildBreakthroughPortentEvent(disciple DiscipleState, seededAtVersion Version) SectEvent {
	sourceDiscipleID := disciple.DiscipleID
	nextStage := nextRealmStage(disciple.Realm.Stage)
	return SectEvent{
		EventID:           EventID(fmt.Sprintf("event-breakthrough-%s-%s-%d", disciple.DiscipleID, disciple.Realm.Stage, seededAtVersion)),
		Kind:              "breakthrough_portent",
		Status:            SectEventStatusSeeded,
		Severity:          breakthroughEventSeverity(disciple.Realm.Stage),
		Title:             fmt.Sprintf("%s 气机翻涌", disciple.Name),
		Description:       fmt.Sprintf("%s 已触及 %s 的门槛，命数正在积聚。", disciple.Name, formatRealmStageLabel(nextStage)),
		OmenText:          fmt.Sprintf("天机示警：%s 的突破契机已至，需先观其气机再行决断。", disciple.Name),
		SourceDiscipleID:  &sourceDiscipleID,
		SeededAtVersion:   seededAtVersion,
		RevealAtVersion:   seededAtVersion + 1,
		RevealedAtVersion: 0,
		ResolvedAtVersion: 0,
		Tags:              []string{"breakthrough", string(disciple.Realm.Stage), string(nextStage)},
	}
}

func dailyCultivationPoints(disciple DiscipleState) int64 {
	points := int64(12)
	if disciple.Support.HousingSatisfied {
		points += 2
	}
	if disciple.Support.FoodSatisfied {
		points += 1
	}
	if strings.HasPrefix(disciple.WorkTarget.Description, "cultivation:cave") {
		points += 6
	}
	if disciple.Pressure >= 60 {
		points -= 4
	}
	if disciple.InjuryLevel > 0 {
		points -= int64(disciple.InjuryLevel * 2)
	}
	if points < 1 {
		return 1
	}
	return points
}

func dailyCultivationPointsForPolicy(state SectState, disciple DiscipleState) int64 {
	points := dailyCultivationPoints(disciple)
	points += institutionCultivationSupport(state, disciple)
	points += formationCultivationBonus(state)
	points += int64(discipleArtifactStat(state, disciple, "cultivation"))
	if state.Policies.CultivationPolicy == CultivationPolicyClosedCultivation {
		points += 3
	}
	if state.Policies.TaskPolicy == TaskPolicyClosedCultivation {
		points += 1
	}
	if state.Policies.CultivationPolicy == CultivationPolicyBreakthroughSafe && disciple.Realm.ReadyForBreakthrough {
		points = maxInt64(1, points-2)
	}
	return points
}

func formationCultivationBonus(state SectState) int64 {
	bonus := int64(0)
	for _, buildingID := range sortedFormationBuildingIDs(state.Formations) {
		formation := state.Formations[buildingID]
		definition, err := formationDefinitionFor(formation.Kind)
		if err != nil {
			continue
		}
		switch formation.Kind {
		case FormationKindGatherSpirit, FormationKindCalmMind:
			bonus += int64(definition.CultivationBonus + formationStrength(formation)/20)
		}
	}
	return bonus
}

func formationPressureRelief(state SectState) int {
	total := 0
	for _, buildingID := range sortedFormationBuildingIDs(state.Formations) {
		formation := state.Formations[buildingID]
		if formation.Kind != FormationKindCalmMind {
			continue
		}
		definition, err := formationDefinitionFor(formation.Kind)
		if err != nil {
			continue
		}
		total += definition.PressureRelief + formationStrength(formation)/12
	}
	return clampInt(total, 0, 20)
}

func cultivationThresholdForStage(stage RealmStage) int64 {
	switch stage {
	case RealmMortal:
		return 30
	case RealmQiEntry:
		return 60
	case RealmQiEarly:
		return 90
	case RealmQiMiddle:
		return 130
	case RealmQiLate:
		return 180
	case RealmFoundation:
		return 260
	default:
		return 1<<62 - 1
	}
}

func nextRealmStage(stage RealmStage) RealmStage {
	switch stage {
	case RealmMortal:
		return RealmQiEntry
	case RealmQiEntry:
		return RealmQiEarly
	case RealmQiEarly:
		return RealmQiMiddle
	case RealmQiMiddle:
		return RealmQiLate
	case RealmQiLate:
		return RealmFoundation
	case RealmFoundation:
		return RealmGoldenCore
	default:
		return stage
	}
}

func breakthroughEventSeverity(stage RealmStage) int {
	switch stage {
	case RealmFoundation:
		return 4
	case RealmQiLate, RealmQiMiddle:
		return 3
	default:
		return 2
	}
}

func breakthroughSpiritStoneCost(stage RealmStage) int64 {
	switch stage {
	case RealmMortal:
		return 6
	case RealmQiEntry:
		return 10
	case RealmQiEarly:
		return 14
	case RealmQiMiddle:
		return 18
	case RealmQiLate:
		return 24
	case RealmFoundation:
		return 40
	default:
		return 0
	}
}

func breakthroughRiskScore(disciple DiscipleState) int {
	return disciple.Pressure + disciple.InjuryLevel*20 + disciple.Realm.FailedBreakthroughCount*5
}

func breakthroughRiskLimit(stage RealmStage) int {
	switch stage {
	case RealmMortal:
		return 45
	case RealmQiEntry:
		return 40
	case RealmQiEarly:
		return 36
	case RealmQiMiddle:
		return 32
	case RealmQiLate:
		return 28
	case RealmFoundation:
		return 24
	default:
		return 0
	}
}

func breakthroughFailurePoints(disciple DiscipleState) int64 {
	loss := cultivationThresholdForStage(disciple.Realm.Stage) / 5
	next := disciple.Realm.CultivationPoints - loss
	if next < 0 {
		return 0
	}
	return next
}

func breakthroughSuccessRate(disciple DiscipleState) int {
	riskOverLimit := breakthroughRiskScore(disciple) - breakthroughRiskLimit(disciple.Realm.Stage)
	if riskOverLimit <= 0 {
		return 100
	}
	return clampInt(100-riskOverLimit*5, 5, 100)
}

func cultivationEnvironmentBonus(disciple DiscipleState) int {
	bonus := 0
	if disciple.Support.FoodSatisfied {
		bonus += 5
	}
	if disciple.Support.HousingSatisfied {
		bonus += 10
	}
	if strings.HasPrefix(disciple.WorkTarget.Description, "cultivation:cave") {
		bonus += 25
	}
	if disciple.Support.MedicalSupported {
		bonus += 5
	}
	return bonus
}

func cultivationPillPointGain() int64 {
	return 20
}

func reserveCaveSpiritStoneCost(durationDays int32) int64 {
	if durationDays <= 0 {
		durationDays = 1
	}
	return int64(durationDays) * 5
}

func pillInventoryItemID(pill PillType) ItemID {
	switch pill {
	case PillCultivation:
		return ItemID("pill-cultivation")
	case PillBreakthrough:
		return ItemID("pill-breakthrough")
	case PillCalmMind:
		return ItemID("pill-calm-mind")
	default:
		return ItemID(fmt.Sprintf("pill-%s", pill))
	}
}

func inventoryPillQuantity(state SectState, pill PillType) int64 {
	item := state.Inventory.Items[pillInventoryItemID(pill)]
	if item.Kind != string(pill) {
		return 0
	}
	if item.Quantity < 0 {
		return 0
	}
	return item.Quantity
}

func refreshCultivationDecisionState(state *SectState) {
	if state == nil {
		return
	}
	for discipleID, disciple := range state.Disciples {
		disciple.Cultivation = buildCultivationDecisionState(*state, disciple)
		state.Disciples[discipleID] = disciple
	}
}

func buildCultivationDecisionState(state SectState, disciple DiscipleState) CultivationDecisionState {
	required := cultivationThresholdForStage(disciple.Realm.Stage)
	progressPercent := 0
	if required > 0 && required < 1<<62-1 {
		progressPercent = clampInt(int(disciple.Realm.CultivationPoints*100/required), 0, 100)
	}
	dailyGain := dailyCultivationPointsForPolicy(state, disciple)
	if state.Resources.Stock[ResourceKindSpiritStone] <= 0 {
		dailyGain = 0
	}
	omenStatus := "none"
	omenText := ""
	if disciple.Realm.ReadyForBreakthrough {
		omenStatus = "pending_seed"
	}
	if omen := activeBreakthroughPortentForDisciple(state.Events, disciple.DiscipleID); omen != nil {
		omenStatus = string(omen.Status)
		if omen.OmenText != "" {
			omenText = omen.OmenText
		} else {
			omenText = omen.Description
		}
	}
	return CultivationDecisionState{
		DailyGain:                   dailyGain,
		RequiredPoints:              required,
		ProgressPercent:             progressPercent,
		EnvironmentBonus:            cultivationEnvironmentBonus(disciple) + discipleArtifactStat(state, disciple, "cultivation"),
		CultivationPillAvailable:    inventoryPillQuantity(state, PillCultivation),
		BreakthroughPillAvailable:   inventoryPillQuantity(state, PillBreakthrough),
		BreakthroughSpiritStoneCost: breakthroughSpiritStoneCost(disciple.Realm.Stage),
		BreakthroughSuccessRate:     breakthroughSuccessRate(disciple),
		BreakthroughRisk:            breakthroughRiskScore(disciple),
		BreakthroughRiskLimit:       breakthroughRiskLimit(disciple.Realm.Stage),
		OmenStatus:                  omenStatus,
		OmenText:                    omenText,
	}
}

func formatRealmStageLabel(stage RealmStage) string {
	return strings.ReplaceAll(string(stage), "_", " ")
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func maxInt64(left int64, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

func (a *SectActor) snapshot(userID UserID, sessionID string) SectSnapshot {
	state := a.state.Clone()
	ensureFormationState(&state)
	ensureInstitutionState(&state)
	refreshCultivationDecisionState(&state)
	eventLog := boundDomainEventLog(cloneDomainEvents(a.eventLog))
	primaryDiscipleID := starterDiscipleID
	for _, discipleID := range sortedDiscipleMapIDs(state.Disciples) {
		primaryDiscipleID = discipleID
		break
	}
	return SectSnapshot{
		SectID:          string(a.state.Meta.SectID),
		UserID:          string(userID),
		SessionID:       normalizeSessionID(sessionID),
		SceneVersion:    a.state.Runtime.Version,
		State:           state,
		DefenseRisk:     buildDefenseRiskProjection(state),
		BuildingCatalog: buildBuildingCatalog(state),
		TaskDispatch:    buildTaskDispatchProjections(state),
		EventLog:        BuildEventLogEntriesFromEventLog(eventLog),
		Diary:           BuildDiscipleDiaryFromEventLog(eventLog, primaryDiscipleID),
		EventSummaries:  BuildSectEventFeedbackFromEventLog(eventLog),
	}
}

func boundDomainEventLog(events []DomainEvent) []DomainEvent {
	const eventLogLimit = 80
	if len(events) <= eventLogLimit {
		return events
	}
	return events[len(events)-eventLogLimit:]
}

func (a *SectActor) rejected(msg SubmitCommand, code CommandErrorCode, message string, retriable bool) SubmitCommandResponse {
	return SubmitCommandResponse{
		Snapshot: a.snapshot(UserID(msg.Command.UserID), msg.SessionID),
		Result: CommandResult{
			CmdID:        msg.Command.CmdID,
			Status:       CommandResultStatusRejected,
			Error:        &CommandError{Code: code, Message: message, Retriable: retriable},
			SectID:       string(a.state.Meta.SectID),
			SceneVersion: a.state.Runtime.Version,
			Patch: StatePatch{
				SectID:      string(a.state.Meta.SectID),
				FromVersion: a.state.Runtime.Version,
				ToVersion:   a.state.Runtime.Version,
			},
		},
	}
}

func DefaultSectIDForUser(userID UserID) SectID {
	if strings.TrimSpace(string(userID)) == "" {
		return SectID("sect-preview-player")
	}
	return SectID(fmt.Sprintf("sect-%s", strings.TrimSpace(string(userID))))
}

func normalizeSessionID(sessionID string) string {
	if strings.TrimSpace(sessionID) == "" {
		return "preview-session"
	}
	return strings.TrimSpace(sessionID)
}
