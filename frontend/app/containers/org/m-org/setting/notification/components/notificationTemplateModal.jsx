import React, { useEffect, useState } from 'react';
import { Button, Divider, Drawer, Form, Input, notification, Select, Tag } from 'antd';
import notificationsAPI from 'services/notifications';
import { ORG_USER } from 'constants/types';

const { Option } = Select;

const FL = {
  labelCol: { span: 4 },
  wrapperCol: { span: 16 }
};

const variableTitle = (item) => {
  const parts = [item.description].filter(Boolean);
  if (item.sample !== undefined && item.sample !== null && item.sample !== '') {
    const sample = typeof item.sample === 'object' ? JSON.stringify(item.sample) : item.sample;
    parts.push(`示例：${sample}`);
  }
  return parts.join('；');
};

export default ({ orgId, projectId, operation, visible, toggleVisible, templateId }) => {
  const [form] = Form.useForm();
  const [variableGroups, setVariableGroups] = useState([]);
  const [preview, setPreview] = useState();
  const [previewLoading, setPreviewLoading] = useState(false);

  useEffect(() => {
    getVariables();
    if (templateId) {
      getDetail();
    } else {
      form.setFieldsValue({
        type: 'webhook',
        status: 'enable'
      });
    }
  }, []);

  const getVariables = async (params = {}) => {
    try {
      const res = await notificationsAPI.notificationTemplateVariables({
        orgId,
        projectId,
        eventType: params.eventType,
        type: params.type
      });
      if (res.code !== 200) {
        throw new Error(res.message);
      }
      setVariableGroups(res.result.groups || []);
    } catch (e) {
      notification.error({
        message: '获取失败',
        description: e.message
      });
    }
  };

  const onValuesChange = (changedValues, allValues) => {
    setPreview();
    if (changedValues.eventType || changedValues.type) {
      getVariables(allValues);
    }
  };

  const getDetail = async () => {
    try {
      const res = await notificationsAPI.detailNotificationTemplate({
        orgId,
        projectId,
        templateId
      });
      if (res.code !== 200) {
        throw new Error(res.message);
      }
      form.setFieldsValue({
        ...res.result,
        type: res.result.notificationType
      });
    } catch (e) {
      notification.error({
        message: '获取失败',
        description: e.message
      });
    }
  };

  const onPreview = async () => {
    try {
      await form.validateFields(['eventType', 'type']);
      setPreviewLoading(true);
      const res = await notificationsAPI.previewNotificationTemplate({
        orgId,
        projectId,
        ...form.getFieldsValue()
      });
      if (res.code !== 200) {
        throw new Error(res.message);
      }
      setPreview(res.result);
      setPreviewLoading(false);
    } catch (e) {
      setPreviewLoading(false);
      if (e && e.errorFields) {
        return;
      }
      notification.error({
        message: '预览失败',
        description: e.message
      });
    }
  };

  const onFinish = async () => {
    const params = await form.validateFields();
    operation({
      doWhat: templateId ? 'edit' : 'add',
      payload: {
        ...params,
        templateId
      }
    }, toggleVisible);
  };

  return <Drawer
    title={templateId ? '编辑通知模板' : '添加通知模板'}
    visible={visible}
    onClose={toggleVisible}
    width={860}
    footer={
      <div style={{ textAlign: 'right' }}>
        <Button onClick={onPreview} loading={previewLoading} style={{ marginRight: 8 }}>
          预览
        </Button>
        <Button onClick={toggleVisible} style={{ marginRight: 8 }}>
          取消
        </Button>
        <Button onClick={onFinish} type='primary'>
          确认
        </Button>
      </div>
    }
  >
    <Form
      form={form}
      {...FL}
      onValuesChange={onValuesChange}
    >
      <Form.Item
        label='名称'
        name='name'
        rules={[{ required: true, message: '请输入名称' }]}
      >
        <Input placeholder='请输入模板名称' />
      </Form.Item>
      <Form.Item
        label='事件类型'
        name='eventType'
        rules={[{ required: true, message: '请选择事件类型' }]}
      >
        <Select getPopupContainer={triggerNode => triggerNode.parentNode} placeholder='请选择事件类型'>
          {Object.keys(ORG_USER.eventType).map(it => <Option value={it} key={it}>{ORG_USER.eventType[it]}</Option>)}
        </Select>
      </Form.Item>
      <Form.Item
        label='通知类型'
        name='type'
        rules={[{ required: true, message: '请选择通知类型' }]}
      >
        <Select getPopupContainer={triggerNode => triggerNode.parentNode} placeholder='请选择通知类型'>
          {Object.keys(ORG_USER.notificationType).map(it => <Option value={it} key={it}>{ORG_USER.notificationType[it]}</Option>)}
        </Select>
      </Form.Item>
      <Form.Item
        label='状态'
        name='status'
        rules={[{ required: true, message: '请选择状态' }]}
      >
        <Select getPopupContainer={triggerNode => triggerNode.parentNode}>
          <Option value='enable'>启用</Option>
          <Option value='disable'>禁用</Option>
        </Select>
      </Form.Item>
      <Form.Item
        label='标题'
        name='title'
      >
        <Input placeholder='请输入标题模板' />
      </Form.Item>
      <Form.Item
        label='Markdown'
        name='markdownContent'
        rules={[{ required: true, message: '请输入Markdown模板' }]}
      >
        <Input.TextArea rows={6} placeholder='请输入Markdown模板' />
      </Form.Item>
      <Form.Item
        label='文本'
        name='content'
      >
        <Input.TextArea rows={6} placeholder='请输入文本模板' />
      </Form.Item>
    </Form>
    <Divider orientation='left'>变量</Divider>
    {
      variableGroups.map(group => <div key={group.name} style={{ marginBottom: 12 }}>
        <div style={{ marginBottom: 8 }}>{group.label}</div>
        {(group.variables || []).map(item => <Tag
          key={`${group.name}-${item.name}`}
          title={variableTitle(item)}
          style={{ marginBottom: 8 }}
        >
          {`{{.${item.name}}}`}
        </Tag>)}
      </div>)
    }
    {
      preview && <div>
        <Divider orientation='left'>预览结果</Divider>
        <Form {...FL}>
          <Form.Item label='标题'>
            <Input value={preview.title || '-'} readOnly />
          </Form.Item>
          <Form.Item label='Markdown'>
            <Input.TextArea rows={5} value={preview.markdownContent || '-'} readOnly />
          </Form.Item>
          <Form.Item label='文本'>
            <Input.TextArea rows={5} value={preview.content || '-'} readOnly />
          </Form.Item>
        </Form>
      </div>
    }
  </Drawer>;
};
