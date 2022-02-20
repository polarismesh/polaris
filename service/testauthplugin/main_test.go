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
	"github.com/polarismesh/polaris-server/bootstrap/config"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/service"
	"github.com/polarismesh/polaris-server/store"
	"gopkg.in/yaml.v2"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/polarismesh/polaris-server/plugin/auth/platform"
	_ "github.com/polarismesh/polaris-server/plugin/history/logger"
	_ "github.com/polarismesh/polaris-server/plugin/ratelimit/token"
	_ "github.com/polarismesh/polaris-server/store/sqldb"
)

var (
	cfg           = config.Config{}
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
	options := log.DefaultOptions()
	log.Configure(options)
	var err error
	once.Do(func() {
		err = loadConfigWithAuthPlugin()
		if err != nil {
			return
		}
		// 初始化ctx
		defaultCtx = context.WithValue(defaultCtx, utils.StringContext("request-id"), "test-auth-plugin")

		// 初始化存储层
		store.SetStoreConfig(&cfg.Store)
		store.GetStore()

		//  初始化插件
		plugin.SetPluginConfig(&cfg.Plugin)

		// 初始化naming server
		ctx := context.Background()

		if err := service.Initialize(ctx, &cfg.Naming, &cfg.Cache, nil); err != nil {
			panic(err)
		}

		server, err = service.GetServer()
		if err != nil {
			panic(err)
		}

		entry := cfg.Store.Option["master"]
		config, ok := entry.(map[interface{}]interface{})
		if !ok {
			panic("database cfg is invalid")
		}

		dbType := config["dbType"].(string)
		dbUser := config["dbUser"].(string)
		dbPwd := config["dbPwd"].(string)
		dbAddr := config["dbAddr"].(string)
		dbName := config["dbName"].(string)

		dbSource := fmt.Sprintf("%s:%s@tcp(%s)/%s", dbUser, dbPwd, dbAddr, dbName)
		db, err = sql.Open(dbType, dbSource)
		if err != nil {
			panic(err)
		}

		// 创建平台
		resp := createPlatform()
		platformToken = resp.GetToken().GetValue()

		// 初始化ctx
		defaultCtx = context.WithValue(defaultCtx, utils.StringContext("request-id"), "test-request-id")

		// 等待数据加载到缓存
		time.Sleep(time.Second * 2)
	})
	return err
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
	if id == "" {
		panic("id is empty")
	}

	log.Infof("clean platform: %s", id)
	str := `delete from platform where id = ?`
	if _, err := db.Exec(str, id); err != nil {
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
