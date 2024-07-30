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

	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
)

// CreateCircuitBreakers creates circuit breakers
func (svr *Server) CreateCircuitBreakers(ctx context.Context,
	reqs []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse {
	return svr.nextSvr.CreateCircuitBreakers(ctx, reqs)
}

// CreateCircuitBreakerVersions creates circuit breaker versions
func (svr *Server) CreateCircuitBreakerVersions(ctx context.Context,
	reqs []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse {
	return svr.nextSvr.CreateCircuitBreakerVersions(ctx, reqs)
}

// DeleteCircuitBreakers delete circuit breakers
func (svr *Server) DeleteCircuitBreakers(ctx context.Context,
	reqs []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse {
	return svr.nextSvr.DeleteCircuitBreakers(ctx, reqs)
}

// UpdateCircuitBreakers update circuit breakers
func (svr *Server) UpdateCircuitBreakers(ctx context.Context,
	reqs []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse {
	return svr.nextSvr.UpdateCircuitBreakers(ctx, reqs)
}

// ReleaseCircuitBreakers release circuit breakers
func (svr *Server) ReleaseCircuitBreakers(ctx context.Context,
	reqs []*apiservice.ConfigRelease) *apiservice.BatchWriteResponse {
	return svr.nextSvr.ReleaseCircuitBreakers(ctx, reqs)
}

// UnBindCircuitBreakers unbind circuit breakers
func (svr *Server) UnBindCircuitBreakers(ctx context.Context,
	reqs []*apiservice.ConfigRelease) *apiservice.BatchWriteResponse {
	return svr.nextSvr.UnBindCircuitBreakers(ctx, reqs)
}

// GetCircuitBreaker get circuit breaker
func (svr *Server) GetCircuitBreaker(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	return svr.nextSvr.GetCircuitBreaker(ctx, query)
}

// GetCircuitBreakerVersions get circuit breaker versions
func (svr *Server) GetCircuitBreakerVersions(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	return svr.nextSvr.GetCircuitBreakerVersions(ctx, query)
}

// GetMasterCircuitBreakers get master circuit breakers
func (svr *Server) GetMasterCircuitBreakers(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	return svr.nextSvr.GetMasterCircuitBreakers(ctx, query)
}

// GetReleaseCircuitBreakers get release circuit breakers
func (svr *Server) GetReleaseCircuitBreakers(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	return svr.nextSvr.GetReleaseCircuitBreakers(ctx, query)
}

// GetCircuitBreakerByService get circuit breaker by service
func (svr *Server) GetCircuitBreakerByService(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	return svr.nextSvr.GetCircuitBreakerByService(ctx, query)
}

// GetCircuitBreakerToken get circuit breaker token
func (svr *Server) GetCircuitBreakerToken(
	ctx context.Context, req *apifault.CircuitBreaker) *apiservice.Response {
	return svr.nextSvr.GetCircuitBreakerToken(ctx, req)
}
