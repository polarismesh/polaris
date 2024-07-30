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
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/wrapperspb"

	defaultuser "github.com/polarismesh/polaris/auth/user"
	"github.com/polarismesh/polaris/cache"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	storemock "github.com/polarismesh/polaris/store/mock"
)

func reset(strict bool) {

}

func initCache(ctrl *gomock.Controller) (*cache.Config, *storemock.MockStore) {
	metrics.InitMetrics()
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
	cfg := &cache.Config{}
	storage := storemock.NewMockStore(ctrl)

	mockTx := storemock.NewMockTx(ctrl)
	mockTx.EXPECT().Commit().Return(nil).AnyTimes()
	mockTx.EXPECT().Rollback().Return(nil).AnyTimes()
	mockTx.EXPECT().CreateReadView().Return(nil).AnyTimes()

	storage.EXPECT().StartReadTx().Return(mockTx, nil).AnyTimes()
	storage.EXPECT().GetServicesCount().AnyTimes().Return(uint32(1), nil)
	storage.EXPECT().GetInstancesCountTx(gomock.Any()).AnyTimes().Return(uint32(1), nil)
	storage.EXPECT().GetMoreInstances(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(map[string]*model.Instance{
		"123": {
			Proto: &service_manage.Instance{
				Id:   wrapperspb.String(uuid.NewString()),
				Host: wrapperspb.String("127.0.0.1"),
				Port: wrapperspb.UInt32(8080),
			},
			Valid: true,
		},
	}, nil).AnyTimes()
	storage.EXPECT().GetUnixSecond(gomock.Any()).AnyTimes().Return(time.Now().Unix(), nil)

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

func createMockStrategy(users []*authcommon.User, groups []*authcommon.UserGroupDetail, services []*model.Service) ([]*authcommon.StrategyDetail, []*authcommon.StrategyDetail) {
	strategies := make([]*authcommon.StrategyDetail, 0, len(users)+len(groups))
	defaultStrategies := make([]*authcommon.StrategyDetail, 0, len(users)+len(groups))

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
		strategies = append(strategies, &authcommon.StrategyDetail{
			ID:      id,
			Name:    fmt.Sprintf("strategy_user_%s_%d", user.Name, i),
			Action:  apisecurity.AuthAction_READ_WRITE.String(),
			Comment: "",
			Principals: []authcommon.Principal{
				{
					PrincipalID:   user.ID,
					PrincipalType: authcommon.PrincipalUser,
				},
			},
			Default: false,
			Owner:   owner,
			Resources: []authcommon.StrategyResource{
				{
					StrategyID: id,
					ResType:    int32(apisecurity.ResourceType_Namespaces),
					ResID:      service.Namespace,
				},
				{
					StrategyID: id,
					ResType:    int32(apisecurity.ResourceType_Services),
					ResID:      service.ID,
				},
			},
			Valid:      true,
			Revision:   utils.NewUUID(),
			CreateTime: time.Time{},
			ModifyTime: time.Time{},
		})

		defaultStrategies = append(defaultStrategies, &authcommon.StrategyDetail{
			ID:      id,
			Name:    fmt.Sprintf("strategy_default_user_%s_%d", user.Name, i),
			Action:  apisecurity.AuthAction_READ_WRITE.String(),
			Comment: "",
			Principals: []authcommon.Principal{
				{
					PrincipalID:   user.ID,
					PrincipalType: authcommon.PrincipalUser,
				},
			},
			Default: true,
			Owner:   owner,
			Resources: []authcommon.StrategyResource{
				{
					StrategyID: id,
					ResType:    int32(apisecurity.ResourceType_Namespaces),
					ResID:      service.Namespace,
				},
				{
					StrategyID: id,
					ResType:    int32(apisecurity.ResourceType_Services),
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
		strategies = append(strategies, &authcommon.StrategyDetail{
			ID:      id,
			Name:    fmt.Sprintf("strategy_group_%s_%d", group.Name, i),
			Action:  apisecurity.AuthAction_READ_WRITE.String(),
			Comment: "",
			Principals: []authcommon.Principal{
				{
					PrincipalID:   group.ID,
					PrincipalType: authcommon.PrincipalGroup,
				},
			},
			Default: false,
			Owner:   owner,
			Resources: []authcommon.StrategyResource{
				{
					StrategyID: id,
					ResType:    int32(apisecurity.ResourceType_Namespaces),
					ResID:      service.Namespace,
				},
				{
					StrategyID: id,
					ResType:    int32(apisecurity.ResourceType_Services),
					ResID:      service.ID,
				},
			},
			Valid:      true,
			Revision:   utils.NewUUID(),
			CreateTime: time.Time{},
			ModifyTime: time.Time{},
		})

		defaultStrategies = append(defaultStrategies, &authcommon.StrategyDetail{
			ID:      id,
			Name:    fmt.Sprintf("strategy_default_group_%s_%d", group.Name, i),
			Action:  apisecurity.AuthAction_READ_WRITE.String(),
			Comment: "",
			Principals: []authcommon.Principal{
				{
					PrincipalID:   group.ID,
					PrincipalType: authcommon.PrincipalGroup,
				},
			},
			Default: true,
			Owner:   owner,
			Resources: []authcommon.StrategyResource{
				{
					StrategyID: id,
					ResType:    int32(apisecurity.ResourceType_Namespaces),
					ResID:      service.Namespace,
				},
				{
					StrategyID: id,
					ResType:    int32(apisecurity.ResourceType_Services),
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

// createMockUser 默认 users[0] 为 owner 用户
func createMockUser(total int, prefix ...string) []*authcommon.User {
	users := make([]*authcommon.User, 0, total)

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
		token, _ := defaultuser.CreateToken(id, "", "polarismesh@2021")
		users = append(users, &authcommon.User{
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
			Type: func() authcommon.UserRoleType {
				if id == ownerId {
					return authcommon.OwnerUserRole
				}
				return authcommon.SubAccountUserRole
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

func createApiMockUser(total int, prefix ...string) []*apisecurity.User {
	users := make([]*apisecurity.User, 0, total)

	models := createMockUser(total, prefix...)

	for i := range models {
		users = append(users, &apisecurity.User{
			Name:     utils.NewStringValue("test-" + models[i].Name),
			Password: utils.NewStringValue("123456"),
			Source:   utils.NewStringValue("Polaris"),
			Comment:  utils.NewStringValue(models[i].Comment),
			Mobile:   utils.NewStringValue(models[i].Mobile),
			Email:    utils.NewStringValue(models[i].Email),
		})
	}

	return users
}

func createMockUserGroup(users []*authcommon.User) []*authcommon.UserGroupDetail {
	groups := make([]*authcommon.UserGroupDetail, 0, len(users))

	for i := range users {
		user := users[i]
		id := utils.NewUUID()

		token, _ := defaultuser.CreateToken("", id, "polarismesh@2021")
		groups = append(groups, &authcommon.UserGroupDetail{
			UserGroup: &authcommon.UserGroup{
				ID:          id,
				Name:        fmt.Sprintf("test-group-%d", i),
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

// createMockApiUserGroup
func createMockApiUserGroup(users []*apisecurity.User) []*apisecurity.UserGroup {
	musers := make([]*authcommon.User, 0, len(users))
	for i := range users {
		musers = append(musers, &authcommon.User{
			ID: users[i].GetId().GetValue(),
		})
	}

	models := createMockUserGroup(musers)
	ret := make([]*apisecurity.UserGroup, 0, len(models))

	for i := range models {
		ret = append(ret, &apisecurity.UserGroup{
			Name:    utils.NewStringValue(models[i].Name),
			Comment: utils.NewStringValue(models[i].Comment),
			Relation: &apisecurity.UserGroupRelation{
				Users: []*apisecurity.User{
					{
						Id: utils.NewStringValue(users[i].GetId().GetValue()),
					},
				},
			},
		})
	}

	return ret
}
