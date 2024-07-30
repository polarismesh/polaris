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
	sortKeys []string
	rules    map[string]*model.ExtendRouterConfig
	revision string

	customv1Rules *apitraffic.Routing
}

func NewServiceWithRouterRules(svcKey model.ServiceKey, direction model.TrafficDirection) *ServiceWithRouterRules {
	return &ServiceWithRouterRules{
		direction: direction,
		Service:   svcKey,
		rules:     make(map[string]*model.ExtendRouterConfig),
	}
}

// AddRouterRule 添加路由规则，注意，这里只会保留处于 Enable 状态的路由规则
func (s *ServiceWithRouterRules) AddRouterRule(rule *model.ExtendRouterConfig) {
	if !rule.Enable {
		return
	}
	if rule.GetRoutingPolicy() == apitraffic.RoutingPolicy_RulePolicy {
		s.customv1Rules = &apitraffic.Routing{
			Inbounds:  []*apitraffic.Route{},
			Outbounds: []*apitraffic.Route{},
		}
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.rules[rule.ID] = rule
}

func (s *ServiceWithRouterRules) DelRouterRule(id string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.rules, id)
}

// IterateRouterRules 这里是可以保证按照路由规则优先顺序进行遍历
func (s *ServiceWithRouterRules) IterateRouterRules(callback func(*model.ExtendRouterConfig)) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, key := range s.sortKeys {
		val, ok := s.rules[key]
		if ok {
			callback(val)
		}
	}

}

func (s *ServiceWithRouterRules) CountRouterRules() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.rules)
}

func (s *ServiceWithRouterRules) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.rules = make(map[string]*model.ExtendRouterConfig)
	s.revision = ""
}

func (s *ServiceWithRouterRules) reload() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.reloadRuleOrder()
	s.reloadRevision()
	s.reloadV1Rules()
}

func (s *ServiceWithRouterRules) reloadRuleOrder() {
	curRules := make([]*model.ExtendRouterConfig, 0, len(s.rules))
	for i := range s.rules {
		curRules = append(curRules, s.rules[i])
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
	revisioins := make([]string, 0, len(s.rules))
	for i := range s.sortKeys {
		revisioins = append(revisioins, s.rules[s.sortKeys[i]].Revision)
	}
	s.revision, _ = types.CompositeComputeRevision(revisioins)
}

func (s *ServiceWithRouterRules) reloadV1Rules() {
	if s.customv1Rules == nil {
		return
	}

	rules := make([]*model.ExtendRouterConfig, 0, 32)
	for i := range s.sortKeys {
		rule, ok := s.rules[s.sortKeys[i]]
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

func (c *ClientRouteRuleContainer) SearchRouteRuleV2(svc model.ServiceKey) []*model.ExtendRouterConfig {
	ret := make([]*model.ExtendRouterConfig, 0, 32)

	exactRule, existExactRule := c.exactRules.Load(svc.Domain())
	if existExactRule {
		exactRule.IterateRouterRules(func(erc *model.ExtendRouterConfig) {
			ret = append(ret, erc)
		})
	}

	nsWildcardRule, existNsWildcardRule := c.nsWildcardRules.Load(svc.Namespace)
	if existNsWildcardRule {
		nsWildcardRule.IterateRouterRules(func(erc *model.ExtendRouterConfig) {
			ret = append(ret, erc)
		})
	}

	c.allWildcardRules.IterateRouterRules(func(erc *model.ExtendRouterConfig) {
		ret = append(ret, erc)
	})
	return ret
}

// SearchCustomRuleV1 针对 v1 客户端拉取路由规则
func (c *ClientRouteRuleContainer) SearchCustomRuleV1(svc model.ServiceKey) (*apitraffic.Routing, []string) {
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
			revisions = append(revisions, exactRule.revision)
		}
		if existNsWildcardRule {
			ret.Outbounds = append(ret.Outbounds, nsWildcardRule.customv1Rules.Outbounds...)
		}
	}
	if existExactRule {
		revisions = append(revisions, exactRule.revision)
	}
	if existNsWildcardRule {
		revisions = append(revisions, nsWildcardRule.revision)
	}

	return ret, revisions
}

func (c *ClientRouteRuleContainer) SaveRule(svcKey model.ServiceKey, item *model.ExtendRouterConfig) {
	// level1 级别 cache 处理
	if svcKey.Name != model.MatchAll && svcKey.Namespace != model.MatchAll {
		c.exactRules.ComputeIfAbsent(svcKey.Domain(), func(k string) *ServiceWithRouterRules {
			return NewServiceWithRouterRules(svcKey, c.direction)
		})
		svcContainer, _ := c.exactRules.Load(svcKey.Domain())
		svcContainer.AddRouterRule(item)
	}
	// level2 级别 cache 处理
	if svcKey.Name == model.MatchAll && svcKey.Namespace != model.MatchAll {
		c.nsWildcardRules.ComputeIfAbsent(svcKey.Namespace, func(k string) *ServiceWithRouterRules {
			return NewServiceWithRouterRules(svcKey, c.direction)
		})

		nsRules, _ := c.nsWildcardRules.Load(svcKey.Namespace)
		nsRules.AddRouterRule(item)
	}
	// level3 级别 cache 处理
	if svcKey.Name == model.MatchAll && svcKey.Namespace == model.MatchAll {
		c.allWildcardRules.AddRouterRule(item)
	}
}

func (c *ClientRouteRuleContainer) RemoveRule(svcKey model.ServiceKey, ruleId string) {
	// level1 级别 cache 处理
	if svcKey.Name != model.MatchAll && svcKey.Namespace != model.MatchAll {
		svcContainer, ok := c.exactRules.Load(svcKey.Domain())
		if !ok {
			return
		}
		svcContainer.DelRouterRule(ruleId)
	}
	// level2 级别 cache 处理
	if svcKey.Name == model.MatchAll && svcKey.Namespace != model.MatchAll {
		nsRules, ok := c.nsWildcardRules.Load(svcKey.Namespace)
		if !ok {
			return
		}
		nsRules.DelRouterRule(ruleId)
	}
	// level3 级别 cache 处理
	if svcKey.Name == model.MatchAll && svcKey.Namespace == model.MatchAll {
		c.allWildcardRules.DelRouterRule(ruleId)
	}
}

func newRouteRuleContainer() *RouteRuleContainer {
	return &RouteRuleContainer{
		rules:            utils.NewSyncMap[string, *model.ExtendRouterConfig](),
		v1rules:          map[string][]*model.ExtendRouterConfig{},
		v1rulesToOld:     map[string]string{},
		nearbyContainers: newClientRouteRuleContainer(model.TrafficDirection_INBOUND),
		customContainers: map[model.TrafficDirection]*ClientRouteRuleContainer{
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

	// 就近路由规则缓存
	nearbyContainers *ClientRouteRuleContainer
	// 自定义路由规则缓存
	customContainers map[model.TrafficDirection]*ClientRouteRuleContainer

	// effect 记录一次缓存更新中，那些服务的路由出现了更新
	effect *utils.SyncSet[model.ServiceKey]

	// ------- 这里的逻辑都是为了兼容老的数据规则，这个将在1.18.2代码中移除，通过升级工具一次性处理 ------
	lock sync.RWMutex
	// v1rules service-id => []*model.ExtendRouterConfig v1 版本的规则自动转为 v2 版本的规则，用于 v2 接口的数据查看
	v1rules map[string][]*model.ExtendRouterConfig
	// v1rulesToOld 转为 v2 规则id 对应的原本的 v1 规则id 信息
	v1rulesToOld map[string]string
}

func (b *RouteRuleContainer) saveV2(conf *model.ExtendRouterConfig) {
	b.rules.Store(conf.ID, conf)
	handler := func(container *ClientRouteRuleContainer, svcKey model.ServiceKey) {
		b.effect.Add(svcKey)
		container.SaveRule(svcKey, conf)
	}

	switch conf.GetRoutingPolicy() {
	case apitraffic.RoutingPolicy_RulePolicy:
		handler(b.customContainers[model.TrafficDirection_OUTBOUND], conf.RuleRouting.Caller)
		handler(b.customContainers[model.TrafficDirection_INBOUND], conf.RuleRouting.Callee)
	case apitraffic.RoutingPolicy_NearbyPolicy:
		handler(b.nearbyContainers, model.ServiceKey{
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

	handler := func(container *ClientRouteRuleContainer, svcKey model.ServiceKey) {
		b.effect.Add(svcKey)
		container.RemoveRule(svcKey, id)
	}

	switch rule.GetRoutingPolicy() {
	case apitraffic.RoutingPolicy_RulePolicy:
		handler(b.customContainers[model.TrafficDirection_OUTBOUND], rule.RuleRouting.Caller)
		handler(b.customContainers[model.TrafficDirection_INBOUND], rule.RuleRouting.Callee)
	case apitraffic.RoutingPolicy_NearbyPolicy:
		handler(b.nearbyContainers, model.ServiceKey{
			Namespace: rule.NearbyRouting.Namespace,
			Name:      rule.NearbyRouting.Service,
		})
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

func (b *RouteRuleContainer) SearchCustomRules(svcName, namespace string) []*model.ExtendRouterConfig {
	ruleIds := map[string]struct{}{}

	svcKey := model.ServiceKey{Namespace: namespace, Name: svcName}

	ret := make([]*model.ExtendRouterConfig, 0, 32)

	rules := b.customContainers[model.TrafficDirection_INBOUND].SearchRouteRuleV2(svcKey)
	ret = append(ret, rules...)
	for i := range rules {
		ruleIds[rules[i].ID] = struct{}{}
	}

	rules = b.customContainers[model.TrafficDirection_OUTBOUND].SearchRouteRuleV2(svcKey)
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
		b.reloadCustom(val)
		b.reloadNearby(val)
	})
}

func (b *RouteRuleContainer) reloadCustom(val model.ServiceKey) {
	// 处理自定义路由
	// 处理 exact
	rules, ok := b.customContainers[model.TrafficDirection_INBOUND].exactRules.Load(val.Domain())
	if ok {
		rules.reload()
	}
	rules, ok = b.customContainers[model.TrafficDirection_OUTBOUND].exactRules.Load(val.Domain())
	if ok {
		rules.reload()
	}

	// 处理 ns wildcard
	rules, ok = b.customContainers[model.TrafficDirection_INBOUND].nsWildcardRules.Load(val.Namespace)
	if ok {
		rules.reload()
	}
	rules, ok = b.customContainers[model.TrafficDirection_OUTBOUND].nsWildcardRules.Load(val.Namespace)
	if ok {
		rules.reload()
	}

	// 处理 all wildcard
	b.customContainers[model.TrafficDirection_INBOUND].allWildcardRules.reload()
	b.customContainers[model.TrafficDirection_OUTBOUND].allWildcardRules.reload()
}

func (b *RouteRuleContainer) reloadNearby(val model.ServiceKey) {
	// 处理 exact
	rules, ok := b.nearbyContainers.exactRules.Load(val.Domain())
	if ok {
		rules.reload()
	}
	// 处理 ns wildcard
	rules, ok = b.nearbyContainers.nsWildcardRules.Load(val.Namespace)
	if ok {
		rules.reload()
	}
	// 处理 all wildcard
	b.nearbyContainers.allWildcardRules.reload()
}
