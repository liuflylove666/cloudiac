// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package forms

import "cloudiac/portal/models"

type SearchCloudAccountForm struct {
	PageForm

	Q                string `form:"q" json:"q"`
	Provider         string `form:"provider" json:"provider"`
	Status           string `form:"status" json:"status" binding:"omitempty,oneof=enable disable"`
	ValidationStatus string `form:"validationStatus" json:"validationStatus" binding:"omitempty,oneof=pending valid invalid"`
}

type CreateCloudAccountForm struct {
	BaseForm

	Name        string   `form:"name" json:"name" binding:"required,gte=2,lte=128"`
	Description string   `form:"description" json:"description" binding:"max=255"`
	Provider    string   `form:"provider" json:"provider" binding:"required,oneof=aws oci oracle alicloud azure gcp tencentcloud huawei"`
	AccountId   string   `form:"accountId" json:"accountId" binding:"max=128"`
	TenantId    string   `form:"tenantId" json:"tenantId" binding:"max=128"`
	Regions     []string `form:"regions" json:"regions"`
	RunnerTags  []string `form:"runnerTags" json:"runnerTags"`
	Credentials []Params `form:"credentials" json:"credentials" binding:"required"`
	Status      string   `form:"status" json:"status" binding:"omitempty,oneof=enable disable"`
}

type UpdateCloudAccountForm struct {
	BaseForm

	Id          models.Id `uri:"id" json:"id" binding:"required" swaggerignore:"true"`
	Name        string    `form:"name" json:"name" binding:"omitempty,gte=2,lte=128"`
	Description string    `form:"description" json:"description" binding:"max=255"`
	Provider    string    `form:"provider" json:"provider" binding:"omitempty,oneof=aws oci oracle alicloud azure gcp tencentcloud huawei"`
	AccountId   string    `form:"accountId" json:"accountId" binding:"max=128"`
	TenantId    string    `form:"tenantId" json:"tenantId" binding:"max=128"`
	Regions     []string  `form:"regions" json:"regions"`
	RunnerTags  []string  `form:"runnerTags" json:"runnerTags"`
	Credentials []Params  `form:"credentials" json:"credentials"`
	Status      string    `form:"status" json:"status" binding:"omitempty,oneof=enable disable"`
}

type CloudAccountParam struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required" swaggerignore:"true"`
}

type UpdateCloudAccountRegionsForm struct {
	BaseForm

	Id      models.Id `uri:"id" json:"id" binding:"required" swaggerignore:"true"`
	Regions []string  `form:"regions" json:"regions"`
}
