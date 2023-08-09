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
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const ()

var (
	_ types.NamespaceCache = (*namespaceCache)(nil)
)

type namespaceCache struct {
	*types.BaseCache
	storage store.Store
	ids     *utils.SyncMap[string, *model.Namespace]
	updater *singleflight.Group
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
		if !ns.Valid {
			nsCache.ids.Delete(ns.Name)
		} else {
			nsCache.ids.Store(ns.Name, ns)
		}

		lastMtime = int64(math.Max(float64(lastMtime), float64(ns.ModifyTime.Unix())))
	}

	return map[string]time.Time{
		nsCache.Name(): time.Unix(lastMtime, 0),
	}
}

// clear
func (nsCache *namespaceCache) Clear() error {
	nsCache.BaseCache.Clear()
	nsCache.ids = utils.NewSyncMap[string, *model.Namespace]()
	return nil
}

// name
//
//	@return string
func (nsCache *namespaceCache) Name() string {
	return types.NamespaceName
}

// GetNamespace
//
//	@receiver nsCache
//	@param id
//	@return *model.Namespace
func (nsCache *namespaceCache) GetNamespace(id string) *model.Namespace {
	val, ok := nsCache.ids.Load(id)

	if !ok {
		return nil
	}

	return val
}

// GetNamespacesByName
//
//	@receiver nsCache
//	@param names
//	@return []*model.Namespace
//	@return error
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

	nsCache.ids.Range(func(key string, ns *model.Namespace) bool {
		nsArr = append(nsArr, ns)

		return true
	})

	return nsArr
}
