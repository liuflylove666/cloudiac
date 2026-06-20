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

type CmdbAssetParam struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
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
	Owner          string      `json:"owner" form:"owner" binding:"max=128"`
	Application    string      `json:"application" form:"application" binding:"max=128"`
	BusinessLine   string      `json:"businessLine" form:"businessLine" binding:"max=128"`
	Lifecycle      string      `json:"lifecycle" form:"lifecycle" binding:"omitempty,oneof=planned active maintenance retired"`
	ComplianceRisk string      `json:"complianceRisk" form:"complianceRisk" binding:"omitempty,oneof=low medium high critical"`
}

type SearchCmdbSyncTaskForm struct {
	PageForm

	Provider      string    `form:"provider" json:"provider"`
	AccountSource string    `form:"accountSource" json:"accountSource"`
	AccountId     models.Id `form:"accountId" json:"accountId"`
	Status        string    `form:"status" json:"status"`
}

type CmdbSyncTaskParam struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type CreateCmdbSyncTaskForm struct {
	BaseForm

	AccountSource string    `json:"accountSource" form:"accountSource" binding:"required,oneof=variable_group resource_account cloud_account"`
	AccountId     models.Id `json:"accountId" form:"accountId" binding:"required,max=32"`
	Provider      string    `json:"provider" form:"provider" binding:"omitempty,oneof=aws oci oracle alicloud"`
	Regions       []string  `json:"regions" form:"regions"`
	AssetTypes    []string  `json:"assetTypes" form:"assetTypes"`
}
