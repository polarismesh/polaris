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
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateServices 批量创建服务
func (svr *ServerAuthAbility) CreateServices(
	ctx context.Context, reqs []*apiservice.Service) *apiservice.BatchWriteResponse {
	authCtx := svr.collectServiceAuthContext(ctx, reqs, authcommon.Create, "CreateServices")

	_, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx)
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

	resp := svr.nextSvr.CreateServices(ctx, reqs)
	return resp
}

// DeleteServices 批量删除服务
func (svr *ServerAuthAbility) DeleteServices(
	ctx context.Context, reqs []*apiservice.Service) *apiservice.BatchWriteResponse {
	authCtx := svr.collectServiceAuthContext(ctx, reqs, authcommon.Delete, "DeleteServices")

	accessRes := authCtx.GetAccessResources()
	delete(accessRes, apisecurity.ResourceType_Namespaces)
	authCtx.SetAccessResources(accessRes)

	if _, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	resp := svr.nextSvr.DeleteServices(ctx, reqs)
	return resp
}

// UpdateServices 对于服务修改来说，只针对服务本身，而不需要检查命名空间
func (svr *ServerAuthAbility) UpdateServices(
	ctx context.Context, reqs []*apiservice.Service) *apiservice.BatchWriteResponse {
	authCtx := svr.collectServiceAuthContext(ctx, reqs, authcommon.Modify, "UpdateServices")

	accessRes := authCtx.GetAccessResources()
	delete(accessRes, apisecurity.ResourceType_Namespaces)
	authCtx.SetAccessResources(accessRes)

	_, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponse(convertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.UpdateServices(ctx, reqs)
}

// UpdateServiceToken 更新服务的 token
func (svr *ServerAuthAbility) UpdateServiceToken(
	ctx context.Context, req *apiservice.Service) *apiservice.Response {
	authCtx := svr.collectServiceAuthContext(
		ctx, []*apiservice.Service{req}, authcommon.Modify, "UpdateServiceToken")

	accessRes := authCtx.GetAccessResources()
	delete(accessRes, apisecurity.ResourceType_Namespaces)
	authCtx.SetAccessResources(accessRes)

	if _, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.UpdateServiceToken(ctx, req)
}

func (svr *ServerAuthAbility) GetAllServices(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAuthContext(ctx, nil, authcommon.Read, "GetAllServices")

	if _, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.GetAllServices(ctx, query)
}

// GetServices 批量获取服务
func (svr *ServerAuthAbility) GetServices(
	ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAuthContext(ctx, nil, authcommon.Read, "GetServices")

	if _, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	// 注入查询条件拦截器

	resp := svr.nextSvr.GetServices(ctx, query)
	if len(resp.Services) != 0 {
		for index := range resp.Services {
			svc := resp.Services[index]
			editable := svr.policyMgr.GetAuthChecker().AllowResourceOperate(authCtx, &authcommon.ResourceOpInfo{
				ResourceType: apisecurity.ResourceType_Services,
				Namespace:    svc.GetNamespace().GetValue(),
				ResourceName: svc.GetName().GetValue(),
				ResourceID:   svc.GetId().GetValue(),
				Operation:    authCtx.GetOperation(),
			})
			svc.Editable = utils.NewBoolValue(editable)
		}
	}
	return resp
}

// GetServicesCount 批量获取服务数量
func (svr *ServerAuthAbility) GetServicesCount(ctx context.Context) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAuthContext(ctx, nil, authcommon.Read, "GetServicesCount")

	if _, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.GetServicesCount(ctx)
}

// GetServiceToken 获取服务的 token
func (svr *ServerAuthAbility) GetServiceToken(ctx context.Context, req *apiservice.Service) *apiservice.Response {
	authCtx := svr.collectServiceAuthContext(ctx, nil, authcommon.Read, "GetServiceToken")

	if _, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.GetServiceToken(ctx, req)
}

// GetServiceOwner 获取服务的 owner
func (svr *ServerAuthAbility) GetServiceOwner(
	ctx context.Context, req []*apiservice.Service) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAuthContext(ctx, nil, authcommon.Read, "GetServiceOwner")

	if _, err := svr.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.GetServiceOwner(ctx, req)
}
