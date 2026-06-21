// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package handlers

import (
	"cloudiac/portal/apps"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models/forms"
)

type CloudWebhook struct {
}

func (CloudWebhook) Search(c *ctx.GinRequest) {
	form := forms.SearchCloudWebhookForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudWebhooks(c.Service(), &form))
}

func (CloudWebhook) Create(c *ctx.GinRequest) {
	form := forms.CreateCloudWebhookForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CreateCloudWebhook(c.Service(), &form))
}

func (CloudWebhook) Update(c *ctx.GinRequest) {
	form := forms.UpdateCloudWebhookForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCloudWebhook(c.Service(), &form))
}

func (CloudWebhook) Delete(c *ctx.GinRequest) {
	form := forms.CloudWebhookParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.DeleteCloudWebhook(c.Service(), &form))
}

func (CloudWebhook) Deliveries(c *ctx.GinRequest) {
	form := forms.SearchCloudWebhookDeliveryForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudWebhookDeliveries(c.Service(), &form))
}

func (CloudWebhook) DeliveryDetail(c *ctx.GinRequest) {
	form := forms.CloudWebhookDeliveryParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.GetCloudWebhookDelivery(c.Service(), &form))
}

func (CloudWebhook) RetryDelivery(c *ctx.GinRequest) {
	form := forms.RetryCloudWebhookDeliveryForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.RetryCloudWebhookDelivery(c.Service(), &form))
}

func (CloudWebhook) ReportSignatureVerification(c *ctx.GinRequest) {
	form := forms.ReportCloudWebhookSignatureVerificationForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.ReportCloudWebhookSignatureVerification(c.Service(), &form))
}

func (CloudWebhook) ReportSignatureVerificationByToken(c *ctx.GinRequest) {
	form := forms.ReportCloudWebhookSignatureVerificationByTokenForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	if form.Token == "" {
		form.Token = c.GetHeader(apps.CloudWebhookSignatureReportTokenHeader)
	}
	c.JSONResult(apps.ReportCloudWebhookSignatureVerificationByToken(c.Service(), &form))
}

func (CloudWebhook) DeadLetters(c *ctx.GinRequest) {
	form := forms.SearchCloudWebhookDeadLetterForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudWebhookDeadLetters(c.Service(), &form))
}

func (CloudWebhook) ReplayDeadLetter(c *ctx.GinRequest) {
	form := forms.ReplayCloudWebhookDeadLetterForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.ReplayCloudWebhookDeadLetter(c.Service(), &form))
}

func (CloudWebhook) IgnoreDeadLetter(c *ctx.GinRequest) {
	form := forms.IgnoreCloudWebhookDeadLetterForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.IgnoreCloudWebhookDeadLetter(c.Service(), &form))
}

func (CloudWebhook) Test(c *ctx.GinRequest) {
	form := forms.TestCloudWebhookForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.TestCloudWebhook(c.Service(), &form))
}

func (CloudWebhook) RotateSecret(c *ctx.GinRequest) {
	form := forms.RotateCloudWebhookSecretForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.RotateCloudWebhookSecret(c.Service(), &form))
}

func (CloudWebhook) QueueSummary(c *ctx.GinRequest) {
	c.JSONResult(apps.CloudWebhookQueueSummary(c.Service()))
}

func (CloudWebhook) RetryDue(c *ctx.GinRequest) {
	form := forms.RetryDueCloudWebhookDeliveriesForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.RetryDueCloudWebhookDeliveriesWithForm(c.Service(), &form))
}
