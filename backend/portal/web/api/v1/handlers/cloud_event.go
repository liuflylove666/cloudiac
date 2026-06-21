// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package handlers

import (
	"cloudiac/portal/apps"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models/forms"
)

type CloudEvent struct {
}

func (CloudEvent) Search(c *ctx.GinRequest) {
	form := forms.SearchCloudEventForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudEvents(c.Service(), &form))
}
