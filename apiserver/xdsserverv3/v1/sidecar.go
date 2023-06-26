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

package v1

import (
	"context"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

func (x *XDSServer) makeSnapshot(ns, version string, tlsMode resource.TLSMode,
	services map[model.ServiceKey]*resource.ServiceInfo) (err error) {

	resources := make(map[resourcev3.Type][]types.Resource)
	resources[resourcev3.EndpointType] = makeEndpoints(services)
	resources[resourcev3.RouteType] = x.makeSidecarVirtualHosts(services)

	cacheKey := ns

	switch tlsMode {
	case resource.TLSModeNone:
		resources[resourcev3.ClusterType] = x.makeClusters(services)
		resources[resourcev3.ListenerType] = makeListeners()
	case resource.TLSModePermissive:
		resources[resourcev3.ClusterType] = x.makePermissiveClusters(services)
		resources[resourcev3.ListenerType] = makePermissiveListeners()
		cacheKey = ns + "/permissive"
	case resource.TLSModeStrict:
		resources[resourcev3.ClusterType] = x.makeStrictClusters(services)
		resources[resourcev3.ListenerType] = makeStrictListeners()
		cacheKey = ns + "/strict"
	}

	snapshot, err := cachev3.NewSnapshot(version, resources)
	if err != nil {
		log.Error("[XDS][Sidecar][V1] fail to create snapshot", zap.String("namespace", ns),
			zap.String("tls", string(tlsMode)), zap.Error(err))
		return err
	}
	if err = snapshot.Consistent(); err != nil {
		return err
	}
	log.Info("[XDS][Sidecar][V1] upsert snapshot success", zap.String("namespace", ns),
		zap.String("tls", string(tlsMode)), zap.String("resource", string(resource.DumpSnapShotJSON(snapshot))))
	// 为每个 ns 刷写 cache ，推送 xds 更新
	if err := x.cache.SetSnapshot(context.Background(), cacheKey, snapshot); err != nil {
		log.Error("[XDS][Sidecar][V1] upsert snapshot error", zap.String("namespace", ns),
			zap.String("tls", string(tlsMode)), zap.Error(err))
		return err
	}
	return
}

func (x *XDSServer) makeSidecarVirtualHosts(services map[model.ServiceKey]*resource.ServiceInfo) []types.Resource {
	// 每个 polaris serviceInfo 对应一个 virtualHost
	var (
		routeConfs []types.Resource
		hosts      []*route.VirtualHost
	)

	for _, serviceInfo := range services {
		vHost := &route.VirtualHost{
			Name:    serviceInfo.Name,
			Domains: resource.GenerateServiceDomains(serviceInfo),
			Routes:  x.makeSidecarRoutes(serviceInfo),
		}
		hosts = append(hosts, vHost)
	}

	// 最后是 allow_any
	hosts = append(hosts, resource.BuildAllowAnyVHost())

	routeConfiguration := &route.RouteConfiguration{
		Name: resource.RouteConfigName,
		ValidateClusters: &wrappers.BoolValue{
			Value: false,
		},
		VirtualHosts: hosts,
	}

	return append(routeConfs, routeConfiguration)
}

func makeEndpoints(services map[model.ServiceKey]*resource.ServiceInfo) []types.Resource {
	var clusterLoads []types.Resource
	for _, serviceInfo := range services {
		var lbEndpoints []*endpoint.LbEndpoint
		for _, instance := range serviceInfo.Instances {
			// 只加入健康的实例
			if !resource.IsNormalEndpoint(instance) {
				continue
			}
			ep := &endpoint.LbEndpoint{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.SocketAddress_TCP,
									Address:  instance.Host.Value,
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: instance.Port.Value,
									},
								},
							},
						},
					},
				},
				HealthStatus:        resource.FormatEndpointHealth(instance),
				LoadBalancingWeight: utils.NewUInt32Value(instance.GetWeight().GetValue()),
				Metadata:            resource.GenEndpointMetaFromPolarisIns(instance),
			}

			lbEndpoints = append(lbEndpoints, ep)
		}

		cla := &endpoint.ClusterLoadAssignment{
			ClusterName: serviceInfo.Name,
			Endpoints: []*endpoint.LocalityLbEndpoints{
				{
					LbEndpoints: lbEndpoints,
				},
			},
		}

		clusterLoads = append(clusterLoads, cla)
	}

	return clusterLoads
}

// makeSidecarRoutes .
func (x *XDSServer) makeSidecarRoutes(serviceInfo *resource.ServiceInfo) []*route.Route {
	var (
		routes        []*route.Route
		matchAllRoute *route.Route
	)
	// 路由目前只处理 inbounds
	rules := resource.FilterInboundRouterRule(serviceInfo)
	for _, rule := range rules {
		var (
			matchAll     bool
			destinations []*traffic_manage.DestinationGroup
		)
		for _, dest := range rule.GetDestinations() {
			if !serviceInfo.MatchService(dest.GetNamespace(), dest.GetService()) {
				continue
			}
			destinations = append(destinations, dest)
		}

		routeMatch := &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{Prefix: "/"},
		}
		// 使用 sources 生成 routeMatch
		for _, source := range rule.GetSources() {
			if len(source.GetArguments()) == 0 {
				matchAll = true
				break
			}
			for _, arg := range source.GetArguments() {
				if arg.Key == utils.MatchAll {
					matchAll = true
					break
				}
			}
			if matchAll {
				break
			} else {
				resource.BuildSidecarRouteMatch(routeMatch, source)
			}
		}

		currentRoute := &route.Route{
			Match: routeMatch,
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_WeightedClusters{
						WeightedClusters: buildWeightClustersForSidecar(serviceInfo, destinations),
					},
				},
			},
		}

		if matchAll {
			matchAllRoute = currentRoute
		} else {
			routes = append(routes, currentRoute)
		}
	}
	if matchAllRoute == nil {
		// 如果没有路由，会进入最后的默认处理
		routes = append(routes, getDefaultRoute(serviceInfo.Name))
	} else {
		routes = append(routes, matchAllRoute)
	}
	return routes
}

// 默认路由
func getDefaultRoute(serviceName string) *route.Route {
	return &route.Route{
		Match: &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: "/",
			},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_Cluster{
					Cluster: serviceName,
				},
			},
		},
	}
}

func buildWeightClustersForSidecar(svcInfo *resource.ServiceInfo,
	destinations []*traffic_manage.DestinationGroup) *route.WeightedCluster {
	weightClusters := buildWeightClustersV2(destinations)
	for i := range weightClusters.Clusters {
		weightClusters.Clusters[i].Name = svcInfo.Name
	}
	return weightClusters
}

func buildWeightClustersV2(destinations []*traffic_manage.DestinationGroup) *route.WeightedCluster {
	var (
		weightedClusters []*route.WeightedCluster_ClusterWeight
		totalWeight      uint32
	)

	// 使用 destinations 生成 weightedClusters。makeClusters() 也使用这个字段生成对应的 subset
	for _, destination := range destinations {
		if destination.GetWeight() == 0 {
			continue
		}
		fields := make(map[string]*_struct.Value)
		for k, v := range destination.GetLabels() {
			if k == utils.MatchAll && v.GetValue().GetValue() == utils.MatchAll {
				// 重置 cluster 的匹配规则
				fields = make(map[string]*_struct.Value)
				break
			}
			fields[k] = &_struct.Value{
				Kind: &_struct.Value_StringValue{
					StringValue: v.Value.Value,
				},
			}
		}
		cluster := &route.WeightedCluster_ClusterWeight{
			Name:   destination.Service,
			Weight: utils.NewUInt32Value(destination.GetWeight()),
			MetadataMatch: &core.Metadata{
				FilterMetadata: map[string]*_struct.Struct{
					"envoy.lb": {
						Fields: fields,
					},
				},
			},
		}
		if len(fields) == 0 {
			cluster.MetadataMatch = nil
		}
		weightedClusters = append(weightedClusters, cluster)
		totalWeight += destination.Weight
	}
	return &route.WeightedCluster{
		TotalWeight: &wrappers.UInt32Value{Value: totalWeight},
		Clusters:    weightedClusters,
	}
}
