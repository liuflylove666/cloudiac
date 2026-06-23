package apps

import (
	"testing"

	"cloudiac/portal/models"
)

func TestCloudRiskS3ConfigCandidates(t *testing.T) {
	asset := models.CmdbAsset{
		TimedModel: models.TimedModel{BaseModel: models.BaseModel{Id: models.Id("ci-s3")}},
		Provider:   "aws",
		AssetType:  models.CmdbAssetTypeObjectStorageBucket,
		NativeType: "aws_s3_bucket",
		NativeId:   "audit-bucket",
		Name:       "audit-bucket",
		Owner:      "platform",
		Attributes: models.ResAttrs{
			"bucket":                   "audit-bucket",
			"bucketPolicy":             models.ResAttrs{"publicAllow": true, "statementCount": 1, "version": "2012-10-17"},
			"bucketAclPublic":          true,
			"bucketAclGrantCount":      2,
			"publicAccessBlockEnabled": false,
			"publicAccessBlock":        models.ResAttrs{"blockPublicAcls": true, "ignorePublicAcls": false},
			"encryptionEnabled":        false,
			"versioning":               models.ResAttrs{"enabled": false, "status": "Suspended"},
			"versioningStatus":         "Suspended",
			"logging":                  models.ResAttrs{"enabled": false},
			"loggingEnabled":           false,
			"objectLock":               models.ResAttrs{"enabled": false},
			"objectLockEnabled":        false,
		},
	}

	candidates := cloudRiskCandidatesFromAsset(asset)
	got := map[string]cloudRiskCandidate{}
	for _, candidate := range candidates {
		got[candidate.RuleKey] = candidate
	}

	expected := map[string]string{
		"s3_bucket_public_policy":         models.CloudOperationRiskCritical,
		"s3_bucket_public_acl":            models.CloudOperationRiskCritical,
		"s3_public_access_block_disabled": models.CloudOperationRiskHigh,
		"s3_default_encryption_disabled":  models.CloudOperationRiskHigh,
		"s3_versioning_disabled":          models.CloudOperationRiskMedium,
		"s3_access_logging_disabled":      models.CloudOperationRiskMedium,
		"s3_object_lock_disabled":         models.CloudOperationRiskMedium,
	}
	for key, level := range expected {
		candidate, ok := got[key]
		if !ok {
			t.Fatalf("expected %s risk, got %#v", key, got)
		}
		if candidate.RiskLevel != level {
			t.Fatalf("unexpected risk level for %s: %s", key, candidate.RiskLevel)
		}
		if candidate.Source != models.CloudRiskSourceCloudConfig {
			t.Fatalf("unexpected source for %s: %s", key, candidate.Source)
		}
	}
	if got["s3_bucket_public_policy"].Evidence["bucket"] != "audit-bucket" {
		t.Fatalf("expected bucket evidence, got %#v", got["s3_bucket_public_policy"].Evidence)
	}
}

func TestCloudRiskS3ConfigCandidatesIgnoreNonAwsBuckets(t *testing.T) {
	asset := models.CmdbAsset{
		TimedModel: models.TimedModel{BaseModel: models.BaseModel{Id: models.Id("ci-gcs")}},
		Provider:   "gcp",
		AssetType:  models.CmdbAssetTypeObjectStorageBucket,
		NativeId:   "gcs-bucket",
		Owner:      "platform",
		Attributes: models.ResAttrs{
			"bucketPolicy":             models.ResAttrs{"publicAllow": true},
			"bucketAclPublic":          true,
			"publicAccessBlockEnabled": false,
			"encryptionEnabled":        false,
			"versioningStatus":         "Suspended",
			"loggingEnabled":           false,
		},
	}

	candidates := cloudRiskCandidatesFromAsset(asset)
	for _, candidate := range candidates {
		if len(candidate.RuleKey) >= 3 && candidate.RuleKey[:3] == "s3_" {
			t.Fatalf("did not expect S3 risk for non-AWS bucket: %#v", candidate)
		}
	}
}

func TestCloudRiskS3VersioningMissingDoesNotCreateRisk(t *testing.T) {
	asset := models.CmdbAsset{
		TimedModel: models.TimedModel{BaseModel: models.BaseModel{Id: models.Id("ci-s3")}},
		Provider:   "aws",
		AssetType:  models.CmdbAssetTypeObjectStorageBucket,
		NativeType: "aws_s3_bucket",
		NativeId:   "logs-bucket",
		Owner:      "platform",
		Attributes: models.ResAttrs{
			"bucket": "logs-bucket",
		},
	}

	for _, candidate := range cloudRiskCandidatesFromAsset(asset) {
		if candidate.RuleKey == "s3_versioning_disabled" {
			t.Fatalf("versioning risk should require collected versioning evidence")
		}
	}
}
