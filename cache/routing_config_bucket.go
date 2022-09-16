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
	"sync"

	apiv2 "github.com/polarismesh/polaris-server/common/api/v2"
	"github.com/polarismesh/polaris-server/common/model"
	v2 "github.com/polarismesh/polaris-server/common/model/v2"
)

type routingLevel int16

const (
	_ routingLevel = iota
	level1RoutingV2
	level2RoutingV2
	level3RoutingV2
)

func newRoutingBucketV1() *routingBucketV1 {
	return &routingBucketV1{
		rules: make(map[string]*model.RoutingConfig),
	}
}

func newRoutingBucketV2() *routingBucketV2 {
	return &routingBucketV2{
		rules:       make(map[string]*v2.ExtendRoutingConfig),
		level1Rules: map[string]map[string]struct{}{},
		level2Rules: map[string]map[string]struct{}{},
		level3Rules: map[string]struct{}{},
		v1rules:     map[string][]*v2.ExtendRoutingConfig{},
	}
}

// routingBucketV1
type routingBucketV1 struct {
	lock  sync.RWMutex
	rules map[string]*model.RoutingConfig
}

func (b *routingBucketV1) get(id string) *model.RoutingConfig {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.rules[id]
}

func (b *routingBucketV1) save(conf *model.RoutingConfig) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.rules[conf.ID] = conf
}

func (b *routingBucketV1) delete(id string) {
	b.lock.Lock()
	defer b.lock.Unlock()

	delete(b.rules, id)
}

func (b *routingBucketV1) size() int {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return len(b.rules)
}

// routingBucketV2 v2 路由规则缓存 bucket
type routingBucketV2 struct {
	lock sync.RWMutex
	// rules id => routing rule
	rules map[string]*v2.ExtendRoutingConfig
	// level1Rules service+namespace => 路由规则ID列表
	level1Rules map[string]map[string]struct{}
	// level2Rules 针对某个命名空间下所有服务都生效的路由规则
	level2Rules map[string]map[string]struct{}
	// level3Rules 针对所有命名空间下的所有服务都生效的规则
	level3Rules map[string]struct{}
	// v1rules v1 版本的规则自动转为 v2 版本的规则，用于 v2 接口的数据查看
	v1rules map[string][]*v2.ExtendRoutingConfig
}

func (b *routingBucketV2) get(id string) *v2.ExtendRoutingConfig {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.rules[id]
}

type serviceInfo interface {
	GetNamespace() string
	GetService() string
}

func (b *routingBucketV2) saveV2(conf *v2.ExtendRoutingConfig) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.rules[conf.ID] = conf

	handler := func(item serviceInfo) {
		if item.GetService() == "*" && item.GetNamespace() == "*" {
			b.level3Rules[conf.ID] = struct{}{}
			return
		}

		if item.GetService() == "*" && item.GetNamespace() != "*" {
			if _, ok := b.level2Rules[item.GetNamespace()]; !ok {
				b.level2Rules[item.GetNamespace()] = map[string]struct{}{}
			}
			b.level1Rules[item.GetNamespace()][conf.ID] = struct{}{}
			return
		}

		if item.GetService() != "*" && item.GetNamespace() != "*" {
			key := buildServiceKey(item.GetNamespace(), item.GetService())
			if _, ok := b.level1Rules[key]; !ok {
				b.level1Rules[key] = map[string]struct{}{}
			}

			b.level1Rules[key][conf.ID] = struct{}{}
		}
	}

	if conf.GetRoutingPolicy() == apiv2.RoutingPolicy_RulePolicy {
		sources := conf.RuleRouting.Sources
		for i := range sources {
			item := sources[i]
			handler(item)
		}

		destinations := conf.RuleRouting.Destinations
		for i := range destinations {
			item := destinations[i]
			handler(item)
		}
	}
}

func (b *routingBucketV2) saveV1(v1rule *model.RoutingConfig, v2rules []*v2.ExtendRoutingConfig) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.v1rules[v1rule.ID] = v2rules
}

func (b *routingBucketV2) deleteV2(id string) {
	b.lock.Lock()
	defer b.lock.Unlock()

	rule := b.rules[id]
	delete(b.rules, id)

	if rule == nil {
		return
	}

	if rule.GetRoutingPolicy() != apiv2.RoutingPolicy_RulePolicy {
		return
	}
	for i := range rule.RuleRouting.GetSources() {
		service := rule.RuleRouting.GetSources()[i].GetService()
		namespace := rule.RuleRouting.GetSources()[i].GetNamespace()

		if service == "*" && namespace == "*" {
			delete(b.level3Rules, id)
		}

		if service == "*" && namespace != "*" {
			delete(b.level2Rules[namespace], id)
		}

		if service != "*" && namespace != "*" {
			key := buildServiceKey(namespace, service)
			delete(b.level1Rules[key], id)
		}

	}
}

// deleteV1 删除 v1 的路由规则
func (b *routingBucketV2) deleteV1(id string) {
	b.lock.Lock()
	defer b.lock.Unlock()

	delete(b.v1rules, id)
}

// size v2 路由缓存的规则数量
func (b *routingBucketV2) size() int {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return len(b.rules)
}

// listByService 通过服务名称查询 v2 版本的路由规则
// TODO 需要按照 l1、l2、l3 返回不同级别的路由规则
func (b *routingBucketV2) listByService(service, namespace string) map[routingLevel][]*v2.ExtendRoutingConfig {

	ret := make(map[routingLevel][]*v2.ExtendRoutingConfig)

	b.lock.RLock()
	defer b.lock.RUnlock()

	// 查询 level1 级别的 v2 版本路由规则
	key := buildServiceKey(namespace, service)
	ids := b.level1Rules[key]
	level1 := make([]*v2.ExtendRoutingConfig, 0, 4)
	for i := range ids {
		if v, ok := b.rules[i]; ok {
			level1 = append(level1, v)
		}
	}
	ret[level1RoutingV2] = level1

	// 查询 level2 级别的 v2 版本路由规则
	level2 := make([]*v2.ExtendRoutingConfig, 0, 4)
	ids = b.level2Rules[namespace]
	for k := range ids {
		if v, ok := b.rules[k]; ok {
			level2 = append(level2, v)
		}
	}
	ret[level2RoutingV2] = level2

	// 查询 level3 级别的 v2 版本路由规则
	level3 := make([]*v2.ExtendRoutingConfig, 0, 4)
	for k := range b.level3Rules {
		if v, ok := b.rules[k]; ok {
			level3 = append(level3, v)
		}
	}
	ret[level3RoutingV2] = level3
	return ret
}

// foreach 遍历所有的路由规则
func (b *routingBucketV2) foreach(proc RoutingIterProc) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for k, v := range b.rules {
		proc(k, v)
	}

	for _, rules := range b.v1rules {
		for i := range rules {
			proc(rules[i].ID, rules[i])
		}
	}
}

func buildServiceKey(namespace, service string) string {
	return fmt.Sprintf("%s@@%s", namespace, service)
}
