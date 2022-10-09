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
	authcommon "github.com/polarismesh/polaris/common/auth"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateInstances create instances
func (svr *serverAuthAbility) CreateInstances(ctx context.Context,
	reqs []*api.Instance) *api.BatchWriteResponse {
	authCtx := svr.collectInstanceAuthContext(ctx, reqs, model.Create, "CreateInstances")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		resp := api.NewResponseWithMsg(convertToErrCode(err), err.Error())
		batchResp := api.NewBatchWriteResponse(api.ExecuteSuccess)
		batchResp.Collect(resp)
		return batchResp
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.CreateInstances(ctx, reqs)
}

// DeleteInstances delete instances
func (svr *serverAuthAbility) DeleteInstances(ctx context.Context,
	reqs []*api.Instance) *api.BatchWriteResponse {
	authCtx := svr.collectInstanceAuthContext(ctx, reqs, model.Delete, "DeleteInstances")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		resp := api.NewResponseWithMsg(convertToErrCode(err), err.Error())
		batchResp := api.NewBatchWriteResponse(api.ExecuteSuccess)
		batchResp.Collect(resp)
		return batchResp
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.DeleteInstances(ctx, reqs)
}

// DeleteInstancesByHost 目前只允许 super account 进行数据删除
func (svr *serverAuthAbility) DeleteInstancesByHost(ctx context.Context,
	reqs []*api.Instance) *api.BatchWriteResponse {
	authCtx := svr.collectInstanceAuthContext(ctx, reqs, model.Delete, "DeleteInstancesByHost")

	if _, err := svr.authMgn.CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(convertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	if authcommon.ParseUserRole(ctx) == model.SubAccountUserRole {
		ret := api.NewBatchWriteResponse(api.ExecuteSuccess)
		ret.Collect(api.NewResponse(api.NotAllowedAccess))
		return ret
	}

	return svr.targetServer.DeleteInstancesByHost(ctx, reqs)
}

// UpdateInstances update instances
func (svr *serverAuthAbility) UpdateInstances(ctx context.Context,
	reqs []*api.Instance) *api.BatchWriteResponse {
	authCtx := svr.collectInstanceAuthContext(ctx, reqs, model.Modify, "UpdateInstances")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.UpdateInstances(ctx, reqs)
}

// UpdateInstancesIsolate update instances
func (svr *serverAuthAbility) UpdateInstancesIsolate(ctx context.Context,
	reqs []*api.Instance) *api.BatchWriteResponse {
	authCtx := svr.collectInstanceAuthContext(ctx, reqs, model.Modify, "UpdateInstancesIsolate")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.UpdateInstancesIsolate(ctx, reqs)
}

// GetInstances get instances
func (svr *serverAuthAbility) GetInstances(ctx context.Context,
	query map[string]string) *api.BatchQueryResponse {
	authCtx := svr.collectInstanceAuthContext(ctx, nil, model.Read, "GetInstances")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchQueryResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.GetInstances(ctx, query)
}

// GetInstancesCount get instances to count
func (svr *serverAuthAbility) GetInstancesCount(ctx context.Context) *api.BatchQueryResponse {
	authCtx := svr.collectInstanceAuthContext(ctx, nil, model.Read, "GetInstancesCount")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchQueryResponseWithMsg(convertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.GetInstancesCount(ctx)
}

// CleanInstance clean instance
func (svr *serverAuthAbility) CleanInstance(ctx context.Context, req *api.Instance) *api.Response {
	authCtx := svr.collectInstanceAuthContext(ctx, []*api.Instance{req}, model.Delete, "CleanInstance")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.CleanInstance(ctx, req)
}

func (svr *serverAuthAbility) GetInstanceLabels(ctx context.Context, query map[string]string) *api.Response {

	authCtx := svr.collectInstanceAuthContext(ctx, nil, model.Read, "GetInstanceLabels")
	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.GetInstanceLabels(ctx, query)
}
