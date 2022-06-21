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
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store/mock"
	"github.com/stretchr/testify/assert"
)

// 创建一个测试mock userCache
func newTestUserCache(t *testing.T) (*gomock.Controller, *mock.MockStore, *userCache) {
	ctl := gomock.NewController(t)

	storage := mock.NewMockStore(ctl)
	uc := newUserCache(storage, make(chan interface{}, 4))
	opt := map[string]interface{}{}
	_ = uc.initialize(opt)

	return ctl, storage, uc.(*userCache)
}

// 生成测试数据
func genModelUsers(total int) []*model.User {
	if total%10 != 0 {
		panic(errors.New("total must like 10, 20, 30, 40, ..."))
	}

	out := make([]*model.User, 0, total)

	var owner *model.User

	for i := 0; i < total; i++ {
		if i%10 == 0 {
			owner = &model.User{
				ID:       fmt.Sprintf("owner-user-%d", i),
				Name:     fmt.Sprintf("owner-user-%d", i),
				Password: fmt.Sprintf("owner-user-%d", i),
				Owner:    "",
				Source:   "Polaris",
				Type:     model.OwnerUserRole,
				Token:    fmt.Sprintf("owner-user-%d", i),
				Valid:    true,
			}
			out = append(out, owner)
			continue
		}

		entry := &model.User{
			ID:       fmt.Sprintf("sub-user-%d", i),
			Name:     fmt.Sprintf("sub-user-%d", i),
			Password: fmt.Sprintf("sub-user-%d", i),
			Owner:    owner.ID,
			Source:   "Polaris",
			Type:     model.SubAccountUserRole,
			Token:    fmt.Sprintf("sub-user-%d", i),
			Valid:    true,
		}

		out = append(out, entry)
	}
	return out
}

func genModelUserGroups(users []*model.User) []*model.UserGroupDetail {

	out := make([]*model.UserGroupDetail, 0, len(users))

	for i := 0; i < len(users); i++ {
		entry := &model.UserGroupDetail{
			UserGroup: &model.UserGroup{
				ID:          utils.NewUUID(),
				Name:        fmt.Sprintf("group-%d", i),
				Owner:       users[0].ID,
				Token:       users[i].Token,
				TokenEnable: true,
				Valid:       true,
				Comment:     "",
				CreateTime:  time.Time{},
				ModifyTime:  time.Time{},
			},
			UserIds: map[string]struct{}{
				users[i].ID: {},
			},
		}

		out = append(out, entry)
	}
	return out
}

func TestUserCache_UpdateNormal(t *testing.T) {

	ctrl, store, uc := newTestUserCache(t)

	defer ctrl.Finish()

	users := genModelUsers(10)
	groups := genModelUserGroups(users)

	copyUsers := make([]*model.User, 0, len(users))
	copyGroups := make([]*model.UserGroupDetail, 0, len(groups))

	for i := range users {
		copyUser := *users[i]
		copyUsers = append(copyUsers, &copyUser)
	}

	for i := range groups {
		copyGroup := *groups[i]
		newUserIds := make(map[string]struct{}, len(copyGroup.UserIds))
		for k, v := range groups[i].UserIds {
			newUserIds[k] = v
		}
		copyGroup.UserIds = newUserIds
		copyGroups = append(copyGroups, &copyGroup)
	}

	t.Run("首次更新用户", func(t *testing.T) {
		store.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).Return(copyUsers, nil)
		store.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).Return(copyGroups, nil)

		err := uc.update(0)
		assert.NoError(t, err, err)

		u := uc.GetUserByID(users[1].ID)
		assert.NotNil(t, u)
		assert.Equal(t, u, users[1])

		u = uc.GetUserByName(users[1].Name, users[0].Name)
		assert.NotNil(t, u)
		assert.Equal(t, u, users[1])

		g := uc.GetGroup(groups[1].ID)
		assert.NotNil(t, g)
		assert.Equal(t, g, groups[1])

		gid := uc.GetUserLinkGroupIds(users[1].ID)
		assert.Equal(t, 1, len(gid))
		assert.Equal(t, groups[1].ID, gid[0])
	})

	t.Run("部分用户删除", func(t *testing.T) {

		deleteCnt := 0
		for i := range users {
			if rand.Int31n(3) < 1 {
				users[i].Valid = false
				delete(groups[i].UserIds, users[i].ID)
				deleteCnt++
			}

			users[i].Comment = fmt.Sprintf("update user %d", i)
		}

		copyUsers := make([]*model.User, 0, len(users))
		copyGroups := make([]*model.UserGroupDetail, 0, len(groups))

		for i := range users {
			copyUser := *users[i]
			copyUsers = append(copyUsers, &copyUser)
		}

		for i := range groups {
			copyGroup := *groups[i]
			newUserIds := make(map[string]struct{}, len(copyGroup.UserIds))
			for k, v := range groups[i].UserIds {
				newUserIds[k] = v
			}
			copyGroup.UserIds = newUserIds
			copyGroups = append(copyGroups, &copyGroup)
		}

		store.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).Return(copyUsers, nil)
		store.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).Return(copyGroups, nil)

		err := uc.update(0)
		assert.NoError(t, err, err)

		for i := range users {
			u := uc.GetUserByID(users[i].ID)
			if users[i].Valid {
				assert.NotNil(t, u)
				assert.Equal(t, u, users[i])

				u = uc.GetUserByName(users[i].Name, users[0].Name)
				assert.NotNil(t, u)
				assert.Equal(t, u, users[i])

				g := uc.GetGroup(groups[i].ID)
				assert.NotNil(t, g)
				assert.Equal(t, g, groups[i])

				gid := uc.GetUserLinkGroupIds(users[i].ID)
				assert.Equal(t, 1, len(gid))
				assert.Equal(t, groups[i].ID, gid[0])
			} else {
				assert.Nil(t, u)

				u = uc.GetUserByName(users[i].Name, users[0].Name)
				assert.Nil(t, u)

				g := uc.GetGroup(groups[i].ID)
				assert.NotNil(t, g)
				assert.Equal(t, g, groups[i])
				assert.Equal(t, 0, len(groups[i].UserIds))

				gid := uc.GetUserLinkGroupIds(users[i].ID)
				assert.Equal(t, 0, len(gid))
			}
		}

	})

}
