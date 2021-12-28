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
	"github.com/polarismesh/polaris-server/core/auth"
)

// strategyServerAuth 用户数据管理 server
type strategyServerAuth struct {
	authMgn *defaultAuthManager
	target  auth.AuthStrategyServer
}

func newStrategyServerWithAuth(authMgn *defaultAuthManager, target auth.AuthStrategyServer) auth.AuthStrategyServer {
	return &strategyServerAuth{
		authMgn: authMgn,
		target:  target,
	}
}

// CreateStrategy
func (svr *strategyServerAuth) CreateStrategy(ctx context.Context, strategy *api.AuthStrategy) *api.Response {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewResponse(api.NotAllowedAccess)
	}

	_, errResp := verifyAuth(ctx, svr.authMgn, strategy.AuthToken.GetValue(), true)
	if errResp != nil {
		errResp.AuthStrategy = strategy
		return errResp
	}

	return svr.target.CreateStrategy(ctx, strategy)
}

// UpdateStrategy
func (svr *strategyServerAuth) UpdateStrategy(ctx context.Context, strategy *api.AuthStrategy) *api.Response {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewResponse(api.NotAllowedAccess)
	}

	_, errResp := verifyAuth(ctx, svr.authMgn, strategy.AuthToken.GetValue(), true)
	if errResp != nil {
		errResp.AuthStrategy = strategy
		return errResp
	}

	return svr.target.UpdateStrategy(ctx, strategy)
}

// DeleteStrategy
func (svr *strategyServerAuth) DeleteStrategy(ctx context.Context, strategy *api.AuthStrategy) *api.Response {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewResponse(api.NotAllowedAccess)
	}

	_, errResp := verifyAuth(ctx, svr.authMgn, strategy.AuthToken.GetValue(), true)
	if errResp != nil {
		errResp.AuthStrategy = strategy
		return errResp
	}

	return svr.target.DeleteStrategy(ctx, strategy)
}

// ListStrategy
func (svr *strategyServerAuth) ListStrategy(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewBatchQueryResponse(api.NotAllowedAccess)
	}

	_, errResp := verifyAuth(ctx, svr.authMgn, authToken, false)
	if errResp != nil {
		return api.NewBatchQueryResponseWithMsg(errResp.GetCode().Value, errResp.Info.Value)
	}

	return svr.target.ListStrategy(ctx, query)
}

// AddStrategyResources
func (svr *strategyServerAuth) AddStrategyResources(ctx context.Context, resources []*api.Resource) *api.BatchWriteResponse {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewBatchWriteResponse(api.NotAllowedAccess)
	}

	_, errResp := verifyAuth(ctx, svr.authMgn, authToken, true)
	if errResp != nil {
		return api.NewBatchWriteResponseWithMsg(errResp.GetCode().Value, errResp.Info.Value)
	}

	return svr.target.AddStrategyResources(ctx, resources)
}

// DeleteStrategyResources
func (svr *strategyServerAuth) DeleteStrategyResources(ctx context.Context, resources []*api.Resource) *api.BatchWriteResponse {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewBatchWriteResponse(api.NotAllowedAccess)
	}

	_, errResp := verifyAuth(ctx, svr.authMgn, authToken, true)
	if errResp != nil {
		return api.NewBatchWriteResponseWithMsg(errResp.GetCode().Value, errResp.Info.Value)
	}

	return svr.target.DeleteStrategyResources(ctx, resources)
}
