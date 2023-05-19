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
	"context"
	"fmt"
	"strconv"
	"time"

	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	v32 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

func (x *XDSServer) pushGatewayInfoToXDSCache(registryInfo map[string][]*ServiceInfo) error {
	nodes := x.xdsNodesMgr.ListGatewayNodes()
	for i := range nodes {
		node := nodes[i]
		_ = x.buildGatewayXDSCache(nodes[i], registryInfo[node.Namespace])
	}
	return nil
}

func (x *XDSServer) buildGatewayXDSCache(xdsNode *XDSClient, services []*ServiceInfo) error {
	if !xdsNode.IsGateway() {
		return fmt.Errorf("xds node=%s run type not gateway or info is invalid", xdsNode.Node.Id)
	}

	if len(services) == 0 {
		registryInfo := map[string][]*ServiceInfo{
			xdsNode.Namespace: {},
		}
		_ = x.getRegistryInfoWithCache(context.Background(), registryInfo)
		services = registryInfo[xdsNode.Namespace]
	}

	versionLocal := time.Now().Format(time.RFC3339) + "/" + strconv.FormatUint(x.versionNum.Inc(), 10)
	_ = x.makeGatewaySnapshot(xdsNode, versionLocal, services)
	return nil
}

// makeGatewaySnapshot nodeId must be like gateway~namespace
func (x *XDSServer) makeGatewaySnapshot(xdsNode *XDSClient, version string, services []*ServiceInfo) (err error) {
	namespace := xdsNode.Namespace
	nodeId := xdsNode.Node.Id

	resources := make(map[resource.Type][]types.Resource)
	resources[resource.EndpointType] = makeEndpoints(services)
	resources[resource.ClusterType] = x.makeClusters(services)
	resources[resource.RouteType] = x.makeGatewayVirtualHosts(namespace, xdsNode)
	resources[resource.ListenerType] = makeListeners()
	snapshot, err := cachev3.NewSnapshot(version, resources)
	if err != nil {
		log.Errorf("[XDS][Gateway] fail to create snapshot for %s, err is %v", nodeId, err)
		return err
	}
	if err = snapshot.Consistent(); err != nil {
		return err
	}
	log.Infof("[XDS][Gateway] will serve ns: %s ,snapshot: %+v", nodeId, string(dumpSnapShotJSON(snapshot)))
	// 为每个 nodeId 刷写 cache ，推送 xds 更新
	if err := x.cache.SetSnapshot(context.Background(), nodeId, snapshot); err != nil {
		log.Errorf("[XDS][Gateway] snapshot error %q for %+v", err, snapshot)
		return err
	}
	return
}

func makeServiceGatewayDomains() []string {
	return []string{"*"}
}

func (x *XDSServer) makeGatewayVirtualHosts(namespace string, xdsNode *XDSClient) []types.Resource {
	// 每个 polaris serviceInfo 对应一个 virtualHost
	var (
		routeConfs []types.Resource
		hosts      []*route.VirtualHost
	)

	vHost := &route.VirtualHost{
		Name:    "gateway-virtualhost",
		Domains: makeServiceGatewayDomains(),
		Routes:  x.makeGatewayRoutes(namespace, xdsNode),
	}
	hosts = append(hosts, vHost)

	routeConfiguration := &route.RouteConfiguration{
		Name: "polaris-router",
		ValidateClusters: &wrappers.BoolValue{
			Value: false,
		},
		VirtualHosts: hosts,
	}

	return append(routeConfs, routeConfiguration)
}

// makeGatewayRoutes builds the route.Route list for the envoy_gateway scenario
// In this scenario, it is mainly for the rule forwarding of path, /serviceA => serviceA
// Currently only routing rules that meet the following conditions support xds converted to envoy_gateway
// require 1: The calling service must match the GatewayService & GatewayNamespace in NodeProxy Metadata
// require 2: The $path parameter must be set in the request tag
// require 3: The information of the called service must be accurate, that is, a clear namespace and service
func (x *XDSServer) makeGatewayRoutes(namespace string, xdsNode *XDSClient) []*route.Route {
	routes := make([]*route.Route, 0, 16)

	callerService := xdsNode.Metadata[GatewayServiceName]
	callerNamespace := xdsNode.Metadata[GatewayNamespaceName]

	routerCache := x.namingServer.Cache().RoutingConfig()
	routerCache.IteratorRouterRule(func(_ string, rule *model.ExtendRouterConfig) {
		if !rule.Enable {
			return
		}
		if rule.GetRoutingPolicy() != traffic_manage.RoutingPolicy_RulePolicy {
			return
		}

		for i := range rule.RuleRouting.Rules {
			routeMatch := &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{Prefix: "/"},
			}
			subRule := rule.RuleRouting.Rules[i]
			// 先判断 dest 的服务是否满足目标 namespace
			var (
				matchNamespace    bool
				findGatewaySource bool
			)
			for _, dest := range subRule.GetDestinations() {
				if dest.Namespace == namespace && dest.Service != utils.MatchAll {
					matchNamespace = true
				}
			}
			if !matchNamespace {
				continue
			}

			for _, source := range subRule.Sources {
				if !isMatchGatewaySource(source, callerService, callerNamespace) {
					continue
				}
				findGatewaySource = true
				buildGatewayRouteMatch(routeMatch, source)
			}

			if !findGatewaySource {
				continue
			}
			route := &route.Route{
				Match: routeMatch,
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_WeightedClusters{
							WeightedClusters: buildWeightClustersV2(subRule.GetDestinations()),
						},
					},
				},
			}
			routes = append(routes, route)
		}
	})

	routes = append(routes, &route.Route{
		Match: &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: "/",
			},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_Cluster{
					Cluster: "PassthroughCluster",
				},
			},
		},
	})

	return routes
}

func buildGatewayRouteMatch(routeMatch *route.RouteMatch, source *traffic_manage.SourceService) {
	for i := range source.GetArguments() {
		argument := source.GetArguments()[i]
		if argument.Type == traffic_manage.SourceMatch_PATH {
			if argument.Value.Type == apimodel.MatchString_EXACT {
				routeMatch.PathSpecifier = &route.RouteMatch_Path{
					Path: argument.GetValue().GetValue().GetValue()}
			} else if argument.Value.Type == apimodel.MatchString_REGEX {
				routeMatch.PathSpecifier = &route.RouteMatch_SafeRegex{SafeRegex: &v32.RegexMatcher{
					Regex: argument.GetValue().GetValue().GetValue()}}
			}
		}
	}
	buildCommonRouteMatch(routeMatch, source)
}

func isMatchGatewaySource(source *traffic_manage.SourceService, service, namespace string) bool {
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

	matchService = source.Service == service && source.Namespace == namespace
	return existPathLabel && matchService
}
