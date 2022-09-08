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

	apiv2 "github.com/polarismesh/polaris-server/common/api/v2"
)

var (
	// RoutingConfigFilterAttrs router config filter attrs
	RoutingConfigV2FilterAttrs = map[string]bool{
		"name":      true,
		"namespace": true,
		"offset":    true,
		"limit":     true,
	}
)

// CreateRoutingConfigsV2 批量创建路由配置
func (s *Server) CreateRoutingConfigsV2(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse {

	return nil
}

// CreateRoutingConfigV2 创建一个路由配置
// 创建路由配置需要锁住服务，防止服务被删除
func (s *Server) CreateRoutingConfigV2(ctx context.Context, req *apiv2.Routing) *apiv2.Response {

	return nil
}

// DeleteRoutingConfigsV2 批量删除路由配置
func (s *Server) DeleteRoutingConfigsV2(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse {

	return nil
}

// DeleteRoutingConfigV2 删除一个路由配置
func (s *Server) DeleteRoutingConfigV2(ctx context.Context, req *apiv2.Routing) *apiv2.Response {

	return nil
}

// UpdateRoutingConfigsV2 批量更新路由配置
func (s *Server) UpdateRoutingConfigsV2(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse {

	return nil
}

// UpdateRoutingConfigV2 更新单个路由配置
func (s *Server) UpdateRoutingConfigV2(ctx context.Context, req *apiv2.Routing) *apiv2.Response {

	return nil
}

// GetRoutingConfigs 提供给OSS的查询路由配置的接口
func (s *Server) GetRoutingConfigsV2(ctx context.Context, query map[string]string) *apiv2.BatchQueryResponse {

	return nil
}

// checkRoutingConfig 检查路由配置基础参数有效性
func checkRoutingConfigV2(req *apiv2.Routing) *apiv2.Response {

	return nil
}
