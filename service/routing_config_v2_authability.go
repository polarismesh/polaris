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

// CreateRoutingConfigsV2 批量创建路由配置
func (svr *serverAuthAbility) CreateRoutingConfigsV2(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse {
	return svr.targetServer.CreateRoutingConfigsV2(ctx, req)
}

// DeleteRoutingConfigsV2 批量删除路由配置
func (svr *serverAuthAbility) DeleteRoutingConfigsV2(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse {
	return svr.targetServer.DeleteRoutingConfigsV2(ctx, req)
}

// UpdateRoutingConfigsV2 批量更新路由配置
func (svr *serverAuthAbility) UpdateRoutingConfigsV2(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse {
	return svr.targetServer.UpdateRoutingConfigsV2(ctx, req)
}

// EnableRoutings batch enable routing rules
func (svr *serverAuthAbility) EnableRoutings(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse {
	return svr.targetServer.EnableRoutings(ctx, req)
}

// GetRoutingConfigsV2 提供给OSS的查询路由配置的接口
func (svr *serverAuthAbility) GetRoutingConfigsV2(ctx context.Context, query map[string]string) *apiv2.BatchQueryResponse {
	return svr.targetServer.GetRoutingConfigsV2(ctx, query)
}
