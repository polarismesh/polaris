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

package auth

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	types "github.com/polarismesh/polaris/cache/api"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const (
	removePrincipalChSize = 8
)

// policyCache
type policyCache struct {
	*types.BaseCache

	rules         *utils.SyncMap[string, *authcommon.PolicyDetailCache]
	allowPolicies map[authcommon.PrincipalType]*utils.SyncMap[string, *utils.SyncSet[string]]
	denyPolicies  map[authcommon.PrincipalType]*utils.SyncMap[string, *utils.SyncSet[string]]

	// principalResources
	principalResources map[authcommon.PrincipalType]*utils.SyncMap[string, *authcommon.PrincipalResourceContainer]

	allowResourceLabels *utils.SyncMap[string, *utils.RefSyncSet[string]]
	denyResourceLabels  *utils.SyncMap[string, *utils.RefSyncSet[string]]

	singleFlight *singleflight.Group
}

// NewStrategyCache
func NewStrategyCache(storage store.Store, cacheMgr types.CacheManager) types.StrategyCache {
	return &policyCache{
		BaseCache:    types.NewBaseCache(storage, cacheMgr),
		singleFlight: new(singleflight.Group),
	}
}

func (sc *policyCache) Initialize(c map[string]interface{}) error {
	sc.initContainers()
	return nil
}

func (sc *policyCache) Clear() error {
	sc.BaseCache.Clear()
	sc.initContainers()
	return nil
}

func (sc *policyCache) initContainers() {
	sc.rules = utils.NewSyncMap[string, *authcommon.PolicyDetailCache]()
	sc.allowPolicies = map[authcommon.PrincipalType]*utils.SyncMap[string, *utils.SyncSet[string]]{
		authcommon.PrincipalUser:  utils.NewSyncMap[string, *utils.SyncSet[string]](),
		authcommon.PrincipalGroup: utils.NewSyncMap[string, *utils.SyncSet[string]](),
	}
	sc.denyPolicies = map[authcommon.PrincipalType]*utils.SyncMap[string, *utils.SyncSet[string]]{
		authcommon.PrincipalUser:  utils.NewSyncMap[string, *utils.SyncSet[string]](),
		authcommon.PrincipalGroup: utils.NewSyncMap[string, *utils.SyncSet[string]](),
	}
	sc.principalResources = map[authcommon.PrincipalType]*utils.SyncMap[string, *authcommon.PrincipalResourceContainer]{
		authcommon.PrincipalUser:  utils.NewSyncMap[string, *authcommon.PrincipalResourceContainer](),
		authcommon.PrincipalGroup: utils.NewSyncMap[string, *authcommon.PrincipalResourceContainer](),
	}
	sc.allowResourceLabels = utils.NewSyncMap[string, *utils.RefSyncSet[string]]()
	sc.denyResourceLabels = utils.NewSyncMap[string, *utils.RefSyncSet[string]]()
}

func (sc *policyCache) Name() string {
	return types.StrategyRuleName
}

func (sc *policyCache) Update() error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := sc.singleFlight.Do(sc.Name(), func() (interface{}, error) {
		return nil, sc.DoCacheUpdate(sc.Name(), sc.realUpdate)
	})
	return err
}

func (sc *policyCache) realUpdate() (map[string]time.Time, int64, error) {
	// 获取几秒前的全部数据
	var (
		start           = time.Now()
		lastTime        = sc.LastFetchTime()
		strategies, err = sc.BaseCache.Store().GetMoreStrategies(lastTime, sc.IsFirstUpdate())
	)
	if err != nil {
		log.Errorf("[Cache][AuthStrategy] refresh auth strategy cache err: %s", err.Error())
		return nil, -1, err
	}

	lastMtimes, add, update, del := sc.setStrategys(strategies)
	log.Info("[Cache][AuthStrategy] get more auth strategy",
		zap.Int("add", add), zap.Int("update", update), zap.Int("delete", del),
		zap.Time("last", lastTime), zap.Duration("used", time.Since(start)))
	return lastMtimes, int64(len(strategies)), nil
}

// setStrategys 处理策略的数据更新情况
// step 1. 先处理resource以及principal的数据更新情况（主要是为了能够获取到新老数据进行对比计算）
// step 2. 处理真正的 strategy 的缓存更新
func (sc *policyCache) setStrategys(strategies []*authcommon.StrategyDetail) (map[string]time.Time, int, int, int) {
	var add, remove, update int
	lastMtime := sc.LastMtime(sc.Name()).Unix()

	for index := range strategies {
		rule := strategies[index]
		sc.handlePrincipalPolicies(rule)
		if !rule.Valid {
			sc.rules.Delete(rule.ID)
			remove++
		} else {
			if _, ok := sc.rules.Load(rule.ID); !ok {
				add++
			} else {
				update++
			}
			sc.rules.Store(rule.ID, authcommon.NewPolicyDetailCache(rule))
		}

		lastMtime = int64(math.Max(float64(lastMtime), float64(rule.ModifyTime.Unix())))
	}

	return map[string]time.Time{sc.Name(): time.Unix(lastMtime, 0)}, add, update, remove
}

// handlePrincipalPolicies
func (sc *policyCache) handlePrincipalPolicies(rule *authcommon.StrategyDetail) {
	// 计算 uid -> auth rule
	principals := rule.Principals

	if oldRule, exist := sc.rules.Load(rule.ID); exist {
		delMembers := make([]authcommon.Principal, 0, 8)
		// 计算前后对比， principal 的变化
		newRes := make(map[string]struct{}, len(principals))
		for i := range principals {
			newRes[fmt.Sprintf("%d_%s", principals[i].PrincipalType, principals[i].PrincipalID)] = struct{}{}
		}

		// 筛选出从策略中被踢出的 principal 列表
		for i := range oldRule.Principals {
			item := oldRule.Principals[i]
			if _, ok := newRes[fmt.Sprintf("%d_%s", item.PrincipalType, item.PrincipalID)]; !ok {
				delMembers = append(delMembers, item)
			}
		}

		// 针对被剔除的 principal 列表，清理掉所关联的鉴权策略信息
		for rIndex := range delMembers {
			principal := delMembers[rIndex]
			sc.writePrincipalLink(principal, rule, true)
		}
	}
	if rule.Valid {
		for pos := range principals {
			principal := principals[pos]
			sc.writePrincipalLink(principal, rule, false)
		}
	} else {
		for pos := range principals {
			principal := principals[pos]
			sc.writePrincipalLink(principal, rule, true)
		}
	}
}

func (sc *policyCache) writePrincipalLink(principal authcommon.Principal, rule *authcommon.StrategyDetail, del bool) {
	linkContainers := sc.allowPolicies[principal.PrincipalType]
	if rule.Action == apisecurity.AuthAction_DENY.String() {
		linkContainers = sc.denyPolicies[principal.PrincipalType]
	}
	values, ok := linkContainers.Load(principal.PrincipalID)
	if !ok && !del {
		linkContainers.ComputeIfAbsent(principal.PrincipalID, func(k string) *utils.SyncSet[string] {
			return utils.NewSyncSet[string]()
		})
	}
	if del {
		values.Remove(rule.ID)
	} else {
		values, _ := linkContainers.Load(principal.PrincipalID)
		values.Add(rule.ID)
	}

	principalResources, _ := sc.principalResources[principal.PrincipalType].ComputeIfAbsent(principal.PrincipalID, func(k string) *authcommon.PrincipalResourceContainer {
		return authcommon.NewPrincipalResourceContainer()
	})

	if rule.IsDeny() {
		for i := range rule.Resources {
			item := rule.Resources[i]
			if rule.Valid {
				principalResources.SaveDenyResource(item)
			} else {
				principalResources.DelDenyResource(item)
			}
		}
		return
	}
	for i := range rule.Resources {
		item := rule.Resources[i]
		if rule.Valid {
			principalResources.SaveAllowResource(item)
		} else {
			principalResources.DelAllowResource(item)
		}
	}
}

func (sc *policyCache) GetPrincipalPolicies(effect string, p authcommon.Principal) []*authcommon.StrategyDetail {
	var ruleIds *utils.SyncSet[string]
	var exist bool
	switch effect {
	case "allow":
		ruleIds, exist = sc.allowPolicies[p.PrincipalType].Load(p.PrincipalID)
	case "deny":
		ruleIds, exist = sc.denyPolicies[p.PrincipalType].Load(p.PrincipalID)
	default:
		allowRuleIds, allowExist := sc.allowPolicies[p.PrincipalType].Load(p.PrincipalID)
		denyRuleIds, denyExist := sc.denyPolicies[p.PrincipalType].Load(p.PrincipalID)
		if allowRuleIds == nil {
			allowRuleIds = utils.NewSyncSet[string]()
		}
		allowRuleIds.AddAll(denyRuleIds)

		ruleIds = allowRuleIds
		exist = allowExist || denyExist
	}

	if !exist {
		return nil
	}
	if ruleIds.Len() == 0 {
		return nil
	}
	result := make([]*authcommon.StrategyDetail, 0, 16)
	ruleIds.Range(func(val string) {
		strategy, ok := sc.rules.Load(val)
		if ok {
			result = append(result, strategy.StrategyDetail)
		}
	})
	return result
}

// GetPrincipalResources 返回 principal 的资源信息，返回顺序为 (allow, deny)
func (sc *policyCache) Hint(p authcommon.Principal, r *authcommon.ResourceEntry) apisecurity.AuthAction {
	resources, ok := sc.principalResources[p.PrincipalType].Load(p.PrincipalID)
	if !ok {
		return apisecurity.AuthAction_DENY
	}
	action, ok := resources.Hint(r.Type, r.ID)
	if ok {
		return action
	}
	// 如果没办法从直接的 resource 中判断出来，那就根据资源标签在确认下，注意，这里必须 allMatch 才可以
	if sc.hintLabels(p, r, sc.denyResourceLabels) {
		return apisecurity.AuthAction_DENY
	}
	if sc.hintLabels(p, r, sc.allowResourceLabels) {
		return apisecurity.AuthAction_ALLOW
	}
	return apisecurity.AuthAction_DENY
}

func (sc *policyCache) hintLabels(p authcommon.Principal, r *authcommon.ResourceEntry,
	containers *utils.SyncMap[string, *utils.RefSyncSet[string]]) bool {
	allMatch := true
	for k, v := range r.Metadata {
		labelVals, ok := sc.denyResourceLabels.Load(k)
		if !ok {
			allMatch = false
		}
		allMatch = labelVals.Contains(v)
		if !allMatch {
			break
		}
	}
	return allMatch
}

// Query implements api.StrategyCache.
func (sc *policyCache) Query(ctx context.Context, args types.PolicySearchArgs) (uint32, []*authcommon.StrategyDetail, error) {
	if err := sc.Update(); err != nil {
		return 0, nil, err
	}

	searchId, hasId := args.Filters["id"]
	searchName, hasName := args.Filters["name"]
	searchOwner, hasOwner := args.Filters["owner"]
	searchDefault, hasDefault := args.Filters["default"]
	searchResType, hasResType := args.Filters["res_type"]
	searchResID, _ := args.Filters["res_id"]
	searchPrincipalId, hasPrincipalId := args.Filters["principal_id"]
	searchPrincipalType, _ := args.Filters["principal_type"]

	predicates := types.LoadAuthPolicyPredicates(ctx)

	rules := make([]*authcommon.StrategyDetail, 0, args.Limit)

	sc.rules.Range(func(key string, val *authcommon.PolicyDetailCache) {
		if hasId && val.ID != searchId {
			return
		}
		if hasName && !utils.IsWildMatch(val.Name, searchName) {
			return
		}
		if hasOwner && searchOwner != val.Owner {
			if !hasPrincipalId {
				return
			}
			if searchPrincipalType == strconv.Itoa(int(authcommon.PrincipalUser)) {
				if _, ok := val.UserPrincipal[searchPrincipalId]; !ok {
					return
				}
			}
			if searchPrincipalType == strconv.Itoa(int(authcommon.PrincipalGroup)) {
				if _, ok := val.GroupPrincipal[searchPrincipalId]; !ok {
					return
				}
			}
		}
		if hasDefault && searchDefault != strconv.FormatBool(val.Default) {
			return
		}
		if hasResType {
			resources, ok := val.ResourceDict[authcommon.SearchTypeMapping[searchResType]]
			if !ok {
				return
			}
			if !resources.Contains(searchResID) {
				return
			}
		}
		if hasPrincipalId {
			if searchPrincipalType == strconv.Itoa(int(authcommon.PrincipalUser)) {
				if _, ok := val.UserPrincipal[searchPrincipalId]; !ok {
					return
				}
			}
			if searchPrincipalType == strconv.Itoa(int(authcommon.PrincipalGroup)) {
				if _, ok := val.GroupPrincipal[searchPrincipalId]; !ok {
					return
				}
			}
		}

		for i := range predicates {
			if !predicates[i](ctx, val.StrategyDetail) {
				return
			}
		}
		rules = append(rules, val.StrategyDetail)
	})

	total, ret := sc.toPage(rules, args)
	return total, ret, nil
}

func (sc *policyCache) toPage(rules []*authcommon.StrategyDetail, args types.PolicySearchArgs) (uint32, []*authcommon.StrategyDetail) {
	beginIndex := args.Offset
	endIndex := beginIndex + args.Limit
	totalCount := uint32(len(rules))

	if totalCount == 0 {
		return totalCount, []*authcommon.StrategyDetail{}
	}
	if beginIndex >= endIndex {
		return totalCount, []*authcommon.StrategyDetail{}
	}
	if beginIndex >= totalCount {
		return totalCount, []*authcommon.StrategyDetail{}
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].ModifyTime.After(rules[j].ModifyTime)
	})

	return totalCount, rules[beginIndex:endIndex]
}
