//go:build integrationauth
// +build integrationauth

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

package testauthplugin

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v2"

	"github.com/polarismesh/polaris-server/auth"
	_ "github.com/polarismesh/polaris-server/auth/defaultauth"
	"github.com/polarismesh/polaris-server/bootstrap/config"
	_ "github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/namespace"
	"github.com/polarismesh/polaris-server/plugin"
	_ "github.com/polarismesh/polaris-server/plugin/auth/defaultauth"
	_ "github.com/polarismesh/polaris-server/plugin/healthchecker/heartbeatmemory"
	_ "github.com/polarismesh/polaris-server/plugin/history/logger"
	_ "github.com/polarismesh/polaris-server/plugin/ratelimit/token"
	"github.com/polarismesh/polaris-server/service"
	"github.com/polarismesh/polaris-server/service/batch"
	"github.com/polarismesh/polaris-server/service/healthcheck"
	"github.com/polarismesh/polaris-server/store"
	_ "github.com/polarismesh/polaris-server/store/boltdb"
	_ "github.com/polarismesh/polaris-server/store/sqldb"

	_ "github.com/polarismesh/polaris-server/plugin/auth/defaultauth"
	_ "github.com/polarismesh/polaris-server/plugin/cmdb/memory"
)

var (
	cfg           = &config.Config{}
	once          = sync.Once{}
	server        = &service.Server{}
	db            = &sql.DB{}
	defaultCtx    = context.Background()
	platformToken = ""
)

const (
	platformID = "test-platform-id"
)

/**
 * @brief 内部初始化函数
 */
func initialize() error {
	once.Do(func() {

		_cfg, err := config.Load("./test.yaml")
		if err != nil {
			panic(err)
		}
		cfg = _cfg

		options := log.DefaultOptions()
		log.Configure(options)
		err = loadConfigWithAuthPlugin()
		if err != nil {
			return
		}
		// 初始化ctx
		defaultCtx = context.WithValue(defaultCtx, utils.StringContext("request-id"), "test-auth-plugin")

		// 初始化存储层
		store.SetStoreConfig(&cfg.Store)
		s, err := store.GetStore()
		if err != nil {
			panic(err)
		}

		//  初始化插件
		plugin.SetPluginConfig(&cfg.Plugin)

		// 初始化naming server
		ctx := context.Background()

		bcCfg, err := batch.ParseBatchConfig(map[string]interface{}{})
		if err != nil {
			panic(err)
		}

		bc, err := batch.NewBatchCtrlWithConfig(s, nil, bcCfg)
		if err != nil {
			log.Errorf("new batch ctrl with config err: %s", err.Error())
			panic(err)
		}
		bc.Start(ctx)

		if err := auth.Initialize(ctx, &cfg.Auth, s, nil); err != nil {
			panic(err)
		}

		if err := namespace.Initialize(ctx, &cfg.Namespace, s, nil); err != nil {
			panic(err)
		}

		if err := healthcheck.Initialize(ctx, &cfg.HealthChecks, true, nil); err != nil {
			panic(err)
		}

		if err := service.Initialize(ctx, &cfg.Naming, &cfg.Cache, nil); err != nil {
			panic(err)
		}

		svr, err := service.GetOriginServer()
		if err != nil {
			panic(err)
		}
		server = svr

		// 创建平台
		resp := createPlatform()
		platformToken = resp.GetToken().GetValue()

		// 初始化ctx
		defaultCtx = context.WithValue(defaultCtx, utils.StringContext("request-id"), "test-request-id")

		// 等待数据加载到缓存
		time.Sleep(time.Second * 2)
	})
	return nil
}

/**
 * @brief 加载配置
 */
func loadConfigWithAuthPlugin() error {
	file, err := os.Open("test.yaml")
	if err != nil {
		fmt.Printf("[ERROR] open file err: %s\n", err.Error())
		return err
	}

	if err := yaml.NewDecoder(file).Decode(&cfg); err != nil {
		fmt.Printf("[ERROR] decode err: %s\n", err.Error())
		return err
	}
	return err
}

/**
 * @brief 判断请求是否成功
 */
func respSuccess(resp api.ResponseMessage) bool {

	return api.CalcCode(resp) == 200
}

/**
 * @brief 创建一个平台
 */
func createPlatform() *api.Platform {
	platform := &api.Platform{
		Id:         utils.NewStringValue(platformID),
		Name:       utils.NewStringValue("test-platform-name"),
		Domain:     utils.NewStringValue("test-platform-domain"),
		Qps:        utils.NewUInt32Value(1),
		Owner:      utils.NewStringValue("test-platform-owner"),
		Department: utils.NewStringValue("test-platform-department"),
		Comment:    utils.NewStringValue("test-platform-comment"),
	}

	cleanPlatform(platformID)

	resp := server.CreatePlatform(defaultCtx, platform)
	if !respSuccess(resp) {
		panic(resp.GetInfo().GetValue())
	}

	return resp.GetPlatform()
}

/**
 * @brief 从数据库中彻底删除平台
 */
func cleanPlatform(id string) {
	s, _ := store.GetStore()
	if id == "" {
		panic("id is empty")
	}

	log.Infof("clean platform: %s", id)
	if err := s.DeletePlatform(id); err != nil {
		panic(err)
	}
}

/**
 * @brief 初始化函数
 */
func init() {
	if err := initialize(); err != nil {
		fmt.Printf("[test_auth_plugin] init err: %s", err.Error())
		panic(err)
	}
}

func Test_Nothing(m *testing.T) {

}
