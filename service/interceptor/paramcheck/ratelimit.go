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

// CreateRateLimits implements service.DiscoverServer.
func (svr *Server) CreateRateLimits(ctx context.Context,
	request []*traffic_manage.Rule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateRateLimits(ctx, request)
}

// DeleteRateLimits implements service.DiscoverServer.
func (svr *Server) DeleteRateLimits(ctx context.Context,
	request []*traffic_manage.Rule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteRateLimits(ctx, request)
}

// EnableRateLimits implements service.DiscoverServer.
func (svr *Server) EnableRateLimits(ctx context.Context,
	request []*traffic_manage.Rule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.EnableRateLimits(ctx, request)
}

// GetRateLimits implements service.DiscoverServer.
func (svr *Server) GetRateLimits(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetRateLimits(ctx, query)
}

// UpdateRateLimits implements service.DiscoverServer.
func (svr *Server) UpdateRateLimits(ctx context.Context, request []*traffic_manage.Rule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateRateLimits(ctx, request)
}
