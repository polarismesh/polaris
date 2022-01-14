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
//  @receiver svr
//  @param ctx
//  @param req
//  @return *api.BatchWriteResponse
func (svr *serverAuthAbility) CreateNamespaces(ctx context.Context, reqs []*api.Namespace) *api.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, reqs, model.Create)

	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	// 填充 ownerId 信息数据
	ownerId := utils.ParseOwnerID(ctx)
	for index := range reqs {
		req := reqs[index]
		req.Owners = utils.NewStringValue(ownerId)
	}

	resp := svr.targetServer.CreateNamespaces(ctx, reqs)
	return resp
}

// DeleteNamespaces 删除命名空间，需要先走权限检查
//  @receiver svr
//  @param ctx
//  @param req
//  @return *api.BatchWriteResponse
func (svr *serverAuthAbility) DeleteNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, req, model.Delete)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	resp := svr.targetServer.DeleteNamespaces(ctx, req)

	return resp
}

// UpdateNamespaces 更新命名空间，需要先走权限检查
//  @receiver svr
//  @param ctx
//  @param req
//  @return *api.BatchWriteResponse
func (svr *serverAuthAbility) UpdateNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse {
	authCtx := svr.collectNamespaceAuthContext(ctx, req, model.Modify)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	resp := svr.targetServer.DeleteNamespaces(ctx, req)
	return resp
}

// UpdateNamespaceToken 更新命名空间的token信息，需要先走权限检查
//  @receiver svr
//  @param ctx
//  @param req
//  @return *api.Response
func (svr *serverAuthAbility) UpdateNamespaceToken(ctx context.Context, req *api.Namespace) *api.Response {
	authCtx := svr.collectNamespaceAuthContext(ctx, []*api.Namespace{req}, model.Modify)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewResponseWithMsg(api.NotAllowedAccess, err.Error())
	}
	resp := svr.targetServer.UpdateNamespaceToken(ctx, req)

	return resp
}

// GetNamespaces 获取命名空间列表信息，暂时不走权限检查
//  @receiver svr
//  @param ctx
//  @param query
//  @return *api.BatchQueryResponse
func (svr *serverAuthAbility) GetNamespaces(ctx context.Context, query map[string][]string) *api.BatchQueryResponse {
	return svr.targetServer.GetNamespaces(ctx, query)
}

// GetNamespaceToken 获取命名空间的token信息，暂时不走权限检查
//  @receiver svr
//  @param ctx
//  @param req
//  @return *api.Response
func (svr *serverAuthAbility) GetNamespaceToken(ctx context.Context, req *api.Namespace) *api.Response {
	return svr.targetServer.GetNamespaceToken(ctx, req)
}
