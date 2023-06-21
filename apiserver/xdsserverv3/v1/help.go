package v1

import (
	"math"
	"time"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	lrl "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/local_ratelimit/v3"
	v32 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	envoy_type_v3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/polarismesh/polaris/common/utils"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func buildCommonRouteMatch(routeMatch *route.RouteMatch, source *traffic_manage.SourceService) {
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

func buildRateLimitConf(prefix string) *lrl.LocalRateLimit {
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
