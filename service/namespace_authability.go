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

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
)

// CreateNamespaces 创建命名空间，只需要要后置鉴权，将数据添加到资源策略中
func (svr *serverAuthAbility) CreateNamespaces(ctx context.Context, reqs []*api.Namespace) *api.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, reqs, model.Create)

	// 验证 token 信息
	if err := svr.authMgn.VerifyToken(authCtx); err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
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

// DeleteNamespaces 删除命名空间，需要先走权限检查
func (svr *serverAuthAbility) DeleteNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, req, model.Delete)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.DeleteNamespaces(ctx, req)
}

// UpdateNamespaces 更新命名空间，需要先走权限检查
func (svr *serverAuthAbility) UpdateNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, req, model.Modify)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.DeleteNamespaces(ctx, req)
}

// UpdateNamespaceToken 更新命名空间的token信息，需要先走权限检查
func (svr *serverAuthAbility) UpdateNamespaceToken(ctx context.Context, req *api.Namespace) *api.Response {
	authCtx := svr.collectNamespaceAuthContext(ctx, []*api.Namespace{req}, model.Modify)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.UpdateNamespaceToken(ctx, req)
}

// GetNamespaces 获取命名空间列表信息，暂时不走权限检查
func (svr *serverAuthAbility) GetNamespaces(ctx context.Context, query map[string][]string) *api.BatchQueryResponse {

	authCtx := svr.collectNamespaceAuthContext(ctx, nil, model.Modify)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchQueryResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.GetNamespaces(ctx, query)
}

// GetNamespaceToken 获取命名空间的token信息，暂时不走权限检查
func (svr *serverAuthAbility) GetNamespaceToken(ctx context.Context, req *api.Namespace) *api.Response {

	authCtx := svr.collectNamespaceAuthContext(ctx, nil, model.Modify)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.GetNamespaceToken(ctx, req)
}
