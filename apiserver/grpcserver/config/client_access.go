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

package config

import (
	"context"
	"fmt"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"

	"github.com/polarismesh/polaris/common/metrics"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

// GetConfigFile 拉取配置
func (g *ConfigGRPCServer) GetConfigFile(ctx context.Context,
	req *apiconfig.ClientConfigFileInfo) (*apiconfig.ConfigClientResponse, error) {
	ctx = utils.ConvertGRPCContext(ctx)

	startTime := commontime.CurrentMillisecond()
	defer func() {
		plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
			ClientIP:  utils.ParseClientAddress(ctx),
			Namespace: req.GetNamespace().GetValue(),
			Resource: fmt.Sprintf("CONFIG_FILE:%s|%s|%d", req.GetGroup().GetValue(),
				req.GetFileName().GetValue(), req.GetVersion().GetValue()),
			Timestamp: startTime,
			CostTime:  commontime.CurrentMillisecond() - startTime,
		})
	}()
	response := g.configServer.GetConfigFileForClient(ctx, req)
	return response, nil
}

// CreateConfigFile 创建或更新配置
func (g *ConfigGRPCServer) CreateConfigFile(ctx context.Context,
	configFile *apiconfig.ConfigFile) (*apiconfig.ConfigClientResponse, error) {
	ctx = utils.ConvertGRPCContext(ctx)
	response := g.configServer.CreateConfigFileFromClient(ctx, configFile)
	return response, nil
}

// UpdateConfigFile 创建或更新配置
func (g *ConfigGRPCServer) UpdateConfigFile(ctx context.Context,
	configFile *apiconfig.ConfigFile) (*apiconfig.ConfigClientResponse, error) {
	ctx = utils.ConvertGRPCContext(ctx)
	response := g.configServer.UpdateConfigFileFromClient(ctx, configFile)
	return response, nil
}

// PublishConfigFile 发布配置
func (g *ConfigGRPCServer) PublishConfigFile(ctx context.Context,
	configFile *apiconfig.ConfigFileRelease) (*apiconfig.ConfigClientResponse, error) {
	ctx = utils.ConvertGRPCContext(ctx)
	response := g.configServer.PublishConfigFileFromClient(ctx, configFile)
	return response, nil
}

// WatchConfigFiles 订阅配置变更
func (g *ConfigGRPCServer) WatchConfigFiles(ctx context.Context,
	request *apiconfig.ClientWatchConfigFileRequest) (*apiconfig.ConfigClientResponse, error) {
	ctx = utils.ConvertGRPCContext(ctx)

	// 阻塞等待响应
	callback, err := g.configServer.LongPullWatchFile(ctx, request)
	if err != nil {
		return nil, err
	}
	return callback(), nil
}

func (g *ConfigGRPCServer) GetConfigFileMetadataList(ctx context.Context,
	req *apiconfig.ConfigFileGroupRequest) (*apiconfig.ConfigClientListResponse, error) {

	startTime := commontime.CurrentMillisecond()
	defer func() {
		plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
			ClientIP:  utils.ParseClientAddress(ctx),
			Namespace: req.GetConfigFileGroup().GetNamespace().GetValue(),
			Resource: fmt.Sprintf("CONFIG_FILE_LIST:%s|%s", req.GetConfigFileGroup().GetName().GetValue(),
				req.GetRevision().GetValue()),
			Timestamp: startTime,
			CostTime:  commontime.CurrentMillisecond() - startTime,
		})
	}()

	ctx = utils.ConvertGRPCContext(ctx)
	return g.configServer.GetConfigFileNamesWithCache(ctx, req), nil
}
