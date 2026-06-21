// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package models

import "cloudiac/portal/libs/db"

const (
	CloudWebhookStatusEnabled  = Enable
	CloudWebhookStatusDisabled = Disable

	CloudWebhookDeliveryPending = "pending"
	CloudWebhookDeliverySuccess = "success"
	CloudWebhookDeliveryFailed  = "failed"

	CloudWebhookDeliveryModeInitial = "initial"
	CloudWebhookDeliveryModeManual  = "manual"
	CloudWebhookDeliveryModeAuto    = "auto"
	CloudWebhookDeliveryModeTest    = "test"

	CloudWebhookSignatureVerifyValid   = "valid"
	CloudWebhookSignatureVerifyInvalid = "invalid"
	CloudWebhookSignatureVerifySkipped = "skipped"

	CloudWebhookDeadLetterStatusOpen     = "open"
	CloudWebhookDeadLetterStatusReplayed = "replayed"
	CloudWebhookDeadLetterStatusIgnored  = "ignored"

	CloudWebhookDeliveryQueueStatusQueued     = "queued"
	CloudWebhookDeliveryQueueStatusProcessing = "processing"
	CloudWebhookDeliveryQueueStatusDone       = "done"
	CloudWebhookDeliveryQueueStatusSkipped    = "skipped"
)

type CloudWebhook struct {
	SoftDeleteModel

	OrgId                   Id       `json:"orgId" gorm:"index;size:32;not null"`
	Name                    string   `json:"name" gorm:"size:128;not null"`
	Description             string   `json:"description" gorm:"size:255;not null;default:''"`
	TargetUrl               string   `json:"targetUrl" gorm:"size:512;not null"`
	Secret                  string   `json:"secret,omitempty" gorm:"size:255;not null;default:''"`
	SecretVersion           int      `json:"secretVersion" gorm:"not null;default:1"`
	PreviousSecret          string   `json:"previousSecret,omitempty" gorm:"size:255;not null;default:''"`
	PreviousSecretVersion   int      `json:"previousSecretVersion" gorm:"not null;default:0"`
	PreviousSecretExpiresAt Time     `json:"previousSecretExpiresAt" gorm:"type:datetime;default:null"`
	SecretRotatedAt         Time     `json:"secretRotatedAt" gorm:"type:datetime;default:null"`
	EventTypes              StrSlice `json:"eventTypes" gorm:"type:json"`
	Sources                 StrSlice `json:"sources" gorm:"type:json"`
	Status                  string   `json:"status" gorm:"index;size:32;not null;default:'enable'"`
	TimeoutSeconds          int      `json:"timeoutSeconds" gorm:"not null;default:5"`
	MaxRetries              int      `json:"maxRetries" gorm:"not null;default:3"`
	RetryInterval           int      `json:"retryInterval" gorm:"not null;default:60"`
	RetryJitterPercent      int      `json:"retryJitterPercent" gorm:"not null;default:0"`
	MaxRetryDuration        int      `json:"maxRetryDuration" gorm:"not null;default:0"`
	LastStatus              string   `json:"lastStatus" gorm:"index;size:32;not null;default:''"`
	LastStatusCode          int      `json:"lastStatusCode" gorm:"not null;default:0"`
	LastMessage             string   `json:"lastMessage" gorm:"type:text"`
	LastDeliveredAt         Time     `json:"lastDeliveredAt" gorm:"type:datetime;default:null"`
}

func (CloudWebhook) TableName() string {
	return "iac_cloud_webhook"
}

func (w CloudWebhook) Migrate(sess *db.Session) error {
	return w.AddUniqueIndex(sess, "unique_cloud_webhook_name", "org_id", "name")
}

type CloudWebhookDelivery struct {
	SoftDeleteModel

	OrgId                         Id       `json:"orgId" gorm:"index;size:32;not null"`
	WebhookId                     Id       `json:"webhookId" gorm:"index;size:32;not null"`
	EventId                       Id       `json:"eventId" gorm:"index;size:32;not null"`
	ParentDeliveryId              Id       `json:"parentDeliveryId" gorm:"index;size:32;not null;default:''"`
	EventType                     string   `json:"eventType" gorm:"index;size:128;not null;default:''"`
	Status                        string   `json:"status" gorm:"index;size:32;not null;default:'pending'"`
	DeliveryMode                  string   `json:"deliveryMode" gorm:"index;size:32;not null;default:'initial'"`
	TargetUrl                     string   `json:"targetUrl" gorm:"size:512;not null;default:''"`
	Attempt                       int      `json:"attempt" gorm:"not null;default:1"`
	RequestPayload                ResAttrs `json:"requestPayload,omitempty" gorm:"type:json"`
	RequestHeaders                ResAttrs `json:"requestHeaders,omitempty" gorm:"type:json"`
	ResponseHeaders               ResAttrs `json:"responseHeaders,omitempty" gorm:"type:json"`
	SignatureInfo                 ResAttrs `json:"signatureInfo,omitempty" gorm:"type:json"`
	SignatureReportTokenHash      string   `json:"-" gorm:"size:128;not null;default:''"`
	SignatureReportTokenExpiresAt Time     `json:"-" gorm:"type:datetime;default:null"`
	SignatureReportTokenUsedAt    Time     `json:"-" gorm:"type:datetime;default:null"`
	SignatureVerifyStatus         string   `json:"signatureVerifyStatus" gorm:"index;size:32;not null;default:''"`
	SignatureVerifyVersion        int      `json:"signatureVerifyVersion" gorm:"not null;default:0"`
	SignatureVerifyMessage        string   `json:"signatureVerifyMessage" gorm:"type:text"`
	SignatureVerifiedAt           Time     `json:"signatureVerifiedAt" gorm:"type:datetime;default:null"`
	ResponseCode                  int      `json:"responseCode" gorm:"not null;default:0"`
	ResponseBody                  string   `json:"responseBody" gorm:"type:text"`
	ErrorMessage                  string   `json:"errorMessage" gorm:"type:text"`
	NextRetryAt                   Time     `json:"nextRetryAt" gorm:"index;type:datetime;default:null"`
	RetriedAt                     Time     `json:"retriedAt" gorm:"type:datetime;default:null"`
	DeliveredAt                   Time     `json:"deliveredAt" gorm:"type:datetime;default:null"`
}

func (CloudWebhookDelivery) TableName() string {
	return "iac_cloud_webhook_delivery"
}

type CloudWebhookDeliveryQueue struct {
	SoftDeleteModel

	OrgId            Id     `json:"orgId" gorm:"index;size:32;not null"`
	WebhookId        Id     `json:"webhookId" gorm:"index;size:32;not null"`
	DeliveryId       Id     `json:"deliveryId" gorm:"index;size:32;not null"`
	EventId          Id     `json:"eventId" gorm:"index;size:32;not null"`
	EventType        string `json:"eventType" gorm:"index;size:128;not null;default:''"`
	Status           string `json:"status" gorm:"index;size:32;not null;default:'queued'"`
	Attempt          int    `json:"attempt" gorm:"not null;default:1"`
	NextRunAt        Time   `json:"nextRunAt" gorm:"index;type:datetime;default:null"`
	LockedAt         Time   `json:"lockedAt" gorm:"type:datetime;default:null"`
	LockedBy         string `json:"lockedBy" gorm:"size:128;not null;default:''"`
	ConsumedAt       Time   `json:"consumedAt" gorm:"type:datetime;default:null"`
	ResultDeliveryId Id     `json:"resultDeliveryId" gorm:"index;size:32;not null;default:''"`
	ErrorMessage     string `json:"errorMessage" gorm:"type:text"`
}

func (CloudWebhookDeliveryQueue) TableName() string {
	return "iac_cloud_webhook_delivery_queue"
}

func (q CloudWebhookDeliveryQueue) Migrate(sess *db.Session) error {
	return q.AddUniqueIndex(sess, "unique_cloud_webhook_delivery_queue_delivery", "org_id", "delivery_id")
}

type CloudWebhookDeadLetter struct {
	SoftDeleteModel

	OrgId              Id       `json:"orgId" gorm:"index;size:32;not null"`
	WebhookId          Id       `json:"webhookId" gorm:"index;size:32;not null"`
	DeliveryId         Id       `json:"deliveryId" gorm:"index;size:32;not null"`
	EventId            Id       `json:"eventId" gorm:"index;size:32;not null"`
	EventType          string   `json:"eventType" gorm:"index;size:128;not null;default:''"`
	Status             string   `json:"status" gorm:"index;size:32;not null;default:'open'"`
	Reason             string   `json:"reason" gorm:"index;size:64;not null;default:''"`
	TargetUrl          string   `json:"targetUrl" gorm:"size:512;not null;default:''"`
	Attempt            int      `json:"attempt" gorm:"not null;default:1"`
	ResponseCode       int      `json:"responseCode" gorm:"not null;default:0"`
	ErrorMessage       string   `json:"errorMessage" gorm:"type:text"`
	Payload            ResAttrs `json:"payload,omitempty" gorm:"type:json"`
	ReplayedAt         Time     `json:"replayedAt" gorm:"type:datetime;default:null"`
	ReplayedDeliveryId Id       `json:"replayedDeliveryId" gorm:"index;size:32;not null;default:''"`
	IgnoredAt          Time     `json:"ignoredAt" gorm:"type:datetime;default:null"`
	Note               string   `json:"note" gorm:"type:text"`
}

func (CloudWebhookDeadLetter) TableName() string {
	return "iac_cloud_webhook_dead_letter"
}

func (d CloudWebhookDeadLetter) Migrate(sess *db.Session) error {
	return d.AddUniqueIndex(sess, "unique_cloud_webhook_dead_letter_delivery", "org_id", "delivery_id")
}
