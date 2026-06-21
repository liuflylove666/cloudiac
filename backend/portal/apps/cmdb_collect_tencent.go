// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
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
	tencentDefaultPageSize    = 100
	tencentMaxCollectionPages = 100
)

type tencentCollectorSpec struct {
	AssetType  string
	Label      string
	Service    string
	Version    string
	Action     string
	ListKeys   []string
	NativeType string
	Normalize  func(*cmdbCloudAccount, string, models.ResAttrs, models.Time) *models.CmdbAsset
}

func collectCmdbTencentAssets(account *cmdbCloudAccount, regions, assetTypes []string) cmdbCloudCollectResult {
	if !hasTencentLiveCredentials(account.Credentials) {
		return collectCmdbInventoryAssets(account, "tencentcloud", regions, assetTypes)
	}
	stats := models.ResAttrs{
		"regions":    regions,
		"assetTypes": assetTypes,
		"mode":       "tencentcloud_api",
	}
	if len(regions) == 0 {
		return cmdbCloudCollectResult{Stats: stats, Err: fmt.Errorf("tencentcloud collector requires at least one region")}
	}

	selected := selectedCmdbAssetTypes(assetTypes)
	specs := []tencentCollectorSpec{
		{models.CmdbAssetTypeComputeInstance, "cvm instances", "cvm", "2017-03-12", "DescribeInstances", []string{"InstanceSet"}, "tencentcloud_instance", tencentInstanceAsset},
		{models.CmdbAssetTypeNetworkVpc, "vpcs", "vpc", "2017-03-12", "DescribeVpcs", []string{"VpcSet"}, "tencentcloud_vpc", tencentVpcAsset},
		{models.CmdbAssetTypeNetworkSubnet, "subnets", "vpc", "2017-03-12", "DescribeSubnets", []string{"SubnetSet"}, "tencentcloud_subnet", tencentSubnetAsset},
		{models.CmdbAssetTypeNetworkSecurityGroup, "security groups", "vpc", "2017-03-12", "DescribeSecurityGroups", []string{"SecurityGroupSet"}, "tencentcloud_security_group", tencentSecurityGroupAsset},
		{models.CmdbAssetTypePublicIP, "eips", "vpc", "2017-03-12", "DescribeAddresses", []string{"AddressSet"}, "tencentcloud_eip", tencentEipAsset},
		{models.CmdbAssetTypeBlockVolume, "cbs disks", "cbs", "2017-03-12", "DescribeDisks", []string{"DiskSet"}, "tencentcloud_cbs_storage", tencentDiskAsset},
		{models.CmdbAssetTypeLoadBalancer, "clb instances", "clb", "2018-03-17", "DescribeLoadBalancers", []string{"LoadBalancerSet"}, "tencentcloud_clb_instance", tencentLoadBalancerAsset},
		{models.CmdbAssetTypeKubernetesCluster, "tke clusters", "tke", "2018-05-25", "DescribeClusters", []string{"Clusters", "ClusterSet"}, "tencentcloud_kubernetes_cluster", tencentClusterAsset},
		{models.CmdbAssetTypeRelationalDatabase, "cdb instances", "cdb", "2017-03-20", "DescribeDBInstances", []string{"Items", "InstanceSet"}, "tencentcloud_mysql_instance", tencentDBAsset},
		{models.CmdbAssetTypeRedisCache, "redis instances", "redis", "2018-04-12", "DescribeInstances", []string{"InstanceSet", "Instances"}, "tencentcloud_redis_instance", tencentRedisAsset},
	}

	assets := make([]*models.CmdbAsset, 0)
	errors := make([]string, 0)
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
			items, err := collectTencentPaged(ctx, account, region, spec)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("tencentcloud %s %s: %v", region, spec.Label, err))
			}
		}
		cancel()
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeObjectStorageBucket) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		items, err := collectTencentCosBuckets(ctx, account, selectedCloudRegions(regions))
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("tencentcloud cos buckets: %v", err))
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

func hasTencentLiveCredentials(credentials map[string]string) bool {
	return firstNonEmpty(credentials["TENCENTCLOUD_SECRET_ID"], credentials["TENCENT_SECRET_ID"]) != "" &&
		firstNonEmpty(credentials["TENCENTCLOUD_SECRET_KEY"], credentials["TENCENT_SECRET_KEY"]) != ""
}

func collectTencentPaged(ctx context.Context, account *cmdbCloudAccount, region string, spec tencentCollectorSpec) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	var firstErr error
	for page := 0; page < tencentMaxCollectionPages; page++ {
		payload := models.ResAttrs{
			"Offset": page * tencentDefaultPageSize,
			"Limit":  tencentDefaultPageSize,
		}
		resp, err := tencentJSONAPI(ctx, account, spec.Service, spec.Version, spec.Action, region, payload)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			break
		}
		items := tencentResponseList(resp, spec.ListKeys...)
		for _, item := range items {
			asset := spec.Normalize(account, region, item, now)
			if asset == nil {
				continue
			}
			asset.NativeType = spec.NativeType
			asset.RawData["mode"] = "tencentcloud_api"
			assets = append(assets, asset)
		}
		if len(items) < tencentDefaultPageSize {
			break
		}
	}
	return assets, firstErr
}

func tencentJSONAPI(ctx context.Context, account *cmdbCloudAccount, service, version, action, region string, payload models.ResAttrs) (models.ResAttrs, error) {
	secretId := firstNonEmpty(account.Credentials["TENCENTCLOUD_SECRET_ID"], account.Credentials["TENCENT_SECRET_ID"])
	secretKey := firstNonEmpty(account.Credentials["TENCENTCLOUD_SECRET_KEY"], account.Credentials["TENCENT_SECRET_KEY"])
	securityToken := firstNonEmpty(account.Credentials["TENCENTCLOUD_TOKEN"], account.Credentials["TENCENT_TOKEN"])
	if secretId == "" || secretKey == "" {
		return nil, fmt.Errorf("tencentcloud collector requires TENCENTCLOUD_SECRET_ID and TENCENTCLOUD_SECRET_KEY")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	host := service + ".tencentcloudapi.com"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+host, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	timestamp := time.Now().Unix()
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", version)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-TC-Region", region)
	if securityToken != "" {
		req.Header.Set("X-TC-Token", securityToken)
	}
	req.Header.Set("Authorization", tencentAuthorization(secretId, secretKey, service, date, timestamp, host, string(body)))

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tencentcloud api %s %s status %d: %s", service, action, resp.StatusCode, string(respBody))
	}
	wrapper := struct {
		Response models.ResAttrs `json:"Response"`
	}{}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return nil, fmt.Errorf("decode tencentcloud response: %w", err)
	}
	if apiErr := modelResAttrs(wrapper.Response["Error"]); apiErr != nil {
		return nil, fmt.Errorf("%s: %s", attrString(apiErr, "Code"), attrString(apiErr, "Message"))
	}
	return wrapper.Response, nil
}

func tencentAuthorization(secretId, secretKey, service, date string, timestamp int64, host, payload string) string {
	hashedPayload := sha256Hex([]byte(payload))
	canonicalHeaders := "content-type:application/json; charset=utf-8\nhost:" + host + "\n"
	canonicalRequest := strings.Join([]string{
		http.MethodPost,
		"/",
		"",
		canonicalHeaders,
		"content-type;host",
		hashedPayload,
	}, "\n")
	credentialScope := strings.Join([]string{date, service, "tc3_request"}, "/")
	stringToSign := strings.Join([]string{
		"TC3-HMAC-SHA256",
		strconv.FormatInt(timestamp, 10),
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	secretDate := hmacSHA256([]byte("TC3"+secretKey), date)
	secretService := hmacSHA256(secretDate, service)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))
	return fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=content-type;host, Signature=%s", secretId, credentialScope, signature)
}

type tencentCosBucket struct {
	Name         string `xml:"Name"`
	Location     string `xml:"Location"`
	CreationDate string `xml:"CreationDate"`
	BucketType   string `xml:"BucketType"`
}

type tencentCosListBucketsResult struct {
	Buckets struct {
		Bucket []tencentCosBucket `xml:"Bucket"`
	} `xml:"Buckets"`
}

func collectTencentCosBuckets(ctx context.Context, account *cmdbCloudAccount, selectedRegions map[string]bool) ([]*models.CmdbAsset, error) {
	secretId := firstNonEmpty(account.Credentials["TENCENTCLOUD_SECRET_ID"], account.Credentials["TENCENT_SECRET_ID"])
	secretKey := firstNonEmpty(account.Credentials["TENCENTCLOUD_SECRET_KEY"], account.Credentials["TENCENT_SECRET_KEY"])
	if secretId == "" || secretKey == "" {
		return nil, fmt.Errorf("tencentcloud COS collector requires TENCENTCLOUD_SECRET_ID and TENCENTCLOUD_SECRET_KEY")
	}
	endpoint := strings.TrimSpace(account.Credentials["TENCENTCLOUD_COS_ENDPOINT"])
	if endpoint == "" {
		endpoint = "https://service.cos.myqcloud.com/"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if req.URL.Path == "" {
		req.URL.Path = "/"
	}
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("Host", req.URL.Host)
	if token := firstNonEmpty(account.Credentials["TENCENTCLOUD_TOKEN"], account.Credentials["TENCENT_TOKEN"]); token != "" {
		req.Header.Set("x-cos-security-token", token)
	}
	req.Header.Set("Authorization", tencentCosAuthorization(secretId, secretKey, req.Method, req.URL.Path, req.URL.Host))

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
		return nil, fmt.Errorf("tencentcloud COS list buckets status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	out := tencentCosListBucketsResult{}
	if err := xml.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode tencentcloud COS buckets: %w", err)
	}
	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0, len(out.Buckets.Bucket))
	for _, bucket := range out.Buckets.Bucket {
		if bucket.Name == "" {
			continue
		}
		region := strings.ToLower(bucket.Location)
		if len(selectedRegions) > 0 && !selectedRegions[region] {
			continue
		}
		asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeObjectStorageBucket, "tencentcloud_cos_bucket", bucket.Name, bucket.Name, now)
		asset.Status = firstNonEmpty(bucket.BucketType, "available")
		asset.Attributes = models.ResAttrs{
			"location":     bucket.Location,
			"creationDate": bucket.CreationDate,
			"bucketType":   bucket.BucketType,
			"endpoint":     endpoint,
		}
		asset.RawData["mode"] = "tencentcloud_cos_api"
		asset.RawData["response"] = bucket
		assets = append(assets, asset)
	}
	return assets, nil
}

func tencentCosAuthorization(secretId, secretKey, method, path, host string) string {
	now := time.Now().Unix()
	signTime := fmt.Sprintf("%d;%d", now, now+600)
	httpString := strings.Join([]string{
		strings.ToLower(method),
		firstNonEmpty(path, "/"),
		"",
		"host=" + strings.ToLower(host),
		"",
	}, "\n")
	stringToSign := strings.Join([]string{
		"sha1",
		signTime,
		sha1Hex([]byte(httpString)),
		"",
	}, "\n")
	signKey := hmacSHA1([]byte(secretKey), signTime)
	signature := hex.EncodeToString(hmacSHA1(signKey, stringToSign))
	return fmt.Sprintf("q-sign-algorithm=sha1&q-ak=%s&q-sign-time=%s&q-key-time=%s&q-header-list=host&q-url-param-list=&q-signature=%s",
		url.QueryEscape(secretId), signTime, signTime, signature)
}

func hmacSHA1(key []byte, value string) []byte {
	h := hmac.New(sha1.New, key)
	_, _ = h.Write([]byte(value))
	return h.Sum(nil)
}

func sha1Hex(value []byte) string {
	sum := sha1.Sum(value)
	return hex.EncodeToString(sum[:])
}

func tencentResponseList(resp models.ResAttrs, keys ...string) []models.ResAttrs {
	for _, key := range keys {
		if items := resAttrsSlice(resp[key]); len(items) > 0 {
			return items
		}
	}
	return nil
}

func tencentInstanceAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "InstanceId")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeComputeInstance, "tencentcloud_instance", id, firstNonEmpty(attrString(item, "InstanceName"), id), now)
	placement := modelResAttrs(item["Placement"])
	asset.Zone = attrString(placement, "Zone")
	asset.Status = attrString(item, "InstanceState")
	asset.PrivateIp = strings.Join(stringSlice(item["PrivateIpAddresses"]), ",")
	asset.PublicIp = strings.Join(stringSlice(item["PublicIpAddresses"]), ",")
	asset.Attributes = models.ResAttrs{
		"instanceType": attrString(item, "InstanceType"),
		"cpu":          item["CPU"],
		"memory":       item["Memory"],
		"vpcId":        attrString(modelResAttrs(item["VirtualPrivateCloud"]), "VpcId"),
	}
	asset.RawData["response"] = item
	return asset
}

func tencentVpcAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "VpcId")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkVpc, "tencentcloud_vpc", id, firstNonEmpty(attrString(item, "VpcName"), id), now)
	asset.Status = firstNonEmpty(attrString(item, "State"), "available")
	asset.Attributes = models.ResAttrs{"cidrBlock": attrString(item, "CidrBlock"), "isDefault": item["IsDefault"]}
	asset.RawData["response"] = item
	return asset
}

func tencentSubnetAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "SubnetId")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkSubnet, "tencentcloud_subnet", id, firstNonEmpty(attrString(item, "SubnetName"), id), now)
	asset.Zone = attrString(item, "Zone")
	asset.Status = firstNonEmpty(attrString(item, "State"), "available")
	asset.Attributes = models.ResAttrs{"vpcId": attrString(item, "VpcId"), "cidrBlock": attrString(item, "CidrBlock")}
	asset.RawData["response"] = item
	return asset
}

func tencentSecurityGroupAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "SecurityGroupId")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkSecurityGroup, "tencentcloud_security_group", id, firstNonEmpty(attrString(item, "SecurityGroupName"), id), now)
	asset.Status = "available"
	asset.Attributes = models.ResAttrs{"description": attrString(item, "SecurityGroupDesc"), "projectId": item["ProjectId"]}
	asset.RawData["response"] = item
	return asset
}

func tencentEipAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "AddressId")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypePublicIP, "tencentcloud_eip", id, firstNonEmpty(attrString(item, "AddressName"), attrString(item, "AddressIp"), id), now)
	asset.Status = attrString(item, "AddressStatus")
	asset.PublicIp = attrString(item, "AddressIp")
	asset.Attributes = models.ResAttrs{"instanceId": attrString(item, "InstanceId"), "networkInterfaceId": attrString(item, "NetworkInterfaceId")}
	asset.RawData["response"] = item
	return asset
}

func tencentDiskAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "DiskId")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeBlockVolume, "tencentcloud_cbs_storage", id, firstNonEmpty(attrString(item, "DiskName"), id), now)
	placement := modelResAttrs(item["Placement"])
	asset.Zone = firstNonEmpty(attrString(placement, "Zone"), attrString(item, "Zone"))
	asset.Status = attrString(item, "DiskState")
	asset.Attributes = models.ResAttrs{"diskSizeGiB": item["DiskSize"], "diskType": attrString(item, "DiskType"), "diskUsage": attrString(item, "DiskUsage")}
	asset.RawData["response"] = item
	return asset
}

func tencentLoadBalancerAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "LoadBalancerId")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeLoadBalancer, "tencentcloud_clb_instance", id, firstNonEmpty(attrString(item, "LoadBalancerName"), id), now)
	asset.Status = attrString(item, "Status")
	asset.Address = strings.Join(stringSlice(item["LoadBalancerVips"]), ",")
	asset.PublicIp = asset.Address
	asset.Attributes = models.ResAttrs{"vpcId": attrString(item, "VpcId"), "type": attrString(item, "LoadBalancerType")}
	asset.RawData["response"] = item
	return asset
}

func tencentClusterAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "ClusterId")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeKubernetesCluster, "tencentcloud_kubernetes_cluster", id, firstNonEmpty(attrString(item, "ClusterName"), id), now)
	asset.Status = attrString(item, "ClusterStatus")
	asset.Attributes = models.ResAttrs{"clusterVersion": attrString(item, "ClusterVersion"), "vpcId": attrString(item, "VpcId")}
	asset.RawData["response"] = item
	return asset
}

func tencentDBAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "InstanceId")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeRelationalDatabase, "tencentcloud_mysql_instance", id, firstNonEmpty(attrString(item, "InstanceName"), id), now)
	asset.Status = attrString(item, "Status")
	asset.PrivateIp = attrString(item, "Vip")
	asset.Attributes = models.ResAttrs{"engineVersion": attrString(item, "EngineVersion"), "port": item["Vport"], "memory": item["Memory"], "volume": item["Volume"]}
	asset.RawData["response"] = item
	return asset
}

func tencentRedisAsset(account *cmdbCloudAccount, region string, item models.ResAttrs, now models.Time) *models.CmdbAsset {
	id := attrString(item, "InstanceId")
	asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeRedisCache, "tencentcloud_redis_instance", id, firstNonEmpty(attrString(item, "InstanceName"), id), now)
	asset.Status = attrString(item, "Status")
	asset.PrivateIp = firstNonEmpty(attrString(item, "Vip"), attrString(item, "WanIp"))
	asset.Attributes = models.ResAttrs{"type": item["Type"], "memSize": item["MemSize"], "port": item["Port"]}
	asset.RawData["response"] = item
	return asset
}

func resAttrsSlice(value interface{}) []models.ResAttrs {
	items := make([]models.ResAttrs, 0)
	switch typed := value.(type) {
	case []models.ResAttrs:
		return typed
	case []map[string]interface{}:
		for _, item := range typed {
			items = append(items, models.ResAttrs(item))
		}
	case []interface{}:
		for _, item := range typed {
			if attrs := modelResAttrs(item); attrs != nil {
				items = append(items, attrs)
			}
		}
	}
	return items
}

func stringSlice(value interface{}) []string {
	out := make([]string, 0)
	switch typed := value.(type) {
	case []string:
		return typed
	case []interface{}:
		for _, item := range typed {
			if item != nil {
				out = append(out, fmt.Sprintf("%v", item))
			}
		}
	case string:
		if typed != "" {
			out = append(out, typed)
		}
	}
	return out
}
