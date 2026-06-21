// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CmdbAssetResp struct {
	models.CmdbAsset

	ProjectName string `json:"projectName"`
	EnvName     string `json:"envName"`

	Relations []CmdbAssetRelationResp `json:"relations,omitempty" gorm:"-"`
	Changes   []CmdbAssetChangeResp   `json:"changes,omitempty" gorm:"-"`
}

type CmdbAssetRelationResp struct {
	models.CmdbAssetRelation

	SourceAssetName string `json:"sourceAssetName"`
	SourceAssetType string `json:"sourceAssetType"`
	TargetAssetName string `json:"targetAssetName"`
	TargetAssetType string `json:"targetAssetType"`
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

type CmdbBackfillResp struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
}

type CmdbImportResp struct {
	Total            int      `json:"total"`
	Created          int      `json:"created"`
	Updated          int      `json:"updated"`
	Skipped          int      `json:"skipped"`
	OwnershipUpdated int      `json:"ownershipUpdated"`
	Errors           []string `json:"errors"`
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
	models.CmdbSyncTask

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

type CmdbSyncTaskLogResp struct {
	models.CmdbSyncTaskLog
}

type CmdbSyncTaskDetailResp struct {
	models.CmdbSyncTask

	Logs []CmdbSyncTaskLogResp `json:"logs"`
}
