// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	"cloudiac/portal/services"
	"cloudiac/utils/logs"
)

type cloudCostAssetIndex struct {
	ByNativeId         map[string]models.CmdbAsset
	ByNativeIdLower    map[string]models.CmdbAsset
	ByResourceRes      map[string]models.CmdbAsset
	ByResourceResLower map[string]models.CmdbAsset
}

type cloudCostAggRow struct {
	Amount float64 `gorm:"column:amount"`
	Count  int64   `gorm:"column:count"`
}

type cloudCostExportFile struct {
	URL          string
	Key          string
	Cursor       string
	Size         int64
	ETag         string
	LastModified string
}

type cloudCostPullIndexResult struct {
	Records        []forms.ImportCloudCostRecordItem
	Files          []models.ResAttrs
	SkippedFiles   []models.ResAttrs
	Cursor         string
	NextCursor     string
	SourceIndexURL string
}

type cloudCostObjectStorageConfig struct {
	Provider string
	Endpoint string
	Bucket   string
	Prefix   string
	BaseURL  string
	Region   string
	Account  *cmdbCloudAccount
}

const (
	cloudCostSyncScheduleDefaultInterval = 24 * time.Hour
	cloudCostSyncScheduleMinInterval     = 60 * time.Second
	cloudCostSyncScheduleMaxInterval     = 30 * 24 * time.Hour
	cloudCostSyncScheduleWorkerDefault   = 5 * time.Minute
	cloudCostSyncScheduleWorkerMin       = 30 * time.Second
	cloudCostSyncScheduleWorkerMax       = 24 * time.Hour
	cloudCostSyncScheduleMaxSilenceMins  = 7 * 24 * 60
	cloudCostSyncScheduleMaxEscalateAt   = 100
)

func CloudCostSummary(c *ctx.ServiceContext, form *forms.CloudCostSummaryForm) (*resps.CloudCostSummaryResp, e.Error) {
	if err := RefreshCloudCostRecords(c); err != nil {
		return nil, err
	}
	period := firstNonEmpty(form.Period, time.Now().Format("2006-01"))
	currency := firstNonEmpty(form.Currency, "CNY")
	base := cloudCostBaseQuery(c).Where("period = ? and currency = ?", period, currency)

	agg := cloudCostAggRow{}
	if err := base.Select("coalesce(sum(amount), 0) as amount, count(*) as count").Scan(&agg); err != nil {
		return nil, e.New(e.DBError, err)
	}
	matched, err := cloudOverviewCount(cloudCostBaseQuery(c).Where("period = ? and currency = ? and matched_asset = ?", period, currency, true))
	if err != nil {
		return nil, e.New(e.DBError, err)
	}
	unmatched, err := cloudOverviewCount(cloudCostBaseQuery(c).Where("period = ? and currency = ? and matched_asset = ?", period, currency, false))
	if err != nil {
		return nil, e.New(e.DBError, err)
	}

	resp := &resps.CloudCostSummaryResp{
		Period:         period,
		TotalAmount:    agg.Amount,
		Currency:       currency,
		RecordCount:    agg.Count,
		MatchedCount:   matched,
		UnmatchedCount: unmatched,
	}
	var eerr e.Error
	if resp.Providers, eerr = cloudCostGroups(c, period, currency, "provider"); eerr != nil {
		return nil, eerr
	}
	if resp.Projects, eerr = cloudCostGroups(c, period, currency, "project_id"); eerr != nil {
		return nil, eerr
	}
	if resp.Applications, eerr = cloudCostGroups(c, period, currency, "application"); eerr != nil {
		return nil, eerr
	}
	if resp.BusinessLines, eerr = cloudCostGroups(c, period, currency, "business_line"); eerr != nil {
		return nil, eerr
	}
	if resp.CostCenters, eerr = cloudCostGroups(c, period, currency, "cost_center"); eerr != nil {
		return nil, eerr
	}
	return resp, nil
}

func CloudCostTrends(c *ctx.ServiceContext, form *forms.CloudCostTrendForm) ([]resps.CloudCostTrendResp, e.Error) {
	if err := RefreshCloudCostRecords(c); err != nil {
		return nil, err
	}
	months := form.Months
	if months <= 0 || months > 24 {
		months = 6
	}
	currency := firstNonEmpty(form.Currency, "CNY")
	minPeriod := time.Now().AddDate(0, -months+1, 0).Format("2006-01")
	query := cloudCostBaseQuery(c).
		Select("period, coalesce(sum(amount), 0) as amount, currency").
		Where("currency = ? and period >= ?", currency, minPeriod).
		Group("period, currency").
		Order("period asc")
	if form.Provider != "" {
		query = query.Where("provider = ?", form.Provider)
	}
	trends := make([]resps.CloudCostTrendResp, 0)
	if err := query.Scan(&trends); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return trends, nil
}

func SearchCloudCostRecords(c *ctx.ServiceContext, form *forms.SearchCloudCostRecordForm) (interface{}, e.Error) {
	if err := RefreshCloudCostRecords(c); err != nil {
		return nil, err
	}
	query := applyCloudCostSearch(cloudCostBaseQuery(c), form)
	if form.SortField() == "" {
		query = query.Order("period desc").Order("amount desc").Order("created_at desc")
	} else {
		query = form.Order(query)
	}

	records := make([]models.CloudCostRecord, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&records); err != nil {
		return nil, e.New(e.DBError, err)
	}
	list := make([]resps.CloudCostRecordResp, 0, len(records))
	for _, record := range records {
		list = append(list, cloudCostRecordResp(c, record))
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func SearchUnmatchedCloudCostRecords(c *ctx.ServiceContext, form *forms.SearchCloudCostRecordForm) (interface{}, e.Error) {
	matched := false
	form.MatchedAsset = &matched
	return SearchCloudCostRecords(c, form)
}

func ImportCloudCostRecords(c *ctx.ServiceContext, form *forms.ImportCloudCostRecordForm) (*resps.CloudCostImportResp, e.Error) {
	if len(form.Records) == 0 {
		return nil, e.New(e.BadParam, fmt.Errorf("成本导入记录不能为空"))
	}
	index, err := cloudCostAssetIndexForOrg(c)
	if err != nil {
		return nil, err
	}
	resp := &resps.CloudCostImportResp{}
	providers := map[string]bool{}
	sources := map[string]bool{}
	periods := map[string]bool{}
	currencies := map[string]bool{}

	for idx, item := range form.Records {
		record := cloudCostRecordFromImportItem(c, form, item, idx)
		asset, matched := lookupCloudCostAsset(index, record.ResourceId)
		if matched {
			mergeCloudCostAsset(&record, asset)
		}
		record.MatchedAsset = matched
		if err := upsertCloudCostRecord(c, record); err != nil {
			return nil, err
		}
		resp.Imported++
		if matched {
			resp.MatchedCount++
		} else {
			resp.UnmatchedCount++
		}
		providers[record.Provider] = true
		sources[record.Source] = true
		periods[record.Period] = true
		currencies[record.Currency] = true
	}

	resp.Providers = sortedCloudCostKeys(providers)
	resp.Sources = sortedCloudCostKeys(sources)
	resp.Periods = sortedCloudCostKeys(periods)
	resp.Currencies = sortedCloudCostKeys(currencies)
	for period := range periods {
		for currency := range currencies {
			if err := RefreshCloudCostInsights(c, period, currency); err != nil {
				return nil, err
			}
		}
	}
	return resp, nil
}

func PullCloudCostRecords(c *ctx.ServiceContext, form *forms.PullCloudCostRecordForm) (*resps.CloudCostPullResp, e.Error) {
	provider := normalizeProvider(form.Provider)
	period := firstNonEmpty(form.Period, time.Now().Format("2006-01"))
	currency := firstNonEmpty(form.Currency, "USD")
	source := cloudCostPullSource(provider, form.Source)
	account, err := cloudCostPullCloudAccount(c, form)
	if err != nil {
		recordCloudCostPullEvent(c, form, account, provider, source, "", "", nil, fmt.Errorf("%v", err))
		return nil, err
	}

	mode := "export_url"
	sourceObjectConfig := cloudCostPullSourceObjectConfig(provider, form, account)
	sourceIndexURL := cloudCostPullSourceIndexURL(provider, form, account)
	sourceURL := cloudCostPullSourceURL(provider, form, account)
	var records []forms.ImportCloudCostRecordItem
	var indexResult *cloudCostPullIndexResult
	var pullErr error
	if sourceObjectConfig.Provider != "" {
		mode = "export_object_list"
		indexResult, pullErr = cloudCostPullFromObjectStorageList(c, form, provider, period, currency, source, sourceObjectConfig)
		if indexResult != nil {
			records = indexResult.Records
		}
	} else if sourceIndexURL != "" {
		mode = "export_index"
		indexResult, pullErr = cloudCostPullFromIndex(c, form, provider, period, currency, source, sourceIndexURL)
		if indexResult != nil {
			records = indexResult.Records
		}
	} else if sourceURL != "" {
		records, pullErr = cloudCostPullFromURL(c, form, provider, period, currency, source, sourceURL)
	} else {
		switch provider {
		case "azure":
			mode = "azure_cost_management"
			source = models.CloudCostSourceAzureCostManagement
			records, pullErr = cloudCostPullAzureCostManagement(account, period, currency)
		case "aws":
			pullErr = fmt.Errorf("aws cost pull requires sourceUrl or AWS_CUR_EXPORT_URL/AWS_COST_EXPLORER_EXPORT_URL/AWS_COST_EXPORT_URL on the cloud account")
		case "oci":
			pullErr = fmt.Errorf("oci cost pull requires sourceUrl or OCI_USAGE_COST_EXPORT_URL/OCI_COST_EXPORT_URL on the cloud account")
		case "gcp":
			pullErr = fmt.Errorf("gcp cost pull requires sourceUrl or GCP_BILLING_EXPORT_URL/GCP_COST_EXPORT_URL on the cloud account")
		case "tencentcloud":
			pullErr = fmt.Errorf("tencentcloud cost pull requires sourceUrl or TENCENTCLOUD_BILLING_EXPORT_URL/TENCENTCLOUD_COST_EXPORT_URL on the cloud account")
		case "huawei":
			pullErr = fmt.Errorf("huawei cost pull requires sourceUrl or HUAWEI_BILLING_EXPORT_URL/HUAWEI_COST_EXPORT_URL on the cloud account")
		default:
			pullErr = fmt.Errorf("unsupported cloud cost pull provider %s", provider)
		}
	}
	if pullErr != nil {
		recordCloudCostPullEvent(c, form, account, provider, source, mode, firstNonEmpty(sourceObjectConfig.Endpoint, sourceIndexURL, sourceURL), nil, pullErr)
		return nil, e.New(e.BadParam, pullErr)
	}
	if len(records) == 0 && mode != "export_index" && mode != "export_object_list" {
		pullErr = fmt.Errorf("成本拉取结果为空")
		recordCloudCostPullEvent(c, form, account, provider, source, mode, sourceURL, nil, pullErr)
		return nil, e.New(e.BadParam, pullErr)
	}

	importResp := &resps.CloudCostImportResp{}
	if len(records) > 0 {
		var importErr e.Error
		importResp, importErr = ImportCloudCostRecords(c, &forms.ImportCloudCostRecordForm{
			Provider:       provider,
			CloudAccountId: form.CloudAccountId,
			AccountId:      firstNonEmpty(form.AccountId, account.AccountId),
			Region:         form.Region,
			Period:         period,
			Currency:       currency,
			Source:         source,
			Records:        records,
		})
		if importErr != nil {
			recordCloudCostPullEvent(c, form, account, provider, source, mode, firstNonEmpty(sourceObjectConfig.Endpoint, sourceIndexURL, sourceURL), nil, fmt.Errorf("%v", importErr))
			return nil, importErr
		}
	}
	resp := &resps.CloudCostPullResp{
		CloudCostImportResp:  *importResp,
		Provider:             provider,
		Mode:                 mode,
		Source:               source,
		SourceURL:            sourceURL,
		SourceIndexURL:       sourceIndexURL,
		SourceObjectProvider: sourceObjectConfig.Provider,
		SourceObjectEndpoint: sourceObjectConfig.Endpoint,
		SourceObjectBucket:   sourceObjectConfig.Bucket,
		SourceObjectPrefix:   sourceObjectConfig.Prefix,
		CloudAccountId:       form.CloudAccountId,
		AccountId:            firstNonEmpty(form.AccountId, account.AccountId),
		Period:               period,
		Currency:             currency,
	}
	if indexResult != nil {
		resp.Cursor = indexResult.Cursor
		resp.NextCursor = indexResult.NextCursor
		resp.FileCount = len(indexResult.Files)
		resp.Files = indexResult.Files
		resp.SkippedFiles = indexResult.SkippedFiles
	}
	recordCloudCostPullEvent(c, form, account, provider, source, mode, firstNonEmpty(sourceObjectConfig.Endpoint, sourceIndexURL, sourceURL), resp, nil)
	return resp, nil
}

func SearchCloudCostSyncTasks(c *ctx.ServiceContext, form *forms.SearchCloudCostSyncTaskForm) (interface{}, e.Error) {
	query := c.DB().Model(&models.CloudCostSyncTask{}).Where("org_id = ?", c.OrgId)
	if form.Q != "" {
		q := "%" + strings.TrimSpace(form.Q) + "%"
		query = query.Where(`id like ? or provider like ? or source like ? or mode like ? or account_id like ? or
			region like ? or period like ? or message like ? or error like ? or params like ? or result like ?`,
			q, q, q, q, q, q, q, q, q, q, q)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	if form.Provider != "" {
		query = query.Where("provider = ?", form.Provider)
	}
	if form.Source != "" {
		query = query.Where("source = ?", form.Source)
	}
	if form.CloudAccountId != "" {
		query = query.Where("cloud_account_id = ?", form.CloudAccountId)
	}
	if form.AccountId != "" {
		query = query.Where("account_id = ?", form.AccountId)
	}
	if form.Period != "" {
		query = query.Where("period = ?", form.Period)
	}
	if form.SortField() == "" {
		query = query.Order("created_at desc")
	} else {
		query = form.Order(query)
	}

	tasks := make([]models.CloudCostSyncTask, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&tasks); err != nil {
		return nil, e.New(e.DBError, err)
	}
	list := make([]resps.CloudCostSyncTaskResp, 0, len(tasks))
	for _, task := range tasks {
		list = append(list, cloudCostSyncTaskResp(c, task))
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func CloudCostSyncTaskDetail(c *ctx.ServiceContext, form *forms.CloudCostSyncTaskParam) (*resps.CloudCostSyncTaskDetailResp, e.Error) {
	task, err := getCloudCostSyncTask(c, form.Id)
	if err != nil {
		return nil, err
	}
	logs := make([]models.CloudCostSyncTaskLog, 0)
	if dbErr := c.DB().Model(&models.CloudCostSyncTaskLog{}).
		Where("org_id = ? and task_id = ?", c.OrgId, task.Id).
		Order("created_at asc").
		Scan(&logs); dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}
	return &resps.CloudCostSyncTaskDetailResp{
		CloudCostSyncTaskResp: cloudCostSyncTaskResp(c, *task),
		Logs:                  logs,
	}, nil
}

func CreateCloudCostSyncTask(c *ctx.ServiceContext, form *forms.CreateCloudCostSyncTaskForm) (*resps.CloudCostSyncTaskDetailResp, e.Error) {
	task := cloudCostSyncTaskFromForm(c, &form.PullCloudCostRecordForm)
	if err := models.Create(c.DB(), &task); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if err := runCloudCostSyncTask(c, &task); err != nil {
		return nil, err
	}
	return CloudCostSyncTaskDetail(c, &forms.CloudCostSyncTaskParam{Id: task.Id})
}

func RetryCloudCostSyncTask(c *ctx.ServiceContext, form *forms.CloudCostSyncTaskParam) (*resps.CloudCostSyncTaskDetailResp, e.Error) {
	task, err := getCloudCostSyncTask(c, form.Id)
	if err != nil {
		return nil, err
	}
	if task.Status != models.CloudCostSyncTaskStatusFailed {
		return nil, e.New(e.BadParam, fmt.Errorf("成本同步任务 %s 当前状态不是失败，不能重试", form.Id))
	}
	if err := runCloudCostSyncTask(c, task); err != nil {
		return nil, err
	}
	return CloudCostSyncTaskDetail(c, form)
}

func SearchCloudCostSyncSchedules(c *ctx.ServiceContext, form *forms.SearchCloudCostSyncScheduleForm) (interface{}, e.Error) {
	query := c.DB().Model(&models.CloudCostSyncSchedule{}).Where("org_id = ?", c.OrgId)
	if form.Q != "" {
		q := "%" + strings.TrimSpace(form.Q) + "%"
		query = query.Where(`id like ? or name like ? or description like ? or provider like ? or source like ? or
			account_id like ? or region like ? or period like ? or last_error like ? or params like ?`,
			q, q, q, q, q, q, q, q, q, q)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	if form.Provider != "" {
		query = query.Where("provider = ?", form.Provider)
	}
	if form.Source != "" {
		query = query.Where("source = ?", form.Source)
	}
	if form.CloudAccountId != "" {
		query = query.Where("cloud_account_id = ?", form.CloudAccountId)
	}
	if form.AccountId != "" {
		query = query.Where("account_id = ?", form.AccountId)
	}
	if form.Period != "" {
		query = query.Where("period = ?", form.Period)
	}
	if form.SortField() == "" {
		query = query.Order("next_sync_at asc").Order("created_at desc")
	} else {
		query = form.Order(query)
	}

	schedules := make([]models.CloudCostSyncSchedule, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&schedules); err != nil {
		return nil, e.New(e.DBError, err)
	}
	list := make([]resps.CloudCostSyncScheduleResp, 0, len(schedules))
	for _, schedule := range schedules {
		list = append(list, cloudCostSyncScheduleResp(c, schedule))
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func CreateCloudCostSyncSchedule(c *ctx.ServiceContext, form *forms.CreateCloudCostSyncScheduleForm) (*resps.CloudCostSyncScheduleResp, e.Error) {
	schedule, err := cloudCostSyncScheduleFromForm(c, form)
	if err != nil {
		return nil, err
	}
	schedule.Id = models.NewId("ccp")
	if err := models.Create(c.DB(), schedule); err != nil {
		return nil, e.New(e.DBError, err)
	}
	resp := cloudCostSyncScheduleResp(c, *schedule)
	return &resp, nil
}

func UpdateCloudCostSyncSchedule(c *ctx.ServiceContext, form *forms.UpdateCloudCostSyncScheduleForm) (*resps.CloudCostSyncScheduleResp, e.Error) {
	existing, err := getCloudCostSyncSchedule(c, form.Id)
	if err != nil {
		return nil, err
	}
	updated, err := cloudCostSyncScheduleFromForm(c, &form.CreateCloudCostSyncScheduleForm)
	if err != nil {
		return nil, err
	}
	cloudCostMergePreservedScheduleParams(updated.Params, existing.Params, form)
	if updated.Status == models.CloudCostSyncScheduleStatusEnabled {
		cloudCostResetScheduleFailureParams(updated.Params)
	}
	attrs := models.Attrs{
		"name":             updated.Name,
		"description":      updated.Description,
		"cloud_account_id": updated.CloudAccountId,
		"provider":         updated.Provider,
		"account_id":       updated.AccountId,
		"region":           updated.Region,
		"source":           updated.Source,
		"period":           updated.Period,
		"currency":         updated.Currency,
		"status":           updated.Status,
		"sync_interval":    updated.SyncInterval,
		"next_sync_at":     updated.NextSyncAt,
		"params":           updated.Params,
	}
	if _, dbErr := c.DB().Model(&models.CloudCostSyncSchedule{}).
		Where("id = ? and org_id = ?", form.Id, c.OrgId).
		UpdateAttrs(attrs); dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}
	*existing = *updated
	existing.Id = form.Id
	existing.OrgId = c.OrgId
	resp := cloudCostSyncScheduleResp(c, *existing)
	return &resp, nil
}

func cloudCostMergePreservedScheduleParams(updated, existing models.ResAttrs, form *forms.UpdateCloudCostSyncScheduleForm) {
	if updated == nil || existing == nil || form == nil {
		return
	}
	preserveStringParams := map[string]string{
		"sourceUrl":            form.SourceURL,
		"sourceIndexUrl":       form.SourceIndexURL,
		"sourceObjectEndpoint": form.SourceObjectEndpoint,
		"sourceObjectBaseUrl":  form.SourceObjectBaseURL,
		"cursor":               form.Cursor,
	}
	for key, submitted := range preserveStringParams {
		if strings.TrimSpace(submitted) == "" {
			if value, ok := existing[key]; ok {
				updated[key] = value
			}
		}
	}
	if form.MaxFiles == 0 {
		if value, ok := existing["maxFiles"]; ok {
			updated["maxFiles"] = value
		}
	}
	if form.MaxRetryAttempts == 0 {
		if value, ok := existing["maxRetryAttempts"]; ok {
			updated["maxRetryAttempts"] = value
		}
	}
	if form.RetryBackoffSeconds == 0 {
		if value, ok := existing["retryBackoffSeconds"]; ok {
			updated["retryBackoffSeconds"] = value
		}
	}
	for _, key := range []string{"failureCount", "lastFailureAt", "lastFailureReason", "nextRetryAt", "autoPausedAt", "autoPauseReason"} {
		if value, ok := existing[key]; ok {
			updated[key] = value
		}
	}
}

func DeleteCloudCostSyncSchedule(c *ctx.ServiceContext, form *forms.CloudCostSyncScheduleParam) (interface{}, e.Error) {
	if _, err := c.DB().Where("id = ? and org_id = ?", form.Id, c.OrgId).Delete(&models.CloudCostSyncSchedule{}); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return nil, nil
}

func RunCloudCostSyncSchedule(c *ctx.ServiceContext, form *forms.CloudCostSyncScheduleParam) (*resps.CloudCostSyncTaskDetailResp, e.Error) {
	schedule, err := getCloudCostSyncSchedule(c, form.Id)
	if err != nil {
		return nil, err
	}
	return runCloudCostSyncSchedule(c, schedule)
}

func RunDueCloudCostSyncSchedules(c *ctx.ServiceContext, form *forms.RunDueCloudCostSyncScheduleForm) (*resps.CloudCostSyncScheduleRunResp, e.Error) {
	schedules := make([]models.CloudCostSyncSchedule, 0)
	query := c.DB().Model(&models.CloudCostSyncSchedule{}).
		Where("org_id = ? and status = ?", c.OrgId, models.CloudCostSyncScheduleStatusEnabled)
	if !form.Force {
		query = query.Where("next_sync_at is null or next_sync_at <= ?", time.Now())
	}
	if err := query.Order("next_sync_at asc").Find(&schedules); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return runCloudCostSyncSchedules(c, schedules)
}

func RunDueCloudCostSyncSchedulesForAllOrgs() (*resps.CloudCostSyncScheduleRunResp, e.Error) {
	schedules := make([]models.CloudCostSyncSchedule, 0)
	if err := db.Get().Model(&models.CloudCostSyncSchedule{}).
		Where("status = ? and (next_sync_at is null or next_sync_at <= ?)", models.CloudCostSyncScheduleStatusEnabled, time.Now()).
		Order("next_sync_at asc").
		Find(&schedules); err != nil {
		return nil, e.New(e.DBError, err)
	}
	resp := &resps.CloudCostSyncScheduleRunResp{TotalCount: len(schedules)}
	for idx := range schedules {
		schedule := &schedules[idx]
		if schedule.OrgId == "" {
			resp.SkippedCount++
			continue
		}
		workerCtx := &ctx.ServiceContext{
			UserId:   consts.SysUserId,
			OrgId:    schedule.OrgId,
			Email:    consts.DefaultSysEmail,
			Username: consts.DefaultSysName,
		}
		itemResp, err := runCloudCostSyncSchedules(workerCtx, []models.CloudCostSyncSchedule{*schedule})
		if err != nil {
			return nil, err
		}
		resp.TriggeredCount += itemResp.TriggeredCount
		resp.CompleteCount += itemResp.CompleteCount
		resp.FailedCount += itemResp.FailedCount
		resp.SkippedCount += itemResp.SkippedCount
		resp.LockSkippedCount += itemResp.LockSkippedCount
		resp.TaskIds = append(resp.TaskIds, itemResp.TaskIds...)
	}
	return resp, nil
}

func StartCloudCostSyncScheduleWorker(serviceId string) {
	interval := cloudCostSyncScheduleWorkerInterval()
	logger := logs.Get().
		WithField("worker", "cloudCostSyncSchedule").
		WithField("serviceId", serviceId).
		WithField("interval", interval.String())
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if result, err := RunDueCloudCostSyncSchedulesForAllOrgs(); err != nil {
			logger.Warnf("run due cloud cost sync schedules failed: %v", err)
		} else if result.TriggeredCount > 0 || result.FailedCount > 0 || result.LockSkippedCount > 0 {
			logger.Infof("run due cloud cost sync schedules result: %+v", result)
		}
		<-ticker.C
	}
}

func runCloudCostSyncSchedules(c *ctx.ServiceContext, schedules []models.CloudCostSyncSchedule) (*resps.CloudCostSyncScheduleRunResp, e.Error) {
	resp := &resps.CloudCostSyncScheduleRunResp{TotalCount: len(schedules)}
	for idx := range schedules {
		schedule := &schedules[idx]
		if schedule.Status != models.CloudCostSyncScheduleStatusEnabled {
			resp.SkippedCount++
			continue
		}
		locked, releaseLock, lockErr := cloudWebhookAcquireMysqlLock(cloudCostSyncScheduleLockName(schedule.OrgId, schedule.Id))
		if lockErr != nil {
			return nil, e.New(e.DBError, lockErr)
		}
		if !locked {
			resp.LockSkippedCount++
			continue
		}
		detail, err := func() (*resps.CloudCostSyncTaskDetailResp, e.Error) {
			defer releaseLock()
			return runCloudCostSyncSchedule(c, schedule)
		}()
		if err != nil {
			return nil, err
		}
		resp.TriggeredCount++
		resp.TaskIds = append(resp.TaskIds, detail.Id.String())
		switch detail.Status {
		case models.CloudCostSyncTaskStatusComplete:
			resp.CompleteCount++
		case models.CloudCostSyncTaskStatusFailed:
			resp.FailedCount++
		}
	}
	return resp, nil
}

func runCloudCostSyncSchedule(c *ctx.ServiceContext, schedule *models.CloudCostSyncSchedule) (*resps.CloudCostSyncTaskDetailResp, e.Error) {
	if schedule == nil {
		return nil, e.New(e.BadParam, fmt.Errorf("成本同步计划不存在"))
	}
	form := cloudCostSyncSchedulePullForm(schedule)
	taskDetail, err := CreateCloudCostSyncTask(c, &forms.CreateCloudCostSyncTaskForm{
		PullCloudCostRecordForm: *form,
	})
	nowTime := time.Now()
	now := models.Time(nowTime)
	nextSyncAt := models.Time(nowTime.Add(time.Duration(cloudCostSyncIntervalSeconds(schedule.SyncInterval)) * time.Second))
	params := modelResAttrs(schedule.Params)
	if params == nil {
		params = models.ResAttrs{}
	}
	attrs := models.Attrs{
		"last_synced_at": now,
		"next_sync_at":   nextSyncAt,
	}
	var failureErr error
	if err != nil {
		attrs["last_sync_status"] = models.CloudCostSyncTaskStatusFailed
		attrs["last_error"] = err.Error()
		failureErr = fmt.Errorf("%v", err)
	} else {
		attrs["last_sync_task_id"] = taskDetail.Id
		attrs["last_sync_status"] = taskDetail.Status
		attrs["last_error"] = taskDetail.Error
		if taskDetail.Status == models.CloudCostSyncTaskStatusFailed {
			failureErr = fmt.Errorf("%s", firstNonEmpty(taskDetail.Error, taskDetail.Message, "成本同步任务失败"))
		} else {
			cloudCostResetScheduleFailureParams(params)
		}
		if taskDetail.Status == models.CloudCostSyncTaskStatusComplete {
			delete(params, "nextRetryAt")
			if nextCursor := attrString(taskDetail.Result, "nextCursor"); nextCursor != "" {
				params["cursor"] = nextCursor
			}
		}
	}
	if failureErr != nil {
		cloudCostApplyScheduleFailure(c, schedule, params, attrs, nowTime, failureErr, taskDetail)
	}
	attrs["params"] = cloudProviderSanitizeAttrs(params)
	if _, dbErr := c.DB().Model(&models.CloudCostSyncSchedule{}).
		Where("id = ? and org_id = ?", schedule.Id, schedule.OrgId).
		UpdateAttrs(attrs); dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}
	if err != nil {
		return nil, err
	}
	return taskDetail, nil
}

func cloudCostSyncTaskFromForm(c *ctx.ServiceContext, form *forms.PullCloudCostRecordForm) models.CloudCostSyncTask {
	provider := normalizeProvider(form.Provider)
	period := firstNonEmpty(form.Period, time.Now().Format("2006-01"))
	currency := firstNonEmpty(form.Currency, "USD")
	source := cloudCostPullSource(provider, form.Source)
	params := models.ResAttrs{
		"provider":             provider,
		"cloudAccountId":       form.CloudAccountId.String(),
		"accountId":            form.AccountId,
		"region":               form.Region,
		"period":               period,
		"currency":             currency,
		"source":               source,
		"sourceUrl":            strings.TrimSpace(form.SourceURL),
		"sourceIndexUrl":       strings.TrimSpace(form.SourceIndexURL),
		"sourceObjectProvider": strings.TrimSpace(form.SourceObjectProvider),
		"sourceObjectEndpoint": strings.TrimSpace(form.SourceObjectEndpoint),
		"sourceObjectBucket":   strings.TrimSpace(form.SourceObjectBucket),
		"sourceObjectPrefix":   strings.TrimSpace(form.SourceObjectPrefix),
		"sourceObjectBaseUrl":  strings.TrimSpace(form.SourceObjectBaseURL),
		"cursor":               strings.TrimSpace(form.Cursor),
		"maxFiles":             form.MaxFiles,
	}
	return models.CloudCostSyncTask{
		SoftDeleteModel: models.SoftDeleteModel{
			TimedModel: models.TimedModel{
				BaseModel: models.BaseModel{Id: models.NewId("ccs")},
			},
		},
		OrgId:          c.OrgId,
		CreatorId:      c.UserId,
		CloudAccountId: form.CloudAccountId,
		Provider:       provider,
		AccountId:      form.AccountId,
		Region:         form.Region,
		Source:         source,
		Period:         period,
		Currency:       currency,
		Status:         models.CloudCostSyncTaskStatusPending,
		Message:        "成本同步任务已创建",
		Params:         cloudProviderSanitizeAttrs(params),
	}
}

func runCloudCostSyncTask(c *ctx.ServiceContext, task *models.CloudCostSyncTask) e.Error {
	if task == nil {
		return e.New(e.BadParam, fmt.Errorf("成本同步任务不存在"))
	}
	now := models.Time(time.Now())
	task.AttemptCount++
	task.Status = models.CloudCostSyncTaskStatusRunning
	task.Message = "成本同步任务运行中"
	task.Error = ""
	task.StartedAt = now
	task.EndedAt = models.Time(time.Time{})
	if _, err := c.DB().Model(&models.CloudCostSyncTask{}).
		Where("id = ? and org_id = ?", task.Id, c.OrgId).
		UpdateAttrs(models.Attrs{
			"status":        task.Status,
			"message":       task.Message,
			"error":         "",
			"attempt_count": task.AttemptCount,
			"started_at":    now,
			"ended_at":      nil,
		}); err != nil {
		return e.New(e.DBError, err)
	}
	_ = createCloudCostSyncTaskLog(c, task.Id, "pull", models.CloudCostSyncTaskStatusRunning, "开始拉取并导入成本账单", nil, "")

	pullForm := cloudCostSyncTaskPullForm(task)
	pullForm.SyncTaskId = task.Id
	resp, pullErr := PullCloudCostRecords(c, pullForm)
	endedAt := models.Time(time.Now())
	if pullErr != nil {
		task.Status = models.CloudCostSyncTaskStatusFailed
		task.Message = "成本同步任务失败"
		task.Error = pullErr.Error()
		task.EndedAt = endedAt
		if _, err := c.DB().Model(&models.CloudCostSyncTask{}).
			Where("id = ? and org_id = ?", task.Id, c.OrgId).
			UpdateAttrs(models.Attrs{
				"status":   task.Status,
				"message":  task.Message,
				"error":    task.Error,
				"ended_at": endedAt,
			}); err != nil {
			return e.New(e.DBError, err)
		}
		_ = createCloudCostSyncTaskLog(c, task.Id, "pull", models.CloudCostSyncTaskStatusFailed, task.Message, nil, task.Error)
		return nil
	}
	result := models.ResAttrs{
		"imported":       resp.Imported,
		"matchedCount":   resp.MatchedCount,
		"unmatchedCount": resp.UnmatchedCount,
		"providers":      resp.Providers,
		"sources":        resp.Sources,
		"periods":        resp.Periods,
		"currencies":     resp.Currencies,
		"mode":           resp.Mode,
		"source":         resp.Source,
	}
	if resp.SourceIndexURL != "" {
		result["sourceIndexUrl"] = cloudCostSourceURLPreview(resp.SourceIndexURL)
	}
	if resp.SourceObjectProvider != "" {
		result["sourceObjectProvider"] = resp.SourceObjectProvider
		result["sourceObjectEndpoint"] = cloudCostSourceURLPreview(resp.SourceObjectEndpoint)
		result["sourceObjectBucket"] = resp.SourceObjectBucket
		result["sourceObjectPrefix"] = resp.SourceObjectPrefix
	}
	if resp.Cursor != "" {
		result["cursor"] = resp.Cursor
	}
	if resp.NextCursor != "" {
		result["nextCursor"] = resp.NextCursor
	}
	if resp.Mode == "export_index" || resp.Mode == "export_object_list" {
		result["fileCount"] = resp.FileCount
		result["files"] = resp.Files
		if len(resp.SkippedFiles) > 0 {
			result["skippedFiles"] = resp.SkippedFiles
		}
	}
	task.Status = models.CloudCostSyncTaskStatusComplete
	task.Message = "成本同步任务完成"
	task.Imported = resp.Imported
	task.MatchedCount = resp.MatchedCount
	task.UnmatchedCount = resp.UnmatchedCount
	task.Mode = resp.Mode
	task.Source = resp.Source
	task.AccountId = resp.AccountId
	task.Result = result
	task.EndedAt = endedAt
	if _, err := c.DB().Model(&models.CloudCostSyncTask{}).
		Where("id = ? and org_id = ?", task.Id, c.OrgId).
		UpdateAttrs(models.Attrs{
			"status":          task.Status,
			"message":         task.Message,
			"mode":            task.Mode,
			"source":          task.Source,
			"account_id":      task.AccountId,
			"imported":        task.Imported,
			"matched_count":   task.MatchedCount,
			"unmatched_count": task.UnmatchedCount,
			"result":          result,
			"ended_at":        endedAt,
		}); err != nil {
		return e.New(e.DBError, err)
	}
	_ = createCloudCostSyncTaskLog(c, task.Id, "pull", models.CloudCostSyncTaskStatusComplete, task.Message, result, "")
	return nil
}

func cloudCostSyncTaskPullForm(task *models.CloudCostSyncTask) *forms.PullCloudCostRecordForm {
	params := modelResAttrs(task.Params)
	return &forms.PullCloudCostRecordForm{
		Provider:             firstNonEmpty(attrString(params, "provider"), task.Provider),
		CloudAccountId:       firstNonEmptyId(models.Id(attrString(params, "cloudAccountId")), task.CloudAccountId),
		AccountId:            firstNonEmpty(attrString(params, "accountId"), task.AccountId),
		Region:               firstNonEmpty(attrString(params, "region"), task.Region),
		Period:               firstNonEmpty(attrString(params, "period"), task.Period),
		Currency:             firstNonEmpty(attrString(params, "currency"), task.Currency),
		Source:               firstNonEmpty(attrString(params, "source"), task.Source),
		SourceURL:            attrString(params, "sourceUrl"),
		SourceIndexURL:       attrString(params, "sourceIndexUrl"),
		SourceObjectProvider: attrString(params, "sourceObjectProvider"),
		SourceObjectEndpoint: attrString(params, "sourceObjectEndpoint"),
		SourceObjectBucket:   attrString(params, "sourceObjectBucket"),
		SourceObjectPrefix:   attrString(params, "sourceObjectPrefix"),
		SourceObjectBaseURL:  attrString(params, "sourceObjectBaseUrl"),
		Cursor:               attrString(params, "cursor"),
		MaxFiles:             attrInt(params, "maxFiles"),
	}
}

func createCloudCostSyncTaskLog(c *ctx.ServiceContext, taskId models.Id, stage string, status string, message string, result models.ResAttrs, errMsg string) e.Error {
	now := models.Time(time.Now())
	log := models.CloudCostSyncTaskLog{
		OrgId:     c.OrgId,
		TaskId:    taskId,
		Stage:     stage,
		Status:    status,
		Message:   message,
		Error:     errMsg,
		Result:    result,
		StartedAt: now,
		EndedAt:   now,
	}
	log.Id = models.NewId("ccl")
	if err := models.Create(c.DB(), &log); err != nil {
		return e.New(e.DBError, err)
	}
	return nil
}

func getCloudCostSyncTask(c *ctx.ServiceContext, id models.Id) (*models.CloudCostSyncTask, e.Error) {
	task := models.CloudCostSyncTask{}
	if err := c.DB().Model(&models.CloudCostSyncTask{}).
		Where("id = ? and org_id = ?", id, c.OrgId).
		First(&task); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	return &task, nil
}

func cloudCostSyncTaskResp(c *ctx.ServiceContext, task models.CloudCostSyncTask) resps.CloudCostSyncTaskResp {
	displayTask := task
	displayTask.Params = cloudProviderSanitizeAttrs(displayTask.Params)
	if sourceURL := attrString(displayTask.Params, "sourceUrl"); sourceURL != "" {
		displayTask.Params["sourceUrl"] = cloudCostSourceURLPreview(sourceURL)
	}
	if sourceIndexURL := attrString(displayTask.Params, "sourceIndexUrl"); sourceIndexURL != "" {
		displayTask.Params["sourceIndexUrl"] = cloudCostSourceURLPreview(sourceIndexURL)
	}
	if sourceObjectEndpoint := attrString(displayTask.Params, "sourceObjectEndpoint"); sourceObjectEndpoint != "" {
		displayTask.Params["sourceObjectEndpoint"] = cloudCostSourceURLPreview(sourceObjectEndpoint)
	}
	if sourceObjectBaseURL := attrString(displayTask.Params, "sourceObjectBaseUrl"); sourceObjectBaseURL != "" {
		displayTask.Params["sourceObjectBaseUrl"] = cloudCostSourceURLPreview(sourceObjectBaseURL)
	}
	return resps.CloudCostSyncTaskResp{
		CloudCostSyncTask:           displayTask,
		CreatorName:                 lookupName(c, &models.User{}, task.CreatorId),
		CloudAccountName:            lookupName(c, &models.CloudAccount{}, task.CloudAccountId),
		SourceURLPreview:            cloudCostSourceURLPreview(attrString(task.Params, "sourceUrl")),
		SourceIndexURLPreview:       cloudCostSourceURLPreview(attrString(task.Params, "sourceIndexUrl")),
		SourceObjectEndpointPreview: cloudCostSourceURLPreview(attrString(task.Params, "sourceObjectEndpoint")),
	}
}

func cloudCostSourceURLPreview(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		if len(rawURL) > 96 {
			return rawURL[:96] + "..."
		}
		return rawURL
	}
	if parsed.RawQuery != "" {
		parsed.RawQuery = "..."
	}
	if parsed.Fragment != "" {
		parsed.Fragment = "..."
	}
	value := parsed.String()
	if len(value) > 160 {
		return value[:160] + "..."
	}
	return value
}

func cloudCostSyncScheduleFromForm(c *ctx.ServiceContext, form *forms.CreateCloudCostSyncScheduleForm) (*models.CloudCostSyncSchedule, e.Error) {
	provider := normalizeProvider(form.Provider)
	if provider == "" {
		return nil, e.New(e.BadParam, fmt.Errorf("provider is required"), http.StatusBadRequest)
	}
	period := firstNonEmpty(form.Period, time.Now().Format("2006-01"))
	currency := firstNonEmpty(strings.ToUpper(form.Currency), "USD")
	source := cloudCostPullSource(provider, form.Source)
	status := firstNonEmpty(form.Status, models.CloudCostSyncScheduleStatusEnabled)
	params := models.ResAttrs{}
	for key, value := range form.Params {
		params[key] = value
	}
	if strings.TrimSpace(form.SourceURL) != "" {
		params["sourceUrl"] = strings.TrimSpace(form.SourceURL)
	}
	if strings.TrimSpace(form.SourceIndexURL) != "" {
		params["sourceIndexUrl"] = strings.TrimSpace(form.SourceIndexURL)
	}
	if strings.TrimSpace(form.SourceObjectProvider) != "" {
		params["sourceObjectProvider"] = strings.TrimSpace(form.SourceObjectProvider)
	}
	if strings.TrimSpace(form.SourceObjectEndpoint) != "" {
		params["sourceObjectEndpoint"] = strings.TrimSpace(form.SourceObjectEndpoint)
	}
	if strings.TrimSpace(form.SourceObjectBucket) != "" {
		params["sourceObjectBucket"] = strings.TrimSpace(form.SourceObjectBucket)
	}
	if strings.TrimSpace(form.SourceObjectPrefix) != "" {
		params["sourceObjectPrefix"] = strings.TrimSpace(form.SourceObjectPrefix)
	}
	if strings.TrimSpace(form.SourceObjectBaseURL) != "" {
		params["sourceObjectBaseUrl"] = strings.TrimSpace(form.SourceObjectBaseURL)
	}
	if strings.TrimSpace(form.Cursor) != "" {
		params["cursor"] = strings.TrimSpace(form.Cursor)
	}
	if form.MaxFiles > 0 {
		params["maxFiles"] = form.MaxFiles
	}
	if form.MaxRetryAttempts > 0 {
		params["maxRetryAttempts"] = cloudCostSyncScheduleMaxRetryAttempts(form.MaxRetryAttempts)
	}
	if form.RetryBackoffSeconds > 0 {
		params["retryBackoffSeconds"] = cloudCostSyncScheduleRetryBackoffSeconds(form.RetryBackoffSeconds)
	}
	if form.NotifyOnFailure {
		params["notifyOnFailure"] = true
	}
	if form.AutoPauseOnFailure {
		params["autoPauseOnFailure"] = true
	}
	params = cloudCostSyncScheduleNormalizeNotificationParams(params)
	nextSyncAt := form.NextSyncAt
	if time.Time(nextSyncAt).IsZero() || time.Time(nextSyncAt).Year() <= 1 {
		nextSyncAt = models.Time(time.Now())
	}
	return &models.CloudCostSyncSchedule{
		OrgId:          c.OrgId,
		CreatorId:      c.UserId,
		Name:           form.Name,
		Description:    form.Description,
		CloudAccountId: form.CloudAccountId,
		Provider:       provider,
		AccountId:      form.AccountId,
		Region:         form.Region,
		Source:         source,
		Period:         period,
		Currency:       currency,
		Status:         status,
		SyncInterval:   cloudCostSyncIntervalSeconds(form.SyncInterval),
		NextSyncAt:     nextSyncAt,
		Params:         cloudProviderSanitizeAttrs(params),
	}, nil
}

func cloudCostSyncSchedulePullForm(schedule *models.CloudCostSyncSchedule) *forms.PullCloudCostRecordForm {
	params := modelResAttrs(schedule.Params)
	return &forms.PullCloudCostRecordForm{
		Provider:             schedule.Provider,
		CloudAccountId:       schedule.CloudAccountId,
		AccountId:            schedule.AccountId,
		Region:               schedule.Region,
		Period:               firstNonEmpty(schedule.Period, time.Now().Format("2006-01")),
		Currency:             firstNonEmpty(schedule.Currency, "USD"),
		Source:               schedule.Source,
		SourceURL:            attrString(params, "sourceUrl"),
		SourceIndexURL:       attrString(params, "sourceIndexUrl"),
		SourceObjectProvider: attrString(params, "sourceObjectProvider"),
		SourceObjectEndpoint: attrString(params, "sourceObjectEndpoint"),
		SourceObjectBucket:   attrString(params, "sourceObjectBucket"),
		SourceObjectPrefix:   attrString(params, "sourceObjectPrefix"),
		SourceObjectBaseURL:  attrString(params, "sourceObjectBaseUrl"),
		Cursor:               attrString(params, "cursor"),
		MaxFiles:             attrInt(params, "maxFiles"),
	}
}

func getCloudCostSyncSchedule(c *ctx.ServiceContext, id models.Id) (*models.CloudCostSyncSchedule, e.Error) {
	schedule := models.CloudCostSyncSchedule{}
	if err := c.DB().Model(&models.CloudCostSyncSchedule{}).
		Where("id = ? and org_id = ?", id, c.OrgId).
		First(&schedule); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	return &schedule, nil
}

func cloudCostSyncScheduleResp(c *ctx.ServiceContext, schedule models.CloudCostSyncSchedule) resps.CloudCostSyncScheduleResp {
	params := modelResAttrs(schedule.Params)
	if params == nil {
		params = models.ResAttrs{}
	}
	displaySchedule := schedule
	displaySchedule.Params = cloudProviderSanitizeAttrs(displaySchedule.Params)
	if sourceURL := attrString(displaySchedule.Params, "sourceUrl"); sourceURL != "" {
		displaySchedule.Params["sourceUrl"] = cloudCostSourceURLPreview(sourceURL)
	}
	if sourceIndexURL := attrString(displaySchedule.Params, "sourceIndexUrl"); sourceIndexURL != "" {
		displaySchedule.Params["sourceIndexUrl"] = cloudCostSourceURLPreview(sourceIndexURL)
	}
	if sourceObjectEndpoint := attrString(displaySchedule.Params, "sourceObjectEndpoint"); sourceObjectEndpoint != "" {
		displaySchedule.Params["sourceObjectEndpoint"] = cloudCostSourceURLPreview(sourceObjectEndpoint)
	}
	if sourceObjectBaseURL := attrString(displaySchedule.Params, "sourceObjectBaseUrl"); sourceObjectBaseURL != "" {
		displaySchedule.Params["sourceObjectBaseUrl"] = cloudCostSourceURLPreview(sourceObjectBaseURL)
	}
	return resps.CloudCostSyncScheduleResp{
		CloudCostSyncSchedule:        displaySchedule,
		CreatorName:                  lookupName(c, &models.User{}, schedule.CreatorId),
		CloudAccountName:             lookupName(c, &models.CloudAccount{}, schedule.CloudAccountId),
		SourceURLPreview:             cloudCostSourceURLPreview(attrString(schedule.Params, "sourceUrl")),
		SourceIndexURLPreview:        cloudCostSourceURLPreview(attrString(schedule.Params, "sourceIndexUrl")),
		SourceObjectEndpointPreview:  cloudCostSourceURLPreview(attrString(schedule.Params, "sourceObjectEndpoint")),
		FailureCount:                 attrInt(params, "failureCount"),
		MaxRetryAttempts:             cloudCostSyncScheduleMaxRetryAttempts(attrInt(params, "maxRetryAttempts")),
		RetryBackoffSeconds:          cloudCostSyncScheduleRetryBackoffSeconds(attrInt(params, "retryBackoffSeconds")),
		NotifyOnFailure:              attrBool(params, "notifyOnFailure"),
		AutoPauseOnFailure:           attrBool(params, "autoPauseOnFailure"),
		NotificationOwner:            attrString(params, "notificationOwner"),
		NotificationRoutes:           cloudCostSyncScheduleNotificationValues(params, "notificationRoutes"),
		NotificationAssignees:        cloudCostSyncScheduleNotificationValues(params, "notificationAssignees"),
		NotificationSilenceMinutes:   cloudCostSyncScheduleNotificationSilenceMinutes(params),
		NotificationWindows:          cloudCostSyncScheduleNotificationWindows(params),
		NotificationFailureRoutes:    cloudCostSyncScheduleNotificationFailureRoutes(params),
		NotificationEscalationAt:     cloudCostSyncScheduleNotificationEscalationAt(params),
		NotificationEscalationRoutes: cloudCostSyncScheduleNotificationValues(params, "notificationEscalationRoutes"),
		NextRetryAt:                  attrString(params, "nextRetryAt"),
		AutoPausedAt:                 attrString(params, "autoPausedAt"),
	}
}

func cloudCostApplyScheduleFailure(c *ctx.ServiceContext, schedule *models.CloudCostSyncSchedule, params models.ResAttrs, attrs models.Attrs, now time.Time, failureErr error, taskDetail *resps.CloudCostSyncTaskDetailResp) {
	failureCount := attrInt(params, "failureCount") + 1
	maxRetryAttempts := cloudCostSyncScheduleMaxRetryAttempts(attrInt(params, "maxRetryAttempts"))
	retryBackoffSeconds := cloudCostSyncScheduleRetryBackoffSeconds(attrInt(params, "retryBackoffSeconds"))
	failureReason := failureErr.Error()
	params["failureCount"] = failureCount
	params["lastFailureAt"] = now.Format(time.RFC3339)
	params["lastFailureReason"] = failureReason
	autoPaused := false
	if maxRetryAttempts > 0 && failureCount <= maxRetryAttempts {
		retryAt := now.Add(time.Duration(cloudCostSyncScheduleRetryDelaySeconds(failureCount, retryBackoffSeconds)) * time.Second)
		params["nextRetryAt"] = retryAt.Format(time.RFC3339)
		attrs["next_sync_at"] = models.Time(retryAt)
	} else {
		delete(params, "nextRetryAt")
		if maxRetryAttempts > 0 && attrBool(params, "autoPauseOnFailure") {
			autoPaused = true
			params["autoPausedAt"] = now.Format(time.RFC3339)
			params["autoPauseReason"] = failureReason
			attrs["status"] = models.CloudCostSyncScheduleStatusDisabled
			attrs["next_sync_at"] = models.Time(time.Time{})
		}
	}
	if attrBool(params, "notifyOnFailure") || autoPaused {
		cloudCostSyncScheduleFailureEvent(c, schedule, params, failureReason, autoPaused, taskDetail)
	}
}

func cloudCostResetScheduleFailureParams(params models.ResAttrs) {
	params["failureCount"] = 0
	delete(params, "lastFailureAt")
	delete(params, "lastFailureReason")
	delete(params, "nextRetryAt")
	delete(params, "autoPausedAt")
	delete(params, "autoPauseReason")
}

func cloudCostSyncScheduleNormalizeNotificationParams(params models.ResAttrs) models.ResAttrs {
	if params == nil {
		params = models.ResAttrs{}
	}
	owner := strings.TrimSpace(attrString(params, "notificationOwner"))
	if owner != "" {
		params["notificationOwner"] = cloudSyncPolicyTruncateString(owner, 80)
	} else {
		delete(params, "notificationOwner")
	}
	routes := cloudCostSyncScheduleNotificationValues(params, "notificationRoutes")
	if len(routes) > 0 {
		params["notificationRoutes"] = routes
	} else {
		delete(params, "notificationRoutes")
	}
	assignees := cloudCostSyncScheduleNotificationValues(params, "notificationAssignees")
	if len(assignees) > 0 {
		params["notificationAssignees"] = assignees
	} else {
		delete(params, "notificationAssignees")
	}
	silenceMinutes := cloudCostSyncScheduleNotificationSilenceMinutes(params)
	if silenceMinutes > 0 {
		params["notificationSilenceMinutes"] = silenceMinutes
	} else {
		delete(params, "notificationSilenceMinutes")
	}
	windows := cloudCostSyncScheduleNotificationWindows(params)
	if len(windows) > 0 {
		params["notificationWindows"] = windows
	} else {
		delete(params, "notificationWindows")
	}
	failureRoutes := cloudCostSyncScheduleNotificationFailureRoutes(params)
	if len(failureRoutes) > 0 {
		params["notificationFailureRoutes"] = failureRoutes
	} else {
		delete(params, "notificationFailureRoutes")
	}
	escalationAt := cloudCostSyncScheduleNotificationEscalationAt(params)
	if escalationAt > 0 {
		params["notificationEscalationAt"] = escalationAt
	} else {
		delete(params, "notificationEscalationAt")
	}
	escalationRoutes := cloudCostSyncScheduleNotificationValues(params, "notificationEscalationRoutes")
	if len(escalationRoutes) > 0 {
		params["notificationEscalationRoutes"] = escalationRoutes
	} else {
		delete(params, "notificationEscalationRoutes")
	}
	return params
}

func cloudCostSyncScheduleNotificationValues(params models.ResAttrs, key string) []string {
	if params == nil {
		return nil
	}
	return cloudSyncPolicyNormalizeRoutingValues(params[key])
}

func cloudCostSyncScheduleNotificationSilenceMinutes(params models.ResAttrs) int {
	if params == nil {
		return 0
	}
	minutes := attrInt(params, "notificationSilenceMinutes")
	if minutes <= 0 {
		return 0
	}
	if minutes > cloudCostSyncScheduleMaxSilenceMins {
		return cloudCostSyncScheduleMaxSilenceMins
	}
	return minutes
}

func cloudCostSyncScheduleNotificationWindows(params models.ResAttrs) []string {
	if params == nil {
		return nil
	}
	raw := normalizeStringList(cloudSyncPolicyAttrStringSlice(params["notificationWindows"]))
	windows := make([]string, 0, len(raw))
	for _, value := range raw {
		start, end, ok := cloudSyncPolicyParsePauseWindow(value)
		if !ok {
			continue
		}
		windows = append(windows, fmt.Sprintf("%02d:%02d-%02d:%02d", start/60, start%60, end/60, end%60))
	}
	return dedupeStrings(windows)
}

func cloudCostSyncScheduleNotificationFailureRoutes(params models.ResAttrs) models.ResAttrs {
	if params == nil {
		return models.ResAttrs{}
	}
	raw := cloudCostSyncScheduleNotificationFailureRouteMap(params["notificationFailureRoutes"])
	result := models.ResAttrs{}
	for category, value := range raw {
		category = cloudCostSyncScheduleNormalizeFailureCategory(category)
		if category == "" {
			continue
		}
		routes := cloudSyncPolicyNormalizeRoutingValues(value)
		if len(routes) == 0 {
			continue
		}
		result[category] = routes
	}
	return result
}

func cloudCostSyncScheduleNotificationFailureRouteMap(value interface{}) models.ResAttrs {
	switch typed := value.(type) {
	case string:
		return cloudCostSyncScheduleParseFailureRouteText(typed)
	default:
		return cloudSyncPolicyAttrMap(value)
	}
}

func cloudCostSyncScheduleParseFailureRouteText(value string) models.ResAttrs {
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

func cloudCostSyncScheduleNotificationEscalationAt(params models.ResAttrs) int {
	if params == nil {
		return 0
	}
	value := attrInt(params, "notificationEscalationAt")
	if value <= 0 {
		return 0
	}
	if value > cloudCostSyncScheduleMaxEscalateAt {
		return cloudCostSyncScheduleMaxEscalateAt
	}
	return value
}

func cloudCostSyncScheduleApplyNotificationRouting(schedule *models.CloudCostSyncSchedule, params models.ResAttrs, payload models.ResAttrs, reason string, autoPaused bool) models.ResAttrs {
	if payload == nil {
		payload = models.ResAttrs{}
	}
	if schedule != nil && schedule.Id != "" {
		payload["notificationSource"] = "cloud_cost_sync_schedule"
	}
	failureCategory := cloudCostSyncScheduleFailureCategory(reason)
	if failureCategory != "" {
		payload["failureCategory"] = failureCategory
	}
	if owner := strings.TrimSpace(attrString(params, "notificationOwner")); owner != "" {
		payload["notificationOwner"] = owner
	}
	routes := cloudCostSyncScheduleNotificationValues(params, "notificationRoutes")
	if categoryRoutes := cloudCostSyncScheduleNotificationFailureRoutesForCategory(params, failureCategory); len(categoryRoutes) > 0 {
		routes = append(routes, categoryRoutes...)
		payload["notificationFailureRoutes"] = categoryRoutes
	}
	escalated, escalationReason := cloudCostSyncScheduleNotificationEscalated(params, attrInt(payload, "failureCount"), autoPaused)
	if escalated {
		payload["notificationEscalated"] = true
		payload["notificationEscalationReason"] = escalationReason
		if escalationAt := cloudCostSyncScheduleNotificationEscalationAt(params); escalationAt > 0 {
			payload["notificationEscalationAt"] = escalationAt
		}
		if escalationRoutes := cloudCostSyncScheduleNotificationValues(params, "notificationEscalationRoutes"); len(escalationRoutes) > 0 {
			routes = append(routes, escalationRoutes...)
			payload["notificationEscalationRoutes"] = escalationRoutes
		}
	}
	if len(routes) > 0 {
		payload["notificationRoutes"] = dedupeStrings(routes)
	}
	if assignees := cloudCostSyncScheduleNotificationValues(params, "notificationAssignees"); len(assignees) > 0 {
		payload["notificationAssignees"] = assignees
	}
	if silenceMinutes := cloudCostSyncScheduleNotificationSilenceMinutes(params); silenceMinutes > 0 {
		payload["notificationSilenceMinutes"] = silenceMinutes
	}
	if windows := cloudCostSyncScheduleNotificationWindows(params); len(windows) > 0 {
		payload["notificationWindows"] = windows
	}
	return payload
}

func cloudCostSyncScheduleNotificationFailureRoutesForCategory(params models.ResAttrs, category string) []string {
	category = cloudCostSyncScheduleNormalizeFailureCategory(category)
	if category == "" {
		return nil
	}
	routes := cloudCostSyncScheduleNotificationFailureRoutes(params)
	return cloudSyncPolicyNormalizeRoutingValues(routes[category])
}

func cloudCostSyncScheduleNotificationEscalated(params models.ResAttrs, failureCount int, autoPaused bool) (bool, string) {
	if autoPaused {
		return true, "auto_paused"
	}
	escalationAt := cloudCostSyncScheduleNotificationEscalationAt(params)
	if escalationAt > 0 && failureCount >= escalationAt {
		return true, "failure_count"
	}
	return false, ""
}

func cloudCostSyncScheduleNormalizeFailureCategory(value string) string {
	switch cloudEventNotificationRouteKey(value) {
	case "rate_limit", "rate-limit", "ratelimit", "throttle", "throttling":
		return "rate_limit"
	case "credential", "credentials", "auth", "authentication", "token", "secret":
		return "credential"
	case "permission", "permissions", "forbidden", "denied", "authorization", "unauthorized":
		return "permission"
	case "network", "timeout", "temporary", "unavailable", "connection":
		return "network"
	case "not_found", "not-found", "notfound", "missing", "404":
		return "not_found"
	case "config", "configuration", "invalid", "unsupported", "bad_request", "bad-request":
		return "config"
	case "unknown", "other", "default":
		return "unknown"
	default:
		return ""
	}
}

func cloudCostSyncScheduleFailureCategory(reason string) string {
	text := strings.ToLower(strings.TrimSpace(reason))
	switch {
	case text == "":
		return "unknown"
	case strings.Contains(text, "429") ||
		strings.Contains(text, "too many request") ||
		strings.Contains(text, "rate limit") ||
		strings.Contains(text, "ratelimit") ||
		strings.Contains(text, "throttl") ||
		strings.Contains(text, "limitexceeded"):
		return "rate_limit"
	case strings.Contains(text, "invalid token") ||
		strings.Contains(text, "expired token") ||
		strings.Contains(text, "expiredtoken") ||
		strings.Contains(text, "invalid access key") ||
		strings.Contains(text, "invalidaccesskey") ||
		strings.Contains(text, "credential") ||
		strings.Contains(text, "signature") ||
		strings.Contains(text, "notauthenticated") ||
		strings.Contains(text, "unauthenticated"):
		return "credential"
	case strings.Contains(text, "access denied") ||
		strings.Contains(text, "accessdenied") ||
		strings.Contains(text, "forbidden") ||
		strings.Contains(text, "not authorized") ||
		strings.Contains(text, "notauthorized") ||
		strings.Contains(text, "unauthorized") ||
		strings.Contains(text, "permission") ||
		strings.Contains(text, "denied"):
		return "permission"
	case strings.Contains(text, "404") ||
		strings.Contains(text, "not found") ||
		strings.Contains(text, "notfound") ||
		strings.Contains(text, "no such") ||
		strings.Contains(text, "nosuch") ||
		strings.Contains(text, "does not exist"):
		return "not_found"
	case strings.Contains(text, "timeout") ||
		strings.Contains(text, "timed out") ||
		strings.Contains(text, "connection") ||
		strings.Contains(text, "temporary") ||
		strings.Contains(text, "no such host") ||
		strings.Contains(text, "dns") ||
		strings.Contains(text, "eof") ||
		strings.Contains(text, "503") ||
		strings.Contains(text, "502") ||
		strings.Contains(text, "500"):
		return "network"
	case strings.Contains(text, "invalid") ||
		strings.Contains(text, "malformed") ||
		strings.Contains(text, "parse") ||
		strings.Contains(text, "unsupported") ||
		strings.Contains(text, "bad request") ||
		strings.Contains(text, "empty"):
		return "config"
	default:
		return "unknown"
	}
}

func cloudCostSyncScheduleNotificationSuppressed(c *ctx.ServiceContext, schedule *models.CloudCostSyncSchedule, params models.ResAttrs, eventType string, now time.Time) bool {
	if c == nil || schedule == nil || schedule.Id == "" {
		return false
	}
	windows := cloudCostSyncScheduleNotificationWindows(params)
	if len(windows) > 0 && !cloudCostSyncScheduleNotificationWindowActive(windows, now) {
		return true
	}
	silenceMinutes := cloudCostSyncScheduleNotificationSilenceMinutes(params)
	if silenceMinutes <= 0 {
		return false
	}
	since := now.Add(-time.Duration(silenceMinutes) * time.Minute)
	exists, err := c.DB().Model(&models.CloudEvent{}).
		Where("org_id = ? and event_type = ? and resource_type = ? and resource_id = ? and occurred_at >= ?",
			schedule.OrgId, eventType, "cloud_cost_sync_schedule", schedule.Id.String(), since).
		Exists()
	if err != nil {
		logs.Get().WithField("cloudCostSyncScheduleId", schedule.Id.String()).
			Warnf("check cloud cost sync schedule notification silence failed: %v", err)
		return false
	}
	return exists
}

func cloudCostSyncScheduleNotificationWindowActive(windows []string, now time.Time) bool {
	for _, window := range windows {
		start, end, ok := cloudSyncPolicyParsePauseWindow(window)
		if ok && cloudSyncPolicyTimeInWindow(now, start, end) {
			return true
		}
	}
	return false
}

func cloudCostSyncScheduleFailureEvent(c *ctx.ServiceContext, schedule *models.CloudCostSyncSchedule, params models.ResAttrs, reason string, autoPaused bool, taskDetail *resps.CloudCostSyncTaskDetailResp) {
	if schedule == nil {
		return
	}
	eventType := "cost.sync.schedule.failed"
	title := "成本同步计划失败"
	if autoPaused {
		eventType = "cost.sync.schedule.auto_paused"
		title = "成本同步计划已自动暂停"
	}
	if !autoPaused && cloudCostSyncScheduleNotificationSuppressed(c, schedule, params, eventType, time.Now()) {
		return
	}
	payload := cloudCostSyncScheduleApplyNotificationRouting(schedule, params, models.ResAttrs{
		"scheduleId":          schedule.Id.String(),
		"scheduleName":        schedule.Name,
		"provider":            schedule.Provider,
		"source":              schedule.Source,
		"period":              schedule.Period,
		"currency":            schedule.Currency,
		"failureCount":        attrInt(params, "failureCount"),
		"maxRetryAttempts":    cloudCostSyncScheduleMaxRetryAttempts(attrInt(params, "maxRetryAttempts")),
		"retryBackoffSeconds": cloudCostSyncScheduleRetryBackoffSeconds(attrInt(params, "retryBackoffSeconds")),
		"nextRetryAt":         attrString(params, "nextRetryAt"),
		"autoPaused":          autoPaused,
		"reason":              reason,
	}, reason, autoPaused)
	if taskDetail != nil {
		payload["taskId"] = taskDetail.Id.String()
		payload["taskStatus"] = taskDetail.Status
	}
	eventStatus := schedule.Status
	if autoPaused {
		eventStatus = models.CloudCostSyncScheduleStatusDisabled
	}
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:        schedule.OrgId,
		Source:       models.CloudEventSourceCost,
		EventType:    eventType,
		Level:        models.CloudEventLevelWarning,
		Status:       eventStatus,
		Provider:     schedule.Provider,
		AccountId:    schedule.AccountId,
		Region:       schedule.Region,
		ResourceType: "cloud_cost_sync_schedule",
		ResourceId:   schedule.Id.String(),
		ResourceName: schedule.Name,
		Title:        title,
		Message:      fmt.Sprintf("%s：%s", title, reason),
		Payload:      payload,
	})
}

func cloudCostSyncScheduleMaxRetryAttempts(value int) int {
	if value < 0 {
		return 0
	}
	if value > 10 {
		return 10
	}
	return value
}

func cloudCostSyncScheduleRetryBackoffSeconds(value int) int {
	if value <= 0 {
		return 300
	}
	if value < 60 {
		return 60
	}
	if value > int((24 * time.Hour).Seconds()) {
		return int((24 * time.Hour).Seconds())
	}
	return value
}

func cloudCostSyncScheduleRetryDelaySeconds(failureCount int, baseSeconds int) int {
	if failureCount <= 1 {
		return cloudCostSyncScheduleRetryBackoffSeconds(baseSeconds)
	}
	delay := cloudCostSyncScheduleRetryBackoffSeconds(baseSeconds)
	for i := 1; i < failureCount && i < 6; i++ {
		delay *= 2
	}
	maxDelay := int((24 * time.Hour).Seconds())
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

func cloudCostSyncIntervalSeconds(value int) int {
	if value <= 0 {
		return int(cloudCostSyncScheduleDefaultInterval.Seconds())
	}
	duration := time.Duration(value) * time.Second
	if duration < cloudCostSyncScheduleMinInterval {
		return int(cloudCostSyncScheduleMinInterval.Seconds())
	}
	if duration > cloudCostSyncScheduleMaxInterval {
		return int(cloudCostSyncScheduleMaxInterval.Seconds())
	}
	return value
}

func cloudCostSyncScheduleWorkerInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("CLOUD_COST_SYNC_SCHEDULE_WORKER_INTERVAL"))
	if raw == "" {
		return cloudCostSyncScheduleWorkerDefault
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return cloudCostSyncScheduleWorkerDefault
	}
	duration := time.Duration(seconds) * time.Second
	if duration < cloudCostSyncScheduleWorkerMin {
		return cloudCostSyncScheduleWorkerMin
	}
	if duration > cloudCostSyncScheduleWorkerMax {
		return cloudCostSyncScheduleWorkerMax
	}
	return duration
}

func cloudCostSyncScheduleLockName(orgId models.Id, scheduleId models.Id) string {
	sum := sha1.Sum([]byte(orgId.String() + ":" + scheduleId.String()))
	return "ccs:" + hex.EncodeToString(sum[:])
}

func recordCloudCostPullEvent(c *ctx.ServiceContext, form *forms.PullCloudCostRecordForm, account *cmdbCloudAccount, provider, source, mode, sourceURL string, resp *resps.CloudCostPullResp, pullErr error) {
	if c == nil || form == nil {
		return
	}
	level := models.CloudEventLevelInfo
	status := "complete"
	eventType := "cost.pull.completed"
	title := "成本账单拉取完成"
	message := fmt.Sprintf("%s %s 成本账单拉取完成", provider, firstNonEmpty(form.Period, time.Now().Format("2006-01")))
	if pullErr != nil {
		level = models.CloudEventLevelError
		status = "failed"
		eventType = "cost.pull.failed"
		title = "成本账单拉取失败"
		message = fmt.Sprintf("%s %s 成本账单拉取失败：%v", provider, firstNonEmpty(form.Period, time.Now().Format("2006-01")), pullErr)
	}
	accountId := form.AccountId
	if accountId == "" && account != nil {
		accountId = account.AccountId
	}
	payload := models.ResAttrs{
		"provider":             provider,
		"source":               source,
		"mode":                 mode,
		"sourceUrl":            cloudCostSourceURLPreview(sourceURL),
		"sourceIndexUrl":       cloudCostSourceURLPreview(form.SourceIndexURL),
		"sourceObjectProvider": form.SourceObjectProvider,
		"sourceObjectEndpoint": cloudCostSourceURLPreview(form.SourceObjectEndpoint),
		"sourceObjectBucket":   form.SourceObjectBucket,
		"sourceObjectPrefix":   form.SourceObjectPrefix,
		"cursor":               form.Cursor,
		"period":               firstNonEmpty(form.Period, time.Now().Format("2006-01")),
		"currency":             firstNonEmpty(form.Currency, "USD"),
		"cloudAccountId":       form.CloudAccountId.String(),
		"accountId":            accountId,
		"region":               form.Region,
	}
	if mode == "export_index" {
		payload["sourceUrl"] = ""
		payload["sourceIndexUrl"] = cloudCostSourceURLPreview(sourceURL)
	}
	if mode == "export_object_list" {
		payload["sourceUrl"] = ""
		payload["sourceIndexUrl"] = ""
		payload["sourceObjectEndpoint"] = cloudCostSourceURLPreview(sourceURL)
	}
	if form.SyncTaskId != "" {
		payload["syncTaskId"] = form.SyncTaskId.String()
	}
	if resp != nil {
		payload["imported"] = resp.Imported
		payload["matchedCount"] = resp.MatchedCount
		payload["unmatchedCount"] = resp.UnmatchedCount
		payload["providers"] = resp.Providers
		payload["sources"] = resp.Sources
		payload["periods"] = resp.Periods
		payload["currencies"] = resp.Currencies
		payload["nextCursor"] = resp.NextCursor
		payload["fileCount"] = resp.FileCount
		payload["sourceObjectProvider"] = resp.SourceObjectProvider
		payload["sourceObjectBucket"] = resp.SourceObjectBucket
		payload["sourceObjectPrefix"] = resp.SourceObjectPrefix
	}
	if pullErr != nil {
		payload["error"] = pullErr.Error()
	}
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:          c.OrgId,
		CloudAccountId: form.CloudAccountId,
		Source:         models.CloudEventSourceCost,
		EventType:      eventType,
		Level:          level,
		Status:         status,
		Provider:       provider,
		AccountId:      accountId,
		Region:         form.Region,
		ResourceType:   "cloud_cost",
		ResourceId:     firstNonEmpty(form.Period, time.Now().Format("2006-01")),
		ResourceName:   firstNonEmpty(source, mode, "cost_pull"),
		Title:          title,
		Message:        message,
		Payload:        payload,
	})
}

func RefreshCloudCostRecords(c *ctx.ServiceContext) e.Error {
	index, err := cloudCostAssetIndexForOrg(c)
	if err != nil {
		return err
	}
	if err := refreshCloudCostRecordsFromBills(c, index); err != nil {
		return err
	}
	return refreshCloudCostRecordsFromAssets(c)
}

func cloudCostBaseQuery(c *ctx.ServiceContext) *db.Session {
	return c.DB().Model(&models.CloudCostRecord{}).Where("org_id = ?", c.OrgId)
}

func applyCloudCostSearch(query *db.Session, form *forms.SearchCloudCostRecordForm) *db.Session {
	if form.Q != "" {
		q := "%" + strings.TrimSpace(form.Q) + "%"
		query = query.Where(`resource_name like ? or resource_id like ? or service like ? or provider like ? or
			account_id like ? or application like ? or business_line like ? or cost_center like ? or payload like ?`,
			q, q, q, q, q, q, q, q, q)
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
	if form.Service != "" {
		query = query.Where("service = ?", form.Service)
	}
	if form.Period != "" {
		query = query.Where("period = ?", form.Period)
	}
	if form.Currency != "" {
		query = query.Where("currency = ?", form.Currency)
	}
	if form.Source != "" {
		query = query.Where("source = ?", form.Source)
	}
	if form.CostCenter != "" {
		query = query.Where("cost_center = ?", form.CostCenter)
	}
	if form.Application != "" {
		query = query.Where("application = ?", form.Application)
	}
	if form.BusinessLine != "" {
		query = query.Where("business_line = ?", form.BusinessLine)
	}
	if form.MatchedAsset != nil {
		query = query.Where("matched_asset = ?", *form.MatchedAsset)
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
	return query
}

func cloudCostRecordFromImportItem(c *ctx.ServiceContext, form *forms.ImportCloudCostRecordForm, item forms.ImportCloudCostRecordItem, idx int) models.CloudCostRecord {
	payload := item.Payload
	if payload == nil {
		payload = models.ResAttrs{}
	}
	provider := normalizeProvider(firstNonEmpty(item.Provider, form.Provider,
		attrString(payload, "provider"), attrString(payload, "Provider"), attrString(payload, "cloudProvider")))
	if provider == "" {
		provider = "unknown"
	}
	currency := firstNonEmpty(item.Currency, cloudCostCurrencyFromPayload(payload), form.Currency)
	if currency == "" {
		currency = "CNY"
	}
	period := firstNonEmpty(item.Period, form.Period, cloudCostPeriodFromPayload(payload), time.Now().Format("2006-01"))
	resourceId := firstNonEmpty(item.ResourceId,
		attrString(payload, "resourceId"), attrString(payload, "ResourceId"), attrString(payload, "ResourceID"),
		attrString(payload, "lineItem/ResourceId"), attrString(payload, "lineItemResourceId"), attrString(payload, "line_item_resource_id"),
		attrString(payload, "instanceId"), attrString(payload, "InstanceId"), attrString(payload, "InstanceID"),
		attrString(payload, "resource_id"), attrString(payload, "resource_ocid"), attrString(payload, "ResourceOCID"),
		attrNestedString(payload, "resource", "global_name"), attrNestedString(payload, "resource", "name"))
	resourceName := firstNonEmpty(item.ResourceName,
		attrString(payload, "resourceName"), attrString(payload, "ResourceName"),
		attrString(payload, "instanceName"), attrString(payload, "InstanceName"),
		attrString(payload, "resource_name"), attrString(payload, "displayName"), attrString(payload, "DisplayName"),
		attrString(payload, "resourceTags/user:Name"), attrString(payload, "resourceTags/aws:cloudformation:stack-name"),
		cloudCostResourceNameFromId(resourceId))
	service := firstNonEmpty(item.Service,
		attrNestedString(payload, "service", "description"), attrNestedString(payload, "sku", "description"),
		attrString(payload, "serviceName"), attrString(payload, "ServiceName"),
		attrString(payload, "product/ProductName"), attrString(payload, "product/ServiceCode"), attrString(payload, "lineItem/ProductCode"),
		attrString(payload, "product_product_name"), attrString(payload, "product_servicecode"), attrString(payload, "line_item_product_code"),
		attrString(payload, "meterCategory"), attrString(payload, "MeterCategory"),
		attrString(payload, "BusinessCodeName"), attrString(payload, "ProductCodeName"), attrString(payload, "ProductName"),
		attrString(payload, "cloud_service_type_name"), attrString(payload, "cloud_service_type"),
		attrString(payload, "service"), attrString(payload, "Service"), attrString(payload, "service_description"), attrString(payload, "ServiceDescription"))
	accountId := firstNonEmpty(item.AccountId, form.AccountId,
		attrString(payload, "accountId"), attrString(payload, "AccountId"),
		attrString(payload, "lineItem/UsageAccountId"), attrString(payload, "bill/PayerAccountId"),
		attrString(payload, "line_item_usage_account_id"), attrString(payload, "payer_account_id"), attrString(payload, "usageAccountId"),
		attrString(payload, "OwnerUin"), attrString(payload, "PayerUin"), attrString(payload, "owner_uin"), attrString(payload, "payer_uin"),
		attrString(payload, "customer_id"), attrString(payload, "CustomerId"),
		attrString(payload, "tenant_id"), attrString(payload, "TenantId"), attrString(payload, "compartment_id"), attrString(payload, "CompartmentId"),
		attrString(payload, "subscriptionId"), attrString(payload, "SubscriptionId"),
		attrString(payload, "projectId"), attrString(payload, "ProjectId"), attrNestedString(payload, "project", "id"))
	region := firstNonEmpty(item.Region, form.Region,
		attrString(payload, "region"), attrString(payload, "Region"),
		attrString(payload, "product/region"), attrString(payload, "product_region"), attrString(payload, "availabilityDomain"), attrString(payload, "AvailabilityDomain"),
		attrString(payload, "RegionName"), attrString(payload, "RegionId"), attrString(payload, "region_id"),
		attrString(payload, "resourceLocation"), attrString(payload, "ResourceLocation"),
		attrString(payload, "location"), attrString(payload, "Location"), attrNestedString(payload, "location", "location"))
	source := firstNonEmpty(item.Source, form.Source, cloudCostImportSource(provider))
	sourceId := firstNonEmpty(item.SourceId, attrString(payload, "sourceId"), attrString(payload, "SourceId"), attrString(payload, "rowId"), attrString(payload, "id"))
	if sourceId == "" {
		sourceId = cloudCostImportSourceId(provider, source, period, resourceId, service, payload, idx)
	}
	cloudAccountId := firstNonEmptyId(item.CloudAccountId, form.CloudAccountId)
	record := models.CloudCostRecord{
		OrgId:          c.OrgId,
		ProjectId:      item.ProjectId,
		EnvId:          item.EnvId,
		CloudAccountId: cloudAccountId,
		Provider:       provider,
		AccountId:      accountId,
		Region:         region,
		Service:        service,
		ResourceType: firstNonEmpty(item.ResourceType,
			attrString(payload, "resourceType"), attrString(payload, "ResourceType"),
			attrString(payload, "ResourceTypeName"), attrString(payload, "resource_type"),
			attrString(payload, "product/instanceType"), attrString(payload, "product/instanceFamily"),
			attrString(payload, "lineItem/UsageType"), attrString(payload, "line_item_usage_type"), attrString(payload, "usageType"),
			attrString(payload, "meterCategory"), attrString(payload, "MeterCategory"),
			attrString(payload, "ProductCodeName"), attrString(payload, "cloud_service_type_name"),
			attrNestedString(payload, "sku", "description")),
		ResourceId:   resourceId,
		ResourceName: resourceName,
		Amount:       cloudCostImportAmount(item, payload),
		Currency:     currency,
		Period:       period,
		CostCenter: firstNonEmpty(item.CostCenter,
			attrString(payload, "costCenter"), attrString(payload, "CostCenter"), attrString(payload, "cost_centre"),
			attrString(payload, "resourceTags/user:CostCenter"), attrString(payload, "resourceTags/cost-center"),
			attrString(payload, "ProjectName"), attrString(payload, "enterprise_project_name")),
		Owner: firstNonEmpty(item.Owner,
			attrString(payload, "owner"), attrString(payload, "Owner"), attrString(payload, "resourceTags/user:Owner")),
		Application: firstNonEmpty(item.Application,
			attrString(payload, "application"), attrString(payload, "Application"), attrString(payload, "app"), attrString(payload, "resourceTags/user:Application")),
		BusinessLine: firstNonEmpty(item.BusinessLine,
			attrString(payload, "businessLine"), attrString(payload, "BusinessLine"), attrString(payload, "business_line"), attrString(payload, "resourceTags/user:BusinessLine")),
		Source:   source,
		SourceId: sourceId,
		Payload: cloudProviderSanitizeAttrs(models.ResAttrs{
			"source":  source,
			"payload": payload,
		}),
	}
	if record.ResourceName == "" {
		record.ResourceName = firstNonEmpty(record.ResourceId, record.Service, fmt.Sprintf("import-row-%d", idx+1))
	}
	if record.ResourceId == "" {
		record.ResourceId = record.SourceId
	}
	return record
}

func cloudCostImportAmount(item forms.ImportCloudCostRecordItem, payload models.ResAttrs) float64 {
	if item.Amount != 0 {
		return item.Amount
	}
	for _, key := range []string{
		"amount", "Amount", "cost", "Cost", "pretaxCost", "PretaxCost",
		"CostInBillingCurrency", "costInBillingCurrency", "costAmount", "CostAmount",
		"lineItem/UnblendedCost", "lineItem/BlendedCost", "lineItem/NetUnblendedCost", "lineItem/NetAmortizedCost",
		"unblendedCost", "blendedCost", "netUnblendedCost", "amortizedCost",
		"RealTotalCost", "TotalCost", "CashPayAmount", "VoucherPayAmount",
		"official_amount", "cash_amount", "credit_amount", "coupon_amount",
		"computed_amount", "ComputedAmount", "myCost", "MyCost",
	} {
		if amount, ok := cloudCostFloat(payload[key]); ok {
			return amount
		}
	}
	for _, path := range [][]string{
		{"Metrics", "UnblendedCost", "Amount"},
		{"Metrics", "BlendedCost", "Amount"},
		{"Metrics", "AmortizedCost", "Amount"},
		{"Metrics", "NetUnblendedCost", "Amount"},
		{"metrics", "unblendedCost", "amount"},
		{"metrics", "blendedCost", "amount"},
	} {
		if amount, ok := cloudCostFloat(attrPathValue(payload, path...)); ok {
			return amount
		}
	}
	return 0
}

func cloudCostCurrencyFromPayload(payload models.ResAttrs) string {
	return firstNonEmpty(
		attrString(payload, "currency"), attrString(payload, "Currency"),
		attrString(payload, "BillingCurrencyCode"), attrString(payload, "billingCurrencyCode"),
		attrString(payload, "lineItem/CurrencyCode"), attrString(payload, "pricing/currency"),
		attrString(payload, "currencyCode"), attrString(payload, "CurrencyCode"), attrString(payload, "pricing_currency"),
		attrPathString(payload, "Metrics", "UnblendedCost", "Unit"),
		attrPathString(payload, "Metrics", "BlendedCost", "Unit"),
		attrPathString(payload, "Metrics", "AmortizedCost", "Unit"),
		attrPathString(payload, "Metrics", "NetUnblendedCost", "Unit"),
	)
}

func cloudCostFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		amount, err := v.Float64()
		return amount, err == nil
	case string:
		amount, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return amount, err == nil
	default:
		return 0, false
	}
}

func cloudCostPeriodFromPayload(payload models.ResAttrs) string {
	for _, key := range []string{
		"period", "Period", "billingPeriod", "BillingPeriod",
		"billingPeriodStartDate", "BillingPeriodStartDate",
		"bill/BillingPeriodStartDate", "bill_billing_period_start_date",
		"lineItem/UsageStartDate", "lineItem/UsageEndDate", "line_item_usage_start_date",
		"usageStartTime", "usage_start_time", "date", "Date",
		"timeUsageStarted", "time_usage_started", "timeUsageEnded", "time_usage_ended",
		"BillMonth", "billMonth", "BillingCycle", "billing_cycle",
		"FeeBeginTime", "FeeEndTime", "effective_time", "EffectiveTime",
	} {
		if period := cloudCostPeriodValue(attrString(payload, key)); period != "" {
			return period
		}
	}
	return ""
}

func cloudCostPeriodValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 7 && value[4] == '-' {
		return value[:7]
	}
	if len(value) >= 6 {
		digits := strings.NewReplacer("-", "", "/", "", ".", "").Replace(value)
		if len(digits) >= 6 {
			return digits[:4] + "-" + digits[4:6]
		}
	}
	return ""
}

func cloudCostImportSource(provider string) string {
	switch normalizeProvider(provider) {
	case "aws":
		return models.CloudCostSourceAwsCurExport
	case "oci":
		return models.CloudCostSourceOciUsageCostExport
	case "azure":
		return models.CloudCostSourceAzureBillingExport
	case "gcp":
		return models.CloudCostSourceGcpBillingExport
	case "tencentcloud":
		return models.CloudCostSourceTencentBillingExport
	case "huawei":
		return models.CloudCostSourceHuaweiBillingExport
	default:
		return models.CloudCostSourceImport
	}
}

func cloudCostImportSourceId(provider string, source string, period string, resourceId string, service string, payload models.ResAttrs, idx int) string {
	body, _ := json.Marshal(payload)
	seed := strings.Join([]string{provider, source, period, resourceId, service, string(body), strconv.Itoa(idx)}, "\x00")
	sum := sha1.Sum([]byte(seed))
	return hex.EncodeToString(sum[:])
}

func cloudCostPullCloudAccount(c *ctx.ServiceContext, form *forms.PullCloudCostRecordForm) (*cmdbCloudAccount, e.Error) {
	provider := normalizeProvider(form.Provider)
	if form.CloudAccountId == "" {
		return &cmdbCloudAccount{
			Provider:    provider,
			AccountId:   form.AccountId,
			Credentials: map[string]string{},
		}, nil
	}
	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, form.CloudAccountId)
	if err != nil {
		return nil, err
	}
	credentials := credentialMapWithRegions(provider, credentialMapFromCloudCredentials(account.Credentials), []string(account.Regions))
	return buildCmdbCloudAccount(
		account.Id,
		models.CmdbCloudAccountSourceCloudAccount,
		account.Name,
		account.Description,
		account.Provider,
		firstNonEmpty(form.AccountId, account.AccountId),
		account.UpdatedAt,
		[]string(account.Regions),
		credentials,
	), nil
}

func cloudCostPullSource(provider, source string) string {
	if source != "" {
		return source
	}
	switch provider {
	case "aws":
		return models.CloudCostSourceAwsCurExport
	case "oci":
		return models.CloudCostSourceOciUsageCostExport
	case "azure":
		return models.CloudCostSourceAzureBillingExport
	case "gcp":
		return models.CloudCostSourceGcpBillingExport
	case "tencentcloud":
		return models.CloudCostSourceTencentBillingExport
	case "huawei":
		return models.CloudCostSourceHuaweiBillingExport
	default:
		return models.CloudCostSourceImport
	}
}

func cloudCostPullSourceURL(provider string, form *forms.PullCloudCostRecordForm, account *cmdbCloudAccount) string {
	if form.SourceURL != "" {
		return strings.TrimSpace(form.SourceURL)
	}
	if account == nil {
		return ""
	}
	keys := []string{"CLOUD_COST_EXPORT_URL", "CLOUD_BILLING_EXPORT_URL"}
	switch provider {
	case "aws":
		keys = append([]string{"AWS_CUR_EXPORT_URL", "AWS_COST_EXPLORER_EXPORT_URL", "AWS_BILLING_EXPORT_URL", "AWS_COST_EXPORT_URL"}, keys...)
	case "oci":
		keys = append([]string{"OCI_USAGE_COST_EXPORT_URL", "OCI_BILLING_EXPORT_URL", "OCI_COST_EXPORT_URL", "ORACLE_USAGE_COST_EXPORT_URL"}, keys...)
	case "azure":
		keys = append([]string{"AZURE_BILLING_EXPORT_URL", "AZURE_COST_EXPORT_URL"}, keys...)
	case "gcp":
		keys = append([]string{"GCP_BILLING_EXPORT_URL", "GCP_COST_EXPORT_URL", "GOOGLE_BILLING_EXPORT_URL"}, keys...)
	case "tencentcloud":
		keys = append([]string{"TENCENTCLOUD_BILLING_EXPORT_URL", "TENCENT_BILLING_EXPORT_URL", "TENCENTCLOUD_COST_EXPORT_URL"}, keys...)
	case "huawei":
		keys = append([]string{"HUAWEI_BILLING_EXPORT_URL", "HUAWEICLOUD_BILLING_EXPORT_URL", "HUAWEI_COST_EXPORT_URL"}, keys...)
	}
	for _, key := range keys {
		if value := strings.TrimSpace(account.Credentials[key]); value != "" {
			return value
		}
	}
	return ""
}

func cloudCostPullSourceIndexURL(provider string, form *forms.PullCloudCostRecordForm, account *cmdbCloudAccount) string {
	if form.SourceIndexURL != "" {
		return strings.TrimSpace(form.SourceIndexURL)
	}
	if account == nil {
		return ""
	}
	keys := []string{"CLOUD_COST_EXPORT_INDEX_URL", "CLOUD_BILLING_EXPORT_INDEX_URL"}
	switch provider {
	case "aws":
		keys = append([]string{"AWS_CUR_EXPORT_INDEX_URL", "AWS_COST_EXPLORER_EXPORT_INDEX_URL", "AWS_BILLING_EXPORT_INDEX_URL", "AWS_COST_EXPORT_INDEX_URL"}, keys...)
	case "oci":
		keys = append([]string{"OCI_USAGE_COST_EXPORT_INDEX_URL", "OCI_BILLING_EXPORT_INDEX_URL", "OCI_COST_EXPORT_INDEX_URL", "ORACLE_USAGE_COST_EXPORT_INDEX_URL"}, keys...)
	case "azure":
		keys = append([]string{"AZURE_BILLING_EXPORT_INDEX_URL", "AZURE_COST_EXPORT_INDEX_URL"}, keys...)
	case "gcp":
		keys = append([]string{"GCP_BILLING_EXPORT_INDEX_URL", "GCP_COST_EXPORT_INDEX_URL", "GOOGLE_BILLING_EXPORT_INDEX_URL"}, keys...)
	case "tencentcloud":
		keys = append([]string{"TENCENTCLOUD_BILLING_EXPORT_INDEX_URL", "TENCENT_BILLING_EXPORT_INDEX_URL", "TENCENTCLOUD_COST_EXPORT_INDEX_URL"}, keys...)
	case "huawei":
		keys = append([]string{"HUAWEI_BILLING_EXPORT_INDEX_URL", "HUAWEICLOUD_BILLING_EXPORT_INDEX_URL", "HUAWEI_COST_EXPORT_INDEX_URL"}, keys...)
	}
	for _, key := range keys {
		if value := strings.TrimSpace(account.Credentials[key]); value != "" {
			return value
		}
	}
	return ""
}

func cloudCostPullSourceObjectConfig(provider string, form *forms.PullCloudCostRecordForm, account *cmdbCloudAccount) cloudCostObjectStorageConfig {
	cfg := cloudCostObjectStorageConfig{
		Provider: strings.TrimSpace(form.SourceObjectProvider),
		Endpoint: strings.TrimSpace(form.SourceObjectEndpoint),
		Bucket:   strings.TrimSpace(form.SourceObjectBucket),
		Prefix:   strings.TrimSpace(form.SourceObjectPrefix),
		BaseURL:  strings.TrimSpace(form.SourceObjectBaseURL),
		Region:   strings.TrimSpace(form.Region),
		Account:  account,
	}
	if account != nil {
		if cfg.Provider == "" {
			cfg.Provider = cloudCostCredentialValue(account, provider, "OBJECT_PROVIDER")
		}
		if cfg.Endpoint == "" {
			cfg.Endpoint = cloudCostCredentialValue(account, provider, "OBJECT_ENDPOINT")
		}
		if cfg.Bucket == "" {
			cfg.Bucket = cloudCostCredentialValue(account, provider, "OBJECT_BUCKET")
		}
		if cfg.Prefix == "" {
			cfg.Prefix = cloudCostCredentialValue(account, provider, "OBJECT_PREFIX")
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = cloudCostCredentialValue(account, provider, "OBJECT_BASE_URL")
		}
		if cfg.Region == "" {
			cfg.Region = firstNonEmpty(
				cloudCostCredentialValue(account, provider, "OBJECT_REGION"),
				account.Credentials["AWS_REGION"],
				account.Credentials["AWS_DEFAULT_REGION"],
				account.Credentials["GCP_REGION"],
				account.Credentials["AZURE_REGION"],
				account.Credentials["OCI_REGION"],
			)
		}
	}
	cfg.Provider = strings.ToLower(firstNonEmpty(cfg.Provider, cloudCostDefaultObjectProvider(provider)))
	switch cfg.Provider {
	case "azure":
		cfg.Provider = "azure_blob"
	case "oci", "oci_object", "oracle_object_storage":
		cfg.Provider = "oci_object_storage"
	}
	if cfg.Provider == "s3" && cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.Provider == "oci_object_storage" && cfg.Endpoint == "" && cfg.Region != "" {
		cfg.Endpoint = "https://" + ociServiceHost("objectstorage", cfg.Region)
	}
	if cfg.Provider == "" || cfg.Endpoint == "" || cfg.Bucket == "" {
		return cloudCostObjectStorageConfig{}
	}
	return cfg
}

func cloudCostCredentialValue(account *cmdbCloudAccount, provider string, suffix string) string {
	if account == nil {
		return ""
	}
	keys := []string{
		"CLOUD_COST_" + suffix,
		"CLOUD_BILLING_" + suffix,
		"S3_" + suffix,
		"AWS_S3_" + suffix,
	}
	switch normalizeProvider(provider) {
	case "aws":
		keys = append([]string{
			"AWS_CUR_" + suffix,
			"AWS_COST_EXPLORER_" + suffix,
			"AWS_COST_" + suffix,
			"AWS_BILLING_" + suffix,
		}, keys...)
	case "oci":
		keys = append([]string{
			"OCI_USAGE_COST_" + suffix,
			"OCI_COST_" + suffix,
			"OCI_BILLING_" + suffix,
			"ORACLE_USAGE_COST_" + suffix,
		}, keys...)
	case "azure":
		keys = append([]string{
			"AZURE_COST_" + suffix,
			"AZURE_BILLING_" + suffix,
			"AZURE_BLOB_" + suffix,
		}, keys...)
	case "gcp":
		keys = append([]string{
			"GCP_COST_" + suffix,
			"GCP_BILLING_" + suffix,
			"GOOGLE_BILLING_" + suffix,
			"GCS_" + suffix,
		}, keys...)
	}
	for _, key := range keys {
		if value := strings.TrimSpace(account.Credentials[key]); value != "" {
			return value
		}
	}
	return ""
}

func cloudCostDefaultObjectProvider(provider string) string {
	switch normalizeProvider(provider) {
	case "aws":
		return "s3"
	case "azure":
		return "azure_blob"
	case "gcp":
		return "gcs"
	case "oci":
		return "oci_object_storage"
	case "tencentcloud", "huawei":
		return "s3"
	default:
		return ""
	}
}

func cloudCostReadURL(form *forms.PullCloudCostRecordForm, rawURL string) ([]byte, error) {
	return cloudCostReadURLWithPrepare(form, rawURL, "application/json,text/csv,text/tab-separated-values,application/gzip,application/zip,*/*", nil)
}

func cloudCostReadURLWithPrepare(form *forms.PullCloudCostRecordForm, rawURL string, accept string, prepare func(*http.Request) error) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", firstNonEmpty(accept, "application/json"))
	for key, value := range form.SourceHeaders {
		headerValue := fmt.Sprintf("%v", value)
		if strings.TrimSpace(key) != "" && strings.TrimSpace(headerValue) != "" {
			req.Header.Set(key, headerValue)
		}
	}
	if prepare != nil {
		if err := prepare(req); err != nil {
			return nil, err
		}
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cost export url status %d: %s", resp.StatusCode, compactCloudCostErrorText(string(body), 240))
	}
	return body, nil
}

func cloudCostPullFromURL(c *ctx.ServiceContext, form *forms.PullCloudCostRecordForm, provider, period, currency, source, sourceURL string) ([]forms.ImportCloudCostRecordItem, error) {
	body, err := cloudCostReadURL(form, sourceURL)
	if err != nil {
		return nil, err
	}
	items, err := cloudCostPullItemsFromExportBody(provider, period, currency, source, body, sourceURL)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		items[idx].CloudAccountId = form.CloudAccountId
		items[idx].AccountId = firstNonEmpty(items[idx].AccountId, form.AccountId)
		items[idx].Region = firstNonEmpty(items[idx].Region, form.Region)
		items[idx].Payload["sourceUrl"] = cloudCostSourceURLPreview(sourceURL)
		items[idx].Payload["orgId"] = c.OrgId.String()
	}
	return items, nil
}

func cloudCostPullFromObjectURL(c *ctx.ServiceContext, form *forms.PullCloudCostRecordForm, provider, period, currency, source string, cfg cloudCostObjectStorageConfig, sourceURL string) ([]forms.ImportCloudCostRecordItem, error) {
	body, err := cloudCostReadObjectURL(form, cfg, sourceURL, "application/json,text/csv,text/tab-separated-values,application/gzip,application/zip,*/*")
	if err != nil {
		return nil, err
	}
	items, err := cloudCostPullItemsFromExportBody(provider, period, currency, source, body, sourceURL)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		items[idx].CloudAccountId = form.CloudAccountId
		items[idx].AccountId = firstNonEmpty(items[idx].AccountId, form.AccountId)
		items[idx].Region = firstNonEmpty(items[idx].Region, form.Region)
		items[idx].Payload["sourceUrl"] = cloudCostSourceURLPreview(sourceURL)
		items[idx].Payload["orgId"] = c.OrgId.String()
	}
	return items, nil
}

func cloudCostPullFromIndex(c *ctx.ServiceContext, form *forms.PullCloudCostRecordForm, provider, period, currency, source, sourceIndexURL string) (*cloudCostPullIndexResult, error) {
	body, err := cloudCostReadURL(form, sourceIndexURL)
	if err != nil {
		return nil, err
	}
	files, err := cloudCostExportFilesFromJSON(body, sourceIndexURL)
	if err != nil {
		return nil, err
	}
	return cloudCostPullFromExportFiles(c, form, provider, period, currency, source, sourceIndexURL, files, nil)
}

func cloudCostPullFromObjectStorageList(c *ctx.ServiceContext, form *forms.PullCloudCostRecordForm, provider, period, currency, source string, cfg cloudCostObjectStorageConfig) (*cloudCostPullIndexResult, error) {
	files, err := cloudCostObjectStorageListFiles(form, cfg)
	if err != nil {
		return nil, err
	}
	extra := models.ResAttrs{
		"sourceObjectProvider": cfg.Provider,
		"sourceObjectEndpoint": cloudCostSourceURLPreview(cfg.Endpoint),
		"sourceObjectBucket":   cfg.Bucket,
		"sourceObjectPrefix":   cfg.Prefix,
	}
	puller := func(fileURL string) ([]forms.ImportCloudCostRecordItem, error) {
		return cloudCostPullFromObjectURL(c, form, provider, period, currency, source, cfg, fileURL)
	}
	return cloudCostPullFromExportFilesWithPuller(c, form, provider, period, currency, source, cfg.Endpoint, files, extra, puller)
}

func cloudCostObjectStorageListFiles(form *forms.PullCloudCostRecordForm, cfg cloudCostObjectStorageConfig) ([]cloudCostExportFile, error) {
	switch cfg.Provider {
	case "s3":
		return cloudCostS3ListFiles(form, cfg)
	case "gcs":
		return cloudCostGCSListFiles(form, cfg)
	case "azure_blob":
		return cloudCostAzureBlobListFiles(form, cfg)
	case "oci_object_storage":
		return cloudCostOCIObjectStorageListFiles(form, cfg)
	default:
		return nil, fmt.Errorf("unsupported cost object storage provider %s", cfg.Provider)
	}
}

func cloudCostS3ListFiles(form *forms.PullCloudCostRecordForm, cfg cloudCostObjectStorageConfig) ([]cloudCostExportFile, error) {
	bucketURL := cloudCostJoinURLPath(cfg.Endpoint, cfg.Bucket)
	type s3Object struct {
		Key          string `xml:"Key"`
		LastModified string `xml:"LastModified"`
		ETag         string `xml:"ETag"`
		Size         int64  `xml:"Size"`
	}
	type s3ListBucketResult struct {
		IsTruncated           bool       `xml:"IsTruncated"`
		NextContinuationToken string     `xml:"NextContinuationToken"`
		Contents              []s3Object `xml:"Contents"`
	}
	files := make([]cloudCostExportFile, 0)
	continuationToken := ""
	for page := 0; page < 100; page++ {
		query := map[string]string{
			"list-type": "2",
			"prefix":    cfg.Prefix,
		}
		if continuationToken != "" {
			query["continuation-token"] = continuationToken
		}
		listURL := cloudCostURLWithQuery(bucketURL, query)
		body, err := cloudCostReadObjectURL(form, cfg, listURL, "application/xml")
		if err != nil {
			return nil, err
		}
		result := s3ListBucketResult{}
		if err := xml.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("parse s3 list objects failed: %w", err)
		}
		for _, item := range result.Contents {
			key := strings.TrimSpace(item.Key)
			if key == "" {
				continue
			}
			files = append(files, cloudCostExportFile{
				URL:          cloudCostObjectFileURL(cfg, bucketURL, key),
				Key:          key,
				Cursor:       firstNonEmpty(item.LastModified, key),
				Size:         item.Size,
				ETag:         strings.Trim(item.ETag, `"`),
				LastModified: item.LastModified,
			})
		}
		continuationToken = strings.TrimSpace(result.NextContinuationToken)
		if !result.IsTruncated || continuationToken == "" {
			return files, nil
		}
	}
	return files, fmt.Errorf("s3 list objects exceeded max pagination pages")
}

func cloudCostGCSListFiles(form *forms.PullCloudCostRecordForm, cfg cloudCostObjectStorageConfig) ([]cloudCostExportFile, error) {
	base := strings.TrimRight(cfg.Endpoint, "/")
	type gcsObject struct {
		Name      string `json:"name"`
		Updated   string `json:"updated"`
		MediaLink string `json:"mediaLink"`
		ETag      string `json:"etag"`
		Size      string `json:"size"`
	}
	files := make([]cloudCostExportFile, 0)
	pageToken := ""
	for page := 0; page < 100; page++ {
		query := map[string]string{
			"prefix": cfg.Prefix,
		}
		if pageToken != "" {
			query["pageToken"] = pageToken
		}
		listURL := cloudCostURLWithQuery(cloudCostJoinURLPath(base, "b", cfg.Bucket, "o"), query)
		body, err := cloudCostReadObjectURL(form, cfg, listURL, "application/json")
		if err != nil {
			return nil, err
		}
		var result struct {
			NextPageToken string      `json:"nextPageToken"`
			Items         []gcsObject `json:"items"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("parse gcs list objects failed: %w", err)
		}
		for _, item := range result.Items {
			key := strings.TrimSpace(item.Name)
			if key == "" {
				continue
			}
			fileURL := strings.TrimSpace(item.MediaLink)
			if fileURL == "" {
				if cfg.BaseURL != "" {
					fileURL = cloudCostJoinURLPath(cfg.BaseURL, key)
				} else {
					fileURL = cloudCostURLWithQuery(cloudCostJoinURLPath(base, "download", "storage", "v1", "b", cfg.Bucket, "o", key), map[string]string{"alt": "media"})
				}
			}
			files = append(files, cloudCostExportFile{
				URL:          fileURL,
				Key:          key,
				Cursor:       firstNonEmpty(item.Updated, key),
				Size:         cloudCostParseInt64(item.Size),
				ETag:         item.ETag,
				LastModified: item.Updated,
			})
		}
		pageToken = strings.TrimSpace(result.NextPageToken)
		if pageToken == "" {
			return files, nil
		}
	}
	return files, fmt.Errorf("gcs list objects exceeded max pagination pages")
}

func cloudCostAzureBlobListFiles(form *forms.PullCloudCostRecordForm, cfg cloudCostObjectStorageConfig) ([]cloudCostExportFile, error) {
	containerURL := cloudCostJoinURLPath(cfg.Endpoint, cfg.Bucket)
	type azureBlobProperties struct {
		LastModified  string `xml:"Last-Modified"`
		ETag          string `xml:"Etag"`
		ContentLength int64  `xml:"Content-Length"`
	}
	type azureBlob struct {
		Name       string              `xml:"Name"`
		URL        string              `xml:"Url"`
		Properties azureBlobProperties `xml:"Properties"`
	}
	var result struct {
		NextMarker string `xml:"NextMarker"`
		Blobs      struct {
			Blob []azureBlob `xml:"Blob"`
		} `xml:"Blobs"`
	}
	files := make([]cloudCostExportFile, 0)
	marker := ""
	for page := 0; page < 100; page++ {
		query := map[string]string{
			"restype": "container",
			"comp":    "list",
			"prefix":  cfg.Prefix,
		}
		if marker != "" {
			query["marker"] = marker
		}
		listURL := cloudCostURLWithQuery(containerURL, query)
		body, err := cloudCostReadObjectURL(form, cfg, listURL, "application/xml")
		if err != nil {
			return nil, err
		}
		result = struct {
			NextMarker string `xml:"NextMarker"`
			Blobs      struct {
				Blob []azureBlob `xml:"Blob"`
			} `xml:"Blobs"`
		}{}
		if err := xml.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("parse azure blob list failed: %w", err)
		}
		for _, item := range result.Blobs.Blob {
			key := strings.TrimSpace(item.Name)
			if key == "" {
				continue
			}
			fileURL := strings.TrimSpace(item.URL)
			if fileURL == "" {
				fileURL = cloudCostObjectFileURL(cfg, containerURL, key)
			}
			files = append(files, cloudCostExportFile{
				URL:          fileURL,
				Key:          key,
				Cursor:       firstNonEmpty(item.Properties.LastModified, key),
				Size:         item.Properties.ContentLength,
				ETag:         strings.Trim(item.Properties.ETag, `"`),
				LastModified: item.Properties.LastModified,
			})
		}
		marker = strings.TrimSpace(result.NextMarker)
		if marker == "" {
			return files, nil
		}
	}
	return files, fmt.Errorf("azure blob list exceeded max pagination pages")
}

func cloudCostOCIObjectStorageListFiles(form *forms.PullCloudCostRecordForm, cfg cloudCostObjectStorageConfig) ([]cloudCostExportFile, error) {
	namespace, err := cloudCostOCIObjectStorageNamespace(cfg)
	if err != nil {
		return nil, err
	}
	listBaseURL := cloudCostJoinURLPath(cfg.Endpoint, "n", namespace, "b", cfg.Bucket, "o")
	type ociObject struct {
		Name         string `json:"name"`
		TimeCreated  string `json:"timeCreated"`
		TimeModified string `json:"timeModified"`
		ETag         string `json:"etag"`
		Size         int64  `json:"size"`
	}
	files := make([]cloudCostExportFile, 0)
	start := ""
	for page := 0; page < 100; page++ {
		query := map[string]string{
			"prefix": cfg.Prefix,
			"limit":  "1000",
		}
		if start != "" {
			query["start"] = start
		}
		listURL := cloudCostURLWithQuery(listBaseURL, query)
		body, err := cloudCostReadObjectURL(form, cfg, listURL, "application/json")
		if err != nil {
			return nil, err
		}
		var result struct {
			NextStartWith string      `json:"nextStartWith"`
			Objects       []ociObject `json:"objects"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("parse oci object storage list failed: %w", err)
		}
		for _, item := range result.Objects {
			key := strings.TrimSpace(item.Name)
			if key == "" {
				continue
			}
			files = append(files, cloudCostExportFile{
				URL:          cloudCostOCIObjectFileURL(cfg, namespace, key),
				Key:          key,
				Cursor:       firstNonEmpty(item.TimeModified, item.TimeCreated, key),
				Size:         item.Size,
				ETag:         item.ETag,
				LastModified: firstNonEmpty(item.TimeModified, item.TimeCreated),
			})
		}
		start = strings.TrimSpace(result.NextStartWith)
		if start == "" {
			return files, nil
		}
	}
	return files, fmt.Errorf("oci object storage list exceeded max pagination pages")
}

func cloudCostOCIObjectStorageNamespace(cfg cloudCostObjectStorageConfig) (string, error) {
	if namespace := cloudCostObjectCredential(cfg,
		"OCI_OBJECT_STORAGE_NAMESPACE",
		"OCI_NAMESPACE",
		"ORACLE_OBJECT_STORAGE_NAMESPACE",
		"OCI_USAGE_COST_OBJECT_NAMESPACE",
		"OCI_USAGE_COST_OBJECT_STORAGE_NAMESPACE",
		"OCI_COST_OBJECT_NAMESPACE",
		"OCI_BILLING_OBJECT_NAMESPACE",
		"CLOUD_COST_OBJECT_NAMESPACE",
		"CLOUD_BILLING_OBJECT_NAMESPACE",
	); namespace != "" {
		return namespace, nil
	}
	if cfg.Account == nil {
		return "", fmt.Errorf("oci object storage namespace requires OCI_OBJECT_STORAGE_NAMESPACE")
	}
	if cfg.Region == "" {
		return "", fmt.Errorf("oci object storage namespace lookup requires region")
	}
	return ociObjectStorageNamespace(context.Background(), cfg.Account, cfg.Region)
}

func cloudCostPullFromExportFiles(c *ctx.ServiceContext, form *forms.PullCloudCostRecordForm, provider, period, currency, source, sourceIndexURL string, files []cloudCostExportFile, extraPayload models.ResAttrs) (*cloudCostPullIndexResult, error) {
	return cloudCostPullFromExportFilesWithPuller(c, form, provider, period, currency, source, sourceIndexURL, files, extraPayload, nil)
}

func cloudCostPullFromExportFilesWithPuller(c *ctx.ServiceContext, form *forms.PullCloudCostRecordForm, provider, period, currency, source, sourceIndexURL string, files []cloudCostExportFile, extraPayload models.ResAttrs, puller func(string) ([]forms.ImportCloudCostRecordItem, error)) (*cloudCostPullIndexResult, error) {
	cursor := strings.TrimSpace(form.Cursor)
	filtered := make([]cloudCostExportFile, 0, len(files))
	skippedFiles := make([]models.ResAttrs, 0)
	seenFiles := make(map[string]struct{})
	for _, file := range files {
		if file.URL == "" {
			continue
		}
		identity := cloudCostExportFileIdentity(file)
		if identity != "" {
			if _, ok := seenFiles[identity]; ok {
				skippedFiles = append(skippedFiles, cloudCostSkippedExportFileAttrs(file, "duplicate"))
				continue
			}
			seenFiles[identity] = struct{}{}
		}
		if cursor != "" && file.Cursor <= cursor {
			continue
		}
		filtered = append(filtered, file)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Cursor < filtered[j].Cursor
	})
	maxFiles := cloudCostPullIndexMaxFiles(form.MaxFiles)
	if len(filtered) > maxFiles {
		filtered = filtered[:maxFiles]
	}
	result := &cloudCostPullIndexResult{
		Cursor:         cursor,
		SourceIndexURL: sourceIndexURL,
		Files:          make([]models.ResAttrs, 0, len(filtered)),
		SkippedFiles:   skippedFiles,
		Records:        make([]forms.ImportCloudCostRecordItem, 0),
	}
	for _, file := range filtered {
		if puller == nil {
			puller = func(fileURL string) ([]forms.ImportCloudCostRecordItem, error) {
				return cloudCostPullFromURL(c, form, provider, period, currency, source, fileURL)
			}
		}
		items, err := puller(file.URL)
		if err != nil {
			return nil, fmt.Errorf("pull cost export file %s failed: %w", cloudCostSourceURLPreview(file.URL), err)
		}
		for idx := range items {
			items[idx].Payload["sourceIndexUrl"] = cloudCostSourceURLPreview(sourceIndexURL)
			items[idx].Payload["sourceFileKey"] = file.Key
			items[idx].Payload["sourceFileCursor"] = file.Cursor
			if file.Size > 0 {
				items[idx].Payload["sourceFileSize"] = file.Size
			}
			if file.ETag != "" {
				items[idx].Payload["sourceFileETag"] = file.ETag
			}
			if file.LastModified != "" {
				items[idx].Payload["sourceFileLastModified"] = file.LastModified
			}
			for key, value := range extraPayload {
				items[idx].Payload[key] = value
			}
		}
		result.Records = append(result.Records, items...)
		result.Files = append(result.Files, cloudCostExportFileAttrs(file, len(items)))
		result.NextCursor = file.Cursor
	}
	return result, nil
}

func cloudCostExportFileIdentity(file cloudCostExportFile) string {
	key := firstNonEmpty(file.Key, file.URL)
	if key == "" {
		return ""
	}
	version := firstNonEmpty(file.ETag, file.Cursor, file.LastModified)
	return key + "|" + version
}

func cloudCostExportFileAttrs(file cloudCostExportFile, records int) models.ResAttrs {
	attrs := models.ResAttrs{
		"key":     file.Key,
		"cursor":  file.Cursor,
		"url":     cloudCostSourceURLPreview(file.URL),
		"records": records,
	}
	cloudCostApplyExportFileMetadata(attrs, file)
	return attrs
}

func cloudCostSkippedExportFileAttrs(file cloudCostExportFile, reason string) models.ResAttrs {
	attrs := models.ResAttrs{
		"key":    file.Key,
		"cursor": file.Cursor,
		"url":    cloudCostSourceURLPreview(file.URL),
		"reason": reason,
	}
	cloudCostApplyExportFileMetadata(attrs, file)
	return attrs
}

func cloudCostApplyExportFileMetadata(attrs models.ResAttrs, file cloudCostExportFile) {
	if attrs == nil {
		return
	}
	if file.Size > 0 {
		attrs["size"] = file.Size
	}
	if file.ETag != "" {
		attrs["eTag"] = file.ETag
	}
	if file.LastModified != "" {
		attrs["lastModified"] = file.LastModified
	}
}

func cloudCostPullIndexMaxFiles(value int) int {
	if value <= 0 {
		return 20
	}
	if value > 100 {
		return 100
	}
	return value
}

func cloudCostExportFilesFromJSON(body []byte, sourceIndexURL string) ([]cloudCostExportFile, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var raw interface{}
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse cost export index failed: %w", err)
	}
	files := cloudCostExportFilesFromValue(raw, sourceIndexURL)
	if len(files) == 0 {
		return nil, fmt.Errorf("成本导出索引没有可用文件")
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Cursor < files[j].Cursor
	})
	return files, nil
}

func cloudCostExportFilesFromValue(value interface{}, sourceIndexURL string) []cloudCostExportFile {
	switch typed := value.(type) {
	case []interface{}:
		files := make([]cloudCostExportFile, 0, len(typed))
		for _, item := range typed {
			if file, ok := cloudCostExportFileFromValue(item, sourceIndexURL); ok {
				files = append(files, file)
			}
		}
		return files
	case map[string]interface{}:
		attrs := models.ResAttrs(typed)
		for _, key := range []string{"files", "items", "exports", "objects"} {
			if list := cloudCostExportFilesFromValue(attrs[key], sourceIndexURL); len(list) > 0 {
				return list
			}
		}
		if file, ok := cloudCostExportFileFromValue(attrs, sourceIndexURL); ok {
			return []cloudCostExportFile{file}
		}
	case models.ResAttrs:
		return cloudCostExportFilesFromValue(map[string]interface{}(typed), sourceIndexURL)
	}
	return nil
}

func cloudCostExportFileFromValue(value interface{}, sourceIndexURL string) (cloudCostExportFile, bool) {
	switch typed := value.(type) {
	case string:
		rawURL := strings.TrimSpace(typed)
		if rawURL == "" {
			return cloudCostExportFile{}, false
		}
		resolvedURL := cloudCostResolveURL(sourceIndexURL, rawURL)
		return cloudCostExportFile{URL: resolvedURL, Key: rawURL, Cursor: rawURL}, true
	case map[string]interface{}:
		return cloudCostExportFileFromAttrs(models.ResAttrs(typed), sourceIndexURL)
	case models.ResAttrs:
		return cloudCostExportFileFromAttrs(typed, sourceIndexURL)
	default:
		return cloudCostExportFile{}, false
	}
}

func cloudCostExportFileFromAttrs(attrs models.ResAttrs, sourceIndexURL string) (cloudCostExportFile, bool) {
	rawURL := firstNonEmpty(
		attrString(attrs, "url"),
		attrString(attrs, "sourceUrl"),
		attrString(attrs, "downloadUrl"),
		attrString(attrs, "href"),
	)
	if rawURL == "" {
		return cloudCostExportFile{}, false
	}
	key := firstNonEmpty(attrString(attrs, "key"), attrString(attrs, "name"), attrString(attrs, "path"), rawURL)
	lastModified := firstNonEmpty(attrString(attrs, "lastModified"), attrString(attrs, "updatedAt"), attrString(attrs, "timeModified"), attrString(attrs, "timeCreated"))
	cursor := firstNonEmpty(attrString(attrs, "cursor"), lastModified, key, rawURL)
	return cloudCostExportFile{
		URL:          cloudCostResolveURL(sourceIndexURL, rawURL),
		Key:          key,
		Cursor:       cursor,
		Size:         cloudCostAttrInt64(attrs, "size", "contentLength", "content_length"),
		ETag:         firstNonEmpty(attrString(attrs, "eTag"), attrString(attrs, "etag"), attrString(attrs, "ETag")),
		LastModified: lastModified,
	}, true
}

func cloudCostAttrInt64(attrs models.ResAttrs, keys ...string) int64 {
	for _, key := range keys {
		if value := attrString(attrs, key); value != "" {
			return cloudCostParseInt64(value)
		}
	}
	return 0
}

func cloudCostParseInt64(value string) int64 {
	number, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return number
}

func cloudCostResolveURL(baseURL, rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.IsAbs() {
		return rawURL
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return rawURL
	}
	return base.ResolveReference(parsed).String()
}

func cloudCostObjectFileURL(cfg cloudCostObjectStorageConfig, fallbackBaseURL, key string) string {
	if cfg.BaseURL != "" {
		return cloudCostJoinURLPath(cfg.BaseURL, key)
	}
	return cloudCostJoinURLPath(fallbackBaseURL, key)
}

func cloudCostOCIObjectFileURL(cfg cloudCostObjectStorageConfig, namespace, key string) string {
	if cfg.BaseURL != "" {
		return cloudCostJoinURLPath(cfg.BaseURL, key)
	}
	return cloudCostJoinEscapedURLPath(cfg.Endpoint, "n", namespace, "b", cfg.Bucket, "o", key)
}

func cloudCostReadObjectURL(form *forms.PullCloudCostRecordForm, cfg cloudCostObjectStorageConfig, rawURL string, accept string) ([]byte, error) {
	return cloudCostReadURLWithPrepare(form, rawURL, accept, func(req *http.Request) error {
		return cloudCostAuthorizeObjectRequest(req, cfg)
	})
}

func cloudCostAuthorizeObjectRequest(req *http.Request, cfg cloudCostObjectStorageConfig) error {
	if req == nil || req.Header.Get("Authorization") != "" {
		return nil
	}
	switch cfg.Provider {
	case "s3":
		return cloudCostAuthorizeS3Request(req, cfg)
	case "gcs":
		return cloudCostAuthorizeGCSRequest(req, cfg)
	case "azure_blob":
		return cloudCostAuthorizeAzureBlobRequest(req, cfg)
	case "oci_object_storage":
		return cloudCostAuthorizeOCIObjectStorageRequest(req, cfg)
	default:
		return nil
	}
}

func cloudCostAuthorizeS3Request(req *http.Request, cfg cloudCostObjectStorageConfig) error {
	if cfg.Account == nil || strings.TrimSpace(cfg.Account.Credentials["AWS_ACCESS_KEY_ID"]) == "" || strings.TrimSpace(cfg.Account.Credentials["AWS_SECRET_ACCESS_KEY"]) == "" {
		return nil
	}
	return signAWSV4(req, nil, cfg.Account, "s3", firstNonEmpty(cfg.Region, "us-east-1"), time.Now().UTC())
}

func cloudCostAuthorizeGCSRequest(req *http.Request, cfg cloudCostObjectStorageConfig) error {
	if cfg.Account == nil {
		return nil
	}
	if strings.TrimSpace(cfg.Account.Credentials["GCP_ACCESS_TOKEN"]) == "" && strings.TrimSpace(cfg.Account.Credentials["GCP_SERVICE_ACCOUNT_JSON"]) == "" {
		return nil
	}
	token, err := gcpAccessToken(req.Context(), cfg.Account)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return nil
}

func cloudCostAuthorizeAzureBlobRequest(req *http.Request, cfg cloudCostObjectStorageConfig) error {
	if cfg.Account == nil {
		return nil
	}
	if sasToken := cloudCostAzureBlobSASToken(cfg); sasToken != "" {
		cloudCostAppendRawQuery(req.URL, sasToken)
		return nil
	}
	accountName := cloudCostAzureBlobAccountName(cfg)
	accountKey := cloudCostAzureBlobAccountKey(cfg)
	if accountName == "" || accountKey == "" {
		return nil
	}
	keyBytes, err := base64.StdEncoding.DecodeString(accountKey)
	if err != nil {
		return fmt.Errorf("decode azure storage account key failed: %w", err)
	}
	req.Header.Set("x-ms-date", time.Now().UTC().Format(http.TimeFormat))
	req.Header.Set("x-ms-version", "2021-12-02")
	stringToSign := strings.Join([]string{
		req.Method,
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
	}, "\n") + "\n" + cloudCostAzureCanonicalizedHeaders(req.Header) + cloudCostAzureCanonicalizedResource(accountName, req.URL)
	signature := base64.StdEncoding.EncodeToString(hmacSHA256(keyBytes, stringToSign))
	req.Header.Set("Authorization", fmt.Sprintf("SharedKey %s:%s", accountName, signature))
	return nil
}

func cloudCostAuthorizeOCIObjectStorageRequest(req *http.Request, cfg cloudCostObjectStorageConfig) error {
	if cfg.Account == nil {
		return nil
	}
	return signOCIRequest(req, cfg.Account, time.Now().UTC())
}

func cloudCostAzureBlobSASToken(cfg cloudCostObjectStorageConfig) string {
	return cloudCostObjectCredential(cfg,
		"CLOUD_COST_OBJECT_SAS_TOKEN",
		"CLOUD_BILLING_OBJECT_SAS_TOKEN",
		"AZURE_BILLING_OBJECT_SAS_TOKEN",
		"AZURE_BLOB_SAS_TOKEN",
		"AZURE_STORAGE_SAS_TOKEN",
		"AZURE_SAS_TOKEN",
	)
}

func cloudCostAzureBlobAccountName(cfg cloudCostObjectStorageConfig) string {
	if value := cloudCostObjectCredential(cfg,
		"AZURE_STORAGE_ACCOUNT",
		"AZURE_STORAGE_ACCOUNT_NAME",
		"AZURE_BLOB_ACCOUNT",
		"AZURE_BLOB_ACCOUNT_NAME",
		"AZURE_BILLING_STORAGE_ACCOUNT",
	); value != "" {
		return value
	}
	parsed, err := url.Parse(cfg.Endpoint)
	if err != nil || parsed.Host == "" {
		return ""
	}
	host := parsed.Host
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}
	parts := strings.Split(host, ".")
	if len(parts) > 2 && strings.Contains(host, "blob.") {
		return parts[0]
	}
	return ""
}

func cloudCostAzureBlobAccountKey(cfg cloudCostObjectStorageConfig) string {
	return cloudCostObjectCredential(cfg,
		"AZURE_STORAGE_KEY",
		"AZURE_STORAGE_ACCOUNT_KEY",
		"AZURE_BLOB_ACCOUNT_KEY",
		"AZURE_BILLING_STORAGE_KEY",
	)
}

func cloudCostObjectCredential(cfg cloudCostObjectStorageConfig, keys ...string) string {
	if cfg.Account == nil {
		return ""
	}
	for _, key := range keys {
		if value := strings.TrimSpace(cfg.Account.Credentials[key]); value != "" {
			return value
		}
	}
	return ""
}

func cloudCostAppendRawQuery(parsed *url.URL, rawQuery string) {
	if parsed == nil {
		return
	}
	rawQuery = strings.TrimSpace(rawQuery)
	rawQuery = strings.TrimPrefix(rawQuery, "?")
	rawQuery = strings.TrimPrefix(rawQuery, "&")
	if rawQuery == "" {
		return
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		if parsed.RawQuery == "" {
			parsed.RawQuery = rawQuery
		} else {
			parsed.RawQuery += "&" + rawQuery
		}
		return
	}
	query := parsed.Query()
	for key, items := range values {
		for _, item := range items {
			query.Add(key, item)
		}
	}
	parsed.RawQuery = query.Encode()
}

func cloudCostAzureCanonicalizedHeaders(headers http.Header) string {
	keys := make([]string, 0)
	for key := range headers {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "x-ms-") {
			keys = append(keys, lower)
		}
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		values := headers.Values(key)
		normalized := make([]string, 0, len(values))
		for _, value := range values {
			normalized = append(normalized, strings.Join(strings.Fields(value), " "))
		}
		builder.WriteString(key)
		builder.WriteByte(':')
		builder.WriteString(strings.Join(normalized, ","))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func cloudCostAzureCanonicalizedResource(accountName string, parsed *url.URL) string {
	pathValue := "/"
	if parsed != nil && parsed.EscapedPath() != "" {
		pathValue = parsed.EscapedPath()
	}
	builder := strings.Builder{}
	builder.WriteString("/")
	builder.WriteString(accountName)
	builder.WriteString(pathValue)
	if parsed == nil {
		return builder.String()
	}
	queryValues := map[string][]string{}
	for key, values := range parsed.Query() {
		lower := strings.ToLower(key)
		queryValues[lower] = append(queryValues[lower], values...)
	}
	keys := make([]string, 0, len(queryValues))
	for key := range queryValues {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		values := queryValues[key]
		sort.Strings(values)
		builder.WriteByte('\n')
		builder.WriteString(key)
		builder.WriteByte(':')
		builder.WriteString(strings.Join(values, ","))
	}
	return builder.String()
}

func cloudCostJoinURLPath(rawURL string, parts ...string) string {
	rawURL = strings.TrimRight(strings.TrimSpace(rawURL), "/")
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	pathValue := strings.TrimRight(parsed.Path, "/")
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), "/")
		if part == "" {
			continue
		}
		if pathValue == "" {
			pathValue = "/" + part
		} else {
			pathValue += "/" + part
		}
	}
	parsed.Path = pathValue
	return parsed.String()
}

func cloudCostJoinEscapedURLPath(rawURL string, parts ...string) string {
	rawURL = strings.TrimRight(strings.TrimSpace(rawURL), "/")
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	escapedPath := strings.TrimRight(parsed.EscapedPath(), "/")
	if escapedPath == "." {
		escapedPath = ""
	}
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), "/")
		if part == "" {
			continue
		}
		if escapedPath == "" {
			escapedPath = "/" + url.PathEscape(part)
		} else {
			escapedPath += "/" + url.PathEscape(part)
		}
	}
	pathValue, err := url.PathUnescape(escapedPath)
	if err != nil {
		parsed.Path = escapedPath
		return parsed.String()
	}
	parsed.Path = pathValue
	parsed.RawPath = escapedPath
	return parsed.String()
}

func cloudCostURLWithQuery(rawURL string, values map[string]string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	for key, value := range values {
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			query.Set(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func compactCloudCostErrorText(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if limit <= 0 || len([]rune(value)) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit]) + "..."
}

func cloudCostPullAzureCostManagement(account *cmdbCloudAccount, period, currency string) ([]forms.ImportCloudCostRecordItem, error) {
	if account == nil || account.Id == "" {
		return nil, fmt.Errorf("azure cost pull requires cloudAccountId when sourceUrl is not provided")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	token, err := azureAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	subscriptionId := azureSubscriptionId(account)
	if subscriptionId == "" {
		return nil, fmt.Errorf("azure cost pull requires AZURE_SUBSCRIPTION_ID")
	}
	from, to := cloudCostPullPeriodRange(period)
	payload := models.ResAttrs{
		"type":      "ActualCost",
		"timeframe": "Custom",
		"timePeriod": models.ResAttrs{
			"from": from,
			"to":   to,
		},
		"dataset": models.ResAttrs{
			"granularity": "Daily",
			"aggregation": models.ResAttrs{
				"totalCost": models.ResAttrs{
					"name":     "PreTaxCost",
					"function": "Sum",
				},
			},
			"grouping": []models.ResAttrs{
				{"type": "Dimension", "name": "ResourceId"},
				{"type": "Dimension", "name": "ServiceName"},
				{"type": "Dimension", "name": "ResourceLocation"},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	values.Set("api-version", "2023-03-01")
	reqURL := fmt.Sprintf("%s/subscriptions/%s/providers/Microsoft.CostManagement/query?%s", azureManagementEndpoint, url.PathEscape(subscriptionId), values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp := models.ResAttrs{}
	if err := azureDoJSON(req, token, &resp); err != nil {
		return nil, err
	}
	items := cloudCostPullItemsFromAzureQuery(resp, period, currency)
	for idx := range items {
		items[idx].CloudAccountId = account.Id
		items[idx].AccountId = firstNonEmpty(items[idx].AccountId, account.AccountId, subscriptionId)
		items[idx].Provider = "azure"
		items[idx].Source = models.CloudCostSourceAzureCostManagement
	}
	return items, nil
}

func cloudCostPullPeriodRange(period string) (string, string) {
	parsed, err := time.Parse("2006-01", period)
	if err != nil {
		parsed = time.Now()
	}
	from := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	return from.Format("2006-01-02T15:04:05Z"), to.Format("2006-01-02T15:04:05Z")
}

func cloudCostPullItemsFromJSON(provider, period, currency, source string, body []byte) ([]forms.ImportCloudCostRecordItem, error) {
	if items := cloudCostPullItemsFromJSONArray(provider, period, currency, source, body); len(items) > 0 {
		return items, nil
	}
	wrapper := models.ResAttrs{}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("decode cost export json: %w", err)
	}
	if records := resAttrsSlice(wrapper["records"]); len(records) > 0 {
		return cloudCostPullItemsFromAttrs(provider, period, currency, source, records), nil
	}
	if records := resAttrsSlice(wrapper["Records"]); len(records) > 0 {
		return cloudCostPullItemsFromAttrs(provider, period, currency, source, records), nil
	}
	if items := cloudCostPullItemsFromAzureQuery(wrapper, period, currency); len(items) > 0 {
		return items, nil
	}
	return nil, fmt.Errorf("cost export json must be an array, {records:[...]}, or Azure Cost Management query response")
}

func cloudCostPullItemsFromExportBody(provider, period, currency, source string, body []byte, sourceName string) ([]forms.ImportCloudCostRecordItem, error) {
	return cloudCostPullItemsFromNamedBody(provider, period, currency, source, body, sourceName, 0)
}

func cloudCostPullItemsFromNamedBody(provider, period, currency, source string, body []byte, sourceName string, depth int) ([]forms.ImportCloudCostRecordItem, error) {
	if depth > 3 {
		return nil, fmt.Errorf("cost export archive nesting too deep")
	}
	name := strings.ToLower(strings.TrimSpace(cloudCostExportFileName(sourceName)))
	if cloudCostLooksLikeGzip(body, name) {
		unzipped, err := cloudCostGunzip(body)
		if err != nil {
			return nil, err
		}
		return cloudCostPullItemsFromNamedBody(provider, period, currency, source, unzipped, strings.TrimSuffix(strings.TrimSuffix(name, ".gzip"), ".gz"), depth+1)
	}
	if cloudCostLooksLikeZip(body, name) {
		return cloudCostPullItemsFromZip(provider, period, currency, source, body, depth+1)
	}
	if cloudCostLooksLikeDelimited(name, '\t') {
		return cloudCostPullItemsFromDelimited(provider, period, currency, source, body, '\t')
	}
	if cloudCostLooksLikeDelimited(name, ',') {
		return cloudCostPullItemsFromDelimited(provider, period, currency, source, body, ',')
	}
	items, jsonErr := cloudCostPullItemsFromJSON(provider, period, currency, source, body)
	if jsonErr == nil {
		return items, nil
	}
	delimiter := cloudCostDetectDelimiter(body)
	if delimiter != 0 {
		if items, err := cloudCostPullItemsFromDelimited(provider, period, currency, source, body, delimiter); err == nil {
			return items, nil
		}
	}
	return nil, jsonErr
}

func cloudCostExportFileName(sourceName string) string {
	raw := strings.TrimSpace(sourceName)
	if parsed, err := url.Parse(raw); err == nil && parsed.Path != "" {
		raw = parsed.Path
	}
	raw = strings.TrimRight(raw, "/")
	if idx := strings.LastIndex(raw, "/"); idx >= 0 {
		raw = raw[idx+1:]
	}
	return raw
}

func cloudCostLooksLikeGzip(body []byte, name string) bool {
	return strings.HasSuffix(name, ".gz") || strings.HasSuffix(name, ".gzip") || (len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b)
}

func cloudCostLooksLikeZip(body []byte, name string) bool {
	return strings.HasSuffix(name, ".zip") || (len(body) >= 4 && body[0] == 'P' && body[1] == 'K' && body[2] == 0x03 && body[3] == 0x04)
}

func cloudCostGunzip(body []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("open gzip cost export failed: %w", err)
	}
	defer reader.Close()
	return cloudCostReadLimited(reader, 32*1024*1024, "gzip cost export")
}

func cloudCostPullItemsFromZip(provider, period, currency, source string, body []byte, depth int) ([]forms.ImportCloudCostRecordItem, error) {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("open zip cost export failed: %w", err)
	}
	items := make([]forms.ImportCloudCostRecordItem, 0)
	for _, file := range reader.File {
		if file.FileInfo().IsDir() || strings.HasPrefix(cloudCostExportFileName(file.Name), ".") {
			continue
		}
		name := strings.ToLower(cloudCostExportFileName(file.Name))
		if !cloudCostSupportedExportFileName(name) {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip cost export file %s failed: %w", file.Name, err)
		}
		fileBody, readErr := cloudCostReadLimited(rc, 32*1024*1024, "zip cost export file")
		rc.Close()
		if readErr != nil {
			return nil, readErr
		}
		fileItems, err := cloudCostPullItemsFromNamedBody(provider, period, currency, source, fileBody, file.Name, depth)
		if err != nil {
			return nil, fmt.Errorf("parse zip cost export file %s failed: %w", file.Name, err)
		}
		for idx := range fileItems {
			fileItems[idx].Payload["sourceArchiveFile"] = file.Name
		}
		items = append(items, fileItems...)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("zip cost export has no supported json/csv/tsv files")
	}
	return items, nil
}

func cloudCostReadLimited(reader io.Reader, limit int64, label string) ([]byte, error) {
	limited := io.LimitReader(reader, limit+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read %s failed: %w", label, err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", label, limit)
	}
	return body, nil
}

func cloudCostSupportedExportFileName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return strings.HasSuffix(name, ".json") ||
		strings.HasSuffix(name, ".csv") ||
		strings.HasSuffix(name, ".tsv") ||
		strings.HasSuffix(name, ".json.gz") ||
		strings.HasSuffix(name, ".csv.gz") ||
		strings.HasSuffix(name, ".tsv.gz") ||
		strings.HasSuffix(name, ".json.gzip") ||
		strings.HasSuffix(name, ".csv.gzip") ||
		strings.HasSuffix(name, ".tsv.gzip")
}

func cloudCostLooksLikeDelimited(name string, delimiter rune) bool {
	switch delimiter {
	case ',':
		return strings.HasSuffix(name, ".csv")
	case '\t':
		return strings.HasSuffix(name, ".tsv") || strings.HasSuffix(name, ".tab")
	default:
		return false
	}
}

func cloudCostDetectDelimiter(body []byte) rune {
	sample := string(body)
	if idx := strings.IndexAny(sample, "\r\n"); idx >= 0 {
		sample = sample[:idx]
	}
	if strings.Count(sample, "\t") >= 2 {
		return '\t'
	}
	if strings.Count(sample, ",") >= 2 {
		return ','
	}
	return 0
}

func cloudCostPullItemsFromDelimited(provider, period, currency, source string, body []byte, delimiter rune) ([]forms.ImportCloudCostRecordItem, error) {
	reader := csv.NewReader(bytes.NewReader(body))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	reader.Comma = delimiter
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("decode cost export delimited file failed: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("cost export delimited file requires header and at least one data row")
	}
	headers := rows[0]
	records := make([]models.ResAttrs, 0, len(rows)-1)
	for rowIdx, row := range rows[1:] {
		if cloudCostDelimitedRowEmpty(row) {
			continue
		}
		payload := models.ResAttrs{}
		for colIdx, header := range headers {
			key := strings.TrimSpace(header)
			if key == "" || colIdx >= len(row) {
				continue
			}
			payload[key] = strings.TrimSpace(row[colIdx])
		}
		payload["pullCsvRow"] = rowIdx + 2
		records = append(records, payload)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("cost export delimited file has no data rows")
	}
	return cloudCostPullItemsFromAttrs(provider, period, currency, source, records), nil
}

func cloudCostDelimitedRowEmpty(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func cloudCostPullItemsFromJSONArray(provider, period, currency, source string, body []byte) []forms.ImportCloudCostRecordItem {
	records := make([]models.ResAttrs, 0)
	if err := json.Unmarshal(body, &records); err != nil {
		return nil
	}
	return cloudCostPullItemsFromAttrs(provider, period, currency, source, records)
}

func cloudCostPullItemsFromAttrs(provider, period, currency, source string, records []models.ResAttrs) []forms.ImportCloudCostRecordItem {
	items := make([]forms.ImportCloudCostRecordItem, 0, len(records))
	for idx, payload := range records {
		payload["pullIndex"] = idx
		items = append(items, forms.ImportCloudCostRecordItem{
			Provider: firstNonEmpty(provider, attrString(payload, "provider"), attrString(payload, "Provider")),
			Period:   firstNonEmpty(cloudCostPeriodFromPayload(payload), period),
			Currency: firstNonEmpty(cloudCostCurrencyFromPayload(payload), currency),
			Source:   source,
			Payload:  payload,
		})
	}
	return items
}

func cloudCostPullItemsFromAzureQuery(resp models.ResAttrs, period, currency string) []forms.ImportCloudCostRecordItem {
	properties := modelResAttrs(resp["properties"])
	if properties == nil {
		properties = resp
	}
	columns := resAttrsSlice(properties["columns"])
	rows, _ := properties["rows"].([]interface{})
	if len(columns) == 0 || len(rows) == 0 {
		return nil
	}
	names := make([]string, 0, len(columns))
	for _, column := range columns {
		names = append(names, firstNonEmpty(attrString(column, "name"), attrString(column, "Name")))
	}
	items := make([]forms.ImportCloudCostRecordItem, 0, len(rows))
	for rowIndex, row := range rows {
		values, ok := row.([]interface{})
		if !ok {
			continue
		}
		payload := models.ResAttrs{"pullIndex": rowIndex}
		for idx, name := range names {
			if name != "" && idx < len(values) {
				payload[name] = values[idx]
			}
		}
		resourceId := firstNonEmpty(attrString(payload, "ResourceId"), attrString(payload, "ResourceID"), attrString(payload, "resourceId"))
		service := firstNonEmpty(attrString(payload, "ServiceName"), attrString(payload, "MeterCategory"), attrString(payload, "service"))
		region := firstNonEmpty(attrString(payload, "ResourceLocation"), attrString(payload, "resourceLocation"))
		if amount, ok := cloudCostFloat(firstNonEmptyInterface(payload["PreTaxCost"], payload["Cost"], payload["totalCost"])); ok {
			payload["amount"] = amount
		}
		items = append(items, forms.ImportCloudCostRecordItem{
			Provider:     "azure",
			Region:       region,
			Service:      service,
			ResourceId:   resourceId,
			ResourceName: cloudCostResourceNameFromId(resourceId),
			Amount:       cloudCostImportAmount(forms.ImportCloudCostRecordItem{}, payload),
			Currency:     firstNonEmpty(attrString(payload, "Currency"), currency),
			Period:       firstNonEmpty(cloudCostPeriodFromPayload(payload), period),
			Source:       models.CloudCostSourceAzureCostManagement,
			Payload:      payload,
		})
	}
	return items
}

func firstNonEmptyInterface(values ...interface{}) interface{} {
	for _, value := range values {
		if value != nil && fmt.Sprintf("%v", value) != "" {
			return value
		}
	}
	return nil
}

func lookupCloudCostAsset(index *cloudCostAssetIndex, resourceId string) (models.CmdbAsset, bool) {
	if index == nil || strings.TrimSpace(resourceId) == "" {
		return models.CmdbAsset{}, false
	}
	resourceId = strings.TrimSpace(resourceId)
	if asset, ok := index.ByNativeId[resourceId]; ok {
		return asset, true
	}
	if asset, ok := index.ByResourceRes[resourceId]; ok {
		return asset, true
	}
	lower := strings.ToLower(resourceId)
	if asset, ok := index.ByNativeIdLower[lower]; ok {
		return asset, true
	}
	if asset, ok := index.ByResourceResLower[lower]; ok {
		return asset, true
	}
	return models.CmdbAsset{}, false
}

func sortedCloudCostKeys(input map[string]bool) []string {
	out := make([]string, 0, len(input))
	for key := range input {
		if strings.TrimSpace(key) != "" {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func attrNestedString(attrs models.ResAttrs, key string, nested string) string {
	child := modelResAttrs(attrs[key])
	if child == nil {
		return ""
	}
	return attrString(child, nested)
}

func attrPathValue(attrs models.ResAttrs, path ...string) interface{} {
	if len(path) == 0 || attrs == nil {
		return nil
	}
	var current interface{} = attrs
	for _, key := range path {
		currentAttrs := modelResAttrs(current)
		if currentAttrs == nil {
			return nil
		}
		current = currentAttrs[key]
	}
	return current
}

func attrPathString(attrs models.ResAttrs, path ...string) string {
	value := attrPathValue(attrs, path...)
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func cloudCostResourceNameFromId(resourceId string) string {
	resourceId = strings.Trim(strings.TrimSpace(resourceId), "/")
	if resourceId == "" {
		return ""
	}
	parts := strings.Split(resourceId, "/")
	return parts[len(parts)-1]
}

func cloudCostGroups(c *ctx.ServiceContext, period string, currency string, column string) ([]resps.CloudCostGroupResp, e.Error) {
	rows := make([]resps.CloudCostGroupResp, 0)
	if err := cloudCostBaseQuery(c).
		Select(fmt.Sprintf("%s as `key`, %s as label, coalesce(sum(amount), 0) as amount, currency, count(*) as count", column, column)).
		Where("period = ? and currency = ?", period, currency).
		Group(column + ", currency").
		Order("amount desc").
		Scan(&rows); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return rows, nil
}

func cloudCostAssetIndexForOrg(c *ctx.ServiceContext) (*cloudCostAssetIndex, e.Error) {
	assets := make([]models.CmdbAsset, 0)
	if err := c.DB().Model(&models.CmdbAsset{}).Where("org_id = ?", c.OrgId).Find(&assets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	resources := make([]models.Resource, 0)
	if err := c.DB().Model(&models.Resource{}).Where("org_id = ?", c.OrgId).Find(&resources); err != nil {
		return nil, e.New(e.DBError, err)
	}
	assetByIacResourceId := make(map[models.Id]models.CmdbAsset, len(assets))
	index := &cloudCostAssetIndex{
		ByNativeId:         make(map[string]models.CmdbAsset),
		ByNativeIdLower:    make(map[string]models.CmdbAsset),
		ByResourceRes:      make(map[string]models.CmdbAsset),
		ByResourceResLower: make(map[string]models.CmdbAsset),
	}
	for _, asset := range assets {
		if asset.NativeId != "" {
			index.ByNativeId[asset.NativeId] = asset
			index.ByNativeIdLower[strings.ToLower(asset.NativeId)] = asset
		}
		if asset.Id != "" {
			assetId := asset.Id.String()
			index.ByNativeId[assetId] = asset
			index.ByNativeIdLower[strings.ToLower(assetId)] = asset
		}
		if asset.IacResourceId != "" {
			assetByIacResourceId[asset.IacResourceId] = asset
		}
	}
	for _, resource := range resources {
		if asset, ok := assetByIacResourceId[resource.Id]; ok && resource.ResId != "" {
			resId := resource.ResId.String()
			index.ByResourceRes[resId] = asset
			index.ByResourceResLower[strings.ToLower(resId)] = asset
		}
	}
	return index, nil
}

func refreshCloudCostRecordsFromBills(c *ctx.ServiceContext, index *cloudCostAssetIndex) e.Error {
	bills := make([]models.Bill, 0)
	if err := c.DB().Model(&models.Bill{}).Where("org_id = ?", c.OrgId).Find(&bills); err != nil {
		return e.New(e.DBError, err)
	}
	for _, bill := range bills {
		asset, matched := index.ByNativeId[bill.InstanceId]
		if !matched {
			asset, matched = index.ByResourceRes[bill.InstanceId]
		}
		record := models.CloudCostRecord{
			OrgId:        bill.OrgId,
			ProjectId:    bill.ProjectId,
			EnvId:        bill.EnvId,
			Provider:     normalizeProvider(bill.Provider),
			Region:       bill.Region,
			Service:      bill.ProductCode,
			ResourceId:   bill.InstanceId,
			ResourceName: bill.InstanceId,
			Amount:       float64(bill.PretaxAmount),
			Currency:     firstNonEmpty(bill.Currency, "CNY"),
			Period:       bill.Cycle,
			MatchedAsset: matched,
			Source:       models.CloudCostSourceBill,
			SourceId:     fmt.Sprintf("%d", bill.Id),
			Payload: models.ResAttrs{
				"billId":         bill.Id,
				"vgId":           bill.VgId.String(),
				"instanceConfig": bill.InstanceConfig,
			},
		}
		if matched {
			mergeCloudCostAsset(&record, asset)
		}
		if record.Period == "" {
			record.Period = time.Now().Format("2006-01")
		}
		if err := upsertCloudCostRecord(c, record); err != nil {
			return err
		}
	}
	return nil
}

func refreshCloudCostRecordsFromAssets(c *ctx.ServiceContext) e.Error {
	assets := make([]models.CmdbAsset, 0)
	if err := c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and cost > 0", c.OrgId).
		Find(&assets); err != nil {
		return e.New(e.DBError, err)
	}
	period := time.Now().Format("2006-01")
	for _, asset := range assets {
		record := models.CloudCostRecord{
			OrgId:        asset.OrgId,
			ProjectId:    asset.ProjectId,
			EnvId:        asset.EnvId,
			Provider:     asset.Provider,
			AccountId:    asset.AccountId,
			Region:       asset.Region,
			Service:      firstNonEmpty(asset.AssetType, asset.NativeType),
			ResourceType: asset.AssetType,
			ResourceId:   firstNonEmpty(asset.NativeId, asset.IacAddress, asset.Id.String()),
			ResourceName: firstNonEmpty(asset.Name, asset.NativeId, asset.Id.String()),
			Amount:       asset.Cost,
			Currency:     "CNY",
			Period:       period,
			MatchedAsset: true,
			Source:       models.CloudCostSourceCmdbAsset,
			SourceId:     asset.Id.String(),
			Payload: models.ResAttrs{
				"assetCost": asset.Cost,
				"source":    "cmdb_asset.cost",
			},
		}
		mergeCloudCostAsset(&record, asset)
		if err := upsertCloudCostRecord(c, record); err != nil {
			return err
		}
	}
	return nil
}

func mergeCloudCostAsset(record *models.CloudCostRecord, asset models.CmdbAsset) {
	record.AssetId = asset.Id
	record.CloudAccountId = asset.CloudAccountId
	record.ProjectId = firstNonEmptyId(record.ProjectId, asset.ProjectId)
	record.EnvId = firstNonEmptyId(record.EnvId, asset.EnvId)
	record.Provider = firstNonEmpty(record.Provider, asset.Provider)
	record.AccountId = firstNonEmpty(record.AccountId, asset.AccountId)
	record.Region = firstNonEmpty(record.Region, asset.Region)
	record.ResourceType = firstNonEmpty(record.ResourceType, asset.AssetType, asset.NativeType)
	record.ResourceId = firstNonEmpty(record.ResourceId, asset.NativeId, asset.IacAddress, asset.Id.String())
	record.ResourceName = firstNonEmpty(asset.Name, record.ResourceName, record.ResourceId)
	record.CostCenter = asset.CostCenter
	record.Owner = asset.Owner
	record.Application = asset.Application
	record.BusinessLine = asset.BusinessLine
}

func upsertCloudCostRecord(c *ctx.ServiceContext, record models.CloudCostRecord) e.Error {
	record.Fingerprint = cloudCostFingerprint(c.OrgId, record)
	existing := models.CloudCostRecord{}
	err := c.DB().Model(&models.CloudCostRecord{}).
		Where("org_id = ? and fingerprint = ?", c.OrgId, record.Fingerprint).
		First(&existing)
	if err != nil && !e.IsRecordNotFound(err) {
		return e.New(e.DBError, err)
	}
	if e.IsRecordNotFound(err) {
		record.Id = models.NewId("ccr")
		if err := models.Create(c.DB(), &record); err != nil {
			return e.New(e.DBError, err)
		}
		return nil
	}
	if _, err := c.DB().Model(&models.CloudCostRecord{}).
		Where("id = ? and org_id = ?", existing.Id, c.OrgId).
		UpdateAttrs(models.Attrs{
			"project_id":       record.ProjectId,
			"env_id":           record.EnvId,
			"asset_id":         record.AssetId,
			"cloud_account_id": record.CloudAccountId,
			"provider":         record.Provider,
			"account_id":       record.AccountId,
			"region":           record.Region,
			"service":          record.Service,
			"resource_type":    record.ResourceType,
			"resource_id":      record.ResourceId,
			"resource_name":    record.ResourceName,
			"amount":           record.Amount,
			"currency":         record.Currency,
			"period":           record.Period,
			"cost_center":      record.CostCenter,
			"owner":            record.Owner,
			"application":      record.Application,
			"business_line":    record.BusinessLine,
			"matched_asset":    record.MatchedAsset,
			"source":           record.Source,
			"source_id":        record.SourceId,
			"payload":          record.Payload,
		}); err != nil {
		return e.New(e.DBError, err)
	}
	return nil
}

func cloudCostFingerprint(orgId models.Id, record models.CloudCostRecord) string {
	seed := strings.Join([]string{
		orgId.String(),
		record.Source,
		record.SourceId,
		record.Period,
		record.Currency,
		record.ResourceId,
	}, "\x00")
	sum := sha1.Sum([]byte(seed))
	return hex.EncodeToString(sum[:])
}

func cloudCostRecordResp(c *ctx.ServiceContext, record models.CloudCostRecord) resps.CloudCostRecordResp {
	assetName := record.ResourceName
	if assetName == "" && record.AssetId != "" {
		assetName = lookupName(c, &models.CmdbAsset{}, record.AssetId)
	}
	return resps.CloudCostRecordResp{
		CloudCostRecord: record,
		ProjectName:     lookupName(c, &models.Project{}, record.ProjectId),
		EnvName:         lookupName(c, &models.Env{}, record.EnvId),
		AssetName:       assetName,
	}
}

func firstNonEmptyId(values ...models.Id) models.Id {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func attrBool(attrs models.ResAttrs, key string) bool {
	if attrs == nil {
		return false
	}
	switch value := attrs[key].(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(strings.TrimSpace(value), "true") || strings.TrimSpace(value) == "1"
	case float64:
		return value != 0
	case int:
		return value != 0
	case int64:
		return value != 0
	case json.Number:
		intValue, err := value.Int64()
		return err == nil && intValue != 0
	default:
		return fmt.Sprintf("%v", value) == "true"
	}
}
