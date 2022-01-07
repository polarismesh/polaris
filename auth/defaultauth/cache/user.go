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
	"sync"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

// UserCache
type UserCache interface {
	Cache

	// GetUser
	//  @param id
	//  @return *model.User
	GetUser(id string) *model.User

	// GetUserByName
	//  @param name
	//  @return *model.User
	GetUserByName(name string) *model.User

	// GetUserGroup
	//  @param id
	//  @return *model.UserGroupDetail
	GetUserGroup(id string) *model.UserGroupDetail

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
func (uc *userCache) initialize() error {
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
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := uc.singleFlight.Do(CacheForUser, func() (interface{}, error) {
		return nil, uc.realUpdate()
	})
	return err
}

func (uc *userCache) realUpdate() error {
	// 获取几秒前的全部数据
	start := time.Now()
	userlastMtime := time.Unix(uc.lastUserCacheUpdateTime, 0)
	users, err := uc.storage.GetUsersForCache(userlastMtime.Add(DefaultTimeDiff), uc.userCacheFirstUpdate)
	if err != nil {
		log.Errorf("[Cache][User] update user err: %s", err.Error())
		return err
	}

	grouplastMtime := time.Unix(uc.lastGroupCacheUpdateTime, 0)
	groups, err := uc.storage.GetUserGroupsForCache(grouplastMtime, uc.groupCacheFirstUpdate)
	if err != nil {
		log.Errorf("[Cache][User] update group err: %s", err.Error())
		return err
	}

	uc.userCacheFirstUpdate = false
	uc.groupCacheFirstUpdate = false
	update, del := uc.setUserAndGroups(users, groups)
	log.Debug("[Cache][User] get more services", zap.Int("update", update), zap.Int("delete", del),
		zap.Time("userLast", userlastMtime), zap.Time("groupLast", grouplastMtime), zap.Duration("used", time.Now().Sub(start)))
	return nil
}

func (uc *userCache) setUserAndGroups(users []*model.User, groups []*model.UserGroupDetail) (int, int) {
	// 更新 users 缓存
	for i := range users {
		user := users[i]
		uc.users.Store(user.ID, user)
		uc.name2Users.Store(user.Name, user)
	}

	// 更新 groups 数据信息
	for i := range groups {
		group := groups[i]

		uc.groups.Store(group.ID, group)
		uc.name2Groups.Store(group.Name, group)

		for j := range group.UserIDs {
			uid := group.UserIDs[j]
			uc.user2Groups.LoadOrStore(uid, make([]string, 0))

			val, _ := uc.user2Groups.Load(uid)
			gids := val.([]string)

			uc.user2Groups.Store(uid, append(gids, group.ID))
		}
	}

	return 0, 0
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
	return CacheForUser
}

func (uc *userCache) IsOwner(id string) bool {
	val, ok := uc.users.Load(id)

	if !ok {
		return false
	}
	return val.(*model.User).Owner == ""
}

func (uc *userCache) GetUser(id string) *model.User {
	val, ok := uc.users.Load(id)

	if !ok {
		return nil
	}

	return val.(*model.User)
}

func (uc *userCache) GetUserByName(name string) *model.User {
	val, ok := uc.name2Users.Load(name)

	if !ok {
		return nil
	}

	return val.(*model.User)
}

func (uc *userCache) GetUserGroup(id string) *model.UserGroupDetail {
	val, ok := uc.groups.Load(id)

	if !ok {
		return nil
	}

	return val.(*model.UserGroupDetail)
}

func (uc *userCache) ListUserBelongGroupIDS(id string) []string {
	return nil
}
