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
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"sync"

	"go.uber.org/zap"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

func newRateLimitRuleBucket() *RateLimitRuleContainer {
	return &RateLimitRuleContainer{
		ids:   utils.NewSyncMap[string, *model.RateLimit](),
		rules: utils.NewSyncMap[string, *subRateLimitRuleBucket](),
	}
}

type RateLimitRuleContainer struct {
	ids   *utils.SyncMap[string, *model.RateLimit]
	rules *utils.SyncMap[string, *subRateLimitRuleBucket]
}

func (r *RateLimitRuleContainer) foreach(proc types.RateLimitIterProc) {
	r.rules.Range(func(key string, val *subRateLimitRuleBucket) {
		val.foreach(proc)
	})
}

func (r *RateLimitRuleContainer) count() int {
	return r.ids.Len()
}

func (r *RateLimitRuleContainer) saveRule(rule *model.RateLimit) {
	r.cleanOldSvcRule(rule)

	r.ids.Store(rule.ID, rule)
	key := (&model.ServiceKey{
		Namespace: rule.Proto.GetNamespace().GetValue(),
		Name:      rule.Proto.GetService().GetValue(),
	}).Domain()

	if _, ok := r.rules.Load(key); !ok {
		r.rules.Store(key, &subRateLimitRuleBucket{
			rules: map[string]*model.RateLimit{},
		})
	}

	b, _ := r.rules.Load(key)
	b.saveRule(rule)
}

// cleanOldSvcRule 清理规则之前绑定的服务数据信息
func (r *RateLimitRuleContainer) cleanOldSvcRule(rule *model.RateLimit) {
	oldRule, ok := r.ids.Load(rule.ID)
	if !ok {
		return
	}

	// 清理原来老记录的绑定数据信息
	key := (&model.ServiceKey{
		Namespace: oldRule.Proto.GetNamespace().GetValue(),
		Name:      oldRule.Proto.GetService().GetValue(),
	}).Domain()
	bucket, ok := r.rules.Load(key)
	if !ok {
		return
	}
	// 删除服务绑定的限流规则信息
	bucket.delRule(rule)
	if bucket.count() == 0 {
		r.rules.Delete(key)
	}
}

func (r *RateLimitRuleContainer) delRule(rule *model.RateLimit) {
	r.cleanOldSvcRule(rule)
	r.ids.Delete(rule.ID)

	key := (&model.ServiceKey{
		Namespace: rule.Proto.GetNamespace().GetValue(),
		Name:      rule.Proto.GetService().GetValue(),
	}).Domain()
	if _, ok := r.rules.Load(key); !ok {
		return
	}

	b, _ := r.rules.Load(key)
	b.delRule(rule)
	if b.count() == 0 {
		r.rules.Delete(key)
	}
}

func (r *RateLimitRuleContainer) getRuleByID(id string) *model.RateLimit {
	ret, _ := r.ids.Load(id)
	return ret
}

func (r *RateLimitRuleContainer) getRules(serviceKey model.ServiceKey) ([]*model.RateLimit, string) {
	key := (&serviceKey).Domain()
	if _, ok := r.rules.Load(key); !ok {
		return nil, ""
	}

	b, _ := r.rules.Load(key)
	return b.toSlice(), b.revision
}

func (r *RateLimitRuleContainer) reloadRevision(serviceKey model.ServiceKey) {
	key := serviceKey.Domain()
	v, ok := r.rules.Load(key)
	if !ok {
		return
	}
	v.reloadRevision()
}

type subRateLimitRuleBucket struct {
	lock     sync.RWMutex
	revision string
	rules    map[string]*model.RateLimit
}

func (r *subRateLimitRuleBucket) saveRule(rule *model.RateLimit) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.rules[rule.ID] = rule
}

func (r *subRateLimitRuleBucket) delRule(rule *model.RateLimit) {
	r.lock.Lock()
	defer r.lock.Unlock()

	delete(r.rules, rule.ID)
}

func (r *subRateLimitRuleBucket) foreach(proc types.RateLimitIterProc) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	for _, v := range r.rules {
		proc(v)
	}
}

func (r *subRateLimitRuleBucket) toSlice() []*model.RateLimit {
	r.lock.RLock()
	defer r.lock.RUnlock()

	ret := make([]*model.RateLimit, 0, len(r.rules))
	for i := range r.rules {
		ret = append(ret, r.rules[i])
	}
	return ret
}

func (r *subRateLimitRuleBucket) count() int {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return len(r.rules)
}

func (r *subRateLimitRuleBucket) reloadRevision() {
	r.lock.Lock()
	defer r.lock.Unlock()

	revisions := make([]string, 0, len(r.rules))
	for i := range r.rules {
		revisions = append(revisions, r.rules[i].Revision)
	}

	sort.Strings(revisions)
	h := sha1.New()
	for i := range revisions {
		if _, err := h.Write([]byte(revisions[i])); err != nil {
			log.Error("[Cache][RateLimit] rebuild ratelimit rule revision", zap.Error(err))
			return
		}
	}

	r.revision = hex.EncodeToString(h.Sum(nil))
}
