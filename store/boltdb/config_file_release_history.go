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
	"sort"
	"strconv"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

const (
	tblConfigFileReleaseHistory string = "ConfigFileReleaseHistory"

	FileHistoryFieldId         string = "Id"
	FileHistoryFieldName       string = "Name"
	FileHistoryFieldNamespace  string = "Namespace"
	FileHistoryFieldGroup      string = "Group"
	FileHistoryFieldFileName   string = "FileName"
	FileHistoryFieldFormat     string = "Format"
	FileHistoryFieldTags       string = "Tags"
	FileHistoryFieldContent    string = "Content"
	FileHistoryFieldComment    string = "Comment"
	FileHistoryFieldMd5        string = "Md5"
	FileHistoryFieldType       string = "Type"
	FileHistoryFieldStatus     string = "Status"
	FileHistoryFieldCreateBy   string = "CreateBy"
	FileHistoryFieldModifyBy   string = "ModifyBy"
	FileHistoryFieldCreateTime string = "CreateTime"
	FileHistoryFieldModifyTime string = "ModifyTime"
	FileHistoryFieldValid      string = "Valid"
)

type configFileReleaseHistoryStore struct {
	handler BoltHandler
}

func newConfigFileReleaseHistoryStore(handler BoltHandler) *configFileReleaseHistoryStore {
	s := &configFileReleaseHistoryStore{handler: handler}
	return s
}

// CreateConfigFileReleaseHistory 创建配置文件发布历史记录
func (rh *configFileReleaseHistoryStore) CreateConfigFileReleaseHistory(
	history *model.ConfigFileReleaseHistory) error {

	err := rh.handler.Execute(true, func(tx *bolt.Tx) error {
		table, err := tx.CreateBucketIfNotExists([]byte(tblConfigFileReleaseHistory))
		if err != nil {
			return err
		}
		nextId, err := table.NextSequence()
		if err != nil {
			return err
		}

		history.Id = nextId
		key := strconv.FormatUint(history.Id, 10)
		history.Valid = true
		history.CreateTime = time.Now()
		history.ModifyTime = history.CreateTime

		if err := saveValue(tx, tblConfigFileReleaseHistory, key, history); err != nil {
			log.Error("[ConfigFileReleaseHistory] save info", zap.Error(err))
			return err
		}
		return nil
	})

	return store.Error(err)
}

// QueryConfigFileReleaseHistories 获取配置文件的发布历史记录
func (rh *configFileReleaseHistoryStore) QueryConfigFileReleaseHistories(filter map[string]string,
	offset, limit uint32) (uint32, []*model.ConfigFileReleaseHistory, error) {

	var (
		namespace = filter["namespace"]
		group     = filter["group"]
		fileName  = filter["file_name"]
		endId, _  = strconv.ParseUint(filter["endId"], 10, 64)
		fields    = []string{FileHistoryFieldNamespace, FileHistoryFieldGroup,
			FileHistoryFieldFileName, FileHistoryFieldId}
		hasNs    = len(namespace) > 0
		hasEndId = endId > 0
	)

	ret, err := rh.handler.LoadValuesByFilter(tblConfigFileReleaseHistory, fields,
		&model.ConfigFileReleaseHistory{}, func(m map[string]interface{}) bool {
			saveNs, _ := m[FileHistoryFieldNamespace].(string)
			saveFileGroup, _ := m[FileHistoryFieldGroup].(string)
			saveFileName, _ := m[FileHistoryFieldFileName].(string)
			saveID, _ := m[FileHistoryFieldId].(uint64)

			if hasNs && strings.Compare(namespace, saveNs) != 0 {
				return false
			}

			if hasEndId && endId <= uint64(saveID) {
				return false
			}

			ret := strings.Contains(saveFileGroup, group) && strings.Contains(saveFileName, fileName)
			return ret
		})

	if err != nil {
		return 0, nil, err
	}

	return uint32(len(ret)), doConfigFileHistoryPage(ret, offset, limit), nil
}

// GetLatestConfigFileReleaseHistory 获取最后一次发布记录
func (rh *configFileReleaseHistoryStore) GetLatestConfigFileReleaseHistory(namespace, group,
	fileName string) (*model.ConfigFileReleaseHistory, error) {
	fields := []string{FileHistoryFieldNamespace, FileHistoryFieldGroup, FileHistoryFieldFileName}
	ret, err := rh.handler.LoadValuesByFilter(tblConfigFileReleaseHistory, fields,
		&model.ConfigFileReleaseHistory{}, func(m map[string]interface{}) bool {
			saveNs, _ := m[FileHistoryFieldNamespace].(string)
			saveFileGroup, _ := m[FileHistoryFieldGroup].(string)
			saveFileName, _ := m[FileHistoryFieldFileName].(string)

			equalNs := strings.Compare(saveNs, namespace) == 0
			equalGroup := strings.Compare(saveFileGroup, group) == 0
			equalName := strings.Compare(saveFileName, fileName) == 0
			return equalNs && equalGroup && equalName
		})

	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, nil
	}

	histories := make([]*model.ConfigFileReleaseHistory, 0, len(ret))
	for k := range ret {
		histories = append(histories, ret[k].(*model.ConfigFileReleaseHistory))
	}

	sort.Slice(histories, func(i, j int) bool {
		return histories[i].Id > histories[j].Id
	})

	return histories[0], nil
}

func (rh *configFileReleaseHistoryStore) CleanConfigFileReleaseHistory(endTime time.Time, limit uint64) error {

	fields := []string{FileHistoryFieldCreateTime, FileHistoryFieldId}
	needDel := make([]string, 0, limit)

	_, err := rh.handler.LoadValuesByFilter(tblConfigFileReleaseHistory, fields,
		&model.ConfigFileReleaseHistory{}, func(m map[string]interface{}) bool {
			saveCreateBy, _ := m[FileHistoryFieldCreateTime].(time.Time)
			saveId := m[FileHistoryFieldId].(uint64)

			if endTime.After(saveCreateBy) {
				needDel = append(needDel, strconv.FormatUint(saveId, 10))
			}
			return false
		})
	if err != nil {
		return err
	}

	return rh.handler.DeleteValues(tblConfigFileReleaseHistory, needDel)
}

// doConfigFileGroupPage 进行分页
func doConfigFileHistoryPage(ret map[string]interface{}, offset, limit uint32) []*model.ConfigFileReleaseHistory {
	var (
		histories  = make([]*model.ConfigFileReleaseHistory, 0, len(ret))
		beginIndex = offset
		endIndex   = beginIndex + limit
		totalCount = uint32(len(ret))
	)

	if totalCount == 0 {
		return histories
	}
	if beginIndex >= endIndex {
		return histories
	}
	if beginIndex >= totalCount {
		return histories
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}
	for k := range ret {
		histories = append(histories, ret[k].(*model.ConfigFileReleaseHistory))
	}

	sort.Slice(histories, func(i, j int) bool {
		return histories[i].Id > histories[j].Id
	})

	return histories[beginIndex:endIndex]
}
