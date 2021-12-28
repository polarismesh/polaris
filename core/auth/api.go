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

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
)

// AuthManager 权限管理通用接口定义
type AuthManager interface {
	// Initialize 执行初始化动作
	Initialize(options *Config) error

	// Login 登陆动作
	Login(name, password string) (string, error)

	// HasPermission 执行检查动作判断是否有权限
	HasPermission(preCtx *model.AcquireContext) (bool, error)

	// ChangeOpenStatus 修改权限功能的开关状态，用于动态变更
	ChangeOpenStatus(status AuthStatus) bool

	// IsOpenAuth 返回是否开启了操作鉴权，可以用于前端查询
	IsOpenAuth() bool

	// Name
	Name() string

	// GetUserServer
	GetUserServer() UserServer

	// GetAuthStrategyServer
	GetAuthStrategyServer() AuthStrategyServer

	// AfterResourceOperation 操作完资源的后置处理逻辑
	AfterResourceOperation(afterCtx *model.AcquireContext)
}

// UserServer 用户数据管理 server
type UserServer interface {
	// CreateUser 创建用户
	CreateUser(ctx context.Context, user *api.User) *api.Response

	// UpdateUser 更新用户信息
	UpdateUser(ctx context.Context, user *api.User) *api.Response

	// DeleteUser 删除用户
	DeleteUser(ctx context.Context, user *api.User) *api.Response

	// ListUsers 查询用户列表
	ListUsers(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetUserToken 获取用户的 token
	GetUserToken(ctx context.Context, user *api.User) *api.Response

	// DisableUserToken 禁止用户的token使用
	DisableUserToken(ctx context.Context, user *api.User) *api.Response

	// EnableUserToken 允许用户的token使用
	EnableUserToken(ctx context.Context, user *api.User) *api.Response

	// RefreshUserToken 重置用户的token
	RefreshUserToken(ctx context.Context, user *api.User) *api.Response

	// CreateUserGroup 创建用户组
	CreateUserGroup(ctx context.Context, group *api.UserGroup) *api.Response

	// UpdateUserGroup 更新用户组
	UpdateUserGroup(ctx context.Context, group *api.UserGroup) *api.Response

	// DeleteUserGroup 删除用户组
	DeleteUserGroup(ctx context.Context, group *api.UserGroup) *api.Response

	// ListUserGroups 查询用户组列表（不带用户详细信息）
	ListUserGroups(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetUserGroupToken 获取用户组的 token
	GetUserGroupToken(ctx context.Context, group *api.UserGroup) *api.Response

	// DisableUserGroupToken 取消用户组的 token 使用
	DisableUserGroupToken(ctx context.Context, group *api.UserGroup) *api.Response

	// EnableUserGroupToken 允许用户组 token 的使用
	EnableUserGroupToken(ctx context.Context, group *api.UserGroup) *api.Response

	// RefreshUserGroupToken 重置用户组的 token
	RefreshUserGroupToken(ctx context.Context, group *api.UserGroup) *api.Response

	// BatchAddUserToGroup 批量将用户加入用户组
	BatchAddUserToGroup(ctx context.Context, relation *api.UserGroupRelation) *api.BatchWriteResponse

	// BatchRemoveUserFromGroup 批量将用户从用户组移除
	BatchRemoveUserFromGroup(ctx context.Context, relation *api.UserGroupRelation) *api.BatchWriteResponse
}

type AuthStrategyServer interface {

	// CreateStrategy
	CreateStrategy(ctx context.Context, strategy *api.AuthStrategy) *api.Response

	// UpdateStrategy
	UpdateStrategy(ctx context.Context, strategy *api.AuthStrategy) *api.Response

	// DeleteStrategy
	DeleteStrategy(ctx context.Context, strategy *api.AuthStrategy) *api.Response

	// ListStrategy
	ListStrategy(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// AddStrategyResources
	AddStrategyResources(ctx context.Context, resources []*api.Resource) *api.BatchWriteResponse

	// DeleteStrategyResources
	DeleteStrategyResources(ctx context.Context, resources []*api.Resource) *api.BatchWriteResponse
}
