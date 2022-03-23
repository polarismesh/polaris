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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris-server/common/model"
)

func createTestStrategy(num int) []*model.StrategyDetail {
	ret := make([]*model.StrategyDetail, 0, num)

	for i := 0; i < num; i++ {
		ret = append(ret, &model.StrategyDetail{
			ID:         "",
			Name:       "",
			Action:     "",
			Comment:    "",
			Principals: []model.Principal{},
			Default:    false,
			Owner:      "",
			Resources:  []model.StrategyResource{},
			Valid:      false,
			Revision:   "",
			CreateTime: time.Now(),
			ModifyTime: time.Now(),
		})
	}

	return ret
}

func Test_strategyStore_AddStrategy(t *testing.T) {
	CreateTableDBHandlerAndRun(t, "test_strategy", func(t *testing.T, handler BoltHandler) {
		ss := &strategyStore{handler: handler}

		assert.NotNil(t, ss)
	})
}
