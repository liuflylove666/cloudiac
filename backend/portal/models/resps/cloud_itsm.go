// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CloudItsmConfigResp struct {
	models.CloudItsmConfig

	TokenConfigured    bool `json:"tokenConfigured"`
	PasswordConfigured bool `json:"passwordConfigured"`
}

type CloudItsmTicketResp struct {
	models.CloudItsmTicket

	ConnectorName string          `json:"connectorName"`
	OperationName string          `json:"operationName"`
	ProjectName   string          `json:"projectName"`
	EnvName       string          `json:"envName"`
	CreatorName   string          `json:"creatorName"`
	DryRunPayload models.ResAttrs `json:"dryRunPayload,omitempty"`
}

type CloudItsmOverviewMetricResp struct {
	TicketTotal               int64   `json:"ticketTotal"`
	AutomatedTicketTotal      int64   `json:"automatedTicketTotal"`
	ManualTicketTotal         int64   `json:"manualTicketTotal"`
	TicketAutomationRate      float64 `json:"ticketAutomationRate"`
	TicketAutomationTarget    float64 `json:"ticketAutomationTarget"`
	SelfServiceCatalogTotal   int64   `json:"selfServiceCatalogTotal"`
	SelfServiceAvailableTotal int64   `json:"selfServiceAvailableTotal"`
	SelfServiceCoverageRate   float64 `json:"selfServiceCoverageRate"`
	SelfServiceCoverageTarget float64 `json:"selfServiceCoverageTarget"`
	AssetTotal                int64   `json:"assetTotal"`
	IacManagedAssets          int64   `json:"iacManagedAssets"`
	IacCoverageRate           float64 `json:"iacCoverageRate"`
	IacCoverageTarget         float64 `json:"iacCoverageTarget"`
	CloudOnlyAssets           int64   `json:"cloudOnlyAssets"`
	EnvTotal                  int64   `json:"envTotal"`
	DriftEnabledEnvs          int64   `json:"driftEnabledEnvs"`
	AutoRepairDriftEnvs       int64   `json:"autoRepairDriftEnvs"`
	GitOpsGuardStatus         string  `json:"gitOpsGuardStatus"`
	GitOpsGuardMessage        string  `json:"gitOpsGuardMessage"`
	DriftGuardStatus          string  `json:"driftGuardStatus"`
	DriftGuardMessage         string  `json:"driftGuardMessage"`
}

type CloudItsmCatalogItemResp struct {
	Key                 string   `json:"key"`
	Name                string   `json:"name"`
	Category            string   `json:"category"`
	Description         string   `json:"description"`
	Source              string   `json:"source"`
	OperationType       string   `json:"operationType"`
	Action              string   `json:"action"`
	AutomationMode      string   `json:"automationMode"`
	Available           bool     `json:"available"`
	Enabled             bool     `json:"enabled"`
	RequiresApproval    bool     `json:"requiresApproval"`
	RiskLevel           string   `json:"riskLevel"`
	SupportedProviders  []string `json:"supportedProviders"`
	SupportedAssetTypes []string `json:"supportedAssetTypes"`
	DisabledReason      string   `json:"disabledReason"`
}

type CloudItsmOverviewResp struct {
	Metrics CloudItsmOverviewMetricResp `json:"metrics"`
	Catalog []CloudItsmCatalogItemResp  `json:"catalog"`
}
