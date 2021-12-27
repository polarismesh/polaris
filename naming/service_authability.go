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

package naming

import (
	"context"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

// CreateServices 批量创建服务
func (svr *serverAuthAbility) CreateServices(ctx context.Context, req []*api.Service) *api.BatchWriteResponse {

}

// CreateService 创建单个服务
func (svr *serverAuthAbility) CreateService(ctx context.Context, req *api.Service) *api.Response {

}

// DeleteServices 批量删除服务
func (svr *serverAuthAbility) DeleteServices(ctx context.Context, req []*api.Service) *api.BatchWriteResponse {

}

// DeleteService Delete a single service, the delete operation needs to lock the service
// 	to prevent the instance of the service associated with the service or a new operation.
func (svr *serverAuthAbility) DeleteService(ctx context.Context, req *api.Service) *api.Response {

}

func (svr *serverAuthAbility) UpdateServices(ctx context.Context, req []*api.Service) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) UpdateService(ctx context.Context, req *api.Service) *api.Response {

}

func (svr *serverAuthAbility) UpdateServiceToken(ctx context.Context, req *api.Service) *api.Response {

}

func (svr *serverAuthAbility) GetServices(ctx context.Context, query map[string]string) *api.BatchQueryResponse {

}

func (svr *serverAuthAbility) GetServicesCount() *api.BatchQueryResponse {

}

func (svr *serverAuthAbility) GetServiceToken(ctx context.Context, req *api.Service) *api.Response {

}

func (svr *serverAuthAbility) GetServiceOwner(ctx context.Context, req []*api.Service) *api.BatchQueryResponse {

}
