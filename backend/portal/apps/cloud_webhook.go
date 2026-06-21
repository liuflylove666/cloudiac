// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package apps

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloudiac/portal/consts"
	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/libs/db"
	"cloudiac/portal/libs/page"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	"cloudiac/portal/models/resps"
	"cloudiac/portal/services"
	cloudwebhooksrv "cloudiac/portal/services/cloudwebhook"
	"cloudiac/utils/logs"
)

type cloudWebhookDeliveryOptions struct {
	Attempt          int
	ParentDeliveryId models.Id
	DeliveryMode     string
	ScheduleRetry    bool
}

type cloudWebhookRetryOptions struct {
	ShardIndex           int
	ShardTotal           int
	SkipQueueMaintenance bool
}

func (o cloudWebhookRetryOptions) sharded() bool {
	return o.ShardTotal > 1
}

const (
	CloudWebhookSignatureReportTokenHeader = "X-CloudIaC-Signature-Report-Token"

	cloudWebhookRetryWorkerInterval            = 15 * time.Second
	cloudWebhookQueueMaintenanceWorkerInterval = 15 * time.Second
	cloudWebhookRetryDefaultShardTotal         = 4
	cloudWebhookRetryMaxShardTotal             = 128
	cloudWebhookDefaultSecretGraceSeconds      = 7 * 24 * 3600
	cloudWebhookMaxSecretGraceSeconds          = 90 * 24 * 3600
	cloudWebhookSignatureReportTokenBytes      = 32
	cloudWebhookSignatureReportTokenTTL        = 7 * 24 * time.Hour
	cloudWebhookDeliveryQueueRetention         = 7 * 24 * time.Hour
)

func SearchCloudWebhooks(c *ctx.ServiceContext, form *forms.SearchCloudWebhookForm) (interface{}, e.Error) {
	if _, err := RetryDueCloudWebhookDeliveries(c); err != nil {
		return nil, err
	}
	query := applyCloudWebhookSearch(cloudWebhookVisibleQuery(c), form)
	if form.SortField() == "" {
		query = query.Order("created_at desc")
	} else {
		query = form.Order(query)
	}
	webhooks := make([]models.CloudWebhook, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&webhooks); err != nil {
		return nil, e.New(e.DBError, err)
	}
	list := make([]resps.CloudWebhookResp, 0, len(webhooks))
	for _, webhook := range webhooks {
		list = append(list, cloudWebhookResp(c, webhook))
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func CreateCloudWebhook(c *ctx.ServiceContext, form *forms.CreateCloudWebhookForm) (*resps.CloudWebhookResp, e.Error) {
	webhook, err := cloudWebhookFromCreateForm(c, form)
	if err != nil {
		return nil, err
	}
	webhook.Id = models.NewId("cwh")
	if err := models.Create(c.DB(), webhook); err != nil {
		return nil, e.New(e.DBError, err)
	}
	resp := cloudWebhookResp(c, *webhook)
	return &resp, nil
}

func UpdateCloudWebhook(c *ctx.ServiceContext, form *forms.UpdateCloudWebhookForm) (*resps.CloudWebhookResp, e.Error) {
	existing := models.CloudWebhook{}
	if err := cloudWebhookVisibleQuery(c).Where("id = ?", form.Id).First(&existing); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	webhook, err := cloudWebhookFromCreateForm(c, &form.CreateCloudWebhookForm)
	if err != nil {
		return nil, err
	}
	attrs := models.Attrs{
		"name":                 webhook.Name,
		"description":          webhook.Description,
		"target_url":           webhook.TargetUrl,
		"event_types":          webhook.EventTypes,
		"sources":              webhook.Sources,
		"status":               webhook.Status,
		"timeout_seconds":      webhook.TimeoutSeconds,
		"max_retries":          webhook.MaxRetries,
		"retry_interval":       webhook.RetryInterval,
		"retry_jitter_percent": webhook.RetryJitterPercent,
		"max_retry_duration":   webhook.MaxRetryDuration,
	}
	if form.HasKey("secret") {
		for key, value := range cloudWebhookSecretRotationAttrs(existing, webhook.Secret, cloudWebhookDefaultSecretGraceSeconds) {
			attrs[key] = value
		}
	}
	if _, err := c.DB().Model(&models.CloudWebhook{}).
		Where("id = ? and org_id = ?", form.Id, c.OrgId).
		UpdateAttrs(attrs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	existing.Name = webhook.Name
	existing.Description = webhook.Description
	existing.TargetUrl = webhook.TargetUrl
	if form.HasKey("secret") {
		existing.Secret = attrs["secret"].(string)
		existing.SecretVersion = attrs["secret_version"].(int)
		existing.PreviousSecret = attrs["previous_secret"].(string)
		existing.PreviousSecretVersion = attrs["previous_secret_version"].(int)
		existing.PreviousSecretExpiresAt = attrs["previous_secret_expires_at"].(models.Time)
		existing.SecretRotatedAt = attrs["secret_rotated_at"].(models.Time)
	}
	existing.EventTypes = webhook.EventTypes
	existing.Sources = webhook.Sources
	existing.Status = webhook.Status
	existing.TimeoutSeconds = webhook.TimeoutSeconds
	existing.MaxRetries = webhook.MaxRetries
	existing.RetryInterval = webhook.RetryInterval
	existing.RetryJitterPercent = webhook.RetryJitterPercent
	existing.MaxRetryDuration = webhook.MaxRetryDuration
	resp := cloudWebhookResp(c, existing)
	return &resp, nil
}

func RotateCloudWebhookSecret(c *ctx.ServiceContext, form *forms.RotateCloudWebhookSecretForm) (*resps.CloudWebhookResp, e.Error) {
	webhook := models.CloudWebhook{}
	if err := cloudWebhookVisibleQuery(c).Where("id = ?", form.Id).First(&webhook); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	nextSecret := strings.TrimSpace(form.Secret)
	if nextSecret == "" {
		return nil, e.New(e.BadParam, fmt.Errorf("secret is required"), http.StatusBadRequest)
	}
	attrs := cloudWebhookSecretRotationAttrs(webhook, nextSecret, form.GracePeriodSeconds)
	if _, err := c.DB().Model(&models.CloudWebhook{}).
		Where("id = ? and org_id = ?", webhook.Id, c.OrgId).
		UpdateAttrs(attrs); err != nil {
		return nil, e.New(e.DBError, err)
	}
	webhook.Secret = nextSecret
	webhook.SecretVersion = attrs["secret_version"].(int)
	webhook.PreviousSecret = attrs["previous_secret"].(string)
	webhook.PreviousSecretVersion = attrs["previous_secret_version"].(int)
	webhook.PreviousSecretExpiresAt = attrs["previous_secret_expires_at"].(models.Time)
	webhook.SecretRotatedAt = attrs["secret_rotated_at"].(models.Time)
	resp := cloudWebhookResp(c, webhook)
	return &resp, nil
}

func DeleteCloudWebhook(c *ctx.ServiceContext, form *forms.CloudWebhookParam) (interface{}, e.Error) {
	if _, err := cloudWebhookVisibleQuery(c).Where("id = ?", form.Id).Delete(&models.CloudWebhook{}); err != nil {
		return nil, e.New(e.DBError, err)
	}
	return nil, nil
}

func SearchCloudWebhookDeliveries(c *ctx.ServiceContext, form *forms.SearchCloudWebhookDeliveryForm) (interface{}, e.Error) {
	if _, err := RetryDueCloudWebhookDeliveries(c); err != nil {
		return nil, err
	}
	query := c.DB().Model(&models.CloudWebhookDelivery{}).
		Where("org_id = ? and webhook_id = ?", c.OrgId, form.Id)
	if form.EventId != "" {
		query = query.Where("event_id = ?", form.EventId)
	}
	if form.EventType != "" {
		query = query.Where("event_type = ?", form.EventType)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	if form.SortField() == "" {
		query = query.Order("created_at desc")
	} else {
		query = form.Order(query)
	}
	deliveries := make([]models.CloudWebhookDelivery, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&deliveries); err != nil {
		return nil, e.New(e.DBError, err)
	}
	webhookName := lookupName(c, &models.CloudWebhook{}, form.Id)
	list := make([]resps.CloudWebhookDeliveryResp, 0, len(deliveries))
	for _, delivery := range deliveries {
		list = append(list, resps.CloudWebhookDeliveryResp{
			CloudWebhookDelivery: delivery,
			WebhookName:          webhookName,
		})
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func GetCloudWebhookDelivery(c *ctx.ServiceContext, form *forms.CloudWebhookDeliveryParam) (*resps.CloudWebhookDeliveryResp, e.Error) {
	delivery := models.CloudWebhookDelivery{}
	if err := c.DB().Model(&models.CloudWebhookDelivery{}).
		Where("org_id = ? and webhook_id = ? and id = ?", c.OrgId, form.Id, form.DeliveryId).
		First(&delivery); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	resp := resps.CloudWebhookDeliveryResp{
		CloudWebhookDelivery: delivery,
		WebhookName:          lookupName(c, &models.CloudWebhook{}, form.Id),
	}
	auditCloudWebhookDeliveryDetail(c, delivery, resp.WebhookName)
	return &resp, nil
}

func ReportCloudWebhookSignatureVerification(c *ctx.ServiceContext, form *forms.ReportCloudWebhookSignatureVerificationForm) (*resps.CloudWebhookDeliveryResp, e.Error) {
	delivery := models.CloudWebhookDelivery{}
	if err := c.DB().Model(&models.CloudWebhookDelivery{}).
		Where("org_id = ? and id = ?", c.OrgId, form.DeliveryId).
		First(&delivery); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	now := models.Time(time.Now())
	if _, err := c.DB().Model(&models.CloudWebhookDelivery{}).
		Where("org_id = ? and id = ?", c.OrgId, delivery.Id).
		UpdateAttrs(models.Attrs{
			"signature_verify_status":  form.Status,
			"signature_verify_version": form.SignatureVersion,
			"signature_verify_message": strings.TrimSpace(form.Message),
			"signature_verified_at":    now,
		}); err != nil {
		return nil, e.New(e.DBError, err)
	}
	delivery.SignatureVerifyStatus = form.Status
	delivery.SignatureVerifyVersion = form.SignatureVersion
	delivery.SignatureVerifyMessage = strings.TrimSpace(form.Message)
	delivery.SignatureVerifiedAt = now
	cloudWebhookSignatureVerificationEvent(c, delivery, "platform")
	resp := resps.CloudWebhookDeliveryResp{
		CloudWebhookDelivery: delivery,
		WebhookName:          lookupName(c, &models.CloudWebhook{}, delivery.WebhookId),
	}
	return &resp, nil
}

func ReportCloudWebhookSignatureVerificationByToken(c *ctx.ServiceContext, form *forms.ReportCloudWebhookSignatureVerificationByTokenForm) (models.ResAttrs, e.Error) {
	delivery := models.CloudWebhookDelivery{}
	if err := c.DB().Model(&models.CloudWebhookDelivery{}).
		Where("id = ?", form.DeliveryId).
		First(&delivery); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	if !cloudWebhookSignatureReportTokenMatches(delivery.SignatureReportTokenHash, form.Token) {
		return nil, e.New(e.PermissionDeny, fmt.Errorf("invalid signature report token"), http.StatusForbidden)
	}
	nowTime := time.Now()
	if expiresAt := time.Time(delivery.SignatureReportTokenExpiresAt); expiresAt.IsZero() || !expiresAt.After(nowTime) {
		return nil, e.New(e.PermissionDeny, fmt.Errorf("signature report token expired"), http.StatusForbidden)
	}
	if usedAt := time.Time(delivery.SignatureReportTokenUsedAt); !usedAt.IsZero() {
		return nil, e.New(e.BadParam, fmt.Errorf("signature report token already used"), http.StatusConflict)
	}
	now := models.Time(nowTime)
	message := strings.TrimSpace(form.Message)
	updated, err := c.DB().Model(&models.CloudWebhookDelivery{}).
		Where(
			"id = ? and signature_report_token_hash = ? and signature_report_token_used_at is null and signature_report_token_expires_at > ?",
			delivery.Id, delivery.SignatureReportTokenHash, now,
		).
		UpdateAttrs(models.Attrs{
			"signature_verify_status":        form.Status,
			"signature_verify_version":       form.SignatureVersion,
			"signature_verify_message":       message,
			"signature_verified_at":          now,
			"signature_report_token_used_at": now,
		})
	if err != nil {
		return nil, e.New(e.DBError, err)
	}
	if updated == 0 {
		return nil, e.New(e.BadParam, fmt.Errorf("signature report token already used or expired"), http.StatusConflict)
	}
	delivery.SignatureVerifyStatus = form.Status
	delivery.SignatureVerifyVersion = form.SignatureVersion
	delivery.SignatureVerifyMessage = message
	delivery.SignatureVerifiedAt = now
	delivery.SignatureReportTokenUsedAt = now
	cloudWebhookSignatureVerificationEvent(c, delivery, "token")
	return models.ResAttrs{
		"deliveryId":             delivery.Id.String(),
		"signatureVerifyStatus":  form.Status,
		"signatureVerifyVersion": form.SignatureVersion,
		"signatureVerifyMessage": message,
		"signatureVerifiedAt":    now,
	}, nil
}

func RetryCloudWebhookDelivery(c *ctx.ServiceContext, form *forms.RetryCloudWebhookDeliveryForm) (*resps.CloudWebhookDeliveryResp, e.Error) {
	original := models.CloudWebhookDelivery{}
	if err := c.DB().Model(&models.CloudWebhookDelivery{}).
		Where("org_id = ? and webhook_id = ? and id = ?", c.OrgId, form.Id, form.DeliveryId).
		First(&original); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	if original.Status != models.CloudWebhookDeliveryFailed {
		return nil, e.New(e.BadParam, fmt.Errorf("only failed webhook deliveries can be retried"), http.StatusBadRequest)
	}
	webhook := models.CloudWebhook{}
	if err := cloudWebhookVisibleQuery(c).Where("id = ?", form.Id).First(&webhook); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	event := models.CloudEvent{}
	if err := c.DB().Model(&models.CloudEvent{}).
		Where("org_id = ? and id = ?", c.OrgId, original.EventId).
		First(&event); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	retried, err := deliverCloudWebhookWithOptions(c, webhook, event, cloudWebhookDeliveryOptions{
		Attempt:          original.Attempt + 1,
		ParentDeliveryId: original.Id,
		DeliveryMode:     models.CloudWebhookDeliveryModeManual,
		ScheduleRetry:    true,
	})
	if err != nil {
		return nil, err
	}
	resp := resps.CloudWebhookDeliveryResp{
		CloudWebhookDelivery: *retried,
		WebhookName:          webhook.Name,
	}
	return &resp, nil
}

func SearchCloudWebhookDeadLetters(c *ctx.ServiceContext, form *forms.SearchCloudWebhookDeadLetterForm) (interface{}, e.Error) {
	if err := syncCloudWebhookDeadLetters(c); err != nil {
		return nil, err
	}
	query := applyCloudWebhookDeadLetterSearch(
		c.DB().Model(&models.CloudWebhookDeadLetter{}).Where("org_id = ?", c.OrgId),
		form,
	)
	if form.SortField() == "" {
		query = query.Order("created_at desc")
	} else {
		query = form.Order(query)
	}
	deadLetters := make([]models.CloudWebhookDeadLetter, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&deadLetters); err != nil {
		return nil, e.New(e.DBError, err)
	}
	list := make([]resps.CloudWebhookDeadLetterResp, 0, len(deadLetters))
	for _, deadLetter := range deadLetters {
		list = append(list, cloudWebhookDeadLetterResp(c, deadLetter))
	}
	return page.PageResp{
		Total:    p.MustTotal(),
		PageSize: p.Size,
		List:     list,
	}, nil
}

func ReplayCloudWebhookDeadLetter(c *ctx.ServiceContext, form *forms.ReplayCloudWebhookDeadLetterForm) (*resps.CloudWebhookDeliveryResp, e.Error) {
	deadLetter, err := getCloudWebhookDeadLetter(c, form.DeadLetterId)
	if err != nil {
		return nil, err
	}
	if deadLetter.Status != models.CloudWebhookDeadLetterStatusOpen {
		return nil, e.New(e.BadParam, fmt.Errorf("only open webhook dead letters can be replayed"), http.StatusBadRequest)
	}
	webhook := models.CloudWebhook{}
	if err := cloudWebhookBaseQuery(c).Where("id = ? and status = ?", deadLetter.WebhookId, models.CloudWebhookStatusEnabled).First(&webhook); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.BadParam, fmt.Errorf("enable webhook before replaying dead letter"), http.StatusBadRequest)
		}
		return nil, e.New(e.DBError, err)
	}
	event := models.CloudEvent{}
	if err := c.DB().Model(&models.CloudEvent{}).
		Where("org_id = ? and id = ?", c.OrgId, deadLetter.EventId).
		First(&event); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	replayed, replayErr := deliverCloudWebhookWithOptions(c, webhook, event, cloudWebhookDeliveryOptions{
		Attempt:          deadLetter.Attempt + 1,
		ParentDeliveryId: deadLetter.DeliveryId,
		DeliveryMode:     models.CloudWebhookDeliveryModeManual,
		ScheduleRetry:    true,
	})
	if replayErr != nil {
		return nil, replayErr
	}
	if _, err := c.DB().Model(&models.CloudWebhookDeadLetter{}).
		Where("org_id = ? and id = ?", c.OrgId, deadLetter.Id).
		UpdateAttrs(models.Attrs{
			"status":               models.CloudWebhookDeadLetterStatusReplayed,
			"replayed_at":          models.Time(time.Now()),
			"replayed_delivery_id": replayed.Id,
		}); err != nil {
		return nil, e.New(e.DBError, err)
	}
	resp := resps.CloudWebhookDeliveryResp{
		CloudWebhookDelivery: *replayed,
		WebhookName:          webhook.Name,
	}
	return &resp, nil
}

func IgnoreCloudWebhookDeadLetter(c *ctx.ServiceContext, form *forms.IgnoreCloudWebhookDeadLetterForm) (*resps.CloudWebhookDeadLetterResp, e.Error) {
	deadLetter, err := getCloudWebhookDeadLetter(c, form.DeadLetterId)
	if err != nil {
		return nil, err
	}
	if deadLetter.Status != models.CloudWebhookDeadLetterStatusOpen {
		return nil, e.New(e.BadParam, fmt.Errorf("only open webhook dead letters can be ignored"), http.StatusBadRequest)
	}
	if _, err := c.DB().Model(&models.CloudWebhookDeadLetter{}).
		Where("org_id = ? and id = ?", c.OrgId, deadLetter.Id).
		UpdateAttrs(models.Attrs{
			"status":     models.CloudWebhookDeadLetterStatusIgnored,
			"ignored_at": models.Time(time.Now()),
			"note":       strings.TrimSpace(form.Note),
		}); err != nil {
		return nil, e.New(e.DBError, err)
	}
	deadLetter.Status = models.CloudWebhookDeadLetterStatusIgnored
	deadLetter.IgnoredAt = models.Time(time.Now())
	deadLetter.Note = strings.TrimSpace(form.Note)
	resp := cloudWebhookDeadLetterResp(c, deadLetter)
	return &resp, nil
}

func TestCloudWebhook(c *ctx.ServiceContext, form *forms.TestCloudWebhookForm) (*resps.CloudWebhookDeliveryResp, e.Error) {
	webhook := models.CloudWebhook{}
	if err := cloudWebhookVisibleQuery(c).Where("id = ?", form.Id).First(&webhook); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return nil, e.New(e.DBError, err)
	}
	event := models.CloudEvent{
		OrgId:        c.OrgId,
		Source:       models.CloudEventSourceNotification,
		EventType:    "webhook.test",
		Level:        models.CloudEventLevelInfo,
		Status:       "test",
		ResourceType: "webhook",
		ResourceId:   webhook.Id.String(),
		ResourceName: webhook.Name,
		Title:        "Webhook 测试投递",
		Message:      fmt.Sprintf("测试 Webhook %s 到 %s 的连通性", webhook.Name, webhook.TargetUrl),
		Payload: models.ResAttrs{
			"webhookId": webhook.Id.String(),
			"targetUrl": webhook.TargetUrl,
		},
		OccurredAt: models.Time(time.Now()),
	}
	event.Id = models.NewId("evt")
	delivery, err := deliverCloudWebhookWithOptions(c, webhook, event, cloudWebhookDeliveryOptions{
		Attempt:       1,
		DeliveryMode:  models.CloudWebhookDeliveryModeTest,
		ScheduleRetry: false,
	})
	if err != nil {
		return nil, err
	}
	resp := resps.CloudWebhookDeliveryResp{
		CloudWebhookDelivery: *delivery,
		WebhookName:          webhook.Name,
	}
	return &resp, nil
}

func CloudWebhookQueueSummary(c *ctx.ServiceContext) (*resps.CloudWebhookQueueSummaryResp, e.Error) {
	if err := syncCloudWebhookDeadLetters(c); err != nil {
		return nil, err
	}
	if _, err := cleanupCloudWebhookDeliveryQueues(c); err != nil {
		return nil, err
	}
	if err := syncCloudWebhookDeliveryQueues(c); err != nil {
		return nil, err
	}
	now := models.Time(time.Now())
	minRetryAt := cloudWebhookMinRetryAt()
	base := func() *db.Session {
		return c.DB().Model(&models.CloudWebhookDelivery{}).Where("org_id = ?", c.OrgId)
	}
	queueBase := func() *db.Session {
		return c.DB().Model(&models.CloudWebhookDeliveryQueue{}).
			Where("org_id = ? and status = ?", c.OrgId, models.CloudWebhookDeliveryQueueStatusQueued)
	}
	consumedQueueBase := func() *db.Session {
		return c.DB().Model(&models.CloudWebhookDeliveryQueue{}).
			Where(
				"org_id = ? and status in (?) and consumed_at > ?",
				c.OrgId,
				[]string{models.CloudWebhookDeliveryQueueStatusDone, models.CloudWebhookDeliveryQueueStatusSkipped},
				minRetryAt,
			)
	}
	count := func(query *db.Session) (int64, e.Error) {
		value, err := cloudOverviewCount(query)
		if err != nil {
			return 0, e.New(e.DBError, err)
		}
		return value, nil
	}
	resp := &resps.CloudWebhookQueueSummaryResp{}
	var err e.Error
	if resp.TotalDeliveries, err = count(base()); err != nil {
		return nil, err
	}
	if resp.Pending, err = count(base().Where("status = ?", models.CloudWebhookDeliveryPending)); err != nil {
		return nil, err
	}
	if resp.Success, err = count(base().Where("status = ?", models.CloudWebhookDeliverySuccess)); err != nil {
		return nil, err
	}
	if resp.Failed, err = count(base().Where("status = ?", models.CloudWebhookDeliveryFailed)); err != nil {
		return nil, err
	}
	resp.SuccessRate = cloudWebhookPercent(resp.Success, resp.Success+resp.Failed)
	if resp.Queued, err = count(queueBase().Where("next_run_at > ?", minRetryAt)); err != nil {
		return nil, err
	}
	if resp.Due, err = count(queueBase().Where("next_run_at > ? and next_run_at <= ?", minRetryAt, now)); err != nil {
		return nil, err
	}
	if resp.Future, err = count(queueBase().Where("next_run_at > ?", now)); err != nil {
		return nil, err
	}
	if resp.DeadLetter, err = count(c.DB().Model(&models.CloudWebhookDeadLetter{}).
		Where("org_id = ? and status = ?", c.OrgId, models.CloudWebhookDeadLetterStatusOpen)); err != nil {
		return nil, err
	}
	if resp.Retried, err = count(base().Where("status = ? and retried_at > ?", models.CloudWebhookDeliveryFailed, minRetryAt)); err != nil {
		return nil, err
	}
	if resp.Initial, err = count(base().Where("delivery_mode = ?", models.CloudWebhookDeliveryModeInitial)); err != nil {
		return nil, err
	}
	if resp.Manual, err = count(base().Where("delivery_mode = ?", models.CloudWebhookDeliveryModeManual)); err != nil {
		return nil, err
	}
	if resp.Auto, err = count(base().Where("delivery_mode = ?", models.CloudWebhookDeliveryModeAuto)); err != nil {
		return nil, err
	}
	if resp.Test, err = count(base().Where("delivery_mode = ?", models.CloudWebhookDeliveryModeTest)); err != nil {
		return nil, err
	}
	if resp.QueueConsumed, err = count(consumedQueueBase()); err != nil {
		return nil, err
	}
	if resp.QueueSuccess, err = count(consumedQueueBase().Where("status = ? and result_delivery_id <> '' and error_message = ''", models.CloudWebhookDeliveryQueueStatusDone)); err != nil {
		return nil, err
	}
	if resp.QueueFailed, err = count(consumedQueueBase().Where("status = ? and error_message <> ''", models.CloudWebhookDeliveryQueueStatusDone)); err != nil {
		return nil, err
	}
	if resp.QueueSkipped, err = count(consumedQueueBase().Where("status = ?", models.CloudWebhookDeliveryQueueStatusSkipped)); err != nil {
		return nil, err
	}
	resp.QueueSuccessRate = cloudWebhookPercent(resp.QueueSuccess, resp.QueueConsumed)
	oldest := models.CloudWebhookDeliveryQueue{}
	if dbErr := queueBase().
		Where("next_run_at > ?", minRetryAt).
		Order("next_run_at asc").
		First(&oldest); dbErr == nil {
		resp.OldestQueuedAt = oldest.NextRunAt
	} else if !e.IsRecordNotFound(dbErr) {
		return nil, e.New(e.DBError, dbErr)
	}
	lastConsumed := models.CloudWebhookDeliveryQueue{}
	if dbErr := consumedQueueBase().
		Order("consumed_at desc").
		First(&lastConsumed); dbErr == nil {
		resp.LastConsumedAt = lastConsumed.ConsumedAt
	} else if !e.IsRecordNotFound(dbErr) {
		return nil, e.New(e.DBError, dbErr)
	}
	last := models.CloudWebhookDelivery{}
	if dbErr := base().
		Where("delivered_at is not null and delivered_at > ?", minRetryAt).
		Order("delivered_at desc").
		First(&last); dbErr == nil {
		resp.LastDeliveredAt = last.DeliveredAt
	} else if !e.IsRecordNotFound(dbErr) {
		return nil, e.New(e.DBError, dbErr)
	}
	shardTotal := cloudWebhookRetryShardTotal()
	resp.ShardTotal = shardTotal
	resp.Shards = make([]resps.CloudWebhookQueueShardSummaryResp, 0, shardTotal)
	for shardIndex := 0; shardIndex < shardTotal; shardIndex++ {
		retryOptions := cloudWebhookRetryOptions{
			ShardIndex: shardIndex,
			ShardTotal: shardTotal,
		}
		shardBase := func() *db.Session {
			return applyCloudWebhookDeliveryQueueShard(queueBase(), retryOptions)
		}
		shardConsumedBase := func() *db.Session {
			return applyCloudWebhookDeliveryQueueShard(consumedQueueBase(), retryOptions)
		}
		shard := resps.CloudWebhookQueueShardSummaryResp{
			ShardIndex: shardIndex,
			ShardTotal: shardTotal,
		}
		if shard.Queued, err = count(shardBase().Where("next_run_at > ?", minRetryAt)); err != nil {
			return nil, err
		}
		if shard.Due, err = count(shardBase().Where("next_run_at > ? and next_run_at <= ?", minRetryAt, now)); err != nil {
			return nil, err
		}
		if shard.Future, err = count(shardBase().Where("next_run_at > ?", now)); err != nil {
			return nil, err
		}
		if shard.Consumed, err = count(shardConsumedBase()); err != nil {
			return nil, err
		}
		if shard.Success, err = count(shardConsumedBase().Where("status = ? and result_delivery_id <> '' and error_message = ''", models.CloudWebhookDeliveryQueueStatusDone)); err != nil {
			return nil, err
		}
		if shard.Failed, err = count(shardConsumedBase().Where("status = ? and error_message <> ''", models.CloudWebhookDeliveryQueueStatusDone)); err != nil {
			return nil, err
		}
		if shard.Skipped, err = count(shardConsumedBase().Where("status = ?", models.CloudWebhookDeliveryQueueStatusSkipped)); err != nil {
			return nil, err
		}
		shard.SuccessRate = cloudWebhookPercent(shard.Success, shard.Consumed)
		shardOldest := models.CloudWebhookDeliveryQueue{}
		if dbErr := shardBase().
			Where("next_run_at > ?", minRetryAt).
			Order("next_run_at asc").
			First(&shardOldest); dbErr == nil {
			shard.OldestQueuedAt = shardOldest.NextRunAt
		} else if !e.IsRecordNotFound(dbErr) {
			return nil, e.New(e.DBError, dbErr)
		}
		shardLastConsumed := models.CloudWebhookDeliveryQueue{}
		if dbErr := shardConsumedBase().
			Order("consumed_at desc").
			First(&shardLastConsumed); dbErr == nil {
			shard.LastConsumedAt = shardLastConsumed.ConsumedAt
		} else if !e.IsRecordNotFound(dbErr) {
			return nil, e.New(e.DBError, dbErr)
		}
		resp.Shards = append(resp.Shards, shard)
	}
	return resp, nil
}

func RetryDueCloudWebhookDeliveries(c *ctx.ServiceContext) (models.ResAttrs, e.Error) {
	return RetryDueCloudWebhookDeliveryShards(c, cloudWebhookRetryShardTotal())
}

func RetryDueCloudWebhookDeliveriesWithForm(c *ctx.ServiceContext, form *forms.RetryDueCloudWebhookDeliveriesForm) (models.ResAttrs, e.Error) {
	retryOptions, err := cloudWebhookRetryOptionsFromForm(form)
	if err != nil {
		return nil, err
	}
	if retryOptions.sharded() {
		return retryDueCloudWebhookDeliveriesWithOptions(c, retryOptions)
	}
	return RetryDueCloudWebhookDeliveryShards(c, cloudWebhookRetryShardTotal())
}

func RetryDueCloudWebhookDeliveryShards(c *ctx.ServiceContext, shardTotal int) (models.ResAttrs, e.Error) {
	if shardTotal <= 1 {
		return retryDueCloudWebhookDeliveriesWithOptions(c, cloudWebhookRetryOptions{})
	}
	startedAt := time.Now()
	cleanedQueues, maintenanceErr := prepareCloudWebhookDeliveryQueues(c)
	if maintenanceErr != nil {
		return nil, maintenanceErr
	}
	result := models.ResAttrs{
		"total":            0,
		"success":          0,
		"failed":           0,
		"skipped":          0,
		"locked":           true,
		"lockSkipped":      false,
		"lockSkippedCount": 0,
		"queueCleaned":     cleanedQueues,
		"shards":           shardTotal,
	}
	defer cloudWebhookRetryFinalizeResult(result, startedAt)
	type shardExecResult struct {
		shardIndex int
		result     models.ResAttrs
		err        e.Error
	}
	shardResults := make([]models.ResAttrs, shardTotal)
	resultCh := make(chan shardExecResult, shardTotal)
	var wg sync.WaitGroup
	for shardIndex := 0; shardIndex < shardTotal; shardIndex++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			shardCtx := cloudWebhookRetryChildContext(c)
			shardResult, err := retryDueCloudWebhookDeliveriesWithOptions(shardCtx, cloudWebhookRetryOptions{
				ShardIndex:           index,
				ShardTotal:           shardTotal,
				SkipQueueMaintenance: true,
			})
			resultCh <- shardExecResult{shardIndex: index, result: shardResult, err: err}
		}(shardIndex)
	}
	wg.Wait()
	close(resultCh)
	for item := range resultCh {
		if item.err != nil {
			return nil, item.err
		}
		shardResult := item.result
		shardResults[item.shardIndex] = shardResult
		for _, key := range []string{"total", "success", "failed", "skipped", "queueCleaned"} {
			result[key] = cloudWebhookRetryResultCount(result, key) + cloudWebhookRetryResultCount(shardResult, key)
		}
		if skipped, _ := shardResult["lockSkipped"].(bool); skipped {
			result["locked"] = false
			result["lockSkipped"] = true
			result["lockSkippedCount"] = cloudWebhookRetryResultCount(result, "lockSkippedCount") + 1
		}
		if lastConsumedAt := cloudWebhookRetryResultTime(shardResult, "lastConsumedAt"); !time.Time(lastConsumedAt).IsZero() {
			cloudWebhookRetrySetLastConsumedAt(result, lastConsumedAt)
		}
	}
	result["shardResults"] = shardResults
	return result, nil
}

func retryDueCloudWebhookDeliveriesWithOptions(c *ctx.ServiceContext, retryOptions cloudWebhookRetryOptions) (models.ResAttrs, e.Error) {
	startedAt := time.Now()
	lockName := cloudWebhookRetryLockName(c.OrgId, retryOptions)
	locked, releaseLock, lockErr := cloudWebhookAcquireMysqlLock(lockName)
	if lockErr != nil {
		return nil, e.New(e.DBError, lockErr)
	}
	if !locked {
		result := models.ResAttrs{
			"total":        0,
			"success":      0,
			"failed":       0,
			"skipped":      0,
			"locked":       false,
			"lockSkipped":  true,
			"queueCleaned": int64(0),
		}
		cloudWebhookRetryApplyShardAttrs(result, retryOptions)
		cloudWebhookRetryFinalizeResult(result, startedAt)
		return result, nil
	}
	defer releaseLock()

	cleanedQueues := int64(0)
	if !retryOptions.SkipQueueMaintenance {
		var maintenanceErr e.Error
		cleanedQueues, maintenanceErr = prepareCloudWebhookDeliveryQueues(c)
		if maintenanceErr != nil {
			return nil, maintenanceErr
		}
	}
	now := models.Time(time.Now())
	minRetryAt := cloudWebhookMinRetryAt()
	queues := make([]models.CloudWebhookDeliveryQueue, 0)
	query := c.DB().Model(&models.CloudWebhookDeliveryQueue{}).
		Where("org_id = ? and status = ? and next_run_at > ? and next_run_at <= ?",
			c.OrgId, models.CloudWebhookDeliveryQueueStatusQueued, minRetryAt, now)
	query = applyCloudWebhookDeliveryQueueShard(query, retryOptions)
	if err := query.
		Order("next_run_at asc").
		Limit(20).
		Find(&queues); err != nil {
		return nil, e.New(e.DBError, err)
	}
	result := models.ResAttrs{
		"total":        len(queues),
		"success":      0,
		"failed":       0,
		"skipped":      0,
		"locked":       true,
		"lockSkipped":  false,
		"queueCleaned": cleanedQueues,
	}
	cloudWebhookRetryApplyShardAttrs(result, retryOptions)
	defer cloudWebhookRetryFinalizeResult(result, startedAt)
	for _, queue := range queues {
		if updated, err := c.DB().Model(&models.CloudWebhookDeliveryQueue{}).
			Where("id = ? and org_id = ? and status = ?", queue.Id, c.OrgId, models.CloudWebhookDeliveryQueueStatusQueued).
			UpdateAttrs(models.Attrs{
				"status":    models.CloudWebhookDeliveryQueueStatusProcessing,
				"locked_at": now,
				"locked_by": lockName,
			}); err != nil {
			return nil, e.New(e.DBError, err)
		} else if updated == 0 {
			result["skipped"] = result["skipped"].(int) + 1
			continue
		}
		delivery := models.CloudWebhookDelivery{}
		if err := c.DB().Model(&models.CloudWebhookDelivery{}).
			Where("org_id = ? and id = ? and status = ?", c.OrgId, queue.DeliveryId, models.CloudWebhookDeliveryFailed).
			First(&delivery); err != nil {
			cloudWebhookRetrySetLastConsumedAt(result, cloudWebhookFinishDeliveryQueue(c, queue, models.CloudWebhookDeliveryQueueStatusSkipped, "", "delivery not found or not failed"))
			result["skipped"] = result["skipped"].(int) + 1
			continue
		}
		if _, err := c.DB().Model(&models.CloudWebhookDelivery{}).
			Where("id = ? and org_id = ?", delivery.Id, c.OrgId).
			UpdateAttrs(models.Attrs{
				"next_retry_at": models.Time{},
				"retried_at":    now,
			}); err != nil {
			return nil, e.New(e.DBError, err)
		}
		webhook := models.CloudWebhook{}
		if err := cloudWebhookBaseQuery(c).Where("id = ? and status = ?", delivery.WebhookId, models.CloudWebhookStatusEnabled).First(&webhook); err != nil {
			cloudWebhookRetrySetLastConsumedAt(result, cloudWebhookFinishDeliveryQueue(c, queue, models.CloudWebhookDeliveryQueueStatusSkipped, "", "webhook not enabled or not found"))
			result["skipped"] = result["skipped"].(int) + 1
			continue
		}
		event := models.CloudEvent{}
		if err := c.DB().Model(&models.CloudEvent{}).
			Where("org_id = ? and id = ?", c.OrgId, delivery.EventId).
			First(&event); err != nil {
			cloudWebhookRetrySetLastConsumedAt(result, cloudWebhookFinishDeliveryQueue(c, queue, models.CloudWebhookDeliveryQueueStatusSkipped, "", "event not found"))
			result["skipped"] = result["skipped"].(int) + 1
			continue
		}
		retried, err := deliverCloudWebhookWithOptions(c, webhook, event, cloudWebhookDeliveryOptions{
			Attempt:          delivery.Attempt + 1,
			ParentDeliveryId: delivery.Id,
			DeliveryMode:     models.CloudWebhookDeliveryModeAuto,
			ScheduleRetry:    true,
		})
		if err != nil || retried.Status != models.CloudWebhookDeliverySuccess {
			resultDeliveryId := models.Id("")
			errorMessage := ""
			if retried != nil {
				resultDeliveryId = retried.Id
				errorMessage = retried.ErrorMessage
			}
			if err != nil {
				errorMessage = err.Error()
			}
			cloudWebhookRetrySetLastConsumedAt(result, cloudWebhookFinishDeliveryQueue(c, queue, models.CloudWebhookDeliveryQueueStatusDone, resultDeliveryId, errorMessage))
			result["failed"] = result["failed"].(int) + 1
			continue
		}
		cloudWebhookRetrySetLastConsumedAt(result, cloudWebhookFinishDeliveryQueue(c, queue, models.CloudWebhookDeliveryQueueStatusDone, retried.Id, ""))
		result["success"] = result["success"].(int) + 1
	}
	return result, nil
}

func RetryDueCloudWebhookDeliveriesForAllOrgs() (models.ResAttrs, e.Error) {
	now := models.Time(time.Now())
	minRetryAt := cloudWebhookMinRetryAt()
	orgIds := make([]models.Id, 0)
	queueOrgIds := make([]models.Id, 0)
	if err := db.Get().Model(&models.CloudWebhookDeliveryQueue{}).
		Where("status = ? and next_run_at > ? and next_run_at <= ?",
			models.CloudWebhookDeliveryQueueStatusQueued, minRetryAt, now).
		Group("org_id").
		Pluck("org_id", &queueOrgIds); err != nil {
		return nil, e.New(e.DBError, err)
	}
	legacyOrgIds := make([]models.Id, 0)
	if err := db.Get().Model(&models.CloudWebhookDelivery{}).
		Where("status = ? and next_retry_at > ? and next_retry_at <= ?",
			models.CloudWebhookDeliveryFailed, minRetryAt, now).
		Group("org_id").
		Pluck("org_id", &legacyOrgIds); err != nil {
		return nil, e.New(e.DBError, err)
	}
	seenOrgIds := map[models.Id]bool{}
	for _, orgId := range append(queueOrgIds, legacyOrgIds...) {
		if orgId == "" || seenOrgIds[orgId] {
			continue
		}
		seenOrgIds[orgId] = true
		orgIds = append(orgIds, orgId)
	}
	result := models.ResAttrs{
		"orgs":             len(orgIds),
		"total":            0,
		"success":          0,
		"failed":           0,
		"skipped":          0,
		"queueCleaned":     0,
		"shards":           cloudWebhookRetryShardTotal(),
		"lockSkippedCount": 0,
	}
	for _, orgId := range orgIds {
		workerCtx := &ctx.ServiceContext{
			UserId:   consts.SysUserId,
			OrgId:    orgId,
			Email:    consts.DefaultSysEmail,
			Username: consts.DefaultSysName,
		}
		orgResult, err := RetryDueCloudWebhookDeliveries(workerCtx)
		if err != nil {
			return nil, err
		}
		for _, key := range []string{"total", "success", "failed", "skipped", "queueCleaned", "lockSkippedCount"} {
			result[key] = cloudWebhookRetryResultCount(result, key) + cloudWebhookRetryResultCount(orgResult, key)
		}
	}
	return result, nil
}

func CleanupExpiredCloudWebhookDeliveryQueuesForAllOrgs() (models.ResAttrs, e.Error) {
	minRetryAt := cloudWebhookMinRetryAt()
	cleanupCutoff := models.Time(time.Now().Add(-cloudWebhookDeliveryQueueRetention))
	orgIds := make([]models.Id, 0)
	if err := db.Get().Model(&models.CloudWebhookDeliveryQueue{}).
		Where("status in (?) and consumed_at > ? and consumed_at <= ?",
			[]string{models.CloudWebhookDeliveryQueueStatusDone, models.CloudWebhookDeliveryQueueStatusSkipped},
			minRetryAt,
			cleanupCutoff,
		).
		Group("org_id").
		Pluck("org_id", &orgIds); err != nil {
		return nil, e.New(e.DBError, err)
	}
	result := models.ResAttrs{
		"orgs":         len(orgIds),
		"queueCleaned": 0,
	}
	for _, orgId := range orgIds {
		workerCtx := &ctx.ServiceContext{
			UserId:   consts.SysUserId,
			OrgId:    orgId,
			Email:    consts.DefaultSysEmail,
			Username: consts.DefaultSysName,
		}
		cleanedQueues, err := prepareCloudWebhookDeliveryQueues(workerCtx)
		if err != nil {
			return nil, err
		}
		result["queueCleaned"] = cloudWebhookRetryResultCount(result, "queueCleaned") + int(cleanedQueues)
	}
	return result, nil
}

func prepareCloudWebhookDeliveryQueues(c *ctx.ServiceContext) (int64, e.Error) {
	lockName := cloudWebhookRetryMaintenanceLockName(c.OrgId)
	locked, releaseLock, lockErr := cloudWebhookAcquireMysqlLock(lockName)
	if lockErr != nil {
		return 0, e.New(e.DBError, lockErr)
	}
	if !locked {
		return 0, nil
	}
	defer releaseLock()
	if err := syncCloudWebhookDeliveryQueues(c); err != nil {
		return 0, err
	}
	cleanedQueues, cleanupErr := cleanupCloudWebhookDeliveryQueues(c)
	if cleanupErr != nil {
		return 0, cleanupErr
	}
	return cleanedQueues, nil
}

type cloudWebhookRetryRequestContext struct {
	sc *ctx.ServiceContext
}

func (r *cloudWebhookRetryRequestContext) BindService(sc *ctx.ServiceContext) {
	r.sc = sc
}

func (r *cloudWebhookRetryRequestContext) Service() *ctx.ServiceContext {
	return r.sc
}

func (r *cloudWebhookRetryRequestContext) Logger() logs.Logger {
	if r.sc == nil {
		return logs.Get()
	}
	return r.sc.Logger()
}

func cloudWebhookRetryChildContext(c *ctx.ServiceContext) *ctx.ServiceContext {
	rc := &cloudWebhookRetryRequestContext{}
	child := ctx.NewServiceContext(rc)
	if c == nil {
		return child
	}
	child.UserId = c.UserId
	child.OrgId = c.OrgId
	child.ProjectId = c.ProjectId
	child.Email = c.Email
	child.Username = c.Username
	child.IsSuperAdmin = c.IsSuperAdmin
	child.UserIpAddr = c.UserIpAddr
	return child
}

func syncCloudWebhookDeliveryQueues(c *ctx.ServiceContext) e.Error {
	minRetryAt := cloudWebhookMinRetryAt()
	deliveries := make([]models.CloudWebhookDelivery, 0)
	if err := c.DB().Model(&models.CloudWebhookDelivery{}).
		Where("org_id = ? and status = ? and next_retry_at > ?",
			c.OrgId, models.CloudWebhookDeliveryFailed, minRetryAt).
		Order("next_retry_at asc").
		Limit(200).
		Find(&deliveries); err != nil {
		return e.New(e.DBError, err)
	}
	for _, delivery := range deliveries {
		if err := ensureCloudWebhookDeliveryQueue(c, delivery, delivery.NextRetryAt); err != nil {
			return err
		}
	}
	return nil
}

func ensureCloudWebhookDeliveryQueue(c *ctx.ServiceContext, delivery models.CloudWebhookDelivery, nextRunAt models.Time) e.Error {
	if c == nil || delivery.Id == "" || delivery.DeliveryMode == models.CloudWebhookDeliveryModeTest || time.Time(nextRunAt).IsZero() {
		return nil
	}
	if err := cloudwebhooksrv.EnsureDeliveryQueue(c.DB(), delivery, nextRunAt); err != nil {
		return e.New(e.DBError, err)
	}
	return nil
}

func cloudWebhookFinishDeliveryQueue(c *ctx.ServiceContext, queue models.CloudWebhookDeliveryQueue, status string, resultDeliveryId models.Id, errorMessage string) models.Time {
	if c == nil || queue.Id == "" {
		return models.Time{}
	}
	consumedAt := models.Time(time.Now())
	attrs := models.Attrs{
		"status":             status,
		"consumed_at":        consumedAt,
		"result_delivery_id": resultDeliveryId,
		"error_message":      strings.TrimSpace(errorMessage),
	}
	if _, err := c.DB().Model(&models.CloudWebhookDeliveryQueue{}).
		Where("org_id = ? and id = ?", queue.OrgId, queue.Id).
		UpdateAttrs(attrs); err != nil {
		c.Logger().Warnf("finish cloud webhook delivery queue failed: %v", err)
	}
	return consumedAt
}

func cleanupCloudWebhookDeliveryQueues(c *ctx.ServiceContext) (int64, e.Error) {
	if c == nil || c.OrgId == "" {
		return 0, nil
	}
	cutoff := models.Time(time.Now().Add(-cloudWebhookDeliveryQueueRetention))
	minTime := cloudWebhookMinRetryAt()
	rows, err := c.DB().
		Where("org_id = ? and status in (?) and consumed_at > ? and consumed_at <= ?",
			c.OrgId,
			[]string{models.CloudWebhookDeliveryQueueStatusDone, models.CloudWebhookDeliveryQueueStatusSkipped},
			minTime,
			cutoff,
		).
		Delete(&models.CloudWebhookDeliveryQueue{})
	if err != nil {
		return rows, e.New(e.DBError, err)
	}
	return rows, nil
}

func StartCloudWebhookRetryWorker(serviceId string) {
	logger := logs.Get().WithField("worker", "cloudWebhookRetry").WithField("serviceId", serviceId)
	ticker := time.NewTicker(cloudWebhookRetryWorkerInterval)
	defer ticker.Stop()
	for {
		if result, err := RetryDueCloudWebhookDeliveriesForAllOrgs(); err != nil {
			logger.Warnf("retry due cloud webhooks failed: %v", err)
		} else if cloudWebhookRetryResultCount(result, "total") > 0 ||
			cloudWebhookRetryResultCount(result, "queueCleaned") > 0 ||
			cloudWebhookRetryResultCount(result, "lockSkippedCount") > 0 {
			logger.Infof("retry due cloud webhooks result: %+v", result)
		}
		<-ticker.C
	}
}

func StartCloudWebhookQueueMaintenanceWorker(serviceId string) {
	logger := logs.Get().WithField("worker", "cloudWebhookQueueMaintenance").WithField("serviceId", serviceId)
	ticker := time.NewTicker(cloudWebhookQueueMaintenanceWorkerInterval)
	defer ticker.Stop()
	for {
		if result, err := CleanupExpiredCloudWebhookDeliveryQueuesForAllOrgs(); err != nil {
			logger.Warnf("cleanup expired cloud webhook delivery queues failed: %v", err)
		} else if cloudWebhookRetryResultCount(result, "queueCleaned") > 0 {
			logger.Infof("cleanup expired cloud webhook delivery queues result: %+v", result)
		}
		<-ticker.C
	}
}

func cloudWebhookRetryResultCount(result models.ResAttrs, key string) int {
	value, ok := result[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func cloudWebhookRetryResultTime(result models.ResAttrs, key string) models.Time {
	value, ok := result[key]
	if !ok || value == nil {
		return models.Time{}
	}
	switch v := value.(type) {
	case models.Time:
		return v
	case time.Time:
		return models.Time(v)
	default:
		return models.Time{}
	}
}

func cloudWebhookRetrySetLastConsumedAt(result models.ResAttrs, consumedAt models.Time) {
	if result == nil || time.Time(consumedAt).IsZero() {
		return
	}
	current := cloudWebhookRetryResultTime(result, "lastConsumedAt")
	if time.Time(current).IsZero() || time.Time(consumedAt).After(time.Time(current)) {
		result["lastConsumedAt"] = consumedAt
	}
}

func cloudWebhookRetryFinalizeResult(result models.ResAttrs, startedAt time.Time) {
	if result == nil {
		return
	}
	finishedAt := time.Now()
	result["startedAt"] = models.Time(startedAt)
	result["finishedAt"] = models.Time(finishedAt)
	result["durationMs"] = finishedAt.Sub(startedAt).Milliseconds()
	result["successRate"] = cloudWebhookPercent(int64(cloudWebhookRetryResultCount(result, "success")), int64(cloudWebhookRetryResultCount(result, "total")))
}

func cloudWebhookPercent(success, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round(float64(success)*10000/float64(total)) / 100
}

func cloudWebhookMinRetryAt() models.Time {
	return models.Time(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC))
}

func cloudWebhookRetryOptionsFromForm(form *forms.RetryDueCloudWebhookDeliveriesForm) (cloudWebhookRetryOptions, e.Error) {
	if form == nil {
		return cloudWebhookRetryOptions{}, nil
	}
	if form.ShardIndex < 0 || form.ShardTotal < 0 {
		return cloudWebhookRetryOptions{}, e.New(e.BadParam, fmt.Errorf("shardIndex and shardTotal must be non-negative"), http.StatusBadRequest)
	}
	if form.ShardTotal == 0 {
		if form.ShardIndex != 0 {
			return cloudWebhookRetryOptions{}, e.New(e.BadParam, fmt.Errorf("shardTotal is required when shardIndex is set"), http.StatusBadRequest)
		}
		return cloudWebhookRetryOptions{}, nil
	}
	if form.ShardTotal > cloudWebhookRetryMaxShardTotal {
		return cloudWebhookRetryOptions{}, e.New(e.BadParam, fmt.Errorf("shardTotal cannot exceed %d", cloudWebhookRetryMaxShardTotal), http.StatusBadRequest)
	}
	if form.ShardIndex >= form.ShardTotal {
		return cloudWebhookRetryOptions{}, e.New(e.BadParam, fmt.Errorf("shardIndex must be less than shardTotal"), http.StatusBadRequest)
	}
	return cloudWebhookRetryOptions{
		ShardIndex: form.ShardIndex,
		ShardTotal: form.ShardTotal,
	}, nil
}

func cloudWebhookRetryShardTotal() int {
	raw := strings.TrimSpace(os.Getenv("CLOUDIAC_WEBHOOK_RETRY_SHARDS"))
	if raw == "" {
		return cloudWebhookRetryDefaultShardTotal
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return cloudWebhookRetryDefaultShardTotal
	}
	if value > cloudWebhookRetryMaxShardTotal {
		return cloudWebhookRetryMaxShardTotal
	}
	return value
}

func cloudWebhookRetryApplyShardAttrs(result models.ResAttrs, retryOptions cloudWebhookRetryOptions) {
	if retryOptions.sharded() {
		result["shardIndex"] = retryOptions.ShardIndex
		result["shardTotal"] = retryOptions.ShardTotal
	}
}

func applyCloudWebhookDeliveryQueueShard(query *db.Session, retryOptions cloudWebhookRetryOptions) *db.Session {
	if !retryOptions.sharded() {
		return query
	}
	return query.Where("cast(conv(substr(md5(id), 1, 8), 16, 10) as unsigned) % ? = ?", retryOptions.ShardTotal, retryOptions.ShardIndex)
}

func cloudWebhookRetryMaintenanceLockName(orgId models.Id) string {
	if orgId == "" {
		return "cloudiac:webhook_retry:unknown:maintenance"
	}
	return fmt.Sprintf("cloudiac:webhook_retry:%s:maintenance", orgId.String())
}

func cloudWebhookRetryLockName(orgId models.Id, retryOptions cloudWebhookRetryOptions) string {
	if orgId == "" {
		return "cloudiac:webhook_retry:unknown"
	}
	if retryOptions.sharded() {
		return fmt.Sprintf("cloudiac:webhook_retry:%s:shard:%d:%d", orgId.String(), retryOptions.ShardTotal, retryOptions.ShardIndex)
	}
	return fmt.Sprintf("cloudiac:webhook_retry:%s", orgId.String())
}

func cloudWebhookAcquireMysqlLock(lockName string) (bool, func(), error) {
	lockTx := db.Get().Begin()
	locked := 0
	if err := lockTx.Raw("select get_lock(?, 0)", lockName).Scan(&locked); err != nil {
		_ = lockTx.Rollback()
		return false, nil, err
	}
	if locked != 1 {
		_ = lockTx.Rollback()
		return false, func() {}, nil
	}
	released := false
	release := func() {
		if released {
			return
		}
		released = true
		releaseResult := 0
		if err := lockTx.Raw("select release_lock(?)", lockName).Scan(&releaseResult); err != nil {
			logs.Get().WithField("lock", lockName).Warnf("release cloud webhook retry lock failed: %v", err)
		}
		if err := lockTx.Commit(); err != nil {
			logs.Get().WithField("lock", lockName).Warnf("commit cloud webhook retry lock transaction failed: %v", err)
		}
	}
	return true, release, nil
}

func dispatchCloudEventWebhooksBestEffort(c *ctx.ServiceContext, event *models.CloudEvent) {
	if c == nil || event == nil || event.OrgId == "" {
		return
	}
	if _, err := RetryDueCloudWebhookDeliveries(c); err != nil {
		c.Logger().Warnf("retry due cloud webhooks failed: %v", err)
	}
	webhooks := make([]models.CloudWebhook, 0)
	if err := c.DB().Model(&models.CloudWebhook{}).
		Where("org_id = ? and status = ?", event.OrgId, models.CloudWebhookStatusEnabled).
		Where("description not like ?", notificationWebhookMirrorDescriptionPrefix+"%").
		Find(&webhooks); err != nil {
		c.Logger().Warnf("search cloud webhooks failed: %v", err)
		return
	}
	for _, webhook := range webhooks {
		if !cloudWebhookMatchesEvent(webhook, *event) {
			continue
		}
		if _, err := deliverCloudWebhook(c, webhook, *event); err != nil {
			c.Logger().Warnf("deliver cloud webhook failed: %v", err)
		}
	}
}

func applyCloudWebhookSearch(query *db.Session, form *forms.SearchCloudWebhookForm) *db.Session {
	if form.Q != "" {
		q := "%" + strings.TrimSpace(form.Q) + "%"
		query = query.Where("name like ? or description like ? or target_url like ?", q, q, q)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	if form.Source != "" {
		q := "%" + form.Source + "%"
		query = query.Where("sources like ?", q)
	}
	return query
}

func applyCloudWebhookDeadLetterSearch(query *db.Session, form *forms.SearchCloudWebhookDeadLetterForm) *db.Session {
	if form.Q != "" {
		q := "%" + strings.TrimSpace(form.Q) + "%"
		query = query.Where("id like ? or event_type like ? or target_url like ? or error_message like ? or reason like ?", q, q, q, q, q)
	}
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	if form.WebhookId != "" {
		query = query.Where("webhook_id = ?", form.WebhookId)
	}
	if form.EventType != "" {
		query = query.Where("event_type = ?", form.EventType)
	}
	return query
}

func cloudWebhookBaseQuery(c *ctx.ServiceContext) *db.Session {
	return c.DB().Model(&models.CloudWebhook{}).
		Where("org_id = ?", c.OrgId)
}

func cloudWebhookVisibleQuery(c *ctx.ServiceContext) *db.Session {
	return cloudWebhookBaseQuery(c).
		Where("description not like ?", notificationWebhookMirrorDescriptionPrefix+"%")
}

func getCloudWebhookDeadLetter(c *ctx.ServiceContext, deadLetterId models.Id) (models.CloudWebhookDeadLetter, e.Error) {
	deadLetter := models.CloudWebhookDeadLetter{}
	if err := c.DB().Model(&models.CloudWebhookDeadLetter{}).
		Where("org_id = ? and id = ?", c.OrgId, deadLetterId).
		First(&deadLetter); err != nil {
		if e.IsRecordNotFound(err) {
			return deadLetter, e.New(e.ObjectNotExistsOrNoPerm, http.StatusNotFound)
		}
		return deadLetter, e.New(e.DBError, err)
	}
	return deadLetter, nil
}

func cloudWebhookFromCreateForm(c *ctx.ServiceContext, form *forms.CreateCloudWebhookForm) (*models.CloudWebhook, e.Error) {
	webhook := &models.CloudWebhook{
		OrgId:              c.OrgId,
		Name:               strings.TrimSpace(form.Name),
		Description:        strings.TrimSpace(form.Description),
		TargetUrl:          strings.TrimSpace(form.TargetUrl),
		Secret:             strings.TrimSpace(form.Secret),
		SecretVersion:      1,
		EventTypes:         models.StrSlice(normalizeStringSlice(form.EventTypes)),
		Sources:            models.StrSlice(normalizeStringSlice(form.Sources)),
		Status:             firstNonEmpty(form.Status, models.CloudWebhookStatusEnabled),
		TimeoutSeconds:     form.TimeoutSeconds,
		MaxRetries:         form.MaxRetries,
		RetryInterval:      form.RetryInterval,
		RetryJitterPercent: form.RetryJitterPercent,
		MaxRetryDuration:   form.MaxRetryDuration,
	}
	if webhook.TimeoutSeconds <= 0 {
		webhook.TimeoutSeconds = 5
	}
	if webhook.TimeoutSeconds > 30 {
		webhook.TimeoutSeconds = 30
	}
	if webhook.MaxRetries <= 0 {
		webhook.MaxRetries = 3
	}
	if webhook.MaxRetries > 10 {
		webhook.MaxRetries = 10
	}
	if webhook.RetryInterval <= 0 {
		webhook.RetryInterval = 60
	}
	if webhook.RetryInterval > 3600 {
		webhook.RetryInterval = 3600
	}
	if webhook.RetryJitterPercent < 0 {
		webhook.RetryJitterPercent = 0
	}
	if webhook.RetryJitterPercent > 100 {
		webhook.RetryJitterPercent = 100
	}
	if webhook.MaxRetryDuration < 0 {
		webhook.MaxRetryDuration = 0
	}
	if webhook.MaxRetryDuration > 604800 {
		webhook.MaxRetryDuration = 604800
	}
	if err := validateCloudWebhook(webhook); err != nil {
		return nil, err
	}
	return webhook, nil
}

func cloudWebhookSecretRotationAttrs(webhook models.CloudWebhook, nextSecret string, gracePeriodSeconds int) models.Attrs {
	now := models.Time(time.Now())
	gracePeriodSeconds = normalizeCloudWebhookSecretGracePeriod(gracePeriodSeconds)
	currentVersion := cloudWebhookSecretVersion(webhook)
	nextVersion := currentVersion + 1
	previousSecret := ""
	previousVersion := 0
	previousExpiresAt := models.Time{}
	if strings.TrimSpace(webhook.Secret) != "" {
		previousSecret = webhook.Secret
		previousVersion = currentVersion
		if gracePeriodSeconds > 0 {
			previousExpiresAt = models.Time(time.Now().Add(time.Duration(gracePeriodSeconds) * time.Second))
		}
	}
	return models.Attrs{
		"secret":                     strings.TrimSpace(nextSecret),
		"secret_version":             nextVersion,
		"previous_secret":            previousSecret,
		"previous_secret_version":    previousVersion,
		"previous_secret_expires_at": previousExpiresAt,
		"secret_rotated_at":          now,
	}
}

func normalizeCloudWebhookSecretGracePeriod(value int) int {
	if value <= 0 {
		return cloudWebhookDefaultSecretGraceSeconds
	}
	if value > cloudWebhookMaxSecretGraceSeconds {
		return cloudWebhookMaxSecretGraceSeconds
	}
	return value
}

func cloudWebhookSecretVersion(webhook models.CloudWebhook) int {
	if webhook.SecretVersion <= 0 {
		return 1
	}
	return webhook.SecretVersion
}

func validateCloudWebhook(webhook *models.CloudWebhook) e.Error {
	if webhook.Name == "" {
		return e.New(e.BadParam, fmt.Errorf("webhook name is required"), http.StatusBadRequest)
	}
	parsed, err := url.Parse(webhook.TargetUrl)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return e.New(e.BadParam, fmt.Errorf("invalid webhook targetUrl"), http.StatusBadRequest)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return e.New(e.BadParam, fmt.Errorf("webhook targetUrl must be http or https"), http.StatusBadRequest)
	}
	if webhook.Status != models.CloudWebhookStatusEnabled && webhook.Status != models.CloudWebhookStatusDisabled {
		return e.New(e.BadParam, fmt.Errorf("invalid webhook status"), http.StatusBadRequest)
	}
	return nil
}

func cloudWebhookResp(c *ctx.ServiceContext, webhook models.CloudWebhook) resps.CloudWebhookResp {
	webhook.Secret = ""
	webhook.PreviousSecret = ""
	return resps.CloudWebhookResp{
		CloudWebhook:  webhook,
		DeliveryCount: cloudWebhookDeliveryCount(c, webhook.Id, ""),
		SuccessCount:  cloudWebhookDeliveryCount(c, webhook.Id, models.CloudWebhookDeliverySuccess),
		FailedCount:   cloudWebhookDeliveryCount(c, webhook.Id, models.CloudWebhookDeliveryFailed),
	}
}

func cloudWebhookDeadLetterResp(c *ctx.ServiceContext, deadLetter models.CloudWebhookDeadLetter) resps.CloudWebhookDeadLetterResp {
	return resps.CloudWebhookDeadLetterResp{
		CloudWebhookDeadLetter: deadLetter,
		WebhookName:            lookupName(c, &models.CloudWebhook{}, deadLetter.WebhookId),
	}
}

func cloudWebhookDeliveryCount(c *ctx.ServiceContext, webhookId models.Id, status string) int64 {
	query := c.DB().Model(&models.CloudWebhookDelivery{}).Where("org_id = ? and webhook_id = ?", c.OrgId, webhookId)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	count, err := cloudOverviewCount(query)
	if err != nil {
		return 0
	}
	return count
}

func auditCloudWebhookDeliveryDetail(c *ctx.ServiceContext, delivery models.CloudWebhookDelivery, webhookName string) {
	if c == nil || delivery.Id == "" {
		return
	}
	services.InsertUserOperateLog(
		c.UserId,
		c.OrgId,
		delivery.Id,
		consts.OperatorObjectTypeWebhookDelivery,
		"view",
		firstNonEmpty(webhookName, delivery.WebhookId.String()),
		models.ResAttrs{
			"webhookId":    delivery.WebhookId.String(),
			"eventId":      delivery.EventId.String(),
			"eventType":    delivery.EventType,
			"status":       delivery.Status,
			"deliveryMode": delivery.DeliveryMode,
			"attempt":      delivery.Attempt,
			"responseCode": delivery.ResponseCode,
			"targetUrl":    delivery.TargetUrl,
		},
	)
}

func cloudWebhookSignatureVerificationEvent(c *ctx.ServiceContext, delivery models.CloudWebhookDelivery, reportSource string) {
	if c == nil || delivery.Id == "" {
		return
	}
	level := models.CloudEventLevelInfo
	if delivery.SignatureVerifyStatus == models.CloudWebhookSignatureVerifyInvalid {
		level = models.CloudEventLevelWarning
	}
	recordCloudEventWithoutDispatch(c, models.CloudEvent{
		OrgId:        delivery.OrgId,
		Source:       models.CloudEventSourceNotification,
		EventType:    "webhook.signature_verification_reported",
		Level:        level,
		Status:       delivery.SignatureVerifyStatus,
		ResourceType: "webhook_delivery",
		ResourceId:   delivery.Id.String(),
		ResourceName: delivery.WebhookId.String(),
		Title:        "Webhook 验签结果回传",
		Message:      fmt.Sprintf("Webhook 投递 %s 接收端验签结果为 %s", delivery.Id.String(), delivery.SignatureVerifyStatus),
		Payload: models.ResAttrs{
			"webhookId":                delivery.WebhookId.String(),
			"deliveryId":               delivery.Id.String(),
			"eventId":                  delivery.EventId.String(),
			"eventType":                delivery.EventType,
			"signatureVerifyStatus":    delivery.SignatureVerifyStatus,
			"signatureVerifyVersion":   delivery.SignatureVerifyVersion,
			"signatureVerifyMessage":   delivery.SignatureVerifyMessage,
			"signatureVerifiedAt":      delivery.SignatureVerifiedAt,
			"signatureReportSource":    reportSource,
			"signatureReportTokenUsed": reportSource == "token",
		},
		OccurredAt: delivery.SignatureVerifiedAt,
	})
}

func cloudWebhookMatchesEvent(webhook models.CloudWebhook, event models.CloudEvent) bool {
	return stringSliceMatches(webhook.EventTypes, event.EventType) && stringSliceMatches(webhook.Sources, event.Source)
}

func stringSliceMatches(values []string, target string) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || value == "*" || value == target || stringWildcardMatches(value, target) {
			return true
		}
	}
	return false
}

func stringWildcardMatches(pattern, target string) bool {
	if !strings.HasSuffix(pattern, ".*") || target == "" {
		return false
	}
	prefix := strings.TrimSuffix(pattern, ".*")
	return target == prefix ||
		strings.HasPrefix(target, prefix+".") ||
		strings.HasPrefix(target, "cloud_"+prefix+".") ||
		(prefix == "cloud" && strings.HasPrefix(target, "cloud_"))
}

func normalizeStringSlice(values []string) []string {
	seen := make(map[string]bool, len(values))
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

func deliverCloudWebhook(c *ctx.ServiceContext, webhook models.CloudWebhook, event models.CloudEvent) (*models.CloudWebhookDelivery, e.Error) {
	return deliverCloudWebhookWithOptions(c, webhook, event, cloudWebhookDeliveryOptions{
		Attempt:       1,
		DeliveryMode:  models.CloudWebhookDeliveryModeInitial,
		ScheduleRetry: true,
	})
}

func deliverCloudWebhookWithOptions(c *ctx.ServiceContext, webhook models.CloudWebhook, event models.CloudEvent, opts cloudWebhookDeliveryOptions) (*models.CloudWebhookDelivery, e.Error) {
	result, err := cloudwebhooksrv.Deliver(c.DB(), webhook, event, cloudwebhooksrv.DeliveryOptions(opts))
	if err != nil {
		return nil, e.New(e.InternalError, err)
	}
	if result != nil && result.Delivery != nil && result.Delivery.Status == models.CloudWebhookDeliveryFailed && result.FinalFailureReason != "" {
		cloudWebhookDeliveryFailedEvent(c, webhook, *result.Delivery, result.FinalFailureReason)
	}
	if result == nil {
		return nil, e.New(e.InternalError, fmt.Errorf("cloud webhook delivery result is nil"))
	}
	return result.Delivery, nil
}

func cloudWebhookSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func cloudWebhookNewSignatureReportToken() (string, string, error) {
	raw := make([]byte, cloudWebhookSignatureReportTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, cloudWebhookSignatureReportTokenHash(token), nil
}

func cloudWebhookSignatureReportTokenHash(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func cloudWebhookSignatureReportTokenMatches(hash string, token string) bool {
	if strings.TrimSpace(hash) == "" || strings.TrimSpace(token) == "" {
		return false
	}
	expected := cloudWebhookSignatureReportTokenHash(token)
	return hmac.Equal([]byte(hash), []byte(expected))
}

func cloudWebhookSignatureInfo(webhook models.CloudWebhook, body []byte) models.ResAttrs {
	info := models.ResAttrs{
		"enabled":      webhook.Secret != "",
		"algorithm":    "HMAC-SHA256",
		"scheme":       "sha256",
		"header":       "X-CloudIaC-Signature",
		"version":      cloudWebhookSecretVersion(webhook),
		"payloadBytes": len(body),
	}
	if !time.Time(webhook.SecretRotatedAt).IsZero() {
		info["rotatedAt"] = webhook.SecretRotatedAt
	}
	if cloudWebhookPreviousSecretInGrace(webhook) {
		info["previousVersion"] = webhook.PreviousSecretVersion
		info["previousExpiresAt"] = webhook.PreviousSecretExpiresAt
		info["previousInGrace"] = true
	} else {
		info["previousInGrace"] = false
	}
	return info
}

func cloudWebhookPreviousSecretInGrace(webhook models.CloudWebhook) bool {
	if webhook.PreviousSecret == "" || webhook.PreviousSecretVersion <= 0 {
		return false
	}
	expiresAt := time.Time(webhook.PreviousSecretExpiresAt)
	return !expiresAt.IsZero() && expiresAt.After(time.Now())
}

func cloudWebhookHeadersAttrs(headers http.Header, maskSignature bool) models.ResAttrs {
	result := models.ResAttrs{}
	for key, values := range headers {
		copied := make([]string, 0, len(values))
		for _, value := range values {
			if maskSignature && (strings.EqualFold(key, "X-CloudIaC-Signature") || strings.EqualFold(key, CloudWebhookSignatureReportTokenHeader)) {
				value = maskCloudWebhookSignature(value)
			}
			if len(value) > 512 {
				value = value[:512] + "..."
			}
			copied = append(copied, value)
		}
		if len(copied) == 1 {
			result[key] = copied[0]
		} else if len(copied) > 1 {
			result[key] = copied
		}
	}
	return result
}

func maskCloudWebhookSignature(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.SplitN(value, "=", 2)
	if len(parts) == 2 {
		return parts[0] + "=<masked>"
	}
	return "<masked>"
}

func cloudWebhookDeliveryFailedEvent(c *ctx.ServiceContext, webhook models.CloudWebhook, delivery models.CloudWebhookDelivery, reason string) {
	recordCloudWebhookDeadLetter(c, webhook, delivery, reason)
	message := fmt.Sprintf("Webhook %s 投递事件 %s 失败，已达到最大重试次数 %d",
		webhook.Name, delivery.EventType, cloudWebhookMaxRetries(webhook))
	if reason == "retry_window_expired" {
		message = fmt.Sprintf("Webhook %s 投递事件 %s 失败，已超过最大重试窗口 %d 秒",
			webhook.Name, delivery.EventType, cloudWebhookMaxRetryDuration(webhook))
	}
	recordCloudEventWithoutDispatch(c, models.CloudEvent{
		OrgId:        webhook.OrgId,
		Source:       models.CloudEventSourceNotification,
		EventType:    "webhook.delivery_failed",
		Level:        models.CloudEventLevelError,
		Status:       models.CloudWebhookDeliveryFailed,
		ResourceType: "webhook",
		ResourceId:   webhook.Id.String(),
		ResourceName: webhook.Name,
		Title:        "Webhook 投递失败",
		Message:      message,
		Payload: models.ResAttrs{
			"webhookId":        webhook.Id.String(),
			"deliveryId":       delivery.Id.String(),
			"eventId":          delivery.EventId.String(),
			"eventType":        delivery.EventType,
			"deliveryMode":     delivery.DeliveryMode,
			"attempt":          delivery.Attempt,
			"maxRetries":       cloudWebhookMaxRetries(webhook),
			"reason":           reason,
			"maxRetryDuration": cloudWebhookMaxRetryDuration(webhook),
			"targetUrl":        delivery.TargetUrl,
			"responseCode":     delivery.ResponseCode,
			"errorMessage":     delivery.ErrorMessage,
			"parentDeliveryId": delivery.ParentDeliveryId.String(),
		},
		OccurredAt: delivery.DeliveredAt,
	})
}

func syncCloudWebhookDeadLetters(c *ctx.ServiceContext) e.Error {
	minRetryAt := cloudWebhookMinRetryAt()
	deliveries := make([]models.CloudWebhookDelivery, 0)
	if err := c.DB().Model(&models.CloudWebhookDelivery{}).
		Where(
			"org_id = ? and status = ? and delivery_mode <> ? and (next_retry_at is null or next_retry_at <= ?) and (retried_at is null or retried_at <= ?)",
			c.OrgId, models.CloudWebhookDeliveryFailed, models.CloudWebhookDeliveryModeTest, minRetryAt, minRetryAt,
		).
		Order("created_at desc").
		Limit(100).
		Find(&deliveries); err != nil {
		return e.New(e.DBError, err)
	}
	for _, delivery := range deliveries {
		webhook := models.CloudWebhook{}
		webhook.OrgId = c.OrgId
		webhook.Id = delivery.WebhookId
		recordCloudWebhookDeadLetter(c, webhook, delivery, "unscheduled_failure")
	}
	return nil
}

func recordCloudWebhookDeadLetter(c *ctx.ServiceContext, webhook models.CloudWebhook, delivery models.CloudWebhookDelivery, reason string) {
	if c == nil || delivery.Id == "" || delivery.DeliveryMode == models.CloudWebhookDeliveryModeTest {
		return
	}
	existing := models.CloudWebhookDeadLetter{}
	if err := c.DB().Model(&models.CloudWebhookDeadLetter{}).
		Where("org_id = ? and delivery_id = ?", delivery.OrgId, delivery.Id).
		First(&existing); err == nil {
		return
	} else if !e.IsRecordNotFound(err) {
		c.Logger().Warnf("lookup cloud webhook dead letter failed: %v", err)
		return
	}
	deadLetter := models.CloudWebhookDeadLetter{
		OrgId:        delivery.OrgId,
		WebhookId:    delivery.WebhookId,
		DeliveryId:   delivery.Id,
		EventId:      delivery.EventId,
		EventType:    delivery.EventType,
		Status:       models.CloudWebhookDeadLetterStatusOpen,
		Reason:       firstNonEmpty(reason, "final_failure"),
		TargetUrl:    delivery.TargetUrl,
		Attempt:      delivery.Attempt,
		ResponseCode: delivery.ResponseCode,
		ErrorMessage: delivery.ErrorMessage,
		Payload: models.ResAttrs{
			"webhookId":        firstNonEmpty(webhook.Id.String(), delivery.WebhookId.String()),
			"deliveryId":       delivery.Id.String(),
			"eventId":          delivery.EventId.String(),
			"eventType":        delivery.EventType,
			"deliveryMode":     delivery.DeliveryMode,
			"attempt":          delivery.Attempt,
			"targetUrl":        delivery.TargetUrl,
			"responseCode":     delivery.ResponseCode,
			"errorMessage":     delivery.ErrorMessage,
			"parentDeliveryId": delivery.ParentDeliveryId.String(),
		},
	}
	deadLetter.Id = models.NewId("cwdl")
	if err := models.Create(c.DB(), &deadLetter); err != nil {
		c.Logger().Warnf("record cloud webhook dead letter failed: %v", err)
	}
}

func cloudWebhookMaxRetries(webhook models.CloudWebhook) int {
	if webhook.MaxRetries <= 0 {
		return 3
	}
	return webhook.MaxRetries
}

func cloudWebhookRetryInterval(webhook models.CloudWebhook) int {
	if webhook.RetryInterval <= 0 {
		return 60
	}
	return webhook.RetryInterval
}

func cloudWebhookRetryJitterPercent(webhook models.CloudWebhook) int {
	if webhook.RetryJitterPercent < 0 {
		return 0
	}
	if webhook.RetryJitterPercent > 100 {
		return 100
	}
	return webhook.RetryJitterPercent
}

func cloudWebhookMaxRetryDuration(webhook models.CloudWebhook) int {
	if webhook.MaxRetryDuration <= 0 {
		return 0
	}
	if webhook.MaxRetryDuration > 604800 {
		return 604800
	}
	return webhook.MaxRetryDuration
}

func cloudWebhookNextRetryAt(c *ctx.ServiceContext, webhook models.CloudWebhook, delivery models.CloudWebhookDelivery, deliveredAt models.Time) (models.Time, bool) {
	delay := cloudWebhookRetryDelay(webhook, delivery.Attempt, delivery.Id.String())
	nextRetryTime := time.Time(deliveredAt).Add(time.Duration(delay) * time.Second)
	maxDuration := cloudWebhookMaxRetryDuration(webhook)
	if maxDuration <= 0 {
		return models.Time(nextRetryTime), true
	}
	startedAt := cloudWebhookRetryWindowStartedAt(c, delivery, deliveredAt)
	if startedAt.IsZero() {
		return models.Time(nextRetryTime), true
	}
	if nextRetryTime.After(startedAt.Add(time.Duration(maxDuration) * time.Second)) {
		return models.Time{}, false
	}
	return models.Time(nextRetryTime), true
}

func cloudWebhookRetryWindowStartedAt(c *ctx.ServiceContext, delivery models.CloudWebhookDelivery, fallback models.Time) time.Time {
	first := models.CloudWebhookDelivery{}
	if err := c.DB().Model(&models.CloudWebhookDelivery{}).
		Where("org_id = ? and webhook_id = ? and event_id = ?", delivery.OrgId, delivery.WebhookId, delivery.EventId).
		Order("created_at asc").
		First(&first); err == nil {
		if deliveredAt := time.Time(first.DeliveredAt); !deliveredAt.IsZero() {
			return deliveredAt
		}
		if createdAt := time.Time(first.CreatedAt); !createdAt.IsZero() {
			return createdAt
		}
	}
	return time.Time(fallback)
}

func cloudWebhookRetryDelay(webhook models.CloudWebhook, attempt int, seed string) int {
	delay := cloudWebhookBaseRetryDelay(webhook, attempt)
	return cloudWebhookRetryDelayWithJitter(delay, cloudWebhookRetryJitterPercent(webhook), seed)
}

func cloudWebhookBaseRetryDelay(webhook models.CloudWebhook, attempt int) int {
	if attempt <= 1 {
		return cloudWebhookRetryInterval(webhook)
	}
	delay := cloudWebhookRetryInterval(webhook)
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= 3600 {
			return 3600
		}
	}
	return delay
}

func cloudWebhookRetryDelayWithJitter(delay int, percent int, seed string) int {
	if delay <= 1 || percent <= 0 {
		return delay
	}
	spread := delay * percent / 100
	if spread <= 0 {
		return delay
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(seed))
	offset := int(hash.Sum32()%uint32(spread*2+1)) - spread
	result := delay + offset
	if result < 1 {
		return 1
	}
	if result > 3600 {
		return 3600
	}
	return result
}
