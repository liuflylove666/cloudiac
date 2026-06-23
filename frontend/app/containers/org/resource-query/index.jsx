import React, { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import moment from 'moment';
import queryString from 'query-string';
import {
  Alert,
  Button,
  Checkbox,
  Collapse,
  ConfigProvider,
  DatePicker,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  notification,
  Progress,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
  Upload
} from 'antd';
import { DownloadOutlined, ReloadOutlined, SaveOutlined, SearchOutlined, SettingOutlined, SyncOutlined, UploadOutlined } from '@ant-design/icons';
import { useRequest } from 'ahooks';
import { requestWrapper } from 'utils/request';
import { useSearchFormAndTable } from 'utils/hooks';
import { downloadImportTemplate } from 'utils/util';
import copy from 'utils/copy';
import PageHeader from 'components/pageHeader';
import Layout from 'components/common/layout';
import EllipsisText from 'components/EllipsisText';
import cmdbAPI from 'services/cmdb';
import cloudAssetAPI from 'services/cloud-asset';
import styles from './styles.less';

const { Option } = Select;
const { Search: InputSearch } = Input;
const { TabPane } = Tabs;
const { Panel: CollapsePanel } = Collapse;
const { Text } = Typography;
const { RangePicker } = DatePicker;

const assetTypeMap = {
  compute_instance: '计算实例',
  kubernetes_cluster: 'Kubernetes集群',
  kubernetes_namespace: 'K8S命名空间',
  kubernetes_node: 'K8S节点',
  kubernetes_workload: 'K8S工作负载',
  kubernetes_pod: 'K8S Pod',
  kubernetes_service: 'K8S Service',
  kubernetes_ingress: 'K8S Ingress',
  network_vpc: 'VPC/VCN',
  network_subnet: '子网',
  network_route_table: '路由表',
  network_nat_gateway: 'NAT网关',
  network_internet_gateway: 'Internet网关',
  network_service_gateway: 'Service网关',
  network_drg: 'DRG',
  network_security_group: '安全组',
  public_ip: '公网IP',
  load_balancer: '负载均衡',
  block_volume: '块存储',
  object_storage_bucket: '对象存储',
  relational_database: '关系型数据库',
  redis_cache: 'Redis缓存',
  unknown: '未分类'
};

const kubernetesAssetTypes = [
  'kubernetes_cluster',
  'kubernetes_namespace',
  'kubernetes_node',
  'kubernetes_workload',
  'kubernetes_pod',
  'kubernetes_service',
  'kubernetes_ingress'
];

const kubernetesProviderQuickFilters = [
  { provider: 'aws', label: 'EKS' },
  { provider: 'oci', label: 'OKE' },
  { provider: 'azure', label: 'AKS' },
  { provider: 'gcp', label: 'GKE' }
];

const sourceMap = {
  iac_resource: 'IaC资源',
  cloud_collect: '云采集',
  manual_edit: '人工维护',
  manual_application: '人工应用关系',
  import: '导入'
};

const importActionMap = {
  create: '新增',
  update: '更新',
  ownership_update: '归属更新',
  skip: '跳过',
  error: '异常'
};

const importActionColorMap = {
  create: 'success',
  update: 'processing',
  ownership_update: 'warning',
  skip: 'default',
  error: 'error'
};

const managedByMap = {
  iac: 'IaC纳管',
  cloud_linked: '云采集已关联',
  cloud_only: '云上未纳管',
  manual: '人工维护'
};

const relationTypeMap = {
  depends_on: '依赖',
  contains: '包含'
};

const relationDirectionMap = {
  all: '全部方向',
  incoming: '上游',
  outgoing: '下游'
};

const changeTypeMap = {
  created: '新增',
  updated: '更新'
};

const changeTypeColorMap = {
  created: 'success',
  updated: 'processing'
};

const accountSourceMap = {
  variable_group: '变量组',
  resource_account: '资源账号',
  cloud_account: '云账号'
};

const taskStatusMap = {
  pending: '等待中',
  approving: '待审批',
  running: '运行中',
  complete: '完成',
  failed: '失败',
  rejected: '已驳回'
};

const taskStatusColorMap = {
  pending: 'default',
  approving: 'warning',
  running: 'processing',
  complete: 'success',
  failed: 'error',
  rejected: 'default'
};

const syncTaskRerunModeTextMap = {
  batch_failed: '失败任务批量重跑'
};

const defaultSyncTaskApiMetricFilters = {
  regions: [],
  services: [],
  statuses: [],
  slowOnly: false,
  slowThresholdMs: 1000
};

const syncTaskSlowApiThresholdMs = (task = {}) => {
  const value = Number(((task.stats || {}).slowApiThresholdMs) || defaultSyncTaskApiMetricFilters.slowThresholdMs);
  return Math.max(1, Math.min(600000, value || defaultSyncTaskApiMetricFilters.slowThresholdMs));
};

const syncLogLevelMap = {
  info: '信息',
  warn: '警告',
  error: '错误'
};

const syncLogLevelColorMap = {
  info: 'processing',
  warn: 'warning',
  error: 'error'
};

const syncTaskMetricStatusMap = {
  complete: '完成',
  partial_failed: '部分失败',
  failed: '失败'
};

const syncTaskMetricStatusColorMap = {
  complete: 'success',
  partial_failed: 'warning',
  failed: 'error'
};

const syncTaskFailureCategoryMap = {
  rate_limit: '限流',
  network: '网络',
  permission: '权限',
  credential: '凭证',
  configuration: '配置',
  unknown: '未知'
};

const syncTaskFailureCategoryColorMap = {
  rate_limit: 'warning',
  network: 'warning',
  permission: 'error',
  credential: 'error',
  configuration: 'warning',
  unknown: 'default'
};

const lifecycleMap = {
  planned: '规划',
  active: '运行',
  maintenance: '维护',
  retired: '下线'
};

const complianceRiskMap = {
  low: '低',
  medium: '中',
  high: '高',
  critical: '严重'
};

const complianceRiskColorMap = {
  low: 'success',
  medium: 'warning',
  high: 'error',
  critical: 'magenta'
};

const applicationRiskMap = {
  low: '低',
  medium: '中',
  high: '高',
  critical: '严重'
};

const applicationRiskColorMap = {
  low: 'success',
  medium: 'warning',
  high: 'error',
  critical: 'magenta'
};

const actionRiskMap = {
  low: '低',
  medium: '中',
  high: '高',
  critical: '严重'
};

const actionRiskColorMap = {
  low: 'success',
  medium: 'warning',
  high: 'error',
  critical: 'magenta'
};

const actionCheckStatusMap = {
  pass: '通过',
  warn: '提醒',
  fail: '失败'
};

const actionCheckStatusColorMap = {
  pass: 'success',
  warn: 'warning',
  fail: 'error'
};

const actionAdapterModeMap = {
  local: '本地安全',
  provider: '云端适配器'
};

const actionAdapterModeColorMap = {
  local: 'processing',
  provider: 'warning'
};

const actionAdapterStatusMap = {
  ready: '就绪',
  registered: '已注册',
  blocked: '受保护',
  unsupported: '不支持'
};

const actionAdapterStatusColorMap = {
  ready: 'success',
  registered: 'processing',
  blocked: 'warning',
  unsupported: 'default'
};

const asCSV = (value) => Array.isArray(value) ? value.join(',') : value;
const firstQueryValue = (value) => Array.isArray(value) ? value[0] : value;
const accountKey = (account) => `${account.source}:${account.id}`;
const hasValue = (value) => value !== undefined && value !== null && value !== '';
const joinList = (value) => {
  const items = Array.isArray(value) ? value : (hasValue(value) ? [value] : []);
  if (!items.length) {
    return '-';
  }
  return items.map((item) => typeof item === 'object' ? JSON.stringify(item) : String(item)).join(', ');
};
const uniqueList = (values) => Array.from(new Set((values || []).filter(Boolean)));
const relationPageSize = 200;
const mergeRelations = (current = [], incoming = []) => {
  const seen = {};
  return [...current, ...incoming].filter((relation) => {
    const key = relation.id || `${relation.sourceAssetId}-${relation.targetAssetId}-${relation.relationType}-${relation.source}`;
    if (seen[key]) {
      return false;
    }
    seen[key] = true;
    return true;
  });
};
const renderTime = (value) => !value || String(value).indexOf('0001-01-01') === 0 ? '-' : value;
const numberText = (value) => Number(value || 0).toLocaleString();
const percent = (value) => Math.max(0, Math.min(100, Number(value || 0)));
const percentText = (value) => `${percent(value).toFixed(1)}%`;
const shortDate = (value) => value ? String(value).slice(5) : '-';
const normalizeTrendDays = (value) => [14, 30].includes(Number(value)) ? Number(value) : 7;
const normalizeFailureThreshold = (value) => [30, 50, 80].includes(Number(value)) ? Number(value) : 50;
const trendDateFormat = 'YYYY-MM-DD';
const normalizeTrendRange = (start, end) => {
  const startMoment = moment(start, trendDateFormat, true);
  const endMoment = moment(end, trendDateFormat, true);
  if (!startMoment.isValid() || !endMoment.isValid() || endMoment.isBefore(startMoment, 'day')) {
    return [];
  }
  if (endMoment.diff(startMoment, 'days') > 89) {
    return [];
  }
  return [startMoment.format(trendDateFormat), endMoment.format(trendDateFormat)];
};
const trendRangeMoments = (range) => Array.isArray(range) && range.length === 2
  ? [moment(range[0], trendDateFormat), moment(range[1], trendDateFormat)]
  : null;
const csvCell = (value) => {
  const text = value === undefined || value === null ? '' : String(value);
  return /[",\r\n]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text;
};
const csvRateText = (count, total) => Number(total || 0) > 0 ? `${((Number(count || 0) * 100) / Number(total || 0)).toFixed(1)}%` : '0.0%';
const safeFilenamePart = (value) => String(value || 'all').replace(/[^a-zA-Z0-9_.-]+/g, '-').replace(/^-+|-+$/g, '') || 'all';
const downloadTextFile = (filename, content, mimeType) => {
  const blob = new Blob([content], { type: mimeType });
  const link = document.createElement('a');
  link.href = window.URL.createObjectURL(blob);
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(link.href);
};
const renderLifecycle = (value) => lifecycleMap[value] || value || '-';
const renderCost = (value) => {
  const cost = Number(value);
  return Number.isNaN(cost) ? '-' : cost.toFixed(2);
};
const renderComplianceRisk = (value) => (
  value ? <Tag color={complianceRiskColorMap[value]}>{complianceRiskMap[value] || value}</Tag> : '-'
);
const renderApplicationRisk = (value) => (
  value ? <Tag color={applicationRiskColorMap[value]}>{applicationRiskMap[value] || value}</Tag> : '-'
);
const renderActionRisk = (value) => (
  value ? <Tag color={actionRiskColorMap[value]}>{actionRiskMap[value] || value}</Tag> : '-'
);
const renderActionAdapterMode = (value) => (
  value ? <Tag color={actionAdapterModeColorMap[value]}>{actionAdapterModeMap[value] || value}</Tag> : '-'
);
const renderActionAdapterStatus = (value) => (
  value ? <Tag color={actionAdapterStatusColorMap[value]}>{actionAdapterStatusMap[value] || value}</Tag> : '-'
);
const renderCompactList = (value) => Array.isArray(value) && value.length ? value.join(' / ') : '-';
const applicationAssetDsl = (application) => `application:"${String(application || '').replace(/"/g, '\\"')}"`;
const volumeSizeGiB = (asset = {}) => {
  const attrs = [asset.attributes || {}, asset.rawData || {}];
  const keys = [
    'sizeGiB', 'size_gib', 'sizeInGiB', 'size_in_gib',
    'sizeGB', 'size_gb', 'sizeInGBs', 'size_in_gbs',
    'volumeSize', 'volume_size', 'VolumeSize', 'size'
  ];
  for (const item of attrs) {
    for (const key of keys) {
      const value = Number(item && item[key]);
      if (value > 0) {
        return value;
      }
    }
  }
  return 0;
};
const defaultResizeVolumeParams = (asset = {}) => {
  const currentSize = volumeSizeGiB(asset);
  return {
    targetSizeGiB: currentSize > 0 ? currentSize + 10 : 100
  };
};
const computeSpecKeys = [
  'instanceType', 'instance_type', 'InstanceType', 'shape', 'Shape',
  'shapeName', 'shape_name', 'flavor', 'flavorId', 'flavor_id'
];
const computeSpec = (asset = {}) => {
  const attrs = [asset.attributes || {}, asset.rawData || {}];
  for (const item of attrs) {
    for (const key of computeSpecKeys) {
      const value = item && item[key];
      if (value !== undefined && value !== null && String(value).trim()) {
        return String(value).trim();
      }
    }
  }
  return '';
};
const defaultAssetActionParams = (actionKey, asset = {}) => {
  switch (actionKey) {
    case 'resize_volume':
      return defaultResizeVolumeParams(asset);
    case 'resize_instance':
      return {
        targetInstanceType: computeSpec(asset) || '请输入目标规格'
      };
    case 'create_snapshot':
      return {
        snapshotName: `cloudiac-${asset.nativeId || asset.id || 'snapshot'}`,
        description: 'Created from CloudIaC'
      };
    case 'update_security_rules':
      return {
        operation: 'authorize',
        direction: 'ingress',
        protocol: 'tcp',
        cidr: '0.0.0.0/0',
        portRange: '22',
        description: 'Created from CloudIaC'
      };
    default:
      return {};
  }
};
const assetActionNeedsParams = (actionKey) => [
  'resize_instance',
  'resize_volume',
  'create_snapshot',
  'update_security_rules'
].includes(actionKey);
const assetActionParamsPlaceholder = (actionKey) => JSON.stringify(defaultAssetActionParams(actionKey, {}), null, 2);
const validateAssetActionParams = (actionKey, params = {}, asset = {}) => {
  if (!assetActionNeedsParams(actionKey)) {
    return '';
  }
  if (!params || Array.isArray(params) || typeof params !== 'object') {
    return '参数 JSON 必须是对象';
  }
  if (actionKey === 'resize_volume') {
    const targetSizeGiB = Number(params && params.targetSizeGiB);
    const currentSizeGiB = volumeSizeGiB(asset);
    if (!targetSizeGiB || Number.isNaN(targetSizeGiB) || targetSizeGiB <= 0) {
      return '请填写 targetSizeGiB 正整数';
    }
    if (currentSizeGiB > 0 && targetSizeGiB <= currentSizeGiB) {
      return `目标容量需大于当前容量 ${currentSizeGiB} GiB`;
    }
  }
  if (actionKey === 'resize_instance' && !String(params.targetInstanceType || params.targetShape || params.shape || '').trim()) {
    return '请填写 targetInstanceType 或 targetShape';
  }
  if (actionKey === 'update_security_rules') {
    const operation = String(params.operation || '').trim();
    const direction = String(params.direction || '').trim();
    const protocol = String(params.protocol || '').trim();
    const cidr = String(params.cidr || params.source || params.destination || '').trim();
    const ruleId = String(params.ruleId || params.rule_id || '').trim();
    const portRange = String(params.portRange || params.port_range || '').trim();
    const fromPort = params.fromPort || params.port;
    if (!['authorize', 'revoke'].includes(operation)) {
      return 'operation 必须为 authorize 或 revoke';
    }
    if (!['ingress', 'egress'].includes(direction)) {
      return 'direction 必须为 ingress 或 egress';
    }
    if (!protocol) {
      return 'protocol 不能为空';
    }
    if (!cidr && !(operation === 'revoke' && ruleId)) {
      return 'cidr 不能为空，撤销规则至少需要 ruleId 或 cidr';
    }
    if (!['all', '-1', 'icmp', '1'].includes(protocol.toLowerCase()) && !portRange && fromPort === undefined) {
      return 'TCP/UDP 规则需要 portRange 或 fromPort/toPort';
    }
  }
  return '';
};

const appendQueryParam = (params, key, value) => {
  if (value === undefined || value === null || value === '') {
    return;
  }
  params.push(`${encodeURIComponent(key)}=${encodeURIComponent(value)}`);
};

const readTextFile = (file) => new Promise((resolve, reject) => {
  const reader = new FileReader();
  reader.onload = () => resolve(reader.result);
  reader.onerror = () => reject(reader.error);
  reader.readAsText(file);
});

const normalizeImportAssets = (payload) => {
  if (Array.isArray(payload)) {
    return payload;
  }
  return payload && Array.isArray(payload.assets) ? payload.assets : [];
};

const renderJSON = (value) => {
  const content = value ? JSON.stringify(value, null, 2) : '{}';
  return <pre className={styles.jsonBlock}>{content}</pre>;
};

const envResourceDetailUrl = (orgId, asset = {}) => {
  if (!orgId || !asset.projectId || !asset.envId || !asset.iacResourceId) {
    return '';
  }
  return `/org/${orgId}/project/${asset.projectId}/m-project-env/detail/${asset.envId}?${queryString.stringify({
    tabKey: 'resource',
    resourceId: asset.iacResourceId
  })}`;
};

const objectPathValue = (value, path) => {
  if (!value || typeof value !== 'object' || !path) {
    return undefined;
  }
  const lookup = (current, key) => {
    if (!current || typeof current !== 'object') {
      return undefined;
    }
    if (Object.prototype.hasOwnProperty.call(current, key)) {
      return current[key];
    }
    const lowerKey = String(key).toLowerCase();
    const matchedKey = Object.keys(current).find((item) => String(item).toLowerCase() === lowerKey);
    return matchedKey ? current[matchedKey] : undefined;
  };
  return String(path).split('.').reduce((current, key) => {
    if (!hasValue(current)) {
      return undefined;
    }
    return lookup(current, key);
  }, value);
};

const assetAttrSources = (asset = {}) => {
  const raw = asset.rawData || {};
  return [
    asset.attributes || {},
    raw,
    objectPathValue(raw, 'response'),
    objectPathValue(raw, 'response.properties'),
    objectPathValue(raw, 'response.cluster'),
    objectPathValue(raw, 'cluster'),
    objectPathValue(raw, 'properties'),
    objectPathValue(raw, 'metadata')
  ].filter((item) => item && typeof item === 'object');
};

const assetAttrValue = (asset, keys) => {
  const sources = assetAttrSources(asset || {});
  for (const source of sources) {
    for (const key of keys) {
      const value = objectPathValue(source, key);
      if (hasValue(value)) {
        return value;
      }
    }
  }
  return undefined;
};

const asObjectList = (value) => {
  if (!hasValue(value)) {
    return [];
  }
  if (Array.isArray(value)) {
    return value;
  }
  if (typeof value === 'object') {
    const nested = value.items || value.Items || value.nodePools || value.nodeGroups || value.clusters;
    if (Array.isArray(nested)) {
      return nested;
    }
    const values = Object.values(value);
    if (values.length && values.every((item) => item && typeof item === 'object')) {
      return values;
    }
    return Object.keys(value).length ? [value] : [];
  }
  if (typeof value === 'string') {
    return value.split(',').map((item) => item.trim()).filter(Boolean);
  }
  return [value];
};

const assetAttrListValue = (asset, keys) => asObjectList(assetAttrValue(asset, keys));

const renderAttrValue = (value) => {
  if (value === undefined || value === null || value === '') {
    return '-';
  }
  if (Array.isArray(value)) {
    return value.length ? value.map((item) => typeof item === 'object' ? JSON.stringify(item) : item).join(', ') : '-';
  }
  if (typeof value === 'object') {
    return <pre className={styles.jsonBlock}>{JSON.stringify(value, null, 2)}</pre>;
  }
  return String(value);
};

const objectAttrValue = (value, keys) => {
  if (!value || typeof value !== 'object') {
    return undefined;
  }
  for (const key of keys) {
    if (value[key] !== undefined && value[key] !== null && value[key] !== '') {
      return value[key];
    }
  }
  return undefined;
};

const isKubernetesClusterAsset = (asset = {}) => {
  const assetType = String(asset.assetType || '').toLowerCase();
  const nativeType = String(asset.nativeType || '').toLowerCase();
  const nativeTypeLooksLikeCluster =
    nativeType.includes('eks_cluster') ||
    nativeType.includes('aws_eks') ||
    nativeType.includes('containerengine_cluster') ||
    nativeType.includes('oci_containerengine') ||
    nativeType.includes('oke_cluster') ||
    nativeType.includes('aks_cluster') ||
    nativeType.includes('azure_kubernetes') ||
    nativeType.includes('gke_cluster') ||
    nativeType.includes('gcp_container') ||
    nativeType.includes('alicloud_cs_kubernetes') ||
    nativeType.includes('tencentcloud_kubernetes') ||
    nativeType.includes('huawei_cce') ||
    nativeType.includes('kubernetes_cluster') ||
    nativeType.includes('k8s_cluster');
  return assetType === 'kubernetes_cluster' ||
    nativeTypeLooksLikeCluster ||
    assetAttrValue(asset, [
      'nodeGroups',
      'nodegroups',
      'node_groups',
      'nodePools',
      'nodepools',
      'node_pools',
      'kubernetesVersion',
      'kubernetes_version',
      'clusterEndpoint',
      'apiEndpoint',
      'endpoint',
      'endpoints.kubernetes',
      'endpointConfig',
      'kubernetesNetwork'
    ]) !== undefined;
};

const isKubernetesAsset = (asset = {}) => {
  const assetType = String(asset.assetType || '').toLowerCase();
  const nativeType = String(asset.nativeType || '').toLowerCase();
  return isKubernetesClusterAsset(asset) ||
    kubernetesAssetTypes.includes(assetType) ||
    assetType.startsWith('k8s_') ||
    nativeType.startsWith('kubernetes_') ||
    nativeType.startsWith('k8s_') ||
    nativeType.includes('_kubernetes_');
};

const kubernetesListItems = (items, kind) => (
  Array.isArray(items)
    ? items.map((item, index) => (
      item && typeof item === 'object'
        ? { ...item, k8sKind: kind, _k8sIndex: index }
        : { name: String(item), id: String(item), k8sKind: kind, _k8sIndex: index }
    ))
    : []
);

const kubernetesNodeItems = (asset = {}) => {
  const groups = assetAttrListValue(asset, ['nodeGroups', 'nodegroups', 'node_groups']);
  const pools = assetAttrListValue(asset, ['nodePools', 'nodepools', 'node_pools', 'agentPoolProfiles']);
  return [
    ...kubernetesListItems(groups, 'NodeGroup'),
    ...kubernetesListItems(pools, 'NodePool')
  ];
};

const kubernetesNodeScaleText = (record = {}) => {
  const scaling = record.scalingConfig || {};
  if (scaling.desiredSize !== undefined || scaling.minSize !== undefined || scaling.maxSize !== undefined) {
    return `期望 ${scaling.desiredSize || 0} / 最小 ${scaling.minSize || 0} / 最大 ${scaling.maxSize || 0}`;
  }
  if (record.nodeCount !== undefined) {
    return `${record.nodeCount}`;
  }
  if (record.initialNodeCount !== undefined) {
    return `${record.initialNodeCount}`;
  }
  const nodeConfig = record.nodeConfigDetails || {};
  if (nodeConfig.size !== undefined) {
    return `${nodeConfig.size}`;
  }
  if (record.quantityPerSubnet !== undefined) {
    return `${record.quantityPerSubnet}/Subnet`;
  }
  return '-';
};

const kubernetesEndpointText = (asset = {}) => {
  const endpoint = asset.address || assetAttrValue(asset, [
    'endpoint',
    'apiEndpoint',
    'clusterEndpoint',
    'endpoints.kubernetes',
    'endpoints.publicEndpoint',
    'endpointConfig.kubernetes',
    'properties.fqdn',
    'fqdn',
    'privateEndpoint',
    'publicEndpoint'
  ]);
  if (endpoint) {
    return typeof endpoint === 'object' ? renderAttrValue(endpoint) : String(endpoint);
  }
  const endpoints = assetAttrValue(asset, ['endpoints', 'endpointConfig']);
  const endpointValue = objectAttrValue(endpoints, ['kubernetes', 'publicEndpoint', 'privateEndpoint', 'endpoint', 'endpointUrl']);
  return endpointValue ? String(endpointValue) : '-';
};

const kubernetesNodeSummary = (asset = {}) => {
  const nodeItems = kubernetesNodeItems(asset);
  const nodeGroupCountValue = assetAttrValue(asset, ['nodeGroupCount', 'nodegroupCount', 'node_group_count']);
  const nodePoolCountValue = assetAttrValue(asset, ['nodePoolCount', 'nodepoolCount', 'node_pool_count']);
  const nodeGroupCount = nodeGroupCountValue !== undefined ? Number(nodeGroupCountValue || 0) : nodeItems.filter((item) => item.k8sKind === 'NodeGroup').length;
  const nodePoolCount = nodePoolCountValue !== undefined ? Number(nodePoolCountValue || 0) : nodeItems.filter((item) => item.k8sKind === 'NodePool').length;
  if (!nodeGroupCount && !nodePoolCount) {
    return nodeItems.length ? `${nodeItems.length}` : '-';
  }
  return `节点组 ${nodeGroupCount} / 节点池 ${nodePoolCount}`;
};

const kubernetesWorkloadItems = (asset = {}) => {
  const workloads = [
    ...kubernetesListItems(assetAttrListValue(asset, ['workloads']), 'Workload'),
    ...kubernetesListItems(assetAttrListValue(asset, ['deployments']), 'Deployment'),
    ...kubernetesListItems(assetAttrListValue(asset, ['statefulSets', 'statefulsets', 'stateful_sets']), 'StatefulSet'),
    ...kubernetesListItems(assetAttrListValue(asset, ['daemonSets', 'daemonsets', 'daemon_sets']), 'DaemonSet'),
    ...kubernetesListItems(assetAttrListValue(asset, ['replicaSets', 'replicasets', 'replica_sets']), 'ReplicaSet'),
    ...kubernetesListItems(assetAttrListValue(asset, ['jobs']), 'Job'),
    ...kubernetesListItems(assetAttrListValue(asset, ['cronJobs', 'cronjobs', 'cron_jobs']), 'CronJob')
  ];
  return [
    ...kubernetesListItems(assetAttrListValue(asset, ['namespaces']), 'Namespace'),
    ...kubernetesListItems(assetAttrListValue(asset, ['nodes']), 'Node'),
    ...workloads,
    ...kubernetesListItems(assetAttrListValue(asset, ['pods']), 'Pod'),
    ...kubernetesListItems(assetAttrListValue(asset, ['services']), 'Service'),
    ...kubernetesListItems(assetAttrListValue(asset, ['ingresses']), 'Ingress')
  ];
};

const kubernetesWorkloadCount = (asset = {}, key, fallbackKinds = []) => {
  const explicit = assetAttrValue(asset, [key]);
  if (explicit !== undefined && explicit !== null && explicit !== '') {
    return Number(explicit || 0);
  }
  const kindSet = fallbackKinds.reduce((acc, kind) => ({ ...acc, [kind]: true }), {});
  return kubernetesWorkloadItems(asset).filter((item) => kindSet[item.k8sKind]).length;
};

const kubernetesWorkloadSummary = (asset = {}) => {
  const namespaceCount = kubernetesWorkloadCount(asset, 'namespaceCount', ['Namespace']);
  const nodeCount = kubernetesWorkloadCount(asset, 'nodeCount', ['Node']);
  const workloadCount = kubernetesWorkloadCount(asset, 'workloadCount', ['Workload', 'Deployment', 'StatefulSet', 'DaemonSet', 'ReplicaSet', 'Job', 'CronJob']);
  const podCount = kubernetesWorkloadCount(asset, 'podCount', ['Pod']);
  const serviceCount = kubernetesWorkloadCount(asset, 'serviceCount', ['Service']);
  const ingressCount = kubernetesWorkloadCount(asset, 'ingressCount', ['Ingress']);
  if (!namespaceCount && !nodeCount && !workloadCount && !podCount && !serviceCount && !ingressCount) {
    return '-';
  }
  return `NS ${namespaceCount} / Node ${nodeCount} / Workload ${workloadCount} / Pod ${podCount} / Svc ${serviceCount} / Ingress ${ingressCount}`;
};

const kubernetesObjectName = (record = {}) => (
  record.name ||
  record.displayName ||
  record.metadataName ||
  objectAttrValue(record.metadata, ['name']) ||
  record.uid ||
  record.id ||
  record.nodegroupName ||
  record.nodeGroupName ||
  record.nodePoolName ||
  '-'
);

const kubernetesObjectStatus = (record = {}) => (
  record.status || record.phase || record.state || record.lifecycleState || record.provisioningState || record.ready || record.conditionsSummary || ''
);

const kubernetesObjectStatusTag = (status) => {
  const text = String(status || '').trim();
  if (!text) {
    return '-';
  }
  const lower = text.toLowerCase();
  const color = lower.includes('running') || lower.includes('ready') || lower.includes('active') || lower.includes('succeeded')
    ? 'success'
    : lower.includes('fail') || lower.includes('error') || lower.includes('crash')
      ? 'error'
      : 'default';
  return <Tag color={color}>{text}</Tag>;
};

const kubernetesObjectSummary = (record = {}) => {
  const parts = [];
  if (record.replicas !== undefined || record.readyReplicas !== undefined) {
    parts.push(`副本 ${record.readyReplicas || 0}/${record.replicas || 0}`);
  }
  if (record.availableReplicas !== undefined) {
    parts.push(`可用 ${record.availableReplicas}`);
  }
  if (record.type) {
    parts.push(`类型 ${record.type}`);
  }
  if (record.clusterIP) {
    parts.push(`ClusterIP ${record.clusterIP}`);
  }
  if (record.podIP) {
    parts.push(`PodIP ${record.podIP}`);
  }
  if (record.nodeName) {
    parts.push(`节点 ${record.nodeName}`);
  }
  if (record.hosts) {
    parts.push(`Host ${joinList(Array.isArray(record.hosts) ? record.hosts : [record.hosts])}`);
  }
  if (record.images) {
    parts.push(`镜像 ${joinList(Array.isArray(record.images) ? record.images : [record.images])}`);
  }
  if (record.ports) {
    parts.push(`端口 ${Array.isArray(record.ports) || typeof record.ports === 'object' ? JSON.stringify(record.ports) : record.ports}`);
  }
  return parts.length ? parts.join('；') : '-';
};

const KubernetesAssetInfo = ({ asset = {} }) => {
  const nodeItems = kubernetesNodeItems(asset);
  const workloadItems = kubernetesWorkloadItems(asset);
  const nodeCollectError = assetAttrValue(asset, ['nodeGroupCollectError', 'nodePoolCollectError']);
  const workloadCollectError = assetAttrValue(asset, ['workloadCollectError', 'kubernetesWorkloadCollectError']);
  const workloadColumns = [
    {
      title: '类型',
      dataIndex: 'k8sKind',
      width: 120
    },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 160,
      render: (text, record) => text || record.namespaceName || '-'
    },
    {
      title: '名称',
      key: 'name',
      width: 220,
      render: (_, record) => kubernetesObjectName(record)
    },
    {
      title: '状态',
      key: 'status',
      width: 140,
      render: (_, record) => kubernetesObjectStatusTag(kubernetesObjectStatus(record))
    },
    {
      title: '摘要',
      key: 'summary',
      width: 420,
      render: (_, record) => kubernetesObjectSummary(record)
    }
  ];
  const nodeColumns = [
    {
      title: '类型',
      dataIndex: 'k8sKind',
      width: 100
    },
    {
      title: '名称',
      dataIndex: 'name',
      width: 180,
      render: (text, record) => text || record.nodegroupName || record.id || '-'
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 110,
      render: (text, record) => kubernetesObjectStatusTag(text || record.lifecycleState || record.provisioningState)
    },
    {
      title: '版本',
      dataIndex: 'version',
      width: 110,
      render: (text, record) => text || record.kubernetesVersion || '-'
    },
    {
      title: '规格',
      dataIndex: 'instanceTypes',
      width: 180,
      render: (value, record) => joinList(value || record.nodeShape || record.vmSize || record.machineType || record.instanceType)
    },
    {
      title: '规模',
      key: 'scale',
      width: 180,
      render: (_, record) => kubernetesNodeScaleText(record)
    },
    {
      title: '子网',
      dataIndex: 'subnets',
      width: 260,
      render: (value, record) => joinList(value || record.subnetIds || record.subnetId || record.subnetIDs || [])
    }
  ];
  return (
    <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
      <Descriptions column={2} size='small' bordered={true}>
        <Descriptions.Item label='集群名称'>{assetAttrValue(asset, ['name', 'displayName', 'clusterName']) || asset.name || '-'}</Descriptions.Item>
        <Descriptions.Item label='K8S版本'>{assetAttrValue(asset, ['version', 'kubernetesVersion', 'currentMasterVersion', 'currentNodeVersion']) || '-'}</Descriptions.Item>
        <Descriptions.Item label='平台版本'>{assetAttrValue(asset, ['platformVersion']) || '-'}</Descriptions.Item>
        <Descriptions.Item label='集群状态'>{asset.status || assetAttrValue(asset, ['status', 'lifecycleState', 'provisioningState']) || '-'}</Descriptions.Item>
        <Descriptions.Item label='对象类型'>{assetTypeMap[asset.assetType] || asset.assetType || '-'}</Descriptions.Item>
        <Descriptions.Item label='命名空间'>{assetAttrValue(asset, ['namespace', 'namespaceName']) || '-'}</Descriptions.Item>
        <Descriptions.Item label='API Endpoint' span={2}>{kubernetesEndpointText(asset)}</Descriptions.Item>
        <Descriptions.Item label='VPC/VNet/VCN'>
          {assetAttrValue(asset, ['vpcId', 'resourcesVpcConfig.vpcId', 'vnetId', 'vcnId', 'networkId', 'network', 'networkConfig.network']) || '-'}
        </Descriptions.Item>
        <Descriptions.Item label='Endpoint配置'>{renderAttrValue(assetAttrValue(asset, ['endpointConfig', 'privateClusterConfig']))}</Descriptions.Item>
        <Descriptions.Item label='子网' span={2}>
          {renderAttrValue(assetAttrValue(asset, ['subnetIds', 'resourcesVpcConfig.subnetIds', 'subnetworkIds', 'subnetwork', 'endpointConfig.subnetId']))}
        </Descriptions.Item>
        <Descriptions.Item label='安全组/NSG' span={2}>
          {renderAttrValue(assetAttrValue(asset, ['securityGroupIds', 'resourcesVpcConfig.securityGroupIds', 'clusterSecurityGroupId', 'resourcesVpcConfig.clusterSecurityGroupId', 'nsgIds', 'endpointConfig.nsgIds']))}
        </Descriptions.Item>
        <Descriptions.Item label='网络配置' span={2}>{renderAttrValue(assetAttrValue(asset, ['kubernetesNetwork', 'networkProfile', 'networkConfig', 'options']))}</Descriptions.Item>
        <Descriptions.Item label='工作负载层' span={2}>{kubernetesWorkloadSummary(asset)}</Descriptions.Item>
      </Descriptions>
      {nodeCollectError && (
        <Alert
          type='warning'
          showIcon={true}
          message='节点组/节点池采集未完成'
          description={nodeCollectError}
        />
      )}
      <Table
        size='small'
        rowKey={(record, index) => `${record.k8sKind}-${record.name || record.id || index}`}
        columns={nodeColumns}
        dataSource={nodeItems}
        pagination={false}
        scroll={{ x: 'max-content' }}
        locale={{
          emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无节点组/节点池数据'/>
        }}
      />
      {workloadCollectError && (
        <Alert
          type='warning'
          showIcon={true}
          message='工作负载层采集未完成'
          description={workloadCollectError}
        />
      )}
      <Table
        size='small'
        rowKey={(record, index) => `${record.k8sKind}-${record.namespace || record.namespaceName || '-'}-${kubernetesObjectName(record)}-${record.uid || record._k8sIndex || index}`}
        columns={workloadColumns}
        dataSource={workloadItems}
        pagination={{ pageSize: 8 }}
        scroll={{ x: 'max-content' }}
        locale={{
          emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无 Namespace/Node/Pod/Workload/Service/Ingress 数据'/>
        }}
      />
    </Space>
  );
};

const renderSyncTaskSchedule = (stats) => {
  const safeStats = stats || {};
  const scheduleName = safeStats.syncPolicyScheduleName || safeStats.syncPolicyScheduleKey;
  if (!scheduleName) {
    return '-';
  }
  return (
    <Space direction='vertical' size={0}>
      <Text>{scheduleName}</Text>
      {safeStats.syncPolicyScheduleKey && safeStats.syncPolicyScheduleKey !== scheduleName && (
        <Text type='secondary'>{safeStats.syncPolicyScheduleKey}</Text>
      )}
    </Space>
  );
};

const durationText = (value) => {
  const ms = Number(value || 0);
  if (!ms) {
    return '-';
  }
  if (ms < 1000) {
    return `${ms}ms`;
  }
  if (ms < 60000) {
    return `${(ms / 1000).toFixed(1)}s`;
  }
  return `${Math.floor(ms / 60000)}m ${Math.round((ms % 60000) / 1000)}s`;
};

const httpStatusColor = (value) => {
  const status = Number(value || 0);
  if (!status) {
    return 'default';
  }
  if (status >= 500 || status === 429) {
    return 'error';
  }
  if (status >= 400) {
    return 'warning';
  }
  return 'success';
};

const syncTaskScopeSummary = (task = {}) => {
  const stats = task.stats || {};
  const scope = task.scopeSummary || {};
  return {
    ...scope,
    regions: scope.regions || stats.regions || task.regions || [],
    assetTypes: scope.assetTypes || stats.assetTypes || task.assetTypes || [],
    regionCount: scope.regionCount || ((scope.regions || stats.regions || task.regions || []).length),
    assetTypeCount: scope.assetTypeCount || ((scope.assetTypes || stats.assetTypes || task.assetTypes || []).length),
    scopeCount: scope.scopeCount || ((scope.scopeMetrics || stats.scopeMetrics || []).length),
    collected: scope.collected !== undefined ? scope.collected : stats.collected,
    created: scope.created !== undefined ? scope.created : stats.created,
    updated: scope.updated !== undefined ? scope.updated : stats.updated,
    skipped: scope.skipped !== undefined ? scope.skipped : stats.skipped,
    durationMs: scope.durationMs !== undefined ? scope.durationMs : stats.durationMs,
    collectorDurationMs: scope.collectorDurationMs !== undefined ? scope.collectorDurationMs : stats.collectorDurationMs,
    regionMetrics: scope.regionMetrics || stats.regionMetrics || [],
    assetTypeMetrics: scope.assetTypeMetrics || stats.assetTypeMetrics || [],
    scopeMetrics: scope.scopeMetrics || stats.scopeMetrics || [],
    failedScopes: scope.failedScopes || []
  };
};

const syncTaskFailureSummary = (task = {}) => {
  const stats = task.stats || {};
  const failure = task.failureSummary || {};
  const rawFailure = stats.failureSummary || {};
  const details = failure.details || stats.failureDetails || [];
  return {
    ...failure,
    total: failure.total !== undefined ? failure.total : rawFailure.total,
    retryableTotal: failure.retryableTotal !== undefined ? failure.retryableTotal : rawFailure.retryableTotal,
    details,
    firstMessage: failure.firstMessage || (details[0] && details[0].message) || task.errorMessage,
    retryHint: failure.retryHint || (details[0] && details[0].retryHint),
    categories: failure.categories || uniqueList(details.map((item) => item.category))
  };
};

const flattenSyncTaskApiMetrics = (stats = {}) => {
  const rawMetrics = stats.apiMetrics || {};
  if (Array.isArray(rawMetrics)) {
    return rawMetrics;
  }
  return Object.keys(rawMetrics).reduce((items, region) => {
    const regionItems = Array.isArray(rawMetrics[region]) ? rawMetrics[region] : [];
    return items.concat(regionItems.map((item, index) => ({
      ...item,
      region: item.region || region,
      metricKey: `${region}-${item.service || '-'}-${item.path || '-'}-${index}`
    })));
  }, []);
};

const renderSyncTaskScopeSummary = (task) => {
  const scope = syncTaskScopeSummary(task);
  const assetTypeText = joinList((scope.assetTypes || []).map((it) => assetTypeMap[it] || it));
  const failedScopes = scope.failedScopes || [];
  return (
    <Space direction='vertical' size={0}>
      <Text>区域 {scope.regionCount || (scope.regions || []).length} / 类型 {scope.assetTypeCount || (scope.assetTypes || []).length} / Scope {scope.scopeCount || '-'}</Text>
      <Text type='secondary'>采集 {numberText(scope.collected)}，新增 {numberText(scope.created)}，更新 {numberText(scope.updated)}，耗时 {durationText(scope.durationMs)}</Text>
      <Text type='secondary'>区域：{joinList(scope.regions)}；类型：{assetTypeText}</Text>
      {failedScopes.length > 0 && (
        <Text type='danger'>失败范围 {failedScopes.length} 个</Text>
      )}
    </Space>
  );
};

const renderSyncTaskFailureSummary = (task) => {
  const failure = syncTaskFailureSummary(task);
  const total = Number(failure.total || 0);
  if (!total && !failure.firstMessage) {
    return '-';
  }
  return (
    <Space direction='vertical' size={0}>
      <Space size={4} wrap={true}>
        <Tag color={total > 0 ? 'error' : 'default'}>失败 {numberText(total)}</Tag>
        {Number(failure.retryableTotal || 0) > 0 && <Tag color='warning'>可重试 {numberText(failure.retryableTotal)}</Tag>}
        {(failure.categories || []).map((category) => (
          <Tag key={category} color={syncTaskFailureCategoryColorMap[category]}>
            {syncTaskFailureCategoryMap[category] || category}
          </Tag>
        ))}
      </Space>
      {failure.firstMessage && <Text type='danger' ellipsis={true}>{failure.firstMessage}</Text>}
      {failure.retryHint && <Text type='secondary'>{failure.retryHint}</Text>}
    </Space>
  );
};

const renderSyncTaskApiMetricTrend = (trend = []) => {
  const points = (trend || []).filter((point) => Number(point.callCount || 0) > 0).slice(-7);
  if (!points.length) {
    return '-';
  }
  return (
    <Space size={4} wrap={true}>
      {points.map((point) => (
        <Tag
          key={point.date}
          color={Number(point.failedCount || 0) > 0 ? 'error' : Number(point.retriedCount || 0) > 0 ? 'warning' : 'default'}
          title={`调用 ${numberText(point.callCount)}，失败 ${numberText(point.failedCount)}，重试 ${numberText(point.retriedCount)}，最大耗时 ${durationText(point.maxDurationMs)}`}
        >
          {shortDate(point.date)} {numberText(point.callCount)}/{numberText(point.failedCount)}
        </Tag>
      ))}
    </Space>
  );
};

const renderRerunParameterDiffs = (stats) => {
  const diffs = stats && stats.rerunParameterDiffs;
  if (!diffs) {
    return '-';
  }
  const renderDiff = (label, diff, mapper) => {
    if (!diff) {
      return null;
    }
    const from = Array.isArray(diff.from) ? diff.from : [];
    const to = Array.isArray(diff.to) ? diff.to : [];
    const format = (values) => joinList((values || []).map((value) => mapper ? mapper(value) : value));
    return (
      <Space key={label} size={8} wrap={true}>
        <Text type='secondary'>{label}</Text>
        <Tag color={diff.changed ? 'orange' : 'default'}>{diff.changed ? '已变化' : '未变化'}</Tag>
        <Tag color={diff.overridden ? 'blue' : 'default'}>{diff.overridden ? '已覆盖' : '沿用原值'}</Tag>
        <Text>{format(from)} → {format(to)}</Text>
      </Space>
    );
  };
  return (
    <Space direction='vertical' size={4}>
      {renderDiff('区域', diffs.regions, null)}
      {renderDiff('资产类型', diffs.assetTypes, (value) => assetTypeMap[value] || value)}
    </Space>
  );
};

const renderTagMap = (tags) => {
  const entries = Object.entries(tags || {});
  if (!entries.length) {
    return '-';
  }
  return (
    <Space size={[4, 4]} wrap={true}>
      {entries.slice(0, 4).map(([key, value]) => (
        <Tag key={key}>{key}: {String(value)}</Tag>
      ))}
      {entries.length > 4 && <Tag>+{entries.length - 4}</Tag>}
    </Space>
  );
};

const relationSourceLabel = (source) => {
  if (source === 'iac_dependency') {
    return 'IaC依赖';
  }
  if (source === 'application_inferred') {
    return '应用依赖推演';
  }
  if (source === 'cloud_inferred') {
    return '云推断';
  }
  if (source === 'manual_application') {
    return '人工应用关系';
  }
  return source || '-';
};
const renderRelationSources = (sources = []) => (
  Array.isArray(sources) && sources.length ? (
    <Space size={[4, 4]} wrap={true}>
      {sources.map((source) => (
        <Tag key={source}>{relationSourceLabel(source)}</Tag>
      ))}
    </Space>
  ) : '-'
);

const relationDirection = (relation = {}, asset = {}) => {
  if (relation.sourceAssetId === asset.id) {
    return 'outgoing';
  }
  if (relation.targetAssetId === asset.id) {
    return 'incoming';
  }
  return 'related';
};

const relationSearchText = (relation = {}) => [
  relation.sourceAssetName,
  relation.sourceAssetId,
  relation.sourceAssetType,
  relation.targetAssetName,
  relation.targetAssetId,
  relation.targetAssetType,
  relation.relationType,
  relation.source,
  relationSourceLabel(relation.source),
  relation.metadata ? JSON.stringify(relation.metadata) : ''
].filter(Boolean).join(' ').toLowerCase();

const assetManagedBy = (asset = {}) => {
  if (asset.managedBy) {
    return asset.managedBy;
  }
  if (asset.source === 'iac_resource' || asset.iacResourceId) {
    return 'iac';
  }
  if (asset.source === 'cloud_collect') {
    return !asset.projectId && !asset.envId && !asset.iacResourceId ? 'cloud_only' : 'cloud_linked';
  }
  return 'manual';
};

const CoverageMetric = ({ title, value, description, tone }) => (
  <div className={`${styles.coverageMetric} ${tone ? styles[tone] : ''}`}>
    <Text type='secondary'>{title}</Text>
    <div className={styles.coverageMetricValue}>{numberText(value)}</div>
    <Text type='secondary'>{description}</Text>
  </div>
);

const AssetInsightHeader = ({ title, description, children }) => (
  <div className={styles.assetInsightHeader}>
    <div className={styles.assetInsightHeaderMain}>
      <Text strong={true}>{title}</Text>
      <div className={styles.assetInsightHeaderDesc}>{description}</div>
    </div>
    {children ? (
      <Space className={styles.assetInsightHeaderStats} size={[6, 6]} wrap={true}>
        {children}
      </Space>
    ) : null}
  </div>
);

const RelationGraph = ({ asset, relations = [], onOpenDetail }) => {
  if (!relations.length) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无关系'/>;
  }
  const outgoing = relations.filter((relation) => relation.sourceAssetId === asset.id);
  const incoming = relations.filter((relation) => relation.targetAssetId === asset.id);

  const renderNode = (relation, direction) => {
    const relatedId = direction === 'outgoing' ? relation.targetAssetId : relation.sourceAssetId;
    const name = direction === 'outgoing' ? relation.targetAssetName : relation.sourceAssetName;
    const type = direction === 'outgoing' ? relation.targetAssetType : relation.sourceAssetType;
    const label = relationTypeMap[relation.relationType] || relation.relationType || '-';
    return (
      <div className={styles.relationNode} key={`${direction}-${relation.id}`}>
        <a onClick={() => onOpenDetail(relatedId)}>{name || relatedId || '-'}</a>
        <Text type='secondary'>{assetTypeMap[type] || type || '-'}</Text>
        <Tag>{direction === 'outgoing' ? label : `被${label}`}</Tag>
        <Text type='secondary'>{relationSourceLabel(relation.source)}</Text>
      </div>
    );
  };

  return (
    <div className={styles.relationGraph}>
      <div className={styles.relationColumn}>
        <Text strong={true}>上游</Text>
        <Space direction='vertical' size={8} style={{ width: '100%' }}>
          {incoming.length ? incoming.map((relation) => renderNode(relation, 'incoming')) : <Text type='secondary'>暂无</Text>}
        </Space>
      </div>
      <div className={styles.relationCenter}>
        <div className={styles.currentNode}>
          <Text strong={true}>{asset.name || asset.nativeId || '-'}</Text>
          <Text type='secondary'>{assetTypeMap[asset.assetType] || asset.assetType || '-'}</Text>
        </div>
      </div>
      <div className={styles.relationColumn}>
        <Text strong={true}>下游</Text>
        <Space direction='vertical' size={8} style={{ width: '100%' }}>
          {outgoing.length ? outgoing.map((relation) => renderNode(relation, 'outgoing')) : <Text type='secondary'>暂无</Text>}
        </Space>
      </div>
    </div>
  );
};

const ApplicationGraph = ({ application, onOpenApplication }) => {
  const upstreams = application && application.upstreams || [];
  const downstreams = application && application.downstreams || [];
  if (!upstreams.length && !downstreams.length) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无应用依赖'/>;
  }

  const renderAppNode = (relation) => (
    <div className={styles.relationNode} key={`${relation.direction}-${relation.application}-${relation.source}`}>
      <a onClick={() => onOpenApplication(relation.application)}>{relation.application || '-'}</a>
      <Text type='secondary'>关系 {relation.assetRelationCount || 0} 条</Text>
      <Tag>{relationSourceLabel(relation.source)}</Tag>
    </div>
  );

  return (
    <div className={styles.relationGraph}>
      <div className={styles.relationColumn}>
        <Text strong={true}>调用本应用</Text>
        <Space direction='vertical' size={8} style={{ width: '100%' }}>
          {upstreams.length ? upstreams.map(renderAppNode) : <Text type='secondary'>暂无</Text>}
        </Space>
      </div>
      <div className={styles.relationCenter}>
        <div className={styles.currentNode}>
          <Text strong={true}>{application.application || '-'}</Text>
          <Text type='secondary'>资源 {application.assetCount || 0} 个</Text>
          {renderApplicationRisk(application.riskLevel)}
        </div>
      </div>
      <div className={styles.relationColumn}>
        <Text strong={true}>本应用调用</Text>
        <Space direction='vertical' size={8} style={{ width: '100%' }}>
          {downstreams.length ? downstreams.map(renderAppNode) : <Text type='secondary'>暂无</Text>}
        </Space>
      </div>
    </div>
  );
};

export default ({ match, location, history }) => {
  const { orgId } = match.params || {};
  const {
    assetId,
    syncTaskId,
    syncPolicyId,
    syncPolicyScheduleKey,
    syncPolicyScheduleName,
    trendDays: queryTrendDays,
    trendStartDate: queryTrendStartDate,
    trendEndDate: queryTrendEndDate,
    failureThreshold: queryFailureThreshold
  } = queryString.parse(location && location.search || '');
  const isCloudAssets = (location && location.pathname || '').includes('/m-cloud-assets');
  const pageTitle = isCloudAssets ? '云资产' : '资产 CMDB';
  const assetAPI = isCloudAssets ? cloudAssetAPI : cmdbAPI;
  const exportAssetsPath = isCloudAssets ? '/api/v1/cloud/assets/export' : '/api/v1/cmdb/assets/export';
  const importTemplatePath = isCloudAssets ? '/api/v1/cloud/assets/import-template' : '/api/v1/cmdb/assets/import-template';
  const [ownershipForm] = Form.useForm();
  const [batchOwnershipForm] = Form.useForm();
  const [riskRuleForm] = Form.useForm();
  const [ activeTab, setActiveTab ] = useState('assets');
  const [ assetKeyword, setAssetKeyword ] = useState('');
  const [ detailVisible, setDetailVisible ] = useState(false);
  const [ applicationDetailVisible, setApplicationDetailVisible ] = useState(false);
  const [ syncTaskDetailVisible, setSyncTaskDetailVisible ] = useState(false);
  const [ batchOwnershipVisible, setBatchOwnershipVisible ] = useState(false);
  const [ applicationDetailTab, setApplicationDetailTab ] = useState('relations');
  const [ assetActionDryRun, setAssetActionDryRun ] = useState(null);
  const [ assetActionDryRunVisible, setAssetActionDryRunVisible ] = useState(false);
  const [ assetActionDryRunRecord, setAssetActionDryRunRecord ] = useState(null);
  const [ assetActionDryRunParamsText, setAssetActionDryRunParamsText ] = useState('');
  const [ actionRequestKey, setActionRequestKey ] = useState('');
  const [ creatingActionKey, setCreatingActionKey ] = useState('');
  const [ assetActionCreateVisible, setAssetActionCreateVisible ] = useState(false);
  const [ assetActionCreateRecord, setAssetActionCreateRecord ] = useState(null);
  const [ assetActionCreateReason, setAssetActionCreateReason ] = useState('');
  const [ assetActionConfirmAction, setAssetActionConfirmAction ] = useState('');
  const [ assetActionConfirmResourceId, setAssetActionConfirmResourceId ] = useState('');
  const [ assetActionTagsText, setAssetActionTagsText ] = useState('');
  const [ assetActionParamsText, setAssetActionParamsText ] = useState('');
  const [ selectedAssetIds, setSelectedAssetIds ] = useState([]);
  const [ dslInput, setDslInput ] = useState('');
  const [ applicationKeyword, setApplicationKeyword ] = useState('');
  const [ applicationRisk, setApplicationRisk ] = useState(undefined);
  const [ riskRuleVisible, setRiskRuleVisible ] = useState(false);
  const [ applicationRelationForm, setApplicationRelationForm ] = useState({
    upstreams: [],
    downstreams: []
  });
  const [ relationFilters, setRelationFilters ] = useState({
    keyword: '',
    sources: [],
    types: [],
    direction: 'all'
  });
  const [ relationResp, setRelationResp ] = useState(null);
  const [ importOverwriteOwnership, setImportOverwriteOwnership ] = useState(true);
  const [ importPreviewVisible, setImportPreviewVisible ] = useState(false);
  const [ importPreview, setImportPreview ] = useState(null);
  const [ importPreviewPayload, setImportPreviewPayload ] = useState(null);
  const [ cloudForm, setCloudForm ] = useState({
    accountKey: undefined,
    regions: [],
    assetTypes: []
  });
  const [ syncTaskQuery, setSyncTaskQuery ] = useState({});
  const [ syncTaskTrendDays, setSyncTaskTrendDays ] = useState(normalizeTrendDays(firstQueryValue(queryTrendDays)));
  const [ syncTaskTrendRange, setSyncTaskTrendRange ] = useState(normalizeTrendRange(firstQueryValue(queryTrendStartDate), firstQueryValue(queryTrendEndDate)));
  const [ syncTaskFailureThreshold, setSyncTaskFailureThreshold ] = useState(normalizeFailureThreshold(firstQueryValue(queryFailureThreshold)));
  const [ selectedFailedSyncTaskIds, setSelectedFailedSyncTaskIds ] = useState([]);
  const [ syncTaskApiMetricFilters, setSyncTaskApiMetricFilters ] = useState(defaultSyncTaskApiMetricFilters);
  const [ batchRerunningSyncTasks, setBatchRerunningSyncTasks ] = useState(false);
  const [ syncTaskRerunReasonVisible, setSyncTaskRerunReasonVisible ] = useState(false);
  const [ syncTaskRerunMode, setSyncTaskRerunMode ] = useState('');
  const [ syncTaskRerunReason, setSyncTaskRerunReason ] = useState('');
  const [ syncTaskRerunRequiresApproval, setSyncTaskRerunRequiresApproval ] = useState(false);
  const [ syncTaskRerunSubmitting, setSyncTaskRerunSubmitting ] = useState(false);
  const [ syncTaskRerunGroupVisible, setSyncTaskRerunGroupVisible ] = useState(false);
  const [ syncTaskRerunGroupApproving, setSyncTaskRerunGroupApproving ] = useState(false);
  const [ syncTaskRerunRegions, setSyncTaskRerunRegions ] = useState([]);
  const [ syncTaskRerunAssetTypes, setSyncTaskRerunAssetTypes ] = useState([]);

  const {
    loading: tableLoading,
    data: tableData,
    run: fetchList
  } = useRequest(
    (params = {}) => requestWrapper(
      assetAPI.listAssets.bind(null, { orgId, ...params })
    ), {
      throttleInterval: 1000,
      manual: true
    }
  );

  const {
    data: filters = {},
    run: fetchFilters
  } = useRequest(
    (params = {}) => requestWrapper(
      assetAPI.assetFilters.bind(null, { orgId, ...params })
    ), {
      manual: true
    }
  );

  const {
    loading: assetPermissionsLoading,
    data: assetPermissions = {},
    run: fetchAssetPermissions
  } = useRequest(
    () => requestWrapper(
      assetAPI.assetPermissions.bind(null, { orgId })
    ), {
      manual: true
    }
  );

  const {
    loading: coverageLoading,
    data: coverage = {},
    run: fetchCoverage
  } = useRequest(
    () => requestWrapper(
      cloudAssetAPI.coverage.bind(null, { orgId })
    ), {
      manual: true
    }
  );

  const {
    loading: governanceReportLoading,
    data: governanceReport = {},
    run: fetchGovernanceReport
  } = useRequest(
    () => requestWrapper(
      assetAPI.assetGovernanceReport.bind(null, { orgId })
    ), {
      manual: true
    }
  );

  const {
    loading: applicationsLoading,
    data: applicationsData,
    run: fetchApplications
  } = useRequest(
    (params = {}) => requestWrapper(
      cmdbAPI.listApplications.bind(null, { orgId, currentPage: 1, pageSize: 10, ...params })
    ), {
      manual: true
    }
  );

  const {
    loading: applicationDetailLoading,
    data: applicationDetail,
    run: fetchApplicationDetail
  } = useRequest(
    (application) => requestWrapper(
      cmdbAPI.applicationDetail.bind(null, { orgId, application })
    ), {
      manual: true
    }
  );

  const {
    loading: applicationRelationSaving,
    run: updateApplicationRelations
  } = useRequest(
    (params) => requestWrapper(
      cmdbAPI.updateApplicationRelations.bind(null, { orgId, ...params }),
      {
        autoSuccess: true,
        successMessage: '应用依赖已保存'
      }
    ), {
      manual: true,
      onSuccess: (resp = {}) => {
        fetchApplications({ currentPage: 1, pageSize: 10, q: applicationKeyword, risk: applicationRisk });
        if (resp.application) {
          fetchApplicationDetail(resp.application);
        }
      }
    }
  );

  const {
    loading: riskRuleLoading,
    data: riskRuleConfig,
    run: fetchRiskRuleConfig
  } = useRequest(
    () => requestWrapper(
      cmdbAPI.riskRuleConfig.bind(null, { orgId })
    ), {
      manual: true,
      onSuccess: (resp = {}) => {
        riskRuleForm.setFieldsValue(resp);
      }
    }
  );

  const {
    loading: riskRuleSaving,
    run: updateRiskRuleConfig
  } = useRequest(
    (params) => requestWrapper(
      cmdbAPI.updateRiskRuleConfig.bind(null, { orgId, ...params }),
      {
        autoSuccess: true,
        successMessage: '风险规则已保存'
      }
    ), {
      manual: true,
      onSuccess: (resp = {}) => {
        riskRuleForm.setFieldsValue(resp);
        refreshApplications({ currentPage: 1, pageSize: 10 });
        setRiskRuleVisible(false);
      }
    }
  );

  const {
    loading: detailLoading,
    data: detail,
    run: fetchDetail
  } = useRequest(
    (id) => requestWrapper(
      assetAPI.assetDetail.bind(null, { orgId, id })
    ), {
      manual: true
    }
  );
  const {
    loading: relationLoading,
    run: fetchRelations
  } = useRequest(
    (id, params = {}) => requestWrapper(
      assetAPI.assetRelations.bind(null, { orgId, id, ...params })
    ), {
      manual: true,
      onSuccess: (resp = {}) => {
        const offset = Number((resp.summary || {}).offset || 0);
        setRelationResp((pre) => {
          if (offset > 0 && pre && pre.assetId === resp.assetId) {
            return {
              ...resp,
              relations: mergeRelations(pre.relations || [], resp.relations || [])
            };
          }
          return resp;
        });
      }
    }
  );
  const {
    loading: securityRulesLoading,
    data: securityRules = {},
    run: fetchSecurityRules
  } = useRequest(
    (id) => requestWrapper(cloudAssetAPI.securityRules.bind(null, { orgId, id })),
    {
      manual: true
    }
  );

  const {
    loading: assetActionsLoading,
    data: assetActions = [],
    run: fetchAssetActions
  } = useRequest(
    (id) => requestWrapper(
      cloudAssetAPI.assetActions.bind(null, { orgId, id })
    ), {
      manual: true
    }
  );

  const {
    loading: assetActionDryRunning,
    run: dryRunAssetAction
  } = useRequest(
    (payload) => requestWrapper(
      cloudAssetAPI.dryRunAssetAction.bind(null, { orgId, id: detail && detail.id, ...payload })
    ), {
      manual: true,
      onSuccess: (resp) => {
        setAssetActionDryRun(resp);
      }
    }
  );

  const {
    loading: assetActionCreating,
    run: submitAssetAction
  } = useRequest(
    (params) => requestWrapper(
      cloudAssetAPI.createAssetAction.bind(null, { orgId, id: detail && detail.id, ...params })
    ), {
      manual: true,
      onSuccess: (resp = {}) => {
        notification.success({
          message: '操作任务已创建',
          description: resp.id ? `任务 ${resp.id} 已完成记录` : '任务已完成记录'
        });
        if (detail && detail.id) {
          fetchDetail(detail.id);
          fetchAssetActions(detail.id);
        }
        refresh();
      }
    }
  );

  const {
    loading: ownershipSaving,
    run: updateAssetOwnership
  } = useRequest(
    (params) => requestWrapper(
      assetAPI.updateAssetOwnership.bind(null, { orgId, id: detail && detail.id, ...params }),
      {
        autoSuccess: true,
        successMessage: '资产归属已保存'
      }
    ), {
      manual: true,
      onSuccess: () => {
        if (detail && detail.id) {
          fetchDetail(detail.id);
        }
        refresh();
      }
    }
  );

  const {
    loading: batchOwnershipSaving,
    run: batchUpdateAssetOwnership
  } = useRequest(
    (params) => requestWrapper(
      assetAPI.batchUpdateAssetOwnership.bind(null, { orgId, ...params }),
      {
        successMessage: '资产治理信息已批量保存'
      }
    ), {
      manual: true,
      onSuccess: (resp = {}) => {
        notification.success({
          message: '批量治理完成',
          description: `总数 ${resp.total || 0}，更新 ${resp.updated || 0}，跳过 ${resp.skipped || 0}`
        });
        if (Array.isArray(resp.errors) && resp.errors.length) {
          notification.warning({
            message: '部分资产未更新',
            description: resp.errors.slice(0, 3).join('；')
          });
        }
        setSelectedAssetIds([]);
        setBatchOwnershipVisible(false);
        batchOwnershipForm.resetFields();
        refresh();
      }
    }
  );

  const {
    loading: importingAssets,
    run: importAssets
  } = useRequest(
    (params) => requestWrapper(
      assetAPI.importAssets.bind(null, { orgId, ...params })
    ), {
      manual: true,
      onSuccess: (resp = {}) => {
        notification.success({
          message: '资产导入完成',
          description: `总数 ${resp.total || 0}，新增 ${resp.created || 0}，更新 ${resp.updated || 0}，归属更新 ${resp.ownershipUpdated || 0}，跳过 ${resp.skipped || 0}`
        });
        if (Array.isArray(resp.errors) && resp.errors.length) {
          notification.warning({
            message: '部分资产导入异常',
            description: resp.errors.slice(0, 3).join('；')
          });
        }
        setImportPreviewVisible(false);
        setImportPreview(null);
        setImportPreviewPayload(null);
        refresh();
      }
    }
  );

  const {
    loading: previewingImportAssets,
    run: previewImportAssets
  } = useRequest(
    (params) => requestWrapper(
      assetAPI.importAssets.bind(null, { orgId, ...params })
    ), {
      manual: true,
      onSuccess: (resp = {}) => {
        setImportPreview(resp);
        setImportPreviewVisible(true);
        if (Array.isArray(resp.errors) && resp.errors.length) {
          notification.warning({
            message: '导入预检发现异常',
            description: resp.errors.slice(0, 3).join('；')
          });
        }
      }
    }
  );

  const {
    loading: backfillLoading,
    run: backfillIacResources
  } = useRequest(
    () => requestWrapper(
      assetAPI.backfillIacResources.bind(null, { orgId }),
      { autoSuccess: true }
    ), {
      manual: true,
      onSuccess: () => {
        fetchFilters();
        fetchList({ currentPage: 1, pageSize: 10 });
        if (isCloudAssets) {
          fetchCoverage();
          fetchGovernanceReport();
        }
      }
    }
  );

  const {
    loading: cloudAccountsLoading,
    data: cloudAccounts = [],
    run: fetchCloudAccounts
  } = useRequest(
    () => requestWrapper(
      cmdbAPI.cloudAccounts.bind(null, { orgId })
    ), {
      manual: true
    }
  );

  const {
    loading: syncTasksLoading,
    data: syncTasksData,
    run: fetchSyncTasks
  } = useRequest(
    (params = {}) => requestWrapper(
      assetAPI.syncTasks.bind(null, { orgId, currentPage: 1, pageSize: 10, ...params })
    ), {
      manual: true
    }
  );

  const {
    loading: syncTaskDetailLoading,
    data: syncTaskDetail,
    run: fetchSyncTaskDetail
  } = useRequest(
    (id) => requestWrapper(
      assetAPI.syncTaskDetail.bind(null, { orgId, id })
    ), {
      manual: true
    }
  );

  useEffect(() => {
    if (!syncTaskDetailVisible || !syncTaskDetail) {
      return;
    }
    setSyncTaskApiMetricFilters((prev) => ({
      ...prev,
      slowThresholdMs: syncTaskSlowApiThresholdMs(syncTaskDetail)
    }));
  }, [syncTaskDetailVisible, syncTaskDetail && syncTaskDetail.id]);

  const {
    loading: syncTaskRerunGroupLoading,
    data: syncTaskRerunGroup,
    run: fetchSyncTaskRerunGroup
  } = useRequest(
    (groupId) => requestWrapper(
      assetAPI.syncTaskRerunGroupDetail.bind(null, { orgId, groupId })
    ), {
      manual: true
    }
  );

  const {
    loading: cloudSyncLoading,
    run: startCloudSync
  } = useRequest(
    (params) => requestWrapper(
      assetAPI.startSyncTask.bind(null, { orgId, ...params }),
      {
        autoSuccess: true,
        successMessage: '采集任务已启动，后台运行中'
      }
    ), {
      manual: true,
      onSuccess: () => {
        refreshSyncTasks();
        fetchList({ currentPage: 1, pageSize: 10 });
        fetchFilters();
        if (isCloudAssets) {
          fetchCoverage();
          fetchGovernanceReport();
        }
      }
    }
  );

  const {
    loading: batchRerunFailedSyncLoading,
    run: batchRerunFailedSyncTasks
  } = useRequest(
    (params) => requestWrapper(
      assetAPI.batchRerunFailedSyncTasks.bind(null, { orgId, ...params })
    ), {
      manual: true
    }
  );

  const {
    tableProps,
    setSearchParams,
    onChangeFormParams,
    searchParams
  } = useSearchFormAndTable({
    tableData,
    onSearch: (params) => {
      const { current: currentPage, keyword, field, order, ...restParams } = params;
      const requestParams = {
        currentPage,
        pageSize: params.pageSize,
        q: keyword,
        sortField: field,
        sortOrder: order,
        projectIds: asCSV(restParams.projectIds),
        envIds: asCSV(restParams.envIds),
        providers: asCSV(restParams.providers),
        accountIds: asCSV(restParams.accountIds),
        assetTypes: asCSV(restParams.assetTypes),
        sources: asCSV(restParams.sources),
        statuses: asCSV(restParams.statuses),
        managedBy: asCSV(restParams.managedBy),
        dsl: restParams.dsl
      };
      fetchList(requestParams);
      fetchFilters(requestParams);
    }
  });

  const buildSyncTaskParams = (params = {}) => {
    const merged = {
      trendDays: syncTaskTrendDays,
      trendStartDate: syncTaskTrendRange[0],
      trendEndDate: syncTaskTrendRange[1],
      failureThreshold: syncTaskFailureThreshold,
      ...syncTaskQuery,
      ...params
    };
    const {
      syncPolicyScheduleName: _syncPolicyScheduleName,
      ...requestParams
    } = merged;
    return requestParams;
  };

  const refreshSyncTasks = (params = {}) => {
    fetchSyncTasks(buildSyncTaskParams(params));
  };

  const clearSyncTaskQuery = () => {
    setSyncTaskQuery({});
    setSelectedFailedSyncTaskIds([]);
    fetchSyncTasks({
      currentPage: 1,
      pageSize: 10,
      trendDays: syncTaskTrendDays,
      trendStartDate: syncTaskTrendRange[0],
      trendEndDate: syncTaskTrendRange[1],
      failureThreshold: syncTaskFailureThreshold
    });
    if (history && history.replace && isCloudAssets) {
      history.replace(`/org/${orgId}/m-cloud-assets`);
    }
  };

  const onSyncTaskTrendDaysChange = (value) => {
    setSyncTaskTrendDays(value);
    setSyncTaskTrendRange([]);
    refreshSyncTasks({
      currentPage: 1,
      pageSize: 10,
      trendDays: value,
      trendStartDate: undefined,
      trendEndDate: undefined
    });
  };

  const onSyncTaskFailureThresholdChange = (value) => {
    setSyncTaskFailureThreshold(value);
    refreshSyncTasks({ currentPage: 1, pageSize: 10, failureThreshold: value });
  };

  const onSyncTaskTrendRangeChange = (dates, dateStrings = []) => {
    const nextRange = normalizeTrendRange(dateStrings[0], dateStrings[1]);
    setSyncTaskTrendRange(nextRange);
    refreshSyncTasks({
      currentPage: 1,
      pageSize: 10,
      trendStartDate: nextRange[0],
      trendEndDate: nextRange[1]
    });
  };

  useEffect(() => {
    fetchFilters();
    fetchAssetPermissions();
    fetchCloudAccounts();
    refreshSyncTasks();
    fetchApplications();
    fetchRiskRuleConfig();
    if (isCloudAssets) {
      fetchCoverage();
      fetchGovernanceReport();
    }
  }, [orgId, isCloudAssets]);

  useEffect(() => {
    if (activeTab === 'applications') {
      fetchApplications({ currentPage: 1, pageSize: 10, q: applicationKeyword, risk: applicationRisk });
    }
  }, [activeTab]);

  useEffect(() => {
    if (assetId) {
      setActiveTab('assets');
      setDetailVisible(true);
      fetchDetail(assetId);
    }
  }, [assetId, orgId]);

  useEffect(() => {
    if (syncTaskId) {
      setActiveTab('cloudSync');
      setSyncTaskDetailVisible(true);
      fetchSyncTaskDetail(syncTaskId);
    }
  }, [syncTaskId, orgId]);

  useEffect(() => {
    const nextQuery = {
      syncPolicyId: firstQueryValue(syncPolicyId) || undefined,
      syncPolicyScheduleKey: firstQueryValue(syncPolicyScheduleKey) || undefined,
      syncPolicyScheduleName: firstQueryValue(syncPolicyScheduleName) || undefined
    };
    const nextTrendDays = normalizeTrendDays(firstQueryValue(queryTrendDays));
    const nextTrendRange = normalizeTrendRange(firstQueryValue(queryTrendStartDate), firstQueryValue(queryTrendEndDate));
    const nextFailureThreshold = normalizeFailureThreshold(firstQueryValue(queryFailureThreshold));
    setSyncTaskTrendDays(nextTrendDays);
    setSyncTaskTrendRange(nextTrendRange);
    setSyncTaskFailureThreshold(nextFailureThreshold);
    if (!nextQuery.syncPolicyId && !nextQuery.syncPolicyScheduleKey) {
      return;
    }
    setActiveTab('cloudSync');
    setSyncTaskQuery(nextQuery);
    fetchSyncTasks({
      currentPage: 1,
      pageSize: 10,
      syncPolicyId: nextQuery.syncPolicyId,
      syncPolicyScheduleKey: nextQuery.syncPolicyScheduleKey,
      trendDays: nextTrendDays,
      trendStartDate: nextTrendRange[0],
      trendEndDate: nextTrendRange[1],
      failureThreshold: nextFailureThreshold
    });
  }, [syncPolicyId, syncPolicyScheduleKey, syncPolicyScheduleName, queryTrendDays, queryTrendStartDate, queryTrendEndDate, queryFailureThreshold, orgId]);

  useEffect(() => {
    if (!applicationDetail) {
      return;
    }
    setApplicationRelationForm({
      upstreams: (applicationDetail.upstreams || [])
        .filter((relation) => relation.source === 'manual_application')
        .map((relation) => relation.application),
      downstreams: (applicationDetail.downstreams || [])
        .filter((relation) => relation.source === 'manual_application')
        .map((relation) => relation.application)
    });
  }, [applicationDetail]);

  useEffect(() => {
    if (!detail) {
      return;
    }
    ownershipForm.setFieldsValue({
      owner: detail.owner || '',
      application: detail.application || '',
      businessLine: detail.businessLine || '',
      lifecycle: detail.lifecycle || undefined,
      cost: detail.cost || 0,
      complianceRisk: detail.complianceRisk || undefined
    });
    if (isCloudAssets && detail.id) {
      fetchAssetActions(detail.id);
      setAssetActionDryRun(null);
    }
  }, [detail, ownershipForm]);

  useEffect(() => {
    setRelationFilters({
      keyword: '',
      sources: [],
      types: [],
      direction: 'all'
    });
    setRelationResp(null);
  }, [detail && detail.id]);

  const buildRelationQueryParams = (filters = relationFilters, cursor = undefined) => ({
    limit: relationPageSize,
    cursor,
    includeApplication: true,
    keyword: String(filters.keyword || '').trim() || undefined,
    sources: asCSV(filters.sources || []),
    relationTypes: asCSV(filters.types || []),
    direction: filters.direction || 'all'
  });

  useEffect(() => {
    if (!detailVisible || !(detail && detail.id)) {
      return;
    }
    fetchRelations(detail.id, buildRelationQueryParams());
  }, [
    detailVisible,
    detail && detail.id,
    relationFilters.keyword,
    (relationFilters.sources || []).join(','),
    (relationFilters.types || []).join(','),
    relationFilters.direction,
    orgId
  ]);

  const selectedCloudAccount = useMemo(() => (
    (cloudAccounts || []).find((account) => accountKey(account) === cloudForm.accountKey)
  ), [cloudAccounts, cloudForm.accountKey]);

  const coverageMetrics = coverage.metrics || {};
  const coverageProviders = coverage.providers || [];
  const coverageAccounts = coverage.accounts || [];
  const coverageAssetTypes = coverage.assetTypes || [];
  const kubernetesCoverageAssetTypes = coverageAssetTypes.filter((item) => kubernetesAssetTypes.includes(item.assetType));
  const kubernetesClusterCoverage = kubernetesCoverageAssetTypes.find((item) => item.assetType === 'kubernetes_cluster') || {};
  const kubernetesTotalAssets = kubernetesCoverageAssetTypes.reduce((sum, item) => sum + Number(item.assetCount || 0), 0);
  const kubernetesWorkloadAssets = Math.max(0, kubernetesTotalAssets - Number(kubernetesClusterCoverage.assetCount || 0));
  const kubernetesCloudCollectedAssets = kubernetesCoverageAssetTypes.reduce((sum, item) => sum + Number(item.cloudCollectedAssetCount || 0), 0);
  const kubernetesCloudOnlyAssets = kubernetesCoverageAssetTypes.reduce((sum, item) => sum + Number(item.cloudOnlyAssetCount || 0), 0);
  const kubernetesUnownedAssets = kubernetesCoverageAssetTypes.reduce((sum, item) => sum + Number(item.unownedAssetCount || 0), 0);
  const iacCoverageRate = percent(coverageMetrics.iacCoverageRate);
  const cloudOnlyRate = percent(coverageMetrics.cloudOnlyRate);
  const ownershipCoverageRate = percent(coverageMetrics.ownershipCoverageRate);
  const governanceLifecycleBreakdown = governanceReport.lifecycleBreakdown || [];
  const governanceComplianceRiskBreakdown = governanceReport.complianceRiskBreakdown || [];
  const governanceTopCostAssets = governanceReport.topCostAssets || [];
  const governanceTopRiskAssets = governanceReport.topRiskAssets || [];
  const governanceOwnerGapRate = percent(
    governanceReport.totalAssets ? (Number(governanceReport.missingOwnerAssets || 0) / Number(governanceReport.totalAssets || 1)) * 100 : 0
  );
  const governanceApplicationGapRate = percent(
    governanceReport.totalAssets ? (Number(governanceReport.missingApplicationAssets || 0) / Number(governanceReport.totalAssets || 1)) * 100 : 0
  );
  const riskWindowDays = riskRuleConfig && riskRuleConfig.changeWindowDays || 7;
  const permissionItems = assetPermissions.permissions || [];
  const permissionMap = useMemo(() => {
    const result = {};
    permissionItems.forEach((item) => {
      result[item.key] = item;
    });
    return result;
  }, [permissionItems]);
  const projectRoleMap = useMemo(() => {
    const result = {};
    (assetPermissions.projectRoles || []).forEach((item) => {
      result[item.projectId] = item.role;
    });
    return result;
  }, [assetPermissions.projectRoles]);
  const permissionMessage = (key) => permissionMap[key] && permissionMap[key].message || '当前账号没有执行该操作的权限';
  const notifyNoPermission = (key, message = '无权限') => {
    notification.warning({
      message,
      description: permissionMessage(key)
    });
  };
  const roleAllows = (role, roles) => roles.includes(role);
  const canManageAllAssets = !!assetPermissions.canManageAll;
  const canExportAssets = !!assetPermissions.canExport;
  const canImportAssets = !!assetPermissions.canImport;
  const canBatchGovernance = !!assetPermissions.canBatchGovernance;
  const canManageApplicationRelations = !!assetPermissions.canManageApplicationRelations;
  const canManageRiskRules = !!assetPermissions.canManageRiskRules;
  const canSyncIac = !!assetPermissions.canSyncIac;
  const canEditDetailOwnership = !!(detail && (
    canManageAllAssets ||
    (detail.projectId && roleAllows(projectRoleMap[detail.projectId], ['manager', 'approver', 'operator']))
  ));
  const applicationList = (applicationsData && applicationsData.list) || [];
  const syncTaskList = (syncTasksData && syncTasksData.list) || [];
  const syncTaskSummary = (syncTasksData && syncTasksData.summary) || {};
  const syncTaskTrend = Array.isArray(syncTaskSummary.trend) ? syncTaskSummary.trend : [];
  const syncTaskRegionBreakdown = Array.isArray(syncTaskSummary.regions) ? syncTaskSummary.regions : [];
  const syncTaskAssetTypeBreakdown = Array.isArray(syncTaskSummary.assetTypes) ? syncTaskSummary.assetTypes : [];
  const syncTaskApiMetricBreakdown = Array.isArray(syncTaskSummary.apiMetrics) ? syncTaskSummary.apiMetrics : [];
  const syncTaskDetailScope = syncTaskScopeSummary(syncTaskDetail || {});
  const syncTaskDetailScopeMetrics = (syncTaskDetailScope.scopeMetrics || []).length
    ? syncTaskDetailScope.scopeMetrics
    : [...(syncTaskDetailScope.regionMetrics || []), ...(syncTaskDetailScope.assetTypeMetrics || [])];
  const syncTaskDetailFailure = syncTaskFailureSummary(syncTaskDetail || {});
  const syncTaskDetailFailureDetails = syncTaskDetailFailure.details || [];
  const syncTaskDetailApiMetrics = flattenSyncTaskApiMetrics((syncTaskDetail || {}).stats || {});
  const syncTaskDetailApiSummary = ((syncTaskDetail || {}).stats || {}).apiMetricSummary || {};
  const syncTaskApiMetricFilterOptions = useMemo(() => ({
    regions: uniqueList(syncTaskDetailApiMetrics.map((item) => item.region).filter(Boolean)),
    services: uniqueList(syncTaskDetailApiMetrics.map((item) => item.service).filter(Boolean)),
    statuses: uniqueList(syncTaskDetailApiMetrics.map((item) => item.status).filter(Boolean))
  }), [syncTaskDetailApiMetrics]);
  const filteredSyncTaskDetailApiMetrics = useMemo(() => {
    const regions = syncTaskApiMetricFilters.regions || [];
    const services = syncTaskApiMetricFilters.services || [];
    const statuses = syncTaskApiMetricFilters.statuses || [];
    const slowOnly = !!syncTaskApiMetricFilters.slowOnly;
    const slowThresholdMs = Math.max(1, Number(syncTaskApiMetricFilters.slowThresholdMs || defaultSyncTaskApiMetricFilters.slowThresholdMs));
    return syncTaskDetailApiMetrics.filter((item) => {
      if (regions.length && !regions.includes(item.region)) {
        return false;
      }
      if (services.length && !services.includes(item.service)) {
        return false;
      }
      if (statuses.length && !statuses.includes(item.status)) {
        return false;
      }
      if (slowOnly && Number(item.durationMs || 0) < slowThresholdMs) {
        return false;
      }
      return true;
    });
  }, [syncTaskApiMetricFilters, syncTaskDetailApiMetrics]);
  const syncTaskApiSlowSummary = useMemo(() => {
    const slowThresholdMs = Math.max(1, Number(syncTaskApiMetricFilters.slowThresholdMs || defaultSyncTaskApiMetricFilters.slowThresholdMs));
    const slowMetrics = syncTaskDetailApiMetrics.filter((item) => Number(item.durationMs || 0) >= slowThresholdMs);
    const slowest = slowMetrics.reduce((current, item) => (
      Number(item.durationMs || 0) > Number((current || {}).durationMs || 0) ? item : current
    ), null);
    return {
      thresholdMs: slowThresholdMs,
      total: slowMetrics.length,
      failed: slowMetrics.filter((item) => item.status === 'failed').length,
      retried: slowMetrics.filter((item) => Number(item.retryCount || 0) > 0 || Number(item.attempts || 0) > 1).length,
      slowest
    };
  }, [syncTaskApiMetricFilters.slowThresholdMs, syncTaskDetailApiMetrics]);
  const syncTaskTrendMax = Math.max(1, ...syncTaskTrend.map((point) => Number(point.totalCount || 0)));
  const showSyncTaskSummary = Boolean(syncTaskQuery.syncPolicyId || syncTaskQuery.syncPolicyScheduleKey);
  const syncTaskTrendRangeLabel = syncTaskSummary.trendCustomRange
    ? `${syncTaskSummary.trendStartDate} 至 ${syncTaskSummary.trendEndDate}`
    : `${syncTaskSummary.trendDays || syncTaskTrendDays}天`;
  const syncTaskTrendTitle = syncTaskSummary.trendCustomRange
    ? `趋势：${syncTaskTrendRangeLabel}`
    : `近${syncTaskSummary.trendDays || syncTaskTrendDays}日趋势`;
  const selectedFailedSyncTasks = syncTaskList.filter((task) => (
    selectedFailedSyncTaskIds.includes(task.id) && task.status === 'failed'
  ));
  const selectedFailedSyncTaskRegions = uniqueList(selectedFailedSyncTasks.reduce((acc, task) => acc.concat(task.regions || []), []));
  const selectedFailedSyncTaskAssetTypes = uniqueList(selectedFailedSyncTasks.reduce((acc, task) => acc.concat(task.assetTypes || []), []));
  const relationRespForDetail = relationResp && detail && relationResp.assetId === detail.id ? relationResp : null;
  const detailBaseRelations = detail && detail.relations || [];
  const detailRelations = relationRespForDetail ? (relationRespForDetail.relations || []) : detailBaseRelations;
  const activeRelationSummary = relationRespForDetail ? relationRespForDetail.summary : detail && detail.relationSummary;
  const relationSourceOptions = useMemo(() => (
    uniqueList([...(detailBaseRelations || []), ...(detailRelations || [])].map((relation) => relation.source))
  ), [detailBaseRelations, detailRelations]);
  const relationTypeOptions = useMemo(() => (
    uniqueList([...(detailBaseRelations || []), ...(detailRelations || [])].map((relation) => relation.relationType))
  ), [detailBaseRelations, detailRelations]);
  const filteredDetailRelations = useMemo(() => {
    const keyword = String(relationFilters.keyword || '').trim().toLowerCase();
    const sourceSet = (relationFilters.sources || []).reduce((acc, source) => ({ ...acc, [source]: true }), {});
    const typeSet = (relationFilters.types || []).reduce((acc, type) => ({ ...acc, [type]: true }), {});
    return detailRelations.filter((relation) => {
      if (keyword && !relationSearchText(relation).includes(keyword)) {
        return false;
      }
      if ((relationFilters.sources || []).length && !sourceSet[relation.source]) {
        return false;
      }
      if ((relationFilters.types || []).length && !typeSet[relation.relationType]) {
        return false;
      }
      if (relationFilters.direction && relationFilters.direction !== 'all' && relationDirection(relation, detail) !== relationFilters.direction) {
        return false;
      }
      return true;
    });
  }, [detail, relationFilters]);
  const relationFilterActive = Boolean(
    relationFilters.keyword ||
    (relationFilters.sources || []).length ||
    (relationFilters.types || []).length ||
    (relationFilters.direction && relationFilters.direction !== 'all')
  );
  const loadedRelationCount = detailRelations.length;
  const relationHasMore = Boolean(activeRelationSummary && activeRelationSummary.hasMore);
  const syncTaskRerunGroupApproval = syncTaskRerunGroup && syncTaskRerunGroup.approval || {};
  const canApproveSyncTaskRerunGroup = Boolean(
    canSyncIac &&
    syncTaskRerunGroup &&
    syncTaskRerunGroup.groupId &&
    Number(syncTaskRerunGroup.approvingCount || 0) > 0 &&
    syncTaskRerunGroupApproval.status === 'pending'
  );
  const exportSyncTaskTrend = () => {
    if (!syncTaskTrend.length) {
      notification.info({ message: '暂无可导出的趋势数据' });
      return;
    }
    const trendDays = syncTaskSummary.trendDays || syncTaskTrendDays;
    const failureThreshold = syncTaskSummary.failureThreshold || syncTaskFailureThreshold;
    const trendRangeLabel = syncTaskSummary.trendCustomRange
      ? `${syncTaskSummary.trendStartDate} 至 ${syncTaskSummary.trendEndDate}`
      : `${trendDays}天`;
    const rows = [
      ['日期', '完成任务', '失败任务', '总任务', '当日成功率', '当日失败率', '统计范围', '失败阈值', '策略ID', '子周期'],
      ...syncTaskTrend.map((point) => {
        const completeCount = Number(point.completeCount || 0);
        const failedCount = Number(point.failedCount || 0);
        const totalCount = Number(point.totalCount || 0);
        return [
          point.date || '',
          completeCount,
          failedCount,
          totalCount,
          csvRateText(completeCount, totalCount),
          csvRateText(failedCount, totalCount),
          trendRangeLabel,
          percentText(failureThreshold),
          syncTaskQuery.syncPolicyId || '',
          syncTaskQuery.syncPolicyScheduleKey || ''
        ];
      })
    ];
    const csv = `\uFEFF${rows.map((row) => row.map(csvCell).join(',')).join('\n')}\n`;
    const rangePart = syncTaskSummary.trendCustomRange
      ? `${safeFilenamePart(syncTaskSummary.trendStartDate)}_${safeFilenamePart(syncTaskSummary.trendEndDate)}`
      : `${trendDays}d`;
    const filename = `cloud-sync-trend-${safeFilenamePart(syncTaskQuery.syncPolicyId)}-${safeFilenamePart(syncTaskQuery.syncPolicyScheduleKey)}-${rangePart}.csv`;
    downloadTextFile(filename, csv, 'text/csv;charset=utf-8;');
  };
  const exportSyncTaskApiMetrics = () => {
    if (!filteredSyncTaskDetailApiMetrics.length) {
      notification.info({ message: '暂无可导出的 API 调用指标' });
      return;
    }
    const task = syncTaskDetail || {};
    const rows = [
      ['任务ID', '账号', '云厂商', '区域', '服务', 'API', '状态', 'HTTP状态', '耗时(ms)', '尝试次数', '重试次数', 'Compartment', '分页请求', '错误分类', '可重试', '重试建议', '错误', 'Request ID', 'Retry-After'],
      ...filteredSyncTaskDetailApiMetrics.map((item) => [
        task.id || '',
        task.accountName || '',
        task.provider || '',
        item.region || '',
        item.service || '',
        item.path || '',
        syncTaskMetricStatusMap[item.status] || item.status || '',
        item.httpStatus || '',
        item.durationMs || 0,
        item.attempts || 0,
        item.retryCount || 0,
        item.compartmentId || '',
        item.pageTokenUsed ? '是' : '否',
        syncTaskFailureCategoryMap[item.errorCategory] || item.errorCategory || '',
        item.retryable === undefined ? '' : item.retryable ? '是' : '否',
        item.retryHint || '',
        item.error || '',
        item.requestId || '',
        item.retryAfter || ''
      ])
    ];
    const csv = `\uFEFF${rows.map((row) => row.map(csvCell).join(',')).join('\n')}\n`;
    const filename = `cloud-sync-api-metrics-${safeFilenamePart(task.id)}.csv`;
    downloadTextFile(filename, csv, 'text/csv;charset=utf-8;');
  };
  const syncTaskHistorySeed = useMemo(() => {
    if (!syncTaskQuery.syncPolicyId || !syncTaskQuery.syncPolicyScheduleKey) {
      return null;
    }
    return syncTaskList.find((task) => (
      task.syncPolicyId === syncTaskQuery.syncPolicyId &&
      (task.stats || {}).syncPolicyScheduleKey === syncTaskQuery.syncPolicyScheduleKey
    )) || null;
  }, [syncTaskList, syncTaskQuery]);
  const applicationOptions = useMemo(() => {
    const names = {};
    applicationList.forEach((app) => {
      if (app.application) {
        names[app.application] = true;
      }
    });
    if (applicationDetail && applicationDetail.application) {
      names[applicationDetail.application] = true;
    }
    [
      ...(applicationDetail && applicationDetail.upstreams || []),
      ...(applicationDetail && applicationDetail.downstreams || [])
    ].forEach((relation) => {
      if (relation.application) {
        names[relation.application] = true;
      }
    });
    return Object.keys(names).sort();
  }, [applicationList, applicationDetail]);
  const hasRunningSyncTask = useMemo(() => (
    syncTaskList.some((task) => ['approving', 'pending', 'running'].includes(task.status))
  ), [syncTasksData]);

  const openDetail = (id) => {
    setDetailVisible(true);
    setRelationFilters({
      keyword: '',
      sources: [],
      types: [],
      direction: 'all'
    });
    setRelationResp(null);
    fetchDetail(id);
    if (isCloudAssets) {
      fetchSecurityRules(id);
    }
  };

  const closeDetail = () => {
    setDetailVisible(false);
    setAssetActionDryRun(null);
    setActionRequestKey('');
    setCreatingActionKey('');
    setAssetActionCreateVisible(false);
    setAssetActionCreateRecord(null);
    setAssetActionCreateReason('');
    setAssetActionConfirmAction('');
    setAssetActionConfirmResourceId('');
    setAssetActionTagsText('');
    setRelationResp(null);
  };

  const openApplicationDetail = (application, tab = 'relations') => {
    setApplicationDetailTab(tab);
    setApplicationDetailVisible(true);
    fetchApplicationDetail(application);
  };

  const closeApplicationDetail = () => {
    setApplicationDetailVisible(false);
    setApplicationDetailTab('relations');
  };

  const openSyncTaskDetail = (record) => {
    setSyncTaskApiMetricFilters({
      ...defaultSyncTaskApiMetricFilters,
      slowThresholdMs: syncTaskSlowApiThresholdMs(record)
    });
    setSyncTaskDetailVisible(true);
    fetchSyncTaskDetail(record.id);
  };

  const closeSyncTaskDetail = () => {
    setSyncTaskApiMetricFilters({ ...defaultSyncTaskApiMetricFilters });
    setSyncTaskDetailVisible(false);
  };

  const openSyncTaskRerunGroup = (groupId) => {
    if (!groupId) {
      return;
    }
    setSyncTaskRerunGroupVisible(true);
    fetchSyncTaskRerunGroup(groupId);
  };

  const closeSyncTaskRerunGroup = () => {
    setSyncTaskRerunGroupVisible(false);
  };

  const onApproveSyncTaskRerunGroup = (action) => {
    if (!syncTaskRerunGroup || !syncTaskRerunGroup.groupId) {
      return;
    }
    const approved = action === 'approved';
    Modal.confirm({
      title: approved ? '确认审批通过该批量重跑任务组？' : '确认驳回该批量重跑任务组？',
      content: approved ? '审批通过后，任务组内采集任务会开始后台执行。' : '驳回后，任务组内采集任务会进入已驳回终态。',
      okText: approved ? '审批通过' : '确认驳回',
      cancelText: '取消',
      onOk: async () => {
        setSyncTaskRerunGroupApproving(true);
        try {
          const res = await assetAPI.approveSyncTaskRerunGroup({
            orgId,
            groupId: syncTaskRerunGroup.groupId,
            action
          });
          if (res.code !== 200) {
            throw new Error(res.message || '审批失败');
          }
          notification.success({ message: approved ? '审批已通过' : '审批已驳回' });
          fetchSyncTaskRerunGroup(syncTaskRerunGroup.groupId);
          refreshSyncTasks();
        } catch (err) {
          notification.error({
            message: approved ? '审批通过失败' : '审批驳回失败',
            description: err.message
          });
        } finally {
          setSyncTaskRerunGroupApproving(false);
        }
      }
    });
  };

  const openBatchOwnership = () => {
    if (!canBatchGovernance) {
      notifyNoPermission('batch_governance');
      return;
    }
    if (!selectedAssetIds.length) {
      notification.warning({ message: '请先选择资产' });
      return;
    }
    setBatchOwnershipVisible(true);
  };

  const closeBatchOwnership = () => {
    setBatchOwnershipVisible(false);
    batchOwnershipForm.resetFields();
  };

  const onSearch = (_, keyword) => {
    setSearchParams((preSearchParams) => ({
      ...preSearchParams,
      form: { ...preSearchParams.form, keyword },
      paginate: { ...preSearchParams.paginate, current: 1 }
    }));
  };

  const onFilterChange = (key, value) => {
    onChangeFormParams({ [key]: value });
  };

  const applyAssetFilters = (filters) => {
    setActiveTab('assets');
    setSearchParams((preSearchParams) => ({
      ...preSearchParams,
      form: {
        ...preSearchParams.form,
        ...filters
      },
      paginate: { ...preSearchParams.paginate, current: 1 }
    }));
  };

  const updateRelationFilter = (key, value) => {
    setRelationResp(null);
    setRelationFilters((pre) => ({
      ...pre,
      [key]: value
    }));
  };

  const resetRelationFilters = () => {
    setRelationResp(null);
    setRelationFilters({
      keyword: '',
      sources: [],
      types: [],
      direction: 'all'
    });
  };

  const loadMoreRelations = () => {
    if (!detail || !detail.id || !relationHasMore) {
      return;
    }
    const nextCursor = activeRelationSummary && activeRelationSummary.nextCursor || String(loadedRelationCount || 0);
    fetchRelations(detail.id, buildRelationQueryParams(relationFilters, nextCursor));
  };

  const buildAssetQueryParams = (options = {}) => {
    const { paginate, form } = searchParams;
    return {
      currentPage: options.currentPage || paginate.current,
      pageSize: options.pageSize || paginate.pageSize,
      q: form.keyword,
      sortField: searchParams.sorter.field,
      sortOrder: searchParams.sorter.order,
      projectIds: asCSV(form.projectIds),
      envIds: asCSV(form.envIds),
      providers: asCSV(form.providers),
      accountIds: asCSV(form.accountIds),
      assetTypes: asCSV(form.assetTypes),
      sources: asCSV(form.sources),
      statuses: asCSV(form.statuses),
      managedBy: asCSV(form.managedBy),
      dsl: form.dsl
    };
  };

  const refresh = () => {
    const queryParams = buildAssetQueryParams();
    fetchList(queryParams);
    fetchFilters(queryParams);
    if (isCloudAssets) {
      fetchCoverage();
      fetchGovernanceReport();
    }
  };

  const refreshApplications = (options = {}) => {
    fetchApplications({
      currentPage: options.currentPage || 1,
      pageSize: options.pageSize || 10,
      q: options.q !== undefined ? options.q : applicationKeyword,
      risk: options.risk !== undefined ? options.risk : applicationRisk
    });
  };

  const openRiskRuleConfig = () => {
    if (!canManageRiskRules) {
      notifyNoPermission('risk_rules');
      return;
    }
    setRiskRuleVisible(true);
    riskRuleForm.setFieldsValue(riskRuleConfig || {});
    fetchRiskRuleConfig();
  };

  const saveRiskRuleConfig = async () => {
    if (!canManageRiskRules) {
      notifyNoPermission('risk_rules');
      return;
    }
    try {
      const values = await riskRuleForm.validateFields();
      await updateRiskRuleConfig(values);
    } catch (err) {
      if (err && err.errorFields) {
        return;
      }
      notification.error({
        message: '风险规则保存失败',
        description: err.message
      });
    }
  };

  const exportAssets = async (format) => {
    if (!canExportAssets) {
      notifyNoPermission('export');
      return;
    }
    const queryParams = buildAssetQueryParams({ currentPage: undefined, pageSize: undefined });
    const params = [];
    appendQueryParam(params, 'format', format);
    appendQueryParam(params, 'q', queryParams.q);
    appendQueryParam(params, 'projectIds', queryParams.projectIds);
    appendQueryParam(params, 'envIds', queryParams.envIds);
    appendQueryParam(params, 'providers', queryParams.providers);
    appendQueryParam(params, 'accountIds', queryParams.accountIds);
    appendQueryParam(params, 'assetTypes', queryParams.assetTypes);
    appendQueryParam(params, 'sources', queryParams.sources);
    appendQueryParam(params, 'statuses', queryParams.statuses);
    appendQueryParam(params, 'managedBy', queryParams.managedBy);
    appendQueryParam(params, 'dsl', queryParams.dsl);
    selectedAssetIds.forEach((id) => appendQueryParam(params, 'ids', id));

    try {
      await downloadImportTemplate(`${exportAssetsPath}?${params.join('&')}`, { orgId });
      if (selectedAssetIds.length) {
        setSelectedAssetIds([]);
      }
    } catch (err) {
      notification.error({
        message: '导出失败',
        description: err.message
      });
    }
  };

  const downloadAssetImportTemplate = async () => {
    try {
      await downloadImportTemplate(importTemplatePath, { orgId });
    } catch (err) {
      notification.error({
        message: '模板下载失败',
        description: err.message
      });
    }
  };

  const onImportAssetsFile = async (file) => {
    if (!canImportAssets) {
      notifyNoPermission('import');
      return false;
    }
    try {
      const content = await readTextFile(file);
      const payload = JSON.parse(content);
      const assets = normalizeImportAssets(payload);
      if (!assets.length) {
        notification.error({ message: '导入失败', description: '文件中未找到 assets 数据' });
        return false;
      }
      const importPayload = {
        assets,
        overwriteOwnership: importOverwriteOwnership
      };
      setImportPreviewPayload(importPayload);
      await previewImportAssets({
        ...importPayload,
        dryRun: true
      });
    } catch (err) {
      notification.error({
        message: '导入预检失败',
        description: err.message
      });
    }
    return false;
  };

  const confirmImportAssets = async () => {
    if (!canImportAssets) {
      notifyNoPermission('import');
      return;
    }
    if (!importPreviewPayload) {
      return;
    }
    await importAssets(importPreviewPayload);
  };

  const closeImportPreview = () => {
    setImportPreviewVisible(false);
    setImportPreview(null);
    setImportPreviewPayload(null);
  };

  const onDslSearch = (dsl) => {
    setSearchParams((preSearchParams) => ({
      ...preSearchParams,
      form: { ...preSearchParams.form, dsl: dsl && dsl.trim() },
      paginate: { ...preSearchParams.paginate, current: 1 }
    }));
  };

  const showApplicationAssets = (application) => {
    const dsl = applicationAssetDsl(application);
    setApplicationDetailVisible(false);
    setDslInput(dsl);
    setActiveTab('assets');
    setSearchParams((preSearchParams) => ({
      ...preSearchParams,
      form: { ...preSearchParams.form, dsl },
      paginate: { ...preSearchParams.paginate, current: 1 }
    }));
  };

  useEffect(() => {
    if (!hasRunningSyncTask) {
      return undefined;
    }
    const timer = setInterval(() => {
      refreshSyncTasks();
      refresh();
      if (isCloudAssets) {
        fetchCoverage();
        fetchGovernanceReport();
      }
    }, 3000);
    return () => clearInterval(timer);
  }, [hasRunningSyncTask, searchParams, isCloudAssets, syncTaskQuery, syncTaskTrendDays, syncTaskTrendRange, syncTaskFailureThreshold]);

  const syncIac = async () => {
    if (!canSyncIac) {
      notifyNoPermission('sync_iac');
      return;
    }
    try {
      await backfillIacResources();
    } catch (err) {
      notification.error({
        message: '同步失败',
        description: err.message
      });
    }
  };

  const onCloudAccountChange = (value) => {
    const account = (cloudAccounts || []).find((it) => accountKey(it) === value);
    setCloudForm({
      accountKey: value,
      regions: account && account.regions || [],
      assetTypes: account && account.supportedAssetTypes || []
    });
  };

  const onStartCloudSync = async () => {
    if (!canSyncIac) {
      notifyNoPermission('sync_iac');
      return;
    }
    if (!selectedCloudAccount) {
      notification.error({ message: '请选择云账号' });
      return;
    }
    try {
      await startCloudSync({
        accountSource: selectedCloudAccount.source,
        accountId: selectedCloudAccount.id,
        provider: selectedCloudAccount.provider,
        regions: cloudForm.regions,
        assetTypes: cloudForm.assetTypes
      });
    } catch (err) {
      notification.error({
        message: '采集任务启动失败',
        description: err.message
      });
    }
  };

  const openSyncTaskRerunReason = (mode) => {
    if (mode === 'schedule' && !syncTaskHistorySeed) {
      notification.warning({ message: '当前子周期暂无可重跑的历史任务' });
      return;
    }
    if (mode === 'schedule' && !canSyncIac) {
      notifyNoPermission('sync_iac');
      return;
    }
    if (mode === 'failed' && !selectedFailedSyncTasks.length) {
      notification.warning({ message: '请选择需要重跑的失败任务' });
      return;
    }
    setSyncTaskRerunMode(mode);
    setSyncTaskRerunReason('');
    setSyncTaskRerunRequiresApproval(mode === 'failed' && !canSyncIac);
    setSyncTaskRerunRegions([]);
    setSyncTaskRerunAssetTypes([]);
    setSyncTaskRerunReasonVisible(true);
  };

  const closeSyncTaskRerunReason = () => {
    setSyncTaskRerunReasonVisible(false);
    setSyncTaskRerunMode('');
    setSyncTaskRerunReason('');
    setSyncTaskRerunRequiresApproval(false);
    setSyncTaskRerunRegions([]);
    setSyncTaskRerunAssetTypes([]);
  };

  const onRerunSyncTaskHistory = () => openSyncTaskRerunReason('schedule');

  const onBatchRerunFailedSyncTasks = () => openSyncTaskRerunReason('failed');

  const onSubmitSyncTaskRerunReason = async () => {
    const reason = syncTaskRerunReason.trim();
    if (!reason) {
      notification.warning({ message: '请输入重跑原因' });
      return;
    }
    setSyncTaskRerunSubmitting(true);
    try {
      if (syncTaskRerunMode === 'schedule') {
        if (!canSyncIac) {
          notifyNoPermission('sync_iac');
          return;
        }
        await startCloudSync({
          accountSource: syncTaskHistorySeed.accountSource,
          accountId: syncTaskHistorySeed.accountId,
          provider: syncTaskHistorySeed.provider,
          regions: syncTaskHistorySeed.regions || [],
          assetTypes: syncTaskHistorySeed.assetTypes || [],
          syncPolicyId: syncTaskQuery.syncPolicyId,
          syncPolicyScheduleKey: syncTaskQuery.syncPolicyScheduleKey,
          syncPolicyScheduleName: syncTaskQuery.syncPolicyScheduleName,
          reason
        });
        closeSyncTaskRerunReason();
        return;
      }

      if (!canSyncIac && !syncTaskRerunRequiresApproval) {
        notifyNoPermission('sync_iac');
        return;
      }
      setBatchRerunningSyncTasks(true);
      const rerunResp = await batchRerunFailedSyncTasks({
        taskIds: selectedFailedSyncTasks.map((task) => task.id),
        reason,
        regions: syncTaskRerunRegions,
        assetTypes: syncTaskRerunAssetTypes,
        requiresApproval: syncTaskRerunRequiresApproval
      });
      const errors = rerunResp && rerunResp.errors || [];
      if (errors.length) {
        notification.warning({
          message: '部分失败任务重跑未启动',
          description: errors.slice(0, 3).map((item) => `${item.taskId || '-'}: ${item.message}`).join('；')
        });
      } else {
        notification.success({
          message: syncTaskRerunRequiresApproval
            ? `已提交 ${rerunResp && rerunResp.created || selectedFailedSyncTasks.length} 个失败任务重跑审批`
            : `已启动 ${rerunResp && rerunResp.created || selectedFailedSyncTasks.length} 个失败任务重跑`,
          description: rerunResp && rerunResp.groupId ? (
            <Space size={4}>
              <Text type='secondary'>任务组</Text>
              <Button type='link' size='small' onClick={() => openSyncTaskRerunGroup(rerunResp.groupId)}>
                {rerunResp.groupId}
              </Button>
            </Space>
          ) : undefined
        });
        setSelectedFailedSyncTaskIds([]);
        closeSyncTaskRerunReason();
      }
      refreshSyncTasks();
    } catch (err) {
      notification.error({
        message: syncTaskRerunMode === 'schedule' ? '子周期重跑失败' : '失败任务重跑失败',
        description: err.message
      });
    } finally {
      setBatchRerunningSyncTasks(false);
      setSyncTaskRerunSubmitting(false);
    }
  };

  const onDryRunAssetAction = async (record) => {
    if (!detail || !record || !record.key) {
      return;
    }
    if (assetActionNeedsParams(record.key)) {
      setAssetActionDryRunRecord(record);
      setAssetActionDryRunParamsText(JSON.stringify(defaultAssetActionParams(record.key, detail), null, 2));
      setAssetActionDryRunVisible(true);
      return;
    }
    setActionRequestKey(record.key);
    try {
      await dryRunAssetAction({ action: record.key });
    } catch (err) {
      notification.error({
        message: '预检查失败',
        description: err.message
      });
    } finally {
      setActionRequestKey('');
    }
  };

  const closeAssetActionDryRun = () => {
    setAssetActionDryRunVisible(false);
    setAssetActionDryRunRecord(null);
    setAssetActionDryRunParamsText('');
  };

  const onSubmitAssetActionDryRun = async () => {
    if (!detail || !assetActionDryRunRecord) {
      return;
    }
    let params = {};
    try {
      params = JSON.parse(assetActionDryRunParamsText || '{}');
    } catch (err) {
      notification.warning({ message: '参数 JSON 格式不正确' });
      return;
    }
    const paramsError = validateAssetActionParams(assetActionDryRunRecord.key, params, detail);
    if (paramsError) {
      notification.warning({ message: paramsError });
      return;
    }
    setActionRequestKey(assetActionDryRunRecord.key);
    try {
      await dryRunAssetAction({
        action: assetActionDryRunRecord.key,
        params
      });
      closeAssetActionDryRun();
    } catch (err) {
      notification.error({
        message: '预检查失败',
        description: err.message
      });
    } finally {
      setActionRequestKey('');
    }
  };

  const onCreateAssetAction = (record) => {
    if (!detail || !record || !record.key) {
      return;
    }
    setAssetActionCreateRecord(record);
    setAssetActionCreateReason(record.adapterMode === 'provider' ? '' : '来自资产详情页');
    setAssetActionConfirmAction('');
    setAssetActionConfirmResourceId('');
    setAssetActionTagsText(record.key === 'update_tags' ? JSON.stringify(detail && detail.tags || {}, null, 2) : '');
    setAssetActionParamsText(assetActionNeedsParams(record.key) ? JSON.stringify(defaultAssetActionParams(record.key, detail), null, 2) : '');
    setAssetActionCreateVisible(true);
  };

  const closeAssetActionCreate = () => {
    setAssetActionCreateVisible(false);
    setAssetActionCreateRecord(null);
    setAssetActionCreateReason('');
    setAssetActionConfirmAction('');
    setAssetActionConfirmResourceId('');
    setAssetActionTagsText('');
    setAssetActionParamsText('');
  };

  const onSubmitAssetActionCreate = async () => {
    if (!detail || !assetActionCreateRecord) {
      return;
    }
    const record = assetActionCreateRecord;
    const requiresConfirm = record.adapterMode === 'provider';
    const reason = (assetActionCreateReason || '').trim();
    const expectedResourceId = detail.nativeId || detail.id || '';
    const confirmAction = (assetActionConfirmAction || '').trim();
    const confirmResourceId = (assetActionConfirmResourceId || '').trim();
    let tags;
    let params;
    if (record.key === 'update_tags') {
      try {
        tags = JSON.parse(assetActionTagsText || '{}');
      } catch (err) {
        notification.warning({ message: '标签 JSON 格式不正确' });
        return;
      }
      if (!tags || Array.isArray(tags) || typeof tags !== 'object' || Object.keys(tags).length === 0) {
        notification.warning({ message: '请填写至少一个标签' });
        return;
      }
    }
    if (assetActionNeedsParams(record.key)) {
      try {
        params = JSON.parse(assetActionParamsText || '{}');
      } catch (err) {
        notification.warning({ message: '参数 JSON 格式不正确' });
        return;
      }
      const paramsError = validateAssetActionParams(record.key, params, detail);
      if (paramsError) {
        notification.warning({ message: paramsError });
        return;
      }
    }
    if (requiresConfirm) {
      if (reason.length < 6) {
        notification.warning({ message: '请填写不少于 6 个字符的操作原因' });
        return;
      }
      if (confirmAction !== record.key) {
        notification.warning({ message: `请确认动作 ${record.key}` });
        return;
      }
      if (confirmResourceId !== expectedResourceId) {
        notification.warning({ message: `请确认资源 ID ${expectedResourceId}` });
        return;
      }
    }

    setCreatingActionKey(record.key);
    try {
      await submitAssetAction({
        action: record.key,
        reason: reason || '来自资产详情页',
        params,
        tags,
        confirmAction,
        confirmResourceId
      });
      closeAssetActionCreate();
    } catch (err) {
      notification.error({
        message: '创建任务失败',
        description: err.message
      });
    } finally {
      setCreatingActionKey('');
    }
  };

  const onSaveOwnership = async () => {
    if (!detail) {
      return;
    }
    if (!canEditDetailOwnership) {
      notifyNoPermission('edit_ownership');
      return;
    }
    try {
      const values = await ownershipForm.validateFields();
      updateAssetOwnership({
        owner: values.owner || '',
        application: values.application || '',
        businessLine: values.businessLine || '',
        lifecycle: values.lifecycle || '',
        cost: Number(values.cost) || 0,
        complianceRisk: values.complianceRisk || ''
      });
    } catch (err) {
      if (err && err.errorFields) {
        return;
      }
      notification.error({
        message: '归属保存失败',
        description: err.message
      });
    }
  };

  const onSaveBatchOwnership = async () => {
    if (!canBatchGovernance) {
      notifyNoPermission('batch_governance');
      return;
    }
    try {
      const values = await batchOwnershipForm.validateFields();
      const payload = { ids: selectedAssetIds };
      if (values.clearProjectEnv) {
        payload.projectId = '';
        payload.envId = '';
      } else {
        ['projectId', 'envId'].forEach((key) => {
          if (values[key] !== undefined && values[key] !== '') {
            payload[key] = values[key];
          }
        });
      }
      ['owner', 'application', 'businessLine', 'lifecycle', 'complianceRisk'].forEach((key) => {
        if (values[key] !== undefined && values[key] !== '') {
          payload[key] = values[key];
        }
      });
      if (Object.keys(payload).length <= 1) {
        notification.warning({ message: '请至少填写一个治理字段' });
        return;
      }
      await batchUpdateAssetOwnership(payload);
    } catch (err) {
      if (err && err.errorFields) {
        return;
      }
      notification.error({
        message: '批量治理保存失败',
        description: err.message
      });
    }
  };

  const onSaveApplicationRelations = async () => {
    if (!applicationDetail) {
      return;
    }
    if (!canManageApplicationRelations) {
      notifyNoPermission('application_relations');
      return;
    }
    try {
      await updateApplicationRelations({
        application: applicationDetail.application,
        upstreams: applicationRelationForm.upstreams || [],
        downstreams: applicationRelationForm.downstreams || []
      });
    } catch (err) {
      notification.error({
        message: '应用依赖保存失败',
        description: err.message
      });
    }
  };

  const columns = useMemo(() => [
    {
      dataIndex: 'projectName',
      title: '项目',
      width: 150,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'envName',
      title: '环境',
      width: 160,
      render: (text, record) => {
        const { projectId, envId } = record;
        if (!projectId || !envId) {
          return text || '-';
        }
        const url = `/org/${orgId}/project/${projectId}/m-project-env/detail/${envId}?tabKey=resource`;
        return (
          <Link to={url}>
            <EllipsisText>{text}</EllipsisText>
          </Link>
        );
      }
    },
    {
      dataIndex: 'provider',
      title: '云厂商',
      width: 120,
      render: (text) => text ? <Tag color='blue'>{text}</Tag> : '-'
    },
    {
      dataIndex: 'accountId',
      title: '账号',
      width: 190,
      ellipsis: true,
      render: (text, record) => (
        <Space direction='vertical' size={0}>
          <span>{text || '-'}</span>
          {record.cloudAccountId && <Text type='secondary'>云账号 {record.cloudAccountId}</Text>}
        </Space>
      )
    },
    {
      title: '纳管状态',
      width: 130,
      render: (_, record) => {
        const managedBy = assetManagedBy(record);
        const color = managedBy === 'cloud_only' ? 'warning' : managedBy === 'iac' ? 'success' : 'processing';
        return <Tag color={color}>{managedByMap[managedBy] || managedBy}</Tag>;
      }
    },
    {
      dataIndex: 'assetType',
      title: '资产类型',
      width: 160,
      render: (text) => assetTypeMap[text] || text || '-'
    },
    {
      dataIndex: 'name',
      title: '名称',
      width: 220,
      render: (text, record) => (
        <a onClick={() => openDetail(record.id)}>
          <EllipsisText>{text || record.nativeId || '-'}</EllipsisText>
        </a>
      )
    },
    {
      dataIndex: 'nativeType',
      title: '原生类型',
      width: 210,
      ellipsis: true
    },
    {
      dataIndex: 'nativeId',
      title: '资源ID',
      width: 220,
      ellipsis: true
    },
    {
      dataIndex: 'status',
      title: '状态',
      width: 110,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'source',
      title: '来源',
      width: 120,
      render: (text) => sourceMap[text] || text || '-'
    },
    {
      dataIndex: 'owner',
      title: '负责人',
      width: 140,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'application',
      title: '应用',
      width: 150,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'complianceRisk',
      title: '合规风险',
      width: 110,
      render: renderComplianceRisk
    },
    {
      dataIndex: 'tags',
      title: '标签',
      width: 220,
      render: renderTagMap
    },
    {
      dataIndex: 'lastSyncAt',
      title: '最近同步',
      width: 180,
      render: renderTime
    }
  ], [orgId]);

  const coverageProviderColumns = useMemo(() => [
    {
      dataIndex: 'provider',
      title: '云厂商',
      width: 90,
      ellipsis: true,
      render: (text) => text ? <a onClick={() => applyAssetFilters({ providers: [text] })}>{text}</a> : '-'
    },
    {
      dataIndex: 'assetCount',
      title: '资产',
      width: 68,
      render: numberText
    },
    {
      dataIndex: 'iacManagedAssetCount',
      title: 'IaC',
      width: 62,
      render: numberText
    },
    {
      dataIndex: 'cloudOnlyAssetCount',
      title: '未纳管',
      width: 78,
      render: (value, record) => value ? <a onClick={() => applyAssetFilters({ providers: [record.provider], managedBy: ['cloud_only'] })}><Tag color='warning'>{numberText(value)}</Tag></a> : 0
    },
    {
      dataIndex: 'unownedAssetCount',
      title: '无负责人',
      width: 88,
      render: (value) => value ? <Tag color='warning'>{numberText(value)}</Tag> : 0
    },
    {
      dataIndex: 'highRiskAssetCount',
      title: '高风险',
      width: 78,
      render: (value) => value ? <Tag color='error'>{numberText(value)}</Tag> : 0
    }
  ], [searchParams]);

  const coverageAccountColumns = useMemo(() => [
    {
      dataIndex: 'accountName',
      title: '账号',
      width: 170,
      ellipsis: true,
      render: (text, record) => (
        <Space direction='vertical' size={0}>
          <a onClick={() => applyAssetFilters({
            providers: record.provider ? [record.provider] : undefined,
            accountIds: record.accountId ? [record.accountId] : undefined
          })}>{text || record.accountId || '-'}</a>
          <Text type='secondary'>{record.provider || '-'} / {record.accountId || '-'}</Text>
        </Space>
      )
    },
    {
      dataIndex: 'validationStatus',
      title: '验证',
      width: 80,
      render: (text) => text ? <Tag>{text}</Tag> : '-'
    },
    {
      dataIndex: 'assetCount',
      title: '资产',
      width: 68,
      render: numberText
    },
    {
      dataIndex: 'cloudOnlyAssetCount',
      title: '未纳管',
      width: 78,
      render: (value, record) => value ? <a onClick={() => applyAssetFilters({
        providers: record.provider ? [record.provider] : undefined,
        accountIds: record.accountId ? [record.accountId] : undefined,
        managedBy: ['cloud_only']
      })}><Tag color='warning'>{numberText(value)}</Tag></a> : 0
    },
    {
      dataIndex: 'unownedAssetCount',
      title: '无负责人',
      width: 88,
      render: (value) => value ? <Tag color='warning'>{numberText(value)}</Tag> : 0
    }
  ], [searchParams]);

  const coverageTypeColumns = useMemo(() => [
    {
      dataIndex: 'assetType',
      title: '资产类型',
      width: 140,
      ellipsis: true,
      render: (text) => <a onClick={() => applyAssetFilters({ assetTypes: [text] })}>{assetTypeMap[text] || text || '-'}</a>
    },
    {
      dataIndex: 'assetCount',
      title: '资产',
      width: 68,
      render: numberText
    },
    {
      dataIndex: 'cloudCollectedAssetCount',
      title: '云采集',
      width: 78,
      render: numberText
    },
    {
      dataIndex: 'cloudOnlyAssetCount',
      title: '未纳管',
      width: 78,
      render: (value, record) => value ? <a onClick={() => applyAssetFilters({ assetTypes: [record.assetType], managedBy: ['cloud_only'] })}><Tag color='warning'>{numberText(value)}</Tag></a> : 0
    },
    {
      dataIndex: 'highRiskAssetCount',
      title: '高风险',
      width: 78,
      render: (value) => value ? <Tag color='error'>{numberText(value)}</Tag> : 0
    }
  ], [searchParams]);

  const governanceLifecycleColumns = useMemo(() => [
    {
      dataIndex: 'name',
      title: '生命周期',
      width: 120,
      render: (text) => renderLifecycle(text)
    },
    {
      dataIndex: 'count',
      title: '资产',
      width: 80,
      render: numberText
    },
    {
      dataIndex: 'cost',
      title: '成本',
      width: 100,
      render: renderCost
    }
  ], []);

  const governanceComplianceColumns = useMemo(() => [
    {
      dataIndex: 'name',
      title: '合规风险',
      width: 120,
      render: (text) => text === '未设置' ? text : renderComplianceRisk(text)
    },
    {
      dataIndex: 'count',
      title: '资产',
      width: 80,
      render: numberText
    },
    {
      dataIndex: 'cost',
      title: '成本',
      width: 100,
      render: renderCost
    }
  ], []);

  const governanceTopCostColumns = useMemo(() => [
    {
      dataIndex: 'name',
      title: '资产',
      width: 220,
      render: (text, record) => (
        <a onClick={() => openDetail(record.id)}>
          <EllipsisText>{text || record.nativeId || '-'}</EllipsisText>
        </a>
      )
    },
    {
      dataIndex: 'provider',
      title: '云厂商',
      width: 100,
      render: (text) => text ? <Tag color='blue'>{text}</Tag> : '-'
    },
    {
      dataIndex: 'assetType',
      title: '类型',
      width: 130,
      render: (text) => assetTypeMap[text] || text || '-'
    },
    {
      dataIndex: 'cost',
      title: '成本',
      width: 100,
      render: renderCost
    }
  ], [orgId]);

  const governanceTopRiskColumns = useMemo(() => [
    {
      dataIndex: 'name',
      title: '资产',
      width: 220,
      render: (text, record) => (
        <a onClick={() => openDetail(record.id)}>
          <EllipsisText>{text || record.nativeId || '-'}</EllipsisText>
        </a>
      )
    },
    {
      dataIndex: 'complianceRisk',
      title: '合规风险',
      width: 100,
      render: renderComplianceRisk
    },
    {
      dataIndex: 'lifecycle',
      title: '生命周期',
      width: 100,
      render: renderLifecycle
    },
    {
      dataIndex: 'riskScore',
      title: '风险分',
      width: 90,
      render: (value) => Number(value || 0).toFixed(1)
    }
  ], [orgId]);

  const applicationColumns = useMemo(() => [
    {
      dataIndex: 'application',
      title: '应用',
      width: 220,
      render: (text) => (
        <a onClick={() => openApplicationDetail(text)}>
          <EllipsisText>{text || '-'}</EllipsisText>
        </a>
      )
    },
    {
      title: '操作',
      width: 110,
      render: (_, record) => (
        <Button type='link' size='small' onClick={() => openApplicationDetail(record.application, 'manageRelations')}>
          维护依赖
        </Button>
      )
    },
    {
      dataIndex: 'assetCount',
      title: '绑定资源',
      width: 110,
      render: (text, record) => <a onClick={() => showApplicationAssets(record.application)}>{text || 0}</a>
    },
    {
      dataIndex: 'incomingAppCount',
      title: '调用方',
      width: 100,
      render: (text) => text || 0
    },
    {
      dataIndex: 'outgoingAppCount',
      title: '调用应用',
      width: 110,
      render: (text) => text || 0
    },
    {
      dataIndex: 'recentChangeCount',
      title: `近${riskWindowDays}天变更`,
      width: 120,
      render: (text) => text || 0
    },
    {
      dataIndex: 'impactedAppCount',
      title: '影响应用',
      width: 110,
      render: (text, record) => (
        text ? <a onClick={() => openApplicationDetail(record.application, 'impact')}>{text}</a> : 0
      )
    },
    {
      dataIndex: 'riskLevel',
      title: '变更风险',
      width: 110,
      render: renderApplicationRisk
    },
    {
      dataIndex: 'riskReason',
      title: '风险原因',
      width: 220,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'owner',
      title: '负责人',
      width: 160,
      ellipsis: true,
      render: renderCompactList
    },
    {
      dataIndex: 'businessLine',
      title: '业务线',
      width: 160,
      ellipsis: true,
      render: renderCompactList
    }
  ], [riskWindowDays]);

  const applicationImpactColumns = useMemo(() => [
    {
      dataIndex: 'application',
      title: '影响应用',
      width: 220,
      render: (text) => <a onClick={() => openApplicationDetail(text)}>{text || '-'}</a>
    },
    {
      dataIndex: 'relationCount',
      title: '关系数',
      width: 90,
      render: (text) => text || 0
    },
    {
      dataIndex: 'sourceAssetCount',
      title: '源资源',
      width: 90,
      render: (text) => text || 0
    },
    {
      dataIndex: 'targetAssetCount',
      title: '目标资源',
      width: 90,
      render: (text) => text || 0
    },
    {
      dataIndex: 'relationSources',
      title: '来源',
      width: 180,
      render: renderRelationSources
    },
    {
      dataIndex: 'latestRelationAt',
      title: '最近关系更新',
      width: 180,
      render: renderTime
    }
  ], []);

  const applicationRelationColumns = useMemo(() => [
    {
      dataIndex: 'application',
      title: '应用',
      width: 220,
      render: (text) => <a onClick={() => openApplicationDetail(text)}>{text || '-'}</a>
    },
    {
      dataIndex: 'assetRelationCount',
      title: '资源关系',
      width: 100,
      render: (text) => text || 0
    },
    {
      dataIndex: 'sourceAssetCount',
      title: '源资源',
      width: 90,
      render: (text) => text || 0
    },
    {
      dataIndex: 'targetAssetCount',
      title: '目标资源',
      width: 90,
      render: (text) => text || 0
    },
    {
      dataIndex: 'source',
      title: '来源',
      width: 130,
      render: relationSourceLabel
    },
    {
      dataIndex: 'latestRelationAt',
      title: '最近关系更新',
      width: 180,
      render: renderTime
    }
  ], []);

  const applicationAssetColumns = useMemo(() => [
    {
      dataIndex: 'name',
      title: '资产',
      width: 220,
      render: (text, record) => <a onClick={() => openDetail(record.id)}>{text || record.nativeId || '-'}</a>
    },
    {
      dataIndex: 'assetType',
      title: '资产类型',
      width: 150,
      render: (text) => assetTypeMap[text] || text || '-'
    },
    {
      dataIndex: 'provider',
      title: '云厂商',
      width: 100,
      render: (text) => text ? <Tag color='blue'>{text}</Tag> : '-'
    },
    {
      dataIndex: 'status',
      title: '状态',
      width: 100,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'complianceRisk',
      title: '合规风险',
      width: 110,
      render: renderComplianceRisk
    },
    {
      dataIndex: 'updatedAt',
      title: '更新时间',
      width: 180,
      render: renderTime
    }
  ], []);

  const applicationChangeColumns = useMemo(() => [
    {
      dataIndex: 'assetName',
      title: '资产',
      width: 220,
      render: (text, record) => <a onClick={() => openDetail(record.assetId)}>{text || record.assetId || '-'}</a>
    },
    {
      dataIndex: 'changeType',
      title: '类型',
      width: 90,
      render: (text) => <Tag color={changeTypeColorMap[text]}>{changeTypeMap[text] || text || '-'}</Tag>
    },
    {
      dataIndex: 'summary',
      title: '摘要',
      width: 180,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'source',
      title: '来源',
      width: 120,
      render: (text) => sourceMap[text] || text || '-'
    },
    {
      dataIndex: 'createdAt',
      title: '时间',
      width: 180,
      render: renderTime
    }
  ], []);

  const syncTaskRegionBreakdownColumns = useMemo(() => [
    {
      dataIndex: 'name',
      title: '区域',
      width: 130,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'taskCount',
      title: '任务',
      width: 72,
      render: numberText
    },
    {
      dataIndex: 'collected',
      title: '采集',
      width: 72,
      render: numberText
    },
    {
      dataIndex: 'failedCount',
      title: '失败',
      width: 72,
      render: numberText
    },
    {
      dataIndex: 'failureRate',
      title: '失败率',
      width: 88,
      render: percentText
    }
  ], []);

  const syncTaskAssetTypeBreakdownColumns = useMemo(() => [
    {
      dataIndex: 'key',
      title: '资产类型',
      width: 150,
      render: (text, record) => assetTypeMap[text] || record.name || text || '-'
    },
    {
      dataIndex: 'taskCount',
      title: '任务',
      width: 72,
      render: numberText
    },
    {
      dataIndex: 'collected',
      title: '采集',
      width: 72,
      render: numberText
    },
    {
      dataIndex: 'failedCount',
      title: '失败',
      width: 72,
      render: numberText
    },
    {
      dataIndex: 'failureRate',
      title: '失败率',
      width: 88,
      render: percentText
    }
  ], []);

  const syncTaskApiMetricBreakdownColumns = useMemo(() => [
    {
      title: 'API',
      width: 320,
      render: (_, record) => (
        <Space direction='vertical' size={0}>
          <Space size={4} wrap={true}>
            {record.provider && <Tag color='blue'>{record.provider}</Tag>}
            <Text type='secondary'>{record.region || '-'}</Text>
            <Text type='secondary'>{record.service || '-'}</Text>
          </Space>
          <EllipsisText>{record.path || '-'}</EllipsisText>
        </Space>
      )
    },
    {
      dataIndex: 'callCount',
      title: '调用',
      width: 72,
      render: numberText
    },
    {
      dataIndex: 'failedCount',
      title: '失败',
      width: 72,
      render: numberText
    },
    {
      dataIndex: 'retriedCount',
      title: '重试',
      width: 72,
      render: numberText
    },
    {
      dataIndex: 'failureRate',
      title: '失败率',
      width: 88,
      render: percentText
    },
    {
      dataIndex: 'avgDurationMs',
      title: '平均耗时',
      width: 100,
      render: durationText
    },
    {
      dataIndex: 'maxDurationMs',
      title: '最大耗时',
      width: 100,
      render: durationText
    },
    {
      dataIndex: 'trend',
      title: '近趋势',
      width: 260,
      render: renderSyncTaskApiMetricTrend
    }
  ], []);

  const syncTaskColumns = useMemo(() => [
    {
      dataIndex: 'accountName',
      title: '账号',
      width: 180,
      ellipsis: true
    },
    {
      dataIndex: 'accountSource',
      title: '来源',
      width: 110,
      render: (text) => accountSourceMap[text] || text || '-'
    },
    {
      dataIndex: 'provider',
      title: '云厂商',
      width: 100,
      render: (text) => text ? <Tag color='blue'>{text}</Tag> : '-'
    },
    {
      dataIndex: 'status',
      title: '状态',
      width: 100,
      render: (text) => <Tag color={taskStatusColorMap[text]}>{taskStatusMap[text] || text || '-'}</Tag>
    },
    {
      dataIndex: 'regions',
      title: '区域',
      width: 180,
      render: joinList
    },
    {
      dataIndex: 'assetTypes',
      title: '资产类型',
      width: 240,
      render: (value) => joinList((value || []).map((it) => assetTypeMap[it] || it))
    },
    {
      title: '范围摘要',
      width: 320,
      render: (_, record) => renderSyncTaskScopeSummary(record)
    },
    {
      dataIndex: 'stats',
      title: '子周期',
      width: 180,
      render: renderSyncTaskSchedule
    },
    {
      dataIndex: 'stats',
      title: '统计',
      width: 260,
      render: (stats, record) => {
        const safeStats = syncTaskScopeSummary(record);
        return (
          <Space direction='vertical' size={0}>
            <Text>采集 {numberText(safeStats.collected)} / 新增 {numberText(safeStats.created)} / 更新 {numberText(safeStats.updated)} / 跳过 {numberText(safeStats.skipped)}</Text>
            <Text type='secondary'>采集耗时 {durationText(safeStats.collectorDurationMs)} / 总耗时 {durationText(safeStats.durationMs)}</Text>
          </Space>
        );
      }
    },
    {
      title: '失败摘要',
      width: 320,
      render: (_, record) => renderSyncTaskFailureSummary(record)
    },
    {
      dataIndex: 'errorMessage',
      title: '错误日志',
      width: 280,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'startedAt',
      title: '开始时间',
      width: 180,
      render: renderTime
    },
    {
      dataIndex: 'endedAt',
      title: '结束时间',
      width: 180,
      render: renderTime
    },
    {
      title: '操作',
      width: 90,
      fixed: 'right',
      render: (_, record) => (
        <Button type='link' size='small' onClick={() => openSyncTaskDetail(record)}>
          详情
        </Button>
      )
    }
  ], []);

  const syncTaskRerunGroupColumns = useMemo(() => [
    {
      dataIndex: 'id',
      title: '新任务',
      width: 180,
      render: (text) => text ? (
        <Button type='link' size='small' onClick={() => openSyncTaskDetail({ id: text })}>
          {text}
        </Button>
      ) : '-'
    },
    {
      dataIndex: 'sourceTaskId',
      title: '源失败任务',
      width: 180,
      render: (text) => text ? (
        <Button type='link' size='small' onClick={() => openSyncTaskDetail({ id: text })}>
          {text}
        </Button>
      ) : '-'
    },
    {
      dataIndex: 'sourceTaskStatus',
      title: '源状态',
      width: 90,
      render: (text) => text ? <Tag color={taskStatusColorMap[text]}>{taskStatusMap[text] || text}</Tag> : '-'
    },
    {
      dataIndex: 'status',
      title: '新状态',
      width: 90,
      render: (text) => <Tag color={taskStatusColorMap[text]}>{taskStatusMap[text] || text || '-'}</Tag>
    },
    {
      dataIndex: 'accountName',
      title: '账号',
      width: 180,
      ellipsis: true
    },
    {
      dataIndex: 'provider',
      title: '云厂商',
      width: 100,
      render: (text) => text ? <Tag color='blue'>{text}</Tag> : '-'
    },
    {
      dataIndex: 'stats',
      title: '子周期',
      width: 180,
      render: renderSyncTaskSchedule
    },
    {
      dataIndex: 'startedAt',
      title: '开始时间',
      width: 180,
      render: renderTime
    },
    {
      dataIndex: 'endedAt',
      title: '结束时间',
      width: 180,
      render: renderTime
    }
  ], []);

  const syncTaskLogColumns = useMemo(() => [
    {
      dataIndex: 'createdAt',
      title: '时间',
      width: 180,
      render: renderTime
    },
    {
      dataIndex: 'level',
      title: '级别',
      width: 90,
      render: (text) => <Tag color={syncLogLevelColorMap[text]}>{syncLogLevelMap[text] || text || '-'}</Tag>
    },
    {
      dataIndex: 'stage',
      title: '阶段',
      width: 120,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'message',
      title: '消息',
      width: 240,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'data',
      title: '数据',
      width: 320,
      render: renderJSON
    }
  ], []);

  const syncTaskScopeMetricColumns = useMemo(() => [
    {
      dataIndex: 'region',
      title: '区域',
      width: 140,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'assetType',
      title: '资产类型',
      width: 160,
      render: (text) => assetTypeMap[text] || text || '-'
    },
    {
      dataIndex: 'status',
      title: '状态',
      width: 110,
      render: (text) => text ? <Tag color={syncTaskMetricStatusColorMap[text]}>{syncTaskMetricStatusMap[text] || text}</Tag> : '-'
    },
    {
      dataIndex: 'collected',
      title: '采集数',
      width: 90,
      render: numberText
    },
    {
      dataIndex: 'durationMs',
      title: '耗时',
      width: 100,
      render: durationText
    }
  ], []);

  const syncTaskFailureDetailColumns = useMemo(() => [
    {
      dataIndex: 'category',
      title: '分类',
      width: 110,
      render: (text) => text ? (
        <Tag color={syncTaskFailureCategoryColorMap[text]}>{syncTaskFailureCategoryMap[text] || text}</Tag>
      ) : '-'
    },
    {
      dataIndex: 'retryable',
      title: '可重试',
      width: 90,
      render: (value) => value ? <Tag color='warning'>是</Tag> : <Tag>否</Tag>
    },
    {
      dataIndex: 'message',
      title: '错误',
      width: 320,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'retryHint',
      title: '建议',
      width: 260,
      render: (text) => text || '-'
    }
  ], []);

  const syncTaskApiMetricColumns = useMemo(() => [
    {
      dataIndex: 'region',
      title: '区域',
      width: 140,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'service',
      title: '服务',
      width: 110,
      render: (text) => text ? <Tag color='blue'>{text}</Tag> : '-'
    },
    {
      dataIndex: 'path',
      title: 'API',
      width: 260,
      render: (text) => text ? <EllipsisText>{text}</EllipsisText> : '-'
    },
    {
      dataIndex: 'status',
      title: '状态',
      width: 150,
      render: (text, record) => (
        <Space size={4}>
          {text ? <Tag color={syncTaskMetricStatusColorMap[text]}>{syncTaskMetricStatusMap[text] || text}</Tag> : '-'}
          {record.httpStatus ? <Tag color={httpStatusColor(record.httpStatus)}>HTTP {record.httpStatus}</Tag> : null}
        </Space>
      )
    },
    {
      dataIndex: 'durationMs',
      title: '耗时',
      width: 100,
      render: durationText
    },
    {
      dataIndex: 'attempts',
      title: '尝试',
      width: 120,
      render: (text, record) => {
        const retryCount = Number(record.retryCount || 0);
        if (!text && !retryCount) {
          return '-';
        }
        return (
          <Space size={4}>
            <Text>{numberText(text || 0)}</Text>
            {retryCount > 0 && <Tag color='warning'>重试 {retryCount}</Tag>}
          </Space>
        );
      }
    },
    {
      title: 'Scope',
      width: 260,
      render: (_, record) => {
        if (!record.compartmentId && !record.pageTokenUsed) {
          return '-';
        }
        return (
          <Space direction='vertical' size={0}>
            {record.compartmentId && <EllipsisText>{record.compartmentId}</EllipsisText>}
            {record.pageTokenUsed && <Tag color='processing'>分页请求</Tag>}
          </Space>
        );
      }
    },
    {
      title: '错误/建议',
      width: 320,
      render: (_, record) => {
        if (!record.errorCategory && !record.retryHint && !record.error && record.retryable === undefined) {
          return '-';
        }
        return (
          <Space direction='vertical' size={0}>
            {record.errorCategory && (
              <Tag color={syncTaskFailureCategoryColorMap[record.errorCategory]}>
                {syncTaskFailureCategoryMap[record.errorCategory] || record.errorCategory}
              </Tag>
            )}
            {record.retryable !== undefined && (
              <Text type='secondary'>可重试：{record.retryable ? '是' : '否'}</Text>
            )}
            {(record.retryHint || record.error) && (
              <EllipsisText>{record.retryHint || record.error}</EllipsisText>
            )}
          </Space>
        );
      }
    }
  ], []);

  const importPreviewColumns = useMemo(() => [
    {
      dataIndex: 'index',
      title: '行号',
      width: 70
    },
    {
      dataIndex: 'action',
      title: '动作',
      width: 110,
      render: (text) => <Tag color={importActionColorMap[text]}>{importActionMap[text] || text || '-'}</Tag>
    },
    {
      dataIndex: 'name',
      title: '资产',
      width: 260,
      render: (_, record) => (
        <Space direction='vertical' size={0}>
          <Text>{record.name || record.nativeId || '-'}</Text>
          <Text type='secondary'>
            {[record.provider, record.region, assetTypeMap[record.assetType] || record.assetType].filter(Boolean).join(' / ') || '-'}
          </Text>
        </Space>
      )
    },
    {
      dataIndex: 'nativeId',
      title: '资源ID',
      width: 220,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'diff',
      title: '差异',
      width: 180,
      render: (_, record) => {
        const assetDiffCount = Object.keys(record.diff || {}).length;
        const ownershipDiffCount = Object.keys(record.ownershipDiff || {}).length;
        return (
          <Space size={4} wrap={true}>
            <Tag color={assetDiffCount ? 'blue' : 'default'}>资产 {assetDiffCount}</Tag>
            <Tag color={ownershipDiffCount ? 'gold' : 'default'}>归属 {ownershipDiffCount}</Tag>
          </Space>
        );
      }
    },
    {
      dataIndex: 'error',
      title: '异常',
      width: 240,
      render: (text) => text ? <Text type='danger'>{text}</Text> : '-'
    }
  ], []);

  const relationColumns = useMemo(() => [
    {
      dataIndex: 'relationType',
      title: '关系',
      width: 110,
      render: (text, record) => {
        const label = relationTypeMap[text] || text || '-';
        return record.sourceAssetId === (detail && detail.id) ? label : `被${label}`;
      }
    },
    {
      dataIndex: 'targetAssetName',
      title: '关联资产',
      width: 260,
      render: (_, record) => {
        const currentIsSource = record.sourceAssetId === (detail && detail.id);
        const assetId = currentIsSource ? record.targetAssetId : record.sourceAssetId;
        const name = currentIsSource ? record.targetAssetName : record.sourceAssetName;
        const type = currentIsSource ? record.targetAssetType : record.sourceAssetType;
        return (
          <Space direction='vertical' size={0}>
            <a onClick={() => openDetail(assetId)}>{name || assetId || '-'}</a>
            <Text type='secondary'>{assetTypeMap[type] || type || '-'}</Text>
          </Space>
        );
      }
    },
    {
      dataIndex: 'source',
      title: '来源',
      width: 130,
      render: relationSourceLabel
    },
    {
      dataIndex: 'metadata',
      title: '元数据',
      width: 260,
      render: renderJSON
    }
  ], [detail]);

  const securityRuleColumns = useMemo(() => [
    {
      dataIndex: 'direction',
      title: '方向',
      width: 90,
      render: (text) => text === 'egress' ? '出站' : '入站'
    },
    {
      dataIndex: 'protocol',
      title: '协议',
      width: 90,
      render: (text) => text || 'all'
    },
    {
      dataIndex: 'portRange',
      title: '端口',
      width: 120,
      render: (text) => text || 'all'
    },
    {
      dataIndex: 'source',
      title: '来源',
      width: 180,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'destination',
      title: '目标',
      width: 180,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'publicExposure',
      title: '公网暴露',
      width: 110,
      render: (value) => value ? <Tag color='error'>公网</Tag> : <Tag>否</Tag>
    },
    {
      dataIndex: 'description',
      title: '说明',
      render: (text) => text || '-'
    }
  ], []);

  const changeColumns = useMemo(() => [
    {
      dataIndex: 'changeType',
      title: '类型',
      width: 90,
      render: (text) => <Tag color={changeTypeColorMap[text]}>{changeTypeMap[text] || text || '-'}</Tag>
    },
    {
      dataIndex: 'summary',
      title: '摘要',
      width: 160,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'source',
      title: '来源',
      width: 120,
      render: (text) => sourceMap[text] || text || '-'
    },
    {
      dataIndex: 'createdAt',
      title: '时间',
      width: 180,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'diff',
      title: '差异',
      width: 320,
      render: renderJSON
    }
  ], []);

  const assetActionColumns = useMemo(() => [
    {
      dataIndex: 'name',
      title: '动作',
      width: 140,
      render: (text, record) => (
        <Space direction='vertical' size={0}>
          <Text strong={true}>{text || record.key || '-'}</Text>
          <Text type='secondary'>{record.description || '-'}</Text>
        </Space>
      )
    },
    {
      dataIndex: 'riskLevel',
      title: '风险',
      width: 90,
      render: renderActionRisk
    },
    {
      dataIndex: 'adapterMode',
      title: '执行方式',
      width: 150,
      render: (_, record) => (
        <Space direction='vertical' size={0}>
          <Space size={4} wrap={true}>
            {renderActionAdapterMode(record.adapterMode)}
            {renderActionAdapterStatus(record.adapterStatus)}
          </Space>
          {record.providerAdapter && (
            <Text type='secondary'>{record.providerAdapter}</Text>
          )}
        </Space>
      )
    },
    {
      dataIndex: 'enabled',
      title: '状态',
      width: 160,
      render: (enabled, record) => (
        enabled ? <Tag color='success'>可创建</Tag> : <Tag color='default'>{record.disabledReason || '未开放'}</Tag>
      )
    },
    {
      title: '要求',
      width: 150,
      render: (_, record) => (
        <Space size={[4, 4]} wrap={true}>
          {record.requiresApproval && <Tag color='warning'>需审批</Tag>}
          {record.destructive && <Tag color='error'>高危</Tag>}
          {!record.requiresApproval && !record.destructive && <Tag>无需审批</Tag>}
        </Space>
      )
    },
    {
      title: '操作',
      width: 170,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button
            size='small'
            onClick={() => onDryRunAssetAction(record)}
            loading={assetActionDryRunning && actionRequestKey === record.key}
          >
            预检查
          </Button>
          <Button
            size='small'
            type='primary'
            disabled={!record.enabled}
            loading={assetActionCreating && creatingActionKey === record.key}
            onClick={() => onCreateAssetAction(record)}
          >
            创建任务
          </Button>
        </Space>
      )
    }
  ], [detail, assetActionDryRunning, actionRequestKey, assetActionCreating, creatingActionKey]);

  const assetActionCheckColumns = useMemo(() => [
    {
      dataIndex: 'name',
      title: '检查项',
      width: 140,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'status',
      title: '状态',
      width: 90,
      render: (text) => <Tag color={actionCheckStatusColorMap[text]}>{actionCheckStatusMap[text] || text || '-'}</Tag>
    },
    {
      dataIndex: 'message',
      title: '说明',
      render: (text) => text || '-'
    }
  ], []);

  const selectFilter = (key, placeholder, options = [], labelGetter = (it) => it, valueGetter = (it) => it) => (
    <Select
      className={styles.selectFilter}
      mode='multiple'
      allowClear={true}
      maxTagCount='responsive'
      placeholder={placeholder}
      value={(searchParams.form || {})[key]}
      onChange={(value) => onFilterChange(key, value)}
    >
      {(options || []).map((it) => (
        <Option key={valueGetter(it)} value={valueGetter(it)}>{labelGetter(it)}</Option>
      ))}
    </Select>
  );

  const createActionRequiresConfirm = assetActionCreateRecord && assetActionCreateRecord.adapterMode === 'provider';
  const createActionExpectedResourceId = detail && (detail.nativeId || detail.id) || '';
  const createActionRequiresTags = assetActionCreateRecord && assetActionCreateRecord.key === 'update_tags';
  const createActionRequiresParams = assetActionCreateRecord && assetActionNeedsParams(assetActionCreateRecord.key);
  const createActionOkDisabled = !!(createActionRequiresConfirm && (
    (assetActionCreateReason || '').trim().length < 6 ||
    (assetActionConfirmAction || '').trim() !== (assetActionCreateRecord && assetActionCreateRecord.key) ||
    (assetActionConfirmResourceId || '').trim() !== createActionExpectedResourceId
  )) ||
    !!(createActionRequiresTags && !(assetActionTagsText || '').trim()) ||
    !!(createActionRequiresParams && !(assetActionParamsText || '').trim());
  const assetListClassName = [
    styles.assetListContent,
    isCloudAssets ? styles.cloudAssetListContent : null
  ].filter(Boolean).join(' ');

  return (
    <Layout
      extraHeader={
        <PageHeader
          title={pageTitle}
          breadcrumb={true}
        />
      }
    >
      <div className='idcos-card'>
        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          <TabPane tab='资产列表' key='assets'>
            <Space className={assetListClassName} size='middle' direction='vertical' style={{ width: '100%', display: 'flex' }}>
              {isCloudAssets && (
                <Collapse
                  className={styles.assetInsightCollapse}
                  bordered={false}
                  defaultActiveKey={[]}
                  expandIconPosition='right'
                >
                  <CollapsePanel
                    key='coverage'
                    header={(
                      <AssetInsightHeader
                        title='资产覆盖率'
                        description='按账号、云厂商和资源类型统计 IaC 纳管与治理缺口'
                      >
                        <Tag color='blue'>资产 {numberText(coverageMetrics.totalAssets)}</Tag>
                        <Tag color={coverageMetrics.cloudOnlyAssets ? 'warning' : 'default'}>
                          未纳管 {numberText(coverageMetrics.cloudOnlyAssets)}
                        </Tag>
                        <Tag color={coverageMetrics.unownedAssets ? 'warning' : 'default'}>
                          无负责人 {numberText(coverageMetrics.unownedAssets)}
                        </Tag>
                      </AssetInsightHeader>
                    )}
                  >
                  <div className={styles.coveragePanel}>
                    <div className={styles.coverageHeader}>
                      <div>
                        <Text strong={true}>资产覆盖率</Text>
                        <div className={styles.coverageDesc}>
                          多云资产按账号、云厂商和资源类型统计 IaC 纳管、云采集、未纳管和归属缺口
                        </div>
                      </div>
                      <Button icon={<ReloadOutlined />} loading={coverageLoading} onClick={fetchCoverage}>
                        刷新覆盖率
                      </Button>
                    </div>
                    <div className={styles.coverageMetricGrid}>
                      <CoverageMetric
                        title='资产总数'
                        value={coverageMetrics.totalAssets}
                        description={`Provider ${numberText(coverageMetrics.providerCount)}，账号 ${numberText(coverageMetrics.accountCount)}`}
                      />
                      <CoverageMetric
                        title='IaC 纳管'
                        value={coverageMetrics.iacManagedAssets}
                        description={`覆盖率 ${iacCoverageRate.toFixed(1)}%`}
                      />
                      <CoverageMetric
                        title='未纳管资产'
                        value={coverageMetrics.cloudOnlyAssets}
                        description={`占比 ${cloudOnlyRate.toFixed(1)}%`}
                        tone={coverageMetrics.cloudOnlyAssets ? 'coverageWarning' : ''}
                      />
                      <CoverageMetric
                        title='无负责人'
                        value={coverageMetrics.unownedAssets}
                        description={`归属覆盖 ${ownershipCoverageRate.toFixed(1)}%`}
                        tone={coverageMetrics.unownedAssets ? 'coverageWarning' : ''}
                      />
                    </div>
                    <div className={styles.coverageProgressGrid}>
                      <div>
                        <div className={styles.coverageProgressMeta}>
                          <Text>IaC 纳管覆盖率</Text>
                          <Text type='secondary'>{iacCoverageRate.toFixed(1)}%</Text>
                        </div>
                        <Progress percent={iacCoverageRate} showInfo={false} strokeColor='#2f7de1'/>
                      </div>
                      <div>
                        <div className={styles.coverageProgressMeta}>
                          <Text>资产归属覆盖率</Text>
                          <Text type='secondary'>{ownershipCoverageRate.toFixed(1)}%</Text>
                        </div>
                        <Progress percent={ownershipCoverageRate} showInfo={false} strokeColor='#2ca58d'/>
                      </div>
                    </div>
                    <div className={styles.coverageTableGrid}>
                      <div>
                        <div className={styles.coverageTableTitle}>Provider 覆盖</div>
                        <Table
                          size='small'
                          rowKey='provider'
                          columns={coverageProviderColumns}
                          dataSource={coverageProviders}
                          loading={coverageLoading}
                          pagination={false}
                          scroll={{ x: 'max-content' }}
                        />
                      </div>
                      <div>
                        <div className={styles.coverageTableTitle}>账号覆盖</div>
                        <Table
                          size='small'
                          rowKey={(record) => `${record.provider}-${record.accountId || record.accountRefId || record.accountName}`}
                          columns={coverageAccountColumns}
                          dataSource={coverageAccounts.slice(0, 8)}
                          loading={coverageLoading}
                          pagination={false}
                          scroll={{ x: 'max-content' }}
                        />
                      </div>
                      <div>
                        <div className={styles.coverageTableTitle}>资产类型覆盖</div>
                        <Table
                          size='small'
                          rowKey='assetType'
                          columns={coverageTypeColumns}
                          dataSource={coverageAssetTypes.slice(0, 8)}
                          loading={coverageLoading}
                          pagination={false}
                          scroll={{ x: 'max-content' }}
                        />
                      </div>
                    </div>
                  </div>
                  </CollapsePanel>
                  <CollapsePanel
                    key='kubernetes'
                    header={(
                      <AssetInsightHeader
                        title='Kubernetes 集群'
                        description='EKS、OKE、AKS、GKE 与工作负载层资产汇总'
                      >
                        <Tag color='geekblue'>集群 {numberText(kubernetesClusterCoverage.assetCount || 0)}</Tag>
                        <Tag color='blue'>K8S资产 {numberText(kubernetesTotalAssets)}</Tag>
                      </AssetInsightHeader>
                    )}
                  >
                  <div className={styles.kubernetesPanel}>
                    <div className={styles.coverageHeader}>
                      <div>
                        <Text strong={true}>Kubernetes 集群</Text>
                        <div className={styles.coverageDesc}>
                          EKS、OKE、AKS、GKE 与工作负载层资产汇总
                        </div>
                      </div>
                      <Space size={[8, 8]} wrap={true}>
                        {kubernetesProviderQuickFilters.map((item) => (
                          <Button
                            key={item.provider}
                            size='small'
                            onClick={() => applyAssetFilters({
                              providers: [item.provider],
                              assetTypes: ['kubernetes_cluster']
                            })}
                          >
                            {item.label}
                          </Button>
                        ))}
                        <Button
                          size='small'
                          type='primary'
                          onClick={() => applyAssetFilters({ assetTypes: kubernetesAssetTypes })}
                        >
                          查看K8S资产
                        </Button>
                      </Space>
                    </div>
                    <div className={styles.coverageMetricGrid}>
                      <CoverageMetric
                        title='集群'
                        value={kubernetesClusterCoverage.assetCount || 0}
                        description={`未纳管 ${numberText(kubernetesClusterCoverage.cloudOnlyAssetCount)}，无负责人 ${numberText(kubernetesClusterCoverage.unownedAssetCount)}`}
                      />
                      <CoverageMetric
                        title='工作负载对象'
                        value={kubernetesWorkloadAssets}
                        description='Namespace / Node / Pod / Service / Ingress'
                      />
                      <CoverageMetric
                        title='K8S资产总数'
                        value={kubernetesTotalAssets}
                        description={`云采集 ${numberText(kubernetesCloudCollectedAssets)}`}
                      />
                      <CoverageMetric
                        title='治理缺口'
                        value={kubernetesCloudOnlyAssets + kubernetesUnownedAssets}
                        description={`未纳管 ${numberText(kubernetesCloudOnlyAssets)}，无负责人 ${numberText(kubernetesUnownedAssets)}`}
                        tone={(kubernetesCloudOnlyAssets || kubernetesUnownedAssets) ? 'coverageWarning' : ''}
                      />
                    </div>
                    <div className={styles.kubernetesTable}>
                      <div className={styles.coverageTableTitle}>K8S 类型覆盖</div>
                      <Table
                        size='small'
                        rowKey='assetType'
                        columns={coverageTypeColumns}
                        dataSource={kubernetesCoverageAssetTypes}
                        loading={coverageLoading}
                        pagination={false}
                        scroll={{ x: 'max-content' }}
                        locale={{
                          emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无K8S资产'/>
                        }}
                      />
                    </div>
                  </div>
                  </CollapsePanel>
                  <CollapsePanel
                    key='governance'
                    header={(
                      <AssetInsightHeader
                        title='治理报表'
                        description='按成本、合规风险和生命周期识别治理缺口'
                      >
                        <Tag color='blue'>成本 {Number(governanceReport.totalCost || 0).toFixed(2)}</Tag>
                        <Tag color={(governanceReport.highRiskAssets || governanceReport.criticalRiskAssets) ? 'error' : 'default'}>
                          高风险 {numberText(Number(governanceReport.highRiskAssets || 0) + Number(governanceReport.criticalRiskAssets || 0))}
                        </Tag>
                      </AssetInsightHeader>
                    )}
                  >
                  <div className={styles.governancePanel}>
                    <div className={styles.coverageHeader}>
                      <div>
                        <Text strong={true}>治理报表</Text>
                        <div className={styles.coverageDesc}>
                          按成本、合规风险和生命周期识别高成本资产、风险资产和归属缺口
                        </div>
                      </div>
                      <Button icon={<ReloadOutlined />} loading={governanceReportLoading} onClick={fetchGovernanceReport}>
                        刷新治理报表
                      </Button>
                    </div>
                    <div className={styles.coverageMetricGrid}>
                      <CoverageMetric
                        title='总成本'
                        value={Number(governanceReport.totalCost || 0).toFixed(2)}
                        description={`平均风险分 ${Number(governanceReport.averageRiskScore || 0).toFixed(1)}`}
                      />
                      <CoverageMetric
                        title='高/严重风险'
                        value={Number(governanceReport.highRiskAssets || 0) + Number(governanceReport.criticalRiskAssets || 0)}
                        description={`严重 ${numberText(governanceReport.criticalRiskAssets)}，高 ${numberText(governanceReport.highRiskAssets)}`}
                        tone={(governanceReport.highRiskAssets || governanceReport.criticalRiskAssets) ? 'coverageWarning' : ''}
                      />
                      <CoverageMetric
                        title='无负责人'
                        value={governanceReport.missingOwnerAssets}
                        description={`缺口 ${governanceOwnerGapRate.toFixed(1)}%`}
                        tone={governanceReport.missingOwnerAssets ? 'coverageWarning' : ''}
                      />
                      <CoverageMetric
                        title='未绑定应用'
                        value={governanceReport.missingApplicationAssets}
                        description={`缺口 ${governanceApplicationGapRate.toFixed(1)}%`}
                        tone={governanceReport.missingApplicationAssets ? 'coverageWarning' : ''}
                      />
                    </div>
                    <div className={styles.coverageProgressGrid}>
                      <div>
                        <div className={styles.coverageProgressMeta}>
                          <Text>负责人缺口</Text>
                          <Text type='secondary'>{governanceOwnerGapRate.toFixed(1)}%</Text>
                        </div>
                        <Progress percent={governanceOwnerGapRate} showInfo={false} strokeColor='#fa8c16'/>
                      </div>
                      <div>
                        <div className={styles.coverageProgressMeta}>
                          <Text>应用绑定缺口</Text>
                          <Text type='secondary'>{governanceApplicationGapRate.toFixed(1)}%</Text>
                        </div>
                        <Progress percent={governanceApplicationGapRate} showInfo={false} strokeColor='#cf1322'/>
                      </div>
                    </div>
                    <div className={styles.governanceTableGrid}>
                      <div>
                        <div className={styles.coverageTableTitle}>生命周期分布</div>
                        <Table
                          size='small'
                          rowKey='name'
                          columns={governanceLifecycleColumns}
                          dataSource={governanceLifecycleBreakdown}
                          loading={governanceReportLoading}
                          pagination={false}
                          scroll={{ x: 'max-content' }}
                        />
                      </div>
                      <div>
                        <div className={styles.coverageTableTitle}>合规风险分布</div>
                        <Table
                          size='small'
                          rowKey='name'
                          columns={governanceComplianceColumns}
                          dataSource={governanceComplianceRiskBreakdown}
                          loading={governanceReportLoading}
                          pagination={false}
                          scroll={{ x: 'max-content' }}
                        />
                      </div>
                      <div>
                        <div className={styles.coverageTableTitle}>高成本资产</div>
                        <Table
                          size='small'
                          rowKey={(record) => record.id || record.nativeId || record.name}
                          columns={governanceTopCostColumns}
                          dataSource={governanceTopCostAssets}
                          loading={governanceReportLoading}
                          pagination={false}
                          scroll={{ x: 'max-content' }}
                        />
                      </div>
                      <div>
                        <div className={styles.coverageTableTitle}>高风险资产</div>
                        <Table
                          size='small'
                          rowKey={(record) => record.id || record.nativeId || record.name}
                          columns={governanceTopRiskColumns}
                          dataSource={governanceTopRiskAssets}
                          loading={governanceReportLoading}
                          pagination={false}
                          scroll={{ x: 'max-content' }}
                        />
                      </div>
                    </div>
                  </div>
                  </CollapsePanel>
                </Collapse>
              )}
              <div className={styles.toolbar}>
                <div className={styles.filterBar}>
                  <InputSearch
                    className={styles.keywordSearch}
                    allowClear={true}
                    enterButton={<SearchOutlined />}
                    placeholder='请输入名称、资源ID、IP或标签'
                    value={assetKeyword}
                    onChange={(e) => {
                      const keyword = e.target.value;
                      setAssetKeyword(keyword);
                      if (!keyword) {
                        onSearch(null, '');
                      }
                    }}
                    onSearch={(keyword) => onSearch(null, keyword)}
                  />
                  <InputSearch
                    className={styles.dslSearch}
                    allowClear={true}
                    enterButton={<SearchOutlined />}
                    placeholder='provider:oci tag.env:demo'
                    value={dslInput}
                    onChange={(e) => setDslInput(e.target.value)}
                    onSearch={onDslSearch}
                  />
                  {selectFilter('providers', '云厂商', filters.providers)}
                  {isCloudAssets && selectFilter('accountIds', '账号', filters.accountIds)}
                  {isCloudAssets && selectFilter('managedBy', '纳管状态', filters.managedBy, (it) => managedByMap[it] || it)}
                  {selectFilter('assetTypes', '资产类型', filters.assetTypes, (it) => assetTypeMap[it] || it)}
                  {selectFilter('projectIds', '项目', filters.projects, (it) => it.projectName, (it) => it.projectId)}
                  {selectFilter('envIds', '环境', filters.envs, (it) => it.envName, (it) => it.envId)}
                  {selectFilter('sources', '来源', filters.sources, (it) => sourceMap[it] || it)}
                  {selectFilter('statuses', '状态', filters.statuses)}
                </div>
                <div className={styles.actionBar}>
                  <Checkbox
                    checked={importOverwriteOwnership}
                    disabled={!canImportAssets}
                    onChange={(e) => setImportOverwriteOwnership(e.target.checked)}
                  >
                    覆盖归属
                  </Checkbox>
                  <Button icon={<DownloadOutlined />} onClick={downloadAssetImportTemplate}>
                    下载导入模板
                  </Button>
                  <Upload
                    accept='.json,application/json'
                    beforeUpload={onImportAssetsFile}
                    showUploadList={false}
                  >
                    <Button
                      icon={<UploadOutlined />}
                      disabled={!canImportAssets}
                      title={!canImportAssets ? permissionMessage('import') : undefined}
                      loading={previewingImportAssets || importingAssets}
                    >
                      导入预检
                    </Button>
                  </Upload>
                  <Button
                    icon={<DownloadOutlined />}
                    disabled={!canExportAssets}
                    title={!canExportAssets ? permissionMessage('export') : undefined}
                    onClick={() => exportAssets('csv')}
                  >
                    {selectedAssetIds.length ? `导出选中 CSV(${selectedAssetIds.length})` : '导出 CSV'}
                  </Button>
                  <Button
                    icon={<DownloadOutlined />}
                    disabled={!canExportAssets}
                    title={!canExportAssets ? permissionMessage('export') : undefined}
                    onClick={() => exportAssets('json')}
                  >
                    {selectedAssetIds.length ? '导出选中 JSON' : '导出 JSON'}
                  </Button>
                  {isCloudAssets && (
                    <Button
                      icon={<SaveOutlined />}
                      disabled={!selectedAssetIds.length || !canBatchGovernance}
                      title={!canBatchGovernance ? permissionMessage('batch_governance') : undefined}
                      onClick={openBatchOwnership}
                    >
                      {selectedAssetIds.length ? `批量治理(${selectedAssetIds.length})` : '批量治理'}
                    </Button>
                  )}
                  <Button icon={<ReloadOutlined />} onClick={refresh}>刷新</Button>
                  <Button
                    type='primary'
                    icon={<SyncOutlined />}
                    disabled={!canSyncIac}
                    title={!canSyncIac ? permissionMessage('sync_iac') : undefined}
                    loading={backfillLoading || assetPermissionsLoading}
                    onClick={syncIac}
                  >
                    同步 IaC 资源
                  </Button>
                </div>
              </div>
              <ConfigProvider
                renderEmpty={
                  () => <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无资产'/>
                }
              >
                <div className={styles.assetTableShell}>
                  <Table
                    rowKey='id'
                    columns={columns}
                    scroll={{ x: 'max-content' }}
                    loading={tableLoading}
                    {...tableProps}
                    rowSelection={{
                      selectedRowKeys: selectedAssetIds,
                      onChange: setSelectedAssetIds
                    }}
                  />
                </div>
              </ConfigProvider>
            </Space>
          </TabPane>
          <TabPane tab='应用依赖' key='applications'>
            <Space size='middle' direction='vertical' style={{ width: '100%', display: 'flex' }}>
              <div className={styles.toolbar}>
                <Space className={styles.filterBar} size={[8, 8]} wrap={true}>
                  <InputSearch
                    className={styles.dslSearch}
                    allowClear={true}
                    enterButton={<SearchOutlined />}
                    placeholder='搜索应用、负责人或业务线'
                    value={applicationKeyword}
                    onChange={(e) => setApplicationKeyword(e.target.value)}
                    onSearch={(q) => refreshApplications({ q, currentPage: 1 })}
                    style={{ width: 320 }}
                  />
                  <Select
                    allowClear={true}
                    placeholder='变更风险'
                    value={applicationRisk}
                    onChange={(risk) => {
                      setApplicationRisk(risk);
                      refreshApplications({ risk, currentPage: 1 });
                    }}
                    style={{ minWidth: 160 }}
                  >
                    {Object.entries(applicationRiskMap).map(([value, label]) => (
                      <Option key={value} value={value}>{label}</Option>
                    ))}
                  </Select>
                </Space>
                <Space className={styles.actionBar} size={[8, 8]} wrap={true}>
                  <Button
                    icon={<SettingOutlined />}
                    disabled={!canManageRiskRules}
                    title={!canManageRiskRules ? permissionMessage('risk_rules') : undefined}
                    loading={riskRuleLoading || assetPermissionsLoading}
                    onClick={openRiskRuleConfig}
                  >
                    风险规则
                  </Button>
                  <Button icon={<ReloadOutlined />} onClick={() => refreshApplications()}>刷新</Button>
                </Space>
              </div>
              <Table
                rowKey='application'
                columns={applicationColumns}
                scroll={{ x: 'min-content' }}
                loading={applicationsLoading}
                dataSource={applicationList}
                pagination={{
                  total: applicationsData && applicationsData.total,
                  pageSize: applicationsData && applicationsData.pageSize || 10,
                  showTotal: (total) => `共${total}条`
                }}
                onChange={({ current, pageSize }) => refreshApplications({ currentPage: current, pageSize })}
              />
            </Space>
          </TabPane>
          <TabPane tab='云采集' key='cloudSync'>
            <Space size='middle' direction='vertical' style={{ width: '100%', display: 'flex' }}>
              {showSyncTaskSummary && (
                <Space size={[8, 8]} wrap={true}>
                  <Text type='secondary'>任务历史筛选</Text>
                  {syncTaskQuery.syncPolicyId && <Tag color='blue'>策略：{syncTaskQuery.syncPolicyId}</Tag>}
                  {syncTaskQuery.syncPolicyScheduleKey && (
                    <Tag color='geekblue'>
                      子周期：{syncTaskQuery.syncPolicyScheduleName || syncTaskQuery.syncPolicyScheduleKey}
                    </Tag>
                  )}
                  <Button
                    type='link'
                    size='small'
                    icon={<SyncOutlined />}
                    loading={cloudSyncLoading || assetPermissionsLoading}
                    disabled={!syncTaskHistorySeed || !canSyncIac}
                    title={!canSyncIac ? permissionMessage('sync_iac') : undefined}
                    onClick={onRerunSyncTaskHistory}
                  >
                    重跑子周期
                  </Button>
                  <Button
                    type='link'
                    size='small'
                    icon={<SyncOutlined />}
                    loading={batchRerunningSyncTasks || batchRerunFailedSyncLoading}
                    disabled={!selectedFailedSyncTasks.length}
                    onClick={onBatchRerunFailedSyncTasks}
                  >
                    {selectedFailedSyncTasks.length ? `重跑失败任务(${selectedFailedSyncTasks.length})` : '重跑失败任务'}
                  </Button>
                  <Button type='link' size='small' onClick={clearSyncTaskQuery}>清除</Button>
                </Space>
              )}
              {showSyncTaskSummary && (
                <div className={styles.syncTaskSummary}>
                  <div className={styles.syncTaskMetric}>
                    <Text type='secondary'>总任务</Text>
                    <div className={styles.syncTaskMetricValue}>{numberText(syncTaskSummary.totalCount)}</div>
                  </div>
                  <div className={styles.syncTaskMetric}>
                    <Text type='secondary'>完成</Text>
                    <div className={styles.syncTaskMetricValue}>{numberText(syncTaskSummary.completeCount)}</div>
                  </div>
                  <div className={styles.syncTaskMetric}>
                    <Text type='secondary'>失败</Text>
                    <div className={styles.syncTaskMetricValue}>{numberText(syncTaskSummary.failedCount)}</div>
                  </div>
                  <div className={styles.syncTaskMetric}>
                    <Text type='secondary'>运行中</Text>
                    <div className={styles.syncTaskMetricValue}>{numberText(syncTaskSummary.runningCount)}</div>
                  </div>
                  <div className={styles.syncTaskMetric}>
                    <Text type='secondary'>成功率</Text>
                    <div className={styles.syncTaskMetricValue}>{percentText(syncTaskSummary.successRate)}</div>
                  </div>
                  <div className={styles.syncTaskMetric}>
                    <Text type='secondary'>失败率</Text>
                    <div className={styles.syncTaskMetricValue}>{percentText(syncTaskSummary.failureRate)}</div>
                  </div>
                  <div className={styles.syncTaskMetric}>
                    <Text type='secondary'>最近成功</Text>
                    <div className={styles.syncTaskMetricTime}>{renderTime(syncTaskSummary.lastSuccessAt)}</div>
                    {syncTaskSummary.lastSuccessTask && (
                      <Button type='link' size='small' onClick={() => openSyncTaskDetail(syncTaskSummary.lastSuccessTask)}>
                        查看任务
                      </Button>
                    )}
                  </div>
                  <div className={styles.syncTaskMetric}>
                    <Text type='secondary'>最近失败</Text>
                    <div className={styles.syncTaskMetricTime}>{renderTime(syncTaskSummary.lastFailureAt)}</div>
                    {syncTaskSummary.lastFailureTask && (
                      <Button type='link' size='small' onClick={() => openSyncTaskDetail(syncTaskSummary.lastFailureTask)}>
                        查看任务
                      </Button>
                    )}
                  </div>
                  {syncTaskSummary.failureThresholdExceeded && (
                    <Alert
                      className={styles.syncTaskFailureAlert}
                      type={syncTaskSummary.failureAlertLevel === 'error' ? 'error' : 'warning'}
                      showIcon={true}
                      message={syncTaskSummary.failureAlertMessage || `当前失败率已达到 ${percentText(syncTaskSummary.failureThreshold || syncTaskFailureThreshold)} 告警阈值`}
                    />
                  )}
                  <div className={styles.syncTaskTrend}>
                    <div className={styles.syncTaskTrendHeader}>
                      <Text strong={true}>{syncTaskTrendTitle}</Text>
                      <Space size={8} wrap={true}>
                        <Text type='secondary'>完成 / 失败 / 总数</Text>
                        <Text type='secondary'>阈值</Text>
                        <Select
                          size='small'
                          value={syncTaskFailureThreshold}
                          onChange={onSyncTaskFailureThresholdChange}
                          style={{ width: 88 }}
                        >
                          <Option value={30}>30%</Option>
                          <Option value={50}>50%</Option>
                          <Option value={80}>80%</Option>
                        </Select>
                        <Select
                          size='small'
                          value={syncTaskTrendDays}
                          onChange={onSyncTaskTrendDaysChange}
                          style={{ width: 88 }}
                        >
                          <Option value={7}>7天</Option>
                          <Option value={14}>14天</Option>
                          <Option value={30}>30天</Option>
                        </Select>
                        <Text type='secondary'>日期</Text>
                        <RangePicker
                          size='small'
                          allowClear={true}
                          format={trendDateFormat}
                          value={trendRangeMoments(syncTaskTrendRange)}
                          onChange={onSyncTaskTrendRangeChange}
                          style={{ width: 230 }}
                        />
                        <Button
                          size='small'
                          icon={<DownloadOutlined />}
                          onClick={exportSyncTaskTrend}
                        >
                          导出趋势
                        </Button>
                      </Space>
                    </div>
                    <div className={styles.syncTaskTrendGrid}>
                      {syncTaskTrend.map((point) => {
                        const completeCount = Number(point.completeCount || 0);
                        const failedCount = Number(point.failedCount || 0);
                        const totalCount = Number(point.totalCount || 0);
                        const completeHeight = completeCount ? Math.max(4, Math.round((completeCount / syncTaskTrendMax) * 36)) : 0;
                        const failedHeight = failedCount ? Math.max(4, Math.round((failedCount / syncTaskTrendMax) * 36)) : 0;
                        return (
                          <div className={styles.syncTaskTrendItem} key={point.date}>
                            <div className={styles.syncTaskTrendBars}>
                              <span className={styles.syncTaskTrendComplete} style={{ height: completeHeight }} />
                              <span className={styles.syncTaskTrendFailed} style={{ height: failedHeight }} />
                            </div>
                            <Text className={styles.syncTaskTrendDate}>{shortDate(point.date)}</Text>
                            <Text className={styles.syncTaskTrendCount}>{completeCount} / {failedCount} / {totalCount}</Text>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                  <div className={styles.syncTaskBreakdownGrid}>
                    <div className={styles.syncTaskBreakdownPanel}>
                      <div className={styles.syncTaskBreakdownTitle}>按区域拆分</div>
                      <Table
                        size='small'
                        rowKey='key'
                        columns={syncTaskRegionBreakdownColumns}
                        dataSource={syncTaskRegionBreakdown}
                        pagination={false}
                        scroll={{ x: 'max-content' }}
                        locale={{
                          emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无区域统计'/>
                        }}
                      />
                    </div>
                    <div className={styles.syncTaskBreakdownPanel}>
                      <div className={styles.syncTaskBreakdownTitle}>按资产类型拆分</div>
                      <Table
                        size='small'
                        rowKey='key'
                        columns={syncTaskAssetTypeBreakdownColumns}
                        dataSource={syncTaskAssetTypeBreakdown}
                        pagination={false}
                        scroll={{ x: 'max-content' }}
                        locale={{
                          emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无资产类型统计'/>
                        }}
                      />
                    </div>
                    <div className={styles.syncTaskBreakdownPanel}>
                      <div className={styles.syncTaskBreakdownTitle}>按 API 拆分（Top 10）</div>
                      <Table
                        size='small'
                        rowKey='key'
                        columns={syncTaskApiMetricBreakdownColumns}
                        dataSource={syncTaskApiMetricBreakdown}
                        pagination={false}
                        scroll={{ x: 'max-content' }}
                        locale={{
                          emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无 API 调用趋势'/>
                        }}
                      />
                    </div>
                  </div>
                </div>
              )}
              <div className={styles.toolbar}>
                <Space className={styles.filterBar} size={[8, 8]} wrap={true}>
                  <Select
                    allowClear={true}
                    loading={cloudAccountsLoading}
                    placeholder='云账号'
                    value={cloudForm.accountKey}
                    onChange={onCloudAccountChange}
                    style={{ minWidth: 260 }}
                  >
                    {(cloudAccounts || []).map((account) => (
                      <Option key={accountKey(account)} value={accountKey(account)}>
                        {account.provider} / {account.name} / {accountSourceMap[account.source] || account.source}{account.ready ? '' : ' / 缺凭证'}
                      </Option>
                    ))}
                  </Select>
                  <Select
                    mode='tags'
                    allowClear={true}
                    placeholder='区域'
                    value={cloudForm.regions}
                    onChange={(regions) => setCloudForm((pre) => ({ ...pre, regions }))}
                    style={{ minWidth: 220 }}
                  >
                    {(selectedCloudAccount && selectedCloudAccount.regions || []).map((region) => (
                      <Option key={region} value={region}>{region}</Option>
                    ))}
                  </Select>
                  <Select
                    mode='multiple'
                    allowClear={true}
                    placeholder='资产类型'
                    value={cloudForm.assetTypes}
                    onChange={(assetTypes) => setCloudForm((pre) => ({ ...pre, assetTypes }))}
                    style={{ minWidth: 260 }}
                  >
                    {(selectedCloudAccount && selectedCloudAccount.supportedAssetTypes || []).map((assetType) => (
                      <Option key={assetType} value={assetType}>{assetTypeMap[assetType] || assetType}</Option>
                    ))}
                  </Select>
                </Space>
                <Space className={styles.actionBar} size={[8, 8]} wrap={true}>
                  <Button icon={<ReloadOutlined />} onClick={() => {
                    fetchCloudAccounts();
                    refreshSyncTasks();
                  }}>刷新</Button>
                  <Button
                    type='primary'
                    icon={<SyncOutlined />}
                    disabled={!selectedCloudAccount || !canSyncIac}
                    title={!canSyncIac ? permissionMessage('sync_iac') : undefined}
                    loading={cloudSyncLoading || assetPermissionsLoading}
                    onClick={onStartCloudSync}
                  >
                    启动云采集
                  </Button>
                </Space>
              </div>
              <Table
                rowKey='id'
                columns={syncTaskColumns}
                scroll={{ x: 'min-content' }}
                loading={syncTasksLoading}
                dataSource={syncTaskList}
                rowSelection={{
                  selectedRowKeys: selectedFailedSyncTaskIds,
                  onChange: setSelectedFailedSyncTaskIds,
                  getCheckboxProps: (record) => ({
                    disabled: record.status !== 'failed'
                  })
                }}
                pagination={{
                  total: syncTasksData && syncTasksData.total,
                  pageSize: syncTasksData && syncTasksData.pageSize || 10,
                  showTotal: (total) => `共${total}条`
                }}
                onChange={({ current, pageSize }) => refreshSyncTasks({ currentPage: current, pageSize })}
              />
            </Space>
          </TabPane>
        </Tabs>
      </div>
      <Modal
        title='应用风险规则'
        visible={riskRuleVisible}
        width={900}
        confirmLoading={riskRuleSaving}
        okText='保存'
        cancelText='取消'
        destroyOnClose={false}
        okButtonProps={{ disabled: !canManageRiskRules }}
        onOk={saveRiskRuleConfig}
        onCancel={() => setRiskRuleVisible(false)}
      >
        <Form form={riskRuleForm} layout='vertical'>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 12 }}>
            <Form.Item name='changeWindowDays' label='变更窗口(天)' rules={[{ required: true, message: '请输入变更窗口' }]}>
              <InputNumber min={1} max={90} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='criticalIncomingThreshold' label='严重调用方阈值' rules={[{ required: true, message: '请输入严重阈值' }]}>
              <InputNumber min={1} max={100} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='mediumIncomingThreshold' label='中风险调用方阈值' rules={[{ required: true, message: '请输入中风险阈值' }]}>
              <InputNumber min={1} max={100} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='mediumOutgoingThreshold' label='中风险调用应用阈值' rules={[{ required: true, message: '请输入调用应用阈值' }]}>
              <InputNumber min={1} max={100} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='criticalScoreThreshold' label='严重评分阈值' rules={[{ required: true, message: '请输入严重评分阈值' }]}>
              <InputNumber min={1} max={1000} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='highScoreThreshold' label='高评分阈值' rules={[{ required: true, message: '请输入高评分阈值' }]}>
              <InputNumber min={1} max={1000} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='mediumScoreThreshold' label='中评分阈值' rules={[{ required: true, message: '请输入中评分阈值' }]}>
              <InputNumber min={1} max={1000} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='recentChangeWeight' label='近期变更权重'>
              <InputNumber min={0} max={100} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='changedAssetWeight' label='变更资产权重'>
              <InputNumber min={0} max={100} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='incomingAppWeight' label='调用方权重'>
              <InputNumber min={0} max={100} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='outgoingAppWeight' label='调用应用权重'>
              <InputNumber min={0} max={100} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='highComplianceRiskWeight' label='高合规风险权重'>
              <InputNumber min={0} max={100} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='criticalComplianceRiskWeight' label='严重合规风险权重'>
              <InputNumber min={0} max={100} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='maintenanceLifecycleWeight' label='维护期权重'>
              <InputNumber min={0} max={100} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='retiredLifecycleWeight' label='退役期权重'>
              <InputNumber min={0} max={100} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='crossBusinessLineWeight' label='跨业务线权重'>
              <InputNumber min={0} max={100} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='recentCriticalBoost' label='近期严重加分'>
              <InputNumber min={0} max={1000} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='recentHighBoost' label='近期高风险加分'>
              <InputNumber min={0} max={1000} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='recentMediumBoost' label='近期中风险加分'>
              <InputNumber min={0} max={1000} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='wideDependencyBoost' label='依赖面加分'>
              <InputNumber min={0} max={1000} style={{ width: '100%' }}/>
            </Form.Item>
          </div>
        </Form>
      </Modal>
      <Modal
        title='导入预检'
        visible={importPreviewVisible}
        width={980}
        confirmLoading={importingAssets}
        okText='确认导入'
        cancelText='取消'
        destroyOnClose={true}
        okButtonProps={{
          disabled: !canImportAssets || !importPreviewPayload || !importPreview || (
            (importPreview.created || 0) + (importPreview.updated || 0) + (importPreview.ownershipUpdated || 0)
          ) <= 0
        }}
        onOk={confirmImportAssets}
        onCancel={closeImportPreview}
      >
        <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
          <Alert
            type={importPreview && importPreview.errors && importPreview.errors.length ? 'warning' : 'info'}
            showIcon={true}
            message={`总数 ${importPreview && importPreview.total || 0}，新增 ${importPreview && importPreview.created || 0}，更新 ${importPreview && importPreview.updated || 0}，归属更新 ${importPreview && importPreview.ownershipUpdated || 0}，跳过 ${importPreview && importPreview.skipped || 0}`}
          />
          <Table
            rowKey={(record) => `${record.index}-${record.nativeId || record.action}`}
            size='small'
            columns={importPreviewColumns}
            dataSource={importPreview && importPreview.items || []}
            pagination={{ pageSize: 8 }}
            scroll={{ x: 1080 }}
          />
        </Space>
      </Modal>
      <Modal
        title={syncTaskRerunMode === 'failed' ? `重跑失败任务（${selectedFailedSyncTasks.length}）` : '重跑当前子周期'}
        visible={syncTaskRerunReasonVisible}
        confirmLoading={syncTaskRerunSubmitting || cloudSyncLoading || batchRerunningSyncTasks || batchRerunFailedSyncLoading}
        okText={syncTaskRerunMode === 'failed' && syncTaskRerunRequiresApproval ? '提交审批' : '启动重跑'}
        cancelText='取消'
        destroyOnClose={true}
        okButtonProps={{
          disabled: syncTaskRerunMode === 'schedule'
            ? !canSyncIac
            : (!canSyncIac && !syncTaskRerunRequiresApproval)
        }}
        onOk={onSubmitSyncTaskRerunReason}
        onCancel={closeSyncTaskRerunReason}
      >
        <Space direction='vertical' size='small' style={{ width: '100%', display: 'flex' }}>
          <Text type='secondary'>
            {syncTaskRerunMode === 'failed'
              ? '将按原失败任务的账号、区域、资产类型和子周期逐条创建新的采集任务。'
              : '将按当前同步策略子周期的历史任务范围创建新的采集任务。'}
          </Text>
          {syncTaskRerunMode === 'failed' && (
            <>
              <Space size='small' wrap={true} style={{ width: '100%' }}>
                <Select
                  mode='tags'
                  allowClear={true}
                  placeholder='覆盖区域'
                  value={syncTaskRerunRegions}
                  onChange={setSyncTaskRerunRegions}
                  style={{ minWidth: 220, flex: 1 }}
                >
                  {selectedFailedSyncTaskRegions.map((region) => (
                    <Option key={region} value={region}>{region}</Option>
                  ))}
                </Select>
                <Select
                  mode='tags'
                  allowClear={true}
                  placeholder='覆盖资产类型'
                  value={syncTaskRerunAssetTypes}
                  onChange={setSyncTaskRerunAssetTypes}
                  style={{ minWidth: 240, flex: 1 }}
                >
                  {selectedFailedSyncTaskAssetTypes.map((assetType) => (
                    <Option key={assetType} value={assetType}>{assetTypeMap[assetType] || assetType}</Option>
                  ))}
                </Select>
              </Space>
              <Checkbox
                checked={syncTaskRerunRequiresApproval}
                disabled={!canSyncIac}
                onChange={(event) => setSyncTaskRerunRequiresApproval(event.target.checked)}
              >
                提交审批后再启动
              </Checkbox>
              {!canSyncIac && (
                <Text type='secondary'>当前账号不能直接启动重跑，失败任务将以审批方式提交。</Text>
              )}
            </>
          )}
          <Input.TextArea
            autoSize={{ minRows: 3, maxRows: 5 }}
            maxLength={255}
            showCount={true}
            value={syncTaskRerunReason}
            onChange={(event) => setSyncTaskRerunReason(event.target.value)}
            placeholder='请输入重跑原因'
          />
        </Space>
      </Modal>
      <Modal
        title={`批量治理资产（${selectedAssetIds.length}）`}
        visible={batchOwnershipVisible}
        confirmLoading={batchOwnershipSaving}
        okText='保存'
        cancelText='取消'
        destroyOnClose={true}
        okButtonProps={{ disabled: !canBatchGovernance }}
        onOk={onSaveBatchOwnership}
        onCancel={closeBatchOwnership}
      >
        <Form form={batchOwnershipForm} layout='vertical'>
          <Form.Item name='projectId' label='绑定项目'>
            <Select allowClear={true} showSearch={true} optionFilterProp='children' placeholder='选择项目'>
              {(filters.projects || []).map((project) => (
                <Option key={project.projectId} value={project.projectId}>{project.projectName}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name='envId' label='绑定环境'>
            <Select allowClear={true} showSearch={true} optionFilterProp='children' placeholder='选择环境'>
              {(filters.envs || []).map((env) => (
                <Option key={env.envId} value={env.envId}>{env.envName}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name='clearProjectEnv' valuePropName='checked'>
            <Checkbox>清空项目/环境绑定</Checkbox>
          </Form.Item>
          <Form.Item name='owner' label='负责人'>
            <Input maxLength={128} placeholder='负责人'/>
          </Form.Item>
          <Form.Item name='application' label='应用'>
            <Input maxLength={128} placeholder='应用'/>
          </Form.Item>
          <Form.Item name='businessLine' label='业务线'>
            <Input maxLength={128} placeholder='业务线'/>
          </Form.Item>
          <Form.Item name='lifecycle' label='生命周期'>
            <Select allowClear={true} placeholder='生命周期'>
              {Object.entries(lifecycleMap).map(([value, label]) => (
                <Option key={value} value={value}>{label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name='complianceRisk' label='合规风险'>
            <Select allowClear={true} placeholder='合规风险'>
              {Object.entries(complianceRiskMap).map(([value, label]) => (
                <Option key={value} value={value}>{label}</Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title={`预检查${assetActionDryRunRecord && assetActionDryRunRecord.name || '云资产操作'}`}
        visible={assetActionDryRunVisible}
        confirmLoading={assetActionDryRunning && actionRequestKey === (assetActionDryRunRecord && assetActionDryRunRecord.key)}
        okText='预检查'
        cancelText='取消'
        destroyOnClose={true}
        onOk={onSubmitAssetActionDryRun}
        onCancel={closeAssetActionDryRun}
      >
        {assetActionDryRunRecord && (
          <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
            <Descriptions column={1} size='small' bordered={true}>
              <Descriptions.Item label='动作'>{assetActionDryRunRecord.key}</Descriptions.Item>
              <Descriptions.Item label='风险'>{renderActionRisk(assetActionDryRunRecord.riskLevel)}</Descriptions.Item>
              <Descriptions.Item label='执行方式'>{renderActionAdapterMode(assetActionDryRunRecord.adapterMode)}</Descriptions.Item>
              <Descriptions.Item label='目标资源'>{detail && (detail.nativeId || detail.id) || '-'}</Descriptions.Item>
            </Descriptions>
            <Form layout='vertical' className={styles.actionCreateForm}>
              <Form.Item label='参数 JSON' required={true}>
                <Input.TextArea
                  autoSize={{ minRows: 5, maxRows: 10 }}
                  value={assetActionDryRunParamsText}
                  onChange={(event) => setAssetActionDryRunParamsText(event.target.value)}
                  placeholder={assetActionParamsPlaceholder(assetActionDryRunRecord.key)}
                />
              </Form.Item>
            </Form>
          </Space>
        )}
      </Modal>
      <Modal
        title={`创建${assetActionCreateRecord && assetActionCreateRecord.name || '云资产操作'}任务`}
        visible={assetActionCreateVisible}
        confirmLoading={assetActionCreating && creatingActionKey === (assetActionCreateRecord && assetActionCreateRecord.key)}
        okText='创建任务'
        cancelText='取消'
        destroyOnClose={true}
        okButtonProps={{ disabled: createActionOkDisabled }}
        onOk={onSubmitAssetActionCreate}
        onCancel={closeAssetActionCreate}
      >
        {assetActionCreateRecord && (
          <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
            <Descriptions column={1} size='small' bordered={true}>
              <Descriptions.Item label='动作'>{assetActionCreateRecord.key}</Descriptions.Item>
              <Descriptions.Item label='风险'>{renderActionRisk(assetActionCreateRecord.riskLevel)}</Descriptions.Item>
              <Descriptions.Item label='执行方式'>{renderActionAdapterMode(assetActionCreateRecord.adapterMode)}</Descriptions.Item>
              <Descriptions.Item label='目标资源'>{createActionExpectedResourceId || '-'}</Descriptions.Item>
            </Descriptions>
            <Form layout='vertical' className={styles.actionCreateForm}>
              <Form.Item label='操作原因' required={!!createActionRequiresConfirm}>
                <Input.TextArea
                  maxLength={255}
                  autoSize={{ minRows: 3, maxRows: 5 }}
                  value={assetActionCreateReason}
                  onChange={(event) => setAssetActionCreateReason(event.target.value)}
                  placeholder={createActionRequiresConfirm ? '请输入操作原因' : '来自资产详情页'}
                />
              </Form.Item>
              {createActionRequiresConfirm && (
                <>
                  <Form.Item label={`确认动作：${assetActionCreateRecord.key}`} required={true}>
                    <Input
                      maxLength={64}
                      value={assetActionConfirmAction}
                      onChange={(event) => setAssetActionConfirmAction(event.target.value)}
                      placeholder={assetActionCreateRecord.key}
                    />
                  </Form.Item>
                  <Form.Item label={`确认资源 ID：${createActionExpectedResourceId}`} required={true}>
                    <Input
                      maxLength={255}
                      value={assetActionConfirmResourceId}
                      onChange={(event) => setAssetActionConfirmResourceId(event.target.value)}
                      placeholder={createActionExpectedResourceId}
                    />
                  </Form.Item>
                </>
              )}
              {createActionRequiresTags && (
                <Form.Item label='标签 JSON' required={true}>
                  <Input.TextArea
                    autoSize={{ minRows: 5, maxRows: 10 }}
                    value={assetActionTagsText}
                    onChange={(event) => setAssetActionTagsText(event.target.value)}
                    placeholder='{"Environment":"dev","Owner":"team-a"}'
                  />
                </Form.Item>
              )}
              {createActionRequiresParams && (
                <Form.Item label='参数 JSON' required={true}>
                  <Input.TextArea
                    autoSize={{ minRows: 5, maxRows: 10 }}
                    value={assetActionParamsText}
                    onChange={(event) => setAssetActionParamsText(event.target.value)}
                    placeholder={assetActionParamsPlaceholder(assetActionCreateRecord.key)}
                  />
                </Form.Item>
              )}
            </Form>
          </Space>
        )}
      </Modal>
      <Drawer
        title='批量重跑任务组'
        width={980}
        visible={syncTaskRerunGroupVisible}
        onClose={closeSyncTaskRerunGroup}
        destroyOnClose={true}
      >
        {syncTaskRerunGroupLoading || !syncTaskRerunGroup ? (
          <Text type='secondary'>加载中...</Text>
        ) : (
          <Space direction='vertical' size='middle' style={{ width: '100%' }}>
            <Descriptions column={2} size='small' bordered={true}>
              <Descriptions.Item label='任务组 ID' span={2}>
                <Space size={8}>
                  <Text code={true}>{syncTaskRerunGroup.groupId || '-'}</Text>
                  <Button
                    type='link'
                    size='small'
                    disabled={!syncTaskRerunGroup.groupId}
                    onClick={() => copy(syncTaskRerunGroup.groupId)}
                  >
                    复制
                  </Button>
                  {canApproveSyncTaskRerunGroup && (
                    <>
                      <Button
                        type='primary'
                        size='small'
                        loading={syncTaskRerunGroupApproving}
                        onClick={() => onApproveSyncTaskRerunGroup('approved')}
                      >
                        审批通过
                      </Button>
                      <Button
                        size='small'
                        danger={true}
                        loading={syncTaskRerunGroupApproving}
                        onClick={() => onApproveSyncTaskRerunGroup('rejected')}
                      >
                        驳回
                      </Button>
                    </>
                  )}
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label='重跑模式'>
                {syncTaskRerunModeTextMap[syncTaskRerunGroup.mode] || syncTaskRerunGroup.mode || '-'}
              </Descriptions.Item>
              <Descriptions.Item label='审批状态'>
                {syncTaskRerunGroupApproval.status ? (
                  <Tag color={syncTaskRerunGroupApproval.status === 'pending' ? 'warning' : syncTaskRerunGroupApproval.status === 'approved' ? 'success' : 'default'}>
                    {syncTaskRerunGroupApproval.status === 'pending' ? '待审批' : syncTaskRerunGroupApproval.status === 'approved' ? '已通过' : '已驳回'}
                  </Tag>
                ) : '-'}
              </Descriptions.Item>
              <Descriptions.Item label='任务总数'>{syncTaskRerunGroup.total || 0}</Descriptions.Item>
              <Descriptions.Item label='待审批'>{syncTaskRerunGroup.approvingCount || 0}</Descriptions.Item>
              <Descriptions.Item label='完成'>{syncTaskRerunGroup.completeCount || 0}</Descriptions.Item>
              <Descriptions.Item label='失败'>{syncTaskRerunGroup.failedCount || 0}</Descriptions.Item>
              <Descriptions.Item label='已驳回'>{syncTaskRerunGroup.rejectedCount || 0}</Descriptions.Item>
              <Descriptions.Item label='运行中'>{syncTaskRerunGroup.runningCount || 0}</Descriptions.Item>
              <Descriptions.Item label='等待中'>{syncTaskRerunGroup.pendingCount || 0}</Descriptions.Item>
              <Descriptions.Item label='创建时间'>{renderTime(syncTaskRerunGroup.createdAt)}</Descriptions.Item>
              <Descriptions.Item label='最近结束'>{renderTime(syncTaskRerunGroup.endedAt)}</Descriptions.Item>
              <Descriptions.Item label='重跑原因' span={2}>{syncTaskRerunGroup.reason || '-'}</Descriptions.Item>
            </Descriptions>
            <Table
              size='small'
              rowKey='id'
              columns={syncTaskRerunGroupColumns}
              dataSource={syncTaskRerunGroup.tasks || []}
              pagination={false}
              scroll={{ x: 'max-content' }}
              locale={{
                emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无任务'/>
              }}
            />
          </Space>
        )}
      </Drawer>
      <Drawer
        title='云采集任务详情'
        width={860}
        visible={syncTaskDetailVisible}
        onClose={closeSyncTaskDetail}
        destroyOnClose={true}
      >
        {syncTaskDetailLoading || !syncTaskDetail ? (
          <Text type='secondary'>加载中...</Text>
        ) : (
          <Space direction='vertical' size='middle' style={{ width: '100%' }}>
            <Descriptions column={2} size='small' bordered={true}>
              <Descriptions.Item label='任务 ID' span={2}>
                <Space size={8}>
                  <Text code={true}>{syncTaskDetail.id || '-'}</Text>
                  <Button
                    type='link'
                    size='small'
                    disabled={!syncTaskDetail.id}
                    onClick={() => copy(syncTaskDetail.id)}
                  >
                    复制
                  </Button>
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label='账号'>{syncTaskDetail.accountName || '-'}</Descriptions.Item>
              <Descriptions.Item label='来源'>{accountSourceMap[syncTaskDetail.accountSource] || syncTaskDetail.accountSource || '-'}</Descriptions.Item>
              <Descriptions.Item label='云厂商'>{syncTaskDetail.provider || '-'}</Descriptions.Item>
              <Descriptions.Item label='状态'>
                <Tag color={taskStatusColorMap[syncTaskDetail.status]}>
                  {taskStatusMap[syncTaskDetail.status] || syncTaskDetail.status || '-'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label='区域'>{joinList(syncTaskDetail.regions)}</Descriptions.Item>
              <Descriptions.Item label='资产类型'>
                {joinList((syncTaskDetail.assetTypes || []).map((it) => assetTypeMap[it] || it))}
              </Descriptions.Item>
              <Descriptions.Item label='采集范围' span={2}>{renderSyncTaskScopeSummary(syncTaskDetail)}</Descriptions.Item>
              <Descriptions.Item label='采集耗时'>{durationText(syncTaskDetailScope.collectorDurationMs)}</Descriptions.Item>
              <Descriptions.Item label='总耗时'>{durationText(syncTaskDetailScope.durationMs)}</Descriptions.Item>
              <Descriptions.Item label='开始时间'>{renderTime(syncTaskDetail.startedAt)}</Descriptions.Item>
              <Descriptions.Item label='结束时间'>{renderTime(syncTaskDetail.endedAt)}</Descriptions.Item>
              <Descriptions.Item label='失败摘要' span={2}>{renderSyncTaskFailureSummary(syncTaskDetail)}</Descriptions.Item>
              <Descriptions.Item label='API调用' span={2}>
                {syncTaskDetailApiMetrics.length > 0 ? (
                  <Space direction='vertical' size={0}>
                    <Text>
                      总数 {numberText(syncTaskDetailApiSummary.total || syncTaskDetailApiMetrics.length)}
                      ，失败 {numberText(syncTaskDetailApiSummary.failed)}
                      ，重试 {numberText(syncTaskDetailApiSummary.retried)}
                      ，最大尝试 {numberText(syncTaskDetailApiSummary.maxAttempts)}
                      ，最大耗时 {durationText(syncTaskDetailApiSummary.maxDurationMs)}
                    </Text>
                    <Text type='secondary'>最慢 API：{syncTaskDetailApiSummary.slowestEndpoint || '-'}</Text>
                  </Space>
                ) : '-'}
              </Descriptions.Item>
              <Descriptions.Item label='重试建议' span={2}>{syncTaskDetailFailure.retryHint || '-'}</Descriptions.Item>
              <Descriptions.Item label='错误日志' span={2}>{syncTaskDetail.errorMessage || '-'}</Descriptions.Item>
              <Descriptions.Item label='重跑原因' span={2}>{(syncTaskDetail.stats || {}).reason || '-'}</Descriptions.Item>
              <Descriptions.Item label='重跑任务组' span={2}>
                {(syncTaskDetail.stats || {}).rerunGroupId ? (
                  <Space size={8}>
                    <Button type='link' size='small' onClick={() => openSyncTaskRerunGroup((syncTaskDetail.stats || {}).rerunGroupId)}>
                      {(syncTaskDetail.stats || {}).rerunGroupId}
                    </Button>
                    <Button type='link' size='small' onClick={() => copy((syncTaskDetail.stats || {}).rerunGroupId)}>
                      复制
                    </Button>
                  </Space>
                ) : '-'}
              </Descriptions.Item>
              <Descriptions.Item label='源任务' span={2}>
                {(syncTaskDetail.stats || {}).rerunFromTaskId ? (
                  <Button type='link' size='small' onClick={() => openSyncTaskDetail({ id: (syncTaskDetail.stats || {}).rerunFromTaskId })}>
                    {(syncTaskDetail.stats || {}).rerunFromTaskId}
                  </Button>
                ) : '-'}
              </Descriptions.Item>
              <Descriptions.Item label='重跑模式' span={2}>
                {syncTaskRerunModeTextMap[(syncTaskDetail.stats || {}).rerunMode] || (syncTaskDetail.stats || {}).rerunMode || '-'}
              </Descriptions.Item>
              <Descriptions.Item label='参数差异' span={2}>
                {renderRerunParameterDiffs(syncTaskDetail.stats || {})}
              </Descriptions.Item>
              <Descriptions.Item label='审批状态' span={2}>
                {((syncTaskDetail.stats || {}).approval || {}).status ? (
                  <Space size={8}>
                    <Tag color={((syncTaskDetail.stats || {}).approval || {}).status === 'pending' ? 'warning' : ((syncTaskDetail.stats || {}).approval || {}).status === 'approved' ? 'success' : 'default'}>
                      {((syncTaskDetail.stats || {}).approval || {}).status === 'pending' ? '待审批' : ((syncTaskDetail.stats || {}).approval || {}).status === 'approved' ? '已通过' : '已驳回'}
                    </Tag>
                    {((syncTaskDetail.stats || {}).approval || {}).comment && (
                      <Text type='secondary'>{((syncTaskDetail.stats || {}).approval || {}).comment}</Text>
                    )}
                  </Space>
                ) : '-'}
              </Descriptions.Item>
            </Descriptions>
            <Tabs defaultActiveKey='logs'>
              <TabPane tab='阶段日志' key='logs'>
                <Table
                  size='small'
                  rowKey='id'
                  columns={syncTaskLogColumns}
                  dataSource={syncTaskDetail.logs || []}
                  pagination={false}
                  scroll={{ x: 'max-content' }}
                  locale={{
                    emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无阶段日志'/>
                  }}
                />
              </TabPane>
              <TabPane tab='范围明细' key='scope'>
                <Table
                  size='small'
                  rowKey={(record, index) => `${record.region || '-'}-${record.assetType || '-'}-${index}`}
                  columns={syncTaskScopeMetricColumns}
                  dataSource={syncTaskDetailScopeMetrics}
                  pagination={false}
                  scroll={{ x: 'max-content' }}
                  locale={{
                    emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无范围明细'/>
                  }}
                />
              </TabPane>
              <TabPane tab='失败明细' key='failures'>
                <Table
                  size='small'
                  rowKey={(record, index) => `${record.category || '-'}-${index}`}
                  columns={syncTaskFailureDetailColumns}
                  dataSource={syncTaskDetailFailureDetails}
                  pagination={false}
                  scroll={{ x: 'max-content' }}
                  locale={{
                    emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无失败明细'/>
                  }}
                />
              </TabPane>
              <TabPane tab={`API调用${syncTaskDetailApiMetrics.length ? `(${syncTaskDetailApiMetrics.length})` : ''}`} key='apiMetrics'>
                <Space direction='vertical' size='small' style={{ width: '100%', display: 'flex' }}>
                  <Space size={[8, 8]} wrap={true}>
                    <Select
                      mode='multiple'
                      allowClear={true}
                      placeholder='区域'
                      value={syncTaskApiMetricFilters.regions}
                      onChange={(regions) => setSyncTaskApiMetricFilters((prev) => ({ ...prev, regions }))}
                      style={{ minWidth: 160 }}
                    >
                      {syncTaskApiMetricFilterOptions.regions.map((region) => (
                        <Option key={region} value={region}>{region}</Option>
                      ))}
                    </Select>
                    <Select
                      mode='multiple'
                      allowClear={true}
                      placeholder='服务'
                      value={syncTaskApiMetricFilters.services}
                      onChange={(services) => setSyncTaskApiMetricFilters((prev) => ({ ...prev, services }))}
                      style={{ minWidth: 140 }}
                    >
                      {syncTaskApiMetricFilterOptions.services.map((service) => (
                        <Option key={service} value={service}>{service}</Option>
                      ))}
                    </Select>
                    <Select
                      mode='multiple'
                      allowClear={true}
                      placeholder='状态'
                      value={syncTaskApiMetricFilters.statuses}
                      onChange={(statuses) => setSyncTaskApiMetricFilters((prev) => ({ ...prev, statuses }))}
                      style={{ minWidth: 140 }}
                    >
                      {syncTaskApiMetricFilterOptions.statuses.map((status) => (
                        <Option key={status} value={status}>{syncTaskMetricStatusMap[status] || status}</Option>
                      ))}
                    </Select>
                    <Checkbox
                      checked={syncTaskApiMetricFilters.slowOnly}
                      onChange={(event) => setSyncTaskApiMetricFilters((prev) => ({ ...prev, slowOnly: event.target.checked }))}
                    >
                      只看慢调用
                    </Checkbox>
                    <InputNumber
                      min={1}
                      step={500}
                      disabled={!syncTaskApiMetricFilters.slowOnly}
                      value={syncTaskApiMetricFilters.slowThresholdMs}
                      onChange={(slowThresholdMs) => setSyncTaskApiMetricFilters((prev) => ({
                        ...prev,
                        slowThresholdMs: slowThresholdMs || defaultSyncTaskApiMetricFilters.slowThresholdMs
                      }))}
                      style={{ width: 110 }}
                    />
                    <Text type='secondary'>ms 以上</Text>
                    <Button
                      size='small'
                      onClick={() => setSyncTaskApiMetricFilters({
                        ...defaultSyncTaskApiMetricFilters,
                        slowThresholdMs: syncTaskSlowApiThresholdMs(syncTaskDetail)
                      })}
                    >
                      重置
                    </Button>
                    <Button
                      size='small'
                      icon={<DownloadOutlined />}
                      disabled={!filteredSyncTaskDetailApiMetrics.length}
                      onClick={exportSyncTaskApiMetrics}
                    >
                      导出CSV
                    </Button>
                    <Text type='secondary'>显示 {numberText(filteredSyncTaskDetailApiMetrics.length)} / {numberText(syncTaskDetailApiMetrics.length)}</Text>
                  </Space>
                  {syncTaskApiSlowSummary.total > 0 && (
                    <Alert
                      type={syncTaskApiSlowSummary.failed > 0 ? 'error' : 'warning'}
                      showIcon={true}
                      message={`慢调用告警：${numberText(syncTaskApiSlowSummary.total)} 个 API 调用耗时不低于 ${durationText(syncTaskApiSlowSummary.thresholdMs)}`}
                      description={`失败 ${numberText(syncTaskApiSlowSummary.failed)} 个，发生重试 ${numberText(syncTaskApiSlowSummary.retried)} 个，最慢 API：${[
                        syncTaskApiSlowSummary.slowest && syncTaskApiSlowSummary.slowest.region,
                        syncTaskApiSlowSummary.slowest && syncTaskApiSlowSummary.slowest.service,
                        syncTaskApiSlowSummary.slowest && syncTaskApiSlowSummary.slowest.path
                      ].filter(Boolean).join(' ') || '-'}，耗时 ${durationText(syncTaskApiSlowSummary.slowest && syncTaskApiSlowSummary.slowest.durationMs)}`}
                    />
                  )}
                  <Table
                    size='small'
                    rowKey={(record, index) => record.metricKey || `${record.region || '-'}-${record.service || '-'}-${record.path || '-'}-${index}`}
                    columns={syncTaskApiMetricColumns}
                    dataSource={filteredSyncTaskDetailApiMetrics}
                    pagination={filteredSyncTaskDetailApiMetrics.length > 20 ? { pageSize: 20, showSizeChanger: true } : false}
                    scroll={{ x: 'max-content' }}
                    locale={{
                      emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={syncTaskDetailApiMetrics.length ? '暂无匹配的 API 调用指标' : '暂无 API 调用指标'}/>
                    }}
                  />
                </Space>
              </TabPane>
              <TabPane tab='统计' key='stats'>
                {renderJSON(syncTaskDetail.stats)}
              </TabPane>
            </Tabs>
          </Space>
        )}
      </Drawer>
      <Drawer
        title='应用依赖详情'
        width={820}
        visible={applicationDetailVisible}
        onClose={closeApplicationDetail}
        destroyOnClose={true}
      >
        {applicationDetailLoading || !applicationDetail ? (
          <Text type='secondary'>加载中...</Text>
        ) : (
          <Space direction='vertical' size='middle' style={{ width: '100%' }}>
            <Descriptions column={2} size='small' bordered={true}>
              <Descriptions.Item label='应用'>{applicationDetail.application || '-'}</Descriptions.Item>
              <Descriptions.Item label='变更风险'>{renderApplicationRisk(applicationDetail.riskLevel)}</Descriptions.Item>
              <Descriptions.Item label='绑定资源'>{applicationDetail.assetCount || 0}</Descriptions.Item>
              <Descriptions.Item label={`近${riskWindowDays}天变更`}>{applicationDetail.recentChangeCount || 0}</Descriptions.Item>
              <Descriptions.Item label='调用方'>{applicationDetail.incomingAppCount || 0}</Descriptions.Item>
              <Descriptions.Item label='调用应用'>{applicationDetail.outgoingAppCount || 0}</Descriptions.Item>
              <Descriptions.Item label='影响应用'>{applicationDetail.impactedAppCount || 0}</Descriptions.Item>
              <Descriptions.Item label='负责人'>{renderCompactList(applicationDetail.owner)}</Descriptions.Item>
              <Descriptions.Item label='业务线'>{renderCompactList(applicationDetail.businessLine)}</Descriptions.Item>
              <Descriptions.Item label='风险原因' span={2}>{applicationDetail.riskReason || '-'}</Descriptions.Item>
            </Descriptions>
            <ApplicationGraph
              application={applicationDetail}
              onOpenApplication={openApplicationDetail}
            />
            <Tabs activeKey={applicationDetailTab} onChange={setApplicationDetailTab}>
              <TabPane tab='影响分析' key='impact'>
                <Table
                  size='small'
                  rowKey={(record) => `impact-${record.application}`}
                  columns={applicationImpactColumns}
                  dataSource={applicationDetail.impactedApplications || []}
                  pagination={false}
                  scroll={{ x: 'max-content' }}
                  locale={{
                    emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无影响应用'/>
                  }}
                />
              </TabPane>
              <TabPane tab='关系' key='relations'>
                <Text strong={true}>调用本应用</Text>
                <Table
                  size='small'
                  rowKey={(record) => `upstream-${record.application}-${record.source}-${record.relationType}`}
                  columns={applicationRelationColumns}
                  dataSource={applicationDetail.upstreams || []}
                  pagination={false}
                  scroll={{ x: 'max-content' }}
                  style={{ marginTop: 8, marginBottom: 16 }}
                />
                <Text strong={true}>本应用调用</Text>
                <Table
                  size='small'
                  rowKey={(record) => `downstream-${record.application}-${record.source}-${record.relationType}`}
                  columns={applicationRelationColumns}
                  dataSource={applicationDetail.downstreams || []}
                  pagination={false}
                  scroll={{ x: 'max-content' }}
                  style={{ marginTop: 8 }}
                />
              </TabPane>
              <TabPane tab='依赖维护' key='manageRelations'>
                <Form layout='vertical' className={styles.ownershipForm}>
                  <Form.Item label='调用本应用'>
                    <Select
                      mode='tags'
                      allowClear={true}
                      maxTagCount='responsive'
                      placeholder='选择调用方应用'
                      value={applicationRelationForm.upstreams}
                      onChange={(upstreams) => setApplicationRelationForm((pre) => ({ ...pre, upstreams }))}
                      style={{ width: '100%' }}
                    >
                      {applicationOptions
                        .filter((name) => name !== applicationDetail.application)
                        .map((name) => <Option key={name} value={name}>{name}</Option>)}
                    </Select>
                  </Form.Item>
                  <Form.Item label='本应用调用'>
                    <Select
                      mode='tags'
                      allowClear={true}
                      maxTagCount='responsive'
                      placeholder='选择被调用应用'
                      value={applicationRelationForm.downstreams}
                      onChange={(downstreams) => setApplicationRelationForm((pre) => ({ ...pre, downstreams }))}
                      style={{ width: '100%' }}
                    >
                      {applicationOptions
                        .filter((name) => name !== applicationDetail.application)
                        .map((name) => <Option key={name} value={name}>{name}</Option>)}
                    </Select>
                  </Form.Item>
                  <div className={styles.ownershipActions}>
                    <Button
                      type='primary'
                      icon={<SaveOutlined />}
                      disabled={!canManageApplicationRelations}
                      title={!canManageApplicationRelations ? permissionMessage('application_relations') : undefined}
                      loading={applicationRelationSaving}
                      onClick={onSaveApplicationRelations}
                    >
                      保存
                    </Button>
                  </div>
                </Form>
              </TabPane>
              <TabPane tab='绑定资源' key='assets'>
                <Space direction='vertical' size='small' style={{ width: '100%', display: 'flex' }}>
                  <Button onClick={() => showApplicationAssets(applicationDetail.application)}>
                    查看资产列表
                  </Button>
                  <Table
                    size='small'
                    rowKey='id'
                    columns={applicationAssetColumns}
                    dataSource={applicationDetail.assets || []}
                    pagination={false}
                    scroll={{ x: 'max-content' }}
                  />
                </Space>
              </TabPane>
              <TabPane tab='近期变更' key='changes'>
                <Table
                  size='small'
                  rowKey='id'
                  columns={applicationChangeColumns}
                  dataSource={applicationDetail.recentChanges || []}
                  pagination={false}
                  scroll={{ x: 'max-content' }}
                />
              </TabPane>
            </Tabs>
          </Space>
        )}
      </Drawer>
      <Drawer
        title='资产详情'
        width={720}
        visible={detailVisible}
        onClose={closeDetail}
        destroyOnClose={true}
      >
        {detailLoading || !detail ? (
          <Text type='secondary'>加载中...</Text>
        ) : (
          <Space direction='vertical' size='middle' style={{ width: '100%' }}>
            <Descriptions column={2} size='small' bordered={true}>
              <Descriptions.Item label='名称'>{detail.name || '-'}</Descriptions.Item>
              <Descriptions.Item label='资产类型'>{assetTypeMap[detail.assetType] || detail.assetType || '-'}</Descriptions.Item>
              <Descriptions.Item label='云厂商'>{detail.provider || '-'}</Descriptions.Item>
              <Descriptions.Item label='原生类型'>{detail.nativeType || '-'}</Descriptions.Item>
              <Descriptions.Item label='资源ID' span={2}>{detail.nativeId || '-'}</Descriptions.Item>
              <Descriptions.Item label='项目'>{detail.projectName || '-'}</Descriptions.Item>
              <Descriptions.Item label='环境'>
                {detail.projectId && detail.envId ? (
                  <Link to={`/org/${orgId}/project/${detail.projectId}/m-project-env/detail/${detail.envId}?tabKey=resource`}>
                    {detail.envName || detail.envId}
                  </Link>
                ) : detail.envName || '-'}
              </Descriptions.Item>
              <Descriptions.Item label='来源资源' span={2}>
                {envResourceDetailUrl(orgId, detail) ? (
                  <Link to={envResourceDetailUrl(orgId, detail)}>
                    <Button size='small' type='link'>返回环境资源详情</Button>
                  </Link>
                ) : '-'}
              </Descriptions.Item>
              <Descriptions.Item label='区域'>{detail.region || '-'}</Descriptions.Item>
              <Descriptions.Item label='可用区'>{detail.zone || '-'}</Descriptions.Item>
              <Descriptions.Item label='公网IP'>{detail.publicIp || '-'}</Descriptions.Item>
              <Descriptions.Item label='私网IP'>{detail.privateIp || '-'}</Descriptions.Item>
              <Descriptions.Item label='状态'>{detail.status || '-'}</Descriptions.Item>
              <Descriptions.Item label='来源'>{sourceMap[detail.source] || detail.source || '-'}</Descriptions.Item>
              <Descriptions.Item label='负责人'>{detail.owner || '-'}</Descriptions.Item>
              <Descriptions.Item label='应用'>{detail.application || '-'}</Descriptions.Item>
              <Descriptions.Item label='业务线'>{detail.businessLine || '-'}</Descriptions.Item>
              <Descriptions.Item label='合规风险'>{renderComplianceRisk(detail.complianceRisk)}</Descriptions.Item>
              <Descriptions.Item label='生命周期'>{renderLifecycle(detail.lifecycle)}</Descriptions.Item>
              <Descriptions.Item label='成本'>{renderCost(detail.cost)}</Descriptions.Item>
              {isKubernetesAsset(detail) && (
                <>
                  <Descriptions.Item label='K8S版本'>{assetAttrValue(detail, ['version', 'kubernetesVersion']) || '-'}</Descriptions.Item>
                  <Descriptions.Item label='节点组/池'>{kubernetesNodeSummary(detail)}</Descriptions.Item>
                  <Descriptions.Item label='工作负载层'>{kubernetesWorkloadSummary(detail)}</Descriptions.Item>
                  <Descriptions.Item label='API Endpoint' span={2}>{kubernetesEndpointText(detail)}</Descriptions.Item>
                </>
              )}
              <Descriptions.Item label='IaC地址' span={2}>{detail.iacAddress || '-'}</Descriptions.Item>
              <Descriptions.Item label='最近同步' span={2}>{renderTime(detail.lastSyncAt)}</Descriptions.Item>
              <Descriptions.Item label='最近操作' span={2}>
                {detail.lastOperationId ? (
                  <Link to={`/org/${orgId}/m-cloud-operations?operationId=${detail.lastOperationId}`}>
                    {detail.lastOperationId}
                  </Link>
                ) : '-'}
              </Descriptions.Item>
            </Descriptions>
            <Tabs defaultActiveKey={isKubernetesAsset(detail) ? 'kubernetes' : 'ownership'}>
              <TabPane tab='归属' key='ownership'>
                <Form form={ownershipForm} layout='vertical' className={styles.ownershipForm}>
                  <div className={styles.ownershipGrid}>
                    <Form.Item name='owner' label='负责人'>
                      <Input maxLength={128} placeholder='负责人'/>
                    </Form.Item>
                    <Form.Item name='application' label='应用'>
                      <Input maxLength={128} placeholder='应用'/>
                    </Form.Item>
                    <Form.Item name='businessLine' label='业务线'>
                      <Input maxLength={128} placeholder='业务线'/>
                    </Form.Item>
                    <Form.Item name='lifecycle' label='生命周期'>
                      <Select allowClear={true} placeholder='生命周期'>
                        {Object.entries(lifecycleMap).map(([value, label]) => (
                          <Option key={value} value={value}>{label}</Option>
                        ))}
                      </Select>
                    </Form.Item>
                    <Form.Item name='cost' label='成本'>
                      <InputNumber min={0} precision={2} style={{ width: '100%' }} placeholder='成本'/>
                    </Form.Item>
                    <Form.Item name='complianceRisk' label='合规风险'>
                      <Select allowClear={true} placeholder='合规风险'>
                        {Object.entries(complianceRiskMap).map(([value, label]) => (
                          <Option key={value} value={value}>{label}</Option>
                        ))}
                      </Select>
                    </Form.Item>
                  </div>
                  <div className={styles.ownershipActions}>
                    <Button
                      type='primary'
                      icon={<SaveOutlined />}
                      disabled={!canEditDetailOwnership}
                      title={!canEditDetailOwnership ? permissionMessage('edit_ownership') : undefined}
                      loading={ownershipSaving}
                      onClick={onSaveOwnership}
                    >
                      保存
                    </Button>
                  </div>
                </Form>
              </TabPane>
              {isCloudAssets && (
                <TabPane tab='操作' key='actions'>
                  <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
                    <Table
                      size='small'
                      rowKey='key'
                      columns={assetActionColumns}
                      dataSource={assetActions || []}
                      loading={assetActionsLoading}
                      pagination={false}
                      scroll={{ x: 'max-content' }}
                      locale={{
                        emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无可用动作'/>
                      }}
                    />
                    {assetActionDryRun && (
                      <div className={styles.actionPreview}>
                        <Descriptions size='small' column={2} bordered={true}>
                          <Descriptions.Item label='动作'>{assetActionDryRun.name || assetActionDryRun.action || '-'}</Descriptions.Item>
                          <Descriptions.Item label='风险'>{renderActionRisk(assetActionDryRun.riskLevel)}</Descriptions.Item>
                          <Descriptions.Item label='执行方式'>{renderActionAdapterMode(assetActionDryRun.adapterMode)}</Descriptions.Item>
                          <Descriptions.Item label='适配器状态'>{renderActionAdapterStatus(assetActionDryRun.adapterStatus)}</Descriptions.Item>
                          <Descriptions.Item label='Provider Adapter'>{assetActionDryRun.providerAdapter || '-'}</Descriptions.Item>
                          <Descriptions.Item label='是否可执行'>
                            {assetActionDryRun.executable ? <Tag color='success'>可执行</Tag> : <Tag color='error'>不可执行</Tag>}
                          </Descriptions.Item>
                          <Descriptions.Item label='审批'>
                            {assetActionDryRun.requiresApproval ? <Tag color='warning'>需要审批</Tag> : <Tag>无需审批</Tag>}
                          </Descriptions.Item>
                          <Descriptions.Item label='原因' span={2}>
                            {assetActionDryRun.disabledReason || '-'}
                          </Descriptions.Item>
                        </Descriptions>
                        <Table
                          style={{ marginTop: 12 }}
                          size='small'
                          rowKey='key'
                          columns={assetActionCheckColumns}
                          dataSource={assetActionDryRun.checks || []}
                          pagination={false}
                        />
                      </div>
                    )}
                  </Space>
                </TabPane>
              )}
              {isKubernetesAsset(detail) && (
                <TabPane tab='K8S信息' key='kubernetes'>
                  <KubernetesAssetInfo asset={detail}/>
                </TabPane>
              )}
              <TabPane tab='属性' key='attributes'>
                {renderJSON(detail.attributes)}
              </TabPane>
              <TabPane tab='标签' key='tags'>
                {renderJSON(detail.tags)}
              </TabPane>
              {isCloudAssets && (
                <TabPane tab='安全规则' key='securityRules'>
                  <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
                    <Descriptions size='small' bordered={true} column={3}>
                      <Descriptions.Item label='规则数'>{securityRules.ruleCount || 0}</Descriptions.Item>
                      <Descriptions.Item label='公网规则'>{securityRules.publicRuleCount || 0}</Descriptions.Item>
                      <Descriptions.Item label='资产类型'>{assetTypeMap[securityRules.assetType] || securityRules.assetType || '-'}</Descriptions.Item>
                    </Descriptions>
                    <Table
                      size='small'
                      rowKey={(record, index) => `${record.direction}-${index}`}
                      columns={securityRuleColumns}
                      dataSource={securityRules.rules || []}
                      loading={securityRulesLoading}
                      pagination={false}
                      scroll={{ x: 'max-content' }}
                    />
                  </Space>
                </TabPane>
              )}
              <TabPane tab='关系' key='relations'>
                {activeRelationSummary && (
                  <Alert
                    showIcon={true}
                    type={activeRelationSummary.truncated ? 'warning' : 'info'}
                    message={`已加载 ${numberText(loadedRelationCount)} / ${numberText(activeRelationSummary.totalRelationCount)} 条关系`}
                    description={`本次返回 ${numberText(activeRelationSummary.returnedRelationCount)} 条，直接关系 ${numberText(activeRelationSummary.directRelationCount)}，应用推演 ${numberText(activeRelationSummary.applicationInferredRelationCount)}，入向 ${numberText(activeRelationSummary.incomingRelationCount)}，出向 ${numberText(activeRelationSummary.outgoingRelationCount)}${activeRelationSummary.truncated ? `；当前每页最多加载 ${numberText(activeRelationSummary.limit)} 条` : ''}`}
                    style={{ marginBottom: 12 }}
                  />
                )}
                <div className={styles.relationFilterBar}>
                  <InputSearch
                    allowClear={true}
                    placeholder='搜索关联资产、ID、类型或来源'
                    value={relationFilters.keyword}
                    onChange={(event) => updateRelationFilter('keyword', event.target.value)}
                    onSearch={(keyword) => updateRelationFilter('keyword', keyword)}
                    style={{ width: 260 }}
                  />
                  <Select
                    mode='multiple'
                    allowClear={true}
                    maxTagCount='responsive'
                    placeholder='关系来源'
                    value={relationFilters.sources}
                    onChange={(sources) => updateRelationFilter('sources', sources)}
                    style={{ minWidth: 180 }}
                  >
                    {relationSourceOptions.map((source) => (
                      <Option key={source} value={source}>{relationSourceLabel(source)}</Option>
                    ))}
                  </Select>
                  <Select
                    mode='multiple'
                    allowClear={true}
                    maxTagCount='responsive'
                    placeholder='关系类型'
                    value={relationFilters.types}
                    onChange={(types) => updateRelationFilter('types', types)}
                    style={{ minWidth: 160 }}
                  >
                    {relationTypeOptions.map((type) => (
                      <Option key={type} value={type}>{relationTypeMap[type] || type || '-'}</Option>
                    ))}
                  </Select>
                  <Select
                    value={relationFilters.direction}
                    onChange={(direction) => updateRelationFilter('direction', direction)}
                    style={{ width: 140 }}
                  >
                    {Object.entries(relationDirectionMap).map(([value, label]) => (
                      <Option key={value} value={value}>{label}</Option>
                    ))}
                  </Select>
                  <Button disabled={!relationFilterActive} onClick={resetRelationFilters}>重置</Button>
                  <Text type='secondary'>{relationLoading ? '关系加载中...' : `显示 ${numberText(filteredDetailRelations.length)} / ${numberText(detailRelations.length)} 条`}</Text>
                </div>
                <RelationGraph
                  asset={detail}
                  relations={filteredDetailRelations}
                  onOpenDetail={openDetail}
                />
                <Table
                  style={{ marginTop: 16 }}
                  size='small'
                  rowKey='id'
                  columns={relationColumns}
                  dataSource={filteredDetailRelations}
                  loading={relationLoading}
                  pagination={false}
                  scroll={{ x: 'max-content' }}
                />
                {relationHasMore && (
                  <div className={styles.relationLoadMore}>
                    <Button loading={relationLoading} onClick={loadMoreRelations}>加载更多关系</Button>
                    <Text type='secondary'>下一批从第 {numberText((activeRelationSummary && activeRelationSummary.nextOffset) || loadedRelationCount)} 条开始</Text>
                  </div>
                )}
              </TabPane>
              <TabPane tab='变更' key='changes'>
                <Table
                  size='small'
                  rowKey='id'
                  columns={changeColumns}
                  dataSource={detail.changes || []}
                  pagination={false}
                  scroll={{ x: 'max-content' }}
                />
              </TabPane>
              <TabPane tab='原始数据' key='rawData'>
                {renderJSON(detail.rawData)}
              </TabPane>
            </Tabs>
          </Space>
        )}
      </Drawer>
    </Layout>
  );
};
