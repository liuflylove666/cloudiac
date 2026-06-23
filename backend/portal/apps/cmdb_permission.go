// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"fmt"

	"cloudiac/portal/consts"
	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	"cloudiac/portal/models/resps"
	"cloudiac/portal/services"
)

const cmdbOrgRoleComplianceManager = "complianceManager"

var (
	cmdbProjectExportRoles     = []string{consts.ProjectRoleManager, consts.ProjectRoleApprover, consts.ProjectRoleOperator}
	cmdbProjectGovernanceRoles = []string{consts.ProjectRoleManager, consts.ProjectRoleApprover, consts.ProjectRoleOperator}
	cmdbProjectSensitiveRoles  = []string{consts.ProjectRoleManager, consts.ProjectRoleApprover}
)

func CmdbAssetPermissions(c *ctx.ServiceContext) (*resps.CmdbAssetPermissionsResp, e.Error) {
	projectRoles, err := cmdbUserProjectRoles(c)
	if err != nil {
		return nil, err
	}

	canManageAll := cmdbUserCanManageOrg(c)
	hasProjectGovernance := cmdbHasAnyProjectRole(projectRoles, cmdbProjectGovernanceRoles...)
	hasProjectExport := cmdbHasAnyProjectRole(projectRoles, cmdbProjectExportRoles...)
	canManageRiskRules := canManageAll || services.UserHasOrgRole(c.UserId, c.OrgId, cmdbOrgRoleComplianceManager)

	resp := &resps.CmdbAssetPermissionsResp{
		OrgId:                         c.OrgId,
		UserId:                        c.UserId,
		IsSuperAdmin:                  c.IsSuperAdmin,
		OrgRole:                       cmdbUserOrgRole(c),
		ProjectRoles:                  projectRoles,
		CanManageAll:                  canManageAll,
		CanExport:                     canManageAll || hasProjectExport,
		CanImport:                     canManageAll,
		CanEditOwnership:              canManageAll || hasProjectGovernance,
		CanBatchGovernance:            canManageAll || hasProjectGovernance,
		CanManageApplicationRelations: canManageAll || hasProjectGovernance,
		CanManageRiskRules:            canManageRiskRules,
		CanSyncIac:                    canManageAll,
		Permissions:                   make([]resps.CmdbAssetPermissionResp, 0),
	}

	resp.Permissions = append(resp.Permissions,
		cmdbPermissionItem("export", "导出资产", resp.CanExport, "组织管理员可导出全组织资产；项目负责人、审批员或操作员可导出自己项目内资产", append([]string{consts.OrgRoleAdmin}, cmdbProjectExportRoles...)),
		cmdbPermissionItem("import", "导入资产", resp.CanImport, "导入会创建或覆盖组织资产，仅组织管理员或平台管理员可执行", []string{consts.OrgRoleAdmin}),
		cmdbPermissionItem("edit_ownership", "编辑归属", resp.CanEditOwnership, "组织管理员可编辑全部资产；项目负责人、审批员或操作员可编辑自己项目内资产", append([]string{consts.OrgRoleAdmin}, cmdbProjectGovernanceRoles...)),
		cmdbPermissionItem("batch_governance", "批量治理", resp.CanBatchGovernance, "组织管理员可治理全部资产；项目负责人、审批员或操作员可治理自己项目内资产", append([]string{consts.OrgRoleAdmin}, cmdbProjectGovernanceRoles...)),
		cmdbPermissionItem("application_relations", "维护应用依赖", resp.CanManageApplicationRelations, "组织管理员可维护全部应用依赖；项目负责人、审批员或操作员可维护自己项目关联应用", append([]string{consts.OrgRoleAdmin}, cmdbProjectGovernanceRoles...)),
		cmdbPermissionItem("risk_rules", "配置风险规则", resp.CanManageRiskRules, "风险规则影响全组织评分，仅组织管理员、平台管理员或合规管理员可配置", []string{consts.OrgRoleAdmin, cmdbOrgRoleComplianceManager}),
		cmdbPermissionItem("sync_iac", "同步 IaC 资源", resp.CanSyncIac, "同步 IaC 资源会写入组织资产，仅组织管理员或平台管理员可执行", []string{consts.OrgRoleAdmin}),
	)
	return resp, nil
}

func EnsureCmdbOrgAdminPermission(c *ctx.ServiceContext, action string) e.Error {
	if cmdbUserCanManageOrg(c) {
		return nil
	}
	return cmdbPermissionDenied("%s需要组织管理员或平台管理员权限", action)
}

func ensureCmdbAssetExportPermission(c *ctx.ServiceContext, form *forms.ExportCmdbAssetForm) e.Error {
	if cmdbUserCanManageOrg(c) {
		return nil
	}

	rows := make([]struct {
		ProjectId models.Id `gorm:"column:project_id"`
	}, 0)
	if err := buildCmdbAssetExportQuery(c, form).
		Select("distinct iac_cmdb_asset.project_id as project_id").
		Scan(&rows); err != nil {
		return e.New(e.DBError, err)
	}
	for _, row := range rows {
		if row.ProjectId == "" {
			return cmdbPermissionDenied("未绑定项目的资产仅组织管理员或平台管理员可导出")
		}
		if !cmdbUserHasProjectRole(c, row.ProjectId, cmdbProjectExportRoles...) {
			return cmdbPermissionDenied("导出项目 %s 下资产需要项目负责人、审批员、操作员、组织管理员或平台管理员权限", row.ProjectId)
		}
	}
	return nil
}

func ensureCmdbAssetImportPermission(c *ctx.ServiceContext) e.Error {
	return EnsureCmdbOrgAdminPermission(c, "导入 CMDB 资产")
}

func ensureCmdbRiskRuleManagePermission(c *ctx.ServiceContext) e.Error {
	if cmdbUserCanManageOrg(c) || services.UserHasOrgRole(c.UserId, c.OrgId, cmdbOrgRoleComplianceManager) {
		return nil
	}
	return cmdbPermissionDenied("配置应用风险规则需要组织管理员、合规管理员或平台管理员权限")
}

func ensureCmdbApplicationRelationPermission(c *ctx.ServiceContext, application string) e.Error {
	if cmdbUserCanManageOrg(c) {
		return nil
	}
	count, err := c.DB().Model(&models.CmdbAsset{}).
		Joins("join iac_user_project on iac_user_project.project_id = iac_cmdb_asset.project_id").
		Where("iac_cmdb_asset.org_id = ? and iac_cmdb_asset.application = ? and iac_user_project.user_id = ? and iac_user_project.role in (?)",
			c.OrgId, application, c.UserId, cmdbProjectGovernanceRoles).
		Count()
	if err != nil {
		return e.New(e.DBError, err)
	}
	if count > 0 {
		return nil
	}
	return cmdbPermissionDenied("维护应用 %s 依赖关系需要该应用关联项目的负责人、审批员、操作员、组织管理员或平台管理员权限", application)
}

func ensureCmdbAssetOwnershipPermission(c *ctx.ServiceContext, asset models.CmdbAsset, fields []string) e.Error {
	if len(fields) == 0 || cmdbUserCanManageOrg(c) {
		return nil
	}
	if asset.ProjectId == "" {
		return cmdbPermissionDenied("未绑定项目的资产仅组织管理员或平台管理员可编辑归属")
	}
	roles := cmdbProjectGovernanceRoles
	if cmdbOwnershipFieldsRequireSensitiveRole(fields) {
		roles = cmdbProjectSensitiveRoles
	}
	if cmdbUserHasProjectRole(c, asset.ProjectId, roles...) {
		return nil
	}
	if cmdbOwnershipFieldsRequireSensitiveRole(fields) {
		return cmdbPermissionDenied("编辑资产 %s 的合规风险或项目绑定需要项目负责人、审批员、组织管理员或平台管理员权限", asset.Id)
	}
	return cmdbPermissionDenied("编辑资产 %s 归属需要项目负责人、审批员、操作员、组织管理员或平台管理员权限", asset.Id)
}

func cmdbOwnershipFieldsRequireSensitiveRole(fields []string) bool {
	for _, field := range fields {
		switch field {
		case "complianceRisk", "projectId", "envId", "managedBy":
			return true
		}
	}
	return false
}

func cmdbUserCanManageOrg(c *ctx.ServiceContext) bool {
	return c.IsSuperAdmin || services.UserHasOrgRole(c.UserId, c.OrgId, consts.OrgRoleAdmin)
}

func cmdbUserOrgRole(c *ctx.ServiceContext) string {
	if c.IsSuperAdmin {
		return consts.RoleRoot
	}
	if org := services.UserOrgRoles(c.UserId)[c.OrgId]; org != nil {
		return org.Role
	}
	return ""
}

func cmdbUserProjectRoles(c *ctx.ServiceContext) ([]resps.CmdbAssetProjectRoleResp, e.Error) {
	rows := make([]resps.CmdbAssetProjectRoleResp, 0)
	if err := c.DB().Model(&models.UserProject{}).
		Select("iac_user_project.project_id, iac_project.name as project_name, iac_user_project.role").
		Joins("join iac_project on iac_project.id = iac_user_project.project_id").
		Where("iac_user_project.user_id = ? and iac_project.org_id = ?", c.UserId, c.OrgId).
		Order("iac_project.name asc").
		Scan(&rows); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return rows, nil
}

func cmdbHasAnyProjectRole(projectRoles []resps.CmdbAssetProjectRoleResp, roles ...string) bool {
	roleSet := make(map[string]bool, len(roles))
	for _, role := range roles {
		roleSet[role] = true
	}
	for _, projectRole := range projectRoles {
		if roleSet[projectRole.Role] {
			return true
		}
	}
	return false
}

func cmdbUserHasProjectRole(c *ctx.ServiceContext, projectId models.Id, roles ...string) bool {
	for _, role := range roles {
		if services.UserHasProjectRole(c.UserId, c.OrgId, projectId, role) {
			return true
		}
	}
	return false
}

func cmdbPermissionItem(key, name string, allowed bool, message string, roles []string) resps.CmdbAssetPermissionResp {
	if allowed {
		return resps.CmdbAssetPermissionResp{
			Key:           key,
			Name:          name,
			Allowed:       true,
			Message:       message,
			RequiredRoles: roles,
		}
	}
	return resps.CmdbAssetPermissionResp{
		Key:           key,
		Name:          name,
		Allowed:       false,
		Message:       message,
		RequiredRoles: roles,
	}
}

func cmdbPermissionDenied(format string, args ...interface{}) e.Error {
	return e.New(e.ObjectNotExistsOrNoPerm, fmt.Errorf(format, args...))
}
