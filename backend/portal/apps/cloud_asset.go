// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"fmt"
	"sort"

	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models"
	"cloudiac/portal/models/resps"
)

type cloudAssetCoverageLastSyncRow struct {
	LastSyncAt models.Time `gorm:"column:last_sync_at"`
}

type cloudAssetCoverageCountRow struct {
	Count int64 `gorm:"column:count"`
}

type cloudAssetCoverageAccountProviderRow struct {
	Provider     string `gorm:"column:provider"`
	AccountCount int64  `gorm:"column:account_count"`
}

// CloudAssetCoverage returns cloud asset management coverage for the asset center.
func CloudAssetCoverage(c *ctx.ServiceContext) (*resps.CloudAssetCoverageResp, e.Error) {
	if _, err := BackfillCmdbAssetsFromIac(c); err != nil {
		return nil, err
	}
	if err := RefreshCmdbAssetGovernanceFields(c); err != nil {
		return nil, err
	}

	metrics, err := cloudAssetCoverageMetrics(c)
	if err != nil {
		return nil, err
	}
	providers, err := cloudAssetCoverageProviders(c)
	if err != nil {
		return nil, err
	}
	accounts, err := cloudAssetCoverageAccounts(c)
	if err != nil {
		return nil, err
	}
	assetTypes, err := cloudAssetCoverageTypes(c)
	if err != nil {
		return nil, err
	}

	metrics.ProviderCount = int64(len(providers))
	metrics.AccountCount = int64(len(accounts))
	metrics.AssetTypeCount = int64(len(assetTypes))

	return &resps.CloudAssetCoverageResp{
		Metrics:    metrics,
		Providers:  providers,
		Accounts:   accounts,
		AssetTypes: assetTypes,
	}, nil
}

func cloudAssetCoverageMetrics(c *ctx.ServiceContext) (resps.CloudAssetCoverageMetricResp, e.Error) {
	var err error
	metrics := resps.CloudAssetCoverageMetricResp{}

	if metrics.TotalAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).Where("org_id = ?", c.OrgId)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.IacManagedAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and managed_by = ?", c.OrgId, models.CmdbManagedByIac)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.CloudCollectedAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and source = ?", c.OrgId, models.CmdbAssetSourceCloudCollect)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.CloudOnlyAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and managed_by = ?", c.OrgId, models.CmdbManagedByCloudOnly)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.LinkedCloudAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and managed_by = ?", c.OrgId, models.CmdbManagedByCloudLinked)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.UnownedAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and owner = ''", c.OrgId)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.HighRiskAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and compliance_risk in (?)", c.OrgId, []string{"high", "critical"})); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	regionCount := cloudAssetCoverageCountRow{}
	if err := c.DB().Model(&models.CmdbAsset{}).
		Select("count(distinct region) as count").
		Where("org_id = ? and region <> ''", c.OrgId).
		Scan(&regionCount); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	metrics.RegionCount = regionCount.Count

	lastSync := cloudAssetCoverageLastSyncRow{}
	if err := c.DB().Model(&models.CmdbAsset{}).
		Select("max(last_sync_at) as last_sync_at").
		Where("org_id = ?", c.OrgId).
		Scan(&lastSync); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	metrics.LastSyncAt = lastSync.LastSyncAt

	if metrics.TotalAssets > 0 {
		metrics.IacCoverageRate = float64(metrics.IacManagedAssets) / float64(metrics.TotalAssets) * 100
		metrics.CloudOnlyRate = float64(metrics.CloudOnlyAssets) / float64(metrics.TotalAssets) * 100
		metrics.OwnershipCoverageRate = float64(metrics.TotalAssets-metrics.UnownedAssets) / float64(metrics.TotalAssets) * 100
	}
	return metrics, nil
}

func cloudAssetCoverageProviders(c *ctx.ServiceContext) ([]resps.CloudAssetCoverageProviderResp, e.Error) {
	providers := make([]resps.CloudAssetCoverageProviderResp, 0)
	if err := c.DB().Model(&models.CmdbAsset{}).
		Select(`provider,
			count(*) as asset_count,
			count(distinct account_id) as account_count,
			count(distinct region) as region_count,
			sum(if(managed_by = ?, 1, 0)) as iac_managed_asset_count,
			sum(if(source = ?, 1, 0)) as cloud_collected_asset_count,
			sum(if(managed_by = ?, 1, 0)) as cloud_only_asset_count,
			sum(if(managed_by = ?, 1, 0)) as linked_cloud_asset_count,
			sum(if(owner = '', 1, 0)) as unowned_asset_count,
			sum(if(compliance_risk in ('high', 'critical'), 1, 0)) as high_risk_asset_count,
			max(last_sync_at) as last_sync_at`,
			models.CmdbManagedByIac,
			models.CmdbAssetSourceCloudCollect,
			models.CmdbManagedByCloudOnly,
			models.CmdbManagedByCloudLinked).
		Where("org_id = ? and provider <> ''", c.OrgId).
		Group("provider").
		Order("asset_count desc").
		Scan(&providers); err != nil {
		return nil, e.New(e.DBError, err)
	}

	accountRows := make([]cloudAssetCoverageAccountProviderRow, 0)
	if err := c.DB().Model(&models.CloudAccount{}).
		Select("provider, count(*) as account_count").
		Where("org_id = ? and provider <> ''", c.OrgId).
		Group("provider").
		Scan(&accountRows); err != nil {
		return nil, e.New(e.DBError, err)
	}
	providerIndex := make(map[string]int, len(providers))
	for i := range providers {
		providerIndex[providers[i].Provider] = i
	}
	for _, row := range accountRows {
		provider := normalizeProvider(row.Provider)
		if index, ok := providerIndex[provider]; ok {
			providers[index].AccountCount = row.AccountCount
			continue
		}
		providerIndex[provider] = len(providers)
		providers = append(providers, resps.CloudAssetCoverageProviderResp{
			Provider:     provider,
			AccountCount: row.AccountCount,
		})
	}
	sort.SliceStable(providers, func(i, j int) bool {
		if providers[i].AssetCount == providers[j].AssetCount {
			return providers[i].Provider < providers[j].Provider
		}
		return providers[i].AssetCount > providers[j].AssetCount
	})
	return providers, nil
}

func cloudAssetCoverageAccounts(c *ctx.ServiceContext) ([]resps.CloudAssetCoverageAccountResp, e.Error) {
	accounts := make([]models.CloudAccount, 0)
	if err := c.DB().Model(&models.CloudAccount{}).
		Where("org_id = ?", c.OrgId).
		Order("provider asc, name asc").
		Find(&accounts); err != nil {
		return nil, e.New(e.DBError, err)
	}

	rows := make([]resps.CloudAssetCoverageAccountResp, 0)
	if err := c.DB().Model(&models.CmdbAsset{}).
		Select(`provider,
			account_id,
			count(*) as asset_count,
			sum(if(managed_by = ?, 1, 0)) as iac_managed_asset_count,
			sum(if(source = ?, 1, 0)) as cloud_collected_asset_count,
			sum(if(managed_by = ?, 1, 0)) as cloud_only_asset_count,
			sum(if(managed_by = ?, 1, 0)) as linked_cloud_asset_count,
			sum(if(owner = '', 1, 0)) as unowned_asset_count,
			sum(if(compliance_risk in ('high', 'critical'), 1, 0)) as high_risk_asset_count,
			count(distinct region) as region_count,
			max(last_sync_at) as last_sync_at`,
			models.CmdbManagedByIac,
			models.CmdbAssetSourceCloudCollect,
			models.CmdbManagedByCloudOnly,
			models.CmdbManagedByCloudLinked).
		Where("org_id = ?", c.OrgId).
		Group("provider, account_id").
		Order("asset_count desc").
		Scan(&rows); err != nil {
		return nil, e.New(e.DBError, err)
	}

	byKey := make(map[string]resps.CloudAssetCoverageAccountResp, len(rows))
	for _, row := range rows {
		byKey[cloudAssetAccountKey(row.Provider, row.AccountId)] = row
	}

	result := make([]resps.CloudAssetCoverageAccountResp, 0, len(accounts)+len(rows))
	seen := make(map[string]bool)
	for _, account := range accounts {
		provider := normalizeProvider(account.Provider)
		accountId := firstNonEmpty(account.AccountId, string(account.Id))
		key := cloudAssetAccountKey(provider, accountId)
		row := byKey[key]
		row.Provider = provider
		row.AccountId = accountId
		row.AccountName = account.Name
		row.AccountRefId = account.Id
		row.AccountStatus = account.Status
		row.ValidationStatus = account.ValidationStatus
		result = append(result, row)
		seen[key] = true
	}
	for _, row := range rows {
		key := cloudAssetAccountKey(row.Provider, row.AccountId)
		if seen[key] {
			continue
		}
		if row.AccountId == "" {
			row.AccountName = "未标识账号"
		} else {
			row.AccountName = "未关联账号"
		}
		result = append(result, row)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].AssetCount == result[j].AssetCount {
			return fmt.Sprintf("%s/%s", result[i].Provider, result[i].AccountName) < fmt.Sprintf("%s/%s", result[j].Provider, result[j].AccountName)
		}
		return result[i].AssetCount > result[j].AssetCount
	})
	return result, nil
}

func cloudAssetCoverageTypes(c *ctx.ServiceContext) ([]resps.CloudAssetCoverageTypeResp, e.Error) {
	assetTypes := make([]resps.CloudAssetCoverageTypeResp, 0)
	if err := c.DB().Model(&models.CmdbAsset{}).
		Select(`asset_type,
			count(*) as asset_count,
			sum(if(managed_by = ?, 1, 0)) as iac_managed_asset_count,
			sum(if(source = ?, 1, 0)) as cloud_collected_asset_count,
			sum(if(managed_by = ?, 1, 0)) as cloud_only_asset_count,
			sum(if(owner = '', 1, 0)) as unowned_asset_count,
			sum(if(compliance_risk in ('high', 'critical'), 1, 0)) as high_risk_asset_count,
			max(last_sync_at) as last_sync_at`,
			models.CmdbManagedByIac,
			models.CmdbAssetSourceCloudCollect,
			models.CmdbManagedByCloudOnly).
		Where("org_id = ? and asset_type <> ''", c.OrgId).
		Group("asset_type").
		Order("asset_count desc").
		Scan(&assetTypes); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return assetTypes, nil
}

func cloudAssetAccountKey(provider string, accountId string) string {
	return fmt.Sprintf("%s/%s", normalizeProvider(provider), accountId)
}
