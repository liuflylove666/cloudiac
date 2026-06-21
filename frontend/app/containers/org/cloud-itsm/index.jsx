import React, { useMemo, useState } from 'react';
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
  Table,
  Tabs,
  Tag,
  Typography
} from 'antd';
import { CheckCircleOutlined, DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, SafetyCertificateOutlined, SyncOutlined, ToolOutlined } from '@ant-design/icons';
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
  operations: '运维操作'
};

const automationModeMap = {
  auto: '自动处理',
  approval: '审批后执行',
  provider: '云端执行',
  provider_approval: '审批后云端执行',
  itsm: 'ITSM 工单'
};

const renderTime = (value) => !value || String(value).indexOf('0001-01-01') === 0 ? '-' : moment(value).format('YYYY-MM-DD HH:mm:ss');
const percent = (value) => Math.max(0, Math.min(100, Number(value || 0)));
const percentText = (value) => `${percent(value).toFixed(1)}%`;
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
const metadataText = (value) => JSON.stringify(value || {}, null, 2);
const parseMetadata = (value) => {
  if (!value || String(value).trim() === '') {
    return {};
  }
  if (typeof value === 'object') {
    return value;
  }
  return JSON.parse(value);
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
  const [ configQuery, setConfigQuery ] = useState({
    currentPage: 1,
    pageSize: 10
  });
  const [ ticketQuery, setTicketQuery ] = useState({
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
      onSuccess: () => fetchConfigs()
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
      }
    }
  );

  const metrics = overviewData.metrics || {};
  const catalog = overviewData.catalog || [];
  const selfServiceCatalog = catalog.filter((item) => item.operationType === 'self_service' && item.available);
  const configs = configsData.list || [];
  const enabledConfigs = configs.filter((item) => item.status === 'enable');
  const tickets = ticketsData.list || [];
  const detail = useMemo(() => tickets.find((item) => item.id === drawer.id) || {}, [ tickets, drawer.id ]);

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
      params: metadataText({})
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
      ellipsis: true,
      render: (text) => text || '-'
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
          description={`目标 ${percentText(metrics.ticketAutomationTarget)}，自动 ${metrics.automatedTicketTotal || 0}/${metrics.ticketTotal || 0} 单`}
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
        <Tabs.TabPane tab='自助目录' key='catalog'>
          <Table
            rowKey='key'
            columns={catalogColumns}
            dataSource={catalog}
            pagination={false}
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
            <TextArea rows={4} placeholder='可填写 createTicketPath / createTicketUrl / browseTicketPath 等 JSON'/>
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
                values: {
                  ...values,
                  params: parseMetadata(values.params)
                }
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
              rules={enabledConfigs.length > 0 ? [ { required: true, message: '请选择连接器' } ] : []}
            >
              <Select placeholder='请选择连接器'>
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
          <Form.Item name='title' label='标题'>
            <Input placeholder='留空则使用申请类型名称'/>
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
