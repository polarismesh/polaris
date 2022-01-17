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

package defaultauth

import (
	"context"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
)

// CreateStrategy
func (svr *serverAuthAbility) CreateStrategy(ctx context.Context, strategy *api.AuthStrategy) *api.Response {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewResponse(api.NotAllowedAccess)
	}

	ctx, errResp := verifyAuth(ctx, svr.authMgn, authToken, true)
	if errResp != nil {
		errResp.AuthStrategy = strategy
		return errResp
	}

	return svr.target.CreateStrategy(ctx, strategy)
}

// UpdateStrategy
func (svr *serverAuthAbility) UpdateStrategy(ctx context.Context, strategy *api.ModifyAuthStrategy) *api.Response {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewResponse(api.NotAllowedAccess)
	}

	ctx, errResp := verifyAuth(ctx, svr.authMgn, authToken, true)
	if errResp != nil {
		errResp.ModifyAuthStrategy = strategy
		return errResp
	}

	return svr.target.UpdateStrategy(ctx, strategy)
}

// DeleteStrategy
func (svr *serverAuthAbility) DeleteStrategies(ctx context.Context, reqs []*api.AuthStrategy) *api.BatchWriteResponse {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewBatchWriteResponse(api.NotAllowedAccess)
	}

	ctx, errResp := verifyAuth(ctx, svr.authMgn, authToken, true)
	if errResp != nil {
		resp := api.NewBatchWriteResponse(api.ExecuteSuccess)
		resp.Collect(errResp)
		return resp
	}

	return svr.target.DeleteStrategies(ctx, reqs)
}

// GetStrategies
func (svr *serverAuthAbility) GetStrategies(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewBatchQueryResponse(api.NotAllowedAccess)
	}

	ctx, errResp := verifyAuth(ctx, svr.authMgn, authToken, false)
	if errResp != nil {
		return api.NewBatchQueryResponseWithMsg(errResp.GetCode().Value, errResp.Info.Value)
	}

	return svr.target.GetStrategies(ctx, query)
}

func (svr *serverAuthAbility) GetStrategy(ctx context.Context, strategy *api.AuthStrategy) *api.Response {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewResponse(api.NotAllowedAccess)
	}

	ctx, errResp := verifyAuth(ctx, svr.authMgn, authToken, false)
	if errResp != nil {
		return api.NewResponseWithMsg(errResp.GetCode().Value, errResp.Info.Value)
	}

	return svr.target.GetStrategy(ctx, strategy)
}
