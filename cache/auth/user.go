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
	"math"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	types "github.com/polarismesh/polaris/cache/api"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const (
	NameLinkOwnerTemp = "%s@%s"
)

type userRefreshResult struct {
	userAdd    int
	userUpdate int
	userDel    int

	groupAdd    int
	groupUpdate int
	groupDel    int
}

// userCache 用户信息缓存
type userCache struct {
	*types.BaseCache

	storage store.Store

	adminUser atomic.Value
	// userid -> user
	users *utils.SyncMap[string, *authcommon.User]
	// username -> user
	name2Users *utils.SyncMap[string, *authcommon.User]
	// groupid -> group
	groups *utils.SyncMap[string, *authcommon.UserGroupDetail]
	// userid -> groups
	user2Groups *utils.SyncMap[string, *utils.SyncSet[string]]

	lastUserMtime  int64
	lastGroupMtime int64

	singleFlight *singleflight.Group
}

// NewUserCache
func NewUserCache(storage store.Store, cacheMgr types.CacheManager) types.UserCache {
	return &userCache{
		BaseCache: types.NewBaseCache(storage, cacheMgr),
		storage:   storage,
	}
}

// Initialize
func (uc *userCache) Initialize(_ map[string]interface{}) error {
	uc.users = utils.NewSyncMap[string, *authcommon.User]()
	uc.name2Users = utils.NewSyncMap[string, *authcommon.User]()
	uc.groups = utils.NewSyncMap[string, *authcommon.UserGroupDetail]()
	uc.user2Groups = utils.NewSyncMap[string, *utils.SyncSet[string]]()
	uc.adminUser = atomic.Value{}
	uc.singleFlight = new(singleflight.Group)
	return nil
}

func (uc *userCache) Update() error {
	// Multiple threads competition, only one thread is updated
	_, err, _ := uc.singleFlight.Do(uc.Name(), func() (interface{}, error) {
		return nil, uc.DoCacheUpdate(uc.Name(), uc.realUpdate)
	})
	return err
}

func (uc *userCache) realUpdate() (map[string]time.Time, int64, error) {
	// Get all data before a few seconds
	start := time.Now()
	users, err := uc.storage.GetUsersForCache(uc.LastFetchTime(), uc.IsFirstUpdate())
	if err != nil {
		log.Errorf("[Cache][User] update user err: %s", err.Error())
		return nil, -1, err
	}

	groups, err := uc.storage.GetGroupsForCache(uc.LastFetchTime(), uc.IsFirstUpdate())
	if err != nil {
		log.Errorf("[Cache][Group] update group err: %s", err.Error())
		return nil, -1, err
	}
	lastMimes, refreshRet := uc.setUserAndGroups(users, groups)

	log.Info("[Cache][User] get more user and user_group",
		zap.Int("user_add", refreshRet.userAdd), zap.Int("user_update", refreshRet.userUpdate),
		zap.Int("user_delete", refreshRet.userDel), zap.Time("user_modify_last", time.Unix(uc.lastUserMtime, 0)),
		zap.Int("group_add", refreshRet.groupAdd), zap.Int("group_update", refreshRet.groupUpdate),
		zap.Int("group_delete", refreshRet.groupDel), zap.Time("group_modify_last", time.Unix(uc.lastGroupMtime, 0)),
		zap.Duration("used", time.Since(start)))

	return lastMimes, int64(len(users) + len(groups)), nil
}

func (uc *userCache) setUserAndGroups(users []*authcommon.User,
	groups []*authcommon.UserGroupDetail) (map[string]time.Time, userRefreshResult) {
	ret := userRefreshResult{}

	ownerSupplier := func(user *authcommon.User) *authcommon.User {
		if user.Type == authcommon.SubAccountUserRole {
			owner, _ := uc.users.Load(user.Owner)
			return owner
		}
		return user
	}

	lastMimes := map[string]time.Time{}

	// 更新 users 缓存
	// step 1. 先更新 owner 用户
	uc.handlerUserCacheUpdate(lastMimes, &ret, users, func(user *authcommon.User) bool {
		return user.Type == authcommon.OwnerUserRole
	}, ownerSupplier)

	// step 2. 更新非 owner 用户
	uc.handlerUserCacheUpdate(lastMimes, &ret, users, func(user *authcommon.User) bool {
		return user.Type == authcommon.SubAccountUserRole
	}, ownerSupplier)

	uc.handlerGroupCacheUpdate(lastMimes, &ret, groups)
	return lastMimes, ret
}

// handlerUserCacheUpdate 处理用户信息更新
func (uc *userCache) handlerUserCacheUpdate(lastMimes map[string]time.Time, ret *userRefreshResult, users []*authcommon.User,
	filter func(user *authcommon.User) bool, ownerSupplier func(user *authcommon.User) *authcommon.User) {

	lastUserMtime := uc.LastMtime("users").Unix()

	for i := range users {
		user := users[i]

		lastUserMtime = int64(math.Max(float64(lastUserMtime), float64(user.ModifyTime.Unix())))

		if user.Type == authcommon.AdminUserRole {
			uc.adminUser.Store(user)
			uc.users.Store(user.ID, user)
			uc.name2Users.Store(fmt.Sprintf(NameLinkOwnerTemp, user.Name, user.Name), user)
			continue
		}

		if !filter(user) {
			continue
		}

		owner := ownerSupplier(user)
		if !user.Valid {
			// 删除 user-id -> user 的缓存
			// 删除 username + ownername -> user 的缓存
			// 删除 user-id -> group-ids 的缓存
			uc.users.Delete(user.ID)
			uc.name2Users.Delete(fmt.Sprintf(NameLinkOwnerTemp, owner.Name, user.Name))
			// uc.user2Groups.Delete(user.ID)
			ret.userDel++
		} else {
			if _, ok := uc.users.Load(user.ID); ok {
				ret.userUpdate++
			} else {
				ret.userAdd++
			}
			uc.users.Store(user.ID, user)
			uc.name2Users.Store(fmt.Sprintf(NameLinkOwnerTemp, owner.Name, user.Name), user)
		}
	}

	lastMimes["users"] = time.Unix(lastUserMtime, 0)
}

// handlerGroupCacheUpdate 处理用户组信息更新
func (uc *userCache) handlerGroupCacheUpdate(lastMimes map[string]time.Time, ret *userRefreshResult,
	groups []*authcommon.UserGroupDetail) {

	lastGroupMtime := uc.LastMtime("group").Unix()

	// 更新 groups 数据信息
	for i := range groups {
		group := groups[i]

		lastGroupMtime = int64(math.Max(float64(lastGroupMtime), float64(group.ModifyTime.Unix())))

		if !group.Valid {
			uc.groups.Delete(group.ID)
			ret.groupDel++
		} else {
			var oldGroup *authcommon.UserGroupDetail
			if oldVal, ok := uc.groups.Load(group.ID); ok {
				ret.groupUpdate++
				oldGroup = oldVal
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
						oldGids.Remove(group.ID)
					}
				}
			}

			for uid := range group.UserIds {
				_, exist := uc.user2Groups.Load(uid)
				if !exist {
					uc.user2Groups.Store(uid, utils.NewSyncSet[string]())
				}
				val, _ := uc.user2Groups.Load(uid)
				val.Add(group.ID)
			}
		}
	}

	lastMimes["group"] = time.Unix(lastGroupMtime, 0)
}

func (uc *userCache) Clear() error {
	uc.BaseCache.Clear()
	uc.users = utils.NewSyncMap[string, *authcommon.User]()
	uc.name2Users = utils.NewSyncMap[string, *authcommon.User]()
	uc.groups = utils.NewSyncMap[string, *authcommon.UserGroupDetail]()
	uc.user2Groups = utils.NewSyncMap[string, *utils.SyncSet[string]]()
	uc.adminUser = atomic.Value{}
	uc.lastUserMtime = 0
	uc.lastGroupMtime = 0
	return nil
}

func (uc *userCache) Name() string {
	return types.UsersName
}

// GetAdmin 获取管理员数据信息
func (uc *userCache) GetAdmin() *authcommon.User {
	val := uc.adminUser.Load()
	if val == nil {
		return nil
	}

	return val.(*authcommon.User)
}

// IsOwner 判断当前用户是否是 owner 角色
func (uc *userCache) IsOwner(id string) bool {
	val, ok := uc.users.Load(id)
	if !ok {
		return false
	}
	ut := val.Type
	return ut == authcommon.AdminUserRole || ut == authcommon.OwnerUserRole
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
func (uc *userCache) GetUserByID(id string) *authcommon.User {
	if id == "" {
		return nil
	}

	val, ok := uc.users.Load(id)
	if !ok {
		return nil
	}
	return val
}

// GetUserByName 通过用户 name 以及 owner 获取用户缓存对象
func (uc *userCache) GetUserByName(name, ownerName string) *authcommon.User {
	val, ok := uc.name2Users.Load(fmt.Sprintf(NameLinkOwnerTemp, ownerName, name))

	if !ok {
		return nil
	}
	return val
}

// GetGroup 通过用户组ID获取用户组缓存对象
func (uc *userCache) GetGroup(id string) *authcommon.UserGroupDetail {
	if id == "" {
		return nil
	}
	val, ok := uc.groups.Load(id)

	if !ok {
		return nil
	}

	return val
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
	return val.ToSlice()
}

// QueryUsers .
func (uc *userCache) QueryUsers(ctx context.Context, args types.UserSearchArgs) (uint32, []*authcommon.User, error) {
	searchId, hasId := args.Filters["id"]
	searchName, hasName := args.Filters["name"]
	searchOwner, hasOwner := args.Filters["owner"]
	searchSource, hasSource := args.Filters["source"]
	searchGroupId, hasGroup := args.Filters["group_id"]

	predicates := types.LoadUserPredicates(ctx)

	if hasGroup {
		g, ok := uc.groups.Load(searchGroupId)
		if !ok {
			return 0, nil, nil
		}
		predicates = append(predicates, func(ctx context.Context, u *authcommon.User) bool {
			_, exist := g.UserIds[u.ID]
			return exist
		})
	}

	result := make([]*authcommon.User, 0, 32)
	uc.users.Range(func(key string, val *authcommon.User) {
		// 超级账户不做展示
		if authcommon.UserRoleType(val.Type) == authcommon.AdminUserRole {
			return
		}
		if hasId && searchId != key {
			return
		}
		if hasOwner && val.Owner != searchOwner {
			return
		}
		if hasName && !utils.IsWildMatch(val.Name, searchName) {
			return
		}
		if hasSource && !utils.IsWildMatch(val.Source, searchSource) {
			return
		}
		for i := range predicates {
			if !predicates[i](ctx, val) {
				return
			}
		}
		result = append(result, val)
	})

	total, ret := uc.listUsersPage(result, args)
	return total, ret, nil
}

func (uc *userCache) listUsersPage(users []*authcommon.User, args types.UserSearchArgs) (uint32, []*authcommon.User) {
	total := uint32(len(users))
	if args.Limit == 0 {
		return total, nil
	}
	start := args.Limit * (args.Offset - 1)
	end := args.Limit * args.Offset
	if start > total {
		return total, nil
	}
	if end > total {
		end = total
	}
	return total, users[start:end]
}

// QueryUserGroups .
func (uc *userCache) QueryUserGroups(ctx context.Context, args types.UserGroupSearchArgs) (uint32, []*authcommon.UserGroupDetail, error) {
	searchId, hasId := args.Filters["id"]
	searchName, hasName := args.Filters["name"]
	searchOwner, hasOwner := args.Filters["owner"]
	searchSource, hasSource := args.Filters["source"]

	predicates := types.LoadUserGroupPredicates(ctx)

	searchUserId, hasUserId := args.Filters["user_id"]
	if hasUserId {
		if _, ok := uc.users.Load(searchUserId); !ok {
			return 0, nil, nil
		}
		predicates = append(predicates, func(ctx context.Context, ugd *authcommon.UserGroupDetail) bool {
			_, exist := ugd.UserIds[searchUserId]
			return exist
		})
	}

	result := make([]*authcommon.UserGroupDetail, 0, 32)
	uc.groups.Range(func(key string, val *authcommon.UserGroupDetail) {
		// 超级账户不做展示
		if hasId && searchId != key {
			return
		}
		if hasOwner && val.Owner != searchOwner {
			return
		}
		if hasName && !utils.IsWildMatch(val.Name, searchName) {
			return
		}
		if hasSource && !utils.IsWildMatch(val.Source, searchSource) {
			return
		}
		for i := range predicates {
			if !predicates[i](ctx, val) {
				return
			}
		}
		result = append(result, val)
	})

	total, ret := uc.listUserGroupsPage(result, args)
	return total, ret, nil
}

func (uc *userCache) listUserGroupsPage(groups []*authcommon.UserGroupDetail, args types.UserGroupSearchArgs) (uint32, []*authcommon.UserGroupDetail) {
	total := uint32(len(groups))
	if args.Limit == 0 {
		return total, nil
	}
	start := args.Limit * (args.Offset - 1)
	end := args.Limit * args.Offset
	if start > total {
		return total, nil
	}
	if end > total {
		end = total
	}
	return total, groups[start:end]
}
