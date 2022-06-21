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
	"sync"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
)

const (
	// RoutingConfigName router config name
	RoutingConfigName = "routingConfig"
)

// RoutingConfigCache routing配置的cache接口
type RoutingConfigCache interface {
	Cache

	// GetRoutingConfig 根据ServiceID获取路由配置
	GetRoutingConfig(id string) *model.RoutingConfig

	// GetRoutingConfigCount 获取路由配置缓存的总个数
	GetRoutingConfigCount() int
}

// routingCache的实现
type routingConfigCache struct {
	*baseCache

	storage     store.Store
	ids         *sync.Map
	lastMtime   time.Time
	firstUpdate bool
}

// init 自注册到缓存列表
func init() {
	RegisterCache(RoutingConfigName, CacheRoutingConfig)
}

// newRoutingConfigCache 返回一个操作RoutingConfigCache的对象
func newRoutingConfigCache(s store.Store) *routingConfigCache {
	return &routingConfigCache{
		baseCache: newBaseCache(),
		storage:   s,
	}
}

// initialize 实现Cache接口的函数
func (rc *routingConfigCache) initialize(opt map[string]interface{}) error {
	rc.ids = new(sync.Map)
	rc.lastMtime = time.Unix(0, 0)
	rc.firstUpdate = true
	if opt == nil {
		return nil
	}
	return nil
}

// update 实现Cache接口的函数
func (rc *routingConfigCache) update(storeRollbackSec time.Duration) error {
	out, err := rc.storage.GetRoutingConfigsForCache(rc.lastMtime.Add(storeRollbackSec), rc.firstUpdate)
	if err != nil {
		log.CacheScope().Errorf("[Cache] routing config cache update err: %s", err.Error())
		return err
	}

	rc.firstUpdate = false
	return rc.setRoutingConfig(out)
}

// clear 实现Cache接口的函数
func (rc *routingConfigCache) clear() error {
	return nil
}

// name 实现Cache接口的函数
func (rc *routingConfigCache) name() string {
	return RoutingConfigName
}

// GetRoutingConfig 根据ServiceID获取路由配置
func (rc *routingConfigCache) GetRoutingConfig(id string) *model.RoutingConfig {
	// TODO
	if id == "" {
	}

	value, ok := rc.ids.Load(id)
	if !ok {
		return nil
	}

	return value.(*model.RoutingConfig)
}

// GetRoutingConfigCount 获取路由配置缓存的总个数
func (rc *routingConfigCache) GetRoutingConfigCount() int {
	count := 0
	rc.ids.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	return count
}

// setRoutingConfig 更新store的数据到cache中
func (rc *routingConfigCache) setRoutingConfig(cs []*model.RoutingConfig) error {
	if len(cs) == 0 {
		return nil
	}

	lastMtime := rc.lastMtime.Unix()
	for _, entry := range cs {
		if entry.ID == "" {
			continue
		}

		if entry.ModifyTime.Unix() > lastMtime {
			lastMtime = entry.ModifyTime.Unix()
		}

		if !entry.Valid {
			rc.ids.Delete(entry.ID)
			continue
		}

		rc.ids.Store(entry.ID, entry)
	}

	if rc.lastMtime.Unix() < lastMtime {
		rc.lastMtime = time.Unix(lastMtime, 0)
	}
	return nil
}
