package apps

import (
	"testing"

	"cloudiac/portal/models"
)

func TestOciRouteRuleRefsExtractsGatewayRefs(t *testing.T) {
	refs := ociRouteRuleRefs([]interface{}{
		map[string]interface{}{
			"networkEntityId": "ocid1.natgateway.oc1..nat1",
			"destination":     "0.0.0.0/0",
		},
		map[string]interface{}{
			"networkEntityId":  "ocid1.internetgateway.oc1..igw1",
			"destination":      "::/0",
			"destinationType":  "CIDR_BLOCK",
			"routeDescription": "ipv6 default",
		},
		map[string]interface{}{
			"networkEntityId": "ocid1.servicegateway.oc1..sgw1",
			"destination":     "all-sin-services-in-oracle-services-network",
			"destinationType": "SERVICE_CIDR_BLOCK",
		},
		map[string]interface{}{
			"networkEntityId": "ocid1.drg.oc1..drg1",
			"destination":     "10.0.0.0/8",
		},
	})

	assertStringSliceEqual(t, refs["natGatewayIds"], []string{"ocid1.natgateway.oc1..nat1"})
	assertStringSliceEqual(t, refs["internetGatewayIds"], []string{"ocid1.internetgateway.oc1..igw1"})
	assertStringSliceEqual(t, refs["serviceGatewayIds"], []string{"ocid1.servicegateway.oc1..sgw1"})
	assertStringSliceEqual(t, refs["drgIds"], []string{"ocid1.drg.oc1..drg1"})
	assertStringSliceEqual(t, refs["destinationCidrBlocks"], []string{"0.0.0.0/0", "::/0", "10.0.0.0/8"})
	assertStringSliceEqual(t, refs["destinationServiceIds"], []string{"all-sin-services-in-oracle-services-network"})
}

func TestInferCloudRelationsAddsOciNetworkGatewayRefs(t *testing.T) {
	vcn := ociTestAsset("vcn-1", models.CmdbAssetTypeNetworkVpc, "ocid1.vcn.oc1..vcn1", nil)
	subnet := ociTestAsset("subnet-1", models.CmdbAssetTypeNetworkSubnet, "ocid1.subnet.oc1..subnet1", models.ResAttrs{
		"vcnId":           "ocid1.vcn.oc1..vcn1",
		"securityListIds": []string{"ocid1.securitylist.oc1..sl1"},
	})
	securityList := ociTestAsset("sl-1", models.CmdbAssetTypeNetworkSecurityGroup, "ocid1.securitylist.oc1..sl1", models.ResAttrs{
		"vcnId": "ocid1.vcn.oc1..vcn1",
	})
	routeTable := ociTestAsset("rt-1", models.CmdbAssetTypeNetworkRouteTable, "ocid1.routetable.oc1..rt1", models.ResAttrs{
		"vcnId":              "ocid1.vcn.oc1..vcn1",
		"subnetIds":          []string{"ocid1.subnet.oc1..subnet1"},
		"natGatewayIds":      []string{"ocid1.natgateway.oc1..nat1"},
		"internetGatewayIds": []string{"ocid1.internetgateway.oc1..igw1"},
		"serviceGatewayIds":  []string{"ocid1.servicegateway.oc1..sgw1"},
		"drgIds":             []string{"ocid1.drg.oc1..drg1"},
	})
	nat := ociTestAsset("nat-1", models.CmdbAssetTypeNetworkNatGateway, "ocid1.natgateway.oc1..nat1", models.ResAttrs{
		"vcnId": "ocid1.vcn.oc1..vcn1",
	})
	igw := ociTestAsset("igw-1", models.CmdbAssetTypeNetworkInternetGateway, "ocid1.internetgateway.oc1..igw1", models.ResAttrs{
		"vcnId": "ocid1.vcn.oc1..vcn1",
	})
	serviceGateway := ociTestAsset("sgw-1", models.CmdbAssetTypeNetworkServiceGateway, "ocid1.servicegateway.oc1..sgw1", models.ResAttrs{
		"vcnId": "ocid1.vcn.oc1..vcn1",
	})
	drg := ociTestAsset("drg-1", models.CmdbAssetTypeNetworkDrg, "ocid1.drg.oc1..drg1", nil)

	assets := []models.CmdbAsset{vcn, subnet, securityList, routeTable, nat, igw, serviceGateway, drg}
	relations := inferCmdbCloudRelations("org-test", "oci", "tenancy-1", assets, newCmdbCloudRelationIndex(assets))

	for _, expected := range []struct {
		sourceId   string
		targetId   string
		inferredBy string
	}{
		{"vcn-1", "subnet-1", "subnet_parent_network"},
		{"subnet-1", "sl-1", "subnet_security"},
		{"vcn-1", "rt-1", "route_table_parent_network"},
		{"rt-1", "subnet-1", "route_table_subnets"},
		{"rt-1", "nat-1", "route_table_nat_gateway"},
		{"rt-1", "igw-1", "route_table_internet_gateway"},
		{"rt-1", "sgw-1", "route_table_service_gateway"},
		{"rt-1", "drg-1", "route_table_drg"},
		{"vcn-1", "nat-1", "nat_gateway_parent_network"},
		{"vcn-1", "igw-1", "internet_gateway_parent_network"},
		{"vcn-1", "sgw-1", "service_gateway_parent_network"},
	} {
		if !hasOciInferredRelation(relations, expected.sourceId, expected.targetId, expected.inferredBy) {
			t.Fatalf("expected relation %s -> %s by %s, got %#v", expected.sourceId, expected.targetId, expected.inferredBy, relations)
		}
	}
}

func TestBuildOciCompartmentRefsAddsPaths(t *testing.T) {
	account := &cmdbCloudAccount{Credentials: map[string]string{
		"OCI_TENANCY_OCID": "ocid1.tenancy.oc1..root",
		"OCI_TENANCY_NAME": "root-tenancy",
	}}
	refs := buildOciCompartmentRefs(account, []string{"ocid1.tenancy.oc1..root"}, []models.ResAttrs{
		{
			"id":             "ocid1.compartment.oc1..prod",
			"name":           "prod",
			"compartmentId":  "ocid1.tenancy.oc1..root",
			"lifecycleState": "ACTIVE",
		},
		{
			"id":             "ocid1.compartment.oc1..app",
			"name":           "app",
			"compartmentId":  "ocid1.compartment.oc1..prod",
			"lifecycleState": "ACTIVE",
		},
	})
	byId := make(map[string]ociCompartmentRef, len(refs))
	for _, ref := range refs {
		byId[ref.Id] = ref
	}

	if got := byId["ocid1.tenancy.oc1..root"].Path; got != "/root-tenancy" {
		t.Fatalf("unexpected tenancy path: %q", got)
	}
	if got := byId["ocid1.compartment.oc1..prod"].Path; got != "/root-tenancy/prod" {
		t.Fatalf("unexpected prod path: %q", got)
	}
	if got := byId["ocid1.compartment.oc1..app"].Path; got != "/root-tenancy/prod/app" {
		t.Fatalf("unexpected app path: %q", got)
	}
}

func TestAnnotateOciCompartmentAssetAddsIdentityAttrs(t *testing.T) {
	account := &cmdbCloudAccount{Credentials: map[string]string{
		"OCI_TENANCY_OCID":   "ocid1.tenancy.oc1..root",
		"OCI_TENANCY_NAME":   "root-tenancy",
		"OCI_USER_OCID":      "ocid1.user.oc1..user1",
		"OCI_FINGERPRINT":    "aa:bb:cc",
		"OCI_COMPARTMENT_ID": "ignored",
	}}
	asset := &models.CmdbAsset{
		Attributes: models.ResAttrs{},
		RawData:    models.ResAttrs{},
	}
	annotateOciCompartmentAsset(account, asset, ociCompartmentRef{
		Id:             "ocid1.compartment.oc1..app",
		Name:           "app",
		ParentId:       "ocid1.compartment.oc1..prod",
		Path:           "/root-tenancy/prod/app",
		LifecycleState: "ACTIVE",
	})

	for key, expected := range map[string]string{
		"tenancyId":                 "ocid1.tenancy.oc1..root",
		"tenancyName":               "root-tenancy",
		"userId":                    "ocid1.user.oc1..user1",
		"credentialFingerprint":     "aa:bb:cc",
		"compartmentId":             "ocid1.compartment.oc1..app",
		"compartmentName":           "app",
		"compartmentPath":           "/root-tenancy/prod/app",
		"compartmentParentId":       "ocid1.compartment.oc1..prod",
		"compartmentLifecycleState": "ACTIVE",
	} {
		if got := attrString(asset.Attributes, key); got != expected {
			t.Fatalf("expected %s=%q, got %q", key, expected, got)
		}
	}
	if got := attrString(asset.RawData, "compartmentPath"); got != "/root-tenancy/prod/app" {
		t.Fatalf("expected raw compartment path, got %q", got)
	}
}

func ociTestAsset(id string, assetType string, nativeId string, attrs models.ResAttrs) models.CmdbAsset {
	if attrs == nil {
		attrs = models.ResAttrs{}
	}
	return models.CmdbAsset{
		TimedModel: models.TimedModel{BaseModel: models.BaseModel{Id: models.Id(id)}},
		AssetType:  assetType,
		NativeId:   nativeId,
		Attributes: attrs,
	}
}

func hasOciInferredRelation(relations []models.CmdbAssetRelation, sourceId, targetId, inferredBy string) bool {
	for _, relation := range relations {
		if relation.SourceAssetId == models.Id(sourceId) &&
			relation.TargetAssetId == models.Id(targetId) &&
			attrString(relation.Metadata, "inferredBy") == inferredBy {
			return true
		}
	}
	return false
}

func assertStringSliceEqual(t *testing.T, value interface{}, expected []string) {
	t.Helper()
	actual, ok := value.([]string)
	if !ok {
		t.Fatalf("expected []string %v, got %#v", expected, value)
	}
	if len(actual) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
	for idx := range expected {
		if actual[idx] != expected[idx] {
			t.Fatalf("expected %v, got %v", expected, actual)
		}
	}
}
