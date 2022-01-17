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

// CreateUser
func (svr *serverAuthAbility) CreateUsers(ctx context.Context, req []*api.User) *api.BatchWriteResponse {
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

	return svr.target.CreateUsers(ctx, req)
}

// UpdateUser
func (svr *serverAuthAbility) UpdateUser(ctx context.Context, user *api.User) *api.Response {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewResponse(api.NotAllowedAccess)
	}

	ctx, errResp := verifyAuth(ctx, svr.authMgn, authToken, false)
	if errResp != nil {
		errResp.User = user
		return errResp
	}

	return svr.target.UpdateUser(ctx, user)
}

// DeleteUsers 批量删除用户
func (svr *serverAuthAbility) DeleteUsers(ctx context.Context, reqs []*api.User) *api.BatchWriteResponse {
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

	return svr.target.DeleteUsers(ctx, reqs)
}

// DeleteUser
func (svr *serverAuthAbility) DeleteUser(ctx context.Context, user *api.User) *api.Response {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewResponse(api.NotAllowedAccess)
	}

	ctx, errResp := verifyAuth(ctx, svr.authMgn, authToken, true)
	if errResp != nil {
		errResp.User = user
		return errResp
	}

	return svr.target.DeleteUser(ctx, user)
}

// GetUsers
func (svr *serverAuthAbility) GetUsers(ctx context.Context, filter map[string]string) *api.BatchQueryResponse {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewBatchQueryResponse(api.NotAllowedAccess)
	}

	ctx, errResp := verifyAuth(ctx, svr.authMgn, authToken, true)
	if errResp != nil {
		return api.NewBatchQueryResponseWithMsg(errResp.GetCode().Value, errResp.Info.Value)
	}

	return svr.target.GetUsers(ctx, filter)
}

// GetUserToken
func (svr *serverAuthAbility) GetUserToken(ctx context.Context, user *api.User) *api.Response {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewResponse(api.NotAllowedAccess)
	}

	ctx, errResp := verifyAuth(ctx, svr.authMgn, authToken, false)
	if errResp != nil {
		return errResp
	}

	return svr.target.GetUserToken(ctx, user)
}

// UpdateUserToken
func (svr *serverAuthAbility) UpdateUserToken(ctx context.Context, user *api.User) *api.Response {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewResponse(api.NotAllowedAccess)
	}

	ctx, errResp := verifyAuth(ctx, svr.authMgn, authToken, true)
	if errResp != nil {
		errResp.User = user
		return errResp
	}

	return svr.target.UpdateUserToken(ctx, user)
}

// ResetUserToken
func (svr *serverAuthAbility) ResetUserToken(ctx context.Context, user *api.User) *api.Response {
	authToken := utils.ParseAuthToken(ctx)
	if authToken == "" {
		return api.NewResponse(api.NotAllowedAccess)
	}

	ctx, errResp := verifyAuth(ctx, svr.authMgn, authToken, false)
	if errResp != nil {
		errResp.User = user
		return errResp
	}

	return svr.target.ResetUserToken(ctx, user)
}
