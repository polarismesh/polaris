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

package policy

import (
	"context"

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
)

// CreateRoles 批量创建角色
func (svr *Server) CreateRoles(ctx context.Context, reqs []*apisecurity.Role) *apiservice.BatchWriteResponse {
	return nil
}

// UpdateRoles 批量更新角色
func (svr *Server) UpdateRoles(ctx context.Context, reqs []*apisecurity.Role) *apiservice.BatchWriteResponse {
	return nil
}

// DeleteRoles 批量删除角色
func (svr *Server) DeleteRoles(ctx context.Context, reqs []*apisecurity.Role) *apiservice.BatchWriteResponse {
	return nil
}

// GetRoles 查询角色列表
func (svr *Server) GetRoles(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	return nil
}
