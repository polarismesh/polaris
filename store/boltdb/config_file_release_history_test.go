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

func mockConfigFileHistory(total int, fileName string) []*model.ConfigFileReleaseHistory {
	ret := make([]*model.ConfigFileReleaseHistory, 0, total)

	for i := 0; i < total; i++ {
		val := &model.ConfigFileReleaseHistory{
			Name:       fmt.Sprintf("history-%d", i),
			Namespace:  "default",
			Group:      "default",
			FileName:   fmt.Sprintf("history-%d", i),
			Format:     "yaml",
			Content:    fmt.Sprintf("history-%d", i),
			CreateTime: time.Now(),
			ModifyTime: time.Now(),
			Valid:      true,
			Metadata: map[string]string{
				"mock_key": "mock_value",
			},
		}

		if len(fileName) != 0 {
			val.Name = fileName
		}

		ret = append(ret, val)
	}
	return ret
}

func resetHistoryTimeAndIDField(tN time.Time, datas ...*model.ConfigFileReleaseHistory) {
	for i := range datas {
		datas[i].Id = 0
		datas[i].CreateTime = tN
		datas[i].ModifyTime = tN
	}
}

func Test_configFileReleaseHistoryStore(t *testing.T) {
	t.Run("配置发布历史插入", func(t *testing.T) {
		CreateTableDBHandlerAndRun(t, tblConfigFileReleaseHistory, func(t *testing.T, handler BoltHandler) {
			store := newConfigFileReleaseHistoryStore(handler)
			total := 10
			mockHistories := mockConfigFileHistory(total, "")

			for i := 0; i < total; i++ {
				if err := store.CreateConfigFileReleaseHistory(mockHistories[i]); err != nil {
					t.Fatal(err)
				}
			}

			idMap := make(map[uint64]struct{})

			for i := 0; i < total; i++ {
				mockVal := mockHistories[i]
				val, err := store.GetLatestConfigFileReleaseHistory(mockVal.Namespace, mockVal.Group, mockVal.FileName)
				if err != nil {
					t.Fatal(err)
				}

				assert.NotNil(t, val)

				copyVal := *val
				resetHistoryTimeAndIDField(time.Now(), val, mockVal)
				assert.Equal(t, mockVal, val)

				idMap[copyVal.Id] = struct{}{}
			}

			assert.Equal(t, total, len(idMap))
		})
	})
}
