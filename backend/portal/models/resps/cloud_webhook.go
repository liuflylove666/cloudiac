// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type CloudWebhookResp struct {
	models.CloudWebhook

	DeliveryCount int64 `json:"deliveryCount"`
	SuccessCount  int64 `json:"successCount"`
	FailedCount   int64 `json:"failedCount"`
}

type CloudWebhookDeliveryResp struct {
	models.CloudWebhookDelivery

	WebhookName string `json:"webhookName"`
}

type CloudWebhookDeadLetterResp struct {
	models.CloudWebhookDeadLetter

	WebhookName string `json:"webhookName"`
}

type CloudWebhookQueueSummaryResp struct {
	TotalDeliveries  int64                               `json:"totalDeliveries"`
	Pending          int64                               `json:"pending"`
	Success          int64                               `json:"success"`
	Failed           int64                               `json:"failed"`
	SuccessRate      float64                             `json:"successRate"`
	Queued           int64                               `json:"queued"`
	Due              int64                               `json:"due"`
	Future           int64                               `json:"future"`
	DeadLetter       int64                               `json:"deadLetter"`
	Retried          int64                               `json:"retried"`
	Initial          int64                               `json:"initial"`
	Manual           int64                               `json:"manual"`
	Auto             int64                               `json:"auto"`
	Test             int64                               `json:"test"`
	QueueConsumed    int64                               `json:"queueConsumed"`
	QueueSuccess     int64                               `json:"queueSuccess"`
	QueueFailed      int64                               `json:"queueFailed"`
	QueueSkipped     int64                               `json:"queueSkipped"`
	QueueSuccessRate float64                             `json:"queueSuccessRate"`
	OldestQueuedAt   models.Time                         `json:"oldestQueuedAt"`
	LastConsumedAt   models.Time                         `json:"lastConsumedAt"`
	LastDeliveredAt  models.Time                         `json:"lastDeliveredAt"`
	ShardTotal       int                                 `json:"shardTotal"`
	Shards           []CloudWebhookQueueShardSummaryResp `json:"shards"`
}

type CloudWebhookQueueShardSummaryResp struct {
	ShardIndex     int         `json:"shardIndex"`
	ShardTotal     int         `json:"shardTotal"`
	Queued         int64       `json:"queued"`
	Due            int64       `json:"due"`
	Future         int64       `json:"future"`
	Consumed       int64       `json:"consumed"`
	Success        int64       `json:"success"`
	Failed         int64       `json:"failed"`
	Skipped        int64       `json:"skipped"`
	SuccessRate    float64     `json:"successRate"`
	OldestQueuedAt models.Time `json:"oldestQueuedAt"`
	LastConsumedAt models.Time `json:"lastConsumedAt"`
}
