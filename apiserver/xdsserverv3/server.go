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
	"math"
	"net"
	"strings"
	"time"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_common_ratelimit_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	ratelimitv32 "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
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
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/apiserver"
	"github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	connlimit "github.com/polarismesh/polaris/common/conn/limit"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/service"
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

	xdsNodesMgr                *XDSNodeManager
	registryInfo               map[string][]*ServiceInfo
	CircuitBreakerConfigGetter CircuitBreakerConfigGetter
	RatelimitConfigGetter      RatelimitConfigGetter
}

// Initialize 初始化
func (x *XDSServer) Initialize(ctx context.Context, option map[string]interface{},
	apiConf map[string]apiserver.APIConfig) error {
	x.registryInfo = make(map[string][]*ServiceInfo)
	x.listenPort = uint32(option["listenPort"].(int))
	x.listenIP = option["listenIP"].(string)
	x.xdsNodesMgr = newXDSNodeManager()
	x.cache = NewSnapshotCache(cachev3.NewSnapshotCache(false, PolarisNodeHash{},
		commonlog.GetScopeOrDefaultByName(commonlog.XDSLoggerName)), x)

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

	if err = x.initRegistryInfo(); err != nil {
		log.Errorf("initRegistryInfo %v", err)
		return err
	}

	if err = x.getRegistryInfoWithCache(ctx, x.registryInfo); err != nil {
		log.Errorf("getRegistryInfoWithCache %v", err)
		return err
	}
	if err = x.pushSidecarInfoToXDSCache(x.registryInfo); err != nil {
		log.Errorf("pushSidecarInfoToXDSCache %v", err)
		return err
	}
	if err = x.pushGatewayInfoToXDSCache(x.registryInfo); err != nil {
		log.Errorf("pushGatewayInfoToXDSCache %v", err)
		return err
	}

	_ = x.startSynTask(ctx)
	return nil
}

// Run 启动运行
func (x *XDSServer) Run(errCh chan error) {
	// 启动 grpc server
	ctx := context.Background()
	cb := &Callbacks{log: commonlog.GetScopeOrDefaultByName(commonlog.XDSLoggerName), nodeMgr: x.xdsNodesMgr}
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
func (x *XDSServer) Restart(option map[string]interface{}, apiConf map[string]apiserver.APIConfig,
	errCh chan error) error {

	log.Infof("restart xds server with new config: +%v", option)

	x.restart = true
	x.Stop()
	if x.start {
		<-x.exitCh
	}

	log.Info("old xds server has stopped, begin restarting it")
	if err := x.Initialize(context.Background(), option, apiConf); err != nil {
		log.Errorf("restart grpc server err: %s", err.Error())
		return err
	}

	log.Info("init grpc server successfully, restart it")
	x.restart = false
	go x.Run(errCh)

	return nil
}

type RatelimitConfigGetter func(serviceKey model.ServiceKey) ([]*model.RateLimit, string)

// GetProtocol 服务注册到北极星中的协议
func (x *XDSServer) GetProtocol() string {
	return "xdsv3"
}

// GetPort 服务注册到北极星中的端口
func (x *XDSServer) GetPort() uint32 {
	return x.listenPort
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
	for _, n := range namespaces {
		x.registryInfo[n.Name.Value] = []*ServiceInfo{}
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

		info := &ServiceInfo{
			ID:        value.ID,
			Name:      value.Name,
			Namespace: value.Namespace,
			Instances: []*apiservice.Instance{},
			Ports:     value.Ports,
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
			s := &apiservice.Service{
				Name:      utils.NewStringValue(svc.Name),
				Namespace: utils.NewStringValue(svc.Namespace),
				Revision:  utils.NewStringValue("-1"),
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
			resp := x.namingServer.ServiceInstancesCache(ctx, s)
			if resp.GetCode().Value != api.ExecuteSuccess {
				log.Errorf("[XDSV3] error sync instances for %s, info : %s", svc.Name, resp.Info.GetValue())
				return fmt.Errorf("error sync instances for %s", svc.Name)
			}

			svc.AliasFor = x.namingServer.Cache().Service().GetAliasFor(svc.Name, svc.Namespace)
			svc.SvcInsRevision = resp.Service.Revision.Value
			svc.Instances = resp.Instances
			ports := x.namingServer.Cache().Instance().GetServicePorts(svc.ID)
			if svc.AliasFor != nil {
				ports = x.namingServer.Cache().Instance().GetServicePorts(svc.AliasFor.ID)
			}
			if len(ports) > 0 {
				svc.Ports = strings.Join(ports, ",")
			}

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
			_ = x.pushSidecarInfoToXDSCache(needPush)
			_ = x.pushGatewayInfoToXDSCache(needPush)
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

func buildLocalRateLimitDescriptors(rule *traffic_manage.Rule) ([]*route.RateLimit_Action,
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
					HeaderValueMatch: buildRateLimitActionHeaderValueMatch(":path", rule.GetMethod()),
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
				headerValueMatch := buildRateLimitActionHeaderValueMatch(arg.Key, arg.Value)
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
				queryParameterValueMatch := buildRateLimitActionQueryParameterValueMatch(arg.Key, arg.Value)
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
