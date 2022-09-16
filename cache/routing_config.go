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
	"fmt"
	"sort"
	"time"

	apiv1 "github.com/polarismesh/polaris-server/common/api/v1"
	apiv2 "github.com/polarismesh/polaris-server/common/api/v2"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	v2 "github.com/polarismesh/polaris-server/common/model/v2"
	routingcommon "github.com/polarismesh/polaris-server/common/routing"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
)

const (
	// RoutingConfigName router config name
	RoutingConfigName = "routingConfig"
)

type (
	RoutingIterProc func(key string, value *v2.ExtendRoutingConfig)

	// RoutingConfigCache routing配置的cache接口
	RoutingConfigCache interface {
		Cache
		// GetRoutingConfigV1 根据ServiceID获取路由配置
		GetRoutingConfigV1(id, service, namespace string) (*apiv1.Routing, error)
		// GetRoutingConfig 根据ServiceID获取路由配置
		GetRoutingConfigV2(id, service, namespace string) ([]*apiv2.Routing, error)
		// GetRoutingConfigCount 获取路由配置缓存的总个数
		GetRoutingConfigCount() int
		// GetRoutingConfigsV2 查询路由配置列表
		GetRoutingConfigsV2(args *RoutingArgs) (uint32, []*v2.ExtendRoutingConfig, error)
	}

	// routingConfigCache 路由规则缓存
	routingConfigCache struct {
		*baseCache

		serviceCache ServiceCache
		storage      store.Store

		firstUpdate bool

		bucketV1 *routingBucketV1
		bucketV2 *routingBucketV2

		lastMtimeV1 time.Time
		lastMtimeV2 time.Time
	}
)

// init 自注册到缓存列表
func init() {
	RegisterCache(RoutingConfigName, CacheRoutingConfig)
}

// newRoutingConfigCache 返回一个操作RoutingConfigCache的对象
func newRoutingConfigCache(s store.Store, serviceCache ServiceCache) *routingConfigCache {
	return &routingConfigCache{
		baseCache:    newBaseCache(),
		storage:      s,
		serviceCache: serviceCache,
	}
}

// initialize 实现Cache接口的函数
func (rc *routingConfigCache) initialize(opt map[string]interface{}) error {
	rc.firstUpdate = true

	rc.initBuckets()
	rc.lastMtimeV1 = time.Unix(0, 0)
	rc.lastMtimeV2 = time.Unix(0, 0)

	if opt == nil {
		return nil
	}
	return nil
}

func (rc *routingConfigCache) initBuckets() {
	rc.bucketV1 = newRoutingBucketV1()
	rc.bucketV2 = newRoutingBucketV2()
}

// update 实现Cache接口的函数
func (rc *routingConfigCache) update(storeRollbackSec time.Duration) error {
	outV1, err := rc.storage.GetRoutingConfigsForCache(rc.lastMtimeV1.Add(storeRollbackSec), rc.firstUpdate)
	if err != nil {
		log.CacheScope().Errorf("[Cache] routing config v1 cache update err: %s", err.Error())
		return err
	}

	outV2, err := rc.storage.GetRoutingConfigsV2ForCache(rc.lastMtimeV2.Add(storeRollbackSec), rc.firstUpdate)
	if err != nil {
		log.CacheScope().Errorf("[Cache] routing config v2 cache update err: %s", err.Error())
		return err
	}

	rc.firstUpdate = false
	if err := rc.setRoutingConfigV1(outV1); err != nil {
		return err
	}
	if err := rc.setRoutingConfigV2(outV2); err != nil {
		return err
	}

	return nil
}

// clear 实现Cache接口的函数
func (rc *routingConfigCache) clear() error {
	rc.firstUpdate = true

	rc.initBuckets()
	rc.lastMtimeV1 = time.Unix(0, 0)
	rc.lastMtimeV2 = time.Unix(0, 0)

	return nil
}

// name 实现Cache接口的函数
func (rc *routingConfigCache) name() string {
	return RoutingConfigName
}

// GetRoutingConfig 根据ServiceID获取路由配置
// case 1: 如果只存在 v2 的路由规则，使用 v2
// case 2: 如果只存在 v1 的路由规则，使用 v1
// case 3: 如果 v1 和 v2 同时存在，将 v1 规则和 v2 规则进行合并
func (rc *routingConfigCache) GetRoutingConfigV1(id, service, namespace string) (*apiv1.Routing, error) {
	if id == "" && service == "" && namespace == "" {
		return nil, nil
	}

	v2rules := rc.bucketV2.listByService(service, namespace)
	v1rules := rc.bucketV1.get(id)
	if v2rules != nil && v1rules == nil {
		return rc.convertRoutingV2toV1(v2rules, namespace, service), nil
	}

	if v2rules == nil && v1rules != nil {
		return routingcommon.RoutingV1Config2API(v1rules, service, namespace)
	}

	compositeRule, err := routingcommon.RoutingV1Config2API(v1rules, service, namespace)
	if err != nil {
		return nil, err
	}
	compositeRule, revisions := routingcommon.CompositeRoutingV1AndV2(compositeRule,
		v2rules[level1RoutingV2], v2rules[level2RoutingV2], v2rules[level3RoutingV2])
	if compositeRule == nil {
		return nil, nil
	}

	revision, err := CompositeComputeRevision(revisions)
	if err != nil {
		log.Error("[Cache][Routing] v2=>v1 compute revisions", zap.Error(err))
		return nil, err
	}

	compositeRule.Revision = utils.NewStringValue(revision)
	return compositeRule, nil
}

// GetRoutingConfigV2 根据服务信息获取该服务下的所有 v2 版本的规则路由
func (rc *routingConfigCache) GetRoutingConfigV2(id, service, namespace string) ([]*apiv2.Routing, error) {
	v2rules := rc.bucketV2.listByService(service, namespace)
	ret := make([]*apiv2.Routing, 0, len(v2rules))
	for level := range v2rules {
		items := v2rules[level]
		for i := range items {
			entry, err := items[i].ToApi()
			if err != nil {
				return nil, err
			}
			ret = append(ret, entry)
		}
	}

	return ret, nil
}

// IteratorRoutings
func (rc *routingConfigCache) IteratorRoutings(iterProc RoutingIterProc) {
	// 这里只需要遍历 v2 的 routing cache bucket 即可
	rc.bucketV2.foreach(iterProc)
}

// GetRoutingConfigCount 获取路由配置缓存的总个数
func (rc *routingConfigCache) GetRoutingConfigCount() int {
	return rc.bucketV1.size() + rc.bucketV2.size()
}

// setRoutingConfigV1 更新store的数据到cache中
func (rc *routingConfigCache) setRoutingConfigV1(cs []*model.RoutingConfig) error {
	if len(cs) == 0 {
		return nil
	}

	lastMtimeV1 := rc.lastMtimeV1.Unix()
	for _, entry := range cs {
		if entry.ID == "" {
			continue
		}
		if entry.ModifyTime.Unix() > lastMtimeV1 {
			lastMtimeV1 = entry.ModifyTime.Unix()
		}
		if !entry.Valid {
			// 删除老的 v1 缓存
			rc.bucketV1.delete(entry.ID)
			// 删除转换为 v2 的缓存
			rc.bucketV2.deleteV1(entry.ID)
			continue
		}

		// 保存到老的 v1 缓存
		rc.bucketV1.save(entry)
		// 保存到新的 v2 缓存

		if v2rule, err := rc.convertRoutingV1toV2(entry); err != nil {
			log.Error("[Cache] routing parse v1 => v2", zap.String("rule-id", entry.ID), zap.Error(err))
		} else {
			rc.bucketV2.saveV1(entry, v2rule)
		}
	}

	if rc.lastMtimeV1.Unix() < lastMtimeV1 {
		rc.lastMtimeV1 = time.Unix(lastMtimeV1, 0)
	}
	return nil
}

// setRoutingConfigV2 存储 v2 路由规则缓存
func (rc *routingConfigCache) setRoutingConfigV2(cs []*v2.RoutingConfig) error {
	if len(cs) == 0 {
		return nil
	}

	lastMtimeV2 := rc.lastMtimeV2.Unix()
	for _, entry := range cs {
		if entry.ID == "" {
			continue
		}
		if entry.ModifyTime.Unix() > lastMtimeV2 {
			lastMtimeV2 = entry.ModifyTime.Unix()
		}
		if !entry.Valid {
			rc.bucketV2.deleteV2(entry.ID)
			continue
		}
		extendEntry, err := entry.ToExpendRoutingConfig()
		if err != nil {
			log.Error("[Cache] routing config v2 convert to expend", zap.Error(err))
			continue
		}
		rc.bucketV2.saveV2(extendEntry)
	}
	if rc.lastMtimeV2.Unix() < lastMtimeV2 {
		rc.lastMtimeV2 = time.Unix(lastMtimeV2, 0)
	}

	return nil
}

// convertRoutingV1toV2 v1 版本的路由规则转为 v2 版本进行存储
func (rc *routingConfigCache) convertRoutingV1toV2(rule *model.RoutingConfig) ([]*v2.ExtendRoutingConfig, error) {
	saveDatas := make([]*v2.ExtendRoutingConfig, 0, 8)

	svc := rc.serviceCache.GetServiceByID(rule.ID)
	if svc == nil {
		return nil, nil
	}

	if rule.InBounds != "" {
		var inBounds []*apiv1.Route
		if err := json.Unmarshal([]byte(rule.InBounds), &inBounds); err != nil {
			return nil, err
		}
		for i := range inBounds {
			routing, err := routingcommon.BuildV2ExtendRouting(&apiv1.Routing{
				Namespace: utils.NewStringValue(svc.Namespace),
			}, inBounds[i])
			if err != nil {
				return nil, err
			}
			routing.Revision = rule.Revision
			routing.CreateTime = rule.CreateTime
			routing.ModifyTime = rule.ModifyTime
			routing.EnableTime = rule.CreateTime
			routing.ExtendInfo = map[string]string{
				v2.V1RuleIDKey:         rule.ID,
				v2.V1RuleRouteIndexKey: fmt.Sprintf("%d", i),
				v2.V1RuleRouteTypeKey:  v2.V1RuleInRoute,
			}

			saveDatas = append(saveDatas, routing)
		}
	}
	if rule.OutBounds != "" {
		var outBounds []*apiv1.Route
		if err := json.Unmarshal([]byte(rule.OutBounds), &outBounds); err != nil {
			return nil, err
		}

		for i := range outBounds {
			routing, err := routingcommon.BuildV2ExtendRouting(&apiv1.Routing{
				Namespace: utils.NewStringValue(svc.Namespace),
			}, outBounds[i])
			if err != nil {
				return nil, err
			}
			routing.Revision = rule.Revision
			routing.CreateTime = rule.CreateTime
			routing.ModifyTime = rule.ModifyTime
			routing.EnableTime = rule.CreateTime
			routing.ExtendInfo = map[string]string{
				v2.V1RuleIDKey:         rule.ID,
				v2.V1RuleRouteIndexKey: fmt.Sprintf("%d", i),
				v2.V1RuleRouteTypeKey:  v2.V1RuleOutRoute,
			}

			saveDatas = append(saveDatas, routing)
		}
	}
	return saveDatas, nil
}

// convertRoutingV2toV1 v2 版本的路由规则转为 v1 版本进行返回给客户端，用于兼容 SDK 下发配置的场景
func (rc *routingConfigCache) convertRoutingV2toV1(entries map[routingLevel][]*v2.ExtendRoutingConfig,
	service, namespace string) *apiv1.Routing {

	level1 := entries[level1RoutingV2]
	// 先确保规则的排序是从最高优先级开始排序
	sort.Slice(level1, func(i, j int) bool {
		return level1[i].Priority < level1[j].Priority
	})

	level2 := entries[level2RoutingV2]
	// 先确保规则的排序是从最高优先级开始排序
	sort.Slice(level2, func(i, j int) bool {
		return level2[i].Priority < level2[j].Priority
	})

	level3 := entries[level3RoutingV2]
	// 先确保规则的排序是从最高优先级开始排序
	sort.Slice(level3, func(i, j int) bool {
		return level3[i].Priority < level3[j].Priority
	})

	level1inRoutes, level1outRoutes, level1Revisions := routingcommon.BuildV1RoutesFromV2(level1)
	level2inRoutes, level2outRoutes, level2Revisions := routingcommon.BuildV1RoutesFromV2(level2)
	level3inRoutes, level3outRoutes, level3Revisions := routingcommon.BuildV1RoutesFromV2(level3)

	revisions := make([]string, 0, len(level1Revisions)+len(level2Revisions)+len(level3Revisions))
	revisions = append(revisions, level1Revisions...)
	revisions = append(revisions, level2Revisions...)
	revisions = append(revisions, level3Revisions...)
	revision, err := CompositeComputeRevision(revisions)
	if err != nil {
		log.Error("[Cache][Routing] v2=>v1 compute revisions", zap.Error(err))
		return nil
	}

	inRoutes := make([]*apiv1.Route, 0, len(level1inRoutes)+len(level2inRoutes)+len(level3inRoutes))
	inRoutes = append(inRoutes, level1inRoutes...)
	inRoutes = append(inRoutes, level2inRoutes...)
	inRoutes = append(inRoutes, level3inRoutes...)

	outRoutes := make([]*apiv1.Route, 0, len(level1outRoutes)+len(level2outRoutes)+len(level3outRoutes))
	outRoutes = append(outRoutes, level1outRoutes...)
	outRoutes = append(outRoutes, level2outRoutes...)
	outRoutes = append(outRoutes, level3outRoutes...)

	return &apiv1.Routing{
		Service:   utils.NewStringValue(service),
		Namespace: utils.NewStringValue(namespace),
		Inbounds:  inRoutes,
		Outbounds: outRoutes,
		Revision:  utils.NewStringValue(revision),
	}
}
