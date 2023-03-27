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
	"strings"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	v32 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	"github.com/polarismesh/polaris/common/model"
	routercommon "github.com/polarismesh/polaris/common/routing"
	"github.com/polarismesh/polaris/common/utils"
)

// makeGatewaySnapshot nodeId must be like gateway~namespace
func (x *XDSServer) makeGatewaySnapshot(nodeId, version string, services []*ServiceInfo) (err error) {
	namespace := strings.Split(nodeId, "~")[1]

	resources := make(map[resource.Type][]types.Resource)
	resources[resource.EndpointType] = makeEndpoints(services)
	resources[resource.ClusterType] = x.makeClusters(services)
	resources[resource.RouteType] = x.makeGatewayVirtualHosts(namespace)
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

func (x *XDSServer) makeGatewayVirtualHosts(namespace string) []types.Resource {
	// 每个 polaris serviceInfo 对应一个 virtualHost
	var (
		routeConfs []types.Resource
		hosts      []*route.VirtualHost
	)

	vHost := &route.VirtualHost{
		Name:    "gateway-virtualhost",
		Domains: makeServiceGatewayDomains(),
		Routes:  x.makeGatewayRoutes(namespace),
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

// makeGatewayRoutes 构建用于 envoy_gateway 场景的 route.Route 列表
// 该场景下主要是针对 path 的规则转发，/serviceA => serviceA
// 当前只有满足以下条件的路由规则支持转为 envoy_gateway 的 xds
// require 1: 主调服务为全部命名空间&全部服务
// require 2: 请求标签中必须设置 $path 参数
func (x *XDSServer) makeGatewayRoutes(namespace string) []*route.Route {
	routes := make([]*route.Route, 0, 16)

	routerCache := x.namingServer.Cache().RoutingConfig()
	routerCache.IteratorRouterRule(func(_ string, rule *model.ExtendRouterConfig) {
		if !rule.Enable {
			return
		}
		if rule.GetRoutingPolicy() != traffic_manage.RoutingPolicy_RulePolicy {
			return
		}

		routeMatch := &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{Prefix: "/"},
		}

		for i := range rule.RuleRouting.Rules {
			subRule := rule.RuleRouting.Rules[i]
			// 先判断 dest 的服务是否满足目标 namespace
			var matchNamespace bool
			for _, dest := range subRule.GetDestinations() {
				if dest.Namespace == namespace {
					matchNamespace = true
				}
			}
			if !matchNamespace {
				continue
			}

			for _, source := range subRule.Sources {
				if !isMatchGatewaySource(source) {
					continue
				}

				v1source := &traffic_manage.Source{
					Namespace: utils.NewStringValue(source.Namespace),
					Service:   utils.NewStringValue(source.Service),
					Metadata:  routercommon.RoutingArguments2Labels(source.GetArguments()),
				}
				buildGatewayRouteMatch(routeMatch, v1source)
			}

			totalWeight, weightedClusters := buildWeightClustersV2(subRule.GetDestinations())
			route := &route.Route{
				Match: routeMatch,
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_WeightedClusters{
							WeightedClusters: &route.WeightedCluster{
								TotalWeight: &wrappers.UInt32Value{Value: totalWeight},
								Clusters:    weightedClusters,
							},
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

func buildGatewayRouteMatch(routeMatch *route.RouteMatch, source *traffic_manage.Source) {
	for name, matchString := range source.Metadata {
		if name == model.LabelKeyPath {
			if matchString.Type == apimodel.MatchString_EXACT {
				routeMatch.PathSpecifier = &route.RouteMatch_Path{
					Path: matchString.GetValue().GetValue()}
			} else if matchString.Type == apimodel.MatchString_REGEX {
				routeMatch.PathSpecifier = &route.RouteMatch_SafeRegex{SafeRegex: &v32.RegexMatcher{
					Regex: matchString.GetValue().GetValue()}}
			}
		}
	}
	buildCommonRouteMatch(routeMatch, source)
}

func buildCommonRouteMatch(routeMatch *route.RouteMatch, source *traffic_manage.Source) {
	for name, matchString := range source.Metadata {
		if strings.HasPrefix(name, model.LabelKeyHeader) {
			headerSubName := name[len(model.LabelKeyHeader):]
			if !(len(headerSubName) > 1 && strings.HasPrefix(headerSubName, ".")) {
				continue
			}
			headerSubName = headerSubName[1:]
			var headerMatch *route.HeaderMatcher
			if matchString.Type == apimodel.MatchString_EXACT {
				headerMatch = &route.HeaderMatcher{
					Name: headerSubName,
					HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
						StringMatch: &v32.StringMatcher{
							MatchPattern: &v32.StringMatcher_Exact{
								Exact: matchString.GetValue().GetValue()}},
					},
				}
			}
			if matchString.Type == apimodel.MatchString_NOT_EQUALS {
				headerMatch = &route.HeaderMatcher{
					Name: headerSubName,
					HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
						StringMatch: &v32.StringMatcher{
							MatchPattern: &v32.StringMatcher_Exact{
								Exact: matchString.GetValue().GetValue()}},
					},
					InvertMatch: true,
				}
			}
			if matchString.Type == apimodel.MatchString_REGEX {
				headerMatch = &route.HeaderMatcher{
					Name: headerSubName,
					HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
						StringMatch: &v32.StringMatcher{MatchPattern: &v32.StringMatcher_SafeRegex{
							SafeRegex: &v32.RegexMatcher{
								EngineType: &v32.RegexMatcher_GoogleRe2{
									GoogleRe2: &v32.RegexMatcher_GoogleRE2{}},
								Regex: matchString.GetValue().GetValue()}}},
					},
				}
			}
			if headerMatch != nil {
				routeMatch.Headers = append(routeMatch.Headers, headerMatch)
			}
		} else if strings.HasPrefix(name, model.LabelKeyQuery) {
			querySubName := name[len(model.LabelKeyQuery):]
			if !(len(querySubName) > 1 && strings.HasPrefix(querySubName, ".")) {
				continue
			}
			querySubName = querySubName[1:]
			var queryMatcher *route.QueryParameterMatcher
			if matchString.Type == apimodel.MatchString_EXACT {
				queryMatcher = &route.QueryParameterMatcher{
					Name: querySubName,
					QueryParameterMatchSpecifier: &route.QueryParameterMatcher_StringMatch{
						StringMatch: &v32.StringMatcher{
							MatchPattern: &v32.StringMatcher_Exact{
								Exact: matchString.GetValue().GetValue()}},
					},
				}
			}
			if matchString.Type == apimodel.MatchString_REGEX {
				queryMatcher = &route.QueryParameterMatcher{
					Name: querySubName,
					QueryParameterMatchSpecifier: &route.QueryParameterMatcher_StringMatch{
						StringMatch: &v32.StringMatcher{
							MatchPattern: &v32.StringMatcher_SafeRegex{SafeRegex: &v32.RegexMatcher{
								EngineType: &v32.RegexMatcher_GoogleRe2{
									GoogleRe2: &v32.RegexMatcher_GoogleRE2{}},
								Regex: matchString.GetValue().GetValue(),
							}}},
					},
				}
			}
			if queryMatcher != nil {
				routeMatch.QueryParameters = append(routeMatch.QueryParameters, queryMatcher)
			}
		}
	}
}

func buildWeightClustersV2(destinations []*traffic_manage.DestinationGroup) (uint32,
	[]*route.WeightedCluster_ClusterWeight) {
	var (
		weightedClusters []*route.WeightedCluster_ClusterWeight
		totalWeight      uint32
	)

	// 使用 destinations 生成 weightedClusters。makeClusters() 也使用这个字段生成对应的 subset
	for _, destination := range destinations {
		fields := make(map[string]*_struct.Value)
		for k, v := range destination.GetLabels() {
			fields[k] = &_struct.Value{
				Kind: &_struct.Value_StringValue{
					StringValue: v.Value.Value,
				},
			}
		}

		weightedClusters = append(weightedClusters, &route.WeightedCluster_ClusterWeight{
			Name:   destination.Service,
			Weight: utils.NewUInt32Value(destination.GetWeight()),
			MetadataMatch: &core.Metadata{
				FilterMetadata: map[string]*_struct.Struct{
					"envoy.lb": {
						Fields: fields,
					},
				},
			},
		})
		totalWeight += destination.Weight
	}

	return totalWeight, weightedClusters
}

func isMatchGatewaySource(source *traffic_manage.SourceService) bool {
	var (
		existPathLabel bool
		isMatchAll     bool
	)

	args := source.GetArguments()
	for i := range args {
		if args[i].Type == traffic_manage.SourceMatch_PATH {
			existPathLabel = true
			break
		}
	}

	isMatchAll = source.Service == utils.MatchAll && source.Namespace == utils.MatchAll
	return existPathLabel && isMatchAll
}
