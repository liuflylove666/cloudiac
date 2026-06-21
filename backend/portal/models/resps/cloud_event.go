// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CloudEventResp struct {
	models.CloudEvent

	ProjectName string `json:"projectName"`
	EnvName     string `json:"envName"`
	AssetName   string `json:"assetName"`
	ActorName   string `json:"actorName"`
}
