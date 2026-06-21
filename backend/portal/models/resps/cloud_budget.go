// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CloudBudgetResp struct {
	models.CloudBudget

	ProjectName      string  `json:"projectName"`
	EnvName          string  `json:"envName"`
	CurrentAmount    float64 `json:"currentAmount"`
	UsagePercent     float64 `json:"usagePercent"`
	RemainingAmount  float64 `json:"remainingAmount"`
	ThresholdReached bool    `json:"thresholdReached"`
	BudgetExceeded   bool    `json:"budgetExceeded"`
}

type CloudBudgetSummaryResp struct {
	Period             string  `json:"period"`
	Currency           string  `json:"currency"`
	TotalLimitAmount   float64 `json:"totalLimitAmount"`
	TotalCurrentAmount float64 `json:"totalCurrentAmount"`
	BudgetCount        int64   `json:"budgetCount"`
	ThresholdCount     int64   `json:"thresholdCount"`
	ExceededCount      int64   `json:"exceededCount"`
}

type CloudBudgetEvaluateResp struct {
	Period             string  `json:"period"`
	Currency           string  `json:"currency"`
	Force              bool    `json:"force"`
	TotalCount         int64   `json:"totalCount"`
	EvaluatedCount     int64   `json:"evaluatedCount"`
	SkippedCount       int64   `json:"skippedCount"`
	ThresholdCount     int64   `json:"thresholdCount"`
	ExceededCount      int64   `json:"exceededCount"`
	NotificationCount  int64   `json:"notificationCount"`
	TotalLimitAmount   float64 `json:"totalLimitAmount"`
	TotalCurrentAmount float64 `json:"totalCurrentAmount"`
}
