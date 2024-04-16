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
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const (
	maxAliveDuration   = time.Minute
	faultAliveDuration = 5 * time.Second
)

func NewServiceContractCache(storage store.Store, cacheMgr types.CacheManager) types.ServiceContractCache {
	return &serviceContractCache{
		lastRunTime:      time.Now(),
		lastStoreErrTime: 0,
		BaseCache:        types.NewBaseCache(storage, cacheMgr),
	}
}

type serviceContractCache struct {
	*types.BaseCache

	lastRunTime      time.Time
	lastStoreErrTime int64
	lastMtimeLogged  int64

	// data namespace/service/name/protocol/version -> *model.EnrichServiceContract
	data        *utils.SyncMap[string, *types.ExpireEntry[*model.EnrichServiceContract]]
	singleGroup *singleflight.Group
}

// Initialize .
func (sc *serviceContractCache) Initialize(c map[string]interface{}) error {
	sc.lastRunTime = time.Now()
	sc.singleGroup = &singleflight.Group{}
	sc.data = utils.NewSyncMap[string, *types.ExpireEntry[*model.EnrichServiceContract]]()
	return nil
}

// Update .
func (sc *serviceContractCache) Update() error {
	if time.Since(sc.lastRunTime) < time.Minute {
		return nil
	}
	sc.lastRunTime = time.Now()

	lastStoreErrTime := atomic.LoadInt64(&sc.lastStoreErrTime)
	// 如果存储层在近 1min 中内发生错误，则不会清理 expire 的数据
	showSkip := time.Now().Unix()-lastStoreErrTime < 60
	log.Info("[ServiceContract] cache expire entry clean start")
	waitDel := make([]string, 0, 4)
	sc.data.ReadRange(func(key string, val *types.ExpireEntry[*model.EnrichServiceContract]) {
		if showSkip {
			return
		}
		if val.IsExpire() {
			waitDel = append(waitDel, key)
		}
	})
	for i := range waitDel {
		sc.data.Delete(waitDel[i])
		log.Info("[ServiceContract] cache expire entry", zap.String("key", waitDel[i]))
	}
	return nil
}

// Clear .
func (sc *serviceContractCache) Clear() error {
	sc.data = utils.NewSyncMap[string, *types.ExpireEntry[*model.EnrichServiceContract]]()
	return nil
}

// Name .
func (sc *serviceContractCache) Name() string {
	return types.ServiceContractName
}

func (sc *serviceContractCache) Get(ctx context.Context, req *model.ServiceContract) *model.EnrichServiceContract {
	ret, _ := sc.data.ComputeIfAbsent(req.GetCacheKey(), func(k string) *types.ExpireEntry[*model.EnrichServiceContract] {
		id, err := utils.CalculateContractID(req.Namespace, req.Service, req.Type, req.Protocol, req.Version)
		if err != nil {
			return types.EmptyExpireEntry(&model.EnrichServiceContract{}, faultAliveDuration)
		}
		val, err := sc.Store().GetServiceContract(id)
		if err != nil {
			atomic.StoreInt64(&sc.lastStoreErrTime, time.Now().Unix())
			return types.EmptyExpireEntry(&model.EnrichServiceContract{}, faultAliveDuration)
		}
		return types.NewExpireEntry(val, maxAliveDuration)
	})

	return ret.Get()
}
