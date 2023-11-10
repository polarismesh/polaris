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

package service

import (
	"github.com/polarismesh/polaris/cache"
	"github.com/polarismesh/polaris/common/model"
)

// DiscoverServer Server discovered by the service
type DiscoverServer interface {
	// DiscoverServerV1 DiscoverServerV1
	DiscoverServerV1
	// ServiceAliasOperateServer Service alias operation interface definition
	ServiceAliasOperateServer
	// ServiceOperateServer Service operation interface definition
	ServiceOperateServer
	// InstanceOperateServer Instance Operation Interface Definition
	InstanceOperateServer
	// ClientServer Client operation interface definition
	ClientServer
	// Cache Get cache management
	Cache() *cache.CacheManager
	// L5OperateServer L5 related operations
	L5OperateServer
	// 
	// GetServiceInstanceRevision Get the version of the service
	GetServiceInstanceRevision(serviceID string, instances []*model.Instance) (string, error)
}
