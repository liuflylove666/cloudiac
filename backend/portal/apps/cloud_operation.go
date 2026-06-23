// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"cloudiac/portal/consts"
	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/libs/page"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	"cloudiac/portal/models/resps"
	"cloudiac/portal/services"
)

func SearchCloudOperations(c *ctx.ServiceContext, form *forms.SearchCloudOperationForm) (interface{}, e.Error) {
	if err := recoverStaleCloudOperations(c); err != nil {
		return nil, err
	}
	query := c.DB().Model(&models.CloudOperation{}).Where("org_id = ?", c.OrgId)
	if form.Q != "" {
		qs := "%" + strings.TrimSpace(form.Q) + "%"
		query = query.Where("id like ? or name like ? or message like ? or resource_name like ? or resource_id like ?", qs, qs, qs, qs, qs)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	if form.Action != "" {
		query = query.Where("action = ?", form.Action)
	}
	if form.OperationType != "" {
		query = query.Where("operation_type = ?", form.OperationType)
	}
	if form.RiskLevel != "" {
		query = query.Where("risk_level = ?", form.RiskLevel)
	}
	if form.AssetId != "" {
		query = query.Where("asset_id = ?", form.AssetId)
	}
	if form.ProjectId != "" {
		query = query.Where("project_id = ?", form.ProjectId)
	}
	if form.EnvId != "" {
		query = query.Where("env_id = ?", form.EnvId)
	}
	if form.SortField() == "" {
		query = query.Order("created_at desc")
	} else {
		query = form.Order(query)
	}

	operations := make([]models.CloudOperation, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&operations); err != nil {
		return nil, e.New(e.DBError, err)
	}

	list := make([]resps.CloudOperationResp, 0, len(operations))
	for _, operation := range operations {
		list = append(list, cloudOperationResp(c, operation))
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func CloudOperationDetail(c *ctx.ServiceContext, form *forms.CloudOperationParam) (*resps.CloudOperationDetailResp, e.Error) {
	if err := recoverStaleCloudOperations(c); err != nil {
		return nil, err
	}
	operation, err := getCloudOperation(c, form.Id)
	if err != nil {
		return nil, err
	}

	steps := make([]models.CloudOperationStep, 0)
	if err := c.DB().Model(&models.CloudOperationStep{}).
		Where("org_id = ? and operation_id = ?", c.OrgId, form.Id).
		Order("`index`, created_at").
		Scan(&steps); err != nil {
		return nil, e.New(e.DBError, err)
	}
	audits, err := cloudOperationAudits(c, form.Id)
	if err != nil {
		return nil, err
	}

	resp := &resps.CloudOperationDetailResp{
		CloudOperationResp: cloudOperationResp(c, *operation),
		Steps:              make([]resps.CloudOperationStepResp, 0, len(steps)),
		Audits:             audits,
	}
	for _, step := range steps {
		resp.Steps = append(resp.Steps, resps.CloudOperationStepResp{CloudOperationStep: step})
	}
	return resp, nil
}

func CloudOperationAudits(c *ctx.ServiceContext, form *forms.CloudOperationParam) ([]resps.CloudOperationAuditResp, e.Error) {
	if _, err := getCloudOperation(c, form.Id); err != nil {
		return nil, err
	}
	return cloudOperationAudits(c, form.Id)
}

func CancelCloudOperation(c *ctx.ServiceContext, form *forms.CloudOperationParam) (*resps.CloudOperationDetailResp, e.Error) {
	operation, err := getCloudOperation(c, form.Id)
	if err != nil {
		return nil, err
	}
	if err := ensureCloudOperationMutationPermission(c, operation, "取消"); err != nil {
		return nil, err
	}
	if operation.Status != models.CloudOperationStatusPending &&
		operation.Status != models.CloudOperationStatusApproving &&
		operation.Status != models.CloudOperationStatusRunning {
		return nil, e.New(e.BadParam, fmt.Errorf("操作任务 %s 当前状态不允许取消", form.Id))
	}
	result := models.ResAttrs{
		"canceledBy":     c.UserId.String(),
		"canceledAt":     time.Now().Format(time.RFC3339),
		"previousStatus": operation.Status,
	}
	if attrString(operation.Params, "executionMode") == cloudOperationExecutionModeAsync {
		result["executionMode"] = cloudOperationExecutionModeAsync
		result["providerOperationMayContinue"] = true
		result["rollbackHint"] = "取消仅停止平台任务跟踪，已提交到云厂商的 provider 操作不会自动回滚，请到云厂商控制台或后续采集结果核对最终状态"
	}
	if err := finishCloudOperation(c, operation, models.CloudOperationStatusAborted, "用户取消操作任务", result); err != nil {
		return nil, err
	}
	return CloudOperationDetail(c, form)
}

func RetryCloudOperation(c *ctx.ServiceContext, form *forms.CloudOperationParam) (*resps.CloudOperationDetailResp, e.Error) {
	operation, err := getCloudOperation(c, form.Id)
	if err != nil {
		return nil, err
	}
	if err := ensureCloudOperationMutationPermission(c, operation, "重试"); err != nil {
		return nil, err
	}
	if operation.Status != models.CloudOperationStatusFailed {
		return nil, e.New(e.BadParam, fmt.Errorf("操作任务 %s 当前状态不是失败，不能重试", form.Id))
	}
	if operation.OperationType != models.CloudOperationTypeAction {
		return nil, e.New(e.BadParam, fmt.Errorf("操作任务 %s 不是云资产动作任务，暂不支持重试", form.Id))
	}
	if operation.AssetId == "" || operation.Action == "" {
		return nil, e.New(e.BadParam, fmt.Errorf("操作任务 %s 缺少资产或动作信息，不能重试", form.Id))
	}

	confirmation := modelResAttrs(operation.Params["confirmation"])
	retryForm := &forms.CreateCloudAssetActionForm{
		Id:                   operation.AssetId,
		Action:               operation.Action,
		Reason:               firstNonEmpty(attrString(operation.Params, "reason"), fmt.Sprintf("重试操作任务 %s", operation.Id)),
		Params:               modelResAttrs(operation.Params["params"]),
		Tags:                 modelResAttrs(operation.Params["tags"]),
		ConfirmAction:        attrString(confirmation, "action"),
		ConfirmResourceId:    attrString(confirmation, "resourceId"),
		RetryFromOperationId: operation.Id,
	}
	detail, retryOperation, retryErr := createCloudAssetAction(c, retryForm)
	retryParams := models.ResAttrs{
		"retryFromOperationId": operation.Id.String(),
	}
	if retryOperation != nil {
		retryParams["retryOperationId"] = retryOperation.Id.String()
	}
	if retryErr != nil {
		if auditErr := recordCloudOperationAudit(c, operation, operation.Status, fmt.Sprintf("重试操作任务失败：%s", retryErr.Error()), retryParams, models.ResAttrs{
			"error": retryErr.Error(),
		}); auditErr != nil {
			return nil, auditErr
		}
		return nil, retryErr
	}
	if err := recordCloudOperationAudit(c, operation, operation.Status, fmt.Sprintf("已创建重试任务 %s", detail.Id), retryParams, nil); err != nil {
		return nil, err
	}
	return detail, nil
}

func ApproveCloudOperation(c *ctx.ServiceContext, form *forms.CloudOperationApprovalForm) (*resps.CloudOperationDetailResp, e.Error) {
	operation, err := getCloudOperation(c, form.Id)
	if err != nil {
		return nil, err
	}
	if operation.Status != models.CloudOperationStatusApproving {
		return nil, e.New(e.BadParam, fmt.Errorf("操作任务 %s 当前状态不是待审批，不能审批", form.Id))
	}
	if err := ensureCloudOperationApprovalPermission(c, operation); err != nil {
		return nil, err
	}

	approval := cloudOperationApprovalAttrs(operation, c, form)
	if operation.OperationType == models.CloudOperationTypeSelfService &&
		operation.Action == models.CloudOperationActionItsmDeadLetter {
		return approveCloudItsmDeadLetterOperation(c, operation, form, approval)
	}
	if form.Action == "rejected" {
		result := models.ResAttrs{
			"approval": approval,
		}
		if err := finishCloudOperation(c, operation, models.CloudOperationStatusRejected, "云资产操作审批驳回", result); err != nil {
			return nil, err
		}
		return CloudOperationDetail(c, &forms.CloudOperationParam{Id: operation.Id})
	}

	asset, err := getCloudAssetForOperation(c, operation.AssetId)
	if err != nil {
		return nil, err
	}
	spec, ok := cloudAssetActionSpecByKey(operation.Action)
	if !ok || !cloudAssetActionVisibleForAsset(asset, spec) {
		return nil, e.New(e.BadParam, fmt.Errorf("云资产动作 %s 不支持", operation.Action))
	}
	dryRun := cloudAssetActionDryRun(c, asset, spec)
	if err := cloudAssetActionExecutionGuard(dryRun); err != nil {
		return nil, err
	}
	if err := markCloudOperationApproved(c, operation, approval); err != nil {
		return nil, err
	}
	if spec.AdapterKey == cloudOperationAdapterProviderOperation {
		go runCloudAssetActionOperation(operation.Id, c, asset, spec, dryRun)
		return CloudOperationDetail(c, &forms.CloudOperationParam{Id: operation.Id})
	}
	if err := startCloudOperation(c, operation, "审批通过，开始执行云资产操作任务"); err != nil {
		return nil, err
	}
	result, execErr := executeCloudAssetAction(c, operation, asset, spec, dryRun)
	if execErr != nil {
		if result == nil {
			result = models.ResAttrs{}
		}
		result["error"] = execErr.Error()
		if finishErr := finishCloudOperation(c, operation, models.CloudOperationStatusFailed, fmt.Sprintf("云资产操作任务失败：%s", execErr.Error()), result); finishErr != nil {
			return nil, finishErr
		}
		return nil, execErr
	}
	if err := finishCloudOperation(c, operation, models.CloudOperationStatusComplete, "云资产操作任务完成", result); err != nil {
		return nil, err
	}
	return CloudOperationDetail(c, &forms.CloudOperationParam{Id: operation.Id})
}

type cloudAssetActionSpec struct {
	Key                 string
	Name                string
	Description         string
	RiskLevel           string
	AdapterKey          string
	AdapterMode         string
	Enabled             bool
	Destructive         bool
	RequiresApproval    bool
	DisabledReason      string
	SupportedProviders  []string
	SupportedAssetTypes []string
}

type cloudOperationAdapterCapability struct {
	AdapterKey      string
	AdapterMode     string
	AdapterStatus   string
	AdapterMessage  string
	ProviderAdapter string
	WriteEnabled    bool
}

type cloudProviderOperationRequest struct {
	OperationId     models.Id
	IdempotencyKey  string
	Action          string
	Provider        string
	ProviderAdapter string
	ResourceType    string
	ResourceId      string
	ResourceName    string
	Region          string
	Zone            string
	AccountId       string
	CloudAccountId  models.Id
	ObservedState   string
	TargetState     string
	ReadMode        string
	Params          models.ResAttrs
}

type cloudProviderResourceState struct {
	RawState        string
	NormalizedState string
	Source          string
	ReadMode        string
	RawResponse     models.ResAttrs
	ErrorCode       string
	ErrorMessage    string
	Retryable       bool
}

type cloudSecurityRuleOperation struct {
	Operation   string
	Direction   string
	Protocol    string
	Cidr        string
	FromPort    int
	ToPort      int
	Description string
	RuleId      string
}

type cloudProviderOperationAdapter interface {
	Key() string
	Provider() string
	BuildRequest(asset *models.CmdbAsset, spec cloudAssetActionSpec, operation *models.CloudOperation, dryRun *resps.CloudAssetActionDryRunResp) cloudProviderOperationRequest
	ReadResourceState(request cloudProviderOperationRequest, account *cmdbCloudAccount) cloudProviderResourceState
	ValidateTransition(request cloudProviderOperationRequest, state cloudProviderResourceState) []resps.CloudAssetActionCheckResp
	Execute(request cloudProviderOperationRequest, state cloudProviderResourceState, account *cmdbCloudAccount) (models.ResAttrs, e.Error)
}

type cloudComputeInstanceOperationAdapter struct {
	provider string
	key      string
}

type cloudBlockVolumeOperationAdapter struct {
	provider string
	key      string
}

type cloudSecurityGroupOperationAdapter struct {
	provider string
	key      string
}

const (
	cloudOperationAdapterLocalMetadata     = "local_metadata"
	cloudOperationAdapterProviderOperation = "provider_operation"
	cloudOperationAdapterModeLocal         = "local"
	cloudOperationAdapterModeProvider      = "provider"
	cloudOperationAdapterStatusReady       = "ready"
	cloudOperationAdapterStatusRegistered  = "registered"
	cloudOperationAdapterStatusBlocked     = "blocked"
	cloudOperationAdapterStatusUnsupported = "unsupported"
	cloudOperationProviderWriteEnv         = "CLOUD_OPERATION_PROVIDER_WRITE_ENABLED"
	cloudOperationProviderReadModeEnv      = "CLOUD_OPERATION_PROVIDER_READ_MODE"
	cloudOperationProviderReadModeCache    = "cache"
	cloudOperationProviderReadModeLive     = "live"
	cloudOperationProviderReadTimeoutEnv   = "CLOUD_OPERATION_PROVIDER_READ_TIMEOUT_SECONDS"
	cloudOperationProviderWriteTimeoutEnv  = "CLOUD_OPERATION_PROVIDER_WRITE_TIMEOUT_SECONDS"
	cloudOperationProviderPollAttemptsEnv  = "CLOUD_OPERATION_PROVIDER_POLL_ATTEMPTS"
	cloudOperationProviderPollIntervalEnv  = "CLOUD_OPERATION_PROVIDER_POLL_INTERVAL_SECONDS"
	cloudOperationProviderPollMaxDelayEnv  = "CLOUD_OPERATION_PROVIDER_POLL_MAX_DELAY_SECONDS"
	cloudOperationStaleMinutesEnv          = "CLOUD_OPERATION_STALE_MINUTES"
	cloudOperationExecutionModeSync        = "sync"
	cloudOperationExecutionModeAsync       = "async"
)

func CloudAssetActions(c *ctx.ServiceContext, form *forms.CloudAssetActionParam) ([]resps.CloudAssetActionResp, e.Error) {
	asset, err := getCloudAssetForOperation(c, form.Id)
	if err != nil {
		return nil, err
	}

	actions := make([]resps.CloudAssetActionResp, 0)
	for _, spec := range cloudAssetActionSpecs() {
		if !cloudAssetActionVisibleForAsset(asset, spec) {
			continue
		}
		actions = append(actions, cloudAssetActionResp(c, asset, spec))
	}
	return actions, nil
}

func DryRunCloudAssetAction(c *ctx.ServiceContext, form *forms.DryRunCloudAssetActionForm) (*resps.CloudAssetActionDryRunResp, e.Error) {
	asset, err := getCloudAssetForOperation(c, form.Id)
	if err != nil {
		return nil, err
	}
	spec, ok := cloudAssetActionSpecByKey(form.Action)
	if !ok || !cloudAssetActionVisibleForAsset(asset, spec) {
		return nil, e.New(e.BadParam, fmt.Errorf("云资产动作 %s 不支持", form.Action))
	}
	return cloudAssetActionDryRunWithParams(c, asset, spec, cloudAssetActionParamsFromForm(form.Params)), nil
}

func CreateCloudAssetAction(c *ctx.ServiceContext, form *forms.CreateCloudAssetActionForm) (*resps.CloudOperationDetailResp, e.Error) {
	detail, _, err := createCloudAssetAction(c, form)
	return detail, err
}

func createCloudAssetAction(c *ctx.ServiceContext, form *forms.CreateCloudAssetActionForm) (*resps.CloudOperationDetailResp, *models.CloudOperation, e.Error) {
	asset, err := getCloudAssetForOperation(c, form.Id)
	if err != nil {
		return nil, nil, err
	}
	spec, ok := cloudAssetActionSpecByKey(form.Action)
	if !ok || !cloudAssetActionVisibleForAsset(asset, spec) {
		return nil, nil, e.New(e.BadParam, fmt.Errorf("云资产动作 %s 不支持", form.Action))
	}
	dryRun := cloudAssetActionDryRun(c, asset, spec)
	if err := cloudAssetActionExecutionGuard(dryRun); err != nil {
		return nil, nil, err
	}
	if err := cloudAssetActionConfirmationGuard(asset, spec, form); err != nil {
		return nil, nil, err
	}
	actionTags := cloudAssetActionTagsFromForm(form.Tags)
	actionParams := cloudAssetActionParamsFromForm(form.Params)
	if err := cloudAssetActionParamsGuard(asset, spec, actionTags, actionParams); err != nil {
		return nil, nil, err
	}

	resourceName := firstNonEmpty(asset.Name, asset.NativeId, asset.Id.String())
	reason := strings.TrimSpace(form.Reason)
	executionMode := cloudOperationExecutionModeSync
	operationStatus := models.CloudOperationStatusRunning
	operationMessage := "正在记录云资产操作任务"
	if spec.AdapterKey == cloudOperationAdapterProviderOperation {
		executionMode = cloudOperationExecutionModeAsync
		operationStatus = models.CloudOperationStatusPending
		operationMessage = "云资产操作任务已创建，等待后台执行"
	}
	approvalId := models.Id("")
	if spec.RequiresApproval {
		approvalId = models.NewId("cpa")
		operationStatus = models.CloudOperationStatusApproving
		operationMessage = "云资产操作任务已创建，等待审批"
	}
	params := models.ResAttrs{
		"assetId":        asset.Id.String(),
		"action":         spec.Key,
		"reason":         reason,
		"provider":       asset.Provider,
		"resourceType":   asset.AssetType,
		"resourceId":     asset.NativeId,
		"cloudAccountId": asset.CloudAccountId.String(),
		"executionMode":  executionMode,
		"confirmation": models.ResAttrs{
			"action":     strings.TrimSpace(form.ConfirmAction),
			"resourceId": strings.TrimSpace(form.ConfirmResourceId),
		},
	}
	if spec.Key == models.CloudOperationActionUpdateTags {
		params["tags"] = actionTags
	}
	if len(actionParams) > 0 {
		params["params"] = actionParams
	}
	if spec.RequiresApproval {
		params["approval"] = models.ResAttrs{
			"required":    true,
			"approvalId":  approvalId.String(),
			"status":      "pending",
			"requestedBy": c.UserId.String(),
			"requestedAt": time.Now().Format(time.RFC3339),
		}
	}
	if form.RetryFromOperationId != "" {
		params["retryFromOperationId"] = form.RetryFromOperationId.String()
	}
	operation := &models.CloudOperation{
		OrgId:          c.OrgId,
		ProjectId:      asset.ProjectId,
		EnvId:          asset.EnvId,
		AssetId:        asset.Id,
		CloudAccountId: asset.CloudAccountId,
		CreatorId:      c.UserId,
		ApprovalId:     approvalId,
		Name:           fmt.Sprintf("%s - %s", spec.Name, resourceName),
		OperationType:  models.CloudOperationTypeAction,
		Action:         spec.Key,
		Status:         operationStatus,
		RiskLevel:      spec.RiskLevel,
		Provider:       asset.Provider,
		ResourceType:   asset.AssetType,
		ResourceId:     asset.NativeId,
		ResourceName:   resourceName,
		Message:        operationMessage,
		Params:         params,
	}
	if err := createCloudOperation(c, operation, "创建云资产操作任务"); err != nil {
		return nil, operation, err
	}
	if spec.RequiresApproval {
		detail, detailErr := CloudOperationDetail(c, &forms.CloudOperationParam{Id: operation.Id})
		return detail, operation, detailErr
	}
	if spec.AdapterKey == cloudOperationAdapterProviderOperation {
		go runCloudAssetActionOperation(operation.Id, c, asset, spec, dryRun)
		detail, detailErr := CloudOperationDetail(c, &forms.CloudOperationParam{Id: operation.Id})
		return detail, operation, detailErr
	}
	result, err := executeCloudAssetAction(c, operation, asset, spec, dryRun)
	if err != nil {
		if result == nil {
			result = models.ResAttrs{}
		}
		result["error"] = err.Error()
		if finishErr := finishCloudOperation(c, operation, models.CloudOperationStatusFailed, fmt.Sprintf("云资产操作任务失败：%s", err.Error()), result); finishErr != nil {
			return nil, operation, finishErr
		}
		return nil, operation, err
	}
	if err := finishCloudOperation(c, operation, models.CloudOperationStatusComplete, "云资产操作任务完成", result); err != nil {
		return nil, operation, err
	}
	detail, detailErr := CloudOperationDetail(c, &forms.CloudOperationParam{Id: operation.Id})
	return detail, operation, detailErr
}

func runCloudAssetActionOperation(operationId models.Id, requestCtx *ctx.ServiceContext, asset *models.CmdbAsset, spec cloudAssetActionSpec, dryRun *resps.CloudAssetActionDryRunResp) {
	workerCtx := cloudOperationWorkerContext(requestCtx)
	operation, lookupErr := getCloudOperation(workerCtx, operationId)
	if lookupErr != nil {
		return
	}
	if operation.Status == models.CloudOperationStatusAborted {
		return
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			result := models.ResAttrs{
				"error": fmt.Sprintf("cloud asset action panic: %v", recovered),
				"stack": string(debug.Stack()),
			}
			if !cloudOperationWasAborted(workerCtx, operation.Id) {
				_ = finishCloudOperation(workerCtx, operation, models.CloudOperationStatusFailed, fmt.Sprintf("云资产操作任务失败：%v", recovered), result)
			}
		}
	}()

	if err := startCloudOperation(workerCtx, operation, "后台执行云资产操作任务"); err != nil {
		return
	}
	result, err := executeCloudAssetAction(workerCtx, operation, asset, spec, dryRun)
	if cloudOperationWasAborted(workerCtx, operation.Id) {
		return
	}
	if err != nil {
		if result == nil {
			result = models.ResAttrs{}
		}
		result["error"] = err.Error()
		_ = finishCloudOperation(workerCtx, operation, models.CloudOperationStatusFailed, fmt.Sprintf("云资产操作任务失败：%s", err.Error()), result)
		return
	}
	_ = finishCloudOperation(workerCtx, operation, models.CloudOperationStatusComplete, "云资产操作任务完成", result)
}

func cloudOperationWorkerContext(requestCtx *ctx.ServiceContext) *ctx.ServiceContext {
	if requestCtx == nil {
		return &ctx.ServiceContext{}
	}
	return &ctx.ServiceContext{
		UserId:       requestCtx.UserId,
		OrgId:        requestCtx.OrgId,
		ProjectId:    requestCtx.ProjectId,
		Email:        requestCtx.Email,
		Username:     requestCtx.Username,
		IsSuperAdmin: requestCtx.IsSuperAdmin,
		UserIpAddr:   requestCtx.UserIpAddr,
	}
}

func getCloudOperation(c *ctx.ServiceContext, id models.Id) (*models.CloudOperation, e.Error) {
	operation := models.CloudOperation{}
	if err := c.DB().Model(&models.CloudOperation{}).
		Where("id = ? and org_id = ?", id, c.OrgId).
		First(&operation); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.BadParam, fmt.Errorf("操作任务 %s 不存在或不属于当前组织", id))
		}
		return nil, e.New(e.DBError, err)
	}
	return &operation, nil
}

func createCloudOperation(c *ctx.ServiceContext, operation *models.CloudOperation, summary string) e.Error {
	now := models.Time(time.Now())
	if operation.Id == "" {
		operation.Id = models.NewId("cop")
	}
	if operation.OrgId == "" {
		operation.OrgId = c.OrgId
	}
	if operation.CreatorId == "" {
		operation.CreatorId = c.UserId
	}
	if operation.OperationType == "" {
		operation.OperationType = models.CloudOperationTypeGovernance
	}
	if operation.Status == "" {
		operation.Status = models.CloudOperationStatusRunning
	}
	if operation.RiskLevel == "" {
		operation.RiskLevel = models.CloudOperationRiskLow
	}
	operation.StartedAt = now
	if err := models.Create(c.DB(), operation); err != nil {
		return e.New(e.DBError, err)
	}
	if err := recordCloudOperationAudit(c, operation, operation.Status, summary, operation.Params, nil); err != nil {
		return err
	}
	cloudOperationEvent(c, operation, "cloud_operation.created", "云操作任务创建", summary, operation.Params)
	return nil
}

func startCloudOperation(c *ctx.ServiceContext, operation *models.CloudOperation, message string) e.Error {
	if operation == nil {
		return e.New(e.BadParam, fmt.Errorf("操作任务不存在"))
	}
	now := models.Time(time.Now())
	attrs := models.Attrs{
		"status":     models.CloudOperationStatusRunning,
		"message":    message,
		"started_at": now,
	}
	affected, err := c.DB().Model(&models.CloudOperation{}).
		Where("id = ? and org_id = ?", operation.Id, operation.OrgId).
		Where("status = ?", models.CloudOperationStatusPending).
		UpdateAttrs(attrs)
	if err != nil {
		return e.New(e.DBError, err)
	}
	if affected == 0 {
		return e.New(e.BadParam, fmt.Errorf("操作任务 %s 已取消或不再处于等待状态，停止后台执行", operation.Id))
	}
	operation.Status = models.CloudOperationStatusRunning
	operation.Message = message
	operation.StartedAt = now
	return recordCloudOperationAudit(c, operation, operation.Status, message, operation.Params, nil)
}

func finishCloudOperation(c *ctx.ServiceContext, operation *models.CloudOperation, status string, message string, result models.ResAttrs) e.Error {
	now := models.Time(time.Now())
	attrs := models.Attrs{
		"status":   status,
		"message":  message,
		"result":   result,
		"ended_at": now,
	}
	query := c.DB().Model(&models.CloudOperation{}).Where("id = ? and org_id = ?", operation.Id, operation.OrgId)
	switch status {
	case models.CloudOperationStatusComplete, models.CloudOperationStatusFailed:
		query = query.Where("status in (?, ?)", models.CloudOperationStatusPending, models.CloudOperationStatusRunning)
	case models.CloudOperationStatusAborted:
		query = query.Where("status in (?, ?, ?)", models.CloudOperationStatusPending, models.CloudOperationStatusApproving, models.CloudOperationStatusRunning)
	case models.CloudOperationStatusRejected:
		query = query.Where("status = ?", models.CloudOperationStatusApproving)
	}
	affected, err := query.UpdateAttrs(attrs)
	if err != nil {
		return e.New(e.DBError, err)
	}
	if affected == 0 {
		if status == models.CloudOperationStatusAborted {
			return e.New(e.BadParam, fmt.Errorf("操作任务 %s 当前状态不允许取消", operation.Id))
		}
		return nil
	}
	operation.Status = status
	operation.Message = message
	operation.Result = result
	operation.EndedAt = now

	step := models.CloudOperationStep{
		OrgId:       operation.OrgId,
		OperationId: operation.Id,
		Index:       1,
		Name:        cloudOperationStepName(operation.OperationType),
		Status:      status,
		Message:     message,
		Result:      result,
		StartedAt:   operation.StartedAt,
		EndedAt:     now,
	}
	step.Id = models.NewId("cos")
	if err := models.Create(c.DB(), &step); err != nil {
		return e.New(e.DBError, err)
	}
	if err := recordCloudOperationAudit(c, operation, status, message, operation.Params, result); err != nil {
		return err
	}
	cloudOperationEvent(c, operation, "cloud_operation."+status, "云操作任务状态变更", message, result)
	return nil
}

func isCloudOperationFinalStatus(status string) bool {
	return status == models.CloudOperationStatusComplete ||
		status == models.CloudOperationStatusFailed ||
		status == models.CloudOperationStatusAborted ||
		status == models.CloudOperationStatusRejected
}

func cloudOperationWasAborted(c *ctx.ServiceContext, operationId models.Id) bool {
	if c == nil || operationId == "" {
		return false
	}
	operation, err := getCloudOperation(c, operationId)
	return err == nil && operation.Status == models.CloudOperationStatusAborted
}

func recoverStaleCloudOperations(c *ctx.ServiceContext) e.Error {
	if c == nil || c.OrgId == "" {
		return nil
	}
	staleAfter := cloudOperationStaleDuration()
	cutoff := time.Now().Add(-staleAfter)
	operations := make([]models.CloudOperation, 0)
	if err := c.DB().Model(&models.CloudOperation{}).
		Where("org_id = ? and operation_type = ? and status in (?, ?) and updated_at < ?",
			c.OrgId, models.CloudOperationTypeAction, models.CloudOperationStatusPending, models.CloudOperationStatusRunning, cutoff).
		Scan(&operations); err != nil {
		return e.New(e.DBError, err)
	}
	for index := range operations {
		operation := &operations[index]
		if attrString(operation.Params, "executionMode") != cloudOperationExecutionModeAsync {
			continue
		}
		result := operation.Result
		if result == nil {
			result = models.ResAttrs{}
		}
		result["recoveredBy"] = "stale_cloud_operation_recovery"
		result["previousStatus"] = operation.Status
		result["staleAfter"] = staleAfter.String()
		result["reason"] = "Portal 后台执行可能因重启或异常中断，平台未自动重放 provider 写操作"
		if err := finishCloudOperation(c, operation, models.CloudOperationStatusFailed, "云资产操作任务超时未完成，已由恢复机制标记失败", result); err != nil {
			return err
		}
	}
	return nil
}

func cloudOperationStaleDuration() time.Duration {
	minutes := 30
	if value := strings.TrimSpace(os.Getenv(cloudOperationStaleMinutesEnv)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			minutes = parsed
		}
	}
	return time.Duration(minutes) * time.Minute
}

func cloudOperationStepName(operationType string) string {
	if operationType == models.CloudOperationTypeAction {
		return "记录操作结果"
	}
	return "记录治理结果"
}

func recordCloudOperationAudit(c *ctx.ServiceContext, operation *models.CloudOperation, status string, summary string, params models.ResAttrs, result models.ResAttrs) e.Error {
	audit := models.CloudOperationAudit{
		OrgId:       operation.OrgId,
		OperationId: operation.Id,
		AssetId:     operation.AssetId,
		OperatorId:  c.UserId,
		Action:      operation.Action,
		Status:      status,
		Summary:     summary,
		Params:      params,
		Result:      result,
		UserIp:      c.UserIpAddr,
	}
	audit.Id = models.NewId("coa")
	if err := models.Create(c.DB(), &audit); err != nil {
		return e.New(e.DBError, err)
	}
	return nil
}

func cloudOperationResp(c *ctx.ServiceContext, operation models.CloudOperation) resps.CloudOperationResp {
	return resps.CloudOperationResp{
		CloudOperation: operation,
		CreatorName:    lookupName(c, &models.User{}, operation.CreatorId),
		ProjectName:    lookupName(c, &models.Project{}, operation.ProjectId),
		EnvName:        lookupName(c, &models.Env{}, operation.EnvId),
		AssetName:      lookupName(c, &models.CmdbAsset{}, operation.AssetId),
	}
}

func cloudOperationAudits(c *ctx.ServiceContext, operationId models.Id) ([]resps.CloudOperationAuditResp, e.Error) {
	audits := make([]models.CloudOperationAudit, 0)
	if err := c.DB().Model(&models.CloudOperationAudit{}).
		Where("org_id = ? and operation_id = ?", c.OrgId, operationId).
		Order("created_at asc").
		Scan(&audits); err != nil {
		return nil, e.New(e.DBError, err)
	}
	resp := make([]resps.CloudOperationAuditResp, 0, len(audits))
	for _, audit := range audits {
		resp = append(resp, resps.CloudOperationAuditResp{
			CloudOperationAudit: audit,
			OperatorName:        lookupName(c, &models.User{}, audit.OperatorId),
		})
	}
	return resp, nil
}

func lookupName(c *ctx.ServiceContext, model interface{}, id models.Id) string {
	if id == "" {
		return ""
	}
	name := ""
	_ = c.DB().Model(model).Where("id = ?", id).Select("name").Row().Scan(&name)
	return name
}

func getCloudAssetForOperation(c *ctx.ServiceContext, id models.Id) (*models.CmdbAsset, e.Error) {
	if _, err := BackfillCmdbAssetsFromIac(c); err != nil {
		return nil, err
	}
	if err := RefreshCmdbAssetGovernanceFields(c); err != nil {
		return nil, err
	}

	asset := models.CmdbAsset{}
	if err := buildCmdbAssetQuery(c).
		Where("iac_cmdb_asset.id = ?", id).
		First(&asset); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, err)
	}
	return &asset, nil
}

func cloudAssetActionSpecs() []cloudAssetActionSpec {
	localProviders := []string{"aws", "oci", "alicloud", "azure", "gcp", "tencentcloud", "huawei"}
	computeProviders := []string{"aws", "oci", "alicloud", "azure", "gcp"}
	providers := []string{"aws", "oci", "alicloud"}
	return []cloudAssetActionSpec{
		{
			Key:                models.CloudOperationActionRefreshMetadata,
			Name:               "刷新元数据",
			Description:        "记录一次资产元数据刷新任务，刷新本地纳管状态和最近操作 ID，不执行云端变更",
			RiskLevel:          models.CloudOperationRiskLow,
			AdapterKey:         cloudOperationAdapterLocalMetadata,
			AdapterMode:        cloudOperationAdapterModeLocal,
			Enabled:            true,
			SupportedProviders: localProviders,
		},
		{
			Key:                models.CloudOperationActionUpdateTags,
			Name:               "更新标签",
			Description:        "更新 CMDB 资产标签，当前版本不调用云厂商标签写入 API",
			RiskLevel:          models.CloudOperationRiskLow,
			AdapterKey:         cloudOperationAdapterLocalMetadata,
			AdapterMode:        cloudOperationAdapterModeLocal,
			Enabled:            true,
			SupportedProviders: localProviders,
		},
		{
			Key:                 models.CloudOperationActionStartInstance,
			Name:                "启动实例",
			Description:         "启动云主机实例",
			RiskLevel:           models.CloudOperationRiskMedium,
			AdapterKey:          cloudOperationAdapterProviderOperation,
			AdapterMode:         cloudOperationAdapterModeProvider,
			Enabled:             true,
			SupportedProviders:  computeProviders,
			SupportedAssetTypes: []string{models.CmdbAssetTypeComputeInstance},
		},
		{
			Key:                 models.CloudOperationActionStopInstance,
			Name:                "停止实例",
			Description:         "停止云主机实例",
			RiskLevel:           models.CloudOperationRiskMedium,
			AdapterKey:          cloudOperationAdapterProviderOperation,
			AdapterMode:         cloudOperationAdapterModeProvider,
			Enabled:             true,
			SupportedProviders:  computeProviders,
			SupportedAssetTypes: []string{models.CmdbAssetTypeComputeInstance},
		},
		{
			Key:                 models.CloudOperationActionRestartInstance,
			Name:                "重启实例",
			Description:         "重启云主机实例",
			RiskLevel:           models.CloudOperationRiskHigh,
			AdapterKey:          cloudOperationAdapterProviderOperation,
			AdapterMode:         cloudOperationAdapterModeProvider,
			Enabled:             true,
			RequiresApproval:    true,
			SupportedProviders:  computeProviders,
			SupportedAssetTypes: []string{models.CmdbAssetTypeComputeInstance},
		},
		{
			Key:                 models.CloudOperationActionResizeInstance,
			Name:                "调整实例规格",
			Description:         "调整云主机实例规格；已接入 provider live 查询、写执行器和写后规格确认，默认受云端写操作保护开关拦截",
			RiskLevel:           models.CloudOperationRiskHigh,
			AdapterKey:          cloudOperationAdapterProviderOperation,
			AdapterMode:         cloudOperationAdapterModeProvider,
			Enabled:             true,
			RequiresApproval:    true,
			SupportedProviders:  providers,
			SupportedAssetTypes: []string{models.CmdbAssetTypeComputeInstance},
		},
		{
			Key:                 models.CloudOperationActionResizeVolume,
			Name:                "磁盘扩容",
			Description:         "对块存储卷执行容量扩容；已接入 provider live 查询、写执行器和写后轮询，默认受云端写操作保护开关拦截",
			RiskLevel:           models.CloudOperationRiskHigh,
			AdapterKey:          cloudOperationAdapterProviderOperation,
			AdapterMode:         cloudOperationAdapterModeProvider,
			Enabled:             true,
			RequiresApproval:    true,
			SupportedProviders:  providers,
			SupportedAssetTypes: []string{models.CmdbAssetTypeBlockVolume},
		},
		{
			Key:                 models.CloudOperationActionCreateSnapshot,
			Name:                "创建快照/备份",
			Description:         "为块存储卷创建云厂商快照或备份；AWS/阿里云创建磁盘快照，OCI 创建 Block Volume Backup",
			RiskLevel:           models.CloudOperationRiskMedium,
			AdapterKey:          cloudOperationAdapterProviderOperation,
			AdapterMode:         cloudOperationAdapterModeProvider,
			Enabled:             true,
			SupportedProviders:  providers,
			SupportedAssetTypes: []string{models.CmdbAssetTypeBlockVolume},
		},
		{
			Key:                 models.CloudOperationActionUpdateSecurityRules,
			Name:                "更新安全组规则",
			Description:         "新增或撤销安全组/NSG 规则；AWS、阿里云支持按 CIDR/协议/端口新增撤销，OCI NSG 支持新增和按 ruleId 移除",
			RiskLevel:           models.CloudOperationRiskHigh,
			AdapterKey:          cloudOperationAdapterProviderOperation,
			AdapterMode:         cloudOperationAdapterModeProvider,
			Enabled:             true,
			RequiresApproval:    true,
			SupportedProviders:  providers,
			SupportedAssetTypes: []string{models.CmdbAssetTypeNetworkSecurityGroup},
		},
		{
			Key:                models.CloudOperationActionDeleteResource,
			Name:               "删除资源",
			Description:        "删除云资源",
			RiskLevel:          models.CloudOperationRiskCritical,
			AdapterKey:         cloudOperationAdapterProviderOperation,
			AdapterMode:        cloudOperationAdapterModeProvider,
			Enabled:            false,
			Destructive:        true,
			RequiresApproval:   true,
			DisabledReason:     "高危云端变更需要 provider operation adapter 和审批流接入后开放",
			SupportedProviders: providers,
		},
	}
}

func cloudAssetActionSpecByKey(key string) (cloudAssetActionSpec, bool) {
	key = strings.TrimSpace(key)
	for _, spec := range cloudAssetActionSpecs() {
		if spec.Key == key {
			return spec, true
		}
	}
	return cloudAssetActionSpec{}, false
}

func cloudAssetActionVisibleForAsset(asset *models.CmdbAsset, spec cloudAssetActionSpec) bool {
	return len(spec.SupportedAssetTypes) == 0 || stringInSlice(asset.AssetType, spec.SupportedAssetTypes)
}

func cloudAssetActionResp(c *ctx.ServiceContext, asset *models.CmdbAsset, spec cloudAssetActionSpec) resps.CloudAssetActionResp {
	enabled := spec.Enabled
	disabledReason := spec.DisabledReason
	capability := cloudAssetActionAdapterCapability(asset, spec)
	if len(spec.SupportedProviders) > 0 && !stringInSlice(normalizeProvider(asset.Provider), spec.SupportedProviders) {
		enabled = false
		disabledReason = fmt.Sprintf("暂不支持云厂商 %s", firstNonEmpty(asset.Provider, "unknown"))
	}
	if spec.Key == models.CloudOperationActionRefreshMetadata {
		if strings.TrimSpace(asset.Provider) == "" {
			enabled = false
			disabledReason = "缺少云厂商信息"
		} else if strings.TrimSpace(asset.NativeId) == "" {
			enabled = false
			disabledReason = "缺少云资源 ID"
		}
	}
	if spec.Key == models.CloudOperationActionUpdateTags && strings.TrimSpace(asset.Provider) == "" {
		enabled = false
		disabledReason = "缺少云厂商信息"
	}
	if enabled && capability.AdapterStatus != cloudOperationAdapterStatusReady {
		enabled = false
		disabledReason = capability.AdapterMessage
	}
	if enabled {
		permissionCheck := cloudAssetActionPermissionCheck(c, asset, spec)
		if permissionCheck.Status == "fail" {
			enabled = false
			disabledReason = permissionCheck.Message
		}
	}
	return resps.CloudAssetActionResp{
		Key:                 spec.Key,
		Name:                spec.Name,
		Description:         spec.Description,
		OperationType:       models.CloudOperationTypeAction,
		RiskLevel:           spec.RiskLevel,
		AdapterKey:          spec.AdapterKey,
		AdapterMode:         spec.AdapterMode,
		AdapterStatus:       capability.AdapterStatus,
		AdapterMessage:      capability.AdapterMessage,
		ProviderAdapter:     capability.ProviderAdapter,
		WriteEnabled:        capability.WriteEnabled,
		Enabled:             enabled,
		Destructive:         spec.Destructive,
		RequiresApproval:    spec.RequiresApproval,
		DisabledReason:      disabledReason,
		SupportedProviders:  spec.SupportedProviders,
		SupportedAssetTypes: spec.SupportedAssetTypes,
	}
}

func cloudAssetActionDryRun(c *ctx.ServiceContext, asset *models.CmdbAsset, spec cloudAssetActionSpec) *resps.CloudAssetActionDryRunResp {
	return cloudAssetActionDryRunWithParams(c, asset, spec, nil)
}

func cloudAssetActionDryRunWithParams(c *ctx.ServiceContext, asset *models.CmdbAsset, spec cloudAssetActionSpec, actionParams models.ResAttrs) *resps.CloudAssetActionDryRunResp {
	action := cloudAssetActionResp(c, asset, spec)
	checks := make([]resps.CloudAssetActionCheckResp, 0)
	appendCheck := func(key string, name string, status string, message string) {
		checks = append(checks, resps.CloudAssetActionCheckResp{
			Key:     key,
			Name:    name,
			Status:  status,
			Message: message,
		})
	}

	appendCheck("asset_exists", "资产存在", "pass", "资产属于当前组织")
	if strings.TrimSpace(asset.Provider) == "" {
		appendCheck("provider", "云厂商", "fail", "资产缺少云厂商信息")
	} else if len(spec.SupportedProviders) > 0 && !stringInSlice(normalizeProvider(asset.Provider), spec.SupportedProviders) {
		appendCheck("provider", "云厂商", "fail", fmt.Sprintf("暂不支持云厂商 %s", asset.Provider))
	} else {
		appendCheck("provider", "云厂商", "pass", fmt.Sprintf("云厂商 %s 可识别", asset.Provider))
	}
	if strings.TrimSpace(asset.NativeId) == "" {
		appendCheck("resource_id", "云资源 ID", "fail", "资产缺少云资源 ID")
	} else {
		appendCheck("resource_id", "云资源 ID", "pass", asset.NativeId)
	}
	if asset.Source == models.CmdbAssetSourceCloudCollect {
		appendCheck("source", "资产来源", "pass", "资产来自云采集")
	} else {
		appendCheck("source", "资产来源", "warn", "资产不是云采集来源，当前仅记录本地操作任务")
	}
	checks = append(checks, cloudAssetActionPermissionCheck(c, asset, spec))
	checks = append(checks, cloudAssetActionProviderPreflightChecks(c, asset, spec, actionParams)...)
	checks = append(checks, cloudAssetActionAdapterChecks(asset, spec)...)
	if spec.Key == models.CloudOperationActionUpdateTags {
		appendCheck("action_parameters", "操作参数", "warn", "创建任务时需要提交 tags JSON 对象；标签值为空字符串时表示删除该标签")
	}
	if spec.Key == models.CloudOperationActionResizeVolume {
		currentSize := cloudAssetVolumeSizeGiB(asset)
		message := "创建任务时需要提交 params.targetSizeGiB，目标容量必须大于当前容量"
		status := "warn"
		if targetSize, ok := cloudOperationIntParam(actionParams, "targetSizeGiB"); ok {
			if targetSize <= 0 {
				status = "fail"
				message = "params.targetSizeGiB 必须为正整数"
			} else if currentSize > 0 && targetSize <= currentSize {
				status = "fail"
				message = fmt.Sprintf("目标容量 %d GiB 必须大于当前容量 %d GiB", targetSize, currentSize)
			} else if currentSize > 0 {
				status = "pass"
				message = fmt.Sprintf("当前容量 %d GiB，目标容量 %d GiB，参数校验通过", currentSize, targetSize)
			} else {
				status = "pass"
				message = fmt.Sprintf("目标容量 %d GiB 已提交；当前容量未识别，创建任务时将继续校验", targetSize)
			}
		} else if currentSize > 0 {
			message = fmt.Sprintf("%s；当前识别容量约 %d GiB", message, currentSize)
		}
		appendCheck("action_parameters", "操作参数", status, message)
	}
	if spec.Key == models.CloudOperationActionResizeInstance {
		currentSpec := cloudAssetComputeSpec(asset)
		message := "创建任务时需要提交 params.targetInstanceType；AWS/阿里云对应实例规格，OCI 对应 shape"
		status := "warn"
		if targetSpec, ok := cloudProviderInstanceTargetSpecFromParams(actionParams); ok {
			if currentSpec != "" && targetSpec == currentSpec {
				status = "fail"
				message = fmt.Sprintf("目标规格 %s 与当前规格一致", targetSpec)
			} else if currentSpec != "" {
				status = "pass"
				message = fmt.Sprintf("当前规格 %s，目标规格 %s，参数校验通过", currentSpec, targetSpec)
			} else {
				status = "pass"
				message = fmt.Sprintf("目标规格 %s 已提交；当前规格未识别，创建任务时将继续校验", targetSpec)
			}
		} else if currentSpec != "" {
			message = fmt.Sprintf("%s；当前识别规格 %s", message, currentSpec)
		}
		appendCheck("action_parameters", "操作参数", status, message)
	}
	if spec.Key == models.CloudOperationActionCreateSnapshot {
		snapshotName := cloudProviderSnapshotNameFromParams(actionParams, asset.NativeId, "")
		message := "可选提交 params.snapshotName、params.description；OCI 可选 params.backupType=FULL/INCREMENTAL"
		if snapshotName != "" {
			message = fmt.Sprintf("快照/备份名称 %s，参数校验通过", snapshotName)
		}
		appendCheck("action_parameters", "操作参数", "pass", message)
	}
	if spec.Key == models.CloudOperationActionUpdateSecurityRules {
		status := "warn"
		message := "创建任务时需要提交 params.operation=authorize/revoke、params.direction=ingress/egress、params.protocol、params.cidr；TCP/UDP 需提交端口"
		if len(actionParams) == 0 {
			// Keep action directory previews non-blocking; creation still validates params strictly.
		} else if rule, errs := cloudSecurityRuleOperationFromParams(actionParams); len(errs) > 0 {
			status = "fail"
			message = strings.Join(errs, "；")
		} else if rule.Operation != "" {
			status = "pass"
			message = fmt.Sprintf("%s %s %s %s %s", rule.Operation, rule.Direction, rule.Protocol, cloudSecurityRulePortText(rule), rule.Cidr)
		}
		appendCheck("action_parameters", "操作参数", status, message)
	}
	if spec.AdapterKey == cloudOperationAdapterProviderOperation {
		appendCheck(
			"operator_confirmation",
			"二次确认",
			"warn",
			fmt.Sprintf("创建任务时需要填写操作原因，并输入动作 %s 与资源 ID %s 进行确认", spec.Key, firstNonEmpty(asset.NativeId, asset.Id.String())),
		)
	}
	if action.RequiresApproval {
		appendCheck("approval", "审批", "warn", "该动作未来开放时需要审批")
	} else {
		appendCheck("approval", "审批", "pass", "当前动作无需审批")
	}

	executable := action.Enabled
	for _, check := range checks {
		if check.Status == "fail" {
			executable = false
			break
		}
	}
	return &resps.CloudAssetActionDryRunResp{
		AssetId:          asset.Id,
		Action:           action.Key,
		Name:             action.Name,
		Executable:       executable,
		RiskLevel:        action.RiskLevel,
		AdapterKey:       action.AdapterKey,
		AdapterMode:      action.AdapterMode,
		AdapterStatus:    action.AdapterStatus,
		AdapterMessage:   action.AdapterMessage,
		ProviderAdapter:  action.ProviderAdapter,
		WriteEnabled:     action.WriteEnabled,
		Destructive:      action.Destructive,
		RequiresApproval: action.RequiresApproval,
		DisabledReason:   action.DisabledReason,
		Checks:           checks,
	}
}

func cloudAssetActionPermissionCheck(c *ctx.ServiceContext, asset *models.CmdbAsset, spec cloudAssetActionSpec) resps.CloudAssetActionCheckResp {
	if c.IsSuperAdmin || services.UserHasOrgRole(c.UserId, c.OrgId, consts.OrgRoleAdmin) {
		return resps.CloudAssetActionCheckResp{
			Key:     "user_permission",
			Name:    "用户权限",
			Status:  "pass",
			Message: "平台管理员或组织管理员可执行云资产操作",
		}
	}
	if asset.ProjectId == "" {
		return resps.CloudAssetActionCheckResp{
			Key:     "user_permission",
			Name:    "用户权限",
			Status:  "fail",
			Message: "未绑定项目的云资产仅平台管理员或组织管理员可操作",
		}
	}
	allowedRoles, allowedMessage, deniedMessage := cloudAssetActionProjectPermissionRule(spec)
	for _, role := range allowedRoles {
		if services.UserHasProjectRole(c.UserId, c.OrgId, asset.ProjectId, role) {
			return resps.CloudAssetActionCheckResp{
				Key:     "user_permission",
				Name:    "用户权限",
				Status:  "pass",
				Message: allowedMessage,
			}
		}
	}
	return resps.CloudAssetActionCheckResp{
		Key:     "user_permission",
		Name:    "用户权限",
		Status:  "fail",
		Message: deniedMessage,
	}
}

func cloudAssetActionProjectPermissionRule(spec cloudAssetActionSpec) ([]string, string, string) {
	switch spec.Key {
	case models.CloudOperationActionRefreshMetadata, models.CloudOperationActionUpdateTags:
		return []string{consts.ProjectRoleManager, consts.ProjectRoleOperator, consts.ProjectRoleApprover},
			"项目负责人、操作员或审批员可执行本地安全资产操作",
			"该本地安全资产操作需要项目负责人、操作员、审批员、组织管理员或平台管理员权限"
	case models.CloudOperationActionStartInstance, models.CloudOperationActionStopInstance:
		return []string{consts.ProjectRoleManager, consts.ProjectRoleOperator},
			"项目负责人或操作员可执行实例开停机",
			"实例开停机需要项目负责人、操作员、组织管理员或平台管理员权限"
	case models.CloudOperationActionRestartInstance:
		return []string{consts.ProjectRoleManager},
			"项目负责人可执行重启实例",
			"重启实例需要项目负责人、组织管理员或平台管理员权限"
	case models.CloudOperationActionResizeInstance:
		return []string{consts.ProjectRoleManager},
			"项目负责人可申请实例规格调整",
			"实例规格调整需要项目负责人、组织管理员或平台管理员权限"
	case models.CloudOperationActionResizeVolume:
		return []string{consts.ProjectRoleManager},
			"项目负责人可申请磁盘扩容",
			"磁盘扩容需要项目负责人、组织管理员或平台管理员权限"
	case models.CloudOperationActionCreateSnapshot:
		return []string{consts.ProjectRoleManager, consts.ProjectRoleOperator},
			"项目负责人或操作员可创建快照/备份",
			"创建快照/备份需要项目负责人、操作员、组织管理员或平台管理员权限"
	case models.CloudOperationActionUpdateSecurityRules:
		return []string{consts.ProjectRoleManager},
			"项目负责人可申请更新安全组规则",
			"更新安全组规则需要项目负责人、组织管理员或平台管理员权限"
	case models.CloudOperationActionDeleteResource:
		return nil,
			"",
			"删除资源需要组织管理员或平台管理员权限，并等待审批流接入后开放"
	default:
		return []string{consts.ProjectRoleManager},
			"项目负责人可执行该云资产操作",
			"该云资产操作需要项目负责人、组织管理员或平台管理员权限"
	}
}

func ensureCloudOperationMutationPermission(c *ctx.ServiceContext, operation *models.CloudOperation, verb string) e.Error {
	if c.IsSuperAdmin || services.UserHasOrgRole(c.UserId, c.OrgId, consts.OrgRoleAdmin) {
		return nil
	}
	if operation.OperationType != models.CloudOperationTypeAction {
		return e.New(e.BadParam, fmt.Errorf("%s操作任务 %s 需要组织管理员或平台管理员权限", verb, operation.Id))
	}
	if operation.AssetId == "" || operation.Action == "" {
		return e.New(e.BadParam, fmt.Errorf("操作任务 %s 缺少资产或动作信息，不能%s", operation.Id, verb))
	}
	spec, ok := cloudAssetActionSpecByKey(operation.Action)
	if !ok {
		return e.New(e.BadParam, fmt.Errorf("操作任务 %s 的动作 %s 不支持%s", operation.Id, operation.Action, verb))
	}
	asset, err := getCloudAssetForOperation(c, operation.AssetId)
	if err != nil {
		return err
	}
	permissionCheck := cloudAssetActionPermissionCheck(c, asset, spec)
	if permissionCheck.Status == "fail" {
		return e.New(e.BadParam, fmt.Errorf("%s操作任务 %s 失败：%s", verb, operation.Id, permissionCheck.Message))
	}
	return nil
}

func ensureCloudOperationApprovalPermission(c *ctx.ServiceContext, operation *models.CloudOperation) e.Error {
	if c.IsSuperAdmin || services.UserHasOrgRole(c.UserId, c.OrgId, consts.OrgRoleAdmin) {
		return nil
	}
	if operation.ProjectId == "" {
		return e.New(e.PermDenyApproval, fmt.Errorf("未绑定项目的云操作任务仅平台管理员或组织管理员可审批"))
	}
	if services.UserHasProjectRole(c.UserId, c.OrgId, operation.ProjectId, consts.ProjectRoleManager) ||
		services.UserHasProjectRole(c.UserId, c.OrgId, operation.ProjectId, consts.ProjectRoleApprover) {
		return nil
	}
	return e.New(e.PermDenyApproval, fmt.Errorf("审批云操作任务需要项目负责人、审批员、组织管理员或平台管理员权限"))
}

func cloudOperationApprovalAttrs(operation *models.CloudOperation, c *ctx.ServiceContext, form *forms.CloudOperationApprovalForm) models.ResAttrs {
	approvalId := operation.ApprovalId
	if approvalId == "" {
		approvalId = models.NewId("cpa")
	}
	status := "approved"
	if form.Action == "rejected" {
		status = "rejected"
	}
	return models.ResAttrs{
		"approvalId": approvalId.String(),
		"status":     status,
		"approverId": c.UserId.String(),
		"action":     form.Action,
		"comment":    strings.TrimSpace(form.Comment),
		"approvedAt": time.Now().Format(time.RFC3339),
	}
}

func markCloudOperationApproved(c *ctx.ServiceContext, operation *models.CloudOperation, approval models.ResAttrs) e.Error {
	params := operation.Params
	if params == nil {
		params = models.ResAttrs{}
	}
	params["approval"] = approval
	attrs := models.Attrs{
		"status":  models.CloudOperationStatusPending,
		"message": "云资产操作审批通过，等待执行",
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
	operation.Message = "云资产操作审批通过，等待执行"
	operation.Params = params
	return recordCloudOperationAudit(c, operation, models.CloudOperationStatusPending, "云资产操作审批通过", params, models.ResAttrs{
		"approval": approval,
	})
}

func cloudAssetActionProviderPreflightChecks(c *ctx.ServiceContext, asset *models.CmdbAsset, spec cloudAssetActionSpec, actionParams models.ResAttrs) []resps.CloudAssetActionCheckResp {
	if spec.AdapterKey != cloudOperationAdapterProviderOperation {
		return nil
	}

	checks := []resps.CloudAssetActionCheckResp{cloudAssetActionResourceScopeCheck(asset, spec)}
	if asset.CloudAccountId == "" {
		checks = append(checks, cloudProviderOperationAdapterPreflightChecks(asset, spec, nil, actionParams)...)
		return append(checks, resps.CloudAssetActionCheckResp{
			Key:     "cloud_account_binding",
			Name:    "云账号绑定",
			Status:  "fail",
			Message: "资产未绑定云账号，不能执行 provider 写操作预检查",
		})
	}

	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, asset.CloudAccountId)
	if err != nil {
		checks = append(checks, cloudProviderOperationAdapterPreflightChecks(asset, spec, nil, actionParams)...)
		return append(checks, resps.CloudAssetActionCheckResp{
			Key:     "cloud_account_binding",
			Name:    "云账号绑定",
			Status:  "fail",
			Message: fmt.Sprintf("云账号 %s 不存在或不属于当前组织", asset.CloudAccountId),
		})
	}
	checks = append(checks, resps.CloudAssetActionCheckResp{
		Key:     "cloud_account_binding",
		Name:    "云账号绑定",
		Status:  "pass",
		Message: fmt.Sprintf("已绑定云账号 %s", firstNonEmpty(account.Name, account.Id.String())),
	})

	checks = append(checks, cloudAssetActionCloudAccountMatchChecks(asset, account)...)
	checks = append(checks, cloudAssetActionCloudAccountAuthorizationChecks(asset, account, spec)...)
	validation := validateCloudAccount(account)
	for _, item := range cloudAccountPermissionItems(account, validation) {
		checks = append(checks, resps.CloudAssetActionCheckResp{
			Key:     "cloud_" + item.Key,
			Name:    item.Name,
			Status:  item.Status,
			Message: item.Message,
		})
	}
	checks = append(checks, cloudAssetActionRegionCoverageCheck(asset, account, validation))
	checks = append(checks, cloudProviderOperationAdapterPreflightChecks(asset, spec, cloudOperationCmdbCloudAccountFromModel(account), actionParams)...)
	return checks
}

func cloudAssetActionResourceScopeCheck(asset *models.CmdbAsset, spec cloudAssetActionSpec) resps.CloudAssetActionCheckResp {
	if len(spec.SupportedAssetTypes) > 0 && !stringInSlice(asset.AssetType, spec.SupportedAssetTypes) {
		return resps.CloudAssetActionCheckResp{
			Key:     "resource_scope",
			Name:    "资源定位",
			Status:  "fail",
			Message: fmt.Sprintf("动作 %s 不支持资产类型 %s", spec.Key, firstNonEmpty(asset.AssetType, "unknown")),
		}
	}
	if strings.TrimSpace(asset.NativeId) == "" {
		return resps.CloudAssetActionCheckResp{
			Key:     "resource_scope",
			Name:    "资源定位",
			Status:  "fail",
			Message: "缺少云资源 ID，无法定位 provider 侧资源",
		}
	}
	if strings.TrimSpace(asset.Region) == "" {
		return resps.CloudAssetActionCheckResp{
			Key:     "resource_scope",
			Name:    "资源定位",
			Status:  "fail",
			Message: "缺少云资源区域，无法定位 provider 侧资源",
		}
	}
	return resps.CloudAssetActionCheckResp{
		Key:     "resource_scope",
		Name:    "资源定位",
		Status:  "pass",
		Message: fmt.Sprintf("%s/%s/%s", asset.Provider, asset.Region, asset.NativeId),
	}
}

func cloudAssetActionCloudAccountMatchChecks(asset *models.CmdbAsset, account *models.CloudAccount) []resps.CloudAssetActionCheckResp {
	providerStatus := "pass"
	providerMessage := fmt.Sprintf("云账号 provider %s 与资产一致", account.Provider)
	if normalizeProvider(account.Provider) != normalizeProvider(asset.Provider) {
		providerStatus = "fail"
		providerMessage = fmt.Sprintf("云账号 provider %s 与资产 provider %s 不一致", firstNonEmpty(account.Provider, "unknown"), firstNonEmpty(asset.Provider, "unknown"))
	}

	accountStatus := "pass"
	accountMessage := fmt.Sprintf("云账号 accountId %s 与资产一致", firstNonEmpty(account.AccountId, asset.AccountId, "-"))
	if account.AccountId != "" && asset.AccountId != "" && account.AccountId != asset.AccountId {
		accountStatus = "fail"
		accountMessage = fmt.Sprintf("云账号 accountId %s 与资产 accountId %s 不一致", account.AccountId, asset.AccountId)
	} else if account.AccountId == "" || asset.AccountId == "" {
		accountStatus = "warn"
		accountMessage = "云账号或资产缺少 accountId，已跳过账号 ID 强校验"
	}

	return []resps.CloudAssetActionCheckResp{
		{
			Key:     "cloud_account_provider",
			Name:    "账号厂商",
			Status:  providerStatus,
			Message: providerMessage,
		},
		{
			Key:     "cloud_account_identity",
			Name:    "账号身份",
			Status:  accountStatus,
			Message: accountMessage,
		},
	}
}

func cloudAssetActionCloudAccountAuthorizationChecks(asset *models.CmdbAsset, account *models.CloudAccount, spec cloudAssetActionSpec) []resps.CloudAssetActionCheckResp {
	policy := cloudAccountOperationPolicy(account)
	return []resps.CloudAssetActionCheckResp{
		cloudAssetActionCloudAccountAuthorizationCheck(asset, policy, spec),
		cloudAssetActionTagAuthorizationCheck(asset, policy),
	}
}

func cloudAssetActionCloudAccountAuthorizationCheck(asset *models.CmdbAsset, policy models.ResAttrs, spec cloudAssetActionSpec) resps.CloudAssetActionCheckResp {
	projectIds := cloudPolicyStringList(policy, "projectIds", "project_ids", "projects")
	actions := cloudPolicyStringList(policy, "actions", "operationActions", "operation_actions")
	resourceTypes := cloudPolicyStringList(policy, "resourceTypes", "resource_types", "assetTypes", "asset_types")
	if len(projectIds) == 0 && len(actions) == 0 && len(resourceTypes) == 0 {
		return resps.CloudAssetActionCheckResp{
			Key:     "cloud_account_authorization",
			Name:    "云账号授权",
			Status:  "pass",
			Message: "云账号未配置项目、动作或资源类型授权限制",
		}
	}

	denied := make([]string, 0)
	if len(projectIds) > 0 && !cloudPolicyValueAllowed(asset.ProjectId.String(), projectIds) {
		denied = append(denied, fmt.Sprintf("项目 %s 不在授权范围 %s 内", firstNonEmpty(asset.ProjectId.String(), "-"), strings.Join(projectIds, ", ")))
	}
	if len(actions) > 0 && !cloudPolicyValueAllowed(spec.Key, actions) {
		denied = append(denied, fmt.Sprintf("动作 %s 不在授权范围 %s 内", spec.Key, strings.Join(actions, ", ")))
	}
	if len(resourceTypes) > 0 && !cloudPolicyValueAllowed(asset.AssetType, resourceTypes) {
		denied = append(denied, fmt.Sprintf("资源类型 %s 不在授权范围 %s 内", firstNonEmpty(asset.AssetType, "-"), strings.Join(resourceTypes, ", ")))
	}
	if len(denied) > 0 {
		return resps.CloudAssetActionCheckResp{
			Key:     "cloud_account_authorization",
			Name:    "云账号授权",
			Status:  "fail",
			Message: strings.Join(denied, "；"),
		}
	}
	return resps.CloudAssetActionCheckResp{
		Key:     "cloud_account_authorization",
		Name:    "云账号授权",
		Status:  "pass",
		Message: "云账号授权策略允许当前项目、动作和资源类型",
	}
}

func cloudAssetActionTagAuthorizationCheck(asset *models.CmdbAsset, policy models.ResAttrs) resps.CloudAssetActionCheckResp {
	tagPolicy := cloudAccountOperationTagPolicy(policy)
	if len(tagPolicy) == 0 {
		return resps.CloudAssetActionCheckResp{
			Key:     "resource_tag_authorization",
			Name:    "资源标签授权",
			Status:  "pass",
			Message: "云账号未配置资源标签授权限制",
		}
	}

	denied := make([]string, 0)
	matched := make([]string, 0, len(tagPolicy))
	for key, raw := range tagPolicy {
		tagKey := strings.TrimSpace(key)
		if tagKey == "" {
			continue
		}
		actual := attrString(asset.Tags, tagKey)
		if actual == "" {
			denied = append(denied, fmt.Sprintf("缺少标签 %s", tagKey))
			continue
		}
		allowedValues := cloudPolicyStringValues(raw)
		if len(allowedValues) > 0 && !cloudPolicyValueAllowed(actual, allowedValues) {
			denied = append(denied, fmt.Sprintf("标签 %s=%s 不在授权范围 %s 内", tagKey, actual, strings.Join(allowedValues, ", ")))
			continue
		}
		if len(allowedValues) == 0 {
			matched = append(matched, fmt.Sprintf("%s 存在", tagKey))
		} else {
			matched = append(matched, fmt.Sprintf("%s=%s", tagKey, actual))
		}
	}
	if len(denied) > 0 {
		return resps.CloudAssetActionCheckResp{
			Key:     "resource_tag_authorization",
			Name:    "资源标签授权",
			Status:  "fail",
			Message: strings.Join(denied, "；"),
		}
	}
	message := "资源标签满足授权策略"
	if len(matched) > 0 {
		message = fmt.Sprintf("资源标签满足授权策略：%s", strings.Join(matched, "，"))
	}
	return resps.CloudAssetActionCheckResp{
		Key:     "resource_tag_authorization",
		Name:    "资源标签授权",
		Status:  "pass",
		Message: message,
	}
}

func cloudAccountOperationPolicy(account *models.CloudAccount) models.ResAttrs {
	if account == nil || len(account.Metadata) == 0 {
		return nil
	}
	for _, key := range []string{"operationPolicy", "operation_policy", "operationAuthorization", "operation_authorization"} {
		if policy := cloudPolicyMap(account.Metadata[key]); len(policy) > 0 {
			return policy
		}
	}
	for _, key := range []string{"projectIds", "project_ids", "projects", "actions", "operationActions", "operation_actions", "resourceTypes", "resource_types", "assetTypes", "asset_types", "requiredTags", "required_tags", "tagScope", "tag_scope"} {
		if _, ok := account.Metadata[key]; ok {
			return account.Metadata
		}
	}
	return nil
}

func cloudAccountOperationTagPolicy(policy models.ResAttrs) models.ResAttrs {
	for _, key := range []string{"requiredTags", "required_tags", "tagScope", "tag_scope", "tags"} {
		if tags := cloudPolicyMap(policy[key]); len(tags) > 0 {
			return tags
		}
	}
	return nil
}

func cloudPolicyMap(value interface{}) models.ResAttrs {
	switch typed := value.(type) {
	case models.ResAttrs:
		return typed
	case map[string]interface{}:
		return models.ResAttrs(typed)
	case map[string]string:
		result := models.ResAttrs{}
		for key, value := range typed {
			result[key] = value
		}
		return result
	default:
		return nil
	}
}

func cloudPolicyStringList(policy models.ResAttrs, keys ...string) []string {
	if len(policy) == 0 {
		return nil
	}
	for _, key := range keys {
		if values := cloudPolicyStringValues(policy[key]); len(values) > 0 {
			return values
		}
	}
	return nil
}

func cloudPolicyStringValues(value interface{}) []string {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		parts := strings.FieldsFunc(typed, func(r rune) bool {
			return r == ',' || r == '，' || r == '\n' || r == '\t'
		})
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			if item := strings.TrimSpace(part); item != "" {
				values = append(values, item)
			}
		}
		return values
	case []string:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if item = strings.TrimSpace(item); item != "" {
				values = append(values, item)
			}
		}
		return values
	case models.StrSlice:
		return cloudPolicyStringValues([]string(typed))
	case []interface{}:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			values = append(values, cloudPolicyStringValues(item)...)
		}
		return values
	case models.ResAttrs:
		for _, key := range []string{"values", "value", "oneOf", "one_of", "in"} {
			if values := cloudPolicyStringValues(typed[key]); len(values) > 0 {
				return values
			}
		}
		return nil
	case map[string]interface{}:
		return cloudPolicyStringValues(models.ResAttrs(typed))
	default:
		value := firstNonEmpty(fmt.Sprintf("%v", typed))
		if value == "" {
			return nil
		}
		return []string{value}
	}
}

func cloudPolicyValueAllowed(value string, allowed []string) bool {
	value = strings.TrimSpace(value)
	for _, item := range allowed {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if item == "*" || strings.EqualFold(item, "all") || item == value {
			return true
		}
	}
	return false
}

func cloudAssetActionRegionCoverageCheck(asset *models.CmdbAsset, account *models.CloudAccount, validation cloudAccountValidation) resps.CloudAssetActionCheckResp {
	regions := firstNonEmptyStringSlice([]string(account.Regions), validation.Regions)
	if len(regions) == 0 {
		return resps.CloudAssetActionCheckResp{
			Key:     "cloud_account_region_coverage",
			Name:    "区域覆盖",
			Status:  "fail",
			Message: "云账号未配置或无法推断可用区域",
		}
	}
	if strings.TrimSpace(asset.Region) == "" {
		return resps.CloudAssetActionCheckResp{
			Key:     "cloud_account_region_coverage",
			Name:    "区域覆盖",
			Status:  "fail",
			Message: "资产缺少区域信息，无法匹配云账号可用区域",
		}
	}
	if !stringInSlice(asset.Region, regions) {
		return resps.CloudAssetActionCheckResp{
			Key:     "cloud_account_region_coverage",
			Name:    "区域覆盖",
			Status:  "fail",
			Message: fmt.Sprintf("资产区域 %s 不在云账号可用区域 %s 内", asset.Region, strings.Join(regions, ", ")),
		}
	}
	return resps.CloudAssetActionCheckResp{
		Key:     "cloud_account_region_coverage",
		Name:    "区域覆盖",
		Status:  "pass",
		Message: fmt.Sprintf("资产区域 %s 已在云账号可用区域内", asset.Region),
	}
}

func cloudProviderOperationAdapterPreflightChecks(asset *models.CmdbAsset, spec cloudAssetActionSpec, account *cmdbCloudAccount, actionParams models.ResAttrs) []resps.CloudAssetActionCheckResp {
	adapter, ok := cloudProviderOperationAdapterForAction(asset.Provider, spec.Key)
	if !ok {
		return nil
	}
	request := adapter.BuildRequest(asset, spec, nil, nil)
	if len(actionParams) > 0 {
		request.Params = actionParams
	}
	state := adapter.ReadResourceState(request, account)
	checks := []resps.CloudAssetActionCheckResp{cloudProviderResourceStateCheck(state)}
	checks = append(checks, adapter.ValidateTransition(request, state)...)
	return checks
}

func cloudProviderResourceStateCheck(state cloudProviderResourceState) resps.CloudAssetActionCheckResp {
	readMode := firstNonEmpty(state.ReadMode, cloudOperationProviderReadModeCache)
	source := firstNonEmpty(state.Source, "未知来源")
	if state.ErrorCode != "" {
		message := fmt.Sprintf("%s(%s) 状态读取未完成：%s", source, readMode, firstNonEmpty(state.ErrorMessage, state.ErrorCode))
		if strings.TrimSpace(state.RawState) != "" {
			message = fmt.Sprintf("%s；已回退到状态 %s 并归一化为 %s", message, state.RawState, state.NormalizedState)
		}
		return resps.CloudAssetActionCheckResp{
			Key:     "provider_state_read",
			Name:    "状态只读确认",
			Status:  "warn",
			Message: message,
		}
	}
	if strings.TrimSpace(state.RawState) == "" {
		return resps.CloudAssetActionCheckResp{
			Key:     "provider_state_read",
			Name:    "状态只读确认",
			Status:  "warn",
			Message: fmt.Sprintf("未从 %s(%s) 读取到实例状态，真实执行前需要重新确认 provider 状态", source, readMode),
		}
	}
	return resps.CloudAssetActionCheckResp{
		Key:     "provider_state_read",
		Name:    "状态只读确认",
		Status:  "pass",
		Message: fmt.Sprintf("%s(%s) 状态 %s 已归一化为 %s", source, readMode, state.RawState, state.NormalizedState),
	}
}

func cloudAssetActionAdapterCapability(asset *models.CmdbAsset, spec cloudAssetActionSpec) cloudOperationAdapterCapability {
	if spec.AdapterKey == cloudOperationAdapterLocalMetadata {
		return cloudOperationAdapterCapability{
			AdapterKey:      spec.AdapterKey,
			AdapterMode:     spec.AdapterMode,
			AdapterStatus:   cloudOperationAdapterStatusReady,
			AdapterMessage:  "已接入本地安全适配器 local_metadata，不调用云端 API",
			ProviderAdapter: spec.AdapterKey,
			WriteEnabled:    true,
		}
	}
	if spec.AdapterKey != cloudOperationAdapterProviderOperation {
		return cloudOperationAdapterCapability{
			AdapterKey:      spec.AdapterKey,
			AdapterMode:     spec.AdapterMode,
			AdapterStatus:   cloudOperationAdapterStatusUnsupported,
			AdapterMessage:  firstNonEmpty(spec.DisabledReason, "动作当前未开放"),
			ProviderAdapter: spec.AdapterKey,
			WriteEnabled:    false,
		}
	}
	if !spec.Enabled && strings.TrimSpace(spec.DisabledReason) != "" {
		return cloudOperationAdapterCapability{
			AdapterKey:      spec.AdapterKey,
			AdapterMode:     spec.AdapterMode,
			AdapterStatus:   cloudOperationAdapterStatusBlocked,
			AdapterMessage:  spec.DisabledReason,
			ProviderAdapter: "",
			WriteEnabled:    false,
		}
	}

	provider := normalizeProvider(asset.Provider)
	providerAdapter, registered := cloudProviderOperationAdapterName(provider, spec.Key)
	if !registered {
		return cloudOperationAdapterCapability{
			AdapterKey:      spec.AdapterKey,
			AdapterMode:     spec.AdapterMode,
			AdapterStatus:   cloudOperationAdapterStatusUnsupported,
			AdapterMessage:  fmt.Sprintf("%s/%s 暂未注册 provider operation adapter", firstNonEmpty(provider, "unknown"), spec.Key),
			ProviderAdapter: "",
			WriteEnabled:    false,
		}
	}
	if !spec.Enabled {
		return cloudOperationAdapterCapability{
			AdapterKey:      spec.AdapterKey,
			AdapterMode:     spec.AdapterMode,
			AdapterStatus:   cloudOperationAdapterStatusBlocked,
			AdapterMessage:  firstNonEmpty(spec.DisabledReason, "动作当前未开放"),
			ProviderAdapter: providerAdapter,
			WriteEnabled:    false,
		}
	}
	if !cloudOperationProviderWriteEnabled() {
		return cloudOperationAdapterCapability{
			AdapterKey:      spec.AdapterKey,
			AdapterMode:     spec.AdapterMode,
			AdapterStatus:   cloudOperationAdapterStatusRegistered,
			AdapterMessage:  fmt.Sprintf("provider adapter %s 已注册；云端写操作保护开关 %s 未开启", providerAdapter, cloudOperationProviderWriteEnv),
			ProviderAdapter: providerAdapter,
			WriteEnabled:    false,
		}
	}
	if cloudOperationProviderReadMode() != cloudOperationProviderReadModeLive {
		return cloudOperationAdapterCapability{
			AdapterKey:      spec.AdapterKey,
			AdapterMode:     spec.AdapterMode,
			AdapterStatus:   cloudOperationAdapterStatusBlocked,
			AdapterMessage:  fmt.Sprintf("provider adapter %s 已接入写操作执行器；真实执行要求 %s=live", providerAdapter, cloudOperationProviderReadModeEnv),
			ProviderAdapter: providerAdapter,
			WriteEnabled:    false,
		}
	}
	return cloudOperationAdapterCapability{
		AdapterKey:      spec.AdapterKey,
		AdapterMode:     spec.AdapterMode,
		AdapterStatus:   cloudOperationAdapterStatusReady,
		AdapterMessage:  fmt.Sprintf("provider adapter %s 已接入写操作执行器，写操作保护已显式开启", providerAdapter),
		ProviderAdapter: providerAdapter,
		WriteEnabled:    true,
	}
}

func cloudAssetActionAdapterChecks(asset *models.CmdbAsset, spec cloudAssetActionSpec) []resps.CloudAssetActionCheckResp {
	capability := cloudAssetActionAdapterCapability(asset, spec)
	if spec.AdapterKey == cloudOperationAdapterLocalMetadata {
		return []resps.CloudAssetActionCheckResp{{
			Key:     "adapter",
			Name:    "操作适配器",
			Status:  "pass",
			Message: capability.AdapterMessage,
		}}
	}

	registryStatus := "pass"
	registryMessage := capability.AdapterMessage
	if capability.AdapterStatus == cloudOperationAdapterStatusUnsupported {
		registryStatus = "fail"
	} else if capability.ProviderAdapter != "" {
		registryMessage = fmt.Sprintf("已注册 provider adapter %s", capability.ProviderAdapter)
	}
	checks := []resps.CloudAssetActionCheckResp{{
		Key:     "adapter_registry",
		Name:    "适配器注册",
		Status:  registryStatus,
		Message: registryMessage,
	}}
	if registryStatus == "fail" {
		return checks
	}

	executionStatus := "pass"
	executionMessage := "云端写操作执行器已开放"
	if !capability.WriteEnabled || capability.AdapterStatus != cloudOperationAdapterStatusReady {
		executionStatus = "fail"
		executionMessage = capability.AdapterMessage
	}
	checks = append(checks, resps.CloudAssetActionCheckResp{
		Key:     "execution_guard",
		Name:    "执行保护",
		Status:  executionStatus,
		Message: executionMessage,
	})
	return checks
}

func cloudProviderOperationAdapterName(provider string, action string) (string, bool) {
	registry := map[string]map[string]string{
		"aws": {
			models.CloudOperationActionStartInstance:       "aws_ec2_instance",
			models.CloudOperationActionStopInstance:        "aws_ec2_instance",
			models.CloudOperationActionRestartInstance:     "aws_ec2_instance",
			models.CloudOperationActionResizeInstance:      "aws_ec2_instance",
			models.CloudOperationActionResizeVolume:        "aws_ebs_volume",
			models.CloudOperationActionCreateSnapshot:      "aws_ebs_snapshot",
			models.CloudOperationActionUpdateSecurityRules: "aws_security_group",
		},
		"oci": {
			models.CloudOperationActionStartInstance:       "oci_compute_instance",
			models.CloudOperationActionStopInstance:        "oci_compute_instance",
			models.CloudOperationActionRestartInstance:     "oci_compute_instance",
			models.CloudOperationActionResizeInstance:      "oci_compute_instance",
			models.CloudOperationActionResizeVolume:        "oci_core_volume",
			models.CloudOperationActionCreateSnapshot:      "oci_block_volume_backup",
			models.CloudOperationActionUpdateSecurityRules: "oci_network_security_group_rules",
		},
		"alicloud": {
			models.CloudOperationActionStartInstance:       "alicloud_ecs_instance",
			models.CloudOperationActionStopInstance:        "alicloud_ecs_instance",
			models.CloudOperationActionRestartInstance:     "alicloud_ecs_instance",
			models.CloudOperationActionResizeInstance:      "alicloud_ecs_instance",
			models.CloudOperationActionResizeVolume:        "alicloud_disk",
			models.CloudOperationActionCreateSnapshot:      "alicloud_disk_snapshot",
			models.CloudOperationActionUpdateSecurityRules: "alicloud_security_group",
		},
		"azure": {
			models.CloudOperationActionStartInstance:   "azure_virtual_machine",
			models.CloudOperationActionStopInstance:    "azure_virtual_machine",
			models.CloudOperationActionRestartInstance: "azure_virtual_machine",
		},
		"gcp": {
			models.CloudOperationActionStartInstance:   "gcp_compute_instance",
			models.CloudOperationActionStopInstance:    "gcp_compute_instance",
			models.CloudOperationActionRestartInstance: "gcp_compute_instance",
		},
	}
	actions, ok := registry[normalizeProvider(provider)]
	if !ok {
		return "", false
	}
	adapter, ok := actions[action]
	return adapter, ok
}

func cloudProviderOperationAdapterForAction(provider string, action string) (cloudProviderOperationAdapter, bool) {
	adapterName, ok := cloudProviderOperationAdapterName(provider, action)
	if !ok {
		return nil, false
	}
	if action == models.CloudOperationActionResizeVolume || action == models.CloudOperationActionCreateSnapshot {
		return cloudBlockVolumeOperationAdapter{
			provider: normalizeProvider(provider),
			key:      adapterName,
		}, true
	}
	if action == models.CloudOperationActionUpdateSecurityRules {
		return cloudSecurityGroupOperationAdapter{
			provider: normalizeProvider(provider),
			key:      adapterName,
		}, true
	}
	return cloudComputeInstanceOperationAdapter{
		provider: normalizeProvider(provider),
		key:      adapterName,
	}, true
}

func (a cloudComputeInstanceOperationAdapter) Key() string {
	return a.key
}

func (a cloudComputeInstanceOperationAdapter) Provider() string {
	return a.provider
}

func (a cloudBlockVolumeOperationAdapter) Key() string {
	return a.key
}

func (a cloudBlockVolumeOperationAdapter) Provider() string {
	return a.provider
}

func (a cloudSecurityGroupOperationAdapter) Key() string {
	return a.key
}

func (a cloudSecurityGroupOperationAdapter) Provider() string {
	return a.provider
}

func (a cloudComputeInstanceOperationAdapter) BuildRequest(asset *models.CmdbAsset, spec cloudAssetActionSpec, operation *models.CloudOperation, dryRun *resps.CloudAssetActionDryRunResp) cloudProviderOperationRequest {
	operationId := models.Id("")
	params := models.ResAttrs{}
	if operation != nil {
		operationId = operation.Id
		params = modelResAttrs(operation.Params["params"])
	}
	idempotencyKey := fmt.Sprintf("%s:%s:%s:%s", normalizeProvider(asset.Provider), spec.Key, asset.NativeId, firstNonEmpty(operationId.String(), "dry-run"))
	providerAdapter := a.Key()
	if dryRun != nil && strings.TrimSpace(dryRun.ProviderAdapter) != "" {
		providerAdapter = dryRun.ProviderAdapter
	}
	targetState := cloudProviderActionTargetState(spec.Key)
	if spec.Key == models.CloudOperationActionResizeInstance {
		if targetSpec, ok := cloudProviderInstanceTargetSpecFromParams(params); ok {
			targetState = targetSpec
		}
	}
	return cloudProviderOperationRequest{
		OperationId:     operationId,
		IdempotencyKey:  idempotencyKey,
		Action:          spec.Key,
		Provider:        normalizeProvider(asset.Provider),
		ProviderAdapter: providerAdapter,
		ResourceType:    asset.AssetType,
		ResourceId:      asset.NativeId,
		ResourceName:    firstNonEmpty(asset.Name, asset.NativeId, asset.Id.String()),
		Region:          asset.Region,
		Zone:            asset.Zone,
		AccountId:       asset.AccountId,
		CloudAccountId:  asset.CloudAccountId,
		ObservedState:   cloudProviderObservedState(asset),
		TargetState:     targetState,
		ReadMode:        cloudOperationProviderReadMode(),
		Params:          params,
	}
}

func (a cloudBlockVolumeOperationAdapter) BuildRequest(asset *models.CmdbAsset, spec cloudAssetActionSpec, operation *models.CloudOperation, dryRun *resps.CloudAssetActionDryRunResp) cloudProviderOperationRequest {
	operationId := models.Id("")
	params := models.ResAttrs{}
	if operation != nil {
		operationId = operation.Id
		params = modelResAttrs(operation.Params["params"])
	}
	idempotencyKey := fmt.Sprintf("%s:%s:%s:%s", normalizeProvider(asset.Provider), spec.Key, asset.NativeId, firstNonEmpty(operationId.String(), "dry-run"))
	providerAdapter := a.Key()
	if dryRun != nil && strings.TrimSpace(dryRun.ProviderAdapter) != "" {
		providerAdapter = dryRun.ProviderAdapter
	}
	targetState := cloudProviderActionTargetState(spec.Key)
	if spec.Key == models.CloudOperationActionCreateSnapshot {
		if snapshotName := cloudProviderSnapshotNameFromParams(params, asset.NativeId, operationId.String()); snapshotName != "" {
			targetState = snapshotName
		}
	}
	return cloudProviderOperationRequest{
		OperationId:     operationId,
		IdempotencyKey:  idempotencyKey,
		Action:          spec.Key,
		Provider:        normalizeProvider(asset.Provider),
		ProviderAdapter: providerAdapter,
		ResourceType:    asset.AssetType,
		ResourceId:      asset.NativeId,
		ResourceName:    firstNonEmpty(asset.Name, asset.NativeId, asset.Id.String()),
		Region:          asset.Region,
		Zone:            asset.Zone,
		AccountId:       asset.AccountId,
		CloudAccountId:  asset.CloudAccountId,
		ObservedState:   firstNonEmpty(cloudAssetVolumeObservedState(asset), cloudProviderObservedState(asset)),
		TargetState:     targetState,
		ReadMode:        cloudOperationProviderReadMode(),
		Params:          params,
	}
}

func (a cloudSecurityGroupOperationAdapter) BuildRequest(asset *models.CmdbAsset, spec cloudAssetActionSpec, operation *models.CloudOperation, dryRun *resps.CloudAssetActionDryRunResp) cloudProviderOperationRequest {
	operationId := models.Id("")
	params := models.ResAttrs{}
	if operation != nil {
		operationId = operation.Id
		params = modelResAttrs(operation.Params["params"])
	}
	idempotencyKey := fmt.Sprintf("%s:%s:%s:%s", normalizeProvider(asset.Provider), spec.Key, asset.NativeId, firstNonEmpty(operationId.String(), "dry-run"))
	providerAdapter := a.Key()
	if dryRun != nil && strings.TrimSpace(dryRun.ProviderAdapter) != "" {
		providerAdapter = dryRun.ProviderAdapter
	}
	return cloudProviderOperationRequest{
		OperationId:     operationId,
		IdempotencyKey:  idempotencyKey,
		Action:          spec.Key,
		Provider:        normalizeProvider(asset.Provider),
		ProviderAdapter: providerAdapter,
		ResourceType:    asset.AssetType,
		ResourceId:      asset.NativeId,
		ResourceName:    firstNonEmpty(asset.Name, asset.NativeId, asset.Id.String()),
		Region:          asset.Region,
		Zone:            asset.Zone,
		AccountId:       asset.AccountId,
		CloudAccountId:  asset.CloudAccountId,
		ObservedState:   cloudSecurityGroupObservedState(asset),
		TargetState:     cloudProviderActionTargetState(spec.Key),
		ReadMode:        cloudOperationProviderReadMode(),
		Params:          params,
	}
}

func (a cloudComputeInstanceOperationAdapter) ReadResourceState(request cloudProviderOperationRequest, account *cmdbCloudAccount) cloudProviderResourceState {
	if request.ReadMode == cloudOperationProviderReadModeLive {
		return cloudProviderLiveComputeState(request, account)
	}
	return cloudProviderCachedComputeState(request, nil)
}

func (a cloudBlockVolumeOperationAdapter) ReadResourceState(request cloudProviderOperationRequest, account *cmdbCloudAccount) cloudProviderResourceState {
	if request.ReadMode == cloudOperationProviderReadModeLive {
		return cloudProviderLiveVolumeState(request, account)
	}
	return cloudProviderCachedVolumeState(request)
}

func (a cloudSecurityGroupOperationAdapter) ReadResourceState(request cloudProviderOperationRequest, account *cmdbCloudAccount) cloudProviderResourceState {
	if request.ReadMode == cloudOperationProviderReadModeLive {
		return cloudProviderLiveSecurityGroupState(request, account)
	}
	return cloudProviderCachedSecurityGroupState(request)
}

func (a cloudComputeInstanceOperationAdapter) ValidateTransition(request cloudProviderOperationRequest, state cloudProviderResourceState) []resps.CloudAssetActionCheckResp {
	status := "pass"
	message := fmt.Sprintf("当前状态 %s 可执行 %s，目标状态 %s", firstNonEmpty(state.RawState, "unknown"), request.Action, request.TargetState)
	normalized := state.NormalizedState
	if normalized == "" || normalized == "unknown" {
		status = "warn"
		message = "当前状态未知，真实执行前需要重新确认实例状态"
	} else if cloudProviderComputeStateTerminal(normalized) {
		status = "fail"
		message = fmt.Sprintf("当前状态 %s 不允许执行实例启停/重启动作", state.RawState)
	} else if cloudProviderComputeStateTransitional(normalized) {
		status = "fail"
		message = fmt.Sprintf("当前状态 %s 正在变更中，不能重复提交 provider 写操作", state.RawState)
	} else {
		switch request.Action {
		case models.CloudOperationActionStartInstance:
			if normalized == "running" {
				status = "warn"
				message = "实例已处于运行中，启动动作将作为幂等空操作处理"
			} else if normalized != "stopped" {
				status = "warn"
				message = fmt.Sprintf("当前状态 %s 未在启动动作白名单内，真实执行前需要重新确认", state.RawState)
			}
		case models.CloudOperationActionStopInstance:
			if normalized == "stopped" {
				status = "warn"
				message = "实例已处于停止状态，停止动作将作为幂等空操作处理"
			} else if normalized != "running" {
				status = "warn"
				message = fmt.Sprintf("当前状态 %s 未在停止动作白名单内，真实执行前需要重新确认", state.RawState)
			}
		case models.CloudOperationActionRestartInstance:
			if normalized != "running" {
				status = "fail"
				message = fmt.Sprintf("重启动作要求实例处于运行中，当前状态为 %s", firstNonEmpty(state.RawState, "unknown"))
			}
		case models.CloudOperationActionResizeInstance:
			targetSpec, ok := cloudProviderInstanceTargetSpec(request)
			currentSpec := cloudProviderComputeCurrentSpec(state)
			if !ok {
				status = "warn"
				message = "实例规格调整需要在创建任务时提交 params.targetInstanceType"
			} else if currentSpec != "" && targetSpec == currentSpec {
				status = "warn"
				message = fmt.Sprintf("实例已是目标规格 %s，调整动作将作为幂等空操作处理", targetSpec)
			} else if normalized != "stopped" {
				status = "fail"
				message = fmt.Sprintf("实例规格调整要求实例处于停止状态，当前状态为 %s", firstNonEmpty(state.RawState, "unknown"))
			} else if currentSpec != "" {
				message = fmt.Sprintf("当前规格 %s，目标规格 %s，可进入规格调整审批/执行流程", currentSpec, targetSpec)
			} else {
				message = fmt.Sprintf("目标规格 %s 已提交；当前规格未识别，真实执行前会重新确认", targetSpec)
			}
		default:
			status = "fail"
			message = fmt.Sprintf("provider adapter %s 不支持动作 %s", a.Key(), request.Action)
		}
	}
	return []resps.CloudAssetActionCheckResp{{
		Key:     "provider_state_transition",
		Name:    "状态变更校验",
		Status:  status,
		Message: message,
	}}
}

func (a cloudBlockVolumeOperationAdapter) ValidateTransition(request cloudProviderOperationRequest, state cloudProviderResourceState) []resps.CloudAssetActionCheckResp {
	if request.Action == models.CloudOperationActionCreateSnapshot {
		status := "pass"
		message := "块存储卷可创建快照/备份"
		if strings.EqualFold(state.RawState, "deleted") || strings.EqualFold(state.NormalizedState, "deleted") {
			status = "fail"
			message = "块存储卷已删除，不能创建快照/备份"
		} else if snapshotName := cloudProviderSnapshotName(request); snapshotName != "" {
			message = fmt.Sprintf("将创建快照/备份 %s", snapshotName)
		}
		return []resps.CloudAssetActionCheckResp{{
			Key:     "provider_snapshot_create",
			Name:    "快照/备份校验",
			Status:  status,
			Message: message,
		}}
	}
	status := "warn"
	message := "磁盘扩容需要在创建任务时提交 params.targetSizeGiB"
	targetSize, ok := cloudOperationIntParam(request.Params, "targetSizeGiB")
	currentSize := cloudProviderVolumeCurrentSizeGiB(request, state)
	if ok && targetSize > 0 {
		status = "pass"
		message = fmt.Sprintf("目标容量 %d GiB 已通过参数校验", targetSize)
		if currentSize > 0 {
			if targetSize <= currentSize {
				status = "fail"
				message = fmt.Sprintf("目标容量 %d GiB 必须大于当前容量 %d GiB", targetSize, currentSize)
			} else {
				message = fmt.Sprintf("当前容量 %d GiB，目标容量 %d GiB，可进入扩容审批/执行流程", currentSize, targetSize)
			}
		}
	}
	return []resps.CloudAssetActionCheckResp{{
		Key:     "provider_volume_resize",
		Name:    "磁盘扩容校验",
		Status:  status,
		Message: message,
	}}
}

func (a cloudSecurityGroupOperationAdapter) ValidateTransition(request cloudProviderOperationRequest, state cloudProviderResourceState) []resps.CloudAssetActionCheckResp {
	status := "warn"
	message := "更新安全组规则需要在创建任务时提交完整规则参数"
	if len(request.Params) == 0 {
		// Action directory preview only. Create path validates params before execution.
	} else if rule, errs := cloudSecurityRuleOperationFromParams(request.Params); len(errs) > 0 {
		status = "fail"
		message = strings.Join(errs, "；")
	} else if rule.Operation != "" {
		status = "pass"
		message = fmt.Sprintf("将执行 %s %s %s %s %s", rule.Operation, rule.Direction, rule.Protocol, cloudSecurityRulePortText(rule), rule.Cidr)
	}
	if state.ErrorCode != "" {
		message = fmt.Sprintf("%s；provider 状态读取未完成：%s", message, firstNonEmpty(state.ErrorMessage, state.ErrorCode))
	}
	return []resps.CloudAssetActionCheckResp{{
		Key:     "provider_security_rule_update",
		Name:    "安全组规则校验",
		Status:  status,
		Message: message,
	}}
}

func cloudProviderObservedState(asset *models.CmdbAsset) string {
	return firstNonEmpty(
		asset.Status,
		attrString(asset.Attributes, "status"),
		attrString(asset.Attributes, "state"),
		attrString(asset.Attributes, "lifecycle_state"),
		attrString(asset.RawData, "status"),
		attrString(asset.RawData, "state"),
		attrString(asset.RawData, "lifecycle_state"),
	)
}

func cloudProviderActionTargetState(action string) string {
	switch action {
	case models.CloudOperationActionStartInstance:
		return "running"
	case models.CloudOperationActionStopInstance:
		return "stopped"
	case models.CloudOperationActionRestartInstance:
		return "running"
	case models.CloudOperationActionResizeInstance:
		return "resize_requested"
	case models.CloudOperationActionResizeVolume:
		return "resize_requested"
	case models.CloudOperationActionCreateSnapshot:
		return "snapshot_requested"
	case models.CloudOperationActionUpdateSecurityRules:
		return "security_rule_update_requested"
	default:
		return ""
	}
}

func cloudProviderCachedVolumeState(request cloudProviderOperationRequest) cloudProviderResourceState {
	sizeGiB := cloudVolumeSizeGiBFromString(request.ObservedState)
	state := cloudVolumeStateValue(sizeGiB)
	rawResponse := models.ResAttrs{
		"source":         "cmdb_cache",
		"observedState":  request.ObservedState,
		"currentSizeGiB": sizeGiB,
	}
	return cloudProviderResourceState{
		RawState:        firstNonEmpty(request.ObservedState, state),
		NormalizedState: state,
		Source:          "CMDB缓存",
		ReadMode:        firstNonEmpty(request.ReadMode, cloudOperationProviderReadModeCache),
		RawResponse:     cloudProviderSanitizeAttrs(rawResponse),
	}
}

func cloudProviderCachedSecurityGroupState(request cloudProviderOperationRequest) cloudProviderResourceState {
	rawResponse := models.ResAttrs{
		"source":        "cmdb_cache",
		"observedState": request.ObservedState,
	}
	return cloudProviderResourceState{
		RawState:        firstNonEmpty(request.ObservedState, "cached"),
		NormalizedState: "ready",
		Source:          "CMDB缓存",
		ReadMode:        firstNonEmpty(request.ReadMode, cloudOperationProviderReadModeCache),
		RawResponse:     cloudProviderSanitizeAttrs(rawResponse),
	}
}

func cloudSecurityGroupObservedState(asset *models.CmdbAsset) string {
	if asset == nil {
		return ""
	}
	rules := cloudAssetSecurityRulesFromAsset(asset)
	return fmt.Sprintf("%d rules", len(rules))
}

func cloudAssetComputeSpec(asset *models.CmdbAsset) string {
	if asset == nil {
		return ""
	}
	for _, attrs := range []models.ResAttrs{asset.Attributes, asset.RawData} {
		if spec := cloudComputeSpecFromAttrs(attrs); spec != "" {
			return spec
		}
	}
	return ""
}

func cloudProviderInstanceTargetSpec(request cloudProviderOperationRequest) (string, bool) {
	return cloudProviderInstanceTargetSpecFromParams(request.Params)
}

func cloudProviderInstanceTargetSpecFromParams(params models.ResAttrs) (string, bool) {
	for _, key := range []string{"targetInstanceType", "target_instance_type", "instanceType", "instance_type", "targetShape", "target_shape", "shape", "targetSpec", "target_spec"} {
		value := strings.TrimSpace(fmt.Sprintf("%v", params[key]))
		if value != "" && value != "<nil>" {
			return value, true
		}
	}
	return "", false
}

func cloudProviderComputeCurrentSpec(state cloudProviderResourceState) string {
	for _, attrs := range []models.ResAttrs{
		state.RawResponse,
		modelResAttrs(state.RawResponse["response"]),
		modelResAttrs(state.RawResponse["request"]),
	} {
		if spec := cloudComputeSpecFromAttrs(attrs); spec != "" {
			return spec
		}
	}
	return ""
}

func cloudSecurityRuleOperationFromParams(params models.ResAttrs) (cloudSecurityRuleOperation, []string) {
	rule := cloudSecurityRuleOperation{}
	if len(params) == 0 {
		return rule, []string{"缺少安全组规则参数"}
	}
	errs := make([]string, 0)
	rule.Operation = cloudSecurityRuleOperationValue(params)
	if rule.Operation == "" {
		errs = append(errs, "params.operation 必须为 authorize/revoke")
	}
	rule.Direction = strings.ToLower(firstNonEmpty(
		attrString(params, "direction"),
		attrString(params, "ruleDirection"),
		attrString(params, "rule_direction"),
	))
	if rule.Direction != "ingress" && rule.Direction != "egress" {
		errs = append(errs, "params.direction 必须为 ingress/egress")
	}
	rule.Protocol = strings.ToLower(firstNonEmpty(
		attrString(params, "protocol"),
		attrString(params, "ipProtocol"),
		attrString(params, "ip_protocol"),
	))
	if rule.Protocol == "" {
		errs = append(errs, "params.protocol 不能为空")
	}
	rule.RuleId = firstNonEmpty(attrString(params, "ruleId"), attrString(params, "rule_id"), attrString(params, "securityRuleId"), attrString(params, "security_rule_id"))
	rule.Cidr = firstNonEmpty(
		attrString(params, "cidr"),
		attrString(params, "source"),
		attrString(params, "sourceCidr"),
		attrString(params, "sourceCidrBlock"),
		attrString(params, "destination"),
		attrString(params, "destinationCidr"),
		attrString(params, "destinationCidrBlock"),
	)
	if rule.Cidr == "" && !(rule.Operation == "revoke" && rule.RuleId != "") {
		errs = append(errs, "params.cidr 不能为空")
	}
	rule.Description = firstNonEmpty(attrString(params, "description"), attrString(params, "ruleDescription"), attrString(params, "rule_description"))
	fromPort, toPort, portErr := cloudSecurityRulePortsFromParams(params, rule.Protocol)
	if portErr != "" {
		errs = append(errs, portErr)
	}
	rule.FromPort = fromPort
	rule.ToPort = toPort
	if rule.Operation == "revoke" && rule.RuleId == "" && rule.Cidr == "" {
		errs = append(errs, "撤销规则需要 params.ruleId 或 params.cidr")
	}
	return rule, errs
}

func cloudSecurityRuleOperationValue(params models.ResAttrs) string {
	value := strings.ToLower(firstNonEmpty(
		attrString(params, "operation"),
		attrString(params, "ruleAction"),
		attrString(params, "rule_action"),
		attrString(params, "effect"),
	))
	switch value {
	case "authorize", "add", "create", "allow":
		return "authorize"
	case "revoke", "remove", "delete", "deny":
		return "revoke"
	default:
		return ""
	}
}

func cloudSecurityRulePortsFromParams(params models.ResAttrs, protocol string) (int, int, string) {
	if protocol == "" || protocol == "-1" || protocol == "all" || protocol == "icmp" || protocol == "1" {
		return -1, -1, ""
	}
	if portRange := firstNonEmpty(attrString(params, "portRange"), attrString(params, "port_range")); portRange != "" {
		parts := strings.FieldsFunc(portRange, func(r rune) bool {
			return r == '-' || r == '/' || r == ':'
		})
		if len(parts) == 1 {
			if port, ok := cloudSecurityParsePort(parts[0]); ok {
				return port, port, ""
			}
		}
		if len(parts) >= 2 {
			fromPort, fromOk := cloudSecurityParsePort(parts[0])
			toPort, toOk := cloudSecurityParsePort(parts[1])
			if fromOk && toOk && fromPort <= toPort {
				return fromPort, toPort, ""
			}
		}
		return 0, 0, "params.portRange 格式应为 80 或 80-443"
	}
	fromPort, fromOk := cloudOperationIntParam(params, "fromPort")
	if !fromOk {
		fromPort, fromOk = cloudOperationIntParam(params, "port")
	}
	toPort, toOk := cloudOperationIntParam(params, "toPort")
	if !toOk {
		toPort = fromPort
		toOk = fromOk
	}
	if !fromOk || !toOk {
		return 0, 0, "TCP/UDP 规则需要 params.fromPort/toPort 或 params.portRange"
	}
	if fromPort < 0 || toPort < 0 || fromPort > 65535 || toPort > 65535 || fromPort > toPort {
		return 0, 0, "端口范围必须在 0-65535 且 fromPort <= toPort"
	}
	return fromPort, toPort, ""
}

func cloudSecurityParsePort(value string) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	return parsed, err == nil && parsed >= 0 && parsed <= 65535
}

func cloudSecurityRulePortText(rule cloudSecurityRuleOperation) string {
	if rule.FromPort < 0 || rule.ToPort < 0 {
		return "all"
	}
	if rule.FromPort == rule.ToPort {
		return strconv.Itoa(rule.FromPort)
	}
	return fmt.Sprintf("%d-%d", rule.FromPort, rule.ToPort)
}

func cloudComputeSpecFromAttrs(attrs models.ResAttrs) string {
	if len(attrs) == 0 {
		return ""
	}
	for _, key := range []string{"instanceType", "instance_type", "InstanceType", "shape", "Shape", "shapeName", "shape_name", "flavor", "flavorId", "flavor_id"} {
		value := strings.TrimSpace(fmt.Sprintf("%v", attrs[key]))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func cloudProviderSnapshotName(request cloudProviderOperationRequest) string {
	return cloudProviderSnapshotNameFromParams(request.Params, request.ResourceId, request.OperationId.String())
}

func cloudProviderSnapshotNameFromParams(params models.ResAttrs, resourceId string, operationId string) string {
	for _, key := range []string{"snapshotName", "snapshot_name", "backupName", "backup_name", "displayName", "display_name", "name"} {
		value := strings.TrimSpace(fmt.Sprintf("%v", params[key]))
		if value != "" && value != "<nil>" {
			return cloudProviderNormalizeSnapshotName(value)
		}
	}
	seed := firstNonEmpty(operationId, resourceId, fmt.Sprintf("%d", time.Now().Unix()))
	return cloudProviderNormalizeSnapshotName("cloudiac-" + seed)
}

func cloudProviderSnapshotDescription(request cloudProviderOperationRequest) string {
	for _, key := range []string{"description", "desc", "remark"} {
		value := strings.TrimSpace(fmt.Sprintf("%v", request.Params[key]))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return fmt.Sprintf("Created by CloudIaC operation %s for %s", firstNonEmpty(request.OperationId.String(), request.IdempotencyKey), request.ResourceId)
}

func cloudProviderSnapshotBackupType(request cloudProviderOperationRequest) string {
	for _, key := range []string{"backupType", "backup_type", "type"} {
		value := strings.ToUpper(strings.TrimSpace(fmt.Sprintf("%v", request.Params[key])))
		if value == "FULL" || value == "INCREMENTAL" {
			return value
		}
	}
	return "FULL"
}

func cloudProviderNormalizeSnapshotName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	builder := strings.Builder{}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteByte('-')
	}
	name := strings.Trim(builder.String(), "-_.")
	if len(name) > 80 {
		name = name[:80]
	}
	if name == "" {
		return "cloudiac-snapshot"
	}
	return name
}

func cloudNormalizeComputeState(provider string, state string) string {
	provider = normalizeProvider(provider)
	normalized := strings.ToLower(strings.TrimSpace(state))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.TrimPrefix(normalized, "powerstate/")
	if provider == "gcp" && (normalized == "terminated" || normalized == "suspended") {
		return "stopped"
	}
	switch normalized {
	case "", "<nil>":
		return ""
	case "running", "active":
		return "running"
	case "stopped", "stop", "deallocated", "stopped-stopping":
		return "stopped"
	case "pending", "starting", "provisioning", "staging":
		return "starting"
	case "stopping", "stopped-starting", "deallocating", "suspending":
		return "stopping"
	case "shutting-down", "terminating", "terminated", "terminated-stopping", "deleted":
		return "terminated"
	default:
		return normalized
	}
}

func cloudProviderComputeStateTerminal(state string) bool {
	return state == "terminated" || state == "deleted"
}

func cloudProviderComputeStateTransitional(state string) bool {
	return state == "starting" || state == "stopping" || state == "pending" || state == "provisioning"
}

func cloudProviderOperationRequestAttrs(request cloudProviderOperationRequest, state cloudProviderResourceState) models.ResAttrs {
	return models.ResAttrs{
		"operationId":     request.OperationId.String(),
		"idempotencyKey":  request.IdempotencyKey,
		"action":          request.Action,
		"provider":        request.Provider,
		"providerAdapter": request.ProviderAdapter,
		"resourceType":    request.ResourceType,
		"resourceId":      request.ResourceId,
		"resourceName":    request.ResourceName,
		"region":          request.Region,
		"zone":            request.Zone,
		"accountId":       request.AccountId,
		"cloudAccountId":  request.CloudAccountId.String(),
		"observedState":   request.ObservedState,
		"normalizedState": state.NormalizedState,
		"targetState":     request.TargetState,
		"readMode":        firstNonEmpty(request.ReadMode, state.ReadMode, cloudOperationProviderReadModeCache),
		"params":          cloudProviderSanitizeAttrs(request.Params),
		"stateSource":     state.Source,
		"rawState":        state.RawState,
		"rawResponse":     cloudProviderSanitizeAttrs(state.RawResponse),
		"readError": models.ResAttrs{
			"code":      state.ErrorCode,
			"message":   state.ErrorMessage,
			"retryable": state.Retryable,
		},
	}
}

func cloudOperationProviderReadMode() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(cloudOperationProviderReadModeEnv))) {
	case cloudOperationProviderReadModeLive:
		return cloudOperationProviderReadModeLive
	default:
		return cloudOperationProviderReadModeCache
	}
}

func cloudOperationProviderWriteEnabled() bool {
	value := strings.TrimSpace(os.Getenv(cloudOperationProviderWriteEnv))
	return strings.EqualFold(value, "true") || value == "1" || strings.EqualFold(value, "yes")
}

func cloudAssetActionCheckByKey(dryRun *resps.CloudAssetActionDryRunResp, key string) *resps.CloudAssetActionCheckResp {
	if dryRun == nil {
		return nil
	}
	for index := range dryRun.Checks {
		if dryRun.Checks[index].Key == key {
			return &dryRun.Checks[index]
		}
	}
	return nil
}

func cloudAssetActionFirstFailedCheck(dryRun *resps.CloudAssetActionDryRunResp) *resps.CloudAssetActionCheckResp {
	if dryRun == nil {
		return nil
	}
	for index := range dryRun.Checks {
		if dryRun.Checks[index].Status == "fail" {
			return &dryRun.Checks[index]
		}
	}
	return nil
}

func cloudAssetActionExecutionGuard(dryRun *resps.CloudAssetActionDryRunResp) e.Error {
	if dryRun == nil {
		return e.New(e.BadParam, fmt.Errorf("云资产动作缺少执行预检查结果"))
	}
	if dryRun.Executable {
		return nil
	}
	if check := cloudAssetActionFirstFailedCheck(dryRun); check != nil {
		if check.Key == "user_permission" {
			return e.New(e.PermissionDeny, fmt.Errorf("%s", check.Message))
		}
		return e.New(e.BadParam, fmt.Errorf("云资产动作 %s 当前不可执行：%s", dryRun.Action, check.Message))
	}
	return e.New(e.BadParam, fmt.Errorf("云资产动作 %s 当前不可执行：%s", dryRun.Action, firstNonEmpty(dryRun.DisabledReason, "执行门禁未通过")))
}

func cloudAssetActionConfirmationGuard(asset *models.CmdbAsset, spec cloudAssetActionSpec, form *forms.CreateCloudAssetActionForm) e.Error {
	if spec.AdapterKey != cloudOperationAdapterProviderOperation {
		return nil
	}

	reason := strings.TrimSpace(form.Reason)
	if len([]rune(reason)) < 6 {
		return e.New(e.BadParam, fmt.Errorf("provider 云资产动作需要填写不少于 6 个字符的操作原因"))
	}
	if strings.TrimSpace(form.ConfirmAction) != spec.Key {
		return e.New(e.BadParam, fmt.Errorf("provider 云资产动作需要输入动作 %s 进行二次确认", spec.Key))
	}

	resourceId := firstNonEmpty(asset.NativeId, asset.Id.String())
	if strings.TrimSpace(form.ConfirmResourceId) != resourceId {
		return e.New(e.BadParam, fmt.Errorf("provider 云资产动作需要输入资源 ID %s 进行二次确认", resourceId))
	}
	return nil
}

func cloudAssetActionParamsGuard(asset *models.CmdbAsset, spec cloudAssetActionSpec, tags models.ResAttrs, params models.ResAttrs) e.Error {
	switch spec.Key {
	case models.CloudOperationActionUpdateTags:
		if len(tags) == 0 {
			return e.New(e.BadParam, fmt.Errorf("更新标签动作需要提交 tags JSON 对象"))
		}
	case models.CloudOperationActionResizeInstance:
		targetSpec, ok := cloudProviderInstanceTargetSpecFromParams(params)
		if !ok {
			return e.New(e.BadParam, fmt.Errorf("实例规格调整动作需要提交 params.targetInstanceType 字符串"))
		}
		currentSpec := cloudAssetComputeSpec(asset)
		if currentSpec != "" && targetSpec == currentSpec {
			return e.New(e.BadParam, fmt.Errorf("实例规格调整目标规格 %s 与当前规格一致", targetSpec))
		}
	case models.CloudOperationActionResizeVolume:
		targetSize, ok := cloudOperationIntParam(params, "targetSizeGiB")
		if !ok || targetSize <= 0 {
			return e.New(e.BadParam, fmt.Errorf("磁盘扩容动作需要提交 params.targetSizeGiB 正整数"))
		}
		currentSize := cloudAssetVolumeSizeGiB(asset)
		if currentSize > 0 && targetSize <= currentSize {
			return e.New(e.BadParam, fmt.Errorf("磁盘扩容目标容量 %d GiB 必须大于当前容量 %d GiB", targetSize, currentSize))
		}
	case models.CloudOperationActionUpdateSecurityRules:
		if _, errs := cloudSecurityRuleOperationFromParams(params); len(errs) > 0 {
			return e.New(e.BadParam, fmt.Errorf("更新安全组规则参数无效：%s", strings.Join(errs, "；")))
		}
	}
	return nil
}

func cloudAssetActionParamsFromForm(params models.ResAttrs) models.ResAttrs {
	if len(params) == 0 {
		return nil
	}
	result := models.ResAttrs{}
	for key, value := range params {
		paramKey := strings.TrimSpace(key)
		if paramKey == "" {
			continue
		}
		result[paramKey] = value
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func cloudAssetActionTagsFromForm(tags models.ResAttrs) models.ResAttrs {
	if len(tags) == 0 {
		return nil
	}
	result := models.ResAttrs{}
	for key, value := range tags {
		tagKey := strings.TrimSpace(key)
		if tagKey == "" {
			continue
		}
		if value == nil {
			result[tagKey] = ""
			continue
		}
		result[tagKey] = strings.TrimSpace(fmt.Sprintf("%v", value))
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func cloudOperationIntParam(params models.ResAttrs, key string) (int, bool) {
	if len(params) == 0 {
		return 0, false
	}
	value, ok := params[key]
	if !ok || value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), typed == float64(int(typed))
	case float32:
		return int(typed), typed == float32(int(typed))
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		parsed, err := strconv.Atoi(strings.TrimSpace(fmt.Sprintf("%v", typed)))
		return parsed, err == nil
	}
}

func cloudAssetVolumeSizeGiB(asset *models.CmdbAsset) int {
	if asset == nil {
		return 0
	}
	for _, attrs := range []models.ResAttrs{asset.Attributes, asset.RawData} {
		if size := cloudVolumeSizeGiBFromAttrs(attrs); size > 0 {
			return size
		}
	}
	return 0
}

func cloudAssetVolumeObservedState(asset *models.CmdbAsset) string {
	if size := cloudAssetVolumeSizeGiB(asset); size > 0 {
		return fmt.Sprintf("%dGiB", size)
	}
	return ""
}

func cloudVolumeSizeGiBFromAttrs(attrs models.ResAttrs) int {
	for _, key := range []string{
		"sizeGiB", "size_gib", "sizeInGiB", "size_in_gib",
		"sizeGB", "size_gb", "sizeInGBs", "size_in_gbs",
		"volumeSize", "volume_size", "VolumeSize", "size",
	} {
		if value, ok := cloudOperationIntParam(attrs, key); ok && value > 0 {
			return value
		}
		if value, ok := attrs[key]; ok {
			if parsed := cloudVolumeSizeGiBFromString(fmt.Sprintf("%v", value)); parsed > 0 {
				return parsed
			}
		}
	}
	for _, key := range []string{"sizeInMBs", "size_in_mbs", "sizeMiB", "size_mib"} {
		if value, ok := cloudOperationIntParam(attrs, key); ok && value > 0 {
			return (value + 1023) / 1024
		}
	}
	return 0
}

func cloudVolumeSizeGiBFromString(value string) int {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" || normalized == "<nil>" {
		return 0
	}
	for _, suffix := range []string{"gib", "gb", "g"} {
		normalized = strings.TrimSuffix(normalized, suffix)
	}
	normalized = strings.TrimSpace(normalized)
	parsed, err := strconv.Atoi(normalized)
	if err == nil && parsed > 0 {
		return parsed
	}
	return 0
}

func cloudVolumeStateValue(sizeGiB int) string {
	if sizeGiB <= 0 {
		return "unknown"
	}
	return fmt.Sprintf("%dGiB", sizeGiB)
}

func cloudProviderVolumeCurrentSizeGiB(request cloudProviderOperationRequest, state cloudProviderResourceState) int {
	for _, attrs := range []models.ResAttrs{
		state.RawResponse,
		modelResAttrs(state.RawResponse["response"]),
		modelResAttrs(state.RawResponse["request"]),
	} {
		if size := cloudVolumeSizeGiBFromAttrs(attrs); size > 0 {
			return size
		}
	}
	for _, value := range []string{state.NormalizedState, state.RawState, request.ObservedState} {
		if size := cloudVolumeSizeGiBFromString(value); size > 0 {
			return size
		}
	}
	return 0
}

func cloudAssetActionCheckAttrs(check *resps.CloudAssetActionCheckResp) models.ResAttrs {
	if check == nil {
		return nil
	}
	return models.ResAttrs{
		"key":     check.Key,
		"name":    check.Name,
		"status":  check.Status,
		"message": check.Message,
	}
}

func cloudAssetActionChecksAttrs(checks []resps.CloudAssetActionCheckResp) []models.ResAttrs {
	attrs := make([]models.ResAttrs, 0, len(checks))
	for index := range checks {
		attrs = append(attrs, cloudAssetActionCheckAttrs(&checks[index]))
	}
	return attrs
}

func cloudAssetActionExecutionPlan(c *ctx.ServiceContext, asset *models.CmdbAsset, spec cloudAssetActionSpec, dryRun *resps.CloudAssetActionDryRunResp) models.ResAttrs {
	if dryRun == nil {
		dryRun = &resps.CloudAssetActionDryRunResp{}
	}
	guardStatus := "pass"
	guardMessage := "执行门禁已通过"
	blockingCheck := cloudAssetActionFirstFailedCheck(dryRun)
	if !dryRun.Executable {
		guardStatus = "blocked"
		guardMessage = firstNonEmpty(dryRun.DisabledReason, "执行门禁未通过")
		if blockingCheck != nil {
			guardMessage = blockingCheck.Message
		}
	}
	plan := models.ResAttrs{
		"action":          spec.Key,
		"adapterKey":      dryRun.AdapterKey,
		"adapterMode":     dryRun.AdapterMode,
		"adapterStatus":   dryRun.AdapterStatus,
		"adapterMessage":  dryRun.AdapterMessage,
		"providerAdapter": dryRun.ProviderAdapter,
		"writeEnabled":    dryRun.WriteEnabled,
		"guardStatus":     guardStatus,
		"guardMessage":    guardMessage,
		"blockingCheck":   cloudAssetActionCheckAttrs(blockingCheck),
		"preflightChecks": cloudAssetActionChecksAttrs(dryRun.Checks),
		"resource": models.ResAttrs{
			"provider":       asset.Provider,
			"resourceType":   asset.AssetType,
			"resourceId":     asset.NativeId,
			"resourceName":   firstNonEmpty(asset.Name, asset.NativeId, asset.Id.String()),
			"region":         asset.Region,
			"accountId":      asset.AccountId,
			"cloudAccountId": asset.CloudAccountId.String(),
		},
	}
	if adapter, ok := cloudProviderOperationAdapterForAction(asset.Provider, spec.Key); ok {
		request := adapter.BuildRequest(asset, spec, nil, dryRun)
		account := cloudOperationCmdbCloudAccountForAsset(c, asset)
		state := adapter.ReadResourceState(request, account)
		plan["providerRequest"] = cloudProviderOperationRequestAttrs(request, state)
		if spec.AdapterKey == cloudOperationAdapterProviderOperation {
			plan["providerStatePolling"] = models.ResAttrs{
				"attempts":        cloudOperationProviderPollAttempts(),
				"initialInterval": cloudOperationProviderPollInterval().String(),
				"maxDelay":        cloudOperationProviderPollMaxDelay().String(),
			}
		}
	}
	return plan
}

func executeCloudAssetAction(c *ctx.ServiceContext, operation *models.CloudOperation, asset *models.CmdbAsset, spec cloudAssetActionSpec, dryRun *resps.CloudAssetActionDryRunResp) (models.ResAttrs, e.Error) {
	if err := cloudAssetActionExecutionGuard(dryRun); err != nil {
		return nil, err
	}
	switch spec.AdapterKey {
	case cloudOperationAdapterLocalMetadata:
		if spec.Key == models.CloudOperationActionUpdateTags {
			return executeCloudAssetTagUpdate(c, operation, asset, spec, dryRun)
		}
		return executeCloudAssetMetadataRefresh(c, operation, asset, spec, dryRun)
	case cloudOperationAdapterProviderOperation:
		return executeCloudAssetProviderAction(c, operation, asset, spec, dryRun)
	default:
		return nil, e.New(e.BadParam, fmt.Errorf("云资产动作 %s 的 provider operation adapter 未接入", spec.Key))
	}
}

func executeCloudAssetProviderAction(c *ctx.ServiceContext, operation *models.CloudOperation, asset *models.CmdbAsset, spec cloudAssetActionSpec, dryRun *resps.CloudAssetActionDryRunResp) (models.ResAttrs, e.Error) {
	plan := cloudAssetActionExecutionPlan(c, asset, spec, dryRun)
	adapter, ok := cloudProviderOperationAdapterForAction(asset.Provider, spec.Key)
	if !ok {
		return models.ResAttrs{
			"executionPlan":           plan,
			"providerAdapterExecuted": false,
			"operationId":             operation.Id.String(),
		}, e.New(e.BadParam, fmt.Errorf("云资产动作 %s 的 provider operation adapter 未注册", spec.Key))
	}
	request := adapter.BuildRequest(asset, spec, operation, dryRun)
	account := cloudOperationCmdbCloudAccountForAsset(c, asset)
	state := adapter.ReadResourceState(request, account)
	result, err := adapter.Execute(request, state, account)
	if result == nil {
		result = models.ResAttrs{}
	}
	result["executionPlan"] = plan
	if _, ok := result["providerAdapterExecuted"]; !ok {
		result["providerAdapterExecuted"] = false
	}
	result["operationId"] = operation.Id.String()
	if _, ok := result["message"]; !ok {
		result["message"] = fmt.Sprintf("provider adapter %s 已完成云端写操作执行分支", adapter.Key())
	}
	return result, err
}

func executeCloudAssetMetadataRefresh(c *ctx.ServiceContext, operation *models.CloudOperation, asset *models.CmdbAsset, spec cloudAssetActionSpec, dryRun *resps.CloudAssetActionDryRunResp) (models.ResAttrs, e.Error) {
	after := *asset
	after.ManagedBy = inferCmdbManagedBy(&after)
	after.LastOperationId = operation.Id
	attrs := models.Attrs{
		"managed_by":        after.ManagedBy,
		"last_operation_id": operation.Id,
	}
	if _, err := c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and id = ?", c.OrgId, asset.Id).
		UpdateAttrs(attrs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if diff := cmdbAssetDiff(*asset, &after); len(diff) > 0 {
		if err := recordCmdbAssetChange(c, c.OrgId, asset.Id, models.CmdbAssetChangeTypeUpdated, models.CmdbAssetChangeSourceManual, diff); err != nil {
			return nil, err
		}
	}
	return models.ResAttrs{
		"adapter": models.ResAttrs{
			"key":  spec.AdapterKey,
			"mode": spec.AdapterMode,
		},
		"executionPlan":           cloudAssetActionExecutionPlan(c, asset, spec, dryRun),
		"dryRun":                  dryRun,
		"providerAdapterExecuted": false,
		"refreshedFields":         []string{"managedBy", "lastOperationId"},
		"message":                 "资产元数据刷新请求已记录，当前版本未执行云端变更",
	}, nil
}

func executeCloudAssetTagUpdate(c *ctx.ServiceContext, operation *models.CloudOperation, asset *models.CmdbAsset, spec cloudAssetActionSpec, dryRun *resps.CloudAssetActionDryRunResp) (models.ResAttrs, e.Error) {
	tagUpdates := cloudAssetActionTagsFromForm(modelResAttrs(operation.Params["tags"]))
	if len(tagUpdates) == 0 {
		return nil, e.New(e.BadParam, fmt.Errorf("更新标签动作缺少 tags 参数"))
	}

	before := copyResAttrs(asset.Tags)
	after := copyResAttrs(asset.Tags)
	changed := make([]string, 0, len(tagUpdates))
	removed := make([]string, 0)
	for key, value := range tagUpdates {
		valueText := strings.TrimSpace(fmt.Sprintf("%v", value))
		if valueText == "" {
			if _, ok := after[key]; ok {
				delete(after, key)
				removed = append(removed, key)
			}
			continue
		}
		if attrString(after, key) != valueText {
			after[key] = valueText
			changed = append(changed, key)
		}
	}

	if _, err := c.DB().Model(&models.CmdbAsset{}).
		Where("id = ? and org_id = ?", asset.Id, asset.OrgId).
		UpdateAttrs(models.Attrs{
			"tags":              after,
			"last_operation_id": operation.Id,
		}); err != nil {
		return nil, e.New(e.DBError, err)
	}
	asset.Tags = after
	asset.LastOperationId = operation.Id

	diff := models.ResAttrs{}
	appendCmdbJSONDiff(diff, "tags", before, after)
	if err := recordCmdbAssetChange(c, asset.OrgId, asset.Id, models.CmdbAssetChangeTypeUpdated, "cloud_operation", diff); err != nil {
		return nil, err
	}

	return models.ResAttrs{
		"adapter": models.ResAttrs{
			"key":  spec.AdapterKey,
			"mode": spec.AdapterMode,
		},
		"executionPlan":           cloudAssetActionExecutionPlan(c, asset, spec, dryRun),
		"dryRun":                  dryRun,
		"providerAdapterExecuted": false,
		"changedTags":             changed,
		"removedTags":             removed,
		"beforeTags":              before,
		"afterTags":               after,
		"message":                 "资产标签已在 CMDB 本地更新，当前版本未执行云端标签写入",
	}, nil
}

func copyResAttrs(attrs models.ResAttrs) models.ResAttrs {
	result := models.ResAttrs{}
	for key, value := range attrs {
		result[key] = value
	}
	return result
}

func stringInSlice(value string, options []string) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}
