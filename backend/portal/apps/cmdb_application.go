// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/libs/page"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	"cloudiac/portal/models/resps"
)

type cmdbApplicationAggregate struct {
	resp            resps.CmdbApplicationResp
	ownerSet        map[string]bool
	businessLineSet map[string]bool
	assetIds        map[models.Id]bool
	changedAssetIds map[models.Id]bool
	upstreams       map[string]*resps.CmdbApplicationRelationResp
	downstreams     map[string]*resps.CmdbApplicationRelationResp
	upstreamSets    map[string]*cmdbApplicationRelationAssetSet
	downstreamSets  map[string]*cmdbApplicationRelationAssetSet
}

type cmdbApplicationRelationAssetSet struct {
	sourceAssets map[models.Id]bool
	targetAssets map[models.Id]bool
}

func SearchCmdbApplications(c *ctx.ServiceContext, form *forms.SearchCmdbApplicationForm) (*page.PageResp, e.Error) {
	apps, err := buildCmdbApplications(c)
	if err != nil {
		return nil, err
	}
	apps = filterCmdbApplications(apps, form)
	sortCmdbApplications(apps)

	total := int64(len(apps))
	start, end := cmdbApplicationPageRange(form.CurrentPage(), form.PageSize(), len(apps))
	return &page.PageResp{
		Total:    total,
		PageSize: form.PageSize(),
		List:     apps[start:end],
	}, nil
}

func CmdbApplicationDetail(c *ctx.ServiceContext, form *forms.CmdbApplicationDetailForm) (*resps.CmdbApplicationResp, e.Error) {
	apps, err := buildCmdbApplications(c)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(form.Application)
	for i := range apps {
		if apps[i].Application == name {
			return &apps[i], nil
		}
	}
	return nil, e.New(e.ObjectNotExistsOrNoPerm, fmt.Errorf("cmdb application %s not found", name))
}

func UpdateCmdbApplicationRelations(c *ctx.ServiceContext, form *forms.UpdateCmdbApplicationRelationsForm) (*resps.CmdbApplicationResp, e.Error) {
	sourceApplication := strings.TrimSpace(form.Application)
	if sourceApplication == "" {
		return nil, e.New(e.BadParam, fmt.Errorf("application is required"))
	}

	apps, err := buildCmdbApplications(c)
	if err != nil {
		return nil, err
	}
	knownApplications := make(map[string]bool, len(apps))
	for _, app := range apps {
		knownApplications[app.Application] = true
	}
	if !knownApplications[sourceApplication] {
		return nil, e.New(e.ObjectNotExistsOrNoPerm, fmt.Errorf("cmdb application %s not found", sourceApplication))
	}
	if err := ensureCmdbApplicationRelationPermission(c, sourceApplication); err != nil {
		return nil, err
	}

	upstreams, normalizeErr := normalizeCmdbApplicationRelationNames(sourceApplication, form.Upstreams, knownApplications)
	if normalizeErr != nil {
		return nil, normalizeErr
	}
	downstreams, normalizeErr := normalizeCmdbApplicationRelationNames(sourceApplication, form.Downstreams, knownApplications)
	if normalizeErr != nil {
		return nil, normalizeErr
	}

	beforeRelations, dbErr := cmdbApplicationManualRelationSnapshot(c, sourceApplication)
	if dbErr != nil {
		return nil, e.New(e.DBError, dbErr)
	}

	tx := c.DB().Begin()
	if _, err := tx.Where(
		"org_id = ? and source = ? and (source_application = ? or target_application = ?)",
		c.OrgId, models.CmdbRelationSourceManualApp, sourceApplication, sourceApplication,
	).Delete(&models.CmdbApplicationRelation{}); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, err)
	}

	relations := make([]models.CmdbApplicationRelation, 0, len(upstreams)+len(downstreams))
	for _, upstream := range upstreams {
		relations = append(relations, newCmdbApplicationRelation(c.OrgId, upstream, sourceApplication))
	}
	for _, downstream := range downstreams {
		relations = append(relations, newCmdbApplicationRelation(c.OrgId, sourceApplication, downstream))
	}
	if len(relations) > 0 {
		if err := tx.Insert(&relations); err != nil {
			_ = tx.Rollback()
			return nil, e.New(e.DBError, err)
		}
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, err)
	}

	if cmdbApplicationRelationSnapshotChanged(beforeRelations, upstreams, downstreams) {
		cmdbApplicationRelationsEvent(c, sourceApplication, beforeRelations.Upstreams, upstreams, beforeRelations.Downstreams, downstreams)
	}

	return CmdbApplicationDetail(c, &forms.CmdbApplicationDetailForm{Application: sourceApplication})
}

type cmdbApplicationManualRelationSnapshotResp struct {
	Upstreams   []string
	Downstreams []string
}

func cmdbApplicationManualRelationSnapshot(c *ctx.ServiceContext, application string) (cmdbApplicationManualRelationSnapshotResp, error) {
	relations := make([]models.CmdbApplicationRelation, 0)
	if err := c.DB().
		Where("org_id = ? and source = ? and (source_application = ? or target_application = ?)",
			c.OrgId, models.CmdbRelationSourceManualApp, application, application).
		Find(&relations); err != nil {
		return cmdbApplicationManualRelationSnapshotResp{}, err
	}

	upstreamSet := make(map[string]bool)
	downstreamSet := make(map[string]bool)
	for _, relation := range relations {
		if strings.TrimSpace(relation.TargetApplication) == application {
			appendCmdbApplicationRelationName(upstreamSet, relation.SourceApplication)
		}
		if strings.TrimSpace(relation.SourceApplication) == application {
			appendCmdbApplicationRelationName(downstreamSet, relation.TargetApplication)
		}
	}
	return cmdbApplicationManualRelationSnapshotResp{
		Upstreams:   sortedCmdbApplicationRelationNames(upstreamSet),
		Downstreams: sortedCmdbApplicationRelationNames(downstreamSet),
	}, nil
}

func appendCmdbApplicationRelationName(set map[string]bool, name string) {
	name = strings.TrimSpace(name)
	if name != "" {
		set[name] = true
	}
}

func sortedCmdbApplicationRelationNames(set map[string]bool) []string {
	items := make([]string, 0, len(set))
	for item := range set {
		items = append(items, item)
	}
	sort.Strings(items)
	return items
}

func cmdbApplicationRelationSnapshotChanged(before cmdbApplicationManualRelationSnapshotResp, upstreams []string, downstreams []string) bool {
	return !sameCmdbApplicationRelationNames(before.Upstreams, upstreams) ||
		!sameCmdbApplicationRelationNames(before.Downstreams, downstreams)
}

func sameCmdbApplicationRelationNames(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func buildCmdbApplications(c *ctx.ServiceContext) ([]resps.CmdbApplicationResp, e.Error) {
	if _, err := BackfillCmdbAssetsFromIac(c); err != nil {
		return nil, err
	}
	riskConfig, err := loadCmdbRiskRuleConfig(c)
	if err != nil {
		return nil, err
	}

	assets := make([]resps.CmdbAssetResp, 0)
	if err := buildCmdbAssetQuery(c).
		Where("iac_cmdb_asset.application <> ''").
		Scan(&assets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if len(assets) == 0 {
		return []resps.CmdbApplicationResp{}, nil
	}

	aggregates := make(map[string]*cmdbApplicationAggregate)
	assetById := make(map[models.Id]resps.CmdbAssetResp)
	assetIds := make([]models.Id, 0, len(assets))
	for _, asset := range assets {
		appName := strings.TrimSpace(asset.Application)
		if appName == "" {
			continue
		}
		agg := ensureCmdbApplicationAggregate(aggregates, appName)
		agg.resp.AssetCount++
		agg.assetIds[asset.Id] = true
		agg.resp.Assets = append(agg.resp.Assets, cmdbApplicationAssetResp(asset))
		appendCmdbApplicationSet(agg.ownerSet, asset.Owner)
		appendCmdbApplicationSet(agg.businessLineSet, asset.BusinessLine)
		assetById[asset.Id] = asset
		assetIds = append(assetIds, asset.Id)
	}
	if len(assetIds) == 0 {
		return []resps.CmdbApplicationResp{}, nil
	}

	if err := fillCmdbApplicationRelations(c, aggregates, assetById, assetIds); err != nil {
		return nil, err
	}
	if err := fillCmdbApplicationManualRelations(c, aggregates); err != nil {
		return nil, err
	}
	if err := fillCmdbApplicationRecentChanges(c, aggregates, assetById, assetIds, riskConfig.ChangeWindowDays); err != nil {
		return nil, err
	}

	resp := make([]resps.CmdbApplicationResp, 0, len(aggregates))
	for _, agg := range aggregates {
		agg.resp.Owner = sortedCmdbApplicationSet(agg.ownerSet)
		agg.resp.BusinessLine = sortedCmdbApplicationSet(agg.businessLineSet)
		agg.resp.IncomingAppCount = len(agg.upstreams)
		agg.resp.OutgoingAppCount = len(agg.downstreams)
		agg.resp.ChangedAssetCount = len(agg.changedAssetIds)
		agg.resp.Upstreams = sortedCmdbApplicationRelations(agg.upstreams)
		agg.resp.Downstreams = sortedCmdbApplicationRelations(agg.downstreams)
		agg.resp.ImpactedApps = cmdbApplicationImpacts(agg.resp)
		agg.resp.ImpactedAppCount = len(agg.resp.ImpactedApps)
		sortCmdbApplicationAssets(agg.resp.Assets)
		sortCmdbApplicationChanges(agg.resp.RecentChanges)
		agg.resp.RiskLevel, agg.resp.RiskReason, agg.resp.RiskScore = cmdbApplicationRisk(agg.resp, riskConfig)
		resp = append(resp, agg.resp)
	}
	return resp, nil
}

func ensureCmdbApplicationAggregate(aggregates map[string]*cmdbApplicationAggregate, appName string) *cmdbApplicationAggregate {
	if agg, ok := aggregates[appName]; ok {
		return agg
	}
	agg := &cmdbApplicationAggregate{
		resp: resps.CmdbApplicationResp{
			Application:   appName,
			Assets:        make([]resps.CmdbApplicationAssetResp, 0),
			Upstreams:     make([]resps.CmdbApplicationRelationResp, 0),
			Downstreams:   make([]resps.CmdbApplicationRelationResp, 0),
			RecentChanges: make([]resps.CmdbApplicationChangeResp, 0),
		},
		ownerSet:        make(map[string]bool),
		businessLineSet: make(map[string]bool),
		assetIds:        make(map[models.Id]bool),
		changedAssetIds: make(map[models.Id]bool),
		upstreams:       make(map[string]*resps.CmdbApplicationRelationResp),
		downstreams:     make(map[string]*resps.CmdbApplicationRelationResp),
		upstreamSets:    make(map[string]*cmdbApplicationRelationAssetSet),
		downstreamSets:  make(map[string]*cmdbApplicationRelationAssetSet),
	}
	aggregates[appName] = agg
	return agg
}

func fillCmdbApplicationRelations(
	c *ctx.ServiceContext,
	aggregates map[string]*cmdbApplicationAggregate,
	assetById map[models.Id]resps.CmdbAssetResp,
	assetIds []models.Id,
) e.Error {
	relations := make([]models.CmdbAssetRelation, 0)
	if err := c.DB().Where("org_id = ? and source_asset_id in (?) and target_asset_id in (?)", c.OrgId, assetIds, assetIds).
		Find(&relations); err != nil {
		return e.New(e.DBError, err)
	}
	for _, relation := range relations {
		sourceAsset, sourceOk := assetById[relation.SourceAssetId]
		targetAsset, targetOk := assetById[relation.TargetAssetId]
		if !sourceOk || !targetOk {
			continue
		}
		sourceApp := strings.TrimSpace(sourceAsset.Application)
		targetApp := strings.TrimSpace(targetAsset.Application)
		if sourceApp == "" || targetApp == "" || sourceApp == targetApp {
			continue
		}
		sourceAgg := ensureCmdbApplicationAggregate(aggregates, sourceApp)
		targetAgg := ensureCmdbApplicationAggregate(aggregates, targetApp)
		accumulateCmdbApplicationRelation(sourceAgg.downstreams, sourceAgg.downstreamSets, targetApp, "outgoing", relation, relation.SourceAssetId, relation.TargetAssetId)
		accumulateCmdbApplicationRelation(targetAgg.upstreams, targetAgg.upstreamSets, sourceApp, "incoming", relation, relation.SourceAssetId, relation.TargetAssetId)
	}
	return nil
}

func accumulateCmdbApplicationRelation(
	relations map[string]*resps.CmdbApplicationRelationResp,
	assetSets map[string]*cmdbApplicationRelationAssetSet,
	relatedApplication string,
	direction string,
	relation models.CmdbAssetRelation,
	sourceAssetId models.Id,
	targetAssetId models.Id,
) {
	key := fmt.Sprintf("%s/%s/%s", relatedApplication, relation.RelationType, relation.Source)
	resp, ok := relations[key]
	if !ok {
		resp = &resps.CmdbApplicationRelationResp{
			Application:  relatedApplication,
			Direction:    direction,
			RelationType: relation.RelationType,
			Source:       relation.Source,
		}
		relations[key] = resp
	}
	assetSet, ok := assetSets[key]
	if !ok {
		assetSet = &cmdbApplicationRelationAssetSet{
			sourceAssets: make(map[models.Id]bool),
			targetAssets: make(map[models.Id]bool),
		}
		assetSets[key] = assetSet
	}
	assetSet.sourceAssets[sourceAssetId] = true
	assetSet.targetAssets[targetAssetId] = true
	resp.AssetRelationCount++
	resp.SourceAssetCount = len(assetSet.sourceAssets)
	resp.TargetAssetCount = len(assetSet.targetAssets)
	if time.Time(relation.UpdatedAt).After(time.Time(resp.LatestRelationAt)) {
		resp.LatestRelationAt = relation.UpdatedAt
	}
}

func fillCmdbApplicationManualRelations(c *ctx.ServiceContext, aggregates map[string]*cmdbApplicationAggregate) e.Error {
	relations := make([]models.CmdbApplicationRelation, 0)
	if err := c.DB().
		Where("org_id = ? and source = ?", c.OrgId, models.CmdbRelationSourceManualApp).
		Find(&relations); err != nil {
		return e.New(e.DBError, err)
	}
	for _, relation := range relations {
		sourceApp := strings.TrimSpace(relation.SourceApplication)
		targetApp := strings.TrimSpace(relation.TargetApplication)
		if sourceApp == "" || targetApp == "" || sourceApp == targetApp {
			continue
		}
		sourceAgg, sourceOk := aggregates[sourceApp]
		targetAgg, targetOk := aggregates[targetApp]
		if !sourceOk || !targetOk {
			continue
		}
		accumulateCmdbApplicationManualRelation(sourceAgg.downstreams, targetApp, "outgoing", relation, sourceAgg.resp.AssetCount, targetAgg.resp.AssetCount)
		accumulateCmdbApplicationManualRelation(targetAgg.upstreams, sourceApp, "incoming", relation, sourceAgg.resp.AssetCount, targetAgg.resp.AssetCount)
	}
	return nil
}

func accumulateCmdbApplicationManualRelation(
	relations map[string]*resps.CmdbApplicationRelationResp,
	relatedApplication string,
	direction string,
	relation models.CmdbApplicationRelation,
	sourceAssetCount int,
	targetAssetCount int,
) {
	key := fmt.Sprintf("%s/%s/%s", relatedApplication, relation.RelationType, relation.Source)
	resp, ok := relations[key]
	if !ok {
		resp = &resps.CmdbApplicationRelationResp{
			Application:  relatedApplication,
			Direction:    direction,
			RelationType: relation.RelationType,
			Source:       relation.Source,
		}
		relations[key] = resp
	}
	resp.AssetRelationCount++
	resp.SourceAssetCount = sourceAssetCount
	resp.TargetAssetCount = targetAssetCount
	if time.Time(relation.UpdatedAt).After(time.Time(resp.LatestRelationAt)) {
		resp.LatestRelationAt = relation.UpdatedAt
	}
}

func fillCmdbApplicationRecentChanges(
	c *ctx.ServiceContext,
	aggregates map[string]*cmdbApplicationAggregate,
	assetById map[models.Id]resps.CmdbAssetResp,
	assetIds []models.Id,
	changeWindowDays int,
) e.Error {
	changes := make([]models.CmdbAssetChange, 0)
	if changeWindowDays <= 0 {
		changeWindowDays = 7
	}
	cutoff := time.Now().AddDate(0, 0, -changeWindowDays)
	if err := c.DB().Where("org_id = ? and asset_id in (?) and created_at >= ?", c.OrgId, assetIds, cutoff).
		Order("created_at desc").
		Find(&changes); err != nil {
		return e.New(e.DBError, err)
	}
	for _, change := range changes {
		asset, ok := assetById[change.AssetId]
		if !ok {
			continue
		}
		appName := strings.TrimSpace(asset.Application)
		if appName == "" {
			continue
		}
		agg := ensureCmdbApplicationAggregate(aggregates, appName)
		agg.resp.RecentChangeCount++
		agg.changedAssetIds[asset.Id] = true
		if len(agg.resp.RecentChanges) >= 20 {
			continue
		}
		agg.resp.RecentChanges = append(agg.resp.RecentChanges, resps.CmdbApplicationChangeResp{
			Id:         change.Id,
			AssetId:    asset.Id,
			AssetName:  firstNonEmpty(asset.Name, asset.NativeId),
			ChangeType: change.ChangeType,
			Source:     change.Source,
			Summary:    normalizeCmdbText(change.Summary),
			Diff:       change.Diff,
			CreatedAt:  change.CreatedAt,
		})
	}
	return nil
}

func normalizeCmdbApplicationRelationNames(sourceApplication string, names []string, knownApplications map[string]bool) ([]string, e.Error) {
	seen := make(map[string]bool)
	normalized := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		if name == sourceApplication {
			return nil, e.New(e.BadParam, fmt.Errorf("application cannot depend on itself: %s", name))
		}
		if !knownApplications[name] {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, fmt.Errorf("cmdb application %s not found", name))
		}
		seen[name] = true
		normalized = append(normalized, name)
	}
	sort.Strings(normalized)
	return normalized, nil
}

func newCmdbApplicationRelation(orgId models.Id, sourceApplication, targetApplication string) models.CmdbApplicationRelation {
	return models.CmdbApplicationRelation{
		TimedModel: models.TimedModel{
			BaseModel: models.BaseModel{Id: models.NewId("car")},
		},
		OrgId:             orgId,
		SourceApplication: sourceApplication,
		TargetApplication: targetApplication,
		RelationType:      models.CmdbRelationTypeDependsOn,
		Source:            models.CmdbRelationSourceManualApp,
		Metadata: models.ResAttrs{
			"maintainedBy": "cmdb_application",
		},
	}
}

func cmdbApplicationAssetResp(asset resps.CmdbAssetResp) resps.CmdbApplicationAssetResp {
	return resps.CmdbApplicationAssetResp{
		Id:             asset.Id,
		Name:           firstNonEmpty(asset.Name, asset.NativeId),
		AssetType:      asset.AssetType,
		Provider:       asset.Provider,
		NativeId:       asset.NativeId,
		Status:         asset.Status,
		Owner:          asset.Owner,
		BusinessLine:   asset.BusinessLine,
		Lifecycle:      asset.Lifecycle,
		ComplianceRisk: asset.ComplianceRisk,
		UpdatedAt:      asset.UpdatedAt,
	}
}

func appendCmdbApplicationSet(set map[string]bool, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		set[value] = true
	}
}

func sortedCmdbApplicationSet(set map[string]bool) []string {
	items := make([]string, 0, len(set))
	for item := range set {
		items = append(items, item)
	}
	sort.Strings(items)
	return items
}

func sortedCmdbApplicationRelations(items map[string]*resps.CmdbApplicationRelationResp) []resps.CmdbApplicationRelationResp {
	relations := make([]resps.CmdbApplicationRelationResp, 0, len(items))
	for _, item := range items {
		relations = append(relations, *item)
	}
	sort.Slice(relations, func(i, j int) bool {
		if relations[i].AssetRelationCount == relations[j].AssetRelationCount {
			return relations[i].Application < relations[j].Application
		}
		return relations[i].AssetRelationCount > relations[j].AssetRelationCount
	})
	return relations
}

func cmdbApplicationImpacts(app resps.CmdbApplicationResp) []resps.CmdbApplicationImpactResp {
	if app.RecentChangeCount == 0 || len(app.Upstreams) == 0 {
		return []resps.CmdbApplicationImpactResp{}
	}

	impacts := make(map[string]*resps.CmdbApplicationImpactResp)
	sourceSets := make(map[string]map[string]bool)
	for _, relation := range app.Upstreams {
		application := strings.TrimSpace(relation.Application)
		if application == "" {
			continue
		}
		impact, ok := impacts[application]
		if !ok {
			impact = &resps.CmdbApplicationImpactResp{
				Application:     application,
				RelationSources: make([]string, 0),
			}
			impacts[application] = impact
			sourceSets[application] = make(map[string]bool)
		}
		impact.RelationCount += relation.AssetRelationCount
		impact.SourceAssetCount += relation.SourceAssetCount
		impact.TargetAssetCount += relation.TargetAssetCount
		if relation.Source != "" {
			sourceSets[application][relation.Source] = true
		}
		if time.Time(relation.LatestRelationAt).After(time.Time(impact.LatestRelationAt)) {
			impact.LatestRelationAt = relation.LatestRelationAt
		}
	}

	resp := make([]resps.CmdbApplicationImpactResp, 0, len(impacts))
	for application, impact := range impacts {
		for source := range sourceSets[application] {
			impact.RelationSources = append(impact.RelationSources, source)
		}
		sort.Strings(impact.RelationSources)
		resp = append(resp, *impact)
	}
	sort.Slice(resp, func(i, j int) bool {
		if resp[i].RelationCount == resp[j].RelationCount {
			return resp[i].Application < resp[j].Application
		}
		return resp[i].RelationCount > resp[j].RelationCount
	})
	return resp
}

func sortCmdbApplicationAssets(assets []resps.CmdbApplicationAssetResp) {
	sort.Slice(assets, func(i, j int) bool {
		return time.Time(assets[i].UpdatedAt).After(time.Time(assets[j].UpdatedAt))
	})
}

func sortCmdbApplicationChanges(changes []resps.CmdbApplicationChangeResp) {
	sort.Slice(changes, func(i, j int) bool {
		return time.Time(changes[i].CreatedAt).After(time.Time(changes[j].CreatedAt))
	})
}

func cmdbApplicationRisk(app resps.CmdbApplicationResp, config *models.CmdbRiskRuleConfig) (string, string, int) {
	if config == nil {
		defaultConfig := cmdbDefaultRiskRuleConfig("")
		config = &defaultConfig
	}
	highCompliance, criticalCompliance := cmdbApplicationComplianceRiskCounts(app)
	maintenanceLifecycle, retiredLifecycle := cmdbApplicationLifecycleCounts(app)
	crossBusinessLineCount := len(app.BusinessLine) - 1
	if crossBusinessLineCount < 0 {
		crossBusinessLineCount = 0
	}
	score := app.RecentChangeCount*config.RecentChangeWeight +
		app.ChangedAssetCount*config.ChangedAssetWeight +
		app.IncomingAppCount*config.IncomingAppWeight +
		app.OutgoingAppCount*config.OutgoingAppWeight +
		highCompliance*config.HighComplianceRiskWeight +
		criticalCompliance*config.CriticalComplianceRiskWeight +
		maintenanceLifecycle*config.MaintenanceLifecycleWeight +
		retiredLifecycle*config.RetiredLifecycleWeight +
		crossBusinessLineCount*config.CrossBusinessLineWeight
	windowText := fmt.Sprintf("近 %d 天", config.ChangeWindowDays)
	switch {
	case app.RecentChangeCount > 0 && app.IncomingAppCount >= config.CriticalIncomingThreshold:
		return "critical", fmt.Sprintf("%s有变更且存在多个调用方", windowText), score + config.RecentCriticalBoost
	case app.RecentChangeCount > 0 && app.IncomingAppCount > 0:
		return "high", fmt.Sprintf("%s有变更，可能影响调用方", windowText), score + config.RecentHighBoost
	case app.RecentChangeCount > 0:
		return "medium", fmt.Sprintf("%s存在资产变更", windowText), score + config.RecentMediumBoost
	case app.IncomingAppCount >= config.MediumIncomingThreshold || app.OutgoingAppCount >= config.MediumOutgoingThreshold:
		return "medium", "应用依赖面较广", score + config.WideDependencyBoost
	case score >= config.CriticalScoreThreshold:
		return "critical", "风险评分达到严重阈值", score
	case score >= config.HighScoreThreshold:
		return "high", "风险评分达到高阈值", score
	case score >= config.MediumScoreThreshold:
		return "medium", "风险评分达到中阈值", score
	default:
		return "low", "暂无明显变更风险", score
	}
}

func cmdbApplicationComplianceRiskCounts(app resps.CmdbApplicationResp) (int, int) {
	high := 0
	critical := 0
	for _, asset := range app.Assets {
		switch strings.TrimSpace(asset.ComplianceRisk) {
		case models.CloudOperationRiskHigh:
			high++
		case models.CloudOperationRiskCritical:
			critical++
		}
	}
	return high, critical
}

func cmdbApplicationLifecycleCounts(app resps.CmdbApplicationResp) (int, int) {
	maintenance := 0
	retired := 0
	for _, asset := range app.Assets {
		switch strings.TrimSpace(asset.Lifecycle) {
		case "maintenance":
			maintenance++
		case "retired":
			retired++
		}
	}
	return maintenance, retired
}

func filterCmdbApplications(apps []resps.CmdbApplicationResp, form *forms.SearchCmdbApplicationForm) []resps.CmdbApplicationResp {
	q := strings.ToLower(strings.TrimSpace(form.Q))
	risk := strings.TrimSpace(form.Risk)
	if q == "" && risk == "" {
		return apps
	}
	filtered := make([]resps.CmdbApplicationResp, 0, len(apps))
	for _, app := range apps {
		if risk != "" && app.RiskLevel != risk {
			continue
		}
		if q != "" && !cmdbApplicationMatches(app, q) {
			continue
		}
		filtered = append(filtered, app)
	}
	return filtered
}

func cmdbApplicationMatches(app resps.CmdbApplicationResp, q string) bool {
	if strings.Contains(strings.ToLower(app.Application), q) {
		return true
	}
	for _, item := range app.Owner {
		if strings.Contains(strings.ToLower(item), q) {
			return true
		}
	}
	for _, item := range app.BusinessLine {
		if strings.Contains(strings.ToLower(item), q) {
			return true
		}
	}
	return false
}

func sortCmdbApplications(apps []resps.CmdbApplicationResp) {
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].RiskScore == apps[j].RiskScore {
			return apps[i].Application < apps[j].Application
		}
		return apps[i].RiskScore > apps[j].RiskScore
	})
}

func cmdbApplicationPageRange(pageNum, pageSize, total int) (int, int) {
	if pageNum <= 0 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 15
	}
	start := (pageNum - 1) * pageSize
	if start > total {
		return total, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}
