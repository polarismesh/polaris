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
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	bolt "go.etcd.io/bbolt"
	"gopkg.in/yaml.v2"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/log"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
	ns "github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/batch"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store"
	"github.com/polarismesh/polaris/store/boltdb"
	sqldb "github.com/polarismesh/polaris/store/mysql"
	testdata "github.com/polarismesh/polaris/test/data"
)

func init() {
	go func() {
		http.ListenAndServe("0.0.0.0:16060", nil)
	}()
}

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

var (
	testNamespace = "testNamespace123qwe"
	testGroup     = "testGroup"
	testFile      = "testFile"
	testContent   = "testContent"
	operator      = "polaris"
	size          = 7
)

const (
	templateName1 = "t1"
	templateName2 = "t2"
)

type Bootstrap struct {
	Logger map[string]*commonlog.Options
}

type TestConfig struct {
	Bootstrap           Bootstrap      `yaml:"bootstrap"`
	Cache               cache.Config   `yaml:"cache"`
	Namespace           ns.Config      `yaml:"namespace"`
	Naming              service.Config `yaml:"naming"`
	DisableConfig       bool
	Config              config.Config      `yaml:"config"`
	HealthChecks        healthcheck.Config `yaml:"healthcheck"`
	Store               store.Config       `yaml:"store"`
	DisableAuth         bool
	Auth                auth.Config   `yaml:"auth"`
	Plugin              plugin.Config `yaml:"plugin"`
	ReplaceStore        store.Store
	ServiceCacheEntries []cachetypes.ConfigEntry
}

var InjectTestDataClean func() TestDataClean

func SetTestDataClean(callback func() TestDataClean) {
	InjectTestDataClean = callback
}

type DiscoverTestSuit struct {
	cfg                 *TestConfig
	configServer        config.ConfigCenterServer
	configOriginSvr     config.ConfigCenterServer
	server              service.DiscoverServer
	originSvr           service.DiscoverServer
	healthCheckServer   *healthcheck.Server
	cacheMgr            *cache.CacheManager
	userMgn             auth.UserServer
	strategyMgn         auth.StrategyServer
	namespaceSvr        ns.NamespaceOperateServer
	cancelFlag          bool
	updateCacheInterval time.Duration
	DefaultCtx          context.Context
	cancel              context.CancelFunc
	Storage             store.Store
	bc                  *batch.Controller
	cleanDataOp         TestDataClean
	caller              func() store.Store
}

func (d *DiscoverTestSuit) InjectSuit(*DiscoverTestSuit) {

}

func (d *DiscoverTestSuit) GetBootstrapConfig() *TestConfig {
	return d.cfg
}

func (d *DiscoverTestSuit) CacheMgr() *cache.CacheManager {
	return d.cacheMgr
}

func (d *DiscoverTestSuit) GetTestDataClean() TestDataClean {
	return d.cleanDataOp
}

func (d *DiscoverTestSuit) DiscoverServer() service.DiscoverServer {
	return d.server
}

func (d *DiscoverTestSuit) OriginDiscoverServer() service.DiscoverServer {
	return d.originSvr
}

func (d *DiscoverTestSuit) ConfigServer() config.ConfigCenterServer {
	return d.configServer
}

func (d *DiscoverTestSuit) OriginConfigServer() *config.Server {
	return d.configOriginSvr.(*config.Server)
}

func (d *DiscoverTestSuit) HealthCheckServer() *healthcheck.Server {
	return d.healthCheckServer
}

func (d *DiscoverTestSuit) NamespaceServer() ns.NamespaceOperateServer {
	return d.namespaceSvr
}

func (d *DiscoverTestSuit) UserServer() auth.UserServer {
	return d.userMgn
}

func (d *DiscoverTestSuit) StrategyServer() auth.StrategyServer {
	return d.strategyMgn
}

func (d *DiscoverTestSuit) UpdateCacheInterval() time.Duration {
	return d.updateCacheInterval
}

func (d *DiscoverTestSuit) BatchController() *batch.Controller {
	return d.bc
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
	} else {
		fmt.Printf("run store mode : boltdb\n")
	}
	// 如果有额外定制的配置文件，优先采用
	if val := os.Getenv("POLARIS_TEST_BOOTSTRAP_FILE"); val != "" {
		confFileName = val
	}
	buf, err := os.ReadFile(confFileName)
	if nil != err {
		return fmt.Errorf("read file %s error", confFileName)
	}

	if err = parseYamlContent(string(buf), d.cfg); err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return err
	}
	d.cfg.Naming.Interceptors = service.GetChainOrder()
	d.cfg.Config.Interceptors = config.GetChainOrder()
	return err
}

func parseYamlContent(content string, conf *TestConfig) error {
	if err := yaml.Unmarshal([]byte(replaceEnv(content)), conf); nil != err {
		return fmt.Errorf("parse yaml %s error:%w", content, err)
	}
	return nil
}

// replaceEnv replace holder by env list
func replaceEnv(configContent string) string {
	return os.ExpandEnv(configContent)
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

func (d *DiscoverTestSuit) ReplaceStore(caller func() store.Store) {
	d.caller = caller
}

// 内部初始化函数
func (d *DiscoverTestSuit) initialize(opts ...options) error {
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

	d.cleanDataOp = d
	if InjectTestDataClean != nil {
		d.cleanDataOp = InjectTestDataClean()
	}
	// 注入测试套件相关数据信息
	d.cleanDataOp.InjectSuit(d)

	_ = commonlog.Configure(d.cfg.Bootstrap.Logger)

	metrics.InitMetrics()
	eventhub.InitEventHub()

	// 初始化存储层
	if d.caller != nil {
		d.Storage = d.caller()
	} else {
		store.SetStoreConfig(&d.cfg.Store)
		s, _ := store.TestGetStore()
		d.Storage = s
	}

	plugin.SetPluginConfig(&d.cfg.Plugin)

	ctx, cancel := context.WithCancel(context.Background())

	d.cancel = cancel

	// 初始化缓存模块
	cacheMgn, err := cache.TestCacheInitialize(ctx, &d.cfg.Cache, d.Storage)
	if err != nil {
		panic(err)
	}
	d.cacheMgr = cacheMgn
	_ = d.cacheMgr.OpenResourceCache(cachetypes.ConfigEntry{
		Name: cachetypes.GrayName,
	})

	if !d.cfg.DisableAuth {
		// 初始化鉴权层
		userMgn, strategyMgn, err := auth.TestInitialize(ctx, &d.cfg.Auth, d.Storage, cacheMgn)
		if err != nil {
			panic(err)
		}
		d.userMgn = userMgn
		d.strategyMgn = strategyMgn
	}

	// 初始化命名空间模块
	namespaceSvr, err := ns.TestInitialize(ctx, &d.cfg.Namespace, d.Storage, cacheMgn, d.userMgn, d.strategyMgn)
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

	bc, err := batch.NewBatchCtrlWithConfig(d.Storage, cacheMgn, batchConfig)
	if err != nil {
		log.Errorf("new batch ctrl with config err: %s", err.Error())
		panic(err)
	}
	bc.Start(ctx)
	d.bc = bc

	if len(d.cfg.HealthChecks.LocalHost) == 0 {
		d.cfg.HealthChecks.LocalHost = utils.LocalHost // 补充healthCheck的配置
	}
	healthCheckServer, err := healthcheck.TestInitialize(ctx, &d.cfg.HealthChecks, bc, d.Storage)
	if err != nil {
		panic(err)
	}
	healthcheck.SetServer(healthCheckServer)
	d.healthCheckServer = healthCheckServer
	healthCheckServer.SetServiceCache(cacheMgn.Service())
	healthCheckServer.SetInstanceCache(cacheMgn.Instance())

	val, originVal, err := service.TestInitialize(ctx, &d.cfg.Naming, &d.cfg.Cache, d.cfg.ServiceCacheEntries,
		bc, cacheMgn, d.Storage, namespaceSvr, healthCheckServer, d.userMgn, d.strategyMgn)
	if err != nil {
		panic(err)
	}
	d.server = val
	d.originSvr = originVal

	if !d.cfg.DisableConfig {
		confVal, confOriginVal, err := config.TestInitialize(ctx, d.cfg.Config, d.Storage, cacheMgn, namespaceSvr, d.userMgn, d.strategyMgn)
		if err != nil {
			panic(err)
		}
		d.configServer = confVal
		d.configOriginSvr = confOriginVal
	}

	// 多等待一会
	d.updateCacheInterval = d.cacheMgr.GetUpdateCacheInterval() + time.Millisecond*500
	if err := cache.TestRun(ctx, d.cacheMgr); err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	return nil
}

func (d *DiscoverTestSuit) Destroy() {
	d.cancel()
	if svr, ok := d.configOriginSvr.(*config.Server); ok {
		svr.WatchCenter().Close()
	}
	d.healthCheckServer.Destroy()
	_ = d.cacheMgr.Close()
	_ = d.Storage.Destroy()
}

func (d *DiscoverTestSuit) CleanReportClient() {
	if d.Storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)
			defer rollbackDbTx(dbTx)

			if _, err := dbTx.Exec("delete from client"); err != nil {
				panic(err)
			}
			if _, err := dbTx.Exec("delete from client_stat"); err != nil {
				panic(err)
			}

			commitDbTx(dbTx)
		}()
	} else if d.Storage.Name() == boltdb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			defer rollbackBoltTx(dbTx)

			if err := dbTx.DeleteBucket([]byte(tblClient)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					rollbackBoltTx(dbTx)
					panic(err)
				}
			}

			commitBoltTx(dbTx)
		}()
	}
}

func rollbackDbTx(dbTx *sqldb.BaseTx) {
	if err := dbTx.Rollback(); err != nil {
		log.Errorf("fail to rollback db tx, err %v", err)
	}
}

func commitDbTx(dbTx *sqldb.BaseTx) {
	if err := dbTx.Commit(); err != nil {
		log.Errorf("fail to commit db tx, err %v", err)
	}
}

func rollbackBoltTx(tx *bolt.Tx) {
	if err := tx.Rollback(); err != nil {
		log.Errorf("fail to rollback bolt tx, err %v", err)
	}
}

func commitBoltTx(tx *bolt.Tx) {
	if err := tx.Commit(); err != nil {
		log.Errorf("fail to commit bolt tx, err %v", err)
	}
}

// 从数据库彻底删除命名空间
func (d *DiscoverTestSuit) CleanNamespace(name string) {
	if name == "" {
		panic("name is empty")
	}

	log.Infof("clean namespace: %s", name)

	if d.Storage.Name() == sqldb.STORENAME {
		str := "delete from namespace where name = ?"
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)
			defer rollbackDbTx(dbTx)

			if _, err := dbTx.Exec(str, name); err != nil {
				panic(err)
			}

			commitDbTx(dbTx)
		}()
	} else if d.Storage.Name() == boltdb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			if err := dbTx.Bucket([]byte(tblNameNamespace)).DeleteBucket([]byte(name)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					rollbackBoltTx(dbTx)
					panic(err)
				}
			}

			commitBoltTx(dbTx)
		}()
	}
}

// 从数据库彻底删除全部服务
func (d *DiscoverTestSuit) CleanAllService() {

	if d.Storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer rollbackDbTx(dbTx)

			if _, err := dbTx.Exec("delete from service_metadata"); err != nil {
				rollbackDbTx(dbTx)
				panic(err)
			}

			if _, err := dbTx.Exec("delete from service"); err != nil {
				rollbackDbTx(dbTx)
				panic(err)
			}

			if _, err := dbTx.Exec("delete from owner_service_map"); err != nil {
				rollbackDbTx(dbTx)
				panic(err)
			}

			commitDbTx(dbTx)
		}()
	} else if d.Storage.Name() == boltdb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			defer rollbackBoltTx(dbTx)

			if err := dbTx.DeleteBucket([]byte(tblNameService)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					rollbackBoltTx(dbTx)
					panic(err)
				}
			}

			commitBoltTx(dbTx)
		}()
	}
}

// 从数据库彻底删除服务
func (d *DiscoverTestSuit) CleanService(name, namespace string) {

	if d.Storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer rollbackDbTx(dbTx)

			str := "select id from service where name = ? and namespace = ?"
			var id string
			err = dbTx.QueryRow(str, name, namespace).Scan(&id)
			switch {
			case err == sql.ErrNoRows:
				return
			case err != nil:
				panic(err)
			}

			if _, err := dbTx.Exec("delete from service_metadata where id = ?", id); err != nil {
				rollbackDbTx(dbTx)
				panic(err)
			}

			if _, err := dbTx.Exec("delete from service where id = ?", id); err != nil {
				rollbackDbTx(dbTx)
				panic(err)
			}

			if _, err := dbTx.Exec(
				"delete from owner_service_map where service=? and namespace=?", name, namespace); err != nil {
				rollbackDbTx(dbTx)
				panic(err)
			}

			commitDbTx(dbTx)
		}()
	} else if d.Storage.Name() == boltdb.STORENAME {
		func() {
			svc, err := d.Storage.GetService(name, namespace)
			if err != nil {
				panic(err)
			}
			if svc == nil {
				return
			}

			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			defer rollbackBoltTx(dbTx)

			if err := dbTx.Bucket([]byte(tblNameService)).DeleteBucket([]byte(svc.ID)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					rollbackBoltTx(dbTx)
					panic(err)
				}
			}

			commitBoltTx(dbTx)
		}()
	}
}

// clean services
func (d *DiscoverTestSuit) CleanServices(services []*apiservice.Service) {

	if d.Storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer rollbackDbTx(dbTx)

			str := "delete from service where name = ? and namespace = ?"
			cleanOwnerSql := "delete from owner_service_map where service=? and namespace=?"
			for _, service := range services {
				if _, err := dbTx.Exec(
					str, service.GetName().GetValue(), service.GetNamespace().GetValue()); err != nil {
					panic(err)
				}
				if _, err := dbTx.Exec(
					cleanOwnerSql, service.GetName().GetValue(), service.GetNamespace().GetValue()); err != nil {
					panic(err)
				}
			}

			commitDbTx(dbTx)
		}()
	} else if d.Storage.Name() == boltdb.STORENAME {
		func() {
			ids := make([]string, 0, len(services))

			for _, service := range services {
				svc, err := d.Storage.GetService(service.GetName().GetValue(), service.GetNamespace().GetValue())
				if err != nil {
					panic(err)
				}

				ids = append(ids, svc.ID)
			}

			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)

			for i := range ids {
				if err := dbTx.Bucket([]byte(tblNameService)).DeleteBucket([]byte(ids[i])); err != nil {
					if !errors.Is(err, bolt.ErrBucketNotFound) {
						rollbackBoltTx(dbTx)
						panic(err)
					}
				}
			}
			commitBoltTx(dbTx)
		}()
	}

}

// 从数据库彻底删除实例
func (d *DiscoverTestSuit) CleanInstance(instanceID string) {
	if instanceID == "" {
		panic("instanceID is empty")
	}
	log.Infof("clean instance: %s", instanceID)

	if d.Storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer rollbackDbTx(dbTx)

			str := "delete from instance where id = ?"
			if _, err := dbTx.Exec(str, instanceID); err != nil {
				rollbackDbTx(dbTx)
				panic(err)
			}

			commitDbTx(dbTx)
		}()
	} else if d.Storage.Name() == boltdb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)

			if err := dbTx.Bucket([]byte(tblNameInstance)).DeleteBucket([]byte(instanceID)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					rollbackBoltTx(dbTx)
					panic(err)
				}
			}
			commitBoltTx(dbTx)
		}()
	}
}

// 彻底删除一个路由配置
func (d *DiscoverTestSuit) CleanCommonRoutingConfig(service string, namespace string) {

	if d.Storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer rollbackDbTx(dbTx)

			str := "delete from routing_config where id in (select id from service where name = ? and namespace = ?)"
			// fmt.Printf("%s %s %s\n", str, service, namespace)
			if _, err := dbTx.Exec(str, service, namespace); err != nil {
				panic(err)
			}
			str = "delete from routing_config_v2"
			// fmt.Printf("%s %s %s\n", str, service, namespace)
			if _, err := dbTx.Exec(str); err != nil {
				panic(err)
			}

			commitDbTx(dbTx)
		}()
	} else if d.Storage.Name() == boltdb.STORENAME {
		func() {
			svc, err := d.Storage.GetService(service, namespace)
			if err != nil {
				panic(err)
			}

			if svc == nil {
				return
			}

			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			defer rollbackBoltTx(dbTx)

			v1Bucket := dbTx.Bucket([]byte(tblNameRouting))
			if v1Bucket != nil {
				if err := v1Bucket.DeleteBucket([]byte(svc.ID)); err != nil {
					if !errors.Is(err, bolt.ErrBucketNotFound) {
						rollbackBoltTx(dbTx)
						panic(err)
					}
				}
			}

			if err := dbTx.DeleteBucket([]byte(tblNameRoutingV2)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					rollbackBoltTx(dbTx)
					panic(err)
				}
			}
			commitBoltTx(dbTx)
		}()
	}
}

func (d *DiscoverTestSuit) TruncateCommonRoutingConfigV2() {
	if d.Storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)
			defer rollbackDbTx(dbTx)

			str := "delete from routing_config_v2"
			if _, err := dbTx.Exec(str); err != nil {
				panic(err)
			}

			commitDbTx(dbTx)
		}()
	} else if d.Storage.Name() == boltdb.STORENAME {
		func() {

			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			defer rollbackBoltTx(dbTx)

			if err := dbTx.DeleteBucket([]byte(tblNameRoutingV2)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					rollbackBoltTx(dbTx)
					panic(err)
				}
			}

			commitBoltTx(dbTx)
		}()
	}
}

// 彻底删除一个路由配置
func (d *DiscoverTestSuit) CleanCommonRoutingConfigV2(rules []*apitraffic.RouteRule) {

	if d.Storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)
			defer rollbackDbTx(dbTx)

			str := "delete from routing_config_v2 where id in (%s)"

			places := []string{}
			args := []interface{}{}
			for i := range rules {
				places = append(places, "?")
				args = append(args, rules[i].Id)
			}

			str = fmt.Sprintf(str, strings.Join(places, ","))
			// fmt.Printf("%s %s %s\n", str, service, namespace)
			if _, err := dbTx.Exec(str, args...); err != nil {
				panic(err)
			}

			commitDbTx(dbTx)
		}()
	} else if d.Storage.Name() == boltdb.STORENAME {
		func() {

			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			defer rollbackBoltTx(dbTx)

			for i := range rules {
				if err := dbTx.Bucket([]byte(tblNameRoutingV2)).DeleteBucket([]byte(rules[i].Id)); err != nil {
					if !errors.Is(err, bolt.ErrBucketNotFound) {
						rollbackBoltTx(dbTx)
						panic(err)
					}
				}
			}

			commitBoltTx(dbTx)
		}()
	}
}

// 彻底删除限流规则
func (d *DiscoverTestSuit) CleanRateLimit(id string) {

	if d.Storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer rollbackDbTx(dbTx)

			str := `delete from ratelimit_config where id = ?`
			if _, err := dbTx.Exec(str, id); err != nil {
				panic(err)
			}

			commitDbTx(dbTx)
		}()
	} else if d.Storage.Name() == boltdb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)

			if err := dbTx.Bucket([]byte(tblRateLimitConfig)).DeleteBucket([]byte(id)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					rollbackBoltTx(dbTx)
					panic(err)
				}
			}
			commitBoltTx(dbTx)
		}()
	}
}

func buildCircuitBreakerKey(id, version string) string {
	return fmt.Sprintf("%s_%s", id, version)
}

// 彻底删除熔断规则
func (d *DiscoverTestSuit) CleanCircuitBreaker(id, version string) {
	log.Infof("clean circuit breaker, id: %s, version: %s", id, version)

	if d.Storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer rollbackDbTx(dbTx)

			str := `delete from circuitbreaker_rule where id = ? and version = ?`
			if _, err := dbTx.Exec(str, id, version); err != nil {
				panic(err)
			}

			commitDbTx(dbTx)
		}()
	} else if d.Storage.Name() == boltdb.STORENAME {
		func() {
			tx, err := d.Storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)

			if err := dbTx.Bucket(
				[]byte(tblCircuitBreaker)).DeleteBucket([]byte(buildCircuitBreakerKey(id, version))); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					panic(err)
				}
			}

			commitBoltTx(dbTx)
		}()
	}
}

// 彻底删除熔断规则发布记录
func (d *DiscoverTestSuit) CleanCircuitBreakerRelation(name, namespace, ruleID, ruleVersion string) {
}

// 彻底删除熔断规则发布记录
func (d *DiscoverTestSuit) CleanServiceContract() error {
	if d.Storage.Name() == boltdb.STORENAME {
		proxyTx, err := d.Storage.StartTx()
		if err != nil {
			return err
		}

		tx := proxyTx.GetDelegateTx().(*bolt.Tx)

		bucketName := []string{
			"service_contract",
		}

		defer tx.Rollback()

		for i := range bucketName {
			if err := tx.DeleteBucket([]byte(bucketName[i])); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					return err
				}
			}
		}
		return tx.Commit()
	}
	if d.Storage.Name() == sqldb.STORENAME {
		proxyTx, err := d.Storage.StartTx()
		if err != nil {
			return err
		}

		tx := proxyTx.GetDelegateTx().(*sqldb.BaseTx)

		defer tx.Rollback()
		_, err = tx.Exec("delete from service_contract")
		if err != nil {
			return err
		}
		_, err = tx.Exec("delete from service_contract_detail")
		if err != nil {
			return err
		}
		return tx.Commit()
	}
	return nil
}

func (d *DiscoverTestSuit) ClearTestDataWhenUseRDS() error {
	if d.Storage.Name() == boltdb.STORENAME {
		proxyTx, err := d.Storage.StartTx()
		if err != nil {
			return err
		}

		tx := proxyTx.GetDelegateTx().(*bolt.Tx)

		bucketName := []string{
			"ConfigFileGroup",
			"ConfigFileGroupID",
			"ConfigFile",
			"ConfigFileID",
			"ConfigFileReleaseHistory",
			"ConfigFileReleaseHistoryID",
			"ConfigFileRelease",
			"ConfigFileReleaseID",
			"ConfigFileTag",
			"ConfigFileTagID",
			"namespace",
			"service_contract",
		}

		defer tx.Rollback()

		for i := range bucketName {
			if err := tx.DeleteBucket([]byte(bucketName[i])); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					return err
				}
			}
		}
		return tx.Commit()
	}
	if d.Storage.Name() == sqldb.STORENAME {
		proxyTx, err := d.Storage.StartTx()
		if err != nil {
			return err
		}

		tx := proxyTx.GetDelegateTx().(*sqldb.BaseTx)

		defer tx.Rollback()

		_, err = tx.Exec("delete from config_file_group where namespace = ? ", testNamespace)
		if err != nil {
			return err
		}
		_, err = tx.Exec("delete from config_file where namespace = ? ", testNamespace)
		if err != nil {
			return err
		}
		_, err = tx.Exec("delete from config_file_release where namespace = ? ", testNamespace)
		if err != nil {
			return err
		}
		_, err = tx.Exec("delete from config_file_release_history where namespace = ? ", testNamespace)
		if err != nil {
			return err
		}
		_, err = tx.Exec("delete from config_file_tag where namespace = ? ", testNamespace)
		if err != nil {
			return err
		}
		_, err = tx.Exec("delete from namespace where name = ? ", testNamespace)
		if err != nil {
			return err
		}
		_, err = tx.Exec("delete from config_file_template where name in (?,?) ", templateName1, templateName2)
		if err != nil {
			return err
		}
		_, err = tx.Exec("delete from service_contract")
		if err != nil {
			return err
		}
		_, err = tx.Exec("delete from service_contract_detail")
		if err != nil {
			return err
		}
		return tx.Commit()
	}
	return nil
}
