// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"fmt"
	"time"

	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/libs/db"
	"cloudiac/portal/models"
	"cloudiac/portal/models/resps"
)

type cloudOverviewAssetProviderRow struct {
	Provider              string `gorm:"column:provider"`
	AssetCount            int64  `gorm:"column:asset_count"`
	IacManagedAssetCount  int64  `gorm:"column:iac_managed_asset_count"`
	CloudLinkedAssetCount int64  `gorm:"column:cloud_linked_asset_count"`
	CloudOnlyAssetCount   int64  `gorm:"column:cloud_only_asset_count"`
}

type cloudOverviewLastSyncRow struct {
	LastSyncAt models.Time `gorm:"column:last_sync_at"`
}

func CloudOverview(c *ctx.ServiceContext) (*resps.CloudOverviewResp, e.Error) {
	if _, err := BackfillCmdbAssetsFromIac(c); err != nil {
		return nil, err
	}

	metrics, err := cloudOverviewMetrics(c)
	if err != nil {
		return nil, err
	}
	providers, err := cloudOverviewProviders(c)
	if err != nil {
		return nil, err
	}
	assetSources, err := cloudOverviewCountRows(c.DB().Model(&models.CmdbAsset{}).
		Select("source as `key`, source as label, count(*) as count").
		Where("org_id = ? and source <> ''", c.OrgId).
		Group("source").
		Order("count desc"))
	if err != nil {
		return nil, err
	}
	assetTypes, err := cloudOverviewCountRows(c.DB().Model(&models.CmdbAsset{}).
		Select("asset_type as `key`, asset_type as label, count(*) as count").
		Where("org_id = ? and asset_type <> ''", c.OrgId).
		Group("asset_type").
		Order("count desc").
		Limit(10))
	if err != nil {
		return nil, err
	}
	riskLevels, err := cloudOverviewCountRows(c.DB().Model(&models.CmdbAsset{}).
		Select("compliance_risk as `key`, compliance_risk as label, count(*) as count").
		Where("org_id = ? and compliance_risk <> ''", c.OrgId).
		Group("compliance_risk").
		Order("count desc"))
	if err != nil {
		return nil, err
	}
	recentSyncTasks, err := cloudOverviewRecentSyncTasks(c)
	if err != nil {
		return nil, err
	}

	return &resps.CloudOverviewResp{
		Metrics:         metrics,
		Providers:       providers,
		AssetSources:    assetSources,
		AssetTypes:      assetTypes,
		RiskLevels:      riskLevels,
		RecentSyncTasks: recentSyncTasks,
		Actions:         cloudOverviewActions(c.OrgId, metrics),
	}, nil
}

func cloudOverviewMetrics(c *ctx.ServiceContext) (resps.CloudOverviewMetricResp, e.Error) {
	var err error
	since := time.Now().Add(-24 * time.Hour)
	metrics := resps.CloudOverviewMetricResp{}

	if metrics.AccountTotal, err = cloudOverviewCount(c.DB().Model(&models.CloudAccount{}).Where("org_id = ?", c.OrgId)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.AccountEnabled, err = cloudOverviewCount(c.DB().Model(&models.CloudAccount{}).
		Where("org_id = ? and status = ?", c.OrgId, models.CloudAccountStatusEnabled)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.AccountReady, err = cloudOverviewCount(c.DB().Model(&models.CloudAccount{}).
		Where("org_id = ? and status = ? and validation_status = ?", c.OrgId, models.CloudAccountStatusEnabled, models.CloudAccountValidationValid)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.AccountInvalid, err = cloudOverviewCount(c.DB().Model(&models.CloudAccount{}).
		Where("org_id = ? and validation_status = ?", c.OrgId, models.CloudAccountValidationInvalid)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.AccountUnsynced, err = cloudOverviewCount(c.DB().Model(&models.CloudAccount{}).
		Where("org_id = ? and status = ? and last_sync_at is null", c.OrgId, models.CloudAccountStatusEnabled)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.AssetTotal, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).Where("org_id = ?", c.OrgId)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.IacDirectAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and source = ?", c.OrgId, models.CmdbAssetSourceIacResource)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.IacLinkedAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and source <> ? and iac_resource_id <> ''", c.OrgId, models.CmdbAssetSourceIacResource)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.IacManagedAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and (source = ? or iac_resource_id <> '')", c.OrgId, models.CmdbAssetSourceIacResource)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.CloudCollectedAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and source = ?", c.OrgId, models.CmdbAssetSourceCloudCollect)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.CloudLinkedAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and source = ? and (project_id <> '' or env_id <> '' or iac_resource_id <> '')", c.OrgId, models.CmdbAssetSourceCloudCollect)); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.CloudOnlyAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and source = ? and project_id = '' and env_id = '' and iac_resource_id = ''", c.OrgId, models.CmdbAssetSourceCloudCollect)); err != nil {
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
	if metrics.CriticalRiskAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and compliance_risk = ?", c.OrgId, "critical")); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.RunningSyncTasks, err = cloudOverviewCount(c.DB().Model(&models.CmdbSyncTask{}).
		Where("org_id = ? and status in (?)", c.OrgId, []string{models.CmdbSyncTaskPending, models.CmdbSyncTaskRunning})); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	if metrics.FailedSyncTasks, err = cloudOverviewCount(c.DB().Model(&models.CmdbSyncTask{}).
		Where("org_id = ? and status = ? and updated_at >= ?", c.OrgId, models.CmdbSyncTaskFailed, since)); err != nil {
		return metrics, e.New(e.DBError, err)
	}

	lastSync := cloudOverviewLastSyncRow{}
	if err := c.DB().Model(&models.CmdbSyncTask{}).
		Select("max(ended_at) as last_sync_at").
		Where("org_id = ? and status = ?", c.OrgId, models.CmdbSyncTaskComplete).
		Scan(&lastSync); err != nil {
		return metrics, e.New(e.DBError, err)
	}
	metrics.LastSyncAt = lastSync.LastSyncAt
	if metrics.AssetTotal > 0 {
		metrics.CoverageRate = float64(metrics.IacManagedAssets) / float64(metrics.AssetTotal) * 100
		metrics.GovernanceRate = float64(metrics.AssetTotal-metrics.CloudOnlyAssets) / float64(metrics.AssetTotal) * 100
		metrics.CloudOnlyRate = float64(metrics.CloudOnlyAssets) / float64(metrics.AssetTotal) * 100
	}
	return metrics, nil
}

func cloudOverviewProviders(c *ctx.ServiceContext) ([]resps.CloudOverviewProviderResp, e.Error) {
	providers := make([]resps.CloudOverviewProviderResp, 0)
	if err := c.DB().Model(&models.CloudAccount{}).
		Select(`provider,
			count(*) as account_count,
			sum(if(status = ?, 1, 0)) as enabled_account_count,
			sum(if(validation_status = ?, 1, 0)) as invalid_account_count,
			max(last_sync_at) as last_sync_at`, models.CloudAccountStatusEnabled, models.CloudAccountValidationInvalid).
		Where("org_id = ? and provider <> ''", c.OrgId).
		Group("provider").
		Order("provider asc").
		Scan(&providers); err != nil {
		return nil, e.New(e.DBError, err)
	}

	assetRows := make([]cloudOverviewAssetProviderRow, 0)
	if err := c.DB().Model(&models.CmdbAsset{}).
		Select(`provider,
			count(*) as asset_count,
			sum(if(source = ? or iac_resource_id <> '', 1, 0)) as iac_managed_asset_count,
			sum(if(source = ? and (project_id <> '' or env_id <> '' or iac_resource_id <> ''), 1, 0)) as cloud_linked_asset_count,
			sum(if(source = ? and project_id = '' and env_id = '' and iac_resource_id = '', 1, 0)) as cloud_only_asset_count`,
			models.CmdbAssetSourceIacResource, models.CmdbAssetSourceCloudCollect, models.CmdbAssetSourceCloudCollect).
		Where("org_id = ? and provider <> ''", c.OrgId).
		Group("provider").
		Scan(&assetRows); err != nil {
		return nil, e.New(e.DBError, err)
	}

	providerIndex := make(map[string]int, len(providers))
	for i := range providers {
		providerIndex[providers[i].Provider] = i
	}
	for _, row := range assetRows {
		if index, ok := providerIndex[row.Provider]; ok {
			providers[index].AssetCount = row.AssetCount
			providers[index].IacManagedAssetCount = row.IacManagedAssetCount
			providers[index].CloudLinkedAssetCount = row.CloudLinkedAssetCount
			providers[index].CloudOnlyAssetCount = row.CloudOnlyAssetCount
			continue
		}
		providers = append(providers, resps.CloudOverviewProviderResp{
			Provider:              row.Provider,
			AssetCount:            row.AssetCount,
			IacManagedAssetCount:  row.IacManagedAssetCount,
			CloudLinkedAssetCount: row.CloudLinkedAssetCount,
			CloudOnlyAssetCount:   row.CloudOnlyAssetCount,
		})
	}
	return providers, nil
}

func cloudOverviewRecentSyncTasks(c *ctx.ServiceContext) ([]resps.CmdbSyncTaskResp, e.Error) {
	modelTasks := make([]models.CmdbSyncTask, 0)
	if err := c.DB().Model(&models.CmdbSyncTask{}).
		Where("org_id = ?", c.OrgId).
		Order("created_at desc").
		Limit(5).
		Scan(&modelTasks); err != nil {
		return nil, e.New(e.DBError, err)
	}
	tasks := make([]resps.CmdbSyncTaskResp, 0, len(modelTasks))
	for _, task := range modelTasks {
		tasks = append(tasks, cmdbSyncTaskResp(task))
	}
	return tasks, nil
}

func cloudOverviewCountRows(query *db.Session) ([]resps.CloudOverviewCountResp, e.Error) {
	rows := make([]resps.CloudOverviewCountResp, 0)
	if err := query.Scan(&rows); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return rows, nil
}

func cloudOverviewCount(query *db.Session) (int64, error) {
	return query.Count()
}

func cloudOverviewActions(orgId models.Id, metrics resps.CloudOverviewMetricResp) []resps.CloudOverviewActionResp {
	actions := make([]resps.CloudOverviewActionResp, 0)
	accountTarget := fmt.Sprintf("/org/%s/m-cloud-account", orgId)
	assetTarget := fmt.Sprintf("/org/%s/m-cloud-assets", orgId)
	if metrics.AccountInvalid > 0 {
		actions = append(actions, cloudOverviewAction("invalidAccounts", "处理异常云账号", "存在凭证或配置不完整的云账号，需要重新验证。", "error", metrics.AccountInvalid, accountTarget))
	}
	if metrics.AccountUnsynced > 0 {
		actions = append(actions, cloudOverviewAction("unsyncedAccounts", "补齐账号同步", "启用账号尚未完成过云采集，资产覆盖可能不完整。", "warning", metrics.AccountUnsynced, accountTarget))
	}
	if metrics.CloudOnlyAssets > 0 {
		actions = append(actions, cloudOverviewAction("cloudOnlyAssets", "治理未纳管资产", "云上存在未关联 IaC 项目/环境的资产。", "warning", metrics.CloudOnlyAssets, assetTarget))
	}
	if metrics.AssetTotal > 0 && metrics.CoverageRate < 80 {
		actions = append(actions, cloudOverviewAction("iacCoverage", "提升 IaC 覆盖率", "IaC 覆盖率低于 80%，建议优先关联云采集资产和 IaC 项目环境。", "warning", metrics.AssetTotal-metrics.IacManagedAssets, assetTarget))
	}
	if metrics.UnownedAssets > 0 {
		actions = append(actions, cloudOverviewAction("unownedAssets", "补齐资产负责人", "部分资产缺少负责人，影响风险和成本归属。", "warning", metrics.UnownedAssets, assetTarget))
	}
	if metrics.HighRiskAssets > 0 {
		actions = append(actions, cloudOverviewAction("highRiskAssets", "关注高风险资产", "存在高/严重合规风险资产，需要优先确认。", "error", metrics.HighRiskAssets, assetTarget))
	}
	if metrics.FailedSyncTasks > 0 {
		actions = append(actions, cloudOverviewAction("failedSyncTasks", "排查同步失败", "近 24 小时存在失败的云采集任务。", "error", metrics.FailedSyncTasks, assetTarget))
	}
	if len(actions) == 0 {
		actions = append(actions, cloudOverviewAction("healthy", "只读治理底座健康", "当前没有需要立即处理的账号、资产或同步异常。", "success", 0, assetTarget))
	}
	return actions
}

func cloudOverviewAction(key, title, description, level string, count int64, target string) resps.CloudOverviewActionResp {
	return resps.CloudOverviewActionResp{
		Key:         key,
		Title:       title,
		Description: description,
		Level:       level,
		Count:       count,
		Target:      target,
	}
}
