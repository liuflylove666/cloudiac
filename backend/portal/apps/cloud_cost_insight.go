// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
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

const (
	cloudCostRuleUnmatchedRecord       = "unmatched_cost_record"
	cloudCostRuleUnownedRecord         = "unowned_cost_record"
	cloudCostRuleUnallocatedCostCenter = "unallocated_cost_center"
	cloudCostRuleHighCostResource      = "high_cost_resource"
	cloudCostRuleIdleComputeInstance   = "idle_compute_instance"
	cloudCostRuleUnattachedVolume      = "unattached_block_volume"
	cloudCostRuleUnusedPublicIP        = "unused_public_ip"
	cloudCostHighAmountThreshold       = 100
)

type cloudCostInsightAmountRow struct {
	Amount float64 `gorm:"column:amount"`
}

func CloudCostInsightSummary(c *ctx.ServiceContext, form *forms.CloudCostInsightSummaryForm) (*resps.CloudCostInsightSummaryResp, e.Error) {
	period := firstNonEmpty(form.Period, time.Now().Format("2006-01"))
	currency := firstNonEmpty(strings.ToUpper(form.Currency), "CNY")
	if err := RefreshCloudCostRecords(c); err != nil {
		return nil, err
	}
	if err := RefreshCloudCostInsights(c, period, currency); err != nil {
		return nil, err
	}

	resp := &resps.CloudCostInsightSummaryResp{
		Period:   period,
		Currency: currency,
	}
	base := func() *db.Session {
		return cloudCostInsightBaseQuery(c).Where("period = ? and currency = ?", period, currency)
	}
	var err error
	if resp.TotalCount, err = cloudOverviewCount(base()); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if resp.OpenCount, err = cloudOverviewCount(base().Where("status = ?", models.CloudCostInsightStatusOpen)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if resp.ResolvedCount, err = cloudOverviewCount(base().Where("status = ?", models.CloudCostInsightStatusResolved)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if resp.IgnoredCount, err = cloudOverviewCount(base().Where("status = ?", models.CloudCostInsightStatusIgnored)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if resp.HighCount, err = cloudOverviewCount(base().Where("status = ? and severity = ?", models.CloudCostInsightStatusOpen, models.CloudCostInsightSeverityHigh)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if resp.AnomalyCount, err = cloudOverviewCount(base().Where("status = ? and type = ?", models.CloudCostInsightStatusOpen, models.CloudCostInsightTypeAnomaly)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if resp.OptimizationCount, err = cloudOverviewCount(base().Where("status = ? and type = ?", models.CloudCostInsightStatusOpen, models.CloudCostInsightTypeOptimization)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	row := cloudCostInsightAmountRow{}
	if err := base().
		Where("status = ?", models.CloudCostInsightStatusOpen).
		Select("coalesce(sum(potential_savings), 0) as amount").
		Scan(&row); err != nil {
		return nil, e.New(e.DBError, err)
	}
	resp.PotentialSavings = row.Amount
	return resp, nil
}

func SearchCloudCostInsights(c *ctx.ServiceContext, form *forms.SearchCloudCostInsightForm) (interface{}, e.Error) {
	period := firstNonEmpty(form.Period, time.Now().Format("2006-01"))
	currency := firstNonEmpty(strings.ToUpper(form.Currency), "CNY")
	if err := RefreshCloudCostRecords(c); err != nil {
		return nil, err
	}
	if err := RefreshCloudCostInsights(c, period, currency); err != nil {
		return nil, err
	}
	form.Period = period
	form.Currency = currency

	query := applyCloudCostInsightSearch(cloudCostInsightBaseQuery(c), form)
	if form.SortField() == "" {
		query = query.Order("case severity when 'high' then 3 when 'medium' then 2 else 1 end desc").
			Order("potential_savings desc").
			Order("amount desc").
			Order("last_seen_at desc")
	} else {
		query = form.Order(query)
	}

	insights := make([]models.CloudCostInsight, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&insights); err != nil {
		return nil, e.New(e.DBError, err)
	}
	list := make([]resps.CloudCostInsightResp, 0, len(insights))
	for _, insight := range insights {
		list = append(list, cloudCostInsightResp(c, insight))
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func UpdateCloudCostInsightStatus(c *ctx.ServiceContext, form *forms.UpdateCloudCostInsightStatusForm) (*resps.CloudCostInsightResp, e.Error) {
	insight := models.CloudCostInsight{}
	if err := cloudCostInsightBaseQuery(c).Where("id = ?", form.Id).First(&insight); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	attrs := models.Attrs{
		"status": form.Status,
	}
	if form.Status == models.CloudCostInsightStatusResolved {
		attrs["resolved_at"] = models.Time(time.Now())
	} else if form.Status == models.CloudCostInsightStatusOpen {
		attrs["resolved_at"] = models.Time{}
	}
	if _, err := c.DB().Model(&models.CloudCostInsight{}).
		Where("id = ? and org_id = ?", form.Id, c.OrgId).
		UpdateAttrs(attrs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	insight.Status = form.Status
	if resolvedAt, ok := attrs["resolved_at"]; ok {
		insight.ResolvedAt = resolvedAt.(models.Time)
	}
	resp := cloudCostInsightResp(c, insight)
	return &resp, nil
}

func RefreshCloudCostInsights(c *ctx.ServiceContext, period string, currency string) e.Error {
	period = firstNonEmpty(period, time.Now().Format("2006-01"))
	currency = firstNonEmpty(strings.ToUpper(currency), "CNY")

	records := make([]models.CloudCostRecord, 0)
	if err := cloudCostBaseQuery(c).
		Where("period = ? and currency = ? and amount > 0", period, currency).
		Find(&records); err != nil {
		return e.New(e.DBError, err)
	}
	assets := make([]models.CmdbAsset, 0)
	if err := c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and cost > 0", c.OrgId).
		Find(&assets); err != nil {
		return e.New(e.DBError, err)
	}

	now := models.Time(time.Now())
	seen := make(map[string]bool)
	for _, record := range records {
		for _, insight := range cloudCostInsightsFromRecord(record) {
			insight.OrgId = c.OrgId
			insight.Period = period
			insight.Currency = currency
			insight.Fingerprint = cloudCostInsightFingerprint(c.OrgId, insight)
			seen[insight.Fingerprint] = true
			if err := upsertCloudCostInsight(c, insight, now); err != nil {
				return err
			}
		}
	}
	for _, asset := range assets {
		for _, insight := range cloudCostInsightsFromAsset(asset, period, currency) {
			insight.OrgId = c.OrgId
			insight.Period = period
			insight.Currency = currency
			insight.Fingerprint = cloudCostInsightFingerprint(c.OrgId, insight)
			seen[insight.Fingerprint] = true
			if err := upsertCloudCostInsight(c, insight, now); err != nil {
				return err
			}
		}
	}
	return resolveStaleCloudCostInsights(c, period, currency, seen, now)
}

func applyCloudCostInsightSearch(query *db.Session, form *forms.SearchCloudCostInsightForm) *db.Session {
	if form.Q != "" {
		q := "%" + strings.TrimSpace(form.Q) + "%"
		query = query.Where(`title like ? or description like ? or recommendation like ? or provider like ? or
			account_id like ? or resource_id like ? or resource_name like ? or rule_key like ? or evidence like ?`,
			q, q, q, q, q, q, q, q, q)
	}
	if form.Type != "" {
		query = query.Where("type = ?", form.Type)
	}
	if form.RuleKey != "" {
		query = query.Where("rule_key = ?", form.RuleKey)
	}
	if form.Severity != "" {
		query = query.Where("severity = ?", form.Severity)
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
	if form.ResourceType != "" {
		query = query.Where("resource_type = ?", form.ResourceType)
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

func cloudCostInsightBaseQuery(c *ctx.ServiceContext) *db.Session {
	return c.DB().Model(&models.CloudCostInsight{}).Where("org_id = ?", c.OrgId)
}

func cloudCostInsightsFromRecord(record models.CloudCostRecord) []models.CloudCostInsight {
	insights := make([]models.CloudCostInsight, 0, 4)
	if !record.MatchedAsset {
		insights = append(insights, cloudCostInsightFromRecord(
			record,
			models.CloudCostInsightTypeAnomaly,
			cloudCostRuleUnmatchedRecord,
			cloudCostInsightSeverityByAmount(record.Amount),
			"未匹配资产成本",
			fmt.Sprintf("账期 %s 存在未匹配到 CMDB/IaC 资产的成本记录，金额 %.2f %s。", record.Period, record.Amount, record.Currency),
			"补充资源 ID 映射、云账号同步或资产导入后重新刷新成本中心。",
			0,
		))
	}
	if strings.TrimSpace(record.Owner) == "" {
		insights = append(insights, cloudCostInsightFromRecord(
			record,
			models.CloudCostInsightTypeAnomaly,
			cloudCostRuleUnownedRecord,
			models.CloudCostInsightSeverityLow,
			"成本缺少负责人",
			fmt.Sprintf("资源 %s 的成本记录缺少 Owner，金额 %.2f %s。", firstNonEmpty(record.ResourceName, record.ResourceId), record.Amount, record.Currency),
			"在资产治理字段中补充 Owner，便于预算、分摊和整改闭环。",
			0,
		))
	}
	if strings.TrimSpace(record.CostCenter) == "" {
		insights = append(insights, cloudCostInsightFromRecord(
			record,
			models.CloudCostInsightTypeAnomaly,
			cloudCostRuleUnallocatedCostCenter,
			models.CloudCostInsightSeverityMedium,
			"成本中心未分摊",
			fmt.Sprintf("资源 %s 尚未归属成本中心，金额 %.2f %s。", firstNonEmpty(record.ResourceName, record.ResourceId), record.Amount, record.Currency),
			"补齐成本中心或业务线标签，确保费用可被准确分摊。",
			0,
		))
	}
	if record.Amount >= cloudCostHighAmountThreshold {
		insights = append(insights, cloudCostInsightFromRecord(
			record,
			models.CloudCostInsightTypeAnomaly,
			cloudCostRuleHighCostResource,
			models.CloudCostInsightSeverityHigh,
			"高成本资源",
			fmt.Sprintf("资源 %s 单账期成本 %.2f %s，超过 %.2f %s 阈值。", firstNonEmpty(record.ResourceName, record.ResourceId), record.Amount, record.Currency, float64(cloudCostHighAmountThreshold), record.Currency),
			"复核资源规格、计费模式和使用率，评估预留实例、节省计划或规格调整。",
			0,
		))
	}
	return insights
}

func cloudCostInsightFromRecord(record models.CloudCostRecord, insightType string, ruleKey string, severity string, title string, description string, recommendation string, savings float64) models.CloudCostInsight {
	return models.CloudCostInsight{
		ProjectId:        record.ProjectId,
		EnvId:            record.EnvId,
		AssetId:          record.AssetId,
		CloudAccountId:   record.CloudAccountId,
		Type:             insightType,
		RuleKey:          ruleKey,
		Severity:         severity,
		Status:           models.CloudCostInsightStatusOpen,
		Provider:         record.Provider,
		AccountId:        record.AccountId,
		Region:           record.Region,
		ResourceType:     record.ResourceType,
		ResourceId:       record.ResourceId,
		ResourceName:     firstNonEmpty(record.ResourceName, record.ResourceId),
		Period:           record.Period,
		Currency:         record.Currency,
		Amount:           record.Amount,
		PotentialSavings: savings,
		Title:            title,
		Description:      description,
		Recommendation:   recommendation,
		Evidence: models.ResAttrs{
			"recordId":     record.Id.String(),
			"source":       record.Source,
			"sourceId":     record.SourceId,
			"matchedAsset": record.MatchedAsset,
			"costCenter":   record.CostCenter,
			"owner":        record.Owner,
			"application":  record.Application,
			"businessLine": record.BusinessLine,
		},
	}
}

func cloudCostInsightsFromAsset(asset models.CmdbAsset, period string, currency string) []models.CloudCostInsight {
	insights := make([]models.CloudCostInsight, 0, 3)
	state := cloudCostAssetState(asset)
	switch asset.AssetType {
	case models.CmdbAssetTypeComputeInstance:
		if cloudCostAssetStateIn(state, "stopped", "stopping", "terminated", "inactive", "shutdown", "stopped_state") {
			insights = append(insights, cloudCostInsightFromAsset(
				asset,
				period,
				currency,
				models.CloudCostInsightTypeOptimization,
				cloudCostRuleIdleComputeInstance,
				cloudCostInsightSeverityByAmount(asset.Cost),
				"闲置计算实例",
				fmt.Sprintf("计算实例 %s 当前状态为 %s，但仍有 %.2f %s 成本。", firstNonEmpty(asset.Name, asset.NativeId, asset.Id.String()), firstNonEmpty(state, "unknown"), asset.Cost, currency),
				"确认业务影响后关停、释放或调整实例规格，并把处理结果回写到 CMDB。",
				asset.Cost,
				models.ResAttrs{"state": state},
			))
		}
	case models.CmdbAssetTypeBlockVolume:
		if cloudCostAssetStateIn(state, "available", "unattached", "detached") || !cloudCostAssetHasAnyAssociation(asset, "instanceId", "instance_id", "attachedInstanceId", "attachmentId", "attachments") {
			insights = append(insights, cloudCostInsightFromAsset(
				asset,
				period,
				currency,
				models.CloudCostInsightTypeOptimization,
				cloudCostRuleUnattachedVolume,
				models.CloudCostInsightSeverityMedium,
				"未挂载磁盘",
				fmt.Sprintf("块存储 %s 未发现挂载关系，但仍有 %.2f %s 成本。", firstNonEmpty(asset.Name, asset.NativeId, asset.Id.String()), asset.Cost, currency),
				"确认无数据保留要求后释放磁盘，或补齐挂载关系和负责人。",
				asset.Cost,
				models.ResAttrs{"state": state},
			))
		}
	case models.CmdbAssetTypePublicIP:
		if cloudCostAssetStateIn(state, "available", "unassigned", "unassociated", "idle") || !cloudCostAssetHasAnyAssociation(asset, "associationId", "networkInterfaceId", "instanceId", "privateIpAddress", "privateIp", "resourceId") {
			insights = append(insights, cloudCostInsightFromAsset(
				asset,
				period,
				currency,
				models.CloudCostInsightTypeOptimization,
				cloudCostRuleUnusedPublicIP,
				models.CloudCostInsightSeverityMedium,
				"未使用公网 IP",
				fmt.Sprintf("公网 IP %s 未发现绑定资源，但仍有 %.2f %s 成本。", firstNonEmpty(asset.PublicIp, asset.Name, asset.NativeId, asset.Id.String()), asset.Cost, currency),
				"确认未被业务使用后释放公网 IP，或补齐绑定关系和应用归属。",
				asset.Cost,
				models.ResAttrs{"state": state, "publicIp": asset.PublicIp},
			))
		}
	}
	return insights
}

func cloudCostInsightFromAsset(asset models.CmdbAsset, period string, currency string, insightType string, ruleKey string, severity string, title string, description string, recommendation string, savings float64, extra models.ResAttrs) models.CloudCostInsight {
	evidence := models.ResAttrs{
		"assetId":        asset.Id.String(),
		"assetType":      asset.AssetType,
		"nativeType":     asset.NativeType,
		"nativeId":       asset.NativeId,
		"status":         asset.Status,
		"lifecycle":      asset.Lifecycle,
		"costCenter":     asset.CostCenter,
		"owner":          asset.Owner,
		"application":    asset.Application,
		"businessLine":   asset.BusinessLine,
		"lastSyncAt":     asset.LastSyncAt,
		"cloudAccountId": asset.CloudAccountId.String(),
	}
	for key, value := range extra {
		evidence[key] = value
	}
	return models.CloudCostInsight{
		ProjectId:        asset.ProjectId,
		EnvId:            asset.EnvId,
		AssetId:          asset.Id,
		CloudAccountId:   asset.CloudAccountId,
		Type:             insightType,
		RuleKey:          ruleKey,
		Severity:         severity,
		Status:           models.CloudCostInsightStatusOpen,
		Provider:         asset.Provider,
		AccountId:        asset.AccountId,
		Region:           asset.Region,
		ResourceType:     asset.AssetType,
		ResourceId:       firstNonEmpty(asset.NativeId, asset.IacAddress, asset.Id.String()),
		ResourceName:     firstNonEmpty(asset.Name, asset.NativeId, asset.Id.String()),
		Period:           period,
		Currency:         currency,
		Amount:           asset.Cost,
		PotentialSavings: savings,
		Title:            title,
		Description:      description,
		Recommendation:   recommendation,
		Evidence:         evidence,
	}
}

func upsertCloudCostInsight(c *ctx.ServiceContext, insight models.CloudCostInsight, now models.Time) e.Error {
	insight.FirstSeenAt = now
	insight.LastSeenAt = now
	existing := models.CloudCostInsight{}
	err := cloudCostInsightBaseQuery(c).
		Where("fingerprint = ?", insight.Fingerprint).
		First(&existing)
	if err != nil && !e.IsRecordNotFound(err) {
		return e.New(e.DBError, err)
	}
	if e.IsRecordNotFound(err) {
		insight.Id = models.NewId("cci")
		if err := models.Create(c.DB(), &insight); err != nil {
			duplicate := models.CloudCostInsight{}
			if findErr := cloudCostInsightBaseQuery(c).
				Where("fingerprint = ?", insight.Fingerprint).
				First(&duplicate); findErr == nil {
				return updateExistingCloudCostInsight(c, duplicate, insight, now)
			}
			return e.New(e.DBError, err)
		}
		cloudCostInsightDetectedEvent(c, insight)
		return nil
	}
	return updateExistingCloudCostInsight(c, existing, insight, now)
}

func updateExistingCloudCostInsight(c *ctx.ServiceContext, existing models.CloudCostInsight, insight models.CloudCostInsight, now models.Time) e.Error {
	attrs := cloudCostInsightUpdateAttrs(insight, now)
	if existing.Status == "" {
		attrs["status"] = models.CloudCostInsightStatusOpen
		attrs["resolved_at"] = models.Time{}
	}
	if _, err := c.DB().Model(&models.CloudCostInsight{}).
		Where("id = ? and org_id = ?", existing.Id, c.OrgId).
		UpdateAttrs(attrs); err != nil {
		return e.New(e.DBError, err)
	}
	if existing.Status == "" {
		insight.Id = existing.Id
		cloudCostInsightDetectedEvent(c, insight)
	}
	return nil
}

func cloudCostInsightUpdateAttrs(insight models.CloudCostInsight, now models.Time) models.Attrs {
	return models.Attrs{
		"project_id":        insight.ProjectId,
		"env_id":            insight.EnvId,
		"asset_id":          insight.AssetId,
		"cloud_account_id":  insight.CloudAccountId,
		"type":              insight.Type,
		"rule_key":          insight.RuleKey,
		"severity":          insight.Severity,
		"provider":          insight.Provider,
		"account_id":        insight.AccountId,
		"region":            insight.Region,
		"resource_type":     insight.ResourceType,
		"resource_id":       insight.ResourceId,
		"resource_name":     insight.ResourceName,
		"period":            insight.Period,
		"currency":          insight.Currency,
		"amount":            insight.Amount,
		"potential_savings": insight.PotentialSavings,
		"title":             insight.Title,
		"description":       insight.Description,
		"recommendation":    insight.Recommendation,
		"evidence":          insight.Evidence,
		"last_seen_at":      now,
	}
}

func resolveStaleCloudCostInsights(c *ctx.ServiceContext, period string, currency string, seen map[string]bool, now models.Time) e.Error {
	query := cloudCostInsightBaseQuery(c).
		Where("period = ? and currency = ?", period, currency).
		Where("status = ?", models.CloudCostInsightStatusOpen).
		Where("last_seen_at < ?", now)
	if len(seen) > 0 {
		fingerprints := make([]string, 0, len(seen))
		for fingerprint := range seen {
			fingerprints = append(fingerprints, fingerprint)
		}
		query = query.Where("fingerprint not in (?)", fingerprints)
	}
	if _, err := query.UpdateAttrs(models.Attrs{
		"status":      models.CloudCostInsightStatusResolved,
		"resolved_at": now,
	}); err != nil {
		return e.New(e.DBError, err)
	}
	return nil
}

func cloudCostInsightResp(c *ctx.ServiceContext, insight models.CloudCostInsight) resps.CloudCostInsightResp {
	assetName := insight.ResourceName
	if assetName == "" && insight.AssetId != "" {
		assetName = lookupName(c, &models.CmdbAsset{}, insight.AssetId)
	}
	return resps.CloudCostInsightResp{
		CloudCostInsight: insight,
		ProjectName:      lookupName(c, &models.Project{}, insight.ProjectId),
		EnvName:          lookupName(c, &models.Env{}, insight.EnvId),
		AssetName:        assetName,
	}
}

func cloudCostInsightDetectedEvent(c *ctx.ServiceContext, insight models.CloudCostInsight) {
	level := models.CloudEventLevelInfo
	if insight.Severity == models.CloudCostInsightSeverityHigh {
		level = models.CloudEventLevelWarning
	}
	recordCloudEventBestEffort(c, models.CloudEvent{
		OrgId:          insight.OrgId,
		ProjectId:      insight.ProjectId,
		EnvId:          insight.EnvId,
		AssetId:        insight.AssetId,
		CloudAccountId: insight.CloudAccountId,
		Source:         models.CloudEventSourceCost,
		EventType:      "cost.insight.detected",
		Level:          level,
		Status:         insight.Status,
		Provider:       insight.Provider,
		AccountId:      insight.AccountId,
		Region:         insight.Region,
		ResourceType:   insight.ResourceType,
		ResourceId:     insight.ResourceId,
		ResourceName:   insight.ResourceName,
		Title:          insight.Title,
		Message:        insight.Description,
		Payload: models.ResAttrs{
			"insightId":        insight.Id.String(),
			"type":             insight.Type,
			"ruleKey":          insight.RuleKey,
			"severity":         insight.Severity,
			"period":           insight.Period,
			"currency":         insight.Currency,
			"amount":           insight.Amount,
			"potentialSavings": insight.PotentialSavings,
			"recommendation":   insight.Recommendation,
		},
	})
}

func cloudCostInsightFingerprint(orgId models.Id, insight models.CloudCostInsight) string {
	seed := strings.Join([]string{
		orgId.String(),
		insight.Type,
		insight.RuleKey,
		insight.Period,
		insight.Currency,
		insight.Provider,
		insight.AccountId,
		insight.Region,
		insight.ResourceType,
		insight.ResourceId,
		insight.AssetId.String(),
	}, "\x00")
	sum := sha1.Sum([]byte(seed))
	return hex.EncodeToString(sum[:])
}

func cloudCostInsightSeverityByAmount(amount float64) string {
	if amount >= cloudCostHighAmountThreshold {
		return models.CloudCostInsightSeverityHigh
	}
	if amount >= 20 {
		return models.CloudCostInsightSeverityMedium
	}
	return models.CloudCostInsightSeverityLow
}

func cloudCostAssetState(asset models.CmdbAsset) string {
	return strings.ToLower(strings.TrimSpace(firstNonEmpty(
		asset.Status,
		asset.Lifecycle,
		cloudCostAttrString(asset.Attributes, "state", "status", "lifecycle"),
		cloudCostAttrString(asset.RawData, "state", "status", "lifecycle"),
	)))
}

func cloudCostAssetStateIn(state string, values ...string) bool {
	for _, value := range values {
		if state == value {
			return true
		}
	}
	return false
}

func cloudCostAssetHasAnyAssociation(asset models.CmdbAsset, keys ...string) bool {
	if strings.TrimSpace(asset.PrivateIp) != "" {
		return true
	}
	for _, key := range keys {
		if value := cloudCostAttrString(asset.Attributes, key); value != "" && value != "[]" {
			return true
		}
		if value := cloudCostAttrString(asset.RawData, key); value != "" && value != "[]" {
			return true
		}
	}
	return false
}

func cloudCostAttrString(attrs models.ResAttrs, keys ...string) string {
	for _, key := range keys {
		if value, ok := cloudCostAttrValue(attrs, key); ok {
			text := strings.TrimSpace(fmt.Sprintf("%v", value))
			if text != "" && text != "<nil>" && text != "[]" {
				return text
			}
		}
	}
	return ""
}

func cloudCostAttrValue(attrs models.ResAttrs, key string) (interface{}, bool) {
	if attrs == nil {
		return nil, false
	}
	if value, ok := attrs[key]; ok {
		return value, true
	}
	for attrKey, value := range attrs {
		if strings.EqualFold(attrKey, key) {
			return value, true
		}
	}
	return nil, false
}
