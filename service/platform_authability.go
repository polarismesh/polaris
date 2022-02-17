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

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

//
func (svr *serverAuthAbility) CreatePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse {
	return svr.targetServer.CreatePlatforms(ctx, req)
}

//
func (svr *serverAuthAbility) CreatePlatform(ctx context.Context, req *api.Platform) *api.Response {
	return svr.targetServer.CreatePlatform(ctx, req)
}

//
func (svr *serverAuthAbility) UpdatePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse {
	return svr.targetServer.UpdatePlatforms(ctx, req)
}

//
func (svr *serverAuthAbility) UpdatePlatform(ctx context.Context, req *api.Platform) *api.Response {
	return svr.targetServer.UpdatePlatform(ctx, req)
}

//
func (svr *serverAuthAbility) DeletePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse {
	return svr.targetServer.DeletePlatforms(ctx, req)
}

//
func (svr *serverAuthAbility) DeletePlatform(ctx context.Context, req *api.Platform) *api.Response {
	return svr.targetServer.DeletePlatform(ctx, req)
}

//
func (svr *serverAuthAbility) GetPlatforms(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	return svr.targetServer.GetPlatforms(ctx, query)
}

//
func (svr *serverAuthAbility) GetPlatformToken(ctx context.Context, req *api.Platform) *api.Response {
	return svr.targetServer.GetPlatformToken(ctx, req)
}
