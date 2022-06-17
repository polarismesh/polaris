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
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
)

const (
	// NamespaceName l5 name
	NamespaceName string = "namespace"
)

func init() {
	RegisterCache(NamespaceName, CacheNamespace)
}

// NamespaceCache 命名空间的 Cache 接口
type NamespaceCache interface {
	Cache

	// GetNamespace
	//  @param id
	//  @return *model.Namespace
	GetNamespace(id string) *model.Namespace

	// GetNamespacesByName
	//  @param names
	//  @return []*model.Namespace
	//  @return error
	GetNamespacesByName(names []string) []*model.Namespace

	// GetNamespaceList
	//  @return []*model.Namespace
	GetNamespaceList() []*model.Namespace
}

type namespaceCache struct {
	*baseCache
	storage     store.Store
	ids         *sync.Map
	lastTime    int64
	firstUpdate bool
	updater     *singleflight.Group
}

func newNamespaceCache(storage store.Store) NamespaceCache {
	return &namespaceCache{
		baseCache: newBaseCache(),
		storage:   storage,
	}
}

// initialize
//  @param c
//  @return error
func (nsCache *namespaceCache) initialize(c map[string]interface{}) error {
	nsCache.ids = new(sync.Map)
	nsCache.lastTime = 0
	nsCache.firstUpdate = true

	nsCache.updater = new(singleflight.Group)

	return nil
}

// update
//  @return error
func (nsCache *namespaceCache) update(storeRollbackSec time.Duration) error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := nsCache.updater.Do(NamespaceName, func() (interface{}, error) {
		return nil, nsCache.realUpdate(storeRollbackSec)
	})
	return err
}

func (nsCache *namespaceCache) realUpdate(storeRollbackSec time.Duration) error {

	lastMtime := time.Unix(nsCache.lastTime, 0).Add(storeRollbackSec)

	ret, err := nsCache.storage.GetMoreNamespaces(lastMtime)
	if err != nil {
		log.CacheScope().Error("[Cache][Namespace] get storage more", zap.Error(err))
		return err
	}
	nsCache.firstUpdate = false
	nsCache.setNamespaces(ret)
	return nil
}

func (nsCache *namespaceCache) setNamespaces(nsSlice []*model.Namespace) error {
	for index := range nsSlice {
		ns := nsSlice[index]
		if !ns.Valid {
			nsCache.ids.Delete(ns.Name)
		} else {
			nsCache.ids.Store(ns.Name, ns)
			nsCache.lastTime = int64(math.Max(float64(nsCache.lastTime), float64(ns.ModifyTime.Unix())))
		}
	}

	return nil
}

// clear
//  @return error
func (nsCache *namespaceCache) clear() error {
	nsCache.ids = new(sync.Map)
	nsCache.lastTime = 0
	nsCache.firstUpdate = true

	return nil
}

// name
//  @return string
func (nsCache *namespaceCache) name() string {
	return NamespaceName
}

// GetNamespace
//  @receiver nsCache
//  @param id
//  @return *model.Namespace
func (nsCache *namespaceCache) GetNamespace(id string) *model.Namespace {
	val, ok := nsCache.ids.Load(id)

	if !ok {
		return nil
	}

	return val.(*model.Namespace)
}

// GetNamespacesByName
//  @receiver nsCache
//  @param names
//  @return []*model.Namespace
//  @return error
func (nsCache *namespaceCache) GetNamespacesByName(names []string) []*model.Namespace {
	nsArr := make([]*model.Namespace, 0, len(names))

	for index := range names {
		name := names[index]
		if ns := nsCache.GetNamespace(name); ns != nil {
			nsArr = append(nsArr, ns)
		}
	}

	return nsArr
}

// GetNamespaceList
//  @receiver nsCache
//  @return []*model.Namespace
func (nsCache *namespaceCache) GetNamespaceList() []*model.Namespace {

	nsArr := make([]*model.Namespace, 0, 8)

	nsCache.ids.Range(func(key, value interface{}) bool {
		ns := value.(*model.Namespace)
		nsArr = append(nsArr, ns)

		return true
	})

	return nsArr

}
