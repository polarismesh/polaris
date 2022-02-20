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
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcp "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimeservice "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretservice "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
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

	"github.com/polarismesh/polaris-server/apiserver"
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/connlimit"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/service"
)

const K8sDnsResolveSuffixSvc = ".svc"
const K8sDnsResolveSuffixSvcCluster = ".svc.cluster"
const K8sDnsResolveSuffixSvcClusterLocal = ".svc.cluster.local"

// XDSServer
type XDSServer struct {
	listenIP        string
	listenPort      uint32
	start           bool
	restart         bool
	exitCh          chan struct{}
	namingServer    *service.Server
	cache           cachev3.SnapshotCache
	versionNum      *atomic.Uint64
	server          *grpc.Server
	connLimitConfig *connlimit.Config

	registryInfo map[string][]*ServiceInfo
}

// PolarisNodeHash 存放 hash 方法
type PolarisNodeHash struct{}

// ID id 的格式是 namespace/uuid~hostIp
func (PolarisNodeHash) ID(node *envoy_config_core_v3.Node) string {
	if node == nil {
		return ""
	}
	if node.Id == "" || !strings.Contains(node.Id, "/") {
		return ""
	}
	// 每个命名空间下的 envoy node 拥有相同的服务视图
	namespace := strings.Split(node.Id, "/")[0]

	return namespace
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
	ID                 string
	Name               string
	Namespace          string
	Instances          []*api.Instance
	SvcInsRevision     string
	Routing            *api.Routing
	SvcRoutingRevision string
	Ports              string
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

		var consecutiveErrConfig *api.CbPolicy_ConsecutiveErrConfig
		var errorRateConfig *api.CbPolicy_ErrRateConfig
		var policy *api.CbPolicy
		var dest *api.DestinationSet

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

	// 默认 passthrough cluster
	passthroughClsuter := &cluster.Cluster{
		Name:                 "PassthroughCluster",
		ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
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

	// 每一个 polaris service 对应一个 envoy cluster
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

	// 路由目前只处理 inbounds
	if serviceInfo.Routing != nil && len(serviceInfo.Routing.Inbounds) > 0 {
		for _, r := range serviceInfo.Routing.Inbounds {

			// 目前只支持从 header 中取 metadata
			var headerMatchers []*route.HeaderMatcher

			// 使用 sources 生成 routeMatch
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

			// 使用 destinations 生成 weightedClusters。makeClusters() 也使用这个字段生成对应的 subset
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

	// 如果没有路由，会进入最后的默认处理
	routes = append(routes, getDefaultRoute(serviceInfo.Name))
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
		}}
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

func makeVirtualHosts(services []*ServiceInfo) []types.Resource {
	// 每个 polaris service 对应一个 virtualHost
	var routeConfs []types.Resource
	var hosts []*route.VirtualHost

	for _, service := range services {

		hosts = append(hosts, &route.VirtualHost{
			Name:    service.Name,
			Domains: generateServiceDomains(service),
			Routes:  makeRoutes(service),
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
		resources[resource.RouteType] = makeVirtualHosts(registryInfo[ns])
		resources[resource.ListenerType] = makeListeners()
		snapshot, err := cachev3.NewSnapshot(versionLocal, resources)
		if err != nil {
			log.Errorf("fail to create snapshot for %s, err is %v", ns, err)
			return err
		}
		// 检查 snapshot 一致性
		if err := snapshot.Consistent(); err != nil {
			log.Errorf("snapshot inconsistency: %v, err is %v", snapshot, err)
			return err
		}

		log.Infof("will serve ns: %s ,snapshot: %+v", ns, snapshot)

		// 为每个 ns 刷写 cache ，推送 xds 更新
		if err := x.cache.SetSnapshot(context.Background(), ns, snapshot); err != nil {
			log.Errorf("snapshot error %q for %+v", err, snapshot)
			return err
		}
	}
	return nil
}

// syncPolarisServiceInfo 初始化本地 cache，初始化 xds cache
func (x *XDSServer) getRegistryInfoWithCache(ctx context.Context, registryInfo map[string][]*ServiceInfo) error {

	// 从 cache 中获取全量的服务信息
	serviceIterProc := func(key string, value *model.Service) (bool, error) {

		if _, ok := registryInfo[value.Namespace]; !ok {
			registryInfo[value.Namespace] = []*ServiceInfo{}
		}

		registryInfo[value.Namespace] = append(registryInfo[value.Namespace], &ServiceInfo{
			ID:        value.ID,
			Name:      value.Name,
			Namespace: value.Namespace,
			Instances: []*api.Instance{},
			Ports:     value.Ports,
		})

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

			routeResp := x.namingServer.GetRoutingConfigWithCache(ctx, s)
			if routeResp.GetCode().Value != api.ExecuteSuccess {
				log.Errorf("error sync instances for %s", svc.Name)
				return fmt.Errorf("error sync instances for %s", svc.Name)
			}

			if routeResp.Routing != nil {
				svc.SvcRoutingRevision = routeResp.Routing.Revision.Value
				svc.Routing = routeResp.Routing
			}

			resp := x.namingServer.ServiceInstancesCache(nil, s)
			if resp.GetCode().Value != api.ExecuteSuccess {
				log.Errorf("error sync instances for %s", svc.Name)
				return fmt.Errorf("error sync instances for %s", svc.Name)
			}

			svc.SvcInsRevision = resp.Service.Revision.Value
			svc.Instances = resp.Instances
		}
	}

	return nil
}

func (x *XDSServer) initRegistryInfo() error {
	resp := x.namingServer.GetNamespaces(make(map[string][]string))
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

// Initialize
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
				// 通过 revision 判断
				if info.SvcInsRevision != serviceInfo.SvcInsRevision {
					return true
				}
				if info.SvcRoutingRevision != serviceInfo.SvcRoutingRevision {
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

// Run
func (x *XDSServer) Run(errCh chan error) {

	// 启动 grpc server
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

// Stop
func (x *XDSServer) Stop() {
	connlimit.RemoveLimitListener(x.GetProtocol())
	if x.server != nil {
		x.server.Stop()
	}
}

// Restart
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
