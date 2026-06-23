// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package forms

import "cloudiac/portal/models"

type SearchCloudRiskForm struct {
	PageForm

	Q            string    `form:"q" json:"q"`
	Provider     string    `form:"provider" json:"provider"`
	AccountId    string    `form:"accountId" json:"accountId"`
	Region       string    `form:"region" json:"region"`
	ResourceType string    `form:"resourceType" json:"resourceType"`
	Source       string    `form:"source" json:"source" binding:"omitempty,oneof=cloud_config cmdb drift policy"`
	RuleKey      string    `form:"ruleKey" json:"ruleKey"`
	RiskLevel    string    `form:"riskLevel" json:"riskLevel" binding:"omitempty,oneof=low medium high critical"`
	Status       string    `form:"status" json:"status" binding:"omitempty,oneof=open in_progress suppressed resolved"`
	AssetId      models.Id `form:"assetId" json:"assetId" binding:"max=32"`
	ProjectId    models.Id `form:"projectId" json:"projectId" binding:"max=32"`
	EnvId        models.Id `form:"envId" json:"envId" binding:"max=32"`
}

type CloudRiskParam struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type UpdateCloudRiskStatusForm struct {
	BaseForm

	Id      models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	Status  string    `form:"status" json:"status" binding:"required,oneof=open in_progress resolved"`
	Comment string    `form:"comment" json:"comment" binding:"max=255"`
}

type SuppressCloudRiskForm struct {
	BaseForm

	Id              models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	SuppressedUntil string    `form:"suppressedUntil" json:"suppressedUntil" binding:"required"`
	Reason          string    `form:"reason" json:"reason" binding:"required,max=255"`
}

type CreateCloudRiskRemediationTicketForm struct {
	BaseForm

	Id          models.Id       `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	ConnectorId models.Id       `form:"connectorId" json:"connectorId" binding:"max=32"`
	Title       string          `form:"title" json:"title" binding:"max=255"`
	Description string          `form:"description" json:"description"`
	Priority    string          `form:"priority" json:"priority" binding:"max=32"`
	Params      models.ResAttrs `form:"params" json:"params"`
	DryRun      bool            `form:"dryRun" json:"dryRun"`
}

type AdoptCloudRiskRecommendationForm struct {
	BaseForm

	Id         models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	Action     string    `form:"action" json:"action" binding:"required,max=64"`
	TargetType string    `form:"targetType" json:"targetType" binding:"max=64"`
	TargetId   string    `form:"targetId" json:"targetId" binding:"max=128"`
	Comment    string    `form:"comment" json:"comment" binding:"max=255"`
}
