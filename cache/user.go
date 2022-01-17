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

package cache

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

func init() {
	RegisterCache(UsersName, CacheUser)
}

const (
	UsersName string = "users"

	NameLinkOwnerTemp string = "%s@%s"
)

// UserCache User information cache
type UserCache interface {
	Cache

	// GetUserByID
	//  @param id
	//  @return *model.User
	GetUserByID(id string) *model.User

	// GetUserByName
	//  @param name
	//  @return *model.User
	GetUserByName(name, owner string) *model.User

	// GetUserGroup
	//  @param id
	//  @return *model.UserGroupDetail
	GetUserGroup(id string) *model.UserGroupDetail

	// IsUserInGroup 判断 userid 是否在对应的 group 中
	//  @param userId
	//  @param groupId
	//  @return bool
	IsUserInGroup(userId, groupId string) bool

	// IsOwner
	//  @param id
	//  @return bool
	IsOwner(id string) bool

	// ListUserBelongGroupIDS
	//  @param id
	//  @return []string
	ListUserBelongGroupIDS(id string) []string
}

// userCache 用户信息缓存
type userCache struct {
	storage                  store.Store
	users                    *sync.Map // userid -> user
	name2Users               *sync.Map // username -> user
	groups                   *sync.Map // groupid -> group
	name2Groups              *sync.Map // groupname -> group
	user2Groups              *sync.Map // userid -> groups
	userCacheFirstUpdate     bool
	groupCacheFirstUpdate    bool
	lastUserCacheUpdateTime  int64
	lastGroupCacheUpdateTime int64

	singleFlight *singleflight.Group
}

// newUserCache
//  @param storage
//  @return UserCache
func newUserCache(storage store.Store) UserCache {
	return &userCache{
		storage: storage,
	}
}

// initialize
//  @receiver uc
//  @return error
func (uc *userCache) initialize(c map[string]interface{}) error {
	uc.users = new(sync.Map)
	uc.groups = new(sync.Map)
	uc.name2Users = new(sync.Map)
	uc.name2Groups = new(sync.Map)
	uc.user2Groups = new(sync.Map)

	uc.userCacheFirstUpdate = true
	uc.groupCacheFirstUpdate = true
	uc.lastUserCacheUpdateTime = 0
	uc.lastGroupCacheUpdateTime = 0

	uc.singleFlight = new(singleflight.Group)
	return nil
}

func (uc *userCache) update() error {
	// Multiple threads competition, only one thread is updated
	_, err, _ := uc.singleFlight.Do(UsersName, func() (interface{}, error) {
		return nil, uc.realUpdate()
	})
	return err
}

func (uc *userCache) realUpdate() error {
	// Get all data before a few seconds
	start := time.Now()
	userlastMtime := time.Unix(uc.lastUserCacheUpdateTime, 0)
	users, err := uc.storage.GetUsersForCache(userlastMtime.Add(DefaultTimeDiff), uc.userCacheFirstUpdate)
	if err != nil {
		log.GetCacheLogger().Errorf("[Cache][User] update user err: %s", err.Error())
		return err
	}

	grouplastMtime := time.Unix(uc.lastGroupCacheUpdateTime, 0)
	groups, err := uc.storage.GetGroupsForCache(grouplastMtime.Add(DefaultTimeDiff), uc.groupCacheFirstUpdate)
	if err != nil {
		log.GetCacheLogger().Errorf("[Cache][Group] update group err: %s", err.Error())
		return err
	}

	uc.userCacheFirstUpdate = false
	uc.groupCacheFirstUpdate = false
	update, del := uc.setUserAndGroups(users, groups)
	log.GetCacheLogger().Debug("[Cache][User] get more user", zap.Int("update", update), zap.Int("delete", del),
		zap.Time("userLast", userlastMtime), zap.Time("groupLast", grouplastMtime), zap.Duration("used", time.Now().Sub(start)))
	return nil
}

func (uc *userCache) setUserAndGroups(users []*model.User, groups []*model.UserGroupDetail) (int, int) {

	// 更新 users 缓存
	// step 1. 先更新 owner 用户
	uc.handlerUserCacheUpdate(users, func(user *model.User) bool {
		return (user.ID == user.Owner || user.Owner == "")
	}, func(user *model.User) *model.User {
		return user
	})

	// step 2. 更新非 owner 用户
	uc.handlerUserCacheUpdate(users, func(user *model.User) bool {
		return (user.Owner != "")
	}, func(user *model.User) *model.User {
		owner, _ := uc.users.Load(user.Owner)
		return owner.(*model.User)
	})

	// 更新 groups 数据信息
	for i := range groups {
		group := groups[i]
		owner, _ := uc.users.Load(group.Owner)
		if !group.Valid {
			uc.groups.Delete(group.ID)
			uc.name2Groups.Delete(fmt.Sprintf(NameLinkOwnerTemp, owner.(*model.User).Name, group.Name))
		} else {
			uc.groups.Store(group.ID, group)
			uc.name2Groups.Store(fmt.Sprintf(NameLinkOwnerTemp, owner.(*model.User).Name, group.Name), group)

			for j := range group.UserIDs {
				uid := group.UserIDs[j]
				uc.user2Groups.LoadOrStore(uid, make([]string, 0))

				val, _ := uc.user2Groups.Load(uid)
				gids := val.([]string)

				uc.user2Groups.Store(uid, append(gids, group.ID))
			}

			uc.lastGroupCacheUpdateTime = int64(math.Max(float64(group.ModifyTime.Unix()), float64(uc.lastGroupCacheUpdateTime)))
		}
	}

	return 0, 0
}

// handlerUserCacheUpdate 处理用户信息更新
func (uc *userCache) handlerUserCacheUpdate(users []*model.User, filter func(user *model.User) bool, ownerSupplier func(user *model.User) *model.User) {
	for i := range users {
		user := users[i]
		if !filter(user) {
			continue
		}
		owner := ownerSupplier(user)
		if !user.Valid {
			uc.users.Delete(user.ID)
			uc.name2Users.Delete(fmt.Sprintf(NameLinkOwnerTemp, owner.Name, user.Name))
		} else {
			uc.users.Store(user.ID, user)
			uc.name2Users.Store(fmt.Sprintf(NameLinkOwnerTemp, owner.Name, user.Name), user)

			uc.lastUserCacheUpdateTime = int64(math.Max(float64(user.ModifyTime.Unix()), float64(uc.lastUserCacheUpdateTime)))
		}
	}
}

func (uc *userCache) clear() error {
	uc.users = new(sync.Map)
	uc.groups = new(sync.Map)
	uc.name2Users = new(sync.Map)
	uc.name2Groups = new(sync.Map)
	uc.user2Groups = new(sync.Map)

	uc.userCacheFirstUpdate = false
	uc.groupCacheFirstUpdate = false
	uc.lastUserCacheUpdateTime = 0
	uc.lastGroupCacheUpdateTime = 0
	return nil
}

func (uc *userCache) name() string {
	return UsersName
}

func (uc *userCache) IsOwner(id string) bool {
	val, ok := uc.users.Load(id)
	if !ok {
		return false
	}
	ut := val.(*model.User).Type
	return ut == model.AdminUserRole || ut == model.OwnerUserRole
}

func (uc *userCache) IsUserInGroup(userId, groupId string) bool {
	group := uc.GetUserGroup(groupId)
	if group == nil {
		return false
	}

	_, exist := group.UserIDs[userId]
	return exist
}

func (uc *userCache) GetUserByID(id string) *model.User {
	if id == "" {
		return nil
	}

	val, ok := uc.users.Load(id)

	if !ok {
		return nil
	}

	return val.(*model.User)
}

func (uc *userCache) GetUserByName(name, owner string) *model.User {
	val, ok := uc.name2Users.Load(fmt.Sprintf(NameLinkOwnerTemp, owner, name))

	if !ok {
		return nil
	}

	return val.(*model.User)
}

func (uc *userCache) GetUserGroup(id string) *model.UserGroupDetail {
	if id == "" {
		return nil
	}
	val, ok := uc.groups.Load(id)

	if !ok {
		return nil
	}

	return val.(*model.UserGroupDetail)
}

func (uc *userCache) ListUserBelongGroupIDS(id string) []string {
	return nil
}
