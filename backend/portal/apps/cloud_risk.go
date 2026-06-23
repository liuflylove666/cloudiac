// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
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

type cloudRiskCandidate struct {
	Asset           models.CmdbAsset
	AssetId         models.Id
	ProjectId       models.Id
	EnvId           models.Id
	CloudAccountId  models.Id
	Provider        string
	AccountId       string
	Region          string
	ResourceType    string
	ResourceId      string
	ResourceName    string
	Source          string
	RuleKey         string
	RuleName        string
	RiskLevel       string
	Evidence        models.ResAttrs
	Recommendation  string
	FingerprintSeed []string
}

type cloudRiskDriftRow struct {
	ResId          models.Id `gorm:"column:res_id"`
	ProjectId      models.Id `gorm:"column:project_id"`
	EnvId          models.Id `gorm:"column:env_id"`
	AssetId        models.Id `gorm:"column:asset_id"`
	CloudAccountId models.Id `gorm:"column:cloud_account_id"`
	Provider       string    `gorm:"column:provider"`
	AccountId      string    `gorm:"column:account_id"`
	Region         string    `gorm:"column:region"`
	ResourceType   string    `gorm:"column:resource_type"`
	ResourceId     string    `gorm:"column:resource_id"`
	ResourceName   string    `gorm:"column:resource_name"`
	Address        string    `gorm:"column:address"`
	DriftDetail    string    `gorm:"column:drift_detail"`
}

func SearchCloudRisks(c *ctx.ServiceContext, form *forms.SearchCloudRiskForm) (*resps.CloudRiskListResp, e.Error) {
	if err := RefreshCloudRiskFindings(c); err != nil {
		return nil, err
	}

	query := c.DB().Model(&models.CloudRiskFinding{}).Where("org_id = ?", c.OrgId)
	query = applyCloudRiskSearch(query, form)
	if form.SortField() == "" {
		query = query.Order("case risk_level when 'critical' then 4 when 'high' then 3 when 'medium' then 2 else 1 end desc").
			Order("last_seen_at desc").
			Order("created_at desc")
	} else {
		query = form.Order(query)
	}

	findings := make([]models.CloudRiskFinding, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&findings); err != nil {
		return nil, e.New(e.DBError, err)
	}

	list := make([]resps.CloudRiskFindingResp, 0, len(findings))
	for _, finding := range findings {
		list = append(list, cloudRiskFindingResp(c, finding))
	}
	summary, err := cloudRiskSummary(c)
	if err != nil {
		return nil, err
	}
	return &resps.CloudRiskListResp{
		Summary:  summary,
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func UpdateCloudRiskStatus(c *ctx.ServiceContext, form *forms.UpdateCloudRiskStatusForm) (*resps.CloudRiskFindingResp, e.Error) {
	finding, err := getCloudRiskFinding(c, form.Id)
	if err != nil {
		return nil, err
	}
	now := models.Time(time.Now())
	attrs := models.Attrs{
		"status":             form.Status,
		"suppression_reason": "",
	}
	if form.Status == models.CloudRiskStatusResolved {
		attrs["resolved_at"] = now
	} else {
		attrs["resolved_at"] = models.Time{}
		attrs["suppressed_until"] = models.Time{}
	}
	if form.Comment != "" {
		attrs["evidence"] = mergeCloudRiskEvidence(finding.Evidence, models.ResAttrs{
			"lastStatusComment": form.Comment,
			"lastStatusBy":      c.UserId.String(),
			"lastStatusAt":      time.Now().Format(time.RFC3339),
		})
	}
	if _, dbErr := c.DB().Model(&models.CloudRiskFinding{}).
		Where("id = ? and org_id = ?", finding.Id, c.OrgId).
		UpdateAttrs(attrs); dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}
	updated, err := getCloudRiskFinding(c, form.Id)
	if err != nil {
		return nil, err
	}
	cloudRiskEvent(c, updated, "risk.status_updated", "风险状态更新", fmt.Sprintf("风险状态更新为 %s", form.Status), models.ResAttrs{
		"previousStatus": finding.Status,
		"status":         form.Status,
		"comment":        form.Comment,
	})
	resp := cloudRiskFindingResp(c, *updated)
	return &resp, nil
}

func SuppressCloudRisk(c *ctx.ServiceContext, form *forms.SuppressCloudRiskForm) (*resps.CloudRiskFindingResp, e.Error) {
	finding, err := getCloudRiskFinding(c, form.Id)
	if err != nil {
		return nil, err
	}
	suppressedUntil, parseErr := models.Time{}.Parse(form.SuppressedUntil)
	if parseErr != nil {
		return nil, e.New(e.BadParam, fmt.Errorf("风险例外到期时间格式不正确: %v", parseErr))
	}
	if time.Time(suppressedUntil).Before(time.Now()) {
		return nil, e.New(e.BadParam, fmt.Errorf("风险例外到期时间必须晚于当前时间"))
	}
	evidence := mergeCloudRiskEvidence(finding.Evidence, models.ResAttrs{
		"lastSuppressedBy": c.UserId.String(),
		"lastSuppressedAt": time.Now().Format(time.RFC3339),
	})
	if _, dbErr := c.DB().Model(&models.CloudRiskFinding{}).
		Where("id = ? and org_id = ?", finding.Id, c.OrgId).
		UpdateAttrs(models.Attrs{
			"status":             models.CloudRiskStatusSuppressed,
			"suppressed_until":   suppressedUntil,
			"suppression_reason": form.Reason,
			"resolved_at":        models.Time{},
			"evidence":           evidence,
		}); dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}
	updated, err := getCloudRiskFinding(c, form.Id)
	if err != nil {
		return nil, err
	}
	cloudRiskEvent(c, updated, "risk.suppressed", "风险例外", form.Reason, models.ResAttrs{
		"suppressedUntil": form.SuppressedUntil,
		"reason":          form.Reason,
	})
	resp := cloudRiskFindingResp(c, *updated)
	return &resp, nil
}

func CreateCloudRiskRemediationTicket(c *ctx.ServiceContext, form *forms.CreateCloudRiskRemediationTicketForm) (*resps.CloudItsmTicketResp, e.Error) {
	finding, err := getCloudRiskFinding(c, form.Id)
	if err != nil {
		return nil, err
	}
	if finding.Status == models.CloudRiskStatusResolved {
		return nil, e.New(e.BadParam, fmt.Errorf("风险发现 %s 已解决，无需创建整改工单", finding.Id))
	}
	title := strings.TrimSpace(form.Title)
	if title == "" {
		title = cloudRiskRemediationTitle(*finding)
	}
	description := strings.TrimSpace(form.Description)
	if description == "" {
		description = cloudRiskRemediationDescription(*finding)
	}
	priority := strings.TrimSpace(form.Priority)
	if priority == "" {
		priority = cloudItsmPriorityFromRisk(finding.RiskLevel)
	}
	ticket, ticketErr := CreateCloudItsmSelfServiceTicket(c, &forms.CreateCloudItsmSelfServiceTicketForm{
		ConnectorId: form.ConnectorId,
		RequestType: models.CloudOperationActionRiskRemediation,
		Title:       title,
		Description: description,
		Priority:    priority,
		ProjectId:   finding.ProjectId,
		EnvId:       finding.EnvId,
		Params:      cloudRiskRemediationParams(*finding, form.Params),
		DryRun:      form.DryRun,
	})
	if ticketErr != nil {
		return nil, ticketErr
	}
	if form.DryRun {
		return ticket, nil
	}

	evidence := mergeCloudRiskEvidence(finding.Evidence, models.ResAttrs{
		"lastRemediationTicketId":     ticket.Id.String(),
		"lastRemediationOperationId":  ticket.OperationId.String(),
		"lastRemediationConnectorId":  ticket.ConnectorId.String(),
		"lastRemediationConnector":    ticket.ConnectorName,
		"lastRemediationTicketStatus": ticket.Status,
		"lastRemediationAt":           time.Now().Format(time.RFC3339),
		"lastRemediationBy":           c.UserId.String(),
	})
	if _, dbErr := c.DB().Model(&models.CloudRiskFinding{}).
		Where("id = ? and org_id = ?", finding.Id, c.OrgId).
		UpdateAttrs(models.Attrs{
			"status":             models.CloudRiskStatusInProgress,
			"suppression_reason": "",
			"resolved_at":        models.Time{},
			"evidence":           evidence,
		}); dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}
	updated, err := getCloudRiskFinding(c, finding.Id)
	if err != nil {
		return nil, err
	}
	cloudRiskEvent(c, updated, "risk.remediation_ticket_created", "风险整改工单已创建", title, models.ResAttrs{
		"previousStatus": finding.Status,
		"status":         models.CloudRiskStatusInProgress,
		"ticketId":       ticket.Id.String(),
		"ticketStatus":   ticket.Status,
		"operationId":    ticket.OperationId.String(),
		"connectorId":    ticket.ConnectorId.String(),
		"connectorName":  ticket.ConnectorName,
	})
	return ticket, nil
}

func AdoptCloudRiskRecommendation(c *ctx.ServiceContext, form *forms.AdoptCloudRiskRecommendationForm) (*resps.CloudRiskFindingResp, e.Error) {
	finding, err := getCloudRiskFinding(c, form.Id)
	if err != nil {
		return nil, err
	}
	evidence, adoption, adoptErr := cloudRiskRecommendationAdoptionEvidence(finding.Evidence, *form, c.UserId, time.Now())
	if adoptErr != nil {
		return nil, adoptErr
	}
	if _, dbErr := c.DB().Model(&models.CloudRiskFinding{}).
		Where("id = ? and org_id = ?", finding.Id, c.OrgId).
		UpdateAttrs(models.Attrs{"evidence": evidence}); dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}
	updated, err := getCloudRiskFinding(c, finding.Id)
	if err != nil {
		return nil, err
	}
	cloudRiskEvent(c, updated, "risk.recommendation_adopted", "风险推荐已采纳", fmt.Sprintf("已采纳推荐：%s", cloudSyncPolicyAttrString(adoption["title"])), adoption)
	resp := cloudRiskFindingResp(c, *updated)
	return &resp, nil
}

func RefreshCloudRiskFindings(c *ctx.ServiceContext) e.Error {
	if _, err := BackfillCmdbAssetsFromIac(c); err != nil {
		return err
	}
	if err := RefreshCmdbAssetGovernanceFields(c); err != nil {
		return err
	}

	now := models.Time(time.Now())
	candidates, err := cloudRiskCandidates(c)
	if err != nil {
		return err
	}
	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		fingerprint := cloudRiskFingerprint(c.OrgId, candidate)
		if fingerprint == "" {
			continue
		}
		seen[fingerprint] = true
		if err := upsertCloudRiskFinding(c, candidate, fingerprint, now); err != nil {
			return err
		}
	}
	return resolveStaleCloudRiskFindings(c, seen, now)
}

func applyCloudRiskSearch(query *db.Session, form *forms.SearchCloudRiskForm) *db.Session {
	if form.Q != "" {
		q := "%" + strings.TrimSpace(form.Q) + "%"
		query = query.Where(`id like ? or resource_name like ? or resource_id like ? or rule_name like ? or
			provider like ? or account_id like ? or region like ? or evidence like ?`,
			q, q, q, q, q, q, q, q)
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
	if form.Source != "" {
		query = query.Where("source = ?", form.Source)
	}
	if form.RuleKey != "" {
		query = query.Where("rule_key = ?", form.RuleKey)
	}
	if form.RiskLevel != "" {
		query = query.Where("risk_level = ?", form.RiskLevel)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
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

func cloudRiskCandidates(c *ctx.ServiceContext) ([]cloudRiskCandidate, e.Error) {
	assets := make([]models.CmdbAsset, 0)
	if err := buildCmdbAssetQuery(c).Scan(&assets); err != nil {
		return nil, e.New(e.DBError, err)
	}

	candidates := make([]cloudRiskCandidate, 0)
	for _, asset := range assets {
		candidates = append(candidates, cloudRiskCandidatesFromAsset(asset)...)
	}
	drifts, err := cloudRiskDriftCandidates(c)
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, drifts...)
	return candidates, nil
}

func cloudRiskCandidatesFromAsset(asset models.CmdbAsset) []cloudRiskCandidate {
	base := cloudRiskCandidateForAsset(asset)
	candidates := make([]cloudRiskCandidate, 0)
	if asset.ManagedBy == models.CmdbManagedByCloudOnly {
		item := base
		item.Source = models.CloudRiskSourceCMDB
		item.RuleKey = "unmanaged_cloud_asset"
		item.RuleName = "未纳管云采集资产"
		item.RiskLevel = models.CloudOperationRiskMedium
		item.Evidence = models.ResAttrs{
			"managedBy": asset.ManagedBy,
			"source":    asset.Source,
			"nativeId":  asset.NativeId,
		}
		item.Recommendation = "确认资源归属并绑定项目/环境，或通过 IaC 纳管该云资源。"
		item.FingerprintSeed = []string{asset.ManagedBy}
		candidates = append(candidates, item)
	}
	if strings.TrimSpace(asset.Owner) == "" {
		item := base
		item.Source = models.CloudRiskSourceCMDB
		item.RuleKey = "unowned_cloud_asset"
		item.RuleName = "云资产缺少负责人"
		item.RiskLevel = models.CloudOperationRiskMedium
		item.Evidence = models.ResAttrs{
			"owner":        asset.Owner,
			"application":  asset.Application,
			"businessLine": asset.BusinessLine,
		}
		item.Recommendation = "在云资产治理中补充 owner、应用和业务线，确保责任边界清晰。"
		candidates = append(candidates, item)
	}
	if asset.ComplianceRisk == models.CloudOperationRiskHigh || asset.ComplianceRisk == models.CloudOperationRiskCritical {
		item := base
		item.Source = models.CloudRiskSourcePolicy
		item.RuleKey = "cmdb_compliance_risk_" + asset.ComplianceRisk
		item.RuleName = "CMDB 高风险合规标记"
		item.RiskLevel = asset.ComplianceRisk
		item.Evidence = models.ResAttrs{
			"complianceRisk": asset.ComplianceRisk,
			"riskScore":      asset.RiskScore,
		}
		item.Recommendation = "查看资产合规风险来源，完成策略整改或提交有效例外。"
		item.FingerprintSeed = []string{asset.ComplianceRisk}
		candidates = append(candidates, item)
	}
	for index, rule := range cloudAssetSecurityRulesFromAsset(&asset) {
		if !rule.PublicExposure {
			continue
		}
		item := base
		item.Source = models.CloudRiskSourceCloudConfig
		item.RuleKey = "public_" + rule.Direction + "_security_rule"
		item.RuleName = cloudRiskPublicRuleName(rule)
		item.RiskLevel = cloudRiskPublicRuleLevel(rule)
		item.Evidence = models.ResAttrs{
			"direction":      rule.Direction,
			"protocol":       rule.Protocol,
			"source":         rule.Source,
			"destination":    rule.Destination,
			"portRange":      rule.PortRange,
			"description":    rule.Description,
			"raw":            rule.Raw,
			"publicExposure": rule.PublicExposure,
		}
		item.Recommendation = "收敛安全组/安全列表公网规则范围，优先改为业务网段、堡垒机或负载均衡入口。"
		item.FingerprintSeed = []string{
			fmt.Sprintf("%d", index),
			rule.Direction,
			rule.Protocol,
			rule.PortRange,
			rule.Source,
			rule.Destination,
		}
		candidates = append(candidates, item)
	}
	candidates = append(candidates, cloudRiskS3ConfigCandidates(base, asset)...)
	return candidates
}

func cloudRiskCandidateForAsset(asset models.CmdbAsset) cloudRiskCandidate {
	return cloudRiskCandidate{
		Asset:          asset,
		AssetId:        asset.Id,
		ProjectId:      asset.ProjectId,
		EnvId:          asset.EnvId,
		CloudAccountId: asset.CloudAccountId,
		Provider:       asset.Provider,
		AccountId:      asset.AccountId,
		Region:         asset.Region,
		ResourceType:   firstNonEmpty(asset.AssetType, asset.NativeType),
		ResourceId:     firstNonEmpty(asset.NativeId, asset.IacAddress, asset.Id.String()),
		ResourceName:   firstNonEmpty(asset.Name, asset.NativeId, asset.IacAddress, asset.Id.String()),
	}
}

func cloudRiskDriftCandidates(c *ctx.ServiceContext) ([]cloudRiskCandidate, e.Error) {
	rows := make([]cloudRiskDriftRow, 0)
	if err := c.DB().Table("iac_resource as r").
		Select(`r.id as res_id,
			r.project_id,
			r.env_id,
			ca.id as asset_id,
			ca.cloud_account_id,
			coalesce(nullif(ca.provider, ''), r.provider) as provider,
			ca.account_id,
			ca.region,
			coalesce(nullif(ca.asset_type, ''), r.type) as resource_type,
			coalesce(nullif(ca.native_id, ''), r.res_id, r.id) as resource_id,
			coalesce(nullif(ca.name, ''), r.name, r.address) as resource_name,
			r.address,
			rd.drift_detail`).
		Joins("join iac_resource_drift as rd on rd.res_id = r.id").
		Joins("left join iac_cmdb_asset as ca on ca.iac_resource_id = r.id and ca.org_id = r.org_id").
		Where("r.org_id = ?", c.OrgId).
		Scan(&rows); err != nil {
		return nil, e.New(e.DBError, err)
	}

	candidates := make([]cloudRiskCandidate, 0, len(rows))
	for _, row := range rows {
		resourceId := firstNonEmpty(row.ResourceId, row.ResId.String(), row.Address)
		item := cloudRiskCandidate{
			AssetId:        row.AssetId,
			ProjectId:      row.ProjectId,
			EnvId:          row.EnvId,
			CloudAccountId: row.CloudAccountId,
			Provider:       row.Provider,
			AccountId:      row.AccountId,
			Region:         row.Region,
			ResourceType:   row.ResourceType,
			ResourceId:     resourceId,
			ResourceName:   firstNonEmpty(row.ResourceName, row.Address, resourceId),
			Source:         models.CloudRiskSourceDrift,
			RuleKey:        "terraform_drift_detected",
			RuleName:       "IaC 资源漂移",
			RiskLevel:      models.CloudOperationRiskHigh,
			Evidence: models.ResAttrs{
				"resId":       row.ResId.String(),
				"address":     row.Address,
				"driftDetail": row.DriftDetail,
			},
			Recommendation:  "重新执行 plan/apply 评估漂移影响，确认后通过 IaC 回收漂移或接受变更并更新模板。",
			FingerprintSeed: []string{row.ResId.String(), row.Address},
		}
		candidates = append(candidates, item)
	}
	return candidates, nil
}

func upsertCloudRiskFinding(c *ctx.ServiceContext, candidate cloudRiskCandidate, fingerprint string, now models.Time) e.Error {
	existing := models.CloudRiskFinding{}
	err := c.DB().Model(&models.CloudRiskFinding{}).
		Where("org_id = ? and fingerprint = ?", c.OrgId, fingerprint).
		First(&existing)
	if err != nil && !e.IsRecordNotFound(err) {
		return e.New(e.DBError, err)
	}
	if e.IsRecordNotFound(err) {
		finding := cloudRiskFindingFromCandidate(c.OrgId, candidate, fingerprint, now)
		finding.Id = models.NewId("crf")
		if dbErr := models.Create(c.DB(), &finding); dbErr != nil {
			return e.New(e.DBError, dbErr)
		}
		cloudRiskEvent(c, &finding, "risk.detected", "发现云风险", finding.RuleName, models.ResAttrs{
			"ruleKey":   finding.RuleKey,
			"riskLevel": finding.RiskLevel,
			"evidence":  finding.Evidence,
		})
		return nil
	}

	status := existing.Status
	attrs := cloudRiskUpdateAttrs(candidate, now)
	if status == models.CloudRiskStatusResolved || status == "" || cloudRiskSuppressionExpired(existing, now) {
		status = models.CloudRiskStatusOpen
		attrs["status"] = status
		attrs["resolved_at"] = models.Time{}
		if existing.Status == models.CloudRiskStatusSuppressed {
			attrs["suppression_reason"] = ""
			attrs["suppressed_until"] = models.Time{}
		}
	}
	if _, dbErr := c.DB().Model(&models.CloudRiskFinding{}).
		Where("id = ? and org_id = ?", existing.Id, c.OrgId).
		UpdateAttrs(attrs); dbErr != nil {
		return e.New(e.DBError, dbErr)
	}
	return nil
}

func cloudRiskFindingFromCandidate(orgId models.Id, candidate cloudRiskCandidate, fingerprint string, now models.Time) models.CloudRiskFinding {
	return models.CloudRiskFinding{
		OrgId:          orgId,
		ProjectId:      candidate.ProjectId,
		EnvId:          candidate.EnvId,
		AssetId:        candidate.AssetId,
		CloudAccountId: candidate.CloudAccountId,
		Provider:       candidate.Provider,
		AccountId:      candidate.AccountId,
		Region:         candidate.Region,
		ResourceType:   candidate.ResourceType,
		ResourceId:     candidate.ResourceId,
		ResourceName:   candidate.ResourceName,
		Source:         candidate.Source,
		RuleKey:        candidate.RuleKey,
		RuleName:       candidate.RuleName,
		RiskLevel:      candidate.RiskLevel,
		Status:         models.CloudRiskStatusOpen,
		Fingerprint:    fingerprint,
		Evidence:       candidate.Evidence,
		Recommendation: candidate.Recommendation,
		FirstSeenAt:    now,
		LastSeenAt:     now,
	}
}

func cloudRiskUpdateAttrs(candidate cloudRiskCandidate, now models.Time) models.Attrs {
	return models.Attrs{
		"project_id":       candidate.ProjectId,
		"env_id":           candidate.EnvId,
		"asset_id":         candidate.AssetId,
		"cloud_account_id": candidate.CloudAccountId,
		"provider":         candidate.Provider,
		"account_id":       candidate.AccountId,
		"region":           candidate.Region,
		"resource_type":    candidate.ResourceType,
		"resource_id":      candidate.ResourceId,
		"resource_name":    candidate.ResourceName,
		"source":           candidate.Source,
		"rule_key":         candidate.RuleKey,
		"rule_name":        candidate.RuleName,
		"risk_level":       candidate.RiskLevel,
		"evidence":         candidate.Evidence,
		"recommendation":   candidate.Recommendation,
		"last_seen_at":     now,
	}
}

func resolveStaleCloudRiskFindings(c *ctx.ServiceContext, seen map[string]bool, now models.Time) e.Error {
	query := c.DB().Model(&models.CloudRiskFinding{}).
		Where("org_id = ?", c.OrgId).
		Where("source in (?)", []string{
			models.CloudRiskSourceCloudConfig,
			models.CloudRiskSourceCMDB,
			models.CloudRiskSourceDrift,
			models.CloudRiskSourcePolicy,
		}).
		Where("status <> ?", models.CloudRiskStatusResolved).
		Where("last_seen_at < ?", now)
	if len(seen) > 0 {
		fingerprints := make([]string, 0, len(seen))
		for fingerprint := range seen {
			fingerprints = append(fingerprints, fingerprint)
		}
		query = query.Where("fingerprint not in (?)", fingerprints)
	}
	if _, err := query.UpdateAttrs(models.Attrs{
		"status":      models.CloudRiskStatusResolved,
		"resolved_at": now,
	}); err != nil {
		return e.New(e.DBError, err)
	}
	return nil
}

func getCloudRiskFinding(c *ctx.ServiceContext, id models.Id) (*models.CloudRiskFinding, e.Error) {
	finding := models.CloudRiskFinding{}
	if err := c.DB().Model(&models.CloudRiskFinding{}).
		Where("id = ? and org_id = ?", id, c.OrgId).
		First(&finding); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.BadParam, fmt.Errorf("风险发现 %s 不存在或不属于当前组织", id))
		}
		return nil, e.New(e.DBError, err)
	}
	return &finding, nil
}

func cloudRiskFindingResp(c *ctx.ServiceContext, finding models.CloudRiskFinding) resps.CloudRiskFindingResp {
	assetName := finding.ResourceName
	if assetName == "" && finding.AssetId != "" {
		assetName = lookupName(c, &models.CmdbAsset{}, finding.AssetId)
	}
	return resps.CloudRiskFindingResp{
		CloudRiskFinding: finding,
		ProjectName:      lookupName(c, &models.Project{}, finding.ProjectId),
		EnvName:          lookupName(c, &models.Env{}, finding.EnvId),
		AssetName:        assetName,
	}
}

func cloudRiskRemediationTitle(finding models.CloudRiskFinding) string {
	rule := firstNonEmpty(finding.RuleName, finding.RuleKey, "云风险")
	resource := firstNonEmpty(finding.ResourceName, finding.ResourceId, finding.AssetId.String())
	if resource == "" {
		return fmt.Sprintf("风险整改申请：%s", rule)
	}
	return fmt.Sprintf("风险整改申请：%s / %s", rule, resource)
}

func cloudRiskRemediationDescription(finding models.CloudRiskFinding) string {
	lines := []string{
		fmt.Sprintf("风险等级：%s", firstNonEmpty(finding.RiskLevel, models.CloudOperationRiskMedium)),
		fmt.Sprintf("风险来源：%s", firstNonEmpty(finding.Source, "-")),
		fmt.Sprintf("风险规则：%s", firstNonEmpty(finding.RuleName, finding.RuleKey, "-")),
		fmt.Sprintf("云厂商/区域：%s / %s", firstNonEmpty(finding.Provider, "-"), firstNonEmpty(finding.Region, "-")),
		fmt.Sprintf("资源：%s", firstNonEmpty(finding.ResourceName, finding.ResourceId, finding.AssetId.String(), "-")),
	}
	if finding.Recommendation != "" {
		lines = append(lines, fmt.Sprintf("修复建议：%s", finding.Recommendation))
	}
	lines = append(lines, "请按风险证据完成整改、验证并回填处理结果。")
	return strings.Join(lines, "\n")
}

func cloudRiskRemediationParams(finding models.CloudRiskFinding, extra models.ResAttrs) models.ResAttrs {
	params := models.ResAttrs{}
	for key, value := range extra {
		params[key] = value
	}
	params["riskRemediation"] = true
	params["riskId"] = finding.Id.String()
	params["riskSource"] = finding.Source
	params["riskStatus"] = finding.Status
	params["riskLevel"] = finding.RiskLevel
	params["ruleKey"] = finding.RuleKey
	params["ruleName"] = finding.RuleName
	params["provider"] = finding.Provider
	params["accountId"] = finding.AccountId
	params["region"] = finding.Region
	params["resourceType"] = finding.ResourceType
	params["resourceId"] = finding.ResourceId
	params["resourceName"] = finding.ResourceName
	params["assetId"] = finding.AssetId.String()
	params["cloudAccountId"] = finding.CloudAccountId.String()
	params["projectId"] = finding.ProjectId.String()
	params["envId"] = finding.EnvId.String()
	params["recommendation"] = finding.Recommendation
	params["evidence"] = finding.Evidence
	if finding.Source == models.CloudRiskSourceDrift || finding.RuleKey == "terraform_drift_detected" {
		params["driftRemediation"] = true
		params["targetState"] = "iac_desired_state"
	}
	return params
}

func cloudRiskSummary(c *ctx.ServiceContext) (resps.CloudRiskSummaryResp, e.Error) {
	summary := resps.CloudRiskSummaryResp{}
	count := func(query *db.Session) (int64, e.Error) {
		value, err := cloudOverviewCount(query)
		if err != nil {
			return 0, e.New(e.DBError, err)
		}
		return value, nil
	}
	base := func() *db.Session {
		return c.DB().Model(&models.CloudRiskFinding{}).Where("org_id = ?", c.OrgId)
	}

	var err e.Error
	if summary.Total, err = count(base()); err != nil {
		return summary, err
	}
	if summary.Open, err = count(base().Where("status = ?", models.CloudRiskStatusOpen)); err != nil {
		return summary, err
	}
	if summary.InProgress, err = count(base().Where("status = ?", models.CloudRiskStatusInProgress)); err != nil {
		return summary, err
	}
	if summary.Suppressed, err = count(base().Where("status = ?", models.CloudRiskStatusSuppressed)); err != nil {
		return summary, err
	}
	if summary.Resolved, err = count(base().Where("status = ?", models.CloudRiskStatusResolved)); err != nil {
		return summary, err
	}
	if summary.High, err = count(base().Where("risk_level = ?", models.CloudOperationRiskHigh)); err != nil {
		return summary, err
	}
	if summary.Critical, err = count(base().Where("risk_level = ?", models.CloudOperationRiskCritical)); err != nil {
		return summary, err
	}
	if summary.ActiveHigh, err = count(base().Where("risk_level = ? and status in (?)", models.CloudOperationRiskHigh, []string{models.CloudRiskStatusOpen, models.CloudRiskStatusInProgress})); err != nil {
		return summary, err
	}
	if summary.ActiveCritical, err = count(base().Where("risk_level = ? and status in (?)", models.CloudOperationRiskCritical, []string{models.CloudRiskStatusOpen, models.CloudRiskStatusInProgress})); err != nil {
		return summary, err
	}
	if summary.PublicExposure, err = count(base().Where("rule_key in (?)", []string{"public_ingress_security_rule", "public_egress_security_rule", "s3_bucket_public_policy", "s3_bucket_public_acl"})); err != nil {
		return summary, err
	}
	if summary.UnmanagedAssets, err = count(base().Where("rule_key = ?", "unmanaged_cloud_asset")); err != nil {
		return summary, err
	}
	if summary.UnownedAssets, err = count(base().Where("rule_key = ?", "unowned_cloud_asset")); err != nil {
		return summary, err
	}
	if summary.DriftRisks, err = count(base().Where("source = ?", models.CloudRiskSourceDrift)); err != nil {
		return summary, err
	}
	if summary.ComplianceRisks, err = count(base().Where("rule_key like ?", "cmdb_compliance_risk_%")); err != nil {
		return summary, err
	}
	return summary, nil
}

func cloudRiskFingerprint(orgId models.Id, candidate cloudRiskCandidate) string {
	parts := []string{
		orgId.String(),
		candidate.Source,
		candidate.RuleKey,
		candidate.AssetId.String(),
		candidate.ResourceId,
	}
	parts = append(parts, candidate.FingerprintSeed...)
	seed := strings.Join(parts, "\x00")
	if strings.TrimSpace(seed) == "" {
		return ""
	}
	sum := sha1.Sum([]byte(seed))
	return hex.EncodeToString(sum[:])
}

func cloudRiskPublicRuleName(rule resps.CloudAssetSecurityRuleResp) string {
	if rule.Direction == "egress" {
		return "安全规则允许公网出方向访问"
	}
	return "安全规则允许公网入方向访问"
}

func cloudRiskPublicRuleLevel(rule resps.CloudAssetSecurityRuleResp) string {
	if rule.Direction == "egress" {
		return models.CloudOperationRiskMedium
	}
	ports := strings.ToLower(rule.PortRange)
	if strings.Contains(ports, "22") || strings.Contains(ports, "3389") || ports == "all" {
		return models.CloudOperationRiskCritical
	}
	return models.CloudOperationRiskHigh
}

func cloudRiskS3ConfigCandidates(base cloudRiskCandidate, asset models.CmdbAsset) []cloudRiskCandidate {
	if !cloudRiskIsS3Bucket(asset) {
		return nil
	}
	attrs := asset.Attributes
	candidates := make([]cloudRiskCandidate, 0)
	if policy := modelResAttrs(attrs["bucketPolicy"]); attrBool(policy, "publicAllow") {
		item := base
		item.Source = models.CloudRiskSourceCloudConfig
		item.RuleKey = "s3_bucket_public_policy"
		item.RuleName = "S3 Bucket Policy 允许公开访问"
		item.RiskLevel = models.CloudOperationRiskCritical
		item.Evidence = models.ResAttrs{
			"bucket":         firstNonEmpty(attrString(attrs, "bucket"), asset.NativeId),
			"publicAllow":    true,
			"statementCount": attrInt(policy, "statementCount"),
			"policyVersion":  attrString(policy, "version"),
		}
		item.Recommendation = "移除 Bucket Policy 中对 * 或公共主体的 Allow 授权，并配合 Public Access Block 阻断公共访问。"
		candidates = append(candidates, item)
	}
	if value, ok := cloudRiskAttrBool(attrs, "bucketAclPublic"); ok && value {
		item := base
		item.Source = models.CloudRiskSourceCloudConfig
		item.RuleKey = "s3_bucket_public_acl"
		item.RuleName = "S3 Bucket ACL 允许公开访问"
		item.RiskLevel = models.CloudOperationRiskCritical
		item.Evidence = models.ResAttrs{
			"bucket":     firstNonEmpty(attrString(attrs, "bucket"), asset.NativeId),
			"publicAcl":  true,
			"grantCount": attrInt(attrs, "bucketAclGrantCount"),
		}
		item.Recommendation = "移除 AllUsers/AuthenticatedUsers ACL 授权，优先启用 Bucket owner enforced 并关闭 ACL 公共访问。"
		candidates = append(candidates, item)
	}
	if value, ok := cloudRiskAttrBool(attrs, "publicAccessBlockEnabled"); ok && !value {
		item := base
		item.Source = models.CloudRiskSourceCloudConfig
		item.RuleKey = "s3_public_access_block_disabled"
		item.RuleName = "S3 Public Access Block 未完全开启"
		item.RiskLevel = models.CloudOperationRiskHigh
		item.Evidence = models.ResAttrs{
			"bucket":            firstNonEmpty(attrString(attrs, "bucket"), asset.NativeId),
			"publicAccessBlock": modelResAttrs(attrs["publicAccessBlock"]),
		}
		item.Recommendation = "开启 BlockPublicAcls、IgnorePublicAcls、BlockPublicPolicy 和 RestrictPublicBuckets 四项配置。"
		candidates = append(candidates, item)
	}
	if value, ok := cloudRiskAttrBool(attrs, "encryptionEnabled"); ok && !value {
		item := base
		item.Source = models.CloudRiskSourceCloudConfig
		item.RuleKey = "s3_default_encryption_disabled"
		item.RuleName = "S3 Bucket 未开启默认加密"
		item.RiskLevel = models.CloudOperationRiskHigh
		item.Evidence = models.ResAttrs{
			"bucket":            firstNonEmpty(attrString(attrs, "bucket"), asset.NativeId),
			"encryptionEnabled": false,
		}
		item.Recommendation = "为 Bucket 启用默认服务端加密，优先使用受控 KMS Key 并开启 Bucket Key 降低成本。"
		candidates = append(candidates, item)
	}
	if cloudRiskS3VersioningCollected(attrs) && !strings.EqualFold(attrString(attrs, "versioningStatus"), "Enabled") {
		item := base
		item.Source = models.CloudRiskSourceCloudConfig
		item.RuleKey = "s3_versioning_disabled"
		item.RuleName = "S3 Bucket 未开启版本控制"
		item.RiskLevel = models.CloudOperationRiskMedium
		item.Evidence = models.ResAttrs{
			"bucket":           firstNonEmpty(attrString(attrs, "bucket"), asset.NativeId),
			"versioningStatus": attrString(attrs, "versioningStatus"),
			"versioning":       modelResAttrs(attrs["versioning"]),
		}
		item.Recommendation = "开启 Bucket Versioning，并结合生命周期规则管理历史版本成本。"
		candidates = append(candidates, item)
	}
	if value, ok := cloudRiskAttrBool(attrs, "loggingEnabled"); ok && !value {
		item := base
		item.Source = models.CloudRiskSourceCloudConfig
		item.RuleKey = "s3_access_logging_disabled"
		item.RuleName = "S3 Bucket 未开启访问日志"
		item.RiskLevel = models.CloudOperationRiskMedium
		item.Evidence = models.ResAttrs{
			"bucket":         firstNonEmpty(attrString(attrs, "bucket"), asset.NativeId),
			"loggingEnabled": false,
			"logging":        modelResAttrs(attrs["logging"]),
		}
		item.Recommendation = "开启 S3 Server Access Logging 或接入 CloudTrail Data Events，满足访问审计和追踪要求。"
		candidates = append(candidates, item)
	}
	if value, ok := cloudRiskAttrBool(attrs, "objectLockEnabled"); ok && !value {
		item := base
		item.Source = models.CloudRiskSourceCloudConfig
		item.RuleKey = "s3_object_lock_disabled"
		item.RuleName = "S3 Bucket 未开启 Object Lock"
		item.RiskLevel = models.CloudOperationRiskMedium
		item.Evidence = models.ResAttrs{
			"bucket":            firstNonEmpty(attrString(attrs, "bucket"), asset.NativeId),
			"objectLockEnabled": false,
			"objectLock":        modelResAttrs(attrs["objectLock"]),
		}
		item.Recommendation = "对关键备份、审计日志或合规留存 Bucket 启用 Object Lock，并配置默认保留周期。"
		candidates = append(candidates, item)
	}
	return candidates
}

func cloudRiskIsS3Bucket(asset models.CmdbAsset) bool {
	if asset.AssetType != models.CmdbAssetTypeObjectStorageBucket {
		return false
	}
	return normalizeProvider(asset.Provider) == "aws" || strings.EqualFold(asset.NativeType, "aws_s3_bucket")
}

func cloudRiskAttrBool(attrs models.ResAttrs, key string) (bool, bool) {
	if attrs == nil {
		return false, false
	}
	if _, ok := attrs[key]; !ok {
		return false, false
	}
	return attrBool(attrs, key), true
}

func cloudRiskS3VersioningCollected(attrs models.ResAttrs) bool {
	if attrs == nil {
		return false
	}
	if _, ok := attrs["versioning"]; ok {
		return true
	}
	_, ok := attrs["versioningStatus"]
	return ok
}

func cloudRiskSuppressionExpired(finding models.CloudRiskFinding, now models.Time) bool {
	if finding.Status != models.CloudRiskStatusSuppressed {
		return false
	}
	suppressedUntil := time.Time(finding.SuppressedUntil)
	if suppressedUntil.IsZero() {
		return true
	}
	return !suppressedUntil.After(time.Time(now))
}

func mergeCloudRiskEvidence(base models.ResAttrs, extra models.ResAttrs) models.ResAttrs {
	merged := models.ResAttrs{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func cloudRiskRecommendationAdoptionEvidence(base models.ResAttrs, form forms.AdoptCloudRiskRecommendationForm, userId models.Id, adoptedAt time.Time) (models.ResAttrs, models.ResAttrs, e.Error) {
	action := strings.TrimSpace(form.Action)
	targetType := strings.TrimSpace(form.TargetType)
	targetId := strings.TrimSpace(form.TargetId)
	if action == "" {
		return nil, nil, e.New(e.BadParam, fmt.Errorf("推荐动作不能为空"))
	}
	recommendations := cloudRiskRecommendationAttrs(base["driftAutoRepairRecommendations"])
	if len(recommendations) == 0 {
		return nil, nil, e.New(e.BadParam, fmt.Errorf("当前风险没有可采纳的漂移自动修复推荐"))
	}
	matchedIndex := -1
	for index, item := range recommendations {
		if !cloudRiskRecommendationMatches(item, action, targetType, targetId) {
			continue
		}
		matchedIndex = index
		if targetType == "" {
			targetType = cloudSyncPolicyAttrString(item["targetType"])
		}
		if targetId == "" {
			targetId = cloudSyncPolicyAttrString(item["targetId"])
		}
		break
	}
	if matchedIndex < 0 {
		return nil, nil, e.New(e.BadParam, fmt.Errorf("未找到可采纳的推荐动作 %s", action))
	}
	matched := recommendations[matchedIndex]
	adoption := models.ResAttrs{
		"action":     action,
		"title":      firstNonEmpty(cloudSyncPolicyAttrString(matched["title"]), action),
		"priority":   cloudSyncPolicyAttrString(matched["priority"]),
		"reason":     cloudSyncPolicyAttrString(matched["reason"]),
		"targetType": targetType,
		"targetId":   targetId,
		"adopted":    true,
		"adoptedAt":  adoptedAt.Format(time.RFC3339),
		"adoptedBy":  userId.String(),
		"comment":    strings.TrimSpace(form.Comment),
	}
	adoptions := cloudRiskRecommendationAttrs(base["driftAutoRepairRecommendationAdoptions"])
	adoptions = append(adoptions, adoption)
	adoptedByKey := map[string]models.ResAttrs{}
	for _, item := range adoptions {
		key := cloudRiskRecommendationKey(
			cloudSyncPolicyAttrString(item["action"]),
			cloudSyncPolicyAttrString(item["targetType"]),
			cloudSyncPolicyAttrString(item["targetId"]),
		)
		if key != "" {
			adoptedByKey[key] = item
		}
	}
	adoptedTotal := 0
	for index, item := range recommendations {
		key := cloudRiskRecommendationKey(
			cloudSyncPolicyAttrString(item["action"]),
			cloudSyncPolicyAttrString(item["targetType"]),
			cloudSyncPolicyAttrString(item["targetId"]),
		)
		if adopted, ok := adoptedByKey[key]; ok {
			recommendations[index]["adopted"] = true
			recommendations[index]["adoptedAt"] = adopted["adoptedAt"]
			recommendations[index]["adoptedBy"] = adopted["adoptedBy"]
			if comment := cloudSyncPolicyAttrString(adopted["comment"]); comment != "" {
				recommendations[index]["adoptionComment"] = comment
			}
			adoptedTotal++
		}
	}
	total := len(recommendations)
	rate := 0.0
	if total > 0 {
		rate = float64(adoptedTotal) / float64(total) * 100
	}
	evidence := mergeCloudRiskEvidence(base, models.ResAttrs{
		"driftAutoRepairRecommendations":               recommendations,
		"driftAutoRepairRecommendationAdoptions":       adoptions,
		"driftAutoRepairLastRecommendationAdoption":    adoption,
		"driftAutoRepairRecommendationTotal":           total,
		"driftAutoRepairRecommendationAdoptedTotal":    adoptedTotal,
		"driftAutoRepairRecommendationAdoptionRate":    rate,
		"driftAutoRepairRecommendationsAllAdopted":     total > 0 && adoptedTotal == total,
		"driftAutoRepairRecommendationLastAdoptedAt":   adoption["adoptedAt"],
		"driftAutoRepairRecommendationLastAdoptedBy":   adoption["adoptedBy"],
		"driftAutoRepairRecommendationLastAdoptedName": adoption["title"],
	})
	return evidence, adoption, nil
}

func cloudRiskRecommendationAttrs(value interface{}) []models.ResAttrs {
	switch typed := value.(type) {
	case []models.ResAttrs:
		items := make([]models.ResAttrs, 0, len(typed))
		for _, item := range typed {
			items = append(items, modelResAttrs(item))
		}
		return items
	case []map[string]interface{}:
		items := make([]models.ResAttrs, 0, len(typed))
		for _, item := range typed {
			items = append(items, modelResAttrs(item))
		}
		return items
	case []interface{}:
		items := make([]models.ResAttrs, 0, len(typed))
		for _, item := range typed {
			if attrs := modelResAttrs(item); attrs != nil {
				items = append(items, attrs)
			}
		}
		return items
	default:
		return nil
	}
}

func cloudRiskRecommendationMatches(item models.ResAttrs, action string, targetType string, targetId string) bool {
	if cloudSyncPolicyAttrString(item["action"]) != action {
		return false
	}
	if targetType != "" && cloudSyncPolicyAttrString(item["targetType"]) != targetType {
		return false
	}
	if targetId != "" && cloudSyncPolicyAttrString(item["targetId"]) != targetId {
		return false
	}
	return true
}

func cloudRiskRecommendationKey(action string, targetType string, targetId string) string {
	action = strings.TrimSpace(action)
	if action == "" {
		return ""
	}
	return strings.Join([]string{action, strings.TrimSpace(targetType), strings.TrimSpace(targetId)}, "|")
}
