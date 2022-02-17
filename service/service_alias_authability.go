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

// CreateServiceAlias
func (svr *serverAuthAbility) CreateServiceAlias(ctx context.Context,
	req *api.ServiceAlias) *api.Response {
	authCtx := svr.collectServiceAliasAuthContext(ctx, []*api.ServiceAlias{req}, model.Create)

	if _, err := svr.authMgn.CheckPermission(authCtx); err != nil {
		return api.NewServiceAliasResponse(convertToErrCode(err), req)
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	// 填充 ownerId 信息数据
	ownerId := utils.ParseOwnerID(ctx)
	if len(ownerId) > 0 {
		req.Owners = utils.NewStringValue(ownerId)
	}

	return svr.targetServer.CreateServiceAlias(ctx, req)
}

// DeleteServiceAliases
func (svr *serverAuthAbility) DeleteServiceAliases(ctx context.Context,
	reqs []*api.ServiceAlias) *api.BatchWriteResponse {
	authCtx := svr.collectServiceAliasAuthContext(ctx, reqs, model.Delete)

	if _, err := svr.authMgn.CheckPermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(convertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.DeleteServiceAliases(ctx, reqs)
}

// UpdateServiceAlias
func (svr *serverAuthAbility) UpdateServiceAlias(ctx context.Context, req *api.ServiceAlias) *api.Response {
	authCtx := svr.collectServiceAliasAuthContext(ctx, []*api.ServiceAlias{req}, model.Modify)

	if _, err := svr.authMgn.CheckPermission(authCtx); err != nil {
		return api.NewServiceAliasResponse(convertToErrCode(err), req)
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.UpdateServiceAlias(ctx, req)
}

// GetServiceAliases
func (svr *serverAuthAbility) GetServiceAliases(ctx context.Context,
	query map[string]string) *api.BatchQueryResponse {
	authCtx := svr.collectServiceAliasAuthContext(ctx, nil, model.Read)

	if _, err := svr.authMgn.CheckPermission(authCtx); err != nil {
		return api.NewBatchQueryResponse(convertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	resp := svr.targetServer.GetServiceAliases(ctx, query)
	if len(resp.Aliases) != 0 {

		// 对于服务别名，则是参考源服务是否有编辑权限
		principal := model.Principal{
			PrincipalID:   utils.ParseUserID(ctx),
			PrincipalRole: model.PrincipalUser,
		}
		for index := range resp.Aliases {
			alias := resp.Aliases[index]
			svc := svr.Cache().Service().GetServiceByName(alias.Service.Value, alias.Namespace.Value)
			editable := true
			// 如果鉴权能力没有开启，那就默认都可以进行编辑
			if svr.authMgn.IsOpenAuth() {
				editable = svr.Cache().AuthStrategy().IsResourceEditable(principal,
					api.ResourceType_Services, svc.ID)
			}
			alias.Editable = utils.NewBoolValue(editable)
		}
	}
	return resp
}
