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
	strategys        *sync.Map
	uid2Strategy     *sync.Map
	groupid2Strategy *sync.Map

	namespace2Strategy   *sync.Map
	service2Strategy     *sync.Map
	configGroup2Strategy *sync.Map

	userCache UserCache

	firstUpdate    bool
	lastUpdateTime int64

	singleFlight *singleflight.Group

	principalCh chan interface{}
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

func (sc *strategyCache) initialize(c map[string]interface{}) error {
	sc.strategys = new(sync.Map)
	sc.uid2Strategy = new(sync.Map)
	sc.groupid2Strategy = new(sync.Map)

	sc.namespace2Strategy = new(sync.Map)
	sc.service2Strategy = new(sync.Map)
	sc.configGroup2Strategy = new(sync.Map)

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
			sc.strategys.Delete(rule.ID)
			remove++
		} else {
			_, ok := sc.strategys.Load(rule.ID)
			if !ok {
				add++
			} else {
				update++
			}
			sc.strategys.Store(rule.ID, buildEnchanceStrategyDetail(rule))

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
	supplier := func(resType int32, resId string) interface{} {
		var val interface{}
		switch resType {
		case int32(api.ResourceType_Namespaces):
			val, _ = sc.namespace2Strategy.LoadOrStore(resId, new(sync.Map))
		case int32(api.ResourceType_Services):
			val, _ = sc.service2Strategy.LoadOrStore(resId, new(sync.Map))
		case int32(api.ResourceType_ConfigGroups):
			val, _ = sc.configGroup2Strategy.LoadOrStore(resId, new(sync.Map))
		}
		return val
	}

	for sIndex := range strategies {
		rule := strategies[sIndex]
		addRes := rule.Resources

		if oldVal, exist := sc.strategys.Load(rule.ID); exist {
			oldRule := oldVal.(*model.StrategyDetailCache)
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
				val := supplier(resource.ResType, resource.ResID)
				val.(*sync.Map).Delete(rule.ID)
			}
		}

		for rIndex := range addRes {
			resource := addRes[rIndex]
			val := supplier(resource.ResType, resource.ResID)
			if rule.Valid {
				val.(*sync.Map).Store(rule.ID, struct{}{})
			} else {
				val.(*sync.Map).Delete(rule.ID)
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

		if oldVal, exist := sc.strategys.Load(rule.ID); exist {
			oldRule := oldVal.(*model.StrategyDetailCache)
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
	sc.operatePrincipalLink(principal, rule, true)
}

func (sc *strategyCache) addPrincipalLink(principal model.Principal, rule *model.StrategyDetail) {
	sc.operatePrincipalLink(principal, rule, false)
}

func (sc *strategyCache) operatePrincipalLink(principal model.Principal, rule *model.StrategyDetail, remove bool) {
	if remove {
		if principal.PrincipalRole == model.PrincipalUser {
			if val, ok := sc.uid2Strategy.Load(principal.PrincipalID); ok {
				val.(*sync.Map).Delete(rule.ID)
			}
		} else {
			if val, ok := sc.groupid2Strategy.Load(principal.PrincipalID); ok {
				val.(*sync.Map).Delete(rule.ID)
			}
		}
	} else {
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
		rulesMap.Store(rule.ID, struct{}{})
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

// 对于 check 逻辑，如果是计算 * 策略，则必须要求 * 资源下必须有策略
// 如果是具体的资源ID，则该资源下不必有策略，如果没有策略就认为这个资源是可以被任何人编辑的
func (sc *strategyCache) checkResourceEditable(val *sync.Map, principal model.Principal, mustCheck bool) bool {
	// 是否可以编辑
	editable := false
	// 是否真的包含策略
	isCheck := false
	val.Range(func(key, _ interface{}) bool {
		isCheck = true
		if val, ok := sc.strategys.Load(key); ok {
			rule := val.(*model.StrategyDetailCache)
			if principal.PrincipalRole == model.PrincipalUser {
				_, editable = rule.UserPrincipal[principal.PrincipalID]
			} else {
				_, editable = rule.GroupPrincipal[principal.PrincipalID]
			}
		}
		return !editable
	})

	// 如果根本没有遍历过，则表示该资源下没有对应的策略列表，直接返回可编辑状态即可
	if !isCheck && !mustCheck {
		return true
	}
	return editable
}

// IsResourceEditable 判断当前资源是否可以操作
// 这里需要考虑两种情况，一种是 “ * ” 策略，另一种是明确指出了具体的资源ID的策略
func (sc *strategyCache) IsResourceEditable(principal model.Principal, resType api.ResourceType, resId string) bool {

	var (
		valAll, val interface{}
		ok          bool
	)
	switch resType {
	case api.ResourceType_Namespaces:
		val, ok = sc.namespace2Strategy.Load(resId)
		valAll, _ = sc.namespace2Strategy.Load("*")
	case api.ResourceType_Services:
		val, ok = sc.service2Strategy.Load(resId)
		valAll, _ = sc.service2Strategy.Load("*")
	case api.ResourceType_ConfigGroups:
		val, ok = sc.configGroup2Strategy.Load(resId)
		valAll, _ = sc.configGroup2Strategy.Load("*")
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
		if valAll != nil && sc.checkResourceEditable(valAll.(*sync.Map), item, true) {
			return true
		}

		if sc.checkResourceEditable(val.(*sync.Map), item, false) {
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
	var strategyIds *sync.Map
	if uid != "" {
		val, ok := sc.uid2Strategy.Load(uid)
		if !ok {
			return nil
		}
		strategyIds = val.(*sync.Map)
	} else if gid != "" {
		val, ok := sc.groupid2Strategy.Load(gid)
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
				result = append(result, strategy.(*model.StrategyDetailCache).StrategyDetail)
			}
			return true
		})

		return result
	}

	return nil
}

// IsResourceLinkStrategy 校验
func (sc *strategyCache) IsResourceLinkStrategy(resType api.ResourceType, resId string) bool {
	switch resType {
	case api.ResourceType_Namespaces:
		val, ok := sc.namespace2Strategy.Load(resId)
		return ok && synMapIsNotEmpty(val.(*sync.Map))
	case api.ResourceType_Services:
		val, ok := sc.service2Strategy.Load(resId)
		return ok && synMapIsNotEmpty(val.(*sync.Map))
	case api.ResourceType_ConfigGroups:
		val, ok := sc.configGroup2Strategy.Load(resId)
		return ok && synMapIsNotEmpty(val.(*sync.Map))
	default:
		return true
	}
}

func synMapIsNotEmpty(val *sync.Map) bool {
	count := 0

	val.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	return count != 0
}
