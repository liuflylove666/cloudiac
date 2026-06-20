// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package models

import "cloudiac/portal/libs/db"

const (
	CmdbAssetSourceIacResource  = "iac_resource"
	CmdbAssetSourceCloudCollect = "cloud_collect"

	CmdbManagedByIac         = "iac"
	CmdbManagedByCloudOnly   = "cloud_only"
	CmdbManagedByCloudLinked = "cloud_linked"
	CmdbManagedByManual      = "manual"

	CmdbCloudAccountSourceVariableGroup   = "variable_group"
	CmdbCloudAccountSourceResourceAccount = "resource_account"
	CmdbCloudAccountSourceCloudAccount    = "cloud_account"

	CmdbRelationSourceIacDependency = "iac_dependency"
	CmdbRelationSourceCloudInferred = "cloud_inferred"
	CmdbRelationSourceManualApp     = "manual_application"
	CmdbRelationSourceAppInferred   = "application_inferred"
	CmdbRelationTypeDependsOn       = "depends_on"
	CmdbRelationTypeContains        = "contains"

	CmdbAssetChangeTypeCreated  = "created"
	CmdbAssetChangeTypeUpdated  = "updated"
	CmdbAssetChangeSourceManual = "manual_edit"
	CmdbAssetChangeSourceImport = "import"

	CmdbSyncTaskPending  = "pending"
	CmdbSyncTaskRunning  = "running"
	CmdbSyncTaskComplete = "complete"
	CmdbSyncTaskFailed   = "failed"

	CmdbSyncLogLevelInfo  = "info"
	CmdbSyncLogLevelWarn  = "warn"
	CmdbSyncLogLevelError = "error"

	CmdbAssetTypeComputeInstance      = "compute_instance"
	CmdbAssetTypeKubernetesCluster    = "kubernetes_cluster"
	CmdbAssetTypeNetworkVpc           = "network_vpc"
	CmdbAssetTypeNetworkSubnet        = "network_subnet"
	CmdbAssetTypeNetworkRouteTable    = "network_route_table"
	CmdbAssetTypeNetworkSecurityGroup = "network_security_group"
	CmdbAssetTypePublicIP             = "public_ip"
	CmdbAssetTypeLoadBalancer         = "load_balancer"
	CmdbAssetTypeBlockVolume          = "block_volume"
	CmdbAssetTypeObjectStorageBucket  = "object_storage_bucket"
	CmdbAssetTypeRelationalDatabase   = "relational_database"
	CmdbAssetTypeRedisCache           = "redis_cache"
	CmdbAssetTypeUnknown              = "unknown"
)

type CmdbAsset struct {
	TimedModel

	OrgId     Id `json:"orgId" gorm:"index;size:32;not null"`
	ProjectId Id `json:"projectId" gorm:"index;size:32;not null;default:''"`
	EnvId     Id `json:"envId" gorm:"index;size:32;not null;default:''"`
	TaskId    Id `json:"taskId" gorm:"index;size:32;not null;default:''"`

	Source     string `json:"source" gorm:"index;size:32;not null;default:'iac_resource'"`
	Provider   string `json:"provider" gorm:"index;size:64;not null;default:''"`
	AccountId  string `json:"accountId" gorm:"index;size:128;not null;default:''"`
	Region     string `json:"region" gorm:"index;size:128;not null;default:''"`
	Zone       string `json:"zone" gorm:"size:128;not null;default:''"`
	AssetType  string `json:"assetType" gorm:"index;size:64;not null;default:'unknown'"`
	NativeType string `json:"nativeType" gorm:"index;size:128;not null;default:''"`
	NativeId   string `json:"nativeId" gorm:"index;size:255;not null;default:''"`

	Name    string `json:"name" gorm:"index;size:255;not null;default:''"`
	Status  string `json:"status" gorm:"index;size:64;not null;default:''"`
	Address string `json:"address" gorm:"size:255;not null;default:''"`
	Module  string `json:"module" gorm:"size:255;not null;default:''"`

	Owner          string  `json:"owner" gorm:"index;size:128;not null;default:''"`
	Application    string  `json:"application" gorm:"index;size:128;not null;default:''"`
	BusinessLine   string  `json:"businessLine" gorm:"index;size:128;not null;default:''"`
	Lifecycle      string  `json:"lifecycle" gorm:"index;size:64;not null;default:''"`
	Cost           float64 `json:"cost" gorm:"type:decimal(18,4);not null;default:0"`
	ComplianceRisk string  `json:"complianceRisk" gorm:"index;size:64;not null;default:''"`

	PublicIp  string `json:"publicIp" gorm:"size:128;not null;default:''"`
	PrivateIp string `json:"privateIp" gorm:"size:128;not null;default:''"`

	IacResourceId Id     `json:"iacResourceId" gorm:"index;size:32;not null;default:''"`
	IacAddress    string `json:"iacAddress" gorm:"size:255;not null;default:''"`

	ManagedBy       string  `json:"managedBy" gorm:"index;size:32;not null;default:''"`
	CloudAccountId  Id      `json:"cloudAccountId" gorm:"index;size:32;not null;default:''"`
	SyncPolicyId    Id      `json:"syncPolicyId" gorm:"index;size:32;not null;default:''"`
	LastOperationId Id      `json:"lastOperationId" gorm:"index;size:32;not null;default:''"`
	RiskScore       float64 `json:"riskScore" gorm:"type:decimal(10,4);not null;default:0"`
	CostCenter      string  `json:"costCenter" gorm:"index;size:128;not null;default:''"`

	Tags       ResAttrs `json:"tags,omitempty" gorm:"type:json"`
	Attributes ResAttrs `json:"attributes,omitempty" gorm:"type:json"`
	RawData    ResAttrs `json:"rawData,omitempty" gorm:"type:json"`

	LastSyncAt Time `json:"lastSyncAt" gorm:"type:datetime;default:null"`
}

func (CmdbAsset) TableName() string {
	return "iac_cmdb_asset"
}

func (a CmdbAsset) Migrate(sess *db.Session) error {
	if err := sess.RemoveIndex(a.TableName(), "unique_cmdb_asset_source_native"); err != nil {
		return err
	}
	return a.AddUniqueIndex(sess, "unique_cmdb_asset_source_native_account", "org_id", "source", "provider", "account_id", "region", "native_id")
}

type CmdbAssetRelation struct {
	TimedModel

	OrgId         Id       `json:"orgId" gorm:"index;size:32;not null"`
	SourceAssetId Id       `json:"sourceAssetId" gorm:"index;size:32;not null"`
	TargetAssetId Id       `json:"targetAssetId" gorm:"index;size:32;not null"`
	RelationType  string   `json:"relationType" gorm:"index;size:64;not null;default:'depends_on'"`
	Source        string   `json:"source" gorm:"index;size:64;not null;default:'iac_dependency'"`
	Metadata      ResAttrs `json:"metadata,omitempty" gorm:"type:json"`
}

func (CmdbAssetRelation) TableName() string {
	return "iac_cmdb_asset_relation"
}

func (r CmdbAssetRelation) Migrate(sess *db.Session) error {
	return r.AddUniqueIndex(sess, "unique_cmdb_asset_relation", "org_id", "source_asset_id", "target_asset_id", "relation_type", "source")
}

type CmdbApplicationRelation struct {
	TimedModel

	OrgId             Id       `json:"orgId" gorm:"index;size:32;not null"`
	SourceApplication string   `json:"sourceApplication" gorm:"index;size:128;not null;default:''"`
	TargetApplication string   `json:"targetApplication" gorm:"index;size:128;not null;default:''"`
	RelationType      string   `json:"relationType" gorm:"index;size:64;not null;default:'depends_on'"`
	Source            string   `json:"source" gorm:"index;size:64;not null;default:'manual_application'"`
	Metadata          ResAttrs `json:"metadata,omitempty" gorm:"type:json"`
}

func (CmdbApplicationRelation) TableName() string {
	return "iac_cmdb_application_relation"
}

func (r CmdbApplicationRelation) Migrate(sess *db.Session) error {
	return r.AddUniqueIndex(sess, "unique_cmdb_application_relation", "org_id", "source_application", "target_application", "relation_type", "source")
}

type CmdbAssetChange struct {
	TimedModel

	OrgId      Id       `json:"orgId" gorm:"index;size:32;not null"`
	AssetId    Id       `json:"assetId" gorm:"index;size:32;not null"`
	ChangeType string   `json:"changeType" gorm:"index;size:32;not null;default:'updated'"`
	Source     string   `json:"source" gorm:"index;size:64;not null;default:''"`
	Summary    string   `json:"summary" gorm:"size:255;not null;default:''"`
	Diff       ResAttrs `json:"diff,omitempty" gorm:"type:json"`
}

func (CmdbAssetChange) TableName() string {
	return "iac_cmdb_asset_change"
}

type CmdbSyncTask struct {
	TimedModel

	OrgId         Id       `json:"orgId" gorm:"index;size:32;not null"`
	AccountSource string   `json:"accountSource" gorm:"index;size:32;not null;default:'variable_group'"`
	AccountId     Id       `json:"accountId" gorm:"index;size:32;not null"`
	AccountName   string   `json:"accountName" gorm:"size:128;not null;default:''"`
	Provider      string   `json:"provider" gorm:"index;size:64;not null;default:''"`
	Regions       StrSlice `json:"regions" gorm:"type:json"`
	AssetTypes    StrSlice `json:"assetTypes" gorm:"type:json"`
	Status        string   `json:"status" gorm:"index;size:32;not null;default:'pending'"`
	ErrorMessage  string   `json:"errorMessage" gorm:"type:text"`
	Stats         ResAttrs `json:"stats,omitempty" gorm:"type:json"`
	StartedAt     Time     `json:"startedAt" gorm:"type:datetime;default:null"`
	EndedAt       Time     `json:"endedAt" gorm:"type:datetime;default:null"`
}

func (CmdbSyncTask) TableName() string {
	return "iac_cmdb_sync_task"
}

type CmdbSyncTaskLog struct {
	TimedModel

	OrgId   Id       `json:"orgId" gorm:"index;size:32;not null"`
	TaskId  Id       `json:"taskId" gorm:"index;size:32;not null"`
	Level   string   `json:"level" gorm:"index;size:32;not null;default:'info'"`
	Stage   string   `json:"stage" gorm:"index;size:64;not null;default:''"`
	Message string   `json:"message" gorm:"type:text"`
	Data    ResAttrs `json:"data,omitempty" gorm:"type:json"`
}

func (CmdbSyncTaskLog) TableName() string {
	return "iac_cmdb_sync_task_log"
}
