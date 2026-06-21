// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CloudOperationResp struct {
	models.CloudOperation

	CreatorName string `json:"creatorName"`
	ProjectName string `json:"projectName"`
	EnvName     string `json:"envName"`
	AssetName   string `json:"assetName"`
}

type CloudOperationStepResp struct {
	models.CloudOperationStep
}

type CloudOperationAuditResp struct {
	models.CloudOperationAudit

	OperatorName string `json:"operatorName"`
}

type CloudOperationDetailResp struct {
	CloudOperationResp

	Steps  []CloudOperationStepResp  `json:"steps"`
	Audits []CloudOperationAuditResp `json:"audits"`
}

type CloudAssetActionResp struct {
	Key                 string   `json:"key"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	OperationType       string   `json:"operationType"`
	RiskLevel           string   `json:"riskLevel"`
	AdapterKey          string   `json:"adapterKey"`
	AdapterMode         string   `json:"adapterMode"`
	AdapterStatus       string   `json:"adapterStatus"`
	AdapterMessage      string   `json:"adapterMessage"`
	ProviderAdapter     string   `json:"providerAdapter"`
	WriteEnabled        bool     `json:"writeEnabled"`
	Enabled             bool     `json:"enabled"`
	Destructive         bool     `json:"destructive"`
	RequiresApproval    bool     `json:"requiresApproval"`
	DisabledReason      string   `json:"disabledReason"`
	SupportedProviders  []string `json:"supportedProviders"`
	SupportedAssetTypes []string `json:"supportedAssetTypes"`
}

type CloudAssetActionCheckResp struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type CloudAssetActionDryRunResp struct {
	AssetId          models.Id                   `json:"assetId"`
	Action           string                      `json:"action"`
	Name             string                      `json:"name"`
	Executable       bool                        `json:"executable"`
	RiskLevel        string                      `json:"riskLevel"`
	AdapterKey       string                      `json:"adapterKey"`
	AdapterMode      string                      `json:"adapterMode"`
	AdapterStatus    string                      `json:"adapterStatus"`
	AdapterMessage   string                      `json:"adapterMessage"`
	ProviderAdapter  string                      `json:"providerAdapter"`
	WriteEnabled     bool                        `json:"writeEnabled"`
	Destructive      bool                        `json:"destructive"`
	RequiresApproval bool                        `json:"requiresApproval"`
	DisabledReason   string                      `json:"disabledReason"`
	Checks           []CloudAssetActionCheckResp `json:"checks"`
}
