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

package service

import (
	"context"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateRoutingConfigs creates routing configs
func (svr *serverAuthAbility) CreateRoutingConfigs(ctx context.Context, reqs []*api.Routing) *api.BatchWriteResponse {
	authCtx := svr.collectRouteRuleAuthContext(ctx, reqs, model.Create, "CreateRoutingConfigs")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.CreateRoutingConfigs(ctx, reqs)
}

// DeleteRoutingConfigs deletes routing configs
func (svr *serverAuthAbility) DeleteRoutingConfigs(ctx context.Context, reqs []*api.Routing) *api.BatchWriteResponse {
	authCtx := svr.collectRouteRuleAuthContext(ctx, reqs, model.Delete, "DeleteRoutingConfigs")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.DeleteRoutingConfigs(ctx, reqs)
}

// UpdateRoutingConfigs updates routing configs
func (svr *serverAuthAbility) UpdateRoutingConfigs(ctx context.Context, reqs []*api.Routing) *api.BatchWriteResponse {
	authCtx := svr.collectRouteRuleAuthContext(ctx, reqs, model.Modify, "UpdateRoutingConfigs")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.UpdateRoutingConfigs(ctx, reqs)
}

// GetRoutingConfigs gets routing configs
func (svr *serverAuthAbility) GetRoutingConfigs(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	authCtx := svr.collectRouteRuleAuthContext(ctx, nil, model.Read, "GetRoutingConfigs")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchQueryResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.GetRoutingConfigs(ctx, query)
}
