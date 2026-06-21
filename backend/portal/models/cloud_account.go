// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package models

import "cloudiac/portal/libs/db"

const (
	CloudAccountStatusEnabled  = Enable
	CloudAccountStatusDisabled = Disable

	CloudAccountValidationPending = "pending"
	CloudAccountValidationValid   = "valid"
	CloudAccountValidationInvalid = "invalid"

	CloudAccountHealthHealthy   = "healthy"
	CloudAccountHealthWarning   = "warning"
	CloudAccountHealthUnhealthy = "unhealthy"
)

type CloudAccount struct {
	TimedModel

	OrgId       Id     `json:"orgId" gorm:"index;size:32;not null;comment:组织ID"`
	Name        string `json:"name" gorm:"index;size:128;not null;comment:云账号名称"`
	Description string `json:"description" gorm:"size:255;comment:云账号描述"`

	Provider  string `json:"provider" gorm:"index;size:64;not null;default:'';comment:云厂商"`
	AccountId string `json:"accountId" gorm:"index;size:128;not null;default:'';comment:云厂商账号ID"`
	TenantId  string `json:"tenantId" gorm:"index;size:128;not null;default:'';comment:租户或订阅ID"`

	Regions     StrSlice `json:"regions" gorm:"type:json;comment:启用区域"`
	RunnerTags  StrSlice `json:"runnerTags" gorm:"type:json;comment:执行Runner标签"`
	Credentials JSON     `json:"credentials" gorm:"type:json;comment:云账号凭证"`
	Metadata    ResAttrs `json:"metadata,omitempty" gorm:"type:json;comment:扩展信息"`

	Status              string   `json:"status" gorm:"index;size:32;not null;default:'enable';comment:状态"`
	ValidationStatus    string   `json:"validationStatus" gorm:"index;size:32;not null;default:'pending';comment:验证状态"`
	ValidationMessage   string   `json:"validationMessage" gorm:"type:text;comment:验证信息"`
	HealthStatus        string   `json:"healthStatus" gorm:"index;size:32;not null;default:'warning';comment:健康状态"`
	HealthMessage       string   `json:"healthMessage" gorm:"type:text;comment:健康信息"`
	LastValidatedAt     Time     `json:"lastValidatedAt" gorm:"type:datetime;default:null;comment:最近验证时间"`
	LastHealthCheckedAt Time     `json:"lastHealthCheckedAt" gorm:"type:datetime;default:null;comment:最近健康检查时间"`
	LastSyncAt          Time     `json:"lastSyncAt" gorm:"type:datetime;default:null;comment:最近同步时间"`
	SupportedTypes      StrSlice `json:"supportedTypes" gorm:"type:json;comment:支持采集的资产类型"`
}

func (CloudAccount) TableName() string {
	return "iac_cloud_account"
}

func (a CloudAccount) Migrate(sess *db.Session) error {
	return a.AddUniqueIndex(sess, "unique_cloud_account_org_name", "org_id", "name")
}
