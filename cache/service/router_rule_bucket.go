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
	"fmt"
	"sync"

	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
)

type (
	routingLevel int16
	boundType    int16

	serviceInfo interface {
		GetNamespace() string
		GetService() string
	}
)

const (
	_ routingLevel = iota
	level1RoutingV2
	level2RoutingV2
	level3RoutingV2

	_ boundType = iota
	inBound
	outBound
)

func newRouteRuleBucket() *routeRuleBucket {
	return &routeRuleBucket{
		rules:       make(map[string]*model.ExtendRouterConfig),
		level1Rules: map[string]map[string]struct{}{},
		level2Rules: map[boundType]map[string]map[string]struct{}{
			inBound:  {},
			outBound: {},
		},
		level3Rules: map[boundType]map[string]struct{}{
			inBound:  {},
			outBound: {},
		},
		v1rules:      map[string][]*model.ExtendRouterConfig{},
		v1rulesToOld: map[string]string{},
	}
}

// ServiceWithCircuitBreakerRules 与服务关系绑定的熔断规则
type ServiceWithRouterRules struct {
	mutex    sync.RWMutex
	Service  model.ServiceKey
	v2Rules  map[string]*model.ExtendRouterConfig
	v1Rules  *apitraffic.Routing
	Revision string
}

func NewServiceWithRouterRules(svcKey model.ServiceKey) *ServiceWithRouterRules {
	return &ServiceWithRouterRules{
		Service: svcKey,
		v2Rules: make(map[string]*model.ExtendRouterConfig),
	}
}

func (s *ServiceWithRouterRules) AddRouterRule(rule *model.ExtendRouterConfig) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.v2Rules[rule.ID] = rule
}

func (s *ServiceWithRouterRules) DelRouterRule(id string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.v2Rules, id)
}

func (s *ServiceWithRouterRules) IterateRouterRules(callback func(*model.ExtendRouterConfig)) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for _, rule := range s.v2Rules {
		callback(rule)
	}
}

func (s *ServiceWithRouterRules) CountRouterRules() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.v2Rules)
}

func (s *ServiceWithRouterRules) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.v2Rules = make(map[string]*model.ExtendRouterConfig)
	s.Revision = ""
}

// routeRuleBucket v2 路由规则缓存 bucket
type routeRuleBucket struct {
	lock sync.RWMutex
	// rules id => routing rule
	rules map[string]*model.ExtendRouterConfig
	// level1Rules service(name)+namespace => 路由规则ID列表，只针对某个具体的服务有效
	level1Rules map[string]map[string]struct{}
	// level2Rules service(*) + namespace =>  路由规则ID列表, 针对某个命名空间下所有服务都生效的路由规则
	level2Rules map[boundType]map[string]map[string]struct{}
	// level3Rules service(*) + namespace(*) =>  路由规则ID列表, 针对所有命名空间下的所有服务都生效的规则
	level3Rules map[boundType]map[string]struct{}
	// v1rules service-id => []*model.ExtendRouterConfig v1 版本的规则自动转为 v2 版本的规则，用于 v2 接口的数据查看
	v1rules map[string][]*model.ExtendRouterConfig

	// fetched service cache
	// key1: namespace, key2: service
	routerRules map[string]map[string]*ServiceWithRouterRules
	// key1: namespace
	nsWildcardRules map[string]*ServiceWithRouterRules
	// all rules are wildcard specific
	allWildcardRules *ServiceWithRouterRules

	// v1rulesToOld 转为 v2 规则id 对应的原本的 v1 规则id 信息
	v1rulesToOld map[string]string
}

func (b *routeRuleBucket) getV2(id string) *model.ExtendRouterConfig {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.rules[id]
}

func (b *routeRuleBucket) saveV2(conf *model.ExtendRouterConfig) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.rules[conf.ID] = conf
	handler := func(bt boundType, item serviceInfo) {
		// level1 级别 cache 处理
		if item.GetService() != model.MatchAll && item.GetNamespace() != model.MatchAll {
			key := buildServiceKey(item.GetNamespace(), item.GetService())
			if _, ok := b.level1Rules[key]; !ok {
				b.level1Rules[key] = map[string]struct{}{}
			}

			b.level1Rules[key][conf.ID] = struct{}{}
			return
		}
		// level2 级别 cache 处理
		if item.GetService() == model.MatchAll && item.GetNamespace() != model.MatchAll {
			if _, ok := b.level2Rules[bt][item.GetNamespace()]; !ok {
				b.level2Rules[bt][item.GetNamespace()] = map[string]struct{}{}
			}
			b.level2Rules[bt][item.GetNamespace()][conf.ID] = struct{}{}
			return
		}
		// level3 级别 cache 处理
		if item.GetService() == model.MatchAll && item.GetNamespace() == model.MatchAll {
			b.level3Rules[bt][conf.ID] = struct{}{}
			return
		}
	}

	if conf.GetRoutingPolicy() == apitraffic.RoutingPolicy_RulePolicy {
		subRules := conf.RuleRouting.Rules
		for i := range subRules {
			sources := subRules[i].Sources
			for i := range sources {
				item := sources[i]
				handler(outBound, item)
			}

			destinations := subRules[i].Destinations
			for i := range destinations {
				item := destinations[i]
				handler(inBound, item)
			}
		}
	}
}

// saveV1 保存 v1 级别的路由规则
func (b *routeRuleBucket) saveV1(v1rule *model.RoutingConfig, v2rules []*model.ExtendRouterConfig) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.v1rules[v1rule.ID] = v2rules

	for i := range v2rules {
		item := v2rules[i]
		b.v1rulesToOld[item.ID] = v1rule.ID
	}
}

func (b *routeRuleBucket) convertV2Size() uint32 {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return uint32(len(b.v1rulesToOld))
}

func (b *routeRuleBucket) deleteV2(id string) {
	b.lock.Lock()
	defer b.lock.Unlock()

	rule := b.rules[id]
	delete(b.rules, id)

	if rule == nil {
		return
	}

	if rule.GetRoutingPolicy() != apitraffic.RoutingPolicy_RulePolicy {
		return
	}

	subRules := rule.RuleRouting.Rules
	for i := range subRules {
		for j := range subRules[i].GetSources() {
			source := subRules[i].GetSources()[j]
			service := source.GetService()
			namespace := source.GetNamespace()

			if service == model.MatchAll && namespace == model.MatchAll {
				delete(b.level3Rules[outBound], id)
				delete(b.level3Rules[inBound], id)
			}

			if service == model.MatchAll && namespace != model.MatchAll {
				delete(b.level2Rules[outBound][namespace], id)
				delete(b.level2Rules[inBound][namespace], id)
			}

			if service != model.MatchAll && namespace != model.MatchAll {
				key := buildServiceKey(namespace, service)
				delete(b.level1Rules[key], id)
			}
		}
	}
}

// deleteV1 删除 v1 的路由规则
func (b *routeRuleBucket) deleteV1(serviceId string) {
	b.lock.Lock()
	defer b.lock.Unlock()

	items, ok := b.v1rules[serviceId]
	if !ok {
		delete(b.v1rules, serviceId)
		return
	}

	for i := range items {
		delete(b.v1rulesToOld, items[i].ID)
	}
	delete(b.v1rules, serviceId)
}

// size Number of routing-v2 cache rules
func (b *routeRuleBucket) size() int {
	b.lock.RLock()
	defer b.lock.RUnlock()

	cnt := len(b.rules)
	for k := range b.v1rules {
		cnt += len(b.v1rules[k])
	}

	return cnt
}

// listEnableRules Inquire the routing rules of the V2 version through the service name,
// and perform some filtering according to the Predicate
func (b *routeRuleBucket) listEnableRules(service, namespace string) map[routingLevel][]*model.ExtendRouterConfig {
	ret := make(map[routingLevel][]*model.ExtendRouterConfig)
	tmpRecord := map[string]struct{}{}

	b.lock.RLock()
	defer b.lock.RUnlock()

	predicate := func(item *model.ExtendRouterConfig) bool {
		return item.Enable
	}

	// Query Level1 V2 version routing rules
	key := buildServiceKey(namespace, service)
	ids := b.level1Rules[key]
	level1 := make([]*model.ExtendRouterConfig, 0, 4)
	for i := range ids {
		if v, ok := b.rules[i]; ok && predicate(v) {
			level1 = append(level1, v)
			tmpRecord[v.ID] = struct{}{}
		}
	}
	ret[level1RoutingV2] = level1

	handler := func(ids map[string]struct{}, bt boundType) []*model.ExtendRouterConfig {
		ret := make([]*model.ExtendRouterConfig, 0, 4)

		for k := range ids {
			v := b.rules[k]
			if v == nil {
				continue
			}
			if _, ok := tmpRecord[v.ID]; ok {
				continue
			}
			if !predicate(v) {
				continue
			}
			ret = append(ret, v)
			tmpRecord[v.ID] = struct{}{}
		}

		return ret
	}

	// Query Level 2 level routing-v2 rules
	level2 := make([]*model.ExtendRouterConfig, 0, 4)
	level2 = append(level2, handler(b.level2Rules[outBound][namespace], outBound)...)
	level2 = append(level2, handler(b.level2Rules[inBound][namespace], inBound)...)
	ret[level2RoutingV2] = level2

	// Query Level3 level routing-v2 rules
	level3 := make([]*model.ExtendRouterConfig, 0, 4)
	level3 = append(level3, handler(b.level3Rules[outBound], outBound)...)
	level3 = append(level3, handler(b.level3Rules[inBound], inBound)...)
	ret[level3RoutingV2] = level3
	return ret
}

// foreach Traversing all routing rules
func (b *routeRuleBucket) foreach(proc types.RouterRuleIterProc) {
	b.lock.RLock()
	defer b.lock.RUnlock()

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
