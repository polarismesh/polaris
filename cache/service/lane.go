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
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	protoV2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

func NewLaneCache(storage store.Store, cacheMgr types.CacheManager) types.LaneCache {
	return &LaneCache{
		BaseCache: types.NewBaseCache(storage, cacheMgr),
	}
}

type LaneCache struct {
	*types.BaseCache
	// single .
	single singleflight.Group
	// groups id -> *model.LaneGroupProto
	rules *utils.SyncMap[string, *model.LaneGroupProto]
	// serviceRules namespace -> service -> []*model.LaneRuleProto
	serviceRules *utils.SyncMap[string, *utils.SyncMap[string, *utils.SyncMap[string, *model.LaneGroupProto]]]
	// revisions namespace -> service -> revision
	revisions *utils.SyncMap[string, *utils.SyncMap[string, string]]
}

// Initialize .
func (lc *LaneCache) Initialize(c map[string]interface{}) error {
	lc.serviceRules = utils.NewSyncMap[string, *utils.SyncMap[string, *utils.SyncMap[string, *model.LaneGroupProto]]]()
	lc.revisions = utils.NewSyncMap[string, *utils.SyncMap[string, string]]()
	lc.rules = utils.NewSyncMap[string, *model.LaneGroupProto]()
	lc.single = singleflight.Group{}
	return nil
}

// Update .
func (lc *LaneCache) Update() error {
	// 多个线程竞争，只有一个线程进行更新
	err, _ := lc.singleUpdate()
	return err
}

func (lc *LaneCache) singleUpdate() (error, bool) {
	// 多个线程竞争，只有一个线程进行更新
	_, err, shared := lc.single.Do(lc.Name(), func() (interface{}, error) {
		return nil, lc.DoCacheUpdate(lc.Name(), lc.realUpdate)
	})
	return err, shared
}

// Update .
func (lc *LaneCache) realUpdate() (map[string]time.Time, int64, error) {
	start := time.Now()

	// 获取泳道规则信息
	rules, err := lc.Store().GetMoreLaneGroups(lc.LastFetchTime(), lc.IsFirstUpdate())
	if err != nil {
		log.Errorf("[Cache] lane cache update err: %s", err.Error())
		return nil, -1, err
	}

	mtime, addCnt, updateCnt, delCnt := lc.setLaneRules(rules)
	log.Info("[Cache][Lane] get more lane rules",
		zap.Int("pull-from-store", len(rules)), zap.Int("add", addCnt), zap.Int("update", updateCnt),
		zap.Int("delete", delCnt), zap.Time("last", lc.LastMtime()), zap.Duration("used", time.Since(start)))
	return map[string]time.Time{
		lc.Name(): mtime,
	}, int64(len(rules)), err
}

func (lc *LaneCache) setLaneRules(items map[string]*model.LaneGroup) (time.Time, int, int, int) {
	lastMtime := lc.LastMtime().Unix()
	add := 0
	update := 0
	del := 0
	affectSvcs := map[string]map[string]struct{}{}

	for i := range items {
		item := items[i]
		saveVal, err := item.ToProto()
		if err != nil {
			log.Error("[Cache][Lane] unmarshal rule text to LaneRule spec", zap.Error(err))
			continue
		}
		if item.ModifyTime.Unix() > lastMtime {
			lastMtime = item.ModifyTime.Unix()
		}

		oldVal, exist := lc.rules.Load(item.ID)
		if !item.Valid {
			del++
			_, _ = lc.rules.Delete(item.ID)
			if exist {
				lc.processLaneRuleDelete(oldVal, affectSvcs)
			}
			continue
		}
		if exist {
			update++
		} else {
			add++
		}
		lc.rules.Store(item.ID, saveVal)
		lc.processLaneRuleUpsert(oldVal, saveVal, affectSvcs)
	}
	lc.postUpdateRevisions(affectSvcs)

	return time.Unix(lastMtime, 0), add, update, del
}

func (lc *LaneCache) processLaneRuleUpsert(old, item *model.LaneGroupProto, affectSvcs map[string]map[string]struct{}) {
	waitDelServices := map[string]map[string]struct{}{}
	addService := func(ns, svc string) {
		if _, ok := waitDelServices[ns]; !ok {
			waitDelServices[ns] = map[string]struct{}{}
		}
		waitDelServices[ns][svc] = struct{}{}
	}
	removeServiceIfExist := func(ns, svc string) {
		if _, ok := waitDelServices[ns]; !ok {
			waitDelServices[ns] = map[string]struct{}{}
		}
		if _, ok := waitDelServices[ns][svc]; ok {
			delete(waitDelServices[ns], svc)
		}
	}

	handle := func(rule *model.LaneGroupProto, serviceOp func(ns, svc string), ruleOp func(string, string, *model.LaneGroupProto)) {
		if rule == nil {
			return
		}

		for i := range rule.Proto.Destinations {
			dest := rule.Proto.Destinations[i]
			serviceOp(dest.Namespace, dest.Service)
			ruleOp(dest.Namespace, dest.Service, rule)
		}

		for i := range rule.Proto.Entries {
			entry := rule.Proto.Entries[i]
			switch model.TrafficEntryType(entry.Type) {
			case model.TrafficEntry_MicroService:
				selector := &apitraffic.ServiceSelector{}
				if err := anyToSelector(entry.Selector, selector); err != nil {
					continue
				}
				serviceOp(selector.Namespace, selector.Service)
				ruleOp(selector.Namespace, selector.Service, rule)
			case model.TrafficEntry_SpringCloudGateway:
				selector := &apitraffic.ServiceGatewaySelector{}
				if err := anyToSelector(entry.Selector, selector); err != nil {
					continue
				}
				serviceOp(selector.Namespace, selector.Service)
				ruleOp(selector.Namespace, selector.Service, rule)
			default:
				// do nothing
			}
		}
	}

	handle(item, addService, func(ns, svc string, group *model.LaneGroupProto) {
		if _, ok := affectSvcs[ns]; !ok {
			affectSvcs[ns] = map[string]struct{}{}
		}
		affectSvcs[ns][svc] = struct{}{}
		lc.upsertServiceRule(ns, svc, group)
	})
	handle(old, removeServiceIfExist, func(ns, svc string, group *model.LaneGroupProto) {
		if _, ok := affectSvcs[ns]; !ok {
			affectSvcs[ns] = map[string]struct{}{}
		}
		affectSvcs[ns][svc] = struct{}{}
	})

	for ns := range waitDelServices {
		for svc := range waitDelServices[ns] {
			lc.cleanServiceRule(ns, svc, old)
		}
	}
}

func (lc *LaneCache) processLaneRuleDelete(item *model.LaneGroupProto, affectSvcs map[string]map[string]struct{}) {
	message := item.Proto
	// 先清理 destinations
	for i := range message.Destinations {
		dest := message.Destinations[i]
		if _, ok := affectSvcs[dest.Namespace]; !ok {
			affectSvcs[dest.Namespace] = map[string]struct{}{}
		}
		affectSvcs[dest.Namespace][dest.Service] = struct{}{}
		lc.cleanServiceRule(dest.Namespace, dest.Service, item)
	}

	for i := range message.Entries {
		entry := message.Entries[i]
		var ns string
		var svc string
		switch model.TrafficEntryType(entry.Type) {
		case model.TrafficEntry_MicroService:
			selector := &apitraffic.ServiceSelector{}
			if err := anyToSelector(entry.Selector, selector); err != nil {
				continue
			}
			ns = selector.Namespace
			svc = selector.Service
			lc.cleanServiceRule(selector.Namespace, selector.Service, item)
		case model.TrafficEntry_SpringCloudGateway:
			selector := &apitraffic.ServiceGatewaySelector{}
			if err := anyToSelector(entry.Selector, selector); err != nil {
				continue
			}
			ns = selector.Namespace
			svc = selector.Service
			lc.cleanServiceRule(selector.Namespace, selector.Service, item)
		}
		if _, ok := affectSvcs[ns]; !ok {
			affectSvcs[ns] = map[string]struct{}{}
		}
		affectSvcs[ns][svc] = struct{}{}
	}
}

func (lc *LaneCache) upsertServiceRule(namespace, service string, item *model.LaneGroupProto) {
	namespaceContainer, _ := lc.serviceRules.ComputeIfAbsent(namespace,
		func(k string) *utils.SyncMap[string, *utils.SyncMap[string, *model.LaneGroupProto]] {
			return utils.NewSyncMap[string, *utils.SyncMap[string, *model.LaneGroupProto]]()
		})
	serviceContainer, _ := namespaceContainer.ComputeIfAbsent(service,
		func(k string) *utils.SyncMap[string, *model.LaneGroupProto] {
			return utils.NewSyncMap[string, *model.LaneGroupProto]()
		})
	serviceContainer.Store(item.ID, item)
}

func (lc *LaneCache) cleanServiceRule(namespace, service string, item *model.LaneGroupProto) {
	namespaceContainer, ok := lc.serviceRules.Load(namespace)
	if !ok {
		return
	}
	serviceContainer, ok := namespaceContainer.Load(service)
	if !ok {
		return
	}
	if item == nil {
		return
	}

	serviceContainer.Delete(item.ID)

	if serviceContainer.Len() == 0 {
		namespaceContainer.Delete(service)
	}
}

func (lc *LaneCache) postUpdateRevisions(affectSvcs map[string]map[string]struct{}) {
	for ns, svsList := range affectSvcs {
		nsContainer, ok := lc.serviceRules.Load(ns)
		if !ok {
			continue
		}
		lc.revisions.ComputeIfAbsent(ns, func(k string) *utils.SyncMap[string, string] {
			return utils.NewSyncMap[string, string]()
		})
		nsRevisions, _ := lc.revisions.Load(ns)
		for svc := range svsList {
			revisions := make([]string, 0, 32)
			svcContainer, ok := nsContainer.Load(svc)
			if !ok {
				continue
			}
			svcContainer.Range(func(key string, val *model.LaneGroupProto) {
				revisions = append(revisions, val.Revision)
			})
			revision, err := types.CompositeComputeRevision(revisions)
			if err != nil {
				continue
			}
			nsRevisions.Store(svc, revision)
		}
	}
}

func (lc *LaneCache) GetLaneRules(serviceKey *model.Service) ([]*model.LaneGroupProto, string) {
	namespaceContainer, ok := lc.serviceRules.Load(serviceKey.Namespace)
	if !ok {
		return []*model.LaneGroupProto{}, ""
	}
	serviceContainer, ok := namespaceContainer.Load(serviceKey.Name)
	if !ok {
		return []*model.LaneGroupProto{}, ""
	}
	ret := make([]*model.LaneGroupProto, 0, 32)
	serviceContainer.Range(func(ruleId string, val *model.LaneGroupProto) {
		ret = append(ret, val)
	})

	nsRevision, ok := lc.revisions.Load(serviceKey.Namespace)
	if !ok {
		return ret, ""
	}
	revision, _ := nsRevision.Load(serviceKey.Name)
	return ret, revision
}

func (lc *LaneCache) LastMtime() time.Time {
	return lc.BaseCache.LastMtime(lc.Name())
}

// Clear .
func (lc *LaneCache) Clear() error {
	lc.revisions = utils.NewSyncMap[string, *utils.SyncMap[string, string]]()
	lc.rules = utils.NewSyncMap[string, *model.LaneGroupProto]()
	lc.serviceRules = utils.NewSyncMap[string, *utils.SyncMap[string, *utils.SyncMap[string, *model.LaneGroupProto]]]()
	return nil
}

// Name .
func (lc *LaneCache) Name() string {
	return types.LaneRuleName
}

func anyToSelector(data *anypb.Any, msg proto.Message) error {
	if err := anypb.UnmarshalTo(data, proto.MessageV2(msg),
		protoV2.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}); err != nil {
		return err
	}
	return nil
}

// Query implements api.LaneCache.
func (lc *LaneCache) Query(context.Context, *types.LaneGroupArgs) (uint32, []*model.LaneGroupProto, error) {
	panic("unimplemented")
}

// GetRule implements api.LaneCache.
func (f *LaneCache) GetRule(id string) *model.LaneGroup {
	rule, _ := f.rules.Load(id)
	return rule.LaneGroup
}
