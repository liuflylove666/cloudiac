// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package forms

import "cloudiac/portal/models"

type SearchCloudSyncPolicyForm struct {
	PageForm

	Q              string    `form:"q" json:"q"`
	Status         string    `form:"status" json:"status" binding:"omitempty,oneof=enable disable"`
	Provider       string    `form:"provider" json:"provider" binding:"omitempty,oneof=aws oci oracle alicloud azure gcp tencentcloud huawei"`
	CloudAccountId models.Id `form:"cloudAccountId" json:"cloudAccountId" binding:"max=32"`
	AccountId      string    `form:"accountId" json:"accountId" binding:"max=128"`
	LastSyncStatus string    `form:"lastSyncStatus" json:"lastSyncStatus" binding:"omitempty,oneof=pending running complete failed"`
}

type CloudSyncPolicyParam struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type CreateCloudSyncPolicyForm struct {
	BaseForm

	Name                string          `form:"name" json:"name" binding:"required,max=128"`
	Description         string          `form:"description" json:"description" binding:"max=255"`
	CloudAccountId      models.Id       `form:"cloudAccountId" json:"cloudAccountId" binding:"required,max=32"`
	Provider            string          `form:"provider" json:"provider" binding:"omitempty,oneof=aws oci oracle alicloud azure gcp tencentcloud huawei"`
	Regions             []string        `form:"regions" json:"regions"`
	AssetTypes          []string        `form:"assetTypes" json:"assetTypes"`
	Status              string          `form:"status" json:"status" binding:"omitempty,oneof=enable disable"`
	SyncInterval        int             `form:"syncInterval" json:"syncInterval"`
	MaxRetryAttempts    int             `form:"maxRetryAttempts" json:"maxRetryAttempts"`
	RetryBackoffSeconds int             `form:"retryBackoffSeconds" json:"retryBackoffSeconds"`
	NotifyOnFailure     bool            `form:"notifyOnFailure" json:"notifyOnFailure"`
	AutoPauseOnFailure  bool            `form:"autoPauseOnFailure" json:"autoPauseOnFailure"`
	NextSyncAt          models.Time     `form:"nextSyncAt" json:"nextSyncAt"`
	Params              models.ResAttrs `form:"params" json:"params"`
}

type UpdateCloudSyncPolicyForm struct {
	CreateCloudSyncPolicyForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type RunDueCloudSyncPolicyForm struct {
	BaseForm

	Force bool `form:"force" json:"force"`
}
