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
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
)

func (svr *serverAuthAbility) CreateCircuitBreakerRules(ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse {
	return svr.targetServer.CreateCircuitBreakerRules(ctx, request)
}

func (svr *serverAuthAbility) DeleteCircuitBreakerRules(ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse {
	return svr.targetServer.DeleteCircuitBreakerRules(ctx, request)
}

func (svr *serverAuthAbility) EnableCircuitBreakerRules(ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse {
	return svr.targetServer.EnableCircuitBreakerRules(ctx, request)
}

func (svr *serverAuthAbility) UpdateCircuitBreakerRules(ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse {
	return svr.targetServer.UpdateCircuitBreakerRules(ctx, request)
}

func (svr *serverAuthAbility) GetCircuitBreakerRules(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	return svr.targetServer.GetCircuitBreakerRules(ctx, query)
}
