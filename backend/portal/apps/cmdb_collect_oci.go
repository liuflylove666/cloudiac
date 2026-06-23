// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cloudiac/portal/models"
)

const (
	ociCoreAPIVersion            = "20160918"
	ociLoadBalancerAPIVersion    = "20170115"
	ociContainerEngineAPIVersion = "20180222"
	ociRedisAPIVersion           = "20220315"
	ociAPIRequestMaxAttempts     = 3
	ociAPIRequestBaseDelay       = 500 * time.Millisecond
	ociAPIRequestMaxRetryDelay   = 5 * time.Second
)

func collectCmdbOciAssets(account *cmdbCloudAccount, regions, assetTypes []string) cmdbCloudCollectResult {
	stats := models.ResAttrs{
		"regions":    regions,
		"assetTypes": assetTypes,
	}
	if len(regions) == 0 {
		return cmdbCloudCollectResult{
			Stats: stats,
			Err:   fmt.Errorf("oci collector requires at least one region"),
		}
	}

	selected := selectedCmdbAssetTypes(assetTypes)
	assets := make([]*models.CmdbAsset, 0)
	errors := make([]string, 0)
	compartmentStats := make(map[string][]string)
	compartmentDetailStats := make(map[string][]models.ResAttrs)
	apiMetrics := make(map[string][]models.ResAttrs)
	for _, region := range regions {
		region = strings.TrimSpace(region)
		if region == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		recorder := &ociAPIMetricsRecorder{}
		ctx = context.WithValue(ctx, ociAPIMetricsContextKey{}, recorder)
		regionAssets, regionCompartments, regionCompartmentDetails, regionErrors := collectCmdbOciRegionAssets(ctx, account, region, selected)
		cancel()
		assets = append(assets, regionAssets...)
		compartmentStats[region] = regionCompartments
		compartmentDetailStats[region] = regionCompartmentDetails
		if metrics := recorder.List(); len(metrics) > 0 {
			apiMetrics[region] = metrics
		}
		errors = append(errors, regionErrors...)
	}

	stats["collected"] = len(assets)
	stats["compartments"] = compartmentStats
	stats["compartmentDetails"] = compartmentDetailStats
	if len(apiMetrics) > 0 {
		stats["apiMetrics"] = apiMetrics
		stats["apiMetricSummary"] = ociAPIMetricSummary(apiMetrics)
	}
	if len(errors) > 0 {
		stats["errors"] = errors
	}
	return cmdbCloudCollectResult{
		Assets: assets,
		Stats:  stats,
		Err:    cmdbCollectError(errors),
	}
}

type ociCompartmentRef struct {
	Id             string
	Name           string
	Description    string
	ParentId       string
	Path           string
	LifecycleState string
}

func collectCmdbOciRegionAssets(ctx context.Context, account *cmdbCloudAccount, region string, selected map[string]bool) ([]*models.CmdbAsset, []string, []models.ResAttrs, []string) {
	assets := make([]*models.CmdbAsset, 0)
	errors := make([]string, 0)

	compartments, err := ociCompartments(ctx, account, region)
	if err != nil {
		errors = append(errors, fmt.Sprintf("oci %s compartments: %v", region, err))
	}
	for _, compartment := range compartments {
		compartmentId := compartment.Id
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeComputeInstance) {
			items, err := collectOciInstances(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s compute instances in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkVpc) {
			items, err := collectOciVcns(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s vcns in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkSubnet) {
			items, err := collectOciSubnets(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s subnets in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkRouteTable) {
			items, err := collectOciRouteTables(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s route tables in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkNatGateway) {
			items, err := collectOciNatGateways(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s nat gateways in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkInternetGateway) {
			items, err := collectOciInternetGateways(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s internet gateways in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkServiceGateway) {
			items, err := collectOciServiceGateways(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s service gateways in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkDrg) {
			items, err := collectOciDrgs(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s drgs in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkSecurityGroup) {
			items, err := collectOciSecurityAssets(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s security assets in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypePublicIP) {
			items, err := collectOciPublicIps(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s public ips in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeBlockVolume) {
			items, err := collectOciVolumes(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s block volumes in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeLoadBalancer) {
			items, err := collectOciLoadBalancers(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s load balancers in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeObjectStorageBucket) {
			items, err := collectOciBuckets(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s buckets in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeKubernetesCluster) {
			items, err := collectOciOkeClusters(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s oke clusters in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeRelationalDatabase) {
			items, err := collectOciDatabases(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s databases in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeRedisCache) {
			items, err := collectOciRedisCaches(ctx, account, region, compartmentId)
			annotateOciCompartmentAssets(account, items, compartment)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s redis caches in %s: %v", region, compartmentId, err))
			}
		}
	}
	return assets, ociCompartmentRefIDs(compartments), ociCompartmentRefStats(compartments), errors
}

func ociCompartmentIDs(ctx context.Context, account *cmdbCloudAccount, region string) ([]string, error) {
	compartments, err := ociCompartments(ctx, account, region)
	return ociCompartmentRefIDs(compartments), err
}

func ociCompartments(ctx context.Context, account *cmdbCloudAccount, region string) ([]ociCompartmentRef, error) {
	configured := normalizeStringList([]string{
		account.Credentials["OCI_COMPARTMENT_OCIDS"],
		account.Credentials["OCI_COMPARTMENT_OCID"],
	})
	tenancyId := strings.TrimSpace(account.Credentials["OCI_TENANCY_OCID"])
	if len(configured) == 0 {
		configured = []string{tenancyId}
	}
	configured = dedupeStrings(configured)
	if !ociCredentialBool(account, "OCI_INCLUDE_SUBCOMPARTMENTS", "OCI_COMPARTMENT_IN_SUBTREE") {
		return buildOciCompartmentRefs(account, configured, nil), nil
	}

	query := url.Values{
		"compartmentId":          []string{tenancyId},
		"compartmentIdInSubtree": []string{"true"},
		"accessLevel":            []string{"ACCESSIBLE"},
		"lifecycleState":         []string{"ACTIVE"},
	}
	items, err := ociListAPI(ctx, account, region, "identity", ociCoreAPIVersion, "/compartments", query)
	if err != nil {
		return buildOciCompartmentRefs(account, configured, nil), err
	}
	return buildOciCompartmentRefs(account, configured, items), nil
}

func buildOciCompartmentRefs(account *cmdbCloudAccount, configured []string, items []models.ResAttrs) []ociCompartmentRef {
	tenancyId := strings.TrimSpace(account.Credentials["OCI_TENANCY_OCID"])
	configured = dedupeStrings(nonEmptyStrings(configured))
	if len(configured) == 0 && tenancyId != "" {
		configured = []string{tenancyId}
	}

	byId := make(map[string]ociCompartmentRef)
	order := make([]string, 0)
	add := func(ref ociCompartmentRef) {
		ref.Id = strings.TrimSpace(ref.Id)
		if ref.Id == "" {
			return
		}
		ref.Name = strings.TrimSpace(ref.Name)
		ref.ParentId = strings.TrimSpace(ref.ParentId)
		ref.Path = strings.TrimSpace(ref.Path)
		if ref.Id == tenancyId {
			ref.Name = firstNonEmpty(ref.Name, strings.TrimSpace(account.Credentials["OCI_TENANCY_NAME"]), "tenancy")
			ref.LifecycleState = firstNonEmpty(ref.LifecycleState, "ACTIVE")
		}
		if ref.Name == "" {
			ref.Name = ref.Id
		}
		if _, exists := byId[ref.Id]; !exists {
			order = append(order, ref.Id)
		}
		byId[ref.Id] = ref
	}

	for _, id := range configured {
		add(ociCompartmentRef{Id: id})
	}
	for _, item := range items {
		add(ociCompartmentRef{
			Id:             ociAttrString(item, "id"),
			Name:           ociAttrString(item, "name"),
			Description:    ociAttrString(item, "description"),
			ParentId:       ociAttrString(item, "compartmentId"),
			LifecycleState: ociAttrString(item, "lifecycleState"),
		})
	}

	var buildPath func(string, map[string]bool) string
	buildPath = func(id string, visiting map[string]bool) string {
		ref, ok := byId[id]
		if !ok {
			return ""
		}
		if ref.Path != "" {
			return ref.Path
		}
		if visiting[id] {
			return "/" + ref.Name
		}
		visiting[id] = true
		parentPath := ""
		if ref.ParentId != "" && ref.ParentId != ref.Id {
			parentPath = buildPath(ref.ParentId, visiting)
		}
		if parentPath == "" {
			ref.Path = "/" + ref.Name
		} else {
			ref.Path = strings.TrimRight(parentPath, "/") + "/" + ref.Name
		}
		byId[id] = ref
		delete(visiting, id)
		return ref.Path
	}

	refs := make([]ociCompartmentRef, 0, len(order))
	for _, id := range order {
		ref := byId[id]
		ref.Path = buildPath(id, map[string]bool{})
		refs = append(refs, ref)
	}
	return refs
}

func ociCompartmentRefIDs(refs []ociCompartmentRef) []string {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.Id)
	}
	return dedupeStrings(nonEmptyStrings(ids))
}

func ociCompartmentRefStats(refs []ociCompartmentRef) []models.ResAttrs {
	stats := make([]models.ResAttrs, 0, len(refs))
	for _, ref := range refs {
		stats = append(stats, models.ResAttrs{
			"id":             ref.Id,
			"name":           ref.Name,
			"path":           ref.Path,
			"parentId":       ref.ParentId,
			"lifecycleState": ref.LifecycleState,
			"description":    ref.Description,
		})
	}
	return stats
}

func annotateOciCompartmentAssets(account *cmdbCloudAccount, assets []*models.CmdbAsset, compartment ociCompartmentRef) {
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		annotateOciCompartmentAsset(account, asset, compartment)
	}
}

func annotateOciCompartmentAsset(account *cmdbCloudAccount, asset *models.CmdbAsset, compartment ociCompartmentRef) {
	if asset == nil {
		return
	}
	if asset.Attributes == nil {
		asset.Attributes = models.ResAttrs{}
	}
	if asset.RawData == nil {
		asset.RawData = models.ResAttrs{}
	}
	tenancyId := strings.TrimSpace(account.Credentials["OCI_TENANCY_OCID"])
	tenancyName := strings.TrimSpace(account.Credentials["OCI_TENANCY_NAME"])
	userId := strings.TrimSpace(account.Credentials["OCI_USER_OCID"])
	fingerprint := strings.TrimSpace(account.Credentials["OCI_FINGERPRINT"])

	asset.Attributes["tenancyId"] = tenancyId
	if tenancyName != "" {
		asset.Attributes["tenancyName"] = tenancyName
	}
	if userId != "" {
		asset.Attributes["userId"] = userId
	}
	if fingerprint != "" {
		asset.Attributes["credentialFingerprint"] = fingerprint
	}
	asset.Attributes["compartmentId"] = compartment.Id
	asset.Attributes["compartmentName"] = compartment.Name
	asset.Attributes["compartmentPath"] = compartment.Path
	asset.Attributes["compartmentParentId"] = compartment.ParentId
	asset.Attributes["compartmentLifecycleState"] = compartment.LifecycleState
	asset.RawData["tenancyId"] = tenancyId
	asset.RawData["compartmentId"] = compartment.Id
	asset.RawData["compartmentName"] = compartment.Name
	asset.RawData["compartmentPath"] = compartment.Path
	asset.RawData["compartmentParentId"] = compartment.ParentId
}

func collectOciInstances(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	return collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeComputeInstance, "oci_core_instance",
		"iaas", ociCoreAPIVersion, "/instances", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Zone = ociAttrString(item, "availabilityDomain")
			asset.Attributes["shape"] = ociAttrString(item, "shape")
			asset.Attributes["imageId"] = ociNestedAttrString(item, "sourceDetails", "imageId")
			if asset.NativeId != "" {
				networkAttrs, _ := ociInstanceNetworkAttrs(ctx, account, region, compartmentId, asset.NativeId)
				for key, value := range networkAttrs {
					asset.Attributes[key] = value
				}
				asset.PublicIp = firstNonEmpty(ociStringFromInterface(networkAttrs["publicIp"]), asset.PublicIp)
				asset.PrivateIp = firstNonEmpty(ociStringFromInterface(networkAttrs["privateIp"]), asset.PrivateIp)
			}
		})
}

func collectOciVcns(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	return collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeNetworkVpc, "oci_core_vcn",
		"iaas", ociCoreAPIVersion, "/vcns", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Attributes["cidrBlock"] = ociAttrString(item, "cidrBlock")
			asset.Attributes["cidrBlocks"] = item["cidrBlocks"]
			asset.Attributes["dnsLabel"] = ociAttrString(item, "dnsLabel")
		})
}

func collectOciSubnets(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	return collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeNetworkSubnet, "oci_core_subnet",
		"iaas", ociCoreAPIVersion, "/subnets", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Zone = ociAttrString(item, "availabilityDomain")
			asset.Attributes["cidrBlock"] = ociAttrString(item, "cidrBlock")
			asset.Attributes["vcnId"] = ociAttrString(item, "vcnId")
			asset.Attributes["routeTableId"] = ociAttrString(item, "routeTableId")
			asset.Attributes["securityListIds"] = item["securityListIds"]
			asset.Attributes["prohibitPublicIpOnVnic"] = item["prohibitPublicIpOnVnic"]
		})
}

func collectOciRouteTables(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	return collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeNetworkRouteTable, "oci_core_route_table",
		"iaas", ociCoreAPIVersion, "/routeTables", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Attributes["vcnId"] = ociAttrString(item, "vcnId")
			asset.Attributes["routeRules"] = item["routeRules"]
			for key, value := range ociRouteRuleRefs(item["routeRules"]) {
				asset.Attributes[key] = value
			}
		})
}

func collectOciNatGateways(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	return collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeNetworkNatGateway, "oci_core_nat_gateway",
		"iaas", ociCoreAPIVersion, "/natGateways", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Attributes["vcnId"] = ociAttrString(item, "vcnId")
			asset.Attributes["blockTraffic"] = item["blockTraffic"]
		})
}

func collectOciInternetGateways(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	return collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeNetworkInternetGateway, "oci_core_internet_gateway",
		"iaas", ociCoreAPIVersion, "/internetGateways", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Attributes["vcnId"] = ociAttrString(item, "vcnId")
			asset.Attributes["isEnabled"] = item["isEnabled"]
		})
}

func collectOciServiceGateways(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	return collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeNetworkServiceGateway, "oci_core_service_gateway",
		"iaas", ociCoreAPIVersion, "/serviceGateways", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Attributes["vcnId"] = ociAttrString(item, "vcnId")
			asset.Attributes["services"] = item["services"]
			asset.Attributes["blockTraffic"] = item["blockTraffic"]
			asset.Attributes["serviceIds"] = ociServiceGatewayServiceIds(item["services"])
		})
}

func collectOciDrgs(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	return collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeNetworkDrg, "oci_core_drg",
		"iaas", ociCoreAPIVersion, "/drgs", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Attributes["defaultDrgRouteTables"] = item["defaultDrgRouteTables"]
			asset.Attributes["defaultExportDrgRouteDistributionId"] = ociAttrString(item, "defaultExportDrgRouteDistributionId")
		})
}

func collectOciSecurityAssets(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	securityLists, err := collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeNetworkSecurityGroup, "oci_core_security_list",
		"iaas", ociCoreAPIVersion, "/securityLists", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Attributes["vcnId"] = ociAttrString(item, "vcnId")
			asset.Attributes["egressSecurityRules"] = item["egressSecurityRules"]
			asset.Attributes["ingressSecurityRules"] = item["ingressSecurityRules"]
		})
	assets = append(assets, securityLists...)
	if err != nil {
		return assets, err
	}

	nsgs, err := collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeNetworkSecurityGroup, "oci_core_network_security_group",
		"iaas", ociCoreAPIVersion, "/networkSecurityGroups", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Attributes["vcnId"] = ociAttrString(item, "vcnId")
		})
	assets = append(assets, nsgs...)
	return assets, err
}

func collectOciPublicIps(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	query := url.Values{"scope": []string{"REGION"}}
	return collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypePublicIP, "oci_core_public_ip",
		"iaas", ociCoreAPIVersion, "/publicIps", query, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Address = ociAttrString(item, "ipAddress")
			asset.PublicIp = asset.Address
			asset.Attributes["lifetime"] = ociAttrString(item, "lifetime")
			asset.Attributes["privateIpId"] = ociAttrString(item, "privateIpId")
		})
}

func collectOciVolumes(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	return collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeBlockVolume, "oci_core_volume",
		"iaas", ociCoreAPIVersion, "/volumes", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Zone = ociAttrString(item, "availabilityDomain")
			asset.Attributes["sizeInGBs"] = item["sizeInGBs"]
			asset.Attributes["vpusPerGB"] = item["vpusPerGB"]
			asset.Attributes["isAutoTuneEnabled"] = item["isAutoTuneEnabled"]
		})
}

func collectOciLoadBalancers(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	return collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeLoadBalancer, "oci_load_balancer",
		"iaas", ociLoadBalancerAPIVersion, "/loadBalancers", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Address = firstOciIPAddress(item["ipAddresses"])
			asset.Attributes["shapeName"] = ociAttrString(item, "shapeName")
			asset.Attributes["isPrivate"] = item["isPrivate"]
			asset.Attributes["subnetIds"] = item["subnetIds"]
			asset.Attributes["networkSecurityGroupIds"] = item["networkSecurityGroupIds"]
		})
}

func collectOciBuckets(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	namespaceName, err := ociObjectStorageNamespace(ctx, account, region)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/n/%s/b", url.PathEscape(namespaceName))
	query := url.Values{"fields": []string{"tags"}}
	return collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeObjectStorageBucket, "oci_objectstorage_bucket",
		"objectstorage", "", path, query, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.NativeId = firstNonEmpty(ociAttrString(item, "id"), ociAttrString(item, "name"), asset.NativeId)
			asset.Name = firstNonEmpty(ociAttrString(item, "name"), asset.Name)
			asset.Status = firstNonEmpty(ociAttrString(item, "lifecycleState"), "active")
			asset.Attributes["namespace"] = namespaceName
			asset.Attributes["storageTier"] = ociAttrString(item, "storageTier")
			asset.Attributes["createdBy"] = ociAttrString(item, "createdBy")
		})
}

func collectOciOkeClusters(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	assets, err := collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeKubernetesCluster, "oci_containerengine_cluster",
		"containerengine", ociContainerEngineAPIVersion, "/clusters", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Attributes["kubernetesVersion"] = ociAttrString(item, "kubernetesVersion")
			asset.Attributes["version"] = ociAttrString(item, "kubernetesVersion")
			asset.Attributes["vcnId"] = ociAttrString(item, "vcnId")
			asset.Attributes["endpoints"] = item["endpoints"]
			asset.Attributes["endpointConfig"] = item["endpointConfig"]
			asset.Attributes["options"] = item["options"]
			asset.Address = firstNonEmpty(ociOkeClusterEndpoint(item), asset.Address)
		})
	if err != nil {
		return assets, err
	}
	for _, asset := range assets {
		nodePools, nodePoolErr := collectOciOkeNodePools(ctx, account, region, compartmentId, asset.NativeId)
		if nodePoolErr != nil {
			asset.Attributes["nodePoolCollectError"] = nodePoolErr.Error()
			continue
		}
		asset.Attributes["nodePools"] = nodePools
		asset.Attributes["nodePoolCount"] = len(nodePools)
	}
	return assets, nil
}

func ociOkeClusterEndpoint(item models.ResAttrs) string {
	endpoints := modelResAttrs(item["endpoints"])
	return firstNonEmpty(
		attrString(endpoints, "kubernetes"),
		attrString(endpoints, "publicEndpoint"),
		attrString(endpoints, "privateEndpoint"),
		attrString(endpoints, "endpoint"),
	)
}

func collectOciOkeNodePools(ctx context.Context, account *cmdbCloudAccount, region, compartmentId, clusterId string) ([]models.ResAttrs, error) {
	query := url.Values{
		"compartmentId": []string{compartmentId},
		"clusterId":     []string{clusterId},
	}
	items, err := ociListAPI(ctx, account, region, "containerengine", ociContainerEngineAPIVersion, "/nodePools", query)
	if err != nil {
		return nil, err
	}
	nodePools := make([]models.ResAttrs, 0, len(items))
	for _, item := range items {
		nodePools = append(nodePools, models.ResAttrs{
			"id":                  ociAttrString(item, "id"),
			"name":                firstNonEmpty(ociAttrString(item, "name"), ociAttrString(item, "displayName")),
			"status":              ociAttrString(item, "lifecycleState"),
			"kubernetesVersion":   ociAttrString(item, "kubernetesVersion"),
			"nodeShape":           ociAttrString(item, "nodeShape"),
			"nodeShapeConfig":     item["nodeShapeConfig"],
			"nodeSourceDetails":   item["nodeSourceDetails"],
			"nodeConfigDetails":   item["nodeConfigDetails"],
			"initialNodeLabels":   item["initialNodeLabels"],
			"quantityPerSubnet":   item["quantityPerSubnet"],
			"subnetIds":           item["subnetIds"],
			"availabilityDomains": item["availabilityDomains"],
			"timeCreated":         item["timeCreated"],
			"timeUpdated":         item["timeUpdated"],
		})
	}
	return nodePools, nil
}

func collectOciDatabases(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	dbSystems, err := collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeRelationalDatabase, "oci_database_db_system",
		"database", ociCoreAPIVersion, "/dbSystems", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Zone = ociAttrString(item, "availabilityDomain")
			asset.Attributes["databaseEdition"] = ociAttrString(item, "databaseEdition")
			asset.Attributes["shape"] = ociAttrString(item, "shape")
			asset.Attributes["hostname"] = ociAttrString(item, "hostname")
			asset.Attributes["subnetId"] = ociAttrString(item, "subnetId")
			asset.Attributes["dataStorageSizeInGBs"] = item["dataStorageSizeInGBs"]
		})
	assets = append(assets, dbSystems...)
	if err != nil {
		return assets, err
	}

	autonomousDatabases, err := collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeRelationalDatabase, "oci_database_autonomous_database",
		"database", ociCoreAPIVersion, "/autonomousDatabases", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Attributes["dbName"] = ociAttrString(item, "dbName")
			asset.Attributes["dbVersion"] = ociAttrString(item, "dbVersion")
			asset.Attributes["dbWorkload"] = ociAttrString(item, "dbWorkload")
			asset.Attributes["cpuCoreCount"] = item["cpuCoreCount"]
			asset.Attributes["dataStorageSizeInTBs"] = item["dataStorageSizeInTBs"]
		})
	assets = append(assets, autonomousDatabases...)
	return assets, err
}

func collectOciRedisCaches(ctx context.Context, account *cmdbCloudAccount, region, compartmentId string) ([]*models.CmdbAsset, error) {
	return collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeRedisCache, "oci_redis_cluster",
		"redis", ociRedisAPIVersion, "/redisClusters", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Attributes["nodeCount"] = item["nodeCount"]
			asset.Attributes["nodeMemoryInGBs"] = item["nodeMemoryInGBs"]
			asset.Attributes["softwareVersion"] = ociAttrString(item, "softwareVersion")
			asset.Attributes["subnetId"] = ociAttrString(item, "subnetId")
			asset.Attributes["nsgIds"] = item["nsgIds"]
			asset.Address = firstNonEmpty(ociNestedAttrString(item, "primaryEndpoint", "fqdn"), ociNestedAttrString(item, "primaryEndpoint", "ipAddress"))
		})
}

func collectOciSimpleAssets(ctx context.Context, account *cmdbCloudAccount, region, compartmentId, assetType, nativeType, service, version, path string, query url.Values, decorate func(*models.CmdbAsset, models.ResAttrs)) ([]*models.CmdbAsset, error) {
	if query == nil {
		query = url.Values{}
	}
	query.Set("compartmentId", compartmentId)
	items, err := ociListAPI(ctx, account, region, service, version, path, query)
	if err != nil {
		return nil, err
	}

	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0, len(items))
	for _, item := range items {
		nativeId := firstNonEmpty(ociAttrString(item, "id"), ociAttrString(item, "ocid"), ociAttrString(item, "name"))
		name := firstNonEmpty(ociAttrString(item, "displayName"), ociAttrString(item, "name"), nativeId)
		asset := newCmdbCloudAsset(account, region, assetType, nativeType, nativeId, name, now)
		asset.Status = ociAttrString(item, "lifecycleState")
		asset.Zone = ociAttrString(item, "availabilityDomain")
		asset.Tags = ociTags(item)
		asset.Attributes = models.ResAttrs(item)
		asset.Attributes["compartmentId"] = compartmentId
		asset.RawData["compartmentId"] = compartmentId
		if decorate != nil {
			decorate(asset, item)
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

func ociRouteRuleRefs(value interface{}) models.ResAttrs {
	natGatewayIds := make([]string, 0)
	internetGatewayIds := make([]string, 0)
	serviceGatewayIds := make([]string, 0)
	drgIds := make([]string, 0)
	gatewayIds := make([]string, 0)
	destinationCidrBlocks := make([]string, 0)
	destinationServiceIds := make([]string, 0)

	for _, rule := range ociResAttrItems(value) {
		networkEntityId := ociAttrString(rule, "networkEntityId")
		if networkEntityId != "" {
			gatewayIds = append(gatewayIds, networkEntityId)
			switch ociNetworkEntityKind(networkEntityId) {
			case "nat_gateway":
				natGatewayIds = append(natGatewayIds, networkEntityId)
			case "internet_gateway":
				internetGatewayIds = append(internetGatewayIds, networkEntityId)
			case "service_gateway":
				serviceGatewayIds = append(serviceGatewayIds, networkEntityId)
			case "drg":
				drgIds = append(drgIds, networkEntityId)
			}
		}

		destination := ociAttrString(rule, "destination", "destinationCidrBlock")
		destinationType := strings.ToLower(ociAttrString(rule, "destinationType"))
		if destination == "" {
			continue
		}
		if strings.Contains(destinationType, "service") {
			destinationServiceIds = append(destinationServiceIds, destination)
		} else {
			destinationCidrBlocks = append(destinationCidrBlocks, destination)
		}
	}

	return models.ResAttrs{
		"natGatewayIds":         dedupeStrings(natGatewayIds),
		"internetGatewayIds":    dedupeStrings(internetGatewayIds),
		"serviceGatewayIds":     dedupeStrings(serviceGatewayIds),
		"drgIds":                dedupeStrings(drgIds),
		"gatewayIds":            dedupeStrings(gatewayIds),
		"destinationCidrBlocks": dedupeStrings(destinationCidrBlocks),
		"destinationServiceIds": dedupeStrings(destinationServiceIds),
	}
}

func ociNetworkEntityKind(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	switch {
	case strings.Contains(id, "natgateway"):
		return "nat_gateway"
	case strings.Contains(id, "internetgateway"):
		return "internet_gateway"
	case strings.Contains(id, "servicegateway"):
		return "service_gateway"
	case strings.Contains(id, ".drg."), strings.Contains(id, "drg"):
		return "drg"
	default:
		return ""
	}
}

func ociServiceGatewayServiceIds(value interface{}) []string {
	ids := make([]string, 0)
	for _, service := range ociResAttrItems(value) {
		if id := ociAttrString(service, "serviceId", "id"); id != "" {
			ids = append(ids, id)
		}
	}
	return dedupeStrings(ids)
}

func ociResAttrItems(value interface{}) []models.ResAttrs {
	switch typed := value.(type) {
	case nil:
		return nil
	case []models.ResAttrs:
		return typed
	case []map[string]interface{}:
		items := make([]models.ResAttrs, 0, len(typed))
		for _, item := range typed {
			items = append(items, models.ResAttrs(item))
		}
		return items
	case []interface{}:
		items := make([]models.ResAttrs, 0, len(typed))
		for _, item := range typed {
			items = append(items, ociResAttrItems(item)...)
		}
		return items
	case models.ResAttrs:
		return []models.ResAttrs{typed}
	case map[string]interface{}:
		return []models.ResAttrs{models.ResAttrs(typed)}
	default:
		return nil
	}
}

func ociInstanceNetworkAttrs(ctx context.Context, account *cmdbCloudAccount, region, compartmentId, instanceId string) (models.ResAttrs, error) {
	query := url.Values{
		"compartmentId": []string{compartmentId},
		"instanceId":    []string{instanceId},
	}
	attachments, err := ociListAPI(ctx, account, region, "iaas", ociCoreAPIVersion, "/vnicAttachments", query)
	if err != nil {
		return models.ResAttrs{}, err
	}
	for _, attachment := range attachments {
		if !strings.EqualFold(ociAttrString(attachment, "lifecycleState"), "ATTACHED") {
			continue
		}
		vnicId := ociAttrString(attachment, "vnicId")
		if vnicId == "" {
			continue
		}
		vnic := models.ResAttrs{}
		if err := ociJSONAPI(ctx, account, region, "iaas", ociCoreAPIVersion, "/vnics/"+url.PathEscape(vnicId), nil, &vnic); err != nil {
			return models.ResAttrs{}, err
		}
		return models.ResAttrs{
			"vnicId":    vnicId,
			"vcnId":     ociAttrString(vnic, "vcnId"),
			"subnetId":  ociAttrString(vnic, "subnetId"),
			"nsgIds":    vnic["nsgIds"],
			"privateIp": ociAttrString(vnic, "privateIp"),
			"publicIp":  ociAttrString(vnic, "publicIp"),
			"hostname":  ociAttrString(vnic, "hostnameLabel"),
		}, nil
	}
	return models.ResAttrs{}, nil
}

func ociObjectStorageNamespace(ctx context.Context, account *cmdbCloudAccount, region string) (string, error) {
	if namespace := strings.TrimSpace(account.Credentials["OCI_OBJECT_STORAGE_NAMESPACE"]); namespace != "" {
		return namespace, nil
	}
	body, err := ociRawAPI(ctx, account, region, "objectstorage", "", "/n/", nil)
	if err != nil {
		return "", err
	}
	namespace := strings.TrimSpace(string(body))
	namespace = strings.Trim(namespace, `"`)
	if namespace == "" {
		return "", fmt.Errorf("empty OCI object storage namespace")
	}
	return namespace, nil
}

func ociListAPI(ctx context.Context, account *cmdbCloudAccount, region, service, version, path string, query url.Values) ([]models.ResAttrs, error) {
	items := make([]models.ResAttrs, 0)
	page := ""
	for {
		values := cloneURLValues(query)
		if page != "" {
			values.Set("page", page)
		}
		body, headers, err := ociDoAPI(ctx, account, region, service, version, path, values)
		if err != nil {
			return items, err
		}
		pageItems, err := decodeOciListItems(body)
		if err != nil {
			return items, err
		}
		items = append(items, pageItems...)
		page = headers.Get("opc-next-page")
		if page == "" {
			return items, nil
		}
	}
}

func ociJSONAPI(ctx context.Context, account *cmdbCloudAccount, region, service, version, path string, query url.Values, out interface{}) error {
	body, err := ociRawAPI(ctx, account, region, service, version, path, query)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func ociRawAPI(ctx context.Context, account *cmdbCloudAccount, region, service, version, path string, query url.Values) ([]byte, error) {
	body, _, err := ociDoAPI(ctx, account, region, service, version, path, query)
	return body, err
}

type ociAPIError struct {
	StatusCode int
	Status     string
	Code       string
	Message    string
	RequestId  string
	RetryAfter string
	Service    string
	Path       string
	Attempts   int
}

func (e *ociAPIError) Error() string {
	parts := []string{"provider=oci"}
	if e.StatusCode > 0 {
		parts = append(parts, fmt.Sprintf("status=%d", e.StatusCode))
	}
	if value := ociAPIErrorToken(e.Code); value != "" {
		parts = append(parts, "code="+value)
	}
	if value := ociAPIErrorToken(e.RequestId); value != "" {
		parts = append(parts, "requestId="+value)
	}
	if value := ociAPIErrorToken(e.RetryAfter); value != "" {
		parts = append(parts, "retryAfter="+value)
	}
	if value := ociAPIErrorToken(e.Service); value != "" {
		parts = append(parts, "service="+value)
	}
	if value := ociAPIErrorToken(e.Path); value != "" {
		parts = append(parts, "path="+value)
	}
	if e.Attempts > 0 {
		parts = append(parts, fmt.Sprintf("attempts=%d", e.Attempts))
	}
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = strings.TrimSpace(e.Status)
	}
	if message != "" {
		parts = append(parts, "message="+message)
	}
	return strings.Join(parts, " ")
}

func newOciAPIError(resp *http.Response, service, path string, body []byte) *ociAPIError {
	payload := struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{}
	_ = json.Unmarshal(body, &payload)

	message := strings.TrimSpace(payload.Message)
	if message == "" {
		message = strings.TrimSpace(string(body))
	}

	return &ociAPIError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Code:       strings.TrimSpace(payload.Code),
		Message:    message,
		RequestId:  strings.TrimSpace(resp.Header.Get("opc-request-id")),
		RetryAfter: strings.TrimSpace(resp.Header.Get("retry-after")),
		Service:    service,
		Path:       path,
	}
}

func ociAPIErrorToken(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), "_")
}

func ociAPIShouldRetry(statusCode int, err error) bool {
	if err != nil {
		return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
	}
	return statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooManyRequests ||
		statusCode >= http.StatusInternalServerError
}

func ociAPIRetryDelay(retryAfter string, attempt int, now time.Time) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	if seconds, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && seconds >= 0 {
		return ociAPIClampRetryDelay(time.Duration(seconds) * time.Second)
	}
	if retryAt, err := http.ParseTime(strings.TrimSpace(retryAfter)); err == nil {
		return ociAPIClampRetryDelay(retryAt.Sub(now))
	}
	delay := ociAPIRequestBaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
	}
	return ociAPIClampRetryDelay(delay)
}

func ociAPIClampRetryDelay(delay time.Duration) time.Duration {
	if delay < 0 {
		return 0
	}
	if delay > ociAPIRequestMaxRetryDelay {
		return ociAPIRequestMaxRetryDelay
	}
	return delay
}

func ociAPIWait(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type ociAPIMetricsContextKey struct{}

type ociAPIMetricsRecorder struct {
	metrics []models.ResAttrs
}

func (r *ociAPIMetricsRecorder) Add(metric models.ResAttrs) {
	if r == nil || metric == nil {
		return
	}
	r.metrics = append(r.metrics, metric)
}

func (r *ociAPIMetricsRecorder) List() []models.ResAttrs {
	if r == nil || len(r.metrics) == 0 {
		return nil
	}
	items := make([]models.ResAttrs, len(r.metrics))
	copy(items, r.metrics)
	return items
}

func ociRecordAPIMetric(ctx context.Context, metric models.ResAttrs) {
	if ctx == nil {
		return
	}
	recorder, ok := ctx.Value(ociAPIMetricsContextKey{}).(*ociAPIMetricsRecorder)
	if !ok || recorder == nil {
		return
	}
	recorder.Add(metric)
}

func ociBuildAPIMetric(region, service, path string, query url.Values, startedAt time.Time, attempts, statusCode int, requestId, retryAfter string, err error) models.ResAttrs {
	metric := models.ResAttrs{
		"provider":   "oci",
		"region":     region,
		"service":    service,
		"path":       path,
		"durationMs": time.Since(startedAt).Milliseconds(),
		"attempts":   attempts,
	}
	if attempts > 1 {
		metric["retryCount"] = attempts - 1
	}
	if statusCode > 0 {
		metric["httpStatus"] = statusCode
	}
	if requestId = strings.TrimSpace(requestId); requestId != "" {
		metric["requestId"] = requestId
	}
	if retryAfter = strings.TrimSpace(retryAfter); retryAfter != "" {
		metric["retryAfter"] = retryAfter
	}
	if query != nil {
		if compartmentId := strings.TrimSpace(query.Get("compartmentId")); compartmentId != "" {
			metric["compartmentId"] = compartmentId
		}
		if query.Get("page") != "" {
			metric["pageTokenUsed"] = true
		}
	}
	if err != nil {
		category, retryable, retryHint := cmdbSyncClassifyFailure(err.Error())
		metric["status"] = "failed"
		metric["error"] = err.Error()
		metric["errorCategory"] = category
		metric["retryable"] = retryable
		metric["retryHint"] = retryHint
		return metric
	}
	metric["status"] = "complete"
	return metric
}

func ociAPIMetricSummary(metricsByRegion map[string][]models.ResAttrs) models.ResAttrs {
	summary := models.ResAttrs{
		"total":           0,
		"failed":          0,
		"retried":         0,
		"maxDurationMs":   int64(0),
		"maxAttempts":     0,
		"slowestEndpoint": "",
	}
	serviceCounts := make(map[string]int)
	for region, metrics := range metricsByRegion {
		for _, metric := range metrics {
			if metric == nil {
				continue
			}
			summary["total"] = summary["total"].(int) + 1
			service := strings.TrimSpace(fmt.Sprint(metric["service"]))
			if service != "" {
				serviceCounts[service]++
			}
			if strings.TrimSpace(fmt.Sprint(metric["status"])) == "failed" {
				summary["failed"] = summary["failed"].(int) + 1
			}
			attempts := ociMetricInt(metric["attempts"])
			if attempts > 1 {
				summary["retried"] = summary["retried"].(int) + 1
			}
			if attempts > summary["maxAttempts"].(int) {
				summary["maxAttempts"] = attempts
			}
			durationMs := int64(ociMetricInt(metric["durationMs"]))
			if durationMs > summary["maxDurationMs"].(int64) {
				summary["maxDurationMs"] = durationMs
				summary["slowestEndpoint"] = strings.TrimSpace(region + " " + service + " " + fmt.Sprint(metric["path"]))
			}
		}
	}
	summary["serviceCounts"] = serviceCounts
	return summary
}

func ociMetricInt(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
		return parsed
	default:
		return 0
	}
}

func ociDoAPI(ctx context.Context, account *cmdbCloudAccount, region, service, version, path string, query url.Values) (body []byte, header http.Header, err error) {
	startedAt := time.Now()
	basePath := path
	if version != "" {
		basePath = "/" + strings.Trim(version, "/") + "/" + strings.TrimLeft(path, "/")
	}
	host := ociServiceHost(service, region)
	var lastHeader http.Header
	attempts := 0
	statusCode := 0
	requestId := ""
	retryAfter := ""
	defer func() {
		ociRecordAPIMetric(ctx, ociBuildAPIMetric(region, service, basePath, query, startedAt, attempts, statusCode, requestId, retryAfter, err))
	}()
	for attempt := 1; attempt <= ociAPIRequestMaxAttempts; attempt++ {
		attempts = attempt
		u := url.URL{
			Scheme:   "https",
			Host:     host,
			Path:     basePath,
			RawQuery: query.Encode(),
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, nil, err
		}
		if err := signOCIRequest(req, account, time.Now().UTC()); err != nil {
			return nil, nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			if attempt < ociAPIRequestMaxAttempts && ociAPIShouldRetry(0, err) {
				if waitErr := ociAPIWait(ctx, ociAPIRetryDelay("", attempt, time.Now().UTC())); waitErr != nil {
					return nil, lastHeader, waitErr
				}
				continue
			}
			return nil, lastHeader, err
		}
		lastHeader = resp.Header
		statusCode = resp.StatusCode
		requestId = strings.TrimSpace(resp.Header.Get("opc-request-id"))
		retryAfter = strings.TrimSpace(resp.Header.Get("retry-after"))
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			if attempt < ociAPIRequestMaxAttempts && ociAPIShouldRetry(0, readErr) {
				if waitErr := ociAPIWait(ctx, ociAPIRetryDelay("", attempt, time.Now().UTC())); waitErr != nil {
					return nil, lastHeader, waitErr
				}
				continue
			}
			return nil, resp.Header, readErr
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, resp.Header, nil
		}
		apiErr := newOciAPIError(resp, service, basePath, body)
		apiErr.Attempts = attempt
		if attempt < ociAPIRequestMaxAttempts && ociAPIShouldRetry(resp.StatusCode, nil) {
			if waitErr := ociAPIWait(ctx, ociAPIRetryDelay(apiErr.RetryAfter, attempt, time.Now().UTC())); waitErr != nil {
				return nil, resp.Header, waitErr
			}
			continue
		}
		return nil, resp.Header, apiErr
	}
	return nil, lastHeader, fmt.Errorf("oci api request failed after %d attempts", ociAPIRequestMaxAttempts)
}

func signOCIRequest(req *http.Request, account *cmdbCloudAccount, now time.Time) error {
	tenancy := strings.TrimSpace(account.Credentials["OCI_TENANCY_OCID"])
	user := strings.TrimSpace(account.Credentials["OCI_USER_OCID"])
	fingerprint := strings.TrimSpace(account.Credentials["OCI_FINGERPRINT"])
	privateKey := strings.TrimSpace(account.Credentials["OCI_PRIVATE_KEY"])
	if tenancy == "" || user == "" || fingerprint == "" || privateKey == "" {
		return fmt.Errorf("missing OCI_TENANCY_OCID, OCI_USER_OCID, OCI_FINGERPRINT or OCI_PRIVATE_KEY")
	}

	date := now.Format(http.TimeFormat)
	req.Header.Set("Date", date)
	req.Header.Set("Host", req.URL.Host)

	requestTarget := strings.ToLower(req.Method) + " " + req.URL.RequestURI()
	signingLines := []string{
		"date: " + date,
		"(request-target): " + requestTarget,
		"host: " + req.URL.Host,
	}
	signedHeaders := []string{"date", "(request-target)", "host"}
	if contentHash := req.Header.Get("X-Content-Sha256"); contentHash != "" {
		signingLines = append(signingLines,
			"x-content-sha256: "+contentHash,
			"content-type: "+req.Header.Get("Content-Type"),
			"content-length: "+req.Header.Get("Content-Length"),
		)
		signedHeaders = append(signedHeaders, "x-content-sha256", "content-type", "content-length")
	}
	signingString := strings.Join(signingLines, "\n")

	key, err := parseOCIRSAPrivateKey(privateKey, account.Credentials["OCI_PRIVATE_KEY_PASSPHRASE"])
	if err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(signingString))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	if err != nil {
		return err
	}
	keyId := strings.Join([]string{tenancy, user, fingerprint}, "/")
	req.Header.Set("Authorization", fmt.Sprintf(`Signature version="1",keyId="%s",algorithm="rsa-sha256",headers="%s",signature="%s"`,
		keyId, strings.Join(signedHeaders, " "), base64.StdEncoding.EncodeToString(signature)))
	return nil
}

func setOCIRequestBodyHeaders(req *http.Request, body []byte) {
	hash := sha256.Sum256(body)
	req.Header.Set("X-Content-Sha256", base64.StdEncoding.EncodeToString(hash[:]))
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
}

func parseOCIRSAPrivateKey(value, passphrase string) (*rsa.PrivateKey, error) {
	value = strings.ReplaceAll(value, `\n`, "\n")
	block, _ := pem.Decode([]byte(value))
	if block == nil {
		return nil, fmt.Errorf("invalid OCI private key PEM")
	}
	if x509.IsEncryptedPEMBlock(block) {
		if strings.TrimSpace(passphrase) == "" {
			return nil, fmt.Errorf("OCI_PRIVATE_KEY_PASSPHRASE is required for encrypted private key")
		}
		decrypted, err := x509.DecryptPEMBlock(block, []byte(passphrase))
		if err != nil {
			return nil, err
		}
		block.Bytes = decrypted
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("OCI private key must be RSA")
	}
	return key, nil
}

func ociServiceHost(service, region string) string {
	switch service {
	case "identity":
		return fmt.Sprintf("identity.%s.oci.oraclecloud.com", region)
	case "objectstorage":
		return fmt.Sprintf("objectstorage.%s.oraclecloud.com", region)
	case "containerengine":
		return fmt.Sprintf("containerengine.%s.oci.oraclecloud.com", region)
	case "database":
		return fmt.Sprintf("database.%s.oraclecloud.com", region)
	case "redis":
		return fmt.Sprintf("redis.%s.oci.oraclecloud.com", region)
	default:
		return fmt.Sprintf("iaas.%s.oraclecloud.com", region)
	}
}

func decodeOciListItems(body []byte) ([]models.ResAttrs, error) {
	rawItems := make([]map[string]interface{}, 0)
	if err := json.Unmarshal(body, &rawItems); err == nil {
		return mapsToResAttrs(rawItems), nil
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	for _, key := range []string{"items", "data"} {
		if raw, ok := payload[key]; ok {
			if err := json.Unmarshal(raw, &rawItems); err != nil {
				return nil, err
			}
			return mapsToResAttrs(rawItems), nil
		}
	}
	return []models.ResAttrs{}, nil
}

func mapsToResAttrs(items []map[string]interface{}) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(items))
	for _, item := range items {
		result = append(result, models.ResAttrs(item))
	}
	return result
}

func ociTags(item models.ResAttrs) models.ResAttrs {
	tags := models.ResAttrs{}
	if freeform, ok := item["freeformTags"].(map[string]interface{}); ok {
		for key, value := range freeform {
			tags[key] = value
		}
	}
	if defined, ok := item["definedTags"].(map[string]interface{}); ok {
		for namespace, values := range defined {
			if fields, ok := values.(map[string]interface{}); ok {
				for key, value := range fields {
					tags[namespace+"."+key] = value
				}
			}
		}
	}
	return tags
}

func ociAttrString(item models.ResAttrs, keys ...string) string {
	for _, key := range keys {
		if value := ociStringFromInterface(item[key]); value != "" {
			return value
		}
	}
	return ""
}

func ociNestedAttrString(item models.ResAttrs, key, nestedKey string) string {
	if nested, ok := item[key].(map[string]interface{}); ok {
		return ociStringFromInterface(nested[nestedKey])
	}
	if nested, ok := item[key].(models.ResAttrs); ok {
		return ociStringFromInterface(nested[nestedKey])
	}
	return ""
}

func ociStringFromInterface(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func firstOciIPAddress(value interface{}) string {
	items, ok := value.([]interface{})
	if !ok {
		return ""
	}
	for _, item := range items {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if ip := ociStringFromInterface(entry["ipAddress"]); ip != "" {
			return ip
		}
	}
	return ""
}

func ociCredentialBool(account *cmdbCloudAccount, keys ...string) bool {
	for _, key := range keys {
		switch strings.ToLower(strings.TrimSpace(account.Credentials[key])) {
		case "1", "true", "yes", "y", "on":
			return true
		}
	}
	return false
}

func cloneURLValues(values url.Values) url.Values {
	cloned := url.Values{}
	for key, list := range values {
		cloned[key] = append([]string{}, list...)
	}
	return cloned
}
