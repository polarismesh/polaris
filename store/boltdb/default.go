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

package boltdb

import (
	"os"
	"time"

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"

	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const (

	// SystemNamespace system namespace
	SystemNamespace = "Polaris"
	// STORENAME database storage name
	STORENAME = "boltdbStore"
	// DefaultConnMaxLifetime default maximum connection lifetime
	DefaultConnMaxLifetime = 60 * 30 // 默认是30分钟
)

const (
	svcSpecificQueryKeyService   = "service"
	svcSpecificQueryKeyNamespace = "serviceNamespace"
	exactName                    = "exactName"
	excludeId                    = "excludeId"
)

const (
	CommonFieldValid       = "Valid"
	CommonFieldEnableTime  = "EnableTime"
	CommonFieldModifyTime  = "ModifyTime"
	CommonFieldRevision    = "Revision"
	CommonFieldID          = "ID"
	CommonFieldName        = "Name"
	CommonFieldNamespace   = "Namespace"
	CommonFieldDescription = "Description"
	CommonFieldEnable      = "Enable"
	CommonFieldMetadata    = "Metadata"
)

type boltStore struct {
	*namespaceStore
	*clientStore

	// 服务注册发现、治理
	*serviceStore
	*instanceStore
	*l5Store
	*routingStore
	*rateLimitStore
	*circuitBreakerStore
	*faultDetectStore
	*routingStoreV2
	*serviceContractStore
	*laneStore

	// 配置中心stores
	*configFileGroupStore
	*configFileStore
	*configFileReleaseStore
	*configFileReleaseHistoryStore
	*configFileTemplateStore

	*grayStore

	// adminStore store
	*adminStore
	// 工具
	*toolStore
	// 鉴权模块相关
	*userStore
	*groupStore
	*strategyStore
	*roleStore

	handler BoltHandler
	start   bool
}

// Name store name
func (m *boltStore) Name() string {
	return STORENAME
}

// Initialize init store
func (m *boltStore) Initialize(c *store.Config) error {
	if m.start {
		return nil
	}
	boltConfig := &BoltConfig{}
	boltConfig.Parse(c.Option)
	handler, err := NewBoltHandler(boltConfig)
	if err != nil {
		return err
	}
	m.handler = handler
	if err = m.newStore(); err != nil {
		_ = handler.Close()
		return err
	}

	if loadFile, ok := c.Option["loadFile"].(string); ok {
		// 仅用于本地测试验证单机数据
		loadFileName := os.Getenv("POLARIS_DEV_BOLT_INIT_DATA_FILA")
		if loadFileName != "" {
			loadFile = loadFileName
		}
		if err := m.loadByFile(loadFile); err != nil {
			return err
		}
	}
	m.start = true
	return nil
}

const (
	namespacePolaris = "Polaris"
	ownerToInit      = "polaris"
)

var (
	namespacesToInit = []string{"default", namespacePolaris}
	servicesToInit   = map[string]string{
		"polaris.checker": "fbca9bfa04ae4ead86e1ecf5811e32a9",
	}
)

func (m *boltStore) initNamingStoreData() error {
	for _, namespace := range namespacesToInit {
		curTime := time.Now()
		val, err := m.GetNamespace(namespace)
		if err != nil {
			return err
		}
		if val == nil {
			if err := m.AddNamespace(&model.Namespace{
				Name:       namespace,
				Token:      utils.NewUUID(),
				Owner:      ownerToInit,
				Valid:      true,
				CreateTime: curTime,
				ModifyTime: curTime,
			}); err != nil {
				return err
			}
		}
	}
	for svc, id := range servicesToInit {
		curTime := time.Now()
		val, err := m.getServiceByNameAndNs(svc, namespacePolaris)
		if err != nil {
			return err
		}
		if val != nil {
			if err := m.AddService(&model.Service{
				ID:         id,
				Name:       svc,
				Namespace:  namespacePolaris,
				Token:      utils.NewUUID(),
				Owner:      ownerToInit,
				Revision:   utils.NewUUID(),
				Valid:      true,
				CreateTime: curTime,
				ModifyTime: curTime,
			}); err != nil {
				return err
			}
		}

	}
	return nil
}

func (m *boltStore) newStore() error {
	var err error

	m.l5Store = &l5Store{handler: m.handler}
	if err = m.l5Store.InitL5Data(); err != nil {
		return err
	}
	m.namespaceStore = &namespaceStore{handler: m.handler}
	if err = m.namespaceStore.InitData(); err != nil {
		return err
	}
	m.clientStore = &clientStore{handler: m.handler}
	m.grayStore = &grayStore{handler: m.handler}
	m.newDiscoverModuleStore()
	m.newAuthModuleStore()
	m.newConfigModuleStore()
	m.newMaintainModuleStore()
	return nil
}

func (m *boltStore) newDiscoverModuleStore() {
	m.serviceStore = &serviceStore{handler: m.handler}
	m.instanceStore = &instanceStore{handler: m.handler}
	m.routingStore = &routingStore{handler: m.handler}
	m.rateLimitStore = &rateLimitStore{handler: m.handler}
	m.circuitBreakerStore = &circuitBreakerStore{handler: m.handler}
	m.faultDetectStore = &faultDetectStore{handler: m.handler}
	m.routingStoreV2 = &routingStoreV2{handler: m.handler}
	m.serviceContractStore = &serviceContractStore{handler: m.handler}
	m.laneStore = &laneStore{handler: m.handler}
}

func (m *boltStore) newAuthModuleStore() {
	m.userStore = &userStore{handler: m.handler}
	m.strategyStore = &strategyStore{handler: m.handler}
	m.groupStore = &groupStore{handler: m.handler}
	m.roleStore = &roleStore{handle: m.handler}
}

func (m *boltStore) newConfigModuleStore() {
	m.configFileStore = newConfigFileStore(m.handler)
	m.configFileGroupStore = newConfigFileGroupStore(m.handler)
	m.configFileReleaseHistoryStore = newConfigFileReleaseHistoryStore(m.handler)
	m.configFileReleaseStore = newConfigFileReleaseStore(m.handler)
	m.configFileTemplateStore = newConfigFileTemplateStore(m.handler)
}

func (m *boltStore) newMaintainModuleStore() {
	m.adminStore = &adminStore{handler: m.handler, leMap: make(map[string]bool)}
}

// Destroy store
func (m *boltStore) Destroy() error {
	m.start = false
	if m.handler != nil {
		return m.handler.Close()
	}
	return nil
}

// CreateTransaction create store transaction
func (m *boltStore) CreateTransaction() (store.Transaction, error) {
	return &transaction{handler: m.handler}, nil
}

// StartTx starting transactions
func (m *boltStore) StartTx() (store.Tx, error) {
	return m.handler.StartTx()
}

func (m *boltStore) StartReadTx() (store.Tx, error) {
	return m.handler.StartTx()
}

func init() {
	s := &boltStore{}
	_ = store.RegisterStore(s)
}

func buildAllResAllow(id string) []authcommon.StrategyResource {
	ret := make([]authcommon.StrategyResource, 0, 8)
	for i := range apisecurity.ResourceType_value {
		ret = append(ret, authcommon.StrategyResource{
			StrategyID: id,
			ResType:    apisecurity.ResourceType_value[i],
			ResID:      "*",
		})
	}
	return ret
}
