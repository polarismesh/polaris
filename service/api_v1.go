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

	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	"github.com/polarismesh/polaris/common/api/l5"
	"github.com/polarismesh/polaris/common/model"
)

// CircuitBreakerOperateServer Melting rule related treatment
type CircuitBreakerOperateServer interface {
	// CreateCircuitBreakers Create a CircuitBreaker rule
	// Deprecated: not support from 1.14.x
	CreateCircuitBreakers(ctx context.Context, req []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse
	// CreateCircuitBreakerVersions Create a melt rule version
	// Deprecated: not support from 1.14.x
	CreateCircuitBreakerVersions(ctx context.Context, req []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse
	// DeleteCircuitBreakers Delete CircuitBreaker rules
	// Deprecated: not support from 1.14.x
	DeleteCircuitBreakers(ctx context.Context, req []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse
	// UpdateCircuitBreakers Modify the CircuitBreaker rule
	// Deprecated: not support from 1.14.x
	UpdateCircuitBreakers(ctx context.Context, req []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse
	// ReleaseCircuitBreakers Release CircuitBreaker rule
	// Deprecated: not support from 1.14.x
	ReleaseCircuitBreakers(ctx context.Context, req []*apiservice.ConfigRelease) *apiservice.BatchWriteResponse
	// UnBindCircuitBreakers Solution CircuitBreaker rule
	// Deprecated: not support from 1.14.x
	UnBindCircuitBreakers(ctx context.Context, req []*apiservice.ConfigRelease) *apiservice.BatchWriteResponse
	// GetCircuitBreaker Get CircuitBreaker regular according to ID and VERSION
	// Deprecated: not support from 1.14.x
	GetCircuitBreaker(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
	// GetCircuitBreakerVersions Query all versions of the CircuitBreaker rule
	// Deprecated: not support from 1.14.x
	GetCircuitBreakerVersions(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
	// GetMasterCircuitBreakers Query Master CircuitBreaker rules
	// Deprecated: not support from 1.14.x
	GetMasterCircuitBreakers(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
	// GetReleaseCircuitBreakers Query the released CircuitBreaker rule according to the rule ID
	// Deprecated: not support from 1.14.x
	GetReleaseCircuitBreakers(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
	// GetCircuitBreakerByService Binding CircuitBreaker rule based on service query
	// Deprecated: not support from 1.14.x
	GetCircuitBreakerByService(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
	// GetCircuitBreakerToken Get CircuitBreaker rules token
	// Deprecated: not support from 1.14.x
	GetCircuitBreakerToken(ctx context.Context, req *apifault.CircuitBreaker) *apiservice.Response
	// CreateCircuitBreakerRules Create a CircuitBreaker rule
	CreateCircuitBreakerRules(ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse
	// DeleteCircuitBreakerRules Delete current CircuitBreaker rules
	DeleteCircuitBreakerRules(ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse
	// EnableCircuitBreakerRules Enable the CircuitBreaker rule
	EnableCircuitBreakerRules(ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse
	// UpdateCircuitBreakerRules Modify the CircuitBreaker rule
	UpdateCircuitBreakerRules(ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse
	// GetCircuitBreakerRules Query CircuitBreaker rules
	GetCircuitBreakerRules(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
}

// RateLimitOperateServer Lamflow rule related operation
type RateLimitOperateServer interface {
	// CreateRateLimits Create a RateLimit rule
	CreateRateLimits(ctx context.Context, request []*apitraffic.Rule) *apiservice.BatchWriteResponse
	// DeleteRateLimits Delete current RateLimit rules
	DeleteRateLimits(ctx context.Context, request []*apitraffic.Rule) *apiservice.BatchWriteResponse
	// EnableRateLimits Enable the RateLimit rule
	EnableRateLimits(ctx context.Context, request []*apitraffic.Rule) *apiservice.BatchWriteResponse
	// UpdateRateLimits Modify the RateLimit rule
	UpdateRateLimits(ctx context.Context, request []*apitraffic.Rule) *apiservice.BatchWriteResponse
	// GetRateLimits Query RateLimit rules
	GetRateLimits(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
}

// RouteRuleOperateServer Routing rules related operations
type RouteRuleOperateServer interface {
	// CreateRoutingConfigs Batch creation routing configuration
	CreateRoutingConfigs(ctx context.Context, req []*apitraffic.Routing) *apiservice.BatchWriteResponse
	// DeleteRoutingConfigs Batch delete routing configuration
	DeleteRoutingConfigs(ctx context.Context, req []*apitraffic.Routing) *apiservice.BatchWriteResponse
	// UpdateRoutingConfigs Batch update routing configuration
	UpdateRoutingConfigs(ctx context.Context, req []*apitraffic.Routing) *apiservice.BatchWriteResponse
	// GetRoutingConfigs Inquiry route configuration to OSS
	GetRoutingConfigs(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
}

// ServiceOperateServer Service related operations
type ServiceOperateServer interface {
	// CreateServices Batch creation service
	CreateServices(ctx context.Context, req []*apiservice.Service) *apiservice.BatchWriteResponse
	// DeleteServices Batch delete service
	DeleteServices(ctx context.Context, req []*apiservice.Service) *apiservice.BatchWriteResponse
	// UpdateServices Batch update service
	UpdateServices(ctx context.Context, req []*apiservice.Service) *apiservice.BatchWriteResponse
	// UpdateServiceToken Update service token
	UpdateServiceToken(ctx context.Context, req *apiservice.Service) *apiservice.Response
	// GetServices Get a list of service
	GetServices(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
	// GetAllServices Get all service list
	GetAllServices(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
	// GetServicesCount Total number of services
	GetServicesCount(ctx context.Context) *apiservice.BatchQueryResponse
	// GetServiceToken Get service token
	GetServiceToken(ctx context.Context, req *apiservice.Service) *apiservice.Response
	// GetServiceOwner Owner for obtaining service
	GetServiceOwner(ctx context.Context, req []*apiservice.Service) *apiservice.BatchQueryResponse
}

// ServiceAliasOperateServer Service alias related operations
type ServiceAliasOperateServer interface {
	// CreateServiceAlias Create a service alias
	CreateServiceAlias(ctx context.Context, req *apiservice.ServiceAlias) *apiservice.Response
	// DeleteServiceAliases Batch delete service alias
	DeleteServiceAliases(ctx context.Context, req []*apiservice.ServiceAlias) *apiservice.BatchWriteResponse
	// UpdateServiceAlias Update service alias
	UpdateServiceAlias(ctx context.Context, req *apiservice.ServiceAlias) *apiservice.Response
	// GetServiceAliases Get a list of service alias
	GetServiceAliases(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
}

// InstanceOperateServer Example related operations
type InstanceOperateServer interface {
	// CreateInstances Batch creation instance
	CreateInstances(ctx context.Context, reqs []*apiservice.Instance) *apiservice.BatchWriteResponse
	// DeleteInstances Batch delete instance
	DeleteInstances(ctx context.Context, req []*apiservice.Instance) *apiservice.BatchWriteResponse
	// DeleteInstancesByHost Delete instance according to HOST information batch
	DeleteInstancesByHost(ctx context.Context, req []*apiservice.Instance) *apiservice.BatchWriteResponse
	// UpdateInstances Batch update instance
	UpdateInstances(ctx context.Context, req []*apiservice.Instance) *apiservice.BatchWriteResponse
	// UpdateInstancesIsolate Batch update instance isolation state
	UpdateInstancesIsolate(ctx context.Context, req []*apiservice.Instance) *apiservice.BatchWriteResponse
	// GetInstances Get an instance list
	GetInstances(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
	// GetInstancesCount Get an instance quantity
	GetInstancesCount(ctx context.Context) *apiservice.BatchQueryResponse
	// GetInstanceLabels Get an instance tag under a service
	GetInstanceLabels(ctx context.Context, query map[string]string) *apiservice.Response
}

// ClientServer Client related operation  Client operation interface definition
type ClientServer interface {
	// RegisterInstance create one instance by client
	RegisterInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response
	// DeregisterInstance delete onr instance by client
	DeregisterInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response
	// ReportClient Client gets geographic location information
	ReportClient(ctx context.Context, req *apiservice.Client) *apiservice.Response
	// GetPrometheusTargets Used to obtain the ReportClient information and serve as the SD result of Prometheus
	GetPrometheusTargets(ctx context.Context, query map[string]string) *model.PrometheusDiscoveryResponse
	// GetServiceWithCache Used for client acquisition service information
	GetServiceWithCache(ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse
	// ServiceInstancesCache Used for client acquisition service instance information
	ServiceInstancesCache(ctx context.Context, filter *apiservice.DiscoverFilter, req *apiservice.Service) *apiservice.DiscoverResponse
	// GetRoutingConfigWithCache User Client Get Service Routing Configuration Information
	GetRoutingConfigWithCache(ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse
	// GetRateLimitWithCache User Client Get Service Limit Configuration Information
	GetRateLimitWithCache(ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse
	// GetCircuitBreakerWithCache Fuse configuration information for obtaining services for clients
	GetCircuitBreakerWithCache(ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse
	// GetFaultDetectWithCache User Client Get FaultDetect Rule Information
	GetFaultDetectWithCache(ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse
	// GetServiceContractWithCache User Client Get ServiceContract Rule Information
	GetServiceContractWithCache(ctx context.Context, req *apiservice.ServiceContract) *apiservice.Response
	// UpdateInstance update one instance by client
	UpdateInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response
	// ReportServiceContract client report service_contract
	ReportServiceContract(ctx context.Context, req *apiservice.ServiceContract) *apiservice.Response
	// GetLaneRuleWithCache fetch lane rules by client
	GetLaneRuleWithCache(ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse
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
	GetReportClients(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
}

// RouterRuleOperateServer Routing rules related operations
type RouterRuleOperateServer interface {
	// CreateRoutingConfigsV2 Batch creation routing configuration
	CreateRoutingConfigsV2(ctx context.Context, req []*apitraffic.RouteRule) *apiservice.BatchWriteResponse
	// DeleteRoutingConfigsV2 Batch delete routing configuration
	DeleteRoutingConfigsV2(ctx context.Context, req []*apitraffic.RouteRule) *apiservice.BatchWriteResponse
	// UpdateRoutingConfigsV2 Batch update routing configuration
	UpdateRoutingConfigsV2(ctx context.Context, req []*apitraffic.RouteRule) *apiservice.BatchWriteResponse
	// QueryRoutingConfigsV2 Inquiry route configuration to OSS
	QueryRoutingConfigsV2(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
	// EnableRoutings batch enable routing rules
	EnableRoutings(ctx context.Context, req []*apitraffic.RouteRule) *apiservice.BatchWriteResponse
}

// FaultDetectRuleOperateServer Fault detect rules related operations
type FaultDetectRuleOperateServer interface {
	// CreateFaultDetectRules create the fault detect rule by request
	CreateFaultDetectRules(ctx context.Context, request []*apifault.FaultDetectRule) *apiservice.BatchWriteResponse
	// DeleteFaultDetectRules delete the fault detect rule by request
	DeleteFaultDetectRules(ctx context.Context, request []*apifault.FaultDetectRule) *apiservice.BatchWriteResponse
	// UpdateFaultDetectRules update the fault detect rule by request
	UpdateFaultDetectRules(ctx context.Context, request []*apifault.FaultDetectRule) *apiservice.BatchWriteResponse
	// GetFaultDetectRules get the fault detect rule by request
	GetFaultDetectRules(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
}

// ServiceContractOperateServer service contract operations
type ServiceContractOperateServer interface {
	// CreateServiceContracts .
	CreateServiceContracts(ctx context.Context, req []*apiservice.ServiceContract) *apiservice.BatchWriteResponse
	// DeleteServiceContracts .
	DeleteServiceContracts(ctx context.Context, req []*apiservice.ServiceContract) *apiservice.BatchWriteResponse
	// GetServiceContracts .
	GetServiceContracts(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse
	// CreateServiceContractInterfaces .
	CreateServiceContractInterfaces(ctx context.Context, contract *apiservice.ServiceContract,
		source apiservice.InterfaceDescriptor_Source) *apiservice.Response
	// AppendServiceContractInterfaces .
	AppendServiceContractInterfaces(ctx context.Context, contract *apiservice.ServiceContract,
		source apiservice.InterfaceDescriptor_Source) *apiservice.Response
	// DeleteServiceContractInterfaces .
	DeleteServiceContractInterfaces(ctx context.Context, contract *apiservice.ServiceContract) *apiservice.Response
	// GetServiceContractVersions .
	GetServiceContractVersions(ctx context.Context, filter map[string]string) *apiservice.BatchQueryResponse
}

type DiscoverServerV1 interface {
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
}
