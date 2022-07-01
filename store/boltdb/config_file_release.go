/**
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package boltdb

import (
	"errors"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
)

const (
	tblConfigFileRelease   string = "ConfigFileRelease"
	tblConfigFileReleaseID string = "ConfigFileReleaseID"

	FileReleaseFieldId         string = "Id"
	FileReleaseFieldName       string = "Name"
	FileReleaseFieldNamespace  string = "Namespace"
	FileReleaseFieldGroup      string = "Group"
	FileReleaseFieldFileName   string = "FileName"
	FileReleaseFieldContent    string = "Content"
	FileReleaseFieldComment    string = "Comment"
	FileReleaseFieldMd5        string = "Md5"
	FileReleaseFieldVersion    string = "Version"
	FileReleaseFieldFlag       string = "Flag"
	FileReleaseFieldCreateTime string = "CreateTime"
	FileReleaseFieldCreateBy   string = "CreateBy"
	FileReleaseFieldModifyTime string = "ModifyTime"
	FileReleaseFieldModifyBy   string = "ModifyBy"
	FileReleaseFieldValid      string = "Valid"
)

var (
	ErrMultipleConfigFileReleaseFound error = errors.New("multiple config_file_release found")
)

type configFileReleaseStore struct {
	id      uint64
	handler BoltHandler
}

func newConfigFileReleaseStore(handler BoltHandler) (*configFileReleaseStore, error) {
	s := &configFileReleaseStore{handler: handler, id: 0}
	ret, err := handler.LoadValues(tblConfigFileReleaseID, []string{tblConfigFileReleaseID}, &IDHolder{})
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return s, nil
	}
	val := ret[tblConfigFileReleaseID].(*IDHolder)
	s.id = val.ID
	return s, nil
}

// CreateConfigFileRelease 新建配置文件发布
func (cfr *configFileReleaseStore) CreateConfigFileRelease(proxyTx store.Tx, fileRelease *model.ConfigFileRelease) (*model.ConfigFileRelease, error) {
	ret, err := DoTransactionIfNeed(proxyTx, cfr.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		cfr.id++
		fileRelease.Id = cfr.id
		fileRelease.Valid = true
		tN := time.Now()
		fileRelease.CreateTime = tN
		fileRelease.ModifyTime = tN

		if err := saveValue(tx, tblConfigFileReleaseID, tblConfigFileReleaseID, &IDHolder{
			ID: cfr.id,
		}); err != nil {
			log.Error("[ConfigFileRelease] save auto_increment id", zap.Error(err))
			return nil, err
		}

		key := fmt.Sprintf("%s@@%s@@%s", fileRelease.Namespace, fileRelease.Group, fileRelease.FileName)
		if err := saveValue(tx, tblConfigFileRelease, key, fileRelease); err != nil {
			log.Error("[ConfigFileRelease] save info", zap.Error(err))
			return nil, err
		}

		data, err := cfr.getConfigFileReleaseByFlag(tx, fileRelease.Namespace, fileRelease.Group, fileRelease.FileName, false)
		if err != nil {
			return nil, err
		}

		if data == nil {
			return nil, nil
		}

		return []interface{}{data}, nil
	})

	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, nil
	}

	return ret[0].(*model.ConfigFileRelease), nil
}

// UpdateConfigFileRelease 更新配置文件发布
func (cfr *configFileReleaseStore) UpdateConfigFileRelease(proxyTx store.Tx, fileRelease *model.ConfigFileRelease) (*model.ConfigFileRelease, error) {
	ret, err := DoTransactionIfNeed(proxyTx, cfr.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		properties := make(map[string]interface{})

		properties[FileReleaseFieldName] = fileRelease.Name
		properties[FileReleaseFieldContent] = fileRelease.Content
		properties[FileReleaseFieldComment] = fileRelease.Comment
		properties[FileReleaseFieldMd5] = fileRelease.Md5
		properties[FileReleaseFieldVersion] = fileRelease.Version
		properties[FileReleaseFieldValid] = true
		properties[FileReleaseFieldFlag] = 0
		properties[FileReleaseFieldModifyTime] = time.Now()
		properties[FileReleaseFieldModifyBy] = fileRelease.ModifyBy

		key := fmt.Sprintf("%s@@%s@@%s", fileRelease.Namespace, fileRelease.Group, fileRelease.FileName)
		if err := updateValue(tx, tblConfigFileRelease, key, properties); err != nil {
			log.Error("[ConfigFileRelease] update info", zap.Error(err))
			return nil, err
		}

		data, err := cfr.getConfigFileReleaseByFlag(tx, fileRelease.Namespace, fileRelease.Group, fileRelease.FileName, false)
		if err != nil {
			return nil, err
		}

		if data == nil {
			return nil, nil
		}

		return []interface{}{data}, nil
	})

	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, nil
	}

	return ret[0].(*model.ConfigFileRelease), nil
}

// GetConfigFileRelease Get the configuration file release, only the record of FLAG = 0
func (cfr *configFileReleaseStore) GetConfigFileRelease(proxyTx store.Tx, namespace, group, fileName string) (*model.ConfigFileRelease, error) {
	ret, err := DoTransactionIfNeed(proxyTx, cfr.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		data, err := cfr.getConfigFileReleaseByFlag(tx, namespace, group, fileName, false)
		if err != nil {
			return nil, err
		}

		if data == nil {
			return nil, nil
		}

		return []interface{}{data}, nil
	})
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, nil
	}

	return ret[0].(*model.ConfigFileRelease), nil
}

// GetConfigFileReleaseWithAllFlag Get all publishing data, including deletion
func (cfr *configFileReleaseStore) GetConfigFileReleaseWithAllFlag(proxyTx store.Tx, namespace, group, fileName string) (*model.ConfigFileRelease, error) {
	ret, err := DoTransactionIfNeed(proxyTx, cfr.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		data, err := cfr.getConfigFileReleaseByFlag(tx, namespace, group, fileName, true)
		if err != nil {
			return nil, err
		}

		if data == nil {
			return nil, nil
		}

		return []interface{}{data}, nil
	})
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, nil
	}

	return ret[0].(*model.ConfigFileRelease), nil
}

// getConfigFileReleaseByFlag Obtain data through FLAG
func (cfr *configFileReleaseStore) getConfigFileReleaseByFlag(tx *bolt.Tx, namespace, group, fileName string, withAllFlag bool) (*model.ConfigFileRelease, error) {
	key := fmt.Sprintf("%s@@%s@@%s", namespace, group, fileName)

	ret := make(map[string]interface{})

	if err := loadValues(tx, tblConfigFileRelease, []string{key}, &model.ConfigFileRelease{}, ret); err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, nil
	}
	if len(ret) > 1 {
		return nil, ErrMultipleConfigFileReleaseFound
	}

	var release *model.ConfigFileRelease
	for _, v := range ret {
		release = v.(*model.ConfigFileRelease)
	}

	if !withAllFlag && !release.Valid {
		return nil, nil
	}

	return release, nil
}

// DeleteConfigFileRelease Delete the release data
func (cfr *configFileReleaseStore) DeleteConfigFileRelease(proxyTx store.Tx, namespace, group, fileName, deleteBy string) error {
	_, err := DoTransactionIfNeed(proxyTx, cfr.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		release, err := cfr.getConfigFileReleaseByFlag(tx, namespace, group, fileName, false)
		if err != nil {
			return nil, err
		}

		properties := make(map[string]interface{})

		properties[FileReleaseFieldMd5] = ""
		properties[FileReleaseFieldVersion] = release.Version + 1
		properties[FileReleaseFieldValid] = false
		properties[FileReleaseFieldFlag] = 1
		properties[FileReleaseFieldModifyTime] = time.Now()
		properties[FileReleaseFieldModifyBy] = deleteBy

		key := fmt.Sprintf("%s@@%s@@%s", namespace, group, fileName)
		if err := updateValue(tx, tblConfigFileRelease, key, properties); err != nil {
			log.Error("[ConfigFileRelease] delete info", zap.Error(err))
			return nil, err
		}

		return nil, nil
	})

	return err
}

// FindConfigFileReleaseByModifyTimeAfter Get the last update time more than a certain time point,
//    pay attention to containing Flag = 1, in order to get the deleted Release
func (cfr *configFileReleaseStore) FindConfigFileReleaseByModifyTimeAfter(modifyTime time.Time) ([]*model.ConfigFileRelease, error) {

	fields := []string{FileReleaseFieldModifyTime}

	ret, err := cfr.handler.LoadValuesByFilter(tblConfigFileRelease, fields, &model.ConfigFileRelease{},
		func(m map[string]interface{}) bool {
			saveMt, _ := m[FileReleaseFieldModifyTime].(time.Time)
			return !saveMt.Before(modifyTime)
		})

	if err != nil {
		return nil, err
	}

	releases := make([]*model.ConfigFileRelease, 0, len(ret))

	for _, v := range ret {
		releases = append(releases, v.(*model.ConfigFileRelease))
	}

	return releases, nil
}
