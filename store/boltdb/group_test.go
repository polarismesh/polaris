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

package boltdb

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/common/model"
)

func buildUserIds(users []*model.User) map[string]struct{} {
	ret := make(map[string]struct{}, len(users))

	for i := range users {
		user := users[i]
		ret[user.ID] = struct{}{}
	}

	return ret
}

func createTestUserGroup(num int) []*model.UserGroupDetail {
	ret := make([]*model.UserGroupDetail, 0, num)

	users := createTestUsers(num)

	for i := 0; i < num; i++ {
		ret = append(ret, &model.UserGroupDetail{
			UserGroup: &model.UserGroup{
				ID:          fmt.Sprintf("test_group_%d", i),
				Name:        fmt.Sprintf("test_group_%d", i),
				Owner:       "polaris",
				Token:       "polaris",
				TokenEnable: true,
				Valid:       true,
				Comment:     "polaris",
				CreateTime:  time.Now(),
				ModifyTime:  time.Now(),
			},
			UserIds: buildUserIds(users),
		})
	}

	return ret
}

func Test_groupStore_AddGroup(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_group", func(t *testing.T, handler BoltHandler) {
		gs := &groupStore{handler: handler}

		groups := createTestUserGroup(1)

		if err := gs.AddGroup(groups[0]); err != nil {
			t.Fatal(err)
		}

		ret, err := gs.GetGroup(groups[0].ID)
		if err != nil {
			t.Fatal(err)
		}

		tn := time.Now()
		groups[0].CreateTime = tn
		groups[0].ModifyTime = tn
		ret.CreateTime = tn
		ret.ModifyTime = tn

		assert.Equal(t, groups[0], ret)
	})
}

func Test_groupStore_UpdateGroup(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_group", func(t *testing.T, handler BoltHandler) {
		gs := &groupStore{handler: handler}

		groups := createTestUserGroup(1)

		if err := gs.AddGroup(groups[0]); err != nil {
			t.Fatal(err)
		}

		groups[0].Comment = time.Now().String()

		if err := gs.UpdateGroup(&model.ModifyUserGroup{
			ID:          groups[0].ID,
			Owner:       groups[0].Owner,
			Token:       groups[0].Token,
			TokenEnable: groups[0].TokenEnable,
			Comment:     groups[0].Comment,
		}); err != nil {
			t.Fatal(err)
		}

		ret, err := gs.GetGroup(groups[0].ID)
		if err != nil {
			t.Fatal(err)
		}

		tn := time.Now()
		groups[0].CreateTime = tn
		groups[0].ModifyTime = tn
		ret.CreateTime = tn
		ret.ModifyTime = tn

		assert.Equal(t, groups[0], ret)
	})
}

func Test_groupStore_DeleteGroup(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_group", func(t *testing.T, handler BoltHandler) {
		gs := &groupStore{handler: handler}

		groups := createTestUserGroup(1)

		if err := gs.AddGroup(groups[0]); err != nil {
			t.Fatal(err)
		}

		groups[0].Comment = time.Now().String()

		if err := gs.DeleteGroup(groups[0]); err != nil {
			t.Fatal(err)
		}

		ret, err := gs.GetGroup(groups[0].ID)
		if err != nil {
			t.Fatal(err)
		}

		assert.Nil(t, ret)
	})
}

func Test_groupStore_GetGroupByName(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_group", func(t *testing.T, handler BoltHandler) {
		gs := &groupStore{handler: handler}

		groups := createTestUserGroup(1)

		if err := gs.AddGroup(groups[0]); err != nil {
			t.Fatal(err)
		}

		ret, err := gs.GetGroupByName(groups[0].Name, groups[0].Owner)
		if err != nil {
			t.Fatal(err)
		}

		tn := time.Now()
		groups[0].CreateTime = tn
		groups[0].ModifyTime = tn
		ret.CreateTime = tn
		ret.ModifyTime = tn

		assert.Equal(t, groups[0].UserGroup, ret)
	})
}

func Test_groupStore_GetGroups(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_group", func(t *testing.T, handler BoltHandler) {
		gs := &groupStore{handler: handler}

		groups := createTestUserGroup(10)

		for i := range groups {
			if err := gs.AddGroup(groups[i]); err != nil {
				t.Fatal(err)
			}
		}

		total, ret, err := gs.GetGroups(map[string]string{
			"name": "gr*",
		}, 0, 2)
		if err != nil {
			t.Fatal(err)
		}

		if !assert.Equal(t, 2, len(ret)) {
			t.Fatal("len(ret) need equal 2")
		}

		if !assert.Equal(t, len(groups), int(total)) {
			t.Fatal("total != len(groups)")
		}

		total, ret, err = gs.GetGroups(map[string]string{
			"name": "gr*",
		}, 100, 2)
		if err != nil {
			t.Fatal(err)
		}

		if !assert.Equal(t, 0, len(ret)) {
			t.Fatal("len(ret) need zero")
		}

		if !assert.Equal(t, len(groups), int(total)) {
			t.Fatal("total != len(groups)")
		}

		total, ret, err = gs.GetGroups(map[string]string{
			"name": "gr*",
		}, 0, 100)
		if err != nil {
			t.Fatal(err)
		}

		if !assert.Equal(t, len(groups), len(ret)) {
			t.Fatal("len(ret) need zero")
		}

		if !assert.Equal(t, len(groups), int(total)) {
			t.Fatal("total != len(groups)")
		}

		total, ret, err = gs.GetGroups(map[string]string{
			"user_id": "user_1",
		}, 0, 100)
		if err != nil {
			t.Fatal(err)
		}

		if !assert.Equal(t, len(groups), len(ret)) {
			t.Fatal("len(ret) need zero")
		}

		if !assert.Equal(t, len(groups), int(total)) {
			t.Fatal("total != len(groups)")
		}

		total, ret, err = gs.GetGroups(map[string]string{
			"owner": "polaris",
		}, 0, 100)
		if err != nil {
			t.Fatal(err)
		}

		if !assert.Equal(t, len(groups), len(ret)) {
			t.Fatal("len(ret) need zero")
		}

		if !assert.Equal(t, len(groups), int(total)) {
			t.Fatal("total != len(groups)")
		}
	})
}
