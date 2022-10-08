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
	ConfigFileTagStore
	ConfigFileTemplateStore
}

// ConfigFileGroupStore 配置文件组存储接口
type ConfigFileGroupStore interface {

	// CreateConfigFileGroup 创建配置文件组
	CreateConfigFileGroup(fileGroup *model.ConfigFileGroup) (*model.ConfigFileGroup, error)

	// GetConfigFileGroup 获取单个配置文件组
	GetConfigFileGroup(namespace, name string) (*model.ConfigFileGroup, error)

	// QueryConfigFileGroups 翻页查询配置文件组, name 为模糊匹配关键字
	QueryConfigFileGroups(namespace, name string, offset, limit uint32) (uint32, []*model.ConfigFileGroup, error)

	// DeleteConfigFileGroup 删除配置文件组
	DeleteConfigFileGroup(namespace, name string) error

	// UpdateConfigFileGroup 更新配置文件组
	UpdateConfigFileGroup(fileGroup *model.ConfigFileGroup) (*model.ConfigFileGroup, error)

	// FindConfigFileGroups 获取一组配置文件组信息
	FindConfigFileGroups(namespace string, names []string) ([]*model.ConfigFileGroup, error)

	// GetConfigFileGroupById 根据Id获取文件组信息
	GetConfigFileGroupById(id uint64) (*model.ConfigFileGroup, error)
}

// ConfigFileStore 配置文件存储接口
type ConfigFileStore interface {

	// CreateConfigFile 创建配置文件
	CreateConfigFile(tx Tx, file *model.ConfigFile) (*model.ConfigFile, error)

	// GetConfigFile 获取配置文件
	GetConfigFile(tx Tx, namespace, group, name string) (*model.ConfigFile, error)

	// QueryConfigFiles 翻页查询配置文件，group、name可为模糊匹配
	QueryConfigFiles(namespace, group, name string, offset, limit uint32) (uint32, []*model.ConfigFile, error)

	// QueryConfigFilesByGroup query config file group's files
	QueryConfigFilesByGroup(namespace, group string, offset, limit uint32) (uint32, []*model.ConfigFile, error)

	// UpdateConfigFile 更新配置文件
	UpdateConfigFile(tx Tx, file *model.ConfigFile) (*model.ConfigFile, error)

	// DeleteConfigFile 删除配置文件
	DeleteConfigFile(tx Tx, namespace, group, name string) error

	// CountByConfigFileGroup 获取一个配置文件组下的文件数量
	CountByConfigFileGroup(namespace, group string) (uint64, error)
}

// ConfigFileReleaseStore 配置文件发布存储接口
type ConfigFileReleaseStore interface {

	// CreateConfigFileRelease 创建配置文件发布
	CreateConfigFileRelease(tx Tx, fileRelease *model.ConfigFileRelease) (*model.ConfigFileRelease, error)

	// UpdateConfigFileRelease 更新配置文件发布
	UpdateConfigFileRelease(tx Tx, fileRelease *model.ConfigFileRelease) (*model.ConfigFileRelease, error)

	// GetConfigFileRelease 获取配置文件发布内容，只获取 flag=0 的记录
	GetConfigFileRelease(tx Tx, namespace, group, fileName string) (*model.ConfigFileRelease, error)

	// GetConfigFileReleaseWithAllFlag 获取配置文件发布内容，返回所有 flag 的记录
	GetConfigFileReleaseWithAllFlag(tx Tx, namespace, group, fileName string) (*model.ConfigFileRelease, error)

	// DeleteConfigFileRelease 删除配置文件发布内容
	DeleteConfigFileRelease(tx Tx, namespace, group, fileName, deleteBy string) error

	// FindConfigFileReleaseByModifyTimeAfter 获取最近更新的配置文件发布
	// 此方法用于 cache 增量更新，需要注意 modifyTime 应为数据库时间戳
	FindConfigFileReleaseByModifyTimeAfter(modifyTime time.Time) ([]*model.ConfigFileRelease, error)
}

// ConfigFileReleaseHistoryStore 配置文件发布历史存储接口
type ConfigFileReleaseHistoryStore interface {

	// CreateConfigFileReleaseHistory 创建配置文件发布历史记录
	CreateConfigFileReleaseHistory(tx Tx, fileReleaseHistory *model.ConfigFileReleaseHistory) error

	// QueryConfigFileReleaseHistories 获取配置文件的发布历史记录
	QueryConfigFileReleaseHistories(namespace, group, fileName string, offset, limit uint32,
		endId uint64) (uint32, []*model.ConfigFileReleaseHistory, error)

	// GetLatestConfigFileReleaseHistory 获取配置文件最后一次发布
	GetLatestConfigFileReleaseHistory(namespace, group, fileName string) (*model.ConfigFileReleaseHistory, error)
}

type ConfigFileTagStore interface {

	// CreateConfigFileTag 创建配置文件标签
	CreateConfigFileTag(tx Tx, fileTag *model.ConfigFileTag) error

	// QueryConfigFileByTag 通过标签查询配置文件
	QueryConfigFileByTag(namespace, group, fileName string, tags ...string) ([]*model.ConfigFileTag, error)

	// QueryTagByConfigFile 查询配置文件标签
	QueryTagByConfigFile(namespace, group, fileName string) ([]*model.ConfigFileTag, error)

	// DeleteConfigFileTag 删除配置文件标签
	DeleteConfigFileTag(tx Tx, namespace, group, fileName, key, value string) error

	// DeleteTagByConfigFile 删除配置文件标签
	DeleteTagByConfigFile(tx Tx, namespace, group, fileName string) error
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
