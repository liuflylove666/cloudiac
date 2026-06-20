// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CmdbAssetResp struct {
	models.CmdbAsset

	ProjectName string `json:"projectName"`
	EnvName     string `json:"envName"`

	Relations []CmdbAssetRelationResp `json:"relations,omitempty" gorm:"-"`
	Changes   []CmdbAssetChangeResp   `json:"changes,omitempty" gorm:"-"`
}

type CmdbAssetRelationResp struct {
	models.CmdbAssetRelation

	SourceAssetName string `json:"sourceAssetName"`
	SourceAssetType string `json:"sourceAssetType"`
	TargetAssetName string `json:"targetAssetName"`
	TargetAssetType string `json:"targetAssetType"`
}

type CmdbAssetChangeResp struct {
	models.CmdbAssetChange
}

type CmdbAssetFilterResp struct {
	Projects   []OrgProjectResp `json:"projects"`
	Envs       []EnvResp        `json:"envs"`
	Providers  []string         `json:"providers"`
	AccountIds []string         `json:"accountIds"`
	AssetTypes []string         `json:"assetTypes"`
	Sources    []string         `json:"sources"`
	Statuses   []string         `json:"statuses"`
	ManagedBy  []string         `json:"managedBy"`
}

type CmdbBackfillResp struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
}

type CmdbImportResp struct {
	Total            int      `json:"total"`
	Created          int      `json:"created"`
	Updated          int      `json:"updated"`
	Skipped          int      `json:"skipped"`
	OwnershipUpdated int      `json:"ownershipUpdated"`
	Errors           []string `json:"errors"`
}

type CmdbBatchOwnershipResp struct {
	Total   int      `json:"total"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

type CmdbApplicationResp struct {
	Application       string                        `json:"application"`
	Owner             []string                      `json:"owner"`
	BusinessLine      []string                      `json:"businessLine"`
	AssetCount        int                           `json:"assetCount"`
	IncomingAppCount  int                           `json:"incomingAppCount"`
	OutgoingAppCount  int                           `json:"outgoingAppCount"`
	ImpactedAppCount  int                           `json:"impactedAppCount"`
	RecentChangeCount int                           `json:"recentChangeCount"`
	ChangedAssetCount int                           `json:"changedAssetCount"`
	RiskLevel         string                        `json:"riskLevel"`
	RiskScore         int                           `json:"riskScore"`
	RiskReason        string                        `json:"riskReason"`
	Assets            []CmdbApplicationAssetResp    `json:"assets,omitempty"`
	Upstreams         []CmdbApplicationRelationResp `json:"upstreams,omitempty"`
	Downstreams       []CmdbApplicationRelationResp `json:"downstreams,omitempty"`
	ImpactedApps      []CmdbApplicationImpactResp   `json:"impactedApplications,omitempty"`
	RecentChanges     []CmdbApplicationChangeResp   `json:"recentChanges,omitempty"`
}

type CmdbApplicationAssetResp struct {
	Id             models.Id   `json:"id"`
	Name           string      `json:"name"`
	AssetType      string      `json:"assetType"`
	Provider       string      `json:"provider"`
	NativeId       string      `json:"nativeId"`
	Status         string      `json:"status"`
	Owner          string      `json:"owner"`
	BusinessLine   string      `json:"businessLine"`
	ComplianceRisk string      `json:"complianceRisk"`
	UpdatedAt      models.Time `json:"updatedAt"`
}

type CmdbApplicationRelationResp struct {
	Application        string      `json:"application"`
	Direction          string      `json:"direction"`
	RelationType       string      `json:"relationType"`
	Source             string      `json:"source"`
	AssetRelationCount int         `json:"assetRelationCount"`
	SourceAssetCount   int         `json:"sourceAssetCount"`
	TargetAssetCount   int         `json:"targetAssetCount"`
	LatestRelationAt   models.Time `json:"latestRelationAt"`
}

type CmdbApplicationImpactResp struct {
	Application      string      `json:"application"`
	RelationSources  []string    `json:"relationSources"`
	RelationCount    int         `json:"relationCount"`
	SourceAssetCount int         `json:"sourceAssetCount"`
	TargetAssetCount int         `json:"targetAssetCount"`
	LatestRelationAt models.Time `json:"latestRelationAt"`
}

type CmdbApplicationChangeResp struct {
	Id         models.Id       `json:"id"`
	AssetId    models.Id       `json:"assetId"`
	AssetName  string          `json:"assetName"`
	ChangeType string          `json:"changeType"`
	Source     string          `json:"source"`
	Summary    string          `json:"summary"`
	Diff       models.ResAttrs `json:"diff,omitempty"`
	CreatedAt  models.Time     `json:"createdAt"`
}

type CmdbCloudAccountResp struct {
	Id                    models.Id   `json:"id"`
	Source                string      `json:"source"`
	Name                  string      `json:"name"`
	Description           string      `json:"description"`
	Provider              string      `json:"provider"`
	AccountId             string      `json:"accountId"`
	Regions               []string    `json:"regions"`
	Ready                 bool        `json:"ready"`
	MissingCredentialKeys []string    `json:"missingCredentialKeys"`
	SupportedAssetTypes   []string    `json:"supportedAssetTypes"`
	UpdatedAt             models.Time `json:"updatedAt"`
}

type CmdbSyncTaskResp struct {
	models.CmdbSyncTask
}

type CmdbSyncTaskLogResp struct {
	models.CmdbSyncTaskLog
}

type CmdbSyncTaskDetailResp struct {
	models.CmdbSyncTask

	Logs []CmdbSyncTaskLogResp `json:"logs"`
}
