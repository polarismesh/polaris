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
 * CONDITIONS OF ANY KIND, either express or serverAuthibilityied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package config

import (
	"context"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

// PublishConfigFile 发布配置文件
func (s *serverAuthibility) PublishConfigFile(ctx context.Context,
	configFileRelease *api.ConfigFileRelease) *api.ConfigResponse {

	authCtx := s.collectBaseTokenInfo(ctx)
	if err := s.authChecker.VerifyCredential(authCtx); err != nil {
		return api.NewConfigFileResponseWithMessage(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()

	return s.targetServer.PublishConfigFile(ctx, configFileRelease)
}

// GetConfigFileRelease 获取配置文件发布内容
func (s *serverAuthibility) GetConfigFileRelease(ctx context.Context,
	namespace, group, fileName string) *api.ConfigResponse {

	return s.targetServer.GetConfigFileRelease(ctx, namespace, group, fileName)
}

// DeleteConfigFileRelease 删除配置文件发布，删除配置文件的时候，同步删除配置文件发布数据
func (s *serverAuthibility) DeleteConfigFileRelease(ctx context.Context, namespace,
	group, fileName, deleteBy string) *api.ConfigResponse {

	return s.targetServer.DeleteConfigFileRelease(ctx, namespace, group, fileName, deleteBy)
}
