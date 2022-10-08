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

// CreateRateLimits creates rate limits for a namespace.
func (svr *serverAuthAbility) CreateRateLimits(ctx context.Context, reqs []*api.Rule) *api.BatchWriteResponse {
	authCtx := svr.collectRateLimitAuthContext(ctx, reqs, model.Create, "CreateRateLimits")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.CreateRateLimits(ctx, reqs)
}

// DeleteRateLimits deletes rate limits for a namespace.
func (svr *serverAuthAbility) DeleteRateLimits(ctx context.Context, reqs []*api.Rule) *api.BatchWriteResponse {
	authCtx := svr.collectRateLimitAuthContext(ctx, reqs, model.Delete, "DeleteRateLimits")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.DeleteRateLimits(ctx, reqs)
}

// UpdateRateLimits updates rate limits for a namespace.
func (svr *serverAuthAbility) UpdateRateLimits(ctx context.Context, reqs []*api.Rule) *api.BatchWriteResponse {
	authCtx := svr.collectRateLimitAuthContext(ctx, reqs, model.Modify, "UpdateRateLimits")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.UpdateRateLimits(ctx, reqs)
}

// EnableRateLimits 启用限流规则
func (svr *serverAuthAbility) EnableRateLimits(ctx context.Context, reqs []*api.Rule) *api.BatchWriteResponse {
	authCtx := svr.collectRateLimitAuthContext(ctx, nil, model.Read, "EnableRateLimits")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.EnableRateLimits(ctx, reqs)
}

// GetRateLimits gets rate limits for a namespace.
func (svr *serverAuthAbility) GetRateLimits(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	authCtx := svr.collectRateLimitAuthContext(ctx, nil, model.Read, "GetRateLimits")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchQueryResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.GetRateLimits(ctx, query)
}
