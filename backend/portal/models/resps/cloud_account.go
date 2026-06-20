// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CloudAccountResp struct {
	models.CloudAccount

	Ready                 bool     `json:"ready"`
	RegionCount           int      `json:"regionCount"`
	MissingCredentialKeys []string `json:"missingCredentialKeys"`
}

type CloudAccountValidationResp struct {
	Id                    models.Id `json:"id"`
	Provider              string    `json:"provider"`
	AccountId             string    `json:"accountId"`
	ValidationStatus      string    `json:"validationStatus"`
	ValidationMessage     string    `json:"validationMessage"`
	MissingCredentialKeys []string  `json:"missingCredentialKeys"`
	SupportedAssetTypes   []string  `json:"supportedAssetTypes"`
	Regions               []string  `json:"regions"`
}

type CloudAccountRegionResp struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Default bool   `json:"default"`
	Source  string `json:"source"`
}

type CloudAccountRegionsResp struct {
	Id                    models.Id                `json:"id"`
	Provider              string                   `json:"provider"`
	AccountId             string                   `json:"accountId"`
	Regions               []CloudAccountRegionResp `json:"regions"`
	RegionNames           []string                 `json:"regionNames"`
	MissingCredentialKeys []string                 `json:"missingCredentialKeys"`
}

type CloudAccountPermissionResp struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type CloudAccountPermissionsResp struct {
	Id                    models.Id                    `json:"id"`
	Provider              string                       `json:"provider"`
	AccountId             string                       `json:"accountId"`
	ValidationStatus      string                       `json:"validationStatus"`
	ValidationMessage     string                       `json:"validationMessage"`
	MissingCredentialKeys []string                     `json:"missingCredentialKeys"`
	SupportedAssetTypes   []string                     `json:"supportedAssetTypes"`
	Regions               []string                     `json:"regions"`
	Permissions           []CloudAccountPermissionResp `json:"permissions"`
}
