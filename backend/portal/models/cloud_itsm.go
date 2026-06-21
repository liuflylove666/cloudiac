// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package models

import "cloudiac/portal/libs/db"

const (
	CloudItsmConfigStatusEnabled  = Enable
	CloudItsmConfigStatusDisabled = Disable

	CloudItsmProviderGeneric    = "generic"
	CloudItsmProviderJira       = "jira"
	CloudItsmProviderServiceNow = "servicenow"

	CloudItsmAuthNone   = "none"
	CloudItsmAuthBearer = "bearer"
	CloudItsmAuthBasic  = "basic"

	CloudItsmTicketStatusPending    = "pending"
	CloudItsmTicketStatusSubmitted  = "submitted"
	CloudItsmTicketStatusFailed     = "failed"
	CloudItsmTicketStatusInProgress = "in_progress"
	CloudItsmTicketStatusResolved   = "resolved"
	CloudItsmTicketStatusClosed     = "closed"
	CloudItsmTicketStatusCanceled   = "canceled"
)

type CloudItsmConfig struct {
	SoftDeleteModel

	OrgId          Id       `json:"orgId" gorm:"index;size:32;not null"`
	Name           string   `json:"name" gorm:"index;size:128;not null"`
	Description    string   `json:"description" gorm:"size:255;not null;default:''"`
	Provider       string   `json:"provider" gorm:"index;size:64;not null;default:'generic'"`
	BaseUrl        string   `json:"baseUrl" gorm:"size:512;not null;default:''"`
	AuthType       string   `json:"authType" gorm:"size:32;not null;default:'none'"`
	Token          string   `json:"-" gorm:"type:text"`
	Username       string   `json:"username" gorm:"size:128;not null;default:''"`
	Password       string   `json:"-" gorm:"type:text"`
	ProjectKey     string   `json:"projectKey" gorm:"size:128;not null;default:''"`
	TicketType     string   `json:"ticketType" gorm:"size:128;not null;default:''"`
	Status         string   `json:"status" gorm:"index;size:32;not null;default:'enable'"`
	TimeoutSeconds int      `json:"timeoutSeconds" gorm:"not null;default:10"`
	Metadata       ResAttrs `json:"metadata,omitempty" gorm:"type:json"`
}

func (CloudItsmConfig) TableName() string {
	return "iac_cloud_itsm_config"
}

func (cfg CloudItsmConfig) Migrate(sess *db.Session) error {
	return cfg.AddUniqueIndex(sess, "unique_cloud_itsm_config_org_name", "org_id", "name")
}

type CloudItsmTicket struct {
	SoftDeleteModel

	OrgId           Id       `json:"orgId" gorm:"index;size:32;not null"`
	ProjectId       Id       `json:"projectId" gorm:"index;size:32;not null;default:''"`
	EnvId           Id       `json:"envId" gorm:"index;size:32;not null;default:''"`
	OperationId     Id       `json:"operationId" gorm:"index;size:32;not null;default:''"`
	CloudEventId    Id       `json:"cloudEventId" gorm:"index;size:32;not null;default:''"`
	ConnectorId     Id       `json:"connectorId" gorm:"index;size:32;not null"`
	CreatorId       Id       `json:"creatorId" gorm:"index;size:32;not null;default:''"`
	ExternalId      string   `json:"externalId" gorm:"index;size:128;not null;default:''"`
	ExternalKey     string   `json:"externalKey" gorm:"index;size:128;not null;default:''"`
	ExternalUrl     string   `json:"externalUrl" gorm:"size:512;not null;default:''"`
	Title           string   `json:"title" gorm:"size:255;not null;default:''"`
	Description     string   `json:"description" gorm:"type:text"`
	Status          string   `json:"status" gorm:"index;size:32;not null;default:'pending'"`
	Priority        string   `json:"priority" gorm:"index;size:32;not null;default:''"`
	RiskLevel       string   `json:"riskLevel" gorm:"index;size:32;not null;default:''"`
	Provider        string   `json:"provider" gorm:"index;size:64;not null;default:''"`
	RequestPayload  ResAttrs `json:"requestPayload,omitempty" gorm:"type:json"`
	ResponsePayload ResAttrs `json:"responsePayload,omitempty" gorm:"type:json"`
	ErrorMessage    string   `json:"errorMessage" gorm:"type:text"`
	SubmittedAt     Time     `json:"submittedAt" gorm:"type:datetime;default:null"`
	LastSyncedAt    Time     `json:"lastSyncedAt" gorm:"type:datetime;default:null"`
	ClosedAt        Time     `json:"closedAt" gorm:"type:datetime;default:null"`
}

func (CloudItsmTicket) TableName() string {
	return "iac_cloud_itsm_ticket"
}

func (ticket CloudItsmTicket) Migrate(sess *db.Session) error {
	return ticket.AddUniqueIndex(sess, "unique_cloud_itsm_ticket_operation_connector", "org_id", "operation_id", "connector_id")
}
