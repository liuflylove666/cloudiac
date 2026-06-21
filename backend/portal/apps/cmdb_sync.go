// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"runtime/debug"
	"sort"
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
	summary, err := cmdbSyncTaskSummary(c, form)
	if err != nil {
		return nil, err
	}
	query := cmdbSyncTaskSearchQuery(c, form)
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

	return &resps.CmdbSyncTaskPageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     tasks,
		Summary:  summary,
	}, nil
}

func cmdbSyncTaskSearchQuery(c *ctx.ServiceContext, form *forms.SearchCmdbSyncTaskForm) *db.Session {
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
	if form.SyncPolicyId != "" {
		query = query.Where("sync_policy_id = ?", form.SyncPolicyId)
	}
	if form.SyncPolicyScheduleKey != "" {
		query = query.Where("JSON_UNQUOTE(JSON_EXTRACT(stats, '$.syncPolicyScheduleKey')) = ?", form.SyncPolicyScheduleKey)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	return query
}

func cmdbSyncTaskSummary(c *ctx.ServiceContext, form *forms.SearchCmdbSyncTaskForm) (resps.CmdbSyncTaskSummaryResp, e.Error) {
	summary := resps.CmdbSyncTaskSummaryResp{}
	var dbErr error

	trendRange := cmdbSyncTaskTrendRangeFor(form)
	summary.TrendDays = trendRange.Days
	summary.TrendStartDate = trendRange.StartDate
	summary.TrendEndDate = trendRange.EndDate
	summary.TrendCustomRange = trendRange.Custom
	summary.FailureThreshold = cmdbSyncTaskFailureThreshold(form)
	if summary.TotalCount, dbErr = cmdbSyncTaskSearchQuery(c, form).Count(); dbErr != nil {
		return summary, e.New(e.DBError, dbErr)
	}
	if summary.CompleteCount, dbErr = cmdbSyncTaskCountByStatus(c, form, models.CmdbSyncTaskComplete); dbErr != nil {
		return summary, e.New(e.DBError, dbErr)
	}
	if summary.FailedCount, dbErr = cmdbSyncTaskCountByStatus(c, form, models.CmdbSyncTaskFailed); dbErr != nil {
		return summary, e.New(e.DBError, dbErr)
	}
	if summary.RejectedCount, dbErr = cmdbSyncTaskCountByStatus(c, form, models.CmdbSyncTaskRejected); dbErr != nil {
		return summary, e.New(e.DBError, dbErr)
	}
	if summary.ApprovingCount, dbErr = cmdbSyncTaskCountByStatus(c, form, models.CmdbSyncTaskApproving); dbErr != nil {
		return summary, e.New(e.DBError, dbErr)
	}
	if summary.RunningCount, dbErr = cmdbSyncTaskCountByStatus(c, form, models.CmdbSyncTaskRunning); dbErr != nil {
		return summary, e.New(e.DBError, dbErr)
	}
	if summary.PendingCount, dbErr = cmdbSyncTaskCountByStatus(c, form, models.CmdbSyncTaskPending); dbErr != nil {
		return summary, e.New(e.DBError, dbErr)
	}
	if summary.TotalCount > 0 {
		summary.SuccessRate = cmdbSyncTaskRate(summary.CompleteCount, summary.TotalCount)
		summary.FailureRate = cmdbSyncTaskRate(summary.FailedCount, summary.TotalCount)
	}
	cmdbSyncTaskApplyFailureAlert(&summary)
	if summary.LastSuccessAt, dbErr = cmdbSyncTaskLastStatusAt(c, form, models.CmdbSyncTaskComplete); dbErr != nil {
		return summary, e.New(e.DBError, dbErr)
	}
	if summary.LastFailureAt, dbErr = cmdbSyncTaskLastStatusAt(c, form, models.CmdbSyncTaskFailed); dbErr != nil {
		return summary, e.New(e.DBError, dbErr)
	}
	if summary.Trend, dbErr = cmdbSyncTaskTrend(c, form, trendRange); dbErr != nil {
		return summary, e.New(e.DBError, dbErr)
	}
	if summary.Regions, summary.AssetTypes, dbErr = cmdbSyncTaskBreakdowns(c, form); dbErr != nil {
		return summary, e.New(e.DBError, dbErr)
	}
	return summary, nil
}

func cmdbSyncTaskCountByStatus(c *ctx.ServiceContext, form *forms.SearchCmdbSyncTaskForm, status string) (int64, error) {
	return cmdbSyncTaskSearchQuery(c, form).Where("status = ?", status).Count()
}

func cmdbSyncTaskLastStatusAt(c *ctx.ServiceContext, form *forms.SearchCmdbSyncTaskForm, status string) (models.Time, error) {
	task := models.CmdbSyncTask{}
	if err := cmdbSyncTaskSearchQuery(c, form).
		Where("status = ?", status).
		Order("ended_at desc").
		Order("created_at desc").
		First(&task); err != nil {
		if e.IsRecordNotFound(err) {
			return models.Time{}, nil
		}
		return models.Time{}, err
	}
	if time.Time(task.EndedAt).IsZero() {
		return task.CreatedAt, nil
	}
	return task.EndedAt, nil
}

func cmdbSyncTaskTrendDays(form *forms.SearchCmdbSyncTaskForm) int {
	if form == nil {
		return 7
	}
	switch form.TrendDays {
	case 14, 30:
		return form.TrendDays
	default:
		return 7
	}
}

type cmdbSyncTaskTrendRange struct {
	Start     time.Time
	End       time.Time
	Days      int
	StartDate string
	EndDate   string
	Custom    bool
}

func cmdbSyncTaskTrendRangeFor(form *forms.SearchCmdbSyncTaskForm) cmdbSyncTaskTrendRange {
	if form != nil {
		start, startOk := cmdbSyncTaskParseTrendDate(form.TrendStartDate)
		end, endOk := cmdbSyncTaskParseTrendDate(form.TrendEndDate)
		if startOk && endOk && !end.Before(start) {
			days := int(end.Sub(start).Hours()/24) + 1
			if days > 0 && days <= 90 {
				return cmdbSyncTaskTrendRange{
					Start:     start,
					End:       end,
					Days:      days,
					StartDate: start.Format("2006-01-02"),
					EndDate:   end.Format("2006-01-02"),
					Custom:    true,
				}
			}
		}
	}

	days := cmdbSyncTaskTrendDays(form)
	end := cmdbSyncTaskDayStart(time.Now())
	start := end.AddDate(0, 0, -days+1)
	return cmdbSyncTaskTrendRange{
		Start:     start,
		End:       end,
		Days:      days,
		StartDate: start.Format("2006-01-02"),
		EndDate:   end.Format("2006-01-02"),
	}
}

func cmdbSyncTaskParseTrendDate(value string) (time.Time, bool) {
	parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return cmdbSyncTaskDayStart(parsed), true
}

func cmdbSyncTaskDayStart(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func cmdbSyncTaskFailureThreshold(form *forms.SearchCmdbSyncTaskForm) float64 {
	if form == nil {
		return 50
	}
	threshold := form.FailureThreshold
	if threshold <= 0 || threshold > 100 {
		return 50
	}
	return math.Round(threshold*10) / 10
}

func cmdbSyncTaskApplyFailureAlert(summary *resps.CmdbSyncTaskSummaryResp) {
	if summary == nil {
		return
	}
	threshold := summary.FailureThreshold
	if threshold <= 0 {
		threshold = 50
		summary.FailureThreshold = threshold
	}
	if summary.TotalCount <= 0 {
		summary.FailureAlertLevel = models.CloudEventLevelInfo
		summary.FailureAlertMessage = fmt.Sprintf("暂无任务历史，未触发 %.1f%% 失败率阈值", threshold)
		return
	}
	if summary.FailedCount > 0 && summary.FailureRate >= threshold {
		summary.FailureThresholdExceeded = true
		summary.FailureAlertLevel = models.CloudEventLevelWarning
		if summary.FailureRate >= 80 {
			summary.FailureAlertLevel = models.CloudEventLevelError
		}
		summary.FailureAlertMessage = fmt.Sprintf("当前子周期失败率 %.1f%%，已达到 %.1f%% 告警阈值（失败 %d / 总数 %d）",
			summary.FailureRate, threshold, summary.FailedCount, summary.TotalCount)
		return
	}
	summary.FailureAlertLevel = models.CloudEventLevelInfo
	summary.FailureAlertMessage = fmt.Sprintf("当前子周期失败率 %.1f%%，未达到 %.1f%% 告警阈值（失败 %d / 总数 %d）",
		summary.FailureRate, threshold, summary.FailedCount, summary.TotalCount)
}

func cmdbSyncTaskTrend(c *ctx.ServiceContext, form *forms.SearchCmdbSyncTaskForm, trendRange cmdbSyncTaskTrendRange) ([]resps.CmdbSyncTaskTrendPoint, error) {
	type trendRow struct {
		Date   string `gorm:"column:date"`
		Status string `gorm:"column:status"`
		Count  int64  `gorm:"column:count"`
	}

	trend := make([]resps.CmdbSyncTaskTrendPoint, 0, trendRange.Days)
	indexByDate := make(map[string]int, trendRange.Days)
	for i := 0; i < trendRange.Days; i++ {
		date := trendRange.Start.AddDate(0, 0, i).Format("2006-01-02")
		indexByDate[date] = len(trend)
		trend = append(trend, resps.CmdbSyncTaskTrendPoint{Date: date})
	}

	rows := make([]trendRow, 0)
	if err := cmdbSyncTaskSearchQuery(c, form).
		Where("created_at >= ? and created_at < ?", trendRange.Start, trendRange.End.AddDate(0, 0, 1)).
		Select("DATE_FORMAT(created_at, '%Y-%m-%d') as date, status, COUNT(*) as count").
		Group("DATE_FORMAT(created_at, '%Y-%m-%d'), status").
		Order("date asc").
		Scan(&rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		idx, ok := indexByDate[row.Date]
		if !ok {
			continue
		}
		trend[idx].TotalCount += row.Count
		switch row.Status {
		case models.CmdbSyncTaskComplete:
			trend[idx].CompleteCount += row.Count
		case models.CmdbSyncTaskFailed:
			trend[idx].FailedCount += row.Count
		}
	}
	return trend, nil
}

func cmdbSyncTaskRate(count int64, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round(float64(count)*1000/float64(total)) / 10
}

type cmdbSyncTaskBreakdownAccumulator struct {
	Key           string
	Name          string
	TaskCount     int64
	CompleteCount int64
	FailedCount   int64
	RunningCount  int64
	PendingCount  int64
	Collected     int64
}

func cmdbSyncTaskBreakdowns(c *ctx.ServiceContext, form *forms.SearchCmdbSyncTaskForm) ([]resps.CmdbSyncTaskBreakdown, []resps.CmdbSyncTaskBreakdown, error) {
	tasks := make([]models.CmdbSyncTask, 0)
	if err := cmdbSyncTaskSearchQuery(c, form).
		Select("regions, asset_types, status, stats").
		Scan(&tasks); err != nil {
		return nil, nil, err
	}
	regionAcc := make(map[string]*cmdbSyncTaskBreakdownAccumulator)
	assetTypeAcc := make(map[string]*cmdbSyncTaskBreakdownAccumulator)
	for _, task := range tasks {
		cmdbSyncTaskAddMetricBreakdown(regionAcc, task.Status, task.Stats, "regionMetrics", "region", []string(task.Regions))
		cmdbSyncTaskAddMetricBreakdown(assetTypeAcc, task.Status, task.Stats, "assetTypeMetrics", "assetType", []string(task.AssetTypes))
	}
	return cmdbSyncTaskBreakdownList(regionAcc), cmdbSyncTaskBreakdownList(assetTypeAcc), nil
}

func cmdbSyncTaskAddMetricBreakdown(acc map[string]*cmdbSyncTaskBreakdownAccumulator, taskStatus string, stats models.ResAttrs, metricKey string, labelKey string, fallbackLabels []string) {
	metrics := cmdbSyncTaskMetricAttrsList(stats[metricKey])
	if len(metrics) > 0 {
		for _, metric := range metrics {
			key := cmdbSyncTaskMetricString(metric[labelKey])
			if key == "" {
				key = "unknown"
			}
			cmdbSyncTaskAddBreakdown(acc, key, key, taskStatus, cmdbSyncTaskMetricInt64(metric["collected"]))
		}
		return
	}

	labels := normalizeStringList(fallbackLabels)
	if len(labels) == 0 {
		labels = []string{"unknown"}
	}
	collected := int64(0)
	if len(labels) == 1 {
		collected = cmdbSyncTaskMetricInt64(stats["collected"])
	}
	for _, label := range labels {
		cmdbSyncTaskAddBreakdown(acc, label, label, taskStatus, collected)
	}
}

func cmdbSyncTaskMetricAttrsList(value interface{}) []models.ResAttrs {
	switch typed := value.(type) {
	case []models.ResAttrs:
		return typed
	case []map[string]interface{}:
		result := make([]models.ResAttrs, 0, len(typed))
		for _, item := range typed {
			result = append(result, models.ResAttrs(item))
		}
		return result
	case []interface{}:
		result := make([]models.ResAttrs, 0, len(typed))
		for _, item := range typed {
			switch attrs := item.(type) {
			case models.ResAttrs:
				result = append(result, attrs)
			case map[string]interface{}:
				result = append(result, models.ResAttrs(attrs))
			}
		}
		return result
	default:
		return nil
	}
}

func cmdbSyncTaskMetricString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return ""
	}
}

func cmdbSyncTaskMetricInt64(value interface{}) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int8:
		return int64(typed)
	case int16:
		return int64(typed)
	case int32:
		return int64(typed)
	case int64:
		return typed
	case uint:
		return int64(typed)
	case uint8:
		return int64(typed)
	case uint16:
		return int64(typed)
	case uint32:
		return int64(typed)
	case uint64:
		return int64(typed)
	case float32:
		return int64(typed)
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

func cmdbSyncTaskAddBreakdown(acc map[string]*cmdbSyncTaskBreakdownAccumulator, key string, name string, taskStatus string, collected int64) {
	key = strings.TrimSpace(key)
	if key == "" {
		key = "unknown"
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = key
	}
	item, ok := acc[key]
	if !ok {
		item = &cmdbSyncTaskBreakdownAccumulator{Key: key, Name: name}
		acc[key] = item
	}
	item.TaskCount++
	item.Collected += collected
	switch taskStatus {
	case models.CmdbSyncTaskComplete:
		item.CompleteCount++
	case models.CmdbSyncTaskFailed:
		item.FailedCount++
	case models.CmdbSyncTaskRunning:
		item.RunningCount++
	case models.CmdbSyncTaskPending:
		item.PendingCount++
	}
}

func cmdbSyncTaskBreakdownList(acc map[string]*cmdbSyncTaskBreakdownAccumulator) []resps.CmdbSyncTaskBreakdown {
	list := make([]resps.CmdbSyncTaskBreakdown, 0, len(acc))
	for _, item := range acc {
		list = append(list, resps.CmdbSyncTaskBreakdown{
			Key:           item.Key,
			Name:          item.Name,
			TaskCount:     item.TaskCount,
			CompleteCount: item.CompleteCount,
			FailedCount:   item.FailedCount,
			RunningCount:  item.RunningCount,
			PendingCount:  item.PendingCount,
			Collected:     item.Collected,
			FailureRate:   cmdbSyncTaskRate(item.FailedCount, item.TaskCount),
		})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].TaskCount != list[j].TaskCount {
			return list[i].TaskCount > list[j].TaskCount
		}
		if list[i].FailedCount != list[j].FailedCount {
			return list[i].FailedCount > list[j].FailedCount
		}
		return list[i].Key < list[j].Key
	})
	return list
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

func CmdbSyncTaskRerunGroupDetail(c *ctx.ServiceContext, form *forms.CmdbSyncTaskRerunGroupParam) (*resps.CmdbSyncTaskRerunGroupResp, e.Error) {
	groupId := models.Id(strings.TrimSpace(form.GroupId.String()))
	if groupId == "" {
		return nil, e.New(e.BadParam, fmt.Errorf("任务组不能为空"), http.StatusBadRequest)
	}

	tasks := make([]models.CmdbSyncTask, 0)
	if err := c.DB().Model(&models.CmdbSyncTask{}).
		Where("org_id = ? and JSON_UNQUOTE(JSON_EXTRACT(stats, '$.rerunGroupId')) = ?", c.OrgId, groupId.String()).
		Order("created_at asc").
		Find(&tasks); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if len(tasks) == 0 {
		return nil, e.New(e.ObjectNotExistsOrNoPerm, fmt.Errorf("任务组不存在或不属于当前组织"), http.StatusNotFound)
	}

	sourceIds := make([]models.Id, 0, len(tasks))
	for _, task := range tasks {
		if sourceTaskId := models.Id(attrString(task.Stats, "rerunFromTaskId")); sourceTaskId != "" {
			sourceIds = append(sourceIds, sourceTaskId)
		}
	}
	sourceIds = cmdbSyncTaskUniqueIds(sourceIds)
	sourceMap := make(map[models.Id]models.CmdbSyncTask, len(sourceIds))
	if len(sourceIds) > 0 {
		sourceTasks := make([]models.CmdbSyncTask, 0, len(sourceIds))
		if err := c.DB().Model(&models.CmdbSyncTask{}).
			Where("org_id = ? and id in (?)", c.OrgId, sourceIds).
			Find(&sourceTasks); err != nil {
			return nil, e.New(e.DBError, err)
		}
		for _, sourceTask := range sourceTasks {
			sourceMap[sourceTask.Id] = sourceTask
		}
	}

	resp := &resps.CmdbSyncTaskRerunGroupResp{
		GroupId:       groupId,
		Total:         len(tasks),
		Approval:      attrResAttrs(tasks[0].Stats, "approval"),
		TaskIds:       make([]models.Id, 0, len(tasks)),
		SourceTaskIds: sourceIds,
		Tasks:         make([]resps.CmdbSyncTaskRerunGroupTask, 0, len(tasks)),
	}
	for _, task := range tasks {
		sourceTaskId := models.Id(attrString(task.Stats, "rerunFromTaskId"))
		sourceTaskStatus := ""
		if sourceTask, ok := sourceMap[sourceTaskId]; ok {
			sourceTaskStatus = sourceTask.Status
		}
		if resp.Reason == "" {
			resp.Reason = attrString(task.Stats, "reason")
		}
		if resp.Mode == "" {
			resp.Mode = attrString(task.Stats, "rerunMode")
		}
		resp.CreatedAt = cmdbSyncTaskMinTime(resp.CreatedAt, task.CreatedAt)
		resp.StartedAt = cmdbSyncTaskMinTime(resp.StartedAt, task.StartedAt)
		resp.EndedAt = cmdbSyncTaskMaxTime(resp.EndedAt, task.EndedAt)
		resp.TaskIds = append(resp.TaskIds, task.Id)
		resp.Tasks = append(resp.Tasks, resps.CmdbSyncTaskRerunGroupTask{
			CmdbSyncTask:     task,
			SourceTaskId:     sourceTaskId,
			SourceTaskStatus: sourceTaskStatus,
		})

		switch task.Status {
		case models.CmdbSyncTaskApproving:
			resp.ApprovingCount++
		case models.CmdbSyncTaskPending:
			resp.PendingCount++
		case models.CmdbSyncTaskRunning:
			resp.RunningCount++
		case models.CmdbSyncTaskComplete:
			resp.CompleteCount++
		case models.CmdbSyncTaskFailed:
			resp.FailedCount++
		case models.CmdbSyncTaskRejected:
			resp.RejectedCount++
		}
	}

	return resp, nil
}

func cmdbSyncTaskMinTime(current, candidate models.Time) models.Time {
	if time.Time(candidate).IsZero() {
		return current
	}
	if time.Time(current).IsZero() || time.Time(candidate).Before(time.Time(current)) {
		return candidate
	}
	return current
}

func cmdbSyncTaskMaxTime(current, candidate models.Time) models.Time {
	if time.Time(candidate).IsZero() {
		return current
	}
	if time.Time(current).IsZero() || time.Time(candidate).After(time.Time(current)) {
		return candidate
	}
	return current
}

type cmdbSyncTaskRerunMeta struct {
	GroupId    models.Id
	FromTaskId models.Id
	Mode       string
}

type cmdbSyncTaskLaunch struct {
	TaskId                 models.Id
	Account                *cmdbCloudAccount
	Regions                []string
	AssetTypes             []string
	SyncPolicyId           models.Id
	SyncPolicyScheduleKey  string
	SyncPolicyScheduleName string
	Reason                 string
	RerunMeta              *cmdbSyncTaskRerunMeta
}

type cmdbSyncTaskBatchPrepared struct {
	SourceTask models.CmdbSyncTask
	Task       *models.CmdbSyncTask
	Launch     cmdbSyncTaskLaunch
}

func StartCmdbSyncTask(c *ctx.ServiceContext, form *forms.CreateCmdbSyncTaskForm) (*resps.CmdbSyncTaskResp, e.Error) {
	return startCmdbSyncTask(c, form, nil)
}

func BatchRerunFailedCmdbSyncTasks(c *ctx.ServiceContext, form *forms.BatchRerunFailedCmdbSyncTasksForm) (*resps.CmdbSyncTaskBatchRerunResp, e.Error) {
	taskIds := cmdbSyncTaskUniqueIds(form.TaskIds)
	resp := &resps.CmdbSyncTaskBatchRerunResp{
		Total:  len(taskIds),
		Items:  make([]resps.CmdbSyncTaskBatchRerunItem, 0, len(taskIds)),
		Errors: make([]resps.CmdbSyncTaskBatchRerunError, 0),
	}
	reason := strings.TrimSpace(form.Reason)
	if reason == "" {
		resp.Errors = append(resp.Errors, resps.CmdbSyncTaskBatchRerunError{Message: "重跑原因不能为空"})
		return resp, nil
	}
	if len(taskIds) == 0 {
		resp.Errors = append(resp.Errors, resps.CmdbSyncTaskBatchRerunError{Message: "请选择需要重跑的失败任务"})
		return resp, nil
	}

	tasks := make([]models.CmdbSyncTask, 0, len(taskIds))
	if err := c.DB().Model(&models.CmdbSyncTask{}).
		Where("org_id = ? and id in (?)", c.OrgId, taskIds).
		Find(&tasks); err != nil {
		return nil, e.New(e.DBError, err)
	}
	taskMap := make(map[models.Id]models.CmdbSyncTask, len(tasks))
	for _, task := range tasks {
		taskMap[task.Id] = task
	}
	for _, taskId := range taskIds {
		task, ok := taskMap[taskId]
		if !ok {
			resp.Errors = append(resp.Errors, resps.CmdbSyncTaskBatchRerunError{
				TaskId:  taskId,
				Message: "任务不存在或不属于当前组织",
			})
			continue
		}
		if task.Status != models.CmdbSyncTaskFailed {
			resp.Errors = append(resp.Errors, resps.CmdbSyncTaskBatchRerunError{
				TaskId:  taskId,
				Message: fmt.Sprintf("当前状态为 %s，只有失败任务可以批量重跑", task.Status),
			})
		}
	}
	if len(resp.Errors) > 0 {
		return resp, nil
	}

	groupId := models.NewId("csr")
	resp.GroupId = groupId
	requiresApproval := form.RequiresApproval
	overrideRegions := normalizeStringList(form.Regions)
	overrideAssetTypes := normalizeStringList(form.AssetTypes)
	prepared := make([]cmdbSyncTaskBatchPrepared, 0, len(taskIds))
	for _, taskId := range taskIds {
		sourceTask := taskMap[taskId]
		sourceRegions := normalizeStringList([]string(sourceTask.Regions))
		sourceAssetTypes := normalizeStringList([]string(sourceTask.AssetTypes))
		targetRegions := firstNonEmptyStringSlice(overrideRegions, sourceRegions)
		targetAssetTypes := firstNonEmptyStringSlice(overrideAssetTypes, sourceAssetTypes)
		taskForm := &forms.CreateCmdbSyncTaskForm{
			AccountSource:          sourceTask.AccountSource,
			AccountId:              sourceTask.AccountId,
			SyncPolicyId:           sourceTask.SyncPolicyId,
			SyncPolicyScheduleKey:  attrString(sourceTask.Stats, "syncPolicyScheduleKey"),
			SyncPolicyScheduleName: attrString(sourceTask.Stats, "syncPolicyScheduleName"),
			Reason:                 reason,
			Provider:               sourceTask.Provider,
			Regions:                targetRegions,
			AssetTypes:             targetAssetTypes,
		}
		rerunMeta := &cmdbSyncTaskRerunMeta{
			GroupId:    groupId,
			FromTaskId: sourceTask.Id,
			Mode:       "batch_failed",
		}
		task, launch, createErr := prepareCmdbSyncTask(c, taskForm, rerunMeta)
		if createErr != nil {
			resp.Errors = append(resp.Errors, resps.CmdbSyncTaskBatchRerunError{
				TaskId:  sourceTask.Id,
				Message: createErr.Error(),
			})
			appendCmdbSyncTaskLog(c, sourceTask.Id, models.CmdbSyncLogLevelError, "rerun_failed", "批量重跑创建新任务失败", models.ResAttrs{
				"rerunGroupId": groupId,
				"reason":       reason,
				"error":        createErr.Error(),
			})
			continue
		}
		if parameterDiffs := cmdbSyncTaskRerunParameterDiffs(sourceRegions, targetRegions, sourceAssetTypes, targetAssetTypes, len(overrideRegions) > 0, len(overrideAssetTypes) > 0); parameterDiffs != nil {
			task.Stats["rerunParameterDiffs"] = parameterDiffs
		}
		if requiresApproval {
			cmdbSyncTaskApplyApprovalRequest(task, c, groupId, reason)
		}
		prepared = append(prepared, cmdbSyncTaskBatchPrepared{
			SourceTask: sourceTask,
			Task:       task,
			Launch:     launch,
		})
	}
	if len(resp.Errors) > 0 {
		resp.GroupId = ""
		return resp, nil
	}

	if err := c.DB().Transaction(func(tx *db.Session) error {
		for _, item := range prepared {
			if err := createCmdbSyncTaskRecords(tx, c, item.Task, item.Launch); err != nil {
				return err
			}
			logData := models.ResAttrs{
				"rerunGroupId": groupId,
				"rerunTaskId":  item.Task.Id,
				"reason":       reason,
			}
			if parameterDiffs := attrResAttrs(item.Task.Stats, "rerunParameterDiffs"); parameterDiffs != nil {
				logData["parameterDiffs"] = parameterDiffs
			}
			if err := createCmdbSyncTaskLog(tx, c, item.SourceTask.Id, models.CmdbSyncLogLevelInfo, "rerun_created", "已创建批量重跑任务", logData); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		resp.Errors = append(resp.Errors, resps.CmdbSyncTaskBatchRerunError{
			Message: fmt.Sprintf("批量重跑事务创建失败: %s", err.Error()),
		})
		for _, item := range prepared {
			appendCmdbSyncTaskLog(c, item.SourceTask.Id, models.CmdbSyncLogLevelError, "rerun_failed", "批量重跑事务创建失败", models.ResAttrs{
				"rerunGroupId": groupId,
				"reason":       reason,
				"error":        err.Error(),
			})
		}
		return resp, nil
	}

	for _, item := range prepared {
		resp.Created++
		resp.Items = append(resp.Items, resps.CmdbSyncTaskBatchRerunItem{
			SourceTaskId: item.SourceTask.Id,
			TaskId:       item.Task.Id,
			Status:       item.Task.Status,
		})
	}
	if requiresApproval {
		cmdbSyncTaskBatchRerunApprovalRequestedEvent(c, groupId, reason, prepared)
		return resp, nil
	}
	cmdbSyncTaskBatchRerunEvent(c, groupId, reason, prepared)
	launches := make([]cmdbSyncTaskLaunch, 0, len(prepared))
	for _, item := range prepared {
		launches = append(launches, item.Launch)
	}
	startCmdbSyncTaskBatch(c, launches)
	return resp, nil
}

func ApproveCmdbSyncTaskRerunGroup(c *ctx.ServiceContext, form *forms.CmdbSyncTaskRerunGroupApprovalForm) (*resps.CmdbSyncTaskRerunGroupResp, e.Error) {
	if err := ensureCmdbSyncTaskRerunApprovalPermission(c); err != nil {
		return nil, err
	}
	tasks, err := cmdbSyncTaskRerunGroupTasks(c, form.GroupId)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, e.New(e.BadParam, fmt.Errorf("批量重跑任务组 %s 不存在或不属于当前组织", form.GroupId))
	}
	for _, task := range tasks {
		if task.Status != models.CmdbSyncTaskApproving {
			return nil, e.New(e.BadParam, fmt.Errorf("批量重跑任务组 %s 当前包含 %s 任务，不能审批", form.GroupId, task.Status))
		}
		if approval := attrResAttrs(task.Stats, "approval"); attrString(approval, "status") != "pending" {
			return nil, e.New(e.BadParam, fmt.Errorf("批量重跑任务组 %s 当前审批状态不是待审批", form.GroupId))
		}
	}

	approval := cmdbSyncTaskRerunApprovalAttrs(c, form)
	if form.Action == "rejected" {
		if err := rejectCmdbSyncTaskRerunGroup(c, form.GroupId, tasks, approval); err != nil {
			return nil, err
		}
		cmdbSyncTaskBatchRerunFinishedEvent(c, form.GroupId)
		return CmdbSyncTaskRerunGroupDetail(c, &forms.CmdbSyncTaskRerunGroupParam{GroupId: form.GroupId})
	}

	launches := make([]cmdbSyncTaskLaunch, 0, len(tasks))
	prepared := make([]cmdbSyncTaskBatchPrepared, 0, len(tasks))
	for idx := range tasks {
		launch, err := cmdbSyncTaskLaunchFromPersistedTask(c, &tasks[idx])
		if err != nil {
			return nil, err
		}
		launches = append(launches, launch)
		prepared = append(prepared, cmdbSyncTaskBatchPrepared{
			SourceTask: cmdbSyncTaskSourceStubFromRerunStats(tasks[idx].Stats),
			Task:       &tasks[idx],
			Launch:     launch,
		})
	}
	if err := approveCmdbSyncTaskRerunGroup(c, tasks, approval); err != nil {
		return nil, err
	}
	reason := attrString(tasks[0].Stats, "reason")
	cmdbSyncTaskBatchRerunEvent(c, form.GroupId, reason, prepared)
	startCmdbSyncTaskBatch(c, launches)
	return CmdbSyncTaskRerunGroupDetail(c, &forms.CmdbSyncTaskRerunGroupParam{GroupId: form.GroupId})
}

func ensureCmdbSyncTaskRerunApprovalPermission(c *ctx.ServiceContext) e.Error {
	if c.IsSuperAdmin || services.UserHasOrgRole(c.UserId, c.OrgId, consts.OrgRoleAdmin) {
		return nil
	}
	return e.New(e.PermDenyApproval, fmt.Errorf("审批批量重跑任务组需要组织管理员或平台管理员权限"))
}

func cmdbSyncTaskRerunGroupTasks(c *ctx.ServiceContext, groupId models.Id) ([]models.CmdbSyncTask, e.Error) {
	tasks := make([]models.CmdbSyncTask, 0)
	if groupId == "" {
		return tasks, nil
	}
	if err := c.DB().Model(&models.CmdbSyncTask{}).
		Where("org_id = ? and JSON_UNQUOTE(JSON_EXTRACT(stats, '$.rerunGroupId')) = ?", c.OrgId, groupId.String()).
		Order("created_at asc").
		Find(&tasks); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return tasks, nil
}

func cmdbSyncTaskApplyApprovalRequest(task *models.CmdbSyncTask, c *ctx.ServiceContext, groupId models.Id, reason string) {
	if task == nil {
		return
	}
	stats := task.Stats
	if stats == nil {
		stats = models.ResAttrs{}
	}
	approvalId := models.NewId("csa")
	stats["stage"] = models.CmdbSyncTaskApproving
	stats["approval"] = models.ResAttrs{
		"required":     true,
		"approvalId":   approvalId.String(),
		"status":       "pending",
		"requestedBy":  c.UserId.String(),
		"requestedAt":  time.Now().Format(time.RFC3339),
		"reason":       reason,
		"rerunGroupId": groupId.String(),
	}
	task.Status = models.CmdbSyncTaskApproving
	task.StartedAt = models.Time(time.Time{})
	task.Stats = stats
}

func cmdbSyncTaskRerunApprovalAttrs(c *ctx.ServiceContext, form *forms.CmdbSyncTaskRerunGroupApprovalForm) models.ResAttrs {
	status := "approved"
	if form.Action == "rejected" {
		status = "rejected"
	}
	return models.ResAttrs{
		"status":       status,
		"action":       form.Action,
		"approverId":   c.UserId.String(),
		"approvedAt":   time.Now().Format(time.RFC3339),
		"comment":      strings.TrimSpace(form.Comment),
		"rerunGroupId": form.GroupId.String(),
	}
}

func approveCmdbSyncTaskRerunGroup(c *ctx.ServiceContext, tasks []models.CmdbSyncTask, approval models.ResAttrs) e.Error {
	if err := c.DB().Transaction(func(tx *db.Session) error {
		for _, task := range tasks {
			stats := cmdbSyncTaskStatsWithApproval(task.Stats, approval)
			stats["stage"] = models.CmdbSyncTaskPending
			if _, err := tx.Model(&models.CmdbSyncTask{}).
				Where("id = ? and org_id = ? and status = ?", task.Id, task.OrgId, models.CmdbSyncTaskApproving).
				UpdateAttrs(models.Attrs{
					"status": models.CmdbSyncTaskPending,
					"stats":  stats,
				}); err != nil {
				return err
			}
			if err := createCmdbSyncTaskLog(tx, c, task.Id, models.CmdbSyncLogLevelInfo, "approval_approved", "批量重跑任务审批通过，等待启动", models.ResAttrs{
				"approval": approval,
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return e.New(e.DBError, err)
	}
	return nil
}

func rejectCmdbSyncTaskRerunGroup(c *ctx.ServiceContext, groupId models.Id, tasks []models.CmdbSyncTask, approval models.ResAttrs) e.Error {
	now := models.Time(time.Now())
	if err := c.DB().Transaction(func(tx *db.Session) error {
		for _, task := range tasks {
			stats := cmdbSyncTaskStatsWithApproval(task.Stats, approval)
			stats["stage"] = models.CmdbSyncTaskRejected
			stats["durationMs"] = int64(0)
			if _, err := tx.Model(&models.CmdbSyncTask{}).
				Where("id = ? and org_id = ? and status = ?", task.Id, task.OrgId, models.CmdbSyncTaskApproving).
				UpdateAttrs(models.Attrs{
					"status":        models.CmdbSyncTaskRejected,
					"error_message": "批量重跑审批驳回",
					"stats":         stats,
					"ended_at":      now,
				}); err != nil {
				return err
			}
			if err := createCmdbSyncTaskLog(tx, c, task.Id, models.CmdbSyncLogLevelWarn, "approval_rejected", "批量重跑任务审批驳回", models.ResAttrs{
				"approval":     approval,
				"rerunGroupId": groupId.String(),
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return e.New(e.DBError, err)
	}
	return nil
}

func cmdbSyncTaskStatsWithApproval(stats models.ResAttrs, approval models.ResAttrs) models.ResAttrs {
	if stats == nil {
		stats = models.ResAttrs{}
	}
	existing := attrResAttrs(stats, "approval")
	merged := models.ResAttrs{}
	for key, value := range existing {
		merged[key] = value
	}
	for key, value := range approval {
		merged[key] = value
	}
	stats["approval"] = merged
	return stats
}

func cmdbSyncTaskRerunParameterDiffs(sourceRegions []string, targetRegions []string, sourceAssetTypes []string, targetAssetTypes []string, regionsOverridden bool, assetTypesOverridden bool) models.ResAttrs {
	regionsChanged := !cmdbSyncTaskStringSlicesSame(sourceRegions, targetRegions)
	assetTypesChanged := !cmdbSyncTaskStringSlicesSame(sourceAssetTypes, targetAssetTypes)
	if !regionsOverridden && !assetTypesOverridden && !regionsChanged && !assetTypesChanged {
		return nil
	}
	return models.ResAttrs{
		"changed": regionsChanged || assetTypesChanged,
		"regions": models.ResAttrs{
			"from":       cloneStringSlice(sourceRegions),
			"to":         cloneStringSlice(targetRegions),
			"overridden": regionsOverridden,
			"changed":    regionsChanged,
		},
		"assetTypes": models.ResAttrs{
			"from":       cloneStringSlice(sourceAssetTypes),
			"to":         cloneStringSlice(targetAssetTypes),
			"overridden": assetTypesOverridden,
			"changed":    assetTypesChanged,
		},
	}
}

func cmdbSyncTaskStringSlicesSame(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func cmdbSyncTaskLaunchFromPersistedTask(c *ctx.ServiceContext, task *models.CmdbSyncTask) (cmdbSyncTaskLaunch, e.Error) {
	account, err := findCmdbCloudAccount(c, task.AccountSource, task.AccountId)
	if err != nil {
		return cmdbSyncTaskLaunch{}, err
	}
	if task.Provider != "" {
		account.Provider = normalizeProvider(task.Provider)
	}
	rerunMeta := cmdbSyncTaskRerunMetaFromStats(task.Stats)
	launch := cmdbSyncTaskLaunch{
		TaskId:                 task.Id,
		Account:                cloneCmdbCloudAccount(account),
		Regions:                cloneStringSlice([]string(task.Regions)),
		AssetTypes:             cloneStringSlice([]string(task.AssetTypes)),
		SyncPolicyId:           task.SyncPolicyId,
		SyncPolicyScheduleKey:  attrString(task.Stats, "syncPolicyScheduleKey"),
		SyncPolicyScheduleName: attrString(task.Stats, "syncPolicyScheduleName"),
		Reason:                 attrString(task.Stats, "reason"),
		RerunMeta:              rerunMeta,
	}
	return launch, nil
}

func cmdbSyncTaskRerunMetaFromStats(stats models.ResAttrs) *cmdbSyncTaskRerunMeta {
	groupId := models.Id(attrString(stats, "rerunGroupId"))
	if groupId == "" {
		return nil
	}
	return &cmdbSyncTaskRerunMeta{
		GroupId:    groupId,
		FromTaskId: models.Id(attrString(stats, "rerunFromTaskId")),
		Mode:       attrString(stats, "rerunMode"),
	}
}

func cmdbSyncTaskSourceStubFromRerunStats(stats models.ResAttrs) models.CmdbSyncTask {
	source := models.CmdbSyncTask{}
	source.Id = models.Id(attrString(stats, "rerunFromTaskId"))
	return source
}

func cmdbSyncTaskUniqueIds(ids []models.Id) []models.Id {
	seen := make(map[models.Id]bool, len(ids))
	result := make([]models.Id, 0, len(ids))
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}

func startCmdbSyncTask(c *ctx.ServiceContext, form *forms.CreateCmdbSyncTaskForm, rerunMeta *cmdbSyncTaskRerunMeta) (*resps.CmdbSyncTaskResp, e.Error) {
	task, launch, err := prepareCmdbSyncTask(c, form, rerunMeta)
	if err != nil {
		return nil, err
	}
	if err := createCmdbSyncTaskRecords(c.DB(), c, task, launch); err != nil {
		return nil, err
	}
	launch.start(c)

	return &resps.CmdbSyncTaskResp{CmdbSyncTask: *task}, nil
}

func prepareCmdbSyncTask(c *ctx.ServiceContext, form *forms.CreateCmdbSyncTaskForm, rerunMeta *cmdbSyncTaskRerunMeta) (*models.CmdbSyncTask, cmdbSyncTaskLaunch, e.Error) {
	account, err := findCmdbCloudAccount(c, form.AccountSource, form.AccountId)
	if err != nil {
		return nil, cmdbSyncTaskLaunch{}, err
	}
	if form.Provider != "" {
		account.Provider = normalizeProvider(form.Provider)
	}
	if account.Provider == "" {
		return nil, cmdbSyncTaskLaunch{}, e.New(e.BadParam, fmt.Errorf("cloud provider cannot be inferred from account %s", account.Name), http.StatusBadRequest)
	}
	if form.SyncPolicyId != "" {
		policy, err := getCloudSyncPolicy(c, form.SyncPolicyId)
		if err != nil {
			return nil, cmdbSyncTaskLaunch{}, err
		}
		if account.Source != models.CmdbCloudAccountSourceCloudAccount || policy.CloudAccountId != account.Id {
			return nil, cmdbSyncTaskLaunch{}, e.New(e.BadParam, fmt.Errorf("同步策略 %s 与云账号 %s 不匹配", form.SyncPolicyId, account.Id), http.StatusBadRequest)
		}
		if policy.Provider != "" && account.Provider != "" && policy.Provider != account.Provider {
			return nil, cmdbSyncTaskLaunch{}, e.New(e.BadParam, fmt.Errorf("同步策略厂商 %s 与采集任务厂商 %s 不匹配", policy.Provider, account.Provider), http.StatusBadRequest)
		}
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
	reason := strings.TrimSpace(form.Reason)
	stats := initialCmdbSyncStats(regions, assetTypes)
	if form.SyncPolicyScheduleKey != "" {
		stats["syncPolicyScheduleKey"] = form.SyncPolicyScheduleKey
		stats["syncPolicyScheduleName"] = form.SyncPolicyScheduleName
	}
	if reason != "" {
		stats["reason"] = reason
	}
	cmdbSyncTaskApplyRerunMeta(stats, rerunMeta)
	task := &models.CmdbSyncTask{
		OrgId:         c.OrgId,
		AccountSource: account.Source,
		AccountId:     account.Id,
		SyncPolicyId:  form.SyncPolicyId,
		AccountName:   account.Name,
		Provider:      account.Provider,
		Regions:       models.StrSlice(regions),
		AssetTypes:    models.StrSlice(assetTypes),
		Status:        models.CmdbSyncTaskRunning,
		Stats:         stats,
		StartedAt:     now,
	}
	task.Id = models.NewId("cst")
	launch := cmdbSyncTaskLaunch{
		TaskId:                 task.Id,
		Account:                cloneCmdbCloudAccount(account),
		Regions:                cloneStringSlice(regions),
		AssetTypes:             cloneStringSlice(assetTypes),
		SyncPolicyId:           task.SyncPolicyId,
		SyncPolicyScheduleKey:  form.SyncPolicyScheduleKey,
		SyncPolicyScheduleName: form.SyncPolicyScheduleName,
		Reason:                 reason,
		RerunMeta:              rerunMeta,
	}
	return task, launch, nil
}

func createCmdbSyncTaskRecords(tx *db.Session, c *ctx.ServiceContext, task *models.CmdbSyncTask, launch cmdbSyncTaskLaunch) e.Error {
	if err := models.Create(tx, task); err != nil {
		return e.New(e.DBError, err)
	}
	if err := createCmdbSyncTaskLog(tx, c, task.Id, models.CmdbSyncLogLevelInfo, "created", "云采集任务已创建", models.ResAttrs{
		"accountSource":          task.AccountSource,
		"accountId":              task.AccountId,
		"syncPolicyId":           task.SyncPolicyId,
		"accountName":            task.AccountName,
		"provider":               task.Provider,
		"regions":                launch.Regions,
		"assetTypes":             launch.AssetTypes,
		"syncPolicyScheduleKey":  launch.SyncPolicyScheduleKey,
		"syncPolicyScheduleName": launch.SyncPolicyScheduleName,
		"reason":                 launch.Reason,
	}); err != nil {
		return e.New(e.DBError, err)
	}
	if launch.RerunMeta != nil {
		if err := createCmdbSyncTaskLog(tx, c, task.Id, models.CmdbSyncLogLevelInfo, "rerun_group", "云采集任务属于批量重跑任务组", models.ResAttrs{
			"rerunGroupId":    launch.RerunMeta.GroupId,
			"rerunFromTaskId": launch.RerunMeta.FromTaskId,
			"rerunMode":       launch.RerunMeta.Mode,
			"reason":          launch.Reason,
		}); err != nil {
			return e.New(e.DBError, err)
		}
	}
	if approval := attrResAttrs(task.Stats, "approval"); approval != nil {
		if err := createCmdbSyncTaskLog(tx, c, task.Id, models.CmdbSyncLogLevelInfo, models.CmdbSyncTaskApproving, "批量重跑任务已创建，等待审批", models.ResAttrs{
			"approval": approval,
			"reason":   launch.Reason,
		}); err != nil {
			return e.New(e.DBError, err)
		}
	}
	return nil
}

func (launch cmdbSyncTaskLaunch) start(c *ctx.ServiceContext) {
	go launch.run(c)
}

func (launch cmdbSyncTaskLaunch) run(c *ctx.ServiceContext) {
	runCmdbSyncTask(launch.TaskId, c, cloneCmdbCloudAccount(launch.Account), cloneStringSlice(launch.Regions), cloneStringSlice(launch.AssetTypes), launch.SyncPolicyId, launch.SyncPolicyScheduleKey, launch.SyncPolicyScheduleName, launch.Reason, launch.RerunMeta)
}

func startCmdbSyncTaskBatch(c *ctx.ServiceContext, launches []cmdbSyncTaskLaunch) {
	if len(launches) == 0 {
		return
	}
	batch := make([]cmdbSyncTaskLaunch, len(launches))
	copy(batch, launches)
	go func() {
		for _, launch := range batch {
			launch.run(c)
		}
	}()
}

func cmdbSyncTaskBatchRerunEvent(c *ctx.ServiceContext, groupId models.Id, reason string, prepared []cmdbSyncTaskBatchPrepared) {
	if c == nil || groupId == "" || len(prepared) == 0 {
		return
	}
	sourceTaskIds := make([]string, 0, len(prepared))
	newTaskIds := make([]string, 0, len(prepared))
	for _, item := range prepared {
		sourceTaskIds = append(sourceTaskIds, item.SourceTask.Id.String())
		newTaskIds = append(newTaskIds, item.Task.Id.String())
	}
	first := prepared[0].Task
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:          c.OrgId,
		CloudAccountId: first.AccountId,
		Source:         models.CloudEventSourceSync,
		EventType:      "cloud.sync.task.batch_rerun_started",
		Level:          models.CloudEventLevelInfo,
		Status:         models.CmdbSyncTaskRunning,
		Provider:       first.Provider,
		AccountId:      first.AccountId.String(),
		ResourceType:   "cmdb_sync_task_group",
		ResourceId:     groupId.String(),
		ResourceName:   groupId.String(),
		Title:          "云采集失败任务批量重跑已启动",
		Message:        fmt.Sprintf("已创建 %d 个失败任务重跑任务", len(prepared)),
		Payload: models.ResAttrs{
			"rerunGroupId":  groupId.String(),
			"reason":        reason,
			"mode":          "batch_failed",
			"created":       len(prepared),
			"sourceTaskIds": sourceTaskIds,
			"taskIds":       newTaskIds,
		},
	})
}

func cmdbSyncTaskBatchRerunApprovalRequestedEvent(c *ctx.ServiceContext, groupId models.Id, reason string, prepared []cmdbSyncTaskBatchPrepared) {
	if c == nil || groupId == "" || len(prepared) == 0 {
		return
	}
	sourceTaskIds := make([]string, 0, len(prepared))
	newTaskIds := make([]string, 0, len(prepared))
	provider := ""
	accountId := ""
	cloudAccountId := models.Id("")
	for _, item := range prepared {
		sourceTaskIds = append(sourceTaskIds, item.SourceTask.Id.String())
		if item.Task != nil {
			newTaskIds = append(newTaskIds, item.Task.Id.String())
			if provider == "" {
				provider = item.Task.Provider
			}
			if accountId == "" {
				accountId = item.Task.AccountId.String()
				cloudAccountId = item.Task.AccountId
			}
		}
	}
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:          c.OrgId,
		CloudAccountId: cloudAccountId,
		Source:         models.CloudEventSourceSync,
		EventType:      "cloud.sync.task.batch_rerun_approval_requested",
		Level:          models.CloudEventLevelInfo,
		Status:         models.CmdbSyncTaskApproving,
		Provider:       provider,
		AccountId:      accountId,
		ResourceType:   "cmdb_sync_task_group",
		ResourceId:     groupId.String(),
		ResourceName:   groupId.String(),
		Title:          "云采集失败任务批量重跑等待审批",
		Message:        fmt.Sprintf("已创建 %d 个失败任务重跑审批任务", len(prepared)),
		OccurredAt:     models.Time(time.Now()),
		Payload: models.ResAttrs{
			"rerunGroupId":     groupId.String(),
			"reason":           reason,
			"mode":             "batch_failed",
			"created":          len(prepared),
			"requiresApproval": true,
			"sourceTaskIds":    sourceTaskIds,
			"taskIds":          newTaskIds,
		},
	})
}

func cmdbSyncTaskBatchRerunFinishedEvent(c *ctx.ServiceContext, groupId models.Id) {
	if c == nil || groupId == "" {
		return
	}
	lockName := cmdbSyncTaskBatchRerunLockName(c.OrgId, groupId)
	locked, release, err := cmdbSyncTaskAcquireMysqlLock(lockName)
	if err != nil {
		logs.Get().WithField("rerunGroupId", groupId).Warnf("acquire cmdb sync rerun group lock failed: %v", err)
		return
	}
	if !locked {
		return
	}
	defer release()

	exists, err := c.DB().Model(&models.CloudEvent{}).
		Where("org_id = ? and event_type = ? and resource_type = ? and resource_id = ?",
			c.OrgId, "cloud.sync.task.batch_rerun_finished", "cmdb_sync_task_group", groupId.String()).
		Exists()
	if err != nil {
		logs.Get().WithField("rerunGroupId", groupId).Warnf("check cmdb sync rerun group event exists failed: %v", err)
		return
	}
	if exists {
		return
	}

	tasks := make([]models.CmdbSyncTask, 0)
	if err := c.DB().Model(&models.CmdbSyncTask{}).
		Where("org_id = ? and JSON_UNQUOTE(JSON_EXTRACT(stats, '$.rerunGroupId')) = ?", c.OrgId, groupId.String()).
		Order("created_at asc").
		Find(&tasks); err != nil {
		logs.Get().WithField("rerunGroupId", groupId).Warnf("query cmdb sync rerun group tasks failed: %v", err)
		return
	}
	if len(tasks) == 0 {
		return
	}

	sourceTaskIds := make([]string, 0, len(tasks))
	taskIds := make([]string, 0, len(tasks))
	reason := ""
	mode := ""
	provider := ""
	accountId := ""
	cloudAccountId := models.Id("")
	createdAt := models.Time{}
	startedAt := models.Time{}
	endedAt := models.Time{}
	approvingCount := 0
	pendingCount := 0
	runningCount := 0
	completeCount := 0
	failedCount := 0
	rejectedCount := 0
	for _, task := range tasks {
		taskIds = append(taskIds, task.Id.String())
		if sourceTaskId := attrString(task.Stats, "rerunFromTaskId"); sourceTaskId != "" {
			sourceTaskIds = append(sourceTaskIds, sourceTaskId)
		}
		if reason == "" {
			reason = attrString(task.Stats, "reason")
		}
		if mode == "" {
			mode = attrString(task.Stats, "rerunMode")
		}
		if provider == "" {
			provider = task.Provider
		}
		if accountId == "" {
			accountId = task.AccountId.String()
			cloudAccountId = task.AccountId
		}
		createdAt = cmdbSyncTaskMinTime(createdAt, task.CreatedAt)
		startedAt = cmdbSyncTaskMinTime(startedAt, task.StartedAt)
		endedAt = cmdbSyncTaskMaxTime(endedAt, task.EndedAt)
		switch task.Status {
		case models.CmdbSyncTaskApproving:
			approvingCount++
		case models.CmdbSyncTaskPending:
			pendingCount++
		case models.CmdbSyncTaskRunning:
			runningCount++
		case models.CmdbSyncTaskComplete:
			completeCount++
		case models.CmdbSyncTaskFailed:
			failedCount++
		case models.CmdbSyncTaskRejected:
			rejectedCount++
		}
	}
	if approvingCount > 0 || pendingCount > 0 || runningCount > 0 {
		return
	}

	status := models.CmdbSyncTaskComplete
	level := models.CloudEventLevelInfo
	title := "云采集失败任务批量重跑已完成"
	message := fmt.Sprintf("任务组 %s 已结束，成功 %d 个，失败 %d 个，驳回 %d 个", groupId.String(), completeCount, failedCount, rejectedCount)
	if failedCount > 0 {
		status = models.CmdbSyncTaskFailed
		level = models.CloudEventLevelWarning
		title = "云采集失败任务批量重跑存在失败"
	} else if rejectedCount > 0 {
		status = models.CmdbSyncTaskRejected
		level = models.CloudEventLevelWarning
		title = "云采集失败任务批量重跑已驳回"
	}
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:          c.OrgId,
		CloudAccountId: cloudAccountId,
		Source:         models.CloudEventSourceSync,
		EventType:      "cloud.sync.task.batch_rerun_finished",
		Level:          level,
		Status:         status,
		Provider:       provider,
		AccountId:      accountId,
		ResourceType:   "cmdb_sync_task_group",
		ResourceId:     groupId.String(),
		ResourceName:   groupId.String(),
		Title:          title,
		Message:        message,
		OccurredAt:     endedAt,
		Payload: models.ResAttrs{
			"rerunGroupId":   groupId.String(),
			"reason":         reason,
			"mode":           mode,
			"total":          len(tasks),
			"approvingCount": approvingCount,
			"pendingCount":   pendingCount,
			"runningCount":   runningCount,
			"completeCount":  completeCount,
			"failedCount":    failedCount,
			"rejectedCount":  rejectedCount,
			"createdAt":      createdAt,
			"startedAt":      startedAt,
			"endedAt":        endedAt,
			"sourceTaskIds":  cmdbSyncTaskUniqueStrings(sourceTaskIds),
			"taskIds":        taskIds,
		},
	})
}

func cmdbSyncTaskBatchRerunLockName(orgId models.Id, groupId models.Id) string {
	sum := md5.Sum([]byte(fmt.Sprintf("%s:%s", orgId.String(), groupId.String())))
	return fmt.Sprintf("cloudiac:cmdb_rerun:%x", sum)
}

func cmdbSyncTaskAcquireMysqlLock(lockName string) (bool, func(), error) {
	lockTx := db.Get().Begin()
	locked := 0
	if err := lockTx.Raw("select get_lock(?, 0)", lockName).Scan(&locked); err != nil {
		_ = lockTx.Rollback()
		return false, nil, err
	}
	if locked != 1 {
		_ = lockTx.Rollback()
		return false, func() {}, nil
	}
	released := false
	release := func() {
		if released {
			return
		}
		released = true
		releaseResult := 0
		if err := lockTx.Raw("select release_lock(?)", lockName).Scan(&releaseResult); err != nil {
			logs.Get().WithField("lock", lockName).Warnf("release cmdb sync rerun group lock failed: %v", err)
		}
		if err := lockTx.Commit(); err != nil {
			logs.Get().WithField("lock", lockName).Warnf("commit cmdb sync rerun group lock transaction failed: %v", err)
		}
	}
	return true, release, nil
}

func cmdbSyncTaskUniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func cmdbSyncTaskApplyRerunMeta(stats models.ResAttrs, rerunMeta *cmdbSyncTaskRerunMeta) {
	if stats == nil || rerunMeta == nil {
		return
	}
	if rerunMeta.GroupId != "" {
		stats["rerunGroupId"] = rerunMeta.GroupId.String()
	}
	if rerunMeta.FromTaskId != "" {
		stats["rerunFromTaskId"] = rerunMeta.FromTaskId.String()
	}
	if rerunMeta.Mode != "" {
		stats["rerunMode"] = rerunMeta.Mode
	}
}

func runCmdbSyncTask(taskId models.Id, requestCtx *ctx.ServiceContext, account *cmdbCloudAccount, regions, assetTypes []string, syncPolicyId models.Id, syncPolicyScheduleKey string, syncPolicyScheduleName string, reason string, rerunMeta *cmdbSyncTaskRerunMeta) {
	taskStartedAt := time.Now()
	workerCtx := &ctx.ServiceContext{
		UserId:       requestCtx.UserId,
		OrgId:        requestCtx.OrgId,
		ProjectId:    requestCtx.ProjectId,
		IsSuperAdmin: requestCtx.IsSuperAdmin,
	}
	logger := logs.Get().WithField("cmdbSyncTaskId", taskId)
	approval := cmdbSyncTaskExistingApproval(workerCtx, taskId)
	rerunParameterDiffs := cmdbSyncTaskExistingRerunParameterDiffs(workerCtx, taskId)
	stats := initialCmdbSyncStats(regions, assetTypes)
	if syncPolicyScheduleKey != "" {
		stats["syncPolicyScheduleKey"] = syncPolicyScheduleKey
		stats["syncPolicyScheduleName"] = syncPolicyScheduleName
	}
	if reason != "" {
		stats["reason"] = reason
	}
	if approval != nil {
		stats["approval"] = approval
	}
	if rerunParameterDiffs != nil {
		stats["rerunParameterDiffs"] = rerunParameterDiffs
	}
	cmdbSyncTaskApplyRerunMeta(stats, rerunMeta)
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
		stats["durationMs"] = time.Since(taskStartedAt).Milliseconds()
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
		if syncPolicyId != "" {
			if err := updateCloudSyncPolicyAfterTask(workerCtx, syncPolicyId, taskId, status, errorMessage, endedAt, stats); err != nil {
				logger.Errorf("update cloud sync policy status error: %v", err)
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
		if rerunMeta != nil && rerunMeta.GroupId != "" {
			cmdbSyncTaskBatchRerunFinishedEvent(workerCtx, rerunMeta.GroupId)
		}
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
		"regions":                regions,
		"assetTypes":             assetTypes,
		"syncPolicyId":           syncPolicyId,
		"syncPolicyScheduleKey":  syncPolicyScheduleKey,
		"syncPolicyScheduleName": syncPolicyScheduleName,
		"reason":                 reason,
	})

	stats["stage"] = "collecting"
	_ = updateCmdbSyncTask(workerCtx, taskId, map[string]interface{}{"stats": stats})
	appendCmdbSyncTaskLog(workerCtx, taskId, models.CmdbSyncLogLevelInfo, "collecting", "开始调用云厂商采集器", models.ResAttrs{
		"provider": account.Provider,
	})
	runStats, runErr := runCmdbCloudCollector(workerCtx, account, regions, assetTypes, syncPolicyId)
	if runStats != nil {
		stats = runStats
	}
	if syncPolicyScheduleKey != "" {
		stats["syncPolicyScheduleKey"] = syncPolicyScheduleKey
		stats["syncPolicyScheduleName"] = syncPolicyScheduleName
	}
	if reason != "" {
		stats["reason"] = reason
	}
	if approval != nil {
		stats["approval"] = approval
	}
	if rerunParameterDiffs != nil {
		stats["rerunParameterDiffs"] = rerunParameterDiffs
	}
	cmdbSyncTaskApplyRerunMeta(stats, rerunMeta)
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

func cmdbSyncTaskExistingApproval(c *ctx.ServiceContext, taskId models.Id) models.ResAttrs {
	if c == nil || taskId == "" {
		return nil
	}
	task := models.CmdbSyncTask{}
	if err := c.DB().Model(&models.CmdbSyncTask{}).
		Select("stats").
		Where("id = ? and org_id = ?", taskId, c.OrgId).
		First(&task); err != nil {
		logs.Get().WithField("cmdbSyncTaskId", taskId).Warnf("load cmdb sync task approval stats failed: %v", err)
		return nil
	}
	return attrResAttrs(task.Stats, "approval")
}

func cmdbSyncTaskExistingRerunParameterDiffs(c *ctx.ServiceContext, taskId models.Id) models.ResAttrs {
	if c == nil || taskId == "" {
		return nil
	}
	task := models.CmdbSyncTask{}
	if err := c.DB().Model(&models.CmdbSyncTask{}).
		Select("stats").
		Where("id = ? and org_id = ?", taskId, c.OrgId).
		First(&task); err != nil {
		logs.Get().WithField("cmdbSyncTaskId", taskId).Warnf("load cmdb sync task rerun parameter diffs failed: %v", err)
		return nil
	}
	return attrResAttrs(task.Stats, "rerunParameterDiffs")
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
	if err := createCmdbSyncTaskLog(c.DB(), c, taskId, level, stage, message, data); err != nil {
		logs.Get().WithField("cmdbSyncTaskId", taskId).Warnf("append cmdb sync task log failed: %v", err)
	}
}

func createCmdbSyncTaskLog(tx *db.Session, c *ctx.ServiceContext, taskId models.Id, level, stage, message string, data models.ResAttrs) error {
	if tx == nil || c == nil || taskId == "" {
		return nil
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
	return models.Create(tx, log)
}

func runCmdbCloudCollector(c *ctx.ServiceContext, account *cmdbCloudAccount, regions, assetTypes []string, syncPolicyId models.Id) (models.ResAttrs, error) {
	startedAt := time.Now()
	stats := initialCmdbSyncStats(regions, assetTypes)
	if !account.Ready {
		stats["missingCredentialKeys"] = account.MissingCredentialKeys
		messages := []string{fmt.Sprintf("cloud account %s missing credentials: %s", account.Name, strings.Join(account.MissingCredentialKeys, ", "))}
		stats["failureDetails"] = cmdbSyncFailureDetails(messages)
		stats["failureSummary"] = cmdbSyncFailureSummary(messages)
		stats["durationMs"] = time.Since(startedAt).Milliseconds()
		return stats, fmt.Errorf("cloud account %s missing credentials: %s", account.Name, strings.Join(account.MissingCredentialKeys, ", "))
	}

	collectorStartedAt := time.Now()
	result := collectCmdbCloudAssets(account, regions, assetTypes)
	collectorDurationMs := time.Since(collectorStartedAt).Milliseconds()
	mergeCmdbStats(stats, result.Stats)
	stats["collected"] = len(result.Assets)
	stats["collectorDurationMs"] = collectorDurationMs
	stats["collectorStatus"] = cmdbSyncCollectorStatus(result.Err)
	errorMessages := cmdbSyncErrorMessages(result.Stats, result.Err)
	stats["failureDetails"] = cmdbSyncFailureDetails(errorMessages)
	stats["failureSummary"] = cmdbSyncFailureSummary(errorMessages)
	stats["regionMetrics"] = cmdbSyncRegionMetrics(regions, result.Assets, result.Err)
	stats["assetTypeMetrics"] = cmdbSyncAssetTypeMetrics(assetTypes, result.Assets, result.Err)
	stats["scopeMetrics"] = cmdbSyncScopeMetrics(regions, assetTypes, result.Assets, collectorDurationMs, result.Err)
	if syncPolicyId != "" {
		stats["syncPolicyId"] = syncPolicyId.String()
		for _, asset := range result.Assets {
			if asset != nil {
				asset.SyncPolicyId = syncPolicyId
			}
		}
	}

	upsertStartedAt := time.Now()
	upsertResp, upsertErr := upsertCmdbCloudAssets(c, result.Assets)
	stats["upsertDurationMs"] = time.Since(upsertStartedAt).Milliseconds()
	if upsertErr != nil {
		stats["durationMs"] = time.Since(startedAt).Milliseconds()
		return stats, upsertErr
	}
	if upsertResp != nil {
		stats["created"] = upsertResp.Created
		stats["updated"] = upsertResp.Updated
		stats["skipped"] = upsertResp.Skipped
	}
	relationStartedAt := time.Now()
	relationCount, relationErr := rebuildCmdbCloudInferredRelations(c, account)
	stats["relationDurationMs"] = time.Since(relationStartedAt).Milliseconds()
	if relationErr != nil {
		stats["durationMs"] = time.Since(startedAt).Milliseconds()
		return stats, relationErr
	}
	stats["cloudInferredRelations"] = relationCount
	stats["durationMs"] = time.Since(startedAt).Milliseconds()
	return stats, result.Err
}

func cmdbSyncCollectorStatus(err error) string {
	if err != nil {
		return "failed"
	}
	return "complete"
}

func cmdbSyncErrorMessages(stats models.ResAttrs, err error) []string {
	messages := make([]string, 0)
	if stats != nil {
		switch errors := stats["errors"].(type) {
		case []string:
			messages = append(messages, errors...)
		case []interface{}:
			for _, item := range errors {
				if msg := strings.TrimSpace(fmt.Sprint(item)); msg != "" {
					messages = append(messages, msg)
				}
			}
		case string:
			if msg := strings.TrimSpace(errors); msg != "" {
				messages = append(messages, msg)
			}
		}
	}
	if len(messages) == 0 && err != nil {
		messages = append(messages, err.Error())
	}
	return dedupeStrings(messages)
}

func cmdbSyncFailureDetails(messages []string) []models.ResAttrs {
	details := make([]models.ResAttrs, 0, len(messages))
	for _, message := range messages {
		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}
		category, retryable, retryHint := cmdbSyncClassifyFailure(message)
		details = append(details, models.ResAttrs{
			"message":   message,
			"category":  category,
			"retryable": retryable,
			"retryHint": retryHint,
		})
	}
	return details
}

func cmdbSyncFailureSummary(messages []string) models.ResAttrs {
	total := 0
	retryableTotal := 0
	for _, message := range messages {
		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}
		_, retryable, _ := cmdbSyncClassifyFailure(message)
		total++
		if retryable {
			retryableTotal++
		}
	}
	return models.ResAttrs{
		"total":          total,
		"retryableTotal": retryableTotal,
	}
}

func cmdbSyncClassifyFailure(message string) (string, bool, string) {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "rate") || strings.Contains(lower, "throttl") ||
		strings.Contains(lower, "too many requests") || strings.Contains(lower, "request limit"):
		return "rate_limit", true, "等待限流窗口恢复后重试，或缩小 regions/assetTypes 范围"
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "connection") ||
		strings.Contains(lower, "no such host") || strings.Contains(lower, "temporary"):
		return "network", true, "网络或云 API 临时异常，可直接重试"
	case strings.Contains(lower, "unauthorized") || strings.Contains(lower, "forbidden") ||
		strings.Contains(lower, "permission") || strings.Contains(lower, "denied"):
		return "permission", false, "检查云账号权限策略后再重试"
	case strings.Contains(lower, "expired") || strings.Contains(lower, "invalid token") ||
		strings.Contains(lower, "invalid access") || strings.Contains(lower, "signature"):
		return "credential", false, "更新云账号凭证后再重试"
	case strings.Contains(lower, "missing") || strings.Contains(lower, "requires"):
		return "configuration", false, "补齐云账号必需配置后再重试"
	default:
		return "unknown", true, "确认错误原因后可按相同 regions/assetTypes 重试"
	}
}

func cmdbSyncRegionMetrics(regions []string, assets []*models.CmdbAsset, err error) []models.ResAttrs {
	counts := make(map[string]int)
	order := normalizeStringList(regions)
	for _, region := range order {
		counts[region] = 0
	}
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		region := strings.TrimSpace(asset.Region)
		if region == "" {
			region = "global"
		}
		if _, ok := counts[region]; !ok {
			order = append(order, region)
		}
		counts[region]++
	}
	metrics := make([]models.ResAttrs, 0, len(order))
	status := cmdbSyncMetricStatus(err)
	for _, region := range order {
		metrics = append(metrics, models.ResAttrs{
			"region":    region,
			"collected": counts[region],
			"status":    status,
		})
	}
	return metrics
}

func cmdbSyncAssetTypeMetrics(assetTypes []string, assets []*models.CmdbAsset, err error) []models.ResAttrs {
	counts := make(map[string]int)
	order := normalizeStringList(assetTypes)
	for _, assetType := range order {
		counts[assetType] = 0
	}
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		assetType := strings.TrimSpace(asset.AssetType)
		if assetType == "" {
			assetType = "unknown"
		}
		if _, ok := counts[assetType]; !ok {
			order = append(order, assetType)
		}
		counts[assetType]++
	}
	metrics := make([]models.ResAttrs, 0, len(order))
	status := cmdbSyncMetricStatus(err)
	for _, assetType := range order {
		metrics = append(metrics, models.ResAttrs{
			"assetType": assetType,
			"collected": counts[assetType],
			"status":    status,
		})
	}
	return metrics
}

func cmdbSyncScopeMetrics(regions []string, assetTypes []string, assets []*models.CmdbAsset, collectorDurationMs int64, err error) []models.ResAttrs {
	counts := make(map[string]int)
	labels := make(map[string]models.ResAttrs)
	regionList := normalizeStringList(regions)
	assetTypeList := normalizeStringList(assetTypes)
	if len(regionList) > 0 && len(assetTypeList) > 0 {
		for _, region := range regionList {
			for _, assetType := range assetTypeList {
				key := cmdbSyncScopeKey(region, assetType)
				counts[key] = 0
				labels[key] = models.ResAttrs{"region": region, "assetType": assetType}
			}
		}
	}
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		region := strings.TrimSpace(asset.Region)
		if region == "" {
			region = "global"
		}
		assetType := strings.TrimSpace(asset.AssetType)
		if assetType == "" {
			assetType = "unknown"
		}
		key := cmdbSyncScopeKey(region, assetType)
		if _, ok := counts[key]; !ok {
			labels[key] = models.ResAttrs{"region": region, "assetType": assetType}
		}
		counts[key]++
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	metrics := make([]models.ResAttrs, 0, len(keys))
	status := cmdbSyncMetricStatus(err)
	for _, key := range keys {
		item := labels[key]
		item["collected"] = counts[key]
		item["status"] = status
		if len(regionList) == 1 && len(assetTypeList) == 1 {
			item["durationMs"] = collectorDurationMs
		}
		metrics = append(metrics, item)
	}
	return metrics
}

func cmdbSyncScopeKey(region string, assetType string) string {
	return strings.TrimSpace(region) + "\x00" + strings.TrimSpace(assetType)
}

func cmdbSyncMetricStatus(err error) string {
	if err != nil {
		return "partial_failed"
	}
	return "complete"
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
	case keySet["AZURE_TENANT_ID"] || keySet["AZURE_CLIENT_ID"] || keySet["AZURE_SUBSCRIPTION_ID"] || keySet["ARM_TENANT_ID"] || keySet["ARM_CLIENT_ID"] || keySet["ARM_SUBSCRIPTION_ID"]:
		return "azure"
	case keySet["GCP_PROJECT_ID"] || keySet["GOOGLE_CLOUD_PROJECT"] || keySet["GCP_ACCESS_TOKEN"] || keySet["GCP_SERVICE_ACCOUNT_JSON"]:
		return "gcp"
	case keySet["TENCENTCLOUD_INVENTORY_JSON"] || keySet["TENCENT_INVENTORY_JSON"] || keySet["TENCENTCLOUD_SECRET_ID"]:
		return "tencentcloud"
	case keySet["HUAWEI_INVENTORY_JSON"] || keySet["HUAWEICLOUD_INVENTORY_JSON"] || keySet["HUAWEI_ACCESS_KEY"]:
		return "huawei"
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
		"aws":          {"AWS_REGIONS", "AWS_REGION", "AWS_DEFAULT_REGION"},
		"oci":          {"OCI_REGIONS", "OCI_REGION"},
		"alicloud":     {"ALICLOUD_REGIONS", "ALICLOUD_REGION"},
		"azure":        {"AZURE_REGIONS", "AZURE_REGION", "ARM_REGIONS", "ARM_REGION"},
		"gcp":          {"GCP_REGIONS", "GCP_REGION", "GOOGLE_CLOUD_REGION"},
		"tencentcloud": {"TENCENTCLOUD_REGIONS", "TENCENTCLOUD_REGION", "TENCENT_REGIONS", "TENCENT_REGION"},
		"huawei":       {"HUAWEI_REGIONS", "HUAWEI_REGION", "HUAWEICLOUD_REGIONS", "HUAWEICLOUD_REGION"},
	}
	regions := make([]string, 0)
	for _, key := range keys[provider] {
		regions = append(regions, splitListValue(credentials[key])...)
	}
	return dedupeStrings(regions)
}

func missingCloudCredentialKeys(provider string, credentials map[string]string) []string {
	switch provider {
	case "tencentcloud":
		if hasTencentLiveCredentials(credentials) || hasAnyCredential(credentials, "TENCENTCLOUD_INVENTORY_JSON", "TENCENT_INVENTORY_JSON", "CLOUD_INVENTORY_JSON") {
			return nil
		}
		return []string{"TENCENTCLOUD_SECRET_ID+TENCENTCLOUD_SECRET_KEY|TENCENTCLOUD_INVENTORY_JSON|TENCENT_INVENTORY_JSON|CLOUD_INVENTORY_JSON"}
	case "huawei":
		if hasHuaweiLiveCredentials(credentials) || hasAnyCredential(credentials, "HUAWEI_INVENTORY_JSON", "HUAWEICLOUD_INVENTORY_JSON", "CLOUD_INVENTORY_JSON") {
			return nil
		}
		return []string{"HUAWEI_AUTH_TOKEN+HUAWEI_PROJECT_ID|HUAWEI_INVENTORY_JSON|HUAWEICLOUD_INVENTORY_JSON|CLOUD_INVENTORY_JSON"}
	}

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
		"azure":    {"AZURE_TENANT_ID", "AZURE_CLIENT_ID", "AZURE_CLIENT_SECRET", "AZURE_SUBSCRIPTION_ID"},
		"gcp":      {"GCP_PROJECT_ID"},
	}
	return required[provider]
}

func requiredCloudCredentialKeyGroups(provider string) [][]string {
	switch provider {
	case "oci":
		return [][]string{{"OCI_REGION", "OCI_REGIONS"}}
	case "gcp":
		return [][]string{{"GCP_ACCESS_TOKEN", "GCP_SERVICE_ACCOUNT_JSON"}}
	case "tencentcloud":
		return [][]string{{"TENCENTCLOUD_INVENTORY_JSON", "TENCENT_INVENTORY_JSON", "CLOUD_INVENTORY_JSON"}}
	case "huawei":
		return [][]string{{"HUAWEI_INVENTORY_JSON", "HUAWEICLOUD_INVENTORY_JSON", "CLOUD_INVENTORY_JSON"}}
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
			models.CmdbAssetTypeKubernetesCluster,
			models.CmdbAssetTypeNetworkVpc,
			models.CmdbAssetTypeNetworkSubnet,
			models.CmdbAssetTypeNetworkSecurityGroup,
			models.CmdbAssetTypePublicIP,
			models.CmdbAssetTypeLoadBalancer,
			models.CmdbAssetTypeBlockVolume,
			models.CmdbAssetTypeRelationalDatabase,
			models.CmdbAssetTypeRedisCache,
			models.CmdbAssetTypeObjectStorageBucket,
		}
	case "azure":
		return []string{
			models.CmdbAssetTypeComputeInstance,
			models.CmdbAssetTypeKubernetesCluster,
			models.CmdbAssetTypeNetworkVpc,
			models.CmdbAssetTypeNetworkSubnet,
			models.CmdbAssetTypeNetworkSecurityGroup,
			models.CmdbAssetTypePublicIP,
			models.CmdbAssetTypeLoadBalancer,
			models.CmdbAssetTypeBlockVolume,
			models.CmdbAssetTypeRelationalDatabase,
			models.CmdbAssetTypeObjectStorageBucket,
		}
	case "gcp":
		return []string{
			models.CmdbAssetTypeComputeInstance,
			models.CmdbAssetTypeKubernetesCluster,
			models.CmdbAssetTypeNetworkVpc,
			models.CmdbAssetTypeNetworkSubnet,
			models.CmdbAssetTypeNetworkSecurityGroup,
			models.CmdbAssetTypeLoadBalancer,
			models.CmdbAssetTypeBlockVolume,
			models.CmdbAssetTypeRelationalDatabase,
			models.CmdbAssetTypeObjectStorageBucket,
		}
	case "tencentcloud", "huawei":
		return []string{
			models.CmdbAssetTypeComputeInstance,
			models.CmdbAssetTypeKubernetesCluster,
			models.CmdbAssetTypeNetworkVpc,
			models.CmdbAssetTypeNetworkSubnet,
			models.CmdbAssetTypeNetworkSecurityGroup,
			models.CmdbAssetTypePublicIP,
			models.CmdbAssetTypeLoadBalancer,
			models.CmdbAssetTypeBlockVolume,
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
