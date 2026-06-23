// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"path"
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
)

const (
	cmdbAssetExportLimit              = 10000
	cmdbAssetRelationDefaultLimit     = 200
	cmdbAssetRelationMaxLimit         = 1000
	cmdbAssetRelationReturnedSortSize = 20
)

type iacResourceForCmdb struct {
	ResourceId   models.Id       `gorm:"column:resource_id"`
	OrgId        models.Id       `gorm:"column:org_id"`
	ProjectId    models.Id       `gorm:"column:project_id"`
	EnvId        models.Id       `gorm:"column:env_id"`
	TaskId       models.Id       `gorm:"column:task_id"`
	ResId        models.Id       `gorm:"column:res_id"`
	Provider     string          `gorm:"column:provider"`
	Module       string          `gorm:"column:module"`
	Address      string          `gorm:"column:address"`
	Type         string          `gorm:"column:type"`
	Name         string          `gorm:"column:name"`
	Attrs        models.ResAttrs `gorm:"column:attrs"`
	Dependencies models.StrSlice `gorm:"column:dependencies"`
	AppliedAt    models.Time     `gorm:"column:applied_at"`
}

type CmdbAssetExportFile struct {
	Data        []byte
	Filename    string
	ContentType string
}

func SearchCmdbAssets(c *ctx.ServiceContext, form *forms.SearchCmdbAssetForm) (interface{}, e.Error) {
	if _, err := BackfillCmdbAssetsFromIac(c); err != nil {
		return nil, err
	}
	if err := RefreshCmdbAssetGovernanceFields(c); err != nil {
		return nil, err
	}

	query := buildCmdbAssetQuery(c)
	query = applyCmdbAssetSearch(query, form)
	if form.SortField() == "" {
		query = query.Order("iac_cmdb_asset.updated_at desc")
	} else {
		query = form.Order(query)
	}

	assets := make([]resps.CmdbAssetResp, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&assets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	sanitizeCmdbAssetResps(assets)

	return &page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     assets,
	}, nil
}

func CmdbAssetDetail(c *ctx.ServiceContext, form *forms.CmdbAssetParam) (*resps.CmdbAssetResp, e.Error) {
	if _, err := BackfillCmdbAssetsFromIac(c); err != nil {
		return nil, err
	}
	if err := RefreshCmdbAssetGovernanceFields(c); err != nil {
		return nil, err
	}

	asset := resps.CmdbAssetResp{}
	query := buildCmdbAssetQuery(c).Where("iac_cmdb_asset.id = ?", form.Id)
	if err := query.First(&asset); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, err)
	}
	if err := fillCmdbAssetDetail(c, &asset); err != nil {
		return nil, err
	}
	sanitizeCmdbAssetResp(&asset)
	return &asset, nil
}

func CmdbAssetRelations(c *ctx.ServiceContext, form *forms.CmdbAssetRelationsForm) (*resps.CmdbAssetRelationsResp, e.Error) {
	if _, err := BackfillCmdbAssetsFromIac(c); err != nil {
		return nil, err
	}
	if err := RefreshCmdbAssetGovernanceFields(c); err != nil {
		return nil, err
	}

	asset := resps.CmdbAssetResp{}
	query := buildCmdbAssetQuery(c).Where("iac_cmdb_asset.id = ?", form.Id)
	if err := query.First(&asset); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, err)
	}
	includeApplication := true
	if form.IncludeApplication != nil {
		includeApplication = *form.IncludeApplication
	}
	offset := normalizeCmdbAssetRelationCursor(form.Cursor, form.Offset)
	return loadCmdbAssetRelations(c, &asset, normalizeCmdbAssetRelationLimit(form.Limit), offset, includeApplication, cmdbAssetRelationFilterFromForm(form))
}

func ExportCmdbAssets(c *ctx.ServiceContext, form *forms.ExportCmdbAssetForm) (*CmdbAssetExportFile, e.Error) {
	if _, err := BackfillCmdbAssetsFromIac(c); err != nil {
		return nil, err
	}
	if err := RefreshCmdbAssetGovernanceFields(c); err != nil {
		return nil, err
	}
	if err := ensureCmdbAssetExportPermission(c, form); err != nil {
		return nil, err
	}

	format := strings.ToLower(strings.TrimSpace(form.Format))
	if format == "" {
		format = "csv"
	}

	query := buildCmdbAssetExportQuery(c, form)
	total, err := query.Count()
	if err != nil {
		return nil, e.New(e.DBError, err)
	}
	if total > cmdbAssetExportLimit {
		return nil, e.New(e.BadParam, fmt.Errorf("export asset count %d exceeds limit %d", total, cmdbAssetExportLimit))
	}

	assets := make([]resps.CmdbAssetResp, 0)
	if err := buildCmdbAssetExportQuery(c, form).
		Order("iac_cmdb_asset.updated_at desc").
		Limit(cmdbAssetExportLimit).
		Scan(&assets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	sanitizeCmdbAssetResps(assets)

	if format == "json" {
		return buildCmdbAssetJSONExportFile(assets)
	}
	return buildCmdbAssetCSVExportFile(assets)
}

func CmdbAssetImportTemplate() (*CmdbAssetExportFile, e.Error) {
	payload := models.ResAttrs{
		"overwriteOwnership": true,
		"assets": []models.ResAttrs{
			{
				"source":         models.CmdbAssetSourceCloudCollect,
				"provider":       "aws",
				"accountId":      "123456789012",
				"region":         "us-east-1",
				"assetType":      models.CmdbAssetTypeComputeInstance,
				"nativeType":     "aws_instance",
				"nativeId":       "i-0123456789abcdef0",
				"name":           "demo-ec2",
				"status":         "running",
				"owner":          "sre",
				"application":    "demo-app",
				"businessLine":   "platform",
				"lifecycle":      "active",
				"cost":           0,
				"complianceRisk": "low",
				"tags": models.ResAttrs{
					"env": "dev",
				},
				"attributes": models.ResAttrs{
					"instanceType": "t3.micro",
				},
				"rawData": models.ResAttrs{
					"importSource": "template",
				},
			},
		},
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, e.New(e.JSONParseError, err)
	}
	return &CmdbAssetExportFile{
		Data:        data,
		Filename:    "cloudiac-cmdb-import-template.json",
		ContentType: "application/json; charset=utf-8",
	}, nil
}

func ImportCmdbAssets(c *ctx.ServiceContext, form *forms.ImportCmdbAssetForm) (*resps.CmdbImportResp, e.Error) {
	if err := ensureCmdbAssetImportPermission(c); err != nil {
		return nil, err
	}
	if form.DryRun {
		return PreviewCmdbAssetImport(c, form)
	}
	resp := newCmdbImportResp(len(form.Assets), false)
	for i := range form.Assets {
		asset := form.Assets[i]
		item := resps.CmdbImportItemResp{Index: i + 1}
		if err := prepareCmdbImportAsset(c, &asset); err != nil {
			resp.Skipped++
			item.Action = "error"
			item.Error = err.Error()
			resp.Errors = append(resp.Errors, fmt.Sprintf("第 %d 条预处理失败：%s", i+1, err.Error()))
			resp.Items = append(resp.Items, item)
			continue
		}
		item = cmdbImportItemResp(i, asset)
		if asset.NativeId == "" {
			resp.Skipped++
			item.Action = "error"
			item.Error = "缺少资源ID"
			resp.Errors = append(resp.Errors, fmt.Sprintf("第 %d 条缺少资源ID，已跳过", i+1))
			resp.Items = append(resp.Items, item)
			continue
		}

		result, err := upsertCmdbAsset(c, &asset, models.CmdbAssetChangeSourceImport)
		if err != nil {
			item.Action = "error"
			item.Error = err.Error()
			resp.Errors = append(resp.Errors, fmt.Sprintf("第 %d 条导入失败：%s", i+1, err.Error()))
			resp.Items = append(resp.Items, item)
			continue
		}
		item.AssetId = asset.Id
		switch {
		case result.Created:
			resp.Created++
			item.Action = "create"
		case result.Updated:
			resp.Updated++
			item.Action = "update"
		default:
			resp.Skipped++
			item.Action = "skip"
		}

		if form.OverwriteOwnership {
			updated, err := updateCmdbAssetOwnershipFromImport(c, asset.Id, asset)
			if err != nil {
				item.Action = "error"
				item.Error = err.Error()
				resp.Errors = append(resp.Errors, fmt.Sprintf("第 %d 条归属更新失败：%s", i+1, err.Error()))
				resp.Items = append(resp.Items, item)
				continue
			}
			if updated {
				resp.OwnershipUpdated++
				item.OwnershipChanged = true
				if item.Action == "skip" {
					item.Action = "ownership_update"
				}
			}
		}
		resp.Items = append(resp.Items, item)
	}
	return resp, nil
}

func PreviewCmdbAssetImport(c *ctx.ServiceContext, form *forms.ImportCmdbAssetForm) (*resps.CmdbImportResp, e.Error) {
	resp := newCmdbImportResp(len(form.Assets), true)
	for i := range form.Assets {
		asset := form.Assets[i]
		item := resps.CmdbImportItemResp{Index: i + 1}
		if err := prepareCmdbImportAsset(c, &asset); err != nil {
			resp.Skipped++
			item.Action = "error"
			item.Error = err.Error()
			resp.Errors = append(resp.Errors, fmt.Sprintf("第 %d 条预处理失败：%s", i+1, err.Error()))
			resp.Items = append(resp.Items, item)
			continue
		}
		item = cmdbImportItemResp(i, asset)
		if asset.NativeId == "" {
			resp.Skipped++
			item.Action = "error"
			item.Error = "缺少资源ID"
			resp.Errors = append(resp.Errors, fmt.Sprintf("第 %d 条缺少资源ID，已跳过", i+1))
			resp.Items = append(resp.Items, item)
			continue
		}

		existing, exists, err := findCmdbImportExistingAsset(c, asset)
		if err != nil {
			resp.Skipped++
			item.Action = "error"
			item.Error = err.Error()
			resp.Errors = append(resp.Errors, fmt.Sprintf("第 %d 条查询现有资产失败：%s", i+1, err.Error()))
			resp.Items = append(resp.Items, item)
			continue
		}
		if !exists {
			resp.Created++
			item.Action = "create"
			item.Diff = models.ResAttrs{"after": cmdbAssetSnapshot(&asset)}
			resp.Items = append(resp.Items, item)
			continue
		}

		asset.Id = existing.Id
		item.AssetId = existing.Id
		item.ExistingName = firstNonEmpty(existing.Name, existing.NativeId)
		item.Diff = cmdbAssetDiff(existing, &asset)
		if form.OverwriteOwnership {
			after := cmdbImportOwnershipAsset(existing, asset)
			item.OwnershipDiff = cmdbAssetOwnershipDiff(existing, after)
			item.OwnershipChanged = len(item.OwnershipDiff) > 0
		}
		switch {
		case len(item.Diff) > 0:
			resp.Updated++
			item.Action = "update"
			if item.OwnershipChanged {
				resp.OwnershipUpdated++
			}
		case item.OwnershipChanged:
			resp.OwnershipUpdated++
			item.Action = "ownership_update"
		default:
			resp.Skipped++
			item.Action = "skip"
		}
		resp.Items = append(resp.Items, item)
	}
	return resp, nil
}

func newCmdbImportResp(total int, dryRun bool) *resps.CmdbImportResp {
	return &resps.CmdbImportResp{
		Total:  total,
		DryRun: dryRun,
		Errors: make([]string, 0),
		Items:  make([]resps.CmdbImportItemResp, 0, total),
	}
}

func prepareCmdbImportAsset(c *ctx.ServiceContext, asset *models.CmdbAsset) e.Error {
	asset.OrgId = c.OrgId
	asset.Id = ""
	asset.NativeId = strings.TrimSpace(asset.NativeId)
	if asset.NativeId == "" {
		return nil
	}
	if asset.Source == "" {
		asset.Source = models.CmdbAssetSourceCloudCollect
	}
	if asset.AssetType == "" {
		asset.AssetType = NormalizeCmdbAssetType(asset.NativeType)
	}
	normalizeCmdbAssetAttrs(asset)
	return prepareCmdbAssetGovernance(c, asset)
}

func cmdbImportItemResp(index int, asset models.CmdbAsset) resps.CmdbImportItemResp {
	return resps.CmdbImportItemResp{
		Index:      index + 1,
		AssetId:    asset.Id,
		Provider:   asset.Provider,
		AccountId:  asset.AccountId,
		Region:     asset.Region,
		AssetType:  asset.AssetType,
		NativeType: asset.NativeType,
		NativeId:   asset.NativeId,
		Name:       firstNonEmpty(asset.Name, asset.NativeId),
	}
}

func findCmdbImportExistingAsset(c *ctx.ServiceContext, asset models.CmdbAsset) (models.CmdbAsset, bool, e.Error) {
	existing := models.CmdbAsset{}
	err := c.DB().Where("org_id = ? and source = ? and provider = ? and account_id = ? and region = ? and native_id = ?",
		asset.OrgId, asset.Source, asset.Provider, asset.AccountId, asset.Region, asset.NativeId).First(&existing)
	if err != nil {
		if e.IsRecordNotFound(err) {
			return existing, false, nil
		}
		return existing, false, e.New(e.DBError, err)
	}
	return existing, true, nil
}

func cmdbImportOwnershipAsset(current models.CmdbAsset, imported models.CmdbAsset) models.CmdbAsset {
	after := current
	after.Owner = strings.TrimSpace(imported.Owner)
	after.Application = strings.TrimSpace(imported.Application)
	after.BusinessLine = strings.TrimSpace(imported.BusinessLine)
	after.Lifecycle = strings.TrimSpace(imported.Lifecycle)
	after.Cost = imported.Cost
	after.ComplianceRisk = strings.TrimSpace(imported.ComplianceRisk)
	return after
}

func UpdateCmdbAssetOwnership(c *ctx.ServiceContext, form *forms.UpdateCmdbAssetOwnershipForm) (*resps.CmdbAssetResp, e.Error) {
	if _, err := BackfillCmdbAssetsFromIac(c); err != nil {
		return nil, err
	}

	asset := resps.CmdbAssetResp{}
	query := buildCmdbAssetQuery(c).Where("iac_cmdb_asset.id = ?", form.Id)
	if err := query.First(&asset); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, err)
	}
	if err := ensureCmdbAssetOwnershipPermission(c, asset.CmdbAsset, cmdbAssetOwnershipFields(form)); err != nil {
		return nil, err
	}

	attrs, after := cmdbAssetOwnershipUpdateAttrs(form, asset.CmdbAsset)
	if len(attrs) == 0 {
		return CmdbAssetDetail(c, &forms.CmdbAssetParam{Id: form.Id})
	}

	diff := cmdbAssetOwnershipDiff(asset.CmdbAsset, after)
	if len(diff) == 0 {
		return CmdbAssetDetail(c, &forms.CmdbAssetParam{Id: form.Id})
	}

	if _, err := c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and id = ?", c.OrgId, form.Id).
		UpdateAttrs(attrs); err != nil {
		return nil, e.New(e.DBError, err)
	}

	if err := recordCmdbAssetChange(c, c.OrgId, form.Id, models.CmdbAssetChangeTypeUpdated, models.CmdbAssetChangeSourceManual, diff); err != nil {
		return nil, err
	}

	return CmdbAssetDetail(c, &forms.CmdbAssetParam{Id: form.Id})
}

func BatchUpdateCmdbAssetOwnership(c *ctx.ServiceContext, form *forms.BatchUpdateCmdbAssetOwnershipForm) (*resps.CmdbBatchOwnershipResp, e.Error) {
	if _, err := BackfillCmdbAssetsFromIac(c); err != nil {
		return nil, err
	}
	projectEnvBinding, err := resolveCmdbBatchProjectEnvBinding(c, form)
	if err != nil {
		return nil, err
	}

	resp := &resps.CmdbBatchOwnershipResp{
		Total:  len(form.Ids),
		Errors: make([]string, 0),
	}
	attrsBuilder := func(asset models.CmdbAsset) (map[string]interface{}, models.CmdbAsset) {
		attrs := make(map[string]interface{})
		after := asset
		if projectEnvBinding.Enabled {
			after.ProjectId = projectEnvBinding.ProjectId
			after.EnvId = projectEnvBinding.EnvId
			after.ManagedBy = inferCmdbManagedBy(&after)
			attrs["project_id"] = after.ProjectId
			attrs["env_id"] = after.EnvId
			attrs["managed_by"] = after.ManagedBy
		}
		if form.HasKey("owner") {
			after.Owner = strings.TrimSpace(form.Owner)
			attrs["owner"] = after.Owner
		}
		if form.HasKey("application") {
			after.Application = strings.TrimSpace(form.Application)
			attrs["application"] = after.Application
		}
		if form.HasKey("businessLine") {
			after.BusinessLine = strings.TrimSpace(form.BusinessLine)
			attrs["business_line"] = after.BusinessLine
		}
		if form.HasKey("lifecycle") {
			after.Lifecycle = strings.TrimSpace(form.Lifecycle)
			attrs["lifecycle"] = after.Lifecycle
		}
		if form.HasKey("complianceRisk") {
			after.ComplianceRisk = strings.TrimSpace(form.ComplianceRisk)
			attrs["compliance_risk"] = after.ComplianceRisk
		}
		return attrs, after
	}

	if attrs, _ := attrsBuilder(models.CmdbAsset{}); len(attrs) == 0 {
		return nil, e.New(e.BadParam, fmt.Errorf("at least one ownership field is required"))
	}
	fields := cmdbBatchOwnershipFields(form, projectEnvBinding)

	assets := make([]models.CmdbAsset, 0)
	if err := buildCmdbAssetQuery(c).
		Where("iac_cmdb_asset.id in (?)", form.Ids).
		Scan(&assets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	assetById := make(map[models.Id]models.CmdbAsset, len(assets))
	for _, asset := range assets {
		assetById[asset.Id] = asset
	}

	operation := &models.CloudOperation{
		OrgId:         c.OrgId,
		ProjectId:     projectEnvBinding.ProjectId,
		EnvId:         projectEnvBinding.EnvId,
		CreatorId:     c.UserId,
		Name:          "批量治理云资产",
		OperationType: models.CloudOperationTypeGovernance,
		Action:        models.CloudOperationActionGovernanceOwnership,
		Status:        models.CloudOperationStatusRunning,
		RiskLevel:     models.CloudOperationRiskLow,
		Message:       "正在记录批量治理结果",
		Params:        cmdbBatchOwnershipOperationParams(form, projectEnvBinding),
	}
	if len(assets) == 1 {
		operation.AssetId = assets[0].Id
		operation.CloudAccountId = assets[0].CloudAccountId
		operation.Provider = assets[0].Provider
		operation.ResourceType = assets[0].AssetType
		operation.ResourceId = assets[0].NativeId
		operation.ResourceName = assets[0].Name
	}
	if err := createCloudOperation(c, operation, "创建批量资产治理任务"); err != nil {
		return nil, err
	}
	resp.OperationId = operation.Id

	for _, id := range form.Ids {
		asset, ok := assetById[id]
		if !ok {
			resp.Skipped++
			resp.Errors = append(resp.Errors, fmt.Sprintf("资产 %s 不存在或无权限", id))
			continue
		}
		if err := ensureCmdbAssetOwnershipPermission(c, asset, fields); err != nil {
			resp.Skipped++
			resp.Errors = append(resp.Errors, fmt.Sprintf("资产 %s 无权限：%s", id, err.Error()))
			continue
		}
		attrs, after := attrsBuilder(asset)
		diff := cmdbAssetOwnershipDiff(asset, after)
		if len(diff) == 0 {
			resp.Skipped++
			continue
		}
		attrs["last_operation_id"] = operation.Id
		if _, err := c.DB().Model(&models.CmdbAsset{}).
			Where("org_id = ? and id = ?", c.OrgId, id).
			UpdateAttrs(attrs); err != nil {
			resp.Skipped++
			resp.Errors = append(resp.Errors, fmt.Sprintf("资产 %s 更新失败：%s", id, err.Error()))
			continue
		}
		if err := recordCmdbAssetChange(c, c.OrgId, id, models.CmdbAssetChangeTypeUpdated, models.CmdbAssetChangeSourceManual, diff); err != nil {
			resp.Skipped++
			resp.Errors = append(resp.Errors, fmt.Sprintf("资产 %s 变更记录失败：%s", id, err.Error()))
			continue
		}
		resp.Updated++
	}

	status := models.CloudOperationStatusComplete
	message := "批量治理完成"
	if resp.Updated == 0 && len(resp.Errors) > 0 {
		status = models.CloudOperationStatusFailed
		message = "批量治理失败"
	} else if len(resp.Errors) > 0 {
		message = "批量治理完成，存在部分失败"
	}
	if err := finishCloudOperation(c, operation, status, message, models.ResAttrs{
		"total":   resp.Total,
		"updated": resp.Updated,
		"skipped": resp.Skipped,
		"errors":  resp.Errors,
	}); err != nil {
		return nil, err
	}

	return resp, nil
}

func cmdbBatchOwnershipOperationParams(form *forms.BatchUpdateCmdbAssetOwnershipForm, binding cmdbProjectEnvBinding) models.ResAttrs {
	params := models.ResAttrs{
		"assetIds": cmdbIdStrings(form.Ids),
		"fields":   cmdbBatchOwnershipFields(form, binding),
	}
	if binding.Enabled {
		params["projectId"] = binding.ProjectId.String()
		params["envId"] = binding.EnvId.String()
	}
	if form.HasKey("owner") {
		params["owner"] = strings.TrimSpace(form.Owner)
	}
	if form.HasKey("application") {
		params["application"] = strings.TrimSpace(form.Application)
	}
	if form.HasKey("businessLine") {
		params["businessLine"] = strings.TrimSpace(form.BusinessLine)
	}
	if form.HasKey("lifecycle") {
		params["lifecycle"] = strings.TrimSpace(form.Lifecycle)
	}
	if form.HasKey("complianceRisk") {
		params["complianceRisk"] = strings.TrimSpace(form.ComplianceRisk)
	}
	return params
}

func cmdbBatchOwnershipFields(form *forms.BatchUpdateCmdbAssetOwnershipForm, binding cmdbProjectEnvBinding) []string {
	fields := make([]string, 0)
	if binding.Enabled {
		fields = append(fields, "projectId", "envId", "managedBy")
	}
	if form.HasKey("owner") {
		fields = append(fields, "owner")
	}
	if form.HasKey("application") {
		fields = append(fields, "application")
	}
	if form.HasKey("businessLine") {
		fields = append(fields, "businessLine")
	}
	if form.HasKey("lifecycle") {
		fields = append(fields, "lifecycle")
	}
	if form.HasKey("complianceRisk") {
		fields = append(fields, "complianceRisk")
	}
	return fields
}

func cmdbAssetOwnershipFields(form *forms.UpdateCmdbAssetOwnershipForm) []string {
	fields := make([]string, 0)
	if form.HasKey("owner") {
		fields = append(fields, "owner")
	}
	if form.HasKey("application") {
		fields = append(fields, "application")
	}
	if form.HasKey("businessLine") {
		fields = append(fields, "businessLine")
	}
	if form.HasKey("lifecycle") {
		fields = append(fields, "lifecycle")
	}
	if form.HasKey("cost") {
		fields = append(fields, "cost")
	}
	if form.HasKey("complianceRisk") {
		fields = append(fields, "complianceRisk")
	}
	return fields
}

func cmdbIdStrings(ids []models.Id) []string {
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, id.String())
	}
	return values
}

type cmdbProjectEnvBinding struct {
	Enabled   bool
	ProjectId models.Id
	EnvId     models.Id
}

func resolveCmdbBatchProjectEnvBinding(c *ctx.ServiceContext, form *forms.BatchUpdateCmdbAssetOwnershipForm) (cmdbProjectEnvBinding, e.Error) {
	hasProjectKey := form.HasKey("projectId")
	hasEnvKey := form.HasKey("envId")
	if !hasProjectKey && !hasEnvKey {
		return cmdbProjectEnvBinding{}, nil
	}

	binding := cmdbProjectEnvBinding{
		Enabled:   true,
		ProjectId: form.ProjectId,
		EnvId:     form.EnvId,
	}
	if binding.EnvId != "" {
		env := models.Env{}
		if err := c.DB().Model(&models.Env{}).
			Where("id = ? and org_id = ?", form.EnvId, c.OrgId).
			First(&env); err != nil {
			if e.IsRecordNotFound(err) {
				return binding, e.New(e.BadParam, fmt.Errorf("环境 %s 不存在或不属于当前组织", form.EnvId))
			}
			return binding, e.New(e.DBError, err)
		}
		if binding.ProjectId == "" {
			binding.ProjectId = env.ProjectId
		}
		if binding.ProjectId != env.ProjectId {
			return binding, e.New(e.BadParam, fmt.Errorf("环境 %s 不属于项目 %s", form.EnvId, binding.ProjectId))
		}
	}
	if binding.ProjectId != "" {
		if err := ensureCmdbProjectBindingAllowed(c, binding.ProjectId); err != nil {
			return binding, err
		}
	}
	return binding, nil
}

func ensureCmdbProjectBindingAllowed(c *ctx.ServiceContext, projectId models.Id) e.Error {
	project := models.Project{}
	if err := c.DB().Model(&models.Project{}).
		Where("id = ? and org_id = ?", projectId, c.OrgId).
		First(&project); err != nil {
		if e.IsRecordNotFound(err) {
			return e.New(e.BadParam, fmt.Errorf("项目 %s 不存在或不属于当前组织", projectId))
		}
		return e.New(e.DBError, err)
	}
	if c.IsSuperAdmin || services.UserHasOrgRole(c.UserId, c.OrgId, consts.OrgRoleAdmin) {
		return nil
	}
	if !cmdbUserHasProjectRole(c, projectId, consts.ProjectRoleManager) {
		return e.New(e.BadParam, fmt.Errorf("绑定到项目 %s 需要项目负责人、组织管理员或平台管理员权限", projectId))
	}
	return nil
}

func CmdbAssetFilters(c *ctx.ServiceContext, form *forms.SearchCmdbAssetForm) (*resps.CmdbAssetFilterResp, e.Error) {
	if _, err := BackfillCmdbAssetsFromIac(c); err != nil {
		return nil, err
	}
	if err := RefreshCmdbAssetGovernanceFields(c); err != nil {
		return nil, err
	}

	baseQuery := buildCmdbAssetQuery(c)
	baseQuery = applyCmdbAssetSearch(baseQuery, form)

	resp := &resps.CmdbAssetFilterResp{
		Projects:   make([]resps.OrgProjectResp, 0),
		Envs:       make([]resps.EnvResp, 0),
		Providers:  make([]string, 0),
		AccountIds: make([]string, 0),
		AssetTypes: make([]string, 0),
		Sources:    make([]string, 0),
		Statuses:   make([]string, 0),
		ManagedBy:  make([]string, 0),
	}

	if err := fillCmdbProjectEnvFilters(c, resp); err != nil {
		return nil, err
	}
	if err := c.DB().Raw("select provider from (?) as t where provider <> '' group by provider", baseQuery.Expr()).Scan(&resp.Providers); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if err := c.DB().Raw("select account_id from (?) as t where account_id <> '' group by account_id", baseQuery.Expr()).Scan(&resp.AccountIds); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if err := c.DB().Raw("select asset_type from (?) as t where asset_type <> '' group by asset_type", baseQuery.Expr()).Scan(&resp.AssetTypes); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if err := c.DB().Raw("select source from (?) as t where source <> '' group by source", baseQuery.Expr()).Scan(&resp.Sources); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if err := c.DB().Raw("select status from (?) as t where status <> '' group by status", baseQuery.Expr()).Scan(&resp.Statuses); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if err := c.DB().Raw(fmt.Sprintf(`select managed_by from (
		select %s as managed_by from (?) as t
	) as m where managed_by <> '' group by managed_by`, cmdbManagedByExpr("")), baseQuery.Expr()).Scan(&resp.ManagedBy); err != nil {
		return nil, e.New(e.DBError, err)
	}

	return resp, nil
}

func fillCmdbProjectEnvFilters(c *ctx.ServiceContext, resp *resps.CmdbAssetFilterResp) e.Error {
	projectQuery := c.DB().Model(&models.Project{}).
		Select("iac_project.id as project_id, iac_project.name as project_name").
		Where("iac_project.org_id = ? and iac_project.status = ?", c.OrgId, models.Enable).
		Order("iac_project.name asc")
	envQuery := c.DB().Model(&models.Env{}).
		Select("iac_env.id as env_id, iac_env.name as env_name").
		Joins("join iac_project on iac_project.id = iac_env.project_id").
		Where("iac_env.org_id = ? and iac_project.status = ?", c.OrgId, models.Enable).
		Order("iac_project.name asc, iac_env.name asc")

	if !c.IsSuperAdmin && !services.UserHasOrgRole(c.UserId, c.OrgId, consts.OrgRoleAdmin) {
		projectQuery = projectQuery.
			Joins("join iac_user_project on iac_user_project.project_id = iac_project.id").
			Where("iac_user_project.user_id = ?", c.UserId)
		envQuery = envQuery.
			Joins("join iac_user_project on iac_user_project.project_id = iac_env.project_id").
			Where("iac_user_project.user_id = ?", c.UserId)
	}
	if err := projectQuery.Scan(&resp.Projects); err != nil {
		return e.New(e.DBError, err)
	}
	if err := envQuery.Scan(&resp.Envs); err != nil {
		return e.New(e.DBError, err)
	}
	return nil
}

func BackfillCmdbAssetsFromIac(c *ctx.ServiceContext) (*resps.CmdbBackfillResp, e.Error) {
	rows := make([]iacResourceForCmdb, 0)
	query := c.DB().Model(&models.Resource{}).
		Select(`iac_resource.id as resource_id, iac_resource.org_id, iac_resource.project_id, iac_resource.env_id,
			iac_resource.task_id, iac_resource.res_id, iac_resource.provider, iac_resource.module,
			iac_resource.address, iac_resource.type, iac_resource.name, iac_resource.attrs,
			iac_resource.dependencies, iac_resource.applied_at`).
		Joins(`join iac_env on iac_env.last_res_task_id = iac_resource.task_id and iac_env.id = iac_resource.env_id`).
		Where("iac_resource.org_id = ?", c.OrgId)
	if err := query.Scan(&rows); err != nil {
		return nil, e.New(e.DBError, err)
	}

	resp := &resps.CmdbBackfillResp{}
	for _, row := range rows {
		result, err := upsertCmdbAsset(c, cmdbAssetFromIacResource(row), models.CmdbAssetSourceIacResource)
		if err != nil {
			return nil, err
		}
		switch {
		case result.Created:
			resp.Created++
		case result.Updated:
			resp.Updated++
		default:
			resp.Skipped++
		}
	}

	if err := rebuildCmdbRelationsFromIac(c, rows); err != nil {
		return nil, err
	}

	return resp, nil
}

func buildCmdbAssetQuery(c *ctx.ServiceContext) *db.Session {
	query := c.DB().Model(&models.CmdbAsset{}).
		Joins("left join iac_project on iac_project.id = iac_cmdb_asset.project_id").
		Joins("left join iac_env on iac_env.id = iac_cmdb_asset.env_id").
		Where("iac_cmdb_asset.org_id = ?", c.OrgId).
		LazySelectAppend("iac_cmdb_asset.*", "iac_project.name as project_name", "iac_env.name as env_name")

	if !c.IsSuperAdmin && !services.UserHasOrgRole(c.UserId, c.OrgId, consts.OrgRoleAdmin) {
		query = query.Joins("left join iac_user_project on iac_user_project.project_id = iac_cmdb_asset.project_id").
			Where("(iac_cmdb_asset.project_id = '' or iac_user_project.user_id = ?)", c.UserId)
	}
	return query
}

func applyCmdbAssetSearch(query *db.Session, form *forms.SearchCmdbAssetForm) *db.Session {
	if form.Q != "" {
		q := fmt.Sprintf("%%%s%%", form.Q)
		query = query.Where(`iac_cmdb_asset.name like ? or iac_cmdb_asset.native_id like ? or
			iac_cmdb_asset.native_type like ? or iac_cmdb_asset.address like ? or
			iac_cmdb_asset.public_ip like ? or iac_cmdb_asset.private_ip like ? or
			iac_cmdb_asset.owner like ? or iac_cmdb_asset.application like ? or
			iac_cmdb_asset.business_line like ? or iac_cmdb_asset.lifecycle like ? or
			iac_cmdb_asset.compliance_risk like ? or iac_cmdb_asset.tags like ? or
			iac_cmdb_asset.attributes like ?`, q, q, q, q, q, q, q, q, q, q, q, q, q)
	}
	query = whereInCSV(query, "iac_cmdb_asset.project_id", form.ProjectIds)
	query = whereInCSV(query, "iac_cmdb_asset.env_id", form.EnvIds)
	query = whereInCSV(query, "iac_cmdb_asset.provider", form.Providers)
	query = whereInCSV(query, "iac_cmdb_asset.account_id", form.AccountIds)
	query = whereInCSV(query, "iac_cmdb_asset.asset_type", form.AssetTypes)
	query = whereInCSV(query, "iac_cmdb_asset.source", form.Sources)
	query = whereInCSV(query, "iac_cmdb_asset.status", form.Statuses)
	query = whereInManagedByCSV(query, form.ManagedBy)
	query = applyCmdbAssetDSL(query, form.Dsl)
	return query
}

func cmdbAssetOwnershipUpdateAttrs(form *forms.UpdateCmdbAssetOwnershipForm, asset models.CmdbAsset) (map[string]interface{}, models.CmdbAsset) {
	attrs := make(map[string]interface{})
	after := asset
	if form.HasKey("owner") {
		after.Owner = strings.TrimSpace(form.Owner)
		attrs["owner"] = after.Owner
	}
	if form.HasKey("application") {
		after.Application = strings.TrimSpace(form.Application)
		attrs["application"] = after.Application
	}
	if form.HasKey("businessLine") {
		after.BusinessLine = strings.TrimSpace(form.BusinessLine)
		attrs["business_line"] = after.BusinessLine
	}
	if form.HasKey("lifecycle") {
		after.Lifecycle = strings.TrimSpace(form.Lifecycle)
		attrs["lifecycle"] = after.Lifecycle
	}
	if form.HasKey("cost") {
		after.Cost = form.Cost
		attrs["cost"] = after.Cost
	}
	if form.HasKey("complianceRisk") {
		after.ComplianceRisk = strings.TrimSpace(form.ComplianceRisk)
		attrs["compliance_risk"] = after.ComplianceRisk
	}
	return attrs, after
}

func cmdbAssetOwnershipDiff(before models.CmdbAsset, after models.CmdbAsset) models.ResAttrs {
	diff := models.ResAttrs{}
	appendCmdbScalarDiff(diff, "projectId", before.ProjectId.String(), after.ProjectId.String())
	appendCmdbScalarDiff(diff, "envId", before.EnvId.String(), after.EnvId.String())
	appendCmdbScalarDiff(diff, "managedBy", before.ManagedBy, after.ManagedBy)
	appendCmdbScalarDiff(diff, "owner", before.Owner, after.Owner)
	appendCmdbScalarDiff(diff, "application", before.Application, after.Application)
	appendCmdbScalarDiff(diff, "businessLine", before.BusinessLine, after.BusinessLine)
	appendCmdbScalarDiff(diff, "lifecycle", before.Lifecycle, after.Lifecycle)
	appendCmdbScalarDiff(diff, "cost", before.Cost, after.Cost)
	appendCmdbScalarDiff(diff, "complianceRisk", before.ComplianceRisk, after.ComplianceRisk)
	return diff
}

func updateCmdbAssetOwnershipFromImport(c *ctx.ServiceContext, assetId models.Id, imported models.CmdbAsset) (bool, e.Error) {
	current := models.CmdbAsset{}
	if err := c.DB().Where("org_id = ? and id = ?", c.OrgId, assetId).First(&current); err != nil {
		return false, e.New(e.DBError, err)
	}
	after := current
	after.Owner = strings.TrimSpace(imported.Owner)
	after.Application = strings.TrimSpace(imported.Application)
	after.BusinessLine = strings.TrimSpace(imported.BusinessLine)
	after.Lifecycle = strings.TrimSpace(imported.Lifecycle)
	after.Cost = imported.Cost
	after.ComplianceRisk = strings.TrimSpace(imported.ComplianceRisk)

	diff := cmdbAssetOwnershipDiff(current, after)
	if len(diff) == 0 {
		return false, nil
	}
	if _, err := c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and id = ?", c.OrgId, assetId).
		UpdateAttrs(map[string]interface{}{
			"owner":           after.Owner,
			"application":     after.Application,
			"business_line":   after.BusinessLine,
			"lifecycle":       after.Lifecycle,
			"cost":            after.Cost,
			"compliance_risk": after.ComplianceRisk,
		}); err != nil {
		return false, e.New(e.DBError, err)
	}
	if err := recordCmdbAssetChange(c, c.OrgId, assetId, models.CmdbAssetChangeTypeUpdated, models.CmdbAssetChangeSourceImport, diff); err != nil {
		return false, err
	}
	return true, nil
}

func whereInCSV(query *db.Session, field, value string) *db.Session {
	values := splitCSV(value)
	if len(values) == 0 {
		return query
	}
	return query.Where(fmt.Sprintf("%s in (?)", field), values)
}

func whereInManagedByCSV(query *db.Session, value string) *db.Session {
	values := splitCSV(value)
	if len(values) == 0 {
		return query
	}
	clauses := make([]string, 0, len(values))
	args := make([]interface{}, 0, len(values)*2)
	for _, item := range values {
		switch item {
		case models.CmdbManagedByIac, models.CmdbManagedByCloudOnly, models.CmdbManagedByCloudLinked, models.CmdbManagedByManual:
			clauses = append(clauses, cmdbManagedByExpr("iac_cmdb_asset.")+" = ?")
			args = append(args, item)
		}
	}
	if len(clauses) == 0 {
		return query
	}
	return query.Where("("+strings.Join(clauses, " or ")+")", args...)
}

func splitCSV(value string) []string {
	items := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

type cmdbAssetRelationFilter struct {
	Sources       []string
	sourceSet     map[string]bool
	RelationTypes []string
	typeSet       map[string]bool
	Direction     string
	Keyword       string
}

func cmdbAssetRelationFilterFromForm(form *forms.CmdbAssetRelationsForm) cmdbAssetRelationFilter {
	if form == nil {
		return cmdbAssetRelationFilter{}
	}
	return newCmdbAssetRelationFilter(splitCSV(form.Sources), splitCSV(form.RelationTypes), form.Direction, form.Keyword)
}

func newCmdbAssetRelationFilter(sources []string, relationTypes []string, direction string, keyword string) cmdbAssetRelationFilter {
	return cmdbAssetRelationFilter{
		Sources:       cmdbUniqueStrings(sources),
		sourceSet:     cmdbStringSet(sources),
		RelationTypes: cmdbUniqueStrings(relationTypes),
		typeSet:       cmdbStringSet(relationTypes),
		Direction:     normalizeCmdbRelationDirection(direction),
		Keyword:       strings.ToLower(strings.TrimSpace(keyword)),
	}
}

func cmdbUniqueStrings(values []string) []string {
	seen := make(map[string]bool)
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

func cmdbStringSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range cmdbUniqueStrings(values) {
		result[value] = true
	}
	return result
}

func normalizeCmdbRelationDirection(direction string) string {
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "incoming", "outgoing":
		return strings.ToLower(strings.TrimSpace(direction))
	default:
		return "all"
	}
}

func (filter cmdbAssetRelationFilter) matchesSource(source string) bool {
	return len(filter.sourceSet) == 0 || filter.sourceSet[source]
}

func (filter cmdbAssetRelationFilter) matchesRelationType(relationType string) bool {
	return len(filter.typeSet) == 0 || filter.typeSet[relationType]
}

func (filter cmdbAssetRelationFilter) matchesDirection(relation resps.CmdbAssetRelationResp, assetId models.Id) bool {
	switch filter.Direction {
	case "incoming":
		return relation.TargetAssetId == assetId
	case "outgoing":
		return relation.SourceAssetId == assetId
	default:
		return true
	}
}

func (filter cmdbAssetRelationFilter) matchesApplicationAggregate(appRelation resps.CmdbApplicationRelationResp) bool {
	if !filter.matchesSource(models.CmdbRelationSourceAppInferred) || !filter.matchesRelationType(appRelation.RelationType) {
		return false
	}
	return filter.Direction == "all" || filter.Direction == appRelation.Direction
}

func (filter cmdbAssetRelationFilter) matchesRelation(relation resps.CmdbAssetRelationResp, assetId models.Id) bool {
	if !filter.matchesSource(relation.Source) || !filter.matchesRelationType(relation.RelationType) || !filter.matchesDirection(relation, assetId) {
		return false
	}
	if filter.Keyword == "" {
		return true
	}
	return strings.Contains(cmdbAssetRelationSearchText(relation), filter.Keyword)
}

func cmdbAssetRelationSearchText(relation resps.CmdbAssetRelationResp) string {
	values := []string{
		string(relation.SourceAssetId),
		string(relation.TargetAssetId),
		relation.SourceAssetName,
		relation.TargetAssetName,
		relation.SourceAssetType,
		relation.TargetAssetType,
		relation.Source,
		relation.RelationType,
	}
	if len(relation.Metadata) > 0 {
		if data, err := json.Marshal(relation.Metadata); err == nil {
			values = append(values, string(data))
		}
	}
	return strings.ToLower(strings.Join(values, " "))
}

func cmdbAssetFromIacResource(row iacResourceForCmdb) *models.CmdbAsset {
	provider := path.Base(row.Provider)
	nativeId := firstNonEmpty(row.ResId.String(), attrString(row.Attrs, "id"), attrString(row.Attrs, "arn"), row.ResourceId.String())
	name := firstNonEmpty(row.Name, attrString(row.Attrs, "name"), attrString(row.Attrs, "display_name"), attrString(row.Attrs, "bucket"), nativeId)

	rawData := models.ResAttrs{
		"attrs":        row.Attrs,
		"dependencies": row.Dependencies,
		"appliedAt":    row.AppliedAt,
		"iac": models.ResAttrs{
			"resourceId": row.ResourceId,
			"address":    row.Address,
			"module":     row.Module,
			"type":       row.Type,
			"provider":   row.Provider,
		},
	}

	return &models.CmdbAsset{
		OrgId:         row.OrgId,
		ProjectId:     row.ProjectId,
		EnvId:         row.EnvId,
		TaskId:        row.TaskId,
		Source:        models.CmdbAssetSourceIacResource,
		Provider:      provider,
		AccountId:     firstNonEmpty(attrString(row.Attrs, "account_id"), attrString(row.Attrs, "tenancy_id"), attrString(row.Attrs, "compartment_id")),
		Region:        firstNonEmpty(attrString(row.Attrs, "region"), attrString(row.Attrs, "availability_domain")),
		Zone:          firstNonEmpty(attrString(row.Attrs, "availability_zone"), attrString(row.Attrs, "availability_domain")),
		AssetType:     NormalizeCmdbAssetType(row.Type),
		NativeType:    row.Type,
		NativeId:      nativeId,
		Name:          name,
		Status:        firstNonEmpty(attrString(row.Attrs, "status"), attrString(row.Attrs, "state"), attrString(row.Attrs, "lifecycle_state")),
		Address:       row.Address,
		Module:        row.Module,
		PublicIp:      firstNonEmpty(attrString(row.Attrs, "public_ip"), attrString(row.Attrs, "public_ip_address"), attrString(row.Attrs, "ip_address")),
		PrivateIp:     firstNonEmpty(attrString(row.Attrs, "private_ip"), attrString(row.Attrs, "private_ip_address"), attrString(row.Attrs, "primary_private_ip")),
		IacResourceId: row.ResourceId,
		IacAddress:    row.Address,
		ManagedBy:     models.CmdbManagedByIac,
		Tags:          extractTags(row.Attrs),
		Attributes:    row.Attrs,
		RawData:       rawData,
		LastSyncAt:    row.AppliedAt,
	}
}

func rebuildCmdbRelationsFromIac(c *ctx.ServiceContext, rows []iacResourceForCmdb) e.Error {
	assets := make([]models.CmdbAsset, 0)
	if err := c.DB().Where("org_id = ? and source = ?", c.OrgId, models.CmdbAssetSourceIacResource).Find(&assets); err != nil {
		return e.New(e.DBError, err)
	}

	assetByAddress := make(map[string]models.CmdbAsset)
	for _, asset := range assets {
		if asset.ProjectId == "" || asset.EnvId == "" || asset.IacAddress == "" {
			continue
		}
		assetByAddress[cmdbIacAddressKey(asset.ProjectId, asset.EnvId, asset.IacAddress)] = asset
	}

	if _, err := c.DB().Where("org_id = ? and source = ?", c.OrgId, models.CmdbRelationSourceIacDependency).Delete(&models.CmdbAssetRelation{}); err != nil {
		return e.New(e.DBError, err)
	}

	seen := make(map[string]bool)
	for _, row := range rows {
		sourceAsset, ok := assetByAddress[cmdbIacAddressKey(row.ProjectId, row.EnvId, row.Address)]
		if !ok {
			continue
		}
		for _, dependency := range row.Dependencies {
			targetAsset, ok := assetByAddress[cmdbIacAddressKey(row.ProjectId, row.EnvId, dependency)]
			if !ok || targetAsset.Id == sourceAsset.Id {
				continue
			}
			key := fmt.Sprintf("%s/%s", sourceAsset.Id, targetAsset.Id)
			if seen[key] {
				continue
			}
			seen[key] = true
			relation := &models.CmdbAssetRelation{
				OrgId:         c.OrgId,
				SourceAssetId: sourceAsset.Id,
				TargetAssetId: targetAsset.Id,
				RelationType:  models.CmdbRelationTypeDependsOn,
				Source:        models.CmdbRelationSourceIacDependency,
				Metadata: models.ResAttrs{
					"sourceAddress": sourceAsset.IacAddress,
					"targetAddress": targetAsset.IacAddress,
				},
			}
			relation.Id = models.NewId("cir")
			if err := models.Create(c.DB(), relation); err != nil {
				return e.New(e.DBError, err)
			}
		}
	}
	return nil
}

func cmdbIacAddressKey(projectId models.Id, envId models.Id, address string) string {
	return fmt.Sprintf("%s/%s/%s", projectId, envId, address)
}

func buildCmdbAssetExportQuery(c *ctx.ServiceContext, form *forms.ExportCmdbAssetForm) *db.Session {
	query := buildCmdbAssetQuery(c)
	query = applyCmdbAssetSearch(query, &form.SearchCmdbAssetForm)
	if len(form.Ids) > 0 {
		query = query.Where("iac_cmdb_asset.id in (?)", form.Ids)
	}
	return query
}

func buildCmdbAssetCSVExportFile(assets []resps.CmdbAssetResp) (*CmdbAssetExportFile, e.Error) {
	buf := &bytes.Buffer{}
	buf.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(buf)
	headers := []string{
		"资产ID", "项目", "环境", "云厂商", "账号", "统一云账号ID", "纳管状态", "同步策略ID", "最近操作ID",
		"区域", "可用区", "资产类型", "原生类型", "资源ID",
		"名称", "状态", "来源", "负责人", "应用", "业务线", "成本中心", "生命周期", "成本", "风险评分", "合规风险",
		"公网IP", "私网IP", "IaC地址", "标签", "最近同步", "更新时间",
	}
	if err := writer.Write(headers); err != nil {
		return nil, e.New(e.JSONParseError, err)
	}

	for _, asset := range assets {
		row := []string{
			asset.Id.String(),
			asset.ProjectName,
			asset.EnvName,
			asset.Provider,
			asset.AccountId,
			asset.CloudAccountId.String(),
			asset.ManagedBy,
			asset.SyncPolicyId.String(),
			asset.LastOperationId.String(),
			asset.Region,
			asset.Zone,
			asset.AssetType,
			asset.NativeType,
			asset.NativeId,
			asset.Name,
			asset.Status,
			asset.Source,
			asset.Owner,
			asset.Application,
			asset.BusinessLine,
			asset.CostCenter,
			asset.Lifecycle,
			strconv.FormatFloat(asset.Cost, 'f', 4, 64),
			strconv.FormatFloat(asset.RiskScore, 'f', 4, 64),
			asset.ComplianceRisk,
			asset.PublicIp,
			asset.PrivateIp,
			asset.IacAddress,
			cmdbAssetExportJSONCell(asset.Tags),
			cmdbAssetExportTime(asset.LastSyncAt),
			cmdbAssetExportTime(asset.UpdatedAt),
		}
		if err := writer.Write(row); err != nil {
			return nil, e.New(e.JSONParseError, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, e.New(e.JSONParseError, err)
	}

	return &CmdbAssetExportFile{
		Data:        buf.Bytes(),
		Filename:    fmt.Sprintf("cloudiac-cmdb-assets-%s.csv", time.Now().Format("20060102150405")),
		ContentType: "text/csv; charset=utf-8",
	}, nil
}

func buildCmdbAssetJSONExportFile(assets []resps.CmdbAssetResp) (*CmdbAssetExportFile, e.Error) {
	payload := map[string]interface{}{
		"exportedAt": time.Now().Format(time.RFC3339),
		"total":      len(assets),
		"assets":     assets,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, e.New(e.JSONParseError, err)
	}
	return &CmdbAssetExportFile{
		Data:        data,
		Filename:    fmt.Sprintf("cloudiac-cmdb-assets-%s.json", time.Now().Format("20060102150405")),
		ContentType: "application/json; charset=utf-8",
	}, nil
}

func cmdbAssetExportJSONCell(value interface{}) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

func cmdbAssetExportTime(value models.Time) string {
	t := time.Time(value)
	if t.IsZero() || t.Year() <= 1 {
		return ""
	}
	return t.Format(time.RFC3339)
}

const cmdbAssetMaskedValue = "<masked>"

func sanitizeCmdbAssetResps(assets []resps.CmdbAssetResp) {
	for i := range assets {
		sanitizeCmdbAssetResp(&assets[i])
	}
}

func sanitizeCmdbAssetResp(asset *resps.CmdbAssetResp) {
	if asset == nil {
		return
	}
	asset.Tags = sanitizeCmdbAssetAttrs(asset.Tags)
	asset.Attributes = sanitizeCmdbAssetAttrs(asset.Attributes)
	asset.RawData = sanitizeCmdbAssetAttrs(asset.RawData)
	sanitizeCmdbRelationResps(asset.Relations)
}

func sanitizeCmdbRelationResps(relations []resps.CmdbAssetRelationResp) {
	for i := range relations {
		relations[i].Metadata = sanitizeCmdbAssetAttrs(relations[i].Metadata)
	}
}

func sanitizeCmdbAssetAttrs(attrs models.ResAttrs) models.ResAttrs {
	sanitized, _ := sanitizeCmdbAssetValue(attrs, "")
	if result, ok := sanitized.(models.ResAttrs); ok {
		return result
	}
	if result, ok := sanitized.(map[string]interface{}); ok {
		return models.ResAttrs(result)
	}
	return models.ResAttrs{}
}

func sanitizeCmdbAssetValue(value interface{}, key string) (interface{}, bool) {
	if cmdbAssetSensitiveFieldName(key) {
		return cmdbAssetMaskedValue, true
	}
	switch typed := value.(type) {
	case models.ResAttrs:
		result := make(models.ResAttrs, len(typed))
		changed := false
		for childKey, childValue := range typed {
			next, masked := sanitizeCmdbAssetValue(childValue, childKey)
			result[childKey] = next
			changed = changed || masked
		}
		return result, changed
	case map[string]interface{}:
		result := make(models.ResAttrs, len(typed))
		changed := false
		for childKey, childValue := range typed {
			next, masked := sanitizeCmdbAssetValue(childValue, childKey)
			result[childKey] = next
			changed = changed || masked
		}
		return result, changed
	case []models.ResAttrs:
		result := make([]models.ResAttrs, 0, len(typed))
		changed := false
		for _, item := range typed {
			next := sanitizeCmdbAssetAttrs(item)
			result = append(result, next)
			changed = true
		}
		return result, changed
	case []interface{}:
		result := make([]interface{}, 0, len(typed))
		changed := false
		for _, item := range typed {
			next, masked := sanitizeCmdbAssetValue(item, "")
			result = append(result, next)
			changed = changed || masked
		}
		return result, changed
	default:
		return value, false
	}
}

func cmdbAssetSensitiveFieldName(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}
	replacer := strings.NewReplacer("-", "", "_", "", ".", "", " ", "")
	compact := replacer.Replace(normalized)
	sensitiveParts := []string{
		"accesskey",
		"accountkey",
		"apikey",
		"apisecret",
		"authorization",
		"authtoken",
		"clientsecret",
		"credential",
		"password",
		"passphrase",
		"privatekey",
		"secretaccesskey",
		"sessiontoken",
		"signature",
		"token",
	}
	for _, part := range sensitiveParts {
		if strings.Contains(compact, part) {
			return true
		}
	}
	return false
}

func fillCmdbAssetDetail(c *ctx.ServiceContext, asset *resps.CmdbAssetResp) e.Error {
	relationResp, err := loadCmdbAssetRelations(c, asset, cmdbAssetRelationDefaultLimit, 0, true, cmdbAssetRelationFilter{})
	if err != nil {
		return err
	}

	changes := make([]resps.CmdbAssetChangeResp, 0)
	if err := c.DB().Model(&models.CmdbAssetChange{}).
		Where("org_id = ? and asset_id = ?", c.OrgId, asset.Id).
		Order("created_at desc").
		Limit(20).
		Scan(&changes); err != nil {
		return e.New(e.DBError, err)
	}
	for i := range changes {
		summary := normalizeCmdbText(changes[i].Summary)
		if summary == changes[i].Summary {
			continue
		}
		if _, err := c.DB().Model(&models.CmdbAssetChange{}).
			Where("org_id = ? and id = ?", c.OrgId, changes[i].Id).
			UpdateColumn("summary", summary); err != nil {
			c.Logger().Warnf("repair cmdb asset change summary encoding failed: %v", err)
		}
		changes[i].Summary = summary
	}

	asset.Relations = relationResp.Relations
	asset.RelationSummary = &relationResp.Summary
	asset.Changes = changes
	return nil
}

func loadCmdbAssetRelations(c *ctx.ServiceContext, asset *resps.CmdbAssetResp, limit int, offset int, includeApplication bool, filter cmdbAssetRelationFilter) (*resps.CmdbAssetRelationsResp, e.Error) {
	limit = normalizeCmdbAssetRelationLimit(limit)
	offset = normalizeCmdbAssetRelationOffset(offset)
	summary := resps.CmdbAssetRelationSummaryResp{
		Limit:           limit,
		Offset:          offset,
		Cursor:          strconv.Itoa(offset),
		SourceBreakdown: make([]resps.CmdbAssetRelationMetricResp, 0),
		TypeBreakdown:   make([]resps.CmdbAssetRelationMetricResp, 0),
	}
	sourceBreakdown := make(map[string]int64)
	typeBreakdown := make(map[string]int64)

	directRelations, directSummary, err := loadCmdbAssetDirectRelations(c, asset.Id, limit, offset, filter)
	if err != nil {
		return nil, err
	}
	relations := directRelations
	summary.DirectRelationCount = directSummary.TotalRelationCount
	summary.IncomingRelationCount = directSummary.IncomingRelationCount
	summary.OutgoingRelationCount = directSummary.OutgoingRelationCount
	cmdbRelationMergeBreakdown(sourceBreakdown, directSummary.SourceBreakdown)
	cmdbRelationMergeBreakdown(typeBreakdown, directSummary.TypeBreakdown)

	if includeApplication && len(relations) < limit {
		applicationOffset := 0
		if int64(offset) > directSummary.TotalRelationCount {
			applicationOffset = offset - int(directSummary.TotalRelationCount)
		}
		inferredRelations, inferredTotal, inferredSourceBreakdown, inferredTypeBreakdown, err := cmdbAssetApplicationInferredRelations(c, asset, relations, limit-len(relations), applicationOffset, filter)
		if err != nil {
			return nil, err
		}
		relations = append(relations, inferredRelations...)
		summary.ApplicationInferredRelationCount = inferredTotal
		cmdbRelationMergeBreakdown(sourceBreakdown, inferredSourceBreakdown)
		cmdbRelationMergeBreakdown(typeBreakdown, inferredTypeBreakdown)
	}

	summary.TotalRelationCount = summary.DirectRelationCount + summary.ApplicationInferredRelationCount
	summary.ReturnedRelationCount = int64(len(relations))
	summary.HasMore = summary.TotalRelationCount > int64(offset+len(relations))
	if summary.HasMore {
		summary.NextOffset = offset + len(relations)
		summary.NextCursor = strconv.Itoa(summary.NextOffset)
	}
	summary.Truncated = summary.HasMore
	summary.SourceBreakdown = cmdbRelationBreakdownList(sourceBreakdown)
	summary.TypeBreakdown = cmdbRelationBreakdownList(typeBreakdown)
	sanitizeCmdbRelationResps(relations)

	return &resps.CmdbAssetRelationsResp{
		AssetId:   asset.Id,
		Relations: relations,
		Summary:   summary,
	}, nil
}

func loadCmdbAssetDirectRelations(c *ctx.ServiceContext, assetId models.Id, limit int, offset int, filter cmdbAssetRelationFilter) ([]resps.CmdbAssetRelationResp, resps.CmdbAssetRelationSummaryResp, e.Error) {
	summary := resps.CmdbAssetRelationSummaryResp{}
	total, err := cmdbAssetDirectRelationQuery(c, assetId, filter).Count()
	if err != nil {
		return nil, summary, e.New(e.DBError, err)
	}
	summary.TotalRelationCount = total
	if filter.Direction == "all" || filter.Direction == "outgoing" {
		outgoingFilter := filter
		outgoingFilter.Direction = "outgoing"
		if summary.OutgoingRelationCount, err = cmdbAssetDirectRelationQuery(c, assetId, outgoingFilter).Count(); err != nil {
			return nil, summary, e.New(e.DBError, err)
		}
	}
	if filter.Direction == "all" || filter.Direction == "incoming" {
		incomingFilter := filter
		incomingFilter.Direction = "incoming"
		if summary.IncomingRelationCount, err = cmdbAssetDirectRelationQuery(c, assetId, incomingFilter).Count(); err != nil {
			return nil, summary, e.New(e.DBError, err)
		}
	}

	sourceRows := make([]resps.CmdbAssetRelationMetricResp, 0)
	if err := cmdbAssetDirectRelationQuery(c, assetId, filter).
		Select("iac_cmdb_asset_relation.source as name, count(*) as count").
		Group("iac_cmdb_asset_relation.source").
		Scan(&sourceRows); err != nil {
		return nil, summary, e.New(e.DBError, err)
	}
	typeRows := make([]resps.CmdbAssetRelationMetricResp, 0)
	if err := cmdbAssetDirectRelationQuery(c, assetId, filter).
		Select("iac_cmdb_asset_relation.relation_type as name, count(*) as count").
		Group("iac_cmdb_asset_relation.relation_type").
		Scan(&typeRows); err != nil {
		return nil, summary, e.New(e.DBError, err)
	}
	summary.SourceBreakdown = sourceRows
	summary.TypeBreakdown = typeRows

	relations := make([]resps.CmdbAssetRelationResp, 0)
	if limit <= 0 {
		return relations, summary, nil
	}
	if err := cmdbAssetDirectRelationQuery(c, assetId, filter).
		Select(`iac_cmdb_asset_relation.*,
			source_asset.name as source_asset_name, source_asset.asset_type as source_asset_type,
			target_asset.name as target_asset_name, target_asset.asset_type as target_asset_type`).
		Order("iac_cmdb_asset_relation.updated_at desc").
		Offset(offset).
		Limit(limit).
		Scan(&relations); err != nil {
		return nil, summary, e.New(e.DBError, err)
	}
	return relations, summary, nil
}

func cmdbAssetDirectRelationQuery(c *ctx.ServiceContext, assetId models.Id, filter cmdbAssetRelationFilter) *db.Session {
	query := c.DB().Model(&models.CmdbAssetRelation{}).
		Joins("left join iac_cmdb_asset source_asset on source_asset.id = iac_cmdb_asset_relation.source_asset_id").
		Joins("left join iac_cmdb_asset target_asset on target_asset.id = iac_cmdb_asset_relation.target_asset_id").
		Where("iac_cmdb_asset_relation.org_id = ?", c.OrgId)

	switch filter.Direction {
	case "incoming":
		query = query.Where("iac_cmdb_asset_relation.target_asset_id = ?", assetId)
	case "outgoing":
		query = query.Where("iac_cmdb_asset_relation.source_asset_id = ?", assetId)
	default:
		query = query.Where("(iac_cmdb_asset_relation.source_asset_id = ? or iac_cmdb_asset_relation.target_asset_id = ?)", assetId, assetId)
	}
	if len(filter.Sources) > 0 {
		query = query.Where("iac_cmdb_asset_relation.source in (?)", filter.Sources)
	}
	if len(filter.RelationTypes) > 0 {
		query = query.Where("iac_cmdb_asset_relation.relation_type in (?)", filter.RelationTypes)
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		query = query.Where(`(
			lower(iac_cmdb_asset_relation.source_asset_id) like ? or
			lower(iac_cmdb_asset_relation.target_asset_id) like ? or
			lower(iac_cmdb_asset_relation.source) like ? or
			lower(iac_cmdb_asset_relation.relation_type) like ? or
			lower(source_asset.name) like ? or
			lower(source_asset.native_id) like ? or
			lower(source_asset.asset_type) like ? or
			lower(source_asset.provider) like ? or
			lower(target_asset.name) like ? or
			lower(target_asset.native_id) like ? or
			lower(target_asset.asset_type) like ? or
			lower(target_asset.provider) like ?
		)`, like, like, like, like, like, like, like, like, like, like, like, like)
	}
	return query
}

func cmdbAssetApplicationInferredRelations(c *ctx.ServiceContext, asset *resps.CmdbAssetResp, existing []resps.CmdbAssetRelationResp, limit int, offset int, filter cmdbAssetRelationFilter) ([]resps.CmdbAssetRelationResp, int64, []resps.CmdbAssetRelationMetricResp, []resps.CmdbAssetRelationMetricResp, e.Error) {
	application := strings.TrimSpace(asset.Application)
	if application == "" {
		return []resps.CmdbAssetRelationResp{}, 0, nil, nil, nil
	}
	offset = normalizeCmdbAssetRelationOffset(offset)

	seen := make(map[string]bool, len(existing))
	for _, relation := range existing {
		seen[cmdbAssetRelationKey(relation.SourceAssetId, relation.TargetAssetId, relation.RelationType, relation.Source)] = true
	}

	appRelations, err := cmdbAssetApplicationRelationAggregates(c, application)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	relations := make([]resps.CmdbAssetRelationResp, 0)
	sourceBreakdown := make(map[string]int64)
	typeBreakdown := make(map[string]int64)
	totalCandidates := int64(0)
	remaining := limit
	skipped := offset
	for _, appRelation := range appRelations {
		candidates := cmdbApplicationRelationCandidateCount(appRelation)
		if candidates <= 0 || !filter.matchesApplicationAggregate(appRelation) {
			continue
		}
		if filter.Keyword == "" {
			totalCandidates += int64(candidates)
			cmdbRelationAddBreakdown(sourceBreakdown, models.CmdbRelationSourceAppInferred, int64(candidates))
			cmdbRelationAddBreakdown(typeBreakdown, appRelation.RelationType, int64(candidates))
		}
		if appRelation.Application == "" || (remaining <= 0 && filter.Keyword == "") {
			continue
		}
		if filter.Keyword == "" && skipped >= candidates {
			skipped -= candidates
			continue
		}
		assetOffset := 0
		fetchLimit := remaining + cmdbAssetRelationReturnedSortSize
		if filter.Keyword != "" {
			fetchLimit = cmdbAssetRelationMaxLimit + cmdbAssetRelationReturnedSortSize
		} else if skipped > 0 {
			assetOffset = skipped
			skipped = 0
		}
		relatedAssets, err := cmdbVisibleApplicationAssets(c, appRelation.Application, fetchLimit, assetOffset)
		if err != nil {
			return nil, 0, nil, nil, err
		}
		for _, relatedAsset := range relatedAssets {
			if remaining <= 0 && filter.Keyword == "" {
				break
			}
			if appRelation.Direction == "outgoing" {
				if relatedAsset.Id == "" || relatedAsset.Id == asset.Id {
					continue
				}
				relation := cmdbAssetApplicationInferredRelation(asset.Id, relatedAsset.Id, appRelation, application, appRelation.Application)
				key := cmdbAssetRelationKey(relation.SourceAssetId, relation.TargetAssetId, relation.RelationType, relation.Source)
				if seen[key] {
					continue
				}
				seen[key] = true
				relation.SourceAssetName = firstNonEmpty(asset.Name, asset.NativeId)
				relation.SourceAssetType = asset.AssetType
				relation.TargetAssetName = firstNonEmpty(relatedAsset.Name, relatedAsset.NativeId)
				relation.TargetAssetType = relatedAsset.AssetType
				if !filter.matchesRelation(relation, asset.Id) {
					continue
				}
				if filter.Keyword != "" {
					totalCandidates++
					cmdbRelationAddBreakdown(sourceBreakdown, models.CmdbRelationSourceAppInferred, 1)
					cmdbRelationAddBreakdown(typeBreakdown, appRelation.RelationType, 1)
				}
				if skipped > 0 {
					skipped--
					continue
				}
				if remaining <= 0 {
					continue
				}
				relations = append(relations, relation)
				remaining--
				continue
			}
			sourceAsset := relatedAsset
			if sourceAsset.Id == "" || sourceAsset.Id == asset.Id {
				continue
			}
			relation := cmdbAssetApplicationInferredRelation(sourceAsset.Id, asset.Id, appRelation, appRelation.Application, application)
			key := cmdbAssetRelationKey(relation.SourceAssetId, relation.TargetAssetId, relation.RelationType, relation.Source)
			if seen[key] {
				continue
			}
			seen[key] = true
			relation.SourceAssetName = firstNonEmpty(sourceAsset.Name, sourceAsset.NativeId)
			relation.SourceAssetType = sourceAsset.AssetType
			relation.TargetAssetName = firstNonEmpty(asset.Name, asset.NativeId)
			relation.TargetAssetType = asset.AssetType
			if !filter.matchesRelation(relation, asset.Id) {
				continue
			}
			if filter.Keyword != "" {
				totalCandidates++
				cmdbRelationAddBreakdown(sourceBreakdown, models.CmdbRelationSourceAppInferred, 1)
				cmdbRelationAddBreakdown(typeBreakdown, appRelation.RelationType, 1)
			}
			if skipped > 0 {
				skipped--
				continue
			}
			if remaining <= 0 {
				continue
			}
			relations = append(relations, relation)
			remaining--
		}
	}

	sort.Slice(relations, func(i, j int) bool {
		if relations[i].SourceAssetName == relations[j].SourceAssetName {
			return relations[i].TargetAssetName < relations[j].TargetAssetName
		}
		return relations[i].SourceAssetName < relations[j].SourceAssetName
	})
	return relations, totalCandidates, cmdbRelationBreakdownList(sourceBreakdown), cmdbRelationBreakdownList(typeBreakdown), nil
}

type cmdbApplicationAssetCountRow struct {
	Application string `gorm:"column:application"`
	Count       int    `gorm:"column:count"`
}

func cmdbAssetApplicationRelationAggregates(c *ctx.ServiceContext, application string) ([]resps.CmdbApplicationRelationResp, e.Error) {
	relations := make([]resps.CmdbApplicationRelationResp, 0)
	if err := cmdbAssetApplicationRelationAggregatesByAsset(c, application, "outgoing", &relations); err != nil {
		return nil, err
	}
	if err := cmdbAssetApplicationRelationAggregatesByAsset(c, application, "incoming", &relations); err != nil {
		return nil, err
	}
	manualRelations := make([]models.CmdbApplicationRelation, 0)
	if err := c.DB().
		Where("org_id = ? and source = ? and (source_application = ? or target_application = ?)",
			c.OrgId, models.CmdbRelationSourceManualApp, application, application).
		Find(&manualRelations); err != nil {
		return nil, e.New(e.DBError, err)
	}
	if len(manualRelations) > 0 {
		assetCounts, err := cmdbVisibleApplicationAssetCounts(c, append(cmdbManualRelationApplications(manualRelations), application))
		if err != nil {
			return nil, err
		}
		for _, relation := range manualRelations {
			sourceApp := strings.TrimSpace(relation.SourceApplication)
			targetApp := strings.TrimSpace(relation.TargetApplication)
			if sourceApp == "" || targetApp == "" || sourceApp == targetApp {
				continue
			}
			if sourceApp == application && assetCounts[targetApp] > 0 {
				relations = append(relations, resps.CmdbApplicationRelationResp{
					Application:        targetApp,
					Direction:          "outgoing",
					RelationType:       relation.RelationType,
					Source:             relation.Source,
					AssetRelationCount: 1,
					SourceAssetCount:   assetCounts[sourceApp],
					TargetAssetCount:   assetCounts[targetApp],
					LatestRelationAt:   relation.UpdatedAt,
				})
			}
			if targetApp == application && assetCounts[sourceApp] > 0 {
				relations = append(relations, resps.CmdbApplicationRelationResp{
					Application:        sourceApp,
					Direction:          "incoming",
					RelationType:       relation.RelationType,
					Source:             relation.Source,
					AssetRelationCount: 1,
					SourceAssetCount:   assetCounts[sourceApp],
					TargetAssetCount:   assetCounts[targetApp],
					LatestRelationAt:   relation.UpdatedAt,
				})
			}
		}
	}
	sort.SliceStable(relations, func(i, j int) bool {
		iCount := cmdbApplicationRelationCandidateCount(relations[i])
		jCount := cmdbApplicationRelationCandidateCount(relations[j])
		if iCount == jCount {
			if relations[i].Direction == relations[j].Direction {
				return relations[i].Application < relations[j].Application
			}
			return relations[i].Direction < relations[j].Direction
		}
		return iCount > jCount
	})
	return relations, nil
}

func cmdbAssetApplicationRelationAggregatesByAsset(c *ctx.ServiceContext, application string, direction string, out *[]resps.CmdbApplicationRelationResp) e.Error {
	sourceJoin, sourceWhere, sourceArgs := cmdbVisibleAssetSQLScope(c, "source_asset", "source_user_project")
	targetJoin, targetWhere, targetArgs := cmdbVisibleAssetSQLScope(c, "target_asset", "target_user_project")
	relatedColumn := "target_asset.application"
	currentColumn := "source_asset.application"
	relatedCondition := "target_asset.application <> '' and target_asset.application <> ?"
	if direction == "incoming" {
		relatedColumn = "source_asset.application"
		currentColumn = "target_asset.application"
		relatedCondition = "source_asset.application <> '' and source_asset.application <> ?"
	}
	sql := fmt.Sprintf(`
		select %s as application,
			? as direction,
			r.relation_type as relation_type,
			r.source as source,
			count(*) as asset_relation_count,
			count(distinct r.source_asset_id) as source_asset_count,
			count(distinct r.target_asset_id) as target_asset_count,
			max(r.updated_at) as latest_relation_at
		from iac_cmdb_asset_relation r
		join iac_cmdb_asset source_asset on source_asset.id = r.source_asset_id
		join iac_cmdb_asset target_asset on target_asset.id = r.target_asset_id
		%s
		%s
		where r.org_id = ? and source_asset.org_id = ? and target_asset.org_id = ?
			%s
			%s
			and %s = ?
			and %s
		group by %s, r.relation_type, r.source`,
		relatedColumn, sourceJoin, targetJoin, sourceWhere, targetWhere, currentColumn, relatedCondition, relatedColumn)

	args := make([]interface{}, 0)
	args = append(args, direction)
	args = append(args, sourceArgs...)
	args = append(args, targetArgs...)
	args = append(args, c.OrgId, c.OrgId, c.OrgId, application, application)
	rows := make([]resps.CmdbApplicationRelationResp, 0)
	if err := c.DB().Raw(sql, args...).Scan(&rows); err != nil {
		return e.New(e.DBError, err)
	}
	*out = append(*out, rows...)
	return nil
}

func cmdbVisibleAssetSQLScope(c *ctx.ServiceContext, assetAlias string, userProjectAlias string) (string, string, []interface{}) {
	if c.IsSuperAdmin || services.UserHasOrgRole(c.UserId, c.OrgId, consts.OrgRoleAdmin) {
		return "", "", nil
	}
	join := fmt.Sprintf("left join iac_user_project %s on %s.project_id = %s.project_id and %s.user_id = ?",
		userProjectAlias, userProjectAlias, assetAlias, userProjectAlias)
	where := fmt.Sprintf("and (%s.project_id = '' or %s.user_id is not null)", assetAlias, userProjectAlias)
	return join, where, []interface{}{c.UserId}
}

func cmdbManualRelationApplications(relations []models.CmdbApplicationRelation) []string {
	values := make([]string, 0, len(relations)*2)
	for _, relation := range relations {
		values = append(values, relation.SourceApplication, relation.TargetApplication)
	}
	return values
}

func cmdbVisibleApplicationAssetCounts(c *ctx.ServiceContext, applications []string) (map[string]int, e.Error) {
	applications = uniqueNonEmptyStrings(applications)
	if len(applications) == 0 {
		return map[string]int{}, nil
	}
	rows := make([]cmdbApplicationAssetCountRow, 0)
	if err := buildCmdbAssetQuery(c).
		Select("iac_cmdb_asset.application as application, count(*) as count").
		Where("iac_cmdb_asset.application in (?)", applications).
		Group("iac_cmdb_asset.application").
		Scan(&rows); err != nil {
		return nil, e.New(e.DBError, err)
	}
	result := make(map[string]int, len(rows))
	for _, row := range rows {
		result[row.Application] = row.Count
	}
	return result, nil
}

func cmdbVisibleApplicationAssets(c *ctx.ServiceContext, application string, limit int, offset int) ([]resps.CmdbApplicationAssetResp, e.Error) {
	assets := make([]resps.CmdbAssetResp, 0)
	query := buildCmdbAssetQuery(c).
		Where("iac_cmdb_asset.application = ?", application).
		Order("iac_cmdb_asset.risk_score desc, iac_cmdb_asset.cost desc, iac_cmdb_asset.updated_at desc").
		Limit(limit)
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Scan(&assets); err != nil {
		return nil, e.New(e.DBError, err)
	}
	result := make([]resps.CmdbApplicationAssetResp, 0, len(assets))
	for _, asset := range assets {
		result = append(result, cmdbApplicationAssetResp(asset))
	}
	return result, nil
}

func cmdbApplicationRelationCandidateCount(relation resps.CmdbApplicationRelationResp) int {
	if relation.Direction == "incoming" {
		return relation.SourceAssetCount
	}
	return relation.TargetAssetCount
}

func normalizeCmdbAssetRelationLimit(limit int) int {
	if limit <= 0 {
		return cmdbAssetRelationDefaultLimit
	}
	if limit > cmdbAssetRelationMaxLimit {
		return cmdbAssetRelationMaxLimit
	}
	return limit
}

func normalizeCmdbAssetRelationOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	if offset > 100000 {
		return 100000
	}
	return offset
}

func normalizeCmdbAssetRelationCursor(cursor string, fallbackOffset int) int {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return normalizeCmdbAssetRelationOffset(fallbackOffset)
	}
	offset, err := strconv.Atoi(cursor)
	if err != nil {
		return normalizeCmdbAssetRelationOffset(fallbackOffset)
	}
	return normalizeCmdbAssetRelationOffset(offset)
}

func cmdbRelationMergeBreakdown(target map[string]int64, items []resps.CmdbAssetRelationMetricResp) {
	for _, item := range items {
		cmdbRelationAddBreakdown(target, item.Name, item.Count)
	}
}

func cmdbRelationAddBreakdown(target map[string]int64, name string, count int64) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "unknown"
	}
	target[name] += count
}

func cmdbRelationBreakdownList(items map[string]int64) []resps.CmdbAssetRelationMetricResp {
	result := make([]resps.CmdbAssetRelationMetricResp, 0, len(items))
	for name, count := range items {
		result = append(result, resps.CmdbAssetRelationMetricResp{Name: name, Count: count})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].Name < result[j].Name
		}
		return result[i].Count > result[j].Count
	})
	return result
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]bool)
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

func cmdbAssetApplicationInferredRelation(sourceAssetId models.Id, targetAssetId models.Id, appRelation resps.CmdbApplicationRelationResp, sourceApplication string, targetApplication string) resps.CmdbAssetRelationResp {
	latestRelationAt := appRelation.LatestRelationAt
	id := models.Id(fmt.Sprintf("appinf-%s-%s-%s", sourceAssetId, targetAssetId, appRelation.Source))
	return resps.CmdbAssetRelationResp{
		CmdbAssetRelation: models.CmdbAssetRelation{
			TimedModel: models.TimedModel{
				BaseModel: models.BaseModel{Id: id},
				CreatedAt: latestRelationAt,
				UpdatedAt: latestRelationAt,
			},
			SourceAssetId: sourceAssetId,
			TargetAssetId: targetAssetId,
			RelationType:  appRelation.RelationType,
			Source:        models.CmdbRelationSourceAppInferred,
			Metadata: models.ResAttrs{
				"sourceApplication":           sourceApplication,
				"targetApplication":           targetApplication,
				"applicationRelationSource":   appRelation.Source,
				"applicationRelationCount":    appRelation.AssetRelationCount,
				"applicationSourceAssetCount": appRelation.SourceAssetCount,
				"applicationTargetAssetCount": appRelation.TargetAssetCount,
			},
		},
	}
}

func cmdbAssetRelationKey(sourceAssetId models.Id, targetAssetId models.Id, relationType string, source string) string {
	return fmt.Sprintf("%s/%s/%s/%s", sourceAssetId, targetAssetId, relationType, source)
}

func NormalizeCmdbAssetType(nativeType string) string {
	t := strings.ToLower(nativeType)
	switch {
	case containsAny(t, "eks_cluster", "containerengine_cluster", "oke", "kubernetes_cluster", "k8s_cluster"):
		return models.CmdbAssetTypeKubernetesCluster
	case containsAny(t, "kubernetes_namespace", "k8s_namespace"):
		return models.CmdbAssetTypeKubernetesNamespace
	case containsAny(t, "kubernetes_node", "k8s_node"):
		return models.CmdbAssetTypeKubernetesNode
	case containsAny(t, "kubernetes_pod", "k8s_pod"):
		return models.CmdbAssetTypeKubernetesPod
	case containsAny(t, "kubernetes_service", "k8s_service"):
		return models.CmdbAssetTypeKubernetesService
	case containsAny(t, "kubernetes_ingress", "k8s_ingress"):
		return models.CmdbAssetTypeKubernetesIngress
	case containsAny(t, "kubernetes_deployment", "k8s_deployment", "kubernetes_statefulset", "k8s_statefulset", "kubernetes_daemonset", "k8s_daemonset", "kubernetes_replicaset", "k8s_replicaset", "kubernetes_job", "k8s_job", "kubernetes_cronjob", "k8s_cronjob", "kubernetes_workload", "k8s_workload"):
		return models.CmdbAssetTypeKubernetesWorkload
	case containsAny(t, "db_instance", "rds_cluster", "rds_instance", "db_system", "autonomous_database", "database"):
		return models.CmdbAssetTypeRelationalDatabase
	case containsAny(t, "elasticache", "redis"):
		return models.CmdbAssetTypeRedisCache
	case containsAny(t, "instance", "compute_instance", "cvm", "ecs_instance", "oci_core_instance"):
		return models.CmdbAssetTypeComputeInstance
	case containsAny(t, "vpc", "vcn"):
		return models.CmdbAssetTypeNetworkVpc
	case strings.Contains(t, "subnet"):
		return models.CmdbAssetTypeNetworkSubnet
	case strings.Contains(t, "route_table"):
		return models.CmdbAssetTypeNetworkRouteTable
	case containsAny(t, "nat_gateway", "natgateway"):
		return models.CmdbAssetTypeNetworkNatGateway
	case containsAny(t, "internet_gateway", "internetgateway", "igw"):
		return models.CmdbAssetTypeNetworkInternetGateway
	case containsAny(t, "service_gateway", "servicegateway"):
		return models.CmdbAssetTypeNetworkServiceGateway
	case containsAny(t, "dynamic_routing_gateway", "dynamicroutinggateway", "oci_core_drg", "_drg", "drg"):
		return models.CmdbAssetTypeNetworkDrg
	case containsAny(t, "security_group", "security_list", "network_security_group", "nsg"):
		return models.CmdbAssetTypeNetworkSecurityGroup
	case containsAny(t, "eip", "public_ip"):
		return models.CmdbAssetTypePublicIP
	case containsAny(t, "load_balancer", "_lb", "elb", "alb", "nlb"):
		return models.CmdbAssetTypeLoadBalancer
	case containsAny(t, "ebs_volume", "block_volume", "core_volume", "disk"):
		return models.CmdbAssetTypeBlockVolume
	case containsAny(t, "s3_bucket", "objectstorage_bucket", "object_storage", "oss_bucket", "bucket"):
		return models.CmdbAssetTypeObjectStorageBucket
	default:
		return models.CmdbAssetTypeUnknown
	}
}

func containsAny(value string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(value, part) {
			return true
		}
	}
	return false
}

func extractTags(attrs models.ResAttrs) models.ResAttrs {
	if attrs == nil {
		return models.ResAttrs{}
	}
	for _, key := range []string{"tags", "defined_tags", "freeform_tags"} {
		if value, ok := attrs[key]; ok {
			if tags, ok := value.(map[string]interface{}); ok {
				return models.ResAttrs(tags)
			}
			if tags, ok := value.(models.ResAttrs); ok {
				return tags
			}
		}
	}
	return models.ResAttrs{}
}

func attrString(attrs models.ResAttrs, key string) string {
	if attrs == nil {
		return ""
	}
	value, ok := attrs[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func attrResAttrs(attrs models.ResAttrs, key string) models.ResAttrs {
	if attrs == nil {
		return nil
	}
	value, ok := attrs[key]
	if !ok || value == nil {
		return nil
	}
	switch v := value.(type) {
	case models.ResAttrs:
		return v
	case map[string]interface{}:
		return models.ResAttrs(v)
	default:
		return nil
	}
}

func attrInt(attrs models.ResAttrs, key string) int {
	if attrs == nil {
		return 0
	}
	value, ok := attrs[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		number, _ := v.Int64()
		return int(number)
	case string:
		number, _ := strconv.Atoi(strings.TrimSpace(v))
		return number
	default:
		number, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprintf("%v", v)))
		return number
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}
