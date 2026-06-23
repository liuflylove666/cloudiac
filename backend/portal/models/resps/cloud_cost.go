// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CloudCostRecordResp struct {
	models.CloudCostRecord

	ProjectName string `json:"projectName"`
	EnvName     string `json:"envName"`
	AssetName   string `json:"assetName"`
}

type CloudCostGroupResp struct {
	Key      string  `json:"key" gorm:"column:key"`
	Label    string  `json:"label" gorm:"column:label"`
	Amount   float64 `json:"amount" gorm:"column:amount"`
	Currency string  `json:"currency" gorm:"column:currency"`
	Count    int64   `json:"count" gorm:"column:count"`
}

type CloudCostSummaryResp struct {
	Period         string               `json:"period"`
	TotalAmount    float64              `json:"totalAmount"`
	Currency       string               `json:"currency"`
	RecordCount    int64                `json:"recordCount"`
	MatchedCount   int64                `json:"matchedCount"`
	UnmatchedCount int64                `json:"unmatchedCount"`
	Providers      []CloudCostGroupResp `json:"providers"`
	Projects       []CloudCostGroupResp `json:"projects"`
	Applications   []CloudCostGroupResp `json:"applications"`
	BusinessLines  []CloudCostGroupResp `json:"businessLines"`
	CostCenters    []CloudCostGroupResp `json:"costCenters"`
}

type CloudCostImportResp struct {
	Imported       int      `json:"imported"`
	MatchedCount   int      `json:"matchedCount"`
	UnmatchedCount int      `json:"unmatchedCount"`
	Providers      []string `json:"providers"`
	Sources        []string `json:"sources"`
	Periods        []string `json:"periods"`
	Currencies     []string `json:"currencies"`
}

type CloudCostPullResp struct {
	CloudCostImportResp

	Provider             string            `json:"provider"`
	Mode                 string            `json:"mode"`
	Source               string            `json:"source"`
	SourceURL            string            `json:"sourceUrl,omitempty"`
	SourceIndexURL       string            `json:"sourceIndexUrl,omitempty"`
	SourceObjectProvider string            `json:"sourceObjectProvider,omitempty"`
	SourceObjectEndpoint string            `json:"sourceObjectEndpoint,omitempty"`
	SourceObjectBucket   string            `json:"sourceObjectBucket,omitempty"`
	SourceObjectPrefix   string            `json:"sourceObjectPrefix,omitempty"`
	Cursor               string            `json:"cursor,omitempty"`
	NextCursor           string            `json:"nextCursor,omitempty"`
	FileCount            int               `json:"fileCount"`
	Files                []models.ResAttrs `json:"files,omitempty"`
	SkippedFiles         []models.ResAttrs `json:"skippedFiles,omitempty"`
	CloudAccountId       models.Id         `json:"cloudAccountId,omitempty"`
	AccountId            string            `json:"accountId,omitempty"`
	Period               string            `json:"period"`
	Currency             string            `json:"currency"`
}

type CloudCostSyncTaskResp struct {
	models.CloudCostSyncTask

	CreatorName                 string `json:"creatorName"`
	CloudAccountName            string `json:"cloudAccountName"`
	SourceURLPreview            string `json:"sourceUrlPreview,omitempty"`
	SourceIndexURLPreview       string `json:"sourceIndexUrlPreview,omitempty"`
	SourceObjectEndpointPreview string `json:"sourceObjectEndpointPreview,omitempty"`
}

type CloudCostSyncTaskDetailResp struct {
	CloudCostSyncTaskResp

	Logs []models.CloudCostSyncTaskLog `json:"logs"`
}

type CloudCostSyncScheduleResp struct {
	models.CloudCostSyncSchedule

	CreatorName                  string          `json:"creatorName"`
	CloudAccountName             string          `json:"cloudAccountName"`
	SourceURLPreview             string          `json:"sourceUrlPreview,omitempty"`
	SourceIndexURLPreview        string          `json:"sourceIndexUrlPreview,omitempty"`
	SourceObjectEndpointPreview  string          `json:"sourceObjectEndpointPreview,omitempty"`
	FailureCount                 int             `json:"failureCount"`
	MaxRetryAttempts             int             `json:"maxRetryAttempts"`
	RetryBackoffSeconds          int             `json:"retryBackoffSeconds"`
	NotifyOnFailure              bool            `json:"notifyOnFailure"`
	AutoPauseOnFailure           bool            `json:"autoPauseOnFailure"`
	NotificationOwner            string          `json:"notificationOwner,omitempty"`
	NotificationRoutes           []string        `json:"notificationRoutes,omitempty"`
	NotificationAssignees        []string        `json:"notificationAssignees,omitempty"`
	NotificationSilenceMinutes   int             `json:"notificationSilenceMinutes,omitempty"`
	NotificationWindows          []string        `json:"notificationWindows,omitempty"`
	NotificationFailureRoutes    models.ResAttrs `json:"notificationFailureRoutes,omitempty"`
	NotificationEscalationAt     int             `json:"notificationEscalationAt,omitempty"`
	NotificationEscalationRoutes []string        `json:"notificationEscalationRoutes,omitempty"`
	NextRetryAt                  string          `json:"nextRetryAt,omitempty"`
	AutoPausedAt                 string          `json:"autoPausedAt,omitempty"`
}

type CloudCostSyncScheduleRunResp struct {
	TotalCount       int      `json:"totalCount"`
	TriggeredCount   int      `json:"triggeredCount"`
	CompleteCount    int      `json:"completeCount"`
	FailedCount      int      `json:"failedCount"`
	SkippedCount     int      `json:"skippedCount"`
	LockSkippedCount int      `json:"lockSkippedCount"`
	TaskIds          []string `json:"taskIds"`
}

type CloudCostTrendResp struct {
	Period   string  `json:"period" gorm:"column:period"`
	Amount   float64 `json:"amount" gorm:"column:amount"`
	Currency string  `json:"currency" gorm:"column:currency"`
}

type CloudCostInsightResp struct {
	models.CloudCostInsight

	ProjectName string `json:"projectName"`
	EnvName     string `json:"envName"`
	AssetName   string `json:"assetName"`
}

type CloudCostInsightSummaryResp struct {
	Period            string  `json:"period"`
	Currency          string  `json:"currency"`
	TotalCount        int64   `json:"totalCount"`
	OpenCount         int64   `json:"openCount"`
	ResolvedCount     int64   `json:"resolvedCount"`
	IgnoredCount      int64   `json:"ignoredCount"`
	HighCount         int64   `json:"highCount"`
	AnomalyCount      int64   `json:"anomalyCount"`
	OptimizationCount int64   `json:"optimizationCount"`
	PotentialSavings  float64 `json:"potentialSavings"`
}
