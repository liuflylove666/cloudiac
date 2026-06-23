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
  updateCatalogPolicy: ({ orgId, key, data }) => {
    return put(`/api/v1/cloud/itsm/catalog-policies/${key}`, data, {
      'IaC-Org-Id': orgId
    });
  },
  catalogPolicyHistory: ({ orgId, key, ...restParams }) => {
    return getWithArgs(`/api/v1/cloud/itsm/catalog-policies/${key}/history`, restParams, {
      'IaC-Org-Id': orgId
    });
  },
  updateTicketStatus: ({ orgId, id, data }) => {
    return put(`/api/v1/cloud/itsm/tickets/${id}/status`, data, {
      'IaC-Org-Id': orgId
    });
  },
  syncDueTicketStatuses: ({ orgId, data }) => {
    return post('/api/v1/cloud/itsm/tickets/sync-due', data, {
      'IaC-Org-Id': orgId
    });
  },
  retryFailedTicketSubmissions: ({ orgId, data }) => {
    return post('/api/v1/cloud/itsm/tickets/retry-failed', data, {
      'IaC-Org-Id': orgId
    });
  },
  retryQueueSummary: ({ orgId }) => {
    return getWithArgs('/api/v1/cloud/itsm/tickets/retry-queue/summary', {}, {
      'IaC-Org-Id': orgId
    });
  },
  retryQueueReport: ({ orgId }) => {
    return getWithArgs('/api/v1/cloud/itsm/tickets/retry-queue/report', {}, {
      'IaC-Org-Id': orgId
    });
  },
  retryQueue: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/itsm/tickets/retry-queue', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  replayTicketSubmission: ({ orgId, id, data }) => {
    return post(`/api/v1/cloud/itsm/tickets/${id}/replay-submit`, data, {
      'IaC-Org-Id': orgId
    });
  },
  batchDeadLetterAction: ({ orgId, data }) => {
    return post('/api/v1/cloud/itsm/tickets/retry-queue/dead-letters/action', data, {
      'IaC-Org-Id': orgId
    });
  },
  createDeadLetterApproval: ({ orgId, data }) => {
    return post('/api/v1/cloud/itsm/tickets/retry-queue/dead-letters/approval', data, {
      'IaC-Org-Id': orgId
    });
  }
};

export default cloudItsmAPI;
