// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
	"cloudiac/utils/logs"
)

const (
	cloudSyncPolicyDefaultInterval        = 24 * time.Hour
	cloudSyncPolicyMinInterval            = 60 * time.Second
	cloudSyncPolicyMaxInterval            = 30 * 24 * time.Hour
	cloudSyncPolicyDefaultRetryBackoff    = 5 * time.Minute
	cloudSyncPolicyMaxRetryBackoff        = 24 * time.Hour
	cloudSyncPolicyDefaultRetryAttempts   = 3
	cloudSyncPolicyMaxRetryAttempts       = 10
	cloudSyncPolicyDefaultAutoRetryScopes = 10
	cloudSyncPolicyMaxAutoRetryScopes     = 50
	cloudSyncPolicyWorkerDefaultInterval  = 5 * time.Minute
	cloudSyncPolicyWorkerMinInterval      = 30 * time.Second
	cloudSyncPolicyWorkerMaxInterval      = 24 * time.Hour
	cloudSyncPolicyMaxEscalationAt        = 100

	cloudSyncPolicyWorkerIntervalEnv       = "CLOUDIAC_CLOUD_SYNC_POLICY_WORKER_INTERVAL_SECONDS"
	cloudSyncPolicyMaxTriggeredPerRunEnv   = "CLOUDIAC_CLOUD_SYNC_POLICY_MAX_TRIGGERED_PER_RUN"
	cloudSyncPolicyMaxConcurrentRunningEnv = "CLOUDIAC_CLOUD_SYNC_POLICY_MAX_CONCURRENT_RUNNING"
)

type cloudSyncPolicyScheduleOverride struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	Regions      []string `json:"regions"`
	AssetTypes   []string `json:"assetTypes"`
	SyncInterval int      `json:"syncInterval"`
}

type cloudSyncPolicyScheduleState struct {
	NextSyncAt      string `json:"nextSyncAt,omitempty"`
	LastSyncedAt    string `json:"lastSyncedAt,omitempty"`
	LastSyncTaskId  string `json:"lastSyncTaskId,omitempty"`
	LastSyncStatus  string `json:"lastSyncStatus,omitempty"`
	LastError       string `json:"lastError,omitempty"`
	FailureCount    int    `json:"failureCount,omitempty"`
	LastFailureAt   string `json:"lastFailureAt,omitempty"`
	LastFailureText string `json:"lastFailureText,omitempty"`
}

type cloudSyncPolicyRunScope struct {
	cloudSyncPolicyScheduleOverride
	Default bool
}

func SearchCloudSyncPolicies(c *ctx.ServiceContext, form *forms.SearchCloudSyncPolicyForm) (interface{}, e.Error) {
	query := c.DB().Model(&models.CloudSyncPolicy{}).Where("org_id = ?", c.OrgId)
	if form.Q != "" {
		q := "%" + strings.TrimSpace(form.Q) + "%"
		query = query.Where(`id like ? or name like ? or description like ? or provider like ? or
			account_id like ? or last_error like ? or params like ?`, q, q, q, q, q, q, q)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	if form.Provider != "" {
		query = query.Where("provider = ?", normalizeProvider(form.Provider))
	}
	if form.CloudAccountId != "" {
		query = query.Where("cloud_account_id = ?", form.CloudAccountId)
	}
	if form.AccountId != "" {
		query = query.Where("account_id = ?", form.AccountId)
	}
	if form.LastSyncStatus != "" {
		query = query.Where("last_sync_status = ?", form.LastSyncStatus)
	}
	if form.SortField() == "" {
		query = query.Order("next_sync_at asc").Order("created_at desc")
	} else {
		query = form.Order(query)
	}

	policies := make([]models.CloudSyncPolicy, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&policies); err != nil {
		return nil, e.New(e.DBError, err)
	}
	list := make([]resps.CloudSyncPolicyResp, 0, len(policies))
	for _, policy := range policies {
		list = append(list, cloudSyncPolicyResp(c, policy))
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func CloudSyncPolicyDetail(c *ctx.ServiceContext, form *forms.CloudSyncPolicyParam) (*resps.CloudSyncPolicyResp, e.Error) {
	policy, err := getCloudSyncPolicy(c, form.Id)
	if err != nil {
		return nil, err
	}
	resp := cloudSyncPolicyResp(c, *policy)
	return &resp, nil
}

func CreateCloudSyncPolicy(c *ctx.ServiceContext, form *forms.CreateCloudSyncPolicyForm) (*resps.CloudSyncPolicyResp, e.Error) {
	policy, err := cloudSyncPolicyFromForm(c, form)
	if err != nil {
		return nil, err
	}
	policy.Id = models.NewId("csp")
	if err := models.Create(c.DB(), policy); err != nil {
		return nil, e.New(e.DBError, err)
	}
	resp := cloudSyncPolicyResp(c, *policy)
	return &resp, nil
}

func UpdateCloudSyncPolicy(c *ctx.ServiceContext, form *forms.UpdateCloudSyncPolicyForm) (*resps.CloudSyncPolicyResp, e.Error) {
	existing, err := getCloudSyncPolicy(c, form.Id)
	if err != nil {
		return nil, err
	}
	updated, err := cloudSyncPolicyFromForm(c, &form.CreateCloudSyncPolicyForm)
	if err != nil {
		return nil, err
	}
	attrs := models.Attrs{
		"name":                  updated.Name,
		"description":           updated.Description,
		"cloud_account_id":      updated.CloudAccountId,
		"provider":              updated.Provider,
		"account_id":            updated.AccountId,
		"regions":               updated.Regions,
		"asset_types":           updated.AssetTypes,
		"status":                updated.Status,
		"sync_interval":         updated.SyncInterval,
		"max_retry_attempts":    updated.MaxRetryAttempts,
		"retry_backoff_seconds": updated.RetryBackoffSeconds,
		"notify_on_failure":     updated.NotifyOnFailure,
		"auto_pause_on_failure": updated.AutoPauseOnFailure,
		"next_sync_at":          updated.NextSyncAt,
		"params":                updated.Params,
	}
	if existing.Status != models.CloudSyncPolicyStatusEnabled && updated.Status == models.CloudSyncPolicyStatusEnabled {
		attrs["failure_count"] = 0
		attrs["last_failure_at"] = nil
		attrs["last_failure_reason"] = ""
		attrs["last_error"] = ""
	}
	if _, dbErr := c.DB().Model(&models.CloudSyncPolicy{}).
		Where("id = ? and org_id = ?", form.Id, c.OrgId).
		UpdateAttrs(attrs); dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}
	updated.Id = form.Id
	updated.OrgId = c.OrgId
	updated.CreatorId = existing.CreatorId
	updated.LastSyncTaskId = existing.LastSyncTaskId
	updated.LastSyncStatus = existing.LastSyncStatus
	updated.LastError = existing.LastError
	updated.FailureCount = existing.FailureCount
	updated.LastFailureAt = existing.LastFailureAt
	updated.LastFailureReason = existing.LastFailureReason
	updated.LastSyncedAt = existing.LastSyncedAt
	if existing.Status != models.CloudSyncPolicyStatusEnabled && updated.Status == models.CloudSyncPolicyStatusEnabled {
		updated.FailureCount = 0
		updated.LastFailureAt = models.Time(time.Time{})
		updated.LastFailureReason = ""
		updated.LastError = ""
	}
	resp := cloudSyncPolicyResp(c, *updated)
	return &resp, nil
}

func DeleteCloudSyncPolicy(c *ctx.ServiceContext, form *forms.CloudSyncPolicyParam) (interface{}, e.Error) {
	if _, err := c.DB().Where("id = ? and org_id = ?", form.Id, c.OrgId).Delete(&models.CloudSyncPolicy{}); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return nil, nil
}

func RunCloudSyncPolicy(c *ctx.ServiceContext, form *forms.CloudSyncPolicyParam) (*resps.CmdbSyncTaskResp, e.Error) {
	policy, err := getCloudSyncPolicy(c, form.Id)
	if err != nil {
		return nil, err
	}
	return runCloudSyncPolicy(c, policy)
}

func RunDueCloudSyncPolicies(c *ctx.ServiceContext, form *forms.RunDueCloudSyncPolicyForm) (*resps.CloudSyncPolicyRunResp, e.Error) {
	policies := make([]models.CloudSyncPolicy, 0)
	query := c.DB().Model(&models.CloudSyncPolicy{}).
		Where("org_id = ? and status = ?", c.OrgId, models.CloudSyncPolicyStatusEnabled)
	if !form.Force {
		query = query.Where("next_sync_at is null or next_sync_at <= ?", time.Now())
	}
	if err := query.Order("next_sync_at asc").Find(&policies); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return runCloudSyncPolicies(c, policies, form.Force)
}

func RunDueCloudSyncPoliciesForAllOrgs() (*resps.CloudSyncPolicyRunResp, e.Error) {
	policies := make([]models.CloudSyncPolicy, 0)
	if err := db.Get().Model(&models.CloudSyncPolicy{}).
		Where("status = ? and (next_sync_at is null or next_sync_at <= ?)", models.CloudSyncPolicyStatusEnabled, time.Now()).
		Order("next_sync_at asc").
		Find(&policies); err != nil {
		return nil, e.New(e.DBError, err)
	}
	resp := &resps.CloudSyncPolicyRunResp{}
	for idx := range policies {
		policy := &policies[idx]
		if policy.OrgId == "" {
			resp.TotalCount++
			resp.SkippedCount++
			continue
		}
		workerCtx := &ctx.ServiceContext{
			UserId:   consts.SysUserId,
			OrgId:    policy.OrgId,
			Email:    consts.DefaultSysEmail,
			Username: consts.DefaultSysName,
		}
		itemResp, err := runCloudSyncPoliciesWithTriggeredOffset(workerCtx, []models.CloudSyncPolicy{*policy}, resp.TriggeredCount, false)
		if err != nil {
			return nil, err
		}
		resp.TotalCount += itemResp.TotalCount
		resp.TriggeredCount += itemResp.TriggeredCount
		resp.FailedCount += itemResp.FailedCount
		resp.SkippedCount += itemResp.SkippedCount
		resp.LockSkippedCount += itemResp.LockSkippedCount
		resp.ProtectionSkippedCount += itemResp.ProtectionSkippedCount
		resp.ProtectionSkippedReasons = append(resp.ProtectionSkippedReasons, itemResp.ProtectionSkippedReasons...)
		resp.TaskIds = append(resp.TaskIds, itemResp.TaskIds...)
	}
	return resp, nil
}

func StartCloudSyncPolicyWorker(serviceId string) {
	interval := cloudSyncPolicyWorkerInterval()
	logger := logs.Get().
		WithField("worker", "cloudSyncPolicy").
		WithField("serviceId", serviceId).
		WithField("interval", interval.String())
	logger.Infof("cloud sync policy worker started")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if result, err := RunDueCloudSyncPoliciesForAllOrgs(); err != nil {
			logger.Warnf("run due cloud sync policies failed: %v", err)
		} else if result.TriggeredCount > 0 || result.FailedCount > 0 || result.LockSkippedCount > 0 || result.ProtectionSkippedCount > 0 {
			logger.Infof("run due cloud sync policies result: %+v", result)
		}
		<-ticker.C
	}
}

func runCloudSyncPolicies(c *ctx.ServiceContext, policies []models.CloudSyncPolicy, force bool) (*resps.CloudSyncPolicyRunResp, e.Error) {
	return runCloudSyncPoliciesWithTriggeredOffset(c, policies, 0, force)
}

func runCloudSyncPoliciesWithTriggeredOffset(c *ctx.ServiceContext, policies []models.CloudSyncPolicy, triggeredOffset int, force bool) (*resps.CloudSyncPolicyRunResp, e.Error) {
	resp := &resps.CloudSyncPolicyRunResp{}
	now := time.Now()
	for idx := range policies {
		policy := &policies[idx]
		if policy.Status != models.CloudSyncPolicyStatusEnabled {
			resp.TotalCount++
			resp.SkippedCount++
			continue
		}
		runScopes := cloudSyncPolicyDueRunScopes(policy, now, force)
		if len(runScopes) == 0 {
			resp.TotalCount++
			resp.SkippedCount++
			continue
		}
		for _, scope := range runScopes {
			resp.TotalCount++
			if reason, protected := cloudSyncPolicyScheduleProtection(c, policy, triggeredOffset+resp.TriggeredCount); protected {
				resp.ProtectionSkippedCount++
				resp.ProtectionSkippedReasons = append(resp.ProtectionSkippedReasons, reason)
				continue
			}
			locked, releaseLock, lockErr := cloudWebhookAcquireMysqlLock(cloudSyncPolicyLockName(policy.OrgId, policy.Id, scope.Key))
			if lockErr != nil {
				return nil, e.New(e.DBError, lockErr)
			}
			if !locked {
				resp.LockSkippedCount++
				continue
			}
			task, err := func() (*resps.CmdbSyncTaskResp, e.Error) {
				defer releaseLock()
				if scope.Default {
					return runCloudSyncPolicy(c, policy)
				}
				return runCloudSyncPolicyScope(c, policy, scope)
			}()
			if err != nil {
				resp.FailedCount++
				continue
			}
			resp.TriggeredCount++
			resp.TaskIds = append(resp.TaskIds, task.Id.String())
		}
	}
	return resp, nil
}

func runCloudSyncPolicy(c *ctx.ServiceContext, policy *models.CloudSyncPolicy) (*resps.CmdbSyncTaskResp, e.Error) {
	if policy == nil {
		return nil, e.New(e.BadParam, fmt.Errorf("云资产同步策略不存在"))
	}
	nowTime := time.Now()
	now := models.Time(nowTime)
	task, err := StartCmdbSyncTask(c, &forms.CreateCmdbSyncTaskForm{
		AccountSource:         models.CmdbCloudAccountSourceCloudAccount,
		AccountId:             policy.CloudAccountId,
		Provider:              policy.Provider,
		Regions:               []string(policy.Regions),
		AssetTypes:            []string(policy.AssetTypes),
		SyncPolicyId:          policy.Id,
		SlowApiThresholdMs:    cloudSyncPolicySlowAPIThresholdMs(policy),
		SlowApiSilenceMinutes: cloudSyncPolicySlowAPISilenceMinutes(policy),
	})
	nextSyncAt := models.Time(nowTime.Add(time.Duration(cloudSyncPolicyIntervalSeconds(policy.SyncInterval)) * time.Second))
	attrs := models.Attrs{
		"last_synced_at": now,
		"next_sync_at":   nextSyncAt,
	}
	if err != nil {
		attrs["last_sync_status"] = models.CmdbSyncTaskFailed
		attrs["last_error"] = err.Error()
		cloudSyncPolicyApplyFailure(policy, attrs, nowTime, err.Error())
		if _, dbErr := c.DB().Model(&models.CloudSyncPolicy{}).
			Where("id = ? and org_id = ?", policy.Id, policy.OrgId).
			UpdateAttrs(attrs); dbErr != nil {
			return nil, e.New(e.DBError, dbErr)
		}
		cloudSyncPolicyFailureEvent(c, policy, "", err.Error(), nil)
		return nil, err
	}
	attrs["last_sync_task_id"] = task.Id
	attrs["last_sync_status"] = task.Status
	attrs["last_error"] = ""
	if _, dbErr := c.DB().Model(&models.CloudSyncPolicy{}).
		Where("id = ? and org_id = ?", policy.Id, policy.OrgId).
		UpdateAttrs(attrs); dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}
	cloudSyncPolicyEvent(c, policy, "cloud.sync.policy.triggered", models.CloudEventLevelInfo, task.Status,
		"云资产同步策略已触发", fmt.Sprintf("同步策略 %s 已创建采集任务", policy.Name), models.ResAttrs{
			"policyId": policy.Id.String(),
			"taskId":   task.Id.String(),
		})
	return task, nil
}

func runCloudSyncPolicyScope(c *ctx.ServiceContext, policy *models.CloudSyncPolicy, scope cloudSyncPolicyRunScope) (*resps.CmdbSyncTaskResp, e.Error) {
	if policy == nil {
		return nil, e.New(e.BadParam, fmt.Errorf("云资产同步策略不存在"))
	}
	nowTime := time.Now()
	now := models.Time(nowTime)
	task, err := StartCmdbSyncTask(c, &forms.CreateCmdbSyncTaskForm{
		AccountSource:          models.CmdbCloudAccountSourceCloudAccount,
		AccountId:              policy.CloudAccountId,
		Provider:               policy.Provider,
		Regions:                scope.Regions,
		AssetTypes:             scope.AssetTypes,
		SyncPolicyId:           policy.Id,
		SyncPolicyScheduleKey:  scope.Key,
		SyncPolicyScheduleName: scope.Name,
		SlowApiThresholdMs:     cloudSyncPolicySlowAPIThresholdMs(policy),
		SlowApiSilenceMinutes:  cloudSyncPolicySlowAPISilenceMinutes(policy),
	})
	stateStatus := models.CmdbSyncTaskRunning
	stateError := ""
	attrs := models.Attrs{
		"last_synced_at": now,
	}
	if err != nil {
		stateStatus = models.CmdbSyncTaskFailed
		stateError = err.Error()
		attrs["last_sync_status"] = models.CmdbSyncTaskFailed
		attrs["last_error"] = err.Error()
		cloudSyncPolicyApplyFailure(policy, attrs, nowTime, err.Error())
		cloudSyncPolicyFailureEvent(c, policy, "", err.Error(), nil)
	} else {
		attrs["last_sync_task_id"] = task.Id
		attrs["last_sync_status"] = task.Status
		attrs["last_error"] = ""
	}
	failureCount := 0
	lastFailureAt := ""
	lastFailureText := ""
	if err != nil {
		failureCount = policy.FailureCount + 1
		lastFailureAt = nowTime.Format(time.RFC3339)
		lastFailureText = err.Error()
	}
	statePolicy, policyErr := getCloudSyncPolicy(c, policy.Id)
	if policyErr != nil {
		return nil, policyErr
	}
	params := cloudSyncPolicyWithScheduleState(statePolicy, scope.Key, cloudSyncPolicyScheduleState{
		NextSyncAt:      cloudSyncPolicyScheduleNextAfterTask(policy, scope.Key, stateStatus, nowTime, failureCount).Format(time.RFC3339),
		LastSyncedAt:    nowTime.Format(time.RFC3339),
		LastSyncTaskId:  taskIdString(task),
		LastSyncStatus:  stateStatus,
		LastError:       stateError,
		FailureCount:    failureCount,
		LastFailureAt:   lastFailureAt,
		LastFailureText: lastFailureText,
	})
	attrs["params"] = cloudProviderSanitizeAttrs(params)
	attrs["next_sync_at"] = cloudSyncPolicyNextSyncAtFromParams(policy, params)
	if _, dbErr := c.DB().Model(&models.CloudSyncPolicy{}).
		Where("id = ? and org_id = ?", policy.Id, policy.OrgId).
		UpdateAttrs(attrs); dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}
	policy.Params = params
	policy.NextSyncAt = attrs["next_sync_at"].(models.Time)
	if err != nil {
		return nil, err
	}
	if refreshErr := refreshCloudSyncPolicyTaskStatusIfFinished(c, policy, task); refreshErr != nil {
		return nil, e.New(e.DBError, refreshErr)
	}
	cloudSyncPolicyEvent(c, policy, "cloud.sync.policy.triggered", models.CloudEventLevelInfo, task.Status,
		"云资产同步策略子计划已触发", fmt.Sprintf("同步策略 %s 的子计划 %s 已创建采集任务", policy.Name, scope.Name), models.ResAttrs{
			"policyId":               policy.Id.String(),
			"taskId":                 task.Id.String(),
			"syncPolicyScheduleKey":  scope.Key,
			"syncPolicyScheduleName": scope.Name,
			"regions":                scope.Regions,
			"assetTypes":             scope.AssetTypes,
		})
	return task, nil
}

func updateCloudSyncPolicyAfterTask(c *ctx.ServiceContext, policyId models.Id, taskId models.Id, status string, errorMessage string, endedAt models.Time, stats models.ResAttrs) error {
	if c == nil || policyId == "" {
		return nil
	}
	policy := models.CloudSyncPolicy{}
	if err := c.DB().Model(&models.CloudSyncPolicy{}).
		Where("id = ? and org_id = ?", policyId, c.OrgId).
		First(&policy); err != nil {
		if e.IsRecordNotFound(err) {
			return nil
		}
		return err
	}
	ended := time.Time(endedAt)
	if ended.IsZero() || ended.Year() <= 1 {
		ended = time.Now()
		endedAt = models.Time(ended)
	}
	attrs := models.Attrs{
		"last_sync_task_id": taskId,
		"last_sync_status":  status,
		"last_error":        errorMessage,
		"last_synced_at":    endedAt,
	}
	if scheduleKey := cloudSyncPolicyStatsString(stats, "syncPolicyScheduleKey"); scheduleKey != "" {
		currentState := cloudSyncPolicyScheduleStates(&policy)[scheduleKey]
		if currentState.LastSyncTaskId == taskId.String() &&
			currentState.LastSyncStatus == status &&
			cloudSyncPolicyTerminalTaskStatus(status) {
			return nil
		}
		state := cloudSyncPolicyScheduleState{
			LastSyncedAt:   ended.Format(time.RFC3339),
			LastSyncTaskId: taskId.String(),
			LastSyncStatus: status,
			LastError:      errorMessage,
		}
		if status == models.CmdbSyncTaskComplete {
			attrs["failure_count"] = 0
			attrs["last_failure_at"] = nil
			attrs["last_failure_reason"] = ""
		} else {
			failureCount := policy.FailureCount + 1
			state.FailureCount = failureCount
			state.LastFailureAt = ended.Format(time.RFC3339)
			state.LastFailureText = errorMessage
			attrs["failure_count"] = failureCount
			attrs["last_failure_at"] = models.Time(ended)
			attrs["last_failure_reason"] = errorMessage
			if policy.AutoPauseOnFailure && policy.MaxRetryAttempts > 0 && failureCount > policy.MaxRetryAttempts {
				attrs["status"] = models.CloudSyncPolicyStatusDisabled
			}
		}
		state.NextSyncAt = cloudSyncPolicyScheduleNextAfterTask(&policy, scheduleKey, status, ended, state.FailureCount).Format(time.RFC3339)
		params := cloudSyncPolicyWithScheduleState(&policy, scheduleKey, state)
		attrs["params"] = cloudProviderSanitizeAttrs(params)
		if attrs["status"] == models.CloudSyncPolicyStatusDisabled {
			attrs["next_sync_at"] = nil
		} else {
			attrs["next_sync_at"] = cloudSyncPolicyNextSyncAtFromParams(&policy, params)
		}
		if status == models.CmdbSyncTaskComplete {
			cloudSyncPolicyEvent(c, &policy, "cloud.sync.policy.completed", models.CloudEventLevelInfo, status,
				"云资产同步策略子计划执行完成", fmt.Sprintf("同步策略 %s 的子计划 %s 已完成", policy.Name, cloudSyncPolicyScheduleName(&policy, scheduleKey)), models.ResAttrs{
					"policyId":              policy.Id.String(),
					"taskId":                taskId.String(),
					"stats":                 stats,
					"syncPolicyScheduleKey": scheduleKey,
				})
		} else {
			cloudSyncPolicyFailureEvent(c, &policy, taskId, errorMessage, stats)
		}
		_, err := c.DB().Model(&models.CloudSyncPolicy{}).
			Where("id = ? and org_id = ?", policyId, c.OrgId).
			UpdateAttrs(attrs)
		return err
	}
	if status == models.CmdbSyncTaskComplete {
		attrs["failure_count"] = 0
		attrs["last_failure_at"] = nil
		attrs["last_failure_reason"] = ""
		attrs["next_sync_at"] = models.Time(ended.Add(time.Duration(cloudSyncPolicyIntervalSeconds(policy.SyncInterval)) * time.Second))
		cloudSyncPolicyEvent(c, &policy, "cloud.sync.policy.completed", models.CloudEventLevelInfo, status,
			"云资产同步策略执行完成", fmt.Sprintf("同步策略 %s 的采集任务已完成", policy.Name), models.ResAttrs{
				"policyId": policy.Id.String(),
				"taskId":   taskId.String(),
				"stats":    stats,
			})
	} else {
		cloudSyncPolicyApplyFailure(&policy, attrs, ended, errorMessage)
		cloudSyncPolicyFailureEvent(c, &policy, taskId, errorMessage, stats)
	}
	_, err := c.DB().Model(&models.CloudSyncPolicy{}).
		Where("id = ? and org_id = ?", policyId, c.OrgId).
		UpdateAttrs(attrs)
	return err
}

func refreshCloudSyncPolicyTaskStatusIfFinished(c *ctx.ServiceContext, policy *models.CloudSyncPolicy, task *resps.CmdbSyncTaskResp) error {
	if c == nil || policy == nil || task == nil {
		return nil
	}
	latestTask := models.CmdbSyncTask{}
	if err := c.DB().Model(&models.CmdbSyncTask{}).
		Where("id = ? and org_id = ?", task.Id, c.OrgId).
		First(&latestTask); err != nil {
		if e.IsRecordNotFound(err) {
			return nil
		}
		return err
	}
	if !cloudSyncPolicyTerminalTaskStatus(latestTask.Status) {
		return nil
	}
	return updateCloudSyncPolicyAfterTask(c, policy.Id, latestTask.Id, latestTask.Status, latestTask.ErrorMessage, latestTask.EndedAt, latestTask.Stats)
}

func cloudSyncPolicyFromForm(c *ctx.ServiceContext, form *forms.CreateCloudSyncPolicyForm) (*models.CloudSyncPolicy, e.Error) {
	account, err := findCmdbCloudAccount(c, models.CmdbCloudAccountSourceCloudAccount, form.CloudAccountId)
	if err != nil {
		return nil, err
	}
	provider := normalizeProvider(form.Provider)
	if provider == "" {
		provider = account.Provider
	}
	if account.Provider != "" && provider != account.Provider {
		return nil, e.New(e.BadParam, fmt.Errorf("同步策略云厂商 %s 与云账号厂商 %s 不一致", provider, account.Provider), http.StatusBadRequest)
	}
	regions := normalizeStringList(form.Regions)
	if len(regions) == 0 {
		regions = account.Regions
	}
	assetTypes := normalizeStringList(form.AssetTypes)
	if len(assetTypes) == 0 {
		assetTypes = account.SupportedAssetTypes
	}
	status := firstNonEmpty(form.Status, models.CloudSyncPolicyStatusEnabled)
	params := models.ResAttrs{}
	for key, value := range form.Params {
		params[key] = value
	}
	params = cloudSyncPolicyNormalizeParams(params)
	nextSyncAt := form.NextSyncAt
	if time.Time(nextSyncAt).IsZero() || time.Time(nextSyncAt).Year() <= 1 {
		nextSyncAt = models.Time(time.Now())
	}
	if len(cloudSyncPolicyDecodeScheduleOverrides(params["scheduleOverrides"])) > 0 {
		nextSyncAt = cloudSyncPolicyNextSyncAtFromParams(nil, params)
	}
	return &models.CloudSyncPolicy{
		OrgId:               c.OrgId,
		CreatorId:           c.UserId,
		Name:                form.Name,
		Description:         form.Description,
		CloudAccountId:      form.CloudAccountId,
		Provider:            provider,
		AccountId:           account.AccountId,
		Regions:             models.StrSlice(regions),
		AssetTypes:          models.StrSlice(assetTypes),
		Status:              status,
		SyncInterval:        cloudSyncPolicyIntervalSeconds(form.SyncInterval),
		MaxRetryAttempts:    cloudSyncPolicyRetryAttempts(form.MaxRetryAttempts),
		RetryBackoffSeconds: cloudSyncPolicyRetryBackoffSeconds(form.RetryBackoffSeconds),
		NotifyOnFailure:     form.NotifyOnFailure,
		AutoPauseOnFailure:  form.AutoPauseOnFailure,
		NextSyncAt:          nextSyncAt,
		Params:              cloudProviderSanitizeAttrs(params),
	}, nil
}

func getCloudSyncPolicy(c *ctx.ServiceContext, id models.Id) (*models.CloudSyncPolicy, e.Error) {
	policy := models.CloudSyncPolicy{}
	if err := c.DB().Model(&models.CloudSyncPolicy{}).
		Where("id = ? and org_id = ?", id, c.OrgId).
		First(&policy); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	return &policy, nil
}

func cloudSyncPolicyResp(c *ctx.ServiceContext, policy models.CloudSyncPolicy) resps.CloudSyncPolicyResp {
	resp := resps.CloudSyncPolicyResp{CloudSyncPolicy: policy}
	resp.Protection = cloudSyncPolicyProtectionResp(c, &policy)
	resp.Schedules = cloudSyncPolicySchedulesResp(&policy, time.Now())
	if policy.CloudAccountId == "" {
		return resp
	}
	account, err := findCmdbCloudAccount(c, models.CmdbCloudAccountSourceCloudAccount, policy.CloudAccountId)
	if err != nil {
		return resp
	}
	resp.CloudAccountName = account.Name
	resp.CloudAccountReady = account.Ready
	resp.MissingCredentialKeys = account.MissingCredentialKeys
	resp.SupportedAssetTypes = account.SupportedAssetTypes
	return resp
}

func cloudSyncPolicyApplyFailure(policy *models.CloudSyncPolicy, attrs models.Attrs, failedAt time.Time, reason string) {
	failureCount := 1
	if policy != nil {
		failureCount = policy.FailureCount + 1
	}
	attrs["failure_count"] = failureCount
	attrs["last_failure_at"] = models.Time(failedAt)
	attrs["last_failure_reason"] = reason
	maxRetryAttempts := cloudSyncPolicyRetryAttempts(0)
	retryBackoffSeconds := int(cloudSyncPolicyDefaultRetryBackoff / time.Second)
	autoPauseOnFailure := false
	syncInterval := int(cloudSyncPolicyDefaultInterval / time.Second)
	if policy != nil {
		maxRetryAttempts = policy.MaxRetryAttempts
		retryBackoffSeconds = policy.RetryBackoffSeconds
		autoPauseOnFailure = policy.AutoPauseOnFailure
		syncInterval = policy.SyncInterval
	}
	if maxRetryAttempts > 0 && failureCount <= maxRetryAttempts {
		attrs["next_sync_at"] = models.Time(failedAt.Add(time.Duration(cloudSyncPolicyFailureBackoffSeconds(retryBackoffSeconds, failureCount)) * time.Second))
		return
	}
	if autoPauseOnFailure && maxRetryAttempts > 0 && failureCount > maxRetryAttempts {
		attrs["status"] = models.CloudSyncPolicyStatusDisabled
		attrs["next_sync_at"] = nil
		return
	}
	attrs["next_sync_at"] = models.Time(failedAt.Add(time.Duration(cloudSyncPolicyIntervalSeconds(syncInterval)) * time.Second))
}

func cloudSyncPolicyEvent(c *ctx.ServiceContext, policy *models.CloudSyncPolicy, eventType, level, status, title, message string, payload models.ResAttrs) {
	if policy == nil {
		return
	}
	if payload == nil {
		payload = models.ResAttrs{}
	}
	if _, ok := payload["policyId"]; !ok {
		payload["policyId"] = policy.Id.String()
	}
	cloudSyncPolicyApplyNotificationRouting(policy, payload)
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:          policy.OrgId,
		CloudAccountId: policy.CloudAccountId,
		Source:         models.CloudEventSourceSync,
		EventType:      eventType,
		Level:          level,
		Status:         status,
		Provider:       policy.Provider,
		AccountId:      policy.AccountId,
		ResourceType:   "cloud_sync_policy",
		ResourceId:     policy.Id.String(),
		ResourceName:   policy.Name,
		Title:          title,
		Message:        message,
		Payload:        payload,
	})
}

func cloudSyncPolicyFailureEvent(c *ctx.ServiceContext, policy *models.CloudSyncPolicy, taskId models.Id, reason string, stats models.ResAttrs) {
	if policy == nil || !policy.NotifyOnFailure {
		return
	}
	eventType := "cloud.sync.policy.failed"
	title := "云资产同步策略执行失败"
	level := models.CloudEventLevelError
	status := models.CmdbSyncTaskFailed
	autoPaused := policy.AutoPauseOnFailure && policy.MaxRetryAttempts > 0 && policy.FailureCount+1 > policy.MaxRetryAttempts
	if autoPaused {
		eventType = "cloud.sync.policy.auto_paused"
		title = "云资产同步策略已自动暂停"
		status = models.CloudSyncPolicyStatusDisabled
	}
	payload := cloudSyncPolicyApplyNotificationRoutingWithContext(policy, models.ResAttrs{
		"policyId":            policy.Id.String(),
		"taskId":              taskId.String(),
		"failureCount":        policy.FailureCount + 1,
		"maxRetryAttempts":    policy.MaxRetryAttempts,
		"retryBackoffSeconds": policy.RetryBackoffSeconds,
		"autoPauseOnFailure":  policy.AutoPauseOnFailure,
	}, reason, stats, autoPaused)
	cloudSyncPolicyEvent(c, policy, eventType, level, status, title, reason, payload)
}

func cloudSyncPolicyApplyNotificationRoutingById(c *ctx.ServiceContext, policyId models.Id, payload models.ResAttrs) models.ResAttrs {
	if payload == nil {
		payload = models.ResAttrs{}
	}
	if c == nil || policyId == "" {
		return payload
	}
	policy, err := getCloudSyncPolicy(c, policyId)
	if err != nil {
		return payload
	}
	return cloudSyncPolicyApplyNotificationRouting(policy, payload)
}

func cloudSyncPolicyApplyNotificationRouting(policy *models.CloudSyncPolicy, payload models.ResAttrs) models.ResAttrs {
	return cloudSyncPolicyApplyNotificationRoutingWithContext(policy, payload, "", nil, false)
}

func cloudSyncPolicyApplyNotificationRoutingWithContext(policy *models.CloudSyncPolicy, payload models.ResAttrs, reason string, stats models.ResAttrs, autoPaused bool) models.ResAttrs {
	if payload == nil {
		payload = models.ResAttrs{}
	}
	if policy != nil && policy.Id != "" {
		payload["notificationSource"] = "cloud_sync_policy"
	}
	failureCategory := cloudSyncPolicyFailureCategory(reason, stats)
	if failureCategory == "" {
		failureCategory = cloudSyncPolicyNormalizeFailureCategory(cloudSyncPolicyAttrString(payload["failureCategory"]))
	}
	if failureCategory != "" {
		payload["failureCategory"] = failureCategory
	}
	cloudService := cloudSyncPolicyNotificationService(payload, reason, stats)
	if cloudService != "" {
		payload["cloudService"] = cloudService
	}
	if owner := strings.TrimSpace(cloudSyncPolicyAttrString(policyParam(policy, "notificationOwner"))); owner != "" {
		payload["notificationOwner"] = owner
	}
	routes := cloudSyncPolicyNormalizeRoutingValues(payload["notificationRoutes"])
	if policy != nil {
		routes = append(routes, cloudSyncPolicyNormalizeRoutingValues(policy.Params["notificationRoutes"])...)
		if failureRoutes := cloudSyncPolicyNotificationFailureRoutesForCategory(policy.Params, failureCategory); len(failureRoutes) > 0 {
			routes = append(routes, failureRoutes...)
			payload["notificationFailureRoutes"] = failureRoutes
		}
		if serviceRoutes := cloudSyncPolicyNotificationServiceRoutesForService(policy.Params, cloudService); len(serviceRoutes) > 0 {
			routes = append(routes, serviceRoutes...)
			payload["notificationServiceRoutes"] = serviceRoutes
		}
		escalated, escalationReason := cloudSyncPolicyNotificationEscalated(policy.Params, cloudSyncPolicyAttrInt(payload["failureCount"]), autoPaused)
		if escalated {
			payload["notificationEscalated"] = true
			payload["notificationEscalationReason"] = escalationReason
			if escalationAt := cloudSyncPolicyNotificationEscalationAt(policy.Params); escalationAt > 0 {
				payload["notificationEscalationAt"] = escalationAt
			}
			if escalationRoutes := cloudSyncPolicyNormalizeRoutingValues(policy.Params["notificationEscalationRoutes"]); len(escalationRoutes) > 0 {
				routes = append(routes, escalationRoutes...)
				payload["notificationEscalationRoutes"] = escalationRoutes
			}
		}
		assignees := append(cloudSyncPolicyNormalizeRoutingValues(payload["notificationAssignees"]), cloudSyncPolicyNormalizeRoutingValues(policy.Params["notificationAssignees"])...)
		if len(assignees) > 0 {
			payload["notificationAssignees"] = dedupeStrings(assignees)
		}
		if cloudSyncPolicyAttrBool(policy.Params["itsmAutoTicket"]) {
			payload["itsmAutoTicket"] = true
		}
		if connectorIds := cloudSyncPolicyNormalizeRoutingValues(policy.Params["itsmConnectorIds"]); len(connectorIds) > 0 {
			payload["itsmConnectorIds"] = connectorIds
		}
		if priority := strings.TrimSpace(cloudSyncPolicyAttrString(policy.Params["itsmPriority"])); priority != "" {
			payload["itsmPriority"] = priority
		}
	}
	if len(routes) > 0 {
		payload["notificationRoutes"] = dedupeStrings(routes)
	}
	return payload
}

func cloudSyncPolicyNotificationRoutingAttrs(policy *models.CloudSyncPolicy) models.ResAttrs {
	if policy == nil {
		return nil
	}
	attrs := models.ResAttrs{}
	owner := strings.TrimSpace(cloudSyncPolicyAttrString(policy.Params["notificationOwner"]))
	if owner != "" {
		attrs["notificationOwner"] = owner
	}
	if routes := cloudSyncPolicyNormalizeRoutingValues(policy.Params["notificationRoutes"]); len(routes) > 0 {
		attrs["notificationRoutes"] = routes
	}
	if assignees := cloudSyncPolicyNormalizeRoutingValues(policy.Params["notificationAssignees"]); len(assignees) > 0 {
		attrs["notificationAssignees"] = assignees
	}
	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

func policyParam(policy *models.CloudSyncPolicy, key string) interface{} {
	if policy == nil || policy.Params == nil {
		return nil
	}
	return policy.Params[key]
}

func firstNonNil(values ...interface{}) interface{} {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func cloudSyncPolicyNotificationRouteMap(value interface{}, normalizeKey func(string) string) models.ResAttrs {
	raw := cloudSyncPolicyNotificationRouteRawMap(value)
	result := models.ResAttrs{}
	for key, value := range raw {
		routeKey := strings.TrimSpace(key)
		if normalizeKey != nil {
			routeKey = normalizeKey(routeKey)
		}
		if routeKey == "" {
			continue
		}
		routes := cloudSyncPolicyNormalizeRoutingValues(value)
		if len(routes) == 0 {
			continue
		}
		result[routeKey] = routes
	}
	return result
}

func cloudSyncPolicyNotificationRouteRawMap(value interface{}) models.ResAttrs {
	switch typed := value.(type) {
	case string:
		return cloudSyncPolicyParseRouteText(typed)
	default:
		return cloudSyncPolicyAttrMap(value)
	}
}

func cloudSyncPolicyParseRouteText(value string) models.ResAttrs {
	result := models.ResAttrs{}
	lines := strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == ';'
	})
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.IndexAny(line, "=:")
		if idx <= 0 || idx >= len(line)-1 {
			continue
		}
		result[strings.TrimSpace(line[:idx])] = strings.TrimSpace(line[idx+1:])
	}
	return result
}

func cloudSyncPolicyNormalizeFailureCategory(value string) string {
	switch cloudEventNotificationRouteKey(value) {
	case "rate_limit", "rate-limit", "ratelimit", "throttle", "throttling":
		return "rate_limit"
	case "credential", "credentials", "auth", "authentication", "token", "secret":
		return "credential"
	case "permission", "permissions", "forbidden", "denied", "authorization", "unauthorized":
		return "permission"
	case "configuration", "config", "invalid", "unsupported", "bad_request", "bad-request":
		return "configuration"
	case "network", "timeout", "temporary", "unavailable", "connection":
		return "network"
	case "unknown", "other", "default":
		return "unknown"
	default:
		return ""
	}
}

func cloudSyncPolicyFailureCategory(reason string, stats models.ResAttrs) string {
	for _, detail := range cmdbSyncTaskMetricAttrsList(stats["failureDetails"]) {
		if category := cloudSyncPolicyNormalizeFailureCategory(cmdbSyncTaskMetricString(detail["category"])); category != "" {
			return category
		}
		if message := cmdbSyncTaskMetricString(detail["message"]); message != "" {
			category, _, _ := cmdbSyncClassifyFailure(message)
			if category = cloudSyncPolicyNormalizeFailureCategory(category); category != "" {
				return category
			}
		}
	}
	if strings.TrimSpace(reason) == "" {
		return ""
	}
	category, _, _ := cmdbSyncClassifyFailure(reason)
	return cloudSyncPolicyNormalizeFailureCategory(category)
}

func cloudSyncPolicyNotificationService(payload models.ResAttrs, reason string, stats models.ResAttrs) string {
	for _, key := range []string{"cloudService", "providerService", "service"} {
		if service := cloudEventNotificationRouteKey(cloudSyncPolicyAttrString(payload[key])); service != "" {
			return service
		}
	}
	for _, key := range []string{"slowest", "top"} {
		if service := cloudSyncPolicyServiceFromPayloadValue(payload[key]); service != "" {
			return service
		}
	}
	for _, detail := range cmdbSyncTaskMetricAttrsList(stats["failureDetails"]) {
		for _, key := range []string{"cloudService", "providerService", "service"} {
			if service := cloudEventNotificationRouteKey(cmdbSyncTaskMetricString(detail[key])); service != "" {
				return service
			}
		}
	}
	metadata := cmdbSyncProviderFailureMetadata(reason)
	if service := cloudEventNotificationRouteKey(cloudSyncPolicyAttrString(metadata["providerService"])); service != "" {
		return service
	}
	return ""
}

func cloudSyncPolicyServiceFromPayloadValue(value interface{}) string {
	for _, item := range cmdbSyncTaskMetricAttrsList(value) {
		for _, key := range []string{"cloudService", "providerService", "service"} {
			if service := cloudEventNotificationRouteKey(cmdbSyncTaskMetricString(item[key])); service != "" {
				return service
			}
		}
	}
	attrs := cloudSyncPolicyAttrMap(value)
	for _, key := range []string{"cloudService", "providerService", "service"} {
		if service := cloudEventNotificationRouteKey(cloudSyncPolicyAttrString(attrs[key])); service != "" {
			return service
		}
	}
	return ""
}

func cloudSyncPolicyNotificationFailureRoutesForCategory(params models.ResAttrs, category string) []string {
	category = cloudSyncPolicyNormalizeFailureCategory(category)
	if category == "" {
		return nil
	}
	routes := cloudSyncPolicyNotificationRouteMap(params["notificationFailureRoutes"], cloudSyncPolicyNormalizeFailureCategory)
	return cloudSyncPolicyNormalizeRoutingValues(routes[category])
}

func cloudSyncPolicyNotificationServiceRoutesForService(params models.ResAttrs, service string) []string {
	service = cloudEventNotificationRouteKey(service)
	if service == "" {
		return nil
	}
	routes := cloudSyncPolicyNotificationRouteMap(params["notificationServiceRoutes"], cloudEventNotificationRouteKey)
	return cloudSyncPolicyNormalizeRoutingValues(routes[service])
}

func cloudSyncPolicyNotificationEscalationAt(params models.ResAttrs) int {
	if params == nil {
		return 0
	}
	value := cloudSyncPolicyAttrInt(params["notificationEscalationAt"])
	if value <= 0 {
		return 0
	}
	if value > cloudSyncPolicyMaxEscalationAt {
		return cloudSyncPolicyMaxEscalationAt
	}
	return value
}

func cloudSyncPolicyNotificationEscalated(params models.ResAttrs, failureCount int, autoPaused bool) (bool, string) {
	if autoPaused {
		return true, "auto_paused"
	}
	escalationAt := cloudSyncPolicyNotificationEscalationAt(params)
	if escalationAt > 0 && failureCount >= escalationAt {
		return true, "failure_count"
	}
	return false, ""
}

func cloudSyncPolicyLockName(orgId models.Id, policyId models.Id, scopeKey string) string {
	source := fmt.Sprintf("%s:%s:%s", orgId.String(), policyId.String(), scopeKey)
	sum := sha1.Sum([]byte(source))
	return fmt.Sprintf("csp:%s:%x", policyId.String(), sum[:4])
}

func cloudSyncPolicyIntervalSeconds(value int) int {
	if value <= 0 {
		return int(cloudSyncPolicyDefaultInterval / time.Second)
	}
	min := int(cloudSyncPolicyMinInterval / time.Second)
	max := int(cloudSyncPolicyMaxInterval / time.Second)
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func cloudSyncPolicyRetryAttempts(value int) int {
	if value <= 0 {
		return cloudSyncPolicyDefaultRetryAttempts
	}
	if value > cloudSyncPolicyMaxRetryAttempts {
		return cloudSyncPolicyMaxRetryAttempts
	}
	return value
}

func cloudSyncPolicyRetryBackoffSeconds(value int) int {
	if value <= 0 {
		return int(cloudSyncPolicyDefaultRetryBackoff / time.Second)
	}
	min := int(cloudSyncPolicyMinInterval / time.Second)
	max := int(cloudSyncPolicyMaxRetryBackoff / time.Second)
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func cloudSyncPolicyFailureBackoffSeconds(baseSeconds int, failureCount int) int {
	seconds := cloudSyncPolicyRetryBackoffSeconds(baseSeconds)
	for i := 1; i < failureCount; i++ {
		seconds *= 2
		max := int(cloudSyncPolicyMaxRetryBackoff / time.Second)
		if seconds > max {
			return max
		}
	}
	return seconds
}

func cloudSyncPolicyWorkerInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv(cloudSyncPolicyWorkerIntervalEnv))
	if raw == "" {
		return cloudSyncPolicyWorkerDefaultInterval
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return cloudSyncPolicyWorkerDefaultInterval
	}
	interval := time.Duration(seconds) * time.Second
	if interval < cloudSyncPolicyWorkerMinInterval {
		return cloudSyncPolicyWorkerMinInterval
	}
	if interval > cloudSyncPolicyWorkerMaxInterval {
		return cloudSyncPolicyWorkerMaxInterval
	}
	return interval
}

func cloudSyncPolicyScheduleProtection(c *ctx.ServiceContext, policy *models.CloudSyncPolicy, triggeredInRun int) (string, bool) {
	if c == nil || policy == nil {
		return "", false
	}
	if reason := cloudSyncPolicyPauseReason(policy, time.Now()); reason != "" {
		return reason, true
	}
	maxRunsPerDay := cloudSyncPolicyMaxRunsPerDay(policy)
	if maxRunsPerDay > 0 {
		runsToday, err := cloudSyncPolicyRunsToday(c, policy)
		if err != nil {
			logs.Get().
				WithField("policyId", policy.Id.String()).
				Warnf("count cloud sync policy daily runs failed: %v", err)
		} else if runsToday >= maxRunsPerDay {
			return fmt.Sprintf("今日同步次数 %d 已达到策略上限 %d", runsToday, maxRunsPerDay), true
		}
	}
	maxConcurrentRunning := cloudSyncPolicyMaxConcurrentRunning()
	if maxConcurrentRunning > 0 {
		runningCount, err := cloudSyncPolicyRunningTaskCount(c)
		if err != nil {
			logs.Get().
				WithField("policyId", policy.Id.String()).
				Warnf("count running cloud sync tasks failed: %v", err)
		} else if runningCount >= maxConcurrentRunning {
			return fmt.Sprintf("运行中采集任务 %d 已达到全局并发上限 %d", runningCount, maxConcurrentRunning), true
		}
	}
	maxTriggeredPerRun := cloudSyncPolicyMaxTriggeredPerRun()
	if maxTriggeredPerRun > 0 && triggeredInRun >= maxTriggeredPerRun {
		return fmt.Sprintf("本轮触发数 %d 已达到全局上限 %d", triggeredInRun, maxTriggeredPerRun), true
	}
	return "", false
}

func cloudSyncPolicyProtectionResp(c *ctx.ServiceContext, policy *models.CloudSyncPolicy) resps.CloudSyncPolicyProtectionResp {
	resp := resps.CloudSyncPolicyProtectionResp{
		PauseWindows:         cloudSyncPolicyPauseWindows(policy),
		MaxRunsPerDay:        cloudSyncPolicyMaxRunsPerDay(policy),
		MaxTriggeredPerRun:   cloudSyncPolicyMaxTriggeredPerRun(),
		MaxConcurrentRunning: cloudSyncPolicyMaxConcurrentRunning(),
	}
	if reason := cloudSyncPolicyPauseReason(policy, time.Now()); reason != "" {
		resp.PausedNow = true
		resp.PauseReason = reason
	}
	if c == nil || policy == nil {
		return resp
	}
	if runsToday, err := cloudSyncPolicyRunsToday(c, policy); err == nil {
		resp.RunsToday = runsToday
	}
	if runningCount, err := cloudSyncPolicyRunningTaskCount(c); err == nil {
		resp.RunningCount = runningCount
	}
	return resp
}

func cloudSyncPolicyNormalizeParams(params models.ResAttrs) models.ResAttrs {
	normalized := models.ResAttrs{}
	for key, value := range params {
		normalized[key] = value
	}
	pauseWindows := cloudSyncPolicyNormalizePauseWindows(normalized["pauseWindows"])
	if len(pauseWindows) > 0 {
		normalized["pauseWindows"] = pauseWindows
	} else {
		delete(normalized, "pauseWindows")
	}
	maxRunsPerDay := cloudSyncPolicyAttrInt(normalized["maxRunsPerDay"])
	if maxRunsPerDay > 0 {
		normalized["maxRunsPerDay"] = maxRunsPerDay
	} else {
		delete(normalized, "maxRunsPerDay")
	}
	slowAPIThresholdMs := cloudSyncPolicyAttrInt(normalized["slowApiThresholdMs"])
	if slowAPIThresholdMs > 0 {
		normalized["slowApiThresholdMs"] = cmdbSyncTaskEffectiveSlowAPIThresholdMs(int64(slowAPIThresholdMs))
	} else {
		delete(normalized, "slowApiThresholdMs")
	}
	slowAPISilenceMinutes := cloudSyncPolicyAttrInt(normalized["slowApiSilenceMinutes"])
	if slowAPISilenceMinutes > 0 {
		normalized["slowApiSilenceMinutes"] = cmdbSyncTaskEffectiveSlowAPISilenceMinutes(int64(slowAPISilenceMinutes))
	} else {
		delete(normalized, "slowApiSilenceMinutes")
	}
	owner := strings.TrimSpace(cloudSyncPolicyAttrString(normalized["notificationOwner"]))
	if owner != "" {
		normalized["notificationOwner"] = owner
	} else {
		delete(normalized, "notificationOwner")
	}
	routes := cloudSyncPolicyNormalizeRoutingValues(normalized["notificationRoutes"])
	if len(routes) > 0 {
		normalized["notificationRoutes"] = routes
	} else {
		delete(normalized, "notificationRoutes")
	}
	assignees := cloudSyncPolicyNormalizeRoutingValues(normalized["notificationAssignees"])
	if len(assignees) > 0 {
		normalized["notificationAssignees"] = assignees
	} else {
		delete(normalized, "notificationAssignees")
	}
	failureRoutes := cloudSyncPolicyNotificationRouteMap(normalized["notificationFailureRoutes"], cloudSyncPolicyNormalizeFailureCategory)
	if len(failureRoutes) > 0 {
		normalized["notificationFailureRoutes"] = failureRoutes
	} else {
		delete(normalized, "notificationFailureRoutes")
	}
	serviceRoutes := cloudSyncPolicyNotificationRouteMap(normalized["notificationServiceRoutes"], cloudEventNotificationRouteKey)
	if len(serviceRoutes) > 0 {
		normalized["notificationServiceRoutes"] = serviceRoutes
	} else {
		delete(normalized, "notificationServiceRoutes")
	}
	escalationAt := cloudSyncPolicyNotificationEscalationAt(normalized)
	if escalationAt > 0 {
		normalized["notificationEscalationAt"] = escalationAt
	} else {
		delete(normalized, "notificationEscalationAt")
	}
	escalationRoutes := cloudSyncPolicyNormalizeRoutingValues(normalized["notificationEscalationRoutes"])
	if len(escalationRoutes) > 0 {
		normalized["notificationEscalationRoutes"] = escalationRoutes
	} else {
		delete(normalized, "notificationEscalationRoutes")
	}
	if cloudSyncPolicyAttrBool(normalized["itsmAutoTicket"]) {
		normalized["itsmAutoTicket"] = true
	} else {
		delete(normalized, "itsmAutoTicket")
	}
	itsmConnectorIds := cloudSyncPolicyNormalizeRoutingValues(firstNonNil(normalized["itsmConnectorIds"], normalized["itsmConnectorId"]))
	if len(itsmConnectorIds) > 0 {
		normalized["itsmConnectorIds"] = itsmConnectorIds
		delete(normalized, "itsmConnectorId")
	} else {
		delete(normalized, "itsmConnectorIds")
		delete(normalized, "itsmConnectorId")
	}
	if priority := strings.TrimSpace(cloudSyncPolicyAttrString(normalized["itsmPriority"])); priority != "" {
		normalized["itsmPriority"] = cloudSyncPolicyTruncateString(priority, 32)
	} else {
		delete(normalized, "itsmPriority")
	}
	autoRetryFailedScopes := cloudSyncPolicyAttrBool(normalized["autoRetryFailedScopes"])
	if autoRetryFailedScopes {
		normalized["autoRetryFailedScopes"] = true
		maxScopes := cloudSyncPolicyAttrInt(normalized["autoRetryMaxScopes"])
		if maxScopes <= 0 {
			maxScopes = cloudSyncPolicyDefaultAutoRetryScopes
		}
		normalized["autoRetryMaxScopes"] = cloudSyncPolicyAutoRetryScopeLimit(maxScopes)
	} else {
		delete(normalized, "autoRetryFailedScopes")
		delete(normalized, "autoRetryMaxScopes")
	}
	schedules := cloudSyncPolicyNormalizeScheduleOverrides(normalized["scheduleOverrides"])
	if len(schedules) > 0 {
		normalized["scheduleOverrides"] = schedules
		normalized["scheduleState"] = cloudSyncPolicyNormalizeScheduleStates(normalized["scheduleState"], schedules)
	} else {
		delete(normalized, "scheduleOverrides")
		delete(normalized, "scheduleState")
	}
	return normalized
}

func cloudSyncPolicySlowAPIThresholdMs(policy *models.CloudSyncPolicy) int64 {
	if policy == nil {
		return cmdbSyncTaskSlowAPIThresholdMs
	}
	return cmdbSyncTaskEffectiveSlowAPIThresholdMs(cloudSyncPolicyAttrInt64(policy.Params["slowApiThresholdMs"]))
}

func cloudSyncPolicySlowAPISilenceMinutes(policy *models.CloudSyncPolicy) int64 {
	if policy == nil {
		return 0
	}
	return cmdbSyncTaskEffectiveSlowAPISilenceMinutes(cloudSyncPolicyAttrInt64(policy.Params["slowApiSilenceMinutes"]))
}

func cloudSyncPolicyAutoRetryFailedScopes(policy *models.CloudSyncPolicy) bool {
	if policy == nil {
		return false
	}
	return cloudSyncPolicyAttrBool(policy.Params["autoRetryFailedScopes"])
}

func cloudSyncPolicyAutoRetryMaxScopes(policy *models.CloudSyncPolicy) int {
	if policy == nil || !cloudSyncPolicyAutoRetryFailedScopes(policy) {
		return 0
	}
	return cloudSyncPolicyAutoRetryScopeLimit(cloudSyncPolicyAttrInt(policy.Params["autoRetryMaxScopes"]))
}

func cloudSyncPolicyAutoRetryScopeLimit(value int) int {
	if value <= 0 {
		return cloudSyncPolicyDefaultAutoRetryScopes
	}
	if value > cloudSyncPolicyMaxAutoRetryScopes {
		return cloudSyncPolicyMaxAutoRetryScopes
	}
	return value
}

func cloudSyncPolicyNormalizeScheduleOverrides(value interface{}) []models.ResAttrs {
	items := cloudSyncPolicyAttrItems(value)
	result := make([]models.ResAttrs, 0, len(items))
	for _, item := range items {
		attrs := cloudSyncPolicyAttrMap(item)
		regions := normalizeStringList(cloudSyncPolicyAttrStringSlice(attrs["regions"]))
		assetTypes := normalizeStringList(cloudSyncPolicyAttrStringSlice(attrs["assetTypes"]))
		if len(regions) == 0 && len(assetTypes) == 0 {
			continue
		}
		interval := cloudSyncPolicyIntervalSeconds(cloudSyncPolicyAttrInt(attrs["syncInterval"]))
		name := cloudSyncPolicyAttrString(attrs["name"])
		key := cloudSyncPolicyScheduleKey(regions, assetTypes, interval)
		if name == "" {
			name = fmt.Sprintf("%s / %s", joinOrDefault(regions, "全部区域"), joinOrDefault(assetTypes, "全部类型"))
		}
		result = append(result, models.ResAttrs{
			"key":          key,
			"name":         name,
			"regions":      regions,
			"assetTypes":   assetTypes,
			"syncInterval": interval,
		})
	}
	return result
}

func cloudSyncPolicyNormalizeScheduleStates(value interface{}, schedules []models.ResAttrs) models.ResAttrs {
	existing := cloudSyncPolicyDecodeScheduleStates(value)
	result := models.ResAttrs{}
	for _, schedule := range schedules {
		key := strings.TrimSpace(fmt.Sprint(schedule["key"]))
		if key == "" {
			continue
		}
		state := existing[key]
		result[key] = cloudSyncPolicyEncodeScheduleState(state)
	}
	return result
}

func cloudSyncPolicyScheduleOverrides(policy *models.CloudSyncPolicy) []cloudSyncPolicyScheduleOverride {
	if policy == nil {
		return nil
	}
	return cloudSyncPolicyDecodeScheduleOverrides(policy.Params["scheduleOverrides"])
}

func cloudSyncPolicyDecodeScheduleOverrides(value interface{}) []cloudSyncPolicyScheduleOverride {
	items := cloudSyncPolicyAttrItems(value)
	result := make([]cloudSyncPolicyScheduleOverride, 0, len(items))
	for _, item := range items {
		attrs := cloudSyncPolicyAttrMap(item)
		regions := normalizeStringList(cloudSyncPolicyAttrStringSlice(attrs["regions"]))
		assetTypes := normalizeStringList(cloudSyncPolicyAttrStringSlice(attrs["assetTypes"]))
		if len(regions) == 0 && len(assetTypes) == 0 {
			continue
		}
		interval := cloudSyncPolicyIntervalSeconds(cloudSyncPolicyAttrInt(attrs["syncInterval"]))
		key := cloudSyncPolicyAttrString(attrs["key"])
		if key == "" {
			key = cloudSyncPolicyScheduleKey(regions, assetTypes, interval)
		}
		name := cloudSyncPolicyAttrString(attrs["name"])
		if name == "" {
			name = fmt.Sprintf("%s / %s", joinOrDefault(regions, "全部区域"), joinOrDefault(assetTypes, "全部类型"))
		}
		result = append(result, cloudSyncPolicyScheduleOverride{
			Key:          key,
			Name:         name,
			Regions:      regions,
			AssetTypes:   assetTypes,
			SyncInterval: interval,
		})
	}
	return result
}

func cloudSyncPolicyDueRunScopes(policy *models.CloudSyncPolicy, now time.Time, force bool) []cloudSyncPolicyRunScope {
	if policy == nil {
		return nil
	}
	schedules := cloudSyncPolicyScheduleOverrides(policy)
	if len(schedules) == 0 {
		return []cloudSyncPolicyRunScope{{
			cloudSyncPolicyScheduleOverride: cloudSyncPolicyScheduleOverride{
				Regions:      []string(policy.Regions),
				AssetTypes:   []string(policy.AssetTypes),
				SyncInterval: policy.SyncInterval,
			},
			Default: true,
		}}
	}
	states := cloudSyncPolicyScheduleStates(policy)
	scopes := make([]cloudSyncPolicyRunScope, 0, len(schedules))
	for _, schedule := range schedules {
		next := cloudSyncPolicyScheduleNextSyncAt(states[schedule.Key])
		if next.IsZero() || force || !next.After(now) {
			if len(schedule.Regions) == 0 {
				schedule.Regions = []string(policy.Regions)
			}
			if len(schedule.AssetTypes) == 0 {
				schedule.AssetTypes = []string(policy.AssetTypes)
			}
			scopes = append(scopes, cloudSyncPolicyRunScope{
				cloudSyncPolicyScheduleOverride: schedule,
			})
		}
	}
	return scopes
}

func cloudSyncPolicySchedulesResp(policy *models.CloudSyncPolicy, now time.Time) []resps.CloudSyncPolicyScheduleResp {
	schedules := cloudSyncPolicyScheduleOverrides(policy)
	if len(schedules) == 0 {
		return []resps.CloudSyncPolicyScheduleResp{}
	}
	states := cloudSyncPolicyScheduleStates(policy)
	result := make([]resps.CloudSyncPolicyScheduleResp, 0, len(schedules))
	for _, schedule := range schedules {
		state := states[schedule.Key]
		next := cloudSyncPolicyScheduleNextSyncAt(state)
		result = append(result, resps.CloudSyncPolicyScheduleResp{
			Key:            schedule.Key,
			Name:           schedule.Name,
			Regions:        schedule.Regions,
			AssetTypes:     schedule.AssetTypes,
			SyncInterval:   schedule.SyncInterval,
			NextSyncAt:     state.NextSyncAt,
			LastSyncedAt:   state.LastSyncedAt,
			LastSyncTaskId: state.LastSyncTaskId,
			LastSyncStatus: state.LastSyncStatus,
			LastError:      state.LastError,
			FailureCount:   state.FailureCount,
			DueNow:         next.IsZero() || !next.After(now),
		})
	}
	return result
}

func cloudSyncPolicyScheduleStates(policy *models.CloudSyncPolicy) map[string]cloudSyncPolicyScheduleState {
	if policy == nil {
		return map[string]cloudSyncPolicyScheduleState{}
	}
	return cloudSyncPolicyDecodeScheduleStates(policy.Params["scheduleState"])
}

func cloudSyncPolicyDecodeScheduleStates(value interface{}) map[string]cloudSyncPolicyScheduleState {
	result := map[string]cloudSyncPolicyScheduleState{}
	attrs := cloudSyncPolicyAttrMap(value)
	for key, value := range attrs {
		stateAttrs := cloudSyncPolicyAttrMap(value)
		result[key] = cloudSyncPolicyScheduleState{
			NextSyncAt:      cloudSyncPolicyAttrString(stateAttrs["nextSyncAt"]),
			LastSyncedAt:    cloudSyncPolicyAttrString(stateAttrs["lastSyncedAt"]),
			LastSyncTaskId:  cloudSyncPolicyAttrString(stateAttrs["lastSyncTaskId"]),
			LastSyncStatus:  cloudSyncPolicyAttrString(stateAttrs["lastSyncStatus"]),
			LastError:       cloudSyncPolicyAttrString(stateAttrs["lastError"]),
			FailureCount:    cloudSyncPolicyAttrInt(stateAttrs["failureCount"]),
			LastFailureAt:   cloudSyncPolicyAttrString(stateAttrs["lastFailureAt"]),
			LastFailureText: cloudSyncPolicyAttrString(stateAttrs["lastFailureText"]),
		}
	}
	return result
}

func cloudSyncPolicyWithScheduleState(policy *models.CloudSyncPolicy, key string, state cloudSyncPolicyScheduleState) models.ResAttrs {
	params := models.ResAttrs{}
	if policy != nil {
		for paramKey, value := range policy.Params {
			params[paramKey] = value
		}
	}
	states := cloudSyncPolicyDecodeScheduleStates(params["scheduleState"])
	existing := states[key]
	state = cloudSyncPolicyMergeScheduleState(existing, state)
	if state.NextSyncAt == "" {
		state.NextSyncAt = existing.NextSyncAt
	}
	if state.LastSyncedAt == "" {
		state.LastSyncedAt = existing.LastSyncedAt
	}
	if state.LastSyncTaskId == "" {
		state.LastSyncTaskId = existing.LastSyncTaskId
	}
	states[key] = state
	params["scheduleState"] = cloudSyncPolicyEncodeScheduleStates(states)
	return cloudSyncPolicyNormalizeParams(params)
}

func cloudSyncPolicyMergeScheduleState(existing cloudSyncPolicyScheduleState, state cloudSyncPolicyScheduleState) cloudSyncPolicyScheduleState {
	if state.LastSyncStatus == models.CmdbSyncTaskRunning &&
		state.LastSyncTaskId != "" &&
		state.LastSyncTaskId == existing.LastSyncTaskId &&
		cloudSyncPolicyTerminalTaskStatus(existing.LastSyncStatus) {
		return existing
	}
	if state.LastSyncStatus == "" {
		state.LastSyncStatus = existing.LastSyncStatus
	}
	return state
}

func cloudSyncPolicyTerminalTaskStatus(status string) bool {
	return status == models.CmdbSyncTaskComplete || status == models.CmdbSyncTaskFailed
}

func cloudSyncPolicyEncodeScheduleStates(states map[string]cloudSyncPolicyScheduleState) models.ResAttrs {
	result := models.ResAttrs{}
	for key, state := range states {
		result[key] = cloudSyncPolicyEncodeScheduleState(state)
	}
	return result
}

func cloudSyncPolicyEncodeScheduleState(state cloudSyncPolicyScheduleState) models.ResAttrs {
	return models.ResAttrs{
		"nextSyncAt":      state.NextSyncAt,
		"lastSyncedAt":    state.LastSyncedAt,
		"lastSyncTaskId":  state.LastSyncTaskId,
		"lastSyncStatus":  state.LastSyncStatus,
		"lastError":       state.LastError,
		"failureCount":    state.FailureCount,
		"lastFailureAt":   state.LastFailureAt,
		"lastFailureText": state.LastFailureText,
	}
}

func cloudSyncPolicyNextSyncAtFromParams(policy *models.CloudSyncPolicy, params models.ResAttrs) models.Time {
	if len(cloudSyncPolicyDecodeScheduleOverrides(params["scheduleOverrides"])) == 0 {
		if policy == nil {
			return models.Time(time.Now())
		}
		return policy.NextSyncAt
	}
	states := cloudSyncPolicyDecodeScheduleStates(params["scheduleState"])
	now := time.Now()
	next := time.Time{}
	for _, schedule := range cloudSyncPolicyDecodeScheduleOverrides(params["scheduleOverrides"]) {
		stateNext := cloudSyncPolicyScheduleNextSyncAt(states[schedule.Key])
		if stateNext.IsZero() {
			stateNext = now
		}
		if next.IsZero() || stateNext.Before(next) {
			next = stateNext
		}
	}
	if next.IsZero() {
		next = now
	}
	return models.Time(next)
}

func cloudSyncPolicyScheduleNextAfterTask(policy *models.CloudSyncPolicy, key string, status string, ended time.Time, failureCount int) time.Time {
	interval := cloudSyncPolicyIntervalSeconds(0)
	if policy != nil {
		interval = policy.SyncInterval
		for _, schedule := range cloudSyncPolicyScheduleOverrides(policy) {
			if schedule.Key == key {
				interval = schedule.SyncInterval
				break
			}
		}
	}
	if status != models.CmdbSyncTaskComplete && policy != nil && policy.MaxRetryAttempts > 0 && failureCount <= policy.MaxRetryAttempts {
		return ended.Add(time.Duration(cloudSyncPolicyFailureBackoffSeconds(policy.RetryBackoffSeconds, failureCount)) * time.Second)
	}
	return ended.Add(time.Duration(cloudSyncPolicyIntervalSeconds(interval)) * time.Second)
}

func cloudSyncPolicyScheduleNextSyncAt(state cloudSyncPolicyScheduleState) time.Time {
	if state.NextSyncAt == "" {
		return time.Time{}
	}
	value, err := time.Parse(time.RFC3339, state.NextSyncAt)
	if err != nil {
		return time.Time{}
	}
	return value
}

func cloudSyncPolicyScheduleName(policy *models.CloudSyncPolicy, key string) string {
	for _, schedule := range cloudSyncPolicyScheduleOverrides(policy) {
		if schedule.Key == key {
			return schedule.Name
		}
	}
	return key
}

func cloudSyncPolicyScheduleKey(regions []string, assetTypes []string, syncInterval int) string {
	regionValues := append([]string{}, regions...)
	assetTypeValues := append([]string{}, assetTypes...)
	sort.Strings(regionValues)
	sort.Strings(assetTypeValues)
	source := strings.Join(regionValues, ",") + "|" + strings.Join(assetTypeValues, ",") + "|" + strconv.Itoa(cloudSyncPolicyIntervalSeconds(syncInterval))
	sum := sha1.Sum([]byte(source))
	return fmt.Sprintf("sch-%x", sum[:6])
}

func cloudSyncPolicyStatsString(stats models.ResAttrs, key string) string {
	if stats == nil {
		return ""
	}
	value, ok := stats[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func taskIdString(task *resps.CmdbSyncTaskResp) string {
	if task == nil {
		return ""
	}
	return task.Id.String()
}

func cloudSyncPolicyNormalizePauseWindows(value interface{}) []string {
	values := normalizeStringList(cloudSyncPolicyAttrStringSlice(value))
	result := make([]string, 0, len(values))
	for _, value := range values {
		start, end, ok := cloudSyncPolicyParsePauseWindow(value)
		if !ok {
			continue
		}
		result = append(result, fmt.Sprintf("%02d:%02d-%02d:%02d", start/60, start%60, end/60, end%60))
	}
	return dedupeStrings(result)
}

func cloudSyncPolicyPauseWindows(policy *models.CloudSyncPolicy) []string {
	if policy == nil {
		return []string{}
	}
	return cloudSyncPolicyNormalizePauseWindows(policy.Params["pauseWindows"])
}

func cloudSyncPolicyNormalizeRoutingValues(value interface{}) []string {
	values := normalizeStringList(cloudSyncPolicyAttrStringSlice(value))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		value = cloudSyncPolicyTruncateString(value, 80)
		result = append(result, value)
		if len(result) >= 20 {
			break
		}
	}
	return dedupeStrings(result)
}

func cloudSyncPolicyTruncateString(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

func cloudSyncPolicyPauseReason(policy *models.CloudSyncPolicy, now time.Time) string {
	for _, window := range cloudSyncPolicyPauseWindows(policy) {
		start, end, ok := cloudSyncPolicyParsePauseWindow(window)
		if !ok {
			continue
		}
		if cloudSyncPolicyTimeInWindow(now, start, end) {
			return fmt.Sprintf("同步策略处于暂停窗口 %s", window)
		}
	}
	return ""
}

func cloudSyncPolicyParsePauseWindow(value string) (int, int, bool) {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 2 {
		return 0, 0, false
	}
	start, ok := cloudSyncPolicyParseClock(parts[0])
	if !ok {
		return 0, 0, false
	}
	end, ok := cloudSyncPolicyParseClock(parts[1])
	if !ok {
		return 0, 0, false
	}
	return start, end, true
}

func cloudSyncPolicyParseClock(value string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return 0, false
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, false
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

func cloudSyncPolicyTimeInWindow(now time.Time, start int, end int) bool {
	minute := now.Hour()*60 + now.Minute()
	if start == end {
		return true
	}
	if start < end {
		return minute >= start && minute <= end
	}
	return minute >= start || minute <= end
}

func cloudSyncPolicyMaxRunsPerDay(policy *models.CloudSyncPolicy) int {
	if policy == nil {
		return 0
	}
	return cloudSyncPolicyAttrInt(policy.Params["maxRunsPerDay"])
}

func cloudSyncPolicyRunsToday(c *ctx.ServiceContext, policy *models.CloudSyncPolicy) (int, error) {
	if c == nil || policy == nil {
		return 0, nil
	}
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	count, err := c.DB().Model(&models.CmdbSyncTask{}).
		Where("org_id = ? and sync_policy_id = ? and created_at >= ?", policy.OrgId, policy.Id, startOfDay).
		Count()
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func cloudSyncPolicyRunningTaskCount(c *ctx.ServiceContext) (int, error) {
	if c == nil {
		return 0, nil
	}
	count, err := c.DB().Model(&models.CmdbSyncTask{}).
		Where("org_id = ? and status in (?)", c.OrgId, []string{models.CmdbSyncTaskPending, models.CmdbSyncTaskRunning}).
		Count()
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func cloudSyncPolicyMaxTriggeredPerRun() int {
	return cloudSyncPolicyEnvInt(cloudSyncPolicyMaxTriggeredPerRunEnv)
}

func cloudSyncPolicyMaxConcurrentRunning() int {
	return cloudSyncPolicyEnvInt(cloudSyncPolicyMaxConcurrentRunningEnv)
}

func cloudSyncPolicyEnvInt(key string) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value <= 0 {
		return 0
	}
	return value
}

func cloudSyncPolicyAttrInt(value interface{}) int {
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case int64:
		if typed > 0 {
			return int(typed)
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	case json.Number:
		if value, err := typed.Int64(); err == nil && value > 0 {
			return int(value)
		}
	case string:
		value, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil && value > 0 {
			return value
		}
	}
	return 0
}

func cloudSyncPolicyAttrInt64(value interface{}) int64 {
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return int64(typed)
		}
	case int64:
		if typed > 0 {
			return typed
		}
	case float64:
		if typed > 0 {
			return int64(typed)
		}
	case json.Number:
		if value, err := typed.Int64(); err == nil && value > 0 {
			return value
		}
	case string:
		value, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err == nil && value > 0 {
			return value
		}
	}
	return 0
}

func cloudSyncPolicyAttrBool(value interface{}) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case int:
		return typed > 0
	case int64:
		return typed > 0
	case float64:
		return typed > 0
	case json.Number:
		parsed, err := typed.Int64()
		return err == nil && parsed > 0
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "y", "on", "enable", "enabled":
			return true
		}
	}
	return false
}

func cloudSyncPolicyAttrStringSlice(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []interface{}:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, fmt.Sprint(item))
		}
		return items
	case string:
		return []string{typed}
	case nil:
		return nil
	default:
		return []string{fmt.Sprint(typed)}
	}
}

func cloudSyncPolicyAttrItems(value interface{}) []interface{} {
	switch typed := value.(type) {
	case []interface{}:
		return typed
	case []models.ResAttrs:
		items := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items
	case []map[string]interface{}:
		items := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items
	case nil:
		return nil
	default:
		bytes, err := json.Marshal(typed)
		if err != nil {
			return nil
		}
		items := make([]interface{}, 0)
		if err := json.Unmarshal(bytes, &items); err == nil {
			return items
		}
		return nil
	}
}

func cloudSyncPolicyAttrMap(value interface{}) models.ResAttrs {
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
	case nil:
		return models.ResAttrs{}
	default:
		bytes, err := json.Marshal(typed)
		if err != nil {
			return models.ResAttrs{}
		}
		result := models.ResAttrs{}
		if err := json.Unmarshal(bytes, &result); err != nil {
			return models.ResAttrs{}
		}
		return result
	}
}

func cloudSyncPolicyAttrString(value interface{}) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func joinOrDefault(values []string, defaultValue string) string {
	if len(values) == 0 {
		return defaultValue
	}
	return strings.Join(values, ", ")
}
