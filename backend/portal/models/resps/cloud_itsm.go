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

	ConnectorName       string          `json:"connectorName"`
	OperationName       string          `json:"operationName"`
	ProjectName         string          `json:"projectName"`
	EnvName             string          `json:"envName"`
	CreatorName         string          `json:"creatorName"`
	RobotProcessed      bool            `json:"robotProcessed"`
	RobotProcessor      string          `json:"robotProcessor"`
	RobotProcessingTags []string        `json:"robotProcessingTags"`
	RobotAutomationMode string          `json:"robotAutomationMode"`
	RobotTicketChannel  string          `json:"robotTicketChannel"`
	DryRunPayload       models.ResAttrs `json:"dryRunPayload,omitempty"`
}

type CloudItsmSubmitRetryQueueItemResp struct {
	CloudItsmTicketResp

	QueueStatus string          `json:"queueStatus"`
	RetryReason string          `json:"retryReason"`
	Attempt     int             `json:"attempt"`
	NextAttempt int             `json:"nextAttempt"`
	NextRetryAt string          `json:"nextRetryAt"`
	DeadLetter  bool            `json:"deadLetter"`
	RetryState  models.ResAttrs `json:"retryState,omitempty"`
}

type CloudItsmSubmitRetryQueueBreakdownResp struct {
	DimensionType string `json:"dimensionType"`
	DimensionId   string `json:"dimensionId"`
	DimensionName string `json:"dimensionName"`
	Provider      string `json:"provider,omitempty"`
	FailedTotal   int    `json:"failedTotal"`
	Due           int    `json:"due"`
	Future        int    `json:"future"`
	DeadLetter    int    `json:"deadLetter"`
	Skipped       int    `json:"skipped"`
	Retryable     int    `json:"retryable"`
	OldestDueAt   string `json:"oldestDueAt,omitempty"`
	NextRetryAt   string `json:"nextRetryAt,omitempty"`
}

type CloudItsmSubmitRetryQueueReportResp struct {
	ConnectorBreakdown            []CloudItsmSubmitRetryQueueBreakdownResp `json:"connectorBreakdown"`
	ReasonBreakdown               []CloudItsmSubmitRetryQueueBreakdownResp `json:"reasonBreakdown"`
	AgeBuckets                    []CloudItsmSubmitRetryQueueBreakdownResp `json:"ageBuckets"`
	ProjectBreakdown              []CloudItsmSubmitRetryQueueBreakdownResp `json:"projectBreakdown"`
	RequestTypeBreakdown          []CloudItsmSubmitRetryQueueBreakdownResp `json:"requestTypeBreakdown"`
	TeamBreakdown                 []CloudItsmSubmitRetryQueueBreakdownResp `json:"teamBreakdown"`
	ErrorCodeBreakdown            []CloudItsmSubmitRetryQueueBreakdownResp `json:"errorCodeBreakdown"`
	ExternalResponseCodeBreakdown []CloudItsmSubmitRetryQueueBreakdownResp `json:"externalResponseCodeBreakdown"`
	RecentDeadLetters             []CloudItsmSubmitRetryQueueItemResp      `json:"recentDeadLetters"`
}

type CloudItsmOverviewMetricResp struct {
	TicketTotal               int64   `json:"ticketTotal"`
	AutomatedTicketTotal      int64   `json:"automatedTicketTotal"`
	ManualTicketTotal         int64   `json:"manualTicketTotal"`
	RobotProcessedTicketTotal int64   `json:"robotProcessedTicketTotal"`
	TicketAutomationRate      float64 `json:"ticketAutomationRate"`
	RobotProcessingRate       float64 `json:"robotProcessingRate"`
	TicketAutomationTarget    float64 `json:"ticketAutomationTarget"`
	SelfServiceCatalogTotal   int64   `json:"selfServiceCatalogTotal"`
	SelfServiceAvailableTotal int64   `json:"selfServiceAvailableTotal"`
	SelfServiceCoverageRate   float64 `json:"selfServiceCoverageRate"`
	SelfServiceCoverageTarget float64 `json:"selfServiceCoverageTarget"`
	SelfServicePolicyTotal    int64   `json:"selfServicePolicyTotal"`
	SelfServicePolicyCovered  int64   `json:"selfServicePolicyCovered"`
	SelfServicePolicyRate     float64 `json:"selfServicePolicyRate"`
	SelfServiceSlaTicketTotal int64   `json:"selfServiceSlaTicketTotal"`
	SelfServiceSlaMetTotal    int64   `json:"selfServiceSlaMetTotal"`
	SelfServiceSlaBreached    int64   `json:"selfServiceSlaBreached"`
	SelfServiceSlaMetRate     float64 `json:"selfServiceSlaMetRate"`
	SelfServiceSlaTarget      float64 `json:"selfServiceSlaTarget"`
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
	Key                 string                           `json:"key"`
	Name                string                           `json:"name"`
	Category            string                           `json:"category"`
	Description         string                           `json:"description"`
	Source              string                           `json:"source"`
	OperationType       string                           `json:"operationType"`
	Action              string                           `json:"action"`
	AutomationMode      string                           `json:"automationMode"`
	Available           bool                             `json:"available"`
	Enabled             bool                             `json:"enabled"`
	RequiresApproval    bool                             `json:"requiresApproval"`
	RiskLevel           string                           `json:"riskLevel"`
	SupportedProviders  []string                         `json:"supportedProviders"`
	SupportedAssetTypes []string                         `json:"supportedAssetTypes"`
	DisabledReason      string                           `json:"disabledReason"`
	PolicyKey           string                           `json:"policyKey"`
	PolicyName          string                           `json:"policyName"`
	PolicyDescription   string                           `json:"policyDescription"`
	RequiredRoles       []string                         `json:"requiredRoles"`
	AllowedScopes       []string                         `json:"allowedScopes"`
	SlaMinutes          int                              `json:"slaMinutes"`
	SlaDescription      string                           `json:"slaDescription"`
	PolicyConfigured    bool                             `json:"policyConfigured"`
	PolicySource        string                           `json:"policySource"`
	PolicyVersion       int                              `json:"policyVersion"`
	PolicyUpdatedAt     string                           `json:"policyUpdatedAt"`
	PolicyUpdatedBy     string                           `json:"policyUpdatedBy"`
	PolicyLastDiff      []CloudItsmCatalogPolicyDiffResp `json:"policyLastDiff,omitempty"`
}

type CloudItsmCatalogPolicyDiffResp struct {
	Field  string      `json:"field"`
	Label  string      `json:"label"`
	Before interface{} `json:"before"`
	After  interface{} `json:"after"`
}

type CloudItsmCatalogPolicyHistoryResp struct {
	Version        int                              `json:"version"`
	Action         string                           `json:"action"`
	Key            string                           `json:"key"`
	PolicyName     string                           `json:"policyName"`
	UpdatedAt      string                           `json:"updatedAt"`
	UpdatedBy      string                           `json:"updatedBy"`
	Diff           []CloudItsmCatalogPolicyDiffResp `json:"diff"`
	BeforeSnapshot models.ResAttrs                  `json:"beforeSnapshot,omitempty"`
	AfterSnapshot  models.ResAttrs                  `json:"afterSnapshot,omitempty"`
}

type CloudItsmSlaTrendResp struct {
	Date        string  `json:"date"`
	TicketTotal int64   `json:"ticketTotal"`
	SlaMetTotal int64   `json:"slaMetTotal"`
	SlaBreached int64   `json:"slaBreached"`
	SlaMetRate  float64 `json:"slaMetRate"`
	Target      float64 `json:"target"`
}

type CloudItsmDimensionMetricResp struct {
	DimensionType             string  `json:"dimensionType"`
	DimensionId               string  `json:"dimensionId"`
	DimensionName             string  `json:"dimensionName"`
	TicketTotal               int64   `json:"ticketTotal"`
	SelfServiceTicketTotal    int64   `json:"selfServiceTicketTotal"`
	SelfServiceCoverageRate   float64 `json:"selfServiceCoverageRate"`
	SelfServiceCoverageTarget float64 `json:"selfServiceCoverageTarget"`
	AutomatedTicketTotal      int64   `json:"automatedTicketTotal"`
	RobotProcessedTicketTotal int64   `json:"robotProcessedTicketTotal"`
	TicketAutomationRate      float64 `json:"ticketAutomationRate"`
	TicketAutomationTarget    float64 `json:"ticketAutomationTarget"`
	SelfServiceSlaTicketTotal int64   `json:"selfServiceSlaTicketTotal"`
	SelfServiceSlaMetTotal    int64   `json:"selfServiceSlaMetTotal"`
	SelfServiceSlaBreached    int64   `json:"selfServiceSlaBreached"`
	SelfServiceSlaMetRate     float64 `json:"selfServiceSlaMetRate"`
	SelfServiceSlaTarget      float64 `json:"selfServiceSlaTarget"`
	WindowDays                int     `json:"windowDays"`
}

type CloudItsmOverviewResp struct {
	Metrics           CloudItsmOverviewMetricResp    `json:"metrics"`
	Catalog           []CloudItsmCatalogItemResp     `json:"catalog"`
	SlaTrend          []CloudItsmSlaTrendResp        `json:"slaTrend"`
	ProjectTrends     []CloudItsmDimensionMetricResp `json:"projectTrends"`
	RequestTypeTrends []CloudItsmDimensionMetricResp `json:"requestTypeTrends"`
	TeamTrends        []CloudItsmDimensionMetricResp `json:"teamTrends"`
}
