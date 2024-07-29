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

package namespace

import (
	"context"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

var _ NamespaceOperateServer = (*serverAuthAbility)(nil)

// CreateNamespaceIfAbsent Create a single name space
func (svr *serverAuthAbility) CreateNamespaceIfAbsent(ctx context.Context,
	req *apimodel.Namespace) (string, *apiservice.Response) {
	return svr.targetServer.CreateNamespaceIfAbsent(ctx, req)
}

// CreateNamespace 创建命名空间，只需要要后置鉴权，将数据添加到资源策略中
func (svr *serverAuthAbility) CreateNamespace(ctx context.Context, req *apimodel.Namespace) *apiservice.Response {
	authCtx := svr.collectNamespaceAuthContext(
		ctx, []*apimodel.Namespace{req}, authcommon.Create, authcommon.CreateNamespace)
	// 验证 token 信息
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	// 填充 ownerId 信息数据
	if ownerId := utils.ParseOwnerID(ctx); len(ownerId) > 0 {
		req.Owners = utils.NewStringValue(ownerId)
	}

	return svr.targetServer.CreateNamespace(ctx, req)
}

// CreateNamespaces 创建命名空间，只需要要后置鉴权，将数据添加到资源策略中
func (svr *serverAuthAbility) CreateNamespaces(
	ctx context.Context, reqs []*apimodel.Namespace) *apiservice.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, reqs, authcommon.Create, authcommon.CreateNamespaces)

	// 验证 token 信息
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	// 填充 ownerId 信息数据
	ownerId := utils.ParseOwnerID(ctx)
	if len(ownerId) > 0 {
		for index := range reqs {
			req := reqs[index]
			req.Owners = utils.NewStringValue(ownerId)
		}
	}

	return svr.targetServer.CreateNamespaces(ctx, reqs)
}

// DeleteNamespace 删除命名空间，需要先走权限检查
func (svr *serverAuthAbility) DeleteNamespace(ctx context.Context, req *apimodel.Namespace) *apiservice.Response {
	authCtx := svr.collectNamespaceAuthContext(
		ctx, []*apimodel.Namespace{req}, authcommon.Delete, authcommon.DeleteNamespace)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.DeleteNamespace(ctx, req)
}

// DeleteNamespaces 删除命名空间，需要先走权限检查
func (svr *serverAuthAbility) DeleteNamespaces(
	ctx context.Context, reqs []*apimodel.Namespace) *apiservice.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, reqs, authcommon.Delete, authcommon.DeleteNamespaces)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.DeleteNamespaces(ctx, reqs)
}

// UpdateNamespaces 更新命名空间，需要先走权限检查
func (svr *serverAuthAbility) UpdateNamespaces(
	ctx context.Context, req []*apimodel.Namespace) *apiservice.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, req, authcommon.Modify, authcommon.UpdateNamespaces)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.UpdateNamespaces(ctx, req)
}

// UpdateNamespaceToken 更新命名空间的token信息，需要先走权限检查
func (svr *serverAuthAbility) UpdateNamespaceToken(ctx context.Context, req *apimodel.Namespace) *apiservice.Response {
	authCtx := svr.collectNamespaceAuthContext(
		ctx, []*apimodel.Namespace{req}, authcommon.Modify, authcommon.UpdateNamespaceToken)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.UpdateNamespaceToken(ctx, req)
}

// GetNamespaces 获取命名空间列表信息，暂时不走权限检查
func (svr *serverAuthAbility) GetNamespaces(
	ctx context.Context, query map[string][]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, nil, authcommon.Read, authcommon.DescribeNamespaces)
	_, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchQueryResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	cachetypes.AppendNamespacePredicate(ctx, func(ctx context.Context, n *model.Namespace) bool {
		return svr.policySvr.GetAuthChecker().ResourcePredicate(authCtx, &authcommon.ResourceEntry{
			Type: apisecurity.ResourceType_Users,
			ID:   n.Name,
		})
	})

	return svr.targetServer.GetNamespaces(ctx, query)
}

// GetNamespaceToken 获取命名空间的token信息，暂时不走权限检查
func (svr *serverAuthAbility) GetNamespaceToken(ctx context.Context, req *apimodel.Namespace) *apiservice.Response {
	authCtx := svr.collectNamespaceAuthContext(
		ctx, []*apimodel.Namespace{req}, authcommon.Read, authcommon.DescribeNamespaceToken)
	_, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.targetServer.GetNamespaceToken(ctx, req)
}
