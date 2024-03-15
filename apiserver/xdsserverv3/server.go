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
	"net"
	"strconv"
	"time"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	healthservice "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimeservice "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretservice "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/sotw/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc"

	"github.com/polarismesh/polaris/apiserver"
	xdscache "github.com/polarismesh/polaris/apiserver/xdsserverv3/cache"
	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	connlimit "github.com/polarismesh/polaris/common/conn/limit"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/healthcheck"
)

type ResourceServer interface {
	Generate(versionLocal string, registryInfo ServiceInfos)
}

// XDSServer is the xDS server
type XDSServer struct {
	ctx             context.Context
	listenIP        string
	listenPort      uint32
	start           bool
	restart         bool
	exitCh          chan struct{}
	namingServer    service.DiscoverServer
	healthSvr       *healthcheck.Server
	cache           *xdscache.ResourceCache
	versionNum      *atomic.Uint64
	server          *grpc.Server
	connLimitConfig *connlimit.Config

	nodeMgr           *resource.XDSNodeManager
	registryInfo      *utils.AtomicValue[ServiceInfos]
	resourceGenerator *XdsResourceGenerator

	active         *atomic.Bool
	finishCtx      context.Context
	singleFlight   singleflight.Group
	activeNotifier context.Context
	activeFinish   context.CancelFunc
}

// Initialize 初始化
func (x *XDSServer) Initialize(ctx context.Context, option map[string]interface{},
	apiConf map[string]apiserver.APIConfig) error {
	x.registryInfo = utils.NewAtomicValue[ServiceInfos](ServiceInfos{})
	x.listenPort = uint32(option["listenPort"].(int))
	x.listenIP = option["listenIP"].(string)
	x.nodeMgr = resource.NewXDSNodeManager()
	x.cache = xdscache.NewResourceCache(x)
	x.active = atomic.NewBool(false)
	x.versionNum = atomic.NewUint64(0)
	x.ctx = ctx
	x.activeNotifier, x.activeFinish = context.WithCancel(context.Background())
	var err error

	x.namingServer, err = service.GetOriginServer()
	if err != nil {
		log.Errorf("%v", err)
		return err
	}
	x.healthSvr, err = healthcheck.GetServer()
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
	x.resourceGenerator = &XdsResourceGenerator{
		namingServer:    x.namingServer,
		cache:           x.cache,
		versionNum:      x.versionNum,
		xdsNodesMgr:     x.nodeMgr,
		svcInfoProvider: x.fetchCurrentServices,
	}
	resource.Init()
	return nil
}

// Run 启动运行
func (x *XDSServer) Run(errCh chan error) {
	// 启动 grpc server
	ctx := context.Background()
	cb := xdscache.NewCallback(x.cache, x.nodeMgr)
	srv := serverv3.NewServer(ctx, x.cache, cb, sotw.WithOrderedADS())
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

	registerServer(grpcServer, srv, x)
	log.Infof("management server listening on %d\n", x.listenPort)
	if err = grpcServer.Serve(listener); err != nil {
		log.Errorf("%v", err)
		errCh <- err
		return
	}
	log.Info("xds server stop")
}

func registerServer(grpcServer *grpc.Server, server serverv3.Server, x *XDSServer) {
	// register services
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	routeservice.RegisterVirtualHostDiscoveryServiceServer(grpcServer, server)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	secretservice.RegisterSecretDiscoveryServiceServer(grpcServer, server)
	runtimeservice.RegisterRuntimeDiscoveryServiceServer(grpcServer, server)
	healthservice.RegisterHealthDiscoveryServiceServer(grpcServer, x)
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

// GetProtocol 服务注册到北极星中的协议
func (x *XDSServer) GetProtocol() string {
	return "xdsv3"
}

// GetPort 服务注册到北极星中的端口
func (x *XDSServer) GetPort() uint32 {
	return x.listenPort
}

func (x *XDSServer) activeUpdateTask() {
	if !x.active.CompareAndSwap(false, true) {
		<-x.activeNotifier.Done()
		return
	}
	defer x.activeFinish()
	log.Info("active update xds resource snapshot task")

	if err := x.initRegistryInfo(); err != nil {
		log.Errorf("initRegistryInfo %v", err)
		return
	}

	if err := x.getRegistryInfoWithCache(x.ctx, x.registryInfo.Load()); err != nil {
		log.Errorf("getRegistryInfoWithCache %v", err)
		return
	}
	// 首次更新没有需要移除的 XDS 资源信息
	x.Generate(x.registryInfo.Load(), nil)
	go x.startSynTask(x.ctx)
}

func (x *XDSServer) startSynTask(ctx context.Context) {
	// 读取 polaris 缓存数据
	synXdsConfFunc := func() {
		curRegistryInfo := make(ServiceInfos)
		if err := x.getRegistryInfoWithCache(ctx, curRegistryInfo); err != nil {
			log.Error("get registry info from cache", zap.Error(err))
			return
		}

		needPush := make(ServiceInfos)
		needRemove := make(ServiceInfos)

		// 与本地缓存对比，是否发生了变化，对发生变化的命名空间，推送配置
		oldRegistryInfo := x.registryInfo.Load()

		// step 1: 这里先生成需要删除 XDS 资源数据的资源信息
		for ns, infos := range oldRegistryInfo {
			// 如果当前整个命名空间都不存在，直接按照整个 namespace 级别进行数据删除
			if _, exist := curRegistryInfo[ns]; !exist {
				needRemove[ns] = infos
				continue
			}

			cacheServiceInfos := curRegistryInfo[ns]
			// 命名空间存在，但是命名空间下的服务有删除情况，需要找出来
			for _, info := range infos {
				if _, ok := cacheServiceInfos[info.ServiceKey]; ok {
					continue
				}
				if _, ok := needRemove[ns]; !ok {
					needRemove[ns] = make(map[model.ServiceKey]*resource.ServiceInfo)
				}
				needRemove[ns][info.ServiceKey] = info
			}
		}

		for ns, infos := range curRegistryInfo {
			cacheServiceInfos, ok := oldRegistryInfo[ns]
			if !ok {
				// 新命名空间，需要处理
				needPush[ns] = infos
				continue
			}
			for _, info := range infos {
				oldSvc, exist := cacheServiceInfos[info.ServiceKey]
				// 如果原来的 cache 不存在，直接就是需要推送
				showPush := !exist
				if exist {
					// 如果原来的 cache 存在，这需要在比对下数据是否出现变化
					showPush = !info.Equal(oldSvc)
				}
				if showPush {
					if _, ok := needPush[ns]; !ok {
						needPush[ns] = make(map[model.ServiceKey]*resource.ServiceInfo)
					}
					needPush[ns][info.ServiceKey] = info
				}
			}
		}

		x.registryInfo.Store(curRegistryInfo)
		if len(needPush) > 0 || len(needRemove) > 0 {
			log.Info("start update xds resource snapshot ticker task", zap.Int("need-push", len(needPush)),
				zap.Int("need-remove", len(needRemove)))
			x.Generate(needPush, needRemove)
		}
	}

	ticker := time.NewTicker(5 * cache.UpdateCacheInterval)
	for {
		select {
		case <-ticker.C:
			synXdsConfFunc()
		case <-ctx.Done():
			ticker.Stop()
			log.Info("stop update xds resource snapshot ticker task")
			return
		}
	}
}

func (x *XDSServer) fetchCurrentServices() ServiceInfos {
	return x.registryInfo.Load()
}

func (x *XDSServer) initRegistryInfo() error {
	cur := map[string]map[model.ServiceKey]*resource.ServiceInfo{}
	namespaces := x.namingServer.Cache().Namespace().GetNamespaceList()
	// 启动时，获取全量的 namespace 信息，用来推送空配置
	for _, n := range namespaces {
		cur[n.Name] = map[model.ServiceKey]*resource.ServiceInfo{}
	}
	x.registryInfo.Store(cur)
	return nil
}

// syncPolarisServiceInfo 初始化本地 cache，初始化 xds cache
func (x *XDSServer) getRegistryInfoWithCache(ctx context.Context,
	registryInfo ServiceInfos) error {

	// 从 cache 中获取全量的服务信息
	serviceIterProc := func(key string, value *model.Service) (bool, error) {
		if _, ok := registryInfo[value.Namespace]; !ok {
			registryInfo[value.Namespace] = map[model.ServiceKey]*resource.ServiceInfo{}
		}

		svcKey := model.ServiceKey{
			Namespace: value.Namespace,
			Name:      value.Name,
		}

		info := &resource.ServiceInfo{
			ID:         value.ID,
			Name:       value.Name,
			Namespace:  value.Namespace,
			ServiceKey: svcKey,
			Instances:  []*apiservice.Instance{},
			Ports:      value.ServicePorts,
		}
		registryInfo[value.Namespace][svcKey] = info
		return true, nil
	}

	if err := x.namingServer.Cache().Service().IteratorServices(serviceIterProc); err != nil {
		log.Errorf("sync polaris services error %v", err)
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
			routerRule, err := x.namingServer.Cache().RoutingConfig().GetRouterConfigV2("", svc.Name, svc.Namespace)
			if err != nil {
				log.Errorf("error sync routing for namespace(%s) service(%s), info : %s", svc.Namespace,
					svc.Name, err.Error())
				return fmt.Errorf("[XDSV3] error sync routing for %s", svc.Name)
			}

			svc.SvcRoutingRevision = routerRule.GetRevision().GetValue()
			svc.Routing = routerRule

			// 获取instance配置
			resp := x.namingServer.ServiceInstancesCache(ctx, &apiservice.DiscoverFilter{}, s)
			if resp.GetCode().Value != api.ExecuteSuccess {
				log.Errorf("[XDSV3] error sync instances for namespace(%s) service(%s), info : %s",
					svc.Namespace, svc.Name, resp.Info.GetValue())
				return fmt.Errorf("error sync instances for %s", svc.Name)
			}

			svc.AliasFor = x.namingServer.Cache().Service().GetAliasFor(svc.Name, svc.Namespace)
			svc.SvcInsRevision = resp.Service.Revision.Value
			svc.Instances = resp.Instances
			ports := x.namingServer.Cache().Instance().GetServicePorts(svc.ID)
			if svc.AliasFor != nil {
				ports = x.namingServer.Cache().Instance().GetServicePorts(svc.AliasFor.ID)
			}
			svc.Ports = ports

			// 获取ratelimit配置
			ratelimitResp := x.namingServer.GetRateLimitWithCache(ctx, s)
			if ratelimitResp.GetCode().Value != api.ExecuteSuccess {
				log.Errorf("[XDSV3] error sync ratelimit for %s, info : %s", svc.Name,
					ratelimitResp.Info.GetValue())
				return fmt.Errorf("error sync ratelimit for %s", svc.Name)
			}
			if ratelimitResp.RateLimit != nil {
				svc.SvcRateLimitRevision = ratelimitResp.RateLimit.Revision.Value
				svc.RateLimit = ratelimitResp.RateLimit
			}
			// 获取circuitBreaker配置
			circuitBreakerResp := x.namingServer.GetCircuitBreakerWithCache(ctx, s)
			if circuitBreakerResp.GetCode().Value != api.ExecuteSuccess {
				log.Errorf("[XDSV3] error sync circuitBreaker for %s, info : %s",
					svc.Name, circuitBreakerResp.Info.GetValue())
				return fmt.Errorf("error sync circuitBreaker for %s", svc.Name)
			}
			if circuitBreakerResp.CircuitBreaker != nil {
				svc.CircuitBreakerRevision = circuitBreakerResp.CircuitBreaker.Revision.Value
				svc.CircuitBreaker = circuitBreakerResp.CircuitBreaker
			}

			// 获取faultDetect配置
			faultDetectResp := x.namingServer.GetFaultDetectWithCache(ctx, s)
			if faultDetectResp.GetCode().Value != api.ExecuteSuccess {
				log.Errorf("[XDSV3] error sync faultDetect for %s, info : %s",
					svc.Name, faultDetectResp.Info.GetValue())
				return fmt.Errorf("error sync faultDetect for %s", svc.Name)
			}
			if faultDetectResp.FaultDetector != nil {
				svc.FaultDetectRevision = faultDetectResp.FaultDetector.Revision
				svc.FaultDetect = faultDetectResp.FaultDetector
			}
		}
	}

	// 清理 namespace 下没有数据的记录
	for k, v := range registryInfo {
		if len(v) == 0 {
			delete(registryInfo, k)
		}
	}
	return nil
}

func (x *XDSServer) Generate(needPush, needRemove ServiceInfos) {
	versionLocal := time.Now().Format(time.RFC3339) + "/" + strconv.FormatUint(x.versionNum.Inc(), 10)
	x.resourceGenerator.Generate(versionLocal, needPush, needRemove)
}

func (x *XDSServer) DebugHandlers() []model.DebugHandler {
	return []model.DebugHandler{
		{
			Path:    "/debug/apiserver/xds/envoy_nodes",
			Desc:    "Query the list of Envoy nodes, query parameter name is 'type', value is [sidecar, gateway]",
			Handler: x.listXDSNodes,
		},
		{
			Path:    "/debug/apiserver/xds/resources",
			Desc:    "Query the list of Envoy nodes, eg. /debug/apiserver/xds/resources?type=&nodeId=, type is [eds,cds,rds,vhds,lds]",
			Handler: x.listXDSResource,
		},
	}
}
