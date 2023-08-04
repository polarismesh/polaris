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
	"time"

	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

var _ store.ConfigFileReleaseStore = (*configFileReleaseStore)(nil)

const (
	tblConfigFileRelease string = "ConfigFileRelease"

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
	FileReleaseFieldActive     string = "Active"
	FileReleaseFieldMetadata   string = "Metadata"
)

var (
	ErrMultipleConfigFileReleaseFound error = errors.New("multiple config_file_release found")
)

type configFileReleaseStore struct {
	handler BoltHandler
}

func newConfigFileReleaseStore(handler BoltHandler) (*configFileReleaseStore, error) {
	s := &configFileReleaseStore{handler: handler}
	return s, nil
}

// CreateConfigFileRelease 新建配置文件发布
func (cfr *configFileReleaseStore) CreateConfigFileReleaseTx(proxyTx store.Tx,
	fileRelease *model.ConfigFileRelease) error {
	_, err := DoTransactionIfNeed(proxyTx, cfr.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		// 是否存在当前 release
		values := map[string]interface{}{}
		if err := loadValues(tx, tblConfigFileRelease, []string{fileRelease.ReleaseKey()},
			&model.ConfigFileRelease{}, values); err != nil {
			return nil, err
		}
		if len(values) != 0 {
			return nil, store.NewStatusError(store.DuplicateEntryErr, "exist record")
		}

		table := tx.Bucket([]byte(tblConfigFileRelease))
		nextId, err := table.NextSequence()
		if err != nil {
			return nil, err
		}
		fileRelease.Id = nextId
		fileRelease.Valid = true
		tN := time.Now()
		fileRelease.CreateTime = tN
		fileRelease.ModifyTime = tN

		maxVersion, err := cfr.inactiveConfigFileRelease(tx, fileRelease)
		if err != nil {
			return nil, err
		}

		fileRelease.Active = true
		fileRelease.Version = maxVersion + 1
		if err := saveValue(tx, tblConfigFileRelease, fileRelease.ReleaseKey(), fileRelease); err != nil {
			log.Error("[ConfigFileRelease] save info", zap.Error(err))
			return nil, err
		}
		return nil, nil
	})
	return err
}

// GetConfigFileActiveRelease Get the configuration file release, only the record of FLAG = 0
func (cfr *configFileReleaseStore) GetConfigFileActiveRelease(namespace,
	group, fileName string) (*model.ConfigFileRelease, error) {
	fields := []string{FileReleaseFieldNamespace, FileReleaseFieldGroup, FileReleaseFieldFileName,
		FileReleaseFieldActive, FileReleaseFieldFlag}

	// 查询这个 release 相关的所有
	values, err := cfr.handler.LoadValuesByFilter(tblConfigFileRelease, fields, &model.ConfigFileRelease{},
		func(m map[string]interface{}) bool {
			flag, _ := m[FileReleaseFieldFlag].(int)
			// 已经删除的不管
			if flag == 1 {
				return false
			}
			active, _ := m[FileReleaseFieldActive].(bool)
			if !active {
				return false
			}
			saveNs, _ := m[FileReleaseFieldNamespace].(string)
			saveGroup, _ := m[FileReleaseFieldGroup].(string)
			saveFileName, _ := m[FileReleaseFieldFileName].(string)

			expect := saveNs == namespace && saveGroup == group && saveFileName == fileName
			return expect
		})
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		return v.(*model.ConfigFileRelease), nil
	}
	return nil, nil
}

// GetConfigFileRelease Get the configuration file release, only the record of FLAG = 0
func (cfr *configFileReleaseStore) GetConfigFileRelease(args *model.ConfigFileReleaseKey) (*model.ConfigFileRelease, error) {

	values, err := cfr.handler.LoadValues(tblConfigFileRelease, []string{args.ReleaseKey()},
		&model.ConfigFileRelease{})
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		return v.(*model.ConfigFileRelease), nil
	}
	return nil, nil
}

// DeleteConfigFileRelease Delete the release data
func (cfr *configFileReleaseStore) DeleteConfigFileRelease(data *model.ConfigFileReleaseKey) error {
	_, err := DoTransactionIfNeed(nil, cfr.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		properties := make(map[string]interface{})

		properties[FileReleaseFieldValid] = false
		properties[FileReleaseFieldFlag] = 1
		properties[FileReleaseFieldModifyTime] = time.Now()
		if err := updateValue(tx, tblConfigFileRelease, data.ReleaseKey(), properties); err != nil {
			log.Error("[ConfigFileRelease] delete info", zap.Error(err))
			return nil, err
		}
		return nil, nil
	})
	return err
}

// CountConfigReleases count the release data
func (cfr *configFileReleaseStore) CountConfigReleases(namespace, group string) (uint64, error) {
	fields := []string{FileReleaseFieldNamespace, FileReleaseFieldGroup, FileReleaseFieldValid}
	ret, err := cfr.handler.LoadValuesByFilter(tblConfigFileRelease, fields, &model.ConfigFileRelease{},
		func(m map[string]interface{}) bool {
			valid, _ := m[FileReleaseFieldValid].(bool)
			if !valid {
				return false
			}
			saveNs, _ := m[FileReleaseFieldNamespace].(string)
			saveGroup, _ := m[FileReleaseFieldNamespace].(string)
			return saveNs == namespace && saveGroup == group
		})
	if err != nil {
		return 0, err
	}
	return uint64(len(ret)), err
}

// CleanConfigFileReleasesTx
func (cfr *configFileReleaseStore) CleanConfigFileReleasesTx(tx store.Tx, namespace, group, fileName string) error {
	dbTx := tx.GetDelegateTx().(*bolt.Tx)

	fields := []string{FileReleaseFieldNamespace, FileReleaseFieldGroup, FileReleaseFieldFileName,
		FileReleaseFieldValid}
	values := map[string]interface{}{}
	err := loadValuesByFilter(dbTx, tblConfigFileRelease, fields, &model.ConfigFileRelease{},
		func(m map[string]interface{}) bool {
			flag, _ := m[FileReleaseFieldValid].(int)
			// 已经删除的不管
			if flag == 1 {
				return false
			}
			saveNs, _ := m[FileReleaseFieldNamespace].(string)
			saveGroup, _ := m[FileReleaseFieldGroup].(string)
			saveFileName, _ := m[FileReleaseFieldFileName].(string)

			expect := saveNs == namespace && saveGroup == group && saveFileName == fileName
			return expect
		}, values)

	properties := map[string]interface{}{
		FileReleaseFieldFlag:       1,
		FileReleaseFieldValid:      false,
		FileReleaseFieldModifyTime: time.Now(),
	}
	for key := range values {
		if err := updateValue(dbTx, tblConfigFileRelease, key, properties); err != nil {
			return nil
		}
	}

	return err
}

// GetMoreReleaseFile Get the last update time more than a certain time point
// pay attention to containing Flag = 1, in order to get the deleted Release
func (cfr *configFileReleaseStore) GetMoreReleaseFile(firstUpdate bool,
	modifyTime time.Time) ([]*model.ConfigFileRelease, error) {

	if firstUpdate {
		modifyTime = time.Time{}
	}

	fields := []string{FileReleaseFieldModifyTime, FileReleaseFieldActive}
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

func (cfr *configFileReleaseStore) ActiveConfigFileRelease(release *model.ConfigFileRelease) error {
	return cfr.handler.Execute(true, func(tx *bolt.Tx) error {
		maxVersion, err := cfr.inactiveConfigFileRelease(tx, release)
		if err != nil {
			return err
		}
		properties := make(map[string]interface{})
		properties[FileReleaseFieldVersion] = maxVersion + 1
		properties[FileReleaseFieldActive] = true
		properties[FileReleaseFieldModifyTime] = time.Now()
		return updateValue(tx, tblConfigFileRelease, release.ReleaseKey(), properties)
	})
}

func (cfr *configFileReleaseStore) inactiveConfigFileRelease(tx *bolt.Tx,
	release *model.ConfigFileRelease) (uint64, error) {

	fields := []string{FileReleaseFieldNamespace, FileReleaseFieldGroup, FileReleaseFieldFileName,
		FileReleaseFieldVersion, FileReleaseFieldFlag}

	values := map[string]interface{}{}
	var maxVersion uint64
	// 查询这个 release 相关的所有
	if err := loadValuesByFilter(tx, tblConfigFileRelease, fields, &model.ConfigFileRelease{},
		func(m map[string]interface{}) bool {
			flag, _ := m[FileReleaseFieldFlag].(int)
			// 已经删除的不管
			if flag == 1 {
				return false
			}
			saveNs, _ := m[FileReleaseFieldNamespace].(string)
			saveGroup, _ := m[FileReleaseFieldGroup].(string)
			saveFileName, _ := m[FileReleaseFieldFileName].(string)

			expect := saveNs == release.Namespace && saveGroup == release.Group && saveFileName == release.FileName
			if expect {
				saveVersion, _ := m[FileReleaseFieldVersion].(uint64)
				if saveVersion > maxVersion {
					maxVersion = saveVersion
				}
			}
			return expect
		}, values); err != nil {
		return 0, err
	}
	properties := map[string]interface{}{
		FileReleaseFieldActive:     false,
		FileReleaseFieldModifyTime: time.Now(),
	}
	for key := range values {
		if err := updateValue(tx, tblConfigFileRelease, key, properties); err != nil {
			return 0, err
		}
	}
	return maxVersion, nil
}

// CleanDeletedConfigFileRelease 清理配置发布历史
func (cfr *configFileReleaseStore) CleanDeletedConfigFileRelease(endTime time.Time, limit uint64) error {

	fields := []string{FileReleaseFieldModifyTime}
	needDel, err := cfr.handler.LoadValuesByFilter(tblConfigFileRelease, fields,
		&model.ConfigFileRelease{}, func(m map[string]interface{}) bool {
			saveModifyTime, _ := m[FileReleaseFieldModifyTime].(time.Time)
			return endTime.After(saveModifyTime)
		})
	if err != nil {
		return err
	}

	keys := make([]string, 0, len(needDel))
	for i := range needDel {
		keys = append(keys, i)
	}
	return cfr.handler.DeleteValues(tblConfigFileRelease, keys)
}
