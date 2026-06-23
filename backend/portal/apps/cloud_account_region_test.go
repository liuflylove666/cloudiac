package apps

import (
	"testing"
	"time"

	"cloudiac/portal/models"
)

func TestCloudAccountRegionsRespFromValidation(t *testing.T) {
	account := &models.CloudAccount{
		Provider:  "aws",
		AccountId: "123456789012",
		Regions:   models.StrSlice{"us-east-1", "us-west-2"},
		Status:    models.CloudAccountStatusEnabled,
	}
	validation := cloudAccountValidation{
		Ready:               true,
		SupportedAssetTypes: []string{"compute_instance", "kubernetes_cluster"},
		Regions:             []string{"us-east-1", "us-west-2"},
	}

	resp := cloudAccountRegionsRespFromValidation(account, validation)
	if len(resp.Regions) != 2 || len(resp.RegionNames) != 2 {
		t.Fatalf("expected two regions, got %#v", resp)
	}
	first := resp.Regions[0]
	if first.Name != "us-east-1" || !first.Default || !first.Enabled || !first.SyncEnabled {
		t.Fatalf("unexpected first region: %#v", first)
	}
	if first.Source != models.CloudAccountRegionSourceConfigured {
		t.Fatalf("expected configured source, got %s", first.Source)
	}
	if first.Status != models.CloudAccountHealthHealthy {
		t.Fatalf("expected healthy region, got %s", first.Status)
	}
}

func TestCloudAccountRegionStatusMessage(t *testing.T) {
	account := &models.CloudAccount{Provider: "aws", Status: models.CloudAccountStatusEnabled}
	status, message := cloudAccountRegionStatusMessage(account, cloudAccountValidation{
		MissingCredentialKeys: []string{"AWS_SECRET_ACCESS_KEY"},
		Regions:               []string{"us-east-1"},
		SupportedAssetTypes:   []string{"compute_instance"},
	})
	if status != models.CloudAccountHealthUnhealthy || message == "" {
		t.Fatalf("expected unhealthy missing credential message, got %s %q", status, message)
	}

	account.Status = models.CloudAccountStatusDisabled
	status, _ = cloudAccountRegionStatusMessage(account, cloudAccountValidation{
		Regions:             []string{"us-east-1"},
		SupportedAssetTypes: []string{"compute_instance"},
	})
	if status != models.CloudAccountHealthWarning {
		t.Fatalf("expected disabled account to mark region warning, got %s", status)
	}
}

func TestCloudAccountRegionAttrsAndResp(t *testing.T) {
	lastSyncAt := models.Time(time.Date(2026, 6, 22, 11, 0, 0, 0, time.UTC))
	account := &models.CloudAccount{
		Provider:         "oci",
		AccountId:        "tenant-a",
		LastSyncAt:       lastSyncAt,
		ValidationStatus: models.CloudAccountValidationValid,
	}
	validation := cloudAccountValidation{
		SupportedAssetTypes: []string{"compute_instance", "kubernetes_cluster"},
	}

	attrs := cloudAccountRegionAttrs(account, "ap-singapore-1", true, true, true, models.CloudAccountRegionSourceConfigured, models.CloudAccountHealthHealthy, "ok", validation)
	row := models.CloudAccountRegion{OrgId: "org-test", CloudAccountId: "cla-test"}
	applyCloudAccountRegionAttrs(&row, attrs)
	resp := cloudAccountRegionRespFromModel(row)

	if row.Region != "ap-singapore-1" || resp.Name != row.Region {
		t.Fatalf("unexpected region mapping: row=%#v resp=%#v", row, resp)
	}
	if !resp.Default || !resp.Enabled || !resp.SyncEnabled {
		t.Fatalf("unexpected region flags: %#v", resp)
	}
	if resp.LastSyncAt != lastSyncAt {
		t.Fatalf("expected lastSyncAt %v, got %v", lastSyncAt, resp.LastSyncAt)
	}
	if len(resp.ResourceTypes) != 2 || resp.ResourceTypes[1] != "kubernetes_cluster" {
		t.Fatalf("unexpected resource types: %#v", resp.ResourceTypes)
	}
}
