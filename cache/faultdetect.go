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
	// FaultDetectRuleName fault detect config name
	FaultDetectRuleName = "faultDetectRule"
)

var _ FaultDetectCache = (*faultDetectCache)(nil)

// FaultDetectCache  fault detect rule cache service
type FaultDetectCache interface {
	Cache

	// GetFaultDetectConfig 根据ServiceID获取探测配置
	GetFaultDetectConfig(svcName string, namespace string) *model.ServiceWithFaultDetectRules
}

type faultDetectCache struct {
	*baseCache

	storage store.Store
	// increment cache
	// fetched service cache
	// key1: namespace, key2: service
	svcSpecificRules map[string]map[string]*model.ServiceWithFaultDetectRules
	// key1: namespace
	nsWildcardRules map[string]*model.ServiceWithFaultDetectRules
	// all rules are wildcard specific
	allWildcardRules *model.ServiceWithFaultDetectRules
	lock             sync.RWMutex

	singleFlight singleflight.Group
}

// init 自注册到缓存列表
func init() {
	RegisterCache(FaultDetectRuleName, CacheFaultDetector)
}

// newFaultDetectCache faultDetectCache constructor
func newFaultDetectCache(s store.Store) *faultDetectCache {
	fdCache := &faultDetectCache{
		baseCache:        newBaseCache(s),
		storage:          s,
		svcSpecificRules: make(map[string]map[string]*model.ServiceWithFaultDetectRules),
		nsWildcardRules:  make(map[string]*model.ServiceWithFaultDetectRules),
		allWildcardRules: model.NewServiceWithFaultDetectRules(model.ServiceKey{
			Namespace: allMatched,
			Name:      allMatched,
		}),
	}
	fdCache.reloadRevision(fdCache.allWildcardRules)
	return fdCache
}

// initialize 实现Cache接口的函数
func (f *faultDetectCache) initialize(_ map[string]interface{}) error {
	return nil
}

func (f *faultDetectCache) update() error {
	_, err, _ := f.singleFlight.Do(f.name(), func() (interface{}, error) {
		return nil, f.doCacheUpdate(f.name(), f.realUpdate)
	})
	return err
}

// update 实现Cache接口的函数
func (f *faultDetectCache) realUpdate() (map[string]time.Time, int64, error) {
	fdRules, err := f.storage.GetFaultDetectRulesForCache(f.LastFetchTime(), f.isFirstUpdate())
	if err != nil {
		log.Errorf("[Cache] fault detect config cache update err:%s", err.Error())
		return nil, -1, err
	}
	lastMtimes := f.setFaultDetectRules(fdRules)

	return lastMtimes, int64(len(fdRules)), nil
}

// clear 实现Cache接口的函数
func (f *faultDetectCache) clear() error {
	f.baseCache.clear()
	f.lock.Lock()
	f.allWildcardRules.Clear()
	f.nsWildcardRules = make(map[string]*model.ServiceWithFaultDetectRules)
	f.svcSpecificRules = make(map[string]map[string]*model.ServiceWithFaultDetectRules)
	f.reloadRevision(f.allWildcardRules)
	f.lock.Unlock()
	return nil
}

// name 实现资源名称
func (f *faultDetectCache) name() string {
	return FaultDetectRuleName
}

// GetFaultDetectConfig 根据serviceID获取探测规则
func (f *faultDetectCache) GetFaultDetectConfig(name string, namespace string) *model.ServiceWithFaultDetectRules {
	// check service specific
	rules := f.checkServiceSpecificCache(name, namespace)
	if nil != rules {
		return rules
	}
	rules = f.checkNamespaceSpecificCache(namespace)
	if nil != rules {
		return rules
	}
	return f.allWildcardRules
}

func (f *faultDetectCache) checkServiceSpecificCache(
	name string, namespace string) *model.ServiceWithFaultDetectRules {
	f.lock.RLock()
	defer f.lock.RUnlock()
	log.Infof(
		"checkServiceSpecificCache name %s, namespace %s, values %v", name, namespace, f.svcSpecificRules)
	svcRules, ok := f.svcSpecificRules[namespace]
	if ok {
		return svcRules[name]
	}
	return nil
}

func (f *faultDetectCache) checkNamespaceSpecificCache(namespace string) *model.ServiceWithFaultDetectRules {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.nsWildcardRules[namespace]
}

func (f *faultDetectCache) reloadRevision(svcRules *model.ServiceWithFaultDetectRules) {
	rulesCount := svcRules.CountFaultDetectRules()
	revisions := make([]string, 0, rulesCount+2)
	revisions = append(revisions, svcRules.Service.Namespace)
	revisions = append(revisions, svcRules.Service.Name)
	svcRules.IterateFaultDetectRules(func(rule *model.FaultDetectRule) {
		revisions = append(revisions, rule.Revision)
	})
	sort.Strings(revisions)
	h := sha1.New()
	revision, err := ComputeRevisionBySlice(h, revisions)
	if err != nil {
		log.Errorf("[Server][Service][FaultDetector] compute revision service(%s) err: %s",
			svcRules.Service, err.Error())
		return
	}
	svcRules.Revision = revision
}

func (f *faultDetectCache) deleteAndReloadFaultDetectRules(svcRules *model.ServiceWithFaultDetectRules, id string) {
	svcRules.DelFaultDetectRule(id)
	f.reloadRevision(svcRules)
}

func (f *faultDetectCache) deleteFaultDetectRuleFromServiceCache(id string, svcKeys map[model.ServiceKey]bool) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if len(svcKeys) == 0 {
		// all wildcard
		log.Infof("[Server][Service][FaultDetector] delete rule %s from all matched cache", id)
		f.deleteAndReloadFaultDetectRules(f.allWildcardRules, id)
		for ns, rules := range f.nsWildcardRules {
			log.Infof("[Server][Service][FaultDetector] delete rule %s from ns specified cache, "+
				"namespace %s", id, ns)
			f.deleteAndReloadFaultDetectRules(rules, id)
		}
		for ns, svcRules := range f.svcSpecificRules {
			for svc, rules := range svcRules {
				log.Infof("[Server][Service][FaultDetector] delete rule %s from svc specified cache, "+
					"namespace %s, service %s", id, ns, svc)
				f.deleteAndReloadFaultDetectRules(rules, id)
			}
		}
		return
	}
	svcToReloads := make(map[model.ServiceKey]bool)
	for svcKey := range svcKeys {
		if svcKey.Name == allMatched {
			rules, ok := f.nsWildcardRules[svcKey.Namespace]
			if ok {
				log.Infof("[Server][Service][FaultDetector] delete rule %s from ns specified cache, "+
					"namespace %s", id, svcKey.Namespace)
				f.deleteAndReloadFaultDetectRules(rules, id)
			}
			svcRules, ok := f.svcSpecificRules[svcKey.Namespace]
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
			svcRules, ok := f.svcSpecificRules[svcToReload.Namespace]
			if ok {
				rules, ok := svcRules[svcToReload.Name]
				if ok {
					log.Infof("[Server][Service][FaultDetector] delete rule %s from svc specified cache, "+
						"namespace %s, service %s", id, svcToReload.Namespace, svcToReload.Name)
					f.deleteAndReloadFaultDetectRules(rules, id)
				}
			}
		}
	}
}

func (f *faultDetectCache) storeAndReloadFaultDetectRules(
	svcRules *model.ServiceWithFaultDetectRules, cbRule *model.FaultDetectRule) {
	svcRules.AddFaultDetectRule(cbRule)
	f.reloadRevision(svcRules)
}

func createAndStoreServiceWithFaultDetectRules(svcKey model.ServiceKey, key string,
	values map[string]*model.ServiceWithFaultDetectRules) *model.ServiceWithFaultDetectRules {
	rules := model.NewServiceWithFaultDetectRules(svcKey)
	values[key] = rules
	return rules
}

func (f *faultDetectCache) storeFaultDetectRuleToServiceCache(
	entry *model.FaultDetectRule, svcKeys map[model.ServiceKey]bool) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if len(svcKeys) == 0 {
		// all wildcard
		log.Infof("[Server][Service][FaultDetector] add rule %s from all matched cache", entry.ID)
		f.storeAndReloadFaultDetectRules(f.allWildcardRules, entry)
		for ns, rules := range f.nsWildcardRules {
			log.Infof("[Server][Service][FaultDetector] add rule %s from ns specific cache, namespace %s",
				entry.ID, ns)
			f.storeAndReloadFaultDetectRules(rules, entry)
		}
		for ns, svcRules := range f.svcSpecificRules {
			for svcName, rules := range svcRules {
				log.Infof("[Server][Service][FaultDetector] add rule %s from svc specific cache, "+
					"namespace %s, service %s", entry.ID, ns, svcName)
				f.storeAndReloadFaultDetectRules(rules, entry)
			}
		}
		return
	}
	svcToReloads := make(map[model.ServiceKey]bool)
	for svcKey := range svcKeys {
		if svcKey.Name == allMatched {
			var wildcardRules *model.ServiceWithFaultDetectRules
			var ok bool
			wildcardRules, ok = f.nsWildcardRules[svcKey.Namespace]
			if !ok {
				wildcardRules = createAndStoreServiceWithFaultDetectRules(svcKey, svcKey.Namespace, f.nsWildcardRules)
			}
			log.Infof("[Server][Service][FaultDetector] add rule %s from ns specific cache, namespace %s",
				entry.ID, svcKey.Namespace)
			f.storeAndReloadFaultDetectRules(wildcardRules, entry)
			svcRules, ok := f.svcSpecificRules[svcKey.Namespace]
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
			var rules *model.ServiceWithFaultDetectRules
			var svcRules map[string]*model.ServiceWithFaultDetectRules
			var ok bool
			svcRules, ok = f.svcSpecificRules[svcToReload.Namespace]
			if !ok {
				svcRules = make(map[string]*model.ServiceWithFaultDetectRules)
				f.svcSpecificRules[svcToReload.Namespace] = svcRules
			}
			rules, ok = svcRules[svcToReload.Name]
			if !ok {
				rules = createAndStoreServiceWithFaultDetectRules(svcToReload, svcToReload.Name, svcRules)
			}
			log.Infof("[Server][Service][FaultDetector] add rule %s from svc specific cache, "+
				"namespace %s, service %s", entry.ID, svcToReload.Namespace, svcToReload.Name)
			f.storeAndReloadFaultDetectRules(rules, entry)
		}
	}
}

func getServicesInvolveByFaultDetectRule(fdRule *model.FaultDetectRule) map[model.ServiceKey]bool {
	svcKeys := make(map[model.ServiceKey]bool)
	addService := func(name string, namespace string) {
		if name == allMatched && namespace == allMatched {
			return
		}
		svcKeys[model.ServiceKey{
			Namespace: namespace,
			Name:      name,
		}] = true
	}
	addService(fdRule.DstService, fdRule.DstNamespace)
	return svcKeys
}

// setCircuitBreaker 更新store的数据到cache中
func (f *faultDetectCache) setFaultDetectRules(fdRules []*model.FaultDetectRule) map[string]time.Time {
	if len(fdRules) == 0 {
		return nil
	}

	lastMtime := f.LastMtime(f.name()).Unix()

	for _, fdRule := range fdRules {
		if fdRule.ModifyTime.Unix() > lastMtime {
			lastMtime = fdRule.ModifyTime.Unix()
		}
		svcKeys := getServicesInvolveByFaultDetectRule(fdRule)
		if !fdRule.Valid {
			f.deleteFaultDetectRuleFromServiceCache(fdRule.ID, svcKeys)
			continue
		}
		f.storeFaultDetectRuleToServiceCache(fdRule, svcKeys)
	}

	return map[string]time.Time{
		f.name(): time.Unix(lastMtime, 0),
	}
}

// GetFaultDetectRuleCount 获取探测规则总数
func (f *faultDetectCache) GetFaultDetectRuleCount(fun func(k, v interface{}) bool) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	for k, v := range f.svcSpecificRules {
		if !fun(k, v) {
			break
		}
	}
}
