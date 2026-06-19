// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"fmt"
	"strings"
	"time"

	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models"
	"cloudiac/portal/models/resps"
)

type cmdbCloudCollectResult struct {
	Assets []*models.CmdbAsset
	Stats  models.ResAttrs
	Err    error
}

func collectCmdbCloudAssets(account *cmdbCloudAccount, regions, assetTypes []string) cmdbCloudCollectResult {
	switch account.Provider {
	case "aws":
		return collectCmdbAwsAssets(account, regions, assetTypes)
	case "oci":
		return collectCmdbOciAssets(account, regions, assetTypes)
	case "alicloud":
		return cmdbCloudCollectResult{
			Stats: models.ResAttrs{
				"regions":    regions,
				"assetTypes": assetTypes,
			},
			Err: fmt.Errorf("alicloud collector sdk is not wired yet; IaC resource backfill is available"),
		}
	default:
		return cmdbCloudCollectResult{
			Stats: models.ResAttrs{
				"regions":    regions,
				"assetTypes": assetTypes,
			},
			Err: fmt.Errorf("unsupported cloud provider %s", account.Provider),
		}
	}
}

func upsertCmdbCloudAssets(c *ctx.ServiceContext, assets []*models.CmdbAsset) (*resps.CmdbBackfillResp, e.Error) {
	resp := &resps.CmdbBackfillResp{}
	for _, asset := range assets {
		result, err := upsertCmdbAsset(c, asset, models.CmdbAssetSourceCloudCollect)
		if err != nil {
			return nil, err
		}
		switch {
		case result.Created:
			resp.Created++
		case result.Updated:
			resp.Updated++
		default:
			resp.Skipped++
		}
	}
	return resp, nil
}

func newCmdbCloudAsset(account *cmdbCloudAccount, region, assetType, nativeType, nativeId, name string, now models.Time) *models.CmdbAsset {
	accountId := firstNonEmpty(account.Credentials["AWS_ACCOUNT_ID"], account.Credentials["OCI_TENANCY_OCID"], string(account.Id))
	return &models.CmdbAsset{
		OrgId:      "",
		Source:     models.CmdbAssetSourceCloudCollect,
		Provider:   account.Provider,
		AccountId:  accountId,
		Region:     region,
		AssetType:  assetType,
		NativeType: nativeType,
		NativeId:   nativeId,
		Name:       firstNonEmpty(name, nativeId),
		Tags:       models.ResAttrs{},
		Attributes: models.ResAttrs{},
		RawData: models.ResAttrs{
			"accountSource": account.Source,
			"accountRefId":  account.Id,
			"provider":      account.Provider,
			"region":        region,
			"nativeType":    nativeType,
			"nativeId":      nativeId,
		},
		LastSyncAt: now,
	}
}

func selectedCmdbAssetTypes(assetTypes []string) map[string]bool {
	selected := make(map[string]bool)
	for _, assetType := range assetTypes {
		assetType = strings.TrimSpace(assetType)
		if assetType == "" {
			continue
		}
		selected[assetType] = true
	}
	return selected
}

func wantsCmdbAssetType(selected map[string]bool, assetType string) bool {
	return len(selected) == 0 || selected[assetType]
}

func mergeCmdbStats(dst models.ResAttrs, src models.ResAttrs) {
	for key, value := range src {
		dst[key] = value
	}
}

func cmdbCollectError(errors []string) error {
	if len(errors) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(errors, "; "))
}

func cmdbNow() models.Time {
	return models.Time(time.Now())
}
