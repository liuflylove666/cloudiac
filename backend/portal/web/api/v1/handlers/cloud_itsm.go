// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package handlers

import (
	"bytes"
	"io"

	"cloudiac/portal/apps"
	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models/forms"
)

type CloudItsm struct {
}

func (CloudItsm) SearchConfigs(c *ctx.GinRequest) {
	form := forms.SearchCloudItsmConfigForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudItsmConfigs(c.Service(), &form))
}

func (CloudItsm) CreateConfig(c *ctx.GinRequest) {
	form := forms.CreateCloudItsmConfigForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CreateCloudItsmConfig(c.Service(), &form))
}

func (CloudItsm) UpdateConfig(c *ctx.GinRequest) {
	form := forms.UpdateCloudItsmConfigForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCloudItsmConfig(c.Service(), &form))
}

func (CloudItsm) DeleteConfig(c *ctx.GinRequest) {
	form := forms.CloudItsmConfigParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.DeleteCloudItsmConfig(c.Service(), &form))
}

func (CloudItsm) SearchTickets(c *ctx.GinRequest) {
	form := forms.SearchCloudItsmTicketForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudItsmTickets(c.Service(), &form))
}

func (CloudItsm) Overview(c *ctx.GinRequest) {
	c.JSONResult(apps.CloudItsmOverview(c.Service()))
}

func (CloudItsm) CreateOperationTicket(c *ctx.GinRequest) {
	form := forms.CreateCloudOperationItsmTicketForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CreateCloudOperationItsmTicket(c.Service(), &form))
}

func (CloudItsm) CreateSelfServiceTicket(c *ctx.GinRequest) {
	form := forms.CreateCloudItsmSelfServiceTicketForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CreateCloudItsmSelfServiceTicket(c.Service(), &form))
}

func (CloudItsm) UpdateCatalogPolicy(c *ctx.GinRequest) {
	form := forms.UpdateCloudItsmCatalogPolicyForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCloudItsmCatalogPolicy(c.Service(), &form))
}

func (CloudItsm) CatalogPolicyHistory(c *ctx.GinRequest) {
	form := forms.CloudItsmCatalogPolicyHistoryForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CloudItsmCatalogPolicyHistory(c.Service(), &form))
}

func (CloudItsm) UpdateTicketStatus(c *ctx.GinRequest) {
	form := forms.UpdateCloudItsmTicketStatusForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCloudItsmTicketStatus(c.Service(), &form))
}

func (CloudItsm) SyncDueTicketStatuses(c *ctx.GinRequest) {
	form := forms.SyncDueCloudItsmTicketStatusForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SyncDueCloudItsmTicketStatuses(c.Service(), &form))
}

func (CloudItsm) RetryDueTicketSubmissions(c *ctx.GinRequest) {
	form := forms.RetryDueCloudItsmTicketSubmitForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.RetryDueCloudItsmTicketSubmissions(c.Service(), &form))
}

func (CloudItsm) SubmitRetryQueueSummary(c *ctx.GinRequest) {
	c.JSONResult(apps.CloudItsmSubmitRetryQueueSummary(c.Service()))
}

func (CloudItsm) SubmitRetryQueueReport(c *ctx.GinRequest) {
	c.JSONResult(apps.CloudItsmSubmitRetryQueueReport(c.Service()))
}

func (CloudItsm) SearchSubmitRetryQueue(c *ctx.GinRequest) {
	form := forms.SearchCloudItsmSubmitRetryQueueForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudItsmSubmitRetryQueue(c.Service(), &form))
}

func (CloudItsm) ReplayTicketSubmission(c *ctx.GinRequest) {
	form := forms.ReplayCloudItsmTicketSubmitForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.ReplayCloudItsmTicketSubmission(c.Service(), &form))
}

func (CloudItsm) BatchDeadLetterAction(c *ctx.GinRequest) {
	form := forms.BatchCloudItsmDeadLetterActionForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.BatchCloudItsmDeadLetterAction(c.Service(), &form))
}

func (CloudItsm) CreateDeadLetterApproval(c *ctx.GinRequest) {
	form := forms.CreateCloudItsmDeadLetterApprovalForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CreateCloudItsmDeadLetterApproval(c.Service(), &form))
}

func (CloudItsm) StatusCallback(c *ctx.GinRequest) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSONError(e.New(e.IOError, err))
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))
	form := forms.CloudItsmTicketStatusCallbackForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	form.RawBody = rawBody
	if form.Signature == "" {
		form.Signature = c.GetHeader(apps.CloudItsmCallbackSignatureHeader)
	}
	if form.Signature == "" {
		form.Signature = c.GetHeader("X-CloudIaC-Signature")
	}
	c.JSONResult(apps.UpdateCloudItsmTicketStatusByCallback(c.Service(), &form))
}

func (CloudItsm) GitOpsGateCallback(c *ctx.GinRequest) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSONError(e.New(e.IOError, err))
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))
	form := forms.CloudItsmGitOpsGateCallbackForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	form.RawBody = rawBody
	if form.Signature == "" {
		form.Signature = c.GetHeader(apps.CloudItsmCallbackSignatureHeader)
	}
	if form.Signature == "" {
		form.Signature = c.GetHeader("X-CloudIaC-Signature")
	}
	c.JSONResult(apps.UpdateCloudItsmGitOpsGateByCallback(c.Service(), &form))
}
