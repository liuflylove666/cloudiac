// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CloudSyncPolicyResp struct {
	models.CloudSyncPolicy

	CloudAccountName      string                        `json:"cloudAccountName"`
	CloudAccountReady     bool                          `json:"cloudAccountReady"`
	MissingCredentialKeys []string                      `json:"missingCredentialKeys"`
	SupportedAssetTypes   []string                      `json:"supportedAssetTypes"`
	Protection            CloudSyncPolicyProtectionResp `json:"protection"`
	Schedules             []CloudSyncPolicyScheduleResp `json:"schedules"`
}

type CloudSyncPolicyProtectionResp struct {
	PausedNow            bool     `json:"pausedNow"`
	PauseReason          string   `json:"pauseReason"`
	PauseWindows         []string `json:"pauseWindows"`
	MaxRunsPerDay        int      `json:"maxRunsPerDay"`
	RunsToday            int      `json:"runsToday"`
	MaxTriggeredPerRun   int      `json:"maxTriggeredPerRun"`
	MaxConcurrentRunning int      `json:"maxConcurrentRunning"`
	RunningCount         int      `json:"runningCount"`
}

type CloudSyncPolicyRunResp struct {
	TotalCount               int      `json:"totalCount"`
	TriggeredCount           int      `json:"triggeredCount"`
	FailedCount              int      `json:"failedCount"`
	SkippedCount             int      `json:"skippedCount"`
	LockSkippedCount         int      `json:"lockSkippedCount"`
	ProtectionSkippedCount   int      `json:"protectionSkippedCount"`
	ProtectionSkippedReasons []string `json:"protectionSkippedReasons"`
	TaskIds                  []string `json:"taskIds"`
}

type CloudSyncPolicyScheduleResp struct {
	Key            string   `json:"key"`
	Name           string   `json:"name"`
	Regions        []string `json:"regions"`
	AssetTypes     []string `json:"assetTypes"`
	SyncInterval   int      `json:"syncInterval"`
	NextSyncAt     string   `json:"nextSyncAt"`
	LastSyncedAt   string   `json:"lastSyncedAt"`
	LastSyncTaskId string   `json:"lastSyncTaskId"`
	LastSyncStatus string   `json:"lastSyncStatus"`
	LastError      string   `json:"lastError"`
	FailureCount   int      `json:"failureCount"`
	DueNow         bool     `json:"dueNow"`
}
