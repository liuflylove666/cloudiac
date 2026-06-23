// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package handlers

import (
	"cloudiac/portal/apps"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models/forms"
)

type CloudRisk struct {
}

func (CloudRisk) Search(c *ctx.GinRequest) {
	form := forms.SearchCloudRiskForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudRisks(c.Service(), &form))
}

func (CloudRisk) UpdateStatus(c *ctx.GinRequest) {
	form := forms.UpdateCloudRiskStatusForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCloudRiskStatus(c.Service(), &form))
}

func (CloudRisk) Suppress(c *ctx.GinRequest) {
	form := forms.SuppressCloudRiskForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SuppressCloudRisk(c.Service(), &form))
}

func (CloudRisk) CreateRemediationTicket(c *ctx.GinRequest) {
	form := forms.CreateCloudRiskRemediationTicketForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CreateCloudRiskRemediationTicket(c.Service(), &form))
}

func (CloudRisk) AdoptRecommendation(c *ctx.GinRequest) {
	form := forms.AdoptCloudRiskRecommendationForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.AdoptCloudRiskRecommendation(c.Service(), &form))
}
