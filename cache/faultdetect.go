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

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
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
	lastMtime        int64
	firstUpdate      bool

	lastCheckAllTime int64
	faultDetectCount int64

	singleFlight singleflight.Group
}

// init 自注册到缓存列表
func init() {
	RegisterCache(FaultDetectRuleName, CacheFaultDetector)
}

// newFaultDetectCache faultDetectCache constructor
func newFaultDetectCache(s store.Store) *faultDetectCache {
	return &faultDetectCache{
		baseCache:        newBaseCache(),
		storage:          s,
		svcSpecificRules: make(map[string]map[string]*model.ServiceWithFaultDetectRules),
		nsWildcardRules:  make(map[string]*model.ServiceWithFaultDetectRules),
		allWildcardRules: model.NewServiceWithFaultDetectRules(model.ServiceKey{
			Namespace: allMatched,
			Name:      allMatched,
		}),
	}
}

// initialize 实现Cache接口的函数
func (f *faultDetectCache) initialize(_ map[string]interface{}) error {
	f.lastMtime = 0
	f.firstUpdate = true
	f.lastCheckAllTime = 0
	f.faultDetectCount = 0
	return nil
}

func (f *faultDetectCache) update() error {
	_, err, _ := f.singleFlight.Do(InstanceName, func() (interface{}, error) {
		curStoreTime, err := f.storage.GetUnixSecond()
		if err != nil {
			curStoreTime = f.lastMtime
			log.Warn("[Cache][FaultDetect] get store timestamp fail, skip update lastMtime", zap.Error(err))
		}
		defer func() {
			f.lastMtime = curStoreTime
			f.checkAll()
		}()
		return nil, f.realUpdate()
	})
	return err
}

func (f *faultDetectCache) checkAll() {
	curTimeSec := time.Now().Unix()
	if curTimeSec-f.lastCheckAllTime < checkAllIntervalSec {
		return
	}
	defer func() {
		f.lastCheckAllTime = curTimeSec
	}()
	count, err := f.storage.GetFaultDetectCount()
	if err != nil {
		log.Errorf("[Cache][FaultDetect] get faultdetect count from storage err: %s", err.Error())
		return
	}
	if f.faultDetectCount == int64(count) {
		return
	}
	log.Infof(
		"[Cache][FaultDetect] faultdetect count not match, expect %d, actual %d, fallback to load all",
		count, f.faultDetectCount)
	f.lastMtime = 0
}

// update 实现Cache接口的函数
func (f *faultDetectCache) realUpdate() error {
	lastTime := time.Unix(f.lastMtime, 0)
	fdRules, err := f.storage.GetFaultDetectRulesForCache(lastTime.Add(DefaultTimeDiff), f.firstUpdate)
	if err != nil {
		log.Errorf("[Cache] fault detect config cache update err:%s", err.Error())
		return err
	}
	f.firstUpdate = false
	return f.setFaultDetectRules(fdRules)
}

// clear 实现Cache接口的函数
func (f *faultDetectCache) clear() error {
	f.lock.Lock()
	f.allWildcardRules.Clear()
	f.nsWildcardRules = make(map[string]*model.ServiceWithFaultDetectRules)
	f.svcSpecificRules = make(map[string]map[string]*model.ServiceWithFaultDetectRules)
	f.lock.Unlock()

	f.lastMtime = 0
	f.lastCheckAllTime = 0
	f.firstUpdate = true
	f.faultDetectCount = 0
	return nil
}

// name 实现资源名称
func (f *faultDetectCache) name() string {
	return FaultDetectRuleName
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
	log.Infof("svcSpecificRules for ns %s, values %v", name, svcRules)
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
		if svcKey.Name == allMatched {
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
		if svcKey.Name == allMatched {
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
func (f *faultDetectCache) setFaultDetectRules(fdRules []*model.FaultDetectRule) error {
	if len(fdRules) == 0 {
		return nil
	}

	for _, fdRule := range fdRules {
		svcKeys := getServicesInvolveByFaultDetectRule(fdRule)
		if !fdRule.Valid {
			f.deleteFaultDetectRuleFromServiceCache(fdRule.ID, svcKeys)
			continue
		}
		f.storeFaultDetectRuleToServiceCache(fdRule, svcKeys)
	}
	return nil
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
