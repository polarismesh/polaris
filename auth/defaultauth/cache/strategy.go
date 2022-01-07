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

package cache

import (
	"sync"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

// StrategyCache
type StrategyCache interface {
	Cache

	// GetStrategyDetailsByUID
	//  @param uid
	//  @return []*model.StrategyDetail
	GetStrategyDetailsByUID(uid string) []*model.StrategyDetail

	// GetStrategyDetailsByGroupID
	//  @param uid
	//  @return []*model.StrategyDetail
	GetStrategyDetailsByGroupID(uid string) []*model.StrategyDetail
}

// strategyCache
type strategyCache struct {
	storage          store.Store
	strategys        *sync.Map
	uid2Strategy     *sync.Map
	groupid2Strategy *sync.Map

	firstUpdate    bool
	lastUpdateTime int64

	singleFlight *singleflight.Group
}

// newStrategyCache
//  @param storage
//  @return StrategyCache
func newStrategyCache(storage store.Store) StrategyCache {
	return &strategyCache{
		storage: storage,
	}
}

func (sc *strategyCache) initialize() error {
	sc.strategys = new(sync.Map)
	sc.uid2Strategy = new(sync.Map)
	sc.groupid2Strategy = new(sync.Map)

	sc.singleFlight = new(singleflight.Group)
	sc.firstUpdate = true
	sc.lastUpdateTime = 0
	return nil
}

func (sc *strategyCache) update() error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := sc.singleFlight.Do(CacheForUser, func() (interface{}, error) {
		return nil, sc.realUpdate()
	})
	return err
}

func (sc *strategyCache) realUpdate() error {
	// 获取几秒前的全部数据
	start := time.Now()
	lastMtime := time.Unix(sc.lastUpdateTime, 0)
	strategys, err := sc.storage.GetStrategyDetailsForCache(lastMtime.Add(DefaultTimeDiff), sc.firstUpdate)
	if err != nil {
		log.Errorf("[Cache][AuthStrategy] update auth strategy err: %s", err.Error())
		return err
	}

	sc.firstUpdate = false
	update, del := sc.setStrategys(strategys)
	log.Debug("[Cache][User] get more services", zap.Int("update", update), zap.Int("delete", del),
		zap.Time("last", lastMtime), zap.Duration("used", time.Now().Sub(start)))
	return nil
}

func (sc *strategyCache) setStrategys(strategys []*model.StrategyDetail) (int, int) {
	return 0, 0
}

func (sc *strategyCache) clear() error {
	sc.strategys = new(sync.Map)
	sc.uid2Strategy = new(sync.Map)
	sc.groupid2Strategy = new(sync.Map)

	sc.firstUpdate = true
	sc.lastUpdateTime = 0
	return nil
}

func (sc *strategyCache) name() string {
	return CacheForStrategy
}

func (sc *strategyCache) GetStrategyDetailsByUID(uid string) []*model.StrategyDetail {

	return sc.getStrategyDetails(uid, "")
}

func (sc *strategyCache) GetStrategyDetailsByGroupID(groupid string) []*model.StrategyDetail {

	return sc.getStrategyDetails("", groupid)
}

func (sc *strategyCache) getStrategyDetails(uid string, gid string) []*model.StrategyDetail {
	var strategyIds *sync.Map
	if uid != "" {
		val, ok := sc.uid2Strategy.Load(uid)
		if !ok {
			return nil
		}
		strategyIds = val.(*sync.Map)
	} else if gid != "" {
		val, ok := sc.groupid2Strategy.Load(uid)
		if !ok {
			return nil
		}
		strategyIds = val.(*sync.Map)
	}

	if strategyIds != nil {
		result := make([]*model.StrategyDetail, 0, 16)
		strategyIds.Range(func(key, value interface{}) bool {
			strategy, ok := sc.strategys.Load(key)
			if ok {
				result = append(result, strategy.(*model.StrategyDetail))
			}
			return true
		})

		return result
	}

	return nil
}
