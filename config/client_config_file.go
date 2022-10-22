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

	"go.uber.org/zap"

	"github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
	utils2 "github.com/polarismesh/polaris/config/utils"
)

type (
	compareFunction func(clientConfigFile *api.ClientConfigFileInfo, cacheEntry *cache.Entry) bool
)

// GetConfigFileForClient 从缓存中获取配置文件，如果客户端的版本号大于服务端，则服务端重新加载缓存
func (s *Server) GetConfigFileForClient(ctx context.Context,
	client *api.ClientConfigFileInfo) *api.ConfigClientResponse {

	namespace := client.GetNamespace().GetValue()
	group := client.GetGroup().GetValue()
	fileName := client.GetFileName().GetValue()
	clientVersion := client.GetVersion().GetValue()

	if namespace == "" || group == "" || fileName == "" {
		return api.NewConfigClientResponseWithMessage(api.BadRequest, "namespace & group & fileName can not be empty")
	}

	requestID := utils.ParseRequestID(ctx)

	log.Info("[Config][Service] load config file from cache.",
		zap.String("requestId", requestID), zap.String("namespace", namespace),
		zap.String("group", group), zap.String("file", fileName))

	// 从缓存中获取配置内容
	entry, err := s.fileCache.GetOrLoadIfAbsent(namespace, group, fileName)

	if err != nil {
		log.Error("[Config][Service] get or load config file from cache error.",
			zap.String("requestId", requestID),
			zap.Error(err))

		return api.NewConfigClientResponseWithMessage(api.ExecuteException, "load config file error")
	}

	if entry.Empty {
		return api.NewConfigClientResponse(api.NotFoundResource, nil)
	}

	// 客户端版本号大于服务端版本号，服务端需要重新加载缓存
	if clientVersion > entry.Version {
		entry, err = s.fileCache.ReLoad(namespace, group, fileName)
		if err != nil {
			log.Error("[Config][Service] reload config file error.",
				zap.String("requestId", requestID),
				zap.Error(err))

			return api.NewConfigClientResponseWithMessage(api.ExecuteException, "load config file error")
		}
	}

	log.Info("[Config][Client] client get config file success.",
		zap.String("requestId", requestID),
		zap.String("client", utils.ParseClientAddress(ctx)),
		zap.String("file", fileName),
		zap.Uint64("version", entry.Version))

	resp := utils2.GenConfigFileResponse(namespace, group, fileName, entry.Content, entry.Md5, entry.Version)
	return resp
}

func (s *Server) WatchConfigFiles(ctx context.Context,
	request *api.ClientWatchConfigFileRequest) (WatchCallback, error) {

	clientAddr := utils.ParseClientAddress(ctx)

	watchFiles := request.GetWatchFiles()
	// 2. 检查客户端是否有版本落后
	if resp := s.doCheckClientConfigFile(ctx, watchFiles, compareByVersion); resp.Code.GetValue() != api.DataNoChange {
		return func() *api.ConfigClientResponse {
			return resp
		}, nil
	}

	// 3. 监听配置变更，hold 请求 30s，30s 内如果有配置发布，则响应请求
	clientId := clientAddr + "@" + utils.NewUUID()[0:8]

	finishChan := s.ConnManager().AddConn(clientId, watchFiles)

	return func() *api.ConfigClientResponse {
		return <-finishChan
	}, nil
}

func compareByVersion(clientConfigFile *api.ClientConfigFileInfo, cacheEntry *cache.Entry) bool {
	return !cacheEntry.Empty && clientConfigFile.Version.GetValue() < cacheEntry.Version
}

func compareByMD5(clientConfigFile *api.ClientConfigFileInfo, cacheEntry *cache.Entry) bool {
	return clientConfigFile.Md5.GetValue() != cacheEntry.Md5
}

func (s *Server) doCheckClientConfigFile(ctx context.Context, configFiles []*api.ClientConfigFileInfo,
	compartor compareFunction) *api.ConfigClientResponse {
	if len(configFiles) == 0 {
		return api.NewConfigClientResponse(api.InvalidWatchConfigFileFormat, nil)
	}

	requestID := utils.ParseRequestID(ctx)

	for _, configFile := range configFiles {
		namespace := configFile.Namespace.GetValue()
		group := configFile.Group.GetValue()
		fileName := configFile.FileName.GetValue()

		if namespace == "" || group == "" || fileName == "" {
			return api.NewConfigClientResponseWithMessage(api.BadRequest,
				"namespace & group & fileName can not be empty")
		}

		// 从缓存中获取最新的配置文件信息
		entry, err := s.fileCache.GetOrLoadIfAbsent(namespace, group, fileName)

		if err != nil {
			log.Error("[Config][Service] get or load config file from cache error.",
				zap.String("requestId", requestID),
				zap.String("fileName", fileName),
				zap.Error(err))

			return api.NewConfigClientResponse(api.ExecuteException, nil)
		}

		if compartor(configFile, entry) {
			return utils2.GenConfigFileResponse(namespace, group, fileName, "", entry.Md5, entry.Version)
		}
	}

	return api.NewConfigClientResponse(api.DataNoChange, nil)
}
