// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package handlers

import (
	"cloudiac/portal/apps"
	"cloudiac/portal/libs/ctrl"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models/forms"
)

type CloudAccount struct {
	ctrl.GinController
}

// Search 搜索云账号
func (CloudAccount) Search(c *ctx.GinRequest) {
	form := &forms.SearchCloudAccountForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudAccounts(c.Service(), form))
}

// Detail 查询云账号详情
func (CloudAccount) Detail(c *ctx.GinRequest) {
	form := &forms.CloudAccountParam{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.CloudAccountDetail(c.Service(), form))
}

// Create 创建云账号
func (CloudAccount) Create(c *ctx.GinRequest) {
	form := &forms.CreateCloudAccountForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.CreateCloudAccount(c.Service(), form))
}

// Update 更新云账号
func (CloudAccount) Update(c *ctx.GinRequest) {
	form := &forms.UpdateCloudAccountForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCloudAccount(c.Service(), form))
}

// Delete 删除云账号
func (CloudAccount) Delete(c *ctx.GinRequest) {
	form := &forms.CloudAccountParam{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.DeleteCloudAccount(c.Service(), form))
}

// Validate 验证云账号本地凭证完整性
func (CloudAccount) Validate(c *ctx.GinRequest) {
	form := &forms.CloudAccountParam{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.ValidateCloudAccount(c.Service(), form))
}

// HealthCheck 检查单个云账号健康状态
func (CloudAccount) HealthCheck(c *ctx.GinRequest) {
	form := &forms.CloudAccountParam{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.CheckCloudAccountHealth(c.Service(), form))
}

// HealthCheckAll 批量检查云账号健康状态
func (CloudAccount) HealthCheckAll(c *ctx.GinRequest) {
	form := &forms.CheckCloudAccountsHealthForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.CheckCloudAccountsHealth(c.Service(), form))
}

// Regions 查询云账号区域配置
func (CloudAccount) Regions(c *ctx.GinRequest) {
	form := &forms.CloudAccountParam{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.CloudAccountRegions(c.Service(), form))
}

// UpdateRegions 更新云账号启用区域
func (CloudAccount) UpdateRegions(c *ctx.GinRequest) {
	form := &forms.UpdateCloudAccountRegionsForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCloudAccountRegions(c.Service(), form))
}

// Permissions 查询云账号权限验证结果
func (CloudAccount) Permissions(c *ctx.GinRequest) {
	form := &forms.CloudAccountParam{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.CloudAccountPermissions(c.Service(), form))
}
