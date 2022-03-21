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
	"sync"
	"testing"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/stretchr/testify/assert"
)

//
func Test_strategyCache_IsResourceEditable_1(t *testing.T) {
	userCache := &userCache{
		users:       &sync.Map{},
		name2Users:  &sync.Map{},
		groups:      &sync.Map{},
		user2Groups: &sync.Map{},
	}
	strategyCache := &strategyCache{
		userCache:            userCache,
		strategys:            &sync.Map{},
		uid2Strategy:         &sync.Map{},
		groupid2Strategy:     &sync.Map{},
		namespace2Strategy:   &sync.Map{},
		service2Strategy:     &sync.Map{},
		configGroup2Strategy: &sync.Map{},
	}

	strategyCache.setStrategys(buildStrategies(10))

	ret := strategyCache.IsResourceEditable(model.Principal{
		PrincipalID:   "user-1",
		PrincipalRole: model.PrincipalUser,
	}, api.ResourceType_Namespaces, "namespace-1")

	assert.True(t, ret, "must be true")
}

func Test_strategyCache_IsResourceEditable_2(t *testing.T) {
	userCache := &userCache{
		users:       &sync.Map{},
		name2Users:  &sync.Map{},
		groups:      &sync.Map{},
		user2Groups: &sync.Map{},
	}
	strategyCache := &strategyCache{
		userCache:            userCache,
		strategys:            &sync.Map{},
		uid2Strategy:         &sync.Map{},
		groupid2Strategy:     &sync.Map{},
		namespace2Strategy:   &sync.Map{},
		service2Strategy:     &sync.Map{},
		configGroup2Strategy: &sync.Map{},
	}

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
}

func Test_strategyCache_IsResourceEditable_3(t *testing.T) {
	userCache := &userCache{
		users:       &sync.Map{},
		name2Users:  &sync.Map{},
		groups:      &sync.Map{},
		user2Groups: &sync.Map{},
	}
	strategyCache := &strategyCache{
		userCache:            userCache,
		strategys:            &sync.Map{},
		uid2Strategy:         &sync.Map{},
		groupid2Strategy:     &sync.Map{},
		namespace2Strategy:   &sync.Map{},
		service2Strategy:     &sync.Map{},
		configGroup2Strategy: &sync.Map{},
	}

	strategyCache.setStrategys(buildStrategies(10))

	ret := strategyCache.IsResourceEditable(model.Principal{
		PrincipalID:   "user-20",
		PrincipalRole: model.PrincipalUser,
	}, api.ResourceType_Namespaces, "namespace-1")

	assert.False(t, ret, "must be false")
}

func Test_strategyCache_IsResourceEditable_4(t *testing.T) {
	userCache := &userCache{
		users:       &sync.Map{},
		name2Users:  &sync.Map{},
		groups:      &sync.Map{},
		user2Groups: &sync.Map{},
	}
	strategyCache := &strategyCache{
		userCache:            userCache,
		strategys:            &sync.Map{},
		uid2Strategy:         &sync.Map{},
		groupid2Strategy:     &sync.Map{},
		namespace2Strategy:   &sync.Map{},
		service2Strategy:     &sync.Map{},
		configGroup2Strategy: &sync.Map{},
	}

	userCache.groups.Store("group-1", &model.UserGroupDetail{
		UserGroup: &model.UserGroup{
			ID: "group-1",
		},
		UserIds: map[string]struct{}{
			"user-1": {},
		},
	})

	links := new(sync.Map)
	links.Store("group-1", struct{}{})
	userCache.user2Groups.Store("user-1", links)

	strategyCache.setStrategys(buildStrategies(10))

	ret := strategyCache.IsResourceEditable(model.Principal{
		PrincipalID:   "user-1",
		PrincipalRole: model.PrincipalUser,
	}, api.ResourceType_Namespaces, "namespace-1")

	assert.True(t, ret, "must be true")
}

func Test_strategyCache_IsResourceEditable_5(t *testing.T) {
	userCache := &userCache{
		users:       &sync.Map{},
		name2Users:  &sync.Map{},
		groups:      &sync.Map{},
		user2Groups: &sync.Map{},
	}
	strategyCache := &strategyCache{
		userCache:            userCache,
		strategys:            &sync.Map{},
		uid2Strategy:         &sync.Map{},
		groupid2Strategy:     &sync.Map{},
		namespace2Strategy:   &sync.Map{},
		service2Strategy:     &sync.Map{},
		configGroup2Strategy: &sync.Map{},
	}

	userCache.groups.Store("group-1", &model.UserGroupDetail{
		UserGroup: &model.UserGroup{
			ID: "group-1",
		},
		UserIds: map[string]struct{}{
			"user-1": {},
		},
	})

	links := new(sync.Map)
	links.Store("group-1", struct{}{})
	userCache.user2Groups.Store("user-1", links)
	strategyCache.strategys.Store("rule-1", &model.StrategyDetailCache{
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
	strategyCache.strategys.Store("rule-2", &model.StrategyDetailCache{
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

	nsRules := &sync.Map{}
	nsRules.Store("rule-1", struct{}{})
	nsRules.Store("rule-2", struct{}{})
	strategyCache.namespace2Strategy.Store("namespace-1", nsRules)

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
}

func Test_strategyCache_IsResourceEditable_6(t *testing.T) {
	userCache := &userCache{
		users:       &sync.Map{},
		name2Users:  &sync.Map{},
		groups:      &sync.Map{},
		user2Groups: &sync.Map{},
	}
	strategyCache := &strategyCache{
		userCache:            userCache,
		strategys:            &sync.Map{},
		uid2Strategy:         &sync.Map{},
		groupid2Strategy:     &sync.Map{},
		namespace2Strategy:   &sync.Map{},
		service2Strategy:     &sync.Map{},
		configGroup2Strategy: &sync.Map{},
	}

	userCache.groups.Store("group-1", &model.UserGroupDetail{
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

	strategyCache.strategys.Store("rule-1", &model.StrategyDetailCache{
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
	strategyCache.strategys.Store("rule-2", &model.StrategyDetailCache{
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
	strategyCache.strategys.Delete(strategyDetail.ID)
	ret = strategyCache.IsResourceEditable(model.Principal{
		PrincipalID:   "user-1",
		PrincipalRole: model.PrincipalUser,
	}, api.ResourceType_Namespaces, "namespace-1")

	assert.False(t, ret, "must be false")
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

func buildPrincipalMap(principals []model.Principal, role model.PrincipalType) map[string]model.Principal {
	ret := make(map[string]model.Principal, 0)
	for i := range principals {
		principal := principals[i]
		if principal.PrincipalRole == role {
			ret[principal.PrincipalID] = principal
		}
	}

	return ret
}
