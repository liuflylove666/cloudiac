import { get, post, put, del, getWithArgs } from 'utils/xFetch2';

const headers = ({ orgId, projectId }) => ({
  'IaC-Org-Id': orgId,
  'IaC-Project-Id': projectId
});

const notificationsAPI = {
  notificationList: ({ orgId, projectId, ...restParams }) => {
    return getWithArgs('/api/v1/notifications', restParams, headers({ orgId, projectId }));
  },
  createNotification: ({ orgId, projectId, ...restParams }) => {
    return post('/api/v1/notifications', restParams, headers({ orgId, projectId }));
  },
  updateNotification: ({ orgId, projectId, notificationId, eventType, name, secret, type, url, userIds }) => {
    return put(`/api/v1/notifications/${notificationId}`, {
      eventType, name, secret, type, url, userIds
    }, headers({ orgId, projectId }));
  },
  delNotification: ({ orgId, projectId, id }) => {
    return del(`/api/v1/notifications/${id}`, {}, headers({ orgId, projectId }));
  },
  detailNotification: ({ orgId, projectId, notificationId }) => {
    return getWithArgs(`/api/v1/notifications/${notificationId}`, {}, headers({ orgId, projectId }));
  },
  notificationDeliveries: ({ orgId, projectId, notificationId, ...restParams }) => {
    return getWithArgs(`/api/v1/notifications/${notificationId}/deliveries`, restParams, headers({ orgId, projectId }));
  },
  notificationTemplateList: ({ orgId, projectId, ...restParams }) => {
    return getWithArgs('/api/v1/notification-templates', restParams, headers({ orgId, projectId }));
  },
  createNotificationTemplate: ({ orgId, projectId, ...restParams }) => {
    return post('/api/v1/notification-templates', restParams, headers({ orgId, projectId }));
  },
  copyNotificationTemplatesToProject: ({ orgId, projectId, sourceTemplateIds }) => {
    return post('/api/v1/notification-templates/copy-to-project', {
      sourceTemplateIds
    }, headers({ orgId, projectId }));
  },
  updateNotificationTemplate: ({ orgId, projectId, templateId, ...restParams }) => {
    return put(`/api/v1/notification-templates/${templateId}`, restParams, headers({ orgId, projectId }));
  },
  detailNotificationTemplate: ({ orgId, projectId, templateId }) => {
    return getWithArgs(`/api/v1/notification-templates/${templateId}`, {}, headers({ orgId, projectId }));
  },
  delNotificationTemplate: ({ orgId, projectId, id }) => {
    return del(`/api/v1/notification-templates/${id}`, {}, headers({ orgId, projectId }));
  },
  notificationTemplateVersions: ({ orgId, projectId, templateId, ...restParams }) => {
    return getWithArgs(`/api/v1/notification-templates/${templateId}/versions`, restParams, headers({ orgId, projectId }));
  },
  rollbackNotificationTemplateVersion: ({ orgId, projectId, templateId, versionId }) => {
    return post(`/api/v1/notification-templates/${templateId}/versions/${versionId}/rollback`, {}, headers({ orgId, projectId }));
  },
  notificationTemplateVariables: ({ orgId, projectId, ...restParams }) => {
    return getWithArgs('/api/v1/notification-templates/variables', restParams, headers({ orgId, projectId }));
  },
  previewNotificationTemplate: ({ orgId, projectId, ...restParams }) => {
    return post('/api/v1/notification-templates/preview', restParams, headers({ orgId, projectId }));
  }
};

export default notificationsAPI;
