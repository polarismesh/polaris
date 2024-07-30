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

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/polarismesh/polaris/admin"
	"github.com/polarismesh/polaris/apiserver"
	"github.com/polarismesh/polaris/auth"
	boot_config "github.com/polarismesh/polaris/bootstrap/config"
	"github.com/polarismesh/polaris/cache"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/common/version"
	config_center "github.com/polarismesh/polaris/config"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/batch"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store"
)

var (
	SelfServiceInstance = make([]*apiservice.Instance, 0)
	ConfigFilePath      = ""
	selfHeathChecker    *SelfHeathChecker
)

// Start 启动
func Start(configFilePath string) {
	// 加载配置
	ConfigFilePath = configFilePath
	utils.ConfDir = parseConfDir(configFilePath)
	cfg, err := boot_config.Load(configFilePath)
	if err != nil {
		fmt.Printf("[ERROR] load config fail\n")
		return
	}

	c, err := yaml.Marshal(cfg)
	if err != nil {
		fmt.Printf("[ERROR] config yaml marshal fail\n")
		return
	}
	fmt.Printf(string(c))

	// 初始化日志打印
	err = log.Configure(cfg.Bootstrap.Logger)
	if err != nil {
		fmt.Printf("[ERROR] configure logger fail: %v\n", err)
		return
	}

	// 初始化
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 获取本地IP地址
	ctx, err = acquireLocalhost(ctx, &cfg.Bootstrap.PolarisService)
	if err != nil {
		fmt.Printf("[ERROR] acquire localhost fail: %v\n", err)
		return
	}
	// 设置默认端口信息数据
	acquireLocalPort(ctx, cfg.APIServers)

	metrics.InitMetrics()
	eventhub.InitEventHub()

	// 设置插件配置
	plugin.SetPluginConfig(&cfg.Plugin)

	// 初始化存储层
	store.SetStoreConfig(&cfg.Store)
	var s store.Store
	s, err = store.GetStore()
	if err != nil {
		fmt.Printf("[ERROR] get store fail: %v", err)
		return
	}

	// 开启进入启动流程，初始化插件，加载数据等
	var tx store.Transaction
	tx, err = StartBootstrapInOrder(s, cfg)
	if err != nil {
		// 多次尝试加锁失败
		fmt.Printf("[ERROR] bootstrap fail: %v\n", err)
		return
	}
	err = StartComponents(ctx, cfg)
	if err != nil {
		fmt.Printf("[ERROR] start components fail: %v\n", err)
		return
	}
	errCh := make(chan error, len(cfg.APIServers))
	servers, err := StartServers(ctx, cfg, errCh)
	if err != nil {
		fmt.Printf("[ERROR] start servers fail: %v\n", err)
		return
	}

	if err := polarisServiceRegister(&cfg.Bootstrap.PolarisService, cfg.APIServers); err != nil {
		fmt.Printf("[ERROR] register polaris service fail: %v\n", err)
		return
	}
	_ = FinishBootstrapOrder(tx) // 启动完成，解锁
	fmt.Println("finish starting server")

	// 等待信号量
	WaitSignal(servers, errCh)
	fmt.Println("begin stop server")
}

// StartComponents start health check and naming components
func StartComponents(ctx context.Context, cfg *boot_config.Config) error {
	var err error

	// 获取存储层对象
	s, err := store.GetStore()
	if err != nil {
		log.Errorf("[Naming][Server] can not get store, err: %s", err.Error())
		return errors.New("can not get store")
	}

	// 初始化缓存模块
	if err := cache.Initialize(ctx, &cfg.Cache, s); err != nil {
		return err
	}

	cacheMgn, err := cache.GetCacheManager()
	if err != nil {
		return err
	}

	// 开启灰度规则缓存
	_ = cacheMgn.OpenResourceCache(cachetypes.ConfigEntry{
		Name: cachetypes.GrayName,
	})

	// 初始化鉴权层
	if err = auth.Initialize(ctx, &cfg.Auth, s, cacheMgn); err != nil {
		return err
	}

	userMgn, err := auth.GetUserServer()
	if err != nil {
		return err
	}

	strategyMgn, err := auth.GetStrategyServer()
	if err != nil {
		return err
	}

	// 初始化命名空间模块
	if err := namespace.Initialize(ctx, &cfg.Namespace, s, cacheMgn); err != nil {
		return err
	}

	// 初始化服务发现模块相关功能
	if err := StartDiscoverComponents(ctx, cfg, s, cacheMgn); err != nil {
		return err
	}

	// 初始化配置中心模块相关功能
	if err := StartConfigCenterComponents(ctx, cfg, s, cacheMgn, userMgn, strategyMgn); err != nil {
		return err
	}

	namingSvr, err := service.GetServer()
	if err != nil {
		return err
	}
	healthCheckServer, err := healthcheck.GetServer()
	if err != nil {
		return err
	}

	// 初始化运维操作模块
	if err := admin.Initialize(ctx, &cfg.Maintain, namingSvr, healthCheckServer, cacheMgn, s); err != nil {
		return err
	}

	// 最后启动 cache
	if err := cache.Run(cacheMgn, ctx); err != nil {
		return err
	}
	return nil
}

func StartDiscoverComponents(ctx context.Context, cfg *boot_config.Config, s store.Store,
	cacheMgn *cache.CacheManager) error {
	// 批量控制器
	namingBatchConfig, err := batch.ParseBatchConfig(cfg.Naming.Batch)
	if err != nil {
		return err
	}
	healthBatchConfig, err := batch.ParseBatchConfig(cfg.HealthChecks.Batch)
	if err != nil {
		return err
	}

	batchConfig := &batch.Config{
		Register:         namingBatchConfig.Register,
		Deregister:       namingBatchConfig.Deregister,
		ClientRegister:   namingBatchConfig.ClientRegister,
		ClientDeregister: namingBatchConfig.ClientDeregister,
		Heartbeat:        healthBatchConfig.Heartbeat,
	}

	bc, err := batch.NewBatchCtrlWithConfig(s, cacheMgn, batchConfig)
	if err != nil {
		log.Errorf("new batch ctrl with config err: %s", err.Error())
		return err
	}
	bc.Start(ctx)

	if len(cfg.HealthChecks.LocalHost) == 0 {
		cfg.HealthChecks.LocalHost = utils.LocalHost // 补充healthCheck的配置
	}
	if err = healthcheck.Initialize(ctx, &cfg.HealthChecks, bc); err != nil {
		return err
	}
	healthCheckServer, err := healthcheck.GetServer()
	if err != nil {
		return err
	}
	if cfg.HealthChecks.IsOpen() {
		healthCheckServer.SetServiceCache(cacheMgn.Service())
		healthCheckServer.SetInstanceCache(cacheMgn.Instance())
	}

	namespaceSvr, err := namespace.GetServer()
	if err != nil {
		return err
	}

	opts := []service.InitOption{
		service.WithBatchController(bc),
		service.WithStorage(s),
		service.WithCacheManager(&cfg.Cache, cacheMgn),
		service.WithHealthCheckSvr(healthCheckServer),
		service.WithNamespaceSvr(namespaceSvr),
	}

	cfg.Naming.Interceptors = service.GetChainOrder()
	// 初始化服务模块
	if err = service.Initialize(ctx, &cfg.Naming, opts...); err != nil {
		return err
	}

	if _, err = service.GetServer(); err != nil {
		return err
	}

	return nil
}

func parseConfDir(path string) string {
	slashIndex := strings.LastIndex(path, "/")
	if slashIndex == -1 {
		return "./"
	}
	return path[0 : slashIndex+1]
}

// StartConfigCenterComponents 启动配置中心模块
func StartConfigCenterComponents(ctx context.Context, cfg *boot_config.Config, s store.Store,
	cacheMgn *cache.CacheManager, userMgn auth.UserServer, strategyMgn auth.StrategyServer) error {
	namespaceOperator, err := namespace.GetServer()
	if err != nil {
		return err
	}
	cfg.Config.Interceptors = config_center.GetChainOrder()
	return config_center.Initialize(ctx, cfg.Config, s, cacheMgn, namespaceOperator)
}

// StartServers 启动server
func StartServers(ctx context.Context, cfg *boot_config.Config, errCh chan error) (
	[]apiserver.Apiserver, error) {
	// 启动API服务器
	var servers []apiserver.Apiserver

	// 等待所有ApiServer都监听完成
	for _, protocol := range cfg.APIServers {
		slot, exist := apiserver.Slots[protocol.Name]
		if !exist {
			log.Warn("[ERROR] apiserver slot not exists", zap.String("name", protocol.Name))
			continue
		}
		// 如果是 http server, 注入所有的 apiserver 实例
		if protocol.Name == "api-http" {
			ctx = context.WithValue(ctx, utils.ContextAPIServerSlot{}, apiserver.Slots)
		}

		err := slot.Initialize(ctx, protocol.Option, protocol.API)
		if err != nil {
			fmt.Printf("[ERROR] %v\n", err)
			return nil, fmt.Errorf("apiserver %s initialize err: %s", protocol.Name, err.Error())
		}

		servers = append(servers, slot)
		go slot.Run(errCh)
	}
	return servers, nil
}

// RestartServers 重启server
func RestartServers(errCh chan error) error {
	// 重新加载配置
	cfg, err := boot_config.Load(ConfigFilePath)
	if err != nil {
		log.Infof("restart servers, reload config")
		return err
	}
	log.Infof("new config: %+v", cfg)
	// TODO: 配置的动态加载后续统一设计
	return nil
}

// StopServers 接受外部信号，停止server
func StopServers(servers []apiserver.Apiserver) {
	// stop health checkers
	if nil != selfHeathChecker {
		selfHeathChecker.Stop()
	}
	// sync stop servers
	wg := &sync.WaitGroup{}
	for _, s := range servers {
		wg.Add(1)
		go func(s apiserver.Apiserver, wg *sync.WaitGroup) {
			defer wg.Done()
			log.Infof("start stop server protocol: %s", s.GetProtocol())
			s.Stop()
			log.Infof("complete stop server protocol: %s", s.GetProtocol())
		}(s, wg)
	}
	wg.Wait()
	// deregister instances
	SelfDeregister()
}

// StartBootstrapInOrder 开始进入启动加锁
// 原因：Server启动的时候会从数据库拉取大量数据，防止同时启动把DB压死
// 还有一种场景，server全部宕机批量重启，导致数据库被压死，导致雪崩
func StartBootstrapInOrder(s store.Store, c *boot_config.Config) (store.Transaction, error) {
	order := c.Bootstrap.StartInOrder
	log.Infof("[Bootstrap] get bootstrap order config: %+v", order)
	open, _ := order["open"].(bool)
	key, _ := order["key"].(string)
	if !open || key == "" {
		log.Infof("[Bootstrap] start in order config is not open or key is null")
		return nil, nil
	}

	log.Infof("bootstrap start in order with key: %s", key)

	// 启动一个日志协程，当等锁的时候，可以看到server正在等待锁
	stopCh := make(chan struct{})
	defer close(stopCh) // 函数退出的时候，关闭stopCh
	go func() {
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Infof("bootstrap waiting the lock")
			case <-stopCh:
				return
			}
		}
	}()

	// 重试多次
	maxTimes := 10
	for i := 0; i < maxTimes; i++ {
		tx, err := s.CreateTransaction()
		if err != nil {
			log.Errorf("create transaction err: %v", err)
			return nil, err
		}
		// 这里可能会出现锁超时，超时则重试
		if err := tx.LockBootstrap(key, utils.LocalHost); err != nil {
			log.Errorf("lock bootstrap err: %s", err.Error())
			_ = tx.Commit()
			continue
		}
		// 加锁成功，直接返回
		log.Infof("lock bootstrap success")
		return tx, nil
	}

	return nil, errors.New("lock bootstrap error")
}

// FinishBootstrapOrder 完成 提交锁
func FinishBootstrapOrder(tx store.Transaction) error {
	if tx != nil {
		return tx.Commit()
	}

	return nil
}

func genContext() context.Context {
	ctx := context.Background()
	reqCtx := context.WithValue(context.Background(), utils.ContextAuthTokenKey, "")
	ctx = context.WithValue(ctx, utils.StringContext("request-id"), fmt.Sprintf("self-%d", time.Now().Nanosecond()))
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey,
		authcommon.NewAcquireContext(
			authcommon.WithOperation(authcommon.Read),
			authcommon.WithModule(authcommon.BootstrapModule),
			authcommon.WithRequestContext(reqCtx)))
	return ctx
}

// acquireLocalhost 探测获取本机IP地址
func acquireLocalhost(ctx context.Context, polarisService *boot_config.PolarisService) (context.Context, error) {
	if polarisService == nil || !polarisService.EnableRegister {
		log.Infof("[Bootstrap] polaris service config not found")
		return ctx, nil
	}
	if len(polarisService.SelfAddress) != 0 {
		utils.LocalHost = polarisService.SelfAddress
		return utils.WithLocalhost(ctx, utils.LocalHost), nil
	}
	if len(polarisService.NetworkInter) != 0 {
		netInter, err := net.InterfaceByName(polarisService.NetworkInter)
		if err != nil {
			log.Errorf("get local host by network_interface: %s err: %s", polarisService.NetworkInter, err.Error())
			return nil, err
		}
		addrs, err := netInter.Addrs()
		if err != nil {
			log.Errorf("get local host by network_interface: %s err: %s", polarisService.NetworkInter, err.Error())
			return nil, err
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					utils.LocalHost = ipnet.IP.String()
					return utils.WithLocalhost(ctx, utils.LocalHost), nil
				}
			}
		}
	}

	localHost, err := getLocalHost(polarisService.ProbeAddress)
	if err != nil {
		log.Errorf("get local host err: %s", err.Error())
		return nil, err
	}
	log.Infof("[Bootstrap] get local host: %s", localHost)
	utils.LocalHost = localHost
	return utils.WithLocalhost(ctx, utils.LocalHost), nil
}

func acquireLocalPort(ctx context.Context, apientries []apiserver.Config) {
	for i := range apientries {
		entry := apientries[i]
		if entry.Name != "service-grpc" {
			continue
		}
		port, _ := entry.Option["listenPort"].(int)
		utils.LocalPort = port
		break
	}
}

// polarisServiceRegister 自注册主函数
func polarisServiceRegister(polarisService *boot_config.PolarisService, apiServers []apiserver.Config) error {
	if polarisService == nil || !polarisService.EnableRegister {
		log.Infof("[Bootstrap] not enable register the polaris service")
		return nil
	}

	apiServerNames := make(map[string]bool)
	for _, server := range apiServers {
		apiServerNames[server.Name] = true
	}
	hbInterval := boot_config.DefaultHeartbeatInterval
	if polarisService.HeartbeatInterval > 0 {
		hbInterval = polarisService.HeartbeatInterval
	}
	// 开始注册每个服务
	for _, svc := range polarisService.Services {
		protocols := svc.Protocols
		// 如果service.Protocols为空，默认采用apiServers的protocols注册，实际为配置中的Name字段,
		// 如：grpcserver, httpserver, xdsserverv3，也隐式表达了协议的意思
		if len(protocols) == 0 {
			for _, server := range apiServers {
				protocols = append(protocols, server.Name)
			}
		}
		for _, name := range protocols {
			if _, exist := apiServerNames[name]; !exist {
				return fmt.Errorf("server(%s) not registered", name)
			}
			slot, exist := apiserver.Slots[name]
			if !exist {
				return fmt.Errorf("server(%s) not supported", name)
			}
			host := utils.LocalHost
			port := slot.GetPort()
			protocol := slot.GetProtocol()
			if err := selfRegister(host, port, protocol, polarisService.Isolated, svc, hbInterval); err != nil {
				log.Errorf("self register err: %s", err.Error())
				return err
			}

			log.Infof("self register success. host = %s, port = %d, protocol = %s, service = %s",
				host, port, protocol, svc)
		}
	}
	if len(SelfServiceInstance) > 0 && !polarisService.DisableHeartbeat {
		log.Infof("start self health checker")
		var err error
		if selfHeathChecker, err = NewSelfHeathChecker(SelfServiceInstance, hbInterval); nil != err {
			log.Errorf("self health checker err: %s", err.Error())
			return err
		}
		go selfHeathChecker.Start()
	}
	return nil
}

// selfRegister 服务自注册
func selfRegister(
	host string, port uint32, protocol string, isolated bool, polarisService *boot_config.Service, hbInterval int) error {
	server, err := service.GetServer()
	if err != nil {
		return err
	}

	name := boot_config.DefaultPolarisName
	polarisNamespace := boot_config.DefaultPolarisNamespace
	if polarisService.Name != "" {
		name = polarisService.Name
	}

	if polarisService.Namespace != "" {
		polarisNamespace = polarisService.Namespace
	}

	metadata := polarisService.Metadata
	if len(metadata) == 0 {
		metadata = make(map[string]string)
	}
	metadata[model.MetaKeyBuildRevision] = version.GetRevision()
	metadata[model.MetaKeyPolarisService] = name

	req := &apiservice.Instance{
		Service:           utils.NewStringValue(name),
		Namespace:         utils.NewStringValue(polarisNamespace),
		Host:              utils.NewStringValue(host),
		Port:              utils.NewUInt32Value(port),
		Protocol:          utils.NewStringValue(protocol),
		Version:           utils.NewStringValue(version.Get()),
		EnableHealthCheck: utils.NewBoolValue(true),
		Isolate:           utils.NewBoolValue(isolated),
		HealthCheck: &apiservice.HealthCheck{
			Type: apiservice.HealthCheck_HEARTBEAT,
			Heartbeat: &apiservice.HeartbeatHealthCheck{
				Ttl: &wrappers.UInt32Value{Value: uint32(hbInterval)},
			},
		},
		Metadata: metadata,
	}

	resp := server.RegisterInstance(genContext(), req)
	if api.CalcCode(resp) != 200 {
		// 如果self之前注册过，那么可以忽略
		if resp.GetCode().GetValue() != api.ExistedResource {
			return fmt.Errorf("%s", resp.GetInfo().GetValue())
		}

		resp = server.UpdateInstance(genContext(), req)
		if api.CalcCode(resp) != 200 {
			return fmt.Errorf("%s", resp.GetInfo().GetValue())
		}
	}
	SelfServiceInstance = append(SelfServiceInstance, req)

	return nil
}

// SelfDeregister Server退出的时候，自动反注册
func SelfDeregister() {
	namingServer, err := service.GetOriginServer()
	if err != nil {
		log.Errorf("get naming server obj err: %s", err.Error())
		return
	}
	for _, req := range SelfServiceInstance {
		log.Infof("Deregister the instance(%+v)", req)
		if resp := namingServer.DeleteInstance(genContext(), req); api.CalcCode(resp) != 200 {
			// 遇到失败，继续反注册其他的实例
			log.Errorf("Deregister instance error: %s", resp.GetInfo().GetValue())
		}
	}
	// wait the async event handler to finish
	time.Sleep(5 * time.Second)
}

// getLocalHost 获取本地IP地址
func getLocalHost(addr string) (string, error) {
	if len(addr) == 0 {
		return "127.0.0.1", nil
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = conn.Close()
	}()

	localAddr := conn.LocalAddr().String() // ip:port
	segs := strings.Split(localAddr, ":")
	if len(segs) != 2 {
		return "", errors.New("get local address format is invalid")
	}

	return segs[0], nil
}

// getSelfRegisterPolarsServiceKeySet 获取自注册的系统服务集合
func getSelfRegisterPolarsServiceKeySet(polarisServiceCfg *boot_config.PolarisService) map[model.ServiceKey]struct{} {
	if polarisServiceCfg == nil {
		return nil
	}
	polarisServiceSet := make(map[model.ServiceKey]struct{})
	for _, svc := range polarisServiceCfg.Services {
		ns, n := svc.Namespace, svc.Name
		if ns == "" {
			ns = boot_config.DefaultPolarisNamespace
		}
		polarisServiceSet[model.ServiceKey{Namespace: ns, Name: n}] = struct{}{}
	}
	return polarisServiceSet
}
