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
	"github.com/polarismesh/polaris-server/store"
)

type AuthServer interface {
	Initialize(authOpt *Config, storage store.Store, cacheMgn *cache.NamingCache) error

	// Name
	Name() string

	// GetAuthManager
	GetAuthManager() AuthManager

	// AfterResourceOperation 操作完资源的后置处理逻辑
	AfterResourceOperation(afterCtx *model.AcquireContext)

	// Login 登陆动作
	Login(req *api.LoginRequest) *api.Response

	// UserOperator
	UserOperator

	// GroupOperator
	GroupOperator

	// StrategyOperator
	StrategyOperator
}

// AuthManager 权限管理通用接口定义
type AuthManager interface {

	// Initialize 执行初始化动作
	//  @param options
	//  @return error
	Initialize(options *Config, cacheMgn *cache.NamingCache) error

	// CheckPermission 执行检查动作判断是否有权限，并且将 RequestContext 进行插入一些新的数据
	//  @param preCtx
	//  @return bool
	//  @return error
	CheckPermission(preCtx *model.AcquireContext) (bool, error)

	// ChangeOpenStatus 修改权限功能的开关状态，用于动态变更
	//  @param status
	//  @return bool
	ChangeOpenStatus(status AuthStatus) bool

	// IsOpenAuth 返回是否开启了操作鉴权，可以用于前端查询
	//  @return bool
	IsOpenAuth() bool
}

// UserServer 用户数据管理 server
type UserOperator interface {

	// CreateUsers 批量创建用户
	CreateUsers(ctx context.Context, users []*api.User) *api.BatchWriteResponse

	// UpdateUser 更新用户信息
	UpdateUser(ctx context.Context, user *api.User) *api.Response

	// DeleteUser 删除用户
	DeleteUsers(ctx context.Context, users []*api.User) *api.BatchWriteResponse

	// GetUsers 查询用户列表
	GetUsers(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetUserToken 获取用户的 token
	GetUserToken(ctx context.Context, user *api.User) *api.Response

	// UpdateUserToken 禁止用户的token使用
	UpdateUserToken(ctx context.Context, user *api.User) *api.Response

	// ResetUserToken 重置用户的token
	ResetUserToken(ctx context.Context, user *api.User) *api.Response
}

// GroupOperator 用户组相关操作
type GroupOperator interface {
	// CreateGroup 创建用户组
	CreateGroup(ctx context.Context, group *api.UserGroup) *api.Response

	// UpdateGroup 更新用户组
	UpdateGroup(ctx context.Context, group *api.ModifyUserGroup) *api.Response

	// DeleteUserGroup 删除用户组
	DeleteGroups(ctx context.Context, group []*api.UserGroup) *api.BatchWriteResponse

	// GetGroups 查询用户组列表（不带用户详细信息）
	GetGroups(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetGroupUsers 根据用户组信息，查询该用户组下的用户相信
	GetGroupUsers(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetGroupToken 获取用户组的 token
	GetGroupToken(ctx context.Context, group *api.UserGroup) *api.Response

	// UpdateGroupToken 取消用户组的 token 使用
	UpdateGroupToken(ctx context.Context, group *api.UserGroup) *api.Response

	// ResetGroupToken 重置用户组的 token
	ResetGroupToken(ctx context.Context, group *api.UserGroup) *api.Response
}

// StrategyOperator 策略相关操作
type StrategyOperator interface {

	// CreateStrategy 创建策略
	CreateStrategy(ctx context.Context, strategy *api.AuthStrategy) *api.Response

	// UpdateStrategy 更新策略
	UpdateStrategy(ctx context.Context, strategy *api.ModifyAuthStrategy) *api.Response

	// DeleteStrategies 删除策略
	DeleteStrategies(ctx context.Context, reqs []*api.AuthStrategy) *api.BatchWriteResponse

	// GetStrategies 获取资源列表
	// support 1. 支持按照 principal-id + principal-role 进行查询
	// support 2. 支持普通的鉴权策略查询
	GetStrategies(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetStrategy 获取策略详细
	GetStrategy(ctx context.Context, strategy *api.AuthStrategy) *api.Response
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
