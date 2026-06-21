// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CloudAccountResp struct {
	models.CloudAccount

	Ready                 bool                         `json:"ready"`
	RegionCount           int                          `json:"regionCount"`
	MissingCredentialKeys []string                     `json:"missingCredentialKeys"`
	HealthDetail          CloudAccountHealthDetailResp `json:"healthDetail"`
}

type CloudAccountValidationResp struct {
	Id                    models.Id `json:"id"`
	Provider              string    `json:"provider"`
	AccountId             string    `json:"accountId"`
	ValidationStatus      string    `json:"validationStatus"`
	ValidationMessage     string    `json:"validationMessage"`
	MissingCredentialKeys []string  `json:"missingCredentialKeys"`
	SupportedAssetTypes   []string  `json:"supportedAssetTypes"`
	Regions               []string  `json:"regions"`
}

type CloudAccountHealthResp struct {
	Id                  models.Id                    `json:"id"`
	Provider            string                       `json:"provider"`
	AccountId           string                       `json:"accountId"`
	HealthStatus        string                       `json:"healthStatus"`
	HealthMessage       string                       `json:"healthMessage"`
	ValidationStatus    string                       `json:"validationStatus"`
	ValidationMessage   string                       `json:"validationMessage"`
	LastHealthCheckedAt models.Time                  `json:"lastHealthCheckedAt"`
	LastValidatedAt     models.Time                  `json:"lastValidatedAt"`
	LastSyncAt          models.Time                  `json:"lastSyncAt"`
	HealthDetail        CloudAccountHealthDetailResp `json:"healthDetail"`
}

type CloudAccountHealthPolicyResp struct {
	Id                  models.Id   `json:"id"`
	Name                string      `json:"name"`
	SyncInterval        int         `json:"syncInterval"`
	HealthWindowSeconds int64       `json:"healthWindowSeconds"`
	HealthWindowText    string      `json:"healthWindowText"`
	EnabledPolicyCount  int64       `json:"enabledPolicyCount"`
	LastSyncTaskId      models.Id   `json:"lastSyncTaskId"`
	LastSyncStatus      string      `json:"lastSyncStatus"`
	LastSyncedAt        models.Time `json:"lastSyncedAt"`
}

type CloudAccountHealthScheduleResp struct {
	PolicyId            models.Id                      `json:"policyId"`
	PolicyName          string                         `json:"policyName"`
	ScheduleKey         string                         `json:"scheduleKey"`
	ScheduleName        string                         `json:"scheduleName"`
	Regions             []string                       `json:"regions"`
	AssetTypes          []string                       `json:"assetTypes"`
	SyncInterval        int                            `json:"syncInterval"`
	HealthWindowSeconds int64                          `json:"healthWindowSeconds"`
	HealthWindowText    string                         `json:"healthWindowText"`
	LastSyncTaskId      models.Id                      `json:"lastSyncTaskId"`
	LastSyncStatus      string                         `json:"lastSyncStatus"`
	LastError           string                         `json:"lastError"`
	LastSyncedAt        models.Time                    `json:"lastSyncedAt"`
	Stale               bool                           `json:"stale"`
	DueNow              bool                           `json:"dueNow"`
	LastSuccessTask     *CloudAccountSyncTaskBriefResp `json:"lastSuccessTask,omitempty"`
	LastFailureTask     *CloudAccountSyncTaskBriefResp `json:"lastFailureTask,omitempty"`
}

type CloudAccountSyncTaskBriefResp struct {
	Id           models.Id       `json:"id"`
	SyncPolicyId models.Id       `json:"syncPolicyId"`
	Status       string          `json:"status"`
	ErrorMessage string          `json:"errorMessage"`
	Stats        models.ResAttrs `json:"stats,omitempty"`
	StartedAt    models.Time     `json:"startedAt"`
	EndedAt      models.Time     `json:"endedAt"`
	CreatedAt    models.Time     `json:"createdAt"`
}

type CloudAccountHealthFailureImpactResp struct {
	TaskId     models.Id   `json:"taskId"`
	Status     string      `json:"status"`
	Category   string      `json:"category"`
	Message    string      `json:"message"`
	Retryable  bool        `json:"retryable"`
	RetryHint  string      `json:"retryHint"`
	OccurredAt models.Time `json:"occurredAt"`
}

type CloudAccountHealthDetailResp struct {
	HealthWindowSeconds int64                                `json:"healthWindowSeconds"`
	HealthWindowText    string                               `json:"healthWindowText"`
	EnabledPolicyCount  int64                                `json:"enabledPolicyCount"`
	Policy              *CloudAccountHealthPolicyResp        `json:"policy,omitempty"`
	Schedules           []CloudAccountHealthScheduleResp     `json:"schedules,omitempty"`
	LastSuccessTask     *CloudAccountSyncTaskBriefResp       `json:"lastSuccessTask,omitempty"`
	LastFailureTask     *CloudAccountSyncTaskBriefResp       `json:"lastFailureTask,omitempty"`
	FailureImpact       *CloudAccountHealthFailureImpactResp `json:"failureImpact,omitempty"`
}

type CloudAccountHealthSummaryResp struct {
	TotalCount       int64                         `json:"totalCount"`
	HealthyCount     int64                         `json:"healthyCount"`
	WarningCount     int64                         `json:"warningCount"`
	UnhealthyCount   int64                         `json:"unhealthyCount"`
	CheckedCount     int64                         `json:"checkedCount"`
	ShardIndex       int                           `json:"shardIndex"`
	ShardTotal       int                           `json:"shardTotal"`
	Concurrency      int                           `json:"concurrency"`
	Locked           bool                          `json:"locked"`
	LockSkipped      bool                          `json:"lockSkipped"`
	LockSkippedCount int                           `json:"lockSkippedCount"`
	Shards           []CloudAccountHealthShardResp `json:"shards,omitempty"`
	List             []CloudAccountHealthResp      `json:"list"`
}

type CloudAccountHealthShardResp struct {
	ShardIndex     int   `json:"shardIndex"`
	ShardTotal     int   `json:"shardTotal"`
	TotalCount     int64 `json:"totalCount"`
	HealthyCount   int64 `json:"healthyCount"`
	WarningCount   int64 `json:"warningCount"`
	UnhealthyCount int64 `json:"unhealthyCount"`
	CheckedCount   int64 `json:"checkedCount"`
	Locked         bool  `json:"locked"`
	LockSkipped    bool  `json:"lockSkipped"`
}

type CloudAccountRegionResp struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Default bool   `json:"default"`
	Source  string `json:"source"`
}

type CloudAccountRegionsResp struct {
	Id                    models.Id                `json:"id"`
	Provider              string                   `json:"provider"`
	AccountId             string                   `json:"accountId"`
	Regions               []CloudAccountRegionResp `json:"regions"`
	RegionNames           []string                 `json:"regionNames"`
	MissingCredentialKeys []string                 `json:"missingCredentialKeys"`
}

type CloudAccountPermissionResp struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type CloudAccountPermissionsResp struct {
	Id                    models.Id                    `json:"id"`
	Provider              string                       `json:"provider"`
	AccountId             string                       `json:"accountId"`
	ValidationStatus      string                       `json:"validationStatus"`
	ValidationMessage     string                       `json:"validationMessage"`
	MissingCredentialKeys []string                     `json:"missingCredentialKeys"`
	SupportedAssetTypes   []string                     `json:"supportedAssetTypes"`
	Regions               []string                     `json:"regions"`
	Permissions           []CloudAccountPermissionResp `json:"permissions"`
}
