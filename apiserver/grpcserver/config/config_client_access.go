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

package configgrpcserver

import (
	"context"
	"github.com/google/uuid"
	"github.com/polarismesh/polaris-server/apiserver/grpcserver"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	commonlog "github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
	"go.uber.org/zap"
)

// GetConfigFile 拉取配置
func (g *ConfigGRPCServer) GetConfigFile(ctx context.Context, configFile *api.ClientConfigFileInfo) (*api.ConfigClientResponse, error) {
	ctx = grpcserver.ConvertContext(ctx)

	namespace := configFile.GetNamespace().GetValue()
	group := configFile.GetGroup().GetValue()
	fileName := configFile.GetFileName().GetValue()
	clientVersion := configFile.GetVersion().GetValue()

	response := g.configServer.Service().GetConfigFileForClient(ctx, namespace, group, fileName, clientVersion)

	var version uint64 = 0
	if response.ConfigFile != nil {
		version = response.ConfigFile.Version.GetValue()
	}

	requestId, _ := ctx.Value(utils.StringContext("request-id")).(string)
	clientAddress, _ := ctx.Value(utils.StringContext("client-address")).(string)

	commonlog.ConfigScope().Info("[Config][Client] client get config file success.",
		zap.String("requestId", requestId),
		zap.String("client", clientAddress),
		zap.String("file", fileName),
		zap.Uint64("version", version))

	return response, nil
}

// WatchConfigFiles 订阅配置变更
func (g *ConfigGRPCServer) WatchConfigFiles(ctx context.Context, watchConfigFileRequest *api.ClientWatchConfigFileRequest) (*api.ConfigClientResponse, error) {
	ctx = grpcserver.ConvertContext(ctx)
	requestId, _ := ctx.Value(utils.StringContext("request-id")).(string)
	clientAddress, _ := ctx.Value(utils.StringContext("client-address")).(string)

	clientIP := watchConfigFileRequest.GetClientIp().GetValue()
	if clientIP == "" {
		clientIP, _ = ctx.Value(utils.StringContext("client-ip")).(string)
	}

	commonlog.ConfigScope().Debug("[Config][Client] received client listener request.",
		zap.String("requestId", requestId),
		zap.String("client", clientAddress))

	watchFiles := watchConfigFileRequest.WatchFiles
	//1. 检查客户端是否有版本落后
	response := g.configServer.Service().CheckClientConfigFileByVersion(ctx, watchFiles)
	if response.Code.GetValue() != api.DataNoChange {
		return response, nil
	}

	//2. 监听配置变更，hold 请求 30s，30s 内如果有配置发布，则响应请求
	id, _ := uuid.NewUUID()
	clientId := clientAddress + "@" + id.String()[0:8]

	finishChan := make(chan *api.ConfigClientResponse)
	defer close(finishChan)

	g.configServer.ConnManager().AddConn(clientId, watchFiles, finishChan)

	//3. 阻塞等待响应
	rsp := <-finishChan

	return rsp, nil
}
