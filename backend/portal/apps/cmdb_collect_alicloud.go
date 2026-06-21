// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"context"
	"encoding/json"
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
	alicloudECSAPIVersion      = "2014-05-26"
	alicloudVPCAPIVersion      = "2016-04-28"
	alicloudSLBAPIVersion      = "2014-05-15"
	alicloudRDSAPIVersion      = "2014-08-15"
	alicloudRedisAPIVersion    = "2015-01-01"
	alicloudACKAPIVersion      = "2015-12-15"
	alicloudDefaultPageSize    = 50
	alicloudMaxCollectionPages = 100
)

func collectCmdbAlicloudAssets(account *cmdbCloudAccount, regions, assetTypes []string) cmdbCloudCollectResult {
	stats := models.ResAttrs{
		"regions":    regions,
		"assetTypes": assetTypes,
	}
	if len(regions) == 0 {
		return cmdbCloudCollectResult{
			Stats: stats,
			Err:   fmt.Errorf("alicloud collector requires at least one region"),
		}
	}

	selected := selectedCmdbAssetTypes(assetTypes)
	assets := make([]*models.CmdbAsset, 0)
	errors := make([]string, 0)
	for _, region := range regions {
		region = strings.TrimSpace(region)
		if region == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		regionAssets, regionErrors := collectCmdbAlicloudRegionAssets(ctx, account, region, selected)
		cancel()
		assets = append(assets, regionAssets...)
		errors = append(errors, regionErrors...)
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

func collectCmdbAlicloudRegionAssets(ctx context.Context, account *cmdbCloudAccount, region string, selected map[string]bool) ([]*models.CmdbAsset, []string) {
	assets := make([]*models.CmdbAsset, 0)
	errors := make([]string, 0)

	if wantsCmdbAssetType(selected, models.CmdbAssetTypeComputeInstance) {
		items, err := collectAlicloudInstances(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("alicloud %s ecs instances: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkVpc) {
		items, err := collectAlicloudVpcs(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("alicloud %s vpcs: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkSubnet) {
		items, err := collectAlicloudVSwitches(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("alicloud %s vswitches: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkSecurityGroup) {
		items, err := collectAlicloudSecurityGroups(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("alicloud %s security groups: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypePublicIP) {
		items, err := collectAlicloudEips(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("alicloud %s eips: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeBlockVolume) {
		items, err := collectAlicloudDisks(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("alicloud %s disks: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeLoadBalancer) {
		items, err := collectAlicloudLoadBalancers(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("alicloud %s load balancers: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeRelationalDatabase) {
		items, err := collectAlicloudRdsInstances(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("alicloud %s rds instances: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeRedisCache) {
		items, err := collectAlicloudRedisInstances(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("alicloud %s redis instances: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeObjectStorageBucket) {
		items, err := collectAlicloudOssBuckets(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("alicloud %s oss buckets: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeKubernetesCluster) {
		items, err := collectAlicloudAckClusters(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("alicloud %s ack clusters: %v", region, err))
		}
	}

	return assets, errors
}

type alicloudTag struct {
	Key      string `json:"Key"`
	Value    string `json:"Value"`
	TagKey   string `json:"TagKey"`
	TagValue string `json:"TagValue"`
}

type alicloudTags struct {
	Tag []alicloudTag `json:"Tag"`
}

type alicloudStringList struct {
	Items     []string `json:"Item"`
	IpAddress []string `json:"IpAddress"`
}

type alicloudDescribeInstancesResponse struct {
	TotalCount int `json:"TotalCount"`
	PageNumber int `json:"PageNumber"`
	PageSize   int `json:"PageSize"`
	Instances  struct {
		Instance []alicloudInstance `json:"Instance"`
	} `json:"Instances"`
}

type alicloudInstance struct {
	InstanceId      string             `json:"InstanceId"`
	InstanceName    string             `json:"InstanceName"`
	HostName        string             `json:"HostName"`
	Status          string             `json:"Status"`
	RegionId        string             `json:"RegionId"`
	ZoneId          string             `json:"ZoneId"`
	InstanceType    string             `json:"InstanceType"`
	ImageId         string             `json:"ImageId"`
	OSName          string             `json:"OSName"`
	OSType          string             `json:"OSType"`
	CreationTime    string             `json:"CreationTime"`
	ExpiredTime     string             `json:"ExpiredTime"`
	PublicIpAddress alicloudStringList `json:"PublicIpAddress"`
	VpcAttributes   struct {
		VpcId            string             `json:"VpcId"`
		VSwitchId        string             `json:"VSwitchId"`
		PrivateIpAddress alicloudStringList `json:"PrivateIpAddress"`
		NatIpAddress     string             `json:"NatIpAddress"`
	} `json:"VpcAttributes"`
	EipAddress struct {
		AllocationId string `json:"AllocationId"`
		IpAddress    string `json:"IpAddress"`
	} `json:"EipAddress"`
	Tags alicloudTags `json:"Tags"`
}

func collectAlicloudInstances(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	err := alicloudPagedRPC(ctx, account, "ecs", region, alicloudECSAPIVersion, "DescribeInstances", nil, func(values url.Values) (int, int, int, error) {
		resp := alicloudDescribeInstancesResponse{}
		if err := alicloudRPCServiceAPI(ctx, account, "ecs", region, values, &resp); err != nil {
			return 0, 0, 0, err
		}
		for _, instance := range resp.Instances.Instance {
			name := firstNonEmpty(instance.InstanceName, instance.HostName, instance.InstanceId)
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeComputeInstance, "alicloud_instance", instance.InstanceId, name, now)
			asset.Zone = instance.ZoneId
			asset.Status = instance.Status
			asset.PublicIp = firstNonEmpty(firstString(instance.PublicIpAddress.IpAddress), instance.EipAddress.IpAddress)
			asset.PrivateIp = firstString(instance.VpcAttributes.PrivateIpAddress.IpAddress)
			asset.Tags = alicloudTagsToAttrs(instance.Tags.Tag)
			asset.Attributes = models.ResAttrs{
				"instanceType": instance.InstanceType,
				"imageId":      instance.ImageId,
				"osName":       instance.OSName,
				"osType":       instance.OSType,
				"vpcId":        instance.VpcAttributes.VpcId,
				"vSwitchId":    instance.VpcAttributes.VSwitchId,
				"natIpAddress": instance.VpcAttributes.NatIpAddress,
				"eip": models.ResAttrs{
					"allocationId": instance.EipAddress.AllocationId,
					"ipAddress":    instance.EipAddress.IpAddress,
				},
				"creationTime": instance.CreationTime,
				"expiredTime":  instance.ExpiredTime,
			}
			assets = append(assets, asset)
		}
		return resp.TotalCount, resp.PageNumber, resp.PageSize, nil
	})
	return assets, err
}

type alicloudDescribeDisksResponse struct {
	TotalCount int `json:"TotalCount"`
	PageNumber int `json:"PageNumber"`
	PageSize   int `json:"PageSize"`
	Disks      struct {
		Disk []alicloudDisk `json:"Disk"`
	} `json:"Disks"`
}

type alicloudDisk struct {
	DiskId       string       `json:"DiskId"`
	DiskName     string       `json:"DiskName"`
	Status       string       `json:"Status"`
	RegionId     string       `json:"RegionId"`
	ZoneId       string       `json:"ZoneId"`
	Size         int          `json:"Size"`
	Type         string       `json:"Type"`
	Category     string       `json:"Category"`
	InstanceId   string       `json:"InstanceId"`
	CreationTime string       `json:"CreationTime"`
	ExpiredTime  string       `json:"ExpiredTime"`
	Tags         alicloudTags `json:"Tags"`
}

func collectAlicloudDisks(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	err := alicloudPagedRPC(ctx, account, "ecs", region, alicloudECSAPIVersion, "DescribeDisks", nil, func(values url.Values) (int, int, int, error) {
		resp := alicloudDescribeDisksResponse{}
		if err := alicloudRPCServiceAPI(ctx, account, "ecs", region, values, &resp); err != nil {
			return 0, 0, 0, err
		}
		for _, disk := range resp.Disks.Disk {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeBlockVolume, "alicloud_disk", disk.DiskId, firstNonEmpty(disk.DiskName, disk.DiskId), now)
			asset.Zone = disk.ZoneId
			asset.Status = disk.Status
			asset.Tags = alicloudTagsToAttrs(disk.Tags.Tag)
			asset.Attributes = models.ResAttrs{
				"sizeGiB":      disk.Size,
				"type":         disk.Type,
				"category":     disk.Category,
				"instanceId":   disk.InstanceId,
				"creationTime": disk.CreationTime,
				"expiredTime":  disk.ExpiredTime,
			}
			assets = append(assets, asset)
		}
		return resp.TotalCount, resp.PageNumber, resp.PageSize, nil
	})
	return assets, err
}

type alicloudDescribeVpcsResponse struct {
	TotalCount int `json:"TotalCount"`
	PageNumber int `json:"PageNumber"`
	PageSize   int `json:"PageSize"`
	Vpcs       struct {
		Vpc []alicloudVpc `json:"Vpc"`
	} `json:"Vpcs"`
}

type alicloudVpc struct {
	VpcId         string       `json:"VpcId"`
	VpcName       string       `json:"VpcName"`
	Status        string       `json:"Status"`
	RegionId      string       `json:"RegionId"`
	CidrBlock     string       `json:"CidrBlock"`
	Ipv6CidrBlock string       `json:"Ipv6CidrBlock"`
	Description   string       `json:"Description"`
	Tags          alicloudTags `json:"Tags"`
}

func collectAlicloudVpcs(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	err := alicloudPagedRPC(ctx, account, "vpc", region, alicloudVPCAPIVersion, "DescribeVpcs", nil, func(values url.Values) (int, int, int, error) {
		resp := alicloudDescribeVpcsResponse{}
		if err := alicloudRPCServiceAPI(ctx, account, "vpc", region, values, &resp); err != nil {
			return 0, 0, 0, err
		}
		for _, vpc := range resp.Vpcs.Vpc {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkVpc, "alicloud_vpc", vpc.VpcId, firstNonEmpty(vpc.VpcName, vpc.VpcId), now)
			asset.Status = vpc.Status
			asset.Tags = alicloudTagsToAttrs(vpc.Tags.Tag)
			asset.Attributes = models.ResAttrs{
				"cidrBlock":     vpc.CidrBlock,
				"ipv6CidrBlock": vpc.Ipv6CidrBlock,
				"description":   vpc.Description,
			}
			assets = append(assets, asset)
		}
		return resp.TotalCount, resp.PageNumber, resp.PageSize, nil
	})
	return assets, err
}

type alicloudDescribeVSwitchesResponse struct {
	TotalCount int `json:"TotalCount"`
	PageNumber int `json:"PageNumber"`
	PageSize   int `json:"PageSize"`
	VSwitches  struct {
		VSwitch []alicloudVSwitch `json:"VSwitch"`
	} `json:"VSwitches"`
}

type alicloudVSwitch struct {
	VSwitchId               string `json:"VSwitchId"`
	VSwitchName             string `json:"VSwitchName"`
	VpcId                   string `json:"VpcId"`
	ZoneId                  string `json:"ZoneId"`
	CidrBlock               string `json:"CidrBlock"`
	Status                  string `json:"Status"`
	AvailableIpAddressCount int    `json:"AvailableIpAddressCount"`
	Description             string `json:"Description"`
}

func collectAlicloudVSwitches(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	err := alicloudPagedRPC(ctx, account, "vpc", region, alicloudVPCAPIVersion, "DescribeVSwitches", nil, func(values url.Values) (int, int, int, error) {
		resp := alicloudDescribeVSwitchesResponse{}
		if err := alicloudRPCServiceAPI(ctx, account, "vpc", region, values, &resp); err != nil {
			return 0, 0, 0, err
		}
		for _, vsw := range resp.VSwitches.VSwitch {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkSubnet, "alicloud_vswitch", vsw.VSwitchId, firstNonEmpty(vsw.VSwitchName, vsw.VSwitchId), now)
			asset.Zone = vsw.ZoneId
			asset.Status = vsw.Status
			asset.Attributes = models.ResAttrs{
				"vpcId":                   vsw.VpcId,
				"cidrBlock":               vsw.CidrBlock,
				"availableIpAddressCount": vsw.AvailableIpAddressCount,
				"description":             vsw.Description,
			}
			assets = append(assets, asset)
		}
		return resp.TotalCount, resp.PageNumber, resp.PageSize, nil
	})
	return assets, err
}

type alicloudDescribeSecurityGroupsResponse struct {
	TotalCount     int `json:"TotalCount"`
	PageNumber     int `json:"PageNumber"`
	PageSize       int `json:"PageSize"`
	SecurityGroups struct {
		SecurityGroup []alicloudSecurityGroup `json:"SecurityGroup"`
	} `json:"SecurityGroups"`
}

type alicloudSecurityGroup struct {
	SecurityGroupId   string       `json:"SecurityGroupId"`
	SecurityGroupName string       `json:"SecurityGroupName"`
	Description       string       `json:"Description"`
	VpcId             string       `json:"VpcId"`
	SecurityGroupType string       `json:"SecurityGroupType"`
	CreationTime      string       `json:"CreationTime"`
	Tags              alicloudTags `json:"Tags"`
}

func collectAlicloudSecurityGroups(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	err := alicloudPagedRPC(ctx, account, "ecs", region, alicloudECSAPIVersion, "DescribeSecurityGroups", nil, func(values url.Values) (int, int, int, error) {
		resp := alicloudDescribeSecurityGroupsResponse{}
		if err := alicloudRPCServiceAPI(ctx, account, "ecs", region, values, &resp); err != nil {
			return 0, 0, 0, err
		}
		for _, group := range resp.SecurityGroups.SecurityGroup {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkSecurityGroup, "alicloud_security_group", group.SecurityGroupId, firstNonEmpty(group.SecurityGroupName, group.SecurityGroupId), now)
			asset.Tags = alicloudTagsToAttrs(group.Tags.Tag)
			asset.Attributes = models.ResAttrs{
				"vpcId":             group.VpcId,
				"description":       group.Description,
				"securityGroupType": group.SecurityGroupType,
				"creationTime":      group.CreationTime,
			}
			if permissions, err := alicloudSecurityGroupPermissions(ctx, account, region, group.SecurityGroupId); err == nil {
				asset.Attributes["permissions"] = permissions
			}
			assets = append(assets, asset)
		}
		return resp.TotalCount, resp.PageNumber, resp.PageSize, nil
	})
	return assets, err
}

func alicloudSecurityGroupPermissions(ctx context.Context, account *cmdbCloudAccount, region string, groupId string) (interface{}, error) {
	resp := struct {
		Permissions struct {
			Permission []models.ResAttrs `json:"Permission"`
		} `json:"Permissions"`
	}{}
	err := alicloudRPCServiceAPI(ctx, account, "ecs", region, url.Values{
		"Action":          []string{"DescribeSecurityGroupAttribute"},
		"Version":         []string{alicloudECSAPIVersion},
		"SecurityGroupId": []string{groupId},
	}, &resp)
	return resp.Permissions.Permission, err
}

type alicloudDescribeEipAddressesResponse struct {
	TotalCount   int `json:"TotalCount"`
	PageNumber   int `json:"PageNumber"`
	PageSize     int `json:"PageSize"`
	EipAddresses struct {
		EipAddress []alicloudEipAddress `json:"EipAddress"`
	} `json:"EipAddresses"`
}

type alicloudEipAddress struct {
	AllocationId       string `json:"AllocationId"`
	IpAddress          string `json:"IpAddress"`
	Status             string `json:"Status"`
	RegionId           string `json:"RegionId"`
	Bandwidth          string `json:"Bandwidth"`
	InstanceId         string `json:"InstanceId"`
	InstanceType       string `json:"InstanceType"`
	InternetChargeType string `json:"InternetChargeType"`
	AllocationTime     string `json:"AllocationTime"`
}

func collectAlicloudEips(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	err := alicloudPagedRPC(ctx, account, "vpc", region, alicloudVPCAPIVersion, "DescribeEipAddresses", nil, func(values url.Values) (int, int, int, error) {
		resp := alicloudDescribeEipAddressesResponse{}
		if err := alicloudRPCServiceAPI(ctx, account, "vpc", region, values, &resp); err != nil {
			return 0, 0, 0, err
		}
		for _, eip := range resp.EipAddresses.EipAddress {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypePublicIP, "alicloud_eip_address", eip.AllocationId, firstNonEmpty(eip.IpAddress, eip.AllocationId), now)
			asset.Status = eip.Status
			asset.Address = eip.IpAddress
			asset.PublicIp = eip.IpAddress
			asset.Attributes = models.ResAttrs{
				"bandwidth":          eip.Bandwidth,
				"instanceId":         eip.InstanceId,
				"instanceType":       eip.InstanceType,
				"internetChargeType": eip.InternetChargeType,
				"allocationTime":     eip.AllocationTime,
			}
			assets = append(assets, asset)
		}
		return resp.TotalCount, resp.PageNumber, resp.PageSize, nil
	})
	return assets, err
}

type alicloudDescribeLoadBalancersResponse struct {
	TotalCount    int `json:"TotalCount"`
	PageNumber    int `json:"PageNumber"`
	PageSize      int `json:"PageSize"`
	LoadBalancers struct {
		LoadBalancer []alicloudLoadBalancer `json:"LoadBalancer"`
	} `json:"LoadBalancers"`
}

type alicloudLoadBalancer struct {
	LoadBalancerId     string `json:"LoadBalancerId"`
	LoadBalancerName   string `json:"LoadBalancerName"`
	LoadBalancerStatus string `json:"LoadBalancerStatus"`
	LoadBalancerSpec   string `json:"LoadBalancerSpec"`
	Address            string `json:"Address"`
	AddressType        string `json:"AddressType"`
	NetworkType        string `json:"NetworkType"`
	VpcId              string `json:"VpcId"`
	VSwitchId          string `json:"VSwitchId"`
	RegionId           string `json:"RegionId"`
	MasterZoneId       string `json:"MasterZoneId"`
	CreateTime         string `json:"CreateTime"`
}

func collectAlicloudLoadBalancers(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	err := alicloudPagedRPC(ctx, account, "slb", region, alicloudSLBAPIVersion, "DescribeLoadBalancers", nil, func(values url.Values) (int, int, int, error) {
		resp := alicloudDescribeLoadBalancersResponse{}
		if err := alicloudRPCServiceAPI(ctx, account, "slb", region, values, &resp); err != nil {
			return 0, 0, 0, err
		}
		for _, lb := range resp.LoadBalancers.LoadBalancer {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeLoadBalancer, "alicloud_slb_load_balancer", lb.LoadBalancerId, firstNonEmpty(lb.LoadBalancerName, lb.LoadBalancerId), now)
			asset.Zone = lb.MasterZoneId
			asset.Status = lb.LoadBalancerStatus
			asset.Address = lb.Address
			if strings.EqualFold(lb.AddressType, "internet") {
				asset.PublicIp = lb.Address
			} else {
				asset.PrivateIp = lb.Address
			}
			asset.Attributes = models.ResAttrs{
				"spec":        lb.LoadBalancerSpec,
				"addressType": lb.AddressType,
				"networkType": lb.NetworkType,
				"vpcId":       lb.VpcId,
				"vSwitchId":   lb.VSwitchId,
				"createTime":  lb.CreateTime,
			}
			assets = append(assets, asset)
		}
		return resp.TotalCount, resp.PageNumber, resp.PageSize, nil
	})
	return assets, err
}

type alicloudDescribeDBInstancesResponse struct {
	TotalRecordCount int `json:"TotalRecordCount"`
	PageNumber       int `json:"PageNumber"`
	PageRecordCount  int `json:"PageRecordCount"`
	Items            struct {
		DBInstance []alicloudDBInstance `json:"DBInstance"`
	} `json:"Items"`
}

type alicloudDBInstance struct {
	DBInstanceId          string `json:"DBInstanceId"`
	DBInstanceDescription string `json:"DBInstanceDescription"`
	DBInstanceStatus      string `json:"DBInstanceStatus"`
	DBInstanceClass       string `json:"DBInstanceClass"`
	DBInstanceStorage     int    `json:"DBInstanceStorage"`
	Engine                string `json:"Engine"`
	EngineVersion         string `json:"EngineVersion"`
	RegionId              string `json:"RegionId"`
	ZoneId                string `json:"ZoneId"`
	VpcId                 string `json:"VpcId"`
	VSwitchId             string `json:"VSwitchId"`
	CreationTime          string `json:"CreationTime"`
	ExpireTime            string `json:"ExpireTime"`
	ConnectionString      string `json:"ConnectionString"`
}

func collectAlicloudRdsInstances(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	err := alicloudPagedRPC(ctx, account, "rds", region, alicloudRDSAPIVersion, "DescribeDBInstances", nil, func(values url.Values) (int, int, int, error) {
		resp := alicloudDescribeDBInstancesResponse{}
		if err := alicloudRPCServiceAPI(ctx, account, "rds", region, values, &resp); err != nil {
			return 0, 0, 0, err
		}
		for _, dbi := range resp.Items.DBInstance {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeRelationalDatabase, "alicloud_db_instance", dbi.DBInstanceId, firstNonEmpty(dbi.DBInstanceDescription, dbi.DBInstanceId), now)
			asset.Zone = dbi.ZoneId
			asset.Status = dbi.DBInstanceStatus
			asset.Address = dbi.ConnectionString
			asset.Attributes = models.ResAttrs{
				"class":        dbi.DBInstanceClass,
				"storageGiB":   dbi.DBInstanceStorage,
				"engine":       dbi.Engine,
				"version":      dbi.EngineVersion,
				"vpcId":        dbi.VpcId,
				"vSwitchId":    dbi.VSwitchId,
				"creationTime": dbi.CreationTime,
				"expireTime":   dbi.ExpireTime,
			}
			assets = append(assets, asset)
		}
		return resp.TotalRecordCount, resp.PageNumber, resp.PageRecordCount, nil
	})
	return assets, err
}

type alicloudDescribeRedisInstancesResponse struct {
	TotalCount int `json:"TotalCount"`
	PageNumber int `json:"PageNumber"`
	PageSize   int `json:"PageSize"`
	Instances  struct {
		KVStoreInstance []alicloudRedisInstance `json:"KVStoreInstance"`
		Instance        []alicloudRedisInstance `json:"Instance"`
	} `json:"Instances"`
}

type alicloudRedisInstance struct {
	InstanceId       string `json:"InstanceId"`
	InstanceName     string `json:"InstanceName"`
	InstanceStatus   string `json:"InstanceStatus"`
	InstanceClass    string `json:"InstanceClass"`
	ArchitectureType string `json:"ArchitectureType"`
	EngineVersion    string `json:"EngineVersion"`
	RegionId         string `json:"RegionId"`
	ZoneId           string `json:"ZoneId"`
	VpcId            string `json:"VpcId"`
	VSwitchId        string `json:"VSwitchId"`
	Capacity         int    `json:"Capacity"`
	CreationTime     string `json:"CreationTime"`
	EndTime          string `json:"EndTime"`
	ConnectionDomain string `json:"ConnectionDomain"`
}

func collectAlicloudRedisInstances(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	err := alicloudPagedRPC(ctx, account, "r-kvstore", region, alicloudRedisAPIVersion, "DescribeInstances", nil, func(values url.Values) (int, int, int, error) {
		resp := alicloudDescribeRedisInstancesResponse{}
		if err := alicloudRPCServiceAPI(ctx, account, "r-kvstore", region, values, &resp); err != nil {
			return 0, 0, 0, err
		}
		items := append(resp.Instances.KVStoreInstance, resp.Instances.Instance...)
		for _, redis := range items {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeRedisCache, "alicloud_kvstore_instance", redis.InstanceId, firstNonEmpty(redis.InstanceName, redis.InstanceId), now)
			asset.Zone = redis.ZoneId
			asset.Status = redis.InstanceStatus
			asset.Address = redis.ConnectionDomain
			asset.Attributes = models.ResAttrs{
				"class":            redis.InstanceClass,
				"architectureType": redis.ArchitectureType,
				"engineVersion":    redis.EngineVersion,
				"capacityMiB":      redis.Capacity,
				"vpcId":            redis.VpcId,
				"vSwitchId":        redis.VSwitchId,
				"creationTime":     redis.CreationTime,
				"endTime":          redis.EndTime,
			}
			assets = append(assets, asset)
		}
		return resp.TotalCount, resp.PageNumber, resp.PageSize, nil
	})
	return assets, err
}

func collectAlicloudOssBuckets(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	items, err := alicloudRPCItems(ctx, account, "oss", region, "2019-05-17", "ListBuckets", nil, "Buckets", "Bucket")
	if err != nil {
		return nil, err
	}
	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0, len(items))
	for _, item := range items {
		name := alicloudAttrString(item, "Name")
		if name == "" {
			name = alicloudAttrString(item, "name")
		}
		if name == "" {
			continue
		}
		asset := newCmdbCloudAsset(account, firstNonEmpty(alicloudAttrString(item, "Location"), region), models.CmdbAssetTypeObjectStorageBucket, "alicloud_oss_bucket", name, name, now)
		asset.Status = firstNonEmpty(alicloudAttrString(item, "StorageClass"), alicloudAttrString(item, "storageClass"))
		asset.Attributes = item
		assets = append(assets, asset)
	}
	return assets, nil
}

func collectAlicloudAckClusters(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	items, err := alicloudRPCItems(ctx, account, "cs", region, alicloudACKAPIVersion, "DescribeClustersV1", nil, "clusters", "cluster")
	if err != nil {
		return nil, err
	}
	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0, len(items))
	for _, item := range items {
		id := firstNonEmpty(alicloudAttrString(item, "cluster_id"), alicloudAttrString(item, "ClusterId"), alicloudAttrString(item, "id"))
		if id == "" {
			continue
		}
		asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeKubernetesCluster, "alicloud_cs_kubernetes", id, firstNonEmpty(alicloudAttrString(item, "name"), id), now)
		asset.Status = firstNonEmpty(alicloudAttrString(item, "state"), alicloudAttrString(item, "status"))
		asset.Attributes = item
		assets = append(assets, asset)
	}
	return assets, nil
}

func alicloudPagedRPC(ctx context.Context, account *cmdbCloudAccount, service string, region string, version string, action string, extra url.Values, fetch func(url.Values) (total int, pageNumber int, pageSize int, err error)) error {
	pageSize := alicloudDefaultPageSize
	for pageNumber := 1; pageNumber <= alicloudMaxCollectionPages; pageNumber++ {
		values := cloneURLValues(extra)
		values.Set("Action", action)
		values.Set("Version", version)
		values.Set("PageNumber", strconv.Itoa(pageNumber))
		values.Set("PageSize", strconv.Itoa(pageSize))
		total, respPageNumber, respPageSize, err := fetch(values)
		if err != nil {
			return err
		}
		if total <= 0 {
			return nil
		}
		if respPageNumber <= 0 {
			respPageNumber = pageNumber
		}
		if respPageSize <= 0 {
			respPageSize = pageSize
		}
		if respPageNumber*respPageSize >= total {
			return nil
		}
	}
	return fmt.Errorf("alicloud %s %s exceeded max collection pages %d", service, action, alicloudMaxCollectionPages)
}

func alicloudRPCItems(ctx context.Context, account *cmdbCloudAccount, service string, region string, version string, action string, extra url.Values, path ...string) ([]models.ResAttrs, error) {
	resp := models.ResAttrs{}
	values := cloneURLValues(extra)
	values.Set("Action", action)
	values.Set("Version", version)
	if err := alicloudRPCServiceAPI(ctx, account, service, region, values, &resp); err != nil {
		return nil, err
	}
	items := alicloudNestedItems(resp, path...)
	return items, nil
}

func alicloudRPCServiceAPI(c context.Context, account *cmdbCloudAccount, service, region string, values url.Values, out interface{}) error {
	values = cloneURLValues(values)
	values.Set("Format", "JSON")
	values.Set("AccessKeyId", strings.TrimSpace(account.Credentials["ALICLOUD_ACCESS_KEY"]))
	values.Set("SignatureMethod", "HMAC-SHA1")
	values.Set("SignatureVersion", "1.0")
	values.Set("SignatureNonce", fmt.Sprintf("%d", time.Now().UnixNano()))
	values.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	if values.Get("RegionId") == "" {
		values.Set("RegionId", region)
	}
	if values.Get("AccessKeyId") == "" || strings.TrimSpace(account.Credentials["ALICLOUD_SECRET_KEY"]) == "" {
		return fmt.Errorf("missing ALICLOUD_ACCESS_KEY or ALICLOUD_SECRET_KEY")
	}

	values.Set("Signature", signAlicloudRPC(values, account.Credentials["ALICLOUD_SECRET_KEY"]))
	req, err := http.NewRequestWithContext(c, http.MethodGet, alicloudEndpointURL(service, region, values), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if alicloudErr := alicloudRPCError(body); alicloudErr != nil {
		return alicloudErr
	}
	return json.Unmarshal(body, out)
}

func alicloudEndpointURL(service, region string, values url.Values) string {
	hostService := service
	switch service {
	case "ecs":
		hostService = "ecs"
	case "vpc":
		hostService = "vpc"
	case "slb":
		hostService = "slb"
	case "rds":
		hostService = "rds"
	case "r-kvstore":
		hostService = "r-kvstore"
	case "oss":
		hostService = "oss"
	case "cs":
		hostService = "cs"
	}
	u := url.URL{
		Scheme:   "https",
		Host:     fmt.Sprintf("%s.%s.aliyuncs.com", hostService, region),
		Path:     "/",
		RawQuery: values.Encode(),
	}
	return u.String()
}

func alicloudNestedItems(root models.ResAttrs, path ...string) []models.ResAttrs {
	var current interface{} = root
	for _, key := range path {
		if attrs, ok := current.(models.ResAttrs); ok {
			current = attrs[key]
			continue
		}
		if attrs, ok := current.(map[string]interface{}); ok {
			current = attrs[key]
			continue
		}
		return nil
	}
	switch typed := current.(type) {
	case []models.ResAttrs:
		return typed
	case []interface{}:
		items := make([]models.ResAttrs, 0, len(typed))
		for _, item := range typed {
			if attrs, ok := item.(map[string]interface{}); ok {
				items = append(items, models.ResAttrs(attrs))
			}
		}
		return items
	case map[string]interface{}:
		return []models.ResAttrs{models.ResAttrs(typed)}
	case models.ResAttrs:
		return []models.ResAttrs{typed}
	default:
		return nil
	}
}

func alicloudTagsToAttrs(tags []alicloudTag) models.ResAttrs {
	attrs := models.ResAttrs{}
	for _, tag := range tags {
		key := firstNonEmpty(tag.Key, tag.TagKey)
		if key == "" {
			continue
		}
		attrs[key] = firstNonEmpty(tag.Value, tag.TagValue)
	}
	return attrs
}

func alicloudAttrString(attrs models.ResAttrs, key string) string {
	if attrs == nil {
		return ""
	}
	value, ok := attrs[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%v", typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func firstString(values []string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
