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
	"fmt"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateConfigFileGroup 创建配置文件组
func (s *serverAuthability) CreateConfigFileGroup(ctx context.Context,
	configFileGroup *api.ConfigFileGroup) *api.ConfigResponse {

	authCtx := s.collectConfigGroupAuthContext(ctx, []*api.ConfigFileGroup{configFileGroup},
		model.Create, "CreateConfigFileGroup")

	// 验证 token 信息
	if _, err := s.checker.CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigFileResponseWithMessage(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.targetServer.CreateConfigFileGroup(ctx, configFileGroup)
}

// QueryConfigFileGroups 查询配置文件组
func (s *serverAuthability) QueryConfigFileGroups(ctx context.Context, namespace, groupName,
	fileName string, offset, limit uint32) *api.ConfigBatchQueryResponse {

	authCtx := s.collectConfigGroupAuthContext(ctx, []*api.ConfigFileGroup{{Name: utils.NewStringValue(groupName),
		Namespace: utils.NewStringValue(namespace)}}, model.Read, "QueryConfigFileGroups")

	if _, err := s.checker.CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigFileGroupBatchQueryResponse(convertToErrCode(err), 0, nil)
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	resp := s.targetServer.QueryConfigFileGroups(ctx, namespace, groupName, fileName, offset, limit)

	if len(resp.ConfigFileGroups) != 0 {
		principal := model.Principal{
			PrincipalID:   utils.ParseUserID(ctx),
			PrincipalRole: model.PrincipalUser,
		}
		for index := range resp.ConfigFileGroups {
			group := resp.ConfigFileGroups[index]
			editable := true
			// 如果鉴权能力没有开启，那就默认都可以进行编辑
			if s.checker.IsOpenConsoleAuth() {
				editable = s.targetServer.caches.AuthStrategy().IsResourceEditable(principal,
					api.ResourceType_ConfigGroups, fmt.Sprintf("%d", group.GetId().GetValue()))
			}
			group.Editable = utils.NewBoolValue(editable)
		}
	}

	return resp
}

// DeleteConfigFileGroup 删除配置文件组
func (s *serverAuthability) DeleteConfigFileGroup(ctx context.Context, namespace, name string) *api.ConfigResponse {

	authCtx := s.collectConfigGroupAuthContext(ctx, []*api.ConfigFileGroup{{Name: utils.NewStringValue(name),
		Namespace: utils.NewStringValue(namespace)}}, model.Delete, "DeleteConfigFileGroup")

	if _, err := s.checker.CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigFileResponseWithMessage(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.targetServer.DeleteConfigFileGroup(ctx, namespace, name)
}

// UpdateConfigFileGroup 更新配置文件组
func (s *serverAuthability) UpdateConfigFileGroup(ctx context.Context,
	configFileGroup *api.ConfigFileGroup) *api.ConfigResponse {

	authCtx := s.collectConfigGroupAuthContext(ctx, []*api.ConfigFileGroup{configFileGroup},
		model.Modify, "UpdateConfigFileGroup")

	if _, err := s.checker.CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigFileResponseWithMessage(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.targetServer.UpdateConfigFileGroup(ctx, configFileGroup)
}
