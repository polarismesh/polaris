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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
)

//
func Test_strategyCache_IsResourceEditable_1(t *testing.T) {
	t.Run("资源没有关联任何策略", func(t *testing.T) {
		userCache := &userCache{}
		userCache.initBuckets()
		strategyCache := &strategyCache{
			userCache: userCache,
		}
		strategyCache.initBuckets()

		strategyCache.setStrategys(buildStrategies(10))

		ret := strategyCache.IsResourceEditable(model.Principal{
			PrincipalID:   "user-1",
			PrincipalRole: model.PrincipalUser,
		}, api.ResourceType_Namespaces, "namespace-1")

		assert.True(t, ret, "must be true")
	})

	t.Run("操作的目标资源关联了策略-自己在principal-user列表中", func(t *testing.T) {
		userCache := &userCache{}
		userCache.initBuckets()
		strategyCache := &strategyCache{
			userCache: userCache,
		}
		strategyCache.initBuckets()

		strategyCache.setStrategys([]*model.StrategyDetail{
			{
				ID:   fmt.Sprintf("rule-%d", 1),
				Name: fmt.Sprintf("rule-%d", 1),
				Principals: []model.Principal{
					{
						PrincipalID:   "user-1",
						PrincipalRole: model.PrincipalUser,
					},
				},
				Valid: true,
				Resources: []model.StrategyResource{
					{
						StrategyID: fmt.Sprintf("rule-%d", 1),
						ResType:    0,
						ResID:      "*",
					},
				},
			},
		})

		ret := strategyCache.IsResourceEditable(model.Principal{
			PrincipalID:   "user-1",
			PrincipalRole: model.PrincipalUser,
		}, api.ResourceType_Namespaces, "namespace-1")

		assert.True(t, ret, "must be true")
	})

	t.Run("操作的目标资源关联了策略-自己不在principal-user列表中", func(t *testing.T) {
		userCache := &userCache{}
		userCache.initBuckets()
		strategyCache := &strategyCache{
			userCache: userCache,
		}
		strategyCache.initBuckets()
		strategyCache.setStrategys(buildStrategies(10))

		ret := strategyCache.IsResourceEditable(model.Principal{
			PrincipalID:   "user-20",
			PrincipalRole: model.PrincipalUser,
		}, api.ResourceType_Namespaces, "namespace-1")

		assert.False(t, ret, "must be false")
	})

	t.Run("操作的目标资源关联了策略-自己属于principal-group中组成员", func(t *testing.T) {
		userCache := &userCache{}
		userCache.initBuckets()
		strategyCache := &strategyCache{
			userCache: userCache,
		}
		strategyCache.initBuckets()

		userCache.groups.save("group-1", &model.UserGroupDetail{
			UserGroup: &model.UserGroup{
				ID: "group-1",
			},
			UserIds: map[string]struct{}{
				"user-1": {},
			},
		})

		userCache.user2Groups.save("user-1")
		links, _ := userCache.user2Groups.get("user-1")
		links.save("group-1")

		strategyCache.setStrategys(buildStrategies(10))

		ret := strategyCache.IsResourceEditable(model.Principal{
			PrincipalID:   "user-1",
			PrincipalRole: model.PrincipalUser,
		}, api.ResourceType_Namespaces, "namespace-1")

		assert.True(t, ret, "must be true")
	})

	t.Run("操作关联策略的资源-策略在操作成功-策略移除操作失败", func(t *testing.T) {
		userCache := &userCache{}
		userCache.initBuckets()
		strategyCache := &strategyCache{
			userCache: userCache,
		}
		strategyCache.initBuckets()

		userCache.groups.save("group-1", &model.UserGroupDetail{
			UserGroup: &model.UserGroup{
				ID: "group-1",
			},
			UserIds: map[string]struct{}{
				"user-1": {},
			},
		})

		userCache.user2Groups.save("user-1")
		links, _ := userCache.user2Groups.get("user-1")
		links.save("group-1")
		strategyCache.strategys.save("rule-1", &model.StrategyDetailCache{
			StrategyDetail: &model.StrategyDetail{
				ID:         "rule-1",
				Name:       "rule-1",
				Principals: []model.Principal{},
				Resources:  []model.StrategyResource{},
			},
			GroupPrincipal: map[string]model.Principal{
				"group-1": {
					PrincipalID: "group-1",
				},
			},
		})
		strategyCache.strategys.save("rule-2", &model.StrategyDetailCache{
			StrategyDetail: &model.StrategyDetail{
				ID:         "rule-2",
				Name:       "rule-2",
				Principals: []model.Principal{},
				Resources:  []model.StrategyResource{},
			},
			GroupPrincipal: map[string]model.Principal{
				"group-2": {
					PrincipalID: "group-2",
				},
			},
		})

		strategyCache.namespace2Strategy.save("namespace-1", "rule-1")
		strategyCache.namespace2Strategy.save("namespace-1", "rule-2")

		ret := strategyCache.IsResourceEditable(model.Principal{
			PrincipalID:   "user-1",
			PrincipalRole: model.PrincipalUser,
		}, api.ResourceType_Namespaces, "namespace-1")

		assert.True(t, ret, "must be true")

		strategyCache.handlerResourceStrategy([]*model.StrategyDetail{
			{
				ID:         "rule-1",
				Name:       "rule-1",
				Valid:      false,
				Principals: []model.Principal{},
				Resources: []model.StrategyResource{
					{
						StrategyID: "rule-1",
						ResType:    0,
						ResID:      "namespace-1",
					},
				},
			},
		})

		ret = strategyCache.IsResourceEditable(model.Principal{
			PrincipalID:   "user-1",
			PrincipalRole: model.PrincipalUser,
		}, api.ResourceType_Namespaces, "namespace-1")

		assert.False(t, ret, "must be false")
	})

	t.Run("", func(t *testing.T) {
		userCache := &userCache{}
		userCache.initBuckets()
		strategyCache := &strategyCache{
			userCache: userCache,
		}
		strategyCache.initBuckets()

		userCache.groups.save("group-1", &model.UserGroupDetail{
			UserGroup: &model.UserGroup{
				ID: "group-1",
			},
			UserIds: map[string]struct{}{
				"user-1": {},
			},
		})

		strategyDetail := &model.StrategyDetail{
			ID:   "rule-1",
			Name: "rule-1",
			Principals: []model.Principal{
				{
					PrincipalID:   "user-1",
					PrincipalRole: model.PrincipalUser,
				},
				{
					PrincipalID:   "group-1",
					PrincipalRole: model.PrincipalGroup,
				},
			},
			Valid: true,
			Resources: []model.StrategyResource{
				{
					StrategyID: "rule-1",
					ResType:    0,
					ResID:      "*",
				},
			},
		}

		strategyDetail2 := &model.StrategyDetail{
			ID:   "rule-2",
			Name: "rule-2",
			Principals: []model.Principal{
				{
					PrincipalID:   "user-2",
					PrincipalRole: model.PrincipalUser,
				},
				{
					PrincipalID:   "group-2",
					PrincipalRole: model.PrincipalGroup,
				},
			},
			Valid: true,
			Resources: []model.StrategyResource{
				{
					StrategyID: "rule-2",
					ResType:    0,
					ResID:      "namespace-1",
				},
			},
		}

		strategyCache.strategys.save("rule-1", &model.StrategyDetailCache{
			StrategyDetail: strategyDetail,
			UserPrincipal: map[string]model.Principal{
				"user-1": {
					PrincipalID: "user-1",
				},
			},
			GroupPrincipal: map[string]model.Principal{
				"group-1": {
					PrincipalID: "group-1",
				},
			},
		})
		strategyCache.strategys.save("rule-2", &model.StrategyDetailCache{
			StrategyDetail: strategyDetail2,
			UserPrincipal: map[string]model.Principal{
				"user-2": {
					PrincipalID: "user-2",
				},
			},
			GroupPrincipal: map[string]model.Principal{
				"group-2": {
					PrincipalID: "group-2",
				},
			},
		})

		strategyCache.handlerPrincipalStrategy([]*model.StrategyDetail{strategyDetail2})
		strategyCache.handlerResourceStrategy([]*model.StrategyDetail{strategyDetail2})
		strategyCache.handlerPrincipalStrategy([]*model.StrategyDetail{strategyDetail})
		strategyCache.handlerResourceStrategy([]*model.StrategyDetail{strategyDetail})
		ret := strategyCache.IsResourceEditable(model.Principal{
			PrincipalID:   "user-1",
			PrincipalRole: model.PrincipalUser,
		}, api.ResourceType_Namespaces, "namespace-1")

		assert.True(t, ret, "must be true")

		strategyDetail.Valid = false

		strategyCache.handlerPrincipalStrategy([]*model.StrategyDetail{strategyDetail})
		strategyCache.handlerResourceStrategy([]*model.StrategyDetail{strategyDetail})
		strategyCache.strategys.delete(strategyDetail.ID)
		ret = strategyCache.IsResourceEditable(model.Principal{
			PrincipalID:   "user-1",
			PrincipalRole: model.PrincipalUser,
		}, api.ResourceType_Namespaces, "namespace-1")

		assert.False(t, ret, "must be false")
	})
}

func buildStrategies(num int) []*model.StrategyDetail {

	ret := make([]*model.StrategyDetail, 0, num)

	for i := 0; i < num; i++ {
		principals := make([]model.Principal, 0, num)
		for j := 0; j < num; j++ {
			principals = append(principals, model.Principal{
				PrincipalID:   fmt.Sprintf("user-%d", i+1),
				PrincipalRole: model.PrincipalUser,
			}, model.Principal{
				PrincipalID:   fmt.Sprintf("group-%d", i+1),
				PrincipalRole: model.PrincipalGroup,
			})
		}

		ret = append(ret, &model.StrategyDetail{
			ID:         fmt.Sprintf("rule-%d", i+1),
			Name:       fmt.Sprintf("rule-%d", i+1),
			Principals: principals,
			Valid:      true,
			Resources: []model.StrategyResource{
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
			},
		})
	}

	return ret
}

func testBuildPrincipalMap(principals []model.Principal, role model.PrincipalType) map[string]model.Principal {
	ret := make(map[string]model.Principal, 0)
	for i := range principals {
		principal := principals[i]
		if principal.PrincipalRole == role {
			ret[principal.PrincipalID] = principal
		}
	}

	return ret
}
