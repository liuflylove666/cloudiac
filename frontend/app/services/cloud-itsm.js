import { del, getWithArgs, post, put } from 'utils/xFetch2';

const cloudItsmAPI = {
  configs: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/itsm/configs', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  overview: ({ orgId }) => {
    return getWithArgs('/api/v1/cloud/itsm/overview', {}, {
      'IaC-Org-Id': orgId
    });
  },
  createConfig: ({ orgId, data }) => {
    return post('/api/v1/cloud/itsm/configs', data, {
      'IaC-Org-Id': orgId
    });
  },
  updateConfig: ({ orgId, id, data }) => {
    return put(`/api/v1/cloud/itsm/configs/${id}`, data, {
      'IaC-Org-Id': orgId
    });
  },
  deleteConfig: ({ orgId, id }) => {
    return del(`/api/v1/cloud/itsm/configs/${id}`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  tickets: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/itsm/tickets', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  createSelfServiceTicket: ({ orgId, data }) => {
    return post('/api/v1/cloud/itsm/self-service-tickets', data, {
      'IaC-Org-Id': orgId
    });
  },
  updateTicketStatus: ({ orgId, id, data }) => {
    return put(`/api/v1/cloud/itsm/tickets/${id}/status`, data, {
      'IaC-Org-Id': orgId
    });
  }
};

export default cloudItsmAPI;
