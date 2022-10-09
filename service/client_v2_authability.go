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

	apiv2 "github.com/polarismesh/polaris/common/api/v2"
)

// GetRoutingConfigV2WithCache User Client Get Service Routing Configuration Information
func (svr *serverAuthAbility) GetRoutingConfigV2WithCache(ctx context.Context, req *apiv2.Service) *apiv2.DiscoverResponse {

	return svr.targetServer.GetRoutingConfigV2WithCache(ctx, req)
}

// GetCircuitBreakerV2WithCache Fuse configuration information for obtaining services for clients
func (svr *serverAuthAbility) GetCircuitBreakerV2WithCache(ctx context.Context, req *apiv2.Service) *apiv2.DiscoverResponse {

	return svr.targetServer.GetCircuitBreakerV2WithCache(ctx, req)
}
