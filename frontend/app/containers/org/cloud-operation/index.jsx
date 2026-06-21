import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Descriptions,
  Drawer,
  Empty,
  Input,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Typography
} from 'antd';
import { CheckOutlined, CloseOutlined, ReloadOutlined, StopOutlined } from '@ant-design/icons';
import { useRequest } from 'ahooks';
import moment from 'moment';
import queryString from 'query-string';
import PageHeader from 'components/pageHeader';
import Layout from 'components/common/layout';
import { requestWrapper } from 'utils/request';
import cloudOperationAPI from 'services/cloud-operation';
import styles from './styles.less';

const { Option } = Select;
const { Search: InputSearch } = Input;
const { Text } = Typography;

const statusMap = {
  pending: { label: '等待中', color: 'default' },
  approving: { label: '待审批', color: 'warning' },
  running: { label: '运行中', color: 'processing' },
  complete: { label: '完成', color: 'success' },
  failed: { label: '失败', color: 'error' },
  aborted: { label: '已取消', color: 'warning' },
  rejected: { label: '已驳回', color: 'error' }
};

const actionMap = {
  governance_ownership: '资产归属治理',
  refresh_metadata: '刷新元数据',
  update_tags: '更新标签',
  start_instance: '启动实例',
  stop_instance: '停止实例',
  restart_instance: '重启实例',
  resize_instance: '调整实例规格',
  resize_volume: '磁盘扩容',
  create_snapshot: '创建快照/备份',
  update_security_rules: '更新安全组规则',
  delete_resource: '删除资源'
};

const typeMap = {
  governance: '治理任务',
  action: '云资产动作'
};

const riskMap = {
  low: { label: '低', color: 'success' },
  medium: { label: '中', color: 'warning' },
  high: { label: '高', color: 'error' },
  critical: { label: '严重', color: 'error' }
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

const OperationDetail = ({
  detail = {},
  loading,
  onCancelOperation,
  canceling,
  onRetryOperation,
  retrying,
  onApproveOperation,
  approving,
  onRejectOperation,
  rejecting
}) => {
  const steps = detail.steps || [];
  const audits = detail.audits || [];
  const params = detail.params || {};
  const result = detail.result || {};
  const canCancel = detail.status === 'pending' || detail.status === 'approving' || detail.status === 'running';
  const canRetry = detail.status === 'failed' && detail.operationType === 'action';
  const canApprove = detail.status === 'approving' && detail.operationType === 'action';
  const rollbackHint = result.rollbackHint ||
    (canCancel && params.executionMode === 'async'
      ? '取消仅停止平台任务跟踪；如果 provider 写操作已经提交到云厂商，平台不会自动回滚云端资源，请到云厂商控制台或后续采集结果核对最终状态。'
      : '');
  const stepColumns = [
    {
      title: '步骤',
      dataIndex: 'name',
      width: 180,
      render: (text) => text || '-'
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: statusTag
    },
    {
      title: '信息',
      dataIndex: 'message',
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      title: '结束时间',
      dataIndex: 'endedAt',
      width: 180,
      render: renderTime
    }
  ];
  const auditColumns = [
    {
      title: '动作',
      dataIndex: 'action',
      width: 160,
      render: (text) => actionMap[text] || text || '-'
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: statusTag
    },
    {
      title: '操作人',
      dataIndex: 'operatorName',
      width: 140,
      render: (text, record) => text || record.operatorId || '-'
    },
    {
      title: '摘要',
      dataIndex: 'summary',
      render: (text) => text || '-'
    },
    {
      title: '时间',
      dataIndex: 'createdAt',
      width: 180,
      render: renderTime
    }
  ];

  if (loading) {
    return <Empty description='加载中'/>;
  }
  return (
    <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
      <Descriptions size='small' bordered={true} column={2}>
        <Descriptions.Item label='任务 ID'>{detail.id || '-'}</Descriptions.Item>
        <Descriptions.Item label='状态'>{statusTag(detail.status)}</Descriptions.Item>
        <Descriptions.Item label='类型'>{typeMap[detail.operationType] || detail.operationType || '-'}</Descriptions.Item>
        <Descriptions.Item label='动作'>{actionMap[detail.action] || detail.action || '-'}</Descriptions.Item>
        <Descriptions.Item label='风险'>{riskTag(detail.riskLevel)}</Descriptions.Item>
        <Descriptions.Item label='创建人'>{detail.creatorName || detail.creatorId || '-'}</Descriptions.Item>
        <Descriptions.Item label='项目'>{detail.projectName || detail.projectId || '-'}</Descriptions.Item>
        <Descriptions.Item label='环境'>{detail.envName || detail.envId || '-'}</Descriptions.Item>
        <Descriptions.Item label='资源'>{detail.resourceName || detail.assetName || detail.assetId || '-'}</Descriptions.Item>
        <Descriptions.Item label='云厂商'>{detail.provider || '-'}</Descriptions.Item>
        <Descriptions.Item label='开始时间'>{renderTime(detail.startedAt)}</Descriptions.Item>
        <Descriptions.Item label='结束时间'>{renderTime(detail.endedAt)}</Descriptions.Item>
        <Descriptions.Item label='信息' span={2}>{detail.message || '-'}</Descriptions.Item>
      </Descriptions>
      {rollbackHint && (
        <Alert
          showIcon={true}
          type='warning'
          message='云端回滚提示'
          description={rollbackHint}
        />
      )}
      {canCancel && (
        <Popconfirm title='确认取消该操作任务？' onConfirm={onCancelOperation}>
          <Button danger={true} icon={<StopOutlined/>} loading={canceling}>取消任务</Button>
        </Popconfirm>
      )}
      {canApprove && (
        <Space size={8}>
          <Popconfirm title='确认审批通过并继续执行该操作任务？' onConfirm={onApproveOperation}>
            <Button type='primary' icon={<CheckOutlined/>} loading={approving}>审批通过</Button>
          </Popconfirm>
          <Popconfirm title='确认驳回该操作任务？' onConfirm={onRejectOperation}>
            <Button danger={true} icon={<CloseOutlined/>} loading={rejecting}>驳回任务</Button>
          </Popconfirm>
        </Space>
      )}
      {canRetry && (
        <Popconfirm title='确认按原参数创建重试任务？' onConfirm={onRetryOperation}>
          <Button type='primary' icon={<ReloadOutlined/>} loading={retrying}>重试任务</Button>
        </Popconfirm>
      )}
      <div className={styles.detailSection}>
        <Text strong={true}>参数</Text>
        <JsonBlock value={detail.params}/>
      </div>
      <div className={styles.detailSection}>
        <Text strong={true}>结果</Text>
        <JsonBlock value={detail.result}/>
      </div>
      <div className={styles.detailSection}>
        <Text strong={true}>步骤</Text>
        <Table rowKey='id' size='small' columns={stepColumns} dataSource={steps} pagination={false}/>
      </div>
      <div className={styles.detailSection}>
        <Text strong={true}>审计</Text>
        <Table rowKey='id' size='small' columns={auditColumns} dataSource={audits} pagination={false}/>
      </div>
    </Space>
  );
};

const CloudOperationPage = ({ match, location, history }) => {
  const { orgId } = match.params || {};
  const locationQuery = queryString.parse(location && location.search || '');
  const locationOperationId = Array.isArray(locationQuery.operationId) ? locationQuery.operationId[0] : locationQuery.operationId;
  const [ query, setQuery ] = useState({
    currentPage: 1,
    pageSize: 10
  });
  const [ drawer, setDrawer ] = useState({
    visible: false,
    id: ''
  });

  const setLocationOperationId = (operationId) => {
    if (!history || !history.replace) {
      return;
    }
    const nextQuery = {
      ...queryString.parse(location && location.search || '')
    };
    if (operationId) {
      nextQuery.operationId = operationId;
    } else {
      delete nextQuery.operationId;
    }
    const nextSearch = queryString.stringify(nextQuery);
    history.replace({
      pathname: location && location.pathname || `/org/${orgId}/m-cloud-operations`,
      search: nextSearch ? `?${nextSearch}` : ''
    });
  };

  const {
    loading,
    data = {},
    run: fetchList
  } = useRequest(
    () => requestWrapper(cloudOperationAPI.list.bind(null, { orgId, ...query })),
    {
      refreshDeps: [ query, orgId ]
    }
  );

  const {
    loading: detailLoading,
    data: detail = {},
    run: fetchDetail
  } = useRequest(
    (id) => requestWrapper(cloudOperationAPI.detail.bind(null, { orgId, id })),
    {
      manual: true
    }
  );

  useEffect(() => {
    if (!locationOperationId || drawer.id === locationOperationId) {
      return;
    }
    setDrawer({ visible: true, id: locationOperationId });
    fetchDetail(locationOperationId);
  }, [ locationOperationId, drawer.id, fetchDetail ]);

  const {
    loading: canceling,
    run: cancelOperation
  } = useRequest(
    (id) => requestWrapper(cloudOperationAPI.cancel.bind(null, { orgId, id }), { autoSuccess: true }),
    {
      manual: true,
      onSuccess: () => {
        fetchDetail(drawer.id);
        fetchList();
      }
    }
  );

  const {
    loading: retrying,
    run: retryOperation
  } = useRequest(
    (id) => requestWrapper(cloudOperationAPI.retry.bind(null, { orgId, id }), {
      autoSuccess: true,
      successMessage: '重试任务已创建'
    }),
    {
      manual: true,
      onSuccess: (resp = {}) => {
        const retryId = resp.id || drawer.id;
        setDrawer({ visible: true, id: retryId });
        setLocationOperationId(retryId);
        fetchDetail(retryId);
        fetchList();
      }
    }
  );

  const {
    loading: approving,
    run: approveOperation
  } = useRequest(
    (id) => requestWrapper(cloudOperationAPI.approve.bind(null, { orgId, id, action: 'approved' }), {
      autoSuccess: true,
      successMessage: '审批已通过'
    }),
    {
      manual: true,
      onSuccess: () => {
        fetchDetail(drawer.id);
        fetchList();
      }
    }
  );

  const {
    loading: rejecting,
    run: rejectOperation
  } = useRequest(
    (id) => requestWrapper(cloudOperationAPI.approve.bind(null, { orgId, id, action: 'rejected' }), {
      autoSuccess: true,
      successMessage: '任务已驳回'
    }),
    {
      manual: true,
      onSuccess: () => {
        fetchDetail(drawer.id);
        fetchList();
      }
    }
  );

  useEffect(() => {
    if (!drawer.visible || !drawer.id || ![ 'pending', 'running' ].includes(detail.status)) {
      return undefined;
    }
    const timer = window.setInterval(() => {
      fetchDetail(drawer.id);
      fetchList();
    }, 3000);
    return () => window.clearInterval(timer);
  }, [ drawer.visible, drawer.id, detail.status, fetchDetail, fetchList ]);

  const columns = useMemo(() => [
    {
      title: '任务',
      dataIndex: 'name',
      width: 180,
      ellipsis: true,
      render: (text, record) => <a onClick={() => {
        setDrawer({ visible: true, id: record.id });
        setLocationOperationId(record.id);
        fetchDetail(record.id);
      }}>{text || record.id}</a>
    },
    {
      title: '类型',
      dataIndex: 'operationType',
      width: 120,
      render: (text) => typeMap[text] || text || '-'
    },
    {
      title: '动作',
      dataIndex: 'action',
      width: 150,
      render: (text) => actionMap[text] || text || '-'
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: statusTag
    },
    {
      title: '风险',
      dataIndex: 'riskLevel',
      width: 90,
      render: riskTag
    },
    {
      title: '资源',
      dataIndex: 'resourceName',
      width: 180,
      ellipsis: true,
      render: (text, record) => text || record.assetName || record.resourceId || '-'
    },
    {
      title: '项目',
      dataIndex: 'projectName',
      width: 160,
      ellipsis: true,
      render: (text, record) => text || record.projectId || '-'
    },
    {
      title: '创建人',
      dataIndex: 'creatorName',
      width: 140,
      render: (text, record) => text || record.creatorId || '-'
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 180,
      render: renderTime
    },
    {
      title: '结束时间',
      dataIndex: 'endedAt',
      width: 180,
      render: renderTime
    }
  ], [ fetchDetail, setLocationOperationId ]);

  const list = data.list || [];
  return (
    <Layout
      extraHeader={<PageHeader title='操作任务' breadcrumb={true}/>}
    >
      <div className='idcos-card'>
        <div className={styles.toolbar}>
          <Space className={styles.filterBar} size={[8, 8]} wrap={true}>
            <InputSearch
              className={styles.keywordSearch}
              allowClear={true}
              placeholder='搜索任务、资源或消息'
              onSearch={(value) => setQuery({ ...query, q: value, currentPage: 1 })}
            />
            <Select
              allowClear={true}
              placeholder='状态'
              style={{ width: 140 }}
              onChange={(status) => setQuery({ ...query, status, currentPage: 1 })}
            >
              {Object.entries(statusMap).map(([value, item]) => (
                <Option key={value} value={value}>{item.label}</Option>
              ))}
            </Select>
            <Select
              allowClear={true}
              placeholder='动作'
              style={{ width: 180 }}
              onChange={(action) => setQuery({ ...query, action, currentPage: 1 })}
            >
              {Object.entries(actionMap).map(([value, label]) => (
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
        title='操作任务详情'
        visible={drawer.visible}
        width={840}
        destroyOnClose={true}
        onClose={() => {
          setDrawer({ visible: false, id: '' });
          setLocationOperationId('');
        }}
      >
        <OperationDetail
          detail={detail}
          loading={detailLoading}
          canceling={canceling}
          retrying={retrying}
          approving={approving}
          rejecting={rejecting}
          onCancelOperation={() => cancelOperation(drawer.id)}
          onRetryOperation={() => retryOperation(drawer.id)}
          onApproveOperation={() => approveOperation(drawer.id)}
          onRejectOperation={() => rejectOperation(drawer.id)}
        />
      </Drawer>
    </Layout>
  );
};

export default CloudOperationPage;
