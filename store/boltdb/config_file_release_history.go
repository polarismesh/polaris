/*
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
	"database/sql"
	"github.com/polarismesh/polaris-server/common/model"
)

type configFileReleaseHistoryStore struct {
	handler BoltHandler
}

// CreateConfigFileReleaseHistory 创建配置文件发布历史记录
func (rh *configFileReleaseHistoryStore) CreateConfigFileReleaseHistory(tx *sql.Tx, fileReleaseHistory *model.ConfigFileReleaseHistory) error {
	return nil
}

// QueryConfigFileReleaseHistories 获取配置文件的发布历史记录
func (rh *configFileReleaseHistoryStore) QueryConfigFileReleaseHistories(namespace, group, fileName string, offset, limit uint32) (uint32, []*model.ConfigFileReleaseHistory, error) {
	return 0, nil, nil
}

func (rh *configFileReleaseHistoryStore) GetLatestConfigFileReleaseHistory(namespace, group, fileName string) (*model.ConfigFileReleaseHistory, error) {
	return nil, nil
}

func (rh *configFileReleaseHistoryStore) transferRows(rows *sql.Rows) ([]*model.ConfigFileReleaseHistory, error) {
	return nil, nil
}
