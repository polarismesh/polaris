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

	"github.com/boltdb/bolt"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
)

const (
	tblConfigFile   string = "ConfigFile"
	tblConfigFileID string = "ConfigFileID"

	FileFieldId         string = "Id"
	FileFieldName       string = "Name"
	FileFieldNamespace  string = "Namespace"
	FileFieldGroup      string = "Group"
	FileFieldContent    string = "Content"
	FileFieldComment    string = "Comment"
	FileFieldFormat     string = "Format"
	FileFieldFlag       string = "Flag"
	FileFieldCreateTime string = "CreateTime"
	FileFieldCreateBy   string = "CreateBy"
	FileFieldModifyTime string = "ModifyTime"
	FileFieldModifyBy   string = "ModifyBy"
	FileFieldValid      string = "Valid"
)

var (
	ErrMultipleConfigFileFound = errors.New("multiple config_file found")
)

type configFileStore struct {
	id      uint64
	handler BoltHandler
}

func newConfigFileStore(handler BoltHandler) (*configFileStore, error) {
	s := &configFileStore{handler: handler, id: 0}
	ret, err := handler.LoadValues(tblConfigFileID, []string{tblConfigFileID}, &IDHolder{})
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return s, nil
	}
	val := ret[tblConfigFileID].(*IDHolder)
	s.id = val.ID
	return s, nil
}

// CreateConfigFile 创建配置文件
func (cf *configFileStore) CreateConfigFile(proxyTx store.Tx, file *model.ConfigFile) (*model.ConfigFile, error) {
	ret, err := DoTransactionIfNeed(proxyTx, cf.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		cf.id++
		file.Id = cf.id
		file.Valid = true
		file.CreateTime = time.Now()
		file.ModifyTime = file.CreateTime

		if err := saveValue(tx, tblConfigFileID, tblConfigFileID, &IDHolder{
			ID: cf.id,
		}); err != nil {
			log.Error("[ConfigFile] save auto_increment id", zap.Error(err))
			return nil, err
		}

		key := fmt.Sprintf("%s@%s@%s", file.Namespace, file.Group, file.Name)
		if err := saveValue(tx, tblConfigFile, key, file); err != nil {
			log.Error("[ConfigFile] save config_file", zap.String("key", key), zap.Error(err))
			return nil, err
		}

		data, err := cf.getConfigFile(tx, file.Namespace, file.Group, file.Name)
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

	return ret[0].(*model.ConfigFile), nil
}

// GetConfigFile 获取配置文件
func (cf *configFileStore) GetConfigFile(proxyTx store.Tx, namespace, group, name string) (*model.ConfigFile, error) {
	ret, err := DoTransactionIfNeed(proxyTx, cf.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		data, err := cf.getConfigFile(tx, namespace, group, name)
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

	return ret[0].(*model.ConfigFile), nil
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
func (cf *configFileStore) QueryConfigFiles(namespace, group, name string, offset, limit uint32) (uint32, []*model.ConfigFile, error) {

	fields := []string{FileFieldNamespace, FileFieldGroup, FileFieldName, FileFieldValid}

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

			if hasNs && !strings.Contains(saveNs, namespace) {
				return false
			}
			if hasGroup && !strings.Contains(saveGroup, group) {
				return false
			}
			if hasName && !strings.Contains(saveFileName, name) {
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
func (cf *configFileStore) UpdateConfigFile(proxyTx store.Tx, file *model.ConfigFile) (*model.ConfigFile, error) {
	ret, err := DoTransactionIfNeed(proxyTx, cf.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		key := fmt.Sprintf("%s@%s@%s", file.Namespace, file.Group, file.Name)

		properties := make(map[string]interface{})
		properties[FileFieldContent] = file.Content
		properties[FileFieldComment] = file.Comment
		properties[FileFieldFormat] = file.Format
		properties[FileFieldModifyTime] = time.Now()
		properties[FileFieldModifyBy] = file.ModifyBy
		if err := updateValue(tx, tblConfigFile, key, properties); err != nil {
			return nil, err
		}
		data, err := cf.getConfigFile(tx, file.Namespace, file.Group, file.Name)
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

	return ret[0].(*model.ConfigFile), nil
}

// DeleteConfigFile 删除配置文件
func (cf *configFileStore) DeleteConfigFile(proxyTx store.Tx, namespace, group, name string) error {
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

// CountByConfigFileGroup 统计配置文件组下的配置文件数量
func (cf *configFileStore) CountByConfigFileGroup(namespace, group string) (uint64, error) {

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
