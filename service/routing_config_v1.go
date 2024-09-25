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

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
)

var (
	// RoutingConfigFilterAttrs router config filter attrs
	RoutingConfigFilterAttrs = map[string]bool{
		"service":   true,
		"namespace": true,
		"offset":    true,
		"limit":     true,
	}
)

// CreateRoutingConfigs Create a routing configuration
func (s *Server) CreateRoutingConfigs(ctx context.Context, req []*apitraffic.Routing) *apiservice.BatchWriteResponse {
	resp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, entry := range req {
		api.Collect(resp, s.CreateRoutingConfig(ctx, entry))
	}
	return api.FormatBatchWriteResponse(resp)
}

// CreateRoutingConfig Create a routing configuration, Creating route configuration requires locking
// services to prevent the service from being deleted
// Deprecated: This method is ready to abandon
func (s *Server) CreateRoutingConfig(ctx context.Context, req *apitraffic.Routing) *apiservice.Response {
	resps := api.NewResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}

// DeleteRoutingConfigs Batch delete routing configuration
func (s *Server) DeleteRoutingConfigs(ctx context.Context, req []*apitraffic.Routing) *apiservice.BatchWriteResponse {
	out := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, entry := range req {
		resp := s.DeleteRoutingConfig(ctx, entry)
		api.Collect(out, resp)
	}
	return api.FormatBatchWriteResponse(out)
}

// DeleteRoutingConfig Delete a routing configuration
// Deprecated: This method is ready to abandon
func (s *Server) DeleteRoutingConfig(ctx context.Context, req *apitraffic.Routing) *apiservice.Response {
	resps := api.NewResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}

// UpdateRoutingConfigs Batch update routing configuration
func (s *Server) UpdateRoutingConfigs(ctx context.Context, req []*apitraffic.Routing) *apiservice.BatchWriteResponse {
	out := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, entry := range req {
		resp := s.UpdateRoutingConfig(ctx, entry)
		api.Collect(out, resp)
	}

	return api.FormatBatchWriteResponse(out)
}

// UpdateRoutingConfig Update a routing configuration
// Deprecated: 该方法准备舍弃
func (s *Server) UpdateRoutingConfig(ctx context.Context, req *apitraffic.Routing) *apiservice.Response {
	resps := api.NewResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}

// GetRoutingConfigs Get the routing configuration in batches, and provide the interface of
// the query routing configuration to the OSS
// Deprecated: This method is ready to abandon
func (s *Server) GetRoutingConfigs(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	resps := api.NewBatchQueryResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}
