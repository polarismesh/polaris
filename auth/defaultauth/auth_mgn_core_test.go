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

package defaultauth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	storemock "github.com/polarismesh/polaris-server/store/mock"
	"github.com/stretchr/testify/assert"

	_ "github.com/polarismesh/polaris-server/plugin/auth/defaultauth"
)

func Test_defaultAuthManager_ParseToken(t *testing.T) {
	AuthOption.Salt = "polaris@a7b068ce3235442b"
	token := "orRm9Zt7sMqQaAM5b7yHLXnhWsr5dfPT0jpRlQ+C0tdy2UmuDa/X3uFG"

	authMgn := &defaultAuthChecker{}

	tokenInfo, err := authMgn.decodeToken(token)

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%#v", tokenInfo)
}

func Test_defaultAuthChecker_VerifyCredential(t *testing.T) {
	reset(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)

	storage := storemock.NewMockStore(ctrl)

	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(users, nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return([]*model.UserGroupDetail{}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	if err := cache.TestCacheInitialize(ctx, &cache.Config{
		Open: true,
		Resources: []cache.ConfigEntry{
			{
				Name: "users",
			},
		},
	}, storage); err != nil {
		t.Fatal(err)
	}

	cacheMgn, err := cache.GetCacheManager()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		cancel()
		cacheMgn.Clear()
		time.Sleep(2 * time.Second)
	}()

	checker := &defaultAuthChecker{}
	checker.Initialize(&auth.Config{
		Name: "",
		Option: map[string]interface{}{
			"": nil,
		},
	}, cacheMgn)
	checker.cacheMgn = cacheMgn
	checker.authPlugin = plugin.GetAuth()

	t.Run("主账户正常情况", func(t *testing.T) {
		reset(false)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[0].Token),
		)

		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
		assert.Equal(t, users[0].ID, utils.ParseUserID(authCtx.GetRequestContext()), "user-id should be equal")
		assert.True(t, utils.ParseIsOwner(authCtx.GetRequestContext()), "should be owner")
	})

	t.Run("子账户在Token被禁用情况下", func(t *testing.T) {
		reset(false)
		users[1].TokenEnable = false
		// 让 cache 可以刷新到
		time.Sleep(time.Second)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[1].Token),
		)
		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
		assert.Equal(t, users[1].ID, utils.ParseUserID(authCtx.GetRequestContext()), "user-id should be equal")
		assert.False(t, utils.ParseIsOwner(authCtx.GetRequestContext()), "should not be owner")
		assert.True(t, authCtx.GetAttachment(model.TokenDetailInfoKey).(OperatorInfo).Disable, "should be disable")
	})

	t.Run("权限检查非严格模式-错误的token字符串-降级为匿名用户", func(t *testing.T) {
		reset(false)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken("Test_defaultAuthChecker_VerifyCredential"),
		)
		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
		assert.True(t, authCtx.GetAttachment(model.TokenDetailInfoKey).(OperatorInfo).Anonymous, "should be anonymous")
	})

	t.Run("权限检查非严格模式-空token字符串-降级为匿名用户", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(""),
		)
		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
		assert.True(t, authCtx.GetAttachment(model.TokenDetailInfoKey).(OperatorInfo).Anonymous, "should be anonymous")
	})

	t.Run("权限检查非严格模式-错误的token字符串-访问鉴权模块", func(t *testing.T) {
		reset(false)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithModule(model.AuthModule),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken("Test_defaultAuthChecker_VerifyCredential"),
		)
		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查非严格模式-空token字符串-访问鉴权模块", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithModule(model.AuthModule),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(""),
		)
		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-token非法-不允许降级为匿名用户", func(t *testing.T) {
		reset(true)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken("Test_defaultAuthChecker_VerifyCredential"),
		)
		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
		assert.True(t, errors.Is(err, model.ErrorTokenInvalid), "should be token-invalid error")
	})

	t.Run("权限检查严格模式-token为空-不允许降级为匿名用户", func(t *testing.T) {
		reset(true)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(""),
		)
		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
		assert.True(t, errors.Is(err, model.ErrorTokenInvalid), "should be token-invalid error")
	})
}

func Test_defaultAuthChecker_CheckPermission_Write_NoStrict(t *testing.T) {
	reset(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)
	groups := createMockUserGroup(users)

	namespaces := createMockNamespace(len(users)+len(groups)+10, users[0].ID)
	services := createMockService(namespaces)
	serviceMap := convertServiceSliceToMap(services)
	strategies, _ := createMockStrategy(users, groups, services[:len(users)+len(groups)])

	cfg, storage := initCache(ctrl)

	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(users, nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(groups, nil)
	storage.EXPECT().GetStrategyDetailsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(strategies, nil)
	storage.EXPECT().GetMoreNamespaces(gomock.Any()).AnyTimes().Return(namespaces, nil)
	storage.EXPECT().GetMoreServices(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(serviceMap, nil)

	ctx, cancel := context.WithCancel(context.Background())
	if err := cache.TestCacheInitialize(ctx, cfg, storage); err != nil {
		t.Fatal(err)
	}

	cacheMgn, err := cache.GetCacheManager()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		cancel()
		cacheMgn.Clear()
		time.Sleep(2 * time.Second)
	}()

	time.Sleep(time.Second)

	checker := &defaultAuthChecker{}
	checker.cacheMgn = cacheMgn
	checker.authPlugin = plugin.GetAuth()

	freeIndex := len(users) + len(groups) + 1

	t.Run("权限检查非严格模式-主账户资源访问检查", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[0].Token),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-子账户资源访问检查（无操作权限）", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[1].Token),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查非严格模式-子账户资源访问检查（有操作权限）", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[1].Token),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[1].ID,
						Owner: services[1].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-子账户资源访问检查（资源无绑定策略）", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[1].Token),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-子账户访问用户组资源检查（属于用户组成员）", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[1].Token),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[(len(users)-1)+2].ID,
						Owner: services[(len(users)-1)+2].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-子账户访问用户组资源检查（不属于用户组成员）", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[1].Token),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[(len(users)-1)+4].ID,
						Owner: services[(len(users)-1)+4].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查非严格模式-用户组访问组内成员资源检查", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(groups[1].Token),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查非严格模式-token非法-匿名账户资源访问检查（资源无绑定策略）", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken("users[1].Token"),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-token为空-匿名账户资源访问检查（资源无绑定策略）", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(""),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})
}

func Test_defaultAuthChecker_CheckPermission_Write_Strict(t *testing.T) {
	reset(true)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)
	groups := createMockUserGroup(users)

	namespaces := createMockNamespace(len(users)+len(groups)+10, users[0].ID)
	services := createMockService(namespaces)
	serviceMap := convertServiceSliceToMap(services)
	strategies, _ := createMockStrategy(users, groups, services[:len(users)+len(groups)])

	cfg, storage := initCache(ctrl)

	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(users, nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(groups, nil)
	storage.EXPECT().GetStrategyDetailsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(strategies, nil)
	storage.EXPECT().GetMoreNamespaces(gomock.Any()).AnyTimes().Return(namespaces, nil)
	storage.EXPECT().GetMoreServices(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(serviceMap, nil)

	ctx, cancel := context.WithCancel(context.Background())
	if err := cache.TestCacheInitialize(ctx, cfg, storage); err != nil {
		t.Fatal(err)
	}

	cacheMgn, err := cache.GetCacheManager()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		cancel()
		cacheMgn.Clear()
		time.Sleep(2 * time.Second)
	}()

	time.Sleep(time.Second)

	checker := &defaultAuthChecker{}
	checker.cacheMgn = cacheMgn
	checker.authPlugin = plugin.GetAuth()

	freeIndex := len(users) + len(groups) + 1

	t.Run("权限检查严格模式-主账户操作资源", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[0].Token),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查严格模式-子账户操作资源（无操作权限）", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[1].Token),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-子账户操作资源（有操作权限）", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[1].Token),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[1].ID,
						Owner: services[1].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查严格模式-token非法-匿名账户操作资源（资源有策略）", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken("Test_defaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[1].ID,
						Owner: services[1].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-token为空-匿名账户操作资源（资源有策略）", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(""),
			model.WithModule(model.DiscoverModule),
			model.WithOperation(model.Create),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[1].ID,
						Owner: services[1].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-token非法-匿名账户操作资源（资源没有策略）", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken("Test_defaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-token为空-匿名账户操作资源（资源没有策略）", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(""),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})
}

func Test_defaultAuthChecker_CheckPermission_Read_NoStrict(t *testing.T) {
	reset(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)
	groups := createMockUserGroup(users)

	namespaces := createMockNamespace(len(users)+len(groups)+10, users[0].ID)
	services := createMockService(namespaces)
	serviceMap := convertServiceSliceToMap(services)
	strategies, _ := createMockStrategy(users, groups, services[:len(users)+len(groups)])

	cfg, storage := initCache(ctrl)

	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(users, nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(groups, nil)
	storage.EXPECT().GetStrategyDetailsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(strategies, nil)
	storage.EXPECT().GetMoreNamespaces(gomock.Any()).AnyTimes().Return(namespaces, nil)
	storage.EXPECT().GetMoreServices(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(serviceMap, nil)

	ctx, cancel := context.WithCancel(context.Background())
	if err := cache.TestCacheInitialize(ctx, cfg, storage); err != nil {
		t.Fatal(err)
	}

	cacheMgn, err := cache.GetCacheManager()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		cancel()
		cacheMgn.Clear()
		time.Sleep(2 * time.Second)
	}()

	time.Sleep(time.Second)

	checker := &defaultAuthChecker{}
	checker.cacheMgn = cacheMgn
	checker.authPlugin = plugin.GetAuth()

	freeIndex := len(users) + len(groups) + 1

	t.Run("权限检查非严格模式-主账户正常读操作", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[0].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-子账户正常读操作-资源有权限", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[1].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[1].ID,
						Owner: services[1].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-子账户正常读操作-资源无权限", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[1].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-子账户正常读操作-资源无绑定策略", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[1].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-匿名账户正常读操作-token为空-资源有策略", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(""),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-匿名账户正常读操作-token为空-资源无策略", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(""),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-匿名账户正常读操作-token非法-资源有策略", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken("Test_defaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-匿名账户正常读操作-token非法-资源无策略", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken("Test_defaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})
}

func Test_defaultAuthChecker_CheckPermission_Read_Strict(t *testing.T) {
	reset(true)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)
	groups := createMockUserGroup(users)

	namespaces := createMockNamespace(len(users)+len(groups)+10, users[0].ID)
	services := createMockService(namespaces)
	serviceMap := convertServiceSliceToMap(services)
	strategies, _ := createMockStrategy(users, groups, services[:len(users)+len(groups)])

	cfg, storage := initCache(ctrl)

	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(users, nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(groups, nil)
	storage.EXPECT().GetStrategyDetailsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(strategies, nil)
	storage.EXPECT().GetMoreNamespaces(gomock.Any()).AnyTimes().Return(namespaces, nil)
	storage.EXPECT().GetMoreServices(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(serviceMap, nil)

	ctx, cancel := context.WithCancel(context.Background())
	if err := cache.TestCacheInitialize(ctx, cfg, storage); err != nil {
		t.Fatal(err)
	}

	cacheMgn, err := cache.GetCacheManager()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		cancel()
		cacheMgn.Clear()
		time.Sleep(2 * time.Second)
	}()

	time.Sleep(time.Second)

	checker := &defaultAuthChecker{}
	checker.cacheMgn = cacheMgn
	checker.authPlugin = plugin.GetAuth()

	freeIndex := len(users) + len(groups) + 1

	t.Run("权限检查严格模式-主账户正常读操作", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[0].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查严格模式-子账户正常读操作-资源有权限", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[1].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[1].ID,
						Owner: services[1].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查严格模式-子账户正常读操作-资源无权限", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[1].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查严格模式-子账户正常读操作-资源无绑定策略", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(users[1].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查严格模式-匿名账户正常读操作-token为空-资源有策略", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(""),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-匿名账户正常读操作-token为空-资源无策略", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken(""),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-匿名账户正常读操作-token非法-资源有策略", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken("Test_defaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-匿名账户正常读操作-token非法-资源无策略", func(t *testing.T) {
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(context.Background()),
			model.WithMethod("Test_defaultAuthChecker_VerifyCredential"),
			model.WithToken("Test_defaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[api.ResourceType][]model.ResourceEntry{
				api.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckPermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})
}
