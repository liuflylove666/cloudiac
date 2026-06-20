// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package handlers

import (
	"cloudiac/portal/apps"
	"cloudiac/portal/libs/ctrl"
	"cloudiac/portal/libs/ctx"
)

type CloudOverview struct {
	ctrl.GinController
}

// Overview 查询多云管理总览
func (CloudOverview) Overview(c *ctx.GinRequest) {
	c.JSONResult(apps.CloudOverview(c.Service()))
}
