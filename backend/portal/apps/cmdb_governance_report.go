// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"sort"
	"strings"

	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models"
	"cloudiac/portal/models/resps"
)

const (
	cmdbGovernanceUnsetLabel        = "未设置"
	cmdbGovernanceUnclassifiedLabel = "未分类"
	cmdbGovernanceTopAssetLimit     = 10
)

type cmdbGovernanceBreakdownMap map[string]*resps.CmdbAssetGovernanceBreakdownResp

// CmdbAssetGovernanceReport returns org-scoped cost, compliance and lifecycle governance summary.
func CmdbAssetGovernanceReport(c *ctx.ServiceContext) (*resps.CmdbAssetGovernanceReportResp, e.Error) {
	if _, err := BackfillCmdbAssetsFromIac(c); err != nil {
		return nil, err
	}
	if err := RefreshCmdbAssetGovernanceFields(c); err != nil {
		return nil, err
	}

	assets := make([]models.CmdbAsset, 0)
	if err := buildCmdbAssetQuery(c).Scan(&assets); err != nil {
		return nil, e.New(e.DBError, err)
	}

	report := &resps.CmdbAssetGovernanceReportResp{}
	providerBreakdown := cmdbGovernanceBreakdownMap{}
	businessLineBreakdown := cmdbGovernanceBreakdownMap{}
	applicationBreakdown := cmdbGovernanceBreakdownMap{}
	ownerBreakdown := cmdbGovernanceBreakdownMap{}
	lifecycleBreakdown := cmdbGovernanceBreakdownMap{}
	complianceRiskBreakdown := cmdbGovernanceBreakdownMap{}

	totalRiskScore := 0.0
	topCostAssets := make([]resps.CmdbAssetGovernanceAssetResp, 0)
	topRiskAssets := make([]resps.CmdbAssetGovernanceAssetResp, 0)

	for _, asset := range assets {
		report.TotalAssets++
		report.TotalCost += asset.Cost
		totalRiskScore += asset.RiskScore

		owner := cmdbGovernanceGroupName(asset.Owner, cmdbGovernanceUnsetLabel)
		application := cmdbGovernanceGroupName(asset.Application, cmdbGovernanceUnsetLabel)
		businessLine := cmdbGovernanceGroupName(asset.BusinessLine, cmdbGovernanceUnsetLabel)
		costCenter := cmdbGovernanceGroupName(asset.CostCenter, cmdbGovernanceUnsetLabel)
		lifecycle := cmdbGovernanceGroupName(asset.Lifecycle, cmdbGovernanceUnclassifiedLabel)
		complianceRisk := cmdbGovernanceGroupName(asset.ComplianceRisk, cmdbGovernanceUnsetLabel)
		provider := cmdbGovernanceGroupName(asset.Provider, cmdbGovernanceUnsetLabel)

		if owner == cmdbGovernanceUnsetLabel {
			report.MissingOwnerAssets++
		}
		if application == cmdbGovernanceUnsetLabel {
			report.MissingApplicationAssets++
		}
		if businessLine == cmdbGovernanceUnsetLabel {
			report.MissingBusinessLineAssets++
		}
		if costCenter == cmdbGovernanceUnsetLabel {
			report.MissingCostCenterAssets++
		}
		if lifecycle == cmdbGovernanceUnclassifiedLabel {
			report.UnclassifiedLifecycleAssets++
		}
		if asset.ProjectId == "" {
			report.UnassignedProjectAssets++
		}

		switch strings.ToLower(strings.TrimSpace(asset.Lifecycle)) {
		case "maintenance":
			report.MaintenanceAssets++
		case "retired":
			report.RetiredAssets++
		}
		switch strings.ToLower(strings.TrimSpace(asset.ComplianceRisk)) {
		case models.CloudOperationRiskHigh:
			report.HighRiskAssets++
		case models.CloudOperationRiskCritical:
			report.CriticalRiskAssets++
		}

		cmdbGovernanceAddBreakdown(providerBreakdown, provider, asset.Cost)
		cmdbGovernanceAddBreakdown(businessLineBreakdown, businessLine, asset.Cost)
		cmdbGovernanceAddBreakdown(applicationBreakdown, application, asset.Cost)
		cmdbGovernanceAddBreakdown(ownerBreakdown, owner, asset.Cost)
		cmdbGovernanceAddBreakdown(lifecycleBreakdown, lifecycle, asset.Cost)
		cmdbGovernanceAddBreakdown(complianceRiskBreakdown, complianceRisk, asset.Cost)

		item := cmdbGovernanceAssetItem(asset)
		if asset.Cost > 0 {
			topCostAssets = append(topCostAssets, item)
		}
		if asset.RiskScore > 0 || asset.ComplianceRisk == models.CloudOperationRiskHigh || asset.ComplianceRisk == models.CloudOperationRiskCritical {
			topRiskAssets = append(topRiskAssets, item)
		}
	}

	if report.TotalAssets > 0 {
		report.AverageRiskScore = totalRiskScore / float64(report.TotalAssets)
	}
	report.CostByProvider = cmdbGovernanceBreakdownList(providerBreakdown)
	report.CostByBusinessLine = cmdbGovernanceBreakdownList(businessLineBreakdown)
	report.CostByApplication = cmdbGovernanceBreakdownList(applicationBreakdown)
	report.CostByOwner = cmdbGovernanceBreakdownList(ownerBreakdown)
	report.LifecycleBreakdown = cmdbGovernanceBreakdownList(lifecycleBreakdown)
	report.ComplianceRiskBreakdown = cmdbGovernanceBreakdownList(complianceRiskBreakdown)
	report.TopCostAssets = cmdbGovernanceTopCostAssets(topCostAssets)
	report.TopRiskAssets = cmdbGovernanceTopRiskAssets(topRiskAssets)

	return report, nil
}

func cmdbGovernanceGroupName(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func cmdbGovernanceAddBreakdown(items cmdbGovernanceBreakdownMap, name string, cost float64) {
	if _, ok := items[name]; !ok {
		items[name] = &resps.CmdbAssetGovernanceBreakdownResp{Name: name}
	}
	items[name].Count++
	items[name].Cost += cost
}

func cmdbGovernanceBreakdownList(items cmdbGovernanceBreakdownMap) []resps.CmdbAssetGovernanceBreakdownResp {
	result := make([]resps.CmdbAssetGovernanceBreakdownResp, 0, len(items))
	for _, item := range items {
		result = append(result, *item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Cost == result[j].Cost {
			if result[i].Count == result[j].Count {
				return result[i].Name < result[j].Name
			}
			return result[i].Count > result[j].Count
		}
		return result[i].Cost > result[j].Cost
	})
	return result
}

func cmdbGovernanceAssetItem(asset models.CmdbAsset) resps.CmdbAssetGovernanceAssetResp {
	return resps.CmdbAssetGovernanceAssetResp{
		Id:             asset.Id,
		Name:           firstNonEmpty(asset.Name, asset.NativeId, asset.IacAddress, asset.Id.String()),
		Provider:       asset.Provider,
		Region:         asset.Region,
		AssetType:      asset.AssetType,
		NativeId:       asset.NativeId,
		Owner:          asset.Owner,
		Application:    asset.Application,
		BusinessLine:   asset.BusinessLine,
		Lifecycle:      asset.Lifecycle,
		ComplianceRisk: asset.ComplianceRisk,
		Cost:           asset.Cost,
		RiskScore:      asset.RiskScore,
		ManagedBy:      asset.ManagedBy,
	}
}

func cmdbGovernanceTopCostAssets(items []resps.CmdbAssetGovernanceAssetResp) []resps.CmdbAssetGovernanceAssetResp {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Cost == items[j].Cost {
			return items[i].Name < items[j].Name
		}
		return items[i].Cost > items[j].Cost
	})
	if len(items) > cmdbGovernanceTopAssetLimit {
		return items[:cmdbGovernanceTopAssetLimit]
	}
	return items
}

func cmdbGovernanceTopRiskAssets(items []resps.CmdbAssetGovernanceAssetResp) []resps.CmdbAssetGovernanceAssetResp {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].RiskScore == items[j].RiskScore {
			iRisk := cmdbGovernanceRiskWeight(items[i].ComplianceRisk)
			jRisk := cmdbGovernanceRiskWeight(items[j].ComplianceRisk)
			if iRisk == jRisk {
				return items[i].Name < items[j].Name
			}
			return iRisk > jRisk
		}
		return items[i].RiskScore > items[j].RiskScore
	})
	if len(items) > cmdbGovernanceTopAssetLimit {
		return items[:cmdbGovernanceTopAssetLimit]
	}
	return items
}

func cmdbGovernanceRiskWeight(risk string) int {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case models.CloudOperationRiskCritical:
		return 4
	case models.CloudOperationRiskHigh:
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}
