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
	"crypto/sha1"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

const (
	// CircuitBreakerName circuit breaker config name
	CircuitBreakerName = "circuitBreakerConfig"
)

var _ CircuitBreakerCache = (*circuitBreakerCache)(nil)

// CircuitBreakerCache  circuitBreaker配置的cache接口
type CircuitBreakerCache interface {
	Cache

	// GetCircuitBreakerConfig 根据ServiceID获取熔断配置
	GetCircuitBreakerConfig(svcName string, namespace string) *model.ServiceWithCircuitBreakerRules
}

// circuitBreaker的实现
type circuitBreakerCache struct {
	*baseCache

	storage store.Store
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

// init 自注册到缓存列表
func init() {
	RegisterCache(CircuitBreakerName, CacheCircuitBreaker)
}

// newCircuitBreakerCache 返回一个操作CircuitBreakerCache的对象
func newCircuitBreakerCache(s store.Store) *circuitBreakerCache {
	cbCache := &circuitBreakerCache{
		baseCache:       newBaseCache(s),
		storage:         s,
		circuitBreakers: make(map[string]map[string]*model.ServiceWithCircuitBreakerRules),
		nsWildcardRules: make(map[string]*model.ServiceWithCircuitBreakerRules),
		allWildcardRules: model.NewServiceWithCircuitBreakerRules(model.ServiceKey{
			Namespace: allMatched,
			Name:      allMatched,
		}),
	}
	cbCache.reloadRevision(cbCache.allWildcardRules)
	return cbCache
}

// initialize 实现Cache接口的函数
func (c *circuitBreakerCache) initialize(_ map[string]interface{}) error {
	return nil
}

// update 实现Cache接口的函数
func (c *circuitBreakerCache) update() error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := c.singleFlight.Do(c.name(), func() (interface{}, error) {
		return nil, c.doCacheUpdate(c.name(), c.realUpdate)
	})
	return err
}

func (c *circuitBreakerCache) realUpdate() (map[string]time.Time, int64, error) {
	cbRules, err := c.storage.GetCircuitBreakerRulesForCache(c.LastFetchTime(), c.isFirstUpdate())
	if err != nil {
		log.Errorf("[Cache] circuit breaker config cache update err:%s", err.Error())
		return nil, -1, err
	}
	lastMtimes := c.setCircuitBreaker(cbRules)
	return lastMtimes, int64(len(cbRules)), nil
}

// clear 实现Cache接口的函数
func (c *circuitBreakerCache) clear() error {
	c.baseCache.clear()
	c.lock.Lock()
	c.allWildcardRules.Clear()
	c.nsWildcardRules = make(map[string]*model.ServiceWithCircuitBreakerRules)
	c.circuitBreakers = make(map[string]map[string]*model.ServiceWithCircuitBreakerRules)
	c.reloadRevision(c.allWildcardRules)
	c.lock.Unlock()
	return nil
}

// name 实现资源名称
func (c *circuitBreakerCache) name() string {
	return CircuitBreakerName
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
	revisions := make([]string, 0, rulesCount+2)
	revisions = append(revisions, svcRules.Service.Namespace)
	revisions = append(revisions, svcRules.Service.Name)
	svcRules.IterateCircuitBreakerRules(func(rule *model.CircuitBreakerRule) {
		revisions = append(revisions, rule.Revision)
	})
	sort.Strings(revisions)
	h := sha1.New()
	revision, err := ComputeRevisionBySlice(h, revisions)
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
		if svcKey.Name == allMatched {
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
		if svcKey.Name == allMatched {
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

const allMatched = "*"

func getServicesInvolveByCircuitBreakerRule(cbRule *model.CircuitBreakerRule) map[model.ServiceKey]bool {
	svcKeys := make(map[model.ServiceKey]bool)
	addService := func(name string, namespace string) {
		if len(name) == 0 && len(namespace) == 0 {
			return
		}
		if name == allMatched && namespace == allMatched {
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
func (c *circuitBreakerCache) setCircuitBreaker(cbRules []*model.CircuitBreakerRule) map[string]time.Time {
	if len(cbRules) == 0 {
		return nil
	}

	lastMtime := c.LastMtime(c.name()).Unix()

	for _, cbRule := range cbRules {
		if cbRule.ModifyTime.Unix() > lastMtime {
			lastMtime = cbRule.ModifyTime.Unix()
		}
		svcKeys := getServicesInvolveByCircuitBreakerRule(cbRule)
		if !cbRule.Valid {
			c.deleteCircuitBreakerFromServiceCache(cbRule.ID, svcKeys)
			continue
		}
		c.storeCircuitBreakerToServiceCache(cbRule, svcKeys)
	}

	return map[string]time.Time{
		c.name(): time.Unix(lastMtime, 0),
	}
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
