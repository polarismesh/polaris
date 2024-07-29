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
	"crypto/sha1"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

// circuitBreaker的实现
type circuitBreakerCache struct {
	*types.BaseCache

	storage store.Store
	// rules record id -> *model.CircuitBreakerRule
	rules *utils.SyncMap[string, *model.CircuitBreakerRule]
	// increment cache
	// fetched service cache
	// key1: namespace, key2: service
	circuitBreakers map[string]map[string]*model.ServiceWithCircuitBreakerRules
	// key1: namespace
	nsWildcardRules map[string]*model.ServiceWithCircuitBreakerRules
	// all rules are wildcard specific
	allWildcardRules *model.ServiceWithCircuitBreakerRules
	lock             sync.RWMutex

	singleFlight singleflight.Group
}

// NewCircuitBreakerCache 返回一个操作CircuitBreakerCache的对象
func NewCircuitBreakerCache(s store.Store, cacheMgr types.CacheManager) types.CircuitBreakerCache {
	return &circuitBreakerCache{
		BaseCache:       types.NewBaseCache(s, cacheMgr),
		storage:         s,
		rules:           utils.NewSyncMap[string, *model.CircuitBreakerRule](),
		circuitBreakers: make(map[string]map[string]*model.ServiceWithCircuitBreakerRules),
		nsWildcardRules: make(map[string]*model.ServiceWithCircuitBreakerRules),
		allWildcardRules: model.NewServiceWithCircuitBreakerRules(model.ServiceKey{
			Namespace: types.AllMatched,
			Name:      types.AllMatched,
		}),
	}
}

// Initialize 实现Cache接口的函数
func (c *circuitBreakerCache) Initialize(_ map[string]interface{}) error {
	return nil
}

// Update 实现Cache接口的函数
func (c *circuitBreakerCache) Update() error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := c.singleFlight.Do(c.Name(), func() (interface{}, error) {
		return nil, c.DoCacheUpdate(c.Name(), c.realUpdate)
	})
	return err
}

func (c *circuitBreakerCache) realUpdate() (map[string]time.Time, int64, error) {
	start := time.Now()
	cbRules, err := c.storage.GetCircuitBreakerRulesForCache(c.LastFetchTime(), c.IsFirstUpdate())
	if err != nil {
		log.Errorf("[Cache][CircuitBreaker] cache update err:%s", err.Error())
		return nil, -1, err
	}
	lastMtimes, upsert, del := c.setCircuitBreaker(cbRules)
	log.Info("[Cache][CircuitBreaker] get more rules",
		zap.Int("pull-from-store", len(cbRules)), zap.Int("upsert", upsert), zap.Int("delete", del),
		zap.Time("last", c.LastMtime(c.Name())), zap.Duration("used", time.Since(start)))
	return lastMtimes, int64(len(cbRules)), nil
}

// clear 实现Cache接口的函数
func (c *circuitBreakerCache) Clear() error {
	c.BaseCache.Clear()
	c.lock.Lock()
	c.allWildcardRules.Clear()
	c.rules = utils.NewSyncMap[string, *model.CircuitBreakerRule]()
	c.nsWildcardRules = make(map[string]*model.ServiceWithCircuitBreakerRules)
	c.circuitBreakers = make(map[string]map[string]*model.ServiceWithCircuitBreakerRules)
	c.lock.Unlock()
	return nil
}

// name 实现资源名称
func (c *circuitBreakerCache) Name() string {
	return types.CircuitBreakerName
}

// GetCircuitBreakerConfig 根据serviceID获取熔断规则
func (c *circuitBreakerCache) GetCircuitBreakerConfig(
	name string, namespace string) *model.ServiceWithCircuitBreakerRules {
	// check service specific
	rules := c.checkServiceSpecificCache(name, namespace)
	if nil != rules {
		return rules
	}
	rules = c.checkNamespaceSpecificCache(namespace)
	if nil != rules {
		return rules
	}
	return c.allWildcardRules
}

func (c *circuitBreakerCache) checkServiceSpecificCache(
	name string, namespace string) *model.ServiceWithCircuitBreakerRules {
	c.lock.RLock()
	defer c.lock.RUnlock()
	svcRules, ok := c.circuitBreakers[namespace]
	if ok {
		return svcRules[name]
	}
	return nil
}

func (c *circuitBreakerCache) checkNamespaceSpecificCache(namespace string) *model.ServiceWithCircuitBreakerRules {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.nsWildcardRules[namespace]
}

func (c *circuitBreakerCache) reloadRevision(svcRules *model.ServiceWithCircuitBreakerRules) {
	rulesCount := svcRules.CountCircuitBreakerRules()
	if rulesCount == 0 {
		svcRules.Revision = ""
		return
	}
	revisions := make([]string, 0, rulesCount)
	svcRules.IterateCircuitBreakerRules(func(rule *model.CircuitBreakerRule) {
		revisions = append(revisions, rule.Revision)
	})
	sort.Strings(revisions)
	h := sha1.New()
	revision, err := types.ComputeRevisionBySlice(h, revisions)
	if err != nil {
		log.Errorf("[Server][Service][CircuitBreaker] compute revision service(%s) err: %s",
			svcRules.Service, err.Error())
		return
	}
	svcRules.Revision = revision
}

func (c *circuitBreakerCache) deleteAndReloadCircuitBreakerRules(
	svcRules *model.ServiceWithCircuitBreakerRules, id string) {
	svcRules.DelCircuitBreakerRule(id)
	c.reloadRevision(svcRules)
}

func (c *circuitBreakerCache) deleteCircuitBreakerFromServiceCache(id string, svcKeys map[model.ServiceKey]bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if len(svcKeys) == 0 {
		// all wildcard
		c.deleteAndReloadCircuitBreakerRules(c.allWildcardRules, id)
		for _, rules := range c.nsWildcardRules {
			c.deleteAndReloadCircuitBreakerRules(rules, id)
		}
		for _, svcRules := range c.circuitBreakers {
			for _, rules := range svcRules {
				c.deleteAndReloadCircuitBreakerRules(rules, id)
			}
		}
		return
	}
	svcToReloads := make(map[model.ServiceKey]bool)
	for svcKey := range svcKeys {
		if svcKey.Name == types.AllMatched {
			rules, ok := c.nsWildcardRules[svcKey.Namespace]
			if ok {
				c.deleteAndReloadCircuitBreakerRules(rules, id)
			}
			svcRules, ok := c.circuitBreakers[svcKey.Namespace]
			if ok {
				for svc := range svcRules {
					svcToReloads[model.ServiceKey{Namespace: svcKey.Namespace, Name: svc}] = true
				}
			}
		} else {
			svcToReloads[svcKey] = true
		}
	}
	if len(svcToReloads) > 0 {
		for svcToReload := range svcToReloads {
			svcRules, ok := c.circuitBreakers[svcToReload.Namespace]
			if ok {
				rules, ok := svcRules[svcToReload.Name]
				if ok {
					c.deleteAndReloadCircuitBreakerRules(rules, id)
				}
			}
		}
	}
}

func (c *circuitBreakerCache) storeAndReloadCircuitBreakerRules(
	svcRules *model.ServiceWithCircuitBreakerRules, cbRule *model.CircuitBreakerRule) {
	svcRules.AddCircuitBreakerRule(cbRule)
	c.reloadRevision(svcRules)
}

func createAndStoreServiceWithCircuitBreakerRules(svcKey model.ServiceKey, key string,
	values map[string]*model.ServiceWithCircuitBreakerRules) *model.ServiceWithCircuitBreakerRules {
	rules := model.NewServiceWithCircuitBreakerRules(svcKey)
	values[key] = rules
	return rules
}

func (c *circuitBreakerCache) storeCircuitBreakerToServiceCache(
	entry *model.CircuitBreakerRule, svcKeys map[model.ServiceKey]bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if len(svcKeys) == 0 {
		// all wildcard
		c.storeAndReloadCircuitBreakerRules(c.allWildcardRules, entry)
		for _, rules := range c.nsWildcardRules {
			c.storeAndReloadCircuitBreakerRules(rules, entry)
		}
		for _, svcRules := range c.circuitBreakers {
			for _, rules := range svcRules {
				c.storeAndReloadCircuitBreakerRules(rules, entry)
			}
		}
		return
	}
	svcToReloads := make(map[model.ServiceKey]bool)
	for svcKey := range svcKeys {
		if svcKey.Name == types.AllMatched {
			var wildcardRules *model.ServiceWithCircuitBreakerRules
			var ok bool
			wildcardRules, ok = c.nsWildcardRules[svcKey.Namespace]
			if !ok {
				wildcardRules = createAndStoreServiceWithCircuitBreakerRules(svcKey, svcKey.Namespace, c.nsWildcardRules)
				// add all exists wildcard rules
				c.allWildcardRules.IterateCircuitBreakerRules(func(rule *model.CircuitBreakerRule) {
					wildcardRules.AddCircuitBreakerRule(rule)
				})
			}
			c.storeAndReloadCircuitBreakerRules(wildcardRules, entry)
			svcRules, ok := c.circuitBreakers[svcKey.Namespace]
			if ok {
				for svc := range svcRules {
					svcToReloads[model.ServiceKey{Namespace: svcKey.Namespace, Name: svc}] = true
				}
			}
		} else {
			svcToReloads[svcKey] = true
		}
	}
	if len(svcToReloads) > 0 {
		for svcToReload := range svcToReloads {
			var rules *model.ServiceWithCircuitBreakerRules
			var svcRules map[string]*model.ServiceWithCircuitBreakerRules
			var ok bool
			svcRules, ok = c.circuitBreakers[svcToReload.Namespace]
			if !ok {
				svcRules = make(map[string]*model.ServiceWithCircuitBreakerRules)
				c.circuitBreakers[svcToReload.Namespace] = svcRules
			}
			rules, ok = svcRules[svcToReload.Name]
			if !ok {
				rules = createAndStoreServiceWithCircuitBreakerRules(svcToReload, svcToReload.Name, svcRules)
				// add all exists wildcard rules
				c.allWildcardRules.IterateCircuitBreakerRules(func(rule *model.CircuitBreakerRule) {
					rules.AddCircuitBreakerRule(rule)
				})
				// add all namespace wildcard rules
				nsRules, ok := c.nsWildcardRules[svcToReload.Namespace]
				if ok {
					nsRules.IterateCircuitBreakerRules(func(rule *model.CircuitBreakerRule) {
						rules.AddCircuitBreakerRule(rule)
					})
				}
			}
			c.storeAndReloadCircuitBreakerRules(rules, entry)
		}
	}
}

func getServicesInvolveByCircuitBreakerRule(cbRule *model.CircuitBreakerRule) map[model.ServiceKey]bool {
	svcKeys := make(map[model.ServiceKey]bool)
	addService := func(name string, namespace string) {
		if len(name) == 0 && len(namespace) == 0 {
			return
		}
		if name == types.AllMatched && namespace == types.AllMatched {
			return
		}
		svcKeys[model.ServiceKey{
			Namespace: namespace,
			Name:      name,
		}] = true
	}
	addService(cbRule.DstService, cbRule.DstNamespace)
	return svcKeys
}

// setCircuitBreaker 更新store的数据到cache中
func (c *circuitBreakerCache) setCircuitBreaker(
	cbRules []*model.CircuitBreakerRule) (map[string]time.Time, int, int) {

	if len(cbRules) == 0 {
		return nil, 0, 0
	}

	var upsert, del int

	lastMtime := c.LastMtime(c.Name()).Unix()

	for _, cbRule := range cbRules {
		if cbRule.ModifyTime.Unix() > lastMtime {
			lastMtime = cbRule.ModifyTime.Unix()
		}

		oldRule, ok := c.rules.Load(cbRule.ID)
		if ok {
			// 对比规则前后绑定的服务是否出现了变化，清理掉之前所绑定的信息数据
			if oldRule.IsServiceChange(cbRule) {
				// 从老的规则中获取所有的 svcKeys 信息列表
				svcKeys := getServicesInvolveByCircuitBreakerRule(oldRule)
				log.Info("[Cache][CircuitBreaker] clean rule bind old service info",
					zap.String("svc-keys", fmt.Sprintf("%#v", svcKeys)), zap.String("rule-id", cbRule.ID))
				// 挨个清空
				c.deleteCircuitBreakerFromServiceCache(cbRule.ID, svcKeys)
			}
		}
		svcKeys := getServicesInvolveByCircuitBreakerRule(cbRule)
		if !cbRule.Valid {
			del++
			c.rules.Delete(cbRule.ID)
			c.deleteCircuitBreakerFromServiceCache(cbRule.ID, svcKeys)
			continue
		}
		upsert++
		c.rules.Store(cbRule.ID, cbRule)
		c.storeCircuitBreakerToServiceCache(cbRule, svcKeys)
	}

	return map[string]time.Time{
		c.Name(): time.Unix(lastMtime, 0),
	}, upsert, del
}

// GetCircuitBreakerCount 获取熔断规则总数
func (c *circuitBreakerCache) GetCircuitBreakerCount() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	names := make(map[string]bool)
	c.allWildcardRules.IterateCircuitBreakerRules(func(rule *model.CircuitBreakerRule) {
		names[rule.Name] = true
	})
	for _, rules := range c.nsWildcardRules {
		rules.IterateCircuitBreakerRules(func(rule *model.CircuitBreakerRule) {
			names[rule.Name] = true
		})
	}
	for _, values := range c.circuitBreakers {
		for _, rules := range values {
			rules.IterateCircuitBreakerRules(func(rule *model.CircuitBreakerRule) {
				names[rule.Name] = true
			})
		}
	}
	return len(names)
}

// Query implements api.CircuitBreakerCache.
func (c *circuitBreakerCache) Query(context.Context, *types.CircuitBreakerRuleArgs) (uint32, []*model.CircuitBreakerRule, error) {
	panic("unimplemented")
}

// GetRule implements api.FaultDetectCache.
func (f *circuitBreakerCache) GetRule(id string) *model.CircuitBreakerRule {
	rule, _ := f.rules.Load(id)
	return rule
}
