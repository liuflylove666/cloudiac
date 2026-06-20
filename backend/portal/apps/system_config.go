// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"cloudiac/configs"
	"cloudiac/portal/consts"
	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	"cloudiac/portal/models/resps"
	"cloudiac/portal/services"
	"fmt"
	"strconv"
	"strings"
)

func SearchSystemConfig(c *ctx.ServiceContext) (interface{}, e.Error) {
	rs := make([]resps.SearchSystemConfigResp, 0)
	err := services.QuerySystemConfig(c.DB()).Find(&rs)
	if err != nil {
		return nil, e.New(e.DBError, err)
	}

	for index, cfg := range rs {
		if cfg.Name == models.SysCfgNameTaskStepTimeout {
			timeoutInSecond, err := strconv.Atoi(cfg.Value)
			if err != nil {
				return nil, e.New(e.InternalError, err)
			}
			rs[index].Value = strconv.Itoa(timeoutInSecond / 60)
		}
	}

	return rs, nil
}

func UpdateSystemConfig(c *ctx.ServiceContext, form *forms.UpdateSystemConfigForm) (cfg *models.SystemCfg, err e.Error) {
	tx := c.Tx()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	for _, v := range form.SystemCfg {
		attrs := models.Attrs{}
		attrs["value"] = v.Value
		if _, err := services.UpdateSystemConfig(tx, v.Name, attrs); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, fmt.Errorf("error commit database, err %s", err))
	}

	return nil, nil
}

func GetRegistryAddr(c *ctx.ServiceContext) (interface{}, e.Error) {
	cfgdb := getSystemCfgValue(c, models.SysCfgNamRegistryAddr)

	return &resps.RegistryAddrResp{
		RegistryAddrFromDB:    cfgdb,
		RegistryAddrFromCfg:   configs.Get().RegistryAddr,
		ImageRegistryType:     defaultString(getSystemCfgValue(c, models.SysCfgNameImageRegistryType), "generic"),
		ImageRegistryAddr:     getSystemCfgValue(c, models.SysCfgNameImageRegistryAddr),
		EcrAccountId:          getSystemCfgValue(c, models.SysCfgNameImageRegistryEcrAccountId),
		EcrRegion:             getSystemCfgValue(c, models.SysCfgNameImageRegistryEcrRegion),
		EcrEndpoint:           defaultString(getSystemCfgValue(c, models.SysCfgNameImageRegistryEcrEndpoint), "amazonaws.com"),
		ImageRepositoryPrefix: getSystemCfgValue(c, models.SysCfgNameImageRegistryRepositoryPrefix),
	}, nil
}

func UpsertRegistryAddr(c *ctx.ServiceContext, form *forms.RegistryAddrForm) (interface{}, e.Error) {
	form.RegistryAddr = strings.TrimSpace(form.RegistryAddr)
	form.ImageRegistryType = defaultString(strings.TrimSpace(form.ImageRegistryType), "generic")
	form.ImageRegistryAddr = strings.TrimSpace(form.ImageRegistryAddr)
	form.EcrAccountId = strings.TrimSpace(form.EcrAccountId)
	form.EcrRegion = strings.TrimSpace(form.EcrRegion)
	form.EcrEndpoint = defaultString(strings.TrimSpace(form.EcrEndpoint), "amazonaws.com")
	form.ImageRepositoryPrefix = strings.Trim(strings.TrimSpace(form.ImageRepositoryPrefix), "/")

	if form.ImageRegistryType == "ecr" {
		if form.EcrAccountId == "" || form.EcrRegion == "" {
			return nil, e.New(e.BadRequest, fmt.Errorf("ecr account id and region are required"))
		}
		if form.ImageRegistryAddr == "" {
			form.ImageRegistryAddr = buildEcrRegistryAddr(form.EcrAccountId, form.EcrRegion, form.EcrEndpoint, form.ImageRepositoryPrefix)
		}
	}

	cfg, err := services.UpsertRegistryAddr(c.DB(), form.RegistryAddr)
	if err != nil {
		return nil, e.New(e.DBError, err)
	}

	for _, item := range []struct {
		name  string
		value string
	}{
		{models.SysCfgNameImageRegistryType, form.ImageRegistryType},
		{models.SysCfgNameImageRegistryAddr, form.ImageRegistryAddr},
		{models.SysCfgNameImageRegistryEcrAccountId, form.EcrAccountId},
		{models.SysCfgNameImageRegistryEcrRegion, form.EcrRegion},
		{models.SysCfgNameImageRegistryEcrEndpoint, form.EcrEndpoint},
		{models.SysCfgNameImageRegistryRepositoryPrefix, form.ImageRepositoryPrefix},
	} {
		if _, err := services.UpsertSystemConfigValue(c.DB(), item.name, item.value); err != nil {
			return nil, e.New(e.DBError, err)
		}
	}

	var (
		address = configs.Get().RegistryAddr
	)

	if cfg.Value != "" {
		address = cfg.Value
	}

	if _, err := c.DB().Model(&models.Vcs{}).
		Where("vcs_type = ?", consts.GitTypeRegistry).
		UpdateColumn("address", address); err != nil {
		return nil, e.New(e.DBError, err)
	}

	return &resps.RegistryAddrResp{
		RegistryAddrFromDB:    cfg.Value,
		RegistryAddrFromCfg:   configs.Get().RegistryAddr,
		ImageRegistryType:     form.ImageRegistryType,
		ImageRegistryAddr:     form.ImageRegistryAddr,
		EcrAccountId:          form.EcrAccountId,
		EcrRegion:             form.EcrRegion,
		EcrEndpoint:           form.EcrEndpoint,
		ImageRepositoryPrefix: form.ImageRepositoryPrefix,
	}, nil
}

func GetRegistryAddrStr(c *ctx.ServiceContext) string {
	return services.GetRegistryAddrStr(c.DB())
}

func getSystemCfgValue(c *ctx.ServiceContext, name string) string {
	cfg, err := services.GetSystemConfigByName(c.DB(), name)
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.Value
}

func defaultString(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func buildEcrRegistryAddr(accountId string, region string, endpoint string, repositoryPrefix string) string {
	addr := fmt.Sprintf("%s.dkr.ecr.%s.%s", accountId, region, strings.TrimPrefix(endpoint, "."))
	if repositoryPrefix != "" {
		addr = fmt.Sprintf("%s/%s", strings.TrimRight(addr, "/"), repositoryPrefix)
	}
	return strings.TrimRight(addr, "/") + "/"
}
