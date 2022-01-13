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
	"time"

	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/cache"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/service/healthcheck"

	"github.com/polarismesh/polaris-server/apiserver"
	"github.com/polarismesh/polaris-server/bootstrap/config"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/common/version"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/service"
	"github.com/polarismesh/polaris-server/store"
)

var (
	SelfServiceInstance = make([]*api.Instance, 0)
	ConfigFilePath      = ""
)

/**
 * @brief 启动
 */
func Start(configFilePath string) {
	// 加载配置
	ConfigFilePath = configFilePath
	cfg, err := config.Load(configFilePath)
	if err != nil {
		fmt.Printf("[ERROR] loadConfig fail\n")
		return
	}

	fmt.Printf("%+v\n", *cfg)

	// 初始化日志打印
	err = log.Configure(cfg.Bootstrap.Logger)
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return
	}

	// 初始化
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 获取本地IP地址
	ctx, err = acquireLocalhost(ctx, &cfg.Bootstrap.PolarisService)
	if err != nil {
		fmt.Printf("[ERROR] %s\n", err.Error())
		return
	}

	// 设置插件配置
	plugin.SetPluginConfig(&cfg.Plugin)

	// 初始化存储层
	store.SetStoreConfig(&cfg.Store)
	var s store.Store
	s, err = store.GetStore()
	if err != nil {
		fmt.Printf("get store error:%s", err.Error())
		return
	}

	// 开启进入启动流程，初始化插件，加载数据等
	var tx store.Transaction
	tx, err = StartBootstrapOrder(s, cfg)
	if err != nil {
		// 多次尝试加锁失败
		fmt.Printf("[ERROR] %v\n", err)
		return
	}
	err = StartComponents(ctx, cfg)
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return
	}
	errCh := make(chan error, len(cfg.APIServers))
	servers, err := StartServers(ctx, cfg, errCh)
	if err != nil {
		return
	}

	if err := polarisServiceRegister(&cfg.Bootstrap.PolarisService, cfg.APIServers); err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return
	}
	_ = FinishBootstrapOrder(tx) // 启动完成，解锁
	fmt.Println("finish starting server")

	RunMainLoop(servers, errCh)
}

// StartComponents start healthcheck and naming components
func StartComponents(ctx context.Context, cfg *config.Config) error {
	var err error

	// 获取存储层对象
	s, err := store.GetStore()
	if err != nil {
		log.Errorf("[Naming][Server] can not get store, err: %s", err.Error())
		return errors.New("can not get store")
	}

	if len(cfg.HealthChecks.LocalHost) == 0 {
		cfg.HealthChecks.LocalHost = utils.LocalHost // 补充healthCheck的配置
	}

	if err = healthcheck.Initialize(ctx, &cfg.HealthChecks, cfg.Cache.Open); err != nil {
		return err
	}

	healthCheckServer, err := healthcheck.GetServer()
	if err != nil {
		return err
	}

	cacheProvider, err := healthCheckServer.CacheProvider()
	if err != nil {
		return err
	}

	if err := cache.Initialize(ctx, &cfg.Cache, s, []cache.Listener{cacheProvider}); err != nil {
		return err
	}

	cacheMgn, err := cache.GetCacheManager()
	if err != nil {
		return err
	}

	// 初始化鉴权层
	if err = auth.Initialize(ctx, &cfg.Auth, cacheMgn); err != nil {
		return err
	}

	if err = service.Initialize(ctx, &cfg.Naming, &cfg.Cache, cacheProvider); err != nil {
		return err
	}

	namingSvr, err := service.GetServer()
	if err != nil {
		return err
	}
	healthCheckServer.SetServiceCache(namingSvr.Cache().Service())
	return nil
}

// StartServers 启动server
func StartServers(ctx context.Context, cfg *config.Config, errCh chan error) (
	[]apiserver.Apiserver, error) {
	// 启动API服务器
	var servers []apiserver.Apiserver
	for _, protocol := range cfg.APIServers {
		slot, exist := apiserver.Slots[protocol.Name]
		if !exist {
			fmt.Printf("[ERROR] apiserver slot %s not exists\n", protocol.Name)
			return nil, fmt.Errorf("apiserver slot %s not exists", protocol.Name)
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

// 重启server
func RestartServers(errCh chan error) error {
	// 重新加载配置
	cfg, err := config.Load(ConfigFilePath)
	if err != nil {
		log.Infof("restart servers, reload config")
		return err
	}
	log.Infof("new config: %+v", cfg)

	// 把配置的每个apiserver，进行重启
	for _, protocol := range cfg.APIServers {
		server, exist := apiserver.Slots[protocol.Name]
		if !exist {
			log.Errorf("apiserver slot %s not exists\n", protocol.Name)
			return err
		}
		log.Infof("begin restarting server: %s", protocol.Name)
		if err := server.Restart(protocol.Option, protocol.API, errCh); err != nil {
			return err
		}
	}
	return nil
}

// 接受外部信号，停止server
func StopServers(servers []apiserver.Apiserver) {
	// 先反注册所有服务
	SelfDeregister()

	// 停掉服务
	for _, s := range servers {
		log.Infof("stop server protocol: %s", s.GetProtocol())
		s.Stop()
	}
}

// 开始进入启动加锁
// 原因：Server启动的时候会从数据库拉取大量数据，防止同时启动把DB压死
// 还有一种场景，server全部宕机批量重启，导致数据库被压死，导致雪崩
func StartBootstrapOrder(s store.Store, c *config.Config) (store.Transaction, error) {
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
			log.Errorf("create transaction err: %s", err.Error())
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

// FinishBootstrapOrder
func FinishBootstrapOrder(tx store.Transaction) error {
	if tx != nil {
		return tx.Commit()
	}

	return nil
}

// 生成一个context
func genContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("request-id"), fmt.Sprintf("self-%d", time.Now().Nanosecond()))
	return ctx
}

// 探测获取本机IP地址
func acquireLocalhost(ctx context.Context, polarisService *config.PolarisService) (context.Context, error) {
	if polarisService == nil || !polarisService.EnableRegister {
		log.Infof("[Bootstrap] not found polaris service config")
		return ctx, nil
	}

	localHost, err := getLocalHost(polarisService.ProbeAddress)
	if err != nil {
		log.Errorf("get local host err: %s", err.Error())
		return nil, err
	}
	log.Infof("[Bootstrap] get local host: %s", localHost)
	utils.LocalHost = localHost

	return utils.WithLocalhost(ctx, localHost), nil
}

// 自注册主函数
func polarisServiceRegister(polarisService *config.PolarisService, apiServers []apiserver.Config) error {
	if polarisService == nil || !polarisService.EnableRegister {
		log.Infof("[Bootstrap] not enable register the polaris service")
		return nil
	}

	apiServerNames := make(map[string]bool)
	for _, server := range apiServers {
		apiServerNames[server.Name] = true
	}

	// 开始注册每个服务
	for _, service := range polarisService.Services {
		protocols := service.Protocols
		// 如果service.Protocols为空，默认采用protocols注册
		if len(protocols) == 0 {
			for _, server := range apiServers {
				protocols = append(protocols, server.Name)
			}
		}

		for _, name := range protocols {
			if _, exist := apiServerNames[name]; !exist {
				return fmt.Errorf("not register the server(%s)", name)
			}
			slot, exist := apiserver.Slots[name]
			if !exist {
				return fmt.Errorf("not exist the server(%s)", name)
			}
			host := utils.LocalHost
			port := slot.GetPort()
			protocol := slot.GetProtocol()
			if err := selfRegister(host, port, protocol, polarisService.Isolated, service); err != nil {
				log.Errorf("self register err: %s", err.Error())
				return err
			}
		}
	}

	return nil
}

// 服务自注册
func selfRegister(host string, port uint32, protocol string, isolated bool, polarisService *config.Service) error {
	server, err := service.GetServer()
	if err != nil {
		return err
	}
	storage, err := store.GetStore()
	if err != nil {
		return err
	}

	name := config.DefaultPolarisName
	namespace := config.DefaultPolarisNamespace
	if polarisService.Name != "" {
		name = polarisService.Name
	}

	if polarisService.Namespace != "" {
		namespace = polarisService.Namespace
	}

	service, err := storage.GetService(name, namespace)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("not found the self service(%s), namespace(%s)",
			name, namespace)
	}

	req := &api.Instance{
		Service:      utils.NewStringValue(name),
		Namespace:    utils.NewStringValue(namespace),
		Host:         utils.NewStringValue(host),
		Port:         utils.NewUInt32Value(port),
		Protocol:     utils.NewStringValue(protocol),
		ServiceToken: utils.NewStringValue(service.Token),
		Version:      utils.NewStringValue(version.Get()),
		Isolate:      utils.NewBoolValue(isolated), // 自注册，默认是隔离的
		Metadata: map[string]string{
			model.MetaKeyBuildRevision:  version.GetRevision(),
			model.MetaKeyPolarisService: name,
		},
	}

	resp := server.CreateInstance(genContext(), req)
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
	namingServer, err := service.GetServer()
	if err != nil {
		log.Errorf("get naming server obj err: %s", err.Error())
		return
	}
	for _, req := range SelfServiceInstance {
		log.Infof("Deregister the instance(%+v)", req)
		if resp := namingServer.DeleteInstances(genContext(), []*api.Instance{req}); api.CalcCode(resp) != 200 {
			// 遇到失败，继续反注册其他的实例
			log.Errorf("Deregister instance error: %s", resp.GetInfo().GetValue())
		}
	}

	return
}

// 获取本地IP地址
func getLocalHost(vip string) (string, error) {
	if len(vip) == 0 {
		return "127.0.0.1", nil
	}
	conn, err := net.Dial("tcp", vip)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().String() // ip:port
	segs := strings.Split(localAddr, ":")
	if len(segs) != 2 {
		return "", errors.New("get local address format is invalid")
	}

	return segs[0], nil
}
