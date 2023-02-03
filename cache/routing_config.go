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
	"sort"
	"time"

	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris/common/model"
	routingcommon "github.com/polarismesh/polaris/common/routing"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const (
	// RoutingConfigName router config name
	RoutingConfigName = "routingConfig"
)

type (
	// RoutingIterProc Method definition of routing rules
	RoutingIterProc func(key string, value *model.ExtendRouterConfig)

	// RoutingConfigCache Cache interface configured by routing
	RoutingConfigCache interface {
		Cache
		// GetRoutingConfigV1 Obtain routing configuration based on serviceid
		GetRoutingConfigV1(id, service, namespace string) (*apitraffic.Routing, error)
		// GetRoutingConfigV2 Obtain routing configuration based on serviceid
		GetRoutingConfigV2(id, service, namespace string) ([]*apitraffic.RouteRule, error)
		// GetRoutingConfigCount Get the total number of routing configuration cache
		GetRoutingConfigCount() int
		// GetRoutingConfigsV2 Query Route Configuration List
		GetRoutingConfigsV2(args *RoutingArgs) (uint32, []*model.ExtendRouterConfig, error)
		// IsConvertFromV1 Whether the current routing rules are converted from the V1 rule
		IsConvertFromV1(id string) (string, bool)
	}

	// routingConfigCache Routing rules cache
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

		// pendingV1RuleIds Records need to be converted from V1 to V2 routing rules ID
		pendingV1RuleIds map[string]struct{}
	}
)

// init From registration to cache list
func init() {
	RegisterCache(RoutingConfigName, CacheRoutingConfig)
}

// newRoutingConfigCache Return a object of operating RoutingConfigcache
func newRoutingConfigCache(s store.Store, serviceCache ServiceCache) *routingConfigCache {
	return &routingConfigCache{
		baseCache:        newBaseCache(),
		storage:          s,
		serviceCache:     serviceCache,
		pendingV1RuleIds: map[string]struct{}{},
	}
}

// initialize The function of implementing the cache interface
func (rc *routingConfigCache) initialize(_ map[string]interface{}) error {
	rc.firstUpdate = true

	rc.initBuckets()
	rc.lastMtimeV1 = time.Unix(0, 0)
	rc.lastMtimeV2 = time.Unix(0, 0)

	return nil
}

func (rc *routingConfigCache) initBuckets() {
	rc.bucketV1 = newRoutingBucketV1()
	rc.bucketV2 = newRoutingBucketV2()
}

// update The function of implementing the cache interface
func (rc *routingConfigCache) update(storeRollbackSec time.Duration) error {
	// Multiple thread competition, only one thread is updated
	_, err, _ := rc.singleFlight.Do("RoutingCache", func() (interface{}, error) {
		return nil, rc.realUpdate(storeRollbackSec)
	})
	return err
}

// update The function of implementing the cache interface
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

// clear The function of implementing the cache interface
func (rc *routingConfigCache) clear() error {
	rc.firstUpdate = true

	rc.initBuckets()
	rc.lastMtimeV1 = time.Unix(0, 0)
	rc.lastMtimeV2 = time.Unix(0, 0)

	return nil
}

// name The function of implementing the cache interface
func (rc *routingConfigCache) name() string {
	return RoutingConfigName
}

// GetRoutingConfigV1 Obtain routing configuration based on serviceid
// case 1: If there is only V2's routing rules, use V2
// case 2: If there is only V1's routing rules, use V1
// case 3: If there are routing rules for V1 and V2 at the same time, merge
func (rc *routingConfigCache) GetRoutingConfigV1(id, service, namespace string) (*apitraffic.Routing, error) {
	if id == "" && service == "" && namespace == "" {
		return nil, nil
	}

	v2rules := rc.bucketV2.listByServiceWithPredicate(service, namespace,
		func(item *model.ExtendRouterConfig) bool {
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

// formatRoutingResponseV1 Give the client's cache, no need to expose EXTENDINFO information data
func formatRoutingResponseV1(ret *apitraffic.Routing) *apitraffic.Routing {
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

// GetRoutingConfigV2 Obtain all V2 versions under the service information according to the service information
func (rc *routingConfigCache) GetRoutingConfigV2(_, service, namespace string) ([]*apitraffic.RouteRule, error) {
	v2rules := rc.bucketV2.listByServiceWithPredicate(service, namespace,
		func(item *model.ExtendRouterConfig) bool {
			return item.Enable
		})

	ret := make([]*apitraffic.RouteRule, 0, len(v2rules))
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
	// need to traverse the Routing cache bucket of V2 here
	rc.bucketV2.foreach(iterProc)
}

// GetRoutingConfigCount Get the total number of routing configuration cache
func (rc *routingConfigCache) GetRoutingConfigCount() int {
	return rc.bucketV2.size()
}

// setRoutingConfigV1 Update the data of the store to the cache
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
			// Delete the old V1 cache
			rc.bucketV1.delete(entry.ID)
			// Delete the cache converted to V2
			rc.bucketV2.deleteV1(entry.ID)
			// Delete the task ID of V1 to V2
			delete(rc.pendingV1RuleIds, entry.ID)
			continue
		}

		// Save to the old V1 cache
		rc.bucketV1.save(entry)
		rc.pendingV1RuleIds[entry.ID] = struct{}{}
	}

	if rc.lastMtimeV1.Unix() < lastMtimeV1 {
		rc.lastMtimeV1 = time.Unix(lastMtimeV1, 0)
	}
	return nil
}

// setRoutingConfigV2 Store V2 Router Caches
func (rc *routingConfigCache) setRoutingConfigV2(cs []*model.RouterConfig) error {
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
		// Save to the new V2 cache
		v2rule, err := rc.convertRoutingV1toV2(entry)
		if err != nil {
			log.Warn("[Cache] routing parse v1 => v2 fail, will try again next",
				zap.String("rule-id", entry.ID), zap.Error(err))
			continue
		}
		if v2rule == nil {
			log.Warn("[Cache] routing parse v1 => v2 is nil, will try again next",
				zap.String("rule-id", entry.ID))
			continue
		}
		rc.bucketV2.saveV1(entry, v2rule)
		delete(rc.pendingV1RuleIds, id)
	}
	log.Infof("[Cache] convert routing parse v1 => v2 count : %d", rc.bucketV2.convertV2Size())
}

func (rc *routingConfigCache) IsConvertFromV1(id string) (string, bool) {
	val, ok := rc.bucketV2.v1rulesToOld[id]
	return val, ok
}

func (rc *routingConfigCache) convertRoutingV1toV2(rule *model.RoutingConfig) ([]*model.ExtendRouterConfig, error) {
	svc := rc.serviceCache.GetServiceByID(rule.ID)
	if svc == nil {
		s, err := rc.storage.GetServiceByID(rule.ID)
		if err != nil {
			return nil, err
		}
		if s == nil {
			return nil, nil
		}
		svc = s
	}
	if svc.IsAlias() {
		return nil, fmt.Errorf("svc: %+v is alias", svc)
	}

	in, out, err := routingcommon.ConvertRoutingV1ToExtendV2(svc.Name, svc.Namespace, rule)
	if err != nil {
		return nil, err
	}

	ret := make([]*model.ExtendRouterConfig, 0, len(in)+len(out))
	ret = append(ret, in...)
	ret = append(ret, out...)

	return ret, nil
}

// convertRoutingV2toV1 The routing rules of the V2 version are converted to V1 version to return to the client,
// which is used to compatible with SDK issuance configuration.
func (rc *routingConfigCache) convertRoutingV2toV1(entries map[routingLevel][]*model.ExtendRouterConfig,
	service, namespace string) *apitraffic.Routing {
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

	inRoutes := make([]*apitraffic.Route, 0, len(level1inRoutes)+len(level2inRoutes)+len(level3inRoutes))
	inRoutes = append(inRoutes, level1inRoutes...)
	inRoutes = append(inRoutes, level2inRoutes...)
	inRoutes = append(inRoutes, level3inRoutes...)

	outRoutes := make([]*apitraffic.Route, 0, len(level1outRoutes)+len(level2outRoutes)+len(level3outRoutes))
	outRoutes = append(outRoutes, level1outRoutes...)
	outRoutes = append(outRoutes, level2outRoutes...)
	outRoutes = append(outRoutes, level3outRoutes...)

	return &apitraffic.Routing{
		Service:   utils.NewStringValue(service),
		Namespace: utils.NewStringValue(namespace),
		Inbounds:  inRoutes,
		Outbounds: outRoutes,
		Revision:  utils.NewStringValue(revision),
	}
}
