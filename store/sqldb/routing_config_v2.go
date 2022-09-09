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

	v2 "github.com/polarismesh/polaris-server/common/model/v2"
)

// RoutingConfigStoreV2 的实现
type routingConfigStoreV2 struct {
	master *BaseDB
	slave  *BaseDB
}

// CreateRoutingConfigV2 新增一个路由配置
func (r *routingConfigStoreV2) CreateRoutingConfigV2(conf *v2.RoutingConfig) error {
	return nil
}

// UpdateRoutingConfigV2 更新一个路由配置
func (r *routingConfigStoreV2) UpdateRoutingConfigV2(conf *v2.RoutingConfig) error {
	return nil
}

// DeleteRoutingConfigV2 删除一个路由配置
func (r *routingConfigStoreV2) DeleteRoutingConfigV2(serviceID string) error {
	return nil
}

// GetRoutingConfigsV2ForCache 通过mtime拉取增量的路由配置信息
// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
func (r *routingConfigStoreV2) GetRoutingConfigsV2ForCache(mtime time.Time, firstUpdate bool) ([]*v2.RoutingConfig, error) {
	return nil, nil
}

// GetRoutingConfigV2WithID 根据服务ID拉取路由配置
func (r *routingConfigStoreV2) GetRoutingConfigV2WithID(id string) (*v2.RoutingConfig, error) {
	return nil, nil
}

// GetRoutingConfigsV2 查询路由配置列表
func (r *routingConfigStoreV2) GetRoutingConfigsV2(filter map[string]string, offset uint32,
	limit uint32) (uint32, []*v2.RoutingConfig, error) {

	return 0, nil, nil
}
