import React from 'react';
import { Alert, Form, Button, Input, Select } from 'antd';
import { useRequest } from 'ahooks';
import { requestWrapper } from 'utils/request';
import sysAPI from 'services/sys';

const layout = {
  labelCol: {
    span: 6
  },
  wrapperCol: {
    span: 16
  }
};

export default () => {

  const [form] = Form.useForm();
  const registryTypeOptions = [
    { label: '通用镜像仓库', value: 'generic' },
    { label: 'Harbor', value: 'harbor' },
    { label: 'AWS ECR', value: 'ecr' }
  ];

  // 查询配置
  const {
    run: fetchCfg
  } = useRequest(
    () => requestWrapper(
      sysAPI.getRegistryAddr.bind(null)
    ), {
      onSuccess: (data) => {
        const {
          registryAddrDB,
          imageRegistryType,
          imageRegistryAddr,
          ecrAccountId,
          ecrRegion,
          ecrEndpoint,
          imageRepositoryPrefix
        } = data || {};
        form.setFieldsValue({
          registryAddr: registryAddrDB,
          imageRegistryType: imageRegistryType || 'generic',
          imageRegistryAddr,
          ecrAccountId,
          ecrRegion,
          ecrEndpoint: ecrEndpoint || 'amazonaws.com',
          imageRepositoryPrefix
        });
      }
    }
  );

  // 更新配置
  const {
    run: updateCfg
  } = useRequest(
    (params) => requestWrapper(
      sysAPI.updateRegistryAddr.bind(null, params), {
        autoSuccess: true
      }
    ), {
      manual: true,
      onSuccess: () => {
        fetchCfg();
      }
    }
  );
  const submitLoading = false;

  const onFinish = (values) => {
    updateCfg({
      ...values,
      ecrEndpoint: values.ecrEndpoint || 'amazonaws.com'
    });
  };

  return (
    <Form
      {...layout}
      style={{ width: 600, margin: '40px auto' }}
      onFinish={onFinish}
      form={form}
      initialValues={{
        imageRegistryType: 'generic',
        ecrEndpoint: 'amazonaws.com'
      }}
    >
      <Alert
        type='info'
        showIcon={true}
        style={{ marginBottom: 24 }}
        message='Registry 地址用于 CloudIaC Registry 与 Terraform Provider Mirror；镜像仓库用于部署镜像前缀，可配置 Harbor 或 AWS ECR。'
      />
      <Form.Item
        label='CloudIaC Registry地址'
        name='registryAddr'
      >
        <Input placeholder='未设置时默认使用SaaS版Registry' />
      </Form.Item>
      <Form.Item
        label='镜像仓库类型'
        name='imageRegistryType'
      >
        <Select options={registryTypeOptions}/>
      </Form.Item>
      <Form.Item noStyle={true} shouldUpdate={true}>
        {({ getFieldValue }) => {
          const imageRegistryType = getFieldValue('imageRegistryType');
          const isEcr = imageRegistryType === 'ecr';
          return (
            <>
              <Form.Item
                label='镜像仓库地址'
                name='imageRegistryAddr'
                extra={isEcr ? '留空时会根据 AWS 账号、区域和仓库前缀自动生成。用于 DOCKER_REGISTRY 时建议以 / 结尾。' : '例如 harbor.local/cloudiac/ 或 registry.local/cloudiac/。'}
              >
                <Input placeholder={isEcr ? '123456789012.dkr.ecr.us-east-1.amazonaws.com/' : '请输入镜像仓库地址'} />
              </Form.Item>
              {isEcr && (
                <>
                  <Form.Item
                    label='AWS账号ID'
                    name='ecrAccountId'
                    rules={[{ required: true, message: '请输入AWS账号ID' }]}
                  >
                    <Input placeholder='例如 123456789012'/>
                  </Form.Item>
                  <Form.Item
                    label='AWS区域'
                    name='ecrRegion'
                    rules={[{ required: true, message: '请输入AWS区域' }]}
                  >
                    <Input placeholder='例如 us-east-1'/>
                  </Form.Item>
                  <Form.Item
                    label='ECR域名后缀'
                    name='ecrEndpoint'
                  >
                    <Input placeholder='amazonaws.com'/>
                  </Form.Item>
                  <Form.Item
                    label='仓库前缀'
                    name='imageRepositoryPrefix'
                    extra='可选，例如 platform；生成后会成为 123456789012.dkr.ecr.us-east-1.amazonaws.com/platform/。'
                  >
                    <Input placeholder='可选'/>
                  </Form.Item>
                </>
              )}
            </>
          );
        }}
      </Form.Item>
      <Form.Item wrapperCol={{ offset: 6, span: 16 }} style={{ paddingTop: 24 }}>
        <Button type='primary' htmlType='submit' loading={submitLoading}>
          保存
        </Button>
      </Form.Item>
    </Form>
  );
};
