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

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateServices 批量创建服务
func (svr *ServerAuthAbility) CreateServices(
	ctx context.Context, reqs []*apiservice.Service) *apiservice.BatchWriteResponse {
	authCtx := svr.collectServiceAuthContext(ctx, reqs, model.Create, "CreateServices")

	_, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponse(convertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	// 填充 ownerID 信息数据
	ownerID := utils.ParseOwnerID(ctx)
	if len(ownerID) > 0 {
		for index := range reqs {
			req := reqs[index]
			req.Owners = utils.NewStringValue(ownerID)
		}
	}

	resp := svr.targetServer.CreateServices(ctx, reqs)
	return resp
}

// DeleteServices 批量删除服务
func (svr *ServerAuthAbility) DeleteServices(
	ctx context.Context, reqs []*apiservice.Service) *apiservice.BatchWriteResponse {
	authCtx := svr.collectServiceAuthContext(ctx, reqs, model.Delete, "DeleteServices")

	accessRes := authCtx.GetAccessResources()
	delete(accessRes, apisecurity.ResourceType_Namespaces)
	authCtx.SetAccessResources(accessRes)

	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	resp := svr.targetServer.DeleteServices(ctx, reqs)
	return resp
}

// UpdateServices 对于服务修改来说，只针对服务本身，而不需要检查命名空间
func (svr *ServerAuthAbility) UpdateServices(
	ctx context.Context, reqs []*apiservice.Service) *apiservice.BatchWriteResponse {
	authCtx := svr.collectServiceAuthContext(ctx, reqs, model.Modify, "UpdateServices")

	accessRes := authCtx.GetAccessResources()
	delete(accessRes, apisecurity.ResourceType_Namespaces)
	authCtx.SetAccessResources(accessRes)

	_, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponse(convertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.targetServer.UpdateServices(ctx, reqs)
}

// UpdateServiceToken 更新服务的 token
func (svr *ServerAuthAbility) UpdateServiceToken(
	ctx context.Context, req *apiservice.Service) *apiservice.Response {
	authCtx := svr.collectServiceAuthContext(
		ctx, []*apiservice.Service{req}, model.Modify, "UpdateServiceToken")

	accessRes := authCtx.GetAccessResources()
	delete(accessRes, apisecurity.ResourceType_Namespaces)
	authCtx.SetAccessResources(accessRes)

	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.targetServer.UpdateServiceToken(ctx, req)
}

func (svr *ServerAuthAbility) GetAllServices(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAuthContext(ctx, nil, model.Read, "GetAllServices")

	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.GetAllServices(ctx, query)
}

// GetServices 批量获取服务
func (svr *ServerAuthAbility) GetServices(
	ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAuthContext(ctx, nil, model.Read, "GetServices")

	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	resp := svr.targetServer.GetServices(ctx, query)
	if len(resp.Services) != 0 {
		principal := model.Principal{
			PrincipalID:   utils.ParseUserID(ctx),
			PrincipalRole: model.PrincipalUser,
		}
		for index := range resp.Services {
			svc := resp.Services[index]
			editable := true
			// 如果鉴权能力没有开启，那就默认都可以进行编辑
			if svr.strategyMgn.GetAuthChecker().IsOpenConsoleAuth() {
				editable = svr.Cache().AuthStrategy().IsResourceEditable(principal,
					apisecurity.ResourceType_Services, svc.Id.GetValue())
			}
			svc.Editable = utils.NewBoolValue(editable)
		}
	}
	return resp
}

// GetServicesCount 批量获取服务数量
func (svr *ServerAuthAbility) GetServicesCount(ctx context.Context) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAuthContext(ctx, nil, model.Read, "GetServicesCount")

	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.targetServer.GetServicesCount(ctx)
}

// GetServiceToken 获取服务的 token
func (svr *ServerAuthAbility) GetServiceToken(ctx context.Context, req *apiservice.Service) *apiservice.Response {
	authCtx := svr.collectServiceAuthContext(ctx, nil, model.Read, "GetServiceToken")

	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.targetServer.GetServiceToken(ctx, req)
}

// GetServiceOwner 获取服务的 owner
func (svr *ServerAuthAbility) GetServiceOwner(
	ctx context.Context, req []*apiservice.Service) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAuthContext(ctx, nil, model.Read, "GetServiceOwner")

	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.targetServer.GetServiceOwner(ctx, req)
}
