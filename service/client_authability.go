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
)

// ReportClient
//  @receiver svr
//  @param ctx
//  @param req
//  @return *api.Response
func (svr *serverAuthAbility) ReportClient(ctx context.Context, req *api.Client) *api.Response {
	return svr.targetServer.ReportClient(ctx, req)
}

// GetServiceWithCache
//  @receiver svr
//  @param ctx
//  @param req
//  @return *api.DiscoverResponse
func (svr *serverAuthAbility) GetServiceWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	return svr.targetServer.GetServiceWithCache(ctx, req)
}

// ServiceInstancesCache
//  @receiver svr
//  @param ctx
//  @param req
//  @return *api.DiscoverResponse
func (svr *serverAuthAbility) ServiceInstancesCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	return svr.targetServer.ServiceInstancesCache(ctx, req)
}

// GetRoutingConfigWithCache
//  @receiver svr
//  @param ctx
//  @param req
//  @return *api.DiscoverResponse
func (svr *serverAuthAbility) GetRoutingConfigWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	return svr.targetServer.GetRoutingConfigWithCache(ctx, req)
}

// GetRateLimitWithCache
//  @receiver svr
//  @param ctx
//  @param req
//  @return *api.DiscoverResponse
func (svr *serverAuthAbility) GetRateLimitWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	return svr.targetServer.GetRateLimitWithCache(ctx, req)
}

// GetCircuitBreakerWithCache
//  @receiver svr
//  @param ctx
//  @param req
//  @return *api.DiscoverResponse
func (svr *serverAuthAbility) GetCircuitBreakerWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse {
	return svr.targetServer.GetCircuitBreakerWithCache(ctx, req)
}
