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

package boltdbStore

import (
	"github.com/polarismesh/polaris-server/common/model"
	"time"
)

type businessStore struct {
	handler BoltHandler
}

// 增加一个业务集
func (b *businessStore) AddBusiness(business *model.Business) error {
	//TODO
	return nil
}

// 删除一个业务集
func (b *businessStore) DeleteBusiness(bid string) error {
	//TODO
	return nil
}

// 更新业务集
func (b *businessStore) UpdateBusiness(business *model.Business) error {
	//TODO
	return nil
}

// 更新业务集token
func (b *businessStore) UpdateBusinessToken(bid string, token string) error {
	//TODO
	return nil
}

// 查询owner下业务集
func (b *businessStore) ListBusiness(owner string) ([]*model.Business, error) {
	//TODO
	return nil, nil
}

// 根据业务集ID获取业务集详情
func (b *businessStore) GetBusinessByID(id string) (*model.Business, error) {
	//TODO
	return nil, nil
}

// 根据mtime获取增量数据
func (b *businessStore) GetMoreBusiness(mtime time.Time) ([]*model.Business, error) {
	//TODO
	return nil, nil
}