// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package v1

import (
	"cloudiac/common"
	"cloudiac/portal/libs/ctrl"
	"cloudiac/portal/web/api/v1/handlers"
	"cloudiac/portal/web/middleware"

	"github.com/gin-gonic/gin"
)

// @title 云霁 CloudIaC 基础设施即代码管理平台
// @version 1.0.0
// @description CloudIaC 是基于基础设施即代码构建的云环境自动化管理平台。
// @description CloudIaC 将易于使用的界面与强大的治理工具相结合，让您和您团队的成员可以快速轻松的在云中部署和管理环境。
// @description 通过将 CloudIaC 集成到您的流程中，您可以获得对组织的云使用情况的可见性、可预测性和更好的治理。

// @BasePath /api/v1
// @schemes http

// @securityDefinitions.apikey AuthToken
// @in header
// @name Authorization

func Register(g *gin.RouterGroup) {
	w := ctrl.WrapHandler
	ac := middleware.AccessControl

	// 非授权用户相关路由
	g.Any("/check", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"success": true,
			"version": common.VERSION,
			"build":   common.BUILD,
		})
	})

	g.Use(gin.Logger())

	g.POST("/trigger/send", w(handlers.ApiTriggerHandler))
	g.POST("/cloud/webhooks/deliveries/:deliveryId/signature-verification/report", w(handlers.CloudWebhook{}.ReportSignatureVerificationByToken))

	// sso token 验证
	g.GET("/sso/tokens/verify", w(handlers.VerifySsoToken))

	// 触发器
	apiToken := g.Group("")
	apiToken.Use(w(middleware.AuthApiToken))
	apiToken.POST("/webhooks/:vcsType/:vcsId", w(handlers.WebhooksApiHandler))

	g.POST("/auth/register", w(handlers.Auth{}.Registry))
	g.POST("/auth/login", w(handlers.Auth{}.Login))
	g.GET("/auth/email", w(handlers.Auth{}.CheckEmail))
	g.POST("/auth/password/reset/email", w(handlers.Auth{}.PasswordResetEmail))

	// 重新发送邮件
	g.GET("/activation/retry", w(handlers.User{}.ActiveUserEmailRetry))
	// 过期重启发送邮件
	g.GET("/activation/expired/retry", w(handlers.User{}.ActiveUserEmailExpiredRetry))

	g.GET("/system_config/switches", w(handlers.SystemSwitchesStatus))

	// Authorization Header 鉴权
	g.Use(w(middleware.Auth)) // 解析 header token

	// 激活邮箱
	g.POST("/activation", w(handlers.User{}.ActiveUserEmail))
	// 重置密码
	g.PUT("/auth/password/reset", w(handlers.Auth{}.PasswordReset))

	// 允许搜索组织内所有用户信息
	g.GET("/users/all", w(handlers.User{}.SearchAllUsers))
	// 创建单点登录 token
	g.POST("/sso/tokens", w(handlers.GenerateSsoToken))
	// ctrl.Register(g.Group("tokens", ac()), &handlers.Token{})

	// TODO 旧的 token 接口己不再使用，先注释, 后续版本删除
	// ctrl.Register(g.Group("token", ac()), &handlers.Auth{})

	g.GET("/auth/me", ac("self", "read"), w(handlers.Auth{}.GetUserByToken))
	g.PUT("/users/self", ac("self", "update"), w(handlers.User{}.UpdateSelf))
	//todo runner list权限怎么划分
	g.GET("/runners", ac(), w(handlers.RunnerSearch))
	g.PUT("/consul/tags/update", ac(), w(handlers.ConsulTagUpdate))
	g.GET("/consul/kv/search", ac(), w(handlers.ConsulKVSearch))
	g.GET("/runners/tags", ac(), w(handlers.RunnerTags)) // 返回所有的runner tags

	ctrl.Register(g.Group("orgs", ac()), &handlers.Organization{})
	g.PUT("/orgs/:id/status", ac(), w(handlers.Organization{}.ChangeOrgStatus))
	ctrl.Register(g.Group("users", ac()), &handlers.User{})
	g.PUT("/users/:id/status", ac(), w(handlers.User{}.ChangeUserStatus))
	g.POST("/users/:id/password/reset", ac(), w(handlers.User{}.PasswordReset))

	// 系统配置
	g.PUT("/systems", ac(), w(handlers.SystemConfig{}.Update))
	g.GET("/systems", ac(), w(handlers.SystemConfig{}.Search))
	// 系统状态
	g.GET("/systems/status", w(handlers.PortalSystemStatusSearch))

	//清理缓存
	g.POST("/systems/provider_cache/remove", ac(), w(handlers.SystemProviderCacheClear))

	// 系统设置registry addr 配置
	g.GET("/system_config/registry/addr", ac(), w(handlers.GetRegistryAddr))     // 获取registry地址的设置
	g.POST("/system_config/registry/addr", ac(), w(handlers.UpsertRegistryAddr)) // 更新registry地址的设置

	// 平台概览
	g.GET("/platform/stat/basedata", ac(), w(handlers.Platform{}.PlatformStatBasedata))
	g.GET("/platform/stat/provider/env", ac(), w(handlers.Platform{}.PlatformStatProEnv))
	g.GET("/platform/stat/provider/resource", ac(), w(handlers.Platform{}.PlatformStatProRes))
	g.GET("/platform/stat/resource/type", ac(), w(handlers.Platform{}.PlatformStatResType))
	g.GET("/platform/stat/resource/week", ac(), w(handlers.Platform{}.PlatformStatResWeekChange))
	g.GET("/platform/stat/resource/active", ac(), w(handlers.Platform{}.PlatformStatActiveResType))
	// 平台概览-合规相关统计
	g.GET("/platform/stat/pg", ac(), w(handlers.Platform{}.PlatformStatPg))
	g.GET("/platform/stat/policy", ac(), w(handlers.Platform{}.PlatformStatPolicy))
	g.GET("/platform/stat/pg_stack/enabled", ac(), w(handlers.Platform{}.PlatformStatPgStackEnabled))
	g.GET("/platform/stat/pg_env/enabled_activate", ac(), w(handlers.Platform{}.PlatformStatPgEnvEnabledActivate))
	g.GET("/platform/stat/pg_stack/ng", ac(), w(handlers.Platform{}.PlatformStatPgStackNG))
	g.GET("/platform/stat/pg_env/ng_activate", ac(), w(handlers.Platform{}.PlatformStatPgEnvNGActivate))

	// 平台概览-当日统计数据
	g.GET("/platform/stat/today/org", ac(), w(handlers.Platform{}.PlatformStatTodayOrg))
	g.GET("/platform/stat/today/project", ac(), w(handlers.Platform{}.PlatformStatTodayProject))
	g.GET("/platform/stat/today/template", ac(), w(handlers.Platform{}.PlatformStatTodayStack))
	g.GET("/platform/stat/today/pg", ac(), w(handlers.Platform{}.PlatformStatTodayPG))
	g.GET("/platform/stat/today/env", ac(), w(handlers.Platform{}.PlatformStatTodayEnv))
	g.GET("/platform/stat/today/destroyed_env", ac(), w(handlers.Platform{}.PlatformStatTodayDestroyedEnv))
	g.GET("/platform/stat/today/res_type", ac(), w(handlers.Platform{}.PlatformStatTodayResType))

	// 用户操作日志
	g.GET("/platform/operation/log", ac(), w(handlers.Platform{}.PlatformOperationLog))

	// 要求组织 header
	g.Use(w(middleware.AuthOrgId))

	// 策略管理
	ctrl.Register(g.Group("policies", ac()), &handlers.Policy{})
	g.GET("/policies/summary", ac(), w(handlers.Policy{}.PolicySummary))
	g.GET("/policies/:id/error", ac(), w(handlers.Policy{}.PolicyError))
	g.GET("/policies/:id/suppress", ac(), w(handlers.Policy{}.SearchPolicySuppress))
	g.POST("/policies/:id/suppress", ac("suppress"), w(handlers.Policy{}.UpdatePolicySuppress))
	g.GET("/policies/:id/suppress/sources", ac(), w(handlers.Policy{}.SearchPolicySuppressSource))
	g.DELETE("/policies/:id/suppress/:suppressId", ac("suppress"), w(handlers.Policy{}.DeletePolicySuppress))
	g.GET("/policies/:id/report", ac(), w(handlers.Policy{}.PolicyReport))
	g.POST("/policies/parse", ac(), w(handlers.Policy{}.Parse))
	g.POST("/policies/test", ac(), w(handlers.Policy{}.Test))

	g.GET("/policies/templates", ac(), w(handlers.Policy{}.SearchPolicyTpl))
	g.PUT("/policies/templates/:id", ac(), w(handlers.Policy{}.UpdatePolicyTpl))
	g.PUT("/policies/templates/:id/enabled", ac("enablescan"), w(handlers.Policy{}.EnablePolicyTpl))
	g.GET("/policies/templates/:id/policies", ac(), w(handlers.Policy{}.TplOfPolicy))
	g.GET("/policies/templates/:id/groups", ac(), w(handlers.Policy{}.TplOfPolicyGroup))
	g.GET("/policies/templates/:id/valid_policies", ac(), w(handlers.Policy{}.ValidTplOfPolicy))
	g.POST("/policies/templates/:id/scan", ac("scan"), w(handlers.Policy{}.ScanTemplate))
	g.POST("/policies/templates/scans", ac("scan"), w(handlers.Policy{}.ScanTemplates))
	g.GET("/policies/templates/:id/result", ac(), w(handlers.Policy{}.TemplateScanResult))

	g.GET("/policies/envs", ac(), w(handlers.Policy{}.SearchPolicyEnv))
	g.PUT("/policies/envs/:id", ac(), w(handlers.Policy{}.UpdatePolicyEnv))
	g.PUT("/policies/envs/:id/enabled", ac("enablescan"), w(handlers.Policy{}.EnablePolicyEnv))
	g.GET("/policies/envs/:id/policies", ac(), w(handlers.Policy{}.EnvOfPolicy))
	g.GET("/policies/envs/:id/valid_policies", ac(), w(handlers.Policy{}.ValidEnvOfPolicy))
	g.POST("/policies/envs/:id/scan", ac("scan"), w(handlers.Policy{}.ScanEnvironment))
	g.GET("/policies/envs/:id/result", ac(), w(handlers.Policy{}.EnvScanResult))

	ctrl.Register(g.Group("policies/groups", ac()), &handlers.PolicyGroup{})
	g.POST("/policies/groups/checks", ac(), w(handlers.PolicyGroupChecks))
	g.GET("/policies/groups/:id/policies", ac(), w(handlers.PolicyGroup{}.SearchGroupOfPolicy))
	g.POST("/policies/groups/:id", ac(), w(handlers.PolicyGroup{}.OpPolicyAndPolicyGroupRel))
	g.GET("/policies/groups/:id/report", ac(), w(handlers.PolicyGroup{}.ScanReport))
	g.GET("/policies/groups/:id/last_tasks", ac(), w(handlers.PolicyGroup{}.LastTasks))

	// 组织下的资源搜索(只需要有组织的读权限即可查看资源)
	g.GET("/orgs/resources", ac("orgs", "read"), w(handlers.Organization{}.SearchOrgResources))
	// 列出组织下资源搜索得到的相关项目名称以及provider名称
	g.GET("/orgs/resources/filters", ac("orgs", "read"), w(handlers.Organization{}.SearchOrgResourcesFilters))

	// 项目下的资源搜索(只需要有项目的读权限即可查看资源)
	g.GET("/projects/resources", ac("projects", "read"), w(handlers.Project{}.SearchProjectResources))
	// 列出项目下资源搜索得到的相关环境名称以及provider名称
	g.GET("/projects/resources/filters", ac("projects", "read"), w(handlers.Project{}.SearchProjectResourcesFilters))

	// 多云账号中心
	g.GET("/cloud/overview", ac("orgs", "read"), w(handlers.CloudOverview{}.Overview))
	ctrl.Register(g.Group("cloud/accounts", ac()), &handlers.CloudAccount{})
	g.POST("/cloud/accounts/health-check", ac("orgs", "read"), w(handlers.CloudAccount{}.HealthCheckAll))
	g.POST("/cloud/accounts/:id/validate", ac(), w(handlers.CloudAccount{}.Validate))
	g.POST("/cloud/accounts/:id/health-check", ac("orgs", "read"), w(handlers.CloudAccount{}.HealthCheck))
	g.GET("/cloud/accounts/:id/regions", ac("orgs", "read"), w(handlers.CloudAccount{}.Regions))
	g.PUT("/cloud/accounts/:id/regions", ac("orgs", "read"), w(handlers.CloudAccount{}.UpdateRegions))
	g.GET("/cloud/accounts/:id/permissions", ac("orgs", "read"), w(handlers.CloudAccount{}.Permissions))
	g.GET("/cloud/sync-policies", ac("orgs", "read"), w(handlers.CloudSyncPolicy{}.Search))
	g.POST("/cloud/sync-policies/run-due", ac("orgs", "read"), w(handlers.CloudSyncPolicy{}.RunDue))
	g.POST("/cloud/sync-policies", ac("orgs", "read"), w(handlers.CloudSyncPolicy{}.Create))
	g.GET("/cloud/sync-policies/:id", ac("orgs", "read"), w(handlers.CloudSyncPolicy{}.Detail))
	g.PUT("/cloud/sync-policies/:id", ac("orgs", "read"), w(handlers.CloudSyncPolicy{}.Update))
	g.DELETE("/cloud/sync-policies/:id", ac("orgs", "read"), w(handlers.CloudSyncPolicy{}.Delete))
	g.POST("/cloud/sync-policies/:id/run", ac("orgs", "read"), w(handlers.CloudSyncPolicy{}.Run))
	g.GET("/cloud/assets", ac("orgs", "read"), w(handlers.CloudAsset{}.SearchAssets))
	g.GET("/cloud/assets/filters", ac("orgs", "read"), w(handlers.CloudAsset{}.AssetFilters))
	g.GET("/cloud/assets/coverage", ac("orgs", "read"), w(handlers.CloudAsset{}.Coverage))
	g.GET("/cloud/assets/export", ac("orgs", "read"), w(handlers.CloudAsset{}.ExportAssets))
	g.POST("/cloud/assets/import", ac("orgs", "read"), w(handlers.CloudAsset{}.ImportAssets))
	g.PUT("/cloud/assets/ownership", ac("orgs", "read"), w(handlers.CloudAsset{}.BatchUpdateAssetOwnership))
	g.GET("/cloud/assets/:id/actions", ac("orgs", "read"), w(handlers.CloudAsset{}.AssetActions))
	g.POST("/cloud/assets/:id/actions/:action/dry-run", ac("orgs", "read"), w(handlers.CloudAsset{}.DryRunAssetAction))
	g.POST("/cloud/assets/:id/actions/:action", ac("orgs", "read"), w(handlers.CloudAsset{}.CreateAssetAction))
	g.GET("/cloud/assets/:id/security-rules", ac("orgs", "read"), w(handlers.CloudAsset{}.SecurityRules))
	g.GET("/cloud/assets/:id", ac("orgs", "read"), w(handlers.CloudAsset{}.AssetDetail))
	g.PUT("/cloud/assets/:id/ownership", ac("orgs", "read"), w(handlers.CloudAsset{}.UpdateAssetOwnership))
	g.POST("/cloud/backfill/iac-resources", ac("orgs", "read"), w(handlers.CloudAsset{}.BackfillIacResources))
	g.GET("/cloud/operations", ac("orgs", "read"), w(handlers.CloudOperation{}.Search))
	g.GET("/cloud/operations/:id", ac("orgs", "read"), w(handlers.CloudOperation{}.Detail))
	g.POST("/cloud/operations/:id/cancel", ac("orgs", "read"), w(handlers.CloudOperation{}.Cancel))
	g.POST("/cloud/operations/:id/retry", ac("orgs", "read"), w(handlers.CloudOperation{}.Retry))
	g.POST("/cloud/operations/:id/approve", ac("orgs", "read"), w(handlers.CloudOperation{}.Approve))
	g.POST("/cloud/operations/:id/itsm-ticket", ac("orgs", "read"), w(handlers.CloudItsm{}.CreateOperationTicket))
	g.GET("/cloud/operations/:id/audits", ac("orgs", "read"), w(handlers.CloudOperation{}.Audits))
	g.GET("/cloud/risks", ac("orgs", "read"), w(handlers.CloudRisk{}.Search))
	g.PUT("/cloud/risks/:id/status", ac("orgs", "read"), w(handlers.CloudRisk{}.UpdateStatus))
	g.POST("/cloud/risks/:id/suppress", ac("orgs", "read"), w(handlers.CloudRisk{}.Suppress))
	g.GET("/cloud/events", ac("orgs", "read"), w(handlers.CloudEvent{}.Search))
	g.GET("/cloud/cost/summary", ac("orgs", "read"), w(handlers.CloudCost{}.Summary))
	g.GET("/cloud/cost/trends", ac("orgs", "read"), w(handlers.CloudCost{}.Trends))
	g.GET("/cloud/cost/insights/summary", ac("orgs", "read"), w(handlers.CloudCost{}.InsightSummary))
	g.GET("/cloud/cost/insights", ac("orgs", "read"), w(handlers.CloudCost{}.Insights))
	g.PUT("/cloud/cost/insights/:id/status", ac("orgs", "read"), w(handlers.CloudCost{}.UpdateInsightStatus))
	g.GET("/cloud/cost/records", ac("orgs", "read"), w(handlers.CloudCost{}.Records))
	g.GET("/cloud/cost/sync-tasks", ac("orgs", "read"), w(handlers.CloudCost{}.SyncTasks))
	g.POST("/cloud/cost/sync-tasks", ac("orgs", "read"), w(handlers.CloudCost{}.CreateSyncTask))
	g.GET("/cloud/cost/sync-tasks/:id", ac("orgs", "read"), w(handlers.CloudCost{}.SyncTaskDetail))
	g.POST("/cloud/cost/sync-tasks/:id/retry", ac("orgs", "read"), w(handlers.CloudCost{}.RetrySyncTask))
	g.GET("/cloud/cost/sync-schedules", ac("orgs", "read"), w(handlers.CloudCost{}.SyncSchedules))
	g.POST("/cloud/cost/sync-schedules/run-due", ac("orgs", "read"), w(handlers.CloudCost{}.RunDueSyncSchedules))
	g.POST("/cloud/cost/sync-schedules", ac("orgs", "read"), w(handlers.CloudCost{}.CreateSyncSchedule))
	g.PUT("/cloud/cost/sync-schedules/:id", ac("orgs", "read"), w(handlers.CloudCost{}.UpdateSyncSchedule))
	g.DELETE("/cloud/cost/sync-schedules/:id", ac("orgs", "read"), w(handlers.CloudCost{}.DeleteSyncSchedule))
	g.POST("/cloud/cost/sync-schedules/:id/run", ac("orgs", "read"), w(handlers.CloudCost{}.RunSyncSchedule))
	g.POST("/cloud/cost/import", ac("orgs", "read"), w(handlers.CloudCost{}.Import))
	g.POST("/cloud/cost/pull", ac("orgs", "read"), w(handlers.CloudCost{}.Pull))
	g.GET("/cloud/cost/unmatched", ac("orgs", "read"), w(handlers.CloudCost{}.Unmatched))
	g.GET("/cloud/budgets/summary", ac("orgs", "read"), w(handlers.CloudBudget{}.Summary))
	g.GET("/cloud/budgets", ac("orgs", "read"), w(handlers.CloudBudget{}.Search))
	g.POST("/cloud/budgets/evaluate-due", ac("orgs", "read"), w(handlers.CloudBudget{}.EvaluateDue))
	g.POST("/cloud/budgets", ac("orgs", "read"), w(handlers.CloudBudget{}.Create))
	g.PUT("/cloud/budgets/:id", ac("orgs", "read"), w(handlers.CloudBudget{}.Update))
	g.DELETE("/cloud/budgets/:id", ac("orgs", "read"), w(handlers.CloudBudget{}.Delete))
	g.GET("/cloud/webhooks", ac("orgs", "read"), w(handlers.CloudWebhook{}.Search))
	g.POST("/cloud/webhooks", ac("orgs", "read"), w(handlers.CloudWebhook{}.Create))
	g.POST("/cloud/webhooks/retry-due", ac("orgs", "read"), w(handlers.CloudWebhook{}.RetryDue))
	g.GET("/cloud/webhooks/queue/summary", ac("orgs", "read"), w(handlers.CloudWebhook{}.QueueSummary))
	g.POST("/cloud/webhooks/deliveries/:deliveryId/signature-verification", ac("orgs", "read"), w(handlers.CloudWebhook{}.ReportSignatureVerification))
	g.GET("/cloud/webhooks/dead-letters", ac("orgs", "read"), w(handlers.CloudWebhook{}.DeadLetters))
	g.POST("/cloud/webhooks/dead-letters/:deadLetterId/replay", ac("orgs", "read"), w(handlers.CloudWebhook{}.ReplayDeadLetter))
	g.POST("/cloud/webhooks/dead-letters/:deadLetterId/ignore", ac("orgs", "read"), w(handlers.CloudWebhook{}.IgnoreDeadLetter))
	g.PUT("/cloud/webhooks/:id", ac("orgs", "read"), w(handlers.CloudWebhook{}.Update))
	g.DELETE("/cloud/webhooks/:id", ac("orgs", "read"), w(handlers.CloudWebhook{}.Delete))
	g.POST("/cloud/webhooks/:id/secret/rotate", ac("orgs", "read"), w(handlers.CloudWebhook{}.RotateSecret))
	g.POST("/cloud/webhooks/:id/test", ac("orgs", "read"), w(handlers.CloudWebhook{}.Test))
	g.GET("/cloud/webhooks/:id/deliveries", ac("orgs", "read"), w(handlers.CloudWebhook{}.Deliveries))
	g.GET("/cloud/webhooks/:id/deliveries/:deliveryId", ac("orgs", "read"), w(handlers.CloudWebhook{}.DeliveryDetail))
	g.POST("/cloud/webhooks/:id/deliveries/:deliveryId/retry", ac("orgs", "read"), w(handlers.CloudWebhook{}.RetryDelivery))
	g.GET("/cloud/itsm/configs", ac("orgs", "read"), w(handlers.CloudItsm{}.SearchConfigs))
	g.POST("/cloud/itsm/configs", ac("orgs", "read"), w(handlers.CloudItsm{}.CreateConfig))
	g.PUT("/cloud/itsm/configs/:id", ac("orgs", "read"), w(handlers.CloudItsm{}.UpdateConfig))
	g.DELETE("/cloud/itsm/configs/:id", ac("orgs", "read"), w(handlers.CloudItsm{}.DeleteConfig))
	g.GET("/cloud/itsm/overview", ac("orgs", "read"), w(handlers.CloudItsm{}.Overview))
	g.POST("/cloud/itsm/self-service-tickets", ac("orgs", "read"), w(handlers.CloudItsm{}.CreateSelfServiceTicket))
	g.GET("/cloud/itsm/tickets", ac("orgs", "read"), w(handlers.CloudItsm{}.SearchTickets))
	g.PUT("/cloud/itsm/tickets/:id/status", ac("orgs", "read"), w(handlers.CloudItsm{}.UpdateTicketStatus))
	g.GET("/cloud/sync-tasks", ac("orgs", "read"), w(handlers.CloudAsset{}.SearchSyncTasks))
	g.GET("/cloud/sync-task-rerun-groups/:groupId", ac("orgs", "read"), w(handlers.CloudAsset{}.SyncTaskRerunGroupDetail))
	g.POST("/cloud/sync-task-rerun-groups/:groupId/approve", ac("orgs", "read"), w(handlers.CloudAsset{}.ApproveSyncTaskRerunGroup))
	g.GET("/cloud/sync-tasks/:id", ac("orgs", "read"), w(handlers.CloudAsset{}.SyncTaskDetail))
	g.POST("/cloud/sync-tasks/rerun-failed", ac("orgs", "read"), w(handlers.CloudAsset{}.BatchRerunFailedSyncTasks))
	g.POST("/cloud/sync-tasks", ac("orgs", "read"), w(handlers.CloudAsset{}.StartSyncTask))

	// CMDB 资产中心
	g.GET("/cmdb/assets", ac("orgs", "read"), w(handlers.Cmdb{}.SearchAssets))
	g.GET("/cmdb/assets/filters", ac("orgs", "read"), w(handlers.Cmdb{}.AssetFilters))
	g.GET("/cmdb/assets/export", ac("orgs", "read"), w(handlers.Cmdb{}.ExportAssets))
	g.POST("/cmdb/assets/import", ac("orgs", "read"), w(handlers.Cmdb{}.ImportAssets))
	g.GET("/cmdb/applications", ac("orgs", "read"), w(handlers.Cmdb{}.SearchApplications))
	g.GET("/cmdb/applications/detail", ac("orgs", "read"), w(handlers.Cmdb{}.ApplicationDetail))
	g.PUT("/cmdb/applications/relations", ac("orgs", "read"), w(handlers.Cmdb{}.UpdateApplicationRelations))
	g.PUT("/cmdb/assets/ownership", ac("orgs", "read"), w(handlers.Cmdb{}.BatchUpdateAssetOwnership))
	g.GET("/cmdb/assets/:id", ac("orgs", "read"), w(handlers.Cmdb{}.AssetDetail))
	g.PUT("/cmdb/assets/:id/ownership", ac("orgs", "read"), w(handlers.Cmdb{}.UpdateAssetOwnership))
	g.POST("/cmdb/backfill/iac-resources", ac("orgs", "read"), w(handlers.Cmdb{}.BackfillIacResources))
	g.GET("/cmdb/cloud/accounts", ac("orgs", "read"), w(handlers.Cmdb{}.CloudAccounts))
	g.GET("/cmdb/sync-tasks", ac("orgs", "read"), w(handlers.Cmdb{}.SearchSyncTasks))
	g.GET("/cmdb/sync-task-rerun-groups/:groupId", ac("orgs", "read"), w(handlers.Cmdb{}.SyncTaskRerunGroupDetail))
	g.POST("/cmdb/sync-task-rerun-groups/:groupId/approve", ac("orgs", "read"), w(handlers.Cmdb{}.ApproveSyncTaskRerunGroup))
	g.GET("/cmdb/sync-tasks/:id", ac("orgs", "read"), w(handlers.Cmdb{}.SyncTaskDetail))
	g.POST("/cmdb/sync-tasks/rerun-failed", ac("orgs", "read"), w(handlers.Cmdb{}.BatchRerunFailedSyncTasks))
	g.POST("/cmdb/sync-tasks", ac("orgs", "read"), w(handlers.Cmdb{}.StartSyncTask))

	// 组织概览统计数据
	g.GET("/orgs/projects/statistics", ac(), w(handlers.Organization{}.OrgProjectsStat))

	// 组织用户管理
	g.GET("/orgs/:id/users", ac("orgs", "listuser"), w(handlers.Organization{}.SearchUser))
	g.POST("/orgs/:id/users", ac("orgs", "adduser"), w(handlers.Organization{}.AddUserToOrg))
	g.PUT("/orgs/:id/users/:userId/role", ac("orgs", "updaterole"), w(handlers.Organization{}.UpdateUserOrgRel))
	g.PUT("/orgs/:id/users/:userId", ac("orgs", "updaterole"), w(handlers.Organization{}.UpdateUserOrg))
	g.POST("/orgs/:id/users/invite", ac("orgs", "adduser"), w(handlers.Organization{}.InviteUser))
	g.POST("/orgs/:id/users/batch_invite", ac("orgs", "adduser"), w(handlers.Organization{}.InviteUsersBatch))
	g.DELETE("/orgs/:id/users/:userId", ac("orgs", "removeuser"), w(handlers.Organization{}.RemoveUserForOrg))

	// orgs ldap 相关
	g.GET("/ldap/org_ous", ac(), w(handlers.GetLdapOUsFromDB))
	g.DELETE("/ldap/org_ou", ac(), w(handlers.DeleteLdapOUFromDB))
	g.PUT("/ldap/org_ou", ac(), w(handlers.UpdateLdapOU))
	g.GET("/ldap/ous", ac(), w(handlers.GetLdapOUsFromLdap))
	g.GET("/ldap/users", ac(), w(handlers.GetLdapUsers))
	g.POST("/ldap/auth/org_user", ac(), w(handlers.AuthLdapUser))
	g.POST("/ldap/auth/org_ou", ac(), w(handlers.AuthLdapOU))

	g.GET("/projects/users", ac(), w(handlers.ProjectUser{}.Search))
	g.GET("/projects/authorization/users", ac(), w(handlers.ProjectUser{}.SearchProjectAuthorizationUser))
	g.POST("/projects/users", ac(), w(handlers.ProjectUser{}.Create))
	g.PUT("/projects/users/:id", ac(), w(handlers.ProjectUser{}.Update))
	g.DELETE("/projects/users/:id", ac(), w(handlers.ProjectUser{}.Delete))

	//项目管理
	ctrl.Register(g.Group("projects", ac()), &handlers.Project{})

	// 项目概览统计数据
	g.GET("/projects/:id/statistics", ac(), w(handlers.Project{}.ProjectStat))

	//变量管理
	g.PUT("/variables/batch", ac(), w(handlers.Variable{}.BatchUpdate))
	g.PUT("/variables/scope/:scope/:id", ac(), w(handlers.Variable{}.UpdateObjectVars))
	// 供第三方系统获取变量的接口，该接口将 terraform 变量和环境变量统一转为环境变量格式返回，方便第三方系统处理
	g.GET("/variables/sample", ac(), w(handlers.Variable{}.SearchSampleVariable))
	ctrl.Register(g.Group("variables", ac()), &handlers.Variable{})

	// 变量组
	ctrl.Register(g.Group("var_groups", ac()), &handlers.VariableGroup{})
	g.GET("/var_groups/relationship", ac(), w(handlers.VariableGroup{}.SearchRelationship))
	g.GET("/var_groups/relationship/all", ac(), w(handlers.VariableGroup{}.SearchRelationshipAll))
	g.PUT("/var_groups/relationship/batch", ac(), w(handlers.VariableGroup{}.BatchUpdateRelationship))
	//g.DELETE("/var_groups/relationship/:id", ac(), w(handlers.VariableGroup{}.DeleteRelationship))

	//token管理
	ctrl.Register(g.Group("tokens", ac()), &handlers.Token{})
	//密钥管理
	ctrl.Register(g.Group("keys", ac()), &handlers.Key{})

	ctrl.Register(g.Group("vcs", ac()), &handlers.Vcs{})
	g.GET("/vcs/registry", ac(), w(handlers.Vcs{}.GetRegistryVcs))
	g.GET("/vcs/:id/repo", ac(), w(handlers.Vcs{}.ListRepos))
	g.GET("/vcs/:id/branch", ac(), w(handlers.Vcs{}.ListBranches))
	g.GET("/vcs/:id/tag", ac(), w(handlers.Vcs{}.ListTags))
	g.GET("/vcs/:id/readme", ac(), w(handlers.Vcs{}.GetReadmeContent))

	g.GET("/registry/policy_groups", w(handlers.SearchRegistryPG))
	g.GET("/registry/policy_groups/versions", w(handlers.SearchRegistryPGVersions))

	// 云模板
	ctrl.Register(g.Group("templates", ac()), &handlers.Template{})
	g.GET("/vcs/:id/repos/variables", ac(), w(handlers.TemplateVariableSearch))
	g.GET("/templates/tfversions", ac(), w(handlers.TemplateTfVersionSearch))
	g.GET("/templates/autotfversion", ac(), w(handlers.AutoTemplateTfVersionChoice))
	g.POST("/templates/checks", ac(), w(handlers.TemplateChecks))
	g.GET("/templates/export", ac(), w(handlers.TemplateExport))
	g.POST("/templates/import", ac(), w(handlers.TemplateImport))
	g.GET("/vcs/:id/repos/tfvars", ac(), w(handlers.TemplateTfvarsSearch))
	g.GET("/vcs/:id/repos/playbook", ac(), w(handlers.TemplatePlaybookSearch))
	g.GET("/vcs/:id/repos/url", ac(), w(handlers.Vcs{}.GetFileFullPath))
	g.GET("/vcs/:id/file", ac(), w(handlers.Vcs{}.GetVcsRepoFileContent))
	g.GET("/notification-templates", ac(), w(handlers.Notification{}.SearchTemplates))
	g.POST("/notification-templates", ac(), w(handlers.Notification{}.CreateTemplate))
	g.GET("/notification-templates/variables", ac(), w(handlers.Notification{}.TemplateVariables))
	g.POST("/notification-templates/preview", ac(), w(handlers.Notification{}.PreviewTemplate))
	g.POST("/notification-templates/copy-to-project", ac(), w(handlers.Notification{}.CopyTemplatesToProject))
	g.GET("/notification-templates/:id/versions", ac(), w(handlers.Notification{}.TemplateVersions))
	g.POST("/notification-templates/:id/versions/:versionId/rollback", ac(), w(handlers.Notification{}.RollbackTemplateVersion))
	g.GET("/notification-templates/:id", ac(), w(handlers.Notification{}.DetailTemplate))
	g.PUT("/notification-templates/:id", ac(), w(handlers.Notification{}.UpdateTemplate))
	g.DELETE("/notification-templates/:id", ac(), w(handlers.Notification{}.DeleteTemplate))
	g.GET("/notifications/:id/deliveries", ac(), w(handlers.Notification{}.Deliveries))
	ctrl.Register(g.Group("notifications", ac()), &handlers.Notification{})

	// 任务实时日志（云模板检测无项目ID）
	g.GET("/tasks/:id/log/sse", ac(), w(handlers.Task{}.FollowLogSse))

	// 项目资源
	g.Use(w(middleware.AuthProjectId))

	// projects ldap 相关
	g.GET("/ldap/project_ous", ac(), w(handlers.GetProjectLdapOUs))
	g.DELETE("/ldap/project_ou", ac(), w(handlers.DeleteProjectLdapOU))
	g.PUT("/ldap/project_ou", ac(), w(handlers.UpdateProjectLdapOU))
	g.POST("/ldap/auth/project_ou", ac(), w(handlers.AuthProjectLdapOU))

	// 环境管理
	ctrl.Register(g.Group("envs", ac()), &handlers.Env{})
	g.PUT("/envs/:id/archive", ac(), w(handlers.Env{}.Archive))
	g.GET("/envs/:id/tasks", ac(), w(handlers.Env{}.SearchTasks))
	g.GET("/envs/:id/tasks/last", ac(), w(handlers.Env{}.LastTask))
	g.POST("/envs/:id/deploy", ac("envs", "deploy"), w(handlers.Env{}.Deploy))
	g.POST("/envs/:id/deploy/check", ac("envs", "deploy"), w(handlers.Env{}.DeployCheck))
	g.POST("/envs/:id/destroy", ac("envs", "destroy"), w(handlers.Env{}.Destroy))
	g.POST("/envs/:id/tags", ac("envs", "tags"), w(handlers.Env{}.UpdateTags))
	g.GET("/envs/:id/resources", ac(), w(handlers.Env{}.SearchResources))
	g.GET("/envs/:id/output", ac(), w(handlers.Env{}.Output))
	g.GET("/envs/:id/resources/:resourceId", ac(), w(handlers.Env{}.ResourceDetail))
	g.GET("/envs/:id/variables", ac(), w(handlers.Env{}.Variables))
	g.GET("/envs/:id/policy_result", ac(), w(handlers.Env{}.PolicyResult))
	g.GET("/envs/:id/resources/graph", ac(), w(handlers.Env{}.SearchResourcesGraph))
	g.GET("/envs/:id/resources/graph/:resourceId", ac(), w(handlers.Env{}.ResourceGraphDetail))
	g.POST("/envs/:id/lock", ac("envs", "lock"), w(handlers.EnvLock))
	g.POST("/envs/:id/unlock", ac("envs", "unlock"), w(handlers.EnvUnLock))
	g.GET("/envs/:id/unlock/confirm", ac(), w(handlers.EnvUnLockConfirm))

	// 环境概览统计数据
	g.GET("/envs/:id/statistics", ac(), w(handlers.Env{}.EnvStat))

	// 任务管理
	g.GET("/tasks", ac(), w(handlers.Task{}.Search))
	g.GET("/tasks/:id", ac(), w(handlers.Task{}.Detail))
	g.GET("/tasks/:id/log", ac(), w(handlers.Task{}.Log))
	g.GET("/tasks/:id/output", ac(), w(handlers.Task{}.Output))
	g.GET("/tasks/:id/resources", ac(), w(handlers.Task{}.Resource))
	g.POST("/tasks/:id/abort", ac("tasks", "abort"), w(handlers.Task{}.TaskAbort))
	g.POST("/tasks/:id/approve", ac("tasks", "approve"), w(handlers.Task{}.TaskApprove))
	g.POST("/tasks/:id/comment", ac(), w(handlers.TaskComment{}.Create))
	g.GET("/tasks/:id/comment", ac(), w(handlers.TaskComment{}.Search))
	g.GET("/tasks/:id/steps", ac(), w(handlers.Task{}.SearchTaskStep))
	g.GET("/tasks/:id/steps/:stepId/log", ac(), w(handlers.Task{}.GetTaskStepLog))
	g.GET("/tasks/:id/steps/:stepId/log/sse", ac(), w(handlers.Task{}.FollowStepLogSse))
	g.GET("/tasks/:id/resources/graph", ac(), w(handlers.Task{}.ResourceGraph))

	//g.GET("/tokens/trigger", ac(), w(handlers.Token{}.VcsWebhookUrl))
	g.GET("/vcs/webhook", ac(), w(handlers.Token{}.VcsWebhookUrl))
	ctrl.Register(g.Group("resource/account", ac()), &handlers.ResourceAccount{})
}
