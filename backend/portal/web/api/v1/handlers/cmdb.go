// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package handlers

import (
	"cloudiac/portal/apps"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models/forms"
)

type Cmdb struct {
}

// SearchAssets 搜索组织 CMDB 资产
func (Cmdb) SearchAssets(c *ctx.GinRequest) {
	form := forms.SearchCmdbAssetForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCmdbAssets(c.Service(), &form))
}

// AssetDetail 查询 CMDB 资产详情
func (Cmdb) AssetDetail(c *ctx.GinRequest) {
	form := forms.CmdbAssetParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CmdbAssetDetail(c.Service(), &form))
}

// ExportAssets 导出组织 CMDB 资产
func (Cmdb) ExportAssets(c *ctx.GinRequest) {
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

// ImportAssets 导入组织 CMDB 资产
func (Cmdb) ImportAssets(c *ctx.GinRequest) {
	form := forms.ImportCmdbAssetForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.ImportCmdbAssets(c.Service(), &form))
}

// SearchApplications 查询 CMDB 应用依赖视图
func (Cmdb) SearchApplications(c *ctx.GinRequest) {
	form := forms.SearchCmdbApplicationForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCmdbApplications(c.Service(), &form))
}

// ApplicationDetail 查询 CMDB 应用依赖详情
func (Cmdb) ApplicationDetail(c *ctx.GinRequest) {
	form := forms.CmdbApplicationDetailForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CmdbApplicationDetail(c.Service(), &form))
}

// UpdateApplicationRelations 更新 CMDB 应用上下游依赖
func (Cmdb) UpdateApplicationRelations(c *ctx.GinRequest) {
	form := forms.UpdateCmdbApplicationRelationsForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCmdbApplicationRelations(c.Service(), &form))
}

// UpdateAssetOwnership 更新 CMDB 资产归属信息
func (Cmdb) UpdateAssetOwnership(c *ctx.GinRequest) {
	form := forms.UpdateCmdbAssetOwnershipForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCmdbAssetOwnership(c.Service(), &form))
}

// BatchUpdateAssetOwnership 批量更新 CMDB 资产归属信息
func (Cmdb) BatchUpdateAssetOwnership(c *ctx.GinRequest) {
	form := forms.BatchUpdateCmdbAssetOwnershipForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.BatchUpdateCmdbAssetOwnership(c.Service(), &form))
}

// AssetFilters 查询 CMDB 资产筛选项
func (Cmdb) AssetFilters(c *ctx.GinRequest) {
	form := forms.SearchCmdbAssetForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CmdbAssetFilters(c.Service(), &form))
}

// BackfillIacResources 从 CloudIaC 现有 IaC 资源回填 CMDB
func (Cmdb) BackfillIacResources(c *ctx.GinRequest) {
	c.JSONResult(apps.BackfillCmdbAssetsFromIac(c.Service()))
}

// CloudAccounts 查询可用于 CMDB 云采集的账号
func (Cmdb) CloudAccounts(c *ctx.GinRequest) {
	c.JSONResult(apps.SearchCmdbCloudAccounts(c.Service()))
}

// SearchSyncTasks 查询 CMDB 云采集任务
func (Cmdb) SearchSyncTasks(c *ctx.GinRequest) {
	form := forms.SearchCmdbSyncTaskForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCmdbSyncTasks(c.Service(), &form))
}

// SyncTaskDetail 查询 CMDB 云采集任务详情和日志
func (Cmdb) SyncTaskDetail(c *ctx.GinRequest) {
	form := forms.CmdbSyncTaskParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CmdbSyncTaskDetail(c.Service(), &form))
}

// SyncTaskRerunGroupDetail 查询 CMDB 云采集任务批量重跑任务组。
func (Cmdb) SyncTaskRerunGroupDetail(c *ctx.GinRequest) {
	form := forms.CmdbSyncTaskRerunGroupParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CmdbSyncTaskRerunGroupDetail(c.Service(), &form))
}

// StartSyncTask 启动一次 CMDB 云采集任务
func (Cmdb) StartSyncTask(c *ctx.GinRequest) {
	form := forms.CreateCmdbSyncTaskForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.StartCmdbSyncTask(c.Service(), &form))
}

// BatchRerunFailedSyncTasks 批量重跑失败的 CMDB 云采集任务
func (Cmdb) BatchRerunFailedSyncTasks(c *ctx.GinRequest) {
	form := forms.BatchRerunFailedCmdbSyncTasksForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.BatchRerunFailedCmdbSyncTasks(c.Service(), &form))
}

// ApproveSyncTaskRerunGroup 审批 CMDB 云采集批量重跑任务组。
func (Cmdb) ApproveSyncTaskRerunGroup(c *ctx.GinRequest) {
	form := forms.CmdbSyncTaskRerunGroupApprovalForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.ApproveCmdbSyncTaskRerunGroup(c.Service(), &form))
}
