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
	"sort"
	"sync"

	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// ServiceWithRouterRules 与服务绑定的路由规则数据
type ServiceWithRouterRules struct {
	direction model.TrafficDirection
	mutex     sync.RWMutex
	Service   model.ServiceKey
	// sortKeys: 针对 customv2Rules 做了排序
	sortKeys       []string
	customv2Rules  map[string]*model.ExtendRouterConfig
	customv1Rules  *apitraffic.Routing
	customRevision string
	// 就近路由规则缓存
	nearbyRules    map[string]*model.ExtendRouterConfig
	nearbyRevision string
}

func NewServiceWithRouterRules(svcKey model.ServiceKey, direction model.TrafficDirection) *ServiceWithRouterRules {
	return &ServiceWithRouterRules{
		direction:     direction,
		Service:       svcKey,
		customv2Rules: make(map[string]*model.ExtendRouterConfig),
		customv1Rules: &apitraffic.Routing{
			Inbounds:  []*apitraffic.Route{},
			Outbounds: []*apitraffic.Route{},
		},
	}
}

// AddRouterRule 添加路由规则，注意，这里只会保留处于 Enable 状态的路由规则
func (s *ServiceWithRouterRules) AddRouterRule(rule *model.ExtendRouterConfig) {
	if !rule.Enable {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch rule.GetRoutingPolicy() {
	case apitraffic.RoutingPolicy_NearbyPolicy:
		s.nearbyRules[rule.ID] = rule
	case apitraffic.RoutingPolicy_RulePolicy:
		s.customv2Rules[rule.ID] = rule
	}
}

func (s *ServiceWithRouterRules) DelRouterRule(id string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.customv2Rules, id)
}

// IterateRouterRules 这里是可以保证按照路由规则优先顺序进行遍历
func (s *ServiceWithRouterRules) IterateRouterRules(policy apitraffic.RoutingPolicy, callback func(*model.ExtendRouterConfig)) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	switch policy {
	case apitraffic.RoutingPolicy_RulePolicy:
		for _, key := range s.sortKeys {
			val, ok := s.customv2Rules[key]
			if ok {
				callback(val)
			}
		}
	case apitraffic.RoutingPolicy_NearbyPolicy:
		for i := range s.nearbyRules {
			callback(s.nearbyRules[i])
		}
	}
}

// IterateNearRules 这里是可以保证按照路由规则优先顺序进行遍历
func (s *ServiceWithRouterRules) IterateNearRules(callback func(*model.ExtendRouterConfig)) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for i := range s.nearbyRules {
		callback(s.nearbyRules[i])
	}
}

func (s *ServiceWithRouterRules) CountRouterRules() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.customv2Rules)
}

func (s *ServiceWithRouterRules) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.customv2Rules = make(map[string]*model.ExtendRouterConfig)
	s.customRevision = ""
}

func (s *ServiceWithRouterRules) reload() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.reloadRuleOrder()
	s.reloadRevision()
	s.reloadV1Rules()
}

func (s *ServiceWithRouterRules) reloadRuleOrder() {
	curRules := make([]*model.ExtendRouterConfig, 0, len(s.customv2Rules))
	for i := range s.customv2Rules {
		curRules = append(curRules, s.customv2Rules[i])
	}

	sort.Slice(curRules, func(i, j int) bool {
		return model.CompareRoutingV2(curRules[i], curRules[j])
	})

	curKeys := make([]string, 0, len(curRules))
	for i := range curRules {
		curKeys = append(curKeys, curRules[i].ID)
	}

	s.sortKeys = curKeys
}

func (s *ServiceWithRouterRules) reloadRevision() {
	revisioins := make([]string, 0, len(s.customv2Rules))
	for i := range s.sortKeys {
		revisioins = append(revisioins, s.customv2Rules[s.sortKeys[i]].Revision)
	}
	s.customRevision, _ = types.CompositeComputeRevision(revisioins)

	revisioins = make([]string, 0, len(s.nearbyRules))
	for i := range s.nearbyRules {
		revisioins = append(revisioins, s.nearbyRules[i].Revision)
	}
	s.nearbyRevision, _ = types.CompositeComputeRevision(revisioins)
}

func (s *ServiceWithRouterRules) reloadV1Rules() {
	rules := make([]*model.ExtendRouterConfig, 0, 32)
	for i := range s.sortKeys {
		rule, ok := s.customv2Rules[s.sortKeys[i]]
		if !ok {
			continue
		}
		rules = append(rules, rule)
	}

	routes := make([]*apitraffic.Route, 0, 32)

	for i := range rules {
		if rules[i].Priority != uint32(apitraffic.RoutingPolicy_RulePolicy) {
			continue
		}
		routes = append(routes, model.BuildRoutes(rules[i], s.direction)...)
	}

	s.customv1Rules = &apitraffic.Routing{}
	switch s.direction {
	case model.TrafficDirection_INBOUND:
		s.customv1Rules.Inbounds = routes
	case model.TrafficDirection_OUTBOUND:
		s.customv1Rules.Outbounds = routes
	}
}

func newClientRouteRuleContainer(direction model.TrafficDirection) *ClientRouteRuleContainer {
	return &ClientRouteRuleContainer{
		direction:        direction,
		exactRules:       utils.NewSyncMap[string, *ServiceWithRouterRules](),
		nsWildcardRules:  utils.NewSyncMap[string, *ServiceWithRouterRules](),
		allWildcardRules: NewServiceWithRouterRules(model.ServiceKey{Namespace: types.AllMatched, Name: types.AllMatched}, direction),
	}
}

type ClientRouteRuleContainer struct {
	direction model.TrafficDirection
	// key1: namespace, key2: service
	exactRules *utils.SyncMap[string, *ServiceWithRouterRules]
	// key1: namespace is exact, service is full match
	nsWildcardRules *utils.SyncMap[string, *ServiceWithRouterRules]
	// all rules are wildcard specific
	allWildcardRules *ServiceWithRouterRules
}

func (c *ClientRouteRuleContainer) SearchRouteRuleV2(policy apitraffic.RoutingPolicy, svc model.ServiceKey) []*model.ExtendRouterConfig {
	ret := make([]*model.ExtendRouterConfig, 0, 32)

	exactRule, existExactRule := c.exactRules.Load(svc.Domain())
	if existExactRule {
		exactRule.IterateRouterRules(policy, func(erc *model.ExtendRouterConfig) {
			ret = append(ret, erc)
		})
	}

	nsWildcardRule, existNsWildcardRule := c.nsWildcardRules.Load(svc.Namespace)
	if existNsWildcardRule {
		nsWildcardRule.IterateRouterRules(policy, func(erc *model.ExtendRouterConfig) {
			ret = append(ret, erc)
		})
	}

	c.allWildcardRules.IterateRouterRules(policy, func(erc *model.ExtendRouterConfig) {
		ret = append(ret, erc)
	})
	return ret
}

// SearchRouteRuleV1 针对 v1 客户端拉取路由规则
func (c *ClientRouteRuleContainer) SearchRouteRuleV1(svc model.ServiceKey) (*apitraffic.Routing, []string) {
	ret := &apitraffic.Routing{
		Inbounds:  make([]*apitraffic.Route, 0, 8),
		Outbounds: make([]*apitraffic.Route, 0, 8),
	}
	exactRule, existExactRule := c.exactRules.Load(svc.Domain())
	nsWildcardRule, existNsWildcardRule := c.nsWildcardRules.Load(svc.Namespace)

	revisions := make([]string, 0, 2)

	switch c.direction {
	case model.TrafficDirection_INBOUND:
		if existExactRule {
			ret.Inbounds = append(ret.Inbounds, exactRule.customv1Rules.Inbounds...)
		}
		if existNsWildcardRule {
			ret.Inbounds = append(ret.Inbounds, nsWildcardRule.customv1Rules.Inbounds...)
		}
	default:
		if existExactRule {
			ret.Outbounds = append(ret.Outbounds, exactRule.customv1Rules.Outbounds...)
			revisions = append(revisions, exactRule.customRevision)
		}
		if existNsWildcardRule {
			ret.Outbounds = append(ret.Outbounds, nsWildcardRule.customv1Rules.Outbounds...)
		}
	}
	if existExactRule {
		revisions = append(revisions, exactRule.customRevision)
	}
	if existNsWildcardRule {
		revisions = append(revisions, nsWildcardRule.customRevision)
	}

	return ret, revisions
}

func (c *ClientRouteRuleContainer) SaveToExact(svc model.ServiceKey, item *model.ExtendRouterConfig) {
	c.exactRules.ComputeIfAbsent(svc.Domain(), func(k string) *ServiceWithRouterRules {
		return NewServiceWithRouterRules(svc, c.direction)
	})

	svcContainer, _ := c.exactRules.Load(svc.Domain())
	svcContainer.AddRouterRule(item)
}

func (c *ClientRouteRuleContainer) RemoveFromExact(svc model.ServiceKey, ruleId string) {
	svcContainer, ok := c.exactRules.Load(svc.Domain())
	if !ok {
		return
	}
	svcContainer.DelRouterRule(ruleId)
}

func (c *ClientRouteRuleContainer) SaveToNamespaceWildcard(svc model.ServiceKey, item *model.ExtendRouterConfig) {
	c.nsWildcardRules.ComputeIfAbsent(svc.Namespace, func(k string) *ServiceWithRouterRules {
		return NewServiceWithRouterRules(svc, c.direction)
	})

	nsRules, _ := c.nsWildcardRules.Load(svc.Namespace)
	nsRules.AddRouterRule(item)
}

func (c *ClientRouteRuleContainer) RemoveFromNamespaceWildcard(svc model.ServiceKey, ruleId string) {
	nsRules, ok := c.nsWildcardRules.Load(svc.Namespace)
	if !ok {
		return
	}

	nsRules.DelRouterRule(ruleId)
}

func (c *ClientRouteRuleContainer) SaveToAllWildcard(item *model.ExtendRouterConfig) {
	c.allWildcardRules.AddRouterRule(item)
}

func (c *ClientRouteRuleContainer) RemoveFromAllWildcard(ruleId string) {
	c.allWildcardRules.DelRouterRule(ruleId)
}

func newRouteRuleContainer() *RouteRuleContainer {
	return &RouteRuleContainer{
		rules:        utils.NewSyncMap[string, *model.ExtendRouterConfig](),
		v1rules:      map[string][]*model.ExtendRouterConfig{},
		v1rulesToOld: map[string]string{},
		directionContainers: map[model.TrafficDirection]*ClientRouteRuleContainer{
			model.TrafficDirection_INBOUND:  newClientRouteRuleContainer(model.TrafficDirection_INBOUND),
			model.TrafficDirection_OUTBOUND: newClientRouteRuleContainer(model.TrafficDirection_OUTBOUND),
		},
		effect: utils.NewSyncSet[model.ServiceKey](),
	}
}

// RouteRuleContainer v2 路由规则缓存 bucket
type RouteRuleContainer struct {
	// rules id => routing rule
	rules *utils.SyncMap[string, *model.ExtendRouterConfig]

	directionContainers map[model.TrafficDirection]*ClientRouteRuleContainer

	lock sync.RWMutex
	// v1rules service-id => []*model.ExtendRouterConfig v1 版本的规则自动转为 v2 版本的规则，用于 v2 接口的数据查看
	v1rules map[string][]*model.ExtendRouterConfig
	// v1rulesToOld 转为 v2 规则id 对应的原本的 v1 规则id 信息
	v1rulesToOld map[string]string
	// effect
	effect *utils.SyncSet[model.ServiceKey]
}

func (b *RouteRuleContainer) saveV2(conf *model.ExtendRouterConfig) {
	b.rules.Store(conf.ID, conf)
	handler := func(direction model.TrafficDirection, svcKey model.ServiceKey) {
		b.effect.Add(svcKey)
		container := b.directionContainers[direction]
		// level1 级别 cache 处理
		if svcKey.Name != model.MatchAll && svcKey.Namespace != model.MatchAll {
			container.SaveToExact(svcKey, conf)
			return
		}
		// level2 级别 cache 处理
		if svcKey.Name == model.MatchAll && svcKey.Namespace != model.MatchAll {
			container.SaveToNamespaceWildcard(svcKey, conf)
			return
		}
		// level3 级别 cache 处理
		if svcKey.Name == model.MatchAll && svcKey.Namespace == model.MatchAll {
			container.SaveToAllWildcard(conf)
			return
		}
	}

	switch conf.GetRoutingPolicy() {
	case apitraffic.RoutingPolicy_RulePolicy:
		handler(model.TrafficDirection_OUTBOUND, conf.RuleRouting.Caller)
		handler(model.TrafficDirection_INBOUND, conf.RuleRouting.Callee)
	case apitraffic.RoutingPolicy_NearbyPolicy:
		handler(model.TrafficDirection_INBOUND, model.ServiceKey{
			Namespace: conf.NearbyRouting.Namespace,
			Name:      conf.NearbyRouting.Service,
		})
	}

}

// saveV1 保存 v1 级别的路由规则
func (b *RouteRuleContainer) saveV1(v1rule *model.RoutingConfig, v2rules []*model.ExtendRouterConfig) {
	for i := range v2rules {
		b.saveV2(v2rules[i])
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.v1rules[v1rule.ID] = v2rules

	for i := range v2rules {
		item := v2rules[i]
		b.v1rulesToOld[item.ID] = v1rule.ID
	}
}

func (b *RouteRuleContainer) convertV2Size() uint32 {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return uint32(len(b.v1rulesToOld))
}

func (b *RouteRuleContainer) deleteV2(id string) {
	rule, exist := b.rules.Load(id)
	b.rules.Delete(id)
	if !exist {
		return
	}

	handler := func(direction model.TrafficDirection, svcKey model.ServiceKey) {
		b.effect.Add(svcKey)
		container := b.directionContainers[direction]
		// level1 级别 cache 处理
		if svcKey.Name != model.MatchAll && svcKey.Namespace != model.MatchAll {
			container.RemoveFromExact(svcKey, id)
			return
		}
		// level2 级别 cache 处理
		if svcKey.Name == model.MatchAll && svcKey.Namespace != model.MatchAll {
			container.RemoveFromNamespaceWildcard(svcKey, id)
			return
		}
		// level3 级别 cache 处理
		if svcKey.Name == model.MatchAll && svcKey.Namespace == model.MatchAll {
			container.RemoveFromAllWildcard(id)
			return
		}
	}

	switch rule.GetRoutingPolicy() {
	case apitraffic.RoutingPolicy_RulePolicy:
		handler(model.TrafficDirection_OUTBOUND, rule.RuleRouting.Caller)
		handler(model.TrafficDirection_INBOUND, rule.RuleRouting.Callee)
	}
}

// deleteV1 删除 v1 的路由规则
func (b *RouteRuleContainer) deleteV1(serviceId string) {
	b.lock.Lock()
	defer b.lock.Unlock()

	items, ok := b.v1rules[serviceId]
	if !ok {
		delete(b.v1rules, serviceId)
		return
	}

	for i := range items {
		delete(b.v1rulesToOld, items[i].ID)
		b.deleteV2(items[i].ID)
	}
	delete(b.v1rules, serviceId)
}

// size Number of routing-v2 cache rules
func (b *RouteRuleContainer) size() int {
	b.lock.RLock()
	defer b.lock.RUnlock()

	cnt := b.rules.Len()
	for k := range b.v1rules {
		cnt += len(b.v1rules[k])
	}

	return cnt
}

func (b *RouteRuleContainer) SearchRouteRules(svcName, namespace string) []*model.ExtendRouterConfig {
	ruleIds := map[string]struct{}{}

	svcKey := model.ServiceKey{Namespace: namespace, Name: svcName}

	ret := make([]*model.ExtendRouterConfig, 0, 32)

	rules := b.directionContainers[model.TrafficDirection_INBOUND].SearchRouteRuleV2(apitraffic.RoutingPolicy_RulePolicy, svcKey)
	ret = append(ret, rules...)
	for i := range rules {
		ruleIds[rules[i].ID] = struct{}{}
	}

	rules = b.directionContainers[model.TrafficDirection_OUTBOUND].SearchRouteRuleV2(apitraffic.RoutingPolicy_RulePolicy, svcKey)
	for i := range rules {
		if _, ok := ruleIds[rules[i].ID]; !ok {
			ret = append(ret, rules[i])
		}
	}

	return ret
}

// foreach Traversing all routing rules
func (b *RouteRuleContainer) foreach(proc types.RouterRuleIterProc) {
	b.rules.Range(func(key string, val *model.ExtendRouterConfig) {
		proc(key, val)
	})

	for _, rules := range b.v1rules {
		for i := range rules {
			proc(rules[i].ID, rules[i])
		}
	}
}

func (b *RouteRuleContainer) reload() {
	b.effect.Range(func(val model.ServiceKey) {
		// 处理 exact
		rules, ok := b.directionContainers[model.TrafficDirection_INBOUND].exactRules.Load(val.Domain())
		if ok {
			rules.reload()
		}
		rules, ok = b.directionContainers[model.TrafficDirection_OUTBOUND].exactRules.Load(val.Domain())
		if ok {
			rules.reload()
		}

		// 处理 ns wildcard
		rules, ok = b.directionContainers[model.TrafficDirection_INBOUND].nsWildcardRules.Load(val.Namespace)
		if ok {
			rules.reload()
		}
		rules, ok = b.directionContainers[model.TrafficDirection_OUTBOUND].nsWildcardRules.Load(val.Namespace)
		if ok {
			rules.reload()
		}

		// 处理 all wildcard
		b.directionContainers[model.TrafficDirection_INBOUND].allWildcardRules.reload()
		b.directionContainers[model.TrafficDirection_OUTBOUND].allWildcardRules.reload()
	})
}
