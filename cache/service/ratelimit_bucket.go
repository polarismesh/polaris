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
)

func newRateLimitRuleBucket() *rateLimitRuleBucket {
	return &rateLimitRuleBucket{
		ids:   map[string]*model.RateLimit{},
		rules: map[string]*subRateLimitRuleBucket{},
	}
}

type rateLimitRuleBucket struct {
	lock  sync.RWMutex
	ids   map[string]*model.RateLimit
	rules map[string]*subRateLimitRuleBucket
}

func (r *rateLimitRuleBucket) foreach(proc types.RateLimitIterProc) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	for _, b := range r.rules {
		b.foreach(proc)
	}
}

func (r *rateLimitRuleBucket) count() int {
	r.lock.RLock()
	defer r.lock.RUnlock()

	count := 0
	for _, b := range r.rules {
		count += b.count()
	}
	return count
}

func (r *rateLimitRuleBucket) saveRule(rule *model.RateLimit) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.cleanOldSvcRule(rule)

	r.ids[rule.ID] = rule
	key := buildServiceKey(rule.Proto.GetNamespace().GetValue(), rule.Proto.GetService().GetValue())

	if _, ok := r.rules[key]; !ok {
		r.rules[key] = &subRateLimitRuleBucket{
			rules: map[string]*model.RateLimit{},
		}
	}

	b := r.rules[key]
	b.saveRule(rule)
}

// cleanOldSvcRule 清理规则之前绑定的服务数据信息
func (r *rateLimitRuleBucket) cleanOldSvcRule(rule *model.RateLimit) {
	oldRule, ok := r.ids[rule.ID]
	if !ok {
		return
	}
	// 清理原来老记录的绑定数据信息
	key := buildServiceKey(oldRule.Proto.GetNamespace().GetValue(), oldRule.Proto.GetService().GetValue())
	bucket, ok := r.rules[key]
	if !ok {
		return
	}
	// 删除服务绑定的限流规则信息
	bucket.delRule(rule)
	if bucket.count() == 0 {
		delete(r.rules, key)
	}
}

func (r *rateLimitRuleBucket) delRule(rule *model.RateLimit) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.cleanOldSvcRule(rule)
	delete(r.ids, rule.ID)

	key := buildServiceKey(rule.Proto.GetNamespace().GetValue(), rule.Proto.GetService().GetValue())
	if _, ok := r.rules[key]; !ok {
		return
	}

	b := r.rules[key]
	b.delRule(rule)
	if b.count() == 0 {
		delete(r.rules, key)
	}
}

func (r *rateLimitRuleBucket) getRuleByID(id string) *model.RateLimit {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.ids[id]
}

func (r *rateLimitRuleBucket) getRules(serviceKey model.ServiceKey) ([]*model.RateLimit, string) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	key := buildServiceKey(serviceKey.Namespace, serviceKey.Name)
	if _, ok := r.rules[key]; !ok {
		return nil, ""
	}

	b := r.rules[key]
	return b.toSlice(), b.revision
}

func (r *rateLimitRuleBucket) reloadRevision(serviceKey model.ServiceKey) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	key := buildServiceKey(serviceKey.Namespace, serviceKey.Name)
	if _, ok := r.rules[key]; !ok {
		return
	}

	r.rules[key].reloadRevision()
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
