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

// AuthServer 鉴权 Server
type AuthServer interface {
	// Initialize 初始化
	Initialize(authOpt *Config, storage store.Store, cacheMgn *cache.CacheManager) error

	// Name 获取服务名称
	Name() string

	// GetAuthChecker 获取鉴权检查器
	GetAuthChecker() AuthChecker

	// AfterResourceOperation 操作完资源的后置处理逻辑
	AfterResourceOperation(afterCtx *model.AcquireContext) error

	// Login 登陆动作
	Login(req *api.LoginRequest) *api.Response

	// UserOperator 用户操作
	UserOperator

	// GroupOperator 组操作
	GroupOperator

	// StrategyOperator 策略操作
	StrategyOperator
}

// AuthChecker 权限管理通用接口定义
type AuthChecker interface {
	// Initialize 执行初始化动作
	Initialize(options *Config, cacheMgn *cache.CacheManager) error
	// VerifyToken 验证令牌
	VerifyCredential(preCtx *model.AcquireContext) error
	// CheckClientPermission 执行检查客户端动作判断是否有权限，并且对 RequestContext 注入操作者数据
	CheckClientPermission(preCtx *model.AcquireContext) (bool, error)
	// CheckConsolePermission 执行检查控制台动作判断是否有权限，并且对 RequestContext 注入操作者数据
	CheckConsolePermission(preCtx *model.AcquireContext) (bool, error)
	// IsOpenConsoleAuth 返回是否开启了操作鉴权，可以用于前端查询
	IsOpenConsoleAuth() bool
	// IsOpenClientAuth
	IsOpenClientAuth() bool
}

// UserOperator 用户数据管理 server
type UserOperator interface {

	// CreateUsers 批量创建用户
	CreateUsers(ctx context.Context, users []*api.User) *api.BatchWriteResponse

	// UpdateUser 更新用户信息
	UpdateUser(ctx context.Context, user *api.User) *api.Response

	// UpdateUserPassword 更新用户密码
	UpdateUserPassword(ctx context.Context, req *api.ModifyUserPassword) *api.Response

	// DeleteUsers 批量删除用户
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

	// UpdateGroups 更新用户组
	UpdateGroups(ctx context.Context, groups []*api.ModifyUserGroup) *api.BatchWriteResponse

	// DeleteGroups 批量删除用户组
	DeleteGroups(ctx context.Context, group []*api.UserGroup) *api.BatchWriteResponse

	// GetGroups 查询用户组列表（不带用户详细信息）
	GetGroups(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetGroup 根据用户组信息，查询该用户组下的用户相信
	GetGroup(ctx context.Context, req *api.UserGroup) *api.Response

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

	// UpdateStrategies 批量更新策略
	UpdateStrategies(ctx context.Context, reqs []*api.ModifyAuthStrategy) *api.BatchWriteResponse

	// DeleteStrategies 删除策略
	DeleteStrategies(ctx context.Context, reqs []*api.AuthStrategy) *api.BatchWriteResponse

	// GetStrategies 获取资源列表
	// support 1. 支持按照 principal-id + principal-role 进行查询
	// support 2. 支持普通的鉴权策略查询
	GetStrategies(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetStrategy 获取策略详细
	GetStrategy(ctx context.Context, strategy *api.AuthStrategy) *api.Response

	// GetPrincipalResources 获取某个 principal 的所有可操作资源列表
	GetPrincipalResources(ctx context.Context, query map[string]string) *api.Response
}
