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
	"fmt"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
)

func init() {
	RegisterCache(StrategyRuleName, CacheAuthStrategy)
}

const (
	StrategyRuleName string = "strategyRule"
)

// StrategyCache is a cache for strategy rules.
type StrategyCache interface {
	Cache

	// GetStrategyDetailsByUID
	//  @param uid
	//  @return []*model.StrategyDetail
	GetStrategyDetailsByUID(uid string) []*model.StrategyDetail

	// GetStrategyDetailsByGroupID returns all strategy details of a group.
	GetStrategyDetailsByGroupID(groupId string) []*model.StrategyDetail

	// IsResourceLinkStrategy 该资源是否关联了鉴权策略
	IsResourceLinkStrategy(resType api.ResourceType, resId string) bool

	// IsResourceEditable 判断该资源是否可以操作
	IsResourceEditable(principal model.Principal, resType api.ResourceType, resId string) bool
}

// strategyCache
type strategyCache struct {
	*baseCache

	storage          store.Store
	strategys        *strategyBucket
	uid2Strategy     *strategyLinkBucket
	groupid2Strategy *strategyLinkBucket

	namespace2Strategy   *strategyLinkBucket
	service2Strategy     *strategyLinkBucket
	configGroup2Strategy *strategyLinkBucket

	userCache UserCache

	firstUpdate    bool
	lastUpdateTime int64

	singleFlight *singleflight.Group

	principalCh chan interface{}
}

type strategyBucket struct {
	lock       sync.RWMutex
	strategies map[string]*model.StrategyDetailCache
}

func (s *strategyBucket) save(key string, val *model.StrategyDetailCache) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.strategies[key] = val
}

func (s *strategyBucket) delete(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.strategies, key)
}

func (s *strategyBucket) get(key string) (*model.StrategyDetailCache, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	val, ok := s.strategies[key]
	return val, ok
}

type strategyIdBucket struct {
	lock sync.RWMutex
	ids  map[string]struct{}
}

func (s *strategyIdBucket) save(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.ids[key] = struct{}{}
}

func (s *strategyIdBucket) delete(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.ids, key)
}

func (s *strategyIdBucket) toSlice() []string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ret := make([]string, 0, len(s.ids))

	for k := range s.ids {
		ret = append(ret, k)
	}

	return ret
}

type strategyLinkBucket struct {
	lock       sync.RWMutex
	strategies map[string]*strategyIdBucket
}

func (s *strategyLinkBucket) save(linkId, strategyId string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.strategies[linkId]; !ok {
		s.strategies[linkId] = &strategyIdBucket{
			lock: sync.RWMutex{},
			ids:  make(map[string]struct{}),
		}
	}

	s.strategies[linkId].save(strategyId)
}

func (s *strategyLinkBucket) deleteAllLink(linkId string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.strategies, linkId)
}

func (s *strategyLinkBucket) delete(linkId, strategyID string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	v, ok := s.strategies[linkId]
	if !ok {
		return
	}

	v.delete(strategyID)

	if len(v.ids) == 0 {
		delete(s.strategies, linkId)
	}
}

func (s *strategyLinkBucket) get(key string) ([]string, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	val, ok := s.strategies[key]
	if !ok {
		return []string{}, ok
	}
	return val.toSlice(), ok
}

// newStrategyCache
func newStrategyCache(storage store.Store, principalCh chan interface{}, userCache UserCache) StrategyCache {
	return &strategyCache{
		baseCache:   newBaseCache(),
		storage:     storage,
		principalCh: principalCh,
		userCache:   userCache,
	}
}

func (sc *strategyCache) initBuckets() {
	sc.strategys = &strategyBucket{
		lock:       sync.RWMutex{},
		strategies: make(map[string]*model.StrategyDetailCache),
	}
	sc.uid2Strategy = &strategyLinkBucket{
		lock:       sync.RWMutex{},
		strategies: make(map[string]*strategyIdBucket),
	}
	sc.groupid2Strategy = &strategyLinkBucket{
		lock:       sync.RWMutex{},
		strategies: make(map[string]*strategyIdBucket),
	}

	sc.namespace2Strategy = &strategyLinkBucket{
		lock:       sync.RWMutex{},
		strategies: make(map[string]*strategyIdBucket),
	}
	sc.service2Strategy = &strategyLinkBucket{
		lock:       sync.RWMutex{},
		strategies: make(map[string]*strategyIdBucket),
	}
	sc.configGroup2Strategy = &strategyLinkBucket{
		lock:       sync.RWMutex{},
		strategies: make(map[string]*strategyIdBucket),
	}
}

func (sc *strategyCache) initialize(c map[string]interface{}) error {
	sc.initBuckets()

	sc.singleFlight = new(singleflight.Group)
	sc.firstUpdate = true
	sc.lastUpdateTime = 0
	return nil
}

func (sc *strategyCache) update(storeRollbackSec time.Duration) error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := sc.singleFlight.Do(StrategyRuleName, func() (interface{}, error) {
		return nil, sc.realUpdate(storeRollbackSec)
	})
	return err
}

func (sc *strategyCache) realUpdate(storeRollbackSec time.Duration) error {
	// 获取几秒前的全部数据
	start := time.Now()
	lastMtime := time.Unix(sc.lastUpdateTime, 0)
	strategys, err := sc.storage.GetStrategyDetailsForCache(lastMtime.Add(storeRollbackSec), sc.firstUpdate)
	if err != nil {
		log.CacheScope().Errorf("[Cache][AuthStrategy] refresh auth strategy cache err: %s", err.Error())
		return err
	}

	sc.firstUpdate = false
	add, update, del := sc.setStrategys(strategys)
	log.CacheScope().Info("[Cache][AuthStrategy] get more auth strategy",
		zap.Int("add", add), zap.Int("update", update), zap.Int("delete", del),
		zap.Time("last", lastMtime), zap.Duration("used", time.Now().Sub(start)))
	return nil
}

// setStrategys 处理策略的数据更新情况
// step 1. 先处理resource以及principal的数据更新情况（主要是为了能够获取到新老数据进行对比计算）
// step 2. 处理真正的 strategy 的缓存更新
func (sc *strategyCache) setStrategys(strategies []*model.StrategyDetail) (int, int, int) {

	var (
		add    int
		remove int
		update int
	)

	sc.handlerResourceStrategy(strategies)
	sc.handlerPrincipalStrategy(strategies)

	for index := range strategies {
		rule := strategies[index]
		if !rule.Valid {
			sc.strategys.delete(rule.ID)
			remove++
		} else {
			_, ok := sc.strategys.get(rule.ID)
			if !ok {
				add++
			} else {
				update++
			}
			sc.strategys.save(rule.ID, buildEnchanceStrategyDetail(rule))

			sc.lastUpdateTime = int64(math.Max(float64(sc.lastUpdateTime), float64(rule.ModifyTime.Unix())))
		}
	}

	sc.postProcessPrincipalCh()

	return add, update, remove
}

func buildEnchanceStrategyDetail(strategy *model.StrategyDetail) *model.StrategyDetailCache {
	users := make(map[string]model.Principal, 0)
	groups := make(map[string]model.Principal, 0)

	for index := range strategy.Principals {
		principal := strategy.Principals[index]
		if principal.PrincipalRole == model.PrincipalUser {
			users[principal.PrincipalID] = principal
		} else {
			groups[principal.PrincipalID] = principal
		}
	}

	return &model.StrategyDetailCache{
		StrategyDetail: strategy,
		UserPrincipal:  users,
		GroupPrincipal: groups,
	}
}

// handlerResourceStrategy 处理资源视角下策略的缓存
// 根据新老策略的资源列表比对，计算出哪些资源不在和该策略存在关联关系，哪些资源新增了相关的策略
func (sc *strategyCache) handlerResourceStrategy(strategies []*model.StrategyDetail) {
	operateLink := func(resType int32, resId, strategyId string, remove bool) {
		switch resType {
		case int32(api.ResourceType_Namespaces):
			if remove {
				sc.namespace2Strategy.delete(resId, strategyId)
			} else {
				sc.namespace2Strategy.save(resId, strategyId)
			}
		case int32(api.ResourceType_Services):
			if remove {
				sc.service2Strategy.delete(resId, strategyId)
			} else {
				sc.service2Strategy.save(resId, strategyId)
			}
		case int32(api.ResourceType_ConfigGroups):
			if remove {
				sc.configGroup2Strategy.delete(resId, strategyId)
			} else {
				sc.configGroup2Strategy.save(resId, strategyId)
			}
		}
	}

	for sIndex := range strategies {
		rule := strategies[sIndex]
		addRes := rule.Resources

		if oldRule, exist := sc.strategys.get(rule.ID); exist {
			delRes := make([]model.StrategyResource, 0, 8)
			// 计算前后对比， resource 的变化
			newRes := make(map[string]struct{}, len(addRes))
			for i := range addRes {
				newRes[fmt.Sprintf("%d_%s", addRes[i].ResType, addRes[i].ResID)] = struct{}{}
			}

			// 筛选出从策略中被踢出的 resource 列表
			for i := range oldRule.Resources {
				item := oldRule.Resources[i]
				if _, ok := newRes[fmt.Sprintf("%d_%s", item.ResType, item.ResID)]; !ok {
					delRes = append(delRes, item)
				}
			}

			// 针对被剔除的 resource 列表，清理掉所关联的鉴权策略信息
			for rIndex := range delRes {
				resource := delRes[rIndex]
				operateLink(resource.ResType, resource.ResID, rule.ID, true)
			}
		}

		for rIndex := range addRes {
			resource := addRes[rIndex]
			if rule.Valid {
				operateLink(resource.ResType, resource.ResID, rule.ID, false)
			} else {
				operateLink(resource.ResType, resource.ResID, rule.ID, true)
			}
		}
	}
}

// handlerPrincipalStrategy
func (sc *strategyCache) handlerPrincipalStrategy(strategies []*model.StrategyDetail) {

	for index := range strategies {
		rule := strategies[index]
		// 计算 uid -> auth rule
		principals := rule.Principals

		if oldRule, exist := sc.strategys.get(rule.ID); exist {
			delMembers := make([]model.Principal, 0, 8)
			// 计算前后对比， principal 的变化
			newRes := make(map[string]struct{}, len(principals))
			for i := range principals {
				newRes[fmt.Sprintf("%d_%s", principals[i].PrincipalRole, principals[i].PrincipalID)] = struct{}{}
			}

			// 筛选出从策略中被踢出的 principal 列表
			for i := range oldRule.Principals {
				item := oldRule.Principals[i]
				if _, ok := newRes[fmt.Sprintf("%d_%s", item.PrincipalRole, item.PrincipalID)]; !ok {
					delMembers = append(delMembers, item)
				}
			}

			// 针对被剔除的 principal 列表，清理掉所关联的鉴权策略信息
			for rIndex := range delMembers {
				principal := delMembers[rIndex]
				sc.removePrincipalLink(principal, rule)
			}
		}
		if rule.Valid {
			for pos := range principals {
				principal := principals[pos]
				sc.addPrincipalLink(principal, rule)
			}
		} else {
			for pos := range principals {
				principal := principals[pos]
				sc.removePrincipalLink(principal, rule)
			}
		}
	}
}

func (sc *strategyCache) removePrincipalLink(principal model.Principal, rule *model.StrategyDetail) {
	if principal.PrincipalRole == model.PrincipalUser {
		sc.uid2Strategy.delete(principal.PrincipalID, rule.ID)
	} else {
		sc.groupid2Strategy.delete(principal.PrincipalID, rule.ID)
	}
}

func (sc *strategyCache) addPrincipalLink(principal model.Principal, rule *model.StrategyDetail) {
	if principal.PrincipalRole == model.PrincipalUser {
		sc.uid2Strategy.save(principal.PrincipalID, rule.ID)
	} else {
		sc.groupid2Strategy.save(principal.PrincipalID, rule.ID)
	}
}

// postProcessPrincipalCh
func (sc *strategyCache) postProcessPrincipalCh() {
	select {
	case event := <-sc.principalCh:
		principals := event.([]model.Principal)
		for index := range principals {
			principal := principals[index]

			if principal.PrincipalRole == model.PrincipalUser {
				sc.uid2Strategy.deleteAllLink(principal.PrincipalID)
			} else {
				sc.groupid2Strategy.deleteAllLink(principal.PrincipalID)
			}
		}
	case <-time.After(time.Duration(100 * time.Millisecond)):
		return
	}
}

func (sc *strategyCache) clear() error {
	sc.initBuckets()

	sc.firstUpdate = true
	sc.lastUpdateTime = 0
	return nil
}

func (sc *strategyCache) name() string {
	return StrategyRuleName
}

// 对于 check 逻辑，如果是计算 * 策略，则必须要求 * 资源下必须有策略
// 如果是具体的资源ID，则该资源下不必有策略，如果没有策略就认为这个资源是可以被任何人编辑的
func (sc *strategyCache) checkResourceEditable(strategIds []string, principal model.Principal, mustCheck bool) bool {
	// 是否可以编辑
	editable := false
	// 是否真的包含策略
	isCheck := len(strategIds) != 0

	// 如果根本没有遍历过，则表示该资源下没有对应的策略列表，直接返回可编辑状态即可
	if !isCheck && !mustCheck {
		return true
	}

	for i := range strategIds {
		isCheck = true
		if rule, ok := sc.strategys.get(strategIds[i]); ok {
			if principal.PrincipalRole == model.PrincipalUser {
				_, exist := rule.UserPrincipal[principal.PrincipalID]
				editable = editable || exist
			} else {
				_, exist := rule.GroupPrincipal[principal.PrincipalID]
				editable = editable || exist
			}
		}
	}

	return editable
}

// IsResourceEditable 判断当前资源是否可以操作
// 这里需要考虑两种情况，一种是 “ * ” 策略，另一种是明确指出了具体的资源ID的策略
func (sc *strategyCache) IsResourceEditable(principal model.Principal, resType api.ResourceType, resId string) bool {

	var (
		valAll, val []string
		ok          bool
	)
	switch resType {
	case api.ResourceType_Namespaces:
		val, ok = sc.namespace2Strategy.get(resId)
		valAll, _ = sc.namespace2Strategy.get("*")
	case api.ResourceType_Services:
		val, ok = sc.service2Strategy.get(resId)
		valAll, _ = sc.service2Strategy.get("*")
	case api.ResourceType_ConfigGroups:
		val, ok = sc.configGroup2Strategy.get(resId)
		valAll, _ = sc.configGroup2Strategy.get("*")
	}

	// 代表该资源没有关联到任何策略，任何人都可以编辑
	if !ok {
		return true
	}

	principals := make([]model.Principal, 0, 4)
	principals = append(principals, principal)
	if principal.PrincipalRole == model.PrincipalUser {
		groupids := sc.userCache.GetUserLinkGroupIds(principal.PrincipalID)
		for i := range groupids {
			principals = append(principals, model.Principal{
				PrincipalID:   groupids[i],
				PrincipalRole: model.PrincipalGroup,
			})
		}
	}

	for i := range principals {
		item := principals[i]
		if valAll != nil && sc.checkResourceEditable(valAll, item, true) {
			return true
		}

		if sc.checkResourceEditable(val, item, false) {
			return true
		}
	}

	return false
}

func (sc *strategyCache) GetStrategyDetailsByUID(uid string) []*model.StrategyDetail {

	return sc.getStrategyDetails(uid, "")
}

func (sc *strategyCache) GetStrategyDetailsByGroupID(groupid string) []*model.StrategyDetail {

	return sc.getStrategyDetails("", groupid)
}

func (sc *strategyCache) getStrategyDetails(uid string, gid string) []*model.StrategyDetail {
	var (
		strategyIds []string
		ok          bool
	)
	if uid != "" {
		strategyIds, ok = sc.uid2Strategy.get(uid)
		if !ok {
			return nil
		}
	} else if gid != "" {
		strategyIds, ok = sc.groupid2Strategy.get(gid)
		if !ok {
			return nil
		}
	}

	if strategyIds != nil {
		result := make([]*model.StrategyDetail, 0, 16)
		for i := range strategyIds {
			strategy, ok := sc.strategys.get(strategyIds[i])
			if ok {
				result = append(result, strategy.StrategyDetail)
			}
		}

		return result
	}

	return nil
}

// IsResourceLinkStrategy 校验
func (sc *strategyCache) IsResourceLinkStrategy(resType api.ResourceType, resId string) bool {
	switch resType {
	case api.ResourceType_Namespaces:
		val, ok := sc.namespace2Strategy.get(resId)
		return ok && hasLinkRule(val)
	case api.ResourceType_Services:
		val, ok := sc.service2Strategy.get(resId)
		return ok && hasLinkRule(val)
	case api.ResourceType_ConfigGroups:
		val, ok := sc.configGroup2Strategy.get(resId)
		return ok && hasLinkRule(val)
	default:
		return true
	}
}

func hasLinkRule(val []string) bool {
	return len(val) != 0
}
