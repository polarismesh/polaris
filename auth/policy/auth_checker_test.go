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

package policy_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/auth/policy"
	defaultuser "github.com/polarismesh/polaris/auth/user"
	"github.com/polarismesh/polaris/cache"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

func newPolicyServer() (*policy.Server, auth.StrategyServer, error) {
	return policy.BuildServer()
}

func Test_DefaultAuthChecker_CheckConsolePermission_Write_NoStrict(t *testing.T) {
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
	storage.EXPECT().GetMoreStrategies(gomock.Any(), gomock.Any()).AnyTimes().Return(strategies, nil)
	storage.EXPECT().GetMoreNamespaces(gomock.Any()).AnyTimes().Return(namespaces, nil)
	storage.EXPECT().GetMoreServices(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(serviceMap, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cacheMgr, err := cache.TestCacheInitialize(ctx, cfg, storage)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		cacheMgr.Close()
	})

	_, proxySvr, err := defaultuser.BuildServer()
	if err != nil {
		t.Fatal(err)
	}
	proxySvr.Initialize(&auth.Config{
		User: &auth.UserConfig{
			Name: auth.DefaultUserMgnPluginName,
			Option: map[string]interface{}{
				"salt": "polarismesh@2021",
			},
		},
	}, storage, nil, cacheMgr)

	_, svr, err := newPolicyServer()
	if err != nil {
		t.Fatal(err)
	}
	if err := svr.Initialize(&auth.Config{
		Strategy: &auth.StrategyConfig{
			Name: auth.DefaultPolicyPluginName,
		},
	}, storage, cacheMgr, proxySvr); err != nil {
		t.Fatal(err)
	}
	checker := svr.GetAuthChecker()

	_ = cacheMgr.TestUpdate()

	freeIndex := len(users) + len(groups) + 1

	t.Run("权限检查非严格模式-主账户资源访问检查", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-子账户资源访问检查（无操作权限）", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查非严格模式-子账户资源访问检查（有操作权限）", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[1].ID,
						Owner: services[1].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-子账户资源访问检查（资源无绑定策略）", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-子账户访问用户组资源检查（属于用户组成员）", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[(len(users)-1)+2].ID,
						Owner: services[(len(users)-1)+2].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-子账户访问用户组资源检查（不属于用户组成员）", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[(len(users)-1)+4].ID,
						Owner: services[(len(users)-1)+4].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查非严格模式-用户组访问组内成员资源检查", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, groups[1].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken(groups[1].Token),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查非严格模式-token非法-匿名账户资源访问检查（资源无绑定策略）", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "users[1].Token")
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-token为空-匿名账户资源访问检查（资源无绑定策略）", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})
}

func Test_DefaultAuthChecker_CheckConsolePermission_Write_Strict(t *testing.T) {
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
	storage.EXPECT().GetMoreStrategies(gomock.Any(), gomock.Any()).AnyTimes().Return(strategies, nil)
	storage.EXPECT().GetMoreNamespaces(gomock.Any()).AnyTimes().Return(namespaces, nil)
	storage.EXPECT().GetMoreServices(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(serviceMap, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cacheMgr, err := cache.TestCacheInitialize(ctx, cfg, storage)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		cacheMgr.Close()
	})

	_, proxySvr, err := defaultuser.BuildServer()
	if err != nil {
		t.Fatal(err)
	}
	proxySvr.Initialize(&auth.Config{
		User: &auth.UserConfig{
			Name: auth.DefaultUserMgnPluginName,
			Option: map[string]interface{}{
				"salt": "polarismesh@2021",
			},
		},
	}, storage, nil, cacheMgr)

	_, svr, err := newPolicyServer()
	if err != nil {
		t.Fatal(err)
	}
	if err := svr.Initialize(&auth.Config{
		Strategy: &auth.StrategyConfig{
			Name: auth.DefaultPolicyPluginName,
		},
	}, storage, cacheMgr, proxySvr); err != nil {
		t.Fatal(err)
	}
	checker := svr.GetAuthChecker()

	_ = cacheMgr.TestUpdate()

	freeIndex := len(users) + len(groups) + 1

	t.Run("权限检查严格模式-主账户操作资源", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_CheckConsolePermission_Write_Strict"),
			// authcommon.WithToken(users[0].Token),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)

		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查严格模式-子账户操作资源（无操作权限）", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_CheckConsolePermission_Write_Strict"),
			// authcommon.WithToken(users[1].Token),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-子账户操作资源（有操作权限）", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_CheckConsolePermission_Write_Strict"),
			// authcommon.WithToken(users[1].Token),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[1].ID,
						Owner: services[1].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查严格模式-token非法-匿名账户操作资源（资源有策略）", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_CheckConsolePermission_Write_Strict")
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_CheckConsolePermission_Write_Strict"),
			// authcommon.WithToken("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[1].ID,
						Owner: services[1].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-token为空-匿名账户操作资源（资源有策略）", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_CheckConsolePermission_Write_Strict"),
			// authcommon.WithToken(""),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[1].ID,
						Owner: services[1].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-token非法-匿名账户操作资源（资源没有策略）", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_CheckConsolePermission_Write_Strict")
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_CheckConsolePermission_Write_Strict"),
			// authcommon.WithToken("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)

		dchecker := checker.(*policy.DefaultAuthChecker)
		oldConf := dchecker.GetConfig()
		defer func() {
			dchecker.SetConfig(oldConf)
		}()
		dchecker.SetConfig(&policy.AuthConfig{
			ConsoleOpen:   true,
			ConsoleStrict: true,
		})

		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-token为空-匿名账户操作资源（资源没有策略）", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_CheckConsolePermission_Write_Strict"),
			// authcommon.WithToken(""),
			authcommon.WithOperation(authcommon.Create),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		dchecker := checker.(*policy.DefaultAuthChecker)
		oldConf := dchecker.GetConfig()
		defer func() {
			dchecker.SetConfig(oldConf)
		}()
		dchecker.SetConfig(&policy.AuthConfig{
			ConsoleOpen:   true,
			ConsoleStrict: true,
		})

		_, err = dchecker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})
}

func Test_DefaultAuthChecker_CheckConsolePermission_Read_NoStrict(t *testing.T) {
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
	storage.EXPECT().GetMoreStrategies(gomock.Any(), gomock.Any()).AnyTimes().Return(strategies, nil)
	storage.EXPECT().GetMoreNamespaces(gomock.Any()).AnyTimes().Return(namespaces, nil)
	storage.EXPECT().GetMoreServices(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(serviceMap, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cacheMgr, err := cache.TestCacheInitialize(ctx, cfg, storage)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		cacheMgr.Close()
	})

	_, proxySvr, err := defaultuser.BuildServer()
	if err != nil {
		t.Fatal(err)
	}
	proxySvr.Initialize(&auth.Config{
		User: &auth.UserConfig{
			Name: auth.DefaultUserMgnPluginName,
			Option: map[string]interface{}{
				"salt": "polarismesh@2021",
			},
		},
	}, storage, nil, cacheMgr)

	_, svr, err := newPolicyServer()
	if err != nil {
		t.Fatal(err)
	}
	if err := svr.Initialize(&auth.Config{
		Strategy: &auth.StrategyConfig{
			Name: auth.DefaultPolicyPluginName,
		},
	}, storage, cacheMgr, proxySvr); err != nil {
		t.Fatal(err)
	}
	checker := svr.GetAuthChecker()

	_ = cacheMgr.TestUpdate()

	freeIndex := len(users) + len(groups) + 1

	t.Run("权限检查非严格模式-主账户正常读操作", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken(users[0].Token),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-子账户正常读操作-资源有权限", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken(users[1].Token),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[1].ID,
						Owner: services[1].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-子账户正常读操作-资源无权限", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken(users[1].Token),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-子账户正常读操作-资源无绑定策略", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken(users[1].Token),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-匿名账户正常读操作-token为空-资源有策略", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken(""),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-匿名账户正常读操作-token为空-资源无策略", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-匿名账户正常读操作-token非法-资源有策略", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_VerifyCredential")
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查非严格模式-匿名账户正常读操作-token非法-资源无策略", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_VerifyCredential")
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})
}

func Test_DefaultAuthChecker_CheckConsolePermission_Read_Strict(t *testing.T) {
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
	storage.EXPECT().GetMoreStrategies(gomock.Any(), gomock.Any()).AnyTimes().Return(strategies, nil)
	storage.EXPECT().GetMoreNamespaces(gomock.Any()).AnyTimes().Return(namespaces, nil)
	storage.EXPECT().GetMoreServices(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(serviceMap, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cacheMgr, err := cache.TestCacheInitialize(ctx, cfg, storage)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		cacheMgr.Close()
	})

	_, proxySvr, err := defaultuser.BuildServer()
	if err != nil {
		t.Fatal(err)
	}
	proxySvr.Initialize(&auth.Config{
		User: &auth.UserConfig{
			Name: auth.DefaultUserMgnPluginName,
			Option: map[string]interface{}{
				"salt": "polarismesh@2021",
			},
		},
	}, storage, nil, cacheMgr)

	_, svr, err := newPolicyServer()
	if err != nil {
		t.Fatal(err)
	}
	if err := svr.Initialize(&auth.Config{
		Strategy: &auth.StrategyConfig{
			Name: auth.DefaultPolicyPluginName,
		},
	}, storage, cacheMgr, proxySvr); err != nil {
		t.Fatal(err)
	}
	checker := svr.GetAuthChecker()
	dchecker := checker.(*policy.DefaultAuthChecker)
	oldConf := dchecker.GetConfig()
	defer func() {
		dchecker.SetConfig(oldConf)
	}()
	dchecker.SetConfig(&policy.AuthConfig{
		ConsoleOpen:   true,
		ConsoleStrict: true,
	})

	_ = cacheMgr.TestUpdate()

	freeIndex := len(users) + len(groups) + 1

	t.Run("权限检查严格模式-主账户正常读操作", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[0].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken(users[0].Token),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查严格模式-子账户正常读操作-资源有权限", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken(users[1].Token),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[1].ID,
						Owner: services[1].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查严格模式-子账户正常读操作-资源无权限", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken(users[1].Token),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查严格模式-子账户正常读操作-资源无绑定策略", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, users[1].Token)
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken(users[1].Token),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.NoError(t, err, "Should be verify success")
	})

	t.Run("权限检查严格模式-匿名账户正常读操作-token为空-资源有策略", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken(""),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-匿名账户正常读操作-token为空-资源无策略", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken(""),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-匿名账户正常读操作-token非法-资源有策略", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_VerifyCredential")
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[0].ID,
						Owner: services[0].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})

	t.Run("权限检查严格模式-匿名账户正常读操作-token非法-资源无策略", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "Test_DefaultAuthChecker_VerifyCredential")
		authCtx := authcommon.NewAcquireContext(
			authcommon.WithRequestContext(ctx),
			authcommon.WithMethod("Test_DefaultAuthChecker_VerifyCredential"),
			// authcommon.WithToken("Test_DefaultAuthChecker_VerifyCredential"),
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.DiscoverModule),
			authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
				apisecurity.ResourceType_Services: {
					{
						ID:    services[freeIndex].ID,
						Owner: services[freeIndex].Owner,
					},
				},
			}),
		)
		_, err = checker.CheckConsolePermission(authCtx)
		t.Logf("%+v", err)
		assert.Error(t, err, "Should be verify fail")
	})
}

func Test_DefaultAuthChecker_Initialize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("使用未迁移至auth.user.option及auth.strategy.option的配置", func(t *testing.T) {
		reset(true)
		authChecker := &policy.Server{}
		cfg := &auth.Config{}
		cfg.SetDefault()
		cfg.Name = ""
		cfg.Option = map[string]interface{}{
			"consoleOpen": true,
			"clientOpen":  true,
			"salt":        "polarismesh@2021",
			"strict":      false,
		}
		err := authChecker.ParseOptions(cfg)
		assert.NoError(t, err)
		assert.Equal(t, &policy.AuthConfig{
			ConsoleOpen:   true,
			ClientOpen:    true,
			Strict:        false,
			ConsoleStrict: false,
			ClientStrict:  false,
		}, authChecker.GetOptions())
	})

	t.Run("使用完全迁移至auth.user.option及auth.strategy.option的配置", func(t *testing.T) {
		reset(true)
		authChecker := &policy.Server{}

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

		err := authChecker.ParseOptions(cfg)
		assert.NoError(t, err)
		assert.Equal(t, &policy.AuthConfig{
			ConsoleOpen:   true,
			ConsoleStrict: false,
			ClientOpen:    true,
			Strict:        false,
		}, authChecker.GetOptions())
	})

	t.Run("使用部分迁移至auth.user.option及auth.strategy.option的配置（应当报错）", func(t *testing.T) {
		reset(true)
		authChecker := &policy.Server{}
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

		err := authChecker.ParseOptions(cfg)
		assert.NoError(t, err)
	})

}

func TestDefaultAuthChecker_isCredible(t *testing.T) {
	type fields struct {
		conf     *policy.AuthConfig
		cacheMgr cachetypes.CacheManager
		userSvr  auth.UserServer
	}
	type args struct {
		authCtx *authcommon.AcquireContext
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &policy.DefaultAuthChecker{}
			d.SetConfig(tt.fields.conf)
			if got := d.IsCredible(tt.args.authCtx); got != tt.want {
				t.Errorf("DefaultAuthChecker.isCredible() = %v, want %v", got, tt.want)
			}
		})
	}
}
