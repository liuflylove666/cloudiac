import React, { useMemo, useState } from 'react';
import {
  Alert,
  Button,
  DatePicker,
  Descriptions,
  Drawer,
  Empty,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Typography
} from 'antd';
import { CheckCircleOutlined, ExceptionOutlined, ReloadOutlined, SyncOutlined } from '@ant-design/icons';
import { useRequest } from 'ahooks';
import moment from 'moment';
import PageHeader from 'components/pageHeader';
import Layout from 'components/common/layout';
import { requestWrapper } from 'utils/request';
import cloudRiskAPI from 'services/cloud-risk';
import styles from './styles.less';

const { Option } = Select;
const { Search: InputSearch, TextArea } = Input;
const { Text } = Typography;

const statusMap = {
  open: { label: '待处理', color: 'error' },
  in_progress: { label: '处理中', color: 'processing' },
  suppressed: { label: '已例外', color: 'warning' },
  resolved: { label: '已解决', color: 'success' }
};

const riskMap = {
  low: { label: '低', color: 'success' },
  medium: { label: '中', color: 'warning' },
  high: { label: '高', color: 'error' },
  critical: { label: '严重', color: 'error' }
};

const sourceMap = {
  cloud_config: '云配置',
  cmdb: 'CMDB',
  drift: '漂移',
  policy: '策略'
};

const ruleMap = {
  unmanaged_cloud_asset: '未纳管资产',
  unowned_cloud_asset: '缺少负责人',
  public_ingress_security_rule: '公网入方向规则',
  public_egress_security_rule: '公网出方向规则',
  terraform_drift_detected: 'IaC 漂移',
  cmdb_compliance_risk_high: '高风险合规标记',
  cmdb_compliance_risk_critical: '严重合规标记'
};

const renderTime = (value) => !value || String(value).indexOf('0001-01-01') === 0 ? '-' : moment(value).format('YYYY-MM-DD HH:mm:ss');
const statusTag = (value) => {
  const item = statusMap[value] || { label: value || '-', color: 'default' };
  return <Tag color={item.color}>{item.label}</Tag>;
};
const riskTag = (value) => {
  const item = riskMap[value] || { label: value || '-', color: 'default' };
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

const Metric = ({ label, value, tone }) => (
  <div className={`${styles.metric} ${tone ? styles[tone] : ''}`}>
    <div className={styles.metricValue}>{value || 0}</div>
    <div className={styles.metricLabel}>{label}</div>
  </div>
);

const RiskDetail = ({ detail = {}, loading, onMarkProgress, onResolve, onReopen, onSuppress, updating }) => {
  if (loading) {
    return <Empty description='加载中'/>;
  }
  if (!detail.id) {
    return <Empty description='请选择风险'/>;
  }
  const canMarkProgress = detail.status === 'open';
  const canResolve = detail.status === 'open' || detail.status === 'in_progress' || detail.status === 'suppressed';
  const canReopen = detail.status === 'resolved' || detail.status === 'suppressed';
  return (
    <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
      <Descriptions size='small' bordered={true} column={2}>
        <Descriptions.Item label='风险 ID'>{detail.id || '-'}</Descriptions.Item>
        <Descriptions.Item label='状态'>{statusTag(detail.status)}</Descriptions.Item>
        <Descriptions.Item label='等级'>{riskTag(detail.riskLevel)}</Descriptions.Item>
        <Descriptions.Item label='来源'>{sourceMap[detail.source] || detail.source || '-'}</Descriptions.Item>
        <Descriptions.Item label='规则'>{ruleMap[detail.ruleKey] || detail.ruleName || detail.ruleKey || '-'}</Descriptions.Item>
        <Descriptions.Item label='云厂商'>{detail.provider || '-'}</Descriptions.Item>
        <Descriptions.Item label='账号'>{detail.accountId || detail.cloudAccountId || '-'}</Descriptions.Item>
        <Descriptions.Item label='区域'>{detail.region || '-'}</Descriptions.Item>
        <Descriptions.Item label='资源'>{detail.resourceName || detail.assetName || detail.resourceId || '-'}</Descriptions.Item>
        <Descriptions.Item label='资源类型'>{detail.resourceType || '-'}</Descriptions.Item>
        <Descriptions.Item label='项目'>{detail.projectName || detail.projectId || '-'}</Descriptions.Item>
        <Descriptions.Item label='环境'>{detail.envName || detail.envId || '-'}</Descriptions.Item>
        <Descriptions.Item label='首次发现'>{renderTime(detail.firstSeenAt)}</Descriptions.Item>
        <Descriptions.Item label='最近发现'>{renderTime(detail.lastSeenAt)}</Descriptions.Item>
        <Descriptions.Item label='解决时间'>{renderTime(detail.resolvedAt)}</Descriptions.Item>
        <Descriptions.Item label='例外到期'>{renderTime(detail.suppressedUntil)}</Descriptions.Item>
      </Descriptions>
      {detail.status === 'suppressed' && (
        <Alert
          showIcon={true}
          type='warning'
          message='风险已例外'
          description={detail.suppressionReason || '未填写例外原因'}
        />
      )}
      <Space size={8} wrap={true}>
        {canMarkProgress && (
          <Button icon={<SyncOutlined/>} loading={updating} onClick={onMarkProgress}>标记处理中</Button>
        )}
        {canResolve && (
          <Popconfirm title='确认将该风险标记为已解决？' onConfirm={onResolve}>
            <Button type='primary' icon={<CheckCircleOutlined/>} loading={updating}>标记已解决</Button>
          </Popconfirm>
        )}
        {canReopen && (
          <Button icon={<ReloadOutlined/>} loading={updating} onClick={onReopen}>重新打开</Button>
        )}
        <Button icon={<ExceptionOutlined/>} loading={updating} onClick={onSuppress}>风险例外</Button>
      </Space>
      <div className={styles.detailSection}>
        <Text strong={true}>修复建议</Text>
        <div>{detail.recommendation || '-'}</div>
      </div>
      <div className={styles.detailSection}>
        <Text strong={true}>证据</Text>
        <JsonBlock value={detail.evidence}/>
      </div>
    </Space>
  );
};

const CloudRiskPage = ({ match }) => {
  const { orgId } = match.params || {};
  const [ query, setQuery ] = useState({
    currentPage: 1,
    pageSize: 10,
    status: 'open'
  });
  const [ drawer, setDrawer ] = useState({
    visible: false,
    id: ''
  });
  const [ suppressModal, setSuppressModal ] = useState({
    visible: false,
    record: null,
    until: null,
    reason: ''
  });

  const {
    loading,
    data = {},
    run: fetchList
  } = useRequest(
    () => requestWrapper(cloudRiskAPI.list.bind(null, { orgId, ...query })),
    {
      refreshDeps: [ query, orgId ]
    }
  );

  const list = data.list || [];
  const summary = data.summary || {};
  const detail = useMemo(() => list.find((item) => item.id === drawer.id) || {}, [ list, drawer.id ]);

  const {
    loading: updating,
    run: updateStatus
  } = useRequest(
    ({ id, status, comment }) => requestWrapper(cloudRiskAPI.updateStatus.bind(null, { orgId, id, status, comment }), { autoSuccess: true }),
    {
      manual: true,
      onSuccess: () => {
        fetchList();
      }
    }
  );

  const {
    loading: suppressing,
    run: suppressRisk
  } = useRequest(
    ({ id, suppressedUntil, reason }) => requestWrapper(cloudRiskAPI.suppress.bind(null, { orgId, id, suppressedUntil, reason }), {
      autoSuccess: true,
      successMessage: '风险例外已保存'
    }),
    {
      manual: true,
      onSuccess: () => {
        setSuppressModal({ visible: false, record: null, until: null, reason: '' });
        fetchList();
      }
    }
  );

  const openDetail = (record) => {
    setDrawer({ visible: true, id: record.id });
  };

  const openSuppress = (record) => {
    setSuppressModal({
      visible: true,
      record,
      until: moment().add(7, 'days').endOf('day'),
      reason: ''
    });
  };

  const columns = useMemo(() => [
    {
      title: '风险',
      dataIndex: 'ruleName',
      width: 220,
      ellipsis: true,
      render: (text, record) => <a onClick={() => openDetail(record)}>{ruleMap[record.ruleKey] || text || record.ruleKey || record.id}</a>
    },
    {
      title: '等级',
      dataIndex: 'riskLevel',
      width: 90,
      render: riskTag
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: statusTag
    },
    {
      title: '来源',
      dataIndex: 'source',
      width: 100,
      render: (text) => sourceMap[text] || text || '-'
    },
    {
      title: '资源',
      dataIndex: 'resourceName',
      width: 220,
      ellipsis: true,
      render: (text, record) => text || record.assetName || record.resourceId || '-'
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
      title: '最近发现',
      dataIndex: 'lastSeenAt',
      width: 180,
      render: renderTime
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <a onClick={() => updateStatus({ id: record.id, status: 'in_progress', comment: '页面标记处理中' })}>处理中</a>
          <a onClick={() => openSuppress(record)}>例外</a>
          <Popconfirm title='确认将该风险标记为已解决？' onConfirm={() => updateStatus({ id: record.id, status: 'resolved', comment: '页面标记已解决' })}>
            <a>解决</a>
          </Popconfirm>
        </Space>
      )
    }
  ], [ updateStatus ]);

  const resetQuery = (patch) => setQuery({ ...query, ...patch, currentPage: 1 });
  const activeCount = (summary.open || 0) + (summary.inProgress || 0);
  return (
    <Layout
      extraHeader={<PageHeader title='风险合规' breadcrumb={true}/>}
    >
      <div className='idcos-card'>
        <div className={styles.metrics}>
          <Metric label='未处理风险' value={activeCount} tone={activeCount ? 'dangerTone' : ''}/>
          <Metric label='严重' value={summary.activeCritical} tone={summary.activeCritical ? 'dangerTone' : ''}/>
          <Metric label='高危' value={summary.activeHigh} tone={summary.activeHigh ? 'warningTone' : ''}/>
          <Metric label='公网暴露' value={summary.publicExposure} tone={summary.publicExposure ? 'warningTone' : ''}/>
          <Metric label='未纳管' value={summary.unmanagedAssets}/>
          <Metric label='已例外' value={summary.suppressed}/>
        </div>
        <div className={styles.toolbar}>
          <Space className={styles.filterBar} size={[8, 8]} wrap={true}>
            <InputSearch
              className={styles.keywordSearch}
              allowClear={true}
              placeholder='搜索风险、资源或证据'
              onSearch={(value) => resetQuery({ q: value })}
            />
            <Select
              allowClear={true}
              placeholder='状态'
              value={query.status}
              style={{ width: 130 }}
              onChange={(status) => resetQuery({ status })}
            >
              {Object.entries(statusMap).map(([value, item]) => (
                <Option key={value} value={value}>{item.label}</Option>
              ))}
            </Select>
            <Select
              allowClear={true}
              placeholder='等级'
              style={{ width: 120 }}
              onChange={(riskLevel) => resetQuery({ riskLevel })}
            >
              {Object.entries(riskMap).map(([value, item]) => (
                <Option key={value} value={value}>{item.label}</Option>
              ))}
            </Select>
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
          </Space>
          <Button icon={<ReloadOutlined/>} onClick={fetchList}>刷新</Button>
        </div>
        <Table
          rowKey='id'
          columns={columns}
          dataSource={list}
          loading={loading || updating}
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
        title='风险详情'
        visible={drawer.visible}
        width={860}
        destroyOnClose={true}
        onClose={() => setDrawer({ visible: false, id: '' })}
      >
        <RiskDetail
          detail={detail}
          loading={loading}
          updating={updating}
          onMarkProgress={() => updateStatus({ id: detail.id, status: 'in_progress', comment: '页面标记处理中' })}
          onResolve={() => updateStatus({ id: detail.id, status: 'resolved', comment: '页面标记已解决' })}
          onReopen={() => updateStatus({ id: detail.id, status: 'open', comment: '页面重新打开风险' })}
          onSuppress={() => openSuppress(detail)}
        />
      </Drawer>
      <Modal
        title='风险例外'
        visible={suppressModal.visible}
        confirmLoading={suppressing}
        onCancel={() => setSuppressModal({ visible: false, record: null, until: null, reason: '' })}
        onOk={() => {
          if (!suppressModal.record || !suppressModal.until) {
            return;
          }
          suppressRisk({
            id: suppressModal.record.id,
            suppressedUntil: suppressModal.until.toISOString(),
            reason: suppressModal.reason
          });
        }}
      >
        <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
          <DatePicker
            showTime={true}
            style={{ width: '100%' }}
            value={suppressModal.until}
            onChange={(until) => setSuppressModal({ ...suppressModal, until })}
          />
          <TextArea
            rows={4}
            maxLength={255}
            value={suppressModal.reason}
            placeholder='填写例外原因'
            onChange={(event) => setSuppressModal({ ...suppressModal, reason: event.target.value })}
          />
        </Space>
      </Modal>
    </Layout>
  );
};

export default CloudRiskPage;
