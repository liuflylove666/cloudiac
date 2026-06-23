// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"fmt"
	"strings"

	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models"
)

type cmdbCloudRelationIndex struct {
	assets []models.CmdbAsset
	byKey  map[string][]models.CmdbAsset
}

func rebuildCmdbCloudInferredRelations(c *ctx.ServiceContext, account *cmdbCloudAccount) (int, e.Error) {
	if c == nil || account == nil {
		return 0, nil
	}
	provider := normalizeProvider(account.Provider)
	if provider == "" {
		return 0, nil
	}

	assets := make([]models.CmdbAsset, 0)
	query := c.DB().Model(&models.CmdbAsset{}).
		Where("org_id = ? and source = ? and provider = ?", c.OrgId, models.CmdbAssetSourceCloudCollect, provider)
	accountId := firstNonEmpty(account.AccountId, string(account.Id))
	if account.Source == models.CmdbCloudAccountSourceCloudAccount && account.Id != "" {
		query = query.Where("(cloud_account_id = ? or account_id = ?)", account.Id, accountId)
	} else if accountId != "" {
		query = query.Where("account_id = ?", accountId)
	}
	if err := query.Find(&assets); err != nil {
		return 0, e.New(e.DBError, err)
	}
	if len(assets) == 0 {
		return 0, nil
	}

	assetIds := make([]models.Id, 0, len(assets))
	for _, asset := range assets {
		assetIds = append(assetIds, asset.Id)
	}
	if _, err := c.DB().Where("org_id = ? and source = ? and (source_asset_id in (?) or target_asset_id in (?))",
		c.OrgId, models.CmdbRelationSourceCloudInferred, assetIds, assetIds).Delete(&models.CmdbAssetRelation{}); err != nil {
		return 0, e.New(e.DBError, err)
	}

	index := newCmdbCloudRelationIndex(assets)
	relations := inferCmdbCloudRelations(c.OrgId, provider, accountId, assets, index)
	for idx := range relations {
		if relations[idx].Id == "" {
			relations[idx].Id = models.NewId("cir")
		}
		if err := models.Create(c.DB(), &relations[idx]); err != nil {
			return idx, e.New(e.DBError, err)
		}
	}
	return len(relations), nil
}

func newCmdbCloudRelationIndex(assets []models.CmdbAsset) *cmdbCloudRelationIndex {
	index := &cmdbCloudRelationIndex{
		assets: assets,
		byKey:  make(map[string][]models.CmdbAsset),
	}
	for _, asset := range assets {
		index.add(asset.NativeId, asset)
		index.add(asset.Name, asset)
		index.add(asset.PublicIp, asset)
		index.add(asset.PrivateIp, asset)
		for _, key := range []string{"selfLink", "azureId", "id", "uid", "name", "ipAddress", "publicIp", "publicIpAddress", "allocationId", "vcnId", "vpcId", "subnetId", "routeTableId", "clusterId", "clusterName", "namespace", "namespaceId", "namespaceName", "ownerUid", "ownerName"} {
			index.add(attrString(asset.Attributes, key), asset)
		}
	}
	return index
}

func (idx *cmdbCloudRelationIndex) add(value string, asset models.CmdbAsset) {
	for _, key := range cmdbCloudRelationKeys(value) {
		idx.byKey[key] = append(idx.byKey[key], asset)
	}
}

func (idx *cmdbCloudRelationIndex) find(ref string, assetTypes ...string) (models.CmdbAsset, bool) {
	typeSet := make(map[string]bool, len(assetTypes))
	for _, assetType := range assetTypes {
		typeSet[assetType] = true
	}
	for _, key := range cmdbCloudRelationKeys(ref) {
		for _, asset := range idx.byKey[key] {
			if len(typeSet) > 0 && !typeSet[asset.AssetType] {
				continue
			}
			return asset, true
		}
	}
	return models.CmdbAsset{}, false
}

func cmdbCloudRelationKeys(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	trimmed := strings.TrimRight(value, "/")
	keys := []string{strings.ToLower(trimmed)}
	if last := cmdbCloudLastPathSegment(trimmed); last != "" && !strings.EqualFold(last, trimmed) {
		keys = append(keys, strings.ToLower(last))
	}
	return dedupeStrings(keys)
}

func inferCmdbCloudRelations(orgId models.Id, provider, accountId string, assets []models.CmdbAsset, index *cmdbCloudRelationIndex) []models.CmdbAssetRelation {
	builder := &cmdbCloudRelationBuilder{
		orgId:     orgId,
		provider:  provider,
		accountId: accountId,
		index:     index,
		seen:      make(map[string]bool),
		relations: make([]models.CmdbAssetRelation, 0),
	}
	for _, asset := range assets {
		builder.inferForAsset(asset)
	}
	return builder.relations
}

type cmdbCloudRelationBuilder struct {
	orgId     models.Id
	provider  string
	accountId string
	index     *cmdbCloudRelationIndex
	seen      map[string]bool
	relations []models.CmdbAssetRelation
}

func (b *cmdbCloudRelationBuilder) inferForAsset(asset models.CmdbAsset) {
	switch asset.AssetType {
	case models.CmdbAssetTypeNetworkVpc:
		b.addContainsRefs(asset, asset, []string{"subnetworks", "subnetIds"}, []string{models.CmdbAssetTypeNetworkSubnet}, "vpc_subnets")
	case models.CmdbAssetTypeNetworkSubnet:
		b.addReverseContainsRefs(asset, []string{"vnetId", "vpcId", "vcnId", "network", "networkId"}, []string{models.CmdbAssetTypeNetworkVpc}, "subnet_parent_network")
		b.addDependsOnRefs(asset, []string{"networkSecurityGroupId", "networkSecurityGroupIds", "securityGroupId", "securityGroupIds", "securityListIds", "nsgIds"}, []string{models.CmdbAssetTypeNetworkSecurityGroup}, "subnet_security")
	case models.CmdbAssetTypeNetworkRouteTable:
		b.addReverseContainsRefs(asset, []string{"vpcId", "vcnId", "network", "networkId"}, []string{models.CmdbAssetTypeNetworkVpc}, "route_table_parent_network")
		b.addDependsOnRefs(asset, []string{"subnetIds"}, []string{models.CmdbAssetTypeNetworkSubnet}, "route_table_subnets")
		b.addDependsOnRefs(asset, []string{"natGatewayIds"}, []string{models.CmdbAssetTypeNetworkNatGateway}, "route_table_nat_gateway")
		b.addDependsOnRefs(asset, []string{"internetGatewayIds", "gatewayIds"}, []string{models.CmdbAssetTypeNetworkInternetGateway}, "route_table_internet_gateway")
		b.addDependsOnRefs(asset, []string{"serviceGatewayIds"}, []string{models.CmdbAssetTypeNetworkServiceGateway}, "route_table_service_gateway")
		b.addDependsOnRefs(asset, []string{"drgIds"}, []string{models.CmdbAssetTypeNetworkDrg}, "route_table_drg")
	case models.CmdbAssetTypeNetworkNatGateway:
		b.addReverseContainsRefs(asset, []string{"vpcId", "vcnId", "network", "networkId"}, []string{models.CmdbAssetTypeNetworkVpc}, "nat_gateway_parent_network")
		b.addReverseContainsRefs(asset, []string{"subnetId", "subnetIds"}, []string{models.CmdbAssetTypeNetworkSubnet}, "nat_gateway_subnet")
		b.addDependsOnRefs(asset, []string{"publicIpIds", "allocationIds"}, []string{models.CmdbAssetTypePublicIP}, "nat_gateway_public_ip")
	case models.CmdbAssetTypeNetworkInternetGateway:
		b.addReverseContainsRefs(asset, []string{"vpcId", "vpcIds", "vcnId", "vcnIds", "network", "networkId"}, []string{models.CmdbAssetTypeNetworkVpc}, "internet_gateway_parent_network")
	case models.CmdbAssetTypeNetworkServiceGateway:
		b.addReverseContainsRefs(asset, []string{"vpcId", "vpcIds", "vcnId", "vcnIds", "network", "networkId"}, []string{models.CmdbAssetTypeNetworkVpc}, "service_gateway_parent_network")
	case models.CmdbAssetTypeNetworkDrg:
		b.addReverseContainsRefs(asset, []string{"vpcId", "vpcIds", "vcnId", "vcnIds", "network", "networkId", "attachedVcnIds"}, []string{models.CmdbAssetTypeNetworkVpc}, "drg_parent_network")
	case models.CmdbAssetTypeNetworkSecurityGroup:
		b.addReverseContainsRefs(asset, []string{"vpcId", "vcnId", "network", "networkId"}, []string{models.CmdbAssetTypeNetworkVpc}, "security_parent_network")
		b.addDependsOnRefs(asset, []string{"subnetIds"}, []string{models.CmdbAssetTypeNetworkSubnet}, "security_subnets")
	case models.CmdbAssetTypeComputeInstance:
		b.addReverseContainsRefs(asset, []string{"networkIds", "vpcIds", "vpcId", "vcnIds", "vcnId", "network", "networkId"}, []string{models.CmdbAssetTypeNetworkVpc}, "compute_network")
		b.addReverseContainsRefs(asset, []string{"subnetIds", "subnetwork", "subnetworkIds", "subnetId"}, []string{models.CmdbAssetTypeNetworkSubnet}, "compute_subnet")
		b.addDependsOnRefs(asset, []string{"securityGroupIds", "securityGroupId", "networkSecurityGroupId", "networkSecurityGroupIds", "nsgIds", "securityListIds"}, []string{models.CmdbAssetTypeNetworkSecurityGroup}, "compute_security")
		b.addDependsOnRefs(asset, []string{"publicIpIds", "publicIpId", "publicIPAddressId"}, []string{models.CmdbAssetTypePublicIP}, "compute_public_ip")
		b.addDependsOnRefs(asset, []string{"diskIds", "diskId", "volumeIds", "volumeId"}, []string{models.CmdbAssetTypeBlockVolume}, "compute_storage")
	case models.CmdbAssetTypeBlockVolume:
		b.addReverseDependsOnRefs(asset, []string{"users", "attachedInstanceIds", "instanceIds"}, []string{models.CmdbAssetTypeComputeInstance}, "storage_users")
	case models.CmdbAssetTypeLoadBalancer:
		b.addReverseContainsRefs(asset, []string{"networkIds", "vpcIds", "vpcId", "vcnIds", "vcnId", "network", "networkId"}, []string{models.CmdbAssetTypeNetworkVpc}, "lb_network")
		b.addReverseContainsRefs(asset, []string{"subnetIds", "subnetwork", "subnetworkIds", "subnetId"}, []string{models.CmdbAssetTypeNetworkSubnet}, "lb_subnet")
		b.addDependsOnRefs(asset, []string{"securityGroupIds", "securityGroupId", "networkSecurityGroupId", "networkSecurityGroupIds", "nsgIds"}, []string{models.CmdbAssetTypeNetworkSecurityGroup}, "lb_security")
		b.addDependsOnRefs(asset, []string{"publicIpIds", "publicIpId", "publicIPAddressId"}, []string{models.CmdbAssetTypePublicIP}, "lb_public_ip")
		b.addDependsOnRefs(asset, []string{"targetInstanceIds", "targetIds", "targetIpAddresses"}, []string{models.CmdbAssetTypeComputeInstance}, "lb_targets")
	case models.CmdbAssetTypeKubernetesCluster:
		b.addReverseContainsRefs(asset, []string{"network", "networkId", "networkIds", "vpcId", "vpcIds", "vcnId", "vcnIds"}, []string{models.CmdbAssetTypeNetworkVpc}, "kubernetes_network")
		b.addReverseContainsRefs(asset, []string{"subnetIds", "subnetwork", "subnetworkIds", "subnetId"}, []string{models.CmdbAssetTypeNetworkSubnet}, "kubernetes_subnet")
		b.addDependsOnRefs(asset, []string{"securityGroupIds", "securityGroupId", "networkSecurityGroupId", "networkSecurityGroupIds", "nsgIds", "securityListIds"}, []string{models.CmdbAssetTypeNetworkSecurityGroup}, "kubernetes_security")
		b.addContainsRefs(asset, asset, []string{"namespaceIds", "namespaces"}, []string{models.CmdbAssetTypeKubernetesNamespace}, "kubernetes_namespaces")
		b.addContainsRefs(asset, asset, []string{"nodeIds", "nodes"}, []string{models.CmdbAssetTypeKubernetesNode}, "kubernetes_nodes")
		b.addContainsRefs(asset, asset, []string{"workloadIds", "workloads", "deployments", "statefulSets", "daemonSets", "replicaSets", "jobs", "cronJobs"}, []string{models.CmdbAssetTypeKubernetesWorkload}, "kubernetes_workloads")
		b.addContainsRefs(asset, asset, []string{"podIds", "pods"}, []string{models.CmdbAssetTypeKubernetesPod}, "kubernetes_pods")
		b.addContainsRefs(asset, asset, []string{"serviceIds", "services"}, []string{models.CmdbAssetTypeKubernetesService}, "kubernetes_services")
		b.addContainsRefs(asset, asset, []string{"ingressIds", "ingresses"}, []string{models.CmdbAssetTypeKubernetesIngress}, "kubernetes_ingresses")
	case models.CmdbAssetTypeKubernetesNamespace:
		b.addReverseContainsRefs(asset, []string{"clusterId", "clusterName"}, []string{models.CmdbAssetTypeKubernetesCluster}, "kubernetes_namespace_cluster")
		b.addContainsRefs(asset, asset, []string{"workloadIds", "workloads", "deployments", "statefulSets", "daemonSets", "replicaSets", "jobs", "cronJobs"}, []string{models.CmdbAssetTypeKubernetesWorkload}, "kubernetes_namespace_workloads")
		b.addContainsRefs(asset, asset, []string{"podIds", "pods"}, []string{models.CmdbAssetTypeKubernetesPod}, "kubernetes_namespace_pods")
		b.addContainsRefs(asset, asset, []string{"serviceIds", "services"}, []string{models.CmdbAssetTypeKubernetesService}, "kubernetes_namespace_services")
		b.addContainsRefs(asset, asset, []string{"ingressIds", "ingresses"}, []string{models.CmdbAssetTypeKubernetesIngress}, "kubernetes_namespace_ingresses")
	case models.CmdbAssetTypeKubernetesNode:
		b.addReverseContainsRefs(asset, []string{"clusterId", "clusterName"}, []string{models.CmdbAssetTypeKubernetesCluster}, "kubernetes_node_cluster")
	case models.CmdbAssetTypeKubernetesWorkload:
		b.addReverseContainsRefs(asset, []string{"clusterId", "clusterName"}, []string{models.CmdbAssetTypeKubernetesCluster}, "kubernetes_workload_cluster")
		b.addReverseContainsRefs(asset, []string{"namespaceId", "namespace", "namespaceName"}, []string{models.CmdbAssetTypeKubernetesNamespace}, "kubernetes_workload_namespace")
		b.addContainsRefs(asset, asset, []string{"podIds", "pods"}, []string{models.CmdbAssetTypeKubernetesPod}, "kubernetes_workload_pods")
	case models.CmdbAssetTypeKubernetesPod:
		b.addReverseContainsRefs(asset, []string{"clusterId", "clusterName"}, []string{models.CmdbAssetTypeKubernetesCluster}, "kubernetes_pod_cluster")
		b.addReverseContainsRefs(asset, []string{"namespaceId", "namespace", "namespaceName"}, []string{models.CmdbAssetTypeKubernetesNamespace}, "kubernetes_pod_namespace")
		b.addReverseContainsRefs(asset, []string{"ownerUid", "ownerName", "workloadId", "workloadName"}, []string{models.CmdbAssetTypeKubernetesWorkload}, "kubernetes_pod_owner")
		b.addDependsOnRefs(asset, []string{"nodeId", "nodeName"}, []string{models.CmdbAssetTypeKubernetesNode}, "kubernetes_pod_node")
	case models.CmdbAssetTypeKubernetesService:
		b.addReverseContainsRefs(asset, []string{"clusterId", "clusterName"}, []string{models.CmdbAssetTypeKubernetesCluster}, "kubernetes_service_cluster")
		b.addReverseContainsRefs(asset, []string{"namespaceId", "namespace", "namespaceName"}, []string{models.CmdbAssetTypeKubernetesNamespace}, "kubernetes_service_namespace")
		b.addDependsOnRefs(asset, []string{"workloadIds", "workloadNames", "targetWorkloads", "selectorWorkloads"}, []string{models.CmdbAssetTypeKubernetesWorkload}, "kubernetes_service_workloads")
		b.addDependsOnRefs(asset, []string{"podIds", "podNames", "targetPods", "endpoints"}, []string{models.CmdbAssetTypeKubernetesPod}, "kubernetes_service_pods")
	case models.CmdbAssetTypeKubernetesIngress:
		b.addReverseContainsRefs(asset, []string{"clusterId", "clusterName"}, []string{models.CmdbAssetTypeKubernetesCluster}, "kubernetes_ingress_cluster")
		b.addReverseContainsRefs(asset, []string{"namespaceId", "namespace", "namespaceName"}, []string{models.CmdbAssetTypeKubernetesNamespace}, "kubernetes_ingress_namespace")
		b.addDependsOnRefs(asset, []string{"serviceIds", "serviceNames", "backendServices", "rules"}, []string{models.CmdbAssetTypeKubernetesService}, "kubernetes_ingress_services")
	case models.CmdbAssetTypeRelationalDatabase, models.CmdbAssetTypeRedisCache:
		b.addReverseContainsRefs(asset, []string{"network", "networkId", "networkIds", "vpcId", "vpcIds", "vcnId", "vcnIds"}, []string{models.CmdbAssetTypeNetworkVpc}, "data_network")
		b.addReverseContainsRefs(asset, []string{"subnetIds", "subnetwork", "subnetworkIds", "subnetId", "vSwitchId"}, []string{models.CmdbAssetTypeNetworkSubnet}, "data_subnet")
		b.addDependsOnRefs(asset, []string{"securityGroupIds", "securityGroupId", "networkSecurityGroupId", "networkSecurityGroupIds", "nsgIds", "securityListIds"}, []string{models.CmdbAssetTypeNetworkSecurityGroup}, "data_security")
	}
}

func (b *cmdbCloudRelationBuilder) addContainsRefs(source models.CmdbAsset, owner models.CmdbAsset, keys []string, targetTypes []string, inferredBy string) {
	for _, ref := range cmdbCloudAttrRefs(source, keys...) {
		if target, ok := b.index.find(ref, targetTypes...); ok {
			b.add(owner, target, models.CmdbRelationTypeContains, inferredBy, ref)
		}
	}
}

func (b *cmdbCloudRelationBuilder) addReverseContainsRefs(child models.CmdbAsset, keys []string, parentTypes []string, inferredBy string) {
	for _, ref := range cmdbCloudAttrRefs(child, keys...) {
		if parent, ok := b.index.find(ref, parentTypes...); ok {
			b.add(parent, child, models.CmdbRelationTypeContains, inferredBy, ref)
		}
	}
}

func (b *cmdbCloudRelationBuilder) addDependsOnRefs(source models.CmdbAsset, keys []string, targetTypes []string, inferredBy string) {
	for _, ref := range cmdbCloudAttrRefs(source, keys...) {
		if target, ok := b.index.find(ref, targetTypes...); ok {
			b.add(source, target, models.CmdbRelationTypeDependsOn, inferredBy, ref)
		}
	}
}

func (b *cmdbCloudRelationBuilder) addReverseDependsOnRefs(target models.CmdbAsset, keys []string, sourceTypes []string, inferredBy string) {
	for _, ref := range cmdbCloudAttrRefs(target, keys...) {
		if source, ok := b.index.find(ref, sourceTypes...); ok {
			b.add(source, target, models.CmdbRelationTypeDependsOn, inferredBy, ref)
		}
	}
}

func (b *cmdbCloudRelationBuilder) add(source, target models.CmdbAsset, relationType, inferredBy, ref string) {
	if source.Id == "" || target.Id == "" || source.Id == target.Id {
		return
	}
	key := cmdbAssetRelationKey(source.Id, target.Id, relationType, models.CmdbRelationSourceCloudInferred)
	if b.seen[key] {
		return
	}
	b.seen[key] = true
	b.relations = append(b.relations, models.CmdbAssetRelation{
		OrgId:         b.orgId,
		SourceAssetId: source.Id,
		TargetAssetId: target.Id,
		RelationType:  relationType,
		Source:        models.CmdbRelationSourceCloudInferred,
		Metadata: models.ResAttrs{
			"provider":       b.provider,
			"accountId":      b.accountId,
			"inferredBy":     inferredBy,
			"ref":            ref,
			"sourceNativeId": source.NativeId,
			"targetNativeId": target.NativeId,
		},
	})
}

func cmdbCloudAttrRefs(asset models.CmdbAsset, keys ...string) []string {
	refs := make([]string, 0)
	for _, key := range keys {
		refs = append(refs, cmdbCloudStringRefs(asset.Attributes[key])...)
	}
	return dedupeStrings(nonEmptyStrings(refs))
}

func cmdbCloudStringRefs(value interface{}) []string {
	refs := make([]string, 0)
	switch typed := value.(type) {
	case nil:
		return refs
	case string:
		return splitListValue(typed)
	case []string:
		return typed
	case []interface{}:
		for _, item := range typed {
			refs = append(refs, cmdbCloudStringRefs(item)...)
		}
	case []models.ResAttrs:
		for _, item := range typed {
			refs = append(refs, cmdbCloudObjectRefs(item)...)
		}
	case map[string]interface{}:
		refs = append(refs, cmdbCloudObjectRefs(models.ResAttrs(typed))...)
	case models.ResAttrs:
		refs = append(refs, cmdbCloudObjectRefs(typed)...)
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", typed))
		if text != "" && text != "<nil>" {
			refs = append(refs, text)
		}
	}
	return nonEmptyStrings(refs)
}

func cmdbCloudObjectRefs(attrs models.ResAttrs) []string {
	refs := make([]string, 0)
	for _, key := range []string{"id", "selfLink", "uid", "name", "namespace", "namespaceId", "namespaceName", "network", "networkId", "vpcId", "vcnId", "subnetwork", "subnetId", "routeTableId", "networkEntityId", "serviceId", "clusterId", "clusterName", "ownerUid", "ownerName", "workloadId", "workloadName", "nodeId", "nodeName", "serviceName"} {
		refs = append(refs, attrString(attrs, key))
	}
	return refs
}

func cmdbCloudLastPathSegment(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "/")
	return strings.TrimSpace(parts[len(parts)-1])
}
