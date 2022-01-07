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
	"time"
)

type configFileReleaseStore struct {
	handler BoltHandler
}

// CreateConfigFileRelease 新建配置文件发布
func (cfr *configFileReleaseStore) CreateConfigFileRelease(tx *sql.Tx, fileRelease *model.ConfigFileRelease) (*model.ConfigFileRelease, error) {
	return nil, nil
}

// UpdateConfigFileRelease 更新配置文件发布
func (cfr *configFileReleaseStore) UpdateConfigFileRelease(tx *sql.Tx, fileRelease *model.ConfigFileRelease) (*model.ConfigFileRelease, error) {
	return nil, nil
}

// GetConfigFileRelease 获取配置文件发布，只返回 flag=0 的记录
func (cfr *configFileReleaseStore) GetConfigFileRelease(tx *sql.Tx, namespace, group, fileName string) (*model.ConfigFileRelease, error) {
	return nil, nil
}

func (cfr *configFileReleaseStore) GetConfigFileReleaseWithAllFlag(tx *sql.Tx, namespace, group, fileName string) (*model.ConfigFileRelease, error) {
	return nil, nil
}

func (cfr *configFileReleaseStore) getConfigFileReleaseByFlag(tx *sql.Tx, namespace, group, fileName string, withAllFlag bool) (*model.ConfigFileRelease, error) {
	return nil, nil
}

func (cfr *configFileReleaseStore) DeleteConfigFileRelease(tx *sql.Tx, namespace, group, fileName, deleteBy string) error {
	return nil
}

// FindConfigFileReleaseByModifyTimeAfter 获取最后更新时间大于某个时间点的发布，注意包含 flag = 1 的，为了能够获取被删除的 release
func (cfr *configFileReleaseStore) FindConfigFileReleaseByModifyTimeAfter(modifyTime time.Time) ([]*model.ConfigFileRelease, error) {
	return nil, nil
}
