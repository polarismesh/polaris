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
	tblConfigFileRelease       string = "ConfigFileRelease"
	tblConfigFileReleaseID     string = "ConfigFileReleaseID"
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
	MultipleConfigFileReleaseFound error = errors.New("multiple config_file_release found")
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
	var err error

	if proxyTx == nil {
		proxyTx, err = cfr.handler.StartTx()
		if err != nil {
			return nil, err
		}
	}

	tx := proxyTx.GetDelegateTx().(*bolt.Tx)
	defer tx.Rollback()

	cfr.id++
	fileRelease.Id = cfr.id
	fileRelease.Valid = true

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

	if err := tx.Commit(); err != nil {
		log.Error("[ConfigFileRelease] create do tx commit", zap.Error(err))
		return nil, err
	}
	return cfr.GetConfigFileRelease(proxyTx, fileRelease.Namespace, fileRelease.Group, fileRelease.FileName)
}

// UpdateConfigFileRelease 更新配置文件发布
func (cfr *configFileReleaseStore) UpdateConfigFileRelease(proxyTx store.Tx, fileRelease *model.ConfigFileRelease) (*model.ConfigFileRelease, error) {
	var err error
	if proxyTx == nil {
		proxyTx, err = cfr.handler.StartTx()
		if err != nil {
			return nil, err
		}
	}

	tx := proxyTx.GetDelegateTx().(*bolt.Tx)
	defer tx.Rollback()

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

	if err := tx.Commit(); err != nil {
		log.Error("[ConfigFileRelease] update do tx commit", zap.Error(err))
		return nil, err
	}

	return cfr.GetConfigFileRelease(proxyTx, fileRelease.Namespace, fileRelease.Group, fileRelease.FileName)
}

// GetConfigFileRelease 获取配置文件发布，只返回 flag=0 的记录
func (cfr *configFileReleaseStore) GetConfigFileRelease(tx store.Tx, namespace, group, fileName string) (*model.ConfigFileRelease, error) {
	return cfr.getConfigFileReleaseByFlag(tx, namespace, group, fileName, false)
}

// GetConfigFileReleaseWithAllFlag 获取所有发布数据，包含删除的
func (cfr *configFileReleaseStore) GetConfigFileReleaseWithAllFlag(tx store.Tx, namespace, group, fileName string) (*model.ConfigFileRelease, error) {
	return cfr.getConfigFileReleaseByFlag(tx, namespace, group, fileName, true)
}

// getConfigFileReleaseByFlag 通过 flag 获取发布数据
func (cfr *configFileReleaseStore) getConfigFileReleaseByFlag(proxyTx store.Tx, namespace, group, fileName string, withAllFlag bool) (*model.ConfigFileRelease, error) {
	var err error
	if proxyTx == nil {
		proxyTx, err = cfr.handler.StartTx()
		if err != nil {
			return nil, err
		}
	}

	tx := proxyTx.GetDelegateTx().(*bolt.Tx)
	defer tx.Rollback()

	key := fmt.Sprintf("%s@@%s@@%s", namespace, group, fileName)

	ret := make(map[string]interface{})

	if err := loadValues(tx, tblConfigFileRelease, []string{key}, &model.ConfigFileRelease{}, ret); err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, nil
	}
	if len(ret) > 1 {
		return nil, MultipleConfigFileReleaseFound
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

// DeleteConfigFileRelease 删除发布数据
func (cfr *configFileReleaseStore) DeleteConfigFileRelease(proxyTx store.Tx, namespace, group, fileName, deleteBy string) error {
	var err error
	if proxyTx == nil {
		proxyTx, err = cfr.handler.StartTx()
		if err != nil {
			return err
		}
	}

	tx := proxyTx.GetDelegateTx().(*bolt.Tx)
	defer tx.Rollback()

	release, err := cfr.getConfigFileReleaseByFlag(proxyTx, namespace, group, fileName, false)
	if err != nil {
		return err
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
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error("[ConfigFileRelease] delete do tx commit", zap.Error(err))
		return err
	}

	return nil
}

// FindConfigFileReleaseByModifyTimeAfter 获取最后更新时间大于某个时间点的发布，注意包含 flag = 1 的，为了能够获取被删除的 release
func (cfr *configFileReleaseStore) FindConfigFileReleaseByModifyTimeAfter(modifyTime time.Time) ([]*model.ConfigFileRelease, error) {

	fields := []string{FileReleaseFieldModifyTime}

	ret, err := cfr.handler.LoadValuesByFilter(tblConfigFileRelease, fields, &model.ConfigFileRelease{}, func(m map[string]interface{}) bool {
		saveMt, _ := m[FileReleaseFieldModifyTime].(time.Time)
		return saveMt.After(modifyTime)
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
