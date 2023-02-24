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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store/mock"
	"github.com/stretchr/testify/assert"
)

func mockModelUsers(total int) (*model.User, []*model.User) {
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
			continue
		}

		entry := &model.User{
			ID:       utils.NewUUID(),
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
	return owner, out
}

func mockModelUserGroups(owner *model.User, users []*model.User) []*model.UserGroupDetail {

	out := make([]*model.UserGroupDetail, 0, len(users))

	for i := 0; i < len(users); i++ {
		entry := &model.UserGroupDetail{
			UserGroup: &model.UserGroup{
				ID:          utils.NewUUID(),
				Name:        fmt.Sprintf("group-%d", i),
				Owner:       owner.ID,
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

type mockListener struct {
	onEvent func(t string, value interface{})
}

// OnCreated callback when cache value created
func (sc *mockListener) OnCreated(value interface{}) {
	sc.onEvent("OnCreated", value)
}

// OnUpdated callback when cache value updated
func (sc *mockListener) OnUpdated(value interface{}) {
	sc.onEvent("OnUpdated", value)
}

// OnDeleted callback when cache value deleted
func (sc *mockListener) OnDeleted(value interface{}) {
	sc.onEvent("OnDeleted", value)
}

// OnBatchCreated callback when cache value created
func (sc *mockListener) OnBatchCreated(value interface{}) {
	sc.onEvent("OnBatchCreated", value)
}

// OnBatchUpdated callback when cache value updated
func (sc *mockListener) OnBatchUpdated(value interface{}) {
	sc.onEvent("OnBatchUpdated", value)
}

// OnBatchDeleted callback when cache value deleted
func (sc *mockListener) OnBatchDeleted(value interface{}) {
	sc.onEvent("OnBatchDeleted", value)
}

// Test_PressRefreshUserAndStrategyCache 构造缓存数据更新异常，UserCache 的更新不会阻塞
func Test_PressRefreshUserAndStrategyCache(t *testing.T) {
	uc := newUserCache(nil).(*userCache)
	sc := newStrategyCache(nil, uc).(*strategyCache)

	uc.initialize(map[string]interface{}{})
	sc.initialize(map[string]interface{}{})

	onEventTotal := int32(0)
	uc.addListener([]Listener{&mockListener{
		onEvent: func(label string, value interface{}) {
			if label != "OnBatchDeleted" {
				return
			}
			principals, ok := value.([]model.Principal)

			assert.True(t, ok)
			assert.Equal(t, int32(18), int32(len(principals)))

			atomic.AddInt32(&onEventTotal, 1)
		},
	}})

	t.Run("StrategyCache刷新出现问题", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		storage := mock.NewMockStore(ctrl)
		t.Cleanup(ctrl.Finish)
		storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
		storage.EXPECT().GetStrategyDetailsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, errors.New("mock store busy"))
		storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
			func(mtime time.Time, firstUpdate bool) ([]*model.User, error) {
				owner, users := mockModelUsers(10)
				for i := range users {
					users[i].Valid = false
				}
				return append(users, owner), nil
			})
		storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
			func(mtime time.Time, firstUpdate bool) ([]*model.UserGroupDetail, error) {
				owner, users := mockModelUsers(10)
				groups := mockModelUserGroups(owner, users)
				for i := range groups {
					groups[i].Valid = false
				}
				return groups, nil
			})

		uc.baseCache.s = storage
		uc.storage = storage
		sc.baseCache.s = storage
		sc.storage = storage

		wait := sync.WaitGroup{}
		wait.Add(2)

		go func() {
			defer wait.Done()
			for i := 0; i < removePrincipalChSize*2; i++ {
				_ = uc.update()
			}
		}()

		go func() {
			defer wait.Done()
			for i := 0; i < removePrincipalChSize*2; i++ {
				err := sc.update()
				assert.Error(t, err)
			}
		}()

		wait.Wait()

		assert.Equal(t, int64(0), sc.lastFetchTime)
		assert.Equal(t, int32(removePrincipalChSize*2), atomic.LoadInt32(&onEventTotal))
	})

	time.Sleep(10 * time.Second)

	atomic.StoreInt32(&onEventTotal, int32(0))
	t.Run("StrategyCache刷新恢复", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		ucStorage := mock.NewMockStore(ctrl)
		ucStorage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
		ucStorage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
			func(mtime time.Time, firstUpdate bool) ([]*model.User, error) {
				owner, users := mockModelUsers(10)
				for i := range users {
					users[i].Valid = false
				}
				return append(users, owner), nil
			})
		ucStorage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
			func(mtime time.Time, firstUpdate bool) ([]*model.UserGroupDetail, error) {
				owner, users := mockModelUsers(10)
				groups := mockModelUserGroups(owner, users)
				for i := range groups {
					groups[i].Valid = false
				}
				return groups, nil
			})

		unixData := int64(0)
		scStorage := mock.NewMockStore(ctrl)
		scStorage.EXPECT().GetUnixSecond().AnyTimes().DoAndReturn(func() (int64, error) {
			data := time.Now().Unix()
			atomic.StoreInt64(&unixData, data)
			return data, nil
		})
		scStorage.EXPECT().GetStrategyDetailsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return([]*model.StrategyDetail{}, nil)

		uc.baseCache.s = ucStorage
		uc.storage = ucStorage
		sc.baseCache.s = scStorage
		sc.storage = scStorage

		wait := sync.WaitGroup{}
		wait.Add(2)

		go func() {
			defer wait.Done()
			for i := 0; i < removePrincipalChSize*2; i++ {
				_ = uc.update()
			}
		}()

		go func() {
			defer wait.Done()
			for i := 0; i < removePrincipalChSize*2; i++ {
				err := sc.update()
				assert.NoError(t, err)
			}
		}()

		wait.Wait()

		assert.Equal(t, atomic.LoadInt64(&unixData), sc.lastFetchTime)
		assert.Equal(t, int32(removePrincipalChSize*2), atomic.LoadInt32(&onEventTotal))
	})
}
