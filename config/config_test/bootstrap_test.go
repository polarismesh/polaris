/*
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

package config_test

import (
	"context"
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/bootstrap/config"
	"github.com/polarismesh/polaris-server/cache"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
	config_center "github.com/polarismesh/polaris-server/config"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/polarismesh/polaris-server/cache"
	_ "github.com/polarismesh/polaris-server/store/boltdb"
	_ "github.com/polarismesh/polaris-server/store/sqldb"

	_ "github.com/polarismesh/polaris-server/auth/defaultauth"
	_ "github.com/polarismesh/polaris-server/plugin/auth/defaultauth"
)

var (
	cfg           = new(config.Config)
	once          = new(sync.Once)
	configService config_center.ConfigCenterServer
	originServer  = new(config_center.Server)
	cancelFlag    = false
	defaultCtx    = context.Background()
)

func init() {
	if err := os.RemoveAll("./config_center_test.bolt"); err != nil {
		panic(err)
	}
	if err := doInitialize(); err != nil {
		fmt.Printf("bootstrap config test module error. %s", err.Error())
		panic(err)
	}
}

func doInitialize() error {
	logOptions := log.DefaultOptions()
	_ = log.Configure(logOptions)

	var err error

	once.Do(func() {
		// 加载启动配置文件
		err = loadBootstrapConfig()
		if err != nil {
			return
		}

		// 初始化defaultCtx
		defaultCtx = context.WithValue(defaultCtx, utils.StringContext("request-id"), "config-test-request-id")
		defaultCtx = context.WithValue(defaultCtx, utils.ContextUserNameKey, "polaris")
		defaultCtx = context.WithValue(defaultCtx, utils.ContextAuthTokenKey, "4azbewS+pdXvrMG1PtYV3SrcLxjmYd0IVNaX9oYziQygRnKzjcSbxl+Reg7zYQC1gRrGiLzmMY+w+aCxOYI=")

		// 初始化日志打印
		err = log.Configure(cfg.Bootstrap.Logger)
		if err != nil {
			fmt.Printf("[ERROR] configure logger fail: %v\n", err)
			return
		}

		plugin.SetPluginConfig(&cfg.Plugin)

		// 初始化存储层
		store.SetStoreConfig(&cfg.Store)
		s, err := store.GetStore()
		if err != nil {
			fmt.Printf("[ERROR] configure get store fail: %v\n", err)
			return
		}

		if err := cache.Initialize(context.Background(), &cfg.Cache, s); err != nil {
			fmt.Printf("[ERROR] configure init cache fail: %v\n", err)
			return
		}

		cacheMgr, err := cache.GetCacheManager()
		if err != nil {
			fmt.Printf("[ERROR] configure get cache fail: %v\n", err)
			return
		}

		if err := auth.Initialize(context.Background(), &cfg.Auth, s, cacheMgr); err != nil {
			fmt.Printf("[ERROR] configure init auth fail: %v\n", err)
			return
		}

		authSvr, err := auth.GetAuthServer()
		if err != nil {
			fmt.Printf("[ERROR] configure get auth fail: %v\n", err)
			return
		}

		plugin.SetPluginConfig(&cfg.Plugin)

		// 初始化配置中心模块
		ctx, cancel := context.WithCancel(context.Background())
		defer func() {
			if cancelFlag {
				cancel()
			}
		}()

		err = config_center.Initialize(ctx, cfg.Config, s, cacheMgr, authSvr)
		if err != nil {
			return
		}

		configService, err = config_center.GetServer()
		if err != nil {
			return
		}
		originServer, err = config_center.GetOriginServer()
		if err != nil {
			return
		}

		if err := cacheMgr.Start(context.Background()); err != nil {
			return
		}
	})
	return err
}

func loadBootstrapConfig() error {
	file, err := os.Open("test.yaml")
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return err
	}

	err = yaml.NewDecoder(file).Decode(&cfg)
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return err
	}

	return err
}
