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

	"github.com/boltdb/bolt"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
)

const (
	tblConfigFileReleaseHistory   string = "ConfigFileReleaseHistory"
	tblConfigFileReleaseHistoryID string = "ConfigFileReleaseHistoryID"
	FileHistoryFieldId            string = "Id"
	FileHistoryFieldName          string = "Name"
	FileHistoryFieldNamespace     string = "Namespace"
	FileHistoryFieldGroup         string = "Group"
	FileHistoryFieldFileName      string = "FileName"
	FileHistoryFieldFormat        string = "Format"
	FileHistoryFieldTags          string = "Tags"
	FileHistoryFieldContent       string = "Content"
	FileHistoryFieldComment       string = "Comment"
	FileHistoryFieldMd5           string = "Md5"
	FileHistoryFieldType          string = "Type"
	FileHistoryFieldStatus        string = "Status"
	FileHistoryFieldCreateBy      string = "CreateBy"
	FileHistoryFieldModifyBy      string = "ModifyBy"
	FileHistoryFieldCreateTime    string = "CreateTime"
	FileHistoryFieldModifyTime    string = "ModifyTime"
	FileHistoryFieldValid         string = "Valid"
)

type configFileReleaseHistoryStore struct {
	id      uint64
	handler BoltHandler
}

// CreateConfigFileReleaseHistory 创建配置文件发布历史记录
func (rh *configFileReleaseHistoryStore) CreateConfigFileReleaseHistory(proxyTx store.Tx,
	fileReleaseHistory *model.ConfigFileReleaseHistory) error {

	var err error

	if proxyTx == nil {
		proxyTx, err = rh.handler.StartTx()
		if err != nil {
			return err
		}
	}

	tx := proxyTx.GetDelegateTx().(*bolt.Tx)
	defer tx.Rollback()

	rh.id++
	fileReleaseHistory.Id = rh.id

	if err := saveValue(tx, tblConfigFileGroupID, tblConfigFileGroupID, rh.id); err != nil {
		log.Error("[ConfigFileReleaseHistory] save auto_increment id", zap.Error(err))
		return err
	}

	key := strconv.FormatUint(rh.id, 10)

	if err := saveValue(tx, tblConfigFileGroup, key, fileReleaseHistory); err != nil {
		log.Error("[ConfigFileReleaseHistory] save info", zap.Error(err))
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error("[ConfigFileReleaseHistory] do tx commit", zap.Error(err))
		return err
	}

	return nil
}

// QueryConfigFileReleaseHistories 获取配置文件的发布历史记录
func (rh *configFileReleaseHistoryStore) QueryConfigFileReleaseHistories(namespace, group,
	fileName string, offset, limit uint32, endId uint64) (uint32, []*model.ConfigFileReleaseHistory, error) {

	fields := []string{FileHistoryFieldNamespace, FileHistoryFieldGroup, FileHistoryFieldFileName, FileHistoryFieldId}

	hasNs := len(namespace)
	hasEndId := endId > 0

	ret, err := rh.handler.LoadValuesByFilter(tblConfigFileReleaseHistory, fields,
		&model.ConfigFileReleaseHistory{}, func(m map[string]interface{}) bool {

			return true
		})

	if err != nil {
		return 0, nil, err
	}

	return uint32(len(ret)), doConfigFileHistoryPage(ret, offset, limit), nil
}

// GetLatestConfigFileReleaseHistory 获取最后一次发布记录
func (rh *configFileReleaseHistoryStore) GetLatestConfigFileReleaseHistory(namespace, group,
	fileName string) (*model.ConfigFileReleaseHistory, error) {

	return nil, nil
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
		return histories[i].ModifyTime.After(histories[j].ModifyTime)
	})

	return histories[beginIndex:endIndex]

}
