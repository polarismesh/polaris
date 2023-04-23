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

package testsuit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v2"

	"github.com/polarismesh/polaris/auth"
	_ "github.com/polarismesh/polaris/auth/defaultauth"
	"github.com/polarismesh/polaris/cache"
	_ "github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/log"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
	ns "github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/plugin"
	_ "github.com/polarismesh/polaris/plugin/cmdb/memory"
	_ "github.com/polarismesh/polaris/plugin/discoverevent/local"
	_ "github.com/polarismesh/polaris/plugin/healthchecker/memory"
	_ "github.com/polarismesh/polaris/plugin/healthchecker/redis"
	_ "github.com/polarismesh/polaris/plugin/history/logger"
	_ "github.com/polarismesh/polaris/plugin/password"
	_ "github.com/polarismesh/polaris/plugin/ratelimit/lrurate"
	_ "github.com/polarismesh/polaris/plugin/ratelimit/token"
	_ "github.com/polarismesh/polaris/plugin/statis/logger"
	_ "github.com/polarismesh/polaris/plugin/statis/prometheus"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/batch"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store"
	_ "github.com/polarismesh/polaris/store/boltdb"
	testdata "github.com/polarismesh/polaris/test/data"
)

const (
	tblNameNamespace          = "namespace"
	tblNameInstance           = "instance"
	tblNameService            = "service"
	tblNameRouting            = "routing"
	tblRateLimitConfig        = "ratelimit_config"
	tblRateLimitRevision      = "ratelimit_revision"
	tblCircuitBreaker         = "circuitbreaker_rule"
	tblCircuitBreakerRelation = "circuitbreaker_rule_relation"
	tblNameL5                 = "l5"
	tblNameRoutingV2          = "routing_config_v2"
	tblClient                 = "client"
)

type Bootstrap struct {
	Logger map[string]*commonlog.Options
}

type TestConfig struct {
	Bootstrap    Bootstrap          `yaml:"bootstrap"`
	Cache        cache.Config       `yaml:"cache"`
	Namespace    ns.Config          `yaml:"namespace"`
	Naming       service.Config     `yaml:"naming"`
	Config       config.Config      `yaml:"config"`
	HealthChecks healthcheck.Config `yaml:"healthcheck"`
	Store        store.Config       `yaml:"store"`
	Auth         auth.Config        `yaml:"auth"`
	Plugin       plugin.Config      `yaml:"plugin"`
}

type DiscoverTestSuit struct {
	cfg                 *TestConfig
	server              service.DiscoverServer
	originSvr           service.DiscoverServer
	healthCheckServer   *healthcheck.Server
	namespaceSvr        ns.NamespaceOperateServer
	cancelFlag          bool
	updateCacheInterval time.Duration
	DefaultCtx          context.Context
	cancel              context.CancelFunc
	Storage             store.Store
}

func (d *DiscoverTestSuit) DiscoverServer() service.DiscoverServer {
	return d.server
}

func (d *DiscoverTestSuit) OriginDiscoverServer() service.DiscoverServer {
	return d.originSvr
}

func (d *DiscoverTestSuit) HealthCheckServer() *healthcheck.Server {
	return d.healthCheckServer
}

func (d *DiscoverTestSuit) NamespaceServer() ns.NamespaceOperateServer {
	return d.namespaceSvr
}

func (d *DiscoverTestSuit) UpdateCacheInterval() time.Duration {
	return d.updateCacheInterval
}

// 加载配置
func (d *DiscoverTestSuit) loadConfig() error {

	d.cfg = new(TestConfig)

	confFileName := testdata.Path("service_test.yaml")
	if os.Getenv("STORE_MODE") == "sqldb" {
		fmt.Printf("run store mode : sqldb\n")
		confFileName = testdata.Path("service_test_sqldb.yaml")
		d.DefaultCtx = context.WithValue(d.DefaultCtx, utils.ContextAuthTokenKey,
			"nu/0WRA4EqSR1FagrjRj0fZwPXuGlMpX+zCuWu4uMqy8xr1vRjisSbA25aAC3mtU8MeeRsKhQiDAynUR09I=")
	}
	file, err := os.Open(confFileName)
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return err
	}

	err = yaml.NewDecoder(file).Decode(d.cfg)
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return err
	}

	return err
}

// 判断一个resp是否执行成功
func RespSuccess(resp api.ResponseMessage) bool {

	ret := api.CalcCode(resp) == 200

	return ret
}

type options func(cfg *TestConfig)

func (d *DiscoverTestSuit) Initialize(opts ...options) error {
	return d.initialize(opts...)
}

// 内部初始化函数
func (d *DiscoverTestSuit) initialize(opts ...options) error {
	eventhub.TestInitEventHub()
	// 初始化defaultCtx
	d.DefaultCtx = context.WithValue(context.Background(), utils.StringContext("request-id"), "test-1")
	d.DefaultCtx = context.WithValue(d.DefaultCtx, utils.ContextAuthTokenKey,
		"nu/0WRA4EqSR1FagrjRj0fZwPXuGlMpX+zCuWu4uMqy8xr1vRjisSbA25aAC3mtU8MeeRsKhQiDAynUR09I=")

	if err := os.RemoveAll("polaris.bolt"); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}
	}

	if err := d.loadConfig(); err != nil {
		panic(err)
	}

	for i := range opts {
		opts[i](d.cfg)
	}

	_ = commonlog.Configure(d.cfg.Bootstrap.Logger)

	commonlog.GetScopeOrDefaultByName(commonlog.DefaultLoggerName).SetOutputLevel(commonlog.ErrorLevel)
	commonlog.GetScopeOrDefaultByName(commonlog.NamingLoggerName).SetOutputLevel(commonlog.ErrorLevel)
	commonlog.GetScopeOrDefaultByName(commonlog.CacheLoggerName).SetOutputLevel(commonlog.ErrorLevel)
	commonlog.GetScopeOrDefaultByName(commonlog.StoreLoggerName).SetOutputLevel(commonlog.ErrorLevel)
	commonlog.GetScopeOrDefaultByName(commonlog.AuthLoggerName).SetOutputLevel(commonlog.ErrorLevel)

	metrics.InitMetrics()
	eventhub.InitEventHub()

	// 初始化存储层
	store.SetStoreConfig(&d.cfg.Store)
	s, _ := store.TestGetStore()
	d.Storage = s

	plugin.SetPluginConfig(&d.cfg.Plugin)

	ctx, cancel := context.WithCancel(context.Background())

	d.cancel = cancel

	// 初始化缓存模块
	cacheMgn, err := cache.TestCacheInitialize(ctx, &d.cfg.Cache, s)
	if err != nil {
		panic(err)
	}

	// 初始化鉴权层
	authSvr, err := auth.TestInitialize(ctx, &d.cfg.Auth, s, cacheMgn)
	if err != nil {
		panic(err)
	}

	// 初始化命名空间模块
	namespaceSvr, err := ns.TestInitialize(ctx, &d.cfg.Namespace, s, cacheMgn, authSvr)
	if err != nil {
		panic(err)
	}
	d.namespaceSvr = namespaceSvr

	// 批量控制器
	namingBatchConfig, err := batch.ParseBatchConfig(d.cfg.Naming.Batch)
	if err != nil {
		panic(err)
	}
	healthBatchConfig, err := batch.ParseBatchConfig(d.cfg.HealthChecks.Batch)
	if err != nil {
		panic(err)
	}

	batchConfig := &batch.Config{
		Register:         namingBatchConfig.Register,
		Deregister:       namingBatchConfig.Register,
		ClientRegister:   namingBatchConfig.ClientRegister,
		ClientDeregister: namingBatchConfig.ClientDeregister,
		Heartbeat:        healthBatchConfig.Heartbeat,
	}

	bc, err := batch.NewBatchCtrlWithConfig(s, cacheMgn, batchConfig)
	if err != nil {
		log.Errorf("new batch ctrl with config err: %s", err.Error())
		panic(err)
	}
	bc.Start(ctx)

	if len(d.cfg.HealthChecks.LocalHost) == 0 {
		d.cfg.HealthChecks.LocalHost = utils.LocalHost // 补充healthCheck的配置
	}
	healthCheckServer, err := healthcheck.TestInitialize(ctx, &d.cfg.HealthChecks, d.cfg.Cache.Open, bc, d.Storage)
	if err != nil {
		panic(err)
	}
	healthcheck.SetServer(healthCheckServer)
	d.healthCheckServer = healthCheckServer
	cacheProvider, err := healthCheckServer.CacheProvider()
	if err != nil {
		panic(err)
	}
	healthCheckServer.SetServiceCache(cacheMgn.Service())
	healthCheckServer.SetInstanceCache(cacheMgn.Instance())

	// 为 instance 的 cache 添加 健康检查的 Listener
	cacheMgn.AddListener(cache.CacheNameInstance, []cache.Listener{cacheProvider})
	cacheMgn.AddListener(cache.CacheNameClient, []cache.Listener{cacheProvider})

	val, originVal, err := service.TestInitialize(ctx, &d.cfg.Naming, &d.cfg.Cache, bc, cacheMgn, d.Storage, namespaceSvr,
		healthCheckServer, authSvr)
	if err != nil {
		panic(err)
	}
	d.server = val
	d.originSvr = originVal

	// 多等待一会
	d.updateCacheInterval = d.server.Cache().GetUpdateCacheInterval() + time.Millisecond*500

	time.Sleep(5 * time.Second)
	return nil
}

func (d *DiscoverTestSuit) Destroy() {
	eventhub.Shutdown()
	d.cancel()
	time.Sleep(5 * time.Second)

	_ = d.Storage.Destroy()
	time.Sleep(5 * time.Second)

	healthcheck.TestDestroy()
	time.Sleep(5 * time.Second)
}
