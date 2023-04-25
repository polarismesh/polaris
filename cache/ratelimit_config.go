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
	"encoding/json"
	"sync"
	"time"

	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var (
	_ RateLimitCache = (*rateLimitCache)(nil)
)

const (
	// RateLimitConfigName rate limit config name
	RateLimitConfigName = "rateLimitConfig"
)

// RateLimitIterProc rate limit iter func
type RateLimitIterProc func(rateLimit *model.RateLimit)

// RateLimitCache rateLimit的cache接口
type RateLimitCache interface {
	Cache
	// GetRateLimit 根据serviceID进行迭代回调
	IteratorRateLimit(rateLimitIterProc RateLimitIterProc)
	// GetRateLimitRules 根据serviceID获取限流数据
	GetRateLimitRules(serviceKey model.ServiceKey) ([]*model.RateLimit, string)
	// QueryRateLimitRules
	QueryRateLimitRules(args RateLimitRuleArgs) (uint32, []*model.RateLimit, error)
	// GetRateLimitsCount 获取限流规则总数
	GetRateLimitsCount() int
}

// rateLimitCache的实现
type rateLimitCache struct {
	*baseCache

	lock         sync.RWMutex
	waitFixRules map[string]struct{}
	svcCache     ServiceCache
	storage      store.Store
	rules        *rateLimitRuleBucket
	singleFlight singleflight.Group
}

// init 自注册到缓存列表
func init() {
	RegisterCache(RateLimitConfigName, CacheRateLimit)
}

// newRateLimitCache 返回一个操作RateLimitCache的对象
func newRateLimitCache(s store.Store, sc ServiceCache) *rateLimitCache {
	return &rateLimitCache{
		baseCache:    newBaseCache(s),
		storage:      s,
		svcCache:     sc,
		waitFixRules: map[string]struct{}{},
	}
}

// initialize 实现Cache接口的initialize函数
func (rlc *rateLimitCache) initialize(_ map[string]interface{}) error {
	rlc.rules = newRateLimitRuleBucket()
	return nil
}

// update 实现Cache接口的update函数
func (rlc *rateLimitCache) update() error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := rlc.singleFlight.Do(rlc.name(), func() (interface{}, error) {
		return nil, rlc.doCacheUpdate(rlc.name(), rlc.realUpdate)
	})

	return err
}

func (rlc *rateLimitCache) realUpdate() (map[string]time.Time, int64, error) {
	rateLimits, err := rlc.storage.GetRateLimitsForCache(rlc.LastFetchTime(), rlc.isFirstUpdate())
	if err != nil {
		log.Errorf("[Cache] rate limit cache update err: %s", err.Error())
		return nil, -1, err
	}
	rlc.setRateLimit(rateLimits)
	return nil, int64(len(rateLimits)), err
}

// name 获取资源名称
func (rlc *rateLimitCache) name() string {
	return RateLimitConfigName
}

// clear 实现Cache接口的clear函数
func (rlc *rateLimitCache) clear() error {
	rlc.baseCache.clear()
	rlc.rules = newRateLimitRuleBucket()
	return nil
}

func (rlc *rateLimitCache) rateLimitToProto(rateLimit *model.RateLimit) error {
	rateLimit.Proto = &apitraffic.Rule{}
	if len(rateLimit.Rule) == 0 {
		return nil
	}
	// 反序列化rule
	if err := json.Unmarshal([]byte(rateLimit.Rule), rateLimit.Proto); err != nil {
		return err
	}
	namespace := rateLimit.Proto.GetNamespace().GetValue()
	name := rateLimit.Proto.GetService().GetValue()
	if namespace == "" || name == "" {
		rlc.fixRuleServiceInfo(rateLimit)
	}
	return rateLimit.AdaptArgumentsAndLabels()
}

// setRateLimit 更新限流规则到缓存中
func (rlc *rateLimitCache) setRateLimit(rateLimits []*model.RateLimit) map[string]time.Time {
	if len(rateLimits) == 0 {
		return nil
	}
	rlc.fixRulesServiceInfo()
	updateService := map[model.ServiceKey]struct{}{}
	lastMtime := rlc.LastMtime(rlc.name()).Unix()
	for _, item := range rateLimits {
		if err := rlc.rateLimitToProto(item); nil != err {
			log.Errorf("[Cache]fail to unmarshal rule to proto, err: %v", err)
			continue
		}
		if item.ModifyTime.Unix() > lastMtime {
			lastMtime = item.ModifyTime.Unix()
		}

		key := model.ServiceKey{
			Namespace: item.Proto.GetNamespace().GetValue(),
			Name:      item.Proto.GetService().GetValue(),
		}
		updateService[key] = struct{}{}

		// 待删除的rateLimit
		if !item.Valid {
			rlc.rules.delRule(item)
			rlc.deleteWaitFixRule(item)
			continue
		}
		rlc.rules.saveRule(item)
	}

	for serviceKey := range updateService {
		rlc.rules.reloadRevision(serviceKey)
	}

	return map[string]time.Time{
		rlc.name(): time.Unix(lastMtime, 0),
	}
}

// IteratorRateLimit 根据serviceID进行迭代回调
func (rlc *rateLimitCache) IteratorRateLimit(proc RateLimitIterProc) {
	rlc.rules.foreach(proc)
}

// GetRateLimitByServiceID 根据serviceID获取限流数据
func (rlc *rateLimitCache) GetRateLimitRules(serviceKey model.ServiceKey) ([]*model.RateLimit, string) {
	rules, revision := rlc.rules.getRules(serviceKey)
	return rules, revision
}

// GetRateLimitsCount 获取限流规则总数
func (rlc *rateLimitCache) GetRateLimitsCount() int {
	return rlc.rules.count()
}

func (rlc *rateLimitCache) deleteWaitFixRule(rule *model.RateLimit) {
	rlc.lock.Lock()
	defer rlc.lock.Unlock()
	delete(rlc.waitFixRules, rule.ID)
}

func (rlc *rateLimitCache) fixRulesServiceInfo() {
	rlc.lock.Lock()
	defer rlc.lock.Unlock()
	for id := range rlc.waitFixRules {
		rule := rlc.rules.getRuleByID(id)
		if rule == nil {
			delete(rlc.waitFixRules, id)
			continue
		}
		svcId := rule.ServiceID
		svc := rlc.svcCache.GetServiceByID(svcId)
		if svc == nil {
			svc2, err := rlc.storage.GetServiceByID(svcId)
			if err != nil {
				continue
			}
			svc = svc2
		}

		rule.Proto.Namespace = utils.NewStringValue(svc.Namespace)
		rule.Proto.Name = utils.NewStringValue(svc.Name)
		delete(rlc.waitFixRules, rule.ID)
	}
}

func (rlc *rateLimitCache) fixRuleServiceInfo(rateLimit *model.RateLimit) {
	rlc.lock.Lock()
	defer rlc.lock.Unlock()
	svcId := rateLimit.ServiceID
	svc := rlc.svcCache.GetServiceByID(svcId)
	if svc == nil {
		svc2, err := rlc.storage.GetServiceByID(svcId)
		if err != nil {
			rlc.waitFixRules[rateLimit.ID] = struct{}{}
			return
		}
		svc = svc2
	}

	rateLimit.Proto.Namespace = utils.NewStringValue(svc.Namespace)
	rateLimit.Proto.Name = utils.NewStringValue(svc.Name)
	delete(rlc.waitFixRules, rateLimit.ID)
}
