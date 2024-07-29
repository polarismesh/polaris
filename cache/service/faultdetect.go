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

type faultDetectCache struct {
	*types.BaseCache

	storage store.Store
	// rules record id -> *model.FaultDetectRule
	rules *utils.SyncMap[string, *model.FaultDetectRule]
	// increment cache
	// fetched service cache
	// key1: namespace, key2: service
	svcSpecificRules map[string]map[string]*model.ServiceWithFaultDetectRules
	// key1: namespace
	nsWildcardRules map[string]*model.ServiceWithFaultDetectRules
	// all rules are wildcard specific
	allWildcardRules *model.ServiceWithFaultDetectRules
	lock             sync.RWMutex
	singleFlight     singleflight.Group
}

// NewFaultDetectCache faultDetectCache constructor
func NewFaultDetectCache(s store.Store, cacheMgr types.CacheManager) types.FaultDetectCache {
	return &faultDetectCache{
		BaseCache:        types.NewBaseCache(s, cacheMgr),
		storage:          s,
		rules:            utils.NewSyncMap[string, *model.FaultDetectRule](),
		svcSpecificRules: make(map[string]map[string]*model.ServiceWithFaultDetectRules),
		nsWildcardRules:  make(map[string]*model.ServiceWithFaultDetectRules),
		allWildcardRules: model.NewServiceWithFaultDetectRules(model.ServiceKey{
			Namespace: types.AllMatched,
			Name:      types.AllMatched,
		}),
	}
}

// Initialize 实现Cache接口的函数
func (f *faultDetectCache) Initialize(_ map[string]interface{}) error {
	return nil
}

func (f *faultDetectCache) Update() error {
	_, err, _ := f.singleFlight.Do(f.Name(), func() (interface{}, error) {
		return nil, f.DoCacheUpdate(f.Name(), f.realUpdate)
	})
	return err
}

// update 实现Cache接口的函数
func (f *faultDetectCache) realUpdate() (map[string]time.Time, int64, error) {
	fdRules, err := f.storage.GetFaultDetectRulesForCache(f.LastFetchTime(), f.IsFirstUpdate())
	if err != nil {
		log.Errorf("[Cache] fault detect config cache update err:%s", err.Error())
		return nil, -1, err
	}
	lastMtimes := f.setFaultDetectRules(fdRules)

	return lastMtimes, int64(len(fdRules)), nil
}

// clear 实现Cache接口的函数
func (f *faultDetectCache) Clear() error {
	f.BaseCache.Clear()
	f.lock.Lock()
	f.allWildcardRules.Clear()
	f.rules = utils.NewSyncMap[string, *model.FaultDetectRule]()
	f.nsWildcardRules = make(map[string]*model.ServiceWithFaultDetectRules)
	f.svcSpecificRules = make(map[string]map[string]*model.ServiceWithFaultDetectRules)
	f.lock.Unlock()
	return nil
}

// Name 实现资源名称
func (f *faultDetectCache) Name() string {
	return types.FaultDetectRuleName
}

// GetFaultDetectConfig 根据serviceID获取探测规则
func (f *faultDetectCache) GetFaultDetectConfig(name string, namespace string) *model.ServiceWithFaultDetectRules {
	log.Infof("GetFaultDetectConfig: name %s, namespace %s", name, namespace)
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
	if rulesCount == 0 {
		svcRules.Revision = ""
		return
	}
	revisions := make([]string, 0, rulesCount)
	svcRules.IterateFaultDetectRules(func(rule *model.FaultDetectRule) {
		revisions = append(revisions, rule.Revision)
	})
	sort.Strings(revisions)
	h := sha1.New()
	revision, err := types.ComputeRevisionBySlice(h, revisions)
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
		f.deleteAndReloadFaultDetectRules(f.allWildcardRules, id)
		for _, rules := range f.nsWildcardRules {
			f.deleteAndReloadFaultDetectRules(rules, id)
		}
		for _, svcRules := range f.svcSpecificRules {
			for _, rules := range svcRules {
				f.deleteAndReloadFaultDetectRules(rules, id)
			}
		}
		return
	}
	svcToReloads := make(map[model.ServiceKey]bool)
	for svcKey := range svcKeys {
		if svcKey.Name == types.AllMatched {
			rules, ok := f.nsWildcardRules[svcKey.Namespace]
			if ok {
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
		f.storeAndReloadFaultDetectRules(f.allWildcardRules, entry)
		for _, rules := range f.nsWildcardRules {
			f.storeAndReloadFaultDetectRules(rules, entry)
		}
		for _, svcRules := range f.svcSpecificRules {
			for _, rules := range svcRules {
				f.storeAndReloadFaultDetectRules(rules, entry)
			}
		}
		return
	}
	svcToReloads := make(map[model.ServiceKey]bool)
	for svcKey := range svcKeys {
		if svcKey.Name == types.AllMatched {
			var wildcardRules *model.ServiceWithFaultDetectRules
			var ok bool
			wildcardRules, ok = f.nsWildcardRules[svcKey.Namespace]
			if !ok {
				wildcardRules = createAndStoreServiceWithFaultDetectRules(svcKey, svcKey.Namespace, f.nsWildcardRules)
			}
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
			f.storeAndReloadFaultDetectRules(rules, entry)
		}
	}
}

func getServicesInvolveByFaultDetectRule(fdRule *model.FaultDetectRule) map[model.ServiceKey]bool {
	svcKeys := make(map[model.ServiceKey]bool)
	addService := func(name string, namespace string) {
		if name == types.AllMatched && namespace == types.AllMatched {
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

	lastMtime := f.LastMtime(f.Name()).Unix()

	for _, fdRule := range fdRules {
		oldRule, ok := f.rules.Load(fdRule.ID)
		if ok {
			// 对比规则前后绑定的服务是否出现了变化，清理掉之前所绑定的信息数据
			if oldRule.IsServiceChange(fdRule) {
				// 从老的规则中获取所有的 svcKeys 信息列表
				svcKeys := getServicesInvolveByFaultDetectRule(oldRule)
				log.Info("[Cache][FaultDetect] clean rule bind old service info",
					zap.String("svc-keys", fmt.Sprintf("%#v", svcKeys)), zap.String("rule-id", fdRule.ID))
				// 挨个清空
				f.deleteFaultDetectRuleFromServiceCache(fdRule.ID, svcKeys)
			}
		}

		if fdRule.ModifyTime.Unix() > lastMtime {
			lastMtime = fdRule.ModifyTime.Unix()
		}
		svcKeys := getServicesInvolveByFaultDetectRule(fdRule)
		if !fdRule.Valid {
			f.rules.Delete(fdRule.ID)
			f.deleteFaultDetectRuleFromServiceCache(fdRule.ID, svcKeys)
			continue
		}
		f.rules.Store(fdRule.ID, fdRule)
		f.storeFaultDetectRuleToServiceCache(fdRule, svcKeys)
	}

	return map[string]time.Time{
		f.Name(): time.Unix(lastMtime, 0),
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

// Query implements api.FaultDetectCache.
func (f *faultDetectCache) Query(context.Context, *types.FaultDetectArgs) (uint32, []*model.FaultDetectRule, error) {
	panic("unimplemented")
}

// GetRule implements api.FaultDetectCache.
func (f *faultDetectCache) GetRule(id string) *model.FaultDetectRule {
	rule, _ := f.rules.Load(id)
	return rule
}
