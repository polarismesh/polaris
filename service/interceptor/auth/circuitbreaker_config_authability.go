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
func (svr *ServerAuthAbility) CreateCircuitBreakers(ctx context.Context,
	reqs []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse {
	return svr.targetServer.CreateCircuitBreakers(ctx, reqs)
}

// CreateCircuitBreakerVersions creates circuit breaker versions
func (svr *ServerAuthAbility) CreateCircuitBreakerVersions(ctx context.Context,
	reqs []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse {
	return svr.targetServer.CreateCircuitBreakerVersions(ctx, reqs)
}

// DeleteCircuitBreakers delete circuit breakers
func (svr *ServerAuthAbility) DeleteCircuitBreakers(ctx context.Context,
	reqs []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse {
	return svr.targetServer.DeleteCircuitBreakers(ctx, reqs)
}

// UpdateCircuitBreakers update circuit breakers
func (svr *ServerAuthAbility) UpdateCircuitBreakers(ctx context.Context,
	reqs []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse {
	return svr.targetServer.UpdateCircuitBreakers(ctx, reqs)
}

// ReleaseCircuitBreakers release circuit breakers
func (svr *ServerAuthAbility) ReleaseCircuitBreakers(ctx context.Context,
	reqs []*apiservice.ConfigRelease) *apiservice.BatchWriteResponse {
	return svr.targetServer.ReleaseCircuitBreakers(ctx, reqs)
}

// UnBindCircuitBreakers unbind circuit breakers
func (svr *ServerAuthAbility) UnBindCircuitBreakers(ctx context.Context,
	reqs []*apiservice.ConfigRelease) *apiservice.BatchWriteResponse {
	return svr.targetServer.UnBindCircuitBreakers(ctx, reqs)
}

// GetCircuitBreaker get circuit breaker
func (svr *ServerAuthAbility) GetCircuitBreaker(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	return svr.targetServer.GetCircuitBreaker(ctx, query)
}

// GetCircuitBreakerVersions get circuit breaker versions
func (svr *ServerAuthAbility) GetCircuitBreakerVersions(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	return svr.targetServer.GetCircuitBreakerVersions(ctx, query)
}

// GetMasterCircuitBreakers get master circuit breakers
func (svr *ServerAuthAbility) GetMasterCircuitBreakers(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	return svr.targetServer.GetMasterCircuitBreakers(ctx, query)
}

// GetReleaseCircuitBreakers get release circuit breakers
func (svr *ServerAuthAbility) GetReleaseCircuitBreakers(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	return svr.targetServer.GetReleaseCircuitBreakers(ctx, query)
}

// GetCircuitBreakerByService get circuit breaker by service
func (svr *ServerAuthAbility) GetCircuitBreakerByService(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	return svr.targetServer.GetCircuitBreakerByService(ctx, query)
}

// GetCircuitBreakerToken get circuit breaker token
func (svr *ServerAuthAbility) GetCircuitBreakerToken(
	ctx context.Context, req *apifault.CircuitBreaker) *apiservice.Response {
	return svr.targetServer.GetCircuitBreakerToken(ctx, req)
}
