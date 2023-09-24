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

package xdsserverv3

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	v32 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
)

// RDSBuilder .
type RDSBuilder struct {
	client *resource.XDSClient
	svr    service.DiscoverServer
}

func (rds *RDSBuilder) Init(client *resource.XDSClient, svr service.DiscoverServer) {
	rds.client = client
	rds.svr = svr
}

func (rds *RDSBuilder) Generate(option *resource.BuildOption) (interface{}, error) {
	var resources []types.Resource

	switch rds.client.RunType {
	case resource.RunTypeGateway:
		// Envoy Gateway 场景只需要支持入流量场景处理即可
		ret, err := rds.makeGatewayVirtualHosts(option)
		if err != nil {
			return nil, err
		}
		resources = ret
	case resource.RunTypeSidecar:
		resources = rds.makeSidecarVirtualHosts(option)
	}
	return resources, nil
}

func (rds *RDSBuilder) makeSidecarVirtualHosts(option *resource.BuildOption) []types.Resource {
	// 每个 polaris serviceInfo 对应一个 virtualHost
	var (
		routeConfs []types.Resource
		hosts      []*route.VirtualHost
	)

	selfService := model.ServiceKey{
		Namespace: rds.client.GetSelfNamespace(),
		Name:      rds.client.GetSelfService(),
	}

	// step 1: 生成服务的 OUTBOUND 规则
	services := option.Services
	for svcKey, serviceInfo := range services {
		vHost := &route.VirtualHost{
			Name:    resource.MakeServiceName(svcKey, corev3.TrafficDirection_OUTBOUND),
			Domains: resource.GenerateServiceDomains(serviceInfo),
			Routes:  rds.makeSidecarOutBoundRoutes(corev3.TrafficDirection_OUTBOUND, serviceInfo),
		}
		hosts = append(hosts, vHost)
	}

	routeConfiguration := &route.RouteConfiguration{
		Name: resource.OutBoundRouteConfigName,
		ValidateClusters: &wrappers.BoolValue{
			Value: false,
		},
		VirtualHosts: append(hosts, resource.BuildAllowAnyVHost()),
	}
	routeConfs = append(routeConfs, routeConfiguration)

	// step 2: 生成 sidecar 所属服务的 INBOUND 规则
	// 服务信息不存在或者不精确，不下发 InBound RDS 规则信息
	if selfService.IsExact() {
		routeConfs = append(routeConfs, &route.RouteConfiguration{
			Name: resource.InBoundRouteConfigName,
			ValidateClusters: &wrappers.BoolValue{
				Value: false,
			},
			VirtualHosts: []*route.VirtualHost{
				{
					Name:    resource.MakeServiceName(selfService, corev3.TrafficDirection_INBOUND),
					Domains: []string{"*"},
					Routes:  rds.makeSidecarInBoundRoutes(selfService, corev3.TrafficDirection_INBOUND),
				},
			},
		})
	}

	return routeConfs
}

func (rds *RDSBuilder) makeSidecarInBoundRoutes(selfService model.ServiceKey,
	trafficDirection corev3.TrafficDirection) []*route.Route {
	currentRoute := &route.Route{
		Match: &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{Prefix: "/"},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_WeightedClusters{
					WeightedClusters: &route.WeightedCluster{
						Clusters: []*route.WeightedCluster_ClusterWeight{
							{
								Name:   resource.MakeServiceName(selfService, trafficDirection),
								Weight: wrapperspb.UInt32(100),
							},
						},
					},
				},
			},
		},
	}

	seacher := rds.svr.Cache().RateLimit()
	limits, typedPerFilterConfig, err := resource.MakeSidecarLocalRateLimit(seacher, selfService)
	if err == nil {
		currentRoute.TypedPerFilterConfig = typedPerFilterConfig
		currentRoute.GetRoute().RateLimits = limits
	}
	return []*route.Route{
		currentRoute,
	}
}

// makeSidecarOutBoundRoutes .
func (rds *RDSBuilder) makeSidecarOutBoundRoutes(trafficDirection corev3.TrafficDirection,
	serviceInfo *resource.ServiceInfo) []*route.Route {
	var (
		routes        []*route.Route
		matchAllRoute *route.Route
	)
	// 路由目前只处理 inbounds, 由于目前 envoy 获取不到自身服务数据，因此获取所有服务的被调规则
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
				resource.BuildSidecarRouteMatch(rds.client, routeMatch, source)
			}
		}

		currentRoute := resource.MakeSidecarRoute(trafficDirection, routeMatch, serviceInfo, destinations)
		if matchAll {
			matchAllRoute = currentRoute
		} else {
			routes = append(routes, currentRoute)
		}
	}
	if matchAllRoute == nil {
		// 如果没有路由，会进入最后的默认处理
		routes = append(routes, resource.MakeDefaultRoute(trafficDirection, serviceInfo.ServiceKey))
	} else {
		routes = append(routes, matchAllRoute)
	}
	return routes
}

// ---------------------- Envoy Gateway ---------------------- //
func (rds *RDSBuilder) makeGatewayVirtualHosts(option *resource.BuildOption) ([]types.Resource, error) {
	// 每个 polaris serviceInfo 对应一个 virtualHost
	var (
		routeConfs []types.Resource
		hosts      []*route.VirtualHost
	)

	routes, err := rds.makeGatewayRoutes(option, rds.client)
	if err != nil {
		return nil, err
	}
	if len(routes) == 0 {
		return []types.Resource{}, nil
	}

	vHost := &route.VirtualHost{
		Name:    "gateway-virtualhost",
		Domains: resource.MakeServiceGatewayDomains(),
		Routes:  routes,
	}
	hosts = append(hosts, vHost)
	routeConfiguration := &route.RouteConfiguration{
		Name:         resource.OutBoundRouteConfigName,
		VirtualHosts: hosts,
	}
	return append(routeConfs, routeConfiguration), nil
}

// makeGatewayRoutes builds the route.Route list for the envoy_gateway scenario
// In this scenario, it is mainly for the rule forwarding of path, /serviceA => serviceA
// Currently only routing rules that meet the following conditions support xds converted to envoy_gateway
// require 1: The calling service must match the GatewayService & GatewayNamespace in NodeProxy Metadata
// require 2: The $path parameter must be set in the request tag
// require 3: The information of the called service must be accurate, that is, a clear namespace and service
func (rds *RDSBuilder) makeGatewayRoutes(option *resource.BuildOption,
	xdsNode *resource.XDSClient) ([]*route.Route, error) {

	routes := make([]*route.Route, 0, 16)
	callerService := xdsNode.GetSelfService()
	callerNamespace := xdsNode.GetSelfNamespace()
	selfService := model.ServiceKey{
		Namespace: callerNamespace,
		Name:      callerService,
	}
	if !selfService.IsExact() {
		return nil, nil
	}

	routerCache := rds.svr.Cache().RoutingConfig()
	routerRules := routerCache.ListRouterRule(callerService, callerNamespace)
	for i := range routerRules {
		rule := routerRules[i]
		if rule.GetRoutingPolicy() != traffic_manage.RoutingPolicy_RulePolicy {
			continue
		}

		for i := range rule.RuleRouting.Rules {
			subRule := rule.RuleRouting.Rules[i]
			// 先判断 dest 的服务是否满足目标 namespace
			var (
				matchNamespace    bool
				findGatewaySource bool
			)
			for _, dest := range subRule.GetDestinations() {
				if dest.Namespace == callerNamespace && dest.Service != utils.MatchAll {
					matchNamespace = true
				}
			}
			if !matchNamespace {
				continue
			}

			routeMatch := &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{Prefix: "/"},
			}
			for _, source := range subRule.Sources {
				if !isMatchGatewaySource(source, callerService, callerNamespace) {
					continue
				}
				findGatewaySource = true
				buildGatewayRouteMatch(rds.client, routeMatch, source)
			}

			if !findGatewaySource {
				continue
			}

			gatewayRoute := resource.MakeGatewayRoute(corev3.TrafficDirection_OUTBOUND, routeMatch,
				subRule.GetDestinations())
			pathInfo := gatewayRoute.GetMatch().GetPath()
			if pathInfo == "" {
				pathInfo = gatewayRoute.GetMatch().GetSafeRegex().GetRegex()
			}

			seacher := rds.svr.Cache().RateLimit()
			limits, typedPerFilterConfig, err := resource.MakeGatewayLocalRateLimit(seacher, pathInfo, selfService)
			if err == nil {
				gatewayRoute.TypedPerFilterConfig = typedPerFilterConfig
				gatewayRoute.GetRoute().RateLimits = limits
			}
			routes = append(routes, gatewayRoute)
		}
	}

	routes = append(routes, &route.Route{
		Match: &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: "/",
			},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_Cluster{
					Cluster: resource.PassthroughClusterName,
				},
			},
		},
	})
	return routes, nil
}

func buildGatewayRouteMatch(client *resource.XDSClient, routeMatch *route.RouteMatch, source *traffic_manage.SourceService) {
	for i := range source.GetArguments() {
		argument := source.GetArguments()[i]
		if argument.Type == traffic_manage.SourceMatch_PATH {
			if argument.Value.Type == apimodel.MatchString_EXACT {
				routeMatch.PathSpecifier = &route.RouteMatch_Path{
					Path: argument.GetValue().GetValue().GetValue(),
				}
			} else if argument.Value.Type == apimodel.MatchString_REGEX {
				routeMatch.PathSpecifier = &route.RouteMatch_SafeRegex{
					SafeRegex: &v32.RegexMatcher{
						Regex: argument.GetValue().GetValue().GetValue(),
					},
				}
			}
		}
	}
	resource.BuildCommonRouteMatch(client, routeMatch, source)
}

func isMatchGatewaySource(source *traffic_manage.SourceService, svcName, svcNamespace string) bool {
	var (
		existPathLabel bool
		matchService   bool
	)

	args := source.GetArguments()
	for i := range args {
		if args[i].Type == traffic_manage.SourceMatch_PATH {
			existPathLabel = true
			break
		}
	}

	matchService = source.Service == svcName && source.Namespace == svcNamespace
	return existPathLabel && matchService
}
