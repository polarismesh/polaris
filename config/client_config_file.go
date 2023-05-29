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
	"encoding/base64"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/rsa"
	"github.com/polarismesh/polaris/common/utils"
	utils2 "github.com/polarismesh/polaris/config/utils"
)

type (
	compareFunction func(clientConfigFile *apiconfig.ClientConfigFileInfo, cacheEntry *cache.Entry) bool
)

// GetConfigFileForClient 从缓存中获取配置文件，如果客户端的版本号大于服务端，则服务端重新加载缓存
func (s *Server) GetConfigFileForClient(ctx context.Context,
	client *apiconfig.ClientConfigFileInfo) *apiconfig.ConfigClientResponse {
	namespace := client.GetNamespace().GetValue()
	group := client.GetGroup().GetValue()
	fileName := client.GetFileName().GetValue()
	clientVersion := client.GetVersion().GetValue()
	publicKey := client.GetPublicKey().GetValue()

	if namespace == "" || group == "" || fileName == "" {
		return api.NewConfigClientResponseWithMessage(
			apimodel.Code_BadRequest, "namespace & group & fileName can not be empty")
	}

	requestID := utils.ParseRequestID(ctx)

	log.Info("[Config][Service] load config file from cache.",
		utils.ZapRequestID(requestID),
		utils.ZapNamespace(namespace),
		utils.ZapGroup(group),
		utils.ZapFileName(fileName),
		zap.String("publicKey", publicKey))

	// 从缓存中获取配置内容
	entry, err := s.fileCache.GetOrLoadIfAbsent(namespace, group, fileName)
	if err != nil {
		log.Error("[Config][Service] get or load config file from cache error.",
			zap.String("requestId", requestID),
			zap.Error(err))
		return api.NewConfigClientResponseWithMessage(
			apimodel.Code_ExecuteException, "load config file error")
	}

	if entry.Empty {
		return api.NewConfigClientResponse(apimodel.Code_NotFoundResource, nil)
	}

	// 客户端版本号大于服务端版本号，服务端需要重新加载缓存
	if clientVersion > entry.Version {
		entry, err = s.fileCache.ReLoad(namespace, group, fileName)
		if err != nil {
			log.Error("[Config][Service] reload config file error.",
				zap.String("requestId", requestID),
				zap.Error(err))

			return api.NewConfigClientResponseWithMessage(
				apimodel.Code_ExecuteException, "load config file error")
		}
	}

	log.Info("[Config][Client] client get config file success.",
		zap.String("requestId", requestID),
		zap.String("client", utils.ParseClientAddress(ctx)),
		zap.String("file", fileName),
		zap.Uint64("version", entry.Version))

	configFile, err := transferEntry2APIModel(client, entry)
	if err != nil {
		log.Error("[Config][Service] transfer entry to api model error.",
			zap.String("requestId", requestID),
			zap.Error(err))
		return api.NewConfigClientResponseWithMessage(
			apimodel.Code_ExecuteException, "transfer entry to api model error")
	}
	return api.NewConfigClientResponse(apimodel.Code_ExecuteSuccess, configFile)
}

// CreateConfigFileFromClient 调用config_file接口获取配置文件
func (s *Server) CreateConfigFileFromClient(ctx context.Context,
	client *apiconfig.ConfigFile) *apiconfig.ConfigClientResponse {
	configResponse := s.CreateConfigFile(ctx, client)
	return api.NewConfigClientResponseFromConfigResponse(configResponse)
}

// UpdateConfigFileFromClient 调用config_file接口更新配置文件
func (s *Server) UpdateConfigFileFromClient(ctx context.Context,
	client *apiconfig.ConfigFile) *apiconfig.ConfigClientResponse {
	configResponse := s.UpdateConfigFile(ctx, client)
	return api.NewConfigClientResponseFromConfigResponse(configResponse)
}

// PublishConfigFileFromClient 调用config_file_release接口删除配置文件
func (s *Server) PublishConfigFileFromClient(ctx context.Context,
	client *apiconfig.ConfigFileRelease) *apiconfig.ConfigClientResponse {
	configResponse := s.PublishConfigFile(ctx, client)
	return api.NewConfigClientResponseFromConfigResponse(configResponse)
}

func (s *Server) WatchConfigFiles(ctx context.Context,
	request *apiconfig.ClientWatchConfigFileRequest) (WatchCallback, error) {
	clientAddr := utils.ParseClientAddress(ctx)
	watchFiles := request.GetWatchFiles()
	// 2. 检查客户端是否有版本落后
	if resp := s.doCheckClientConfigFile(ctx, watchFiles, compareByVersion); resp.Code.GetValue() != api.DataNoChange {
		return func() *apiconfig.ConfigClientResponse {
			return resp
		}, nil
	}

	// 3. 监听配置变更，hold 请求 30s，30s 内如果有配置发布，则响应请求
	clientId := clientAddr + "@" + utils.NewUUID()[0:8]

	finishChan := s.ConnManager().AddConn(clientId, watchFiles)

	return func() *apiconfig.ConfigClientResponse {
		return <-finishChan
	}, nil
}

func compareByVersion(clientConfigFile *apiconfig.ClientConfigFileInfo, cacheEntry *cache.Entry) bool {
	return !cacheEntry.Empty && clientConfigFile.Version.GetValue() < cacheEntry.Version
}

func compareByMD5(clientConfigFile *apiconfig.ClientConfigFileInfo, cacheEntry *cache.Entry) bool {
	return clientConfigFile.Md5.GetValue() != cacheEntry.Md5
}

func (s *Server) doCheckClientConfigFile(ctx context.Context, configFiles []*apiconfig.ClientConfigFileInfo,
	compartor compareFunction) *apiconfig.ConfigClientResponse {
	if len(configFiles) == 0 {
		return api.NewConfigClientResponse(apimodel.Code_InvalidWatchConfigFileFormat, nil)
	}

	requestID := utils.ParseRequestID(ctx)
	for _, configFile := range configFiles {
		namespace := configFile.Namespace.GetValue()
		group := configFile.Group.GetValue()
		fileName := configFile.FileName.GetValue()

		if namespace == "" || group == "" || fileName == "" {
			return api.NewConfigClientResponseWithMessage(apimodel.Code_BadRequest,
				"namespace & group & fileName can not be empty")
		}

		// 从缓存中获取最新的配置文件信息
		entry, err := s.fileCache.GetOrLoadIfAbsent(namespace, group, fileName)

		if err != nil {
			log.Error("[Config][Service] get or load config file from cache error.",
				zap.String("requestId", requestID),
				zap.String("fileName", fileName),
				zap.Error(err))

			return api.NewConfigClientResponse(apimodel.Code_ExecuteException, nil)
		}

		if compartor(configFile, entry) {
			return utils2.GenConfigFileResponse(namespace, group, fileName, "", entry.Md5, entry.Version)
		}
	}

	return api.NewConfigClientResponse(apimodel.Code_DataNoChange, nil)
}

func transferEntry2APIModel(client *apiconfig.ClientConfigFileInfo,
	entry *cache.Entry) (*apiconfig.ClientConfigFileInfo, error) {
	namespace := client.GetNamespace().GetValue()
	group := client.GetGroup().GetValue()
	fileName := client.GetFileName().GetValue()
	publicKey := client.GetPublicKey().GetValue()

	configFile := &apiconfig.ClientConfigFileInfo{
		Namespace: utils.NewStringValue(namespace),
		Group:     utils.NewStringValue(group),
		FileName:  utils.NewStringValue(fileName),
		Content:   utils.NewStringValue(entry.Content),
		Version:   utils.NewUInt64Value(entry.Version),
		Md5:       utils.NewStringValue(entry.Md5),
		Encrypted: utils.NewBoolValue(entry.Encrypted()),
	}
	for _, tag := range entry.Tags {
		if tag.Key != utils.ConfigFileTagKeyDataKey && tag.Key != utils.ConfigFileTagKeyEncryptAlgo {
			configFile.Tags = append(configFile.Tags, &apiconfig.ConfigFileTag{
				Key:   utils.NewStringValue(tag.Key),
				Value: utils.NewStringValue(tag.Value),
			})
		}
	}

	dataKey := entry.GetDataKey()
	encryptAlgo := entry.GetEncryptAlgo()
	if dataKey != "" && publicKey != "" {
		dataKeyBytes, err := base64.StdEncoding.DecodeString(dataKey)
		if err != nil {
			log.Error("[Config][Service] base64 decode data key error.",
				zap.String("dataKey", dataKey),
				zap.Error(err))
			return nil, err
		}
		cipherDataKey, err := rsa.EncryptToBase64(dataKeyBytes, publicKey)
		if err != nil {
			log.Error("[Config][Service] rsa encrypt data key error.",
				zap.String("dataKey", dataKey),
				zap.Error(err))
		}
		configFile.Tags = append(configFile.Tags,
			&apiconfig.ConfigFileTag{
				Key:   utils.NewStringValue(utils.ConfigFileTagKeyDataKey),
				Value: utils.NewStringValue(cipherDataKey),
			},
			&apiconfig.ConfigFileTag{
				Key:   utils.NewStringValue(utils.ConfigFileTagKeyEncryptAlgo),
				Value: utils.NewStringValue(encryptAlgo),
			},
		)
		return configFile, nil
	}
	return configFile, nil
}
