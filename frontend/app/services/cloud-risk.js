import { getWithArgs, post, put } from 'utils/xFetch2';

const cloudRiskAPI = {
  list: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/risks', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  updateStatus: ({ orgId, id, ...restParams }) => {
    return put(`/api/v1/cloud/risks/${id}/status`, restParams, {
      'IaC-Org-Id': orgId
    });
  },
  suppress: ({ orgId, id, ...restParams }) => {
    return post(`/api/v1/cloud/risks/${id}/suppress`, restParams, {
      'IaC-Org-Id': orgId
    });
  }
};

export default cloudRiskAPI;
