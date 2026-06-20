// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package forms

type SearchSystemConfigForm struct {
	PageForm

	Q string `form:"q" json:"q" binding:""`
}

type UpdateSystemConfigForm struct {
	BaseForm
	SystemCfg []SystemCfg `json:"systemCfg" form:"systemCfg" binding:"required,dive,required" `
}

type SystemCfg struct {
	Name        string `form:"name" json:"name" binding:"required,gte=1,lte=255"`
	Value       string `form:"value" json:"value" binding:"required,gte=1,lte=32"`
	Description string `form:"description" json:"description" binding:"max=255"`
}

type RegistryAddrForm struct {
	BaseForm
	RegistryAddr          string `form:"registryAddr" json:"registryAddr" binding:""`
	ImageRegistryType     string `form:"imageRegistryType" json:"imageRegistryType" binding:"omitempty,oneof=generic harbor ecr"`
	ImageRegistryAddr     string `form:"imageRegistryAddr" json:"imageRegistryAddr" binding:""`
	EcrAccountId          string `form:"ecrAccountId" json:"ecrAccountId" binding:""`
	EcrRegion             string `form:"ecrRegion" json:"ecrRegion" binding:""`
	EcrEndpoint           string `form:"ecrEndpoint" json:"ecrEndpoint" binding:""`
	ImageRepositoryPrefix string `form:"imageRepositoryPrefix" json:"imageRepositoryPrefix" binding:""`
}
