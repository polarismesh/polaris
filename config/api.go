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

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
)

const (
	MaxPageSize = 100
)

// ConfigFileGroupAPI 配置文件组接口
type ConfigFileGroupOperate interface {
	// CreateConfigFileGroup 创建配置文件组
	CreateConfigFileGroup(ctx context.Context, configFileGroup *api.ConfigFileGroup) *api.ConfigResponse

	// QueryConfigFileGroups 查询配置文件组, namespace 为完全匹配，groupName 为模糊匹配, fileName 为模糊匹配文件名
	QueryConfigFileGroups(ctx context.Context, namespace, groupName, fileName string, offset, limit uint32) *api.ConfigBatchQueryResponse

	// DeleteConfigFileGroup 删除配置文件组
	DeleteConfigFileGroup(ctx context.Context, namespace, name string) *api.ConfigResponse

	// UpdateConfigFileGroup 更新配置文件组
	UpdateConfigFileGroup(ctx context.Context, configFileGroup *api.ConfigFileGroup) *api.ConfigResponse
}

// ConfigFileAPI 配置文件接口
type ConfigFileOperate interface {
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

// ConfigFileReleaseOperate 配置文件发布接口
type ConfigFileReleaseOperate interface {
	// PublishConfigFile 发布配置文件
	PublishConfigFile(ctx context.Context, configFileRelease *api.ConfigFileRelease) *api.ConfigResponse

	// GetConfigFileRelease 获取配置文件发布
	GetConfigFileRelease(ctx context.Context, namespace, group, fileName string) *api.ConfigResponse

	// DeleteConfigFileRelease 删除配置文件发布内容
	DeleteConfigFileRelease(ctx context.Context, namespace, group, fileName, deleteBy string) *api.ConfigResponse
}

// ConfigFileReleaseHistoryOperate 配置文件发布历史接口
type ConfigFileReleaseHistoryOperate interface {
	// RecordConfigFileReleaseHistory 记录发布
	RecordConfigFileReleaseHistory(ctx context.Context, fileRelease *model.ConfigFileRelease, releaseType, status string)

	// GetConfigFileReleaseHistory 获取配置文件的发布历史
	GetConfigFileReleaseHistory(ctx context.Context, namespace, group, fileName string, offset, limit uint32, endId uint64) *api.ConfigBatchQueryResponse

	// GetConfigFileLatestReleaseHistory 获取最后一次发布记录
	GetConfigFileLatestReleaseHistory(ctx context.Context, namespace, group, fileName string) *api.ConfigResponse
}

// ConfigFileClientAPI 给客户端提供服务接口，不同的上层协议抽象的公共服务逻辑
type ConfigFileClientOperate interface {
	// GetConfigFileForClient 获取配置文件
	GetConfigFileForClient(ctx context.Context, configFile *api.ClientConfigFileInfo) *api.ConfigClientResponse

	// WatchConfigFiles 客户端监听配置文件
	WatchConfigFiles(ctx context.Context,
		request *api.ClientWatchConfigFileRequest) (func() *api.ConfigClientResponse, error)
}

// ConfigCenterServer 配置中心server
type ConfigCenterServer interface {
	ConfigFileGroupOperate
	ConfigFileOperate
	ConfigFileReleaseOperate
	ConfigFileReleaseHistoryOperate
	ConfigFileClientOperate
}
