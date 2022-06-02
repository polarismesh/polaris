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

	"github.com/polarismesh/polaris-server/common/model"
)

// UserStore User-related operation interface
type UserStore interface {

	// AddUser Create a user
	AddUser(user *model.User) error

	// UpdateUser Update user
	UpdateUser(user *model.User) error

	// DeleteUser delete users
	DeleteUser(user *model.User) error

	// GetSubCount Number of getting a child account
	GetSubCount(user *model.User) (uint32, error)

	// GetUser Obtain user
	GetUser(id string) (*model.User, error)

	// GetUserByName Get a unique user according to Name + Owner
	GetUserByName(name, ownerId string) (*model.User, error)

	// GetUserByIDS Get users according to USER IDS batch
	GetUserByIds(ids []string) ([]*model.User, error)

	// GetUsers Query user list
	GetUsers(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.User, error)

	// GetUsersForCache Used to refresh user cache
	// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
	GetUsersForCache(mtime time.Time, firstUpdate bool) ([]*model.User, error)
}

// GroupStore User group storage operation interface
type GroupStore interface {

	// AddGroup Add a user group
	AddGroup(group *model.UserGroupDetail) error

	// UpdateGroup Update user group
	UpdateGroup(group *model.ModifyUserGroup) error

	// DeleteGroup Delete user group
	DeleteGroup(group *model.UserGroupDetail) error

	// GetGroup Get user group details
	GetGroup(id string) (*model.UserGroupDetail, error)

	// GetGroupByName Get user groups according to Name and Owner
	GetGroupByName(name, owner string) (*model.UserGroup, error)

	// GetGroups Get a list of user groups
	GetGroups(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.UserGroup, error)

	// GetUserGroupsForCache Refresh of getting user groups for cache
	// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
	GetGroupsForCache(mtime time.Time, firstUpdate bool) ([]*model.UserGroupDetail, error)
}

// StrategyStore Authentication policy related storage operation interface
type StrategyStore interface {

	// AddStrategy Create authentication strategy
	AddStrategy(strategy *model.StrategyDetail) error

	// UpdateStrategy Update authentication strategy
	UpdateStrategy(strategy *model.ModifyStrategyDetail) error

	// DeleteStrategy Delete authentication strategy
	DeleteStrategy(id string) error

	// LooseAddStrategyResources Song requires the resources of the authentication strategy,
	//   allowing the issue of ignoring the primary key conflict
	LooseAddStrategyResources(resources []model.StrategyResource) error

	// RemoveStrategyResources Clean all the strategies associated with corresponding resources
	RemoveStrategyResources(resources []model.StrategyResource) error

	// GetStrategyResources Gets a Principal's corresponding resource ID data information
	GetStrategyResources(principalId string, principalRole model.PrincipalType) ([]model.StrategyResource, error)

	// GetDefaultStrategyDetailByPrincipal Get a default policy for a Principal
	GetDefaultStrategyDetailByPrincipal(principalId string, principalType model.PrincipalType) (*model.StrategyDetail, error)

	// GetStrategyDetail Get strategy details
	GetStrategyDetail(id string) (*model.StrategyDetail, error)

	// GetStrategies Get a list of strategies
	GetStrategies(filters map[string]string, offset uint32, limit uint32) (uint32,
		[]*model.StrategyDetail, error)

	// GetStrategyDetailsForCache Used to refresh policy cache
	// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
	GetStrategyDetailsForCache(mtime time.Time, firstUpdate bool) ([]*model.StrategyDetail, error)
}
