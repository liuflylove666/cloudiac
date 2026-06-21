// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CloudRiskFindingResp struct {
	models.CloudRiskFinding

	ProjectName string `json:"projectName"`
	EnvName     string `json:"envName"`
	AssetName   string `json:"assetName"`
}

type CloudRiskSummaryResp struct {
	Total           int64 `json:"total"`
	Open            int64 `json:"open"`
	InProgress      int64 `json:"inProgress"`
	Suppressed      int64 `json:"suppressed"`
	Resolved        int64 `json:"resolved"`
	High            int64 `json:"high"`
	Critical        int64 `json:"critical"`
	ActiveHigh      int64 `json:"activeHigh"`
	ActiveCritical  int64 `json:"activeCritical"`
	PublicExposure  int64 `json:"publicExposure"`
	UnmanagedAssets int64 `json:"unmanagedAssets"`
	UnownedAssets   int64 `json:"unownedAssets"`
	DriftRisks      int64 `json:"driftRisks"`
	ComplianceRisks int64 `json:"complianceRisks"`
}

type CloudRiskListResp struct {
	Summary  CloudRiskSummaryResp   `json:"summary"`
	Total    int64                  `json:"total"`
	PageSize int                    `json:"pageSize"`
	List     []CloudRiskFindingResp `json:"list"`
}
