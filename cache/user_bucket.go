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

	"github.com/polarismesh/polaris/common/model"
)

type userBucket struct {
	lock sync.RWMutex
	// userid -> user
	users map[string]*model.User
}

func (u *userBucket) get(userId string) (*model.User, bool) {
	u.lock.RLock()
	defer u.lock.RUnlock()

	v, ok := u.users[userId]

	return v, ok
}

func (u *userBucket) save(key string, user *model.User) {
	u.lock.Lock()
	defer u.lock.Unlock()

	u.users[key] = user
}

func (u *userBucket) delete(key string) {
	u.lock.Lock()
	defer u.lock.Unlock()

	delete(u.users, key)
}

type usernameBucket struct {
	lock sync.RWMutex
	// username -> user
	users map[string]*model.User
}

func (u *usernameBucket) get(userId string) (*model.User, bool) {
	u.lock.RLock()
	defer u.lock.RUnlock()

	v, ok := u.users[userId]

	return v, ok
}

func (u *usernameBucket) save(key string, user *model.User) {
	u.lock.Lock()
	defer u.lock.Unlock()

	u.users[key] = user
}

func (u *usernameBucket) delete(key string) {
	u.lock.Lock()
	defer u.lock.Unlock()

	delete(u.users, key)
}

type userGroupsBucket struct {
	lock sync.RWMutex
	// userid -> groups
	groups map[string]*groupIdSlice
}

func (u *userGroupsBucket) get(userId string) (*groupIdSlice, bool) {
	u.lock.RLock()
	defer u.lock.RUnlock()

	v, ok := u.groups[userId]

	return v, ok
}

func (u *userGroupsBucket) save(key string) {
	u.lock.Lock()
	defer u.lock.Unlock()

	if _, ok := u.groups[key]; ok {
		return
	}

	u.groups[key] = &groupIdSlice{
		lock:     sync.RWMutex{},
		groupIds: make(map[string]struct{}),
	}
}

func (u *userGroupsBucket) delete(key string) {
	u.lock.Lock()
	defer u.lock.Unlock()

	delete(u.groups, key)
}

type groupIdSlice struct {
	lock     sync.RWMutex
	groupIds map[string]struct{}
}

func (u *groupIdSlice) toSlice() []string {
	u.lock.RLock()
	defer u.lock.RUnlock()

	ret := make([]string, 0, len(u.groupIds))

	for k := range u.groupIds {
		ret = append(ret, k)
	}

	return ret
}

func (u *groupIdSlice) contains(groupID string) bool {
	u.lock.RLock()
	defer u.lock.RUnlock()

	_, ok := u.groupIds[groupID]

	return ok
}

func (u *groupIdSlice) save(groupID string) {
	u.lock.Lock()
	defer u.lock.Unlock()

	u.groupIds[groupID] = struct{}{}
}

func (u *groupIdSlice) delete(key string) {
	u.lock.Lock()
	defer u.lock.Unlock()

	delete(u.groupIds, key)
}

type groupBucket struct {
	lock   sync.RWMutex
	groups map[string]*model.UserGroupDetail
}

func (u *groupBucket) get(userId string) (*model.UserGroupDetail, bool) {
	u.lock.RLock()
	defer u.lock.RUnlock()

	v, ok := u.groups[userId]

	return v, ok
}

func (u *groupBucket) save(key string, group *model.UserGroupDetail) {
	u.lock.Lock()
	defer u.lock.Unlock()

	u.groups[key] = group
}

func (u *groupBucket) delete(key string) {
	u.lock.Lock()
	defer u.lock.Unlock()

	delete(u.groups, key)
}
