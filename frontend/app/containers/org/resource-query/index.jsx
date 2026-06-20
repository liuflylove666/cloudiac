import React, { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import queryString from 'query-string';
import {
  Button,
  Checkbox,
  ConfigProvider,
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
  running: '运行中',
  complete: '完成',
  failed: '失败'
};

const taskStatusColorMap = {
  pending: 'default',
  running: 'processing',
  complete: 'success',
  failed: 'error'
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

const asCSV = (value) => Array.isArray(value) ? value.join(',') : value;
const accountKey = (account) => `${account.source}:${account.id}`;
const joinList = (value) => Array.isArray(value) && value.length ? value.join(', ') : '-';
const renderTime = (value) => !value || String(value).indexOf('0001-01-01') === 0 ? '-' : value;
const numberText = (value) => Number(value || 0).toLocaleString();
const percent = (value) => Math.max(0, Math.min(100, Number(value || 0)));
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
const renderCompactList = (value) => Array.isArray(value) && value.length ? value.join(' / ') : '-';
const applicationAssetDsl = (application) => `application:"${String(application || '').replace(/"/g, '\\"')}"`;

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

export default ({ match, location }) => {
  const { orgId } = match.params || {};
  const { assetId } = queryString.parse(location && location.search || '');
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
        fetchSyncTasks();
        fetchList({ currentPage: 1, pageSize: 10 });
        fetchFilters();
        if (isCloudAssets) {
          fetchCoverage();
        }
      }
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

  useEffect(() => {
    fetchFilters();
    fetchCloudAccounts();
    fetchSyncTasks();
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
    syncTaskList.some((task) => ['pending', 'running'].includes(task.status))
  ), [syncTasksData]);

  const openDetail = (id) => {
    setDetailVisible(true);
    fetchDetail(id);
  };

  const closeDetail = () => {
    setDetailVisible(false);
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
      fetchSyncTasks();
      refresh();
      if (isCloudAssets) {
        fetchCoverage();
      }
    }, 3000);
    return () => clearInterval(timer);
  }, [hasRunningSyncTask, searchParams, isCloudAssets]);

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
                    fetchSyncTasks();
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
                pagination={{
                  total: syncTasksData && syncTasksData.total,
                  pageSize: syncTasksData && syncTasksData.pageSize || 10,
                  showTotal: (total) => `共${total}条`
                }}
                onChange={({ current, pageSize }) => fetchSyncTasks({ currentPage: current, pageSize })}
              />
            </Space>
          </TabPane>
        </Tabs>
      </div>
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
              <TabPane tab='属性' key='attributes'>
                {renderJSON(detail.attributes)}
              </TabPane>
              <TabPane tab='标签' key='tags'>
                {renderJSON(detail.tags)}
              </TabPane>
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
