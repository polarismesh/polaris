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

	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/common/model"
)

func mockConfigFileGroup(total int) []*model.ConfigFileGroup {
	ret := make([]*model.ConfigFileGroup, 0, total)

	for i := 0; i < total; i++ {
		val := &model.ConfigFileGroup{
			Name:       fmt.Sprintf("history-%d", i),
			Namespace:  "default",
			CreateTime: time.Now(),
			ModifyTime: time.Now(),
			Valid:      true,
			Metadata: map[string]string{
				"mock_data": "mock_value",
			},
		}
		ret = append(ret, val)
	}
	return ret
}

func resetConfigFileGroupTimeAndIDField(tN time.Time, restID bool, datas ...*model.ConfigFileGroup) {
	for i := range datas {
		if restID {
			datas[i].Id = 0
		}
		datas[i].CreateTime = tN
		datas[i].ModifyTime = tN
	}
}

func Test_configFileGroupStore(t *testing.T) {
	t.Run("配置分组插入", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFileGroup, func(t *testing.T, handler BoltHandler) {
			store := newConfigFileGroupStore(handler)
			total := 10
			mockGroups := mockConfigFileGroup(total)

			for i := 0; i < total; i++ {
				if _, err := store.CreateConfigFileGroup(mockGroups[i]); err != nil {
					t.Fatal(err)
				}
			}

			idMap := make(map[uint64]struct{})

			for i := 0; i < total; i++ {
				mockVal := mockGroups[i]
				val, err := store.GetConfigFileGroup(mockVal.Namespace, mockVal.Name)
				if err != nil {
					t.Fatal(err)
				}

				assert.NotNil(t, val)

				copyVal := *val
				resetConfigFileGroupTimeAndIDField(time.Now(), true, val, mockVal)
				assert.Equal(t, mockVal, val)
				idMap[copyVal.Id] = struct{}{}
			}
			assert.Equal(t, total, len(idMap))
		})
	})

	t.Run("配置分组更新", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFileGroup, func(t *testing.T, handler BoltHandler) {
			store := newConfigFileGroupStore(handler)
			total := 10
			mockGroups := mockConfigFileGroup(total)

			for i := 0; i < total; i++ {
				if _, err := store.CreateConfigFileGroup(mockGroups[i]); err != nil {
					t.Fatal(err)
				}

				mockGroups[i].Comment = fmt.Sprintf("update_group_%d", i)

				if err := store.UpdateConfigFileGroup(mockGroups[i]); err != nil {
					t.Fatal(err)
				}
			}

			for i := 0; i < total; i++ {
				mockVal := mockGroups[i]
				val, err := store.GetConfigFileGroup(mockVal.Namespace, mockVal.Name)
				if err != nil {
					t.Fatal(err)
				}

				assert.NotNil(t, val)

				resetConfigFileGroupTimeAndIDField(time.Now(), false, val, mockVal)
				assert.Equal(t, mockVal, val)
			}
		})
	})

	t.Run("配置分组插入-删除", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFileGroup, func(t *testing.T, handler BoltHandler) {
			store := newConfigFileGroupStore(handler)
			total := 10
			mockGroups := mockConfigFileGroup(total)

			for i := 0; i < total; i++ {
				if _, err := store.CreateConfigFileGroup(mockGroups[i]); err != nil {
					t.Fatal(err)
				}
			}

			for i := 0; i < total; i++ {
				if err := store.DeleteConfigFileGroup(mockGroups[i].Namespace, mockGroups[i].Name); err != nil {
					t.Fatal(err)
				}
			}

			for i := 0; i < total; i++ {
				mockVal := mockGroups[i]
				val, err := store.GetConfigFileGroup(mockVal.Namespace, mockVal.Name)
				if err != nil {
					t.Fatal(err)
				}

				assert.Nil(t, val)
			}
		})
	})
}
