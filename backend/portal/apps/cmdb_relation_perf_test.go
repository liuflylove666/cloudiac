package apps

import (
	"testing"

	"cloudiac/portal/models"
	"cloudiac/portal/models/resps"
)

func TestNormalizeCmdbAssetRelationLimit(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{name: "default for zero", input: 0, want: cmdbAssetRelationDefaultLimit},
		{name: "default for negative", input: -1, want: cmdbAssetRelationDefaultLimit},
		{name: "keeps valid limit", input: 50, want: 50},
		{name: "caps large limit", input: cmdbAssetRelationMaxLimit + 1, want: cmdbAssetRelationMaxLimit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeCmdbAssetRelationLimit(tt.input); got != tt.want {
				t.Fatalf("normalizeCmdbAssetRelationLimit(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeCmdbAssetRelationOffset(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{name: "keeps zero", input: 0, want: 0},
		{name: "floors negative", input: -1, want: 0},
		{name: "keeps valid offset", input: 250, want: 250},
		{name: "caps large offset", input: 100001, want: 100000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeCmdbAssetRelationOffset(tt.input); got != tt.want {
				t.Fatalf("normalizeCmdbAssetRelationOffset(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeCmdbAssetRelationCursor(t *testing.T) {
	tests := []struct {
		name     string
		cursor   string
		fallback int
		want     int
	}{
		{name: "empty uses fallback", cursor: "", fallback: 20, want: 20},
		{name: "valid cursor wins", cursor: "40", fallback: 20, want: 40},
		{name: "invalid cursor uses fallback", cursor: "bad", fallback: 20, want: 20},
		{name: "negative cursor floors", cursor: "-1", fallback: 20, want: 0},
		{name: "large cursor caps", cursor: "100001", fallback: 20, want: 100000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeCmdbAssetRelationCursor(tt.cursor, tt.fallback); got != tt.want {
				t.Fatalf("normalizeCmdbAssetRelationCursor(%q, %d) = %d, want %d", tt.cursor, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestCmdbRelationBreakdownListSortsByCountThenName(t *testing.T) {
	items := map[string]int64{
		"cloud_inferred":       3,
		"iac_dependency":       5,
		"application_inferred": 5,
	}
	got := cmdbRelationBreakdownList(items)
	want := []resps.CmdbAssetRelationMetricResp{
		{Name: "application_inferred", Count: 5},
		{Name: "iac_dependency", Count: 5},
		{Name: "cloud_inferred", Count: 3},
	}
	if len(got) != len(want) {
		t.Fatalf("breakdown length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("breakdown[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestCmdbAssetRelationFilterMatchesRelation(t *testing.T) {
	assetId := models.Id("asset-main")
	relation := resps.CmdbAssetRelationResp{
		CmdbAssetRelation: models.CmdbAssetRelation{
			SourceAssetId: assetId,
			TargetAssetId: models.Id("asset-db"),
			RelationType:  models.CmdbRelationTypeDependsOn,
			Source:        models.CmdbRelationSourceIacDependency,
			Metadata: models.ResAttrs{
				"port": "443",
			},
		},
		SourceAssetName: "web",
		SourceAssetType: "compute_instance",
		TargetAssetName: "database",
		TargetAssetType: "relational_database",
	}

	filter := newCmdbAssetRelationFilter(
		[]string{models.CmdbRelationSourceIacDependency},
		[]string{models.CmdbRelationTypeDependsOn},
		"outgoing",
		"web",
	)
	if !filter.matchesRelation(relation, assetId) {
		t.Fatal("expected outgoing dependency relation to match")
	}

	filter = newCmdbAssetRelationFilter(nil, nil, "incoming", "")
	if filter.matchesRelation(relation, assetId) {
		t.Fatal("expected incoming filter to reject outgoing relation")
	}

	filter = newCmdbAssetRelationFilter(nil, nil, "all", "443")
	if !filter.matchesRelation(relation, assetId) {
		t.Fatal("expected keyword filter to match relation metadata")
	}
}

func TestCmdbAssetRelationFilterMatchesApplicationAggregate(t *testing.T) {
	filter := newCmdbAssetRelationFilter(
		[]string{models.CmdbRelationSourceAppInferred},
		[]string{models.CmdbRelationTypeDependsOn},
		"incoming",
		"",
	)

	if !filter.matchesApplicationAggregate(resps.CmdbApplicationRelationResp{
		Direction:    "incoming",
		RelationType: models.CmdbRelationTypeDependsOn,
	}) {
		t.Fatal("expected incoming application aggregate to match")
	}
	if filter.matchesApplicationAggregate(resps.CmdbApplicationRelationResp{
		Direction:    "outgoing",
		RelationType: models.CmdbRelationTypeDependsOn,
	}) {
		t.Fatal("expected outgoing application aggregate to be rejected")
	}
}

func TestSanitizeCmdbAssetAttrsMasksSensitiveNestedFields(t *testing.T) {
	attrs := models.ResAttrs{
		"name":           "demo",
		"privateKey":     "private-value",
		"metadata":       models.ResAttrs{"ACCESS_KEY_ID": "ak-value", "region": "us-east-1"},
		"securityGroups": []interface{}{models.ResAttrs{"session-token": "token-value", "cidr": "0.0.0.0/0"}},
	}

	got := sanitizeCmdbAssetAttrs(attrs)
	if got["name"] != "demo" {
		t.Fatalf("expected non-sensitive name to be preserved, got %#v", got["name"])
	}
	if got["privateKey"] != cmdbAssetMaskedValue {
		t.Fatalf("expected privateKey to be masked, got %#v", got["privateKey"])
	}
	metadata, ok := got["metadata"].(models.ResAttrs)
	if !ok {
		t.Fatalf("expected metadata to stay as ResAttrs, got %#v", got["metadata"])
	}
	if metadata["ACCESS_KEY_ID"] != cmdbAssetMaskedValue {
		t.Fatalf("expected nested access key to be masked, got %#v", metadata["ACCESS_KEY_ID"])
	}
	if metadata["region"] != "us-east-1" {
		t.Fatalf("expected nested region to be preserved, got %#v", metadata["region"])
	}
	groups, ok := got["securityGroups"].([]interface{})
	if !ok || len(groups) != 1 {
		t.Fatalf("expected securityGroups list to be preserved, got %#v", got["securityGroups"])
	}
	group, ok := groups[0].(models.ResAttrs)
	if !ok {
		t.Fatalf("expected security group item to stay as ResAttrs, got %#v", groups[0])
	}
	if group["session-token"] != cmdbAssetMaskedValue {
		t.Fatalf("expected session token to be masked, got %#v", group["session-token"])
	}
	if group["cidr"] != "0.0.0.0/0" {
		t.Fatalf("expected cidr to be preserved, got %#v", group["cidr"])
	}
}

func TestSanitizeCmdbAssetRespMasksAssetAndRelationMetadata(t *testing.T) {
	asset := resps.CmdbAssetResp{
		CmdbAsset: models.CmdbAsset{
			Tags:       models.ResAttrs{"owner": "sre", "client-secret": "secret-value"},
			Attributes: models.ResAttrs{"password": "password-value"},
			RawData:    models.ResAttrs{"Authorization": "Bearer token"},
		},
		Relations: []resps.CmdbAssetRelationResp{
			{CmdbAssetRelation: models.CmdbAssetRelation{Metadata: models.ResAttrs{"signature": "sig-value", "sourceAddress": "web"}}},
		},
	}

	sanitizeCmdbAssetResp(&asset)
	if asset.Tags["owner"] != "sre" {
		t.Fatalf("expected owner tag to be preserved, got %#v", asset.Tags["owner"])
	}
	if asset.Tags["client-secret"] != cmdbAssetMaskedValue {
		t.Fatalf("expected client secret tag to be masked, got %#v", asset.Tags["client-secret"])
	}
	if asset.Attributes["password"] != cmdbAssetMaskedValue {
		t.Fatalf("expected password attribute to be masked, got %#v", asset.Attributes["password"])
	}
	if asset.RawData["Authorization"] != cmdbAssetMaskedValue {
		t.Fatalf("expected authorization raw data to be masked, got %#v", asset.RawData["Authorization"])
	}
	if asset.Relations[0].Metadata["signature"] != cmdbAssetMaskedValue {
		t.Fatalf("expected relation signature metadata to be masked, got %#v", asset.Relations[0].Metadata["signature"])
	}
	if asset.Relations[0].Metadata["sourceAddress"] != "web" {
		t.Fatalf("expected relation sourceAddress metadata to be preserved, got %#v", asset.Relations[0].Metadata["sourceAddress"])
	}
}
