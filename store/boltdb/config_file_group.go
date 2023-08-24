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
	"github.com/polarismesh/polaris/store"
)

const (
	tblConfigFileGroup string = "ConfigFileGroup"

	FileGroupFieldId         string = "Id"
	FileGroupFieldName       string = "Name"
	FileGroupFieldNamespace  string = "Namespace"
	FileGroupFieldComment    string = "Comment"
	FileGroupFieldCreateBy   string = "CreateBy"
	FileGroupFieldModifyBy   string = "ModifyBy"
	FileGroupFieldCreateTime string = "CreateTime"
	FileGroupFieldModifyTime string = "ModifyTime"
	FileGroupFieldValid      string = "Valid"
	FileGroupFieldBusiness   string = "Business"
	FileGroupFieldDepartment string = "Department"
	FileGroupFieldMetadata   string = "Metadata"
)

var (
	ErrMultipleConfigFileGroupFound error = errors.New("multiple config_file_group found")
)

type configFileGroupStore struct {
	handler BoltHandler
}

func newConfigFileGroupStore(handler BoltHandler) *configFileGroupStore {
	s := &configFileGroupStore{handler: handler}
	return s
}

// CreateConfigFileGroup 创建配置文件组
func (fg *configFileGroupStore) CreateConfigFileGroup(
	fileGroup *model.ConfigFileGroup) (*model.ConfigFileGroup, error) {
	if fileGroup.Namespace == "" || fileGroup.Name == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, "ConfigFileGroup miss some param")
	}

	err := fg.handler.Execute(true, func(tx *bolt.Tx) error {
		table, err := tx.CreateBucketIfNotExists([]byte(tblConfigFileGroup))
		if err != nil {
			return err
		}
		nextId, err := table.NextSequence()
		if err != nil {
			return err
		}
		fileGroup.Id = nextId
		fileGroup.Valid = true
		fileGroup.CreateTime = time.Now()
		fileGroup.ModifyTime = fileGroup.CreateTime

		key := fmt.Sprintf("%s@@%s", fileGroup.Namespace, fileGroup.Name)
		if err := saveValue(tx, tblConfigFileGroup, key, fileGroup); err != nil {
			log.Error("[ConfigFileGroup] save info", zap.Error(err))
			return err
		}
		return nil
	})
	if err != nil {
		return nil, store.Error(err)
	}
	return fileGroup, nil
}

// GetConfigFileGroup 获取配置文件组
func (fg *configFileGroupStore) GetConfigFileGroup(namespace, name string) (*model.ConfigFileGroup, error) {
	if namespace == "" || name == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, "ConfigFileGroup miss some param")
	}

	key := fmt.Sprintf("%s@@%s", namespace, name)
	ret, err := fg.handler.LoadValues(tblConfigFileGroup, []string{key}, &model.ConfigFileGroup{})
	if err != nil {
		log.Error("[ConfigFileGroup] find by namespace and name", zap.Error(err))
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}
	val := ret[key]
	if val == nil {
		return nil, nil
	}
	cfg := val.(*model.ConfigFileGroup)
	if !cfg.Valid {
		return nil, nil
	}

	return cfg, nil
}

// QueryConfigFileGroups 翻页查询配置文件组, name 为模糊匹配关键字
func (fg *configFileGroupStore) QueryConfigFileGroups(namespace, name string, offset, limit uint32) (uint32,
	[]*model.ConfigFileGroup, error) {
	var (
		fields  = []string{FileGroupFieldNamespace, FileGroupFieldName, FileGroupFieldValid}
		hasNs   = len(namespace) != 0
		hasName = len(name) != 0
	)
	ret, err := fg.handler.LoadValuesByFilter(tblConfigFileGroup, fields, &model.ConfigFileGroup{},
		func(m map[string]interface{}) bool {
			valid, ok := m[FileGroupFieldValid].(bool)
			if ok && !valid {
				return false
			}

			saveNamespace, _ := m[FileGroupFieldNamespace].(string)
			saveName, _ := m[FileGroupFieldName].(string)

			if hasNs && strings.Compare(namespace, saveNamespace) != 0 {
				return false
			}

			if hasName {
				if !strings.Contains(saveName, name[:len(name)-1]) {
					return false
				}
			}

			return true
		})

	if err != nil {
		log.Error("[ConfigFileGroup] find by page", zap.Error(err))
		return 0, nil, err
	}

	return uint32(len(ret)), doConfigFileGroupPage(ret, offset, limit), nil
}

// DeleteConfigFileGroup 删除配置文件组
func (fg *configFileGroupStore) DeleteConfigFileGroup(namespace, name string) error {
	if namespace == "" || name == "" {
		return store.NewStatusError(store.EmptyParamsErr, "ConfigFileGroup miss some param")
	}

	key := fmt.Sprintf("%s@@%s", namespace, name)
	properties := make(map[string]interface{})
	properties[FileGroupFieldValid] = false
	properties[FileGroupFieldModifyTime] = time.Now()

	if err := fg.handler.UpdateValue(tblConfigFileGroup, key, properties); err != nil {
		log.Error("[ConfigFileGroup] do delete", zap.Error(err))
		return store.Error(err)
	}
	return nil
}

// UpdateConfigFileGroup 更新配置文件组信息
func (fg *configFileGroupStore) UpdateConfigFileGroup(fileGroup *model.ConfigFileGroup) error {
	if fileGroup.Namespace == "" || fileGroup.Name == "" {
		return store.NewStatusError(store.EmptyParamsErr, "ConfigFileGroup miss some param")
	}

	key := fmt.Sprintf("%s@@%s", fileGroup.Namespace, fileGroup.Name)
	properties := make(map[string]interface{})
	properties[FileGroupFieldComment] = fileGroup.Comment
	properties[FileGroupFieldModifyBy] = fileGroup.ModifyBy
	properties[FileGroupFieldBusiness] = fileGroup.Business
	properties[FileGroupFieldDepartment] = fileGroup.Department
	properties[FileGroupFieldMetadata] = fileGroup.Metadata
	properties[FileGroupFieldModifyTime] = time.Now()

	if err := fg.handler.UpdateValue(tblConfigFileGroup, key, properties); err != nil {
		log.Error("[ConfigFileGroup] do update", zap.Error(err))
		return store.Error(err)
	}
	return nil
}

func (fg *configFileGroupStore) GetMoreConfigGroup(firstUpdate bool,
	mtime time.Time) ([]*model.ConfigFileGroup, error) {

	if firstUpdate {
		mtime = time.Time{}
	}

	fields := []string{FileGroupFieldModifyTime}
	ret, err := fg.handler.LoadValuesByFilter(tblConfigFileGroup, fields, &model.ConfigFileGroup{},
		func(m map[string]interface{}) bool {
			saveMt, _ := m[FileGroupFieldModifyTime].(time.Time)
			return !saveMt.Before(mtime)
		})

	if err != nil {
		return nil, err
	}

	groups := make([]*model.ConfigFileGroup, 0, len(ret))
	for _, v := range ret {
		groups = append(groups, v.(*model.ConfigFileGroup))
	}

	return groups, nil
}

// CountConfigReleases count the release data
func (fg *configFileGroupStore) CountConfigGroups(namespace string) (uint64, error) {
	fields := []string{FileGroupFieldNamespace, FileGroupFieldValid}
	ret, err := fg.handler.LoadValuesByFilter(tblConfigFileGroup, fields, &model.ConfigFileGroup{},
		func(m map[string]interface{}) bool {
			valid, _ := m[FileGroupFieldValid].(bool)
			if !valid {
				return false
			}
			saveNs, _ := m[FileGroupFieldNamespace].(string)
			return saveNs == namespace
		})
	if err != nil {
		return 0, err
	}
	return uint64(len(ret)), err
}

// doConfigFileGroupPage 进行分页
func doConfigFileGroupPage(ret map[string]interface{}, offset, limit uint32) []*model.ConfigFileGroup {
	var (
		groups     = make([]*model.ConfigFileGroup, 0, len(ret))
		beginIndex = offset
		endIndex   = beginIndex + limit
		totalCount = uint32(len(ret))
	)
	if totalCount == 0 {
		return groups
	}
	if beginIndex >= endIndex {
		return groups
	}
	if beginIndex >= totalCount {
		return groups
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}
	for k := range ret {
		groups = append(groups, ret[k].(*model.ConfigFileGroup))
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].ModifyTime.After(groups[j].ModifyTime)
	})

	return groups[beginIndex:endIndex]
}
