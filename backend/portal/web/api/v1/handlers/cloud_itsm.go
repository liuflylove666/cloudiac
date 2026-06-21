// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package handlers

import (
	"cloudiac/portal/apps"
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

func (CloudItsm) UpdateTicketStatus(c *ctx.GinRequest) {
	form := forms.UpdateCloudItsmTicketStatusForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCloudItsmTicketStatus(c.Service(), &form))
}
