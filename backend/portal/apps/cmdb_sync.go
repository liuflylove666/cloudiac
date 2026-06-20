// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/libs/page"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	"cloudiac/portal/models/resps"
	"cloudiac/utils"
	"cloudiac/utils/logs"
)

type cmdbCloudAccount struct {
	Id                    models.Id
	Source                string
	Name                  string
	Description           string
	Provider              string
	AccountId             string
	Regions               []string
	Ready                 bool
	MissingCredentialKeys []string
	SupportedAssetTypes   []string
	UpdatedAt             models.Time
	Credentials           map[string]string
}

func SearchCmdbCloudAccounts(c *ctx.ServiceContext) ([]resps.CmdbCloudAccountResp, e.Error) {
	accounts, err := discoverCmdbCloudAccounts(c)
	if err != nil {
		return nil, err
	}

	resp := make([]resps.CmdbCloudAccountResp, 0, len(accounts))
	for _, account := range accounts {
		resp = append(resp, resps.CmdbCloudAccountResp{
			Id:                    account.Id,
			Source:                account.Source,
			Name:                  account.Name,
			Description:           account.Description,
			Provider:              account.Provider,
			AccountId:             account.AccountId,
			Regions:               account.Regions,
			Ready:                 account.Ready,
			MissingCredentialKeys: account.MissingCredentialKeys,
			SupportedAssetTypes:   account.SupportedAssetTypes,
			UpdatedAt:             account.UpdatedAt,
		})
	}
	return resp, nil
}

func SearchCmdbSyncTasks(c *ctx.ServiceContext, form *forms.SearchCmdbSyncTaskForm) (interface{}, e.Error) {
	query := c.DB().Model(&models.CmdbSyncTask{}).Where("org_id = ?", c.OrgId)
	if form.Provider != "" {
		query = query.Where("provider = ?", form.Provider)
	}
	if form.AccountSource != "" {
		query = query.Where("account_source = ?", form.AccountSource)
	}
	if form.AccountId != "" {
		query = query.Where("account_id = ?", form.AccountId)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	if form.SortField() == "" {
		query = query.Order("created_at desc")
	} else {
		query = form.Order(query)
	}

	tasks := make([]resps.CmdbSyncTaskResp, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&tasks); err != nil {
		return nil, e.New(e.DBError, err)
	}

	return &page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     tasks,
	}, nil
}

func CmdbSyncTaskDetail(c *ctx.ServiceContext, form *forms.CmdbSyncTaskParam) (*resps.CmdbSyncTaskDetailResp, e.Error) {
	task := models.CmdbSyncTask{}
	if err := c.DB().Model(&models.CmdbSyncTask{}).
		Where("id = ? and org_id = ?", form.Id, c.OrgId).
		First(&task); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}

	logs := make([]resps.CmdbSyncTaskLogResp, 0)
	if err := c.DB().Model(&models.CmdbSyncTaskLog{}).
		Where("org_id = ? and task_id = ?", c.OrgId, form.Id).
		Order("created_at asc").
		Scan(&logs); err != nil {
		return nil, e.New(e.DBError, err)
	}

	return &resps.CmdbSyncTaskDetailResp{
		CmdbSyncTask: task,
		Logs:         logs,
	}, nil
}

func StartCmdbSyncTask(c *ctx.ServiceContext, form *forms.CreateCmdbSyncTaskForm) (*resps.CmdbSyncTaskResp, e.Error) {
	account, err := findCmdbCloudAccount(c, form.AccountSource, form.AccountId)
	if err != nil {
		return nil, err
	}
	if form.Provider != "" {
		account.Provider = normalizeProvider(form.Provider)
	}
	if account.Provider == "" {
		return nil, e.New(e.BadParam, fmt.Errorf("cloud provider cannot be inferred from account %s", account.Name), http.StatusBadRequest)
	}

	regions := normalizeStringList(form.Regions)
	if len(regions) == 0 {
		regions = account.Regions
	}
	assetTypes := normalizeStringList(form.AssetTypes)
	if len(assetTypes) == 0 {
		assetTypes = account.SupportedAssetTypes
	}

	now := models.Time(time.Now())
	task := &models.CmdbSyncTask{
		OrgId:         c.OrgId,
		AccountSource: account.Source,
		AccountId:     account.Id,
		AccountName:   account.Name,
		Provider:      account.Provider,
		Regions:       models.StrSlice(regions),
		AssetTypes:    models.StrSlice(assetTypes),
		Status:        models.CmdbSyncTaskRunning,
		Stats:         initialCmdbSyncStats(regions, assetTypes),
		StartedAt:     now,
	}
	task.Id = models.NewId("cst")
	if err := models.Create(c.DB(), task); err != nil {
		return nil, e.New(e.DBError, err)
	}
	appendCmdbSyncTaskLog(c, task.Id, models.CmdbSyncLogLevelInfo, "created", "云采集任务已创建", models.ResAttrs{
		"accountSource": task.AccountSource,
		"accountId":     task.AccountId,
		"accountName":   task.AccountName,
		"provider":      task.Provider,
		"regions":       regions,
		"assetTypes":    assetTypes,
	})

	go runCmdbSyncTask(task.Id, c, cloneCmdbCloudAccount(account), cloneStringSlice(regions), cloneStringSlice(assetTypes))

	return &resps.CmdbSyncTaskResp{CmdbSyncTask: *task}, nil
}

func runCmdbSyncTask(taskId models.Id, requestCtx *ctx.ServiceContext, account *cmdbCloudAccount, regions, assetTypes []string) {
	workerCtx := &ctx.ServiceContext{
		UserId:       requestCtx.UserId,
		OrgId:        requestCtx.OrgId,
		ProjectId:    requestCtx.ProjectId,
		IsSuperAdmin: requestCtx.IsSuperAdmin,
	}
	logger := logs.Get().WithField("cmdbSyncTaskId", taskId)
	stats := initialCmdbSyncStats(regions, assetTypes)
	status := models.CmdbSyncTaskComplete
	errorMessage := ""

	defer func() {
		if recovered := recover(); recovered != nil {
			status = models.CmdbSyncTaskFailed
			errorMessage = fmt.Sprintf("cmdb sync task panic: %v", recovered)
			stats["panic"] = fmt.Sprintf("%v", recovered)
			appendCmdbSyncTaskLog(workerCtx, taskId, models.CmdbSyncLogLevelError, "panic", errorMessage, models.ResAttrs{
				"stack": string(debug.Stack()),
			})
			logger.Errorf("%s\n%s", errorMessage, string(debug.Stack()))
		}
		endedAt := models.Time(time.Now())
		stats["stage"] = status
		if err := updateCmdbSyncTask(workerCtx, taskId, map[string]interface{}{
			"status":        status,
			"error_message": errorMessage,
			"stats":         stats,
			"ended_at":      endedAt,
		}); err != nil {
			logger.Errorf("update cmdb sync task finished status error: %v", err)
		}
		if account.Source == models.CmdbCloudAccountSourceCloudAccount && status == models.CmdbSyncTaskComplete {
			if err := updateCloudAccountLastSyncAt(workerCtx, account.Id, endedAt); err != nil {
				logger.Errorf("update cloud account last sync time error: %v", err)
			}
		}
		level := models.CmdbSyncLogLevelInfo
		message := "云采集任务完成"
		if status == models.CmdbSyncTaskFailed {
			level = models.CmdbSyncLogLevelError
			message = "云采集任务失败"
		}
		appendCmdbSyncTaskLog(workerCtx, taskId, level, status, message, models.ResAttrs{
			"status":       status,
			"errorMessage": errorMessage,
			"stats":        stats,
		})
	}()

	startedAt := models.Time(time.Now())
	stats["stage"] = models.CmdbSyncTaskRunning
	if err := updateCmdbSyncTask(workerCtx, taskId, map[string]interface{}{
		"status":     models.CmdbSyncTaskRunning,
		"started_at": startedAt,
		"stats":      stats,
	}); err != nil {
		logger.Errorf("update cmdb sync task running status error: %v", err)
	}
	appendCmdbSyncTaskLog(workerCtx, taskId, models.CmdbSyncLogLevelInfo, "running", "后台采集开始", models.ResAttrs{
		"regions":    regions,
		"assetTypes": assetTypes,
	})

	stats["stage"] = "collecting"
	_ = updateCmdbSyncTask(workerCtx, taskId, map[string]interface{}{"stats": stats})
	appendCmdbSyncTaskLog(workerCtx, taskId, models.CmdbSyncLogLevelInfo, "collecting", "开始调用云厂商采集器", models.ResAttrs{
		"provider": account.Provider,
	})
	runStats, runErr := runCmdbCloudCollector(workerCtx, account, regions, assetTypes)
	if runStats != nil {
		stats = runStats
	}
	if runErr != nil {
		status = models.CmdbSyncTaskFailed
		errorMessage = runErr.Error()
		appendCmdbSyncTaskLog(workerCtx, taskId, models.CmdbSyncLogLevelError, "collector", errorMessage, models.ResAttrs{
			"stats": stats,
		})
	} else {
		appendCmdbSyncTaskLog(workerCtx, taskId, models.CmdbSyncLogLevelInfo, "collector", "云厂商采集器执行完成", models.ResAttrs{
			"stats": stats,
		})
	}
}

func updateCmdbSyncTask(c *ctx.ServiceContext, taskId models.Id, attrs map[string]interface{}) error {
	_, err := c.DB().Model(&models.CmdbSyncTask{}).
		Where("id = ? and org_id = ?", taskId, c.OrgId).
		UpdateAttrs(attrs)
	return err
}

func updateCloudAccountLastSyncAt(c *ctx.ServiceContext, accountId models.Id, lastSyncAt models.Time) error {
	_, err := c.DB().Model(&models.CloudAccount{}).
		Where("id = ? and org_id = ?", accountId, c.OrgId).
		UpdateAttrs(map[string]interface{}{
			"last_sync_at": lastSyncAt,
		})
	return err
}

func appendCmdbSyncTaskLog(c *ctx.ServiceContext, taskId models.Id, level, stage, message string, data models.ResAttrs) {
	if c == nil || taskId == "" {
		return
	}
	log := &models.CmdbSyncTaskLog{
		OrgId:   c.OrgId,
		TaskId:  taskId,
		Level:   level,
		Stage:   stage,
		Message: message,
		Data:    data,
	}
	log.Id = models.NewId("csl")
	if err := models.Create(c.DB(), log); err != nil {
		logs.Get().WithField("cmdbSyncTaskId", taskId).Warnf("append cmdb sync task log failed: %v", err)
	}
}

func runCmdbCloudCollector(c *ctx.ServiceContext, account *cmdbCloudAccount, regions, assetTypes []string) (models.ResAttrs, error) {
	stats := initialCmdbSyncStats(regions, assetTypes)
	if !account.Ready {
		stats["missingCredentialKeys"] = account.MissingCredentialKeys
		return stats, fmt.Errorf("cloud account %s missing credentials: %s", account.Name, strings.Join(account.MissingCredentialKeys, ", "))
	}

	result := collectCmdbCloudAssets(account, regions, assetTypes)
	mergeCmdbStats(stats, result.Stats)
	stats["collected"] = len(result.Assets)

	upsertResp, upsertErr := upsertCmdbCloudAssets(c, result.Assets)
	if upsertErr != nil {
		return stats, upsertErr
	}
	if upsertResp != nil {
		stats["created"] = upsertResp.Created
		stats["updated"] = upsertResp.Updated
		stats["skipped"] = upsertResp.Skipped
	}
	return stats, result.Err
}

func initialCmdbSyncStats(regions, assetTypes []string) models.ResAttrs {
	return models.ResAttrs{
		"collected":  0,
		"created":    0,
		"updated":    0,
		"skipped":    0,
		"regions":    cloneStringSlice(regions),
		"assetTypes": cloneStringSlice(assetTypes),
	}
}

func cloneCmdbCloudAccount(account *cmdbCloudAccount) *cmdbCloudAccount {
	if account == nil {
		return nil
	}
	cloned := *account
	cloned.Regions = cloneStringSlice(account.Regions)
	cloned.MissingCredentialKeys = cloneStringSlice(account.MissingCredentialKeys)
	cloned.SupportedAssetTypes = cloneStringSlice(account.SupportedAssetTypes)
	cloned.Credentials = make(map[string]string, len(account.Credentials))
	for key, value := range account.Credentials {
		cloned.Credentials[key] = value
	}
	return &cloned
}

func cloneStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func discoverCmdbCloudAccounts(c *ctx.ServiceContext) ([]*cmdbCloudAccount, e.Error) {
	accounts := make([]*cmdbCloudAccount, 0)

	cloudAccounts := make([]models.CloudAccount, 0)
	if err := c.DB().Model(&models.CloudAccount{}).
		Where("org_id = ? and status = ?", c.OrgId, models.CloudAccountStatusEnabled).
		Find(&cloudAccounts); err != nil {
		return nil, e.New(e.DBError, err)
	}
	for _, account := range cloudAccounts {
		credentials := credentialMapFromCloudCredentials(account.Credentials)
		provider := normalizeProvider(account.Provider)
		if provider == "" {
			provider = inferCloudProvider("", credentialKeys(credentials))
		}
		if provider == "" {
			continue
		}
		credentials = credentialMapWithRegions(provider, credentials, []string(account.Regions))
		accounts = append(accounts, buildCmdbCloudAccount(account.Id, models.CmdbCloudAccountSourceCloudAccount,
			account.Name, account.Description, provider, account.AccountId, account.UpdatedAt, []string(account.Regions), credentials))
	}

	varGroups := make([]models.VariableGroup, 0)
	if err := c.DB().Model(&models.VariableGroup{}).
		Where("org_id = ? and type = ?", c.OrgId, "environment").
		Find(&varGroups); err != nil {
		return nil, e.New(e.DBError, err)
	}
	for _, vg := range varGroups {
		credentials := credentialMapFromVarGroup(vg.Variables)
		provider := inferCloudProvider(vg.Provider, credentialKeys(credentials))
		if provider == "" {
			continue
		}
		accounts = append(accounts, buildCmdbCloudAccount(vg.Id, models.CmdbCloudAccountSourceVariableGroup,
			vg.Name, "", provider, "", vg.UpdatedAt, nil, credentials))
	}

	resourceAccounts := make([]models.ResourceAccount, 0)
	if err := c.DB().Model(&models.ResourceAccount{}).
		Where("org_id = ? and status = ?", c.OrgId, models.Enable).
		Find(&resourceAccounts); err != nil {
		return nil, e.New(e.DBError, err)
	}
	for _, account := range resourceAccounts {
		credentials := credentialMapFromResourceAccount(account.Params)
		provider := inferCloudProvider("", credentialKeys(credentials))
		if provider == "" {
			continue
		}
		accounts = append(accounts, buildCmdbCloudAccount(account.Id, models.CmdbCloudAccountSourceResourceAccount,
			account.Name, account.Description, provider, "", account.UpdatedAt, nil, credentials))
	}

	sort.SliceStable(accounts, func(i, j int) bool {
		if accounts[i].Provider == accounts[j].Provider {
			return accounts[i].Name < accounts[j].Name
		}
		return accounts[i].Provider < accounts[j].Provider
	})
	return accounts, nil
}

func findCmdbCloudAccount(c *ctx.ServiceContext, source string, id models.Id) (*cmdbCloudAccount, e.Error) {
	accounts, err := discoverCmdbCloudAccounts(c)
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		if account.Source == source && account.Id == id {
			return account, nil
		}
	}
	return nil, e.New(e.ObjectNotExistsOrNoPerm, fmt.Errorf("cloud account %s/%s not found", source, id), http.StatusNotFound)
}

func buildCmdbCloudAccount(id models.Id, source, name, description, provider, accountId string, updatedAt models.Time, configuredRegions []string, credentials map[string]string) *cmdbCloudAccount {
	provider = normalizeProvider(provider)
	missing := missingCloudCredentialKeys(provider, credentials)
	regions := firstNonEmptyStringSlice(normalizeStringList(configuredRegions), inferCloudRegions(provider, credentials))
	return &cmdbCloudAccount{
		Id:                    id,
		Source:                source,
		Name:                  name,
		Description:           description,
		Provider:              provider,
		AccountId:             inferCloudAccountId(provider, accountId, "", credentials),
		Regions:               regions,
		Ready:                 len(missing) == 0,
		MissingCredentialKeys: missing,
		SupportedAssetTypes:   supportedCmdbCloudAssetTypes(provider),
		UpdatedAt:             updatedAt,
		Credentials:           credentials,
	}
}

func credentialMapFromVarGroup(vars models.VarGroupVariables) map[string]string {
	credentials := make(map[string]string)
	for _, item := range vars {
		value := item.Value
		if item.Sensitive && value != "" {
			if decrypted, err := utils.DecryptSecretVarForce(value); err == nil {
				value = decrypted
			}
		}
		credentials[strings.ToUpper(strings.TrimSpace(item.Name))] = value
	}
	return credentials
}

func credentialMapFromResourceAccount(params models.JSON) map[string]string {
	credentials := make(map[string]string)
	if params.IsNull() {
		return credentials
	}

	items := make([]forms.Params, 0)
	if err := json.Unmarshal(params, &items); err != nil {
		return credentials
	}
	for _, item := range items {
		value := item.Value
		if item.IsSecret != nil && *item.IsSecret && value != "" {
			if decrypted, err := utils.AesDecrypt(value); err == nil {
				value = decrypted
			}
		}
		credentials[strings.ToUpper(strings.TrimSpace(item.Key))] = value
	}
	return credentials
}

func credentialKeys(credentials map[string]string) []string {
	keys := make([]string, 0, len(credentials))
	for key := range credentials {
		keys = append(keys, key)
	}
	return keys
}

func inferCloudProvider(provider string, keys []string) string {
	provider = normalizeProvider(provider)
	if provider != "" {
		return provider
	}
	keySet := make(map[string]bool)
	for _, key := range keys {
		keySet[strings.ToUpper(key)] = true
	}
	switch {
	case keySet["AWS_ACCESS_KEY_ID"] || keySet["AWS_SECRET_ACCESS_KEY"] || keySet["AWS_REGION"]:
		return "aws"
	case keySet["OCI_TENANCY_OCID"] || keySet["OCI_USER_OCID"] || keySet["OCI_FINGERPRINT"] || keySet["OCI_PRIVATE_KEY"]:
		return "oci"
	case keySet["ALICLOUD_ACCESS_KEY"] || keySet["ALICLOUD_SECRET_KEY"] || keySet["ALICLOUD_REGION"]:
		return "alicloud"
	default:
		return ""
	}
}

func normalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "oracle", "oracle_cloud", "oraclecloud":
		return "oci"
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

func inferCloudRegions(provider string, credentials map[string]string) []string {
	keys := map[string][]string{
		"aws":      {"AWS_REGIONS", "AWS_REGION", "AWS_DEFAULT_REGION"},
		"oci":      {"OCI_REGIONS", "OCI_REGION"},
		"alicloud": {"ALICLOUD_REGIONS", "ALICLOUD_REGION"},
	}
	regions := make([]string, 0)
	for _, key := range keys[provider] {
		regions = append(regions, splitListValue(credentials[key])...)
	}
	return dedupeStrings(regions)
}

func missingCloudCredentialKeys(provider string, credentials map[string]string) []string {
	missing := make([]string, 0)
	for _, key := range requiredCloudCredentialKeys(provider) {
		if strings.TrimSpace(credentials[key]) == "" {
			missing = append(missing, key)
		}
	}
	for _, group := range requiredCloudCredentialKeyGroups(provider) {
		if hasAnyCredential(credentials, group...) {
			continue
		}
		missing = append(missing, strings.Join(group, "|"))
	}
	return missing
}

func requiredCloudCredentialKeys(provider string) []string {
	required := map[string][]string{
		"aws":      {"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"},
		"oci":      {"OCI_TENANCY_OCID", "OCI_USER_OCID", "OCI_FINGERPRINT", "OCI_PRIVATE_KEY"},
		"alicloud": {"ALICLOUD_ACCESS_KEY", "ALICLOUD_SECRET_KEY", "ALICLOUD_REGION"},
	}
	return required[provider]
}

func requiredCloudCredentialKeyGroups(provider string) [][]string {
	switch provider {
	case "oci":
		return [][]string{{"OCI_REGION", "OCI_REGIONS"}}
	default:
		return nil
	}
}

func hasAnyCredential(credentials map[string]string, keys ...string) bool {
	for _, key := range keys {
		if strings.TrimSpace(credentials[key]) != "" {
			return true
		}
	}
	return false
}

func supportedCmdbCloudAssetTypes(provider string) []string {
	switch provider {
	case "aws", "oci":
		return []string{
			models.CmdbAssetTypeComputeInstance,
			models.CmdbAssetTypeKubernetesCluster,
			models.CmdbAssetTypeNetworkVpc,
			models.CmdbAssetTypeNetworkSubnet,
			models.CmdbAssetTypeNetworkRouteTable,
			models.CmdbAssetTypeNetworkSecurityGroup,
			models.CmdbAssetTypePublicIP,
			models.CmdbAssetTypeLoadBalancer,
			models.CmdbAssetTypeBlockVolume,
			models.CmdbAssetTypeObjectStorageBucket,
			models.CmdbAssetTypeRelationalDatabase,
			models.CmdbAssetTypeRedisCache,
		}
	case "alicloud":
		return []string{
			models.CmdbAssetTypeComputeInstance,
			models.CmdbAssetTypeNetworkVpc,
			models.CmdbAssetTypeNetworkSubnet,
			models.CmdbAssetTypeLoadBalancer,
			models.CmdbAssetTypeRelationalDatabase,
			models.CmdbAssetTypeRedisCache,
			models.CmdbAssetTypeObjectStorageBucket,
		}
	default:
		return []string{}
	}
}

func normalizeStringList(values []string) []string {
	items := make([]string, 0)
	for _, value := range values {
		items = append(items, splitListValue(value)...)
	}
	return dedupeStrings(items)
}

func splitListValue(value string) []string {
	items := make([]string, 0)
	for _, item := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t' || r == ' '
	}) {
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func dedupeStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
