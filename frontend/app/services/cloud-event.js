import { del, getWithArgs, post, put } from 'utils/xFetch2';

const cloudEventAPI = {
  list: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/events', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  webhooks: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/webhooks', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  createWebhook: ({ orgId, data }) => {
    return post('/api/v1/cloud/webhooks', data, {
      'IaC-Org-Id': orgId
    });
  },
  updateWebhook: ({ orgId, id, data }) => {
    return put(`/api/v1/cloud/webhooks/${id}`, data, {
      'IaC-Org-Id': orgId
    });
  },
  deleteWebhook: ({ orgId, id }) => {
    return del(`/api/v1/cloud/webhooks/${id}`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  testWebhook: ({ orgId, id }) => {
    return post(`/api/v1/cloud/webhooks/${id}/test`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  rotateWebhookSecret: ({ orgId, id, data }) => {
    return post(`/api/v1/cloud/webhooks/${id}/secret/rotate`, data, {
      'IaC-Org-Id': orgId
    });
  },
  retryDueWebhooks: ({ orgId }) => {
    return post('/api/v1/cloud/webhooks/retry-due', {}, {
      'IaC-Org-Id': orgId
    });
  },
  webhookQueueSummary: ({ orgId }) => {
    return getWithArgs('/api/v1/cloud/webhooks/queue/summary', {}, {
      'IaC-Org-Id': orgId
    });
  },
  webhookDeadLetters: ({ orgId, ...restParams }) => {
    return getWithArgs('/api/v1/cloud/webhooks/dead-letters', restParams, {
      'IaC-Org-Id': orgId
    });
  },
  replayWebhookDeadLetter: ({ orgId, deadLetterId }) => {
    return post(`/api/v1/cloud/webhooks/dead-letters/${deadLetterId}/replay`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  ignoreWebhookDeadLetter: ({ orgId, deadLetterId, data }) => {
    return post(`/api/v1/cloud/webhooks/dead-letters/${deadLetterId}/ignore`, data || {}, {
      'IaC-Org-Id': orgId
    });
  },
  webhookDeliveries: ({ orgId, id, ...restParams }) => {
    return getWithArgs(`/api/v1/cloud/webhooks/${id}/deliveries`, restParams, {
      'IaC-Org-Id': orgId
    });
  },
  webhookDelivery: ({ orgId, id, deliveryId }) => {
    return getWithArgs(`/api/v1/cloud/webhooks/${id}/deliveries/${deliveryId}`, {}, {
      'IaC-Org-Id': orgId
    });
  },
  retryWebhookDelivery: ({ orgId, id, deliveryId }) => {
    return post(`/api/v1/cloud/webhooks/${id}/deliveries/${deliveryId}/retry`, {}, {
      'IaC-Org-Id': orgId
    });
  }
};

export default cloudEventAPI;
