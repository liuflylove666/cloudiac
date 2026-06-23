// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"bytes"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloudiac/portal/consts"
	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/libs/db"
	"cloudiac/portal/libs/page"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	"cloudiac/portal/models/resps"
	"cloudiac/portal/services"
	cloudwebhooksrv "cloudiac/portal/services/cloudwebhook"
	"cloudiac/utils/logs"
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

type cloudEventItsmDispatchSpec struct {
	Enabled      bool
	ConnectorIds []models.Id
	Title        string
	Description  string
	Priority     string
	Reason       string
}

const (
	defaultCloudItsmConnectorName        = "CloudIaC 本地工单"
	defaultCloudItsmConnectorDescription = "平台内置本地工单连接器，未配置外部 ITSM 时用于自助申请和审计流转"
	cloudItsmDefaultSlaTarget            = 90
	cloudItsmGitOpsGateStatusPassed      = "passed"
	cloudItsmGitOpsGateStatusWaiting     = "waiting"
	cloudItsmGitOpsGateStatusBlocked     = "blocked"
	cloudItsmCatalogPolicyHistoryLimit   = 50
)

const (
	cloudItsmRobotProcessorCloudiac = "cloudiac_robot"
	cloudItsmRobotChannelLocal      = "local"
	cloudItsmRobotChannelExternal   = "external"
	cloudItsmRobotModeEventTicket   = "event_auto_ticket"
)

const cloudItsmDeadLetterApprovalEvidenceItemLimit = 10

const (
	cloudItsmRobotTagSelfService      = "self_service"
	cloudItsmRobotTagCloudOperation   = "cloud_operation"
	cloudItsmRobotTagGitOpsIac        = "gitops_iac"
	cloudItsmRobotTagDriftRemediation = "drift_auto_repair"
	cloudItsmRobotTagRiskRemediation  = "risk_remediation"
	cloudItsmRobotTagPermission       = "permission_request"
	cloudItsmRobotTagEventAutoTicket  = "event_auto_ticket"
	cloudItsmRobotTagApprovalRequired = "approval_required"
	cloudItsmRobotTagExternalItsm     = "external_itsm"
	cloudItsmRobotTagLocalTicket      = "local_ticket"
)

const (
	CloudItsmCallbackSignatureHeader    = "X-CloudIaC-ITSM-Signature"
	cloudItsmCallbackGenericSignHeader  = "X-CloudIaC-Signature"
	cloudItsmCallbackSignatureAlgorithm = "HMAC-SHA256"
)

const (
	cloudItsmStatusSyncDefaultLimit           = 50
	cloudItsmStatusSyncMaxLimit               = 200
	cloudItsmStatusSyncDefaultIntervalSeconds = 300
	cloudItsmStatusSyncWorkerInterval         = 5 * time.Minute
	cloudItsmSubmitRetryDefaultLimit          = 50
	cloudItsmSubmitRetryMaxLimit              = 200
	cloudItsmSubmitRetryDefaultMaxAttempts    = 3
	cloudItsmSubmitRetryMaxAttemptsLimit      = 10
	cloudItsmSubmitRetryDefaultBackoffSeconds = 300
	cloudItsmSubmitRetryDefaultMaxBackoff     = 3600
	cloudItsmSubmitRetryWorkerInterval        = 5 * time.Minute
	cloudItsmDriftAutoRepairRetryDefaultMax   = 1
	cloudItsmDriftAutoRepairRetryMax          = 3
)

type cloudItsmSubmitRetryEligibility struct {
	Eligible    bool
	Reason      string
	Attempt     int
	NextAttempt int
	NextRetryAt time.Time
	DeadLetter  bool
}

const (
	cloudItsmDriftAutoRepairApprovalContextRepair = "auto_repair"
	cloudItsmDriftAutoRepairApprovalContextRetry  = "retry"

	cloudItsmDriftAutoRepairApprovalModeInheritEnv       = "inherit_env"
	cloudItsmDriftAutoRepairApprovalModeRequireApproval  = "require_approval"
	cloudItsmDriftAutoRepairApprovalModeAutoApprove      = "auto_approve"
	cloudItsmDriftAutoRepairApprovalModeRetryRequireOnly = "retry_require_approval"
)

type cloudItsmDriftAutoRepairApprovalPolicy struct {
	Context          string
	Source           string
	Mode             string
	AutoApprove      bool
	RequiresApproval bool
	Reason           string
	ApproverRoles    []string
}

type cloudItsmDriftAutoRepairSlaPolicy struct {
	Source                   string
	ApprovalDueMinutes       int
	RollbackReviewDueMinutes int
	DueSoonMinutes           int
	NotificationRoutes       []string
	NotificationAssignees    []string
	OwnerRoles               []string
	AutoTicket               bool
}

type cloudItsmStatusPollResult struct {
	Status          string
	ExternalId      string
	ExternalKey     string
	ExternalUrl     string
	ResponsePayload models.ResAttrs
}

type cloudItsmSystemRequestContext struct {
	sc *ctx.ServiceContext
}

func (r *cloudItsmSystemRequestContext) BindService(sc *ctx.ServiceContext) {
	r.sc = sc
}

func (r *cloudItsmSystemRequestContext) Service() *ctx.ServiceContext {
	return r.sc
}

func (r *cloudItsmSystemRequestContext) Logger() logs.Logger {
	if r.sc != nil {
		return r.sc.Logger()
	}
	return logs.Get()
}

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
	catalog := cloudItsmCatalog(c)
	metrics := resps.CloudItsmOverviewMetricResp{
		TicketAutomationTarget:    60,
		SelfServiceCoverageTarget: 80,
		SelfServiceSlaTarget:      cloudItsmDefaultSlaTarget,
		IacCoverageTarget:         100,
		SelfServiceCatalogTotal:   int64(len(catalog)),
	}
	for _, item := range catalog {
		if item.Available {
			metrics.SelfServiceAvailableTotal++
		}
		if item.PolicyKey != "" {
			metrics.SelfServicePolicyTotal++
		}
		if item.PolicyKey != "" && (len(item.RequiredRoles) > 0 || len(item.AllowedScopes) > 0) {
			metrics.SelfServicePolicyCovered++
		}
	}
	if metrics.SelfServiceCatalogTotal > 0 {
		metrics.SelfServiceCoverageRate = float64(metrics.SelfServiceAvailableTotal) / float64(metrics.SelfServiceCatalogTotal) * 100
	}
	if metrics.SelfServicePolicyTotal > 0 {
		metrics.SelfServicePolicyRate = float64(metrics.SelfServicePolicyCovered) / float64(metrics.SelfServicePolicyTotal) * 100
	}

	var err error
	if metrics.TicketTotal, err = cloudOverviewCount(c.DB().Model(&models.CloudItsmTicket{}).Where("org_id = ?", c.OrgId)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	robotProcessedWhere := "(JSON_UNQUOTE(JSON_EXTRACT(request_payload, '$.robotProcessed')) = 'true' or JSON_UNQUOTE(JSON_EXTRACT(request_payload, '$.robot.processed')) = 'true')"
	if metrics.RobotProcessedTicketTotal, err = cloudOverviewCount(c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and "+robotProcessedWhere, c.OrgId)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	automatedStatuses := []string{
		models.CloudItsmTicketStatusSubmitted,
		models.CloudItsmTicketStatusInProgress,
		models.CloudItsmTicketStatusResolved,
		models.CloudItsmTicketStatusClosed,
	}
	if metrics.AutomatedTicketTotal, err = cloudOverviewCount(c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and ("+robotProcessedWhere+" or (status in (?) and (submitted_at is not null or external_id <> '' or external_key <> '' or external_url <> '')))", c.OrgId, automatedStatuses)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	metrics.ManualTicketTotal = metrics.TicketTotal - metrics.AutomatedTicketTotal
	if metrics.TicketTotal > 0 {
		metrics.TicketAutomationRate = float64(metrics.AutomatedTicketTotal) / float64(metrics.TicketTotal) * 100
		metrics.RobotProcessingRate = float64(metrics.RobotProcessedTicketTotal) / float64(metrics.TicketTotal) * 100
	}
	slaTrend, slaMetrics, slaErr := cloudItsmSelfServiceSlaTrend(c, time.Now(), metrics.SelfServiceSlaTarget)
	if slaErr != nil {
		return nil, slaErr
	}
	metrics.SelfServiceSlaTicketTotal = slaMetrics.SelfServiceSlaTicketTotal
	metrics.SelfServiceSlaMetTotal = slaMetrics.SelfServiceSlaMetTotal
	metrics.SelfServiceSlaBreached = slaMetrics.SelfServiceSlaBreached
	metrics.SelfServiceSlaMetRate = slaMetrics.SelfServiceSlaMetRate
	projectTrends, requestTypeTrends, teamTrends, trendErr := cloudItsmDimensionTrends(c, time.Now(), metrics)
	if trendErr != nil {
		return nil, trendErr
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
		Metrics:           metrics,
		Catalog:           catalog,
		SlaTrend:          slaTrend,
		ProjectTrends:     projectTrends,
		RequestTypeTrends: requestTypeTrends,
		TeamTrends:        teamTrends,
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
	item, ok := cloudItsmCatalogItemByKey(c, form.RequestType)
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
	if normalized, err := normalizeCloudItsmSelfServiceParams(form.RequestType, params); err != nil {
		return nil, err
	} else {
		params = normalized
	}
	cloudItsmApplySelfServicePolicyToParams(params, item)
	cloudItsmApplyRobotProcessingToParams(params, cloudItsmRobotProcessingForSelfService(item, *cfg, params))

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

func UpdateCloudItsmCatalogPolicy(c *ctx.ServiceContext, form *forms.UpdateCloudItsmCatalogPolicyForm) (*resps.CloudItsmCatalogItemResp, e.Error) {
	key := strings.TrimSpace(form.Key)
	baseItem, ok := cloudItsmCatalogItemByKey(nil, key)
	if !ok {
		return nil, e.New(e.BadParam, fmt.Errorf("自助目录服务项 %s 不存在", key), http.StatusBadRequest)
	}
	config := cloudItsmCatalogPolicyConfig(c)
	history := cloudItsmCatalogPolicyHistoryConfig(c)
	beforeItem, _ := cloudItsmCatalogItemByKey(c, key)
	beforeSnapshot := cloudItsmCatalogPolicySnapshot(beforeItem)
	version := cloudItsmCatalogPolicyNextVersion(history, key, config[key])
	updatedAt := time.Now().Format(time.RFC3339)
	updatedBy := c.UserId.String()
	action := "update"
	afterItem := baseItem
	if form.Reset {
		delete(config, key)
		action = "reset"
	} else {
		roles := cloudItsmNormalizeCatalogPolicyValues(form.RequiredRoles, map[string]bool{
			"project_member": true,
			"project_owner":  true,
			"iac_reviewer":   true,
			"operator":       true,
			"sre":            true,
			"security_owner": true,
		})
		scopes := cloudItsmNormalizeCatalogPolicyValues(form.AllowedScopes, map[string]bool{
			"org":     true,
			"project": true,
			"env":     true,
			"asset":   true,
		})
		if len(roles) == 0 {
			return nil, e.New(e.BadParam, fmt.Errorf("自助目录策略至少需要一个角色"), http.StatusBadRequest)
		}
		if len(scopes) == 0 {
			return nil, e.New(e.BadParam, fmt.Errorf("自助目录策略至少需要一个适用范围"), http.StatusBadRequest)
		}
		if form.SlaMinutes <= 0 || form.SlaMinutes > 43200 {
			return nil, e.New(e.BadParam, fmt.Errorf("SLA 分钟数必须在 1 到 43200 之间"), http.StatusBadRequest)
		}
		override := models.ResAttrs{
			"enabled":           form.Enabled,
			"policyName":        strings.TrimSpace(form.PolicyName),
			"policyDescription": strings.TrimSpace(form.PolicyDescription),
			"requiredRoles":     roles,
			"allowedScopes":     scopes,
			"slaMinutes":        form.SlaMinutes,
			"policyVersion":     version,
			"updatedAt":         updatedAt,
			"updatedBy":         updatedBy,
		}
		afterItem = cloudItsmApplyCatalogPolicyOverride(baseItem, override)
		diff := cloudItsmCatalogPolicyDiff(beforeSnapshot, cloudItsmCatalogPolicySnapshot(afterItem))
		override["policyLastDiff"] = diff
		config[key] = override
		afterItem.PolicyLastDiff = diff
	}
	afterSnapshot := cloudItsmCatalogPolicySnapshot(afterItem)
	historyEntry := cloudItsmCatalogPolicyHistoryEntry(key, action, version, updatedAt, updatedBy, beforeSnapshot, afterSnapshot)
	history = cloudItsmAppendCatalogPolicyHistory(history, key, historyEntry)
	if err := saveCloudItsmCatalogPolicyConfig(c, config); err != nil {
		return nil, err
	}
	if err := saveCloudItsmCatalogPolicyHistoryConfig(c, history); err != nil {
		return nil, err
	}
	cloudItsmCatalogPolicyEvent(c, afterItem, historyEntry)
	item, ok := cloudItsmCatalogItemByKey(c, key)
	if !ok {
		return nil, e.New(e.BadParam, fmt.Errorf("自助目录服务项 %s 不存在", key), http.StatusBadRequest)
	}
	if form.Reset {
		item.PolicyVersion = version
		item.PolicyUpdatedAt = updatedAt
		item.PolicyUpdatedBy = updatedBy
		item.PolicyLastDiff = cloudItsmCatalogPolicyDiffFromAttr(historyEntry["diff"])
	}
	return &item, nil
}

func CloudItsmCatalogPolicyHistory(c *ctx.ServiceContext, form *forms.CloudItsmCatalogPolicyHistoryForm) ([]resps.CloudItsmCatalogPolicyHistoryResp, e.Error) {
	key := strings.TrimSpace(form.Key)
	if _, ok := cloudItsmCatalogItemByKey(nil, key); !ok {
		return nil, e.New(e.BadParam, fmt.Errorf("自助目录服务项 %s 不存在", key), http.StatusBadRequest)
	}
	limit := form.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	entries := cloudItsmCatalogPolicyHistoryConfig(c)[key]
	result := make([]resps.CloudItsmCatalogPolicyHistoryResp, 0, len(entries))
	for _, entry := range entries {
		result = append(result, cloudItsmCatalogPolicyHistoryResp(entry))
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func normalizeCloudItsmSelfServiceParams(requestType string, params models.ResAttrs) (models.ResAttrs, e.Error) {
	if params == nil {
		params = models.ResAttrs{}
	}
	switch strings.TrimSpace(requestType) {
	case models.CloudOperationActionGitOpsIacChange:
		gate, err := cloudItsmGitOpsGateFromParams(params)
		if err != nil {
			return nil, err
		}
		params["gitOpsRequired"] = true
		params["gitOpsGate"] = gate
		params["gateStatus"] = gate["status"]
	case models.CloudOperationActionDriftRemediation:
		params["driftRemediationRequired"] = true
		if params["targetState"] == nil {
			params["targetState"] = "iac_desired_state"
		}
	case models.CloudOperationActionRiskRemediation:
		params["riskRemediationRequired"] = true
		if params["targetState"] == nil {
			params["targetState"] = "risk_remediated_state"
		}
	case models.CloudOperationActionPermissionRequest:
		params["permissionWorkflowRequired"] = true
	}
	return params, nil
}

func cloudItsmGitOpsGateFromParams(params models.ResAttrs) (models.ResAttrs, e.Error) {
	repository := cloudItsmParamString(params, "gitOpsRepository", "gitopsRepository", "repository", "repo")
	branch := cloudItsmParamString(params, "gitOpsBranch", "gitopsBranch", "branch", "sourceBranch")
	targetBranch := cloudItsmParamString(params, "gitOpsTargetBranch", "gitopsTargetBranch", "targetBranch", "baseBranch")
	changePath := cloudItsmParamString(params, "gitOpsChangePath", "gitopsChangePath", "changePath", "iacPath")
	prUrl := cloudItsmParamString(params, "gitOpsPullRequestUrl", "gitopsPullRequestUrl", "pullRequestUrl", "mergeRequestUrl", "prUrl", "mrUrl")
	pipelineUrl := cloudItsmParamString(params, "gitOpsPipelineUrl", "gitopsPipelineUrl", "pipelineUrl", "ciUrl")
	reviewStatus := cloudItsmNormalizeReviewStatus(cloudItsmParamString(params, "gitOpsReviewStatus", "gitopsReviewStatus", "reviewStatus", "prReviewStatus", "mrReviewStatus"))
	pipelineStatus := cloudItsmNormalizePipelineStatus(cloudItsmParamString(params, "gitOpsPipelineStatus", "gitopsPipelineStatus", "pipelineStatus", "ciStatus"))
	if prUrl == "" {
		return nil, e.New(e.BadParam, fmt.Errorf("GitOps/IaC 变更申请必须填写 PR/MR 地址"), http.StatusBadRequest)
	}
	if err := cloudItsmValidateHTTPURL(prUrl, "PR/MR 地址"); err != nil {
		return nil, err
	}
	if pipelineUrl != "" {
		if err := cloudItsmValidateHTTPURL(pipelineUrl, "流水线地址"); err != nil {
			return nil, err
		}
	}
	status := cloudItsmGitOpsGateStatus(reviewStatus, pipelineStatus, pipelineUrl != "")
	message := "PR review 或自动化流水线仍在等待，暂不能视为可执行变更"
	switch status {
	case cloudItsmGitOpsGateStatusPassed:
		message = "PR review 已通过且自动化流水线已通过"
	case cloudItsmGitOpsGateStatusBlocked:
		message = "PR review 或自动化流水线未通过，需修正后重新提交"
	}
	return models.ResAttrs{
		"status":             status,
		"message":            message,
		"required":           true,
		"pullRequestUrl":     prUrl,
		"pullRequestPresent": true,
		"reviewRequired":     true,
		"reviewStatus":       reviewStatus,
		"reviewPassed":       reviewStatus == "approved",
		"pipelineRequired":   true,
		"pipelineUrl":        pipelineUrl,
		"pipelinePresent":    pipelineUrl != "",
		"pipelineStatus":     pipelineStatus,
		"pipelinePassed":     pipelineStatus == "passed",
		"repository":         repository,
		"branch":             branch,
		"targetBranch":       targetBranch,
		"changePath":         changePath,
	}, nil
}

func cloudItsmGitOpsGateStatus(reviewStatus string, pipelineStatus string, pipelinePresent bool) string {
	if reviewStatus == "rejected" || reviewStatus == "changes_requested" || pipelineStatus == "failed" || pipelineStatus == "canceled" {
		return cloudItsmGitOpsGateStatusBlocked
	}
	if reviewStatus == "approved" && pipelineStatus == "passed" && pipelinePresent {
		return cloudItsmGitOpsGateStatusPassed
	}
	return cloudItsmGitOpsGateStatusWaiting
}

func cloudItsmNormalizeReviewStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "approved", "approve", "pass", "passed", "success", "ok":
		return "approved"
	case "changes_requested", "change_requested", "changes-requested", "needs_changes", "need_changes":
		return "changes_requested"
	case "rejected", "reject", "failed", "failure":
		return "rejected"
	case "pending", "open", "waiting", "reviewing", "":
		return "pending"
	default:
		return status
	}
}

func cloudItsmNormalizePipelineStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "passed", "pass", "success", "succeeded", "ok":
		return "passed"
	case "failed", "failure", "error":
		return "failed"
	case "canceled", "cancelled", "aborted":
		return "canceled"
	case "running", "in_progress", "processing":
		return "running"
	case "pending", "queued", "waiting", "":
		return "pending"
	default:
		return status
	}
}

func cloudItsmValidateHTTPURL(rawURL string, label string) e.Error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return e.New(e.BadParam, fmt.Errorf("%s不是有效 URL", label), http.StatusBadRequest)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return e.New(e.BadParam, fmt.Errorf("%s只支持 http/https URL", label), http.StatusBadRequest)
	}
	return nil
}

func cloudItsmParamString(params models.ResAttrs, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(cloudSyncPolicyAttrString(params[key])); value != "" {
			return value
		}
	}
	return ""
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
	if syncErr := syncCloudItsmRiskRemediationStatus(c, &ticket, operation, form); syncErr != nil {
		return nil, syncErr
	}
	resp := cloudItsmTicketResp(c, ticket, nil)
	return &resp, nil
}

func UpdateCloudItsmTicketStatusByCallback(c *ctx.ServiceContext, form *forms.CloudItsmTicketStatusCallbackForm) (*resps.CloudItsmTicketResp, e.Error) {
	cfg := models.CloudItsmConfig{}
	if err := c.DB().Model(&models.CloudItsmConfig{}).Where("id = ?", form.ConnectorId).First(&cfg); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	if cfg.Status != models.CloudItsmConfigStatusEnabled {
		return nil, e.New(e.BadParam, fmt.Errorf("ITSM 连接器 %s 已禁用", cfg.Name), http.StatusBadRequest)
	}
	if err := validateCloudItsmCallbackSignature(cfg, form); err != nil {
		return nil, err
	}
	ticket, err := findCloudItsmCallbackTicket(c, cfg, form)
	if err != nil {
		return nil, err
	}
	c.OrgId = cfg.OrgId
	updateForm := cloudItsmCallbackUpdateStatusForm(ticket.Id, form)
	resp, updateErr := UpdateCloudItsmTicketStatus(c, updateForm)
	if updateErr != nil {
		return nil, updateErr
	}
	if resp != nil {
		resp.ResponsePayload = updateForm.Payload
	}
	return resp, nil
}

func UpdateCloudItsmGitOpsGateByCallback(c *ctx.ServiceContext, form *forms.CloudItsmGitOpsGateCallbackForm) (models.ResAttrs, e.Error) {
	cfg := models.CloudItsmConfig{}
	if err := c.DB().Model(&models.CloudItsmConfig{}).Where("id = ?", form.ConnectorId).First(&cfg); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	if cfg.Status != models.CloudItsmConfigStatusEnabled {
		return nil, e.New(e.BadParam, fmt.Errorf("ITSM 连接器 %s 已禁用", cfg.Name), http.StatusBadRequest)
	}
	if err := validateCloudItsmGitOpsGateCallbackSignature(cfg, form); err != nil {
		return nil, err
	}
	c.OrgId = cfg.OrgId
	ticket, operation, err := findCloudItsmGitOpsGateCallbackTarget(c, cfg, form)
	if err != nil {
		return nil, err
	}
	params, gate, callback, err := cloudItsmGitOpsGateCallbackParams(operation.Params, form, time.Now())
	if err != nil {
		return nil, err
	}
	result := cloudItsmCopyResAttrs(operation.Result)
	result["gitOpsGate"] = gate
	result["gitOpsGateStatus"] = gate["status"]
	result["gitOpsLastCallback"] = callback
	message := fmt.Sprintf("GitOps/IaC 门禁状态已回写：%s", gate["status"])
	if text := strings.TrimSpace(cloudSyncPolicyAttrString(gate["message"])); text != "" {
		message = text
	}
	if _, dbErr := c.DB().Model(&models.CloudOperation{}).
		Where("id = ? and org_id = ?", operation.Id, operation.OrgId).
		UpdateAttrs(models.Attrs{
			"params":  params,
			"result":  result,
			"message": message,
		}); dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}
	operation.Params = params
	operation.Result = result
	operation.Message = message
	if auditErr := recordCloudOperationAudit(c, operation, operation.Status, message, models.ResAttrs{
		"callback": callback,
	}, models.ResAttrs{
		"gitOpsGate": gate,
	}); auditErr != nil {
		return nil, auditErr
	}

	resp := models.ResAttrs{
		"operationId": operation.Id.String(),
		"gate":        gate,
		"callback":    callback,
	}
	if ticket != nil {
		if err := cloudItsmUpdateTicketGitOpsGate(c, ticket, operation, gate, callback, message); err != nil {
			return nil, err
		}
		ticketResp := cloudItsmTicketResp(c, *ticket, nil)
		resp["ticket"] = ticketResp
		resp["ticketId"] = ticket.Id.String()
	}
	return resp, nil
}

func validateCloudItsmCallbackSignature(cfg models.CloudItsmConfig, form *forms.CloudItsmTicketStatusCallbackForm) e.Error {
	secret := cloudItsmCallbackSecret(cfg)
	if secret == "" {
		return e.New(e.PermissionDeny, fmt.Errorf("ITSM callback secret is not configured"), http.StatusForbidden)
	}
	if !cloudItsmCallbackSignatureMatches(secret, form.RawBody, form.Signature) {
		return e.New(e.PermissionDeny, fmt.Errorf("invalid ITSM callback signature"), http.StatusForbidden)
	}
	return nil
}

func validateCloudItsmGitOpsGateCallbackSignature(cfg models.CloudItsmConfig, form *forms.CloudItsmGitOpsGateCallbackForm) e.Error {
	secret := cloudItsmGitOpsGateCallbackSecret(cfg)
	if secret == "" {
		return e.New(e.PermissionDeny, fmt.Errorf("GitOps/IaC gate callback secret is not configured"), http.StatusForbidden)
	}
	if !cloudItsmCallbackSignatureMatches(secret, form.RawBody, form.Signature) {
		return e.New(e.PermissionDeny, fmt.Errorf("invalid GitOps/IaC gate callback signature"), http.StatusForbidden)
	}
	return nil
}

func findCloudItsmGitOpsGateCallbackTarget(c *ctx.ServiceContext, cfg models.CloudItsmConfig, form *forms.CloudItsmGitOpsGateCallbackForm) (*models.CloudItsmTicket, *models.CloudOperation, e.Error) {
	if form.TicketId != "" {
		ticket := models.CloudItsmTicket{}
		if err := c.DB().Model(&models.CloudItsmTicket{}).
			Where("org_id = ? and connector_id = ? and id = ?", cfg.OrgId, cfg.Id, form.TicketId).
			First(&ticket); err != nil {
			if e.IsRecordNotFound(err) {
				return nil, nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
			}
			return nil, nil, e.New(e.DBError, err)
		}
		operation, err := getCloudItsmGitOpsOperation(c, cfg.OrgId, ticket.OperationId)
		return &ticket, operation, err
	}
	if form.OperationId != "" {
		operation, err := getCloudItsmGitOpsOperation(c, cfg.OrgId, form.OperationId)
		if err != nil {
			return nil, nil, err
		}
		ticket, err := findCloudItsmTicketByOperation(c, cfg, operation.Id)
		if err != nil && err.Code() != e.ObjectNotExistsOrNoPerm {
			return nil, nil, err
		}
		return ticket, operation, nil
	}
	if strings.TrimSpace(form.PullRequestUrl) != "" {
		return findCloudItsmGitOpsTargetByPullRequest(c, cfg, form.PullRequestUrl)
	}
	return nil, nil, e.New(e.BadParam, fmt.Errorf("operationId、ticketId 或 pullRequestUrl 至少需要填写一个"), http.StatusBadRequest)
}

func getCloudItsmGitOpsOperation(c *ctx.ServiceContext, orgId models.Id, operationId models.Id) (*models.CloudOperation, e.Error) {
	if operationId == "" {
		return nil, e.New(e.BadParam, fmt.Errorf("GitOps/IaC 变更操作任务不存在"), http.StatusBadRequest)
	}
	operation := models.CloudOperation{}
	if err := c.DB().Model(&models.CloudOperation{}).
		Where("id = ? and org_id = ?", operationId, orgId).
		First(&operation); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	if operation.OperationType != models.CloudOperationTypeSelfService || operation.Action != models.CloudOperationActionGitOpsIacChange {
		return nil, e.New(e.BadParam, fmt.Errorf("操作任务 %s 不是 GitOps/IaC 变更申请", operation.Id), http.StatusBadRequest)
	}
	return &operation, nil
}

func findCloudItsmTicketByOperation(c *ctx.ServiceContext, cfg models.CloudItsmConfig, operationId models.Id) (*models.CloudItsmTicket, e.Error) {
	ticket := models.CloudItsmTicket{}
	if err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and connector_id = ? and operation_id = ?", cfg.OrgId, cfg.Id, operationId).
		Order("updated_at desc").
		First(&ticket); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	return &ticket, nil
}

func findCloudItsmGitOpsTargetByPullRequest(c *ctx.ServiceContext, cfg models.CloudItsmConfig, pullRequestUrl string) (*models.CloudItsmTicket, *models.CloudOperation, e.Error) {
	tickets := make([]models.CloudItsmTicket, 0)
	if err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and connector_id = ?", cfg.OrgId, cfg.Id).
		Order("updated_at desc").
		Limit(200).
		Find(&tickets); err != nil {
		return nil, nil, e.New(e.DBError, err)
	}
	for idx := range tickets {
		ticket := &tickets[idx]
		if !cloudItsmGitOpsTicketMatchesPullRequest(*ticket, pullRequestUrl) {
			continue
		}
		operation, err := getCloudItsmGitOpsOperation(c, cfg.OrgId, ticket.OperationId)
		if err != nil {
			return nil, nil, err
		}
		return ticket, operation, nil
	}
	return nil, nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
}

func cloudItsmGitOpsTicketMatchesPullRequest(ticket models.CloudItsmTicket, pullRequestUrl string) bool {
	want := strings.TrimSpace(pullRequestUrl)
	if want == "" {
		return false
	}
	payload := ticket.RequestPayload
	operationPayload := modelResAttrs(payload["operation"])
	params := modelResAttrs(operationPayload["params"])
	if got := cloudItsmGitOpsPullRequestUrlFromParams(params); got != "" && got == want {
		return true
	}
	gate := modelResAttrs(params["gitOpsGate"])
	if got := strings.TrimSpace(cloudSyncPolicyAttrString(gate["pullRequestUrl"])); got != "" && got == want {
		return true
	}
	return false
}

func cloudItsmGitOpsGateCallbackParams(params models.ResAttrs, form *forms.CloudItsmGitOpsGateCallbackForm, now time.Time) (models.ResAttrs, models.ResAttrs, models.ResAttrs, e.Error) {
	next := cloudItsmCopyResAttrs(params)
	cloudItsmSeedGitOpsParamsFromGate(next)
	cloudItsmSetStringParam(next, "gitOpsRepository", form.Repository)
	cloudItsmSetStringParam(next, "gitOpsBranch", form.Branch)
	cloudItsmSetStringParam(next, "gitOpsTargetBranch", form.TargetBranch)
	cloudItsmSetStringParam(next, "gitOpsChangePath", form.ChangePath)
	cloudItsmSetStringParam(next, "gitOpsPullRequestUrl", form.PullRequestUrl)
	if strings.TrimSpace(form.ReviewStatus) != "" {
		next["gitOpsReviewStatus"] = cloudItsmNormalizeReviewStatus(form.ReviewStatus)
	}
	cloudItsmSetStringParam(next, "gitOpsPipelineUrl", form.PipelineUrl)
	if strings.TrimSpace(form.PipelineStatus) != "" {
		next["gitOpsPipelineStatus"] = cloudItsmNormalizePipelineStatus(form.PipelineStatus)
	}
	gate, err := cloudItsmGitOpsGateFromParams(next)
	if err != nil {
		return nil, nil, nil, err
	}
	gate["updatedAt"] = now.Format(time.RFC3339)
	gate["updatedByCallback"] = true
	next["gitOpsRequired"] = true
	next["gitOpsGate"] = gate
	next["gateStatus"] = gate["status"]
	next["gitOpsGateUpdatedAt"] = gate["updatedAt"]
	callback := cloudItsmGitOpsGateCallbackSnapshot(form, gate, now)
	next["gitOpsLastCallback"] = callback
	next["gitOpsCallbackCount"] = attrInt(next, "gitOpsCallbackCount") + 1
	return next, gate, callback, nil
}

func cloudItsmSeedGitOpsParamsFromGate(params models.ResAttrs) {
	if params == nil {
		return
	}
	gate := modelResAttrs(params["gitOpsGate"])
	if len(gate) == 0 {
		return
	}
	seed := map[string]string{
		"gitOpsRepository":     "repository",
		"gitOpsBranch":         "branch",
		"gitOpsTargetBranch":   "targetBranch",
		"gitOpsChangePath":     "changePath",
		"gitOpsPullRequestUrl": "pullRequestUrl",
		"gitOpsReviewStatus":   "reviewStatus",
		"gitOpsPipelineUrl":    "pipelineUrl",
		"gitOpsPipelineStatus": "pipelineStatus",
	}
	for target, source := range seed {
		if strings.TrimSpace(cloudSyncPolicyAttrString(params[target])) == "" {
			cloudItsmSetStringParam(params, target, cloudSyncPolicyAttrString(gate[source]))
		}
	}
}

func cloudItsmGitOpsGateCallbackSnapshot(form *forms.CloudItsmGitOpsGateCallbackForm, gate models.ResAttrs, now time.Time) models.ResAttrs {
	callback := models.ResAttrs{
		"connectorId":        form.ConnectorId.String(),
		"signatureVerified":  true,
		"signatureAlgorithm": cloudItsmCallbackSignatureAlgorithm,
		"signatureVersion":   strings.TrimSpace(form.SignatureVersion),
		"receivedAt":         now.Format(time.RFC3339),
		"gateStatus":         gate["status"],
		"reviewStatus":       gate["reviewStatus"],
		"pipelineStatus":     gate["pipelineStatus"],
		"pullRequestUrl":     gate["pullRequestUrl"],
		"pipelineUrl":        gate["pipelineUrl"],
	}
	cloudItsmSetStringParam(callback, "operationId", form.OperationId.String())
	cloudItsmSetStringParam(callback, "ticketId", form.TicketId.String())
	cloudItsmSetStringParam(callback, "commitSha", form.CommitSha)
	cloudItsmSetStringParam(callback, "externalRunId", form.ExternalRunId)
	cloudItsmSetStringParam(callback, "comment", form.Comment)
	if len(form.Payload) > 0 {
		callback["payload"] = form.Payload
	}
	return callback
}

func cloudItsmUpdateTicketGitOpsGate(c *ctx.ServiceContext, ticket *models.CloudItsmTicket, operation *models.CloudOperation, gate models.ResAttrs, callback models.ResAttrs, message string) e.Error {
	requestPayload := cloudItsmCopyResAttrs(ticket.RequestPayload)
	operationPayload := modelResAttrs(requestPayload["operation"])
	if operationPayload == nil {
		operationPayload = models.ResAttrs{}
	}
	operationPayload["params"] = operation.Params
	operationPayload["result"] = operation.Result
	operationPayload["message"] = operation.Message
	requestPayload["operation"] = operationPayload
	requestPayload["gitOpsGate"] = gate
	requestPayload["gitOpsGateCallback"] = callback
	responsePayload := cloudItsmCopyResAttrs(ticket.ResponsePayload)
	responsePayload["gitOpsGate"] = gate
	responsePayload["gitOpsGateCallback"] = callback
	now := models.Time(time.Now())
	attrs := models.Attrs{
		"request_payload":  requestPayload,
		"response_payload": responsePayload,
		"last_synced_at":   now,
	}
	if _, err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("id = ? and org_id = ?", ticket.Id, ticket.OrgId).
		UpdateAttrs(attrs); err != nil {
		return e.New(e.DBError, err)
	}
	ticket.RequestPayload = requestPayload
	ticket.ResponsePayload = responsePayload
	ticket.LastSyncedAt = now
	eventPayload := models.ResAttrs{
		"operationId": operation.Id.String(),
		"gitOpsGate":  gate,
		"callback":    callback,
	}
	eventId := cloudItsmTicketEvent(c, ticket, operation, "itsm.gitops_gate.updated", cloudItsmGitOpsGateEventLevel(gate), "GitOps/IaC 门禁状态已回写", message, eventPayload)
	if _, err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("id = ? and org_id = ?", ticket.Id, ticket.OrgId).
		UpdateAttrs(models.Attrs{"cloud_event_id": eventId}); err != nil {
		return e.New(e.DBError, err)
	}
	ticket.CloudEventId = eventId
	return nil
}

func cloudItsmGitOpsGateEventLevel(gate models.ResAttrs) string {
	switch cloudSyncPolicyAttrString(gate["status"]) {
	case cloudItsmGitOpsGateStatusBlocked:
		return models.CloudEventLevelError
	case cloudItsmGitOpsGateStatusWaiting:
		return models.CloudEventLevelWarning
	default:
		return models.CloudEventLevelInfo
	}
}

func cloudItsmCopyResAttrs(attrs models.ResAttrs) models.ResAttrs {
	if len(attrs) == 0 {
		return models.ResAttrs{}
	}
	body, err := json.Marshal(attrs)
	if err != nil {
		next := models.ResAttrs{}
		for key, value := range attrs {
			next[key] = value
		}
		return next
	}
	next := models.ResAttrs{}
	if err := json.Unmarshal(body, &next); err != nil {
		next = models.ResAttrs{}
		for key, value := range attrs {
			next[key] = value
		}
	}
	return next
}

func cloudItsmSetStringParam(attrs models.ResAttrs, key string, value string) {
	if attrs == nil {
		return
	}
	if value = strings.TrimSpace(value); value != "" {
		attrs[key] = value
	}
}

func cloudItsmGitOpsPullRequestUrlFromParams(params models.ResAttrs) string {
	return cloudItsmParamString(params, "gitOpsPullRequestUrl", "gitopsPullRequestUrl", "pullRequestUrl", "mergeRequestUrl", "prUrl", "mrUrl")
}

func findCloudItsmCallbackTicket(c *ctx.ServiceContext, cfg models.CloudItsmConfig, form *forms.CloudItsmTicketStatusCallbackForm) (*models.CloudItsmTicket, e.Error) {
	query := c.DB().Model(&models.CloudItsmTicket{}).Where("org_id = ? and connector_id = ?", cfg.OrgId, cfg.Id)
	switch {
	case form.TicketId != "":
		query = query.Where("id = ?", form.TicketId)
	case strings.TrimSpace(form.ExternalId) != "" && strings.TrimSpace(form.ExternalKey) != "":
		query = query.Where("(external_id = ? and external_id <> '') or (external_key = ? and external_key <> '')", strings.TrimSpace(form.ExternalId), strings.TrimSpace(form.ExternalKey))
	case strings.TrimSpace(form.ExternalId) != "":
		query = query.Where("external_id = ? and external_id <> ''", strings.TrimSpace(form.ExternalId))
	case strings.TrimSpace(form.ExternalKey) != "":
		query = query.Where("external_key = ? and external_key <> ''", strings.TrimSpace(form.ExternalKey))
	default:
		return nil, e.New(e.BadParam, fmt.Errorf("ticketId, externalId or externalKey is required"), http.StatusBadRequest)
	}
	ticket := models.CloudItsmTicket{}
	if err := query.First(&ticket); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	return &ticket, nil
}

func cloudItsmCallbackUpdateStatusForm(ticketId models.Id, form *forms.CloudItsmTicketStatusCallbackForm) *forms.UpdateCloudItsmTicketStatusForm {
	payload := models.ResAttrs{}
	for key, value := range form.Payload {
		payload[key] = value
	}
	payload["callback"] = models.ResAttrs{
		"connectorId":        form.ConnectorId.String(),
		"signatureVerified":  true,
		"signatureAlgorithm": cloudItsmCallbackSignatureAlgorithm,
		"signatureVersion":   strings.TrimSpace(form.SignatureVersion),
		"receivedAt":         time.Now().Format(time.RFC3339),
	}
	updateForm := &forms.UpdateCloudItsmTicketStatusForm{
		Id:          ticketId,
		Status:      form.Status,
		ExternalId:  form.ExternalId,
		ExternalKey: form.ExternalKey,
		ExternalUrl: form.ExternalUrl,
		Comment:     form.Comment,
		Payload:     payload,
	}
	values := url.Values{}
	values.Set("id", ticketId.String())
	values.Set("status", form.Status)
	if form.HasKey("externalId") {
		values.Set("externalId", form.ExternalId)
	}
	if form.HasKey("externalKey") {
		values.Set("externalKey", form.ExternalKey)
	}
	if form.HasKey("externalUrl") {
		values.Set("externalUrl", form.ExternalUrl)
	}
	values.Set("comment", form.Comment)
	values.Set("payload", "true")
	updateForm.Bind(values)
	return updateForm
}

func cloudItsmCallbackSecret(cfg models.CloudItsmConfig) string {
	for _, key := range []string{"callbackSecret", "statusCallbackSecret", "incomingSecret", "webhookSecret", "signatureSecret"} {
		if value := strings.TrimSpace(attrString(cfg.Metadata, key)); value != "" {
			return value
		}
	}
	return ""
}

func cloudItsmGitOpsGateCallbackSecret(cfg models.CloudItsmConfig) string {
	for _, key := range []string{"gitOpsGateCallbackSecret", "gitOpsCallbackSecret", "vcsCallbackSecret", "ciCallbackSecret"} {
		if value := strings.TrimSpace(attrString(cfg.Metadata, key)); value != "" {
			return value
		}
	}
	return cloudItsmCallbackSecret(cfg)
}

func cloudItsmCallbackSignatureMatches(secret string, body []byte, signature string) bool {
	provided := strings.TrimSpace(signature)
	if strings.TrimSpace(secret) == "" || provided == "" {
		return false
	}
	expected := cloudwebhooksrv.Signature(secret, body)
	if hmac.Equal([]byte(expected), []byte(provided)) {
		return true
	}
	if strings.HasPrefix(expected, "sha256=") && !strings.Contains(provided, "=") {
		return hmac.Equal([]byte(strings.TrimPrefix(expected, "sha256=")), []byte(provided))
	}
	return false
}

func SyncDueCloudItsmTicketStatuses(c *ctx.ServiceContext, form *forms.SyncDueCloudItsmTicketStatusForm) (models.ResAttrs, e.Error) {
	if form == nil {
		form = &forms.SyncDueCloudItsmTicketStatusForm{}
	}
	limit := form.Limit
	if limit <= 0 {
		limit = cloudItsmStatusSyncDefaultLimit
	}
	if limit > cloudItsmStatusSyncMaxLimit {
		limit = cloudItsmStatusSyncMaxLimit
	}
	result := cloudItsmStatusSyncResult()
	configs := make([]models.CloudItsmConfig, 0)
	query := c.DB().Model(&models.CloudItsmConfig{}).Where("org_id = ? and status = ?", c.OrgId, models.CloudItsmConfigStatusEnabled)
	if form.ConnectorId != "" {
		query = query.Where("id = ?", form.ConnectorId)
	}
	if err := query.Order("updated_at desc").Find(&configs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	result["connectorCount"] = len(configs)
	remaining := limit
	for idx := range configs {
		if remaining <= 0 {
			break
		}
		cfg := &configs[idx]
		if !cloudItsmStatusSyncEnabled(*cfg) {
			cloudItsmStatusSyncAdd(result, "connectorSkipped", 1)
			continue
		}
		syncResult, err := syncCloudItsmConnectorTicketStatuses(c, cfg, remaining, form.Force)
		if err != nil {
			return nil, err
		}
		remaining -= cloudItsmStatusSyncCount(syncResult, "total")
		cloudItsmStatusSyncMerge(result, syncResult)
	}
	return result, nil
}

func SyncDueCloudItsmTicketStatusesForAllOrgs() (models.ResAttrs, e.Error) {
	configs := make([]models.CloudItsmConfig, 0)
	if err := db.Get().Model(&models.CloudItsmConfig{}).
		Where("status = ?", models.CloudItsmConfigStatusEnabled).
		Order("org_id asc, updated_at desc").
		Find(&configs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	result := cloudItsmStatusSyncResult()
	result["connectorCount"] = len(configs)
	seenOrgs := map[models.Id]bool{}
	for idx := range configs {
		cfg := &configs[idx]
		if cfg.OrgId == "" {
			cloudItsmStatusSyncAdd(result, "skipped", 1)
			continue
		}
		if !cloudItsmStatusSyncEnabled(*cfg) {
			cloudItsmStatusSyncAdd(result, "connectorSkipped", 1)
			continue
		}
		if !seenOrgs[cfg.OrgId] {
			seenOrgs[cfg.OrgId] = true
			cloudItsmStatusSyncAdd(result, "orgCount", 1)
		}
		locked, releaseLock, lockErr := cloudWebhookAcquireMysqlLock(cloudItsmStatusSyncLockName(cfg.OrgId))
		if lockErr != nil {
			return nil, e.New(e.DBError, lockErr)
		}
		if !locked {
			cloudItsmStatusSyncAdd(result, "lockSkipped", 1)
			continue
		}
		item, err := func() (models.ResAttrs, e.Error) {
			defer releaseLock()
			workerCtx := &ctx.ServiceContext{
				UserId:   consts.SysUserId,
				OrgId:    cfg.OrgId,
				Email:    consts.DefaultSysEmail,
				Username: consts.DefaultSysName,
			}
			return syncCloudItsmConnectorTicketStatuses(workerCtx, cfg, cloudItsmStatusSyncDefaultLimit, false)
		}()
		if err != nil {
			return nil, err
		}
		cloudItsmStatusSyncMerge(result, item)
	}
	return result, nil
}

func StartCloudItsmStatusSyncWorker(serviceId string) {
	logger := logs.Get().
		WithField("worker", "cloudItsmStatusSync").
		WithField("serviceId", serviceId).
		WithField("interval", cloudItsmStatusSyncWorkerInterval.String())
	ticker := time.NewTicker(cloudItsmStatusSyncWorkerInterval)
	defer ticker.Stop()
	for {
		if result, err := SyncDueCloudItsmTicketStatusesForAllOrgs(); err != nil {
			logger.Warnf("sync due cloud ITSM ticket statuses failed: %v", err)
		} else if cloudItsmStatusSyncCount(result, "synced") > 0 ||
			cloudItsmStatusSyncCount(result, "failed") > 0 ||
			cloudItsmStatusSyncCount(result, "lockSkipped") > 0 {
			logger.Infof("sync due cloud ITSM ticket statuses result: %+v", result)
		}
		<-ticker.C
	}
}

func RetryDueCloudItsmTicketSubmissions(c *ctx.ServiceContext, form *forms.RetryDueCloudItsmTicketSubmitForm) (models.ResAttrs, e.Error) {
	if form == nil {
		form = &forms.RetryDueCloudItsmTicketSubmitForm{}
	}
	limit := form.Limit
	if limit <= 0 {
		limit = cloudItsmSubmitRetryDefaultLimit
	}
	if limit > cloudItsmSubmitRetryMaxLimit {
		limit = cloudItsmSubmitRetryMaxLimit
	}
	result := cloudItsmSubmitRetryResult()
	configs := make([]models.CloudItsmConfig, 0)
	query := c.DB().Model(&models.CloudItsmConfig{}).Where("org_id = ? and status = ?", c.OrgId, models.CloudItsmConfigStatusEnabled)
	if form.ConnectorId != "" {
		query = query.Where("id = ?", form.ConnectorId)
	}
	if err := query.Order("updated_at desc").Find(&configs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	result["connectorCount"] = len(configs)
	remaining := limit
	for idx := range configs {
		if remaining <= 0 {
			break
		}
		cfg := &configs[idx]
		if !cloudItsmSubmitRetryEnabled(*cfg) {
			cloudItsmSubmitRetryAdd(result, "connectorSkipped", 1)
			continue
		}
		retryResult, err := retryCloudItsmConnectorTicketSubmissions(c, cfg, remaining, form.Force)
		if err != nil {
			return nil, err
		}
		remaining -= cloudItsmSubmitRetryCount(retryResult, "total")
		cloudItsmSubmitRetryMerge(result, retryResult)
	}
	return result, nil
}

func CloudItsmSubmitRetryQueueSummary(c *ctx.ServiceContext) (models.ResAttrs, e.Error) {
	configs, err := cloudItsmSubmitRetryConfigMap(c)
	if err != nil {
		return nil, err
	}
	tickets := make([]models.CloudItsmTicket, 0)
	if err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and status = ?", c.OrgId, models.CloudItsmTicketStatusFailed).
		Order("updated_at desc").
		Find(&tickets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	now := time.Now()
	summary := models.ResAttrs{
		"failedTotal":    len(tickets),
		"connectorCount": len(configs),
		"due":            0,
		"future":         0,
		"deadLetter":     0,
		"skipped":        0,
		"retryable":      0,
		"reasons":        models.ResAttrs{},
	}
	var oldestDue time.Time
	var oldestFuture time.Time
	for idx := range tickets {
		item := cloudItsmSubmitRetryQueueItem(c, tickets[idx], configs[tickets[idx].ConnectorId], now)
		cloudItsmSubmitRetryQueueSummaryAdd(summary, item.QueueStatus, item.RetryReason)
		nextRetryAt := cloudItsmSubmitRetryQueueNextRetryAt(item)
		if item.QueueStatus == "due" && !nextRetryAt.IsZero() && (oldestDue.IsZero() || nextRetryAt.Before(oldestDue)) {
			oldestDue = nextRetryAt
		}
		if item.QueueStatus == "future" && !nextRetryAt.IsZero() && (oldestFuture.IsZero() || nextRetryAt.Before(oldestFuture)) {
			oldestFuture = nextRetryAt
		}
	}
	summary["retryable"] = cloudItsmStatusSyncCount(summary, "due") + cloudItsmStatusSyncCount(summary, "future")
	if !oldestDue.IsZero() {
		summary["oldestDueAt"] = oldestDue.Format(time.RFC3339)
	}
	if !oldestFuture.IsZero() {
		summary["nextRetryAt"] = oldestFuture.Format(time.RFC3339)
	}
	return summary, nil
}

func CloudItsmSubmitRetryQueueReport(c *ctx.ServiceContext) (*resps.CloudItsmSubmitRetryQueueReportResp, e.Error) {
	configs, err := cloudItsmSubmitRetryConfigMap(c)
	if err != nil {
		return nil, err
	}
	tickets := make([]models.CloudItsmTicket, 0)
	if err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and status = ?", c.OrgId, models.CloudItsmTicketStatusFailed).
		Order("updated_at desc").
		Find(&tickets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return cloudItsmSubmitRetryQueueReport(c, tickets, configs, time.Now()), nil
}

func SearchCloudItsmSubmitRetryQueue(c *ctx.ServiceContext, form *forms.SearchCloudItsmSubmitRetryQueueForm) (interface{}, e.Error) {
	if form == nil {
		form = &forms.SearchCloudItsmSubmitRetryQueueForm{}
	}
	configs, err := cloudItsmSubmitRetryConfigMap(c)
	if err != nil {
		return nil, err
	}
	query := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and status = ?", c.OrgId, models.CloudItsmTicketStatusFailed)
	if form.ConnectorId != "" {
		query = query.Where("connector_id = ?", form.ConnectorId)
	}
	if strings.TrimSpace(form.Q) != "" {
		q := "%" + strings.TrimSpace(form.Q) + "%"
		query = query.Where("(title like ? or external_id like ? or external_key like ? or error_message like ?)", q, q, q, q)
	}
	tickets := make([]models.CloudItsmTicket, 0)
	if err := query.Order("updated_at desc").Find(&tickets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	now := time.Now()
	items := make([]resps.CloudItsmSubmitRetryQueueItemResp, 0, len(tickets))
	for idx := range tickets {
		item := cloudItsmSubmitRetryQueueItem(c, tickets[idx], configs[tickets[idx].ConnectorId], now)
		if form.QueueStatus != "" && item.QueueStatus != form.QueueStatus {
			continue
		}
		items = append(items, item)
	}
	total := len(items)
	pageSize := form.PageSize()
	start := (form.CurrentPage() - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return page.PageResp{
		Total:    int64(total),
		PageSize: pageSize,
		List:     items[start:end],
	}, nil
}

func ReplayCloudItsmTicketSubmission(c *ctx.ServiceContext, form *forms.ReplayCloudItsmTicketSubmitForm) (models.ResAttrs, e.Error) {
	ticket := models.CloudItsmTicket{}
	if err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and id = ?", c.OrgId, form.Id).
		First(&ticket); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	cfg, err := getCloudItsmConfig(c, ticket.ConnectorId)
	if err != nil {
		return nil, err
	}
	if cfg.Status != models.CloudItsmConfigStatusEnabled {
		return nil, e.New(e.BadParam, fmt.Errorf("ITSM 连接器 %s 已禁用", cfg.Name), http.StatusBadRequest)
	}
	result, err := retryCloudItsmSingleTicketSubmission(c, cfg, &ticket, time.Now(), form.Force, true)
	if err != nil {
		return nil, err
	}
	resp := cloudItsmTicketResp(c, ticket, nil)
	result["ticket"] = resp
	return result, nil
}

func CreateCloudItsmDeadLetterApproval(c *ctx.ServiceContext, form *forms.CreateCloudItsmDeadLetterApprovalForm) (*resps.CloudOperationDetailResp, e.Error) {
	if form == nil || len(form.Ids) == 0 {
		return nil, e.New(e.BadParam, fmt.Errorf("请选择死信工单"), http.StatusBadRequest)
	}
	if len(form.Ids) > 50 {
		return nil, e.New(e.BadParam, fmt.Errorf("单次最多处理 50 条死信"), http.StatusBadRequest)
	}
	action := strings.TrimSpace(form.Action)
	if action != "replay" && action != "close" {
		return nil, e.New(e.BadParam, fmt.Errorf("不支持的死信操作 %s", action), http.StatusBadRequest)
	}
	reason := strings.TrimSpace(form.Reason)
	if reason == "" {
		return nil, e.New(e.BadParam, fmt.Errorf("请输入审批申请原因"), http.StatusBadRequest)
	}

	tickets, precheck, err := cloudItsmDeadLetterApprovalTickets(c, form.Ids)
	if err != nil {
		return nil, err
	}
	if errors := cloudItsmStatusSyncErrors(precheck); len(errors) > 0 {
		return nil, e.New(e.BadParam, fmt.Errorf("存在不符合审批条件的死信工单：%s", cloudItsmDeadLetterApprovalErrorSummary(errors)), http.StatusBadRequest)
	}

	approvalId := models.NewId("cpa")
	ticketIds := cloudItsmDeadLetterApprovalTicketIds(tickets)
	projectId, envId := cloudItsmDeadLetterApprovalScope(tickets)
	requestedAt := time.Now().Format(time.RFC3339)
	actionLabel := cloudItsmDeadLetterActionLabel(action)
	evidence, err := cloudItsmDeadLetterApprovalEvidence(form, tickets, precheck, action, actionLabel, reason, c.UserId.String(), requestedAt)
	if err != nil {
		return nil, err
	}
	notificationStrategy := cloudItsmDeadLetterApprovalNotificationStrategy(action, len(ticketIds), evidence)
	riskLevel := models.CloudOperationRiskMedium
	if action == "replay" {
		riskLevel = models.CloudOperationRiskHigh
	}
	params := models.ResAttrs{
		"approval": models.ResAttrs{
			"required":    true,
			"approvalId":  approvalId.String(),
			"status":      "pending",
			"requestedBy": c.UserId.String(),
			"requestedAt": requestedAt,
		},
		"itsmDeadLetter": models.ResAttrs{
			"action":               action,
			"actionLabel":          actionLabel,
			"ticketIds":            ticketIds,
			"ticketCount":          len(ticketIds),
			"force":                form.Force,
			"reason":               reason,
			"evidence":             evidence,
			"requestedBy":          c.UserId.String(),
			"requestedAt":          requestedAt,
			"precheck":             precheck,
			"notificationStrategy": notificationStrategy,
		},
		"notificationStrategy": notificationStrategy,
		"operationSource":      "cloud_itsm_dead_letter",
	}
	operation := &models.CloudOperation{
		OrgId:         c.OrgId,
		ProjectId:     projectId,
		EnvId:         envId,
		CreatorId:     c.UserId,
		ApprovalId:    approvalId,
		Name:          fmt.Sprintf("ITSM 死信%s审批（%d 条）", actionLabel, len(ticketIds)),
		OperationType: models.CloudOperationTypeSelfService,
		Action:        models.CloudOperationActionItsmDeadLetter,
		Status:        models.CloudOperationStatusApproving,
		RiskLevel:     riskLevel,
		ResourceType:  "cloud_itsm_ticket",
		ResourceId:    cloudItsmDeadLetterApprovalResourceId(ticketIds),
		ResourceName:  fmt.Sprintf("ITSM 死信工单 %d 条", len(ticketIds)),
		Message:       "ITSM 死信处置审批已创建，等待审批",
		Params:        params,
	}
	if err := createCloudOperation(c, operation, "创建 ITSM 死信处置审批"); err != nil {
		return nil, err
	}
	cloudItsmDeadLetterApprovalEvent(c, operation, "itsm.dead_letter.approval_requested", models.CloudEventLevelWarning, "ITSM 死信处置审批已提交", fmt.Sprintf("%s %d 条死信工单等待审批", actionLabel, len(ticketIds)), params)
	return CloudOperationDetail(c, &forms.CloudOperationParam{Id: operation.Id})
}

func approveCloudItsmDeadLetterOperation(c *ctx.ServiceContext, operation *models.CloudOperation, form *forms.CloudOperationApprovalForm, approval models.ResAttrs) (*resps.CloudOperationDetailResp, e.Error) {
	payload := modelResAttrs(operation.Params["itsmDeadLetter"])
	action := strings.TrimSpace(cloudSyncPolicyAttrString(payload["action"]))
	actionLabel := cloudItsmDeadLetterActionLabel(action)
	ticketIds := cloudItsmDeadLetterApprovalIdsFromPayload(payload["ticketIds"])
	if action != "replay" && action != "close" {
		return nil, e.New(e.BadParam, fmt.Errorf("操作任务 %s 缺少有效死信处置动作", operation.Id), http.StatusBadRequest)
	}
	if len(ticketIds) == 0 {
		return nil, e.New(e.BadParam, fmt.Errorf("操作任务 %s 缺少死信工单 ID", operation.Id), http.StatusBadRequest)
	}
	if form.Action == "rejected" {
		result := models.ResAttrs{
			"approval":       approval,
			"itsmDeadLetter": payload,
		}
		if err := finishCloudOperation(c, operation, models.CloudOperationStatusRejected, "ITSM 死信处置审批驳回", result); err != nil {
			return nil, err
		}
		cloudItsmDeadLetterApprovalEvent(c, operation, "itsm.dead_letter.approval_rejected", models.CloudEventLevelWarning, "ITSM 死信处置审批驳回", fmt.Sprintf("%s %d 条死信工单被驳回", actionLabel, len(ticketIds)), result)
		return CloudOperationDetail(c, &forms.CloudOperationParam{Id: operation.Id})
	}
	if err := markCloudItsmDeadLetterOperationApproved(c, operation, approval); err != nil {
		return nil, err
	}
	if err := startCloudOperation(c, operation, "审批通过，开始执行 ITSM 死信处置"); err != nil {
		return nil, err
	}
	result, actionErr := BatchCloudItsmDeadLetterAction(c, &forms.BatchCloudItsmDeadLetterActionForm{
		Ids:    ticketIds,
		Action: action,
		Force:  action == "replay" || attrBool(payload, "force"),
		Reason: strings.TrimSpace(cloudSyncPolicyAttrString(payload["reason"])),
	})
	if result == nil {
		result = models.ResAttrs{}
	}
	result["approval"] = approval
	result["itsmDeadLetter"] = payload
	if actionErr != nil {
		result["error"] = actionErr.Error()
		if finishErr := finishCloudOperation(c, operation, models.CloudOperationStatusFailed, fmt.Sprintf("ITSM 死信%s失败：%s", actionLabel, actionErr.Error()), result); finishErr != nil {
			return nil, finishErr
		}
		cloudItsmDeadLetterApprovalEvent(c, operation, "itsm.dead_letter.approval_execute_failed", models.CloudEventLevelError, "ITSM 死信审批处置失败", actionErr.Error(), result)
		return nil, actionErr
	}
	if err := finishCloudOperation(c, operation, models.CloudOperationStatusComplete, fmt.Sprintf("ITSM 死信%s完成", actionLabel), result); err != nil {
		return nil, err
	}
	cloudItsmDeadLetterApprovalEvent(c, operation, "itsm.dead_letter.approval_executed", models.CloudEventLevelInfo, "ITSM 死信审批处置完成", fmt.Sprintf("%s %d 条死信工单完成", actionLabel, len(ticketIds)), result)
	return CloudOperationDetail(c, &forms.CloudOperationParam{Id: operation.Id})
}

func markCloudItsmDeadLetterOperationApproved(c *ctx.ServiceContext, operation *models.CloudOperation, approval models.ResAttrs) e.Error {
	params := operation.Params
	if params == nil {
		params = models.ResAttrs{}
	}
	params["approval"] = approval
	attrs := models.Attrs{
		"status":  models.CloudOperationStatusPending,
		"message": "ITSM 死信处置审批通过，等待执行",
		"params":  params,
	}
	if approvalId := attrString(approval, "approvalId"); approvalId != "" {
		attrs["approval_id"] = approvalId
		operation.ApprovalId = models.Id(approvalId)
	}
	affected, err := c.DB().Model(&models.CloudOperation{}).
		Where("id = ? and org_id = ?", operation.Id, operation.OrgId).
		Where("status = ?", models.CloudOperationStatusApproving).
		UpdateAttrs(attrs)
	if err != nil {
		return e.New(e.DBError, err)
	}
	if affected == 0 {
		return e.New(e.BadParam, fmt.Errorf("操作任务 %s 当前状态不是待审批，不能审批", operation.Id))
	}
	operation.Status = models.CloudOperationStatusPending
	operation.Message = "ITSM 死信处置审批通过，等待执行"
	operation.Params = params
	return recordCloudOperationAudit(c, operation, models.CloudOperationStatusPending, "ITSM 死信处置审批通过", params, models.ResAttrs{
		"approval": approval,
	})
}

func cloudItsmDeadLetterApprovalTickets(c *ctx.ServiceContext, ids []models.Id) ([]models.CloudItsmTicket, models.ResAttrs, e.Error) {
	result := cloudItsmDeadLetterActionResult("approval_precheck", len(ids))
	tickets := make([]models.CloudItsmTicket, 0, len(ids))
	if err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and id in (?)", c.OrgId, ids).
		Find(&tickets); err != nil {
		return nil, nil, e.New(e.DBError, err)
	}
	ticketMap := map[models.Id]*models.CloudItsmTicket{}
	for idx := range tickets {
		ticketMap[tickets[idx].Id] = &tickets[idx]
	}
	configs, err := cloudItsmSubmitRetryConfigMap(c)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	approvedTickets := make([]models.CloudItsmTicket, 0, len(ids))
	for _, id := range ids {
		ticket := ticketMap[id]
		if ticket == nil {
			cloudItsmSubmitRetryAdd(result, "skipped", 1)
			cloudItsmDeadLetterActionAddError(result, id, "not_found", "工单不存在或无权限")
			continue
		}
		queueStatus, eligibility := cloudItsmSubmitRetryQueueStatus(configs[ticket.ConnectorId], *ticket, now)
		if queueStatus != "dead_letter" {
			cloudItsmSubmitRetryAdd(result, "skipped", 1)
			cloudItsmDeadLetterActionAddError(result, ticket.Id, "not_dead_letter", fmt.Sprintf("当前队列状态为 %s", queueStatus))
			continue
		}
		approvedTickets = append(approvedTickets, *ticket)
		cloudItsmSubmitRetryAdd(result, "deadLetter", 1)
		result["lastRetryReason"] = eligibility.Reason
	}
	return approvedTickets, result, nil
}

func cloudItsmDeadLetterApprovalErrorSummary(errors []models.ResAttrs) string {
	items := make([]string, 0, len(errors))
	for _, item := range errors {
		items = append(items, fmt.Sprintf("%s/%s", cloudSyncPolicyAttrString(item["ticketId"]), cloudSyncPolicyAttrString(item["reason"])))
	}
	return strings.Join(items, "；")
}

func cloudItsmDeadLetterApprovalTicketIds(tickets []models.CloudItsmTicket) []models.Id {
	ids := make([]models.Id, 0, len(tickets))
	for _, ticket := range tickets {
		ids = append(ids, ticket.Id)
	}
	return ids
}

func cloudItsmDeadLetterApprovalScope(tickets []models.CloudItsmTicket) (models.Id, models.Id) {
	var projectId models.Id
	var envId models.Id
	for idx, ticket := range tickets {
		if idx == 0 {
			projectId = ticket.ProjectId
			envId = ticket.EnvId
			continue
		}
		if ticket.ProjectId != projectId {
			projectId = ""
		}
		if ticket.EnvId != envId {
			envId = ""
		}
	}
	return projectId, envId
}

func cloudItsmDeadLetterApprovalResourceId(ids []models.Id) string {
	if len(ids) == 0 {
		return ""
	}
	if len(ids) == 1 {
		return ids[0].String()
	}
	return fmt.Sprintf("%s +%d", ids[0], len(ids)-1)
}

func cloudItsmDeadLetterApprovalEvidence(form *forms.CreateCloudItsmDeadLetterApprovalForm, tickets []models.CloudItsmTicket, precheck models.ResAttrs, action string, actionLabel string, reason string, requestedBy string, requestedAt string) (models.ResAttrs, e.Error) {
	evidence := copyResAttrs(form.Evidence)
	items, err := cloudItsmDeadLetterApprovalEvidenceItems(form.EvidenceUrl, form.EvidenceItems)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		evidence["items"] = items
		evidence["itemCount"] = len(items)
		if _, ok := evidence["url"]; !ok {
			if firstUrl := cloudSyncPolicyAttrString(items[0]["url"]); firstUrl != "" {
				evidence["url"] = firstUrl
			}
		}
	}
	evidence["snapshot"] = cloudItsmDeadLetterApprovalEvidenceSnapshot(tickets, precheck, action, actionLabel, reason, requestedBy, requestedAt)
	evidence["snapshotVersion"] = "v1"
	evidence["snapshotGeneratedAt"] = requestedAt
	return evidence, nil
}

func cloudItsmDeadLetterApprovalEvidenceItems(evidenceUrl string, raw []forms.CloudItsmDeadLetterApprovalEvidenceItem) ([]models.ResAttrs, e.Error) {
	items := make([]models.ResAttrs, 0, len(raw)+1)
	if url := strings.TrimSpace(evidenceUrl); url != "" {
		if err := cloudItsmValidateHTTPURL(url, "外部证据链接"); err != nil {
			return nil, err
		}
		items = append(items, models.ResAttrs{
			"type":  "link",
			"label": "外部证据链接",
			"url":   url,
		})
	}
	for _, rawItem := range raw {
		itemType := cloudItsmDeadLetterApprovalEvidenceItemType(rawItem.Type)
		label := strings.TrimSpace(rawItem.Label)
		itemURL := strings.TrimSpace(rawItem.Url)
		note := strings.TrimSpace(rawItem.Note)
		if label == "" && itemURL == "" && note == "" {
			continue
		}
		if itemURL != "" {
			if err := cloudItsmValidateHTTPURL(itemURL, "证据链接"); err != nil {
				return nil, err
			}
		}
		if label == "" {
			label = cloudItsmDeadLetterApprovalEvidenceItemLabel(itemType)
		}
		if len(items) >= cloudItsmDeadLetterApprovalEvidenceItemLimit {
			return nil, e.New(e.BadParam, fmt.Errorf("审批证据最多支持 %d 条", cloudItsmDeadLetterApprovalEvidenceItemLimit), http.StatusBadRequest)
		}
		item := models.ResAttrs{
			"type":  itemType,
			"label": label,
		}
		if itemURL != "" {
			item["url"] = itemURL
		}
		if note != "" {
			item["note"] = note
		}
		items = append(items, item)
	}
	return items, nil
}

func cloudItsmDeadLetterApprovalEvidenceItemType(itemType string) string {
	switch strings.ToLower(strings.TrimSpace(itemType)) {
	case "", "link", "url":
		return "link"
	case "pr", "pull_request", "merge_request":
		return "pull_request"
	case "ticket", "change", "change_request", "incident", "document", "runbook", "snapshot", "other":
		return strings.ToLower(strings.TrimSpace(itemType))
	default:
		return "other"
	}
}

func cloudItsmDeadLetterApprovalEvidenceItemLabel(itemType string) string {
	switch itemType {
	case "pull_request":
		return "PR/评审链接"
	case "ticket":
		return "外部工单"
	case "change", "change_request":
		return "变更记录"
	case "incident":
		return "事故记录"
	case "document":
		return "文档"
	case "runbook":
		return "Runbook"
	case "snapshot":
		return "现场快照"
	case "other":
		return "补充证据"
	default:
		return "证据链接"
	}
}

func cloudItsmDeadLetterApprovalEvidenceSnapshot(tickets []models.CloudItsmTicket, precheck models.ResAttrs, action string, actionLabel string, reason string, requestedBy string, requestedAt string) models.ResAttrs {
	ticketSnapshots := make([]models.ResAttrs, 0, len(tickets))
	for _, ticket := range tickets {
		ticketSnapshots = append(ticketSnapshots, cloudItsmDeadLetterApprovalTicketSnapshot(ticket))
	}
	return models.ResAttrs{
		"version":     "v1",
		"generatedAt": requestedAt,
		"requestedBy": requestedBy,
		"action":      action,
		"actionLabel": actionLabel,
		"reason":      reason,
		"ticketCount": len(ticketSnapshots),
		"precheck":    precheck,
		"tickets":     ticketSnapshots,
	}
}

func cloudItsmDeadLetterApprovalTicketSnapshot(ticket models.CloudItsmTicket) models.ResAttrs {
	retryState := copyResAttrs(modelResAttrs(ticket.ResponsePayload["submitRetry"]))
	retryReason := cloudSyncPolicyAttrString(retryState["reason"])
	errorCodeId, errorCodeName := cloudItsmSubmitRetryQueueErrorCodeDimension(ticket, resps.CloudItsmSubmitRetryQueueItemResp{RetryReason: retryReason})
	externalCodeId, externalCodeName := cloudItsmSubmitRetryQueueExternalResponseCodeDimension(ticket)
	snapshot := models.ResAttrs{
		"id":                   ticket.Id.String(),
		"connectorId":          ticket.ConnectorId.String(),
		"projectId":            ticket.ProjectId.String(),
		"envId":                ticket.EnvId.String(),
		"operationId":          ticket.OperationId.String(),
		"cloudEventId":         ticket.CloudEventId.String(),
		"externalId":           ticket.ExternalId,
		"externalKey":          ticket.ExternalKey,
		"externalUrl":          ticket.ExternalUrl,
		"title":                ticket.Title,
		"status":               ticket.Status,
		"priority":             ticket.Priority,
		"riskLevel":            ticket.RiskLevel,
		"provider":             ticket.Provider,
		"retryReason":          retryReason,
		"responseStatusCode":   cloudItsmSubmitRetryQueueResponseStatusCode(ticket),
		"errorCode":            models.ResAttrs{"id": errorCodeId, "name": errorCodeName},
		"externalResponseCode": models.ResAttrs{"id": externalCodeId, "name": externalCodeName},
		"submittedAt":          cloudItsmTimeString(ticket.SubmittedAt),
		"lastSyncedAt":         cloudItsmTimeString(ticket.LastSyncedAt),
		"closedAt":             cloudItsmTimeString(ticket.ClosedAt),
	}
	if len(retryState) > 0 {
		snapshot["retryState"] = retryState
	}
	if strings.TrimSpace(ticket.ErrorMessage) != "" {
		snapshot["errorMessage"] = ticket.ErrorMessage
	}
	return snapshot
}

func cloudItsmTimeString(value models.Time) string {
	t := time.Time(value)
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func cloudItsmDeadLetterApprovalNotificationStrategy(action string, ticketCount int, evidence models.ResAttrs) models.ResAttrs {
	actionLabel := cloudItsmDeadLetterActionLabel(action)
	severity := models.CloudEventLevelWarning
	if action == "replay" {
		severity = models.CloudEventLevelError
	}
	itemCount := 0
	switch value := evidence["itemCount"].(type) {
	case int:
		itemCount = value
	case float64:
		itemCount = int(value)
	}
	return models.ResAttrs{
		"enabled":                 true,
		"source":                  "cloud_itsm_dead_letter",
		"severity":                severity,
		"channels":                []string{"cloud_event", "notification_policy"},
		"routes":                  []string{"itsm.dead_letter", "self_service.approval"},
		"ownerRoles":              []string{"org_admin", "sre", "operator"},
		"eventTypes":              []string{"itsm.dead_letter.approval_requested", "itsm.dead_letter.approval_rejected", "itsm.dead_letter.approval_executed", "itsm.dead_letter.approval_execute_failed"},
		"includeEvidenceSnapshot": true,
		"evidenceItemCount":       itemCount,
		"ticketCount":             ticketCount,
		"messageTemplate":         fmt.Sprintf("ITSM 死信%s审批 {{status}}：{{ticketCount}} 条", actionLabel),
	}
}

func cloudItsmDeadLetterApprovalIdsFromPayload(value interface{}) []models.Id {
	if typed, ok := value.([]models.Id); ok {
		ids := make([]models.Id, 0, len(typed))
		for _, item := range typed {
			if item != "" {
				ids = append(ids, item)
			}
		}
		return ids
	}
	raw := cloudSyncPolicyAttrStringSlice(value)
	ids := make([]models.Id, 0, len(raw))
	for _, item := range raw {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			ids = append(ids, models.Id(trimmed))
		}
	}
	return ids
}

func cloudItsmDeadLetterActionLabel(action string) string {
	if action == "close" {
		return "关闭"
	}
	return "重放"
}

func cloudItsmDeadLetterApprovalEvent(c *ctx.ServiceContext, operation *models.CloudOperation, eventType string, level string, title string, message string, payload models.ResAttrs) {
	if c == nil || operation == nil {
		return
	}
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:        operation.OrgId,
		ProjectId:    operation.ProjectId,
		EnvId:        operation.EnvId,
		OperationId:  operation.Id,
		ActorId:      c.UserId,
		Source:       models.CloudEventSourceITSM,
		EventType:    eventType,
		Level:        level,
		Status:       operation.Status,
		ResourceType: operation.ResourceType,
		ResourceId:   operation.ResourceId,
		ResourceName: operation.ResourceName,
		Title:        title,
		Message:      message,
		Payload:      payload,
	})
}

func BatchCloudItsmDeadLetterAction(c *ctx.ServiceContext, form *forms.BatchCloudItsmDeadLetterActionForm) (models.ResAttrs, e.Error) {
	if form == nil || len(form.Ids) == 0 {
		return nil, e.New(e.BadParam, fmt.Errorf("请选择死信工单"), http.StatusBadRequest)
	}
	if len(form.Ids) > 50 {
		return nil, e.New(e.BadParam, fmt.Errorf("单次最多处理 50 条死信"), http.StatusBadRequest)
	}
	action := strings.TrimSpace(form.Action)
	if action != "replay" && action != "close" {
		return nil, e.New(e.BadParam, fmt.Errorf("不支持的死信操作 %s", action), http.StatusBadRequest)
	}
	now := time.Now()
	result := cloudItsmDeadLetterActionResult(action, len(form.Ids))
	tickets := make([]models.CloudItsmTicket, 0, len(form.Ids))
	if err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and id in (?)", c.OrgId, form.Ids).
		Find(&tickets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	ticketMap := map[models.Id]*models.CloudItsmTicket{}
	for idx := range tickets {
		ticketMap[tickets[idx].Id] = &tickets[idx]
	}
	configs, err := cloudItsmSubmitRetryConfigMap(c)
	if err != nil {
		return nil, err
	}
	for _, id := range form.Ids {
		ticket := ticketMap[id]
		if ticket == nil {
			cloudItsmSubmitRetryAdd(result, "skipped", 1)
			cloudItsmDeadLetterActionAddError(result, id, "not_found", "工单不存在或无权限")
			continue
		}
		cfg := configs[ticket.ConnectorId]
		queueStatus, eligibility := cloudItsmSubmitRetryQueueStatus(cfg, *ticket, now)
		if queueStatus != "dead_letter" {
			cloudItsmSubmitRetryAdd(result, "skipped", 1)
			cloudItsmDeadLetterActionAddError(result, ticket.Id, "not_dead_letter", fmt.Sprintf("当前队列状态为 %s", queueStatus))
			continue
		}
		switch action {
		case "replay":
			if cfg == nil {
				cloudItsmSubmitRetryAdd(result, "skipped", 1)
				cloudItsmDeadLetterActionAddError(result, ticket.Id, "connector_missing", "连接器不存在")
				continue
			}
			if cfg.Status != models.CloudItsmConfigStatusEnabled {
				cloudItsmSubmitRetryAdd(result, "skipped", 1)
				cloudItsmDeadLetterActionAddError(result, ticket.Id, "connector_disabled", "连接器已停用")
				continue
			}
			item, retryErr := retryCloudItsmSingleTicketSubmission(c, cfg, ticket, now, form.Force || eligibility.DeadLetter, true)
			if retryErr != nil {
				cloudItsmSubmitRetryAdd(result, "failed", 1)
				cloudItsmDeadLetterActionAddError(result, ticket.Id, "replay_failed", retryErr.Error())
				continue
			}
			if !attrBool(item, "retried") {
				cloudItsmSubmitRetryAdd(result, "skipped", 1)
				cloudItsmDeadLetterActionAddError(result, ticket.Id, cloudSyncPolicyAttrString(item["reason"]), "死信未满足重放条件")
				continue
			}
			cloudItsmSubmitRetryAdd(result, "replayed", 1)
			if cloudSyncPolicyAttrString(item["status"]) == models.CloudItsmTicketStatusSubmitted {
				cloudItsmSubmitRetryAdd(result, "submitted", 1)
			} else {
				cloudItsmSubmitRetryAdd(result, "failed", 1)
				cloudItsmDeadLetterActionAddError(result, ticket.Id, "replay_failed", cloudSyncPolicyAttrString(item["errorMessage"]))
			}
		case "close":
			if closeErr := closeCloudItsmDeadLetterTicket(c, ticket, strings.TrimSpace(form.Reason), now); closeErr != nil {
				cloudItsmSubmitRetryAdd(result, "failed", 1)
				cloudItsmDeadLetterActionAddError(result, ticket.Id, "close_failed", closeErr.Error())
				continue
			}
			cloudItsmSubmitRetryAdd(result, "closed", 1)
		}
	}
	cloudItsmDeadLetterBatchEvent(c, action, result)
	return result, nil
}

func cloudItsmDeadLetterActionResult(action string, total int) models.ResAttrs {
	return models.ResAttrs{
		"action":    action,
		"total":     total,
		"replayed":  0,
		"submitted": 0,
		"closed":    0,
		"failed":    0,
		"skipped":   0,
		"errors":    []models.ResAttrs{},
	}
}

func cloudItsmDeadLetterActionAddError(result models.ResAttrs, ticketId models.Id, reason string, message string) {
	if reason == "" {
		reason = "unknown"
	}
	cloudItsmSubmitRetryAddError(result, models.ResAttrs{
		"ticketId": ticketId.String(),
		"reason":   reason,
		"message":  message,
	})
}

func closeCloudItsmDeadLetterTicket(c *ctx.ServiceContext, ticket *models.CloudItsmTicket, reason string, now time.Time) error {
	if c == nil || ticket == nil {
		return fmt.Errorf("ITSM 工单不存在")
	}
	responsePayload, closeReason := cloudItsmDeadLetterClosePayload(*ticket, reason, c.UserId, now)
	attrs := models.Attrs{
		"status":           models.CloudItsmTicketStatusCanceled,
		"response_payload": responsePayload,
		"last_synced_at":   models.Time(now),
		"closed_at":        models.Time(now),
	}
	if _, err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("id = ? and org_id = ?", ticket.Id, c.OrgId).
		UpdateAttrs(attrs); err != nil {
		return err
	}
	ticket.Status = models.CloudItsmTicketStatusCanceled
	ticket.ResponsePayload = responsePayload
	ticket.LastSyncedAt = models.Time(now)
	ticket.ClosedAt = models.Time(now)
	eventId := cloudItsmDeadLetterTicketEvent(c, ticket, "itsm.ticket.dead_letter_closed", models.CloudEventLevelWarning, "ITSM 死信已人工关闭", closeReason, models.ResAttrs{
		"action": "close",
		"reason": closeReason,
	})
	if eventId != "" {
		_, _ = c.DB().Model(&models.CloudItsmTicket{}).
			Where("id = ? and org_id = ?", ticket.Id, c.OrgId).
			UpdateAttrs(models.Attrs{"cloud_event_id": eventId})
		ticket.CloudEventId = eventId
	}
	return nil
}

func cloudItsmDeadLetterClosePayload(ticket models.CloudItsmTicket, reason string, operatorId models.Id, now time.Time) (models.ResAttrs, string) {
	responsePayload := copyResAttrs(ticket.ResponsePayload)
	retryState := copyResAttrs(modelResAttrs(responsePayload["submitRetry"]))
	if reason == "" {
		reason = "人工确认关闭死信"
	}
	retryState["deadLetter"] = true
	retryState["deadLetterClosed"] = true
	retryState["deadLetterClosedAt"] = now.Format(time.RFC3339)
	retryState["deadLetterClosedBy"] = operatorId.String()
	retryState["deadLetterCloseReason"] = reason
	retryState["deadLetterAction"] = "close"
	responsePayload["submitRetry"] = retryState
	return responsePayload, reason
}

func cloudItsmDeadLetterTicketEvent(c *ctx.ServiceContext, ticket *models.CloudItsmTicket, eventType string, level string, title string, message string, payload models.ResAttrs) models.Id {
	if c == nil || ticket == nil {
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
		ActorId:     c.UserId,
		Source:      models.CloudEventSourceITSM,
		EventType:   eventType,
		Level:       level,
		Status:      ticket.Status,
		Title:       title,
		Message:     message,
		Payload:     payload,
	}
	event.Id = models.NewId("cev")
	recordCloudEventBestEffort(c, event)
	return event.Id
}

func cloudItsmDeadLetterBatchEvent(c *ctx.ServiceContext, action string, result models.ResAttrs) {
	if c == nil || result == nil {
		return
	}
	eventType := "itsm.dead_letter.batch_replayed"
	title := "ITSM 死信批量重放"
	message := fmt.Sprintf("重放 %d 条，失败 %d 条，跳过 %d 条", cloudItsmSubmitRetryCount(result, "replayed"), cloudItsmSubmitRetryCount(result, "failed"), cloudItsmSubmitRetryCount(result, "skipped"))
	if action == "close" {
		eventType = "itsm.dead_letter.batch_closed"
		title = "ITSM 死信批量关闭"
		message = fmt.Sprintf("关闭 %d 条，失败 %d 条，跳过 %d 条", cloudItsmSubmitRetryCount(result, "closed"), cloudItsmSubmitRetryCount(result, "failed"), cloudItsmSubmitRetryCount(result, "skipped"))
	}
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:     c.OrgId,
		ActorId:   c.UserId,
		Source:    models.CloudEventSourceITSM,
		EventType: eventType,
		Level:     models.CloudEventLevelInfo,
		Status:    "completed",
		Title:     title,
		Message:   message,
		Payload:   result,
	})
}

func RetryDueCloudItsmTicketSubmissionsForAllOrgs() (models.ResAttrs, e.Error) {
	configs := make([]models.CloudItsmConfig, 0)
	if err := db.Get().Model(&models.CloudItsmConfig{}).
		Where("status = ?", models.CloudItsmConfigStatusEnabled).
		Order("org_id asc, updated_at desc").
		Find(&configs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	result := cloudItsmSubmitRetryResult()
	result["connectorCount"] = len(configs)
	seenOrgs := map[models.Id]bool{}
	for idx := range configs {
		cfg := &configs[idx]
		if cfg.OrgId == "" {
			cloudItsmSubmitRetryAdd(result, "skipped", 1)
			continue
		}
		if !cloudItsmSubmitRetryEnabled(*cfg) {
			cloudItsmSubmitRetryAdd(result, "connectorSkipped", 1)
			continue
		}
		if !seenOrgs[cfg.OrgId] {
			seenOrgs[cfg.OrgId] = true
			cloudItsmSubmitRetryAdd(result, "orgCount", 1)
		}
		locked, releaseLock, lockErr := cloudWebhookAcquireMysqlLock(cloudItsmSubmitRetryLockName(cfg.OrgId))
		if lockErr != nil {
			return nil, e.New(e.DBError, lockErr)
		}
		if !locked {
			cloudItsmSubmitRetryAdd(result, "lockSkipped", 1)
			continue
		}
		item, err := func() (models.ResAttrs, e.Error) {
			defer releaseLock()
			workerCtx := &ctx.ServiceContext{
				UserId:   consts.SysUserId,
				OrgId:    cfg.OrgId,
				Email:    consts.DefaultSysEmail,
				Username: consts.DefaultSysName,
			}
			return retryCloudItsmConnectorTicketSubmissions(workerCtx, cfg, cloudItsmSubmitRetryDefaultLimit, false)
		}()
		if err != nil {
			return nil, err
		}
		cloudItsmSubmitRetryMerge(result, item)
	}
	return result, nil
}

func StartCloudItsmSubmitRetryWorker(serviceId string) {
	logger := logs.Get().
		WithField("worker", "cloudItsmSubmitRetry").
		WithField("serviceId", serviceId).
		WithField("interval", cloudItsmSubmitRetryWorkerInterval.String())
	ticker := time.NewTicker(cloudItsmSubmitRetryWorkerInterval)
	defer ticker.Stop()
	for {
		if result, err := RetryDueCloudItsmTicketSubmissionsForAllOrgs(); err != nil {
			logger.Warnf("retry due cloud ITSM ticket submissions failed: %v", err)
		} else if cloudItsmSubmitRetryCount(result, "retried") > 0 ||
			cloudItsmSubmitRetryCount(result, "failed") > 0 ||
			cloudItsmSubmitRetryCount(result, "deadLetter") > 0 ||
			cloudItsmSubmitRetryCount(result, "lockSkipped") > 0 {
			logger.Infof("retry due cloud ITSM ticket submissions result: %+v", result)
		}
		<-ticker.C
	}
}

func cloudItsmSubmitRetryConfigMap(c *ctx.ServiceContext) (map[models.Id]*models.CloudItsmConfig, e.Error) {
	configs := make([]models.CloudItsmConfig, 0)
	if err := c.DB().Model(&models.CloudItsmConfig{}).
		Where("org_id = ?", c.OrgId).
		Find(&configs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	result := make(map[models.Id]*models.CloudItsmConfig, len(configs))
	for idx := range configs {
		cfg := &configs[idx]
		result[cfg.Id] = cfg
	}
	return result, nil
}

func cloudItsmSubmitRetryQueueSummaryAdd(summary models.ResAttrs, queueStatus string, reason string) {
	switch queueStatus {
	case "due", "future", "skipped":
		cloudItsmSubmitRetryAdd(summary, queueStatus, 1)
	case "dead_letter":
		cloudItsmSubmitRetryAdd(summary, "deadLetter", 1)
	default:
		cloudItsmSubmitRetryAdd(summary, "skipped", 1)
	}
	reasons := modelResAttrs(summary["reasons"])
	if reasons == nil {
		reasons = models.ResAttrs{}
	}
	if reason == "" {
		reason = "unknown"
	}
	reasons[reason] = attrInt(reasons, reason) + 1
	summary["reasons"] = reasons
}

func cloudItsmSubmitRetryQueueReport(c *ctx.ServiceContext, tickets []models.CloudItsmTicket, configs map[models.Id]*models.CloudItsmConfig, now time.Time) *resps.CloudItsmSubmitRetryQueueReportResp {
	connectorBreakdowns := map[string]*resps.CloudItsmSubmitRetryQueueBreakdownResp{}
	reasonBreakdowns := map[string]*resps.CloudItsmSubmitRetryQueueBreakdownResp{}
	ageBreakdowns := map[string]*resps.CloudItsmSubmitRetryQueueBreakdownResp{}
	projectBreakdowns := map[string]*resps.CloudItsmSubmitRetryQueueBreakdownResp{}
	requestTypeBreakdowns := map[string]*resps.CloudItsmSubmitRetryQueueBreakdownResp{}
	teamBreakdowns := map[string]*resps.CloudItsmSubmitRetryQueueBreakdownResp{}
	errorCodeBreakdowns := map[string]*resps.CloudItsmSubmitRetryQueueBreakdownResp{}
	externalResponseCodeBreakdowns := map[string]*resps.CloudItsmSubmitRetryQueueBreakdownResp{}
	projectNames := map[string]string{}
	requestTypeNames := cloudItsmCatalogNameMap(c)
	teamCache := &cloudItsmTeamDimensionCache{
		projectTeams: map[string]cloudItsmTeamDimension{},
		userTeams:    map[string]cloudItsmTeamDimension{},
	}
	recentDeadLetters := make([]resps.CloudItsmSubmitRetryQueueItemResp, 0)
	for idx := range tickets {
		ticket := tickets[idx]
		cfg := configs[ticket.ConnectorId]
		item := cloudItsmSubmitRetryQueueItem(c, ticket, cfg, now)
		if item.ConnectorName == "" && cfg != nil {
			item.ConnectorName = cfg.Name
		}
		connectorId, connectorName, provider := cloudItsmSubmitRetryQueueConnectorDimension(ticket, cfg)
		cloudItsmSubmitRetryQueueReportAdd(connectorBreakdowns, "connector", connectorId, connectorName, provider, item)

		reasonId := firstNonEmpty(item.RetryReason, "unknown")
		cloudItsmSubmitRetryQueueReportAdd(reasonBreakdowns, "reason", reasonId, cloudItsmSubmitRetryQueueReasonLabel(reasonId), "", item)

		ageId, ageName := cloudItsmSubmitRetryQueueAgeBucket(now, ticket)
		cloudItsmSubmitRetryQueueReportAdd(ageBreakdowns, "age", ageId, ageName, "", item)

		projectId, projectName := cloudItsmSubmitRetryQueueProjectDimension(c, ticket, projectNames)
		cloudItsmSubmitRetryQueueReportAdd(projectBreakdowns, "project", projectId, projectName, "", item)

		team := cloudItsmTicketTeamDimension(c, ticket, projectId, projectName, teamCache)
		cloudItsmSubmitRetryQueueReportAdd(teamBreakdowns, "team", team.Id, team.Name, "", item)

		requestTypeId, requestTypeName := cloudItsmSubmitRetryQueueRequestTypeDimension(ticket, requestTypeNames)
		cloudItsmSubmitRetryQueueReportAdd(requestTypeBreakdowns, "request_type", requestTypeId, requestTypeName, "", item)

		errorCodeId, errorCodeName := cloudItsmSubmitRetryQueueErrorCodeDimension(ticket, item)
		cloudItsmSubmitRetryQueueReportAdd(errorCodeBreakdowns, "error_code", errorCodeId, errorCodeName, "", item)

		externalResponseCodeId, externalResponseCodeName := cloudItsmSubmitRetryQueueExternalResponseCodeDimension(ticket)
		cloudItsmSubmitRetryQueueReportAdd(externalResponseCodeBreakdowns, "external_response_code", externalResponseCodeId, externalResponseCodeName, "", item)

		if item.QueueStatus == "dead_letter" && len(recentDeadLetters) < 8 {
			recentDeadLetters = append(recentDeadLetters, item)
		}
	}
	return &resps.CloudItsmSubmitRetryQueueReportResp{
		ConnectorBreakdown:            cloudItsmSubmitRetryQueueReportBreakdowns(connectorBreakdowns, false),
		ReasonBreakdown:               cloudItsmSubmitRetryQueueReportBreakdowns(reasonBreakdowns, false),
		AgeBuckets:                    cloudItsmSubmitRetryQueueReportBreakdowns(ageBreakdowns, true),
		ProjectBreakdown:              cloudItsmSubmitRetryQueueReportBreakdowns(projectBreakdowns, false),
		RequestTypeBreakdown:          cloudItsmSubmitRetryQueueReportBreakdowns(requestTypeBreakdowns, false),
		TeamBreakdown:                 cloudItsmSubmitRetryQueueReportBreakdowns(teamBreakdowns, false),
		ErrorCodeBreakdown:            cloudItsmSubmitRetryQueueReportBreakdowns(errorCodeBreakdowns, false),
		ExternalResponseCodeBreakdown: cloudItsmSubmitRetryQueueReportBreakdowns(externalResponseCodeBreakdowns, false),
		RecentDeadLetters:             recentDeadLetters,
	}
}

func cloudItsmSubmitRetryQueueConnectorDimension(ticket models.CloudItsmTicket, cfg *models.CloudItsmConfig) (string, string, string) {
	if cfg != nil {
		return cfg.Id.String(), firstNonEmpty(cfg.Name, cfg.Id.String()), cfg.Provider
	}
	if ticket.ConnectorId != "" {
		return ticket.ConnectorId.String(), ticket.ConnectorId.String(), ""
	}
	return "missing", "连接器缺失", ""
}

func cloudItsmSubmitRetryQueueProjectDimension(c *ctx.ServiceContext, ticket models.CloudItsmTicket, projectNames map[string]string) (string, string) {
	if ticket.ProjectId == "" {
		return "unassigned", "未关联项目"
	}
	projectId := ticket.ProjectId.String()
	if projectNames == nil {
		projectNames = map[string]string{}
	}
	if name, ok := projectNames[projectId]; ok {
		return projectId, name
	}
	projectName := projectId
	if c != nil {
		if name := lookupName(c, &models.Project{}, ticket.ProjectId); name != "" {
			projectName = name
		}
	}
	projectNames[projectId] = projectName
	return projectId, projectName
}

func cloudItsmSubmitRetryQueueRequestTypeDimension(ticket models.CloudItsmTicket, requestTypeNames map[string]string) (string, string) {
	requestType := cloudItsmTicketRequestType(ticket)
	if requestType == "" {
		return "unknown", "未知申请类型"
	}
	return requestType, firstNonEmpty(requestTypeNames[requestType], requestType)
}

func cloudItsmSubmitRetryQueueErrorCodeDimension(ticket models.CloudItsmTicket, item resps.CloudItsmSubmitRetryQueueItemResp) (string, string) {
	for _, path := range []string{
		"errorCode",
		"code",
		"error.code",
		"error.errorCode",
		"result.errorCode",
		"result.error.code",
		"ticket.errorCode",
		"ticket.error.code",
		"data.errorCode",
		"data.error.code",
		"json.errorCode",
		"json.code",
		"json.error.code",
		"json.error.errorCode",
		"json.result.errorCode",
		"json.result.error.code",
		"submitRetry.errorCode",
		"submitRetry.lastErrorCode",
	} {
		if code := strings.TrimSpace(cloudItsmNestedAttrString(ticket.ResponsePayload, path)); code != "" {
			return cloudItsmSubmitRetryQueueCodeDimension(code, code)
		}
	}
	if statusCode := cloudItsmSubmitRetryQueueResponseStatusCode(ticket); statusCode > 0 {
		return cloudItsmSubmitRetryQueueHTTPStatusDimension(statusCode)
	}
	if item.RetryReason != "" {
		return cloudItsmSubmitRetryQueueCodeDimension(item.RetryReason, cloudItsmSubmitRetryQueueReasonLabel(item.RetryReason))
	}
	return "unknown", "未知错误码"
}

func cloudItsmSubmitRetryQueueExternalResponseCodeDimension(ticket models.CloudItsmTicket) (string, string) {
	if statusCode := cloudItsmSubmitRetryQueueResponseStatusCode(ticket); statusCode > 0 {
		return cloudItsmSubmitRetryQueueHTTPStatusDimension(statusCode)
	}
	return "unknown", "未返回"
}

func cloudItsmSubmitRetryQueueResponseStatusCode(ticket models.CloudItsmTicket) int {
	for _, path := range []string{
		"statusCode",
		"httpStatus",
		"http_status",
		"responseStatusCode",
		"response.statusCode",
		"result.statusCode",
		"ticket.statusCode",
		"data.statusCode",
		"json.statusCode",
		"json.httpStatus",
		"json.response.statusCode",
		"submitRetry.lastStatusCode",
	} {
		statusCode := cloudItsmSubmitRetryQueueHTTPStatusCode(cloudItsmNestedAttrString(ticket.ResponsePayload, path))
		if statusCode > 0 {
			return statusCode
		}
	}
	return 0
}

func cloudItsmSubmitRetryQueueHTTPStatusCode(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	statusCode, err := strconv.Atoi(value)
	if err != nil || statusCode < 100 || statusCode > 599 {
		return 0
	}
	return statusCode
}

func cloudItsmSubmitRetryQueueHTTPStatusDimension(statusCode int) (string, string) {
	return fmt.Sprintf("http_%d", statusCode), fmt.Sprintf("HTTP %d", statusCode)
}

func cloudItsmSubmitRetryQueueCodeDimension(code string, label string) (string, string) {
	code = strings.TrimSpace(code)
	label = strings.TrimSpace(label)
	if code == "" {
		return "unknown", "未知错误码"
	}
	return code, firstNonEmpty(label, code)
}

func cloudItsmSubmitRetryQueueReportAdd(breakdowns map[string]*resps.CloudItsmSubmitRetryQueueBreakdownResp, dimensionType string, dimensionId string, dimensionName string, provider string, item resps.CloudItsmSubmitRetryQueueItemResp) {
	if dimensionId == "" {
		dimensionId = "unknown"
	}
	if dimensionName == "" {
		dimensionName = dimensionId
	}
	breakdown := breakdowns[dimensionId]
	if breakdown == nil {
		breakdown = &resps.CloudItsmSubmitRetryQueueBreakdownResp{
			DimensionType: dimensionType,
			DimensionId:   dimensionId,
			DimensionName: dimensionName,
			Provider:      provider,
		}
		breakdowns[dimensionId] = breakdown
	}
	breakdown.FailedTotal++
	switch item.QueueStatus {
	case "due":
		breakdown.Due++
		breakdown.Retryable++
		nextRetryAt := cloudItsmSubmitRetryQueueNextRetryAt(item)
		if !nextRetryAt.IsZero() && (breakdown.OldestDueAt == "" || nextRetryAt.Before(cloudItsmSubmitRetryTime(breakdown.OldestDueAt))) {
			breakdown.OldestDueAt = nextRetryAt.Format(time.RFC3339)
		}
	case "future":
		breakdown.Future++
		breakdown.Retryable++
		nextRetryAt := cloudItsmSubmitRetryQueueNextRetryAt(item)
		if !nextRetryAt.IsZero() && (breakdown.NextRetryAt == "" || nextRetryAt.Before(cloudItsmSubmitRetryTime(breakdown.NextRetryAt))) {
			breakdown.NextRetryAt = nextRetryAt.Format(time.RFC3339)
		}
	case "dead_letter":
		breakdown.DeadLetter++
	case "skipped":
		breakdown.Skipped++
	default:
		breakdown.Skipped++
	}
}

func cloudItsmSubmitRetryQueueReportBreakdowns(breakdowns map[string]*resps.CloudItsmSubmitRetryQueueBreakdownResp, ageOrder bool) []resps.CloudItsmSubmitRetryQueueBreakdownResp {
	result := make([]resps.CloudItsmSubmitRetryQueueBreakdownResp, 0, len(breakdowns))
	for _, breakdown := range breakdowns {
		result = append(result, *breakdown)
	}
	sort.Slice(result, func(i, j int) bool {
		if ageOrder {
			return cloudItsmSubmitRetryQueueAgeOrder(result[i].DimensionId) < cloudItsmSubmitRetryQueueAgeOrder(result[j].DimensionId)
		}
		if result[i].FailedTotal == result[j].FailedTotal {
			return result[i].DimensionName < result[j].DimensionName
		}
		return result[i].FailedTotal > result[j].FailedTotal
	})
	return result
}

func cloudItsmSubmitRetryQueueAgeBucket(now time.Time, ticket models.CloudItsmTicket) (string, string) {
	failedAt := time.Time(ticket.UpdatedAt)
	if failedAt.IsZero() {
		failedAt = time.Time(ticket.CreatedAt)
	}
	if failedAt.IsZero() || now.Before(failedAt) {
		return "unknown", "未知"
	}
	age := now.Sub(failedAt)
	switch {
	case age < time.Hour:
		return "lt_1h", "< 1小时"
	case age < 6*time.Hour:
		return "1_6h", "1-6小时"
	case age < 24*time.Hour:
		return "6_24h", "6-24小时"
	default:
		return "gt_24h", "> 24小时"
	}
}

func cloudItsmSubmitRetryQueueAgeOrder(value string) int {
	switch value {
	case "lt_1h":
		return 1
	case "1_6h":
		return 2
	case "6_24h":
		return 3
	case "gt_24h":
		return 4
	default:
		return 99
	}
}

func cloudItsmSubmitRetryQueueReasonLabel(reason string) string {
	labels := map[string]string{
		"due":                       "到期可重试",
		"not_due":                   "等待退避",
		"max_attempts_reached":      "达到最大尝试次数",
		"connector_missing":         "连接器缺失",
		"connector_disabled":        "连接器停用",
		"disabled":                  "补偿策略停用",
		"no_endpoint":               "缺少提交地址",
		"missing_request_payload":   "缺少请求载荷",
		"external_identity_present": "已有外部单号保护",
		"dead_letter":               "死信",
		"status_not_failed":         "状态非失败",
		"unknown":                   "未知原因",
	}
	return firstNonEmpty(labels[reason], reason)
}

func cloudItsmSubmitRetryQueueItem(c *ctx.ServiceContext, ticket models.CloudItsmTicket, cfg *models.CloudItsmConfig, now time.Time) resps.CloudItsmSubmitRetryQueueItemResp {
	state := modelResAttrs(ticket.ResponsePayload["submitRetry"])
	queueStatus, eligibility := cloudItsmSubmitRetryQueueStatus(cfg, ticket, now)
	ticketResp := resps.CloudItsmTicketResp{CloudItsmTicket: ticket}
	if c != nil {
		ticketResp = cloudItsmTicketResp(c, ticket, nil)
	}
	resp := resps.CloudItsmSubmitRetryQueueItemResp{
		CloudItsmTicketResp: ticketResp,
		QueueStatus:         queueStatus,
		RetryReason:         eligibility.Reason,
		Attempt:             eligibility.Attempt,
		NextAttempt:         eligibility.NextAttempt,
		DeadLetter:          eligibility.DeadLetter || attrBool(state, "deadLetter"),
		RetryState:          state,
	}
	if !eligibility.NextRetryAt.IsZero() {
		resp.NextRetryAt = eligibility.NextRetryAt.Format(time.RFC3339)
	} else if value := strings.TrimSpace(cloudSyncPolicyAttrString(state["nextRetryAt"])); value != "" {
		resp.NextRetryAt = value
	}
	if resp.Attempt <= 0 {
		resp.Attempt = attrInt(state, "attempt")
	}
	if resp.NextAttempt <= 0 {
		resp.NextAttempt = attrInt(state, "nextAttempt")
	}
	return resp
}

func cloudItsmSubmitRetryQueueStatus(cfg *models.CloudItsmConfig, ticket models.CloudItsmTicket, now time.Time) (string, cloudItsmSubmitRetryEligibility) {
	state := modelResAttrs(ticket.ResponsePayload["submitRetry"])
	if attrBool(state, "deadLetter") && (cfg == nil || cfg.Status != models.CloudItsmConfigStatusEnabled) {
		reason := cloudSyncPolicyAttrString(state["reason"])
		if reason == "" {
			if cfg == nil {
				reason = "connector_missing"
			} else {
				reason = "connector_disabled"
			}
		}
		return "dead_letter", cloudItsmSubmitRetryEligibility{
			Attempt:     attrInt(state, "attempt"),
			NextAttempt: attrInt(state, "nextAttempt"),
			Reason:      reason,
			DeadLetter:  true,
			NextRetryAt: cloudItsmSubmitRetryTime(state["nextRetryAt"]),
		}
	}
	if cfg == nil {
		return "skipped", cloudItsmSubmitRetryEligibility{
			Attempt:     attrInt(state, "attempt"),
			NextAttempt: attrInt(state, "nextAttempt"),
			Reason:      "connector_missing",
			DeadLetter:  attrBool(state, "deadLetter"),
			NextRetryAt: cloudItsmSubmitRetryTime(state["nextRetryAt"]),
		}
	}
	if cfg.Status != models.CloudItsmConfigStatusEnabled {
		return "skipped", cloudItsmSubmitRetryEligibility{
			Attempt:     attrInt(state, "attempt"),
			NextAttempt: attrInt(state, "nextAttempt"),
			Reason:      "connector_disabled",
			DeadLetter:  attrBool(state, "deadLetter"),
			NextRetryAt: cloudItsmSubmitRetryTime(state["nextRetryAt"]),
		}
	}
	eligibility := cloudItsmTicketSubmitRetryEligibility(*cfg, ticket, now, false)
	if eligibility.DeadLetter || attrBool(state, "deadLetter") {
		if eligibility.Reason == "" {
			eligibility.Reason = cloudSyncPolicyAttrString(state["reason"])
		}
		if eligibility.Reason == "" {
			eligibility.Reason = "dead_letter"
		}
		return "dead_letter", eligibility
	}
	if eligibility.Eligible {
		return "due", eligibility
	}
	if eligibility.Reason == "not_due" {
		return "future", eligibility
	}
	return "skipped", eligibility
}

func cloudItsmSubmitRetryQueueNextRetryAt(item resps.CloudItsmSubmitRetryQueueItemResp) time.Time {
	return cloudItsmSubmitRetryTime(item.NextRetryAt)
}

func retryCloudItsmConnectorTicketSubmissions(c *ctx.ServiceContext, cfg *models.CloudItsmConfig, limit int, force bool) (models.ResAttrs, e.Error) {
	result := cloudItsmSubmitRetryResult()
	if cfg == nil || limit <= 0 {
		return result, nil
	}
	tickets := make([]models.CloudItsmTicket, 0)
	if err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and connector_id = ? and status = ?", c.OrgId, cfg.Id, models.CloudItsmTicketStatusFailed).
		Order("last_synced_at asc, updated_at asc").
		Limit(limit).
		Find(&tickets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	now := time.Now()
	for idx := range tickets {
		ticket := &tickets[idx]
		cloudItsmSubmitRetryAdd(result, "total", 1)
		item, err := retryCloudItsmSingleTicketSubmission(c, cfg, ticket, now, force, false)
		if err != nil {
			cloudItsmSubmitRetryAdd(result, "failed", 1)
			cloudItsmSubmitRetryAddTicketError(result, ticket.Id, err.Error())
			continue
		}
		if !attrBool(item, "retried") {
			if attrBool(item, "deadLetter") {
				cloudItsmSubmitRetryAdd(result, "deadLetter", 1)
			} else {
				cloudItsmSubmitRetryAdd(result, "skipped", 1)
			}
			continue
		}
		cloudItsmSubmitRetryAdd(result, "retried", 1)
		if cloudSyncPolicyAttrString(item["status"]) == models.CloudItsmTicketStatusSubmitted {
			cloudItsmSubmitRetryAdd(result, "submitted", 1)
		} else {
			cloudItsmSubmitRetryAdd(result, "failed", 1)
			cloudItsmSubmitRetryAddTicketError(result, ticket.Id, cloudSyncPolicyAttrString(item["errorMessage"]))
		}
	}
	return result, nil
}

func retryCloudItsmSingleTicketSubmission(c *ctx.ServiceContext, cfg *models.CloudItsmConfig, ticket *models.CloudItsmTicket, now time.Time, force bool, manualReplay bool) (models.ResAttrs, e.Error) {
	if cfg == nil || ticket == nil {
		return nil, e.New(e.BadParam, fmt.Errorf("ITSM 工单或连接器不存在"), http.StatusBadRequest)
	}
	eligibility := cloudItsmTicketSubmitRetryEligibility(*cfg, *ticket, now, force)
	if force && eligibility.DeadLetter && eligibility.Reason == "max_attempts_reached" {
		eligibility.Eligible = true
		eligibility.Reason = "forced_dead_letter_replay"
		eligibility.NextAttempt = eligibility.Attempt + 1
	}
	result := models.ResAttrs{
		"ticketId":     ticket.Id.String(),
		"eligible":     eligibility.Eligible,
		"reason":       eligibility.Reason,
		"attempt":      eligibility.Attempt,
		"nextAttempt":  eligibility.NextAttempt,
		"deadLetter":   eligibility.DeadLetter,
		"force":        force,
		"manualReplay": manualReplay,
	}
	if !eligibility.NextRetryAt.IsZero() {
		result["nextRetryAt"] = eligibility.NextRetryAt.Format(time.RFC3339)
	}
	if !eligibility.Eligible {
		return result, nil
	}
	submit := submitCloudItsmTicketWithAttempt(*cfg, ticket.RequestPayload, eligibility.NextAttempt, now)
	responsePayload := submit.ResponsePayload
	if responsePayload == nil {
		responsePayload = models.ResAttrs{}
	}
	responsePayload["manualReplay"] = manualReplay
	responsePayload["forcedReplay"] = force
	responsePayload["retryEligibilityReason"] = eligibility.Reason
	attrs := models.Attrs{
		"status":           submit.Status,
		"external_id":      submit.ExternalId,
		"external_key":     submit.ExternalKey,
		"external_url":     submit.ExternalUrl,
		"request_payload":  ticket.RequestPayload,
		"response_payload": responsePayload,
		"error_message":    submit.ErrorMessage,
		"submitted_at":     submit.SubmittedAt,
		"last_synced_at":   models.Time(now),
	}
	if submit.Status == models.CloudItsmTicketStatusSubmitted {
		attrs["error_message"] = ""
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
	ticket.ResponsePayload = responsePayload
	ticket.ErrorMessage = submit.ErrorMessage
	ticket.SubmittedAt = submit.SubmittedAt
	ticket.LastSyncedAt = models.Time(now)
	eventType := "itsm.ticket.retry_failed"
	eventLevel := models.CloudEventLevelError
	eventTitle := "ITSM 工单补偿提交失败"
	eventMessage := submit.ErrorMessage
	if submit.Status == models.CloudItsmTicketStatusSubmitted {
		eventType = "itsm.ticket.retry_submitted"
		eventLevel = models.CloudEventLevelInfo
		eventTitle = "ITSM 工单补偿提交成功"
		eventMessage = fmt.Sprintf("第 %d 次提交已成功", eligibility.NextAttempt)
	}
	cloudItsmTicketEvent(c, ticket, nil, eventType, eventLevel, eventTitle, eventMessage, responsePayload)
	result["retried"] = true
	result["status"] = submit.Status
	result["externalId"] = submit.ExternalId
	result["externalKey"] = submit.ExternalKey
	result["externalUrl"] = submit.ExternalUrl
	result["errorMessage"] = submit.ErrorMessage
	result["responsePayload"] = responsePayload
	return result, nil
}

func syncCloudItsmConnectorTicketStatuses(c *ctx.ServiceContext, cfg *models.CloudItsmConfig, limit int, force bool) (models.ResAttrs, e.Error) {
	result := cloudItsmStatusSyncResult()
	if cfg == nil {
		return result, nil
	}
	if limit <= 0 {
		return result, nil
	}
	interval := cloudItsmStatusSyncInterval(*cfg)
	tickets := make([]models.CloudItsmTicket, 0)
	query := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and connector_id = ? and status in (?)", c.OrgId, cfg.Id, []string{
			models.CloudItsmTicketStatusPending,
			models.CloudItsmTicketStatusSubmitted,
			models.CloudItsmTicketStatusInProgress,
		})
	if !force {
		query = query.Where("last_synced_at is null or last_synced_at <= ?", time.Now().Add(-interval))
	}
	if err := query.Order("last_synced_at asc").Limit(limit).Find(&tickets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	for idx := range tickets {
		ticket := &tickets[idx]
		cloudItsmStatusSyncAdd(result, "total", 1)
		poll, err := pollCloudItsmTicketStatus(*cfg, *ticket)
		if err != nil {
			cloudItsmStatusSyncAdd(result, "failed", 1)
			cloudItsmStatusSyncAddTicketError(result, ticket.Id, err.Error())
			continue
		}
		if poll.Status == "" {
			cloudItsmStatusSyncAdd(result, "skipped", 1)
			continue
		}
		updateForm := cloudItsmStatusPollUpdateForm(ticket, poll)
		if _, err := UpdateCloudItsmTicketStatus(c, updateForm); err != nil {
			cloudItsmStatusSyncAdd(result, "failed", 1)
			cloudItsmStatusSyncAddTicketError(result, ticket.Id, err.Error())
			continue
		}
		cloudItsmStatusSyncAdd(result, "synced", 1)
		if poll.Status == ticket.Status {
			cloudItsmStatusSyncAdd(result, "unchanged", 1)
		}
	}
	return result, nil
}

func pollCloudItsmTicketStatus(cfg models.CloudItsmConfig, ticket models.CloudItsmTicket) (cloudItsmStatusPollResult, e.Error) {
	endpoint := cloudItsmTicketStatusEndpoint(cfg, ticket)
	if endpoint == "" {
		return cloudItsmStatusPollResult{}, nil
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return cloudItsmStatusPollResult{}, e.New(e.BadParam, err, http.StatusBadRequest)
	}
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
		return cloudItsmStatusPollResult{}, e.New(e.InternalError, err)
	}
	defer resp.Body.Close()
	bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if readErr != nil {
		return cloudItsmStatusPollResult{}, e.New(e.IOError, readErr)
	}
	payload := models.ResAttrs{
		"endpoint":   endpoint,
		"statusCode": resp.StatusCode,
		"body":       string(bodyBytes),
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return cloudItsmStatusPollResult{ResponsePayload: payload}, e.New(e.InternalError, fmt.Errorf("ITSM 状态接口返回 HTTP %d", resp.StatusCode))
	}
	decoded := models.ResAttrs{}
	if len(bodyBytes) > 0 && json.Unmarshal(bodyBytes, &decoded) == nil {
		payload["json"] = decoded
	}
	rawStatus := cloudItsmExternalStatusString(decoded)
	status := cloudItsmNormalizeExternalStatus(cfg, rawStatus)
	if status == "" {
		return cloudItsmStatusPollResult{ResponsePayload: payload}, e.New(e.BadParam, fmt.Errorf("ITSM 状态接口未返回可识别状态"), http.StatusBadRequest)
	}
	return cloudItsmStatusPollResult{
		Status:          status,
		ExternalId:      firstNonEmpty(cloudItsmNestedAttrString(decoded, "externalId"), cloudItsmNestedAttrString(decoded, "id"), cloudItsmNestedAttrString(decoded, "result.sys_id"), cloudItsmNestedAttrString(decoded, "ticket.id"), cloudItsmNestedAttrString(decoded, "data.id")),
		ExternalKey:     firstNonEmpty(cloudItsmNestedAttrString(decoded, "externalKey"), cloudItsmNestedAttrString(decoded, "key"), cloudItsmNestedAttrString(decoded, "number"), cloudItsmNestedAttrString(decoded, "result.number"), cloudItsmNestedAttrString(decoded, "ticket.key"), cloudItsmNestedAttrString(decoded, "data.key")),
		ExternalUrl:     firstNonEmpty(cloudItsmNestedAttrString(decoded, "externalUrl"), cloudItsmNestedAttrString(decoded, "url"), cloudItsmNestedAttrString(decoded, "self"), cloudItsmNestedAttrString(decoded, "ticket.url"), cloudItsmNestedAttrString(decoded, "data.url")),
		ResponsePayload: payload,
	}, nil
}

func cloudItsmStatusPollUpdateForm(ticket *models.CloudItsmTicket, poll cloudItsmStatusPollResult) *forms.UpdateCloudItsmTicketStatusForm {
	payload := models.ResAttrs{}
	for key, value := range poll.ResponsePayload {
		payload[key] = value
	}
	payload["statusSync"] = models.ResAttrs{
		"source":       "poll",
		"syncedAt":     time.Now().Format(time.RFC3339),
		"remoteStatus": poll.Status,
	}
	form := &forms.UpdateCloudItsmTicketStatusForm{
		Id:          ticket.Id,
		Status:      poll.Status,
		ExternalId:  poll.ExternalId,
		ExternalKey: poll.ExternalKey,
		ExternalUrl: poll.ExternalUrl,
		Comment:     "周期同步外部 ITSM 状态",
		Payload:     payload,
	}
	values := url.Values{}
	values.Set("id", ticket.Id.String())
	values.Set("status", poll.Status)
	values.Set("comment", form.Comment)
	values.Set("payload", "true")
	if strings.TrimSpace(poll.ExternalId) != "" {
		values.Set("externalId", poll.ExternalId)
	}
	if strings.TrimSpace(poll.ExternalKey) != "" {
		values.Set("externalKey", poll.ExternalKey)
	}
	if strings.TrimSpace(poll.ExternalUrl) != "" {
		values.Set("externalUrl", poll.ExternalUrl)
	}
	form.Bind(values)
	return form
}

func cloudItsmStatusSyncEnabled(cfg models.CloudItsmConfig) bool {
	if cloudItsmParamBool(cfg.Metadata, "statusSyncEnabled", "statusPollingEnabled", "pollStatusEnabled") {
		return true
	}
	return cloudItsmConfiguredStatusEndpoint(cfg) != ""
}

func cloudItsmStatusSyncInterval(cfg models.CloudItsmConfig) time.Duration {
	seconds := attrInt(cfg.Metadata, "statusSyncIntervalSeconds")
	if seconds <= 0 {
		seconds = attrInt(cfg.Metadata, "statusPollingIntervalSeconds")
	}
	if seconds <= 0 {
		seconds = cloudItsmStatusSyncDefaultIntervalSeconds
	}
	if seconds < 60 {
		seconds = 60
	}
	if seconds > 86400 {
		seconds = 86400
	}
	return time.Duration(seconds) * time.Second
}

func cloudItsmTicketStatusEndpoint(cfg models.CloudItsmConfig, ticket models.CloudItsmTicket) string {
	tpl := cloudItsmConfiguredStatusEndpoint(cfg)
	if tpl == "" {
		switch cfg.Provider {
		case models.CloudItsmProviderJira:
			if strings.TrimSpace(ticket.ExternalKey) != "" {
				tpl = "/rest/api/2/issue/{externalKey}"
			}
		case models.CloudItsmProviderServiceNow:
			table := firstNonEmpty(attrString(cfg.Metadata, "statusTable"), attrString(cfg.Metadata, "ticketTable"), cfg.TicketType, "incident")
			if strings.TrimSpace(ticket.ExternalId) != "" {
				tpl = fmt.Sprintf("/api/now/table/%s/{externalId}", strings.Trim(table, "/"))
			}
		}
	}
	if tpl == "" {
		return ""
	}
	replaced := cloudItsmTicketStatusEndpointReplace(tpl, ticket)
	if parsed, err := url.Parse(replaced); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		return replaced
	}
	if cfg.BaseUrl == "" {
		return ""
	}
	return strings.TrimRight(cfg.BaseUrl, "/") + "/" + strings.TrimLeft(replaced, "/")
}

func cloudItsmConfiguredStatusEndpoint(cfg models.CloudItsmConfig) string {
	return strings.TrimSpace(firstNonEmpty(
		attrString(cfg.Metadata, "statusFetchUrl"),
		attrString(cfg.Metadata, "statusQueryUrl"),
		attrString(cfg.Metadata, "statusUrl"),
		attrString(cfg.Metadata, "getTicketUrl"),
		attrString(cfg.Metadata, "statusFetchPath"),
		attrString(cfg.Metadata, "statusQueryPath"),
		attrString(cfg.Metadata, "statusPath"),
		attrString(cfg.Metadata, "getTicketPath"),
	))
}

func cloudItsmTicketStatusEndpointReplace(tpl string, ticket models.CloudItsmTicket) string {
	values := map[string]string{
		"ticketId":    ticket.Id.String(),
		"id":          firstNonEmpty(ticket.ExternalId, ticket.ExternalKey, ticket.Id.String()),
		"externalId":  ticket.ExternalId,
		"externalKey": ticket.ExternalKey,
		"key":         ticket.ExternalKey,
		"number":      ticket.ExternalKey,
	}
	replaced := tpl
	for key, value := range values {
		replaced = strings.ReplaceAll(replaced, "{"+key+"}", url.PathEscape(value))
	}
	return replaced
}

func cloudItsmExternalStatusString(attrs models.ResAttrs) string {
	for _, path := range []string{
		"status", "state", "workflowStatus", "externalStatus", "incident_state",
		"result.status", "result.state", "result.incident_state",
		"ticket.status", "ticket.state", "data.status", "data.state",
		"fields.status.name", "fields.status.statusCategory.key",
	} {
		if value := cloudItsmNestedAttrString(attrs, path); value != "" {
			return value
		}
	}
	return ""
}

func cloudItsmNormalizeExternalStatus(cfg models.CloudItsmConfig, status string) string {
	raw := strings.TrimSpace(status)
	if raw == "" {
		return ""
	}
	statusMap := modelResAttrs(cfg.Metadata["statusMap"])
	for key, value := range statusMap {
		if strings.EqualFold(strings.TrimSpace(key), raw) {
			return cloudItsmNormalizeInternalTicketStatus(fmt.Sprintf("%v", value))
		}
	}
	normalized := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(raw, "_", " ")))
	switch normalized {
	case "pending", "queued", "waiting", "wait", "awaiting approval":
		return models.CloudItsmTicketStatusPending
	case "open", "new", "todo", "to do", "backlog", "created", "accepted", "1":
		return models.CloudItsmTicketStatusSubmitted
	case "in progress", "inprogress", "doing", "working", "active", "assigned", "implementing", "on hold", "2", "3":
		return models.CloudItsmTicketStatusInProgress
	case "resolved", "resolve", "done", "complete", "completed", "success", "succeeded", "fixed", "solved", "6":
		return models.CloudItsmTicketStatusResolved
	case "closed", "close", "closure", "7":
		return models.CloudItsmTicketStatusClosed
	case "canceled", "cancelled", "cancel", "rejected", "aborted", "void", "8":
		return models.CloudItsmTicketStatusCanceled
	case "failed", "failure", "error":
		return models.CloudItsmTicketStatusFailed
	}
	return cloudItsmNormalizeInternalTicketStatus(raw)
}

func cloudItsmNormalizeInternalTicketStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case models.CloudItsmTicketStatusPending,
		models.CloudItsmTicketStatusSubmitted,
		models.CloudItsmTicketStatusFailed,
		models.CloudItsmTicketStatusInProgress,
		models.CloudItsmTicketStatusResolved,
		models.CloudItsmTicketStatusClosed,
		models.CloudItsmTicketStatusCanceled:
		return strings.ToLower(strings.TrimSpace(status))
	}
	return ""
}

func cloudItsmNestedAttrString(attrs models.ResAttrs, path string) string {
	if attrs == nil || strings.TrimSpace(path) == "" {
		return ""
	}
	var current interface{} = attrs
	for _, part := range strings.Split(path, ".") {
		switch typed := current.(type) {
		case models.ResAttrs:
			current = typed[part]
		case map[string]interface{}:
			current = typed[part]
		default:
			return ""
		}
		if current == nil {
			return ""
		}
	}
	switch typed := current.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func cloudItsmStatusSyncResult() models.ResAttrs {
	return models.ResAttrs{
		"orgCount":         0,
		"connectorCount":   0,
		"connectorSkipped": 0,
		"total":            0,
		"synced":           0,
		"unchanged":        0,
		"skipped":          0,
		"failed":           0,
		"lockSkipped":      0,
		"errors":           []models.ResAttrs{},
	}
}

func cloudItsmStatusSyncAdd(result models.ResAttrs, key string, delta int) {
	result[key] = cloudItsmStatusSyncCount(result, key) + delta
}

func cloudItsmStatusSyncCount(result models.ResAttrs, key string) int {
	switch value := result[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func cloudItsmStatusSyncMerge(dst models.ResAttrs, src models.ResAttrs) {
	for _, key := range []string{"orgCount", "connectorSkipped", "total", "synced", "unchanged", "skipped", "failed", "lockSkipped"} {
		cloudItsmStatusSyncAdd(dst, key, cloudItsmStatusSyncCount(src, key))
	}
	for _, item := range cloudItsmStatusSyncErrors(src) {
		cloudItsmStatusSyncAddError(dst, item)
	}
}

func cloudItsmStatusSyncAddTicketError(result models.ResAttrs, ticketId models.Id, message string) {
	cloudItsmStatusSyncAddError(result, models.ResAttrs{
		"ticketId": ticketId.String(),
		"message":  message,
	})
}

func cloudItsmStatusSyncAddError(result models.ResAttrs, item models.ResAttrs) {
	errors := cloudItsmStatusSyncErrors(result)
	errors = append(errors, item)
	result["errors"] = errors
}

func cloudItsmStatusSyncErrors(result models.ResAttrs) []models.ResAttrs {
	switch typed := result["errors"].(type) {
	case []models.ResAttrs:
		return typed
	case []interface{}:
		items := make([]models.ResAttrs, 0, len(typed))
		for _, item := range typed {
			items = append(items, modelResAttrs(item))
		}
		return items
	default:
		return []models.ResAttrs{}
	}
}

func cloudItsmStatusSyncLockName(orgId models.Id) string {
	if orgId == "" {
		return "cloudiac:itsm_status_sync:unknown"
	}
	return fmt.Sprintf("cloudiac:itsm_status_sync:%s", orgId.String())
}

func cloudItsmAttachSubmitRetryResult(cfg models.CloudItsmConfig, result cloudItsmSubmitResult, attempt int, now time.Time) cloudItsmSubmitResult {
	if result.ResponsePayload == nil {
		result.ResponsePayload = models.ResAttrs{}
	}
	result.ResponsePayload["submitAttempt"] = attempt
	result.ResponsePayload["submitRetry"] = cloudItsmSubmitRetryState(cfg, result, attempt, now)
	return result
}

func cloudItsmSubmitRetryState(cfg models.CloudItsmConfig, result cloudItsmSubmitResult, attempt int, now time.Time) models.ResAttrs {
	if attempt <= 0 {
		attempt = 1
	}
	maxAttempts := cloudItsmSubmitRetryMaxAttempts(cfg)
	state := models.ResAttrs{
		"enabled":     cloudItsmSubmitRetryEnabled(cfg),
		"attempt":     attempt,
		"maxAttempts": maxAttempts,
		"status":      result.Status,
		"attemptedAt": now.Format(time.RFC3339),
		"lastError":   result.ErrorMessage,
	}
	if result.ResponsePayload != nil {
		if statusCode := attrInt(result.ResponsePayload, "statusCode"); statusCode > 0 {
			state["lastStatusCode"] = statusCode
		}
	}
	if result.Status != models.CloudItsmTicketStatusFailed {
		state["eligible"] = false
		state["reason"] = result.Status
		return state
	}
	if !cloudItsmSubmitRetryEnabled(cfg) {
		state["eligible"] = false
		state["reason"] = "disabled"
		return state
	}
	if attempt >= maxAttempts {
		state["eligible"] = false
		state["deadLetter"] = true
		state["reason"] = "max_attempts_reached"
		return state
	}
	backoff := cloudItsmSubmitRetryBackoff(cfg, attempt)
	nextRetryAt := now.Add(backoff)
	state["eligible"] = true
	state["reason"] = "waiting_retry"
	state["nextAttempt"] = attempt + 1
	state["backoffSeconds"] = int(backoff.Seconds())
	state["nextRetryAt"] = nextRetryAt.Format(time.RFC3339)
	return state
}

func cloudItsmTicketSubmitRetryEligibility(cfg models.CloudItsmConfig, ticket models.CloudItsmTicket, now time.Time, force bool) cloudItsmSubmitRetryEligibility {
	state := modelResAttrs(ticket.ResponsePayload["submitRetry"])
	attempt := attrInt(state, "attempt")
	if attempt <= 0 && ticket.Status == models.CloudItsmTicketStatusFailed {
		attempt = 1
	}
	eligibility := cloudItsmSubmitRetryEligibility{
		Attempt:     attempt,
		NextAttempt: attempt + 1,
		NextRetryAt: cloudItsmSubmitRetryTime(state["nextRetryAt"]),
	}
	if ticket.Status != models.CloudItsmTicketStatusFailed {
		eligibility.Reason = "status_not_failed"
		return eligibility
	}
	if cloudItsmTicketEndpoint(cfg) == "" {
		eligibility.Reason = "no_endpoint"
		return eligibility
	}
	if !cloudItsmSubmitRetryEnabled(cfg) {
		eligibility.Reason = "disabled"
		return eligibility
	}
	if len(ticket.RequestPayload) == 0 {
		eligibility.Reason = "missing_request_payload"
		return eligibility
	}
	if !cloudItsmSubmitRetryAllowExternalIdentity(cfg) &&
		firstNonEmpty(ticket.ExternalId, ticket.ExternalKey, ticket.ExternalUrl) != "" {
		eligibility.Reason = "external_identity_present"
		return eligibility
	}
	if attempt >= cloudItsmSubmitRetryMaxAttempts(cfg) {
		eligibility.Reason = "max_attempts_reached"
		eligibility.DeadLetter = true
		return eligibility
	}
	if !force && !eligibility.NextRetryAt.IsZero() && eligibility.NextRetryAt.After(now) {
		eligibility.Reason = "not_due"
		return eligibility
	}
	eligibility.Eligible = true
	eligibility.Reason = "due"
	return eligibility
}

func cloudItsmSubmitRetryEnabled(cfg models.CloudItsmConfig) bool {
	return cloudItsmMetadataBoolDefault(cfg.Metadata, true, "submitRetryEnabled", "ticketSubmitRetryEnabled", "retryFailedSubmitEnabled")
}

func cloudItsmSubmitRetryAllowExternalIdentity(cfg models.CloudItsmConfig) bool {
	return cloudItsmMetadataBoolDefault(cfg.Metadata, false, "submitRetryAllowExternalIdentity", "retryFailedSubmitAllowExternalIdentity")
}

func cloudItsmSubmitRetryMaxAttempts(cfg models.CloudItsmConfig) int {
	value := cloudItsmFirstAttrInt(cfg.Metadata, "submitRetryMaxAttempts", "ticketSubmitRetryMaxAttempts", "maxSubmitRetryAttempts", "maxRetryAttempts")
	if value <= 0 {
		value = cloudItsmSubmitRetryDefaultMaxAttempts
	}
	if value < 1 {
		value = 1
	}
	if value > cloudItsmSubmitRetryMaxAttemptsLimit {
		value = cloudItsmSubmitRetryMaxAttemptsLimit
	}
	return value
}

func cloudItsmSubmitRetryBackoff(cfg models.CloudItsmConfig, attempt int) time.Duration {
	baseSeconds := cloudItsmFirstAttrInt(cfg.Metadata, "submitRetryBackoffSeconds", "ticketSubmitRetryBackoffSeconds", "retryBackoffSeconds")
	if baseSeconds <= 0 {
		baseSeconds = cloudItsmSubmitRetryDefaultBackoffSeconds
	}
	if baseSeconds < 60 {
		baseSeconds = 60
	}
	maxSeconds := cloudItsmFirstAttrInt(cfg.Metadata, "submitRetryMaxBackoffSeconds", "ticketSubmitRetryMaxBackoffSeconds", "maxRetryBackoffSeconds")
	if maxSeconds <= 0 {
		maxSeconds = cloudItsmSubmitRetryDefaultMaxBackoff
	}
	if maxSeconds < baseSeconds {
		maxSeconds = baseSeconds
	}
	seconds := baseSeconds
	for i := 1; i < attempt; i++ {
		seconds *= 2
		if seconds >= maxSeconds {
			seconds = maxSeconds
			break
		}
	}
	return time.Duration(seconds) * time.Second
}

func cloudItsmFirstAttrInt(attrs models.ResAttrs, keys ...string) int {
	for _, key := range keys {
		if value := attrInt(attrs, key); value > 0 {
			return value
		}
	}
	return 0
}

func cloudItsmMetadataBoolDefault(attrs models.ResAttrs, defaultValue bool, keys ...string) bool {
	if attrs == nil {
		return defaultValue
	}
	for _, key := range keys {
		if _, ok := attrs[key]; ok {
			return cloudItsmParamBool(attrs, key)
		}
	}
	return defaultValue
}

func cloudItsmSubmitRetryTime(value interface{}) time.Time {
	raw := strings.TrimSpace(cloudSyncPolicyAttrString(value))
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func cloudItsmSubmitRetryResult() models.ResAttrs {
	return models.ResAttrs{
		"orgCount":         0,
		"connectorCount":   0,
		"connectorSkipped": 0,
		"total":            0,
		"retried":          0,
		"submitted":        0,
		"failed":           0,
		"skipped":          0,
		"deadLetter":       0,
		"lockSkipped":      0,
		"errors":           []models.ResAttrs{},
	}
}

func cloudItsmSubmitRetryAdd(result models.ResAttrs, key string, delta int) {
	result[key] = cloudItsmSubmitRetryCount(result, key) + delta
}

func cloudItsmSubmitRetryCount(result models.ResAttrs, key string) int {
	return cloudItsmStatusSyncCount(result, key)
}

func cloudItsmSubmitRetryMerge(dst models.ResAttrs, src models.ResAttrs) {
	for _, key := range []string{"orgCount", "connectorSkipped", "total", "retried", "submitted", "failed", "skipped", "deadLetter", "lockSkipped"} {
		cloudItsmSubmitRetryAdd(dst, key, cloudItsmSubmitRetryCount(src, key))
	}
	for _, item := range cloudItsmStatusSyncErrors(src) {
		cloudItsmSubmitRetryAddError(dst, item)
	}
}

func cloudItsmSubmitRetryAddTicketError(result models.ResAttrs, ticketId models.Id, message string) {
	cloudItsmSubmitRetryAddError(result, models.ResAttrs{
		"ticketId": ticketId.String(),
		"message":  message,
	})
}

func cloudItsmSubmitRetryAddError(result models.ResAttrs, item models.ResAttrs) {
	cloudItsmStatusSyncAddError(result, item)
}

func cloudItsmSubmitRetryLockName(orgId models.Id) string {
	if orgId == "" {
		return "cloudiac:itsm_submit_retry:unknown"
	}
	return fmt.Sprintf("cloudiac:itsm_submit_retry:%s", orgId.String())
}

func syncCloudItsmRiskRemediationStatus(c *ctx.ServiceContext, ticket *models.CloudItsmTicket, operation *models.CloudOperation, form *forms.UpdateCloudItsmTicketStatusForm) e.Error {
	if ticket == nil || operation == nil {
		return nil
	}
	params := modelResAttrs(operation.Params)
	if !cloudItsmOperationIsRiskRemediation(*operation, params) {
		return nil
	}
	riskId := cloudItsmOperationRiskId(params)
	if riskId == "" {
		return nil
	}
	nextStatus, ok := cloudItsmRiskStatusForTicketStatus(ticket.Status)
	if !ok {
		return nil
	}
	finding, err := getCloudRiskFinding(c, riskId)
	if err != nil {
		if err.Code() == e.BadParam {
			c.Logger().Warnf("skip ITSM risk remediation sync, risk %s not found: %v", riskId, err)
			return nil
		}
		return err
	}
	now := models.Time(time.Now())
	itsmEvidence := cloudItsmRiskRemediationEvidence(ticket, operation, form, nextStatus, now)
	driftAutoRepairEvidence := cloudItsmDriftAutoRepairForRiskRemediation(c, finding, operation, params, nextStatus, now)
	evidence := mergeCloudRiskEvidence(finding.Evidence, itsmEvidence)
	if len(driftAutoRepairEvidence) > 0 {
		evidence = mergeCloudRiskEvidence(evidence, driftAutoRepairEvidence)
	}
	attrs := models.Attrs{
		"status":             nextStatus,
		"evidence":           evidence,
		"suppression_reason": "",
	}
	if nextStatus == models.CloudRiskStatusResolved {
		attrs["resolved_at"] = now
	} else {
		attrs["resolved_at"] = models.Time{}
		attrs["suppressed_until"] = models.Time{}
	}
	if _, dbErr := c.DB().Model(&models.CloudRiskFinding{}).
		Where("id = ? and org_id = ?", finding.Id, c.OrgId).
		UpdateAttrs(attrs); dbErr != nil {
		return e.New(e.DBError, dbErr)
	}
	updated, err := getCloudRiskFinding(c, finding.Id)
	if err != nil {
		return err
	}
	comment := ""
	if form != nil {
		comment = form.Comment
	}
	eventPayload := models.ResAttrs{
		"previousStatus": finding.Status,
		"status":         nextStatus,
		"ticketId":       ticket.Id.String(),
		"ticketStatus":   ticket.Status,
		"operationId":    operation.Id.String(),
		"externalId":     ticket.ExternalId,
		"externalKey":    ticket.ExternalKey,
		"externalUrl":    ticket.ExternalUrl,
		"comment":        comment,
	}
	if len(driftAutoRepairEvidence) > 0 {
		eventPayload["driftAutoRepair"] = driftAutoRepairEvidence
	}
	cloudRiskEvent(c, updated, "risk.remediation_status_synced", "风险整改状态已同步", fmt.Sprintf("ITSM 工单状态 %s 已同步为风险状态 %s", ticket.Status, nextStatus), eventPayload)
	if cloudItsmDriftAutoRepairTriggered(driftAutoRepairEvidence) {
		cloudRiskEvent(c, updated, "risk.drift_auto_repair_triggered", "漂移自动修复任务已触发", fmt.Sprintf("已根据 ITSM 工单 %s 触发环境 %s 的漂移自动修复任务", ticket.Id, cloudSyncPolicyAttrString(driftAutoRepairEvidence["driftAutoRepairEnvId"])), driftAutoRepairEvidence)
	}
	cloudItsmDriftAutoRepairDispatchSlaEvent(c, updated, driftAutoRepairEvidence)
	return nil
}

func cloudItsmDriftAutoRepairForRiskRemediation(c *ctx.ServiceContext, finding *models.CloudRiskFinding, operation *models.CloudOperation, params models.ResAttrs, nextRiskStatus string, checkedAt models.Time) models.ResAttrs {
	if finding == nil || operation == nil || nextRiskStatus != models.CloudRiskStatusResolved {
		return nil
	}
	if !cloudItsmRiskFindingIsDriftRemediation(*finding, params) {
		return nil
	}
	envId := cloudItsmDriftAutoRepairEnvId(*finding, *operation, params)
	evidence := cloudItsmDriftAutoRepairEvidence(checkedAt, envId, false, "not_checked")
	defer cloudItsmDriftAutoRepairAppendTimeline(evidence)
	if envId == "" {
		evidence["driftAutoRepairSkippedReason"] = "missing_env_id"
		return evidence
	}

	env := models.Env{}
	if err := c.DB().Model(&models.Env{}).Where("id = ? and org_id = ?", envId, c.OrgId).First(&env); err != nil {
		if e.IsRecordNotFound(err) {
			evidence["driftAutoRepairSkippedReason"] = "env_not_found"
			return evidence
		}
		evidence["driftAutoRepairSkippedReason"] = "env_lookup_failed"
		evidence["driftAutoRepairError"] = err.Error()
		c.Logger().Warnf("lookup drift auto repair env %s failed: %v", envId, err)
		return evidence
	}

	evidence["driftAutoRepairProjectId"] = env.ProjectId.String()
	evidence["driftAutoRepairEnvStatus"] = env.Status
	evidence["driftAutoRepairOpenCronDrift"] = env.OpenCronDrift
	evidence["driftAutoRepairEnabled"] = env.AutoRepairDrift
	for key, value := range cloudItsmDriftAutoRepairSlaPolicyEvidence(cloudItsmDriftAutoRepairSlaPolicyForEnv(env)) {
		evidence[key] = value
	}
	if env.Status != models.EnvStatusActive {
		evidence["driftAutoRepairSkippedReason"] = "env_not_active"
		return evidence
	}
	if env.Locked {
		evidence["driftAutoRepairSkippedReason"] = "env_locked"
		return evidence
	}
	if !env.OpenCronDrift {
		evidence["driftAutoRepairSkippedReason"] = "cron_drift_disabled"
		return evidence
	}
	if !env.AutoRepairDrift {
		evidence["driftAutoRepairSkippedReason"] = "auto_repair_disabled"
		return evidence
	}
	if env.LastTaskId == "" {
		evidence["driftAutoRepairSkippedReason"] = "missing_source_task"
		return evidence
	}

	evidence["driftAutoRepairSourceTaskId"] = env.LastTaskId.String()
	pending, err := services.ListPendingCronTask(c.DB(), env.Id)
	if err != nil {
		evidence["driftAutoRepairSkippedReason"] = "pending_task_lookup_failed"
		evidence["driftAutoRepairError"] = err.Error()
		c.Logger().Warnf("lookup pending drift task for env %s failed: %v", env.Id, err)
		return evidence
	}
	if pending {
		evidence["driftAutoRepairSkippedReason"] = "pending_drift_task_exists"
		return evidence
	}

	sourceTask, err := services.GetTaskById(c.DB(), env.LastTaskId)
	if err != nil {
		evidence["driftAutoRepairSkippedReason"] = "source_task_lookup_failed"
		evidence["driftAutoRepairError"] = err.Error()
		c.Logger().Warnf("lookup drift source task %s failed: %v", env.LastTaskId, err)
		return evidence
	}
	approvalPolicy := cloudItsmDriftAutoRepairApprovalPolicyForEnv(env, cloudItsmDriftAutoRepairApprovalContextRepair)
	for key, value := range cloudItsmDriftAutoRepairApprovalPolicyEvidence(approvalPolicy) {
		evidence[key] = value
	}
	sourceTask.Type = models.TaskTypeApply
	sourceTask.IsDriftTask = true
	newTask, err := services.CloneNewDriftTaskWithAutoApprove(c.DB(), *sourceTask, &env, approvalPolicy.AutoApprove)
	if err != nil {
		evidence["driftAutoRepairSkippedReason"] = "clone_drift_task_failed"
		evidence["driftAutoRepairError"] = err.Error()
		c.Logger().Warnf("clone drift auto repair task for env %s failed: %v", env.Id, err)
		return evidence
	}
	evidence["driftAutoRepairTriggered"] = true
	evidence["driftAutoRepairSkippedReason"] = ""
	evidence["driftAutoRepairTaskId"] = newTask.Id.String()
	evidence["driftAutoRepairTaskType"] = newTask.Type
	evidence["driftAutoRepairTaskStatus"] = newTask.Status
	evidence["driftAutoRepairTaskAutoApprove"] = newTask.AutoApprove
	return evidence
}

func cloudItsmRiskFindingIsDriftRemediation(finding models.CloudRiskFinding, params models.ResAttrs) bool {
	if finding.Source == models.CloudRiskSourceDrift || finding.RuleKey == "terraform_drift_detected" {
		return true
	}
	if cloudItsmParamBool(params, "driftRemediation", "driftRemediationRequired") {
		return true
	}
	if cloudItsmParamString(params, "requestType") == models.CloudOperationActionDriftRemediation {
		return true
	}
	return cloudItsmParamString(params, "targetState") == "iac_desired_state"
}

func cloudItsmDriftAutoRepairEnvId(finding models.CloudRiskFinding, operation models.CloudOperation, params models.ResAttrs) models.Id {
	if finding.EnvId != "" {
		return finding.EnvId
	}
	if operation.EnvId != "" {
		return operation.EnvId
	}
	for _, key := range []string{"envId", "environmentId"} {
		if value := strings.TrimSpace(cloudSyncPolicyAttrString(params[key])); value != "" {
			return models.Id(value)
		}
	}
	return ""
}

func cloudItsmDriftAutoRepairEvidence(checkedAt models.Time, envId models.Id, triggered bool, reason string) models.ResAttrs {
	evidence := models.ResAttrs{
		"driftAutoRepairCheckedAt": time.Time(checkedAt).Format(time.RFC3339),
		"driftAutoRepairSource":    "itsm_risk_remediation_resolved",
		"driftAutoRepairEnvId":     envId.String(),
		"driftAutoRepairTriggered": triggered,
	}
	if reason != "" {
		evidence["driftAutoRepairSkippedReason"] = reason
	}
	return evidence
}

func cloudItsmDriftAutoRepairTriggered(evidence models.ResAttrs) bool {
	if evidence == nil {
		return false
	}
	switch typed := evidence["driftAutoRepairTriggered"].(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func cloudItsmDriftAutoRepairApprovalPolicyForEnv(env models.Env, context string) cloudItsmDriftAutoRepairApprovalPolicy {
	if context == "" {
		context = cloudItsmDriftAutoRepairApprovalContextRepair
	}
	policy := cloudItsmDefaultDriftAutoRepairApprovalPolicy(env, context)
	attrs := cloudItsmDriftAutoRepairApprovalPolicyAttrs(env)
	if len(attrs) == 0 {
		return policy
	}

	policy.Source = "env_extra_data"
	if roles := normalizeStringList(cloudSyncPolicyAttrStringSlice(firstNonNil(
		attrs["approverRoles"],
		attrs["approvalRoles"],
		attrs["approvers"],
	))); len(roles) > 0 {
		policy.ApproverRoles = roles
	}
	if reason := strings.TrimSpace(cloudSyncPolicyAttrString(firstNonNil(attrs["reason"], attrs["approvalReason"]))); reason != "" {
		policy.Reason = reason
	}

	modeRaw := strings.TrimSpace(cloudSyncPolicyAttrString(firstNonNil(attrs["mode"], attrs["approvalMode"], attrs["approval_mode"])))
	if modeRaw != "" {
		mode := cloudItsmNormalizeDriftAutoRepairApprovalMode(modeRaw)
		if mode == "" {
			policy.Mode = modeRaw
			policy.AutoApprove = false
			policy.RequiresApproval = true
			policy.Reason = "invalid_policy_mode_requires_approval"
			return policy
		}
		policy.Mode = mode
		switch mode {
		case cloudItsmDriftAutoRepairApprovalModeRequireApproval:
			policy.AutoApprove = false
			policy.Reason = "policy_requires_approval"
		case cloudItsmDriftAutoRepairApprovalModeAutoApprove:
			policy.AutoApprove = true
			policy.Reason = "policy_auto_approve"
		case cloudItsmDriftAutoRepairApprovalModeRetryRequireOnly:
			policy.AutoApprove = env.AutoApproval
			policy.Reason = "policy_inherits_env_auto_approval"
			if context == cloudItsmDriftAutoRepairApprovalContextRetry {
				policy.AutoApprove = false
				policy.Reason = "policy_retry_requires_approval"
			}
		default:
			policy.AutoApprove = env.AutoApproval
			policy.Reason = "policy_inherits_env_auto_approval"
		}
	}

	if value, ok := cloudItsmDriftAutoRepairApprovalAttr(attrs, "autoApprove", "autoApproval", "auto_approve"); ok {
		policy.AutoApprove = cloudSyncPolicyAttrBool(value)
		if policy.AutoApprove {
			policy.Reason = "policy_auto_approve_override"
		} else {
			policy.Reason = "policy_auto_approve_disabled"
		}
	}
	if context == cloudItsmDriftAutoRepairApprovalContextRetry {
		if value, ok := cloudItsmDriftAutoRepairApprovalAttr(attrs, "retryAutoApprove", "retryAutoApproval", "retry_auto_approve"); ok {
			policy.AutoApprove = cloudSyncPolicyAttrBool(value)
			if policy.AutoApprove {
				policy.Reason = "policy_retry_auto_approve_override"
			} else {
				policy.Reason = "policy_retry_auto_approve_disabled"
			}
		}
	}
	if value, ok := cloudItsmDriftAutoRepairApprovalAttr(attrs, "requireApproval", "approvalRequired", "require_approval"); ok && cloudSyncPolicyAttrBool(value) {
		policy.AutoApprove = false
		policy.Reason = "policy_requires_approval"
	}
	if context == cloudItsmDriftAutoRepairApprovalContextRetry {
		if value, ok := cloudItsmDriftAutoRepairApprovalAttr(attrs, "retryRequireApproval", "retryApprovalRequired", "retry_require_approval"); ok && cloudSyncPolicyAttrBool(value) {
			policy.AutoApprove = false
			policy.Reason = "policy_retry_requires_approval"
		}
	}
	policy.RequiresApproval = !policy.AutoApprove
	return policy
}

func cloudItsmDefaultDriftAutoRepairApprovalPolicy(env models.Env, context string) cloudItsmDriftAutoRepairApprovalPolicy {
	autoApprove := env.AutoApproval
	source := "env_auto_approval"
	mode := cloudItsmDriftAutoRepairApprovalModeInheritEnv
	reason := "env_auto_approval_disabled"
	if autoApprove {
		reason = "env_auto_approval_enabled"
	}
	if context == cloudItsmDriftAutoRepairApprovalContextRetry {
		autoApprove = false
		source = "default_retry_guardrail"
		mode = cloudItsmDriftAutoRepairApprovalModeRetryRequireOnly
		reason = "retry_after_failed_auto_repair_requires_approval"
	}
	return cloudItsmDriftAutoRepairApprovalPolicy{
		Context:          context,
		Source:           source,
		Mode:             mode,
		AutoApprove:      autoApprove,
		RequiresApproval: !autoApprove,
		Reason:           reason,
	}
}

func cloudItsmDriftAutoRepairApprovalPolicyAttrs(env models.Env) models.ResAttrs {
	extraData := cloudItsmEnvExtraDataAttrs(env)
	for _, key := range []string{
		"driftAutoRepairApprovalPolicy",
		"driftAutoRepairApproval",
		"driftApprovalPolicy",
	} {
		if attrs := modelResAttrs(extraData[key]); len(attrs) > 0 {
			return attrs
		}
	}
	return nil
}

func cloudItsmEnvExtraDataAttrs(env models.Env) models.ResAttrs {
	if env.ExtraData.IsNull() {
		return nil
	}
	attrs := models.ResAttrs{}
	if err := json.Unmarshal(env.ExtraData, &attrs); err != nil {
		return nil
	}
	return attrs
}

func cloudItsmDriftAutoRepairApprovalAttr(attrs models.ResAttrs, keys ...string) (interface{}, bool) {
	for _, key := range keys {
		if value, ok := attrs[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func cloudItsmNormalizeDriftAutoRepairApprovalMode(value string) string {
	normalized := strings.NewReplacer("-", "_", " ", "_").Replace(strings.ToLower(strings.TrimSpace(value)))
	switch normalized {
	case "", cloudItsmDriftAutoRepairApprovalModeInheritEnv, "env", "environment", "env_auto_approval", "inherit_auto_approval":
		return cloudItsmDriftAutoRepairApprovalModeInheritEnv
	case cloudItsmDriftAutoRepairApprovalModeRequireApproval, "manual", "manual_approval", "approval_required", "always_require", "always_require_approval":
		return cloudItsmDriftAutoRepairApprovalModeRequireApproval
	case cloudItsmDriftAutoRepairApprovalModeAutoApprove, "auto", "auto_approval", "always_auto", "always_auto_approve", "skip_approval":
		return cloudItsmDriftAutoRepairApprovalModeAutoApprove
	case cloudItsmDriftAutoRepairApprovalModeRetryRequireOnly, "retry_require", "retry_manual", "retry_manual_approval":
		return cloudItsmDriftAutoRepairApprovalModeRetryRequireOnly
	default:
		return ""
	}
}

func cloudItsmDriftAutoRepairApprovalPolicyEvidence(policy cloudItsmDriftAutoRepairApprovalPolicy) models.ResAttrs {
	evidence := models.ResAttrs{
		"driftAutoRepairApprovalPolicyContext": policy.Context,
		"driftAutoRepairApprovalPolicySource":  policy.Source,
		"driftAutoRepairApprovalPolicyMode":    policy.Mode,
		"driftAutoRepairApprovalAutoApprove":   policy.AutoApprove,
		"driftAutoRepairApprovalRequired":      policy.RequiresApproval,
		"driftAutoRepairApprovalReason":        policy.Reason,
	}
	if len(policy.ApproverRoles) > 0 {
		evidence["driftAutoRepairApprovalRoles"] = policy.ApproverRoles
	}
	return evidence
}

func cloudItsmDriftAutoRepairSlaPolicyForEnv(env models.Env) cloudItsmDriftAutoRepairSlaPolicy {
	policy := cloudItsmDefaultDriftAutoRepairSlaPolicy()
	attrs := cloudItsmDriftAutoRepairSlaPolicyAttrs(env)
	if len(attrs) == 0 {
		return policy
	}
	policy.Source = "env_extra_data"
	if value := cloudSyncPolicyAttrInt(firstNonNil(attrs["approvalDueMinutes"], attrs["approvalSlaMinutes"], attrs["approvalMinutes"])); value > 0 {
		policy.ApprovalDueMinutes = value
	}
	if value := cloudSyncPolicyAttrInt(firstNonNil(attrs["approvalDueSeconds"], attrs["approvalSlaSeconds"])); value > 0 {
		policy.ApprovalDueMinutes = value / 60
		if policy.ApprovalDueMinutes <= 0 {
			policy.ApprovalDueMinutes = 1
		}
	}
	if value := cloudSyncPolicyAttrInt(firstNonNil(attrs["rollbackReviewDueMinutes"], attrs["rollbackDueMinutes"], attrs["rollbackReviewMinutes"])); value > 0 {
		policy.RollbackReviewDueMinutes = value
	}
	if value := cloudSyncPolicyAttrInt(firstNonNil(attrs["rollbackReviewDueSeconds"], attrs["rollbackDueSeconds"])); value > 0 {
		policy.RollbackReviewDueMinutes = value / 60
		if policy.RollbackReviewDueMinutes <= 0 {
			policy.RollbackReviewDueMinutes = 1
		}
	}
	if value := cloudSyncPolicyAttrInt(firstNonNil(attrs["dueSoonMinutes"], attrs["warningBeforeMinutes"], attrs["slaWarningMinutes"])); value > 0 {
		policy.DueSoonMinutes = value
	}
	if routes := normalizeStringList(cloudSyncPolicyAttrStringSlice(firstNonNil(attrs["notificationRoutes"], attrs["routes"], attrs["escalationRoutes"]))); len(routes) > 0 {
		policy.NotificationRoutes = routes
	}
	if assignees := normalizeStringList(cloudSyncPolicyAttrStringSlice(firstNonNil(attrs["notificationAssignees"], attrs["assignees"], attrs["escalationAssignees"]))); len(assignees) > 0 {
		policy.NotificationAssignees = assignees
	}
	if owners := normalizeStringList(cloudSyncPolicyAttrStringSlice(firstNonNil(attrs["ownerRoles"], attrs["owners"], attrs["escalationOwners"]))); len(owners) > 0 {
		policy.OwnerRoles = owners
	}
	if value, ok := cloudItsmDriftAutoRepairApprovalAttr(attrs, "autoTicket", "itsmAutoTicket", "slaAutoTicket"); ok {
		policy.AutoTicket = cloudSyncPolicyAttrBool(value)
	}
	return policy
}

func cloudItsmDefaultDriftAutoRepairSlaPolicy() cloudItsmDriftAutoRepairSlaPolicy {
	return cloudItsmDriftAutoRepairSlaPolicy{
		Source:                   "default",
		ApprovalDueMinutes:       240,
		RollbackReviewDueMinutes: 1440,
		DueSoonMinutes:           30,
		NotificationRoutes:       []string{"drift_auto_repair", "drift_auto_repair_sla"},
		OwnerRoles:               []string{"sre", "platform_ops"},
	}
}

func cloudItsmDriftAutoRepairSlaPolicyAttrs(env models.Env) models.ResAttrs {
	extraData := cloudItsmEnvExtraDataAttrs(env)
	for _, key := range []string{
		"driftAutoRepairSlaPolicy",
		"driftAutoRepairSLA",
		"driftAutoRepairSla",
		"driftSlaPolicy",
	} {
		if attrs := modelResAttrs(extraData[key]); len(attrs) > 0 {
			return attrs
		}
	}
	return nil
}

func cloudItsmDriftAutoRepairSlaPolicyEvidence(policy cloudItsmDriftAutoRepairSlaPolicy) models.ResAttrs {
	evidence := models.ResAttrs{
		"driftAutoRepairSlaPolicySource":             policy.Source,
		"driftAutoRepairSlaApprovalDueMinutes":       policy.ApprovalDueMinutes,
		"driftAutoRepairSlaRollbackReviewDueMinutes": policy.RollbackReviewDueMinutes,
		"driftAutoRepairSlaDueSoonMinutes":           policy.DueSoonMinutes,
		"driftAutoRepairSlaNotificationRoutes":       policy.NotificationRoutes,
		"driftAutoRepairSlaNotificationOwnerRoles":   policy.OwnerRoles,
		"driftAutoRepairSlaAutoTicketOnEscalation":   policy.AutoTicket,
	}
	if len(policy.NotificationAssignees) > 0 {
		evidence["driftAutoRepairSlaNotificationAssignees"] = policy.NotificationAssignees
	}
	return evidence
}

func cloudItsmDriftAutoRepairAppendSlaEvidence(evidence models.ResAttrs, now time.Time) {
	if evidence == nil {
		return
	}
	action, targetType, targetId, startedAt, dueMinutes := cloudItsmDriftAutoRepairSlaAction(evidence)
	evidence["driftAutoRepairSlaAction"] = action
	evidence["driftAutoRepairSlaTargetType"] = targetType
	evidence["driftAutoRepairSlaTargetId"] = targetId
	evidence["driftAutoRepairSlaEvaluatedAt"] = now.Format(time.RFC3339)
	if action == "none" || startedAt.IsZero() || dueMinutes <= 0 {
		evidence["driftAutoRepairSlaStatus"] = "not_required"
		evidence["driftAutoRepairSlaEscalationRequired"] = false
		evidence["driftAutoRepairSlaDueAt"] = ""
		evidence["driftAutoRepairSlaStartedAt"] = ""
		evidence["driftAutoRepairSlaMinutesRemaining"] = 0
		evidence["driftAutoRepairSlaEscalationReason"] = ""
		return
	}

	dueAt := startedAt.Add(time.Duration(dueMinutes) * time.Minute)
	remaining := int(dueAt.Sub(now).Minutes())
	if dueAt.After(now) && dueAt.Sub(now) < time.Minute {
		remaining = 1
	}
	status := "pending"
	escalationRequired := false
	escalationReason := ""
	if !now.Before(dueAt) {
		status = "breached"
		escalationRequired = true
		escalationReason = action + "_sla_breached"
	} else if dueSoonMinutes := cloudItsmDriftAutoRepairSlaDueSoonMinutes(evidence); dueSoonMinutes > 0 && remaining <= dueSoonMinutes {
		status = "due_soon"
		escalationReason = action + "_sla_due_soon"
	}
	evidence["driftAutoRepairSlaStatus"] = status
	evidence["driftAutoRepairSlaEscalationRequired"] = escalationRequired
	evidence["driftAutoRepairSlaStartedAt"] = startedAt.Format(time.RFC3339)
	evidence["driftAutoRepairSlaDueAt"] = dueAt.Format(time.RFC3339)
	evidence["driftAutoRepairSlaDueMinutes"] = dueMinutes
	evidence["driftAutoRepairSlaMinutesRemaining"] = remaining
	evidence["driftAutoRepairSlaEscalationReason"] = escalationReason
}

func cloudItsmDriftAutoRepairSlaAction(evidence models.ResAttrs) (string, string, string, time.Time, int) {
	taskId := cloudSyncPolicyAttrString(evidence["driftAutoRepairTaskId"])
	retryTaskId := cloudSyncPolicyAttrString(evidence["driftAutoRepairRetryTaskId"])
	if cloudSyncPolicyAttrBool(evidence["driftAutoRepairRetryApprovalRequired"]) && retryTaskId != "" {
		return "approve_retry", "task", retryTaskId,
			cloudItsmDriftAutoRepairSlaTime(firstNonNil(evidence["driftAutoRepairRetryCheckedAt"], evidence["driftAutoRepairTaskLastSyncedAt"], evidence["driftAutoRepairCheckedAt"])),
			cloudItsmDriftAutoRepairSlaApprovalMinutes(evidence)
	}
	if cloudSyncPolicyAttrBool(evidence["driftAutoRepairApprovalRequired"]) &&
		!cloudSyncPolicyAttrBool(evidence["driftAutoRepairApprovalAutoApprove"]) &&
		taskId != "" {
		return "approve_repair", "task", taskId,
			cloudItsmDriftAutoRepairSlaTime(firstNonNil(evidence["driftAutoRepairCheckedAt"], evidence["driftAutoRepairTaskLastSyncedAt"])),
			cloudItsmDriftAutoRepairSlaApprovalMinutes(evidence)
	}
	if cloudItsmDriftAutoRepairRollbackRequired(evidence) {
		targetId := cloudSyncPolicyAttrString(firstNonNil(evidence["driftAutoRepairRollbackTaskId"], evidence["driftAutoRepairLastFailedTaskId"], evidence["driftAutoRepairTaskId"]))
		return "review_rollback", "task", targetId,
			cloudItsmDriftAutoRepairSlaTime(firstNonNil(evidence["driftAutoRepairRollbackEvaluatedAt"], evidence["driftAutoRepairTaskLastSyncedAt"])),
			cloudItsmDriftAutoRepairSlaRollbackMinutes(evidence)
	}
	return "none", "", "", time.Time{}, 0
}

func cloudItsmDriftAutoRepairSlaApprovalMinutes(evidence models.ResAttrs) int {
	if value := cloudSyncPolicyAttrInt(evidence["driftAutoRepairSlaApprovalDueMinutes"]); value > 0 {
		return value
	}
	return cloudItsmDefaultDriftAutoRepairSlaPolicy().ApprovalDueMinutes
}

func cloudItsmDriftAutoRepairSlaRollbackMinutes(evidence models.ResAttrs) int {
	if value := cloudSyncPolicyAttrInt(evidence["driftAutoRepairSlaRollbackReviewDueMinutes"]); value > 0 {
		return value
	}
	return cloudItsmDefaultDriftAutoRepairSlaPolicy().RollbackReviewDueMinutes
}

func cloudItsmDriftAutoRepairSlaDueSoonMinutes(evidence models.ResAttrs) int {
	if value := cloudSyncPolicyAttrInt(evidence["driftAutoRepairSlaDueSoonMinutes"]); value > 0 {
		return value
	}
	return cloudItsmDefaultDriftAutoRepairSlaPolicy().DueSoonMinutes
}

func cloudItsmDriftAutoRepairSlaTime(value interface{}) time.Time {
	raw := strings.TrimSpace(cloudSyncPolicyAttrString(value))
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func cloudItsmDriftAutoRepairAppendTimeline(evidence models.ResAttrs) {
	if evidence == nil {
		return
	}
	cloudItsmDriftAutoRepairAppendSlaEvidence(evidence, time.Now())
	evidence["driftAutoRepairTimeline"] = cloudItsmDriftAutoRepairTimeline(evidence)
	evidence["driftAutoRepairRecommendations"] = cloudItsmDriftAutoRepairRecommendations(evidence)
}

func cloudItsmDriftAutoRepairTimeline(evidence models.ResAttrs) []models.ResAttrs {
	timeline := make([]models.ResAttrs, 0, 8)
	add := func(stage string, title string, status string, at string, attrs models.ResAttrs) {
		if strings.TrimSpace(title) == "" {
			return
		}
		item := models.ResAttrs{
			"stage":  stage,
			"title":  title,
			"status": status,
		}
		if at != "" {
			item["time"] = at
		}
		for key, value := range attrs {
			if value != nil && cloudSyncPolicyAttrString(value) != "" {
				item[key] = value
			}
		}
		timeline = append(timeline, item)
	}

	checkedAt := cloudSyncPolicyAttrString(evidence["driftAutoRepairCheckedAt"])
	triggerStatus := "checked"
	triggerReason := cloudSyncPolicyAttrString(evidence["driftAutoRepairSkippedReason"])
	if cloudItsmDriftAutoRepairTriggered(evidence) {
		triggerStatus = "triggered"
	} else if triggerReason != "" && triggerReason != "not_checked" {
		triggerStatus = "skipped"
	}
	add("trigger_check", "自动修复触发检查", triggerStatus, checkedAt, models.ResAttrs{
		"envId":  evidence["driftAutoRepairEnvId"],
		"reason": triggerReason,
	})

	if sourceTaskId := cloudSyncPolicyAttrString(evidence["driftAutoRepairSourceTaskId"]); sourceTaskId != "" {
		add("source_task", "定位源部署任务", "found", checkedAt, models.ResAttrs{
			"taskId": sourceTaskId,
		})
	}

	if mode := cloudSyncPolicyAttrString(evidence["driftAutoRepairApprovalPolicyMode"]); mode != "" {
		status := "approval_required"
		if cloudSyncPolicyAttrBool(evidence["driftAutoRepairApprovalAutoApprove"]) {
			status = "auto_approved"
		}
		policyAt := checkedAt
		if context := cloudSyncPolicyAttrString(evidence["driftAutoRepairApprovalPolicyContext"]); context == cloudItsmDriftAutoRepairApprovalContextRetry {
			if retryCheckedAt := cloudSyncPolicyAttrString(evidence["driftAutoRepairRetryCheckedAt"]); retryCheckedAt != "" {
				policyAt = retryCheckedAt
			}
		}
		add("approval_policy", "审批策略判定", status, policyAt, models.ResAttrs{
			"mode":    mode,
			"source":  evidence["driftAutoRepairApprovalPolicySource"],
			"reason":  evidence["driftAutoRepairApprovalReason"],
			"context": evidence["driftAutoRepairApprovalPolicyContext"],
		})
	}

	taskId := cloudSyncPolicyAttrString(evidence["driftAutoRepairTaskId"])
	retryTaskId := cloudSyncPolicyAttrString(evidence["driftAutoRepairRetryTaskId"])
	lastFailedTaskId := cloudSyncPolicyAttrString(evidence["driftAutoRepairLastFailedTaskId"])
	if cloudItsmDriftAutoRepairTriggered(evidence) && taskId != "" && taskId != retryTaskId {
		add("repair_task_created", "创建漂移自动修复任务", cloudSyncPolicyAttrString(evidence["driftAutoRepairTaskStatus"]), checkedAt, models.ResAttrs{
			"taskId":      taskId,
			"taskType":    evidence["driftAutoRepairTaskType"],
			"autoApprove": evidence["driftAutoRepairTaskAutoApprove"],
		})
	}

	if startedAt := cloudSyncPolicyAttrString(evidence["driftAutoRepairTaskStartedAt"]); startedAt != "" && lastFailedTaskId == "" {
		add("task_started", "自动修复任务开始执行", "running", startedAt, models.ResAttrs{
			"taskId": taskId,
		})
	}
	if endedAt := cloudSyncPolicyAttrString(evidence["driftAutoRepairTaskEndedAt"]); endedAt != "" {
		stage := "task_finished"
		title := "自动修复任务结束"
		finishedTaskId := taskId
		status := cloudSyncPolicyAttrString(evidence["driftAutoRepairTaskStatus"])
		if lastFailedTaskId != "" {
			stage = "last_failed_task_finished"
			title = "上次自动修复任务结束"
			finishedTaskId = lastFailedTaskId
			status = cloudSyncPolicyAttrString(evidence["driftAutoRepairLastFailedTaskStatus"])
		}
		add(stage, title, status, endedAt, models.ResAttrs{
			"taskId":  finishedTaskId,
			"message": evidence["driftAutoRepairTaskMessage"],
		})
	}

	if syncedAt := cloudSyncPolicyAttrString(evidence["driftAutoRepairTaskLastSyncedAt"]); syncedAt != "" {
		add("risk_result_synced", "同步自动修复结果到风险", cloudSyncPolicyAttrString(evidence["driftAutoRepairTaskRiskStatus"]), syncedAt, models.ResAttrs{
			"riskStatus": evidence["driftAutoRepairTaskRiskStatus"],
			"taskId":     firstNonNil(evidence["driftAutoRepairLastFailedTaskId"], evidence["driftAutoRepairTaskId"]),
		})
	}

	if retryCheckedAt := cloudSyncPolicyAttrString(evidence["driftAutoRepairRetryCheckedAt"]); retryCheckedAt != "" {
		status := cloudSyncPolicyAttrString(evidence["driftAutoRepairRetryApprovalStatus"])
		if status == "" && cloudSyncPolicyAttrString(evidence["driftAutoRepairRetrySkippedReason"]) != "" {
			status = "skipped"
		}
		if status == "" {
			status = "checked"
		}
		title := "失败重试检查"
		if retryTaskId != "" {
			title = "创建失败重试任务"
		}
		add("retry", title, status, retryCheckedAt, models.ResAttrs{
			"taskId":      retryTaskId,
			"attempt":     evidence["driftAutoRepairRetryAttempt"],
			"maxAttempts": evidence["driftAutoRepairRetryMaxAttempts"],
			"reason":      evidence["driftAutoRepairRetrySkippedReason"],
		})
	}

	if rollbackAt := cloudSyncPolicyAttrString(evidence["driftAutoRepairRollbackEvaluatedAt"]); rollbackAt != "" {
		status := "no_rollback_required"
		if cloudItsmDriftAutoRepairRollbackRequired(evidence) {
			status = "rollback_review_required"
		}
		add("rollback_evaluated", "回滚策略评估", status, rollbackAt, models.ResAttrs{
			"strategy":    evidence["driftAutoRepairRollbackStrategy"],
			"safetyLevel": evidence["driftAutoRepairRollbackSafetyLevel"],
			"taskId":      evidence["driftAutoRepairRollbackTaskId"],
		})
	}
	if slaAction := cloudSyncPolicyAttrString(evidence["driftAutoRepairSlaAction"]); slaAction != "" && slaAction != "none" {
		add("sla", "审批/评审 SLA", cloudSyncPolicyAttrString(evidence["driftAutoRepairSlaStatus"]), cloudSyncPolicyAttrString(evidence["driftAutoRepairSlaEvaluatedAt"]), models.ResAttrs{
			"action":           slaAction,
			"taskId":           evidence["driftAutoRepairSlaTargetId"],
			"dueAt":            evidence["driftAutoRepairSlaDueAt"],
			"minutesRemaining": evidence["driftAutoRepairSlaMinutesRemaining"],
			"reason":           evidence["driftAutoRepairSlaEscalationReason"],
		})
	}
	return timeline
}

func cloudItsmDriftAutoRepairRecommendations(evidence models.ResAttrs) []models.ResAttrs {
	recommendations := make([]models.ResAttrs, 0, 6)
	add := func(action string, title string, priority string, targetType string, targetId interface{}, reason string) {
		if strings.TrimSpace(title) == "" {
			return
		}
		item := models.ResAttrs{
			"action":   action,
			"title":    title,
			"priority": priority,
			"reason":   reason,
		}
		if targetType != "" {
			item["targetType"] = targetType
		}
		if targetId != nil && cloudSyncPolicyAttrString(targetId) != "" {
			item["targetId"] = targetId
		}
		recommendations = append(recommendations, item)
	}

	envId := cloudSyncPolicyAttrString(evidence["driftAutoRepairEnvId"])
	taskId := cloudSyncPolicyAttrString(evidence["driftAutoRepairTaskId"])
	retryTaskId := cloudSyncPolicyAttrString(evidence["driftAutoRepairRetryTaskId"])
	lastFailedTaskId := cloudSyncPolicyAttrString(evidence["driftAutoRepairLastFailedTaskId"])
	skippedReason := cloudSyncPolicyAttrString(evidence["driftAutoRepairSkippedReason"])
	if envId != "" {
		add("open_env", "查看环境和漂移配置", "medium", "env", envId, "drift_env_context")
	}

	if skippedReason != "" && skippedReason != "not_checked" && !cloudItsmDriftAutoRepairTriggered(evidence) {
		switch skippedReason {
		case "cron_drift_disabled":
			add("enable_drift_detection", "开启环境漂移检测", "high", "env", envId, skippedReason)
		case "auto_repair_disabled":
			add("enable_auto_repair", "开启环境自动修复", "high", "env", envId, skippedReason)
		case "env_locked":
			add("unlock_env", "评估并解除环境锁定", "high", "env", envId, skippedReason)
		case "env_not_active":
			add("check_env_status", "恢复环境到活跃状态后重试", "high", "env", envId, skippedReason)
		case "missing_source_task":
			add("run_apply_once", "先完成一次环境部署任务以生成漂移修复源任务", "high", "env", envId, skippedReason)
		case "pending_drift_task_exists":
			add("check_pending_drift_task", "查看已有漂移任务处理进度", "medium", "env", envId, skippedReason)
		default:
			add("review_auto_repair_config", "检查漂移自动修复配置", "medium", "env", envId, skippedReason)
		}
		return recommendations
	}

	if cloudItsmDriftAutoRepairTriggered(evidence) && taskId != "" && taskId != retryTaskId {
		add("open_repair_task", "查看漂移自动修复任务", "high", "task", taskId, "repair_task_created")
	}
	if cloudSyncPolicyAttrBool(evidence["driftAutoRepairApprovalRequired"]) && !cloudSyncPolicyAttrBool(evidence["driftAutoRepairApprovalAutoApprove"]) {
		targetId := taskId
		title := "审批漂移自动修复任务"
		if retryTaskId != "" {
			targetId = retryTaskId
			title = "审批漂移自动修复重试任务"
		}
		add("approve_task", title, "high", "task", targetId, "approval_required")
	}
	if lastFailedTaskId != "" || cloudSyncPolicyAttrBool(evidence["driftAutoRepairTaskFailed"]) {
		targetId := firstNonNil(evidence["driftAutoRepairLastFailedTaskId"], evidence["driftAutoRepairTaskId"])
		add("inspect_task_logs", "查看失败任务日志和 Terraform 输出", "high", "task", targetId, "repair_task_failed")
	}
	if cloudSyncPolicyAttrBool(evidence["driftAutoRepairRetryApprovalRequired"]) && retryTaskId != "" {
		add("approve_retry_task", "审批失败后的漂移修复重试任务", "high", "task", retryTaskId, "retry_approval_required")
	}
	if cloudItsmDriftAutoRepairRollbackRequired(evidence) {
		targetId := firstNonNil(evidence["driftAutoRepairRollbackTaskId"], evidence["driftAutoRepairLastFailedTaskId"], evidence["driftAutoRepairTaskId"])
		add("review_rollback", "评审云端资源、Terraform state 和 IaC 代码是否需要回滚", "critical", "task", targetId, "rollback_review_required")
		add("create_gitops_pr", "如需回退，通过 GitOps/IaC PR 提交修复或回滚变更", "high", "env", envId, "gitops_pr_required")
	}
	if cloudSyncPolicyAttrBool(evidence["driftAutoRepairTaskCompleted"]) && !cloudItsmDriftAutoRepairRollbackRequired(evidence) {
		add("verify_drift_clean", "确认风险关闭并观察下一次漂移检测结果", "medium", "env", envId, "repair_completed")
	}
	return recommendations
}

func cloudItsmDriftAutoRepairDispatchSlaEvent(c *ctx.ServiceContext, finding *models.CloudRiskFinding, evidence models.ResAttrs) {
	if c == nil || finding == nil || evidence == nil {
		return
	}
	action := cloudSyncPolicyAttrString(evidence["driftAutoRepairSlaAction"])
	if action == "" || action == "none" {
		return
	}
	status := cloudSyncPolicyAttrString(evidence["driftAutoRepairSlaStatus"])
	if status == "" || status == "not_required" {
		return
	}
	eventType := "risk.drift_auto_repair_approval_notification_requested"
	title := cloudItsmDriftAutoRepairSlaTitle(action, false)
	message := cloudItsmDriftAutoRepairSlaMessage(action, evidence)
	if cloudSyncPolicyAttrBool(evidence["driftAutoRepairSlaEscalationRequired"]) {
		eventType = "risk.drift_auto_repair_sla_escalated"
		title = cloudItsmDriftAutoRepairSlaTitle(action, true)
		message = "漂移自动修复审批/评审已超过 SLA，请升级处理"
	}
	cloudRiskEvent(c, finding, eventType, title, message, cloudItsmDriftAutoRepairSlaEventPayload(evidence))
}

func cloudItsmDriftAutoRepairSlaTitle(action string, escalated bool) string {
	if escalated {
		switch action {
		case "approve_retry":
			return "漂移自动修复重试审批 SLA 已升级"
		case "review_rollback":
			return "漂移自动修复回滚评审 SLA 已升级"
		default:
			return "漂移自动修复审批 SLA 已升级"
		}
	}
	switch action {
	case "approve_retry":
		return "漂移自动修复重试等待审批"
	case "review_rollback":
		return "漂移自动修复回滚评审通知"
	default:
		return "漂移自动修复等待审批"
	}
}

func cloudItsmDriftAutoRepairSlaMessage(action string, evidence models.ResAttrs) string {
	dueAt := cloudSyncPolicyAttrString(evidence["driftAutoRepairSlaDueAt"])
	targetId := cloudSyncPolicyAttrString(evidence["driftAutoRepairSlaTargetId"])
	switch action {
	case "approve_retry":
		return fmt.Sprintf("漂移自动修复重试任务 %s 等待审批，SLA 截止时间 %s", targetId, dueAt)
	case "review_rollback":
		return fmt.Sprintf("漂移自动修复回滚评审任务 %s 等待处理，SLA 截止时间 %s", targetId, dueAt)
	default:
		return fmt.Sprintf("漂移自动修复任务 %s 等待审批，SLA 截止时间 %s", targetId, dueAt)
	}
}

func cloudItsmDriftAutoRepairSlaEventPayload(evidence models.ResAttrs) models.ResAttrs {
	action := cloudSyncPolicyAttrString(evidence["driftAutoRepairSlaAction"])
	payload := models.ResAttrs{
		"action":                   action,
		"status":                   evidence["driftAutoRepairSlaStatus"],
		"targetType":               evidence["driftAutoRepairSlaTargetType"],
		"targetId":                 evidence["driftAutoRepairSlaTargetId"],
		"startedAt":                evidence["driftAutoRepairSlaStartedAt"],
		"dueAt":                    evidence["driftAutoRepairSlaDueAt"],
		"dueMinutes":               evidence["driftAutoRepairSlaDueMinutes"],
		"minutesRemaining":         evidence["driftAutoRepairSlaMinutesRemaining"],
		"escalationRequired":       evidence["driftAutoRepairSlaEscalationRequired"],
		"escalationReason":         evidence["driftAutoRepairSlaEscalationReason"],
		"policySource":             evidence["driftAutoRepairSlaPolicySource"],
		"approvalDueMinutes":       evidence["driftAutoRepairSlaApprovalDueMinutes"],
		"rollbackReviewDueMinutes": evidence["driftAutoRepairSlaRollbackReviewDueMinutes"],
		"envId":                    evidence["driftAutoRepairEnvId"],
		"taskId":                   evidence["driftAutoRepairTaskId"],
		"retryTaskId":              evidence["driftAutoRepairRetryTaskId"],
		"lastFailedTaskId":         evidence["driftAutoRepairLastFailedTaskId"],
		"rollbackTaskId":           evidence["driftAutoRepairRollbackTaskId"],
	}
	routes := normalizeStringList(cloudSyncPolicyAttrStringSlice(evidence["driftAutoRepairSlaNotificationRoutes"]))
	if action != "" {
		routes = append(routes, "drift_auto_repair_"+action)
	}
	if cloudSyncPolicyAttrBool(evidence["driftAutoRepairSlaEscalationRequired"]) {
		routes = append(routes, "sla_breach")
		payload["notificationEscalationReason"] = firstNonEmpty(
			cloudSyncPolicyAttrString(evidence["driftAutoRepairSlaEscalationReason"]),
			action+"_sla_breached",
		)
		if cloudSyncPolicyAttrBool(evidence["driftAutoRepairSlaAutoTicketOnEscalation"]) {
			payload["itsmAutoTicket"] = true
			payload["itsmPriority"] = models.CloudOperationRiskHigh
			payload["itsmTitle"] = cloudItsmDriftAutoRepairSlaTitle(action, true)
			payload["itsmDescription"] = cloudItsmDriftAutoRepairSlaMessage(action, evidence)
		}
	}
	payload["notificationRoutes"] = uniqueStrings(normalizeStringList(routes))
	if owners := normalizeStringList(cloudSyncPolicyAttrStringSlice(evidence["driftAutoRepairSlaNotificationOwnerRoles"])); len(owners) > 0 {
		payload["notificationOwner"] = owners
	}
	if assignees := normalizeStringList(cloudSyncPolicyAttrStringSlice(evidence["driftAutoRepairSlaNotificationAssignees"])); len(assignees) > 0 {
		payload["notificationAssignees"] = assignees
	}
	return payload
}

func SyncCloudRiskDriftAutoRepairTaskResult(task *models.Task) e.Error {
	if task == nil || !cloudItsmTaskIsDriftAutoRepair(task) || !task.Exited() {
		return nil
	}
	c := cloudItsmSystemContextForTask(task)
	findings := make([]models.CloudRiskFinding, 0)
	if err := c.DB().Model(&models.CloudRiskFinding{}).
		Where("org_id = ?", task.OrgId).
		Where("JSON_UNQUOTE(JSON_EXTRACT(evidence, '$.driftAutoRepairTaskId')) = ?", task.Id.String()).
		Find(&findings); err != nil {
		return e.New(e.DBError, err)
	}
	if len(findings) == 0 {
		return nil
	}
	syncedAt := models.Time(time.Now())
	for idx := range findings {
		finding := &findings[idx]
		nextStatus := cloudItsmRiskStatusForDriftAutoRepairTaskStatus(task.Status, finding.Status)
		resultEvidence := cloudItsmDriftAutoRepairTaskResultEvidence(task, nextStatus, syncedAt)
		retryApprovalEvidence := cloudItsmDriftAutoRepairRetryApprovalForFailedTask(c, finding, task, syncedAt)
		evidence := mergeCloudRiskEvidence(finding.Evidence, resultEvidence)
		if len(retryApprovalEvidence) > 0 {
			evidence = mergeCloudRiskEvidence(evidence, retryApprovalEvidence)
		}
		cloudItsmDriftAutoRepairAppendTimeline(evidence)
		attrs := models.Attrs{
			"status":             nextStatus,
			"evidence":           evidence,
			"suppression_reason": "",
		}
		if nextStatus == models.CloudRiskStatusResolved {
			if time.Time(finding.ResolvedAt).IsZero() {
				attrs["resolved_at"] = syncedAt
			}
		} else {
			attrs["resolved_at"] = models.Time{}
			attrs["suppressed_until"] = models.Time{}
		}
		if _, dbErr := c.DB().Model(&models.CloudRiskFinding{}).
			Where("id = ? and org_id = ?", finding.Id, task.OrgId).
			UpdateAttrs(attrs); dbErr != nil {
			return e.New(e.DBError, dbErr)
		}
		updated, err := getCloudRiskFinding(c, finding.Id)
		if err != nil {
			return err
		}
		cloudRiskEvent(c, updated, "risk.drift_auto_repair_result_synced", "漂移自动修复结果已同步", fmt.Sprintf("漂移自动修复任务 %s 已结束，任务状态 %s 同步为风险状态 %s", task.Id, task.Status, nextStatus), models.ResAttrs{
			"previousStatus": finding.Status,
			"status":         nextStatus,
			"taskId":         task.Id.String(),
			"taskStatus":     task.Status,
			"taskType":       task.Type,
			"taskSource":     task.Source,
			"taskMessage":    task.Message,
			"envId":          task.EnvId.String(),
		})
		if cloudItsmDriftAutoRepairRollbackRequired(resultEvidence) {
			cloudRiskEvent(c, updated, "risk.drift_auto_repair_rollback_strategy_recorded", "漂移自动修复回滚策略已记录", fmt.Sprintf("漂移自动修复任务 %s 未完全成功，已记录回滚评审策略", task.Id), models.ResAttrs{
				"taskId":              task.Id.String(),
				"taskStatus":          task.Status,
				"rollbackStrategy":    resultEvidence["driftAutoRepairRollbackStrategy"],
				"rollbackSafetyLevel": resultEvidence["driftAutoRepairRollbackSafetyLevel"],
				"rollbackHint":        resultEvidence["driftAutoRepairRollbackHint"],
				"rollbackNextAction":  resultEvidence["driftAutoRepairRollbackNextAction"],
			})
		}
		if cloudItsmDriftAutoRepairRetryApprovalRequested(retryApprovalEvidence) {
			cloudRiskEvent(c, updated, "risk.drift_auto_repair_retry_approval_requested", "漂移自动修复重试等待审批", fmt.Sprintf("漂移自动修复任务 %s 失败后已创建待审批重试任务 %s", task.Id, cloudSyncPolicyAttrString(retryApprovalEvidence["driftAutoRepairRetryTaskId"])), retryApprovalEvidence)
		}
		cloudItsmDriftAutoRepairDispatchSlaEvent(c, updated, evidence)
	}
	return nil
}

func cloudItsmSystemContextForTask(task *models.Task) *ctx.ServiceContext {
	rc := &cloudItsmSystemRequestContext{}
	c := ctx.NewServiceContext(rc)
	if task != nil {
		c.OrgId = task.OrgId
		c.ProjectId = task.ProjectId
	}
	c.UserId = consts.SysUserId
	c.Email = consts.DefaultSysEmail
	c.Username = consts.DefaultSysName
	return c
}

func cloudItsmTaskIsDriftAutoRepair(task *models.Task) bool {
	return task != nil && task.IsDriftTask && task.Type == models.TaskTypeApply && task.Source == consts.TaskSourceDriftApply
}

func cloudItsmRiskStatusForDriftAutoRepairTaskStatus(taskStatus string, fallback string) string {
	switch taskStatus {
	case models.TaskComplete:
		return models.CloudRiskStatusResolved
	case models.TaskFailed, models.TaskAborted, models.TaskRejected:
		return models.CloudRiskStatusOpen
	default:
		if fallback != "" {
			return fallback
		}
		return models.CloudRiskStatusInProgress
	}
}

func cloudItsmDriftAutoRepairRetryApprovalForFailedTask(c *ctx.ServiceContext, finding *models.CloudRiskFinding, task *models.Task, checkedAt models.Time) models.ResAttrs {
	if c == nil || finding == nil || task == nil || !cloudItsmDriftAutoRepairTaskNeedsRetryApproval(task) {
		return nil
	}
	maxAttempts := cloudItsmDriftAutoRepairRetryMaxAttempts(finding.Evidence)
	attempt := cloudItsmDriftAutoRepairRetryAttempt(finding.Evidence)
	evidence := cloudItsmDriftAutoRepairRetryApprovalEvidence(checkedAt, task, attempt+1, maxAttempts, false, "not_checked")
	if attempt >= maxAttempts {
		evidence["driftAutoRepairRetrySkippedReason"] = "max_retry_attempts_reached"
		return evidence
	}
	if existingRetryTaskId := models.Id(cloudSyncPolicyAttrString(finding.Evidence["driftAutoRepairRetryTaskId"])); existingRetryTaskId != "" && existingRetryTaskId != task.Id {
		evidence["driftAutoRepairRetrySkippedReason"] = "retry_task_exists"
		return evidence
	}
	if task.EnvId == "" {
		evidence["driftAutoRepairRetrySkippedReason"] = "missing_env_id"
		return evidence
	}

	env := models.Env{}
	if err := c.DB().Model(&models.Env{}).Where("id = ? and org_id = ?", task.EnvId, task.OrgId).First(&env); err != nil {
		if e.IsRecordNotFound(err) {
			evidence["driftAutoRepairRetrySkippedReason"] = "env_not_found"
			return evidence
		}
		evidence["driftAutoRepairRetrySkippedReason"] = "env_lookup_failed"
		evidence["driftAutoRepairRetryError"] = err.Error()
		c.Logger().Warnf("lookup drift auto repair retry env %s failed: %v", task.EnvId, err)
		return evidence
	}

	evidence["driftAutoRepairProjectId"] = env.ProjectId.String()
	evidence["driftAutoRepairEnvId"] = env.Id.String()
	evidence["driftAutoRepairEnvStatus"] = env.Status
	evidence["driftAutoRepairOpenCronDrift"] = env.OpenCronDrift
	evidence["driftAutoRepairEnabled"] = env.AutoRepairDrift
	for key, value := range cloudItsmDriftAutoRepairSlaPolicyEvidence(cloudItsmDriftAutoRepairSlaPolicyForEnv(env)) {
		evidence[key] = value
	}
	if env.Status != models.EnvStatusActive {
		evidence["driftAutoRepairRetrySkippedReason"] = "env_not_active"
		return evidence
	}
	if env.Locked {
		evidence["driftAutoRepairRetrySkippedReason"] = "env_locked"
		return evidence
	}
	if !env.OpenCronDrift {
		evidence["driftAutoRepairRetrySkippedReason"] = "cron_drift_disabled"
		return evidence
	}
	if !env.AutoRepairDrift {
		evidence["driftAutoRepairRetrySkippedReason"] = "auto_repair_disabled"
		return evidence
	}
	pending, err := services.ListPendingCronTask(c.DB(), env.Id)
	if err != nil {
		evidence["driftAutoRepairRetrySkippedReason"] = "pending_task_lookup_failed"
		evidence["driftAutoRepairRetryError"] = err.Error()
		c.Logger().Warnf("lookup pending drift retry task for env %s failed: %v", env.Id, err)
		return evidence
	}
	if pending {
		evidence["driftAutoRepairRetrySkippedReason"] = "pending_drift_task_exists"
		return evidence
	}

	approvalPolicy := cloudItsmDriftAutoRepairApprovalPolicyForEnv(env, cloudItsmDriftAutoRepairApprovalContextRetry)
	for key, value := range cloudItsmDriftAutoRepairApprovalPolicyEvidence(approvalPolicy) {
		evidence[key] = value
	}
	retryTask, err := services.CloneNewDriftTaskWithAutoApprove(c.DB(), *task, &env, approvalPolicy.AutoApprove)
	if err != nil {
		evidence["driftAutoRepairRetrySkippedReason"] = "clone_retry_task_failed"
		evidence["driftAutoRepairRetryError"] = err.Error()
		c.Logger().Warnf("clone drift auto repair retry task for env %s failed: %v", env.Id, err)
		return evidence
	}
	evidence["driftAutoRepairRetryApprovalRequired"] = approvalPolicy.RequiresApproval
	evidence["driftAutoRepairRetryApprovalRequested"] = approvalPolicy.RequiresApproval
	evidence["driftAutoRepairRetrySkippedReason"] = ""
	evidence["driftAutoRepairRetryTaskId"] = retryTask.Id.String()
	evidence["driftAutoRepairRetryTaskStatus"] = retryTask.Status
	if approvalPolicy.RequiresApproval {
		evidence["driftAutoRepairRetryApprovalStatus"] = "pending"
	} else {
		evidence["driftAutoRepairRetryApprovalStatus"] = "auto_approved"
	}
	evidence["driftAutoRepairTaskId"] = retryTask.Id.String()
	evidence["driftAutoRepairTaskStatus"] = retryTask.Status
	evidence["driftAutoRepairTaskType"] = retryTask.Type
	evidence["driftAutoRepairTaskSource"] = retryTask.Source
	evidence["driftAutoRepairTaskAutoApprove"] = retryTask.AutoApprove
	if approvalPolicy.RequiresApproval {
		evidence["driftAutoRepairTaskMessage"] = "自动修复失败，已创建待审批重试任务"
	} else {
		evidence["driftAutoRepairTaskMessage"] = "自动修复失败，已按审批策略创建自动审批重试任务"
	}
	evidence["driftAutoRepairTaskCompleted"] = false
	evidence["driftAutoRepairTaskFailed"] = false
	evidence["driftAutoRepairTaskRetryPendingApproval"] = approvalPolicy.RequiresApproval
	return evidence
}

func cloudItsmDriftAutoRepairTaskNeedsRetryApproval(task *models.Task) bool {
	if task == nil {
		return false
	}
	switch task.Status {
	case models.TaskFailed, models.TaskAborted, models.TaskRejected:
		return true
	default:
		return false
	}
}

func cloudItsmDriftAutoRepairRetryAttempt(evidence models.ResAttrs) int {
	attempt := cloudSyncPolicyAttrInt(evidence["driftAutoRepairRetryAttempt"])
	if attempt < 0 {
		return 0
	}
	return attempt
}

func cloudItsmDriftAutoRepairRetryMaxAttempts(evidence models.ResAttrs) int {
	maxAttempts := cloudSyncPolicyAttrInt(firstNonNil(
		evidence["driftAutoRepairRetryMaxAttempts"],
		evidence["driftAutoRepairMaxRetryAttempts"],
	))
	if maxAttempts <= 0 {
		return cloudItsmDriftAutoRepairRetryDefaultMax
	}
	if maxAttempts > cloudItsmDriftAutoRepairRetryMax {
		return cloudItsmDriftAutoRepairRetryMax
	}
	return maxAttempts
}

func cloudItsmDriftAutoRepairRetryApprovalEvidence(checkedAt models.Time, task *models.Task, attempt int, maxAttempts int, requested bool, reason string) models.ResAttrs {
	evidence := models.ResAttrs{
		"driftAutoRepairRetryCheckedAt":           time.Time(checkedAt).Format(time.RFC3339),
		"driftAutoRepairRetrySource":              "drift_auto_repair_task_failed",
		"driftAutoRepairRetryApprovalRequired":    true,
		"driftAutoRepairRetryApprovalRequested":   requested,
		"driftAutoRepairRetryAttempt":             attempt,
		"driftAutoRepairRetryMaxAttempts":         maxAttempts,
		"driftAutoRepairRetryApprovalStatus":      "pending",
		"driftAutoRepairRetryReason":              "auto_repair_task_failed",
		"driftAutoRepairTaskRetryPendingApproval": requested,
	}
	if reason != "" {
		evidence["driftAutoRepairRetrySkippedReason"] = reason
	}
	if task != nil {
		evidence["driftAutoRepairRetryFromTaskId"] = task.Id.String()
		evidence["driftAutoRepairRetryFromTaskStatus"] = task.Status
		evidence["driftAutoRepairLastFailedTaskId"] = task.Id.String()
		evidence["driftAutoRepairLastFailedTaskStatus"] = task.Status
		evidence["driftAutoRepairLastFailedTaskMessage"] = task.Message
		evidence["driftAutoRepairEnvId"] = task.EnvId.String()
	}
	return evidence
}

func cloudItsmDriftAutoRepairRetryApprovalRequested(evidence models.ResAttrs) bool {
	if evidence == nil {
		return false
	}
	return cloudSyncPolicyAttrBool(evidence["driftAutoRepairRetryApprovalRequested"])
}

func cloudItsmDriftAutoRepairRollbackRequired(evidence models.ResAttrs) bool {
	if evidence == nil {
		return false
	}
	return cloudSyncPolicyAttrBool(evidence["driftAutoRepairRollbackRequired"])
}

func cloudItsmDriftAutoRepairTaskResultEvidence(task *models.Task, nextRiskStatus string, syncedAt models.Time) models.ResAttrs {
	evidence := models.ResAttrs{
		"driftAutoRepairTaskLastSyncedAt": time.Time(syncedAt).Format(time.RFC3339),
		"driftAutoRepairTaskRiskStatus":   nextRiskStatus,
	}
	if task == nil {
		return evidence
	}
	evidence["driftAutoRepairTaskId"] = task.Id.String()
	evidence["driftAutoRepairTaskStatus"] = task.Status
	evidence["driftAutoRepairTaskType"] = task.Type
	evidence["driftAutoRepairTaskSource"] = task.Source
	evidence["driftAutoRepairTaskMessage"] = task.Message
	evidence["driftAutoRepairTaskCompleted"] = task.Status == models.TaskComplete
	evidence["driftAutoRepairTaskFailed"] = task.Status == models.TaskFailed ||
		task.Status == models.TaskAborted ||
		task.Status == models.TaskRejected
	evidence["driftAutoRepairTaskRetryPendingApproval"] = false
	for key, value := range cloudItsmDriftAutoRepairRollbackEvidence(task, syncedAt) {
		evidence[key] = value
	}
	if task.StartAt != nil {
		evidence["driftAutoRepairTaskStartedAt"] = time.Time(*task.StartAt).Format(time.RFC3339)
	}
	if task.EndAt != nil {
		evidence["driftAutoRepairTaskEndedAt"] = time.Time(*task.EndAt).Format(time.RFC3339)
	}
	return evidence
}

func cloudItsmDriftAutoRepairRollbackEvidence(task *models.Task, evaluatedAt models.Time) models.ResAttrs {
	evidence := models.ResAttrs{
		"driftAutoRepairRollbackEvaluatedAt": time.Time(evaluatedAt).Format(time.RFC3339),
		"driftAutoRepairRollbackSource":      "drift_auto_repair_task_result",
		"driftAutoRepairRollbackRequired":    false,
	}
	if task == nil {
		evidence["driftAutoRepairRollbackStrategy"] = "not_evaluated"
		evidence["driftAutoRepairRollbackHint"] = "未获取到漂移自动修复任务，无法评估回滚策略"
		return evidence
	}
	evidence["driftAutoRepairRollbackTaskId"] = task.Id.String()
	evidence["driftAutoRepairRollbackTaskStatus"] = task.Status
	if summary := cloudItsmTaskResultChangeSummary(task.PlanResult); len(summary) > 0 {
		evidence["driftAutoRepairRollbackPlanChanges"] = summary
	}
	if summary := cloudItsmTaskResultChangeSummary(task.Result); len(summary) > 0 {
		evidence["driftAutoRepairRollbackApplyChanges"] = summary
	}

	partialApplySuspected := cloudItsmTaskResultHasChanges(task.Result)
	evidence["driftAutoRepairRollbackPartialApplySuspected"] = partialApplySuspected
	switch task.Status {
	case models.TaskComplete:
		evidence["driftAutoRepairRollbackStrategy"] = "no_platform_rollback_required"
		evidence["driftAutoRepairRollbackSafetyLevel"] = "low"
		evidence["driftAutoRepairRollbackHint"] = "漂移自动修复已按 IaC 期望态完成，平台不需要自动回滚"
		evidence["driftAutoRepairRollbackNextAction"] = "如业务需要回退，请提交 GitOps/IaC 变更申请，经 PR review 和流水线通过后重新执行"
		evidence["driftAutoRepairRollbackRequiresApproval"] = false
	case models.TaskFailed, models.TaskAborted, models.TaskRejected:
		evidence["driftAutoRepairRollbackRequired"] = true
		evidence["driftAutoRepairRollbackStrategy"] = "gitops_iac_change_or_manual_review_required"
		evidence["driftAutoRepairRollbackSafetyLevel"] = "medium"
		evidence["driftAutoRepairRollbackRequiresApproval"] = true
		evidence["driftAutoRepairRollbackHint"] = "漂移自动修复未完全成功，需先评审云端和 state 当前状态，再决定重试或通过 GitOps/IaC 变更回滚"
		evidence["driftAutoRepairRollbackNextAction"] = "优先审批并执行自动修复重试任务；如确认需要回退，提交 GitOps/IaC 变更 PR 并通过流水线后执行"
		if partialApplySuspected {
			evidence["driftAutoRepairRollbackSafetyLevel"] = "high"
			evidence["driftAutoRepairRollbackHint"] = "漂移自动修复失败且已检测到资源变更结果，可能存在部分 apply，必须人工核对云端资源、Terraform state 和 IaC 代码后再处理"
		}
	default:
		evidence["driftAutoRepairRollbackStrategy"] = "wait_task_finished"
		evidence["driftAutoRepairRollbackSafetyLevel"] = "medium"
		evidence["driftAutoRepairRollbackHint"] = "漂移自动修复任务尚未进入最终态，暂不执行回滚评估"
		evidence["driftAutoRepairRollbackNextAction"] = "等待任务完成后由结果回写链路重新评估"
		evidence["driftAutoRepairRollbackRequiresApproval"] = false
	}
	return evidence
}

func cloudItsmTaskResultChangeSummary(result models.TaskResult) models.ResAttrs {
	summary := models.ResAttrs{}
	if result.ResAdded != nil {
		summary["added"] = *result.ResAdded
	}
	if result.ResChanged != nil {
		summary["changed"] = *result.ResChanged
	}
	if result.ResDestroyed != nil {
		summary["destroyed"] = *result.ResDestroyed
	}
	if result.ResAddedCost != nil {
		summary["addedCost"] = *result.ResAddedCost
	}
	if result.ResUpdatedCost != nil {
		summary["updatedCost"] = *result.ResUpdatedCost
	}
	if result.ResDestroyedCost != nil {
		summary["destroyedCost"] = *result.ResDestroyedCost
	}
	if len(result.ForecastFailed) > 0 {
		summary["forecastFailed"] = result.ForecastFailed
	}
	return summary
}

func cloudItsmTaskResultHasChanges(result models.TaskResult) bool {
	for _, value := range []*int{result.ResAdded, result.ResChanged, result.ResDestroyed} {
		if value != nil && *value > 0 {
			return true
		}
	}
	return false
}

func cloudItsmOperationIsRiskRemediation(operation models.CloudOperation, params models.ResAttrs) bool {
	if operation.OperationType == models.CloudOperationTypeSelfService && operation.Action == models.CloudOperationActionRiskRemediation {
		return true
	}
	if cloudItsmParamBool(params, "riskRemediation", "riskRemediationRequired") {
		return true
	}
	return cloudItsmParamString(params, "requestType") == models.CloudOperationActionRiskRemediation
}

func cloudItsmOperationRiskId(params models.ResAttrs) models.Id {
	for _, key := range []string{"riskId", "riskFindingId", "cloudRiskId"} {
		if value := strings.TrimSpace(cloudSyncPolicyAttrString(params[key])); value != "" {
			return models.Id(value)
		}
	}
	return ""
}

func cloudItsmRiskStatusForTicketStatus(status string) (string, bool) {
	switch strings.TrimSpace(status) {
	case models.CloudItsmTicketStatusSubmitted, models.CloudItsmTicketStatusInProgress:
		return models.CloudRiskStatusInProgress, true
	case models.CloudItsmTicketStatusResolved, models.CloudItsmTicketStatusClosed:
		return models.CloudRiskStatusResolved, true
	case models.CloudItsmTicketStatusFailed, models.CloudItsmTicketStatusCanceled:
		return models.CloudRiskStatusOpen, true
	default:
		return "", false
	}
}

func cloudItsmRiskRemediationEvidence(ticket *models.CloudItsmTicket, operation *models.CloudOperation, form *forms.UpdateCloudItsmTicketStatusForm, nextRiskStatus string, syncedAt models.Time) models.ResAttrs {
	comment := ""
	if form != nil {
		comment = form.Comment
	}
	evidence := models.ResAttrs{
		"lastItsmSyncedAt":      time.Time(syncedAt).Format(time.RFC3339),
		"lastItsmTicketId":      ticket.Id.String(),
		"lastItsmTicketStatus":  ticket.Status,
		"lastItsmRiskStatus":    nextRiskStatus,
		"lastItsmOperationId":   operation.Id.String(),
		"lastItsmOperationName": operation.Name,
		"lastItsmExternalId":    ticket.ExternalId,
		"lastItsmExternalKey":   ticket.ExternalKey,
		"lastItsmExternalUrl":   ticket.ExternalUrl,
		"lastItsmConnectorId":   ticket.ConnectorId.String(),
		"lastItsmStatusComment": comment,
	}
	if form != nil && len(form.Payload) > 0 {
		evidence["lastItsmStatusPayload"] = form.Payload
	}
	return evidence
}

func cloudItsmParamBool(params models.ResAttrs, keys ...string) bool {
	for _, key := range keys {
		switch strings.ToLower(strings.TrimSpace(cloudSyncPolicyAttrString(params[key]))) {
		case "true", "1", "yes", "y", "on":
			return true
		}
	}
	return false
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
	payload := ticket.RequestPayload
	if len(payload) == 0 && dryRunPayload != nil {
		payload = dryRunPayload
	}
	robot := cloudItsmRobotProcessingFromPayload(payload)
	resp := resps.CloudItsmTicketResp{
		CloudItsmTicket:     ticket,
		ConnectorName:       lookupName(c, &models.CloudItsmConfig{}, ticket.ConnectorId),
		OperationName:       lookupName(c, &models.CloudOperation{}, ticket.OperationId),
		ProjectName:         lookupName(c, &models.Project{}, ticket.ProjectId),
		EnvName:             lookupName(c, &models.Env{}, ticket.EnvId),
		CreatorName:         lookupName(c, &models.User{}, ticket.CreatorId),
		RobotProcessed:      attrBool(robot, "processed"),
		RobotProcessor:      cloudSyncPolicyAttrString(robot["processor"]),
		RobotProcessingTags: cloudItsmRobotProcessingTags(robot["tags"]),
		RobotAutomationMode: cloudSyncPolicyAttrString(robot["automationMode"]),
		RobotTicketChannel:  cloudSyncPolicyAttrString(robot["ticketChannel"]),
		DryRunPayload:       dryRunPayload,
	}
	return resp
}

func dispatchCloudEventItsmBestEffort(c *ctx.ServiceContext, event *models.CloudEvent) {
	if c == nil || event == nil || event.OrgId == "" || event.EventType == "" || event.Source == models.CloudEventSourceITSM {
		return
	}
	spec := cloudEventItsmDispatchSpecFromPayload(*event)
	if !spec.Enabled {
		return
	}
	configs := make([]*models.CloudItsmConfig, 0, len(spec.ConnectorIds))
	if len(spec.ConnectorIds) == 0 {
		cfg, err := ensureDefaultCloudItsmConfig(c)
		if err != nil {
			c.Logger().Warnf("ensure default cloud itsm config failed: %v", err)
			return
		}
		configs = append(configs, cfg)
	} else {
		for _, connectorId := range spec.ConnectorIds {
			cfg, err := getCloudItsmConfig(c, connectorId)
			if err != nil {
				c.Logger().Warnf("get cloud itsm config %s failed: %v", connectorId, err)
				continue
			}
			configs = append(configs, cfg)
		}
	}
	for _, cfg := range configs {
		if cfg == nil || cfg.Status != models.CloudItsmConfigStatusEnabled {
			continue
		}
		if _, err := createCloudItsmTicketForEvent(c, *event, cfg, spec); err != nil {
			c.Logger().Warnf("create cloud event itsm ticket failed: %v", err)
		}
	}
}

func cloudEventItsmDispatchSpecFromPayload(event models.CloudEvent) cloudEventItsmDispatchSpec {
	payload := event.Payload
	spec := cloudEventItsmDispatchSpec{
		ConnectorIds: cloudEventItsmConnectorIds(payload),
		Title:        strings.TrimSpace(cloudSyncPolicyAttrString(payloadValue(payload, "itsmTitle"))),
		Description:  strings.TrimSpace(cloudSyncPolicyAttrString(payloadValue(payload, "itsmDescription"))),
		Priority:     strings.TrimSpace(cloudSyncPolicyAttrString(payloadValue(payload, "itsmPriority"))),
	}
	autoTicketSet := payloadHasAny(payload, "itsmAutoTicket", "itsmTicketAutoCreate")
	autoTicket := cloudEventNotificationPayloadBool(payload, "itsmAutoTicket") ||
		cloudEventNotificationPayloadBool(payload, "itsmTicketAutoCreate")
	spec.Enabled = autoTicket || len(spec.ConnectorIds) > 0
	if autoTicketSet && !autoTicket {
		spec.Enabled = false
		spec.ConnectorIds = nil
		spec.Reason = "disabled"
		return spec
	}
	if spec.Title == "" {
		spec.Title = firstNonEmpty(event.Title, event.EventType)
	}
	if spec.Description == "" {
		spec.Description = firstNonEmpty(event.Message, event.Title, event.EventType)
	}
	if spec.Priority == "" {
		spec.Priority = cloudItsmPriorityFromEvent(event)
	}
	if autoTicket {
		spec.Reason = "auto_ticket"
	} else if len(spec.ConnectorIds) > 0 {
		spec.Reason = "connector_route"
	}
	return spec
}

func cloudEventItsmConnectorIds(payload models.ResAttrs) []models.Id {
	values := []string{}
	for _, key := range []string{"itsmConnectorId", "itsmConnectorIds", "itsmTicketConnectorId", "itsmTicketConnectorIds"} {
		values = append(values, cloudEventNotificationPayloadValues(payload, key)...)
	}
	values = dedupeStrings(values)
	result := make([]models.Id, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result = append(result, models.Id(value))
	}
	return result
}

func payloadValue(payload models.ResAttrs, key string) interface{} {
	if payload == nil {
		return nil
	}
	return payload[key]
}

func payloadHasAny(payload models.ResAttrs, keys ...string) bool {
	if payload == nil {
		return false
	}
	for _, key := range keys {
		if _, ok := payload[key]; ok {
			return true
		}
	}
	return false
}

func createCloudItsmTicketForEvent(c *ctx.ServiceContext, event models.CloudEvent, cfg *models.CloudItsmConfig, spec cloudEventItsmDispatchSpec) (*resps.CloudItsmTicketResp, e.Error) {
	if cfg == nil {
		return nil, e.New(e.BadParam, fmt.Errorf("ITSM 连接器不存在"), http.StatusBadRequest)
	}
	existing := models.CloudItsmTicket{}
	if err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and cloud_event_id = ? and connector_id = ?", event.OrgId, event.Id, cfg.Id).
		First(&existing); err == nil {
		resp := cloudItsmTicketResp(c, existing, nil)
		return &resp, nil
	} else if !e.IsRecordNotFound(err) {
		return nil, e.New(e.DBError, err)
	}

	payload := cloudItsmTicketRequestPayloadForEvent(c, event, *cfg, spec)
	ticket := &models.CloudItsmTicket{
		OrgId:          event.OrgId,
		ProjectId:      event.ProjectId,
		EnvId:          event.EnvId,
		OperationId:    event.OperationId,
		CloudEventId:   event.Id,
		ConnectorId:    cfg.Id,
		CreatorId:      firstNonEmptyId(event.ActorId, c.UserId),
		Title:          spec.Title,
		Description:    spec.Description,
		Status:         models.CloudItsmTicketStatusPending,
		Priority:       spec.Priority,
		RiskLevel:      cloudItsmRiskFromEvent(event),
		Provider:       cfg.Provider,
		RequestPayload: payload,
	}
	ticket.Id = models.NewId("cit")
	if err := models.Create(c.DB(), ticket); err != nil {
		return nil, e.New(e.DBError, err)
	}

	if strings.TrimSpace(cfg.BaseUrl) == "" && strings.TrimSpace(attrString(cfg.Metadata, "createTicketUrl")) == "" {
		cloudItsmTicketEventForCloudEvent(c, ticket, event, "itsm.ticket.created", models.CloudEventLevelInfo, "ITSM 事件工单已创建", "未配置外部工单地址，已创建本地待提交工单", nil)
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
	titleMsg := "ITSM 事件工单已提交"
	message := fmt.Sprintf("已向 %s 提交事件工单", cfg.Name)
	if ticket.Status == models.CloudItsmTicketStatusFailed {
		eventType = "itsm.ticket.failed"
		level = models.CloudEventLevelError
		titleMsg = "ITSM 事件工单提交失败"
		message = submit.ErrorMessage
	}
	cloudItsmTicketEventForCloudEvent(c, ticket, event, eventType, level, titleMsg, message, submit.ResponsePayload)
	resp := cloudItsmTicketResp(c, *ticket, nil)
	return &resp, nil
}

func cloudItsmTicketRequestPayloadForEvent(c *ctx.ServiceContext, event models.CloudEvent, cfg models.CloudItsmConfig, spec cloudEventItsmDispatchSpec) models.ResAttrs {
	payload := models.ResAttrs{
		"title":       spec.Title,
		"description": spec.Description,
		"priority":    spec.Priority,
		"projectKey":  cfg.ProjectKey,
		"ticketType":  cfg.TicketType,
		"source":      "cloudiac",
		"dispatch": models.ResAttrs{
			"reason":       spec.Reason,
			"autoTicket":   cloudEventNotificationPayloadBool(event.Payload, "itsmAutoTicket"),
			"connectorIds": idStrings(spec.ConnectorIds),
		},
		"event": models.ResAttrs{
			"id":             event.Id.String(),
			"eventType":      event.EventType,
			"source":         event.Source,
			"level":          event.Level,
			"status":         event.Status,
			"title":          event.Title,
			"message":        event.Message,
			"provider":       event.Provider,
			"accountId":      event.AccountId,
			"cloudAccountId": event.CloudAccountId.String(),
			"region":         event.Region,
			"resourceType":   event.ResourceType,
			"resourceId":     event.ResourceId,
			"resourceName":   event.ResourceName,
			"assetId":        event.AssetId.String(),
			"operationId":    event.OperationId.String(),
			"riskFindingId":  event.RiskFindingId.String(),
			"projectId":      event.ProjectId.String(),
			"envId":          event.EnvId.String(),
			"occurredAt":     event.OccurredAt,
			"payload":        event.Payload,
		},
		"requester": models.ResAttrs{
			"userId":   firstNonEmptyId(event.ActorId, c.UserId).String(),
			"username": c.Username,
			"email":    c.Email,
		},
		"connector": models.ResAttrs{
			"id":       cfg.Id.String(),
			"name":     cfg.Name,
			"provider": cfg.Provider,
		},
	}
	cloudItsmAttachRobotProcessing(payload, cloudItsmRobotProcessingForEvent(event, cfg, spec))
	cloudItsmAttachExternalPayload(cfg, payload)
	return payload
}

func cloudItsmTicketEventForCloudEvent(c *ctx.ServiceContext, ticket *models.CloudItsmTicket, sourceEvent models.CloudEvent, eventType string, level string, title string, message string, payload models.ResAttrs) models.Id {
	if ticket == nil {
		return ""
	}
	if payload == nil {
		payload = models.ResAttrs{}
	}
	payload["ticketId"] = ticket.Id.String()
	payload["connectorId"] = ticket.ConnectorId.String()
	payload["sourceEventId"] = sourceEvent.Id.String()
	payload["sourceEventType"] = sourceEvent.EventType
	payload["externalId"] = ticket.ExternalId
	payload["externalKey"] = ticket.ExternalKey
	payload["externalUrl"] = ticket.ExternalUrl
	event := models.CloudEvent{
		OrgId:          ticket.OrgId,
		ProjectId:      ticket.ProjectId,
		EnvId:          ticket.EnvId,
		AssetId:        sourceEvent.AssetId,
		OperationId:    sourceEvent.OperationId,
		RiskFindingId:  sourceEvent.RiskFindingId,
		CloudAccountId: sourceEvent.CloudAccountId,
		ActorId:        ticket.CreatorId,
		Source:         models.CloudEventSourceITSM,
		EventType:      eventType,
		Level:          level,
		Status:         ticket.Status,
		Provider:       sourceEvent.Provider,
		AccountId:      sourceEvent.AccountId,
		Region:         sourceEvent.Region,
		ResourceType:   sourceEvent.ResourceType,
		ResourceId:     sourceEvent.ResourceId,
		ResourceName:   sourceEvent.ResourceName,
		Title:          title,
		Message:        message,
		Payload:        payload,
	}
	event.Id = models.NewId("cev")
	recordCloudEventBestEffort(c, event)
	return event.Id
}

func cloudItsmPriorityFromEvent(event models.CloudEvent) string {
	switch event.Level {
	case models.CloudEventLevelError:
		return "high"
	case models.CloudEventLevelWarning:
		return "medium"
	default:
		return "low"
	}
}

func cloudItsmRiskFromEvent(event models.CloudEvent) string {
	switch event.Level {
	case models.CloudEventLevelError:
		return models.CloudOperationRiskHigh
	case models.CloudEventLevelWarning:
		return models.CloudOperationRiskMedium
	default:
		return models.CloudOperationRiskLow
	}
}

func idStrings(values []models.Id) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value.String())
		}
	}
	return result
}

func cloudItsmCatalog(c *ctx.ServiceContext) []resps.CloudItsmCatalogItemResp {
	policyConfig := cloudItsmCatalogPolicyConfig(c)
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
		item := resps.CloudItsmCatalogItemResp{
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
		}
		items = append(items, cloudItsmCatalogItemWithPolicy(policyConfig, item))
	}
	selfServiceItems := []resps.CloudItsmCatalogItemResp{
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
			Key:            models.CloudOperationActionRiskRemediation,
			Name:           "风险整改申请",
			Category:       "risk",
			Description:    "从风险合规或漂移详情发起整改工单，关联风险证据、资源、修复建议和处理状态",
			Source:         "itsm_self_service",
			OperationType:  models.CloudOperationTypeSelfService,
			Action:         models.CloudOperationActionRiskRemediation,
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
	}
	for _, item := range selfServiceItems {
		items = append(items, cloudItsmCatalogItemWithPolicy(policyConfig, item))
	}
	return items
}

func cloudItsmCatalogItemByKey(c *ctx.ServiceContext, key string) (resps.CloudItsmCatalogItemResp, bool) {
	key = strings.TrimSpace(key)
	for _, item := range cloudItsmCatalog(c) {
		if item.Key == key {
			return item, true
		}
	}
	return resps.CloudItsmCatalogItemResp{}, false
}

func cloudItsmCatalogItemWithPolicy(policyConfig map[string]models.ResAttrs, item resps.CloudItsmCatalogItemResp) resps.CloudItsmCatalogItemResp {
	policy := resps.CloudItsmCatalogItemResp{
		PolicyKey:         firstNonEmpty(item.Category, "operations") + "_self_service_policy",
		PolicyName:        "常规自助运维策略",
		PolicyDescription: "允许项目成员在项目、环境或资源范围内发起常规自助运维申请",
		RequiredRoles:     []string{"project_member"},
		AllowedScopes:     []string{"project", "env", "asset"},
		SlaMinutes:        240,
	}
	switch item.Key {
	case models.CloudOperationActionPermissionRequest:
		policy.PolicyKey = "self_service_permission_policy"
		policy.PolicyName = "权限申请策略"
		policy.PolicyDescription = "允许项目成员自助提交云账号、项目、环境或资源权限申请"
		policy.RequiredRoles = []string{"project_member"}
		policy.AllowedScopes = []string{"org", "project", "env", "asset"}
		policy.SlaMinutes = 480
	case models.CloudOperationActionGitOpsIacChange:
		policy.PolicyKey = "gitops_iac_change_policy"
		policy.PolicyName = "GitOps/IaC 变更策略"
		policy.PolicyDescription = "要求基础设施变更具备 PR review 和自动化流水线证据后进入工单闭环"
		policy.RequiredRoles = []string{"project_owner", "iac_reviewer"}
		policy.AllowedScopes = []string{"project", "env"}
		policy.SlaMinutes = 1440
	case models.CloudOperationActionRiskRemediation:
		policy.PolicyKey = "risk_remediation_policy"
		policy.PolicyName = "风险整改策略"
		policy.PolicyDescription = "允许安全、SRE 或项目负责人发起带证据链的风险整改申请"
		policy.RequiredRoles = []string{"sre", "security_owner", "project_owner"}
		policy.AllowedScopes = []string{"org", "project", "env", "asset"}
		policy.SlaMinutes = 240
	case models.CloudOperationActionDriftRemediation:
		policy.PolicyKey = "drift_remediation_policy"
		policy.PolicyName = "漂移修复策略"
		policy.PolicyDescription = "允许 SRE 或项目负责人对检测到的配置漂移发起修复申请"
		policy.RequiredRoles = []string{"sre", "project_owner"}
		policy.AllowedScopes = []string{"project", "env", "asset"}
		policy.SlaMinutes = 240
	default:
		switch item.Category {
		case "lifecycle":
			policy.PolicyKey = "cloud_lifecycle_policy"
			policy.PolicyName = "生命周期自助策略"
			policy.PolicyDescription = "允许项目成员或运维人员在资源范围内执行启动、停止、重启等常规操作"
			policy.RequiredRoles = []string{"project_member", "operator"}
			policy.AllowedScopes = []string{"project", "env", "asset"}
			policy.SlaMinutes = 60
		case "scaling":
			policy.PolicyKey = "cloud_scaling_policy"
			policy.PolicyName = "扩缩容自助策略"
			policy.PolicyDescription = "允许运维人员或项目负责人对资源规格、容量进行受控变更"
			policy.RequiredRoles = []string{"operator", "project_owner"}
			policy.AllowedScopes = []string{"env", "asset"}
			policy.SlaMinutes = 120
		case "configuration":
			policy.PolicyKey = "cloud_configuration_policy"
			policy.PolicyName = "配置变更自助策略"
			policy.PolicyDescription = "允许运维人员或项目负责人发起标签、安全规则等配置变更"
			policy.RequiredRoles = []string{"operator", "project_owner"}
			policy.AllowedScopes = []string{"env", "asset"}
			policy.SlaMinutes = 120
		case "backup":
			policy.PolicyKey = "cloud_backup_policy"
			policy.PolicyName = "备份快照自助策略"
			policy.PolicyDescription = "允许运维人员在资源范围内发起备份或快照操作"
			policy.RequiredRoles = []string{"operator"}
			policy.AllowedScopes = []string{"env", "asset"}
			policy.SlaMinutes = 240
		}
	}
	if item.RequiresApproval || strings.Contains(item.AutomationMode, "approval") {
		policy.PolicyKey = firstNonEmpty(policy.PolicyKey, "approval_required_policy")
		policy.PolicyName = firstNonEmpty(policy.PolicyName, "审批型自助策略")
		policy.PolicyDescription = firstNonEmpty(policy.PolicyDescription, "高风险自助操作需项目负责人或运维审批后执行")
		if len(policy.RequiredRoles) == 0 {
			policy.RequiredRoles = []string{"project_owner", "operator"}
		}
		if policy.SlaMinutes <= 0 {
			policy.SlaMinutes = 240
		}
	}
	item.PolicyKey = policy.PolicyKey
	item.PolicyName = policy.PolicyName
	item.PolicyDescription = policy.PolicyDescription
	item.RequiredRoles = policy.RequiredRoles
	item.AllowedScopes = policy.AllowedScopes
	item.SlaMinutes = policy.SlaMinutes
	item.SlaDescription = cloudItsmSlaDescription(policy.SlaMinutes)
	item.PolicySource = "default"
	if override := policyConfig[item.Key]; len(override) > 0 {
		item = cloudItsmApplyCatalogPolicyOverride(item, override)
	}
	return item
}

func cloudItsmCatalogPolicyConfig(c *ctx.ServiceContext) map[string]models.ResAttrs {
	result := map[string]models.ResAttrs{}
	if c == nil || c.OrgId == "" {
		return result
	}
	cfg, err := services.GetSystemConfigByName(c.DB(), cloudItsmCatalogPolicyConfigName(c.OrgId))
	if err != nil || cfg == nil || strings.TrimSpace(cfg.Value) == "" {
		return result
	}
	raw := map[string]models.ResAttrs{}
	if jsonErr := json.Unmarshal([]byte(cfg.Value), &raw); jsonErr != nil {
		c.Logger().Warnf("parse cloud itsm catalog policy config failed: %v", jsonErr)
		return result
	}
	for key, value := range raw {
		if normalizedKey := strings.TrimSpace(key); normalizedKey != "" && len(value) > 0 {
			result[normalizedKey] = value
		}
	}
	return result
}

func saveCloudItsmCatalogPolicyConfig(c *ctx.ServiceContext, config map[string]models.ResAttrs) e.Error {
	if config == nil {
		config = map[string]models.ResAttrs{}
	}
	payload, err := json.Marshal(config)
	if err != nil {
		return e.New(e.InternalError, err)
	}
	if _, err := services.UpsertSystemConfigValue(c.DB(), cloudItsmCatalogPolicyConfigName(c.OrgId), string(payload)); err != nil {
		return err
	}
	return nil
}

func cloudItsmCatalogPolicyConfigName(orgId models.Id) string {
	return models.SysCfgNameCloudItsmCatalogPolicyPrefix + orgId.String()
}

func cloudItsmCatalogPolicyHistoryConfig(c *ctx.ServiceContext) map[string][]models.ResAttrs {
	result := map[string][]models.ResAttrs{}
	if c == nil || c.OrgId == "" {
		return result
	}
	cfg, err := services.GetSystemConfigByName(c.DB(), cloudItsmCatalogPolicyHistoryConfigName(c.OrgId))
	if err != nil || cfg == nil || strings.TrimSpace(cfg.Value) == "" {
		return result
	}
	raw := map[string][]models.ResAttrs{}
	if jsonErr := json.Unmarshal([]byte(cfg.Value), &raw); jsonErr != nil {
		c.Logger().Warnf("parse cloud itsm catalog policy history config failed: %v", jsonErr)
		return result
	}
	for key, entries := range raw {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		normalizedEntries := make([]models.ResAttrs, 0, len(entries))
		for _, entry := range entries {
			if len(entry) > 0 {
				normalizedEntries = append(normalizedEntries, entry)
			}
		}
		if len(normalizedEntries) > 0 {
			result[normalizedKey] = normalizedEntries
		}
	}
	return result
}

func saveCloudItsmCatalogPolicyHistoryConfig(c *ctx.ServiceContext, history map[string][]models.ResAttrs) e.Error {
	if history == nil {
		history = map[string][]models.ResAttrs{}
	}
	payload, err := json.Marshal(history)
	if err != nil {
		return e.New(e.InternalError, err)
	}
	if _, err := services.UpsertSystemConfigValue(c.DB(), cloudItsmCatalogPolicyHistoryConfigName(c.OrgId), string(payload)); err != nil {
		return err
	}
	return nil
}

func cloudItsmCatalogPolicyHistoryConfigName(orgId models.Id) string {
	return models.SysCfgNameCloudItsmCatalogPolicyHistoryPrefix + orgId.String()
}

func cloudItsmCatalogPolicySnapshot(item resps.CloudItsmCatalogItemResp) models.ResAttrs {
	return models.ResAttrs{
		"enabled":           item.Enabled,
		"policyName":        item.PolicyName,
		"policyDescription": item.PolicyDescription,
		"requiredRoles":     append([]string{}, item.RequiredRoles...),
		"allowedScopes":     append([]string{}, item.AllowedScopes...),
		"slaMinutes":        item.SlaMinutes,
	}
}

func cloudItsmCatalogPolicyDiff(before models.ResAttrs, after models.ResAttrs) []resps.CloudItsmCatalogPolicyDiffResp {
	labels := map[string]string{
		"enabled":           "策略状态",
		"policyName":        "策略名称",
		"policyDescription": "策略说明",
		"requiredRoles":     "允许角色",
		"allowedScopes":     "适用范围",
		"slaMinutes":        "SLA 分钟",
	}
	fields := []string{"enabled", "policyName", "policyDescription", "requiredRoles", "allowedScopes", "slaMinutes"}
	result := make([]resps.CloudItsmCatalogPolicyDiffResp, 0)
	for _, field := range fields {
		beforeValue := before[field]
		afterValue := after[field]
		if cloudItsmCatalogPolicyDiffValueEqual(beforeValue, afterValue) {
			continue
		}
		result = append(result, resps.CloudItsmCatalogPolicyDiffResp{
			Field:  field,
			Label:  labels[field],
			Before: beforeValue,
			After:  afterValue,
		})
	}
	return result
}

func cloudItsmCatalogPolicyDiffValueEqual(before interface{}, after interface{}) bool {
	beforePayload, beforeErr := json.Marshal(before)
	afterPayload, afterErr := json.Marshal(after)
	if beforeErr != nil || afterErr != nil {
		return fmt.Sprintf("%v", before) == fmt.Sprintf("%v", after)
	}
	return string(beforePayload) == string(afterPayload)
}

func cloudItsmCatalogPolicyNextVersion(history map[string][]models.ResAttrs, key string, override models.ResAttrs) int {
	version := cloudSyncPolicyAttrInt(override["policyVersion"])
	for _, entry := range history[key] {
		if entryVersion := cloudSyncPolicyAttrInt(entry["version"]); entryVersion > version {
			version = entryVersion
		}
	}
	return version + 1
}

func cloudItsmCatalogPolicyHistoryEntry(key string, action string, version int, updatedAt string, updatedBy string, before models.ResAttrs, after models.ResAttrs) models.ResAttrs {
	diff := cloudItsmCatalogPolicyDiff(before, after)
	return models.ResAttrs{
		"version":        version,
		"action":         action,
		"key":            key,
		"policyName":     cloudSyncPolicyAttrString(after["policyName"]),
		"updatedAt":      updatedAt,
		"updatedBy":      updatedBy,
		"diff":           diff,
		"beforeSnapshot": before,
		"afterSnapshot":  after,
	}
}

func cloudItsmAppendCatalogPolicyHistory(history map[string][]models.ResAttrs, key string, entry models.ResAttrs) map[string][]models.ResAttrs {
	if history == nil {
		history = map[string][]models.ResAttrs{}
	}
	entries := append([]models.ResAttrs{entry}, history[key]...)
	if len(entries) > cloudItsmCatalogPolicyHistoryLimit {
		entries = entries[:cloudItsmCatalogPolicyHistoryLimit]
	}
	history[key] = entries
	return history
}

func cloudItsmCatalogPolicyHistoryResp(entry models.ResAttrs) resps.CloudItsmCatalogPolicyHistoryResp {
	return resps.CloudItsmCatalogPolicyHistoryResp{
		Version:        cloudSyncPolicyAttrInt(entry["version"]),
		Action:         strings.TrimSpace(cloudSyncPolicyAttrString(entry["action"])),
		Key:            strings.TrimSpace(cloudSyncPolicyAttrString(entry["key"])),
		PolicyName:     strings.TrimSpace(cloudSyncPolicyAttrString(entry["policyName"])),
		UpdatedAt:      strings.TrimSpace(cloudSyncPolicyAttrString(entry["updatedAt"])),
		UpdatedBy:      strings.TrimSpace(cloudSyncPolicyAttrString(entry["updatedBy"])),
		Diff:           cloudItsmCatalogPolicyDiffFromAttr(entry["diff"]),
		BeforeSnapshot: modelResAttrs(entry["beforeSnapshot"]),
		AfterSnapshot:  modelResAttrs(entry["afterSnapshot"]),
	}
}

func cloudItsmCatalogPolicyDiffFromAttr(value interface{}) []resps.CloudItsmCatalogPolicyDiffResp {
	if value == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	result := make([]resps.CloudItsmCatalogPolicyDiffResp, 0)
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil
	}
	return result
}

func cloudItsmCatalogPolicyEvent(c *ctx.ServiceContext, item resps.CloudItsmCatalogItemResp, historyEntry models.ResAttrs) {
	if c == nil || c.OrgId == "" {
		return
	}
	action := strings.TrimSpace(cloudSyncPolicyAttrString(historyEntry["action"]))
	eventType := "itsm.catalog_policy.updated"
	title := "ITSM 自助目录策略已更新"
	if action == "reset" {
		eventType = "itsm.catalog_policy.reset"
		title = "ITSM 自助目录策略已恢复默认"
	}
	diff := cloudItsmCatalogPolicyDiffFromAttr(historyEntry["diff"])
	event := models.CloudEvent{
		OrgId:        c.OrgId,
		ActorId:      c.UserId,
		Source:       models.CloudEventSourceITSM,
		EventType:    eventType,
		Level:        models.CloudEventLevelInfo,
		Status:       action,
		ResourceType: "cloud_itsm_catalog_policy",
		ResourceId:   item.Key,
		ResourceName: item.Name,
		Title:        title,
		Message:      fmt.Sprintf("%s：%s，版本 v%d，差异 %d 项", title, item.Name, cloudSyncPolicyAttrInt(historyEntry["version"]), len(diff)),
		Payload:      historyEntry,
		OccurredAt:   models.Time(time.Now()),
	}
	event.Id = models.NewId("cev")
	recordCloudEventBestEffort(c, event)
}

func cloudItsmApplyCatalogPolicyOverride(item resps.CloudItsmCatalogItemResp, override models.ResAttrs) resps.CloudItsmCatalogItemResp {
	baseAvailable := item.Available
	item.PolicyConfigured = true
	item.PolicySource = "org_config"
	if value, ok := override["enabled"]; ok {
		enabled := cloudSyncPolicyAttrBool(value)
		item.Enabled = enabled
		item.Available = baseAvailable && enabled
		if !enabled {
			item.DisabledReason = "策略已停用"
		} else if !baseAvailable && item.DisabledReason == "" {
			item.DisabledReason = "服务项暂未开放"
		}
	}
	if value := strings.TrimSpace(cloudSyncPolicyAttrString(override["policyName"])); value != "" {
		item.PolicyName = value
	}
	if value := strings.TrimSpace(cloudSyncPolicyAttrString(override["policyDescription"])); value != "" {
		item.PolicyDescription = value
	}
	if roles := cloudItsmNormalizeCatalogPolicyValues(cloudSyncPolicyAttrStringSlice(override["requiredRoles"]), nil); len(roles) > 0 {
		item.RequiredRoles = roles
	}
	if scopes := cloudItsmNormalizeCatalogPolicyValues(cloudSyncPolicyAttrStringSlice(override["allowedScopes"]), nil); len(scopes) > 0 {
		item.AllowedScopes = scopes
	}
	if minutes := cloudSyncPolicyAttrInt(override["slaMinutes"]); minutes > 0 {
		item.SlaMinutes = minutes
		item.SlaDescription = cloudItsmSlaDescription(minutes)
	}
	item.PolicyVersion = cloudSyncPolicyAttrInt(override["policyVersion"])
	item.PolicyUpdatedAt = strings.TrimSpace(cloudSyncPolicyAttrString(override["updatedAt"]))
	item.PolicyUpdatedBy = strings.TrimSpace(cloudSyncPolicyAttrString(override["updatedBy"]))
	item.PolicyLastDiff = cloudItsmCatalogPolicyDiffFromAttr(override["policyLastDiff"])
	return item
}

func cloudItsmNormalizeCatalogPolicyValues(values []string, allowed map[string]bool) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		item := strings.TrimSpace(value)
		if item == "" || seen[item] {
			continue
		}
		if allowed != nil && !allowed[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

func cloudItsmSlaDescription(minutes int) string {
	if minutes <= 0 {
		return ""
	}
	if minutes%1440 == 0 {
		days := minutes / 1440
		return fmt.Sprintf("目标 %d 天内完成处理", days)
	}
	if minutes%60 == 0 {
		hours := minutes / 60
		return fmt.Sprintf("目标 %d 小时内完成处理", hours)
	}
	return fmt.Sprintf("目标 %d 分钟内完成处理", minutes)
}

func cloudItsmApplySelfServicePolicyToParams(params models.ResAttrs, item resps.CloudItsmCatalogItemResp) {
	if params == nil {
		return
	}
	params["selfServicePolicy"] = cloudItsmSelfServicePolicySnapshot(item)
	params["sla"] = cloudItsmSelfServiceSlaSnapshot(item, cloudItsmDefaultSlaTarget)
	params["slaMinutes"] = item.SlaMinutes
}

func cloudItsmSelfServicePolicySnapshot(item resps.CloudItsmCatalogItemResp) models.ResAttrs {
	return models.ResAttrs{
		"key":           item.PolicyKey,
		"name":          item.PolicyName,
		"description":   item.PolicyDescription,
		"requiredRoles": item.RequiredRoles,
		"allowedScopes": item.AllowedScopes,
		"catalogKey":    item.Key,
		"category":      item.Category,
	}
}

func cloudItsmSelfServiceSlaSnapshot(item resps.CloudItsmCatalogItemResp, target float64) models.ResAttrs {
	return models.ResAttrs{
		"minutes":     item.SlaMinutes,
		"description": item.SlaDescription,
		"target":      target,
		"catalogKey":  item.Key,
	}
}

func cloudItsmSelfServicePolicyFromParams(params models.ResAttrs) models.ResAttrs {
	policy := modelResAttrs(params["selfServicePolicy"])
	if len(policy) > 0 {
		return policy
	}
	item, ok := cloudItsmCatalogItemByKey(nil, firstNonEmpty(
		cloudSyncPolicyAttrString(params["catalogKey"]),
		cloudSyncPolicyAttrString(params["requestType"]),
	))
	if !ok {
		return models.ResAttrs{}
	}
	return cloudItsmSelfServicePolicySnapshot(item)
}

func cloudItsmSelfServiceSlaFromParams(params models.ResAttrs) models.ResAttrs {
	sla := modelResAttrs(params["sla"])
	if len(sla) > 0 {
		return sla
	}
	item, ok := cloudItsmCatalogItemByKey(nil, firstNonEmpty(
		cloudSyncPolicyAttrString(params["catalogKey"]),
		cloudSyncPolicyAttrString(params["requestType"]),
	))
	if !ok {
		return models.ResAttrs{}
	}
	return cloudItsmSelfServiceSlaSnapshot(item, cloudItsmDefaultSlaTarget)
}

type cloudItsmTicketSlaEvaluation struct {
	Date     string
	Counted  bool
	Met      bool
	Breached bool
}

func cloudItsmSelfServiceSlaTrend(c *ctx.ServiceContext, now time.Time, target float64) ([]resps.CloudItsmSlaTrendResp, resps.CloudItsmOverviewMetricResp, e.Error) {
	start := cloudItsmSlaTrendStart(now)
	trend := make([]resps.CloudItsmSlaTrendResp, 0, 7)
	byDate := map[string]int{}
	for idx := 0; idx < 7; idx++ {
		date := start.AddDate(0, 0, idx).Format("2006-01-02")
		byDate[date] = idx
		trend = append(trend, resps.CloudItsmSlaTrendResp{
			Date:   date,
			Target: target,
		})
	}

	tickets := make([]models.CloudItsmTicket, 0)
	if err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and created_at >= ?", c.OrgId, models.Time(start)).
		Find(&tickets); err != nil {
		return nil, resps.CloudItsmOverviewMetricResp{}, e.New(e.DBError, err)
	}

	metrics := resps.CloudItsmOverviewMetricResp{
		SelfServiceSlaTarget: target,
	}
	for _, ticket := range tickets {
		result := cloudItsmTicketSlaResult(ticket, now)
		if !result.Counted {
			continue
		}
		metrics.SelfServiceSlaTicketTotal++
		if result.Met {
			metrics.SelfServiceSlaMetTotal++
		}
		if result.Breached {
			metrics.SelfServiceSlaBreached++
		}
		idx, ok := byDate[result.Date]
		if !ok {
			continue
		}
		trend[idx].TicketTotal++
		if result.Met {
			trend[idx].SlaMetTotal++
		}
		if result.Breached {
			trend[idx].SlaBreached++
		}
	}
	metrics.SelfServiceSlaMetRate = cloudItsmSlaMetRate(metrics.SelfServiceSlaMetTotal, metrics.SelfServiceSlaBreached)
	for idx := range trend {
		trend[idx].SlaMetRate = cloudItsmSlaMetRate(trend[idx].SlaMetTotal, trend[idx].SlaBreached)
	}
	return trend, metrics, nil
}

func cloudItsmDimensionTrends(c *ctx.ServiceContext, now time.Time, targets resps.CloudItsmOverviewMetricResp) ([]resps.CloudItsmDimensionMetricResp, []resps.CloudItsmDimensionMetricResp, []resps.CloudItsmDimensionMetricResp, e.Error) {
	start := cloudItsmDimensionTrendStart(now)
	tickets := make([]models.CloudItsmTicket, 0)
	if err := c.DB().Model(&models.CloudItsmTicket{}).
		Where("org_id = ? and created_at >= ?", c.OrgId, models.Time(start)).
		Find(&tickets); err != nil {
		return nil, nil, nil, e.New(e.DBError, err)
	}

	projectTrends := map[string]*resps.CloudItsmDimensionMetricResp{}
	requestTypeTrends := map[string]*resps.CloudItsmDimensionMetricResp{}
	teamTrends := map[string]*resps.CloudItsmDimensionMetricResp{}
	projectNames := map[string]string{}
	requestTypeNames := cloudItsmCatalogNameMap(c)
	teamCache := &cloudItsmTeamDimensionCache{
		projectTeams: map[string]cloudItsmTeamDimension{},
		userTeams:    map[string]cloudItsmTeamDimension{},
	}
	for _, ticket := range tickets {
		projectId := ticket.ProjectId.String()
		projectName := "未关联项目"
		if projectId != "" {
			if name, ok := projectNames[projectId]; ok {
				projectName = name
			} else if name := lookupName(c, &models.Project{}, ticket.ProjectId); name != "" {
				projectName = name
				projectNames[projectId] = name
			} else {
				projectName = projectId
				projectNames[projectId] = projectId
			}
		}
		projectMetric := cloudItsmDimensionMetric(projectTrends, "project", projectId, projectName, targets)
		cloudItsmDimensionMetricAddTicket(projectMetric, ticket, now)

		team := cloudItsmTicketTeamDimension(c, ticket, projectId, projectName, teamCache)
		teamMetric := cloudItsmDimensionMetric(teamTrends, "team", team.Id, team.Name, targets)
		cloudItsmDimensionMetricAddTicket(teamMetric, ticket, now)

		requestType := cloudItsmTicketRequestType(ticket)
		if requestType == "" {
			requestType = "unknown"
		}
		requestTypeName := requestTypeNames[requestType]
		if requestTypeName == "" {
			requestTypeName = requestType
		}
		requestTypeMetric := cloudItsmDimensionMetric(requestTypeTrends, "request_type", requestType, requestTypeName, targets)
		cloudItsmDimensionMetricAddTicket(requestTypeMetric, ticket, now)
	}
	return cloudItsmDimensionMetricList(projectTrends), cloudItsmDimensionMetricList(requestTypeTrends), cloudItsmDimensionMetricList(teamTrends), nil
}

func cloudItsmDimensionTrendStart(now time.Time) time.Time {
	if now.IsZero() {
		now = time.Now()
	}
	year, month, day := now.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -29)
}

func cloudItsmCatalogNameMap(c *ctx.ServiceContext) map[string]string {
	result := map[string]string{}
	for _, item := range cloudItsmCatalog(c) {
		result[item.Key] = item.Name
	}
	return result
}

func cloudItsmDimensionMetric(metrics map[string]*resps.CloudItsmDimensionMetricResp, dimensionType string, dimensionId string, dimensionName string, targets resps.CloudItsmOverviewMetricResp) *resps.CloudItsmDimensionMetricResp {
	key := dimensionType + ":" + dimensionId
	if metric, ok := metrics[key]; ok {
		return metric
	}
	metric := &resps.CloudItsmDimensionMetricResp{
		DimensionType:             dimensionType,
		DimensionId:               dimensionId,
		DimensionName:             dimensionName,
		SelfServiceCoverageTarget: targets.SelfServiceCoverageTarget,
		TicketAutomationTarget:    targets.TicketAutomationTarget,
		SelfServiceSlaTarget:      targets.SelfServiceSlaTarget,
		WindowDays:                30,
	}
	metrics[key] = metric
	return metric
}

func cloudItsmDimensionMetricAddTicket(metric *resps.CloudItsmDimensionMetricResp, ticket models.CloudItsmTicket, now time.Time) {
	if metric == nil {
		return
	}
	metric.TicketTotal++
	if cloudItsmTicketSelfService(ticket) {
		metric.SelfServiceTicketTotal++
	}
	if cloudItsmTicketAutomated(ticket) {
		metric.AutomatedTicketTotal++
	}
	if cloudItsmTicketRobotProcessed(ticket) {
		metric.RobotProcessedTicketTotal++
	}
	slaResult := cloudItsmTicketSlaResult(ticket, now)
	if slaResult.Counted {
		metric.SelfServiceSlaTicketTotal++
		if slaResult.Met {
			metric.SelfServiceSlaMetTotal++
		}
		if slaResult.Breached {
			metric.SelfServiceSlaBreached++
		}
	}
	cloudItsmFinalizeDimensionMetric(metric)
}

func cloudItsmFinalizeDimensionMetric(metric *resps.CloudItsmDimensionMetricResp) {
	if metric.TicketTotal > 0 {
		metric.SelfServiceCoverageRate = float64(metric.SelfServiceTicketTotal) / float64(metric.TicketTotal) * 100
		metric.TicketAutomationRate = float64(metric.AutomatedTicketTotal) / float64(metric.TicketTotal) * 100
	}
	metric.SelfServiceSlaMetRate = cloudItsmSlaMetRate(metric.SelfServiceSlaMetTotal, metric.SelfServiceSlaBreached)
}

func cloudItsmDimensionMetricList(metrics map[string]*resps.CloudItsmDimensionMetricResp) []resps.CloudItsmDimensionMetricResp {
	items := make([]resps.CloudItsmDimensionMetricResp, 0, len(metrics))
	for _, metric := range metrics {
		cloudItsmFinalizeDimensionMetric(metric)
		items = append(items, *metric)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].TicketTotal == items[j].TicketTotal {
			return items[i].DimensionName < items[j].DimensionName
		}
		return items[i].TicketTotal > items[j].TicketTotal
	})
	return items
}

type cloudItsmTeamDimension struct {
	Id     string
	Name   string
	Source string
}

type cloudItsmTeamDimensionCache struct {
	projectTeams map[string]cloudItsmTeamDimension
	userTeams    map[string]cloudItsmTeamDimension
}

func cloudItsmTicketTeamDimension(c *ctx.ServiceContext, ticket models.CloudItsmTicket, projectId string, projectName string, cache *cloudItsmTeamDimensionCache) cloudItsmTeamDimension {
	payload := ticket.RequestPayload
	operation := modelResAttrs(payload["operation"])
	params := modelResAttrs(operation["params"])
	if params == nil {
		params = modelResAttrs(payload["params"])
	}
	requester := modelResAttrs(payload["requester"])
	externalPayload := modelResAttrs(payload["externalPayload"])
	for _, attrs := range []models.ResAttrs{params, operation, payload, requester, externalPayload} {
		if team := cloudItsmTeamDimensionFromAttrs(attrs); team.Id != "" {
			return team
		}
	}
	if team := cloudItsmTeamDimensionFromProjectOU(c, projectId, cache); team.Id != "" {
		return team
	}
	if team := cloudItsmTeamDimensionFromUser(c, ticket.CreatorId, cache); team.Id != "" {
		return team
	}
	if projectId != "" && projectName != "" && projectName != "未关联项目" {
		return cloudItsmNormalizeTeamDimension("project:"+projectId, projectName, "project")
	}
	return cloudItsmNormalizeTeamDimension("unassigned", "未标记团队", "fallback")
}

func cloudItsmTeamDimensionFromAttrs(attrs models.ResAttrs) cloudItsmTeamDimension {
	if len(attrs) == 0 {
		return cloudItsmTeamDimension{}
	}
	for _, key := range []string{"team", "businessTeam", "requestTeam", "requesterTeam", "ownerTeam", "owner_team", "oncallTeam"} {
		if team := cloudItsmTeamDimensionFromValue(attrs[key], key); team.Id != "" {
			return team
		}
	}
	id := firstNonEmpty(
		cloudSyncPolicyAttrString(attrs["teamId"]),
		cloudSyncPolicyAttrString(attrs["teamID"]),
		cloudSyncPolicyAttrString(attrs["businessTeamId"]),
		cloudSyncPolicyAttrString(attrs["businessUnitId"]),
		cloudSyncPolicyAttrString(attrs["departmentId"]),
		cloudSyncPolicyAttrString(attrs["ownerTeamId"]),
	)
	name := firstNonEmpty(
		cloudSyncPolicyAttrString(attrs["teamName"]),
		cloudSyncPolicyAttrString(attrs["businessTeamName"]),
		cloudSyncPolicyAttrString(attrs["businessUnit"]),
		cloudSyncPolicyAttrString(attrs["department"]),
		cloudSyncPolicyAttrString(attrs["ownerTeamName"]),
		cloudSyncPolicyAttrString(attrs["costCenter"]),
	)
	return cloudItsmNormalizeTeamDimension(id, name, "payload")
}

func cloudItsmTeamDimensionFromValue(value interface{}, source string) cloudItsmTeamDimension {
	if value == nil {
		return cloudItsmTeamDimension{}
	}
	if attrs := modelResAttrs(value); len(attrs) > 0 {
		return cloudItsmNormalizeTeamDimension(
			firstNonEmpty(
				cloudSyncPolicyAttrString(attrs["id"]),
				cloudSyncPolicyAttrString(attrs["teamId"]),
				cloudSyncPolicyAttrString(attrs["key"]),
				cloudSyncPolicyAttrString(attrs["code"]),
			),
			firstNonEmpty(
				cloudSyncPolicyAttrString(attrs["name"]),
				cloudSyncPolicyAttrString(attrs["teamName"]),
				cloudSyncPolicyAttrString(attrs["displayName"]),
				cloudSyncPolicyAttrString(attrs["label"]),
			),
			source,
		)
	}
	values := normalizeStringList(cloudSyncPolicyAttrStringSlice(value))
	if len(values) > 0 {
		return cloudItsmNormalizeTeamDimension(values[0], values[0], source)
	}
	text := strings.TrimSpace(cloudSyncPolicyAttrString(value))
	return cloudItsmNormalizeTeamDimension(text, text, source)
}

func cloudItsmTeamDimensionFromProjectOU(c *ctx.ServiceContext, projectId string, cache *cloudItsmTeamDimensionCache) cloudItsmTeamDimension {
	if c == nil || projectId == "" {
		return cloudItsmTeamDimension{}
	}
	if cache != nil {
		if team, ok := cache.projectTeams[projectId]; ok {
			return team
		}
	}
	ou := models.LdapOUProject{}
	err := c.DB().Model(&models.LdapOUProject{}).
		Where("org_id = ? and project_id = ?", c.OrgId, models.Id(projectId)).
		First(&ou)
	if err != nil {
		if !e.IsRecordNotFound(err) {
			c.Logger().Warnf("lookup ITSM team dimension project ou failed: %v", err)
		}
		if cache != nil {
			cache.projectTeams[projectId] = cloudItsmTeamDimension{}
		}
		return cloudItsmTeamDimension{}
	}
	team := cloudItsmNormalizeTeamDimension(firstNonEmpty(ou.OU, ou.DN), firstNonEmpty(ou.OU, ou.DN), "ldap_project_ou")
	if cache != nil {
		cache.projectTeams[projectId] = team
	}
	return team
}

func cloudItsmTeamDimensionFromUser(c *ctx.ServiceContext, userId models.Id, cache *cloudItsmTeamDimensionCache) cloudItsmTeamDimension {
	if c == nil || userId == "" {
		return cloudItsmTeamDimension{}
	}
	key := userId.String()
	if cache != nil {
		if team, ok := cache.userTeams[key]; ok {
			return team
		}
	}
	user := models.User{}
	err := c.DB().Model(&models.User{}).Where("id = ?", userId).First(&user)
	if err != nil {
		if !e.IsRecordNotFound(err) {
			c.Logger().Warnf("lookup ITSM team dimension user failed: %v", err)
		}
		if cache != nil {
			cache.userTeams[key] = cloudItsmTeamDimension{}
		}
		return cloudItsmTeamDimension{}
	}
	team := cloudItsmNormalizeTeamDimension(user.Company, user.Company, "user_company")
	if cache != nil {
		cache.userTeams[key] = team
	}
	return team
}

func cloudItsmNormalizeTeamDimension(id string, name string, source string) cloudItsmTeamDimension {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if name == "" {
		name = id
	}
	if id == "" {
		id = name
	}
	if id == "" {
		return cloudItsmTeamDimension{}
	}
	return cloudItsmTeamDimension{
		Id:     id,
		Name:   name,
		Source: source,
	}
}

func cloudItsmTicketSelfService(ticket models.CloudItsmTicket) bool {
	payload := ticket.RequestPayload
	operation := modelResAttrs(payload["operation"])
	params := modelResAttrs(operation["params"])
	if params == nil {
		params = modelResAttrs(payload["params"])
	}
	if attrBool(params, "selfService") || cloudSyncPolicyAttrString(operation["operationType"]) == models.CloudOperationTypeSelfService {
		return true
	}
	requestType := cloudItsmTicketRequestType(ticket)
	if item, ok := cloudItsmCatalogItemByKey(nil, requestType); ok {
		return item.OperationType == models.CloudOperationTypeSelfService
	}
	return false
}

func cloudItsmTicketAutomated(ticket models.CloudItsmTicket) bool {
	if cloudItsmTicketRobotProcessed(ticket) {
		return true
	}
	switch ticket.Status {
	case models.CloudItsmTicketStatusSubmitted,
		models.CloudItsmTicketStatusInProgress,
		models.CloudItsmTicketStatusResolved,
		models.CloudItsmTicketStatusClosed:
		return !time.Time(ticket.SubmittedAt).IsZero() ||
			ticket.ExternalId != "" ||
			ticket.ExternalKey != "" ||
			ticket.ExternalUrl != ""
	default:
		return false
	}
}

func cloudItsmTicketRobotProcessed(ticket models.CloudItsmTicket) bool {
	robot := cloudItsmRobotProcessingFromPayload(ticket.RequestPayload)
	return attrBool(robot, "processed")
}

func cloudItsmTicketRequestType(ticket models.CloudItsmTicket) string {
	payload := ticket.RequestPayload
	operation := modelResAttrs(payload["operation"])
	params := modelResAttrs(operation["params"])
	if params == nil {
		params = modelResAttrs(payload["params"])
	}
	robot := cloudItsmRobotProcessingFromPayload(payload)
	return firstNonEmpty(
		cloudSyncPolicyAttrString(payload["requestType"]),
		cloudSyncPolicyAttrString(operation["action"]),
		cloudSyncPolicyAttrString(params["requestType"]),
		cloudSyncPolicyAttrString(params["catalogKey"]),
		cloudSyncPolicyAttrString(robot["requestType"]),
		ticket.Title,
	)
}

func cloudItsmSlaTrendStart(now time.Time) time.Time {
	if now.IsZero() {
		now = time.Now()
	}
	year, month, day := now.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -6)
}

func cloudItsmTicketSlaResult(ticket models.CloudItsmTicket, now time.Time) cloudItsmTicketSlaEvaluation {
	start := cloudItsmTicketSlaStartTime(ticket, now)
	result := cloudItsmTicketSlaEvaluation{
		Date:    start.Format("2006-01-02"),
		Counted: cloudItsmTicketSlaMinutes(ticket) > 0,
	}
	if !result.Counted {
		return result
	}
	if now.IsZero() {
		now = time.Now()
	}
	minutes := cloudItsmTicketSlaMinutes(ticket)
	deadline := start.Add(time.Duration(minutes) * time.Minute)
	terminal := cloudItsmTicketSlaTerminal(ticket.Status)
	success := ticket.Status == models.CloudItsmTicketStatusResolved || ticket.Status == models.CloudItsmTicketStatusClosed
	end := cloudItsmTicketSlaEndTime(ticket, now)
	if success && !end.After(deadline) {
		result.Met = true
		return result
	}
	if terminal || now.After(deadline) {
		result.Breached = true
	}
	return result
}

func cloudItsmTicketSlaMinutes(ticket models.CloudItsmTicket) int {
	return cloudItsmSlaMinutesFromPayload(ticket.RequestPayload)
}

func cloudItsmSlaMinutesFromPayload(payload models.ResAttrs) int {
	if payload == nil {
		return 0
	}
	if minutes := cloudSyncPolicyAttrInt(modelResAttrs(payload["sla"])["minutes"]); minutes > 0 {
		return minutes
	}
	if minutes := cloudSyncPolicyAttrInt(payload["slaMinutes"]); minutes > 0 {
		return minutes
	}
	operation := modelResAttrs(payload["operation"])
	params := modelResAttrs(operation["params"])
	if minutes := cloudSyncPolicyAttrInt(modelResAttrs(params["sla"])["minutes"]); minutes > 0 {
		return minutes
	}
	return cloudSyncPolicyAttrInt(params["slaMinutes"])
}

func cloudItsmTicketSlaStartTime(ticket models.CloudItsmTicket, fallback time.Time) time.Time {
	for _, candidate := range []models.Time{ticket.CreatedAt, ticket.SubmittedAt} {
		value := time.Time(candidate)
		if !value.IsZero() {
			return value
		}
	}
	if fallback.IsZero() {
		return time.Now()
	}
	return fallback
}

func cloudItsmTicketSlaEndTime(ticket models.CloudItsmTicket, fallback time.Time) time.Time {
	for _, candidate := range []models.Time{ticket.ClosedAt, ticket.LastSyncedAt, ticket.UpdatedAt} {
		value := time.Time(candidate)
		if !value.IsZero() {
			return value
		}
	}
	if fallback.IsZero() {
		return time.Now()
	}
	return fallback
}

func cloudItsmTicketSlaTerminal(status string) bool {
	switch status {
	case models.CloudItsmTicketStatusResolved,
		models.CloudItsmTicketStatusClosed,
		models.CloudItsmTicketStatusCanceled,
		models.CloudItsmTicketStatusFailed:
		return true
	default:
		return false
	}
}

func cloudItsmSlaMetRate(met int64, breached int64) float64 {
	total := met + breached
	if total <= 0 {
		return 0
	}
	return float64(met) / float64(total) * 100
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
	payload := models.ResAttrs{
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
	robot := cloudItsmRobotProcessingFromAttrs(operation.Params)
	if !attrBool(robot, "processed") && operation.OperationType == models.CloudOperationTypeSelfService {
		item, _ := cloudItsmCatalogItemByKey(nil, firstNonEmpty(cloudSyncPolicyAttrString(operation.Params["catalogKey"]), operation.Action))
		robot = cloudItsmRobotProcessingForSelfService(item, cfg, operation.Params)
	}
	cloudItsmAttachRobotProcessing(payload, robot)
	if policy := cloudItsmSelfServicePolicyFromParams(operation.Params); len(policy) > 0 {
		payload["selfServicePolicy"] = policy
	}
	if sla := cloudItsmSelfServiceSlaFromParams(operation.Params); len(sla) > 0 {
		payload["sla"] = sla
	}
	cloudItsmAttachExternalPayload(cfg, payload)
	return payload
}

func cloudItsmAttachExternalPayload(cfg models.CloudItsmConfig, payload models.ResAttrs) {
	if payload == nil {
		return
	}
	externalPayload := cloudItsmExternalRequestPayload(cfg, payload)
	if len(externalPayload) == 0 {
		return
	}
	payload["externalPayload"] = externalPayload
	payload["externalPayloadProvider"] = cloudItsmExternalPayloadProvider(cfg)
	payload["externalPayloadMode"] = cloudItsmExternalPayloadMode(cfg)
}

func cloudItsmExternalRequestPayload(cfg models.CloudItsmConfig, payload models.ResAttrs) models.ResAttrs {
	provider := cloudItsmExternalPayloadProvider(cfg)
	metadata := cloudItsmMetadata(cfg.Metadata)
	template := cloudItsmFirstAttrMap(metadata, "externalPayloadTemplate", "payloadTemplate", "requestPayloadTemplate")
	defaults := cloudItsmFirstAttrMap(metadata, "fieldDefaults", "defaultFields", "externalFieldDefaults")
	mappings := cloudItsmFirstAttrMap(metadata, "fieldMappings", "fieldMapping", "payloadMappings", "externalFieldMappings")

	var externalPayload models.ResAttrs
	switch provider {
	case models.CloudItsmProviderJira:
		externalPayload = cloudItsmJiraRequestPayload(cfg, payload)
	case models.CloudItsmProviderServiceNow:
		externalPayload = cloudItsmServiceNowRequestPayload(cfg, payload)
	default:
		if len(template) == 0 && len(defaults) == 0 && len(mappings) == 0 {
			return nil
		}
		externalPayload = models.ResAttrs{}
	}
	cloudItsmMergePayloadDefaults(externalPayload, template)
	cloudItsmMergePayloadDefaults(externalPayload, defaults)
	cloudItsmApplyExternalFieldMappings(externalPayload, payload, mappings)
	return externalPayload
}

func cloudItsmExternalPayloadProvider(cfg models.CloudItsmConfig) string {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		return models.CloudItsmProviderGeneric
	}
	return provider
}

func cloudItsmExternalPayloadMode(cfg models.CloudItsmConfig) string {
	metadata := cloudItsmMetadata(cfg.Metadata)
	hasCustomMapping := len(cloudItsmFirstAttrMap(metadata, "externalPayloadTemplate", "payloadTemplate", "requestPayloadTemplate")) > 0 ||
		len(cloudItsmFirstAttrMap(metadata, "fieldDefaults", "defaultFields", "externalFieldDefaults")) > 0 ||
		len(cloudItsmFirstAttrMap(metadata, "fieldMappings", "fieldMapping", "payloadMappings", "externalFieldMappings")) > 0
	switch cloudItsmExternalPayloadProvider(cfg) {
	case models.CloudItsmProviderJira:
		if hasCustomMapping || len(modelResAttrs(metadata["priorityMapping"])) > 0 || len(cloudSyncPolicyAttrStringSlice(metadata["labels"])) > 0 {
			return "provider_custom_field_mapping"
		}
		return "provider_default_field_mapping"
	case models.CloudItsmProviderServiceNow:
		if hasCustomMapping || len(modelResAttrs(metadata["urgencyMapping"])) > 0 || len(modelResAttrs(metadata["impactMapping"])) > 0 ||
			firstNonEmpty(attrString(metadata, "ticketTable"), attrString(metadata, "table"), attrString(metadata, "category"), attrString(metadata, "subcategory")) != "" {
			return "provider_custom_field_mapping"
		}
		return "provider_default_field_mapping"
	default:
		if hasCustomMapping {
			return "custom_field_mapping"
		}
		return ""
	}
}

func cloudItsmJiraRequestPayload(cfg models.CloudItsmConfig, payload models.ResAttrs) models.ResAttrs {
	metadata := cloudItsmMetadata(cfg.Metadata)
	priority := strings.ToLower(cloudItsmParamString(payload, "priority"))
	projectKey := firstNonEmpty(cfg.ProjectKey, cloudItsmParamString(payload, "projectKey"))
	ticketType := firstNonEmpty(cfg.TicketType, cloudItsmParamString(payload, "ticketType"), "Task")
	fields := models.ResAttrs{
		"summary":     cloudItsmParamString(payload, "title"),
		"description": cloudItsmParamString(payload, "description"),
		"issuetype":   models.ResAttrs{"name": ticketType},
		"labels":      cloudItsmJiraLabels(metadata, priority),
	}
	if projectKey != "" {
		fields["project"] = models.ResAttrs{"key": projectKey}
	}
	if priorityName := cloudItsmMappedExternalString(metadata["priorityMapping"], priority, cloudItsmDefaultJiraPriority(priority)); priorityName != "" {
		fields["priority"] = models.ResAttrs{"name": priorityName}
	}
	return models.ResAttrs{
		"fields": fields,
	}
}

func cloudItsmJiraLabels(metadata models.ResAttrs, priority string) []string {
	labels := append([]string{}, cloudSyncPolicyAttrStringSlice(metadata["labels"])...)
	labels = append(labels, "cloudiac")
	if priority != "" {
		labels = append(labels, "cloudiac_"+strings.ReplaceAll(priority, " ", "_"))
	}
	return normalizeStringList(dedupeStrings(labels))
}

func cloudItsmDefaultJiraPriority(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "critical":
		return "Highest"
	case "high":
		return "High"
	case "medium":
		return "Medium"
	case "low":
		return "Low"
	default:
		return ""
	}
}

func cloudItsmServiceNowRequestPayload(cfg models.CloudItsmConfig, payload models.ResAttrs) models.ResAttrs {
	metadata := cloudItsmMetadata(cfg.Metadata)
	priority := strings.ToLower(cloudItsmParamString(payload, "priority"))
	externalPayload := models.ResAttrs{
		"short_description":   cloudItsmParamString(payload, "title"),
		"description":         cloudItsmParamString(payload, "description"),
		"category":            firstNonEmpty(attrString(metadata, "category"), "cloud"),
		"subcategory":         firstNonEmpty(attrString(metadata, "subcategory"), "cloudiac"),
		"urgency":             cloudItsmMappedExternalString(metadata["urgencyMapping"], priority, cloudItsmDefaultServiceNowUrgency(priority)),
		"impact":              cloudItsmMappedExternalString(metadata["impactMapping"], priority, cloudItsmDefaultServiceNowImpact(priority)),
		"u_cloudiac_source":   "cloudiac",
		"u_cloudiac_priority": priority,
	}
	if table := firstNonEmpty(attrString(metadata, "ticketTable"), attrString(metadata, "table"), cfg.TicketType); table != "" {
		externalPayload["u_cloudiac_ticket_table"] = table
	}
	operation := modelResAttrs(payload["operation"])
	if len(operation) > 0 {
		externalPayload["u_cloudiac_operation_id"] = cloudSyncPolicyAttrString(operation["id"])
		externalPayload["u_cloudiac_project_id"] = cloudSyncPolicyAttrString(operation["projectId"])
		externalPayload["u_cloudiac_env_id"] = cloudSyncPolicyAttrString(operation["envId"])
		externalPayload["u_cloudiac_resource_type"] = cloudSyncPolicyAttrString(operation["resourceType"])
		externalPayload["u_cloudiac_resource_id"] = cloudSyncPolicyAttrString(operation["resourceId"])
	}
	event := modelResAttrs(payload["event"])
	if len(event) > 0 {
		externalPayload["u_cloudiac_event_id"] = cloudSyncPolicyAttrString(event["id"])
		externalPayload["u_cloudiac_event_type"] = cloudSyncPolicyAttrString(event["eventType"])
		externalPayload["u_cloudiac_project_id"] = cloudSyncPolicyAttrString(event["projectId"])
		externalPayload["u_cloudiac_env_id"] = cloudSyncPolicyAttrString(event["envId"])
		externalPayload["u_cloudiac_resource_type"] = cloudSyncPolicyAttrString(event["resourceType"])
		externalPayload["u_cloudiac_resource_id"] = cloudSyncPolicyAttrString(event["resourceId"])
	}
	return externalPayload
}

func cloudItsmDefaultServiceNowUrgency(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "critical", "high":
		return "1"
	case "medium":
		return "2"
	case "low":
		return "3"
	default:
		return "3"
	}
}

func cloudItsmDefaultServiceNowImpact(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "critical":
		return "1"
	case "high", "medium":
		return "2"
	case "low":
		return "3"
	default:
		return "3"
	}
}

func cloudItsmMappedExternalString(mapping interface{}, key string, fallback string) string {
	attrs := modelResAttrs(mapping)
	if len(attrs) == 0 {
		return fallback
	}
	key = strings.TrimSpace(key)
	for _, candidate := range []string{key, strings.ToLower(key), strings.ToUpper(key), "default"} {
		if value := strings.TrimSpace(attrString(attrs, candidate)); value != "" {
			return value
		}
	}
	return fallback
}

func cloudItsmFirstAttrMap(attrs models.ResAttrs, keys ...string) models.ResAttrs {
	for _, key := range keys {
		if value := modelResAttrs(attrs[key]); len(value) > 0 {
			return value
		}
	}
	return nil
}

func cloudItsmMergePayloadDefaults(target models.ResAttrs, defaults models.ResAttrs) {
	if target == nil || len(defaults) == 0 {
		return
	}
	for key, value := range defaults {
		if strings.Contains(key, ".") || strings.HasPrefix(key, "$") {
			cloudItsmSetPayloadPath(target, key, value)
			continue
		}
		nextValue := modelResAttrs(value)
		existing := modelResAttrs(target[key])
		if len(nextValue) > 0 && len(existing) > 0 {
			cloudItsmMergePayloadDefaults(existing, nextValue)
			continue
		}
		target[key] = cloudItsmClonePayloadValue(value)
	}
}

func cloudItsmApplyExternalFieldMappings(target models.ResAttrs, payload models.ResAttrs, mappings models.ResAttrs) {
	if target == nil || len(mappings) == 0 {
		return
	}
	for targetPath, source := range mappings {
		value, ok := cloudItsmExternalFieldMappingValue(payload, source)
		if !ok {
			continue
		}
		cloudItsmSetPayloadPath(target, targetPath, value)
	}
}

func cloudItsmExternalFieldMappingValue(payload models.ResAttrs, source interface{}) (interface{}, bool) {
	if source == nil {
		return nil, false
	}
	if sourceAttrs := modelResAttrs(source); len(sourceAttrs) > 0 {
		if path := firstNonEmpty(attrString(sourceAttrs, "path"), attrString(sourceAttrs, "source"), attrString(sourceAttrs, "from")); path != "" {
			if value, ok := cloudItsmPayloadPathValue(payload, path); ok {
				return value, true
			}
			if defaultValue, exists := sourceAttrs["default"]; exists {
				return cloudItsmClonePayloadValue(defaultValue), true
			}
			return nil, false
		}
		if value, exists := sourceAttrs["value"]; exists {
			return cloudItsmClonePayloadValue(value), true
		}
		return cloudItsmClonePayloadValue(source), true
	}
	if text, ok := source.(string); ok {
		trimmed := strings.TrimSpace(text)
		if strings.HasPrefix(trimmed, "$") || strings.HasPrefix(trimmed, ".") {
			return cloudItsmPayloadPathValue(payload, trimmed)
		}
		return text, true
	}
	return cloudItsmClonePayloadValue(source), true
}

func cloudItsmPayloadPathValue(payload models.ResAttrs, path string) (interface{}, bool) {
	if payload == nil {
		return nil, false
	}
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")
	if path == "" {
		return payload, true
	}
	current := interface{}(payload)
	for _, part := range strings.Split(path, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		attrs := modelResAttrs(current)
		if attrs == nil {
			return nil, false
		}
		value, ok := attrs[part]
		if !ok {
			return nil, false
		}
		current = value
	}
	return cloudItsmClonePayloadValue(current), true
}

func cloudItsmSetPayloadPath(target models.ResAttrs, path string, value interface{}) {
	if target == nil {
		return
	}
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")
	if path == "" {
		return
	}
	parts := strings.Split(path, ".")
	current := target
	for index, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if index == len(parts)-1 {
			current[part] = cloudItsmClonePayloadValue(value)
			return
		}
		next := modelResAttrs(current[part])
		if next == nil {
			next = models.ResAttrs{}
			current[part] = next
		}
		current = next
	}
}

func cloudItsmClonePayloadValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case models.ResAttrs:
		cloned := models.ResAttrs{}
		for key, item := range typed {
			cloned[key] = cloudItsmClonePayloadValue(item)
		}
		return cloned
	case map[string]interface{}:
		cloned := models.ResAttrs{}
		for key, item := range typed {
			cloned[key] = cloudItsmClonePayloadValue(item)
		}
		return cloned
	case []interface{}:
		cloned := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			cloned = append(cloned, cloudItsmClonePayloadValue(item))
		}
		return cloned
	case []string:
		return append([]string{}, typed...)
	default:
		return value
	}
}

func cloudItsmApplyRobotProcessingToParams(params models.ResAttrs, robot models.ResAttrs) {
	if params == nil {
		return
	}
	cloudItsmAttachRobotProcessing(params, robot)
}

func cloudItsmAttachRobotProcessing(payload models.ResAttrs, robot models.ResAttrs) {
	if payload == nil || !attrBool(robot, "processed") {
		return
	}
	tags := cloudItsmRobotProcessingTags(robot["tags"])
	robot["tags"] = tags
	payload["robot"] = robot
	payload["robotProcessed"] = true
	payload["robotProcessor"] = cloudSyncPolicyAttrString(robot["processor"])
	payload["robotProcessingTags"] = tags
	payload["robotAutomationMode"] = cloudSyncPolicyAttrString(robot["automationMode"])
	payload["robotTicketChannel"] = cloudSyncPolicyAttrString(robot["ticketChannel"])
}

func cloudItsmRobotProcessingForSelfService(item resps.CloudItsmCatalogItemResp, cfg models.CloudItsmConfig, params models.ResAttrs) models.ResAttrs {
	if params == nil {
		params = models.ResAttrs{}
	}
	requestType := firstNonEmpty(item.Key, cloudSyncPolicyAttrString(params["requestType"]), cloudSyncPolicyAttrString(params["catalogKey"]))
	automationMode := firstNonEmpty(item.AutomationMode, cloudSyncPolicyAttrString(params["automationMode"]), "itsm")
	tags := []string{cloudItsmRobotTagSelfService}
	switch requestType {
	case models.CloudOperationActionGitOpsIacChange:
		tags = append(tags, cloudItsmRobotTagGitOpsIac)
	case models.CloudOperationActionDriftRemediation:
		tags = append(tags, cloudItsmRobotTagDriftRemediation)
	case models.CloudOperationActionRiskRemediation:
		tags = append(tags, cloudItsmRobotTagRiskRemediation)
	case models.CloudOperationActionPermissionRequest:
		tags = append(tags, cloudItsmRobotTagPermission)
	}
	if attrBool(params, "gitOpsRequired") {
		tags = append(tags, cloudItsmRobotTagGitOpsIac)
	}
	if attrBool(params, "driftRemediationRequired") || attrBool(params, "driftRemediation") {
		tags = append(tags, cloudItsmRobotTagDriftRemediation)
	}
	if attrBool(params, "riskRemediationRequired") || attrBool(params, "riskRemediation") {
		tags = append(tags, cloudItsmRobotTagRiskRemediation)
	}
	if attrBool(params, "permissionWorkflowRequired") {
		tags = append(tags, cloudItsmRobotTagPermission)
	}
	requiresApproval := item.RequiresApproval || attrBool(params, "requiresApproval") || strings.Contains(automationMode, "approval")
	if requiresApproval {
		tags = append(tags, cloudItsmRobotTagApprovalRequired)
	}
	channel := cloudItsmRobotTicketChannel(cfg)
	tags = append(tags, cloudItsmRobotChannelTag(channel))
	return models.ResAttrs{
		"processed":        true,
		"processor":        cloudItsmRobotProcessorCloudiac,
		"tags":             cloudItsmRobotProcessingTags(tags),
		"automationMode":   automationMode,
		"ticketChannel":    channel,
		"catalogKey":       firstNonEmpty(item.Key, cloudSyncPolicyAttrString(params["catalogKey"]), requestType),
		"requestType":      requestType,
		"source":           firstNonEmpty(item.Source, "itsm_self_service"),
		"operationType":    firstNonEmpty(item.OperationType, models.CloudOperationTypeSelfService),
		"requiresApproval": requiresApproval,
		"gateStatus":       cloudSyncPolicyAttrString(params["gateStatus"]),
	}
}

func cloudItsmRobotProcessingForEvent(event models.CloudEvent, cfg models.CloudItsmConfig, spec cloudEventItsmDispatchSpec) models.ResAttrs {
	channel := cloudItsmRobotTicketChannel(cfg)
	tags := []string{
		cloudItsmRobotTagEventAutoTicket,
		cloudItsmRobotChannelTag(channel),
	}
	return models.ResAttrs{
		"processed":       true,
		"processor":       cloudItsmRobotProcessorCloudiac,
		"tags":            cloudItsmRobotProcessingTags(tags),
		"automationMode":  cloudItsmRobotModeEventTicket,
		"ticketChannel":   channel,
		"source":          "cloud_event",
		"requestType":     event.EventType,
		"dispatchReason":  spec.Reason,
		"failureCategory": cloudSyncPolicyAttrString(payloadValue(event.Payload, "failureCategory")),
		"cloudService":    cloudSyncPolicyAttrString(payloadValue(event.Payload, "cloudService")),
	}
}

func cloudItsmRobotProcessingFromPayload(payload models.ResAttrs) models.ResAttrs {
	robot := cloudItsmRobotProcessingFromAttrs(payload)
	if attrBool(robot, "processed") {
		return robot
	}
	operation := modelResAttrs(payload["operation"])
	if len(operation) == 0 {
		return robot
	}
	params := modelResAttrs(operation["params"])
	fromParams := cloudItsmRobotProcessingFromAttrs(params)
	if attrBool(fromParams, "processed") {
		return fromParams
	}
	return robot
}

func cloudItsmRobotProcessingFromAttrs(attrs models.ResAttrs) models.ResAttrs {
	if attrs == nil {
		return models.ResAttrs{}
	}
	robot := modelResAttrs(attrs["robot"])
	if robot == nil {
		robot = models.ResAttrs{}
	}
	if attrBool(attrs, "robotProcessed") {
		robot["processed"] = true
	}
	if robot["processor"] == nil {
		robot["processor"] = cloudSyncPolicyAttrString(attrs["robotProcessor"])
	}
	if robot["automationMode"] == nil {
		robot["automationMode"] = cloudSyncPolicyAttrString(attrs["robotAutomationMode"])
	}
	if robot["ticketChannel"] == nil {
		robot["ticketChannel"] = cloudSyncPolicyAttrString(attrs["robotTicketChannel"])
	}
	tags := cloudItsmRobotProcessingTags(firstNonNil(robot["tags"], attrs["robotProcessingTags"]))
	if len(tags) > 0 {
		robot["tags"] = tags
	}
	return robot
}

func cloudItsmRobotProcessingTags(value interface{}) []string {
	return normalizeStringList(cloudSyncPolicyAttrStringSlice(value))
}

func cloudItsmRobotTicketChannel(cfg models.CloudItsmConfig) string {
	if strings.TrimSpace(cfg.BaseUrl) == "" && strings.TrimSpace(attrString(cfg.Metadata, "createTicketUrl")) == "" {
		return cloudItsmRobotChannelLocal
	}
	return cloudItsmRobotChannelExternal
}

func cloudItsmRobotChannelTag(channel string) string {
	if channel == cloudItsmRobotChannelExternal {
		return cloudItsmRobotTagExternalItsm
	}
	return cloudItsmRobotTagLocalTicket
}

func submitCloudItsmTicket(cfg models.CloudItsmConfig, payload models.ResAttrs) cloudItsmSubmitResult {
	return submitCloudItsmTicketWithAttempt(cfg, payload, 1, time.Now())
}

func submitCloudItsmTicketWithAttempt(cfg models.CloudItsmConfig, payload models.ResAttrs, attempt int, now time.Time) cloudItsmSubmitResult {
	if attempt <= 0 {
		attempt = 1
	}
	result := cloudItsmSubmitResult{
		Status:      models.CloudItsmTicketStatusFailed,
		SubmittedAt: models.Time(now),
	}
	endpoint := cloudItsmTicketEndpoint(cfg)
	if endpoint == "" {
		result.Status = models.CloudItsmTicketStatusPending
		return result
	}
	submitPayload := cloudItsmSubmitPayload(cfg, payload)
	body, err := json.Marshal(submitPayload)
	if err != nil {
		result.ErrorMessage = err.Error()
		return cloudItsmAttachSubmitRetryResult(cfg, result, attempt, now)
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		result.ErrorMessage = err.Error()
		return cloudItsmAttachSubmitRetryResult(cfg, result, attempt, now)
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
		return cloudItsmAttachSubmitRetryResult(cfg, result, attempt, now)
	}
	defer resp.Body.Close()

	bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if readErr != nil {
		result.ErrorMessage = readErr.Error()
	}
	responsePayload := models.ResAttrs{
		"endpoint":               endpoint,
		"statusCode":             resp.StatusCode,
		"body":                   string(bodyBytes),
		"requestPayloadMode":     cloudItsmSubmitPayloadMode(payload),
		"requestPayloadProvider": cloudItsmExternalPayloadProvider(cfg),
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
		if nested, ok := decoded["result"].(map[string]interface{}); ok {
			nestedAttrs := models.ResAttrs(nested)
			result.ExternalId = firstNonEmpty(result.ExternalId, attrString(nestedAttrs, "id"), attrString(nestedAttrs, "sys_id"), attrString(nestedAttrs, "ticketId"))
			result.ExternalKey = firstNonEmpty(result.ExternalKey, attrString(nestedAttrs, "key"), attrString(nestedAttrs, "number"), attrString(nestedAttrs, "display_value"))
			result.ExternalUrl = firstNonEmpty(result.ExternalUrl, attrString(nestedAttrs, "url"), attrString(nestedAttrs, "self"), attrString(nestedAttrs, "link"))
		}
	}
	result.ResponsePayload = responsePayload
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Status = models.CloudItsmTicketStatusSubmitted
		if result.ExternalUrl == "" && result.ExternalKey != "" {
			result.ExternalUrl = cloudItsmTicketExternalUrl(cfg, result.ExternalKey)
		}
		return cloudItsmAttachSubmitRetryResult(cfg, result, attempt, now)
	}
	result.ErrorMessage = fmt.Sprintf("ITSM 工单接口返回 HTTP %d", resp.StatusCode)
	return cloudItsmAttachSubmitRetryResult(cfg, result, attempt, now)
}

func cloudItsmSubmitPayload(cfg models.CloudItsmConfig, payload models.ResAttrs) models.ResAttrs {
	if externalPayload := modelResAttrs(payload["externalPayload"]); len(externalPayload) > 0 {
		return externalPayload
	}
	if externalPayload := cloudItsmExternalRequestPayload(cfg, payload); len(externalPayload) > 0 {
		if payload != nil {
			payload["externalPayload"] = externalPayload
			payload["externalPayloadProvider"] = cloudItsmExternalPayloadProvider(cfg)
			payload["externalPayloadMode"] = cloudItsmExternalPayloadMode(cfg)
		}
		return externalPayload
	}
	return payload
}

func cloudItsmSubmitPayloadMode(payload models.ResAttrs) string {
	if payload == nil {
		return "cloudiac_canonical"
	}
	if externalPayload := modelResAttrs(payload["externalPayload"]); len(externalPayload) > 0 {
		return firstNonEmpty(cloudSyncPolicyAttrString(payload["externalPayloadMode"]), "external_field_mapping")
	}
	return "cloudiac_canonical"
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
