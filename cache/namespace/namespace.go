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

package namespace

import (
	"math"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var (
	_ types.NamespaceCache = (*namespaceCache)(nil)
)

type namespaceCache struct {
	*types.BaseCache
	storage store.Store
	ids     *utils.SyncMap[string, *model.Namespace]
	updater *singleflight.Group
	// exportNamespace 某个命名空间下的所有服务的可见性
	exportNamespace *utils.SyncMap[string, *utils.SyncSet[string]]
}

func NewNamespaceCache(storage store.Store, cacheMgr types.CacheManager) types.NamespaceCache {
	return &namespaceCache{
		BaseCache: types.NewBaseCache(storage, cacheMgr),
		storage:   storage,
	}
}

// Initialize
func (nsCache *namespaceCache) Initialize(c map[string]interface{}) error {
	nsCache.ids = utils.NewSyncMap[string, *model.Namespace]()
	nsCache.updater = new(singleflight.Group)
	nsCache.exportNamespace = utils.NewSyncMap[string, *utils.SyncSet[string]]()
	return nil
}

// Update
func (nsCache *namespaceCache) Update() error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := nsCache.updater.Do(nsCache.Name(), func() (interface{}, error) {
		return nil, nsCache.DoCacheUpdate(nsCache.Name(), nsCache.realUpdate)
	})
	return err
}

func (nsCache *namespaceCache) realUpdate() (map[string]time.Time, int64, error) {
	var (
		lastTime = nsCache.LastFetchTime()
		ret, err = nsCache.storage.GetMoreNamespaces(lastTime)
	)
	if err != nil {
		log.Error("[Cache][Namespace] get storage more", zap.Error(err))
		return nil, -1, err
	}
	lastMtimes := nsCache.setNamespaces(ret)
	return lastMtimes, int64(len(ret)), nil
}

func (nsCache *namespaceCache) setNamespaces(nsSlice []*model.Namespace) map[string]time.Time {

	lastMtime := nsCache.LastMtime(nsCache.Name()).Unix()

	for index := range nsSlice {
		ns := nsSlice[index]
		oldNs, hasOldVal := nsCache.ids.Load(ns.Name)
		eventType := eventhub.EventCreated
		if !ns.Valid {
			eventType = eventhub.EventDeleted
			nsCache.ids.Delete(ns.Name)
		} else {
			if !hasOldVal {
				eventType = eventhub.EventCreated
			} else {
				eventType = eventhub.EventUpdated
			}
			nsCache.ids.Store(ns.Name, ns)
		}
		nsCache.handleNamespaceChange(eventType, oldNs, ns)
		_ = eventhub.Publish(eventhub.CacheNamespaceEventTopic, &eventhub.CacheNamespaceEvent{
			OldItem:   oldNs,
			Item:      ns,
			EventType: eventType,
		})
		lastMtime = int64(math.Max(float64(lastMtime), float64(ns.ModifyTime.Unix())))
	}

	return map[string]time.Time{
		nsCache.Name(): time.Unix(lastMtime, 0),
	}
}

func (nsCache *namespaceCache) handleNamespaceChange(et eventhub.EventType, oldItem, item *model.Namespace) {
	switch et {
	case eventhub.EventUpdated, eventhub.EventCreated:
		exportTo := item.ServiceExportTo
		viewer := utils.NewSyncSet[string]()
		for i := range exportTo {
			viewer.Add(i)
		}
		nsCache.exportNamespace.Store(item.Name, viewer)
	case eventhub.EventDeleted:
		nsCache.exportNamespace.Delete(item.Name)
	}
}

func (nsCache *namespaceCache) GetVisibleNamespaces(namespace string) []*model.Namespace {
	ret := make(map[string]*model.Namespace, 8)

	// 根据命名空间级别的可见性进行查询
	// 先看精确的
	nsCache.exportNamespace.Range(func(exportNs string, viewerNs *utils.SyncSet[string]) {
		exactMatch := viewerNs.Contains(namespace)
		allMatch := viewerNs.Contains(types.AllMatched)
		if !exactMatch && !allMatch {
			return
		}
		val := nsCache.GetNamespace(exportNs)
		if val != nil {
			ret[val.Name] = val
		}
	})

	values := make([]*model.Namespace, 0, len(ret))
	for _, item := range ret {
		values = append(values, item)
	}
	return values
}

// Clear .
func (nsCache *namespaceCache) Clear() error {
	nsCache.BaseCache.Clear()
	nsCache.ids = utils.NewSyncMap[string, *model.Namespace]()
	nsCache.exportNamespace = utils.NewSyncMap[string, *utils.SyncSet[string]]()
	return nil
}

// Name .
func (nsCache *namespaceCache) Name() string {
	return types.NamespaceName
}

// GetNamespace get namespace by id
func (nsCache *namespaceCache) GetNamespace(id string) *model.Namespace {
	val, ok := nsCache.ids.Load(id)
	if !ok {
		return nil
	}
	return val
}

// GetNamespacesByName batch get namespace by name
func (nsCache *namespaceCache) GetNamespacesByName(names []string) []*model.Namespace {
	nsArr := make([]*model.Namespace, 0, len(names))
	for _, name := range names {
		if ns := nsCache.GetNamespace(name); ns != nil {
			nsArr = append(nsArr, ns)
		}
	}

	return nsArr
}

// GetNamespaceList
//
//	@receiver nsCache
//	@return []*model.Namespace
func (nsCache *namespaceCache) GetNamespaceList() []*model.Namespace {
	nsArr := make([]*model.Namespace, 0, 8)

	nsCache.ids.Range(func(key string, ns *model.Namespace) {
		nsArr = append(nsArr, ns)
	})

	return nsArr
}
