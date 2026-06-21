// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package forms

import "cloudiac/portal/models"

type SearchCloudEventForm struct {
	PageForm

	Q             string    `form:"q" json:"q"`
	Source        string    `form:"source" json:"source" binding:"omitempty,oneof=account sync operation risk cost cmdb notification itsm"`
	EventType     string    `form:"eventType" json:"eventType"`
	Level         string    `form:"level" json:"level" binding:"omitempty,oneof=info warning error"`
	Status        string    `form:"status" json:"status"`
	Provider      string    `form:"provider" json:"provider"`
	AccountId     string    `form:"accountId" json:"accountId"`
	Region        string    `form:"region" json:"region"`
	AssetId       models.Id `form:"assetId" json:"assetId" binding:"max=32"`
	OperationId   models.Id `form:"operationId" json:"operationId" binding:"max=32"`
	RiskFindingId models.Id `form:"riskFindingId" json:"riskFindingId" binding:"max=32"`
	ProjectId     models.Id `form:"projectId" json:"projectId" binding:"max=32"`
	EnvId         models.Id `form:"envId" json:"envId" binding:"max=32"`
}
