// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package handlers

import (
	"cloudiac/portal/apps"
	"cloudiac/portal/libs/ctrl"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models/forms"
)

type Notification struct {
	ctrl.GinController
}

// Search 查询通知
// @Summary 查询通知
// @Description 查询通知
// @Tags 通知
// @Accept  json
// @Produce  json
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织ID"
// @Param form query forms.SearchNotificationForm true "parameter"
// @Success 200 {object} ctx.JSONResult{result=page.PageResp{list=[]resps.RespNotification}}
// @Router /notifications [get]
func (Notification) Search(c *ctx.GinRequest) {
	form := &forms.SearchNotificationForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.SearchNotification(c.Service(), form))
}

// Create 创建通知
// @Tags 通知
// @Summary 创建通知
// @Description 创建通知
// @Accept multipart/form-data
// @Accept json
// @Security AuthToken
// @Produce json
// @Param IaC-Org-Id header string true "组织ID"
// @Param json body forms.CreateNotificationForm true "parameter"
// @Router /notifications [post]
// @Success 200 {object} ctx.JSONResult{result=models.Notification}
func (Notification) Create(c *ctx.GinRequest) {
	form := &forms.CreateNotificationForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.CreateNotification(c.Service(), form))
}

// Delete 删除通知信息
// @Summary 删除通知信息
// @Description 删除Token账号
// @Tags 通知
// @Accept  json
// @Produce  json
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织ID"
// @Param id path string true "通知id"
// @Success 200
// @Router /notifications/{id} [delete]
func (Notification) Delete(c *ctx.GinRequest) {
	form := &forms.DeleteNotificationForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.DeleteNotification(c.Service(), form.Id))
}

// Update 修改通知信息
// @Summary 修改通知信息
// @Description 修改通知信息
// @Tags 通知
// @Accept  json
// @Produce  json
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织ID"
// @Param id path string true "通知id"
// @Param data body forms.UpdateNotificationForm true "ApiToken信息"
// @Success 200 {object} ctx.JSONResult{result=models.Notification}
// @Router /notifications/{id} [put]
func (Notification) Update(c *ctx.GinRequest) {
	form := &forms.UpdateNotificationForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateNotification(c.Service(), form))
}

// Detail 查询通知详情
// @Summary 查询通知详情
// @Description 查询通知详情
// @Tags 通知
// @Accept  json
// @Produce  json
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织id"
// @Param notificationId path string true "通知id"
// @Success 200 {object} ctx.JSONResult{result=resps.RespDetailNotification}
// @Router /notifications/{notificationId}  [get]
func (Notification) Detail(c *ctx.GinRequest) {
	form := &forms.DetailNotificationForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.DetailNotification(c.Service(), form))
}

// Deliveries 查询通知投递历史
// @Summary 查询通知投递历史
// @Description 查询通知投递历史
// @Tags 通知
// @Accept  json
// @Produce  json
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织id"
// @Param id path string true "通知id"
// @Param form query forms.SearchNotificationDeliveryForm true "parameter"
// @Success 200 {object} ctx.JSONResult{result=page.PageResp{list=[]resps.NotificationDeliveryResp}}
// @Router /notifications/{id}/deliveries  [get]
func (Notification) Deliveries(c *ctx.GinRequest) {
	form := &forms.SearchNotificationDeliveryForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.SearchNotificationDeliveries(c.Service(), form))
}

// SearchTemplates 查询通知模板
// @Summary 查询通知模板
// @Tags 通知
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织id"
// @Param form query forms.SearchNotificationTemplateForm true "parameter"
// @Success 200 {object} ctx.JSONResult{result=page.PageResp{list=[]resps.NotificationTemplateResp}}
// @Router /notification-templates [get]
func (Notification) SearchTemplates(c *ctx.GinRequest) {
	form := &forms.SearchNotificationTemplateForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.SearchNotificationTemplates(c.Service(), form))
}

// CreateTemplate 创建通知模板
// @Summary 创建通知模板
// @Tags 通知
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织id"
// @Param json body forms.CreateNotificationTemplateForm true "parameter"
// @Success 200 {object} ctx.JSONResult{result=models.NotificationTemplate}
// @Router /notification-templates [post]
func (Notification) CreateTemplate(c *ctx.GinRequest) {
	form := &forms.CreateNotificationTemplateForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.CreateNotificationTemplate(c.Service(), form))
}

// CopyTemplatesToProject 批量复制组织通知模板到项目
// @Summary 批量复制组织通知模板到项目
// @Tags 通知
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织id"
// @Param IaC-Project-Id header string true "项目id"
// @Param json body forms.CopyNotificationTemplatesForm true "parameter"
// @Success 200 {object} ctx.JSONResult{result=resps.NotificationTemplateCopyResp}
// @Router /notification-templates/copy-to-project [post]
func (Notification) CopyTemplatesToProject(c *ctx.GinRequest) {
	form := &forms.CopyNotificationTemplatesForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.CopyNotificationTemplatesToProject(c.Service(), form))
}

// DetailTemplate 查询通知模板详情
// @Summary 查询通知模板详情
// @Tags 通知
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织id"
// @Param id path string true "通知模板id"
// @Success 200 {object} ctx.JSONResult{result=resps.NotificationTemplateResp}
// @Router /notification-templates/{id} [get]
func (Notification) DetailTemplate(c *ctx.GinRequest) {
	form := &forms.DetailNotificationTemplateForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.DetailNotificationTemplate(c.Service(), form))
}

// UpdateTemplate 更新通知模板
// @Summary 更新通知模板
// @Tags 通知
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织id"
// @Param id path string true "通知模板id"
// @Param json body forms.UpdateNotificationTemplateForm true "parameter"
// @Success 200 {object} ctx.JSONResult{result=models.NotificationTemplate}
// @Router /notification-templates/{id} [put]
func (Notification) UpdateTemplate(c *ctx.GinRequest) {
	form := &forms.UpdateNotificationTemplateForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateNotificationTemplate(c.Service(), form))
}

// DeleteTemplate 删除通知模板
// @Summary 删除通知模板
// @Tags 通知
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织id"
// @Param id path string true "通知模板id"
// @Success 200
// @Router /notification-templates/{id} [delete]
func (Notification) DeleteTemplate(c *ctx.GinRequest) {
	form := &forms.DeleteNotificationTemplateForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.DeleteNotificationTemplate(c.Service(), form))
}

// TemplateVersions 查询通知模板版本
// @Summary 查询通知模板版本
// @Tags 通知
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织id"
// @Param id path string true "通知模板id"
// @Param form query forms.SearchNotificationTemplateVersionsForm true "parameter"
// @Success 200 {object} ctx.JSONResult{result=page.PageResp{list=[]resps.NotificationTemplateVersionResp}}
// @Router /notification-templates/{id}/versions [get]
func (Notification) TemplateVersions(c *ctx.GinRequest) {
	form := &forms.SearchNotificationTemplateVersionsForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.SearchNotificationTemplateVersions(c.Service(), form))
}

// RollbackTemplateVersion 回滚通知模板版本
// @Summary 回滚通知模板版本
// @Tags 通知
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织id"
// @Param id path string true "通知模板id"
// @Param versionId path string true "通知模板版本id"
// @Success 200 {object} ctx.JSONResult{result=models.NotificationTemplate}
// @Router /notification-templates/{id}/versions/{versionId}/rollback [post]
func (Notification) RollbackTemplateVersion(c *ctx.GinRequest) {
	form := &forms.RollbackNotificationTemplateVersionForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.RollbackNotificationTemplateVersion(c.Service(), form))
}

// TemplateVariables 查询通知模板变量字典
// @Summary 查询通知模板变量字典
// @Tags 通知
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织id"
// @Param form query forms.NotificationTemplateVariablesForm true "parameter"
// @Success 200 {object} ctx.JSONResult{result=resps.NotificationTemplateVariablesResp}
// @Router /notification-templates/variables [get]
func (Notification) TemplateVariables(c *ctx.GinRequest) {
	form := &forms.NotificationTemplateVariablesForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.NotificationTemplateVariables(c.Service(), form))
}

// PreviewTemplate 预览通知模板
// @Summary 预览通知模板
// @Tags 通知
// @Security AuthToken
// @Param IaC-Org-Id header string true "组织id"
// @Param json body forms.PreviewNotificationTemplateForm true "parameter"
// @Success 200 {object} ctx.JSONResult{result=resps.NotificationTemplatePreviewResp}
// @Router /notification-templates/preview [post]
func (Notification) PreviewTemplate(c *ctx.GinRequest) {
	form := &forms.PreviewNotificationTemplateForm{}
	if err := c.Bind(form); err != nil {
		return
	}
	c.JSONResult(apps.PreviewNotificationTemplate(c.Service(), form))
}
