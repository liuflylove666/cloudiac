// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cloudiac/portal/models"
)

const (
	azureManagementEndpoint = "https://management.azure.com"
	azureResourceAPIVersion = "2021-04-01"
)

func collectCmdbAzureAssets(account *cmdbCloudAccount, regions, assetTypes []string) cmdbCloudCollectResult {
	stats := models.ResAttrs{
		"regions":    regions,
		"assetTypes": assetTypes,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	token, err := azureAccessToken(ctx, account)
	if err != nil {
		return cmdbCloudCollectResult{Stats: stats, Err: err}
	}
	subscriptionId := azureSubscriptionId(account)
	if subscriptionId == "" {
		return cmdbCloudCollectResult{Stats: stats, Err: fmt.Errorf("azure collector requires AZURE_SUBSCRIPTION_ID")}
	}

	resources, err := azureListResources(ctx, token, subscriptionId)
	if err != nil {
		return cmdbCloudCollectResult{Stats: stats, Err: err}
	}
	resourceIndex := azureResourceIndex(resources)

	selected := selectedCmdbAssetTypes(assetTypes)
	selectedRegions := selectedAzureRegions(regions)
	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0, len(resources))
	skippedTypes := make(map[string]int)
	for _, resource := range resources {
		assetType, nativeType := azureCmdbAssetType(resource.Type)
		if assetType == "" || !wantsCmdbAssetType(selected, assetType) {
			if strings.TrimSpace(resource.Type) != "" {
				skippedTypes[resource.Type]++
			}
			continue
		}
		if len(selectedRegions) > 0 && !selectedRegions[strings.ToLower(resource.Location)] {
			continue
		}
		asset := azureResourceToCmdbAsset(account, resource, assetType, nativeType, resourceIndex, now)
		assets = append(assets, asset)
	}

	stats["collected"] = len(assets)
	stats["totalResources"] = len(resources)
	if len(skippedTypes) > 0 {
		stats["skippedNativeTypes"] = skippedTypes
	}
	return cmdbCloudCollectResult{
		Assets: assets,
		Stats:  stats,
	}
}

type azureTokenResponse struct {
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

func azureAccessToken(ctx context.Context, account *cmdbCloudAccount) (string, error) {
	if token := strings.TrimSpace(firstNonEmpty(account.Credentials["AZURE_ACCESS_TOKEN"], account.Credentials["ARM_ACCESS_TOKEN"])); token != "" {
		return token, nil
	}

	tenantId := firstNonEmpty(account.Credentials["AZURE_TENANT_ID"], account.Credentials["ARM_TENANT_ID"])
	clientId := firstNonEmpty(account.Credentials["AZURE_CLIENT_ID"], account.Credentials["ARM_CLIENT_ID"])
	clientSecret := firstNonEmpty(account.Credentials["AZURE_CLIENT_SECRET"], account.Credentials["ARM_CLIENT_SECRET"])
	if tenantId == "" || clientId == "" || clientSecret == "" {
		return "", fmt.Errorf("azure collector requires AZURE_TENANT_ID, AZURE_CLIENT_ID and AZURE_CLIENT_SECRET or AZURE_ACCESS_TOKEN")
	}

	values := url.Values{}
	values.Set("client_id", clientId)
	values.Set("client_secret", clientSecret)
	values.Set("grant_type", "client_credentials")
	values.Set("scope", azureManagementEndpoint+"/.default")

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", url.PathEscape(tenantId))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp := azureTokenResponse{}
	if err := azureDoJSON(req, "", &resp); err != nil {
		return "", err
	}
	if resp.AccessToken == "" {
		if resp.Error != "" {
			return "", fmt.Errorf("azure token error %s: %s", resp.Error, resp.Description)
		}
		return "", fmt.Errorf("azure token response missing access_token")
	}
	return resp.AccessToken, nil
}

type azureResourceListResponse struct {
	Value    []azureResource `json:"value"`
	NextLink string          `json:"nextLink"`
}

type azureResource struct {
	Id         string          `json:"id"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Location   string          `json:"location"`
	Kind       string          `json:"kind"`
	Tags       models.ResAttrs `json:"tags"`
	Sku        models.ResAttrs `json:"sku"`
	Properties models.ResAttrs `json:"properties"`
}

func azureListResources(ctx context.Context, token, subscriptionId string) ([]azureResource, error) {
	values := url.Values{}
	values.Set("api-version", azureResourceAPIVersion)
	values.Set("$expand", "createdTime,changedTime")
	nextURL := fmt.Sprintf("%s/subscriptions/%s/resources?%s", azureManagementEndpoint, url.PathEscape(subscriptionId), values.Encode())

	resources := make([]azureResource, 0)
	for page := 0; page < 100 && nextURL != ""; page++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, nextURL, nil)
		if err != nil {
			return resources, err
		}
		req.Header.Set("Accept", "application/json")
		resp := azureResourceListResponse{}
		if err := azureDoJSON(req, token, &resp); err != nil {
			return resources, err
		}
		resources = append(resources, resp.Value...)
		nextURL = resp.NextLink
	}
	return resources, nil
}

func azureDoJSON(req *http.Request, token string, out interface{}) error {
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("azure api %s %s status %d: %s", req.Method, req.URL.String(), resp.StatusCode, string(body))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode azure response: %w", err)
	}
	return nil
}

func azureCmdbAssetType(resourceType string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(resourceType)) {
	case "microsoft.compute/virtualmachines":
		return models.CmdbAssetTypeComputeInstance, "azure_virtual_machine"
	case "microsoft.network/virtualnetworks":
		return models.CmdbAssetTypeNetworkVpc, "azure_virtual_network"
	case "microsoft.network/virtualnetworks/subnets":
		return models.CmdbAssetTypeNetworkSubnet, "azure_subnet"
	case "microsoft.network/networksecuritygroups":
		return models.CmdbAssetTypeNetworkSecurityGroup, "azure_network_security_group"
	case "microsoft.network/publicipaddresses":
		return models.CmdbAssetTypePublicIP, "azure_public_ip"
	case "microsoft.compute/disks":
		return models.CmdbAssetTypeBlockVolume, "azure_managed_disk"
	case "microsoft.network/loadbalancers":
		return models.CmdbAssetTypeLoadBalancer, "azure_load_balancer"
	case "microsoft.containerservice/managedclusters":
		return models.CmdbAssetTypeKubernetesCluster, "azure_kubernetes_cluster"
	case "microsoft.sql/servers", "microsoft.sql/servers/databases", "microsoft.dbformysql/flexibleservers", "microsoft.dbforpostgresql/flexibleservers":
		return models.CmdbAssetTypeRelationalDatabase, "azure_database"
	case "microsoft.storage/storageaccounts":
		return models.CmdbAssetTypeObjectStorageBucket, "azure_storage_account"
	default:
		return "", ""
	}
}

func azureResourceToCmdbAsset(account *cmdbCloudAccount, resource azureResource, assetType, nativeType string, resourceIndex map[string]azureResource, now models.Time) *models.CmdbAsset {
	asset := newCmdbCloudAsset(account, resource.Location, assetType, nativeType, firstNonEmpty(resource.Id, resource.Name), firstNonEmpty(resource.Name, resource.Id), now)
	asset.Status = azureResourceStatus(resource)
	asset.Tags = azureTagsToAttrs(resource.Tags)
	asset.Attributes = models.ResAttrs{
		"azureId":           resource.Id,
		"resourceType":      resource.Type,
		"kind":              resource.Kind,
		"resourceGroup":     azureResourceGroup(resource.Id),
		"subscriptionId":    azureSubscriptionId(account),
		"provisioningState": attrString(resource.Properties, "provisioningState"),
		"sku":               resource.Sku,
		"properties":        resource.Properties,
	}
	azureEnrichAssetFromProperties(asset, resource, resourceIndex)
	return asset
}

func azureEnrichAssetFromProperties(asset *models.CmdbAsset, resource azureResource, resourceIndex map[string]azureResource) {
	props := resource.Properties
	switch asset.NativeType {
	case "azure_virtual_machine":
		azureEnrichVirtualMachine(asset, resource, resourceIndex)
	case "azure_public_ip":
		asset.PublicIp = attrString(props, "ipAddress")
		asset.Address = asset.PublicIp
		asset.Attributes["ipAddress"] = asset.PublicIp
		asset.Attributes["ipConfigurationId"] = azureNestedId(props, "ipConfiguration")
		asset.Attributes["publicIPAllocationMethod"] = attrString(props, "publicIPAllocationMethod")
	case "azure_network_security_group":
		ingress, egress := azureSecurityRules(props)
		asset.Attributes["ingressSecurityRules"] = ingress
		asset.Attributes["egressSecurityRules"] = egress
		asset.Attributes["subnetIds"] = azureIDRefsFromObjects(props["subnets"])
		asset.Attributes["networkInterfaceIds"] = azureIDRefsFromObjects(props["networkInterfaces"])
	case "azure_virtual_network":
		asset.Attributes["addressPrefixes"] = azureAddressPrefixes(props)
	case "azure_subnet":
		asset.Attributes["vnetId"] = azureParentResourceId(resource.Id, "virtualNetworks")
		asset.Attributes["addressPrefix"] = firstNonEmpty(attrString(props, "addressPrefix"), strings.Join(azureStringList(props["addressPrefixes"]), ","))
		asset.Attributes["networkSecurityGroupId"] = azureNestedId(props, "networkSecurityGroup")
	case "azure_managed_disk":
		asset.Attributes["sizeGiB"] = attrString(props, "diskSizeGB")
		asset.Attributes["diskState"] = attrString(props, "diskState")
	case "azure_load_balancer":
		asset.Attributes["frontendIPConfigurations"] = props["frontendIPConfigurations"]
		publicIpIds, subnetIds := azureLoadBalancerNetworkRefs(props)
		asset.Attributes["publicIpIds"] = publicIpIds
		asset.Attributes["subnetIds"] = subnetIds
	case "azure_kubernetes_cluster":
		endpoint := firstNonEmpty(attrString(props, "fqdn"), attrString(props, "privateFQDN"))
		subnetIds := azureClusterSubnetIds(props)
		asset.Address = endpoint
		asset.Attributes["fqdn"] = attrString(props, "fqdn")
		asset.Attributes["privateFQDN"] = attrString(props, "privateFQDN")
		asset.Attributes["endpoint"] = endpoint
		asset.Attributes["apiEndpoint"] = endpoint
		asset.Attributes["kubernetesVersion"] = attrString(props, "kubernetesVersion")
		asset.Attributes["version"] = attrString(props, "kubernetesVersion")
		asset.Attributes["subnetIds"] = subnetIds
		asset.Attributes["vnetId"] = azureClusterVNetId(subnetIds)
		asset.Attributes["securityGroupIds"] = azureClusterSecurityGroupIds(subnetIds, resourceIndex)
		asset.Attributes["nodePools"] = azureClusterNodePools(props)
		asset.Attributes["nodePoolCount"] = len(resAttrsSlice(asset.Attributes["nodePools"]))
		asset.Attributes["networkProfile"] = props["networkProfile"]
		asset.Attributes["endpointConfig"] = models.ResAttrs{
			"publicEndpoint":  attrString(props, "fqdn"),
			"privateEndpoint": attrString(props, "privateFQDN"),
		}
	case "azure_database":
		asset.Attributes["engine"] = resource.Type
		asset.Attributes["version"] = firstNonEmpty(attrString(props, "version"), attrString(props, "currentSku"))
		asset.Attributes["subnetIds"] = azureDatabaseSubnetIds(props)
	case "azure_storage_account":
		asset.Attributes["primaryEndpoints"] = props["primaryEndpoints"]
	}
}

func azureEnrichVirtualMachine(asset *models.CmdbAsset, resource azureResource, resourceIndex map[string]azureResource) {
	props := resource.Properties
	networkProfile := modelResAttrs(props["networkProfile"])
	storageProfile := modelResAttrs(props["storageProfile"])
	nicIds := make([]string, 0)
	subnetIds := make([]string, 0)
	securityGroupIds := make([]string, 0)
	publicIpIds := make([]string, 0)
	privateIps := make([]string, 0)
	publicIps := make([]string, 0)

	if networkProfile != nil {
		for _, nicRef := range resAttrsSlice(networkProfile["networkInterfaces"]) {
			nicId := attrString(nicRef, "id")
			if nicId == "" {
				continue
			}
			nicIds = append(nicIds, nicId)
			nic, ok := azureLookupResource(resourceIndex, nicId)
			if !ok {
				continue
			}
			nicProps := nic.Properties
			if sgId := azureNestedId(nicProps, "networkSecurityGroup"); sgId != "" {
				securityGroupIds = append(securityGroupIds, sgId)
			}
			for _, ipConfig := range resAttrsSlice(nicProps["ipConfigurations"]) {
				ipProps := modelResAttrs(ipConfig["properties"])
				if ipProps == nil {
					ipProps = ipConfig
				}
				if subnetId := azureNestedId(ipProps, "subnet"); subnetId != "" {
					subnetIds = append(subnetIds, subnetId)
				}
				if privateIp := attrString(ipProps, "privateIPAddress"); privateIp != "" {
					privateIps = append(privateIps, privateIp)
				}
				if publicIpId := azureNestedId(ipProps, "publicIPAddress"); publicIpId != "" {
					publicIpIds = append(publicIpIds, publicIpId)
					if publicIpResource, ok := azureLookupResource(resourceIndex, publicIpId); ok {
						if publicIp := attrString(publicIpResource.Properties, "ipAddress"); publicIp != "" {
							publicIps = append(publicIps, publicIp)
						}
					}
				}
			}
		}
	}

	asset.PrivateIp = firstNonEmpty(asset.PrivateIp, strings.Join(dedupeStrings(privateIps), ","))
	asset.PublicIp = firstNonEmpty(asset.PublicIp, strings.Join(dedupeStrings(publicIps), ","))
	asset.Address = firstNonEmpty(asset.PublicIp, asset.PrivateIp)
	asset.Attributes["networkInterfaceIds"] = dedupeStrings(nicIds)
	asset.Attributes["subnetIds"] = dedupeStrings(subnetIds)
	asset.Attributes["securityGroupIds"] = dedupeStrings(securityGroupIds)
	asset.Attributes["publicIpIds"] = dedupeStrings(publicIpIds)
	asset.Attributes["diskIds"] = azureVirtualMachineDiskIds(storageProfile)
}

func azureVirtualMachineDiskIds(storageProfile models.ResAttrs) []string {
	if storageProfile == nil {
		return nil
	}
	diskIds := make([]string, 0)
	if osDisk := modelResAttrs(storageProfile["osDisk"]); osDisk != nil {
		if diskId := azureNestedId(osDisk, "managedDisk"); diskId != "" {
			diskIds = append(diskIds, diskId)
		}
	}
	for _, disk := range resAttrsSlice(storageProfile["dataDisks"]) {
		if diskId := azureNestedId(disk, "managedDisk"); diskId != "" {
			diskIds = append(diskIds, diskId)
		}
	}
	return dedupeStrings(diskIds)
}

func azureLoadBalancerNetworkRefs(props models.ResAttrs) ([]string, []string) {
	publicIpIds := make([]string, 0)
	subnetIds := make([]string, 0)
	for _, item := range resAttrsSlice(props["frontendIPConfigurations"]) {
		ipProps := modelResAttrs(item["properties"])
		if ipProps == nil {
			ipProps = item
		}
		if publicIpId := azureNestedId(ipProps, "publicIPAddress"); publicIpId != "" {
			publicIpIds = append(publicIpIds, publicIpId)
		}
		if subnetId := azureNestedId(ipProps, "subnet"); subnetId != "" {
			subnetIds = append(subnetIds, subnetId)
		}
	}
	return dedupeStrings(publicIpIds), dedupeStrings(subnetIds)
}

func azureClusterSubnetIds(props models.ResAttrs) []string {
	subnetIds := make([]string, 0)
	for _, profile := range resAttrsSlice(props["agentPoolProfiles"]) {
		subnetIds = append(subnetIds, attrString(profile, "vnetSubnetID"))
		subnetIds = append(subnetIds, attrString(profile, "podSubnetID"))
	}
	if networkProfile := modelResAttrs(props["networkProfile"]); networkProfile != nil {
		subnetIds = append(subnetIds, attrString(networkProfile, "vnetSubnetID"))
		subnetIds = append(subnetIds, attrString(networkProfile, "podSubnetID"))
	}
	return dedupeStrings(nonEmptyStrings(subnetIds))
}

func azureClusterNodePools(props models.ResAttrs) []models.ResAttrs {
	pools := make([]models.ResAttrs, 0)
	for _, profile := range resAttrsSlice(props["agentPoolProfiles"]) {
		subnetIds := dedupeStrings(nonEmptyStrings([]string{
			attrString(profile, "vnetSubnetID"),
			attrString(profile, "podSubnetID"),
		}))
		scaling := models.ResAttrs{
			"desiredSize": attrInt(profile, "count"),
		}
		if attrString(profile, "enableAutoScaling") != "" {
			scaling["autoScaling"] = attrString(profile, "enableAutoScaling")
		}
		if attrString(profile, "minCount") != "" {
			scaling["minSize"] = attrInt(profile, "minCount")
		}
		if attrString(profile, "maxCount") != "" {
			scaling["maxSize"] = attrInt(profile, "maxCount")
		}
		pools = append(pools, models.ResAttrs{
			"id":                  attrString(profile, "name"),
			"name":                attrString(profile, "name"),
			"status":              attrString(profile, "provisioningState"),
			"mode":                attrString(profile, "mode"),
			"type":                attrString(profile, "type"),
			"version":             firstNonEmpty(attrString(profile, "currentOrchestratorVersion"), attrString(profile, "orchestratorVersion")),
			"kubernetesVersion":   firstNonEmpty(attrString(profile, "currentOrchestratorVersion"), attrString(profile, "orchestratorVersion")),
			"instanceTypes":       nonEmptyStrings([]string{attrString(profile, "vmSize")}),
			"vmSize":              attrString(profile, "vmSize"),
			"nodeCount":           attrInt(profile, "count"),
			"osType":              attrString(profile, "osType"),
			"osSKU":               attrString(profile, "osSKU"),
			"subnetIds":           subnetIds,
			"zones":               azureStringList(profile["availabilityZones"]),
			"scalingConfig":       scaling,
			"maxPods":             attrString(profile, "maxPods"),
			"enableAutoScaling":   attrString(profile, "enableAutoScaling"),
			"enableNodePublicIP":  attrString(profile, "enableNodePublicIP"),
			"nodeTaints":          profile["nodeTaints"],
			"nodeLabels":          profile["nodeLabels"],
			"upgradeSettings":     profile["upgradeSettings"],
			"powerState":          profile["powerState"],
			"rawAgentPoolProfile": profile,
		})
	}
	return pools
}

func azureClusterVNetId(subnetIds []string) string {
	for _, subnetId := range subnetIds {
		if vnetId := azureParentResourceId(subnetId, "virtualNetworks"); vnetId != "" {
			return vnetId
		}
	}
	return ""
}

func azureClusterSecurityGroupIds(subnetIds []string, resourceIndex map[string]azureResource) []string {
	securityGroupIds := make([]string, 0)
	for _, subnetId := range subnetIds {
		subnet, ok := azureLookupResource(resourceIndex, subnetId)
		if !ok {
			continue
		}
		if sgId := azureNestedId(subnet.Properties, "networkSecurityGroup"); sgId != "" {
			securityGroupIds = append(securityGroupIds, sgId)
		}
	}
	return dedupeStrings(nonEmptyStrings(securityGroupIds))
}

func azureDatabaseSubnetIds(props models.ResAttrs) []string {
	subnetIds := []string{
		attrString(props, "delegatedSubnetResourceId"),
		attrString(props, "subnetId"),
		attrString(props, "virtualNetworkSubnetId"),
	}
	if network := modelResAttrs(props["network"]); network != nil {
		subnetIds = append(subnetIds, attrString(network, "delegatedSubnetResourceId"), attrString(network, "subnetId"))
	}
	return dedupeStrings(nonEmptyStrings(subnetIds))
}

func azureResourceIndex(resources []azureResource) map[string]azureResource {
	index := make(map[string]azureResource, len(resources))
	for _, resource := range resources {
		if key := azureResourceKey(resource.Id); key != "" {
			index[key] = resource
		}
	}
	return index
}

func azureLookupResource(index map[string]azureResource, id string) (azureResource, bool) {
	resource, ok := index[azureResourceKey(id)]
	return resource, ok
}

func azureResourceKey(id string) string {
	return strings.ToLower(strings.TrimRight(strings.TrimSpace(id), "/"))
}

func azureNestedId(attrs models.ResAttrs, key string) string {
	child := modelResAttrs(attrs[key])
	if child == nil {
		return ""
	}
	return attrString(child, "id")
}

func azureIDRefsFromObjects(value interface{}) []string {
	refs := make([]string, 0)
	for _, item := range resAttrsSlice(value) {
		if id := attrString(item, "id"); id != "" {
			refs = append(refs, id)
			continue
		}
		if props := modelResAttrs(item["properties"]); props != nil {
			if id := attrString(props, "id"); id != "" {
				refs = append(refs, id)
			}
		}
	}
	return dedupeStrings(refs)
}

func azureParentResourceId(resourceId, parentType string) string {
	parts := strings.Split(resourceId, "/")
	for i := 0; i < len(parts)-1; i++ {
		if strings.EqualFold(parts[i], parentType) {
			return strings.Join(parts[:i+2], "/")
		}
	}
	return ""
}

func azureResourceStatus(resource azureResource) string {
	props := resource.Properties
	return firstNonEmpty(
		attrString(props, "powerState"),
		attrString(props, "status"),
		attrString(props, "state"),
		attrString(props, "provisioningState"),
	)
}

func azureSecurityRules(props models.ResAttrs) ([]models.ResAttrs, []models.ResAttrs) {
	ingress := make([]models.ResAttrs, 0)
	egress := make([]models.ResAttrs, 0)
	for _, raw := range append(azureRuleList(props["securityRules"]), azureRuleList(props["defaultSecurityRules"])...) {
		ruleProps := modelResAttrs(raw["properties"])
		if ruleProps == nil {
			ruleProps = raw
		}
		rule := models.ResAttrs{
			"name":              attrString(raw, "name"),
			"protocol":          attrString(ruleProps, "protocol"),
			"source":            firstNonEmpty(attrString(ruleProps, "sourceAddressPrefix"), strings.Join(azureStringList(ruleProps["sourceAddressPrefixes"]), ",")),
			"destination":       firstNonEmpty(attrString(ruleProps, "destinationAddressPrefix"), strings.Join(azureStringList(ruleProps["destinationAddressPrefixes"]), ",")),
			"portRange":         firstNonEmpty(attrString(ruleProps, "destinationPortRange"), strings.Join(azureStringList(ruleProps["destinationPortRanges"]), ",")),
			"sourcePortRange":   firstNonEmpty(attrString(ruleProps, "sourcePortRange"), strings.Join(azureStringList(ruleProps["sourcePortRanges"]), ",")),
			"access":            attrString(ruleProps, "access"),
			"priority":          attrString(ruleProps, "priority"),
			"description":       attrString(ruleProps, "description"),
			"provisioningState": attrString(ruleProps, "provisioningState"),
			"raw":               raw,
		}
		if strings.EqualFold(attrString(ruleProps, "direction"), "outbound") {
			egress = append(egress, rule)
		} else {
			ingress = append(ingress, rule)
		}
	}
	return ingress, egress
}

func azureRuleList(value interface{}) []models.ResAttrs {
	items := make([]models.ResAttrs, 0)
	switch typed := value.(type) {
	case []models.ResAttrs:
		return typed
	case []interface{}:
		for _, item := range typed {
			if attrs := modelResAttrs(item); attrs != nil {
				items = append(items, attrs)
			}
		}
	}
	return items
}

func azureAddressPrefixes(props models.ResAttrs) []string {
	addressSpace := modelResAttrs(props["addressSpace"])
	if addressSpace == nil {
		return nil
	}
	return azureStringList(addressSpace["addressPrefixes"])
}

func azureStringList(value interface{}) []string {
	result := make([]string, 0)
	switch typed := value.(type) {
	case []string:
		return typed
	case []interface{}:
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprintf("%v", item)); text != "" && text != "<nil>" {
				result = append(result, text)
			}
		}
	case string:
		if strings.TrimSpace(typed) != "" {
			result = append(result, strings.TrimSpace(typed))
		}
	}
	return result
}

func nonEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func azureTagsToAttrs(tags models.ResAttrs) models.ResAttrs {
	if tags == nil {
		return models.ResAttrs{}
	}
	result := models.ResAttrs{}
	for key, value := range tags {
		result[key] = fmt.Sprintf("%v", value)
	}
	return result
}

func azureResourceGroup(resourceId string) string {
	parts := strings.Split(resourceId, "/")
	for i := 0; i < len(parts)-1; i++ {
		if strings.EqualFold(parts[i], "resourceGroups") {
			return parts[i+1]
		}
	}
	return ""
}

func azureSubscriptionId(account *cmdbCloudAccount) string {
	if account == nil {
		return ""
	}
	return firstNonEmpty(account.AccountId, account.Credentials["AZURE_SUBSCRIPTION_ID"], account.Credentials["ARM_SUBSCRIPTION_ID"])
}

func selectedAzureRegions(regions []string) map[string]bool {
	selected := make(map[string]bool)
	for _, region := range regions {
		region = strings.ToLower(strings.TrimSpace(region))
		if region == "" {
			continue
		}
		selected[region] = true
	}
	return selected
}
