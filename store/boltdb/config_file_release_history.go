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

	"github.com/boltdb/bolt"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

const (
	tblConfigFileReleaseHistory   string = "ConfigFileReleaseHistory"
	tblConfigFileReleaseHistoryID string = "ConfigFileReleaseHistoryID"

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
	id      uint64
	handler BoltHandler
}

func newConfigFileReleaseHistoryStore(handler BoltHandler) (*configFileReleaseHistoryStore, error) {
	s := &configFileReleaseHistoryStore{handler: handler, id: 0}
	ret, err := handler.LoadValues(tblConfigFileReleaseHistoryID, []string{tblConfigFileReleaseHistoryID}, &IDHolder{})
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return s, err
	}
	val := ret[tblConfigFileReleaseHistoryID].(*IDHolder)
	s.id = val.ID
	return s, nil
}

// CreateConfigFileReleaseHistory 创建配置文件发布历史记录
func (rh *configFileReleaseHistoryStore) CreateConfigFileReleaseHistory(proxyTx store.Tx,
	fileReleaseHistory *model.ConfigFileReleaseHistory) error {

	_, err := DoTransactionIfNeed(proxyTx, rh.handler, func(tx *bolt.Tx) ([]interface{}, error) {
		rh.id++
		fileReleaseHistory.Id = rh.id

		if err := saveValue(tx, tblConfigFileReleaseHistoryID, tblConfigFileReleaseHistoryID, &IDHolder{
			ID: rh.id,
		}); err != nil {
			log.Error("[ConfigFileReleaseHistory] save auto_increment id", zap.Error(err))
			return nil, err
		}

		key := strconv.FormatUint(rh.id, 10)

		fileReleaseHistory.Valid = true
		fileReleaseHistory.CreateTime = time.Now()
		fileReleaseHistory.ModifyTime = fileReleaseHistory.CreateTime

		if err := saveValue(tx, tblConfigFileReleaseHistory, key, fileReleaseHistory); err != nil {
			log.Error("[ConfigFileReleaseHistory] save info", zap.Error(err))
			return nil, err
		}

		return nil, nil
	})

	return err
}

// QueryConfigFileReleaseHistories 获取配置文件的发布历史记录
func (rh *configFileReleaseHistoryStore) QueryConfigFileReleaseHistories(namespace, group,
	fileName string, offset, limit uint32, endId uint64) (uint32, []*model.ConfigFileReleaseHistory, error) {

	fields := []string{FileHistoryFieldNamespace, FileHistoryFieldGroup, FileHistoryFieldFileName, FileHistoryFieldId}

	hasNs := len(namespace) > 0
	hasEndId := endId > 0

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

// doConfigFileGroupPage 进行分页
func doConfigFileHistoryPage(ret map[string]interface{}, offset, limit uint32) []*model.ConfigFileReleaseHistory {

	histories := make([]*model.ConfigFileReleaseHistory, 0, len(ret))

	beginIndex := offset
	endIndex := beginIndex + limit
	totalCount := uint32(len(ret))

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
