// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package models

import "cloudiac/portal/libs/db"

const (
	CloudSyncPolicyStatusEnabled  = Enable
	CloudSyncPolicyStatusDisabled = Disable
)

type CloudSyncPolicy struct {
	SoftDeleteModel

	OrgId          Id       `json:"orgId" gorm:"index;size:32;not null"`
	CreatorId      Id       `json:"creatorId" gorm:"index;size:32;not null;default:''"`
	Name           string   `json:"name" gorm:"index;size:128;not null;default:''"`
	Description    string   `json:"description" gorm:"type:text"`
	CloudAccountId Id       `json:"cloudAccountId" gorm:"index;size:32;not null;default:''"`
	Provider       string   `json:"provider" gorm:"index;size:64;not null;default:''"`
	AccountId      string   `json:"accountId" gorm:"index;size:128;not null;default:''"`
	Regions        StrSlice `json:"regions" gorm:"type:json"`
	AssetTypes     StrSlice `json:"assetTypes" gorm:"type:json"`
	Status         string   `json:"status" gorm:"index;size:32;not null;default:'enable'"`

	SyncInterval        int  `json:"syncInterval" gorm:"not null;default:86400"`
	MaxRetryAttempts    int  `json:"maxRetryAttempts" gorm:"not null;default:3"`
	RetryBackoffSeconds int  `json:"retryBackoffSeconds" gorm:"not null;default:300"`
	NotifyOnFailure     bool `json:"notifyOnFailure" gorm:"not null;default:false"`
	AutoPauseOnFailure  bool `json:"autoPauseOnFailure" gorm:"not null;default:false"`

	LastSyncTaskId    Id     `json:"lastSyncTaskId" gorm:"index;size:32;not null;default:''"`
	LastSyncStatus    string `json:"lastSyncStatus" gorm:"index;size:32;not null;default:''"`
	LastError         string `json:"lastError" gorm:"type:text"`
	FailureCount      int    `json:"failureCount" gorm:"not null;default:0"`
	LastFailureAt     Time   `json:"lastFailureAt" gorm:"type:datetime;default:null"`
	LastFailureReason string `json:"lastFailureReason" gorm:"type:text"`
	LastSyncedAt      Time   `json:"lastSyncedAt" gorm:"type:datetime;default:null"`
	NextSyncAt        Time   `json:"nextSyncAt" gorm:"index;type:datetime;default:null"`

	Params ResAttrs `json:"params,omitempty" gorm:"type:json"`
}

func (CloudSyncPolicy) TableName() string {
	return "iac_cloud_sync_policy"
}

func (p CloudSyncPolicy) Migrate(sess *db.Session) error {
	return p.AddUniqueIndex(sess, "unique_cloud_sync_policy_org_name", "org_id", "name")
}
