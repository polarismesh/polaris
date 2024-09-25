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

package store

import (
	"time"

	authcommon "github.com/polarismesh/polaris/common/model/auth"
)

type AuthStore interface {
	// UserStore 用户接口
	UserStore
	// GroupStore 用户组接口
	GroupStore
	// StrategyStore 鉴权策略接口
	StrategyStore
	// RoleStore 角色接口
	RoleStore
}

// UserStore User-related operation interface
type UserStore interface {
	// AddUser Create a user
	AddUser(tx Tx, user *authcommon.User) error
	// UpdateUser Update user
	UpdateUser(user *authcommon.User) error
	// DeleteUser delete users
	DeleteUser(tx Tx, user *authcommon.User) error
	// GetSubCount Number of getting a child account
	GetSubCount(user *authcommon.User) (uint32, error)
	// GetUser Obtain user
	GetUser(id string) (*authcommon.User, error)
	// GetUserByName Get a unique user according to Name + Owner
	GetUserByName(name, ownerId string) (*authcommon.User, error)
	// GetUserByIDS Get users according to USER IDS batch
	GetUserByIds(ids []string) ([]*authcommon.User, error)
	// GetUsers Query user list
	GetUsers(filters map[string]string, offset uint32, limit uint32) (uint32, []*authcommon.User, error)
	// GetUsersForCache Used to refresh user cache
	// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
	GetUsersForCache(mtime time.Time, firstUpdate bool) ([]*authcommon.User, error)
}

// GroupStore User group storage operation interface
type GroupStore interface {
	// AddGroup Add a user group
	AddGroup(tx Tx, group *authcommon.UserGroupDetail) error
	// UpdateGroup Update user group
	UpdateGroup(group *authcommon.ModifyUserGroup) error
	// DeleteGroup Delete user group
	DeleteGroup(tx Tx, group *authcommon.UserGroupDetail) error
	// GetGroup Get user group details
	GetGroup(id string) (*authcommon.UserGroupDetail, error)
	// GetGroupByName Get user groups according to Name and Owner
	GetGroupByName(name, owner string) (*authcommon.UserGroup, error)
	// GetGroups Get a list of user groups
	GetGroups(filters map[string]string, offset uint32, limit uint32) (uint32, []*authcommon.UserGroup, error)
	// GetUserGroupsForCache Refresh of getting user groups for cache
	// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
	GetGroupsForCache(mtime time.Time, firstUpdate bool) ([]*authcommon.UserGroupDetail, error)
}

// StrategyStore Authentication policy related storage operation interface
type StrategyStore interface {
	// AddStrategy Create authentication strategy
	AddStrategy(tx Tx, strategy *authcommon.StrategyDetail) error
	// UpdateStrategy Update authentication strategy
	UpdateStrategy(strategy *authcommon.ModifyStrategyDetail) error
	// DeleteStrategy Delete authentication strategy
	DeleteStrategy(id string) error
	// CleanPrincipalPolicies Clean all the policies associated with the principal
	CleanPrincipalPolicies(tx Tx, p authcommon.Principal) error
	// LooseAddStrategyResources Song requires the resources of the authentication strategy,
	//   allowing the issue of ignoring the primary key conflict
	LooseAddStrategyResources(resources []authcommon.StrategyResource) error
	// RemoveStrategyResources Clean all the strategies associated with corresponding resources
	RemoveStrategyResources(resources []authcommon.StrategyResource) error
	// GetStrategyResources Gets a Principal's corresponding resource ID data information
	GetStrategyResources(principalId string, principalRole authcommon.PrincipalType) ([]authcommon.StrategyResource, error)
	// GetDefaultStrategyDetailByPrincipal Get a default policy for a Principal
	GetDefaultStrategyDetailByPrincipal(principalId string,
		principalType authcommon.PrincipalType) (*authcommon.StrategyDetail, error)
	// GetStrategyDetail Get strategy details
	GetStrategyDetail(id string) (*authcommon.StrategyDetail, error)
	// GetStrategies Get a list of strategies
	GetStrategies(filters map[string]string, offset uint32, limit uint32) (uint32,
		[]*authcommon.StrategyDetail, error)
	// GetMoreStrategies Used to refresh policy cache
	// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
	GetMoreStrategies(mtime time.Time, firstUpdate bool) ([]*authcommon.StrategyDetail, error)
}

// RoleStore Role related storage operation interface
type RoleStore interface {
	// AddRole Add a role
	AddRole(role *authcommon.Role) error
	// UpdateRole Update a role
	UpdateRole(role *authcommon.Role) error
	// DeleteRole Delete a role
	DeleteRole(role *authcommon.Role) error
	// CleanPrincipalRoles Clean all the roles associated with the principal
	CleanPrincipalRoles(tx Tx, p *authcommon.Principal) error
	// GetRole get more role for cache update
	GetMoreRoles(firstUpdate bool, modifyTime time.Time) ([]*authcommon.Role, error)
}
