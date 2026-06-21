import React, { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import moment from 'moment';
import queryString from 'query-string';
import {
  Alert,
  Button,
  Checkbox,
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
import { DownloadOutlined, ReloadOutlined, SaveOutlined, SearchOutlined, SyncOutlined, UploadOutlined } from '@ant-design/icons';
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
const { Text } = Typography;
const { RangePicker } = DatePicker;

const assetTypeMap = {
  compute_instance: '计算实例',
  kubernetes_cluster: 'Kubernetes集群',
  network_vpc: 'VPC/VCN',
  network_subnet: '子网',
  network_route_table: '路由表',
  network_security_group: '安全组',
  public_ip: '公网IP',
  load_balancer: '负载均衡',
  block_volume: '块存储',
  object_storage_bucket: '对象存储',
  relational_database: '关系型数据库',
  redis_cache: 'Redis缓存',
  unknown: '未分类'
};

const sourceMap = {
  iac_resource: 'IaC资源',
  cloud_collect: '云采集',
  manual_edit: '人工维护',
  manual_application: '人工应用关系',
  import: '导入'
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
const joinList = (value) => Array.isArray(value) && value.length ? value.join(', ') : '-';
const uniqueList = (values) => Array.from(new Set((values || []).filter(Boolean)));
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

const assetAttrValue = (asset, keys) => {
  const attrs = asset && asset.attributes || {};
  const raw = asset && asset.rawData || {};
  for (const key of keys) {
    if (attrs[key] !== undefined && attrs[key] !== null && attrs[key] !== '') {
      return attrs[key];
    }
    if (raw[key] !== undefined && raw[key] !== null && raw[key] !== '') {
      return raw[key];
    }
  }
  return undefined;
};

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

const kubernetesNodeItems = (asset = {}) => {
  const attrs = asset.attributes || {};
  const groups = attrs.nodeGroups || attrs.nodegroups || [];
  const pools = attrs.nodePools || attrs.nodepools || [];
  return [
    ...(Array.isArray(groups) ? groups.map((item) => ({ ...item, k8sKind: 'NodeGroup' })) : []),
    ...(Array.isArray(pools) ? pools.map((item) => ({ ...item, k8sKind: 'NodePool' })) : [])
  ];
};

const kubernetesNodeScaleText = (record = {}) => {
  const scaling = record.scalingConfig || {};
  if (scaling.desiredSize !== undefined || scaling.minSize !== undefined || scaling.maxSize !== undefined) {
    return `期望 ${scaling.desiredSize || 0} / 最小 ${scaling.minSize || 0} / 最大 ${scaling.maxSize || 0}`;
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

const KubernetesAssetInfo = ({ asset = {} }) => {
  const attrs = asset.attributes || {};
  const nodeItems = kubernetesNodeItems(asset);
  const nodeCollectError = attrs.nodeGroupCollectError || attrs.nodePoolCollectError;
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
      render: (text) => text ? <Tag color={String(text).toLowerCase().includes('active') ? 'success' : 'default'}>{text}</Tag> : '-'
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
      render: (value, record) => joinList(value || (record.nodeShape ? [record.nodeShape] : []))
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
      render: (value, record) => joinList(value || record.subnetIds || [])
    }
  ];
  return (
    <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
      <Descriptions column={2} size='small' bordered={true}>
        <Descriptions.Item label='集群名称'>{assetAttrValue(asset, ['name']) || asset.name || '-'}</Descriptions.Item>
        <Descriptions.Item label='K8S版本'>{assetAttrValue(asset, ['version', 'kubernetesVersion']) || '-'}</Descriptions.Item>
        <Descriptions.Item label='平台版本'>{assetAttrValue(asset, ['platformVersion']) || '-'}</Descriptions.Item>
        <Descriptions.Item label='集群状态'>{asset.status || '-'}</Descriptions.Item>
        <Descriptions.Item label='API Endpoint' span={2}>{asset.address || '-'}</Descriptions.Item>
        <Descriptions.Item label='VPC/VCN'>{assetAttrValue(asset, ['vpcId', 'vcnId']) || '-'}</Descriptions.Item>
        <Descriptions.Item label='Endpoint配置'>{renderAttrValue(assetAttrValue(asset, ['endpointConfig']))}</Descriptions.Item>
        <Descriptions.Item label='子网' span={2}>{renderAttrValue(assetAttrValue(asset, ['subnetIds']))}</Descriptions.Item>
        <Descriptions.Item label='安全组/NSG' span={2}>{renderAttrValue(assetAttrValue(asset, ['securityGroupIds', 'clusterSecurityGroupId', 'nsgIds']))}</Descriptions.Item>
        <Descriptions.Item label='网络配置' span={2}>{renderAttrValue(assetAttrValue(asset, ['kubernetesNetwork', 'options']))}</Descriptions.Item>
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
  const [ownershipForm] = Form.useForm();
  const [batchOwnershipForm] = Form.useForm();
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
  const [ applicationRelationForm, setApplicationRelationForm ] = useState({
    upstreams: [],
    downstreams: []
  });
  const [ importOverwriteOwnership, setImportOverwriteOwnership ] = useState(true);
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
        refresh();
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
    fetchCloudAccounts();
    refreshSyncTasks();
    fetchApplications();
    if (isCloudAssets) {
      fetchCoverage();
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

  const selectedCloudAccount = useMemo(() => (
    (cloudAccounts || []).find((account) => accountKey(account) === cloudForm.accountKey)
  ), [cloudAccounts, cloudForm.accountKey]);

  const coverageMetrics = coverage.metrics || {};
  const coverageProviders = coverage.providers || [];
  const coverageAccounts = coverage.accounts || [];
  const coverageAssetTypes = coverage.assetTypes || [];
  const iacCoverageRate = percent(coverageMetrics.iacCoverageRate);
  const cloudOnlyRate = percent(coverageMetrics.cloudOnlyRate);
  const ownershipCoverageRate = percent(coverageMetrics.ownershipCoverageRate);
  const applicationList = (applicationsData && applicationsData.list) || [];
  const syncTaskList = (syncTasksData && syncTasksData.list) || [];
  const syncTaskSummary = (syncTasksData && syncTasksData.summary) || {};
  const syncTaskTrend = Array.isArray(syncTaskSummary.trend) ? syncTaskSummary.trend : [];
  const syncTaskRegionBreakdown = Array.isArray(syncTaskSummary.regions) ? syncTaskSummary.regions : [];
  const syncTaskAssetTypeBreakdown = Array.isArray(syncTaskSummary.assetTypes) ? syncTaskSummary.assetTypes : [];
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
  const syncTaskRerunGroupApproval = syncTaskRerunGroup && syncTaskRerunGroup.approval || {};
  const canApproveSyncTaskRerunGroup = Boolean(
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
    setSyncTaskDetailVisible(true);
    fetchSyncTaskDetail(record.id);
  };

  const closeSyncTaskDetail = () => {
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

  const exportAssets = async (format) => {
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

  const onImportAssetsFile = async (file) => {
    try {
      const content = await readTextFile(file);
      const payload = JSON.parse(content);
      const assets = normalizeImportAssets(payload);
      if (!assets.length) {
        notification.error({ message: '导入失败', description: '文件中未找到 assets 数据' });
        return false;
      }
      await importAssets({
        assets,
        overwriteOwnership: importOverwriteOwnership
      });
    } catch (err) {
      notification.error({
        message: '导入失败',
        description: err.message
      });
    }
    return false;
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
      }
    }, 3000);
    return () => clearInterval(timer);
  }, [hasRunningSyncTask, searchParams, isCloudAssets, syncTaskQuery, syncTaskTrendDays, syncTaskTrendRange, syncTaskFailureThreshold]);

  const syncIac = async () => {
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
    if (mode === 'failed' && !selectedFailedSyncTasks.length) {
      notification.warning({ message: '请选择需要重跑的失败任务' });
      return;
    }
    setSyncTaskRerunMode(mode);
    setSyncTaskRerunReason('');
    setSyncTaskRerunRequiresApproval(false);
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
    if (record.key === 'resize_volume') {
      setAssetActionDryRunRecord(record);
      setAssetActionDryRunParamsText(JSON.stringify(defaultResizeVolumeParams(detail), null, 2));
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
    const targetSizeGiB = Number(params && params.targetSizeGiB);
    if (!targetSizeGiB || Number.isNaN(targetSizeGiB) || targetSizeGiB <= 0) {
      notification.warning({ message: '请填写 targetSizeGiB 正整数' });
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
    setAssetActionParamsText(record.key === 'resize_volume' ? JSON.stringify(defaultResizeVolumeParams(detail), null, 2) : '');
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
    if (record.key === 'resize_volume') {
      try {
        params = JSON.parse(assetActionParamsText || '{}');
      } catch (err) {
        notification.warning({ message: '参数 JSON 格式不正确' });
        return;
      }
      const targetSizeGiB = Number(params && params.targetSizeGiB);
      const currentSizeGiB = volumeSizeGiB(detail);
      if (!targetSizeGiB || Number.isNaN(targetSizeGiB) || targetSizeGiB <= 0) {
        notification.warning({ message: '请填写 targetSizeGiB 正整数' });
        return;
      }
      if (currentSizeGiB > 0 && targetSizeGiB <= currentSizeGiB) {
        notification.warning({ message: `目标容量需大于当前容量 ${currentSizeGiB} GiB` });
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
      width: 110,
      render: (text) => text ? <a onClick={() => applyAssetFilters({ providers: [text] })}>{text}</a> : '-'
    },
    {
      dataIndex: 'assetCount',
      title: '资产',
      width: 90,
      render: numberText
    },
    {
      dataIndex: 'iacManagedAssetCount',
      title: 'IaC',
      width: 80,
      render: numberText
    },
    {
      dataIndex: 'cloudOnlyAssetCount',
      title: '未纳管',
      width: 90,
      render: (value, record) => value ? <a onClick={() => applyAssetFilters({ providers: [record.provider], managedBy: ['cloud_only'] })}><Tag color='warning'>{numberText(value)}</Tag></a> : 0
    },
    {
      dataIndex: 'unownedAssetCount',
      title: '无负责人',
      width: 100,
      render: (value) => value ? <Tag color='warning'>{numberText(value)}</Tag> : 0
    },
    {
      dataIndex: 'highRiskAssetCount',
      title: '高风险',
      width: 90,
      render: (value) => value ? <Tag color='error'>{numberText(value)}</Tag> : 0
    }
  ], [searchParams]);

  const coverageAccountColumns = useMemo(() => [
    {
      dataIndex: 'accountName',
      title: '账号',
      width: 180,
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
      width: 90,
      render: (text) => text ? <Tag>{text}</Tag> : '-'
    },
    {
      dataIndex: 'assetCount',
      title: '资产',
      width: 80,
      render: numberText
    },
    {
      dataIndex: 'cloudOnlyAssetCount',
      title: '未纳管',
      width: 90,
      render: (value, record) => value ? <a onClick={() => applyAssetFilters({
        providers: record.provider ? [record.provider] : undefined,
        accountIds: record.accountId ? [record.accountId] : undefined,
        managedBy: ['cloud_only']
      })}><Tag color='warning'>{numberText(value)}</Tag></a> : 0
    },
    {
      dataIndex: 'unownedAssetCount',
      title: '无负责人',
      width: 100,
      render: (value) => value ? <Tag color='warning'>{numberText(value)}</Tag> : 0
    }
  ], [searchParams]);

  const coverageTypeColumns = useMemo(() => [
    {
      dataIndex: 'assetType',
      title: '资产类型',
      width: 160,
      render: (text) => <a onClick={() => applyAssetFilters({ assetTypes: [text] })}>{assetTypeMap[text] || text || '-'}</a>
    },
    {
      dataIndex: 'assetCount',
      title: '资产',
      width: 80,
      render: numberText
    },
    {
      dataIndex: 'cloudCollectedAssetCount',
      title: '云采集',
      width: 90,
      render: numberText
    },
    {
      dataIndex: 'cloudOnlyAssetCount',
      title: '未纳管',
      width: 90,
      render: (value, record) => value ? <a onClick={() => applyAssetFilters({ assetTypes: [record.assetType], managedBy: ['cloud_only'] })}><Tag color='warning'>{numberText(value)}</Tag></a> : 0
    },
    {
      dataIndex: 'highRiskAssetCount',
      title: '高风险',
      width: 90,
      render: (value) => value ? <Tag color='error'>{numberText(value)}</Tag> : 0
    }
  ], [searchParams]);

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
      title: '近7天变更',
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
  ], []);

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
      dataIndex: 'stats',
      title: '子周期',
      width: 180,
      render: renderSyncTaskSchedule
    },
    {
      dataIndex: 'stats',
      title: '统计',
      width: 260,
      render: (stats) => {
        const safeStats = stats || {};
        return (
          <Space direction='vertical' size={0}>
            <Text>采集 {safeStats.collected || 0} / 新增 {safeStats.created || 0} / 更新 {safeStats.updated || 0} / 跳过 {safeStats.skipped || 0}</Text>
            {Array.isArray(safeStats.errors) && safeStats.errors.length > 0 && (
              <Text type='danger'>错误 {safeStats.errors.length} 条</Text>
            )}
          </Space>
        );
      }
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
      mode='multiple'
      allowClear={true}
      maxTagCount='responsive'
      placeholder={placeholder}
      value={(searchParams.form || {})[key]}
      onChange={(value) => onFilterChange(key, value)}
      style={{ minWidth: 180 }}
    >
      {(options || []).map((it) => (
        <Option key={valueGetter(it)} value={valueGetter(it)}>{labelGetter(it)}</Option>
      ))}
    </Select>
  );

  const createActionRequiresConfirm = assetActionCreateRecord && assetActionCreateRecord.adapterMode === 'provider';
  const createActionExpectedResourceId = detail && (detail.nativeId || detail.id) || '';
  const createActionRequiresTags = assetActionCreateRecord && assetActionCreateRecord.key === 'update_tags';
  const createActionRequiresParams = assetActionCreateRecord && assetActionCreateRecord.key === 'resize_volume';
  const createActionOkDisabled = !!(createActionRequiresConfirm && (
    (assetActionCreateReason || '').trim().length < 6 ||
    (assetActionConfirmAction || '').trim() !== (assetActionCreateRecord && assetActionCreateRecord.key) ||
    (assetActionConfirmResourceId || '').trim() !== createActionExpectedResourceId
  )) ||
    !!(createActionRequiresTags && !(assetActionTagsText || '').trim()) ||
    !!(createActionRequiresParams && !(assetActionParamsText || '').trim());

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
            <Space size='middle' direction='vertical' style={{ width: '100%', display: 'flex' }}>
              {isCloudAssets && (
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
              )}
              <div className={styles.toolbar}>
                <Space className={styles.filterBar} size={[8, 8]} wrap={true}>
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
                    style={{ width: 320 }}
                  />
                  {selectFilter('providers', '云厂商', filters.providers)}
                  {isCloudAssets && selectFilter('accountIds', '账号', filters.accountIds)}
                  {isCloudAssets && selectFilter('managedBy', '纳管状态', filters.managedBy, (it) => managedByMap[it] || it)}
                  {selectFilter('assetTypes', '资产类型', filters.assetTypes, (it) => assetTypeMap[it] || it)}
                  {selectFilter('projectIds', '项目', filters.projects, (it) => it.projectName, (it) => it.projectId)}
                  {selectFilter('envIds', '环境', filters.envs, (it) => it.envName, (it) => it.envId)}
                  {selectFilter('sources', '来源', filters.sources, (it) => sourceMap[it] || it)}
                  {selectFilter('statuses', '状态', filters.statuses)}
                </Space>
                <Space className={styles.actionBar} size={[8, 8]} wrap={true}>
                  <Checkbox
                    checked={importOverwriteOwnership}
                    onChange={(e) => setImportOverwriteOwnership(e.target.checked)}
                  >
                    覆盖归属
                  </Checkbox>
                  <Upload
                    accept='.json,application/json'
                    beforeUpload={onImportAssetsFile}
                    showUploadList={false}
                  >
                    <Button icon={<UploadOutlined />} loading={importingAssets}>导入 JSON</Button>
                  </Upload>
                  <Button icon={<DownloadOutlined />} onClick={() => exportAssets('csv')}>
                    {selectedAssetIds.length ? `导出选中 CSV(${selectedAssetIds.length})` : '导出 CSV'}
                  </Button>
                  <Button icon={<DownloadOutlined />} onClick={() => exportAssets('json')}>
                    {selectedAssetIds.length ? '导出选中 JSON' : '导出 JSON'}
                  </Button>
                  {isCloudAssets && (
                    <Button icon={<SaveOutlined />} disabled={!selectedAssetIds.length} onClick={openBatchOwnership}>
                      {selectedAssetIds.length ? `批量治理(${selectedAssetIds.length})` : '批量治理'}
                    </Button>
                  )}
                  <Button icon={<ReloadOutlined />} onClick={refresh}>刷新</Button>
                  <Button type='primary' icon={<SyncOutlined />} loading={backfillLoading} onClick={syncIac}>
                    同步 IaC 资源
                  </Button>
                </Space>
              </div>
              <ConfigProvider
                renderEmpty={
                  () => <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无资产'/>
                }
              >
                <Table
                  rowKey='id'
                  columns={columns}
                  scroll={{ x: 'min-content' }}
                  loading={tableLoading}
                  {...tableProps}
                  rowSelection={{
                    selectedRowKeys: selectedAssetIds,
                    onChange: setSelectedAssetIds
                  }}
                />
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
                    loading={cloudSyncLoading}
                    disabled={!syncTaskHistorySeed}
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
                  </div>
                  <div className={styles.syncTaskMetric}>
                    <Text type='secondary'>最近失败</Text>
                    <div className={styles.syncTaskMetricTime}>{renderTime(syncTaskSummary.lastFailureAt)}</div>
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
                    disabled={!selectedCloudAccount}
                    loading={cloudSyncLoading}
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
        title={syncTaskRerunMode === 'failed' ? `重跑失败任务（${selectedFailedSyncTasks.length}）` : '重跑当前子周期'}
        visible={syncTaskRerunReasonVisible}
        confirmLoading={syncTaskRerunSubmitting || cloudSyncLoading || batchRerunningSyncTasks || batchRerunFailedSyncLoading}
        okText={syncTaskRerunMode === 'failed' && syncTaskRerunRequiresApproval ? '提交审批' : '启动重跑'}
        cancelText='取消'
        destroyOnClose={true}
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
                onChange={(event) => setSyncTaskRerunRequiresApproval(event.target.checked)}
              >
                提交审批后再启动
              </Checkbox>
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
                  placeholder='{"targetSizeGiB":100}'
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
                    placeholder='{"targetSizeGiB":100}'
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
              <Descriptions.Item label='开始时间'>{renderTime(syncTaskDetail.startedAt)}</Descriptions.Item>
              <Descriptions.Item label='结束时间'>{renderTime(syncTaskDetail.endedAt)}</Descriptions.Item>
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
              <Descriptions.Item label='近7天变更'>{applicationDetail.recentChangeCount || 0}</Descriptions.Item>
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
            <Tabs defaultActiveKey='ownership'>
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
              {detail.assetType === 'kubernetes_cluster' && (
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
                <RelationGraph
                  asset={detail}
                  relations={detail.relations || []}
                  onOpenDetail={openDetail}
                />
                <Table
                  style={{ marginTop: 16 }}
                  size='small'
                  rowKey='id'
                  columns={relationColumns}
                  dataSource={detail.relations || []}
                  pagination={false}
                  scroll={{ x: 'max-content' }}
                />
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
