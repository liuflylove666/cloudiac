// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CloudOverviewResp struct {
	Metrics         CloudOverviewMetricResp     `json:"metrics"`
	Providers       []CloudOverviewProviderResp `json:"providers"`
	AssetSources    []CloudOverviewCountResp    `json:"assetSources"`
	AssetTypes      []CloudOverviewCountResp    `json:"assetTypes"`
	RiskLevels      []CloudOverviewCountResp    `json:"riskLevels"`
	RecentSyncTasks []CmdbSyncTaskResp          `json:"recentSyncTasks"`
	Actions         []CloudOverviewActionResp   `json:"actions"`
}

type CloudOverviewMetricResp struct {
	AccountTotal         int64       `json:"accountTotal"`
	AccountEnabled       int64       `json:"accountEnabled"`
	AccountReady         int64       `json:"accountReady"`
	AccountInvalid       int64       `json:"accountInvalid"`
	AccountUnsynced      int64       `json:"accountUnsynced"`
	AssetTotal           int64       `json:"assetTotal"`
	IacManagedAssets     int64       `json:"iacManagedAssets"`
	IacDirectAssets      int64       `json:"iacDirectAssets"`
	IacLinkedAssets      int64       `json:"iacLinkedAssets"`
	CloudCollectedAssets int64       `json:"cloudCollectedAssets"`
	CloudLinkedAssets    int64       `json:"cloudLinkedAssets"`
	CloudOnlyAssets      int64       `json:"cloudOnlyAssets"`
	UnownedAssets        int64       `json:"unownedAssets"`
	HighRiskAssets       int64       `json:"highRiskAssets"`
	CriticalRiskAssets   int64       `json:"criticalRiskAssets"`
	RunningSyncTasks     int64       `json:"runningSyncTasks"`
	FailedSyncTasks      int64       `json:"failedSyncTasks"`
	CoverageRate         float64     `json:"coverageRate"`
	GovernanceRate       float64     `json:"governanceRate"`
	CloudOnlyRate        float64     `json:"cloudOnlyRate"`
	LastSyncAt           models.Time `json:"lastSyncAt"`
}

type CloudOverviewProviderResp struct {
	Provider              string      `json:"provider" gorm:"column:provider"`
	AccountCount          int64       `json:"accountCount" gorm:"column:account_count"`
	EnabledAccountCount   int64       `json:"enabledAccountCount" gorm:"column:enabled_account_count"`
	InvalidAccountCount   int64       `json:"invalidAccountCount" gorm:"column:invalid_account_count"`
	AssetCount            int64       `json:"assetCount" gorm:"column:asset_count"`
	IacManagedAssetCount  int64       `json:"iacManagedAssetCount" gorm:"column:iac_managed_asset_count"`
	CloudLinkedAssetCount int64       `json:"cloudLinkedAssetCount" gorm:"column:cloud_linked_asset_count"`
	CloudOnlyAssetCount   int64       `json:"cloudOnlyAssetCount" gorm:"column:cloud_only_asset_count"`
	LastSyncAt            models.Time `json:"lastSyncAt" gorm:"column:last_sync_at"`
}

type CloudOverviewCountResp struct {
	Key   string `json:"key" gorm:"column:key"`
	Label string `json:"label" gorm:"column:label"`
	Count int64  `json:"count" gorm:"column:count"`
}

type CloudOverviewActionResp struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Level       string `json:"level"`
	Count       int64  `json:"count"`
	Target      string `json:"target"`
}
