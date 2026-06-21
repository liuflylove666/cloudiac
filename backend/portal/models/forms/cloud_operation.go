// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package forms

import "cloudiac/portal/models"

type SearchCloudOperationForm struct {
	PageForm

	Q             string    `form:"q" json:"q"`
	Status        string    `form:"status" json:"status" binding:"omitempty,oneof=pending approving running complete failed aborted rejected"`
	Action        string    `form:"action" json:"action"`
	OperationType string    `form:"operationType" json:"operationType"`
	RiskLevel     string    `form:"riskLevel" json:"riskLevel" binding:"omitempty,oneof=low medium high critical"`
	AssetId       models.Id `form:"assetId" json:"assetId" binding:"max=32"`
	ProjectId     models.Id `form:"projectId" json:"projectId" binding:"max=32"`
	EnvId         models.Id `form:"envId" json:"envId" binding:"max=32"`
}

type CloudOperationParam struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type CloudOperationApprovalForm struct {
	BaseForm

	Id      models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	Action  string    `form:"action" json:"action" binding:"required,oneof=approved rejected"`
	Comment string    `form:"comment" json:"comment" binding:"max=255"`
}

type CloudAssetActionParam struct {
	BaseForm

	Id     models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	Action string    `uri:"action" json:"action" binding:"omitempty,max=64" swaggerignore:"true"`
}

type DryRunCloudAssetActionForm struct {
	BaseForm

	Id     models.Id       `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	Action string          `uri:"action" json:"action" binding:"required,max=64" swaggerignore:"true"`
	Params models.ResAttrs `json:"params" form:"params"`
	Tags   models.ResAttrs `json:"tags" form:"tags"`
}

type CreateCloudAssetActionForm struct {
	BaseForm

	Id                   models.Id       `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	Action               string          `uri:"action" json:"action" binding:"required,max=64" swaggerignore:"true"`
	Reason               string          `json:"reason" form:"reason" binding:"max=255"`
	Params               models.ResAttrs `json:"params" form:"params"`
	Tags                 models.ResAttrs `json:"tags" form:"tags"`
	ConfirmAction        string          `json:"confirmAction" form:"confirmAction" binding:"max=64"`
	ConfirmResourceId    string          `json:"confirmResourceId" form:"confirmResourceId" binding:"max=255"`
	RetryFromOperationId models.Id       `json:"retryFromOperationId" form:"retryFromOperationId" binding:"max=32" swaggerignore:"true"`
}
