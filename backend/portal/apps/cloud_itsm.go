// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

type cloudItsmSubmitResult struct {
	Status          string
	ExternalId      string
	ExternalKey     string
	ExternalUrl     string
	ResponsePayload models.ResAttrs
	ErrorMessage    string
	SubmittedAt     models.Time
}

const (
	defaultCloudItsmConnectorName        = "CloudIaC 本地工单"
	defaultCloudItsmConnectorDescription = "平台内置本地工单连接器，未配置外部 ITSM 时用于自助申请和审计流转"
)

func SearchCloudItsmConfigs(c *ctx.ServiceContext, form *forms.SearchCloudItsmConfigForm) (interface{}, e.Error) {
	if _, err := ensureDefaultCloudItsmConfig(c); err != nil {
		return nil, err
	}
	query := applyCloudItsmConfigSearch(cloudItsmConfigVisibleQuery(c), form)
	if form.SortField() == "" {
		query = query.Order("created_at desc")
	} else {
		query = form.Order(query)
	}
	configs := make([]models.CloudItsmConfig, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&configs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	list := make([]resps.CloudItsmConfigResp, 0, len(configs))
	for _, cfg := range configs {
		list = append(list, cloudItsmConfigResp(cfg))
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func CreateCloudItsmConfig(c *ctx.ServiceContext, form *forms.CreateCloudItsmConfigForm) (*resps.CloudItsmConfigResp, e.Error) {
	cfg, err := cloudItsmConfigFromForm(c, form)
	if err != nil {
		return nil, err
	}
	cfg.Id = models.NewId("citc")
	if err := models.Create(c.DB(), cfg); err != nil {
		return nil, e.New(e.DBError, err)
	}
	resp := cloudItsmConfigResp(*cfg)
	return &resp, nil
}

func UpdateCloudItsmConfig(c *ctx.ServiceContext, form *forms.UpdateCloudItsmConfigForm) (*resps.CloudItsmConfigResp, e.Error) {
	existing, err := getCloudItsmConfig(c, form.Id)
	if err != nil {
		return nil, err
	}
	next, err := cloudItsmConfigFromForm(c, &form.CreateCloudItsmConfigForm)
	if err != nil {
		return nil, err
	}
	attrs := models.Attrs{
		"name":            next.Name,
		"description":     next.Description,
		"provider":        next.Provider,
		"base_url":        next.BaseUrl,
		"auth_type":       next.AuthType,
		"username":        next.Username,
		"project_key":     next.ProjectKey,
		"ticket_type":     next.TicketType,
		"status":          next.Status,
		"timeout_seconds": next.TimeoutSeconds,
		"metadata":        next.Metadata,
	}
	if form.HasKey("token") {
		attrs["token"] = next.Token
		existing.Token = next.Token
	}
	if form.HasKey("password") {
		attrs["password"] = next.Password
		existing.Password = next.Password
	}
	if _, dbErr := c.DB().Model(&models.CloudItsmConfig{}).
		Where("id = ? and org_id = ?", form.Id, c.OrgId).
		UpdateAttrs(attrs); dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}
	existing.Name = next.Name
	existing.Description = next.Description
	existing.Provider = next.Provider
	existing.BaseUrl = next.BaseUrl
	existing.AuthType = next.AuthType
	existing.Username = next.Username
	existing.ProjectKey = next.ProjectKey
	existing.TicketType = next.TicketType
	existing.Status = next.Status
	existing.TimeoutSeconds = next.TimeoutSeconds
	existing.Metadata = next.Metadata
	resp := cloudItsmConfigResp(*existing)
	return &resp, nil
}

func DeleteCloudItsmConfig(c *ctx.ServiceContext, form *forms.CloudItsmConfigParam) (interface{}, e.Error) {
	if _, err := cloudItsmConfigVisibleQuery(c).Where("id = ?", form.Id).Delete(&models.CloudItsmConfig{}); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return nil, nil
}

func SearchCloudItsmTickets(c *ctx.ServiceContext, form *forms.SearchCloudItsmTicketForm) (interface{}, e.Error) {
	query := applyCloudItsmTicketSearch(c.DB().Model(&models.CloudItsmTicket{}).Where("org_id = ?", c.OrgId), form)
	if form.SortField() == "" {
		query = query.Order("created_at desc")
	} else {
		query = form.Order(query)
	}
	tickets := make([]models.CloudItsmTicket, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&tickets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	list := make([]resps.CloudItsmTicketResp, 0, len(tickets))
	for _, ticket := range tickets {
		list = append(list, cloudItsmTicketResp(c, ticket, nil))
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func CloudItsmOverview(c *ctx.ServiceContext) (*resps.CloudItsmOverviewResp, e.Error) {
	catalog := cloudItsmCatalog()
	metrics := resps.CloudItsmOverviewMetricResp{
		TicketAutomationTarget:    60,
		SelfServiceCoverageTarget: 80,
		IacCoverageTarget:         100,
		SelfServiceCatalogTotal:   int64(len(catalog)),
	}
	for _, item := range catalog {
		if item.Available {
			metrics.SelfServiceAvailableTotal++
		}
	}
	if metrics.SelfServiceCatalogTotal > 0 {
		metrics.SelfServiceCoverageRate = float64(metrics.SelfServiceAvailableTotal) / float64(metrics.SelfServiceCatalogTotal) * 100
	}

	var err error
	if metrics.TicketTotal, err = cloudOverviewCount(c.DB().Model(&models.CloudItsmTicket{}).Where("org_id = ?", c.OrgId)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	automatedStatuses := []string{
		models.CloudItsmTicketStatusSubmitted,
		models.CloudItsmTicketStatusInProgress,
		models.CloudItsmTicketStatusResolved,
		models.CloudItsmTicketStatusClosed,
	}
	if metrics.AutomatedTicketTotal, err = cloudOverviewCount(c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and status in (?) and (submitted_at is not null or external_id <> '' or external_key <> '' or external_url <> '')", c.OrgId, automatedStatuses)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	metrics.ManualTicketTotal = metrics.TicketTotal - metrics.AutomatedTicketTotal
	if metrics.TicketTotal > 0 {
		metrics.TicketAutomationRate = float64(metrics.AutomatedTicketTotal) / float64(metrics.TicketTotal) * 100
	}

	if metrics.AssetTotal, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).Where("org_id = ?", c.OrgId)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if metrics.IacManagedAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and (source = ? or iac_resource_id <> '')", c.OrgId, models.CmdbAssetSourceIacResource)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if metrics.CloudOnlyAssets, err = cloudOverviewCount(c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and source = ? and project_id = '' and env_id = '' and iac_resource_id = ''", c.OrgId, models.CmdbAssetSourceCloudCollect)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if metrics.AssetTotal > 0 {
		metrics.IacCoverageRate = float64(metrics.IacManagedAssets) / float64(metrics.AssetTotal) * 100
	}

	if metrics.EnvTotal, err = cloudOverviewCount(c.DB().Model(&models.Env{}).Where("org_id = ?", c.OrgId)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if metrics.DriftEnabledEnvs, err = cloudOverviewCount(c.DB().Model(&models.Env{}).Where("org_id = ? and open_cron_drift = ?", c.OrgId, true)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if metrics.AutoRepairDriftEnvs, err = cloudOverviewCount(c.DB().Model(&models.Env{}).Where("org_id = ? and open_cron_drift = ? and auto_repair_drift = ?", c.OrgId, true, true)); err != nil {
		return nil, e.New(e.DBError, err)
	}

	metrics.GitOpsGuardStatus = "warning"
	metrics.GitOpsGuardMessage = "IaC 覆盖尚未达到 100%，云上资产仍需回填或关联 IaC 项目环境"
	if metrics.AssetTotal == 0 {
		metrics.GitOpsGuardMessage = "当前组织暂无 CMDB 资产，无法评估 IaC 代码化覆盖率"
	} else if metrics.IacManagedAssets == metrics.AssetTotal && metrics.CloudOnlyAssets == 0 {
		metrics.GitOpsGuardStatus = "pass"
		metrics.GitOpsGuardMessage = "CMDB 资产已全部具备 IaC 来源或 IaC 关联"
	}

	metrics.DriftGuardStatus = "warning"
	metrics.DriftGuardMessage = "仍有环境未开启漂移检测或自动修复"
	if metrics.EnvTotal == 0 {
		metrics.DriftGuardMessage = "当前组织暂无环境，无法评估漂移检测与自动修复覆盖率"
	} else if metrics.DriftEnabledEnvs == metrics.EnvTotal && metrics.AutoRepairDriftEnvs == metrics.EnvTotal {
		metrics.DriftGuardStatus = "pass"
		metrics.DriftGuardMessage = "所有环境均已开启漂移检测和自动修复"
	}

	return &resps.CloudItsmOverviewResp{
		Metrics: metrics,
		Catalog: catalog,
	}, nil
}

func CreateCloudOperationItsmTicket(c *ctx.ServiceContext, form *forms.CreateCloudOperationItsmTicketForm) (*resps.CloudItsmTicketResp, e.Error) {
	operation, err := getCloudOperation(c, form.Id)
	if err != nil {
		return nil, err
	}
	if err := ensureCloudOperationMutationPermission(c, operation, "创建ITSM工单"); err != nil {
		return nil, err
	}
	cfg, err := getCloudItsmConfig(c, form.ConnectorId)
	if err != nil {
		return nil, err
	}
	if cfg.Status != models.CloudItsmConfigStatusEnabled {
		return nil, e.New(e.BadParam, fmt.Errorf("ITSM 连接器 %s 已禁用", cfg.Name), http.StatusBadRequest)
	}

	return createCloudItsmTicketForOperation(c, operation, cfg, form.Title, form.Description, form.Priority, form.DryRun)
}

func CreateCloudItsmSelfServiceTicket(c *ctx.ServiceContext, form *forms.CreateCloudItsmSelfServiceTicketForm) (*resps.CloudItsmTicketResp, e.Error) {
	var cfg *models.CloudItsmConfig
	var err e.Error
	if form.ConnectorId == "" {
		cfg, err = ensureDefaultCloudItsmConfig(c)
	} else {
		cfg, err = getCloudItsmConfig(c, form.ConnectorId)
	}
	if err != nil {
		return nil, err
	}
	if cfg.Status != models.CloudItsmConfigStatusEnabled {
		return nil, e.New(e.BadParam, fmt.Errorf("ITSM 连接器 %s 已禁用", cfg.Name), http.StatusBadRequest)
	}
	item, ok := cloudItsmCatalogItemByKey(form.RequestType)
	if !ok || item.OperationType != models.CloudOperationTypeSelfService || !item.Available {
		return nil, e.New(e.BadParam, fmt.Errorf("自助申请类型 %s 不支持", form.RequestType), http.StatusBadRequest)
	}
	title := strings.TrimSpace(form.Title)
	if title == "" {
		title = item.Name
	}
	description := strings.TrimSpace(form.Description)
	if description == "" {
		description = item.Description
	}
	priority := strings.TrimSpace(form.Priority)
	if priority == "" {
		priority = cloudItsmPriorityFromRisk(item.RiskLevel)
	}
	params := modelResAttrs(form.Params)
	if params == nil {
		params = models.ResAttrs{}
	}
	params["selfService"] = true
	params["requestType"] = form.RequestType
	params["automationMode"] = item.AutomationMode
	params["catalogKey"] = item.Key

	operation := &models.CloudOperation{
		OrgId:         c.OrgId,
		ProjectId:     form.ProjectId,
		EnvId:         form.EnvId,
		CreatorId:     c.UserId,
		Name:          title,
		OperationType: models.CloudOperationTypeSelfService,
		Action:        form.RequestType,
		Status:        models.CloudOperationStatusRunning,
		RiskLevel:     item.RiskLevel,
		Message:       "自助运维申请工单创建中",
		Params:        params,
	}
	operation.Id = models.NewId("cop")
	if form.DryRun {
		return createCloudItsmTicketForOperation(c, operation, cfg, title, description, priority, true)
	}
	if err := createCloudOperation(c, operation, "创建自助运维申请"); err != nil {
		return nil, err
	}
	resp, ticketErr := createCloudItsmTicketForOperation(c, operation, cfg, title, description, priority, false)
	result := models.ResAttrs{}
	if resp != nil {
		result["ticketId"] = resp.Id.String()
		result["ticketStatus"] = resp.Status
		result["externalId"] = resp.ExternalId
		result["externalKey"] = resp.ExternalKey
		result["externalUrl"] = resp.ExternalUrl
	}
	if ticketErr != nil {
		result["error"] = ticketErr.Error()
		_ = finishCloudOperation(c, operation, models.CloudOperationStatusFailed, fmt.Sprintf("自助运维申请工单创建失败：%s", ticketErr.Error()), result)
		return nil, ticketErr
	}
	if resp != nil && resp.Status == models.CloudItsmTicketStatusFailed {
		result["error"] = resp.ErrorMessage
		if err := finishCloudOperation(c, operation, models.CloudOperationStatusFailed, "自助运维申请工单提交失败", result); err != nil {
			return nil, err
		}
		return resp, nil
	}
	if err := finishCloudOperation(c, operation, models.CloudOperationStatusComplete, "自助运维申请工单已创建", result); err != nil {
		return nil, err
	}
	return resp, nil
}

func createCloudItsmTicketForOperation(c *ctx.ServiceContext, operation *models.CloudOperation, cfg *models.CloudItsmConfig, formTitle string, formDescription string, formPriority string, dryRun bool) (*resps.CloudItsmTicketResp, e.Error) {
	title := strings.TrimSpace(formTitle)
	if title == "" {
		title = fmt.Sprintf("云操作工单：%s", operation.Name)
	}
	description := strings.TrimSpace(formDescription)
	if description == "" {
		description = operation.Message
	}
	priority := strings.TrimSpace(formPriority)
	if priority == "" {
		priority = cloudItsmPriorityFromRisk(operation.RiskLevel)
	}
	payload := cloudItsmTicketRequestPayload(c, *operation, *cfg, title, description, priority)
	if dryRun {
		ticket := models.CloudItsmTicket{
			OrgId:          operation.OrgId,
			ProjectId:      operation.ProjectId,
			EnvId:          operation.EnvId,
			OperationId:    operation.Id,
			ConnectorId:    cfg.Id,
			CreatorId:      c.UserId,
			Title:          title,
			Description:    description,
			Status:         models.CloudItsmTicketStatusPending,
			Priority:       priority,
			RiskLevel:      operation.RiskLevel,
			Provider:       cfg.Provider,
			RequestPayload: payload,
		}
		resp := cloudItsmTicketResp(c, ticket, payload)
		return &resp, nil
	}
	existing := models.CloudItsmTicket{}
	if err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and operation_id = ? and connector_id = ?", c.OrgId, operation.Id, cfg.Id).
		First(&existing); err == nil {
		resp := cloudItsmTicketResp(c, existing, nil)
		return &resp, nil
	} else if !e.IsRecordNotFound(err) {
		return nil, e.New(e.DBError, err)
	}

	ticket := &models.CloudItsmTicket{
		OrgId:          operation.OrgId,
		ProjectId:      operation.ProjectId,
		EnvId:          operation.EnvId,
		OperationId:    operation.Id,
		ConnectorId:    cfg.Id,
		CreatorId:      c.UserId,
		Title:          title,
		Description:    description,
		Status:         models.CloudItsmTicketStatusPending,
		Priority:       priority,
		RiskLevel:      operation.RiskLevel,
		Provider:       cfg.Provider,
		RequestPayload: payload,
	}
	ticket.Id = models.NewId("cit")
	if err := models.Create(c.DB(), ticket); err != nil {
		return nil, e.New(e.DBError, err)
	}

	if strings.TrimSpace(cfg.BaseUrl) == "" && strings.TrimSpace(attrString(cfg.Metadata, "createTicketUrl")) == "" {
		ticket.CloudEventId = cloudItsmTicketEvent(c, ticket, operation, "itsm.ticket.created", models.CloudEventLevelInfo, "ITSM 本地工单已创建", "未配置外部工单地址，已创建本地待提交工单", nil)
		if _, dbErr := c.DB().Model(&models.CloudItsmTicket{}).
			Where("id = ? and org_id = ?", ticket.Id, c.OrgId).
			UpdateAttrs(models.Attrs{"cloud_event_id": ticket.CloudEventId}); dbErr != nil {
			return nil, e.New(e.DBError, dbErr)
		}
		resp := cloudItsmTicketResp(c, *ticket, nil)
		return &resp, nil
	}

	submit := submitCloudItsmTicket(*cfg, payload)
	attrs := models.Attrs{
		"status":           submit.Status,
		"external_id":      submit.ExternalId,
		"external_key":     submit.ExternalKey,
		"external_url":     submit.ExternalUrl,
		"response_payload": submit.ResponsePayload,
		"error_message":    submit.ErrorMessage,
		"submitted_at":     submit.SubmittedAt,
		"last_synced_at":   models.Time(time.Now()),
	}
	if _, dbErr := c.DB().Model(&models.CloudItsmTicket{}).
		Where("id = ? and org_id = ?", ticket.Id, c.OrgId).
		UpdateAttrs(attrs); dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}
	ticket.Status = submit.Status
	ticket.ExternalId = submit.ExternalId
	ticket.ExternalKey = submit.ExternalKey
	ticket.ExternalUrl = submit.ExternalUrl
	ticket.ResponsePayload = submit.ResponsePayload
	ticket.ErrorMessage = submit.ErrorMessage
	ticket.SubmittedAt = submit.SubmittedAt
	ticket.LastSyncedAt = attrs["last_synced_at"].(models.Time)

	eventType := "itsm.ticket.submitted"
	level := models.CloudEventLevelInfo
	titleMsg := "ITSM 工单已提交"
	message := fmt.Sprintf("已向 %s 提交 ITSM 工单", cfg.Name)
	if ticket.Status == models.CloudItsmTicketStatusFailed {
		eventType = "itsm.ticket.failed"
		level = models.CloudEventLevelError
		titleMsg = "ITSM 工单提交失败"
		message = submit.ErrorMessage
	}
	eventId := cloudItsmTicketEvent(c, ticket, operation, eventType, level, titleMsg, message, submit.ResponsePayload)
	if _, dbErr := c.DB().Model(&models.CloudItsmTicket{}).
		Where("id = ? and org_id = ?", ticket.Id, c.OrgId).
		UpdateAttrs(models.Attrs{"cloud_event_id": eventId}); dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}
	ticket.CloudEventId = eventId
	if auditErr := recordCloudOperationAudit(c, operation, operation.Status, titleMsg, models.ResAttrs{
		"connectorId": cfg.Id.String(),
		"ticketId":    ticket.Id.String(),
	}, models.ResAttrs{
		"status":      ticket.Status,
		"externalId":  ticket.ExternalId,
		"externalKey": ticket.ExternalKey,
		"externalUrl": ticket.ExternalUrl,
	}); auditErr != nil {
		return nil, auditErr
	}
	resp := cloudItsmTicketResp(c, *ticket, nil)
	return &resp, nil
}

func UpdateCloudItsmTicketStatus(c *ctx.ServiceContext, form *forms.UpdateCloudItsmTicketStatusForm) (*resps.CloudItsmTicketResp, e.Error) {
	ticket := models.CloudItsmTicket{}
	if err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and id = ?", c.OrgId, form.Id).
		First(&ticket); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	attrs := models.Attrs{
		"status":         form.Status,
		"last_synced_at": models.Time(time.Now()),
	}
	if form.HasKey("externalId") {
		attrs["external_id"] = strings.TrimSpace(form.ExternalId)
	}
	if form.HasKey("externalKey") {
		attrs["external_key"] = strings.TrimSpace(form.ExternalKey)
	}
	if form.HasKey("externalUrl") {
		attrs["external_url"] = strings.TrimSpace(form.ExternalUrl)
	}
	if form.HasKey("payload") {
		attrs["response_payload"] = form.Payload
	}
	if cloudItsmTicketClosed(form.Status) {
		attrs["closed_at"] = models.Time(time.Now())
	}
	if _, err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("id = ? and org_id = ?", form.Id, c.OrgId).
		UpdateAttrs(attrs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	ticket.Status = form.Status
	ticket.LastSyncedAt = attrs["last_synced_at"].(models.Time)
	if value, ok := attrs["external_id"].(string); ok {
		ticket.ExternalId = value
	}
	if value, ok := attrs["external_key"].(string); ok {
		ticket.ExternalKey = value
	}
	if value, ok := attrs["external_url"].(string); ok {
		ticket.ExternalUrl = value
	}
	if value, ok := attrs["response_payload"].(models.ResAttrs); ok {
		ticket.ResponsePayload = value
	}
	if value, ok := attrs["closed_at"].(models.Time); ok {
		ticket.ClosedAt = value
	}

	var operation *models.CloudOperation
	if ticket.OperationId != "" {
		if op, opErr := getCloudOperation(c, ticket.OperationId); opErr == nil {
			operation = op
		}
	}
	eventPayload := models.ResAttrs{
		"ticketId":    ticket.Id.String(),
		"status":      ticket.Status,
		"comment":     form.Comment,
		"externalId":  ticket.ExternalId,
		"externalKey": ticket.ExternalKey,
		"externalUrl": ticket.ExternalUrl,
	}
	for key, value := range form.Payload {
		eventPayload[key] = value
	}
	ticket.CloudEventId = cloudItsmTicketEvent(c, &ticket, operation, "itsm.ticket.updated", models.CloudEventLevelInfo, "ITSM 工单状态已更新", fmt.Sprintf("工单状态更新为 %s", ticket.Status), eventPayload)
	if _, err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("id = ? and org_id = ?", ticket.Id, c.OrgId).
		UpdateAttrs(models.Attrs{"cloud_event_id": ticket.CloudEventId}); err != nil {
		return nil, e.New(e.DBError, err)
	}
	resp := cloudItsmTicketResp(c, ticket, nil)
	return &resp, nil
}

func cloudItsmConfigVisibleQuery(c *ctx.ServiceContext) *db.Session {
	return c.DB().Model(&models.CloudItsmConfig{}).Where("org_id = ?", c.OrgId)
}

func ensureDefaultCloudItsmConfig(c *ctx.ServiceContext) (*models.CloudItsmConfig, e.Error) {
	cfg := models.CloudItsmConfig{}
	if err := cloudItsmConfigVisibleQuery(c).
		Where("status = ?", models.CloudItsmConfigStatusEnabled).
		Order("created_at asc").
		First(&cfg); err == nil {
		return &cfg, nil
	} else if !e.IsRecordNotFound(err) {
		return nil, e.New(e.DBError, err)
	}

	cfg = models.CloudItsmConfig{}
	if err := cloudItsmConfigVisibleQuery(c).
		Where("name = ?", defaultCloudItsmConnectorName).
		First(&cfg); err == nil {
		attrs := models.Attrs{
			"status": models.CloudItsmConfigStatusEnabled,
		}
		cfg.Status = models.CloudItsmConfigStatusEnabled
		if strings.TrimSpace(cfg.Description) == "" {
			attrs["description"] = defaultCloudItsmConnectorDescription
			cfg.Description = defaultCloudItsmConnectorDescription
		}
		if strings.TrimSpace(cfg.Provider) == "" {
			attrs["provider"] = models.CloudItsmProviderGeneric
			cfg.Provider = models.CloudItsmProviderGeneric
		}
		if strings.TrimSpace(cfg.AuthType) == "" {
			attrs["auth_type"] = models.CloudItsmAuthNone
			cfg.AuthType = models.CloudItsmAuthNone
		}
		if cfg.TimeoutSeconds <= 0 {
			attrs["timeout_seconds"] = 10
			cfg.TimeoutSeconds = 10
		}
		if cfg.Metadata == nil {
			attrs["metadata"] = defaultCloudItsmConnectorMetadata()
			cfg.Metadata = defaultCloudItsmConnectorMetadata()
		}
		if _, dbErr := c.DB().Model(&models.CloudItsmConfig{}).
			Where("id = ? and org_id = ?", cfg.Id, c.OrgId).
			UpdateAttrs(attrs); dbErr != nil {
			return nil, e.New(e.DBError, dbErr)
		}
		return &cfg, nil
	} else if !e.IsRecordNotFound(err) {
		return nil, e.New(e.DBError, err)
	}

	cfg = models.CloudItsmConfig{
		OrgId:          c.OrgId,
		Name:           defaultCloudItsmConnectorName,
		Description:    defaultCloudItsmConnectorDescription,
		Provider:       models.CloudItsmProviderGeneric,
		AuthType:       models.CloudItsmAuthNone,
		Status:         models.CloudItsmConfigStatusEnabled,
		TimeoutSeconds: 10,
		Metadata:       defaultCloudItsmConnectorMetadata(),
	}
	cfg.Id = models.NewId("citc")
	if err := models.Create(c.DB(), &cfg); err != nil {
		retry := models.CloudItsmConfig{}
		if retryErr := cloudItsmConfigVisibleQuery(c).
			Where("name = ?", defaultCloudItsmConnectorName).
			First(&retry); retryErr == nil {
			return &retry, nil
		}
		return nil, e.New(e.DBError, err)
	}
	return &cfg, nil
}

func defaultCloudItsmConnectorMetadata() models.ResAttrs {
	return models.ResAttrs{
		"builtin":   true,
		"localOnly": true,
		"managedBy": "cloudiac",
	}
}

func applyCloudItsmConfigSearch(query *db.Session, form *forms.SearchCloudItsmConfigForm) *db.Session {
	if form.Q != "" {
		q := "%" + strings.TrimSpace(form.Q) + "%"
		query = query.Where("id like ? or name like ? or description like ? or base_url like ? or project_key like ?", q, q, q, q, q)
	}
	if form.Provider != "" {
		query = query.Where("provider = ?", form.Provider)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	return query
}

func applyCloudItsmTicketSearch(query *db.Session, form *forms.SearchCloudItsmTicketForm) *db.Session {
	if form.Q != "" {
		q := "%" + strings.TrimSpace(form.Q) + "%"
		query = query.Where("id like ? or external_id like ? or external_key like ? or title like ? or description like ? or error_message like ?", q, q, q, q, q, q)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	if form.ConnectorId != "" {
		query = query.Where("connector_id = ?", form.ConnectorId)
	}
	if form.OperationId != "" {
		query = query.Where("operation_id = ?", form.OperationId)
	}
	if form.ProjectId != "" {
		query = query.Where("project_id = ?", form.ProjectId)
	}
	if form.EnvId != "" {
		query = query.Where("env_id = ?", form.EnvId)
	}
	return query
}

func cloudItsmConfigFromForm(c *ctx.ServiceContext, form *forms.CreateCloudItsmConfigForm) (*models.CloudItsmConfig, e.Error) {
	provider := strings.TrimSpace(form.Provider)
	if provider == "" {
		provider = models.CloudItsmProviderGeneric
	}
	authType := strings.TrimSpace(form.AuthType)
	if authType == "" {
		authType = models.CloudItsmAuthNone
	}
	status := strings.TrimSpace(form.Status)
	if status == "" {
		status = models.CloudItsmConfigStatusEnabled
	}
	baseUrl := strings.TrimSpace(form.BaseUrl)
	if baseUrl != "" {
		parsed, err := url.Parse(baseUrl)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return nil, e.New(e.BadParam, fmt.Errorf("baseUrl 不是有效 URL"), http.StatusBadRequest)
		}
	}
	timeoutSeconds := form.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 10
	}
	if timeoutSeconds > 120 {
		timeoutSeconds = 120
	}
	return &models.CloudItsmConfig{
		OrgId:          c.OrgId,
		Name:           strings.TrimSpace(form.Name),
		Description:    strings.TrimSpace(form.Description),
		Provider:       provider,
		BaseUrl:        baseUrl,
		AuthType:       authType,
		Token:          strings.TrimSpace(form.Token),
		Username:       strings.TrimSpace(form.Username),
		Password:       form.Password,
		ProjectKey:     strings.TrimSpace(form.ProjectKey),
		TicketType:     strings.TrimSpace(form.TicketType),
		Status:         status,
		TimeoutSeconds: timeoutSeconds,
		Metadata:       cloudItsmMetadata(form.Metadata),
	}, nil
}

func getCloudItsmConfig(c *ctx.ServiceContext, id models.Id) (*models.CloudItsmConfig, e.Error) {
	cfg := models.CloudItsmConfig{}
	if err := cloudItsmConfigVisibleQuery(c).Where("id = ?", id).First(&cfg); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	return &cfg, nil
}

func cloudItsmConfigResp(cfg models.CloudItsmConfig) resps.CloudItsmConfigResp {
	return resps.CloudItsmConfigResp{
		CloudItsmConfig:    cfg,
		TokenConfigured:    cfg.Token != "",
		PasswordConfigured: cfg.Password != "",
	}
}

func cloudItsmTicketResp(c *ctx.ServiceContext, ticket models.CloudItsmTicket, dryRunPayload models.ResAttrs) resps.CloudItsmTicketResp {
	resp := resps.CloudItsmTicketResp{
		CloudItsmTicket: ticket,
		ConnectorName:   lookupName(c, &models.CloudItsmConfig{}, ticket.ConnectorId),
		OperationName:   lookupName(c, &models.CloudOperation{}, ticket.OperationId),
		ProjectName:     lookupName(c, &models.Project{}, ticket.ProjectId),
		EnvName:         lookupName(c, &models.Env{}, ticket.EnvId),
		CreatorName:     lookupName(c, &models.User{}, ticket.CreatorId),
		DryRunPayload:   dryRunPayload,
	}
	return resp
}

func cloudItsmCatalog() []resps.CloudItsmCatalogItemResp {
	items := make([]resps.CloudItsmCatalogItemResp, 0)
	for _, spec := range cloudAssetActionSpecs() {
		category := "operations"
		switch spec.Key {
		case models.CloudOperationActionResizeInstance, models.CloudOperationActionResizeVolume:
			category = "scaling"
		case models.CloudOperationActionUpdateTags, models.CloudOperationActionUpdateSecurityRules:
			category = "configuration"
		case models.CloudOperationActionStartInstance, models.CloudOperationActionStopInstance, models.CloudOperationActionRestartInstance:
			category = "lifecycle"
		case models.CloudOperationActionCreateSnapshot:
			category = "backup"
		}
		mode := "auto"
		if spec.RequiresApproval {
			mode = "approval"
		}
		if spec.AdapterMode == cloudOperationAdapterModeProvider {
			mode = "provider"
			if spec.RequiresApproval {
				mode = "provider_approval"
			}
		}
		items = append(items, resps.CloudItsmCatalogItemResp{
			Key:                 spec.Key,
			Name:                spec.Name,
			Category:            category,
			Description:         spec.Description,
			Source:              "cloud_operation",
			OperationType:       models.CloudOperationTypeAction,
			Action:              spec.Key,
			AutomationMode:      mode,
			Available:           spec.Enabled,
			Enabled:             spec.Enabled,
			RequiresApproval:    spec.RequiresApproval,
			RiskLevel:           spec.RiskLevel,
			SupportedProviders:  spec.SupportedProviders,
			SupportedAssetTypes: spec.SupportedAssetTypes,
			DisabledReason:      spec.DisabledReason,
		})
	}
	items = append(items,
		resps.CloudItsmCatalogItemResp{
			Key:            models.CloudOperationActionPermissionRequest,
			Name:           "权限申请",
			Category:       "permission",
			Description:    "业务团队自助提交云账号、项目、环境或资源操作权限申请，进入 ITSM 工单流转",
			Source:         "itsm_self_service",
			OperationType:  models.CloudOperationTypeSelfService,
			Action:         models.CloudOperationActionPermissionRequest,
			AutomationMode: "itsm",
			Available:      true,
			Enabled:        true,
			RiskLevel:      models.CloudOperationRiskMedium,
		},
		resps.CloudItsmCatalogItemResp{
			Key:            models.CloudOperationActionGitOpsIacChange,
			Name:           "GitOps/IaC 变更申请",
			Category:       "gitops_iac",
			Description:    "基础设施变更通过 PR review 和自动化流水线执行；平台保留申请、审批和外部工单追踪",
			Source:         "itsm_self_service",
			OperationType:  models.CloudOperationTypeSelfService,
			Action:         models.CloudOperationActionGitOpsIacChange,
			AutomationMode: "itsm",
			Available:      true,
			Enabled:        true,
			RiskLevel:      models.CloudOperationRiskHigh,
		},
		resps.CloudItsmCatalogItemResp{
			Key:            models.CloudOperationActionDriftRemediation,
			Name:           "漂移修复申请",
			Category:       "drift",
			Description:    "dev/staging/prod 环境发现配置漂移后，自助提交修复申请并关联自动修复流程",
			Source:         "itsm_self_service",
			OperationType:  models.CloudOperationTypeSelfService,
			Action:         models.CloudOperationActionDriftRemediation,
			AutomationMode: "itsm",
			Available:      true,
			Enabled:        true,
			RiskLevel:      models.CloudOperationRiskHigh,
		},
	)
	return items
}

func cloudItsmCatalogItemByKey(key string) (resps.CloudItsmCatalogItemResp, bool) {
	key = strings.TrimSpace(key)
	for _, item := range cloudItsmCatalog() {
		if item.Key == key {
			return item, true
		}
	}
	return resps.CloudItsmCatalogItemResp{}, false
}

func cloudItsmMetadata(metadata models.ResAttrs) models.ResAttrs {
	if metadata == nil {
		return models.ResAttrs{}
	}
	return metadata
}

func cloudItsmPriorityFromRisk(risk string) string {
	switch risk {
	case models.CloudOperationRiskCritical:
		return "critical"
	case models.CloudOperationRiskHigh:
		return "high"
	case models.CloudOperationRiskMedium:
		return "medium"
	default:
		return "low"
	}
}

func cloudItsmTicketRequestPayload(c *ctx.ServiceContext, operation models.CloudOperation, cfg models.CloudItsmConfig, title string, description string, priority string) models.ResAttrs {
	return models.ResAttrs{
		"title":       title,
		"description": description,
		"priority":    priority,
		"projectKey":  cfg.ProjectKey,
		"ticketType":  cfg.TicketType,
		"source":      "cloudiac",
		"operation": models.ResAttrs{
			"id":             operation.Id.String(),
			"name":           operation.Name,
			"operationType":  operation.OperationType,
			"action":         operation.Action,
			"status":         operation.Status,
			"riskLevel":      operation.RiskLevel,
			"provider":       operation.Provider,
			"resourceType":   operation.ResourceType,
			"resourceId":     operation.ResourceId,
			"resourceName":   operation.ResourceName,
			"projectId":      operation.ProjectId.String(),
			"envId":          operation.EnvId.String(),
			"assetId":        operation.AssetId.String(),
			"cloudAccountId": operation.CloudAccountId.String(),
			"params":         operation.Params,
			"result":         operation.Result,
			"message":        operation.Message,
		},
		"requester": models.ResAttrs{
			"userId":   c.UserId.String(),
			"username": c.Username,
			"email":    c.Email,
		},
		"connector": models.ResAttrs{
			"id":       cfg.Id.String(),
			"name":     cfg.Name,
			"provider": cfg.Provider,
		},
	}
}

func submitCloudItsmTicket(cfg models.CloudItsmConfig, payload models.ResAttrs) cloudItsmSubmitResult {
	result := cloudItsmSubmitResult{
		Status:      models.CloudItsmTicketStatusFailed,
		SubmittedAt: models.Time(time.Now()),
	}
	endpoint := cloudItsmTicketEndpoint(cfg)
	if endpoint == "" {
		result.Status = models.CloudItsmTicketStatusPending
		return result
	}
	body, err := json.Marshal(payload)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	switch cfg.AuthType {
	case models.CloudItsmAuthBearer:
		if cfg.Token != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.Token)
		}
	case models.CloudItsmAuthBasic:
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 10
	}
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		result.ErrorMessage = err.Error()
		result.ResponsePayload = models.ResAttrs{
			"endpoint": endpoint,
			"error":    err.Error(),
		}
		return result
	}
	defer resp.Body.Close()

	bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if readErr != nil {
		result.ErrorMessage = readErr.Error()
	}
	responsePayload := models.ResAttrs{
		"endpoint":   endpoint,
		"statusCode": resp.StatusCode,
		"body":       string(bodyBytes),
	}
	decoded := models.ResAttrs{}
	if len(bodyBytes) > 0 && json.Unmarshal(bodyBytes, &decoded) == nil {
		responsePayload["json"] = decoded
		result.ExternalId = firstNonEmpty(attrString(decoded, "id"), attrString(decoded, "externalId"), attrString(decoded, "ticketId"))
		result.ExternalKey = firstNonEmpty(attrString(decoded, "key"), attrString(decoded, "externalKey"), attrString(decoded, "number"))
		result.ExternalUrl = firstNonEmpty(attrString(decoded, "url"), attrString(decoded, "externalUrl"), attrString(decoded, "self"))
		if nested, ok := decoded["ticket"].(map[string]interface{}); ok {
			nestedAttrs := models.ResAttrs(nested)
			result.ExternalId = firstNonEmpty(result.ExternalId, attrString(nestedAttrs, "id"))
			result.ExternalKey = firstNonEmpty(result.ExternalKey, attrString(nestedAttrs, "key"), attrString(nestedAttrs, "number"))
			result.ExternalUrl = firstNonEmpty(result.ExternalUrl, attrString(nestedAttrs, "url"), attrString(nestedAttrs, "self"))
		}
		if nested, ok := decoded["data"].(map[string]interface{}); ok {
			nestedAttrs := models.ResAttrs(nested)
			result.ExternalId = firstNonEmpty(result.ExternalId, attrString(nestedAttrs, "id"), attrString(nestedAttrs, "ticketId"))
			result.ExternalKey = firstNonEmpty(result.ExternalKey, attrString(nestedAttrs, "key"), attrString(nestedAttrs, "number"))
			result.ExternalUrl = firstNonEmpty(result.ExternalUrl, attrString(nestedAttrs, "url"), attrString(nestedAttrs, "self"))
		}
	}
	result.ResponsePayload = responsePayload
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Status = models.CloudItsmTicketStatusSubmitted
		if result.ExternalUrl == "" && result.ExternalKey != "" {
			result.ExternalUrl = cloudItsmTicketExternalUrl(cfg, result.ExternalKey)
		}
		return result
	}
	result.ErrorMessage = fmt.Sprintf("ITSM 工单接口返回 HTTP %d", resp.StatusCode)
	return result
}

func cloudItsmTicketEndpoint(cfg models.CloudItsmConfig) string {
	if value := strings.TrimSpace(attrString(cfg.Metadata, "createTicketUrl")); value != "" {
		if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			return value
		}
		if cfg.BaseUrl != "" {
			return strings.TrimRight(cfg.BaseUrl, "/") + "/" + strings.TrimLeft(value, "/")
		}
	}
	if cfg.BaseUrl == "" {
		return ""
	}
	path := strings.TrimSpace(attrString(cfg.Metadata, "createTicketPath"))
	if path == "" {
		path = "tickets"
	}
	return strings.TrimRight(cfg.BaseUrl, "/") + "/" + strings.TrimLeft(path, "/")
}

func cloudItsmTicketExternalUrl(cfg models.CloudItsmConfig, key string) string {
	if cfg.BaseUrl == "" || key == "" {
		return ""
	}
	path := strings.TrimSpace(attrString(cfg.Metadata, "browseTicketPath"))
	if path == "" {
		path = "tickets/" + url.PathEscape(key)
	} else {
		path = strings.ReplaceAll(path, "{key}", url.PathEscape(key))
	}
	return strings.TrimRight(cfg.BaseUrl, "/") + "/" + strings.TrimLeft(path, "/")
}

func cloudItsmTicketClosed(status string) bool {
	return status == models.CloudItsmTicketStatusResolved ||
		status == models.CloudItsmTicketStatusClosed ||
		status == models.CloudItsmTicketStatusCanceled
}

func cloudItsmTicketEvent(c *ctx.ServiceContext, ticket *models.CloudItsmTicket, operation *models.CloudOperation, eventType string, level string, title string, message string, payload models.ResAttrs) models.Id {
	if ticket == nil {
		return ""
	}
	if payload == nil {
		payload = models.ResAttrs{}
	}
	payload["ticketId"] = ticket.Id.String()
	payload["connectorId"] = ticket.ConnectorId.String()
	payload["externalId"] = ticket.ExternalId
	payload["externalKey"] = ticket.ExternalKey
	payload["externalUrl"] = ticket.ExternalUrl
	event := models.CloudEvent{
		OrgId:       ticket.OrgId,
		ProjectId:   ticket.ProjectId,
		EnvId:       ticket.EnvId,
		OperationId: ticket.OperationId,
		ActorId:     ticket.CreatorId,
		Source:      models.CloudEventSourceITSM,
		EventType:   eventType,
		Level:       level,
		Status:      ticket.Status,
		Title:       title,
		Message:     message,
		Payload:     payload,
	}
	if operation != nil {
		event.AssetId = operation.AssetId
		event.CloudAccountId = operation.CloudAccountId
		event.Provider = operation.Provider
		event.ResourceType = operation.ResourceType
		event.ResourceId = operation.ResourceId
		event.ResourceName = operation.ResourceName
	}
	event.Id = models.NewId("cev")
	recordCloudEventBestEffort(c, event)
	return event.Id
}
