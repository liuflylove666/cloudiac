import { getWithArgs } from 'utils/xFetch2';

const cloudOverviewAPI = {
  overview: ({ orgId }) => {
    return getWithArgs('/api/v1/cloud/overview', {}, {
      'IaC-Org-Id': orgId
    });
  }
};

export default cloudOverviewAPI;
