// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cloudiac/portal/models"
)

const (
	huaweiDefaultPageSize    = 100
	huaweiMaxCollectionPages = 100
	huaweiDefaultEndpoint    = "myhuaweicloud.com"
)

type huaweiCollectorSpec struct {
	AssetType  string
	Label      string
	Service    string
	Path       string
	ListKeys   []string
	NativeType string
	Normalize  func(*cmdbCloudAccount, string, models.ResAttrs, models.Time) *models.CmdbAsset
}

func collectCmdbHuaweiAssets(account *cmdbCloudAccount, regions, assetTypes []string) cmdbCloudCollectResult {
	if !hasHuaweiLiveCredentials(account.Credentials) {
		return collectCmdbInventoryAssets(account, "huawei", regions, assetTypes)
	}
	stats := models.ResAttrs{
		"regions":    regions,
		"assetTypes": assetTypes,
		"mode":       "huawei_api",
	}
	if len(regions) == 0 {
		return cmdbCloudCollectResult{Stats: stats, Err: fmt.Errorf("huawei collector requires at least one region")}
	}

	selected := selectedCmdbAssetTypes(assetTypes)
	projectId := huaweiProjectId(account)
	hasToken := hasHuaweiTokenCredentials(account.Credentials)
	specs := []huaweiCollectorSpec{
		{models.CmdbAssetTypeComputeInstance, "ecs servers", "ecs", "/v1/%s/cloudservers/detail", []string{"servers"}, "huawei_ecs_server", huaweiServerAsset},
		{models.CmdbAssetTypeNetworkVpc, "vpcs", "vpc", "/v1/%s/vpcs", []string{"vpcs"}, "huawei_vpc", huaweiVpcAsset},
		{models.CmdbAssetTypeNetworkSubnet, "subnets", "vpc", "/v1/%s/subnets", []string{"subnets"}, "huawei_subnet", huaweiSubnetAsset},
		{models.CmdbAssetTypeNetworkSecurityGroup, "security groups", "vpc", "/v1/%s/security-groups", []string{"security_groups", "securityGroups"}, "huawei_security_group", huaweiSecurityGroupAsset},
		{models.CmdbAssetTypePublicIP, "public ips", "vpc", "/v1/%s/publicips", []string{"publicips", "public_ips"}, "huawei_public_ip", huaweiPublicIPAsset},
		{models.CmdbAssetTypeBlockVolume, "evs volumes", "evs", "/v2/%s/volumes/detail", []string{"volumes"}, "huawei_evs_volume", huaweiVolumeAsset},
		{models.CmdbAssetTypeLoadBalancer, "load balancers", "elb", "/v2/%s/elb/loadbalancers", []string{"loadbalancers", "load_balancers"}, "huawei_elb_loadbalancer", huaweiLoadBalancerAsset},
		{models.CmdbAssetTypeKubernetesCluster, "cce clusters", "cce", "/api/v3/projects/%s/clusters", []string{"items", "clusters"}, "huawei_cce_cluster", huaweiClusterAsset},
		{models.CmdbAssetTypeRelationalDatabase, "rds instances", "rds", "/v3/%s/instances", []string{"instances"}, "huawei_rds_instance", huaweiDatabaseAsset},
		{models.CmdbAssetTypeRedisCache, "dcs instances", "dcs", "/v2/%s/instances", []string{"instances"}, "huawei_dcs_redis_instance", huaweiRedisAsset},
	}

	assets := make([]*models.CmdbAsset, 0)
	errors := make([]string, 0)
	if hasToken && projectId != "" {
		for _, region := range regions {
			region = strings.TrimSpace(region)
			if region == "" {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			for _, spec := range specs {
				if !wantsCmdbAssetType(selected, spec.AssetType) {
					continue
				}
				items, err := collectHuaweiPaged(ctx, account, projectId, region, spec)
				assets = append(assets, items...)
				if err != nil {
					errors = append(errors, fmt.Sprintf("huawei %s %s: %v", region, spec.Label, err))
				}
			}
			cancel()
		}
	} else if wantsHuaweiRegionalAssetTypes(selected) {
		errors = append(errors, "huawei regional collectors require HUAWEI_AUTH_TOKEN and HUAWEI_PROJECT_ID")
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeObjectStorageBucket) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		items, err := collectHuaweiObsBuckets(ctx, account, regions, selectedCloudRegions(regions))
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("huawei OBS buckets: %v", err))
		}
		cancel()
	}

	stats["collected"] = len(assets)
	if len(errors) > 0 {
		stats["errors"] = errors
	}
	return cmdbCloudCollectResult{
		Assets: assets,
		Stats:  stats,
		Err:    cmdbCollectError(errors),
	}
}

func hasHuaweiLiveCredentials(credentials map[string]string) bool {
	return hasHuaweiTokenCredentials(credentials) || hasHuaweiObsCredentials(credentials)
}

func hasHuaweiTokenCredentials(credentials map[string]string) bool {
	return huaweiAuthToken(credentials) != "" &&
		firstNonEmpty(credentials["HUAWEI_PROJECT_ID"], credentials["HUAWEICLOUD_PROJECT_ID"], credentials["OS_PROJECT_ID"]) != ""
}

func hasHuaweiObsCredentials(credentials map[string]string) bool {
	return huaweiObsAccessKey(credentials) != "" && huaweiObsSecretKey(credentials) != ""
}

func huaweiProjectId(account *cmdbCloudAccount) string {
	return firstNonEmpty(account.Credentials["HUAWEI_PROJECT_ID"], account.Credentials["HUAWEICLOUD_PROJECT_ID"], account.Credentials["OS_PROJECT_ID"], account.AccountId)
}

func huaweiAuthToken(credentials map[string]string) string {
	return firstNonEmpty(credentials["HUAWEI_AUTH_TOKEN"], credentials["HUAWEICLOUD_AUTH_TOKEN"], credentials["OS_AUTH_TOKEN"])
}

func huaweiObsAccessKey(credentials map[string]string) string {
	return firstNonEmpty(credentials["HUAWEI_ACCESS_KEY"], credentials["HUAWEICLOUD_ACCESS_KEY"], credentials["HUAWEI_OBS_ACCESS_KEY"], credentials["OBS_ACCESS_KEY_ID"])
}

func huaweiObsSecretKey(credentials map[string]string) string {
	return firstNonEmpty(credentials["HUAWEI_SECRET_KEY"], credentials["HUAWEICLOUD_SECRET_KEY"], credentials["HUAWEI_OBS_SECRET_KEY"], credentials["OBS_SECRET_ACCESS_KEY"])
}

func wantsHuaweiRegionalAssetTypes(selected map[string]bool) bool {
	regionalTypes := []string{
		models.CmdbAssetTypeComputeInstance,
		models.CmdbAssetTypeNetworkVpc,
		models.CmdbAssetTypeNetworkSubnet,
		models.CmdbAssetTypeNetworkSecurityGroup,
		models.CmdbAssetTypePublicIP,
		models.CmdbAssetTypeBlockVolume,
		models.CmdbAssetTypeLoadBalancer,
		models.CmdbAssetTypeKubernetesCluster,
		models.CmdbAssetTypeRelationalDatabase,
		models.CmdbAssetTypeRedisCache,
	}
	for _, assetType := range regionalTypes {
		if wantsCmdbAssetType(selected, assetType) {
			return true
		}
	}
	return false
}

func collectHuaweiPaged(ctx context.Context, account *cmdbCloudAccount, projectId, region string, spec huaweiCollectorSpec) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	var firstErr error
	for page := 0; page < huaweiMaxCollectionPages; page++ {
		query := url.Values{}
		query.Set("limit", strconvItoa(huaweiDefaultPageSize))
		query.Set("offset", strconvItoa(page*huaweiDefaultPageSize))
		resp, err := huaweiJSONAPI(ctx, account, spec.Service, region, fmt.Sprintf(spec.Path, url.PathEscape(projectId)), query)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			break
		}
		items := huaweiResponseList(resp, spec.ListKeys...)
		for _, item := range items {
			asset := spec.Normalize(account, region, item, now)
			if asset == nil {
				continue
			}
			asset.NativeType = spec.NativeType
			asset.RawData["mode"] = "huawei_api"
			assets = append(assets, asset)
		}
		if len(items) < huaweiDefaultPageSize {
			break
		}
	}
	return assets, firstErr
}

type huaweiObsBucket struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
	Location     string `xml:"Location"`
}

type huaweiObsListBucketsResult struct {
	Buckets struct {
		Bucket []huaweiObsBucket `xml:"Bucket"`
	} `xml:"Buckets"`
}

func collectHuaweiObsBuckets(ctx context.Context, account *cmdbCloudAccount, regions []string, selectedRegions map[string]bool) ([]*models.CmdbAsset, error) {
	accessKey := huaweiObsAccessKey(account.Credentials)
	secretKey := huaweiObsSecretKey(account.Credentials)
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("huawei OBS collector requires HUAWEI_ACCESS_KEY and HUAWEI_SECRET_KEY")
	}
	endpoint := strings.TrimSpace(account.Credentials["HUAWEI_OBS_ENDPOINT"])
	if endpoint == "" {
		endpointSuffix := strings.TrimSpace(firstNonEmpty(account.Credentials["HUAWEI_ENDPOINT_SUFFIX"], account.Credentials["HUAWEICLOUD_ENDPOINT_SUFFIX"]))
		if endpointSuffix == "" {
			endpointSuffix = huaweiDefaultEndpoint
		}
		endpointSuffix = strings.TrimPrefix(strings.TrimPrefix(endpointSuffix, "https://"), "http://")
		region := ""
		if len(regions) > 0 {
			region = strings.TrimSpace(regions[0])
		}
		if region == "" {
			endpoint = "https://obs." + endpointSuffix + "/"
		} else {
			endpoint = fmt.Sprintf("https://obs.%s.%s/", region, endpointSuffix)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if req.URL.Path == "" {
		req.URL.Path = "/"
	}
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("Date", date)
	req.Header.Set("Authorization", huaweiObsAuthorization(accessKey, secretKey, req.Method, req.URL.Path, date))

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("huawei OBS list buckets status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	out := huaweiObsListBucketsResult{}
	if err := xml.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode huawei OBS buckets: %w", err)
	}
	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0, len(out.Buckets.Bucket))
	for _, bucket := range out.Buckets.Bucket {
		if bucket.Name == "" {
			continue
		}
		region := strings.ToLower(bucket.Location)
		if len(selectedRegions) > 0 && region != "" && !selectedRegions[region] {
			continue
		}
		if region == "" && len(regions) > 0 {
			region = strings.ToLower(strings.TrimSpace(regions[0]))
		}
		asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeObjectStorageBucket, "huaweicloud_obs_bucket", bucket.Name, bucket.Name, now)
		asset.Status = "available"
		asset.Attributes = models.ResAttrs{
			"location":     bucket.Location,
			"creationDate": bucket.CreationDate,
			"endpoint":     endpoint,
		}
		asset.RawData["mode"] = "huawei_obs_api"
		asset.RawData["response"] = bucket
		assets = append(assets, asset)
	}
	return assets, nil
}

func huaweiObsAuthorization(accessKey, secretKey, method, path, date string) string {
	stringToSign := strings.Join([]string{
		method,
		"",
		"",
		date,
		firstNonEmpty(path, "/"),
	}, "\n")
	h := hmac.New(sha1.New, []byte(secretKey))
	_, _ = h.Write([]byte(stringToSign))
	return "OBS " + accessKey + ":" + base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func huaweiJSONAPI(ctx context.Context, account *cmdbCloudAccount, service, region, path string, query url.Values) (models.ResAttrs, error) {
	token := huaweiAuthToken(account.Credentials)
	if token == "" {
		return nil, fmt.Errorf("huawei collector requires HUAWEI_AUTH_TOKEN")
	}
	endpointSuffix := strings.TrimSpace(firstNonEmpty(account.Credentials["HUAWEI_ENDPOINT_SUFFIX"], account.Credentials["HUAWEICLOUD_ENDPOINT_SUFFIX"]))
	if endpointSuffix == "" {
		endpointSuffix = huaweiDefaultEndpoint
	}
	endpointSuffix = strings.TrimPrefix(strings.TrimPrefix(endpointSuffix, "https://"), "http://")
	reqURL := fmt.Sprintf("https://%s.%s.%s%s", service, region, endpointSuffix, path)
	if encoded := query.Encode(); encoded != "" {
		reqURL += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Auth-Token", token)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("huawei api %s status %d: %s", reqURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	out := models.ResAttrs{}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode huawei response: %w", err)
	}
	return out, nil
}

func huaweiResponseList(resp models.ResAttrs, keys ...string) []models.ResAttrs {
	for _, key := range keys {
		if items := resAttrsSlice(resp[key]); len(items) > 0 {
			return items
		}
	}
	return nil
}

func huaweiServerAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "id")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeComputeInstance, "huawei_ecs_server", id, firstNonEmpty(attrString(item, "name"), id), now)
	asset.Zone = attrString(item, "OS-EXT-AZ:availability_zone")
	asset.Status = attrString(item, "status")
	asset.PrivateIp, asset.PublicIp = huaweiServerIPs(item)
	asset.Attributes = models.ResAttrs{
		"flavor":   item["flavor"],
		"image":    item["image"],
		"metadata": item["metadata"],
	}
	asset.RawData["response"] = item
	return asset
}

func huaweiVpcAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "id")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkVpc, "huawei_vpc", id, firstNonEmpty(attrString(item, "name"), id), now)
	asset.Status = firstNonEmpty(attrString(item, "status"), "available")
	asset.Attributes = models.ResAttrs{"cidrBlock": attrString(item, "cidr"), "enterpriseProjectId": attrString(item, "enterprise_project_id")}
	asset.RawData["response"] = item
	return asset
}

func huaweiSubnetAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "id")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkSubnet, "huawei_subnet", id, firstNonEmpty(attrString(item, "name"), id), now)
	asset.Status = firstNonEmpty(attrString(item, "status"), "available")
	asset.Attributes = models.ResAttrs{
		"vpcId":     attrString(item, "vpc_id"),
		"cidrBlock": attrString(item, "cidr"),
		"gatewayIp": attrString(item, "gateway_ip"),
	}
	asset.RawData["response"] = item
	return asset
}

func huaweiSecurityGroupAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "id")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkSecurityGroup, "huawei_security_group", id, firstNonEmpty(attrString(item, "name"), id), now)
	asset.Status = firstNonEmpty(attrString(item, "status"), "available")
	asset.Attributes = models.ResAttrs{
		"description":          attrString(item, "description"),
		"ingressSecurityRules": huaweiSecurityRules(item, "ingress"),
		"egressSecurityRules":  huaweiSecurityRules(item, "egress"),
	}
	asset.RawData["response"] = item
	return asset
}

func huaweiPublicIPAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "id")
	publicIP := firstNonEmpty(attrString(item, "public_ip_address"), attrString(item, "public_ip"))
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypePublicIP, "huawei_public_ip", id, firstNonEmpty(publicIP, id), now)
	asset.Status = attrString(item, "status")
	asset.PublicIp = publicIP
	asset.Address = publicIP
	asset.Attributes = models.ResAttrs{
		"type":        attrString(modelResAttrs(item["type"]), "type"),
		"privateIp":   attrString(item, "private_ip_address"),
		"bandwidth":   item["bandwidth"],
		"portId":      attrString(item, "port_id"),
		"tenantId":    attrString(item, "tenant_id"),
		"ipVersion":   item["ip_version"],
		"publicIpRaw": item["publicip"],
	}
	asset.RawData["response"] = item
	return asset
}

func huaweiVolumeAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "id")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeBlockVolume, "huawei_evs_volume", id, firstNonEmpty(attrString(item, "name"), id), now)
	asset.Zone = attrString(item, "availability_zone")
	asset.Status = attrString(item, "status")
	asset.Attributes = models.ResAttrs{
		"sizeGiB":     item["size"],
		"volumeType":  attrString(item, "volume_type"),
		"attachments": item["attachments"],
		"bootable":    attrString(item, "bootable"),
	}
	asset.RawData["response"] = item
	return asset
}

func huaweiLoadBalancerAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "id")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeLoadBalancer, "huawei_elb_loadbalancer", id, firstNonEmpty(attrString(item, "name"), id), now)
	asset.Status = firstNonEmpty(attrString(item, "provisioning_status"), attrString(item, "operating_status"))
	asset.Address = attrString(item, "vip_address")
	asset.PublicIp = asset.Address
	asset.Attributes = models.ResAttrs{
		"vpcId":           attrString(item, "vpc_id"),
		"vipSubnetCidr":   attrString(item, "vip_subnet_cidr_id"),
		"provider":        attrString(item, "provider"),
		"listeners":       item["listeners"],
		"operatingStatus": attrString(item, "operating_status"),
	}
	asset.RawData["response"] = item
	return asset
}

func huaweiClusterAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	metadata := modelResAttrs(item["metadata"])
	spec := modelResAttrs(item["spec"])
	status := modelResAttrs(item["status"])
	id := firstNonEmpty(attrString(metadata, "uid"), attrString(item, "id"), attrString(metadata, "name"))
	name := firstNonEmpty(attrString(metadata, "name"), attrString(item, "name"), id)
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeKubernetesCluster, "huawei_cce_cluster", id, name, now)
	asset.Status = firstNonEmpty(attrString(status, "phase"), attrString(item, "status"))
	asset.Attributes = models.ResAttrs{
		"clusterVersion": firstNonEmpty(attrString(spec, "version"), attrString(item, "version")),
		"clusterType":    firstNonEmpty(attrString(spec, "type"), attrString(item, "type")),
		"spec":           spec,
	}
	asset.RawData["response"] = item
	return asset
}

func huaweiDatabaseAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "id")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeRelationalDatabase, "huawei_rds_instance", id, firstNonEmpty(attrString(item, "name"), id), now)
	asset.Status = attrString(item, "status")
	asset.PrivateIp = strings.Join(stringSlice(item["private_ips"]), ",")
	asset.PublicIp = strings.Join(stringSlice(item["public_ips"]), ",")
	asset.Attributes = models.ResAttrs{
		"type":      attrString(item, "type"),
		"datastore": item["datastore"],
		"flavorRef": attrString(item, "flavor_ref"),
		"volume":    item["volume"],
	}
	asset.RawData["response"] = item
	return asset
}

func huaweiRedisAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := firstNonEmpty(attrString(item, "instance_id"), attrString(item, "id"))
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeRedisCache, "huawei_dcs_redis_instance", id, firstNonEmpty(attrString(item, "name"), id), now)
	asset.Status = attrString(item, "status")
	asset.PrivateIp = firstNonEmpty(attrString(item, "ip"), attrString(item, "private_ip"))
	asset.Attributes = models.ResAttrs{
		"engine":     attrString(item, "engine"),
		"engineType": attrString(item, "engine_type"),
		"capacity":   item["capacity"],
		"port":       item["port"],
	}
	asset.RawData["response"] = item
	return asset
}

func huaweiServerIPs(item models.ResAttrs) (string, string) {
	privateIPs := make([]string, 0)
	publicIPs := make([]string, 0)
	addresses := modelResAttrs(item["addresses"])
	for _, value := range addresses {
		for _, addr := range resAttrsSlice(value) {
			ip := firstNonEmpty(attrString(addr, "addr"), attrString(addr, "ip"))
			if ip == "" {
				continue
			}
			switch strings.ToLower(firstNonEmpty(attrString(addr, "OS-EXT-IPS:type"), attrString(addr, "type"))) {
			case "floating", "public":
				publicIPs = append(publicIPs, ip)
			default:
				privateIPs = append(privateIPs, ip)
			}
		}
	}
	return strings.Join(privateIPs, ","), strings.Join(publicIPs, ",")
}

func huaweiSecurityRules(item models.ResAttrs, direction string) []models.ResAttrs {
	out := make([]models.ResAttrs, 0)
	for _, rule := range resAttrsSlice(item["security_group_rules"]) {
		if strings.EqualFold(attrString(rule, "direction"), direction) {
			out = append(out, rule)
		}
	}
	return out
}

func strconvItoa(value int) string {
	return fmt.Sprintf("%d", value)
}
