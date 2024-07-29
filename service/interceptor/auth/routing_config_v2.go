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

package service_auth

import (
	"context"

	"github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateRoutingConfigsV2 批量创建路由配置
func (svr *Server) CreateRoutingConfigsV2(ctx context.Context,
	req []*apitraffic.RouteRule) *apiservice.BatchWriteResponse {

	// TODO not support RouteRuleV2 resource auth, so we set op is read
	authCtx := svr.collectRouteRuleV2AuthContext(ctx, req, authcommon.Read, authcommon.CreateRouteRules)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.CreateRoutingConfigsV2(ctx, req)
}

// DeleteRoutingConfigsV2 批量删除路由配置
func (svr *Server) DeleteRoutingConfigsV2(ctx context.Context,
	req []*apitraffic.RouteRule) *apiservice.BatchWriteResponse {

	authCtx := svr.collectRouteRuleV2AuthContext(ctx, req, authcommon.Read, authcommon.DeleteRouteRules)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.DeleteRoutingConfigsV2(ctx, req)
}

// UpdateRoutingConfigsV2 批量更新路由配置
func (svr *Server) UpdateRoutingConfigsV2(ctx context.Context,
	req []*apitraffic.RouteRule) *apiservice.BatchWriteResponse {

	authCtx := svr.collectRouteRuleV2AuthContext(ctx, req, authcommon.Read, authcommon.UpdateRouteRules)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.UpdateRoutingConfigsV2(ctx, req)
}

// EnableRoutings batch enable routing rules
func (svr *Server) EnableRoutings(ctx context.Context,
	req []*apitraffic.RouteRule) *apiservice.BatchWriteResponse {

	authCtx := svr.collectRouteRuleV2AuthContext(ctx, req, authcommon.Read, authcommon.EnableRouteRules)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.EnableRoutings(ctx, req)
}

// QueryRoutingConfigsV2 提供给OSS的查询路由配置的接口
func (svr *Server) QueryRoutingConfigsV2(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectRouteRuleV2AuthContext(ctx, nil, authcommon.Read, authcommon.DescribeRouteRules)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	cachetypes.AppendRouterRulePredicate(ctx, func(ctx context.Context, cbr *model.ExtendRouterConfig) bool {
		return svr.policySvr.GetAuthChecker().ResourcePredicate(authCtx, &authcommon.ResourceEntry{
			Type:     security.ResourceType_RouteRules,
			ID:       cbr.ID,
			Metadata: cbr.Metadata,
		})
	})

	return svr.nextSvr.QueryRoutingConfigsV2(ctx, query)
}
