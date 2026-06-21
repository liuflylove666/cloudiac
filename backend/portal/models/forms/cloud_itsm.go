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
	RequestType string          `form:"requestType" json:"requestType" binding:"required,oneof=permission_request gitops_iac_change drift_remediation"`
	Title       string          `form:"title" json:"title" binding:"max=255"`
	Description string          `form:"description" json:"description"`
	Priority    string          `form:"priority" json:"priority" binding:"max=32"`
	ProjectId   models.Id       `form:"projectId" json:"projectId" binding:"max=32"`
	EnvId       models.Id       `form:"envId" json:"envId" binding:"max=32"`
	Params      models.ResAttrs `form:"params" json:"params"`
	DryRun      bool            `form:"dryRun" json:"dryRun"`
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
