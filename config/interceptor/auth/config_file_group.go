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

package config_auth

import (
	"context"
	"strconv"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/model/auth"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateConfigFileGroup 创建配置文件组
func (s *Server) CreateConfigFileGroup(ctx context.Context,
	configFileGroup *apiconfig.ConfigFileGroup) *apiconfig.ConfigResponse {
	authCtx := s.collectConfigGroupAuthContext(ctx, []*apiconfig.ConfigFileGroup{configFileGroup},
		authcommon.Create, authcommon.CreateConfigFileGroup)

	// 验证 token 信息
	if _, err := s.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.nextServer.CreateConfigFileGroup(ctx, configFileGroup)
}

// QueryConfigFileGroups 查询配置文件组
func (s *Server) QueryConfigFileGroups(ctx context.Context,
	filter map[string]string) *apiconfig.ConfigBatchQueryResponse {
	authCtx := s.collectConfigGroupAuthContext(ctx, nil, authcommon.Read, authcommon.DescribeConfigFileGroups)

	if _, err := s.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigBatchQueryResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	ctx = cachetypes.AppendConfigGroupPredicate(ctx, func(ctx context.Context, cfg *model.ConfigFileGroup) bool {
		return s.policySvr.GetAuthChecker().ResourcePredicate(authCtx, &authcommon.ResourceEntry{
			Type:     apisecurity.ResourceType_ConfigGroups,
			ID:       strconv.FormatUint(cfg.Id, 10),
			Metadata: cfg.Metadata,
		})
	})
	authCtx.SetRequestContext(ctx)

	resp := s.nextServer.QueryConfigFileGroups(ctx, filter)
	if len(resp.ConfigFileGroups) != 0 {
		for index := range resp.ConfigFileGroups {
			item := resp.ConfigFileGroups[index]
			authCtx.SetAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_ConfigGroups: {
					{
						Type:     apisecurity.ResourceType_ConfigGroups,
						ID:       strconv.FormatUint(item.GetId().GetValue(), 10),
						Metadata: item.Metadata,
					},
				},
			})

			// 检查 write 操作权限
			authCtx.SetMethod([]authcommon.ServerFunctionName{authcommon.UpdateConfigFileGroup})
			// 如果检查不通过，设置 editable 为 false
			if _, err := s.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
				item.Editable = utils.NewBoolValue(false)
			}

			// 检查 delete 操作权限
			authCtx.SetMethod([]authcommon.ServerFunctionName{authcommon.DeleteConfigFileGroup})
			// 如果检查不通过，设置 editable 为 false
			if _, err := s.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
				item.Deleteable = utils.NewBoolValue(false)
			}
		}
	}
	return resp
}

// DeleteConfigFileGroup 删除配置文件组
func (s *Server) DeleteConfigFileGroup(
	ctx context.Context, namespace, name string) *apiconfig.ConfigResponse {
	authCtx := s.collectConfigGroupAuthContext(ctx, []*apiconfig.ConfigFileGroup{{Name: utils.NewStringValue(name),
		Namespace: utils.NewStringValue(namespace)}}, auth.Delete, auth.DeleteConfigFileGroup)

	if _, err := s.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigResponse(auth.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.nextServer.DeleteConfigFileGroup(ctx, namespace, name)
}

// UpdateConfigFileGroup 更新配置文件组
func (s *Server) UpdateConfigFileGroup(ctx context.Context,
	configFileGroup *apiconfig.ConfigFileGroup) *apiconfig.ConfigResponse {
	authCtx := s.collectConfigGroupAuthContext(ctx, []*apiconfig.ConfigFileGroup{configFileGroup},
		auth.Modify, auth.UpdateConfigFileGroup)

	if _, err := s.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigResponse(auth.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.nextServer.UpdateConfigFileGroup(ctx, configFileGroup)
}
