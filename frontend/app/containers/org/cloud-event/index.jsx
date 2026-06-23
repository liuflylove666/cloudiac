import React, { useMemo, useState } from 'react';
import {
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  Button
} from 'antd';
import { DeleteOutlined, EditOutlined, EyeOutlined, InboxOutlined, KeyOutlined, PauseCircleOutlined, PlayCircleOutlined, PlusOutlined, ReloadOutlined, SendOutlined, SyncOutlined, UnorderedListOutlined } from '@ant-design/icons';
import { useRequest } from 'ahooks';
import moment from 'moment';
import PageHeader from 'components/pageHeader';
import Layout from 'components/common/layout';
import { requestWrapper } from 'utils/request';
import cloudEventAPI from 'services/cloud-event';
import styles from './styles.less';

const { Option } = Select;
const { Search: InputSearch } = Input;
const { Text } = Typography;

const sourceMap = {
  account: '云账号',
  sync: '同步',
  operation: '云操作',
  risk: '风险',
  cost: '成本',
  cmdb: 'CMDB',
  notification: '通知',
  itsm: 'ITSM'
};

const levelMap = {
  info: { label: '信息', color: 'processing' },
  warning: { label: '警告', color: 'warning' },
  error: { label: '错误', color: 'error' }
};

const eventTypeMap = {
  'cloud.*': '多云全部事件',
  'account.*': '云账号事件',
  'sync.*': '同步事件',
  'operation.*': '云操作事件',
  'risk.*': '风险事件',
  'cost.*': '成本事件',
  'cmdb.*': 'CMDB事件',
  'notification.*': '通知事件',
  'webhook.*': 'Webhook事件',
  'itsm.*': 'ITSM事件',
  'cloud_account.validated': '云账号验证',
  'cloud_account.health_checked': '云账号健康检查',
  'cloud.sync.task.batch_rerun_approval_requested': '云采集批量重跑待审批',
  'cloud.sync.task.batch_rerun_started': '云采集批量重跑启动',
  'cloud.sync.task.batch_rerun_finished': '云采集批量重跑完成',
  'cloud.sync.task.slow_api_detected': '云采集慢 API 告警',
  'cloud_operation.created': '云操作创建',
  'cloud_operation.complete': '云操作完成',
  'cloud_operation.failed': '云操作失败',
  'cloud_operation.aborted': '云操作取消',
  'cloud_operation.rejected': '云操作驳回',
  'risk.detected': '风险发现',
  'risk.status_updated': '风险状态更新',
  'risk.suppressed': '风险例外',
  'risk.remediation_ticket_created': '风险整改工单创建',
  'risk.remediation_status_synced': '风险整改状态同步',
  'risk.drift_auto_repair_triggered': '漂移自动修复任务触发',
  'risk.drift_auto_repair_result_synced': '漂移自动修复结果同步',
  'risk.drift_auto_repair_retry_approval_requested': '漂移自动修复重试审批',
  'risk.drift_auto_repair_rollback_strategy_recorded': '漂移自动修复回滚策略记录',
  'risk.drift_auto_repair_approval_notification_requested': '漂移自动修复审批通知',
  'risk.drift_auto_repair_sla_escalated': '漂移自动修复 SLA 升级',
  'cost.budget.threshold_exceeded': '预算超阈值',
  'cost.insight.detected': '成本建议发现',
  'cost.pull.completed': '成本拉取完成',
  'cost.pull.failed': '成本拉取失败',
  'cost.sync.schedule.failed': '成本同步计划失败',
  'cost.sync.schedule.auto_paused': '成本同步计划自动暂停',
  'webhook.test': 'Webhook 测试',
  'webhook.delivery_failed': 'Webhook 投递失败',
  'itsm.ticket.created': 'ITSM 工单创建',
  'itsm.ticket.submitted': 'ITSM 工单提交',
  'itsm.ticket.failed': 'ITSM 工单失败',
  'itsm.ticket.updated': 'ITSM 工单更新',
  'itsm.ticket.retry_submitted': 'ITSM 工单补偿提交成功',
  'itsm.ticket.retry_failed': 'ITSM 工单补偿提交失败',
  'itsm.ticket.dead_letter_closed': 'ITSM 死信人工关闭',
  'itsm.dead_letter.batch_replayed': 'ITSM 死信批量重放',
  'itsm.dead_letter.batch_closed': 'ITSM 死信批量关闭',
  'itsm.dead_letter.approval_requested': 'ITSM 死信处置审批提交',
  'itsm.dead_letter.approval_rejected': 'ITSM 死信处置审批驳回',
  'itsm.dead_letter.approval_executed': 'ITSM 死信审批处置完成',
  'itsm.dead_letter.approval_execute_failed': 'ITSM 死信审批处置失败',
  'cmdb.asset.created': '资产新增',
  'cmdb.asset.updated': '资产变更'
};

const webhookStatusMap = {
  enable: { label: '启用', color: 'success' },
  disable: { label: '停用', color: 'default' }
};

const deliveryStatusMap = {
  pending: { label: '等待', color: 'processing' },
  success: { label: '成功', color: 'success' },
  failed: { label: '失败', color: 'error' }
};

const deliveryModeMap = {
  initial: '初始',
  manual: '手动',
  auto: '自动',
  test: '测试'
};

const deadLetterStatusMap = {
  open: { label: '待处理', color: 'error' },
  replayed: { label: '已重放', color: 'success' },
  ignored: { label: '已忽略', color: 'default' }
};

const deadLetterReasonMap = {
  max_retries_reached: '达到最大重试次数',
  retry_window_expired: '超过最大重试窗口',
  unscheduled_failure: '无后续重试'
};

const signatureVerifyStatusMap = {
  valid: { label: '通过', color: 'success' },
  invalid: { label: '失败', color: 'error' },
  skipped: { label: '跳过', color: 'default' }
};

const renderTime = (value) => !value || String(value).indexOf('0001-01-01') === 0 ? '-' : moment(value).format('YYYY-MM-DD HH:mm:ss');
const formatPercent = (value) => {
  const number = Number(value);
  if (!Number.isFinite(number)) {
    return '-';
  }
  return `${number.toFixed(2)}%`;
};
const levelTag = (value) => {
  const item = levelMap[value] || { label: value || '-', color: 'default' };
  return <Tag color={item.color}>{item.label}</Tag>;
};
const jsonText = (value) => {
  if (!value || Object.keys(value || {}).length === 0) {
    return '-';
  }
  return JSON.stringify(value, null, 2);
};

const JsonBlock = ({ value }) => (
  <pre className={styles.jsonBlock}>{jsonText(value)}</pre>
);

const QueueMetric = ({ label, value, tone, compact }) => (
  <div className={`${styles.queueMetric} ${tone ? styles[tone] : ''}`}>
    <div className={`${styles.queueMetricValue} ${compact ? styles.compactMetricValue : ''}`}>{value || 0}</div>
    <div className={styles.queueMetricLabel}>{label}</div>
  </div>
);

const EventDetail = ({ detail = {}, loading }) => {
  if (loading) {
    return <Empty description='加载中'/>;
  }
  if (!detail.id) {
    return <Empty description='请选择事件'/>;
  }
  return (
    <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
      <Descriptions size='small' bordered={true} column={2}>
        <Descriptions.Item label='事件 ID'>{detail.id || '-'}</Descriptions.Item>
        <Descriptions.Item label='级别'>{levelTag(detail.level)}</Descriptions.Item>
        <Descriptions.Item label='来源'>{sourceMap[detail.source] || detail.source || '-'}</Descriptions.Item>
        <Descriptions.Item label='类型'>{eventTypeMap[detail.eventType] || detail.eventType || '-'}</Descriptions.Item>
        <Descriptions.Item label='状态'>{detail.status || '-'}</Descriptions.Item>
        <Descriptions.Item label='发生时间'>{renderTime(detail.occurredAt)}</Descriptions.Item>
        <Descriptions.Item label='资源'>{detail.resourceName || detail.assetName || detail.resourceId || '-'}</Descriptions.Item>
        <Descriptions.Item label='云厂商'>{detail.provider || '-'}</Descriptions.Item>
        <Descriptions.Item label='账号'>{detail.accountId || detail.cloudAccountId || '-'}</Descriptions.Item>
        <Descriptions.Item label='区域'>{detail.region || '-'}</Descriptions.Item>
        <Descriptions.Item label='项目'>{detail.projectName || detail.projectId || '-'}</Descriptions.Item>
        <Descriptions.Item label='环境'>{detail.envName || detail.envId || '-'}</Descriptions.Item>
        <Descriptions.Item label='操作者'>{detail.actorName || detail.actorId || '-'}</Descriptions.Item>
        <Descriptions.Item label='标题'>{detail.title || '-'}</Descriptions.Item>
        <Descriptions.Item label='消息' span={2}>{detail.message || '-'}</Descriptions.Item>
      </Descriptions>
      <div className={styles.detailSection}>
        <Text strong={true}>事件载荷</Text>
        <JsonBlock value={detail.payload}/>
      </div>
    </Space>
  );
};

const DeliveryDetail = ({ detail = {} }) => {
  if (!detail.id) {
    return <Empty description='请选择投递记录'/>;
  }
  const statusItem = deliveryStatusMap[detail.status] || {};
  return (
    <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
      <Descriptions size='small' bordered={true} column={2}>
        <Descriptions.Item label='投递 ID'>{detail.id}</Descriptions.Item>
        <Descriptions.Item label='Webhook'>{detail.webhookName || detail.webhookId || '-'}</Descriptions.Item>
        <Descriptions.Item label='状态'>{statusItem.label || detail.status || '-'}</Descriptions.Item>
        <Descriptions.Item label='模式'>{deliveryModeMap[detail.deliveryMode] || detail.deliveryMode || '-'}</Descriptions.Item>
        <Descriptions.Item label='事件类型'>{eventTypeMap[detail.eventType] || detail.eventType || '-'}</Descriptions.Item>
        <Descriptions.Item label='事件 ID'>{detail.eventId || '-'}</Descriptions.Item>
        <Descriptions.Item label='尝试次数'>{detail.attempt || 1}</Descriptions.Item>
        <Descriptions.Item label='响应码'>{detail.responseCode || '-'}</Descriptions.Item>
        <Descriptions.Item label='目标地址' span={2}>{detail.targetUrl || '-'}</Descriptions.Item>
        <Descriptions.Item label='父投递'>{detail.parentDeliveryId || '-'}</Descriptions.Item>
        <Descriptions.Item label='下一次重试'>{renderTime(detail.nextRetryAt)}</Descriptions.Item>
        <Descriptions.Item label='已重试时间'>{renderTime(detail.retriedAt)}</Descriptions.Item>
        <Descriptions.Item label='投递时间'>{renderTime(detail.deliveredAt)}</Descriptions.Item>
        <Descriptions.Item label='验签结果'>
          {detail.signatureVerifyStatus ? (
            <Tag color={(signatureVerifyStatusMap[detail.signatureVerifyStatus] || {}).color || 'default'}>
              {(signatureVerifyStatusMap[detail.signatureVerifyStatus] || {}).label || detail.signatureVerifyStatus}
            </Tag>
          ) : '-'}
        </Descriptions.Item>
        <Descriptions.Item label='验签版本'>{detail.signatureVerifyVersion || '-'}</Descriptions.Item>
        <Descriptions.Item label='验签时间'>{renderTime(detail.signatureVerifiedAt)}</Descriptions.Item>
        <Descriptions.Item label='错误' span={2}>{detail.errorMessage || '-'}</Descriptions.Item>
        <Descriptions.Item label='验签消息' span={2}>{detail.signatureVerifyMessage || '-'}</Descriptions.Item>
      </Descriptions>
      <div className={styles.detailSection}>
        <Text strong={true}>请求载荷</Text>
        <JsonBlock value={detail.requestPayload}/>
      </div>
      <div className={styles.detailSection}>
        <Text strong={true}>请求头</Text>
        <JsonBlock value={detail.requestHeaders}/>
      </div>
      <div className={styles.detailSection}>
        <Text strong={true}>签名信息</Text>
        <JsonBlock value={detail.signatureInfo}/>
      </div>
      <div className={styles.detailSection}>
        <Text strong={true}>响应头</Text>
        <JsonBlock value={detail.responseHeaders}/>
      </div>
      <div className={styles.detailSection}>
        <Text strong={true}>响应体</Text>
        <pre className={styles.jsonBlock}>{detail.responseBody || '-'}</pre>
      </div>
    </Space>
  );
};

const CloudEventPage = ({ match }) => {
  const { orgId } = match.params || {};
  const [ webhookForm ] = Form.useForm();
  const [ secretForm ] = Form.useForm();
  const [ query, setQuery ] = useState({
    currentPage: 1,
    pageSize: 10
  });
  const [ drawer, setDrawer ] = useState({
    visible: false,
    id: ''
  });
  const [ webhookModal, setWebhookModal ] = useState({
    visible: false,
    record: null
  });
  const [ secretModal, setSecretModal ] = useState({
    visible: false,
    record: null
  });
  const [ deliveryDrawer, setDeliveryDrawer ] = useState({
    visible: false,
    record: null
  });
  const [ deadLetterDrawer, setDeadLetterDrawer ] = useState({
    visible: false
  });
  const [ deliveryDetail, setDeliveryDetail ] = useState(null);
  const [ deliveryDetailLoading, setDeliveryDetailLoading ] = useState(false);
  const [ webhookSaving, setWebhookSaving ] = useState(false);
  const [ secretSaving, setSecretSaving ] = useState(false);

  const {
    loading,
    data = {},
    run: fetchList
  } = useRequest(
    () => requestWrapper(cloudEventAPI.list.bind(null, { orgId, ...query })),
    {
      refreshDeps: [ query, orgId ]
    }
  );
  const {
    loading: webhookLoading,
    data: webhookData = {},
    run: fetchWebhooks
  } = useRequest(
    () => requestWrapper(cloudEventAPI.webhooks.bind(null, { orgId, currentPage: 1, pageSize: 5 })),
    {
      refreshDeps: [ orgId ]
    }
  );
  const {
    loading: deliveryLoading,
    data: deliveryData = {},
    run: fetchDeliveries
  } = useRequest(
    (record) => requestWrapper(cloudEventAPI.webhookDeliveries.bind(null, { orgId, id: record.id, currentPage: 1, pageSize: 20 })),
    {
      manual: true
    }
  );
  const {
    loading: queueSummaryLoading,
    data: queueSummary = {},
    run: fetchQueueSummary
  } = useRequest(
    () => requestWrapper(cloudEventAPI.webhookQueueSummary.bind(null, { orgId })),
    {
      refreshDeps: [ orgId ]
    }
  );
  const {
    loading: deadLetterLoading,
    data: deadLetterData = {},
    run: fetchDeadLetters
  } = useRequest(
    (params = {}) => requestWrapper(cloudEventAPI.webhookDeadLetters.bind(null, { orgId, currentPage: 1, pageSize: 20, ...params })),
    {
      manual: true
    }
  );

  const list = data.list || [];
  const webhookList = webhookData.list || [];
  const deliveryList = deliveryData.list || [];
  const deadLetterList = deadLetterData.list || [];
  const detail = useMemo(() => list.find((item) => item.id === drawer.id) || {}, [ list, drawer.id ]);
  const refreshAll = () => {
    fetchList();
    fetchWebhooks();
    fetchQueueSummary();
    if (deliveryDrawer.record) {
      fetchDeliveries(deliveryDrawer.record);
    }
    if (deadLetterDrawer.visible) {
      fetchDeadLetters();
    }
  };

  const columns = useMemo(() => [
    {
      title: '事件',
      dataIndex: 'title',
      width: 220,
      ellipsis: true,
      render: (text, record) => <a onClick={() => setDrawer({ visible: true, id: record.id })}>{text || eventTypeMap[record.eventType] || record.eventType || record.id}</a>
    },
    {
      title: '级别',
      dataIndex: 'level',
      width: 90,
      render: levelTag
    },
    {
      title: '来源',
      dataIndex: 'source',
      width: 100,
      render: (text) => sourceMap[text] || text || '-'
    },
    {
      title: '类型',
      dataIndex: 'eventType',
      width: 170,
      render: (text) => eventTypeMap[text] || text || '-'
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      render: (text) => text || '-'
    },
    {
      title: '资源',
      dataIndex: 'resourceName',
      width: 220,
      ellipsis: true,
      render: (text, record) => text || record.assetName || record.resourceId || '-'
    },
    {
      title: '操作者',
      dataIndex: 'actorName',
      width: 140,
      render: (text, record) => text || record.actorId || '-'
    },
    {
      title: '发生时间',
      dataIndex: 'occurredAt',
      width: 180,
      render: renderTime
    }
  ], []);
  const webhookColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      width: 180,
      ellipsis: true
    },
    {
      title: '目标地址',
      dataIndex: 'targetUrl',
      width: 260,
      ellipsis: true
    },
    {
      title: '事件类型',
      dataIndex: 'eventTypes',
      width: 180,
      render: (value) => !value || value.length === 0 ? '全部' : value.slice(0, 2).map((item) => <Tag key={item}>{eventTypeMap[item] || item}</Tag>)
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (value) => {
        const item = webhookStatusMap[value] || { label: value || '-', color: 'default' };
        return <Tag color={item.color}>{item.label}</Tag>;
      }
    },
    {
      title: '最近投递',
      dataIndex: 'lastStatus',
      width: 120,
      render: (value, record) => {
        const item = deliveryStatusMap[value] || { label: value || '-', color: 'default' };
        return <Tag color={item.color}>{record.lastStatusCode ? `${item.label} ${record.lastStatusCode}` : item.label}</Tag>;
      }
    },
    {
      title: '成功/失败',
      dataIndex: 'deliveryCount',
      width: 110,
      render: (_, record) => `${record.successCount || 0}/${record.failedCount || 0}`
    },
    {
      title: '重试',
      dataIndex: 'maxRetries',
      width: 180,
      render: (_, record) => {
        const windowText = record.maxRetryDuration ? `${record.maxRetryDuration}s` : '不限';
        return `${record.maxRetries || 1} 次 / ${record.retryInterval || 60}s / 抖动${record.retryJitterPercent || 0}% / 窗口${windowText}`;
      }
    },
    {
      title: '签名',
      dataIndex: 'secretVersion',
      width: 120,
      render: (value, record) => {
        if (!value) {
          return '-';
        }
        const inGrace = record.previousSecretVersion && renderTime(record.previousSecretExpiresAt) !== '-';
        return (
          <Space size={4}>
            <Tag>v{value}</Tag>
            {inGrace ? <Tag color='warning'>宽限</Tag> : null}
          </Space>
        );
      }
    },
    {
      title: '操作',
      dataIndex: 'op',
      width: 420,
      render: (_, record) => (
        <Space size={4}>
          <Button type='link' size='small' icon={<SendOutlined/>} onClick={() => testWebhook(record)}>测试</Button>
          <Button type='link' size='small' icon={<UnorderedListOutlined/>} onClick={() => openDeliveryDrawer(record)}>投递</Button>
          <Button type='link' size='small' icon={record.status === 'enable' ? <PauseCircleOutlined/> : <PlayCircleOutlined/>} onClick={() => toggleWebhookStatus(record)}>
            {record.status === 'enable' ? '暂停' : '恢复'}
          </Button>
          <Button type='link' size='small' icon={<KeyOutlined/>} onClick={() => openSecretModal(record)}>轮换</Button>
          <Button type='link' size='small' icon={<EditOutlined/>} onClick={() => openWebhookModal(record)}>编辑</Button>
          <Button type='link' size='small' danger={true} icon={<DeleteOutlined/>} onClick={() => deleteWebhook(record)}>删除</Button>
        </Space>
      )
    }
  ];
  const shardColumns = [
    {
      title: '分片',
      dataIndex: 'shardIndex',
      width: 90,
      render: (value, record) => `${Number(value || 0) + 1}/${record.shardTotal || queueSummary.shardTotal || '-'}`
    },
    {
      title: '排队',
      dataIndex: 'queued',
      width: 80,
      render: (value) => value || 0
    },
    {
      title: '已到期',
      dataIndex: 'due',
      width: 80,
      render: (value) => value || 0
    },
    {
      title: '未来',
      dataIndex: 'future',
      width: 80,
      render: (value) => value || 0
    },
    {
      title: '已消费',
      dataIndex: 'consumed',
      width: 90,
      render: (value) => value || 0
    },
    {
      title: '成功/失败/跳过',
      dataIndex: 'success',
      width: 140,
      render: (_, record) => `${record.success || 0}/${record.failed || 0}/${record.skipped || 0}`
    },
    {
      title: '成功率',
      dataIndex: 'successRate',
      width: 90,
      render: formatPercent
    },
    {
      title: '最早待跑',
      dataIndex: 'oldestQueuedAt',
      width: 170,
      render: renderTime
    },
    {
      title: '最近消费',
      dataIndex: 'lastConsumedAt',
      width: 170,
      render: renderTime
    }
  ];
  const deliveryColumns = [
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (value) => {
        const item = deliveryStatusMap[value] || { label: value || '-', color: 'default' };
        return <Tag color={item.color}>{item.label}</Tag>;
      }
    },
    {
      title: '模式',
      dataIndex: 'deliveryMode',
      width: 90,
      render: (value) => deliveryModeMap[value] || value || '-'
    },
    {
      title: '事件类型',
      dataIndex: 'eventType',
      width: 180,
      render: (value) => eventTypeMap[value] || value || '-'
    },
    {
      title: '尝试',
      dataIndex: 'attempt',
      width: 80,
      render: (value) => value || 1
    },
    {
      title: '响应码',
      dataIndex: 'responseCode',
      width: 90,
      render: (value) => value || '-'
    },
    {
      title: '下一次重试',
      dataIndex: 'nextRetryAt',
      width: 170,
      render: renderTime
    },
    {
      title: '投递时间',
      dataIndex: 'deliveredAt',
      width: 170,
      render: renderTime
    },
    {
      title: '错误',
      dataIndex: 'errorMessage',
      width: 260,
      ellipsis: true,
      render: (value) => value || '-'
    },
    {
      title: '操作',
      dataIndex: 'op',
      width: 150,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button type='link' size='small' icon={<EyeOutlined/>} loading={deliveryDetailLoading} onClick={() => openDeliveryDetail(record)}>详情</Button>
          {record.status === 'failed' ? (
            <Button type='link' size='small' icon={<ReloadOutlined/>} onClick={() => retryDelivery(record)}>重试</Button>
          ) : null}
        </Space>
      )
    }
  ];
  const deadLetterColumns = [
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (value) => {
        const item = deadLetterStatusMap[value] || { label: value || '-', color: 'default' };
        return <Tag color={item.color}>{item.label}</Tag>;
      }
    },
    {
      title: 'Webhook',
      dataIndex: 'webhookName',
      width: 160,
      ellipsis: true,
      render: (value, record) => value || record.webhookId || '-'
    },
    {
      title: '事件类型',
      dataIndex: 'eventType',
      width: 180,
      render: (value) => eventTypeMap[value] || value || '-'
    },
    {
      title: '原因',
      dataIndex: 'reason',
      width: 150,
      render: (value) => deadLetterReasonMap[value] || value || '-'
    },
    {
      title: '尝试',
      dataIndex: 'attempt',
      width: 70,
      render: (value) => value || 1
    },
    {
      title: '响应码',
      dataIndex: 'responseCode',
      width: 90,
      render: (value) => value || '-'
    },
    {
      title: '目标地址',
      dataIndex: 'targetUrl',
      width: 260,
      ellipsis: true
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 170,
      render: renderTime
    },
    {
      title: '重放时间',
      dataIndex: 'replayedAt',
      width: 170,
      render: renderTime
    },
    {
      title: '操作',
      dataIndex: 'op',
      width: 130,
      fixed: 'right',
      render: (_, record) => record.status === 'open' ? (
        <Space size={4}>
          <Button type='link' size='small' icon={<SendOutlined/>} onClick={() => replayDeadLetter(record)}>重放</Button>
          <Button type='link' size='small' onClick={() => ignoreDeadLetter(record)}>忽略</Button>
        </Space>
      ) : '-'
    }
  ];

  const resetQuery = (patch) => setQuery({ ...query, ...patch, currentPage: 1 });
  function openDeliveryDrawer(record) {
    setDeliveryDetail(null);
    setDeliveryDrawer({
      visible: true,
      record
    });
    fetchDeliveries(record);
  }
  function closeDeliveryDrawer() {
    setDeliveryDrawer({ visible: false, record: null });
    setDeliveryDetail(null);
  }
  function openDeadLetterDrawer() {
    setDeadLetterDrawer({ visible: true });
    fetchDeadLetters();
  }
  function closeDeadLetterDrawer() {
    setDeadLetterDrawer({ visible: false });
  }
  async function openDeliveryDetail(record) {
    const webhook = deliveryDrawer.record;
    if (!webhook || !record) {
      return;
    }
    setDeliveryDetailLoading(true);
    try {
      const detail = await requestWrapper(cloudEventAPI.webhookDelivery.bind(null, {
        orgId,
        id: webhook.id,
        deliveryId: record.id
      }));
      setDeliveryDetail(detail);
    } finally {
      setDeliveryDetailLoading(false);
    }
  }
  function openWebhookModal(record) {
    setWebhookModal({
      visible: true,
      record: record || null
    });
    webhookForm.setFieldsValue(record ? {
      name: record.name,
      description: record.description,
      targetUrl: record.targetUrl,
      eventTypes: record.eventTypes || [],
      sources: record.sources || [],
      status: record.status || 'enable',
      timeoutSeconds: record.timeoutSeconds || 5,
      maxRetries: record.maxRetries || 3,
      retryInterval: record.retryInterval || 60,
      retryJitterPercent: record.retryJitterPercent || 0,
      maxRetryDuration: record.maxRetryDuration || 0,
      secret: ''
    } : {
      status: 'enable',
      timeoutSeconds: 5,
      maxRetries: 3,
      retryInterval: 60,
      retryJitterPercent: 0,
      maxRetryDuration: 0,
      eventTypes: [],
      sources: []
    });
  }
  const closeWebhookModal = () => {
    setWebhookModal({ visible: false, record: null });
    webhookForm.resetFields();
  };
  function openSecretModal(record) {
    setSecretModal({
      visible: true,
      record
    });
    secretForm.setFieldsValue({
      secret: '',
      gracePeriodSeconds: 604800
    });
  }
  const closeSecretModal = () => {
    setSecretModal({ visible: false, record: null });
    secretForm.resetFields();
  };
  const submitSecretRotation = async () => {
    const values = await secretForm.validateFields();
    if (!secretModal.record) {
      return;
    }
    setSecretSaving(true);
    try {
      await requestWrapper(cloudEventAPI.rotateWebhookSecret.bind(null, {
        orgId,
        id: secretModal.record.id,
        data: values
      }), { autoSuccess: true });
      closeSecretModal();
      fetchWebhooks();
    } finally {
      setSecretSaving(false);
    }
  };
  const submitWebhook = async () => {
    const values = await webhookForm.validateFields();
    const payload = {
      ...values,
      eventTypes: values.eventTypes || [],
      sources: values.sources || []
    };
    if (webhookModal.record && !payload.secret) {
      delete payload.secret;
    }
    setWebhookSaving(true);
    try {
      await requestWrapper(
        webhookModal.record
          ? cloudEventAPI.updateWebhook.bind(null, { orgId, id: webhookModal.record.id, data: payload })
          : cloudEventAPI.createWebhook.bind(null, { orgId, data: payload }),
        { autoSuccess: true }
      );
      closeWebhookModal();
      fetchWebhooks();
      fetchQueueSummary();
    } finally {
      setWebhookSaving(false);
    }
  };
  function deleteWebhook(record) {
    Modal.confirm({
      title: '删除 Webhook',
      content: `确认删除 Webhook「${record.name}」？`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        await requestWrapper(cloudEventAPI.deleteWebhook.bind(null, { orgId, id: record.id }), { autoSuccess: true });
        fetchWebhooks();
        fetchQueueSummary();
      }
    });
  }
  async function testWebhook(record) {
    await requestWrapper(cloudEventAPI.testWebhook.bind(null, { orgId, id: record.id }), { autoSuccess: true });
    fetchWebhooks();
    fetchQueueSummary();
  }
  async function toggleWebhookStatus(record) {
    const nextStatus = record.status === 'enable' ? 'disable' : 'enable';
    await requestWrapper(cloudEventAPI.updateWebhook.bind(null, {
      orgId,
      id: record.id,
      data: {
        name: record.name,
        description: record.description,
        targetUrl: record.targetUrl,
        eventTypes: record.eventTypes || [],
        sources: record.sources || [],
        status: nextStatus,
        timeoutSeconds: record.timeoutSeconds || 5,
        maxRetries: record.maxRetries || 3,
        retryInterval: record.retryInterval || 60,
        retryJitterPercent: record.retryJitterPercent || 0,
        maxRetryDuration: record.maxRetryDuration || 0
      }
    }), { autoSuccess: true });
    fetchWebhooks();
    fetchQueueSummary();
  }
  async function retryDueWebhooks() {
    await requestWrapper(cloudEventAPI.retryDueWebhooks.bind(null, { orgId }), { autoSuccess: true });
    fetchWebhooks();
    fetchQueueSummary();
    if (deliveryDrawer.record) {
      fetchDeliveries(deliveryDrawer.record);
    }
    if (deadLetterDrawer.visible) {
      fetchDeadLetters();
    }
  }
  async function retryDelivery(record) {
    const webhook = deliveryDrawer.record;
    if (!webhook) {
      return;
    }
    await requestWrapper(cloudEventAPI.retryWebhookDelivery.bind(null, {
      orgId,
      id: webhook.id,
      deliveryId: record.id
    }), { autoSuccess: true });
    fetchWebhooks();
    fetchQueueSummary();
    fetchDeliveries(webhook);
  }
  async function replayDeadLetter(record) {
    await requestWrapper(cloudEventAPI.replayWebhookDeadLetter.bind(null, {
      orgId,
      deadLetterId: record.id
    }), { autoSuccess: true });
    fetchWebhooks();
    fetchQueueSummary();
    fetchDeadLetters();
  }
  function ignoreDeadLetter(record) {
    Modal.confirm({
      title: '忽略死信',
      content: `确认忽略死信「${record.id}」？`,
      okText: '忽略',
      cancelText: '取消',
      onOk: async () => {
        await requestWrapper(cloudEventAPI.ignoreWebhookDeadLetter.bind(null, {
          orgId,
          deadLetterId: record.id,
          data: { note: '页面忽略' }
        }), { autoSuccess: true });
        fetchQueueSummary();
        fetchDeadLetters();
      }
    });
  }

  return (
    <Layout
      extraHeader={<PageHeader title='事件中心' breadcrumb={true}/>}
    >
      <div className='idcos-card'>
        <div className={styles.webhookPanel}>
          <div className={styles.webhookHeader}>
            <Space size={8}>
              <Text strong={true}>Webhook</Text>
              <Tag>{webhookData.total || 0} 个配置</Tag>
            </Space>
            <Space size={8}>
              <Button icon={<InboxOutlined/>} onClick={openDeadLetterDrawer}>死信</Button>
              <Button icon={<SyncOutlined/>} onClick={retryDueWebhooks}>重试到期</Button>
              <Button type='primary' icon={<PlusOutlined/>} onClick={() => openWebhookModal()}>新建 Webhook</Button>
            </Space>
          </div>
          <div className={styles.queueMetrics}>
            <QueueMetric label='排队' value={queueSummary.queued} tone={queueSummary.queued ? 'warningTone' : ''}/>
            <QueueMetric label='已到期' value={queueSummary.due} tone={queueSummary.due ? 'dangerTone' : ''}/>
            <QueueMetric label='未来重试' value={queueSummary.future}/>
            <QueueMetric label='死信' value={queueSummary.deadLetter} tone={queueSummary.deadLetter ? 'dangerTone' : ''}/>
            <QueueMetric label='已成功' value={queueSummary.success}/>
            <QueueMetric label='已失败' value={queueSummary.failed} tone={queueSummary.failed ? 'warningTone' : ''}/>
            <QueueMetric label='自动重试' value={queueSummary.auto}/>
            <QueueMetric label='已消费' value={queueSummary.retried}/>
            <QueueMetric label='投递成功率' value={formatPercent(queueSummary.successRate)}/>
            <QueueMetric label='队列成功率' value={formatPercent(queueSummary.queueSuccessRate)}/>
            <QueueMetric label='分片数' value={queueSummary.shardTotal}/>
            <QueueMetric label='最近消费' value={renderTime(queueSummary.lastConsumedAt)} compact={true}/>
          </div>
          {queueSummary.shards && queueSummary.shards.length ? (
            <Table
              className={styles.shardTable}
              rowKey='shardIndex'
              size='small'
              columns={shardColumns}
              dataSource={queueSummary.shards}
              pagination={false}
              scroll={{ x: 'min-content' }}
            />
          ) : null}
          {queueSummaryLoading ? <div className={styles.queueLoading}>队列状态刷新中</div> : null}
          <Table
            rowKey='id'
            size='small'
            columns={webhookColumns}
            dataSource={webhookList}
            loading={webhookLoading}
            scroll={{ x: 'min-content' }}
            pagination={false}
            locale={{ emptyText: <Empty description='暂无 Webhook'/> }}
          />
        </div>
        <div className={styles.toolbar}>
          <Space className={styles.filterBar} size={[8, 8]} wrap={true}>
            <InputSearch
              className={styles.keywordSearch}
              allowClear={true}
              placeholder='搜索事件、资源或载荷'
              onSearch={(value) => resetQuery({ q: value })}
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
              placeholder='级别'
              style={{ width: 120 }}
              onChange={(level) => resetQuery({ level })}
            >
              {Object.entries(levelMap).map(([value, item]) => (
                <Option key={value} value={value}>{item.label}</Option>
              ))}
            </Select>
          </Space>
          <Button icon={<ReloadOutlined/>} onClick={refreshAll}>刷新</Button>
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
        title='事件详情'
        visible={drawer.visible}
        width={860}
        destroyOnClose={true}
        onClose={() => setDrawer({ visible: false, id: '' })}
      >
        <EventDetail detail={detail} loading={loading}/>
      </Drawer>
      <Modal
        title={deliveryDrawer.record ? `Webhook 投递记录 - ${deliveryDrawer.record.name}` : 'Webhook 投递记录'}
        visible={deliveryDrawer.visible}
        width={1100}
        destroyOnClose={true}
        footer={null}
        onCancel={closeDeliveryDrawer}
      >
        <Table
          rowKey='id'
          size='small'
          columns={deliveryColumns}
          dataSource={deliveryList}
          loading={deliveryLoading}
          scroll={{ x: 1230 }}
          pagination={false}
          locale={{ emptyText: <Empty description='暂无投递记录'/> }}
        />
      </Modal>
      <Modal
        title='投递详情'
        visible={!!deliveryDetail}
        width={860}
        destroyOnClose={true}
        footer={null}
        onCancel={() => setDeliveryDetail(null)}
      >
        <DeliveryDetail detail={deliveryDetail || {}}/>
      </Modal>
      <Modal
        title='Webhook 死信'
        visible={deadLetterDrawer.visible}
        width={1120}
        destroyOnClose={true}
        footer={null}
        onCancel={closeDeadLetterDrawer}
      >
        <Table
          rowKey='id'
          size='small'
          columns={deadLetterColumns}
          dataSource={deadLetterList}
          loading={deadLetterLoading}
          scroll={{ x: 1300 }}
          pagination={false}
          locale={{ emptyText: <Empty description='暂无死信'/> }}
        />
      </Modal>
      <Modal
        title={webhookModal.record ? '编辑 Webhook' : '新建 Webhook'}
        visible={webhookModal.visible}
        width={760}
        destroyOnClose={true}
        confirmLoading={webhookSaving}
        onCancel={closeWebhookModal}
        onOk={submitWebhook}
      >
        <Form form={webhookForm} layout='vertical'>
          <div className={styles.webhookFormGrid}>
            <Form.Item name='name' label='名称' rules={[{ required: true, message: '请输入名称' }]}>
              <Input placeholder='请输入 Webhook 名称'/>
            </Form.Item>
            <Form.Item name='status' label='状态'>
              <Select>
                <Option value='enable'>启用</Option>
                <Option value='disable'>停用</Option>
              </Select>
            </Form.Item>
            <Form.Item name='targetUrl' label='目标地址' rules={[{ required: true, message: '请输入目标地址' }]}>
              <Input placeholder='https://example.com/webhook'/>
            </Form.Item>
            <Form.Item name='timeoutSeconds' label='超时时间'>
              <InputNumber min={1} max={30} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='maxRetries' label='最大重试次数'>
              <InputNumber min={1} max={10} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='retryInterval' label='重试间隔'>
              <InputNumber min={1} max={3600} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='retryJitterPercent' label='重试抖动(%)'>
              <InputNumber min={0} max={100} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='maxRetryDuration' label='最大重试窗口(秒)'>
              <InputNumber min={0} max={604800} style={{ width: '100%' }}/>
            </Form.Item>
            <Form.Item name='eventTypes' label='事件类型'>
              <Select mode='tags' placeholder='留空表示全部事件'>
                {Object.entries(eventTypeMap).map(([value, label]) => (
                  <Option key={value} value={value}>{label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name='sources' label='事件来源'>
              <Select mode='multiple' placeholder='留空表示全部来源'>
                {Object.entries(sourceMap).map(([value, label]) => (
                  <Option key={value} value={value}>{label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name='secret' label='签名密钥'>
              <Input.Password placeholder={webhookModal.record ? '留空则不修改' : '可选'}/>
            </Form.Item>
          </div>
          <Form.Item name='description' label='说明'>
            <Input.TextArea rows={3} placeholder='可填写目标系统、负责人或投递约定'/>
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title={secretModal.record ? `轮换密钥 - ${secretModal.record.name}` : '轮换密钥'}
        visible={secretModal.visible}
        width={520}
        destroyOnClose={true}
        confirmLoading={secretSaving}
        onCancel={closeSecretModal}
        onOk={submitSecretRotation}
      >
        <Form form={secretForm} layout='vertical'>
          <Form.Item name='secret' label='新签名密钥' rules={[{ required: true, message: '请输入新签名密钥' }]}>
            <Input.Password placeholder='请输入新签名密钥'/>
          </Form.Item>
          <Form.Item name='gracePeriodSeconds' label='旧密钥宽限期(秒)'>
            <InputNumber min={0} max={7776000} style={{ width: '100%' }}/>
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  );
};

export default CloudEventPage;
