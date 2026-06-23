// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package forms

import "cloudiac/portal/models"

type SearchCloudItsmConfigForm struct {
	PageForm

	Q        string `form:"q" json:"q"`
	Provider string `form:"provider" json:"provider" binding:"omitempty,oneof=generic jira servicenow"`
	Status   string `form:"status" json:"status" binding:"omitempty,oneof=enable disable"`
}

type CloudItsmConfigParam struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type CreateCloudItsmConfigForm struct {
	BaseForm

	Name           string          `form:"name" json:"name" binding:"required,gte=2,lte=128"`
	Description    string          `form:"description" json:"description" binding:"max=255"`
	Provider       string          `form:"provider" json:"provider" binding:"omitempty,oneof=generic jira servicenow"`
	BaseUrl        string          `form:"baseUrl" json:"baseUrl" binding:"max=512"`
	AuthType       string          `form:"authType" json:"authType" binding:"omitempty,oneof=none bearer basic"`
	Token          string          `form:"token" json:"token"`
	Username       string          `form:"username" json:"username" binding:"max=128"`
	Password       string          `form:"password" json:"password"`
	ProjectKey     string          `form:"projectKey" json:"projectKey" binding:"max=128"`
	TicketType     string          `form:"ticketType" json:"ticketType" binding:"max=128"`
	Status         string          `form:"status" json:"status" binding:"omitempty,oneof=enable disable"`
	TimeoutSeconds int             `form:"timeoutSeconds" json:"timeoutSeconds"`
	Metadata       models.ResAttrs `form:"metadata" json:"metadata"`
}

type UpdateCloudItsmConfigForm struct {
	CreateCloudItsmConfigForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type SearchCloudItsmTicketForm struct {
	PageForm

	Q           string    `form:"q" json:"q"`
	Status      string    `form:"status" json:"status" binding:"omitempty,oneof=pending submitted failed in_progress resolved closed canceled"`
	ConnectorId models.Id `form:"connectorId" json:"connectorId" binding:"max=32"`
	OperationId models.Id `form:"operationId" json:"operationId" binding:"max=32"`
	ProjectId   models.Id `form:"projectId" json:"projectId" binding:"max=32"`
	EnvId       models.Id `form:"envId" json:"envId" binding:"max=32"`
}

type CreateCloudOperationItsmTicketForm struct {
	BaseForm

	Id          models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	ConnectorId models.Id `form:"connectorId" json:"connectorId" binding:"required,max=32"`
	Title       string    `form:"title" json:"title" binding:"max=255"`
	Description string    `form:"description" json:"description"`
	Priority    string    `form:"priority" json:"priority" binding:"max=32"`
	DryRun      bool      `form:"dryRun" json:"dryRun"`
}

type CreateCloudItsmSelfServiceTicketForm struct {
	BaseForm

	ConnectorId models.Id       `form:"connectorId" json:"connectorId" binding:"max=32"`
	RequestType string          `form:"requestType" json:"requestType" binding:"required,oneof=permission_request gitops_iac_change drift_remediation risk_remediation"`
	Title       string          `form:"title" json:"title" binding:"max=255"`
	Description string          `form:"description" json:"description"`
	Priority    string          `form:"priority" json:"priority" binding:"max=32"`
	ProjectId   models.Id       `form:"projectId" json:"projectId" binding:"max=32"`
	EnvId       models.Id       `form:"envId" json:"envId" binding:"max=32"`
	Params      models.ResAttrs `form:"params" json:"params"`
	DryRun      bool            `form:"dryRun" json:"dryRun"`
}

type UpdateCloudItsmCatalogPolicyForm struct {
	BaseForm

	Key               string   `uri:"key" json:"key" binding:"required,max=128" swaggerignore:"true"`
	Enabled           bool     `form:"enabled" json:"enabled"`
	PolicyName        string   `form:"policyName" json:"policyName" binding:"max=128"`
	PolicyDescription string   `form:"policyDescription" json:"policyDescription" binding:"max=512"`
	RequiredRoles     []string `form:"requiredRoles" json:"requiredRoles"`
	AllowedScopes     []string `form:"allowedScopes" json:"allowedScopes"`
	SlaMinutes        int      `form:"slaMinutes" json:"slaMinutes"`
	Reset             bool     `form:"reset" json:"reset"`
}

type CloudItsmCatalogPolicyHistoryForm struct {
	BaseForm

	Key   string `uri:"key" json:"key" binding:"required,max=128" swaggerignore:"true"`
	Limit int    `form:"limit" json:"limit"`
}

type CloudItsmTicketStatusParam struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type UpdateCloudItsmTicketStatusForm struct {
	BaseForm

	Id          models.Id       `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	Status      string          `form:"status" json:"status" binding:"required,oneof=pending submitted failed in_progress resolved closed canceled"`
	ExternalId  string          `form:"externalId" json:"externalId" binding:"max=128"`
	ExternalKey string          `form:"externalKey" json:"externalKey" binding:"max=128"`
	ExternalUrl string          `form:"externalUrl" json:"externalUrl" binding:"max=512"`
	Comment     string          `form:"comment" json:"comment" binding:"max=512"`
	Payload     models.ResAttrs `form:"payload" json:"payload"`
}

type CloudItsmTicketStatusCallbackForm struct {
	BaseForm

	ConnectorId      models.Id       `uri:"connectorId" json:"connectorId" binding:"required,max=32" swaggerignore:"true"`
	TicketId         models.Id       `form:"ticketId" json:"ticketId" binding:"max=32"`
	Status           string          `form:"status" json:"status" binding:"required,oneof=pending submitted failed in_progress resolved closed canceled"`
	ExternalId       string          `form:"externalId" json:"externalId" binding:"max=128"`
	ExternalKey      string          `form:"externalKey" json:"externalKey" binding:"max=128"`
	ExternalUrl      string          `form:"externalUrl" json:"externalUrl" binding:"max=512"`
	Comment          string          `form:"comment" json:"comment" binding:"max=512"`
	Signature        string          `form:"signature" json:"signature" binding:"max=256"`
	SignatureVersion string          `form:"signatureVersion" json:"signatureVersion" binding:"max=32"`
	Payload          models.ResAttrs `form:"payload" json:"payload"`
	RawBody          []byte          `json:"-" swaggerignore:"true"`
}

type CloudItsmGitOpsGateCallbackForm struct {
	BaseForm

	ConnectorId      models.Id       `uri:"connectorId" json:"connectorId" binding:"required,max=32" swaggerignore:"true"`
	OperationId      models.Id       `form:"operationId" json:"operationId" binding:"max=32"`
	TicketId         models.Id       `form:"ticketId" json:"ticketId" binding:"max=32"`
	PullRequestUrl   string          `form:"pullRequestUrl" json:"pullRequestUrl" binding:"max=512"`
	Repository       string          `form:"repository" json:"repository" binding:"max=512"`
	Branch           string          `form:"branch" json:"branch" binding:"max=255"`
	TargetBranch     string          `form:"targetBranch" json:"targetBranch" binding:"max=255"`
	ChangePath       string          `form:"changePath" json:"changePath" binding:"max=512"`
	ReviewStatus     string          `form:"reviewStatus" json:"reviewStatus" binding:"max=64"`
	PipelineUrl      string          `form:"pipelineUrl" json:"pipelineUrl" binding:"max=512"`
	PipelineStatus   string          `form:"pipelineStatus" json:"pipelineStatus" binding:"max=64"`
	CommitSha        string          `form:"commitSha" json:"commitSha" binding:"max=128"`
	ExternalRunId    string          `form:"externalRunId" json:"externalRunId" binding:"max=128"`
	Comment          string          `form:"comment" json:"comment" binding:"max=512"`
	Signature        string          `form:"signature" json:"signature" binding:"max=256"`
	SignatureVersion string          `form:"signatureVersion" json:"signatureVersion" binding:"max=32"`
	Payload          models.ResAttrs `form:"payload" json:"payload"`
	RawBody          []byte          `json:"-" swaggerignore:"true"`
}

type SyncDueCloudItsmTicketStatusForm struct {
	BaseForm

	ConnectorId models.Id `form:"connectorId" json:"connectorId" binding:"max=32"`
	Force       bool      `form:"force" json:"force"`
	Limit       int       `form:"limit" json:"limit"`
}

type RetryDueCloudItsmTicketSubmitForm struct {
	BaseForm

	ConnectorId models.Id `form:"connectorId" json:"connectorId" binding:"max=32"`
	Force       bool      `form:"force" json:"force"`
	Limit       int       `form:"limit" json:"limit"`
}

type SearchCloudItsmSubmitRetryQueueForm struct {
	PageForm

	Q           string    `form:"q" json:"q"`
	ConnectorId models.Id `form:"connectorId" json:"connectorId" binding:"max=32"`
	QueueStatus string    `form:"queueStatus" json:"queueStatus" binding:"omitempty,oneof=due future dead_letter skipped"`
}

type ReplayCloudItsmTicketSubmitForm struct {
	BaseForm

	Id    models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	Force bool      `form:"force" json:"force"`
}

type BatchCloudItsmDeadLetterActionForm struct {
	BaseForm

	Ids    []models.Id `form:"ids" json:"ids" binding:"required,min=1,max=50,dive,max=32"`
	Action string      `form:"action" json:"action" binding:"required,oneof=replay close"`
	Force  bool        `form:"force" json:"force"`
	Reason string      `form:"reason" json:"reason" binding:"max=512"`
}

type CloudItsmDeadLetterApprovalEvidenceItem struct {
	Type  string `form:"type" json:"type" binding:"max=32"`
	Label string `form:"label" json:"label" binding:"max=128"`
	Url   string `form:"url" json:"url" binding:"max=512"`
	Note  string `form:"note" json:"note" binding:"max=512"`
}

type CreateCloudItsmDeadLetterApprovalForm struct {
	BaseForm

	Ids           []models.Id                               `form:"ids" json:"ids" binding:"required,min=1,max=50,dive,max=32"`
	Action        string                                    `form:"action" json:"action" binding:"required,oneof=replay close"`
	Force         bool                                      `form:"force" json:"force"`
	Reason        string                                    `form:"reason" json:"reason" binding:"required,max=512"`
	EvidenceUrl   string                                    `form:"evidenceUrl" json:"evidenceUrl" binding:"omitempty,max=512"`
	EvidenceItems []CloudItsmDeadLetterApprovalEvidenceItem `form:"evidenceItems" json:"evidenceItems" binding:"omitempty,dive"`
	Evidence      models.ResAttrs                           `form:"evidence" json:"evidence"`
}
