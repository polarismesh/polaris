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
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
)

type configFileTagStore struct {
	handler BoltHandler
}

// CreateConfigFileTag 创建配置文件标签
func (t *configFileTagStore) CreateConfigFileTag(tx store.Tx, fileTag *model.ConfigFileTag) error {
	return nil
}

// QueryConfigFileByTag 通过标签查询配置文件
func (t *configFileTagStore) QueryConfigFileByTag(namespace, group, fileName string, tags ...string) ([]*model.ConfigFileTag, error) {
	return nil, nil
}

// QueryTagByConfigFile 查询配置文件标签
func (t *configFileTagStore) QueryTagByConfigFile(namespace, group, fileName string) ([]*model.ConfigFileTag, error) {
	return nil, nil
}

// DeleteConfigFileTag 删除配置文件标签
func (t *configFileTagStore) DeleteConfigFileTag(tx store.Tx, namespace, group, fileName, key, value string) error {
	return nil
}

// DeleteTagByConfigFile 删除配置文件的标签
func (t *configFileTagStore) DeleteTagByConfigFile(tx store.Tx, namespace, group, fileName string) error {
	return nil
}
