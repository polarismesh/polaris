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

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
)

// CreateRoutingConfigs creates routing configs
func (svr *Server) CreateRoutingConfigs(
	ctx context.Context, reqs []*apitraffic.Routing) *apiservice.BatchWriteResponse {
	return svr.nextSvr.CreateRoutingConfigs(ctx, reqs)
}

// DeleteRoutingConfigs deletes routing configs
func (svr *Server) DeleteRoutingConfigs(
	ctx context.Context, reqs []*apitraffic.Routing) *apiservice.BatchWriteResponse {
	return svr.nextSvr.DeleteRoutingConfigs(ctx, reqs)
}

// UpdateRoutingConfigs updates routing configs
func (svr *Server) UpdateRoutingConfigs(
	ctx context.Context, reqs []*apitraffic.Routing) *apiservice.BatchWriteResponse {
	return svr.nextSvr.UpdateRoutingConfigs(ctx, reqs)
}

// GetRoutingConfigs gets routing configs
func (svr *Server) GetRoutingConfigs(
	ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	return svr.nextSvr.GetRoutingConfigs(ctx, query)
}
