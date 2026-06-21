// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package models

import "cloudiac/portal/libs/db"

const (
	CloudBudgetScopeOrg          = "org"
	CloudBudgetScopeProject      = "project"
	CloudBudgetScopeEnv          = "env"
	CloudBudgetScopeProvider     = "provider"
	CloudBudgetScopeAccount      = "account"
	CloudBudgetScopeApplication  = "application"
	CloudBudgetScopeBusinessLine = "business_line"
	CloudBudgetScopeCostCenter   = "cost_center"
	CloudBudgetScopeOwner        = "owner"

	CloudBudgetStatusEnabled  = Enable
	CloudBudgetStatusDisabled = Disable
)

type CloudBudget struct {
	SoftDeleteModel

	OrgId              Id      `json:"orgId" gorm:"index;size:32;not null"`
	ProjectId          Id      `json:"projectId" gorm:"index;size:32;not null;default:''"`
	EnvId              Id      `json:"envId" gorm:"index;size:32;not null;default:''"`
	CloudAccountId     Id      `json:"cloudAccountId" gorm:"index;size:32;not null;default:''"`
	Name               string  `json:"name" gorm:"size:128;not null"`
	Description        string  `json:"description" gorm:"size:255;not null;default:''"`
	Scope              string  `json:"scope" gorm:"index;size:32;not null;default:'org'"`
	Provider           string  `json:"provider" gorm:"index;size:64;not null;default:''"`
	AccountId          string  `json:"accountId" gorm:"index;size:128;not null;default:''"`
	Region             string  `json:"region" gorm:"index;size:128;not null;default:''"`
	Application        string  `json:"application" gorm:"index;size:128;not null;default:''"`
	BusinessLine       string  `json:"businessLine" gorm:"index;size:128;not null;default:''"`
	CostCenter         string  `json:"costCenter" gorm:"index;size:128;not null;default:''"`
	Owner              string  `json:"owner" gorm:"index;size:128;not null;default:''"`
	Period             string  `json:"period" gorm:"index;size:32;not null;default:''"`
	Currency           string  `json:"currency" gorm:"index;size:16;not null;default:'CNY'"`
	LimitAmount        float64 `json:"limitAmount" gorm:"type:decimal(18,4);not null;default:0"`
	ThresholdPercent   float64 `json:"thresholdPercent" gorm:"type:decimal(8,2);not null;default:80"`
	EvaluationInterval int     `json:"evaluationInterval" gorm:"not null;default:3600"`
	Status             string  `json:"status" gorm:"index;size:32;not null;default:'enable'"`
	LastAmount         float64 `json:"lastAmount" gorm:"type:decimal(18,4);not null;default:0"`
	LastUsagePercent   float64 `json:"lastUsagePercent" gorm:"type:decimal(8,2);not null;default:0"`
	LastEvaluatedAt    Time    `json:"lastEvaluatedAt" gorm:"type:datetime;default:null"`
	LastExceededAt     Time    `json:"lastExceededAt" gorm:"type:datetime;default:null"`
}

func (CloudBudget) TableName() string {
	return "iac_cloud_budget"
}

func (b CloudBudget) Migrate(sess *db.Session) error {
	return b.AddUniqueIndex(sess, "unique_cloud_budget_name", "org_id", "name")
}
