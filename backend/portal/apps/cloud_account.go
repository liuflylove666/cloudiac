// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/libs/page"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	"cloudiac/portal/models/resps"
	"cloudiac/portal/services"
	"cloudiac/utils"
)

type cloudAccountValidation struct {
	Ready                 bool
	Status                string
	Message               string
	MissingCredentialKeys []string
	SupportedAssetTypes   []string
	Regions               []string
}

func SearchCloudAccounts(c *ctx.ServiceContext, form *forms.SearchCloudAccountForm) (interface{}, e.Error) {
	query := services.QueryCloudAccount(c.DB()).Where("org_id = ?", c.OrgId)
	if form.Q != "" {
		qs := "%" + form.Q + "%"
		query = query.Where("name like ? or description like ? or account_id like ?", qs, qs, qs)
	}
	if form.Provider != "" {
		query = query.Where("provider = ?", normalizeProvider(form.Provider))
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	if form.ValidationStatus != "" {
		query = query.Where("validation_status = ?", form.ValidationStatus)
	}
	if form.SortField() == "" {
		query = query.Order("created_at desc")
	} else {
		query = form.Order(query)
	}

	accounts := make([]models.CloudAccount, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&accounts); err != nil {
		return nil, e.New(e.DBError, err)
	}

	list := make([]resps.CloudAccountResp, 0, len(accounts))
	for _, account := range accounts {
		list = append(list, cloudAccountResp(account))
	}

	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func CloudAccountDetail(c *ctx.ServiceContext, form *forms.CloudAccountParam) (*resps.CloudAccountResp, e.Error) {
	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, form.Id)
	if err != nil {
		return nil, err
	}
	resp := cloudAccountResp(*account)
	return &resp, nil
}

func CreateCloudAccount(c *ctx.ServiceContext, form *forms.CreateCloudAccountForm) (*resps.CloudAccountResp, e.Error) {
	c.AddLogField("action", fmt.Sprintf("create cloud account %s", form.Name))

	provider := normalizeProvider(form.Provider)
	regions := normalizeStringList(form.Regions)
	credentials, err := marshalCloudCredentialParams(form.Credentials, nil)
	if err != nil {
		return nil, e.New(e.BadParam, err)
	}
	credentialMap := credentialMapFromCloudCredentials(credentials)
	regions = firstNonEmptyStringSlice(regions, inferCloudRegions(provider, credentialMap))

	account := &models.CloudAccount{
		OrgId:       c.OrgId,
		Name:        form.Name,
		Description: form.Description,
		Provider:    provider,
		AccountId:   inferCloudAccountId(provider, form.AccountId, form.TenantId, credentialMap),
		TenantId:    form.TenantId,
		Regions:     models.StrSlice(regions),
		RunnerTags:  models.StrSlice(normalizeStringList(form.RunnerTags)),
		Credentials: credentials,
		Status:      firstNonEmpty(form.Status, models.CloudAccountStatusEnabled),
	}
	account.Id = models.NewId("cla")
	applyCloudAccountValidation(account)

	created, createErr := services.CreateCloudAccount(c.DB(), account)
	if createErr != nil {
		return nil, createErr
	}
	resp := cloudAccountResp(*created)
	return &resp, nil
}

func UpdateCloudAccount(c *ctx.ServiceContext, form *forms.UpdateCloudAccountForm) (*resps.CloudAccountResp, e.Error) {
	c.AddLogField("action", fmt.Sprintf("update cloud account %s", form.Id))

	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, form.Id)
	if err != nil {
		return nil, err
	}

	attrs := models.Attrs{}
	if form.HasKey("name") {
		attrs["name"] = form.Name
		account.Name = form.Name
	}
	if form.HasKey("description") {
		attrs["description"] = form.Description
		account.Description = form.Description
	}
	if form.HasKey("provider") {
		account.Provider = normalizeProvider(form.Provider)
		attrs["provider"] = account.Provider
	}
	if form.HasKey("tenantId") {
		attrs["tenant_id"] = form.TenantId
		account.TenantId = form.TenantId
	}
	if form.HasKey("regions") {
		regions := normalizeStringList(form.Regions)
		attrs["regions"] = models.StrSlice(regions)
		account.Regions = models.StrSlice(regions)
	}
	if form.HasKey("runnerTags") {
		runnerTags := normalizeStringList(form.RunnerTags)
		attrs["runner_tags"] = models.StrSlice(runnerTags)
		account.RunnerTags = models.StrSlice(runnerTags)
	}
	if form.HasKey("credentials") {
		credentials, err := marshalCloudCredentialParams(form.Credentials, encryptedCloudCredentialValues(account.Credentials))
		if err != nil {
			return nil, e.New(e.BadParam, err)
		}
		attrs["credentials"] = credentials
		account.Credentials = credentials
	}
	if form.HasKey("accountId") {
		attrs["account_id"] = form.AccountId
		account.AccountId = form.AccountId
	}
	if form.HasKey("status") {
		attrs["status"] = form.Status
		account.Status = form.Status
	}
	if form.HasKey("provider") || form.HasKey("regions") || form.HasKey("credentials") || form.HasKey("accountId") || form.HasKey("tenantId") {
		credentials := credentialMapFromCloudCredentials(account.Credentials)
		account.AccountId = inferCloudAccountId(account.Provider, account.AccountId, account.TenantId, credentials)
		account.Regions = models.StrSlice(firstNonEmptyStringSlice([]string(account.Regions), inferCloudRegions(account.Provider, credentials)))
		attrs["account_id"] = account.AccountId
		attrs["regions"] = account.Regions
		validationAttrs := cloudAccountValidationAttrs(account)
		for key, value := range validationAttrs {
			attrs[key] = value
		}
	}
	if len(attrs) == 0 {
		resp := cloudAccountResp(*account)
		return &resp, nil
	}

	updated, updateErr := services.UpdateCloudAccount(c.DB(), c.OrgId, form.Id, attrs)
	if updateErr != nil {
		return nil, updateErr
	}
	resp := cloudAccountResp(*updated)
	return &resp, nil
}

func DeleteCloudAccount(c *ctx.ServiceContext, form *forms.CloudAccountParam) (interface{}, e.Error) {
	c.AddLogField("action", fmt.Sprintf("delete cloud account %s", form.Id))
	return nil, services.DeleteCloudAccount(c.DB(), c.OrgId, form.Id)
}

func ValidateCloudAccount(c *ctx.ServiceContext, form *forms.CloudAccountParam) (*resps.CloudAccountValidationResp, e.Error) {
	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, form.Id)
	if err != nil {
		return nil, err
	}

	attrs := cloudAccountValidationAttrs(account)
	if _, err := c.DB().Model(&models.CloudAccount{}).
		Where("id = ? and org_id = ?", form.Id, c.OrgId).
		UpdateAttrs(attrs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	for key, value := range attrs {
		switch key {
		case "validation_status":
			account.ValidationStatus = value.(string)
		case "validation_message":
			account.ValidationMessage = value.(string)
		case "last_validated_at":
			account.LastValidatedAt = value.(models.Time)
		case "supported_types":
			account.SupportedTypes = value.(models.StrSlice)
		case "regions":
			account.Regions = value.(models.StrSlice)
		}
	}

	validation := validateCloudAccount(account)
	return &resps.CloudAccountValidationResp{
		Id:                    account.Id,
		Provider:              account.Provider,
		AccountId:             account.AccountId,
		ValidationStatus:      account.ValidationStatus,
		ValidationMessage:     account.ValidationMessage,
		MissingCredentialKeys: validation.MissingCredentialKeys,
		SupportedAssetTypes:   validation.SupportedAssetTypes,
		Regions:               validation.Regions,
	}, nil
}

func CloudAccountRegions(c *ctx.ServiceContext, form *forms.CloudAccountParam) (*resps.CloudAccountRegionsResp, e.Error) {
	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, form.Id)
	if err != nil {
		return nil, err
	}
	return cloudAccountRegionsResp(account), nil
}

func UpdateCloudAccountRegions(c *ctx.ServiceContext, form *forms.UpdateCloudAccountRegionsForm) (*resps.CloudAccountRegionsResp, e.Error) {
	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, form.Id)
	if err != nil {
		return nil, err
	}

	regions := normalizeStringList(form.Regions)
	account.Regions = models.StrSlice(regions)
	attrs := models.Attrs{
		"regions": account.Regions,
	}
	for key, value := range cloudAccountValidationAttrs(account) {
		attrs[key] = value
	}

	updated, updateErr := services.UpdateCloudAccount(c.DB(), c.OrgId, form.Id, attrs)
	if updateErr != nil {
		return nil, updateErr
	}
	return cloudAccountRegionsResp(updated), nil
}

func CloudAccountPermissions(c *ctx.ServiceContext, form *forms.CloudAccountParam) (*resps.CloudAccountPermissionsResp, e.Error) {
	account, err := services.GetCloudAccountById(c.DB(), c.OrgId, form.Id)
	if err != nil {
		return nil, err
	}

	validation := validateCloudAccount(account)
	return &resps.CloudAccountPermissionsResp{
		Id:                    account.Id,
		Provider:              account.Provider,
		AccountId:             account.AccountId,
		ValidationStatus:      account.ValidationStatus,
		ValidationMessage:     validation.Message,
		MissingCredentialKeys: validation.MissingCredentialKeys,
		SupportedAssetTypes:   validation.SupportedAssetTypes,
		Regions:               validation.Regions,
		Permissions:           cloudAccountPermissionItems(account, validation),
	}, nil
}

func cloudAccountResp(account models.CloudAccount) resps.CloudAccountResp {
	validation := validateCloudAccount(&account)
	account.Credentials = maskedCloudCredentials(account.Credentials)
	if len(account.SupportedTypes) == 0 {
		account.SupportedTypes = models.StrSlice(validation.SupportedAssetTypes)
	}
	if len(account.Regions) == 0 {
		account.Regions = models.StrSlice(validation.Regions)
	}
	return resps.CloudAccountResp{
		CloudAccount:          account,
		Ready:                 validation.Ready,
		RegionCount:           len(account.Regions),
		MissingCredentialKeys: validation.MissingCredentialKeys,
	}
}

func cloudAccountRegionsResp(account *models.CloudAccount) *resps.CloudAccountRegionsResp {
	validation := validateCloudAccount(account)
	source := "inferred"
	if len(account.Regions) > 0 {
		source = "configured"
	}
	regions := make([]resps.CloudAccountRegionResp, 0, len(validation.Regions))
	for index, region := range validation.Regions {
		regions = append(regions, resps.CloudAccountRegionResp{
			Name:    region,
			Enabled: true,
			Default: index == 0,
			Source:  source,
		})
	}
	return &resps.CloudAccountRegionsResp{
		Id:                    account.Id,
		Provider:              account.Provider,
		AccountId:             account.AccountId,
		Regions:               regions,
		RegionNames:           validation.Regions,
		MissingCredentialKeys: validation.MissingCredentialKeys,
	}
}

func cloudAccountPermissionItems(account *models.CloudAccount, validation cloudAccountValidation) []resps.CloudAccountPermissionResp {
	accountStatus := "pass"
	accountMessage := "账号已启用"
	if account.Status == models.CloudAccountStatusDisabled {
		accountStatus = "fail"
		accountMessage = "账号已禁用，不能用于同步或云操作"
	}

	credentialStatus := "pass"
	credentialMessage := "必需凭证字段完整"
	if len(validation.MissingCredentialKeys) > 0 {
		credentialStatus = "fail"
		credentialMessage = fmt.Sprintf("缺少凭证字段：%s", strings.Join(validation.MissingCredentialKeys, ", "))
	}

	regionStatus := "pass"
	regionMessage := "已配置可用区域"
	if len(validation.Regions) == 0 {
		regionStatus = "fail"
		regionMessage = "未配置或无法推断可用区域"
	}

	assetStatus := "pass"
	assetMessage := fmt.Sprintf("支持采集 %d 类资产", len(validation.SupportedAssetTypes))
	if len(validation.SupportedAssetTypes) == 0 {
		assetStatus = "fail"
		assetMessage = fmt.Sprintf("暂不支持 provider %s 的资产采集", account.Provider)
	} else if !validation.Ready {
		assetStatus = "warn"
		assetMessage = "资产采集能力已声明，但账号状态或凭证尚未满足执行条件"
	}

	return []resps.CloudAccountPermissionResp{
		{
			Key:      "account_status",
			Name:     "账号状态",
			Resource: "cloud_account",
			Action:   "use",
			Status:   accountStatus,
			Message:  accountMessage,
		},
		{
			Key:      "credential_validation",
			Name:     "凭证完整性",
			Resource: "credential",
			Action:   "validate",
			Status:   credentialStatus,
			Message:  credentialMessage,
		},
		{
			Key:      "region_scope",
			Name:     "区域范围",
			Resource: "region",
			Action:   "list",
			Status:   regionStatus,
			Message:  regionMessage,
		},
		{
			Key:      "asset_collection",
			Name:     "资产采集",
			Resource: "cmdb_asset",
			Action:   "read",
			Status:   assetStatus,
			Message:  assetMessage,
		},
	}
}

func applyCloudAccountValidation(account *models.CloudAccount) {
	attrs := cloudAccountValidationAttrs(account)
	account.ValidationStatus = attrs["validation_status"].(string)
	account.ValidationMessage = attrs["validation_message"].(string)
	account.LastValidatedAt = attrs["last_validated_at"].(models.Time)
	account.SupportedTypes = attrs["supported_types"].(models.StrSlice)
	if regions, ok := attrs["regions"]; ok {
		account.Regions = regions.(models.StrSlice)
	}
}

func cloudAccountValidationAttrs(account *models.CloudAccount) map[string]interface{} {
	validation := validateCloudAccount(account)
	status := models.CloudAccountValidationInvalid
	if validation.Ready {
		status = models.CloudAccountValidationValid
	}
	attrs := map[string]interface{}{
		"validation_status":  status,
		"validation_message": validation.Message,
		"last_validated_at":  models.Time(time.Now()),
		"supported_types":    models.StrSlice(validation.SupportedAssetTypes),
	}
	if len(account.Regions) == 0 && len(validation.Regions) > 0 {
		attrs["regions"] = models.StrSlice(validation.Regions)
	}
	return attrs
}

func validateCloudAccount(account *models.CloudAccount) cloudAccountValidation {
	provider := normalizeProvider(account.Provider)
	credentials := credentialMapWithRegions(provider, credentialMapFromCloudCredentials(account.Credentials), []string(account.Regions))
	missing := missingCloudCredentialKeys(provider, credentials)
	regions := firstNonEmptyStringSlice([]string(account.Regions), inferCloudRegions(provider, credentials))
	supportedTypes := supportedCmdbCloudAssetTypes(provider)

	message := "cloud account credentials are valid for local checks"
	ready := len(missing) == 0 && len(supportedTypes) > 0 && account.Status != models.CloudAccountStatusDisabled
	if len(supportedTypes) == 0 {
		ready = false
		message = fmt.Sprintf("unsupported cloud provider %s", provider)
	} else if len(missing) > 0 {
		message = fmt.Sprintf("missing credentials: %s", strings.Join(missing, ", "))
	} else if account.Status == models.CloudAccountStatusDisabled {
		message = "cloud account is disabled"
	}

	return cloudAccountValidation{
		Ready:                 ready,
		Message:               message,
		MissingCredentialKeys: missing,
		SupportedAssetTypes:   supportedTypes,
		Regions:               regions,
	}
}

func marshalCloudCredentialParams(params []forms.Params, previous map[string]string) (models.JSON, error) {
	normalized := make([]forms.Params, 0, len(params))
	for _, param := range params {
		param.Key = strings.ToUpper(strings.TrimSpace(param.Key))
		if param.Key == "" {
			continue
		}
		if param.Id == "" {
			param.Id = param.Key
		}
		if param.IsSecret == nil {
			isSecret := false
			param.IsSecret = &isSecret
		}
		if *param.IsSecret {
			if param.Value == "" {
				param.Value = firstNonEmpty(previous[param.Id], previous[param.Key])
			} else {
				encrypted, err := utils.AesEncrypt(param.Value)
				if err != nil {
					return nil, err
				}
				param.Value = encrypted
			}
		}
		normalized = append(normalized, param)
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	return models.JSON(data), nil
}

func maskedCloudCredentials(credentials models.JSON) models.JSON {
	params := cloudCredentialParams(credentials)
	for index := range params {
		if params[index].IsSecret != nil && *params[index].IsSecret {
			params[index].Value = ""
		}
	}
	data, _ := json.Marshal(params)
	return models.JSON(data)
}

func credentialMapFromCloudCredentials(credentials models.JSON) map[string]string {
	result := make(map[string]string)
	for _, param := range cloudCredentialParams(credentials) {
		key := strings.ToUpper(strings.TrimSpace(param.Key))
		if key == "" {
			continue
		}
		value := param.Value
		if param.IsSecret != nil && *param.IsSecret && value != "" {
			if decrypted, err := utils.AesDecrypt(value); err == nil {
				value = decrypted
			}
		}
		result[key] = value
	}
	return result
}

func encryptedCloudCredentialValues(credentials models.JSON) map[string]string {
	result := make(map[string]string)
	for _, param := range cloudCredentialParams(credentials) {
		result[param.Id] = param.Value
		result[strings.ToUpper(strings.TrimSpace(param.Key))] = param.Value
	}
	return result
}

func cloudCredentialParams(credentials models.JSON) []forms.Params {
	params := make([]forms.Params, 0)
	if credentials.IsNull() {
		return params
	}
	_ = json.Unmarshal(credentials, &params)
	return params
}

func credentialMapWithRegions(provider string, credentials map[string]string, regions []string) map[string]string {
	result := make(map[string]string, len(credentials)+2)
	for key, value := range credentials {
		result[key] = value
	}
	if len(regions) == 0 {
		return result
	}
	joined := strings.Join(regions, ",")
	switch provider {
	case "aws":
		result["AWS_REGIONS"] = firstNonEmpty(result["AWS_REGIONS"], joined)
		result["AWS_REGION"] = firstNonEmpty(result["AWS_REGION"], regions[0])
	case "oci":
		result["OCI_REGIONS"] = firstNonEmpty(result["OCI_REGIONS"], joined)
		result["OCI_REGION"] = firstNonEmpty(result["OCI_REGION"], regions[0])
	case "alicloud":
		result["ALICLOUD_REGIONS"] = firstNonEmpty(result["ALICLOUD_REGIONS"], joined)
		result["ALICLOUD_REGION"] = firstNonEmpty(result["ALICLOUD_REGION"], regions[0])
	}
	return result
}

func inferCloudAccountId(provider, accountId, tenantId string, credentials map[string]string) string {
	switch provider {
	case "aws":
		return firstNonEmpty(accountId, credentials["AWS_ACCOUNT_ID"])
	case "oci":
		return firstNonEmpty(accountId, tenantId, credentials["OCI_TENANCY_OCID"])
	case "alicloud":
		return firstNonEmpty(accountId, credentials["ALICLOUD_ACCOUNT_ID"])
	default:
		return firstNonEmpty(accountId, tenantId)
	}
}

func firstNonEmptyStringSlice(values ...[]string) []string {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}
