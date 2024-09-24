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
	"sort"
	"time"

	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
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
	FileReleaseFieldType       string = "Typ"
)

var (
	ErrMultipleConfigFileReleaseFound error = errors.New("multiple config_file_release found")
)

type configFileReleaseStore struct {
	handler BoltHandler
}

func newConfigFileReleaseStore(handler BoltHandler) *configFileReleaseStore {
	s := &configFileReleaseStore{handler: handler}
	return s
}

// CreateConfigFileReleaseTx 新建配置文件发布
func (cfr *configFileReleaseStore) CreateConfigFileReleaseTx(proxyTx store.Tx,
	fileRelease *model.ConfigFileRelease) error {
	tx := proxyTx.GetDelegateTx().(*bolt.Tx)
	// 是否存在当前 release
	values := map[string]interface{}{}
	if err := loadValues(tx, tblConfigFileRelease, []string{fileRelease.ReleaseKey()},
		&ConfigFileRelease{}, values); err != nil {
		return err
	}
	for i := range values {
		if ret := cfr.toValisModelData(values[i].(*ConfigFileRelease)); ret != nil {
			return store.NewStatusError(store.DuplicateEntryErr, "exist record")
		}
	}

	table, err := tx.CreateBucketIfNotExists([]byte(tblConfigFileRelease))
	if err != nil {
		return store.Error(err)
	}
	nextId, err := table.NextSequence()
	if err != nil {
		return store.Error(err)
	}
	fileRelease.Id = nextId
	fileRelease.Valid = true
	tN := time.Now()
	fileRelease.CreateTime = tN
	fileRelease.ModifyTime = tN

	maxVersion, err := cfr.inactiveConfigFileRelease(tx, fileRelease)
	if err != nil {
		return store.Error(err)
	}

	fileRelease.Active = true
	fileRelease.Version = maxVersion + 1

	log.Debug("[ConfigFileRelease] cur release version", utils.ZapNamespace(fileRelease.Namespace),
		utils.ZapGroup(fileRelease.Group), utils.ZapFileName(fileRelease.FileName), utils.ZapVersion(fileRelease.Version))

	err = saveValue(tx, tblConfigFileRelease, fileRelease.ReleaseKey(), cfr.toStoreData(fileRelease))
	if err != nil {
		log.Error("[ConfigFileRelease] save info", zap.Error(err))
		return store.Error(err)
	}
	return nil
}

// GetConfigFileRelease Get the configuration file release, only the record of FLAG = 0
func (cfr *configFileReleaseStore) GetConfigFileRelease(args *model.ConfigFileReleaseKey) (*model.ConfigFileRelease, error) {

	values, err := cfr.handler.LoadValues(tblConfigFileRelease, []string{args.ReleaseKey()},
		&ConfigFileRelease{})
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		return cfr.toValisModelData(v.(*ConfigFileRelease)), nil
	}
	return nil, nil
}

// GetConfigFileRelease Get the configuration file release, only the record of FLAG = 0
func (cfr *configFileReleaseStore) GetConfigFileReleaseTx(tx store.Tx,
	args *model.ConfigFileReleaseKey) (*model.ConfigFileRelease, error) {
	dbTx := tx.GetDelegateTx().(*bolt.Tx)
	values := make(map[string]interface{}, 1)
	err := loadValues(dbTx, tblConfigFileRelease, []string{args.ReleaseKey()},
		&ConfigFileRelease{}, values)
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		return cfr.toValisModelData(v.(*ConfigFileRelease)), nil
	}
	return nil, nil
}

// GetConfigFileActiveRelease .
func (cfr *configFileReleaseStore) GetConfigFileActiveRelease(file *model.ConfigFileKey) (*model.ConfigFileRelease, error) {
	tx, err := cfr.handler.StartTx()
	if err != nil {
		return nil, store.Error(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	return cfr.GetConfigFileActiveReleaseTx(tx, file)
}

func (cfr *configFileReleaseStore) GetConfigFileActiveReleaseTx(tx store.Tx,
	file *model.ConfigFileKey) (*model.ConfigFileRelease, error) {
	dbTx := tx.GetDelegateTx().(*bolt.Tx)

	fields := []string{FileReleaseFieldActive, FileReleaseFieldNamespace, FileReleaseFieldGroup,
		FileReleaseFieldFileName, FileReleaseFieldValid, FileReleaseFieldType}
	values := make(map[string]interface{}, 1)
	err := loadValuesByFilter(dbTx, tblConfigFileRelease, fields, &ConfigFileRelease{},
		func(m map[string]interface{}) bool {
			valid, _ := m[FileReleaseFieldValid].(bool)
			// 已经删除的不管
			if !valid {
				return false
			}
			active, _ := m[FileReleaseFieldActive].(bool)
			if !active {
				return false
			}
			if relType, _ := m[FileReleaseFieldType].(string); relType != model.ReleaseTypeFull {
				return false
			}
			saveNs, _ := m[FileReleaseFieldNamespace].(string)
			saveGroup, _ := m[FileReleaseFieldGroup].(string)
			saveFileName, _ := m[FileReleaseFieldFileName].(string)

			expect := saveNs == file.Namespace && saveGroup == file.Group && saveFileName == file.Name
			return expect
		}, values)
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		return cfr.toValisModelData(v.(*ConfigFileRelease)), nil
	}
	return nil, nil
}

func (cfr *configFileReleaseStore) GetConfigFileBetaReleaseTx(tx store.Tx,
	file *model.ConfigFileKey) (*model.ConfigFileRelease, error) {
	dbTx := tx.GetDelegateTx().(*bolt.Tx)

	fields := []string{FileReleaseFieldActive, FileReleaseFieldNamespace, FileReleaseFieldGroup,
		FileReleaseFieldFileName, FileReleaseFieldValid, FileReleaseFieldType}
	values := make(map[string]interface{}, 1)
	err := loadValuesByFilter(dbTx, tblConfigFileRelease, fields, &ConfigFileRelease{},
		func(m map[string]interface{}) bool {
			valid, _ := m[FileReleaseFieldValid].(bool)
			// 已经删除的不管
			if !valid {
				return false
			}
			active, _ := m[FileReleaseFieldActive].(bool)
			if !active {
				return false
			}
			if relType, _ := m[FileReleaseFieldType].(string); relType != model.ReleaseTypeGray {
				return false
			}
			saveNs, _ := m[FileReleaseFieldNamespace].(string)
			saveGroup, _ := m[FileReleaseFieldGroup].(string)
			saveFileName, _ := m[FileReleaseFieldFileName].(string)

			expect := saveNs == file.Namespace && saveGroup == file.Group && saveFileName == file.Name
			return expect
		}, values)
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		return cfr.toValisModelData(v.(*ConfigFileRelease)), nil
	}
	return nil, nil
}

// DeleteConfigFileRelease Delete the release data
func (cfr *configFileReleaseStore) DeleteConfigFileReleaseTx(tx store.Tx, data *model.ConfigFileReleaseKey) error {
	dbTx := tx.GetDelegateTx().(*bolt.Tx)
	properties := make(map[string]interface{})

	properties[FileReleaseFieldValid] = false
	properties[FileReleaseFieldFlag] = 1
	properties[FileReleaseFieldModifyTime] = time.Now()
	if err := updateValue(dbTx, tblConfigFileRelease, data.ReleaseKey(), properties); err != nil {
		log.Error("[ConfigFileRelease] delete info", zap.Error(err))
		return store.Error(err)
	}
	return nil
}

// CountConfigReleases count the release data
func (cfr *configFileReleaseStore) CountConfigReleases(namespace, group string, onlyActive bool) (uint64, error) {
	fields := []string{FileReleaseFieldNamespace, FileReleaseFieldGroup, FileReleaseFieldValid, FileReleaseFieldActive}
	ret, err := cfr.handler.LoadValuesByFilter(tblConfigFileRelease, fields, &ConfigFileRelease{},
		func(m map[string]interface{}) bool {
			valid, _ := m[FileReleaseFieldValid].(bool)
			if !valid {
				return false
			}
			if onlyActive {
				active, _ := m[FileReleaseFieldActive].(bool)
				if !active {
					return false
				}
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
	err := loadValuesByFilter(dbTx, tblConfigFileRelease, fields, &ConfigFileRelease{},
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
	if err != nil {
		return err
	}
	properties := map[string]interface{}{
		FileReleaseFieldFlag:       1,
		FileReleaseFieldValid:      false,
		FileReleaseFieldModifyTime: time.Now(),
	}
	for key := range values {
		if err := updateValue(dbTx, tblConfigFileRelease, key, properties); err != nil {
			return err
		}
	}
	return nil
}

// GetMoreReleaseFile Get the last update time more than a certain time point
// pay attention to containing Flag = 1, in order to get the deleted Release
func (cfr *configFileReleaseStore) GetMoreReleaseFile(firstUpdate bool,
	modifyTime time.Time) ([]*model.ConfigFileRelease, error) {

	if firstUpdate {
		modifyTime = time.Unix(0, 0)
	}

	fields := []string{FileReleaseFieldModifyTime}
	ret, err := cfr.handler.LoadValuesByFilter(tblConfigFileRelease, fields, &ConfigFileRelease{},
		func(m map[string]interface{}) bool {
			saveMt, _ := m[FileReleaseFieldModifyTime].(time.Time)
			return !saveMt.Before(modifyTime)
		})

	if err != nil {
		return nil, err
	}

	releases := make([]*model.ConfigFileRelease, 0, len(ret))
	for _, v := range ret {
		releases = append(releases, cfr.toModelData(v.(*ConfigFileRelease)))
	}
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Version > releases[j].Version
	})
	return releases, nil
}

func (cfr *configFileReleaseStore) ActiveConfigFileReleaseTx(tx store.Tx, release *model.ConfigFileRelease) error {
	dbTx := tx.GetDelegateTx().(*bolt.Tx)
	maxVersion, err := cfr.inactiveConfigFileRelease(dbTx, release)
	if err != nil {
		return err
	}
	properties := make(map[string]interface{})
	properties[FileReleaseFieldVersion] = maxVersion + 1
	properties[FileReleaseFieldActive] = true
	properties[FileReleaseFieldModifyTime] = time.Now()
	return updateValue(dbTx, tblConfigFileRelease, release.ReleaseKey(), properties)
}

func (cfr *configFileReleaseStore) InactiveConfigFileReleaseTx(tx store.Tx, release *model.ConfigFileRelease) error {
	dbTx := tx.GetDelegateTx().(*bolt.Tx)
	properties := make(map[string]interface{})
	properties[FileReleaseFieldActive] = false
	properties[FileReleaseFieldModifyTime] = time.Now()
	return updateValue(dbTx, tblConfigFileRelease, release.ReleaseKey(), properties)
}

func (cfr *configFileReleaseStore) inactiveConfigFileRelease(tx *bolt.Tx,
	release *model.ConfigFileRelease) (uint64, error) {

	fields := []string{FileReleaseFieldNamespace, FileReleaseFieldGroup, FileReleaseFieldFileName,
		FileReleaseFieldVersion, FileReleaseFieldValid, FileReleaseFieldActive, FileReleaseFieldType}

	values := map[string]interface{}{}
	var maxVersion uint64
	// 查询这个 release 相关的所有
	if err := loadValuesByFilter(tx, tblConfigFileRelease, fields, &ConfigFileRelease{},
		func(m map[string]interface{}) bool {
			// 已经删除的不管
			if valid, _ := m[FileReleaseFieldValid].(bool); !valid {
				return false
			}
			isActive, _ := m[FileReleaseFieldActive].(bool)
			if !isActive {
				return false
			}
			saveNs, _ := m[FileReleaseFieldNamespace].(string)
			saveGroup, _ := m[FileReleaseFieldGroup].(string)
			saveFileName, _ := m[FileReleaseFieldFileName].(string)
			releaseType, _ := m[FileReleaseFieldType].(string)
			if releaseType != string(release.ReleaseType) {
				return false
			}

			expect := saveNs == release.Namespace && saveGroup == release.Group &&
				saveFileName == release.FileName
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
		&ConfigFileRelease{}, func(m map[string]interface{}) bool {
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

type ConfigFileRelease struct {
	Id         uint64
	Name       string
	Namespace  string
	Group      string
	FileName   string
	Version    uint64
	Comment    string
	Md5        string
	Flag       int
	Active     bool
	Valid      bool
	Format     string
	Metadata   map[string]string
	CreateTime time.Time
	CreateBy   string
	ModifyTime time.Time
	ModifyBy   string
	Content    string
	Typ        string
}

func (cfr *configFileReleaseStore) toValisModelData(data *ConfigFileRelease) *model.ConfigFileRelease {
	saveData := cfr.toModelData(data)
	if !saveData.Valid {
		return nil
	}
	return saveData
}

func (cfr *configFileReleaseStore) toModelData(data *ConfigFileRelease) *model.ConfigFileRelease {
	return &model.ConfigFileRelease{
		SimpleConfigFileRelease: &model.SimpleConfigFileRelease{
			ConfigFileReleaseKey: &model.ConfigFileReleaseKey{
				Id:          data.Id,
				Name:        data.Name,
				Namespace:   data.Namespace,
				Group:       data.Group,
				FileName:    data.FileName,
				ReleaseType: model.ReleaseType(data.Typ),
			},
			Comment:    data.Comment,
			Md5:        data.Md5,
			Active:     data.Active,
			Valid:      data.Valid,
			Flag:       data.Flag,
			Format:     data.Format,
			Metadata:   data.Metadata,
			Version:    data.Version,
			CreateTime: data.CreateTime,
			CreateBy:   data.CreateBy,
			ModifyTime: data.ModifyTime,
			ModifyBy:   data.ModifyBy,
		},
		Content: data.Content,
	}
}

func (cfr *configFileReleaseStore) toStoreData(data *model.ConfigFileRelease) *ConfigFileRelease {
	return &ConfigFileRelease{
		Id:         data.Id,
		Name:       data.Name,
		Namespace:  data.Namespace,
		Group:      data.Group,
		FileName:   data.FileName,
		Version:    data.Version,
		Comment:    data.Comment,
		Md5:        data.Md5,
		Flag:       data.Flag,
		Active:     data.Active,
		Valid:      data.Valid,
		Format:     data.Format,
		Metadata:   data.Metadata,
		CreateTime: data.CreateTime,
		CreateBy:   data.CreateBy,
		ModifyTime: data.ModifyTime,
		ModifyBy:   data.ModifyBy,
		Content:    data.Content,
		Typ:        string(data.ConfigFileReleaseKey.ReleaseType),
	}
}
