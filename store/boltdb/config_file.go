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
	"sort"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var _ store.ConfigFileStore = (*configFileStore)(nil)

const (
	tblConfigFile   string = "ConfigFile"
	tblConfigFileID string = "ConfigFileID"

	FileFieldId          string = "Id"
	FileFieldName        string = "Name"
	FileFieldNamespace   string = "Namespace"
	FileFieldGroup       string = "Group"
	FileFieldContent     string = "Content"
	FileFieldComment     string = "Comment"
	FileFieldFormat      string = "Format"
	FileFieldFlag        string = "Flag"
	FileFieldCreateTime  string = "CreateTime"
	FileFieldCreateBy    string = "CreateBy"
	FileFieldModifyTime  string = "ModifyTime"
	FileFieldModifyBy    string = "ModifyBy"
	FileFieldValid       string = "Valid"
	FileFieldMetadata    string = "Metadata"
	FileFieldEncrypt     string = "Encrypt"
	FileFieldEncryptAlgo string = "EncryptAlgo"
)

var (
	ErrMultipleConfigFileFound = errors.New("multiple config_file found")
)

type configFileStore struct {
	handler BoltHandler
}

func newConfigFileStore(handler BoltHandler) *configFileStore {
	s := &configFileStore{handler: handler}
	return s
}

func (cf *configFileStore) LockConfigFile(tx store.Tx, file *model.ConfigFileKey) (*model.ConfigFile, error) {
	return cf.GetConfigFileTx(tx, file.Namespace, file.Group, file.Name)
}

// CreateConfigFile 创建配置文件
func (cf *configFileStore) CreateConfigFileTx(proxyTx store.Tx, file *model.ConfigFile) error {
	dbTx := proxyTx.GetDelegateTx().(*bolt.Tx)
	table, err := dbTx.CreateBucketIfNotExists([]byte(tblConfigFile))
	if err != nil {
		return store.Error(err)
	}
	nextId, err := table.NextSequence()
	if err != nil {
		return store.Error(err)
	}

	file.Id = nextId
	file.Valid = true
	file.CreateTime = time.Now()
	file.ModifyTime = file.CreateTime

	key := fmt.Sprintf("%s@%s@%s", file.Namespace, file.Group, file.Name)
	if err := saveValue(dbTx, tblConfigFile, key, file); err != nil {
		log.Error("[ConfigFile] save config_file", zap.String("key", key), zap.Error(err))
		return err
	}
	return nil
}

func (cf *configFileStore) GetConfigFile(namespace, group, name string) (*model.ConfigFile, error) {
	tx, err := cf.handler.StartTx()
	if err != nil {
		return nil, store.Error(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	return cf.GetConfigFileTx(tx, namespace, group, name)
}

// GetConfigFileTx 获取配置文件
func (cf *configFileStore) GetConfigFileTx(tx store.Tx, namespace, group, name string) (*model.ConfigFile, error) {
	dbTx := tx.GetDelegateTx().(*bolt.Tx)
	data, err := cf.getConfigFile(dbTx, namespace, group, name)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// GetConfigFile 获取配置文件
func (cf *configFileStore) getConfigFile(tx *bolt.Tx, namespace, group, name string) (*model.ConfigFile, error) {
	key := fmt.Sprintf("%s@%s@%s", namespace, group, name)

	values := make(map[string]interface{})
	if err := loadValues(tx, tblConfigFile, []string{key}, &model.ConfigFile{}, values); err != nil {
		return nil, err
	}

	if len(values) == 0 {
		return nil, nil
	}

	if len(values) > 1 {
		return nil, ErrMultipleConfigFileFound
	}

	data := values[key].(*model.ConfigFile)
	if data.Valid {
		return data, nil
	}
	return nil, nil
}

// QueryConfigFiles 翻页查询配置文件，group、name可为模糊匹配
func (cf *configFileStore) QueryConfigFiles(filter map[string]string,
	offset, limit uint32) (uint32, []*model.ConfigFile, error) {
	fields := []string{FileFieldNamespace, FileFieldGroup, FileFieldName, FileFieldValid}

	namespace := filter["namespace"]
	group := filter["group"]
	name := filter["name"]

	hasNs := len(namespace) != 0
	hasGroup := len(group) != 0
	hasName := len(name) != 0

	ret, err := cf.handler.LoadValuesByFilter(tblConfigFile, fields, &model.ConfigFile{},
		func(m map[string]interface{}) bool {
			valid, _ := m[FileFieldValid].(bool)
			if !valid {
				return false
			}

			saveNs, _ := m[FileFieldNamespace].(string)
			saveGroup, _ := m[FileFieldGroup].(string)
			saveFileName, _ := m[FileFieldName].(string)

			if hasNs && utils.IsWildNotMatch(saveNs, namespace) {
				return false
			}
			if hasGroup && utils.IsWildNotMatch(saveGroup, group) {
				return false
			}
			if hasName && utils.IsWildNotMatch(saveFileName, name) {
				return false
			}

			return true
		})

	if err != nil {
		return 0, nil, err
	}

	return uint32(len(ret)), doConfigFilePage(ret, offset, limit), nil
}

// UpdateConfigFile 更新配置文件
func (cf *configFileStore) UpdateConfigFileTx(tx store.Tx, file *model.ConfigFile) error {
	dbTx := tx.GetDelegateTx().(*bolt.Tx)
	key := fmt.Sprintf("%s@%s@%s", file.Namespace, file.Group, file.Name)
	properties := make(map[string]interface{})
	properties[FileFieldContent] = file.Content
	properties[FileFieldComment] = file.Comment
	properties[FileFieldFormat] = file.Format
	properties[FileFieldMetadata] = file.Metadata
	properties[FileFieldEncrypt] = file.Encrypt
	properties[FileFieldEncryptAlgo] = file.EncryptAlgo
	properties[FileFieldModifyTime] = time.Now()
	properties[FileFieldModifyBy] = file.ModifyBy
	if err := updateValue(dbTx, tblConfigFile, key, properties); err != nil {
		return err
	}
	return nil
}

// DeleteConfigFile 删除配置文件
func (cf *configFileStore) DeleteConfigFileTx(proxyTx store.Tx, namespace, group, name string) error {
	_, err := DoTransactionIfNeed(proxyTx, cf.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		key := fmt.Sprintf("%s@%s@%s", namespace, group, name)

		properties := make(map[string]interface{})
		properties[FileFieldValid] = false
		properties[FileFieldModifyTime] = time.Now()

		err := updateValue(tx, tblConfigFile, key, properties)
		return nil, err
	})
	return err
}

// CountConfigFiles 统计配置文件组下的配置文件数量
func (cf *configFileStore) CountConfigFiles(namespace, group string) (uint64, error) {
	hasNs := len(namespace) != 0
	hasGroup := len(group) != 0

	fields := []string{FileFieldNamespace, FileFieldGroup, FileFieldValid}

	ret, err := cf.handler.LoadValuesByFilter(tblConfigFile, fields, &model.ConfigFile{},
		func(m map[string]interface{}) bool {
			valid, _ := m[FileFieldValid].(bool)
			if !valid {
				return false
			}

			saveNs, _ := m[FileFieldNamespace].(string)
			saveGroup, _ := m[FileFieldGroup].(string)

			if hasNs && strings.Compare(saveNs, namespace) != 0 {
				return false
			}
			if hasGroup && strings.Compare(saveGroup, group) != 0 {
				return false
			}

			return true
		})

	if err != nil {
		return 0, err
	}

	return uint64(len(ret)), nil
}

func (cf *configFileStore) CountConfigFileEachGroup() (map[string]map[string]int64, error) {
	values, err := cf.handler.LoadValuesAll(tblConfigFile, &model.ConfigFile{})
	if err != nil {
		return nil, err
	}

	ret := make(map[string]map[string]int64)
	for i := range values {
		file := values[i].(*model.ConfigFile)
		if !file.Valid {
			continue
		}
		if _, ok := ret[file.Namespace]; !ok {
			ret[file.Namespace] = map[string]int64{}
		}
		if _, ok := ret[file.Namespace][file.Group]; !ok {
			ret[file.Namespace][file.Group] = 0
		}
		ret[file.Namespace][file.Group] = ret[file.Namespace][file.Group] + 1
	}

	return ret, nil
}

// doConfigFilePage 进行分页
func doConfigFilePage(ret map[string]interface{}, offset, limit uint32) []*model.ConfigFile {
	files := make([]*model.ConfigFile, 0, len(ret))
	beginIndex := offset
	endIndex := beginIndex + limit
	totalCount := uint32(len(ret))

	if totalCount == 0 {
		return files
	}
	if beginIndex >= endIndex {
		return files
	}
	if beginIndex >= totalCount {
		return files
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}
	for k := range ret {
		files = append(files, ret[k].(*model.ConfigFile))
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Id > files[j].Id
	})

	return files[beginIndex:endIndex]
}
