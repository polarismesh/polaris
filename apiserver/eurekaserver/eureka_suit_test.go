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

package eurekaserver

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/cache"
	commonlog "github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/namespace"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/service"
	"github.com/polarismesh/polaris-server/service/batch"
	"github.com/polarismesh/polaris-server/service/healthcheck"
	"github.com/polarismesh/polaris-server/store"
	storemock "github.com/polarismesh/polaris-server/store/mock"
	"gopkg.in/yaml.v2"

	// 注册相关默认插件
	_ "github.com/polarismesh/polaris-server/plugin/auth/defaultauth"
	_ "github.com/polarismesh/polaris-server/plugin/auth/platform"
	_ "github.com/polarismesh/polaris-server/plugin/cmdb/memory"
	_ "github.com/polarismesh/polaris-server/plugin/discoverevent/local"
	_ "github.com/polarismesh/polaris-server/plugin/discoverstat/discoverlocal"
	_ "github.com/polarismesh/polaris-server/plugin/healthchecker/heartbeatmemory"
	_ "github.com/polarismesh/polaris-server/plugin/healthchecker/heartbeatredis"
	_ "github.com/polarismesh/polaris-server/plugin/history/logger"
	_ "github.com/polarismesh/polaris-server/plugin/password"
	_ "github.com/polarismesh/polaris-server/plugin/ratelimit/lrurate"
	_ "github.com/polarismesh/polaris-server/plugin/ratelimit/token"
	_ "github.com/polarismesh/polaris-server/plugin/statis/local"
	_ "github.com/polarismesh/polaris-server/store/boltdb"
	_ "github.com/polarismesh/polaris-server/store/sqldb"
)

type Bootstrap struct {
	Logger map[string]*commonlog.Options
}

type TestConfig struct {
	Bootstrap    Bootstrap          `yaml:"bootstrap"`
	Cache        cache.Config       `yaml:"cache"`
	Namespace    namespace.Config   `yaml:"namespace"`
	Naming       service.Config     `yaml:"naming"`
	HealthChecks healthcheck.Config `yaml:"healthcheck"`
	Store        store.Config       `yaml:"store"`
	Auth         auth.Config        `yaml:"auth"`
	Plugin       plugin.Config      `yaml:"plugin"`
}

type EurekaTestSuit struct {
	cfg                 *TestConfig
	server              service.DiscoverServer
	healthSvr           *healthcheck.Server
	namespaceSvr        namespace.NamespaceOperateServer
	cancelFlag          bool
	updateCacheInterval time.Duration
	cancel              context.CancelFunc
	storage             store.Store
}

type options func(cfg *TestConfig)

// 内部初始化函数
func (d *EurekaTestSuit) initialize(t *testing.T, callback func(t *testing.T, s *storemock.MockStore) error, opts ...options) error {
	if err := d.loadConfig(); err != nil {
		return err
	}

	for i := range opts {
		opts[i](d.cfg)
	}

	_ = commonlog.Configure(d.cfg.Bootstrap.Logger)

	commonlog.DefaultScope().SetOutputLevel(commonlog.ErrorLevel)
	commonlog.NamingScope().SetOutputLevel(commonlog.ErrorLevel)
	commonlog.CacheScope().SetOutputLevel(commonlog.ErrorLevel)
	commonlog.StoreScope().SetOutputLevel(commonlog.ErrorLevel)
	commonlog.AuthScope().SetOutputLevel(commonlog.ErrorLevel)

	plugin.SetPluginConfig(&d.cfg.Plugin)

	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel

	// 初始化存储层
	ctrl := gomock.NewController(t)
	s := storemock.NewMockStore(ctrl)
	d.storage = s

	if err := callback(t, s); err != nil {
		return err
	}

	// 初始化缓存模块
	if err := cache.TestCacheInitialize(ctx, &d.cfg.Cache, d.storage); err != nil {
		return err
	}

	cacheMgn, err := cache.GetCacheManager()
	if err != nil {
		return err
	}

	// 批量控制器
	namingBatchConfig, err := batch.ParseBatchConfig(d.cfg.Naming.Batch)
	if err != nil {
		return err
	}
	healthBatchConfig, err := batch.ParseBatchConfig(d.cfg.HealthChecks.Batch)
	if err != nil {
		return err
	}

	batchConfig := &batch.Config{
		Register:         namingBatchConfig.Register,
		Deregister:       namingBatchConfig.Register,
		ClientRegister:   namingBatchConfig.ClientRegister,
		ClientDeregister: namingBatchConfig.ClientDeregister,
		Heartbeat:        healthBatchConfig.Heartbeat,
	}

	bc, err := batch.NewBatchCtrlWithConfig(d.storage, cacheMgn, batchConfig)
	if err != nil {
		log.Errorf("new batch ctrl with config err: %s", err.Error())
		return err
	}
	bc.Start(ctx)

	if len(d.cfg.HealthChecks.LocalHost) == 0 {
		d.cfg.HealthChecks.LocalHost = utils.LocalHost // 补充healthCheck的配置
	}
	healthCheckServer, err := healthcheck.TestInitialize(ctx, &d.cfg.HealthChecks, d.cfg.Cache.Open, bc, d.storage)
	if err != nil {
		return err
	}
	cacheProvider, err := healthCheckServer.CacheProvider()
	if err != nil {
		return err
	}
	healthCheckServer.SetServiceCache(cacheMgn.Service())
	healthCheckServer.SetInstanceCache(cacheMgn.Instance())

	// 为 instance 的 cache 添加 健康检查的 Listener
	cacheMgn.AddListener(cache.CacheNameInstance, []cache.Listener{cacheProvider})
	cacheMgn.AddListener(cache.CacheNameClient, []cache.Listener{cacheProvider})

	d.healthSvr = healthCheckServer
	time.Sleep(5 * time.Second)
	return nil
}

// 加载配置
func (d *EurekaTestSuit) loadConfig() error {
	d.cfg = new(TestConfig)
	confFileName := "test.yaml"
	if os.Getenv("STORE_MODE") == "sqldb" {
		fmt.Printf("run store mode : sqldb\n")
		confFileName = "test_sqldb.yaml"
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

func (d *EurekaTestSuit) Destroy() {
	d.cancel()
	time.Sleep(5 * time.Second)

	d.storage.Destroy()
	time.Sleep(5 * time.Second)
}
