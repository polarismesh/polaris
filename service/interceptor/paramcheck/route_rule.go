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

package paramcheck

import (
	"context"

	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
)

// UpdateRoutingConfigs implements service.DiscoverServer.
func (svr *Server) UpdateRoutingConfigs(ctx context.Context, req []*traffic_manage.Routing) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateRoutingConfigs(ctx, req)
}

// UpdateRoutingConfigsV2 implements service.DiscoverServer.
func (svr *Server) UpdateRoutingConfigsV2(ctx context.Context, req []*traffic_manage.RouteRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateRoutingConfigsV2(ctx, req)
}

// QueryRoutingConfigsV2 implements service.DiscoverServer.
func (svr *Server) QueryRoutingConfigsV2(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.QueryRoutingConfigsV2(ctx, query)
}

// GetRoutingConfigs implements service.DiscoverServer.
func (svr *Server) GetRoutingConfigs(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetRoutingConfigs(ctx, query)
}

// EnableRoutings implements service.DiscoverServer.
func (svr *Server) EnableRoutings(ctx context.Context,
	req []*traffic_manage.RouteRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.EnableRoutings(ctx, req)
}

// CreateRoutingConfigs implements service.DiscoverServer.
func (svr *Server) CreateRoutingConfigs(ctx context.Context,
	req []*traffic_manage.Routing) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateRoutingConfigs(ctx, req)
}

// DeleteRoutingConfigs implements service.DiscoverServer.
func (svr *Server) DeleteRoutingConfigs(ctx context.Context,
	req []*traffic_manage.Routing) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteRoutingConfigs(ctx, req)
}

// CreateRoutingConfigsV2 implements service.DiscoverServer.
func (svr *Server) CreateRoutingConfigsV2(ctx context.Context,
	req []*traffic_manage.RouteRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateRoutingConfigsV2(ctx, req)
}

// DeleteRoutingConfigsV2 implements service.DiscoverServer.
func (svr *Server) DeleteRoutingConfigsV2(ctx context.Context,
	req []*traffic_manage.RouteRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteRoutingConfigsV2(ctx, req)
}
