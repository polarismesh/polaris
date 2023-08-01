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

package config

import (
	"context"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
)

type (
	// WatchCallback 监听回调函数
	WatchCallback func() *apiconfig.ConfigClientResponse
)

const (
	// MaxPageSize 最大分页大小
	MaxPageSize = 100
)

// ConfigFileGroupOperate 配置文件组接口
type ConfigFileGroupOperate interface {
	// CreateConfigFileGroup 创建配置文件组
	CreateConfigFileGroup(ctx context.Context, configFileGroup *apiconfig.ConfigFileGroup) *apiconfig.ConfigResponse
	// QueryConfigFileGroups 查询配置文件组
	QueryConfigFileGroups(ctx context.Context, filter map[string]string) *apiconfig.ConfigBatchQueryResponse
	// DeleteConfigFileGroup 删除配置文件组
	DeleteConfigFileGroup(ctx context.Context, namespace, name string) *apiconfig.ConfigResponse
	// UpdateConfigFileGroup 更新配置文件组
	UpdateConfigFileGroup(ctx context.Context, configFileGroup *apiconfig.ConfigFileGroup) *apiconfig.ConfigResponse
}

// ConfigFileOperate 配置文件接口
type ConfigFileOperate interface {
	// CreateConfigFile 创建配置文件
	CreateConfigFile(ctx context.Context, configFile *apiconfig.ConfigFile) *apiconfig.ConfigResponse
	// GetConfigFileBaseInfo 获取单个配置文件基础信息，不包含发布信息
	GetConfigFileBaseInfo(ctx context.Context, req *apiconfig.ConfigFile) *apiconfig.ConfigResponse
	// GetConfigFileRichInfo 获取单个配置文件基础信息，包含发布状态等信息
	GetConfigFileRichInfo(ctx context.Context, req *apiconfig.ConfigFile) *apiconfig.ConfigResponse
	// QueryConfigFilesByGroup query file group's config file
	QueryConfigFilesByGroup(ctx context.Context, filter map[string]string) *apiconfig.ConfigBatchQueryResponse
	// SearchConfigFile 按 group 和 name 模糊搜索配置文件
	SearchConfigFile(ctx context.Context, filter map[string]string) *apiconfig.ConfigBatchQueryResponse
	// UpdateConfigFile 更新配置文件
	UpdateConfigFile(ctx context.Context, configFile *apiconfig.ConfigFile) *apiconfig.ConfigResponse
	// DeleteConfigFile 删除配置文件
	DeleteConfigFile(ctx context.Context, req *apiconfig.ConfigFile) *apiconfig.ConfigResponse
	// BatchDeleteConfigFile 批量删除配置文件
	BatchDeleteConfigFile(ctx context.Context,
		configFiles []*apiconfig.ConfigFile, operator string) *apiconfig.ConfigResponse
	// ExportConfigFile 导出配置文件
	ExportConfigFile(ctx context.Context,
		configFileExport *apiconfig.ConfigFileExportRequest) *apiconfig.ConfigExportResponse
	// ImportConfigFile 导入配置文件
	ImportConfigFile(ctx context.Context,
		configFiles []*apiconfig.ConfigFile, conflictHandling string) *apiconfig.ConfigImportResponse
	// GetAllConfigEncryptAlgorithms 获取配置加密算法
	GetAllConfigEncryptAlgorithms(ctx context.Context) *apiconfig.ConfigEncryptAlgorithmResponse
}

// ConfigFileReleaseOperate 配置文件发布接口
type ConfigFileReleaseOperate interface {
	// PublishConfigFile 发布配置文件
	PublishConfigFile(ctx context.Context, configFileRelease *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse
	// GetConfigFileRelease 获取配置文件发布
	GetConfigFileRelease(ctx context.Context, req *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse
	// DeleteConfigFileReleases 删除配置文件发布内容
	DeleteConfigFileReleases(ctx context.Context, reqs []*apiconfig.ConfigFileRelease) *apiconfig.ConfigBatchWriteResponse
	// RollbackConfigFileReleases 批量回滚配置到指定版本
	RollbackConfigFileReleases(ctx context.Context, releases []*apiconfig.ConfigFileRelease) *apiconfig.ConfigBatchWriteResponse
	// GetConfigFileReleases 查询所有的配置发布版本信息
	GetConfigFileReleases(ctx context.Context, filters map[string]string) *apiconfig.ConfigBatchQueryResponse
	// GetConfigFileReleaseVersions 查询所有的配置发布版本信息
	GetConfigFileReleaseVersions(ctx context.Context, filters map[string]string) *apiconfig.ConfigBatchQueryResponse
	// GetConfigFileReleaseHistories 获取配置文件的发布历史
	GetConfigFileReleaseHistories(ctx context.Context, filter map[string]string) *apiconfig.ConfigBatchQueryResponse
}

// ConfigFileClientOperate 给客户端提供服务接口，不同的上层协议抽象的公共服务逻辑
type ConfigFileClientOperate interface {
	// GetConfigFileForClient 获取配置文件
	GetConfigFileForClient(ctx context.Context, configFile *apiconfig.ClientConfigFileInfo) *apiconfig.ConfigClientResponse
	// CreateConfigFileFromClient 调用config_file的方法创建配置文件
	CreateConfigFileFromClient(ctx context.Context, fileInfo *apiconfig.ConfigFile) *apiconfig.ConfigClientResponse
	// UpdateConfigFileFromClient 调用config_file的方法更新配置文件
	UpdateConfigFileFromClient(ctx context.Context, fileInfo *apiconfig.ConfigFile) *apiconfig.ConfigClientResponse
	// PublishConfigFileFromClient 调用config_file_release的方法发布配置文件
	PublishConfigFileFromClient(ctx context.Context, fileInfo *apiconfig.ConfigFileRelease) *apiconfig.ConfigClientResponse
	// WatchConfigFiles 客户端监听配置文件
	WatchConfigFiles(ctx context.Context, request *apiconfig.ClientWatchConfigFileRequest) (WatchCallback, error)
	// GetConfigFileNamesWithCache 获取某个配置分组下的配置文件
	GetConfigFileNamesWithCache(ctx context.Context, req *apiconfig.ConfigFileGroupRequest) *apiconfig.ConfigClientListResponse
}

// ConfigFileTemplateOperate config file template operate
type ConfigFileTemplateOperate interface {
	// GetAllConfigFileTemplates get all config file templates
	GetAllConfigFileTemplates(ctx context.Context) *apiconfig.ConfigBatchQueryResponse
	// CreateConfigFileTemplate create config file template
	CreateConfigFileTemplate(ctx context.Context, template *apiconfig.ConfigFileTemplate) *apiconfig.ConfigResponse
	// GetConfigFileTemplate get config file template
	GetConfigFileTemplate(ctx context.Context, name string) *apiconfig.ConfigResponse
}

// ConfigCenterServer 配置中心server
type ConfigCenterServer interface {
	ConfigFileGroupOperate
	ConfigFileOperate
	ConfigFileReleaseOperate
	ConfigFileClientOperate
	ConfigFileTemplateOperate
}
