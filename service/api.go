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

	"github.com/polarismesh/polaris-server/cache"
	"github.com/polarismesh/polaris-server/common/api/l5"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
)

// CircuitBreakerServer
type CircuitBreakerOperateServer interface {
	CreateCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse

	CreateCircuitBreakerVersions(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse

	DeleteCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse

	UpdateCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse

	ReleaseCircuitBreakers(ctx context.Context, req []*api.ConfigRelease) *api.BatchWriteResponse

	UnBindCircuitBreakers(ctx context.Context, req []*api.ConfigRelease) *api.BatchWriteResponse

	GetCircuitBreaker(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	GetCircuitBreakerVersions(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	GetMasterCircuitBreakers(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	GetReleaseCircuitBreakers(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	GetCircuitBreakerByService(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	GetCircuitBreakerToken(ctx context.Context, req *api.CircuitBreaker) *api.Response
}

type RateLimitOperateServer interface {
	CreateRateLimits(ctx context.Context, request []*api.Rule) *api.BatchWriteResponse

	DeleteRateLimits(ctx context.Context, request []*api.Rule) *api.BatchWriteResponse

	UpdateRateLimits(ctx context.Context, request []*api.Rule) *api.BatchWriteResponse

	GetRateLimits(ctx context.Context, query map[string]string) *api.BatchQueryResponse
}

type RouteRuleOperateServer interface {
	CreateRoutingConfigs(ctx context.Context, req []*api.Routing) *api.BatchWriteResponse

	DeleteRoutingConfigs(ctx context.Context, req []*api.Routing) *api.BatchWriteResponse

	UpdateRoutingConfigs(ctx context.Context, req []*api.Routing) *api.BatchWriteResponse

	GetRoutingConfigs(ctx context.Context, query map[string]string) *api.BatchQueryResponse
}

type ServiceOperateServer interface {

	// CreateServices 批量创建服务
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	CreateServices(ctx context.Context, req []*api.Service) *api.BatchWriteResponse

	// DeleteServices 批量删除服务
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	DeleteServices(ctx context.Context, req []*api.Service) *api.BatchWriteResponse

	// UpdateServices
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	UpdateServices(ctx context.Context, req []*api.Service) *api.BatchWriteResponse

	// UpdateServiceToken
	//  @param ctx
	//  @param req
	//  @return *api.Response
	UpdateServiceToken(ctx context.Context, req *api.Service) *api.Response

	// GetServices
	//  @param ctx
	//  @param query
	//  @return *api.BatchQueryResponse
	GetServices(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetServicesCount
	//  @return *api.BatchQueryResponse
	GetServicesCount() *api.BatchQueryResponse

	// GetServiceToken
	//  @param ctx
	//  @param req
	//  @return *api.Response
	GetServiceToken(ctx context.Context, req *api.Service) *api.Response

	// GetServiceOwner
	//  @param ctx
	//  @param req
	//  @return *api.BatchQueryResponse
	GetServiceOwner(ctx context.Context, req []*api.Service) *api.BatchQueryResponse
}

// ServiceAliasOperateServer
type ServiceAliasOperateServer interface {

	// CreateServiceAlias
	//  @param ctx
	//  @param req
	//  @return *api.Response
	CreateServiceAlias(ctx context.Context, req *api.ServiceAlias) *api.Response

	// DeleteServiceAliases
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	DeleteServiceAliases(ctx context.Context, req []*api.ServiceAlias) *api.BatchWriteResponse

	// UpdateServiceAlias
	//  @param ctx
	//  @param req
	//  @return *api.Response
	UpdateServiceAlias(ctx context.Context, req *api.ServiceAlias) *api.Response

	// GetServiceAliases
	//  @param query
	//  @return *api.BatchQueryResponse
	GetServiceAliases(ctx context.Context, query map[string]string) *api.BatchQueryResponse
}

// InstanceOperateServer
type InstanceOperateServer interface {

	// CreateInstances
	//  @param ctx
	//  @param reqs
	//  @return *api.BatchWriteResponse
	CreateInstances(ctx context.Context, reqs []*api.Instance) *api.BatchWriteResponse

	// CreateInstance
	//  @param ctx
	//  @param reqs
	//  @return *api.BatchWriteResponse
	CreateInstance(ctx context.Context, req *api.Instance) *api.Response

	// DeleteInstances
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	DeleteInstances(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse

	// DeleteInstancesByHost
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	DeleteInstancesByHost(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse

	// UpdateInstances
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	UpdateInstances(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse

	// UpdateInstance
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	UpdateInstance(ctx context.Context, req *api.Instance) *api.Response

	// UpdateInstancesIsolate
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	UpdateInstancesIsolate(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse

	// GetInstances
	//  @param query
	//  @return *api.BatchQueryResponse
	GetInstances(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetInstancesCount
	//  @return *api.BatchQueryResponse
	GetInstancesCount() *api.BatchQueryResponse

	// CleanInstance
	//  @param ctx
	//  @param req
	//  @return *api.Response
	CleanInstance(ctx context.Context, req *api.Instance) *api.Response
}

// NamespaceOperateServer
type NamespaceOperateServer interface {

	// CreateNamespaces
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	CreateNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse

	// DeleteNamespaces
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	DeleteNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse

	// UpdateNamespaces
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	UpdateNamespaces(ctx context.Context, req []*api.Namespace) *api.BatchWriteResponse

	// UpdateNamespaceToken
	//  @param ctx
	//  @param req
	//  @return *api.Response
	UpdateNamespaceToken(ctx context.Context, req *api.Namespace) *api.Response

	// GetNamespaces
	//  @param ctx
	//  @param query
	//  @return *api.BatchQueryResponse
	GetNamespaces(ctx context.Context, query map[string][]string) *api.BatchQueryResponse

	// GetNamespaceToken
	//  @param ctx
	//  @param req
	//  @return *api.Response
	GetNamespaceToken(ctx context.Context, req *api.Namespace) *api.Response
}

// Client operation interface definition
type ClientServer interface {

	// ReportClient
	//  @param ctx
	//  @param req
	//  @return *api.Response
	ReportClient(ctx context.Context, req *api.Client) *api.Response

	// GetServiceWithCache
	//  @param ctx
	//  @param req
	//  @return *api.DiscoverResponse
	GetServiceWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse

	// ServiceInstancesCache
	//  @param ctx
	//  @param req
	//  @return *api.DiscoverResponse
	ServiceInstancesCache(ctx context.Context, req *api.Service) *api.DiscoverResponse

	// GetRoutingConfigWithCache
	//  @param ctx
	//  @param req
	//  @return *api.DiscoverResponse
	GetRoutingConfigWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse

	// GetRateLimitWithCache
	//  @param ctx
	//  @param req
	//  @return *api.DiscoverResponse
	GetRateLimitWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse

	// GetCircuitBreakerWithCache
	//  @param ctx
	//  @param req
	//  @return *api.DiscoverResponse
	GetCircuitBreakerWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse
}

// PlatformOperateServer
type PlatformOperateServer interface {

	// CreatePlatforms
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	CreatePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse

	// UpdatePlatforms
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	UpdatePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse

	// DeletePlatforms
	//  @param ctx
	//  @param req
	//  @return *api.BatchWriteResponse
	DeletePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse

	// GetPlatforms
	//  @param query
	//  @return *api.BatchQueryResponse
	GetPlatforms(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetPlatformToken
	//  @param ctx
	//  @param req
	//  @return *api.Response
	GetPlatformToken(ctx context.Context, req *api.Platform) *api.Response
}

// L5OperateServer
type L5OperateServer interface {

	// SyncByAgentCmd
	//  @param ctx
	//  @param sbac
	//  @return *l5.Cl5SyncByAgentAckCmd
	//  @return error
	SyncByAgentCmd(ctx context.Context, sbac *l5.Cl5SyncByAgentCmd) (*l5.Cl5SyncByAgentAckCmd, error)

	// RegisterByNameCmd
	//  @param rbnc
	//  @return *l5.Cl5RegisterByNameAckCmd
	//  @return error
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

	GetServiceInstanceRevision(serviceID string, instances []*model.Instance) (string, error) 
}
