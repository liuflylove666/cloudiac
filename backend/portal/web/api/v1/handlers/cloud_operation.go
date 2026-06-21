// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package handlers

import (
	"cloudiac/portal/apps"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models/forms"
)

type CloudOperation struct {
}

// Search 查询云操作任务列表。
func (CloudOperation) Search(c *ctx.GinRequest) {
	form := forms.SearchCloudOperationForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudOperations(c.Service(), &form))
}

// Detail 查询云操作任务详情。
func (CloudOperation) Detail(c *ctx.GinRequest) {
	form := forms.CloudOperationParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CloudOperationDetail(c.Service(), &form))
}

// Cancel 取消尚未完成的云操作任务。
func (CloudOperation) Cancel(c *ctx.GinRequest) {
	form := forms.CloudOperationParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CancelCloudOperation(c.Service(), &form))
}

// Retry 重试失败的云资产动作任务。
func (CloudOperation) Retry(c *ctx.GinRequest) {
	form := forms.CloudOperationParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.RetryCloudOperation(c.Service(), &form))
}

// Approve 审批云操作任务。
func (CloudOperation) Approve(c *ctx.GinRequest) {
	form := forms.CloudOperationApprovalForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.ApproveCloudOperation(c.Service(), &form))
}

// Audits 查询云操作任务审计记录。
func (CloudOperation) Audits(c *ctx.GinRequest) {
	form := forms.CloudOperationParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CloudOperationAudits(c.Service(), &form))
}
