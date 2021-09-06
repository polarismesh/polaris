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

type routingStore struct {
	handler BoltHandler
}

// 新增一个路由配置
func (r *routingStore) CreateRoutingConfig(conf *model.RoutingConfig) error {
	//TODO
	return nil
}

// 更新一个路由配置
func (r *routingStore) UpdateRoutingConfig(conf *model.RoutingConfig) error {
	//TODO
	return nil
}

// 删除一个路由配置
func (r *routingStore) DeleteRoutingConfig(serviceID string) error {
	//TODO
	return nil
}

// 通过mtime拉取增量的路由配置信息
func (r *routingStore) GetRoutingConfigsForCache(mtime time.Time, firstUpdate bool) ([]*model.RoutingConfig, error) {
	//TODO
	return nil, nil
}

// 根据服务名和命名空间拉取路由配置
func (r *routingStore) GetRoutingConfigWithService(name string, namespace string) (*model.RoutingConfig, error) {
	//TODO
	return nil, nil
}

// 根据服务ID拉取路由配置
func (r *routingStore) GetRoutingConfigWithID(id string) (*model.RoutingConfig, error) {
	//TODO
	return nil, nil
}

// 查询路由配置列表
func (r *routingStore) GetRoutingConfigs(
	filter map[string]string, offset uint32, limit uint32) (uint32, []*model.ExtendRoutingConfig, error) {
	//TODO
	return 0, nil, nil
}
