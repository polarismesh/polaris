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
	"math"
	"sync"
	"time"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

func init() {
	RegisterCache(StrategyRuleName, CacheAuthStrategy)
}

const (
	StrategyRuleName string = "strategyRule"
)

// StrategyCache
type StrategyCache interface {
	Cache

	Listener

	// GetStrategyDetailsByUID
	//  @param uid
	//  @return []*model.StrategyDetail
	GetStrategyDetailsByUID(uid string) []*model.StrategyDetail

	// GetStrategyDetailsByGroupID
	//  @param uid
	//  @return []*model.StrategyDetail
	GetStrategyDetailsByGroupID(uid string) []*model.StrategyDetail

	// IsResourceLinkStrategy 该资源是否关联了鉴权策略
	IsResourceLinkStrategy(resType api.ResourceType, resId string) bool
}

// strategyCache
type strategyCache struct {
	storage          store.Store
	strategys        *sync.Map
	uid2Strategy     *sync.Map
	groupid2Strategy *sync.Map

	namespace2Strategy   *sync.Map
	service2Strategy     *sync.Map
	configGroup2Strategy *sync.Map

	firstUpdate    bool
	lastUpdateTime int64

	singleFlight *singleflight.Group

	principalCh chan interface{}
}

// newStrategyCache
//  @param storage
//  @return StrategyCache
func newStrategyCache(storage store.Store) StrategyCache {
	return &strategyCache{
		storage: storage,
	}
}

func (sc *strategyCache) initialize(c map[string]interface{}) error {
	sc.strategys = new(sync.Map)
	sc.uid2Strategy = new(sync.Map)
	sc.groupid2Strategy = new(sync.Map)

	sc.namespace2Strategy = new(sync.Map)
	sc.service2Strategy = new(sync.Map)
	sc.configGroup2Strategy = new(sync.Map)

	sc.principalCh = make(chan interface{}, 32)
	sc.singleFlight = new(singleflight.Group)
	sc.firstUpdate = true
	sc.lastUpdateTime = 0
	return nil
}

func (sc *strategyCache) update() error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := sc.singleFlight.Do(StrategyRuleName, func() (interface{}, error) {
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
		log.CacheScope().Errorf("[Cache][AuthStrategy] refresh auth strategy cache err: %s", err.Error())
		return err
	}

	sc.firstUpdate = false
	add, update, del := sc.setStrategys(strategys)
	log.CacheScope().Debug("[Cache][AuthStrategy] get more auth strategy", zap.Int("add", add), zap.Int("update", update), zap.Int("delete", del),
		zap.Time("last", lastMtime), zap.Duration("used", time.Now().Sub(start)))
	return nil
}

func (sc *strategyCache) setStrategys(strategies []*model.StrategyDetail) (int, int, int) {

	var (
		add    int
		remove int
		update int
	)

	for index := range strategies {
		rule := strategies[index]
		if !rule.Valid {
			sc.strategys.Delete(rule.ID)
			remove++

			principals := rule.Principals
			for pos := range principals {
				principal := principals[pos]

				if principal.PrincipalRole == model.PrincipalUser {
					if val, ok := sc.uid2Strategy.Load(principal.PrincipalID); ok {
						val.(*sync.Map).Delete(rule.ID)
					}
				} else {
					if val, ok := sc.groupid2Strategy.Load(principal.PrincipalID); ok {
						val.(*sync.Map).Delete(rule.ID)
					}
				}
			}

		} else {
			_, ok := sc.strategys.Load(rule.ID)
			if !ok {
				add++
			} else {
				update++
			}
			sc.strategys.Store(rule.ID, rule)

			sc.lastUpdateTime = int64(math.Max(float64(sc.lastUpdateTime), float64(rule.ModifyTime.Unix())))
		}
	}

	sc.handlerResourceStrategy(strategies)
	sc.handlerPrincipalStrategy(strategies)
	sc.postProcessPrincipalCh()

	return add, update, remove
}

// handlerResourceStrategy
func (sc *strategyCache) handlerResourceStrategy(strategies []*model.StrategyDetail) {
	for index := range strategies {
		rule := strategies[index]
		if rule.Valid {
			resources := rule.Resources
			for index := range resources {
				resource := resources[index]

				switch resource.ResType {
				case int32(api.ResourceType_Namespaces):
					val, _ := sc.namespace2Strategy.LoadOrStore(resource.ResID, new(sync.Map))
					val.(*sync.Map).Store(rule.ID, struct{}{})
				case int32(api.ResourceType_Services):
					val, _ := sc.service2Strategy.LoadOrStore(resource.ResID, new(sync.Map))
					val.(*sync.Map).Store(rule.ID, struct{}{})
				case int32(api.ResourceType_ConfigGroups):
					val, _ := sc.configGroup2Strategy.LoadOrStore(resource.ResID, new(sync.Map))
					val.(*sync.Map).Store(rule.ID, struct{}{})
				}
			}
		}
	}
}

// handlerPrincipalStrategy
func (sc *strategyCache) handlerPrincipalStrategy(strategies []*model.StrategyDetail) {
	for index := range strategies {
		rule := strategies[index]
		if rule.Valid {
			// 计算 uid -> auth rule
			principals := rule.Principals
			for pos := range principals {
				principal := principals[pos]

				var rulesMap *sync.Map

				if principal.PrincipalRole == model.PrincipalUser {
					sc.uid2Strategy.LoadOrStore(principal.PrincipalID, new(sync.Map))
					val, _ := sc.uid2Strategy.Load(principal.PrincipalID)
					rulesMap = val.(*sync.Map)
				} else {
					sc.groupid2Strategy.LoadOrStore(principal.PrincipalID, new(sync.Map))
					val, _ := sc.groupid2Strategy.Load(principal.PrincipalID)
					rulesMap = val.(*sync.Map)
				}

				rulesMap.Store(rule.ID, rule)
			}
		}
	}
}

func (sc *strategyCache) postProcessPrincipalCh() {
	select {
	case event := <-sc.principalCh:
		principals := event.([]model.Principal)
		for index := range principals {
			principal := principals[index]

			if principal.PrincipalRole == model.PrincipalUser {
				sc.uid2Strategy.Delete(principal.PrincipalID)
			} else {
				sc.groupid2Strategy.Delete(principal.PrincipalID)
			}
		}
	case <-time.After(time.Duration(100 * time.Millisecond)):
		return
	}
}

func (sc *strategyCache) clear() error {
	sc.strategys = new(sync.Map)
	sc.uid2Strategy = new(sync.Map)
	sc.groupid2Strategy = new(sync.Map)

	sc.namespace2Strategy = new(sync.Map)
	sc.service2Strategy = new(sync.Map)
	sc.configGroup2Strategy = new(sync.Map)

	sc.firstUpdate = true
	sc.lastUpdateTime = 0
	return nil
}

func (sc *strategyCache) name() string {
	return StrategyRuleName
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

// IsResourceLinkStrategy
func (sc *strategyCache) IsResourceLinkStrategy(resType api.ResourceType, resId string) bool {
	switch resType {
	case api.ResourceType_Namespaces:
		_, ok := sc.namespace2Strategy.Load(resId)
		return ok
	case api.ResourceType_Services:
		_, ok := sc.service2Strategy.Load(resId)
		return ok
	case api.ResourceType_ConfigGroups:
		_, ok := sc.configGroup2Strategy.Load(resId)
		return ok
	default:
		return true
	}
}

// OnCreated callback when cache value created
func (sc *strategyCache) OnCreated(value interface{}) {

}

// OnUpdated callback when cache value updated
func (sc *strategyCache) OnUpdated(value interface{}) {

}

// OnDeleted callback when cache value deleted
func (sc *strategyCache) OnDeleted(value interface{}) {

}

// OnBatchCreated callback when cache value created
func (sc *strategyCache) OnBatchCreated(value interface{}) {

}

// OnBatchUpdated callback when cache value updated
func (sc *strategyCache) OnBatchUpdated(value interface{}) {

}

// OnBatchDeleted callback when cache value deleted
func (sc *strategyCache) OnBatchDeleted(value interface{}) {
	if principals, ok := value.([]model.Principal); ok {
		sc.principalCh <- principals
	}
}
