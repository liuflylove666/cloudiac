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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cloudiac/portal/models"
)

const (
	gcpComputeEndpoint   = "https://compute.googleapis.com/compute/v1"
	gcpContainerEndpoint = "https://container.googleapis.com/v1"
	gcpSqlAdminEndpoint  = "https://sqladmin.googleapis.com/sql/v1beta4"
	gcpStorageEndpoint   = "https://storage.googleapis.com/storage/v1"
	gcpOAuthScope        = "https://www.googleapis.com/auth/cloud-platform"
	gcpDefaultTokenURI   = "https://oauth2.googleapis.com/token"
)

func collectCmdbGcpAssets(account *cmdbCloudAccount, regions, assetTypes []string) cmdbCloudCollectResult {
	stats := models.ResAttrs{
		"regions":    regions,
		"assetTypes": assetTypes,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	token, err := gcpAccessToken(ctx, account)
	if err != nil {
		return cmdbCloudCollectResult{Stats: stats, Err: err}
	}
	projectId := gcpProjectId(account)
	if projectId == "" {
		return cmdbCloudCollectResult{Stats: stats, Err: fmt.Errorf("gcp collector requires GCP_PROJECT_ID")}
	}

	selected := selectedCmdbAssetTypes(assetTypes)
	selectedRegions := selectedGcpRegions(regions)
	assets := make([]*models.CmdbAsset, 0)
	errors := make([]string, 0)

	collect := func(assetType, label string, fn func(context.Context, *cmdbCloudAccount, string, string, map[string]bool) ([]*models.CmdbAsset, error)) {
		if !wantsCmdbAssetType(selected, assetType) {
			return
		}
		items, err := fn(ctx, account, token, projectId, selectedRegions)
		assets = append(assets, items...)
		if err != nil {
			errors = append(errors, fmt.Sprintf("gcp %s: %v", label, err))
		}
	}

	collect(models.CmdbAssetTypeComputeInstance, "compute instances", collectGcpInstances)
	collect(models.CmdbAssetTypeNetworkVpc, "networks", collectGcpNetworks)
	collect(models.CmdbAssetTypeNetworkSubnet, "subnetworks", collectGcpSubnetworks)
	collect(models.CmdbAssetTypeNetworkSecurityGroup, "firewalls", collectGcpFirewalls)
	collect(models.CmdbAssetTypeBlockVolume, "disks", collectGcpDisks)
	collect(models.CmdbAssetTypeLoadBalancer, "forwarding rules", collectGcpForwardingRules)
	collect(models.CmdbAssetTypeKubernetesCluster, "gke clusters", collectGcpClusters)
	collect(models.CmdbAssetTypeRelationalDatabase, "cloud sql instances", collectGcpSqlInstances)
	collect(models.CmdbAssetTypeObjectStorageBucket, "storage buckets", collectGcpBuckets)

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

func gcpAccessToken(ctx context.Context, account *cmdbCloudAccount) (string, error) {
	if token := strings.TrimSpace(account.Credentials["GCP_ACCESS_TOKEN"]); token != "" {
		return token, nil
	}
	rawJSON := strings.TrimSpace(account.Credentials["GCP_SERVICE_ACCOUNT_JSON"])
	if rawJSON == "" {
		return "", fmt.Errorf("gcp collector requires GCP_ACCESS_TOKEN or GCP_SERVICE_ACCOUNT_JSON")
	}
	var svc gcpServiceAccountKey
	if err := json.Unmarshal([]byte(rawJSON), &svc); err != nil {
		return "", fmt.Errorf("decode GCP_SERVICE_ACCOUNT_JSON: %w", err)
	}
	if svc.ClientEmail == "" || svc.PrivateKey == "" {
		return "", fmt.Errorf("GCP_SERVICE_ACCOUNT_JSON requires client_email and private_key")
	}
	tokenURI := firstNonEmpty(svc.TokenURI, gcpDefaultTokenURI)
	assertion, err := gcpServiceAccountAssertion(svc, tokenURI)
	if err != nil {
		return "", err
	}
	values := url.Values{}
	values.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	values.Set("assertion", assertion)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURI, strings.NewReader(values.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp := gcpTokenResponse{}
	if err := gcpDoJSON(req, "", &resp); err != nil {
		return "", err
	}
	if resp.AccessToken == "" {
		return "", fmt.Errorf("gcp token response missing access_token")
	}
	return resp.AccessToken, nil
}

type gcpServiceAccountKey struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

type gcpTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

func gcpServiceAccountAssertion(svc gcpServiceAccountKey, tokenURI string) (string, error) {
	now := time.Now().Unix()
	header := models.ResAttrs{"alg": "RS256", "typ": "JWT"}
	claim := models.ResAttrs{
		"iss":   svc.ClientEmail,
		"scope": gcpOAuthScope,
		"aud":   tokenURI,
		"iat":   now,
		"exp":   now + 3600,
	}
	unsigned, err := gcpJWTUnsigned(header, claim)
	if err != nil {
		return "", err
	}
	privateKey, err := gcpParsePrivateKey(svc.PrivateKey)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func gcpJWTUnsigned(header, claim models.ResAttrs) (string, error) {
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimJSON, err := json.Marshal(claim)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimJSON), nil
}

func gcpParsePrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("decode gcp private key pem")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, fmt.Errorf("gcp private key is not RSA")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func gcpDoJSON(req *http.Request, token string, out interface{}) error {
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
		return fmt.Errorf("gcp api %s %s status %d: %s", req.Method, req.URL.String(), resp.StatusCode, string(body))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode gcp response: %w", err)
	}
	return nil
}

type gcpInstance struct {
	Id                json.Number       `json:"id"`
	Name              string            `json:"name"`
	Status            string            `json:"status"`
	Zone              string            `json:"zone"`
	MachineType       string            `json:"machineType"`
	SelfLink          string            `json:"selfLink"`
	CreationTimestamp string            `json:"creationTimestamp"`
	Labels            map[string]string `json:"labels"`
	NetworkInterfaces []gcpNetworkIface `json:"networkInterfaces"`
	Disks             []models.ResAttrs `json:"disks"`
	Tags              models.ResAttrs   `json:"tags"`
	Metadata          models.ResAttrs   `json:"metadata"`
}

type gcpNetworkIface struct {
	Network       string            `json:"network"`
	Subnetwork    string            `json:"subnetwork"`
	NetworkIP     string            `json:"networkIP"`
	AccessConfigs []gcpAccessConfig `json:"accessConfigs"`
}

type gcpAccessConfig struct {
	NatIP string `json:"natIP"`
	Type  string `json:"type"`
	Name  string `json:"name"`
}

func collectGcpInstances(ctx context.Context, account *cmdbCloudAccount, token, projectId string, selectedRegions map[string]bool) ([]*models.CmdbAsset, error) {
	resp := struct {
		Items map[string]struct {
			Instances []gcpInstance `json:"instances"`
		} `json:"items"`
		NextPageToken string `json:"nextPageToken"`
	}{}
	if err := gcpGet(ctx, token, fmt.Sprintf("%s/projects/%s/aggregated/instances", gcpComputeEndpoint, url.PathEscape(projectId)), &resp); err != nil {
		return nil, err
	}
	assets := make([]*models.CmdbAsset, 0)
	now := cmdbNow()
	for _, scoped := range resp.Items {
		for _, instance := range scoped.Instances {
			region := gcpRegionFromZone(instance.Zone)
			if !gcpRegionSelected(selectedRegions, region) {
				continue
			}
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeComputeInstance, "gcp_compute_instance", firstNonEmpty(instance.SelfLink, instance.Id.String(), instance.Name), instance.Name, now)
			asset.Zone = gcpLastPathSegment(instance.Zone)
			asset.Status = instance.Status
			asset.PrivateIp, asset.PublicIp = gcpInstanceIPs(instance)
			asset.Tags = gcpLabelsToAttrs(instance.Labels)
			asset.Attributes = models.ResAttrs{
				"machineType":       gcpLastPathSegment(instance.MachineType),
				"selfLink":          instance.SelfLink,
				"creationTimestamp": instance.CreationTimestamp,
				"networkInterfaces": instance.NetworkInterfaces,
				"networkIds":        gcpInstanceNetworkRefs(instance),
				"subnetIds":         gcpInstanceSubnetworkRefs(instance),
				"publicIpAddresses": gcpInstancePublicIPs(instance),
				"diskIds":           gcpInstanceDiskSources(instance),
				"disks":             instance.Disks,
				"networkTags":       instance.Tags,
				"metadata":          instance.Metadata,
			}
			assets = append(assets, asset)
		}
	}
	return assets, nil
}

type gcpNetwork struct {
	Id                    json.Number       `json:"id"`
	Name                  string            `json:"name"`
	SelfLink              string            `json:"selfLink"`
	AutoCreateSubnetworks bool              `json:"autoCreateSubnetworks"`
	IPv4Range             string            `json:"IPv4Range"`
	Subnetworks           []string          `json:"subnetworks"`
	RoutingConfig         models.ResAttrs   `json:"routingConfig"`
	Labels                map[string]string `json:"labels"`
}

func collectGcpNetworks(ctx context.Context, account *cmdbCloudAccount, token, projectId string, selectedRegions map[string]bool) ([]*models.CmdbAsset, error) {
	resp := struct {
		Items []gcpNetwork `json:"items"`
	}{}
	if err := gcpGet(ctx, token, fmt.Sprintf("%s/projects/%s/global/networks", gcpComputeEndpoint, url.PathEscape(projectId)), &resp); err != nil {
		return nil, err
	}
	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0, len(resp.Items))
	for _, network := range resp.Items {
		asset := newCmdbCloudAsset(account, "global", models.CmdbAssetTypeNetworkVpc, "gcp_compute_network", firstNonEmpty(network.SelfLink, network.Id.String(), network.Name), network.Name, now)
		asset.Tags = gcpLabelsToAttrs(network.Labels)
		asset.Attributes = models.ResAttrs{
			"selfLink":              network.SelfLink,
			"autoCreateSubnetworks": network.AutoCreateSubnetworks,
			"ipv4Range":             network.IPv4Range,
			"subnetworks":           network.Subnetworks,
			"routingConfig":         network.RoutingConfig,
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

type gcpSubnetwork struct {
	Id                    json.Number       `json:"id"`
	Name                  string            `json:"name"`
	SelfLink              string            `json:"selfLink"`
	Region                string            `json:"region"`
	Network               string            `json:"network"`
	IpCidrRange           string            `json:"ipCidrRange"`
	GatewayAddress        string            `json:"gatewayAddress"`
	PrivateIpGoogleAccess bool              `json:"privateIpGoogleAccess"`
	Labels                map[string]string `json:"labels"`
}

func collectGcpSubnetworks(ctx context.Context, account *cmdbCloudAccount, token, projectId string, selectedRegions map[string]bool) ([]*models.CmdbAsset, error) {
	resp := struct {
		Items map[string]struct {
			Subnetworks []gcpSubnetwork `json:"subnetworks"`
		} `json:"items"`
	}{}
	if err := gcpGet(ctx, token, fmt.Sprintf("%s/projects/%s/aggregated/subnetworks", gcpComputeEndpoint, url.PathEscape(projectId)), &resp); err != nil {
		return nil, err
	}
	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0)
	for _, scoped := range resp.Items {
		for _, subnet := range scoped.Subnetworks {
			region := gcpLastPathSegment(subnet.Region)
			if !gcpRegionSelected(selectedRegions, region) {
				continue
			}
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeNetworkSubnet, "gcp_compute_subnetwork", firstNonEmpty(subnet.SelfLink, subnet.Id.String(), subnet.Name), subnet.Name, now)
			asset.Tags = gcpLabelsToAttrs(subnet.Labels)
			asset.Attributes = models.ResAttrs{
				"selfLink":              subnet.SelfLink,
				"network":               subnet.Network,
				"cidrBlock":             subnet.IpCidrRange,
				"gatewayAddress":        subnet.GatewayAddress,
				"privateIpGoogleAccess": subnet.PrivateIpGoogleAccess,
			}
			assets = append(assets, asset)
		}
	}
	return assets, nil
}

type gcpFirewall struct {
	Id                json.Number       `json:"id"`
	Name              string            `json:"name"`
	SelfLink          string            `json:"selfLink"`
	Network           string            `json:"network"`
	Direction         string            `json:"direction"`
	Priority          int               `json:"priority"`
	SourceRanges      []string          `json:"sourceRanges"`
	DestinationRanges []string          `json:"destinationRanges"`
	TargetTags        []string          `json:"targetTags"`
	SourceTags        []string          `json:"sourceTags"`
	Allowed           []gcpFirewallRule `json:"allowed"`
	Denied            []gcpFirewallRule `json:"denied"`
	Disabled          bool              `json:"disabled"`
	Description       string            `json:"description"`
	Labels            map[string]string `json:"labels"`
}

type gcpFirewallRule struct {
	IPProtocol string   `json:"IPProtocol"`
	Ports      []string `json:"ports"`
}

func collectGcpFirewalls(ctx context.Context, account *cmdbCloudAccount, token, projectId string, selectedRegions map[string]bool) ([]*models.CmdbAsset, error) {
	resp := struct {
		Items []gcpFirewall `json:"items"`
	}{}
	if err := gcpGet(ctx, token, fmt.Sprintf("%s/projects/%s/global/firewalls", gcpComputeEndpoint, url.PathEscape(projectId)), &resp); err != nil {
		return nil, err
	}
	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0, len(resp.Items))
	for _, firewall := range resp.Items {
		asset := newCmdbCloudAsset(account, "global", models.CmdbAssetTypeNetworkSecurityGroup, "gcp_compute_firewall", firstNonEmpty(firewall.SelfLink, firewall.Id.String(), firewall.Name), firewall.Name, now)
		if firewall.Disabled {
			asset.Status = "disabled"
		} else {
			asset.Status = "enabled"
		}
		asset.Tags = gcpLabelsToAttrs(firewall.Labels)
		ingress, egress := gcpFirewallSecurityRules(firewall)
		asset.Attributes = models.ResAttrs{
			"selfLink":             firewall.SelfLink,
			"network":              firewall.Network,
			"direction":            firewall.Direction,
			"priority":             firewall.Priority,
			"description":          firewall.Description,
			"disabled":             firewall.Disabled,
			"sourceRanges":         firewall.SourceRanges,
			"destinationRanges":    firewall.DestinationRanges,
			"targetTags":           firewall.TargetTags,
			"sourceTags":           firewall.SourceTags,
			"ingressSecurityRules": ingress,
			"egressSecurityRules":  egress,
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

type gcpDisk struct {
	Id                json.Number       `json:"id"`
	Name              string            `json:"name"`
	SelfLink          string            `json:"selfLink"`
	Zone              string            `json:"zone"`
	Status            string            `json:"status"`
	SizeGb            string            `json:"sizeGb"`
	Type              string            `json:"type"`
	Users             []string          `json:"users"`
	Labels            map[string]string `json:"labels"`
	CreationTimestamp string            `json:"creationTimestamp"`
}

func collectGcpDisks(ctx context.Context, account *cmdbCloudAccount, token, projectId string, selectedRegions map[string]bool) ([]*models.CmdbAsset, error) {
	resp := struct {
		Items map[string]struct {
			Disks []gcpDisk `json:"disks"`
		} `json:"items"`
	}{}
	if err := gcpGet(ctx, token, fmt.Sprintf("%s/projects/%s/aggregated/disks", gcpComputeEndpoint, url.PathEscape(projectId)), &resp); err != nil {
		return nil, err
	}
	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0)
	for _, scoped := range resp.Items {
		for _, disk := range scoped.Disks {
			region := gcpRegionFromZone(disk.Zone)
			if !gcpRegionSelected(selectedRegions, region) {
				continue
			}
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeBlockVolume, "gcp_compute_disk", firstNonEmpty(disk.SelfLink, disk.Id.String(), disk.Name), disk.Name, now)
			asset.Zone = gcpLastPathSegment(disk.Zone)
			asset.Status = disk.Status
			asset.Tags = gcpLabelsToAttrs(disk.Labels)
			asset.Attributes = models.ResAttrs{
				"selfLink":          disk.SelfLink,
				"sizeGiB":           disk.SizeGb,
				"type":              gcpLastPathSegment(disk.Type),
				"users":             disk.Users,
				"creationTimestamp": disk.CreationTimestamp,
			}
			assets = append(assets, asset)
		}
	}
	return assets, nil
}

type gcpForwardingRule struct {
	Id                  json.Number       `json:"id"`
	Name                string            `json:"name"`
	SelfLink            string            `json:"selfLink"`
	Region              string            `json:"region"`
	IPAddress           string            `json:"IPAddress"`
	IPProtocol          string            `json:"IPProtocol"`
	LoadBalancingScheme string            `json:"loadBalancingScheme"`
	BackendService      string            `json:"backendService"`
	Target              string            `json:"target"`
	Network             string            `json:"network"`
	Subnetwork          string            `json:"subnetwork"`
	Labels              map[string]string `json:"labels"`
}

func collectGcpForwardingRules(ctx context.Context, account *cmdbCloudAccount, token, projectId string, selectedRegions map[string]bool) ([]*models.CmdbAsset, error) {
	resp := struct {
		Items map[string]struct {
			ForwardingRules []gcpForwardingRule `json:"forwardingRules"`
		} `json:"items"`
	}{}
	if err := gcpGet(ctx, token, fmt.Sprintf("%s/projects/%s/aggregated/forwardingRules", gcpComputeEndpoint, url.PathEscape(projectId)), &resp); err != nil {
		return nil, err
	}
	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0)
	for _, scoped := range resp.Items {
		for _, rule := range scoped.ForwardingRules {
			region := gcpLastPathSegment(rule.Region)
			if region == "" {
				region = "global"
			}
			if !gcpRegionSelected(selectedRegions, region) {
				continue
			}
			asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeLoadBalancer, "gcp_compute_forwarding_rule", firstNonEmpty(rule.SelfLink, rule.Id.String(), rule.Name), rule.Name, now)
			asset.Address = rule.IPAddress
			if strings.EqualFold(rule.LoadBalancingScheme, "external") || strings.EqualFold(rule.LoadBalancingScheme, "external_managed") {
				asset.PublicIp = rule.IPAddress
			} else {
				asset.PrivateIp = rule.IPAddress
			}
			asset.Tags = gcpLabelsToAttrs(rule.Labels)
			asset.Attributes = models.ResAttrs{
				"selfLink":            rule.SelfLink,
				"ipAddress":           rule.IPAddress,
				"ipProtocol":          rule.IPProtocol,
				"loadBalancingScheme": rule.LoadBalancingScheme,
				"backendService":      rule.BackendService,
				"target":              rule.Target,
				"network":             rule.Network,
				"subnetwork":          rule.Subnetwork,
			}
			assets = append(assets, asset)
		}
	}
	return assets, nil
}

type gcpCluster struct {
	Name                  string            `json:"name"`
	SelfLink              string            `json:"selfLink"`
	Location              string            `json:"location"`
	Zone                  string            `json:"zone"`
	Status                string            `json:"status"`
	Endpoint              string            `json:"endpoint"`
	CurrentMasterVersion  string            `json:"currentMasterVersion"`
	CurrentNodeVersion    string            `json:"currentNodeVersion"`
	InitialClusterVersion string            `json:"initialClusterVersion"`
	Network               string            `json:"network"`
	Subnetwork            string            `json:"subnetwork"`
	ClusterIpv4Cidr       string            `json:"clusterIpv4Cidr"`
	ServicesIpv4Cidr      string            `json:"servicesIpv4Cidr"`
	ResourceLabels        map[string]string `json:"resourceLabels"`
	NodePools             []gcpNodePool     `json:"nodePools"`
	NetworkConfig         models.ResAttrs   `json:"networkConfig"`
	IpAllocationPolicy    models.ResAttrs   `json:"ipAllocationPolicy"`
	PrivateClusterConfig  models.ResAttrs   `json:"privateClusterConfig"`
	ReleaseChannel        models.ResAttrs   `json:"releaseChannel"`
	Locations             []string          `json:"locations"`
}

type gcpNodePool struct {
	Name              string          `json:"name"`
	Status            string          `json:"status"`
	Version           string          `json:"version"`
	InitialNodeCount  int             `json:"initialNodeCount"`
	Locations         []string        `json:"locations"`
	InstanceGroupUrls []string        `json:"instanceGroupUrls"`
	Config            models.ResAttrs `json:"config"`
	Autoscaling       models.ResAttrs `json:"autoscaling"`
	NetworkConfig     models.ResAttrs `json:"networkConfig"`
	Management        models.ResAttrs `json:"management"`
	MaxPodsConstraint models.ResAttrs `json:"maxPodsConstraint"`
	UpgradeSettings   models.ResAttrs `json:"upgradeSettings"`
}

func collectGcpClusters(ctx context.Context, account *cmdbCloudAccount, token, projectId string, selectedRegions map[string]bool) ([]*models.CmdbAsset, error) {
	resp := struct {
		Clusters []gcpCluster `json:"clusters"`
	}{}
	if err := gcpGet(ctx, token, fmt.Sprintf("%s/projects/%s/locations/-/clusters", gcpContainerEndpoint, url.PathEscape(projectId)), &resp); err != nil {
		return nil, err
	}
	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0, len(resp.Clusters))
	for _, cluster := range resp.Clusters {
		region := gcpRegionFromLocation(firstNonEmpty(cluster.Location, cluster.Zone))
		if !gcpRegionSelected(selectedRegions, region) {
			continue
		}
		asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeKubernetesCluster, "gcp_container_cluster", firstNonEmpty(cluster.SelfLink, cluster.Name), cluster.Name, now)
		asset.Zone = cluster.Zone
		asset.Status = cluster.Status
		asset.Address = cluster.Endpoint
		asset.Tags = gcpLabelsToAttrs(cluster.ResourceLabels)
		version := firstNonEmpty(cluster.CurrentMasterVersion, cluster.CurrentNodeVersion, cluster.InitialClusterVersion)
		nodePools := gcpClusterNodePools(cluster)
		subnetIds := dedupeStrings(nonEmptyStrings([]string{cluster.Subnetwork}))
		asset.Attributes = models.ResAttrs{
			"selfLink":              cluster.SelfLink,
			"location":              cluster.Location,
			"locations":             cluster.Locations,
			"endpoint":              cluster.Endpoint,
			"apiEndpoint":           cluster.Endpoint,
			"currentMasterVersion":  cluster.CurrentMasterVersion,
			"currentNodeVersion":    cluster.CurrentNodeVersion,
			"initialClusterVersion": cluster.InitialClusterVersion,
			"kubernetesVersion":     version,
			"version":               version,
			"network":               cluster.Network,
			"networkId":             cluster.Network,
			"vpcId":                 cluster.Network,
			"subnetwork":            cluster.Subnetwork,
			"subnetworkIds":         subnetIds,
			"subnetIds":             subnetIds,
			"nodePools":             nodePools,
			"nodePoolCount":         len(nodePools),
			"endpointConfig": models.ResAttrs{
				"publicEndpoint":  cluster.Endpoint,
				"privateEndpoint": attrString(cluster.PrivateClusterConfig, "privateEndpoint"),
			},
			"kubernetesNetwork": models.ResAttrs{
				"networkConfig":        cluster.NetworkConfig,
				"ipAllocationPolicy":   cluster.IpAllocationPolicy,
				"privateClusterConfig": cluster.PrivateClusterConfig,
				"releaseChannel":       cluster.ReleaseChannel,
				"clusterIpv4Cidr":      cluster.ClusterIpv4Cidr,
				"servicesIpv4Cidr":     cluster.ServicesIpv4Cidr,
			},
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

func gcpClusterNodePools(cluster gcpCluster) []models.ResAttrs {
	pools := make([]models.ResAttrs, 0, len(cluster.NodePools))
	for _, pool := range cluster.NodePools {
		scaling := models.ResAttrs{}
		if pool.InitialNodeCount > 0 {
			scaling["desiredSize"] = pool.InitialNodeCount
		}
		if pool.Autoscaling != nil {
			if attrString(pool.Autoscaling, "enabled") != "" {
				scaling["autoScaling"] = attrString(pool.Autoscaling, "enabled")
			}
			if attrString(pool.Autoscaling, "minNodeCount") != "" {
				scaling["minSize"] = attrInt(pool.Autoscaling, "minNodeCount")
			}
			if attrString(pool.Autoscaling, "maxNodeCount") != "" {
				scaling["maxSize"] = attrInt(pool.Autoscaling, "maxNodeCount")
			}
			if attrString(pool.Autoscaling, "totalMinNodeCount") != "" {
				scaling["totalMinSize"] = attrInt(pool.Autoscaling, "totalMinNodeCount")
			}
			if attrString(pool.Autoscaling, "totalMaxNodeCount") != "" {
				scaling["totalMaxSize"] = attrInt(pool.Autoscaling, "totalMaxNodeCount")
			}
		}
		poolSubnetIds := dedupeStrings(nonEmptyStrings([]string{cluster.Subnetwork}))
		pools = append(pools, models.ResAttrs{
			"id":                pool.Name,
			"name":              pool.Name,
			"status":            pool.Status,
			"version":           pool.Version,
			"kubernetesVersion": pool.Version,
			"instanceTypes":     nonEmptyStrings([]string{attrString(pool.Config, "machineType")}),
			"nodeCount":         pool.InitialNodeCount,
			"subnetIds":         poolSubnetIds,
			"locations":         pool.Locations,
			"zones":             pool.Locations,
			"instanceGroupUrls": pool.InstanceGroupUrls,
			"scalingConfig":     scaling,
			"nodeConfigDetails": pool.Config,
			"autoscaling":       pool.Autoscaling,
			"networkConfig":     pool.NetworkConfig,
			"management":        pool.Management,
			"maxPodsConstraint": pool.MaxPodsConstraint,
			"upgradeSettings":   pool.UpgradeSettings,
		})
	}
	return pools
}

type gcpSqlInstance struct {
	Name            string            `json:"name"`
	SelfLink        string            `json:"selfLink"`
	Region          string            `json:"region"`
	State           string            `json:"state"`
	DatabaseVersion string            `json:"databaseVersion"`
	InstanceType    string            `json:"instanceType"`
	Settings        models.ResAttrs   `json:"settings"`
	Labels          map[string]string `json:"labels"`
}

func collectGcpSqlInstances(ctx context.Context, account *cmdbCloudAccount, token, projectId string, selectedRegions map[string]bool) ([]*models.CmdbAsset, error) {
	resp := struct {
		Items []gcpSqlInstance `json:"items"`
	}{}
	if err := gcpGet(ctx, token, fmt.Sprintf("%s/projects/%s/instances", gcpSqlAdminEndpoint, url.PathEscape(projectId)), &resp); err != nil {
		return nil, err
	}
	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0, len(resp.Items))
	for _, instance := range resp.Items {
		if !gcpRegionSelected(selectedRegions, instance.Region) {
			continue
		}
		asset := newCmdbCloudAsset(account, instance.Region, models.CmdbAssetTypeRelationalDatabase, "gcp_sql_instance", firstNonEmpty(instance.SelfLink, instance.Name), instance.Name, now)
		asset.Status = instance.State
		asset.Tags = gcpLabelsToAttrs(instance.Labels)
		asset.Attributes = models.ResAttrs{
			"selfLink":        instance.SelfLink,
			"databaseVersion": instance.DatabaseVersion,
			"instanceType":    instance.InstanceType,
			"settings":        instance.Settings,
			"network":         gcpSqlPrivateNetwork(instance.Settings),
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

type gcpBucket struct {
	Id           string            `json:"id"`
	Name         string            `json:"name"`
	Location     string            `json:"location"`
	StorageClass string            `json:"storageClass"`
	TimeCreated  string            `json:"timeCreated"`
	Updated      string            `json:"updated"`
	Labels       map[string]string `json:"labels"`
}

func collectGcpBuckets(ctx context.Context, account *cmdbCloudAccount, token, projectId string, selectedRegions map[string]bool) ([]*models.CmdbAsset, error) {
	resp := struct {
		Items []gcpBucket `json:"items"`
	}{}
	values := url.Values{}
	values.Set("project", projectId)
	if err := gcpGet(ctx, token, gcpStorageEndpoint+"/b?"+values.Encode(), &resp); err != nil {
		return nil, err
	}
	now := cmdbNow()
	assets := make([]*models.CmdbAsset, 0, len(resp.Items))
	for _, bucket := range resp.Items {
		region := strings.ToLower(bucket.Location)
		if !gcpRegionSelected(selectedRegions, region) {
			continue
		}
		asset := newCmdbCloudAsset(account, region, models.CmdbAssetTypeObjectStorageBucket, "gcp_storage_bucket", firstNonEmpty(bucket.Id, bucket.Name), bucket.Name, now)
		asset.Tags = gcpLabelsToAttrs(bucket.Labels)
		asset.Attributes = models.ResAttrs{
			"storageClass": bucket.StorageClass,
			"timeCreated":  bucket.TimeCreated,
			"updated":      bucket.Updated,
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

func gcpGet(ctx context.Context, token, rawURL string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	return gcpDoJSON(req, token, out)
}

func gcpInstanceIPs(instance gcpInstance) (string, string) {
	privateIp := ""
	publicIp := ""
	for _, iface := range instance.NetworkInterfaces {
		privateIp = firstNonEmpty(privateIp, iface.NetworkIP)
		for _, accessConfig := range iface.AccessConfigs {
			publicIp = firstNonEmpty(publicIp, accessConfig.NatIP)
		}
	}
	return privateIp, publicIp
}

func gcpInstanceNetworkRefs(instance gcpInstance) []string {
	refs := make([]string, 0)
	for _, iface := range instance.NetworkInterfaces {
		refs = append(refs, iface.Network)
	}
	return dedupeStrings(nonEmptyStrings(refs))
}

func gcpInstanceSubnetworkRefs(instance gcpInstance) []string {
	refs := make([]string, 0)
	for _, iface := range instance.NetworkInterfaces {
		refs = append(refs, iface.Subnetwork)
	}
	return dedupeStrings(nonEmptyStrings(refs))
}

func gcpInstancePublicIPs(instance gcpInstance) []string {
	refs := make([]string, 0)
	for _, iface := range instance.NetworkInterfaces {
		for _, accessConfig := range iface.AccessConfigs {
			refs = append(refs, accessConfig.NatIP)
		}
	}
	return dedupeStrings(nonEmptyStrings(refs))
}

func gcpInstanceDiskSources(instance gcpInstance) []string {
	refs := make([]string, 0)
	for _, disk := range instance.Disks {
		refs = append(refs, attrString(disk, "source"))
	}
	return dedupeStrings(nonEmptyStrings(refs))
}

func gcpSqlPrivateNetwork(settings models.ResAttrs) string {
	if settings == nil {
		return ""
	}
	ipConfig := modelResAttrs(settings["ipConfiguration"])
	if ipConfig == nil {
		return ""
	}
	return attrString(ipConfig, "privateNetwork")
}

func gcpFirewallSecurityRules(firewall gcpFirewall) ([]models.ResAttrs, []models.ResAttrs) {
	rules := make([]models.ResAttrs, 0)
	add := func(effect string, entries []gcpFirewallRule) {
		for _, entry := range entries {
			rules = append(rules, models.ResAttrs{
				"name":        firewall.Name,
				"protocol":    entry.IPProtocol,
				"source":      strings.Join(firewall.SourceRanges, ","),
				"destination": strings.Join(firewall.DestinationRanges, ","),
				"portRange":   strings.Join(entry.Ports, ","),
				"effect":      effect,
				"priority":    firewall.Priority,
				"description": firewall.Description,
			})
		}
	}
	add("allow", firewall.Allowed)
	add("deny", firewall.Denied)
	if strings.EqualFold(firewall.Direction, "egress") {
		return nil, rules
	}
	return rules, nil
}

func gcpLabelsToAttrs(labels map[string]string) models.ResAttrs {
	if labels == nil {
		return models.ResAttrs{}
	}
	result := models.ResAttrs{}
	for key, value := range labels {
		result[key] = value
	}
	return result
}

func gcpProjectId(account *cmdbCloudAccount) string {
	if account == nil {
		return ""
	}
	return firstNonEmpty(account.AccountId, account.Credentials["GCP_PROJECT_ID"], account.Credentials["GOOGLE_CLOUD_PROJECT"])
}

func selectedGcpRegions(regions []string) map[string]bool {
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

func gcpRegionSelected(selected map[string]bool, region string) bool {
	if len(selected) == 0 || region == "" || region == "global" {
		return true
	}
	return selected[strings.ToLower(region)]
}

func gcpRegionFromZone(zoneURL string) string {
	zone := gcpLastPathSegment(zoneURL)
	if zone == "" {
		return ""
	}
	return gcpRegionFromLocation(zone)
}

func gcpRegionFromLocation(location string) string {
	location = strings.ToLower(strings.TrimSpace(location))
	parts := strings.Split(location, "-")
	if len(parts) >= 3 {
		return strings.Join(parts[:len(parts)-1], "-")
	}
	return location
}

func gcpLastPathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(strings.TrimRight(value, "/"), "/")
	return parts[len(parts)-1]
}
