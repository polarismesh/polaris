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

// CreateConfigFileGroup 创建配置文件组
func (cs *serverAuthibility) CreateConfigFileGroup(ctx context.Context,
	configFileGroup *api.ConfigFileGroup) *api.ConfigResponse {

	authCtx := cs.collectBaseTokenInfo(ctx)
	if err := cs.authChecker.VerifyCredential(authCtx); err != nil {
		return api.NewConfigFileResponseWithMessage(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()

	return cs.targetServer.CreateConfigFileGroup(ctx, configFileGroup)
}

// QueryConfigFileGroups 查询配置文件组
func (cs *serverAuthibility) QueryConfigFileGroups(ctx context.Context, namespace, groupName,
	fileName string, offset, limit uint32) *api.ConfigBatchQueryResponse {

	return cs.targetServer.QueryConfigFileGroups(ctx, namespace, groupName, fileName, offset, limit)
}

// DeleteConfigFileGroup 删除配置文件组
func (cs *serverAuthibility) DeleteConfigFileGroup(ctx context.Context, namespace, name string) *api.ConfigResponse {

	authCtx := cs.collectBaseTokenInfo(ctx)
	if err := cs.authChecker.VerifyCredential(authCtx); err != nil {
		return api.NewConfigFileResponseWithMessage(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()

	return cs.targetServer.DeleteConfigFileGroup(ctx, namespace, name)
}

// UpdateConfigFileGroup 更新配置文件组
func (cs *serverAuthibility) UpdateConfigFileGroup(ctx context.Context,
	configFileGroup *api.ConfigFileGroup) *api.ConfigResponse {

	authCtx := cs.collectBaseTokenInfo(ctx)
	if err := cs.authChecker.VerifyCredential(authCtx); err != nil {
		return api.NewConfigFileResponseWithMessage(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()

	return cs.targetServer.UpdateConfigFileGroup(ctx, configFileGroup)
}
