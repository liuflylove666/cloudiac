// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package resps

import "cloudiac/portal/models"

type RespNotification struct {
	models.Notification
	EventType   string   `json:"-" form:"-" gorm:"event_type"`
	EventTypes  []string `json:"eventType" form:"eventType" gorm:"-"`
	CreatorName string   `json:"creatorName" form:"creatorName" `
}

type RespDetailNotification struct {
	models.Notification
	EventType  string   `json:"-" `
	EventTypes []string `json:"eventType" gorm:"-"`
}

type NotificationDeliveryResp struct {
	models.NotificationDelivery

	NotificationName string `json:"notificationName"`
}

type NotificationTemplateResp struct {
	models.NotificationTemplate

	CreatorName string `json:"creatorName"`
}

type NotificationTemplateCopyItem struct {
	SourceTemplateId  models.Id `json:"sourceTemplateId"`
	ProjectTemplateId models.Id `json:"projectTemplateId"`
	Name              string    `json:"name"`
	EventType         string    `json:"eventType"`
	Type              string    `json:"type"`
	Status            string    `json:"status"`
	Reason            string    `json:"reason"`
}

type NotificationTemplateCopyResp struct {
	Copied  int                            `json:"copied"`
	Skipped int                            `json:"skipped"`
	Items   []NotificationTemplateCopyItem `json:"items"`
}

type NotificationTemplateVersionResp struct {
	models.NotificationTemplateVersion

	OperatorName string `json:"operatorName"`
}

type NotificationTemplateVariable struct {
	Name        string      `json:"name"`
	Label       string      `json:"label"`
	Description string      `json:"description"`
	Sample      interface{} `json:"sample"`
}

type NotificationTemplateVariableGroup struct {
	Name      string                         `json:"name"`
	Label     string                         `json:"label"`
	Variables []NotificationTemplateVariable `json:"variables"`
}

type NotificationTemplateVariablesResp struct {
	Groups     []NotificationTemplateVariableGroup `json:"groups"`
	SampleData map[string]interface{}              `json:"sampleData"`
}

type NotificationTemplatePreviewResp struct {
	Title           string                 `json:"title"`
	Content         string                 `json:"content"`
	MarkdownContent string                 `json:"markdownContent"`
	SampleData      map[string]interface{} `json:"sampleData"`
}
