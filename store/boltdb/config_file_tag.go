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
	"fmt"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
)

const (
	tbleConfigFileTag   string = "ConfigFileTag"
	tbleConfigFileTagID string = "ConfigFileTagID"
	TagFieldId          string = "Id"
	TagFieldKey         string = "Key"
	TagFieldValue       string = "Value"
	TagFieldNamespace   string = "Namespace"
	TagFieldGroup       string = "Group"
	TagFieldFileName    string = "FileName"
	TagFieldCreateTime  string = "CreateTime"
	TagFieldCreateBy    string = "CreateBy"
	TagFieldModifyTime  string = "ModifyTime"
	TagFieldModifyBy    string = "ModifyBy"
	TagFieldValid       string = "Valid"
)

type configFileTagStore struct {
	id      uint64
	handler BoltHandler
}

func newConfigFileTagStore(handler BoltHandler) (*configFileTagStore, error) {
	s := &configFileTagStore{handler: handler, id: 0}
	ret, err := handler.LoadValues(tbleConfigFileTagID, []string{tbleConfigFileTagID}, &IDHolder{})
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return s, nil
	}
	val := ret[tbleConfigFileTagID].(*IDHolder)
	s.id = val.ID
	return s, nil
}

// CreateConfigFileTag 创建配置文件标签
func (t *configFileTagStore) CreateConfigFileTag(proxyTx store.Tx, fileTag *model.ConfigFileTag) error {
	_, err := DoTransactionIfNeed(proxyTx, t.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		t.id++
		fileTag.Id = t.id
		fileTag.Valid = true

		if err := saveValue(tx, tbleConfigFileTagID, tbleConfigFileTagID, &IDHolder{
			ID: t.id,
		}); err != nil {
			log.Error("[ConfigFileTag] save auto_increment id", zap.Error(err))
			return nil, err
		}

		key := fmt.Sprintf("%s@%s@%s@%s@%s", fileTag.Key, fileTag.Value, fileTag.Namespace, fileTag.Group, fileTag.FileName)
		if err := saveValue(tx, tbleConfigFileTag, key, fileTag); err != nil {
			log.Error("[ConfigFileTag] save info", zap.Error(err))
			return nil, err
		}

		return nil, nil
	})

	return err
}

// QueryConfigFileByTag 通过标签查询配置文件
func (t *configFileTagStore) QueryConfigFileByTag(namespace, group, fileName string, tags ...string) ([]*model.ConfigFileTag, error) {

	fields := []string{TagFieldNamespace, TagFieldGroup, TagFieldFileName, TagFieldKey, TagFieldValue}

	tagSearchMap := make(map[string]string)
	for i := 0; i < len(tags); i = i + 2 {
		tagSearchMap[tags[i]] = tags[i+1]
	}

	ret, err := t.handler.LoadValuesByFilter(tbleConfigFileTag, fields, &model.ConfigFileTag{}, func(m map[string]interface{}) bool {
		saveNs, _ := m[TagFieldNamespace].(string)
		saveGroup, _ := m[TagFieldGroup].(string)
		saveFileName, _ := m[TagFieldFileName].(string)
		saveTagKey, _ := m[TagFieldKey].(string)
		saveTagValue, _ := m[TagFieldValue].(string)

		equalNs := strings.Compare(saveNs, namespace) == 0
		equalGroup := strings.Contains(saveGroup, group)
		equalFileName := strings.Contains(saveFileName, fileName)

		if !equalNs || !equalGroup || !equalFileName {
			return false
		}

		tagVal, ok := tagSearchMap[saveTagKey]
		if !ok {
			return false
		}

		return strings.Compare(tagVal, saveTagValue) == 0
	})

	if err != nil {
		return nil, err
	}

	tagList := make([]*model.ConfigFileTag, 0, len(ret))

	for _, v := range ret {
		tagList = append(tagList, v.(*model.ConfigFileTag))
	}

	return tagList, nil
}

// QueryTagByConfigFile 查询配置文件标签
func (t *configFileTagStore) QueryTagByConfigFile(namespace, group, fileName string) ([]*model.ConfigFileTag, error) {

	fields := []string{TagFieldNamespace, TagFieldGroup, TagFieldFileName}

	ret, err := t.handler.LoadValuesByFilter(tbleConfigFileTag, fields, &model.ConfigFileTag{},
		func(m map[string]interface{}) bool {
			saveNs, _ := m[TagFieldNamespace].(string)
			saveGroup, _ := m[TagFieldGroup].(string)
			saveFileName, _ := m[TagFieldFileName].(string)

			equalNs := strings.Compare(saveNs, namespace) == 0
			equalGroup := strings.Compare(saveGroup, group) == 0
			equalFile := strings.Compare(saveFileName, fileName) == 0

			return equalNs && equalGroup && equalFile
		})

	if err != nil {
		return nil, err
	}

	tags := make([]*model.ConfigFileTag, 0, len(ret))

	for _, v := range ret {
		tags = append(tags, v.(*model.ConfigFileTag))
	}

	return tags, nil
}

// DeleteConfigFileTag 删除配置文件标签
func (t *configFileTagStore) DeleteConfigFileTag(proxyTx store.Tx, namespace, group, fileName, key, value string) error {
	_, err := DoTransactionIfNeed(proxyTx, t.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		dataKey := fmt.Sprintf("%s@%s@%s@%s@%s", key, value, namespace, group, fileName)

		if err := deleteValues(tx, tbleConfigFileTag, []string{dataKey}, false); err != nil {
			return nil, err
		}
		return nil, nil
	})

	return err
}

// DeleteTagByConfigFile 删除配置文件的标签
func (t *configFileTagStore) DeleteTagByConfigFile(proxyTx store.Tx, namespace, group, fileName string) error {
	_, err := DoTransactionIfNeed(proxyTx, t.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		fields := []string{TagFieldNamespace, TagFieldGroup, TagFieldFileName}

		ret := make(map[string]interface{})
		err := loadValuesByFilter(tx, tbleConfigFileTag, fields, &model.ConfigFileTag{},
			func(m map[string]interface{}) bool {
				saveNs, _ := m[TagFieldNamespace].(string)
				saveGroup, _ := m[TagFieldGroup].(string)
				saveFileName, _ := m[TagFieldFileName].(string)

				equalNs := strings.Compare(saveNs, namespace) == 0
				equalGroup := strings.Compare(saveGroup, group) == 0
				equalFile := strings.Compare(saveFileName, fileName) == 0

				return equalNs && equalGroup && equalFile
			}, ret)

		if err != nil {
			return nil, err
		}

		keys := make([]string, 0, len(ret))

		for _, v := range ret {
			dataKey := fmt.Sprintf("%s@%s@%s@%s@%s",
				v.(*model.ConfigFileTag).Key, v.(*model.ConfigFileTag).Value, namespace, group, fileName)
			keys = append(keys, dataKey)
		}

		if err := deleteValues(tx, tbleConfigFileTag, keys, false); err != nil {
			return nil, err
		}

		return nil, nil
	})
	return err
}
