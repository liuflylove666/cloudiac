// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package services

import (
	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/db"
	"cloudiac/portal/models"
	"cloudiac/portal/models/resps"
	"fmt"
	"strings"
)

func SearchNotification(dbSess *db.Session, orgId, projectId models.Id) *db.Session {
	n := models.Notification{}.TableName()
	query := dbSess.Table(n).
		Joins(fmt.Sprintf("left join %s as ne on %s.id = ne.notification_id",
			models.NotificationEvent{}.TableName(), n)).
		Joins(fmt.Sprintf("left join %s as user on %s.creator = user.id",
			models.User{}.TableName(), n)).
		Where(fmt.Sprintf("%s.org_id = ?", n), orgId)
	if projectId != "" {
		query = query.Where(fmt.Sprintf("%s.project_id = ?", n), projectId)
	} else {
		query = query.Where(fmt.Sprintf("(%s.project_id = '' or %s.project_id is null)", n, n))
	}
	return query.LazySelectAppend(fmt.Sprintf("%s.*", n), "group_concat(ne.event_type) as event_type").
		LazySelectAppend("user.name as creator_name").
		Group(fmt.Sprintf("%s.id", n))
}

func SearchNotifyEventType(dbSess *db.Session, notifyId models.Id) ([]string, e.Error) {
	events := make([]string, 0)
	if err := dbSess.Table(models.NotificationEvent{}.TableName()).
		Where("notification_id = ?", notifyId).
		Pluck("event_type", &events); err != nil {
		return nil, e.New(e.DBError, err)
	}

	return events, nil
}

func UpdateNotification(tx *db.Session, id models.Id, orgId models.Id, projectId models.Id, attrs models.Attrs) (notificationCfg *models.Notification, err e.Error) {
	query := tx.Where("id = ? and org_id = ?", id, orgId)
	if projectId != "" {
		query = query.Where("project_id = ?", projectId)
	} else {
		query = query.Where("(project_id = '' or project_id is null)")
	}
	if err := query.First(&notificationCfg); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, fmt.Errorf("query notification cfg error: %v", err))
	}
	if len(attrs) > 0 {
		query = tx.Where("id = ? and org_id = ?", id, orgId)
		if projectId != "" {
			query = query.Where("project_id = ?", projectId)
		} else {
			query = query.Where("(project_id = '' or project_id is null)")
		}
		if _, err := models.UpdateAttr(query, &models.Notification{}, attrs); err != nil {
			return nil, e.New(e.DBError, fmt.Errorf("update notification cfg error: %v", err))
		}
	}
	query = tx.Where("id = ? and org_id = ?", id, orgId)
	if projectId != "" {
		query = query.Where("project_id = ?", projectId)
	} else {
		query = query.Where("(project_id = '' or project_id is null)")
	}
	if err := query.First(&notificationCfg); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, fmt.Errorf("query notification cfg error: %v", err))
	}
	return
}

func CreateNotification(tx *db.Session, notification models.Notification, eventType []string) (*models.Notification, e.Error) {
	if notification.Id == "" {
		notification.Id = models.NewId("notif-")
	}
	if err := models.Create(tx, &notification); err != nil {
		return nil, e.New(e.DBError, err)
	}

	events := make([]models.NotificationEvent, 0, len(eventType))
	for _, v := range eventType {
		events = append(events, models.NotificationEvent{
			NotificationId: notification.Id,
			EventType:      v,
		})
	}

	if err := tx.Insert(&events); err != nil {
		return nil, e.New(e.DBError, err)
	}

	return &notification, nil
}

func DeleteNotificationEvent(tx *db.Session, nId models.Id) e.Error {
	if _, err := tx.Where("notification_id = ?", nId).Delete(&models.NotificationEvent{}); err != nil {
		return e.New(e.DBError, err)
	}

	return nil
}

func DeleteNotification(tx *db.Session, id models.Id, orgId models.Id, projectId models.Id) e.Error {
	query := tx.Where("id = ? AND org_id = ?", id, orgId)
	if projectId != "" {
		query = query.Where("project_id = ?", projectId)
	} else {
		query = query.Where("(project_id = '' or project_id is null)")
	}
	affected, err := query.Delete(&models.Notification{})
	if err != nil {
		return e.New(e.DBError, fmt.Errorf("delete notification cfg error: %v", err))
	}
	if affected == 0 {
		return e.New(e.ObjectNotExistsOrNoPerm, fmt.Errorf("notification cfg %s not found", id))
	}

	if err := DeleteNotificationEvent(tx, id); err != nil {
		return e.New(e.DBError, fmt.Errorf("delete notification cfg error: %v", err))
	}

	return nil
}

func DetailNotification(dbSess *db.Session, id models.Id, orgId models.Id, projectId models.Id) (interface{}, e.Error) {
	resp := resps.RespDetailNotification{}
	n := models.Notification{}.TableName()
	query := dbSess.Table(n).
		Joins(fmt.Sprintf("left join %s as ne on %s.id = ne.notification_id",
			models.NotificationEvent{}.TableName(), n)).
		Where(fmt.Sprintf("%s.id = ? and %s.org_id = ?", n, n), id, orgId)
	if projectId != "" {
		query = query.Where(fmt.Sprintf("%s.project_id = ?", n), projectId)
	} else {
		query = query.Where(fmt.Sprintf("(%s.project_id = '' or %s.project_id is null)", n, n))
	}
	if err := query.
		LazySelectAppend(fmt.Sprintf("%s.*", n)).
		LazySelectAppend("group_concat(ne.event_type) as event_type").
		Group(fmt.Sprintf("%s.id", n)).
		First(&resp); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExistsOrNoPerm, err)
		}
		return nil, e.New(e.DBError, err)
	}
	resp.EventTypes = strings.Split(resp.EventType, ",")
	return resp, nil
}
