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

	CloudAccountRegionSourceConfigured = "configured"
	CloudAccountRegionSourceInferred   = "inferred"
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

type CloudAccountRegion struct {
	TimedModel

	OrgId          Id       `json:"orgId" gorm:"index;size:32;not null;comment:组织ID"`
	CloudAccountId Id       `json:"cloudAccountId" gorm:"index;size:32;not null;default:'';comment:云账号ID"`
	Provider       string   `json:"provider" gorm:"index;size:64;not null;default:'';comment:云厂商"`
	AccountId      string   `json:"accountId" gorm:"index;size:128;not null;default:'';comment:云厂商账号ID"`
	Region         string   `json:"region" gorm:"index;size:128;not null;default:'';comment:区域"`
	Enabled        bool     `json:"enabled" gorm:"not null;default:true;comment:是否启用"`
	IsDefault      bool     `json:"default" gorm:"column:is_default;not null;default:false;comment:是否默认区域"`
	SyncEnabled    bool     `json:"syncEnabled" gorm:"not null;default:true;comment:是否参与同步"`
	Source         string   `json:"source" gorm:"index;size:64;not null;default:'';comment:区域来源"`
	Status         string   `json:"status" gorm:"index;size:32;not null;default:'';comment:区域状态"`
	Message        string   `json:"message" gorm:"type:text;comment:区域状态说明"`
	ResourceTypes  StrSlice `json:"resourceTypes" gorm:"type:json;comment:区域资源类型范围"`
	LastSyncAt     Time     `json:"lastSyncAt" gorm:"type:datetime;default:null;comment:最近同步时间"`
	Metadata       ResAttrs `json:"metadata,omitempty" gorm:"type:json;comment:扩展信息"`
}

func (CloudAccountRegion) TableName() string {
	return "iac_cloud_account_region"
}

func (r CloudAccountRegion) Migrate(sess *db.Session) error {
	return r.AddUniqueIndex(sess, "unique_cloud_account_region", "org_id", "cloud_account_id", "region")
}

type CloudAccountPermission struct {
	TimedModel

	OrgId          Id       `json:"orgId" gorm:"index;size:32;not null;comment:组织ID"`
	CloudAccountId Id       `json:"cloudAccountId" gorm:"index;size:32;not null;default:'';comment:云账号ID"`
	Provider       string   `json:"provider" gorm:"index;size:64;not null;default:'';comment:云厂商"`
	AccountId      string   `json:"accountId" gorm:"index;size:128;not null;default:'';comment:云厂商账号ID"`
	PermissionKey  string   `json:"key" gorm:"column:permission_key;index;size:128;not null;default:'';comment:权限检查项"`
	Name           string   `json:"name" gorm:"size:128;not null;default:'';comment:检查项名称"`
	Resource       string   `json:"resource" gorm:"index;size:128;not null;default:'';comment:资源类型"`
	Action         string   `json:"action" gorm:"index;size:128;not null;default:'';comment:操作"`
	Status         string   `json:"status" gorm:"index;size:32;not null;default:'';comment:检查结果"`
	Message        string   `json:"message" gorm:"type:text;comment:检查说明"`
	Source         string   `json:"source" gorm:"index;size:64;not null;default:'';comment:检查来源"`
	CheckedAt      Time     `json:"checkedAt" gorm:"type:datetime;default:null;comment:检查时间"`
	Evidence       ResAttrs `json:"evidence,omitempty" gorm:"type:json;comment:检查证据"`
}

func (CloudAccountPermission) TableName() string {
	return "iac_cloud_account_permission"
}

func (p CloudAccountPermission) Migrate(sess *db.Session) error {
	return p.AddUniqueIndex(sess, "unique_cloud_account_permission_item", "org_id", "cloud_account_id", "permission_key")
}
