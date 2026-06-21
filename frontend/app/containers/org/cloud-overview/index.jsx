import React, { useMemo } from 'react';
import { Link } from 'react-router-dom';
import {
  Button,
  Empty,
  Progress,
  Space,
  Spin,
  Table,
  Tag,
  Typography
} from 'antd';
import {
  AppstoreOutlined,
  CloudServerOutlined,
  DatabaseOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SyncOutlined,
  WarningOutlined
} from '@ant-design/icons';
import { useRequest } from 'ahooks';
import moment from 'moment';
import PageHeader from 'components/pageHeader';
import Layout from 'components/common/layout';
import { requestWrapper } from 'utils/request';
import cloudOverviewAPI from 'services/cloud-overview';
import styles from './styles.less';

const { Text } = Typography;

const providerMap = {
  aws: 'AWS',
  oci: 'OCI',
  alicloud: 'AliCloud',
  azure: 'Azure',
  gcp: 'GCP',
  tencentcloud: '腾讯云',
  huawei: '华为云'
};

const sourceMap = {
  iac_resource: 'IaC 资源',
  cloud_collect: '云采集'
};

const riskMap = {
  low: '低',
  medium: '中',
  high: '高',
  critical: '严重'
};

const taskStatusMap = {
  pending: '等待中',
  approving: '待审批',
  running: '运行中',
  complete: '完成',
  failed: '失败',
  rejected: '已驳回'
};

const taskStatusColorMap = {
  pending: 'default',
  approving: 'warning',
  running: 'processing',
  complete: 'success',
  failed: 'error',
  rejected: 'default'
};

const actionLevelColorMap = {
  success: 'success',
  warning: 'warning',
  error: 'error'
};

const renderTime = (value) => !value || String(value).indexOf('0001-01-01') === 0 ? '-' : moment(value).format('YYYY-MM-DD HH:mm:ss');
const providerLabel = (value) => providerMap[value] || value || '-';
const sourceLabel = (value) => sourceMap[value] || value || '-';
const riskLabel = (value) => riskMap[value] || value || '-';
const numberText = (value) => Number(value || 0).toLocaleString();
const percent = (value) => Math.max(0, Math.min(100, Number(value || 0)));
const percentText = (value) => `${percent(value).toFixed(1)}%`;

const MetricItem = ({ icon, title, value, suffix, description, tone }) => (
  <div className={`${styles.metricItem} ${tone ? styles[tone] : ''}`}>
    <div className={styles.metricIcon}>{icon}</div>
    <div className={styles.metricBody}>
      <Text type='secondary'>{title}</Text>
      <div className={styles.metricValue}>{numberText(value)}{suffix || ''}</div>
      <Text type='secondary'>{description}</Text>
    </div>
  </div>
);

const MiniBars = ({ data = [], labelRender }) => {
  if (!data.length) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无数据'/>;
  }
  const max = Math.max(...data.map((item) => Number(item.count || 0)), 1);
  return (
    <Space direction='vertical' size={10} style={{ width: '100%', display: 'flex' }}>
      {data.map((item) => (
        <div className={styles.miniBar} key={item.key || item.label}>
          <div className={styles.miniBarMeta}>
            <Text>{labelRender ? labelRender(item.key || item.label) : item.label || item.key}</Text>
            <Text type='secondary'>{numberText(item.count)}</Text>
          </div>
          <div className={styles.miniBarTrack}>
            <div className={styles.miniBarInner} style={{ width: `${Number(item.count || 0) / max * 100}%` }}/>
          </div>
        </div>
      ))}
    </Space>
  );
};

const CloudOverviewPage = ({ match }) => {
  const { orgId } = match.params || {};
  const {
    loading,
    data = {},
    run: fetchOverview
  } = useRequest(
    () => requestWrapper(cloudOverviewAPI.overview.bind(null, { orgId })),
    {
      refreshDeps: [orgId]
    }
  );

  const metrics = data.metrics || {};
  const providers = data.providers || [];
  const actions = data.actions || [];
  const recentSyncTasks = data.recentSyncTasks || [];
  const coverageRate = percent(metrics.coverageRate);
  const governanceRate = percent(metrics.governanceRate);
  const cloudOnlyRate = percent(metrics.cloudOnlyRate);
  const accountReadyRate = metrics.accountEnabled ? percent(metrics.accountReady / metrics.accountEnabled * 100) : 0;

  const providerColumns = useMemo(() => [
    {
      dataIndex: 'provider',
      title: '云厂商',
      width: 120,
      render: (text) => <Tag color='blue'>{providerLabel(text)}</Tag>
    },
    {
      dataIndex: 'accountCount',
      title: '账号',
      width: 90,
      render: (value, record) => `${numberText(record.enabledAccountCount)} / ${numberText(value)}`
    },
    {
      dataIndex: 'invalidAccountCount',
      title: '异常账号',
      width: 100,
      render: (value) => value ? <Tag color='error'>{numberText(value)}</Tag> : 0
    },
    {
      dataIndex: 'assetCount',
      title: '资产',
      width: 100,
      render: numberText
    },
    {
      dataIndex: 'iacManagedAssetCount',
      title: 'IaC覆盖',
      width: 130,
      render: (value, record) => (
        <Space size={4}>
          <Text>{numberText(value)}</Text>
          <Text type='secondary'>/{numberText(record.assetCount)}</Text>
        </Space>
      )
    },
    {
      dataIndex: 'cloudOnlyAssetCount',
      title: '未纳管',
      width: 100,
      render: (value) => value ? <Tag color='warning'>{numberText(value)}</Tag> : 0
    },
    {
      dataIndex: 'lastSyncAt',
      title: '最近同步',
      width: 180,
      render: renderTime
    }
  ], []);

  const taskColumns = useMemo(() => [
    {
      dataIndex: 'accountName',
      title: '账号',
      width: 180,
      ellipsis: true,
      render: (text) => text || '-'
    },
    {
      dataIndex: 'provider',
      title: '云厂商',
      width: 100,
      render: (text) => text ? <Tag color='blue'>{providerLabel(text)}</Tag> : '-'
    },
    {
      dataIndex: 'status',
      title: '状态',
      width: 100,
      render: (text) => <Tag color={taskStatusColorMap[text]}>{taskStatusMap[text] || text || '-'}</Tag>
    },
    {
      dataIndex: 'stats',
      title: '统计',
      width: 240,
      render: (stats = {}) => `采集 ${stats.collected || 0} / 新增 ${stats.created || 0} / 更新 ${stats.updated || 0} / 跳过 ${stats.skipped || 0}`
    },
    {
      dataIndex: 'startedAt',
      title: '开始时间',
      width: 180,
      render: renderTime
    },
    {
      dataIndex: 'endedAt',
      title: '结束时间',
      width: 180,
      render: renderTime
    }
  ], []);

  return (
    <Layout
      extraHeader={<PageHeader title='多云总览' breadcrumb={true}/>}
    >
      <Spin spinning={loading}>
        <Space direction='vertical' size='middle' style={{ width: '100%', display: 'flex' }}>
          <div className='idcos-card'>
            <div className={styles.headerRow}>
              <div>
                <Text strong={true}>只读治理底座</Text>
                <div className={styles.headerDesc}>账号、资产、同步和治理缺口的组织级视图</div>
              </div>
              <Button icon={<ReloadOutlined/>} onClick={fetchOverview}>刷新</Button>
            </div>
            <div className={styles.metricGrid}>
              <MetricItem
                icon={<CloudServerOutlined/>}
                title='云账号'
                value={metrics.accountTotal}
                description={`启用 ${numberText(metrics.accountEnabled)}，有效 ${numberText(metrics.accountReady)}`}
              />
              <MetricItem
                icon={<DatabaseOutlined/>}
                title='CMDB 资产'
                value={metrics.assetTotal}
                description={`云采集 ${numberText(metrics.cloudCollectedAssets)}，IaC覆盖 ${numberText(metrics.iacManagedAssets)}`}
              />
              <MetricItem
                icon={<AppstoreOutlined/>}
                title='IaC 覆盖'
                value={metrics.iacManagedAssets}
                description={`直接 ${numberText(metrics.iacDirectAssets)}，关联 ${numberText(metrics.iacLinkedAssets)}`}
                tone={coverageRate < 80 ? 'warningTone' : ''}
              />
              <MetricItem
                icon={<WarningOutlined/>}
                title='治理缺口'
                value={metrics.cloudOnlyAssets}
                description={`无负责人 ${numberText(metrics.unownedAssets)}，高风险 ${numberText(metrics.highRiskAssets)}`}
                tone={metrics.cloudOnlyAssets || metrics.highRiskAssets ? 'warningTone' : ''}
              />
              <MetricItem
                icon={<SyncOutlined/>}
                title='同步健康'
                value={metrics.runningSyncTasks}
                description={`近24小时失败 ${numberText(metrics.failedSyncTasks)}，最近完成 ${renderTime(metrics.lastSyncAt)}`}
                tone={metrics.failedSyncTasks ? 'errorTone' : ''}
              />
            </div>
            <div className={styles.progressGrid}>
              <div className={styles.progressItem}>
                <div className={styles.progressMeta}>
                  <Text>IaC 覆盖率</Text>
                  <Text type='secondary'>{percentText(coverageRate)}</Text>
                </div>
                <Progress percent={coverageRate} showInfo={false} strokeColor='#2f7de1'/>
              </div>
              <div className={styles.progressItem}>
                <div className={styles.progressMeta}>
                  <Text>启用账号有效率</Text>
                  <Text type='secondary'>{accountReadyRate.toFixed(1)}%</Text>
                </div>
                <Progress percent={accountReadyRate} showInfo={false} strokeColor='#2ca58d'/>
              </div>
              <div className={styles.progressItem}>
                <div className={styles.progressMeta}>
                  <Text>治理关联率</Text>
                  <Text type='secondary'>{percentText(governanceRate)}</Text>
                </div>
                <Progress percent={governanceRate} showInfo={false} strokeColor='#6f8f2f'/>
              </div>
              <div className={styles.progressItem}>
                <div className={styles.progressMeta}>
                  <Text>云上未纳管率</Text>
                  <Text type='secondary'>{percentText(cloudOnlyRate)}</Text>
                </div>
                <Progress percent={cloudOnlyRate} showInfo={false} strokeColor={cloudOnlyRate > 0 ? '#d46b08' : '#2ca58d'}/>
              </div>
            </div>
          </div>

          <div className='idcos-card'>
            <div className={styles.sectionTitle}>
              <AppstoreOutlined/>
              <Text strong={true}>IaC 治理视图</Text>
            </div>
            <div className={styles.iacGrid}>
              <div className={styles.iacItem}>
                <Text type='secondary'>直接来自 IaC</Text>
                <div className={styles.iacValue}>{numberText(metrics.iacDirectAssets)}</div>
                <Text type='secondary'>由环境资源回填到 CMDB</Text>
              </div>
              <div className={styles.iacItem}>
                <Text type='secondary'>云采集已关联 IaC</Text>
                <div className={styles.iacValue}>{numberText(metrics.iacLinkedAssets)}</div>
                <Text type='secondary'>云上资源已绑定 IaC 资源 ID</Text>
              </div>
              <div className={styles.iacItem}>
                <Text type='secondary'>云采集已绑定项目/环境</Text>
                <div className={styles.iacValue}>{numberText(metrics.cloudLinkedAssets)}</div>
                <Text type='secondary'>进入项目治理范围的云资产</Text>
              </div>
              <div className={`${styles.iacItem} ${metrics.cloudOnlyAssets ? styles.warningIacItem : ''}`}>
                <Text type='secondary'>云上未纳管</Text>
                <div className={styles.iacValue}>{numberText(metrics.cloudOnlyAssets)}</div>
                <Link to={`/org/${orgId}/m-cloud-assets?managedBy=cloud_only`}>查看并治理</Link>
              </div>
            </div>
          </div>

          <div className={styles.twoColumn}>
            <div className='idcos-card'>
              <div className={styles.sectionTitle}>
                <AppstoreOutlined/>
                <Text strong={true}>Provider 覆盖</Text>
              </div>
              <Table
                rowKey='provider'
                size='small'
                columns={providerColumns}
                dataSource={providers}
                pagination={false}
                scroll={{ x: 'max-content' }}
              />
            </div>
            <div className='idcos-card'>
              <div className={styles.sectionTitle}>
                <SafetyCertificateOutlined/>
                <Text strong={true}>治理事项</Text>
              </div>
              <Space direction='vertical' size={10} style={{ width: '100%', display: 'flex' }}>
                {actions.map((item) => (
                  <Link to={item.target} className={styles.actionItem} key={item.key}>
                    <div>
                      <Tag color={actionLevelColorMap[item.level] || 'default'}>{numberText(item.count)}</Tag>
                      <Text strong={true}>{item.title}</Text>
                    </div>
                    <Text type='secondary'>{item.description}</Text>
                  </Link>
                ))}
              </Space>
            </div>
          </div>

          <div className={styles.threeColumn}>
            <div className='idcos-card'>
              <div className={styles.sectionTitle}><Text strong={true}>资产来源</Text></div>
              <MiniBars data={data.assetSources || []} labelRender={sourceLabel}/>
            </div>
            <div className='idcos-card'>
              <div className={styles.sectionTitle}><Text strong={true}>资产类型 Top 10</Text></div>
              <MiniBars data={data.assetTypes || []}/>
            </div>
            <div className='idcos-card'>
              <div className={styles.sectionTitle}><Text strong={true}>风险分布</Text></div>
              <MiniBars data={data.riskLevels || []} labelRender={riskLabel}/>
            </div>
          </div>

          <div className='idcos-card'>
            <div className={styles.sectionTitle}>
              <SyncOutlined/>
              <Text strong={true}>最近云采集任务</Text>
            </div>
            <Table
              rowKey='id'
              size='small'
              columns={taskColumns}
              dataSource={recentSyncTasks}
              pagination={false}
              scroll={{ x: 'max-content' }}
              locale={{
                emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无同步任务'/>
              }}
            />
          </div>
        </Space>
      </Spin>
    </Layout>
  );
};

export default CloudOverviewPage;
