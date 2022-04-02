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
			ID:         fmt.Sprintf("strategy-%d", i),
			Name:       fmt.Sprintf("strategy-%d", i),
			Action:     api.AuthAction_READ_WRITE.String(),
			Comment:    fmt.Sprintf("strategy-%d", i),
			Principals: []model.Principal{},
			Default:    true,
			Owner:      "polaris",
			Resources:  []model.StrategyResource{},
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

		v, err := ss.GetStrategyDetail(rules[0].ID, rules[0].Default)
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
	})
}

func Test_strategyStore_RemoveStrategyResources(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_strategy", func(t *testing.T, handler BoltHandler) {
		ss := &strategyStore{handler: handler}

		rules := createTestStrategy(1)
		err := ss.AddStrategy(rules[0])
		assert.Nil(t, err, "add strategy must success")

		err = ss.DeleteStrategy(rules[0].ID)
		assert.Nil(t, err, "delete strategy must success")
	})
}

func Test_strategyStore_LooseAddStrategyResources(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_strategy", func(t *testing.T, handler BoltHandler) {
		ss := &strategyStore{handler: handler}

		rules := createTestStrategy(1)
		err := ss.AddStrategy(rules[0])
		assert.Nil(t, err, "add strategy must success")

		err = ss.DeleteStrategy(rules[0].ID)
		assert.Nil(t, err, "delete strategy must success")
	})
}

func Test_strategyStore_GetStrategyDetail(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_strategy", func(t *testing.T, handler BoltHandler) {
		ss := &strategyStore{handler: handler}

		rules := createTestStrategy(1)
		err := ss.AddStrategy(rules[0])
		assert.Nil(t, err, "add strategy must success")

		err = ss.DeleteStrategy(rules[0].ID)
		assert.Nil(t, err, "delete strategy must success")
	})
}

func Test_strategyStore_GetStrategyResources(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_strategy", func(t *testing.T, handler BoltHandler) {
		ss := &strategyStore{handler: handler}

		rules := createTestStrategy(1)
		err := ss.AddStrategy(rules[0])
		assert.Nil(t, err, "add strategy must success")

		err = ss.DeleteStrategy(rules[0].ID)
		assert.Nil(t, err, "delete strategy must success")
	})
}

func Test_strategyStore_GetDefaultStrategyDetailByPrincipal(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_strategy", func(t *testing.T, handler BoltHandler) {
		ss := &strategyStore{handler: handler}

		rules := createTestStrategy(1)
		err := ss.AddStrategy(rules[0])
		assert.Nil(t, err, "add strategy must success")

		err = ss.DeleteStrategy(rules[0].ID)
		assert.Nil(t, err, "delete strategy must success")
	})
}
