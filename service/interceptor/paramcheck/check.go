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

package paramcheck

import (
	"context"

	"github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	"github.com/polarismesh/polaris/common/api/l5"
	"github.com/polarismesh/polaris/common/model"
)

// AppendServiceContractInterfaces implements service.DiscoverServer.
func (svr *Server) AppendServiceContractInterfaces(ctx context.Context,
	contract *service_manage.ServiceContract,
	source service_manage.InterfaceDescriptor_Source) *service_manage.Response {
	return svr.nextSvr.AppendServiceContractInterfaces(ctx, contract, source)
}

// CreateCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) CreateCircuitBreakerRules(ctx context.Context,
	request []*fault_tolerance.CircuitBreakerRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateCircuitBreakerRules(ctx, request)
}

// CreateCircuitBreakerVersions implements service.DiscoverServer.
func (svr *Server) CreateCircuitBreakerVersions(ctx context.Context,
	req []*fault_tolerance.CircuitBreaker) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateCircuitBreakerVersions(ctx, req)
}

// CreateCircuitBreakers implements service.DiscoverServer.
func (svr *Server) CreateCircuitBreakers(ctx context.Context,
	req []*fault_tolerance.CircuitBreaker) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateCircuitBreakers(ctx, req)
}

// CreateFaultDetectRules implements service.DiscoverServer.
func (svr *Server) CreateFaultDetectRules(ctx context.Context,
	request []*fault_tolerance.FaultDetectRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateFaultDetectRules(ctx, request)
}

// CreateInstances implements service.DiscoverServer.
func (svr *Server) CreateInstances(ctx context.Context,
	reqs []*service_manage.Instance) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateInstances(ctx, reqs)
}

// CreateRateLimits implements service.DiscoverServer.
func (svr *Server) CreateRateLimits(ctx context.Context,
	request []*traffic_manage.Rule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateRateLimits(ctx, request)
}

// CreateRoutingConfigs implements service.DiscoverServer.
func (svr *Server) CreateRoutingConfigs(ctx context.Context,
	req []*traffic_manage.Routing) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateRoutingConfigs(ctx, req)
}

// CreateRoutingConfigsV2 implements service.DiscoverServer.
func (svr *Server) CreateRoutingConfigsV2(ctx context.Context,
	req []*traffic_manage.RouteRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateRoutingConfigsV2(ctx, req)
}

// CreateServiceAlias implements service.DiscoverServer.
func (svr *Server) CreateServiceAlias(ctx context.Context,
	req *service_manage.ServiceAlias) *service_manage.Response {
	return svr.nextSvr.CreateServiceAlias(ctx, req)
}

// CreateServiceContractInterfaces implements service.DiscoverServer.
func (svr *Server) CreateServiceContractInterfaces(ctx context.Context,
	contract *service_manage.ServiceContract, source service_manage.InterfaceDescriptor_Source) *service_manage.Response {
	return svr.nextSvr.CreateServiceContractInterfaces(ctx, contract, source)
}

// CreateServiceContracts implements service.DiscoverServer.
func (svr *Server) CreateServiceContracts(ctx context.Context,
	req []*service_manage.ServiceContract) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateServiceContracts(ctx, req)
}

// CreateServices implements service.DiscoverServer.
func (svr *Server) CreateServices(ctx context.Context,
	req []*service_manage.Service) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateServices(ctx, req)
}

// DeleteCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) DeleteCircuitBreakerRules(ctx context.Context,
	request []*fault_tolerance.CircuitBreakerRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteCircuitBreakerRules(ctx, request)
}

// DeleteCircuitBreakers implements service.DiscoverServer.
func (svr *Server) DeleteCircuitBreakers(ctx context.Context,
	req []*fault_tolerance.CircuitBreaker) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteCircuitBreakers(ctx, req)
}

// DeleteFaultDetectRules implements service.DiscoverServer.
func (svr *Server) DeleteFaultDetectRules(ctx context.Context,
	request []*fault_tolerance.FaultDetectRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteFaultDetectRules(ctx, request)
}

// DeleteInstances implements service.DiscoverServer.
func (svr *Server) DeleteInstances(ctx context.Context,
	req []*service_manage.Instance) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteInstances(ctx, req)
}

// DeleteInstancesByHost implements service.DiscoverServer.
func (svr *Server) DeleteInstancesByHost(ctx context.Context,
	req []*service_manage.Instance) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteInstancesByHost(ctx, req)
}

// DeleteRateLimits implements service.DiscoverServer.
func (svr *Server) DeleteRateLimits(ctx context.Context,
	request []*traffic_manage.Rule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteRateLimits(ctx, request)
}

// DeleteRoutingConfigs implements service.DiscoverServer.
func (svr *Server) DeleteRoutingConfigs(ctx context.Context,
	req []*traffic_manage.Routing) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteRoutingConfigs(ctx, req)
}

// DeleteRoutingConfigsV2 implements service.DiscoverServer.
func (svr *Server) DeleteRoutingConfigsV2(ctx context.Context,
	req []*traffic_manage.RouteRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteRoutingConfigsV2(ctx, req)
}

// DeleteServiceAliases implements service.DiscoverServer.
func (svr *Server) DeleteServiceAliases(ctx context.Context,
	req []*service_manage.ServiceAlias) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteServiceAliases(ctx, req)
}

// DeleteServiceContractInterfaces implements service.DiscoverServer.
func (svr *Server) DeleteServiceContractInterfaces(ctx context.Context,
	contract *service_manage.ServiceContract) *service_manage.Response {
	return svr.nextSvr.DeleteServiceContractInterfaces(ctx, contract)
}

// DeleteServiceContracts implements service.DiscoverServer.
func (svr *Server) DeleteServiceContracts(ctx context.Context,
	req []*service_manage.ServiceContract) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteServiceContracts(ctx, req)
}

// DeleteServices implements service.DiscoverServer.
func (svr *Server) DeleteServices(ctx context.Context,
	req []*service_manage.Service) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteServices(ctx, req)
}

// EnableCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) EnableCircuitBreakerRules(ctx context.Context,
	request []*fault_tolerance.CircuitBreakerRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.EnableCircuitBreakerRules(ctx, request)
}

// EnableRateLimits implements service.DiscoverServer.
func (svr *Server) EnableRateLimits(ctx context.Context,
	request []*traffic_manage.Rule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.EnableRateLimits(ctx, request)
}

// EnableRoutings implements service.DiscoverServer.
func (svr *Server) EnableRoutings(ctx context.Context,
	req []*traffic_manage.RouteRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.EnableRoutings(ctx, req)
}

// GetAllServices implements service.DiscoverServer.
func (svr *Server) GetAllServices(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetAllServices(ctx, query)
}

// GetCircuitBreaker implements service.DiscoverServer.
func (svr *Server) GetCircuitBreaker(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetCircuitBreaker(ctx, query)
}

// GetCircuitBreakerByService implements service.DiscoverServer.
func (svr *Server) GetCircuitBreakerByService(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetCircuitBreakerByService(ctx, query)
}

// GetCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) GetCircuitBreakerRules(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetCircuitBreakerRules(ctx, query)
}

// GetCircuitBreakerToken implements service.DiscoverServer.
func (svr *Server) GetCircuitBreakerToken(ctx context.Context, req *fault_tolerance.CircuitBreaker) *service_manage.Response {
	return svr.nextSvr.GetCircuitBreakerToken(ctx, req)
}

// GetCircuitBreakerVersions implements service.DiscoverServer.
func (svr *Server) GetCircuitBreakerVersions(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetCircuitBreakerVersions(ctx, query)
}

// GetFaultDetectRules implements service.DiscoverServer.
func (svr *Server) GetFaultDetectRules(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetFaultDetectRules(ctx, query)
}

// GetInstanceLabels implements service.DiscoverServer.
func (svr *Server) GetInstanceLabels(ctx context.Context, query map[string]string) *service_manage.Response {
	return svr.nextSvr.GetInstanceLabels(ctx, query)
}

// GetInstances implements service.DiscoverServer.
func (svr *Server) GetInstances(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetInstances(ctx, query)
}

// GetInstancesCount implements service.DiscoverServer.
func (svr *Server) GetInstancesCount(ctx context.Context) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetInstancesCount(ctx)
}

// GetMasterCircuitBreakers implements service.DiscoverServer.
func (svr *Server) GetMasterCircuitBreakers(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetMasterCircuitBreakers(ctx, query)
}

// GetPrometheusTargets implements service.DiscoverServer.
func (svr *Server) GetPrometheusTargets(ctx context.Context, query map[string]string) *model.PrometheusDiscoveryResponse {
	return svr.nextSvr.GetPrometheusTargets(ctx, query)
}

// GetRateLimits implements service.DiscoverServer.
func (svr *Server) GetRateLimits(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetRateLimits(ctx, query)
}

// GetReleaseCircuitBreakers implements service.DiscoverServer.
func (svr *Server) GetReleaseCircuitBreakers(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetReleaseCircuitBreakers(ctx, query)
}

// GetRoutingConfigs implements service.DiscoverServer.
func (svr *Server) GetRoutingConfigs(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetRoutingConfigs(ctx, query)
}

// GetServiceAliases implements service.DiscoverServer.
func (svr *Server) GetServiceAliases(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetServiceAliases(ctx, query)
}

// GetServiceContractVersions implements service.DiscoverServer.
func (svr *Server) GetServiceContractVersions(ctx context.Context, filter map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetServiceContractVersions(ctx, filter)
}

// GetServiceContracts implements service.DiscoverServer.
func (svr *Server) GetServiceContracts(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetServiceContracts(ctx, query)
}

// GetServiceOwner implements service.DiscoverServer.
func (svr *Server) GetServiceOwner(ctx context.Context, req []*service_manage.Service) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetServiceOwner(ctx, req)
}

// GetServiceToken implements service.DiscoverServer.
func (svr *Server) GetServiceToken(ctx context.Context, req *service_manage.Service) *service_manage.Response {
	return svr.nextSvr.GetServiceToken(ctx, req)
}

// GetServices implements service.DiscoverServer.
func (svr *Server) GetServices(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetServices(ctx, query)
}

// GetServicesCount implements service.DiscoverServer.
func (svr *Server) GetServicesCount(ctx context.Context) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetServicesCount(ctx)
}

// QueryRoutingConfigsV2 implements service.DiscoverServer.
func (svr *Server) QueryRoutingConfigsV2(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.QueryRoutingConfigsV2(ctx, query)
}

// RegisterByNameCmd implements service.DiscoverServer.
func (svr *Server) RegisterByNameCmd(rbnc *l5.Cl5RegisterByNameCmd) (*l5.Cl5RegisterByNameAckCmd, error) {
	return svr.nextSvr.RegisterByNameCmd(rbnc)
}

// ReleaseCircuitBreakers implements service.DiscoverServer.
func (svr *Server) ReleaseCircuitBreakers(ctx context.Context, req []*service_manage.ConfigRelease) *service_manage.BatchWriteResponse {
	return svr.nextSvr.ReleaseCircuitBreakers(ctx, req)
}

// SyncByAgentCmd implements service.DiscoverServer.
func (svr *Server) SyncByAgentCmd(ctx context.Context, sbac *l5.Cl5SyncByAgentCmd) (*l5.Cl5SyncByAgentAckCmd, error) {
	return svr.nextSvr.SyncByAgentCmd(ctx, sbac)
}

// UnBindCircuitBreakers implements service.DiscoverServer.
func (svr *Server) UnBindCircuitBreakers(ctx context.Context, req []*service_manage.ConfigRelease) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UnBindCircuitBreakers(ctx, req)
}

// UpdateCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) UpdateCircuitBreakerRules(ctx context.Context, request []*fault_tolerance.CircuitBreakerRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateCircuitBreakerRules(ctx, request)
}

// UpdateCircuitBreakers implements service.DiscoverServer.
func (svr *Server) UpdateCircuitBreakers(ctx context.Context, req []*fault_tolerance.CircuitBreaker) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateCircuitBreakers(ctx, req)
}

// UpdateFaultDetectRules implements service.DiscoverServer.
func (svr *Server) UpdateFaultDetectRules(ctx context.Context, request []*fault_tolerance.FaultDetectRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateFaultDetectRules(ctx, request)
}

// UpdateInstances implements service.DiscoverServer.
func (svr *Server) UpdateInstances(ctx context.Context, req []*service_manage.Instance) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateInstances(ctx, req)
}

// UpdateInstancesIsolate implements service.DiscoverServer.
func (svr *Server) UpdateInstancesIsolate(ctx context.Context, req []*service_manage.Instance) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateInstancesIsolate(ctx, req)
}

// UpdateRateLimits implements service.DiscoverServer.
func (svr *Server) UpdateRateLimits(ctx context.Context, request []*traffic_manage.Rule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateRateLimits(ctx, request)
}

// UpdateRoutingConfigs implements service.DiscoverServer.
func (svr *Server) UpdateRoutingConfigs(ctx context.Context, req []*traffic_manage.Routing) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateRoutingConfigs(ctx, req)
}

// UpdateRoutingConfigsV2 implements service.DiscoverServer.
func (svr *Server) UpdateRoutingConfigsV2(ctx context.Context, req []*traffic_manage.RouteRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateRoutingConfigsV2(ctx, req)
}

// UpdateServiceAlias implements service.DiscoverServer.
func (svr *Server) UpdateServiceAlias(ctx context.Context, req *service_manage.ServiceAlias) *service_manage.Response {
	return svr.nextSvr.UpdateServiceAlias(ctx, req)
}

// UpdateServiceToken implements service.DiscoverServer.
func (svr *Server) UpdateServiceToken(ctx context.Context, req *service_manage.Service) *service_manage.Response {
	return svr.nextSvr.UpdateServiceToken(ctx, req)
}

// UpdateServices implements service.DiscoverServer.
func (svr *Server) UpdateServices(ctx context.Context, req []*service_manage.Service) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateServices(ctx, req)
}
