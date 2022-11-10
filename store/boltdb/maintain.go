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

import "github.com/polarismesh/polaris/common/model"

type maintainStore struct {
	handler BoltHandler
}

// BatchCleanDeletedInstances
func (m *maintainStore) BatchCleanDeletedInstances(batchSize uint32) (uint32, error) {
	fields := []string{insFieldValid}
	values, err := m.handler.LoadValuesByFilter(tblNameInstance, fields, &model.Instance{},
		func(m map[string]interface{}) bool {
			valid, ok := m[insFieldValid]
			if ok && !valid.(bool) {
				return true
			}
			return false
		})
	if err != nil {
		return 0, err
	}
	if len(values) == 0 {
		return 0, nil
	}

	var count uint32 = 0
	keys := make([]string, 0, batchSize)
	for k := range values {
		keys = append(keys, k)
		count++
		if count >= batchSize {
			break
		}
	}
	err = m.handler.DeleteValues(tblNameInstance, keys)
	if err != nil {
		return count, err
	}
	return count, nil
}
