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

// CreateServices 批量创建服务
func (svr *serverAuthAbility) CreateServices(ctx context.Context, reqs []*api.Service) *api.BatchWriteResponse {
	authCtx := svr.collectServiceAuthContext(ctx, reqs, model.Create)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	// 填充 ownerId 信息数据
	ownerId := utils.ParseOwnerID(ctx)
	for index := range reqs {
		req := reqs[index]
		req.Owners = utils.NewStringValue(ownerId)
	}

	resp := svr.targetServer.CreateServices(ctx, reqs)

	return resp
}

// DeleteServices 批量删除服务
func (svr *serverAuthAbility) DeleteServices(ctx context.Context, reqs []*api.Service) *api.BatchWriteResponse {
	authCtx := svr.collectServiceAuthContext(ctx, reqs, model.Delete)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	resp := svr.targetServer.CreateServices(ctx, reqs)
	return resp
}

func (svr *serverAuthAbility) UpdateServices(ctx context.Context, reqs []*api.Service) *api.BatchWriteResponse {
	authCtx := svr.collectServiceAuthContext(ctx, reqs, model.Modify)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	return svr.targetServer.UpdateServices(ctx, reqs)
}

func (svr *serverAuthAbility) UpdateServiceToken(ctx context.Context, req *api.Service) *api.Response {
	authCtx := svr.collectServiceAuthContext(ctx, []*api.Service{req}, model.Create)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	return svr.targetServer.UpdateServiceToken(ctx, req)
}

func (svr *serverAuthAbility) GetServices(ctx context.Context, query map[string]string) *api.BatchQueryResponse {

	return svr.targetServer.GetServices(ctx, query)
}

func (svr *serverAuthAbility) GetServicesCount() *api.BatchQueryResponse {

	return svr.targetServer.GetServicesCount()
}

func (svr *serverAuthAbility) GetServiceToken(ctx context.Context, req *api.Service) *api.Response {

	return svr.targetServer.GetServiceToken(ctx, req)
}

func (svr *serverAuthAbility) GetServiceOwner(ctx context.Context, req []*api.Service) *api.BatchQueryResponse {

	return svr.targetServer.GetServiceOwner(ctx, req)
}
