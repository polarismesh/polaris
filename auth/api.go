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

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

// AuthChecker 权限管理通用接口定义
type AuthChecker interface {
	// Initialize 执行初始化动作
	Initialize(options *Config, storage store.Store, cacheMgn cachetypes.CacheManager) error
	// VerifyCredential 验证令牌
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

// UserServer 用户数据管理 server
type UserServer interface {
	// Initialize 初始化
	Initialize(authOpt *Config, storage store.Store, cacheMgn cachetypes.CacheManager) error
	// Name 用户数据管理server名称
	Name() string
	// CreateUsers 批量创建用户
	CreateUsers(ctx context.Context, users []*apisecurity.User) *apiservice.BatchWriteResponse
	// UpdateUser 更新用户信息
	UpdateUser(ctx context.Context, user *apisecurity.User) *apiservice.Response
	// UpdateUserPassword 更新用户密码
	UpdateUserPassword(ctx context.Context, req *apisecurity.ModifyUserPassword) *apiservice.Response
	// DeleteUsers 批量删除用户
	DeleteUsers(ctx context.Context, users []*apisecurity.User) *apiservice.BatchWriteResponse
	// GetUsers 查询用户列表
	GetUsers(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
	// GetUserToken 获取用户的 token
	GetUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response
	// UpdateUserToken 禁止用户的token使用
	UpdateUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response
	// ResetUserToken 重置用户的token
	ResetUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response
	// Login 登录动作
	Login(req *apisecurity.LoginRequest) *apiservice.Response
	GroupOperator
}

// GroupOperator 用户组相关操作
type GroupOperator interface {
	// CreateGroup 创建用户组
	CreateGroup(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response
	// UpdateGroups 更新用户组
	UpdateGroups(ctx context.Context, groups []*apisecurity.ModifyUserGroup) *apiservice.BatchWriteResponse
	// DeleteGroups 批量删除用户组
	DeleteGroups(ctx context.Context, group []*apisecurity.UserGroup) *apiservice.BatchWriteResponse
	// GetGroups 查询用户组列表（不带用户详细信息）
	GetGroups(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
	// GetGroup 根据用户组信息，查询该用户组下的用户相信
	GetGroup(ctx context.Context, req *apisecurity.UserGroup) *apiservice.Response
	// GetGroupToken 获取用户组的 token
	GetGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response
	// UpdateGroupToken 取消用户组的 token 使用
	UpdateGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response
	// ResetGroupToken 重置用户组的 token
	ResetGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response
}

// StrategyServer 策略相关操作
type StrategyServer interface {
	// Initialize 初始化
	Initialize(authOpt *Config, storage store.Store, cacheMgn cachetypes.CacheManager) error
	// Name 策略管理server名称
	Name() string
	// CreateStrategy 创建策略
	CreateStrategy(ctx context.Context, strategy *apisecurity.AuthStrategy) *apiservice.Response
	// UpdateStrategies 批量更新策略
	UpdateStrategies(ctx context.Context, reqs []*apisecurity.ModifyAuthStrategy) *apiservice.BatchWriteResponse
	// DeleteStrategies 删除策略
	DeleteStrategies(ctx context.Context, reqs []*apisecurity.AuthStrategy) *apiservice.BatchWriteResponse
	// GetStrategies 获取资源列表
	// support 1. 支持按照 principal-id + principal-role 进行查询
	// support 2. 支持普通的鉴权策略查询
	GetStrategies(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
	// GetStrategy 获取策略详细
	GetStrategy(ctx context.Context, strategy *apisecurity.AuthStrategy) *apiservice.Response
	// GetPrincipalResources 获取某个 principal 的所有可操作资源列表
	GetPrincipalResources(ctx context.Context, query map[string]string) *apiservice.Response
	// GetAuthChecker 获取鉴权检查器
	GetAuthChecker() AuthChecker
	// AfterResourceOperation 操作完资源的后置处理逻辑
	AfterResourceOperation(afterCtx *model.AcquireContext) error
}
