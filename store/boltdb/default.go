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
	"errors"
	"time"

	"github.com/boltdb/bolt"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
)

const (
	storeName = "boltdbStore"
)

type boltStore struct {
	*namespaceStore
	*businessStore
	*serviceStore
	*instanceStore
	*l5Store
	*routingStore
	*rateLimitStore
	*platformStore
	*circuitBreakerStore
	*toolStore
	*userStore
	*groupStore
	*strategyStore

	//配置中心stores
	*configFileGroupStore
	*configFileStore
	*configFileReleaseStore
	*configFileReleaseHistoryStore
	*configFileTagStore

	handler BoltHandler
	start   bool
}

// Name store name
func (m *boltStore) Name() string {
	return storeName
}

// Initialize init store
func (m *boltStore) Initialize(c *store.Config) error {
	if m.start {
		return errors.New("store has been Initialize")
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
	if err = m.initNamingStoreData(); err != nil {
		_ = handler.Close()
		return err
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
	servicesToInit   = map[string]string{"polaris.checker": "fbca9bfa04ae4ead86e1ecf5811e32a9", "polaris.monitor": "bbfdda174ea64e11ac862adf14593c03"}

	adminUser = &model.User{
		ID:          "65e4789a6d5b49669adf1e9e8387549c",
		Name:        "polarisadmin",
		Password:    "$2a$10$5XMjs.oqo4PnpbTGy9dQqewL4eb4yoA7b/6ZKL33IPhFyIxzj4lRy",
		Owner:       "",
		Source:      "Polaris",
		Mobile:      "",
		Email:       "",
		Type:        0,
		Token:       "nu/0WRA4EqSR1FagrjRj0fZwPXuGlMpX+zCuWu4uMqy8xr1vRjisSbA25aAC3mtU8MeeRsKhQiDAynUR09I=",
		TokenEnable: true,
		Valid:       true,
		Comment:     "default polaris admin account",
		CreateTime:  time.Now(),
		ModifyTime:  time.Now(),
	}

	adminDefaultStrategy = &model.StrategyDetail{
		ID:      "fbca9bfa04ae4ead86e1ecf5811e32a9",
		Name:    "(用户) PolarisAdmin的默认策略",
		Action:  "READ_WRITE",
		Comment: "default admin",
		Principals: []model.Principal{
			{
				StrategyID:    "fbca9bfa04ae4ead86e1ecf5811e32a9",
				PrincipalID:   "65e4789a6d5b49669adf1e9e8387549c",
				PrincipalRole: model.PrincipalUser,
			},
		},
		Default: true,
		Owner:   "65e4789a6d5b49669adf1e9e8387549c",
		Resources: []model.StrategyResource{
			{
				StrategyID: "fbca9bfa04ae4ead86e1ecf5811e32a9",
				ResType:    int32(api.ResourceType_Namespaces),
				ResID:      "*",
			},
			{
				StrategyID: "fbca9bfa04ae4ead86e1ecf5811e32a9",
				ResType:    int32(api.ResourceType_Services),
				ResID:      "*",
			},
			{
				StrategyID: "fbca9bfa04ae4ead86e1ecf5811e32a9",
				ResType:    int32(api.ResourceType_ConfigGroups),
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
	if err := m.handler.Execute(true, func(tx *bolt.Tx) error {
		// 添加管理主体信息
		if err := m.addUserMain(tx, adminUser); err != nil {
			return err
		}

		// 添加管理员的默认鉴权策略信息
		return m.addStrategy(tx, adminDefaultStrategy)
	}); err != nil {
		return err
	}
	return nil
}

func (m *boltStore) newStore() error {
	m.l5Store = &l5Store{handler: m.handler}
	if err := m.l5Store.InitL5Data(); err != nil {
		return err
	}

	m.namespaceStore = &namespaceStore{handler: m.handler}
	if err := m.namespaceStore.InitData(); err != nil {
		return err
	}
	m.businessStore = &businessStore{handler: m.handler}

	m.serviceStore = &serviceStore{handler: m.handler}

	m.instanceStore = &instanceStore{handler: m.handler}

	m.routingStore = &routingStore{handler: m.handler}

	m.rateLimitStore = &rateLimitStore{handler: m.handler}

	m.circuitBreakerStore = &circuitBreakerStore{handler: m.handler}

	m.platformStore = &platformStore{handler: m.handler}

	m.userStore = &userStore{handler: m.handler}

	m.strategyStore = &strategyStore{handler: m.handler}

	m.groupStore = &groupStore{handler: m.handler}

	return nil
}

// Destroy destroy store
func (m *boltStore) Destroy() error {
	if m.handler != nil {
		return m.handler.Close()
	}
	return nil
}

//CreateTransaction create store transaction
func (m *boltStore) CreateTransaction() (store.Transaction, error) {
	return &transaction{handler: m.handler}, nil
}

func (m *boltStore) StartTx() (store.Tx, error) {
	return m.handler.StartTx()
}

func init() {
	s := &boltStore{}
	_ = store.RegisterStore(s)
}
