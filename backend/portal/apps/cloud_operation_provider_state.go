// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models"
	"cloudiac/portal/services"
)

const azureComputeAPIVersion = "2023-09-01"

func cloudOperationCmdbCloudAccountForAsset(c *ctx.ServiceContext, asset *models.CmdbAsset) *cmdbCloudAccount {
	if c == nil || asset == nil || asset.CloudAccountId == "" {
		return nil
	}
	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, asset.CloudAccountId)
	if err != nil {
		return nil
	}
	return cloudOperationCmdbCloudAccountFromModel(account)
}

func cloudOperationCmdbCloudAccountFromModel(account *models.CloudAccount) *cmdbCloudAccount {
	if account == nil {
		return nil
	}
	provider := normalizeProvider(account.Provider)
	credentials := credentialMapWithRegions(provider, credentialMapFromCloudCredentials(account.Credentials), []string(account.Regions))
	return buildCmdbCloudAccount(account.Id, models.CmdbCloudAccountSourceCloudAccount,
		account.Name, account.Description, provider, account.AccountId, account.UpdatedAt, []string(account.Regions), credentials)
}

func cloudProviderCachedComputeState(request cloudProviderOperationRequest, rawResponse models.ResAttrs) cloudProviderResourceState {
	if rawResponse == nil {
		rawResponse = models.ResAttrs{
			"source":          "cmdb_cache",
			"observedState":   request.ObservedState,
			"normalizedState": cloudNormalizeComputeState(request.Provider, request.ObservedState),
		}
	}
	return cloudProviderResourceState{
		RawState:        request.ObservedState,
		NormalizedState: cloudNormalizeComputeState(request.Provider, request.ObservedState),
		Source:          "CMDB缓存",
		ReadMode:        firstNonEmpty(request.ReadMode, cloudOperationProviderReadModeCache),
		RawResponse:     cloudProviderSanitizeAttrs(rawResponse),
	}
}

func cloudProviderLiveComputeState(request cloudProviderOperationRequest, account *cmdbCloudAccount) cloudProviderResourceState {
	if account == nil {
		return cloudProviderComputeStateReadError(request, "cloud_account_missing", "未绑定可用于 provider 只读查询的云账号", false, nil)
	}
	if !account.Ready {
		return cloudProviderComputeStateReadError(request, "credential_incomplete",
			fmt.Sprintf("云账号凭证不完整：%s", strings.Join(account.MissingCredentialKeys, ", ")), false, nil)
	}
	if strings.TrimSpace(request.Region) == "" {
		return cloudProviderComputeStateReadError(request, "region_missing", "缺少云资源区域，无法执行 provider 只读查询", false, nil)
	}
	if strings.TrimSpace(request.ResourceId) == "" {
		return cloudProviderComputeStateReadError(request, "resource_id_missing", "缺少云资源 ID，无法执行 provider 只读查询", false, nil)
	}

	timeout := cloudOperationProviderReadTimeout()
	readCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	switch normalizeProvider(request.Provider) {
	case "aws":
		return cloudProviderReadAwsComputeState(readCtx, account, request)
	case "oci":
		return cloudProviderReadOciComputeState(readCtx, account, request)
	case "alicloud":
		return cloudProviderReadAlicloudComputeState(readCtx, account, request)
	case "azure":
		return cloudProviderReadAzureComputeState(readCtx, account, request)
	case "gcp":
		return cloudProviderReadGcpComputeState(readCtx, account, request)
	default:
		return cloudProviderComputeStateReadError(request, "provider_reader_unsupported",
			fmt.Sprintf("暂不支持 %s 的 provider 只读状态查询", firstNonEmpty(request.Provider, "unknown")), false, nil)
	}
}

func cloudProviderLiveVolumeState(request cloudProviderOperationRequest, account *cmdbCloudAccount) cloudProviderResourceState {
	if account == nil {
		return cloudProviderVolumeStateReadError(request, "cloud_account_missing", "未绑定可用于 provider 只读查询的云账号", false, nil)
	}
	if !account.Ready {
		return cloudProviderVolumeStateReadError(request, "credential_incomplete",
			fmt.Sprintf("云账号凭证不完整：%s", strings.Join(account.MissingCredentialKeys, ", ")), false, nil)
	}
	if strings.TrimSpace(request.Region) == "" {
		return cloudProviderVolumeStateReadError(request, "region_missing", "缺少云资源区域，无法执行 provider 只读查询", false, nil)
	}
	if strings.TrimSpace(request.ResourceId) == "" {
		return cloudProviderVolumeStateReadError(request, "resource_id_missing", "缺少云资源 ID，无法执行 provider 只读查询", false, nil)
	}

	timeout := cloudOperationProviderReadTimeout()
	readCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	switch normalizeProvider(request.Provider) {
	case "aws":
		return cloudProviderReadAwsVolumeState(readCtx, account, request)
	case "oci":
		return cloudProviderReadOciVolumeState(readCtx, account, request)
	case "alicloud":
		return cloudProviderReadAlicloudVolumeState(readCtx, account, request)
	default:
		return cloudProviderVolumeStateReadError(request, "provider_reader_unsupported",
			fmt.Sprintf("暂不支持 %s 的块存储 provider 只读查询", firstNonEmpty(request.Provider, "unknown")), false, nil)
	}
}

func cloudProviderLiveSecurityGroupState(request cloudProviderOperationRequest, account *cmdbCloudAccount) cloudProviderResourceState {
	if account == nil {
		return cloudProviderSecurityGroupStateReadError(request, "cloud_account_missing", "未绑定可用于 provider 只读查询的云账号", false, nil)
	}
	if !account.Ready {
		return cloudProviderSecurityGroupStateReadError(request, "credential_incomplete",
			fmt.Sprintf("云账号凭证不完整：%s", strings.Join(account.MissingCredentialKeys, ", ")), false, nil)
	}
	if strings.TrimSpace(request.Region) == "" {
		return cloudProviderSecurityGroupStateReadError(request, "region_missing", "缺少云资源区域，无法执行 provider 只读查询", false, nil)
	}
	if strings.TrimSpace(request.ResourceId) == "" {
		return cloudProviderSecurityGroupStateReadError(request, "resource_id_missing", "缺少云资源 ID，无法执行 provider 只读查询", false, nil)
	}

	timeout := cloudOperationProviderReadTimeout()
	readCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	switch normalizeProvider(request.Provider) {
	case "aws":
		return cloudProviderReadAwsSecurityGroupState(readCtx, account, request)
	case "oci":
		return cloudProviderReadOciSecurityGroupState(readCtx, account, request)
	case "alicloud":
		return cloudProviderReadAlicloudSecurityGroupState(readCtx, account, request)
	default:
		return cloudProviderSecurityGroupStateReadError(request, "provider_reader_unsupported",
			fmt.Sprintf("暂不支持 %s 的安全组 provider 只读查询", firstNonEmpty(request.Provider, "unknown")), false, nil)
	}
}

func cloudProviderReadAwsComputeState(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) cloudProviderResourceState {
	values := url.Values{
		"Action":       []string{"DescribeInstances"},
		"Version":      []string{awsEC2APIVersion},
		"InstanceId.1": []string{request.ResourceId},
	}
	resp := awsEC2DescribeInstancesResponse{}
	if err := awsQueryAPIWithValues(c, account, "ec2", request.Region, values, &resp); err != nil {
		return cloudProviderComputeStateReadError(request, "provider_read_failed", err.Error(), cloudProviderErrorRetryable(err), models.ResAttrs{
			"provider": "aws",
			"service":  "ec2",
			"action":   "DescribeInstances",
			"request": models.ResAttrs{
				"region":     request.Region,
				"instanceId": request.ResourceId,
			},
		})
	}

	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			if instance.InstanceId != request.ResourceId {
				continue
			}
			rawState := strings.TrimSpace(instance.State.Name)
			return cloudProviderResourceState{
				RawState:        rawState,
				NormalizedState: cloudNormalizeComputeState(request.Provider, rawState),
				Source:          "Provider只读查询",
				ReadMode:        cloudOperationProviderReadModeLive,
				RawResponse: cloudProviderSanitizeAttrs(models.ResAttrs{
					"provider":     "aws",
					"service":      "ec2",
					"action":       "DescribeInstances",
					"instanceType": instance.InstanceType,
					"response": models.ResAttrs{
						"reservationId":    reservation.ReservationId,
						"instanceId":       instance.InstanceId,
						"instanceType":     instance.InstanceType,
						"state":            rawState,
						"availabilityZone": instance.Placement.AvailabilityZone,
					},
				}),
			}
		}
	}

	return cloudProviderComputeStateReadError(request, "resource_not_found",
		fmt.Sprintf("AWS EC2 实例 %s 未返回", request.ResourceId), false, models.ResAttrs{
			"provider":     "aws",
			"service":      "ec2",
			"action":       "DescribeInstances",
			"reservations": len(resp.Reservations),
		})
}

func cloudProviderReadAwsVolumeState(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) cloudProviderResourceState {
	values := url.Values{
		"Action":     []string{"DescribeVolumes"},
		"Version":    []string{awsEC2APIVersion},
		"VolumeId.1": []string{request.ResourceId},
	}
	resp := awsEC2DescribeVolumesResponse{}
	if err := awsQueryAPIWithValues(c, account, "ec2", request.Region, values, &resp); err != nil {
		return cloudProviderVolumeStateReadError(request, "provider_read_failed", err.Error(), cloudProviderErrorRetryable(err), models.ResAttrs{
			"provider": "aws",
			"service":  "ec2",
			"action":   "DescribeVolumes",
			"request": models.ResAttrs{
				"region":   request.Region,
				"volumeId": request.ResourceId,
			},
		})
	}

	for _, volume := range resp.Volumes {
		if volume.VolumeId != request.ResourceId {
			continue
		}
		rawState := strings.TrimSpace(volume.Status)
		return cloudProviderResourceState{
			RawState:        rawState,
			NormalizedState: cloudVolumeStateValue(volume.Size),
			Source:          "Provider只读查询",
			ReadMode:        cloudOperationProviderReadModeLive,
			RawResponse: cloudProviderSanitizeAttrs(models.ResAttrs{
				"provider":       "aws",
				"service":        "ec2",
				"action":         "DescribeVolumes",
				"currentSizeGiB": volume.Size,
				"response": models.ResAttrs{
					"volumeId":         volume.VolumeId,
					"status":           rawState,
					"sizeGiB":          volume.Size,
					"volumeType":       volume.VolumeType,
					"iops":             volume.Iops,
					"encrypted":        volume.Encrypted,
					"availabilityZone": volume.AvailabilityZone,
				},
			}),
		}
	}

	return cloudProviderVolumeStateReadError(request, "resource_not_found",
		fmt.Sprintf("AWS EBS 卷 %s 未返回", request.ResourceId), false, models.ResAttrs{
			"provider": "aws",
			"service":  "ec2",
			"action":   "DescribeVolumes",
			"volumes":  len(resp.Volumes),
		})
}

func cloudProviderReadAwsSecurityGroupState(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) cloudProviderResourceState {
	values := url.Values{
		"Action":    []string{"DescribeSecurityGroups"},
		"Version":   []string{awsEC2APIVersion},
		"GroupId.1": []string{request.ResourceId},
	}
	resp := awsEC2DescribeSecurityGroupsResponse{}
	if err := awsQueryAPIWithValues(c, account, "ec2", request.Region, values, &resp); err != nil {
		return cloudProviderSecurityGroupStateReadError(request, "provider_read_failed", err.Error(), cloudProviderErrorRetryable(err), models.ResAttrs{
			"provider": "aws",
			"service":  "ec2",
			"action":   "DescribeSecurityGroups",
			"request": models.ResAttrs{
				"region":  request.Region,
				"groupId": request.ResourceId,
			},
		})
	}
	for _, group := range resp.SecurityGroups {
		if group.GroupId != request.ResourceId {
			continue
		}
		ruleCount := len(group.IpPermissions) + len(group.IpPermissionsEgress)
		return cloudProviderResourceState{
			RawState:        "available",
			NormalizedState: "ready",
			Source:          "Provider只读查询",
			ReadMode:        cloudOperationProviderReadModeLive,
			RawResponse: cloudProviderSanitizeAttrs(models.ResAttrs{
				"provider":  "aws",
				"service":   "ec2",
				"action":    "DescribeSecurityGroups",
				"ruleCount": ruleCount,
				"response": models.ResAttrs{
					"groupId":          group.GroupId,
					"groupName":        group.GroupName,
					"vpcId":            group.VpcId,
					"ingressRuleCount": len(group.IpPermissions),
					"egressRuleCount":  len(group.IpPermissionsEgress),
				},
			}),
		}
	}
	return cloudProviderSecurityGroupStateReadError(request, "resource_not_found",
		fmt.Sprintf("AWS Security Group %s 未返回", request.ResourceId), false, models.ResAttrs{
			"provider": "aws",
			"service":  "ec2",
			"action":   "DescribeSecurityGroups",
		})
}

func cloudProviderReadOciComputeState(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) cloudProviderResourceState {
	item := models.ResAttrs{}
	path := "/instances/" + url.PathEscape(request.ResourceId)
	if err := ociJSONAPI(c, account, request.Region, "iaas", ociCoreAPIVersion, path, nil, &item); err != nil {
		return cloudProviderComputeStateReadError(request, "provider_read_failed", err.Error(), cloudProviderErrorRetryable(err), models.ResAttrs{
			"provider": "oci",
			"service":  "iaas",
			"action":   "GetInstance",
			"request": models.ResAttrs{
				"region":     request.Region,
				"instanceId": request.ResourceId,
			},
		})
	}

	rawState := ociAttrString(item, "lifecycleState")
	return cloudProviderResourceState{
		RawState:        rawState,
		NormalizedState: cloudNormalizeComputeState(request.Provider, rawState),
		Source:          "Provider只读查询",
		ReadMode:        cloudOperationProviderReadModeLive,
		RawResponse: cloudProviderSanitizeAttrs(models.ResAttrs{
			"provider": "oci",
			"service":  "iaas",
			"action":   "GetInstance",
			"response": models.ResAttrs{
				"id":                 ociAttrString(item, "id"),
				"displayName":        ociAttrString(item, "displayName"),
				"lifecycleState":     rawState,
				"shape":              ociAttrString(item, "shape"),
				"availabilityDomain": ociAttrString(item, "availabilityDomain"),
				"compartmentId":      ociAttrString(item, "compartmentId"),
			},
		}),
	}
}

func cloudProviderReadOciVolumeState(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) cloudProviderResourceState {
	item := models.ResAttrs{}
	path := "/volumes/" + url.PathEscape(request.ResourceId)
	if err := ociJSONAPI(c, account, request.Region, "iaas", ociCoreAPIVersion, path, nil, &item); err != nil {
		return cloudProviderVolumeStateReadError(request, "provider_read_failed", err.Error(), cloudProviderErrorRetryable(err), models.ResAttrs{
			"provider": "oci",
			"service":  "iaas",
			"action":   "GetVolume",
			"request": models.ResAttrs{
				"region":   request.Region,
				"volumeId": request.ResourceId,
			},
		})
	}

	sizeGiB := cloudVolumeSizeGiBFromAttrs(item)
	rawState := ociAttrString(item, "lifecycleState")
	return cloudProviderResourceState{
		RawState:        rawState,
		NormalizedState: cloudVolumeStateValue(sizeGiB),
		Source:          "Provider只读查询",
		ReadMode:        cloudOperationProviderReadModeLive,
		RawResponse: cloudProviderSanitizeAttrs(models.ResAttrs{
			"provider":       "oci",
			"service":        "iaas",
			"action":         "GetVolume",
			"currentSizeGiB": sizeGiB,
			"response": models.ResAttrs{
				"id":                 ociAttrString(item, "id"),
				"displayName":        ociAttrString(item, "displayName"),
				"lifecycleState":     rawState,
				"sizeGiB":            sizeGiB,
				"sizeInGBs":          item["sizeInGBs"],
				"vpusPerGB":          item["vpusPerGB"],
				"availabilityDomain": ociAttrString(item, "availabilityDomain"),
				"compartmentId":      ociAttrString(item, "compartmentId"),
			},
		}),
	}
}

type alicloudECSDescribeInstancesResponse struct {
	RequestId  string `json:"RequestId"`
	TotalCount int    `json:"TotalCount"`
	Instances  struct {
		Instance []alicloudECSInstance `json:"Instance"`
	} `json:"Instances"`
}

type alicloudECSInstance struct {
	InstanceId   string `json:"InstanceId"`
	InstanceName string `json:"InstanceName"`
	InstanceType string `json:"InstanceType"`
	Status       string `json:"Status"`
	ZoneId       string `json:"ZoneId"`
	RegionId     string `json:"RegionId"`
}

type alicloudECSDescribeDisksResponse struct {
	RequestId  string `json:"RequestId"`
	TotalCount int    `json:"TotalCount"`
	Disks      struct {
		Disk []alicloudECSDisk `json:"Disk"`
	} `json:"Disks"`
}

type alicloudECSDisk struct {
	DiskId   string `json:"DiskId"`
	DiskName string `json:"DiskName"`
	Status   string `json:"Status"`
	Size     int    `json:"Size"`
	Type     string `json:"Type"`
	ZoneId   string `json:"ZoneId"`
	RegionId string `json:"RegionId"`
	Category string `json:"Category"`
}

type alicloudECSDescribeSecurityGroupAttributeResponse struct {
	RequestId       string `json:"RequestId"`
	SecurityGroupId string `json:"SecurityGroupId"`
	Permissions     struct {
		Permission []models.ResAttrs `json:"Permission"`
	} `json:"Permissions"`
}

func cloudProviderReadOciSecurityGroupState(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) cloudProviderResourceState {
	resourceId := strings.TrimSpace(request.ResourceId)
	path := "/networkSecurityGroups/" + url.PathEscape(resourceId)
	action := "GetNetworkSecurityGroup"
	if strings.HasPrefix(resourceId, "ocid1.securitylist") {
		path = "/securityLists/" + url.PathEscape(resourceId)
		action = "GetSecurityList"
	}
	item := models.ResAttrs{}
	if err := ociJSONAPI(c, account, request.Region, "iaas", ociCoreAPIVersion, path, nil, &item); err != nil {
		return cloudProviderSecurityGroupStateReadError(request, "provider_read_failed", err.Error(), cloudProviderErrorRetryable(err), models.ResAttrs{
			"provider": "oci",
			"service":  "iaas",
			"action":   action,
			"request": models.ResAttrs{
				"region":     request.Region,
				"resourceId": request.ResourceId,
			},
		})
	}
	rawState := firstNonEmpty(ociAttrString(item, "lifecycleState"), "available")
	return cloudProviderResourceState{
		RawState:        rawState,
		NormalizedState: "ready",
		Source:          "Provider只读查询",
		ReadMode:        cloudOperationProviderReadModeLive,
		RawResponse: cloudProviderSanitizeAttrs(models.ResAttrs{
			"provider": "oci",
			"service":  "iaas",
			"action":   action,
			"response": models.ResAttrs{
				"id":             ociAttrString(item, "id"),
				"displayName":    ociAttrString(item, "displayName"),
				"lifecycleState": rawState,
				"vcnId":          ociAttrString(item, "vcnId"),
			},
		}),
	}
}

func cloudProviderReadAlicloudComputeState(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) cloudProviderResourceState {
	resp := alicloudECSDescribeInstancesResponse{}
	values := url.Values{
		"Action":      []string{"DescribeInstances"},
		"Version":     []string{"2014-05-26"},
		"RegionId":    []string{request.Region},
		"InstanceIds": []string{fmt.Sprintf("[\"%s\"]", request.ResourceId)},
	}
	if err := alicloudRPCAPI(c, account, request.Region, values, &resp); err != nil {
		return cloudProviderComputeStateReadError(request, "provider_read_failed", err.Error(), cloudProviderErrorRetryable(err), models.ResAttrs{
			"provider": "alicloud",
			"service":  "ecs",
			"action":   "DescribeInstances",
			"request": models.ResAttrs{
				"region":     request.Region,
				"instanceId": request.ResourceId,
			},
		})
	}

	for _, instance := range resp.Instances.Instance {
		if instance.InstanceId != request.ResourceId {
			continue
		}
		rawState := strings.TrimSpace(instance.Status)
		return cloudProviderResourceState{
			RawState:        rawState,
			NormalizedState: cloudNormalizeComputeState(request.Provider, rawState),
			Source:          "Provider只读查询",
			ReadMode:        cloudOperationProviderReadModeLive,
			RawResponse: cloudProviderSanitizeAttrs(models.ResAttrs{
				"provider":     "alicloud",
				"service":      "ecs",
				"action":       "DescribeInstances",
				"instanceType": instance.InstanceType,
				"response": models.ResAttrs{
					"requestId":    resp.RequestId,
					"totalCount":   resp.TotalCount,
					"instanceId":   instance.InstanceId,
					"instanceName": instance.InstanceName,
					"instanceType": instance.InstanceType,
					"status":       rawState,
					"zoneId":       instance.ZoneId,
					"regionId":     firstNonEmpty(instance.RegionId, request.Region),
				},
			}),
		}
	}

	return cloudProviderComputeStateReadError(request, "resource_not_found",
		fmt.Sprintf("阿里云 ECS 实例 %s 未返回", request.ResourceId), false, models.ResAttrs{
			"provider":   "alicloud",
			"service":    "ecs",
			"action":     "DescribeInstances",
			"requestId":  resp.RequestId,
			"totalCount": resp.TotalCount,
		})
}

func cloudProviderReadAlicloudVolumeState(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) cloudProviderResourceState {
	resp := alicloudECSDescribeDisksResponse{}
	values := url.Values{
		"Action":   []string{"DescribeDisks"},
		"Version":  []string{"2014-05-26"},
		"RegionId": []string{request.Region},
		"DiskIds":  []string{fmt.Sprintf("[\"%s\"]", request.ResourceId)},
	}
	if err := alicloudRPCAPI(c, account, request.Region, values, &resp); err != nil {
		return cloudProviderVolumeStateReadError(request, "provider_read_failed", err.Error(), cloudProviderErrorRetryable(err), models.ResAttrs{
			"provider": "alicloud",
			"service":  "ecs",
			"action":   "DescribeDisks",
			"request": models.ResAttrs{
				"region": request.Region,
				"diskId": request.ResourceId,
			},
		})
	}

	for _, disk := range resp.Disks.Disk {
		if disk.DiskId != request.ResourceId {
			continue
		}
		rawState := strings.TrimSpace(disk.Status)
		return cloudProviderResourceState{
			RawState:        rawState,
			NormalizedState: cloudVolumeStateValue(disk.Size),
			Source:          "Provider只读查询",
			ReadMode:        cloudOperationProviderReadModeLive,
			RawResponse: cloudProviderSanitizeAttrs(models.ResAttrs{
				"provider":       "alicloud",
				"service":        "ecs",
				"action":         "DescribeDisks",
				"currentSizeGiB": disk.Size,
				"response": models.ResAttrs{
					"requestId":  resp.RequestId,
					"totalCount": resp.TotalCount,
					"diskId":     disk.DiskId,
					"diskName":   disk.DiskName,
					"status":     rawState,
					"sizeGiB":    disk.Size,
					"type":       disk.Type,
					"category":   disk.Category,
					"zoneId":     disk.ZoneId,
					"regionId":   firstNonEmpty(disk.RegionId, request.Region),
				},
			}),
		}
	}

	return cloudProviderVolumeStateReadError(request, "resource_not_found",
		fmt.Sprintf("阿里云磁盘 %s 未返回", request.ResourceId), false, models.ResAttrs{
			"provider":   "alicloud",
			"service":    "ecs",
			"action":     "DescribeDisks",
			"requestId":  resp.RequestId,
			"totalCount": resp.TotalCount,
		})
}

func cloudProviderReadAlicloudSecurityGroupState(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) cloudProviderResourceState {
	resp := alicloudECSDescribeSecurityGroupAttributeResponse{}
	values := url.Values{
		"Action":          []string{"DescribeSecurityGroupAttribute"},
		"Version":         []string{"2014-05-26"},
		"RegionId":        []string{request.Region},
		"SecurityGroupId": []string{request.ResourceId},
	}
	if err := alicloudRPCAPI(c, account, request.Region, values, &resp); err != nil {
		return cloudProviderSecurityGroupStateReadError(request, "provider_read_failed", err.Error(), cloudProviderErrorRetryable(err), models.ResAttrs{
			"provider": "alicloud",
			"service":  "ecs",
			"action":   "DescribeSecurityGroupAttribute",
			"request": models.ResAttrs{
				"region":          request.Region,
				"securityGroupId": request.ResourceId,
			},
		})
	}
	return cloudProviderResourceState{
		RawState:        "available",
		NormalizedState: "ready",
		Source:          "Provider只读查询",
		ReadMode:        cloudOperationProviderReadModeLive,
		RawResponse: cloudProviderSanitizeAttrs(models.ResAttrs{
			"provider":  "alicloud",
			"service":   "ecs",
			"action":    "DescribeSecurityGroupAttribute",
			"ruleCount": len(resp.Permissions.Permission),
			"response": models.ResAttrs{
				"requestId":       resp.RequestId,
				"securityGroupId": firstNonEmpty(resp.SecurityGroupId, request.ResourceId),
				"ruleCount":       len(resp.Permissions.Permission),
			},
		}),
	}
}

type azureVMInstanceViewResponse struct {
	Statuses []struct {
		Code          string `json:"code"`
		DisplayStatus string `json:"displayStatus"`
		Level         string `json:"level"`
		Message       string `json:"message"`
	} `json:"statuses"`
}

func cloudProviderReadAzureComputeState(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) cloudProviderResourceState {
	token, err := azureAccessToken(c, account)
	if err != nil {
		return cloudProviderComputeStateReadError(request, "provider_read_failed", err.Error(), cloudProviderErrorRetryable(err), models.ResAttrs{
			"provider": "azure",
			"service":  "compute",
			"action":   "GetVirtualMachineInstanceView",
		})
	}
	resourcePath := azureResourcePath(request.ResourceId)
	if resourcePath == "" {
		return cloudProviderComputeStateReadError(request, "resource_id_invalid",
			"Azure VM provider live read requires a full ARM resource ID", false, models.ResAttrs{
				"provider":   "azure",
				"service":    "compute",
				"action":     "GetVirtualMachineInstanceView",
				"resourceId": request.ResourceId,
			})
	}
	rawURL := fmt.Sprintf("%s%s/instanceView?api-version=%s", azureManagementEndpoint, resourcePath, url.QueryEscape(azureComputeAPIVersion))
	req, err := http.NewRequestWithContext(c, http.MethodGet, rawURL, nil)
	if err != nil {
		return cloudProviderComputeStateReadError(request, "provider_read_failed", err.Error(), false, nil)
	}
	req.Header.Set("Accept", "application/json")
	resp := azureVMInstanceViewResponse{}
	if err := azureDoJSON(req, token, &resp); err != nil {
		return cloudProviderComputeStateReadError(request, "provider_read_failed", err.Error(), cloudProviderErrorRetryable(err), models.ResAttrs{
			"provider": "azure",
			"service":  "compute",
			"action":   "GetVirtualMachineInstanceView",
			"request": models.ResAttrs{
				"resourceId": request.ResourceId,
			},
		})
	}
	rawState := azurePowerState(resp.Statuses)
	return cloudProviderResourceState{
		RawState:        rawState,
		NormalizedState: cloudNormalizeComputeState(request.Provider, rawState),
		Source:          "Provider只读查询",
		ReadMode:        cloudOperationProviderReadModeLive,
		RawResponse: cloudProviderSanitizeAttrs(models.ResAttrs{
			"provider": "azure",
			"service":  "compute",
			"action":   "GetVirtualMachineInstanceView",
			"response": models.ResAttrs{
				"resourceId": request.ResourceId,
				"powerState": rawState,
				"statuses":   resp.Statuses,
			},
		}),
	}
}

func cloudProviderReadGcpComputeState(c context.Context, account *cmdbCloudAccount, request cloudProviderOperationRequest) cloudProviderResourceState {
	token, err := gcpAccessToken(c, account)
	if err != nil {
		return cloudProviderComputeStateReadError(request, "provider_read_failed", err.Error(), cloudProviderErrorRetryable(err), models.ResAttrs{
			"provider": "gcp",
			"service":  "compute",
			"action":   "GetInstance",
		})
	}
	projectId := gcpProjectId(account)
	zone, instanceName := gcpComputeInstanceTarget(request)
	if projectId == "" || zone == "" || instanceName == "" {
		return cloudProviderComputeStateReadError(request, "resource_id_invalid",
			"GCP provider live read requires project, zone and instance name", false, models.ResAttrs{
				"provider":   "gcp",
				"service":    "compute",
				"action":     "GetInstance",
				"projectId":  projectId,
				"zone":       zone,
				"resourceId": request.ResourceId,
			})
	}
	item := gcpInstance{}
	rawURL := fmt.Sprintf("%s/projects/%s/zones/%s/instances/%s",
		gcpComputeEndpoint, url.PathEscape(projectId), url.PathEscape(zone), url.PathEscape(instanceName))
	if err := gcpGet(c, token, rawURL, &item); err != nil {
		return cloudProviderComputeStateReadError(request, "provider_read_failed", err.Error(), cloudProviderErrorRetryable(err), models.ResAttrs{
			"provider": "gcp",
			"service":  "compute",
			"action":   "GetInstance",
			"request": models.ResAttrs{
				"projectId":    projectId,
				"zone":         zone,
				"instanceName": instanceName,
			},
		})
	}
	rawState := strings.TrimSpace(item.Status)
	return cloudProviderResourceState{
		RawState:        rawState,
		NormalizedState: cloudNormalizeComputeState(request.Provider, rawState),
		Source:          "Provider只读查询",
		ReadMode:        cloudOperationProviderReadModeLive,
		RawResponse: cloudProviderSanitizeAttrs(models.ResAttrs{
			"provider":     "gcp",
			"service":      "compute",
			"action":       "GetInstance",
			"instanceType": gcpLastPathSegment(item.MachineType),
			"response": models.ResAttrs{
				"projectId":    projectId,
				"zone":         zone,
				"instanceId":   item.Id,
				"instanceName": item.Name,
				"status":       rawState,
				"machineType":  gcpLastPathSegment(item.MachineType),
				"selfLink":     item.SelfLink,
			},
		}),
	}
}

func azureResourcePath(resourceId string) string {
	resourceId = strings.TrimSpace(resourceId)
	if resourceId == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(resourceId), "https://") {
		if parsed, err := url.Parse(resourceId); err == nil {
			resourceId = parsed.Path
		}
	}
	resourceId = "/" + strings.Trim(resourceId, "/")
	if !strings.HasPrefix(strings.ToLower(resourceId), "/subscriptions/") {
		return ""
	}
	return resourceId
}

func azurePowerState(statuses []struct {
	Code          string `json:"code"`
	DisplayStatus string `json:"displayStatus"`
	Level         string `json:"level"`
	Message       string `json:"message"`
}) string {
	for _, status := range statuses {
		code := strings.TrimSpace(status.Code)
		if strings.HasPrefix(strings.ToLower(code), "powerstate/") {
			return code
		}
	}
	for _, status := range statuses {
		if display := strings.TrimSpace(status.DisplayStatus); display != "" {
			return display
		}
	}
	return "unknown"
}

func gcpComputeInstanceTarget(request cloudProviderOperationRequest) (string, string) {
	resourceId := strings.TrimSpace(request.ResourceId)
	if strings.Contains(resourceId, "/zones/") && strings.Contains(resourceId, "/instances/") {
		parts := strings.Split(strings.Trim(resourceId, "/"), "/")
		zone := ""
		instance := ""
		for idx, part := range parts {
			if part == "zones" && idx+1 < len(parts) {
				zone = parts[idx+1]
			}
			if part == "instances" && idx+1 < len(parts) {
				instance = parts[idx+1]
			}
		}
		if zone != "" && instance != "" {
			return zone, instance
		}
	}
	return firstNonEmpty(request.Zone, request.Region), resourceId
}

func cloudProviderComputeStateReadError(request cloudProviderOperationRequest, code string, message string, retryable bool, rawResponse models.ResAttrs) cloudProviderResourceState {
	if rawResponse == nil {
		rawResponse = models.ResAttrs{}
	}
	rawResponse["readMode"] = cloudOperationProviderReadModeLive
	rawResponse["fallback"] = "cmdb_cache"
	rawResponse["error"] = models.ResAttrs{
		"code":      code,
		"message":   message,
		"retryable": retryable,
	}
	state := cloudProviderCachedComputeState(request, rawResponse)
	state.Source = "Provider只读查询"
	state.ReadMode = cloudOperationProviderReadModeLive
	state.ErrorCode = code
	state.ErrorMessage = message
	state.Retryable = retryable
	return state
}

func cloudProviderVolumeStateReadError(request cloudProviderOperationRequest, code string, message string, retryable bool, rawResponse models.ResAttrs) cloudProviderResourceState {
	if rawResponse == nil {
		rawResponse = models.ResAttrs{}
	}
	rawResponse["readMode"] = cloudOperationProviderReadModeLive
	rawResponse["fallback"] = "cmdb_cache"
	rawResponse["error"] = models.ResAttrs{
		"code":      code,
		"message":   message,
		"retryable": retryable,
	}
	state := cloudProviderCachedVolumeState(request)
	state.RawResponse = cloudProviderSanitizeAttrs(rawResponse)
	state.Source = "Provider只读查询"
	state.ReadMode = cloudOperationProviderReadModeLive
	state.ErrorCode = code
	state.ErrorMessage = message
	state.Retryable = retryable
	return state
}

func cloudProviderSecurityGroupStateReadError(request cloudProviderOperationRequest, code string, message string, retryable bool, rawResponse models.ResAttrs) cloudProviderResourceState {
	if rawResponse == nil {
		rawResponse = models.ResAttrs{}
	}
	rawResponse["readMode"] = cloudOperationProviderReadModeLive
	rawResponse["fallback"] = "cmdb_cache"
	rawResponse["error"] = models.ResAttrs{
		"code":      code,
		"message":   message,
		"retryable": retryable,
	}
	state := cloudProviderCachedSecurityGroupState(request)
	state.RawResponse = cloudProviderSanitizeAttrs(rawResponse)
	state.Source = "Provider只读查询"
	state.ReadMode = cloudOperationProviderReadModeLive
	state.ErrorCode = code
	state.ErrorMessage = message
	state.Retryable = retryable
	return state
}

func alicloudRPCAPI(c context.Context, account *cmdbCloudAccount, region string, values url.Values, out interface{}) error {
	return alicloudRPCServiceAPI(c, account, "ecs", region, values, out)
}

func signAlicloudRPC(values url.Values, secret string) string {
	canonicalized := canonicalAlicloudRPCQuery(values)
	stringToSign := "GET&%2F&" + alicloudPercentEncode(canonicalized)
	mac := hmac.New(sha1.New, []byte(strings.TrimSpace(secret)+"&"))
	_, _ = mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func canonicalAlicloudRPCQuery(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key == "Signature" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		items := append([]string{}, values[key]...)
		sort.Strings(items)
		for _, value := range items {
			pairs = append(pairs, alicloudPercentEncode(key)+"="+alicloudPercentEncode(value))
		}
	}
	return strings.Join(pairs, "&")
}

func alicloudPercentEncode(value string) string {
	encoded := url.QueryEscape(value)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

func alicloudRPCError(body []byte) error {
	resp := struct {
		Code      string `json:"Code"`
		Message   string `json:"Message"`
		RequestId string `json:"RequestId"`
	}{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if resp.Code == "" {
		return nil
	}
	return fmt.Errorf("alicloud %s: %s requestId=%s", resp.Code, resp.Message, resp.RequestId)
}

func cloudOperationProviderReadTimeout() time.Duration {
	seconds := 15
	if value := strings.TrimSpace(os.Getenv(cloudOperationProviderReadTimeoutEnv)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			seconds = parsed
		}
	}
	return time.Duration(seconds) * time.Second
}

func cloudProviderErrorRetryable(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	for _, token := range []string{"timeout", "temporary", "too many requests", "throttl", "429", "500", "502", "503", "504"} {
		if strings.Contains(message, token) {
			return true
		}
	}
	return false
}

func cloudProviderSanitizeAttrs(attrs models.ResAttrs) models.ResAttrs {
	if attrs == nil {
		return nil
	}
	sanitized := models.ResAttrs{}
	for key, value := range attrs {
		sanitized[key] = cloudProviderSanitizeValue(key, value)
	}
	return sanitized
}

func cloudProviderSanitizeValue(key string, value interface{}) interface{} {
	if cloudProviderSensitiveKey(key) && value != nil {
		return "***"
	}
	switch typed := value.(type) {
	case models.ResAttrs:
		return cloudProviderSanitizeAttrs(typed)
	case map[string]interface{}:
		return cloudProviderSanitizeAttrs(models.ResAttrs(typed))
	case map[string]string:
		result := models.ResAttrs{}
		for nestedKey, nestedValue := range typed {
			result[nestedKey] = cloudProviderSanitizeValue(nestedKey, nestedValue)
		}
		return result
	case []models.ResAttrs:
		result := make([]models.ResAttrs, 0, len(typed))
		for _, item := range typed {
			result = append(result, cloudProviderSanitizeAttrs(item))
		}
		return result
	case []interface{}:
		result := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			result = append(result, cloudProviderSanitizeValue("", item))
		}
		return result
	default:
		return value
	}
}

func cloudProviderSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "_", ""))
	for _, token := range []string{"secret", "password", "passwd", "token", "privatekey", "accesskey", "credential", "authorization", "signature"} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}
