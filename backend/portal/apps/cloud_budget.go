// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"fmt"
	"net/http"
	"os"
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

type cloudBudgetAmountRow struct {
	Amount float64 `gorm:"column:amount"`
}

type cloudBudgetEvaluationResult struct {
	Amount             float64
	UsagePercent       float64
	ThresholdReached   bool
	BudgetExceeded     bool
	NotificationPushed bool
}

type cloudBudgetDueGroup struct {
	OrgId    models.Id `gorm:"column:org_id"`
	Period   string    `gorm:"column:period"`
	Currency string    `gorm:"column:currency"`
	DueCount int64     `gorm:"column:due_count"`
}

const (
	cloudBudgetEvaluationWorkerDefaultInterval = 60 * time.Second
	cloudBudgetEvaluationWorkerMinInterval     = 15 * time.Second
	cloudBudgetEvaluationWorkerMaxInterval     = 24 * time.Hour
)

func SearchCloudBudgets(c *ctx.ServiceContext, form *forms.SearchCloudBudgetForm) (interface{}, e.Error) {
	period := firstNonEmpty(form.Period, time.Now().Format("2006-01"))
	currency := firstNonEmpty(form.Currency, "CNY")
	if err := RefreshCloudCostRecords(c); err != nil {
		return nil, err
	}
	if err := EvaluateCloudBudgets(c, period, currency); err != nil {
		return nil, err
	}

	query := applyCloudBudgetSearch(cloudBudgetBaseQuery(c), form)
	if form.SortField() == "" {
		query = query.Order("period desc").Order("last_usage_percent desc").Order("created_at desc")
	} else {
		query = form.Order(query)
	}

	budgets := make([]models.CloudBudget, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&budgets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	list := make([]resps.CloudBudgetResp, 0, len(budgets))
	for _, budget := range budgets {
		list = append(list, cloudBudgetResp(c, budget))
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func CloudBudgetSummary(c *ctx.ServiceContext, form *forms.CloudBudgetSummaryForm) (*resps.CloudBudgetSummaryResp, e.Error) {
	period := firstNonEmpty(form.Period, time.Now().Format("2006-01"))
	currency := firstNonEmpty(form.Currency, "CNY")
	if err := RefreshCloudCostRecords(c); err != nil {
		return nil, err
	}
	if err := EvaluateCloudBudgets(c, period, currency); err != nil {
		return nil, err
	}

	budgets := make([]models.CloudBudget, 0)
	if err := cloudBudgetBaseQuery(c).
		Where("status = ? and period = ? and currency = ?", models.CloudBudgetStatusEnabled, period, currency).
		Find(&budgets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	resp := &resps.CloudBudgetSummaryResp{
		Period:   period,
		Currency: currency,
	}
	for _, budget := range budgets {
		item := cloudBudgetResp(c, budget)
		resp.BudgetCount++
		resp.TotalLimitAmount += budget.LimitAmount
		resp.TotalCurrentAmount += item.CurrentAmount
		if item.ThresholdReached {
			resp.ThresholdCount++
		}
		if item.BudgetExceeded {
			resp.ExceededCount++
		}
	}
	return resp, nil
}

func EvaluateDueCloudBudgets(c *ctx.ServiceContext, form *forms.EvaluateCloudBudgetForm) (*resps.CloudBudgetEvaluateResp, e.Error) {
	period := firstNonEmpty(form.Period, time.Now().Format("2006-01"))
	currency := firstNonEmpty(strings.ToUpper(form.Currency), "CNY")
	if err := RefreshCloudCostRecords(c); err != nil {
		return nil, err
	}
	budgets := make([]models.CloudBudget, 0)
	if err := cloudBudgetBaseQuery(c).
		Where("status = ? and period = ? and currency = ?", models.CloudBudgetStatusEnabled, period, currency).
		Find(&budgets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	resp := &resps.CloudBudgetEvaluateResp{
		Period:   period,
		Currency: currency,
		Force:    form.Force,
	}
	now := time.Now()
	for idx := range budgets {
		budget := &budgets[idx]
		resp.TotalCount++
		resp.TotalLimitAmount += budget.LimitAmount
		if !form.Force && !cloudBudgetEvaluationDue(*budget, now) {
			resp.SkippedCount++
			resp.TotalCurrentAmount += budget.LastAmount
			if budget.LimitAmount > 0 && budget.LastUsagePercent >= budget.ThresholdPercent {
				resp.ThresholdCount++
			}
			if budget.LimitAmount > 0 && budget.LastAmount >= budget.LimitAmount {
				resp.ExceededCount++
			}
			continue
		}
		result, err := evaluateSingleCloudBudgetResult(c, budget)
		if err != nil {
			return nil, err
		}
		resp.EvaluatedCount++
		resp.TotalCurrentAmount += result.Amount
		if result.ThresholdReached {
			resp.ThresholdCount++
		}
		if result.BudgetExceeded {
			resp.ExceededCount++
		}
		if result.NotificationPushed {
			resp.NotificationCount++
		}
	}
	return resp, nil
}

func CreateCloudBudget(c *ctx.ServiceContext, form *forms.CreateCloudBudgetForm) (*resps.CloudBudgetResp, e.Error) {
	budget, err := cloudBudgetFromCreateForm(c, form)
	if err != nil {
		return nil, err
	}
	budget.Id = models.NewId("cbd")
	if err := models.Create(c.DB(), budget); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if err := RefreshCloudCostRecords(c); err != nil {
		return nil, err
	}
	if err := evaluateSingleCloudBudget(c, budget); err != nil {
		return nil, err
	}
	resp := cloudBudgetResp(c, *budget)
	return &resp, nil
}

func UpdateCloudBudget(c *ctx.ServiceContext, form *forms.UpdateCloudBudgetForm) (*resps.CloudBudgetResp, e.Error) {
	budget := models.CloudBudget{}
	if err := cloudBudgetBaseQuery(c).Where("id = ?", form.Id).First(&budget); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	updated, err := cloudBudgetFromCreateForm(c, &form.CreateCloudBudgetForm)
	if err != nil {
		return nil, err
	}
	attrs := models.Attrs{
		"name":                updated.Name,
		"description":         updated.Description,
		"project_id":          updated.ProjectId,
		"env_id":              updated.EnvId,
		"cloud_account_id":    updated.CloudAccountId,
		"scope":               updated.Scope,
		"provider":            updated.Provider,
		"account_id":          updated.AccountId,
		"region":              updated.Region,
		"application":         updated.Application,
		"business_line":       updated.BusinessLine,
		"cost_center":         updated.CostCenter,
		"owner":               updated.Owner,
		"period":              updated.Period,
		"currency":            updated.Currency,
		"limit_amount":        updated.LimitAmount,
		"threshold_percent":   updated.ThresholdPercent,
		"evaluation_interval": updated.EvaluationInterval,
		"status":              updated.Status,
	}
	if _, err := c.DB().Model(&models.CloudBudget{}).
		Where("id = ? and org_id = ?", form.Id, c.OrgId).
		UpdateAttrs(attrs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	for key, value := range attrs {
		switch key {
		case "name":
			budget.Name = value.(string)
		case "description":
			budget.Description = value.(string)
		case "project_id":
			budget.ProjectId = value.(models.Id)
		case "env_id":
			budget.EnvId = value.(models.Id)
		case "cloud_account_id":
			budget.CloudAccountId = value.(models.Id)
		case "scope":
			budget.Scope = value.(string)
		case "provider":
			budget.Provider = value.(string)
		case "account_id":
			budget.AccountId = value.(string)
		case "region":
			budget.Region = value.(string)
		case "application":
			budget.Application = value.(string)
		case "business_line":
			budget.BusinessLine = value.(string)
		case "cost_center":
			budget.CostCenter = value.(string)
		case "owner":
			budget.Owner = value.(string)
		case "period":
			budget.Period = value.(string)
		case "currency":
			budget.Currency = value.(string)
		case "limit_amount":
			budget.LimitAmount = value.(float64)
		case "threshold_percent":
			budget.ThresholdPercent = value.(float64)
		case "evaluation_interval":
			budget.EvaluationInterval = value.(int)
		case "status":
			budget.Status = value.(string)
		}
	}
	if err := RefreshCloudCostRecords(c); err != nil {
		return nil, err
	}
	if err := evaluateSingleCloudBudget(c, &budget); err != nil {
		return nil, err
	}
	resp := cloudBudgetResp(c, budget)
	return &resp, nil
}

func DeleteCloudBudget(c *ctx.ServiceContext, form *forms.CloudBudgetParam) (interface{}, e.Error) {
	if _, err := c.DB().Where("id = ? and org_id = ?", form.Id, c.OrgId).Delete(&models.CloudBudget{}); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return nil, nil
}

func EvaluateCloudBudgets(c *ctx.ServiceContext, period string, currency string) e.Error {
	query := cloudBudgetBaseQuery(c).Where("status = ?", models.CloudBudgetStatusEnabled)
	if period != "" {
		query = query.Where("period = ?", period)
	}
	if currency != "" {
		query = query.Where("currency = ?", currency)
	}
	budgets := make([]models.CloudBudget, 0)
	if err := query.Find(&budgets); err != nil {
		return e.New(e.DBError, err)
	}
	for idx := range budgets {
		if err := evaluateSingleCloudBudget(c, &budgets[idx]); err != nil {
			return err
		}
	}
	return nil
}

func EvaluateDueCloudBudgetsForAllOrgs() (models.ResAttrs, e.Error) {
	groups := make([]cloudBudgetDueGroup, 0)
	now := time.Now()
	if err := db.Get().Model(&models.CloudBudget{}).
		Select("org_id, period, currency, count(*) as due_count").
		Where(`status = ? and period <> '' and currency <> '' and
			(last_evaluated_at is null or last_evaluated_at <= ? or
			 timestampdiff(second, last_evaluated_at, ?) >= greatest(evaluation_interval, 1))`,
			models.CloudBudgetStatusEnabled,
			cloudBudgetMinEvaluationTime(),
			now,
		).
		Group("org_id, period, currency").
		Order("org_id asc, period asc, currency asc").
		Scan(&groups); err != nil {
		return nil, e.New(e.DBError, err)
	}
	result := models.ResAttrs{
		"orgs":              0,
		"groups":            len(groups),
		"dueCount":          0,
		"totalCount":        int64(0),
		"evaluatedCount":    int64(0),
		"skippedCount":      int64(0),
		"thresholdCount":    int64(0),
		"exceededCount":     int64(0),
		"notificationCount": int64(0),
		"lockSkippedCount":  int64(0),
	}
	seenOrgIds := map[models.Id]bool{}
	for _, group := range groups {
		if group.OrgId == "" {
			continue
		}
		if !seenOrgIds[group.OrgId] {
			seenOrgIds[group.OrgId] = true
			result["orgs"] = result["orgs"].(int) + 1
		}
		result["dueCount"] = result["dueCount"].(int) + int(group.DueCount)
		locked, releaseLock, lockErr := cloudWebhookAcquireMysqlLock(
			cloudBudgetEvaluationLockName(group.OrgId, group.Period, group.Currency),
		)
		if lockErr != nil {
			return nil, e.New(e.DBError, lockErr)
		}
		if !locked {
			cloudBudgetAddEvaluateResultCount(result, "lockSkippedCount", 1)
			continue
		}
		orgResult, evalErr := func() (*resps.CloudBudgetEvaluateResp, e.Error) {
			defer releaseLock()
			workerCtx := &ctx.ServiceContext{
				UserId:   consts.SysUserId,
				OrgId:    group.OrgId,
				Email:    consts.DefaultSysEmail,
				Username: consts.DefaultSysName,
			}
			return EvaluateDueCloudBudgets(workerCtx, &forms.EvaluateCloudBudgetForm{
				Period:   group.Period,
				Currency: group.Currency,
			})
		}()
		if evalErr != nil {
			return nil, evalErr
		}
		cloudBudgetAddEvaluateResultCount(result, "totalCount", orgResult.TotalCount)
		cloudBudgetAddEvaluateResultCount(result, "evaluatedCount", orgResult.EvaluatedCount)
		cloudBudgetAddEvaluateResultCount(result, "skippedCount", orgResult.SkippedCount)
		cloudBudgetAddEvaluateResultCount(result, "thresholdCount", orgResult.ThresholdCount)
		cloudBudgetAddEvaluateResultCount(result, "exceededCount", orgResult.ExceededCount)
		cloudBudgetAddEvaluateResultCount(result, "notificationCount", orgResult.NotificationCount)
	}
	return result, nil
}

func StartCloudBudgetEvaluationWorker(serviceId string) {
	interval := cloudBudgetEvaluationWorkerInterval()
	logger := logs.Get().
		WithField("worker", "cloudBudgetEvaluation").
		WithField("serviceId", serviceId).
		WithField("interval", interval.String())
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if result, err := EvaluateDueCloudBudgetsForAllOrgs(); err != nil {
			logger.Warnf("evaluate due cloud budgets failed: %v", err)
		} else if cloudBudgetEvaluateResultCount(result, "evaluatedCount") > 0 ||
			cloudBudgetEvaluateResultCount(result, "notificationCount") > 0 ||
			cloudBudgetEvaluateResultCount(result, "lockSkippedCount") > 0 {
			logger.Infof("evaluate due cloud budgets result: %+v", result)
		}
		<-ticker.C
	}
}

func cloudBudgetEvaluationDue(budget models.CloudBudget, now time.Time) bool {
	last := time.Time(budget.LastEvaluatedAt)
	if last.IsZero() || last.Year() <= 1 {
		return true
	}
	return now.Sub(last) >= time.Duration(cloudBudgetEvaluationInterval(budget))*time.Second
}

func applyCloudBudgetSearch(query *db.Session, form *forms.SearchCloudBudgetForm) *db.Session {
	if form.Q != "" {
		q := "%" + strings.TrimSpace(form.Q) + "%"
		query = query.Where(`name like ? or description like ? or provider like ? or account_id like ? or
			application like ? or business_line like ? or cost_center like ? or owner like ?`,
			q, q, q, q, q, q, q, q)
	}
	if form.Scope != "" {
		query = query.Where("scope = ?", form.Scope)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	if form.Period != "" {
		query = query.Where("period = ?", form.Period)
	}
	if form.Currency != "" {
		query = query.Where("currency = ?", form.Currency)
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
	if form.Application != "" {
		query = query.Where("application = ?", form.Application)
	}
	if form.BusinessLine != "" {
		query = query.Where("business_line = ?", form.BusinessLine)
	}
	if form.CostCenter != "" {
		query = query.Where("cost_center = ?", form.CostCenter)
	}
	if form.Owner != "" {
		query = query.Where("owner = ?", form.Owner)
	}
	if form.ProjectId != "" {
		query = query.Where("project_id = ?", form.ProjectId)
	}
	if form.EnvId != "" {
		query = query.Where("env_id = ?", form.EnvId)
	}
	if form.ThresholdReached != nil {
		if *form.ThresholdReached {
			query = query.Where("limit_amount > 0 and last_usage_percent >= threshold_percent")
		} else {
			query = query.Where("limit_amount <= 0 or last_usage_percent < threshold_percent")
		}
	}
	return query
}

func cloudBudgetBaseQuery(c *ctx.ServiceContext) *db.Session {
	return c.DB().Model(&models.CloudBudget{}).Where("org_id = ?", c.OrgId)
}

func cloudBudgetFromCreateForm(c *ctx.ServiceContext, form *forms.CreateCloudBudgetForm) (*models.CloudBudget, e.Error) {
	budget := &models.CloudBudget{
		OrgId:              c.OrgId,
		Name:               strings.TrimSpace(form.Name),
		Description:        strings.TrimSpace(form.Description),
		ProjectId:          form.ProjectId,
		EnvId:              form.EnvId,
		CloudAccountId:     form.CloudAccountId,
		Scope:              firstNonEmpty(form.Scope, models.CloudBudgetScopeOrg),
		Provider:           strings.TrimSpace(form.Provider),
		AccountId:          strings.TrimSpace(form.AccountId),
		Region:             strings.TrimSpace(form.Region),
		Application:        strings.TrimSpace(form.Application),
		BusinessLine:       strings.TrimSpace(form.BusinessLine),
		CostCenter:         strings.TrimSpace(form.CostCenter),
		Owner:              strings.TrimSpace(form.Owner),
		Period:             firstNonEmpty(form.Period, time.Now().Format("2006-01")),
		Currency:           firstNonEmpty(strings.ToUpper(form.Currency), "CNY"),
		LimitAmount:        form.LimitAmount,
		ThresholdPercent:   form.ThresholdPercent,
		EvaluationInterval: form.EvaluationInterval,
		Status:             firstNonEmpty(form.Status, models.CloudBudgetStatusEnabled),
	}
	if budget.ThresholdPercent <= 0 {
		budget.ThresholdPercent = 80
	}
	if budget.EvaluationInterval <= 0 {
		budget.EvaluationInterval = 3600
	}
	if budget.EvaluationInterval > 86400 {
		budget.EvaluationInterval = 86400
	}
	if err := validateCloudBudget(budget); err != nil {
		return nil, err
	}
	return budget, nil
}

func validateCloudBudget(budget *models.CloudBudget) e.Error {
	if budget.Name == "" {
		return e.New(e.BadParam, fmt.Errorf("budget name is required"), http.StatusBadRequest)
	}
	if budget.LimitAmount <= 0 {
		return e.New(e.BadParam, fmt.Errorf("budget limit amount must be greater than 0"), http.StatusBadRequest)
	}
	if budget.ThresholdPercent <= 0 || budget.ThresholdPercent > 100 {
		return e.New(e.BadParam, fmt.Errorf("budget threshold percent must be between 0 and 100"), http.StatusBadRequest)
	}
	switch budget.Scope {
	case models.CloudBudgetScopeOrg:
	case models.CloudBudgetScopeProject:
		if budget.ProjectId == "" {
			return e.New(e.BadParam, fmt.Errorf("projectId is required for project budget"), http.StatusBadRequest)
		}
	case models.CloudBudgetScopeEnv:
		if budget.EnvId == "" {
			return e.New(e.BadParam, fmt.Errorf("envId is required for env budget"), http.StatusBadRequest)
		}
	case models.CloudBudgetScopeProvider:
		if budget.Provider == "" {
			return e.New(e.BadParam, fmt.Errorf("provider is required for provider budget"), http.StatusBadRequest)
		}
	case models.CloudBudgetScopeAccount:
		if budget.AccountId == "" && budget.CloudAccountId == "" {
			return e.New(e.BadParam, fmt.Errorf("accountId or cloudAccountId is required for account budget"), http.StatusBadRequest)
		}
	case models.CloudBudgetScopeApplication:
		if budget.Application == "" {
			return e.New(e.BadParam, fmt.Errorf("application is required for application budget"), http.StatusBadRequest)
		}
	case models.CloudBudgetScopeBusinessLine:
		if budget.BusinessLine == "" {
			return e.New(e.BadParam, fmt.Errorf("businessLine is required for business line budget"), http.StatusBadRequest)
		}
	case models.CloudBudgetScopeCostCenter:
		if budget.CostCenter == "" {
			return e.New(e.BadParam, fmt.Errorf("costCenter is required for cost center budget"), http.StatusBadRequest)
		}
	case models.CloudBudgetScopeOwner:
		if budget.Owner == "" {
			return e.New(e.BadParam, fmt.Errorf("owner is required for owner budget"), http.StatusBadRequest)
		}
	default:
		return e.New(e.BadParam, fmt.Errorf("invalid budget scope"), http.StatusBadRequest)
	}
	return nil
}

func evaluateSingleCloudBudget(c *ctx.ServiceContext, budget *models.CloudBudget) e.Error {
	_, err := evaluateSingleCloudBudgetResult(c, budget)
	return err
}

func evaluateSingleCloudBudgetResult(c *ctx.ServiceContext, budget *models.CloudBudget) (*cloudBudgetEvaluationResult, e.Error) {
	if budget == nil || budget.Status != models.CloudBudgetStatusEnabled {
		return &cloudBudgetEvaluationResult{}, nil
	}
	amount, err := cloudBudgetCurrentAmount(c, *budget)
	if err != nil {
		return nil, err
	}
	usage := cloudBudgetUsagePercent(amount, budget.LimitAmount)
	wasThresholdReached := budget.LimitAmount > 0 && budget.LastUsagePercent >= budget.ThresholdPercent
	thresholdReached := budget.LimitAmount > 0 && usage >= budget.ThresholdPercent
	budgetExceeded := budget.LimitAmount > 0 && amount >= budget.LimitAmount
	now := models.Time(time.Now())
	attrs := models.Attrs{
		"last_amount":        amount,
		"last_usage_percent": usage,
		"last_evaluated_at":  now,
	}
	if thresholdReached && !wasThresholdReached {
		attrs["last_exceeded_at"] = now
	}
	if _, err := c.DB().Model(&models.CloudBudget{}).
		Where("id = ? and org_id = ?", budget.Id, c.OrgId).
		UpdateAttrs(attrs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	budget.LastAmount = amount
	budget.LastUsagePercent = usage
	budget.LastEvaluatedAt = now
	notificationPushed := false
	if exceededAt, ok := attrs["last_exceeded_at"]; ok {
		budget.LastExceededAt = exceededAt.(models.Time)
		cloudBudgetThresholdEvent(c, budget, amount, usage)
		notificationPushed = true
	}
	return &cloudBudgetEvaluationResult{
		Amount:             amount,
		UsagePercent:       usage,
		ThresholdReached:   thresholdReached,
		BudgetExceeded:     budgetExceeded,
		NotificationPushed: notificationPushed,
	}, nil
}

func cloudBudgetCurrentAmount(c *ctx.ServiceContext, budget models.CloudBudget) (float64, e.Error) {
	query := cloudCostBaseQuery(c).Where("period = ? and currency = ?", budget.Period, budget.Currency)
	if budget.ProjectId != "" {
		query = query.Where("project_id = ?", budget.ProjectId)
	}
	if budget.EnvId != "" {
		query = query.Where("env_id = ?", budget.EnvId)
	}
	if budget.CloudAccountId != "" {
		query = query.Where("cloud_account_id = ?", budget.CloudAccountId)
	}
	if budget.Provider != "" {
		query = query.Where("provider = ?", budget.Provider)
	}
	if budget.AccountId != "" {
		query = query.Where("account_id = ?", budget.AccountId)
	}
	if budget.Region != "" {
		query = query.Where("region = ?", budget.Region)
	}
	if budget.Application != "" {
		query = query.Where("application = ?", budget.Application)
	}
	if budget.BusinessLine != "" {
		query = query.Where("business_line = ?", budget.BusinessLine)
	}
	if budget.CostCenter != "" {
		query = query.Where("cost_center = ?", budget.CostCenter)
	}
	if budget.Owner != "" {
		query = query.Where("owner = ?", budget.Owner)
	}
	row := cloudBudgetAmountRow{}
	if err := query.Select("coalesce(sum(amount), 0) as amount").Scan(&row); err != nil {
		return 0, e.New(e.DBError, err)
	}
	return row.Amount, nil
}

func cloudBudgetResp(c *ctx.ServiceContext, budget models.CloudBudget) resps.CloudBudgetResp {
	currentAmount := budget.LastAmount
	usagePercent := cloudBudgetUsagePercent(currentAmount, budget.LimitAmount)
	remainingAmount := budget.LimitAmount - currentAmount
	if remainingAmount < 0 {
		remainingAmount = 0
	}
	return resps.CloudBudgetResp{
		CloudBudget:      budget,
		ProjectName:      lookupName(c, &models.Project{}, budget.ProjectId),
		EnvName:          lookupName(c, &models.Env{}, budget.EnvId),
		CurrentAmount:    currentAmount,
		UsagePercent:     usagePercent,
		RemainingAmount:  remainingAmount,
		ThresholdReached: budget.LimitAmount > 0 && usagePercent >= budget.ThresholdPercent,
		BudgetExceeded:   budget.LimitAmount > 0 && currentAmount >= budget.LimitAmount,
	}
}

func cloudBudgetUsagePercent(amount float64, limit float64) float64 {
	if limit <= 0 {
		return 0
	}
	return amount / limit * 100
}

func cloudBudgetEvaluationInterval(budget models.CloudBudget) int {
	if budget.EvaluationInterval <= 0 {
		return 3600
	}
	return budget.EvaluationInterval
}

func cloudBudgetEvaluationWorkerInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("CLOUDIAC_BUDGET_EVALUATION_WORKER_INTERVAL_SECONDS"))
	if raw == "" {
		return cloudBudgetEvaluationWorkerDefaultInterval
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return cloudBudgetEvaluationWorkerDefaultInterval
	}
	interval := time.Duration(seconds) * time.Second
	if interval < cloudBudgetEvaluationWorkerMinInterval {
		return cloudBudgetEvaluationWorkerMinInterval
	}
	if interval > cloudBudgetEvaluationWorkerMaxInterval {
		return cloudBudgetEvaluationWorkerMaxInterval
	}
	return interval
}

func cloudBudgetMinEvaluationTime() time.Time {
	return time.Date(1971, 1, 1, 0, 0, 0, 0, time.UTC)
}

func cloudBudgetEvaluationLockName(orgId models.Id, period string, currency string) string {
	if orgId == "" {
		return "cloudiac:budget_evaluation:unknown"
	}
	return fmt.Sprintf("cloudiac:budget_evaluation:%s:%s:%s", orgId.String(), period, currency)
}

func cloudBudgetEvaluateResultCount(result models.ResAttrs, key string) int64 {
	value, ok := result[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func cloudBudgetAddEvaluateResultCount(result models.ResAttrs, key string, delta int64) {
	result[key] = cloudBudgetEvaluateResultCount(result, key) + delta
}

func cloudBudgetThresholdEvent(c *ctx.ServiceContext, budget *models.CloudBudget, amount float64, usage float64) {
	if budget == nil {
		return
	}
	level := models.CloudEventLevelWarning
	if amount >= budget.LimitAmount {
		level = models.CloudEventLevelError
	}
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:          budget.OrgId,
		ProjectId:      budget.ProjectId,
		EnvId:          budget.EnvId,
		CloudAccountId: budget.CloudAccountId,
		Source:         models.CloudEventSourceCost,
		EventType:      "cost.budget.threshold_exceeded",
		Level:          level,
		Status:         "threshold_exceeded",
		Provider:       budget.Provider,
		AccountId:      budget.AccountId,
		Region:         budget.Region,
		ResourceType:   "budget",
		ResourceId:     budget.Id.String(),
		ResourceName:   budget.Name,
		Title:          "成本预算超阈值",
		Message:        fmt.Sprintf("预算 %s 使用率 %.2f%%，已达到 %.2f%% 阈值", budget.Name, usage, budget.ThresholdPercent),
		Payload: models.ResAttrs{
			"budgetId":         budget.Id.String(),
			"budgetName":       budget.Name,
			"scope":            budget.Scope,
			"period":           budget.Period,
			"currency":         budget.Currency,
			"limitAmount":      budget.LimitAmount,
			"currentAmount":    amount,
			"usagePercent":     usage,
			"thresholdPercent": budget.ThresholdPercent,
			"application":      budget.Application,
			"businessLine":     budget.BusinessLine,
			"costCenter":       budget.CostCenter,
			"owner":            budget.Owner,
		},
	})
}
