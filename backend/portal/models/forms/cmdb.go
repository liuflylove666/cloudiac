// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package forms

import "cloudiac/portal/models"

type SearchCmdbAssetForm struct {
	PageForm

	Q          string `form:"q" json:"q"`
	ProjectIds string `form:"projectIds" json:"projectIds"`
	EnvIds     string `form:"envIds" json:"envIds"`
	Providers  string `form:"providers" json:"providers"`
	AccountIds string `form:"accountIds" json:"accountIds"`
	AssetTypes string `form:"assetTypes" json:"assetTypes"`
	Sources    string `form:"sources" json:"sources"`
	Statuses   string `form:"statuses" json:"statuses"`
	ManagedBy  string `form:"managedBy" json:"managedBy"`
	Dsl        string `form:"dsl" json:"dsl"`
}

type ExportCmdbAssetForm struct {
	SearchCmdbAssetForm

	Format string      `form:"format" json:"format" binding:"omitempty,oneof=csv json"`
	Ids    []models.Id `form:"ids" json:"ids"`
}

type ImportCmdbAssetForm struct {
	BaseForm

	Assets             []models.CmdbAsset `json:"assets" binding:"required"`
	OverwriteOwnership bool               `json:"overwriteOwnership" form:"overwriteOwnership"`
	DryRun             bool               `json:"dryRun" form:"dryRun"`
}

type SearchCmdbApplicationForm struct {
	PageForm

	Q    string `form:"q" json:"q"`
	Risk string `form:"risk" json:"risk"`
}

type CmdbApplicationDetailForm struct {
	BaseForm

	Application string `form:"application" json:"application" binding:"required,max=128"`
}

type UpdateCmdbApplicationRelationsForm struct {
	BaseForm

	Application string   `json:"application" form:"application" binding:"required,max=128"`
	Upstreams   []string `json:"upstreams" form:"upstreams" binding:"dive,max=128"`
	Downstreams []string `json:"downstreams" form:"downstreams" binding:"dive,max=128"`
}

type UpdateCmdbRiskRuleConfigForm struct {
	BaseForm

	ChangeWindowDays int `json:"changeWindowDays" form:"changeWindowDays" binding:"omitempty,gte=1,lte=90"`

	RecentChangeWeight           int `json:"recentChangeWeight" form:"recentChangeWeight" binding:"omitempty,gte=0,lte=100"`
	ChangedAssetWeight           int `json:"changedAssetWeight" form:"changedAssetWeight" binding:"omitempty,gte=0,lte=100"`
	IncomingAppWeight            int `json:"incomingAppWeight" form:"incomingAppWeight" binding:"omitempty,gte=0,lte=100"`
	OutgoingAppWeight            int `json:"outgoingAppWeight" form:"outgoingAppWeight" binding:"omitempty,gte=0,lte=100"`
	HighComplianceRiskWeight     int `json:"highComplianceRiskWeight" form:"highComplianceRiskWeight" binding:"omitempty,gte=0,lte=100"`
	CriticalComplianceRiskWeight int `json:"criticalComplianceRiskWeight" form:"criticalComplianceRiskWeight" binding:"omitempty,gte=0,lte=100"`
	MaintenanceLifecycleWeight   int `json:"maintenanceLifecycleWeight" form:"maintenanceLifecycleWeight" binding:"omitempty,gte=0,lte=100"`
	RetiredLifecycleWeight       int `json:"retiredLifecycleWeight" form:"retiredLifecycleWeight" binding:"omitempty,gte=0,lte=100"`
	CrossBusinessLineWeight      int `json:"crossBusinessLineWeight" form:"crossBusinessLineWeight" binding:"omitempty,gte=0,lte=100"`

	CriticalIncomingThreshold int `json:"criticalIncomingThreshold" form:"criticalIncomingThreshold" binding:"omitempty,gte=1,lte=100"`
	MediumIncomingThreshold   int `json:"mediumIncomingThreshold" form:"mediumIncomingThreshold" binding:"omitempty,gte=1,lte=100"`
	MediumOutgoingThreshold   int `json:"mediumOutgoingThreshold" form:"mediumOutgoingThreshold" binding:"omitempty,gte=1,lte=100"`

	CriticalScoreThreshold int `json:"criticalScoreThreshold" form:"criticalScoreThreshold" binding:"omitempty,gte=1,lte=1000"`
	HighScoreThreshold     int `json:"highScoreThreshold" form:"highScoreThreshold" binding:"omitempty,gte=1,lte=1000"`
	MediumScoreThreshold   int `json:"mediumScoreThreshold" form:"mediumScoreThreshold" binding:"omitempty,gte=1,lte=1000"`

	RecentCriticalBoost int `json:"recentCriticalBoost" form:"recentCriticalBoost" binding:"omitempty,gte=0,lte=1000"`
	RecentHighBoost     int `json:"recentHighBoost" form:"recentHighBoost" binding:"omitempty,gte=0,lte=1000"`
	RecentMediumBoost   int `json:"recentMediumBoost" form:"recentMediumBoost" binding:"omitempty,gte=0,lte=1000"`
	WideDependencyBoost int `json:"wideDependencyBoost" form:"wideDependencyBoost" binding:"omitempty,gte=0,lte=1000"`
}

type CmdbAssetParam struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type CmdbAssetRelationsForm struct {
	CmdbAssetParam

	Limit              int    `form:"limit" json:"limit" binding:"omitempty,gte=1,lte=1000"`
	Offset             int    `form:"offset" json:"offset" binding:"omitempty,gte=0,lte=100000"`
	Cursor             string `form:"cursor" json:"cursor" binding:"omitempty,max=64"`
	IncludeApplication *bool  `form:"includeApplication" json:"includeApplication"`
	Sources            string `form:"sources" json:"sources" binding:"omitempty,max=255"`
	RelationTypes      string `form:"relationTypes" json:"relationTypes" binding:"omitempty,max=255"`
	Direction          string `form:"direction" json:"direction" binding:"omitempty,oneof=all incoming outgoing"`
	Keyword            string `form:"keyword" json:"keyword" binding:"omitempty,max=255"`
}

type UpdateCmdbAssetOwnershipForm struct {
	BaseForm

	Id             models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	Owner          string    `json:"owner" form:"owner" binding:"max=128"`
	Application    string    `json:"application" form:"application" binding:"max=128"`
	BusinessLine   string    `json:"businessLine" form:"businessLine" binding:"max=128"`
	Lifecycle      string    `json:"lifecycle" form:"lifecycle" binding:"omitempty,oneof=planned active maintenance retired"`
	Cost           float64   `json:"cost" form:"cost" binding:"omitempty,gte=0"`
	ComplianceRisk string    `json:"complianceRisk" form:"complianceRisk" binding:"omitempty,oneof=low medium high critical"`
}

type BatchUpdateCmdbAssetOwnershipForm struct {
	BaseForm

	Ids            []models.Id `json:"ids" form:"ids" binding:"required,min=1,dive,max=32"`
	ProjectId      models.Id   `json:"projectId" form:"projectId" binding:"max=32"`
	EnvId          models.Id   `json:"envId" form:"envId" binding:"max=32"`
	Owner          string      `json:"owner" form:"owner" binding:"max=128"`
	Application    string      `json:"application" form:"application" binding:"max=128"`
	BusinessLine   string      `json:"businessLine" form:"businessLine" binding:"max=128"`
	Lifecycle      string      `json:"lifecycle" form:"lifecycle" binding:"omitempty,oneof=planned active maintenance retired"`
	ComplianceRisk string      `json:"complianceRisk" form:"complianceRisk" binding:"omitempty,oneof=low medium high critical"`
}

type SearchCmdbSyncTaskForm struct {
	PageForm

	Provider              string    `form:"provider" json:"provider"`
	AccountSource         string    `form:"accountSource" json:"accountSource"`
	AccountId             models.Id `form:"accountId" json:"accountId"`
	SyncPolicyId          models.Id `form:"syncPolicyId" json:"syncPolicyId"`
	SyncPolicyScheduleKey string    `form:"syncPolicyScheduleKey" json:"syncPolicyScheduleKey"`
	TrendDays             int       `form:"trendDays" json:"trendDays"`
	TrendStartDate        string    `form:"trendStartDate" json:"trendStartDate"`
	TrendEndDate          string    `form:"trendEndDate" json:"trendEndDate"`
	FailureThreshold      float64   `form:"failureThreshold" json:"failureThreshold"`
	Status                string    `form:"status" json:"status"`
}

type CmdbSyncTaskParam struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type CmdbSyncTaskRerunGroupParam struct {
	BaseForm

	GroupId models.Id `uri:"groupId" json:"groupId" binding:"required,max=32" swaggerignore:"true"`
}

type CmdbSyncTaskRerunGroupApprovalForm struct {
	BaseForm

	GroupId models.Id `uri:"groupId" json:"groupId" binding:"required,max=32" swaggerignore:"true"`
	Action  string    `form:"action" json:"action" binding:"required,oneof=approved rejected"`
	Comment string    `form:"comment" json:"comment" binding:"max=255"`
}

type BatchRerunFailedCmdbSyncTasksForm struct {
	BaseForm

	TaskIds          []models.Id `json:"taskIds" form:"taskIds" binding:"required,min=1,max=50,dive,max=32"`
	Reason           string      `json:"reason" form:"reason" binding:"required,max=255"`
	Regions          []string    `json:"regions" form:"regions"`
	AssetTypes       []string    `json:"assetTypes" form:"assetTypes"`
	RequiresApproval bool        `json:"requiresApproval" form:"requiresApproval"`
}

type CreateCmdbSyncTaskForm struct {
	BaseForm

	AccountSource          string    `json:"accountSource" form:"accountSource" binding:"required,oneof=variable_group resource_account cloud_account"`
	AccountId              models.Id `json:"accountId" form:"accountId" binding:"required,max=32"`
	SyncPolicyId           models.Id `json:"syncPolicyId" form:"syncPolicyId" binding:"max=32"`
	SyncPolicyScheduleKey  string    `json:"syncPolicyScheduleKey" form:"syncPolicyScheduleKey" binding:"max=64"`
	SyncPolicyScheduleName string    `json:"syncPolicyScheduleName" form:"syncPolicyScheduleName" binding:"max=128"`
	SlowApiThresholdMs     int64     `json:"slowApiThresholdMs" form:"slowApiThresholdMs" binding:"omitempty,gte=1,lte=600000"`
	SlowApiSilenceMinutes  int64     `json:"slowApiSilenceMinutes" form:"slowApiSilenceMinutes" binding:"omitempty,gte=0,lte=10080"`
	Reason                 string    `json:"reason" form:"reason" binding:"max=255"`
	Provider               string    `json:"provider" form:"provider" binding:"omitempty,oneof=aws oci oracle alicloud azure gcp tencentcloud huawei"`
	Regions                []string  `json:"regions" form:"regions"`
	AssetTypes             []string  `json:"assetTypes" form:"assetTypes"`
}
