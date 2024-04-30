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

package authcheck

import (
	"context"

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
)

// CreateStrategy 创建鉴权策略
func (svr *Server) CreateStrategy(ctx context.Context, req *apisecurity.AuthStrategy) *apiservice.Response {
	return svr.handleCreateStrategy(ctx, req)
}

// UpdateStrategies 批量修改鉴权
func (svr *Server) UpdateStrategies(
	ctx context.Context, reqs []*apisecurity.ModifyAuthStrategy) *apiservice.BatchWriteResponse {
	return svr.handleUpdateStrategies(ctx, reqs)
}

// DeleteStrategies 批量删除鉴权策略
func (svr *Server) DeleteStrategies(
	ctx context.Context, reqs []*apisecurity.AuthStrategy) *apiservice.BatchWriteResponse {
	return svr.handleDeleteStrategies(ctx, reqs)
}

// GetStrategies 批量查询鉴权策略
func (svr *Server) GetStrategies(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	return svr.handleGetStrategies(ctx, query)
}

// GetStrategy 查询单个鉴权策略
func (svr *Server) GetStrategy(ctx context.Context, req *apisecurity.AuthStrategy) *apiservice.Response {
	return svr.handleGetStrategy(ctx, req)
}

// GetPrincipalResources 查询鉴权策略所属资源
func (svr *Server) GetPrincipalResources(ctx context.Context, query map[string]string) *apiservice.Response {
	return svr.handleGetPrincipalResources(ctx, query)
}
