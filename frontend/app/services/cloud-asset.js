import { getWithArgs, post, put } from 'utils/xFetch2';

const cloudAssetAPI = {
  coverage: ({ orgId }) => {
    return getWithArgs('/api/v1/cloud/assets/coverage', {}, {
      'IaC-Org-Id': orgId
    });
  },
  listAssets: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/assets', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  assetFilters: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/assets/filters', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  assetPermissions: ({ orgId }) => {
    return getWithArgs('/api/v1/cloud/assets/permissions', {}, {
      'IaC-Org-Id': orgId
    });
  },
  assetGovernanceReport: ({ orgId }) => {
    return getWithArgs('/api/v1/cloud/assets/governance-report', {}, {
      'IaC-Org-Id': orgId
    });
  },
  assetDetail: ({ orgId, id }) => {
    return getWithArgs(`/api/v1/cloud/assets/${id}`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  assetRelations: ({ orgId, id, ...restParams }) => {
    return getWithArgs(`/api/v1/cloud/assets/${id}/relations`, restParams, {
      'IaC-Org-Id': orgId
    });
  },
  securityRules: ({ orgId, id }) => {
    return getWithArgs(`/api/v1/cloud/assets/${id}/security-rules`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  assetActions: ({ orgId, id }) => {
    return getWithArgs(`/api/v1/cloud/assets/${id}/actions`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  dryRunAssetAction: ({ orgId, id, action, ...restParams }) => {
    return post(`/api/v1/cloud/assets/${id}/actions/${action}/dry-run`, restParams, {
      'IaC-Org-Id': orgId
    });
  },
  createAssetAction: ({ orgId, id, action, ...restParams }) => {
    return post(`/api/v1/cloud/assets/${id}/actions/${action}`, restParams, {
      'IaC-Org-Id': orgId
    });
  },
  updateAssetOwnership: ({ orgId, id, ...restParams }) => {
    return put(`/api/v1/cloud/assets/${id}/ownership`, restParams, {
      'IaC-Org-Id': orgId
    });
  },
  batchUpdateAssetOwnership: ({ orgId, ...restParams }) => {
    return put('/api/v1/cloud/assets/ownership', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  importAssets: ({ orgId, ...restParams }) => {
    return post('/api/v1/cloud/assets/import', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  backfillIacResources: ({ orgId }) => {
    return post('/api/v1/cloud/backfill/iac-resources', {}, {
      'IaC-Org-Id': orgId
    });
  },
  syncTasks: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/sync-tasks', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  syncTaskDetail: ({ orgId, id }) => {
    return getWithArgs(`/api/v1/cloud/sync-tasks/${id}`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  syncTaskRerunGroupDetail: ({ orgId, groupId }) => {
    return getWithArgs(`/api/v1/cloud/sync-task-rerun-groups/${groupId}`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  startSyncTask: ({ orgId, ...restParams }) => {
    return post('/api/v1/cloud/sync-tasks', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  batchRerunFailedSyncTasks: ({ orgId, ...restParams }) => {
    return post('/api/v1/cloud/sync-tasks/rerun-failed', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  approveSyncTaskRerunGroup: ({ orgId, groupId, ...restParams }) => {
    return post(`/api/v1/cloud/sync-task-rerun-groups/${groupId}/approve`, restParams, {
      'IaC-Org-Id': orgId
    });
  }
};

export default cloudAssetAPI;
