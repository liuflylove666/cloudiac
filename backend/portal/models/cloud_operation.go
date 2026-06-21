// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package models

const (
	CloudOperationTypeGovernance  = "governance"
	CloudOperationTypeAction      = "action"
	CloudOperationTypeSelfService = "self_service"

	CloudOperationActionGovernanceOwnership = "governance_ownership"
	CloudOperationActionRefreshMetadata     = "refresh_metadata"
	CloudOperationActionUpdateTags          = "update_tags"
	CloudOperationActionStartInstance       = "start_instance"
	CloudOperationActionStopInstance        = "stop_instance"
	CloudOperationActionRestartInstance     = "restart_instance"
	CloudOperationActionResizeInstance      = "resize_instance"
	CloudOperationActionResizeVolume        = "resize_volume"
	CloudOperationActionCreateSnapshot      = "create_snapshot"
	CloudOperationActionUpdateSecurityRules = "update_security_rules"
	CloudOperationActionDeleteResource      = "delete_resource"
	CloudOperationActionPermissionRequest   = "permission_request"
	CloudOperationActionGitOpsIacChange     = "gitops_iac_change"
	CloudOperationActionDriftRemediation    = "drift_remediation"

	CloudOperationStatusPending   = "pending"
	CloudOperationStatusApproving = "approving"
	CloudOperationStatusRunning   = "running"
	CloudOperationStatusComplete  = "complete"
	CloudOperationStatusFailed    = "failed"
	CloudOperationStatusAborted   = "aborted"
	CloudOperationStatusRejected  = "rejected"

	CloudOperationRiskLow      = "low"
	CloudOperationRiskMedium   = "medium"
	CloudOperationRiskHigh     = "high"
	CloudOperationRiskCritical = "critical"
)

type CloudOperation struct {
	SoftDeleteModel

	OrgId          Id       `json:"orgId" gorm:"index;size:32;not null"`
	ProjectId      Id       `json:"projectId" gorm:"index;size:32;not null;default:''"`
	EnvId          Id       `json:"envId" gorm:"index;size:32;not null;default:''"`
	AssetId        Id       `json:"assetId" gorm:"index;size:32;not null;default:''"`
	CloudAccountId Id       `json:"cloudAccountId" gorm:"index;size:32;not null;default:''"`
	CreatorId      Id       `json:"creatorId" gorm:"index;size:32;not null;default:''"`
	ApprovalId     Id       `json:"approvalId" gorm:"index;size:32;not null;default:''"`
	Name           string   `json:"name" gorm:"index;size:128;not null;default:''"`
	OperationType  string   `json:"operationType" gorm:"index;size:64;not null;default:'governance'"`
	Action         string   `json:"action" gorm:"index;size:64;not null;default:''"`
	Status         string   `json:"status" gorm:"index;size:32;not null;default:'pending'"`
	RiskLevel      string   `json:"riskLevel" gorm:"index;size:32;not null;default:'low'"`
	Provider       string   `json:"provider" gorm:"index;size:64;not null;default:''"`
	ResourceType   string   `json:"resourceType" gorm:"index;size:64;not null;default:''"`
	ResourceId     string   `json:"resourceId" gorm:"index;size:255;not null;default:''"`
	ResourceName   string   `json:"resourceName" gorm:"size:255;not null;default:''"`
	Message        string   `json:"message" gorm:"type:text"`
	Params         ResAttrs `json:"params,omitempty" gorm:"type:json"`
	Result         ResAttrs `json:"result,omitempty" gorm:"type:json"`
	StartedAt      Time     `json:"startedAt" gorm:"type:datetime;default:null"`
	EndedAt        Time     `json:"endedAt" gorm:"type:datetime;default:null"`
}

func (CloudOperation) TableName() string {
	return "iac_cloud_operation"
}

type CloudOperationStep struct {
	TimedModel

	OrgId       Id       `json:"orgId" gorm:"index;size:32;not null"`
	OperationId Id       `json:"operationId" gorm:"index;size:32;not null"`
	Index       int      `json:"index" gorm:"index;not null;default:0"`
	Name        string   `json:"name" gorm:"size:128;not null;default:''"`
	Status      string   `json:"status" gorm:"index;size:32;not null;default:'pending'"`
	Message     string   `json:"message" gorm:"type:text"`
	Error       string   `json:"error" gorm:"type:text"`
	Result      ResAttrs `json:"result,omitempty" gorm:"type:json"`
	StartedAt   Time     `json:"startedAt" gorm:"type:datetime;default:null"`
	EndedAt     Time     `json:"endedAt" gorm:"type:datetime;default:null"`
}

func (CloudOperationStep) TableName() string {
	return "iac_cloud_operation_step"
}

type CloudOperationAudit struct {
	TimedModel

	OrgId       Id       `json:"orgId" gorm:"index;size:32;not null"`
	OperationId Id       `json:"operationId" gorm:"index;size:32;not null"`
	AssetId     Id       `json:"assetId" gorm:"index;size:32;not null;default:''"`
	OperatorId  Id       `json:"operatorId" gorm:"index;size:32;not null;default:''"`
	Action      string   `json:"action" gorm:"index;size:64;not null;default:''"`
	Status      string   `json:"status" gorm:"index;size:32;not null;default:''"`
	Summary     string   `json:"summary" gorm:"type:text"`
	Params      ResAttrs `json:"params,omitempty" gorm:"type:json"`
	Result      ResAttrs `json:"result,omitempty" gorm:"type:json"`
	UserIp      string   `json:"userIp" gorm:"size:128;not null;default:''"`
	UserAgent   string   `json:"userAgent" gorm:"size:255;not null;default:''"`
}

func (CloudOperationAudit) TableName() string {
	return "iac_cloud_operation_audit"
}
