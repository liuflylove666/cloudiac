import { getWithArgs, post, put } from 'utils/xFetch2';

const cmdbAPI = {
  listAssets: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cmdb/assets', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  assetFilters: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cmdb/assets/filters', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  assetDetail: ({ orgId, id }) => {
    return getWithArgs(`/api/v1/cmdb/assets/${id}`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  updateAssetOwnership: ({ orgId, id, ...restParams }) => {
    return put(`/api/v1/cmdb/assets/${id}/ownership`, restParams, {
      'IaC-Org-Id': orgId
    });
  },
  importAssets: ({ orgId, ...restParams }) => {
    return post('/api/v1/cmdb/assets/import', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  listApplications: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cmdb/applications', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  applicationDetail: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cmdb/applications/detail', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  updateApplicationRelations: ({ orgId, ...restParams }) => {
    return put('/api/v1/cmdb/applications/relations', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  backfillIacResources: ({ orgId }) => {
    return post('/api/v1/cmdb/backfill/iac-resources', {}, {
      'IaC-Org-Id': orgId
    });
  },
  cloudAccounts: ({ orgId }) => {
    return getWithArgs('/api/v1/cmdb/cloud/accounts', {}, {
      'IaC-Org-Id': orgId
    });
  },
  syncTasks: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cmdb/sync-tasks', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  startSyncTask: ({ orgId, ...restParams }) => {
    return post('/api/v1/cmdb/sync-tasks', restParams, {
      'IaC-Org-Id': orgId
    });
  }
};

export default cmdbAPI;
