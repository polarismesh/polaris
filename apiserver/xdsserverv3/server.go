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
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_common_ratelimit_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	lrl "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/local_ratelimit/v3"
	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimeservice "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretservice "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	v32 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	envoy_type_v3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/golang/protobuf/ptypes"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/polarismesh/polaris/apiserver"
	"github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/connlimit"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/service"
)

const (
	K8sDnsResolveSuffixSvc             = ".svc"
	K8sDnsResolveSuffixSvcCluster      = ".svc.cluster"
	K8sDnsResolveSuffixSvcClusterLocal = ".svc.cluster.local"
)

const (
	TLSModeTag        = "polarismesh.cn/tls-mode"
	TLSModeNone       = "none"
	TLSModeStrict     = "strict"
	TLSModePermissive = "permissive"
)

// XDSServer is the xDS server
type XDSServer struct {
	listenIP        string
	listenPort      uint32
	start           bool
	restart         bool
	exitCh          chan struct{}
	namingServer    service.DiscoverServer
	cache           cachev3.SnapshotCache
	versionNum      *atomic.Uint64
	server          *grpc.Server
	connLimitConfig *connlimit.Config

	registryInfo               map[string][]*ServiceInfo
	CircuitBreakerConfigGetter CircuitBreakerConfigGetter
	RatelimitConfigGetter      RatelimitConfigGetter
}

// Initialize 初始化
func (x *XDSServer) Initialize(ctx context.Context, option map[string]interface{},
	api map[string]apiserver.APIConfig,
) error {
	x.cache = cachev3.NewSnapshotCache(false, PolarisNodeHash{}, commonlog.GetScopeOrDefaultByName(commonlog.XDSLoggerName))
	x.registryInfo = make(map[string][]*ServiceInfo)
	x.listenPort = uint32(option["listenPort"].(int))
	x.listenIP = option["listenIP"].(string)

	x.versionNum = atomic.NewUint64(0)
	var err error

	x.namingServer, err = service.GetServer()
	if err != nil {
		log.Errorf("%v", err)
		return err
	}

	if raw, _ := option["connLimit"].(map[interface{}]interface{}); raw != nil {
		connConfig, err := connlimit.ParseConnLimitConfig(raw)
		if err != nil {
			return err
		}
		x.connLimitConfig = connConfig
	}

	err = x.initRegistryInfo()
	if err != nil {
		log.Errorf("%v", err)
		return err
	}

	err = x.getRegistryInfoWithCache(ctx, x.registryInfo)
	if err != nil {
		log.Errorf("%v", err)
		return err
	}

	err = x.pushRegistryInfoToXDSCache(x.registryInfo)
	if err != nil {
		log.Errorf("%v", err)
		return err
	}

	_ = x.startSynTask(ctx)

	return nil
}

// Run 启动运行
func (x *XDSServer) Run(errCh chan error) {
	// 启动 grpc server
	ctx := context.Background()
	cb := &Callbacks{log: commonlog.GetScopeOrDefaultByName(commonlog.XDSLoggerName)}
	srv := serverv3.NewServer(ctx, x.cache, cb)
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(1000))
	grpcServer := grpc.NewServer(grpcOptions...)
	x.server = grpcServer
	address := fmt.Sprintf("%v:%v", x.listenIP, x.listenPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}

	if x.connLimitConfig != nil && x.connLimitConfig.OpenConnLimit {
		log.Infof("grpc server use max connection limit: %d, grpc max limit: %d",
			x.connLimitConfig.MaxConnPerHost, x.connLimitConfig.MaxConnLimit)
		listener, err = connlimit.NewListener(listener, x.GetProtocol(), x.connLimitConfig)
		if err != nil {
			log.Errorf("conn limit init err: %s", err.Error())
			errCh <- err
			return
		}

	}

	registerServer(grpcServer, srv)

	log.Infof("management server listening on %d\n", x.listenPort)

	if err = grpcServer.Serve(listener); err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}

	log.Info("xds server stop")
}

func registerServer(grpcServer *grpc.Server, server serverv3.Server) {
	// register services
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	secretservice.RegisterSecretDiscoveryServiceServer(grpcServer, server)
	runtimeservice.RegisterRuntimeDiscoveryServiceServer(grpcServer, server)
}

// Stop 停止服务
func (x *XDSServer) Stop() {
	connlimit.RemoveLimitListener(x.GetProtocol())
	if x.server != nil {
		x.server.Stop()
	}
}

// Restart 重启服务
func (x *XDSServer) Restart(option map[string]interface{}, api map[string]apiserver.APIConfig, errCh chan error) error {
	log.Infof("restart xds server with new config: +%v", option)

	x.restart = true
	x.Stop()
	if x.start {
		<-x.exitCh
	}

	log.Info("old xds server has stopped, begin restarting it")
	if err := x.Initialize(context.Background(), option, api); err != nil {
		log.Errorf("restart grpc server err: %s", err.Error())
		return err
	}

	log.Info("init grpc server successfully, restart it")
	x.restart = false
	go x.Run(errCh)

	return nil
}

type RatelimitConfigGetter func(serviceID string) []*model.RateLimit

// PolarisNodeHash 存放 hash 方法
type PolarisNodeHash struct{}

// node id 的格式是:
// 1. namespace/uuid~hostIp
var nodeIDFormat = regexp.MustCompile(`^(\S+)\/([^~]+)~([^~]+)$`)

func parseNodeID(nodeID string) (namespace string, uuid string, hostip string) {
	groups := nodeIDFormat.FindStringSubmatch(nodeID)
	if len(groups) == 0 {
		// invalid node format
		return
	}
	namespace = groups[1]
	uuid = groups[2]
	hostip = groups[3]
	return
}

// ID id 的格式是 namespace/uuid~hostIp
func (PolarisNodeHash) ID(node *core.Node) string {
	if node == nil {
		return ""
	}
	ns, _, _ := parseNodeID(node.Id)
	if node.Metadata != nil && node.Metadata.Fields != nil {
		tlsMode := node.Metadata.Fields[TLSModeTag].GetStringValue()
		if tlsMode == TLSModePermissive || tlsMode == TLSModeStrict {
			return ns + "/" + tlsMode
		}
	}

	return ns
}

// GetProtocol 服务注册到北极星中的协议
func (x *XDSServer) GetProtocol() string {
	return "xdsv3"
}

// GetPort 服务注册到北极星中的端口
func (x *XDSServer) GetPort() uint32 {
	return x.listenPort
}

// ServiceInfo 北极星服务结构体
type ServiceInfo struct {
	ID                   string
	Name                 string
	Namespace            string
	Instances            []*api.Instance
	SvcInsRevision       string
	Routing              *api.Routing
	SvcRoutingRevision   string
	Ports                string
	RateLimit            *api.RateLimit
	SvcRateLimitRevision string
}

func makeLbSubsetConfig(serviceInfo *ServiceInfo) *cluster.Cluster_LbSubsetConfig {
	if serviceInfo.Routing != nil && serviceInfo.Routing.Inbounds != nil &&
		len(serviceInfo.Routing.Inbounds) > 0 {
		lbSubsetConfig := &cluster.Cluster_LbSubsetConfig{}
		var subsetSelectors []*cluster.Cluster_LbSubsetConfig_LbSubsetSelector
		lbSubsetConfig.FallbackPolicy = cluster.Cluster_LbSubsetConfig_ANY_ENDPOINT

		for _, inbound := range serviceInfo.Routing.Inbounds {
			// 对每一个 destination 产生一个 subset
			for _, destination := range inbound.Destinations {
				var keys []string
				for s := range destination.Metadata {
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
	return nil
}

// Translate the circuit breaker configuration of Polaris into OutlierDetection
func makeOutlierDetection(conf *model.ServiceWithCircuitBreaker) *cluster.OutlierDetection {
	if conf != nil {
		cbRules := conf.CircuitBreaker.Inbounds
		if cbRules == "" {
			return nil
		}

		var inBounds []*api.CbRule
		if err := json.Unmarshal([]byte(cbRules), &inBounds); err != nil {
			log.Errorf("unmarshal inbounds circuitBreaker rule error, %v", err)
			return nil
		}

		if len(inBounds) == 0 || len(inBounds[0].GetDestinations()) == 0 ||
			inBounds[0].GetDestinations()[0].Policy == nil {
			return nil
		}

		var (
			consecutiveErrConfig *api.CbPolicy_ConsecutiveErrConfig
			errorRateConfig      *api.CbPolicy_ErrRateConfig
			policy               *api.CbPolicy
			dest                 *api.DestinationSet
		)

		dest = inBounds[0].GetDestinations()[0]
		policy = dest.Policy
		consecutiveErrConfig = policy.Consecutive
		errorRateConfig = policy.ErrorRate

		outlierDetection := &cluster.OutlierDetection{}

		if consecutiveErrConfig != nil {
			outlierDetection.Consecutive_5Xx = &wrappers.UInt32Value{Value: consecutiveErrConfig.ConsecutiveErrorToOpen.Value}
		}
		if errorRateConfig != nil {
			outlierDetection.FailurePercentageRequestVolume = &wrappers.UInt32Value{Value: errorRateConfig.RequestVolumeThreshold.Value}
			outlierDetection.FailurePercentageThreshold = &wrappers.UInt32Value{Value: errorRateConfig.ErrorRateToOpen.Value}
		}

		return outlierDetection
	}
	return nil
}

func getEndpointMetaFromPolarisIns(ins *api.Instance) *core.Metadata {
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

func makeEndpoints(services []*ServiceInfo) []types.Resource {
	var clusterLoads []types.Resource

	for _, service := range services {

		var lbEndpoints []*endpoint.LbEndpoint
		for _, instance := range service.Instances {
			// 只加入健康的实例
			if instance.Healthy.Value {
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
					Metadata: getEndpointMetaFromPolarisIns(instance),
				}

				lbEndpoints = append(lbEndpoints, ep)
			}
		}

		cla := &endpoint.ClusterLoadAssignment{
			ClusterName: service.Name,
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

func makeRoutes(serviceInfo *ServiceInfo) []*route.Route {
	var routes []*route.Route
	var matchAllRoute *route.Route
	// 路由目前只处理 inbounds
	if serviceInfo.Routing != nil && len(serviceInfo.Routing.Inbounds) > 0 {
		for _, inbound := range serviceInfo.Routing.Inbounds {

			routeMatch := &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{Prefix: "/"},
			}
			var matchAll bool
			// 使用 sources 生成 routeMatch
			for _, source := range inbound.Sources {
				if source.Metadata == nil || len(source.Metadata) == 0 {
					matchAll = true
					break
				}
				for name := range source.Metadata {
					if name == "*" {
						matchAll = true
						break
					}
				}
				if matchAll {
					break
				} else {
					for name, matchString := range source.Metadata {
						if name == model.LabelKeyPath {
							if matchString.Type == api.MatchString_EXACT {
								routeMatch.PathSpecifier = &route.RouteMatch_Path{
									Path: matchString.GetValue().GetValue()}
							} else if matchString.Type == api.MatchString_REGEX {
								routeMatch.PathSpecifier = &route.RouteMatch_SafeRegex{SafeRegex: &v32.RegexMatcher{
									Regex: matchString.GetValue().GetValue()}}
							}
						} else if strings.HasPrefix(name, model.LabelKeyHeader) {
							headerSubName := name[len(model.LabelKeyHeader):]
							if !(len(headerSubName) > 1 && strings.HasPrefix(headerSubName, ".")) {
								continue
							}
							headerSubName = headerSubName[1:]
							var headerMatch *route.HeaderMatcher
							if matchString.Type == api.MatchString_EXACT {
								headerMatch = &route.HeaderMatcher{
									Name: headerSubName,
									HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
										StringMatch: &v32.StringMatcher{MatchPattern: &v32.StringMatcher_Exact{Exact: matchString.GetValue().GetValue()}},
									},
								}
							}
							if matchString.Type == api.MatchString_NOT_EQUALS {
								headerMatch = &route.HeaderMatcher{
									Name: headerSubName,
									HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
										StringMatch: &v32.StringMatcher{MatchPattern: &v32.StringMatcher_Exact{Exact: matchString.GetValue().GetValue()}},
									},
									InvertMatch: true,
								}
							}
							if matchString.Type == api.MatchString_REGEX {
								headerMatch = &route.HeaderMatcher{
									Name: headerSubName,
									HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
										StringMatch: &v32.StringMatcher{MatchPattern: &v32.StringMatcher_SafeRegex{
											SafeRegex: &v32.RegexMatcher{
												EngineType: &v32.RegexMatcher_GoogleRe2{GoogleRe2: &v32.RegexMatcher_GoogleRE2{}},
												Regex:      matchString.GetValue().GetValue()}}},
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
							if matchString.Type == api.MatchString_EXACT {
								queryMatcher = &route.QueryParameterMatcher{
									Name: querySubName,
									QueryParameterMatchSpecifier: &route.QueryParameterMatcher_StringMatch{
										StringMatch: &v32.StringMatcher{MatchPattern: &v32.StringMatcher_Exact{Exact: matchString.GetValue().GetValue()}},
									},
								}
							}
							if matchString.Type == api.MatchString_REGEX {
								queryMatcher = &route.QueryParameterMatcher{
									Name: querySubName,
									QueryParameterMatchSpecifier: &route.QueryParameterMatcher_StringMatch{
										StringMatch: &v32.StringMatcher{MatchPattern: &v32.StringMatcher_SafeRegex{SafeRegex: &v32.RegexMatcher{
											EngineType: &v32.RegexMatcher_GoogleRe2{GoogleRe2: &v32.RegexMatcher_GoogleRE2{}},
											Regex:      matchString.GetValue().GetValue(),
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
			}

			var weightedClusters []*route.WeightedCluster_ClusterWeight
			var totalWeight uint32

			// 使用 destinations 生成 weightedClusters。makeClusters() 也使用这个字段生成对应的 subset
			for _, destination := range inbound.Destinations {

				fields := make(map[string]*_struct.Value)
				for k, v := range destination.Metadata {
					fields[k] = &_struct.Value{
						Kind: &_struct.Value_StringValue{
							StringValue: v.Value.Value,
						},
					}
				}

				weightedClusters = append(weightedClusters, &route.WeightedCluster_ClusterWeight{
					Name:   serviceInfo.Name,
					Weight: destination.Weight,
					MetadataMatch: &core.Metadata{
						FilterMetadata: map[string]*_struct.Struct{
							"envoy.lb": {
								Fields: fields,
							},
						},
					},
				})

				totalWeight += destination.Weight.Value
			}

			currentRoute := &route.Route{
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
			if matchAll {
				matchAllRoute = currentRoute
			} else {
				routes = append(routes, currentRoute)
			}
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
	ports := strings.Split(serviceInfo.Ports, ",")
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

func makeLocalRateLimit(conf []*model.RateLimit) map[string]*anypb.Any {
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
			rule := new(api.Rule)
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
			if rule.Type == api.Rule_GLOBAL || rule.Disable.Value {
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
			if rule.AmountMode == api.Rule_GLOBAL_TOTAL {
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

func (x *XDSServer) makeVirtualHosts(services []*ServiceInfo) []types.Resource {
	// 每个 polaris service 对应一个 virtualHost
	var routeConfs []types.Resource
	var hosts []*route.VirtualHost

	for _, service := range services {
		ratelimitGetter := x.RatelimitConfigGetter
		if ratelimitGetter == nil {
			ratelimitGetter = x.namingServer.Cache().RateLimit().GetRateLimitByServiceID
		}
		rateLimitConf := ratelimitGetter(service.ID)
		hosts = append(hosts, &route.VirtualHost{
			Name:                 service.Name,
			Domains:              generateServiceDomains(service),
			Routes:               makeRoutes(service),
			TypedPerFilterConfig: makeLocalRateLimit(rateLimitConf),
		})
	}

	// 最后是 allow_any
	hosts = append(hosts, &route.VirtualHost{
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
	})

	routeConfiguration := &route.RouteConfiguration{
		Name: "polaris-router",
		ValidateClusters: &wrappers.BoolValue{
			Value: false,
		},
		VirtualHosts: hosts,
	}

	return append(routeConfs, routeConfiguration)
}

func (x *XDSServer) pushRegistryInfoToXDSCache(registryInfo map[string][]*ServiceInfo) error {
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
	resources[resource.RouteType] = x.makeVirtualHosts(services)
	resources[resource.ListenerType] = makeListeners()
	snapshot, err := cachev3.NewSnapshot(version, resources)
	if err != nil {
		log.Errorf("fail to create snapshot for %s, err is %v", ns, err)
		return err
	}
	err = snapshot.Consistent()
	if err != nil {
		return err
	}
	log.Infof("will serve ns: %s ,snapshot: %+v", ns, string(dumpSnapShotJSON(snapshot)))
	// 为每个 ns 刷写 cache ，推送 xds 更新
	if err := x.cache.SetSnapshot(context.Background(), ns, snapshot); err != nil {
		log.Errorf("snapshot error %q for %+v", err, snapshot)
		return err
	}
	return
}

func (x *XDSServer) makePermissiveSnapshot(ns, version string, services []*ServiceInfo) (err error) {
	resources := make(map[resource.Type][]types.Resource)
	resources[resource.EndpointType] = makeEndpoints(services)
	resources[resource.ClusterType] = x.makePermissiveClusters(services)
	resources[resource.RouteType] = x.makeVirtualHosts(services)
	resources[resource.ListenerType] = makePermissiveListeners()
	snapshot, err := cachev3.NewSnapshot(version, resources)
	if err != nil {
		return err
	}
	err = snapshot.Consistent()
	if err != nil {
		return err
	}
	log.Infof("will serve ns: %s ,mode permissive,snapshot: %+v", ns, string(dumpSnapShotJSON(snapshot)))
	// 为每个 ns 刷写 cache ，推送 xds 更新
	if err := x.cache.SetSnapshot(context.Background(), ns+"/permissive", snapshot); err != nil {
		log.Errorf("snapshot error %q for %+v", err, snapshot)
		return err
	}
	return
}

func (x *XDSServer) makeStrictSnapshot(ns, version string, services []*ServiceInfo) (err error) {
	resources := make(map[resource.Type][]types.Resource)
	resources[resource.EndpointType] = makeEndpoints(services)
	resources[resource.ClusterType] = x.makeStrictClusters(services)
	resources[resource.RouteType] = x.makeVirtualHosts(services)
	resources[resource.ListenerType] = makeStrictListeners()
	snapshot, err := cachev3.NewSnapshot(version, resources)
	if err != nil {
		return err
	}
	err = snapshot.Consistent()
	if err != nil {
		return err
	}
	log.Infof("will serve ns: %s ,mode strict,snapshot: %+v", ns, string(dumpSnapShotJSON(snapshot)))
	// 为每个 ns 刷写 cache ，推送 xds 更新
	if err := x.cache.SetSnapshot(context.Background(), ns+"/strict", snapshot); err != nil {
		log.Errorf("snapshot error %q for %+v", err, snapshot)
		return err
	}
	return
}

// syncPolarisServiceInfo 初始化本地 cache，初始化 xds cache
func (x *XDSServer) getRegistryInfoWithCache(ctx context.Context, registryInfo map[string][]*ServiceInfo) error {
	// 从 cache 中获取全量的服务信息
	serviceIterProc := func(key string, value *model.Service) (bool, error) {
		if _, ok := registryInfo[value.Namespace]; !ok {
			registryInfo[value.Namespace] = []*ServiceInfo{}
		}

		info := &ServiceInfo{
			ID:        value.ID,
			Name:      value.Name,
			Namespace: value.Namespace,
			Instances: []*api.Instance{},
			Ports:     value.Ports,
		}

		if info.Ports == "" {
			ports := x.namingServer.Cache().Instance().GetServicePorts(value.ID)
			if len(ports) != 0 {
				info.Ports = strings.Join(ports, ",")
			}
		}

		registryInfo[value.Namespace] = append(registryInfo[value.Namespace], info)

		return true, nil
	}

	if err := x.namingServer.Cache().Service().IteratorServices(serviceIterProc); err != nil {
		log.Errorf("syn polaris services error %v", err)
		return err
	}

	// 遍历每一个服务，获取路由、熔断策略和全量的服务实例信息
	for _, v := range registryInfo {
		for _, svc := range v {

			s := &api.Service{
				Name: &wrappers.StringValue{
					Value: svc.Name,
				},
				Namespace: &wrappers.StringValue{
					Value: svc.Namespace,
				},
				Revision: &wrappers.StringValue{
					Value: "-1",
				},
			}

			// 获取routing配置
			routeResp := x.namingServer.GetRoutingConfigWithCache(ctx, s)
			if routeResp.GetCode().Value != api.ExecuteSuccess {
				log.Errorf("error sync routing for %s, info : %s", svc.Name, routeResp.Info.GetValue())
				return fmt.Errorf("[XDSV3] error sync routing for %s", svc.Name)
			}

			if routeResp.Routing != nil {
				svc.SvcRoutingRevision = routeResp.Routing.Revision.Value
				svc.Routing = routeResp.Routing
			}

			// 获取instance配置
			resp := x.namingServer.ServiceInstancesCache(context.TODO(), s)
			if resp.GetCode().Value != api.ExecuteSuccess {
				log.Errorf("[XDSV3] error sync instances for %s, info : %s", svc.Name, resp.Info.GetValue())
				return fmt.Errorf("error sync instances for %s", svc.Name)
			}

			svc.SvcInsRevision = resp.Service.Revision.Value
			svc.Instances = resp.Instances

			// 获取ratelimit配置
			ratelimitResp := x.namingServer.GetRateLimitWithCache(ctx, s)
			if ratelimitResp.GetCode().Value != api.ExecuteSuccess {
				log.Errorf("[XDSV3] error sync ratelimit for %s, info : %s", svc.Name, ratelimitResp.Info.GetValue())
				return fmt.Errorf("error sync ratelimit for %s", svc.Name)
			}
			if ratelimitResp.RateLimit != nil {
				svc.SvcRateLimitRevision = ratelimitResp.RateLimit.Revision.Value
				svc.RateLimit = ratelimitResp.RateLimit
			}
		}
	}

	return nil
}

func (x *XDSServer) initRegistryInfo() error {
	namespaceServer, err := namespace.GetOriginServer()
	if err != nil {
		return err
	}

	resp := namespaceServer.GetNamespaces(context.Background(), make(map[string][]string))
	if resp.Code.Value != api.ExecuteSuccess {
		return fmt.Errorf("error to init registry info %s", resp.Code)
	}
	namespaces := resp.Namespaces
	// 启动时，获取全量的 namespace 信息，用来推送空配置
	for _, namespace := range namespaces {
		x.registryInfo[namespace.Name.Value] = []*ServiceInfo{}
	}

	return nil
}

func (x *XDSServer) startSynTask(ctx context.Context) error {
	// 读取 polaris 缓存数据
	synXdsConfFunc := func() {
		registryInfo := make(map[string][]*ServiceInfo)

		err := x.getRegistryInfoWithCache(ctx, registryInfo)
		if err != nil {
			log.Errorf("get registry info from cache error %v", err)
			return
		}

		needPush := make(map[string][]*ServiceInfo)

		// 处理删除 ns 中最后一个 service
		for ns, infos := range x.registryInfo {
			_, ok := registryInfo[ns]
			if !ok && len(infos) > 0 {
				// 这一次轮询时，该命名空间下的最后一个服务已经被删除了，此时，当前的命名空间需要处理
				needPush[ns] = []*ServiceInfo{}
				x.registryInfo[ns] = []*ServiceInfo{}
			}
		}

		// 与本地缓存对比，是否发生了变化，对发生变化的命名空间，推送配置
		for ns, infos := range registryInfo {
			cacheServiceInfos, ok := x.registryInfo[ns]
			if !ok {
				// 新命名空间，需要处理
				needPush[ns] = infos
				x.registryInfo[ns] = infos
				continue
			}

			// todo 不考虑命名空间删除的情况
			// 判断当前这个空间，是否需要更新配置
			if x.checkUpdate(infos, cacheServiceInfos) {
				needPush[ns] = infos
				x.registryInfo[ns] = infos
			}
		}

		if len(needPush) > 0 {
			_ = x.pushRegistryInfoToXDSCache(needPush)
		}
	}

	go func() {
		ticker := time.NewTicker(5 * cache.UpdateCacheInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				synXdsConfFunc()
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (x *XDSServer) checkUpdate(curServiceInfo, cacheServiceInfo []*ServiceInfo) bool {
	if len(curServiceInfo) != len(cacheServiceInfo) {
		return true
	}

	for _, info := range curServiceInfo {
		find := false
		for _, serviceInfo := range cacheServiceInfo {
			if info.Name == serviceInfo.Name {
				// 通过 revision 判断
				if info.SvcInsRevision != serviceInfo.SvcInsRevision {
					return true
				}
				if info.SvcRoutingRevision != serviceInfo.SvcRoutingRevision {
					return true
				}
				if info.SvcRateLimitRevision != serviceInfo.SvcRateLimitRevision {
					return true
				}

				find = true
			}
		}
		if !find {
			return true
		}
	}

	return false
}
