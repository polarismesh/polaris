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

package sqldb

import (
	"errors"
	"fmt"
	"github.com/polarismesh/polaris-server/plugin"

	"github.com/polarismesh/polaris-server/store"

	// 使用mysql库
	_ "github.com/go-sql-driver/mysql"
)

const (
	SystemNamespace        = "Polaris"
	STORENAME              = "defaultStore"
	DefaultConnMaxLifetime = 60 * 30 // 默认是30分钟
)

/**
 * @brief 自动引入包初始化函数
 */
func init() {
	s := &stableStore{}
	_ = store.RegisterStore(s)
}

/**
 * @brief 实现了Store接口
 */
type stableStore struct {
	*namespaceStore
	*businessStore
	*serviceStore
	*instanceStore
	*routingConfigStore
	*l5Store
	*rateLimitStore
	*circuitBreakerStore
	*platformStore
	*toolStore

	//配置中心stores
	*configFileGroupStore
	*configFileStore
	*configFileReleaseStore
	*configFileReleaseHistoryStore
	*configFileTagStore

	// 主数据库，可以进行读写
	master *BaseDB
	// 对主数据库的事务操作，可读写
	masterTx *BaseDB
	// 备数据库，提供只读
	slave    *BaseDB
	start    bool
	metaTask *TaskManager
}

/**
 * @brief 实现Name函数
 */
func (s *stableStore) Name() string {
	return STORENAME
}

/**
 * @brief 初始化函数
 */
func (s *stableStore) Initialize(conf *store.Config) error {
	if s.start {
		return errors.New("store has been Initialize")
	}

	masterConfig, slaveConfig, err := parseDatabaseConf(conf.Option)
	if err != nil {
		return err
	}
	master, err := NewBaseDB(masterConfig, plugin.GetParsePassword())
	if err != nil {
		return err
	}
	s.master = master

	masterTx, err := NewBaseDB(masterConfig, plugin.GetParsePassword())
	if err != nil {
		return err
	}
	s.masterTx = masterTx

	if slaveConfig != nil {
		log.Infof("[Store][database] use slave database config: %+v", slaveConfig)
		slave, err := NewBaseDB(slaveConfig, plugin.GetParsePassword())
		if err != nil {
			return err
		}
		s.slave = slave
	}
	// 如果slave为空，意味着slaveConfig为空，用master数据库替代
	if s.slave == nil {
		s.slave = s.master
	}

	log.Infof("[Store][database] connect the database successfully")

	s.start = true
	s.newStore()
	return nil
}

// return slave, master, error
func parseDatabaseConf(opt map[string]interface{}) (*dbConfig, *dbConfig, error) {
	// 必填
	masterEnter, ok := opt["master"]
	if !ok || masterEnter == nil {
		return nil, nil, errors.New("database master db config is missing")
	}
	masterConfig, err := parseStoreConfig(masterEnter)
	if err != nil {
		return nil, nil, err
	}

	// 只读数据库可选
	slaveEntry, ok := opt["slave"]
	if !ok || slaveEntry == nil {
		return masterConfig, nil, nil
	}
	slaveConfig, err := parseStoreConfig(slaveEntry)
	if err != nil {
		return nil, nil, err
	}

	return masterConfig, slaveConfig, nil
}

// 解析store的配置
func parseStoreConfig(opts interface{}) (*dbConfig, error) {
	obj, ok := opts.(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("database config is error")
	}
	dbType, _ := obj["dbType"].(string)
	dbUser, _ := obj["dbUser"].(string)
	dbPwd, _ := obj["dbPwd"].(string)
	dbAddr, _ := obj["dbAddr"].(string)
	dbName, _ := obj["dbName"].(string)
	if dbType == "" || dbUser == "" || dbPwd == "" || dbAddr == "" || dbName == "" {
		return nil, fmt.Errorf("Config Plugin %s missing database param", STORENAME)
	}

	c := &dbConfig{
		dbType: dbType,
		dbUser: dbUser,
		dbPwd:  dbPwd,
		dbAddr: dbAddr,
		dbName: dbName,
	}
	if maxOpenConns, _ := obj["maxOpenConns"].(int); maxOpenConns > 0 {
		c.maxOpenConns = maxOpenConns
	}
	if maxIdleConns, _ := obj["maxIdleConns"].(int); maxIdleConns > 0 {
		c.maxIdleConns = maxIdleConns
	}
	c.connMaxLifetime = DefaultConnMaxLifetime
	if connMaxLifetime, _ := obj["connMaxLifetime"].(int); connMaxLifetime > 0 {
		c.connMaxLifetime = connMaxLifetime
	}

	if isolationLevel, _ := obj["txIsolationLevel"].(int); isolationLevel > 0 {
		c.txIsolationLevel = isolationLevel
	}
	return c, nil
}

/**
 * @brief 退出函数
 */
func (s *stableStore) Destroy() error {
	if s.master != nil {
		_ = s.master.Close()
	}
	if s.masterTx != nil {
		_ = s.masterTx.Close()
	}
	if s.slave != nil {
		_ = s.slave.Close()
	}

	return nil
}

/**
 * @brief 创建一个事务
 */
func (s *stableStore) CreateTransaction() (store.Transaction, error) {
	// 每次创建事务前，还是需要ping一下
	_ = s.masterTx.Ping()

	nt := &transaction{}
	tx, err := s.masterTx.Begin()
	if err != nil {
		log.Errorf("[Store][database] database begin err: %s", err.Error())
		return nil, err
	}

	nt.tx = tx
	return nt, nil
}

// 初始化子类
func (s *stableStore) newStore() {
	s.namespaceStore = &namespaceStore{db: s.master}

	s.businessStore = &businessStore{db: s.master}

	s.serviceStore = &serviceStore{master: s.master, slave: s.slave}

	s.instanceStore = &instanceStore{master: s.master, slave: s.slave}

	s.routingConfigStore = &routingConfigStore{master: s.master, slave: s.slave}

	s.l5Store = &l5Store{db: s.master}

	s.rateLimitStore = &rateLimitStore{db: s.master}

	s.circuitBreakerStore = &circuitBreakerStore{master: s.master, slave: s.slave}

	s.platformStore = &platformStore{master: s.master}

	s.toolStore = &toolStore{db: s.master}

	s.configFileGroupStore = &configFileGroupStore{db: s.master}

	s.configFileStore = &configFileStore{db: s.master}

	s.configFileReleaseStore = &configFileReleaseStore{db: s.master}

	s.configFileReleaseHistoryStore = &configFileReleaseHistoryStore{db: s.master}

	s.configFileTagStore = &configFileTagStore{db: s.master}
}
