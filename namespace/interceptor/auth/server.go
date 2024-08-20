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

package auth

import (
	"context"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/auth"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/namespace"
)

var _ namespace.NamespaceOperateServer = (*Server)(nil)

// Server 带有鉴权能力的 NamespaceOperateServer
// 该层会对请求参数做一些调整，根据具体的请求发起人，设置为数据对应的 owner，不可为为别人进行创建资源
type Server struct {
	nextSvr   namespace.NamespaceOperateServer
	userSvr   auth.UserServer
	policySvr auth.StrategyServer
	cacheSvr  cachetypes.CacheManager
}

func NewServer(nextSvr namespace.NamespaceOperateServer, userSvr auth.UserServer,
	policySvr auth.StrategyServer, cacheSvr cachetypes.CacheManager) namespace.NamespaceOperateServer {
	proxy := &Server{
		nextSvr:   nextSvr,
		userSvr:   userSvr,
		policySvr: policySvr,
		cacheSvr:  cacheSvr,
	}

	if actualSvr, ok := nextSvr.(*namespace.Server); ok {
		actualSvr.SetResourceHooks(proxy)
	}
	return proxy
}

// CreateNamespaceIfAbsent Create a single name space
func (svr *Server) CreateNamespaceIfAbsent(ctx context.Context,
	req *apimodel.Namespace) (string, *apiservice.Response) {
	return svr.nextSvr.CreateNamespaceIfAbsent(ctx, req)
}

// CreateNamespace 创建命名空间，只需要要后置鉴权，将数据添加到资源策略中
func (svr *Server) CreateNamespace(ctx context.Context, req *apimodel.Namespace) *apiservice.Response {
	authCtx := svr.collectNamespaceAuthContext(
		ctx, []*apimodel.Namespace{req}, authcommon.Create, authcommon.CreateNamespace)
	// 验证 token 信息
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	// 填充 ownerId 信息数据
	if ownerId := utils.ParseOwnerID(ctx); len(ownerId) > 0 {
		req.Owners = utils.NewStringValue(ownerId)
	}

	return svr.nextSvr.CreateNamespace(ctx, req)
}

// CreateNamespaces 创建命名空间，只需要要后置鉴权，将数据添加到资源策略中
func (svr *Server) CreateNamespaces(
	ctx context.Context, reqs []*apimodel.Namespace) *apiservice.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, reqs, authcommon.Create, authcommon.CreateNamespaces)

	// 验证 token 信息
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
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
	return svr.nextSvr.CreateNamespaces(ctx, reqs)
}

// DeleteNamespace 删除命名空间，需要先走权限检查
func (svr *Server) DeleteNamespace(ctx context.Context, req *apimodel.Namespace) *apiservice.Response {
	authCtx := svr.collectNamespaceAuthContext(
		ctx, []*apimodel.Namespace{req}, authcommon.Delete, authcommon.DeleteNamespace)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.DeleteNamespace(ctx, req)
}

// DeleteNamespaces 删除命名空间，需要先走权限检查
func (svr *Server) DeleteNamespaces(
	ctx context.Context, reqs []*apimodel.Namespace) *apiservice.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, reqs, authcommon.Delete, authcommon.DeleteNamespaces)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.DeleteNamespaces(ctx, reqs)
}

// UpdateNamespaces 更新命名空间，需要先走权限检查
func (svr *Server) UpdateNamespaces(
	ctx context.Context, req []*apimodel.Namespace) *apiservice.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, req, authcommon.Modify, authcommon.UpdateNamespaces)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.UpdateNamespaces(ctx, req)
}

// UpdateNamespaceToken 更新命名空间的token信息，需要先走权限检查
func (svr *Server) UpdateNamespaceToken(ctx context.Context, req *apimodel.Namespace) *apiservice.Response {
	authCtx := svr.collectNamespaceAuthContext(
		ctx, []*apimodel.Namespace{req}, authcommon.Modify, authcommon.UpdateNamespaceToken)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.UpdateNamespaceToken(ctx, req)
}

// GetNamespaces 获取命名空间列表信息，暂时不走权限检查
func (svr *Server) GetNamespaces(
	ctx context.Context, query map[string][]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, nil, authcommon.Read, authcommon.DescribeNamespaces)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	ctx = cachetypes.AppendNamespacePredicate(ctx, func(ctx context.Context, n *model.Namespace) bool {
		return svr.policySvr.GetAuthChecker().ResourcePredicate(authCtx, &authcommon.ResourceEntry{
			Type:     apisecurity.ResourceType_Namespaces,
			ID:       n.Name,
			Metadata: n.Metadata,
		})
	})

	authCtx.SetRequestContext(ctx)
	resp := svr.nextSvr.GetNamespaces(ctx, query)
	for i := range resp.Namespaces {
		item := resp.Namespaces[i]
		authCtx.SetAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
			apisecurity.ResourceType_Namespaces: {
				{
					Type: apisecurity.ResourceType_Namespaces,
					ID:   item.GetId().GetValue(),
				},
			},
		})
		authCtx.SetMethod([]authcommon.ServerFunctionName{
			authcommon.UpdateNamespaces, authcommon.DeleteNamespaces, authcommon.DeleteNamespace,
		})
		// 如果检查不通过，设置 editable 为 false
		if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
			item.Editable = utils.NewBoolValue(false)
		}
	}
	return resp
}

// GetNamespaceToken 获取命名空间的token信息，暂时不走权限检查
func (svr *Server) GetNamespaceToken(ctx context.Context, req *apimodel.Namespace) *apiservice.Response {
	authCtx := svr.collectNamespaceAuthContext(
		ctx, []*apimodel.Namespace{req}, authcommon.Read, authcommon.DescribeNamespaceToken)
	_, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.GetNamespaceToken(ctx, req)
}

// collectNamespaceAuthContext 对于命名空间的处理，收集所有的与鉴权的相关信息
func (svr *Server) collectNamespaceAuthContext(ctx context.Context, req []*apimodel.Namespace,
	resourceOp authcommon.ResourceOperation, methodName authcommon.ServerFunctionName) *authcommon.AcquireContext {
	return authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(resourceOp),
		authcommon.WithModule(authcommon.CoreModule),
		authcommon.WithMethod(methodName),
		authcommon.WithAccessResources(svr.queryNamespaceResource(req)),
	)
}

// queryNamespaceResource 根据所给的 namespace 信息，收集对应的 ResourceEntry 列表
func (svr *Server) queryNamespaceResource(
	req []*apimodel.Namespace) map[apisecurity.ResourceType][]authcommon.ResourceEntry {
	if len(req) == 0 {
		return map[apisecurity.ResourceType][]authcommon.ResourceEntry{}
	}

	names := utils.NewSet[string]()
	for index := range req {
		names.Add(req[index].Name.GetValue())
	}
	param := names.ToSlice()
	nsArr := svr.cacheSvr.Namespace().GetNamespacesByName(param)

	temp := make([]authcommon.ResourceEntry, 0, len(nsArr))

	for index := range nsArr {
		ns := nsArr[index]
		temp = append(temp, authcommon.ResourceEntry{
			Type:  apisecurity.ResourceType_Namespaces,
			ID:    ns.Name,
			Owner: ns.Owner,
		})
	}

	ret := map[apisecurity.ResourceType][]authcommon.ResourceEntry{
		apisecurity.ResourceType_Namespaces: temp,
	}
	authLog.Debug("[Auth][Server] collect namespace access res", zap.Any("res", ret))
	return ret
}
