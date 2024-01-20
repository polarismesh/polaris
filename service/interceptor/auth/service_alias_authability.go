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

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateServiceAlias creates a service alias
func (svr *ServerAuthAbility) CreateServiceAlias(
	ctx context.Context, req *apiservice.ServiceAlias) *apiservice.Response {
	authCtx := svr.collectServiceAliasAuthContext(
		ctx, []*apiservice.ServiceAlias{req}, model.Create, "CreateServiceAlias")

	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
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

// DeleteServiceAliases deletes service aliases
func (svr *ServerAuthAbility) DeleteServiceAliases(ctx context.Context,
	reqs []*apiservice.ServiceAlias) *apiservice.BatchWriteResponse {
	authCtx := svr.collectServiceAliasAuthContext(ctx, reqs, model.Delete, "DeleteServiceAliases")

	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(convertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.DeleteServiceAliases(ctx, reqs)
}

// UpdateServiceAlias updates service alias
func (svr *ServerAuthAbility) UpdateServiceAlias(
	ctx context.Context, req *apiservice.ServiceAlias) *apiservice.Response {
	authCtx := svr.collectServiceAliasAuthContext(
		ctx, []*apiservice.ServiceAlias{req}, model.Modify, "UpdateServiceAlias")

	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewServiceAliasResponse(convertToErrCode(err), req)
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.UpdateServiceAlias(ctx, req)
}

// GetServiceAliases gets service aliases
func (svr *ServerAuthAbility) GetServiceAliases(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAliasAuthContext(ctx, nil, model.Read, "GetServiceAliases")

	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
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
			if svc == nil {
				continue
			}
			editable := true
			// 如果鉴权能力没有开启，那就默认都可以进行编辑
			if svr.strategyMgn.GetAuthChecker().IsOpenConsoleAuth() {
				editable = svr.Cache().AuthStrategy().IsResourceEditable(principal,
					apisecurity.ResourceType_Services, svc.ID)
			}
			alias.Editable = utils.NewBoolValue(editable)
		}
	}
	return resp
}
