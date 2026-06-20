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
  assetDetail: ({ orgId, id }) => {
    return getWithArgs(`/api/v1/cloud/assets/${id}`, {}, {
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
  startSyncTask: ({ orgId, ...restParams }) => {
    return post('/api/v1/cloud/sync-tasks', restParams, {
      'IaC-Org-Id': orgId
    });
  }
};

export default cloudAssetAPI;
