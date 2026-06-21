// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"encoding/json"
	"fmt"
	"strings"

	"cloudiac/portal/models"
)

type cmdbInventoryPayload struct {
	Assets []cmdbInventoryAsset `json:"assets"`
}

type cmdbInventoryAsset struct {
	AssetType  string          `json:"assetType"`
	NativeType string          `json:"nativeType"`
	NativeId   string          `json:"nativeId"`
	Id         string          `json:"id"`
	Name       string          `json:"name"`
	Region     string          `json:"region"`
	Zone       string          `json:"zone"`
	Status     string          `json:"status"`
	Address    string          `json:"address"`
	PublicIp   string          `json:"publicIp"`
	PrivateIp  string          `json:"privateIp"`
	Tags       models.ResAttrs `json:"tags"`
	Attributes models.ResAttrs `json:"attributes"`
	RawData    models.ResAttrs `json:"rawData"`
}

func collectCmdbInventoryAssets(account *cmdbCloudAccount, provider string, regions, assetTypes []string) cmdbCloudCollectResult {
	stats := models.ResAttrs{
		"regions":    regions,
		"assetTypes": assetTypes,
		"mode":       "offline_inventory",
	}
	raw := cmdbInventoryJSON(account, provider)
	if strings.TrimSpace(raw) == "" {
		return cmdbCloudCollectResult{
			Stats: stats,
			Err:   fmt.Errorf("%s collector requires offline inventory json credential; live provider sdk is not wired yet", provider),
		}
	}
	items, err := parseCmdbInventoryAssets(raw)
	if err != nil {
		return cmdbCloudCollectResult{Stats: stats, Err: err}
	}

	selectedTypes := selectedCmdbAssetTypes(assetTypes)
	selectedRegions := selectedCmdbInventoryRegions(regions)
	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0, len(items))
	for _, item := range items {
		assetType := firstNonEmpty(item.AssetType, inventoryAssetType(provider, item.NativeType))
		if assetType == "" || !wantsCmdbAssetType(selectedTypes, assetType) {
			continue
		}
		region := firstNonEmpty(item.Region, firstString(account.Regions))
		if !cmdbInventoryRegionSelected(selectedRegions, region) {
			continue
		}
		nativeType := firstNonEmpty(item.NativeType, inventoryNativeType(provider, assetType))
		nativeId := firstNonEmpty(item.NativeId, item.Id, item.Name)
		asset := newCmdbCloudAsset(account, region, assetType, nativeType, nativeId, firstNonEmpty(item.Name, nativeId), now)
		asset.Zone = item.Zone
		asset.Status = item.Status
		asset.Address = item.Address
		asset.PublicIp = item.PublicIp
		asset.PrivateIp = item.PrivateIp
		asset.Tags = ensureResAttrs(item.Tags)
		asset.Attributes = ensureResAttrs(item.Attributes)
		asset.RawData = ensureResAttrs(item.RawData)
		asset.RawData["accountSource"] = account.Source
		asset.RawData["accountRefId"] = account.Id
		asset.RawData["provider"] = provider
		asset.RawData["region"] = region
		asset.RawData["nativeType"] = nativeType
		asset.RawData["nativeId"] = nativeId
		asset.RawData["inventoryMode"] = "offline_json"
		assets = append(assets, asset)
	}

	stats["totalResources"] = len(items)
	stats["collected"] = len(assets)
	return cmdbCloudCollectResult{
		Assets: assets,
		Stats:  stats,
	}
}

func cmdbInventoryJSON(account *cmdbCloudAccount, provider string) string {
	if account == nil {
		return ""
	}
	keys := []string{"CLOUD_INVENTORY_JSON"}
	switch provider {
	case "tencentcloud":
		keys = append([]string{"TENCENTCLOUD_INVENTORY_JSON", "TENCENT_INVENTORY_JSON"}, keys...)
	case "huawei":
		keys = append([]string{"HUAWEI_INVENTORY_JSON", "HUAWEICLOUD_INVENTORY_JSON"}, keys...)
	}
	for _, key := range keys {
		if value := strings.TrimSpace(account.Credentials[key]); value != "" {
			return value
		}
	}
	return ""
}

func parseCmdbInventoryAssets(raw string) ([]cmdbInventoryAsset, error) {
	items := make([]cmdbInventoryAsset, 0)
	if err := json.Unmarshal([]byte(raw), &items); err == nil {
		return items, nil
	}
	payload := cmdbInventoryPayload{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("decode offline inventory json: %w", err)
	}
	return payload.Assets, nil
}

func inventoryAssetType(provider string, nativeType string) string {
	normalized := strings.ToLower(strings.TrimSpace(nativeType))
	switch provider {
	case "tencentcloud":
		switch normalized {
		case "tencentcloud_instance", "tencentcloud_cvm_instance", "cvm_instance":
			return models.CmdbAssetTypeComputeInstance
		case "tencentcloud_vpc", "vpc":
			return models.CmdbAssetTypeNetworkVpc
		case "tencentcloud_subnet", "subnet":
			return models.CmdbAssetTypeNetworkSubnet
		case "tencentcloud_security_group", "security_group":
			return models.CmdbAssetTypeNetworkSecurityGroup
		case "tencentcloud_eip", "tencentcloud_eip_address", "eip":
			return models.CmdbAssetTypePublicIP
		case "tencentcloud_clb_instance", "clb_instance", "load_balancer":
			return models.CmdbAssetTypeLoadBalancer
		case "tencentcloud_cbs_storage", "cbs_storage", "disk":
			return models.CmdbAssetTypeBlockVolume
		case "tencentcloud_kubernetes_cluster", "tke_cluster":
			return models.CmdbAssetTypeKubernetesCluster
		case "tencentcloud_mysql_instance", "cdb_instance":
			return models.CmdbAssetTypeRelationalDatabase
		case "tencentcloud_redis_instance", "redis_instance":
			return models.CmdbAssetTypeRedisCache
		case "tencentcloud_cos_bucket", "cos_bucket":
			return models.CmdbAssetTypeObjectStorageBucket
		}
	case "huawei":
		switch normalized {
		case "huaweicloud_compute_instance", "huawei_ecs_instance", "ecs_instance":
			return models.CmdbAssetTypeComputeInstance
		case "huaweicloud_vpc", "huawei_vpc", "vpc":
			return models.CmdbAssetTypeNetworkVpc
		case "huaweicloud_vpc_subnet", "huawei_subnet", "subnet":
			return models.CmdbAssetTypeNetworkSubnet
		case "huaweicloud_networking_secgroup", "huawei_security_group", "security_group":
			return models.CmdbAssetTypeNetworkSecurityGroup
		case "huaweicloud_vpc_eip", "huawei_eip", "eip":
			return models.CmdbAssetTypePublicIP
		case "huaweicloud_elb_loadbalancer", "huawei_elb", "load_balancer":
			return models.CmdbAssetTypeLoadBalancer
		case "huaweicloud_evs_volume", "huawei_evs_volume", "evs_volume":
			return models.CmdbAssetTypeBlockVolume
		case "huaweicloud_cce_cluster", "huawei_cce_cluster", "cce_cluster":
			return models.CmdbAssetTypeKubernetesCluster
		case "huaweicloud_rds_instance", "huawei_rds_instance":
			return models.CmdbAssetTypeRelationalDatabase
		case "huaweicloud_dcs_instance", "huawei_dcs_instance":
			return models.CmdbAssetTypeRedisCache
		case "huaweicloud_obs_bucket", "huawei_obs_bucket", "obs_bucket":
			return models.CmdbAssetTypeObjectStorageBucket
		}
	}
	return ""
}

func inventoryNativeType(provider string, assetType string) string {
	switch provider + ":" + assetType {
	case "tencentcloud:" + models.CmdbAssetTypeComputeInstance:
		return "tencentcloud_instance"
	case "tencentcloud:" + models.CmdbAssetTypeNetworkVpc:
		return "tencentcloud_vpc"
	case "tencentcloud:" + models.CmdbAssetTypeNetworkSubnet:
		return "tencentcloud_subnet"
	case "tencentcloud:" + models.CmdbAssetTypeNetworkSecurityGroup:
		return "tencentcloud_security_group"
	case "tencentcloud:" + models.CmdbAssetTypePublicIP:
		return "tencentcloud_eip"
	case "tencentcloud:" + models.CmdbAssetTypeLoadBalancer:
		return "tencentcloud_clb_instance"
	case "tencentcloud:" + models.CmdbAssetTypeBlockVolume:
		return "tencentcloud_cbs_storage"
	case "tencentcloud:" + models.CmdbAssetTypeKubernetesCluster:
		return "tencentcloud_kubernetes_cluster"
	case "tencentcloud:" + models.CmdbAssetTypeRelationalDatabase:
		return "tencentcloud_mysql_instance"
	case "tencentcloud:" + models.CmdbAssetTypeRedisCache:
		return "tencentcloud_redis_instance"
	case "tencentcloud:" + models.CmdbAssetTypeObjectStorageBucket:
		return "tencentcloud_cos_bucket"
	case "huawei:" + models.CmdbAssetTypeComputeInstance:
		return "huaweicloud_compute_instance"
	case "huawei:" + models.CmdbAssetTypeNetworkVpc:
		return "huaweicloud_vpc"
	case "huawei:" + models.CmdbAssetTypeNetworkSubnet:
		return "huaweicloud_vpc_subnet"
	case "huawei:" + models.CmdbAssetTypeNetworkSecurityGroup:
		return "huaweicloud_networking_secgroup"
	case "huawei:" + models.CmdbAssetTypePublicIP:
		return "huaweicloud_vpc_eip"
	case "huawei:" + models.CmdbAssetTypeLoadBalancer:
		return "huaweicloud_elb_loadbalancer"
	case "huawei:" + models.CmdbAssetTypeBlockVolume:
		return "huaweicloud_evs_volume"
	case "huawei:" + models.CmdbAssetTypeKubernetesCluster:
		return "huaweicloud_cce_cluster"
	case "huawei:" + models.CmdbAssetTypeRelationalDatabase:
		return "huaweicloud_rds_instance"
	case "huawei:" + models.CmdbAssetTypeRedisCache:
		return "huaweicloud_dcs_instance"
	case "huawei:" + models.CmdbAssetTypeObjectStorageBucket:
		return "huaweicloud_obs_bucket"
	default:
		return provider + "_" + assetType
	}
}

func selectedCmdbInventoryRegions(regions []string) map[string]bool {
	selected := make(map[string]bool)
	for _, region := range regions {
		region = strings.ToLower(strings.TrimSpace(region))
		if region != "" {
			selected[region] = true
		}
	}
	return selected
}

func cmdbInventoryRegionSelected(selected map[string]bool, region string) bool {
	if len(selected) == 0 {
		return true
	}
	return selected[strings.ToLower(strings.TrimSpace(region))]
}

func ensureResAttrs(attrs models.ResAttrs) models.ResAttrs {
	if attrs == nil {
		return models.ResAttrs{}
	}
	return attrs
}
