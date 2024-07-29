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
	"time"

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

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
		if err := m.loadByFile(loadFile); err != nil {
			return err
		}
	} else {
		if err := m.loadByDefault(); err != nil {
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

	mainUser = &authcommon.User{
		ID:          "65e4789a6d5b49669adf1e9e8387549c",
		Name:        "polaris",
		Password:    "$2a$10$3izWuZtE5SBdAtSZci.gs.iZ2pAn9I8hEqYrC6gwJp1dyjqQnrrum",
		Owner:       "",
		Source:      "Polaris",
		Mobile:      "",
		Email:       "",
		Type:        20,
		Token:       "nu/0WRA4EqSR1FagrjRj0fZwPXuGlMpX+zCuWu4uMqy8xr1vRjisSbA25aAC3mtU8MeeRsKhQiDAynUR09I=",
		TokenEnable: true,
		Valid:       true,
		Comment:     "default polaris admin account",
		CreateTime:  time.Now(),
		ModifyTime:  time.Now(),
	}

	superDefaultStrategy = &authcommon.StrategyDetail{
		ID:      "super_user_default_strategy",
		Name:    "(用户) polarissys@admin的默认策略",
		Action:  "READ_WRITE",
		Comment: "default admin",
		Principals: []authcommon.Principal{
			{
				StrategyID:    "super_user_default_strategy",
				PrincipalID:   "",
				PrincipalType: authcommon.PrincipalUser,
			},
		},
		Default: true,
		Owner:   "",
		Resources: []authcommon.StrategyResource{
			{
				StrategyID: "super_user_default_strategy",
				ResType:    int32(apisecurity.ResourceType_Namespaces),
				ResID:      "*",
			},
			{
				StrategyID: "super_user_default_strategy",
				ResType:    int32(apisecurity.ResourceType_Services),
				ResID:      "*",
			},
			{
				StrategyID: "super_user_default_strategy",
				ResType:    int32(apisecurity.ResourceType_ConfigGroups),
				ResID:      "*",
			},
		},
		Valid:      true,
		Revision:   "fbca9bfa04ae4ead86e1ecf5811e32a9",
		CreateTime: time.Now(),
		ModifyTime: time.Now(),
	}

	mainDefaultStrategy = &authcommon.StrategyDetail{
		ID:      "fbca9bfa04ae4ead86e1ecf5811e32a9",
		Name:    "(用户) polaris的默认策略",
		Action:  "READ_WRITE",
		Comment: "default admin",
		Principals: []authcommon.Principal{
			{
				StrategyID:    "fbca9bfa04ae4ead86e1ecf5811e32a9",
				PrincipalID:   "65e4789a6d5b49669adf1e9e8387549c",
				PrincipalType: authcommon.PrincipalUser,
			},
		},
		Default: true,
		Owner:   "65e4789a6d5b49669adf1e9e8387549c",
		Resources: []authcommon.StrategyResource{
			{
				StrategyID: "fbca9bfa04ae4ead86e1ecf5811e32a9",
				ResType:    int32(apisecurity.ResourceType_Namespaces),
				ResID:      "*",
			},
			{
				StrategyID: "fbca9bfa04ae4ead86e1ecf5811e32a9",
				ResType:    int32(apisecurity.ResourceType_Services),
				ResID:      "*",
			},
			{
				StrategyID: "fbca9bfa04ae4ead86e1ecf5811e32a9",
				ResType:    int32(apisecurity.ResourceType_ConfigGroups),
				ResID:      "*",
			},
		},
		Valid:      true,
		Revision:   "fbca9bfa04ae4ead86e1ecf5811e32a9",
		CreateTime: time.Now(),
		ModifyTime: time.Now(),
	}
)

func (m *boltStore) initNamingStoreData() error {
	for _, namespace := range namespacesToInit {
		curTime := time.Now()
		err := m.AddNamespace(&model.Namespace{
			Name:       namespace,
			Token:      utils.NewUUID(),
			Owner:      ownerToInit,
			Valid:      true,
			CreateTime: curTime,
			ModifyTime: curTime,
		})
		if err != nil {
			return err
		}
	}
	for svc, id := range servicesToInit {
		curTime := time.Now()
		err := m.AddService(&model.Service{
			ID:         id,
			Name:       svc,
			Namespace:  namespacePolaris,
			Token:      utils.NewUUID(),
			Owner:      ownerToInit,
			Revision:   utils.NewUUID(),
			Valid:      true,
			CreateTime: curTime,
			ModifyTime: curTime,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *boltStore) initAuthStoreData() error {
	return m.handler.Execute(true, func(tx *bolt.Tx) error {
		user, err := m.getUser(tx, mainUser.ID)
		if err != nil {
			return err
		}

		if user == nil {
			user = mainUser
			// 添加主账户主体信息
			if err := saveValue(tx, tblUser, user.ID, converToUserStore(user)); err != nil {
				authLog.Error("[Store][User] save user fail", zap.Error(err), zap.String("name", user.Name))
				return err
			}
		}

		rule, err := m.getStrategyDetail(tx, mainDefaultStrategy.ID)
		if err != nil {
			return err
		}

		if rule == nil {
			strategy := mainDefaultStrategy
			// 添加主账户的默认鉴权策略信息
			if err := saveValue(tx, tblStrategy, strategy.ID, convertForStrategyStore(strategy)); err != nil {
				authLog.Error("[Store][Strategy] save auth_strategy", zap.Error(err),
					zap.String("name", strategy.Name), zap.String("owner", strategy.Owner))
				return err
			}
		}
		return nil
	})
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
