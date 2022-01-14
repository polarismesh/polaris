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
)

// CreateServiceAlias
//  @receiver svr
//  @param ctx
//  @param req
//  @return *api.Response
func (svr *serverAuthAbility) CreateServiceAlias(ctx context.Context, req *api.ServiceAlias) *api.Response {
	authCtx := svr.collectServiceAliasAuthContext(ctx, []*api.ServiceAlias{req}, model.Create)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	return svr.targetServer.CreateServiceAlias(ctx, req)
}

// DeleteServiceAliases
//  @receiver svr
//  @param ctx
//  @param req
//  @return *api.BatchWriteResponse
func (svr *serverAuthAbility) DeleteServiceAliases(ctx context.Context, reqs []*api.ServiceAlias) *api.BatchWriteResponse {
	authCtx := svr.collectServiceAliasAuthContext(ctx, reqs, model.Modify)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	return svr.targetServer.DeleteServiceAliases(ctx, reqs)
}

// UpdateServiceAlias
//  @receiver svr
//  @param ctx
//  @param req
//  @return *api.Response
func (svr *serverAuthAbility) UpdateServiceAlias(ctx context.Context, req *api.ServiceAlias) *api.Response {
	authCtx := svr.collectServiceAliasAuthContext(ctx, []*api.ServiceAlias{req}, model.Create)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	return svr.targetServer.UpdateServiceAlias(ctx, req)
}

// GetServiceAliases
//  @receiver svr
//  @param ctx
//  @param query
//  @return *api.BatchQueryResponse
func (svr *serverAuthAbility) GetServiceAliases(ctx context.Context, query map[string]string) *api.BatchQueryResponse {

	return svr.targetServer.GetServiceAliases(ctx, query)
}
