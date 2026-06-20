import React, { useState, useEffect, useImperativeHandle, memo } from 'react';
import { Space, Select, Form, Input, Button, Empty, notification, Tooltip, Row, Col } from "antd";
import { QuestionCircleOutlined } from '@ant-design/icons';
import isEqual from 'lodash/isEqual';
import uniqBy from 'lodash/uniqBy';
import { useRequest } from 'ahooks';
import { requestWrapper } from 'utils/request';
import tplAPI from 'services/tpl';
import vcsAPI from 'services/vcs';
import OpModal from 'components/vcs-modal';

const FL = {
  labelCol: { span: 6 },
  wrapperCol: { span: 14 }
};
const { Option, OptGroup } = Select;
const DEFAULT_TERRAFORM_VERSION = '1.15.6';

const Repo = ({ onlineCheckForm, goCTlist, childRef, stepHelper, orgId, ctData, type, opType, saveLoading }) => {

  const formData = ctData[type] || {};
  const [form] = Form.useForm();
  const [ vcsVisible, setVcsVisible ] = useState(false);
  const [ repoBranches, setRepoBranches ] = useState([]);
  const [ repoTags, setRepoTags ] = useState([]);
  const [ vcsList, setVcsList ] = useState([]);

  // Terraform版本选项列表
  const {
    data: tfversionOptions = []
  } = useRequest(
    () => requestWrapper(
      tplAPI.listTfversions.bind(null, { orgId })
    )
  );
  const tfversionOptionList = uniqBy(
    [ ...(tfversionOptions || []), DEFAULT_TERRAFORM_VERSION ].map(value => ({ value })),
    'value'
  ).map(it => it.value);
  const latestTfVersion = tfversionOptionList[0] || DEFAULT_TERRAFORM_VERSION;
  const normalizeTfVersion = value => tfversionOptionList.includes(value) ? value : latestTfVersion;
  
  useEffect(() => {
    fetchVcsList();
  }, []);

  useEffect(() => {
    form.setFieldsValue({
      ...formData,
      tfVersion: normalizeTfVersion(formData.tfVersion)
    });
    if (formData.vcsId) {
      fetchRepos(formData);
    }
    if (formData.repoId) {
      fetchRepoBranches(formData);
      fetchRepoTags(formData);
    }
  }, [ctData, type, latestTfVersion]);

  const fetchVcsList = async () => {
    try {
      const res = await vcsAPI.searchVcs({
        orgId,
        status: 'enable',
        isShowDefaultVcs: true,
        pageSize: 0
      });
      if (res.code !== 200) {
        throw new Error(res.message);
      }
      setVcsList(res.result.list || []);
    } catch (e) {
      notification.error({
        message: '获取失败',
        description: e.message
      });
    }
  };

  // 获取仓库列表
  const {
    data: repos = [],
    run: fetchRepos,
    mutate: setRepos
  } = useRequest(
    ({ vcsId, q }) => requestWrapper(
      vcsAPI.listRepo.bind(null, { 
        orgId,
        vcsId,
        currentPage: 1,
        pageSize: 50,
        q
      })
    ),
    {
      manual: true,
      debounceInterval: 300,
      formatResult: data => {
        const { repoId, repoFullName } = form.getFieldsValue();
        const hasSelectedItem = (data.list || []).find((it) => it.id === repoId);
        if (repoId && repoFullName && !hasSelectedItem) {
          return [ ...data.list, { id: repoId, fullName: repoFullName } ];
        } else {
          return data.list || [];
        }
      }
    }
  );

  const fetchRepoBranches = async ({ vcsId, repoId }) => {
    try {
      const res = await vcsAPI.listRepoBranch({
        orgId,
        vcsId,
        repoId
      });
      if (res.code != 200) {
        throw new Error(res.message);
      }
      setRepoBranches(res.result || []);
    } catch (e) {
      notification.error({
        message: '获取仓库分支失败',
        description: e.message
      });
    }
  };

  const fetchRepoTags = async ({ vcsId, repoId }) => {
    try {
      const res = await vcsAPI.listRepoTag({
        orgId,
        vcsId,
        repoId
      });
      if (res.code != 200) {
        throw new Error(res.message);
      }
      setRepoTags(res.result || []);
    } catch (e) {
      notification.error({
        message: '获取仓库标签失败',
        description: e.message
      });
    }
  };

  const vcsOperation = async ({ doWhat, payload }, cb) => {
    try {
      const method = {
        add: (param) => vcsAPI.createVcs(param)
      };
      const res = await method[doWhat]({
        orgId,
        ...payload
      });
      if (res.code != 200) {
        throw new Error(res.message);
      }
      notification.success({
        message: '操作成功'
      });
      fetchVcsList();
      cb && cb();
    } catch (e) {
      cb && cb(e);
      notification.error({
        message: '操作失败',
        description: e.message
      });
    }
  };

  const onValuesChange = (changedValues, allValues) => {
    if (changedValues.vcsId) {
      // 切换vcs需要将关联的数据源【仓库、分支、标签】清空 ，再重新查询数据源
      setRepos([]);
      setRepoBranches([]);
      setRepoTags([]);
      fetchRepos(allValues);
      mutateAutoMatchTfVersion(undefined);
      form.setFieldsValue({
        repoId: undefined,
        repoRevision: undefined,
        workdir: undefined,
        tfVersion: latestTfVersion
      });
    }
    if (changedValues.repoId) {
      // 切换仓库需要将关联的数据源【分支、标签】清空 ，再重新查询数据源
      setRepoBranches([]);
      setRepoTags([]);
      fetchRepoBranches(allValues);
      fetchRepoTags(allValues);
      form.setFieldsValue({
        repoFullName: (repos.find(it => it.id === changedValues.repoId) || {}).fullName,
        repoRevision: undefined,
        workdir: undefined,
        tfVersion: latestTfVersion
      });
    }
    if (changedValues.repoRevision) {
      form.setFieldsValue({
        tfVersion: latestTfVersion,
      });
    }
  };

  useImperativeHandle(childRef, () => ({
    onFinish: async (index) => {
      const values = await form.validateFields();
      await onlineCheckForm(values);
      stepHelper.updateData({
        type, 
        data: values
      });
      stepHelper.go(index);
    }
  }));

  const onFinish = async (values) => {
    await onlineCheckForm(values);
    stepHelper.updateData({
      type, 
      data: values,
      isSubmit: opType === 'edit'
    });
    opType === 'add' && stepHelper.next();
  };

  const opVcsModal = () => setVcsVisible(true);

  const clVcsModal = () => setVcsVisible(false);

  const onSearchRepos = (value) => {
    const vcsId = form.getFieldValue('vcsId');
    vcsId && fetchRepos({ vcsId, q: value });
  };

  return <div className='form-wrapper' style={{ width: 600 }}>
    <Form
      form={form}
      {...FL}
      onFinish={onFinish}
      onValuesChange={onValuesChange}
    >
      <Form.Item
        label='代码仓库'
        name='vcsId'
        rules={[
          {
            required: true,
            message: '请选择'
          }
        ]}
      >
        <Select 
          placeholder='请选择代码仓库'
          showSearch={true}
          optionFilterProp='children'
          notFoundContent={(
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              imageStyle={{ height: 60 }}
              description={(
                <span>暂无数据，&nbsp;<a onClick={opVcsModal}>创建代码仓库</a></span>
              )}
            />
          )}
        >
          {vcsList.map(it => <Option value={it.id}>{it.name}</Option>)}
        </Select>
      </Form.Item>
      <Form.Item
        label='仓库名称'
        name='repoId'
        rules={[
          {
            required: true,
            message: '请选择'
          }
        ]}
      >
        <Select 
          showSearch={true}
          filterOption={false}
          onDropdownVisibleChange={(open) => open && onSearchRepos()}
          onSearch={onSearchRepos}
          placeholder='请输入仓库名称搜索'
        >
          {repos.map(it => <Option value={it.id}>{it.fullName}</Option>)}
        </Select>
      </Form.Item>
      <Form.Item
        name='repoFullName'
        hidden={true}
      >
        <Input />
      </Form.Item>
      <Form.Item
        label='分支/标签'
        name='repoRevision'
        rules={[
          {
            required: true,
            message: '请选择'
          }
        ]}
      >
        <Select 
          showSearch={true}
          optionFilterProp='children'
          placeholder='请选择分支'
        >
          <OptGroup label='分支'>
            {repoBranches.map(it => <Option value={it.name}>{it.name}</Option>)}
          </OptGroup>
          <OptGroup label='标签'>
            {repoTags.map(it => <Option value={it.name}>{it.name}</Option>)}
          </OptGroup>
        </Select>
      </Form.Item>
      <Form.Item label='工作目录' wrapperCol={{ span: 18 }}>
        <Row>
          <Col flex='14'>
            <Form.Item
              noStyle={true}
              name='workdir'
            >
              <Input placeholder='请注意工作目录注意事项' />
            </Form.Item>
          </Col>
          <Col flex='4'>
            <Tooltip title='Terraform执行时的工作目录，不填时默认为代码仓库的根目录，指定的工作目录必须是代码仓库中存在的目录，否则执行部署时将会失败'>
              <QuestionCircleOutlined style={{ fontSize: 16, marginLeft: 12, marginTop: 8, color: '#898989' }}/>
            </Tooltip>
          </Col>
        </Row>
      </Form.Item>
      <Form.Item label='Terraform版本' required={true} wrapperCol={{ span: 18 }}>
        <Row>
          <Col flex='14'>
            <Form.Item
              noStyle={true}
              name='tfVersion'
              rules={[
                {
                  required: true,
                  message: '请选择'
                }
              ]}
            >
              <Select placeholder='请选择Terraform版本'>
                {
                  tfversionOptionList.map(it => <Option key={it} value={it}>v{it}</Option>)
                }
              </Select>
            </Form.Item>
          </Col>
          <Col flex='4'>
            <Tooltip title='当前平台只支持 Terraform v1.15.6，旧版本模板在编辑或执行时会自动收敛到该版本。'>
              <QuestionCircleOutlined style={{ fontSize: 16, marginLeft: 12, marginTop: 8, color: '#898989' }}/>
            </Tooltip>
          </Col>
        </Row>
      </Form.Item>
      <Form.Item wrapperCol={{ offset: 6, span: 14 }} style={{ paddingTop: 24, marginBottom: 0 }}>
        <Space size={24}>
          {
            opType === 'add' ? (
              <>
                <Button type='primary' htmlType={'submit'}>下一步</Button>
              </>
            ) : (
              <>
                <Button className='ant-btn-tertiary' onClick={goCTlist}>取消</Button>
                <Button type='primary' htmlType={'submit'} loading={saveLoading}>提交</Button>
              </>
            )
          }
        </Space>
      </Form.Item>
    </Form>
    {
      vcsVisible && <OpModal
        visible={vcsVisible}
        toggleVisible={clVcsModal}
        opt={'add'}
        operation={vcsOperation}
      />
    }
  </div>;
};

export default memo(Repo, (prevProps, nextProps) => {
  const preFormData = prevProps.ctData[prevProps.type];
  const nextFormData = nextProps.ctData[nextProps.type];
  // 判断前后表单数据源值是否相等 从而避免额外渲染业务
  if (isEqual(preFormData, nextFormData)) {
    return true;
  }
  return false;
});
