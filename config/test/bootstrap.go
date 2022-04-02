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

package test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/polarismesh/polaris-server/bootstrap/config"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
	config2 "github.com/polarismesh/polaris-server/config"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store"

	_ "github.com/go-sql-driver/mysql"

	_ "github.com/polarismesh/polaris-server/store/sqldb"
)

var (
	cfg           = new(config.Config)
	once          = new(sync.Once)
	configService = new(config2.Server)
	db            = new(sql.DB)
	cancelFlag    = false
	defaultCtx    = context.Background()
)

func init() {
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

		// 初始化存储层
		store.SetStoreConfig(&cfg.Store)
		_, _ = store.GetStore()

		// 初始化 DB 对象
		db, err = initDB()
		if err != nil {
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

		err = config2.InitConfigModule(ctx, cfg.Config)
		if err != nil {
			return
		}

		configService, err = config2.GetConfigServer()
		if err != nil {
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

func initDB() (*sql.DB, error) {
	masterEntry := cfg.Store.Option["master"]
	masterConfig, ok := masterEntry.(map[interface{}]interface{})
	if !ok {
		panic("database cfg is invalid")
	}

	dbType := masterConfig["dbType"].(string)
	dbUser := masterConfig["dbUser"].(string)
	dbPwd := masterConfig["dbPwd"].(string)
	dbAddr := masterConfig["dbAddr"].(string)
	dbName := masterConfig["dbName"].(string)

	dbSource := fmt.Sprintf("%s:%s@tcp(%s)/%s", dbUser, dbPwd, dbAddr, dbName)
	return sql.Open(dbType, dbSource)
}
