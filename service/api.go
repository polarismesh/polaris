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
	"context"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
)

// DiscoverServer Server discovered by the service
type DiscoverServer interface {
	// CircuitBreakerOperateServer Fuse rule operation interface definition
	CircuitBreakerOperateServer
	// RateLimitOperateServer Lamflow rule operation interface definition
	RateLimitOperateServer
	// RouteRuleOperateServer Routing rules operation interface definition
	RouteRuleOperateServer
	// RouterRuleOperateServer Routing rules operation interface definition
	RouterRuleOperateServer
	// FaultDetectRuleOperateServer fault detect rules operation interface definition
	FaultDetectRuleOperateServer
	// ServiceContractOperateServer service contract rules operation inerface definition
	ServiceContractOperateServer
	// ServiceAliasOperateServer Service alias operation interface definition
	ServiceAliasOperateServer
	// ServiceOperateServer Service operation interface definition
	ServiceOperateServer
	// InstanceOperateServer Instance Operation Interface Definition
	InstanceOperateServer
	// LaneOperateServer lane rule operation interface definition
	LaneOperateServer
	// ClientServer Client operation interface definition
	ClientServer
	// Cache Get cache management
	Cache() cachetypes.CacheManager
	// L5OperateServer L5 related operations
	L5OperateServer
	// GetServiceInstanceRevision Get the version of the service
	GetServiceInstanceRevision(serviceID string, instances []*model.Instance) (string, error)
}

// ResourceHook The listener is placed before and after the resource operation, only normal flow
type ResourceHook interface {

	// Before
	//  @param ctx
	//  @param resourceType
	Before(ctx context.Context, resourceType model.Resource)

	// After
	//  @param ctx
	//  @param resourceType
	//  @param res
	After(ctx context.Context, resourceType model.Resource, res *ResourceEvent) error
}

// ResourceEvent 资源事件
type ResourceEvent struct {
	Resource authcommon.ResourceEntry

	AddPrincipals []authcommon.Principal
	DelPrincipals []authcommon.Principal
	IsRemove      bool
}
