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
	"fmt"

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/store"
)

// AuthChecker 权限管理通用接口定义
type AuthChecker interface {
	// CheckClientPermission 执行检查客户端动作判断是否有权限，并且对 RequestContext 注入操作者数据
	CheckClientPermission(preCtx *authcommon.AcquireContext) (bool, error)
	// CheckConsolePermission 执行检查控制台动作判断是否有权限，并且对 RequestContext 注入操作者数据
	CheckConsolePermission(preCtx *authcommon.AcquireContext) (bool, error)
	// IsOpenConsoleAuth 返回是否开启了操作鉴权，可以用于前端查询
	IsOpenConsoleAuth() bool
	// IsOpenClientAuth
	IsOpenClientAuth() bool
	// ResourcePredicate 是否允许资源的操作
	ResourcePredicate(ctx *authcommon.AcquireContext, opInfo *authcommon.ResourceEntry) bool
}

// StrategyServer 策略相关操作
type StrategyServer interface {
	// Initialize 执行初始化动作
	Initialize(*Config, store.Store, cachetypes.CacheManager, UserServer) error
	// Name 策略管理server名称
	Name() string
	// PolicyOperator .
	PolicyOperator
	// RoleOperator .
	RoleOperator
	// PolicyHelper .
	PolicyHelper() PolicyHelper
	// GetAuthChecker 获取鉴权检查器
	GetAuthChecker() AuthChecker
	// AfterResourceOperation 操作完资源的后置处理逻辑
	AfterResourceOperation(afterCtx *authcommon.AcquireContext) error
}

// PolicyOperator 策略管理
type PolicyOperator interface {
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
}

// RoleOperator 角色管理
type RoleOperator interface {
	// CreateRoles 批量创建角色
	CreateRoles(ctx context.Context, reqs []*apisecurity.Role) *apiservice.BatchWriteResponse
	// UpdateRoles 批量更新角色
	UpdateRoles(ctx context.Context, reqs []*apisecurity.Role) *apiservice.BatchWriteResponse
	// DeleteRoles 批量删除角色
	DeleteRoles(ctx context.Context, reqs []*apisecurity.Role) *apiservice.BatchWriteResponse
	// GetRoles 查询角色列表
	GetRoles(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
}

// UserServer 用户数据管理 server
type UserServer interface {
	// Initialize 初始化
	Initialize(*Config, store.Store, StrategyServer, cachetypes.CacheManager) error
	// Name 用户数据管理server名称
	Name() string
	// Login 登录动作
	Login(req *apisecurity.LoginRequest) *apiservice.Response
	// CheckCredential 检查当前操作用户凭证
	CheckCredential(authCtx *authcommon.AcquireContext) error
	// UserOperator
	UserOperator
	// GroupOperator
	GroupOperator
	// GetUserHelper
	GetUserHelper() UserHelper
}

type UserOperator interface {
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
	// EnableUserToken 禁止用户的token使用
	EnableUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response
	// ResetUserToken 重置用户的token
	ResetUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response
}

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
	// EnableGroupToken 取消用户组的 token 使用
	EnableGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response
	// ResetGroupToken 重置用户组的 token
	ResetGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response
}

type UserHelper interface {
	// CheckUserInGroup 检查用户是否在用户组中
	CheckUserInGroup(ctx context.Context, group *apisecurity.UserGroup, user *apisecurity.User) bool
	// CheckGroupsExist 批量检查用户组是否存在
	CheckGroupsExist(ctx context.Context, groups []*apisecurity.UserGroup) error
	// CheckUsersExist 批量检查用户是否存在
	CheckUsersExist(ctx context.Context, users []*apisecurity.User) error
	// GetUserOwnGroup 查询某个用户所在的所有用户组
	GetUserOwnGroup(ctx context.Context, user *apisecurity.User) []*apisecurity.UserGroup
	// GetUser 查询用户信息
	GetUser(ctx context.Context, user *apisecurity.User) *apisecurity.User
	// GetUserByID 查询用户信息
	GetUserByID(ctx context.Context, id string) *apisecurity.User
	// GetGroup 查询用户组信息
	GetGroup(ctx context.Context, req *apisecurity.UserGroup) *apisecurity.UserGroup
}

// PolicyHelper .
type PolicyHelper interface {
	// CreatePrincipal 创建 principal 的默认 policy 资源
	CreatePrincipal(ctx context.Context, tx store.Tx, p authcommon.Principal) error
	// CleanPrincipal 清理 principal 所关联的 policy、role 资源
	CleanPrincipal(ctx context.Context, tx store.Tx, p authcommon.Principal) error
}

// OperatorInfo 根据 token 解析出来的具体额外信息
type OperatorInfo struct {
	// Origin 原始 token 字符串
	Origin string
	// OperatorID 当前 token 绑定的 用户/用户组 ID
	OperatorID string
	// OwnerID 当前用户/用户组对应的 owner
	OwnerID string
	// Role 如果当前是 user token 的话，该值才能有信息
	Role authcommon.UserRoleType
	// IsUserToken 当前 token 是否是 user 的 token
	IsUserToken bool
	// Disable 标识用户 token 是否被禁用
	Disable bool
	// 是否属于匿名操作者
	Anonymous bool
}

func NewAnonymous() OperatorInfo {
	return OperatorInfo{
		Origin:     "",
		OwnerID:    "",
		OperatorID: "__anonymous__",
		Anonymous:  true,
	}
}

// IsEmptyOperator token 是否是一个空类型
func IsEmptyOperator(t OperatorInfo) bool {
	return t.Origin == "" || t.Anonymous
}

// IsSubAccount 当前 token 对应的账户类型
func IsSubAccount(t OperatorInfo) bool {
	return t.Role == authcommon.SubAccountUserRole
}

func (t *OperatorInfo) String() string {
	return fmt.Sprintf("operator-id=%s, owner=%s, role=%d, is-user=%v, disable=%v",
		t.OperatorID, t.OwnerID, t.Role, t.IsUserToken, t.Disable)
}
