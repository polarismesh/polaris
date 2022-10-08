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

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateInstances create one instance
func (svr *serverAuthAbility) RegisterInstance(ctx context.Context, req *api.Instance) *api.Response {
	authCtx := svr.collectClientInstanceAuthContext(ctx, []*api.Instance{req}, model.Create, "RegisterInstance")

	_, err := svr.authMgn.CheckClientPermission(authCtx)
	if err != nil {
		resp := api.NewResponseWithMsg(convertToErrCode(err), err.Error())
		return resp
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.RegisterInstance(ctx, req)
}

// DeleteInstance delete onr instance
func (svr *serverAuthAbility) DeregisterInstance(ctx context.Context, req *api.Instance) *api.Response {
	authCtx := svr.collectClientInstanceAuthContext(ctx, []*api.Instance{req}, model.Create, "DeregisterInstance")

	_, err := svr.authMgn.CheckClientPermission(authCtx)
	if err != nil {
		resp := api.NewResponseWithMsg(convertToErrCode(err), err.Error())
		return resp
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.DeregisterInstance(ctx, req)
}

// ReportClient is the interface for reporting client authability
func (svr *serverAuthAbility) ReportClient(ctx context.Context, req *api.Client) *api.Response {
	return svr.targetServer.ReportClient(ctx, req)
}

// GetReportClientWithCache Used for client acquisition service information
func (svr *serverAuthAbility) GetReportClientWithCache(ctx context.Context, query map[string]string) *model.PrometheusDiscoveryResponse {
	return svr.targetServer.GetReportClientWithCache(ctx, query)
}

// GetServiceWithCache is the interface for getting service with cache
func (svr *serverAuthAbility) GetServiceWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	return svr.targetServer.GetServiceWithCache(ctx, req)
}

// ServiceInstancesCache is the interface for getting service instances cache
func (svr *serverAuthAbility) ServiceInstancesCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	return svr.targetServer.ServiceInstancesCache(ctx, req)
}

// GetRoutingConfigWithCache is the interface for getting routing config with cache
func (svr *serverAuthAbility) GetRoutingConfigWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	return svr.targetServer.GetRoutingConfigWithCache(ctx, req)
}

// GetRateLimitWithCache is the interface for getting rate limit with cache
func (svr *serverAuthAbility) GetRateLimitWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	return svr.targetServer.GetRateLimitWithCache(ctx, req)
}

// GetCircuitBreakerWithCache is the interface for getting a circuit breaker with cache
func (svr *serverAuthAbility) GetCircuitBreakerWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	return svr.targetServer.GetCircuitBreakerWithCache(ctx, req)
}
