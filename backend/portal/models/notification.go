// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package models

import (
	"cloudiac/portal/libs/db"

	"github.com/lib/pq"
)

const (
	NotificationTypeEmail    = "email"
	NotificationTypeWebhook  = "webhook"
	NotificationTypeWeChat   = "wechat"
	NotificationTypeSlack    = "slack"
	NotificationTypeDingTalk = "dingtalk"

	NotificationDeliveryStatusSuccess = "success"
	NotificationDeliveryStatusFailed  = "failed"

	NotificationTemplateStatusEnable  = Enable
	NotificationTemplateStatusDisable = Disable

	NotificationTemplateVersionActionCreate   = "create"
	NotificationTemplateVersionActionUpdate   = "update"
	NotificationTemplateVersionActionCopy     = "copy"
	NotificationTemplateVersionActionRollback = "rollback"
)

// 通知类型 email, webhook, 钉钉， 企业微信，slack
// 事件 running(发起)、approving(审批)、complete(成功)、failed(失败)

type Notification struct {
	BaseModel

	OrgId     Id             `json:"orgId" gorm:"size:32;not null;comment:组织ID"`
	ProjectId Id             `json:"projectId" form:"projectId"  gorm:"size:32;not null;comment:项目ID"`
	Name      string         `json:"name" form:"name" `
	Type      string         `json:"notificationType" gorm:"type:enum('email', 'webhook', 'wechat', 'slack','dingtalk');default:'email';comment:通知类型"`
	Secret    string         `json:"secret" form:"secret" gorm:"comment:dingtalk加签秘钥"`
	Url       string         `json:"url" form:"url" gorm:"comment:回调url"`
	UserIds   pq.StringArray `json:"userIds"  gorm:"type:text;comment:用户ID"  swaggertype:"array,string"`
	Creator   Id             `json:"creator" form:"creator" `
}

func (Notification) TableName() string {
	return "iac_notification"
}

type NotificationEvent struct {
	AutoUintIdModel

	EventType      string `json:"eventType" form:"eventType"  gorm:"size:128;not null;default:'task.running';comment:事件类型"`
	NotificationId Id     `json:"notificationId" form:"notificationId" gorm:"size:32;not null"`
}

func (NotificationEvent) TableName() string {
	return "iac_notification_event"
}

func (NotificationEvent) Migrate(tx *db.Session) error {
	if err := tx.ModifyModelColumn(&NotificationEvent{}, "event_type"); err != nil {
		return err
	}
	return nil
}

type NotificationDelivery struct {
	SoftDeleteModel

	OrgId            Id     `json:"orgId" gorm:"index;size:32;not null;comment:组织ID"`
	ProjectId        Id     `json:"projectId" gorm:"index;size:32;not null;default:'';comment:项目ID"`
	NotificationId   Id     `json:"notificationId" gorm:"index;size:32;not null;comment:通知配置ID"`
	EventId          Id     `json:"eventId" gorm:"index;size:32;not null;default:'';comment:事件ID"`
	EventType        string `json:"eventType" gorm:"index;size:128;not null;default:'';comment:事件类型"`
	NotificationType string `json:"notificationType" gorm:"index;size:32;not null;default:'';comment:通知类型"`
	Target           string `json:"target" gorm:"size:512;not null;default:'';comment:通知目标"`
	Title            string `json:"title" gorm:"size:255;not null;default:'';comment:通知标题"`
	Message          string `json:"message" gorm:"type:text;comment:通知内容"`
	Status           string `json:"status" gorm:"index;size:32;not null;default:'success';comment:投递状态"`
	ErrorMessage     string `json:"errorMessage" gorm:"type:text;comment:错误信息"`
	DeliveredAt      Time   `json:"deliveredAt" gorm:"index;type:datetime;default:null;comment:投递时间"`
}

func (NotificationDelivery) TableName() string {
	return "iac_notification_delivery"
}

type NotificationTemplate struct {
	SoftDeleteModel

	OrgId            Id     `json:"orgId" gorm:"index;size:32;not null;comment:组织ID"`
	ProjectId        Id     `json:"projectId" form:"projectId" gorm:"index;size:32;not null;default:'';comment:项目ID"`
	Name             string `json:"name" form:"name" gorm:"size:255;not null;comment:模板名称"`
	EventType        string `json:"eventType" form:"eventType" gorm:"index;size:128;not null;comment:事件类型"`
	NotificationType string `json:"notificationType" form:"notificationType" gorm:"index;size:32;not null;comment:通知类型"`
	Title            string `json:"title" form:"title" gorm:"size:255;not null;default:'';comment:通知标题模板"`
	Content          string `json:"content" form:"content" gorm:"type:text;comment:文本/邮件模板"`
	MarkdownContent  string `json:"markdownContent" form:"markdownContent" gorm:"type:text;comment:Markdown模板"`
	Status           string `json:"status" form:"status" gorm:"index;size:32;not null;default:'enable';comment:状态"`
	Creator          Id     `json:"creator" form:"creator" gorm:"size:32;not null;comment:创建人"`
	SourceTemplateId Id     `json:"sourceTemplateId" form:"sourceTemplateId" gorm:"index;size:32;not null;default:'';comment:来源组织模板ID"`
}

func (NotificationTemplate) TableName() string {
	return "iac_notification_template"
}

func (NotificationTemplate) NewId() Id {
	return NewId("ntpl")
}

func (t NotificationTemplate) Migrate(tx *db.Session) error {
	return t.AddUniqueIndex(tx, "unique__notification_template_scope",
		"org_id", "project_id", "event_type", "notification_type")
}

type NotificationTemplateVersion struct {
	SoftDeleteModel

	OrgId                  Id     `json:"orgId" gorm:"index;size:32;not null;comment:组织ID"`
	ProjectId              Id     `json:"projectId" gorm:"index;size:32;not null;default:'';comment:项目ID"`
	TemplateId             Id     `json:"templateId" gorm:"index;size:32;not null;comment:通知模板ID"`
	VersionNo              int    `json:"versionNo" gorm:"index;not null;comment:版本号"`
	Action                 string `json:"action" gorm:"index;size:32;not null;default:'update';comment:版本动作"`
	Name                   string `json:"name" gorm:"size:255;not null;comment:模板名称快照"`
	EventType              string `json:"eventType" gorm:"index;size:128;not null;comment:事件类型快照"`
	NotificationType       string `json:"notificationType" gorm:"index;size:32;not null;comment:通知类型快照"`
	Title                  string `json:"title" gorm:"size:255;not null;default:'';comment:标题模板快照"`
	Content                string `json:"content" gorm:"type:text;comment:文本模板快照"`
	MarkdownContent        string `json:"markdownContent" gorm:"type:text;comment:Markdown模板快照"`
	Status                 string `json:"status" gorm:"index;size:32;not null;default:'enable';comment:状态快照"`
	SourceTemplateId       Id     `json:"sourceTemplateId" gorm:"index;size:32;not null;default:'';comment:来源组织模板ID"`
	SourceName             string `json:"sourceName" gorm:"size:255;not null;default:'';comment:来源模板名称快照"`
	SourceEventType        string `json:"sourceEventType" gorm:"size:128;not null;default:'';comment:来源事件类型快照"`
	SourceNotificationType string `json:"sourceNotificationType" gorm:"size:32;not null;default:'';comment:来源通知类型快照"`
	SourceTitle            string `json:"sourceTitle" gorm:"size:255;not null;default:'';comment:来源标题模板快照"`
	SourceContent          string `json:"sourceContent" gorm:"type:text;comment:来源文本模板快照"`
	SourceMarkdownContent  string `json:"sourceMarkdownContent" gorm:"type:text;comment:来源Markdown模板快照"`
	SourceStatus           string `json:"sourceStatus" gorm:"size:32;not null;default:'';comment:来源状态快照"`
	Operator               Id     `json:"operator" gorm:"index;size:32;not null;default:'';comment:操作人"`
}

func (NotificationTemplateVersion) TableName() string {
	return "iac_notification_template_version"
}

func (v NotificationTemplateVersion) Migrate(tx *db.Session) error {
	return v.AddUniqueIndex(tx, "unique__notification_template_version_no", "template_id", "version_no")
}

func FindEnabledNotificationTemplate(tx *db.Session, orgId, projectId Id, eventType, notificationType string) (*NotificationTemplate, error) {
	if tx == nil || orgId == "" || eventType == "" || notificationType == "" {
		return nil, nil
	}
	if projectId != "" {
		templates := make([]NotificationTemplate, 0, 1)
		if err := tx.Where("org_id = ? and project_id = ? and event_type = ? and notification_type = ? and status = ?",
			orgId, projectId, eventType, notificationType, NotificationTemplateStatusEnable).
			Limit(1).Find(&templates); err != nil {
			return nil, err
		}
		if len(templates) > 0 {
			return &templates[0], nil
		}
	}

	templates := make([]NotificationTemplate, 0, 1)
	if err := tx.Where("org_id = ? and (project_id = '' or project_id is null) and event_type = ? and notification_type = ? and status = ?",
		orgId, eventType, notificationType, NotificationTemplateStatusEnable).
		Limit(1).Find(&templates); err != nil {
		return nil, err
	}
	if len(templates) == 0 {
		return nil, nil
	}
	return &templates[0], nil
}
