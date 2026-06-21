// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package cloudwebhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"strings"
	"time"

	"cloudiac/portal/libs/db"
	"cloudiac/portal/models"
)

const (
	SignatureReportTokenHeader = "X-CloudIaC-Signature-Report-Token"

	signatureReportTokenBytes = 32
	signatureReportTokenTTL   = 7 * 24 * time.Hour
)

type DeliveryOptions struct {
	Attempt          int
	ParentDeliveryId models.Id
	DeliveryMode     string
	ScheduleRetry    bool
}

type DeliveryResult struct {
	Delivery           *models.CloudWebhookDelivery
	FinalFailureReason string
}

func Deliver(sess *db.Session, webhook models.CloudWebhook, event models.CloudEvent, opts DeliveryOptions) (*DeliveryResult, error) {
	if opts.Attempt <= 0 {
		opts.Attempt = 1
	}
	if opts.DeliveryMode == "" {
		opts.DeliveryMode = models.CloudWebhookDeliveryModeInitial
	}
	payload := DeliveryPayload(webhook, event)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	reportToken, reportTokenHash, err := newSignatureReportToken()
	if err != nil {
		return nil, err
	}
	reportTokenExpiresAt := models.Time(time.Now().Add(signatureReportTokenTTL))
	delivery := models.CloudWebhookDelivery{
		OrgId:                         event.OrgId,
		WebhookId:                     webhook.Id,
		EventId:                       event.Id,
		ParentDeliveryId:              opts.ParentDeliveryId,
		EventType:                     event.EventType,
		Status:                        models.CloudWebhookDeliveryPending,
		DeliveryMode:                  opts.DeliveryMode,
		TargetUrl:                     webhook.TargetUrl,
		Attempt:                       opts.Attempt,
		RequestPayload:                payload,
		SignatureReportTokenHash:      reportTokenHash,
		SignatureReportTokenExpiresAt: reportTokenExpiresAt,
	}
	delivery.Id = models.NewId("cwd")
	if err := models.Create(sess, &delivery); err != nil {
		return nil, err
	}

	deliveredAt := models.Time(time.Now())
	status := models.CloudWebhookDeliveryFailed
	responseCode := 0
	responseBody := ""
	errorMessage := ""
	requestHeaders := models.ResAttrs{}
	responseHeaders := models.ResAttrs{}
	signatureInfo := SignatureInfo(webhook, body)
	signatureInfo["reportTokenHeader"] = SignatureReportTokenHeader
	signatureInfo["reportEndpoint"] = fmt.Sprintf("/api/v1/cloud/webhooks/deliveries/%s/signature-verification/report", delivery.Id.String())
	signatureInfo["reportTokenExpiresAt"] = reportTokenExpiresAt
	req, err := http.NewRequest(http.MethodPost, webhook.TargetUrl, bytes.NewReader(body))
	if err != nil {
		errorMessage = err.Error()
	} else {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "CloudIaC-Webhook/1.0")
		req.Header.Set("X-CloudIaC-Event", event.EventType)
		req.Header.Set("X-CloudIaC-Delivery", delivery.Id.String())
		req.Header.Set(SignatureReportTokenHeader, reportToken)
		if webhook.Secret != "" {
			req.Header.Set("X-CloudIaC-Signature", Signature(webhook.Secret, body))
			req.Header.Set("X-CloudIaC-Signature-Version", fmt.Sprintf("%d", SecretVersion(webhook)))
			if PreviousSecretInGrace(webhook) {
				req.Header.Set("X-CloudIaC-Previous-Signature-Version", fmt.Sprintf("%d", webhook.PreviousSecretVersion))
				req.Header.Set("X-CloudIaC-Previous-Signature-Expires-At", time.Time(webhook.PreviousSecretExpiresAt).Format(time.RFC3339))
			}
		}
		requestHeaders = HeadersAttrs(req.Header, true)
		client := &http.Client{Timeout: time.Duration(webhookTimeoutSeconds(webhook)) * time.Second}
		resp, httpErr := client.Do(req)
		if httpErr != nil {
			errorMessage = httpErr.Error()
		} else {
			defer resp.Body.Close()
			responseCode = resp.StatusCode
			responseHeaders = HeadersAttrs(resp.Header, false)
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			responseBody = string(respBody)
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				status = models.CloudWebhookDeliverySuccess
			} else {
				errorMessage = resp.Status
			}
		}
	}

	attrs := models.Attrs{
		"status":           status,
		"response_code":    responseCode,
		"response_body":    responseBody,
		"error_message":    errorMessage,
		"delivered_at":     deliveredAt,
		"request_headers":  requestHeaders,
		"response_headers": responseHeaders,
		"signature_info":   signatureInfo,
	}
	nextRetryAt := models.Time{}
	finalFailureReason := ""
	if opts.ScheduleRetry && status == models.CloudWebhookDeliveryFailed {
		if delivery.Attempt < MaxRetries(webhook) {
			if scheduledAt, ok := NextRetryAt(sess, webhook, delivery, deliveredAt); ok {
				nextRetryAt = scheduledAt
				attrs["next_retry_at"] = nextRetryAt
			} else {
				finalFailureReason = "retry_window_expired"
			}
		} else {
			finalFailureReason = "max_retries_reached"
		}
	}
	if _, err := sess.Model(&models.CloudWebhookDelivery{}).
		Where("id = ? and org_id = ?", delivery.Id, event.OrgId).
		UpdateAttrs(attrs); err != nil {
		return nil, err
	}
	delivery.Status = status
	delivery.ResponseCode = responseCode
	delivery.ResponseBody = responseBody
	delivery.ErrorMessage = errorMessage
	delivery.RequestHeaders = requestHeaders
	delivery.ResponseHeaders = responseHeaders
	delivery.SignatureInfo = signatureInfo
	delivery.DeliveredAt = deliveredAt
	delivery.NextRetryAt = nextRetryAt
	if !time.Time(nextRetryAt).IsZero() {
		if err := EnsureDeliveryQueue(sess, delivery, nextRetryAt); err != nil {
			return nil, err
		}
	}
	if _, err := sess.Model(&models.CloudWebhook{}).
		Where("id = ? and org_id = ?", webhook.Id, event.OrgId).
		UpdateAttrs(models.Attrs{
			"last_status":       status,
			"last_status_code":  responseCode,
			"last_message":      firstNonEmpty(errorMessage, responseBody),
			"last_delivered_at": deliveredAt,
		}); err != nil {
		return nil, err
	}
	return &DeliveryResult{Delivery: &delivery, FinalFailureReason: finalFailureReason}, nil
}

func DeliveryPayload(webhook models.CloudWebhook, event models.CloudEvent) models.ResAttrs {
	return models.ResAttrs{
		"event": models.ResAttrs{
			"id":             event.Id.String(),
			"source":         event.Source,
			"eventType":      event.EventType,
			"level":          event.Level,
			"status":         event.Status,
			"provider":       event.Provider,
			"accountId":      event.AccountId,
			"region":         event.Region,
			"resourceType":   event.ResourceType,
			"resourceId":     event.ResourceId,
			"resourceName":   event.ResourceName,
			"title":          event.Title,
			"message":        event.Message,
			"payload":        event.Payload,
			"occurredAt":     event.OccurredAt,
			"orgId":          event.OrgId.String(),
			"projectId":      event.ProjectId.String(),
			"envId":          event.EnvId.String(),
			"assetId":        event.AssetId.String(),
			"operationId":    event.OperationId.String(),
			"riskFindingId":  event.RiskFindingId.String(),
			"cloudAccountId": event.CloudAccountId.String(),
		},
		"webhook": models.ResAttrs{
			"id":   webhook.Id.String(),
			"name": webhook.Name,
		},
	}
}

func EnsureDeliveryQueue(sess *db.Session, delivery models.CloudWebhookDelivery, nextRunAt models.Time) error {
	if delivery.Id == "" || delivery.DeliveryMode == models.CloudWebhookDeliveryModeTest || time.Time(nextRunAt).IsZero() {
		return nil
	}
	existing := models.CloudWebhookDeliveryQueue{}
	if err := sess.Model(&models.CloudWebhookDeliveryQueue{}).
		Where("org_id = ? and delivery_id = ?", delivery.OrgId, delivery.Id).
		First(&existing); err == nil {
		if existing.Status == models.CloudWebhookDeliveryQueueStatusDone {
			return nil
		}
		_, updateErr := sess.Model(&models.CloudWebhookDeliveryQueue{}).
			Where("org_id = ? and id = ?", delivery.OrgId, existing.Id).
			UpdateAttrs(models.Attrs{
				"status":        models.CloudWebhookDeliveryQueueStatusQueued,
				"attempt":       delivery.Attempt,
				"next_run_at":   nextRunAt,
				"error_message": "",
			})
		return updateErr
	} else if !isRecordNotFound(err) {
		return err
	}
	queue := models.CloudWebhookDeliveryQueue{
		OrgId:      delivery.OrgId,
		WebhookId:  delivery.WebhookId,
		DeliveryId: delivery.Id,
		EventId:    delivery.EventId,
		EventType:  delivery.EventType,
		Status:     models.CloudWebhookDeliveryQueueStatusQueued,
		Attempt:    delivery.Attempt,
		NextRunAt:  nextRunAt,
	}
	queue.Id = models.NewId("cwq")
	return models.Create(sess, &queue)
}

func Signature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func SignatureReportTokenMatches(hash string, token string) bool {
	if strings.TrimSpace(hash) == "" || strings.TrimSpace(token) == "" {
		return false
	}
	expected := SignatureReportTokenHash(token)
	return hmac.Equal([]byte(hash), []byte(expected))
}

func SignatureReportTokenHash(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func SignatureInfo(webhook models.CloudWebhook, body []byte) models.ResAttrs {
	info := models.ResAttrs{
		"enabled":      webhook.Secret != "",
		"algorithm":    "HMAC-SHA256",
		"scheme":       "sha256",
		"header":       "X-CloudIaC-Signature",
		"version":      SecretVersion(webhook),
		"payloadBytes": len(body),
	}
	if !time.Time(webhook.SecretRotatedAt).IsZero() {
		info["rotatedAt"] = webhook.SecretRotatedAt
	}
	if PreviousSecretInGrace(webhook) {
		info["previousVersion"] = webhook.PreviousSecretVersion
		info["previousExpiresAt"] = webhook.PreviousSecretExpiresAt
		info["previousInGrace"] = true
	} else {
		info["previousInGrace"] = false
	}
	return info
}

func SecretVersion(webhook models.CloudWebhook) int {
	if webhook.SecretVersion <= 0 {
		return 1
	}
	return webhook.SecretVersion
}

func PreviousSecretInGrace(webhook models.CloudWebhook) bool {
	if webhook.PreviousSecret == "" || webhook.PreviousSecretVersion <= 0 {
		return false
	}
	expiresAt := time.Time(webhook.PreviousSecretExpiresAt)
	return !expiresAt.IsZero() && expiresAt.After(time.Now())
}

func HeadersAttrs(headers http.Header, maskSignature bool) models.ResAttrs {
	result := models.ResAttrs{}
	for key, values := range headers {
		copied := make([]string, 0, len(values))
		for _, value := range values {
			if maskSignature && (strings.EqualFold(key, "X-CloudIaC-Signature") || strings.EqualFold(key, SignatureReportTokenHeader)) {
				value = maskSignatureValue(value)
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

func MaxRetries(webhook models.CloudWebhook) int {
	if webhook.MaxRetries <= 0 {
		return 3
	}
	return webhook.MaxRetries
}

func MaxRetryDuration(webhook models.CloudWebhook) int {
	if webhook.MaxRetryDuration <= 0 {
		return 0
	}
	if webhook.MaxRetryDuration > 604800 {
		return 604800
	}
	return webhook.MaxRetryDuration
}

func NextRetryAt(sess *db.Session, webhook models.CloudWebhook, delivery models.CloudWebhookDelivery, deliveredAt models.Time) (models.Time, bool) {
	delay := RetryDelay(webhook, delivery.Attempt, delivery.Id.String())
	nextRetryTime := time.Time(deliveredAt).Add(time.Duration(delay) * time.Second)
	maxDuration := MaxRetryDuration(webhook)
	if maxDuration <= 0 {
		return models.Time(nextRetryTime), true
	}
	startedAt := retryWindowStartedAt(sess, delivery, deliveredAt)
	if startedAt.IsZero() {
		return models.Time(nextRetryTime), true
	}
	if nextRetryTime.After(startedAt.Add(time.Duration(maxDuration) * time.Second)) {
		return models.Time{}, false
	}
	return models.Time(nextRetryTime), true
}

func RetryDelay(webhook models.CloudWebhook, attempt int, seed string) int {
	delay := baseRetryDelay(webhook, attempt)
	return retryDelayWithJitter(delay, retryJitterPercent(webhook), seed)
}

func retryWindowStartedAt(sess *db.Session, delivery models.CloudWebhookDelivery, fallback models.Time) time.Time {
	first := models.CloudWebhookDelivery{}
	if err := sess.Model(&models.CloudWebhookDelivery{}).
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

func newSignatureReportToken() (string, string, error) {
	raw := make([]byte, signatureReportTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, SignatureReportTokenHash(token), nil
}

func webhookTimeoutSeconds(webhook models.CloudWebhook) int {
	if webhook.TimeoutSeconds <= 0 {
		return 5
	}
	if webhook.TimeoutSeconds > 30 {
		return 30
	}
	return webhook.TimeoutSeconds
}

func retryInterval(webhook models.CloudWebhook) int {
	if webhook.RetryInterval <= 0 {
		return 60
	}
	return webhook.RetryInterval
}

func retryJitterPercent(webhook models.CloudWebhook) int {
	if webhook.RetryJitterPercent < 0 {
		return 0
	}
	if webhook.RetryJitterPercent > 100 {
		return 100
	}
	return webhook.RetryJitterPercent
}

func baseRetryDelay(webhook models.CloudWebhook, attempt int) int {
	if attempt <= 1 {
		return retryInterval(webhook)
	}
	delay := retryInterval(webhook)
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= 3600 {
			return 3600
		}
	}
	return delay
}

func retryDelayWithJitter(delay int, percent int, seed string) int {
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

func maskSignatureValue(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.SplitN(value, "=", 2)
	if len(parts) == 2 {
		return parts[0] + "=<masked>"
	}
	return "<masked>"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func isRecordNotFound(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "record not found")
}
