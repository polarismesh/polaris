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

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/specification/source/go/api/v1/security"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
)

// CreateLaneGroups 批量创建泳道组
func (svr *Server) CreateLaneGroups(ctx context.Context, reqs []*apitraffic.LaneGroup) *apiservice.BatchWriteResponse {

	authCtx := svr.collectLaneRuleAuthContext(ctx, reqs, authcommon.Create, authcommon.CreateLaneGroups)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.CreateLaneGroups(ctx, reqs)
}

// UpdateLaneGroups 批量更新泳道组
func (svr *Server) UpdateLaneGroups(ctx context.Context, reqs []*apitraffic.LaneGroup) *apiservice.BatchWriteResponse {
	authCtx := svr.collectLaneRuleAuthContext(ctx, reqs, authcommon.Modify, authcommon.UpdateLaneGroups)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.UpdateLaneGroups(ctx, reqs)
}

// DeleteLaneGroups 批量删除泳道组
func (svr *Server) DeleteLaneGroups(ctx context.Context, reqs []*apitraffic.LaneGroup) *apiservice.BatchWriteResponse {
	authCtx := svr.collectLaneRuleAuthContext(ctx, reqs, authcommon.Delete, authcommon.DeleteLaneGroups)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.DeleteLaneGroups(ctx, reqs)
}

// GetLaneGroups 查询泳道组列表
func (svr *Server) GetLaneGroups(ctx context.Context, filter map[string]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectFaultDetectAuthContext(ctx, nil, authcommon.Read, authcommon.DescribeFaultDetectRules)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	ctx = cachetypes.AppendLaneRulePredicate(ctx, func(ctx context.Context, cbr *model.LaneGroupProto) bool {
		return svr.policySvr.GetAuthChecker().ResourcePredicate(authCtx, &authcommon.ResourceEntry{
			Type:     security.ResourceType_LaneRules,
			ID:       cbr.ID,
			Metadata: cbr.Proto.Metadata,
		})
	})
	authCtx.SetRequestContext(ctx)

	return svr.nextSvr.GetLaneGroups(ctx, filter)
}

// collectLaneRuleAuthContext 收集全链路灰度规则
func (svr *Server) collectLaneRuleAuthContext(ctx context.Context,
	req []*apitraffic.LaneGroup, resourceOp authcommon.ResourceOperation, methodName authcommon.ServerFunctionName) *authcommon.AcquireContext {

	resources := make([]authcommon.ResourceEntry, 0, len(req))
	for i := range req {
		saveRule := svr.Cache().LaneRule().GetRule(req[i].GetId())
		if saveRule != nil {
			resources = append(resources, authcommon.ResourceEntry{
				Type:     apisecurity.ResourceType_LaneRules,
				ID:       saveRule.ID,
				Metadata: saveRule.Labels,
			})
		}
	}

	return authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(resourceOp),
		authcommon.WithModule(authcommon.DiscoverModule),
		authcommon.WithMethod(methodName),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
			apisecurity.ResourceType_LaneRules: resources,
		}),
	)
}
