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
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	_ "github.com/polarismesh/polaris-server/plugin/auth/defaultauth"
	storemock "github.com/polarismesh/polaris-server/store/mock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
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

	svr *serverAuthAbility

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

	storage := storemock.NewMockStore(ctrl)

	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)
	storage.EXPECT().GetUsersForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(users, nil)
	storage.EXPECT().GetGroupsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(groups, nil)
	storage.EXPECT().GetStrategyDetailsForCache(gomock.Any(), gomock.Any()).AnyTimes().Return(allStrategies, nil)
	storage.EXPECT().GetMoreNamespaces(gomock.Any()).AnyTimes().Return(namespaces, nil)
	storage.EXPECT().GetMoreServices(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(serviceMap, nil)
	storage.EXPECT().GetStrategyResources(gomock.Eq(users[1].ID), gomock.Any()).AnyTimes().Return(strategies[1].Resources, nil)
	storage.EXPECT().GetStrategyResources(gomock.Eq(groups[1].ID), gomock.Any()).AnyTimes().Return(strategies[len(users)-1+2].Resources, nil)

	cfg, _ := initCache(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	if err := cache.TestCacheInitialize(ctx, cfg, storage); err != nil {
		t.Fatal(err)
	}

	cacheMgn, err := cache.GetCacheManager()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	checker := &defaultAuthChecker{}
	checker.Initialize(&auth.Config{
		Name: "",
		Option: map[string]interface{}{
			"consoleOpen": true,
			"clientOpen":  true,
			"salt":        "polarismesh@2021",
			"strict":      false,
		},
	}, cacheMgn)
	checker.cacheMgn = cacheMgn
	checker.authPlugin = plugin.GetAuth()

	svr := &serverAuthAbility{
		authMgn: checker,
		target: &server{
			storage:  storage,
			cacheMgn: cacheMgn,
			authMgn:  checker,
		},
	}

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
	_ = g.cacheMgn.Clear()
	g.ctrl.Finish()
	time.Sleep(5 * time.Second)
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

		resp := strategyTest.svr.CreateStrategy(valCtx, &api.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyId},
			Name: &wrapperspb.StringValue{
				Value: "正常创建鉴权策略",
			},
			Principals: &api.Principals{
				Users: []*api.Principal{{
					Id: &wrapperspb.StringValue{
						Value: strategyTest.users[1].ID,
					},
					Name: &wrapperspb.StringValue{
						Value: strategyTest.users[1].Name,
					},
				}},
				Groups: []*api.Principal{},
			},
			Resources: &api.StrategyResources{
				StrategyId: &wrapperspb.StringValue{
					Value: strategyId,
				},
				Namespaces:   []*api.StrategyResourceEntry{},
				Services:     []*api.StrategyResourceEntry{},
				ConfigGroups: []*api.StrategyResourceEntry{},
			},
			Action: 0,
		})

		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("创建鉴权策略-非owner用户发起", func(t *testing.T) {
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)
		strategyId := utils.NewUUID()

		resp := strategyTest.svr.CreateStrategy(valCtx, &api.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyId},
			Name: &wrapperspb.StringValue{
				Value: "创建鉴权策略-非owner用户发起",
			},
			Principals: &api.Principals{
				Users: []*api.Principal{{
					Id: &wrapperspb.StringValue{
						Value: strategyTest.users[1].ID,
					},
					Name: &wrapperspb.StringValue{
						Value: strategyTest.users[1].Name,
					},
				}},
				Groups: []*api.Principal{},
			},
			Resources: &api.StrategyResources{
				StrategyId: &wrapperspb.StringValue{
					Value: strategyId,
				},
				Namespaces:   []*api.StrategyResourceEntry{},
				Services:     []*api.StrategyResourceEntry{},
				ConfigGroups: []*api.StrategyResourceEntry{},
			},
			Action: 0,
		})

		assert.Equal(t, api.OperationRoleException, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("创建鉴权策略-关联用户不存在", func(t *testing.T) {
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		strategyId := utils.NewUUID()

		resp := strategyTest.svr.CreateStrategy(valCtx, &api.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyId},
			Name: &wrapperspb.StringValue{
				Value: "创建鉴权策略-关联用户不存在",
			},
			Principals: &api.Principals{
				Users: []*api.Principal{{
					Id: &wrapperspb.StringValue{
						Value: utils.NewUUID(),
					},
					Name: &wrapperspb.StringValue{
						Value: "user-1",
					},
				}},
				Groups: []*api.Principal{},
			},
			Resources: &api.StrategyResources{
				StrategyId: &wrapperspb.StringValue{
					Value: strategyId,
				},
				Namespaces:   []*api.StrategyResourceEntry{},
				Services:     []*api.StrategyResourceEntry{},
				ConfigGroups: []*api.StrategyResourceEntry{},
			},
			Action: 0,
		})

		assert.Equal(t, api.NotFoundUser, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("创建鉴权策略-关联用户组不存在", func(t *testing.T) {
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		strategyId := utils.NewUUID()

		resp := strategyTest.svr.CreateStrategy(valCtx, &api.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyId},
			Name: &wrapperspb.StringValue{
				Value: "创建鉴权策略-关联用户组不存在",
			},
			Principals: &api.Principals{
				Groups: []*api.Principal{{
					Id: &wrapperspb.StringValue{
						Value: utils.NewUUID(),
					},
					Name: &wrapperspb.StringValue{
						Value: "user-1",
					},
				}},
			},
			Resources: &api.StrategyResources{
				StrategyId: &wrapperspb.StringValue{
					Value: strategyId,
				},
				Namespaces:   []*api.StrategyResourceEntry{},
				Services:     []*api.StrategyResourceEntry{},
				ConfigGroups: []*api.StrategyResourceEntry{},
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

		resp := strategyTest.svr.UpdateStrategies(valCtx, []*api.ModifyAuthStrategy{
			{
				Id: &wrapperspb.StringValue{
					Value: strategyId,
				},
				Name: &wrapperspb.StringValue{
					Value: strategyTest.strategies[0].Name,
				},
				AddPrincipals: &api.Principals{
					Users: []*api.Principal{
						{
							Id: &wrapperspb.StringValue{Value: strategyTest.users[2].ID},
						},
					},
				},
				RemovePrincipals: &api.Principals{
					Users: []*api.Principal{
						{
							Id: &wrapperspb.StringValue{Value: strategyTest.users[3].ID},
						},
					},
				},
				AddResources: &api.StrategyResources{
					StrategyId: &wrapperspb.StringValue{
						Value: strategyId,
					},
					Namespaces: []*api.StrategyResourceEntry{
						{Id: &wrapperspb.StringValue{Value: strategyTest.namespaces[0].Name}},
					},
					Services: []*api.StrategyResourceEntry{
						{Id: &wrapperspb.StringValue{Value: strategyTest.services[0].ID}},
					},
					ConfigGroups: []*api.StrategyResourceEntry{},
				},
				RemoveResources: &api.StrategyResources{
					StrategyId: &wrapperspb.StringValue{
						Value: strategyId,
					},
					Namespaces: []*api.StrategyResourceEntry{
						{Id: &wrapperspb.StringValue{Value: strategyTest.namespaces[1].Name}},
					},
					Services: []*api.StrategyResourceEntry{
						{Id: &wrapperspb.StringValue{Value: strategyTest.services[1].ID}},
					},
					ConfigGroups: []*api.StrategyResourceEntry{},
				},
			},
		})

		assert.Equal(t, api.ExecuteSuccess, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("更新鉴权策略-非owner用户发起", func(t *testing.T) {
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)
		strategyId := utils.NewUUID()

		resp := strategyTest.svr.UpdateStrategies(valCtx, []*api.ModifyAuthStrategy{
			{
				Id: &wrapperspb.StringValue{
					Value: strategyId,
				},
				Name: &wrapperspb.StringValue{
					Value: strategyTest.strategies[0].Name,
				},
				AddPrincipals: &api.Principals{
					Users:  []*api.Principal{},
					Groups: []*api.Principal{},
				},
				RemovePrincipals: &api.Principals{
					Users:  []*api.Principal{},
					Groups: []*api.Principal{},
				},
				AddResources: &api.StrategyResources{
					StrategyId: &wrapperspb.StringValue{
						Value: "",
					},
					Namespaces:   []*api.StrategyResourceEntry{},
					Services:     []*api.StrategyResourceEntry{},
					ConfigGroups: []*api.StrategyResourceEntry{},
				},
				RemoveResources: &api.StrategyResources{
					StrategyId: &wrapperspb.StringValue{
						Value: "",
					},
					Namespaces:   []*api.StrategyResourceEntry{},
					Services:     []*api.StrategyResourceEntry{},
					ConfigGroups: []*api.StrategyResourceEntry{},
				},
			},
		})

		assert.Equal(t, api.OperationRoleException, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("更新鉴权策略-目标策略不存在", func(t *testing.T) {

		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(nil, nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)

		strategyId := strategyTest.defaultStrategies[0].ID

		resp := strategyTest.svr.UpdateStrategies(valCtx, []*api.ModifyAuthStrategy{
			{
				Id: &wrapperspb.StringValue{Value: strategyId},
				AddPrincipals: &api.Principals{
					Users: []*api.Principal{
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

		resp := strategyTest.svr.UpdateStrategies(valCtx, []*api.ModifyAuthStrategy{
			{
				Id: &wrapperspb.StringValue{Value: strategyId},
				AddPrincipals: &api.Principals{
					Users: []*api.Principal{
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

		resp := strategyTest.svr.UpdateStrategies(valCtx, []*api.ModifyAuthStrategy{
			{
				Id: &wrapperspb.StringValue{Value: strategyId},
				AddPrincipals: &api.Principals{
					Users: []*api.Principal{
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

		resp := strategyTest.svr.UpdateStrategies(valCtx, []*api.ModifyAuthStrategy{
			{
				Id: &wrapperspb.StringValue{Value: strategyId},
				AddPrincipals: &api.Principals{
					Groups: []*api.Principal{
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

		resp := strategyTest.svr.UpdateStrategies(valCtx, []*api.ModifyAuthStrategy{
			{
				Id: &wrapperspb.StringValue{Value: strategyId},
				AddPrincipals: &api.Principals{
					Users: []*api.Principal{
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

		resp := strategyTest.svr.DeleteStrategies(valCtx, []*api.AuthStrategy{
			{Id: &wrapperspb.StringValue{Value: strategyTest.strategies[index].ID}},
		})

		assert.Equal(t, api.ExecuteSuccess, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("删除鉴权策略-非owner用户发起", func(t *testing.T) {
		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)

		resp := strategyTest.svr.DeleteStrategies(valCtx, []*api.AuthStrategy{
			{Id: &wrapperspb.StringValue{Value: strategyTest.strategies[rand.Intn(len(strategyTest.strategies))].ID}},
		})

		assert.Equal(t, api.OperationRoleException, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("删除鉴权策略-目标策略不存在", func(t *testing.T) {

		index := rand.Intn(len(strategyTest.strategies))

		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(nil, nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)

		resp := strategyTest.svr.DeleteStrategies(valCtx, []*api.AuthStrategy{
			{Id: &wrapperspb.StringValue{Value: strategyTest.strategies[index].ID}},
		})

		assert.Equal(t, api.ExecuteSuccess, resp.Responses[0].Code.GetValue(), resp.Responses[0].Info.GetValue())
	})

	t.Run("删除鉴权策略-目标为默认鉴权策略", func(t *testing.T) {
		index := rand.Intn(len(strategyTest.defaultStrategies))

		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.defaultStrategies[index], nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)

		resp := strategyTest.svr.DeleteStrategies(valCtx, []*api.AuthStrategy{
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

		resp := strategyTest.svr.DeleteStrategies(valCtx, []*api.AuthStrategy{
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
		resp := strategyTest.svr.GetStrategy(valCtx, &api.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyTest.strategies[0].ID},
		})
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), resp.Info.GetValue())

		// 主账户查询自己自账户的策略
		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[1], nil)
		valCtx = context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)
		resp = strategyTest.svr.GetStrategy(valCtx, &api.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyTest.strategies[1].ID},
		})
		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("查询鉴权策略-目标owner不为自己", func(t *testing.T) {
		index := rand.Intn(len(strategyTest.defaultStrategies))
		oldOwner := strategyTest.strategies[index].Owner

		defer func() {
			strategyTest.strategies[index].Owner = oldOwner
		}()

		strategyTest.strategies[index].Owner = strategyTest.users[2].ID

		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[index], nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[0].Token)

		resp := strategyTest.svr.GetStrategy(valCtx, &api.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyTest.strategies[0].ID},
		})

		assert.Equal(t, api.NotAllowedAccess, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("查询鉴权策略-非owner用户查询自己的", func(t *testing.T) {

		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[1], nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)

		resp := strategyTest.svr.GetStrategy(valCtx, &api.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyTest.strategies[1].ID},
		})

		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("查询鉴权策略-非owner用户查询自己所在用户组的", func(t *testing.T) {
		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[len(strategyTest.users)-1+2], nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)

		resp := strategyTest.svr.GetStrategy(valCtx, &api.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyTest.strategies[len(strategyTest.users)-1+2].ID},
		})

		assert.Equal(t, api.ExecuteSuccess, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("查询鉴权策略-非owner用户查询别人的", func(t *testing.T) {
		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(strategyTest.strategies[2], nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)

		resp := strategyTest.svr.GetStrategy(valCtx, &api.AuthStrategy{
			Id: &wrapperspb.StringValue{Value: strategyTest.strategies[2].ID},
		})

		assert.Equal(t, api.NotAllowedAccess, resp.Code.GetValue(), resp.Info.GetValue())
	})

	t.Run("查询鉴权策略-目标策略不存在", func(t *testing.T) {
		strategyTest.storage.EXPECT().GetStrategyDetail(gomock.Any()).Return(nil, nil)

		valCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, strategyTest.users[1].Token)

		resp := strategyTest.svr.GetStrategy(valCtx, &api.AuthStrategy{
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
			if got := parseStrategySearchArgs(tt.args.ctx, tt.args.searchFilters); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseStrategySearchArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
