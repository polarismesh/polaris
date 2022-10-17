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
	"sort"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	apiv1 "github.com/polarismesh/polaris/common/api/v1"
	apiv2 "github.com/polarismesh/polaris/common/api/v2"
	"github.com/polarismesh/polaris/common/model"
	v2 "github.com/polarismesh/polaris/common/model/v2"
	routingcommon "github.com/polarismesh/polaris/common/routing"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const (
	// RoutingConfigName router config name
	RoutingConfigName = "routingConfig"
)

type (
	// RoutingIterProc 遍历路由规则的方法定义
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
		// IsConvertFromV1 当前路由规则是否是从 v1 规则转换过来的
		IsConvertFromV1(id string) (string, bool)
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

		singleFlight singleflight.Group

		// pendingV1RuleIds 记录需要从 v1 转换到 v2 的路由规则id
		pendingV1RuleIds map[string]struct{}
	}
)

// init 自注册到缓存列表
func init() {
	RegisterCache(RoutingConfigName, CacheRoutingConfig)
}

// newRoutingConfigCache 返回一个操作RoutingConfigCache的对象
func newRoutingConfigCache(s store.Store, serviceCache ServiceCache) *routingConfigCache {
	return &routingConfigCache{
		baseCache:        newBaseCache(),
		storage:          s,
		serviceCache:     serviceCache,
		pendingV1RuleIds: map[string]struct{}{},
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
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := rc.singleFlight.Do("RoutingCache", func() (interface{}, error) {
		return nil, rc.realUpdate(storeRollbackSec)
	})
	return err
}

// update 实现Cache接口的函数
func (rc *routingConfigCache) realUpdate(storeRollbackSec time.Duration) error {
	outV1, err := rc.storage.GetRoutingConfigsForCache(rc.lastMtimeV1.Add(storeRollbackSec), rc.firstUpdate)
	if err != nil {
		log.Errorf("[Cache] routing config v1 cache get from store err: %s", err.Error())
		return err
	}

	outV2, err := rc.storage.GetRoutingConfigsV2ForCache(rc.lastMtimeV2.Add(storeRollbackSec), rc.firstUpdate)
	if err != nil {
		log.Errorf("[Cache] routing config v2 cache get from store err: %s", err.Error())
		return err
	}

	rc.firstUpdate = false
	if err := rc.setRoutingConfigV1(outV1); err != nil {
		log.Errorf("[Cache] routing config v1 cache update err: %s", err.Error())
		return err
	}
	if err := rc.setRoutingConfigV2(outV2); err != nil {
		log.Errorf("[Cache] routing config v2 cache update err: %s", err.Error())
		return err
	}
	rc.setRoutingConfigV1ToV2()
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
// case 3: 如果同时存在 v1 和 v2 的路由规则，进行合并
func (rc *routingConfigCache) GetRoutingConfigV1(id, service, namespace string) (*apiv1.Routing, error) {
	if id == "" && service == "" && namespace == "" {
		return nil, nil
	}

	v2rules := rc.bucketV2.listByServiceWithPredicate(service, namespace,
		// 只返回 enable 状态的路由规则进行下发
		func(item *v2.ExtendRoutingConfig) bool {
			return item.Enable
		})
	v1rules := rc.bucketV1.get(id)
	if v2rules != nil && v1rules == nil {
		return formatRoutingResponseV1(rc.convertRoutingV2toV1(v2rules, service, namespace)), nil
	}

	if v2rules == nil && v1rules != nil {
		ret, err := routingcommon.RoutingConfigV1ToAPI(v1rules, service, namespace)
		if err != nil {
			return nil, err
		}
		return formatRoutingResponseV1(ret), nil
	}

	apiv1rule, err := routingcommon.RoutingConfigV1ToAPI(v1rules, service, namespace)
	if err != nil {
		return nil, err
	}
	compositeRule, revisions := routingcommon.CompositeRoutingV1AndV2(apiv1rule,
		v2rules[level1RoutingV2], v2rules[level2RoutingV2], v2rules[level3RoutingV2])

	revision, err := CompositeComputeRevision(revisions)
	if err != nil {
		log.Error("[Cache][Routing] v2=>v1 compute revisions", zap.Error(err))
		return nil, err
	}

	compositeRule.Revision = utils.NewStringValue(revision)
	return formatRoutingResponseV1(compositeRule), nil
}

// formatRoutingResponseV1 给客户端的缓存，不需要暴露 ExtendInfo 信息数据
func formatRoutingResponseV1(ret *apiv1.Routing) *apiv1.Routing {
	inBounds := ret.Inbounds
	outBounds := ret.Outbounds

	for i := range inBounds {
		inBounds[i].ExtendInfo = nil
	}

	for i := range outBounds {
		outBounds[i].ExtendInfo = nil
	}
	return ret
}

// GetRoutingConfigV2 根据服务信息获取该服务下的所有 v2 版本的规则路由
func (rc *routingConfigCache) GetRoutingConfigV2(id, service, namespace string) ([]*apiv2.Routing, error) {
	v2rules := rc.bucketV2.listByServiceWithPredicate(service, namespace,
		func(item *v2.ExtendRoutingConfig) bool {
			return item.Enable
		})

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
	return rc.bucketV2.size()
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
			// 删除 v1 转换到 v2 的任务id
			delete(rc.pendingV1RuleIds, entry.ID)
			continue
		}

		// 保存到老的 v1 缓存
		rc.bucketV1.save(entry)
		rc.pendingV1RuleIds[entry.ID] = struct{}{}
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

func (rc *routingConfigCache) setRoutingConfigV1ToV2() {
	for id := range rc.pendingV1RuleIds {

		entry := rc.bucketV1.get(id)

		// 保存到新的 v2 缓存
		if v2rule, err := rc.convertRoutingV1toV2(entry); err != nil {
			log.Error("[Cache] routing parse v1 => v2, will try again next",
				zap.String("rule-id", entry.ID), zap.Error(err))
		} else {
			rc.bucketV2.saveV1(entry, v2rule)
			delete(rc.pendingV1RuleIds, id)
		}
	}

	log.Infof("[Cache] convert routing parse v1 => v2 count : %d", rc.bucketV2.convertV2Size())
}

func (rc *routingConfigCache) IsConvertFromV1(id string) (string, bool) {
	val, ok := rc.bucketV2.v1rulesToOld[id]
	return val, ok
}

func (rc *routingConfigCache) convertRoutingV1toV2(rule *model.RoutingConfig) ([]*v2.ExtendRoutingConfig, error) {
	svc := rc.serviceCache.GetServiceByID(rule.ID)
	if svc == nil {
		_svc, err := rc.storage.GetServiceByID(rule.ID)
		if err != nil {
			return nil, err
		}
		if _svc == nil {
			return nil, nil
		}

		svc = _svc
	}

	in, out, err := routingcommon.ConvertRoutingV1ToExtendV2(svc.Name, svc.Namespace, rule)
	if err != nil {
		return nil, err
	}

	ret := make([]*v2.ExtendRoutingConfig, 0, len(in)+len(out))
	ret = append(ret, in...)
	ret = append(ret, out...)

	return ret, nil
}

// convertRoutingV2toV1 v2 版本的路由规则转为 v1 版本进行返回给客户端，用于兼容 SDK 下发配置的场景
func (rc *routingConfigCache) convertRoutingV2toV1(entries map[routingLevel][]*v2.ExtendRoutingConfig,
	service, namespace string) *apiv1.Routing {

	level1 := entries[level1RoutingV2]
	sort.Slice(level1, func(i, j int) bool {
		return routingcommon.CompareRoutingV2(level1[i], level1[j])
	})

	level2 := entries[level2RoutingV2]
	sort.Slice(level2, func(i, j int) bool {
		return routingcommon.CompareRoutingV2(level2[i], level2[j])
	})

	level3 := entries[level3RoutingV2]
	sort.Slice(level3, func(i, j int) bool {
		return routingcommon.CompareRoutingV2(level3[i], level3[j])
	})

	level1inRoutes, level1outRoutes, level1Revisions := routingcommon.BuildV1RoutesFromV2(service, namespace, level1)
	level2inRoutes, level2outRoutes, level2Revisions := routingcommon.BuildV1RoutesFromV2(service, namespace, level2)
	level3inRoutes, level3outRoutes, level3Revisions := routingcommon.BuildV1RoutesFromV2(service, namespace, level3)

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
