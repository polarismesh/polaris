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

	"github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateServiceAlias creates a service alias
func (svr *Server) CreateServiceAlias(
	ctx context.Context, req *apiservice.ServiceAlias) *apiservice.Response {
	authCtx := svr.collectServiceAliasAuthContext(
		ctx, []*apiservice.ServiceAlias{req}, authcommon.Create, authcommon.CreateServiceAlias)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewServiceAliasResponse(authcommon.ConvertToErrCode(err), req)
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	// 填充 ownerId 信息数据
	ownerId := utils.ParseOwnerID(ctx)
	if len(ownerId) > 0 {
		req.Owners = utils.NewStringValue(ownerId)
	}

	return svr.nextSvr.CreateServiceAlias(ctx, req)
}

// DeleteServiceAliases deletes service aliases
func (svr *Server) DeleteServiceAliases(ctx context.Context,
	reqs []*apiservice.ServiceAlias) *apiservice.BatchWriteResponse {
	authCtx := svr.collectServiceAliasAuthContext(ctx, reqs, authcommon.Delete, authcommon.DeleteServiceAliases)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.DeleteServiceAliases(ctx, reqs)
}

// UpdateServiceAlias updates service alias
func (svr *Server) UpdateServiceAlias(
	ctx context.Context, req *apiservice.ServiceAlias) *apiservice.Response {
	authCtx := svr.collectServiceAliasAuthContext(
		ctx, []*apiservice.ServiceAlias{req}, authcommon.Modify, authcommon.UpdateServiceAlias)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewServiceAliasResponse(authcommon.ConvertToErrCode(err), req)
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.UpdateServiceAlias(ctx, req)
}

// GetServiceAliases gets service aliases
func (svr *Server) GetServiceAliases(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAliasAuthContext(ctx, nil, authcommon.Read, authcommon.DescribeServiceAliases)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	cachetypes.AppendServicePredicate(ctx, func(ctx context.Context, cbr *model.Service) bool {
		return svr.policySvr.GetAuthChecker().ResourcePredicate(authCtx, &authcommon.ResourceEntry{
			Type:     security.ResourceType_Services,
			ID:       cbr.ID,
			Metadata: cbr.Meta,
		})
	})

	return svr.nextSvr.GetServiceAliases(ctx, query)
}
