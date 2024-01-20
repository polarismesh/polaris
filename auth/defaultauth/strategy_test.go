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
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/auth/defaultauth"
	"github.com/polarismesh/polaris/cache"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	storemock "github.com/polarismesh/polaris/store/mock"
)

type StrategyTest struct {
	admin    *model.User
	ownerOne *model.User
	ownerTwo *model.User

	namespaces        []*model.Namespace
	services          []*model.Service
	strategies        []*model.StrategyDetail
	allStrategies     []*model.StrategyDetail
	defaultStrategies []*model.StrategyDetail

	users  []*model.User
	groups []*model.UserGroupDetail

	storage  *storemock.MockStore
	cacheMgn *cache.CacheManager
	checker  auth.AuthChecker

	svr *defaultauth.StrategyAuthAbility

	cancel context.CancelFunc

	ctrl *gomock.Controller
}

func newStrategyTest(t *testing.T) *StrategyTest {
	reset(false)

	ctrl := gomock.NewController(t)

	users := createMockUser(10)
	groups := createMockUserGroup(users)

	namespaces := createMockNamespace(len(users)+len(groups)+10, users[0].ID)
	services := createMockService(namespaces)
	serviceMap := convertServiceSliceToMap(services)
	defaultStrategies, strategies := createMockStrategy(users, groups, services[:len(users)+len(groups)])

	allStrategies := make([]*model.StrategyDetail, 0, len(defaultStrategies)+len(strategies))
	allStrategies = append(allStrategies, defaultStrategies...)
	allStrategies = append(allStrategies, strategies...)

	cfg, storage := initCache(ctrl)

	storage.EXPECT().GetServicesCount().AnyTimes().Return(uint32(1), nil)
	storage.EXPECT().GetUnixSecond(gomock.Any()).AnyTimes().Return(time.Now().Unix(), nil)
	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(users, nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(groups, nil)
	storage.EXPECT().GetStrategyDetailsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(allStrategies, nil)
	storage.EXPECT().GetMoreNamespaces(gomock.Any()).AnyTimes().Return(namespaces, nil)
	storage.EXPECT().GetMoreServices(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(serviceMap, nil)
	storage.EXPECT().GetStrategyResources(gomock.Eq(users[1].ID), gomock.Any()).AnyTimes().Return(strategies[1].Resources, nil)
	storage.EXPECT().GetStrategyResources(gomock.Eq(groups[1].ID), gomock.Any()).AnyTimes().Return(strategies[len(users)-1+2].Resources, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cacheMgn, err := cache.TestCacheInitialize(ctx, cfg, storage)
	if err != nil {
		t.Fatal(err)
	}
	_ = cacheMgn.OpenResourceCache([]cachetypes.ConfigEntry{
		{
			Name: cachetypes.ServiceName,
			Option: map[string]interface{}{
				"disableBusiness": false,
				"needMeta":        true,
			},
		},
		{
			Name: cachetypes.InstanceName,
		},
		{
			Name: cachetypes.UsersName,
		},
		{
			Name: cachetypes.StrategyRuleName,
		},
		{
			Name: cachetypes.NamespaceName,
		},
	}...)

	_ = cacheMgn.TestUpdate()

	checker := &defaultauth.DefaultAuthChecker{}
	checker.Initialize(&auth.Config{
		User: &auth.UserConfig{
			Name: "",
			Option: map[string]interface{}{
				"salt": "polarismesh@2021",
			},
		},
		Strategy: &auth.StrategyConfig{
			Name: "",
			Option: map[string]interface{}{
				"consoleOpen": true,
				"clientOpen":  true,
				"strict":      false,
			},
		},
	}, storage, cacheMgn)
	checker.SetCacheMgr(cacheMgn)

	t.Cleanup(func() {
		cacheMgn.Close()
	})

	svr := defaultauth.NewStrategyAuthAbility(
		checker,
		defaultauth.NewServer(storage, nil, cacheMgn, checker),
	)

	return &StrategyTest{
		ownerOne: users[0],

		users:  users,
		groups: groups,

		namespaces:        namespaces,
		services:          services,
		strategies:        strategies,
		allStrategies:     allStrategies,
		defaultStrategies: defaultStrategies,

		storage:  storage,
		cacheMgn: cacheMgn,
		checker:  checker,
		svr:      svr,

		cancel: cancel,

		ctrl: ctrl,
	}
}

func (g *StrategyTest) Clean() {
	g.cancel()
	_ = g.cacheMgn.Close()
}

func Test_GetPrincipalResources(t *testing.T) {

	strategyTest := newStrategyTest(t)
	defer strategyTest.Clean()

	valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)

	ret := strategyTest.svr.GetPrincipalResources(valCtx, map[string]string{
		"principal_id":   strategyTest.users[1].ID,
		"principal_type": "user",
	})

	t.Logf("GetPrincipalResources resp : %+v", ret)
	assert.EqualValues(t, api.ExecuteSuccess, ret.Code.GetValue(), "need query success")
	resources := ret.Resources
	assert.Equal(t, 2, len(resources.Services), "need query 2 service resources")
}

func Test_CreateStrategy(t *testing.T) {

	strategyTest := newStrategyTest(t)
	defer strategyTest.Clean()

	t.Run("正常创建鉴权策略", func(t *testing.T) {
		strategyTest.storage.EXPECT().AddStrategy(gomock.Any()).Return(nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		strategyId := utils.NewUUID()

		resp := strategyTest.svr.CreateStrategy(valCtx, &apisecurity.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyId},
			Name: &wrapperspb.StringValue{
				Value: "正常创建鉴权策略",
			},
			Principals: &apisecurity.Principals{
				Users: []*apisecurity.Principal{{
					Id: &wrapperspb.StringValue{
						Value: strategyTest.users[1].ID,
					},
					Name: &wrapperspb.StringValue{
						Value: strategyTest.users[1].Name,
					},
				}},
				Groups: []*apisecurity.Principal{},
			},
			Resources: &apisecurity.StrategyResources{
				StrategyId: &wrapperspb.StringValue{
					Value: strategyId,
				},
				Namespaces:   []*apisecurity.StrategyResourceEntry{},
				Services:     []*apisecurity.StrategyResourceEntry{},
				ConfigGroups: []*apisecurity.StrategyResourceEntry{},
			},
			Action: 0,
		})

		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("创建鉴权策略-非owner用户发起", func(t *testing.T) {
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)
		strategyId := utils.NewUUID()

		resp := strategyTest.svr.CreateStrategy(valCtx, &apisecurity.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyId},
			Name: &wrapperspb.StringValue{
				Value: "创建鉴权策略-非owner用户发起",
			},
			Principals: &apisecurity.Principals{
				Users: []*apisecurity.Principal{{
					Id: &wrapperspb.StringValue{
						Value: strategyTest.users[1].ID,
					},
					Name: &wrapperspb.StringValue{
						Value: strategyTest.users[1].Name,
					},
				}},
				Groups: []*apisecurity.Principal{},
			},
			Resources: &apisecurity.StrategyResources{
				StrategyId: &wrapperspb.StringValue{
					Value: strategyId,
				},
				Namespaces:   []*apisecurity.StrategyResourceEntry{},
				Services:     []*apisecurity.StrategyResourceEntry{},
				ConfigGroups: []*apisecurity.StrategyResourceEntry{},
			},
			Action: 0,
		})

		assert.Equal(t, api.OperationRoleException, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("创建鉴权策略-关联用户不存在", func(t *testing.T) {
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		strategyId := utils.NewUUID()

		resp := strategyTest.svr.CreateStrategy(valCtx, &apisecurity.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyId},
			Name: &wrapperspb.StringValue{
				Value: "创建鉴权策略-关联用户不存在",
			},
			Principals: &apisecurity.Principals{
				Users: []*apisecurity.Principal{{
					Id: &wrapperspb.StringValue{
						Value: utils.NewUUID(),
					},
					Name: &wrapperspb.StringValue{
						Value: "user-1",
					},
				}},
				Groups: []*apisecurity.Principal{},
			},
			Resources: &apisecurity.StrategyResources{
				StrategyId: &wrapperspb.StringValue{
					Value: strategyId,
				},
				Namespaces:   []*apisecurity.StrategyResourceEntry{},
				Services:     []*apisecurity.StrategyResourceEntry{},
				ConfigGroups: []*apisecurity.StrategyResourceEntry{},
			},
			Action: 0,
		})

		assert.Equal(t, api.NotFoundUser, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("创建鉴权策略-关联用户组不存在", func(t *testing.T) {
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		strategyId := utils.NewUUID()

		resp := strategyTest.svr.CreateStrategy(valCtx, &apisecurity.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyId},
			Name: &wrapperspb.StringValue{
				Value: "创建鉴权策略-关联用户组不存在",
			},
			Principals: &apisecurity.Principals{
				Groups: []*apisecurity.Principal{{
					Id: &wrapperspb.StringValue{
						Value: utils.NewUUID(),
					},
					Name: &wrapperspb.StringValue{
						Value: "user-1",
					},
				}},
			},
			Resources: &apisecurity.StrategyResources{
				StrategyId: &wrapperspb.StringValue{
					Value: strategyId,
				},
				Namespaces:   []*apisecurity.StrategyResourceEntry{},
				Services:     []*apisecurity.StrategyResourceEntry{},
				ConfigGroups: []*apisecurity.StrategyResourceEntry{},
			},
			Action: 0,
		})

		assert.Equal(t, api.NotFoundUserGroup, resp.Code.GetValue(), resp.Info.GetValue())
	})

}

func Test_UpdateStrategy(t *testing.T) {
	strategyTest := newStrategyTest(t)
	defer strategyTest.Clean()

	t.Run("正常更新鉴权策略", func(t *testing.T) {
		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[0], nil)
		strategyTest.storage.EXPECT().UpdateStrategy(gomock.Any()).Return(nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		strategyId := strategyTest.strategies[0].ID

		resp := strategyTest.svr.UpdateStrategies(valCtx, []*apisecurity.ModifyAuthStrategy{
			{
				Id: &wrapperspb.StringValue{
					Value: strategyId,
				},
				Name: &wrapperspb.StringValue{
					Value: strategyTest.strategies[0].Name,
				},
				AddPrincipals: &apisecurity.Principals{
					Users: []*apisecurity.Principal{
						{
							Id: &wrapperspb.StringValue{Value: strategyTest.users[2].ID},
						},
					},
				},
				RemovePrincipals: &apisecurity.Principals{
					Users: []*apisecurity.Principal{
						{
							Id: &wrapperspb.StringValue{Value: strategyTest.users[3].ID},
						},
					},
				},
				AddResources: &apisecurity.StrategyResources{
					StrategyId: &wrapperspb.StringValue{
						Value: strategyId,
					},
					Namespaces: []*apisecurity.StrategyResourceEntry{
						{Id: &wrapperspb.StringValue{Value: strategyTest.namespaces[0].Name}},
					},
					Services: []*apisecurity.StrategyResourceEntry{
						{Id: &wrapperspb.StringValue{Value: strategyTest.services[0].ID}},
					},
					ConfigGroups: []*apisecurity.StrategyResourceEntry{},
				},
				RemoveResources: &apisecurity.StrategyResources{
					StrategyId: &wrapperspb.StringValue{
						Value: strategyId,
					},
					Namespaces: []*apisecurity.StrategyResourceEntry{
						{Id: &wrapperspb.StringValue{Value: strategyTest.namespaces[1].Name}},
					},
					Services: []*apisecurity.StrategyResourceEntry{
						{Id: &wrapperspb.StringValue{Value: strategyTest.services[1].ID}},
					},
					ConfigGroups: []*apisecurity.StrategyResourceEntry{},
				},
			},
		})

		assert.Equal(t, api.ExecuteSuccess, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("更新鉴权策略-非owner用户发起", func(t *testing.T) {
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)
		strategyId := utils.NewUUID()

		resp := strategyTest.svr.UpdateStrategies(valCtx, []*apisecurity.ModifyAuthStrategy{
			{
				Id: &wrapperspb.StringValue{
					Value: strategyId,
				},
				Name: &wrapperspb.StringValue{
					Value: strategyTest.strategies[0].Name,
				},
				AddPrincipals: &apisecurity.Principals{
					Users:  []*apisecurity.Principal{},
					Groups: []*apisecurity.Principal{},
				},
				RemovePrincipals: &apisecurity.Principals{
					Users:  []*apisecurity.Principal{},
					Groups: []*apisecurity.Principal{},
				},
				AddResources: &apisecurity.StrategyResources{
					StrategyId: &wrapperspb.StringValue{
						Value: "",
					},
					Namespaces:   []*apisecurity.StrategyResourceEntry{},
					Services:     []*apisecurity.StrategyResourceEntry{},
					ConfigGroups: []*apisecurity.StrategyResourceEntry{},
				},
				RemoveResources: &apisecurity.StrategyResources{
					StrategyId: &wrapperspb.StringValue{
						Value: "",
					},
					Namespaces:   []*apisecurity.StrategyResourceEntry{},
					Services:     []*apisecurity.StrategyResourceEntry{},
					ConfigGroups: []*apisecurity.StrategyResourceEntry{},
				},
			},
		})

		assert.Equal(t, api.OperationRoleException, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("更新鉴权策略-目标策略不存在", func(t *testing.T) {

		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(nil, nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)

		strategyId := strategyTest.defaultStrategies[0].ID

		resp := strategyTest.svr.UpdateStrategies(valCtx, []*apisecurity.ModifyAuthStrategy{
			{
				Id: &wrapperspb.StringValue{Value: strategyId},
				AddPrincipals: &apisecurity.Principals{
					Users: []*apisecurity.Principal{
						{
							Id: &wrapperspb.StringValue{Value: utils.NewUUID()},
						},
					},
				},
			},
		})

		assert.Equal(t, api.NotFoundAuthStrategyRule, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("更新鉴权策略-owner不为自己", func(t *testing.T) {
		oldOwner := strategyTest.strategies[2].Owner

		defer func() {
			strategyTest.strategies[2].Owner = oldOwner
		}()

		strategyTest.strategies[2].Owner = strategyTest.users[2].ID
		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[2], nil)
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		strategyId := strategyTest.strategies[2].ID
		resp := strategyTest.svr.UpdateStrategies(valCtx, []*apisecurity.ModifyAuthStrategy{
			{
				Id: &wrapperspb.StringValue{Value: strategyId},
				AddPrincipals: &apisecurity.Principals{
					Users: []*apisecurity.Principal{
						{
							Id: &wrapperspb.StringValue{Value: utils.NewUUID()},
						},
					},
				},
			},
		})

		assert.Equal(t, api.NotAllowedAccess, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("更新鉴权策略-关联用户不存在", func(t *testing.T) {

		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[0], nil)
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		strategyId := strategyTest.strategies[0].ID
		resp := strategyTest.svr.UpdateStrategies(valCtx, []*apisecurity.ModifyAuthStrategy{
			{
				Id: &wrapperspb.StringValue{Value: strategyId},
				AddPrincipals: &apisecurity.Principals{
					Users: []*apisecurity.Principal{
						{
							Id: &wrapperspb.StringValue{Value: utils.NewUUID()},
						},
					},
				},
			},
		})

		assert.Equal(t, api.NotFoundUser, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("更新鉴权策略-关联用户组不存在", func(t *testing.T) {

		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[0], nil)
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		strategyId := strategyTest.strategies[0].ID
		resp := strategyTest.svr.UpdateStrategies(valCtx, []*apisecurity.ModifyAuthStrategy{
			{
				Id: &wrapperspb.StringValue{Value: strategyId},
				AddPrincipals: &apisecurity.Principals{
					Groups: []*apisecurity.Principal{
						{
							Id: &wrapperspb.StringValue{Value: utils.NewUUID()},
						},
					},
				},
			},
		})

		assert.Equal(t, api.NotFoundUserGroup, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("更新默认鉴权策略-不能更改principals成员", func(t *testing.T) {

		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.defaultStrategies[0], nil)
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		strategyId := strategyTest.defaultStrategies[0].ID
		resp := strategyTest.svr.UpdateStrategies(valCtx, []*apisecurity.ModifyAuthStrategy{
			{
				Id: &wrapperspb.StringValue{Value: strategyId},
				AddPrincipals: &apisecurity.Principals{
					Users: []*apisecurity.Principal{
						{
							Id: &wrapperspb.StringValue{Value: strategyTest.users[3].ID},
						},
					},
				},
			},
		})

		assert.Equal(t, api.NotAllowModifyDefaultStrategyPrincipal, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

}

func Test_DeleteStrategy(t *testing.T) {
	strategyTest := newStrategyTest(t)
	defer strategyTest.Clean()

	t.Run("正常删除鉴权策略", func(t *testing.T) {

		index := rand.Intn(len(strategyTest.strategies))

		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[index], nil)
		strategyTest.storage.EXPECT().DeleteStrategy(gomock.Any()).Return(nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)

		resp := strategyTest.svr.DeleteStrategies(valCtx, []*apisecurity.AuthStrategy{
			{Id: &wrapperspb.StringValue{Value: strategyTest.strategies[index].ID}},
		})

		assert.Equal(t, api.ExecuteSuccess, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("删除鉴权策略-非owner用户发起", func(t *testing.T) {
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)

		resp := strategyTest.svr.DeleteStrategies(valCtx, []*apisecurity.AuthStrategy{
			{Id: &wrapperspb.StringValue{Value: strategyTest.strategies[rand.Intn(len(strategyTest.strategies))].ID}},
		})

		assert.Equal(t, api.OperationRoleException, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("删除鉴权策略-目标策略不存在", func(t *testing.T) {

		index := rand.Intn(len(strategyTest.strategies))
		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(nil, nil)
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		resp := strategyTest.svr.DeleteStrategies(valCtx, []*apisecurity.AuthStrategy{
			{Id: &wrapperspb.StringValue{Value: strategyTest.strategies[index].ID}},
		})

		assert.Equal(t, api.ExecuteSuccess, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("删除鉴权策略-目标为默认鉴权策略", func(t *testing.T) {
		index := rand.Intn(len(strategyTest.defaultStrategies))

		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.defaultStrategies[index], nil)
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		resp := strategyTest.svr.DeleteStrategies(valCtx, []*apisecurity.AuthStrategy{
			{Id: &wrapperspb.StringValue{Value: strategyTest.defaultStrategies[index].ID}},
		})

		assert.Equal(t, api.BadRequest, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("删除鉴权策略-目标owner不为自己", func(t *testing.T) {
		index := rand.Intn(len(strategyTest.defaultStrategies))
		oldOwner := strategyTest.strategies[index].Owner

		defer func() {
			strategyTest.strategies[index].Owner = oldOwner
		}()

		strategyTest.strategies[index].Owner = strategyTest.users[2].ID
		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[index], nil)
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		resp := strategyTest.svr.DeleteStrategies(valCtx, []*apisecurity.AuthStrategy{
			{Id: &wrapperspb.StringValue{Value: strategyTest.strategies[index].ID}},
		})

		assert.Equal(t, api.NotAllowedAccess, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

}

func Test_GetStrategy(t *testing.T) {
	strategyTest := newStrategyTest(t)
	defer strategyTest.Clean()

	t.Run("正常查询鉴权策略", func(t *testing.T) {
		// 主账户查询自己的策略
		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[0], nil)
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		_ = strategyTest.cacheMgn.TestUpdate()
		resp := strategyTest.svr.GetStrategy(valCtx, &apisecurity.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyTest.strategies[0].ID},
		})
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), resp.Info.GetValue())

		// 主账户查询自己自账户的策略
		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[1], nil)
		valCtx = context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		_ = strategyTest.cacheMgn.TestUpdate()
		resp = strategyTest.svr.GetStrategy(valCtx, &apisecurity.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyTest.strategies[1].ID},
		})
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("查询鉴权策略-目标owner不为自己", func(t *testing.T) {
		t.Skip()
		var index int
		for {
			index = rand.Intn(len(strategyTest.defaultStrategies))
			if index != 2 {
				break
			}
		}
		oldOwner := strategyTest.strategies[index].Owner

		defer func() {
			strategyTest.strategies[index].Owner = oldOwner
		}()

		strategyTest.strategies[index].Owner = strategyTest.users[2].ID
		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[index], nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)

		_ = strategyTest.cacheMgn.TestUpdate()
		resp := strategyTest.svr.GetStrategy(valCtx, &apisecurity.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyTest.strategies[index].ID},
		})

		assert.Equal(t, api.NotAllowedAccess, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("查询鉴权策略-非owner用户查询自己的", func(t *testing.T) {

		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[1], nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)

		_ = strategyTest.cacheMgn.TestUpdate()
		resp := strategyTest.svr.GetStrategy(valCtx, &apisecurity.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyTest.strategies[1].ID},
		})

		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("查询鉴权策略-非owner用户查询自己所在用户组的", func(t *testing.T) {
		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[len(strategyTest.users)-1+2], nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)

		_ = strategyTest.cacheMgn.TestUpdate()
		resp := strategyTest.svr.GetStrategy(valCtx, &apisecurity.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyTest.strategies[len(strategyTest.users)-1+2].ID},
		})

		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("查询鉴权策略-非owner用户查询别人的", func(t *testing.T) {
		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[2], nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)

		_ = strategyTest.cacheMgn.TestUpdate()
		resp := strategyTest.svr.GetStrategy(valCtx, &apisecurity.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyTest.strategies[2].ID},
		})

		assert.Equal(t, api.NotAllowedAccess, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("查询鉴权策略-目标策略不存在", func(t *testing.T) {
		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(nil, nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)

		_ = strategyTest.cacheMgn.TestUpdate()
		resp := strategyTest.svr.GetStrategy(valCtx, &apisecurity.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: utils.NewUUID()},
		})

		assert.Equal(t, api.NotFoundAuthStrategyRule, resp.Code.GetValue(), resp.Info.GetValue())
	})

}

func Test_parseStrategySearchArgs(t *testing.T) {
	type args struct {
		ctx           context.Context
		searchFilters map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "res_type(namespace) 查询",
			args: args{
				ctx: func() context.Context {
					ctx := context.WithValue(context.Background(), utils.ContextOwnerIDKey, "owner")
					ctx = context.WithValue(ctx, utils.ContextUserIDKey, "user")
					ctx = context.WithValue(ctx, utils.ContextUserRoleIDKey, model.OwnerUserRole)
					ctx = context.WithValue(ctx, utils.ContextIsOwnerKey, true)
					return ctx
				}(),
				searchFilters: map[string]string{
					"res_type": "namespace",
				},
			},
			want: map[string]string{
				"res_type": "0",
				"owner":    "owner",
			},
		},
		{
			name: "res_type(service) 查询",
			args: args{
				ctx: func() context.Context {
					ctx := context.WithValue(context.Background(), utils.ContextOwnerIDKey, "owner")
					ctx = context.WithValue(ctx, utils.ContextUserIDKey, "user")
					ctx = context.WithValue(ctx, utils.ContextUserRoleIDKey, model.OwnerUserRole)
					ctx = context.WithValue(ctx, utils.ContextIsOwnerKey, true)
					return ctx
				}(),
				searchFilters: map[string]string{
					"res_type": "service",
				},
			},
			want: map[string]string{
				"res_type": "1",
				"owner":    "owner",
			},
		},
		{
			name: "principal_type(user) 查询",
			args: args{
				ctx: func() context.Context {
					ctx := context.WithValue(context.Background(), utils.ContextOwnerIDKey, "owner")
					ctx = context.WithValue(ctx, utils.ContextUserIDKey, "user")
					ctx = context.WithValue(ctx, utils.ContextUserRoleIDKey, model.SubAccountUserRole)
					ctx = context.WithValue(ctx, utils.ContextIsOwnerKey, false)
					return ctx
				}(),
				searchFilters: map[string]string{
					"principal_type": "user",
				},
			},
			want: map[string]string{
				"principal_type": "1",
				"owner":          "owner",
				"principal_id":   "user",
			},
		},
		{
			name: "principal_type(group) 查询",
			args: args{
				ctx: func() context.Context {
					ctx := context.WithValue(context.Background(), utils.ContextOwnerIDKey, "owner")
					ctx = context.WithValue(ctx, utils.ContextUserIDKey, "user")
					ctx = context.WithValue(ctx, utils.ContextUserRoleIDKey, model.OwnerUserRole)
					ctx = context.WithValue(ctx, utils.ContextIsOwnerKey, true)
					return ctx
				}(),
				searchFilters: map[string]string{
					"principal_type": "group",
				},
			},
			want: map[string]string{
				"principal_type": "2",
				"owner":          "owner",
			},
		},
		{
			name: "按照资源ID查询",
			args: args{
				ctx: func() context.Context {
					ctx := context.WithValue(context.Background(), utils.ContextOwnerIDKey, "owner")
					ctx = context.WithValue(ctx, utils.ContextUserIDKey, "user")
					ctx = context.WithValue(ctx, utils.ContextUserRoleIDKey, model.OwnerUserRole)
					ctx = context.WithValue(ctx, utils.ContextIsOwnerKey, true)
					return ctx
				}(),
				searchFilters: map[string]string{
					"res_id": "res_id",
				},
			},
			want: map[string]string{
				"res_id": "res_id",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultauth.TestParseStrategySearchArgs(tt.args.ctx, tt.args.searchFilters); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseStrategySearchArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_AuthServer_NormalOperateStrategy(t *testing.T) {
	suit := &AuthTestSuit{}
	if err := suit.Initialize(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		suit.cleanAllAuthStrategy()
		suit.cleanAllUser()
		suit.cleanAllUserGroup()
		suit.Destroy()
	})

	users := createApiMockUser(10, "test")

	t.Run("正常创建用户", func(t *testing.T) {
		resp := suit.UserServer().CreateUsers(suit.DefaultCtx, users)

		if !respSuccess(resp) {
			t.Fatal(resp.GetInfo().GetValue())
		}
	})

	t.Run("正常更新用户", func(t *testing.T) {
		users[0].Comment = utils.NewStringValue("update user comment")
		resp := suit.UserServer().UpdateUser(suit.DefaultCtx, users[0])

		if !respSuccess(resp) {
			t.Fatal(resp.GetInfo().GetValue())
		}

		qresp := suit.UserServer().GetUsers(suit.DefaultCtx, map[string]string{
			"id": users[0].GetId().GetValue(),
		})

		if !respSuccess(resp) {
			t.Fatal(resp.GetInfo().GetValue())
		}

		assert.Equal(t, 1, int(qresp.Amount.GetValue()))
		assert.Equal(t, 1, int(qresp.Size.GetValue()))

		retUsers := qresp.GetUsers()[0]
		assert.Equal(t, users[0].GetComment().GetValue(), retUsers.GetComment().GetValue())
	})

	t.Run("正常删除用户", func(t *testing.T) {
		resp := suit.UserServer().DeleteUsers(suit.DefaultCtx, []*apisecurity.User{users[3]})

		if !respSuccess(resp) {
			t.Fatal(resp.GetInfo().GetValue())
		}

		qresp := suit.UserServer().GetUsers(suit.DefaultCtx, map[string]string{
			"id": users[3].GetId().GetValue(),
		})

		if !respSuccess(resp) {
			t.Fatal(resp.GetInfo().GetValue())
		}

		assert.Equal(t, 0, int(qresp.Amount.GetValue()))
		assert.Equal(t, 0, int(qresp.Size.GetValue()))
	})

	t.Run("正常更新用户Token", func(t *testing.T) {
		resp := suit.UserServer().ResetUserToken(suit.DefaultCtx, users[0])

		if !respSuccess(resp) {
			t.Fatal(resp.GetInfo().GetValue())
		}

		_ = suit.CacheMgr().TestUpdate()

		qresp := suit.UserServer().GetUserToken(suit.DefaultCtx, users[0])
		if !respSuccess(qresp) {
			t.Fatal(resp.GetInfo().GetValue())
		}
		assert.Equal(t, resp.GetUser().GetAuthToken().GetValue(), qresp.GetUser().GetAuthToken().GetValue())
	})
}
