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

// ClientV2Server Client related operation  Client operation interface definition
type ClientV2Server interface {
	// GetRoutingConfigWithCache User Client Get Service Routing Configuration Information
	GetRoutingConfigV2WithCache(ctx context.Context, req *apiv2.Service) *apiv2.DiscoverResponse

	// GetCircuitBreakerWithCache Fuse configuration information for obtaining services for clients
	GetCircuitBreakerV2WithCache(ctx context.Context, req *apiv2.Service) *apiv2.DiscoverResponse
}

// RouteRuleV2OperateServer Routing rules related operations
type RouteRuleV2OperateServer interface {
	// CreateRoutingConfigs Batch creation routing configuration
	CreateRoutingConfigsV2(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse
	// DeleteRoutingConfigs Batch delete routing configuration
	DeleteRoutingConfigsV2(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse
	// UpdateRoutingConfigs Batch update routing configuration
	UpdateRoutingConfigsV2(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse
	// GetRoutingConfigs Inquiry route configuration to OSS
	GetRoutingConfigsV2(ctx context.Context, query map[string]string) *apiv2.BatchQueryResponse
	// EnableRoutings batch enable routing rules
	EnableRoutings(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse
}

type DiscoverServerV2 interface {
	// ClientV2Server
	ClientV2Server
	// RouteRuleV2OperateServer Routing rules operation interface definition
	RouteRuleV2OperateServer
}
