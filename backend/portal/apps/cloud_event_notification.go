// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"fmt"
	"strings"
	"time"
	"unicode"

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
	for _, route := range cloudEventNotificationPayloadRouteKeys(event.Payload, "notificationRoutes") {
		candidates = append(candidates, "cloud.route."+route)
		if event.Source != "" {
			candidates = append(candidates, event.Source+".route."+route)
		}
	}
	for _, owner := range cloudEventNotificationPayloadRouteKeys(event.Payload, "notificationOwner") {
		candidates = append(candidates, "cloud.owner."+owner)
		if event.Source != "" {
			candidates = append(candidates, event.Source+".owner."+owner)
		}
	}
	for _, assignee := range cloudEventNotificationPayloadRouteKeys(event.Payload, "notificationAssignees") {
		candidates = append(candidates, "cloud.assignee."+assignee)
		if event.Source != "" {
			candidates = append(candidates, event.Source+".assignee."+assignee)
		}
	}
	if failureCategory := cloudEventNotificationPayloadScalarKey(event.Payload, "failureCategory"); failureCategory != "" {
		candidates = append(candidates, "cloud.failure."+failureCategory)
		if event.Source != "" {
			candidates = append(candidates, event.Source+".failure."+failureCategory)
		}
	}
	if cloudService := cloudEventNotificationPayloadScalarKey(event.Payload, "cloudService"); cloudService != "" {
		candidates = append(candidates, "cloud.service."+cloudService)
		if event.Source != "" {
			candidates = append(candidates, event.Source+".service."+cloudService)
		}
	}
	if escalationReason := cloudEventNotificationPayloadScalarKey(event.Payload, "notificationEscalationReason"); escalationReason != "" {
		candidates = append(candidates, "cloud.escalation."+escalationReason)
		if event.Source != "" {
			candidates = append(candidates, event.Source+".escalation."+escalationReason)
		}
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
	if owner := cloudEventNotificationPayloadDisplay(event.Payload, "notificationOwner"); owner != "" {
		lines = append(lines, fmt.Sprintf("- 通知负责人：%s", owner))
	}
	if routes := cloudEventNotificationPayloadDisplay(event.Payload, "notificationRoutes"); routes != "" {
		lines = append(lines, fmt.Sprintf("- 通知路由：%s", routes))
	}
	if assignees := cloudEventNotificationPayloadDisplay(event.Payload, "notificationAssignees"); assignees != "" {
		lines = append(lines, fmt.Sprintf("- 分派对象：%s", assignees))
	}
	if failureCategory := cloudEventNotificationPayloadScalarDisplay(event.Payload, "failureCategory"); failureCategory != "" {
		lines = append(lines, fmt.Sprintf("- 失败类型：%s", failureCategory))
	}
	if cloudService := cloudEventNotificationPayloadScalarDisplay(event.Payload, "cloudService"); cloudService != "" {
		lines = append(lines, fmt.Sprintf("- 云服务：%s", cloudService))
	}
	if cloudEventNotificationPayloadBool(event.Payload, "notificationEscalated") {
		lines = append(lines, "- 通知升级：已升级")
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

func cloudEventNotificationPayloadDisplay(payload models.ResAttrs, key string) string {
	if key == "notificationOwner" {
		return cloudEventNotificationPayloadOwner(payload)
	}
	values := cloudEventNotificationPayloadValues(payload, key)
	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, ", ")
}

func cloudEventNotificationPayloadRouteKeys(payload models.ResAttrs, key string) []string {
	values := []string{}
	if key == "notificationOwner" {
		if owner := cloudEventNotificationPayloadOwner(payload); owner != "" {
			values = append(values, owner)
		}
	} else {
		values = cloudEventNotificationPayloadValues(payload, key)
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		routeKey := cloudEventNotificationRouteKey(value)
		if routeKey != "" {
			result = append(result, routeKey)
		}
	}
	return uniqueStrings(result)
}

func cloudEventNotificationPayloadScalarKey(payload models.ResAttrs, key string) string {
	return cloudEventNotificationRouteKey(cloudEventNotificationPayloadScalarDisplay(payload, key))
}

func cloudEventNotificationPayloadScalarDisplay(payload models.ResAttrs, key string) string {
	if payload == nil {
		return ""
	}
	return strings.TrimSpace(cloudSyncPolicyAttrString(payload[key]))
}

func cloudEventNotificationPayloadBool(payload models.ResAttrs, key string) bool {
	if payload == nil {
		return false
	}
	switch value := payload[key].(type) {
	case bool:
		return value
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "1", "true", "yes", "y", "on", "enable", "enabled":
			return true
		}
	}
	return false
}

func cloudEventNotificationPayloadOwner(payload models.ResAttrs) string {
	if payload == nil {
		return ""
	}
	return strings.TrimSpace(cloudSyncPolicyAttrString(payload["notificationOwner"]))
}

func cloudEventNotificationPayloadValues(payload models.ResAttrs, key string) []string {
	if payload == nil {
		return nil
	}
	value := payload[key]
	switch typed := value.(type) {
	case []string:
		return cloudEventNotificationNormalizePayloadValues(typed, false)
	case models.StrSlice:
		return cloudEventNotificationNormalizePayloadValues([]string(typed), false)
	case []interface{}:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			values = append(values, fmt.Sprint(item))
		}
		return cloudEventNotificationNormalizePayloadValues(values, false)
	case string:
		return cloudEventNotificationNormalizePayloadValues([]string{typed}, true)
	case nil:
		return nil
	default:
		return cloudEventNotificationNormalizePayloadValues(cloudSyncPolicyAttrStringSlice(value), true)
	}
}

func cloudEventNotificationNormalizePayloadValues(values []string, split bool) []string {
	if split {
		return normalizeStringList(values)
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return uniqueStrings(result)
}

func cloudEventNotificationRouteKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.':
			builder.WriteRune(r)
			lastDash = false
		case r == '-' || unicode.IsSpace(r) || r == '/' || r == ':' || r == '@':
			if !lastDash && builder.Len() > 0 {
				builder.WriteRune('-')
				lastDash = true
			}
		}
		if builder.Len() >= 80 {
			break
		}
	}
	return strings.Trim(builder.String(), "-.")
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
