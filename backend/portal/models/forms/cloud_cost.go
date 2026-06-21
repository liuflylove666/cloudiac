// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package forms

import "cloudiac/portal/models"

type SearchCloudCostRecordForm struct {
	PageForm

	Q            string    `form:"q" json:"q"`
	Provider     string    `form:"provider" json:"provider"`
	AccountId    string    `form:"accountId" json:"accountId"`
	Region       string    `form:"region" json:"region"`
	Service      string    `form:"service" json:"service"`
	Period       string    `form:"period" json:"period"`
	Currency     string    `form:"currency" json:"currency"`
	Source       string    `form:"source" json:"source" binding:"omitempty,oneof=bill cmdb_asset import azure_cost_management azure_billing_export gcp_billing_export aws_cur_export aws_cost_explorer_export oci_usage_cost_export tencentcloud_billing_export huawei_billing_export"`
	CostCenter   string    `form:"costCenter" json:"costCenter"`
	Application  string    `form:"application" json:"application"`
	BusinessLine string    `form:"businessLine" json:"businessLine"`
	MatchedAsset *bool     `form:"matchedAsset" json:"matchedAsset"`
	AssetId      models.Id `form:"assetId" json:"assetId" binding:"max=32"`
	ProjectId    models.Id `form:"projectId" json:"projectId" binding:"max=32"`
	EnvId        models.Id `form:"envId" json:"envId" binding:"max=32"`
}

type ImportCloudCostRecordItem struct {
	ProjectId      models.Id       `form:"projectId" json:"projectId" binding:"max=32"`
	EnvId          models.Id       `form:"envId" json:"envId" binding:"max=32"`
	CloudAccountId models.Id       `form:"cloudAccountId" json:"cloudAccountId" binding:"max=32"`
	Provider       string          `form:"provider" json:"provider" binding:"max=64"`
	AccountId      string          `form:"accountId" json:"accountId" binding:"max=128"`
	Region         string          `form:"region" json:"region" binding:"max=128"`
	Service        string          `form:"service" json:"service" binding:"max=128"`
	ResourceType   string          `form:"resourceType" json:"resourceType" binding:"max=64"`
	ResourceId     string          `form:"resourceId" json:"resourceId" binding:"max=255"`
	ResourceName   string          `form:"resourceName" json:"resourceName" binding:"max=255"`
	Amount         float64         `form:"amount" json:"amount"`
	Currency       string          `form:"currency" json:"currency" binding:"max=16"`
	Period         string          `form:"period" json:"period" binding:"max=32"`
	CostCenter     string          `form:"costCenter" json:"costCenter" binding:"max=128"`
	Owner          string          `form:"owner" json:"owner" binding:"max=128"`
	Application    string          `form:"application" json:"application" binding:"max=128"`
	BusinessLine   string          `form:"businessLine" json:"businessLine" binding:"max=128"`
	Source         string          `form:"source" json:"source" binding:"omitempty,oneof=import azure_cost_management azure_billing_export gcp_billing_export aws_cur_export aws_cost_explorer_export oci_usage_cost_export tencentcloud_billing_export huawei_billing_export"`
	SourceId       string          `form:"sourceId" json:"sourceId" binding:"max=128"`
	Payload        models.ResAttrs `form:"payload" json:"payload"`
}

type ImportCloudCostRecordForm struct {
	BaseForm

	Provider       string                      `form:"provider" json:"provider" binding:"omitempty,oneof=aws oci oracle alicloud azure gcp tencentcloud huawei"`
	CloudAccountId models.Id                   `form:"cloudAccountId" json:"cloudAccountId" binding:"max=32"`
	AccountId      string                      `form:"accountId" json:"accountId" binding:"max=128"`
	Region         string                      `form:"region" json:"region" binding:"max=128"`
	Period         string                      `form:"period" json:"period" binding:"max=32"`
	Currency       string                      `form:"currency" json:"currency" binding:"max=16"`
	Source         string                      `form:"source" json:"source" binding:"omitempty,oneof=import azure_cost_management azure_billing_export gcp_billing_export aws_cur_export aws_cost_explorer_export oci_usage_cost_export tencentcloud_billing_export huawei_billing_export"`
	Records        []ImportCloudCostRecordItem `form:"records" json:"records" binding:"required"`
}

type PullCloudCostRecordForm struct {
	BaseForm

	Provider             string          `form:"provider" json:"provider" binding:"required,oneof=aws oci azure gcp tencentcloud huawei"`
	CloudAccountId       models.Id       `form:"cloudAccountId" json:"cloudAccountId" binding:"max=32"`
	AccountId            string          `form:"accountId" json:"accountId" binding:"max=128"`
	Region               string          `form:"region" json:"region" binding:"max=128"`
	Period               string          `form:"period" json:"period" binding:"max=32"`
	Currency             string          `form:"currency" json:"currency" binding:"max=16"`
	Source               string          `form:"source" json:"source" binding:"omitempty,oneof=azure_cost_management azure_billing_export gcp_billing_export aws_cur_export aws_cost_explorer_export oci_usage_cost_export tencentcloud_billing_export huawei_billing_export"`
	SourceURL            string          `form:"sourceUrl" json:"sourceUrl" binding:"max=2048"`
	SourceIndexURL       string          `form:"sourceIndexUrl" json:"sourceIndexUrl" binding:"max=2048"`
	SourceObjectProvider string          `form:"sourceObjectProvider" json:"sourceObjectProvider" binding:"omitempty,oneof=s3 gcs azure_blob oci_object_storage"`
	SourceObjectEndpoint string          `form:"sourceObjectEndpoint" json:"sourceObjectEndpoint" binding:"max=2048"`
	SourceObjectBucket   string          `form:"sourceObjectBucket" json:"sourceObjectBucket" binding:"max=255"`
	SourceObjectPrefix   string          `form:"sourceObjectPrefix" json:"sourceObjectPrefix" binding:"max=1024"`
	SourceObjectBaseURL  string          `form:"sourceObjectBaseUrl" json:"sourceObjectBaseUrl" binding:"max=2048"`
	Cursor               string          `form:"cursor" json:"cursor" binding:"max=255"`
	MaxFiles             int             `form:"maxFiles" json:"maxFiles"`
	SourceHeaders        models.ResAttrs `form:"sourceHeaders" json:"sourceHeaders"`
	SyncTaskId           models.Id       `form:"syncTaskId" json:"syncTaskId" binding:"max=32" swaggerignore:"true"`
}

type SearchCloudCostSyncTaskForm struct {
	PageForm

	Q              string    `form:"q" json:"q"`
	Status         string    `form:"status" json:"status" binding:"omitempty,oneof=pending running complete failed"`
	Provider       string    `form:"provider" json:"provider" binding:"omitempty,oneof=aws oci azure gcp tencentcloud huawei"`
	Source         string    `form:"source" json:"source" binding:"omitempty,oneof=azure_cost_management azure_billing_export gcp_billing_export aws_cur_export aws_cost_explorer_export oci_usage_cost_export tencentcloud_billing_export huawei_billing_export"`
	CloudAccountId models.Id `form:"cloudAccountId" json:"cloudAccountId" binding:"max=32"`
	AccountId      string    `form:"accountId" json:"accountId" binding:"max=128"`
	Period         string    `form:"period" json:"period" binding:"max=32"`
}

type CloudCostSyncTaskParam struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type CreateCloudCostSyncTaskForm struct {
	PullCloudCostRecordForm
}

type SearchCloudCostSyncScheduleForm struct {
	PageForm

	Q              string    `form:"q" json:"q"`
	Status         string    `form:"status" json:"status" binding:"omitempty,oneof=enable disable"`
	Provider       string    `form:"provider" json:"provider" binding:"omitempty,oneof=aws oci azure gcp tencentcloud huawei"`
	Source         string    `form:"source" json:"source" binding:"omitempty,oneof=azure_cost_management azure_billing_export gcp_billing_export aws_cur_export aws_cost_explorer_export oci_usage_cost_export tencentcloud_billing_export huawei_billing_export"`
	CloudAccountId models.Id `form:"cloudAccountId" json:"cloudAccountId" binding:"max=32"`
	AccountId      string    `form:"accountId" json:"accountId" binding:"max=128"`
	Period         string    `form:"period" json:"period" binding:"max=32"`
}

type CloudCostSyncScheduleParam struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type CreateCloudCostSyncScheduleForm struct {
	BaseForm

	Name                 string          `form:"name" json:"name" binding:"required,max=128"`
	Description          string          `form:"description" json:"description" binding:"max=255"`
	Provider             string          `form:"provider" json:"provider" binding:"required,oneof=aws oci azure gcp tencentcloud huawei"`
	CloudAccountId       models.Id       `form:"cloudAccountId" json:"cloudAccountId" binding:"max=32"`
	AccountId            string          `form:"accountId" json:"accountId" binding:"max=128"`
	Region               string          `form:"region" json:"region" binding:"max=128"`
	Period               string          `form:"period" json:"period" binding:"max=32"`
	Currency             string          `form:"currency" json:"currency" binding:"max=16"`
	Source               string          `form:"source" json:"source" binding:"omitempty,oneof=azure_cost_management azure_billing_export gcp_billing_export aws_cur_export aws_cost_explorer_export oci_usage_cost_export tencentcloud_billing_export huawei_billing_export"`
	SourceURL            string          `form:"sourceUrl" json:"sourceUrl" binding:"max=2048"`
	SourceIndexURL       string          `form:"sourceIndexUrl" json:"sourceIndexUrl" binding:"max=2048"`
	SourceObjectProvider string          `form:"sourceObjectProvider" json:"sourceObjectProvider" binding:"omitempty,oneof=s3 gcs azure_blob oci_object_storage"`
	SourceObjectEndpoint string          `form:"sourceObjectEndpoint" json:"sourceObjectEndpoint" binding:"max=2048"`
	SourceObjectBucket   string          `form:"sourceObjectBucket" json:"sourceObjectBucket" binding:"max=255"`
	SourceObjectPrefix   string          `form:"sourceObjectPrefix" json:"sourceObjectPrefix" binding:"max=1024"`
	SourceObjectBaseURL  string          `form:"sourceObjectBaseUrl" json:"sourceObjectBaseUrl" binding:"max=2048"`
	Cursor               string          `form:"cursor" json:"cursor" binding:"max=255"`
	MaxFiles             int             `form:"maxFiles" json:"maxFiles"`
	SyncInterval         int             `form:"syncInterval" json:"syncInterval"`
	MaxRetryAttempts     int             `form:"maxRetryAttempts" json:"maxRetryAttempts"`
	RetryBackoffSeconds  int             `form:"retryBackoffSeconds" json:"retryBackoffSeconds"`
	NotifyOnFailure      bool            `form:"notifyOnFailure" json:"notifyOnFailure"`
	AutoPauseOnFailure   bool            `form:"autoPauseOnFailure" json:"autoPauseOnFailure"`
	Status               string          `form:"status" json:"status" binding:"omitempty,oneof=enable disable"`
	NextSyncAt           models.Time     `form:"nextSyncAt" json:"nextSyncAt"`
	Params               models.ResAttrs `form:"params" json:"params"`
}

type UpdateCloudCostSyncScheduleForm struct {
	CreateCloudCostSyncScheduleForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type RunDueCloudCostSyncScheduleForm struct {
	BaseForm

	Force bool `form:"force" json:"force"`
}

type CloudCostSummaryForm struct {
	BaseForm

	Period   string `form:"period" json:"period"`
	Currency string `form:"currency" json:"currency"`
}

type CloudCostTrendForm struct {
	BaseForm

	Provider string `form:"provider" json:"provider"`
	Currency string `form:"currency" json:"currency"`
	Months   int    `form:"months" json:"months"`
}

type SearchCloudCostInsightForm struct {
	PageForm

	Q            string    `form:"q" json:"q"`
	Type         string    `form:"type" json:"type" binding:"omitempty,oneof=anomaly optimization"`
	RuleKey      string    `form:"ruleKey" json:"ruleKey"`
	Severity     string    `form:"severity" json:"severity" binding:"omitempty,oneof=low medium high"`
	Status       string    `form:"status" json:"status" binding:"omitempty,oneof=open resolved ignored"`
	Period       string    `form:"period" json:"period"`
	Currency     string    `form:"currency" json:"currency"`
	Provider     string    `form:"provider" json:"provider"`
	AccountId    string    `form:"accountId" json:"accountId"`
	Region       string    `form:"region" json:"region"`
	ResourceType string    `form:"resourceType" json:"resourceType"`
	AssetId      models.Id `form:"assetId" json:"assetId" binding:"max=32"`
	ProjectId    models.Id `form:"projectId" json:"projectId" binding:"max=32"`
	EnvId        models.Id `form:"envId" json:"envId" binding:"max=32"`
}

type CloudCostInsightSummaryForm struct {
	BaseForm

	Period   string `form:"period" json:"period"`
	Currency string `form:"currency" json:"currency"`
}

type UpdateCloudCostInsightStatusForm struct {
	BaseForm

	Id     models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	Status string    `form:"status" json:"status" binding:"required,oneof=open resolved ignored"`
}
