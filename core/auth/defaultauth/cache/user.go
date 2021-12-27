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

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
)

type UserCache interface {
	Cache

	GetUser(id string) *model.User

	GetUserGroup(id string) *model.UserGroup
}

type userCache struct {
	storage     store.Store
	users       *sync.Map
	groups      *sync.Map
	user2Groups *sync.Map
}

func (uc *userCache) initialize(c map[string]interface{}) error {
	return nil
}

func (uc *userCache) update() error {
	return nil
}

func (uc *userCache) realUpdate() error {
	return nil
}

func (uc *userCache) setUserAndGroups(users map[string]*model.User, groups map[string]*model.UserGroupDetail) error {
	return nil
}

func (uc *userCache) clear() error {
	uc.users = new(sync.Map)
	uc.groups = new(sync.Map)
	return nil
}

func (uc *userCache) name() string {
	return CacheForUser
}

func (uc *userCache) GetUser(id string) *model.User {
	return nil
}

func (uc *userCache) GetUserGroup(id string) *model.UserGroupDetail {
	return nil
}

func (uc *userCache) ListUserBelongGroupIDS(id string) []string {
	return nil
}
