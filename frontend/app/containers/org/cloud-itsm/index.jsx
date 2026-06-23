import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Progress,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography
} from 'antd';
import { CheckCircleOutlined, ClockCircleOutlined, DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, SafetyCertificateOutlined, SyncOutlined, ToolOutlined } from '@ant-design/icons';
import { useRequest } from 'ahooks';
import moment from 'moment';
import PageHeader from 'components/pageHeader';
import Layout from 'components/common/layout';
import { requestWrapper } from 'utils/request';
import cloudItsmAPI from 'services/cloud-itsm';
import styles from './styles.less';

const { Option } = Select;
const { Search: InputSearch, TextArea } = Input;
const { Text } = Typography;

const catalogPolicyActionMap = {
  update: '更新',
  reset: '恢复默认'
};

const providerMap = {
  generic: '通用 HTTP',
  jira: 'Jira',
  servicenow: 'ServiceNow'
};

const authMap = {
  none: '无认证',
  bearer: 'Bearer Token',
  basic: 'Basic Auth'
};

const configStatusMap = {
  enable: { label: '启用', color: 'success' },
  disable: { label: '停用', color: 'default' }
};

const ticketStatusMap = {
  pending: { label: '待提交', color: 'default' },
  submitted: { label: '已提交', color: 'processing' },
  failed: { label: '提交失败', color: 'error' },
  in_progress: { label: '处理中', color: 'processing' },
  resolved: { label: '已解决', color: 'success' },
  closed: { label: '已关闭', color: 'success' },
  canceled: { label: '已取消', color: 'warning' }
};

const retryQueueStatusMap = {
  due: { label: '待重试', color: 'processing' },
  future: { label: '等待退避', color: 'warning' },
  dead_letter: { label: '死信', color: 'error' },
  skipped: { label: '跳过', color: 'default' }
};

const riskMap = {
  low: { label: '低', color: 'success' },
  medium: { label: '中', color: 'warning' },
  high: { label: '高', color: 'error' },
  critical: { label: '严重', color: 'error' }
};

const catalogCategoryMap = {
  lifecycle: '生命周期',
  scaling: '扩缩容',
  configuration: '配置变更',
  backup: '备份',
  permission: '权限申请',
  gitops_iac: 'GitOps/IaC',
  drift: '漂移修复',
  risk: '风险整改',
  operations: '运维操作'
};

const automationModeMap = {
  auto: '自动处理',
  approval: '审批后执行',
  provider: '云端执行',
  provider_approval: '审批后云端执行',
  itsm: 'ITSM 工单',
  event_auto_ticket: '事件自动建单'
};

const robotChannelMap = {
  local: '本地工单',
  external: '外部 ITSM'
};

const robotTagMap = {
  self_service: '自助申请',
  cloud_operation: '云操作',
  gitops_iac: 'GitOps/IaC',
  drift_auto_repair: '漂移修复',
  risk_remediation: '风险整改',
  permission_request: '权限申请',
  event_auto_ticket: '事件建单',
  approval_required: '需审批',
  external_itsm: '外部 ITSM',
  local_ticket: '本地工单'
};

const deadLetterEvidenceTypeOptions = [
  { value: 'link', label: '链接' },
  { value: 'pull_request', label: 'PR/评审' },
  { value: 'ticket', label: '外部工单' },
  { value: 'change_request', label: '变更记录' },
  { value: 'incident', label: '事故记录' },
  { value: 'document', label: '文档' },
  { value: 'snapshot', label: '现场快照' },
  { value: 'other', label: '其他' }
];

const policyRoleMap = {
  project_member: '项目成员',
  project_owner: '项目负责人',
  iac_reviewer: 'IaC 评审',
  operator: '运维',
  sre: 'SRE',
  security_owner: '安全负责人'
};

const policyScopeMap = {
  org: '组织',
  project: '项目',
  env: '环境',
  asset: '资源'
};

const renderTime = (value) => !value || String(value).indexOf('0001-01-01') === 0 ? '-' : moment(value).format('YYYY-MM-DD HH:mm:ss');
const percent = (value) => Math.max(0, Math.min(100, Number(value || 0)));
const percentText = (value) => `${percent(value).toFixed(1)}%`;
const normalizeDeadLetterEvidenceItems = (items = []) => (items || [])
  .map((item) => ({
    type: item && item.type || 'link',
    label: item && item.label && item.label.trim(),
    url: item && item.url && item.url.trim(),
    note: item && item.note && item.note.trim()
  }))
  .filter((item) => item.label || item.url || item.note);
const statusTag = (map, value) => {
  const item = map[value] || { label: value || '-', color: 'default' };
  return <Tag color={item.color}>{item.label}</Tag>;
};
const jsonText = (value) => {
  if (!value || Object.keys(value || {}).length === 0) {
    return '-';
  }
  return JSON.stringify(value, null, 2);
};
const minutesText = (minutes) => {
  const value = Number(minutes || 0);
  if (!value) {
    return '-';
  }
  if (value % 1440 === 0) {
    return `${value / 1440} 天`;
  }
  if (value % 60 === 0) {
    return `${value / 60} 小时`;
  }
  return `${value} 分钟`;
};
const renderTagList = (values = [], labelMap = {}) => {
  const items = Array.isArray(values) ? values.filter(Boolean) : [];
  if (items.length === 0) {
    return '-';
  }
  return (
    <Space size={[4, 4]} wrap={true}>
      {items.map((item) => <Tag key={item}>{labelMap[item] || item}</Tag>)}
    </Space>
  );
};
const renderPolicyDiffTags = (diff = []) => {
  const items = Array.isArray(diff) ? diff.filter((item) => item && item.field) : [];
  if (!items.length) {
    return <Text type='secondary'>无字段变化</Text>;
  }
  return (
    <Space size={[4, 4]} wrap={true}>
      {items.map((item) => <Tag key={item.field}>{item.label || item.field}</Tag>)}
    </Space>
  );
};
const robotTags = (record) => Array.isArray(record?.robotProcessingTags) ? record.robotProcessingTags.filter(Boolean) : [];
const renderRobotTags = (record) => {
  const tags = robotTags(record);
  if (!record?.robotProcessed && tags.length === 0) {
    return <Tag>人工/未标记</Tag>;
  }
  return (
    <Space size={[4, 4]} wrap={true}>
      <Tag color='processing'>{record?.robotProcessor || 'cloudiac_robot'}</Tag>
      {tags.map((tag) => <Tag key={tag} color='blue'>{robotTagMap[tag] || tag}</Tag>)}
    </Space>
  );
};
const renderRobotSummary = (record) => {
  if (!record?.robotProcessed) {
    return <Tag>人工/未标记</Tag>;
  }
  const tags = robotTags(record).slice(0, 2);
  return (
    <Space size={4}>
      <Tag color='processing'>机器人</Tag>
      {tags.map((tag) => <Tag key={tag} color='blue'>{robotTagMap[tag] || tag}</Tag>)}
    </Space>
  );
};
const metadataText = (value) => JSON.stringify(value || {}, null, 2);
const callbackUrlText = (record) => {
  if (!record?.id) {
    return '保存连接器后生成回调地址';
  }
  return `${window.location.origin}/api/v1/cloud/itsm/callbacks/${record.id}/status`;
};
const gitOpsGateCallbackUrlText = (record) => {
  if (!record?.id) {
    return '保存连接器后生成回调地址';
  }
  return `${window.location.origin}/api/v1/cloud/itsm/callbacks/${record.id}/gitops-gate`;
};
const parseMetadata = (value) => {
  if (!value || String(value).trim() === '') {
    return {};
  }
  if (typeof value === 'object') {
    return value;
  }
  return JSON.parse(value);
};
const cleanObject = (value) => Object.keys(value || {}).reduce((acc, key) => {
  const item = value[key];
  if (item !== undefined && item !== null && String(item).trim() !== '') {
    acc[key] = item;
  }
  return acc;
}, {});
const selfServiceSubmitPayload = (values) => {
  const params = {
    ...parseMetadata(values.params)
  };
  if (values.requestType === 'gitops_iac_change') {
    Object.assign(params, cleanObject({
      gitOpsRepository: values.gitOpsRepository,
      gitOpsBranch: values.gitOpsBranch,
      gitOpsTargetBranch: values.gitOpsTargetBranch,
      gitOpsChangePath: values.gitOpsChangePath,
      gitOpsPullRequestUrl: values.gitOpsPullRequestUrl,
      gitOpsReviewStatus: values.gitOpsReviewStatus,
      gitOpsPipelineUrl: values.gitOpsPipelineUrl,
      gitOpsPipelineStatus: values.gitOpsPipelineStatus
    }));
  }
  const payload = { ...values };
  [
    'gitOpsRepository',
    'gitOpsBranch',
    'gitOpsTargetBranch',
    'gitOpsChangePath',
    'gitOpsPullRequestUrl',
    'gitOpsReviewStatus',
    'gitOpsPipelineUrl',
    'gitOpsPipelineStatus',
    'params'
  ].forEach((key) => {
    delete payload[key];
  });
  return {
    ...payload,
    params
  };
};

const JsonBlock = ({ value }) => (
  <pre className={styles.jsonBlock}>{jsonText(value)}</pre>
);

const GoalMetric = ({ icon, title, value, target, description, tone }) => (
  <div className={`${styles.goalItem} ${tone ? styles[tone] : ''}`}>
    <div className={styles.goalHeader}>
      {icon}
      <Text strong={true}>{title}</Text>
    </div>
    <div className={styles.goalValue}>{percentText(value)}</div>
    <Progress percent={percent(value)} showInfo={false} strokeColor={percent(value) >= Number(target || 0) ? '#2ca58d' : '#d46b08'}/>
    <Text type='secondary'>{description}</Text>
  </div>
);

const CountMetric = ({ title, value, description, tone }) => (
  <div className={`${styles.goalItem} ${tone ? styles[tone] : ''}`}>
    <div className={styles.goalHeader}>
      <Text strong={true}>{title}</Text>
    </div>
    <div className={styles.goalValue}>{value || 0}</div>
    <Text type='secondary'>{description}</Text>
  </div>
);

const TicketDetail = ({ detail = {}, loading, onUpdateStatus }) => {
  if (loading) {
    return <Empty description='加载中'/>;
  }
  if (!detail.id) {
    return <Empty description='请选择工单'/>;
  }
  return (
    <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
      <Descriptions size='small' bordered={true} column={2}>
        <Descriptions.Item label='工单 ID'>{detail.id || '-'}</Descriptions.Item>
        <Descriptions.Item label='状态'>{statusTag(ticketStatusMap, detail.status)}</Descriptions.Item>
        <Descriptions.Item label='标题'>{detail.title || '-'}</Descriptions.Item>
        <Descriptions.Item label='连接器'>{detail.connectorName || detail.connectorId || '-'}</Descriptions.Item>
        <Descriptions.Item label='机器人处理'>{detail.robotProcessed ? <Tag color='processing'>已标记</Tag> : <Tag>人工/未标记</Tag>}</Descriptions.Item>
        <Descriptions.Item label='处理器'>{detail.robotProcessor || '-'}</Descriptions.Item>
        <Descriptions.Item label='自动化模式'>{automationModeMap[detail.robotAutomationMode] || detail.robotAutomationMode || '-'}</Descriptions.Item>
        <Descriptions.Item label='处理通道'>{robotChannelMap[detail.robotTicketChannel] || detail.robotTicketChannel || '-'}</Descriptions.Item>
        <Descriptions.Item label='外部单号'>{detail.externalKey || detail.externalId || '-'}</Descriptions.Item>
        <Descriptions.Item label='外部地址'>{detail.externalUrl || '-'}</Descriptions.Item>
        <Descriptions.Item label='优先级'>{detail.priority || '-'}</Descriptions.Item>
        <Descriptions.Item label='风险'>{statusTag(riskMap, detail.riskLevel)}</Descriptions.Item>
        <Descriptions.Item label='操作任务'>{detail.operationName || detail.operationId || '-'}</Descriptions.Item>
        <Descriptions.Item label='项目'>{detail.projectName || detail.projectId || '-'}</Descriptions.Item>
        <Descriptions.Item label='环境'>{detail.envName || detail.envId || '-'}</Descriptions.Item>
        <Descriptions.Item label='创建人'>{detail.creatorName || detail.creatorId || '-'}</Descriptions.Item>
        <Descriptions.Item label='提交时间'>{renderTime(detail.submittedAt)}</Descriptions.Item>
        <Descriptions.Item label='同步时间'>{renderTime(detail.lastSyncedAt)}</Descriptions.Item>
        <Descriptions.Item label='关闭时间'>{renderTime(detail.closedAt)}</Descriptions.Item>
        <Descriptions.Item label='错误'>{detail.errorMessage || '-'}</Descriptions.Item>
        <Descriptions.Item label='处理标签' span={2}>{renderRobotTags(detail)}</Descriptions.Item>
        <Descriptions.Item label='描述' span={2}>{detail.description || '-'}</Descriptions.Item>
      </Descriptions>
      <Button icon={<SyncOutlined/>} onClick={() => onUpdateStatus(detail)}>更新状态</Button>
      <div>
        <Text strong={true}>请求载荷</Text>
        <JsonBlock value={detail.requestPayload}/>
      </div>
      <div>
        <Text strong={true}>响应载荷</Text>
        <JsonBlock value={detail.responsePayload}/>
      </div>
    </Space>
  );
};

const CloudItsmPage = ({ match }) => {
  const { orgId } = match.params || {};
  const [ configForm ] = Form.useForm();
  const [ statusForm ] = Form.useForm();
  const [ selfServiceForm ] = Form.useForm();
  const [ catalogPolicyForm ] = Form.useForm();
  const [ deadLetterCloseForm ] = Form.useForm();
  const [ deadLetterApprovalForm ] = Form.useForm();
  const [ configQuery, setConfigQuery ] = useState({
    currentPage: 1,
    pageSize: 10
  });
  const [ ticketQuery, setTicketQuery ] = useState({
    currentPage: 1,
    pageSize: 10
  });
  const [ retryQueueQuery, setRetryQueueQuery ] = useState({
    currentPage: 1,
    pageSize: 10
  });
  const [ configModal, setConfigModal ] = useState({
    visible: false,
    record: null
  });
  const [ statusModal, setStatusModal ] = useState({
    visible: false,
    record: null
  });
  const [ selfServiceModal, setSelfServiceModal ] = useState({
    visible: false
  });
  const [ catalogPolicyModal, setCatalogPolicyModal ] = useState({
    visible: false,
    record: null
  });
  const [ deadLetterCloseModal, setDeadLetterCloseModal ] = useState({
    visible: false
  });
  const [ deadLetterApprovalModal, setDeadLetterApprovalModal ] = useState({
    visible: false,
    action: 'replay'
  });
  const [ selectedDeadLetterIds, setSelectedDeadLetterIds ] = useState([]);
  const [ catalogPolicyHistory, setCatalogPolicyHistory ] = useState([]);
  const [ drawer, setDrawer ] = useState({
    visible: false,
    id: ''
  });

  const {
    data: overviewData = {},
    run: fetchOverview
  } = useRequest(
    () => requestWrapper(cloudItsmAPI.overview.bind(null, { orgId })),
    {
      refreshDeps: [ orgId ]
    }
  );

  const {
    loading: configsLoading,
    data: configsData = {},
    run: fetchConfigs
  } = useRequest(
    () => requestWrapper(cloudItsmAPI.configs.bind(null, { orgId, ...configQuery })),
    {
      refreshDeps: [ orgId, configQuery ]
    }
  );

  const {
    loading: ticketsLoading,
    data: ticketsData = {},
    run: fetchTickets
  } = useRequest(
    () => requestWrapper(cloudItsmAPI.tickets.bind(null, { orgId, ...ticketQuery })),
    {
      refreshDeps: [ orgId, ticketQuery ]
    }
  );

  const {
    data: retryQueueSummary = {},
    run: fetchRetryQueueSummary
  } = useRequest(
    () => requestWrapper(cloudItsmAPI.retryQueueSummary.bind(null, { orgId })),
    {
      refreshDeps: [ orgId ]
    }
  );

  const {
    loading: retryQueueReportLoading,
    data: retryQueueReport = {},
    run: fetchRetryQueueReport
  } = useRequest(
    () => requestWrapper(cloudItsmAPI.retryQueueReport.bind(null, { orgId })),
    {
      refreshDeps: [ orgId ]
    }
  );

  const {
    loading: retryQueueLoading,
    data: retryQueueData = {},
    run: fetchRetryQueue
  } = useRequest(
    () => requestWrapper(cloudItsmAPI.retryQueue.bind(null, { orgId, ...retryQueueQuery })),
    {
      refreshDeps: [ orgId, retryQueueQuery ]
    }
  );

  const {
    loading: savingConfig,
    run: saveConfig
  } = useRequest(
    ({ record, values }) => {
      const payload = { ...values };
      if (record) {
        if (!String(payload.token || '').trim()) {
          delete payload.token;
        }
        if (!String(payload.password || '').trim()) {
          delete payload.password;
        }
      }
      const api = record
        ? cloudItsmAPI.updateConfig.bind(null, { orgId, id: record.id, data: payload })
        : cloudItsmAPI.createConfig.bind(null, { orgId, data: payload });
      return requestWrapper(api, { autoSuccess: true, successMessage: '连接器已保存' });
    },
    {
      manual: true,
      onSuccess: () => {
        setConfigModal({ visible: false, record: null });
        configForm.resetFields();
        fetchConfigs();
        fetchRetryQueueReport();
      }
    }
  );

  const {
    loading: deletingConfig,
    run: deleteConfig
  } = useRequest(
    ({ id }) => requestWrapper(cloudItsmAPI.deleteConfig.bind(null, { orgId, id }), { autoSuccess: true, successMessage: '连接器已删除' }),
    {
      manual: true,
      onSuccess: () => {
        fetchConfigs();
        fetchRetryQueueReport();
      }
    }
  );

  const {
    loading: updatingStatus,
    run: updateTicketStatus
  } = useRequest(
    ({ id, values }) => requestWrapper(cloudItsmAPI.updateTicketStatus.bind(null, { orgId, id, data: values }), {
      autoSuccess: true,
      successMessage: '工单状态已更新'
    }),
    {
      manual: true,
      onSuccess: () => {
        setStatusModal({ visible: false, record: null });
        statusForm.resetFields();
        fetchTickets();
      }
    }
  );

  const {
    loading: syncingStatuses,
    run: syncDueTicketStatuses
  } = useRequest(
    () => requestWrapper(cloudItsmAPI.syncDueTicketStatuses.bind(null, { orgId, data: { force: true } }), {
      autoSuccess: true,
      successMessage: '外部状态同步已触发'
    }),
    {
      manual: true,
      onSuccess: () => {
        fetchTickets();
        fetchOverview();
      }
    }
  );

  const {
    loading: retryingFailedSubmissions,
    run: retryFailedTicketSubmissions
  } = useRequest(
    () => requestWrapper(cloudItsmAPI.retryFailedTicketSubmissions.bind(null, { orgId, data: { force: true } }), {
      autoSuccess: true,
      successMessage: '失败提交补偿已触发'
    }),
    {
      manual: true,
      onSuccess: () => {
        fetchTickets();
        fetchOverview();
        fetchRetryQueue();
        fetchRetryQueueSummary();
        fetchRetryQueueReport();
      }
    }
  );

  const {
    loading: replayingTicketSubmission,
    run: replayTicketSubmission
  } = useRequest(
    ({ id }) => requestWrapper(cloudItsmAPI.replayTicketSubmission.bind(null, { orgId, id, data: { force: true } }), {
      autoSuccess: true,
      successMessage: '失败提交已重放'
    }),
    {
      manual: true,
      onSuccess: () => {
        fetchTickets();
        fetchOverview();
        fetchRetryQueue();
        fetchRetryQueueSummary();
        fetchRetryQueueReport();
      }
    }
  );

  const {
    loading: batchDeadLetterActionLoading,
    run: batchDeadLetterAction
  } = useRequest(
    ({ action, reason }) => requestWrapper(cloudItsmAPI.batchDeadLetterAction.bind(null, {
      orgId,
      data: {
        ids: selectedDeadLetterIds,
        action,
        force: true,
        reason
      }
    }), {
      autoSuccess: true,
      successMessage: action === 'close' ? '死信已人工关闭' : '死信重放已触发'
    }),
    {
      manual: true,
      onSuccess: () => {
        setSelectedDeadLetterIds([]);
        setDeadLetterCloseModal({ visible: false });
        deadLetterCloseForm.resetFields();
        fetchTickets();
        fetchOverview();
        fetchRetryQueue();
        fetchRetryQueueSummary();
        fetchRetryQueueReport();
      }
    }
  );

  const {
    loading: creatingDeadLetterApproval,
    run: createDeadLetterApproval
  } = useRequest(
    ({ action, values }) => {
      const evidenceItems = normalizeDeadLetterEvidenceItems(values.evidenceItems);
      return requestWrapper(cloudItsmAPI.createDeadLetterApproval.bind(null, {
        orgId,
        data: {
          ids: selectedDeadLetterIds,
          action,
          force: true,
          reason: values.reason,
          evidenceUrl: values.evidenceUrl,
          evidenceItems
        }
      }), {
        autoSuccess: true,
        successMessage: '死信处置审批已提交'
      });
    },
    {
      manual: true,
      onSuccess: () => {
        setSelectedDeadLetterIds([]);
        setDeadLetterApprovalModal({ visible: false, action: 'replay' });
        deadLetterApprovalForm.resetFields();
        fetchRetryQueue();
        fetchRetryQueueSummary();
        fetchRetryQueueReport();
      }
    }
  );

  const {
    loading: creatingSelfService,
    run: createSelfServiceTicket
  } = useRequest(
    ({ values }) => requestWrapper(cloudItsmAPI.createSelfServiceTicket.bind(null, { orgId, data: values }), {
      autoSuccess: true,
      successMessage: '自助申请已提交'
    }),
    {
      manual: true,
      onSuccess: () => {
        setSelfServiceModal({ visible: false });
        selfServiceForm.resetFields();
        fetchTickets();
        fetchOverview();
        fetchRetryQueue();
        fetchRetryQueueSummary();
        fetchRetryQueueReport();
      }
    }
  );

  const {
    loading: savingCatalogPolicy,
    run: saveCatalogPolicy
  } = useRequest(
    ({ record, values }) => requestWrapper(cloudItsmAPI.updateCatalogPolicy.bind(null, {
      orgId,
      key: record.key,
      data: values
    }), {
      autoSuccess: true,
      successMessage: values.reset ? '策略已恢复默认' : '自助目录策略已保存'
    }),
    {
      manual: true,
      onSuccess: () => {
        setCatalogPolicyModal({ visible: false, record: null });
        setCatalogPolicyHistory([]);
        catalogPolicyForm.resetFields();
        fetchOverview();
      }
    }
  );

  const {
    loading: catalogPolicyHistoryLoading,
    run: fetchCatalogPolicyHistory
  } = useRequest(
    ({ key }) => requestWrapper(cloudItsmAPI.catalogPolicyHistory.bind(null, {
      orgId,
      key,
      limit: 20
    })),
    {
      manual: true,
      onSuccess: (data) => setCatalogPolicyHistory(Array.isArray(data) ? data : [])
    }
  );

  const metrics = overviewData.metrics || {};
  const catalog = overviewData.catalog || [];
  const slaTrend = overviewData.slaTrend || [];
  const projectTrends = overviewData.projectTrends || [];
  const requestTypeTrends = overviewData.requestTypeTrends || [];
  const teamTrends = overviewData.teamTrends || [];
  const selfServiceCatalog = catalog.filter((item) => item.operationType === 'self_service' && item.available);
  const configs = configsData.list || [];
  const enabledConfigs = configs.filter((item) => item.status === 'enable');
  const defaultEnabledConnectorId = enabledConfigs[0]?.id;
  const tickets = ticketsData.list || [];
  const retryQueueItems = retryQueueData.list || [];
  const retryConnectorBreakdown = retryQueueReport.connectorBreakdown || [];
  const retryReasonBreakdown = retryQueueReport.reasonBreakdown || [];
  const retryAgeBuckets = retryQueueReport.ageBuckets || [];
  const retryProjectBreakdown = retryQueueReport.projectBreakdown || [];
  const retryRequestTypeBreakdown = retryQueueReport.requestTypeBreakdown || [];
  const retryTeamBreakdown = retryQueueReport.teamBreakdown || [];
  const retryErrorCodeBreakdown = retryQueueReport.errorCodeBreakdown || [];
  const retryExternalResponseCodeBreakdown = retryQueueReport.externalResponseCodeBreakdown || [];
  const retryRecentDeadLetters = retryQueueReport.recentDeadLetters || [];
  const deadLetterRowSelection = {
    selectedRowKeys: selectedDeadLetterIds,
    onChange: setSelectedDeadLetterIds,
    getCheckboxProps: (record) => ({
      disabled: record.queueStatus !== 'dead_letter'
    })
  };
  const detail = useMemo(() => (
    tickets.find((item) => item.id === drawer.id) ||
    retryQueueItems.find((item) => item.id === drawer.id) ||
    retryRecentDeadLetters.find((item) => item.id === drawer.id) ||
    {}
  ), [ tickets, retryQueueItems, retryRecentDeadLetters, drawer.id ]);

  useEffect(() => {
    if (!selfServiceModal.visible || !defaultEnabledConnectorId) {
      return;
    }
    if (!selfServiceForm.getFieldValue('connectorId')) {
      selfServiceForm.setFieldsValue({ connectorId: defaultEnabledConnectorId });
    }
  }, [selfServiceModal.visible, defaultEnabledConnectorId, selfServiceForm]);

  const openConfigModal = (record) => {
    setConfigModal({ visible: true, record });
    configForm.setFieldsValue(record ? {
      ...record,
      token: '',
      password: '',
      metadata: metadataText(record.metadata)
    } : {
      provider: 'generic',
      authType: 'none',
      status: 'enable',
      timeoutSeconds: 10,
      metadata: metadataText({})
    });
  };

  const openStatusModal = (record) => {
    setStatusModal({ visible: true, record });
    statusForm.setFieldsValue({
      status: record.status,
      externalId: record.externalId,
      externalKey: record.externalKey,
      externalUrl: record.externalUrl,
      comment: '',
      payload: {}
    });
  };

  const openSelfServiceModal = () => {
    fetchConfigs();
    setSelfServiceModal({ visible: true });
    selfServiceForm.setFieldsValue({
      requestType: selfServiceCatalog[0]?.key,
      connectorId: enabledConfigs[0]?.id,
      priority: 'medium',
      gitOpsReviewStatus: 'pending',
      gitOpsPipelineStatus: 'pending',
      params: metadataText({})
    });
  };

  const openCatalogPolicyModal = (record) => {
    setCatalogPolicyModal({ visible: true, record });
    setCatalogPolicyHistory([]);
    fetchCatalogPolicyHistory({ key: record.key });
    catalogPolicyForm.setFieldsValue({
      enabled: record.policyConfigured ? record.enabled : true,
      policyName: record.policyName,
      policyDescription: record.policyDescription,
      requiredRoles: record.requiredRoles || [],
      allowedScopes: record.allowedScopes || [],
      slaMinutes: record.slaMinutes || 240,
      reset: false
    });
  };

  const configColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      width: 180,
      render: (text) => text || '-'
    },
    {
      title: '类型',
      dataIndex: 'provider',
      width: 120,
      render: (text) => providerMap[text] || text || '-'
    },
    {
      title: '地址',
      dataIndex: 'baseUrl',
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      title: '认证',
      dataIndex: 'authType',
      width: 120,
      render: (text) => authMap[text] || text || '-'
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (text) => statusTag(configStatusMap, text)
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      width: 180,
      render: renderTime
    },
    {
      title: '操作',
      key: 'op',
      width: 130,
      render: (_, record) => (
        <Space size={8}>
          <Button size='small' icon={<EditOutlined/>} onClick={() => openConfigModal(record)}/>
          <Popconfirm title='确认删除该连接器？' onConfirm={() => deleteConfig({ id: record.id })}>
            <Button danger={true} size='small' icon={<DeleteOutlined/>} loading={deletingConfig}/>
          </Popconfirm>
        </Space>
      )
    }
  ];

  const ticketColumns = [
    {
      title: '标题',
      dataIndex: 'title',
      width: 260,
      ellipsis: true,
      render: (text, record) => <a onClick={() => setDrawer({ visible: true, id: record.id })}>{text || record.id}</a>
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 110,
      render: (text) => statusTag(ticketStatusMap, text)
    },
    {
      title: '机器人处理',
      dataIndex: 'robotProcessed',
      width: 180,
      render: (_, record) => renderRobotSummary(record)
    },
    {
      title: '连接器',
      dataIndex: 'connectorName',
      width: 150,
      render: (text, record) => text || record.connectorId || '-'
    },
    {
      title: '外部单号',
      dataIndex: 'externalKey',
      width: 160,
      render: (text, record) => text || record.externalId || '-'
    },
    {
      title: '风险',
      dataIndex: 'riskLevel',
      width: 90,
      render: (text) => statusTag(riskMap, text)
    },
    {
      title: '操作任务',
      dataIndex: 'operationName',
      width: 190,
      ellipsis: true,
      render: (text, record) => text || record.operationId || '-'
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 180,
      render: renderTime
    },
    {
      title: '操作',
      key: 'op',
      width: 110,
      render: (_, record) => (
        <Button size='small' icon={<SyncOutlined/>} onClick={() => openStatusModal(record)}>状态</Button>
      )
    }
  ];

  const retryQueueColumns = [
    {
      title: '标题',
      dataIndex: 'title',
      width: 260,
      ellipsis: true,
      render: (text, record) => <a onClick={() => setDrawer({ visible: true, id: record.id })}>{text || record.id}</a>
    },
    {
      title: '队列状态',
      dataIndex: 'queueStatus',
      width: 120,
      render: (text) => statusTag(retryQueueStatusMap, text)
    },
    {
      title: '原因',
      dataIndex: 'retryReason',
      width: 180,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      title: '尝试',
      dataIndex: 'attempt',
      width: 90,
      render: (value, record) => `${value || 0}/${record.retryState?.maxAttempts || '-'}`
    },
    {
      title: '下次尝试',
      dataIndex: 'nextAttempt',
      width: 100,
      render: (value) => value || '-'
    },
    {
      title: '下次重试',
      dataIndex: 'nextRetryAt',
      width: 180,
      render: renderTime
    },
    {
      title: '连接器',
      dataIndex: 'connectorName',
      width: 150,
      render: (text, record) => text || record.connectorId || '-'
    },
    {
      title: '最近同步',
      dataIndex: 'lastSyncedAt',
      width: 180,
      render: renderTime
    },
    {
      title: '操作',
      key: 'op',
      width: 120,
      render: (_, record) => (
        <Popconfirm title='确认重放该失败提交？' onConfirm={() => replayTicketSubmission({ id: record.id })}>
          <Button size='small' icon={<SyncOutlined/>} loading={replayingTicketSubmission}>重放</Button>
        </Popconfirm>
      )
    }
  ];

  const retryQueueBreakdownColumns = [
    {
      title: '维度',
      dataIndex: 'dimensionName',
      width: 170,
      ellipsis: true,
      render: (text, record) => (
        <Space size={4}>
          <Text>{text || record.dimensionId || '-'}</Text>
          {record.provider ? <Tag>{providerMap[record.provider] || record.provider}</Tag> : null}
        </Space>
      )
    },
    {
      title: '失败',
      dataIndex: 'failedTotal',
      width: 80,
      render: (value) => value || 0
    },
    {
      title: '可重试',
      dataIndex: 'retryable',
      width: 90,
      render: (value, record) => (
        <Space size={4}>
          <Text>{value || 0}</Text>
          <Tag color='processing'>到期 {record.due || 0}</Tag>
          <Tag color='warning'>等待 {record.future || 0}</Tag>
        </Space>
      )
    },
    {
      title: '死信',
      dataIndex: 'deadLetter',
      width: 80,
      render: (value) => value || 0
    },
    {
      title: '跳过',
      dataIndex: 'skipped',
      width: 80,
      render: (value) => value || 0
    },
    {
      title: '时间',
      key: 'retryTime',
      width: 190,
      render: (_, record) => (
        <Space direction='vertical' size={0}>
          <Text>{record.oldestDueAt ? `最早到期 ${renderTime(record.oldestDueAt)}` : '-'}</Text>
          {record.nextRetryAt ? <Text type='secondary'>下次 {renderTime(record.nextRetryAt)}</Text> : null}
        </Space>
      )
    }
  ];

  const retryDeadLetterColumns = [
    {
      title: '标题',
      dataIndex: 'title',
      width: 220,
      ellipsis: true,
      render: (text, record) => <a onClick={() => setDrawer({ visible: true, id: record.id })}>{text || record.id}</a>
    },
    {
      title: '连接器',
      dataIndex: 'connectorName',
      width: 140,
      render: (text, record) => text || record.connectorId || '-'
    },
    {
      title: '原因',
      dataIndex: 'retryReason',
      width: 150,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      title: '尝试',
      dataIndex: 'attempt',
      width: 90,
      render: (value, record) => `${value || 0}/${record.retryState?.maxAttempts || '-'}`
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      width: 170,
      render: renderTime
    },
    {
      title: '操作',
      key: 'op',
      width: 120,
      render: (_, record) => (
        <Popconfirm title='确认重放该死信提交？' onConfirm={() => replayTicketSubmission({ id: record.id })}>
          <Button size='small' icon={<SyncOutlined/>} loading={replayingTicketSubmission}>重放</Button>
        </Popconfirm>
      )
    }
  ];

  const catalogColumns = [
    {
      title: '服务项',
      dataIndex: 'name',
      width: 180,
      render: (text, record) => (
        <Space size={6}>
          <Text>{text || record.key}</Text>
          {record.requiresApproval && <Tag color='warning'>需审批</Tag>}
        </Space>
      )
    },
    {
      title: '类别',
      dataIndex: 'category',
      width: 110,
      render: (text) => catalogCategoryMap[text] || text || '-'
    },
    {
      title: '处理方式',
      dataIndex: 'automationMode',
      width: 140,
      render: (text) => automationModeMap[text] || text || '-'
    },
    {
      title: '来源',
      dataIndex: 'source',
      width: 130,
      render: (text) => text === 'cloud_operation' ? '云操作目录' : 'ITSM 自助申请'
    },
    {
      title: '权限策略',
      dataIndex: 'policyName',
      width: 220,
      render: (text, record) => (
        <Space direction='vertical' size={2}>
          <Space size={4}>
            <Text>{text || record.policyKey || '-'}</Text>
            {record.policyConfigured && <Tag color='blue'>已配置</Tag>}
            {record.policyVersion ? <Tag color='geekblue'>v{record.policyVersion}</Tag> : null}
          </Space>
          <Text type='secondary'>{renderTagList(record.requiredRoles, policyRoleMap)}</Text>
        </Space>
      )
    },
    {
      title: '范围',
      dataIndex: 'allowedScopes',
      width: 170,
      render: (value) => renderTagList(value, policyScopeMap)
    },
    {
      title: 'SLA',
      dataIndex: 'slaMinutes',
      width: 120,
      render: (value, record) => record.slaDescription || minutesText(value)
    },
    {
      title: '风险',
      dataIndex: 'riskLevel',
      width: 90,
      render: (text) => statusTag(riskMap, text)
    },
    {
      title: '状态',
      dataIndex: 'available',
      width: 100,
      render: (value, record) => value ? <Tag color='success'>已上架</Tag> : <Tag>{record.disabledReason || '未开放'}</Tag>
    },
    {
      title: '说明',
      dataIndex: 'description',
      width: 260,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      title: '操作',
      key: 'op',
      width: 110,
      render: (_, record) => (
        <Button size='small' icon={<EditOutlined/>} onClick={() => openCatalogPolicyModal(record)}>策略</Button>
      )
    }
  ];

  const catalogPolicyHistoryColumns = [
    {
      title: '版本',
      dataIndex: 'version',
      width: 80,
      render: (value) => value ? `v${value}` : '-'
    },
    {
      title: '动作',
      dataIndex: 'action',
      width: 100,
      render: (value) => catalogPolicyActionMap[value] || value || '-'
    },
    {
      title: '变更人',
      dataIndex: 'updatedBy',
      width: 120,
      render: (value) => value || '-'
    },
    {
      title: '变更时间',
      dataIndex: 'updatedAt',
      width: 170,
      render: renderTime
    },
    {
      title: '差异',
      dataIndex: 'diff',
      render: renderPolicyDiffTags
    }
  ];

  const slaTrendColumns = [
    {
      title: '日期',
      dataIndex: 'date',
      width: 130
    },
    {
      title: 'SLA 工单',
      dataIndex: 'ticketTotal',
      width: 120,
      render: (value) => value || 0
    },
    {
      title: '达成',
      dataIndex: 'slaMetTotal',
      width: 100,
      render: (value) => value || 0
    },
    {
      title: '超时',
      dataIndex: 'slaBreached',
      width: 100,
      render: (value) => value || 0
    },
    {
      title: '达成率',
      dataIndex: 'slaMetRate',
      width: 120,
      render: (value, record) => (
        <Space size={6}>
          <Text>{percentText(value)}</Text>
          <Tag color={percent(value) >= Number(record.target || metrics.selfServiceSlaTarget || 0) ? 'success' : 'warning'}>
            目标 {percentText(record.target || metrics.selfServiceSlaTarget)}
          </Tag>
        </Space>
      )
    }
  ];

  const dimensionTrendColumns = [
    {
      title: '维度',
      dataIndex: 'dimensionName',
      width: 160,
      ellipsis: true,
      render: (text, record) => text || record.dimensionId || '-'
    },
    {
      title: '工单',
      dataIndex: 'ticketTotal',
      width: 90,
      render: (value) => value || 0
    },
    {
      title: '自助占比',
      dataIndex: 'selfServiceCoverageRate',
      width: 130,
      render: (value, record) => (
        <Space size={6}>
          <Text>{percentText(value)}</Text>
          <Tag color={percent(value) >= Number(record.selfServiceCoverageTarget || metrics.selfServiceCoverageTarget || 0) ? 'success' : 'warning'}>
            {record.selfServiceTicketTotal || 0}/{record.ticketTotal || 0}
          </Tag>
        </Space>
      )
    },
    {
      title: '自动化',
      dataIndex: 'ticketAutomationRate',
      width: 130,
      render: (value, record) => (
        <Space size={6}>
          <Text>{percentText(value)}</Text>
          <Tag color={percent(value) >= Number(record.ticketAutomationTarget || metrics.ticketAutomationTarget || 0) ? 'success' : 'warning'}>
            {record.automatedTicketTotal || 0}/{record.ticketTotal || 0}
          </Tag>
        </Space>
      )
    },
    {
      title: 'SLA 达成',
      dataIndex: 'selfServiceSlaMetRate',
      width: 140,
      render: (value, record) => (
        <Space size={6}>
          <Text>{percentText(value)}</Text>
          <Tag color={percent(value) >= Number(record.selfServiceSlaTarget || metrics.selfServiceSlaTarget || 0) ? 'success' : 'warning'}>
            {record.selfServiceSlaMetTotal || 0}/{record.selfServiceSlaTicketTotal || 0}
          </Tag>
        </Space>
      )
    }
  ];

  return (
    <Layout
      contentStyle={{
        padding: 0
      }}
    >
      <PageHeader title='ITSM 工单'/>
      <div className={styles.goalGrid}>
        <GoalMetric
          icon={<ToolOutlined/>}
          title='自助运维覆盖'
          value={metrics.selfServiceCoverageRate}
          target={metrics.selfServiceCoverageTarget}
          description={`目标 ${percentText(metrics.selfServiceCoverageTarget)}，已上架 ${metrics.selfServiceAvailableTotal || 0}/${metrics.selfServiceCatalogTotal || 0} 项`}
          tone={percent(metrics.selfServiceCoverageRate) >= Number(metrics.selfServiceCoverageTarget || 0) ? 'passTone' : 'warnTone'}
        />
        <GoalMetric
          icon={<SyncOutlined/>}
          title='工单自动化处理'
          value={metrics.ticketAutomationRate}
          target={metrics.ticketAutomationTarget}
          description={`目标 ${percentText(metrics.ticketAutomationTarget)}，机器人 ${metrics.robotProcessedTicketTotal || 0}/${metrics.ticketTotal || 0} 单，自动 ${metrics.automatedTicketTotal || 0} 单`}
          tone={percent(metrics.ticketAutomationRate) >= Number(metrics.ticketAutomationTarget || 0) ? 'passTone' : 'warnTone'}
        />
        <GoalMetric
          icon={<SafetyCertificateOutlined/>}
          title='IaC 代码化覆盖'
          value={metrics.iacCoverageRate}
          target={metrics.iacCoverageTarget}
          description={`目标 ${percentText(metrics.iacCoverageTarget)}，IaC ${metrics.iacManagedAssets || 0}/${metrics.assetTotal || 0} 资产`}
          tone={percent(metrics.iacCoverageRate) >= Number(metrics.iacCoverageTarget || 0) ? 'passTone' : 'warnTone'}
        />
        <GoalMetric
          icon={<ClockCircleOutlined/>}
          title='自助 SLA 达成'
          value={metrics.selfServiceSlaMetRate}
          target={metrics.selfServiceSlaTarget}
          description={`目标 ${percentText(metrics.selfServiceSlaTarget)}，达成 ${metrics.selfServiceSlaMetTotal || 0}/${metrics.selfServiceSlaTicketTotal || 0} 单，超时 ${metrics.selfServiceSlaBreached || 0} 单`}
          tone={percent(metrics.selfServiceSlaMetRate) >= Number(metrics.selfServiceSlaTarget || 0) ? 'passTone' : 'warnTone'}
        />
      </div>
      <div className={styles.guardGrid}>
        <Alert
          showIcon={true}
          icon={<CheckCircleOutlined/>}
          type={metrics.gitOpsGuardStatus === 'pass' ? 'success' : 'warning'}
          message='GitOps/IaC 变更门禁'
          description={metrics.gitOpsGuardMessage || '基础设施变更应通过 PR review 和自动化流水线执行'}
        />
        <Alert
          showIcon={true}
          icon={<CheckCircleOutlined/>}
          type={metrics.driftGuardStatus === 'pass' ? 'success' : 'warning'}
          message='环境一致性保障'
          description={metrics.driftGuardMessage || 'dev/staging/prod 应开启漂移检测和自动修复'}
        />
      </div>
      <Tabs defaultActiveKey='tickets'>
        <Tabs.TabPane tab='工单' key='tickets'>
          <div className={styles.toolbar}>
            <Space className={styles.filterBar} size={8} wrap={true}>
              <InputSearch
                className={styles.keywordSearch}
                allowClear={true}
                placeholder='搜索标题 / 单号'
                onSearch={(q) => setTicketQuery({ ...ticketQuery, q, currentPage: 1 })}
              />
              <Select
                allowClear={true}
                placeholder='状态'
                style={{ width: 140 }}
                onChange={(status) => setTicketQuery({ ...ticketQuery, status, currentPage: 1 })}
              >
                {Object.keys(ticketStatusMap).map((key) => (
                  <Option value={key} key={key}>{ticketStatusMap[key].label}</Option>
                ))}
              </Select>
            </Space>
            <Space>
              <Button type='primary' icon={<PlusOutlined/>} onClick={openSelfServiceModal}>发起自助申请</Button>
              <Button icon={<SyncOutlined/>} loading={syncingStatuses} onClick={syncDueTicketStatuses}>同步外部状态</Button>
              <Button icon={<SyncOutlined/>} loading={retryingFailedSubmissions} onClick={retryFailedTicketSubmissions}>重试失败提交</Button>
              <Button icon={<ReloadOutlined/>} onClick={() => { fetchTickets(); fetchOverview(); }}>刷新</Button>
            </Space>
          </div>
          <Table
            rowKey='id'
            loading={ticketsLoading}
            columns={ticketColumns}
            dataSource={tickets}
            pagination={{
              current: ticketQuery.currentPage,
              pageSize: ticketQuery.pageSize,
              total: ticketsData.total || 0,
              showSizeChanger: true,
              onChange: (currentPage, pageSize) => setTicketQuery({ ...ticketQuery, currentPage, pageSize })
            }}
          />
        </Tabs.TabPane>
        <Tabs.TabPane tab='失败补偿队列' key='retryQueue'>
          <div className={styles.goalGrid}>
            <CountMetric
              title='待重试'
              value={retryQueueSummary.due}
              description={`失败总数 ${retryQueueSummary.failedTotal || 0}，可重试 ${retryQueueSummary.retryable || 0}`}
              tone={Number(retryQueueSummary.due || 0) > 0 ? 'warnTone' : 'passTone'}
            />
            <CountMetric
              title='等待退避'
              value={retryQueueSummary.future}
              description={retryQueueSummary.nextRetryAt ? `下一次 ${renderTime(retryQueueSummary.nextRetryAt)}` : '暂无等待中的补偿任务'}
              tone={Number(retryQueueSummary.future || 0) > 0 ? 'warnTone' : 'passTone'}
            />
            <CountMetric
              title='死信'
              value={retryQueueSummary.deadLetter}
              description='达到最大次数后需要人工重放或确认'
              tone={Number(retryQueueSummary.deadLetter || 0) > 0 ? 'warnTone' : 'passTone'}
            />
            <CountMetric
              title='跳过'
              value={retryQueueSummary.skipped}
              description='通常因连接器、载荷或外部单号保护被跳过'
              tone={Number(retryQueueSummary.skipped || 0) > 0 ? 'warnTone' : 'passTone'}
            />
          </div>
          <div className={styles.retryReportGrid}>
            <div className={styles.retryReportPanel}>
              <div className={styles.retryReportTitle}>按连接器</div>
              <Table
                size='small'
                rowKey={(record) => `${record.dimensionType}-${record.dimensionId}`}
                columns={retryQueueBreakdownColumns}
                dataSource={retryConnectorBreakdown}
                loading={retryQueueReportLoading}
                pagination={false}
                scroll={{ x: 'max-content' }}
              />
            </div>
            <div className={styles.retryReportPanel}>
              <div className={styles.retryReportTitle}>按失败原因</div>
              <Table
                size='small'
                rowKey={(record) => `${record.dimensionType}-${record.dimensionId}`}
                columns={retryQueueBreakdownColumns}
                dataSource={retryReasonBreakdown}
                loading={retryQueueReportLoading}
                pagination={false}
                scroll={{ x: 'max-content' }}
              />
            </div>
            <div className={styles.retryReportPanel}>
              <div className={styles.retryReportTitle}>按失败年龄</div>
              <Table
                size='small'
                rowKey={(record) => `${record.dimensionType}-${record.dimensionId}`}
                columns={retryQueueBreakdownColumns}
                dataSource={retryAgeBuckets}
                loading={retryQueueReportLoading}
                pagination={false}
                scroll={{ x: 'max-content' }}
              />
            </div>
            <div className={styles.retryReportPanel}>
              <div className={styles.retryReportTitle}>按项目</div>
              <Table
                size='small'
                rowKey={(record) => `${record.dimensionType}-${record.dimensionId}`}
                columns={retryQueueBreakdownColumns}
                dataSource={retryProjectBreakdown}
                loading={retryQueueReportLoading}
                pagination={false}
                scroll={{ x: 'max-content' }}
              />
            </div>
            <div className={styles.retryReportPanel}>
              <div className={styles.retryReportTitle}>按团队</div>
              <Table
                size='small'
                rowKey={(record) => `${record.dimensionType}-${record.dimensionId}`}
                columns={retryQueueBreakdownColumns}
                dataSource={retryTeamBreakdown}
                loading={retryQueueReportLoading}
                pagination={false}
                scroll={{ x: 'max-content' }}
              />
            </div>
            <div className={styles.retryReportPanel}>
              <div className={styles.retryReportTitle}>按申请类型</div>
              <Table
                size='small'
                rowKey={(record) => `${record.dimensionType}-${record.dimensionId}`}
                columns={retryQueueBreakdownColumns}
                dataSource={retryRequestTypeBreakdown}
                loading={retryQueueReportLoading}
                pagination={false}
                scroll={{ x: 'max-content' }}
              />
            </div>
            <div className={styles.retryReportPanel}>
              <div className={styles.retryReportTitle}>按错误码</div>
              <Table
                size='small'
                rowKey={(record) => `${record.dimensionType}-${record.dimensionId}`}
                columns={retryQueueBreakdownColumns}
                dataSource={retryErrorCodeBreakdown}
                loading={retryQueueReportLoading}
                pagination={false}
                scroll={{ x: 'max-content' }}
              />
            </div>
            <div className={styles.retryReportPanel}>
              <div className={styles.retryReportTitle}>按外部响应码</div>
              <Table
                size='small'
                rowKey={(record) => `${record.dimensionType}-${record.dimensionId}`}
                columns={retryQueueBreakdownColumns}
                dataSource={retryExternalResponseCodeBreakdown}
                loading={retryQueueReportLoading}
                pagination={false}
                scroll={{ x: 'max-content' }}
              />
            </div>
          </div>
          <div className={styles.retryReportPanel}>
            <div className={styles.retryReportTitleRow}>
              <div className={styles.retryReportTitle}>最近死信</div>
              <Space size={8} wrap={true}>
                <Button
                  size='small'
                  icon={<SyncOutlined/>}
                  disabled={!selectedDeadLetterIds.length}
                  loading={batchDeadLetterActionLoading}
                  onClick={() => batchDeadLetterAction({ action: 'replay' })}
                >
                  批量重放
                </Button>
                <Button
                  size='small'
                  disabled={!selectedDeadLetterIds.length}
                  loading={creatingDeadLetterApproval}
                  onClick={() => {
                    deadLetterApprovalForm.resetFields();
                    setDeadLetterApprovalModal({ visible: true, action: 'replay' });
                  }}
                >
                  重放审批
                </Button>
                <Button
                  size='small'
                  disabled={!selectedDeadLetterIds.length}
                  loading={batchDeadLetterActionLoading}
                  onClick={() => {
                    deadLetterCloseForm.resetFields();
                    setDeadLetterCloseModal({ visible: true });
                  }}
                >
                  人工关闭
                </Button>
                <Button
                  size='small'
                  disabled={!selectedDeadLetterIds.length}
                  loading={creatingDeadLetterApproval}
                  onClick={() => {
                    deadLetterApprovalForm.resetFields();
                    setDeadLetterApprovalModal({ visible: true, action: 'close' });
                  }}
                >
                  关闭审批
                </Button>
              </Space>
            </div>
            <Table
              size='small'
              rowKey='id'
              columns={retryDeadLetterColumns}
              dataSource={retryRecentDeadLetters}
              loading={retryQueueReportLoading}
              rowSelection={deadLetterRowSelection}
              pagination={false}
              scroll={{ x: 'max-content' }}
              locale={{
                emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无死信'/>
              }}
            />
          </div>
          <div className={styles.toolbar}>
            <Space className={styles.filterBar} size={8} wrap={true}>
              <InputSearch
                className={styles.keywordSearch}
                allowClear={true}
                placeholder='搜索标题 / 单号 / 错误'
                onSearch={(q) => setRetryQueueQuery({ ...retryQueueQuery, q, currentPage: 1 })}
              />
              <Select
                allowClear={true}
                placeholder='队列状态'
                style={{ width: 150 }}
                onChange={(queueStatus) => setRetryQueueQuery({ ...retryQueueQuery, queueStatus, currentPage: 1 })}
              >
                {Object.keys(retryQueueStatusMap).map((key) => (
                  <Option value={key} key={key}>{retryQueueStatusMap[key].label}</Option>
                ))}
              </Select>
            </Space>
            <Space wrap={true}>
              <Button icon={<SyncOutlined/>} loading={retryingFailedSubmissions} onClick={retryFailedTicketSubmissions}>批量重试到期</Button>
              <Button
                icon={<SyncOutlined/>}
                disabled={!selectedDeadLetterIds.length}
                loading={batchDeadLetterActionLoading}
                onClick={() => batchDeadLetterAction({ action: 'replay' })}
              >
                批量重放死信
              </Button>
              <Button
                disabled={!selectedDeadLetterIds.length}
                loading={creatingDeadLetterApproval}
                onClick={() => {
                  deadLetterApprovalForm.resetFields();
                  setDeadLetterApprovalModal({ visible: true, action: 'replay' });
                }}
              >
                提交重放审批
              </Button>
              <Button
                disabled={!selectedDeadLetterIds.length}
                loading={batchDeadLetterActionLoading}
                onClick={() => {
                  deadLetterCloseForm.resetFields();
                  setDeadLetterCloseModal({ visible: true });
                }}
              >
                人工关闭死信
              </Button>
              <Button
                disabled={!selectedDeadLetterIds.length}
                loading={creatingDeadLetterApproval}
                onClick={() => {
                  deadLetterApprovalForm.resetFields();
                  setDeadLetterApprovalModal({ visible: true, action: 'close' });
                }}
              >
                提交关闭审批
              </Button>
              <Button icon={<ReloadOutlined/>} onClick={() => { fetchRetryQueue(); fetchRetryQueueSummary(); fetchRetryQueueReport(); }}>刷新</Button>
            </Space>
          </div>
          <Table
            rowKey='id'
            loading={retryQueueLoading}
            columns={retryQueueColumns}
            dataSource={retryQueueItems}
            rowSelection={deadLetterRowSelection}
            pagination={{
              current: retryQueueQuery.currentPage,
              pageSize: retryQueueQuery.pageSize,
              total: retryQueueData.total || 0,
              showSizeChanger: true,
              onChange: (currentPage, pageSize) => setRetryQueueQuery({ ...retryQueueQuery, currentPage, pageSize })
            }}
          />
        </Tabs.TabPane>
        <Tabs.TabPane tab='自助目录' key='catalog'>
          <div className={styles.catalogSummary}>
            <Text strong={true}>自助 SLA 趋势</Text>
            <Table
              size='small'
              rowKey='date'
              columns={slaTrendColumns}
              dataSource={slaTrend}
              pagination={false}
            />
          </div>
          <div className={styles.dimensionGrid}>
            <div className={styles.dimensionPanel}>
              <Text strong={true}>团队目标趋势</Text>
              <Table
                size='small'
                rowKey={(record) => `${record.dimensionType}-${record.dimensionId || 'empty'}`}
                columns={dimensionTrendColumns}
                dataSource={teamTrends}
                pagination={false}
              />
            </div>
            <div className={styles.dimensionPanel}>
              <Text strong={true}>项目目标趋势</Text>
              <Table
                size='small'
                rowKey={(record) => `${record.dimensionType}-${record.dimensionId || 'empty'}`}
                columns={dimensionTrendColumns}
                dataSource={projectTrends}
                pagination={false}
              />
            </div>
            <div className={styles.dimensionPanel}>
              <Text strong={true}>申请类型趋势</Text>
              <Table
                size='small'
                rowKey={(record) => `${record.dimensionType}-${record.dimensionId || 'empty'}`}
                columns={dimensionTrendColumns}
                dataSource={requestTypeTrends}
                pagination={false}
              />
            </div>
          </div>
          <Table
            rowKey='key'
            columns={catalogColumns}
            dataSource={catalog}
            pagination={false}
            scroll={{ x: 'max-content' }}
          />
        </Tabs.TabPane>
        <Tabs.TabPane tab='连接器' key='configs'>
          <div className={styles.toolbar}>
            <Space className={styles.filterBar} size={8} wrap={true}>
              <InputSearch
                className={styles.keywordSearch}
                allowClear={true}
                placeholder='搜索连接器'
                onSearch={(q) => setConfigQuery({ ...configQuery, q, currentPage: 1 })}
              />
              <Select
                allowClear={true}
                placeholder='类型'
                style={{ width: 140 }}
                onChange={(provider) => setConfigQuery({ ...configQuery, provider, currentPage: 1 })}
              >
                {Object.keys(providerMap).map((key) => (
                  <Option value={key} key={key}>{providerMap[key]}</Option>
                ))}
              </Select>
            </Space>
            <Button type='primary' icon={<PlusOutlined/>} onClick={() => openConfigModal(null)}>新增连接器</Button>
          </div>
          <Table
            rowKey='id'
            loading={configsLoading}
            columns={configColumns}
            dataSource={configs}
            pagination={{
              current: configQuery.currentPage,
              pageSize: configQuery.pageSize,
              total: configsData.total || 0,
              showSizeChanger: true,
              onChange: (currentPage, pageSize) => setConfigQuery({ ...configQuery, currentPage, pageSize })
            }}
          />
        </Tabs.TabPane>
      </Tabs>
      <Modal
        visible={configModal.visible}
        title={configModal.record ? '编辑连接器' : '新增连接器'}
        width={720}
        confirmLoading={savingConfig}
        onCancel={() => setConfigModal({ visible: false, record: null })}
        onOk={() => {
          configForm.validateFields().then((values) => {
            try {
              saveConfig({
                record: configModal.record,
                values: {
                  ...values,
                  metadata: parseMetadata(values.metadata)
                }
              });
            } catch (err) {
              Modal.error({
                title: '扩展配置不是有效 JSON',
                content: err.message
              });
            }
          });
        }}
      >
        <Form form={configForm} layout='vertical'>
          {configModal.record?.id && (
            <Alert
              className={styles.callbackAlert}
              type='info'
              showIcon={true}
              message='外部 ITSM 状态回调'
              description={(
                <Space direction='vertical' size={4}>
                  <Text>回调地址：<Text code={true}>{callbackUrlText(configModal.record)}</Text></Text>
                  <Text>GitOps/IaC 门禁回写：<Text code={true}>{gitOpsGateCallbackUrlText(configModal.record)}</Text></Text>
                  <Text>签名头：<Text code={true}>X-CloudIaC-ITSM-Signature</Text>，格式：<Text code={true}>sha256=&lt;hmac_sha256&gt;</Text></Text>
                  <Text>回调密钥：在扩展配置中填写 <Text code={true}>callbackSecret</Text> 或 <Text code={true}>gitOpsGateCallbackSecret</Text>，签名内容为原始 JSON 请求体。</Text>
                  <Text>周期拉取：在扩展配置中填写 <Text code={true}>statusSyncEnabled</Text> 和 <Text code={true}>statusFetchPath</Text> 或 <Text code={true}>statusFetchUrl</Text>。</Text>
                </Space>
              )}
            />
          )}
          <Form.Item name='name' label='名称' rules={[ { required: true, message: '请输入名称' } ]}>
            <Input placeholder='例如：内部工单系统'/>
          </Form.Item>
          <Form.Item name='description' label='描述'>
            <Input placeholder='请输入描述'/>
          </Form.Item>
          <Space size={12} style={{ width: '100%' }} align='start'>
            <Form.Item name='provider' label='类型' style={{ width: 180 }}>
              <Select>
                {Object.keys(providerMap).map((key) => (
                  <Option value={key} key={key}>{providerMap[key]}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name='authType' label='认证' style={{ width: 180 }}>
              <Select>
                {Object.keys(authMap).map((key) => (
                  <Option value={key} key={key}>{authMap[key]}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name='status' label='状态' style={{ width: 160 }}>
              <Select>
                {Object.keys(configStatusMap).map((key) => (
                  <Option value={key} key={key}>{configStatusMap[key].label}</Option>
                ))}
              </Select>
            </Form.Item>
          </Space>
          <Form.Item name='baseUrl' label='Base URL'>
            <Input placeholder='https://itsm.example.com/api'/>
          </Form.Item>
          <Space size={12} style={{ width: '100%' }} align='start'>
            <Form.Item name='projectKey' label='项目 Key' style={{ width: 220 }}>
              <Input placeholder='CLOUD'/>
            </Form.Item>
            <Form.Item name='ticketType' label='工单类型' style={{ width: 220 }}>
              <Input placeholder='incident / change'/>
            </Form.Item>
            <Form.Item name='timeoutSeconds' label='超时秒数' style={{ width: 160 }}>
              <InputNumber min={1} max={120} style={{ width: '100%' }}/>
            </Form.Item>
          </Space>
          <Form.Item name='username' label='用户名'>
            <Input placeholder='Basic Auth 用户名'/>
          </Form.Item>
          <Form.Item name='password' label='密码'>
            <Input.Password placeholder={configModal.record?.passwordConfigured ? '留空则保持原密码' : '请输入密码'}/>
          </Form.Item>
          <Form.Item name='token' label='Token'>
            <Input.Password placeholder={configModal.record?.tokenConfigured ? '留空则保持原 Token' : '请输入 Token'}/>
          </Form.Item>
          <Form.Item name='metadata' label='扩展配置'>
            <TextArea rows={6} placeholder={`可填写 createTicketPath / createTicketUrl / browseTicketPath / callbackSecret / fieldMappings 等 JSON\n${JSON.stringify({
              createTicketPath: '/tickets',
              statusSyncEnabled: true,
              statusFetchPath: '/tickets/{externalKey}',
              submitRetryEnabled: true,
              submitRetryMaxAttempts: 3,
              submitRetryBackoffSeconds: 300,
              submitRetryMaxBackoffSeconds: 3600,
              browseTicketPath: '/tickets/{key}',
              callbackSecret: '请替换为外部 ITSM 回调密钥',
              gitOpsGateCallbackSecret: '请替换为 GitOps/IaC 门禁回写密钥',
              fieldDefaults: {
                'fields.labels': ['cloudiac']
              },
              fieldMappings: {
                'fields.summary': '$.title',
                'fields.description': '$.description',
                'u_cloudiac_operation_id': '$.operation.id'
              }
            }, null, 2)}`}/>
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        visible={selfServiceModal.visible}
        title='发起自助运维申请'
        width={720}
        confirmLoading={creatingSelfService}
        onCancel={() => setSelfServiceModal({ visible: false })}
        onOk={() => {
          selfServiceForm.validateFields().then((values) => {
            try {
              createSelfServiceTicket({
                values: selfServiceSubmitPayload(values)
              });
            } catch (err) {
              Modal.error({
                title: '参数 JSON 不是有效 JSON',
                content: err.message
              });
            }
          });
        }}
      >
        <Form form={selfServiceForm} layout='vertical'>
          <Space size={12} style={{ width: '100%' }} align='start'>
            <Form.Item name='requestType' label='申请类型' style={{ width: 220 }} rules={[ { required: true, message: '请选择申请类型' } ]}>
              <Select>
                {selfServiceCatalog.map((item) => (
                  <Option value={item.key} key={item.key}>{item.name}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item
              name='connectorId'
              label='ITSM 连接器'
              style={{ width: 260 }}
            >
              <Select allowClear={true} placeholder='可选，留空使用 CloudIaC 本地工单'>
                {enabledConfigs.map((item) => (
                  <Option value={item.id} key={item.id}>{item.name}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name='priority' label='优先级' style={{ width: 140 }}>
              <Select>
                <Option value='low'>低</Option>
                <Option value='medium'>中</Option>
                <Option value='high'>高</Option>
                <Option value='critical'>严重</Option>
              </Select>
            </Form.Item>
          </Space>
          <Alert
            type={enabledConfigs.length > 0 ? 'info' : 'warning'}
            showIcon={true}
            style={{ marginBottom: 12 }}
            message={enabledConfigs.length > 0 ? '未选择连接器时使用 CloudIaC 本地工单' : '暂无启用的外部 ITSM 连接器，将使用 CloudIaC 本地工单兜底'}
            description='自助运维申请始终会生成平台工单和审计记录；选择外部连接器时会继续尝试同步到对应 ITSM 系统。'
          />
          <Form.Item name='title' label='标题'>
            <Input placeholder='留空则使用申请类型名称'/>
          </Form.Item>
          <Form.Item noStyle={true} shouldUpdate={(prev, next) => prev.requestType !== next.requestType}>
            {({ getFieldValue }) => getFieldValue('requestType') === 'gitops_iac_change' && (
              <div>
                <Space size={12} style={{ width: '100%' }} align='start'>
                  <Form.Item name='gitOpsRepository' label='IaC 仓库' style={{ width: 260 }}>
                    <Input placeholder='例如 gitlab.example.com/platform/iac'/>
                  </Form.Item>
                  <Form.Item name='gitOpsBranch' label='变更分支' style={{ width: 180 }}>
                    <Input placeholder='feature/change'/>
                  </Form.Item>
                  <Form.Item name='gitOpsTargetBranch' label='目标分支' style={{ width: 180 }}>
                    <Input placeholder='main / prod'/>
                  </Form.Item>
                </Space>
                <Space size={12} style={{ width: '100%' }} align='start'>
                  <Form.Item name='gitOpsPullRequestUrl' label='PR/MR 地址' style={{ width: 360 }} rules={[ { required: true, message: '请输入 PR/MR 地址' } ]}>
                    <Input placeholder='https://gitlab.example.com/group/repo/-/merge_requests/1'/>
                  </Form.Item>
                  <Form.Item name='gitOpsReviewStatus' label='Review 状态' style={{ width: 160 }} rules={[ { required: true, message: '请选择 Review 状态' } ]}>
                    <Select>
                      <Option value='pending'>待评审</Option>
                      <Option value='approved'>已通过</Option>
                      <Option value='changes_requested'>需修改</Option>
                      <Option value='rejected'>已拒绝</Option>
                    </Select>
                  </Form.Item>
                </Space>
                <Space size={12} style={{ width: '100%' }} align='start'>
                  <Form.Item name='gitOpsPipelineUrl' label='流水线地址' style={{ width: 360 }}>
                    <Input placeholder='https://gitlab.example.com/group/repo/-/pipelines/1'/>
                  </Form.Item>
                  <Form.Item name='gitOpsPipelineStatus' label='流水线状态' style={{ width: 160 }} rules={[ { required: true, message: '请选择流水线状态' } ]}>
                    <Select>
                      <Option value='pending'>待执行</Option>
                      <Option value='running'>执行中</Option>
                      <Option value='passed'>已通过</Option>
                      <Option value='failed'>失败</Option>
                      <Option value='canceled'>已取消</Option>
                    </Select>
                  </Form.Item>
                </Space>
                <Form.Item name='gitOpsChangePath' label='IaC 变更路径'>
                  <Input placeholder='例如 envs/prod/network'/>
                </Form.Item>
                <Alert
                  type='info'
                  showIcon={true}
                  style={{ marginBottom: 12 }}
                  message='GitOps/IaC 门禁会写入工单载荷'
                  description='只有 Review 已通过且流水线已通过时，门禁状态才会标记为 passed；否则会记录为 waiting 或 blocked。'
                />
              </div>
            )}
          </Form.Item>
          <Form.Item name='description' label='申请说明' rules={[ { required: true, message: '请输入申请说明' } ]}>
            <TextArea rows={4} placeholder='说明业务背景、目标环境、变更范围、期望完成时间和回滚要求'/>
          </Form.Item>
          <Space size={12} style={{ width: '100%' }} align='start'>
            <Form.Item name='projectId' label='项目 ID' style={{ width: 260 }}>
              <Input placeholder='可选'/>
            </Form.Item>
            <Form.Item name='envId' label='环境 ID' style={{ width: 260 }}>
              <Input placeholder='可选'/>
            </Form.Item>
          </Space>
          <Form.Item name='params' label='补充参数 JSON'>
            <TextArea rows={4} placeholder='例如 {"role":"viewer","scope":"prod"}'/>
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        visible={catalogPolicyModal.visible}
        title='配置自助目录策略'
        width={760}
        confirmLoading={savingCatalogPolicy}
        onCancel={() => {
          setCatalogPolicyModal({ visible: false, record: null });
          setCatalogPolicyHistory([]);
        }}
        onOk={() => {
          catalogPolicyForm.validateFields().then((values) => {
            saveCatalogPolicy({
              record: catalogPolicyModal.record,
              values
            });
          });
        }}
      >
        <Form form={catalogPolicyForm} layout='vertical'>
          <Alert
            type='info'
            showIcon={true}
            style={{ marginBottom: 12 }}
            message={catalogPolicyModal.record?.name || catalogPolicyModal.record?.key}
            description='策略配置会影响后续自助申请的上架状态、权限角色、适用范围和 SLA；已创建工单保留提交时的策略快照。'
          />
          <Space size={12} style={{ width: '100%' }} align='start'>
            <Form.Item name='enabled' label='策略状态' valuePropName='checked'>
              <Switch checkedChildren='启用' unCheckedChildren='停用'/>
            </Form.Item>
            <Form.Item name='slaMinutes' label='SLA 分钟' style={{ width: 180 }} rules={[ { required: true, message: '请输入 SLA 分钟数' } ]}>
              <InputNumber min={1} max={43200} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='reset' label='恢复默认' valuePropName='checked'>
              <Switch checkedChildren='恢复' unCheckedChildren='保留'/>
            </Form.Item>
          </Space>
          <Form.Item name='policyName' label='策略名称' rules={[ { required: true, message: '请输入策略名称' } ]}>
            <Input placeholder='请输入策略名称'/>
          </Form.Item>
          <Form.Item name='policyDescription' label='策略说明'>
            <TextArea rows={3} placeholder='说明适用场景、审批口径或团队约束'/>
          </Form.Item>
          <Form.Item name='requiredRoles' label='允许角色' rules={[ { required: true, message: '请选择允许角色' } ]}>
            <Select mode='multiple' placeholder='请选择角色'>
              {Object.entries(policyRoleMap).map(([value, label]) => (
                <Option key={value} value={value}>{label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name='allowedScopes' label='适用范围' rules={[ { required: true, message: '请选择适用范围' } ]}>
            <Select mode='multiple' placeholder='请选择范围'>
              {Object.entries(policyScopeMap).map(([value, label]) => (
                <Option key={value} value={value}>{label}</Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
        <div className={styles.catalogPolicyHistory}>
          <div className={styles.catalogPolicyHistoryHeader}>
            <Text strong={true}>策略变更历史</Text>
            {catalogPolicyModal.record?.policyVersion ? (
              <Tag color='geekblue'>当前 v{catalogPolicyModal.record.policyVersion}</Tag>
            ) : null}
          </div>
          <Table
            size='small'
            rowKey={(record) => `${record.key}-${record.version}-${record.updatedAt}`}
            columns={catalogPolicyHistoryColumns}
            dataSource={catalogPolicyHistory}
            loading={catalogPolicyHistoryLoading}
            pagination={false}
            scroll={{ x: 'max-content' }}
            locale={{
              emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无策略历史'/>
            }}
          />
        </div>
      </Modal>
      <Modal
        visible={deadLetterCloseModal.visible}
        title={`人工关闭死信（${selectedDeadLetterIds.length}）`}
        confirmLoading={batchDeadLetterActionLoading}
        onCancel={() => setDeadLetterCloseModal({ visible: false })}
        onOk={() => {
          deadLetterCloseForm.validateFields().then((values) => {
            batchDeadLetterAction({
              action: 'close',
              reason: values.reason
            });
          });
        }}
      >
        <Form form={deadLetterCloseForm} layout='vertical'>
          <Form.Item name='reason' label='关闭原因' rules={[ { required: true, message: '请输入关闭原因' } ]}>
            <TextArea rows={4} placeholder='说明人工确认依据、外部处理结果或无需继续提交的原因'/>
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        visible={deadLetterApprovalModal.visible}
        title={`提交死信${deadLetterApprovalModal.action === 'close' ? '关闭' : '重放'}审批（${selectedDeadLetterIds.length}）`}
        confirmLoading={creatingDeadLetterApproval}
        onCancel={() => setDeadLetterApprovalModal({ visible: false, action: 'replay' })}
        onOk={() => {
          deadLetterApprovalForm.validateFields().then((values) => {
            createDeadLetterApproval({
              action: deadLetterApprovalModal.action,
              values
            });
          });
        }}
      >
        <Form form={deadLetterApprovalForm} layout='vertical'>
          <Form.Item name='reason' label='申请原因' rules={[ { required: true, message: '请输入申请原因' } ]}>
            <TextArea rows={4} placeholder='说明处理背景、风险判断、期望处置方式和回滚要求'/>
          </Form.Item>
          <Form.Item name='evidenceUrl' label='主证据链接'>
            <Input placeholder='可填写 PR、变更单、事故复盘或外部工单链接'/>
          </Form.Item>
          <Form.List name='evidenceItems'>
            {(fields, { add, remove }) => (
              <div className={styles.deadLetterEvidenceList}>
                <div className={styles.deadLetterEvidenceHeader}>
                  <Text strong={true}>补充证据</Text>
                  <Button size='small' icon={<PlusOutlined />} onClick={() => add({ type: 'link' })}>
                    添加证据
                  </Button>
                </div>
                {fields.map(({ key, name, fieldKey }) => (
                  <div className={styles.deadLetterEvidenceItem} key={key}>
                    <div className={styles.deadLetterEvidenceItemHeader}>
                      <Form.Item
                        name={[name, 'label']}
                        fieldKey={[fieldKey, 'label']}
                        className={styles.deadLetterEvidenceLabel}
                      >
                        <Input placeholder='证据标题'/>
                      </Form.Item>
                      <Form.Item
                        name={[name, 'type']}
                        fieldKey={[fieldKey, 'type']}
                        className={styles.deadLetterEvidenceType}
                        initialValue='link'
                      >
                        <Select placeholder='类型'>
                          {deadLetterEvidenceTypeOptions.map((item) => (
                            <Option key={item.value} value={item.value}>{item.label}</Option>
                          ))}
                        </Select>
                      </Form.Item>
                      <Button icon={<DeleteOutlined />} onClick={() => remove(name)} />
                    </div>
                    <Form.Item
                      name={[name, 'url']}
                      fieldKey={[fieldKey, 'url']}
                    >
                      <Input placeholder='https://example.com/change/123'/>
                    </Form.Item>
                    <Form.Item
                      name={[name, 'note']}
                      fieldKey={[fieldKey, 'note']}
                    >
                      <TextArea rows={2} placeholder='证据说明、审批口径或回滚依据'/>
                    </Form.Item>
                  </div>
                ))}
              </div>
            )}
          </Form.List>
        </Form>
	    </Modal>
      <Modal
        visible={statusModal.visible}
        title='更新工单状态'
        confirmLoading={updatingStatus}
        onCancel={() => setStatusModal({ visible: false, record: null })}
        onOk={() => {
          statusForm.validateFields().then((values) => {
            updateTicketStatus({ id: statusModal.record.id, values });
          });
        }}
      >
        <Form form={statusForm} layout='vertical'>
          <Form.Item name='status' label='状态' rules={[ { required: true, message: '请选择状态' } ]}>
            <Select>
              {Object.keys(ticketStatusMap).map((key) => (
                <Option value={key} key={key}>{ticketStatusMap[key].label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name='externalId' label='外部 ID'>
            <Input/>
          </Form.Item>
          <Form.Item name='externalKey' label='外部单号'>
            <Input/>
          </Form.Item>
          <Form.Item name='externalUrl' label='外部地址'>
            <Input/>
          </Form.Item>
          <Form.Item name='comment' label='备注'>
            <Input/>
          </Form.Item>
        </Form>
      </Modal>
      <Drawer
        title='工单详情'
        visible={drawer.visible}
        width={760}
        onClose={() => setDrawer({ visible: false, id: '' })}
      >
        <TicketDetail
          detail={detail}
          loading={ticketsLoading}
          onUpdateStatus={openStatusModal}
        />
      </Drawer>
    </Layout>
  );
};

export default CloudItsmPage;
