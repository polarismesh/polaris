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

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/store"
)

type userAuthAbility struct {
	authMgn *defaultAuthChecker
	target  *server
	*groupAuthAbility
}

// Initialize 执行初始化动作
func (svr *userAuthAbility) Initialize(authOpt *auth.Config, storage store.Store,
	cacheMgn *cache.CacheManager) error {
	var (
		history = plugin.GetHistory()
		authMgn = &defaultAuthChecker{}
	)
	if err := authMgn.Initialize(authOpt, storage, cacheMgn); err != nil {
		return err
	}

	svr.authMgn = authMgn
	svr.target = &server{
		storage:  storage,
		history:  history,
		cacheMgn: cacheMgn,
		authMgn:  authMgn,
	}
	svr.groupAuthAbility = &groupAuthAbility{
		authMgn: svr.authMgn,
		target:  svr.target,
	}

	return nil
}

// Name of the user operator plugin
func (svr *userAuthAbility) Name() string {
	return auth.DefaultUserMgnPluginName
}

// CreateUsers 创建用户，只能由超级账户 or 主账户调用
//
//	case 1. 超级账户调用：创建的是主账户
//	case 2. 主账户调用：创建的是子账户
func (svr *userAuthAbility) CreateUsers(ctx context.Context, req []*apisecurity.User) *apiservice.BatchWriteResponse {
	ctx, rsp := verifyAuth(ctx, WriteOp, MustOwner, svr.authMgn)
	if rsp != nil {
		resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(resp, rsp)
		return resp
	}

	return svr.target.CreateUsers(ctx, req)
}

// UpdateUser 更新用户，任意账户均可以操作
// 用户token被禁止也只是表示不能对北极星资源执行写操作，但是改用户信息还是可以执行的
func (svr *userAuthAbility) UpdateUser(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	ctx, rsp := verifyAuth(ctx, ReadOp, NotOwner, svr.authMgn)
	if rsp != nil {
		rsp.User = user
		return rsp
	}

	return svr.target.UpdateUser(ctx, user)
}

// UpdateUserPassword 更新用户信息
func (svr *userAuthAbility) UpdateUserPassword(
	ctx context.Context, req *apisecurity.ModifyUserPassword) *apiservice.Response {
	ctx, rsp := verifyAuth(ctx, ReadOp, NotOwner, svr.authMgn)
	if rsp != nil {
		return rsp
	}

	return svr.target.UpdateUserPassword(ctx, req)
}

// DeleteUsers 批量删除用户，只能由超级账户 or 主账户操作
func (svr *userAuthAbility) DeleteUsers(
	ctx context.Context, reqs []*apisecurity.User) *apiservice.BatchWriteResponse {
	ctx, rsp := verifyAuth(ctx, WriteOp, MustOwner, svr.authMgn)
	if rsp != nil {
		resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(resp, rsp)
		return resp
	}

	return svr.target.DeleteUsers(ctx, reqs)
}

// DeleteUser 删除用户，只能由超级账户 or 主账户操作
func (svr *userAuthAbility) DeleteUser(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	ctx, rsp := verifyAuth(ctx, WriteOp, MustOwner, svr.authMgn)
	if rsp != nil {
		rsp.User = user
		return rsp
	}

	return svr.target.DeleteUser(ctx, user)
}

// GetUsers 获取用户列表，任意账户均可以操作
func (svr *userAuthAbility) GetUsers(ctx context.Context, filter map[string]string) *apiservice.BatchQueryResponse {
	ctx, rsp := verifyAuth(ctx, ReadOp, NotOwner, svr.authMgn)
	if rsp != nil {
		return api.NewAuthBatchQueryResponseWithMsg(apimodel.Code(rsp.GetCode().Value), rsp.Info.Value)
	}

	return svr.target.GetUsers(ctx, filter)
}

// GetUserToken 获取用户token，任意账户均可以操作
func (svr *userAuthAbility) GetUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	ctx, rsp := verifyAuth(ctx, ReadOp, NotOwner, svr.authMgn)
	if rsp != nil {
		return rsp
	}

	return svr.target.GetUserToken(ctx, user)
}

// UpdateUserToken 更新用户的 token 状态，只允许超级、主账户进行操作
func (svr *userAuthAbility) UpdateUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	ctx, rsp := verifyAuth(ctx, WriteOp, MustOwner, svr.authMgn)
	if rsp != nil {
		rsp.User = user
		return rsp
	}

	return svr.target.UpdateUserToken(ctx, user)
}

// ResetUserToken 重置用户token，允许子账户进行操作
func (svr *userAuthAbility) ResetUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	ctx, rsp := verifyAuth(ctx, WriteOp, NotOwner, svr.authMgn)
	if rsp != nil {
		rsp.User = user
		return rsp
	}

	return svr.target.ResetUserToken(ctx, user)
}

// Login login servers
func (svr *userAuthAbility) Login(req *apisecurity.LoginRequest) *apiservice.Response {
	return svr.target.Login(req)
}
