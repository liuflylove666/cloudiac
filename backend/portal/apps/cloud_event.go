// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"fmt"
	"strings"
	"time"

	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/libs/db"
	"cloudiac/portal/libs/page"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	"cloudiac/portal/models/resps"
)

func SearchCloudEvents(c *ctx.ServiceContext, form *forms.SearchCloudEventForm) (interface{}, e.Error) {
	query := c.DB().Model(&models.CloudEvent{}).Where("org_id = ?", c.OrgId)
	query = applyCloudEventSearch(query, form)
	if form.SortField() == "" {
		query = query.Order("occurred_at desc").Order("created_at desc")
	} else {
		query = form.Order(query)
	}

	events := make([]models.CloudEvent, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&events); err != nil {
		return nil, e.New(e.DBError, err)
	}

	list := make([]resps.CloudEventResp, 0, len(events))
	for _, event := range events {
		list = append(list, cloudEventResp(c, event))
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func applyCloudEventSearch(query *db.Session, form *forms.SearchCloudEventForm) *db.Session {
	if form.Q != "" {
		q := "%" + strings.TrimSpace(form.Q) + "%"
		query = query.Where(`id like ? or event_type like ? or title like ? or message like ? or
			resource_name like ? or resource_id like ? or provider like ? or account_id like ? or payload like ?`,
			q, q, q, q, q, q, q, q, q)
	}
	if form.Source != "" {
		query = query.Where("source = ?", form.Source)
	}
	if form.EventType != "" {
		query = query.Where("event_type = ?", form.EventType)
	}
	if form.Level != "" {
		query = query.Where("level = ?", form.Level)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	if form.Provider != "" {
		query = query.Where("provider = ?", form.Provider)
	}
	if form.AccountId != "" {
		query = query.Where("account_id = ?", form.AccountId)
	}
	if form.Region != "" {
		query = query.Where("region = ?", form.Region)
	}
	if form.AssetId != "" {
		query = query.Where("asset_id = ?", form.AssetId)
	}
	if form.OperationId != "" {
		query = query.Where("operation_id = ?", form.OperationId)
	}
	if form.RiskFindingId != "" {
		query = query.Where("risk_finding_id = ?", form.RiskFindingId)
	}
	if form.ProjectId != "" {
		query = query.Where("project_id = ?", form.ProjectId)
	}
	if form.EnvId != "" {
		query = query.Where("env_id = ?", form.EnvId)
	}
	return query
}

func recordCloudEvent(c *ctx.ServiceContext, event models.CloudEvent) e.Error {
	return recordCloudEventWithDispatch(c, event, true)
}

func recordCloudEventWithoutDispatch(c *ctx.ServiceContext, event models.CloudEvent) e.Error {
	return recordCloudEventWithDispatch(c, event, false)
}

func recordCloudEventWithDispatch(c *ctx.ServiceContext, event models.CloudEvent, dispatch bool) e.Error {
	if c == nil {
		return nil
	}
	if event.OrgId == "" {
		event.OrgId = c.OrgId
	}
	if event.OrgId == "" || event.EventType == "" {
		return nil
	}
	if event.Id == "" {
		event.Id = models.NewId("cev")
	}
	if event.Level == "" {
		event.Level = models.CloudEventLevelInfo
	}
	if event.ActorId == "" {
		event.ActorId = c.UserId
	}
	if event.UserIp == "" {
		event.UserIp = c.UserIpAddr
	}
	if time.Time(event.OccurredAt).IsZero() {
		event.OccurredAt = models.Time(time.Now())
	}
	if event.Title == "" {
		event.Title = event.EventType
	}
	if err := models.Create(c.DB(), &event); err != nil {
		return e.New(e.DBError, err)
	}
	if dispatch {
		dispatchCloudEventWebhooksBestEffort(c, &event)
		dispatchCloudEventNotificationsBestEffort(c, &event)
		dispatchCloudEventItsmBestEffort(c, &event)
	}
	return nil
}

func recordCloudEventBestEffort(c *ctx.ServiceContext, event models.CloudEvent) {
	if err := recordCloudEvent(c, event); err != nil && c != nil {
		c.Logger().Warnf("record cloud event failed: %v", err)
	}
}

func cloudEventResp(c *ctx.ServiceContext, event models.CloudEvent) resps.CloudEventResp {
	assetName := event.ResourceName
	if assetName == "" && event.AssetId != "" {
		assetName = lookupName(c, &models.CmdbAsset{}, event.AssetId)
	}
	return resps.CloudEventResp{
		CloudEvent:  event,
		ProjectName: lookupName(c, &models.Project{}, event.ProjectId),
		EnvName:     lookupName(c, &models.Env{}, event.EnvId),
		AssetName:   assetName,
		ActorName:   lookupName(c, &models.User{}, event.ActorId),
	}
}

func cloudOperationEvent(c *ctx.ServiceContext, operation *models.CloudOperation, eventType string, title string, message string, payload models.ResAttrs) {
	if operation == nil {
		return
	}
	level := models.CloudEventLevelInfo
	if operation.Status == models.CloudOperationStatusFailed || operation.Status == models.CloudOperationStatusRejected {
		level = models.CloudEventLevelError
	} else if operation.Status == models.CloudOperationStatusAborted || operation.RiskLevel == models.CloudOperationRiskHigh || operation.RiskLevel == models.CloudOperationRiskCritical {
		level = models.CloudEventLevelWarning
	}
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:          operation.OrgId,
		ProjectId:      operation.ProjectId,
		EnvId:          operation.EnvId,
		AssetId:        operation.AssetId,
		OperationId:    operation.Id,
		CloudAccountId: operation.CloudAccountId,
		Source:         models.CloudEventSourceOperation,
		EventType:      eventType,
		Level:          level,
		Status:         operation.Status,
		Provider:       operation.Provider,
		ResourceType:   operation.ResourceType,
		ResourceId:     operation.ResourceId,
		ResourceName:   operation.ResourceName,
		Title:          title,
		Message:        message,
		Payload:        payload,
	})
}

func cloudRiskEvent(c *ctx.ServiceContext, finding *models.CloudRiskFinding, eventType string, title string, message string, payload models.ResAttrs) {
	if finding == nil {
		return
	}
	level := models.CloudEventLevelInfo
	if finding.RiskLevel == models.CloudOperationRiskCritical || finding.RiskLevel == models.CloudOperationRiskHigh {
		level = models.CloudEventLevelWarning
	}
	if finding.Status == models.CloudRiskStatusResolved {
		level = models.CloudEventLevelInfo
	}
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:          finding.OrgId,
		ProjectId:      finding.ProjectId,
		EnvId:          finding.EnvId,
		AssetId:        finding.AssetId,
		RiskFindingId:  finding.Id,
		CloudAccountId: finding.CloudAccountId,
		Source:         models.CloudEventSourceRisk,
		EventType:      eventType,
		Level:          level,
		Status:         finding.Status,
		Provider:       finding.Provider,
		AccountId:      finding.AccountId,
		Region:         finding.Region,
		ResourceType:   finding.ResourceType,
		ResourceId:     finding.ResourceId,
		ResourceName:   finding.ResourceName,
		Title:          title,
		Message:        message,
		Payload:        payload,
	})
}

func cmdbAssetChangeEvent(c *ctx.ServiceContext, change *models.CmdbAssetChange) {
	if change == nil {
		return
	}
	eventType := "cmdb.asset.updated"
	title := "CMDB 资产变更"
	if change.ChangeType == models.CmdbAssetChangeTypeCreated {
		eventType = "cmdb.asset.created"
		title = "CMDB 资产新增"
	}
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:     change.OrgId,
		AssetId:   change.AssetId,
		Source:    models.CloudEventSourceCMDB,
		EventType: eventType,
		Level:     models.CloudEventLevelInfo,
		Status:    change.ChangeType,
		Title:     title,
		Message:   fmt.Sprintf("%s：%s", change.Source, change.Summary),
		Payload: models.ResAttrs{
			"changeId":   change.Id.String(),
			"changeType": change.ChangeType,
			"source":     change.Source,
			"diff":       change.Diff,
		},
	})
}

func cmdbApplicationRelationsEvent(
	c *ctx.ServiceContext,
	application string,
	beforeUpstreams []string,
	afterUpstreams []string,
	beforeDownstreams []string,
	afterDownstreams []string,
) {
	if c == nil {
		return
	}
	application = strings.TrimSpace(application)
	if application == "" {
		return
	}
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:        c.OrgId,
		Source:       models.CloudEventSourceCMDB,
		EventType:    "cmdb.application.relations_updated",
		Level:        models.CloudEventLevelInfo,
		Status:       models.CmdbAssetChangeTypeUpdated,
		ResourceType: "cmdb_application",
		ResourceName: application,
		Title:        "CMDB 应用依赖变更",
		Message: fmt.Sprintf("应用 %s 依赖关系已更新：上游 %d -> %d，下游 %d -> %d",
			application, len(beforeUpstreams), len(afterUpstreams), len(beforeDownstreams), len(afterDownstreams)),
		Payload: models.ResAttrs{
			"application":  application,
			"source":       models.CmdbRelationSourceManualApp,
			"relationType": models.CmdbRelationTypeDependsOn,
			"before": models.ResAttrs{
				"upstreams":   beforeUpstreams,
				"downstreams": beforeDownstreams,
			},
			"after": models.ResAttrs{
				"upstreams":   afterUpstreams,
				"downstreams": afterDownstreams,
			},
		},
	})
}
