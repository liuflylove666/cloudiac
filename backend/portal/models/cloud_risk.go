// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package models

import "cloudiac/portal/libs/db"

const (
	CloudRiskStatusOpen       = "open"
	CloudRiskStatusInProgress = "in_progress"
	CloudRiskStatusSuppressed = "suppressed"
	CloudRiskStatusResolved   = "resolved"

	CloudRiskSourceCloudConfig = "cloud_config"
	CloudRiskSourceCMDB        = "cmdb"
	CloudRiskSourceDrift       = "drift"
	CloudRiskSourcePolicy      = "policy"
)

type CloudRiskFinding struct {
	SoftDeleteModel

	OrgId          Id       `json:"orgId" gorm:"index;size:32;not null"`
	ProjectId      Id       `json:"projectId" gorm:"index;size:32;not null;default:''"`
	EnvId          Id       `json:"envId" gorm:"index;size:32;not null;default:''"`
	AssetId        Id       `json:"assetId" gorm:"index;size:32;not null;default:''"`
	CloudAccountId Id       `json:"cloudAccountId" gorm:"index;size:32;not null;default:''"`
	Provider       string   `json:"provider" gorm:"index;size:64;not null;default:''"`
	AccountId      string   `json:"accountId" gorm:"index;size:128;not null;default:''"`
	Region         string   `json:"region" gorm:"index;size:128;not null;default:''"`
	ResourceType   string   `json:"resourceType" gorm:"index;size:64;not null;default:''"`
	ResourceId     string   `json:"resourceId" gorm:"index;size:255;not null;default:''"`
	ResourceName   string   `json:"resourceName" gorm:"size:255;not null;default:''"`
	Source         string   `json:"source" gorm:"index;size:64;not null;default:''"`
	RuleKey        string   `json:"ruleKey" gorm:"index;size:128;not null;default:''"`
	RuleName       string   `json:"ruleName" gorm:"size:255;not null;default:''"`
	RiskLevel      string   `json:"riskLevel" gorm:"index;size:32;not null;default:'medium'"`
	Status         string   `json:"status" gorm:"index;size:32;not null;default:'open'"`
	Fingerprint    string   `json:"fingerprint" gorm:"index;size:64;not null;default:''"`
	Evidence       ResAttrs `json:"evidence,omitempty" gorm:"type:json"`
	Recommendation string   `json:"recommendation" gorm:"type:text"`

	FirstSeenAt       Time   `json:"firstSeenAt" gorm:"type:datetime;default:null"`
	LastSeenAt        Time   `json:"lastSeenAt" gorm:"type:datetime;default:null"`
	ResolvedAt        Time   `json:"resolvedAt" gorm:"type:datetime;default:null"`
	SuppressedUntil   Time   `json:"suppressedUntil" gorm:"type:datetime;default:null"`
	SuppressionReason string `json:"suppressionReason" gorm:"type:text"`
}

func (CloudRiskFinding) TableName() string {
	return "iac_cloud_risk_finding"
}

func (r CloudRiskFinding) Migrate(sess *db.Session) error {
	return r.AddUniqueIndex(sess, "unique_cloud_risk_fingerprint", "org_id", "fingerprint")
}
