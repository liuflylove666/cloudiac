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
	awsELBAPIVersion         = "2012-06-01"
	awsELBV2APIVersion       = "2015-12-01"
	awsRDSAPIVersion         = "2014-10-31"
	awsElastiCacheAPIVersion = "2015-02-02"
	awsS3DefaultRegion       = "us-east-1"
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
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkRouteTable) {
		items, err := collectAwsRouteTables(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("aws %s route tables: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkNatGateway) {
		items, err := collectAwsNatGateways(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("aws %s nat gateways: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkInternetGateway) {
		items, err := collectAwsInternetGateways(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("aws %s internet gateways: %v", region, err))
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
	if wantsCmdbAssetType(selected, models.CmdbAssetTypePublicIP) {
		items, err := collectAwsAddresses(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("aws %s elastic ips: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeLoadBalancer) {
		items, err := collectAwsLoadBalancers(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("aws %s load balancers: %v", region, err))
		}
	}
	if wantsCmdbAssetType(selected, models.CmdbAssetTypeObjectStorageBucket) {
		items, err := collectAwsS3Buckets(ctx, account, region)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("aws %s s3 buckets: %v", region, err))
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

type awsEC2DescribeRouteTablesResponse struct {
	NextToken   string             `xml:"nextToken"`
	RouteTables []awsEC2RouteTable `xml:"routeTableSet>item"`
}

type awsEC2RouteTable struct {
	RouteTableId    string                     `xml:"routeTableId"`
	VpcId           string                     `xml:"vpcId"`
	OwnerId         string                     `xml:"ownerId"`
	Associations    []awsEC2RouteTableAssoc    `xml:"associationSet>item"`
	Routes          []awsEC2Route              `xml:"routeSet>item"`
	PropagatingVgws []awsEC2PropagatingGateway `xml:"propagatingVgwSet>item"`
	Tags            []awsTag                   `xml:"tagSet>item"`
}

type awsEC2RouteTableAssoc struct {
	RouteTableAssociationId string `xml:"routeTableAssociationId"`
	RouteTableId            string `xml:"routeTableId"`
	SubnetId                string `xml:"subnetId"`
	GatewayId               string `xml:"gatewayId"`
	Main                    bool   `xml:"main"`
	AssociationState        struct {
		State string `xml:"state"`
	} `xml:"associationState"`
}

type awsEC2Route struct {
	DestinationCidrBlock        string `xml:"destinationCidrBlock"`
	DestinationIpv6CidrBlock    string `xml:"destinationIpv6CidrBlock"`
	DestinationPrefixListId     string `xml:"destinationPrefixListId"`
	GatewayId                   string `xml:"gatewayId"`
	NatGatewayId                string `xml:"natGatewayId"`
	InstanceId                  string `xml:"instanceId"`
	NetworkInterfaceId          string `xml:"networkInterfaceId"`
	TransitGatewayId            string `xml:"transitGatewayId"`
	VpcPeeringConnectionId      string `xml:"vpcPeeringConnectionId"`
	EgressOnlyInternetGatewayId string `xml:"egressOnlyInternetGatewayId"`
	CarrierGatewayId            string `xml:"carrierGatewayId"`
	LocalGatewayId              string `xml:"localGatewayId"`
	State                       string `xml:"state"`
	Origin                      string `xml:"origin"`
}

type awsEC2PropagatingGateway struct {
	GatewayId string `xml:"gatewayId"`
}

func collectAwsRouteTables(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	nextToken := ""
	for {
		resp := awsEC2DescribeRouteTablesResponse{}
		err := awsQueryAPI(ctx, account, "ec2", region, awsEC2APIVersion, "DescribeRouteTables", nextToken, &resp)
		if err != nil {
			return assets, err
		}
		for _, rt := range resp.RouteTables {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkRouteTable, "aws_route_table", rt.RouteTableId, awsTagName(rt.Tags), now)
			asset.Status = "available"
			asset.Tags = awsTagsToAttrs(rt.Tags)
			subnetIds, gatewayIds := awsRouteTableAssociationRefs(rt.Associations)
			natGatewayIds, internetGatewayIds := awsRouteRefs(rt.Routes)
			asset.Attributes = models.ResAttrs{
				"routeTableId":       rt.RouteTableId,
				"vpcId":              rt.VpcId,
				"ownerId":            rt.OwnerId,
				"subnetIds":          subnetIds,
				"gatewayIds":         gatewayIds,
				"natGatewayIds":      natGatewayIds,
				"internetGatewayIds": internetGatewayIds,
				"associations":       awsRouteTableAssociations(rt.Associations),
				"routes":             awsRoutes(rt.Routes),
				"propagatingVgws":    awsPropagatingGateways(rt.PropagatingVgws),
			}
			assets = append(assets, asset)
		}
		if resp.NextToken == "" {
			return assets, nil
		}
		nextToken = resp.NextToken
	}
}

type awsEC2DescribeNatGatewaysResponse struct {
	NextToken   string             `xml:"nextToken"`
	NatGateways []awsEC2NatGateway `xml:"natGatewaySet>item"`
}

type awsEC2NatGateway struct {
	NatGatewayId        string                    `xml:"natGatewayId"`
	SubnetId            string                    `xml:"subnetId"`
	VpcId               string                    `xml:"vpcId"`
	State               string                    `xml:"state"`
	FailureCode         string                    `xml:"failureCode"`
	FailureMessage      string                    `xml:"failureMessage"`
	CreateTime          string                    `xml:"createTime"`
	DeleteTime          string                    `xml:"deleteTime"`
	ConnectivityType    string                    `xml:"connectivityType"`
	NatGatewayAddresses []awsEC2NatGatewayAddress `xml:"natGatewayAddressSet>item"`
	Tags                []awsTag                  `xml:"tagSet>item"`
}

type awsEC2NatGatewayAddress struct {
	AllocationId       string `xml:"allocationId"`
	AssociationId      string `xml:"associationId"`
	PublicIp           string `xml:"publicIp"`
	PrivateIp          string `xml:"privateIp"`
	NetworkInterfaceId string `xml:"networkInterfaceId"`
}

func collectAwsNatGateways(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	nextToken := ""
	for {
		resp := awsEC2DescribeNatGatewaysResponse{}
		err := awsQueryAPI(ctx, account, "ec2", region, awsEC2APIVersion, "DescribeNatGateways", nextToken, &resp)
		if err != nil {
			return assets, err
		}
		for _, nat := range resp.NatGateways {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkNatGateway, "aws_nat_gateway", nat.NatGatewayId, awsTagName(nat.Tags), now)
			asset.Status = nat.State
			asset.Tags = awsTagsToAttrs(nat.Tags)
			publicIps, allocationIds := awsNatGatewayPublicRefs(nat.NatGatewayAddresses)
			asset.PublicIp = strings.Join(publicIps, ",")
			asset.Attributes = models.ResAttrs{
				"natGatewayId":     nat.NatGatewayId,
				"vpcId":            nat.VpcId,
				"subnetId":         nat.SubnetId,
				"connectivityType": nat.ConnectivityType,
				"failureCode":      nat.FailureCode,
				"failureMessage":   nat.FailureMessage,
				"createTime":       nat.CreateTime,
				"deleteTime":       nat.DeleteTime,
				"publicIps":        publicIps,
				"publicIpIds":      allocationIds,
				"allocationIds":    allocationIds,
				"addresses":        awsNatGatewayAddresses(nat.NatGatewayAddresses),
			}
			assets = append(assets, asset)
		}
		if resp.NextToken == "" {
			return assets, nil
		}
		nextToken = resp.NextToken
	}
}

type awsEC2DescribeInternetGatewaysResponse struct {
	NextToken        string                  `xml:"nextToken"`
	InternetGateways []awsEC2InternetGateway `xml:"internetGatewaySet>item"`
}

type awsEC2InternetGateway struct {
	InternetGatewayId string                        `xml:"internetGatewayId"`
	OwnerId           string                        `xml:"ownerId"`
	Attachments       []awsEC2InternetGatewayAttach `xml:"attachmentSet>item"`
	Tags              []awsTag                      `xml:"tagSet>item"`
}

type awsEC2InternetGatewayAttach struct {
	VpcId string `xml:"vpcId"`
	State string `xml:"state"`
}

func collectAwsInternetGateways(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	nextToken := ""
	for {
		resp := awsEC2DescribeInternetGatewaysResponse{}
		err := awsQueryAPI(ctx, account, "ec2", region, awsEC2APIVersion, "DescribeInternetGateways", nextToken, &resp)
		if err != nil {
			return assets, err
		}
		for _, igw := range resp.InternetGateways {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkInternetGateway, "aws_internet_gateway", igw.InternetGatewayId, awsTagName(igw.Tags), now)
			asset.Status = awsInternetGatewayStatus(igw.Attachments)
			asset.Tags = awsTagsToAttrs(igw.Tags)
			asset.Attributes = models.ResAttrs{
				"internetGatewayId": igw.InternetGatewayId,
				"ownerId":           igw.OwnerId,
				"vpcIds":            awsInternetGatewayVpcIds(igw.Attachments),
				"attachments":       awsInternetGatewayAttachments(igw.Attachments),
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
	GroupId             string                          `xml:"groupId"`
	GroupName           string                          `xml:"groupName"`
	Description         string                          `xml:"groupDescription"`
	VpcId               string                          `xml:"vpcId"`
	Tags                []awsTag                        `xml:"tagSet>item"`
	IpPermissions       []awsEC2SecurityGroupPermission `xml:"ipPermissions>item"`
	IpPermissionsEgress []awsEC2SecurityGroupPermission `xml:"ipPermissionsEgress>item"`
}

type awsEC2SecurityGroupPermission struct {
	IpProtocol       string                          `xml:"ipProtocol"`
	FromPort         string                          `xml:"fromPort"`
	ToPort           string                          `xml:"toPort"`
	IpRanges         []awsEC2SecurityGroupIpRange    `xml:"ipRanges>item"`
	Ipv6Ranges       []awsEC2SecurityGroupIpv6Range  `xml:"ipv6Ranges>item"`
	PrefixListIds    []awsEC2SecurityGroupPrefixList `xml:"prefixListIds>item"`
	UserIdGroupPairs []awsEC2SecurityGroupPair       `xml:"groups>item"`
}

type awsEC2SecurityGroupIpRange struct {
	CidrIp      string `xml:"cidrIp"`
	Description string `xml:"description"`
}

type awsEC2SecurityGroupIpv6Range struct {
	CidrIpv6    string `xml:"cidrIpv6"`
	Description string `xml:"description"`
}

type awsEC2SecurityGroupPrefixList struct {
	PrefixListId string `xml:"prefixListId"`
	Description  string `xml:"description"`
}

type awsEC2SecurityGroupPair struct {
	GroupId     string `xml:"groupId"`
	GroupName   string `xml:"groupName"`
	UserId      string `xml:"userId"`
	Description string `xml:"description"`
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
				"description":  sg.Description,
				"groupName":    sg.GroupName,
				"vpcId":        sg.VpcId,
				"ingressRules": awsSecurityGroupPermissionRules(sg.IpPermissions, "ingress"),
				"egressRules":  awsSecurityGroupPermissionRules(sg.IpPermissionsEgress, "egress"),
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

type awsEC2DescribeAddressesResponse struct {
	NextToken string          `xml:"nextToken"`
	Addresses []awsEC2Address `xml:"addressesSet>item"`
}

type awsEC2Address struct {
	AllocationId            string   `xml:"allocationId"`
	AssociationId           string   `xml:"associationId"`
	PublicIp                string   `xml:"publicIp"`
	PrivateIpAddress        string   `xml:"privateIpAddress"`
	Domain                  string   `xml:"domain"`
	InstanceId              string   `xml:"instanceId"`
	NetworkInterfaceId      string   `xml:"networkInterfaceId"`
	NetworkInterfaceOwnerId string   `xml:"networkInterfaceOwnerId"`
	PublicIpv4Pool          string   `xml:"publicIpv4Pool"`
	NetworkBorderGroup      string   `xml:"networkBorderGroup"`
	Tags                    []awsTag `xml:"tagSet>item"`
}

func collectAwsAddresses(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	nextToken := ""
	for {
		resp := awsEC2DescribeAddressesResponse{}
		err := awsQueryAPI(ctx, account, "ec2", region, awsEC2APIVersion, "DescribeAddresses", nextToken, &resp)
		if err != nil {
			return assets, err
		}
		for _, address := range resp.Addresses {
			nativeId := firstNonEmpty(address.AllocationId, address.PublicIp)
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypePublicIP, "aws_eip", nativeId, firstNonEmpty(awsTagName(address.Tags), address.PublicIp, nativeId), now)
			asset.Status = awsAddressStatus(address)
			asset.Address = address.PublicIp
			asset.PublicIp = address.PublicIp
			asset.PrivateIp = address.PrivateIpAddress
			asset.Tags = awsTagsToAttrs(address.Tags)
			asset.Attributes = models.ResAttrs{
				"allocationId":            address.AllocationId,
				"associationId":           address.AssociationId,
				"publicIp":                address.PublicIp,
				"ipAddress":               address.PublicIp,
				"privateIpAddress":        address.PrivateIpAddress,
				"domain":                  address.Domain,
				"instanceId":              address.InstanceId,
				"networkInterfaceId":      address.NetworkInterfaceId,
				"networkInterfaceOwnerId": address.NetworkInterfaceOwnerId,
				"publicIpv4Pool":          address.PublicIpv4Pool,
				"networkBorderGroup":      address.NetworkBorderGroup,
			}
			assets = append(assets, asset)
		}
		if resp.NextToken == "" {
			return assets, nil
		}
		nextToken = resp.NextToken
	}
}

func collectAwsLoadBalancers(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	classic, err := collectAwsClassicLoadBalancers(ctx, account, region)
	assets = append(assets, classic...)
	if err != nil {
		return assets, err
	}
	v2, err := collectAwsV2LoadBalancers(ctx, account, region)
	assets = append(assets, v2...)
	return assets, err
}

type awsELBDescribeLoadBalancersResponse struct {
	Result struct {
		NextMarker               string                       `xml:"NextMarker"`
		LoadBalancerDescriptions []awsClassicLoadBalancerDesc `xml:"LoadBalancerDescriptions>member"`
	} `xml:"DescribeLoadBalancersResult"`
}

type awsClassicLoadBalancerDesc struct {
	LoadBalancerName          string   `xml:"LoadBalancerName"`
	DNSName                   string   `xml:"DNSName"`
	CanonicalHostedZoneName   string   `xml:"CanonicalHostedZoneName"`
	CanonicalHostedZoneNameID string   `xml:"CanonicalHostedZoneNameID"`
	Scheme                    string   `xml:"Scheme"`
	VPCId                     string   `xml:"VPCId"`
	CreatedTime               string   `xml:"CreatedTime"`
	AvailabilityZones         []string `xml:"AvailabilityZones>member"`
	Subnets                   []string `xml:"Subnets>member"`
	SecurityGroups            []string `xml:"SecurityGroups>member"`
	Instances                 []struct {
		InstanceId string `xml:"InstanceId"`
	} `xml:"Instances>member"`
	SourceSecurityGroup struct {
		GroupName  string `xml:"GroupName"`
		OwnerAlias string `xml:"OwnerAlias"`
	} `xml:"SourceSecurityGroup"`
}

func collectAwsClassicLoadBalancers(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	marker := ""
	for {
		resp := awsELBDescribeLoadBalancersResponse{}
		err := awsQueryAPI(ctx, account, "elasticloadbalancing", region, awsELBAPIVersion, "DescribeLoadBalancers", marker, &resp)
		if err != nil {
			return assets, err
		}
		for _, lb := range resp.Result.LoadBalancerDescriptions {
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeLoadBalancer, "aws_elb_load_balancer", lb.LoadBalancerName, lb.LoadBalancerName, now)
			asset.Status = "active"
			asset.Address = lb.DNSName
			asset.Attributes = models.ResAttrs{
				"name":                      lb.LoadBalancerName,
				"dnsName":                   lb.DNSName,
				"canonicalHostedZoneName":   lb.CanonicalHostedZoneName,
				"canonicalHostedZoneNameID": lb.CanonicalHostedZoneNameID,
				"scheme":                    lb.Scheme,
				"type":                      "classic",
				"vpcId":                     lb.VPCId,
				"availabilityZones":         lb.AvailabilityZones,
				"subnetIds":                 lb.Subnets,
				"securityGroupIds":          lb.SecurityGroups,
				"instanceIds":               awsClassicLoadBalancerInstanceIds(lb.Instances),
				"sourceSecurityGroup": models.ResAttrs{
					"groupName":  lb.SourceSecurityGroup.GroupName,
					"ownerAlias": lb.SourceSecurityGroup.OwnerAlias,
				},
				"createdTime": lb.CreatedTime,
			}
			assets = append(assets, asset)
		}
		if resp.Result.NextMarker == "" {
			return assets, nil
		}
		marker = resp.Result.NextMarker
	}
}

type awsELBV2DescribeLoadBalancersResponse struct {
	Result struct {
		NextMarker    string              `xml:"NextMarker"`
		LoadBalancers []awsV2LoadBalancer `xml:"LoadBalancers>member"`
	} `xml:"DescribeLoadBalancersResult"`
}

type awsV2LoadBalancer struct {
	LoadBalancerArn       string   `xml:"LoadBalancerArn"`
	LoadBalancerName      string   `xml:"LoadBalancerName"`
	DNSName               string   `xml:"DNSName"`
	CanonicalHostedZoneId string   `xml:"CanonicalHostedZoneId"`
	Scheme                string   `xml:"Scheme"`
	Type                  string   `xml:"Type"`
	VpcId                 string   `xml:"VpcId"`
	IpAddressType         string   `xml:"IpAddressType"`
	CreatedTime           string   `xml:"CreatedTime"`
	SecurityGroups        []string `xml:"SecurityGroups>member"`
	State                 struct {
		Code   string `xml:"Code"`
		Reason string `xml:"Reason"`
	} `xml:"State"`
	AvailabilityZones []awsV2LoadBalancerAZ `xml:"AvailabilityZones>member"`
}

type awsV2LoadBalancerAZ struct {
	ZoneName              string                     `xml:"ZoneName"`
	SubnetId              string                     `xml:"SubnetId"`
	LoadBalancerAddresses []awsV2LoadBalancerAddress `xml:"LoadBalancerAddresses>member"`
}

type awsV2LoadBalancerAddress struct {
	IpAddress          string `xml:"IpAddress"`
	AllocationId       string `xml:"AllocationId"`
	PrivateIPv4Address string `xml:"PrivateIPv4Address"`
	IPv6Address        string `xml:"IPv6Address"`
}

type awsELBV2DescribeListenersResponse struct {
	Result struct {
		NextMarker string          `xml:"NextMarker"`
		Listeners  []awsV2Listener `xml:"Listeners>member"`
	} `xml:"DescribeListenersResult"`
}

type awsV2Listener struct {
	ListenerArn     string `xml:"ListenerArn"`
	LoadBalancerArn string `xml:"LoadBalancerArn"`
	Port            string `xml:"Port"`
	Protocol        string `xml:"Protocol"`
	SslPolicy       string `xml:"SslPolicy"`
	Certificates    []struct {
		CertificateArn string `xml:"CertificateArn"`
		IsDefault      string `xml:"IsDefault"`
	} `xml:"Certificates>member"`
	DefaultActions []awsV2ListenerAction `xml:"DefaultActions>member"`
}

type awsV2ListenerAction struct {
	Type                      string                         `xml:"Type"`
	TargetGroupArn            string                         `xml:"TargetGroupArn"`
	Order                     string                         `xml:"Order"`
	AuthenticateCognitoConfig awsV2AuthenticateCognitoConfig `xml:"AuthenticateCognitoConfig"`
	AuthenticateOidcConfig    awsV2AuthenticateOidcConfig    `xml:"AuthenticateOidcConfig"`
	RedirectConfig            struct {
		Protocol   string `xml:"Protocol"`
		Port       string `xml:"Port"`
		Host       string `xml:"Host"`
		Path       string `xml:"Path"`
		Query      string `xml:"Query"`
		StatusCode string `xml:"StatusCode"`
	} `xml:"RedirectConfig"`
	FixedResponseConfig struct {
		MessageBody string `xml:"MessageBody"`
		StatusCode  string `xml:"StatusCode"`
		ContentType string `xml:"ContentType"`
	} `xml:"FixedResponseConfig"`
	ForwardConfig struct {
		TargetGroups []struct {
			TargetGroupArn string `xml:"TargetGroupArn"`
			Weight         string `xml:"Weight"`
		} `xml:"TargetGroups>member"`
	} `xml:"ForwardConfig"`
	JwtValidationConfig awsV2JwtValidationConfig `xml:"JwtValidationConfig"`
}

type awsV2AuthExtraParam struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

type awsV2AuthenticateCognitoConfig struct {
	UserPoolArn                      string                `xml:"UserPoolArn"`
	UserPoolClientId                 string                `xml:"UserPoolClientId"`
	UserPoolDomain                   string                `xml:"UserPoolDomain"`
	SessionCookieName                string                `xml:"SessionCookieName"`
	Scope                            string                `xml:"Scope"`
	SessionTimeout                   string                `xml:"SessionTimeout"`
	AuthenticationRequestExtraParams []awsV2AuthExtraParam `xml:"AuthenticationRequestExtraParams>entry"`
	OnUnauthenticatedRequest         string                `xml:"OnUnauthenticatedRequest"`
}

type awsV2AuthenticateOidcConfig struct {
	Issuer                           string                `xml:"Issuer"`
	AuthorizationEndpoint            string                `xml:"AuthorizationEndpoint"`
	TokenEndpoint                    string                `xml:"TokenEndpoint"`
	UserInfoEndpoint                 string                `xml:"UserInfoEndpoint"`
	ClientId                         string                `xml:"ClientId"`
	ClientSecret                     string                `xml:"ClientSecret"`
	SessionCookieName                string                `xml:"SessionCookieName"`
	Scope                            string                `xml:"Scope"`
	SessionTimeout                   string                `xml:"SessionTimeout"`
	AuthenticationRequestExtraParams []awsV2AuthExtraParam `xml:"AuthenticationRequestExtraParams>entry"`
	OnUnauthenticatedRequest         string                `xml:"OnUnauthenticatedRequest"`
	UseExistingClientSecret          string                `xml:"UseExistingClientSecret"`
}

type awsV2JwtValidationConfig struct {
	Issuer           string                          `xml:"Issuer"`
	JwksEndpoint     string                          `xml:"JwksEndpoint"`
	AdditionalClaims []awsV2JwtValidationClaimConfig `xml:"AdditionalClaims>member"`
}

type awsV2JwtValidationClaimConfig struct {
	Name   string   `xml:"Name"`
	Format string   `xml:"Format"`
	Values []string `xml:"Values>member"`
}

type awsELBV2DescribeRulesResponse struct {
	Result struct {
		NextMarker string      `xml:"NextMarker"`
		Rules      []awsV2Rule `xml:"Rules>member"`
	} `xml:"DescribeRulesResult"`
}

type awsV2Rule struct {
	RuleArn    string                `xml:"RuleArn"`
	Priority   string                `xml:"Priority"`
	IsDefault  string                `xml:"IsDefault"`
	Conditions []awsV2RuleCondition  `xml:"Conditions>member"`
	Actions    []awsV2ListenerAction `xml:"Actions>member"`
}

type awsV2RuleCondition struct {
	Field            string   `xml:"Field"`
	Values           []string `xml:"Values>member"`
	HostHeaderConfig struct {
		Values []string `xml:"Values>member"`
	} `xml:"HostHeaderConfig"`
	PathPatternConfig struct {
		Values []string `xml:"Values>member"`
	} `xml:"PathPatternConfig"`
	HttpHeaderConfig struct {
		HttpHeaderName string   `xml:"HttpHeaderName"`
		Values         []string `xml:"Values>member"`
	} `xml:"HttpHeaderConfig"`
	QueryStringConfig struct {
		Values []awsV2RuleQueryStringValue `xml:"Values>member"`
	} `xml:"QueryStringConfig"`
	HttpRequestMethodConfig struct {
		Values []string `xml:"Values>member"`
	} `xml:"HttpRequestMethodConfig"`
	SourceIpConfig struct {
		Values []string `xml:"Values>member"`
	} `xml:"SourceIpConfig"`
}

type awsV2RuleQueryStringValue struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type awsELBV2DescribeTargetGroupsResponse struct {
	Result struct {
		NextMarker   string             `xml:"NextMarker"`
		TargetGroups []awsV2TargetGroup `xml:"TargetGroups>member"`
	} `xml:"DescribeTargetGroupsResult"`
}

type awsV2TargetGroup struct {
	TargetGroupArn             string   `xml:"TargetGroupArn"`
	TargetGroupName            string   `xml:"TargetGroupName"`
	Protocol                   string   `xml:"Protocol"`
	ProtocolVersion            string   `xml:"ProtocolVersion"`
	Port                       string   `xml:"Port"`
	VpcId                      string   `xml:"VpcId"`
	TargetType                 string   `xml:"TargetType"`
	IpAddressType              string   `xml:"IpAddressType"`
	HealthCheckEnabled         string   `xml:"HealthCheckEnabled"`
	HealthCheckProtocol        string   `xml:"HealthCheckProtocol"`
	HealthCheckPort            string   `xml:"HealthCheckPort"`
	HealthCheckPath            string   `xml:"HealthCheckPath"`
	HealthCheckIntervalSeconds string   `xml:"HealthCheckIntervalSeconds"`
	HealthCheckTimeoutSeconds  string   `xml:"HealthCheckTimeoutSeconds"`
	HealthyThresholdCount      string   `xml:"HealthyThresholdCount"`
	UnhealthyThresholdCount    string   `xml:"UnhealthyThresholdCount"`
	LoadBalancerArns           []string `xml:"LoadBalancerArns>member"`
	Matcher                    struct {
		HttpCode string `xml:"HttpCode"`
		GrpcCode string `xml:"GrpcCode"`
	} `xml:"Matcher"`
}

type awsELBV2DescribeTargetHealthResponse struct {
	Result struct {
		TargetHealthDescriptions []awsV2TargetHealthDescription `xml:"TargetHealthDescriptions>member"`
	} `xml:"DescribeTargetHealthResult"`
}

type awsV2TargetHealthDescription struct {
	Target struct {
		Id               string `xml:"Id"`
		Port             string `xml:"Port"`
		AvailabilityZone string `xml:"AvailabilityZone"`
	} `xml:"Target"`
	HealthCheckPort string `xml:"HealthCheckPort"`
	TargetHealth    struct {
		State       string `xml:"State"`
		Reason      string `xml:"Reason"`
		Description string `xml:"Description"`
	} `xml:"TargetHealth"`
}

func collectAwsV2LoadBalancers(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	marker := ""
	for {
		resp := awsELBV2DescribeLoadBalancersResponse{}
		err := awsQueryAPI(ctx, account, "elasticloadbalancing", region, awsELBV2APIVersion, "DescribeLoadBalancers", marker, &resp)
		if err != nil {
			return assets, err
		}
		for _, lb := range resp.Result.LoadBalancers {
			nativeId := firstNonEmpty(lb.LoadBalancerArn, lb.LoadBalancerName)
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeLoadBalancer, "aws_lb", nativeId, firstNonEmpty(lb.LoadBalancerName, nativeId), now)
			asset.Status = lb.State.Code
			asset.Address = lb.DNSName
			subnetIds, zoneNames, addresses, publicIpIds := awsV2LoadBalancerNetworkRefs(lb.AvailabilityZones)
			asset.PublicIp = strings.Join(awsV2LoadBalancerPublicIps(addresses), ",")
			asset.Attributes = models.ResAttrs{
				"arn":                   lb.LoadBalancerArn,
				"name":                  lb.LoadBalancerName,
				"dnsName":               lb.DNSName,
				"canonicalHostedZoneId": lb.CanonicalHostedZoneId,
				"scheme":                lb.Scheme,
				"type":                  lb.Type,
				"vpcId":                 lb.VpcId,
				"ipAddressType":         lb.IpAddressType,
				"stateReason":           lb.State.Reason,
				"availabilityZones":     zoneNames,
				"subnetIds":             subnetIds,
				"securityGroupIds":      lb.SecurityGroups,
				"addresses":             addresses,
				"publicIpIds":           publicIpIds,
				"createdTime":           lb.CreatedTime,
			}
			enrichAwsV2LoadBalancerRuntime(ctx, account, region, lb.LoadBalancerArn, asset.Attributes)
			assets = append(assets, asset)
		}
		if resp.Result.NextMarker == "" {
			return assets, nil
		}
		marker = resp.Result.NextMarker
	}
}

func enrichAwsV2LoadBalancerRuntime(ctx context.Context, account *cmdbCloudAccount, region, loadBalancerArn string, attrs models.ResAttrs) {
	if loadBalancerArn == "" {
		return
	}
	listeners, listenerErr := collectAwsV2LoadBalancerListeners(ctx, account, region, loadBalancerArn)
	if listenerErr != nil {
		attrs["listenerCollectError"] = listenerErr.Error()
	} else {
		rulesByListener, ruleErrors := collectAwsV2LoadBalancerListenerRules(ctx, account, region, listeners)
		if len(ruleErrors) > 0 {
			attrs["listenerRuleCollectErrors"] = ruleErrors
		}
		rules := awsV2FlattenRules(rulesByListener)
		ruleTargetGroupArns := awsV2RuleTargetGroupArns(rules)
		attrs["listeners"] = awsV2ListenerAttrs(listeners, rulesByListener)
		attrs["listenerCount"] = len(listeners)
		attrs["listenerPorts"] = awsV2ListenerPorts(listeners)
		attrs["listenerProtocols"] = awsV2ListenerProtocols(listeners)
		attrs["listenerTargetGroupArns"] = dedupeStrings(append(awsV2ListenerTargetGroupArns(listeners), ruleTargetGroupArns...))
		attrs["listenerRules"] = awsV2RuleAttrs(rules)
		attrs["listenerRuleCount"] = len(rules)
		attrs["ruleTargetGroupArns"] = ruleTargetGroupArns
	}

	targetGroups, targetGroupErr := collectAwsV2LoadBalancerTargetGroups(ctx, account, region, loadBalancerArn)
	if targetGroupErr != nil {
		attrs["targetGroupCollectError"] = targetGroupErr.Error()
		return
	}
	groups := make([]models.ResAttrs, 0, len(targetGroups))
	allTargetGroupArns := make([]string, 0, len(targetGroups))
	allTargetIds := make([]string, 0)
	allInstanceIds := make([]string, 0)
	allIpTargets := make([]string, 0)
	healthyCount := 0
	unhealthyCount := 0
	for _, group := range targetGroups {
		targets, targetHealthErr := collectAwsV2TargetHealth(ctx, account, region, group.TargetGroupArn)
		groupAttrs := awsV2TargetGroupAttrs(group, targets)
		if targetHealthErr != nil {
			groupAttrs["targetHealthCollectError"] = targetHealthErr.Error()
		}
		groups = append(groups, groupAttrs)
		if group.TargetGroupArn != "" {
			allTargetGroupArns = append(allTargetGroupArns, group.TargetGroupArn)
		}
		targetIds, instanceIds, ipTargets, healthy, unhealthy := awsV2TargetHealthRefs(targets)
		allTargetIds = append(allTargetIds, targetIds...)
		allInstanceIds = append(allInstanceIds, instanceIds...)
		allIpTargets = append(allIpTargets, ipTargets...)
		healthyCount += healthy
		unhealthyCount += unhealthy
	}
	attrs["targetGroups"] = groups
	attrs["targetGroupCount"] = len(groups)
	attrs["targetGroupArns"] = dedupeStrings(allTargetGroupArns)
	attrs["targetIds"] = dedupeStrings(allTargetIds)
	attrs["targetInstanceIds"] = dedupeStrings(allInstanceIds)
	attrs["targetIpAddresses"] = dedupeStrings(allIpTargets)
	attrs["healthyTargetCount"] = healthyCount
	attrs["unhealthyTargetCount"] = unhealthyCount
}

func collectAwsV2LoadBalancerListeners(ctx context.Context, account *cmdbCloudAccount, region, loadBalancerArn string) ([]awsV2Listener, error) {
	listeners := make([]awsV2Listener, 0)
	marker := ""
	for {
		values := url.Values{
			"Action":          []string{"DescribeListeners"},
			"Version":         []string{awsELBV2APIVersion},
			"LoadBalancerArn": []string{loadBalancerArn},
		}
		if marker != "" {
			values.Set("Marker", marker)
		}
		resp := awsELBV2DescribeListenersResponse{}
		if err := awsQueryAPIWithValues(ctx, account, "elasticloadbalancing", region, values, &resp); err != nil {
			return listeners, err
		}
		listeners = append(listeners, resp.Result.Listeners...)
		if resp.Result.NextMarker == "" {
			return listeners, nil
		}
		marker = resp.Result.NextMarker
	}
}

func collectAwsV2LoadBalancerListenerRules(ctx context.Context, account *cmdbCloudAccount, region string, listeners []awsV2Listener) (map[string][]awsV2Rule, []models.ResAttrs) {
	rulesByListener := make(map[string][]awsV2Rule, len(listeners))
	errors := make([]models.ResAttrs, 0)
	for _, listener := range listeners {
		if listener.ListenerArn == "" {
			continue
		}
		rules, err := collectAwsV2ListenerRules(ctx, account, region, listener.ListenerArn)
		if err != nil {
			errors = append(errors, models.ResAttrs{
				"listenerArn": listener.ListenerArn,
				"port":        listener.Port,
				"protocol":    listener.Protocol,
				"error":       err.Error(),
			})
			continue
		}
		rulesByListener[listener.ListenerArn] = rules
	}
	return rulesByListener, errors
}

func collectAwsV2ListenerRules(ctx context.Context, account *cmdbCloudAccount, region, listenerArn string) ([]awsV2Rule, error) {
	rules := make([]awsV2Rule, 0)
	marker := ""
	for {
		values := url.Values{
			"Action":      []string{"DescribeRules"},
			"Version":     []string{awsELBV2APIVersion},
			"ListenerArn": []string{listenerArn},
		}
		if marker != "" {
			values.Set("Marker", marker)
		}
		resp := awsELBV2DescribeRulesResponse{}
		if err := awsQueryAPIWithValues(ctx, account, "elasticloadbalancing", region, values, &resp); err != nil {
			return rules, err
		}
		rules = append(rules, resp.Result.Rules...)
		if resp.Result.NextMarker == "" {
			return rules, nil
		}
		marker = resp.Result.NextMarker
	}
}

func collectAwsV2LoadBalancerTargetGroups(ctx context.Context, account *cmdbCloudAccount, region, loadBalancerArn string) ([]awsV2TargetGroup, error) {
	groups := make([]awsV2TargetGroup, 0)
	marker := ""
	for {
		values := url.Values{
			"Action":          []string{"DescribeTargetGroups"},
			"Version":         []string{awsELBV2APIVersion},
			"LoadBalancerArn": []string{loadBalancerArn},
		}
		if marker != "" {
			values.Set("Marker", marker)
		}
		resp := awsELBV2DescribeTargetGroupsResponse{}
		if err := awsQueryAPIWithValues(ctx, account, "elasticloadbalancing", region, values, &resp); err != nil {
			return groups, err
		}
		groups = append(groups, resp.Result.TargetGroups...)
		if resp.Result.NextMarker == "" {
			return groups, nil
		}
		marker = resp.Result.NextMarker
	}
}

func collectAwsV2TargetHealth(ctx context.Context, account *cmdbCloudAccount, region, targetGroupArn string) ([]awsV2TargetHealthDescription, error) {
	if targetGroupArn == "" {
		return nil, nil
	}
	values := url.Values{
		"Action":         []string{"DescribeTargetHealth"},
		"Version":        []string{awsELBV2APIVersion},
		"TargetGroupArn": []string{targetGroupArn},
	}
	resp := awsELBV2DescribeTargetHealthResponse{}
	if err := awsQueryAPIWithValues(ctx, account, "elasticloadbalancing", region, values, &resp); err != nil {
		return nil, err
	}
	return resp.Result.TargetHealthDescriptions, nil
}

type awsS3ListBucketsResponse struct {
	Buckets []awsS3Bucket `xml:"Buckets>Bucket"`
}

type awsS3Bucket struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

type awsS3BucketLocationResponse struct {
	Location string `xml:",chardata"`
}

type awsS3BucketTaggingResponse struct {
	Tags []awsS3BucketTag `xml:"TagSet>Tag"`
}

type awsS3BucketTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type awsS3BucketEncryptionResponse struct {
	Rules []awsS3BucketEncryptionRule `xml:"Rule"`
}

type awsS3BucketEncryptionRule struct {
	ApplyServerSideEncryptionByDefault struct {
		SSEAlgorithm   string `xml:"SSEAlgorithm"`
		KMSMasterKeyID string `xml:"KMSMasterKeyID"`
	} `xml:"ApplyServerSideEncryptionByDefault"`
	BucketKeyEnabled string `xml:"BucketKeyEnabled"`
}

type awsS3BucketVersioningResponse struct {
	Status    string `xml:"Status"`
	MFADelete string `xml:"MfaDelete"`
}

type awsS3PublicAccessBlockResponse struct {
	BlockPublicAcls       string `xml:"BlockPublicAcls"`
	IgnorePublicAcls      string `xml:"IgnorePublicAcls"`
	BlockPublicPolicy     string `xml:"BlockPublicPolicy"`
	RestrictPublicBuckets string `xml:"RestrictPublicBuckets"`
}

type awsS3BucketLifecycleResponse struct {
	Rules []awsS3BucketLifecycleRule `xml:"Rule"`
}

type awsS3BucketLifecycleRule struct {
	ID                             string                           `xml:"ID"`
	Status                         string                           `xml:"Status"`
	Prefix                         string                           `xml:"Prefix"`
	Filter                         awsS3BucketLifecycleFilter       `xml:"Filter"`
	Transitions                    []awsS3BucketLifecycleTransition `xml:"Transition"`
	NoncurrentVersionTransitions   []awsS3BucketLifecycleTransition `xml:"NoncurrentVersionTransition"`
	Expiration                     awsS3BucketLifecycleExpiration   `xml:"Expiration"`
	NoncurrentVersionExpiration    awsS3BucketLifecycleExpiration   `xml:"NoncurrentVersionExpiration"`
	AbortIncompleteMultipartUpload struct {
		DaysAfterInitiation string `xml:"DaysAfterInitiation"`
	} `xml:"AbortIncompleteMultipartUpload"`
}

type awsS3BucketLifecycleFilter struct {
	Prefix                string                  `xml:"Prefix"`
	Tag                   awsS3BucketTag          `xml:"Tag"`
	ObjectSizeGreaterThan string                  `xml:"ObjectSizeGreaterThan"`
	ObjectSizeLessThan    string                  `xml:"ObjectSizeLessThan"`
	And                   awsS3BucketLifecycleAnd `xml:"And"`
}

type awsS3BucketLifecycleAnd struct {
	Prefix                string           `xml:"Prefix"`
	Tags                  []awsS3BucketTag `xml:"Tag"`
	ObjectSizeGreaterThan string           `xml:"ObjectSizeGreaterThan"`
	ObjectSizeLessThan    string           `xml:"ObjectSizeLessThan"`
}

type awsS3BucketLifecycleTransition struct {
	Date           string `xml:"Date"`
	Days           string `xml:"Days"`
	NoncurrentDays string `xml:"NoncurrentDays"`
	StorageClass   string `xml:"StorageClass"`
}

type awsS3BucketLifecycleExpiration struct {
	Date                      string `xml:"Date"`
	Days                      string `xml:"Days"`
	ExpiredObjectDeleteMarker string `xml:"ExpiredObjectDeleteMarker"`
	NoncurrentDays            string `xml:"NoncurrentDays"`
	NewerNoncurrentVersions   string `xml:"NewerNoncurrentVersions"`
}

type awsS3BucketACLResponse struct {
	Owner  awsS3Owner      `xml:"Owner"`
	Grants []awsS3ACLGrant `xml:"AccessControlList>Grant"`
}

type awsS3Owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type awsS3ACLGrant struct {
	Grantee    awsS3ACLGrantee `xml:"Grantee"`
	Permission string          `xml:"Permission"`
}

type awsS3ACLGrantee struct {
	Type         string `xml:"type,attr"`
	ID           string `xml:"ID"`
	DisplayName  string `xml:"DisplayName"`
	EmailAddress string `xml:"EmailAddress"`
	URI          string `xml:"URI"`
}

type awsS3ObjectLockResponse struct {
	ObjectLockEnabled string `xml:"ObjectLockEnabled"`
	Rule              struct {
		DefaultRetention struct {
			Mode  string `xml:"Mode"`
			Days  string `xml:"Days"`
			Years string `xml:"Years"`
		} `xml:"DefaultRetention"`
	} `xml:"Rule"`
}

type awsS3BucketReplicationResponse struct {
	Role  string                 `xml:"Role"`
	Rules []awsS3ReplicationRule `xml:"Rule"`
}

type awsS3ReplicationRule struct {
	ID                        string                       `xml:"ID"`
	Status                    string                       `xml:"Status"`
	Priority                  string                       `xml:"Priority"`
	Prefix                    string                       `xml:"Prefix"`
	Filter                    awsS3BucketLifecycleFilter   `xml:"Filter"`
	DeleteMarkerReplication   awsS3StatusConfig            `xml:"DeleteMarkerReplication"`
	ExistingObjectReplication awsS3StatusConfig            `xml:"ExistingObjectReplication"`
	Destination               awsS3ReplicationDestination  `xml:"Destination"`
	SourceSelectionCriteria   awsS3SourceSelectionCriteria `xml:"SourceSelectionCriteria"`
}

type awsS3StatusConfig struct {
	Status string `xml:"Status"`
}

type awsS3ReplicationDestination struct {
	Account                  string `xml:"Account"`
	Bucket                   string `xml:"Bucket"`
	StorageClass             string `xml:"StorageClass"`
	AccessControlTranslation struct {
		Owner string `xml:"Owner"`
	} `xml:"AccessControlTranslation"`
	EncryptionConfiguration struct {
		ReplicaKmsKeyID string `xml:"ReplicaKmsKeyID"`
	} `xml:"EncryptionConfiguration"`
	Metrics struct {
		Status         string `xml:"Status"`
		EventThreshold struct {
			Minutes string `xml:"Minutes"`
		} `xml:"EventThreshold"`
	} `xml:"Metrics"`
	ReplicationTime struct {
		Status string `xml:"Status"`
		Time   struct {
			Minutes string `xml:"Minutes"`
		} `xml:"Time"`
	} `xml:"ReplicationTime"`
}

type awsS3SourceSelectionCriteria struct {
	ReplicaModifications struct {
		Status string `xml:"Status"`
	} `xml:"ReplicaModifications"`
	SseKmsEncryptedObjects struct {
		Status string `xml:"Status"`
	} `xml:"SseKmsEncryptedObjects"`
}

type awsS3BucketLoggingResponse struct {
	LoggingEnabled awsS3LoggingEnabled `xml:"LoggingEnabled"`
}

type awsS3LoggingEnabled struct {
	TargetBucket string          `xml:"TargetBucket"`
	TargetPrefix string          `xml:"TargetPrefix"`
	TargetGrants []awsS3ACLGrant `xml:"TargetGrants>Grant"`
}

type awsS3BucketNotificationResponse struct {
	TopicConfigurations          []awsS3NotificationTargetConfig `xml:"TopicConfiguration"`
	QueueConfigurations          []awsS3NotificationTargetConfig `xml:"QueueConfiguration"`
	LambdaFunctionConfigurations []awsS3NotificationTargetConfig `xml:"CloudFunctionConfiguration"`
	EventBridgeConfiguration     *struct{}                       `xml:"EventBridgeConfiguration"`
}

type awsS3NotificationTargetConfig struct {
	ID            string                  `xml:"Id"`
	Topic         string                  `xml:"Topic"`
	Queue         string                  `xml:"Queue"`
	CloudFunction string                  `xml:"CloudFunction"`
	Events        []string                `xml:"Event"`
	Filter        awsS3NotificationFilter `xml:"Filter"`
}

type awsS3NotificationFilter struct {
	S3Key struct {
		Rules []awsS3NotificationFilterRule `xml:"FilterRule"`
	} `xml:"S3Key"`
}

type awsS3NotificationFilterRule struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

type awsS3BucketWebsiteResponse struct {
	RedirectAllRequestsTo struct {
		HostName string `xml:"HostName"`
		Protocol string `xml:"Protocol"`
	} `xml:"RedirectAllRequestsTo"`
	IndexDocument struct {
		Suffix string `xml:"Suffix"`
	} `xml:"IndexDocument"`
	ErrorDocument struct {
		Key string `xml:"Key"`
	} `xml:"ErrorDocument"`
	RoutingRules []awsS3WebsiteRoutingRule `xml:"RoutingRules>RoutingRule"`
}

type awsS3WebsiteRoutingRule struct {
	Condition struct {
		HttpErrorCodeReturnedEquals string `xml:"HttpErrorCodeReturnedEquals"`
		KeyPrefixEquals             string `xml:"KeyPrefixEquals"`
	} `xml:"Condition"`
	Redirect struct {
		HostName             string `xml:"HostName"`
		HttpRedirectCode     string `xml:"HttpRedirectCode"`
		Protocol             string `xml:"Protocol"`
		ReplaceKeyPrefixWith string `xml:"ReplaceKeyPrefixWith"`
		ReplaceKeyWith       string `xml:"ReplaceKeyWith"`
	} `xml:"Redirect"`
}

type awsS3BucketInventoryResponse struct {
	ContinuationToken       string                        `xml:"ContinuationToken"`
	InventoryConfigurations []awsS3InventoryConfiguration `xml:"InventoryConfiguration"`
	IsTruncated             string                        `xml:"IsTruncated"`
	NextContinuationToken   string                        `xml:"NextContinuationToken"`
}

type awsS3InventoryConfiguration struct {
	ID                     string `xml:"Id"`
	IsEnabled              string `xml:"IsEnabled"`
	IncludedObjectVersions string `xml:"IncludedObjectVersions"`
	Filter                 struct {
		Prefix string `xml:"Prefix"`
	} `xml:"Filter"`
	Destination struct {
		S3BucketDestination struct {
			AccountID  string `xml:"AccountId"`
			Bucket     string `xml:"Bucket"`
			Format     string `xml:"Format"`
			Prefix     string `xml:"Prefix"`
			Encryption struct {
				SSEKMS struct {
					KeyID string `xml:"KeyId"`
				} `xml:"SSE-KMS"`
				SSES3 *struct{} `xml:"SSE-S3"`
			} `xml:"Encryption"`
		} `xml:"S3BucketDestination"`
	} `xml:"Destination"`
	OptionalFields []string `xml:"OptionalFields>Field"`
	Schedule       struct {
		Frequency string `xml:"Frequency"`
	} `xml:"Schedule"`
}

func collectAwsS3Buckets(ctx context.Context, account *cmdbCloudAccount, region string) ([]*models.CmdbAsset, error) {
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	resp := awsS3ListBucketsResponse{}
	if err := awsS3API(ctx, account, awsS3DefaultRegion, "s3.amazonaws.com", "/", nil, &resp); err != nil {
		return assets, err
	}
	for _, bucket := range resp.Buckets {
		bucketRegion, err := awsS3BucketRegion(ctx, account, bucket.Name)
		if err != nil {
			return assets, err
		}
		if !strings.EqualFold(bucketRegion, region) {
			continue
		}
		asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeObjectStorageBucket, "aws_s3_bucket", bucket.Name, bucket.Name, now)
		asset.Status = "available"
		asset.Attributes = models.ResAttrs{
			"name":         bucket.Name,
			"bucket":       bucket.Name,
			"creationDate": bucket.CreationDate,
			"location":     bucketRegion,
		}
		enrichAwsS3BucketRuntime(ctx, account, bucketRegion, bucket.Name, asset.Attributes)
		if tags, ok := asset.Attributes["bucketTags"].(models.ResAttrs); ok {
			asset.Tags = tags
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

func enrichAwsS3BucketRuntime(ctx context.Context, account *cmdbCloudAccount, region, bucket string, attrs models.ResAttrs) {
	if bucket == "" {
		return
	}
	tags, err := collectAwsS3BucketTags(ctx, account, region, bucket)
	if err != nil {
		if !awsS3BucketOptionalConfigMissing(err) {
			attrs["bucketTagsCollectError"] = err.Error()
		}
	} else {
		attrs["bucketTags"] = awsS3BucketTagsToAttrs(tags)
		attrs["bucketTagList"] = awsS3BucketTagListAttrs(tags)
		attrs["bucketTagCount"] = len(tags)
	}

	encryption, err := collectAwsS3BucketEncryption(ctx, account, region, bucket)
	if err != nil {
		if !awsS3BucketOptionalConfigMissing(err) {
			attrs["bucketEncryptionCollectError"] = err.Error()
		}
	} else {
		attrs["encryption"] = awsS3BucketEncryptionAttrs(encryption)
		attrs["encryptionEnabled"] = len(encryption.Rules) > 0
	}

	versioning, err := collectAwsS3BucketVersioning(ctx, account, region, bucket)
	if err != nil {
		attrs["bucketVersioningCollectError"] = err.Error()
	} else {
		attrs["versioning"] = awsS3BucketVersioningAttrs(versioning)
		attrs["versioningStatus"] = versioning.Status
		attrs["mfaDelete"] = versioning.MFADelete
	}

	publicAccess, err := collectAwsS3PublicAccessBlock(ctx, account, region, bucket)
	if err != nil {
		if !awsS3BucketOptionalConfigMissing(err) {
			attrs["publicAccessBlockCollectError"] = err.Error()
		}
	} else {
		attrs["publicAccessBlock"] = awsS3PublicAccessBlockAttrs(publicAccess)
		attrs["publicAccessBlockEnabled"] = awsS3PublicAccessBlockEnabled(publicAccess)
	}

	lifecycle, err := collectAwsS3BucketLifecycle(ctx, account, region, bucket)
	if err != nil {
		if !awsS3BucketOptionalConfigMissing(err) {
			attrs["lifecycleCollectError"] = err.Error()
		}
	} else {
		attrs["lifecycleRules"] = awsS3BucketLifecycleRuleAttrs(lifecycle.Rules)
		attrs["lifecycleRuleCount"] = len(lifecycle.Rules)
	}

	policy, err := collectAwsS3BucketPolicy(ctx, account, region, bucket)
	if err != nil {
		if !awsS3BucketOptionalConfigMissing(err) {
			attrs["bucketPolicyCollectError"] = err.Error()
		}
	} else {
		attrs["bucketPolicy"] = awsS3BucketPolicyAttrs(policy)
	}

	acl, err := collectAwsS3BucketACL(ctx, account, region, bucket)
	if err != nil {
		attrs["bucketACLCollectError"] = err.Error()
	} else {
		attrs["bucketAcl"] = awsS3BucketACLAttrs(acl)
		attrs["bucketAclGrantCount"] = len(acl.Grants)
		attrs["bucketAclPublic"] = awsS3BucketACLPublic(acl.Grants)
	}

	objectLock, err := collectAwsS3ObjectLock(ctx, account, region, bucket)
	if err != nil {
		if !awsS3BucketOptionalConfigMissing(err) {
			attrs["objectLockCollectError"] = err.Error()
		}
	} else {
		attrs["objectLock"] = awsS3ObjectLockAttrs(objectLock)
		attrs["objectLockEnabled"] = strings.EqualFold(objectLock.ObjectLockEnabled, "Enabled")
	}

	replication, err := collectAwsS3BucketReplication(ctx, account, region, bucket)
	if err != nil {
		if !awsS3BucketOptionalConfigMissing(err) {
			attrs["replicationCollectError"] = err.Error()
		}
	} else {
		attrs["replication"] = awsS3BucketReplicationAttrs(replication)
		attrs["replicationRuleCount"] = len(replication.Rules)
		attrs["replicationEnabledRuleCount"] = awsS3EnabledReplicationRuleCount(replication.Rules)
	}

	logging, err := collectAwsS3BucketLogging(ctx, account, region, bucket)
	if err != nil {
		attrs["loggingCollectError"] = err.Error()
	} else {
		attrs["logging"] = awsS3BucketLoggingAttrs(logging)
		attrs["loggingEnabled"] = strings.TrimSpace(logging.LoggingEnabled.TargetBucket) != ""
	}

	notification, err := collectAwsS3BucketNotification(ctx, account, region, bucket)
	if err != nil {
		attrs["notificationCollectError"] = err.Error()
	} else {
		attrs["notifications"] = awsS3BucketNotificationAttrs(notification)
		attrs["notificationRuleCount"] = awsS3BucketNotificationCount(notification)
		attrs["eventBridgeEnabled"] = notification.EventBridgeConfiguration != nil
	}

	website, err := collectAwsS3BucketWebsite(ctx, account, region, bucket)
	if err != nil {
		if !awsS3BucketOptionalConfigMissing(err) {
			attrs["websiteCollectError"] = err.Error()
		}
	} else {
		attrs["website"] = awsS3BucketWebsiteAttrs(website)
		attrs["websiteEnabled"] = awsS3BucketWebsiteEnabled(website)
	}

	inventories, err := collectAwsS3BucketInventories(ctx, account, region, bucket)
	if err != nil {
		if !awsS3BucketOptionalConfigMissing(err) {
			attrs["inventoryCollectError"] = err.Error()
		}
	} else {
		attrs["inventoryConfigurations"] = awsS3BucketInventoryAttrs(inventories)
		attrs["inventoryConfigurationCount"] = len(inventories)
		attrs["inventoryEnabledCount"] = awsS3BucketInventoryEnabledCount(inventories)
	}
}

func collectAwsS3BucketTags(ctx context.Context, account *cmdbCloudAccount, region, bucket string) ([]awsS3BucketTag, error) {
	resp := awsS3BucketTaggingResponse{}
	values := url.Values{"tagging": []string{""}}
	err := awsS3API(ctx, account, region, awsS3BucketHost(region), "/"+url.PathEscape(bucket), values, &resp)
	return resp.Tags, err
}

func collectAwsS3BucketEncryption(ctx context.Context, account *cmdbCloudAccount, region, bucket string) (awsS3BucketEncryptionResponse, error) {
	resp := awsS3BucketEncryptionResponse{}
	values := url.Values{"encryption": []string{""}}
	err := awsS3API(ctx, account, region, awsS3BucketHost(region), "/"+url.PathEscape(bucket), values, &resp)
	return resp, err
}

func collectAwsS3BucketVersioning(ctx context.Context, account *cmdbCloudAccount, region, bucket string) (awsS3BucketVersioningResponse, error) {
	resp := awsS3BucketVersioningResponse{}
	values := url.Values{"versioning": []string{""}}
	err := awsS3API(ctx, account, region, awsS3BucketHost(region), "/"+url.PathEscape(bucket), values, &resp)
	return resp, err
}

func collectAwsS3PublicAccessBlock(ctx context.Context, account *cmdbCloudAccount, region, bucket string) (awsS3PublicAccessBlockResponse, error) {
	resp := awsS3PublicAccessBlockResponse{}
	values := url.Values{"publicAccessBlock": []string{""}}
	err := awsS3API(ctx, account, region, awsS3BucketHost(region), "/"+url.PathEscape(bucket), values, &resp)
	return resp, err
}

func collectAwsS3BucketLifecycle(ctx context.Context, account *cmdbCloudAccount, region, bucket string) (awsS3BucketLifecycleResponse, error) {
	resp := awsS3BucketLifecycleResponse{}
	values := url.Values{"lifecycle": []string{""}}
	err := awsS3API(ctx, account, region, awsS3BucketHost(region), "/"+url.PathEscape(bucket), values, &resp)
	return resp, err
}

func collectAwsS3BucketPolicy(ctx context.Context, account *cmdbCloudAccount, region, bucket string) (map[string]interface{}, error) {
	values := url.Values{"policy": []string{""}}
	body, err := awsS3RawAPI(ctx, account, region, awsS3BucketHost(region), "/"+url.PathEscape(bucket), values)
	if err != nil {
		return nil, err
	}
	document := map[string]interface{}{}
	if err := json.Unmarshal(body, &document); err != nil {
		return nil, err
	}
	return document, nil
}

func collectAwsS3BucketACL(ctx context.Context, account *cmdbCloudAccount, region, bucket string) (awsS3BucketACLResponse, error) {
	resp := awsS3BucketACLResponse{}
	values := url.Values{"acl": []string{""}}
	err := awsS3API(ctx, account, region, awsS3BucketHost(region), "/"+url.PathEscape(bucket), values, &resp)
	return resp, err
}

func collectAwsS3ObjectLock(ctx context.Context, account *cmdbCloudAccount, region, bucket string) (awsS3ObjectLockResponse, error) {
	resp := awsS3ObjectLockResponse{}
	values := url.Values{"object-lock": []string{""}}
	err := awsS3API(ctx, account, region, awsS3BucketHost(region), "/"+url.PathEscape(bucket), values, &resp)
	return resp, err
}

func collectAwsS3BucketReplication(ctx context.Context, account *cmdbCloudAccount, region, bucket string) (awsS3BucketReplicationResponse, error) {
	resp := awsS3BucketReplicationResponse{}
	values := url.Values{"replication": []string{""}}
	err := awsS3API(ctx, account, region, awsS3BucketHost(region), "/"+url.PathEscape(bucket), values, &resp)
	return resp, err
}

func collectAwsS3BucketLogging(ctx context.Context, account *cmdbCloudAccount, region, bucket string) (awsS3BucketLoggingResponse, error) {
	resp := awsS3BucketLoggingResponse{}
	values := url.Values{"logging": []string{""}}
	err := awsS3API(ctx, account, region, awsS3BucketHost(region), "/"+url.PathEscape(bucket), values, &resp)
	return resp, err
}

func collectAwsS3BucketNotification(ctx context.Context, account *cmdbCloudAccount, region, bucket string) (awsS3BucketNotificationResponse, error) {
	resp := awsS3BucketNotificationResponse{}
	values := url.Values{"notification": []string{""}}
	err := awsS3API(ctx, account, region, awsS3BucketHost(region), "/"+url.PathEscape(bucket), values, &resp)
	return resp, err
}

func collectAwsS3BucketWebsite(ctx context.Context, account *cmdbCloudAccount, region, bucket string) (awsS3BucketWebsiteResponse, error) {
	resp := awsS3BucketWebsiteResponse{}
	values := url.Values{"website": []string{""}}
	err := awsS3API(ctx, account, region, awsS3BucketHost(region), "/"+url.PathEscape(bucket), values, &resp)
	return resp, err
}

func collectAwsS3BucketInventories(ctx context.Context, account *cmdbCloudAccount, region, bucket string) ([]awsS3InventoryConfiguration, error) {
	configs := make([]awsS3InventoryConfiguration, 0)
	values := url.Values{"inventory": []string{""}}
	for {
		resp := awsS3BucketInventoryResponse{}
		err := awsS3API(ctx, account, region, awsS3BucketHost(region), "/"+url.PathEscape(bucket), values, &resp)
		if err != nil {
			return configs, err
		}
		configs = append(configs, resp.InventoryConfigurations...)
		if !strings.EqualFold(resp.IsTruncated, "true") || strings.TrimSpace(resp.NextContinuationToken) == "" {
			break
		}
		values.Set("continuation-token", resp.NextContinuationToken)
	}
	return configs, nil
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

type awsEKSListNodegroupsResponse struct {
	Nodegroups []string `json:"nodegroups"`
	NextToken  string   `json:"nextToken"`
}

type awsEKSDescribeNodegroupResponse struct {
	Nodegroup awsEKSNodegroup `json:"nodegroup"`
}

type awsEKSNodegroup struct {
	NodegroupName  string            `json:"nodegroupName"`
	NodegroupArn   string            `json:"nodegroupArn"`
	ClusterName    string            `json:"clusterName"`
	Version        string            `json:"version"`
	ReleaseVersion string            `json:"releaseVersion"`
	Status         string            `json:"status"`
	CapacityType   string            `json:"capacityType"`
	InstanceTypes  []string          `json:"instanceTypes"`
	Subnets        []string          `json:"subnets"`
	AmiType        string            `json:"amiType"`
	NodeRole       string            `json:"nodeRole"`
	ScalingConfig  models.ResAttrs   `json:"scalingConfig"`
	RemoteAccess   models.ResAttrs   `json:"remoteAccess"`
	Labels         models.ResAttrs   `json:"labels"`
	Taints         []models.ResAttrs `json:"taints"`
	Tags           map[string]string `json:"tags"`
	CreatedAt      string            `json:"createdAt"`
	ModifiedAt     string            `json:"modifiedAt"`
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
			nodegroups, nodegroupErr := collectAwsEksNodegroups(ctx, account, region, cluster.Name)
			if nodegroupErr != nil {
				asset.Attributes["nodeGroupCollectError"] = nodegroupErr.Error()
			} else {
				asset.Attributes["nodeGroups"] = nodegroups
				asset.Attributes["nodeGroupCount"] = len(nodegroups)
			}
			assets = append(assets, asset)
		}
		if resp.NextToken == "" {
			return assets, nil
		}
		nextToken = resp.NextToken
	}
}

func collectAwsEksNodegroups(ctx context.Context, account *cmdbCloudAccount, region, clusterName string) ([]models.ResAttrs, error) {
	nodegroups := make([]models.ResAttrs, 0)
	nextToken := ""
	for {
		values := url.Values{"maxResults": []string{"100"}}
		if nextToken != "" {
			values.Set("nextToken", nextToken)
		}
		resp := awsEKSListNodegroupsResponse{}
		if err := awsJSONAPI(ctx, account, "eks", region, "/clusters/"+url.PathEscape(clusterName)+"/node-groups", values, &resp); err != nil {
			return nodegroups, err
		}
		for _, nodegroupName := range resp.Nodegroups {
			detail := awsEKSDescribeNodegroupResponse{}
			if err := awsJSONAPI(ctx, account, "eks", region, "/clusters/"+url.PathEscape(clusterName)+"/node-groups/"+url.PathEscape(nodegroupName), nil, &detail); err != nil {
				return nodegroups, err
			}
			ng := detail.Nodegroup
			nodegroups = append(nodegroups, models.ResAttrs{
				"name":           ng.NodegroupName,
				"arn":            ng.NodegroupArn,
				"status":         ng.Status,
				"version":        ng.Version,
				"releaseVersion": ng.ReleaseVersion,
				"capacityType":   ng.CapacityType,
				"instanceTypes":  ng.InstanceTypes,
				"subnets":        ng.Subnets,
				"amiType":        ng.AmiType,
				"nodeRole":       ng.NodeRole,
				"scalingConfig":  ng.ScalingConfig,
				"remoteAccess":   ng.RemoteAccess,
				"labels":         ng.Labels,
				"taints":         ng.Taints,
				"tags":           stringMapToResAttrs(ng.Tags),
				"createdAt":      ng.CreatedAt,
				"modifiedAt":     ng.ModifiedAt,
			})
		}
		if resp.NextToken == "" {
			return nodegroups, nil
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

func awsS3API(ctx context.Context, account *cmdbCloudAccount, signingRegion, host, path string, values url.Values, out interface{}) error {
	respBody, err := awsS3RawAPI(ctx, account, signingRegion, host, path, values)
	if err != nil {
		return err
	}
	return xml.Unmarshal(respBody, out)
}

func awsS3RawAPI(ctx context.Context, account *cmdbCloudAccount, signingRegion, host, path string, values url.Values) ([]byte, error) {
	u := url.URL{
		Scheme:   "https",
		Host:     host,
		Path:     path,
		RawQuery: values.Encode(),
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if err := signAWSV4(req, nil, account, "s3", signingRegion, time.Now().UTC()); err != nil {
		return nil, err
	}
	return doAWSRequest(req)
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

func awsSecurityGroupPermissionRules(permissions []awsEC2SecurityGroupPermission, direction string) []models.ResAttrs {
	rules := make([]models.ResAttrs, 0)
	for _, permission := range permissions {
		base := models.ResAttrs{
			"direction":  direction,
			"protocol":   permission.IpProtocol,
			"fromPort":   permission.FromPort,
			"toPort":     permission.ToPort,
			"portRange":  awsSecurityGroupPortRange(permission),
			"permission": permission.IpProtocol,
		}
		for _, item := range permission.IpRanges {
			rules = append(rules, awsSecurityGroupRuleWithTarget(base, direction, item.CidrIp, item.Description))
		}
		for _, item := range permission.Ipv6Ranges {
			rules = append(rules, awsSecurityGroupRuleWithTarget(base, direction, item.CidrIpv6, item.Description))
		}
		for _, item := range permission.PrefixListIds {
			rules = append(rules, awsSecurityGroupRuleWithTarget(base, direction, item.PrefixListId, item.Description))
		}
		for _, item := range permission.UserIdGroupPairs {
			target := firstNonEmpty(item.GroupId, item.GroupName, item.UserId)
			rules = append(rules, awsSecurityGroupRuleWithTarget(base, direction, target, item.Description))
		}
		if len(permission.IpRanges) == 0 && len(permission.Ipv6Ranges) == 0 &&
			len(permission.PrefixListIds) == 0 && len(permission.UserIdGroupPairs) == 0 {
			rules = append(rules, awsSecurityGroupRuleWithTarget(base, direction, "", ""))
		}
	}
	return rules
}

func awsSecurityGroupRuleWithTarget(base models.ResAttrs, direction, target, description string) models.ResAttrs {
	rule := models.ResAttrs{}
	for key, value := range base {
		rule[key] = value
	}
	if direction == "egress" {
		rule["destination"] = target
	} else {
		rule["source"] = target
	}
	if description != "" {
		rule["description"] = description
	}
	return rule
}

func awsSecurityGroupPortRange(permission awsEC2SecurityGroupPermission) string {
	fromPort := strings.TrimSpace(permission.FromPort)
	toPort := strings.TrimSpace(permission.ToPort)
	if fromPort == "" && toPort == "" {
		return "all"
	}
	if toPort == "" || fromPort == toPort {
		return fromPort
	}
	return fmt.Sprintf("%s-%s", fromPort, toPort)
}

func awsRouteTableAssociationRefs(associations []awsEC2RouteTableAssoc) ([]string, []string) {
	subnetIds := make([]string, 0)
	gatewayIds := make([]string, 0)
	for _, assoc := range associations {
		if assoc.SubnetId != "" {
			subnetIds = append(subnetIds, assoc.SubnetId)
		}
		if assoc.GatewayId != "" {
			gatewayIds = append(gatewayIds, assoc.GatewayId)
		}
	}
	return dedupeStrings(subnetIds), dedupeStrings(gatewayIds)
}

func awsRouteRefs(routes []awsEC2Route) ([]string, []string) {
	natGatewayIds := make([]string, 0)
	internetGatewayIds := make([]string, 0)
	for _, route := range routes {
		if route.NatGatewayId != "" {
			natGatewayIds = append(natGatewayIds, route.NatGatewayId)
		}
		for _, gatewayId := range []string{route.GatewayId, route.EgressOnlyInternetGatewayId} {
			if strings.HasPrefix(gatewayId, "igw-") || strings.HasPrefix(gatewayId, "eigw-") {
				internetGatewayIds = append(internetGatewayIds, gatewayId)
			}
		}
	}
	return dedupeStrings(natGatewayIds), dedupeStrings(internetGatewayIds)
}

func awsRouteTableAssociations(associations []awsEC2RouteTableAssoc) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(associations))
	for _, assoc := range associations {
		result = append(result, models.ResAttrs{
			"routeTableAssociationId": assoc.RouteTableAssociationId,
			"routeTableId":            assoc.RouteTableId,
			"subnetId":                assoc.SubnetId,
			"gatewayId":               assoc.GatewayId,
			"main":                    assoc.Main,
			"state":                   assoc.AssociationState.State,
		})
	}
	return result
}

func awsRoutes(routes []awsEC2Route) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(routes))
	for _, route := range routes {
		result = append(result, models.ResAttrs{
			"destinationCidrBlock":        route.DestinationCidrBlock,
			"destinationIpv6CidrBlock":    route.DestinationIpv6CidrBlock,
			"destinationPrefixListId":     route.DestinationPrefixListId,
			"gatewayId":                   route.GatewayId,
			"natGatewayId":                route.NatGatewayId,
			"instanceId":                  route.InstanceId,
			"networkInterfaceId":          route.NetworkInterfaceId,
			"transitGatewayId":            route.TransitGatewayId,
			"vpcPeeringConnectionId":      route.VpcPeeringConnectionId,
			"egressOnlyInternetGatewayId": route.EgressOnlyInternetGatewayId,
			"carrierGatewayId":            route.CarrierGatewayId,
			"localGatewayId":              route.LocalGatewayId,
			"state":                       route.State,
			"origin":                      route.Origin,
		})
	}
	return result
}

func awsPropagatingGateways(gateways []awsEC2PropagatingGateway) []string {
	result := make([]string, 0, len(gateways))
	for _, gateway := range gateways {
		if gateway.GatewayId != "" {
			result = append(result, gateway.GatewayId)
		}
	}
	return dedupeStrings(result)
}

func awsNatGatewayPublicRefs(addresses []awsEC2NatGatewayAddress) ([]string, []string) {
	publicIps := make([]string, 0)
	allocationIds := make([]string, 0)
	for _, address := range addresses {
		if address.PublicIp != "" {
			publicIps = append(publicIps, address.PublicIp)
		}
		if address.AllocationId != "" {
			allocationIds = append(allocationIds, address.AllocationId)
		}
	}
	return dedupeStrings(publicIps), dedupeStrings(allocationIds)
}

func awsNatGatewayAddresses(addresses []awsEC2NatGatewayAddress) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(addresses))
	for _, address := range addresses {
		result = append(result, models.ResAttrs{
			"allocationId":       address.AllocationId,
			"associationId":      address.AssociationId,
			"publicIp":           address.PublicIp,
			"privateIp":          address.PrivateIp,
			"networkInterfaceId": address.NetworkInterfaceId,
		})
	}
	return result
}

func awsInternetGatewayStatus(attachments []awsEC2InternetGatewayAttach) string {
	if len(attachments) == 0 {
		return "detached"
	}
	for _, attachment := range attachments {
		if attachment.State == "available" {
			return "available"
		}
	}
	return attachments[0].State
}

func awsInternetGatewayVpcIds(attachments []awsEC2InternetGatewayAttach) []string {
	vpcIds := make([]string, 0, len(attachments))
	for _, attachment := range attachments {
		if attachment.VpcId != "" {
			vpcIds = append(vpcIds, attachment.VpcId)
		}
	}
	return dedupeStrings(vpcIds)
}

func awsInternetGatewayAttachments(attachments []awsEC2InternetGatewayAttach) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(attachments))
	for _, attachment := range attachments {
		result = append(result, models.ResAttrs{
			"vpcId": attachment.VpcId,
			"state": attachment.State,
		})
	}
	return result
}

func awsAddressStatus(address awsEC2Address) string {
	if address.AssociationId != "" || address.InstanceId != "" || address.NetworkInterfaceId != "" {
		return "associated"
	}
	return "available"
}

func awsClassicLoadBalancerInstanceIds(instances []struct {
	InstanceId string `xml:"InstanceId"`
}) []string {
	result := make([]string, 0, len(instances))
	for _, instance := range instances {
		if instance.InstanceId != "" {
			result = append(result, instance.InstanceId)
		}
	}
	return dedupeStrings(result)
}

func awsV2LoadBalancerNetworkRefs(zones []awsV2LoadBalancerAZ) ([]string, []string, []models.ResAttrs, []string) {
	subnetIds := make([]string, 0)
	zoneNames := make([]string, 0)
	addresses := make([]models.ResAttrs, 0)
	publicIpIds := make([]string, 0)
	for _, zone := range zones {
		if zone.SubnetId != "" {
			subnetIds = append(subnetIds, zone.SubnetId)
		}
		if zone.ZoneName != "" {
			zoneNames = append(zoneNames, zone.ZoneName)
		}
		for _, address := range zone.LoadBalancerAddresses {
			if address.AllocationId != "" {
				publicIpIds = append(publicIpIds, address.AllocationId)
			}
			addresses = append(addresses, models.ResAttrs{
				"zoneName":           zone.ZoneName,
				"subnetId":           zone.SubnetId,
				"ipAddress":          address.IpAddress,
				"allocationId":       address.AllocationId,
				"privateIPv4Address": address.PrivateIPv4Address,
				"ipv6Address":        address.IPv6Address,
			})
		}
	}
	return dedupeStrings(subnetIds), dedupeStrings(zoneNames), addresses, dedupeStrings(publicIpIds)
}

func awsV2LoadBalancerPublicIps(addresses []models.ResAttrs) []string {
	result := make([]string, 0)
	for _, address := range addresses {
		if value := attrString(address, "ipAddress"); value != "" {
			result = append(result, value)
		}
	}
	return dedupeStrings(result)
}

func awsV2ListenerAttrs(listeners []awsV2Listener, rulesByListenerOpt ...map[string][]awsV2Rule) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(listeners))
	rulesByListener := map[string][]awsV2Rule{}
	if len(rulesByListenerOpt) > 0 && rulesByListenerOpt[0] != nil {
		rulesByListener = rulesByListenerOpt[0]
	}
	for _, listener := range listeners {
		item := models.ResAttrs{
			"listenerArn":     listener.ListenerArn,
			"loadBalancerArn": listener.LoadBalancerArn,
			"port":            listener.Port,
			"protocol":        listener.Protocol,
			"sslPolicy":       listener.SslPolicy,
			"certificates":    awsV2ListenerCertificates(listener),
			"defaultActions":  awsV2ListenerActions(listener.DefaultActions),
			"targetGroupArns": awsV2ListenerActionTargetGroupArns(listener.DefaultActions),
		}
		if rules, ok := rulesByListener[listener.ListenerArn]; ok {
			item["rules"] = awsV2RuleAttrs(rules)
			item["ruleCount"] = len(rules)
			item["ruleTargetGroupArns"] = awsV2RuleTargetGroupArns(rules)
		}
		result = append(result, item)
	}
	return result
}

func awsV2ListenerCertificates(listener awsV2Listener) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(listener.Certificates))
	for _, cert := range listener.Certificates {
		result = append(result, models.ResAttrs{
			"certificateArn": cert.CertificateArn,
			"default":        strings.EqualFold(cert.IsDefault, "true"),
		})
	}
	return result
}

func awsV2ListenerActions(actions []awsV2ListenerAction) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(actions))
	for _, action := range actions {
		item := models.ResAttrs{
			"type":           action.Type,
			"targetGroupArn": action.TargetGroupArn,
			"order":          action.Order,
		}
		if len(action.ForwardConfig.TargetGroups) > 0 {
			groups := make([]models.ResAttrs, 0, len(action.ForwardConfig.TargetGroups))
			for _, group := range action.ForwardConfig.TargetGroups {
				groups = append(groups, models.ResAttrs{
					"targetGroupArn": group.TargetGroupArn,
					"weight":         group.Weight,
				})
			}
			item["forwardTargetGroups"] = groups
		}
		if action.RedirectConfig.StatusCode != "" {
			item["redirect"] = models.ResAttrs{
				"protocol":   action.RedirectConfig.Protocol,
				"port":       action.RedirectConfig.Port,
				"host":       action.RedirectConfig.Host,
				"path":       action.RedirectConfig.Path,
				"query":      action.RedirectConfig.Query,
				"statusCode": action.RedirectConfig.StatusCode,
			}
		}
		if action.FixedResponseConfig.StatusCode != "" {
			item["fixedResponse"] = models.ResAttrs{
				"messageBody": action.FixedResponseConfig.MessageBody,
				"statusCode":  action.FixedResponseConfig.StatusCode,
				"contentType": action.FixedResponseConfig.ContentType,
			}
		}
		if awsV2HasAuthenticateCognitoConfig(action) {
			item["authenticateCognito"] = awsV2AuthenticateCognitoAttrs(action.AuthenticateCognitoConfig)
		}
		if awsV2HasAuthenticateOidcConfig(action) {
			item["authenticateOidc"] = awsV2AuthenticateOidcAttrs(action.AuthenticateOidcConfig)
		}
		if awsV2HasJwtValidationConfig(action) {
			item["jwtValidation"] = awsV2JwtValidationAttrs(action.JwtValidationConfig)
		}
		result = append(result, item)
	}
	return result
}

func awsV2HasAuthenticateCognitoConfig(action awsV2ListenerAction) bool {
	return strings.EqualFold(action.Type, "authenticate-cognito") ||
		action.AuthenticateCognitoConfig.UserPoolArn != "" ||
		action.AuthenticateCognitoConfig.UserPoolClientId != "" ||
		action.AuthenticateCognitoConfig.UserPoolDomain != ""
}

func awsV2AuthenticateCognitoAttrs(config awsV2AuthenticateCognitoConfig) models.ResAttrs {
	return models.ResAttrs{
		"userPoolArn":                      config.UserPoolArn,
		"userPoolClientId":                 config.UserPoolClientId,
		"userPoolDomain":                   config.UserPoolDomain,
		"sessionCookieName":                config.SessionCookieName,
		"scope":                            config.Scope,
		"sessionTimeout":                   config.SessionTimeout,
		"authenticationRequestExtraParams": awsV2AuthExtraParamAttrs(config.AuthenticationRequestExtraParams),
		"onUnauthenticatedRequest":         config.OnUnauthenticatedRequest,
	}
}

func awsV2HasAuthenticateOidcConfig(action awsV2ListenerAction) bool {
	return strings.EqualFold(action.Type, "authenticate-oidc") ||
		action.AuthenticateOidcConfig.Issuer != "" ||
		action.AuthenticateOidcConfig.AuthorizationEndpoint != "" ||
		action.AuthenticateOidcConfig.ClientId != ""
}

func awsV2AuthenticateOidcAttrs(config awsV2AuthenticateOidcConfig) models.ResAttrs {
	attrs := models.ResAttrs{
		"issuer":                           config.Issuer,
		"authorizationEndpoint":            config.AuthorizationEndpoint,
		"tokenEndpoint":                    config.TokenEndpoint,
		"userInfoEndpoint":                 config.UserInfoEndpoint,
		"clientId":                         config.ClientId,
		"sessionCookieName":                config.SessionCookieName,
		"scope":                            config.Scope,
		"sessionTimeout":                   config.SessionTimeout,
		"authenticationRequestExtraParams": awsV2AuthExtraParamAttrs(config.AuthenticationRequestExtraParams),
		"onUnauthenticatedRequest":         config.OnUnauthenticatedRequest,
		"useExistingClientSecret":          strings.EqualFold(config.UseExistingClientSecret, "true"),
		"clientSecretConfigured":           config.ClientSecret != "",
	}
	if config.ClientSecret != "" {
		attrs["clientSecret"] = cmdbAssetMaskedValue
	}
	return attrs
}

func awsV2AuthExtraParamAttrs(params []awsV2AuthExtraParam) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(params))
	for _, param := range params {
		result = append(result, models.ResAttrs{
			"key":   param.Key,
			"value": param.Value,
		})
	}
	return result
}

func awsV2HasJwtValidationConfig(action awsV2ListenerAction) bool {
	return strings.EqualFold(action.Type, "jwt-validation") ||
		action.JwtValidationConfig.Issuer != "" ||
		action.JwtValidationConfig.JwksEndpoint != "" ||
		len(action.JwtValidationConfig.AdditionalClaims) > 0
}

func awsV2JwtValidationAttrs(config awsV2JwtValidationConfig) models.ResAttrs {
	return models.ResAttrs{
		"issuer":           config.Issuer,
		"jwksEndpoint":     config.JwksEndpoint,
		"additionalClaims": awsV2JwtValidationClaimAttrs(config.AdditionalClaims),
	}
}

func awsV2JwtValidationClaimAttrs(claims []awsV2JwtValidationClaimConfig) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(claims))
	for _, claim := range claims {
		result = append(result, models.ResAttrs{
			"name":   claim.Name,
			"format": claim.Format,
			"values": dedupeStrings(claim.Values),
		})
	}
	return result
}

func awsV2ListenerPorts(listeners []awsV2Listener) []string {
	result := make([]string, 0, len(listeners))
	for _, listener := range listeners {
		if listener.Port != "" {
			result = append(result, listener.Port)
		}
	}
	return dedupeStrings(result)
}

func awsV2ListenerProtocols(listeners []awsV2Listener) []string {
	result := make([]string, 0, len(listeners))
	for _, listener := range listeners {
		if listener.Protocol != "" {
			result = append(result, listener.Protocol)
		}
	}
	return dedupeStrings(result)
}

func awsV2ListenerTargetGroupArns(listeners []awsV2Listener) []string {
	result := make([]string, 0)
	for _, listener := range listeners {
		result = append(result, awsV2ListenerActionTargetGroupArns(listener.DefaultActions)...)
	}
	return dedupeStrings(result)
}

func awsV2ListenerActionTargetGroupArns(actions []awsV2ListenerAction) []string {
	result := make([]string, 0, len(actions))
	for _, action := range actions {
		if action.TargetGroupArn != "" {
			result = append(result, action.TargetGroupArn)
		}
		for _, group := range action.ForwardConfig.TargetGroups {
			if group.TargetGroupArn != "" {
				result = append(result, group.TargetGroupArn)
			}
		}
	}
	return dedupeStrings(result)
}

func awsV2FlattenRules(rulesByListener map[string][]awsV2Rule) []awsV2Rule {
	result := make([]awsV2Rule, 0)
	for _, rules := range rulesByListener {
		result = append(result, rules...)
	}
	return result
}

func awsV2RuleAttrs(rules []awsV2Rule) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(rules))
	for _, rule := range rules {
		result = append(result, models.ResAttrs{
			"ruleArn":         rule.RuleArn,
			"priority":        rule.Priority,
			"default":         strings.EqualFold(rule.IsDefault, "true"),
			"conditions":      awsV2RuleConditionAttrs(rule.Conditions),
			"actions":         awsV2ListenerActions(rule.Actions),
			"targetGroupArns": awsV2ListenerActionTargetGroupArns(rule.Actions),
		})
	}
	return result
}

func awsV2RuleConditionAttrs(conditions []awsV2RuleCondition) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(conditions))
	for _, condition := range conditions {
		item := models.ResAttrs{
			"field":  condition.Field,
			"values": dedupeStrings(condition.Values),
		}
		if values := dedupeStrings(condition.HostHeaderConfig.Values); len(values) > 0 {
			item["hostHeaderValues"] = values
		}
		if values := dedupeStrings(condition.PathPatternConfig.Values); len(values) > 0 {
			item["pathPatternValues"] = values
		}
		if condition.HttpHeaderConfig.HttpHeaderName != "" || len(condition.HttpHeaderConfig.Values) > 0 {
			item["httpHeaderName"] = condition.HttpHeaderConfig.HttpHeaderName
			item["httpHeaderValues"] = dedupeStrings(condition.HttpHeaderConfig.Values)
		}
		if len(condition.QueryStringConfig.Values) > 0 {
			item["queryStringValues"] = awsV2RuleQueryStringAttrs(condition.QueryStringConfig.Values)
		}
		if values := dedupeStrings(condition.HttpRequestMethodConfig.Values); len(values) > 0 {
			item["httpRequestMethods"] = values
		}
		if values := dedupeStrings(condition.SourceIpConfig.Values); len(values) > 0 {
			item["sourceIpValues"] = values
		}
		result = append(result, item)
	}
	return result
}

func awsV2RuleQueryStringAttrs(values []awsV2RuleQueryStringValue) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(values))
	for _, value := range values {
		result = append(result, models.ResAttrs{
			"key":   value.Key,
			"value": value.Value,
		})
	}
	return result
}

func awsV2RuleTargetGroupArns(rules []awsV2Rule) []string {
	result := make([]string, 0)
	for _, rule := range rules {
		result = append(result, awsV2ListenerActionTargetGroupArns(rule.Actions)...)
	}
	return dedupeStrings(result)
}

func awsV2TargetGroupAttrs(group awsV2TargetGroup, targets []awsV2TargetHealthDescription) models.ResAttrs {
	targetIds, instanceIds, ipTargets, healthy, unhealthy := awsV2TargetHealthRefs(targets)
	return models.ResAttrs{
		"targetGroupArn":             group.TargetGroupArn,
		"targetGroupName":            group.TargetGroupName,
		"protocol":                   group.Protocol,
		"protocolVersion":            group.ProtocolVersion,
		"port":                       group.Port,
		"vpcId":                      group.VpcId,
		"targetType":                 group.TargetType,
		"ipAddressType":              group.IpAddressType,
		"healthCheckEnabled":         strings.EqualFold(group.HealthCheckEnabled, "true"),
		"healthCheckProtocol":        group.HealthCheckProtocol,
		"healthCheckPort":            group.HealthCheckPort,
		"healthCheckPath":            group.HealthCheckPath,
		"healthCheckIntervalSeconds": group.HealthCheckIntervalSeconds,
		"healthCheckTimeoutSeconds":  group.HealthCheckTimeoutSeconds,
		"healthyThresholdCount":      group.HealthyThresholdCount,
		"unhealthyThresholdCount":    group.UnhealthyThresholdCount,
		"matcher": models.ResAttrs{
			"httpCode": group.Matcher.HttpCode,
			"grpcCode": group.Matcher.GrpcCode,
		},
		"loadBalancerArns":     group.LoadBalancerArns,
		"targets":              awsV2TargetHealthAttrs(targets),
		"targetIds":            targetIds,
		"targetInstanceIds":    instanceIds,
		"targetIpAddresses":    ipTargets,
		"healthyTargetCount":   healthy,
		"unhealthyTargetCount": unhealthy,
	}
}

func awsV2TargetHealthAttrs(targets []awsV2TargetHealthDescription) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(targets))
	for _, target := range targets {
		result = append(result, models.ResAttrs{
			"id":               target.Target.Id,
			"port":             target.Target.Port,
			"availabilityZone": target.Target.AvailabilityZone,
			"healthCheckPort":  target.HealthCheckPort,
			"state":            target.TargetHealth.State,
			"reason":           target.TargetHealth.Reason,
			"description":      target.TargetHealth.Description,
		})
	}
	return result
}

func awsV2TargetHealthRefs(targets []awsV2TargetHealthDescription) ([]string, []string, []string, int, int) {
	targetIds := make([]string, 0, len(targets))
	instanceIds := make([]string, 0)
	ipTargets := make([]string, 0)
	healthy := 0
	unhealthy := 0
	for _, target := range targets {
		id := strings.TrimSpace(target.Target.Id)
		if id != "" {
			targetIds = append(targetIds, id)
			if strings.HasPrefix(id, "i-") {
				instanceIds = append(instanceIds, id)
			} else if strings.Count(id, ".") == 3 || strings.Contains(id, ":") {
				ipTargets = append(ipTargets, id)
			}
		}
		if strings.EqualFold(target.TargetHealth.State, "healthy") {
			healthy++
		} else if target.TargetHealth.State != "" {
			unhealthy++
		}
	}
	return dedupeStrings(targetIds), dedupeStrings(instanceIds), dedupeStrings(ipTargets), healthy, unhealthy
}

func awsS3BucketRegion(ctx context.Context, account *cmdbCloudAccount, bucket string) (string, error) {
	location := awsS3BucketLocationResponse{}
	values := url.Values{"location": []string{""}}
	err := awsS3API(ctx, account, awsS3DefaultRegion, "s3.amazonaws.com", "/"+url.PathEscape(bucket), values, &location)
	if err != nil {
		return "", err
	}
	region := strings.TrimSpace(location.Location)
	switch region {
	case "":
		return awsS3DefaultRegion, nil
	case "EU":
		return "eu-west-1", nil
	default:
		return region, nil
	}
}

func awsS3BucketHost(region string) string {
	if region == "" || strings.EqualFold(region, awsS3DefaultRegion) {
		return "s3.amazonaws.com"
	}
	return fmt.Sprintf("s3.%s.amazonaws.com", region)
}

func awsS3BucketOptionalConfigMissing(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	missingCodes := []string{
		"NoSuchTagSet",
		"ServerSideEncryptionConfigurationNotFoundError",
		"NoSuchPublicAccessBlockConfiguration",
		"NoSuchLifecycleConfiguration",
		"NoSuchBucketPolicy",
		"NoSuchBucketPolicyException",
		"NoSuchObjectLockConfiguration",
		"ObjectLockConfigurationNotFoundError",
		"ReplicationConfigurationNotFoundError",
		"NoSuchReplicationConfiguration",
		"NoSuchBucketWebsite",
		"NoSuchWebsiteConfiguration",
	}
	for _, code := range missingCodes {
		if strings.Contains(message, code) {
			return true
		}
	}
	return false
}

func awsS3BucketTagsToAttrs(tags []awsS3BucketTag) models.ResAttrs {
	result := models.ResAttrs{}
	for _, tag := range tags {
		key := strings.TrimSpace(tag.Key)
		if key == "" {
			continue
		}
		result[key] = tag.Value
	}
	return result
}

func awsS3BucketTagListAttrs(tags []awsS3BucketTag) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(tags))
	for _, tag := range tags {
		result = append(result, models.ResAttrs{
			"key":   tag.Key,
			"value": tag.Value,
		})
	}
	return result
}

func awsS3BucketEncryptionAttrs(encryption awsS3BucketEncryptionResponse) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(encryption.Rules))
	for _, rule := range encryption.Rules {
		result = append(result, models.ResAttrs{
			"sseAlgorithm":     rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm,
			"kmsMasterKeyId":   rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID,
			"bucketKeyEnabled": strings.EqualFold(rule.BucketKeyEnabled, "true"),
		})
	}
	return result
}

func awsS3BucketVersioningAttrs(versioning awsS3BucketVersioningResponse) models.ResAttrs {
	return models.ResAttrs{
		"status":    versioning.Status,
		"mfaDelete": versioning.MFADelete,
		"enabled":   strings.EqualFold(versioning.Status, "Enabled"),
		"suspended": strings.EqualFold(versioning.Status, "Suspended"),
	}
}

func awsS3PublicAccessBlockAttrs(config awsS3PublicAccessBlockResponse) models.ResAttrs {
	return models.ResAttrs{
		"blockPublicAcls":       strings.EqualFold(config.BlockPublicAcls, "true"),
		"ignorePublicAcls":      strings.EqualFold(config.IgnorePublicAcls, "true"),
		"blockPublicPolicy":     strings.EqualFold(config.BlockPublicPolicy, "true"),
		"restrictPublicBuckets": strings.EqualFold(config.RestrictPublicBuckets, "true"),
	}
}

func awsS3PublicAccessBlockEnabled(config awsS3PublicAccessBlockResponse) bool {
	return strings.EqualFold(config.BlockPublicAcls, "true") &&
		strings.EqualFold(config.IgnorePublicAcls, "true") &&
		strings.EqualFold(config.BlockPublicPolicy, "true") &&
		strings.EqualFold(config.RestrictPublicBuckets, "true")
}

func awsS3BucketLifecycleRuleAttrs(rules []awsS3BucketLifecycleRule) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(rules))
	for _, rule := range rules {
		result = append(result, models.ResAttrs{
			"id":                             rule.ID,
			"status":                         rule.Status,
			"enabled":                        strings.EqualFold(rule.Status, "Enabled"),
			"prefix":                         firstNonEmpty(rule.Prefix, rule.Filter.Prefix, rule.Filter.And.Prefix),
			"filter":                         awsS3BucketLifecycleFilterAttrs(rule.Filter),
			"transitions":                    awsS3BucketLifecycleTransitionAttrs(rule.Transitions),
			"noncurrentVersionTransitions":   awsS3BucketLifecycleTransitionAttrs(rule.NoncurrentVersionTransitions),
			"expiration":                     awsS3BucketLifecycleExpirationAttrs(rule.Expiration),
			"noncurrentVersionExpiration":    awsS3BucketLifecycleExpirationAttrs(rule.NoncurrentVersionExpiration),
			"abortIncompleteMultipartUpload": models.ResAttrs{"daysAfterInitiation": rule.AbortIncompleteMultipartUpload.DaysAfterInitiation},
		})
	}
	return result
}

func awsS3BucketLifecycleFilterAttrs(filter awsS3BucketLifecycleFilter) models.ResAttrs {
	attrs := models.ResAttrs{
		"prefix":                filter.Prefix,
		"objectSizeGreaterThan": filter.ObjectSizeGreaterThan,
		"objectSizeLessThan":    filter.ObjectSizeLessThan,
	}
	if filter.Tag.Key != "" {
		attrs["tag"] = models.ResAttrs{"key": filter.Tag.Key, "value": filter.Tag.Value}
	}
	if filter.And.Prefix != "" || len(filter.And.Tags) > 0 || filter.And.ObjectSizeGreaterThan != "" || filter.And.ObjectSizeLessThan != "" {
		attrs["and"] = models.ResAttrs{
			"prefix":                filter.And.Prefix,
			"tags":                  awsS3BucketTagListAttrs(filter.And.Tags),
			"objectSizeGreaterThan": filter.And.ObjectSizeGreaterThan,
			"objectSizeLessThan":    filter.And.ObjectSizeLessThan,
		}
	}
	return attrs
}

func awsS3BucketLifecycleTransitionAttrs(transitions []awsS3BucketLifecycleTransition) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(transitions))
	for _, transition := range transitions {
		result = append(result, models.ResAttrs{
			"date":           transition.Date,
			"days":           transition.Days,
			"noncurrentDays": transition.NoncurrentDays,
			"storageClass":   transition.StorageClass,
		})
	}
	return result
}

func awsS3BucketLifecycleExpirationAttrs(expiration awsS3BucketLifecycleExpiration) models.ResAttrs {
	return models.ResAttrs{
		"date":                      expiration.Date,
		"days":                      expiration.Days,
		"expiredObjectDeleteMarker": strings.EqualFold(expiration.ExpiredObjectDeleteMarker, "true"),
		"noncurrentDays":            expiration.NoncurrentDays,
		"newerNoncurrentVersions":   expiration.NewerNoncurrentVersions,
	}
}

func awsS3BucketPolicyAttrs(policy map[string]interface{}) models.ResAttrs {
	attrs := models.ResAttrs{
		"document":       policy,
		"version":        stringFromInterface(policy["Version"]),
		"id":             stringFromInterface(policy["Id"]),
		"statementCount": awsS3PolicyStatementCount(policy),
		"publicAllow":    awsS3PolicyPublicAllow(policy),
	}
	return attrs
}

func awsS3PolicyStatementCount(policy map[string]interface{}) int {
	statements := awsS3PolicyStatements(policy)
	return len(statements)
}

func awsS3PolicyPublicAllow(policy map[string]interface{}) bool {
	for _, statement := range awsS3PolicyStatements(policy) {
		if !strings.EqualFold(stringFromInterface(statement["Effect"]), "Allow") {
			continue
		}
		if awsS3PolicyPrincipalPublic(statement["Principal"]) {
			return true
		}
	}
	return false
}

func awsS3PolicyStatements(policy map[string]interface{}) []map[string]interface{} {
	raw := policy["Statement"]
	switch value := raw.(type) {
	case []interface{}:
		statements := make([]map[string]interface{}, 0, len(value))
		for _, item := range value {
			if statement, ok := item.(map[string]interface{}); ok {
				statements = append(statements, statement)
			}
		}
		return statements
	case map[string]interface{}:
		return []map[string]interface{}{value}
	default:
		return nil
	}
}

func awsS3PolicyPrincipalPublic(principal interface{}) bool {
	switch value := principal.(type) {
	case string:
		return value == "*"
	case []interface{}:
		for _, item := range value {
			if awsS3PolicyPrincipalPublic(item) {
				return true
			}
		}
	case map[string]interface{}:
		for _, item := range value {
			if awsS3PolicyPrincipalPublic(item) {
				return true
			}
		}
	}
	return false
}

func awsS3BucketACLAttrs(acl awsS3BucketACLResponse) models.ResAttrs {
	return models.ResAttrs{
		"owner":  awsS3OwnerAttrs(acl.Owner),
		"grants": awsS3ACLGrantAttrs(acl.Grants),
		"public": awsS3BucketACLPublic(acl.Grants),
	}
}

func awsS3OwnerAttrs(owner awsS3Owner) models.ResAttrs {
	return models.ResAttrs{
		"id":          owner.ID,
		"displayName": owner.DisplayName,
	}
}

func awsS3ACLGrantAttrs(grants []awsS3ACLGrant) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(grants))
	for _, grant := range grants {
		result = append(result, models.ResAttrs{
			"permission": grant.Permission,
			"grantee":    awsS3ACLGranteeAttrs(grant.Grantee),
			"public":     awsS3ACLGranteePublic(grant.Grantee),
		})
	}
	return result
}

func awsS3ACLGranteeAttrs(grantee awsS3ACLGrantee) models.ResAttrs {
	return models.ResAttrs{
		"type":         grantee.Type,
		"id":           grantee.ID,
		"displayName":  grantee.DisplayName,
		"emailAddress": grantee.EmailAddress,
		"uri":          grantee.URI,
	}
}

func awsS3BucketACLPublic(grants []awsS3ACLGrant) bool {
	for _, grant := range grants {
		if awsS3ACLGranteePublic(grant.Grantee) {
			return true
		}
	}
	return false
}

func awsS3ACLGranteePublic(grantee awsS3ACLGrantee) bool {
	uri := strings.TrimSpace(grantee.URI)
	return strings.Contains(uri, "AllUsers") || strings.Contains(uri, "AuthenticatedUsers")
}

func awsS3ObjectLockAttrs(config awsS3ObjectLockResponse) models.ResAttrs {
	return models.ResAttrs{
		"enabled": strings.EqualFold(config.ObjectLockEnabled, "Enabled"),
		"status":  config.ObjectLockEnabled,
		"defaultRetention": models.ResAttrs{
			"mode":  config.Rule.DefaultRetention.Mode,
			"days":  config.Rule.DefaultRetention.Days,
			"years": config.Rule.DefaultRetention.Years,
		},
	}
}

func awsS3BucketReplicationAttrs(replication awsS3BucketReplicationResponse) models.ResAttrs {
	rules := make([]models.ResAttrs, 0, len(replication.Rules))
	destinationBuckets := make([]string, 0, len(replication.Rules))
	for _, rule := range replication.Rules {
		destinationBucket := strings.TrimSpace(rule.Destination.Bucket)
		if destinationBucket != "" {
			destinationBuckets = append(destinationBuckets, destinationBucket)
		}
		rules = append(rules, models.ResAttrs{
			"id":                        rule.ID,
			"status":                    rule.Status,
			"enabled":                   strings.EqualFold(rule.Status, "Enabled"),
			"priority":                  rule.Priority,
			"prefix":                    firstNonEmpty(rule.Prefix, rule.Filter.Prefix, rule.Filter.And.Prefix),
			"filter":                    awsS3BucketLifecycleFilterAttrs(rule.Filter),
			"deleteMarkerReplication":   rule.DeleteMarkerReplication.Status,
			"existingObjectReplication": rule.ExistingObjectReplication.Status,
			"destination":               awsS3ReplicationDestinationAttrs(rule.Destination),
			"sourceSelectionCriteria":   awsS3SourceSelectionCriteriaAttrs(rule.SourceSelectionCriteria),
		})
	}
	return models.ResAttrs{
		"role":               replication.Role,
		"rules":              rules,
		"destinationBuckets": dedupeStrings(destinationBuckets),
	}
}

func awsS3EnabledReplicationRuleCount(rules []awsS3ReplicationRule) int {
	count := 0
	for _, rule := range rules {
		if strings.EqualFold(rule.Status, "Enabled") {
			count++
		}
	}
	return count
}

func awsS3ReplicationDestinationAttrs(destination awsS3ReplicationDestination) models.ResAttrs {
	return models.ResAttrs{
		"account":                      destination.Account,
		"bucket":                       destination.Bucket,
		"storageClass":                 destination.StorageClass,
		"owner":                        destination.AccessControlTranslation.Owner,
		"replicaKmsKeyId":              destination.EncryptionConfiguration.ReplicaKmsKeyID,
		"metricsStatus":                destination.Metrics.Status,
		"metricsEventThresholdMinutes": destination.Metrics.EventThreshold.Minutes,
		"replicationTimeStatus":        destination.ReplicationTime.Status,
		"replicationTimeMinutes":       destination.ReplicationTime.Time.Minutes,
	}
}

func awsS3SourceSelectionCriteriaAttrs(criteria awsS3SourceSelectionCriteria) models.ResAttrs {
	return models.ResAttrs{
		"replicaModifications":   criteria.ReplicaModifications.Status,
		"sseKmsEncryptedObjects": criteria.SseKmsEncryptedObjects.Status,
	}
}

func awsS3BucketLoggingAttrs(logging awsS3BucketLoggingResponse) models.ResAttrs {
	return models.ResAttrs{
		"enabled":      strings.TrimSpace(logging.LoggingEnabled.TargetBucket) != "",
		"targetBucket": logging.LoggingEnabled.TargetBucket,
		"targetPrefix": logging.LoggingEnabled.TargetPrefix,
		"targetGrants": awsS3ACLGrantAttrs(logging.LoggingEnabled.TargetGrants),
	}
}

func awsS3BucketNotificationAttrs(config awsS3BucketNotificationResponse) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, awsS3BucketNotificationCount(config))
	result = append(result, awsS3NotificationTargetAttrs("topic", config.TopicConfigurations)...)
	result = append(result, awsS3NotificationTargetAttrs("queue", config.QueueConfigurations)...)
	result = append(result, awsS3NotificationTargetAttrs("lambda", config.LambdaFunctionConfigurations)...)
	if config.EventBridgeConfiguration != nil {
		result = append(result, models.ResAttrs{"type": "eventBridge"})
	}
	return result
}

func awsS3NotificationTargetAttrs(targetType string, configs []awsS3NotificationTargetConfig) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(configs))
	for _, config := range configs {
		result = append(result, models.ResAttrs{
			"type":        targetType,
			"id":          config.ID,
			"target":      firstNonEmpty(config.Topic, config.Queue, config.CloudFunction),
			"events":      config.Events,
			"filterRules": awsS3NotificationFilterRuleAttrs(config.Filter.S3Key.Rules),
		})
	}
	return result
}

func awsS3NotificationFilterRuleAttrs(rules []awsS3NotificationFilterRule) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(rules))
	for _, rule := range rules {
		result = append(result, models.ResAttrs{
			"name":  rule.Name,
			"value": rule.Value,
		})
	}
	return result
}

func awsS3BucketNotificationCount(config awsS3BucketNotificationResponse) int {
	count := len(config.TopicConfigurations) + len(config.QueueConfigurations) + len(config.LambdaFunctionConfigurations)
	if config.EventBridgeConfiguration != nil {
		count++
	}
	return count
}

func awsS3BucketWebsiteAttrs(website awsS3BucketWebsiteResponse) models.ResAttrs {
	return models.ResAttrs{
		"enabled": awsS3BucketWebsiteEnabled(website),
		"redirectAllRequestsTo": models.ResAttrs{
			"hostName": website.RedirectAllRequestsTo.HostName,
			"protocol": website.RedirectAllRequestsTo.Protocol,
		},
		"indexDocument": models.ResAttrs{
			"suffix": website.IndexDocument.Suffix,
		},
		"errorDocument": models.ResAttrs{
			"key": website.ErrorDocument.Key,
		},
		"routingRules":     awsS3WebsiteRoutingRuleAttrs(website.RoutingRules),
		"routingRuleCount": len(website.RoutingRules),
	}
}

func awsS3BucketWebsiteEnabled(website awsS3BucketWebsiteResponse) bool {
	return strings.TrimSpace(website.IndexDocument.Suffix) != "" ||
		strings.TrimSpace(website.RedirectAllRequestsTo.HostName) != "" ||
		len(website.RoutingRules) > 0
}

func awsS3WebsiteRoutingRuleAttrs(rules []awsS3WebsiteRoutingRule) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(rules))
	for _, rule := range rules {
		result = append(result, models.ResAttrs{
			"condition": models.ResAttrs{
				"httpErrorCodeReturnedEquals": rule.Condition.HttpErrorCodeReturnedEquals,
				"keyPrefixEquals":             rule.Condition.KeyPrefixEquals,
			},
			"redirect": models.ResAttrs{
				"hostName":             rule.Redirect.HostName,
				"httpRedirectCode":     rule.Redirect.HttpRedirectCode,
				"protocol":             rule.Redirect.Protocol,
				"replaceKeyPrefixWith": rule.Redirect.ReplaceKeyPrefixWith,
				"replaceKeyWith":       rule.Redirect.ReplaceKeyWith,
			},
		})
	}
	return result
}

func awsS3BucketInventoryAttrs(configs []awsS3InventoryConfiguration) []models.ResAttrs {
	result := make([]models.ResAttrs, 0, len(configs))
	for _, config := range configs {
		result = append(result, models.ResAttrs{
			"id":                     config.ID,
			"enabled":                strings.EqualFold(config.IsEnabled, "true"),
			"includedObjectVersions": config.IncludedObjectVersions,
			"filterPrefix":           config.Filter.Prefix,
			"optionalFields":         config.OptionalFields,
			"scheduleFrequency":      config.Schedule.Frequency,
			"destination": models.ResAttrs{
				"accountId":       config.Destination.S3BucketDestination.AccountID,
				"bucket":          config.Destination.S3BucketDestination.Bucket,
				"format":          config.Destination.S3BucketDestination.Format,
				"prefix":          config.Destination.S3BucketDestination.Prefix,
				"sseKmsKeyId":     config.Destination.S3BucketDestination.Encryption.SSEKMS.KeyID,
				"sseS3Configured": config.Destination.S3BucketDestination.Encryption.SSES3 != nil,
			},
		})
	}
	return result
}

func awsS3BucketInventoryEnabledCount(configs []awsS3InventoryConfiguration) int {
	count := 0
	for _, config := range configs {
		if strings.EqualFold(config.IsEnabled, "true") {
			count++
		}
	}
	return count
}

func stringFromInterface(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
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
