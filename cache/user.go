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
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
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

	// GetAdmin 获取管理员信息
	GetAdmin() *model.User

	// GetUserByID
	//  @param id
	//  @return *model.User
	GetUserByID(id string) *model.User

	// GetUserByName
	//  @param name
	//  @return *model.User
	GetUserByName(name, ownerName string) *model.User

	// GetUserGroup
	//  @param id
	//  @return *model.UserGroupDetail
	GetGroup(id string) *model.UserGroupDetail

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
	GetUserLinkGroupIds(id string) []string
}

type userAndGroupCacheRefreshResult struct {
	userAdd    int
	userUpdate int
	userDel    int

	groupAdd    int
	groupUpdate int
	groupDel    int
}

// userCache 用户信息缓存
type userCache struct {
	*basCache

	storage store.Store

	adminUser                atomic.Value
	users                    *sync.Map // userid -> user
	name2Users               *sync.Map // username -> user
	groups                   *sync.Map // groupid -> group
	user2Groups              *sync.Map // userid -> groups
	userCacheFirstUpdate     bool
	groupCacheFirstUpdate    bool
	lastUserCacheUpdateTime  int64
	lastGroupCacheUpdateTime int64

	notifyCh chan interface{}

	singleFlight *singleflight.Group
}

// newUserCache
func newUserCache(storage store.Store, notifyCh chan interface{}) UserCache {
	return &userCache{
		basCache: newBaseCache(),
		storage:  storage,
		notifyCh: notifyCh,
	}
}

// initialize
func (uc *userCache) initialize(c map[string]interface{}) error {
	uc.users = new(sync.Map)
	uc.groups = new(sync.Map)
	uc.name2Users = new(sync.Map)
	uc.user2Groups = new(sync.Map)
	uc.adminUser = atomic.Value{}

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
		log.CacheScope().Errorf("[Cache][User] update user err: %s", err.Error())
		return err
	}

	grouplastMtime := time.Unix(uc.lastGroupCacheUpdateTime, 0)
	groups, err := uc.storage.GetGroupsForCache(grouplastMtime.Add(DefaultTimeDiff), uc.groupCacheFirstUpdate)
	if err != nil {
		log.CacheScope().Errorf("[Cache][Group] update group err: %s", err.Error())
		return err
	}

	uc.userCacheFirstUpdate = false
	uc.groupCacheFirstUpdate = false
	refreshRet := uc.setUserAndGroups(users, groups)
	log.CacheScope().Info("[Cache][User] get more user",
		zap.Int("add", refreshRet.userAdd),
		zap.Int("update", refreshRet.userUpdate),
		zap.Int("delete", refreshRet.userDel),
		zap.Time("last", userlastMtime), zap.Duration("used", time.Now().Sub(start)))

	log.CacheScope().Info("[Cache][Group] get more group",
		zap.Int("add", refreshRet.groupAdd),
		zap.Int("update", refreshRet.groupUpdate),
		zap.Int("delete", refreshRet.groupDel),
		zap.Time("last", grouplastMtime), zap.Duration("used", time.Now().Sub(start)))
	return nil
}

func (uc *userCache) setUserAndGroups(users []*model.User,
	groups []*model.UserGroupDetail) userAndGroupCacheRefreshResult {
	ret := userAndGroupCacheRefreshResult{}

	// 更新 users 缓存
	// step 1. 先更新 owner 用户
	uc.handlerUserCacheUpdate(&ret, users, func(user *model.User) bool {
		return (user.ID == user.Owner || user.Owner == "")
	}, func(user *model.User) *model.User {
		return user
	})

	// step 2. 更新非 owner 用户
	uc.handlerUserCacheUpdate(&ret, users, func(user *model.User) bool {
		return (user.Owner != "")
	}, func(user *model.User) *model.User {
		owner, _ := uc.users.Load(user.Owner)
		return owner.(*model.User)
	})

	uc.handlerGroupCacheUpdate(&ret, groups)

	uc.postProcess(users, groups)

	return ret
}

// handlerUserCacheUpdate 处理用户信息更新
func (uc *userCache) handlerUserCacheUpdate(ret *userAndGroupCacheRefreshResult, users []*model.User,
	filter func(user *model.User) bool,
	ownerSupplier func(user *model.User) *model.User) {

	for i := range users {
		user := users[i]
		if user.Type == model.AdminUserRole {
			uc.adminUser.Store(user)
		}

		if !filter(user) {
			continue
		}
		owner := ownerSupplier(user)
		if !user.Valid {
			uc.users.Delete(user.ID)
			uc.name2Users.Delete(fmt.Sprintf(NameLinkOwnerTemp, owner.Name, user.Name))

			ret.userDel++
		} else {
			if _, ok := uc.users.Load(user.ID); ok {
				ret.userUpdate++
			} else {
				ret.userAdd++
			}

			uc.users.Store(user.ID, user)
			uc.name2Users.Store(fmt.Sprintf(NameLinkOwnerTemp, owner.Name, user.Name), user)

			uc.lastUserCacheUpdateTime = int64(math.Max(float64(user.ModifyTime.Unix()),
				float64(uc.lastUserCacheUpdateTime)))
		}
	}
}

// handlerGroupCacheUpdate 处理用户组信息更新
func (uc *userCache) handlerGroupCacheUpdate(ret *userAndGroupCacheRefreshResult,
	groups []*model.UserGroupDetail) {

	// 更新 groups 数据信息
	for i := range groups {
		group := groups[i]
		if !group.Valid {
			uc.groups.Delete(group.ID)
			ret.groupDel++
		} else {
			var oldGroup *model.UserGroupDetail
			if oldVal, ok := uc.groups.Load(group.ID); ok {
				ret.groupUpdate++
				oldGroup = oldVal.(*model.UserGroupDetail)
			} else {
				ret.groupAdd++
			}
			uc.groups.Store(group.ID, group)

			if oldGroup != nil {
				oldUserIds := oldGroup.UserIds
				delUserIds := make([]string, 0, 4)
				for oldUserId := range oldUserIds {
					if _, ok := group.UserIds[oldUserId]; !ok {
						delUserIds = append(delUserIds, oldUserId)
					}
				}

				for di := range delUserIds {
					waitDel := delUserIds[di]
					if oldGids, ok := uc.user2Groups.Load(waitDel); ok {
						oldGids.(*sync.Map).Delete(group.ID)
					}
				}
			}

			for uid := range group.UserIds {
				uc.user2Groups.LoadOrStore(uid, new(sync.Map))
				val, _ := uc.user2Groups.Load(uid)
				gids := val.(*sync.Map)
				gids.Store(group.ID, struct{}{})
			}

			uc.lastGroupCacheUpdateTime = int64(math.Max(float64(group.ModifyTime.Unix()),
				float64(uc.lastGroupCacheUpdateTime)))
		}
	}
}

func (uc *userCache) clear() error {
	uc.users = new(sync.Map)
	uc.groups = new(sync.Map)
	uc.name2Users = new(sync.Map)
	uc.user2Groups = new(sync.Map)
	uc.adminUser = atomic.Value{}

	uc.userCacheFirstUpdate = false
	uc.groupCacheFirstUpdate = false
	uc.lastUserCacheUpdateTime = 0
	uc.lastGroupCacheUpdateTime = 0
	return nil
}

func (uc *userCache) name() string {
	return UsersName
}

// GetAdmin 获取管理员数据信息
func (uc *userCache) GetAdmin() *model.User {
	val := uc.adminUser.Load()
	if val == nil {
		return nil
	}

	return val.(*model.User)
}

// IsOwner 判断当前用户是否是 owner 角色
func (uc *userCache) IsOwner(id string) bool {
	val, ok := uc.users.Load(id)
	if !ok {
		return false
	}
	ut := val.(*model.User).Type
	return ut == model.AdminUserRole || ut == model.OwnerUserRole
}

func (uc *userCache) IsUserInGroup(userId, groupId string) bool {
	group := uc.GetGroup(groupId)
	if group == nil {
		return false
	}

	_, exist := group.UserIds[userId]
	return exist
}

// GetUserByID 根据用户ID获取用户缓存对象
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

// GetUserByName 通过用户 name 以及 owner 获取用户缓存对象
func (uc *userCache) GetUserByName(name, ownerName string) *model.User {
	val, ok := uc.name2Users.Load(fmt.Sprintf(NameLinkOwnerTemp, ownerName, name))

	if !ok {
		return nil
	}

	return val.(*model.User)
}

// GetGroup 通过用户组ID获取用户组缓存对象
func (uc *userCache) GetGroup(id string) *model.UserGroupDetail {
	if id == "" {
		return nil
	}
	val, ok := uc.groups.Load(id)

	if !ok {
		return nil
	}

	return val.(*model.UserGroupDetail)
}

// GetUserLinkGroupIds 根据用户ID查询该用户关联的用户组ID列表
func (uc *userCache) GetUserLinkGroupIds(userId string) []string {
	if userId == "" {
		return nil
	}
	val, ok := uc.user2Groups.Load(userId)

	if !ok {
		return nil
	}

	ret := make([]string, 0, 4)
	val.(*sync.Map).Range(func(key, value interface{}) bool {
		ret = append(ret, key.(string))
		return true
	})

	return ret
}

func (uc *userCache) postProcess(users []*model.User, groups []*model.UserGroupDetail) {
	userRemoves := make([]*model.User, 0, 8)
	groupRemoves := make([]*model.UserGroup, 0, 8)

	for index := range users {
		user := users[index]
		if !user.Valid {
			userRemoves = append(userRemoves, user)
		}
	}

	for index := range groups {
		group := groups[index]
		if !group.Valid {
			groupRemoves = append(groupRemoves, group.UserGroup)
		}
	}

	uc.onRemove(userRemoves, groupRemoves)
}

// onRemove 通知 listner 出现了批量的用户、用户组移除事件
func (uc *userCache) onRemove(users []*model.User, groups []*model.UserGroup) {
	principals := make([]model.Principal, 0, len(users)+len(groups))

	for index := range users {
		user := users[index]

		principals = append(principals, model.Principal{
			PrincipalID:   user.ID,
			PrincipalRole: model.PrincipalUser,
		})
	}

	for index := range groups {
		group := groups[index]

		principals = append(principals, model.Principal{
			PrincipalID:   group.ID,
			PrincipalRole: model.PrincipalGroup,
		})
	}

	uc.notifyCh <- principals
}
