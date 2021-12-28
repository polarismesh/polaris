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

package naming

import (
	"context"

	"github.com/polarismesh/polaris-server/common/api/l5"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/naming/cache"
)

// CircuitBreakerServer
type CircuitBreakerOperateServer interface {
	CreateCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse

	CreateCircuitBreaker(ctx context.Context, req *api.CircuitBreaker) *api.Response

	CreateCircuitBreakerVersions(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse

	CreateCircuitBreakerVersion(ctx context.Context, req *api.CircuitBreaker) *api.Response

	DeleteCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse

	DeleteCircuitBreaker(ctx context.Context, req *api.CircuitBreaker) *api.Response

	UpdateCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse

	UpdateCircuitBreaker(ctx context.Context, req *api.CircuitBreaker) *api.Response

	ReleaseCircuitBreakers(ctx context.Context, req []*api.ConfigRelease) *api.BatchWriteResponse

	ReleaseCircuitBreaker(ctx context.Context, req *api.ConfigRelease) *api.Response

	UnBindCircuitBreakers(ctx context.Context, req []*api.ConfigRelease) *api.BatchWriteResponse

	UnBindCircuitBreaker(ctx context.Context, req *api.ConfigRelease) *api.Response

	GetCircuitBreaker(query map[string]string) *api.BatchQueryResponse

	GetCircuitBreakerVersions(query map[string]string) *api.BatchQueryResponse

	GetMasterCircuitBreakers(query map[string]string) *api.BatchQueryResponse

	GetReleaseCircuitBreakers(query map[string]string) *api.BatchQueryResponse

	GetCircuitBreakerByService(query map[string]string) *api.BatchQueryResponse

	GetCircuitBreakerToken(ctx context.Context, req *api.CircuitBreaker) *api.Response
}

type RateLimitOperateServer interface {
	CreateRateLimits(ctx context.Context, request []*api.Rule) *api.BatchWriteResponse

	CreateRateLimit(ctx context.Context, req *api.Rule) *api.Response

	DeleteRateLimits(ctx context.Context, request []*api.Rule) *api.BatchWriteResponse

	DeleteRateLimit(ctx context.Context, req *api.Rule) *api.Response

	UpdateRateLimits(ctx context.Context, request []*api.Rule) *api.BatchWriteResponse

	UpdateRateLimit(ctx context.Context, req *api.Rule) *api.Response

	GetRateLimits(query map[string]string) *api.BatchQueryResponse
}

type RouteRuleOperateServer interface {
	CreateRoutingConfigs(ctx context.Context, req []*api.Routing) *api.BatchWriteResponse

	CreateRoutingConfig(ctx context.Context, req *api.Routing) *api.Response

	DeleteRoutingConfigs(ctx context.Context, req []*api.Routing) *api.BatchWriteResponse

	DeleteRoutingConfig(ctx context.Context, req *api.Routing) *api.Response

	UpdateRoutingConfigs(ctx context.Context, req []*api.Routing) *api.BatchWriteResponse

	UpdateRoutingConfig(ctx context.Context, req *api.Routing) *api.Response

	GetRoutingConfigs(ctx context.Context, query map[string]string) *api.BatchQueryResponse
}

type ServiceOperateServer interface {

	// CreateServices 批量创建服务
	CreateServices(ctx context.Context, req []*api.Service) *api.BatchWriteResponse

	// CreateService 创建单个服务
	CreateService(ctx context.Context, req *api.Service) *api.Response

	// DeleteServices 批量删除服务
	DeleteServices(ctx context.Context, req []*api.Service) *api.BatchWriteResponse

	// DeleteService Delete a single service, the delete operation needs to lock the service
	// 	to prevent the instance of the service associated with the service or a new operation.
	DeleteService(ctx context.Context, req *api.Service) *api.Response

	UpdateServices(ctx context.Context, req []*api.Service) *api.BatchWriteResponse

	UpdateService(ctx context.Context, req *api.Service) *api.Response

	UpdateServiceToken(ctx context.Context, req *api.Service) *api.Response

	GetServices(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	GetServicesCount() *api.BatchQueryResponse

	GetServiceToken(ctx context.Context, req *api.Service) *api.Response

	GetServiceOwner(ctx context.Context, req []*api.Service) *api.BatchQueryResponse
}

type ServiceAliasOperateServer interface {
	CreateServiceAlias(ctx context.Context, req *api.ServiceAlias) *api.Response

	DeleteServiceAlias(ctx context.Context, req *api.ServiceAlias) *api.Response

	DeleteServiceAliases(ctx context.Context, req []*api.ServiceAlias) *api.BatchWriteResponse

	UpdateServiceAlias(ctx context.Context, req *api.ServiceAlias) *api.Response

	GetServiceAliases(query map[string]string) *api.BatchQueryResponse
}

type InstanceOperateServer interface {
	CreateInstances(ctx context.Context, reqs []*api.Instance) *api.BatchWriteResponse

	CreateInstance(ctx context.Context, req *api.Instance) *api.Response

	DeleteInstances(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse

	DeleteInstance(ctx context.Context, req *api.Instance) *api.Response

	DeleteInstancesByHost(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse

	DeleteInstanceByHost(ctx context.Context, req *api.Instance) *api.Response

	UpdateInstances(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse

	UpdateInstance(ctx context.Context, req *api.Instance) *api.Response

	UpdateInstancesIsolate(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse

	UpdateInstanceIsolate(ctx context.Context, req *api.Instance) *api.Response

	GetInstances(query map[string]string) *api.BatchQueryResponse

	GetInstancesCount() *api.BatchQueryResponse

	CleanInstance(ctx context.Context, req *api.Instance) *api.Response
}

type NamespaceOperateServer interface {
	CreateNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse

	CreateNamespace(ctx context.Context, req *api.Namespace) *api.Response

	DeleteNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse

	DeleteNamespace(ctx context.Context, req *api.Namespace) *api.Response

	UpdateNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse

	UpdateNamespace(ctx context.Context, req *api.Namespace) *api.Response

	UpdateNamespaceToken(ctx context.Context, req *api.Namespace) *api.Response

	GetNamespaces(query map[string][]string) *api.BatchQueryResponse

	GetNamespaceToken(ctx context.Context, req *api.Namespace) *api.Response
}

// Client operation interface definition
type ClientServer interface {

	// ReportClient
	ReportClient(ctx context.Context, req *api.Client) *api.Response

	// GetServiceWithCache
	GetServiceWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse

	// ServiceInstancesCache
	ServiceInstancesCache(ctx context.Context, req *api.Service) *api.DiscoverResponse

	// GetRoutingConfigWithCache
	GetRoutingConfigWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse

	// GetRateLimitWithCache
	GetRateLimitWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse

	// GetCircuitBreakerWithCache
	GetCircuitBreakerWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse
}

type PlatformOperateServer interface {
	//
	CreatePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse
	//
	CreatePlatform(ctx context.Context, req *api.Platform) *api.Response
	//
	UpdatePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse
	//
	UpdatePlatform(ctx context.Context, req *api.Platform) *api.Response
	//
	DeletePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse
	//
	DeletePlatform(ctx context.Context, req *api.Platform) *api.Response
	//
	GetPlatforms(query map[string]string) *api.BatchQueryResponse
	//
	GetPlatformToken(ctx context.Context, req *api.Platform) *api.Response
}

type L5OperateServer interface {
	//
	SyncByAgentCmd(ctx context.Context, sbac *l5.Cl5SyncByAgentCmd) (*l5.Cl5SyncByAgentAckCmd, error)
	//
	RegisterByNameCmd(rbnc *l5.Cl5RegisterByNameCmd) (*l5.Cl5RegisterByNameAckCmd, error)
}

// DiscoverServer
type DiscoverServer interface {
	// Fuse rule operation interface definition
	CircuitBreakerOperateServer
	// Lamflow rule operation interface definition
	RateLimitOperateServer
	// Routing rules operation interface definition
	RouteRuleOperateServer
	// Service alias operation interface definition
	ServiceAliasOperateServer
	// Service operation interface definition
	ServiceOperateServer
	// Instance Operation Interface Definition
	InstanceOperateServer
	// Namespace Operation Interface Definition
	NamespaceOperateServer
	// Client operation interface definition
	ClientServer
	// Get cache management
	Cache() *cache.NamingCache
	//
	PlatformOperateServer
	//
	L5OperateServer
	// 
}
