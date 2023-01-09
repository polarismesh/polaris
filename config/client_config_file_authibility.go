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

// GetConfigFileForClient 从缓存中获取配置文件，如果客户端的版本号大于服务端，则服务端重新加载缓存
func (s *serverAuthability) GetConfigFileForClient(ctx context.Context,
	fileInfo *apiconfig.ClientConfigFileInfo) *apiconfig.ConfigClientResponse {
	return s.targetServer.GetConfigFileForClient(ctx, fileInfo)
}

// WatchConfigFiles 监听配置文件变化
func (s *serverAuthability) WatchConfigFiles(ctx context.Context,
	request *apiconfig.ClientWatchConfigFileRequest) (WatchCallback, error) {
	return s.targetServer.WatchConfigFiles(ctx, request)
}
