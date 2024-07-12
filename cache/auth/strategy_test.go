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
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	"github.com/stretchr/testify/assert"

	types "github.com/polarismesh/polaris/cache/api"
	cachemock "github.com/polarismesh/polaris/cache/mock"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store/mock"
)

func Test_strategyCache(t *testing.T) {
	t.Run("get_policy", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCacheMgr := cachemock.NewMockCacheManager(ctrl)
		mockStore := mock.NewMockStore(ctrl)

		t.Cleanup(func() {
			ctrl.Finish()
		})

		userCache := NewUserCache(mockStore, mockCacheMgr)
		strategyCache := NewStrategyCache(mockStore, mockCacheMgr).(*strategyCache)

		mockStore.EXPECT().GetUnixSecond(gomock.Any()).Return(time.Now().Unix(), nil)
		mockStore.EXPECT().GetStrategyDetailsForCache(gomock.Any(), gomock.Any()).Return(buildStrategies(10), nil).AnyTimes()
		mockCacheMgr.EXPECT().GetCacher(types.CacheUser).Return(userCache).AnyTimes()
		mockCacheMgr.EXPECT().GetReportInterval().Return(time.Second).AnyTimes()
		mockCacheMgr.EXPECT().GetUpdateCacheInterval().Return(time.Second).AnyTimes()

		userCache.Initialize(map[string]interface{}{})
		strategyCache.Initialize(map[string]interface{}{})

		_ = strategyCache.ForceSync()
		_, _, _ = strategyCache.realUpdate()

		policies := strategyCache.GetStrategyDetailsByUID("user-1")
		assert.True(t, len(policies) > 0, len(policies))

		policies = strategyCache.GetStrategyDetailsByGroupID("group-1")
		assert.True(t, len(policies) > 0, len(policies))

		policies = strategyCache.GetStrategyDetailsByUID("fake-user-1")
		assert.True(t, len(policies) == 0, len(policies))

		policies = strategyCache.GetStrategyDetailsByGroupID("fake-group-1")
		assert.True(t, len(policies) == 0, len(policies))
	})

	t.Run("资源没有关联任何策略", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCacheMgr := cachemock.NewMockCacheManager(ctrl)
		mockStore := mock.NewMockStore(ctrl)

		t.Cleanup(func() {
			ctrl.Finish()
		})

		userCache := NewUserCache(mockStore, mockCacheMgr)
		strategyCache := NewStrategyCache(mockStore, mockCacheMgr).(*strategyCache)

		mockStore.EXPECT().GetUnixSecond(gomock.Any()).Return(time.Now().Unix(), nil)
		mockStore.EXPECT().GetStrategyDetailsForCache(gomock.Any(), gomock.Any()).Return(buildStrategies(10), nil).AnyTimes()
		mockCacheMgr.EXPECT().GetCacher(types.CacheUser).Return(userCache).AnyTimes()
		mockCacheMgr.EXPECT().GetReportInterval().Return(time.Second).AnyTimes()
		mockCacheMgr.EXPECT().GetUpdateCacheInterval().Return(time.Second).AnyTimes()

		userCache.Initialize(map[string]interface{}{})
		strategyCache.Initialize(map[string]interface{}{})

		_ = strategyCache.ForceSync()
		_, _, _ = strategyCache.realUpdate()

		ret := strategyCache.IsResourceEditable(authcommon.Principal{
			PrincipalID:   "user-1",
			PrincipalRole: authcommon.PrincipalUser,
		}, apisecurity.ResourceType_Namespaces, "namespace-1")

		assert.True(t, ret, "must be true")

		ret = strategyCache.IsResourceLinkStrategy(apisecurity.ResourceType_Namespaces, "namespace-1")
		assert.True(t, ret, "must be true")
		ret = strategyCache.IsResourceLinkStrategy(apisecurity.ResourceType_Services, "service-1")
		assert.True(t, ret, "must be true")
		ret = strategyCache.IsResourceLinkStrategy(apisecurity.ResourceType_ConfigGroups, "config_group-1")
		assert.True(t, ret, "must be true")

		strategyCache.Clear()
	})

	t.Run("操作的目标资源关联了策略-自己在principal-user列表中", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCacheMgr := cachemock.NewMockCacheManager(ctrl)
		mockStore := mock.NewMockStore(ctrl)

		t.Cleanup(func() {
			ctrl.Finish()
		})

		userCache := NewUserCache(mockStore, mockCacheMgr)
		strategyCache := NewStrategyCache(mockStore, mockCacheMgr).(*strategyCache)

		mockCacheMgr.EXPECT().GetCacher(types.CacheUser).Return(userCache).AnyTimes()
		mockCacheMgr.EXPECT().GetReportInterval().Return(time.Second).AnyTimes()
		mockCacheMgr.EXPECT().GetUpdateCacheInterval().Return(time.Second).AnyTimes()

		userCache.Initialize(map[string]interface{}{})
		strategyCache.Initialize(map[string]interface{}{})

		strategyCache.setStrategys([]*authcommon.StrategyDetail{
			{
				ID:   fmt.Sprintf("rule-%d", 1),
				Name: fmt.Sprintf("rule-%d", 1),
				Principals: []authcommon.Principal{
					{
						PrincipalID:   "user-1",
						PrincipalRole: authcommon.PrincipalUser,
					},
				},
				Valid: true,
				Resources: []authcommon.StrategyResource{
					{
						StrategyID: fmt.Sprintf("rule-%d", 1),
						ResType:    0,
						ResID:      "*",
					},
				},
			},
		})

		ret := strategyCache.IsResourceEditable(authcommon.Principal{
			PrincipalID:   "user-1",
			PrincipalRole: authcommon.PrincipalUser,
		}, apisecurity.ResourceType_Namespaces, "namespace-1")

		assert.True(t, ret, "must be true")
	})

	t.Run("操作的目标资源关联了策略-自己不在principal-user列表中", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCacheMgr := cachemock.NewMockCacheManager(ctrl)
		mockStore := mock.NewMockStore(ctrl)

		t.Cleanup(func() {
			ctrl.Finish()
		})

		userCache := NewUserCache(mockStore, mockCacheMgr)
		strategyCache := NewStrategyCache(mockStore, mockCacheMgr).(*strategyCache)

		mockCacheMgr.EXPECT().GetCacher(types.CacheUser).Return(userCache).AnyTimes()
		mockCacheMgr.EXPECT().GetReportInterval().Return(time.Second).AnyTimes()
		mockCacheMgr.EXPECT().GetUpdateCacheInterval().Return(time.Second).AnyTimes()

		userCache.Initialize(map[string]interface{}{})
		strategyCache.Initialize(map[string]interface{}{})

		strategyCache.setStrategys(buildStrategies(10))

		ret := strategyCache.IsResourceEditable(authcommon.Principal{
			PrincipalID:   "user-20",
			PrincipalRole: authcommon.PrincipalUser,
		}, apisecurity.ResourceType_Namespaces, "namespace-1")
		assert.False(t, ret, "must be false")

		ret = strategyCache.IsResourceEditable(authcommon.Principal{
			PrincipalID:   "user-20",
			PrincipalRole: authcommon.PrincipalUser,
		}, apisecurity.ResourceType_Services, "service-1")
		assert.False(t, ret, "must be false")

		ret = strategyCache.IsResourceEditable(authcommon.Principal{
			PrincipalID:   "user-20",
			PrincipalRole: authcommon.PrincipalUser,
		}, apisecurity.ResourceType_ConfigGroups, "config_group-1")
		assert.False(t, ret, "must be false")
	})

	t.Run("操作的目标资源关联了策略-自己属于principal-group中组成员", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCacheMgr := cachemock.NewMockCacheManager(ctrl)
		mockStore := mock.NewMockStore(ctrl)

		t.Cleanup(func() {
			ctrl.Finish()
		})

		userCache := NewUserCache(mockStore, mockCacheMgr).(*userCache)
		strategyCache := NewStrategyCache(mockStore, mockCacheMgr).(*strategyCache)

		mockCacheMgr.EXPECT().GetCacher(types.CacheUser).Return(userCache).AnyTimes()
		mockCacheMgr.EXPECT().GetReportInterval().Return(time.Second).AnyTimes()
		mockCacheMgr.EXPECT().GetUpdateCacheInterval().Return(time.Second).AnyTimes()

		userCache.Initialize(map[string]interface{}{})
		strategyCache.Initialize(map[string]interface{}{})

		userCache.groups.Store("group-1", &authcommon.UserGroupDetail{
			UserGroup: &authcommon.UserGroup{
				ID: "group-1",
			},
			UserIds: map[string]struct{}{
				"user-1": {},
			},
		})

		userCache.user2Groups.Store("user-1", utils.NewSyncSet[string]())
		links, _ := userCache.user2Groups.Load("user-1")
		links.Add("group-1")

		strategyCache.setStrategys(buildStrategies(10))

		ret := strategyCache.IsResourceEditable(authcommon.Principal{
			PrincipalID:   "user-1",
			PrincipalRole: authcommon.PrincipalUser,
		}, apisecurity.ResourceType_Namespaces, "namespace-1")

		assert.True(t, ret, "must be true")
	})

	t.Run("操作关联策略的资源-策略在操作成功-策略移除操作失败", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCacheMgr := cachemock.NewMockCacheManager(ctrl)
		mockStore := mock.NewMockStore(ctrl)

		t.Cleanup(func() {
			ctrl.Finish()
		})

		userCache := NewUserCache(mockStore, mockCacheMgr).(*userCache)
		strategyCache := NewStrategyCache(mockStore, mockCacheMgr).(*strategyCache)

		mockCacheMgr.EXPECT().GetCacher(types.CacheUser).Return(userCache).AnyTimes()
		mockCacheMgr.EXPECT().GetReportInterval().Return(time.Second).AnyTimes()
		mockCacheMgr.EXPECT().GetUpdateCacheInterval().Return(time.Second).AnyTimes()

		userCache.Initialize(map[string]interface{}{})
		strategyCache.Initialize(map[string]interface{}{})

		userCache.groups.Store("group-1", &authcommon.UserGroupDetail{
			UserGroup: &authcommon.UserGroup{
				ID: "group-1",
			},
			UserIds: map[string]struct{}{
				"user-1": {},
			},
		})

		userCache.user2Groups.Store("user-1", utils.NewSyncSet[string]())
		links, _ := userCache.user2Groups.Load("user-1")
		links.Add("group-1")
		strategyCache.strategys.Store("rule-1", &authcommon.StrategyDetailCache{
			StrategyDetail: &authcommon.StrategyDetail{
				ID:         "rule-1",
				Name:       "rule-1",
				Principals: []authcommon.Principal{},
				Resources:  []authcommon.StrategyResource{},
			},
			GroupPrincipal: map[string]authcommon.Principal{
				"group-1": {
					PrincipalID: "group-1",
				},
			},
		})
		strategyCache.strategys.Store("rule-2", &authcommon.StrategyDetailCache{
			StrategyDetail: &authcommon.StrategyDetail{
				ID:         "rule-2",
				Name:       "rule-2",
				Principals: []authcommon.Principal{},
				Resources:  []authcommon.StrategyResource{},
			},
			GroupPrincipal: map[string]authcommon.Principal{
				"group-2": {
					PrincipalID: "group-2",
				},
			},
		})

		strategyCache.writeSet(strategyCache.namespace2Strategy, "namespace-1", "rule-1", false)
		strategyCache.writeSet(strategyCache.namespace2Strategy, "namespace-1", "rule-2", false)

		ret := strategyCache.IsResourceEditable(authcommon.Principal{
			PrincipalID:   "user-1",
			PrincipalRole: authcommon.PrincipalUser,
		}, apisecurity.ResourceType_Namespaces, "namespace-1")

		assert.True(t, ret, "must be true")

		strategyCache.handlerResourceStrategy([]*authcommon.StrategyDetail{
			{
				ID:         "rule-1",
				Name:       "rule-1",
				Valid:      false,
				Principals: []authcommon.Principal{},
				Resources: []authcommon.StrategyResource{
					{
						StrategyID: "rule-1",
						ResType:    0,
						ResID:      "namespace-1",
					},
				},
			},
		})

		ret = strategyCache.IsResourceEditable(authcommon.Principal{
			PrincipalID:   "user-1",
			PrincipalRole: authcommon.PrincipalUser,
		}, apisecurity.ResourceType_Namespaces, "namespace-1")

		assert.False(t, ret, "must be false")
	})

	t.Run("", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCacheMgr := cachemock.NewMockCacheManager(ctrl)
		mockStore := mock.NewMockStore(ctrl)

		t.Cleanup(func() {
			ctrl.Finish()
		})

		userCache := NewUserCache(mockStore, mockCacheMgr).(*userCache)
		strategyCache := NewStrategyCache(mockStore, mockCacheMgr).(*strategyCache)

		mockCacheMgr.EXPECT().GetCacher(types.CacheUser).Return(userCache).AnyTimes()
		mockCacheMgr.EXPECT().GetReportInterval().Return(time.Second).AnyTimes()
		mockCacheMgr.EXPECT().GetUpdateCacheInterval().Return(time.Second).AnyTimes()

		userCache.Initialize(map[string]interface{}{})
		strategyCache.Initialize(map[string]interface{}{})

		userCache.groups.Store("group-1", &authcommon.UserGroupDetail{
			UserGroup: &authcommon.UserGroup{
				ID: "group-1",
			},
			UserIds: map[string]struct{}{
				"user-1": {},
			},
		})

		strategyDetail := &authcommon.StrategyDetail{
			ID:   "rule-1",
			Name: "rule-1",
			Principals: []authcommon.Principal{
				{
					PrincipalID:   "user-1",
					PrincipalRole: authcommon.PrincipalUser,
				},
				{
					PrincipalID:   "group-1",
					PrincipalRole: authcommon.PrincipalGroup,
				},
			},
			Valid: true,
			Resources: []authcommon.StrategyResource{
				{
					StrategyID: "rule-1",
					ResType:    0,
					ResID:      "*",
				},
			},
		}

		strategyDetail2 := &authcommon.StrategyDetail{
			ID:   "rule-2",
			Name: "rule-2",
			Principals: []authcommon.Principal{
				{
					PrincipalID:   "user-2",
					PrincipalRole: authcommon.PrincipalUser,
				},
				{
					PrincipalID:   "group-2",
					PrincipalRole: authcommon.PrincipalGroup,
				},
			},
			Valid: true,
			Resources: []authcommon.StrategyResource{
				{
					StrategyID: "rule-2",
					ResType:    0,
					ResID:      "namespace-1",
				},
			},
		}

		strategyCache.strategys.Store("rule-1", &authcommon.StrategyDetailCache{
			StrategyDetail: strategyDetail,
			UserPrincipal: map[string]authcommon.Principal{
				"user-1": {
					PrincipalID: "user-1",
				},
			},
			GroupPrincipal: map[string]authcommon.Principal{
				"group-1": {
					PrincipalID: "group-1",
				},
			},
		})
		strategyCache.strategys.Store("rule-2", &authcommon.StrategyDetailCache{
			StrategyDetail: strategyDetail2,
			UserPrincipal: map[string]authcommon.Principal{
				"user-2": {
					PrincipalID: "user-2",
				},
			},
			GroupPrincipal: map[string]authcommon.Principal{
				"group-2": {
					PrincipalID: "group-2",
				},
			},
		})

		strategyCache.handlerPrincipalStrategy([]*authcommon.StrategyDetail{strategyDetail2})
		strategyCache.handlerResourceStrategy([]*authcommon.StrategyDetail{strategyDetail2})
		strategyCache.handlerPrincipalStrategy([]*authcommon.StrategyDetail{strategyDetail})
		strategyCache.handlerResourceStrategy([]*authcommon.StrategyDetail{strategyDetail})
		ret := strategyCache.IsResourceEditable(authcommon.Principal{
			PrincipalID:   "user-1",
			PrincipalRole: authcommon.PrincipalUser,
		}, apisecurity.ResourceType_Namespaces, "namespace-1")

		assert.True(t, ret, "must be true")

		ret = strategyCache.IsResourceLinkStrategy(apisecurity.ResourceType_Namespaces, "namespace-1")
		assert.True(t, ret, "must be true")

		strategyDetail.Valid = false

		strategyCache.handlerPrincipalStrategy([]*authcommon.StrategyDetail{strategyDetail})
		strategyCache.handlerResourceStrategy([]*authcommon.StrategyDetail{strategyDetail})
		strategyCache.strategys.Delete(strategyDetail.ID)
		ret = strategyCache.IsResourceEditable(authcommon.Principal{
			PrincipalID:   "user-1",
			PrincipalRole: authcommon.PrincipalUser,
		}, apisecurity.ResourceType_Namespaces, "namespace-1")

		assert.False(t, ret, "must be false")
	})
}

func buildStrategies(num int) []*authcommon.StrategyDetail {

	ret := make([]*authcommon.StrategyDetail, 0, num)

	for i := 0; i < num; i++ {
		principals := make([]authcommon.Principal, 0, num)
		for j := 0; j < num; j++ {
			principals = append(principals, authcommon.Principal{
				PrincipalID:   fmt.Sprintf("user-%d", i+1),
				PrincipalRole: authcommon.PrincipalUser,
			}, authcommon.Principal{
				PrincipalID:   fmt.Sprintf("group-%d", i+1),
				PrincipalRole: authcommon.PrincipalGroup,
			})
		}

		ret = append(ret, &authcommon.StrategyDetail{
			ID:         fmt.Sprintf("rule-%d", i+1),
			Name:       fmt.Sprintf("rule-%d", i+1),
			Principals: principals,
			Valid:      true,
			Resources: []authcommon.StrategyResource{
				{
					StrategyID: fmt.Sprintf("rule-%d", i+1),
					ResType:    0,
					ResID:      fmt.Sprintf("namespace-%d", i+1),
				},
				{
					StrategyID: fmt.Sprintf("rule-%d", i+1),
					ResType:    1,
					ResID:      fmt.Sprintf("service-%d", i+1),
				},
				{
					StrategyID: fmt.Sprintf("rule-%d", i+1),
					ResType:    2,
					ResID:      fmt.Sprintf("config_group-%d", i+1),
				},
			},
		})
	}

	return ret
}

func testBuildPrincipalMap(principals []authcommon.Principal, role authcommon.PrincipalType) map[string]authcommon.Principal {
	ret := make(map[string]authcommon.Principal, 0)
	for i := range principals {
		principal := principals[i]
		if principal.PrincipalRole == role {
			ret[principal.PrincipalID] = principal
		}
	}

	return ret
}
