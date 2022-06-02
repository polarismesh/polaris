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

package boltdb

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"

	"github.com/stretchr/testify/assert"
)

func createTestStrategy(num int) []*model.StrategyDetail {
	ret := make([]*model.StrategyDetail, 0, num)

	for i := 0; i < num; i++ {
		ret = append(ret, &model.StrategyDetail{
			ID:      fmt.Sprintf("strategy-%d", i),
			Name:    fmt.Sprintf("strategy-%d", i),
			Action:  api.AuthAction_READ_WRITE.String(),
			Comment: fmt.Sprintf("strategy-%d", i),
			Principals: []model.Principal{
				{
					StrategyID:    fmt.Sprintf("strategy-%d", i),
					PrincipalID:   fmt.Sprintf("user-%d", i),
					PrincipalRole: model.PrincipalUser,
				},
			},
			Default: true,
			Owner:   "polaris",
			Resources: []model.StrategyResource{
				{
					StrategyID: "",
					ResType:    int32(api.ResourceType_Namespaces),
					ResID:      fmt.Sprintf("namespace_%d", i),
				},
			},
			Valid:      false,
			Revision:   utils.NewUUID(),
			CreateTime: time.Now(),
			ModifyTime: time.Now(),
		})
	}

	return ret
}

func Test_strategyStore_AddStrategy(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_strategy", func(t *testing.T, handler BoltHandler) {
		ss := &strategyStore{handler: handler}

		rules := createTestStrategy(1)
		err := ss.AddStrategy(rules[0])

		assert.Nil(t, err, "add strategy must success")
	})
}

func Test_strategyStore_UpdateStrategy(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_strategy", func(t *testing.T, handler BoltHandler) {
		ss := &strategyStore{handler: handler}

		rules := createTestStrategy(1)
		err := ss.AddStrategy(rules[0])
		assert.Nil(t, err, "add strategy must success")

		addPrincipals := []model.Principal{{
			StrategyID:    rules[0].ID,
			PrincipalID:   utils.NewUUID(),
			PrincipalRole: model.PrincipalGroup,
		}}

		req := &model.ModifyStrategyDetail{
			ID:               rules[0].ID,
			Name:             rules[0].Name,
			Action:           rules[0].Action,
			Comment:          "update-strategy",
			AddPrincipals:    addPrincipals,
			RemovePrincipals: []model.Principal{},
			AddResources: []model.StrategyResource{
				{
					StrategyID: rules[0].ID,
					ResType:    int32(api.ResourceType_Services),
					ResID:      utils.NewUUID(),
				},
			},
			RemoveResources: []model.StrategyResource{},
			ModifyTime:      time.Time{},
		}

		err = ss.UpdateStrategy(req)
		assert.Nil(t, err, "update strategy must success")

		v, err := ss.GetStrategyDetail(rules[0].ID)
		assert.Nil(t, err, "update strategy must success")
		assert.Equal(t, req.Comment, v.Comment, "comment")
		assert.ElementsMatch(t, append(rules[0].Principals, addPrincipals...), v.Principals, "principals")
	})
}

func Test_strategyStore_DeleteStrategy(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_strategy", func(t *testing.T, handler BoltHandler) {
		ss := &strategyStore{handler: handler}

		rules := createTestStrategy(1)
		err := ss.AddStrategy(rules[0])
		assert.Nil(t, err, "add strategy must success")

		err = ss.DeleteStrategy(rules[0].ID)
		assert.Nil(t, err, "delete strategy must success")

		ret, err := ss.GetStrategyDetail(rules[0].ID)
		assert.Nil(t, err, "get strategy must success")
		assert.Nil(t, ret, "get strategy ret must nil")
	})
}

func Test_strategyStore_RemoveStrategyResources(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_strategy", func(t *testing.T, handler BoltHandler) {
		ss := &strategyStore{handler: handler}

		rules := createTestStrategy(1)
		err := ss.AddStrategy(rules[0])
		assert.Nil(t, err, "add strategy must success")

		err = ss.RemoveStrategyResources([]model.StrategyResource{
			{
				StrategyID: rules[0].ID,
				ResType:    int32(api.ResourceType_Namespaces),
				ResID:      "namespace_0",
			},
		})
		assert.Nil(t, err, "RemoveStrategyResources must success")
		ret, err := ss.GetStrategyDetail(rules[0].ID)
		assert.Nil(t, err, "get strategy must success")

		for i := range ret.Resources {
			res := ret.Resources[i]
			t.Logf("resource=%#v", res)
			assert.NotEqual(t, res, model.StrategyResource{
				StrategyID: rules[0].ID,
				ResType:    int32(api.ResourceType_Namespaces),
				ResID:      "namespace_0",
			})
		}
	})
}

func Test_strategyStore_LooseAddStrategyResources(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_strategy", func(t *testing.T, handler BoltHandler) {
		ss := &strategyStore{handler: handler}

		rules := createTestStrategy(1)
		err := ss.AddStrategy(rules[0])
		assert.Nil(t, err, "add strategy must success")

		err = ss.LooseAddStrategyResources([]model.StrategyResource{
			{
				StrategyID: rules[0].ID,
				ResType:    int32(api.ResourceType_Namespaces),
				ResID:      "namespace_1",
			},
		})
		assert.Nil(t, err, "RemoveStrategyResources must success")
		ret, err := ss.GetStrategyDetail(rules[0].ID)
		assert.Nil(t, err, "get strategy must success")

		ans := make([]model.StrategyResource, 0)
		for i := range ret.Resources {
			res := ret.Resources[i]
			t.Logf("resource=%#v", res)
			res.StrategyID = rules[0].ID
			if reflect.DeepEqual(res, model.StrategyResource{
				StrategyID: rules[0].ID,
				ResType:    int32(api.ResourceType_Namespaces),
				ResID:      "namespace_1",
			}) {
				ans = append(ans, res)
			}
		}

		assert.Equal(t, 1, len(ans))
	})
}

func Test_strategyStore_GetStrategyDetail(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_strategy", func(t *testing.T, handler BoltHandler) {
		ss := &strategyStore{handler: handler}

		rules := createTestStrategy(1)
		err := ss.AddStrategy(rules[0])
		assert.Nil(t, err, "add strategy must success")

		v, err := ss.GetStrategyDetail(rules[0].ID)
		assert.Nil(t, err, "get strategy-detail must success")

		rules[0].ModifyTime = rules[0].CreateTime
		v.CreateTime = rules[0].CreateTime
		v.ModifyTime = rules[0].CreateTime

		assert.Equal(t, rules[0], v)
	})
}

func Test_strategyStore_GetStrategyResources(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_strategy", func(t *testing.T, handler BoltHandler) {
		ss := &strategyStore{handler: handler}

		rules := createTestStrategy(2)
		for i := range rules {
			rule := rules[i]
			err := ss.AddStrategy(rule)
			assert.Nil(t, err, "add strategy must success")
		}

		res, err := ss.GetStrategyResources("user-1", model.PrincipalUser)
		assert.Nil(t, err, "GetStrategyResources must success")

		assert.ElementsMatch(t, []model.StrategyResource{
			{
				StrategyID: "strategy-1",
				ResType:    int32(api.ResourceType_Namespaces),
				ResID:      "namespace_1",
			},
		}, res)
	})
}

func Test_strategyStore_GetDefaultStrategyDetailByPrincipal(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_strategy", func(t *testing.T, handler BoltHandler) {
		ss := &strategyStore{handler: handler}

		rules := createTestStrategy(2)
		for i := range rules {
			rule := rules[i]
			rule.Default = i == 1
			rules[i] = rule
			err := ss.AddStrategy(rule)
			assert.Nil(t, err, "add strategy must success")
		}

		res, err := ss.GetDefaultStrategyDetailByPrincipal("user-1", model.PrincipalUser)
		assert.Nil(t, err, "GetStrategyResources must success")

		rules[1].ModifyTime = rules[1].CreateTime
		res.CreateTime = rules[1].CreateTime
		res.ModifyTime = rules[1].CreateTime
		assert.Equal(t, rules[1], res)
	})
}
