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

package paramcheck

import (
	"context"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateServiceAlias implements service.DiscoverServer.
func (svr *Server) CreateServiceAlias(ctx context.Context,
	req *service_manage.ServiceAlias) *service_manage.Response {
	if resp := checkCreateServiceAliasReq(ctx, req); resp != nil {
		return resp
	}
	return svr.nextSvr.CreateServiceAlias(ctx, req)
}

// DeleteServiceAliases implements service.DiscoverServer.
func (svr *Server) DeleteServiceAliases(ctx context.Context,
	req []*service_manage.ServiceAlias) *service_manage.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > utils.MaxBatchSize {
		return api.NewBatchWriteResponse(apimodel.Code_BatchSizeOverLimit)
	}

	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range req {
		rsp := checkDeleteServiceAliasReq(ctx, req[i])
		api.Collect(batchRsp, rsp)
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.DeleteServiceAliases(ctx, req)
}

// UpdateServiceAlias implements service.DiscoverServer.
func (svr *Server) UpdateServiceAlias(ctx context.Context, req *service_manage.ServiceAlias) *service_manage.Response {
	// 检查请求参数
	if resp := checkReviseServiceAliasReq(ctx, req); resp != nil {
		return resp
	}
	return svr.nextSvr.UpdateServiceAlias(ctx, req)
}

// GetServiceAliases implements service.DiscoverServer.
func (svr *Server) GetServiceAliases(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetServiceAliases(ctx, query)
}

// checkCreateServiceAliasReq 检查别名请求
func checkCreateServiceAliasReq(ctx context.Context, req *apiservice.ServiceAlias) *apiservice.Response {
	response, done := preCheckAlias(req)
	if done {
		return response
	}
	// 检查字段长度是否大于DB中对应字段长
	err, notOk := CheckDbServiceAliasFieldLen(req)
	if notOk {
		return err
	}
	return nil
}

// checkReviseServiceAliasReq 检查删除、修改别名请求
func checkReviseServiceAliasReq(ctx context.Context, req *apiservice.ServiceAlias) *apiservice.Response {
	resp := checkDeleteServiceAliasReq(ctx, req)
	if resp != nil {
		return resp
	}
	// 检查服务名
	if err := utils.CheckResourceName(req.GetService()); err != nil {
		return api.NewServiceAliasResponse(apimodel.Code_InvalidServiceName, req)
	}

	// 检查命名空间
	if err := utils.CheckResourceName(req.GetNamespace()); err != nil {
		return api.NewServiceAliasResponse(apimodel.Code_InvalidNamespaceName, req)
	}
	return nil
}

// checkDeleteServiceAliasReq 检查删除、修改别名请求
func checkDeleteServiceAliasReq(ctx context.Context, req *apiservice.ServiceAlias) *apiservice.Response {
	if req == nil {
		return api.NewServiceAliasResponse(apimodel.Code_EmptyRequest, req)
	}

	// 检查服务别名
	if err := utils.CheckResourceName(req.GetAlias()); err != nil {
		return api.NewServiceAliasResponse(apimodel.Code_InvalidServiceAlias, req)
	}

	// 检查服务别名命名空间
	if err := utils.CheckResourceName(req.GetAliasNamespace()); err != nil {
		return api.NewServiceAliasResponse(apimodel.Code_InvalidNamespaceWithAlias, req)
	}

	// 检查字段长度是否大于DB中对应字段长
	err, notOk := CheckDbServiceAliasFieldLen(req)
	if notOk {
		return err
	}

	return nil
}

func preCheckAlias(req *apiservice.ServiceAlias) (*apiservice.Response, bool) {
	if req == nil {
		return api.NewServiceAliasResponse(apimodel.Code_EmptyRequest, req), true
	}

	if err := utils.CheckResourceName(req.GetService()); err != nil {
		return api.NewServiceAliasResponse(apimodel.Code_InvalidServiceName, req), true
	}

	if err := utils.CheckResourceName(req.GetNamespace()); err != nil {
		return api.NewServiceAliasResponse(apimodel.Code_InvalidNamespaceName, req), true
	}

	if err := utils.CheckResourceName(req.GetAliasNamespace()); err != nil {
		return api.NewServiceAliasResponse(apimodel.Code_InvalidNamespaceName, req), true
	}

	// 默认类型，需要检查alias是否为空
	if req.GetType() == apiservice.AliasType_DEFAULT {
		if err := utils.CheckResourceName(req.GetAlias()); err != nil {
			return api.NewServiceAliasResponse(apimodel.Code_InvalidServiceAlias, req), true
		}
	}
	return nil, false
}

// CheckDbServiceAliasFieldLen 检查DB中service表对应的入参字段合法性
func CheckDbServiceAliasFieldLen(req *apiservice.ServiceAlias) (*apiservice.Response, bool) {
	if err := utils.CheckDbStrFieldLen(req.GetService(), utils.MaxDbServiceNameLength); err != nil {
		return api.NewServiceAliasResponse(apimodel.Code_InvalidServiceName, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetNamespace(), utils.MaxDbServiceNamespaceLength); err != nil {
		return api.NewServiceAliasResponse(apimodel.Code_InvalidNamespaceName, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetAlias(), utils.MaxDbServiceNameLength); err != nil {
		return api.NewServiceAliasResponse(apimodel.Code_InvalidServiceAlias, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetAliasNamespace(), utils.MaxDbServiceNamespaceLength); err != nil {
		return api.NewServiceAliasResponse(apimodel.Code_InvalidNamespaceWithAlias, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetComment(), utils.MaxDbServiceCommentLength); err != nil {
		return api.NewServiceAliasResponse(apimodel.Code_InvalidServiceAliasComment, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetOwners(), utils.MaxDbServiceOwnerLength); err != nil {
		return api.NewServiceAliasResponse(apimodel.Code_InvalidServiceAliasOwners, req), true
	}
	return nil, false
}
