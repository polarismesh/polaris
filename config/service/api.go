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

package service

import (
	"context"

	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
)

const (
	MaxPageSize = 100
)

// API 配置模块服务接口
type API interface {
	// StartTxAndSetToContext 创建一个事务并放到 context 里
	StartTxAndSetToContext(ctx context.Context) (store.Tx, context.Context, error)

	ConfigFileGroupAPI
	ConfigFileAPI
	ConfigFileReleaseAPI
	ConfigFileReleaseHistoryAPI
	ConfigFileClientAPI
}

// ConfigFileGroupAPI 配置文件组接口
type ConfigFileGroupAPI interface {
	// CreateConfigFileGroup 创建配置文件组
	CreateConfigFileGroup(ctx context.Context, configFileGroup *api.ConfigFileGroup) *api.ConfigResponse

	// CreateConfigFileGroupIfAbsent 如果不存在则创建配置文件组
	CreateConfigFileGroupIfAbsent(ctx context.Context, configFileGroup *api.ConfigFileGroup) *api.ConfigResponse

	// QueryConfigFileGroups 查询配置文件组, namespace 为完全匹配，groupName 为模糊匹配, fileName 为模糊匹配文件名
	QueryConfigFileGroups(ctx context.Context, namespace, groupName, fileName string, offset, limit uint32) *api.ConfigBatchQueryResponse

	// DeleteConfigFileGroup 删除配置文件组
	DeleteConfigFileGroup(ctx context.Context, namespace, name string) *api.ConfigResponse

	// UpdateConfigFileGroup 更新配置文件组
	UpdateConfigFileGroup(ctx context.Context, configFileGroup *api.ConfigFileGroup) *api.ConfigResponse
}

// ConfigFileAPI 配置文件接口
type ConfigFileAPI interface {
	// CreateConfigFile 创建配置文件
	CreateConfigFile(ctx context.Context, configFile *api.ConfigFile) *api.ConfigResponse

	// GetConfigFileBaseInfo 获取单个配置文件基础信息，不包含发布信息
	GetConfigFileBaseInfo(ctx context.Context, namespace, group, name string) *api.ConfigResponse

	// GetConfigFileRichInfo 获取单个配置文件基础信息，包含发布状态等信息
	GetConfigFileRichInfo(ctx context.Context, namespace, group, name string) *api.ConfigResponse

	// SearchConfigFile 按 group 和 name 模糊搜索配置文件
	SearchConfigFile(ctx context.Context, namespace, group, name, tags string, offset, limit uint32) *api.ConfigBatchQueryResponse

	// UpdateConfigFile 更新配置文件
	UpdateConfigFile(ctx context.Context, configFile *api.ConfigFile) *api.ConfigResponse

	// DeleteConfigFile 删除配置文件
	DeleteConfigFile(ctx context.Context, namespace, group, name, deleteBy string) *api.ConfigResponse

	// BatchDeleteConfigFile 批量删除配置文件
	BatchDeleteConfigFile(ctx context.Context, configFiles []*api.ConfigFile, operator string) *api.ConfigResponse
}

// ConfigFileReleaseAPI 配置文件发布接口
type ConfigFileReleaseAPI interface {
	// PublishConfigFile 发布配置文件
	PublishConfigFile(ctx context.Context, configFileRelease *api.ConfigFileRelease) *api.ConfigResponse

	// GetConfigFileRelease 获取配置文件发布
	GetConfigFileRelease(ctx context.Context, namespace, group, fileName string) *api.ConfigResponse

	// DeleteConfigFileRelease 删除配置文件发布内容
	DeleteConfigFileRelease(ctx context.Context, namespace, group, fileName, deleteBy string) *api.ConfigResponse
}

// ConfigFileReleaseHistoryAPI 配置文件发布历史接口
type ConfigFileReleaseHistoryAPI interface {
	// RecordConfigFileReleaseHistory 记录发布
	RecordConfigFileReleaseHistory(ctx context.Context, fileRelease *model.ConfigFileRelease, releaseType, status string)

	// GetConfigFileReleaseHistory 获取配置文件的发布历史
	GetConfigFileReleaseHistory(ctx context.Context, namespace, group, fileName string, offset, limit uint32, endId uint64) *api.ConfigBatchQueryResponse

	// GetConfigFileLatestReleaseHistory 获取最后一次发布记录
	GetConfigFileLatestReleaseHistory(ctx context.Context, namespace, group, fileName string) *api.ConfigResponse
}

// ConfigFileTagAPI 配置文件标签相关的接口
type ConfigFileTagAPI interface {
	// CreateConfigFileTags 创建配置文件标签，tags 格式：k1,v1,k2,v2,k3,v3...
	CreateConfigFileTags(ctx context.Context, namespace, group, fileName, operator string, tags ...string) error

	// QueryConfigFileByTags 通过标签查询配置文件, 多个标签之间或的关系，tags 格式：k1,v1,k2,v2,k3,v3...
	QueryConfigFileByTags(ctx context.Context, namespace, group, fileName string, offset, limit uint32, tags ...string) (int, []*model.ConfigFileTag, error)

	// QueryTagsByConfigFileWithAPIModels 通过标签查询配置文件，返回 APIModel 对象
	QueryTagsByConfigFileWithAPIModels(ctx context.Context, namespace, group, fileName string) ([]*api.ConfigFileTag, error)

	// QueryTagsByConfigFile 查询配置文件的标签
	QueryTagsByConfigFile(ctx context.Context, namespace, group, fileName string) ([]*model.ConfigFileTag, error)

	// DeleteTagByConfigFile 删除配置文件标签
	DeleteTagByConfigFile(ctx context.Context, namespace, group, fileName string) error
}

// ConfigFileClientAPI 给客户端提供服务接口，不同的上层协议抽象的公共服务逻辑
type ConfigFileClientAPI interface {
	// CheckClientConfigFileByVersion 通过比较版本号来检查客户端版本是否落后
	CheckClientConfigFileByVersion(ctx context.Context, configFiles []*api.ClientConfigFileInfo) *api.ConfigClientResponse

	// CheckClientConfigFileByMd5 通过比较md5来检查客户端版本是否落后
	CheckClientConfigFileByMd5(ctx context.Context, configFiles []*api.ClientConfigFileInfo) *api.ConfigClientResponse

	// GetConfigFileForClient 获取配置文件
	GetConfigFileForClient(ctx context.Context, namespace, group, fileName string, clientVersion uint64) *api.ConfigClientResponse
}

// Impl 服务接口实现类
type Impl struct {
	API
	storage store.Store
	cache   *cache.FileCache
}

// NewServiceImpl 新建配置中心服务实现类
func NewServiceImpl(storage store.Store, cache *cache.FileCache) API {
	return &Impl{
		storage: storage,
		cache:   cache,
	}
}
