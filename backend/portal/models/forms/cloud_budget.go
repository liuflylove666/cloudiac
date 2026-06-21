// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package forms

import "cloudiac/portal/models"

type SearchCloudBudgetForm struct {
	PageForm

	Q                string    `form:"q" json:"q"`
	Scope            string    `form:"scope" json:"scope" binding:"omitempty,oneof=org project env provider account application business_line cost_center owner"`
	Status           string    `form:"status" json:"status" binding:"omitempty,oneof=enable disable"`
	Period           string    `form:"period" json:"period"`
	Currency         string    `form:"currency" json:"currency"`
	Provider         string    `form:"provider" json:"provider"`
	AccountId        string    `form:"accountId" json:"accountId"`
	Region           string    `form:"region" json:"region"`
	Application      string    `form:"application" json:"application"`
	BusinessLine     string    `form:"businessLine" json:"businessLine"`
	CostCenter       string    `form:"costCenter" json:"costCenter"`
	Owner            string    `form:"owner" json:"owner"`
	ThresholdReached *bool     `form:"thresholdReached" json:"thresholdReached"`
	ProjectId        models.Id `form:"projectId" json:"projectId" binding:"max=32"`
	EnvId            models.Id `form:"envId" json:"envId" binding:"max=32"`
}

type CloudBudgetSummaryForm struct {
	BaseForm

	Period   string `form:"period" json:"period"`
	Currency string `form:"currency" json:"currency"`
}

type EvaluateCloudBudgetForm struct {
	BaseForm

	Period   string `form:"period" json:"period"`
	Currency string `form:"currency" json:"currency"`
	Force    bool   `form:"force" json:"force"`
}

type CloudBudgetParam struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type CreateCloudBudgetForm struct {
	BaseForm

	Name               string    `form:"name" json:"name" binding:"required,gte=2,lte=128"`
	Description        string    `form:"description" json:"description" binding:"max=255"`
	Scope              string    `form:"scope" json:"scope" binding:"omitempty,oneof=org project env provider account application business_line cost_center owner"`
	ProjectId          models.Id `form:"projectId" json:"projectId" binding:"max=32"`
	EnvId              models.Id `form:"envId" json:"envId" binding:"max=32"`
	CloudAccountId     models.Id `form:"cloudAccountId" json:"cloudAccountId" binding:"max=32"`
	Provider           string    `form:"provider" json:"provider" binding:"max=64"`
	AccountId          string    `form:"accountId" json:"accountId" binding:"max=128"`
	Region             string    `form:"region" json:"region" binding:"max=128"`
	Application        string    `form:"application" json:"application" binding:"max=128"`
	BusinessLine       string    `form:"businessLine" json:"businessLine" binding:"max=128"`
	CostCenter         string    `form:"costCenter" json:"costCenter" binding:"max=128"`
	Owner              string    `form:"owner" json:"owner" binding:"max=128"`
	Period             string    `form:"period" json:"period" binding:"max=32"`
	Currency           string    `form:"currency" json:"currency" binding:"max=16"`
	LimitAmount        float64   `form:"limitAmount" json:"limitAmount" binding:"required"`
	ThresholdPercent   float64   `form:"thresholdPercent" json:"thresholdPercent"`
	EvaluationInterval int       `form:"evaluationInterval" json:"evaluationInterval"`
	Status             string    `form:"status" json:"status" binding:"omitempty,oneof=enable disable"`
}

type UpdateCloudBudgetForm struct {
	CreateCloudBudgetForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}
