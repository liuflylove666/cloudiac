import React, { useState, useEffect } from 'react';
import { Button, notification, Table, Divider, Popconfirm, Drawer, Tag, Tabs, Modal } from 'antd';
import notificationsAPI from 'services/notifications';
import { ORG_USER } from 'constants/types';
import moment from 'moment';
import AddModal from './components/notificationModal';
import TemplateModal from './components/notificationTemplateModal';

const statusMap = {
  success: { label: '成功', color: 'success' },
  failed: { label: '失败', color: 'error' }
};

const templateStatusMap = {
  enable: { label: '启用', color: 'success' },
  disable: { label: '禁用', color: 'default' }
};

const templateActionMap = {
  create: { label: '创建', color: 'processing' },
  update: { label: '更新', color: 'default' },
  copy: { label: '复制', color: 'success' },
  rollback: { label: '回滚', color: 'warning' }
};

const renderTime = (value) => !value || String(value).indexOf('0001-01-01') === 0 ? '-' : moment(value).format('YYYY-MM-DD HH:mm:ss');
const renderEventType = (value) => Array.isArray(value)
  ? value.map(it => ORG_USER.eventType[it] || it).join('、')
  : ORG_USER.eventType[value] || value || '-';
const renderNotificationType = (value) => ORG_USER.notificationType[value] || value || '-';

export default ({ orgId, projectId }) => {
  const [ loading, setLoading ] = useState(false),
    [ activeTab, setActiveTab ] = useState('notifications'),
    [ visible, setVisible ] = useState(false),
    [ notificationId, setNotificationId ] = useState(),
    [ templateVisible, setTemplateVisible ] = useState(false),
    [ templateId, setTemplateId ] = useState(),
    [ templateLoading, setTemplateLoading ] = useState(false),
    [ templateMap, setTemplateMap ] = useState({
      list: [],
      total: 0
    }),
    [ templateQuery, setTemplateQuery ] = useState({
      pageNo: 1,
      pageSize: 10
    }),
    [ orgTemplateLoading, setOrgTemplateLoading ] = useState(false),
    [ orgTemplateMap, setOrgTemplateMap ] = useState({
      list: [],
      total: 0
    }),
    [ orgTemplateQuery, setOrgTemplateQuery ] = useState({
      pageNo: 1,
      pageSize: 10
    }),
    [ orgTemplateSelectedRowKeys, setOrgTemplateSelectedRowKeys ] = useState([]),
    [ copyingTemplates, setCopyingTemplates ] = useState(false),
    [ templateDiffDrawer, setTemplateDiffDrawer ] = useState({
      visible: false,
      project: null,
      source: null
    }),
    [ templateHistoryDrawer, setTemplateHistoryDrawer ] = useState({
      visible: false,
      record: null
    }),
    [ templateVersionLoading, setTemplateVersionLoading ] = useState(false),
    [ templateVersionMap, setTemplateVersionMap ] = useState({
      list: [],
      total: 0
    }),
    [ templateVersionQuery, setTemplateVersionQuery ] = useState({
      pageNo: 1,
      pageSize: 10
    }),
    [ deliveryDrawer, setDeliveryDrawer ] = useState({
      visible: false,
      record: null
    }),
    [ deliveryLoading, setDeliveryLoading ] = useState(false),
    [ deliveryMap, setDeliveryMap ] = useState({
      list: [],
      total: 0
    }),
    [ deliveryQuery, setDeliveryQuery ] = useState({
      pageNo: 1,
      pageSize: 10
    }),
    [ resultMap, setResultMap ] = useState({
      list: [],
      total: 0
    }),
    [ query, setQuery ] = useState({
      pageNo: 1,
      pageSize: 10
    });

  useEffect(() => {
    if (activeTab === 'notifications') {
      fetchList();
    }
  }, [query, activeTab]);

  useEffect(() => {
    if (activeTab === 'templates') {
      fetchTemplateList();
      if (projectId) {
        fetchOrgTemplateList();
      }
    }
  }, [templateQuery, orgTemplateQuery, activeTab]);


  const fetchList = async () => {
    try {
      setLoading(true);
      const res = await notificationsAPI.notificationList({
        ...query,
        orgId,
        projectId
      });
      if (res.code !== 200) {
        throw new Error(res.message);
      }
      setResultMap({
        list: res.result.list || [],
        total: res.result.total || 0
      });
      setLoading(false);
    } catch (e) {
      setLoading(false);
      notification.error({
        message: '获取失败',
        description: e.message
      });
    }
  };

  const changeQuery = (payload) => {
    setQuery({
      ...query,
      ...payload
    });
  };

  const changeTemplateQuery = (payload) => {
    setTemplateQuery({
      ...templateQuery,
      ...payload
    });
  };

  const changeOrgTemplateQuery = (payload) => {
    setOrgTemplateQuery({
      ...orgTemplateQuery,
      ...payload
    });
  };

  const toggleVisible = () => {
    setVisible(false); 
    setNotificationId();
  };

  const toggleTemplateVisible = () => {
    setTemplateVisible(false);
    setTemplateId();
  };

  const fetchTemplateList = async () => {
    try {
      setTemplateLoading(true);
      const res = await notificationsAPI.notificationTemplateList({
        ...templateQuery,
        orgId,
        projectId
      });
      if (res.code !== 200) {
        throw new Error(res.message);
      }
      setTemplateMap({
        list: res.result.list || [],
        total: res.result.total || 0
      });
      setTemplateLoading(false);
    } catch (e) {
      setTemplateLoading(false);
      notification.error({
        message: '获取失败',
        description: e.message
      });
    }
  };

  const fetchOrgTemplateList = async () => {
    try {
      setOrgTemplateLoading(true);
      const res = await notificationsAPI.notificationTemplateList({
        ...orgTemplateQuery,
        orgId
      });
      if (res.code !== 200) {
        throw new Error(res.message);
      }
      setOrgTemplateMap({
        list: res.result.list || [],
        total: res.result.total || 0
      });
      setOrgTemplateLoading(false);
    } catch (e) {
      setOrgTemplateLoading(false);
      notification.error({
        message: '获取失败',
        description: e.message
      });
    }
  };

  const closeDeliveryDrawer = () => {
    setDeliveryDrawer({
      visible: false,
      record: null
    });
    setDeliveryMap({
      list: [],
      total: 0
    });
  };

  const fetchDeliveries = async (record, payload = {}) => {
    if (!record || !record.id) {
      return;
    }
    const nextQuery = {
      ...deliveryQuery,
      ...payload
    };
    setDeliveryQuery(nextQuery);
    try {
      setDeliveryLoading(true);
      const res = await notificationsAPI.notificationDeliveries({
        orgId,
        projectId,
        notificationId: record.id,
        ...nextQuery
      });
      if (res.code !== 200) {
        throw new Error(res.message);
      }
      setDeliveryMap({
        list: res.result.list || [],
        total: res.result.total || 0
      });
      setDeliveryLoading(false);
    } catch (e) {
      setDeliveryLoading(false);
      notification.error({
        message: '获取失败',
        description: e.message
      });
    }
  };

  const openDeliveryDrawer = (record) => {
    const initialQuery = {
      pageNo: 1,
      pageSize: 10
    };
    setDeliveryDrawer({
      visible: true,
      record
    });
    setDeliveryQuery(initialQuery);
    fetchDeliveries(record, initialQuery);
  };

  const getOrgTemplateForProject = (record) => {
    const orgTemplates = orgTemplateMap.list || [];
    if (!record) {
      return null;
    }
    return orgTemplates.find(item => record.sourceTemplateId && item.id === record.sourceTemplateId)
      || orgTemplates.find(item => item.eventType === record.eventType && item.notificationType === record.notificationType)
      || null;
  };

  const isOrgTemplateCovered = (record) => {
    return (templateMap.list || []).some(item => item.eventType === record.eventType && item.notificationType === record.notificationType);
  };

  const getTemplateDiffRows = (projectTemplate, sourceTemplate) => {
    const fields = [
      { key: 'name', label: '名称' },
      { key: 'title', label: '标题' },
      { key: 'markdownContent', label: 'Markdown' },
      { key: 'content', label: '文本' },
      { key: 'status', label: '状态', render: (value) => (templateStatusMap[value] || {}).label || value || '-' }
    ];
    return fields.map(field => {
      const projectValue = projectTemplate ? projectTemplate[field.key] : '';
      const sourceValue = sourceTemplate ? sourceTemplate[field.key] : '';
      const render = field.render || ((value) => value || '-');
      return {
        key: field.key,
        label: field.label,
        projectValue: render(projectValue),
        sourceValue: render(sourceValue),
        same: String(projectValue || '') === String(sourceValue || '')
      };
    });
  };

  const templateHasDiff = (record) => {
    const source = getOrgTemplateForProject(record);
    if (!source) {
      return false;
    }
    return getTemplateDiffRows(record, source).some(item => !item.same);
  };

  const openTemplateDiffDrawer = async (record) => {
    let source = getOrgTemplateForProject(record);
    if (!source && record.sourceTemplateId) {
      try {
        const res = await notificationsAPI.detailNotificationTemplate({
          orgId,
          templateId: record.sourceTemplateId
        });
        if (res.code !== 200) {
          throw new Error(res.message);
        }
        source = res.result;
      } catch (e) {
        notification.error({
          message: '获取失败',
          description: e.message
        });
        return;
      }
    }
    if (!source) {
      notification.info({
        message: '未找到组织模板'
      });
      return;
    }
    setTemplateDiffDrawer({
      visible: true,
      project: record,
      source
    });
  };

  const closeTemplateDiffDrawer = () => {
    setTemplateDiffDrawer({
      visible: false,
      project: null,
      source: null
    });
  };

  const closeTemplateHistoryDrawer = () => {
    setTemplateHistoryDrawer({
      visible: false,
      record: null
    });
    setTemplateVersionMap({
      list: [],
      total: 0
    });
  };

  const fetchTemplateVersions = async (record, payload = {}) => {
    if (!record || !record.id) {
      return;
    }
    const nextQuery = {
      ...templateVersionQuery,
      ...payload
    };
    setTemplateVersionQuery(nextQuery);
    try {
      setTemplateVersionLoading(true);
      const res = await notificationsAPI.notificationTemplateVersions({
        orgId,
        projectId,
        templateId: record.id,
        ...nextQuery
      });
      if (res.code !== 200) {
        throw new Error(res.message);
      }
      setTemplateVersionMap({
        list: res.result.list || [],
        total: res.result.total || 0
      });
      setTemplateVersionLoading(false);
    } catch (e) {
      setTemplateVersionLoading(false);
      notification.error({
        message: '获取失败',
        description: e.message
      });
    }
  };

  const openTemplateHistoryDrawer = (record) => {
    const initialQuery = {
      pageNo: 1,
      pageSize: 10
    };
    setTemplateHistoryDrawer({
      visible: true,
      record
    });
    setTemplateVersionQuery(initialQuery);
    fetchTemplateVersions(record, initialQuery);
  };

  const columns = [
    {
      dataIndex: 'name',
      title: '名称',
      width: 165,
      ellipsis: true
    },
    {
      dataIndex: 'notificationType',
      title: '类型',
      width: 149,
      ellipsis: true,
      render: (text) => ORG_USER.notificationType[text]
    },
    {
      dataIndex: 'eventType',
      title: '事件类型',
      width: 277,
      ellipsis: true,
      render: (text) => (text || []).map(it => ORG_USER.eventType[it]).join('、')
    },
    {
      dataIndex: 'creatorName',
      title: '创建人',
      width: 169,
      ellipsis: true
    },
    {
      dataIndex: 'createdAt',
      title: '创建时间',
      width: 219,
      ellipsis: true,
      render: (text) => moment(text).format('YYYY-MM-DD HH:mm:ss')
    },
    {
      title: '操作',
      width: 220,
      ellipsis: true,
      fixed: 'right',
      render: (_, record) => <span>
        <a
          onClick={() => {
            setVisible(true);
            setNotificationId(record.id);
          }}
        >
          编辑
        </a>
        <Divider type={'vertical'}/>
        <a
          onClick={() => openDeliveryDrawer(record)}
        >
          历史
        </a>
        <Divider type={'vertical'}/>
        <Popconfirm
          title='确定要删除该通知？'
          onConfirm={() => operation({ doWhat: 'del', payload: { id: record.id } })}
        >
          <a>
            删除
          </a>
        </Popconfirm>
      </span> 
    }
  ];

  const deliveryColumns = [
    {
      dataIndex: 'status',
      title: '状态',
      width: 90,
      render: (text) => {
        const item = statusMap[text] || { label: text || '-', color: 'default' };
        return <Tag color={item.color}>{item.label}</Tag>;
      }
    },
    {
      dataIndex: 'notificationType',
      title: '类型',
      width: 100,
      render: (text) => ORG_USER.notificationType[text] || text || '-'
    },
    {
      dataIndex: 'eventType',
      title: '事件类型',
      width: 180,
      ellipsis: true,
      render: (text) => ORG_USER.eventType[text] || text || '-'
    },
    {
      dataIndex: 'target',
      title: '目标',
      width: 240,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'title',
      title: '标题',
      width: 180,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'errorMessage',
      title: '错误',
      width: 240,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'deliveredAt',
      title: '投递时间',
      width: 180,
      render: renderTime
    }
  ];

  const templateColumns = [
    {
      dataIndex: 'name',
      title: '名称',
      width: 180,
      ellipsis: true
    },
    {
      dataIndex: 'eventType',
      title: '事件类型',
      width: 210,
      ellipsis: true,
      render: renderEventType
    },
    {
      dataIndex: 'notificationType',
      title: '通知类型',
      width: 120,
      render: renderNotificationType
    },
    {
      dataIndex: 'status',
      title: '状态',
      width: 100,
      render: (text) => {
        const item = templateStatusMap[text] || { label: text || '-', color: 'default' };
        return <Tag color={item.color}>{item.label}</Tag>;
      }
    },
    {
      dataIndex: 'creatorName',
      title: '创建人',
      width: 150,
      ellipsis: true
    },
    {
      title: '来源',
      width: 150,
      ellipsis: true,
      hidden: !projectId,
      render: (_, record) => {
        const source = getOrgTemplateForProject(record);
        if (source) {
          return source.name;
        }
        return record.sourceTemplateId ? '已关联' : '-';
      }
    },
    {
      title: '差异',
      width: 100,
      hidden: !projectId,
      render: (_, record) => {
        const source = getOrgTemplateForProject(record);
        if (!source && !record.sourceTemplateId) {
          return '-';
        }
        if (!source && record.sourceTemplateId) {
          return <Tag color='processing'>已关联</Tag>;
        }
        return templateHasDiff(record) ? <Tag color='warning'>有差异</Tag> : <Tag color='success'>一致</Tag>;
      }
    },
    {
      dataIndex: 'updatedAt',
      title: '更新时间',
      width: 180,
      render: renderTime
    },
    {
      title: '操作',
      width: projectId ? 320 : 180,
      fixed: 'right',
      render: (_, record) => <span>
        <a
          onClick={() => {
            setTemplateVisible(true);
            setTemplateId(record.id);
          }}
        >
          编辑
        </a>
        <Divider type={'vertical'}/>
        <a onClick={() => openTemplateHistoryDrawer(record)}>
          历史
        </a>
        {
          projectId && <>
            <Divider type={'vertical'}/>
            <a onClick={() => openTemplateDiffDrawer(record)}>
              差异
            </a>
            <Divider type={'vertical'}/>
            <Popconfirm
              title='确定回退到组织模板？'
              onConfirm={() => templateOperation({ doWhat: 'del', payload: { id: record.id } }, fetchOrgTemplateList)}
            >
              <a>
                回退
              </a>
            </Popconfirm>
          </>
        }
        <Divider type={'vertical'}/>
        <Popconfirm
          title='确定要删除该通知模板？'
          onConfirm={() => templateOperation({ doWhat: 'del', payload: { id: record.id } })}
        >
          <a>
            删除
          </a>
        </Popconfirm>
      </span>
    }
  ].filter(item => !item.hidden);

  const orgTemplateColumns = [
    {
      dataIndex: 'name',
      title: '名称',
      width: 180,
      ellipsis: true
    },
    {
      dataIndex: 'eventType',
      title: '事件类型',
      width: 210,
      ellipsis: true,
      render: renderEventType
    },
    {
      dataIndex: 'notificationType',
      title: '通知类型',
      width: 120,
      render: renderNotificationType
    },
    {
      dataIndex: 'status',
      title: '状态',
      width: 100,
      render: (text) => {
        const item = templateStatusMap[text] || { label: text || '-', color: 'default' };
        return <Tag color={item.color}>{item.label}</Tag>;
      }
    },
    {
      dataIndex: 'creatorName',
      title: '创建人',
      width: 150,
      ellipsis: true
    },
    {
      dataIndex: 'updatedAt',
      title: '更新时间',
      width: 180,
      render: renderTime
    },
    {
      title: '操作',
      width: 100,
      fixed: 'right',
      render: (_, record) => {
        const covered = isOrgTemplateCovered(record);
        return covered ? <span>已覆盖</span> : <a onClick={() => copyOrgTemplate(record)}>复制</a>;
      }
    }
  ];

  const operation = async ({ doWhat, payload }, cb) => {
    try {
      const method = {
        add: (param) => notificationsAPI.createNotification(param),
        edit: (param) => notificationsAPI.updateNotification(param),
        del: ({ orgId, projectId, id }) => notificationsAPI.delNotification({ orgId, projectId, id })
      };
      const res = await method[doWhat]({
        orgId,
        projectId,
        ...payload
      });
      if (res.code != 200) {
        throw new Error(res.message);
      }
      notification.success({
        message: '操作成功'
      });
      fetchList();
      cb && cb();
    } catch (e) {
      notification.error({
        message: '操作失败',
        description: e.message
      });
    }
  };

  const templateOperation = async ({ doWhat, payload }, cb) => {
    try {
      const method = {
        add: (param) => notificationsAPI.createNotificationTemplate(param),
        edit: (param) => notificationsAPI.updateNotificationTemplate(param),
        del: ({ orgId, projectId, id }) => notificationsAPI.delNotificationTemplate({ orgId, projectId, id })
      };
      const res = await method[doWhat]({
        orgId,
        projectId,
        ...payload
      });
      if (res.code != 200) {
        throw new Error(res.message);
      }
      notification.success({
        message: '操作成功'
      });
      if (doWhat === 'del') {
        closeTemplateDiffDrawer();
      }
      fetchTemplateList();
      cb && cb();
    } catch (e) {
      notification.error({
        message: '操作失败',
        description: e.message
      });
    }
  };

  const rollbackTemplateVersion = async (version) => {
    if (!templateHistoryDrawer.record || !version || !version.id) {
      return;
    }
    try {
      const res = await notificationsAPI.rollbackNotificationTemplateVersion({
        orgId,
        projectId,
        templateId: templateHistoryDrawer.record.id,
        versionId: version.id
      });
      if (res.code !== 200) {
        throw new Error(res.message);
      }
      notification.success({
        message: '回滚成功'
      });
      setTemplateHistoryDrawer({
        visible: true,
        record: res.result || templateHistoryDrawer.record
      });
      fetchTemplateList();
      if (projectId) {
        fetchOrgTemplateList();
      }
      fetchTemplateVersions(templateHistoryDrawer.record, {
        pageNo: 1,
        pageSize: templateVersionQuery.pageSize
      });
    } catch (e) {
      notification.error({
        message: '回滚失败',
        description: e.message
      });
    }
  };

  const copyOrgTemplates = async (sourceTemplateIds = []) => {
    const ids = sourceTemplateIds.filter(Boolean);
    if (!ids.length) {
      notification.info({
        message: '请选择组织模板'
      });
      return;
    }
    try {
      setCopyingTemplates(true);
      const res = await notificationsAPI.copyNotificationTemplatesToProject({
        orgId,
        projectId,
        sourceTemplateIds: ids
      });
      if (res.code !== 200) {
        throw new Error(res.message);
      }
      const result = res.result || {};
      notification.success({
        message: '复制完成',
        description: `新增${result.copied || 0}个，跳过${result.skipped || 0}个`
      });
      setOrgTemplateSelectedRowKeys([]);
      fetchTemplateList();
      fetchOrgTemplateList();
    } catch (e) {
      notification.error({
        message: '操作失败',
        description: e.message
      });
    } finally {
      setCopyingTemplates(false);
    }
  };

  const copyOrgTemplate = (record) => {
    copyOrgTemplates([record.id]);
  };

  const confirmCopySelectedOrgTemplates = () => {
    if (!orgTemplateSelectedRowKeys.length) {
      notification.info({
        message: '请选择组织模板'
      });
      return;
    }
    Modal.confirm({
      title: '确定批量复制已选择的组织模板？',
      content: '已覆盖的项目模板将自动跳过。',
      okText: '复制',
      cancelText: '取消',
      onOk: () => copyOrgTemplates(orgTemplateSelectedRowKeys)
    });
  };

  const templateDiffColumns = [
    {
      dataIndex: 'label',
      title: '字段',
      width: 120
    },
    {
      dataIndex: 'projectValue',
      title: '项目模板',
      ellipsis: true
    },
    {
      dataIndex: 'sourceValue',
      title: '组织模板',
      ellipsis: true
    },
    {
      dataIndex: 'same',
      title: '状态',
      width: 90,
      render: (same) => same ? <Tag color='success'>一致</Tag> : <Tag color='warning'>有差异</Tag>
    }
  ];

  const templateVersionColumns = [
    {
      dataIndex: 'versionNo',
      title: '版本',
      width: 80,
      render: (text) => `v${text}`
    },
    {
      dataIndex: 'action',
      title: '动作',
      width: 90,
      render: (text) => {
        const item = templateActionMap[text] || { label: text || '-', color: 'default' };
        return <Tag color={item.color}>{item.label}</Tag>;
      }
    },
    {
      dataIndex: 'name',
      title: '名称',
      width: 180,
      ellipsis: true
    },
    {
      dataIndex: 'sourceName',
      title: '来源快照',
      width: 180,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'status',
      title: '状态',
      width: 90,
      render: (text) => {
        const item = templateStatusMap[text] || { label: text || '-', color: 'default' };
        return <Tag color={item.color}>{item.label}</Tag>;
      }
    },
    {
      dataIndex: 'operatorName',
      title: '操作人',
      width: 140,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'createdAt',
      title: '版本时间',
      width: 180,
      render: renderTime
    },
    {
      title: '操作',
      width: 90,
      fixed: 'right',
      render: (_, record) => <Popconfirm
        title={`确定回滚到 v${record.versionNo}？`}
        onConfirm={() => rollbackTemplateVersion(record)}
      >
        <a>
          回滚
        </a>
      </Popconfirm>
    }
  ];

  return <div>
    <Tabs activeKey={activeTab} onChange={setActiveTab}>
      <Tabs.TabPane tab='通知配置' key='notifications'>
        <div style={{ marginBottom: 20 }}>
          <Button
            type='primary'
            onClick={() => {
              setVisible(true);
            }}
          >添加通知</Button>
        </div>
        <Table
          rowKey='id'
          columns={columns}
          dataSource={resultMap.list}
          loading={loading}
          scroll={{ x: 'min-content' }}
          pagination={{
            current: query.pageNo,
            pageSize: query.pageSize,
            total: resultMap.total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共${total}条`,
            onChange: (page, pageSize) => {
              changeQuery({
                pageNo: page,
                pageSize
              });
            }
          }}
        />
        {
          visible && <AddModal
            orgId={orgId}
            projectId={projectId}
            reload={fetchList}
            operation={operation}
            visible={visible}
            toggleVisible={toggleVisible}
            notificationId={notificationId}
          />
        }
      </Tabs.TabPane>
      <Tabs.TabPane tab='通知模板' key='templates'>
        <div style={{ marginBottom: 20 }}>
          <Button
            type='primary'
            onClick={() => {
              setTemplateVisible(true);
            }}
          >添加模板</Button>
        </div>
        <Table
          title={projectId ? () => '项目模板' : undefined}
          rowKey='id'
          columns={templateColumns}
          dataSource={templateMap.list}
          loading={templateLoading}
          scroll={{ x: 'min-content' }}
          pagination={{
            current: templateQuery.pageNo,
            pageSize: templateQuery.pageSize,
            total: templateMap.total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共${total}条`,
            onChange: (page, pageSize) => {
              changeTemplateQuery({
                pageNo: page,
                pageSize
              });
            }
          }}
        />
        {
          projectId && <Table
            title={() => <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span>组织模板</span>
              <Button
                size='small'
                loading={copyingTemplates}
                disabled={!orgTemplateSelectedRowKeys.length}
                onClick={confirmCopySelectedOrgTemplates}
              >
                批量复制
              </Button>
            </div>}
            rowKey='id'
            rowSelection={{
              selectedRowKeys: orgTemplateSelectedRowKeys,
              onChange: setOrgTemplateSelectedRowKeys,
              getCheckboxProps: (record) => ({
                disabled: isOrgTemplateCovered(record)
              })
            }}
            columns={orgTemplateColumns}
            dataSource={orgTemplateMap.list}
            loading={orgTemplateLoading}
            scroll={{ x: 'min-content' }}
            style={{ marginTop: 24 }}
            pagination={{
              current: orgTemplateQuery.pageNo,
              pageSize: orgTemplateQuery.pageSize,
              total: orgTemplateMap.total,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total) => `共${total}条`,
              onChange: (page, pageSize) => {
                changeOrgTemplateQuery({
                  pageNo: page,
                  pageSize
                });
              }
            }}
          />
        }
        {
          templateVisible && <TemplateModal
            orgId={orgId}
            projectId={projectId}
            operation={templateOperation}
            visible={templateVisible}
            toggleVisible={toggleTemplateVisible}
            templateId={templateId}
          />
        }
      </Tabs.TabPane>
    </Tabs>
    <Drawer
      title={`投递历史${deliveryDrawer.record ? `：${deliveryDrawer.record.name}` : ''}`}
      visible={deliveryDrawer.visible}
      onClose={closeDeliveryDrawer}
      width={960}
    >
      <Table
        rowKey='id'
        size='small'
        columns={deliveryColumns}
        dataSource={deliveryMap.list}
        loading={deliveryLoading}
        scroll={{ x: 'min-content' }}
        pagination={{
          current: deliveryQuery.pageNo,
          pageSize: deliveryQuery.pageSize,
          total: deliveryMap.total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共${total}条`,
          onChange: (page, pageSize) => {
            fetchDeliveries(deliveryDrawer.record, {
              pageNo: page,
              pageSize
            });
          }
        }}
      />
    </Drawer>
    {
      templateDiffDrawer.visible && <Drawer
        title='模板差异'
        visible={templateDiffDrawer.visible}
        onClose={closeTemplateDiffDrawer}
        width={900}
      >
        <Table
          rowKey='key'
          size='small'
          pagination={false}
          columns={templateDiffColumns}
          dataSource={getTemplateDiffRows(templateDiffDrawer.project, templateDiffDrawer.source)}
        />
      </Drawer>
    }
    {
      templateHistoryDrawer.visible && <Drawer
        title={`模板历史${templateHistoryDrawer.record ? `：${templateHistoryDrawer.record.name}` : ''}`}
        visible={templateHistoryDrawer.visible}
        onClose={closeTemplateHistoryDrawer}
        width={1080}
      >
        <Table
          rowKey='id'
          size='small'
          columns={templateVersionColumns}
          dataSource={templateVersionMap.list}
          loading={templateVersionLoading}
          scroll={{ x: 'min-content' }}
          pagination={{
            current: templateVersionQuery.pageNo,
            pageSize: templateVersionQuery.pageSize,
            total: templateVersionMap.total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共${total}条`,
            onChange: (page, pageSize) => {
              fetchTemplateVersions(templateHistoryDrawer.record, {
                pageNo: page,
                pageSize
              });
            }
          }}
        />
      </Drawer>
    }
  </div>;
};
