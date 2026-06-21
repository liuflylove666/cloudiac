import React, { useMemo, useState } from 'react';
import {
  Button,
  Checkbox,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Progress,
  Select,
  Space,
  Table,
  Tag,
  Typography
} from 'antd';
import { CheckCircleOutlined, CloudDownloadOutlined, DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, StopOutlined, SyncOutlined, UndoOutlined } from '@ant-design/icons';
import { useRequest } from 'ahooks';
import moment from 'moment';
import PageHeader from 'components/pageHeader';
import Layout from 'components/common/layout';
import { requestWrapper } from 'utils/request';
import cloudCostAPI from 'services/cloud-cost';
import styles from './styles.less';

const { Option } = Select;
const { Search: InputSearch } = Input;
const { Text } = Typography;

const sourceMap = {
  bill: '账单',
  cmdb_asset: '资产成本',
  import: '导入',
  aws_cur_export: 'AWS CUR 导出',
  aws_cost_explorer_export: 'AWS Cost Explorer 导出',
  oci_usage_cost_export: 'OCI Usage/Cost 导出',
  azure_cost_management: 'Azure Cost Management',
  azure_billing_export: 'Azure 成本导出',
  gcp_billing_export: 'GCP Billing Export',
  tencentcloud_billing_export: '腾讯云账单导出',
  huawei_billing_export: '华为云账单导出'
};

const importSourceMap = {
  aws_cur_export: 'AWS CUR 导出',
  aws_cost_explorer_export: 'AWS Cost Explorer 导出',
  oci_usage_cost_export: 'OCI Usage/Cost 导出',
  azure_cost_management: 'Azure Cost Management',
  azure_billing_export: 'Azure 成本导出',
  gcp_billing_export: 'GCP Billing Export',
  tencentcloud_billing_export: '腾讯云账单导出',
  huawei_billing_export: '华为云账单导出',
  import: '通用导入'
};

const pullSourceMap = {
  aws_cur_export: 'AWS CUR 导出 URL',
  aws_cost_explorer_export: 'AWS Cost Explorer 导出 URL',
  oci_usage_cost_export: 'OCI Usage/Cost 导出 URL',
  azure_cost_management: 'Azure Cost Management',
  azure_billing_export: 'Azure 成本导出 URL',
  gcp_billing_export: 'GCP Billing Export URL',
  tencentcloud_billing_export: '腾讯云账单导出 URL',
  huawei_billing_export: '华为云账单导出 URL'
};

const objectStorageProviderMap = {
  s3: 'S3',
  gcs: 'GCS',
  azure_blob: 'Azure Blob',
  oci_object_storage: 'OCI Object Storage'
};

const syncTaskStatusMap = {
  pending: { label: '待执行', color: 'default' },
  running: { label: '运行中', color: 'blue' },
  complete: { label: '完成', color: 'green' },
  failed: { label: '失败', color: 'red' }
};

const syncScheduleStatusMap = {
  enable: { label: '启用', color: 'success' },
  disable: { label: '停用', color: 'default' }
};

const budgetScopeMap = {
  org: '组织',
  project: '项目',
  env: '环境',
  provider: '云厂商',
  account: '云账号',
  application: '应用',
  business_line: '业务线',
  cost_center: '成本中心',
  owner: 'Owner'
};

const budgetScopeFieldMap = {
  project: { name: 'projectId', label: '项目 ID', placeholder: '请输入项目 ID' },
  env: { name: 'envId', label: '环境 ID', placeholder: '请输入环境 ID' },
  provider: { name: 'provider', label: '云厂商', placeholder: '例如 aws / oci / alicloud' },
  account: { name: 'accountId', label: '云账号 ID', placeholder: '请输入云账号原生 ID' },
  application: { name: 'application', label: '应用', placeholder: '请输入应用名称' },
  business_line: { name: 'businessLine', label: '业务线', placeholder: '请输入业务线' },
  cost_center: { name: 'costCenter', label: '成本中心', placeholder: '请输入成本中心' },
  owner: { name: 'owner', label: 'Owner', placeholder: '请输入负责人' }
};

const insightTypeMap = {
  anomaly: '成本异常',
  optimization: '优化建议'
};

const insightSeverityMap = {
  high: { label: '高', color: 'red' },
  medium: { label: '中', color: 'orange' },
  low: { label: '低', color: 'blue' }
};

const insightStatusMap = {
  open: { label: '待处理', color: 'processing' },
  resolved: { label: '已解决', color: 'success' },
  ignored: { label: '已忽略', color: 'default' }
};

const money = (value, currency = 'CNY') => {
  const amount = Number(value || 0);
  return `${amount.toFixed(2)} ${currency || ''}`.trim();
};

const percent = (value) => `${Number(value || 0).toFixed(1)}%`;
const renderTime = (value) => !value || String(value).indexOf('0001-01-01') === 0 ? '-' : moment(value).format('YYYY-MM-DD HH:mm:ss');
const renderFileSize = (value) => {
  const bytes = Number(value || 0);
  if (!bytes) {
    return '-';
  }
  if (bytes >= 1024 * 1024 * 1024) {
    return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  }
  if (bytes >= 1024 * 1024) {
    return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  }
  if (bytes >= 1024) {
    return `${(bytes / 1024).toFixed(2)} KB`;
  }
  return `${bytes} B`;
};
const renderInlineText = (value) => value ? <Text className={styles.inlineHint}>{value}</Text> : '-';

const JsonBlock = ({ value }) => (
  <pre className={styles.jsonBlock}>{!value || Object.keys(value || {}).length === 0 ? '-' : JSON.stringify(value, null, 2)}</pre>
);

const Metric = ({ label, value, tone }) => (
  <div className={`${styles.metric} ${tone ? styles[tone] : ''}`}>
    <div className={styles.metricValue}>{value}</div>
    <div className={styles.metricLabel}>{label}</div>
  </div>
);

const CostDetail = ({ detail = {}, loading }) => {
  if (loading) {
    return <Empty description='加载中'/>;
  }
  if (!detail.id) {
    return <Empty description='请选择成本记录'/>;
  }
  return (
    <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
      <Descriptions size='small' bordered={true} column={2}>
        <Descriptions.Item label='记录 ID'>{detail.id || '-'}</Descriptions.Item>
        <Descriptions.Item label='周期'>{detail.period || '-'}</Descriptions.Item>
        <Descriptions.Item label='金额'>{money(detail.amount, detail.currency)}</Descriptions.Item>
        <Descriptions.Item label='来源'>{sourceMap[detail.source] || detail.source || '-'}</Descriptions.Item>
        <Descriptions.Item label='资源'>{detail.resourceName || detail.assetName || detail.resourceId || '-'}</Descriptions.Item>
        <Descriptions.Item label='服务'>{detail.service || '-'}</Descriptions.Item>
        <Descriptions.Item label='云厂商'>{detail.provider || '-'}</Descriptions.Item>
        <Descriptions.Item label='区域'>{detail.region || '-'}</Descriptions.Item>
        <Descriptions.Item label='项目'>{detail.projectName || detail.projectId || '-'}</Descriptions.Item>
        <Descriptions.Item label='环境'>{detail.envName || detail.envId || '-'}</Descriptions.Item>
        <Descriptions.Item label='应用'>{detail.application || '-'}</Descriptions.Item>
        <Descriptions.Item label='业务线'>{detail.businessLine || '-'}</Descriptions.Item>
        <Descriptions.Item label='成本中心'>{detail.costCenter || '-'}</Descriptions.Item>
        <Descriptions.Item label='匹配资产'>{detail.matchedAsset ? '是' : '否'}</Descriptions.Item>
      </Descriptions>
      <div className={styles.detailSection}>
        <Text strong={true}>原始载荷</Text>
        <JsonBlock value={detail.payload}/>
      </div>
    </Space>
  );
};

const CloudCostPage = ({ match }) => {
  const { orgId } = match.params || {};
  const currentPeriod = moment().format('YYYY-MM');
  const [ budgetForm ] = Form.useForm();
  const [ importForm ] = Form.useForm();
  const [ pullForm ] = Form.useForm();
  const [ syncScheduleForm ] = Form.useForm();
  const [ period, setPeriod ] = useState(currentPeriod);
  const [ query, setQuery ] = useState({
    currentPage: 1,
    pageSize: 10,
    period: currentPeriod
  });
  const [ insightQuery, setInsightQuery ] = useState({
    currentPage: 1,
    pageSize: 8,
    period: currentPeriod,
    status: 'open'
  });
  const [ drawer, setDrawer ] = useState({
    visible: false,
    id: ''
  });
  const [ syncTaskDrawer, setSyncTaskDrawer ] = useState({
    visible: false,
    id: ''
  });
  const [ budgetModal, setBudgetModal ] = useState({
    visible: false,
    record: null
  });
  const [ syncScheduleModal, setSyncScheduleModal ] = useState({
    visible: false,
    record: null
  });
  const [ importModalVisible, setImportModalVisible ] = useState(false);
  const [ pullModalVisible, setPullModalVisible ] = useState(false);
  const [ budgetSaving, setBudgetSaving ] = useState(false);
  const [ importSaving, setImportSaving ] = useState(false);
  const [ pullSaving, setPullSaving ] = useState(false);
  const [ syncScheduleSaving, setSyncScheduleSaving ] = useState(false);

  const {
    loading: summaryLoading,
    data: summary = {},
    run: fetchSummary
  } = useRequest(
    () => requestWrapper(cloudCostAPI.summary.bind(null, { orgId, period })),
    {
      refreshDeps: [ orgId, period ]
    }
  );

  const {
    loading: trendsLoading,
    data: trends = [],
    run: fetchTrends
  } = useRequest(
    () => requestWrapper(cloudCostAPI.trends.bind(null, { orgId, months: 6, currency: summary.currency || 'CNY' })),
    {
      refreshDeps: [ orgId, summary.currency ]
    }
  );

  const {
    loading,
    data = {},
    run: fetchRecords
  } = useRequest(
    () => requestWrapper(cloudCostAPI.records.bind(null, { orgId, ...query })),
    {
      refreshDeps: [ orgId, query ]
    }
  );
  const {
    loading: insightSummaryLoading,
    data: insightSummary = {},
    run: fetchInsightSummary
  } = useRequest(
    () => requestWrapper(cloudCostAPI.insightSummary.bind(null, { orgId, period, currency: summary.currency || 'CNY' })),
    {
      refreshDeps: [ orgId, period, summary.currency ]
    }
  );
  const {
    loading: insightLoading,
    data: insightData = {},
    run: fetchInsights
  } = useRequest(
    () => requestWrapper(cloudCostAPI.insights.bind(null, {
      orgId,
      ...insightQuery,
      currency: summary.currency || 'CNY'
    })),
    {
      refreshDeps: [ orgId, insightQuery, summary.currency ]
    }
  );
  const {
    loading: budgetSummaryLoading,
    data: budgetSummary = {},
    run: fetchBudgetSummary
  } = useRequest(
    () => requestWrapper(cloudCostAPI.budgetSummary.bind(null, { orgId, period, currency: summary.currency || 'CNY' })),
    {
      refreshDeps: [ orgId, period, summary.currency ]
    }
  );
  const {
    loading: budgetLoading,
    data: budgetData = {},
    run: fetchBudgets
  } = useRequest(
    () => requestWrapper(cloudCostAPI.budgets.bind(null, {
      orgId,
      currentPage: 1,
      pageSize: 8,
      period,
      currency: summary.currency || 'CNY'
    })),
    {
      refreshDeps: [ orgId, period, summary.currency ]
    }
  );
  const {
    loading: syncScheduleLoading,
    data: syncScheduleData = {},
    run: fetchSyncSchedules
  } = useRequest(
    () => requestWrapper(cloudCostAPI.syncSchedules.bind(null, {
      orgId,
      currentPage: 1,
      pageSize: 8,
      period
    })),
    {
      refreshDeps: [ orgId, period ]
    }
  );
  const {
    loading: syncTaskLoading,
    data: syncTaskData = {},
    run: fetchSyncTasks
  } = useRequest(
    () => requestWrapper(cloudCostAPI.syncTasks.bind(null, {
      orgId,
      currentPage: 1,
      pageSize: 8,
      period
    })),
    {
      refreshDeps: [ orgId, period ]
    }
  );
  const {
    loading: syncTaskDetailLoading,
    data: syncTaskDetail = {},
    run: fetchSyncTaskDetail
  } = useRequest(
    (id) => requestWrapper(cloudCostAPI.syncTaskDetail.bind(null, { orgId, id })),
    {
      manual: true
    }
  );

  const list = data.list || [];
  const insightList = insightData.list || [];
  const budgetList = budgetData.list || [];
  const syncScheduleList = syncScheduleData.list || [];
  const syncTaskList = syncTaskData.list || [];
  const detail = useMemo(() => list.find((item) => item.id === drawer.id) || {}, [ list, drawer.id ]);
  const refreshAll = () => {
    fetchSummary();
    fetchTrends();
    fetchRecords();
    fetchInsightSummary();
    fetchInsights();
    fetchBudgetSummary();
    fetchBudgets();
    fetchSyncSchedules();
    fetchSyncTasks();
  };

  const columns = useMemo(() => [
    {
      title: '资源',
      dataIndex: 'resourceName',
      width: 220,
      ellipsis: true,
      render: (text, record) => <a onClick={() => setDrawer({ visible: true, id: record.id })}>{text || record.assetName || record.resourceId || record.id}</a>
    },
    {
      title: '金额',
      dataIndex: 'amount',
      width: 130,
      render: (text, record) => money(text, record.currency)
    },
    {
      title: '周期',
      dataIndex: 'period',
      width: 110
    },
    {
      title: '来源',
      dataIndex: 'source',
      width: 110,
      render: (text) => sourceMap[text] || text || '-'
    },
    {
      title: '云厂商/区域',
      dataIndex: 'provider',
      width: 160,
      render: (text, record) => [ text, record.region ].filter(Boolean).join(' / ') || '-'
    },
    {
      title: '项目',
      dataIndex: 'projectName',
      width: 160,
      ellipsis: true,
      render: (text, record) => text || record.projectId || '-'
    },
    {
      title: '应用',
      dataIndex: 'application',
      width: 160,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      title: '成本中心',
      dataIndex: 'costCenter',
      width: 140,
      render: (text) => text || '-'
    },
    {
      title: '匹配',
      dataIndex: 'matchedAsset',
      width: 90,
      render: (value) => value ? <Tag color='success'>已匹配</Tag> : <Tag color='warning'>未匹配</Tag>
    }
  ], []);

  const groupColumns = [
    {
      title: '维度',
      dataIndex: 'label',
      ellipsis: true,
      render: (text, record) => text || record.key || '-'
    },
    {
      title: '金额',
      dataIndex: 'amount',
      width: 120,
      render: (text, record) => money(text, record.currency)
    }
  ];
  const trendColumns = [
    {
      title: '周期',
      dataIndex: 'period',
      width: 100
    },
    {
      title: '金额',
      dataIndex: 'amount',
      render: (text, record) => money(text, record.currency)
    }
  ];
  const budgetColumns = [
    {
      title: '预算',
      dataIndex: 'name',
      width: 220,
      ellipsis: true,
      render: (text, record) => (
        <Space size={6}>
          <Text strong={true}>{text}</Text>
          {record.budgetExceeded ? <Tag color='red'>超预算</Tag> : record.thresholdReached ? <Tag color='orange'>超阈值</Tag> : null}
        </Space>
      )
    },
    {
      title: '范围',
      dataIndex: 'scope',
      width: 140,
      render: (text, record) => {
        const field = budgetScopeFieldMap[text];
        const value = field ? record[field.name] : '';
        return [ budgetScopeMap[text] || text || '-', value ].filter(Boolean).join(' / ');
      }
    },
    {
      title: '预算金额',
      dataIndex: 'limitAmount',
      width: 140,
      render: (text, record) => money(text, record.currency)
    },
    {
      title: '当前成本',
      dataIndex: 'currentAmount',
      width: 140,
      render: (text, record) => money(text, record.currency)
    },
    {
      title: '使用率',
      dataIndex: 'usagePercent',
      width: 180,
      render: (text, record) => (
        <Progress
          size='small'
          percent={Math.min(Number(text || 0), 100)}
          status={record.budgetExceeded ? 'exception' : record.thresholdReached ? 'active' : 'normal'}
          format={() => percent(text)}
        />
      )
    },
    {
      title: '阈值',
      dataIndex: 'thresholdPercent',
      width: 100,
      render: (text) => percent(text)
    },
    {
      title: '评估',
      dataIndex: 'evaluationInterval',
      width: 170,
      render: (text, record) => (
        <Space direction='vertical' size={0}>
          <Text>{Number(text || 3600)}s</Text>
          <Text type='secondary' className={styles.inlineHint}>{renderTime(record.lastEvaluatedAt)}</Text>
        </Space>
      )
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (text) => text === 'disable' ? <Tag>停用</Tag> : <Tag color='success'>启用</Tag>
    },
    {
      title: '操作',
      dataIndex: 'op',
      width: 150,
      render: (_, record) => (
        <Space size={4}>
          <Button type='link' size='small' icon={<EditOutlined/>} onClick={() => openBudgetModal(record)}>编辑</Button>
          <Button type='link' size='small' danger={true} icon={<DeleteOutlined/>} onClick={() => deleteBudget(record)}>删除</Button>
        </Space>
      )
    }
  ];
  const insightColumns = [
    {
      title: '建议',
      dataIndex: 'title',
      width: 260,
      ellipsis: true,
      render: (text, record) => (
        <Space direction='vertical' size={0}>
          <Text strong={true}>{text || record.ruleKey || '-'}</Text>
          <Text type='secondary' className={styles.inlineHint}>{record.recommendation || '-'}</Text>
        </Space>
      )
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 110,
      render: (text) => insightTypeMap[text] || text || '-'
    },
    {
      title: '等级',
      dataIndex: 'severity',
      width: 80,
      render: (text) => {
        const item = insightSeverityMap[text] || {};
        return <Tag color={item.color}>{item.label || text || '-'}</Tag>;
      }
    },
    {
      title: '资源',
      dataIndex: 'resourceName',
      width: 220,
      ellipsis: true,
      render: (text, record) => text || record.assetName || record.resourceId || '-'
    },
    {
      title: '金额',
      dataIndex: 'amount',
      width: 130,
      render: (text, record) => money(text, record.currency)
    },
    {
      title: '可节省',
      dataIndex: 'potentialSavings',
      width: 130,
      render: (text, record) => money(text, record.currency)
    },
    {
      title: '云厂商/区域',
      dataIndex: 'provider',
      width: 160,
      render: (text, record) => [ text, record.region ].filter(Boolean).join(' / ') || '-'
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (text) => {
        const item = insightStatusMap[text] || {};
        return <Tag color={item.color}>{item.label || text || '-'}</Tag>;
      }
    },
    {
      title: '操作',
      dataIndex: 'op',
      width: 180,
      render: (_, record) => record.status === 'open' ? (
        <Space size={4}>
          <Button type='link' size='small' icon={<CheckCircleOutlined/>} onClick={() => updateInsightStatus(record, 'resolved')}>解决</Button>
          <Button type='link' size='small' icon={<StopOutlined/>} onClick={() => updateInsightStatus(record, 'ignored')}>忽略</Button>
        </Space>
      ) : (
        <Button type='link' size='small' icon={<UndoOutlined/>} onClick={() => updateInsightStatus(record, 'open')}>重开</Button>
      )
    }
  ];
  const syncScheduleColumns = [
    {
      title: '计划',
      dataIndex: 'name',
      width: 220,
      ellipsis: true,
      render: (text, record) => (
        <Space direction='vertical' size={0}>
          <Text strong={true}>{text || record.id}</Text>
          <Text type='secondary' className={styles.inlineHint}>{record.description || record.id}</Text>
        </Space>
      )
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (text) => {
        const item = syncScheduleStatusMap[text] || {};
        return <Tag color={item.color}>{item.label || text || '-'}</Tag>;
      }
    },
    {
      title: '来源',
      dataIndex: 'source',
      width: 210,
      render: (text, record) => (
        <Space direction='vertical' size={0}>
          <Text>{[ record.provider, pullSourceMap[text] || sourceMap[text] || text ].filter(Boolean).join(' / ') || '-'}</Text>
          <Text type='secondary' className={styles.inlineHint}>
            {record.sourceObjectEndpointPreview || record.sourceIndexUrlPreview || record.sourceUrlPreview || record.cloudAccountName || record.cloudAccountId || '-'}
          </Text>
        </Space>
      )
    },
    {
      title: '账期',
      dataIndex: 'period',
      width: 120,
      render: (text, record) => [ text, record.currency ].filter(Boolean).join(' / ') || '-'
    },
    {
      title: '间隔',
      dataIndex: 'syncInterval',
      width: 90,
      render: (text) => `${Number(text || 0)}s`
    },
    {
      title: '最近同步',
      dataIndex: 'lastSyncStatus',
      width: 190,
      render: (text, record) => {
        const item = syncTaskStatusMap[text] || {};
        return (
          <Space direction='vertical' size={0}>
            <Text>{text ? <Tag color={item.color}>{item.label || text}</Tag> : '-'}</Text>
            <Text type='secondary' className={styles.inlineHint}>{renderTime(record.lastSyncedAt)}</Text>
            {record.failureCount ? <Text type='secondary' className={styles.inlineHint}>失败 {record.failureCount} 次</Text> : null}
            {record.nextRetryAt ? <Text type='secondary' className={styles.inlineHint}>重试 {renderTime(record.nextRetryAt)}</Text> : null}
            {record.autoPausedAt ? <Tag color='red'>自动暂停</Tag> : null}
          </Space>
        );
      }
    },
    {
      title: '下次同步',
      dataIndex: 'nextSyncAt',
      width: 180,
      render: renderTime
    },
    {
      title: '操作',
      dataIndex: 'op',
      width: 220,
      render: (_, record) => (
        <Space size={4}>
          <Button type='link' size='small' icon={<SyncOutlined/>} onClick={() => runSyncSchedule(record)}>运行</Button>
          <Button type='link' size='small' icon={<EditOutlined/>} onClick={() => openSyncScheduleModal(record)}>编辑</Button>
          <Button type='link' size='small' danger={true} icon={<DeleteOutlined/>} onClick={() => deleteSyncSchedule(record)}>删除</Button>
        </Space>
      )
    }
  ];
  const syncTaskColumns = [
    {
      title: '任务',
      dataIndex: 'id',
      width: 180,
      render: (text, record) => <a onClick={() => openSyncTaskDrawer(record)}>{text}</a>
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (text) => {
        const item = syncTaskStatusMap[text] || {};
        return <Tag color={item.color}>{item.label || text || '-'}</Tag>;
      }
    },
    {
      title: '来源',
      dataIndex: 'source',
      width: 190,
      render: (text, record) => (
        <Space direction='vertical' size={0}>
          <Text>{[ record.provider, pullSourceMap[text] || sourceMap[text] || text ].filter(Boolean).join(' / ') || '-'}</Text>
          <Text type='secondary' className={styles.inlineHint}>
            {record.sourceObjectEndpointPreview || record.sourceIndexUrlPreview || record.sourceUrlPreview || record.cloudAccountName || record.cloudAccountId || '-'}
          </Text>
        </Space>
      )
    },
    {
      title: '账期',
      dataIndex: 'period',
      width: 120,
      render: (text, record) => [ text, record.currency ].filter(Boolean).join(' / ') || '-'
    },
    {
      title: '导入',
      dataIndex: 'imported',
      width: 150,
      render: (text, record) => `${text || 0} / 匹配 ${record.matchedCount || 0}`
    },
    {
      title: '尝试',
      dataIndex: 'attemptCount',
      width: 80,
      render: (text) => text || 0
    },
    {
      title: '时间',
      dataIndex: 'startedAt',
      width: 180,
      render: (text, record) => (
        <Space direction='vertical' size={0}>
          <Text>{renderTime(text)}</Text>
          <Text type='secondary' className={styles.inlineHint}>{renderTime(record.endedAt)}</Text>
        </Space>
      )
    },
    {
      title: '消息',
      dataIndex: 'message',
      ellipsis: true,
      render: (text, record) => record.error || text || '-'
    },
    {
      title: '操作',
      dataIndex: 'op',
      width: 90,
      render: (_, record) => record.status === 'failed'
        ? <Button type='link' size='small' icon={<UndoOutlined/>} onClick={() => retrySyncTask(record)}>重试</Button>
        : null
    }
  ];
  const syncTaskLogColumns = [
    {
      title: '阶段',
      dataIndex: 'stage',
      width: 100
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (text) => {
        const item = syncTaskStatusMap[text] || {};
        return <Tag color={item.color}>{item.label || text || '-'}</Tag>;
      }
    },
    {
      title: '消息',
      dataIndex: 'message',
      ellipsis: true,
      render: (text, record) => record.error || text || '-'
    },
    {
      title: '时间',
      dataIndex: 'createdAt',
      width: 180,
      render: renderTime
    }
  ];
  const syncTaskFileColumns = [
    {
      title: '文件',
      dataIndex: 'key',
      width: 220,
      render: renderInlineText
    },
    {
      title: '游标',
      dataIndex: 'cursor',
      width: 180,
      render: renderInlineText
    },
    {
      title: '记录',
      dataIndex: 'records',
      width: 80,
      render: (text) => text || 0
    },
    {
      title: '大小',
      dataIndex: 'size',
      width: 100,
      render: renderFileSize
    },
    {
      title: 'ETag',
      dataIndex: 'eTag',
      width: 180,
      render: renderInlineText
    },
    {
      title: '更新时间',
      dataIndex: 'lastModified',
      width: 180,
      render: (text) => renderTime(text)
    }
  ];
  const syncTaskSkippedFileColumns = [
    ...syncTaskFileColumns.slice(0, 2),
    {
      title: '原因',
      dataIndex: 'reason',
      width: 110,
      render: (text) => text === 'duplicate' ? '重复文件' : (text || '-')
    },
    ...syncTaskFileColumns.slice(3)
  ];

  const resetQuery = (patch) => setQuery({ ...query, ...patch, currentPage: 1 });
  const resetInsightQuery = (patch) => setInsightQuery({ ...insightQuery, ...patch, currentPage: 1 });
  const updatePeriod = (value) => {
    const nextPeriod = value || currentPeriod;
    setPeriod(nextPeriod);
    setQuery({ ...query, period: nextPeriod, currentPage: 1 });
    setInsightQuery({ ...insightQuery, period: nextPeriod, currentPage: 1 });
  };
  async function updateInsightStatus(record, status) {
    await requestWrapper(cloudCostAPI.updateInsightStatus.bind(null, {
      orgId,
      id: record.id,
      data: { status }
    }), { autoSuccess: true });
    fetchInsightSummary();
    fetchInsights();
  }
  const budgetDefaults = {
    scope: 'org',
    period,
    currency: summary.currency || 'CNY',
    thresholdPercent: 80,
    evaluationInterval: 3600,
    status: 'enable'
  };
  const syncScheduleDefaults = {
    provider: 'azure',
    source: 'azure_cost_management',
    period,
    currency: summary.currency || 'CNY',
    maxFiles: 20,
    sourceObjectProvider: 'azure_blob',
    syncInterval: 86400,
    maxRetryAttempts: 3,
    retryBackoffSeconds: 300,
    notifyOnFailure: true,
    autoPauseOnFailure: true,
    status: 'enable'
  };
  function openBudgetModal(record) {
    setBudgetModal({
      visible: true,
      record: record || null
    });
    budgetForm.setFieldsValue(record ? {
      name: record.name,
      description: record.description,
      scope: record.scope || 'org',
      projectId: record.projectId,
      envId: record.envId,
      provider: record.provider,
      accountId: record.accountId,
      application: record.application,
      businessLine: record.businessLine,
      costCenter: record.costCenter,
      owner: record.owner,
      period: record.period || period,
      currency: record.currency || summary.currency || 'CNY',
      limitAmount: record.limitAmount,
      thresholdPercent: record.thresholdPercent || 80,
      evaluationInterval: record.evaluationInterval || 3600,
      status: record.status || 'enable'
    } : budgetDefaults);
  }
  const closeBudgetModal = () => {
    setBudgetModal({ visible: false, record: null });
    budgetForm.resetFields();
  };
  function openSyncScheduleModal(record) {
    setSyncScheduleModal({
      visible: true,
      record: record || null
    });
    syncScheduleForm.setFieldsValue(record ? {
      name: record.name,
      description: record.description,
      provider: record.provider || 'azure',
      source: record.source || 'azure_cost_management',
      cloudAccountId: record.cloudAccountId,
      accountId: record.accountId,
      region: record.region,
      period: record.period || period,
      currency: record.currency || summary.currency || 'CNY',
      sourceUrl: '',
      sourceIndexUrl: '',
      sourceObjectProvider: record.params && record.params.sourceObjectProvider,
      sourceObjectEndpoint: '',
      sourceObjectBucket: record.params && record.params.sourceObjectBucket,
      sourceObjectPrefix: record.params && record.params.sourceObjectPrefix,
      sourceObjectBaseUrl: '',
      cursor: record.params && record.params.cursor,
      maxFiles: (record.params && record.params.maxFiles) || 20,
      syncInterval: record.syncInterval || 86400,
      maxRetryAttempts: record.maxRetryAttempts || (record.params && record.params.maxRetryAttempts) || 3,
      retryBackoffSeconds: record.retryBackoffSeconds || (record.params && record.params.retryBackoffSeconds) || 300,
      notifyOnFailure: record.notifyOnFailure !== undefined ? record.notifyOnFailure : !!(record.params && record.params.notifyOnFailure),
      autoPauseOnFailure: record.autoPauseOnFailure !== undefined ? record.autoPauseOnFailure : !!(record.params && record.params.autoPauseOnFailure),
      status: record.status || 'enable'
    } : syncScheduleDefaults);
  }
  const closeSyncScheduleModal = () => {
    setSyncScheduleModal({ visible: false, record: null });
    syncScheduleForm.resetFields();
  };
  const refreshBudgets = () => {
    fetchBudgetSummary();
    fetchBudgets();
  };
  const refreshSyncSchedules = () => {
    fetchSyncSchedules();
    fetchSyncTasks();
  };
  const evaluateBudgets = async () => {
    await requestWrapper(cloudCostAPI.evaluateBudgets.bind(null, {
      orgId,
      data: {
        period,
        currency: summary.currency || 'CNY',
        force: true
      }
    }), { autoSuccess: true });
    refreshAll();
  };
  const openImportModal = () => {
    setImportModalVisible(true);
    importForm.setFieldsValue({
      provider: 'azure',
      source: 'azure_billing_export',
      period,
      currency: summary.currency || 'CNY',
      records: `[
  {
    "resourceId": "/subscriptions/.../resourceGroups/.../providers/Microsoft.Compute/virtualMachines/demo",
    "service": "Virtual Machines",
    "amount": 12.34,
    "currency": "CNY",
    "period": "${period}"
  }
]`
    });
  };
  const closeImportModal = () => {
    setImportModalVisible(false);
    importForm.resetFields();
  };
  const openPullModal = () => {
    setPullModalVisible(true);
    pullForm.setFieldsValue({
      provider: 'azure',
      source: 'azure_cost_management',
      period,
      currency: summary.currency || 'CNY',
      sourceObjectProvider: 'azure_blob',
      maxFiles: 20
    });
  };
  const closePullModal = () => {
    setPullModalVisible(false);
    pullForm.resetFields();
  };
  const openSyncTaskDrawer = (record) => {
    setSyncTaskDrawer({ visible: true, id: record.id });
    fetchSyncTaskDetail(record.id);
  };
  const closeSyncTaskDrawer = () => {
    setSyncTaskDrawer({ visible: false, id: '' });
  };
  const submitImport = async () => {
    const values = await importForm.validateFields();
    let parsed;
    try {
      parsed = JSON.parse(values.records || '[]');
    } catch (err) {
      Modal.error({
        title: 'JSON 格式错误',
        content: err.message
      });
      return;
    }
    const records = Array.isArray(parsed) ? parsed : parsed.records;
    if (!Array.isArray(records) || records.length === 0) {
      Modal.error({
        title: '导入内容为空',
        content: '请粘贴 JSON 数组，或包含 records 数组的 JSON 对象。'
      });
      return;
    }
    setImportSaving(true);
    try {
      await requestWrapper(cloudCostAPI.importRecords.bind(null, {
        orgId,
        data: {
          provider: values.provider,
          source: values.source,
          period: values.period || period,
          currency: values.currency || summary.currency || 'CNY',
          accountId: values.accountId,
          region: values.region,
          records
        }
      }), { autoSuccess: true });
      closeImportModal();
      refreshAll();
    } finally {
      setImportSaving(false);
    }
  };
  const submitPull = async () => {
    const values = await pullForm.validateFields();
    setPullSaving(true);
    try {
      await requestWrapper(cloudCostAPI.createSyncTask.bind(null, {
        orgId,
        data: {
          provider: values.provider,
          source: values.source,
          cloudAccountId: values.cloudAccountId,
          accountId: values.accountId,
          region: values.region,
          period: values.period || period,
          currency: values.currency || summary.currency || 'CNY',
          sourceUrl: values.sourceUrl,
          sourceIndexUrl: values.sourceIndexUrl,
          sourceObjectProvider: values.sourceObjectProvider,
          sourceObjectEndpoint: values.sourceObjectEndpoint,
          sourceObjectBucket: values.sourceObjectBucket,
          sourceObjectPrefix: values.sourceObjectPrefix,
          sourceObjectBaseUrl: values.sourceObjectBaseUrl,
          cursor: values.cursor,
          maxFiles: values.maxFiles
        }
      }), { autoSuccess: true });
      closePullModal();
      refreshAll();
    } finally {
      setPullSaving(false);
    }
  };
  const submitSyncSchedule = async () => {
    const values = await syncScheduleForm.validateFields();
    const payload = {
      name: values.name,
      description: values.description,
      provider: values.provider,
      source: values.source,
      cloudAccountId: values.cloudAccountId,
      accountId: values.accountId,
      region: values.region,
      period: values.period || period,
      currency: values.currency || summary.currency || 'CNY',
      sourceUrl: values.sourceUrl,
      sourceIndexUrl: values.sourceIndexUrl,
      sourceObjectProvider: values.sourceObjectProvider,
      sourceObjectEndpoint: values.sourceObjectEndpoint,
      sourceObjectBucket: values.sourceObjectBucket,
      sourceObjectPrefix: values.sourceObjectPrefix,
      sourceObjectBaseUrl: values.sourceObjectBaseUrl,
      cursor: values.cursor,
      maxFiles: values.maxFiles,
      syncInterval: values.syncInterval,
      maxRetryAttempts: values.maxRetryAttempts,
      retryBackoffSeconds: values.retryBackoffSeconds,
      notifyOnFailure: values.notifyOnFailure,
      autoPauseOnFailure: values.autoPauseOnFailure,
      status: values.status
    };
    const record = syncScheduleModal.record;
    setSyncScheduleSaving(true);
    try {
      await requestWrapper(
        record
          ? cloudCostAPI.updateSyncSchedule.bind(null, { orgId, id: record.id, data: payload })
          : cloudCostAPI.createSyncSchedule.bind(null, { orgId, data: payload }),
        { autoSuccess: true }
      );
      closeSyncScheduleModal();
      refreshSyncSchedules();
    } finally {
      setSyncScheduleSaving(false);
    }
  };
  async function runSyncSchedule(record) {
    const detail = await requestWrapper(cloudCostAPI.runSyncSchedule.bind(null, {
      orgId,
      id: record.id
    }), { autoSuccess: true });
    refreshAll();
    if (detail && detail.id) {
      setSyncTaskDrawer({ visible: true, id: detail.id });
      fetchSyncTaskDetail(detail.id);
    }
  }
  const runDueSyncSchedules = async () => {
    await requestWrapper(cloudCostAPI.runDueSyncSchedules.bind(null, {
      orgId,
      data: { force: false }
    }), { autoSuccess: true });
    refreshAll();
  };
  function deleteSyncSchedule(record) {
    Modal.confirm({
      title: '删除账单同步计划',
      content: `确认删除同步计划「${record.name}」？`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        await requestWrapper(cloudCostAPI.deleteSyncSchedule.bind(null, { orgId, id: record.id }), { autoSuccess: true });
        refreshSyncSchedules();
      }
    });
  }
  function retrySyncTask(record) {
    Modal.confirm({
      title: '重试账单同步任务',
      content: `确认重试任务 ${record.id}？`,
      okText: '重试',
      cancelText: '取消',
      onOk: async () => {
        await requestWrapper(cloudCostAPI.retrySyncTask.bind(null, { orgId, id: record.id }), { autoSuccess: true });
        fetchSyncTasks();
        if (syncTaskDrawer.id === record.id) {
          fetchSyncTaskDetail(record.id);
        }
        refreshAll();
      }
    });
  }
  const normalizeBudgetPayload = (values) => {
    const field = budgetScopeFieldMap[values.scope];
    const payload = { ...values };
    [ 'projectId', 'envId', 'provider', 'accountId', 'application', 'businessLine', 'costCenter', 'owner' ].forEach((name) => {
      if (!field || field.name !== name) {
        payload[name] = '';
      }
    });
    return payload;
  };
  const submitBudget = async () => {
    const values = await budgetForm.validateFields();
    const payload = {
      ...normalizeBudgetPayload(values),
      period: values.period || period,
      currency: values.currency || summary.currency || 'CNY'
    };
    const record = budgetModal.record;
    setBudgetSaving(true);
    try {
      await requestWrapper(
        record
          ? cloudCostAPI.updateBudget.bind(null, { orgId, id: record.id, data: payload })
          : cloudCostAPI.createBudget.bind(null, { orgId, data: payload }),
        { autoSuccess: true }
      );
      closeBudgetModal();
      refreshBudgets();
    } finally {
      setBudgetSaving(false);
    }
  };
  function deleteBudget(record) {
    Modal.confirm({
      title: '删除预算',
      content: `确认删除预算「${record.name}」？`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        await requestWrapper(cloudCostAPI.deleteBudget.bind(null, { orgId, id: record.id }), { autoSuccess: true });
        refreshBudgets();
      }
    });
  }
  const renderBudgetScopeField = () => (
    <Form.Item noStyle={true} shouldUpdate={(prev, next) => prev.scope !== next.scope}>
      {({ getFieldValue }) => {
        const scope = getFieldValue('scope');
        const field = budgetScopeFieldMap[scope];
        if (!field) {
          return null;
        }
        return (
          <Form.Item
            name={field.name}
            label={field.label}
            rules={[{ required: true, message: field.placeholder }]}
          >
            <Input placeholder={field.placeholder}/>
          </Form.Item>
        );
      }}
    </Form.Item>
  );
  const syncTaskResult = syncTaskDetail.result || {};
  const syncTaskFiles = syncTaskResult.files || [];
  const syncTaskSkippedFiles = syncTaskResult.skippedFiles || [];

  return (
    <Layout
      extraHeader={<PageHeader title='成本中心' breadcrumb={true}/>}
    >
      <div className='idcos-card'>
        <div className={styles.metrics}>
          <Metric label='总成本' value={money(summary.totalAmount, summary.currency)} tone={(summary.totalAmount || 0) > 0 ? 'moneyTone' : ''}/>
          <Metric label='成本记录' value={summary.recordCount || 0}/>
          <Metric label='已匹配资产' value={summary.matchedCount || 0}/>
          <Metric label='未匹配账单' value={summary.unmatchedCount || 0} tone={summary.unmatchedCount ? 'warningTone' : ''}/>
        </div>
        <div className={styles.summaryGrid}>
          <div>
            <Text strong={true}>Provider 成本</Text>
            <Table rowKey='key' size='small' columns={groupColumns} dataSource={summary.providers || []} pagination={false} loading={summaryLoading}/>
          </div>
          <div>
            <Text strong={true}>应用成本</Text>
            <Table rowKey='key' size='small' columns={groupColumns} dataSource={summary.applications || []} pagination={false} loading={summaryLoading}/>
          </div>
          <div>
            <Text strong={true}>近 6 个月趋势</Text>
            <Table rowKey='period' size='small' columns={trendColumns} dataSource={trends || []} pagination={false} loading={trendsLoading}/>
          </div>
        </div>
        <div className={styles.budgetPanel}>
          <div className={styles.budgetHeader}>
            <Space size={8}>
              <Text strong={true}>预算管理</Text>
              {budgetSummaryLoading ? null : <Tag>{budgetSummary.budgetCount || 0} 个预算</Tag>}
            </Space>
            <Space size={8}>
              <Button icon={<SyncOutlined/>} onClick={evaluateBudgets}>立即评估</Button>
              <Button type='primary' icon={<PlusOutlined/>} onClick={() => openBudgetModal()}>新建预算</Button>
            </Space>
          </div>
          <div className={styles.budgetMetrics}>
            <Metric label='预算总额' value={money(budgetSummary.totalLimitAmount, budgetSummary.currency || summary.currency)}/>
            <Metric label='当前消耗' value={money(budgetSummary.totalCurrentAmount, budgetSummary.currency || summary.currency)} tone={(budgetSummary.totalCurrentAmount || 0) > 0 ? 'moneyTone' : ''}/>
            <Metric label='超阈值预算' value={budgetSummary.thresholdCount || 0} tone={budgetSummary.thresholdCount ? 'warningTone' : ''}/>
            <Metric label='超预算' value={budgetSummary.exceededCount || 0} tone={budgetSummary.exceededCount ? 'warningTone' : ''}/>
          </div>
          <Table
            rowKey='id'
            size='small'
            columns={budgetColumns}
            dataSource={budgetList}
            loading={budgetLoading}
            scroll={{ x: 'min-content' }}
            pagination={false}
            locale={{ emptyText: <Empty description='暂无预算'/> }}
          />
        </div>
        <div className={styles.insightPanel}>
          <div className={styles.budgetHeader}>
            <Space size={8}>
              <Text strong={true}>成本异常与优化建议</Text>
              {insightSummaryLoading ? null : <Tag>{insightSummary.openCount || 0} 条待处理</Tag>}
            </Space>
            <Space size={[8, 8]} wrap={true}>
              <InputSearch
                className={styles.keywordSearch}
                allowClear={true}
                placeholder='搜索建议或资源'
                onSearch={(value) => resetInsightQuery({ q: value })}
              />
              <Select
                allowClear={true}
                placeholder='类型'
                style={{ width: 130 }}
                onChange={(type) => resetInsightQuery({ type })}
              >
                {Object.entries(insightTypeMap).map(([value, label]) => (
                  <Option key={value} value={value}>{label}</Option>
                ))}
              </Select>
              <Select
                value={insightQuery.status}
                placeholder='状态'
                style={{ width: 120 }}
                onChange={(status) => resetInsightQuery({ status })}
              >
                {Object.entries(insightStatusMap).map(([value, item]) => (
                  <Option key={value} value={value}>{item.label}</Option>
                ))}
              </Select>
            </Space>
          </div>
          <div className={styles.budgetMetrics}>
            <Metric label='待处理' value={insightSummary.openCount || 0} tone={insightSummary.openCount ? 'warningTone' : ''}/>
            <Metric label='高等级' value={insightSummary.highCount || 0} tone={insightSummary.highCount ? 'warningTone' : ''}/>
            <Metric label='异常' value={insightSummary.anomalyCount || 0}/>
            <Metric label='优化建议' value={insightSummary.optimizationCount || 0}/>
            <Metric label='预计可节省' value={money(insightSummary.potentialSavings, insightSummary.currency || summary.currency)} tone={(insightSummary.potentialSavings || 0) > 0 ? 'moneyTone' : ''}/>
          </div>
          <Table
            rowKey='id'
            size='small'
            columns={insightColumns}
            dataSource={insightList}
            loading={insightLoading}
            scroll={{ x: 'min-content' }}
            pagination={{
              current: insightQuery.currentPage,
              pageSize: insightQuery.pageSize,
              total: insightData.total || 0,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total) => `共${total}条`,
              onChange: (currentPage, pageSize) => setInsightQuery({ ...insightQuery, currentPage, pageSize })
            }}
            locale={{ emptyText: <Empty description='暂无成本建议'/> }}
          />
        </div>
        <div className={styles.syncPanel}>
          <div className={styles.budgetHeader}>
            <Space size={8}>
              <Text strong={true}>账单同步计划</Text>
              {syncScheduleLoading ? null : <Tag>{syncScheduleData.total || 0} 个计划</Tag>}
            </Space>
            <Space size={8}>
              <Button icon={<SyncOutlined/>} onClick={runDueSyncSchedules}>扫描到期</Button>
              <Button type='primary' icon={<PlusOutlined/>} onClick={() => openSyncScheduleModal()}>新建计划</Button>
            </Space>
          </div>
          <Table
            rowKey='id'
            size='small'
            columns={syncScheduleColumns}
            dataSource={syncScheduleList}
            loading={syncScheduleLoading}
            scroll={{ x: 'min-content' }}
            pagination={false}
            locale={{ emptyText: <Empty description='暂无账单同步计划'/> }}
          />
        </div>
        <div className={styles.syncPanel}>
          <div className={styles.budgetHeader}>
            <Space size={8}>
              <Text strong={true}>账单同步任务</Text>
              {syncTaskLoading ? null : <Tag>{syncTaskData.total || 0} 条</Tag>}
            </Space>
            <Space size={8}>
              <Button icon={<CloudDownloadOutlined/>} onClick={openPullModal}>拉取账单</Button>
              <Button icon={<ReloadOutlined/>} onClick={fetchSyncTasks}>刷新任务</Button>
            </Space>
          </div>
          <Table
            rowKey='id'
            size='small'
            columns={syncTaskColumns}
            dataSource={syncTaskList}
            loading={syncTaskLoading}
            scroll={{ x: 'min-content' }}
            pagination={false}
            locale={{ emptyText: <Empty description='暂无账单同步任务'/> }}
          />
        </div>
        <div className={styles.toolbar}>
          <Space className={styles.filterBar} size={[8, 8]} wrap={true}>
            <InputSearch
              className={styles.keywordSearch}
              allowClear={true}
              placeholder='搜索资源、服务或载荷'
              onSearch={(value) => resetQuery({ q: value })}
            />
            <Input
              className={styles.periodInput}
              value={period}
              placeholder='YYYY-MM'
              onChange={(event) => updatePeriod(event.target.value)}
            />
            <Select
              allowClear={true}
              placeholder='来源'
              style={{ width: 130 }}
              onChange={(source) => resetQuery({ source })}
            >
              {Object.entries(sourceMap).map(([value, label]) => (
                <Option key={value} value={value}>{label}</Option>
              ))}
            </Select>
            <Select
              allowClear={true}
              placeholder='资产匹配'
              style={{ width: 130 }}
              onChange={(matchedAsset) => resetQuery({ matchedAsset })}
            >
              <Option value='true'>已匹配</Option>
              <Option value='false'>未匹配</Option>
            </Select>
          </Space>
          <Space size={8}>
            <Button icon={<CloudDownloadOutlined/>} onClick={openPullModal}>拉取账单</Button>
            <Button icon={<PlusOutlined/>} onClick={openImportModal}>导入账单</Button>
            <Button icon={<ReloadOutlined/>} onClick={refreshAll}>刷新</Button>
          </Space>
        </div>
        <Table
          rowKey='id'
          columns={columns}
          dataSource={list}
          loading={loading}
          scroll={{ x: 'min-content' }}
          pagination={{
            current: query.currentPage,
            pageSize: query.pageSize,
            total: data.total || 0,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共${total}条`,
            onChange: (currentPage, pageSize) => setQuery({ ...query, currentPage, pageSize })
          }}
        />
      </div>
      <Drawer
        title='成本记录详情'
        visible={drawer.visible}
        width={860}
        destroyOnClose={true}
        onClose={() => setDrawer({ visible: false, id: '' })}
      >
        <CostDetail detail={detail} loading={loading}/>
      </Drawer>
      <Drawer
        title='账单同步任务详情'
        visible={syncTaskDrawer.visible}
        width={860}
        destroyOnClose={true}
        onClose={closeSyncTaskDrawer}
      >
        {syncTaskDetailLoading ? (
          <Empty description='加载中'/>
        ) : (
          <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
            <Descriptions size='small' bordered={true} column={2}>
              <Descriptions.Item label='任务 ID'>{syncTaskDetail.id || '-'}</Descriptions.Item>
              <Descriptions.Item label='状态'>
                {(() => {
                  const item = syncTaskStatusMap[syncTaskDetail.status] || {};
                  return <Tag color={item.color}>{item.label || syncTaskDetail.status || '-'}</Tag>;
                })()}
              </Descriptions.Item>
              <Descriptions.Item label='云厂商'>{syncTaskDetail.provider || '-'}</Descriptions.Item>
              <Descriptions.Item label='来源'>{pullSourceMap[syncTaskDetail.source] || sourceMap[syncTaskDetail.source] || syncTaskDetail.source || '-'}</Descriptions.Item>
              <Descriptions.Item label='账期'>{syncTaskDetail.period || '-'}</Descriptions.Item>
              <Descriptions.Item label='币种'>{syncTaskDetail.currency || '-'}</Descriptions.Item>
              <Descriptions.Item label='导入数量'>{syncTaskDetail.imported || 0}</Descriptions.Item>
              <Descriptions.Item label='匹配数量'>{syncTaskDetail.matchedCount || 0}</Descriptions.Item>
              <Descriptions.Item label='账号'>{syncTaskDetail.accountId || syncTaskDetail.cloudAccountName || syncTaskDetail.cloudAccountId || '-'}</Descriptions.Item>
              <Descriptions.Item label='区域'>{syncTaskDetail.region || '-'}</Descriptions.Item>
              <Descriptions.Item label='开始时间'>{renderTime(syncTaskDetail.startedAt)}</Descriptions.Item>
              <Descriptions.Item label='结束时间'>{renderTime(syncTaskDetail.endedAt)}</Descriptions.Item>
              <Descriptions.Item label='消息' span={2}>{syncTaskDetail.error || syncTaskDetail.message || '-'}</Descriptions.Item>
              <Descriptions.Item label='来源 URL' span={2}>{syncTaskDetail.sourceUrlPreview || '-'}</Descriptions.Item>
              <Descriptions.Item label='索引 URL' span={2}>{syncTaskDetail.sourceIndexUrlPreview || '-'}</Descriptions.Item>
              <Descriptions.Item label='对象存储 Endpoint' span={2}>{syncTaskDetail.sourceObjectEndpointPreview || '-'}</Descriptions.Item>
            </Descriptions>
            <div className={styles.detailSection}>
              <Text strong={true}>导入文件</Text>
              <Table
                rowKey={(record) => `${record.key || record.url || ''}-${record.cursor || ''}`}
                size='small'
                columns={syncTaskFileColumns}
                dataSource={syncTaskFiles}
                pagination={false}
                scroll={{ x: 'min-content' }}
                locale={{ emptyText: <Empty description='暂无导入文件'/> }}
              />
            </div>
            <div className={styles.detailSection}>
              <Text strong={true}>跳过文件</Text>
              <Table
                rowKey={(record) => `${record.reason || ''}-${record.key || record.url || ''}-${record.cursor || ''}`}
                size='small'
                columns={syncTaskSkippedFileColumns}
                dataSource={syncTaskSkippedFiles}
                pagination={false}
                scroll={{ x: 'min-content' }}
                locale={{ emptyText: <Empty description='暂无跳过文件'/> }}
              />
            </div>
            <div className={styles.detailSection}>
              <Text strong={true}>任务日志</Text>
              <Table
                rowKey='id'
                size='small'
                columns={syncTaskLogColumns}
                dataSource={syncTaskDetail.logs || []}
                pagination={false}
                locale={{ emptyText: <Empty description='暂无任务日志'/> }}
              />
            </div>
            <div className={styles.detailSection}>
              <Text strong={true}>执行结果</Text>
              <JsonBlock value={syncTaskDetail.result}/>
            </div>
          </Space>
        )}
      </Drawer>
      <Modal
        title={budgetModal.record ? '编辑预算' : '新建预算'}
        visible={budgetModal.visible}
        width={720}
        destroyOnClose={true}
        confirmLoading={budgetSaving}
        onCancel={closeBudgetModal}
        onOk={submitBudget}
      >
        <Form form={budgetForm} layout='vertical' initialValues={budgetDefaults}>
          <div className={styles.budgetFormGrid}>
            <Form.Item name='name' label='预算名称' rules={[{ required: true, message: '请输入预算名称' }]}>
              <Input placeholder='请输入预算名称'/>
            </Form.Item>
            <Form.Item name='scope' label='预算范围' rules={[{ required: true, message: '请选择预算范围' }]}>
              <Select>
                {Object.entries(budgetScopeMap).map(([value, label]) => (
                  <Option key={value} value={value}>{label}</Option>
                ))}
              </Select>
            </Form.Item>
            {renderBudgetScopeField()}
            <Form.Item name='period' label='账期' rules={[{ required: true, message: '请输入账期' }]}>
              <Input placeholder='YYYY-MM'/>
            </Form.Item>
            <Form.Item name='currency' label='币种' rules={[{ required: true, message: '请输入币种' }]}>
              <Input placeholder='CNY'/>
            </Form.Item>
            <Form.Item name='limitAmount' label='预算金额' rules={[{ required: true, message: '请输入预算金额' }]}>
              <InputNumber min={0.01} precision={2} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='thresholdPercent' label='预警阈值' rules={[{ required: true, message: '请输入预警阈值' }]}>
              <InputNumber min={1} max={100} precision={1} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='evaluationInterval' label='评估间隔'>
              <InputNumber min={60} max={86400} precision={0} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='status' label='状态'>
              <Select>
                <Option value='enable'>启用</Option>
                <Option value='disable'>停用</Option>
              </Select>
            </Form.Item>
          </div>
          <Form.Item name='description' label='说明'>
            <Input.TextArea rows={3} placeholder='可填写预算口径、归属或审批说明'/>
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title={syncScheduleModal.record ? '编辑账单同步计划' : '新建账单同步计划'}
        visible={syncScheduleModal.visible}
        width={720}
        destroyOnClose={true}
        confirmLoading={syncScheduleSaving}
        onCancel={closeSyncScheduleModal}
        onOk={submitSyncSchedule}
      >
        <Form form={syncScheduleForm} layout='vertical' initialValues={syncScheduleDefaults}>
          <div className={styles.budgetFormGrid}>
            <Form.Item name='name' label='计划名称' rules={[{ required: true, message: '请输入计划名称' }]}>
              <Input placeholder='请输入计划名称'/>
            </Form.Item>
            <Form.Item name='status' label='状态'>
              <Select>
                <Option value='enable'>启用</Option>
                <Option value='disable'>停用</Option>
              </Select>
            </Form.Item>
            <Form.Item name='provider' label='云厂商' rules={[{ required: true, message: '请选择云厂商' }]}>
              <Select>
                <Option value='aws'>AWS</Option>
                <Option value='oci'>OCI</Option>
                <Option value='azure'>Azure</Option>
                <Option value='gcp'>GCP</Option>
                <Option value='tencentcloud'>腾讯云</Option>
                <Option value='huawei'>华为云</Option>
              </Select>
            </Form.Item>
            <Form.Item name='source' label='来源' rules={[{ required: true, message: '请选择来源' }]}>
              <Select>
                {Object.entries(pullSourceMap).map(([value, label]) => (
                  <Option key={value} value={value}>{label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name='cloudAccountId' label='云账号 ID'>
              <Input placeholder='使用云账号凭证时填写'/>
            </Form.Item>
            <Form.Item name='sourceUrl' label='账单导出 URL'>
              <Input placeholder={syncScheduleModal.record ? '留空保留原地址' : '使用 Billing Export URL 时填写'}/>
            </Form.Item>
            <Form.Item name='sourceIndexUrl' label='账单索引 URL'>
              <Input placeholder={syncScheduleModal.record ? '留空保留原索引地址' : '使用对象存储文件列表时填写'}/>
            </Form.Item>
            <Form.Item name='sourceObjectProvider' label='对象存储类型'>
              <Select allowClear={true} placeholder='使用原生对象存储列表时选择'>
                {Object.entries(objectStorageProviderMap).map(([value, label]) => (
                  <Option key={value} value={value}>{label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name='sourceObjectEndpoint' label='对象存储 Endpoint'>
              <Input placeholder={syncScheduleModal.record ? '留空保留原 Endpoint' : '例如 https://storage.googleapis.com/storage/v1'}/>
            </Form.Item>
            <Form.Item name='sourceObjectBucket' label='Bucket / Container'>
              <Input placeholder='请输入 Bucket 或 Container 名称'/>
            </Form.Item>
            <Form.Item name='sourceObjectPrefix' label='对象前缀'>
              <Input placeholder='例如 billing/2026-06/'/>
            </Form.Item>
            <Form.Item name='sourceObjectBaseUrl' label='文件 Base URL'>
              <Input placeholder={syncScheduleModal.record ? '留空保留原 Base URL' : '可选，文件下载地址与列表地址不同时填写'}/>
            </Form.Item>
            <Form.Item name='period' label='账期'>
              <Input placeholder='YYYY-MM'/>
            </Form.Item>
            <Form.Item name='currency' label='币种'>
              <Input placeholder='CNY'/>
            </Form.Item>
            <Form.Item name='accountId' label='账号/订阅/项目 ID'>
              <Input placeholder='可选，未填写时从云账号或记录载荷推断'/>
            </Form.Item>
            <Form.Item name='region' label='区域'>
              <Input placeholder='可选，未填写时从记录载荷推断'/>
            </Form.Item>
            <Form.Item name='cursor' label='增量游标'>
              <Input placeholder='可选，留空从第一批文件开始'/>
            </Form.Item>
            <Form.Item name='maxFiles' label='单次最大文件数'>
              <InputNumber min={1} max={100} precision={0} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='syncInterval' label='同步间隔'>
              <InputNumber min={60} max={2592000} precision={0} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='maxRetryAttempts' label='失败最大重试次数'>
              <InputNumber min={0} max={10} precision={0} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='retryBackoffSeconds' label='失败退避秒数'>
              <InputNumber min={60} max={86400} precision={0} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='notifyOnFailure' valuePropName='checked'>
              <Checkbox>失败时发送通知事件</Checkbox>
            </Form.Item>
            <Form.Item name='autoPauseOnFailure' valuePropName='checked'>
              <Checkbox>超过重试次数后自动暂停</Checkbox>
            </Form.Item>
          </div>
          <Form.Item name='description' label='说明'>
            <Input.TextArea rows={3} placeholder='可填写账单来源、归属或同步窗口'/>
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title='拉取账单'
        visible={pullModalVisible}
        width={720}
        destroyOnClose={true}
        confirmLoading={pullSaving}
        onCancel={closePullModal}
        onOk={submitPull}
      >
        <Form form={pullForm} layout='vertical'>
          <div className={styles.budgetFormGrid}>
            <Form.Item name='provider' label='云厂商' rules={[{ required: true, message: '请选择云厂商' }]}>
              <Select>
                <Option value='aws'>AWS</Option>
                <Option value='oci'>OCI</Option>
                <Option value='azure'>Azure</Option>
                <Option value='gcp'>GCP</Option>
                <Option value='tencentcloud'>腾讯云</Option>
                <Option value='huawei'>华为云</Option>
              </Select>
            </Form.Item>
            <Form.Item name='source' label='来源' rules={[{ required: true, message: '请选择来源' }]}>
              <Select>
                {Object.entries(pullSourceMap).map(([value, label]) => (
                  <Option key={value} value={value}>{label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name='cloudAccountId' label='云账号 ID'>
              <Input placeholder='使用云账号凭证时填写'/>
            </Form.Item>
            <Form.Item name='sourceUrl' label='账单导出 URL'>
              <Input placeholder='使用 Billing Export URL 时填写'/>
            </Form.Item>
            <Form.Item name='sourceIndexUrl' label='账单索引 URL'>
              <Input placeholder='使用对象存储文件列表时填写'/>
            </Form.Item>
            <Form.Item name='sourceObjectProvider' label='对象存储类型'>
              <Select allowClear={true} placeholder='使用原生对象存储列表时选择'>
                {Object.entries(objectStorageProviderMap).map(([value, label]) => (
                  <Option key={value} value={value}>{label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name='sourceObjectEndpoint' label='对象存储 Endpoint'>
              <Input placeholder='例如 https://storage.googleapis.com/storage/v1'/>
            </Form.Item>
            <Form.Item name='sourceObjectBucket' label='Bucket / Container'>
              <Input placeholder='请输入 Bucket 或 Container 名称'/>
            </Form.Item>
            <Form.Item name='sourceObjectPrefix' label='对象前缀'>
              <Input placeholder='例如 billing/2026-06/'/>
            </Form.Item>
            <Form.Item name='sourceObjectBaseUrl' label='文件 Base URL'>
              <Input placeholder='可选，文件下载地址与列表地址不同时填写'/>
            </Form.Item>
            <Form.Item name='period' label='账期'>
              <Input placeholder='YYYY-MM'/>
            </Form.Item>
            <Form.Item name='currency' label='币种'>
              <Input placeholder='CNY'/>
            </Form.Item>
            <Form.Item name='accountId' label='账号/订阅/项目 ID'>
              <Input placeholder='可选，未填写时从云账号或记录载荷推断'/>
            </Form.Item>
            <Form.Item name='region' label='区域'>
              <Input placeholder='可选，未填写时从记录载荷推断'/>
            </Form.Item>
            <Form.Item name='cursor' label='增量游标'>
              <Input placeholder='可选，留空从第一批文件开始'/>
            </Form.Item>
            <Form.Item name='maxFiles' label='单次最大文件数'>
              <InputNumber min={1} max={100} precision={0} style={{ width: '100%' }}/>
            </Form.Item>
          </div>
        </Form>
      </Modal>
      <Modal
        title='导入账单'
        visible={importModalVisible}
        width={760}
        destroyOnClose={true}
        confirmLoading={importSaving}
        onCancel={closeImportModal}
        onOk={submitImport}
      >
        <Form form={importForm} layout='vertical'>
          <div className={styles.budgetFormGrid}>
            <Form.Item name='provider' label='云厂商' rules={[{ required: true, message: '请选择云厂商' }]}>
              <Select>
                <Option value='aws'>AWS</Option>
                <Option value='oci'>OCI</Option>
                <Option value='azure'>Azure</Option>
                <Option value='gcp'>GCP</Option>
                <Option value='tencentcloud'>腾讯云</Option>
                <Option value='huawei'>华为云</Option>
              </Select>
            </Form.Item>
            <Form.Item name='source' label='来源' rules={[{ required: true, message: '请选择来源' }]}>
              <Select>
                {Object.entries(importSourceMap).map(([value, label]) => (
                  <Option key={value} value={value}>{label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name='period' label='账期'>
              <Input placeholder='YYYY-MM'/>
            </Form.Item>
            <Form.Item name='currency' label='币种'>
              <Input placeholder='CNY'/>
            </Form.Item>
            <Form.Item name='accountId' label='账号/订阅/项目 ID'>
              <Input placeholder='可选，未填写时从记录载荷推断'/>
            </Form.Item>
            <Form.Item name='region' label='区域'>
              <Input placeholder='可选，未填写时从记录载荷推断'/>
            </Form.Item>
          </div>
          <Form.Item name='records' label='账单 JSON' rules={[{ required: true, message: '请粘贴账单 JSON' }]}>
            <Input.TextArea rows={12}/>
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  );
};

export default CloudCostPage;
