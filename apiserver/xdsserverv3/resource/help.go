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

package resource

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	accesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	filev3 "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	envoy_extensions_common_ratelimit_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	ratelimitv32 "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	lrl "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/local_ratelimit/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcp "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	v32 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	envoy_type_v3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/polarismesh/polaris/cache"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	PassthroughClusterName = "PassthroughCluster"
	RouteConfigName        = "polaris-router"
)

func MakeServiceGatewayDomains() []string {
	return []string{"*"}
}

func FilterInboundRouterRule(svc *ServiceInfo) []*traffic_manage.SubRuleRouting {
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
				if svc.MatchService(dest.GetNamespace(), dest.GetService()) {
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

func BuildSidecarRouteMatch(routeMatch *route.RouteMatch, source *traffic_manage.SourceService) {
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
	BuildCommonRouteMatch(routeMatch, source)
}

func BuildCommonRouteMatch(routeMatch *route.RouteMatch, source *traffic_manage.SourceService) {
	for i := range source.GetArguments() {
		argument := source.GetArguments()[i]
		switch argument.Type {
		case traffic_manage.SourceMatch_HEADER:
			headerSubName := argument.Key
			var headerMatch *route.HeaderMatcher
			if argument.Value.Type == apimodel.MatchString_EXACT {
				headerMatch = &route.HeaderMatcher{
					Name: headerSubName,
					HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
						StringMatch: &v32.StringMatcher{
							MatchPattern: &v32.StringMatcher_Exact{
								Exact: argument.GetValue().GetValue().GetValue()}},
					},
				}
			}
			if argument.Value.Type == apimodel.MatchString_NOT_EQUALS {
				headerMatch = &route.HeaderMatcher{
					Name: headerSubName,
					HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
						StringMatch: &v32.StringMatcher{
							MatchPattern: &v32.StringMatcher_Exact{
								Exact: argument.GetValue().GetValue().GetValue()}},
					},
					InvertMatch: true,
				}
			}
			if argument.Value.Type == apimodel.MatchString_REGEX {
				headerMatch = &route.HeaderMatcher{
					Name: headerSubName,
					HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
						StringMatch: &v32.StringMatcher{MatchPattern: &v32.StringMatcher_SafeRegex{
							SafeRegex: &v32.RegexMatcher{
								EngineType: &v32.RegexMatcher_GoogleRe2{
									GoogleRe2: &v32.RegexMatcher_GoogleRE2{}},
								Regex: argument.GetValue().GetValue().GetValue()}}},
					},
				}
			}
			if headerMatch != nil {
				routeMatch.Headers = append(routeMatch.Headers, headerMatch)
			}
		case traffic_manage.SourceMatch_QUERY:
			querySubName := argument.Key
			var queryMatcher *route.QueryParameterMatcher
			if argument.Value.Type == apimodel.MatchString_EXACT {
				queryMatcher = &route.QueryParameterMatcher{
					Name: querySubName,
					QueryParameterMatchSpecifier: &route.QueryParameterMatcher_StringMatch{
						StringMatch: &v32.StringMatcher{
							MatchPattern: &v32.StringMatcher_Exact{
								Exact: argument.GetValue().GetValue().GetValue()}},
					},
				}
			}
			if argument.Value.Type == apimodel.MatchString_REGEX {
				queryMatcher = &route.QueryParameterMatcher{
					Name: querySubName,
					QueryParameterMatchSpecifier: &route.QueryParameterMatcher_StringMatch{
						StringMatch: &v32.StringMatcher{
							MatchPattern: &v32.StringMatcher_SafeRegex{SafeRegex: &v32.RegexMatcher{
								EngineType: &v32.RegexMatcher_GoogleRe2{
									GoogleRe2: &v32.RegexMatcher_GoogleRE2{}},
								Regex: argument.GetValue().GetValue().GetValue(),
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

func BuildWeightClustersV2(trafficDirection corev3.TrafficDirection,
	destinations []*traffic_manage.DestinationGroup) *route.WeightedCluster {
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
			Name: MakeServiceName(model.ServiceKey{
				Namespace: destination.Namespace,
				Name:      destination.Service,
			}, trafficDirection),
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

func BuildRateLimitConf(prefix string) *lrl.LocalRateLimit {
	rateLimitConf := &lrl.LocalRateLimit{
		StatPrefix: prefix,
		// 默认全局限流没限制，由于 envoy 这里必须设置一个 TokenBucket，因此这里只能设置一个认为不可能达到的一个 TPS 进行实现不限流
		// TPS = 4294967295/s
		TokenBucket: &typev3.TokenBucket{
			MaxTokens:     math.MaxUint32,
			TokensPerFill: wrapperspb.UInt32(math.MaxUint32),
			FillInterval:  durationpb.New(time.Second),
		},
		FilterEnabled: &core.RuntimeFractionalPercent{
			RuntimeKey: prefix + "_local_rate_limit_enabled",
			DefaultValue: &envoy_type_v3.FractionalPercent{
				Numerator:   uint32(100),
				Denominator: envoy_type_v3.FractionalPercent_HUNDRED,
			},
		},
		FilterEnforced: &core.RuntimeFractionalPercent{
			RuntimeKey: prefix + "_local_rate_limit_enforced",
			DefaultValue: &envoy_type_v3.FractionalPercent{
				Numerator:   uint32(100),
				Denominator: envoy_type_v3.FractionalPercent_HUNDRED,
			},
		},
		ResponseHeadersToAdd: []*core.HeaderValueOption{
			{
				Header: &core.HeaderValue{
					Key:   "x-local-rate-limit",
					Value: "true",
				},
				Append: wrapperspb.Bool(false),
			},
		},
		LocalRateLimitPerDownstreamConnection: true,
	}
	return rateLimitConf
}

func BuildLocalRateLimitDescriptors(rule *traffic_manage.Rule) ([]*route.RateLimit_Action,
	[]*ratelimitv32.LocalRateLimitDescriptor) {
	actions := make([]*route.RateLimit_Action, 0, 8)
	descriptors := make([]*ratelimitv32.LocalRateLimitDescriptor, 0, 8)
	for _, amount := range rule.Amounts {
		descriptor := &envoy_extensions_common_ratelimit_v3.LocalRateLimitDescriptor{
			TokenBucket: &envoy_type_v3.TokenBucket{
				MaxTokens:     amount.GetMaxAmount().GetValue(),
				TokensPerFill: wrapperspb.UInt32(amount.GetMaxAmount().GetValue()),
				FillInterval:  amount.GetValidDuration(),
			},
		}
		entries := make([]*envoy_extensions_common_ratelimit_v3.RateLimitDescriptor_Entry, 0, len(rule.Labels))
		if len(rule.GetMethod().GetValue().GetValue()) != 0 {
			actions = append(actions, &route.RateLimit_Action{
				ActionSpecifier: &route.RateLimit_Action_HeaderValueMatch_{
					HeaderValueMatch: BuildRateLimitActionHeaderValueMatch(":path", rule.GetMethod()),
				},
			})
			entries = append(entries, &ratelimitv32.RateLimitDescriptor_Entry{
				Key:   "header_match",
				Value: rule.GetMethod().GetValue().GetValue(),
			})
		}
		arguments := rule.GetArguments()

		for i := range arguments {
			arg := arguments[i]
			switch arg.Type {
			case apitraffic.MatchArgument_HEADER:
				headerValueMatch := BuildRateLimitActionHeaderValueMatch(arg.Key, arg.Value)
				actions = append(actions, &route.RateLimit_Action{
					ActionSpecifier: &route.RateLimit_Action_HeaderValueMatch_{
						HeaderValueMatch: headerValueMatch,
					},
				})
				entries = append(entries, &ratelimitv32.RateLimitDescriptor_Entry{
					Key:   "header_match",
					Value: arg.GetValue().GetValue().GetValue(),
				})
			case apitraffic.MatchArgument_QUERY:
				queryParameterValueMatch := BuildRateLimitActionQueryParameterValueMatch(arg.Key, arg.Value)
				actions = append(actions, &route.RateLimit_Action{
					ActionSpecifier: &route.RateLimit_Action_QueryParameterValueMatch_{
						QueryParameterValueMatch: queryParameterValueMatch,
					},
				})
				entries = append(entries, &ratelimitv32.RateLimitDescriptor_Entry{
					Key:   "query_match",
					Value: arg.GetValue().GetValue().GetValue(),
				})
			case apitraffic.MatchArgument_METHOD:
				actions = append(actions, &route.RateLimit_Action{
					ActionSpecifier: &route.RateLimit_Action_RequestHeaders_{
						RequestHeaders: &route.RateLimit_Action_RequestHeaders{
							HeaderName:    ":method",
							DescriptorKey: arg.Key,
						},
					},
				})
				entries = append(entries, &envoy_extensions_common_ratelimit_v3.RateLimitDescriptor_Entry{
					Key:   arg.Key,
					Value: arg.GetValue().GetValue().GetValue(),
				})
			case apitraffic.MatchArgument_CALLER_IP:
				actions = append(actions, &route.RateLimit_Action{
					ActionSpecifier: &route.RateLimit_Action_RemoteAddress_{
						RemoteAddress: &route.RateLimit_Action_RemoteAddress{},
					},
				})
			}
		}
		descriptor.Entries = entries
		descriptors = append(descriptors, descriptor)
	}
	return actions, descriptors
}

func BuildRateLimitActionQueryParameterValueMatch(key string,
	value *apimodel.MatchString) *route.RateLimit_Action_QueryParameterValueMatch {
	queryParameterValueMatch := &route.RateLimit_Action_QueryParameterValueMatch{
		DescriptorKey:   key,
		DescriptorValue: "true",
		ExpectMatch:     wrapperspb.Bool(true),
		QueryParameters: []*route.QueryParameterMatcher{},
	}
	switch value.GetType() {
	case apimodel.MatchString_EXACT:
		queryParameterValueMatch.QueryParameters = []*route.QueryParameterMatcher{
			{
				Name: key,
				QueryParameterMatchSpecifier: &route.QueryParameterMatcher_StringMatch{
					StringMatch: &v32.StringMatcher{
						MatchPattern: &v32.StringMatcher_Exact{
							Exact: value.GetValue().GetValue(),
						},
					},
				},
			},
		}
	case apimodel.MatchString_REGEX:
		queryParameterValueMatch.QueryParameters = []*route.QueryParameterMatcher{
			{
				Name: key,
				QueryParameterMatchSpecifier: &route.QueryParameterMatcher_StringMatch{
					StringMatch: &v32.StringMatcher{
						MatchPattern: &v32.StringMatcher_SafeRegex{
							SafeRegex: &v32.RegexMatcher{
								EngineType: &v32.RegexMatcher_GoogleRe2{},
								Regex:      value.GetValue().GetValue(),
							},
						},
					},
				},
			},
		}
	}

	return queryParameterValueMatch
}

func BuildRateLimitActionHeaderValueMatch(key string,
	value *apimodel.MatchString) *route.RateLimit_Action_HeaderValueMatch {
	headerValueMatch := &route.RateLimit_Action_HeaderValueMatch{
		DescriptorValue: value.GetValue().GetValue(),
		Headers:         []*route.HeaderMatcher{},
	}
	switch value.GetType() {
	case apimodel.MatchString_EXACT, apimodel.MatchString_NOT_EQUALS:
		headerValueMatch.Headers = []*route.HeaderMatcher{
			{
				Name:        key,
				InvertMatch: value.GetType() == apimodel.MatchString_NOT_EQUALS,
				HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
					StringMatch: &v32.StringMatcher{
						MatchPattern: &v32.StringMatcher_Exact{
							Exact: value.GetValue().GetValue(),
						},
					},
				},
			},
		}
	case apimodel.MatchString_REGEX:
		headerValueMatch.Headers = []*route.HeaderMatcher{
			{
				Name: key,
				HeaderMatchSpecifier: &route.HeaderMatcher_SafeRegexMatch{
					SafeRegexMatch: &v32.RegexMatcher{
						EngineType: &v32.RegexMatcher_GoogleRe2{},
						Regex:      value.GetValue().GetValue(),
					},
				},
			},
		}
	}
	return headerValueMatch
}

// 默认路由
func MakeDefaultRoute(trafficDirection corev3.TrafficDirection, svcKey model.ServiceKey) *route.Route {
	return &route.Route{
		Match: &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: "/",
			},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_Cluster{
					Cluster: MakeServiceName(svcKey, trafficDirection),
				},
			},
		},
	}
}

func GenerateServiceDomains(serviceInfo *ServiceInfo) []string {
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

func BuildAllowAnyVHost() *route.VirtualHost {
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
							Cluster: PassthroughClusterName,
						},
					},
				},
			},
		},
	}
}

func MakeGatewayRoute(trafficDirection corev3.TrafficDirection, routeMatch *route.RouteMatch,
	destinations []*traffic_manage.DestinationGroup) *route.Route {
	route := &route.Route{
		Match: routeMatch,
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_WeightedClusters{
					WeightedClusters: BuildWeightClustersV2(trafficDirection, destinations),
				},
			},
		},
	}
	return route
}

func MakeSidecarRoute(trafficDirection corev3.TrafficDirection, routeMatch *route.RouteMatch,
	svcInfo *ServiceInfo, destinations []*traffic_manage.DestinationGroup) *route.Route {
	weightClusters := BuildWeightClustersV2(trafficDirection, destinations)
	for i := range weightClusters.Clusters {
		weightClusters.Clusters[i].Name = MakeServiceName(svcInfo.ServiceKey, trafficDirection)
	}
	currentRoute := &route.Route{
		Match: routeMatch,
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_WeightedClusters{
					WeightedClusters: weightClusters,
				},
			},
		},
	}
	return currentRoute
}

var PassthroughCluster = &cluster.Cluster{
	Name:                 PassthroughClusterName,
	ConnectTimeout:       durationpb.New(5 * time.Second),
	ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_ORIGINAL_DST},
	LbPolicy:             cluster.Cluster_CLUSTER_PROVIDED,
	CircuitBreakers: &cluster.CircuitBreakers{
		Thresholds: []*cluster.CircuitBreakers_Thresholds{
			{
				MaxConnections:     &wrappers.UInt32Value{Value: 4294967295},
				MaxPendingRequests: &wrappers.UInt32Value{Value: 4294967295},
				MaxRequests:        &wrappers.UInt32Value{Value: 4294967295},
				MaxRetries:         &wrappers.UInt32Value{Value: 4294967295},
			},
		},
	},
}

func MakeServiceName(svcKey model.ServiceKey, trafficDirection corev3.TrafficDirection) string {
	return fmt.Sprintf("%s|%s|%s", corev3.TrafficDirection_name[int32(trafficDirection)],
		svcKey.Namespace, svcKey.Name)
}

func MakeDefaultFilterChain() *listenerv3.FilterChain {
	return &listenerv3.FilterChain{
		Name: "PassthroughFilterChain",
		Filters: []*listenerv3.Filter{
			{
				Name: wellknown.TCPProxy,
				ConfigType: &listenerv3.Filter_TypedConfig{
					TypedConfig: MustNewAny(&tcp.TcpProxy{
						StatPrefix: PassthroughClusterName,
						ClusterSpecifier: &tcp.TcpProxy_Cluster{
							Cluster: PassthroughClusterName,
						},
					}),
				},
			},
		},
	}
}

func MakeBoundHCM(trafficDirection corev3.TrafficDirection) *hcm.HttpConnectionManager {
	hcmFilters := []*hcm.HttpFilter{
		{
			Name: wellknown.Router,
		},
	}
	if trafficDirection == corev3.TrafficDirection_INBOUND {
		hcmFilters = append([]*hcm.HttpFilter{
			{
				Name: "envoy.filters.http.local_ratelimit",
				ConfigType: &hcm.HttpFilter_TypedConfig{
					TypedConfig: MustNewAny(&lrl.LocalRateLimit{
						StatPrefix: "http_local_rate_limiter",
					}),
				},
			},
		}, hcmFilters...)
	}

	trafficDirectionName := corev3.TrafficDirection_name[int32(trafficDirection)]
	manager := &hcm.HttpConnectionManager{
		CodecType:           hcm.HttpConnectionManager_AUTO,
		StatPrefix:          trafficDirectionName + "_HTTP",
		RouteSpecifier:      routeSpecifier(),
		AccessLog:           accessLog(),
		HttpFilters:         hcmFilters,
		HttpProtocolOptions: &core.Http1ProtocolOptions{AcceptHttp_10: true},
	}
	return manager
}

func routeSpecifier() *hcm.HttpConnectionManager_Rds {
	return &hcm.HttpConnectionManager_Rds{
		Rds: &hcm.Rds{
			ConfigSource: &core.ConfigSource{
				ResourceApiVersion: resourcev3.DefaultAPIVersion,
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
			RouteConfigName: RouteConfigName,
		},
	}
}

func accessLog() []*accesslog.AccessLog {
	return []*accesslog.AccessLog{
		{
			Name: wellknown.FileAccessLog,
			ConfigType: &accesslog.AccessLog_TypedConfig{
				TypedConfig: MustNewAny(&filev3.FileAccessLog{
					Path: "/dev/stdout",
				}),
			},
		},
	}
}

func MustNewAny(src proto.Message) *anypb.Any {
	a, _ := anypb.New(src)
	return a
}

func MakeGatewayLocalRateLimit(rateLimitCache cache.RateLimitCache, pathSpecifier string,
	svcKey model.ServiceKey) ([]*route.RateLimit, map[string]*anypb.Any, error) {
	conf, _ := rateLimitCache.GetRateLimitRules(svcKey)
	if conf == nil {
		return nil, nil, nil
	}
	confKey := fmt.Sprintf("INBOUND|GATEWAY|%s|%s|%s", svcKey.Namespace, svcKey.Name, pathSpecifier)
	rateLimitConf := BuildRateLimitConf(confKey)
	filters := make(map[string]*anypb.Any)
	ratelimits := make([]*route.RateLimit, 0, len(conf))
	for _, c := range conf {
		rule := c.Proto
		if rule == nil {
			continue
		}
		// 跳过全局限流配置
		// TODO 暂时不放开全局限流规则下发，后续等待 envoy polaris filter 插件开发或者在 polaris-sidecar 中实现 RLS 协议后
		// 在放开该设置
		if rule.GetType() == apitraffic.Rule_GLOBAL || rule.GetDisable().GetValue() {
			continue
		}
		if rule.GetMethod().GetValue().GetValue() != pathSpecifier {
			continue
		}
		actions, descriptors := BuildLocalRateLimitDescriptors(rule)
		rateLimitConf.Descriptors = descriptors
		ratelimits = append(ratelimits, &route.RateLimit{
			Actions: actions,
		})
		break
	}
	if len(ratelimits) == 0 {
		return nil, nil, nil
	}
	filters["envoy.filters.http.local_ratelimit"] = MustNewAny(rateLimitConf)
	return ratelimits, filters, nil
}

func MakeSidecarLocalRateLimit(rateLimitCache cache.RateLimitCache,
	svcKey model.ServiceKey) ([]*route.RateLimit, map[string]*anypb.Any, error) {
	conf, _ := rateLimitCache.GetRateLimitRules(svcKey)
	if conf == nil {
		return nil, nil, nil
	}
	confKey := fmt.Sprintf("INBOUND|SIDECAR|%s|%s", svcKey.Namespace, svcKey.Name)
	rateLimitConf := BuildRateLimitConf(confKey)
	filters := make(map[string]*anypb.Any)
	ratelimits := make([]*route.RateLimit, 0, len(conf))
	for _, c := range conf {
		rule := c.Proto
		if rule == nil {
			continue
		}
		// 跳过全局限流配置
		// TODO 暂时不放开全局限流规则下发，后续等待 envoy polaris filter 插件开发或者在 polaris-sidecar 中实现 RLS 协议后
		// 在放开该设置
		if rule.GetType() == apitraffic.Rule_GLOBAL || rule.GetDisable().GetValue() {
			continue
		}
		actions, descriptors := BuildLocalRateLimitDescriptors(rule)
		rateLimitConf.Descriptors = descriptors
		ratelimits = append(ratelimits, &route.RateLimit{
			Actions: actions,
		})
	}
	if len(ratelimits) == 0 {
		return nil, nil, nil
	}
	filters["envoy.filters.http.local_ratelimit"] = MustNewAny(rateLimitConf)
	return ratelimits, filters, nil
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

func MakeLbSubsetConfig(serviceInfo *ServiceInfo) *cluster.Cluster_LbSubsetConfig {
	rules := FilterInboundRouterRule(serviceInfo)
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

func GenEndpointMetaFromPolarisIns(ins *apiservice.Instance) *core.Metadata {
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
		meta.FilterMetadata["envoy.transport_socket_match"] = MTLSTransportSocketMatch
	}
	return meta
}

func IsNormalEndpoint(ins *apiservice.Instance) bool {
	if ins.GetIsolate().GetValue() {
		return false
	}
	if ins.GetWeight().GetValue() == 0 {
		return false
	}
	return true
}

func FormatEndpointHealth(ins *apiservice.Instance) core.HealthStatus {
	if ins.GetHealthy().GetValue() {
		return core.HealthStatus_HEALTHY
	}
	return core.HealthStatus_UNHEALTHY
}
