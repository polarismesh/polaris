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
)

// AcquireContext 每次鉴权请求上下文信息
type AcquireContext struct {

	// Token 本次请求的访问凭据
	Token string

	// Module 来自那个业务层（服务注册与服务治理、配置模块）
	Module BzModule

	// Operation 本次操作涉及的动作
	Operation ResourceOperation

	// Resources 本次
	Resources map[api.ResourceType][]string

	// Attachment 携带信息，用于操作完权限检查和资源操作的后置处理逻辑，解决信息需要二次查询问题
	Attachment map[string]interface{}
}

// AuthManager 权限管理通用接口定义
type AuthManager interface {
	// Initialize 执行初始化动作
	Initialize(options *Config) error

	// Login 登陆动作
	Login(name, password string) (string, error)

	// HasPermission 执行检查动作判断是否有权限
	HasPermission(preCtx *AcquireContext) (bool, error)

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
	AfterResourceOperation(afterCtx *AcquireContext)
}

// UserServer 用户数据管理 server
type UserServer interface {
	// CreateUser
	CreateUser(ctx context.Context, user *api.User) *api.Response

	// UpdateUser
	UpdateUser(ctx context.Context, user *api.User) *api.Response

	// DeleteUser
	DeleteUser(ctx context.Context, user *api.User) *api.Response

	// ListUsers
	ListUsers(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetUserToken
	GetUserToken(ctx context.Context, user *api.User) *api.Response

	// CreateUserGroup
	CreateUserGroup(ctx context.Context, group *api.UserGroup) *api.Response

	// UpdateUserGroup
	UpdateUserGroup(ctx context.Context, group *api.UserGroup) *api.Response

	// DeleteUserGroup
	DeleteUserGroup(ctx context.Context, group *api.UserGroup) *api.Response

	// ListUserGroups
	ListUserGroups(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetUserGroupToken
	GetUserGroupToken(ctx context.Context, group *api.UserGroup) *api.Response

	// BatchAddUserToGroup
	BatchAddUserToGroup(ctx context.Context, relation *api.UserGroupRelation) *api.BatchWriteResponse

	// BatchRemoveUserToGroup
	BatchRemoveUserToGroup(ctx context.Context, relation *api.UserGroupRelation) *api.BatchWriteResponse
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
}
