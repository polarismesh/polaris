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

package auth

import (
	"context"

	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
)

// AuthManager 权限管理通用接口定义
type AuthManager interface {

	// Initialize 执行初始化动作
	//  @param options
	//  @return error
	Initialize(options *Config, cacheMgn *cache.NamingCache) error

	// Login 登陆动作
	//  @param name
	//  @param password
	//  @return string
	//  @return error
	Login(req *api.LoginRequest) *api.Response

	// HasPermission 执行检查动作判断是否有权限
	//  @param preCtx
	//  @return bool
	//  @return error
	HasPermission(preCtx *model.AcquireContext) (bool, error)

	// ChangeOpenStatus 修改权限功能的开关状态，用于动态变更
	//  @param status
	//  @return bool
	ChangeOpenStatus(status AuthStatus) bool

	// IsOpenAuth 返回是否开启了操作鉴权，可以用于前端查询
	//  @return bool
	IsOpenAuth() bool

	// Name
	//  @return string
	Name() string

	// GetUserServer
	//  @return UserServer
	GetUserServer() UserServer

	// GetAuthStrategyServer
	//  @return AuthStrategyServer
	GetAuthStrategyServer() AuthStrategyServer

	// AfterResourceOperation 操作完资源的后置处理逻辑
	//  @param afterCtx
	AfterResourceOperation(afterCtx *model.AcquireContext)
}

// UserServer 用户数据管理 server
type UserServer interface {

	// CreateUsers 批量创建用户
	//  @param ctx
	//  @param user
	//  @return *api.Response
	CreateUsers(ctx context.Context, users []*api.User) *api.BatchWriteResponse

	// UpdateUser 更新用户信息
	//  @param ctx
	//  @param user
	//  @return *api.Response
	UpdateUser(ctx context.Context, user *api.User) *api.Response

	// DeleteUser 删除用户
	//  @param ctx
	//  @param user
	//  @return *api.Response
	DeleteUser(ctx context.Context, user *api.User) *api.Response

	// ListUsers 查询用户列表
	//  @param ctx
	//  @param query
	//  @return *api.BatchQueryResponse
	ListUsers(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetUserToken 获取用户的 token
	//  @param ctx
	//  @param user
	//  @return *api.Response
	GetUserToken(ctx context.Context, filter map[string]string) *api.Response

	// ChangeUserTokenStatus 禁止用户的token使用
	//  @param ctx
	//  @param user
	//  @return *api.Response
	ChangeUserTokenStatus(ctx context.Context, user *api.User) *api.Response

	// RefreshUserToken 重置用户的token
	//  @param ctx
	//  @param user
	//  @return *api.Response
	RefreshUserToken(ctx context.Context, user *api.User) *api.Response

	// CreateUserGroup 创建用户组
	//  @param ctx
	//  @param group
	//  @return *api.Response
	CreateUserGroup(ctx context.Context, group *api.UserGroup) *api.Response

	// UpdateUserGroup 更新用户组
	//  @param ctx
	//  @param group
	//  @return *api.Response
	UpdateUserGroup(ctx context.Context, group *api.ModifyUserGroup) *api.Response

	// DeleteUserGroup 删除用户组
	//  @param ctx
	//  @param group
	//  @return *api.Response
	DeleteUserGroup(ctx context.Context, group *api.UserGroup) *api.Response

	// ListUserGroups 查询用户组列表（不带用户详细信息）
	//  @param ctx
	//  @param query
	//  @return *api.BatchQueryResponse
	ListUserGroups(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// ListUserByGroup 根据用户组信息，查询该用户组下的用户相信
	//  @param ctx
	//  @param query
	//  @return *api.BatchQueryResponse
	ListUserByGroup(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetUserGroupToken 获取用户组的 token
	//  @param ctx
	//  @param group
	//  @return *api.Response
	GetUserGroupToken(ctx context.Context, filter map[string]string) *api.Response

	// ChangeUserGroupTokenStatus 取消用户组的 token 使用
	//  @param ctx
	//  @param group
	//  @return *api.Response
	ChangeUserGroupTokenStatus(ctx context.Context, group *api.UserGroup) *api.Response

	// RefreshUserGroupToken 重置用户组的 token
	//  @param ctx
	//  @param group
	//  @return *api.Response
	RefreshUserGroupToken(ctx context.Context, group *api.UserGroup) *api.Response
}

type AuthStrategyServer interface {

	// CreateStrategy
	//  @param ctx
	//  @param strategy
	//  @return *api.Response
	CreateStrategy(ctx context.Context, strategy *api.AuthStrategy) *api.Response

	// UpdateStrategy
	//  @param ctx
	//  @param strategy
	//  @return *api.Response
	UpdateStrategy(ctx context.Context, strategy *api.ModifyAuthStrategy) *api.Response

	// DeleteStrategy
	//  @param ctx
	//  @param strategy
	//  @return *api.Response
	DeleteStrategy(ctx context.Context, strategy *api.AuthStrategy) *api.Response

	// ListStrategy
	//  @param ctx
	//  @param query
	//  @return *api.BatchQueryResponse
	ListStrategy(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetStrategy 
	//  @param ctx 
	//  @param query 
	//  @return *api.Response 
	GetStrategy(ctx context.Context, query map[string]string) *api.Response
}

type Authority interface {
	// VerifyToken 检查Token格式是否合法
	VerifyToken(actualToken string) bool

	// VerifyNamespace 校验命名空间是否合法
	VerifyNamespace(expectToken string, actualToken string) bool

	// VerifyService 校验服务是否合法
	VerifyService(expectToken string, actualToken string) bool

	// VerifyInstance 校验实例是否合法
	VerifyInstance(expectToken string, actualToken string) bool

	// VerifyRule 校验规则是否合法
	VerifyRule(expectToken string, actualToken string) bool

	// VerifyPlatform 校验平台是否合法
	VerifyPlatform(expectToken string, actualToken string) bool

	// VerifyMesh 校验网格权限是否合法
	VerifyMesh(expectToken string, actualToken string) bool
}
