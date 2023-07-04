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
	"math"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const (
	// NamespaceName l5 name
	NamespaceName = "namespace"
)

func init() {
	RegisterCache(NamespaceName, CacheNamespace)
}

// NamespaceCache 命名空间的 Cache 接口
type NamespaceCache interface {
	Cache
	// GetNamespace
	GetNamespace(id string) *model.Namespace
	// GetNamespacesByName
	GetNamespacesByName(names []string) []*model.Namespace
	// GetNamespaceList
	GetNamespaceList() []*model.Namespace
}

type namespaceCache struct {
	*baseCache
	storage store.Store
	ids     *utils.SyncMap[string, *model.Namespace]
	updater *singleflight.Group
}

func newNamespaceCache(storage store.Store) NamespaceCache {
	return &namespaceCache{
		baseCache: newBaseCache(storage),
		storage:   storage,
	}
}

// initialize
func (nsCache *namespaceCache) initialize(c map[string]interface{}) error {
	nsCache.ids = utils.NewSyncMap[string, *model.Namespace]()
	nsCache.updater = new(singleflight.Group)
	return nil
}

// update
func (nsCache *namespaceCache) update() error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := nsCache.updater.Do(nsCache.name(), func() (interface{}, error) {
		return nil, nsCache.doCacheUpdate(nsCache.name(), nsCache.realUpdate)
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

	lastMtime := nsCache.LastMtime(nsCache.name()).Unix()

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
		nsCache.name(): time.Unix(lastMtime, 0),
	}
}

// clear
func (nsCache *namespaceCache) clear() error {
	nsCache.baseCache.clear()
	nsCache.ids = utils.NewSyncMap[string, *model.Namespace]()
	return nil
}

// name
//
//	@return string
func (nsCache *namespaceCache) name() string {
	return NamespaceName
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
