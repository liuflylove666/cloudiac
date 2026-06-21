// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package notificationrc

import (
	"cloudiac/configs"
	"cloudiac/portal/consts"
	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/db"
	"cloudiac/portal/models"
	"cloudiac/portal/services/cloudwebhook"
	"cloudiac/utils"
	"cloudiac/utils/logs"
	"cloudiac/utils/mail"
	"fmt"
	"strings"
	"time"
)

const (
	notificationWebhookMirrorDescriptionPrefix = "[notification-webhook-mirror]"
	taskNotificationWebhookDefaultTimeout      = 5
	taskNotificationWebhookDefaultMaxRetries   = 3
	taskNotificationWebhookDefaultRetrySeconds = 60
)

type NotificationService struct {
	Tpl       *models.Template     `json:"tpl" form:"tpl" `
	Project   *models.Project      `json:"project" form:"project" `
	Org       *models.Organization `json:"org" form:"org" `
	OrgId     models.Id            `json:"orgId" form:"orgId" `
	ProjectId models.Id            `json:"projectId" form:"projectId" `
	Env       *models.Env          `json:"env" form:"env" `
	Task      *models.Task         `json:"task" form:"task" `
	EventType string               `json:"eventType" form:"eventType" `
}

type NotificationOptions struct {
	Tpl       *models.Template     `json:"tpl" form:"tpl" `
	Project   *models.Project      `json:"project" form:"project" `
	Org       *models.Organization `json:"org" form:"org" `
	OrgId     models.Id            `json:"orgId" form:"orgId" `
	ProjectId models.Id            `json:"projectId" form:"projectId" `
	Env       *models.Env          `json:"env" form:"env" `
	Task      *models.Task         `json:"task" form:"task" `
	EventType string               `json:"eventType" form:"eventType" `
}

func NewNotificationService(options *NotificationOptions) NotificationService {
	return NotificationService{
		OrgId:     options.OrgId,
		ProjectId: options.ProjectId,
		Env:       options.Env,
		Task:      options.Task,
		Tpl:       options.Tpl,
		Project:   options.Project,
		Org:       options.Org,
		EventType: options.EventType,
	}
}

func (ns *NotificationService) SendMessage() {
	go utils.RecoverdCall(ns.SyncSendMessage, func(err error) {
		logs.Get().Warnf("sync send message panic: %v", err)
	})
}

func (ns *NotificationService) SyncSendMessage() {
	logger := logs.Get().WithField("action", "SyncSendMessage")
	notifications, messageTpl, mdMessageTpl, err := ns.FindNotificationsAndMessageTpl()
	if err != nil {
		logger.Warnf("FindNotificationsAndMessageTpl error: %v", err)
		return
	}
	if len(notifications) == 0 {
		logger.Debugln("no notifications")
		return
	}
	u := models.User{}
	if err := db.Get().Where("id = ?", ns.Task.CreatorId).First(&u); err != nil {
		logs.Get().Warnf("get task creator(%s): %v", ns.Task.CreatorId, err)
		return
	}

	data := struct {
		Creator      string
		OrgName      string
		ProjectName  string
		TemplateName string
		Revision     string
		EnvName      string
		Addr         string
		ResAdded     *int
		ResChanged   *int
		ResDestroyed *int
		Message      string
		TaskType     string
	}{
		Creator:      u.Name,
		OrgName:      ns.Org.Name,
		ProjectName:  ns.Project.Name,
		TemplateName: ns.Tpl.Name,
		Revision:     ns.Env.Revision,
		EnvName:      ns.Env.Name,
		//http://{{addr}}/org/{{orgId}}/project/{{ProjectId}}/m-project-env/detail/{{envId}}/task/{{TaskId}}
		Addr:         fmt.Sprintf("%s/org/%s/project/%s/m-project-env/detail/%s/task/%s", configs.Get().Portal.Address, ns.Org.Id, ns.ProjectId, ns.Env.Id, ns.Task.Id),
		ResAdded:     ns.Task.Result.ResAdded,
		ResChanged:   ns.Task.Result.ResChanged,
		ResDestroyed: ns.Task.Result.ResDestroyed,
		Message:      ns.Task.Message,
		TaskType:     ns.Task.Type,
	}

	// 获取消息通知模板
	defaultMdMessage := utils.SprintTemplate(mdMessageTpl, data)
	defaultMessage := utils.SprintTemplate(messageTpl, data)
	emailNotifications := make([]models.Notification, 0)
	// 判断消息类型，下发至的消息通道
	for _, notification := range notifications {
		if notification.Type == models.NotificationTypeEmail {
			emailNotifications = append(emailNotifications, notification)
			continue
		}
		title, message := ns.renderTaskNotificationContent(notification, data, defaultMessage, defaultMdMessage)
		var sendErr error
		switch notification.Type {
		case models.NotificationTypeDingTalk:
			sendErr = ns.SendDingTalkMessage(notification, title, message)
		case models.NotificationTypeWebhook:
			sendErr = ns.SendWebhookMessage(notification, title, message)
		case models.NotificationTypeWeChat:
			sendErr = ns.SendWechatMessage(notification, message)
		case models.NotificationTypeSlack:
			sendErr = ns.SendSlackMessage(notification, message)
		default:
			sendErr = fmt.Errorf("unsupported notification type %s", notification.Type)
		}
		ns.recordTaskNotificationDelivery(notification, notification.Url, title, message, sendErr)
		if sendErr != nil {
			logger.Warnf("send task notification %s(%s) failed: %v", notification.Id, notification.Type, sendErr)
		}
	}

	sentEmails := map[string]bool{}
	for _, notification := range emailNotifications {
		userIds := utils.RemoveDuplicateElement([]string(notification.UserIds))
		if len(userIds) == 0 {
			continue
		}

		// 获取用户邮箱列表
		users := make([]models.User, 0)
		if err := db.Get().Where("id in (?)", userIds).Find(&users); err != nil {
			logger.Warnf("find notification users error: %v", err)
			title, message := ns.renderTaskNotificationContent(notification, data, defaultMessage, defaultMdMessage)
			ns.recordTaskNotificationDelivery(notification, "", title, message, err)
			continue
		}
		title, message := ns.renderTaskNotificationContent(notification, data, defaultMessage, defaultMdMessage)
		for _, v := range users {
			if v.Email == "" || sentEmails[v.Email] {
				continue
			}
			sentEmails[v.Email] = true
			// 单个用户发送邮件，避免暴露其他用户邮箱
			sendErr := ns.SendEmailMessage([]string{v.Email}, title, message)
			ns.recordTaskNotificationDelivery(notification, v.Email, title, message, sendErr)
			if sendErr != nil {
				logger.Warnf("send task email notification %s to %s failed: %v", notification.Id, v.Email, sendErr)
			}
		}
	}
}

func (ns *NotificationService) SendDingTalkMessage(n models.Notification, title, message string) error {
	dingTalk := NewDingTalkRobot(n.Url, n.Secret)
	return dingTalk.SendMarkdownMessage(title, message, nil, false)
}

func (ns *NotificationService) SendWechatMessage(n models.Notification, message string) error {
	wechat := WeChatRobot{Url: n.Url}
	_, err := wechat.SendMarkdown(message)
	return err
}

func (ns *NotificationService) SendWebhookMessage(n models.Notification, title, message string) error {
	webhook, ok := ns.ensureTaskNotificationWebhookMirror(n)
	if !ok {
		return fmt.Errorf("ensure task notification webhook mirror failed")
	}
	event, err := ns.createTaskNotificationWebhookEvent(n, title, message)
	if err != nil {
		return err
	}
	result, err := cloudwebhook.Deliver(db.Get(), webhook, event, cloudwebhook.DeliveryOptions{
		Attempt:       1,
		DeliveryMode:  models.CloudWebhookDeliveryModeInitial,
		ScheduleRetry: true,
	})
	if err != nil {
		return err
	}
	if result == nil || result.Delivery == nil {
		return fmt.Errorf("cloud webhook delivery result is nil")
	}
	if result.Delivery.Status == models.CloudWebhookDeliveryFailed {
		if strings.TrimSpace(result.Delivery.ErrorMessage) != "" {
			return fmt.Errorf("%s", result.Delivery.ErrorMessage)
		}
		return fmt.Errorf("cloud webhook delivery failed with status %s", result.Delivery.Status)
	}
	return nil
}

func (ns *NotificationService) SendSlackMessage(n models.Notification, message string) error {
	if errs := SendSlack(n.Url, Payload{Text: message, Markdown: true}); len(errs) != 0 {
		return fmt.Errorf("send slack message err: %v", errs)
	}
	return nil
}

func (ns *NotificationService) SendEmailMessage(emails []string, title, message string) error {
	if len(emails) < 1 {
		return nil
	}
	return mail.SendMail(emails, title, message)
}

func (ns *NotificationService) renderTaskNotificationContent(notification models.Notification, data interface{}, defaultMessage, defaultMarkdownMessage string) (string, string) {
	title := consts.NotificationMessageTitle
	message := defaultMarkdownMessage
	if notification.Type == models.NotificationTypeEmail {
		message = defaultMessage
	}
	template, err := models.FindEnabledNotificationTemplate(db.Get(), ns.OrgId, ns.ProjectId, ns.EventType, notification.Type)
	if err != nil {
		logs.Get().WithField("eventType", ns.EventType).
			WithField("notificationType", notification.Type).
			Warnf("find notification template failed: %v", err)
		return title, message
	}
	if template == nil {
		return title, message
	}
	if template.Title != "" {
		title = utils.SprintTemplate(template.Title, data)
	}
	content := template.MarkdownContent
	if notification.Type == models.NotificationTypeEmail {
		content = template.Content
	}
	if content == "" {
		if notification.Type == models.NotificationTypeEmail {
			content = template.MarkdownContent
		} else {
			content = template.Content
		}
	}
	if content != "" {
		message = utils.SprintTemplate(content, data)
	}
	return title, message
}

func (ns *NotificationService) recordTaskNotificationDelivery(notification models.Notification, target, title, message string, deliveryErr error) {
	if ns == nil || notification.Id == "" || ns.OrgId == "" {
		return
	}
	status := models.NotificationDeliveryStatusSuccess
	errorMessage := ""
	if deliveryErr != nil {
		status = models.NotificationDeliveryStatusFailed
		errorMessage = deliveryErr.Error()
	}
	delivery := models.NotificationDelivery{
		OrgId:            ns.OrgId,
		ProjectId:        ns.ProjectId,
		NotificationId:   notification.Id,
		EventType:        ns.EventType,
		NotificationType: notification.Type,
		Target:           target,
		Title:            title,
		Message:          message,
		Status:           status,
		ErrorMessage:     errorMessage,
		DeliveredAt:      models.Time(time.Now()),
	}
	if ns.Task != nil {
		delivery.EventId = ns.Task.Id
	}
	delivery.Id = models.NewId("ndl")
	if err := models.Create(db.Get(), &delivery); err != nil {
		logs.Get().WithField("notificationId", notification.Id).
			WithField("eventType", ns.EventType).
			Warnf("record task notification delivery failed: %v", err)
	}
}

func (ns *NotificationService) createTaskNotificationWebhookEvent(notification models.Notification, title, message string) (models.CloudEvent, error) {
	now := time.Now()
	event := models.CloudEvent{
		OrgId:        ns.OrgId,
		ProjectId:    ns.ProjectId,
		Source:       models.CloudEventSourceNotification,
		EventType:    ns.EventType,
		Level:        taskNotificationWebhookEventLevel(ns.EventType),
		Status:       "pending",
		ResourceType: "task",
		Title:        title,
		Message:      message,
		Payload: models.ResAttrs{
			"notification": models.ResAttrs{
				"id":      notification.Id.String(),
				"type":    notification.Type,
				"name":    notification.Name,
				"title":   title,
				"message": message,
			},
			"task": ns.taskNotificationWebhookTaskPayload(),
		},
		OccurredAt: models.Time(now),
	}
	event.Id = models.NewId("evt")
	if ns.Env != nil {
		event.EnvId = ns.Env.Id
		event.ResourceName = ns.Env.Name
	}
	if ns.Task != nil {
		event.ResourceId = ns.Task.Id.String()
		event.Status = ns.Task.Status
	}
	if err := models.Create(db.Get(), &event); err != nil {
		logs.Get().WithField("notificationId", notification.Id).
			Warnf("record task notification webhook event failed: %v", err)
		return event, err
	}
	return event, nil
}

func (ns *NotificationService) ensureTaskNotificationWebhookMirror(notification models.Notification) (models.CloudWebhook, bool) {
	webhook := models.CloudWebhook{}
	if err := db.Get().Model(&models.CloudWebhook{}).
		Where("id = ? and org_id = ?", notification.Id, ns.OrgId).
		First(&webhook); err == nil {
		return webhook, true
	} else if !e.IsRecordNotFound(err) {
		logs.Get().WithField("notificationId", notification.Id).
			Warnf("query task notification webhook mirror failed: %v", err)
		return webhook, false
	}

	eventTypes := ns.taskNotificationWebhookEventTypes(notification.Id)
	webhook = models.CloudWebhook{
		OrgId:          ns.OrgId,
		Name:           taskNotificationWebhookMirrorName(notification),
		Description:    taskNotificationWebhookMirrorDescription(notification.Id),
		TargetUrl:      strings.TrimSpace(notification.Url),
		Secret:         strings.TrimSpace(notification.Secret),
		SecretVersion:  1,
		EventTypes:     models.StrSlice(eventTypes),
		Sources:        models.StrSlice{},
		Status:         models.CloudWebhookStatusEnabled,
		TimeoutSeconds: taskNotificationWebhookDefaultTimeout,
		MaxRetries:     taskNotificationWebhookDefaultMaxRetries,
		RetryInterval:  taskNotificationWebhookDefaultRetrySeconds,
	}
	webhook.Id = notification.Id
	if err := models.Create(db.Get(), &webhook); err != nil {
		logs.Get().WithField("notificationId", notification.Id).
			Warnf("create task notification webhook mirror failed: %v", err)
		return webhook, false
	}
	return webhook, true
}

func (ns *NotificationService) taskNotificationWebhookEventTypes(notificationId models.Id) []string {
	eventTypes := make([]string, 0)
	if err := db.Get().Table(models.NotificationEvent{}.TableName()).
		Where("notification_id = ?", notificationId).
		Pluck("event_type", &eventTypes); err != nil {
		logs.Get().WithField("notificationId", notificationId).
			Warnf("query task notification event types failed: %v", err)
	}
	if len(eventTypes) == 0 && ns.EventType != "" {
		eventTypes = append(eventTypes, ns.EventType)
	}
	return uniqueTaskNotificationWebhookStrings(eventTypes)
}

func taskNotificationWebhookMirrorName(notification models.Notification) string {
	name := strings.TrimSpace(notification.Name)
	if name == "" {
		name = notification.Id.String()
	}
	value := fmt.Sprintf("通知Webhook镜像-%s-%s", notification.Id, name)
	runes := []rune(value)
	if len(runes) > 128 {
		return string(runes[:128])
	}
	return value
}

func taskNotificationWebhookMirrorDescription(notificationId models.Id) string {
	return fmt.Sprintf("%s notificationId=%s", notificationWebhookMirrorDescriptionPrefix, notificationId)
}

func taskNotificationWebhookEventLevel(eventType string) string {
	switch eventType {
	case consts.EventTaskFailed:
		return models.CloudEventLevelError
	case consts.EventTaskApproving:
		return models.CloudEventLevelWarning
	default:
		return models.CloudEventLevelInfo
	}
}

func (ns *NotificationService) taskNotificationWebhookTaskPayload() models.ResAttrs {
	payload := models.ResAttrs{
		"eventType": ns.EventType,
		"orgId":     ns.OrgId.String(),
		"projectId": ns.ProjectId.String(),
	}
	if ns.Org != nil {
		payload["orgName"] = ns.Org.Name
	}
	if ns.Project != nil {
		payload["projectName"] = ns.Project.Name
	}
	if ns.Tpl != nil {
		payload["templateId"] = ns.Tpl.Id.String()
		payload["templateName"] = ns.Tpl.Name
	}
	if ns.Env != nil {
		payload["envId"] = ns.Env.Id.String()
		payload["envName"] = ns.Env.Name
		payload["revision"] = ns.Env.Revision
	}
	if ns.Task != nil {
		payload["taskId"] = ns.Task.Id.String()
		payload["taskType"] = ns.Task.Type
		payload["taskStatus"] = ns.Task.Status
		payload["taskMessage"] = ns.Task.Message
	}
	return payload
}

func uniqueTaskNotificationWebhookStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func (ns *NotificationService) FindNotificationsAndMessageTpl() ([]models.Notification, string, string, error) {
	orgNotification := make([]models.Notification, 0)
	projectNotification := make([]models.Notification, 0)
	notifications := make([]models.Notification, 0)
	dbSess := db.Get().Where("org_id = ?", ns.OrgId).
		Joins(fmt.Sprintf("left join %s as ne on %s.id = ne.notification_id",
			models.NotificationEvent{}.TableName(), models.Notification{}.TableName())).
		Where("ne.event_type = ?", ns.EventType)
	var (
		tplNotificationTemplate      string
		markdownNotificationTemplate string
	)

	switch ns.EventType {
	case consts.EventTaskRunning:
		tplNotificationTemplate = consts.IacTaskRunning
		markdownNotificationTemplate = consts.IacTaskRunningMarkdown
	case consts.EventTaskApproving:
		tplNotificationTemplate = consts.IacTaskApprovingTpl
		markdownNotificationTemplate = consts.IacTaskApprovingMarkdown
	case consts.EventTaskFailed:
		tplNotificationTemplate = consts.IacTaskFailedTpl
		markdownNotificationTemplate = consts.IacTaskFailedMarkdown
	case consts.EventTaskComplete:
		tplNotificationTemplate = consts.IacTaskCompleteTpl
		markdownNotificationTemplate = consts.IacTaskCompleteMarkdown
	case consts.EvenvtCronDrift:
		if ns.Task.Type == models.TaskTypeApply && ns.Task.IsDriftTask {
			tplNotificationTemplate = consts.IacCronDriftApplyTaskTpl
			markdownNotificationTemplate = consts.IacCronDriftApplyTaskMarkDown
		} else {
			tplNotificationTemplate = consts.IacCronDriftPlanTaskTpl
			markdownNotificationTemplate = consts.IacCronDriftPlanTaskMarkDown
		}

	default:
		return nil, "", "", fmt.Errorf("unknown event type '%s'", ns.EventType)
	}

	// 查询需要组织下需要通知的人
	if err := dbSess.
		Where("project_id = '' or project_id is null or project_id = ?", ns.ProjectId).
		Find(&orgNotification); err != nil {
		return notifications, tplNotificationTemplate, markdownNotificationTemplate, err
	}
	// 将需要通知的数据进行整理
	notifications = append(notifications, orgNotification...)
	notifications = append(notifications, projectNotification...)
	return notifications, tplNotificationTemplate, markdownNotificationTemplate, nil
}
