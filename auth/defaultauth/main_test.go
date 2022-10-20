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

package defaultauth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/boltdb/bolt"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v2"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	_ "github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	apiv2 "github.com/polarismesh/polaris/common/api/v2"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/plugin"
	_ "github.com/polarismesh/polaris/plugin/auth/defaultauth"
	_ "github.com/polarismesh/polaris/plugin/cmdb/memory"
	_ "github.com/polarismesh/polaris/plugin/discoverevent/local"
	_ "github.com/polarismesh/polaris/plugin/discoverstat/discoverlocal"
	_ "github.com/polarismesh/polaris/plugin/healthchecker/heartbeatmemory"
	_ "github.com/polarismesh/polaris/plugin/healthchecker/heartbeatredis"
	_ "github.com/polarismesh/polaris/plugin/history/logger"
	_ "github.com/polarismesh/polaris/plugin/password"
	_ "github.com/polarismesh/polaris/plugin/ratelimit/lrurate"
	_ "github.com/polarismesh/polaris/plugin/ratelimit/token"
	_ "github.com/polarismesh/polaris/plugin/statis/local"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store"
	"github.com/polarismesh/polaris/store/boltdb"
	_ "github.com/polarismesh/polaris/store/boltdb"
	"github.com/polarismesh/polaris/store/sqldb"
	_ "github.com/polarismesh/polaris/store/sqldb"
	"github.com/polarismesh/polaris/testdata"
)

const (
	tblUser     string = "user"
	tblStrategy string = "strategy"
	tblGroup    string = "group"
)

type Bootstrap struct {
	Logger map[string]*commonlog.Options
}

type TestConfig struct {
	Bootstrap    Bootstrap          `yaml:"bootstrap"`
	Cache        cache.Config       `yaml:"cache"`
	Namespace    namespace.Config   `yaml:"namespace"`
	HealthChecks healthcheck.Config `yaml:"healthcheck"`
	Store        store.Config       `yaml:"store"`
	Auth         auth.Config        `yaml:"auth"`
	Plugin       plugin.Config      `yaml:"plugin"`
}

type AuthTestSuit struct {
	cfg                 *TestConfig
	cancelFlag          bool
	updateCacheInterval time.Duration
	defaultCtx          context.Context
	cancel              context.CancelFunc
	storage             store.Store
	server              auth.AuthServer
}

// 加载配置
func (d *AuthTestSuit) loadConfig() error {

	d.cfg = new(TestConfig)

	confFileName := testdata.Path("auth_test.yaml")
	if os.Getenv("STORE_MODE") == "sqldb" {
		fmt.Printf("run store mode : sqldb\n")
		confFileName = testdata.Path("auth_test_sqldb.yaml")
		d.defaultCtx = context.WithValue(d.defaultCtx, utils.ContextAuthTokenKey,
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
func respSuccess(resp api.ResponseMessage) bool {

	ret := api.CalcCode(resp) == 200

	return ret
}

// 判断一个resp是否执行成功
func respSuccessV2(resp apiv2.ResponseMessage) bool {
	ret := apiv2.CalcCode(resp) == 200

	return ret
}

type options func(cfg *TestConfig)

// 内部初始化函数
func (d *AuthTestSuit) initialize(opts ...options) error {
	// 初始化defaultCtx
	d.defaultCtx = context.WithValue(context.Background(), utils.StringContext("request-id"), "test-1")
	d.defaultCtx = context.WithValue(d.defaultCtx, utils.ContextAuthTokenKey,
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
	commonlog.GetScopeOrDefaultByName(commonlog.ConfigLoggerName).SetOutputLevel(commonlog.ErrorLevel)
	commonlog.GetScopeOrDefaultByName(commonlog.StoreLoggerName).SetOutputLevel(commonlog.ErrorLevel)
	commonlog.GetScopeOrDefaultByName(commonlog.AuthLoggerName).SetOutputLevel(commonlog.ErrorLevel)

	metrics.InitMetrics()

	// 初始化存储层
	store.SetStoreConfig(&d.cfg.Store)
	s, _ := store.TestGetStore()
	d.storage = s

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

	d.server = authSvr

	// 多等待一会
	d.updateCacheInterval = cacheMgn.GetUpdateCacheInterval() + time.Millisecond*500

	time.Sleep(5 * time.Second)
	return nil
}

func (d *AuthTestSuit) Destroy() {
	d.cleanAllUser()
	d.cleanAllUserGroup()
	d.cleanAllAuthStrategy()

	d.cancel()
	time.Sleep(5 * time.Second)

	d.storage.Destroy()
	time.Sleep(5 * time.Second)
}

func (d *AuthTestSuit) cleanAllUser() {
	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

			if _, err := dbTx.Exec("delete from user where name like 'test%'"); err != nil {
				dbTx.Rollback()
				panic(err)
			}

			dbTx.Commit()
		}()
	} else if d.storage.Name() == boltdb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			defer dbTx.Rollback()

			if err := dbTx.DeleteBucket([]byte(tblUser)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					panic(err)
				}
			}

			dbTx.Commit()
		}()
	}
}

func (d *AuthTestSuit) cleanAllUserGroup() {
	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

			if _, err := dbTx.Exec("delete from user_group where name like 'test%'"); err != nil {
				dbTx.Rollback()
				panic(err)
			}
			if _, err := dbTx.Exec("delete from user_group_relation"); err != nil {
				dbTx.Rollback()
				panic(err)
			}

			dbTx.Commit()
		}()
	} else if d.storage.Name() == boltdb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			defer dbTx.Rollback()

			if err := dbTx.DeleteBucket([]byte(tblGroup)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					panic(err)
				}
			}

			dbTx.Commit()
		}()
	}
}

func (d *AuthTestSuit) cleanAllAuthStrategy() {
	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

			if _, err := dbTx.Exec("delete from auth_strategy where id != 'fbca9bfa04ae4ead86e1ecf5811e32a9'"); err != nil {
				dbTx.Rollback()
				panic(err)
			}
			if _, err := dbTx.Exec("delete from auth_principal where strategy_id != 'fbca9bfa04ae4ead86e1ecf5811e32a9'"); err != nil {
				dbTx.Rollback()
				panic(err)
			}
			if _, err := dbTx.Exec("delete from auth_strategy_resource where strategy_id != 'fbca9bfa04ae4ead86e1ecf5811e32a9'"); err != nil {
				dbTx.Rollback()
				panic(err)
			}

			dbTx.Commit()
		}()
	} else if d.storage.Name() == boltdb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			defer dbTx.Rollback()

			if err := dbTx.DeleteBucket([]byte(tblStrategy)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					panic(err)
				}
			}

			dbTx.Commit()
		}()
	}
}
