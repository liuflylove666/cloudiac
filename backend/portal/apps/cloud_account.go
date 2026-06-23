// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
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
	"cloudiac/utils"
	"cloudiac/utils/logs"
)

const (
	cloudAccountHealthWorkerDefaultInterval = 30 * time.Minute
	cloudAccountHealthWorkerMinInterval     = time.Minute
	cloudAccountHealthWorkerMaxInterval     = 24 * time.Hour
	cloudAccountHealthEventDefaultQuiet     = 24 * time.Hour
	cloudAccountHealthEventMinQuiet         = time.Hour
	cloudAccountHealthEventMaxQuiet         = 30 * 24 * time.Hour
	cloudAccountHealthDefaultShardTotal     = 1
	cloudAccountHealthMaxShardTotal         = 128
	cloudAccountHealthManualConcurrency     = 1
	cloudAccountHealthWorkerConcurrency     = 4
	cloudAccountHealthMaxConcurrency        = 32

	cloudAccountHealthWorkerIntervalEnv    = "CLOUDIAC_CLOUD_ACCOUNT_HEALTH_WORKER_INTERVAL_SECONDS"
	cloudAccountHealthEventQuietEnv        = "CLOUDIAC_CLOUD_ACCOUNT_HEALTH_EVENT_QUIET_HOURS"
	cloudAccountHealthWorkerShardsEnv      = "CLOUDIAC_CLOUD_ACCOUNT_HEALTH_WORKER_SHARDS"
	cloudAccountHealthWorkerConcurrencyEnv = "CLOUDIAC_CLOUD_ACCOUNT_HEALTH_WORKER_CONCURRENCY"
)

type cloudAccountValidation struct {
	Ready                 bool
	Status                string
	Message               string
	MissingCredentialKeys []string
	SupportedAssetTypes   []string
	Regions               []string
}

type cloudAccountHealthCheck struct {
	Status  string
	Message string
}

type cloudAccountHealthCheckOptions struct {
	ShardIndex  int
	ShardTotal  int
	Concurrency int
}

func (o cloudAccountHealthCheckOptions) sharded() bool {
	return o.ShardTotal > 1
}

type cloudAccountSyncHealthWindow struct {
	MaxAge         time.Duration
	PolicyId       models.Id
	PolicyName     string
	SyncInterval   int
	PolicyCount    int64
	ScheduleKey    string
	ScheduleName   string
	Regions        []string
	AssetTypes     []string
	LastSyncTaskId models.Id
	LastSyncStatus string
	LastError      string
	LastSyncedAt   models.Time
	NextSyncAt     time.Time
}

func SearchCloudAccounts(c *ctx.ServiceContext, form *forms.SearchCloudAccountForm) (interface{}, e.Error) {
	query := services.QueryCloudAccount(c.DB()).Where("org_id = ?", c.OrgId)
	if form.Q != "" {
		qs := "%" + form.Q + "%"
		query = query.Where("name like ? or description like ? or account_id like ?", qs, qs, qs)
	}
	if form.Provider != "" {
		query = query.Where("provider = ?", normalizeProvider(form.Provider))
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	if form.ValidationStatus != "" {
		query = query.Where("validation_status = ?", form.ValidationStatus)
	}
	if form.SortField() == "" {
		query = query.Order("created_at desc")
	} else {
		query = form.Order(query)
	}

	accounts := make([]models.CloudAccount, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&accounts); err != nil {
		return nil, e.New(e.DBError, err)
	}

	list := make([]resps.CloudAccountResp, 0, len(accounts))
	for _, account := range accounts {
		list = append(list, cloudAccountResp(c, account))
	}

	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func CloudAccountDetail(c *ctx.ServiceContext, form *forms.CloudAccountParam) (*resps.CloudAccountResp, e.Error) {
	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, form.Id)
	if err != nil {
		return nil, err
	}
	resp := cloudAccountResp(c, *account)
	return &resp, nil
}

func CreateCloudAccount(c *ctx.ServiceContext, form *forms.CreateCloudAccountForm) (*resps.CloudAccountResp, e.Error) {
	c.AddLogField("action", fmt.Sprintf("create cloud account %s", form.Name))

	provider := normalizeProvider(form.Provider)
	regions := normalizeStringList(form.Regions)
	credentials, err := marshalCloudCredentialParams(form.Credentials, nil)
	if err != nil {
		return nil, e.New(e.BadParam, err)
	}
	credentialMap := credentialMapFromCloudCredentials(credentials)
	regions = firstNonEmptyStringSlice(regions, inferCloudRegions(provider, credentialMap))

	account := &models.CloudAccount{
		OrgId:       c.OrgId,
		Name:        form.Name,
		Description: form.Description,
		Provider:    provider,
		AccountId:   inferCloudAccountId(provider, form.AccountId, form.TenantId, credentialMap),
		TenantId:    form.TenantId,
		Regions:     models.StrSlice(regions),
		RunnerTags:  models.StrSlice(normalizeStringList(form.RunnerTags)),
		Credentials: credentials,
		Metadata:    form.Metadata,
		Status:      firstNonEmpty(form.Status, models.CloudAccountStatusEnabled),
	}
	account.Id = models.NewId("cla")
	applyCloudAccountValidation(account)

	created, createErr := services.CreateCloudAccount(c.DB(), account)
	if createErr != nil {
		return nil, createErr
	}
	if err := syncCloudAccountRegionSnapshot(c, created, validateCloudAccount(created)); err != nil {
		return nil, err
	}
	resp := cloudAccountResp(c, *created)
	return &resp, nil
}

func UpdateCloudAccount(c *ctx.ServiceContext, form *forms.UpdateCloudAccountForm) (*resps.CloudAccountResp, e.Error) {
	c.AddLogField("action", fmt.Sprintf("update cloud account %s", form.Id))

	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, form.Id)
	if err != nil {
		return nil, err
	}

	attrs := models.Attrs{}
	if form.HasKey("name") {
		attrs["name"] = form.Name
		account.Name = form.Name
	}
	if form.HasKey("description") {
		attrs["description"] = form.Description
		account.Description = form.Description
	}
	if form.HasKey("provider") {
		account.Provider = normalizeProvider(form.Provider)
		attrs["provider"] = account.Provider
	}
	if form.HasKey("tenantId") {
		attrs["tenant_id"] = form.TenantId
		account.TenantId = form.TenantId
	}
	if form.HasKey("regions") {
		regions := normalizeStringList(form.Regions)
		attrs["regions"] = models.StrSlice(regions)
		account.Regions = models.StrSlice(regions)
	}
	if form.HasKey("runnerTags") {
		runnerTags := normalizeStringList(form.RunnerTags)
		attrs["runner_tags"] = models.StrSlice(runnerTags)
		account.RunnerTags = models.StrSlice(runnerTags)
	}
	if form.HasKey("credentials") {
		credentials, err := marshalCloudCredentialParams(form.Credentials, encryptedCloudCredentialValues(account.Credentials))
		if err != nil {
			return nil, e.New(e.BadParam, err)
		}
		attrs["credentials"] = credentials
		account.Credentials = credentials
	}
	if form.HasKey("metadata") {
		attrs["metadata"] = form.Metadata
		account.Metadata = form.Metadata
	}
	if form.HasKey("accountId") {
		attrs["account_id"] = form.AccountId
		account.AccountId = form.AccountId
	}
	if form.HasKey("status") {
		attrs["status"] = form.Status
		account.Status = form.Status
	}
	if form.HasKey("provider") || form.HasKey("regions") || form.HasKey("credentials") || form.HasKey("accountId") || form.HasKey("tenantId") {
		credentials := credentialMapFromCloudCredentials(account.Credentials)
		account.AccountId = inferCloudAccountId(account.Provider, account.AccountId, account.TenantId, credentials)
		account.Regions = models.StrSlice(firstNonEmptyStringSlice([]string(account.Regions), inferCloudRegions(account.Provider, credentials)))
		attrs["account_id"] = account.AccountId
		attrs["regions"] = account.Regions
		validationAttrs := cloudAccountValidationAttrs(account)
		for key, value := range validationAttrs {
			attrs[key] = value
		}
	}
	if len(attrs) == 0 {
		resp := cloudAccountResp(c, *account)
		return &resp, nil
	}

	updated, updateErr := services.UpdateCloudAccount(c.DB(), c.OrgId, form.Id, attrs)
	if updateErr != nil {
		return nil, updateErr
	}
	if form.HasKey("provider") || form.HasKey("regions") || form.HasKey("credentials") || form.HasKey("accountId") || form.HasKey("tenantId") || form.HasKey("status") {
		if err := syncCloudAccountRegionSnapshot(c, updated, validateCloudAccount(updated)); err != nil {
			return nil, err
		}
	}
	resp := cloudAccountResp(c, *updated)
	return &resp, nil
}

func DeleteCloudAccount(c *ctx.ServiceContext, form *forms.CloudAccountParam) (interface{}, e.Error) {
	c.AddLogField("action", fmt.Sprintf("delete cloud account %s", form.Id))
	return nil, services.DeleteCloudAccount(c.DB(), c.OrgId, form.Id)
}

func ValidateCloudAccount(c *ctx.ServiceContext, form *forms.CloudAccountParam) (*resps.CloudAccountValidationResp, e.Error) {
	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, form.Id)
	if err != nil {
		return nil, err
	}

	attrs := cloudAccountValidationAttrs(account)
	if _, err := c.DB().Model(&models.CloudAccount{}).
		Where("id = ? and org_id = ?", form.Id, c.OrgId).
		UpdateAttrs(attrs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	applyCloudAccountAttrs(account, attrs)
	cloudAccountEvent(c, account, "cloud_account.validated", "云账号验证", account.ValidationStatus, account.ValidationMessage)

	validation := validateCloudAccount(account)
	permissions := cloudAccountPermissionItems(account, validation)
	checkedAt := account.LastValidatedAt
	if time.Time(checkedAt).IsZero() || time.Time(checkedAt).Year() <= 1 {
		checkedAt = models.Time(time.Now())
	}
	if err := saveCloudAccountPermissionSnapshot(c, account, permissions, checkedAt, "local_precheck"); err != nil {
		return nil, err
	}
	if err := syncCloudAccountRegionSnapshot(c, account, validation); err != nil {
		return nil, err
	}
	return &resps.CloudAccountValidationResp{
		Id:                    account.Id,
		Provider:              account.Provider,
		AccountId:             account.AccountId,
		ValidationStatus:      account.ValidationStatus,
		ValidationMessage:     account.ValidationMessage,
		MissingCredentialKeys: validation.MissingCredentialKeys,
		SupportedAssetTypes:   validation.SupportedAssetTypes,
		Regions:               validation.Regions,
	}, nil
}

func CheckCloudAccountHealth(c *ctx.ServiceContext, form *forms.CloudAccountParam) (*resps.CloudAccountHealthResp, e.Error) {
	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, form.Id)
	if err != nil {
		return nil, err
	}
	return checkCloudAccountHealth(c, account, true)
}

func CheckCloudAccountsHealth(c *ctx.ServiceContext, form *forms.CheckCloudAccountsHealthForm) (*resps.CloudAccountHealthSummaryResp, e.Error) {
	options, err := cloudAccountHealthCheckOptionsFromForm(form, cloudAccountHealthManualConcurrency)
	if err != nil {
		return nil, err
	}
	accounts := make([]models.CloudAccount, 0)
	query := services.QueryCloudAccount(c.DB()).Where("org_id = ?", c.OrgId)
	query = applyCloudAccountHealthShard(query, options)
	if err := query.Find(&accounts); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return runCloudAccountHealthChecks(accounts, options, true, func(account models.CloudAccount) *ctx.ServiceContext {
		return cloudAccountHealthChildContext(c, account)
	})
}

func CheckCloudAccountsHealthForAllOrgs() (*resps.CloudAccountHealthSummaryResp, e.Error) {
	return checkCloudAccountsHealthForAllOrgsWithOptions(cloudAccountHealthCheckOptions{
		ShardTotal:  1,
		Concurrency: cloudAccountHealthManualConcurrency,
	})
}

func checkCloudAccountsHealthForAllOrgsWithOptions(options cloudAccountHealthCheckOptions) (*resps.CloudAccountHealthSummaryResp, e.Error) {
	options = cloudAccountHealthNormalizeOptions(options, cloudAccountHealthManualConcurrency)
	accounts := make([]models.CloudAccount, 0)
	query := applyCloudAccountHealthShard(db.Get().Model(&models.CloudAccount{}), options)
	if err := query.Find(&accounts); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return runCloudAccountHealthChecks(accounts, options, false, cloudAccountHealthSystemContext)
}

func CheckCloudAccountsHealthForAllOrgsWithLock() (*resps.CloudAccountHealthSummaryResp, bool, e.Error) {
	options := cloudAccountHealthWorkerOptions()
	if options.sharded() {
		return checkCloudAccountsHealthShardsForAllOrgsWithLock(options)
	}
	return checkCloudAccountsHealthForAllOrgsWithLockOptions(options)
}

func checkCloudAccountsHealthForAllOrgsWithLockOptions(options cloudAccountHealthCheckOptions) (*resps.CloudAccountHealthSummaryResp, bool, e.Error) {
	options = cloudAccountHealthNormalizeOptions(options, cloudAccountHealthWorkerConcurrency)
	locked, releaseLock, lockErr := cloudWebhookAcquireMysqlLock(cloudAccountHealthLockName(options))
	if lockErr != nil {
		return nil, false, e.New(e.DBError, lockErr)
	}
	if !locked {
		result := cloudAccountHealthNewSummary(options, 0)
		result.LockSkipped = true
		result.LockSkippedCount = 1
		return result, false, nil
	}
	defer releaseLock()
	result, err := checkCloudAccountsHealthForAllOrgsWithOptions(options)
	if result != nil {
		result.Locked = true
	}
	return result, true, err
}

func StartCloudAccountHealthWorker(serviceId string) {
	interval := cloudAccountHealthWorkerInterval()
	options := cloudAccountHealthWorkerOptions()
	logger := logs.Get().
		WithField("worker", "cloudAccountHealth").
		WithField("serviceId", serviceId).
		WithField("interval", interval.String()).
		WithField("shardTotal", options.ShardTotal).
		WithField("concurrency", options.Concurrency)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if result, locked, err := CheckCloudAccountsHealthForAllOrgsWithLock(); err != nil {
			logger.Warnf("check cloud account health failed: %v", err)
		} else if !locked {
			logger.Infof("check cloud account health skipped: lock is held by another portal")
		} else {
			logger.Infof("check cloud account health result: total=%d checked=%d healthy=%d warning=%d unhealthy=%d shardTotal=%d concurrency=%d lockSkipped=%d",
				result.TotalCount, result.CheckedCount, result.HealthyCount, result.WarningCount, result.UnhealthyCount,
				result.ShardTotal, result.Concurrency, result.LockSkippedCount)
		}
		<-ticker.C
	}
}

func checkCloudAccountsHealthShardsForAllOrgsWithLock(options cloudAccountHealthCheckOptions) (*resps.CloudAccountHealthSummaryResp, bool, e.Error) {
	options = cloudAccountHealthNormalizeOptions(options, cloudAccountHealthWorkerConcurrency)
	result := cloudAccountHealthNewSummary(options, 0)
	result.Shards = make([]resps.CloudAccountHealthShardResp, options.ShardTotal)
	lockedShards := make([]int, 0, options.ShardTotal)
	releaseLocks := make([]func(), 0, options.ShardTotal)
	for shardIndex := 0; shardIndex < options.ShardTotal; shardIndex++ {
		shardOptions := options
		shardOptions.ShardIndex = shardIndex
		shard := resps.CloudAccountHealthShardResp{
			ShardIndex: shardIndex,
			ShardTotal: options.ShardTotal,
		}
		locked, releaseLock, lockErr := cloudWebhookAcquireMysqlLock(cloudAccountHealthLockName(shardOptions))
		if lockErr != nil {
			for _, release := range releaseLocks {
				release()
			}
			return nil, false, e.New(e.DBError, lockErr)
		}
		if !locked {
			shard.LockSkipped = true
			result.LockSkipped = true
			result.LockSkippedCount++
			result.Shards[shardIndex] = shard
			continue
		}
		shard.Locked = true
		result.Locked = true
		result.Shards[shardIndex] = shard
		lockedShards = append(lockedShards, shardIndex)
		releaseLocks = append(releaseLocks, releaseLock)
	}
	defer func() {
		for _, release := range releaseLocks {
			release()
		}
	}()
	if len(lockedShards) == 0 {
		return result, false, nil
	}

	accounts := make([]models.CloudAccount, 0)
	query := applyCloudAccountHealthShardIndexes(db.Get().Model(&models.CloudAccount{}), options.ShardTotal, lockedShards)
	if err := query.Find(&accounts); err != nil {
		return nil, false, e.New(e.DBError, err)
	}
	for idx := range accounts {
		shardIndex := cloudAccountHealthShardIndex(accounts[idx].Id, options.ShardTotal)
		if shardIndex >= 0 && shardIndex < len(result.Shards) {
			result.Shards[shardIndex].TotalCount++
		}
	}

	checkResult, err := runCloudAccountHealthChecks(accounts, options, false, cloudAccountHealthSystemContext)
	if err != nil {
		return nil, true, err
	}
	result.TotalCount = checkResult.TotalCount
	result.CheckedCount = checkResult.CheckedCount
	result.HealthyCount = checkResult.HealthyCount
	result.WarningCount = checkResult.WarningCount
	result.UnhealthyCount = checkResult.UnhealthyCount
	result.List = checkResult.List
	for idx := range result.List {
		shardIndex := cloudAccountHealthShardIndex(result.List[idx].Id, options.ShardTotal)
		if shardIndex < 0 || shardIndex >= len(result.Shards) {
			continue
		}
		cloudAccountHealthAccumulateShard(&result.Shards[shardIndex], result.List[idx])
	}
	return result, true, nil
}

func runCloudAccountHealthChecks(accounts []models.CloudAccount, options cloudAccountHealthCheckOptions, forceEvent bool, contextFn func(models.CloudAccount) *ctx.ServiceContext) (*resps.CloudAccountHealthSummaryResp, e.Error) {
	options = cloudAccountHealthNormalizeOptions(options, cloudAccountHealthManualConcurrency)
	result := cloudAccountHealthNewSummary(options, len(accounts))
	if len(accounts) == 0 {
		return result, nil
	}
	concurrency := options.Concurrency
	if concurrency > len(accounts) {
		concurrency = len(accounts)
	}
	type healthResult struct {
		index int
		item  *resps.CloudAccountHealthResp
		err   e.Error
	}
	jobs := make(chan int)
	results := make(chan healthResult, len(accounts))
	var wg sync.WaitGroup
	for workerIndex := 0; workerIndex < concurrency; workerIndex++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				account := accounts[index]
				if account.OrgId == "" {
					results <- healthResult{index: index}
					continue
				}
				workerCtx := contextFn(account)
				item, err := checkCloudAccountHealth(workerCtx, &account, forceEvent)
				results <- healthResult{index: index, item: item, err: err}
			}
		}()
	}
	for idx := range accounts {
		jobs <- idx
	}
	close(jobs)
	wg.Wait()
	close(results)

	ordered := make([]healthResult, len(accounts))
	for item := range results {
		ordered[item.index] = item
	}
	for idx := range ordered {
		item := ordered[idx]
		if item.err != nil {
			return nil, item.err
		}
		if item.item == nil {
			continue
		}
		cloudAccountHealthAccumulateSummary(result, *item.item)
		result.List = append(result.List, *item.item)
	}
	if options.sharded() {
		shard := resps.CloudAccountHealthShardResp{
			ShardIndex: options.ShardIndex,
			ShardTotal: options.ShardTotal,
			TotalCount: result.TotalCount,
		}
		for idx := range result.List {
			cloudAccountHealthAccumulateShard(&shard, result.List[idx])
		}
		result.Shards = []resps.CloudAccountHealthShardResp{shard}
	}
	return result, nil
}

func cloudAccountHealthNewSummary(options cloudAccountHealthCheckOptions, capacity int) *resps.CloudAccountHealthSummaryResp {
	options = cloudAccountHealthNormalizeOptions(options, cloudAccountHealthManualConcurrency)
	return &resps.CloudAccountHealthSummaryResp{
		TotalCount:  int64(capacity),
		ShardIndex:  options.ShardIndex,
		ShardTotal:  options.ShardTotal,
		Concurrency: options.Concurrency,
		List:        make([]resps.CloudAccountHealthResp, 0, capacity),
	}
}

func cloudAccountHealthAccumulateSummary(result *resps.CloudAccountHealthSummaryResp, item resps.CloudAccountHealthResp) {
	result.CheckedCount++
	switch item.HealthStatus {
	case models.CloudAccountHealthHealthy:
		result.HealthyCount++
	case models.CloudAccountHealthUnhealthy:
		result.UnhealthyCount++
	default:
		result.WarningCount++
	}
}

func cloudAccountHealthAccumulateShard(result *resps.CloudAccountHealthShardResp, item resps.CloudAccountHealthResp) {
	result.CheckedCount++
	switch item.HealthStatus {
	case models.CloudAccountHealthHealthy:
		result.HealthyCount++
	case models.CloudAccountHealthUnhealthy:
		result.UnhealthyCount++
	default:
		result.WarningCount++
	}
}

func cloudAccountHealthChildContext(parent *ctx.ServiceContext, account models.CloudAccount) *ctx.ServiceContext {
	child := &ctx.ServiceContext{
		UserId:       parent.UserId,
		OrgId:        account.OrgId,
		ProjectId:    parent.ProjectId,
		Email:        parent.Email,
		Username:     parent.Username,
		IsSuperAdmin: parent.IsSuperAdmin,
		UserIpAddr:   parent.UserIpAddr,
	}
	if child.OrgId == "" {
		child.OrgId = parent.OrgId
	}
	return child
}

func cloudAccountHealthSystemContext(account models.CloudAccount) *ctx.ServiceContext {
	return &ctx.ServiceContext{
		UserId:   consts.SysUserId,
		OrgId:    account.OrgId,
		Email:    consts.DefaultSysEmail,
		Username: consts.DefaultSysName,
	}
}

func CloudAccountRegions(c *ctx.ServiceContext, form *forms.CloudAccountParam) (*resps.CloudAccountRegionsResp, e.Error) {
	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, form.Id)
	if err != nil {
		return nil, err
	}
	return cloudAccountRegionsResp(c, account)
}

func UpdateCloudAccountRegions(c *ctx.ServiceContext, form *forms.UpdateCloudAccountRegionsForm) (*resps.CloudAccountRegionsResp, e.Error) {
	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, form.Id)
	if err != nil {
		return nil, err
	}

	regions := normalizeStringList(form.Regions)
	account.Regions = models.StrSlice(regions)
	attrs := models.Attrs{
		"regions": account.Regions,
	}
	for key, value := range cloudAccountValidationAttrs(account) {
		attrs[key] = value
	}

	updated, updateErr := services.UpdateCloudAccount(c.DB(), c.OrgId, form.Id, attrs)
	if updateErr != nil {
		return nil, updateErr
	}
	validation := validateCloudAccount(updated)
	if err := syncCloudAccountRegionSnapshot(c, updated, validation); err != nil {
		return nil, err
	}
	return cloudAccountRegionsResp(c, updated)
}

func CloudAccountPermissions(c *ctx.ServiceContext, form *forms.CloudAccountParam) (*resps.CloudAccountPermissionsResp, e.Error) {
	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, form.Id)
	if err != nil {
		return nil, err
	}

	validation := validateCloudAccount(account)
	permissions, lastCheckedAt, err := cloudAccountPermissionItemsWithSnapshot(c, account, cloudAccountPermissionItems(account, validation))
	if err != nil {
		return nil, err
	}
	return &resps.CloudAccountPermissionsResp{
		Id:                    account.Id,
		Provider:              account.Provider,
		AccountId:             account.AccountId,
		ValidationStatus:      account.ValidationStatus,
		ValidationMessage:     validation.Message,
		MissingCredentialKeys: validation.MissingCredentialKeys,
		SupportedAssetTypes:   validation.SupportedAssetTypes,
		Regions:               validation.Regions,
		LastCheckedAt:         lastCheckedAt,
		Permissions:           permissions,
	}, nil
}

func cloudAccountResp(c *ctx.ServiceContext, account models.CloudAccount) resps.CloudAccountResp {
	validation := validateCloudAccount(&account)
	account.Credentials = maskedCloudCredentials(account.Credentials)
	if len(account.SupportedTypes) == 0 {
		account.SupportedTypes = models.StrSlice(validation.SupportedAssetTypes)
	}
	if len(account.Regions) == 0 {
		account.Regions = models.StrSlice(validation.Regions)
	}
	return resps.CloudAccountResp{
		CloudAccount:          account,
		Ready:                 validation.Ready,
		RegionCount:           len(account.Regions),
		MissingCredentialKeys: validation.MissingCredentialKeys,
		HealthDetail:          cloudAccountHealthDetail(c, &account),
	}
}

func checkCloudAccountHealth(c *ctx.ServiceContext, account *models.CloudAccount, forceEvent bool) (*resps.CloudAccountHealthResp, e.Error) {
	previousStatus := account.HealthStatus
	previousMessage := account.HealthMessage
	previousCheckedAt := account.LastHealthCheckedAt
	attrs := cloudAccountValidationAttrs(account)
	applyCloudAccountAttrs(account, attrs)
	health := cloudAccountHealthWithPolicies(c, account)
	attrs["health_status"] = health.Status
	attrs["health_message"] = health.Message
	attrs["last_health_checked_at"] = models.Time(time.Now())
	if _, err := c.DB().Model(&models.CloudAccount{}).
		Where("id = ? and org_id = ?", account.Id, c.OrgId).
		UpdateAttrs(attrs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	applyCloudAccountAttrs(account, attrs)
	if cloudAccountHealthShouldEmitEvent(forceEvent, previousStatus, previousMessage, health.Status, health.Message, previousCheckedAt) {
		cloudAccountEvent(c, account, "cloud_account.health_checked", "云账号健康检查", account.HealthStatus, account.HealthMessage)
	}
	resp := cloudAccountHealthResp(c, *account)
	return &resp, nil
}

func cloudAccountHealthResp(c *ctx.ServiceContext, account models.CloudAccount) resps.CloudAccountHealthResp {
	return resps.CloudAccountHealthResp{
		Id:                  account.Id,
		Provider:            account.Provider,
		AccountId:           account.AccountId,
		HealthStatus:        account.HealthStatus,
		HealthMessage:       account.HealthMessage,
		ValidationStatus:    account.ValidationStatus,
		ValidationMessage:   account.ValidationMessage,
		LastHealthCheckedAt: account.LastHealthCheckedAt,
		LastValidatedAt:     account.LastValidatedAt,
		LastSyncAt:          account.LastSyncAt,
		HealthDetail:        cloudAccountHealthDetail(c, &account),
	}
}

func cloudAccountHealth(account *models.CloudAccount) cloudAccountHealthCheck {
	return cloudAccountHealthWithWindow(account, cloudAccountSyncHealthWindow{
		MaxAge: 24 * time.Hour,
	})
}

func cloudAccountHealthWithPolicies(c *ctx.ServiceContext, account *models.CloudAccount) cloudAccountHealthCheck {
	window := cloudAccountSyncHealthWindow{
		MaxAge: 24 * time.Hour,
	}
	if c != nil && account != nil && account.Id != "" {
		if policyWindow, ok := cloudAccountPolicyHealthWindow(c, account); ok {
			window = policyWindow
		}
	}
	health := cloudAccountHealthWithWindow(account, window)
	if impact := cloudAccountHealthFailureImpact(c, account, window); impact != nil {
		health = cloudAccountHealthApplyFailureImpact(health, impact, cloudAccountHealthTargetName(window))
	}
	return health
}

func cloudAccountHealthWithWindow(account *models.CloudAccount, syncWindow cloudAccountSyncHealthWindow) cloudAccountHealthCheck {
	validation := validateCloudAccount(account)
	if account.Status == models.CloudAccountStatusDisabled {
		return cloudAccountHealthCheck{
			Status:  models.CloudAccountHealthWarning,
			Message: "云账号已禁用，不会参与同步或云操作",
		}
	}
	if len(validation.SupportedAssetTypes) == 0 {
		return cloudAccountHealthCheck{
			Status:  models.CloudAccountHealthUnhealthy,
			Message: validation.Message,
		}
	}
	if len(validation.MissingCredentialKeys) > 0 {
		return cloudAccountHealthCheck{
			Status:  models.CloudAccountHealthUnhealthy,
			Message: validation.Message,
		}
	}
	if len(validation.Regions) == 0 {
		return cloudAccountHealthCheck{
			Status:  models.CloudAccountHealthWarning,
			Message: "未配置或无法推断可用区域",
		}
	}
	if syncWindow.MaxAge <= 0 {
		syncWindow.MaxAge = 24 * time.Hour
	}
	lastSync := time.Time(account.LastSyncAt)
	if windowLastSync := time.Time(syncWindow.LastSyncedAt); !windowLastSync.IsZero() && windowLastSync.Year() > 1 {
		lastSync = windowLastSync
	}
	if syncWindow.LastSyncStatus == models.CmdbSyncTaskFailed {
		target := cloudAccountHealthTargetName(syncWindow)
		if target != "" {
			return cloudAccountHealthCheck{
				Status:  models.CloudAccountHealthWarning,
				Message: fmt.Sprintf("%s 最近一次同步失败：%s", target, firstNonEmpty(syncWindow.LastError, "未返回错误详情")),
			}
		}
		return cloudAccountHealthCheck{
			Status:  models.CloudAccountHealthWarning,
			Message: fmt.Sprintf("账号最近一次云采集同步失败：%s", firstNonEmpty(syncWindow.LastError, "未返回错误详情")),
		}
	}
	if lastSync.IsZero() || lastSync.Year() <= 1 {
		if target := cloudAccountHealthTargetName(syncWindow); target != "" {
			return cloudAccountHealthCheck{
				Status:  models.CloudAccountHealthWarning,
				Message: fmt.Sprintf("%s 尚未完成云采集同步", target),
			}
		}
		return cloudAccountHealthCheck{
			Status:  models.CloudAccountHealthWarning,
			Message: "账号尚未完成云采集同步",
		}
	}
	if time.Since(lastSync) > syncWindow.MaxAge {
		if syncWindow.ScheduleName != "" {
			return cloudAccountHealthCheck{
				Status: models.CloudAccountHealthWarning,
				Message: fmt.Sprintf("同步策略 %s 的子周期 %s 最近同步时间超过健康阈值 %s：%s",
					syncWindow.PolicyName, syncWindow.ScheduleName, formatCloudAccountDuration(syncWindow.MaxAge), lastSync.Format(time.RFC3339)),
			}
		}
		if syncWindow.PolicyName != "" {
			return cloudAccountHealthCheck{
				Status: models.CloudAccountHealthWarning,
				Message: fmt.Sprintf("账号最近同步时间超过同步策略 %s 的健康阈值 %s：%s",
					syncWindow.PolicyName, formatCloudAccountDuration(syncWindow.MaxAge), lastSync.Format(time.RFC3339)),
			}
		}
		return cloudAccountHealthCheck{
			Status:  models.CloudAccountHealthWarning,
			Message: fmt.Sprintf("账号最近同步时间超过 24 小时：%s", lastSync.Format(time.RFC3339)),
		}
	}
	return cloudAccountHealthCheck{
		Status:  models.CloudAccountHealthHealthy,
		Message: "账号凭证、区域和同步状态正常",
	}
}

func cloudAccountPolicyHealthWindow(c *ctx.ServiceContext, account *models.CloudAccount) (cloudAccountSyncHealthWindow, bool) {
	policies, err := cloudAccountEnabledSyncPolicies(c, account)
	if err != nil {
		c.Logger().Warnf("query cloud sync policies for health check failed: %v", err)
		return cloudAccountSyncHealthWindow{}, false
	}
	if len(policies) == 0 {
		return cloudAccountSyncHealthWindow{}, false
	}
	windows := make([]cloudAccountSyncHealthWindow, 0, len(policies))
	for idx := range policies {
		windows = append(windows, cloudAccountPolicyHealthWindows(account, &policies[idx], int64(len(policies)))...)
	}
	if len(windows) == 0 {
		return cloudAccountSyncHealthWindow{}, false
	}
	selected := windows[0]
	selectedSeverity, selectedOverdue := cloudAccountHealthWindowSeverity(selected, time.Now())
	for _, window := range windows[1:] {
		severity, overdue := cloudAccountHealthWindowSeverity(window, time.Now())
		switch {
		case severity > selectedSeverity:
			selected = window
			selectedSeverity = severity
			selectedOverdue = overdue
		case severity == selectedSeverity && severity > 0 && overdue > selectedOverdue:
			selected = window
			selectedOverdue = overdue
		case severity == 0 && selectedSeverity == 0 && window.MaxAge < selected.MaxAge:
			selected = window
		}
	}
	return selected, true
}

func cloudAccountPolicyHealthWindows(account *models.CloudAccount, policy *models.CloudSyncPolicy, policyCount int64) []cloudAccountSyncHealthWindow {
	if policy == nil {
		return nil
	}
	schedules := cloudSyncPolicyScheduleOverrides(policy)
	if len(schedules) == 0 {
		lastSyncedAt := policy.LastSyncedAt
		if lastSynced := time.Time(lastSyncedAt); (lastSynced.IsZero() || lastSynced.Year() <= 1) && account != nil {
			lastSyncedAt = account.LastSyncAt
		}
		return []cloudAccountSyncHealthWindow{cloudAccountBuildHealthWindow(
			policy.Id,
			policy.Name,
			"",
			"",
			nil,
			nil,
			policy.SyncInterval,
			policyCount,
			policy.LastSyncTaskId,
			policy.LastSyncStatus,
			policy.LastError,
			lastSyncedAt,
			time.Time(policy.NextSyncAt),
		)}
	}
	states := cloudSyncPolicyScheduleStates(policy)
	windows := make([]cloudAccountSyncHealthWindow, 0, len(schedules))
	for _, schedule := range schedules {
		state := states[schedule.Key]
		windows = append(windows, cloudAccountBuildHealthWindow(
			policy.Id,
			policy.Name,
			schedule.Key,
			schedule.Name,
			schedule.Regions,
			schedule.AssetTypes,
			schedule.SyncInterval,
			policyCount,
			models.Id(state.LastSyncTaskId),
			state.LastSyncStatus,
			state.LastError,
			models.Time(cloudAccountParseRFC3339(state.LastSyncedAt)),
			cloudAccountParseRFC3339(state.NextSyncAt),
		))
	}
	return windows
}

func cloudAccountBuildHealthWindow(policyId models.Id, policyName string, scheduleKey string, scheduleName string, regions []string, assetTypes []string, syncInterval int, policyCount int64, lastSyncTaskId models.Id, lastSyncStatus string, lastError string, lastSyncedAt models.Time, nextSyncAt time.Time) cloudAccountSyncHealthWindow {
	intervalSeconds := cloudSyncPolicyIntervalSeconds(syncInterval)
	interval := time.Duration(intervalSeconds) * time.Second
	grace := interval / 5
	if grace < 5*time.Minute {
		grace = 5 * time.Minute
	}
	if grace > time.Hour {
		grace = time.Hour
	}
	return cloudAccountSyncHealthWindow{
		MaxAge:         interval + grace,
		PolicyId:       policyId,
		PolicyName:     policyName,
		SyncInterval:   intervalSeconds,
		PolicyCount:    policyCount,
		ScheduleKey:    scheduleKey,
		ScheduleName:   scheduleName,
		Regions:        cloneStringSlice(regions),
		AssetTypes:     cloneStringSlice(assetTypes),
		LastSyncTaskId: lastSyncTaskId,
		LastSyncStatus: lastSyncStatus,
		LastError:      lastError,
		LastSyncedAt:   lastSyncedAt,
		NextSyncAt:     nextSyncAt,
	}
}

func cloudAccountHealthDetail(c *ctx.ServiceContext, account *models.CloudAccount) resps.CloudAccountHealthDetailResp {
	window := cloudAccountSyncHealthWindow{MaxAge: 24 * time.Hour}
	var policyResp *resps.CloudAccountHealthPolicyResp
	schedules := make([]resps.CloudAccountHealthScheduleResp, 0)
	if c != nil && account != nil && account.Id != "" {
		if policyWindow, ok := cloudAccountPolicyHealthWindow(c, account); ok {
			window = policyWindow
			if policy := cloudAccountHealthPolicy(c, account.OrgId, policyWindow.PolicyId); policy != nil {
				policyResp = &resps.CloudAccountHealthPolicyResp{
					Id:                  policy.Id,
					Name:                policy.Name,
					SyncInterval:        policy.SyncInterval,
					HealthWindowSeconds: int64(policyWindow.MaxAge / time.Second),
					HealthWindowText:    formatCloudAccountDuration(policyWindow.MaxAge),
					EnabledPolicyCount:  policyWindow.PolicyCount,
					LastSyncTaskId:      policy.LastSyncTaskId,
					LastSyncStatus:      policy.LastSyncStatus,
					LastSyncedAt:        policy.LastSyncedAt,
				}
			}
		}
		schedules = cloudAccountHealthSchedules(c, account)
	}
	return resps.CloudAccountHealthDetailResp{
		HealthWindowSeconds: int64(window.MaxAge / time.Second),
		HealthWindowText:    formatCloudAccountDuration(window.MaxAge),
		EnabledPolicyCount:  window.PolicyCount,
		Policy:              policyResp,
		Schedules:           schedules,
		LastSuccessTask:     cloudAccountLatestSyncTask(c, account, models.CmdbSyncTaskComplete),
		LastFailureTask:     cloudAccountLatestSyncTask(c, account, models.CmdbSyncTaskFailed),
		FailureImpact:       cloudAccountHealthFailureImpact(c, account, window),
	}
}

func cloudAccountHealthSchedules(c *ctx.ServiceContext, account *models.CloudAccount) []resps.CloudAccountHealthScheduleResp {
	policies, err := cloudAccountEnabledSyncPolicies(c, account)
	if err != nil {
		c.Logger().Warnf("query cloud sync policy schedules for health detail failed: %v", err)
		return []resps.CloudAccountHealthScheduleResp{}
	}
	resp := make([]resps.CloudAccountHealthScheduleResp, 0)
	now := time.Now()
	for idx := range policies {
		windows := cloudAccountPolicyHealthWindows(account, &policies[idx], int64(len(policies)))
		for _, window := range windows {
			if window.ScheduleKey == "" {
				continue
			}
			resp = append(resp, resps.CloudAccountHealthScheduleResp{
				PolicyId:            window.PolicyId,
				PolicyName:          window.PolicyName,
				ScheduleKey:         window.ScheduleKey,
				ScheduleName:        window.ScheduleName,
				Regions:             cloneStringSlice(window.Regions),
				AssetTypes:          cloneStringSlice(window.AssetTypes),
				SyncInterval:        window.SyncInterval,
				HealthWindowSeconds: int64(window.MaxAge / time.Second),
				HealthWindowText:    formatCloudAccountDuration(window.MaxAge),
				LastSyncTaskId:      window.LastSyncTaskId,
				LastSyncStatus:      window.LastSyncStatus,
				LastError:           window.LastError,
				LastSyncedAt:        window.LastSyncedAt,
				Stale:               cloudAccountHealthWindowIsStale(window, now),
				DueNow:              window.NextSyncAt.IsZero() || !window.NextSyncAt.After(now),
				LastSuccessTask:     cloudAccountLatestSyncTaskForSchedule(c, account, window.PolicyId, window.ScheduleKey, models.CmdbSyncTaskComplete),
				LastFailureTask:     cloudAccountLatestSyncTaskForSchedule(c, account, window.PolicyId, window.ScheduleKey, models.CmdbSyncTaskFailed),
			})
		}
	}
	return resp
}

func cloudAccountHealthFailureImpact(c *ctx.ServiceContext, account *models.CloudAccount, window cloudAccountSyncHealthWindow) *resps.CloudAccountHealthFailureImpactResp {
	if c == nil || account == nil || account.Id == "" {
		return nil
	}
	var failureTask *resps.CloudAccountSyncTaskBriefResp
	var successTask *resps.CloudAccountSyncTaskBriefResp
	switch {
	case window.PolicyId != "" && window.ScheduleKey != "":
		failureTask = cloudAccountLatestSyncTaskForSchedule(c, account, window.PolicyId, window.ScheduleKey, models.CmdbSyncTaskFailed)
		successTask = cloudAccountLatestSyncTaskForSchedule(c, account, window.PolicyId, window.ScheduleKey, models.CmdbSyncTaskComplete)
	case window.PolicyId != "":
		failureTask = cloudAccountLatestSyncTaskForPolicy(c, account, window.PolicyId, models.CmdbSyncTaskFailed)
		successTask = cloudAccountLatestSyncTaskForPolicy(c, account, window.PolicyId, models.CmdbSyncTaskComplete)
	default:
		failureTask = cloudAccountLatestSyncTask(c, account, models.CmdbSyncTaskFailed)
		successTask = cloudAccountLatestSyncTask(c, account, models.CmdbSyncTaskComplete)
	}
	if failureTask == nil {
		return nil
	}
	failureAt := cloudAccountSyncTaskOccurredAt(failureTask)
	successAt := cloudAccountSyncTaskOccurredAt(successTask)
	if !successAt.IsZero() && !failureAt.After(successAt) {
		return nil
	}
	return cloudAccountHealthFailureImpactFromTask(failureTask)
}

func cloudAccountHealthApplyFailureImpact(base cloudAccountHealthCheck, impact *resps.CloudAccountHealthFailureImpactResp, target string) cloudAccountHealthCheck {
	if impact == nil {
		return base
	}
	if base.Status == models.CloudAccountHealthUnhealthy {
		return base
	}
	if cloudAccountHealthStatusSeverity(impact.Status) < cloudAccountHealthStatusSeverity(base.Status) {
		return base
	}
	prefix := "账号最近一次云采集失败"
	if target != "" {
		prefix = target + " 最近一次同步失败"
	}
	message := fmt.Sprintf("%s（%s）：%s", prefix, cloudAccountFailureCategoryLabel(impact.Category), firstNonEmpty(impact.Message, "未返回错误详情"))
	if impact.RetryHint != "" {
		message = fmt.Sprintf("%s；建议：%s", message, impact.RetryHint)
	}
	return cloudAccountHealthCheck{
		Status:  impact.Status,
		Message: message,
	}
}

func cloudAccountHealthStatusSeverity(status string) int {
	switch status {
	case models.CloudAccountHealthUnhealthy:
		return 2
	case models.CloudAccountHealthWarning:
		return 1
	default:
		return 0
	}
}

func cloudAccountEnabledSyncPolicies(c *ctx.ServiceContext, account *models.CloudAccount) ([]models.CloudSyncPolicy, error) {
	policies := make([]models.CloudSyncPolicy, 0)
	if c == nil || account == nil || account.Id == "" {
		return policies, nil
	}
	err := c.DB().Model(&models.CloudSyncPolicy{}).
		Where("org_id = ? and cloud_account_id = ? and status = ?", c.OrgId, account.Id, models.CloudSyncPolicyStatusEnabled).
		Find(&policies)
	return policies, err
}

func cloudAccountHealthWindowSeverity(window cloudAccountSyncHealthWindow, now time.Time) (int, time.Duration) {
	if window.LastSyncStatus == models.CmdbSyncTaskFailed {
		return 3, 0
	}
	lastSyncedAt := time.Time(window.LastSyncedAt)
	if lastSyncedAt.IsZero() || lastSyncedAt.Year() <= 1 {
		return 2, 0
	}
	overdue := now.Sub(lastSyncedAt) - window.MaxAge
	if overdue > 0 {
		return 1, overdue
	}
	return 0, 0
}

func cloudAccountHealthWindowIsStale(window cloudAccountSyncHealthWindow, now time.Time) bool {
	severity, _ := cloudAccountHealthWindowSeverity(window, now)
	return severity > 0
}

func cloudAccountHealthTargetName(window cloudAccountSyncHealthWindow) string {
	if window.PolicyName == "" {
		return ""
	}
	if window.ScheduleName != "" {
		return fmt.Sprintf("同步策略 %s 的子周期 %s", window.PolicyName, window.ScheduleName)
	}
	return fmt.Sprintf("同步策略 %s", window.PolicyName)
}

func cloudAccountParseRFC3339(value string) time.Time {
	if strings.TrimSpace(value) == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func cloudAccountHealthPolicy(c *ctx.ServiceContext, orgId models.Id, policyId models.Id) *models.CloudSyncPolicy {
	if c == nil || policyId == "" {
		return nil
	}
	policy := models.CloudSyncPolicy{}
	if err := c.DB().Model(&models.CloudSyncPolicy{}).
		Where("id = ? and org_id = ?", policyId, orgId).
		First(&policy); err != nil {
		c.Logger().Warnf("query cloud sync policy health detail failed: %v", err)
		return nil
	}
	return &policy
}

func cloudAccountLatestSyncTask(c *ctx.ServiceContext, account *models.CloudAccount, status string) *resps.CloudAccountSyncTaskBriefResp {
	if c == nil || account == nil || account.Id == "" {
		return nil
	}
	tasks := make([]models.CmdbSyncTask, 0, 1)
	if err := c.DB().Model(&models.CmdbSyncTask{}).
		Where("org_id = ? and account_source = ? and account_id = ? and provider = ? and status = ?",
			c.OrgId, models.CmdbCloudAccountSourceCloudAccount, account.Id, account.Provider, status).
		Order("ended_at desc").
		Order("created_at desc").
		Limit(1).
		Find(&tasks); err != nil {
		c.Logger().Warnf("query cloud account latest sync task failed: %v", err)
		return nil
	}
	if len(tasks) == 0 {
		return nil
	}
	return cloudAccountSyncTaskBrief(tasks[0])
}

func cloudAccountLatestSyncTaskForPolicy(c *ctx.ServiceContext, account *models.CloudAccount, policyId models.Id, status string) *resps.CloudAccountSyncTaskBriefResp {
	if c == nil || account == nil || account.Id == "" || policyId == "" {
		return nil
	}
	tasks := make([]models.CmdbSyncTask, 0, 1)
	if err := c.DB().Model(&models.CmdbSyncTask{}).
		Where("org_id = ? and account_source = ? and account_id = ? and sync_policy_id = ? and status = ?",
			c.OrgId, models.CmdbCloudAccountSourceCloudAccount, account.Id, policyId, status).
		Order("ended_at desc").
		Order("created_at desc").
		Limit(1).
		Find(&tasks); err != nil {
		c.Logger().Warnf("query cloud account policy latest sync task failed: %v", err)
		return nil
	}
	if len(tasks) == 0 {
		return nil
	}
	return cloudAccountSyncTaskBrief(tasks[0])
}

func cloudAccountLatestSyncTaskForSchedule(c *ctx.ServiceContext, account *models.CloudAccount, policyId models.Id, scheduleKey string, status string) *resps.CloudAccountSyncTaskBriefResp {
	if c == nil || account == nil || account.Id == "" || policyId == "" || scheduleKey == "" {
		return nil
	}
	tasks := make([]models.CmdbSyncTask, 0, 1)
	if err := c.DB().Model(&models.CmdbSyncTask{}).
		Where("org_id = ? and account_source = ? and account_id = ? and sync_policy_id = ? and status = ?",
			c.OrgId, models.CmdbCloudAccountSourceCloudAccount, account.Id, policyId, status).
		Where("JSON_UNQUOTE(JSON_EXTRACT(stats, '$.syncPolicyScheduleKey')) = ?", scheduleKey).
		Order("ended_at desc").
		Order("created_at desc").
		Limit(1).
		Find(&tasks); err != nil {
		c.Logger().Warnf("query cloud account schedule latest sync task failed: %v", err)
		return nil
	}
	if len(tasks) == 0 {
		return nil
	}
	return cloudAccountSyncTaskBrief(tasks[0])
}

func cloudAccountSyncTaskBrief(task models.CmdbSyncTask) *resps.CloudAccountSyncTaskBriefResp {
	return &resps.CloudAccountSyncTaskBriefResp{
		Id:           task.Id,
		SyncPolicyId: task.SyncPolicyId,
		Status:       task.Status,
		ErrorMessage: task.ErrorMessage,
		Stats:        task.Stats,
		StartedAt:    task.StartedAt,
		EndedAt:      task.EndedAt,
		CreatedAt:    task.CreatedAt,
	}
}

func cloudAccountHealthFailureImpactFromTask(task *resps.CloudAccountSyncTaskBriefResp) *resps.CloudAccountHealthFailureImpactResp {
	if task == nil {
		return nil
	}
	category, message, retryable, retryHint := cloudAccountSyncFailureDetail(task)
	if strings.TrimSpace(message) == "" {
		message = task.ErrorMessage
	}
	if strings.TrimSpace(message) == "" {
		message = "未返回错误详情"
	}
	if category == "" {
		category, retryable, retryHint = cloudAccountClassifySyncFailure(message)
	}
	occurredAt := cloudAccountSyncTaskOccurredAt(task)
	return &resps.CloudAccountHealthFailureImpactResp{
		TaskId:     task.Id,
		Status:     cloudAccountHealthStatusForFailureCategory(category, retryable),
		Category:   category,
		Message:    message,
		Retryable:  retryable,
		RetryHint:  retryHint,
		OccurredAt: models.Time(occurredAt),
	}
}

func cloudAccountSyncFailureDetail(task *resps.CloudAccountSyncTaskBriefResp) (string, string, bool, string) {
	if task == nil || task.Stats == nil {
		return "", "", false, ""
	}
	raw := task.Stats["failureDetails"]
	switch details := raw.(type) {
	case []models.ResAttrs:
		for _, detail := range details {
			if category := strings.TrimSpace(fmt.Sprint(detail["category"])); category != "" {
				return category, strings.TrimSpace(fmt.Sprint(detail["message"])), cloudAccountBoolValue(detail["retryable"]), strings.TrimSpace(fmt.Sprint(detail["retryHint"]))
			}
		}
	case []map[string]interface{}:
		for _, detail := range details {
			if category := strings.TrimSpace(fmt.Sprint(detail["category"])); category != "" {
				return category, strings.TrimSpace(fmt.Sprint(detail["message"])), cloudAccountBoolValue(detail["retryable"]), strings.TrimSpace(fmt.Sprint(detail["retryHint"]))
			}
		}
	case []interface{}:
		for _, item := range details {
			if detail, ok := item.(map[string]interface{}); ok {
				if category := strings.TrimSpace(fmt.Sprint(detail["category"])); category != "" {
					return category, strings.TrimSpace(fmt.Sprint(detail["message"])), cloudAccountBoolValue(detail["retryable"]), strings.TrimSpace(fmt.Sprint(detail["retryHint"]))
				}
			}
			if detail, ok := item.(models.ResAttrs); ok {
				if category := strings.TrimSpace(fmt.Sprint(detail["category"])); category != "" {
					return category, strings.TrimSpace(fmt.Sprint(detail["message"])), cloudAccountBoolValue(detail["retryable"]), strings.TrimSpace(fmt.Sprint(detail["retryHint"]))
				}
			}
		}
	case models.ResAttrs:
		if category := strings.TrimSpace(fmt.Sprint(details["category"])); category != "" {
			return category, strings.TrimSpace(fmt.Sprint(details["message"])), cloudAccountBoolValue(details["retryable"]), strings.TrimSpace(fmt.Sprint(details["retryHint"]))
		}
	case map[string]interface{}:
		if category := strings.TrimSpace(fmt.Sprint(details["category"])); category != "" {
			return category, strings.TrimSpace(fmt.Sprint(details["message"])), cloudAccountBoolValue(details["retryable"]), strings.TrimSpace(fmt.Sprint(details["retryHint"]))
		}
	}
	return "", "", false, ""
}

func cloudAccountBoolValue(value interface{}) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true") || strings.TrimSpace(typed) == "1"
	case int:
		return typed != 0
	case int64:
		return typed != 0
	case float64:
		return typed != 0
	default:
		return false
	}
}

func cloudAccountSyncTaskOccurredAt(task *resps.CloudAccountSyncTaskBriefResp) time.Time {
	if task == nil {
		return time.Time{}
	}
	for _, value := range []models.Time{task.EndedAt, task.StartedAt, task.CreatedAt} {
		occurredAt := time.Time(value)
		if !occurredAt.IsZero() && occurredAt.Year() > 1 {
			return occurredAt
		}
	}
	return time.Time{}
}

func cloudAccountClassifySyncFailure(message string) (string, bool, string) {
	return cmdbSyncClassifyFailure(message)
}

func cloudAccountHealthStatusForFailureCategory(category string, retryable bool) string {
	switch category {
	case "credential", "permission", "configuration":
		return models.CloudAccountHealthUnhealthy
	case "rate_limit", "network", "unknown":
		return models.CloudAccountHealthWarning
	default:
		if retryable {
			return models.CloudAccountHealthWarning
		}
		return models.CloudAccountHealthUnhealthy
	}
}

func cloudAccountFailureCategoryLabel(category string) string {
	switch category {
	case "credential":
		return "凭证异常"
	case "permission":
		return "权限异常"
	case "configuration":
		return "配置异常"
	case "rate_limit":
		return "API 限流"
	case "network":
		return "网络异常"
	case "unknown":
		return "未分类异常"
	default:
		if category == "" {
			return "未分类异常"
		}
		return category
	}
}

func formatCloudAccountDuration(value time.Duration) string {
	if value%time.Hour == 0 {
		return fmt.Sprintf("%d 小时", int(value/time.Hour))
	}
	if value%time.Minute == 0 {
		return fmt.Sprintf("%d 分钟", int(value/time.Minute))
	}
	return fmt.Sprintf("%d 秒", int(value/time.Second))
}

func cloudAccountHealthCheckOptionsFromForm(form *forms.CheckCloudAccountsHealthForm, defaultConcurrency int) (cloudAccountHealthCheckOptions, e.Error) {
	options := cloudAccountHealthCheckOptions{
		ShardTotal:  1,
		Concurrency: defaultConcurrency,
	}
	if form == nil {
		return cloudAccountHealthNormalizeOptions(options, defaultConcurrency), nil
	}
	if form.ShardIndex < 0 || form.ShardTotal < 0 || form.Concurrency < 0 {
		return cloudAccountHealthCheckOptions{}, e.New(e.BadParam, fmt.Errorf("shardIndex, shardTotal and concurrency must be non-negative"), http.StatusBadRequest)
	}
	if form.ShardTotal > 0 {
		if form.ShardTotal > cloudAccountHealthMaxShardTotal {
			return cloudAccountHealthCheckOptions{}, e.New(e.BadParam, fmt.Errorf("shardTotal cannot exceed %d", cloudAccountHealthMaxShardTotal), http.StatusBadRequest)
		}
		if form.ShardIndex >= form.ShardTotal {
			return cloudAccountHealthCheckOptions{}, e.New(e.BadParam, fmt.Errorf("shardIndex must be less than shardTotal"), http.StatusBadRequest)
		}
		options.ShardIndex = form.ShardIndex
		options.ShardTotal = form.ShardTotal
	} else if form.ShardIndex != 0 {
		return cloudAccountHealthCheckOptions{}, e.New(e.BadParam, fmt.Errorf("shardTotal is required when shardIndex is set"), http.StatusBadRequest)
	}
	if form.Concurrency > 0 {
		if form.Concurrency > cloudAccountHealthMaxConcurrency {
			return cloudAccountHealthCheckOptions{}, e.New(e.BadParam, fmt.Errorf("concurrency cannot exceed %d", cloudAccountHealthMaxConcurrency), http.StatusBadRequest)
		}
		options.Concurrency = form.Concurrency
	}
	return cloudAccountHealthNormalizeOptions(options, defaultConcurrency), nil
}

func cloudAccountHealthNormalizeOptions(options cloudAccountHealthCheckOptions, defaultConcurrency int) cloudAccountHealthCheckOptions {
	if options.ShardTotal <= 0 {
		options.ShardTotal = 1
	}
	if options.ShardTotal > cloudAccountHealthMaxShardTotal {
		options.ShardTotal = cloudAccountHealthMaxShardTotal
	}
	if options.ShardIndex < 0 {
		options.ShardIndex = 0
	}
	if options.ShardIndex >= options.ShardTotal {
		options.ShardIndex = options.ShardTotal - 1
	}
	if defaultConcurrency <= 0 {
		defaultConcurrency = cloudAccountHealthManualConcurrency
	}
	if options.Concurrency <= 0 {
		options.Concurrency = defaultConcurrency
	}
	if options.Concurrency > cloudAccountHealthMaxConcurrency {
		options.Concurrency = cloudAccountHealthMaxConcurrency
	}
	return options
}

func cloudAccountHealthWorkerOptions() cloudAccountHealthCheckOptions {
	return cloudAccountHealthCheckOptions{
		ShardTotal:  cloudAccountHealthIntEnv(cloudAccountHealthWorkerShardsEnv, cloudAccountHealthDefaultShardTotal, 1, cloudAccountHealthMaxShardTotal),
		Concurrency: cloudAccountHealthIntEnv(cloudAccountHealthWorkerConcurrencyEnv, cloudAccountHealthWorkerConcurrency, 1, cloudAccountHealthMaxConcurrency),
	}
}

func cloudAccountHealthIntEnv(env string, defaultValue int, minValue int, maxValue int) int {
	raw := strings.TrimSpace(os.Getenv(env))
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minValue {
		return defaultValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func applyCloudAccountHealthShard(query *db.Session, options cloudAccountHealthCheckOptions) *db.Session {
	if !options.sharded() {
		return query
	}
	return query.Where("cast(conv(substr(md5(id), 1, 8), 16, 10) as unsigned) % ? = ?", options.ShardTotal, options.ShardIndex)
}

func applyCloudAccountHealthShardIndexes(query *db.Session, shardTotal int, shardIndexes []int) *db.Session {
	if shardTotal <= 1 || len(shardIndexes) == 0 {
		return query
	}
	clauses := make([]string, 0, len(shardIndexes))
	args := make([]interface{}, 0, len(shardIndexes)*2)
	for _, shardIndex := range shardIndexes {
		clauses = append(clauses, "cast(conv(substr(md5(id), 1, 8), 16, 10) as unsigned) % ? = ?")
		args = append(args, shardTotal, shardIndex)
	}
	return query.Where("("+strings.Join(clauses, " or ")+")", args...)
}

func cloudAccountHealthShardIndex(id models.Id, shardTotal int) int {
	if shardTotal <= 1 {
		return 0
	}
	sum := md5.Sum([]byte(id.String()))
	return int(binary.BigEndian.Uint32(sum[:4]) % uint32(shardTotal))
}

func cloudAccountHealthWorkerInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv(cloudAccountHealthWorkerIntervalEnv))
	if raw == "" {
		return cloudAccountHealthWorkerDefaultInterval
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return cloudAccountHealthWorkerDefaultInterval
	}
	interval := time.Duration(seconds) * time.Second
	if interval < cloudAccountHealthWorkerMinInterval {
		return cloudAccountHealthWorkerMinInterval
	}
	if interval > cloudAccountHealthWorkerMaxInterval {
		return cloudAccountHealthWorkerMaxInterval
	}
	return interval
}

func cloudAccountHealthEventQuietWindow() time.Duration {
	raw := strings.TrimSpace(os.Getenv(cloudAccountHealthEventQuietEnv))
	if raw == "" {
		return cloudAccountHealthEventDefaultQuiet
	}
	hours, err := strconv.Atoi(raw)
	if err != nil || hours <= 0 {
		return cloudAccountHealthEventDefaultQuiet
	}
	window := time.Duration(hours) * time.Hour
	if window < cloudAccountHealthEventMinQuiet {
		return cloudAccountHealthEventMinQuiet
	}
	if window > cloudAccountHealthEventMaxQuiet {
		return cloudAccountHealthEventMaxQuiet
	}
	return window
}

func cloudAccountHealthShouldEmitEvent(force bool, previousStatus, previousMessage, nextStatus, nextMessage string, previousCheckedAt models.Time) bool {
	if force {
		return true
	}
	if previousStatus != nextStatus || previousMessage != nextMessage {
		return true
	}
	lastChecked := time.Time(previousCheckedAt)
	if lastChecked.IsZero() || lastChecked.Year() <= 1 {
		return true
	}
	return time.Since(lastChecked) >= cloudAccountHealthEventQuietWindow()
}

func cloudAccountHealthLockName(options cloudAccountHealthCheckOptions) string {
	if options.sharded() {
		return fmt.Sprintf("cloudiac:cloud_account_health:all:shard:%d:%d", options.ShardTotal, options.ShardIndex)
	}
	return "cloudiac:cloud_account_health:all"
}

func cloudAccountEvent(c *ctx.ServiceContext, account *models.CloudAccount, eventType string, title string, status string, message string) {
	if account == nil {
		return
	}
	level := models.CloudEventLevelInfo
	if status == models.CloudAccountValidationInvalid || status == models.CloudAccountHealthUnhealthy {
		level = models.CloudEventLevelError
	} else if status == models.CloudAccountValidationPending || status == models.CloudAccountHealthWarning {
		level = models.CloudEventLevelWarning
	}
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:          account.OrgId,
		CloudAccountId: account.Id,
		Source:         models.CloudEventSourceAccount,
		EventType:      eventType,
		Level:          level,
		Status:         status,
		Provider:       account.Provider,
		AccountId:      account.AccountId,
		ResourceType:   "cloud_account",
		ResourceId:     account.Id.String(),
		ResourceName:   account.Name,
		Title:          title,
		Message:        message,
		Payload: models.ResAttrs{
			"cloudAccountId":      account.Id.String(),
			"name":                account.Name,
			"provider":            account.Provider,
			"accountId":           account.AccountId,
			"validationStatus":    account.ValidationStatus,
			"validationMessage":   account.ValidationMessage,
			"healthStatus":        account.HealthStatus,
			"healthMessage":       account.HealthMessage,
			"lastValidatedAt":     account.LastValidatedAt,
			"lastHealthCheckedAt": account.LastHealthCheckedAt,
			"lastSyncAt":          account.LastSyncAt,
		},
	})
}

func applyCloudAccountAttrs(account *models.CloudAccount, attrs map[string]interface{}) {
	for key, value := range attrs {
		switch key {
		case "validation_status":
			account.ValidationStatus = value.(string)
		case "validation_message":
			account.ValidationMessage = value.(string)
		case "health_status":
			account.HealthStatus = value.(string)
		case "health_message":
			account.HealthMessage = value.(string)
		case "last_validated_at":
			account.LastValidatedAt = value.(models.Time)
		case "last_health_checked_at":
			account.LastHealthCheckedAt = value.(models.Time)
		case "supported_types":
			account.SupportedTypes = value.(models.StrSlice)
		case "regions":
			account.Regions = value.(models.StrSlice)
		}
	}
}

func cloudAccountRegionsResp(c *ctx.ServiceContext, account *models.CloudAccount) (*resps.CloudAccountRegionsResp, e.Error) {
	validation := validateCloudAccount(account)
	rows := make([]models.CloudAccountRegion, 0)
	if err := c.DB().Model(&models.CloudAccountRegion{}).
		Where("org_id = ? and cloud_account_id = ?", c.OrgId, account.Id).
		Order("enabled desc").
		Order("is_default desc").
		Order("region asc").
		Find(&rows); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if len(rows) > 0 {
		return cloudAccountRegionsRespFromSnapshot(account, validation, rows), nil
	}
	return cloudAccountRegionsRespFromValidation(account, validation), nil
}

func cloudAccountRegionsRespFromSnapshot(account *models.CloudAccount, validation cloudAccountValidation, rows []models.CloudAccountRegion) *resps.CloudAccountRegionsResp {
	regions := make([]resps.CloudAccountRegionResp, 0, len(rows))
	regionNames := make([]string, 0, len(rows))
	for _, row := range rows {
		item := cloudAccountRegionRespFromModel(row)
		regions = append(regions, item)
		if item.Enabled {
			regionNames = append(regionNames, item.Name)
		}
	}
	return &resps.CloudAccountRegionsResp{
		Id:                    account.Id,
		Provider:              account.Provider,
		AccountId:             account.AccountId,
		Regions:               regions,
		RegionNames:           regionNames,
		MissingCredentialKeys: validation.MissingCredentialKeys,
	}
}

func cloudAccountRegionsRespFromValidation(account *models.CloudAccount, validation cloudAccountValidation) *resps.CloudAccountRegionsResp {
	source := models.CloudAccountRegionSourceInferred
	if len(account.Regions) > 0 {
		source = models.CloudAccountRegionSourceConfigured
	}
	regions := make([]resps.CloudAccountRegionResp, 0, len(validation.Regions))
	status, message := cloudAccountRegionStatusMessage(account, validation)
	for index, region := range validation.Regions {
		regions = append(regions, resps.CloudAccountRegionResp{
			Name:          region,
			Enabled:       true,
			Default:       index == 0,
			SyncEnabled:   true,
			Source:        source,
			Status:        status,
			Message:       message,
			ResourceTypes: cloneStringSlice(validation.SupportedAssetTypes),
		})
	}
	return &resps.CloudAccountRegionsResp{
		Id:                    account.Id,
		Provider:              account.Provider,
		AccountId:             account.AccountId,
		Regions:               regions,
		RegionNames:           validation.Regions,
		MissingCredentialKeys: validation.MissingCredentialKeys,
	}
}

func cloudAccountRegionRespFromModel(row models.CloudAccountRegion) resps.CloudAccountRegionResp {
	return resps.CloudAccountRegionResp{
		Name:          row.Region,
		Enabled:       row.Enabled,
		Default:       row.IsDefault,
		SyncEnabled:   row.SyncEnabled,
		Source:        row.Source,
		Status:        row.Status,
		Message:       row.Message,
		ResourceTypes: cloneStringSlice([]string(row.ResourceTypes)),
		LastSyncAt:    row.LastSyncAt,
		Metadata:      row.Metadata,
	}
}

func syncCloudAccountRegionSnapshot(c *ctx.ServiceContext, account *models.CloudAccount, validation cloudAccountValidation) e.Error {
	if c == nil || account == nil || account.Id == "" {
		return nil
	}
	regions := normalizeStringList(validation.Regions)
	source := models.CloudAccountRegionSourceInferred
	if len(account.Regions) > 0 {
		source = models.CloudAccountRegionSourceConfigured
	}
	status, message := cloudAccountRegionStatusMessage(account, validation)

	existingRows := make([]models.CloudAccountRegion, 0)
	if err := c.DB().Model(&models.CloudAccountRegion{}).
		Where("org_id = ? and cloud_account_id = ?", c.OrgId, account.Id).
		Find(&existingRows); err != nil {
		return e.New(e.DBError, err)
	}
	existing := make(map[string]models.CloudAccountRegion, len(existingRows))
	for _, row := range existingRows {
		existing[strings.ToLower(row.Region)] = row
	}

	active := make(map[string]bool, len(regions))
	for index, region := range regions {
		key := strings.ToLower(region)
		active[key] = true
		row, exists := existing[key]
		attrs := cloudAccountRegionAttrs(account, region, index == 0, true, true, source, status, message, validation)
		if !exists {
			row = models.CloudAccountRegion{
				OrgId:          c.OrgId,
				CloudAccountId: account.Id,
				Region:         region,
			}
			row.Id = models.NewId("car")
			applyCloudAccountRegionAttrs(&row, attrs)
			if err := models.Create(c.DB(), &row); err != nil {
				if e.IsDuplicate(err) {
					if _, updateErr := c.DB().Model(&models.CloudAccountRegion{}).
						Where("org_id = ? and cloud_account_id = ? and region = ?", c.OrgId, account.Id, region).
						UpdateAttrs(attrs); updateErr != nil {
						return e.New(e.DBError, updateErr)
					}
					continue
				}
				return e.New(e.DBError, err)
			}
			continue
		}
		if _, err := c.DB().Model(&models.CloudAccountRegion{}).
			Where("id = ? and org_id = ?", row.Id, c.OrgId).
			UpdateAttrs(attrs); err != nil {
			return e.New(e.DBError, err)
		}
	}

	for _, row := range existingRows {
		if active[strings.ToLower(row.Region)] {
			continue
		}
		attrs := cloudAccountRegionAttrs(account, row.Region, false, false, false, models.CloudAccountRegionSourceConfigured,
			models.CloudAccountHealthWarning, "区域已从账号启用区域移除，不再参与同步", validation)
		if _, err := c.DB().Model(&models.CloudAccountRegion{}).
			Where("id = ? and org_id = ?", row.Id, c.OrgId).
			UpdateAttrs(attrs); err != nil {
			return e.New(e.DBError, err)
		}
	}
	return nil
}

func cloudAccountRegionAttrs(account *models.CloudAccount, region string, isDefault bool, enabled bool, syncEnabled bool, source string, status string, message string, validation cloudAccountValidation) models.Attrs {
	return models.Attrs{
		"provider":       account.Provider,
		"account_id":     account.AccountId,
		"region":         region,
		"enabled":        enabled,
		"is_default":     isDefault,
		"sync_enabled":   syncEnabled,
		"source":         source,
		"status":         status,
		"message":        message,
		"resource_types": models.StrSlice(validation.SupportedAssetTypes),
		"last_sync_at":   account.LastSyncAt,
		"metadata": models.ResAttrs{
			"validationStatus":    account.ValidationStatus,
			"missingCredentials":  cloneStringSlice(validation.MissingCredentialKeys),
			"supportedAssetCount": len(validation.SupportedAssetTypes),
		},
	}
}

func applyCloudAccountRegionAttrs(row *models.CloudAccountRegion, attrs models.Attrs) {
	row.Provider = fmt.Sprint(attrs["provider"])
	row.AccountId = fmt.Sprint(attrs["account_id"])
	row.Region = fmt.Sprint(attrs["region"])
	row.Enabled = attrs["enabled"].(bool)
	row.IsDefault = attrs["is_default"].(bool)
	row.SyncEnabled = attrs["sync_enabled"].(bool)
	row.Source = fmt.Sprint(attrs["source"])
	row.Status = fmt.Sprint(attrs["status"])
	row.Message = fmt.Sprint(attrs["message"])
	row.ResourceTypes = attrs["resource_types"].(models.StrSlice)
	row.LastSyncAt = attrs["last_sync_at"].(models.Time)
	row.Metadata = attrs["metadata"].(models.ResAttrs)
}

func cloudAccountRegionStatusMessage(account *models.CloudAccount, validation cloudAccountValidation) (string, string) {
	switch {
	case account.Status == models.CloudAccountStatusDisabled:
		return models.CloudAccountHealthWarning, "云账号已禁用，区域不会参与同步或云操作"
	case len(validation.MissingCredentialKeys) > 0:
		return models.CloudAccountHealthUnhealthy, fmt.Sprintf("缺少凭证字段：%s", strings.Join(validation.MissingCredentialKeys, ", "))
	case len(validation.Regions) == 0:
		return models.CloudAccountHealthWarning, "未配置或无法推断可用区域"
	case len(validation.SupportedAssetTypes) == 0:
		return models.CloudAccountHealthUnhealthy, fmt.Sprintf("暂不支持 provider %s 的资产采集", account.Provider)
	default:
		return models.CloudAccountHealthHealthy, "区域配置可用于同步"
	}
}

func cloudAccountPermissionItemsWithSnapshot(c *ctx.ServiceContext, account *models.CloudAccount, computed []resps.CloudAccountPermissionResp) ([]resps.CloudAccountPermissionResp, models.Time, e.Error) {
	rows := make([]models.CloudAccountPermission, 0)
	if err := c.DB().Model(&models.CloudAccountPermission{}).
		Where("org_id = ? and cloud_account_id = ?", c.OrgId, account.Id).
		Find(&rows); err != nil {
		return nil, models.Time{}, e.New(e.DBError, err)
	}
	byKey := make(map[string]models.CloudAccountPermission, len(rows))
	for _, row := range rows {
		byKey[row.PermissionKey] = row
	}

	items := make([]resps.CloudAccountPermissionResp, 0, len(computed))
	var lastCheckedAt models.Time
	for _, item := range computed {
		if row, ok := byKey[item.Key]; ok && cloudAccountPermissionSnapshotFresh(row, account) {
			item = cloudAccountPermissionRespFromModel(row)
		} else {
			item.Source = "computed"
		}
		if cloudAccountTimeAfter(item.CheckedAt, lastCheckedAt) {
			lastCheckedAt = item.CheckedAt
		}
		items = append(items, item)
	}
	return items, lastCheckedAt, nil
}

func cloudAccountPermissionSnapshotFresh(row models.CloudAccountPermission, account *models.CloudAccount) bool {
	checkedAt := time.Time(row.CheckedAt)
	if checkedAt.IsZero() || checkedAt.Year() <= 1 {
		return false
	}
	validatedAt := time.Time(account.LastValidatedAt)
	if validatedAt.IsZero() || validatedAt.Year() <= 1 {
		return true
	}
	return !checkedAt.Add(time.Second).Before(validatedAt)
}

func cloudAccountPermissionRespFromModel(row models.CloudAccountPermission) resps.CloudAccountPermissionResp {
	return resps.CloudAccountPermissionResp{
		Key:       row.PermissionKey,
		Name:      row.Name,
		Resource:  row.Resource,
		Action:    row.Action,
		Status:    row.Status,
		Message:   row.Message,
		Source:    row.Source,
		CheckedAt: row.CheckedAt,
		Evidence:  row.Evidence,
	}
}

func cloudAccountTimeAfter(value models.Time, base models.Time) bool {
	valueTime := time.Time(value)
	if valueTime.IsZero() || valueTime.Year() <= 1 {
		return false
	}
	baseTime := time.Time(base)
	return baseTime.IsZero() || valueTime.After(baseTime)
}

func saveCloudAccountPermissionSnapshot(c *ctx.ServiceContext, account *models.CloudAccount, items []resps.CloudAccountPermissionResp, checkedAt models.Time, source string) e.Error {
	if c == nil || account == nil || account.Id == "" {
		return nil
	}
	checkedTime := time.Time(checkedAt)
	if checkedTime.IsZero() || checkedTime.Year() <= 1 {
		checkedAt = models.Time(time.Now())
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = "local_precheck"
	}
	for _, item := range items {
		item.Key = strings.TrimSpace(item.Key)
		if item.Key == "" {
			continue
		}
		row := models.CloudAccountPermission{}
		err := c.DB().Model(&models.CloudAccountPermission{}).
			Where("org_id = ? and cloud_account_id = ? and permission_key = ?", c.OrgId, account.Id, item.Key).
			First(&row)
		attrs := cloudAccountPermissionAttrs(account, item, checkedAt, source)
		if err != nil && !e.IsRecordNotFound(err) {
			return e.New(e.DBError, err)
		}
		if e.IsRecordNotFound(err) {
			row = models.CloudAccountPermission{
				OrgId:          c.OrgId,
				CloudAccountId: account.Id,
				PermissionKey:  item.Key,
			}
			row.Id = models.NewId("cap")
			applyCloudAccountPermissionAttrs(&row, attrs)
			if err := models.Create(c.DB(), &row); err != nil {
				if e.IsDuplicate(err) {
					if _, updateErr := c.DB().Model(&models.CloudAccountPermission{}).
						Where("org_id = ? and cloud_account_id = ? and permission_key = ?", c.OrgId, account.Id, item.Key).
						UpdateAttrs(attrs); updateErr != nil {
						return e.New(e.DBError, updateErr)
					}
					continue
				}
				return e.New(e.DBError, err)
			}
			continue
		}
		if _, err := c.DB().Model(&models.CloudAccountPermission{}).
			Where("id = ? and org_id = ?", row.Id, c.OrgId).
			UpdateAttrs(attrs); err != nil {
			return e.New(e.DBError, err)
		}
	}
	return nil
}

func cloudAccountPermissionAttrs(account *models.CloudAccount, item resps.CloudAccountPermissionResp, checkedAt models.Time, source string) models.Attrs {
	return models.Attrs{
		"provider":       account.Provider,
		"account_id":     account.AccountId,
		"name":           item.Name,
		"resource":       item.Resource,
		"action":         item.Action,
		"status":         item.Status,
		"message":        item.Message,
		"source":         source,
		"checked_at":     checkedAt,
		"evidence":       cloudAccountPermissionEvidence(account, item),
		"permission_key": item.Key,
	}
}

func applyCloudAccountPermissionAttrs(row *models.CloudAccountPermission, attrs models.Attrs) {
	row.Provider = fmt.Sprint(attrs["provider"])
	row.AccountId = fmt.Sprint(attrs["account_id"])
	row.Name = fmt.Sprint(attrs["name"])
	row.Resource = fmt.Sprint(attrs["resource"])
	row.Action = fmt.Sprint(attrs["action"])
	row.Status = fmt.Sprint(attrs["status"])
	row.Message = fmt.Sprint(attrs["message"])
	row.Source = fmt.Sprint(attrs["source"])
	row.CheckedAt = attrs["checked_at"].(models.Time)
	row.Evidence = attrs["evidence"].(models.ResAttrs)
	row.PermissionKey = fmt.Sprint(attrs["permission_key"])
}

func cloudAccountPermissionEvidence(account *models.CloudAccount, item resps.CloudAccountPermissionResp) models.ResAttrs {
	evidence := models.ResAttrs{
		"provider":      account.Provider,
		"accountId":     account.AccountId,
		"tenantId":      account.TenantId,
		"regions":       []string(account.Regions),
		"permissionKey": item.Key,
	}
	if len(account.SupportedTypes) > 0 {
		evidence["supportedAssetTypes"] = []string(account.SupportedTypes)
	}
	return evidence
}

func cloudAccountPermissionItems(account *models.CloudAccount, validation cloudAccountValidation) []resps.CloudAccountPermissionResp {
	accountStatus := "pass"
	accountMessage := "账号已启用"
	if account.Status == models.CloudAccountStatusDisabled {
		accountStatus = "fail"
		accountMessage = "账号已禁用，不能用于同步或云操作"
	}

	credentialStatus := "pass"
	credentialMessage := "必需凭证字段完整"
	if len(validation.MissingCredentialKeys) > 0 {
		credentialStatus = "fail"
		credentialMessage = fmt.Sprintf("缺少凭证字段：%s", strings.Join(validation.MissingCredentialKeys, ", "))
	}

	regionStatus := "pass"
	regionMessage := "已配置可用区域"
	if len(validation.Regions) == 0 {
		regionStatus = "fail"
		regionMessage = "未配置或无法推断可用区域"
	}

	assetStatus := "pass"
	assetMessage := fmt.Sprintf("支持采集 %d 类资产", len(validation.SupportedAssetTypes))
	if len(validation.SupportedAssetTypes) == 0 {
		assetStatus = "fail"
		assetMessage = fmt.Sprintf("暂不支持 provider %s 的资产采集", account.Provider)
	} else if !validation.Ready {
		assetStatus = "warn"
		assetMessage = "资产采集能力已声明，但账号状态或凭证尚未满足执行条件"
	}
	operationPolicyStatus := "warn"
	operationPolicyMessage := "未配置云账号操作授权策略，云操作仅受项目角色、审批和 provider 预检控制"
	if policy := cloudAccountOperationPolicy(account); len(policy) > 0 {
		operationPolicyStatus = "pass"
		operationPolicyMessage = "已配置云账号操作授权策略，可按项目、动作、资源类型或资源标签约束云操作"
	}

	return []resps.CloudAccountPermissionResp{
		{
			Key:      "account_status",
			Name:     "账号状态",
			Resource: "cloud_account",
			Action:   "use",
			Status:   accountStatus,
			Message:  accountMessage,
		},
		{
			Key:      "credential_validation",
			Name:     "凭证完整性",
			Resource: "credential",
			Action:   "validate",
			Status:   credentialStatus,
			Message:  credentialMessage,
		},
		{
			Key:      "region_scope",
			Name:     "区域范围",
			Resource: "region",
			Action:   "list",
			Status:   regionStatus,
			Message:  regionMessage,
		},
		{
			Key:      "asset_collection",
			Name:     "资产采集",
			Resource: "cmdb_asset",
			Action:   "read",
			Status:   assetStatus,
			Message:  assetMessage,
		},
		{
			Key:      "operation_policy",
			Name:     "操作授权策略",
			Resource: "cloud_operation",
			Action:   "write",
			Status:   operationPolicyStatus,
			Message:  operationPolicyMessage,
		},
	}
}

func applyCloudAccountValidation(account *models.CloudAccount) {
	attrs := cloudAccountValidationAttrs(account)
	account.ValidationStatus = attrs["validation_status"].(string)
	account.ValidationMessage = attrs["validation_message"].(string)
	account.LastValidatedAt = attrs["last_validated_at"].(models.Time)
	account.SupportedTypes = attrs["supported_types"].(models.StrSlice)
	if regions, ok := attrs["regions"]; ok {
		account.Regions = regions.(models.StrSlice)
	}
	health := cloudAccountHealth(account)
	account.HealthStatus = health.Status
	account.HealthMessage = health.Message
	account.LastHealthCheckedAt = models.Time(time.Now())
}

func cloudAccountValidationAttrs(account *models.CloudAccount) map[string]interface{} {
	validation := validateCloudAccount(account)
	status := models.CloudAccountValidationInvalid
	if validation.Ready {
		status = models.CloudAccountValidationValid
	}
	attrs := map[string]interface{}{
		"validation_status":  status,
		"validation_message": validation.Message,
		"last_validated_at":  models.Time(time.Now()),
		"supported_types":    models.StrSlice(validation.SupportedAssetTypes),
	}
	if len(account.Regions) == 0 && len(validation.Regions) > 0 {
		attrs["regions"] = models.StrSlice(validation.Regions)
	}
	return attrs
}

func validateCloudAccount(account *models.CloudAccount) cloudAccountValidation {
	provider := normalizeProvider(account.Provider)
	credentials := credentialMapWithRegions(provider, credentialMapFromCloudCredentials(account.Credentials), []string(account.Regions))
	missing := missingCloudCredentialKeys(provider, credentials)
	regions := firstNonEmptyStringSlice([]string(account.Regions), inferCloudRegions(provider, credentials))
	supportedTypes := supportedCmdbCloudAssetTypes(provider)

	message := "cloud account credentials are valid for local checks"
	ready := len(missing) == 0 && len(supportedTypes) > 0 && account.Status != models.CloudAccountStatusDisabled
	if len(supportedTypes) == 0 {
		ready = false
		message = fmt.Sprintf("unsupported cloud provider %s", provider)
	} else if len(missing) > 0 {
		message = fmt.Sprintf("missing credentials: %s", strings.Join(missing, ", "))
	} else if account.Status == models.CloudAccountStatusDisabled {
		message = "cloud account is disabled"
	}

	return cloudAccountValidation{
		Ready:                 ready,
		Message:               message,
		MissingCredentialKeys: missing,
		SupportedAssetTypes:   supportedTypes,
		Regions:               regions,
	}
}

func marshalCloudCredentialParams(params []forms.Params, previous map[string]string) (models.JSON, error) {
	normalized := make([]forms.Params, 0, len(params))
	for _, param := range params {
		param.Key = strings.ToUpper(strings.TrimSpace(param.Key))
		if param.Key == "" {
			continue
		}
		if param.Id == "" {
			param.Id = param.Key
		}
		if param.IsSecret == nil {
			isSecret := false
			param.IsSecret = &isSecret
		}
		if *param.IsSecret {
			if param.Value == "" {
				param.Value = firstNonEmpty(previous[param.Id], previous[param.Key])
			} else {
				encrypted, err := utils.AesEncrypt(param.Value)
				if err != nil {
					return nil, err
				}
				param.Value = encrypted
			}
		}
		normalized = append(normalized, param)
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	return models.JSON(data), nil
}

func maskedCloudCredentials(credentials models.JSON) models.JSON {
	params := cloudCredentialParams(credentials)
	for index := range params {
		if cloudCredentialParamSensitive(params[index]) || (params[index].IsSecret != nil && *params[index].IsSecret) {
			params[index].Value = ""
			isSecret := true
			params[index].IsSecret = &isSecret
		}
	}
	data, _ := json.Marshal(params)
	return models.JSON(data)
}

func cloudCredentialParamSensitive(param forms.Params) bool {
	key := strings.ToUpper(strings.TrimSpace(firstNonEmpty(param.Key, param.Id)))
	if key == "" {
		return false
	}
	sensitiveParts := []string{
		"ACCESS_KEY",
		"ACCOUNT_KEY",
		"CLIENT_SECRET",
		"PASSWORD",
		"PASSPHRASE",
		"PRIVATE_KEY",
		"SECRET",
		"TOKEN",
	}
	for _, part := range sensitiveParts {
		if strings.Contains(key, part) {
			return true
		}
	}
	return false
}

func credentialMapFromCloudCredentials(credentials models.JSON) map[string]string {
	result := make(map[string]string)
	for _, param := range cloudCredentialParams(credentials) {
		key := strings.ToUpper(strings.TrimSpace(param.Key))
		if key == "" {
			continue
		}
		value := param.Value
		if param.IsSecret != nil && *param.IsSecret && value != "" {
			if decrypted, err := utils.AesDecrypt(value); err == nil {
				value = decrypted
			}
		}
		result[key] = value
	}
	return result
}

func encryptedCloudCredentialValues(credentials models.JSON) map[string]string {
	result := make(map[string]string)
	for _, param := range cloudCredentialParams(credentials) {
		result[param.Id] = param.Value
		result[strings.ToUpper(strings.TrimSpace(param.Key))] = param.Value
	}
	return result
}

func cloudCredentialParams(credentials models.JSON) []forms.Params {
	params := make([]forms.Params, 0)
	if credentials.IsNull() {
		return params
	}
	_ = json.Unmarshal(credentials, &params)
	return params
}

func credentialMapWithRegions(provider string, credentials map[string]string, regions []string) map[string]string {
	result := make(map[string]string, len(credentials)+2)
	for key, value := range credentials {
		result[key] = value
	}
	if len(regions) == 0 {
		return result
	}
	joined := strings.Join(regions, ",")
	switch provider {
	case "aws":
		result["AWS_REGIONS"] = firstNonEmpty(result["AWS_REGIONS"], joined)
		result["AWS_REGION"] = firstNonEmpty(result["AWS_REGION"], regions[0])
	case "oci":
		result["OCI_REGIONS"] = firstNonEmpty(result["OCI_REGIONS"], joined)
		result["OCI_REGION"] = firstNonEmpty(result["OCI_REGION"], regions[0])
	case "alicloud":
		result["ALICLOUD_REGIONS"] = firstNonEmpty(result["ALICLOUD_REGIONS"], joined)
		result["ALICLOUD_REGION"] = firstNonEmpty(result["ALICLOUD_REGION"], regions[0])
	case "azure":
		result["AZURE_REGIONS"] = firstNonEmpty(result["AZURE_REGIONS"], joined)
		result["AZURE_REGION"] = firstNonEmpty(result["AZURE_REGION"], regions[0])
	case "gcp":
		result["GCP_REGIONS"] = firstNonEmpty(result["GCP_REGIONS"], joined)
		result["GCP_REGION"] = firstNonEmpty(result["GCP_REGION"], regions[0])
	case "tencentcloud":
		result["TENCENTCLOUD_REGIONS"] = firstNonEmpty(result["TENCENTCLOUD_REGIONS"], joined)
		result["TENCENTCLOUD_REGION"] = firstNonEmpty(result["TENCENTCLOUD_REGION"], regions[0])
	case "huawei":
		result["HUAWEI_REGIONS"] = firstNonEmpty(result["HUAWEI_REGIONS"], joined)
		result["HUAWEI_REGION"] = firstNonEmpty(result["HUAWEI_REGION"], regions[0])
	}
	return result
}

func inferCloudAccountId(provider, accountId, tenantId string, credentials map[string]string) string {
	switch provider {
	case "aws":
		return firstNonEmpty(accountId, credentials["AWS_ACCOUNT_ID"])
	case "oci":
		return firstNonEmpty(accountId, tenantId, credentials["OCI_TENANCY_OCID"])
	case "alicloud":
		return firstNonEmpty(accountId, credentials["ALICLOUD_ACCOUNT_ID"])
	case "azure":
		return firstNonEmpty(accountId, tenantId, credentials["AZURE_SUBSCRIPTION_ID"], credentials["ARM_SUBSCRIPTION_ID"])
	case "gcp":
		return firstNonEmpty(accountId, tenantId, credentials["GCP_PROJECT_ID"], credentials["GOOGLE_CLOUD_PROJECT"])
	case "tencentcloud":
		return firstNonEmpty(accountId, tenantId, credentials["TENCENTCLOUD_ACCOUNT_ID"], credentials["TENCENT_ACCOUNT_ID"])
	case "huawei":
		return firstNonEmpty(accountId, tenantId, credentials["HUAWEI_ACCOUNT_ID"], credentials["HUAWEICLOUD_ACCOUNT_ID"])
	default:
		return firstNonEmpty(accountId, tenantId)
	}
}

func firstNonEmptyStringSlice(values ...[]string) []string {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}
