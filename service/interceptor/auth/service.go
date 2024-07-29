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
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateServices 批量创建服务
func (svr *Server) CreateServices(
	ctx context.Context, reqs []*apiservice.Service) *apiservice.BatchWriteResponse {
	authCtx := svr.collectServiceAuthContext(ctx, reqs, authcommon.Create, authcommon.CreateServices)

	_, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
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
func (svr *Server) DeleteServices(
	ctx context.Context, reqs []*apiservice.Service) *apiservice.BatchWriteResponse {
	authCtx := svr.collectServiceAuthContext(ctx, reqs, authcommon.Delete, authcommon.DeleteServices)

	accessRes := authCtx.GetAccessResources()
	delete(accessRes, apisecurity.ResourceType_Namespaces)
	authCtx.SetAccessResources(accessRes)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	resp := svr.nextSvr.DeleteServices(ctx, reqs)
	return resp
}

// UpdateServices 对于服务修改来说，只针对服务本身，而不需要检查命名空间
func (svr *Server) UpdateServices(
	ctx context.Context, reqs []*apiservice.Service) *apiservice.BatchWriteResponse {
	authCtx := svr.collectServiceAuthContext(ctx, reqs, authcommon.Modify, authcommon.UpdateServices)

	accessRes := authCtx.GetAccessResources()
	delete(accessRes, apisecurity.ResourceType_Namespaces)
	authCtx.SetAccessResources(accessRes)

	_, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.UpdateServices(ctx, reqs)
}

// UpdateServiceToken 更新服务的 token
func (svr *Server) UpdateServiceToken(
	ctx context.Context, req *apiservice.Service) *apiservice.Response {
	authCtx := svr.collectServiceAuthContext(
		ctx, []*apiservice.Service{req}, authcommon.Modify, authcommon.UpdateServiceToken)

	accessRes := authCtx.GetAccessResources()
	delete(accessRes, apisecurity.ResourceType_Namespaces)
	authCtx.SetAccessResources(accessRes)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.UpdateServiceToken(ctx, req)
}

func (svr *Server) GetAllServices(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAuthContext(ctx, nil, authcommon.Read, authcommon.DescribeAllServices)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.GetAllServices(ctx, query)
}

// GetServices 批量获取服务
func (svr *Server) GetServices(
	ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAuthContext(ctx, nil, authcommon.Read, authcommon.DescribeServices)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	// 注入查询条件拦截器

	resp := svr.nextSvr.GetServices(ctx, query)
	if len(resp.Services) != 0 {
		for index := range resp.Services {
			svc := resp.Services[index]
			// TODO 需要配合 metadata 做调整
			svc.Editable = utils.NewBoolValue(true)
		}
	}
	return resp
}

// GetServicesCount 批量获取服务数量
func (svr *Server) GetServicesCount(ctx context.Context) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAuthContext(ctx, nil, authcommon.Read, authcommon.DescribeServicesCount)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.GetServicesCount(ctx)
}

// GetServiceToken 获取服务的 token
func (svr *Server) GetServiceToken(ctx context.Context, req *apiservice.Service) *apiservice.Response {
	authCtx := svr.collectServiceAuthContext(ctx, nil, authcommon.Read, authcommon.DescribeServiceToken)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.GetServiceToken(ctx, req)
}

// GetServiceOwner 获取服务的 owner
func (svr *Server) GetServiceOwner(
	ctx context.Context, req []*apiservice.Service) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAuthContext(ctx, nil, authcommon.Read, authcommon.DescribeServiceOwner)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	cachetypes.AppendServicePredicate(ctx, func(ctx context.Context, cbr *model.Service) bool {
		return svr.policySvr.GetAuthChecker().ResourcePredicate(authCtx, &authcommon.ResourceEntry{
			Type:     security.ResourceType_Services,
			ID:       cbr.ID,
			Metadata: cbr.Meta,
		})
	})

	return svr.nextSvr.GetServiceOwner(ctx, req)
}
