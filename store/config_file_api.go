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

package store

import (
	"time"

	"github.com/polarismesh/polaris/common/model"
)

// ConfigFileModuleStore 配置中心模块存储接口
type ConfigFileModuleStore interface {
	ConfigFileGroupStore
	ConfigFileStore
	ConfigFileReleaseStore
	ConfigFileReleaseHistoryStore
	ConfigFileTemplateStore
}

// ConfigFileGroupStore 配置文件组存储接口
type ConfigFileGroupStore interface {
	// CreateConfigFileGroup 创建配置文件组
	CreateConfigFileGroup(fileGroup *model.ConfigFileGroup) (*model.ConfigFileGroup, error)
	// UpdateConfigFileGroup 更新配置文件组
	UpdateConfigFileGroup(fileGroup *model.ConfigFileGroup) error
	// GetConfigFileGroup 获取单个配置文件组
	GetConfigFileGroup(namespace, name string) (*model.ConfigFileGroup, error)
	// DeleteConfigFileGroup 删除配置文件组
	DeleteConfigFileGroup(namespace, name string) error
	// GetMoreConfigGroup 获取配置分组
	GetMoreConfigGroup(firstUpdate bool, mtime time.Time) ([]*model.ConfigFileGroup, error)
	// CountConfigGroups 获取一个命名空间下的配置分组数量
	CountConfigGroups(namespace string) (uint64, error)
}

// ConfigFileStore 配置文件存储接口
type ConfigFileStore interface {
	// LockConfigFile 加锁配置文件
	LockConfigFile(tx Tx, file *model.ConfigFileKey) (*model.ConfigFile, error)
	// CreateConfigFileTx 创建配置文件
	CreateConfigFileTx(tx Tx, file *model.ConfigFile) error
	// GetConfigFile 获取配置文件
	GetConfigFile(namespace, group, name string) (*model.ConfigFile, error)
	// GetConfigFileTx 获取配置文件
	GetConfigFileTx(tx Tx, namespace, group, name string) (*model.ConfigFile, error)
	// QueryConfigFiles 翻页查询配置文件，group、name可为模糊匹配
	QueryConfigFiles(filter map[string]string, offset uint32, limit uint32) (uint32, []*model.ConfigFile, error)
	// UpdateConfigFileTx 更新配置文件
	UpdateConfigFileTx(tx Tx, file *model.ConfigFile) error
	// DeleteConfigFileTx 删除配置文件
	DeleteConfigFileTx(tx Tx, namespace, group, name string) error
	// CountConfigFiles 获取一个配置文件组下的文件数量
	CountConfigFiles(namespace, group string) (uint64, error)
	// CountConfigFileEachGroup 统计 namespace.group 下的配置文件数量
	CountConfigFileEachGroup() (map[string]map[string]int64, error)
}

// ConfigFileReleaseStore 配置文件发布存储接口
type ConfigFileReleaseStore interface {
	// GetConfigFileActiveRelease	获取配置文件处于 Active 的配置发布记录
	GetConfigFileActiveRelease(file *model.ConfigFileKey) (*model.ConfigFileRelease, error)
	// GetConfigFileActiveReleaseTx	获取配置文件处于 Active 的配置发布记录
	GetConfigFileActiveReleaseTx(tx Tx, file *model.ConfigFileKey) (*model.ConfigFileRelease, error)
	// CreateConfigFileReleaseTx 创建配置文件发布
	CreateConfigFileReleaseTx(tx Tx, fileRelease *model.ConfigFileRelease) error
	// GetConfigFileRelease 获取配置文件发布内容，只获取 flag=0 的记录
	GetConfigFileRelease(req *model.ConfigFileReleaseKey) (*model.ConfigFileRelease, error)
	// GetConfigFileReleaseTx 在已开启的事务中获取配置文件发布内容，只获取 flag=0 的记录
	GetConfigFileReleaseTx(tx Tx, req *model.ConfigFileReleaseKey) (*model.ConfigFileRelease, error)
	// DeleteConfigFileReleaseTx 删除配置文件发布内容
	DeleteConfigFileReleaseTx(tx Tx, data *model.ConfigFileReleaseKey) error
	// ActiveConfigFileReleaseTx 指定激活发布的配置文件（激活具有排他性，同一个配置文件的所有 release 中只能有一个处于 active == true 状态）
	ActiveConfigFileReleaseTx(tx Tx, release *model.ConfigFileRelease) error
	// InactiveConfigFileReleaseTx 指定失效发布的配置文件（失效具有排他性，同一个配置文件的所有 release 中能有多个处于 active == false 状态）
	InactiveConfigFileReleaseTx(tx Tx, release *model.ConfigFileRelease) error
	// CleanConfigFileReleasesTx 清空配置文件发布
	CleanConfigFileReleasesTx(tx Tx, namespace, group, fileName string) error
	// GetMoreReleaseFile 获取最近更新的配置文件发布, 此方法用于 cache 增量更新，需要注意 modifyTime 应为数据库时间戳
	GetMoreReleaseFile(firstUpdate bool, modifyTime time.Time) ([]*model.ConfigFileRelease, error)
	// CountConfigReleases 获取一个配置文件组下的文件数量
	CountConfigReleases(namespace, group string, onlyActive bool) (uint64, error)
	// GetConfigFileBetaReleaseTx 获取灰度发布的配置文件信息
	GetConfigFileBetaReleaseTx(tx Tx, file *model.ConfigFileKey) (*model.ConfigFileRelease, error)
}

// ConfigFileReleaseHistoryStore 配置文件发布历史存储接口
type ConfigFileReleaseHistoryStore interface {
	// CreateConfigFileReleaseHistory 创建配置文件发布历史记录
	CreateConfigFileReleaseHistory(history *model.ConfigFileReleaseHistory) error
	// QueryConfigFileReleaseHistories 获取配置文件的发布历史记录
	QueryConfigFileReleaseHistories(filter map[string]string, offset, limit uint32) (uint32, []*model.ConfigFileReleaseHistory, error)
	// CleanConfigFileReleaseHistory 清理配置发布历史
	CleanConfigFileReleaseHistory(endTime time.Time, limit uint64) error
}

// ConfigFileTemplateStore config file template store
type ConfigFileTemplateStore interface {
	// QueryAllConfigFileTemplates query all config file templates
	QueryAllConfigFileTemplates() ([]*model.ConfigFileTemplate, error)
	// CreateConfigFileTemplate create config file template
	CreateConfigFileTemplate(template *model.ConfigFileTemplate) (*model.ConfigFileTemplate, error)
	// GetConfigFileTemplate get config file template by name
	GetConfigFileTemplate(name string) (*model.ConfigFileTemplate, error)
}
