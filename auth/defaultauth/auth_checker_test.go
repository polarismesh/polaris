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

package defaultauth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/auth/defaultauth"
	"github.com/polarismesh/polaris/cache"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	storemock "github.com/polarismesh/polaris/store/mock"
)

func Test_defaultAuthManager_ParseToken(t *testing.T) {
	defaultauth.AuthOption.Salt = "polaris@a7b068ce3235442b"
	token := "orRm9Zt7sMqQaAM5b7yHLXnhWsr5dfPT0jpRlQ+C0tdy2UmuDa/X3uFG"

	authMgn := &defaultauth.DefaultAuthChecker{}

	tokenInfo, err := authMgn.DecodeToken(token)

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%#v", tokenInfo)
}

func Test_DefaultAuthChecker_VerifyCredential(t *testing.T) {
	reset(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)

	storage := storemock.NewMockStore(ctrl)

	storage.EXPECT().GetServicesCount().AnyTimes().Return(uint32(1), nil)
	storage.EXPECT().GetUnixSecond(gomock.Any()).AnyTimes().Return(time.Now().Unix(), nil)
	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(users, nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return([]*model.UserGroupDetail{}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cacheMgn, err := cache.TestCacheInitialize(ctx, &cache.Config{}, storage)
	if err != nil {
		t.Fatal(err)
	}
	_ = cacheMgn.OpenResourceCache([]cachetypes.ConfigEntry{
		{
			Name: cachetypes.UsersName,
		},
	}...)

	_ = cacheMgn.TestUpdate()

	t.Cleanup(func() {
		cancel()
		cacheMgn.Close()
	})

	checker := &defaultauth.DefaultAuthChecker{}
	checker.Initialize(&auth.Config{
		User: &auth.UserConfig{
			Name:   "",
			Option: map[string]interface{}{},
		},
		Strategy: &auth.StrategyConfig{
			Name: "",
			Option: map[string]interface{}{
				"": nil,
			},
		},
	}, storage, cacheMgn)
	checker.SetCacheMgr(cacheMgn)

	t.Run("主账户正常情况", func(t *testing.T) {
		reset(false)
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
		)
		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
		assert.Equal(t, users[1].ID, utils.ParseUserID(authCtx.GetRequestContext()), "user-id should be equal")
		assert.False(t, utils.ParseIsOwner(authCtx.GetRequestContext()), "should not be owner")
		assert.True(t, authCtx.GetAttachment(model.TokenDetailInfoKey).(defaultauth.OperatorInfo).Disable, "should be disable")
	})

	t.Run("权限检查非严格模式-错误的token字符串-降级为匿名用户", func(t *testing.T) {
		reset(false)
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_VerifyCredential")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
		)
		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
		assert.True(t, authCtx.GetAttachment(model.TokenDetailInfoKey).(defaultauth.OperatorInfo).Anonymous, "should be anonymous")
	})

	t.Run("权限检查非严格模式-空token字符串-降级为匿名用户", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
		)
		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
		assert.True(t, authCtx.GetAttachment(model.TokenDetailInfoKey).(defaultauth.OperatorInfo).Anonymous, "should be anonymous")
	})

	t.Run("权限检查非严格模式-错误的token字符串-访问鉴权模块", func(t *testing.T) {
		reset(false)
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_VerifyCredential")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithModule(model.AuthModule),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
		)
		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查非严格模式-空token字符串-访问鉴权模块", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithModule(model.AuthModule),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
		)
		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-token非法-不允许降级为匿名用户", func(t *testing.T) {
		reset(true)
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_VerifyCredential")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
		)
		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
		assert.True(t, errors.Is(err, model.ErrorTokenInvalid), "should be token-invalid error")
	})

	t.Run("权限检查严格模式-token为空-不允许降级为匿名用户", func(t *testing.T) {
		reset(true)
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
		)
		err = checker.VerifyCredential(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
		assert.True(t, errors.Is(err, model.ErrorTokenInvalid), "should be token-invalid error")
	})
}

func Test_DefaultAuthChecker_CheckPermission_Write_NoStrict(t *testing.T) {
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
	cacheMgn, err := cache.TestCacheInitialize(ctx, cfg, storage)
	if err != nil {
		t.Fatal(err)
	}
	_ = cacheMgn.OpenResourceCache([]cachetypes.ConfigEntry{
		{
			Name: cachetypes.UsersName,
		},
		{
			Name: cachetypes.StrategyRuleName,
		},
	}...)
	_ = cacheMgn.TestUpdate()

	t.Cleanup(func() {
		cancel()
		cacheMgn.Close()
	})

	checker := &defaultauth.DefaultAuthChecker{}
	checker.SetCacheMgr(cacheMgn)

	freeIndex := len(users) + len(groups) + 1

	t.Run("权限检查非严格模式-主账户资源访问检查", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groups[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(groups[1].Token),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "users[1].Token")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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

func Test_DefaultAuthChecker_CheckPermission_Write_Strict(t *testing.T) {
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
	cacheMgn, err := cache.TestCacheInitialize(ctx, cfg, storage)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		cacheMgn.Close()
	})

	_ = cacheMgn.OpenResourceCache([]cachetypes.ConfigEntry{
		{
			Name: cachetypes.UsersName,
		},
		{
			Name: cachetypes.StrategyRuleName,
		},
	}...)
	_ = cacheMgn.TestUpdate()

	checker := &defaultauth.DefaultAuthChecker{}
	checker.SetCacheMgr(cacheMgn)

	freeIndex := len(users) + len(groups) + 1

	t.Run("权限检查严格模式-主账户操作资源", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(users[0].Token),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(users[1].Token),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(users[1].Token),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_VerifyCredential")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken("Test_DefaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(""),
			model.WithModule(model.DiscoverModule),
			model.WithOperation(model.Create),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_VerifyCredential")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken("Test_DefaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(""),
			model.WithOperation(model.Create),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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

func Test_DefaultAuthChecker_CheckPermission_Read_NoStrict(t *testing.T) {
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
	cacheMgn, err := cache.TestCacheInitialize(ctx, cfg, storage)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		cacheMgn.Close()
	})
	_ = cacheMgn.OpenResourceCache([]cachetypes.ConfigEntry{
		{
			Name: cachetypes.UsersName,
		},
		{
			Name: cachetypes.StrategyRuleName,
		},
	}...)
	_ = cacheMgn.TestUpdate()

	checker := &defaultauth.DefaultAuthChecker{}
	checker.SetCacheMgr(cacheMgn)

	freeIndex := len(users) + len(groups) + 1

	t.Run("权限检查非严格模式-主账户正常读操作", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(users[0].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(users[1].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(users[1].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(users[1].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(""),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(""),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_VerifyCredential")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken("Test_DefaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_VerifyCredential")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken("Test_DefaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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

func Test_DefaultAuthChecker_CheckPermission_Read_Strict(t *testing.T) {
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
	cacheMgn, err := cache.TestCacheInitialize(ctx, cfg, storage)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		cacheMgn.Close()
	})
	_ = cacheMgn.OpenResourceCache([]cachetypes.ConfigEntry{
		{
			Name: cachetypes.UsersName,
		},
		{
			Name: cachetypes.StrategyRuleName,
		},
	}...)
	_ = cacheMgn.TestUpdate()

	checker := &defaultauth.DefaultAuthChecker{}
	checker.SetCacheMgr(cacheMgn)

	freeIndex := len(users) + len(groups) + 1

	t.Run("权限检查严格模式-主账户正常读操作", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(users[0].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(users[1].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(users[1].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(users[1].Token),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(""),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken(""),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_VerifyCredential")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken("Test_DefaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_VerifyCredential")
		authCtx := model.NewAcquireContext(
			model.WithRequestContext(ctx),
			model.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// model.WithToken("Test_DefaultAuthChecker_VerifyCredential"),
			model.WithOperation(model.Read),
			model.WithModule(model.DiscoverModule),
			model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{
				apisecurity.ResourceType_Services: {
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

func Test_DefaultAuthChecker_Initialize(t *testing.T) {
	reset(true)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := createMockUser(10)

	storage := storemock.NewMockStore(ctrl)
	storage.EXPECT().GetUnixSecond(gomock.Any()).AnyTimes().Return(time.Now().Unix(), nil)
	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(users, nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return([]*model.UserGroupDetail{}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cacheMgn, err := cache.TestCacheInitialize(ctx, &cache.Config{}, storage)
	if err != nil {
		t.Fatal(err)
	}

	_ = cacheMgn.OpenResourceCache([]cachetypes.ConfigEntry{
		{
			Name: cachetypes.UsersName,
		},
	}...)
	t.Cleanup(func() {
		cancel()
		cacheMgn.Close()
	})

	t.Run("使用未迁移至auth.user.option及auth.strategy.option的配置", func(t *testing.T) {
		reset(true)
		authChecker := &defaultauth.DefaultAuthChecker{}
		cfg := &auth.Config{}
		cfg.SetDefault()
		cfg.Name = ""
		cfg.Option = map[string]interface{}{
			"consoleOpen": true,
			"clientOpen":  true,
			"salt":        "polarismesh@2021",
			"strict":      false,
		}
		err := authChecker.Initialize(cfg, storage, cacheMgn)
		assert.NoError(t, err)
		assert.Equal(t, &defaultauth.AuthConfig{
			ConsoleOpen:   true,
			ClientOpen:    true,
			Salt:          "polarismesh@2021",
			Strict:        false,
			ConsoleStrict: true,
			ClientStrict:  false,
		}, defaultauth.AuthOption)
	})

	t.Run("使用完全迁移至auth.user.option及auth.strategy.option的配置", func(t *testing.T) {
		reset(true)
		authChecker := &defaultauth.DefaultAuthChecker{}

		cfg := &auth.Config{}
		cfg.SetDefault()
		cfg.User = &auth.UserConfig{
			Name:   "",
			Option: map[string]interface{}{"salt": "polarismesh@2021"},
		}
		cfg.Strategy = &auth.StrategyConfig{
			Name: "",
			Option: map[string]interface{}{
				"consoleOpen": true,
				"clientOpen":  true,
				"strict":      false,
			},
		}

		err := authChecker.Initialize(cfg, storage, cacheMgn)
		assert.NoError(t, err)
		assert.Equal(t, &defaultauth.AuthConfig{
			ConsoleOpen:   true,
			ClientOpen:    true,
			Salt:          "polarismesh@2021",
			Strict:        false,
			ConsoleStrict: true,
		}, defaultauth.AuthOption)
	})

	t.Run("使用部分迁移至auth.user.option及auth.strategy.option的配置（应当报错）", func(t *testing.T) {
		reset(true)
		authChecker := &defaultauth.DefaultAuthChecker{}
		cfg := &auth.Config{}
		cfg.SetDefault()
		cfg.Name = ""
		cfg.Option = map[string]interface{}{
			"clientOpen": true,
			"strict":     false,
		}
		cfg.User = &auth.UserConfig{
			Name:   "",
			Option: map[string]interface{}{"salt": "polarismesh@2021"},
		}
		cfg.Strategy = &auth.StrategyConfig{
			Name: "",
			Option: map[string]interface{}{
				"consoleOpen": true,
			},
		}

		err := authChecker.Initialize(cfg, storage, cacheMgn)
		assert.NoError(t, err)
	})

}
