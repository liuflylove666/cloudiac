// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package models

const (
	CloudEventSourceAccount      = "account"
	CloudEventSourceSync         = "sync"
	CloudEventSourceOperation    = "operation"
	CloudEventSourceRisk         = "risk"
	CloudEventSourceCost         = "cost"
	CloudEventSourceCMDB         = "cmdb"
	CloudEventSourceNotification = "notification"
	CloudEventSourceITSM         = "itsm"

	CloudEventLevelInfo    = "info"
	CloudEventLevelWarning = "warning"
	CloudEventLevelError   = "error"
)

type CloudEvent struct {
	SoftDeleteModel

	OrgId          Id       `json:"orgId" gorm:"index;size:32;not null"`
	ProjectId      Id       `json:"projectId" gorm:"index;size:32;not null;default:''"`
	EnvId          Id       `json:"envId" gorm:"index;size:32;not null;default:''"`
	AssetId        Id       `json:"assetId" gorm:"index;size:32;not null;default:''"`
	OperationId    Id       `json:"operationId" gorm:"index;size:32;not null;default:''"`
	RiskFindingId  Id       `json:"riskFindingId" gorm:"index;size:32;not null;default:''"`
	CloudAccountId Id       `json:"cloudAccountId" gorm:"index;size:32;not null;default:''"`
	ActorId        Id       `json:"actorId" gorm:"index;size:32;not null;default:''"`
	Source         string   `json:"source" gorm:"index;size:64;not null;default:''"`
	EventType      string   `json:"eventType" gorm:"index;size:128;not null;default:''"`
	Level          string   `json:"level" gorm:"index;size:32;not null;default:'info'"`
	Status         string   `json:"status" gorm:"index;size:32;not null;default:''"`
	Provider       string   `json:"provider" gorm:"index;size:64;not null;default:''"`
	AccountId      string   `json:"accountId" gorm:"index;size:128;not null;default:''"`
	Region         string   `json:"region" gorm:"index;size:128;not null;default:''"`
	ResourceType   string   `json:"resourceType" gorm:"index;size:64;not null;default:''"`
	ResourceId     string   `json:"resourceId" gorm:"index;size:255;not null;default:''"`
	ResourceName   string   `json:"resourceName" gorm:"size:255;not null;default:''"`
	Title          string   `json:"title" gorm:"size:255;not null;default:''"`
	Message        string   `json:"message" gorm:"type:text"`
	Payload        ResAttrs `json:"payload,omitempty" gorm:"type:json"`
	UserIp         string   `json:"userIp" gorm:"size:128;not null;default:''"`
	OccurredAt     Time     `json:"occurredAt" gorm:"index;type:datetime;default:null"`
}

func (CloudEvent) TableName() string {
	return "iac_cloud_event"
}
