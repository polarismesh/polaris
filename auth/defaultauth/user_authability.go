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
)

// CreateUsers 创建用户，只能由超级账户 or 主账户调用
//  case 1. 超级账户调用：创建的是主账户
//  case 2. 主账户调用：创建的是子账户
func (svr *serverAuthAbility) CreateUsers(ctx context.Context, req []*api.User) *api.BatchWriteResponse {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		resp := api.NewBatchWriteResponse(api.ExecuteSuccess)
		resp.Collect(rsp)
		return resp
	}

	return svr.target.CreateUsers(ctx, req)
}

// UpdateUser 更新用户，任意账户均可以操作
// 用户token被禁止也只是表示不能对北极星资源执行写操作，但是改用户信息还是可以执行的
func (svr *serverAuthAbility) UpdateUser(ctx context.Context, user *api.User) *api.Response {
	ctx, rsp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if rsp != nil {
		rsp.User = user
		return rsp
	}

	return svr.target.UpdateUser(ctx, user)
}

// UpdateUserPassword 更新用户信息
func (svr *serverAuthAbility) UpdateUserPassword(ctx context.Context, req *api.ModifyUserPassword) *api.Response {
	ctx, rsp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if rsp != nil {
		return rsp
	}

	return svr.target.UpdateUserPassword(ctx, req)
}

// DeleteUsers 批量删除用户，只能由超级账户 or 主账户操作
func (svr *serverAuthAbility) DeleteUsers(ctx context.Context, reqs []*api.User) *api.BatchWriteResponse {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		resp := api.NewBatchWriteResponse(api.ExecuteSuccess)
		resp.Collect(rsp)
		return resp
	}

	return svr.target.DeleteUsers(ctx, reqs)
}

// DeleteUser 删除用户，只能由超级账户 or 主账户操作
func (svr *serverAuthAbility) DeleteUser(ctx context.Context, user *api.User) *api.Response {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		rsp.User = user
		return rsp
	}

	return svr.target.DeleteUser(ctx, user)
}

// GetUsers 获取用户列表，任意账户均可以操作
func (svr *serverAuthAbility) GetUsers(ctx context.Context, filter map[string]string) *api.BatchQueryResponse {
	ctx, rsp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if rsp != nil {
		return api.NewBatchQueryResponseWithMsg(rsp.GetCode().Value, rsp.Info.Value)
	}

	return svr.target.GetUsers(ctx, filter)
}

// GetUserToken 获取用户token，任意账户均可以操作
func (svr *serverAuthAbility) GetUserToken(ctx context.Context, user *api.User) *api.Response {
	ctx, rsp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if rsp != nil {
		return rsp
	}

	return svr.target.GetUserToken(ctx, user)
}

// UpdateUserToken 更新用户的 token 状态，只允许超级、主账户进行操作
func (svr *serverAuthAbility) UpdateUserToken(ctx context.Context, user *api.User) *api.Response {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		rsp.User = user
		return rsp
	}

	return svr.target.UpdateUserToken(ctx, user)
}

// ResetUserToken 重置用户token，允许子账户进行操作
func (svr *serverAuthAbility) ResetUserToken(ctx context.Context, user *api.User) *api.Response {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, NotOwner)
	if rsp != nil {
		rsp.User = user
		return rsp
	}

	return svr.target.ResetUserToken(ctx, user)
}
