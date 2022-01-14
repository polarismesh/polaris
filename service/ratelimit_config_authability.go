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
	"github.com/polarismesh/polaris-server/common/model"
)

func (svr *serverAuthAbility) CreateRateLimits(ctx context.Context, reqs []*api.Rule) *api.BatchWriteResponse {
	authCtx := svr.collectRateLimitAuthContext(ctx, reqs, model.Create)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	return svr.targetServer.CreateRateLimits(ctx, reqs)
}

func (svr *serverAuthAbility) DeleteRateLimits(ctx context.Context, reqs []*api.Rule) *api.BatchWriteResponse {
	authCtx := svr.collectRateLimitAuthContext(ctx, reqs, model.Delete)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	return svr.targetServer.DeleteRateLimits(ctx, reqs)
}

func (svr *serverAuthAbility) UpdateRateLimits(ctx context.Context, reqs []*api.Rule) *api.BatchWriteResponse {
	authCtx := svr.collectRateLimitAuthContext(ctx, reqs, model.Modify)

	_, err := svr.authMgn.CheckPermission(authCtx)
	if err != nil {
		return api.NewBatchWriteResponseWithMsg(api.NotAllowedAccess, err.Error())
	}

	return svr.targetServer.UpdateRateLimits(ctx, reqs)
}

func (svr *serverAuthAbility) GetRateLimits(ctx context.Context, query map[string]string) *api.BatchQueryResponse {

	return svr.targetServer.GetRateLimits(ctx, query)
}
