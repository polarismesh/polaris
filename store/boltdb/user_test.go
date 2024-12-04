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
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	authcommon "github.com/polarismesh/polaris/common/model/auth"
)

func createTestUsers(num int) []*authcommon.User {
	ret := make([]*authcommon.User, 0, num)

	for i := 0; i < num; i++ {
		ret = append(ret, &authcommon.User{
			ID:          fmt.Sprintf("user_%d", i),
			Name:        fmt.Sprintf("user_%d", i),
			Password:    fmt.Sprintf("user_%d", i),
			Owner:       "polaris",
			Source:      "Polaris",
			Type:        authcommon.SubAccountUserRole,
			Token:       "polaris",
			TokenEnable: true,
			Valid:       true,
			Comment:     "",
			CreateTime:  time.Now(),
			ModifyTime:  time.Now(),
		})
	}

	return ret
}

func Test_userStore_AddUser(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_user", func(t *testing.T, handler BoltHandler) {
		us := &userStore{handler: handler}

		users := createTestUsers(1)
		tx, err := handler.StartTx()
		assert.NoError(t, err)
		if err := us.AddUser(tx, users[0]); err != nil {
			t.Fatal(err)
		}
		assert.NoError(t, tx.Commit())

		ret, err := us.GetUser(users[0].ID)
		if err != nil {
			t.Fatal(err)
		}

		tn := time.Now()

		users[0].CreateTime = tn
		users[0].ModifyTime = tn
		ret.CreateTime = tn
		ret.ModifyTime = tn

		if !assert.Equal(t, users[0], ret) {
			t.FailNow()
		}
	})
}

func Test_userStore_UpdateUser(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_user", func(t *testing.T, handler BoltHandler) {
		us := &userStore{handler: handler}

		users := createTestUsers(1)

		tx, err := handler.StartTx()
		assert.NoError(t, err)
		if err := us.AddUser(tx, users[0]); err != nil {
			t.Fatal(err)
		}
		assert.NoError(t, tx.Commit())

		users[0].Comment = "user update test"

		if err := us.UpdateUser(users[0]); err != nil {
			t.Fatal(err)
		}

		ret, err := us.GetUser(users[0].ID)
		if err != nil {
			t.Fatal(err)
		}

		tn := time.Now()

		users[0].CreateTime = tn
		users[0].ModifyTime = tn
		ret.CreateTime = tn
		ret.ModifyTime = tn

		if !assert.Equal(t, users[0], ret) {
			t.FailNow()
		}
	})
}

func Test_userStore_DeleteUser(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_user", func(t *testing.T, handler BoltHandler) {
		us := &userStore{handler: handler}

		users := createTestUsers(1)

		tx, err := handler.StartTx()
		assert.NoError(t, err)
		if err := us.AddUser(tx, users[0]); err != nil {
			t.Fatal(err)
		}
		assert.NoError(t, tx.Commit())

		ret, err := us.GetUser(users[0].ID)
		if err != nil {
			t.Fatal(err)
		}

		if !assert.NotNil(t, ret) {
			t.FailNow()
		}

		tx, err = handler.StartTx()
		assert.NoError(t, err)
		if err = us.DeleteUser(tx, users[0]); err != nil {
			t.Fatal(err)
		}
		assert.NoError(t, tx.Commit())

		ret, err = us.GetUser(users[0].ID)
		if err != nil {
			t.Fatal(err)
		}

		if !assert.Nil(t, ret) {
			t.FailNow()
		}
	})
}

func Test_userStore_GetUserByName(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_user", func(t *testing.T, handler BoltHandler) {
		us := &userStore{handler: handler}

		users := createTestUsers(1)

		tx, err := handler.StartTx()
		assert.NoError(t, err)
		if err := us.AddUser(tx, users[0]); err != nil {
			t.Fatal(err)
		}
		assert.NoError(t, tx.Commit())

		ret, err := us.GetUserByName(users[0].Name, users[0].Owner)
		if err != nil {
			t.Fatal(err)
		}

		tn := time.Now()

		users[0].CreateTime = tn
		users[0].ModifyTime = tn
		ret.CreateTime = tn
		ret.ModifyTime = tn

		if !assert.Equal(t, users[0], ret) {
			t.FailNow()
		}
	})
}

func Test_userStore_GetUserByIds(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_user", func(t *testing.T, handler BoltHandler) {
		us := &userStore{handler: handler}

		users := createTestUsers(5)
		ids := make([]string, 0, len(users))

		for i := range users {
			tx, err := handler.StartTx()
			assert.NoError(t, err)
			if err := us.AddUser(tx, users[i]); err != nil {
				t.Fatal(err)
			}
			assert.NoError(t, tx.Commit())
			ids = append(ids, users[i].ID)
		}

		ret, err := us.GetUserByIds(ids)
		if err != nil {
			t.Fatal(err)
		}

		if len(ret) != len(users) {
			t.Fatal("len(ret) != len(users)")
		}

		tn := time.Now()

		sort.Slice(users, func(i, j int) bool {

			users[i].CreateTime = tn
			users[i].ModifyTime = tn
			users[j].CreateTime = tn
			users[j].ModifyTime = tn

			return strings.Compare(users[i].ID, users[j].ID) < 0
		})

		sort.Slice(ret, func(i, j int) bool {

			ret[i].CreateTime = tn
			ret[i].ModifyTime = tn
			ret[j].CreateTime = tn
			ret[j].ModifyTime = tn

			return strings.Compare(ret[i].ID, ret[j].ID) < 0
		})

		if !assert.ElementsMatch(t, users, ret) {
			t.FailNow()
		}

	})
}

func Test_userStore_GetSubCount(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_user", func(t *testing.T, handler BoltHandler) {
		us := &userStore{handler: handler}

		users := createTestUsers(5)

		for i := range users {
			tx, err := handler.StartTx()
			assert.NoError(t, err)
			if err := us.AddUser(tx, users[i]); err != nil {
				t.Fatal(err)
			}
			assert.NoError(t, tx.Commit())
		}

		total, err := us.GetSubCount(&authcommon.User{
			ID: "polaris",
		})

		if err != nil {
			t.Fatal(err)
		}

		if !assert.Equal(t, int(len(users)), int(total)) {
			t.FailNow()
		}

	})
}
