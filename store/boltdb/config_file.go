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

type configFileStore struct {
	handler BoltHandler
}

func (cf *configFileStore) StartTx() (*sql.Tx, error) {
	return nil, nil
}

// CreateConfigFile 创建配置文件
func (cf *configFileStore) CreateConfigFile(tx *sql.Tx, file *model.ConfigFile) (*model.ConfigFile, error) {
	return nil, nil
}

// GetConfigFile 获取配置文件
func (cf *configFileStore) GetConfigFile(tx *sql.Tx, namespace, group, name string) (*model.ConfigFile, error) {
	return nil, nil
}

// QueryConfigFiles 翻页查询配置文件，group、name可为模糊匹配
func (cf *configFileStore) QueryConfigFiles(namespace, group, name string, offset, limit int) (uint32, []*model.ConfigFile, error) {
	return 0, nil, nil

}

// UpdateConfigFile 更新配置文件
func (cf *configFileStore) UpdateConfigFile(tx *sql.Tx, file *model.ConfigFile) (*model.ConfigFile, error) {
	return nil, nil
}

// DeleteConfigFile 删除配置文件
func (cf *configFileStore) DeleteConfigFile(tx *sql.Tx, namespace, group, name string) error {
	return nil
}
