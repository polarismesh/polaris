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

package sqldb

import "github.com/polarismesh/polaris/store"

// maintainStore implement MaintainStore interface
type maintainStore struct {
	master *BaseDB
}

// BatchCleanDeletedInstances batch clean soft deleted instances
func (maintain *maintainStore) BatchCleanDeletedInstances(batchSize uint32) (uint32, error) {
	log.Infof("[Store][database] batch clean soft deleted instances(%d)", batchSize)
	mainStr := "delete from instance where flag = 1 limit ?"
	result, err := maintain.master.Exec(mainStr, batchSize)
	if err != nil {
		log.Errorf("[Store][database] batch clean soft deleted instances(%d), err: %s", batchSize, err.Error())
		return 0, store.Error(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Warnf("[Store][database] batch clean soft deleted instances(%d), get RowsAffected err: %s", batchSize, err.Error())
		return 0, store.Error(err)
	}

	return uint32(rows), nil
}
