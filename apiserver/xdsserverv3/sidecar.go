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
	"encoding/json"
	"strconv"
	"strings"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_common_ratelimit_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	lrl "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/local_ratelimit/v3"
	v32 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	envoy_type_v3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/ptypes"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

func (x *XDSServer) pushSidecarInfoToXDSCache(registryInfo map[string][]*ServiceInfo) error {
	versionLocal := time.Now().Format(time.RFC3339) + "/" + strconv.FormatUint(x.versionNum.Inc(), 10)
	for ns, services := range registryInfo {
		_ = x.makeSnapshot(ns, versionLocal, services)
		_ = x.makePermissiveSnapshot(ns, versionLocal, services)
		_ = x.makeStrictSnapshot(ns, versionLocal, services)
	}
	return nil
}

func (x *XDSServer) makeSnapshot(ns, version string, services []*ServiceInfo) (err error) {
	resources := make(map[resource.Type][]types.Resource)
	resources[resource.EndpointType] = makeEndpoints(services)
	resources[resource.ClusterType] = x.makeClusters(services)
	resources[resource.RouteType] = x.makeSidecarVirtualHosts(services)
	resources[resource.ListenerType] = makeListeners()
	snapshot, err := cachev3.NewSnapshot(version, resources)
	if err != nil {
		log.Errorf("[XDS][Sidecar] fail to create snapshot for %s, err is %v", ns, err)
		return err
	}
	err = snapshot.Consistent()
	if err != nil {
		return err
	}
	log.Infof("[XDS][Sidecar] will serve ns: %s ,snapshot: %+v", ns, string(dumpSnapShotJSON(snapshot)))
	// 为每个 ns 刷写 cache ，推送 xds 更新
	if err := x.cache.SetSnapshot(context.Background(), ns, snapshot); err != nil {
		log.Errorf("[XDS][Sidecar] snapshot error %q for %+v", err, snapshot)
		return err
	}
	return
}

func (x *XDSServer) makePermissiveSnapshot(ns, version string, services []*ServiceInfo) (err error) {
	resources := make(map[resource.Type][]types.Resource)
	resources[resource.EndpointType] = makeEndpoints(services)
	resources[resource.ClusterType] = x.makePermissiveClusters(services)
	resources[resource.RouteType] = x.makeSidecarVirtualHosts(services)
	resources[resource.ListenerType] = makePermissiveListeners()
	snapshot, err := cachev3.NewSnapshot(version, resources)
	if err != nil {
		return err
	}
	err = snapshot.Consistent()
	if err != nil {
		return err
	}
	log.Infof("[XDS][Sidecar] will serve ns: %s ,mode permissive,snapshot: %+v", ns, string(dumpSnapShotJSON(snapshot)))
	// 为每个 ns 刷写 cache ，推送 xds 更新
	if err := x.cache.SetSnapshot(context.Background(), ns+"/permissive", snapshot); err != nil {
		log.Errorf("[XDS][Sidecar] snapshot error %q for %+v", err, snapshot)
		return err
	}
	return
}

func (x *XDSServer) makeStrictSnapshot(ns, version string, services []*ServiceInfo) (err error) {
	resources := make(map[resource.Type][]types.Resource)
	resources[resource.EndpointType] = makeEndpoints(services)
	resources[resource.ClusterType] = x.makeStrictClusters(services)
	resources[resource.RouteType] = x.makeSidecarVirtualHosts(services)
	resources[resource.ListenerType] = makeStrictListeners()
	snapshot, err := cachev3.NewSnapshot(version, resources)
	if err != nil {
		return err
	}
	if err = snapshot.Consistent(); err != nil {
		return err
	}
	log.Infof("[XDS][Sidecar] will serve ns: %s ,mode strict,snapshot: %+v", ns, string(dumpSnapShotJSON(snapshot)))
	// 为每个 ns 刷写 cache ，推送 xds 更新
	if err := x.cache.SetSnapshot(context.Background(), ns+"/strict", snapshot); err != nil {
		log.Errorf("[XDS][Sidecar] snapshot error %q for %+v", err, snapshot)
		return err
	}
	return
}

func buildSidecarRouteMatch(routeMatch *route.RouteMatch, source *traffic_manage.SourceService) {
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

func (x *XDSServer) makeLocalRateLimit(svcKey model.ServiceKey) map[string]*anypb.Any {
	ratelimitGetter := x.RatelimitConfigGetter
	if ratelimitGetter == nil {
		ratelimitGetter = x.namingServer.Cache().RateLimit().GetRateLimitRules
	}
	conf, _ := ratelimitGetter(svcKey)
	filters := make(map[string]*anypb.Any)
	if conf != nil {
		rateLimitConf := &lrl.LocalRateLimit{
			StatPrefix: "http_local_rate_limiter",
			// TokenBucket: &envoy_type_v3.TokenBucket{
			// 	MaxTokens:    rule.Amounts[0].MaxAmount.Value,
			// 	FillInterval: rule.Amounts[0].ValidDuration,
			// },
		}
		rateLimitConf.FilterEnabled = &core.RuntimeFractionalPercent{
			RuntimeKey: "local_rate_limit_enabled",
			DefaultValue: &envoy_type_v3.FractionalPercent{
				Numerator:   uint32(100),
				Denominator: envoy_type_v3.FractionalPercent_HUNDRED,
			},
		}
		rateLimitConf.FilterEnforced = &core.RuntimeFractionalPercent{
			RuntimeKey: "local_rate_limit_enforced",
			DefaultValue: &envoy_type_v3.FractionalPercent{
				Numerator:   uint32(100),
				Denominator: envoy_type_v3.FractionalPercent_HUNDRED,
			},
		}
		for _, c := range conf {
			rlRule := c.Rule
			rlLabels := c.Labels
			if rlRule == "" {
				continue
			}
			rule := new(apitraffic.Rule)
			if err := json.Unmarshal([]byte(rlRule), rule); err != nil {
				log.Errorf("unmarshal local rate limit rule error,%v", err)
				continue
			}
			if len(rlRule) > 0 {
				if err := json.Unmarshal([]byte(rlLabels), &rule.Labels); err != nil {
					log.Errorf("unmarshal local rate limit labels error,%v", err)
				}
			}

			// 跳过全局限流配置
			if rule.Type == apitraffic.Rule_GLOBAL || rule.Disable.Value {
				continue
			}

			for _, amount := range rule.Amounts {
				descriptor := &envoy_extensions_common_ratelimit_v3.LocalRateLimitDescriptor{
					TokenBucket: &envoy_type_v3.TokenBucket{
						MaxTokens:    amount.MaxAmount.Value,
						FillInterval: amount.ValidDuration,
					},
				}
				entries := make([]*envoy_extensions_common_ratelimit_v3.RateLimitDescriptor_Entry, len(rule.Labels))
				pos := 0
				for k, v := range rule.Labels {
					entries[pos] = &envoy_extensions_common_ratelimit_v3.RateLimitDescriptor_Entry{
						Key:   k,
						Value: v.Value.Value,
					}
					pos++
				}
				descriptor.Entries = entries
				rateLimitConf.Descriptors = append(rateLimitConf.Descriptors, descriptor)
			}
			if rule.AmountMode == apitraffic.Rule_GLOBAL_TOTAL {
				rateLimitConf.LocalRateLimitPerDownstreamConnection = true
			}
		}
		if len(rateLimitConf.Descriptors) == 0 {
			return nil
		}
		pbst, err := ptypes.MarshalAny(rateLimitConf)
		if err != nil {
			panic(err)
		}
		filters["envoy.filters.http.local_ratelimit"] = pbst
		return filters
	}
	return nil
}

type (
	ServiceDomainBuilder   func(*ServiceInfo) []string
	RoutesBuilder          func(*ServiceInfo) []*route.Route
	PerFilterConfigBuilder func(*ServiceInfo) map[string]*anypb.Any
)

func (x *XDSServer) makeSidecarVirtualHosts(services []*ServiceInfo) []types.Resource {
	// 每个 polaris serviceInfo 对应一个 virtualHost
	var (
		routeConfs []types.Resource
		hosts      []*route.VirtualHost
	)

	for _, serviceInfo := range services {
		vHost := &route.VirtualHost{
			Name:    serviceInfo.Name,
			Domains: generateServiceDomains(serviceInfo),
			Routes:  makeSidecarRoutes(serviceInfo),
			// TypedPerFilterConfig: x.makeLocalRateLimit(model.ServiceKey{
			// 	Namespace: serviceInfo.Namespace,
			// 	Name:      serviceInfo.Name,
			// }),
		}
		hosts = append(hosts, vHost)
	}

	// 最后是 allow_any
	hosts = append(hosts, buildAllowAnyVHost())

	routeConfiguration := &route.RouteConfiguration{
		Name: "polaris-router",
		ValidateClusters: &wrappers.BoolValue{
			Value: false,
		},
		VirtualHosts: hosts,
	}

	return append(routeConfs, routeConfiguration)
}

func makeLbSubsetConfig(serviceInfo *ServiceInfo) *cluster.Cluster_LbSubsetConfig {
	rules := filterInboundRouterRule(serviceInfo)
	if len(rules) == 0 {
		return nil
	}

	lbSubsetConfig := &cluster.Cluster_LbSubsetConfig{}
	var subsetSelectors []*cluster.Cluster_LbSubsetConfig_LbSubsetSelector
	lbSubsetConfig.FallbackPolicy = cluster.Cluster_LbSubsetConfig_ANY_ENDPOINT

	for _, rule := range rules {
		// 对每一个 destination 产生一个 subset
		for _, destination := range rule.GetDestinations() {
			var keys []string
			for s := range destination.GetLabels() {
				keys = append(keys, s)
			}
			subsetSelectors = append(subsetSelectors, &cluster.Cluster_LbSubsetConfig_LbSubsetSelector{
				Keys:           keys,
				FallbackPolicy: cluster.Cluster_LbSubsetConfig_LbSubsetSelector_NO_FALLBACK,
			})
		}
	}

	lbSubsetConfig.SubsetSelectors = subsetSelectors
	return lbSubsetConfig
}

// Translate the circuit breaker configuration of Polaris into OutlierDetection
func makeOutlierDetection(conf *model.ServiceWithCircuitBreaker) *cluster.OutlierDetection {
	if conf != nil {
		cbRules := conf.CircuitBreaker.Inbounds
		if cbRules == "" {
			return nil
		}

		var inBounds []*apifault.CbRule
		if err := json.Unmarshal([]byte(cbRules), &inBounds); err != nil {
			log.Errorf("unmarshal inbounds circuitBreaker rule error, %v", err)
			return nil
		}

		if len(inBounds) == 0 || len(inBounds[0].GetDestinations()) == 0 ||
			inBounds[0].GetDestinations()[0].Policy == nil {
			return nil
		}

		var (
			consecutiveErrConfig *apifault.CbPolicy_ConsecutiveErrConfig
			errorRateConfig      *apifault.CbPolicy_ErrRateConfig
			policy               *apifault.CbPolicy
			dest                 *apifault.DestinationSet
		)

		dest = inBounds[0].GetDestinations()[0]
		policy = dest.Policy
		consecutiveErrConfig = policy.Consecutive
		errorRateConfig = policy.ErrorRate

		outlierDetection := &cluster.OutlierDetection{}

		if consecutiveErrConfig != nil {
			outlierDetection.Consecutive_5Xx = &wrappers.UInt32Value{
				Value: consecutiveErrConfig.ConsecutiveErrorToOpen.Value}
		}
		if errorRateConfig != nil {
			outlierDetection.FailurePercentageRequestVolume = &wrappers.UInt32Value{
				Value: errorRateConfig.RequestVolumeThreshold.Value}
			outlierDetection.FailurePercentageThreshold = &wrappers.UInt32Value{
				Value: errorRateConfig.ErrorRateToOpen.Value}
		}

		return outlierDetection
	}
	return nil
}

func getEndpointMetaFromPolarisIns(ins *apiservice.Instance) *core.Metadata {
	meta := &core.Metadata{}
	fields := make(map[string]*_struct.Value)
	for k, v := range ins.Metadata {
		fields[k] = &_struct.Value{
			Kind: &_struct.Value_StringValue{
				StringValue: v,
			},
		}
	}

	meta.FilterMetadata = make(map[string]*_struct.Struct)
	meta.FilterMetadata["envoy.lb"] = &_struct.Struct{
		Fields: fields,
	}
	if ins.Metadata != nil && ins.Metadata[TLSModeTag] != "" {
		meta.FilterMetadata["envoy.transport_socket_match"] = mtlsTransportSocketMatch
	}
	return meta
}

func isNormalEndpoint(ins *apiservice.Instance) bool {
	if ins.GetIsolate().GetValue() {
		return false
	}
	if ins.GetWeight().GetValue() == 0 {
		return false
	}
	return true
}

func formatEndpointHealth(ins *apiservice.Instance) core.HealthStatus {
	if ins.GetHealthy().GetValue() {
		return core.HealthStatus_HEALTHY
	}
	return core.HealthStatus_UNHEALTHY
}

func makeEndpoints(services []*ServiceInfo) []types.Resource {
	var clusterLoads []types.Resource
	for _, serviceInfo := range services {
		var lbEndpoints []*endpoint.LbEndpoint
		for _, instance := range serviceInfo.Instances {
			// 只加入健康的实例
			if !isNormalEndpoint(instance) {
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
				HealthStatus:        formatEndpointHealth(instance),
				LoadBalancingWeight: utils.NewUInt32Value(instance.GetWeight().GetValue()),
				Metadata:            getEndpointMetaFromPolarisIns(instance),
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
func makeSidecarRoutes(serviceInfo *ServiceInfo) []*route.Route {
	var (
		routes        []*route.Route
		matchAllRoute *route.Route
	)
	// 路由目前只处理 inbounds
	rules := filterInboundRouterRule(serviceInfo)
	for _, rule := range rules {
		var (
			matchAll     bool
			destinations []*traffic_manage.DestinationGroup
		)
		for _, dest := range rule.GetDestinations() {
			if !serviceInfo.matchService(dest.GetNamespace(), dest.GetService()) {
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
				buildSidecarRouteMatch(routeMatch, source)
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

func filterInboundRouterRule(svc *ServiceInfo) []*traffic_manage.SubRuleRouting {
	ret := make([]*traffic_manage.SubRuleRouting, 0, 16)
	for _, rule := range svc.Routing.GetRules() {
		if rule.GetRoutingPolicy() != traffic_manage.RoutingPolicy_RulePolicy {
			continue
		}
		routerRule := &traffic_manage.RuleRoutingConfig{}
		if err := ptypes.UnmarshalAny(rule.RoutingConfig, routerRule); err != nil {
			continue
		}

		for i, subRule := range routerRule.Rules {
			var match bool
			for _, dest := range subRule.GetDestinations() {
				if svc.matchService(dest.GetNamespace(), dest.GetService()) {
					match = true
					break
				}
			}
			if match {
				ret = append(ret, routerRule.Rules[i])
			}
		}
	}
	return ret
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

func generateServiceDomains(serviceInfo *ServiceInfo) []string {
	var domains []string

	// 只有服务名
	domains = append(domains, serviceInfo.Name)

	// k8s dns 可解析的服务名
	domain := serviceInfo.Name + "." + serviceInfo.Namespace
	domains = append(append(append(append(domains, domain),
		domain+K8sDnsResolveSuffixSvc),
		domain+K8sDnsResolveSuffixSvcCluster),
		domain+K8sDnsResolveSuffixSvcClusterLocal)

	resDomains := domains
	// 上面各种服务名加服务端口
	portsStr := serviceInfo.Ports
	ports := strings.Split(portsStr, ",")
	for _, port := range ports {
		if _, err := strconv.Atoi(port); err == nil {
			// 如果是数字，则为每个域名产生一个带端口的域名
			for _, s := range domains {
				resDomains = append(resDomains, s+":"+port)
			}
		}
	}
	return resDomains
}

func buildAllowAnyVHost() *route.VirtualHost {
	return &route.VirtualHost{
		Name:    "allow_any",
		Domains: []string{"*"},
		Routes: []*route.Route{
			{
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
			},
		},
	}
}

func buildWeightClustersForSidecar(svcInfo *ServiceInfo,
	destinations []*traffic_manage.DestinationGroup) *route.WeightedCluster {
	weightClusters := buildWeightClustersV2(destinations)
	for i := range weightClusters.Clusters {
		weightClusters.Clusters[i].Name = svcInfo.Name
	}
	return weightClusters
}
