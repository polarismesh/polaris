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

package paramcheck

import (
	"context"

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/auth"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	authmodel "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/store"
)

func NewServer(nextSvr auth.UserServer) auth.UserServer {
	return &Server{
		nextSvr: nextSvr,
	}
}

type Server struct {
	nextSvr   auth.UserServer
	policySvr auth.StrategyServer
}

// Initialize 初始化
func (svr *Server) Initialize(authOpt *auth.Config, storage store.Store, policyMgr auth.StrategyServer, cacheMgr cachetypes.CacheManager) error {
	return svr.nextSvr.Initialize(authOpt, storage, policyMgr, cacheMgr)
}

// Name 用户数据管理server名称
func (svr *Server) Name() string {
	return svr.nextSvr.Name()
}

// Login 登录动作
func (svr *Server) Login(req *apisecurity.LoginRequest) *apiservice.Response {
	return svr.nextSvr.Login(req)
}

// CheckCredential 检查当前操作用户凭证
func (svr *Server) CheckCredential(authCtx *authmodel.AcquireContext) error {
	return svr.nextSvr.CheckCredential(authCtx)
}

// GetUserHelper
func (svr *Server) GetUserHelper() auth.UserHelper {
	return svr.nextSvr.GetUserHelper()
}

// CreateUsers 批量创建用户
func (svr *Server) CreateUsers(ctx context.Context, users []*apisecurity.User) *apiservice.BatchWriteResponse {
	return svr.nextSvr.CreateUsers(ctx, users)
}

// UpdateUser 更新用户信息
func (svr *Server) UpdateUser(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	return svr.nextSvr.UpdateUser(ctx, user)
}

// UpdateUserPassword 更新用户密码
func (svr *Server) UpdateUserPassword(ctx context.Context, req *apisecurity.ModifyUserPassword) *apiservice.Response {
	return svr.nextSvr.UpdateUserPassword(ctx, req)
}

// DeleteUsers 批量删除用户
func (svr *Server) DeleteUsers(ctx context.Context, users []*apisecurity.User) *apiservice.BatchWriteResponse {
	return svr.nextSvr.DeleteUsers(ctx, users)
}

// GetUsers 查询用户列表
func (svr *Server) GetUsers(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	return svr.nextSvr.GetUsers(ctx, query)
}

// GetUserToken 获取用户的 token
func (svr *Server) GetUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	return svr.nextSvr.GetUserToken(ctx, user)
}

// UpdateUserToken 禁止用户的token使用
func (svr *Server) UpdateUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	return svr.nextSvr.UpdateUserToken(ctx, user)
}

// ResetUserToken 重置用户的token
func (svr *Server) ResetUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	return svr.nextSvr.ResetUserToken(ctx, user)
}

// CreateGroup 创建用户组
func (svr *Server) CreateGroup(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	return svr.nextSvr.CreateGroup(ctx, group)
}

// UpdateGroups 更新用户组
func (svr *Server) UpdateGroups(ctx context.Context, groups []*apisecurity.ModifyUserGroup) *apiservice.BatchWriteResponse {
	return svr.nextSvr.UpdateGroups(ctx, groups)
}

// DeleteGroups 批量删除用户组
func (svr *Server) DeleteGroups(ctx context.Context, groups []*apisecurity.UserGroup) *apiservice.BatchWriteResponse {
	return svr.nextSvr.DeleteGroups(ctx, groups)
}

// GetGroups 查询用户组列表（不带用户详细信息）
func (svr *Server) GetGroups(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	return svr.nextSvr.GetGroups(ctx, query)
}

// GetGroup 根据用户组信息，查询该用户组下的用户相信
func (svr *Server) GetGroup(ctx context.Context, req *apisecurity.UserGroup) *apiservice.Response {
	return svr.nextSvr.GetGroup(ctx, req)
}

// GetGroupToken 获取用户组的 token
func (svr *Server) GetGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	return svr.nextSvr.GetGroupToken(ctx, group)
}

// UpdateGroupToken 取消用户组的 token 使用
func (svr *Server) UpdateGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	return svr.nextSvr.UpdateGroupToken(ctx, group)
}

// ResetGroupToken 重置用户组的 token
func (svr *Server) ResetGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	return svr.nextSvr.ResetGroupToken(ctx, group)
}
