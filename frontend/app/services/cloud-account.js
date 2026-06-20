import { del, getWithArgs, post, put } from 'utils/xFetch2';

const cloudAccountAPI = {
  list: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/accounts', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  detail: ({ orgId, id }) => {
    return getWithArgs(`/api/v1/cloud/accounts/${id}`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  create: ({ orgId, ...restParams }) => {
    return post('/api/v1/cloud/accounts', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  update: ({ orgId, id, ...restParams }) => {
    return put(`/api/v1/cloud/accounts/${id}`, restParams, {
      'IaC-Org-Id': orgId
    });
  },
  delete: ({ orgId, id }) => {
    return del(`/api/v1/cloud/accounts/${id}`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  validate: ({ orgId, id }) => {
    return post(`/api/v1/cloud/accounts/${id}/validate`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  regions: ({ orgId, id }) => {
    return getWithArgs(`/api/v1/cloud/accounts/${id}/regions`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  updateRegions: ({ orgId, id, regions }) => {
    return put(`/api/v1/cloud/accounts/${id}/regions`, { regions }, {
      'IaC-Org-Id': orgId
    });
  },
  permissions: ({ orgId, id }) => {
    return getWithArgs(`/api/v1/cloud/accounts/${id}/permissions`, {}, {
      'IaC-Org-Id': orgId
    });
  }
};

export default cloudAccountAPI;
