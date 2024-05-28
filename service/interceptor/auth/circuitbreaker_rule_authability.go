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

	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

func (svr *ServerAuthAbility) CreateCircuitBreakerRules(
	ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse {
	// TODO not support CircuitBreaker resource auth, so we set op is read
	authCtx := svr.collectCircuitBreakerRuleV2AuthContext(ctx, request, model.Read, "CreateCircuitBreakerRules")

	_, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponse(convertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.CreateCircuitBreakerRules(ctx, request)
}

func (svr *ServerAuthAbility) DeleteCircuitBreakerRules(
	ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse {
	authCtx := svr.collectCircuitBreakerRuleV2AuthContext(ctx, request, model.Read, "DeleteCircuitBreakerRules")
	_, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponse(convertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.DeleteCircuitBreakerRules(ctx, request)
}

func (svr *ServerAuthAbility) EnableCircuitBreakerRules(
	ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse {
	authCtx := svr.collectCircuitBreakerRuleV2AuthContext(ctx, request, model.Read, "EnableCircuitBreakerRules")
	_, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponse(convertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.EnableCircuitBreakerRules(ctx, request)
}

func (svr *ServerAuthAbility) UpdateCircuitBreakerRules(
	ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse {
	authCtx := svr.collectCircuitBreakerRuleV2AuthContext(ctx, request, model.Read, "UpdateCircuitBreakerRules")
	_, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponse(convertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.UpdateCircuitBreakerRules(ctx, request)
}

func (svr *ServerAuthAbility) GetCircuitBreakerRules(
	ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectCircuitBreakerRuleV2AuthContext(ctx, nil, model.Read, "GetCircuitBreakerRules")
	_, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchQueryResponse(convertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.GetCircuitBreakerRules(ctx, query)
}
