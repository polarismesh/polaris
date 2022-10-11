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

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

var _ NamespaceOperateServer = (*serverAuthAbility)(nil)

// CreateNamespaceIfAbsent Create a single name space
func (svr *serverAuthAbility) CreateNamespaceIfAbsent(ctx context.Context, req *api.Namespace) error {
	return svr.targetServer.CreateNamespaceIfAbsent(ctx, req)
}

// CreateNamespaces 创建命名空间，只需要要后置鉴权，将数据添加到资源策略中
func (svr *serverAuthAbility) CreateNamespace(ctx context.Context, req *api.Namespace) *api.Response {
	authCtx := svr.collectNamespaceAuthContext(ctx, []*api.Namespace{req}, model.Create, "CreateNamespace")

	// 验证 token 信息
	if _, err := svr.authMgn.CheckConsolePermission(authCtx); err != nil {
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
func (svr *serverAuthAbility) CreateNamespaces(ctx context.Context, reqs []*api.Namespace) *api.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, reqs, model.Create, "CreateNamespaces")

	// 验证 token 信息
	if _, err := svr.authMgn.CheckConsolePermission(authCtx); err != nil {
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
func (svr *serverAuthAbility) DeleteNamespace(ctx context.Context, req *api.Namespace) *api.Response {
	authCtx := svr.collectNamespaceAuthContext(ctx, []*api.Namespace{req}, model.Delete, "DeleteNamespace")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.DeleteNamespace(ctx, req)
}

// DeleteNamespaces 删除命名空间，需要先走权限检查
func (svr *serverAuthAbility) DeleteNamespaces(ctx context.Context, reqs []*api.Namespace) *api.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, reqs, model.Delete, "DeleteNamespaces")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.DeleteNamespaces(ctx, reqs)
}

// UpdateNamespaces 更新命名空间，需要先走权限检查
func (svr *serverAuthAbility) UpdateNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, req, model.Modify, "UpdateNamespaces")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.UpdateNamespaces(ctx, req)
}

// UpdateNamespaceToken 更新命名空间的token信息，需要先走权限检查
func (svr *serverAuthAbility) UpdateNamespaceToken(ctx context.Context, req *api.Namespace) *api.Response {
	authCtx := svr.collectNamespaceAuthContext(ctx, []*api.Namespace{req}, model.Modify, "UpdateNamespaceToken")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.UpdateNamespaceToken(ctx, req)
}

// GetNamespaces 获取命名空间列表信息，暂时不走权限检查
func (svr *serverAuthAbility) GetNamespaces(ctx context.Context, query map[string][]string) *api.BatchQueryResponse {

	authCtx := svr.collectNamespaceAuthContext(ctx, nil, model.Read, "GetNamespaces")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewBatchQueryResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	resp := svr.targetServer.GetNamespaces(ctx, query)
	if len(resp.Namespaces) != 0 {

		principal := model.Principal{
			PrincipalID:   utils.ParseUserID(ctx),
			PrincipalRole: model.PrincipalUser,
		}
		for index := range resp.Namespaces {
			ns := resp.Namespaces[index]
			editable := true
			// 如果鉴权能力没有开启，那就默认都可以进行编辑
			if svr.authMgn.IsOpenConsoleAuth() {
				editable = svr.targetServer.caches.AuthStrategy().IsResourceEditable(principal,
					api.ResourceType_Namespaces, ns.Id.GetValue())
			}
			ns.Editable = utils.NewBoolValue(editable)
		}
	}

	return resp
}

// GetNamespaceToken 获取命名空间的token信息，暂时不走权限检查
func (svr *serverAuthAbility) GetNamespaceToken(ctx context.Context, req *api.Namespace) *api.Response {

	authCtx := svr.collectNamespaceAuthContext(ctx, []*api.Namespace{req}, model.Read, "GetNamespaceToken")

	_, err := svr.authMgn.CheckConsolePermission(authCtx)
	if err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.GetNamespaceToken(ctx, req)
}
