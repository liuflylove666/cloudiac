// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package handlers

import (
	"cloudiac/portal/apps"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models/forms"
)

type CloudAsset struct {
}

// Coverage 查询多云资产覆盖率、未纳管和归属统计。
func (CloudAsset) Coverage(c *ctx.GinRequest) {
	c.JSONResult(apps.CloudAssetCoverage(c.Service()))
}

// SearchAssets 搜索多云资产，当前复用 CMDB 资产中心查询。
func (CloudAsset) SearchAssets(c *ctx.GinRequest) {
	form := forms.SearchCmdbAssetForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCmdbAssets(c.Service(), &form))
}

// AssetFilters 查询多云资产筛选项，当前复用 CMDB 资产中心筛选。
func (CloudAsset) AssetFilters(c *ctx.GinRequest) {
	form := forms.SearchCmdbAssetForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CmdbAssetFilters(c.Service(), &form))
}

// AssetPermissions 查询多云资产编辑、导出和治理权限。
func (CloudAsset) AssetPermissions(c *ctx.GinRequest) {
	c.JSONResult(apps.CmdbAssetPermissions(c.Service()))
}

// AssetGovernanceReport 查询多云资产成本、合规和生命周期治理报表。
func (CloudAsset) AssetGovernanceReport(c *ctx.GinRequest) {
	c.JSONResult(apps.CmdbAssetGovernanceReport(c.Service()))
}

// AssetDetail 查询多云资产详情，当前复用 CMDB 资产详情。
func (CloudAsset) AssetDetail(c *ctx.GinRequest) {
	form := forms.CmdbAssetParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CmdbAssetDetail(c.Service(), &form))
}

// AssetRelations 查询多云资产关系图关系和摘要。
func (CloudAsset) AssetRelations(c *ctx.GinRequest) {
	form := forms.CmdbAssetRelationsForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CmdbAssetRelations(c.Service(), &form))
}

// SecurityRules 查询多云资产安全组/安全列表规则视图。
func (CloudAsset) SecurityRules(c *ctx.GinRequest) {
	form := forms.CmdbAssetParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CloudAssetSecurityRules(c.Service(), &form))
}

// AssetActions 查询多云资产可执行动作目录。
func (CloudAsset) AssetActions(c *ctx.GinRequest) {
	form := forms.CloudAssetActionParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CloudAssetActions(c.Service(), &form))
}

// DryRunAssetAction 对多云资产动作做预检查。
func (CloudAsset) DryRunAssetAction(c *ctx.GinRequest) {
	form := forms.DryRunCloudAssetActionForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.DryRunCloudAssetAction(c.Service(), &form))
}

// CreateAssetAction 创建多云资产操作任务。
func (CloudAsset) CreateAssetAction(c *ctx.GinRequest) {
	form := forms.CreateCloudAssetActionForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CreateCloudAssetAction(c.Service(), &form))
}

// ExportAssets 导出多云资产，当前复用 CMDB 资产导出。
func (CloudAsset) ExportAssets(c *ctx.GinRequest) {
	form := forms.ExportCmdbAssetForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	file, err := apps.ExportCmdbAssets(c.Service(), &form)
	if err != nil {
		c.JSONError(err)
		return
	}
	c.FileDownloadResponse(file.Data, file.Filename, file.ContentType)
}

// ImportAssets 导入多云资产，当前复用 CMDB 资产导入。
func (CloudAsset) ImportAssets(c *ctx.GinRequest) {
	form := forms.ImportCmdbAssetForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.ImportCmdbAssets(c.Service(), &form))
}

// ImportTemplate 下载多云资产导入模板，当前复用 CMDB 资产导入模板。
func (CloudAsset) ImportTemplate(c *ctx.GinRequest) {
	file, err := apps.CmdbAssetImportTemplate()
	if err != nil {
		c.JSONError(err)
		return
	}
	c.FileDownloadResponse(file.Data, file.Filename, file.ContentType)
}

// UpdateAssetOwnership 更新多云资产归属信息，当前复用 CMDB 资产归属。
func (CloudAsset) UpdateAssetOwnership(c *ctx.GinRequest) {
	form := forms.UpdateCmdbAssetOwnershipForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCmdbAssetOwnership(c.Service(), &form))
}

// BatchUpdateAssetOwnership 批量更新多云资产归属信息。
func (CloudAsset) BatchUpdateAssetOwnership(c *ctx.GinRequest) {
	form := forms.BatchUpdateCmdbAssetOwnershipForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.BatchUpdateCmdbAssetOwnership(c.Service(), &form))
}

// BackfillIacResources 从 CloudIaC 现有 IaC 资源回填多云资产中心。
func (CloudAsset) BackfillIacResources(c *ctx.GinRequest) {
	if err := apps.EnsureCmdbOrgAdminPermission(c.Service(), "同步 IaC 资源"); err != nil {
		c.JSONError(err)
		return
	}
	c.JSONResult(apps.BackfillCmdbAssetsFromIac(c.Service()))
}

// SearchSyncTasks 查询多云资产同步任务，当前复用 CMDB 云采集任务。
func (CloudAsset) SearchSyncTasks(c *ctx.GinRequest) {
	form := forms.SearchCmdbSyncTaskForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCmdbSyncTasks(c.Service(), &form))
}

// SyncTaskDetail 查询多云资产同步任务详情和日志。
func (CloudAsset) SyncTaskDetail(c *ctx.GinRequest) {
	form := forms.CmdbSyncTaskParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CmdbSyncTaskDetail(c.Service(), &form))
}

// SyncTaskRerunGroupDetail 查询多云资产同步任务批量重跑任务组。
func (CloudAsset) SyncTaskRerunGroupDetail(c *ctx.GinRequest) {
	form := forms.CmdbSyncTaskRerunGroupParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CmdbSyncTaskRerunGroupDetail(c.Service(), &form))
}

// StartSyncTask 启动一次多云资产采集任务。
func (CloudAsset) StartSyncTask(c *ctx.GinRequest) {
	form := forms.CreateCmdbSyncTaskForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	if err := apps.EnsureCmdbOrgAdminPermission(c.Service(), "启动多云资产采集任务"); err != nil {
		c.JSONError(err)
		return
	}
	c.JSONResult(apps.StartCmdbSyncTask(c.Service(), &form))
}

// BatchRerunFailedSyncTasks 批量重跑失败的多云资产采集任务。
func (CloudAsset) BatchRerunFailedSyncTasks(c *ctx.GinRequest) {
	form := forms.BatchRerunFailedCmdbSyncTasksForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.BatchRerunFailedCmdbSyncTasks(c.Service(), &form))
}

// ApproveSyncTaskRerunGroup 审批多云资产采集批量重跑任务组。
func (CloudAsset) ApproveSyncTaskRerunGroup(c *ctx.GinRequest) {
	form := forms.CmdbSyncTaskRerunGroupApprovalForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.ApproveCmdbSyncTaskRerunGroup(c.Service(), &form))
}
