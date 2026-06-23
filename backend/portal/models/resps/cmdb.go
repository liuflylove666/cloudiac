// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CmdbAssetResp struct {
	models.CmdbAsset

	ProjectName string `json:"projectName"`
	EnvName     string `json:"envName"`

	Relations       []CmdbAssetRelationResp       `json:"relations,omitempty" gorm:"-"`
	RelationSummary *CmdbAssetRelationSummaryResp `json:"relationSummary,omitempty" gorm:"-"`
	Changes         []CmdbAssetChangeResp         `json:"changes,omitempty" gorm:"-"`
}

type CmdbAssetRelationResp struct {
	models.CmdbAssetRelation

	SourceAssetName string `json:"sourceAssetName"`
	SourceAssetType string `json:"sourceAssetType"`
	TargetAssetName string `json:"targetAssetName"`
	TargetAssetType string `json:"targetAssetType"`
}

type CmdbAssetRelationsResp struct {
	AssetId   models.Id                    `json:"assetId"`
	Relations []CmdbAssetRelationResp      `json:"relations"`
	Summary   CmdbAssetRelationSummaryResp `json:"summary"`
}

type CmdbAssetRelationSummaryResp struct {
	TotalRelationCount               int64                         `json:"totalRelationCount"`
	DirectRelationCount              int64                         `json:"directRelationCount"`
	ApplicationInferredRelationCount int64                         `json:"applicationInferredRelationCount"`
	ReturnedRelationCount            int64                         `json:"returnedRelationCount"`
	IncomingRelationCount            int64                         `json:"incomingRelationCount"`
	OutgoingRelationCount            int64                         `json:"outgoingRelationCount"`
	Limit                            int                           `json:"limit"`
	Offset                           int                           `json:"offset"`
	NextOffset                       int                           `json:"nextOffset"`
	Cursor                           string                        `json:"cursor"`
	NextCursor                       string                        `json:"nextCursor"`
	HasMore                          bool                          `json:"hasMore"`
	Truncated                        bool                          `json:"truncated"`
	SourceBreakdown                  []CmdbAssetRelationMetricResp `json:"sourceBreakdown"`
	TypeBreakdown                    []CmdbAssetRelationMetricResp `json:"typeBreakdown"`
}

type CmdbAssetRelationMetricResp struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type CmdbAssetChangeResp struct {
	models.CmdbAssetChange
}

type CloudAssetSecurityRulesResp struct {
	AssetId         models.Id                    `json:"assetId"`
	Provider        string                       `json:"provider"`
	AssetType       string                       `json:"assetType"`
	NativeType      string                       `json:"nativeType"`
	RuleCount       int                          `json:"ruleCount"`
	PublicRuleCount int                          `json:"publicRuleCount"`
	Rules           []CloudAssetSecurityRuleResp `json:"rules"`
}

type CloudAssetSecurityRuleResp struct {
	Direction      string          `json:"direction"`
	Protocol       string          `json:"protocol"`
	Source         string          `json:"source"`
	Destination    string          `json:"destination"`
	PortRange      string          `json:"portRange"`
	Description    string          `json:"description"`
	PublicExposure bool            `json:"publicExposure"`
	Raw            models.ResAttrs `json:"raw,omitempty"`
}

type CmdbAssetFilterResp struct {
	Projects   []OrgProjectResp `json:"projects"`
	Envs       []EnvResp        `json:"envs"`
	Providers  []string         `json:"providers"`
	AccountIds []string         `json:"accountIds"`
	AssetTypes []string         `json:"assetTypes"`
	Sources    []string         `json:"sources"`
	Statuses   []string         `json:"statuses"`
	ManagedBy  []string         `json:"managedBy"`
}

type CmdbAssetPermissionResp struct {
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	Allowed       bool     `json:"allowed"`
	Message       string   `json:"message"`
	RequiredRoles []string `json:"requiredRoles"`
}

type CmdbAssetProjectRoleResp struct {
	ProjectId   models.Id `json:"projectId"`
	ProjectName string    `json:"projectName"`
	Role        string    `json:"role"`
}

type CmdbAssetPermissionsResp struct {
	OrgId                         models.Id                  `json:"orgId"`
	UserId                        models.Id                  `json:"userId"`
	IsSuperAdmin                  bool                       `json:"isSuperAdmin"`
	OrgRole                       string                     `json:"orgRole"`
	ProjectRoles                  []CmdbAssetProjectRoleResp `json:"projectRoles"`
	CanManageAll                  bool                       `json:"canManageAll"`
	CanExport                     bool                       `json:"canExport"`
	CanImport                     bool                       `json:"canImport"`
	CanEditOwnership              bool                       `json:"canEditOwnership"`
	CanBatchGovernance            bool                       `json:"canBatchGovernance"`
	CanManageApplicationRelations bool                       `json:"canManageApplicationRelations"`
	CanManageRiskRules            bool                       `json:"canManageRiskRules"`
	CanSyncIac                    bool                       `json:"canSyncIac"`
	Permissions                   []CmdbAssetPermissionResp  `json:"permissions"`
}

type CmdbAssetGovernanceReportResp struct {
	TotalAssets                 int64                              `json:"totalAssets"`
	TotalCost                   float64                            `json:"totalCost"`
	AverageRiskScore            float64                            `json:"averageRiskScore"`
	MissingOwnerAssets          int64                              `json:"missingOwnerAssets"`
	MissingApplicationAssets    int64                              `json:"missingApplicationAssets"`
	MissingBusinessLineAssets   int64                              `json:"missingBusinessLineAssets"`
	MissingCostCenterAssets     int64                              `json:"missingCostCenterAssets"`
	UnclassifiedLifecycleAssets int64                              `json:"unclassifiedLifecycleAssets"`
	HighRiskAssets              int64                              `json:"highRiskAssets"`
	CriticalRiskAssets          int64                              `json:"criticalRiskAssets"`
	MaintenanceAssets           int64                              `json:"maintenanceAssets"`
	RetiredAssets               int64                              `json:"retiredAssets"`
	UnassignedProjectAssets     int64                              `json:"unassignedProjectAssets"`
	CostByProvider              []CmdbAssetGovernanceBreakdownResp `json:"costByProvider"`
	CostByBusinessLine          []CmdbAssetGovernanceBreakdownResp `json:"costByBusinessLine"`
	CostByApplication           []CmdbAssetGovernanceBreakdownResp `json:"costByApplication"`
	CostByOwner                 []CmdbAssetGovernanceBreakdownResp `json:"costByOwner"`
	LifecycleBreakdown          []CmdbAssetGovernanceBreakdownResp `json:"lifecycleBreakdown"`
	ComplianceRiskBreakdown     []CmdbAssetGovernanceBreakdownResp `json:"complianceRiskBreakdown"`
	TopCostAssets               []CmdbAssetGovernanceAssetResp     `json:"topCostAssets"`
	TopRiskAssets               []CmdbAssetGovernanceAssetResp     `json:"topRiskAssets"`
}

type CmdbAssetGovernanceBreakdownResp struct {
	Name  string  `json:"name"`
	Count int64   `json:"count"`
	Cost  float64 `json:"cost"`
}

type CmdbAssetGovernanceAssetResp struct {
	Id             models.Id `json:"id"`
	Name           string    `json:"name"`
	Provider       string    `json:"provider"`
	Region         string    `json:"region"`
	AssetType      string    `json:"assetType"`
	NativeId       string    `json:"nativeId"`
	Owner          string    `json:"owner"`
	Application    string    `json:"application"`
	BusinessLine   string    `json:"businessLine"`
	Lifecycle      string    `json:"lifecycle"`
	ComplianceRisk string    `json:"complianceRisk"`
	Cost           float64   `json:"cost"`
	RiskScore      float64   `json:"riskScore"`
	ManagedBy      string    `json:"managedBy"`
}

type CmdbBackfillResp struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
}

type CmdbImportResp struct {
	Total            int                  `json:"total"`
	Created          int                  `json:"created"`
	Updated          int                  `json:"updated"`
	Skipped          int                  `json:"skipped"`
	OwnershipUpdated int                  `json:"ownershipUpdated"`
	DryRun           bool                 `json:"dryRun"`
	Errors           []string             `json:"errors"`
	Items            []CmdbImportItemResp `json:"items,omitempty"`
}

type CmdbImportItemResp struct {
	Index            int             `json:"index"`
	Action           string          `json:"action"`
	AssetId          models.Id       `json:"assetId"`
	Provider         string          `json:"provider"`
	AccountId        string          `json:"accountId"`
	Region           string          `json:"region"`
	AssetType        string          `json:"assetType"`
	NativeType       string          `json:"nativeType"`
	NativeId         string          `json:"nativeId"`
	Name             string          `json:"name"`
	ExistingName     string          `json:"existingName,omitempty"`
	OwnershipChanged bool            `json:"ownershipChanged"`
	Diff             models.ResAttrs `json:"diff,omitempty"`
	OwnershipDiff    models.ResAttrs `json:"ownershipDiff,omitempty"`
	Error            string          `json:"error,omitempty"`
}

type CmdbBatchOwnershipResp struct {
	Total       int       `json:"total"`
	Updated     int       `json:"updated"`
	Skipped     int       `json:"skipped"`
	OperationId models.Id `json:"operationId"`
	Errors      []string  `json:"errors"`
}

type CmdbApplicationResp struct {
	Application       string                        `json:"application"`
	Owner             []string                      `json:"owner"`
	BusinessLine      []string                      `json:"businessLine"`
	AssetCount        int                           `json:"assetCount"`
	IncomingAppCount  int                           `json:"incomingAppCount"`
	OutgoingAppCount  int                           `json:"outgoingAppCount"`
	ImpactedAppCount  int                           `json:"impactedAppCount"`
	RecentChangeCount int                           `json:"recentChangeCount"`
	ChangedAssetCount int                           `json:"changedAssetCount"`
	RiskLevel         string                        `json:"riskLevel"`
	RiskScore         int                           `json:"riskScore"`
	RiskReason        string                        `json:"riskReason"`
	Assets            []CmdbApplicationAssetResp    `json:"assets,omitempty"`
	Upstreams         []CmdbApplicationRelationResp `json:"upstreams,omitempty"`
	Downstreams       []CmdbApplicationRelationResp `json:"downstreams,omitempty"`
	ImpactedApps      []CmdbApplicationImpactResp   `json:"impactedApplications,omitempty"`
	RecentChanges     []CmdbApplicationChangeResp   `json:"recentChanges,omitempty"`
}

type CmdbApplicationAssetResp struct {
	Id             models.Id   `json:"id"`
	Name           string      `json:"name"`
	AssetType      string      `json:"assetType"`
	Provider       string      `json:"provider"`
	NativeId       string      `json:"nativeId"`
	Status         string      `json:"status"`
	Owner          string      `json:"owner"`
	BusinessLine   string      `json:"businessLine"`
	Lifecycle      string      `json:"lifecycle"`
	ComplianceRisk string      `json:"complianceRisk"`
	UpdatedAt      models.Time `json:"updatedAt"`
}

type CmdbApplicationRelationResp struct {
	Application        string      `json:"application"`
	Direction          string      `json:"direction"`
	RelationType       string      `json:"relationType"`
	Source             string      `json:"source"`
	AssetRelationCount int         `json:"assetRelationCount"`
	SourceAssetCount   int         `json:"sourceAssetCount"`
	TargetAssetCount   int         `json:"targetAssetCount"`
	LatestRelationAt   models.Time `json:"latestRelationAt"`
}

type CmdbApplicationImpactResp struct {
	Application      string      `json:"application"`
	RelationSources  []string    `json:"relationSources"`
	RelationCount    int         `json:"relationCount"`
	SourceAssetCount int         `json:"sourceAssetCount"`
	TargetAssetCount int         `json:"targetAssetCount"`
	LatestRelationAt models.Time `json:"latestRelationAt"`
}

type CmdbApplicationChangeResp struct {
	Id         models.Id       `json:"id"`
	AssetId    models.Id       `json:"assetId"`
	AssetName  string          `json:"assetName"`
	ChangeType string          `json:"changeType"`
	Source     string          `json:"source"`
	Summary    string          `json:"summary"`
	Diff       models.ResAttrs `json:"diff,omitempty"`
	CreatedAt  models.Time     `json:"createdAt"`
}

type CmdbCloudAccountResp struct {
	Id                    models.Id   `json:"id"`
	Source                string      `json:"source"`
	Name                  string      `json:"name"`
	Description           string      `json:"description"`
	Provider              string      `json:"provider"`
	AccountId             string      `json:"accountId"`
	Regions               []string    `json:"regions"`
	Ready                 bool        `json:"ready"`
	MissingCredentialKeys []string    `json:"missingCredentialKeys"`
	SupportedAssetTypes   []string    `json:"supportedAssetTypes"`
	UpdatedAt             models.Time `json:"updatedAt"`
}

type CmdbSyncTaskResp struct {
	models.CmdbSyncTask

	ScopeSummary   CmdbSyncTaskScopeSummaryResp   `json:"scopeSummary" gorm:"-"`
	FailureSummary CmdbSyncTaskFailureSummaryResp `json:"failureSummary" gorm:"-"`
}

type CmdbSyncTaskBriefResp struct {
	Id             models.Id                      `json:"id"`
	AccountName    string                         `json:"accountName"`
	AccountSource  string                         `json:"accountSource"`
	AccountId      models.Id                      `json:"accountId"`
	SyncPolicyId   models.Id                      `json:"syncPolicyId"`
	Provider       string                         `json:"provider"`
	Regions        []string                       `json:"regions"`
	AssetTypes     []string                       `json:"assetTypes"`
	Status         string                         `json:"status"`
	ErrorMessage   string                         `json:"errorMessage"`
	Stats          models.ResAttrs                `json:"stats,omitempty"`
	ScopeSummary   CmdbSyncTaskScopeSummaryResp   `json:"scopeSummary"`
	FailureSummary CmdbSyncTaskFailureSummaryResp `json:"failureSummary"`
	StartedAt      models.Time                    `json:"startedAt"`
	EndedAt        models.Time                    `json:"endedAt"`
	CreatedAt      models.Time                    `json:"createdAt"`
}

type CmdbSyncTaskScopeSummaryResp struct {
	Regions             []string                 `json:"regions"`
	AssetTypes          []string                 `json:"assetTypes"`
	RegionCount         int                      `json:"regionCount"`
	AssetTypeCount      int                      `json:"assetTypeCount"`
	ScopeCount          int                      `json:"scopeCount"`
	Collected           int64                    `json:"collected"`
	Created             int64                    `json:"created"`
	Updated             int64                    `json:"updated"`
	Skipped             int64                    `json:"skipped"`
	DurationMs          int64                    `json:"durationMs"`
	CollectorDurationMs int64                    `json:"collectorDurationMs"`
	UpsertDurationMs    int64                    `json:"upsertDurationMs"`
	RelationDurationMs  int64                    `json:"relationDurationMs"`
	RegionMetrics       []CmdbSyncTaskMetricResp `json:"regionMetrics"`
	AssetTypeMetrics    []CmdbSyncTaskMetricResp `json:"assetTypeMetrics"`
	ScopeMetrics        []CmdbSyncTaskMetricResp `json:"scopeMetrics"`
	FailedScopes        []CmdbSyncTaskMetricResp `json:"failedScopes"`
}

type CmdbSyncTaskMetricResp struct {
	Region     string `json:"region"`
	AssetType  string `json:"assetType"`
	Status     string `json:"status"`
	Collected  int64  `json:"collected"`
	DurationMs int64  `json:"durationMs"`
}

type CmdbSyncTaskFailureSummaryResp struct {
	Total          int64                           `json:"total"`
	RetryableTotal int64                           `json:"retryableTotal"`
	Retryable      bool                            `json:"retryable"`
	Categories     []string                        `json:"categories"`
	FirstMessage   string                          `json:"firstMessage"`
	RetryHint      string                          `json:"retryHint"`
	Details        []CmdbSyncTaskFailureDetailResp `json:"details"`
}

type CmdbSyncTaskFailureDetailResp struct {
	Message   string `json:"message"`
	Category  string `json:"category"`
	Retryable bool   `json:"retryable"`
	RetryHint string `json:"retryHint"`
}

type CmdbSyncTaskPageResp struct {
	Total    int64                   `json:"total"`
	PageSize int                     `json:"pageSize"`
	List     []CmdbSyncTaskResp      `json:"list"`
	Summary  CmdbSyncTaskSummaryResp `json:"summary"`
}

type CmdbSyncTaskBatchRerunResp struct {
	GroupId models.Id                     `json:"groupId"`
	Total   int                           `json:"total"`
	Created int                           `json:"created"`
	Items   []CmdbSyncTaskBatchRerunItem  `json:"items"`
	Errors  []CmdbSyncTaskBatchRerunError `json:"errors"`
}

type CmdbSyncTaskBatchRerunItem struct {
	SourceTaskId models.Id `json:"sourceTaskId"`
	TaskId       models.Id `json:"taskId"`
	Status       string    `json:"status"`
}

type CmdbSyncTaskBatchRerunError struct {
	TaskId  models.Id `json:"taskId"`
	Message string    `json:"message"`
}

type CmdbSyncTaskRerunGroupResp struct {
	GroupId        models.Id                    `json:"groupId"`
	Reason         string                       `json:"reason"`
	Mode           string                       `json:"mode"`
	Approval       models.ResAttrs              `json:"approval,omitempty"`
	Total          int                          `json:"total"`
	ApprovingCount int                          `json:"approvingCount"`
	PendingCount   int                          `json:"pendingCount"`
	RunningCount   int                          `json:"runningCount"`
	CompleteCount  int                          `json:"completeCount"`
	FailedCount    int                          `json:"failedCount"`
	RejectedCount  int                          `json:"rejectedCount"`
	CreatedAt      models.Time                  `json:"createdAt"`
	StartedAt      models.Time                  `json:"startedAt"`
	EndedAt        models.Time                  `json:"endedAt"`
	TaskIds        []models.Id                  `json:"taskIds"`
	SourceTaskIds  []models.Id                  `json:"sourceTaskIds"`
	Tasks          []CmdbSyncTaskRerunGroupTask `json:"tasks"`
}

type CmdbSyncTaskRerunGroupTask struct {
	CmdbSyncTaskResp

	SourceTaskId     models.Id `json:"sourceTaskId"`
	SourceTaskStatus string    `json:"sourceTaskStatus"`
}

type CmdbSyncTaskSummaryResp struct {
	TotalCount               int64                    `json:"totalCount"`
	CompleteCount            int64                    `json:"completeCount"`
	FailedCount              int64                    `json:"failedCount"`
	RejectedCount            int64                    `json:"rejectedCount"`
	ApprovingCount           int64                    `json:"approvingCount"`
	RunningCount             int64                    `json:"runningCount"`
	PendingCount             int64                    `json:"pendingCount"`
	SuccessRate              float64                  `json:"successRate"`
	FailureRate              float64                  `json:"failureRate"`
	LastSuccessAt            models.Time              `json:"lastSuccessAt"`
	LastFailureAt            models.Time              `json:"lastFailureAt"`
	TrendDays                int                      `json:"trendDays"`
	TrendStartDate           string                   `json:"trendStartDate"`
	TrendEndDate             string                   `json:"trendEndDate"`
	TrendCustomRange         bool                     `json:"trendCustomRange"`
	Trend                    []CmdbSyncTaskTrendPoint `json:"trend"`
	FailureThreshold         float64                  `json:"failureThreshold"`
	FailureThresholdExceeded bool                     `json:"failureThresholdExceeded"`
	FailureAlertLevel        string                   `json:"failureAlertLevel"`
	FailureAlertMessage      string                   `json:"failureAlertMessage"`
	Regions                  []CmdbSyncTaskBreakdown  `json:"regions"`
	AssetTypes               []CmdbSyncTaskBreakdown  `json:"assetTypes"`
	ApiMetrics               []CmdbSyncTaskApiMetric  `json:"apiMetrics"`
	LastSuccessTask          *CmdbSyncTaskBriefResp   `json:"lastSuccessTask,omitempty"`
	LastFailureTask          *CmdbSyncTaskBriefResp   `json:"lastFailureTask,omitempty"`
}

type CmdbSyncTaskTrendPoint struct {
	Date          string `json:"date"`
	TotalCount    int64  `json:"totalCount"`
	CompleteCount int64  `json:"completeCount"`
	FailedCount   int64  `json:"failedCount"`
}

type CmdbSyncTaskBreakdown struct {
	Key           string  `json:"key"`
	Name          string  `json:"name"`
	TaskCount     int64   `json:"taskCount"`
	CompleteCount int64   `json:"completeCount"`
	FailedCount   int64   `json:"failedCount"`
	RunningCount  int64   `json:"runningCount"`
	PendingCount  int64   `json:"pendingCount"`
	Collected     int64   `json:"collected"`
	FailureRate   float64 `json:"failureRate"`
}

type CmdbSyncTaskApiMetric struct {
	Key           string                       `json:"key"`
	Provider      string                       `json:"provider"`
	Region        string                       `json:"region"`
	Service       string                       `json:"service"`
	Path          string                       `json:"path"`
	CallCount     int64                        `json:"callCount"`
	FailedCount   int64                        `json:"failedCount"`
	RetriedCount  int64                        `json:"retriedCount"`
	AvgDurationMs int64                        `json:"avgDurationMs"`
	MaxDurationMs int64                        `json:"maxDurationMs"`
	FailureRate   float64                      `json:"failureRate"`
	Trend         []CmdbSyncTaskApiMetricTrend `json:"trend"`
}

type CmdbSyncTaskApiMetricTrend struct {
	Date          string `json:"date"`
	CallCount     int64  `json:"callCount"`
	FailedCount   int64  `json:"failedCount"`
	RetriedCount  int64  `json:"retriedCount"`
	AvgDurationMs int64  `json:"avgDurationMs"`
	MaxDurationMs int64  `json:"maxDurationMs"`
}

type CmdbSyncTaskLogResp struct {
	models.CmdbSyncTaskLog
}

type CmdbSyncTaskDetailResp struct {
	CmdbSyncTaskResp

	Logs []CmdbSyncTaskLogResp `json:"logs"`
}
