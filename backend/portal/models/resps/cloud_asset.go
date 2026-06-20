// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CloudAssetCoverageResp struct {
	Metrics    CloudAssetCoverageMetricResp     `json:"metrics"`
	Providers  []CloudAssetCoverageProviderResp `json:"providers"`
	Accounts   []CloudAssetCoverageAccountResp  `json:"accounts"`
	AssetTypes []CloudAssetCoverageTypeResp     `json:"assetTypes"`
}

type CloudAssetCoverageMetricResp struct {
	TotalAssets           int64       `json:"totalAssets"`
	IacManagedAssets      int64       `json:"iacManagedAssets"`
	CloudCollectedAssets  int64       `json:"cloudCollectedAssets"`
	CloudOnlyAssets       int64       `json:"cloudOnlyAssets"`
	LinkedCloudAssets     int64       `json:"linkedCloudAssets"`
	UnownedAssets         int64       `json:"unownedAssets"`
	HighRiskAssets        int64       `json:"highRiskAssets"`
	ProviderCount         int64       `json:"providerCount"`
	AccountCount          int64       `json:"accountCount"`
	AssetTypeCount        int64       `json:"assetTypeCount"`
	RegionCount           int64       `json:"regionCount"`
	IacCoverageRate       float64     `json:"iacCoverageRate"`
	CloudOnlyRate         float64     `json:"cloudOnlyRate"`
	OwnershipCoverageRate float64     `json:"ownershipCoverageRate"`
	LastSyncAt            models.Time `json:"lastSyncAt"`
}

type CloudAssetCoverageProviderResp struct {
	Provider                 string      `json:"provider" gorm:"column:provider"`
	AccountCount             int64       `json:"accountCount" gorm:"column:account_count"`
	AssetCount               int64       `json:"assetCount" gorm:"column:asset_count"`
	IacManagedAssetCount     int64       `json:"iacManagedAssetCount" gorm:"column:iac_managed_asset_count"`
	CloudCollectedAssetCount int64       `json:"cloudCollectedAssetCount" gorm:"column:cloud_collected_asset_count"`
	CloudOnlyAssetCount      int64       `json:"cloudOnlyAssetCount" gorm:"column:cloud_only_asset_count"`
	LinkedCloudAssetCount    int64       `json:"linkedCloudAssetCount" gorm:"column:linked_cloud_asset_count"`
	UnownedAssetCount        int64       `json:"unownedAssetCount" gorm:"column:unowned_asset_count"`
	HighRiskAssetCount       int64       `json:"highRiskAssetCount" gorm:"column:high_risk_asset_count"`
	RegionCount             int64       `json:"regionCount" gorm:"column:region_count"`
	LastSyncAt               models.Time `json:"lastSyncAt" gorm:"column:last_sync_at"`
}

type CloudAssetCoverageAccountResp struct {
	Provider                 string      `json:"provider" gorm:"column:provider"`
	AccountId                string      `json:"accountId" gorm:"column:account_id"`
	AccountName              string      `json:"accountName"`
	AccountRefId             models.Id   `json:"accountRefId,omitempty"`
	AccountStatus            string      `json:"accountStatus,omitempty"`
	ValidationStatus         string      `json:"validationStatus,omitempty"`
	AssetCount               int64       `json:"assetCount" gorm:"column:asset_count"`
	IacManagedAssetCount     int64       `json:"iacManagedAssetCount" gorm:"column:iac_managed_asset_count"`
	CloudCollectedAssetCount int64       `json:"cloudCollectedAssetCount" gorm:"column:cloud_collected_asset_count"`
	CloudOnlyAssetCount      int64       `json:"cloudOnlyAssetCount" gorm:"column:cloud_only_asset_count"`
	LinkedCloudAssetCount    int64       `json:"linkedCloudAssetCount" gorm:"column:linked_cloud_asset_count"`
	UnownedAssetCount        int64       `json:"unownedAssetCount" gorm:"column:unowned_asset_count"`
	HighRiskAssetCount       int64       `json:"highRiskAssetCount" gorm:"column:high_risk_asset_count"`
	RegionCount             int64       `json:"regionCount" gorm:"column:region_count"`
	LastSyncAt               models.Time `json:"lastSyncAt" gorm:"column:last_sync_at"`
}

type CloudAssetCoverageTypeResp struct {
	AssetType                string      `json:"assetType" gorm:"column:asset_type"`
	AssetCount               int64       `json:"assetCount" gorm:"column:asset_count"`
	IacManagedAssetCount     int64       `json:"iacManagedAssetCount" gorm:"column:iac_managed_asset_count"`
	CloudCollectedAssetCount int64       `json:"cloudCollectedAssetCount" gorm:"column:cloud_collected_asset_count"`
	CloudOnlyAssetCount      int64       `json:"cloudOnlyAssetCount" gorm:"column:cloud_only_asset_count"`
	UnownedAssetCount        int64       `json:"unownedAssetCount" gorm:"column:unowned_asset_count"`
	HighRiskAssetCount       int64       `json:"highRiskAssetCount" gorm:"column:high_risk_asset_count"`
	LastSyncAt               models.Time `json:"lastSyncAt" gorm:"column:last_sync_at"`
}
