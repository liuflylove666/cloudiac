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
	ociCoreAPIVersion            = "20160918"
	ociLoadBalancerAPIVersion    = "20170115"
	ociContainerEngineAPIVersion = "20180222"
	ociRedisAPIVersion           = "20220315"
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
	for _, region := range regions {
		region = strings.TrimSpace(region)
		if region == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		regionAssets, regionCompartments, regionErrors := collectCmdbOciRegionAssets(ctx, account, region, selected)
		cancel()
		assets = append(assets, regionAssets...)
		compartmentStats[region] = regionCompartments
		errors = append(errors, regionErrors...)
	}

	stats["collected"] = len(assets)
	stats["compartments"] = compartmentStats
	if len(errors) > 0 {
		stats["errors"] = errors
	}
	return cmdbCloudCollectResult{
		Assets: assets,
		Stats:  stats,
		Err:    cmdbCollectError(errors),
	}
}

func collectCmdbOciRegionAssets(ctx context.Context, account *cmdbCloudAccount, region string, selected map[string]bool) ([]*models.CmdbAsset, []string, []string) {
	assets := make([]*models.CmdbAsset, 0)
	errors := make([]string, 0)

	compartmentIds, err := ociCompartmentIDs(ctx, account, region)
	if err != nil {
		errors = append(errors, fmt.Sprintf("oci %s compartments: %v", region, err))
	}
	for _, compartmentId := range compartmentIds {
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeComputeInstance) {
			items, err := collectOciInstances(ctx, account, region, compartmentId)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s compute instances in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkVpc) {
			items, err := collectOciVcns(ctx, account, region, compartmentId)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s vcns in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkSubnet) {
			items, err := collectOciSubnets(ctx, account, region, compartmentId)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s subnets in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeNetworkSecurityGroup) {
			items, err := collectOciSecurityAssets(ctx, account, region, compartmentId)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s security assets in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypePublicIP) {
			items, err := collectOciPublicIps(ctx, account, region, compartmentId)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s public ips in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeBlockVolume) {
			items, err := collectOciVolumes(ctx, account, region, compartmentId)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s block volumes in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeLoadBalancer) {
			items, err := collectOciLoadBalancers(ctx, account, region, compartmentId)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s load balancers in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeObjectStorageBucket) {
			items, err := collectOciBuckets(ctx, account, region, compartmentId)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s buckets in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeKubernetesCluster) {
			items, err := collectOciOkeClusters(ctx, account, region, compartmentId)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s oke clusters in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeRelationalDatabase) {
			items, err := collectOciDatabases(ctx, account, region, compartmentId)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s databases in %s: %v", region, compartmentId, err))
			}
		}
		if wantsCmdbAssetType(selected, models.CmdbAssetTypeRedisCache) {
			items, err := collectOciRedisCaches(ctx, account, region, compartmentId)
			assets = append(assets, items...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("oci %s redis caches in %s: %v", region, compartmentId, err))
			}
		}
	}
	return assets, compartmentIds, errors
}

func ociCompartmentIDs(ctx context.Context, account *cmdbCloudAccount, region string) ([]string, error) {
	configured := normalizeStringList([]string{
		account.Credentials["OCI_COMPARTMENT_OCIDS"],
		account.Credentials["OCI_COMPARTMENT_OCID"],
	})
	if len(configured) == 0 {
		configured = []string{strings.TrimSpace(account.Credentials["OCI_TENANCY_OCID"])}
	}
	configured = dedupeStrings(configured)
	if !ociCredentialBool(account, "OCI_INCLUDE_SUBCOMPARTMENTS", "OCI_COMPARTMENT_IN_SUBTREE") {
		return configured, nil
	}

	query := url.Values{
		"compartmentId":          []string{strings.TrimSpace(account.Credentials["OCI_TENANCY_OCID"])},
		"compartmentIdInSubtree": []string{"true"},
		"accessLevel":            []string{"ACCESSIBLE"},
		"lifecycleState":         []string{"ACTIVE"},
	}
	items, err := ociListAPI(ctx, account, region, "identity", ociCoreAPIVersion, "/compartments", query)
	if err != nil {
		return configured, err
	}
	compartmentIds := append([]string{}, configured...)
	for _, item := range items {
		if id := ociAttrString(item, "id"); id != "" {
			compartmentIds = append(compartmentIds, id)
		}
	}
	return dedupeStrings(compartmentIds), nil
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
	return collectOciSimpleAssets(ctx, account, region, compartmentId, models.CmdbAssetTypeKubernetesCluster, "oci_containerengine_cluster",
		"containerengine", ociContainerEngineAPIVersion, "/clusters", nil, func(asset *models.CmdbAsset, item models.ResAttrs) {
			asset.Attributes["kubernetesVersion"] = ociAttrString(item, "kubernetesVersion")
			asset.Attributes["vcnId"] = ociAttrString(item, "vcnId")
			asset.Attributes["endpointConfig"] = item["endpointConfig"]
			asset.Attributes["options"] = item["options"]
		})
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
			"subnetId":  ociAttrString(vnic, "subnetId"),
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

func ociDoAPI(ctx context.Context, account *cmdbCloudAccount, region, service, version, path string, query url.Values) ([]byte, http.Header, error) {
	basePath := path
	if version != "" {
		basePath = "/" + strings.Trim(version, "/") + "/" + strings.TrimLeft(path, "/")
	}
	u := url.URL{
		Scheme:   "https",
		Host:     ociServiceHost(service, region),
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
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.Header, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.Header, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return body, resp.Header, nil
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
	signingString := strings.Join([]string{
		"date: " + date,
		"(request-target): " + requestTarget,
		"host: " + req.URL.Host,
	}, "\n")

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
	req.Header.Set("Authorization", fmt.Sprintf(`Signature version="1",keyId="%s",algorithm="rsa-sha256",headers="date (request-target) host",signature="%s"`,
		keyId, base64.StdEncoding.EncodeToString(signature)))
	return nil
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
