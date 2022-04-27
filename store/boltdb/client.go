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
	"time"

	"github.com/polarismesh/polaris-server/common/model"
)

type clientStore struct {
	handler BoltHandler
}

// BatchAddClients insert the client info
func (cs *clientStore) BatchAddClients(clients []*model.Client) error {
	return nil
}

// BatchDeleteClients delete the client info
func (cs *clientStore) BatchDeleteClients(ids []interface{}) error {
	return nil
}

// GetMoreClients 根据mtime获取增量clients，返回所有store的变更信息
func (cs *clientStore) GetMoreClients(mtime time.Time, firstUpdate bool) (map[string]*model.Client, error) {
	return nil, nil
}
