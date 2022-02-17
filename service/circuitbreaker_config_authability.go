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

func (svr *serverAuthAbility) CreateCircuitBreakers(ctx context.Context,
	reqs []*api.CircuitBreaker) *api.BatchWriteResponse {
	authCtx := svr.collectCircuitBreakerAuthContext(ctx, reqs, model.Create)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	return svr.targetServer.CreateCircuitBreakers(ctx, reqs)
}

func (svr *serverAuthAbility) CreateCircuitBreakerVersions(ctx context.Context,
	reqs []*api.CircuitBreaker) *api.BatchWriteResponse {
	authCtx := svr.collectCircuitBreakerAuthContext(ctx, reqs, model.Create)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	return svr.targetServer.CreateCircuitBreakerVersions(ctx, reqs)
}

func (svr *serverAuthAbility) DeleteCircuitBreakers(ctx context.Context,
	reqs []*api.CircuitBreaker) *api.BatchWriteResponse {
	authCtx := svr.collectCircuitBreakerAuthContext(ctx, reqs, model.Delete)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	return svr.targetServer.DeleteCircuitBreakers(ctx, reqs)
}

func (svr *serverAuthAbility) UpdateCircuitBreakers(ctx context.Context,
	reqs []*api.CircuitBreaker) *api.BatchWriteResponse {
	authCtx := svr.collectCircuitBreakerAuthContext(ctx, reqs, model.Modify)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	return svr.targetServer.UpdateCircuitBreakers(ctx, reqs)
}

func (svr *serverAuthAbility) ReleaseCircuitBreakers(ctx context.Context,
	reqs []*api.ConfigRelease) *api.BatchWriteResponse {
	authCtx := svr.collectCircuitBreakerReleaseAuthContext(ctx, reqs, model.Create)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	return svr.targetServer.ReleaseCircuitBreakers(ctx, reqs)
}

func (svr *serverAuthAbility) UnBindCircuitBreakers(ctx context.Context,
	reqs []*api.ConfigRelease) *api.BatchWriteResponse {
	authCtx := svr.collectCircuitBreakerReleaseAuthContext(ctx, reqs, model.Modify)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	return svr.targetServer.UnBindCircuitBreakers(ctx, reqs)
}

func (svr *serverAuthAbility) GetCircuitBreaker(ctx context.Context,
	query map[string]string) *api.BatchQueryResponse {

	return svr.targetServer.GetCircuitBreaker(ctx, query)
}

func (svr *serverAuthAbility) GetCircuitBreakerVersions(ctx context.Context,
	query map[string]string) *api.BatchQueryResponse {

	return svr.targetServer.GetCircuitBreakerVersions(ctx, query)
}

func (svr *serverAuthAbility) GetMasterCircuitBreakers(ctx context.Context,
	query map[string]string) *api.BatchQueryResponse {

	return svr.targetServer.GetMasterCircuitBreakers(ctx, query)
}

func (svr *serverAuthAbility) GetReleaseCircuitBreakers(ctx context.Context,
	query map[string]string) *api.BatchQueryResponse {

	return svr.targetServer.GetReleaseCircuitBreakers(ctx, query)
}

func (svr *serverAuthAbility) GetCircuitBreakerByService(ctx context.Context,
	query map[string]string) *api.BatchQueryResponse {

	return svr.targetServer.GetCircuitBreakerByService(ctx, query)
}

func (svr *serverAuthAbility) GetCircuitBreakerToken(ctx context.Context, req *api.CircuitBreaker) *api.Response {

	return svr.targetServer.GetCircuitBreakerToken(ctx, req)
}
