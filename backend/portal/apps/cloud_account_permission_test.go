package apps

import (
	"testing"
	"time"

	"cloudiac/portal/models"
	"cloudiac/portal/models/resps"
)

func TestCloudAccountPermissionSnapshotFresh(t *testing.T) {
	validatedAt := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	account := &models.CloudAccount{LastValidatedAt: models.Time(validatedAt)}

	if !cloudAccountPermissionSnapshotFresh(models.CloudAccountPermission{
		CheckedAt: models.Time(validatedAt.Add(500 * time.Millisecond)),
	}, account) {
		t.Fatal("expected snapshot checked after validation to be fresh")
	}

	if cloudAccountPermissionSnapshotFresh(models.CloudAccountPermission{
		CheckedAt: models.Time(validatedAt.Add(-2 * time.Second)),
	}, account) {
		t.Fatal("expected old snapshot to be stale")
	}

	if cloudAccountPermissionSnapshotFresh(models.CloudAccountPermission{}, account) {
		t.Fatal("expected empty snapshot time to be stale")
	}
}

func TestCloudAccountPermissionAttrsAndResp(t *testing.T) {
	checkedAt := models.Time(time.Date(2026, 6, 22, 10, 1, 0, 0, time.UTC))
	account := &models.CloudAccount{
		Provider:       "aws",
		AccountId:      "123456789012",
		TenantId:       "tenant-a",
		Regions:        models.StrSlice{"us-east-1", "us-west-2"},
		SupportedTypes: models.StrSlice{"compute_instance", "kubernetes_cluster"},
	}
	item := resps.CloudAccountPermissionResp{
		Key:      "asset_collection",
		Name:     "资产采集",
		Resource: "cmdb_asset",
		Action:   "read",
		Status:   "pass",
		Message:  "支持采集 2 类资产",
	}

	attrs := cloudAccountPermissionAttrs(account, item, checkedAt, "local_precheck")
	row := models.CloudAccountPermission{OrgId: "org-test", CloudAccountId: "cla-test"}
	applyCloudAccountPermissionAttrs(&row, attrs)
	resp := cloudAccountPermissionRespFromModel(row)

	if row.PermissionKey != item.Key || resp.Key != item.Key {
		t.Fatalf("expected permission key %s, got row=%s resp=%s", item.Key, row.PermissionKey, resp.Key)
	}
	if resp.Source != "local_precheck" || resp.CheckedAt != checkedAt {
		t.Fatalf("unexpected response source/time: %#v", resp)
	}
	if got := resp.Evidence["provider"]; got != "aws" {
		t.Fatalf("expected provider evidence aws, got %#v", got)
	}
	regions, ok := resp.Evidence["regions"].([]string)
	if !ok || len(regions) != 2 || regions[0] != "us-east-1" {
		t.Fatalf("unexpected regions evidence: %#v", resp.Evidence["regions"])
	}
}
