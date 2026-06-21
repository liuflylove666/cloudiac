// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"bytes"
	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/libs/db"
	"cloudiac/portal/libs/page"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	"cloudiac/portal/models/resps"
	"cloudiac/portal/services"
	"cloudiac/utils/logs"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/lib/pq"
)

const (
	notificationDeliveryCleanupIntervalEnv     = "CLOUDIAC_NOTIFICATION_DELIVERY_CLEANUP_INTERVAL_SECONDS"
	notificationDeliveryRetentionDaysEnv       = "CLOUDIAC_NOTIFICATION_DELIVERY_RETENTION_DAYS"
	notificationDeliveryDefaultInterval        = time.Hour
	notificationDeliveryDefaultRetention       = 90
	notificationDeliveryMinInterval            = time.Minute
	notificationDeliveryMaxInterval            = 24 * time.Hour
	notificationDeliveryMinRetention           = 1
	notificationDeliveryMaxRetention           = 3650
	notificationWebhookMirrorDescriptionPrefix = "[notification-webhook-mirror]"
)

func SearchNotification(c *ctx.ServiceContext, form *forms.SearchNotificationForm) (interface{}, e.Error) {
	notify := make([]*resps.RespNotification, 0)
	query := services.SearchNotification(c.DB(), c.OrgId, c.ProjectId)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&notify); err != nil {
		return nil, e.New(e.DBError, err)
	}
	for index, v := range notify {
		notify[index].EventTypes = strings.Split(v.EventType, ",")
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     notify,
	}, nil
}

func DeleteNotification(c *ctx.ServiceContext, id models.Id) (result interface{}, err e.Error) {
	c.AddLogField("action", fmt.Sprintf("Delete notification id: %s", id))
	err = services.DeleteNotification(c.DB(), id, c.OrgId, c.ProjectId)
	if err != nil {
		return nil, err
	}
	if err := deleteNotificationWebhookMirror(c.DB(), c.OrgId, id); err != nil {
		return nil, err
	}
	return
}

func UpdateNotification(c *ctx.ServiceContext, form *forms.UpdateNotificationForm) (cfg *models.Notification, err e.Error) {
	c.AddLogField("action", fmt.Sprintf("update org notification cfg id: %s", form.Id))

	if form.Id == "" {
		return nil, e.New(e.BadRequest, fmt.Errorf("missing 'id'"))
	}
	if err := validateNotificationEventTypes(form.EventType); err != nil {
		return nil, err
	}

	tx := c.Tx()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	attrs := models.Attrs{}
	if form.HasKey("name") {
		attrs["name"] = form.Name
	}

	if form.HasKey("type") {
		attrs["type"] = form.Type
	}

	if form.HasKey("secret") {
		attrs["secret"] = form.Secret
	}

	if form.HasKey("url") {
		attrs["url"] = form.Url
	}

	if form.HasKey("userIds") {
		attrs["userIds"] = pq.StringArray(form.UserIds)
	}

	cfg, err = services.UpdateNotification(tx, form.Id, c.OrgId, c.ProjectId, attrs)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := services.DeleteNotificationEvent(tx, form.Id); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, err)
	}

	events := make([]models.NotificationEvent, 0, len(form.EventType))
	for _, v := range form.EventType {
		events = append(events, models.NotificationEvent{
			NotificationId: form.Id,
			EventType:      v,
		})
	}

	if err := tx.Insert(&events); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, err)
	}
	if err := syncNotificationWebhookMirror(tx, cfg, form.EventType); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, err)
	}

	return cfg, err
}

func CreateNotification(c *ctx.ServiceContext, form *forms.CreateNotificationForm) (*models.Notification, e.Error) {
	c.AddLogField("action", fmt.Sprintf("create org notification cfg %s", form.Type))
	if err := validateNotificationEventTypes(form.EventType); err != nil {
		return nil, err
	}

	tx := c.Tx()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	notification, err := services.CreateNotification(tx, models.Notification{
		OrgId:     c.OrgId,
		ProjectId: c.ProjectId,
		Name:      form.Name,
		Type:      form.Type,
		Secret:    form.Secret,
		Url:       form.Url,
		UserIds:   pq.StringArray(form.UserIds),
		Creator:   c.UserId,
	}, form.EventType)

	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := syncNotificationWebhookMirror(tx, notification, form.EventType); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, err)
	}

	return notification, nil
}

func syncNotificationWebhookMirror(tx *db.Session, notification *models.Notification, eventTypes []string) e.Error {
	if tx == nil || notification == nil || notification.Id == "" || notification.OrgId == "" {
		return nil
	}
	if notification.Type != models.NotificationTypeWebhook || strings.TrimSpace(notification.Url) == "" {
		return deleteNotificationWebhookMirror(tx, notification.OrgId, notification.Id)
	}

	webhook := models.CloudWebhook{
		OrgId:              notification.OrgId,
		Name:               notificationWebhookMirrorName(notification),
		Description:        notificationWebhookMirrorDescription(notification.Id),
		TargetUrl:          strings.TrimSpace(notification.Url),
		Secret:             strings.TrimSpace(notification.Secret),
		SecretVersion:      1,
		EventTypes:         models.StrSlice(normalizeStringSlice(eventTypes)),
		Status:             models.CloudWebhookStatusEnabled,
		TimeoutSeconds:     5,
		MaxRetries:         3,
		RetryInterval:      60,
		RetryJitterPercent: 0,
		MaxRetryDuration:   0,
	}
	webhook.Id = notification.Id

	existing := models.CloudWebhook{}
	if err := tx.Where("id = ? and org_id = ?", notification.Id, notification.OrgId).First(&existing); err != nil {
		if !e.IsRecordNotFound(err) {
			return e.New(e.DBError, err)
		}
		if err := models.Create(tx, &webhook); err != nil {
			return e.New(e.DBError, err)
		}
		return nil
	}

	attrs := models.Attrs{
		"name":                 webhook.Name,
		"description":          webhook.Description,
		"target_url":           webhook.TargetUrl,
		"event_types":          webhook.EventTypes,
		"sources":              models.StrSlice{},
		"status":               webhook.Status,
		"timeout_seconds":      webhook.TimeoutSeconds,
		"max_retries":          webhook.MaxRetries,
		"retry_interval":       webhook.RetryInterval,
		"retry_jitter_percent": webhook.RetryJitterPercent,
		"max_retry_duration":   webhook.MaxRetryDuration,
	}
	if strings.TrimSpace(existing.Secret) != webhook.Secret {
		for key, value := range cloudWebhookSecretRotationAttrs(existing, webhook.Secret, cloudWebhookDefaultSecretGraceSeconds) {
			attrs[key] = value
		}
	}
	if _, err := tx.Model(&models.CloudWebhook{}).
		Where("id = ? and org_id = ?", notification.Id, notification.OrgId).
		UpdateAttrs(attrs); err != nil {
		return e.New(e.DBError, err)
	}
	return nil
}

func deleteNotificationWebhookMirror(tx *db.Session, orgId models.Id, notificationId models.Id) e.Error {
	if tx == nil || orgId == "" || notificationId == "" {
		return nil
	}
	if _, err := tx.Where("id = ? and org_id = ?", notificationId, orgId).Delete(&models.CloudWebhook{}); err != nil {
		return e.New(e.DBError, err)
	}
	return nil
}

func notificationWebhookMirrorName(notification *models.Notification) string {
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

func notificationWebhookMirrorDescription(notificationId models.Id) string {
	return fmt.Sprintf("%s notificationId=%s", notificationWebhookMirrorDescriptionPrefix, notificationId)
}

func DetailNotification(c *ctx.ServiceContext, form *forms.DetailNotificationForm) (interface{}, e.Error) {
	return services.DetailNotification(c.DB(), form.Id, c.OrgId, c.ProjectId)
}

func SearchNotificationDeliveries(c *ctx.ServiceContext, form *forms.SearchNotificationDeliveryForm) (interface{}, e.Error) {
	notification := models.Notification{}
	notificationQuery := c.DB().Model(&models.Notification{}).
		Where("id = ? and org_id = ?", form.Id, c.OrgId)
	if c.ProjectId != "" {
		notificationQuery = notificationQuery.Where("project_id = ?", c.ProjectId)
	} else {
		notificationQuery = notificationQuery.Where("(project_id = '' or project_id is null)")
	}
	if err := notificationQuery.First(&notification); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, err)
	}

	query := c.DB().Model(&models.NotificationDelivery{}).
		Where("org_id = ? and notification_id = ?", c.OrgId, form.Id)
	if form.EventId != "" {
		query = query.Where("event_id = ?", form.EventId)
	}
	if form.EventType != "" {
		query = query.Where("event_type = ?", strings.TrimSpace(form.EventType))
	}
	if form.Type != "" {
		query = query.Where("notification_type = ?", form.Type)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	query = query.Order("delivered_at desc").Order("created_at desc")

	deliveries := make([]models.NotificationDelivery, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&deliveries); err != nil {
		return nil, e.New(e.DBError, err)
	}
	list := make([]resps.NotificationDeliveryResp, 0, len(deliveries))
	for _, delivery := range deliveries {
		list = append(list, resps.NotificationDeliveryResp{
			NotificationDelivery: delivery,
			NotificationName:     notification.Name,
		})
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func SearchNotificationTemplates(c *ctx.ServiceContext, form *forms.SearchNotificationTemplateForm) (interface{}, e.Error) {
	nt := models.NotificationTemplate{}.TableName()
	query := c.DB().Table(nt).
		Joins(fmt.Sprintf("left join %s as user on %s.creator = user.id", models.User{}.TableName(), nt)).
		Where(fmt.Sprintf("%s.org_id = ? and %s.deleted_at_t = 0", nt, nt), c.OrgId)
	if c.ProjectId != "" {
		query = query.Where(fmt.Sprintf("%s.project_id = ?", nt), c.ProjectId)
	} else {
		query = query.Where(fmt.Sprintf("(%s.project_id = '' or %s.project_id is null)", nt, nt))
	}
	if form.EventType != "" {
		query = query.Where(fmt.Sprintf("%s.event_type = ?", nt), strings.TrimSpace(form.EventType))
	}
	if form.Type != "" {
		query = query.Where(fmt.Sprintf("%s.notification_type = ?", nt), form.Type)
	}
	if form.Status != "" {
		query = query.Where(fmt.Sprintf("%s.status = ?", nt), form.Status)
	}
	query = query.LazySelectAppend(fmt.Sprintf("%s.*", nt), "user.name as creator_name").
		Order(fmt.Sprintf("%s.created_at desc", nt))

	templates := make([]resps.NotificationTemplateResp, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&templates); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     templates,
	}, nil
}

func CreateNotificationTemplate(c *ctx.ServiceContext, form *forms.CreateNotificationTemplateForm) (*models.NotificationTemplate, e.Error) {
	if err := validateNotificationTemplateForm(form); err != nil {
		return nil, err
	}
	sourceTemplateId := models.Id("")
	var sourceTemplate *models.NotificationTemplate
	if form.SourceTemplateId != "" {
		templateSource, err := validateNotificationTemplateSource(c, form)
		if err != nil {
			return nil, err
		}
		sourceTemplate = templateSource
		sourceTemplateId = templateSource.Id
	}
	template := models.NotificationTemplate{
		OrgId:            c.OrgId,
		ProjectId:        c.ProjectId,
		Name:             strings.TrimSpace(form.Name),
		EventType:        strings.TrimSpace(form.EventType),
		NotificationType: form.Type,
		Title:            strings.TrimSpace(form.Title),
		Content:          form.Content,
		MarkdownContent:  form.MarkdownContent,
		Status:           notificationTemplateStatus(form.Status),
		Creator:          c.UserId,
		SourceTemplateId: sourceTemplateId,
	}
	template.Id = models.NewId("ntpl")

	tx := c.Tx()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()
	if err := models.Create(tx, &template); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, err)
	}
	if err := recordNotificationTemplateVersion(tx, c, &template, sourceTemplate, models.NotificationTemplateVersionActionCreate); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, err)
	}
	return &template, nil
}

func CopyNotificationTemplatesToProject(c *ctx.ServiceContext, form *forms.CopyNotificationTemplatesForm) (*resps.NotificationTemplateCopyResp, e.Error) {
	if c.ProjectId == "" {
		return nil, e.New(e.BadParam, fmt.Errorf("notification template copy is only supported for project templates"))
	}
	c.AddLogField("action", fmt.Sprintf("copy notification templates to project %s", c.ProjectId))

	tx := c.Tx()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	resp := &resps.NotificationTemplateCopyResp{
		Items: make([]resps.NotificationTemplateCopyItem, 0, len(form.SourceTemplateIds)),
	}
	seen := map[models.Id]bool{}
	for _, sourceTemplateId := range form.SourceTemplateIds {
		if seen[sourceTemplateId] {
			resp.Skipped++
			resp.Items = append(resp.Items, resps.NotificationTemplateCopyItem{
				SourceTemplateId: sourceTemplateId,
				Status:           "skipped",
				Reason:           "重复来源模板",
			})
			continue
		}
		seen[sourceTemplateId] = true

		source := models.NotificationTemplate{}
		if err := tx.
			Where("id = ? and org_id = ? and (project_id = '' or project_id is null) and deleted_at_t = 0", sourceTemplateId, c.OrgId).
			First(&source); err != nil {
			_ = tx.Rollback()
			if e.IsRecordNotFound(err) {
				return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
			}
			return nil, e.New(e.DBError, err)
		}

		existing := models.NotificationTemplate{}
		if err := tx.
			Where("org_id = ? and project_id = ? and event_type = ? and notification_type = ? and deleted_at_t = 0",
				c.OrgId, c.ProjectId, source.EventType, source.NotificationType).
			First(&existing); err == nil {
			resp.Skipped++
			resp.Items = append(resp.Items, resps.NotificationTemplateCopyItem{
				SourceTemplateId:  source.Id,
				ProjectTemplateId: existing.Id,
				Name:              source.Name,
				EventType:         source.EventType,
				Type:              source.NotificationType,
				Status:            "skipped",
				Reason:            "项目模板已覆盖",
			})
			continue
		} else if !e.IsRecordNotFound(err) {
			_ = tx.Rollback()
			return nil, e.New(e.DBError, err)
		}

		template := models.NotificationTemplate{
			OrgId:            c.OrgId,
			ProjectId:        c.ProjectId,
			Name:             source.Name,
			EventType:        source.EventType,
			NotificationType: source.NotificationType,
			Title:            source.Title,
			Content:          source.Content,
			MarkdownContent:  source.MarkdownContent,
			Status:           source.Status,
			Creator:          c.UserId,
			SourceTemplateId: source.Id,
		}
		template.Id = models.NewId("ntpl")
		if err := models.Create(tx, &template); err != nil {
			_ = tx.Rollback()
			return nil, e.New(e.DBError, err)
		}
		if err := recordNotificationTemplateVersion(tx, c, &template, &source, models.NotificationTemplateVersionActionCopy); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		resp.Copied++
		resp.Items = append(resp.Items, resps.NotificationTemplateCopyItem{
			SourceTemplateId:  source.Id,
			ProjectTemplateId: template.Id,
			Name:              source.Name,
			EventType:         source.EventType,
			Type:              source.NotificationType,
			Status:            "copied",
			Reason:            "已复制",
		})
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, err)
	}
	return resp, nil
}

func DetailNotificationTemplate(c *ctx.ServiceContext, form *forms.DetailNotificationTemplateForm) (*resps.NotificationTemplateResp, e.Error) {
	resp := resps.NotificationTemplateResp{}
	nt := models.NotificationTemplate{}.TableName()
	query := c.DB().Table(nt).
		Joins(fmt.Sprintf("left join %s as user on %s.creator = user.id", models.User{}.TableName(), nt)).
		Where(fmt.Sprintf("%s.id = ? and %s.org_id = ? and %s.deleted_at_t = 0", nt, nt, nt), form.Id, c.OrgId)
	if c.ProjectId != "" {
		query = query.Where(fmt.Sprintf("%s.project_id = ?", nt), c.ProjectId)
	} else {
		query = query.Where(fmt.Sprintf("(%s.project_id = '' or %s.project_id is null)", nt, nt))
	}
	if err := query.
		LazySelectAppend(fmt.Sprintf("%s.*", nt), "user.name as creator_name").
		First(&resp); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, err)
	}
	return &resp, nil
}

func UpdateNotificationTemplate(c *ctx.ServiceContext, form *forms.UpdateNotificationTemplateForm) (*models.NotificationTemplate, e.Error) {
	if err := validateNotificationTemplateForm(&form.CreateNotificationTemplateForm); err != nil {
		return nil, err
	}
	tx := c.Tx()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	template := models.NotificationTemplate{}
	query := tx.Where("id = ? and org_id = ? and deleted_at_t = 0", form.Id, c.OrgId)
	if c.ProjectId != "" {
		query = query.Where("project_id = ?", c.ProjectId)
	} else {
		query = query.Where("(project_id = '' or project_id is null)")
	}
	if err := query.First(&template); err != nil {
		_ = tx.Rollback()
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, err)
	}
	attrs := models.Attrs{
		"name":              strings.TrimSpace(form.Name),
		"event_type":        strings.TrimSpace(form.EventType),
		"notification_type": form.Type,
		"title":             strings.TrimSpace(form.Title),
		"content":           form.Content,
		"markdown_content":  form.MarkdownContent,
		"status":            notificationTemplateStatus(form.Status),
	}
	query = tx.Model(&models.NotificationTemplate{}).
		Where("id = ? and org_id = ? and deleted_at_t = 0", form.Id, c.OrgId)
	if c.ProjectId != "" {
		query = query.Where("project_id = ?", c.ProjectId)
	} else {
		query = query.Where("(project_id = '' or project_id is null)")
	}
	if _, err := query.UpdateAttrs(attrs); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, err)
	}
	query = tx.Where("id = ? and org_id = ? and deleted_at_t = 0", form.Id, c.OrgId)
	if c.ProjectId != "" {
		query = query.Where("project_id = ?", c.ProjectId)
	} else {
		query = query.Where("(project_id = '' or project_id is null)")
	}
	if err := query.First(&template); err != nil {
		_ = tx.Rollback()
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, err)
	}
	source, err := getNotificationTemplateSourceForSnapshot(tx, c, template.SourceTemplateId)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := recordNotificationTemplateVersion(tx, c, &template, source, models.NotificationTemplateVersionActionUpdate); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, err)
	}
	return &template, nil
}

func DeleteNotificationTemplate(c *ctx.ServiceContext, form *forms.DeleteNotificationTemplateForm) (interface{}, e.Error) {
	query := c.DB().Where("id = ? and org_id = ? and deleted_at_t = 0", form.Id, c.OrgId)
	if c.ProjectId != "" {
		query = query.Where("project_id = ?", c.ProjectId)
	} else {
		query = query.Where("(project_id = '' or project_id is null)")
	}
	affected, err := query.Delete(&models.NotificationTemplate{})
	if err != nil {
		return nil, e.New(e.DBError, err)
	}
	if affected == 0 {
		return nil, e.New(e.ObjectNotExistsOrNoPerm, fmt.Errorf("notification template %s not found", form.Id))
	}
	return nil, nil
}

func SearchNotificationTemplateVersions(c *ctx.ServiceContext, form *forms.SearchNotificationTemplateVersionsForm) (interface{}, e.Error) {
	if _, err := getNotificationTemplateInScope(c.DB(), c, form.Id); err != nil {
		return nil, err
	}
	ntv := models.NotificationTemplateVersion{}.TableName()
	query := c.DB().Table(ntv).
		Joins(fmt.Sprintf("left join %s as user on %s.operator = user.id", models.User{}.TableName(), ntv)).
		Where(fmt.Sprintf("%s.org_id = ? and %s.template_id = ? and %s.deleted_at_t = 0", ntv, ntv, ntv), c.OrgId, form.Id)
	if c.ProjectId != "" {
		query = query.Where(fmt.Sprintf("%s.project_id = ?", ntv), c.ProjectId)
	} else {
		query = query.Where(fmt.Sprintf("(%s.project_id = '' or %s.project_id is null)", ntv, ntv))
	}
	query = query.LazySelectAppend(fmt.Sprintf("%s.*", ntv), "user.name as operator_name").
		Order(fmt.Sprintf("%s.version_no desc", ntv))

	versions := make([]resps.NotificationTemplateVersionResp, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&versions); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     versions,
	}, nil
}

func RollbackNotificationTemplateVersion(c *ctx.ServiceContext, form *forms.RollbackNotificationTemplateVersionForm) (*models.NotificationTemplate, e.Error) {
	tx := c.Tx()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	template, err := getNotificationTemplateInScope(tx, c, form.Id)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	version := models.NotificationTemplateVersion{}
	versionQuery := tx.Where("id = ? and org_id = ? and template_id = ? and deleted_at_t = 0", form.VersionId, c.OrgId, form.Id)
	if c.ProjectId != "" {
		versionQuery = versionQuery.Where("project_id = ?", c.ProjectId)
	} else {
		versionQuery = versionQuery.Where("(project_id = '' or project_id is null)")
	}
	if err := versionQuery.First(&version); err != nil {
		_ = tx.Rollback()
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, err)
	}
	if err := ensureNotificationTemplateRollbackTargetAvailable(tx, template, &version); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	attrs := models.Attrs{
		"name":               version.Name,
		"event_type":         version.EventType,
		"notification_type":  version.NotificationType,
		"title":              version.Title,
		"content":            version.Content,
		"markdown_content":   version.MarkdownContent,
		"status":             version.Status,
		"source_template_id": version.SourceTemplateId,
	}
	query := tx.Model(&models.NotificationTemplate{}).
		Where("id = ? and org_id = ? and deleted_at_t = 0", template.Id, c.OrgId)
	if c.ProjectId != "" {
		query = query.Where("project_id = ?", c.ProjectId)
	} else {
		query = query.Where("(project_id = '' or project_id is null)")
	}
	if _, err := query.UpdateAttrs(attrs); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, err)
	}
	if err := tx.Where("id = ? and org_id = ? and deleted_at_t = 0", template.Id, c.OrgId).First(template); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, err)
	}
	if err := recordNotificationTemplateVersion(tx, c, template, notificationTemplateSourceFromVersion(&version), models.NotificationTemplateVersionActionRollback); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return nil, e.New(e.DBError, err)
	}
	return template, nil
}

func NotificationTemplateVariables(c *ctx.ServiceContext, form *forms.NotificationTemplateVariablesForm) (*resps.NotificationTemplateVariablesResp, e.Error) {
	if form.EventType != "" {
		if err := validateNotificationEventTypes([]string{form.EventType}); err != nil {
			return nil, err
		}
	}
	sampleData := notificationTemplateSampleData(c, form.EventType)
	if form.EventType != "" {
		sampleData["EventType"] = strings.TrimSpace(form.EventType)
	}
	return &resps.NotificationTemplateVariablesResp{
		Groups:     notificationTemplateVariableGroups(c, form.EventType, sampleData),
		SampleData: sampleData,
	}, nil
}

func PreviewNotificationTemplate(c *ctx.ServiceContext, form *forms.PreviewNotificationTemplateForm) (*resps.NotificationTemplatePreviewResp, e.Error) {
	if form.EventType != "" {
		if err := validateNotificationEventTypes([]string{form.EventType}); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(form.Title) == "" && strings.TrimSpace(form.Content) == "" && strings.TrimSpace(form.MarkdownContent) == "" {
		return nil, e.New(e.BadParam, fmt.Errorf("notification template preview content is required"))
	}
	sampleData := notificationTemplateSampleData(c, form.EventType)
	for key, value := range form.SampleData {
		sampleData[key] = value
	}
	title, err := renderNotificationTemplateField("title", form.Title, sampleData)
	if err != nil {
		return nil, e.New(e.BadParam, err)
	}
	content, err := renderNotificationTemplateField("content", form.Content, sampleData)
	if err != nil {
		return nil, e.New(e.BadParam, err)
	}
	markdownContent, err := renderNotificationTemplateField("markdownContent", form.MarkdownContent, sampleData)
	if err != nil {
		return nil, e.New(e.BadParam, err)
	}
	return &resps.NotificationTemplatePreviewResp{
		Title:           title,
		Content:         content,
		MarkdownContent: markdownContent,
		SampleData:      sampleData,
	}, nil
}

func StartNotificationDeliveryCleanupWorker(serviceId string) {
	logger := logs.Get().WithField("worker", "notificationDeliveryCleanup").WithField("serviceId", serviceId)
	interval := notificationDeliveryCleanupInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if cleaned, err := CleanupExpiredNotificationDeliveries(); err != nil {
			logger.Warnf("cleanup expired notification deliveries failed: %v", err)
		} else if cleaned > 0 {
			logger.Infof("cleanup expired notification deliveries result: cleaned=%d retentionDays=%d", cleaned, notificationDeliveryRetentionDays())
		}
		<-ticker.C
	}
}

func CleanupExpiredNotificationDeliveries() (int64, e.Error) {
	lockName := "cloudiac:notification_delivery:cleanup"
	locked, releaseLock, lockErr := cloudWebhookAcquireMysqlLock(lockName)
	if lockErr != nil {
		return 0, e.New(e.DBError, lockErr)
	}
	if !locked {
		return 0, nil
	}
	defer releaseLock()

	cutoff := models.Time(time.Now().AddDate(0, 0, -notificationDeliveryRetentionDays()))
	minTime := models.Time(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC))
	rows, err := db.Get().
		Where("delivered_at > ? and delivered_at <= ?", minTime, cutoff).
		Delete(&models.NotificationDelivery{})
	if err != nil {
		return rows, e.New(e.DBError, err)
	}
	return rows, nil
}

func notificationDeliveryCleanupInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv(notificationDeliveryCleanupIntervalEnv))
	if raw == "" {
		return notificationDeliveryDefaultInterval
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return notificationDeliveryDefaultInterval
	}
	interval := time.Duration(seconds) * time.Second
	if interval < notificationDeliveryMinInterval {
		return notificationDeliveryMinInterval
	}
	if interval > notificationDeliveryMaxInterval {
		return notificationDeliveryMaxInterval
	}
	return interval
}

func notificationDeliveryRetentionDays() int {
	raw := strings.TrimSpace(os.Getenv(notificationDeliveryRetentionDaysEnv))
	if raw == "" {
		return notificationDeliveryDefaultRetention
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days <= 0 {
		return notificationDeliveryDefaultRetention
	}
	if days < notificationDeliveryMinRetention {
		return notificationDeliveryMinRetention
	}
	if days > notificationDeliveryMaxRetention {
		return notificationDeliveryMaxRetention
	}
	return days
}

func getNotificationTemplateInScope(tx *db.Session, c *ctx.ServiceContext, id models.Id) (*models.NotificationTemplate, e.Error) {
	template := models.NotificationTemplate{}
	query := tx.Where("id = ? and org_id = ? and deleted_at_t = 0", id, c.OrgId)
	if c.ProjectId != "" {
		query = query.Where("project_id = ?", c.ProjectId)
	} else {
		query = query.Where("(project_id = '' or project_id is null)")
	}
	if err := query.First(&template); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, err)
	}
	return &template, nil
}

func getNotificationTemplateSourceForSnapshot(tx *db.Session, c *ctx.ServiceContext, sourceTemplateId models.Id) (*models.NotificationTemplate, e.Error) {
	if sourceTemplateId == "" {
		return nil, nil
	}
	source := models.NotificationTemplate{}
	if err := tx.
		Where("id = ? and org_id = ? and (project_id = '' or project_id is null) and deleted_at_t = 0", sourceTemplateId, c.OrgId).
		First(&source); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, nil
		}
		return nil, e.New(e.DBError, err)
	}
	return &source, nil
}

func ensureNotificationTemplateRollbackTargetAvailable(tx *db.Session, template *models.NotificationTemplate, version *models.NotificationTemplateVersion) e.Error {
	query := tx.
		Where("org_id = ? and event_type = ? and notification_type = ? and id <> ? and deleted_at_t = 0",
			template.OrgId, version.EventType, version.NotificationType, template.Id)
	if template.ProjectId != "" {
		query = query.Where("project_id = ?", template.ProjectId)
	} else {
		query = query.Where("(project_id = '' or project_id is null)")
	}
	existing := models.NotificationTemplate{}
	if err := query.First(&existing); err == nil {
		return e.New(e.BadParam, fmt.Errorf("notification template event type and notification type already exists"))
	} else if !e.IsRecordNotFound(err) {
		return e.New(e.DBError, err)
	}
	return nil
}

func notificationTemplateSourceFromVersion(version *models.NotificationTemplateVersion) *models.NotificationTemplate {
	if version == nil || version.SourceTemplateId == "" {
		return nil
	}
	source := &models.NotificationTemplate{
		OrgId:            version.OrgId,
		Name:             version.SourceName,
		EventType:        version.SourceEventType,
		NotificationType: version.SourceNotificationType,
		Title:            version.SourceTitle,
		Content:          version.SourceContent,
		MarkdownContent:  version.SourceMarkdownContent,
		Status:           version.SourceStatus,
	}
	source.Id = version.SourceTemplateId
	return source
}

func recordNotificationTemplateVersion(tx *db.Session, c *ctx.ServiceContext, template *models.NotificationTemplate, source *models.NotificationTemplate, action string) e.Error {
	if tx == nil || c == nil || template == nil || template.Id == "" {
		return nil
	}
	maxVersion := 0
	if err := tx.Model(&models.NotificationTemplateVersion{}).
		Where("template_id = ? and deleted_at_t = 0", template.Id).
		Select("coalesce(max(version_no), 0)").
		Row().
		Scan(&maxVersion); err != nil {
		return e.New(e.DBError, err)
	}
	version := models.NotificationTemplateVersion{
		OrgId:            template.OrgId,
		ProjectId:        template.ProjectId,
		TemplateId:       template.Id,
		VersionNo:        maxVersion + 1,
		Action:           action,
		Name:             template.Name,
		EventType:        template.EventType,
		NotificationType: template.NotificationType,
		Title:            template.Title,
		Content:          template.Content,
		MarkdownContent:  template.MarkdownContent,
		Status:           template.Status,
		SourceTemplateId: template.SourceTemplateId,
		Operator:         c.UserId,
	}
	version.Id = models.NewId("ntplv")
	if source != nil {
		version.SourceTemplateId = source.Id
		version.SourceName = source.Name
		version.SourceEventType = source.EventType
		version.SourceNotificationType = source.NotificationType
		version.SourceTitle = source.Title
		version.SourceContent = source.Content
		version.SourceMarkdownContent = source.MarkdownContent
		version.SourceStatus = source.Status
	}
	if err := models.Create(tx, &version); err != nil {
		return e.New(e.DBError, err)
	}
	return nil
}

func validateNotificationEventTypes(eventTypes []string) e.Error {
	allowedPrefixes := []string{
		"task.",
		"cloud.",
		"cloud_",
		"account.",
		"sync.",
		"operation.",
		"risk.",
		"cost.",
		"cmdb.",
		"webhook.",
		"notification.",
	}
	for _, eventType := range eventTypes {
		eventType = strings.TrimSpace(eventType)
		if eventType == "" {
			return e.New(e.BadParam, fmt.Errorf("notification event type is required"))
		}
		if len(eventType) > 128 {
			return e.New(e.BadParam, fmt.Errorf("notification event type %s is too long", eventType))
		}
		allowed := false
		for _, prefix := range allowedPrefixes {
			if strings.HasPrefix(eventType, prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return e.New(e.BadParam, fmt.Errorf("unsupported notification event type %s", eventType))
		}
	}
	return nil
}

func validateNotificationTemplateForm(form *forms.CreateNotificationTemplateForm) e.Error {
	if err := validateNotificationEventTypes([]string{form.EventType}); err != nil {
		return err
	}
	if strings.TrimSpace(form.Content) == "" && strings.TrimSpace(form.MarkdownContent) == "" {
		return e.New(e.BadParam, fmt.Errorf("notification template content is required"))
	}
	if err := parseNotificationTemplateSyntax("title", form.Title); err != nil {
		return e.New(e.BadParam, err)
	}
	if err := parseNotificationTemplateSyntax("content", form.Content); err != nil {
		return e.New(e.BadParam, err)
	}
	if err := parseNotificationTemplateSyntax("markdownContent", form.MarkdownContent); err != nil {
		return e.New(e.BadParam, err)
	}
	return nil
}

func notificationTemplateStatus(status string) string {
	if strings.TrimSpace(status) == "" {
		return models.NotificationTemplateStatusEnable
	}
	return strings.TrimSpace(status)
}

func validateNotificationTemplateSource(c *ctx.ServiceContext, form *forms.CreateNotificationTemplateForm) (*models.NotificationTemplate, e.Error) {
	if form.SourceTemplateId == "" {
		return nil, nil
	}
	if c.ProjectId == "" {
		return nil, e.New(e.BadParam, fmt.Errorf("sourceTemplateId is only supported for project templates"))
	}
	source := models.NotificationTemplate{}
	if err := c.DB().
		Where("id = ? and org_id = ? and (project_id = '' or project_id is null) and deleted_at_t = 0", form.SourceTemplateId, c.OrgId).
		First(&source); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, err)
	}
	if source.EventType != strings.TrimSpace(form.EventType) || source.NotificationType != form.Type {
		return nil, e.New(e.BadParam, fmt.Errorf("source template event type or notification type mismatch"))
	}
	return &source, nil
}

func parseNotificationTemplateSyntax(field, tpl string) error {
	if strings.TrimSpace(tpl) == "" {
		return nil
	}
	if _, err := template.New(field).Parse(tpl); err != nil {
		return fmt.Errorf("%s template parse failed: %w", field, err)
	}
	return nil
}

func renderNotificationTemplateField(field, tpl string, data map[string]interface{}) (string, error) {
	if strings.TrimSpace(tpl) == "" {
		return "", nil
	}
	t, err := template.New(field).Option("missingkey=zero").Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("%s template parse failed: %w", field, err)
	}
	buf := bytes.Buffer{}
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%s template execute failed: %w", field, err)
	}
	return buf.String(), nil
}

func notificationTemplateSampleData(c *ctx.ServiceContext, eventType string) map[string]interface{} {
	orgId := ""
	if c != nil {
		orgId = c.OrgId.String()
	}
	if orgId == "" {
		orgId = "org-sample"
	}
	sample := map[string]interface{}{
		"Creator":      "admin@example.com",
		"OrgName":      "示例组织",
		"ProjectName":  "示例项目",
		"TemplateName": "示例云模板",
		"Revision":     "main",
		"EnvName":      "production",
		"Addr":         fmt.Sprintf("http://127.0.0.1/org/%s/project/p-sample/m-project-env/detail/env-sample/task/run-sample", orgId),
		"ResAdded":     1,
		"ResChanged":   2,
		"ResDestroyed": 0,
		"Message":      "示例通知消息",
		"TaskType":     "plan",
		"ViewUrl":      fmt.Sprintf("http://127.0.0.1/org/%s/m-cloud-events", orgId),
		"Source":       "operation",
		"EventType":    "cloud_operation.failed",
		"Level":        "error",
		"Status":       "failed",
		"Provider":     "aws",
		"AccountId":    "123456789012",
		"Region":       "ap-southeast-1",
		"ResourceType": "aws_instance",
		"ResourceId":   "i-0123456789abcdef0",
		"ResourceName": "sample-ec2",
		"Title":        "示例多云事件",
		"Payload":      map[string]interface{}{},
	}
	notificationTemplateApplyBatchRerunEventSample(sample, eventType, true)
	event := notificationTemplateLatestCloudEvent(c, eventType)
	if event == nil {
		return sample
	}
	sample["ViewUrl"] = cloudEventNotificationViewUrl(*event)
	sample["Source"] = event.Source
	sample["EventType"] = event.EventType
	sample["Level"] = event.Level
	sample["Status"] = event.Status
	sample["Provider"] = event.Provider
	sample["AccountId"] = event.AccountId
	sample["Region"] = event.Region
	sample["ResourceType"] = event.ResourceType
	sample["ResourceId"] = event.ResourceId
	sample["ResourceName"] = event.ResourceName
	sample["Title"] = event.Title
	sample["Message"] = event.Message
	sample["ProjectId"] = event.ProjectId.String()
	sample["EnvId"] = event.EnvId.String()
	sample["AssetId"] = event.AssetId.String()
	sample["OperationId"] = event.OperationId.String()
	sample["RiskFindingId"] = event.RiskFindingId.String()
	sample["CloudAccountId"] = event.CloudAccountId.String()
	sample["ActorId"] = event.ActorId.String()
	sample["UserIp"] = event.UserIp
	sample["OccurredAt"] = event.OccurredAt
	if event.Payload != nil {
		sample["Payload"] = map[string]interface{}(event.Payload)
	}
	notificationTemplateApplyBatchRerunEventSample(sample, event.EventType, false)
	return sample
}

func notificationTemplateVariableGroups(c *ctx.ServiceContext, eventType string, sampleData map[string]interface{}) []resps.NotificationTemplateVariableGroup {
	taskGroup := resps.NotificationTemplateVariableGroup{
		Name:  "task",
		Label: "任务通知变量",
		Variables: []resps.NotificationTemplateVariable{
			{Name: "Creator", Label: "创建人", Description: "任务创建人", Sample: "admin@example.com"},
			{Name: "OrgName", Label: "组织名称", Description: "任务所属组织", Sample: "示例组织"},
			{Name: "ProjectName", Label: "项目名称", Description: "任务所属项目", Sample: "示例项目"},
			{Name: "TemplateName", Label: "云模板名称", Description: "任务使用的云模板", Sample: "示例云模板"},
			{Name: "Revision", Label: "代码版本", Description: "任务使用的分支、标签或提交", Sample: "main"},
			{Name: "EnvName", Label: "环境名称", Description: "任务所属环境", Sample: "production"},
			{Name: "Addr", Label: "任务链接", Description: "任务详情页面地址", Sample: "http://127.0.0.1/org/org-sample/project/p-sample/m-project-env/detail/env-sample/task/run-sample"},
			{Name: "ResAdded", Label: "新增资源数", Description: "Terraform 结果新增资源数", Sample: 1},
			{Name: "ResChanged", Label: "变更资源数", Description: "Terraform 结果变更资源数", Sample: 2},
			{Name: "ResDestroyed", Label: "销毁资源数", Description: "Terraform 结果销毁资源数", Sample: 0},
			{Name: "Message", Label: "任务消息", Description: "任务状态或错误消息", Sample: "示例通知消息"},
			{Name: "TaskType", Label: "任务类型", Description: "plan、apply 或 destroy", Sample: "plan"},
		},
	}
	eventGroup := resps.NotificationTemplateVariableGroup{
		Name:  "cloudEvent",
		Label: "多云事件变量",
		Variables: []resps.NotificationTemplateVariable{
			{Name: "ViewUrl", Label: "事件链接", Description: "事件中心页面地址", Sample: "http://127.0.0.1/org/org-sample/m-cloud-events"},
			{Name: "Source", Label: "事件来源", Description: "account、operation、risk、cost、cmdb 等来源", Sample: "operation"},
			{Name: "EventType", Label: "事件类型", Description: "平台事件类型", Sample: "cloud_operation.failed"},
			{Name: "Level", Label: "事件级别", Description: "info、warning 或 error", Sample: "error"},
			{Name: "Status", Label: "事件状态", Description: "事件业务状态", Sample: "failed"},
			{Name: "Provider", Label: "云厂商", Description: "云厂商标识", Sample: "aws"},
			{Name: "AccountId", Label: "云账号", Description: "云厂商账号 ID", Sample: "123456789012"},
			{Name: "Region", Label: "区域", Description: "云资源区域", Sample: "ap-southeast-1"},
			{Name: "ResourceType", Label: "资源类型", Description: "云资源类型", Sample: "aws_instance"},
			{Name: "ResourceId", Label: "资源 ID", Description: "云资源原生 ID", Sample: "i-0123456789abcdef0"},
			{Name: "ResourceName", Label: "资源名称", Description: "云资源名称", Sample: "sample-ec2"},
			{Name: "Title", Label: "事件标题", Description: "事件中心标题", Sample: "示例多云事件"},
			{Name: "Message", Label: "事件消息", Description: "事件中心消息内容", Sample: "示例通知消息"},
		},
	}
	dynamicPayloadGroup := notificationTemplatePayloadVariableGroup(c, eventType, sampleData)
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		groups := []resps.NotificationTemplateVariableGroup{taskGroup, eventGroup}
		if dynamicPayloadGroup != nil {
			groups = append(groups, *dynamicPayloadGroup)
		}
		return groups
	}
	if strings.HasPrefix(eventType, "task.") {
		return []resps.NotificationTemplateVariableGroup{taskGroup}
	}
	groups := []resps.NotificationTemplateVariableGroup{eventGroup}
	if batchRerunGroup := notificationTemplateBatchRerunVariableGroup(eventType, sampleData); batchRerunGroup != nil {
		groups = append(groups, *batchRerunGroup)
	}
	if dynamicPayloadGroup != nil {
		groups = append(groups, *dynamicPayloadGroup)
	}
	return groups
}

func notificationTemplateApplyBatchRerunEventSample(sample map[string]interface{}, eventType string, overwrite bool) {
	if sample == nil || !notificationTemplateIsBatchRerunEventType(eventType) {
		return
	}
	eventType = strings.TrimSpace(eventType)
	if eventType == "" || strings.HasSuffix(eventType, ".*") {
		eventType = "cloud.sync.task.batch_rerun_finished"
	}
	setNotificationTemplateSampleValue(sample, "Source", "sync", overwrite)
	setNotificationTemplateSampleValue(sample, "EventType", eventType, overwrite)
	setNotificationTemplateSampleValue(sample, "Provider", "tencentcloud", overwrite)
	setNotificationTemplateSampleValue(sample, "AccountId", "codex-tencent-account", overwrite)
	setNotificationTemplateSampleValue(sample, "ResourceType", "cmdb_sync_task_group", overwrite)
	setNotificationTemplateSampleValue(sample, "ResourceId", "csr-sample-rerun-group", overwrite)
	setNotificationTemplateSampleValue(sample, "ResourceName", "csr-sample-rerun-group", overwrite)

	status := models.CmdbSyncTaskComplete
	level := models.CloudEventLevelInfo
	title := "云采集失败任务批量重跑已完成"
	message := "任务组 csr-sample-rerun-group 已结束，成功 2 个，失败 0 个"
	if eventType == "cloud.sync.task.batch_rerun_approval_requested" {
		status = models.CmdbSyncTaskApproving
		title = "云采集失败任务批量重跑等待审批"
		message = "已创建 2 个失败任务重跑审批任务"
	} else if eventType == "cloud.sync.task.batch_rerun_started" {
		status = models.CmdbSyncTaskRunning
		title = "云采集失败任务批量重跑已启动"
		message = "已创建 2 个失败任务重跑任务"
	} else if eventType == "cloud.sync.task.batch_rerun_finished" {
		status = models.CmdbSyncTaskComplete
	}
	setNotificationTemplateSampleValue(sample, "Status", status, overwrite)
	setNotificationTemplateSampleValue(sample, "Level", level, overwrite)
	setNotificationTemplateSampleValue(sample, "Title", title, overwrite)
	setNotificationTemplateSampleValue(sample, "Message", message, overwrite)

	payload := notificationTemplateEnsurePayloadSample(sample)
	requiresApproval := eventType == "cloud.sync.task.batch_rerun_approval_requested"
	approvingCount := 0
	completeCount := 2
	if requiresApproval {
		approvingCount = 2
		completeCount = 0
	}
	setNotificationTemplateSampleValue(payload, "rerunGroupId", "csr-sample-rerun-group", overwrite)
	setNotificationTemplateSampleValue(payload, "reason", "示例批量重跑原因", overwrite)
	setNotificationTemplateSampleValue(payload, "mode", "batch_failed", overwrite)
	setNotificationTemplateSampleValue(payload, "created", 2, overwrite)
	setNotificationTemplateSampleValue(payload, "total", 2, overwrite)
	setNotificationTemplateSampleValue(payload, "requiresApproval", requiresApproval, overwrite)
	setNotificationTemplateSampleValue(payload, "approvingCount", approvingCount, overwrite)
	setNotificationTemplateSampleValue(payload, "pendingCount", 0, overwrite)
	setNotificationTemplateSampleValue(payload, "runningCount", 0, overwrite)
	setNotificationTemplateSampleValue(payload, "completeCount", completeCount, overwrite)
	setNotificationTemplateSampleValue(payload, "failedCount", 0, overwrite)
	setNotificationTemplateSampleValue(payload, "rejectedCount", 0, overwrite)
	setNotificationTemplateSampleValue(payload, "createdAt", "2026-06-21T10:00:00Z", overwrite)
	setNotificationTemplateSampleValue(payload, "startedAt", "2026-06-21T10:00:02Z", overwrite)
	setNotificationTemplateSampleValue(payload, "endedAt", "2026-06-21T10:00:12Z", overwrite)
	setNotificationTemplateSampleValue(payload, "sourceTaskIds", []string{"cst-source-a", "cst-source-b"}, overwrite)
	setNotificationTemplateSampleValue(payload, "taskIds", []string{"cst-rerun-a", "cst-rerun-b"}, overwrite)
}

func notificationTemplateBatchRerunVariableGroup(eventType string, sampleData map[string]interface{}) *resps.NotificationTemplateVariableGroup {
	if !notificationTemplateIsBatchRerunEventType(eventType) {
		return nil
	}
	payload := notificationTemplatePayloadSample(sampleData)
	variables := []resps.NotificationTemplateVariable{
		{Name: "Payload.rerunGroupId", Label: "重跑任务组 ID", Description: "批量重跑任务组 ID，可用于跳转或关联事件", Sample: payload["rerunGroupId"]},
		{Name: "Payload.reason", Label: "重跑原因", Description: "用户提交批量重跑时填写的原因", Sample: payload["reason"]},
		{Name: "Payload.mode", Label: "重跑模式", Description: "当前为 batch_failed，表示批量重跑失败任务", Sample: payload["mode"]},
		{Name: "Payload.created", Label: "已创建任务数", Description: "启动事件中本次创建的新采集任务数量", Sample: payload["created"]},
		{Name: "Payload.total", Label: "任务总数", Description: "任务组内新采集任务总数", Sample: payload["total"]},
		{Name: "Payload.requiresApproval", Label: "是否需要审批", Description: "本次任务组是否需要审批后启动", Sample: payload["requiresApproval"]},
		{Name: "Payload.approvingCount", Label: "待审批任务数", Description: "任务组内仍在等待审批的新采集任务数量", Sample: payload["approvingCount"]},
		{Name: "Payload.completeCount", Label: "成功任务数", Description: "任务组内成功完成的新采集任务数量", Sample: payload["completeCount"]},
		{Name: "Payload.failedCount", Label: "失败任务数", Description: "任务组内失败的新采集任务数量", Sample: payload["failedCount"]},
		{Name: "Payload.rejectedCount", Label: "驳回任务数", Description: "任务组内被审批驳回的新采集任务数量", Sample: payload["rejectedCount"]},
		{Name: "Payload.runningCount", Label: "运行中任务数", Description: "任务组内仍在运行的新采集任务数量", Sample: payload["runningCount"]},
		{Name: "Payload.pendingCount", Label: "等待中任务数", Description: "任务组内仍在等待的新采集任务数量", Sample: payload["pendingCount"]},
		{Name: "Payload.createdAt", Label: "组创建时间", Description: "任务组内最早任务创建时间", Sample: payload["createdAt"]},
		{Name: "Payload.startedAt", Label: "组开始时间", Description: "任务组内最早任务开始时间", Sample: payload["startedAt"]},
		{Name: "Payload.endedAt", Label: "组结束时间", Description: "任务组内最近任务结束时间", Sample: payload["endedAt"]},
		{Name: "Payload.sourceTaskIds", Label: "源失败任务 ID", Description: "被批量重跑的源失败任务 ID 列表", Sample: payload["sourceTaskIds"]},
		{Name: "Payload.taskIds", Label: "新任务 ID", Description: "批量重跑创建的新采集任务 ID 列表", Sample: payload["taskIds"]},
	}
	return &resps.NotificationTemplateVariableGroup{
		Name:      "syncBatchRerun",
		Label:     "批量重跑任务组变量",
		Variables: variables,
	}
}

func notificationTemplateIsBatchRerunEventType(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "cloud.sync.task.batch_rerun_approval_requested", "cloud.sync.task.batch_rerun_started", "cloud.sync.task.batch_rerun_finished":
		return true
	default:
		return false
	}
}

func notificationTemplateEnsurePayloadSample(sample map[string]interface{}) map[string]interface{} {
	if sample == nil {
		return map[string]interface{}{}
	}
	payload := notificationTemplatePayloadSample(sample)
	if payload == nil {
		payload = map[string]interface{}{}
	}
	sample["Payload"] = payload
	return payload
}

func notificationTemplatePayloadSample(sample map[string]interface{}) map[string]interface{} {
	if sample == nil {
		return nil
	}
	switch payload := sample["Payload"].(type) {
	case map[string]interface{}:
		return payload
	case models.ResAttrs:
		return map[string]interface{}(payload)
	default:
		return nil
	}
}

func setNotificationTemplateSampleValue(sample map[string]interface{}, key string, value interface{}, overwrite bool) {
	if sample == nil || key == "" {
		return
	}
	if !overwrite {
		if _, exists := sample[key]; exists {
			return
		}
	}
	sample[key] = value
}

func notificationTemplatePayloadVariableGroup(c *ctx.ServiceContext, eventType string, sampleData map[string]interface{}) *resps.NotificationTemplateVariableGroup {
	if strings.HasPrefix(strings.TrimSpace(eventType), "task.") {
		return nil
	}
	events := notificationTemplateRecentCloudEvents(c, eventType, 20)
	if len(events) == 0 {
		return nil
	}
	payloadSample := map[string]interface{}{}
	if sampleData != nil {
		if raw, ok := sampleData["Payload"].(map[string]interface{}); ok {
			payloadSample = raw
		}
	}
	fields := map[string]resps.NotificationTemplateVariable{}
	for _, event := range events {
		collectNotificationTemplatePayloadVariables(fields, "Payload", event.Payload, payloadSample, 0)
	}
	if len(fields) == 0 {
		return nil
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	variables := make([]resps.NotificationTemplateVariable, 0, len(names))
	for _, name := range names {
		variables = append(variables, fields[name])
	}
	return &resps.NotificationTemplateVariableGroup{
		Name:      "eventPayload",
		Label:     "真实事件载荷变量",
		Variables: variables,
	}
}

func collectNotificationTemplatePayloadVariables(fields map[string]resps.NotificationTemplateVariable, prefix string, value interface{}, sampleRoot map[string]interface{}, depth int) {
	if fields == nil || depth > 2 {
		return
	}
	payload, ok := value.(models.ResAttrs)
	if ok {
		value = map[string]interface{}(payload)
	}
	items, ok := value.(map[string]interface{})
	if !ok {
		return
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		if isNotificationTemplateVariableSegment(key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		item := items[key]
		name := prefix + "." + key
		if _, exists := fields[name]; !exists {
			fields[name] = resps.NotificationTemplateVariable{
				Name:        name,
				Label:       name,
				Description: "从最近真实多云事件 payload 推导的变量",
				Sample:      item,
			}
		}
		if prefix == "Payload" && sampleRoot != nil {
			if _, exists := sampleRoot[key]; !exists {
				sampleRoot[key] = item
			}
		}
		if nested, ok := item.(map[string]interface{}); ok {
			collectNotificationTemplatePayloadVariables(fields, name, nested, sampleRoot, depth+1)
		}
	}
}

func isNotificationTemplateVariableSegment(key string) bool {
	if key == "" {
		return false
	}
	for index, r := range key {
		if r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			continue
		}
		if index > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func notificationTemplateLatestCloudEvent(c *ctx.ServiceContext, eventType string) *models.CloudEvent {
	events := notificationTemplateRecentCloudEvents(c, eventType, 1)
	if len(events) == 0 {
		return nil
	}
	return &events[0]
}

func notificationTemplateRecentCloudEvents(c *ctx.ServiceContext, eventType string, limit int) []models.CloudEvent {
	if c == nil || c.OrgId == "" {
		return nil
	}
	eventType = strings.TrimSpace(eventType)
	if strings.HasPrefix(eventType, "task.") {
		return nil
	}
	if limit <= 0 {
		limit = 10
	}
	query := c.DB().Model(&models.CloudEvent{}).
		Where("org_id = ? and deleted_at_t = 0", c.OrgId)
	if c.ProjectId != "" {
		query = query.Where("(project_id = '' or project_id is null or project_id = ?)", c.ProjectId)
	}
	query = applyNotificationTemplateCloudEventSchemaFilter(query, eventType)
	events := make([]models.CloudEvent, 0)
	if err := query.Order("occurred_at desc").Order("created_at desc").Limit(limit).Find(&events); err != nil {
		c.Logger().Warnf("query notification template event schema failed: %v", err)
		return nil
	}
	return events
}

func applyNotificationTemplateCloudEventSchemaFilter(query *db.Session, eventType string) *db.Session {
	if eventType == "" || eventType == "cloud.*" {
		return query
	}
	if strings.HasSuffix(eventType, ".*") {
		prefix := strings.TrimSuffix(eventType, ".*")
		likePattern := prefix + ".%"
		cloudLikePattern := "cloud_" + prefix + ".%"
		return query.Where("(source = ? or event_type like ? or event_type like ?)", prefix, likePattern, cloudLikePattern)
	}
	return query.Where("event_type = ?", eventType)
}
