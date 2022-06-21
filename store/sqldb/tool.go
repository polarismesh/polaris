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

import (
	"time"
)

// toolStore 实现了ToolStoreStore
type toolStore struct {
	db *BaseDB
}

const (
	nowSql           = `select UNIX_TIMESTAMP(SYSDATE())`
	maxQueryInterval = time.Second
)

// GetNow 获取当前时间，单位秒
func (t *toolStore) GetUnixSecond() (int64, error) {
	startTime := time.Now()
	rows, err := t.db.Query(nowSql)
	if err != nil {
		log.Errorf("[Store][database] query now err: %s", err.Error())
		return 0, err
	}
	defer rows.Close()
	timePass := time.Since(startTime)
	if timePass > maxQueryInterval {
		log.Infof("[Store][database] query now spend %s, exceed %s, skip", timePass, maxQueryInterval)
		return 0, nil
	}
	var value int64
	for rows.Next() {
		if err := rows.Scan(&value); err != nil {
			log.Errorf("[Store][database] get now rows scan err: %s", err.Error())
			return 0, err
		}
	}
	return value, nil
}
