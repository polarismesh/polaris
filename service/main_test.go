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

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/boltdb/bolt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"gopkg.in/yaml.v2"

	"github.com/polarismesh/polaris/auth"
	_ "github.com/polarismesh/polaris/auth/defaultauth"
	"github.com/polarismesh/polaris/cache"
	_ "github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/eventhub"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/plugin"
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
	"github.com/polarismesh/polaris/service/batch"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store"
	"github.com/polarismesh/polaris/store/boltdb"
	_ "github.com/polarismesh/polaris/store/boltdb"
	sqldb "github.com/polarismesh/polaris/store/mysql"
	"github.com/polarismesh/polaris/testdata"
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
	Namespace    namespace.Config   `yaml:"namespace"`
	Naming       Config             `yaml:"naming"`
	Config       Config             `yaml:"config"`
	HealthChecks healthcheck.Config `yaml:"healthcheck"`
	Store        store.Config       `yaml:"store"`
	Auth         auth.Config        `yaml:"auth"`
	Plugin       plugin.Config      `yaml:"plugin"`
}

type DiscoverTestSuit struct {
	cfg                 *TestConfig
	server              DiscoverServer
	healthCheckServer   *healthcheck.Server
	namespaceSvr        namespace.NamespaceOperateServer
	cancelFlag          bool
	updateCacheInterval time.Duration
	defaultCtx          context.Context
	cancel              context.CancelFunc
	storage             store.Store
}

// 加载配置
func (d *DiscoverTestSuit) loadConfig() error {

	d.cfg = new(TestConfig)

	confFileName := testdata.Path("service_test.yaml")
	if os.Getenv("STORE_MODE") == "sqldb" {
		fmt.Printf("run store mode : sqldb\n")
		confFileName = testdata.Path("service_test_sqldb.yaml")
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

func respNotFound(resp api.ResponseMessage) bool {
	res := apimodel.Code(resp.GetCode().GetValue()) == apimodel.Code_NotFoundResource

	return res
}

type options func(cfg *TestConfig)

// 内部初始化函数
func (d *DiscoverTestSuit) initialize(opts ...options) error {
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
	commonlog.GetScopeOrDefaultByName(commonlog.CacheLoggerName).SetOutputLevel(commonlog.ErrorLevel)
	commonlog.GetScopeOrDefaultByName(commonlog.StoreLoggerName).SetOutputLevel(commonlog.ErrorLevel)
	commonlog.GetScopeOrDefaultByName(commonlog.AuthLoggerName).SetOutputLevel(commonlog.ErrorLevel)

	metrics.TestInitMetrics()
	eventhub.InitEventHub()

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

	// 初始化命名空间模块
	namespaceSvr, err := namespace.TestInitialize(ctx, &d.cfg.Namespace, s, cacheMgn, authSvr)
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
	healthCheckServer, err := healthcheck.TestInitialize(ctx, &d.cfg.HealthChecks, d.cfg.Cache.Open, bc, d.storage)
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

	val, err := TestInitialize(ctx, &d.cfg.Naming, &d.cfg.Cache, bc, cacheMgn, d.storage, namespaceSvr,
		healthCheckServer, authSvr)
	if err != nil {
		panic(err)
	}
	d.server = val

	// 多等待一会
	d.updateCacheInterval = d.server.Cache().GetUpdateCacheInterval() + time.Millisecond*500

	time.Sleep(5 * time.Second)
	return nil
}

func (d *DiscoverTestSuit) Destroy() {
	d.cancel()
	time.Sleep(5 * time.Second)

	d.storage.Destroy()
	time.Sleep(5 * time.Second)
}

func (d *DiscoverTestSuit) cleanReportClient() {
	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)
			defer dbTx.Rollback()

			if _, err := dbTx.Exec("delete from client"); err != nil {
				panic(err)
			}
			if _, err := dbTx.Exec("delete from client_stat"); err != nil {
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

			if err := dbTx.DeleteBucket([]byte(tblClient)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					dbTx.Rollback()
					panic(err)
				}
			}

			dbTx.Commit()
		}()
	}
}

// 从数据库彻底删除命名空间
func (d *DiscoverTestSuit) cleanNamespace(name string) {
	if name == "" {
		panic("name is empty")
	}

	log.Infof("clean namespace: %s", name)

	if d.storage.Name() == sqldb.STORENAME {
		str := "delete from namespace where name = ?"
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)
			defer dbTx.Rollback()

			if _, err := dbTx.Exec(str, name); err != nil {
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
			if err := dbTx.Bucket([]byte(tblNameNamespace)).DeleteBucket([]byte(name)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					dbTx.Rollback()
					panic(err)
				}
			}

			dbTx.Commit()
		}()
	}
}

// 从数据库彻底删除全部服务
func (d *DiscoverTestSuit) cleanAllService() {

	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

			if _, err := dbTx.Exec("delete from service_metadata"); err != nil {
				dbTx.Rollback()
				panic(err)
			}

			if _, err := dbTx.Exec("delete from service"); err != nil {
				dbTx.Rollback()
				panic(err)
			}

			if _, err := dbTx.Exec("delete from owner_service_map"); err != nil {
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

			if err := dbTx.DeleteBucket([]byte(tblNameService)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					dbTx.Rollback()
					panic(err)
				}
			}

			dbTx.Commit()
		}()
	}
}

// 从数据库彻底删除服务
func (d *DiscoverTestSuit) cleanService(name, namespace string) {

	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

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
				dbTx.Rollback()
				panic(err)
			}

			if _, err := dbTx.Exec("delete from service where id = ?", id); err != nil {
				dbTx.Rollback()
				panic(err)
			}

			if _, err := dbTx.Exec("delete from owner_service_map where service=? and namespace=?", name, namespace); err != nil {
				dbTx.Rollback()
				panic(err)
			}

			dbTx.Commit()
		}()
	} else if d.storage.Name() == boltdb.STORENAME {
		func() {
			svc, err := d.storage.GetService(name, namespace)
			if err != nil {
				panic(err)
			}
			if svc == nil {
				return
			}

			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			defer dbTx.Rollback()

			if err := dbTx.Bucket([]byte(tblNameService)).DeleteBucket([]byte(svc.ID)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					dbTx.Rollback()
					panic(err)
				}
			}

			dbTx.Commit()
		}()
	}
}

// 从数据库彻底删除服务名对应的服务
func (d *DiscoverTestSuit) cleanServiceName(name string, namespace string) {
	// log.Infof("clean service %s, %s", name, namespace)
	d.cleanService(name, namespace)
}

// clean services
func (d *DiscoverTestSuit) cleanServices(services []*apiservice.Service) {

	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

			str := "delete from service where name = ? and namespace = ?"
			cleanOwnerSql := "delete from owner_service_map where service=? and namespace=?"
			for _, service := range services {
				if _, err := dbTx.Exec(str, service.GetName().GetValue(), service.GetNamespace().GetValue()); err != nil {
					panic(err)
				}
				if _, err := dbTx.Exec(cleanOwnerSql, service.GetName().GetValue(), service.GetNamespace().GetValue()); err != nil {
					panic(err)
				}
			}

			dbTx.Commit()
		}()
	} else if d.storage.Name() == boltdb.STORENAME {
		func() {
			ids := make([]string, 0, len(services))

			for _, service := range services {
				svc, err := d.storage.GetService(service.GetName().GetValue(), service.GetNamespace().GetValue())
				if err != nil {
					panic(err)
				}

				ids = append(ids, svc.ID)
			}

			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)

			for i := range ids {
				if err := dbTx.Bucket([]byte(tblNameService)).DeleteBucket([]byte(ids[i])); err != nil {
					if !errors.Is(err, bolt.ErrBucketNotFound) {
						dbTx.Rollback()
						panic(err)
					}
				}
			}
			dbTx.Commit()
		}()
	}

}

// 从数据库彻底删除实例
func (d *DiscoverTestSuit) cleanInstance(instanceID string) {
	if instanceID == "" {
		panic("instanceID is empty")
	}
	log.Infof("clean instance: %s", instanceID)

	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

			str := "delete from instance where id = ?"
			if _, err := dbTx.Exec(str, instanceID); err != nil {
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

			if err := dbTx.Bucket([]byte(tblNameInstance)).DeleteBucket([]byte(instanceID)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					dbTx.Rollback()
					panic(err)
				}
			}
			dbTx.Commit()
		}()
	}

}

// 增加一个服务
func (d *DiscoverTestSuit) createCommonService(t *testing.T, id int) (*apiservice.Service, *apiservice.Service) {
	serviceReq := genMainService(id)
	for i := 0; i < 10; i++ {
		k := fmt.Sprintf("key-%d-%d", id, i)
		v := fmt.Sprintf("value-%d-%d", id, i)
		serviceReq.Metadata[k] = v
	}

	d.cleanServiceName(serviceReq.GetName().GetValue(), serviceReq.GetNamespace().GetValue())

	resp := d.server.CreateServices(d.defaultCtx, []*apiservice.Service{serviceReq})
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}

	return serviceReq, resp.Responses[0].GetService()
}

func (d *DiscoverTestSuit) HeartBeat(t *testing.T, service *apiservice.Service, instanceID string) {
	req := &apiservice.Instance{
		ServiceToken: utils.NewStringValue(service.GetToken().GetValue()),
		Id:           utils.NewStringValue(instanceID),
	}

	resp := d.healthCheckServer.Report(d.defaultCtx, req)
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}
}

func (d *DiscoverTestSuit) GetLastHeartBeat(t *testing.T, service *apiservice.Service, instanceID string) *apiservice.Response {
	req := &apiservice.Instance{
		ServiceToken: utils.NewStringValue(service.GetToken().GetValue()),
		Id:           utils.NewStringValue(instanceID),
	}

	return d.healthCheckServer.GetLastHeartbeat(req)
}


// 生成服务的主要数据
func genMainService(id int) *apiservice.Service {
	return &apiservice.Service{
		Name:       utils.NewStringValue(fmt.Sprintf("test-service-%d", id)),
		Namespace:  utils.NewStringValue(DefaultNamespace),
		Metadata:   make(map[string]string),
		Ports:      utils.NewStringValue(fmt.Sprintf("ports-%d", id)),
		Business:   utils.NewStringValue(fmt.Sprintf("business-%d", id)),
		Department: utils.NewStringValue(fmt.Sprintf("department-%d", id)),
		CmdbMod1:   utils.NewStringValue(fmt.Sprintf("cmdb-mod1-%d", id)),
		CmdbMod2:   utils.NewStringValue(fmt.Sprintf("cmdb-mod2-%d", id)),
		CmdbMod3:   utils.NewStringValue(fmt.Sprintf("cmdb-mod2-%d", id)),
		Comment:    utils.NewStringValue(fmt.Sprintf("service-comment-%d", id)),
		Owners:     utils.NewStringValue(fmt.Sprintf("service-owner-%d", id)),
	}
}

// removeCommonService
func (d *DiscoverTestSuit) removeCommonServices(t *testing.T, req []*apiservice.Service) {
	if resp := d.server.DeleteServices(d.defaultCtx, req); !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}
}

// removeCommonService
func (d *DiscoverTestSuit) removeCommonServiceAliases(t *testing.T, req []*apiservice.ServiceAlias) {
	if resp := d.server.DeleteServiceAliases(d.defaultCtx, req); !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}
}

// 新增一个实例
func (d *DiscoverTestSuit) createCommonInstance(t *testing.T, svc *apiservice.Service, id int) (
	*apiservice.Instance, *apiservice.Instance) {
	instanceReq := &apiservice.Instance{
		ServiceToken: utils.NewStringValue(svc.GetToken().GetValue()),
		Service:      utils.NewStringValue(svc.GetName().GetValue()),
		Namespace:    utils.NewStringValue(svc.GetNamespace().GetValue()),
		VpcId:        utils.NewStringValue(fmt.Sprintf("vpcid-%d", id)),
		Host:         utils.NewStringValue(fmt.Sprintf("9.9.9.%d", id)),
		Port:         utils.NewUInt32Value(8000 + uint32(id)),
		Protocol:     utils.NewStringValue(fmt.Sprintf("protocol-%d", id)),
		Version:      utils.NewStringValue(fmt.Sprintf("version-%d", id)),
		Priority:     utils.NewUInt32Value(1 + uint32(id)%10),
		Weight:       utils.NewUInt32Value(1 + uint32(id)%1000),
		HealthCheck: &apiservice.HealthCheck{
			Type: apiservice.HealthCheck_HEARTBEAT,
			Heartbeat: &apiservice.HeartbeatHealthCheck{
				Ttl: utils.NewUInt32Value(3),
			},
		},
		Healthy:  utils.NewBoolValue(false), // 默认是非健康，因为打开了healthCheck
		Isolate:  utils.NewBoolValue(false),
		LogicSet: utils.NewStringValue(fmt.Sprintf("logic-set-%d", id)),
		Metadata: map[string]string{
			"internal-personal-xxx":        fmt.Sprintf("internal-personal-xxx_%d", id),
			"2my-meta":                     fmt.Sprintf("my-meta-%d", id),
			"my-meta-a1":                   "1111",
			"smy-xmeta-h2":                 "2222",
			"my-1meta-o3":                  "2222",
			"my-2meta-4c":                  "2222",
			"my-3meta-d5":                  "2222",
			"dmy-meta-6p":                  "2222",
			"1my-pmeta-d7":                 "2222",
			"my-dmeta-8c":                  "2222",
			"my-xmeta-9p":                  "2222",
			"other-meta-x":                 "xxx",
			"other-meta-1":                 "xx11",
			"amy-instance":                 "my-instance",
			"very-long-key-data-xxxxxxxxx": "Y",
			"very-long-key-data-uuuuuuuuu": "P",
		},
	}

	resp := d.server.CreateInstances(d.defaultCtx, []*apiservice.Instance{instanceReq})
	if respSuccess(resp) {
		return instanceReq, resp.Responses[0].GetInstance()
	}

	if resp.GetCode().GetValue() != api.ExistedResource {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}

	// repeated
	InstanceID, _ := utils.CalculateInstanceID(instanceReq.GetNamespace().GetValue(), instanceReq.GetService().GetValue(),
		instanceReq.GetVpcId().GetValue(), instanceReq.GetHost().GetValue(), instanceReq.GetPort().GetValue())
	d.cleanInstance(InstanceID)
	t.Logf("repeatd create instance(%s)", InstanceID)
	resp = d.server.CreateInstances(d.defaultCtx, []*apiservice.Instance{instanceReq})
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}

	return instanceReq, resp.Responses[0].GetInstance()
}

// 指定 IP 和端口为一个服务创建实例
func (d *DiscoverTestSuit) addHostPortInstance(t *testing.T, service *apiservice.Service, host string, port uint32) (
	*apiservice.Instance, *apiservice.Instance) {
	instanceReq := &apiservice.Instance{
		ServiceToken: utils.NewStringValue(service.GetToken().GetValue()),
		Service:      utils.NewStringValue(service.GetName().GetValue()),
		Namespace:    utils.NewStringValue(service.GetNamespace().GetValue()),
		Host:         utils.NewStringValue(host),
		Port:         utils.NewUInt32Value(port),
		Healthy:      utils.NewBoolValue(true),
		Isolate:      utils.NewBoolValue(false),
	}
	resp := d.server.CreateInstances(d.defaultCtx, []*apiservice.Instance{instanceReq})
	if respSuccess(resp) {
		return instanceReq, resp.Responses[0].GetInstance()
	}

	if resp.GetCode().GetValue() != api.ExistedResource {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}
	return instanceReq, resp.Responses[0].GetInstance()
}

// 添加一个实例
func (d *DiscoverTestSuit) addInstance(t *testing.T, ins *apiservice.Instance) (
	*apiservice.Instance, *apiservice.Instance) {
	resp := d.server.CreateInstances(d.defaultCtx, []*apiservice.Instance{ins})
	if !respSuccess(resp) {
		if resp.GetCode().GetValue() == api.ExistedResource {
			id, _ := utils.CalculateInstanceID(ins.GetNamespace().GetValue(), ins.GetService().GetValue(),
				ins.GetHost().GetValue(), ins.GetHost().GetValue(), ins.GetPort().GetValue())
			d.cleanInstance(id)
		}
	} else {
		return ins, resp.Responses[0].GetInstance()
	}

	resp = d.server.CreateInstances(d.defaultCtx, []*apiservice.Instance{ins})
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}

	return ins, resp.Responses[0].GetInstance()
}

// 删除一个实例
func (d *DiscoverTestSuit) removeCommonInstance(t *testing.T, service *apiservice.Service, instanceID string) {
	req := &apiservice.Instance{
		ServiceToken: utils.NewStringValue(service.GetToken().GetValue()),
		Id:           utils.NewStringValue(instanceID),
	}

	resp := d.server.DeleteInstances(d.defaultCtx, []*apiservice.Instance{req})
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}

}

// 通过四元组或者五元组删除实例
func (d *DiscoverTestSuit) removeInstanceWithAttrs(t *testing.T, service *apiservice.Service, instance *apiservice.Instance) {
	req := &apiservice.Instance{
		ServiceToken: utils.NewStringValue(service.GetToken().GetValue()),
		Service:      utils.NewStringValue(service.GetName().GetValue()),
		Namespace:    utils.NewStringValue(service.GetNamespace().GetValue()),
		VpcId:        utils.NewStringValue(instance.GetVpcId().GetValue()),
		Host:         utils.NewStringValue(instance.GetHost().GetValue()),
		Port:         utils.NewUInt32Value(instance.GetPort().GetValue()),
	}
	if resp := d.server.DeleteInstances(d.defaultCtx, []*apiservice.Instance{req}); !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}
}

// 创建一个路由配置
func (d *DiscoverTestSuit) createCommonRoutingConfig(t *testing.T, service *apiservice.Service, inCount int, outCount int) (*apitraffic.Routing, *apitraffic.Routing) {
	inBounds := make([]*apitraffic.Route, 0, inCount)
	for i := 0; i < inCount; i++ {
		matchString := &apimodel.MatchString{
			Type:  apimodel.MatchString_EXACT,
			Value: utils.NewStringValue(fmt.Sprintf("in-meta-value-%d", i)),
		}
		source := &apitraffic.Source{
			Service:   utils.NewStringValue(fmt.Sprintf("in-source-service-%d", i)),
			Namespace: utils.NewStringValue(fmt.Sprintf("in-source-service-%d", i)),
			Metadata: map[string]*apimodel.MatchString{
				fmt.Sprintf("in-metadata-%d", i): matchString,
			},
		}
		destination := &apitraffic.Destination{
			Service:   service.Name,
			Namespace: service.Namespace,
			Metadata: map[string]*apimodel.MatchString{
				fmt.Sprintf("in-metadata-%d", i): matchString,
			},
			Priority: utils.NewUInt32Value(120),
			Weight:   utils.NewUInt32Value(100),
			Transfer: utils.NewStringValue("abcdefg"),
		}

		entry := &apitraffic.Route{
			Sources:      []*apitraffic.Source{source},
			Destinations: []*apitraffic.Destination{destination},
		}
		inBounds = append(inBounds, entry)
	}

	conf := &apitraffic.Routing{
		Service:      utils.NewStringValue(service.GetName().GetValue()),
		Namespace:    utils.NewStringValue(service.GetNamespace().GetValue()),
		Inbounds:     inBounds,
		ServiceToken: utils.NewStringValue(service.GetToken().GetValue()),
	}

	// TODO 是否应该先删除routing

	resp := d.server.CreateRoutingConfigs(d.defaultCtx, []*apitraffic.Routing{conf})
	if !respSuccess(resp) {
		t.Fatalf("error: %+v", resp)
	}

	return conf, resp.Responses[0].GetRouting()
}

// 创建一个路由配置
func (d *DiscoverTestSuit) createCommonRoutingConfigV1IntoOldStore(t *testing.T, service *apiservice.Service,
	inCount int, outCount int) (*apitraffic.Routing, *apitraffic.Routing) {

	inBounds := make([]*apitraffic.Route, 0, inCount)
	for i := 0; i < inCount; i++ {
		matchString := &apimodel.MatchString{
			Type:  apimodel.MatchString_EXACT,
			Value: utils.NewStringValue(fmt.Sprintf("in-meta-value-%d", i)),
		}
		source := &apitraffic.Source{
			Service:   utils.NewStringValue(fmt.Sprintf("in-source-service-%d", i)),
			Namespace: utils.NewStringValue(fmt.Sprintf("in-source-service-%d", i)),
			Metadata: map[string]*apimodel.MatchString{
				fmt.Sprintf("in-metadata-%d", i): matchString,
			},
		}
		destination := &apitraffic.Destination{
			Service:   service.Name,
			Namespace: service.Namespace,
			Metadata: map[string]*apimodel.MatchString{
				fmt.Sprintf("in-metadata-%d", i): matchString,
			},
			Priority: utils.NewUInt32Value(120),
			Weight:   utils.NewUInt32Value(100),
			Transfer: utils.NewStringValue("abcdefg"),
		}

		entry := &apitraffic.Route{
			Sources:      []*apitraffic.Source{source},
			Destinations: []*apitraffic.Destination{destination},
		}
		inBounds = append(inBounds, entry)
	}

	conf := &apitraffic.Routing{
		Service:      utils.NewStringValue(service.GetName().GetValue()),
		Namespace:    utils.NewStringValue(service.GetNamespace().GetValue()),
		Inbounds:     inBounds,
		ServiceToken: utils.NewStringValue(service.GetToken().GetValue()),
	}

	resp := d.server.(*serverAuthAbility).targetServer.CreateRoutingConfig(d.defaultCtx, conf)
	if !respSuccess(resp) {
		t.Fatalf("error: %+v", resp)
	}

	return conf, resp.GetRouting()
}

func mockRoutingV1(serviceName, serviceNamespace string, inCount int) *apitraffic.Routing {
	inBounds := make([]*apitraffic.Route, 0, inCount)
	for i := 0; i < inCount; i++ {
		matchString := &apimodel.MatchString{
			Type:  apimodel.MatchString_EXACT,
			Value: utils.NewStringValue(fmt.Sprintf("in-meta-value-%d", i)),
		}
		source := &apitraffic.Source{
			Service:   utils.NewStringValue(fmt.Sprintf("in-source-service-%d", i)),
			Namespace: utils.NewStringValue(fmt.Sprintf("in-source-service-%d", i)),
			Metadata: map[string]*apimodel.MatchString{
				fmt.Sprintf("in-metadata-%d", i): matchString,
			},
		}
		destination := &apitraffic.Destination{
			Service:   utils.NewStringValue(serviceName),
			Namespace: utils.NewStringValue(serviceNamespace),
			Metadata: map[string]*apimodel.MatchString{
				fmt.Sprintf("in-metadata-%d", i): matchString,
			},
			Priority: utils.NewUInt32Value(120),
			Weight:   utils.NewUInt32Value(100),
			Transfer: utils.NewStringValue("abcdefg"),
		}

		entry := &apitraffic.Route{
			Sources:      []*apitraffic.Source{source},
			Destinations: []*apitraffic.Destination{destination},
		}
		inBounds = append(inBounds, entry)
	}

	conf := &apitraffic.Routing{
		Service:   utils.NewStringValue(serviceName),
		Namespace: utils.NewStringValue(serviceNamespace),
		Inbounds:  inBounds,
	}

	return conf
}

func mockRoutingV2(t *testing.T, cnt int32) []*apitraffic.RouteRule {
	rules := make([]*apitraffic.RouteRule, 0, cnt)
	for i := int32(0); i < cnt; i++ {
		matchString := &apimodel.MatchString{
			Type:  apimodel.MatchString_EXACT,
			Value: utils.NewStringValue(fmt.Sprintf("in-meta-value-%d", i)),
		}
		source := &apitraffic.SourceService{
			Service:   fmt.Sprintf("in-source-service-%d", i),
			Namespace: fmt.Sprintf("in-source-service-%d", i),
			Arguments: []*apitraffic.SourceMatch{
				{},
			},
		}
		destination := &apitraffic.DestinationGroup{
			Service:   fmt.Sprintf("in-destination-service-%d", i),
			Namespace: fmt.Sprintf("in-destination-service-%d", i),
			Labels: map[string]*apimodel.MatchString{
				fmt.Sprintf("in-metadata-%d", i): matchString,
			},
			Priority: 120,
			Weight:   100,
			Transfer: "abcdefg",
		}

		entry := &apitraffic.RuleRoutingConfig{
			Sources:      []*apitraffic.SourceService{source},
			Destinations: []*apitraffic.DestinationGroup{destination},
		}

		any, err := ptypes.MarshalAny(entry)
		if err != nil {
			t.Fatal(err)
		}

		item := &apitraffic.RouteRule{
			Id:            "",
			Name:          fmt.Sprintf("test-routing-name-%d", i),
			Namespace:     "",
			Enable:        false,
			RoutingPolicy: apitraffic.RoutingPolicy_RulePolicy,
			RoutingConfig: any,
			Revision:      "",
			Etime:         "",
			Priority:      0,
			Description:   "",
		}

		rules = append(rules, item)
	}

	return rules
}

// 创建一个路由配置
func (d *DiscoverTestSuit) createCommonRoutingConfigV2(t *testing.T, cnt int32) []*apitraffic.RouteRule {
	rules := mockRoutingV2(t, cnt)

	return d.createCommonRoutingConfigV2WithReq(t, rules)
}

// 创建一个路由配置
func (d *DiscoverTestSuit) createCommonRoutingConfigV2WithReq(t *testing.T, rules []*apitraffic.RouteRule) []*apitraffic.RouteRule {
	resp := d.server.CreateRoutingConfigsV2(d.defaultCtx, rules)
	if !respSuccess(resp) {
		t.Fatalf("error: %+v", resp)
	}

	if len(rules) != len(resp.GetResponses()) {
		t.Fatal("error: create v2 routings not equal resp")
	}

	ret := []*apitraffic.RouteRule{}
	for i := range resp.GetResponses() {
		item := resp.GetResponses()[i]
		msg := &apitraffic.RouteRule{}

		if err := ptypes.UnmarshalAny(item.GetData(), msg); err != nil {
			t.Fatal(err)
			return nil
		}

		ret = append(ret, msg)
	}

	return ret
}

// 删除一个路由配置
func (d *DiscoverTestSuit) deleteCommonRoutingConfig(t *testing.T, req *apitraffic.Routing) {
	resp := d.server.DeleteRoutingConfigs(d.defaultCtx, []*apitraffic.Routing{req})
	if !respSuccess(resp) {
		t.Fatalf("%s", resp.GetInfo().GetValue())
	}
}

// 删除一个路由配置
func (d *DiscoverTestSuit) deleteCommonRoutingConfigV2(t *testing.T, req *apitraffic.RouteRule) {
	resp := d.server.DeleteRoutingConfigsV2(d.defaultCtx, []*apitraffic.RouteRule{req})
	if !respSuccess(resp) {
		t.Fatalf("%s", resp.GetInfo())
	}
}

// 更新一个路由配置
func (d *DiscoverTestSuit) updateCommonRoutingConfig(t *testing.T, req *apitraffic.Routing) {
	resp := d.server.UpdateRoutingConfigs(d.defaultCtx, []*apitraffic.Routing{req})
	if !respSuccess(resp) {
		t.Fatalf("%s", resp.GetInfo().GetValue())
	}
}

// 彻底删除一个路由配置
func (d *DiscoverTestSuit) cleanCommonRoutingConfig(service string, namespace string) {

	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

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

			dbTx.Commit()
		}()
	} else if d.storage.Name() == boltdb.STORENAME {
		func() {
			svc, err := d.storage.GetService(service, namespace)
			if err != nil {
				panic(err)
			}

			if svc == nil {
				return
			}

			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)
			defer dbTx.Rollback()

			v1Bucket := dbTx.Bucket([]byte(tblNameRouting))
			if v1Bucket != nil {
				if err := v1Bucket.DeleteBucket([]byte(svc.ID)); err != nil {
					if !errors.Is(err, bolt.ErrBucketNotFound) {
						dbTx.Rollback()
						panic(err)
					}
				}
			}

			if err := dbTx.DeleteBucket([]byte(tblNameRoutingV2)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					dbTx.Rollback()
					panic(err)
				}
			}
			dbTx.Commit()
		}()
	}
}

func (d *DiscoverTestSuit) truncateCommonRoutingConfigV2() {
	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)
			defer dbTx.Rollback()

			str := "delete from routing_config_v2"
			if _, err := dbTx.Exec(str); err != nil {
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

			if err := dbTx.DeleteBucket([]byte(tblNameRoutingV2)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					dbTx.Rollback()
					panic(err)
				}
			}

			dbTx.Commit()
		}()
	}
}

// 彻底删除一个路由配置
func (d *DiscoverTestSuit) cleanCommonRoutingConfigV2(rules []*apitraffic.RouteRule) {

	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)
			defer dbTx.Rollback()

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

			for i := range rules {
				if err := dbTx.Bucket([]byte(tblNameRoutingV2)).DeleteBucket([]byte(rules[i].Id)); err != nil {
					if !errors.Is(err, bolt.ErrBucketNotFound) {
						dbTx.Rollback()
						panic(err)
					}
				}
			}

			dbTx.Commit()
		}()
	}
}

func (d *DiscoverTestSuit) CheckGetService(t *testing.T, expectReqs []*apiservice.Service, actualReqs []*apiservice.Service) {
	if len(expectReqs) != len(actualReqs) {
		t.Fatalf("error: %d %d", len(expectReqs), len(actualReqs))
	}

	for _, expect := range expectReqs {
		found := false
		for _, actual := range actualReqs {
			if expect.GetName().GetValue() != actual.GetName().GetValue() ||
				expect.GetNamespace().GetValue() != actual.GetNamespace().GetValue() {
				continue
			}

			found = true

			if expect.GetPorts().GetValue() != actual.GetPorts().GetValue() ||
				expect.GetOwners().GetValue() != actual.GetOwners().GetValue() ||
				expect.GetComment().GetValue() != actual.GetComment().GetValue() ||
				actual.GetToken().GetValue() != "" || actual.GetRevision().GetValue() == "" {
				t.Fatalf("error: %+v, %+v", expect, actual)
			}

			if len(expect.Metadata) != len(actual.Metadata) {
				t.Fatalf("error: %d, %d", len(expect.Metadata), len(actual.Metadata))
			}
			for key, value := range expect.Metadata {
				match, ok := actual.Metadata[key]
				if !ok {
					t.Fatalf("error")
				}
				if value != match {
					t.Fatalf("error")
				}
			}
		}
		if !found {
			t.Fatalf("error: %s, %s", expect.GetName().GetValue(), expect.GetNamespace().GetValue())
		}

	}
}

// 检查服务发现的字段是否一致
func (d *DiscoverTestSuit) discoveryCheck(t *testing.T, req *apiservice.Service, resp *apiservice.DiscoverResponse) {
	if resp == nil {
		t.Fatalf("error")
	}

	if resp.GetService().GetName().GetValue() != req.GetName().GetValue() ||
		resp.GetService().GetNamespace().GetValue() != req.GetNamespace().GetValue() ||
		resp.GetService().GetRevision().GetValue() == "" {
		t.Fatalf("error: %+v", resp)
	}

	if resp.Service == nil {
		t.Fatalf("error")
	}
	// t.Logf("%+v", resp.Service)

	if resp.Service.GetName().GetValue() != req.GetName().GetValue() ||
		resp.Service.GetNamespace().GetValue() != req.GetNamespace().GetValue() {
		t.Fatalf("error: %+v", resp.Service)
	}
}

// 实例校验
func instanceCheck(t *testing.T, expect *apiservice.Instance, actual *apiservice.Instance) {
	// #lizard forgives
	switch {
	case expect.GetService().GetValue() != actual.GetService().GetValue():
		t.Fatalf("error %s---%s", expect.GetService().GetValue(), actual.GetService().GetValue())
	case expect.GetNamespace().GetValue() != actual.GetNamespace().GetValue():
		t.Fatalf("error")
	case expect.GetPort().GetValue() != actual.GetPort().GetValue():
		t.Fatalf("error")
	case expect.GetHost().GetValue() != actual.GetHost().GetValue():
		t.Fatalf("error")
	case expect.GetVpcId().GetValue() != actual.GetVpcId().GetValue():
		t.Fatalf("error")
	case expect.GetProtocol().GetValue() != actual.GetProtocol().GetValue():
		t.Fatalf("error")
	case expect.GetVersion().GetValue() != actual.GetVersion().GetValue():
		t.Fatalf("error")
	case expect.GetWeight().GetValue() != actual.GetWeight().GetValue():
		t.Fatalf("error")
	case expect.GetHealthy().GetValue() != actual.GetHealthy().GetValue():
		t.Fatalf("error")
	case expect.GetIsolate().GetValue() != actual.GetIsolate().GetValue():
		t.Fatalf("error")
	case expect.GetLogicSet().GetValue() != actual.GetLogicSet().GetValue():
		t.Fatalf("error")
	default:
		break

		// 实例创建，无法指定cmdb信息
		/*case expect.GetCmdbRegion().GetValue() != actual.GetCmdbRegion().GetValue():
		  	t.Fatalf("error")
		  case expect.GetCmdbCampus().GetValue() != actual.GetCmdbRegion().GetValue():
		  	t.Fatalf("error")
		  case expect.GetCmdbZone().GetValue() != actual.GetCmdbZone().GetValue():
		  	t.Fatalf("error")*/

	}
	for key, value := range expect.GetMetadata() {
		actualValue := actual.GetMetadata()[key]
		if value != actualValue {
			t.Fatalf("error %+v, %+v", expect.Metadata, actual.Metadata)
		}
	}

	if expect.GetHealthCheck().GetType() != actual.GetHealthCheck().GetType() {
		t.Fatalf("error")
	}
	if expect.GetHealthCheck().GetHeartbeat().GetTtl().GetValue() !=
		actual.GetHealthCheck().GetHeartbeat().GetTtl().GetValue() {
		t.Fatalf("error")
	}
}

// 完整对比service的各个属性
func serviceCheck(t *testing.T, expect *apiservice.Service, actual *apiservice.Service) {
	switch {
	case expect.GetName().GetValue() != actual.GetName().GetValue():
		t.Fatalf("error")
	case expect.GetNamespace().GetValue() != actual.GetNamespace().GetValue():
		t.Fatalf("error")
	case expect.GetPorts().GetValue() != actual.GetPorts().GetValue():
		t.Fatalf("error")
	case expect.GetBusiness().GetValue() != actual.GetBusiness().GetValue():
		t.Fatalf("error")
	case expect.GetDepartment().GetValue() != actual.GetDepartment().GetValue():
		t.Fatalf("error")
	case expect.GetCmdbMod1().GetValue() != actual.GetCmdbMod1().GetValue():
		t.Fatalf("error")
	case expect.GetCmdbMod2().GetValue() != actual.GetCmdbMod2().GetValue():
		t.Fatalf("error")
	case expect.GetCmdbMod3().GetValue() != actual.GetCmdbMod3().GetValue():
		t.Fatalf("error")
	case expect.GetComment().GetValue() != actual.GetComment().GetValue():
		t.Fatalf("error")
	case expect.GetOwners().GetValue() != actual.GetOwners().GetValue():
		t.Fatalf("error")
	default:
		break
	}

	for key, value := range expect.GetMetadata() {
		actualValue := actual.GetMetadata()[key]
		if actualValue != value {
			t.Fatalf("error")
		}
	}
}

// 创建限流规则
func (d *DiscoverTestSuit) createCommonRateLimit(t *testing.T, service *apiservice.Service, index int) (*apitraffic.Rule, *apitraffic.Rule) {
	// 先不考虑Cluster
	rateLimit := &apitraffic.Rule{
		Name:      &wrappers.StringValue{Value: fmt.Sprintf("rule_name_%d", index)},
		Service:   service.GetName(),
		Namespace: service.GetNamespace(),
		Priority:  utils.NewUInt32Value(uint32(index)),
		Resource:  apitraffic.Rule_QPS,
		Type:      apitraffic.Rule_GLOBAL,
		Arguments: []*apitraffic.MatchArgument{
			{
				Type: apitraffic.MatchArgument_CUSTOM,
				Key:  fmt.Sprintf("name-%d", index),
				Value: &apimodel.MatchString{
					Type:  apimodel.MatchString_EXACT,
					Value: utils.NewStringValue(fmt.Sprintf("value-%d", index)),
				},
			},
			{
				Type: apitraffic.MatchArgument_CUSTOM,
				Key:  fmt.Sprintf("name-%d", index+1),
				Value: &apimodel.MatchString{
					Type:  apimodel.MatchString_EXACT,
					Value: utils.NewStringValue(fmt.Sprintf("value-%d", index+1)),
				},
			},
		},
		Amounts: []*apitraffic.Amount{
			{
				MaxAmount: utils.NewUInt32Value(uint32(10 * index)),
				ValidDuration: &duration.Duration{
					Seconds: int64(index),
					Nanos:   int32(index),
				},
			},
		},
		Action:  utils.NewStringValue(fmt.Sprintf("behavior-%d", index)),
		Disable: utils.NewBoolValue(false),
		Report: &apitraffic.Report{
			Interval: &duration.Duration{
				Seconds: int64(index),
			},
			AmountPercent: utils.NewUInt32Value(uint32(index)),
		},
		ServiceToken: utils.NewStringValue(service.GetToken().GetValue()),
	}

	resp := d.server.CreateRateLimits(d.defaultCtx, []*apitraffic.Rule{rateLimit})
	if !respSuccess(resp) {
		t.Fatalf("error: %+v", resp)
	}
	return rateLimit, resp.Responses[0].GetRateLimit()
}

// 删除限流规则
func (d *DiscoverTestSuit) deleteRateLimit(t *testing.T, rateLimit *apitraffic.Rule) {
	if resp := d.server.DeleteRateLimits(d.defaultCtx, []*apitraffic.Rule{rateLimit}); !respSuccess(resp) {
		t.Fatalf("%s", resp.GetInfo().GetValue())
	}
}

// 更新单个限流规则
func (d *DiscoverTestSuit) updateRateLimit(t *testing.T, rateLimit *apitraffic.Rule) {
	if resp := d.server.UpdateRateLimits(d.defaultCtx, []*apitraffic.Rule{rateLimit}); !respSuccess(resp) {
		t.Fatalf("%s", resp.GetInfo().GetValue())
	}
}

// 彻底删除限流规则
func (d *DiscoverTestSuit) cleanRateLimit(id string) {

	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

			str := `delete from ratelimit_config where id = ?`
			if _, err := dbTx.Exec(str, id); err != nil {
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

			if err := dbTx.Bucket([]byte(tblRateLimitConfig)).DeleteBucket([]byte(id)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					dbTx.Rollback()
					panic(err)
				}
			}
			dbTx.Commit()
		}()
	}
}

// 彻底删除限流规则版本号
func (d *DiscoverTestSuit) cleanRateLimitRevision(service, namespace string) {

	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

			str := `delete from ratelimit_revision using ratelimit_revision, service where service_id = service.id and name = ? and namespace = ?`
			if _, err := dbTx.Exec(str, service, namespace); err != nil {
				panic(err)
			}

			dbTx.Commit()
		}()
	} else if d.storage.Name() == boltdb.STORENAME {
		func() {

			svc, err := d.storage.GetService(service, namespace)
			if err != nil {
				panic(err)
			}

			if svc == nil {
				panic("service not found " + service + ", namespace" + namespace)
			}

			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)

			if err := dbTx.Bucket([]byte(tblRateLimitRevision)).DeleteBucket([]byte(svc.ID)); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					dbTx.Rollback()
					panic(err)
				}
			}
			dbTx.Commit()
		}()
	}
}

// 更新限流规则内容
func updateRateLimitContent(rateLimit *apitraffic.Rule, index int) {
	rateLimit.Priority = utils.NewUInt32Value(uint32(index))
	rateLimit.Resource = apitraffic.Rule_CONCURRENCY
	rateLimit.Type = apitraffic.Rule_LOCAL
	rateLimit.Labels = map[string]*apimodel.MatchString{
		fmt.Sprintf("name-%d", index): {
			Type:  apimodel.MatchString_EXACT,
			Value: utils.NewStringValue(fmt.Sprintf("value-%d", index)),
		},
		fmt.Sprintf("name-%d", index+1): {
			Type:  apimodel.MatchString_REGEX,
			Value: utils.NewStringValue(fmt.Sprintf("value-%d", index+1)),
		},
	}
	rateLimit.Amounts = []*apitraffic.Amount{
		{
			MaxAmount: utils.NewUInt32Value(uint32(index)),
			ValidDuration: &duration.Duration{
				Seconds: int64(index),
			},
		},
	}
	rateLimit.Action = utils.NewStringValue(fmt.Sprintf("value-%d", index))
	rateLimit.Disable = utils.NewBoolValue(true)
	rateLimit.Report = &apitraffic.Report{
		Interval: &duration.Duration{
			Seconds: int64(index),
		},
		AmountPercent: utils.NewUInt32Value(uint32(index)),
	}
}

/*
 * @brief 对比限流规则的各个属性
 */
func checkRateLimit(t *testing.T, expect *apitraffic.Rule, actual *apitraffic.Rule) {
	switch {
	case expect.GetId().GetValue() != actual.GetId().GetValue():
		t.Fatalf("error id, expect %s, actual %s", expect.GetId().GetValue(), actual.GetId().GetValue())
	case expect.GetService().GetValue() != actual.GetService().GetValue():
		t.Fatalf("error service, expect %s, actual %s", expect.GetService().GetValue(), actual.GetService().GetValue())
	case expect.GetNamespace().GetValue() != actual.GetNamespace().GetValue():
		t.Fatalf("error namespace, expect %s, actual %s", expect.GetNamespace().GetValue(), actual.GetNamespace().GetValue())
	case expect.GetPriority().GetValue() != actual.GetPriority().GetValue():
		t.Fatalf("error priority, expect %v, actual %v", expect.GetPriority().GetValue(), actual.GetPriority().GetValue())
	case expect.GetResource() != actual.GetResource():
		t.Fatalf("error resource, expect %v, actual %v", expect.GetResource(), actual.GetResource())
	case expect.GetType() != actual.GetType():
		t.Fatalf("error type, expect %v, actual %v", expect.GetType(), actual.GetType())
	case expect.GetDisable().GetValue() != actual.GetDisable().GetValue():
		t.Fatalf("error disable, expect %v, actual %v", expect.GetDisable().GetValue(), actual.GetDisable().GetValue())
	case expect.GetAction().GetValue() != actual.GetAction().GetValue():
		t.Fatalf("error action, expect %s, actual %s", expect.GetAction().GetValue(), actual.GetAction().GetValue())
	default:
		break
	}

	expectSubset, err := json.Marshal(expect.GetSubset())
	if err != nil {
		panic(err)
	}
	actualSubset, err := json.Marshal(actual.GetSubset())
	if err != nil {
		panic(err)
	}
	if string(expectSubset) != string(actualSubset) {
		t.Fatal("error subset")
	}

	expectLabels, err := json.Marshal(expect.GetArguments())
	if err != nil {
		panic(err)
	}
	actualLabels, err := json.Marshal(actual.GetArguments())
	if err != nil {
		panic(err)
	}
	if string(expectLabels) != string(actualLabels) {
		t.Fatal("error labels")
	}

	expectAmounts, err := json.Marshal(expect.GetAmounts())
	if err != nil {
		panic(err)
	}
	actualAmounts, err := json.Marshal(actual.GetAmounts())
	if err != nil {
		panic(err)
	}
	if string(expectAmounts) != string(actualAmounts) {
		t.Fatal("error amounts")
	}
}

// 增加熔断规则
func (d *DiscoverTestSuit) createCommonCircuitBreaker(t *testing.T, id int) (*apifault.CircuitBreaker, *apifault.CircuitBreaker) {
	circuitBreaker := &apifault.CircuitBreaker{
		Name:       utils.NewStringValue(fmt.Sprintf("name-test-%d", id)),
		Namespace:  utils.NewStringValue(DefaultNamespace),
		Owners:     utils.NewStringValue("owner-test"),
		Comment:    utils.NewStringValue("comment-test"),
		Department: utils.NewStringValue("department-test"),
		Business:   utils.NewStringValue("business-test"),
	}
	ruleNum := 1
	// 填充source规则
	sources := make([]*apifault.SourceMatcher, 0, ruleNum)
	for i := 0; i < ruleNum; i++ {
		source := &apifault.SourceMatcher{
			Service:   utils.NewStringValue(fmt.Sprintf("service-test-%d", i)),
			Namespace: utils.NewStringValue(fmt.Sprintf("namespace-test-%d", i)),
			Labels: map[string]*apimodel.MatchString{
				fmt.Sprintf("name-%d", i): {
					Type:  apimodel.MatchString_EXACT,
					Value: utils.NewStringValue(fmt.Sprintf("value-%d", i)),
				},
				fmt.Sprintf("name-%d", i+1): {
					Type:  apimodel.MatchString_REGEX,
					Value: utils.NewStringValue(fmt.Sprintf("value-%d", i+1)),
				},
			},
		}
		sources = append(sources, source)
	}

	// 填充destination规则
	destinations := make([]*apifault.DestinationSet, 0, ruleNum)
	for i := 0; i < ruleNum; i++ {
		destination := &apifault.DestinationSet{
			Service:   utils.NewStringValue(fmt.Sprintf("service-test-%d", i)),
			Namespace: utils.NewStringValue(fmt.Sprintf("namespace-test-%d", i)),
			Metadata: map[string]*apimodel.MatchString{
				fmt.Sprintf("name-%d", i): {
					Type:  apimodel.MatchString_EXACT,
					Value: utils.NewStringValue(fmt.Sprintf("value-%d", i)),
				},
				fmt.Sprintf("name-%d", i+1): {
					Type:  apimodel.MatchString_REGEX,
					Value: utils.NewStringValue(fmt.Sprintf("value-%d", i+1)),
				},
			},
			Resource: 0,
			Type:     0,
			Scope:    0,
			MetricWindow: &duration.Duration{
				Seconds: int64(i),
			},
			MetricPrecision: utils.NewUInt32Value(uint32(i)),
			UpdateInterval: &duration.Duration{
				Seconds: int64(i),
			},
			Recover: &apifault.RecoverConfig{},
			Policy:  &apifault.CbPolicy{},
		}
		destinations = append(destinations, destination)
	}

	// 填充inbound规则
	inbounds := make([]*apifault.CbRule, 0, ruleNum)
	for i := 0; i < ruleNum; i++ {
		inbound := &apifault.CbRule{
			Sources:      sources,
			Destinations: destinations,
		}
		inbounds = append(inbounds, inbound)
	}
	// 填充outbound规则
	outbounds := make([]*apifault.CbRule, 0, ruleNum)
	for i := 0; i < ruleNum; i++ {
		outbound := &apifault.CbRule{
			Sources:      sources,
			Destinations: destinations,
		}
		outbounds = append(outbounds, outbound)
	}
	circuitBreaker.Inbounds = inbounds
	circuitBreaker.Outbounds = outbounds

	resp := d.server.CreateCircuitBreakers(d.defaultCtx, []*apifault.CircuitBreaker{circuitBreaker})
	if !respSuccess(resp) {
		t.Fatalf("error: %+v", resp)
	}
	return circuitBreaker, resp.Responses[0].GetCircuitBreaker()
}

// 增加熔断规则版本
func (d *DiscoverTestSuit) createCommonCircuitBreakerVersion(t *testing.T, cb *apifault.CircuitBreaker, index int) (
	*apifault.CircuitBreaker, *apifault.CircuitBreaker) {
	cbVersion := &apifault.CircuitBreaker{
		Id:        cb.GetId(),
		Name:      cb.GetName(),
		Namespace: cb.GetNamespace(),
		Version:   utils.NewStringValue(fmt.Sprintf("test-version-%d", index)),
		Inbounds:  cb.GetInbounds(),
		Outbounds: cb.GetOutbounds(),
		Token:     cb.GetToken(),
	}

	resp := d.server.CreateCircuitBreakerVersions(d.defaultCtx, []*apifault.CircuitBreaker{cbVersion})
	if !respSuccess(resp) {
		t.Fatalf("error: %+v", resp)
	}
	return cbVersion, resp.Responses[0].GetCircuitBreaker()
}

// 删除熔断规则
func (d *DiscoverTestSuit) deleteCircuitBreaker(t *testing.T, circuitBreaker *apifault.CircuitBreaker) {
	if resp := d.server.DeleteCircuitBreakers(d.defaultCtx, []*apifault.CircuitBreaker{circuitBreaker}); !respSuccess(resp) {
		t.Fatalf("%s", resp.GetInfo().GetValue())
	}
}

// 更新熔断规则内容
func (d *DiscoverTestSuit) updateCircuitBreaker(t *testing.T, circuitBreaker *apifault.CircuitBreaker) {
	if resp := d.server.UpdateCircuitBreakers(d.defaultCtx, []*apifault.CircuitBreaker{circuitBreaker}); !respSuccess(resp) {
		t.Fatalf("%s", resp.GetInfo().GetValue())
	}
}

// 发布熔断规则
func (d *DiscoverTestSuit) releaseCircuitBreaker(t *testing.T, cb *apifault.CircuitBreaker, service *apiservice.Service) {
	release := &apiservice.ConfigRelease{
		Service:        service,
		CircuitBreaker: cb,
	}

	resp := d.server.ReleaseCircuitBreakers(d.defaultCtx, []*apiservice.ConfigRelease{release})
	if !respSuccess(resp) {
		t.Fatalf("error: %+v", resp)
	}
}

// 解绑熔断规则
func (d *DiscoverTestSuit) unBindCircuitBreaker(t *testing.T, cb *apifault.CircuitBreaker, service *apiservice.Service) {
	unbind := &apiservice.ConfigRelease{
		Service:        service,
		CircuitBreaker: cb,
	}

	resp := d.server.UnBindCircuitBreakers(d.defaultCtx, []*apiservice.ConfigRelease{unbind})
	if !respSuccess(resp) {
		t.Fatalf("error: %+v", resp)
	}
}

// 对比熔断规则的各个属性
func checkCircuitBreaker(t *testing.T, expect, expectMaster *apifault.CircuitBreaker, actual *apifault.CircuitBreaker) {
	switch {
	case expectMaster.GetId().GetValue() != actual.GetId().GetValue():
		t.Fatal("error id")
	case expect.GetVersion().GetValue() != actual.GetVersion().GetValue():
		t.Fatal("error version")
	case expectMaster.GetName().GetValue() != actual.GetName().GetValue():
		t.Fatal("error name")
	case expectMaster.GetNamespace().GetValue() != actual.GetNamespace().GetValue():
		t.Fatal("error namespace")
	case expectMaster.GetOwners().GetValue() != actual.GetOwners().GetValue():
		t.Fatal("error owners")
	case expectMaster.GetComment().GetValue() != actual.GetComment().GetValue():
		t.Fatal("error comment")
	case expectMaster.GetBusiness().GetValue() != actual.GetBusiness().GetValue():
		t.Fatal("error business")
	case expectMaster.GetDepartment().GetValue() != actual.GetDepartment().GetValue():
		t.Fatal("error department")
	default:
		break
	}

	expectInbounds, err := json.Marshal(expect.GetInbounds())
	if err != nil {
		panic(err)
	}
	inbounds, err := json.Marshal(actual.GetInbounds())
	if err != nil {
		panic(err)
	}
	if string(expectInbounds) != string(inbounds) {
		t.Fatal("error inbounds")
	}

	expectOutbounds, err := json.Marshal(expect.GetOutbounds())
	if err != nil {
		panic(err)
	}
	outbounds, err := json.Marshal(actual.GetOutbounds())
	if err != nil {
		panic(err)
	}
	if string(expectOutbounds) != string(outbounds) {
		t.Fatal("error inbounds")
	}
}

func buildCircuitBreakerKey(id, version string) string {
	return fmt.Sprintf("%s_%s", id, version)
}

// 彻底删除熔断规则
func (d *DiscoverTestSuit) cleanCircuitBreaker(id, version string) {
	log.Infof("clean circuit breaker, id: %s, version: %s", id, version)

	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

			str := `delete from circuitbreaker_rule where id = ? and version = ?`
			if _, err := dbTx.Exec(str, id, version); err != nil {
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

			if err := dbTx.Bucket([]byte(tblCircuitBreaker)).DeleteBucket([]byte(buildCircuitBreakerKey(id, version))); err != nil {
				if !errors.Is(err, bolt.ErrBucketNotFound) {
					panic(err)
				}
			}

			dbTx.Commit()
		}()
	}
}

// 彻底删除熔断规则发布记录
func (d *DiscoverTestSuit) cleanCircuitBreakerRelation(name, namespace, ruleID, ruleVersion string) {

	if d.storage.Name() == sqldb.STORENAME {
		func() {
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*sqldb.BaseTx)

			defer dbTx.Rollback()

			str := `delete from circuitbreaker_rule_relation using circuitbreaker_rule_relation, service where service_id = service.id and name = ? and namespace = ? and rule_id = ? and rule_version = ?`
			if _, err := dbTx.Exec(str, name, namespace, ruleID, ruleVersion); err != nil {
				panic(err)
			}

			dbTx.Commit()
		}()
	} else if d.storage.Name() == boltdb.STORENAME {
		func() {
			releations, err := d.storage.GetCircuitBreakerRelation(ruleID, ruleVersion)
			if err != nil {
				panic(err)
			}
			tx, err := d.storage.StartTx()
			if err != nil {
				panic(err)
			}

			dbTx := tx.GetDelegateTx().(*bolt.Tx)

			for i := range releations {
				if err := dbTx.Bucket([]byte(tblCircuitBreakerRelation)).DeleteBucket([]byte(releations[i].ServiceID)); err != nil {
					if !errors.Is(err, bolt.ErrBucketNotFound) {
						tx.Rollback()
						panic(err)
					}
				}
			}

			dbTx.Commit()
		}()
	}
}

// 获取指定长度str
func genSpecialStr(n int) string {
	str := ""
	for i := 0; i < n; i++ {
		str += "a"
	}
	return str
}

// 解析字符串sid为modID和cmdID
func parseStr2Sid(sid string) (uint32, uint32) {
	items := strings.Split(sid, ":")
	if len(items) != 2 {
		return 0, 0
	}

	mod, _ := strconv.ParseUint(items[0], 10, 32)
	cmd, _ := strconv.ParseUint(items[1], 10, 32)
	return uint32(mod), uint32(cmd)
}
