// Copyright (c) 2015-2023 CloudJ Technology Co., Ltd.

package services

import (
	"fmt"

	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/db"
	"cloudiac/portal/models"
)

func CreateCloudAccount(tx *db.Session, account *models.CloudAccount) (*models.CloudAccount, e.Error) {
	if err := models.Create(tx, account); err != nil {
		if e.IsDuplicate(err) {
			return nil, e.New(e.NameDuplicate, err)
		}
		return nil, e.New(e.DBError, err)
	}
	return account, nil
}

func UpdateCloudAccount(tx *db.Session, orgId, id models.Id, attrs models.Attrs) (*models.CloudAccount, e.Error) {
	if _, err := models.UpdateAttr(tx.Where("id = ? and org_id = ?", id, orgId), &models.CloudAccount{}, attrs); err != nil {
		if e.IsDuplicate(err) {
			return nil, e.New(e.NameDuplicate, err)
		}
		return nil, e.New(e.DBError, fmt.Errorf("update cloud account error: %v", err))
	}
	return GetCloudAccountById(tx, orgId, id)
}

func DeleteCloudAccount(tx *db.Session, orgId, id models.Id) e.Error {
	if _, err := tx.Where("id = ? and org_id = ?", id, orgId).Delete(&models.CloudAccount{}); err != nil {
		return e.New(e.DBError, fmt.Errorf("delete cloud account error: %v", err))
	}
	return nil
}

func GetCloudAccountById(tx *db.Session, orgId, id models.Id) (*models.CloudAccount, e.Error) {
	account := models.CloudAccount{}
	if err := tx.Where("id = ? and org_id = ?", id, orgId).First(&account); err != nil {
		if e.IsRecordNotFound(err) {
			return nil, e.New(e.ObjectNotExists, err)
		}
		return nil, e.New(e.DBError, err)
	}
	return &account, nil
}

func QueryCloudAccount(query *db.Session) *db.Session {
	return query.Model(&models.CloudAccount{})
}
