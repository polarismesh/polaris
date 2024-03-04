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
	"encoding/hex"
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
	ratelimitconfv3 "github.com/envoyproxy/go-control-plane/envoy/config/ratelimit/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	filev3 "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	envoy_extensions_common_ratelimit_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	ratelimitv32 "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	lrl "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/local_ratelimit/v3"
	on_demandv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/on_demand/v3"
	ratelimitfilter "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ratelimit/v3"
	routerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcp "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	v32 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	envoy_type_v3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
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
		case traffic_manage.SourceMatch_HEADER, traffic_manage.SourceMatch_METHOD:
			headerSubName := argument.Key
			if argument.Type == traffic_manage.SourceMatch_METHOD {
				headerSubName = ":method"
			}
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
	destinations []*traffic_manage.DestinationGroup, opt *BuildOption) *route.WeightedCluster {
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
		weightCluster := &route.WeightedCluster_ClusterWeight{
			Name: MakeServiceName(model.ServiceKey{
				Namespace: destination.Namespace,
				Name:      destination.Service,
			}, trafficDirection, opt),
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
			weightCluster.MetadataMatch = nil
		}
		weightedClusters = append(weightedClusters, weightCluster)
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
		// the token bucket must shared across all worker threads
		LocalRateLimitPerDownstreamConnection: false,
	}
	return rateLimitConf
}

func BuildRateLimitDescriptors(rule *traffic_manage.Rule) ([]*route.RateLimit_Action,
	[]*ratelimitv32.LocalRateLimitDescriptor) {
	actions := make([]*route.RateLimit_Action, 0, 8)
	descriptors := make([]*ratelimitv32.LocalRateLimitDescriptor, 0, 8)

	entries := make([]*envoy_extensions_common_ratelimit_v3.RateLimitDescriptor_Entry, 0, len(rule.Labels))

	methodMatchType := rule.GetMethod().GetType()
	methodName := rule.GetMethod().GetValue().GetValue()
	if methodName == "" {
		methodName = "/"
		methodMatchType = MatchString_Prefix
	}
	actions = append(actions, &route.RateLimit_Action{
		ActionSpecifier: &route.RateLimit_Action_HeaderValueMatch_{
			HeaderValueMatch: BuildRateLimitActionHeaderValueMatch(":path", methodName, &apitraffic.MatchArgument{
				Key: ":path",
				Value: &apimodel.MatchString{
					Type:      methodMatchType,
					Value:     wrapperspb.String(methodName),
					ValueType: apimodel.MatchString_TEXT,
				},
			}),
		},
	})
	entries = append(entries, &ratelimitv32.RateLimitDescriptor_Entry{
		Key:   ":path",
		Value: methodName,
	})
	arguments := rule.GetArguments()

	for i := range arguments {
		arg := arguments[i]
		// 仅支持文本类型参数
		if arg.GetValue().GetValueType() != apimodel.MatchString_TEXT {
			continue
		}

		descriptorKey := strings.ToLower(arg.GetType().String()) + "." + arg.Key
		descriptorValue := arg.GetValue().GetValue().GetValue()
		switch arg.Type {
		case apitraffic.MatchArgument_HEADER:
			headerValueMatch := BuildRateLimitActionHeaderValueMatch(descriptorKey, descriptorValue, arg)
			actions = append(actions, &route.RateLimit_Action{
				ActionSpecifier: &route.RateLimit_Action_HeaderValueMatch_{
					HeaderValueMatch: headerValueMatch,
				},
			})
			entries = append(entries, &ratelimitv32.RateLimitDescriptor_Entry{
				Key:   descriptorKey,
				Value: arg.GetValue().GetValue().GetValue(),
			})
		case apitraffic.MatchArgument_QUERY:
			queryParameterValueMatch := BuildRateLimitActionQueryParameterValueMatch(descriptorKey, arg)
			actions = append(actions, &route.RateLimit_Action{
				ActionSpecifier: &route.RateLimit_Action_QueryParameterValueMatch_{
					QueryParameterValueMatch: queryParameterValueMatch,
				},
			})
			entries = append(entries, &ratelimitv32.RateLimitDescriptor_Entry{
				Key:   descriptorKey,
				Value: arg.GetValue().GetValue().GetValue(),
			})
		case apitraffic.MatchArgument_METHOD:
			actions = append(actions, &route.RateLimit_Action{
				ActionSpecifier: &route.RateLimit_Action_RequestHeaders_{
					RequestHeaders: &route.RateLimit_Action_RequestHeaders{
						HeaderName:    ":method",
						DescriptorKey: descriptorKey,
					},
				},
			})
			entries = append(entries, &envoy_extensions_common_ratelimit_v3.RateLimitDescriptor_Entry{
				Key:   descriptorKey,
				Value: arg.GetValue().GetValue().GetValue(),
			})
		case apitraffic.MatchArgument_CALLER_SERVICE:
			descriptorKey := "source_cluster"
			descriptorValue := fmt.Sprintf("%s|%s", arg.GetKey(), arg.GetValue().GetValue().GetValue())

			// 如果是匹配来源服务，则 spec 中的 key 为 namespace，value 为 service

			// 只有匹配规则为全匹配时才能新增 envoy 本身的标签支持
			if arg.GetValue().GetType() == apimodel.MatchString_EXACT {
				// 支持 envoy 原生的标签属性
				actions = append(actions, &route.RateLimit_Action{
					ActionSpecifier: &route.RateLimit_Action_SourceCluster_{
						SourceCluster: &route.RateLimit_Action_SourceCluster{},
					},
				})
			}

			// 从 header 中获取, 支持 Spring Cloud Tencent 查询标签设置
			actions = append(actions, &route.RateLimit_Action{
				ActionSpecifier: &route.RateLimit_Action_HeaderValueMatch_{
					HeaderValueMatch: BuildRateLimitActionHeaderValueMatch(descriptorKey, descriptorValue, []*apitraffic.MatchArgument{
						{
							Key: "source_service_namespace",
							Value: &apimodel.MatchString{
								Type:  arg.GetValue().Type,
								Value: wrapperspb.String(arg.Key),
							},
						},
						{
							Key: "source_service_name",
							Value: &apimodel.MatchString{
								Type:  arg.GetValue().Type,
								Value: wrapperspb.String(arg.Key),
							},
						},
					}...),
				},
			})
			entries = append(entries, &envoy_extensions_common_ratelimit_v3.RateLimitDescriptor_Entry{
				Key:   descriptorKey,
				Value: descriptorValue,
			})
		case apitraffic.MatchArgument_CALLER_IP:
			actions = append(actions, &route.RateLimit_Action{
				ActionSpecifier: &route.RateLimit_Action_RemoteAddress_{
					RemoteAddress: &route.RateLimit_Action_RemoteAddress{},
				},
			})
			entries = append(entries, &envoy_extensions_common_ratelimit_v3.RateLimitDescriptor_Entry{
				Key:   "remote_address",
				Value: arg.GetValue().GetValue().GetValue(),
			})
		}
	}

	for _, amount := range rule.Amounts {
		descriptor := &envoy_extensions_common_ratelimit_v3.LocalRateLimitDescriptor{
			TokenBucket: &envoy_type_v3.TokenBucket{
				MaxTokens:     amount.GetMaxAmount().GetValue(),
				TokensPerFill: wrapperspb.UInt32(amount.GetMaxAmount().GetValue()),
				FillInterval:  amount.GetValidDuration(),
			},
		}
		descriptor.Entries = entries
		descriptors = append(descriptors, descriptor)
	}
	return actions, descriptors
}

func BuildRateLimitActionQueryParameterValueMatch(key string,
	arg *apitraffic.MatchArgument) *route.RateLimit_Action_QueryParameterValueMatch {
	queryParameterValueMatch := &route.RateLimit_Action_QueryParameterValueMatch{
		DescriptorKey:   key,
		DescriptorValue: arg.GetValue().GetValue().GetValue(),
		ExpectMatch:     wrapperspb.Bool(true),
		QueryParameters: []*route.QueryParameterMatcher{},
	}
	switch arg.GetValue().GetType() {
	case apimodel.MatchString_EXACT:
		queryParameterValueMatch.QueryParameters = []*route.QueryParameterMatcher{
			{
				Name: arg.GetKey(),
				QueryParameterMatchSpecifier: &route.QueryParameterMatcher_StringMatch{
					StringMatch: &v32.StringMatcher{
						MatchPattern: &v32.StringMatcher_Exact{
							Exact: arg.GetValue().GetValue().GetValue(),
						},
					},
				},
			},
		}
	case apimodel.MatchString_REGEX:
		queryParameterValueMatch.QueryParameters = []*route.QueryParameterMatcher{
			{
				Name: arg.GetKey(),
				QueryParameterMatchSpecifier: &route.QueryParameterMatcher_StringMatch{
					StringMatch: &v32.StringMatcher{
						MatchPattern: &v32.StringMatcher_SafeRegex{
							SafeRegex: &v32.RegexMatcher{
								EngineType: &v32.RegexMatcher_GoogleRe2{},
								Regex:      arg.GetValue().GetValue().GetValue(),
							},
						},
					},
				},
			},
		}
	}

	return queryParameterValueMatch
}

func BuildRateLimitActionHeaderValueMatch(key, value string,
	arguments ...*apitraffic.MatchArgument) *route.RateLimit_Action_HeaderValueMatch {

	headerValueMatch := &route.RateLimit_Action_HeaderValueMatch{
		DescriptorKey:   key,
		DescriptorValue: value,
		Headers:         []*route.HeaderMatcher{},
	}
	for i := range arguments {
		argument := arguments[i]
		switch argument.GetValue().GetType() {
		case apimodel.MatchString_EXACT, apimodel.MatchString_NOT_EQUALS:
			headerValueMatch.Headers = append(headerValueMatch.Headers, &route.HeaderMatcher{
				Name:        argument.GetKey(),
				InvertMatch: argument.GetValue().GetType() == apimodel.MatchString_NOT_EQUALS,
				HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
					StringMatch: &v32.StringMatcher{
						MatchPattern: &v32.StringMatcher_Exact{
							Exact: argument.GetValue().GetValue().GetValue(),
						},
					},
				},
			})
		case apimodel.MatchString_REGEX:
			headerValueMatch.Headers = append(headerValueMatch.Headers, &route.HeaderMatcher{
				Name: argument.GetKey(),
				HeaderMatchSpecifier: &route.HeaderMatcher_SafeRegexMatch{
					SafeRegexMatch: &v32.RegexMatcher{
						EngineType: &v32.RegexMatcher_GoogleRe2{},
						Regex:      argument.GetValue().GetValue().GetValue(),
					},
				},
			})
		case MatchString_Prefix:
			// 专门用于 prefix
			headerValueMatch.Headers = append(headerValueMatch.Headers, &route.HeaderMatcher{
				Name: argument.GetKey(),
				HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
					StringMatch: &v32.StringMatcher{
						MatchPattern: &v32.StringMatcher_Prefix{
							Prefix: argument.GetValue().GetValue().GetValue(),
						},
					},
				},
			})
		}
	}
	return headerValueMatch
}

func GenerateServiceDomains(serviceInfo *ServiceInfo) []string {
	// k8s dns 可解析的服务名
	domain := serviceInfo.Name + "." + serviceInfo.Namespace
	domains := []string{serviceInfo.Name, domain,
		domain + K8sDnsResolveSuffixSvc,
		domain + K8sDnsResolveSuffixSvcCluster,
		domain + K8sDnsResolveSuffixSvcClusterLocal}

	resDomains := domains
	// 上面各种服务名加服务端口
	ports := serviceInfo.Ports
	for _, port := range ports {
		// 如果是数字，则为每个域名产生一个带端口的域名
		for _, s := range domains {
			resDomains = append(resDomains, s+":"+strconv.FormatUint(uint64(port.Port), 10))
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
	destinations []*traffic_manage.DestinationGroup, opt *BuildOption) *route.Route {
	sidecarRoute := &route.Route{
		Match: routeMatch,
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_WeightedClusters{
					WeightedClusters: BuildWeightClustersV2(trafficDirection, destinations, opt),
				},
			},
		},
	}
	return sidecarRoute
}

// 默认路由
func MakeDefaultRoute(trafficDirection corev3.TrafficDirection, svcKey model.ServiceKey, opt *BuildOption) *route.Route {
	routeConf := &route.Route{
		Match: &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: "/",
			},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_Cluster{
					Cluster: MakeServiceName(svcKey, trafficDirection, opt),
				},
			},
		},
	}
	if opt.IsDemand() {
		routeConf.TypedPerFilterConfig = map[string]*anypb.Any{
			EnvoyHttpFilter_OnDemand: BuildOnDemandRouteTypedPerFilterConfig(),
		}
	}
	return routeConf
}

func MakeSidecarRoute(trafficDirection corev3.TrafficDirection, routeMatch *route.RouteMatch,
	svcInfo *ServiceInfo, destinations []*traffic_manage.DestinationGroup, opt *BuildOption) *route.Route {
	weightClusters := BuildWeightClustersV2(trafficDirection, destinations, opt)
	for i := range weightClusters.Clusters {
		weightClusters.Clusters[i].Name = MakeServiceName(svcInfo.ServiceKey, trafficDirection, opt)
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
	if opt.IsDemand() {
		currentRoute.TypedPerFilterConfig = map[string]*anypb.Any{
			EnvoyHttpFilter_OnDemand: BuildOnDemandRouteTypedPerFilterConfig(),
		}
	}
	return currentRoute
}

func BuildOnDemandRouteTypedPerFilterConfig() *anypb.Any {
	return MustNewAny(&on_demandv3.PerRouteConfig{
		Odcds: &on_demandv3.OnDemandCds{
			Source: &corev3.ConfigSource{
				ConfigSourceSpecifier: &corev3.ConfigSource_ApiConfigSource{
					ApiConfigSource: &corev3.ApiConfigSource{
						ApiType:             corev3.ApiConfigSource_DELTA_GRPC,
						TransportApiVersion: corev3.ApiVersion_V3,
						GrpcServices: []*corev3.GrpcService{
							{
								TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
									EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
										ClusterName: "polaris_xds_server",
									},
								},
							},
						},
					},
				},
			},
		},
	})
}

var PassthroughCluster = &cluster.Cluster{
	Name:                 PassthroughClusterName,
	ConnectTimeout:       durationpb.New(5 * time.Second),
	ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_ORIGINAL_DST},
	LbPolicy:             cluster.Cluster_CLUSTER_PROVIDED,
	CircuitBreakers: &cluster.CircuitBreakers{
		Thresholds: []*cluster.CircuitBreakers_Thresholds{
			{
				MaxConnections:     &wrappers.UInt32Value{Value: math.MaxUint32},
				MaxPendingRequests: &wrappers.UInt32Value{Value: math.MaxUint32},
				MaxRequests:        &wrappers.UInt32Value{Value: math.MaxUint32},
				MaxRetries:         &wrappers.UInt32Value{Value: math.MaxUint32},
			},
		},
	},
}

// MakeInBoundRouteConfigName .
func MakeInBoundRouteConfigName(svcKey model.ServiceKey, demand bool) string {
	if demand {
		return InBoundRouteConfigName + "|" + svcKey.Domain() + "|DEMAND"
	}
	return InBoundRouteConfigName + "|" + svcKey.Domain()
}

// MakeServiceName .
func MakeServiceName(svcKey model.ServiceKey, trafficDirection corev3.TrafficDirection,
	opt *BuildOption) string {
	if trafficDirection == core.TrafficDirection_INBOUND || !opt.IsDemand() {
		return fmt.Sprintf("%s|%s|%s", corev3.TrafficDirection_name[int32(trafficDirection)],
			svcKey.Namespace, svcKey.Name)
	}
	// return svcKey.Name + "." + svcKey.Namespace
	return fmt.Sprintf("%s|%s|%s", corev3.TrafficDirection_name[int32(trafficDirection)],
		svcKey.Namespace, svcKey.Name)
}

// MakeVHDSServiceName .
func MakeVHDSServiceName(prefix string, svcKey model.ServiceKey) string {
	return prefix + svcKey.Name + "." + svcKey.Namespace
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

func makeRateLimitHCMFilter(svcKey model.ServiceKey) []*hcm.HttpFilter {
	return []*hcm.HttpFilter{
		{
			Name: "envoy.filters.http.local_ratelimit",
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: MustNewAny(&lrl.LocalRateLimit{
					StatPrefix: "http_local_rate_limiter",
					Stage:      LocalRateLimitStage,
				}),
			},
		},
		{
			Name: "envoy.filters.http.ratelimit",
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: MustNewAny(&ratelimitfilter.RateLimit{
					Domain:      fmt.Sprintf("%s.%s", svcKey.Name, svcKey.Namespace),
					Stage:       DistributedRateLimitStage,
					RequestType: "external",
					Timeout:     durationpb.New(2 * time.Second),
					RateLimitService: &ratelimitconfv3.RateLimitServiceConfig{
						GrpcService: &corev3.GrpcService{
							TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
								EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
									ClusterName: "polaris_ratelimit",
								},
							},
							Timeout: durationpb.New(time.Second),
						},
						TransportApiVersion: core.ApiVersion_V3,
					},
				}),
			},
		},
	}
}

func makeSidecarOnDemandHCMFilter(option *BuildOption) []*hcm.HttpFilter {
	return []*hcm.HttpFilter{
		{
			Name: EnvoyHttpFilter_OnDemand,
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: MustNewAny(&on_demandv3.OnDemand{}),
			},
		},
		{
			Name: wellknown.Router,
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: MustNewAny(&routerv3.Router{}),
			},
		},
	}
}

func MakeSidecarOnDemandOutBoundHCM(svcKey model.ServiceKey, option *BuildOption) *hcm.HttpConnectionManager {

	hcmFilters := makeSidecarOnDemandHCMFilter(option)

	manager := &hcm.HttpConnectionManager{
		CodecType:           hcm.HttpConnectionManager_AUTO,
		StatPrefix:          corev3.TrafficDirection_name[int32(corev3.TrafficDirection_OUTBOUND)] + "_HTTP",
		RouteSpecifier:      routeSpecifier(core.TrafficDirection_OUTBOUND, option),
		AccessLog:           accessLog(),
		HttpFilters:         hcmFilters,
		HttpProtocolOptions: &core.Http1ProtocolOptions{AcceptHttp_10: true},
	}
	return manager
}

func MakeSidecarBoundHCM(svcKey model.ServiceKey, trafficDirection corev3.TrafficDirection, opt *BuildOption) *hcm.HttpConnectionManager {
	hcmFilters := []*hcm.HttpFilter{
		{
			Name: wellknown.Router,
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: MustNewAny(&routerv3.Router{}),
			},
		},
	}
	if trafficDirection == corev3.TrafficDirection_INBOUND {
		hcmFilters = append(makeRateLimitHCMFilter(svcKey), hcmFilters...)
	}
	if opt.IsDemand() {
		hcmFilters = append([]*hcm.HttpFilter{
			{
				Name: EnvoyHttpFilter_OnDemand,
				ConfigType: &hcm.HttpFilter_TypedConfig{
					TypedConfig: MustNewAny(&on_demandv3.OnDemand{}),
				},
			},
		}, hcmFilters...)
	}

	manager := &hcm.HttpConnectionManager{
		CodecType:            hcm.HttpConnectionManager_AUTO,
		StatPrefix:           corev3.TrafficDirection_name[int32(trafficDirection)] + "_HTTP",
		RouteSpecifier:       routeSpecifier(trafficDirection, opt),
		AccessLog:            accessLog(),
		HttpFilters:          hcmFilters,
		HttpProtocolOptions:  &core.Http1ProtocolOptions{AcceptHttp_10: true},
		Http2ProtocolOptions: &corev3.Http2ProtocolOptions{},
		Http3ProtocolOptions: &corev3.Http3ProtocolOptions{},
	}

	// 重写 RouteSpecifier 的路由规则数据信息
	if trafficDirection == core.TrafficDirection_INBOUND {
		manager.GetRds().RouteConfigName = MakeInBoundRouteConfigName(svcKey, opt.IsDemand())
	}

	return manager
}

func MakeGatewayBoundHCM(svcKey model.ServiceKey, opt *BuildOption) *hcm.HttpConnectionManager {
	hcmFilters := makeRateLimitHCMFilter(svcKey)
	hcmFilters = append(hcmFilters, &hcm.HttpFilter{
		Name: wellknown.Router,
		ConfigType: &hcm.HttpFilter_TypedConfig{
			TypedConfig: MustNewAny(&routerv3.Router{}),
		},
	})
	trafficDirectionName := corev3.TrafficDirection_name[int32(corev3.TrafficDirection_INBOUND)]
	manager := &hcm.HttpConnectionManager{
		CodecType:           hcm.HttpConnectionManager_AUTO,
		StatPrefix:          trafficDirectionName + "_HTTP",
		RouteSpecifier:      routeSpecifier(corev3.TrafficDirection_OUTBOUND, opt),
		AccessLog:           accessLog(),
		HttpFilters:         hcmFilters,
		HttpProtocolOptions: &core.Http1ProtocolOptions{AcceptHttp_10: true},
	}
	return manager
}

func routeSpecifier(trafficDirection corev3.TrafficDirection, opt *BuildOption) *hcm.HttpConnectionManager_Rds {
	baseRouteName := TrafficBoundRoute[trafficDirection]
	if opt.IsDemand() {
		baseRouteName = fmt.Sprintf("%s|%s|demand", TrafficBoundRoute[trafficDirection],
			opt.Namespace)
	}
	if opt.RunType == RunTypeGateway {
		baseRouteName += "-gateway"
	}
	return &hcm.HttpConnectionManager_Rds{
		Rds: &hcm.Rds{
			ConfigSource: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
			RouteConfigName: baseRouteName,
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

func MakeGatewayLocalRateLimit(rateLimitCache types.RateLimitCache, pathSpecifier string,
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
		if rule.GetDisable().GetValue() {
			continue
		}
		if rule.GetMethod().GetValue().GetValue() != pathSpecifier {
			continue
		}
		actions, descriptors := BuildRateLimitDescriptors(rule)
		rateLimitConf.Descriptors = descriptors
		ratelimitRule := &route.RateLimit{Actions: actions}
		switch rule.GetType() {
		case apitraffic.Rule_LOCAL:
			ratelimitRule.Stage = wrapperspb.UInt32(LocalRateLimitStage)
		case apitraffic.Rule_GLOBAL:
			ratelimitRule.Stage = wrapperspb.UInt32(DistributedRateLimitStage)
		}
		ratelimits = append(ratelimits, ratelimitRule)
	}
	if len(ratelimits) == 0 {
		return nil, nil, nil
	}
	filters["envoy.filters.http.local_ratelimit"] = MustNewAny(rateLimitConf)
	return ratelimits, filters, nil
}

func MakeSidecarLocalRateLimit(rateLimitCache types.RateLimitCache,
	svcKey model.ServiceKey) ([]*route.RateLimit, map[string]*anypb.Any, error) {
	conf, _ := rateLimitCache.GetRateLimitRules(svcKey)
	if conf == nil {
		return nil, map[string]*anypb.Any{}, nil
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
		if rule.GetDisable().GetValue() {
			continue
		}
		actions, descriptors := BuildRateLimitDescriptors(rule)
		rateLimitConf.Descriptors = descriptors
		ratelimitRule := &route.RateLimit{Actions: actions}
		switch rule.GetType() {
		case apitraffic.Rule_LOCAL:
			ratelimitRule.Stage = wrapperspb.UInt32(LocalRateLimitStage)
		case apitraffic.Rule_GLOBAL:
			ratelimitRule.Stage = wrapperspb.UInt32(DistributedRateLimitStage)
		}
		ratelimits = append(ratelimits, ratelimitRule)
	}
	filters["envoy.filters.http.local_ratelimit"] = MustNewAny(rateLimitConf)
	return ratelimits, filters, nil
}

// Translate the circuit breaker configuration of Polaris into OutlierDetection
func MakeOutlierDetection(serviceInfo *ServiceInfo) *cluster.OutlierDetection {
	circuitBreaker := serviceInfo.CircuitBreaker
	if circuitBreaker == nil || len(circuitBreaker.Rules) == 0 {
		return nil
	}
	var rule *apifault.CircuitBreakerRule
	for _, item := range circuitBreaker.Rules {
		if item.Level == apifault.Level_INSTANCE {
			rule = item
			break
		}
	}
	// not config or close circuit breaker
	if rule == nil || len(rule.TriggerCondition) == 0 || !rule.Enable {
		return nil
	}
	triggerCondtion := rule.TriggerCondition[0]
	outlierDetection := &cluster.OutlierDetection{}
	outlierDetection.Interval = durationpb.New(time.Duration(triggerCondtion.GetInterval()) * time.Second)
	outlierDetection.Consecutive_5Xx = &wrappers.UInt32Value{
		Value: triggerCondtion.GetErrorCount()}
	outlierDetection.FailurePercentageThreshold = &wrappers.UInt32Value{
		Value: triggerCondtion.GetErrorPercent()}
	outlierDetection.FailurePercentageRequestVolume = &wrappers.UInt32Value{
		Value: triggerCondtion.GetMinimumRequest()}
	if rule.RecoverCondition != nil {
		outlierDetection.BaseEjectionTime =
			durationpb.New(time.Duration(rule.GetRecoverCondition().GetSleepWindow()) * time.Second)
	}

	return outlierDetection
}

// Translate the FaultDetector configuration of Polaris into HealthCheck
func MakeHealthCheck(serviceInfo *ServiceInfo) []*core.HealthCheck {
	if serviceInfo.FaultDetect == nil || len(serviceInfo.FaultDetect.Rules) == 0 {
		return nil
	}
	var healthChecks []*core.HealthCheck
	for _, rule := range serviceInfo.FaultDetect.Rules {
		healthCheck := &core.HealthCheck{
			Timeout:            durationpb.New(time.Duration(rule.GetTimeout()) * time.Second),
			Interval:           durationpb.New(time.Duration(rule.GetInterval()) * time.Second),
			UnhealthyThreshold: &wrappers.UInt32Value{Value: 3},
			HealthyThreshold:   &wrappers.UInt32Value{Value: 1},
		}
		if rule.GetProtocol() == apifault.FaultDetectRule_HTTP {
			config := rule.GetHttpConfig()
			if config == nil {
				continue
			}
			var headers []*core.HeaderValueOption
			for _, item := range config.GetHeaders() {
				header := core.HeaderValueOption{
					Header: &core.HeaderValue{
						Key:   item.Key,
						Value: item.Value,
					},
				}
				headers = append(headers, &header)
			}

			httpHealthCheck := &core.HealthCheck_HttpHealthCheck{
				Path:                config.Url,
				Method:              core.RequestMethod(core.RequestMethod_value[config.Method]),
				RequestHeadersToAdd: headers,
			}
			healthCheck.HealthChecker = &core.HealthCheck_HttpHealthCheck_{HttpHealthCheck: httpHealthCheck}
			healthChecks = append(healthChecks, healthCheck)
		} else if rule.GetProtocol() == apifault.FaultDetectRule_TCP {
			config := rule.GetTcpConfig()
			if config == nil {
				continue
			}
			var receives []*core.HealthCheck_Payload
			for _, item := range config.GetReceive() {
				receives = append(receives, &core.HealthCheck_Payload{
					Payload: &core.HealthCheck_Payload_Text{Text: hex.EncodeToString([]byte(item))},
				})
			}
			tcpHealthCheck := &core.HealthCheck_TcpHealthCheck{
				Send: &core.HealthCheck_Payload{
					Payload: &core.HealthCheck_Payload_Text{Text: hex.EncodeToString([]byte(config.Send))},
				},
				Receive: receives,
			}
			healthCheck.HealthChecker = &core.HealthCheck_TcpHealthCheck_{TcpHealthCheck: tcpHealthCheck}
			healthChecks = append(healthChecks, healthCheck)
		}
	}
	return healthChecks
}

func MakeLbSubsetConfig(serviceInfo *ServiceInfo) *cluster.Cluster_LbSubsetConfig {
	rules := FilterInboundRouterRule(serviceInfo)
	if len(rules) == 0 {
		return nil
	}

	var subsetSelectors []*cluster.Cluster_LbSubsetConfig_LbSubsetSelector
	for _, rule := range rules {
		// 对每一个 destination 产生一个 subset
		for _, destination := range rule.GetDestinations() {
			var keys []string
			for s := range destination.GetLabels() {
				keys = append(keys, s)
			}
			subsetSelectors = append(subsetSelectors, &cluster.Cluster_LbSubsetConfig_LbSubsetSelector{
				Keys:           keys,
				FallbackPolicy: cluster.Cluster_LbSubsetConfig_LbSubsetSelector_ANY_ENDPOINT,
			})
		}
	}

	return &cluster.Cluster_LbSubsetConfig{
		SubsetSelectors: subsetSelectors,
		FallbackPolicy:  cluster.Cluster_LbSubsetConfig_ANY_ENDPOINT,
	}
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
	if ins.Metadata != nil && EnableTLS(TLSMode(ins.Metadata[TLSModeTag])) {
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

func SupportTLS(x XDSType) bool {
	switch x {
	case CDS, LDS:
		return true
	default:
		return false
	}
}
