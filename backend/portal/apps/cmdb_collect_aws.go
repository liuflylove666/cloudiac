// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"cloudiac/portal/models"
)

const (
	awsEC2APIVersion         = "2016-11-15"
	awsRDSAPIVersion         = "2014-10-31"
	awsElastiCacheAPIVersion = "2015-02-02"
)

type awsTag struct {
	Key   string `xml:"key" json:"key"`
	Value string `xml:"value" json:"value"`
}

func collectCmdbAwsAssets(account *cmdbCloudAccount, regions, assetTypes []string) cmdbCloudCollectResult {
	stats := models.ResAttrs{
		"regions":    regions,
		"assetTypes": assetTypes,
	}
	if len(regions) == 0 {
		return cmdbCloudCollectResult{
			Stats: stats,
			Err:   fmt.Errorf("aws collector requires at least one region"),
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
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		regionAssets, regionErrors := collectCmdbAwsRegionAssets(ctx, account, region, selected)
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

func collectCmdbAwsRegionAssets(ctx context.Context, account *cmdbCloudAccount, region string, selected map[string]bool) ([]*models.CmdbAsset, []string) {
	assets := make([]*models.CmdbAsset, 0)
	errors := make([]string, 0)

	if wantsCmdbAssetType(selected, models.CmdbAssetTypeComputeInstance) {
		items, err := collectAwsEc2Instances(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("aws %s ec2 instances: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkVpc) {
		items, err := collectAwsVpcs(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("aws %s vpcs: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkSubnet) {
		items, err := collectAwsSubnets(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("aws %s subnets: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkSecurityGroup) {
		items, err := collectAwsSecurityGroups(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("aws %s security groups: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeBlockVolume) {
		items, err := collectAwsVolumes(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("aws %s volumes: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeKubernetesCluster) {
		items, err := collectAwsEksClusters(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("aws %s eks clusters: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeRelationalDatabase) {
		items, err := collectAwsRdsInstances(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("aws %s rds instances: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeRedisCache) {
		items, err := collectAwsRedisCaches(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("aws %s elasticache redis: %v", region, err))
		}
	}

	return assets, errors
}

type awsEC2DescribeInstancesResponse struct {
	NextToken    string              `xml:"nextToken"`
	Reservations []awsEC2Reservation `xml:"reservationSet>item"`
}

type awsEC2Reservation struct {
	ReservationId string           `xml:"reservationId"`
	Instances     []awsEC2Instance `xml:"instancesSet>item"`
}

type awsEC2Instance struct {
	InstanceId       string `xml:"instanceId"`
	ImageId          string `xml:"imageId"`
	InstanceType     string `xml:"instanceType"`
	PrivateDnsName   string `xml:"privateDnsName"`
	PublicDnsName    string `xml:"dnsName"`
	PublicIpAddress  string `xml:"ipAddress"`
	PrivateIPAddress string `xml:"privateIpAddress"`
	VpcId            string `xml:"vpcId"`
	SubnetId         string `xml:"subnetId"`
	Platform         string `xml:"platform"`
	PlatformDetails  string `xml:"platformDetails"`
	LaunchTime       string `xml:"launchTime"`
	State            struct {
		Name string `xml:"name"`
	} `xml:"instanceState"`
	Placement struct {
		AvailabilityZone string `xml:"availabilityZone"`
	} `xml:"placement"`
	Tags []awsTag `xml:"tagSet>item"`
}

func collectAwsEc2Instances(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	nextToken := ""
	for {
		resp := awsEC2DescribeInstancesResponse{}
		err := awsQueryAPI(ctx, account, "ec2", region, awsEC2APIVersion, "DescribeInstances", nextToken, &resp)
		if err != nil {
			return assets, err
		}
		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeComputeInstance, "aws_instance", instance.InstanceId, awsTagName(instance.Tags), now)
				asset.Zone = instance.Placement.AvailabilityZone
				asset.Status = instance.State.Name
				asset.PublicIp = instance.PublicIpAddress
				asset.PrivateIp = instance.PrivateIPAddress
				asset.Tags = awsTagsToAttrs(instance.Tags)
				asset.Attributes = models.ResAttrs{
					"instanceType":   instance.InstanceType,
					"imageId":        instance.ImageId,
					"vpcId":          instance.VpcId,
					"subnetId":       instance.SubnetId,
					"privateDnsName": instance.PrivateDnsName,
					"publicDnsName":  instance.PublicDnsName,
					"platform":       instance.Platform,
					"platformDetail": instance.PlatformDetails,
					"launchTime":     instance.LaunchTime,
				}
				asset.RawData["reservationId"] = reservation.ReservationId
				assets = append(assets, asset)
			}
		}
		if resp.NextToken == "" {
			return assets, nil
		}
		nextToken = resp.NextToken
	}
}

type awsEC2DescribeVpcsResponse struct {
	NextToken string      `xml:"nextToken"`
	Vpcs      []awsEC2Vpc `xml:"vpcSet>item"`
}

type awsEC2Vpc struct {
	VpcId     string   `xml:"vpcId"`
	State     string   `xml:"state"`
	CidrBlock string   `xml:"cidrBlock"`
	IsDefault bool     `xml:"isDefault"`
	Tags      []awsTag `xml:"tagSet>item"`
}

func collectAwsVpcs(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	nextToken := ""
	for {
		resp := awsEC2DescribeVpcsResponse{}
		err := awsQueryAPI(ctx, account, "ec2", region, awsEC2APIVersion, "DescribeVpcs", nextToken, &resp)
		if err != nil {
			return assets, err
		}
		for _, vpc := range resp.Vpcs {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkVpc, "aws_vpc", vpc.VpcId, awsTagName(vpc.Tags), now)
			asset.Status = vpc.State
			asset.Tags = awsTagsToAttrs(vpc.Tags)
			asset.Attributes = models.ResAttrs{
				"cidrBlock": vpc.CidrBlock,
				"isDefault": vpc.IsDefault,
			}
			assets = append(assets, asset)
		}
		if resp.NextToken == "" {
			return assets, nil
		}
		nextToken = resp.NextToken
	}
}

type awsEC2DescribeSubnetsResponse struct {
	NextToken string         `xml:"nextToken"`
	Subnets   []awsEC2Subnet `xml:"subnetSet>item"`
}

type awsEC2Subnet struct {
	SubnetId                string   `xml:"subnetId"`
	VpcId                   string   `xml:"vpcId"`
	State                   string   `xml:"state"`
	CidrBlock               string   `xml:"cidrBlock"`
	AvailabilityZone        string   `xml:"availabilityZone"`
	AvailabilityZoneId      string   `xml:"availabilityZoneId"`
	AvailableIpAddressCount int      `xml:"availableIpAddressCount"`
	DefaultForAz            bool     `xml:"defaultForAz"`
	MapPublicIpOnLaunch     bool     `xml:"mapPublicIpOnLaunch"`
	Tags                    []awsTag `xml:"tagSet>item"`
}

func collectAwsSubnets(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	nextToken := ""
	for {
		resp := awsEC2DescribeSubnetsResponse{}
		err := awsQueryAPI(ctx, account, "ec2", region, awsEC2APIVersion, "DescribeSubnets", nextToken, &resp)
		if err != nil {
			return assets, err
		}
		for _, subnet := range resp.Subnets {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkSubnet, "aws_subnet", subnet.SubnetId, awsTagName(subnet.Tags), now)
			asset.Zone = subnet.AvailabilityZone
			asset.Status = subnet.State
			asset.Tags = awsTagsToAttrs(subnet.Tags)
			asset.Attributes = models.ResAttrs{
				"vpcId":                   subnet.VpcId,
				"cidrBlock":               subnet.CidrBlock,
				"availabilityZoneId":      subnet.AvailabilityZoneId,
				"availableIpAddressCount": subnet.AvailableIpAddressCount,
				"defaultForAz":            subnet.DefaultForAz,
				"mapPublicIpOnLaunch":     subnet.MapPublicIpOnLaunch,
			}
			assets = append(assets, asset)
		}
		if resp.NextToken == "" {
			return assets, nil
		}
		nextToken = resp.NextToken
	}
}

type awsEC2DescribeSecurityGroupsResponse struct {
	NextToken      string                `xml:"nextToken"`
	SecurityGroups []awsEC2SecurityGroup `xml:"securityGroupInfo>item"`
}

type awsEC2SecurityGroup struct {
	GroupId     string   `xml:"groupId"`
	GroupName   string   `xml:"groupName"`
	Description string   `xml:"groupDescription"`
	VpcId       string   `xml:"vpcId"`
	Tags        []awsTag `xml:"tagSet>item"`
}

func collectAwsSecurityGroups(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	nextToken := ""
	for {
		resp := awsEC2DescribeSecurityGroupsResponse{}
		err := awsQueryAPI(ctx, account, "ec2", region, awsEC2APIVersion, "DescribeSecurityGroups", nextToken, &resp)
		if err != nil {
			return assets, err
		}
		for _, sg := range resp.SecurityGroups {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkSecurityGroup, "aws_security_group", sg.GroupId, sg.GroupName, now)
			asset.Tags = awsTagsToAttrs(sg.Tags)
			asset.Attributes = models.ResAttrs{
				"description": sg.Description,
				"groupName":   sg.GroupName,
				"vpcId":       sg.VpcId,
			}
			assets = append(assets, asset)
		}
		if resp.NextToken == "" {
			return assets, nil
		}
		nextToken = resp.NextToken
	}
}

type awsEC2DescribeVolumesResponse struct {
	NextToken string         `xml:"nextToken"`
	Volumes   []awsEC2Volume `xml:"volumeSet>item"`
}

type awsEC2Volume struct {
	VolumeId         string   `xml:"volumeId"`
	AvailabilityZone string   `xml:"availabilityZone"`
	Status           string   `xml:"status"`
	Size             int      `xml:"size"`
	VolumeType       string   `xml:"volumeType"`
	Iops             int      `xml:"iops"`
	Encrypted        bool     `xml:"encrypted"`
	Tags             []awsTag `xml:"tagSet>item"`
}

func collectAwsVolumes(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	nextToken := ""
	for {
		resp := awsEC2DescribeVolumesResponse{}
		err := awsQueryAPI(ctx, account, "ec2", region, awsEC2APIVersion, "DescribeVolumes", nextToken, &resp)
		if err != nil {
			return assets, err
		}
		for _, volume := range resp.Volumes {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeBlockVolume, "aws_ebs_volume", volume.VolumeId, awsTagName(volume.Tags), now)
			asset.Zone = volume.AvailabilityZone
			asset.Status = volume.Status
			asset.Tags = awsTagsToAttrs(volume.Tags)
			asset.Attributes = models.ResAttrs{
				"size":       volume.Size,
				"volumeType": volume.VolumeType,
				"iops":       volume.Iops,
				"encrypted":  volume.Encrypted,
			}
			assets = append(assets, asset)
		}
		if resp.NextToken == "" {
			return assets, nil
		}
		nextToken = resp.NextToken
	}
}

type awsEKSListClustersResponse struct {
	Clusters  []string `json:"clusters"`
	NextToken string   `json:"nextToken"`
}

type awsEKSDescribeClusterResponse struct {
	Cluster awsEKSCluster `json:"cluster"`
}

type awsEKSCluster struct {
	Arn                  string            `json:"arn"`
	Name                 string            `json:"name"`
	Status               string            `json:"status"`
	Endpoint             string            `json:"endpoint"`
	Version              string            `json:"version"`
	PlatformVersion      string            `json:"platformVersion"`
	CreatedAt            string            `json:"createdAt"`
	Tags                 map[string]string `json:"tags"`
	ResourcesVpcConfig   awsEKSVpcConfig   `json:"resourcesVpcConfig"`
	EncryptionConfig     interface{}       `json:"encryptionConfig"`
	KubernetesNetworkCfg interface{}       `json:"kubernetesNetworkConfig"`
}

type awsEKSVpcConfig struct {
	VpcId                  string   `json:"vpcId"`
	SubnetIds              []string `json:"subnetIds"`
	SecurityGroupIds       []string `json:"securityGroupIds"`
	ClusterSecurityGroupId string   `json:"clusterSecurityGroupId"`
}

func collectAwsEksClusters(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	nextToken := ""
	now := cmdbNow()
	for {
		values := url.Values{"maxResults": []string{"100"}}
		if nextToken != "" {
			values.Set("nextToken", nextToken)
		}
		resp := awsEKSListClustersResponse{}
		if err := awsJSONAPI(ctx, account, "eks", region, "/clusters", values, &resp); err != nil {
			return assets, err
		}
		for _, clusterName := range resp.Clusters {
			detail := awsEKSDescribeClusterResponse{}
			if err := awsJSONAPI(ctx, account, "eks", region, "/clusters/"+url.PathEscape(clusterName), nil, &detail); err != nil {
				return assets, err
			}
			cluster := detail.Cluster
			nativeId := firstNonEmpty(cluster.Arn, cluster.Name)
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeKubernetesCluster, "aws_eks_cluster", nativeId, cluster.Name, now)
			asset.Status = cluster.Status
			asset.Address = cluster.Endpoint
			asset.Tags = stringMapToResAttrs(cluster.Tags)
			asset.Attributes = models.ResAttrs{
				"name":                   cluster.Name,
				"version":                cluster.Version,
				"platformVersion":        cluster.PlatformVersion,
				"createdAt":              cluster.CreatedAt,
				"vpcId":                  cluster.ResourcesVpcConfig.VpcId,
				"subnetIds":              cluster.ResourcesVpcConfig.SubnetIds,
				"securityGroupIds":       cluster.ResourcesVpcConfig.SecurityGroupIds,
				"clusterSecurityGroupId": cluster.ResourcesVpcConfig.ClusterSecurityGroupId,
				"encryptionConfig":       cluster.EncryptionConfig,
				"kubernetesNetwork":      cluster.KubernetesNetworkCfg,
			}
			assets = append(assets, asset)
		}
		if resp.NextToken == "" {
			return assets, nil
		}
		nextToken = resp.NextToken
	}
}

type awsRDSDescribeDBInstancesResponse struct {
	Result struct {
		Marker      string             `xml:"Marker"`
		DBInstances []awsRDSDBInstance `xml:"DBInstances>DBInstance"`
	} `xml:"DescribeDBInstancesResult"`
}

type awsRDSDBInstance struct {
	DBInstanceIdentifier string `xml:"DBInstanceIdentifier"`
	DBInstanceArn        string `xml:"DBInstanceArn"`
	DBInstanceClass      string `xml:"DBInstanceClass"`
	DBInstanceStatus     string `xml:"DBInstanceStatus"`
	Engine               string `xml:"Engine"`
	EngineVersion        string `xml:"EngineVersion"`
	AllocatedStorage     int    `xml:"AllocatedStorage"`
	StorageType          string `xml:"StorageType"`
	MultiAZ              bool   `xml:"MultiAZ"`
	AvailabilityZone     string `xml:"AvailabilityZone"`
	Endpoint             struct {
		Address string `xml:"Address"`
		Port    int    `xml:"Port"`
	} `xml:"Endpoint"`
	DBSubnetGroup struct {
		VpcId             string `xml:"VpcId"`
		DBSubnetGroupName string `xml:"DBSubnetGroupName"`
	} `xml:"DBSubnetGroup"`
	TagList []awsRDSTag `xml:"TagList>Tag"`
}

type awsRDSTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

func collectAwsRdsInstances(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	marker := ""
	for {
		resp := awsRDSDescribeDBInstancesResponse{}
		err := awsQueryAPI(ctx, account, "rds", region, awsRDSAPIVersion, "DescribeDBInstances", marker, &resp)
		if err != nil {
			return assets, err
		}
		for _, db := range resp.Result.DBInstances {
			nativeId := firstNonEmpty(db.DBInstanceArn, db.DBInstanceIdentifier)
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeRelationalDatabase, "aws_db_instance", nativeId, db.DBInstanceIdentifier, now)
			asset.Zone = db.AvailabilityZone
			asset.Status = db.DBInstanceStatus
			asset.Address = db.Endpoint.Address
			asset.Tags = awsRDSTagsToAttrs(db.TagList)
			asset.Attributes = models.ResAttrs{
				"engine":            db.Engine,
				"engineVersion":     db.EngineVersion,
				"dbInstanceClass":   db.DBInstanceClass,
				"allocatedStorage":  db.AllocatedStorage,
				"storageType":       db.StorageType,
				"multiAZ":           db.MultiAZ,
				"dbInstanceArn":     db.DBInstanceArn,
				"dbInstanceAddress": db.DBInstanceIdentifier,
				"endpoint":          db.Endpoint.Address,
				"port":              db.Endpoint.Port,
				"vpcId":             db.DBSubnetGroup.VpcId,
				"dbSubnetGroupName": db.DBSubnetGroup.DBSubnetGroupName,
			}
			assets = append(assets, asset)
		}
		if resp.Result.Marker == "" {
			return assets, nil
		}
		marker = resp.Result.Marker
	}
}

func collectAwsRedisCaches(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	clusters, err := collectAwsRedisClusters(ctx, account, region)
	assets = append(assets, clusters...)
	if err != nil {
		return assets, err
	}
	replicationGroups, err := collectAwsRedisReplicationGroups(ctx, account, region)
	assets = append(assets, replicationGroups...)
	return assets, err
}

type awsElastiCacheDescribeCacheClustersResponse struct {
	Result struct {
		Marker        string                  `xml:"Marker"`
		CacheClusters []awsElastiCacheCluster `xml:"CacheClusters>CacheCluster"`
	} `xml:"DescribeCacheClustersResult"`
}

type awsElastiCacheCluster struct {
	ARN                       string `xml:"ARN"`
	CacheClusterId            string `xml:"CacheClusterId"`
	CacheClusterStatus        string `xml:"CacheClusterStatus"`
	Engine                    string `xml:"Engine"`
	EngineVersion             string `xml:"EngineVersion"`
	CacheNodeType             string `xml:"CacheNodeType"`
	NumCacheNodes             int    `xml:"NumCacheNodes"`
	PreferredAvailabilityZone string `xml:"PreferredAvailabilityZone"`
	CacheSubnetGroupName      string `xml:"CacheSubnetGroupName"`
	ConfigurationEndpoint     struct {
		Address string `xml:"Address"`
		Port    int    `xml:"Port"`
	} `xml:"ConfigurationEndpoint"`
	CacheNodes []struct {
		Endpoint struct {
			Address string `xml:"Address"`
			Port    int    `xml:"Port"`
		} `xml:"Endpoint"`
	} `xml:"CacheNodes>CacheNode"`
}

func collectAwsRedisClusters(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	marker := ""
	for {
		resp := awsElastiCacheDescribeCacheClustersResponse{}
		values := url.Values{
			"Action":            []string{"DescribeCacheClusters"},
			"Version":           []string{awsElastiCacheAPIVersion},
			"ShowCacheNodeInfo": []string{"true"},
		}
		if marker != "" {
			values.Set("Marker", marker)
		}
		err := awsQueryAPIWithValues(ctx, account, "elasticache", region, values, &resp)
		if err != nil {
			return assets, err
		}
		for _, cluster := range resp.Result.CacheClusters {
			if !strings.EqualFold(cluster.Engine, "redis") && !strings.EqualFold(cluster.Engine, "valkey") {
				continue
			}
			nativeId := firstNonEmpty(cluster.ARN, cluster.CacheClusterId)
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeRedisCache, "aws_elasticache_cluster", nativeId, cluster.CacheClusterId, now)
			asset.Zone = cluster.PreferredAvailabilityZone
			asset.Status = cluster.CacheClusterStatus
			attrs := models.ResAttrs{
				"engine":               cluster.Engine,
				"engineVersion":        cluster.EngineVersion,
				"cacheNodeType":        cluster.CacheNodeType,
				"numCacheNodes":        cluster.NumCacheNodes,
				"cacheSubnetGroupName": cluster.CacheSubnetGroupName,
			}
			if cluster.ConfigurationEndpoint.Address != "" {
				asset.Address = cluster.ConfigurationEndpoint.Address
				attrs["configurationEndpoint"] = awsAddressPortAttrs(cluster.ConfigurationEndpoint.Address, cluster.ConfigurationEndpoint.Port)
			} else if len(cluster.CacheNodes) > 0 {
				asset.Address = cluster.CacheNodes[0].Endpoint.Address
				attrs["endpoint"] = awsAddressPortAttrs(cluster.CacheNodes[0].Endpoint.Address, cluster.CacheNodes[0].Endpoint.Port)
			}
			asset.Attributes = attrs
			assets = append(assets, asset)
		}
		if resp.Result.Marker == "" {
			return assets, nil
		}
		marker = resp.Result.Marker
	}
}

type awsElastiCacheDescribeReplicationGroupsResponse struct {
	Result struct {
		Marker            string                           `xml:"Marker"`
		ReplicationGroups []awsElastiCacheReplicationGroup `xml:"ReplicationGroups>ReplicationGroup"`
	} `xml:"DescribeReplicationGroupsResult"`
}

type awsElastiCacheReplicationGroup struct {
	ARN                      string   `xml:"ARN"`
	ReplicationGroupId       string   `xml:"ReplicationGroupId"`
	Status                   string   `xml:"Status"`
	Description              string   `xml:"Description"`
	CacheNodeType            string   `xml:"CacheNodeType"`
	MemberClusters           []string `xml:"MemberClusters>ClusterId"`
	ClusterEnabled           bool     `xml:"ClusterEnabled"`
	AutomaticFailover        string   `xml:"AutomaticFailover"`
	TransitEncryptionEnabled bool     `xml:"TransitEncryptionEnabled"`
	AtRestEncryptionEnabled  bool     `xml:"AtRestEncryptionEnabled"`
	ConfigurationEndpoint    struct {
		Address string `xml:"Address"`
		Port    int    `xml:"Port"`
	} `xml:"ConfigurationEndpoint"`
	NodeGroups []awsElastiCacheNodeGroup `xml:"NodeGroups>NodeGroup"`
}

type awsElastiCacheNodeGroup struct {
	NodeGroupId string `xml:"NodeGroupId"`
	Status      string `xml:"Status"`
	Primary     struct {
		Address string `xml:"Address"`
		Port    int    `xml:"Port"`
	} `xml:"PrimaryEndpoint"`
	Reader struct {
		Address string `xml:"Address"`
		Port    int    `xml:"Port"`
	} `xml:"ReaderEndpoint"`
}

func collectAwsRedisReplicationGroups(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	marker := ""
	for {
		resp := awsElastiCacheDescribeReplicationGroupsResponse{}
		err := awsQueryAPI(ctx, account, "elasticache", region, awsElastiCacheAPIVersion, "DescribeReplicationGroups", marker, &resp)
		if err != nil {
			return assets, err
		}
		for _, group := range resp.Result.ReplicationGroups {
			nativeId := firstNonEmpty(group.ARN, group.ReplicationGroupId)
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeRedisCache, "aws_elasticache_replication_group", nativeId, group.ReplicationGroupId, now)
			asset.Status = group.Status
			attrs := models.ResAttrs{
				"description":              group.Description,
				"cacheNodeType":            group.CacheNodeType,
				"memberClusters":           group.MemberClusters,
				"clusterEnabled":           group.ClusterEnabled,
				"automaticFailover":        group.AutomaticFailover,
				"transitEncryptionEnabled": group.TransitEncryptionEnabled,
				"atRestEncryptionEnabled":  group.AtRestEncryptionEnabled,
				"nodeGroups":               awsRedisNodeGroups(group.NodeGroups),
			}
			if group.ConfigurationEndpoint.Address != "" {
				asset.Address = group.ConfigurationEndpoint.Address
				attrs["configurationEndpoint"] = awsAddressPortAttrs(group.ConfigurationEndpoint.Address, group.ConfigurationEndpoint.Port)
			} else if len(group.NodeGroups) > 0 {
				asset.Address = group.NodeGroups[0].Primary.Address
			}
			asset.Attributes = attrs
			assets = append(assets, asset)
		}
		if resp.Result.Marker == "" {
			return assets, nil
		}
		marker = resp.Result.Marker
	}
}

func awsQueryAPI(ctx context.Context, account *cmdbCloudAccount, service, region, version, action, marker string, out interface{}) error {
	values := url.Values{
		"Action":  []string{action},
		"Version": []string{version},
	}
	if marker != "" {
		switch service {
		case "ec2":
			values.Set("NextToken", marker)
		default:
			values.Set("Marker", marker)
		}
	}
	return awsQueryAPIWithValues(ctx, account, service, region, values, out)
}

func awsQueryAPIWithValues(ctx context.Context, account *cmdbCloudAccount, service, region string, values url.Values, out interface{}) error {
	host := fmt.Sprintf("%s.%s.amazonaws.com", service, region)
	body := []byte(values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+host+"/", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	if err := signAWSV4(req, body, account, service, region, time.Now().UTC()); err != nil {
		return err
	}
	respBody, err := doAWSRequest(req)
	if err != nil {
		return err
	}
	return xml.Unmarshal(respBody, out)
}

func awsJSONAPI(ctx context.Context, account *cmdbCloudAccount, service, region, path string, values url.Values, out interface{}) error {
	host := fmt.Sprintf("%s.%s.amazonaws.com", service, region)
	u := url.URL{
		Scheme:   "https",
		Host:     host,
		Path:     path,
		RawQuery: values.Encode(),
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	if err := signAWSV4(req, nil, account, service, region, time.Now().UTC()); err != nil {
		return err
	}
	respBody, err := doAWSRequest(req)
	if err != nil {
		return err
	}
	return json.Unmarshal(respBody, out)
}

func doAWSRequest(req *http.Request) ([]byte, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func signAWSV4(req *http.Request, body []byte, account *cmdbCloudAccount, service, region string, now time.Time) error {
	accessKey := strings.TrimSpace(account.Credentials["AWS_ACCESS_KEY_ID"])
	secretKey := strings.TrimSpace(account.Credentials["AWS_SECRET_ACCESS_KEY"])
	if accessKey == "" || secretKey == "" {
		return fmt.Errorf("missing AWS_ACCESS_KEY_ID or AWS_SECRET_ACCESS_KEY")
	}

	payloadHash := sha256Hex(body)
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if token := strings.TrimSpace(account.Credentials["AWS_SESSION_TOKEN"]); token != "" {
		req.Header.Set("X-Amz-Security-Token", token)
	}

	signedHeaders := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	if req.Header.Get("X-Amz-Security-Token") != "" {
		signedHeaders = append(signedHeaders, "x-amz-security-token")
	}
	sort.Strings(signedHeaders)

	canonicalHeaders := strings.Builder{}
	for _, header := range signedHeaders {
		canonicalHeaders.WriteString(header)
		canonicalHeaders.WriteByte(':')
		if header == "host" {
			canonicalHeaders.WriteString(req.URL.Host)
		} else {
			canonicalHeaders.WriteString(strings.TrimSpace(req.Header.Get(header)))
		}
		canonicalHeaders.WriteByte('\n')
	}

	canonicalURI := req.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	credentialScope := strings.Join([]string{dateStamp, region, service, "aws4_request"}, "/")
	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQuery(req.URL.Query()),
		canonicalHeaders.String(),
		strings.Join(signedHeaders, ";"),
		payloadHash,
	}, "\n")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	signingKey := awsSigningKey(secretKey, dateStamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, credentialScope, strings.Join(signedHeaders, ";"), signature))
	return nil
}

func awsSigningKey(secretKey, dateStamp, region, service string) []byte {
	dateKey := hmacSHA256([]byte("AWS4"+secretKey), dateStamp)
	dateRegionKey := hmacSHA256(dateKey, region)
	dateRegionServiceKey := hmacSHA256(dateRegionKey, service)
	return hmacSHA256(dateRegionServiceKey, "aws4_request")
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func canonicalQuery(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0)
	for _, key := range keys {
		items := append([]string(nil), values[key]...)
		sort.Strings(items)
		for _, value := range items {
			parts = append(parts, awsURIEncode(key)+"="+awsURIEncode(value))
		}
	}
	return strings.Join(parts, "&")
}

func awsURIEncode(value string) string {
	escaped := url.QueryEscape(value)
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	escaped = strings.ReplaceAll(escaped, "%7E", "~")
	return escaped
}

func awsTagName(tags []awsTag) string {
	for _, tag := range tags {
		if strings.EqualFold(tag.Key, "Name") {
			return tag.Value
		}
	}
	return ""
}

func awsTagsToAttrs(tags []awsTag) models.ResAttrs {
	result := models.ResAttrs{}
	for _, tag := range tags {
		if tag.Key == "" {
			continue
		}
		result[tag.Key] = tag.Value
	}
	return result
}

func awsRDSTagsToAttrs(tags []awsRDSTag) models.ResAttrs {
	result := models.ResAttrs{}
	for _, tag := range tags {
		if tag.Key == "" {
			continue
		}
		result[tag.Key] = tag.Value
	}
	return result
}

func stringMapToResAttrs(tags map[string]string) models.ResAttrs {
	result := models.ResAttrs{}
	for key, value := range tags {
		if key == "" {
			continue
		}
		result[key] = value
	}
	return result
}

func awsAddressPortAttrs(address string, port int) models.ResAttrs {
	return models.ResAttrs{
		"address": address,
		"port":    port,
	}
}

func awsRedisNodeGroups(groups []awsElastiCacheNodeGroup) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(groups))
	for _, group := range groups {
		item := models.ResAttrs{
			"nodeGroupId": group.NodeGroupId,
			"status":      group.Status,
		}
		if group.Primary.Address != "" {
			item["primaryEndpoint"] = awsAddressPortAttrs(group.Primary.Address, group.Primary.Port)
		}
		if group.Reader.Address != "" {
			item["readerEndpoint"] = awsAddressPortAttrs(group.Reader.Address, group.Reader.Port)
		}
		result = append(result, item)
	}
	return result
}
