// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package forms

import "cloudiac/portal/models"

type SearchCloudWebhookForm struct {
	PageForm

	Q      string `form:"q" json:"q"`
	Status string `form:"status" json:"status" binding:"omitempty,oneof=enable disable"`
	Source string `form:"source" json:"source"`
}

type CloudWebhookParam struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type CreateCloudWebhookForm struct {
	BaseForm

	Name               string   `form:"name" json:"name" binding:"required,gte=2,lte=128"`
	Description        string   `form:"description" json:"description" binding:"max=255"`
	TargetUrl          string   `form:"targetUrl" json:"targetUrl" binding:"required,max=512"`
	Secret             string   `form:"secret" json:"secret" binding:"max=255"`
	EventTypes         []string `form:"eventTypes" json:"eventTypes"`
	Sources            []string `form:"sources" json:"sources"`
	Status             string   `form:"status" json:"status" binding:"omitempty,oneof=enable disable"`
	TimeoutSeconds     int      `form:"timeoutSeconds" json:"timeoutSeconds"`
	MaxRetries         int      `form:"maxRetries" json:"maxRetries"`
	RetryInterval      int      `form:"retryInterval" json:"retryInterval"`
	RetryJitterPercent int      `form:"retryJitterPercent" json:"retryJitterPercent"`
	MaxRetryDuration   int      `form:"maxRetryDuration" json:"maxRetryDuration"`
}

type UpdateCloudWebhookForm struct {
	CreateCloudWebhookForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type SearchCloudWebhookDeliveryForm struct {
	PageForm

	Id        models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	EventId   models.Id `form:"eventId" json:"eventId" binding:"max=32"`
	EventType string    `form:"eventType" json:"eventType"`
	Status    string    `form:"status" json:"status" binding:"omitempty,oneof=pending success failed"`
}

type CloudWebhookDeliveryParam struct {
	BaseForm

	Id         models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	DeliveryId models.Id `uri:"deliveryId" json:"deliveryId" binding:"required,max=32" swaggerignore:"true"`
}

type CloudWebhookDeliverySignatureParam struct {
	BaseForm

	DeliveryId models.Id `uri:"deliveryId" form:"deliveryId" json:"deliveryId" binding:"required,max=32" swaggerignore:"true"`
}

type RetryCloudWebhookDeliveryForm struct {
	CloudWebhookDeliveryParam
}

type RetryDueCloudWebhookDeliveriesForm struct {
	BaseForm

	ShardIndex int `form:"shardIndex" json:"shardIndex"`
	ShardTotal int `form:"shardTotal" json:"shardTotal"`
}

type ReportCloudWebhookSignatureVerificationForm struct {
	BaseForm

	DeliveryId       models.Id `uri:"deliveryId" form:"deliveryId" json:"deliveryId" binding:"required,max=32" swaggerignore:"true"`
	Status           string    `form:"status" json:"status" binding:"required,oneof=valid invalid skipped"`
	SignatureVersion int       `form:"signatureVersion" json:"signatureVersion"`
	Message          string    `form:"message" json:"message" binding:"max=512"`
}

type ReportCloudWebhookSignatureVerificationByTokenForm struct {
	BaseForm

	DeliveryId       models.Id `uri:"deliveryId" form:"deliveryId" json:"deliveryId" binding:"required,max=32" swaggerignore:"true"`
	Token            string    `form:"token" json:"token" binding:"max=255"`
	Status           string    `form:"status" json:"status" binding:"required,oneof=valid invalid skipped"`
	SignatureVersion int       `form:"signatureVersion" json:"signatureVersion"`
	Message          string    `form:"message" json:"message" binding:"max=512"`
}

type TestCloudWebhookForm struct {
	BaseForm

	Id models.Id `uri:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
}

type RotateCloudWebhookSecretForm struct {
	BaseForm

	Id                 models.Id `uri:"id" form:"id" json:"id" binding:"required,max=32" swaggerignore:"true"`
	Secret             string    `form:"secret" json:"secret" binding:"required,max=255"`
	GracePeriodSeconds int       `form:"gracePeriodSeconds" json:"gracePeriodSeconds"`
}

type SearchCloudWebhookDeadLetterForm struct {
	PageForm

	Q         string    `form:"q" json:"q"`
	Status    string    `form:"status" json:"status" binding:"omitempty,oneof=open replayed ignored"`
	WebhookId models.Id `form:"webhookId" json:"webhookId" binding:"max=32"`
	EventType string    `form:"eventType" json:"eventType"`
}

type CloudWebhookDeadLetterParam struct {
	BaseForm

	DeadLetterId models.Id `uri:"deadLetterId" form:"deadLetterId" json:"deadLetterId" binding:"required,max=32" swaggerignore:"true"`
}

type ReplayCloudWebhookDeadLetterForm struct {
	BaseForm

	DeadLetterId models.Id `uri:"deadLetterId" form:"deadLetterId" json:"deadLetterId" binding:"required,max=32" swaggerignore:"true"`
}

type IgnoreCloudWebhookDeadLetterForm struct {
	BaseForm

	DeadLetterId models.Id `uri:"deadLetterId" form:"deadLetterId" json:"deadLetterId" binding:"required,max=32" swaggerignore:"true"`

	Note string `form:"note" json:"note" binding:"max=255"`
}
