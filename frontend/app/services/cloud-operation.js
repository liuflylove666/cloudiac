import { getWithArgs, post } from 'utils/xFetch2';

const cloudOperationAPI = {
  list: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/operations', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  detail: ({ orgId, id }) => {
    return getWithArgs(`/api/v1/cloud/operations/${id}`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  audits: ({ orgId, id }) => {
    return getWithArgs(`/api/v1/cloud/operations/${id}/audits`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  cancel: ({ orgId, id }) => {
    return post(`/api/v1/cloud/operations/${id}/cancel`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  retry: ({ orgId, id }) => {
    return post(`/api/v1/cloud/operations/${id}/retry`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  approve: ({ orgId, id, ...restParams }) => {
    return post(`/api/v1/cloud/operations/${id}/approve`, restParams, {
      'IaC-Org-Id': orgId
    });
  }
};

export default cloudOperationAPI;
