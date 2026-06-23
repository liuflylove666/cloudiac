import React from 'react';
import loadable from 'utils/loadable';

const asyncLoadFallback = {
  fallback: 'Loading......'
};

const routeLoadable = importFunc => loadable(importFunc, asyncLoadFallback);

export default function createRoutes() {
  return [
    {
      path: '/',
      name: '组织',
      component: routeLoadable(() => import(/* webpackChunkName: "route-org-select" */ 'containers/org-select-page')),
      exact: true
    },
    {
      path: '/org/:orgId/compliance/:configKey?/:typeKey',
      name: '合规配置',
      component: routeLoadable(() => import(/* webpackChunkName: "route-compliance-shell" */ 'containers/compliance')),
      routes: [
        {
          path: '/org/:orgId/compliance/dashboard',
          name: '仪表盘',
          component: routeLoadable(() => import(/* webpackChunkName: "route-compliance-dashboard" */ 'containers/compliance/dashboard')),
          exact: true
        },
        {
          path: '/org/:orgId/compliance/compliance-config/ct',
          name: '云模板',
          component: routeLoadable(() => import(/* webpackChunkName: "route-compliance-ct" */ 'containers/compliance/compliance-config/ct')),
          exact: true
        },
        {
          path: '/org/:orgId/compliance/compliance-config/env',
          name: '环境',
          component: routeLoadable(() => import(/* webpackChunkName: "route-compliance-env" */ 'containers/compliance/compliance-config/env')),
          exact: true
        },
        {
          path: '/org/:orgId/compliance/policy-config/policy-group',
          name: '策略组',
          component: routeLoadable(() => import(/* webpackChunkName: "route-compliance-policy-group" */ 'containers/compliance/policy-config/policy-group')),
          exact: true
        },
        {
          path: '/org/:orgId/compliance/policy-config/policy-group/policy-group-form/:policyGroupId?',
          name: '策略组创建/编辑页',
          component: routeLoadable(() => import(/* webpackChunkName: "route-compliance-policy-group-form" */ 'containers/compliance/policy-config/policy-group/form-page')),
          exact: true
        },
        {
          path: '/org/:orgId/compliance/policy-config/policy',
          name: '策略',
          component: routeLoadable(() => import(/* webpackChunkName: "route-compliance-policy" */ 'containers/compliance/policy-config/policy')),
          exact: true
        },
        {
          path: '/org/:orgId/compliance/policy-config/policy/online-test/:policyId?',
          name: '在线测试',
          component: routeLoadable(() => import(/* webpackChunkName: "route-compliance-policy-online-test" */ 'containers/compliance/policy-config/policy/online-test')),
          exact: true
        }
      ]
    },
    {
      path: '/org/:orgId/project/:projectId/:mProjectKey',
      name: '组织主页',
      component: routeLoadable(() => import(/* webpackChunkName: "route-org-shell" */ 'containers/org')),
      routes: [
        {
          path: '/org/:orgId/project/:projectId/m-project-env',
          name: '项目信息：环境',
          component: routeLoadable(() => import(/* webpackChunkName: "route-project-env" */ 'containers/org/m-project/env')),
          exact: true
        },
        {
          path: '/org/:orgId/project/:projectId/m-project-env/deploy/:tplId/:envId?',
          name: '部署新环境：选择云模板',
          component: routeLoadable(() => import(/* webpackChunkName: "route-project-env-deploy" */ 'containers/org/m-project/env/deploy')),
          exact: true
        },
        {
          path: '/org/:orgId/project/:projectId/m-project-env/detail/:envId',
          //(resource,output,deploy,deployHistory,variable,setting)
          component: routeLoadable(() => import(/* webpackChunkName: "route-project-env-detail" */ 'containers/org/m-project/env/detail')),
          exact: true
        },
        {
          path: '/org/:orgId/project/:projectId/m-project-env/detail/:envId/task/:taskId',
          component: routeLoadable(() => import(/* webpackChunkName: "route-project-env-task" */ 'containers/org/m-project/env/detail/task-detail')),
          exact: true
        },
        {
          path: '/org/:orgId/project/:projectId/m-project-ct',
          name: '项目信息：云模板',
          component: routeLoadable(() => import(/* webpackChunkName: "route-project-ct" */ 'containers/org/m-project/ct')),
          exact: true
        },
        {
          path: '/org/:orgId/project/:projectId/m-project-variable',
          name: '项目信息：变量',
          component: routeLoadable(() => import(/* webpackChunkName: "route-project-variable" */ 'containers/org/m-project/variable')),
          exact: true
        },
        {
          path: '/org/:orgId/project/:projectId/m-project-setting',
          name: '项目信息：设置',
          component: routeLoadable(() => import(/* webpackChunkName: "route-project-setting" */ 'containers/org/m-project/setting')),
          exact: true
        }
      ]
    },
    {
      path: '/org/:orgId/:mOrgKey',
      name: '组织主页',
      component: routeLoadable(() => import(/* webpackChunkName: "route-org-shell" */ 'containers/org')),
      routes: [
        {
          path: '/org/:orgId/m-cloud-overview',
          name: '多云总览',
          component: routeLoadable(() => import(/* webpackChunkName: "route-cloud-overview" */ 'containers/org/cloud-overview')),
          exact: true
        },
        {
          path: '/org/:orgId/m-cloud-account',
          name: '云账号',
          component: routeLoadable(() => import(/* webpackChunkName: "route-cloud-account" */ 'containers/org/cloud-account')),
          exact: true
        },
        {
          path: '/org/:orgId/m-cloud-assets',
          name: '云资产',
          component: routeLoadable(() => import(/* webpackChunkName: "route-cloud-assets" */ 'containers/org/resource-query')),
          exact: true
        },
        {
          path: '/org/:orgId/m-cloud-operations',
          name: '操作任务',
          component: routeLoadable(() => import(/* webpackChunkName: "route-cloud-operations" */ 'containers/org/cloud-operation')),
          exact: true
        },
        {
          path: '/org/:orgId/m-cloud-risks',
          name: '风险合规',
          component: routeLoadable(() => import(/* webpackChunkName: "route-cloud-risks" */ 'containers/org/cloud-risk')),
          exact: true
        },
        {
          path: '/org/:orgId/m-cloud-costs',
          name: '成本中心',
          component: routeLoadable(() => import(/* webpackChunkName: "route-cloud-costs" */ 'containers/org/cloud-cost')),
          exact: true
        },
        {
          path: '/org/:orgId/m-cloud-events',
          name: '事件中心',
          component: routeLoadable(() => import(/* webpackChunkName: "route-cloud-events" */ 'containers/org/cloud-event')),
          exact: true
        },
        {
          path: '/org/:orgId/m-cloud-itsm',
          name: 'ITSM 工单',
          component: routeLoadable(() => import(/* webpackChunkName: "route-cloud-itsm" */ 'containers/org/cloud-itsm')),
          exact: true
        },
        {
          path: '/org/:orgId/m-other-resource',
          name: '资产 CMDB',
          component: routeLoadable(() => import(/* webpackChunkName: "route-cloud-assets" */ 'containers/org/resource-query')),
          exact: true
        },
        {
          path: '/org/:orgId/m-org-project',
          name: '组织设置：项目',
          component: routeLoadable(() => import(/* webpackChunkName: "route-org-project" */ 'containers/org/m-org/project')),
          exact: true
        },
        {
          path: '/org/:orgId/m-org-ct',
          name: '组织设置：云模板',
          component: routeLoadable(() => import(/* webpackChunkName: "route-org-ct" */ 'containers/org/m-org/ct')),
          exact: true
        },
        {
          path: '/org/:orgId/m-org-ct/createCT',
          name: '新建云模板',
          component: routeLoadable(() => import(/* webpackChunkName: "route-org-ct-create" */ 'containers/org/m-org/ct/create')),
          exact: true
        },
        {
          path: '/org/:orgId/m-org-ct/updateCT/:tplId',
          name: '编辑云模板',
          component: routeLoadable(() => import(/* webpackChunkName: "route-org-ct-update" */ 'containers/org/m-org/ct/update')),
          exact: true
        },
        {
          path: '/org/:orgId/m-org-variable',
          name: '组织设置：变量',
          component: routeLoadable(() => import(/* webpackChunkName: "route-org-variable" */ 'containers/org/m-org/variable')),
          exact: true
        },
        {
          path: '/org/:orgId/m-org-setting',
          name: '组织设置：设定',
          component: routeLoadable(() => import(/* webpackChunkName: "route-org-setting" */ 'containers/org/m-org/setting')),
          exact: true
        },
        {
          path: '/org/:orgId/m-project-create',
          name: '项目信息：创建项目',
          component: routeLoadable(() => import(/* webpackChunkName: "route-project-create" */ 'containers/org/m-project/create')),
          exact: true
        }
      ]
    },
    {
      path: '/project-select-page',
      name: '全部项目',
      component: routeLoadable(() => import(/* webpackChunkName: "route-project-select" */ 'containers/project-select-page')),
      exact: true
    },
    {
      path: '/sys/setting',
      name: '系统设置',
      component: routeLoadable(() => import(/* webpackChunkName: "route-sys-setting" */ 'containers/sys')),
      exact: true
    },
    {
      path: '/sys/status',
      name: '系统状态',
      component: routeLoadable(() => import(/* webpackChunkName: "route-sys-status" */ 'containers/sys/status')),
      exact: true
    },
    {
      path: '/user/setting',
      name: '用户设置',
      component: routeLoadable(() => import(/* webpackChunkName: "route-user-setting" */ 'containers/user')),
      exact: true
    },
    {
      path: '/devManual',
      name: '帮助文档',
      component: routeLoadable(() => import(/* webpackChunkName: "route-dev-manual" */ 'containers/devManual')),
      exact: true
    },
    {
      path: '/no-access',
      name: 'NoAccessPage',
      component: routeLoadable(() => import(/* webpackChunkName: "route-no-access" */ 'containers/no-access'))
    },
    {
      path: '*',
      name: 'NotFoundPage',
      component: routeLoadable(() => import(/* webpackChunkName: "route-not-found" */ 'containers/NotFoundPage'))
    }
  ];
}
