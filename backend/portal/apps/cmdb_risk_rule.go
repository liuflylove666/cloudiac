// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
)

func CmdbRiskRuleConfig(c *ctx.ServiceContext) (*models.CmdbRiskRuleConfig, e.Error) {
	config, err := loadCmdbRiskRuleConfig(c)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func UpdateCmdbRiskRuleConfig(c *ctx.ServiceContext, form *forms.UpdateCmdbRiskRuleConfigForm) (*models.CmdbRiskRuleConfig, e.Error) {
	if err := ensureCmdbRiskRuleManagePermission(c); err != nil {
		return nil, err
	}

	existing := models.CmdbRiskRuleConfig{}
	err := c.DB().Where("org_id = ?", c.OrgId).First(&existing)
	if err != nil && !e.IsRecordNotFound(err) {
		return nil, e.New(e.DBError, err)
	}

	config := cmdbDefaultRiskRuleConfig(c.OrgId)
	exists := !e.IsRecordNotFound(err)
	if exists {
		config = existing
	}
	cmdbRiskRuleConfigApplyForm(&config, form)
	normalizeCmdbRiskRuleConfig(&config)

	if e.IsRecordNotFound(err) {
		config.Id = models.NewId("crr")
		if err := models.Create(c.DB(), &config); err != nil {
			return nil, e.New(e.DBError, err)
		}
		return &config, nil
	}

	config.Id = existing.Id
	if _, err := c.DB().Model(&models.CmdbRiskRuleConfig{}).
		Where("org_id = ?", c.OrgId).
		UpdateAttrs(cmdbRiskRuleConfigAttrs(config)); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return loadCmdbRiskRuleConfig(c)
}

func cmdbRiskRuleConfigApplyForm(config *models.CmdbRiskRuleConfig, form *forms.UpdateCmdbRiskRuleConfigForm) {
	if form.HasKey("changeWindowDays") {
		config.ChangeWindowDays = form.ChangeWindowDays
	}
	if form.HasKey("recentChangeWeight") {
		config.RecentChangeWeight = form.RecentChangeWeight
	}
	if form.HasKey("changedAssetWeight") {
		config.ChangedAssetWeight = form.ChangedAssetWeight
	}
	if form.HasKey("incomingAppWeight") {
		config.IncomingAppWeight = form.IncomingAppWeight
	}
	if form.HasKey("outgoingAppWeight") {
		config.OutgoingAppWeight = form.OutgoingAppWeight
	}
	if form.HasKey("highComplianceRiskWeight") {
		config.HighComplianceRiskWeight = form.HighComplianceRiskWeight
	}
	if form.HasKey("criticalComplianceRiskWeight") {
		config.CriticalComplianceRiskWeight = form.CriticalComplianceRiskWeight
	}
	if form.HasKey("maintenanceLifecycleWeight") {
		config.MaintenanceLifecycleWeight = form.MaintenanceLifecycleWeight
	}
	if form.HasKey("retiredLifecycleWeight") {
		config.RetiredLifecycleWeight = form.RetiredLifecycleWeight
	}
	if form.HasKey("crossBusinessLineWeight") {
		config.CrossBusinessLineWeight = form.CrossBusinessLineWeight
	}
	if form.HasKey("criticalIncomingThreshold") {
		config.CriticalIncomingThreshold = form.CriticalIncomingThreshold
	}
	if form.HasKey("mediumIncomingThreshold") {
		config.MediumIncomingThreshold = form.MediumIncomingThreshold
	}
	if form.HasKey("mediumOutgoingThreshold") {
		config.MediumOutgoingThreshold = form.MediumOutgoingThreshold
	}
	if form.HasKey("criticalScoreThreshold") {
		config.CriticalScoreThreshold = form.CriticalScoreThreshold
	}
	if form.HasKey("highScoreThreshold") {
		config.HighScoreThreshold = form.HighScoreThreshold
	}
	if form.HasKey("mediumScoreThreshold") {
		config.MediumScoreThreshold = form.MediumScoreThreshold
	}
	if form.HasKey("recentCriticalBoost") {
		config.RecentCriticalBoost = form.RecentCriticalBoost
	}
	if form.HasKey("recentHighBoost") {
		config.RecentHighBoost = form.RecentHighBoost
	}
	if form.HasKey("recentMediumBoost") {
		config.RecentMediumBoost = form.RecentMediumBoost
	}
	if form.HasKey("wideDependencyBoost") {
		config.WideDependencyBoost = form.WideDependencyBoost
	}
}

func loadCmdbRiskRuleConfig(c *ctx.ServiceContext) (*models.CmdbRiskRuleConfig, e.Error) {
	config := models.CmdbRiskRuleConfig{}
	err := c.DB().Where("org_id = ?", c.OrgId).First(&config)
	if err != nil {
		if e.IsRecordNotFound(err) {
			config = cmdbDefaultRiskRuleConfig(c.OrgId)
			return &config, nil
		}
		return nil, e.New(e.DBError, err)
	}
	normalizeCmdbRiskRuleConfig(&config)
	return &config, nil
}

func cmdbDefaultRiskRuleConfig(orgId models.Id) models.CmdbRiskRuleConfig {
	return models.CmdbRiskRuleConfig{
		OrgId:                        orgId,
		ChangeWindowDays:             7,
		RecentChangeWeight:           3,
		ChangedAssetWeight:           2,
		IncomingAppWeight:            2,
		OutgoingAppWeight:            1,
		HighComplianceRiskWeight:     8,
		CriticalComplianceRiskWeight: 12,
		MaintenanceLifecycleWeight:   2,
		RetiredLifecycleWeight:       4,
		CrossBusinessLineWeight:      2,
		CriticalIncomingThreshold:    3,
		MediumIncomingThreshold:      3,
		MediumOutgoingThreshold:      5,
		CriticalScoreThreshold:       20,
		HighScoreThreshold:           12,
		MediumScoreThreshold:         6,
		RecentCriticalBoost:          20,
		RecentHighBoost:              12,
		RecentMediumBoost:            6,
		WideDependencyBoost:          4,
	}
}

func normalizeCmdbRiskRuleConfig(config *models.CmdbRiskRuleConfig) {
	defaults := cmdbDefaultRiskRuleConfig(config.OrgId)
	if config.ChangeWindowDays <= 0 {
		config.ChangeWindowDays = defaults.ChangeWindowDays
	}
	if config.CriticalIncomingThreshold <= 0 {
		config.CriticalIncomingThreshold = defaults.CriticalIncomingThreshold
	}
	if config.MediumIncomingThreshold <= 0 {
		config.MediumIncomingThreshold = defaults.MediumIncomingThreshold
	}
	if config.MediumOutgoingThreshold <= 0 {
		config.MediumOutgoingThreshold = defaults.MediumOutgoingThreshold
	}
	if config.CriticalScoreThreshold <= 0 {
		config.CriticalScoreThreshold = defaults.CriticalScoreThreshold
	}
	if config.HighScoreThreshold <= 0 {
		config.HighScoreThreshold = defaults.HighScoreThreshold
	}
	if config.MediumScoreThreshold <= 0 {
		config.MediumScoreThreshold = defaults.MediumScoreThreshold
	}
	if config.CriticalScoreThreshold < config.HighScoreThreshold {
		config.CriticalScoreThreshold = config.HighScoreThreshold
	}
	if config.HighScoreThreshold < config.MediumScoreThreshold {
		config.HighScoreThreshold = config.MediumScoreThreshold
	}
}

func cmdbRiskRuleConfigAttrs(config models.CmdbRiskRuleConfig) map[string]interface{} {
	return map[string]interface{}{
		"change_window_days":              config.ChangeWindowDays,
		"recent_change_weight":            config.RecentChangeWeight,
		"changed_asset_weight":            config.ChangedAssetWeight,
		"incoming_app_weight":             config.IncomingAppWeight,
		"outgoing_app_weight":             config.OutgoingAppWeight,
		"high_compliance_risk_weight":     config.HighComplianceRiskWeight,
		"critical_compliance_risk_weight": config.CriticalComplianceRiskWeight,
		"maintenance_lifecycle_weight":    config.MaintenanceLifecycleWeight,
		"retired_lifecycle_weight":        config.RetiredLifecycleWeight,
		"cross_business_line_weight":      config.CrossBusinessLineWeight,
		"critical_incoming_threshold":     config.CriticalIncomingThreshold,
		"medium_incoming_threshold":       config.MediumIncomingThreshold,
		"medium_outgoing_threshold":       config.MediumOutgoingThreshold,
		"critical_score_threshold":        config.CriticalScoreThreshold,
		"high_score_threshold":            config.HighScoreThreshold,
		"medium_score_threshold":          config.MediumScoreThreshold,
		"recent_critical_boost":           config.RecentCriticalBoost,
		"recent_high_boost":               config.RecentHighBoost,
		"recent_medium_boost":             config.RecentMediumBoost,
		"wide_dependency_boost":           config.WideDependencyBoost,
	}
}
