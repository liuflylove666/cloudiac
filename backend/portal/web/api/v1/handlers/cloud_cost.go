// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package handlers

import (
	"cloudiac/portal/apps"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models/forms"
)

type CloudCost struct {
}

func (CloudCost) Summary(c *ctx.GinRequest) {
	form := forms.CloudCostSummaryForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CloudCostSummary(c.Service(), &form))
}

func (CloudCost) Trends(c *ctx.GinRequest) {
	form := forms.CloudCostTrendForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CloudCostTrends(c.Service(), &form))
}

func (CloudCost) Records(c *ctx.GinRequest) {
	form := forms.SearchCloudCostRecordForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudCostRecords(c.Service(), &form))
}

func (CloudCost) Import(c *ctx.GinRequest) {
	form := forms.ImportCloudCostRecordForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.ImportCloudCostRecords(c.Service(), &form))
}

func (CloudCost) Pull(c *ctx.GinRequest) {
	form := forms.PullCloudCostRecordForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.PullCloudCostRecords(c.Service(), &form))
}

func (CloudCost) SyncTasks(c *ctx.GinRequest) {
	form := forms.SearchCloudCostSyncTaskForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudCostSyncTasks(c.Service(), &form))
}

func (CloudCost) CreateSyncTask(c *ctx.GinRequest) {
	form := forms.CreateCloudCostSyncTaskForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CreateCloudCostSyncTask(c.Service(), &form))
}

func (CloudCost) SyncTaskDetail(c *ctx.GinRequest) {
	form := forms.CloudCostSyncTaskParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CloudCostSyncTaskDetail(c.Service(), &form))
}

func (CloudCost) RetrySyncTask(c *ctx.GinRequest) {
	form := forms.CloudCostSyncTaskParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.RetryCloudCostSyncTask(c.Service(), &form))
}

func (CloudCost) SyncSchedules(c *ctx.GinRequest) {
	form := forms.SearchCloudCostSyncScheduleForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudCostSyncSchedules(c.Service(), &form))
}

func (CloudCost) CreateSyncSchedule(c *ctx.GinRequest) {
	form := forms.CreateCloudCostSyncScheduleForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CreateCloudCostSyncSchedule(c.Service(), &form))
}

func (CloudCost) UpdateSyncSchedule(c *ctx.GinRequest) {
	form := forms.UpdateCloudCostSyncScheduleForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCloudCostSyncSchedule(c.Service(), &form))
}

func (CloudCost) DeleteSyncSchedule(c *ctx.GinRequest) {
	form := forms.CloudCostSyncScheduleParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.DeleteCloudCostSyncSchedule(c.Service(), &form))
}

func (CloudCost) RunSyncSchedule(c *ctx.GinRequest) {
	form := forms.CloudCostSyncScheduleParam{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.RunCloudCostSyncSchedule(c.Service(), &form))
}

func (CloudCost) RunDueSyncSchedules(c *ctx.GinRequest) {
	form := forms.RunDueCloudCostSyncScheduleForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.RunDueCloudCostSyncSchedules(c.Service(), &form))
}

func (CloudCost) Unmatched(c *ctx.GinRequest) {
	form := forms.SearchCloudCostRecordForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchUnmatchedCloudCostRecords(c.Service(), &form))
}

func (CloudCost) InsightSummary(c *ctx.GinRequest) {
	form := forms.CloudCostInsightSummaryForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.CloudCostInsightSummary(c.Service(), &form))
}

func (CloudCost) Insights(c *ctx.GinRequest) {
	form := forms.SearchCloudCostInsightForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.SearchCloudCostInsights(c.Service(), &form))
}

func (CloudCost) UpdateInsightStatus(c *ctx.GinRequest) {
	form := forms.UpdateCloudCostInsightStatusForm{}
	if err := c.Bind(&form); err != nil {
		return
	}
	c.JSONResult(apps.UpdateCloudCostInsightStatus(c.Service(), &form))
}
