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
	"strconv"
	"strings"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_common_ratelimit_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcp "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimeservice "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretservice "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	envoy_type_v3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	testv3 "github.com/envoyproxy/go-control-plane/pkg/test/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	lrl "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/local_ratelimit/v3"
	"github.com/polarismesh/polaris-server/apiserver"
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/connlimit"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/namespace"
	"github.com/polarismesh/polaris-server/service"
)

const K8sDnsResolveSuffixSvc = ".svc"
const K8sDnsResolveSuffixSvcCluster = ".svc.cluster"
const K8sDnsResolveSuffixSvcClusterLocal = ".svc.cluster.local"

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

	registryInfo map[string][]*ServiceInfo
}

// PolarisNodeHash ?????? hash ??????
type PolarisNodeHash struct{}

// ID id ???????????? namespace/uuid~hostIp
func (PolarisNodeHash) ID(node *envoy_config_core_v3.Node) string {
	if node == nil {
		return ""
	}
	if node.Id == "" || !strings.Contains(node.Id, "/") {
		return ""
	}
	// ???????????????????????? envoy node ???????????????????????????
	namespace := strings.Split(node.Id, "/")[0]

	return namespace
}

// GetProtocol ????????????????????????????????????
func (x *XDSServer) GetProtocol() string {
	return "xdsv3"
}

// GetPort ????????????????????????????????????
func (x *XDSServer) GetPort() uint32 {
	return x.listenPort
}

// ServiceInfo ????????????????????????
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
			// ???????????? destination ???????????? subset
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
			outlierDetection.Consecutive_5Xx =
				&wrappers.UInt32Value{Value: consecutiveErrConfig.ConsecutiveErrorToOpen.Value}
		}
		if errorRateConfig != nil {
			outlierDetection.FailurePercentageRequestVolume =
				&wrappers.UInt32Value{Value: errorRateConfig.RequestVolumeThreshold.Value}
			outlierDetection.FailurePercentageThreshold =
				&wrappers.UInt32Value{Value: errorRateConfig.ErrorRateToOpen.Value}
		}

		return outlierDetection
	}
	return nil
}

func (x *XDSServer) makeClusters(services []*ServiceInfo) []types.Resource {
	var clusters []types.Resource
	// ?????? passthrough cluster
	passthroughClsuter := &cluster.Cluster{
		Name:                 "PassthroughCluster",
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

	clusters = append(clusters, passthroughClsuter)

	// ????????? polaris service ???????????? envoy cluster
	for _, service := range services {
		circuitBreakerConf := x.namingServer.Cache().CircuitBreaker().GetCircuitBreakerConfig(service.ID)
		cluster := &cluster.Cluster{
			Name:                 service.Name,
			ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
			ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
			EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
				ServiceName: service.Name,
				EdsConfig: &core.ConfigSource{
					ResourceApiVersion: resource.DefaultAPIVersion,
					ConfigSourceSpecifier: &core.ConfigSource_Ads{
						Ads: &core.AggregatedConfigSource{},
					},
				},
			},
			LbSubsetConfig:   makeLbSubsetConfig(service),
			OutlierDetection: makeOutlierDetection(circuitBreakerConf),
		}

		clusters = append(clusters, cluster)
	}

	return clusters
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
	return meta
}

func makeEndpoints(services []*ServiceInfo) []types.Resource {

	var clusterLoads []types.Resource

	for _, service := range services {

		var lbEndpoints []*endpoint.LbEndpoint
		for _, instance := range service.Instances {
			// ????????????????????????
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

	// ????????????????????? inbounds
	if serviceInfo.Routing != nil && len(serviceInfo.Routing.Inbounds) > 0 {
		for _, r := range serviceInfo.Routing.Inbounds {

			// ?????????????????? header ?????? metadata
			var headerMatchers []*route.HeaderMatcher

			// ?????? sources ?????? routeMatch
			for _, source := range r.Sources {
				if source.Metadata != nil && len(source.Metadata) > 0 {
					for name, matchString := range source.Metadata {
						headerMatch := &route.HeaderMatcher{}
						headerMatch.Name = name
						if matchString.Type == api.MatchString_EXACT {
							headerMatch.HeaderMatchSpecifier = &route.HeaderMatcher_ExactMatch{
								ExactMatch: matchString.Value.Value,
							}
						} else {
							headerMatch.HeaderMatchSpecifier = &route.HeaderMatcher_SuffixMatch{
								SuffixMatch: matchString.Value.Value,
							}
						}
						headerMatchers = append(headerMatchers, headerMatch)
					}
				}
			}

			var weightedClusters []*route.WeightedCluster_ClusterWeight
			var totalWeight uint32

			// ?????? destinations ?????? weightedClusters???makeClusters() ???????????????????????????????????? subset
			for _, destination := range r.Destinations {

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

			route := &route.Route{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
					Headers: headerMatchers,
				},
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
	}

	// ???????????????????????????????????????????????????
	routes = append(routes, getDefaultRoute(serviceInfo.Name))
	return routes
}

// ????????????
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
		}}
}

func generateServiceDomains(serviceInfo *ServiceInfo) []string {
	var domains []string

	// ???????????????
	domains = append(domains, serviceInfo.Name)

	// k8s dns ?????????????????????
	domain := serviceInfo.Name + "." + serviceInfo.Namespace
	domains = append(append(append(append(domains, domain),
		domain+K8sDnsResolveSuffixSvc),
		domain+K8sDnsResolveSuffixSvcCluster),
		domain+K8sDnsResolveSuffixSvcClusterLocal)

	resDomains := domains

	// ????????????????????????????????????
	ports := strings.Split(serviceInfo.Ports, ",")
	for _, port := range ports {
		if _, err := strconv.Atoi(port); err == nil {
			// ??????????????????????????????????????????????????????????????????
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

			// ????????????????????????
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
	// ?????? polaris service ???????????? virtualHost
	var routeConfs []types.Resource
	var hosts []*route.VirtualHost

	for _, service := range services {

		rateLimitConf := x.namingServer.Cache().RateLimit().GetRateLimitByServiceID(service.ID)
		hosts = append(hosts, &route.VirtualHost{
			Name:                 service.Name,
			Domains:              generateServiceDomains(service),
			Routes:               makeRoutes(service),
			TypedPerFilterConfig: makeLocalRateLimit(rateLimitConf),
		})
	}

	// ????????? allow_any
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

func makeListeners() []types.Resource {
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource: &core.ConfigSource{
					ResourceApiVersion: resource.DefaultAPIVersion,
					ConfigSourceSpecifier: &core.ConfigSource_Ads{
						Ads: &core.AggregatedConfigSource{},
					},
				},
				RouteConfigName: "polaris-router",
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
	}

	pbst, err := ptypes.MarshalAny(manager)
	if err != nil {
		panic(err)
	}

	tcpConfig := &tcp.TcpProxy{
		StatPrefix: "PassthroughCluster",
		ClusterSpecifier: &tcp.TcpProxy_Cluster{
			Cluster: "PassthroughCluster",
		},
	}

	tcpC, err := ptypes.MarshalAny(tcpConfig)
	if err != nil {
		panic(err)
	}

	return []types.Resource{
		&listener.Listener{
			Name: "listener_15001",
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.SocketAddress_TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: 15001,
						},
					},
				},
			},
			FilterChains: []*listener.FilterChain{
				{
					Filters: []*listener.Filter{
						{
							Name: wellknown.HTTPConnectionManager,
							ConfigType: &listener.Filter_TypedConfig{
								TypedConfig: pbst,
							},
						},
					},
				},
			},
			DefaultFilterChain: &listener.FilterChain{
				Name: "PassthroughFilterChain",
				Filters: []*listener.Filter{
					{
						Name: wellknown.TCPProxy,
						ConfigType: &listener.Filter_TypedConfig{
							TypedConfig: tcpC,
						},
					},
				},
			},
			ListenerFilters: []*listener.ListenerFilter{
				{
					Name: "envoy.filters.listener.original_dst",
				},
			},
		}}
}

func (x *XDSServer) pushRegistryInfoToXDSCache(registryInfo map[string][]*ServiceInfo) error {
	versionLocal := time.Now().Format(time.RFC3339) + "/" + strconv.FormatUint(x.versionNum.Inc(), 10)

	for ns := range registryInfo {
		resources := make(map[resource.Type][]types.Resource)
		resources[resource.EndpointType] = makeEndpoints(registryInfo[ns])
		resources[resource.ClusterType] = x.makeClusters(registryInfo[ns])
		resources[resource.RouteType] = x.makeVirtualHosts(registryInfo[ns])
		resources[resource.ListenerType] = makeListeners()
		snapshot, err := cachev3.NewSnapshot(versionLocal, resources)
		if err != nil {
			log.Errorf("fail to create snapshot for %s, err is %v", ns, err)
			return err
		}
		// ?????? snapshot ?????????
		if err := snapshot.Consistent(); err != nil {
			log.Errorf("snapshot inconsistency: %v, err is %v", snapshot, err)
			return err
		}

		log.Infof("will serve ns: %s ,snapshot: %+v", ns, snapshot)

		// ????????? ns ?????? cache ????????? xds ??????
		if err := x.cache.SetSnapshot(context.Background(), ns, snapshot); err != nil {
			log.Errorf("snapshot error %q for %+v", err, snapshot)
			return err
		}
	}
	return nil
}

// syncPolarisServiceInfo ??????????????? cache???????????? xds cache
func (x *XDSServer) getRegistryInfoWithCache(ctx context.Context, registryInfo map[string][]*ServiceInfo) error {

	// ??? cache ??????????????????????????????
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

	// ?????????????????????????????????????????????????????????????????????????????????
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

			// ??????routing??????
			routeResp := x.namingServer.GetRoutingConfigWithCache(ctx, s)
			if routeResp.GetCode().Value != api.ExecuteSuccess {
				log.Errorf("error sync routing for %s, info : %s", svc.Name, routeResp.Info.GetValue())
				return fmt.Errorf("[XDSV3] error sync routing for %s", svc.Name)
			}

			if routeResp.Routing != nil {
				svc.SvcRoutingRevision = routeResp.Routing.Revision.Value
				svc.Routing = routeResp.Routing
			}

			// ??????instance??????
			resp := x.namingServer.ServiceInstancesCache(nil, s)
			if resp.GetCode().Value != api.ExecuteSuccess {
				log.Errorf("[XDSV3] error sync instances for %s, info : %s", svc.Name, resp.Info.GetValue())
				return fmt.Errorf("error sync instances for %s", svc.Name)
			}

			svc.SvcInsRevision = resp.Service.Revision.Value
			svc.Instances = resp.Instances

			// ??????ratelimit??????
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
	// ??????????????????????????? namespace ??????????????????????????????
	for _, namespace := range namespaces {
		x.registryInfo[namespace.Name.Value] = []*ServiceInfo{}
	}

	return nil
}

// Initialize ?????????
func (x *XDSServer) Initialize(ctx context.Context, option map[string]interface{},
	api map[string]apiserver.APIConfig) error {

	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	l := logger.Sugar()

	x.cache = cachev3.NewSnapshotCache(false, PolarisNodeHash{}, l)
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

	x.startSynTask(ctx)

	return nil
}

func (x *XDSServer) startSynTask(ctx context.Context) error {

	// ?????? polaris ????????????
	synXdsConfFunc := func() {
		registryInfo := make(map[string][]*ServiceInfo)

		err := x.getRegistryInfoWithCache(ctx, registryInfo)
		if err != nil {
			log.Errorf("get registry info from cache error %v", err)
			return
		}

		needPush := make(map[string][]*ServiceInfo)

		// ???????????? ns ??????????????? service
		for ns, infos := range x.registryInfo {
			_, ok := registryInfo[ns]
			if !ok && len(infos) > 0 {
				// ???????????????????????????????????????????????????????????????????????????????????????????????????????????????????????????
				needPush[ns] = []*ServiceInfo{}
				x.registryInfo[ns] = []*ServiceInfo{}
			}
		}

		// ?????????????????????????????????????????????????????????????????????????????????????????????
		for ns, infos := range registryInfo {
			cacheServiceInfos, ok := x.registryInfo[ns]
			if !ok {
				// ??????????????????????????????
				needPush[ns] = infos
				x.registryInfo[ns] = infos
				continue
			}

			// todo ????????????????????????????????????
			// ???????????????????????????????????????????????????
			if x.checkUpdate(infos, cacheServiceInfos) {
				needPush[ns] = infos
				x.registryInfo[ns] = infos
			}
		}

		if len(needPush) > 0 {
			x.pushRegistryInfoToXDSCache(needPush)
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
				// ?????? revision ??????
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

// Run ????????????
func (x *XDSServer) Run(errCh chan error) {

	// ?????? grpc server
	ctx := context.Background()
	cb := &testv3.Callbacks{Debug: true}
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

// Stop ????????????
func (x *XDSServer) Stop() {
	connlimit.RemoveLimitListener(x.GetProtocol())
	if x.server != nil {
		x.server.Stop()
	}
}

// Restart ????????????
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
