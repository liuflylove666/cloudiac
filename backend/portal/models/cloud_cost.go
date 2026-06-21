// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package models

import "cloudiac/portal/libs/db"

const (
	CloudCostSourceBill                 = "bill"
	CloudCostSourceCmdbAsset            = "cmdb_asset"
	CloudCostSourceImport               = "import"
	CloudCostSourceAzureCostManagement  = "azure_cost_management"
	CloudCostSourceAzureBillingExport   = "azure_billing_export"
	CloudCostSourceGcpBillingExport     = "gcp_billing_export"
	CloudCostSourceAwsCurExport         = "aws_cur_export"
	CloudCostSourceAwsCostExplorer      = "aws_cost_explorer_export"
	CloudCostSourceOciUsageCostExport   = "oci_usage_cost_export"
	CloudCostSourceTencentBillingExport = "tencentcloud_billing_export"
	CloudCostSourceHuaweiBillingExport  = "huawei_billing_export"

	CloudCostSyncTaskStatusPending  = "pending"
	CloudCostSyncTaskStatusRunning  = "running"
	CloudCostSyncTaskStatusComplete = "complete"
	CloudCostSyncTaskStatusFailed   = "failed"

	CloudCostSyncScheduleStatusEnabled  = "enable"
	CloudCostSyncScheduleStatusDisabled = "disable"

	CloudCostInsightTypeAnomaly      = "anomaly"
	CloudCostInsightTypeOptimization = "optimization"

	CloudCostInsightSeverityLow    = "low"
	CloudCostInsightSeverityMedium = "medium"
	CloudCostInsightSeverityHigh   = "high"

	CloudCostInsightStatusOpen     = "open"
	CloudCostInsightStatusResolved = "resolved"
	CloudCostInsightStatusIgnored  = "ignored"
)

type CloudCostRecord struct {
	SoftDeleteModel

	OrgId          Id       `json:"orgId" gorm:"index;size:32;not null"`
	ProjectId      Id       `json:"projectId" gorm:"index;size:32;not null;default:''"`
	EnvId          Id       `json:"envId" gorm:"index;size:32;not null;default:''"`
	AssetId        Id       `json:"assetId" gorm:"index;size:32;not null;default:''"`
	CloudAccountId Id       `json:"cloudAccountId" gorm:"index;size:32;not null;default:''"`
	Provider       string   `json:"provider" gorm:"index;size:64;not null;default:''"`
	AccountId      string   `json:"accountId" gorm:"index;size:128;not null;default:''"`
	Region         string   `json:"region" gorm:"index;size:128;not null;default:''"`
	Service        string   `json:"service" gorm:"index;size:128;not null;default:''"`
	ResourceType   string   `json:"resourceType" gorm:"index;size:64;not null;default:''"`
	ResourceId     string   `json:"resourceId" gorm:"index;size:255;not null;default:''"`
	ResourceName   string   `json:"resourceName" gorm:"size:255;not null;default:''"`
	Amount         float64  `json:"amount" gorm:"type:decimal(18,4);not null;default:0"`
	Currency       string   `json:"currency" gorm:"index;size:16;not null;default:'CNY'"`
	Period         string   `json:"period" gorm:"index;size:32;not null;default:''"`
	CostCenter     string   `json:"costCenter" gorm:"index;size:128;not null;default:''"`
	Owner          string   `json:"owner" gorm:"index;size:128;not null;default:''"`
	Application    string   `json:"application" gorm:"index;size:128;not null;default:''"`
	BusinessLine   string   `json:"businessLine" gorm:"index;size:128;not null;default:''"`
	MatchedAsset   bool     `json:"matchedAsset" gorm:"index;not null;default:false"`
	Source         string   `json:"source" gorm:"index;size:64;not null;default:''"`
	SourceId       string   `json:"sourceId" gorm:"index;size:128;not null;default:''"`
	Fingerprint    string   `json:"fingerprint" gorm:"index;size:64;not null;default:''"`
	Payload        ResAttrs `json:"payload,omitempty" gorm:"type:json"`
}

func (CloudCostRecord) TableName() string {
	return "iac_cloud_cost_record"
}

func (r CloudCostRecord) Migrate(sess *db.Session) error {
	return r.AddUniqueIndex(sess, "unique_cloud_cost_fingerprint", "org_id", "fingerprint")
}

type CloudCostSyncTask struct {
	SoftDeleteModel

	OrgId          Id       `json:"orgId" gorm:"index;size:32;not null"`
	CreatorId      Id       `json:"creatorId" gorm:"index;size:32;not null;default:''"`
	CloudAccountId Id       `json:"cloudAccountId" gorm:"index;size:32;not null;default:''"`
	Provider       string   `json:"provider" gorm:"index;size:64;not null;default:''"`
	AccountId      string   `json:"accountId" gorm:"index;size:128;not null;default:''"`
	Region         string   `json:"region" gorm:"index;size:128;not null;default:''"`
	Source         string   `json:"source" gorm:"index;size:64;not null;default:''"`
	Mode           string   `json:"mode" gorm:"index;size:64;not null;default:''"`
	Period         string   `json:"period" gorm:"index;size:32;not null;default:''"`
	Currency       string   `json:"currency" gorm:"index;size:16;not null;default:'CNY'"`
	Status         string   `json:"status" gorm:"index;size:32;not null;default:'pending'"`
	Imported       int      `json:"imported" gorm:"not null;default:0"`
	MatchedCount   int      `json:"matchedCount" gorm:"not null;default:0"`
	UnmatchedCount int      `json:"unmatchedCount" gorm:"not null;default:0"`
	AttemptCount   int      `json:"attemptCount" gorm:"not null;default:0"`
	Message        string   `json:"message" gorm:"type:text"`
	Error          string   `json:"error" gorm:"type:text"`
	Params         ResAttrs `json:"params,omitempty" gorm:"type:json"`
	Result         ResAttrs `json:"result,omitempty" gorm:"type:json"`
	StartedAt      Time     `json:"startedAt" gorm:"type:datetime;default:null"`
	EndedAt        Time     `json:"endedAt" gorm:"type:datetime;default:null"`
}

func (CloudCostSyncTask) TableName() string {
	return "iac_cloud_cost_sync_task"
}

type CloudCostSyncTaskLog struct {
	TimedModel

	OrgId     Id       `json:"orgId" gorm:"index;size:32;not null"`
	TaskId    Id       `json:"taskId" gorm:"index;size:32;not null"`
	Stage     string   `json:"stage" gorm:"index;size:64;not null;default:''"`
	Status    string   `json:"status" gorm:"index;size:32;not null;default:'pending'"`
	Message   string   `json:"message" gorm:"type:text"`
	Error     string   `json:"error" gorm:"type:text"`
	Result    ResAttrs `json:"result,omitempty" gorm:"type:json"`
	StartedAt Time     `json:"startedAt" gorm:"type:datetime;default:null"`
	EndedAt   Time     `json:"endedAt" gorm:"type:datetime;default:null"`
}

func (CloudCostSyncTaskLog) TableName() string {
	return "iac_cloud_cost_sync_task_log"
}

type CloudCostSyncSchedule struct {
	SoftDeleteModel

	OrgId          Id       `json:"orgId" gorm:"index;size:32;not null"`
	CreatorId      Id       `json:"creatorId" gorm:"index;size:32;not null;default:''"`
	Name           string   `json:"name" gorm:"index;size:128;not null;default:''"`
	Description    string   `json:"description" gorm:"type:text"`
	CloudAccountId Id       `json:"cloudAccountId" gorm:"index;size:32;not null;default:''"`
	Provider       string   `json:"provider" gorm:"index;size:64;not null;default:''"`
	AccountId      string   `json:"accountId" gorm:"index;size:128;not null;default:''"`
	Region         string   `json:"region" gorm:"index;size:128;not null;default:''"`
	Source         string   `json:"source" gorm:"index;size:64;not null;default:''"`
	Period         string   `json:"period" gorm:"index;size:32;not null;default:''"`
	Currency       string   `json:"currency" gorm:"index;size:16;not null;default:'CNY'"`
	Status         string   `json:"status" gorm:"index;size:32;not null;default:'enable'"`
	SyncInterval   int      `json:"syncInterval" gorm:"not null;default:86400"`
	LastSyncTaskId Id       `json:"lastSyncTaskId" gorm:"index;size:32;not null;default:''"`
	LastSyncStatus string   `json:"lastSyncStatus" gorm:"index;size:32;not null;default:''"`
	LastError      string   `json:"lastError" gorm:"type:text"`
	LastSyncedAt   Time     `json:"lastSyncedAt" gorm:"type:datetime;default:null"`
	NextSyncAt     Time     `json:"nextSyncAt" gorm:"index;type:datetime;default:null"`
	Params         ResAttrs `json:"params,omitempty" gorm:"type:json"`
}

func (CloudCostSyncSchedule) TableName() string {
	return "iac_cloud_cost_sync_schedule"
}

type CloudCostInsight struct {
	SoftDeleteModel

	OrgId            Id       `json:"orgId" gorm:"index;size:32;not null"`
	ProjectId        Id       `json:"projectId" gorm:"index;size:32;not null;default:''"`
	EnvId            Id       `json:"envId" gorm:"index;size:32;not null;default:''"`
	AssetId          Id       `json:"assetId" gorm:"index;size:32;not null;default:''"`
	CloudAccountId   Id       `json:"cloudAccountId" gorm:"index;size:32;not null;default:''"`
	Type             string   `json:"type" gorm:"index;size:32;not null;default:'anomaly'"`
	RuleKey          string   `json:"ruleKey" gorm:"index;size:64;not null;default:''"`
	Severity         string   `json:"severity" gorm:"index;size:32;not null;default:'low'"`
	Status           string   `json:"status" gorm:"index;size:32;not null;default:'open'"`
	Provider         string   `json:"provider" gorm:"index;size:64;not null;default:''"`
	AccountId        string   `json:"accountId" gorm:"index;size:128;not null;default:''"`
	Region           string   `json:"region" gorm:"index;size:128;not null;default:''"`
	ResourceType     string   `json:"resourceType" gorm:"index;size:64;not null;default:''"`
	ResourceId       string   `json:"resourceId" gorm:"index;size:255;not null;default:''"`
	ResourceName     string   `json:"resourceName" gorm:"size:255;not null;default:''"`
	Period           string   `json:"period" gorm:"index;size:32;not null;default:''"`
	Currency         string   `json:"currency" gorm:"index;size:16;not null;default:'CNY'"`
	Amount           float64  `json:"amount" gorm:"type:decimal(18,4);not null;default:0"`
	PotentialSavings float64  `json:"potentialSavings" gorm:"type:decimal(18,4);not null;default:0"`
	Title            string   `json:"title" gorm:"size:255;not null;default:''"`
	Description      string   `json:"description" gorm:"type:text"`
	Recommendation   string   `json:"recommendation" gorm:"type:text"`
	Evidence         ResAttrs `json:"evidence,omitempty" gorm:"type:json"`
	Fingerprint      string   `json:"fingerprint" gorm:"index;size:64;not null;default:''"`
	FirstSeenAt      Time     `json:"firstSeenAt" gorm:"type:datetime;default:null"`
	LastSeenAt       Time     `json:"lastSeenAt" gorm:"type:datetime;default:null"`
	ResolvedAt       Time     `json:"resolvedAt" gorm:"type:datetime;default:null"`
}

func (CloudCostInsight) TableName() string {
	return "iac_cloud_cost_insight"
}

func (i CloudCostInsight) Migrate(sess *db.Session) error {
	return i.AddUniqueIndex(sess, "unique_cloud_cost_insight_fingerprint", "org_id", "fingerprint")
}
