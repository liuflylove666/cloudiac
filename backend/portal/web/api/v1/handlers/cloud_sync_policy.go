// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package handlers

import (
	"cloudiac/portal/apps"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models/forms"
)

type CloudSyncPolicy struct {
}

func (CloudSyncPolicy) Search(c *ctx.GinRequest) {
	form := forms.SearchCloudSyncPolicyForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudSyncPolicies(c.Service(), &form))
}

func (CloudSyncPolicy) Detail(c *ctx.GinRequest) {
	form := forms.CloudSyncPolicyParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CloudSyncPolicyDetail(c.Service(), &form))
}

func (CloudSyncPolicy) Create(c *ctx.GinRequest) {
	form := forms.CreateCloudSyncPolicyForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CreateCloudSyncPolicy(c.Service(), &form))
}

func (CloudSyncPolicy) Update(c *ctx.GinRequest) {
	form := forms.UpdateCloudSyncPolicyForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCloudSyncPolicy(c.Service(), &form))
}

func (CloudSyncPolicy) Delete(c *ctx.GinRequest) {
	form := forms.CloudSyncPolicyParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.DeleteCloudSyncPolicy(c.Service(), &form))
}

func (CloudSyncPolicy) Run(c *ctx.GinRequest) {
	form := forms.CloudSyncPolicyParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.RunCloudSyncPolicy(c.Service(), &form))
}

func (CloudSyncPolicy) RunDue(c *ctx.GinRequest) {
	form := forms.RunDueCloudSyncPolicyForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.RunDueCloudSyncPolicies(c.Service(), &form))
}
