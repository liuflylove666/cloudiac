import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Descriptions,
  Drawer,
  Form,
  Input,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag
} from 'antd';
import { CloudSyncOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useRequest } from 'ahooks';
import moment from 'moment';
import PageHeader from 'components/pageHeader';
import Layout from 'components/common/layout';
import { requestWrapper } from 'utils/request';
import cloudAccountAPI from 'services/cloud-account';
import styles from './styles.less';

const { Option } = Select;
const { TextArea } = Input;
const { Search: InputSearch } = Input;

const providerOptions = [
  { label: 'AWS', value: 'aws' },
  { label: 'OCI', value: 'oci' },
  { label: 'AliCloud', value: 'alicloud' }
];

const statusMap = {
  enable: { label: '启用', color: 'success' },
  disable: { label: '禁用', color: 'default' }
};

const validationStatusMap = {
  pending: { label: '待验证', color: 'default' },
  valid: { label: '有效', color: 'success' },
  invalid: { label: '异常', color: 'error' }
};

const permissionStatusMap = {
  pass: { label: '通过', color: 'success' },
  warn: { label: '提醒', color: 'warning' },
  fail: { label: '失败', color: 'error' }
};

const providerCredentialTemplates = {
  aws: [
    { id: 'AWS_ACCESS_KEY_ID', key: 'AWS_ACCESS_KEY_ID', value: '', isSecret: false },
    { id: 'AWS_SECRET_ACCESS_KEY', key: 'AWS_SECRET_ACCESS_KEY', value: '', isSecret: true },
    { id: 'AWS_SESSION_TOKEN', key: 'AWS_SESSION_TOKEN', value: '', isSecret: true },
    { id: 'AWS_ACCOUNT_ID', key: 'AWS_ACCOUNT_ID', value: '', isSecret: false }
  ],
  oci: [
    { id: 'OCI_TENANCY_OCID', key: 'OCI_TENANCY_OCID', value: '', isSecret: false },
    { id: 'OCI_USER_OCID', key: 'OCI_USER_OCID', value: '', isSecret: false },
    { id: 'OCI_FINGERPRINT', key: 'OCI_FINGERPRINT', value: '', isSecret: false },
    { id: 'OCI_PRIVATE_KEY', key: 'OCI_PRIVATE_KEY', value: '', isSecret: true },
    { id: 'OCI_PRIVATE_KEY_PASSPHRASE', key: 'OCI_PRIVATE_KEY_PASSPHRASE', value: '', isSecret: true }
  ],
  alicloud: [
    { id: 'ALICLOUD_ACCESS_KEY', key: 'ALICLOUD_ACCESS_KEY', value: '', isSecret: false },
    { id: 'ALICLOUD_SECRET_KEY', key: 'ALICLOUD_SECRET_KEY', value: '', isSecret: true },
    { id: 'ALICLOUD_ACCOUNT_ID', key: 'ALICLOUD_ACCOUNT_ID', value: '', isSecret: false }
  ]
};

const splitList = (value) => {
  if (Array.isArray(value)) {
    return value;
  }
  return String(value || '')
    .split(/[,;\n\s]+/)
    .map((item) => item.trim())
    .filter(Boolean);
};

const joinList = (value) => Array.isArray(value) && value.length ? value.join(', ') : '-';
const renderTime = (value) => !value || String(value).indexOf('0001-01-01') === 0 ? '-' : moment(value).format('YYYY-MM-DD HH:mm:ss');

const statusTag = (value) => {
  const item = statusMap[value] || { label: value || '-', color: 'default' };
  return <Tag color={item.color}>{item.label}</Tag>;
};

const validationTag = (value) => {
  const item = validationStatusMap[value] || { label: value || '-', color: 'default' };
  return <Tag color={item.color}>{item.label}</Tag>;
};

const normalizeCredentialRows = (provider, credentials) => {
  const rows = credentials && credentials.length ? credentials : providerCredentialTemplates[provider] || [];
  return rows.map((item) => ({
    id: item.id || item.key,
    key: item.key,
    value: item.value || '',
    isSecret: !!item.isSecret
  }));
};

const AccountDrawer = ({ visible, mode, record, onClose, onSubmit, submitting }) => {
  const [ form ] = Form.useForm();
  const [ provider, setProvider ] = useState('aws');

  useEffect(() => {
    if (!visible) {
      return;
    }
    const initialProvider = record.provider || 'aws';
    setProvider(initialProvider);
    form.setFieldsValue({
      name: record.name,
      description: record.description,
      provider: initialProvider,
      accountId: record.accountId,
      tenantId: record.tenantId,
      regionsText: record.regions && record.regions.length ? joinList(record.regions) : '',
      runnerTagsText: record.runnerTags && record.runnerTags.length ? joinList(record.runnerTags) : '',
      status: (record.status || 'enable') === 'enable',
      credentials: normalizeCredentialRows(initialProvider, record.credentials)
    });
  }, [ visible, record ]);

  useEffect(() => {
    if (!visible || mode !== 'add') {
      return;
    }
    form.setFieldsValue({
      credentials: normalizeCredentialRows(provider || 'aws', [])
    });
  }, [ provider ]);

  const submit = async () => {
    const values = await form.validateFields();
    onSubmit({
      name: values.name,
      description: values.description,
      provider: values.provider,
      accountId: values.accountId,
      tenantId: values.tenantId,
      regions: splitList(values.regionsText),
      runnerTags: splitList(values.runnerTagsText),
      status: values.status ? 'enable' : 'disable',
      credentials: (values.credentials || []).map((item) => ({
        id: item.id || item.key,
        key: item.key,
        value: item.value || '',
        isSecret: !!item.isSecret
      }))
    });
  };

  return (
    <Drawer
      title={mode === 'add' ? '创建云账号' : '编辑云账号'}
      width={720}
      visible={visible}
      onClose={onClose}
      destroyOnClose={true}
      footer={(
        <div className={styles.drawerFooter}>
          <Button onClick={onClose}>取消</Button>
          <Button type='primary' loading={submitting} onClick={submit}>保存</Button>
        </div>
      )}
    >
      <Form
        form={form}
        layout='vertical'
        onValuesChange={(changedValues) => {
          if (changedValues.provider) {
            setProvider(changedValues.provider);
          }
        }}
      >
        <div className={styles.formGrid}>
          <Form.Item name='name' label='账号名称' rules={[{ required: true, message: '请输入账号名称' }]}>
            <Input placeholder='例如：生产 AWS 账号'/>
          </Form.Item>
          <Form.Item name='provider' label='云厂商' rules={[{ required: true, message: '请选择云厂商' }]}>
            <Select>
              {providerOptions.map((item) => <Option key={item.value} value={item.value}>{item.label}</Option>)}
            </Select>
          </Form.Item>
          <Form.Item name='accountId' label='云账号 ID'>
            <Input placeholder='可留空，由凭证字段推断'/>
          </Form.Item>
          <Form.Item name='tenantId' label='租户/订阅 ID'>
            <Input placeholder='OCI tenancy 或 Azure subscription 等'/>
          </Form.Item>
        </div>
        <Form.Item name='regionsText' label='启用区域'>
          <Input placeholder='多个区域用逗号、空格或换行分隔'/>
        </Form.Item>
        <Form.Item name='runnerTagsText' label='Runner 标签'>
          <Input placeholder='多个标签用逗号、空格或换行分隔'/>
        </Form.Item>
        <Form.Item name='description' label='描述'>
          <TextArea rows={3}/>
        </Form.Item>
        <Form.Item name='status' label='状态' valuePropName='checked'>
          <Switch checkedChildren='启用' unCheckedChildren='禁用'/>
        </Form.Item>
        <Form.List name='credentials'>
          {(fields) => (
            <div className={styles.credentials}>
              <div className={styles.sectionTitle}>凭证字段</div>
              {fields.map((field) => (
                <div className={styles.credentialRow} key={field.key}>
                  <Form.Item {...field} name={[field.name, 'id']} hidden={true}>
                    <Input/>
                  </Form.Item>
                  <Form.Item {...field} name={[field.name, 'isSecret']} valuePropName='checked' hidden={true}>
                    <Switch/>
                  </Form.Item>
                  <Form.Item {...field} name={[field.name, 'key']} className={styles.credentialKey}>
                    <Input disabled={true}/>
                  </Form.Item>
                  <Form.Item {...field} name={[field.name, 'value']} className={styles.credentialValue}>
                    <Input.Password placeholder='敏感字段留空将保留原值'/>
                  </Form.Item>
                </div>
              ))}
            </div>
          )}
        </Form.List>
      </Form>
    </Drawer>
  );
};

const RegionPermissionDrawer = ({ visible, record, orgId, onClose, onUpdated }) => {
  const [ form ] = Form.useForm();

  const {
    loading: regionsLoading,
    data: regionsData,
    run: fetchRegions
  } = useRequest(
    () => requestWrapper(cloudAccountAPI.regions.bind(null, { orgId, id: record.id })),
    {
      manual: true,
      onSuccess: (data) => {
        form.setFieldsValue({
          regionsText: (data.regionNames || []).join(', ')
        });
      }
    }
  );

  const {
    loading: permissionsLoading,
    data: permissionsData,
    run: fetchPermissions
  } = useRequest(
    () => requestWrapper(cloudAccountAPI.permissions.bind(null, { orgId, id: record.id })),
    {
      manual: true
    }
  );

  const {
    loading: savingRegions,
    run: saveRegions
  } = useRequest(
    (regions) => requestWrapper(
      cloudAccountAPI.updateRegions.bind(null, { orgId, id: record.id, regions }),
      { autoSuccess: true }
    ),
    {
      manual: true,
      onSuccess: () => {
        fetchRegions();
        fetchPermissions();
        onUpdated();
      }
    }
  );

  useEffect(() => {
    if (!visible || !record.id) {
      return;
    }
    fetchRegions();
    fetchPermissions();
  }, [ visible, record.id ]);

  const saveRegionConfig = async () => {
    const values = await form.validateFields();
    saveRegions(splitList(values.regionsText));
  };

  const permissionColumns = [
    {
      title: '检查项',
      dataIndex: 'name',
      width: 140
    },
    {
      title: '资源',
      dataIndex: 'resource',
      width: 140
    },
    {
      title: '动作',
      dataIndex: 'action',
      width: 100
    },
    {
      title: '结果',
      dataIndex: 'status',
      width: 90,
      render: (value) => {
        const item = permissionStatusMap[value] || { label: value || '-', color: 'default' };
        return <Tag color={item.color}>{item.label}</Tag>;
      }
    },
    {
      title: '说明',
      dataIndex: 'message'
    }
  ];

  const regionColumns = [
    {
      title: '区域',
      dataIndex: 'name'
    },
    {
      title: '默认',
      dataIndex: 'default',
      width: 90,
      render: (value) => value ? <Tag color='processing'>默认</Tag> : '-'
    },
    {
      title: '来源',
      dataIndex: 'source',
      width: 110,
      render: (value) => value === 'configured' ? '手动配置' : '自动推断'
    }
  ];

  const regionRows = (regionsData || {}).regions || [];
  const permissionRows = (permissionsData || {}).permissions || [];
  const supportedTypes = (permissionsData || {}).supportedAssetTypes || [];
  const missingKeys = (permissionsData || {}).missingCredentialKeys || [];

  return (
    <Drawer
      title='区域与权限'
      width={760}
      visible={visible}
      onClose={onClose}
      destroyOnClose={true}
    >
      <Descriptions size='small' column={2} className={styles.summary}>
        <Descriptions.Item label='账号'>{record.name || '-'}</Descriptions.Item>
        <Descriptions.Item label='云厂商'>{(providerOptions.find((item) => item.value === record.provider) || {}).label || record.provider || '-'}</Descriptions.Item>
        <Descriptions.Item label='账号 ID'>{record.accountId || '-'}</Descriptions.Item>
        <Descriptions.Item label='验证状态'>{validationTag(record.validationStatus)}</Descriptions.Item>
      </Descriptions>
      <Alert
        type='info'
        showIcon={true}
        style={{ marginBottom: 16 }}
        message='当前权限检查为本地只读预检查，重点校验账号状态、凭证字段、区域范围和已支持的资产采集能力。真实云 API 权限校验将在 provider adapter 中继续增强。'
      />
      <div className={styles.sectionTitle}>启用区域</div>
      <Form form={form} layout='vertical'>
        <Form.Item
          name='regionsText'
          label='区域列表'
          extra='多个区域用逗号、空格或换行分隔。清空后会按凭证或云厂商默认规则重新推断。'
        >
          <TextArea rows={3} placeholder='例如 us-east-1, us-west-2'/>
        </Form.Item>
        <Space style={{ marginBottom: 16 }}>
          <Button type='primary' loading={savingRegions} onClick={saveRegionConfig}>保存区域</Button>
          <Button loading={regionsLoading || permissionsLoading} onClick={() => {
            fetchRegions();
            fetchPermissions();
          }}>刷新</Button>
        </Space>
      </Form>
      <Table
        rowKey='name'
        size='small'
        columns={regionColumns}
        dataSource={regionRows}
        loading={regionsLoading}
        pagination={false}
      />
      <div className={styles.sectionTitle}>权限验证结果</div>
      {!!missingKeys.length && (
        <Alert
          type='warning'
          showIcon={true}
          style={{ marginBottom: 12 }}
          message={`缺少凭证字段：${missingKeys.join(', ')}`}
        />
      )}
      <Table
        rowKey='key'
        size='small'
        columns={permissionColumns}
        dataSource={permissionRows}
        loading={permissionsLoading}
        pagination={false}
      />
      <div className={styles.sectionTitle}>支持采集资产类型</div>
      <div className={styles.tagList}>
        {supportedTypes.length ? supportedTypes.map((item) => <Tag key={item}>{item}</Tag>) : '-'}
      </div>
    </Drawer>
  );
};

const CloudAccountPage = ({ match }) => {
  const { orgId } = match.params || {};
  const [ query, setQuery ] = useState({
    currentPage: 1,
    pageSize: 10
  });
  const [ drawer, setDrawer ] = useState({
    visible: false,
    mode: 'add',
    record: {}
  });
  const [ regionDrawer, setRegionDrawer ] = useState({
    visible: false,
    record: {}
  });

  const {
    loading,
    data,
    run: fetchList
  } = useRequest(
    () => requestWrapper(cloudAccountAPI.list.bind(null, { orgId, ...query })),
    {
      refreshDeps: [ query, orgId ]
    }
  );

  const {
    loading: submitting,
    run: saveAccount
  } = useRequest(
    (payload) => requestWrapper(
      drawer.mode === 'add'
        ? cloudAccountAPI.create.bind(null, { orgId, ...payload })
        : cloudAccountAPI.update.bind(null, { orgId, id: drawer.record.id, ...payload }),
      { autoSuccess: true }
    ),
    {
      manual: true,
      onSuccess: () => {
        setDrawer({ visible: false, mode: 'add', record: {} });
        fetchList();
      }
    }
  );

  const validateAccount = async (record) => {
    await requestWrapper(cloudAccountAPI.validate.bind(null, { orgId, id: record.id }), { autoSuccess: true });
    fetchList();
  };

  const removeAccount = async (record) => {
    await requestWrapper(cloudAccountAPI.delete.bind(null, { orgId, id: record.id }), { autoSuccess: true });
    fetchList();
  };

  const toggleStatus = async (record) => {
    await requestWrapper(
      cloudAccountAPI.update.bind(null, {
        orgId,
        id: record.id,
        status: record.status === 'enable' ? 'disable' : 'enable'
      }),
      { autoSuccess: true }
    );
    fetchList();
  };

  const columns = useMemo(() => [
    {
      title: '账号名称',
      dataIndex: 'name',
      width: 180,
      ellipsis: true
    },
    {
      title: '云厂商',
      dataIndex: 'provider',
      width: 110,
      render: (value) => (providerOptions.find((item) => item.value === value) || {}).label || value || '-'
    },
    {
      title: '云账号 ID',
      dataIndex: 'accountId',
      width: 220,
      ellipsis: true,
      render: (value) => value || '-'
    },
    {
      title: '区域',
      dataIndex: 'regions',
      width: 220,
      render: joinList
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: statusTag
    },
    {
      title: '验证',
      dataIndex: 'validationStatus',
      width: 110,
      render: validationTag
    },
    {
      title: '最近验证',
      dataIndex: 'lastValidatedAt',
      width: 170,
      render: renderTime
    },
    {
      title: '最近同步',
      dataIndex: 'lastSyncAt',
      width: 170,
      render: renderTime
    },
    {
      title: '操作',
      fixed: 'right',
      width: 340,
      render: (_, record) => (
        <span className='inlineOp'>
          <a onClick={() => setDrawer({ visible: true, mode: 'edit', record })}>编辑</a>
          <span className='ant-divider ant-divider-vertical'/>
          <a onClick={() => validateAccount(record)}>验证</a>
          <span className='ant-divider ant-divider-vertical'/>
          <a onClick={() => setRegionDrawer({ visible: true, record })}>区域/权限</a>
          <span className='ant-divider ant-divider-vertical'/>
          <a onClick={() => toggleStatus(record)}>{record.status === 'enable' ? '禁用' : '启用'}</a>
          <span className='ant-divider ant-divider-vertical'/>
          <Popconfirm title='确定删除该云账号？' onConfirm={() => removeAccount(record)}>
            <a>删除</a>
          </Popconfirm>
        </span>
      )
    }
  ], [ orgId ]);

  const list = (data || {}).list || [];

  return (
    <Layout
      extraHeader={<PageHeader title='云账号' breadcrumb={true}/>}
    >
      <div className='idcos-card'>
        <div className={styles.toolbar}>
          <Space className={styles.filterBar} size={[8, 8]} wrap={true}>
            <InputSearch
              className={styles.keywordSearch}
              allowClear={true}
              placeholder='搜索账号名称、描述或账号 ID'
              onSearch={(value) => setQuery({ ...query, q: value, currentPage: 1 })}
            />
            <Select
              allowClear={true}
              placeholder='云厂商'
              style={{ width: 160 }}
              onChange={(provider) => setQuery({ ...query, provider, currentPage: 1 })}
            >
              {providerOptions.map((item) => <Option key={item.value} value={item.value}>{item.label}</Option>)}
            </Select>
            <Select
              allowClear={true}
              placeholder='验证状态'
              style={{ width: 160 }}
              onChange={(validationStatus) => setQuery({ ...query, validationStatus, currentPage: 1 })}
            >
              {Object.entries(validationStatusMap).map(([value, item]) => (
                <Option key={value} value={value}>{item.label}</Option>
              ))}
            </Select>
          </Space>
          <Space className={styles.actionBar}>
            <Button icon={<ReloadOutlined/>} onClick={fetchList}>刷新</Button>
            <Button type='primary' icon={<PlusOutlined/>} onClick={() => setDrawer({ visible: true, mode: 'add', record: {} })}>
              创建云账号
            </Button>
          </Space>
        </div>
        <Descriptions className={styles.summary} size='small' column={4}>
          <Descriptions.Item label='账号数'>{(data || {}).total || 0}</Descriptions.Item>
          <Descriptions.Item label='可用'>{list.filter((item) => item.ready).length}</Descriptions.Item>
          <Descriptions.Item label='异常'>{list.filter((item) => item.validationStatus === 'invalid').length}</Descriptions.Item>
          <Descriptions.Item label='同步源'><CloudSyncOutlined/> CMDB</Descriptions.Item>
        </Descriptions>
        <Table
          rowKey='id'
          columns={columns}
          dataSource={list}
          loading={loading}
          scroll={{ x: 'min-content' }}
          pagination={{
            current: query.currentPage,
            pageSize: query.pageSize,
            total: (data || {}).total || 0,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共${total}条`,
            onChange: (currentPage, pageSize) => setQuery({ ...query, currentPage, pageSize })
          }}
        />
      </div>
      {drawer.visible && (
        <AccountDrawer
          visible={drawer.visible}
          mode={drawer.mode}
          record={drawer.record}
          onClose={() => setDrawer({ visible: false, mode: 'add', record: {} })}
          onSubmit={saveAccount}
          submitting={submitting}
        />
      )}
      {regionDrawer.visible && (
        <RegionPermissionDrawer
          visible={regionDrawer.visible}
          record={regionDrawer.record}
          orgId={orgId}
          onClose={() => setRegionDrawer({ visible: false, record: {} })}
          onUpdated={fetchList}
        />
      )}
    </Layout>
  );
};

export default CloudAccountPage;
