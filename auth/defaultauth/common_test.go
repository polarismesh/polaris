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
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	storemock "github.com/polarismesh/polaris-server/store/mock"
	"golang.org/x/crypto/bcrypt"
)

func reset(strict bool) {
	initDefaultAuth()
	AuthOption = DefaultAuthConfig()
	AuthOption.ClientOpen = true
	AuthOption.ConsoleOpen = true
	AuthOption.Strict = strict
}

func initDefaultAuth() {
	// 设置好默认的鉴权策略插件
	plugin.SetPluginConfig(&plugin.Config{
		Auth: plugin.ConfigEntry{
			Name: "defaultAuth",
		},
	})
}

func initCache(ctrl *gomock.Controller) (*cache.Config, *storemock.MockStore) {
	/*
		- name: service # 加载服务数据
		  option:
			disableBusiness: false # 不加载业务服务
			needMeta: true # 加载服务元数据
		- name: instance # 加载实例数据
		  option:
			disableBusiness: false # 不加载业务服务实例
			needMeta: true # 加载实例元数据
		- name: routingConfig # 加载路由数据
		- name: rateLimitConfig # 加载限流数据
		- name: circuitBreakerConfig # 加载熔断数据
		- name: l5 # 加载l5数据
		- name: users
		- name: strategyRule
		- name: namespace
	*/
	cfg := &cache.Config{
		Open: true,
		Resources: []cache.ConfigEntry{
			{
				Name: "service",
				Option: map[string]interface{}{
					"disableBusiness": false,
					"needMeta":        true,
				},
			},
			{
				Name: "users",
			},
			{
				Name: "strategyRule",
			},
			{
				Name: "namespace",
			},
		},
	}

	storage := storemock.NewMockStore(ctrl)

	storage.EXPECT().GetUnixSecond().AnyTimes().Return(time.Now().Unix(), nil)

	return cfg, storage
}

func createMockNamespace(total int, owner string) []*model.Namespace {
	namespaces := make([]*model.Namespace, 0, total)

	for i := 0; i < total; i++ {
		namespaces = append(namespaces, &model.Namespace{
			Name:  fmt.Sprintf("namespace_%d", i),
			Owner: owner,
			Valid: true,
		})
	}

	return namespaces
}

func createMockService(namespaces []*model.Namespace) []*model.Service {
	services := make([]*model.Service, 0, len(namespaces))

	for i := 0; i < len(namespaces); i++ {
		ns := namespaces[i]
		services = append(services, &model.Service{
			ID:        utils.NewUUID(),
			Namespace: ns.Name,
			Owner:     ns.Owner,
			Name:      fmt.Sprintf("service_%d", i),
			Valid:     true,
		})
	}

	return services
}

// createMockUser 默认 users[0] 为 owner 用户
func createMockUser(total int, prefix ...string) []*model.User {
	users := make([]*model.User, 0, total)

	ownerId := utils.NewUUID()

	nameTemp := "user-%d"
	if len(prefix) != 0 {
		nameTemp = prefix[0] + nameTemp
	}

	for i := 0; i < total; i++ {
		id := fmt.Sprintf("fake-user-id-%d-%s", i, utils.NewUUID())
		if i == 0 {
			id = ownerId
		}
		pwd, _ := bcrypt.GenerateFromPassword([]byte("polaris"), bcrypt.DefaultCost)
		token, _ := createToken(id, "")
		users = append(users, &model.User{
			ID:       id,
			Name:     fmt.Sprintf(nameTemp, i),
			Password: string(pwd),
			Owner: func() string {
				if id == ownerId {
					return ""
				}
				return ownerId
			}(),
			Source: "Polaris",
			Mobile: "",
			Email:  "",
			Type: func() model.UserRoleType {
				if id == ownerId {
					return model.OwnerUserRole
				}
				return model.SubAccountUserRole
			}(),
			Token:       token,
			TokenEnable: true,
			Valid:       true,
			CreateTime:  time.Time{},
			ModifyTime:  time.Time{},
		})
	}
	return users
}

func createMockUserGroup(users []*model.User) []*model.UserGroupDetail {
	groups := make([]*model.UserGroupDetail, 0, len(users))

	for i := range users {
		user := users[i]
		id := utils.NewUUID()

		token, _ := createToken("", id)

		groups = append(groups, &model.UserGroupDetail{
			UserGroup: &model.UserGroup{
				ID:          id,
				Name:        fmt.Sprintf("group-%d", i),
				Owner:       users[0].ID,
				Token:       token,
				TokenEnable: true,
				Valid:       true,
				Comment:     "",
				CreateTime:  time.Time{},
				ModifyTime:  time.Time{},
			},
			UserIds: map[string]struct{}{
				user.ID: {},
			},
		})
	}

	return groups
}

func createMockStrategy(users []*model.User, groups []*model.UserGroupDetail, services []*model.Service) ([]*model.StrategyDetail, []*model.StrategyDetail) {
	strategies := make([]*model.StrategyDetail, 0, len(users)+len(groups))
	defaultStrategies := make([]*model.StrategyDetail, 0, len(users)+len(groups))

	owner := ""
	for i := 0; i < len(users); i++ {
		user := users[i]
		if user.Owner == "" {
			owner = user.ID
			break
		}
	}

	for i := 0; i < len(users); i++ {
		user := users[i]
		service := services[i]
		id := utils.NewUUID()
		strategies = append(strategies, &model.StrategyDetail{
			ID:      id,
			Name:    fmt.Sprintf("strategy_user_%s_%d", user.Name, i),
			Action:  api.AuthAction_READ_WRITE.String(),
			Comment: "",
			Principals: []model.Principal{
				{
					PrincipalID:   user.ID,
					PrincipalRole: model.PrincipalUser,
				},
			},
			Default: false,
			Owner:   owner,
			Resources: []model.StrategyResource{
				{
					StrategyID: id,
					ResType:    int32(api.ResourceType_Namespaces),
					ResID:      service.Namespace,
				},
				{
					StrategyID: id,
					ResType:    int32(api.ResourceType_Services),
					ResID:      service.ID,
				},
			},
			Valid:      true,
			Revision:   utils.NewUUID(),
			CreateTime: time.Time{},
			ModifyTime: time.Time{},
		})

		defaultStrategies = append(defaultStrategies, &model.StrategyDetail{
			ID:      id,
			Name:    fmt.Sprintf("strategy_default_user_%s_%d", user.Name, i),
			Action:  api.AuthAction_READ_WRITE.String(),
			Comment: "",
			Principals: []model.Principal{
				{
					PrincipalID:   user.ID,
					PrincipalRole: model.PrincipalUser,
				},
			},
			Default: true,
			Owner:   owner,
			Resources: []model.StrategyResource{
				{
					StrategyID: id,
					ResType:    int32(api.ResourceType_Namespaces),
					ResID:      service.Namespace,
				},
				{
					StrategyID: id,
					ResType:    int32(api.ResourceType_Services),
					ResID:      service.ID,
				},
			},
			Valid:      true,
			Revision:   utils.NewUUID(),
			CreateTime: time.Time{},
			ModifyTime: time.Time{},
		})
	}

	for i := 0; i < len(groups); i++ {
		group := groups[i]
		service := services[len(users)+i]
		id := utils.NewUUID()
		strategies = append(strategies, &model.StrategyDetail{
			ID:      id,
			Name:    fmt.Sprintf("strategy_group_%s_%d", group.Name, i),
			Action:  api.AuthAction_READ_WRITE.String(),
			Comment: "",
			Principals: []model.Principal{
				{
					PrincipalID:   group.ID,
					PrincipalRole: model.PrincipalGroup,
				},
			},
			Default: false,
			Owner:   owner,
			Resources: []model.StrategyResource{
				{
					StrategyID: id,
					ResType:    int32(api.ResourceType_Namespaces),
					ResID:      service.Namespace,
				},
				{
					StrategyID: id,
					ResType:    int32(api.ResourceType_Services),
					ResID:      service.ID,
				},
			},
			Valid:      true,
			Revision:   utils.NewUUID(),
			CreateTime: time.Time{},
			ModifyTime: time.Time{},
		})

		defaultStrategies = append(defaultStrategies, &model.StrategyDetail{
			ID:      id,
			Name:    fmt.Sprintf("strategy_default_group_%s_%d", group.Name, i),
			Action:  api.AuthAction_READ_WRITE.String(),
			Comment: "",
			Principals: []model.Principal{
				{
					PrincipalID:   group.ID,
					PrincipalRole: model.PrincipalGroup,
				},
			},
			Default: true,
			Owner:   owner,
			Resources: []model.StrategyResource{
				{
					StrategyID: id,
					ResType:    int32(api.ResourceType_Namespaces),
					ResID:      service.Namespace,
				},
				{
					StrategyID: id,
					ResType:    int32(api.ResourceType_Services),
					ResID:      service.ID,
				},
			},
			Valid:      true,
			Revision:   utils.NewUUID(),
			CreateTime: time.Time{},
			ModifyTime: time.Time{},
		})
	}

	return defaultStrategies, strategies
}

func convertServiceSliceToMap(services []*model.Service) map[string]*model.Service {
	ret := make(map[string]*model.Service)

	for i := range services {
		service := services[i]
		ret[service.ID] = service
	}

	return ret
}
