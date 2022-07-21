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

// CircuitBreakerOperateServer Melting rule related treatment
type CircuitBreakerOperateServer interface {

	// CreateCircuitBreakers Create a CircuitBreaker rule
	CreateCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse

	// CreateCircuitBreakerVersions Create a melt rule version
	CreateCircuitBreakerVersions(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse

	// DeleteCircuitBreakers Delete CircuitBreaker rules
	DeleteCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse

	// UpdateCircuitBreakers Modify the CircuitBreaker rule
	UpdateCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse

	// ReleaseCircuitBreakers Release CircuitBreaker rule
	ReleaseCircuitBreakers(ctx context.Context, req []*api.ConfigRelease) *api.BatchWriteResponse

	// UnBindCircuitBreakers Solution CircuitBreaker rule
	UnBindCircuitBreakers(ctx context.Context, req []*api.ConfigRelease) *api.BatchWriteResponse

	// GetCircuitBreaker Get CircuitBreaker regular according to ID and VERSION
	GetCircuitBreaker(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetCircuitBreakerVersions Query all versions of the CircuitBreaker rule
	GetCircuitBreakerVersions(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetMasterCircuitBreakers Query Master CircuitBreaker rules
	GetMasterCircuitBreakers(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetReleaseCircuitBreakers Query the released CircuitBreaker rule according to the rule ID
	GetReleaseCircuitBreakers(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetCircuitBreakerByService Binding CircuitBreaker rule based on service query
	GetCircuitBreakerByService(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetCircuitBreakerToken Get CircuitBreaker rules token
	GetCircuitBreakerToken(ctx context.Context, req *api.CircuitBreaker) *api.Response
}

// RateLimitOperateServer Lamflow rule related operation
type RateLimitOperateServer interface {

	// CreateRateLimits Create a RateLimit rule
	CreateRateLimits(ctx context.Context, request []*api.Rule) *api.BatchWriteResponse

	// DeleteRateLimits Delete current RateLimit rules
	DeleteRateLimits(ctx context.Context, request []*api.Rule) *api.BatchWriteResponse

	// UpdateRateLimits Modify the RateLimit rule
	UpdateRateLimits(ctx context.Context, request []*api.Rule) *api.BatchWriteResponse

	// GetRateLimits Query RateLimit rules
	GetRateLimits(ctx context.Context, query map[string]string) *api.BatchQueryResponse
}

// RouteRuleOperateServer Routing rules related operations
type RouteRuleOperateServer interface {

	// CreateRoutingConfigs Batch creation routing configuration
	CreateRoutingConfigs(ctx context.Context, req []*api.Routing) *api.BatchWriteResponse

	// DeleteRoutingConfigs Batch delete routing configuration
	DeleteRoutingConfigs(ctx context.Context, req []*api.Routing) *api.BatchWriteResponse

	// UpdateRoutingConfigs Batch update routing configuration
	UpdateRoutingConfigs(ctx context.Context, req []*api.Routing) *api.BatchWriteResponse

	// GetRoutingConfigs Inquiry route configuration to OSS
	GetRoutingConfigs(ctx context.Context, query map[string]string) *api.BatchQueryResponse
}

// ServiceOperateServer Service related operations
type ServiceOperateServer interface {

	// CreateServices Batch creation service
	CreateServices(ctx context.Context, req []*api.Service) *api.BatchWriteResponse

	// DeleteServices Batch delete service
	DeleteServices(ctx context.Context, req []*api.Service) *api.BatchWriteResponse

	// UpdateServices Batch update service
	UpdateServices(ctx context.Context, req []*api.Service) *api.BatchWriteResponse

	// UpdateServiceToken Update service token
	UpdateServiceToken(ctx context.Context, req *api.Service) *api.Response

	// GetServices Get a list of service
	GetServices(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetServicesCount Total number of services
	GetServicesCount(ctx context.Context) *api.BatchQueryResponse

	// GetServiceToken Get service token
	GetServiceToken(ctx context.Context, req *api.Service) *api.Response

	// GetServiceOwner Owner for obtaining service
	GetServiceOwner(ctx context.Context, req []*api.Service) *api.BatchQueryResponse
}

// ServiceAliasOperateServer Service alias related operations
type ServiceAliasOperateServer interface {

	// CreateServiceAlias Create a service alias
	CreateServiceAlias(ctx context.Context, req *api.ServiceAlias) *api.Response

	// DeleteServiceAliases Batch delete service alias
	DeleteServiceAliases(ctx context.Context, req []*api.ServiceAlias) *api.BatchWriteResponse

	// UpdateServiceAlias Update service alias
	UpdateServiceAlias(ctx context.Context, req *api.ServiceAlias) *api.Response

	// GetServiceAliases Get a list of service alias
	GetServiceAliases(ctx context.Context, query map[string]string) *api.BatchQueryResponse
}

// InstanceOperateServer Example related operations
type InstanceOperateServer interface {

	// CreateInstances Batch creation instance
	CreateInstances(ctx context.Context, reqs []*api.Instance) *api.BatchWriteResponse

	// DeleteInstances Batch delete instance
	DeleteInstances(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse

	// DeleteInstancesByHost Delete instance according to HOST information batch
	DeleteInstancesByHost(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse

	// UpdateInstances Batch update instance
	UpdateInstances(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse

	// UpdateInstancesIsolate Batch update instance isolation state
	UpdateInstancesIsolate(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse

	// GetInstances Get an instance list
	GetInstances(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetInstancesCount Get an instance quantity
	GetInstancesCount(ctx context.Context) *api.BatchQueryResponse

	// CleanInstance Clean up instance
	CleanInstance(ctx context.Context, req *api.Instance) *api.Response
}

// ClientServer Client related operation  Client operation interface definition
type ClientServer interface {

	// RegisterInstance create one instance by client
	RegisterInstance(ctx context.Context, req *api.Instance) *api.Response

	// DeregisterInstance delete onr instance by client
	DeregisterInstance(ctx context.Context, req *api.Instance) *api.Response

	// ReportClient Client gets geographic location information
	ReportClient(ctx context.Context, req *api.Client) *api.Response

	// GetReportClientWithCache Used to obtain the ReportClient information and serve as the SD result of Prometheus
	GetReportClientWithCache(ctx context.Context, query map[string]string) *model.PrometheusDiscoveryResponse

	// GetServiceWithCache Used for client acquisition service information
	GetServiceWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse

	// ServiceInstancesCache Used for client acquisition service instance information
	ServiceInstancesCache(ctx context.Context, req *api.Service) *api.DiscoverResponse

	// GetRoutingConfigWithCache User Client Get Service Routing Configuration Information
	GetRoutingConfigWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse

	// GetRateLimitWithCache User Client Get Service Limit Configuration Information
	GetRateLimitWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse

	// GetCircuitBreakerWithCache Fuse configuration information for obtaining services for clients
	GetCircuitBreakerWithCache(ctx context.Context, req *api.Service) *api.DiscoverResponse
}

// PlatformOperateServer Position of the platform
type PlatformOperateServer interface {

	// CreatePlatforms Batch creation related platform
	CreatePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse

	// UpdatePlatforms Batch update platform information
	UpdatePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse

	// DeletePlatforms Batch delete platform information
	DeletePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse

	// GetPlatforms Get a list of platforms
	GetPlatforms(ctx context.Context, query map[string]string) *api.BatchQueryResponse

	// GetPlatformToken Get the platform token
	GetPlatformToken(ctx context.Context, req *api.Platform) *api.Response
}

// L5OperateServer L5 related operations
type L5OperateServer interface {

	// SyncByAgentCmd Get routing information according to SID list
	SyncByAgentCmd(ctx context.Context, sbac *l5.Cl5SyncByAgentCmd) (*l5.Cl5SyncByAgentAckCmd, error)

	// RegisterByNameCmd Look for the corresponding SID list according to the list of service names
	RegisterByNameCmd(rbnc *l5.Cl5RegisterByNameCmd) (*l5.Cl5RegisterByNameAckCmd, error)
}

// ReportClientOperateServer Report information operation interface on the client
type ReportClientOperateServer interface {
	// GetReportClients Query the client information reported
	GetReportClients(ctx context.Context, query map[string]string) *api.BatchQueryResponse
}

// DiscoverServer Server discovered by the service
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
	// Client operation interface definition
	ClientServer
	// Get cache management
	Cache() *cache.CacheManager
	// Platform-related operation
	PlatformOperateServer
	// L5 related operations
	L5OperateServer
	// GetServiceInstanceRevision Get the version of the service
	GetServiceInstanceRevision(serviceID string, instances []*model.Instance) (string, error)
}
