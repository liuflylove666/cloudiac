// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"encoding/json"
	"fmt"
	"time"

	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models"
)

type cmdbAssetUpsertResult struct {
	Created bool
	Updated bool
	Skipped bool
}

func upsertCmdbAsset(c *ctx.ServiceContext, asset *models.CmdbAsset, changeSource string) (cmdbAssetUpsertResult, e.Error) {
	if asset.NativeId == "" {
		return cmdbAssetUpsertResult{Skipped: true}, nil
	}
	if asset.OrgId == "" {
		asset.OrgId = c.OrgId
	}
	if asset.Source == "" {
		asset.Source = models.CmdbAssetSourceCloudCollect
	}
	if changeSource == "" {
		changeSource = asset.Source
	}
	normalizeCmdbAssetAttrs(asset)
	if err := prepareCmdbAssetGovernance(c, asset); err != nil {
		return cmdbAssetUpsertResult{}, err
	}

	existing := models.CmdbAsset{}
	err := c.DB().Where("org_id = ? and source = ? and provider = ? and account_id = ? and region = ? and native_id = ?",
		asset.OrgId, asset.Source, asset.Provider, asset.AccountId, asset.Region, asset.NativeId).First(&existing)
	if err != nil && !e.IsRecordNotFound(err) {
		return cmdbAssetUpsertResult{}, e.New(e.DBError, err)
	}

	if e.IsRecordNotFound(err) {
		if asset.Id == "" {
			asset.Id = models.NewId("ci")
		}
		if err := models.Create(c.DB(), asset); err != nil {
			return cmdbAssetUpsertResult{}, e.New(e.DBError, err)
		}
		if err := recordCmdbAssetChange(c, asset.OrgId, asset.Id, models.CmdbAssetChangeTypeCreated, changeSource, models.ResAttrs{
			"after": cmdbAssetSnapshot(asset),
		}); err != nil {
			return cmdbAssetUpsertResult{}, err
		}
		return cmdbAssetUpsertResult{Created: true}, nil
	}

	asset.Id = existing.Id
	diff := cmdbAssetDiff(existing, asset)
	if len(diff) == 0 {
		if asset.Source == models.CmdbAssetSourceCloudCollect && !cmdbSameTime(existing.LastSyncAt, asset.LastSyncAt) {
			if _, err := c.DB().Model(&models.CmdbAsset{}).Where("id = ?", existing.Id).UpdateColumn("last_sync_at", asset.LastSyncAt); err != nil {
				return cmdbAssetUpsertResult{}, e.New(e.DBError, err)
			}
		}
		return cmdbAssetUpsertResult{Skipped: true}, nil
	}

	if _, err := c.DB().Model(&models.CmdbAsset{}).Where("id = ?", existing.Id).UpdateAttrs(cmdbAssetUpdateAttrs(asset)); err != nil {
		return cmdbAssetUpsertResult{}, e.New(e.DBError, err)
	}
	if err := recordCmdbAssetChange(c, asset.OrgId, existing.Id, models.CmdbAssetChangeTypeUpdated, changeSource, diff); err != nil {
		return cmdbAssetUpsertResult{}, err
	}
	return cmdbAssetUpsertResult{Updated: true}, nil
}

func normalizeCmdbAssetAttrs(asset *models.CmdbAsset) {
	if asset.Tags == nil {
		asset.Tags = models.ResAttrs{}
	}
	if asset.Attributes == nil {
		asset.Attributes = models.ResAttrs{}
	}
	if asset.RawData == nil {
		asset.RawData = models.ResAttrs{}
	}
}

func prepareCmdbAssetGovernance(c *ctx.ServiceContext, asset *models.CmdbAsset) e.Error {
	asset.ManagedBy = inferCmdbManagedBy(asset)
	if asset.CloudAccountId != "" {
		return nil
	}
	if attrString(asset.RawData, "accountSource") == models.CmdbCloudAccountSourceCloudAccount {
		asset.CloudAccountId = models.Id(attrString(asset.RawData, "accountRefId"))
		if asset.CloudAccountId != "" {
			return nil
		}
	}
	if asset.Provider == "" || asset.AccountId == "" {
		return nil
	}
	account := models.CloudAccount{}
	if err := c.DB().Model(&models.CloudAccount{}).
		Where("org_id = ? and provider = ? and account_id = ?", asset.OrgId, normalizeProvider(asset.Provider), asset.AccountId).
		First(&account); err != nil {
		if e.IsRecordNotFound(err) {
			return nil
		}
		return e.New(e.DBError, err)
	}
	asset.CloudAccountId = account.Id
	return nil
}

func RefreshCmdbAssetGovernanceFields(c *ctx.ServiceContext) e.Error {
	managedByExpr := cmdbInferredManagedByExpr("")
	if _, err := c.DB().Exec(fmt.Sprintf(`update iac_cmdb_asset
		set managed_by = %s
		where org_id = ? and managed_by <> %s`, managedByExpr, managedByExpr), c.OrgId); err != nil {
		return e.New(e.DBError, err)
	}

	accounts := make([]models.CloudAccount, 0)
	if err := c.DB().Model(&models.CloudAccount{}).
		Where("org_id = ? and provider <> '' and account_id <> ''", c.OrgId).
		Find(&accounts); err != nil {
		return e.New(e.DBError, err)
	}
	for _, account := range accounts {
		if _, err := c.DB().Model(&models.CmdbAsset{}).
			Where("org_id = ? and provider = ? and account_id = ? and cloud_account_id = ''", c.OrgId, normalizeProvider(account.Provider), account.AccountId).
			UpdateAttrs(map[string]interface{}{"cloud_account_id": account.Id}); err != nil {
			return e.New(e.DBError, err)
		}
	}
	return nil
}

func cmdbManagedByExpr(prefix string) string {
	return fmt.Sprintf(`case
		when %[1]smanaged_by <> '' then %[1]smanaged_by
		else %[2]s
	end`, prefix, cmdbInferredManagedByExpr(prefix))
}

func cmdbInferredManagedByExpr(prefix string) string {
	return fmt.Sprintf(`case
		when %[1]ssource = '%[2]s' or %[1]siac_resource_id <> '' then '%[3]s'
		when %[1]ssource = '%[4]s' and %[1]sproject_id = '' and %[1]senv_id = '' and %[1]siac_resource_id = '' then '%[5]s'
		when %[1]ssource = '%[4]s' then '%[6]s'
		else '%[7]s'
	end`,
		prefix,
		models.CmdbAssetSourceIacResource,
		models.CmdbManagedByIac,
		models.CmdbAssetSourceCloudCollect,
		models.CmdbManagedByCloudOnly,
		models.CmdbManagedByCloudLinked,
		models.CmdbManagedByManual)
}

func inferCmdbManagedBy(asset *models.CmdbAsset) string {
	if asset.Source == models.CmdbAssetSourceIacResource || asset.IacResourceId != "" {
		return models.CmdbManagedByIac
	}
	if asset.Source == models.CmdbAssetSourceCloudCollect {
		if asset.ProjectId == "" && asset.EnvId == "" && asset.IacResourceId == "" {
			return models.CmdbManagedByCloudOnly
		}
		return models.CmdbManagedByCloudLinked
	}
	return models.CmdbManagedByManual
}

func cmdbAssetUpdateAttrs(asset *models.CmdbAsset) map[string]interface{} {
	return map[string]interface{}{
		"project_id":      asset.ProjectId,
		"env_id":          asset.EnvId,
		"task_id":         asset.TaskId,
		"account_id":      asset.AccountId,
		"region":          asset.Region,
		"zone":            asset.Zone,
		"asset_type":      asset.AssetType,
		"native_type":     asset.NativeType,
		"name":            asset.Name,
		"status":          asset.Status,
		"address":         asset.Address,
		"module":          asset.Module,
		"public_ip":       asset.PublicIp,
		"private_ip":      asset.PrivateIp,
		"iac_resource_id": asset.IacResourceId,
		"iac_address":     asset.IacAddress,
		"managed_by":      asset.ManagedBy,
		"cloud_account_id": asset.CloudAccountId,
		"sync_policy_id":   asset.SyncPolicyId,
		"last_operation_id": asset.LastOperationId,
		"risk_score":        asset.RiskScore,
		"cost_center":       asset.CostCenter,
		"tags":            asset.Tags,
		"attributes":      asset.Attributes,
		"raw_data":        asset.RawData,
		"last_sync_at":    asset.LastSyncAt,
	}
}

func cmdbAssetSnapshot(asset *models.CmdbAsset) models.ResAttrs {
	return models.ResAttrs{
		"projectId":     asset.ProjectId.String(),
		"envId":         asset.EnvId.String(),
		"taskId":        asset.TaskId.String(),
		"source":        asset.Source,
		"provider":      asset.Provider,
		"accountId":     asset.AccountId,
		"region":        asset.Region,
		"zone":          asset.Zone,
		"assetType":     asset.AssetType,
		"nativeType":    asset.NativeType,
		"nativeId":      asset.NativeId,
		"name":          asset.Name,
		"status":        asset.Status,
		"address":       asset.Address,
		"module":        asset.Module,
		"publicIp":      asset.PublicIp,
		"privateIp":     asset.PrivateIp,
		"iacResourceId": asset.IacResourceId.String(),
		"iacAddress":    asset.IacAddress,
		"managedBy":     asset.ManagedBy,
		"cloudAccountId": asset.CloudAccountId.String(),
		"syncPolicyId":   asset.SyncPolicyId.String(),
		"lastOperationId": asset.LastOperationId.String(),
		"riskScore":       asset.RiskScore,
		"costCenter":      asset.CostCenter,
		"tags":          asset.Tags,
		"attributes":    asset.Attributes,
		"rawData":       asset.RawData,
		"lastSyncAt":    cmdbTimeValue(asset.LastSyncAt),
		"owner":          asset.Owner,
		"application":    asset.Application,
		"businessLine":   asset.BusinessLine,
		"lifecycle":      asset.Lifecycle,
		"cost":           asset.Cost,
		"complianceRisk": asset.ComplianceRisk,
	}
}

func cmdbAssetDiff(before models.CmdbAsset, after *models.CmdbAsset) models.ResAttrs {
	diff := models.ResAttrs{}
	appendCmdbScalarDiff(diff, "projectId", before.ProjectId.String(), after.ProjectId.String())
	appendCmdbScalarDiff(diff, "envId", before.EnvId.String(), after.EnvId.String())
	appendCmdbScalarDiff(diff, "taskId", before.TaskId.String(), after.TaskId.String())
	appendCmdbScalarDiff(diff, "accountId", before.AccountId, after.AccountId)
	appendCmdbScalarDiff(diff, "region", before.Region, after.Region)
	appendCmdbScalarDiff(diff, "zone", before.Zone, after.Zone)
	appendCmdbScalarDiff(diff, "assetType", before.AssetType, after.AssetType)
	appendCmdbScalarDiff(diff, "nativeType", before.NativeType, after.NativeType)
	appendCmdbScalarDiff(diff, "name", before.Name, after.Name)
	appendCmdbScalarDiff(diff, "status", before.Status, after.Status)
	appendCmdbScalarDiff(diff, "address", before.Address, after.Address)
	appendCmdbScalarDiff(diff, "module", before.Module, after.Module)
	appendCmdbScalarDiff(diff, "publicIp", before.PublicIp, after.PublicIp)
	appendCmdbScalarDiff(diff, "privateIp", before.PrivateIp, after.PrivateIp)
	appendCmdbScalarDiff(diff, "iacResourceId", before.IacResourceId.String(), after.IacResourceId.String())
	appendCmdbScalarDiff(diff, "iacAddress", before.IacAddress, after.IacAddress)
	appendCmdbScalarDiff(diff, "managedBy", before.ManagedBy, after.ManagedBy)
	appendCmdbScalarDiff(diff, "cloudAccountId", before.CloudAccountId.String(), after.CloudAccountId.String())
	appendCmdbScalarDiff(diff, "syncPolicyId", before.SyncPolicyId.String(), after.SyncPolicyId.String())
	appendCmdbScalarDiff(diff, "lastOperationId", before.LastOperationId.String(), after.LastOperationId.String())
	appendCmdbScalarDiff(diff, "riskScore", before.RiskScore, after.RiskScore)
	appendCmdbScalarDiff(diff, "costCenter", before.CostCenter, after.CostCenter)
	appendCmdbJSONDiff(diff, "tags", before.Tags, after.Tags)
	appendCmdbJSONDiff(diff, "attributes", before.Attributes, after.Attributes)
	appendCmdbJSONDiff(diff, "rawData", before.RawData, after.RawData)
	return diff
}

func appendCmdbScalarDiff(diff models.ResAttrs, key string, before interface{}, after interface{}) {
	if fmt.Sprintf("%v", before) == fmt.Sprintf("%v", after) {
		return
	}
	diff[key] = models.ResAttrs{
		"before": before,
		"after":  after,
	}
}

func appendCmdbJSONDiff(diff models.ResAttrs, key string, before models.ResAttrs, after models.ResAttrs) {
	if before == nil {
		before = models.ResAttrs{}
	}
	if after == nil {
		after = models.ResAttrs{}
	}
	if cmdbJSONString(before) == cmdbJSONString(after) {
		return
	}
	diff[key] = models.ResAttrs{
		"before": before,
		"after":  after,
	}
}

func cmdbJSONString(value interface{}) string {
	body, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(body)
}

func cmdbSameTime(left models.Time, right models.Time) bool {
	leftTime := time.Time(left)
	rightTime := time.Time(right)
	if leftTime.IsZero() && rightTime.IsZero() {
		return true
	}
	return leftTime.Equal(rightTime)
}

func cmdbTimeValue(value models.Time) string {
	t := time.Time(value)
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func recordCmdbAssetChange(c *ctx.ServiceContext, orgId models.Id, assetId models.Id, changeType string, source string, diff models.ResAttrs) e.Error {
	summary := "资产字段发生变化"
	if changeType == models.CmdbAssetChangeTypeCreated {
		summary = "资产首次进入 CMDB"
	} else if len(diff) > 0 {
		summary = fmt.Sprintf("更新 %d 个字段", len(diff))
	}
	summary = normalizeCmdbText(summary)
	change := &models.CmdbAssetChange{
		OrgId:      orgId,
		AssetId:    assetId,
		ChangeType: changeType,
		Source:     source,
		Summary:    summary,
		Diff:       diff,
	}
	change.Id = models.NewId("cic")
	if err := models.Create(c.DB(), change); err != nil {
		return e.New(e.DBError, err)
	}
	return nil
}
