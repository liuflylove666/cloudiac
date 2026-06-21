// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package handlers

import (
	"cloudiac/portal/apps"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models/forms"
)

type CloudBudget struct {
}

func (CloudBudget) Search(c *ctx.GinRequest) {
	form := forms.SearchCloudBudgetForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudBudgets(c.Service(), &form))
}

func (CloudBudget) Summary(c *ctx.GinRequest) {
	form := forms.CloudBudgetSummaryForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CloudBudgetSummary(c.Service(), &form))
}

func (CloudBudget) EvaluateDue(c *ctx.GinRequest) {
	form := forms.EvaluateCloudBudgetForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.EvaluateDueCloudBudgets(c.Service(), &form))
}

func (CloudBudget) Create(c *ctx.GinRequest) {
	form := forms.CreateCloudBudgetForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CreateCloudBudget(c.Service(), &form))
}

func (CloudBudget) Update(c *ctx.GinRequest) {
	form := forms.UpdateCloudBudgetForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCloudBudget(c.Service(), &form))
}

func (CloudBudget) Delete(c *ctx.GinRequest) {
	form := forms.CloudBudgetParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.DeleteCloudBudget(c.Service(), &form))
}
