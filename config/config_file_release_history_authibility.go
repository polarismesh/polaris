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

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// GetConfigFileReleaseHistory 获取配置文件发布历史记录
func (s *serverAuthability) GetConfigFileReleaseHistory(ctx context.Context, namespace, group, fileName string, offset,
	limit uint32, endId uint64) *apiconfig.ConfigBatchQueryResponse {
	configFileReleaseHistory := &apiconfig.ConfigFileReleaseHistory{
		Namespace: utils.NewStringValue(namespace),
		Group:     utils.NewStringValue(group),
		FileName:  utils.NewStringValue(fileName),
	}
	authCtx := s.collectConfigFileReleaseHistoryAuthContext(ctx,
		[]*apiconfig.ConfigFileReleaseHistory{configFileReleaseHistory}, model.Read, "GetConfigFileReleaseHistory")

	if _, err := s.checker.CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigFileBatchQueryResponseWithMessage(convertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.targetServer.GetConfigFileReleaseHistory(ctx, namespace, group, fileName, offset, limit, endId)
}

// GetConfigFileLatestReleaseHistory 获取配置文件最后一次发布记录
func (s *serverAuthability) GetConfigFileLatestReleaseHistory(ctx context.Context, namespace, group,
	fileName string) *apiconfig.ConfigResponse {
	configFileReleaseHistory := &apiconfig.ConfigFileReleaseHistory{
		Namespace: utils.NewStringValue(namespace),
		Group:     utils.NewStringValue(group),
		FileName:  utils.NewStringValue(fileName),
	}
	authCtx := s.collectConfigFileReleaseHistoryAuthContext(ctx,
		[]*apiconfig.ConfigFileReleaseHistory{configFileReleaseHistory}, model.Read, "GetConfigFileLatestReleaseHistory")

	if _, err := s.checker.CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigFileReleaseResponseWithMessage(convertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.targetServer.GetConfigFileLatestReleaseHistory(ctx, namespace, group, fileName)
}
