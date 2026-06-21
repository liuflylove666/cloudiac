import { del, getWithArgs, post, put } from 'utils/xFetch2';

const cloudCostAPI = {
  summary: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/cost/summary', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  trends: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/cost/trends', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  records: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/cost/records', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  importRecords: ({ orgId, data }) => {
    return post('/api/v1/cloud/cost/import', data, {
      'IaC-Org-Id': orgId
    });
  },
  pullRecords: ({ orgId, data }) => {
    return post('/api/v1/cloud/cost/pull', data, {
      'IaC-Org-Id': orgId
    });
  },
  syncTasks: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/cost/sync-tasks', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  createSyncTask: ({ orgId, data }) => {
    return post('/api/v1/cloud/cost/sync-tasks', data, {
      'IaC-Org-Id': orgId
    });
  },
  syncTaskDetail: ({ orgId, id }) => {
    return getWithArgs(`/api/v1/cloud/cost/sync-tasks/${id}`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  retrySyncTask: ({ orgId, id }) => {
    return post(`/api/v1/cloud/cost/sync-tasks/${id}/retry`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  syncSchedules: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/cost/sync-schedules', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  createSyncSchedule: ({ orgId, data }) => {
    return post('/api/v1/cloud/cost/sync-schedules', data, {
      'IaC-Org-Id': orgId
    });
  },
  updateSyncSchedule: ({ orgId, id, data }) => {
    return put(`/api/v1/cloud/cost/sync-schedules/${id}`, data, {
      'IaC-Org-Id': orgId
    });
  },
  deleteSyncSchedule: ({ orgId, id }) => {
    return del(`/api/v1/cloud/cost/sync-schedules/${id}`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  runSyncSchedule: ({ orgId, id }) => {
    return post(`/api/v1/cloud/cost/sync-schedules/${id}/run`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  runDueSyncSchedules: ({ orgId, data }) => {
    return post('/api/v1/cloud/cost/sync-schedules/run-due', data || {}, {
      'IaC-Org-Id': orgId
    });
  },
  unmatched: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/cost/unmatched', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  insightSummary: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/cost/insights/summary', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  insights: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/cost/insights', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  updateInsightStatus: ({ orgId, id, data }) => {
    return put(`/api/v1/cloud/cost/insights/${id}/status`, data, {
      'IaC-Org-Id': orgId
    });
  },
  budgetSummary: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/budgets/summary', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  evaluateBudgets: ({ orgId, data }) => {
    return post('/api/v1/cloud/budgets/evaluate-due', data, {
      'IaC-Org-Id': orgId
    });
  },
  budgets: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/budgets', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  createBudget: ({ orgId, data }) => {
    return post('/api/v1/cloud/budgets', data, {
      'IaC-Org-Id': orgId
    });
  },
  updateBudget: ({ orgId, id, data }) => {
    return put(`/api/v1/cloud/budgets/${id}`, data, {
      'IaC-Org-Id': orgId
    });
  },
  deleteBudget: ({ orgId, id }) => {
    return del(`/api/v1/cloud/budgets/${id}`, {}, {
      'IaC-Org-Id': orgId
    });
  }
};

export default cloudCostAPI;
