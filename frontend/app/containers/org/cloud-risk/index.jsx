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
  Timeline,
  Typography
} from 'antd';
import { CheckCircleOutlined, ExceptionOutlined, ReloadOutlined, SyncOutlined, ToolOutlined } from '@ant-design/icons';
import { useRequest } from 'ahooks';
import moment from 'moment';
import { Link } from 'react-router-dom';
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

const taskStatusMap = {
  pending: { label: '等待中', color: 'default' },
  running: { label: '执行中', color: 'processing' },
  approving: { label: '待审批', color: 'warning' },
  rejected: { label: '已驳回', color: 'error' },
  failed: { label: '失败', color: 'error' },
  complete: { label: '完成', color: 'success' },
  aborted: { label: '已取消', color: 'default' }
};

const driftAutoRepairSkipReasonMap = {
  missing_env_id: '缺少环境 ID',
  env_not_found: '环境不存在或无权限',
  env_lookup_failed: '环境查询失败',
  env_not_active: '环境不是活跃状态',
  env_locked: '环境已锁定',
  cron_drift_disabled: '未开启漂移检测',
  auto_repair_disabled: '未开启自动修复',
  missing_source_task: '缺少源部署任务',
  pending_task_lookup_failed: '查询待处理漂移任务失败',
  pending_drift_task_exists: '已有漂移任务在处理',
  source_task_lookup_failed: '源部署任务查询失败',
  clone_drift_task_failed: '创建自动修复任务失败',
  max_retry_attempts_reached: '已达到自动修复重试上限',
  retry_task_exists: '已存在自动修复重试任务',
  clone_retry_task_failed: '创建自动修复重试任务失败',
  not_checked: '尚未检查'
};

const driftAutoRepairRollbackStrategyMap = {
  not_evaluated: '未评估',
  wait_task_finished: '等待任务完成',
  no_platform_rollback_required: '无需平台自动回滚',
  gitops_iac_change_or_manual_review_required: '需 GitOps/IaC 变更或人工评审'
};

const driftAutoRepairRollbackSafetyMap = {
  low: { label: '低', color: 'success' },
  medium: { label: '中', color: 'warning' },
  high: { label: '高', color: 'error' }
};

const driftAutoRepairApprovalModeMap = {
  inherit_env: '继承环境自动审批',
  require_approval: '强制审批',
  auto_approve: '自动审批',
  retry_require_approval: '失败重试需审批'
};

const driftAutoRepairApprovalSourceMap = {
  env_auto_approval: '环境自动审批',
  default_retry_guardrail: '默认重试保护',
  env_extra_data: '环境扩展策略'
};

const driftAutoRepairApprovalReasonMap = {
  env_auto_approval_enabled: '环境已开启自动审批',
  env_auto_approval_disabled: '环境未开启自动审批',
  retry_after_failed_auto_repair_requires_approval: '自动修复失败后重试默认需要审批',
  policy_requires_approval: '策略要求审批',
  policy_auto_approve: '策略允许自动审批',
  policy_retry_requires_approval: '策略要求失败重试审批',
  policy_inherits_env_auto_approval: '策略继承环境自动审批',
  policy_auto_approve_override: '策略覆盖为自动审批',
  policy_auto_approve_disabled: '策略关闭自动审批',
  policy_retry_auto_approve_override: '策略覆盖为失败重试自动审批',
  policy_retry_auto_approve_disabled: '策略关闭失败重试自动审批',
  invalid_policy_mode_requires_approval: '策略模式无效，已按需审批处理'
};

const driftAutoRepairTimelineStatusMap = {
  checked: { label: '已检查', color: 'blue' },
  triggered: { label: '已触发', color: 'green' },
  skipped: { label: '已跳过', color: 'gray' },
  found: { label: '已定位', color: 'green' },
  auto_approved: { label: '自动审批', color: 'green' },
  approval_required: { label: '需要审批', color: 'orange' },
  pending: { label: '待审批', color: 'orange' },
  approving: { label: '待审批', color: 'orange' },
  running: { label: '执行中', color: 'blue' },
  complete: { label: '完成', color: 'green' },
  failed: { label: '失败', color: 'red' },
  aborted: { label: '已取消', color: 'gray' },
  rejected: { label: '已驳回', color: 'red' },
  resolved: { label: '已解决', color: 'green' },
  open: { label: '待处理', color: 'red' },
  no_rollback_required: { label: '无需回滚', color: 'green' },
  rollback_review_required: { label: '需回滚评审', color: 'orange' },
  due_soon: { label: '即将超时', color: 'orange' },
  breached: { label: '已超时', color: 'red' },
  not_required: { label: '无需SLA', color: 'default' }
};

const driftAutoRepairSlaActionMap = {
  none: '无需处理',
  approve_repair: '审批自动修复任务',
  approve_retry: '审批重试任务',
  review_rollback: '回滚评审'
};

const driftAutoRepairRecommendationPriorityMap = {
  critical: { label: '严重', color: 'red' },
  high: { label: '高', color: 'orange' },
  medium: { label: '中', color: 'blue' },
  low: { label: '低', color: 'default' }
};

const driftAutoRepairRecommendationActionMap = {
  open_env: '查看环境',
  enable_drift_detection: '开启漂移检测',
  enable_auto_repair: '开启自动修复',
  unlock_env: '解除环境锁定',
  check_env_status: '检查环境状态',
  run_apply_once: '执行一次部署',
  check_pending_drift_task: '查看待处理任务',
  review_auto_repair_config: '检查自动修复配置',
  open_repair_task: '查看修复任务',
  approve_task: '审批任务',
  inspect_task_logs: '查看失败日志',
  approve_retry_task: '审批重试任务',
  review_rollback: '回滚评审',
  create_gitops_pr: '提交 GitOps/IaC PR',
  verify_drift_clean: '确认漂移闭环'
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
const taskStatusTag = (value) => {
  const item = taskStatusMap[value] || { label: value || '-', color: 'default' };
  return <Tag color={item.color}>{item.label}</Tag>;
};
const rollbackSafetyTag = (value) => {
  const item = driftAutoRepairRollbackSafetyMap[value] || { label: value || '-', color: 'default' };
  return <Tag color={item.color}>{item.label}</Tag>;
};
const recommendationPriorityTag = (value) => {
  const item = driftAutoRepairRecommendationPriorityMap[value] || { label: value || '-', color: 'default' };
  return <Tag color={item.color}>{item.label}</Tag>;
};
const boolText = (value) => value === true ? '是' : value === false ? '否' : '-';
const arrayText = (value) => Array.isArray(value) && value.length > 0 ? value.join('、') : '-';
const timelineStatusTag = (value) => {
  const item = driftAutoRepairTimelineStatusMap[value] || taskStatusMap[value] || statusMap[value] || { label: value || '-', color: 'default' };
  return <Tag color={item.color}>{item.label}</Tag>;
};
const timelineColor = (value) => {
  const item = driftAutoRepairTimelineStatusMap[value] || taskStatusMap[value] || statusMap[value];
  if (!item) {
    return 'blue';
  }
  if (item.color === 'error') {
    return 'red';
  }
  if (item.color === 'warning') {
    return 'orange';
  }
  if (item.color === 'success') {
    return 'green';
  }
  if (item.color === 'processing') {
    return 'blue';
  }
  return item.color || 'blue';
};
const timelineMetaText = (item = {}) => {
  const parts = [];
  if (item.taskId) {
    parts.push(`任务 ${item.taskId}`);
  }
  if (item.riskStatus) {
    const riskStatus = statusMap[item.riskStatus];
    parts.push(`风险 ${riskStatus ? riskStatus.label : item.riskStatus}`);
  }
  if (item.mode) {
    parts.push(driftAutoRepairApprovalModeMap[item.mode] || item.mode);
  }
  if (item.reason) {
    parts.push(driftAutoRepairSkipReasonMap[item.reason] || driftAutoRepairApprovalReasonMap[item.reason] || item.reason);
  }
  if (item.attempt) {
    parts.push(`第 ${item.attempt}/${item.maxAttempts || '-'} 次`);
  }
  if (item.safetyLevel) {
    const safetyLevel = driftAutoRepairRollbackSafetyMap[item.safetyLevel];
    parts.push(`安全级别 ${safetyLevel ? safetyLevel.label : item.safetyLevel}`);
  }
  if (item.message) {
    parts.push(item.message);
  }
  if (item.dueAt) {
    parts.push(`截止 ${renderTime(item.dueAt)}`);
  }
  if (item.minutesRemaining !== undefined && item.minutesRemaining !== null && item.minutesRemaining !== '') {
    parts.push(`剩余 ${item.minutesRemaining} 分钟`);
  }
  return parts.join(' | ');
};
const recommendationReasonText = (value) => driftAutoRepairSkipReasonMap[value] || driftAutoRepairApprovalReasonMap[value] || value || '-';
const jsonText = (value) => {
  if (!value || Object.keys(value || {}).length === 0) {
    return '-';
  }
  return JSON.stringify(value, null, 2);
};

const JsonBlock = ({ value }) => (
  <pre className={styles.jsonBlock}>{jsonText(value)}</pre>
);

const hasDriftAutoRepairEvidence = (evidence = {}) => Object.keys(evidence || {}).some((key) => key.indexOf('driftAutoRepair') === 0);

const DriftAutoRepairDetail = ({ detail = {}, orgId, onAdoptRecommendation, adoptingRecommendation }) => {
  const evidence = detail.evidence || {};
  if (!hasDriftAutoRepairEvidence(evidence)) {
    return null;
  }
  const projectId = detail.projectId || evidence.driftAutoRepairProjectId;
  const envId = detail.envId || evidence.driftAutoRepairEnvId;
  const taskId = evidence.driftAutoRepairTaskId;
  const sourceTaskId = evidence.driftAutoRepairSourceTaskId;
  const retryTaskId = evidence.driftAutoRepairRetryTaskId;
  const lastFailedTaskId = evidence.driftAutoRepairLastFailedTaskId;
  const envUrl = orgId && projectId && envId ? `/org/${orgId}/project/${projectId}/m-project-env/detail/${envId}` : '';
  const taskUrl = envUrl && taskId ? `${envUrl}/task/${taskId}` : '';
  const sourceTaskUrl = envUrl && sourceTaskId ? `${envUrl}/task/${sourceTaskId}` : '';
  const retryTaskUrl = envUrl && retryTaskId ? `${envUrl}/task/${retryTaskId}` : '';
  const lastFailedTaskUrl = envUrl && lastFailedTaskId ? `${envUrl}/task/${lastFailedTaskId}` : '';
  const skippedReason = driftAutoRepairSkipReasonMap[evidence.driftAutoRepairSkippedReason] || evidence.driftAutoRepairSkippedReason || '';
  const retrySkippedReason = driftAutoRepairSkipReasonMap[evidence.driftAutoRepairRetrySkippedReason] || evidence.driftAutoRepairRetrySkippedReason || '';
  const taskFailed = evidence.driftAutoRepairTaskFailed === true;
  const taskCompleted = evidence.driftAutoRepairTaskCompleted === true;
  const retryApprovalRequested = evidence.driftAutoRepairRetryApprovalRequested === true;
  const retryPendingApproval = evidence.driftAutoRepairTaskRetryPendingApproval === true;
  const rollbackRequired = evidence.driftAutoRepairRollbackRequired === true;
  const rollbackStrategy = driftAutoRepairRollbackStrategyMap[evidence.driftAutoRepairRollbackStrategy] || evidence.driftAutoRepairRollbackStrategy || '-';
  const approvalMode = driftAutoRepairApprovalModeMap[evidence.driftAutoRepairApprovalPolicyMode] || evidence.driftAutoRepairApprovalPolicyMode || '-';
  const approvalSource = driftAutoRepairApprovalSourceMap[evidence.driftAutoRepairApprovalPolicySource] || evidence.driftAutoRepairApprovalPolicySource || '-';
  const approvalReason = driftAutoRepairApprovalReasonMap[evidence.driftAutoRepairApprovalReason] || evidence.driftAutoRepairApprovalReason || '-';
  const repairTimeline = Array.isArray(evidence.driftAutoRepairTimeline) ? evidence.driftAutoRepairTimeline : [];
  const recommendations = Array.isArray(evidence.driftAutoRepairRecommendations) ? evidence.driftAutoRepairRecommendations : [];
  const slaAction = evidence.driftAutoRepairSlaAction || 'none';
  const slaStatus = evidence.driftAutoRepairSlaStatus || 'not_required';
  const slaEscalationRequired = evidence.driftAutoRepairSlaEscalationRequired === true;
  const recommendationLink = (item = {}) => {
    if (item.targetType === 'env' && envUrl) {
      return <Link to={envUrl}>{item.targetId || envId}</Link>;
    }
    if (item.targetType === 'task' && envUrl && item.targetId) {
      return <Link to={`${envUrl}/task/${item.targetId}`}>{item.targetId}</Link>;
    }
    return item.targetId || '-';
  };
  const recommendationAlertType = (priority) => priority === 'critical' || priority === 'high' ? 'warning' : 'info';

  return (
    <div className={styles.detailSection}>
      <Text strong={true}>漂移自动修复</Text>
      {evidence.driftAutoRepairError && (
        <Alert
          showIcon={true}
          type='error'
          message='自动修复触发异常'
          description={evidence.driftAutoRepairError}
        />
      )}
      {skippedReason && !evidence.driftAutoRepairTriggered && (
        <Alert
          showIcon={true}
          type='warning'
          message='自动修复未触发'
          description={skippedReason}
        />
      )}
      {taskFailed && (
        <Alert
          showIcon={true}
          type='error'
          message='自动修复任务失败'
          description={evidence.driftAutoRepairTaskMessage || '任务结束状态为失败、取消或驳回'}
        />
      )}
      {retryPendingApproval && (
        <Alert
          showIcon={true}
          type='warning'
          message='自动修复失败后已创建重试审批'
          description='待审批通过后继续执行 drift apply 重试任务'
        />
      )}
      {rollbackRequired && (
        <Alert
          showIcon={true}
          type='warning'
          message='需要回滚评审'
          description={evidence.driftAutoRepairRollbackHint || '自动修复未完全成功，请先评审云端资源、Terraform state 和 IaC 代码'}
        />
      )}
      {retrySkippedReason && !retryApprovalRequested && evidence.driftAutoRepairRetryApprovalRequired && (
        <Alert
          showIcon={true}
          type='warning'
          message='自动修复失败后未创建重试'
          description={retrySkippedReason}
        />
      )}
      {slaEscalationRequired && (
        <Alert
          showIcon={true}
          type='error'
          message='审批/评审 SLA 已升级'
          description={`处理动作：${driftAutoRepairSlaActionMap[slaAction] || slaAction}，截止时间：${renderTime(evidence.driftAutoRepairSlaDueAt)}`}
        />
      )}
      {!slaEscalationRequired && slaAction !== 'none' && slaStatus !== 'not_required' && (
        <Alert
          showIcon={true}
          type={slaStatus === 'due_soon' ? 'warning' : 'info'}
          message='审批/评审 SLA'
          description={`处理动作：${driftAutoRepairSlaActionMap[slaAction] || slaAction}，截止时间：${renderTime(evidence.driftAutoRepairSlaDueAt)}`}
        />
      )}
      {taskCompleted && (
        <Alert
          showIcon={true}
          type='success'
          message='自动修复任务已完成'
          description='风险状态已根据 drift apply 任务结果同步'
        />
      )}
      {recommendations.length > 0 && (
        <div>
          <Text strong={true}>关联推荐</Text>
          <Space direction='vertical' size='small' style={{ width: '100%', marginTop: 12 }}>
            {recommendations.map((item, index) => (
              <Alert
                key={`${item.action || 'recommendation'}-${index}`}
                showIcon={true}
                type={recommendationAlertType(item.priority)}
                message={(
                  <Space size='small'>
                    <Text>{item.title || driftAutoRepairRecommendationActionMap[item.action] || item.action || '-'}</Text>
                    {recommendationPriorityTag(item.priority)}
                  </Space>
                )}
                description={(
                  <Space size='middle' wrap={true}>
                    <span>{driftAutoRepairRecommendationActionMap[item.action] || item.action || '-'}</span>
                    <span>目标：{recommendationLink(item)}</span>
                    <span>原因：{recommendationReasonText(item.reason)}</span>
                    {item.adopted === true ? (
                      <Tag color='success'>已采纳 {renderTime(item.adoptedAt)}</Tag>
                    ) : (
                      <Popconfirm title='确认已采纳该推荐？' onConfirm={() => onAdoptRecommendation && onAdoptRecommendation(item)}>
                        <Button size='small' icon={<CheckCircleOutlined/>} loading={adoptingRecommendation}>确认采纳</Button>
                      </Popconfirm>
                    )}
                  </Space>
                )}
              />
            ))}
          </Space>
        </div>
      )}
      {repairTimeline.length > 0 && (
        <div>
          <Text strong={true}>处理时间线</Text>
          <Timeline style={{ marginTop: 12 }}>
            {repairTimeline.map((item, index) => (
              <Timeline.Item color={timelineColor(item.status)} key={`${item.stage || 'stage'}-${index}`}>
                <Space direction='vertical' size={2}>
                  <Space size='small'>
                    <Text>{item.title || item.stage || '-'}</Text>
                    {timelineStatusTag(item.status)}
                  </Space>
                  <Text type='secondary'>{renderTime(item.time)}</Text>
                  {timelineMetaText(item) && <Text type='secondary'>{timelineMetaText(item)}</Text>}
                </Space>
              </Timeline.Item>
            ))}
          </Timeline>
        </div>
      )}
      <Descriptions size='small' bordered={true} column={2}>
        <Descriptions.Item label='触发状态'>
          {evidence.driftAutoRepairTriggered ? <Tag color='success'>已触发</Tag> : <Tag>未触发</Tag>}
        </Descriptions.Item>
        <Descriptions.Item label='触发检查'>{renderTime(evidence.driftAutoRepairCheckedAt)}</Descriptions.Item>
        <Descriptions.Item label='环境'>
          {envUrl ? <Link to={envUrl}>{envId}</Link> : envId || '-'}
        </Descriptions.Item>
        <Descriptions.Item label='环境状态'>{evidence.driftAutoRepairEnvStatus || '-'}</Descriptions.Item>
        <Descriptions.Item label='漂移检测'>{boolText(evidence.driftAutoRepairOpenCronDrift)}</Descriptions.Item>
        <Descriptions.Item label='自动修复'>{boolText(evidence.driftAutoRepairEnabled)}</Descriptions.Item>
        <Descriptions.Item label='审批策略'>{approvalMode}</Descriptions.Item>
        <Descriptions.Item label='策略来源'>{approvalSource}</Descriptions.Item>
        <Descriptions.Item label='自动审批'>{boolText(evidence.driftAutoRepairApprovalAutoApprove)}</Descriptions.Item>
        <Descriptions.Item label='需要审批'>{boolText(evidence.driftAutoRepairApprovalRequired)}</Descriptions.Item>
        <Descriptions.Item label='审批角色'>{arrayText(evidence.driftAutoRepairApprovalRoles)}</Descriptions.Item>
        <Descriptions.Item label='策略原因'>{approvalReason}</Descriptions.Item>
        <Descriptions.Item label='源任务'>
          {sourceTaskUrl ? <Link to={sourceTaskUrl}>{sourceTaskId}</Link> : sourceTaskId || '-'}
        </Descriptions.Item>
        <Descriptions.Item label='修复任务'>
          {taskUrl ? <Link to={taskUrl}>{taskId}</Link> : taskId || '-'}
        </Descriptions.Item>
        <Descriptions.Item label='任务状态'>{taskStatusTag(evidence.driftAutoRepairTaskStatus)}</Descriptions.Item>
        <Descriptions.Item label='风险同步状态'>{statusTag(evidence.driftAutoRepairTaskRiskStatus)}</Descriptions.Item>
        <Descriptions.Item label='失败任务'>
          {lastFailedTaskUrl ? <Link to={lastFailedTaskUrl}>{lastFailedTaskId}</Link> : lastFailedTaskId || '-'}
        </Descriptions.Item>
        <Descriptions.Item label='失败状态'>{taskStatusTag(evidence.driftAutoRepairLastFailedTaskStatus)}</Descriptions.Item>
        <Descriptions.Item label='重试任务'>
          {retryTaskUrl ? <Link to={retryTaskUrl}>{retryTaskId}</Link> : retryTaskId || '-'}
        </Descriptions.Item>
        <Descriptions.Item label='重试审批'>{retryPendingApproval ? <Tag color='warning'>待审批</Tag> : retryApprovalRequested ? <Tag color='processing'>已创建</Tag> : evidence.driftAutoRepairRetryApprovalRequired ? <Tag>未创建</Tag> : '-'}</Descriptions.Item>
        <Descriptions.Item label='重试次数'>{evidence.driftAutoRepairRetryAttempt ? `${evidence.driftAutoRepairRetryAttempt}/${evidence.driftAutoRepairRetryMaxAttempts || '-'}` : '-'}</Descriptions.Item>
        <Descriptions.Item label='重试检查'>{renderTime(evidence.driftAutoRepairRetryCheckedAt)}</Descriptions.Item>
        <Descriptions.Item label='回滚策略'>{rollbackStrategy}</Descriptions.Item>
        <Descriptions.Item label='回滚安全级别'>{rollbackSafetyTag(evidence.driftAutoRepairRollbackSafetyLevel)}</Descriptions.Item>
        <Descriptions.Item label='需要回滚评审'>{boolText(evidence.driftAutoRepairRollbackRequired)}</Descriptions.Item>
        <Descriptions.Item label='疑似部分变更'>{boolText(evidence.driftAutoRepairRollbackPartialApplySuspected)}</Descriptions.Item>
        <Descriptions.Item label='回滚评估'>{renderTime(evidence.driftAutoRepairRollbackEvaluatedAt)}</Descriptions.Item>
        <Descriptions.Item label='回滚审批'>{boolText(evidence.driftAutoRepairRollbackRequiresApproval)}</Descriptions.Item>
        <Descriptions.Item label='SLA 动作'>{driftAutoRepairSlaActionMap[slaAction] || slaAction}</Descriptions.Item>
        <Descriptions.Item label='SLA 状态'>{timelineStatusTag(slaStatus)}</Descriptions.Item>
        <Descriptions.Item label='SLA 开始'>{renderTime(evidence.driftAutoRepairSlaStartedAt)}</Descriptions.Item>
        <Descriptions.Item label='SLA 截止'>{renderTime(evidence.driftAutoRepairSlaDueAt)}</Descriptions.Item>
        <Descriptions.Item label='SLA 剩余'>{evidence.driftAutoRepairSlaMinutesRemaining !== undefined ? `${evidence.driftAutoRepairSlaMinutesRemaining} 分钟` : '-'}</Descriptions.Item>
        <Descriptions.Item label='SLA 升级'>{boolText(evidence.driftAutoRepairSlaEscalationRequired)}</Descriptions.Item>
        <Descriptions.Item label='通知路由'>{arrayText(evidence.driftAutoRepairSlaNotificationRoutes)}</Descriptions.Item>
        <Descriptions.Item label='负责人'>{arrayText(evidence.driftAutoRepairSlaNotificationOwnerRoles)}</Descriptions.Item>
        <Descriptions.Item label='任务开始'>{renderTime(evidence.driftAutoRepairTaskStartedAt)}</Descriptions.Item>
        <Descriptions.Item label='任务结束'>{renderTime(evidence.driftAutoRepairTaskEndedAt)}</Descriptions.Item>
        <Descriptions.Item label='结果同步'>{renderTime(evidence.driftAutoRepairTaskLastSyncedAt)}</Descriptions.Item>
        <Descriptions.Item label='任务消息'>{evidence.driftAutoRepairTaskMessage || '-'}</Descriptions.Item>
        <Descriptions.Item label='回滚下一步' span={2}>{evidence.driftAutoRepairRollbackNextAction || '-'}</Descriptions.Item>
      </Descriptions>
    </div>
  );
};

const Metric = ({ label, value, tone }) => (
  <div className={`${styles.metric} ${tone ? styles[tone] : ''}`}>
    <div className={styles.metricValue}>{value || 0}</div>
    <div className={styles.metricLabel}>{label}</div>
  </div>
);

const RiskDetail = ({ detail = {}, orgId, loading, onMarkProgress, onResolve, onReopen, onSuppress, onRemediate, onAdoptRecommendation, updating, remediating, adoptingRecommendation }) => {
  if (loading) {
    return <Empty description='加载中'/>;
  }
  if (!detail.id) {
    return <Empty description='请选择风险'/>;
  }
  const canMarkProgress = detail.status === 'open';
  const canResolve = detail.status === 'open' || detail.status === 'in_progress' || detail.status === 'suppressed';
  const canReopen = detail.status === 'resolved' || detail.status === 'suppressed';
  const canRemediate = detail.status === 'open' || detail.status === 'in_progress' || detail.status === 'suppressed';
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
        {canRemediate && (
          <Button type='primary' icon={<ToolOutlined/>} loading={remediating} onClick={onRemediate}>发起整改</Button>
        )}
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
      <DriftAutoRepairDetail
        detail={detail}
        orgId={orgId}
        onAdoptRecommendation={onAdoptRecommendation}
        adoptingRecommendation={adoptingRecommendation}
      />
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

  const {
    loading: remediating,
    run: createRemediationTicket
  } = useRequest(
    ({ id }) => requestWrapper(cloudRiskAPI.createRemediationTicket.bind(null, { orgId, id }), {
      autoSuccess: true,
      successMessage: '整改工单已创建'
    }),
    {
      manual: true,
      onSuccess: () => {
        fetchList();
      }
    }
  );

  const {
    loading: adoptingRecommendation,
    run: adoptRecommendation
  } = useRequest(
    ({ id, item }) => requestWrapper(cloudRiskAPI.adoptRecommendation.bind(null, {
      orgId,
      id,
      action: item.action,
      targetType: item.targetType,
      targetId: item.targetId,
      comment: '页面确认采纳推荐'
    }), {
      autoSuccess: true,
      successMessage: '推荐采纳已记录'
    }),
    {
      manual: true,
      onSuccess: () => {
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
      width: 230,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          {record.status !== 'resolved' && (
            <a onClick={() => createRemediationTicket({ id: record.id })}>整改</a>
          )}
          <a onClick={() => updateStatus({ id: record.id, status: 'in_progress', comment: '页面标记处理中' })}>处理中</a>
          <a onClick={() => openSuppress(record)}>例外</a>
          <Popconfirm title='确认将该风险标记为已解决？' onConfirm={() => updateStatus({ id: record.id, status: 'resolved', comment: '页面标记已解决' })}>
            <a>解决</a>
          </Popconfirm>
        </Space>
      )
    }
  ], [ createRemediationTicket, updateStatus ]);

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
          loading={loading || updating || remediating || adoptingRecommendation}
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
          orgId={orgId}
          loading={loading}
          updating={updating}
          remediating={remediating}
          adoptingRecommendation={adoptingRecommendation}
          onRemediate={() => createRemediationTicket({ id: detail.id })}
          onAdoptRecommendation={(item) => adoptRecommendation({ id: detail.id, item })}
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
