// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"fmt"
	"strings"
	"time"

	"cloudiac/configs"
	"cloudiac/portal/consts"
	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models"
	"cloudiac/portal/services"
	"cloudiac/portal/services/notificationrc"
	"cloudiac/utils"
	"cloudiac/utils/mail"
)

func dispatchCloudEventNotificationsBestEffort(c *ctx.ServiceContext, event *models.CloudEvent) {
	if c == nil || event == nil || event.OrgId == "" || event.EventType == "" {
		return
	}
	notifications := make([]models.Notification, 0)
	notificationTable := models.Notification{}.TableName()
	eventTable := models.NotificationEvent{}.TableName()
	query := c.DB().Table(notificationTable).
		Joins(fmt.Sprintf("left join %s as ne on %s.id = ne.notification_id", eventTable, notificationTable)).
		Where(fmt.Sprintf("%s.org_id = ?", notificationTable), event.OrgId).
		Where("ne.event_type in (?)", cloudEventNotificationCandidates(*event)).
		Where(fmt.Sprintf("(%s.project_id = '' or %s.project_id is null or %s.project_id = ?)",
			notificationTable, notificationTable, notificationTable), event.ProjectId).
		Group(fmt.Sprintf("%s.id", notificationTable))
	if err := query.Find(&notifications); err != nil {
		c.Logger().Warnf("search cloud event notifications failed: %v", err)
		return
	}
	if len(notifications) == 0 {
		return
	}
	for _, notification := range notifications {
		title, message := cloudEventNotificationContent(c, notification, *event)
		err := sendCloudEventNotification(c, notification, *event, title, message)
		recordCloudEventNotificationDelivery(c, notification, *event, title, message, err)
		if err != nil {
			c.Logger().Warnf("send cloud event notification %s(%s) failed: %v", notification.Id, notification.Type, err)
		}
	}
}

func cloudEventNotificationCandidates(event models.CloudEvent) []string {
	candidates := []string{event.EventType, "cloud.*"}
	if event.Source != "" {
		candidates = append(candidates, event.Source+".*")
	}
	if idx := strings.Index(event.EventType, "."); idx > 0 {
		candidates = append(candidates, event.EventType[:idx]+".*")
	}
	return uniqueStrings(candidates)
}

func sendCloudEventNotification(c *ctx.ServiceContext, notification models.Notification, event models.CloudEvent, title, message string) error {
	switch notification.Type {
	case models.NotificationTypeEmail:
		return sendCloudEventEmailNotification(c, notification, title, message)
	case models.NotificationTypeDingTalk:
		return notificationrc.NewDingTalkRobot(notification.Url, notification.Secret).
			SendMarkdownMessage(title, message, nil, false)
	case models.NotificationTypeWebhook:
		return sendCloudEventWebhookNotification(c, notification, event, title, message)
	case models.NotificationTypeWeChat:
		_, err := (&notificationrc.WeChatRobot{Url: notification.Url}).SendMarkdown(message)
		return err
	case models.NotificationTypeSlack:
		if errs := notificationrc.SendSlack(notification.Url, notificationrc.Payload{Text: message, Markdown: true}); len(errs) > 0 {
			return errs[0]
		}
		return nil
	default:
		return fmt.Errorf("unsupported notification type %s", notification.Type)
	}
}

func sendCloudEventWebhookNotification(c *ctx.ServiceContext, notification models.Notification, event models.CloudEvent, title, message string) error {
	if c == nil {
		return fmt.Errorf("service context is nil")
	}
	if strings.TrimSpace(notification.Url) == "" {
		return fmt.Errorf("webhook notification url is empty")
	}
	webhook, err := ensureCloudEventNotificationWebhookMirror(c, notification, event)
	if err != nil {
		return err
	}
	eventForDelivery := event
	eventForDelivery.Payload = cloneResAttrs(event.Payload)
	eventForDelivery.Payload["notification"] = models.ResAttrs{
		"id":      notification.Id.String(),
		"type":    notification.Type,
		"name":    notification.Name,
		"title":   title,
		"message": message,
	}
	delivery, err := deliverCloudWebhookWithOptions(c, webhook, eventForDelivery, cloudWebhookDeliveryOptions{
		Attempt:       1,
		DeliveryMode:  models.CloudWebhookDeliveryModeInitial,
		ScheduleRetry: true,
	})
	if err != nil {
		return err
	}
	if delivery.Status == models.CloudWebhookDeliveryFailed {
		return fmt.Errorf("webhook delivery %s failed: %s", delivery.Id, firstNonEmpty(delivery.ErrorMessage, delivery.ResponseBody))
	}
	return nil
}

func ensureCloudEventNotificationWebhookMirror(c *ctx.ServiceContext, notification models.Notification, event models.CloudEvent) (models.CloudWebhook, error) {
	webhook := models.CloudWebhook{}
	err := c.DB().Model(&models.CloudWebhook{}).
		Where("id = ? and org_id = ?", notification.Id, event.OrgId).
		First(&webhook)
	if err == nil {
		return webhook, nil
	}
	if !e.IsRecordNotFound(err) {
		return webhook, err
	}

	eventTypes, eventErr := services.SearchNotifyEventType(c.DB(), notification.Id)
	if eventErr != nil {
		return webhook, eventErr
	}
	if len(eventTypes) == 0 {
		eventTypes = []string{event.EventType}
	}
	if syncErr := syncNotificationWebhookMirror(c.DB(), &notification, eventTypes); syncErr != nil {
		return webhook, syncErr
	}
	if err := c.DB().Model(&models.CloudWebhook{}).
		Where("id = ? and org_id = ?", notification.Id, event.OrgId).
		First(&webhook); err != nil {
		if e.IsRecordNotFound(err) {
			return webhook, fmt.Errorf("notification webhook mirror %s not found", notification.Id)
		}
		return webhook, err
	}
	return webhook, nil
}

func cloneResAttrs(attrs models.ResAttrs) models.ResAttrs {
	result := models.ResAttrs{}
	for key, value := range attrs {
		result[key] = value
	}
	return result
}

func recordCloudEventNotificationDelivery(c *ctx.ServiceContext, notification models.Notification, event models.CloudEvent, title, message string, deliveryErr error) {
	if c == nil || notification.Id == "" || event.OrgId == "" {
		return
	}
	status := models.NotificationDeliveryStatusSuccess
	errorMessage := ""
	if deliveryErr != nil {
		status = models.NotificationDeliveryStatusFailed
		errorMessage = deliveryErr.Error()
	}
	delivery := models.NotificationDelivery{
		OrgId:            event.OrgId,
		ProjectId:        event.ProjectId,
		NotificationId:   notification.Id,
		EventId:          event.Id,
		EventType:        event.EventType,
		NotificationType: notification.Type,
		Target:           cloudEventNotificationTarget(c, notification),
		Title:            title,
		Message:          message,
		Status:           status,
		ErrorMessage:     errorMessage,
		DeliveredAt:      models.Time(time.Now()),
	}
	delivery.Id = models.NewId("ndl")
	if err := models.Create(c.DB(), &delivery); err != nil {
		c.Logger().Warnf("record cloud event notification delivery failed: %v", err)
	}
}

func cloudEventNotificationTarget(c *ctx.ServiceContext, notification models.Notification) string {
	if notification.Type != models.NotificationTypeEmail {
		return notification.Url
	}
	if len(notification.UserIds) == 0 || c == nil {
		return ""
	}
	users := make([]models.User, 0)
	if err := c.DB().Model(&models.User{}).Where("id in (?)", []string(notification.UserIds)).Find(&users); err != nil {
		return ""
	}
	emails := make([]string, 0, len(users))
	for _, user := range users {
		if user.Email != "" {
			emails = append(emails, user.Email)
		}
	}
	return strings.Join(emails, ",")
}

func sendCloudEventEmailNotification(c *ctx.ServiceContext, notification models.Notification, title, message string) error {
	if len(notification.UserIds) == 0 {
		return nil
	}
	users := make([]models.User, 0)
	if err := c.DB().Model(&models.User{}).Where("id in (?)", []string(notification.UserIds)).Find(&users); err != nil {
		return err
	}
	for _, user := range users {
		if user.Email == "" {
			continue
		}
		if err := mail.SendMail([]string{user.Email}, title, message); err != nil {
			return err
		}
	}
	return nil
}

func cloudEventNotificationContent(c *ctx.ServiceContext, notification models.Notification, event models.CloudEvent) (string, string) {
	title := cloudEventNotificationTitle(event)
	message := cloudEventNotificationMarkdown(event)
	template, err := models.FindEnabledNotificationTemplate(c.DB(), event.OrgId, event.ProjectId, event.EventType, notification.Type)
	if err != nil {
		c.Logger().Warnf("find cloud event notification template failed: %v", err)
		return title, message
	}
	if template == nil {
		return title, message
	}
	data := struct {
		models.CloudEvent
		ViewUrl string
	}{
		CloudEvent: event,
		ViewUrl:    cloudEventNotificationViewUrl(event),
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

func cloudEventNotificationTitle(event models.CloudEvent) string {
	if event.Title != "" {
		return event.Title
	}
	return consts.NotificationMessageTitle
}

func cloudEventNotificationMarkdown(event models.CloudEvent) string {
	lines := []string{
		fmt.Sprintf("### CloudIaC 多云事件通知：%s", cloudEventNotificationTitle(event)),
		fmt.Sprintf("- 事件类型：%s", valueOrDash(event.EventType)),
		fmt.Sprintf("- 来源：%s", valueOrDash(event.Source)),
		fmt.Sprintf("- 级别：%s", valueOrDash(event.Level)),
		fmt.Sprintf("- 状态：%s", valueOrDash(event.Status)),
		fmt.Sprintf("- 发生时间：%s", cloudEventNotificationTime(event)),
	}
	if event.Provider != "" {
		lines = append(lines, fmt.Sprintf("- 云厂商：%s", event.Provider))
	}
	if event.AccountId != "" || event.CloudAccountId != "" {
		lines = append(lines, fmt.Sprintf("- 云账号：%s", valueOrDash(firstNonEmpty(event.AccountId, event.CloudAccountId.String()))))
	}
	if event.Region != "" {
		lines = append(lines, fmt.Sprintf("- 区域：%s", event.Region))
	}
	if event.ResourceName != "" || event.ResourceId != "" {
		lines = append(lines, fmt.Sprintf("- 资源：%s", valueOrDash(firstNonEmpty(event.ResourceName, event.ResourceId))))
	}
	if event.ProjectId != "" {
		lines = append(lines, fmt.Sprintf("- 项目 ID：%s", event.ProjectId))
	}
	if event.EnvId != "" {
		lines = append(lines, fmt.Sprintf("- 环境 ID：%s", event.EnvId))
	}
	if event.Message != "" {
		lines = append(lines, "", event.Message)
	}
	if viewUrl := cloudEventNotificationViewUrl(event); viewUrl != "" {
		lines = append(lines, "", fmt.Sprintf("[查看事件](%s)", viewUrl))
	}
	return strings.Join(lines, "\n")
}

func cloudEventNotificationViewUrl(event models.CloudEvent) string {
	if configs.Get().Portal.Address == "" {
		return ""
	}
	return fmt.Sprintf("%s/org/%s/m-cloud-events", strings.TrimRight(configs.Get().Portal.Address, "/"), event.OrgId)
}

func cloudEventNotificationTime(event models.CloudEvent) string {
	occurredAt := time.Time(event.OccurredAt)
	if occurredAt.IsZero() || occurredAt.Year() <= 1 {
		return "-"
	}
	return occurredAt.Format("2006-01-02 15:04:05")
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func uniqueStrings(values []string) []string {
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
