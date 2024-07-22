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
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

func (svr *ServerAuthAbility) CreateFaultDetectRules(
	ctx context.Context, request []*apifault.FaultDetectRule) *apiservice.BatchWriteResponse {

	authCtx := svr.collectFaultDetectAuthContext(ctx, request, authcommon.Read, "CreateFaultDetectRules")
	if _, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.CreateFaultDetectRules(ctx, request)
}

func (svr *ServerAuthAbility) DeleteFaultDetectRules(
	ctx context.Context, request []*apifault.FaultDetectRule) *apiservice.BatchWriteResponse {

	authCtx := svr.collectFaultDetectAuthContext(ctx, request, authcommon.Read, "DeleteFaultDetectRules")
	if _, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.DeleteFaultDetectRules(ctx, request)
}

func (svr *ServerAuthAbility) UpdateFaultDetectRules(
	ctx context.Context, request []*apifault.FaultDetectRule) *apiservice.BatchWriteResponse {

	authCtx := svr.collectFaultDetectAuthContext(ctx, request, authcommon.Read, "UpdateFaultDetectRules")
	if _, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.UpdateFaultDetectRules(ctx, request)
}

func (svr *ServerAuthAbility) GetFaultDetectRules(
	ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectFaultDetectAuthContext(ctx, nil, authcommon.Read, "GetFaultDetectRules")
	if _, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.GetFaultDetectRules(ctx, query)
}
