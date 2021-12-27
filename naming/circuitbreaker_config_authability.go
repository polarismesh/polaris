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

package naming

import (
	"context"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

func (svr *serverAuthAbility) CreateCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) CreateCircuitBreaker(ctx context.Context, req *api.CircuitBreaker) *api.Response {

}

func (svr *serverAuthAbility) CreateCircuitBreakerVersions(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) CreateCircuitBreakerVersion(ctx context.Context, req *api.CircuitBreaker) *api.Response {

}

func (svr *serverAuthAbility) DeleteCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) DeleteCircuitBreaker(ctx context.Context, req *api.CircuitBreaker) *api.Response {

}

func (svr *serverAuthAbility) UpdateCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) UpdateCircuitBreaker(ctx context.Context, req *api.CircuitBreaker) *api.Response {

}

func (svr *serverAuthAbility) ReleaseCircuitBreakers(ctx context.Context, req []*api.ConfigRelease) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) ReleaseCircuitBreaker(ctx context.Context, req *api.ConfigRelease) *api.Response {

}

func (svr *serverAuthAbility) UnBindCircuitBreakers(ctx context.Context, req []*api.ConfigRelease) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) UnBindCircuitBreaker(ctx context.Context, req *api.ConfigRelease) *api.Response {

}

func (svr *serverAuthAbility) GetCircuitBreaker(query map[string]string) *api.BatchQueryResponse {

}

func (svr *serverAuthAbility) GetCircuitBreakerVersions(query map[string]string) *api.BatchQueryResponse {

}

func (svr *serverAuthAbility) GetMasterCircuitBreakers(query map[string]string) *api.BatchQueryResponse {

}

func (svr *serverAuthAbility) GetReleaseCircuitBreakers(query map[string]string) *api.BatchQueryResponse {

}

func (svr *serverAuthAbility) GetCircuitBreakerByService(query map[string]string) *api.BatchQueryResponse {

}

func (svr *serverAuthAbility) GetCircuitBreakerToken(ctx context.Context, req *api.CircuitBreaker) *api.Response {

}
