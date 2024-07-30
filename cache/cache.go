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
	"context"
	"fmt"
	"sync"
	"time"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var (
	cacheSet = map[string]int{}
)

const (
	// UpdateCacheInterval 缓存更新时间间隔
	UpdateCacheInterval = 1 * time.Second
)

var (
	ReportInterval = 1 * time.Second
)

// CacheManager 名字服务缓存
type CacheManager struct {
	storage  store.Store
	caches   []types.Cache
	needLoad *utils.SyncSet[string]
}

// Initialize 缓存对象初始化
func (nc *CacheManager) Initialize() error {
	if config.DiffTime != 0 {
		types.DefaultTimeDiff = -1 * (config.DiffTime.Abs())
	}
	if types.DefaultTimeDiff > 0 {
		return fmt.Errorf("cache diff time to pull store must negative number: %+v", types.DefaultTimeDiff)
	}
	return nil
}

// OpenResourceCache 开启资源缓存
func (nc *CacheManager) OpenResourceCache(entries ...types.ConfigEntry) error {
	for _, obj := range nc.caches {
		var entryItem *types.ConfigEntry
		for _, entry := range entries {
			if obj.Name() == entry.Name {
				entryItem = &entry
				break
			}
		}
		if entryItem == nil {
			continue
		}
		if err := obj.Initialize(entryItem.Option); err != nil {
			return err
		}
		nc.needLoad.Add(entryItem.Name)
	}
	return nil
}

// warmUp 缓存更新
func (nc *CacheManager) warmUp() error {
	var wg sync.WaitGroup
	entries := nc.needLoad.ToSlice()
	for i := range entries {
		name := entries[i]
		index, exist := cacheSet[name]
		if !exist {
			return fmt.Errorf("cache resource %s not exists", name)
		}
		wg.Add(1)
		go func(c types.Cache) {
			defer wg.Done()
			_ = c.Update()
		}(nc.caches[index])
	}

	wg.Wait()
	return nil
}

// clear 清除caches的所有缓存数据
func (nc *CacheManager) clear() error {
	for _, obj := range nc.caches {
		if err := obj.Clear(); err != nil {
			return err
		}
	}

	return nil
}

// Close 关闭所有的 Cache 缓存
func (nc *CacheManager) Close() error {
	for _, obj := range nc.caches {
		if err := obj.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Start 缓存对象启动协程，定时更新缓存
func (nc *CacheManager) Start(ctx context.Context) error {
	log.Infof("[Cache] cache goroutine start")

	// 启动的时候，先更新一版缓存
	log.Infof("[Cache] cache update now first time")
	if err := nc.warmUp(); err != nil {
		return err
	}
	log.Infof("[Cache] cache update done")

	// 启动协程，开始定时更新缓存数据
	entries := nc.needLoad.ToSlice()
	for i := range entries {
		name := entries[i]
		index, exist := cacheSet[name]
		if !exist {
			return fmt.Errorf("cache resource %s not exists", name)
		}
		// 每个缓存各自在自己的协程内部按照期望的缓存更新时间完成数据缓存刷新
		go func(c types.Cache) {
			ticker := time.NewTicker(nc.GetUpdateCacheInterval())
			for {
				select {
				case <-ticker.C:
					_ = c.Update()
				case <-ctx.Done():
					ticker.Stop()
					return
				}
			}
		}(nc.caches[index])
	}

	return nil
}

// Clear 主动清除缓存数据
func (nc *CacheManager) Clear() error {
	return nc.clear()
}

// GetUpdateCacheInterval 获取当前cache的更新间隔
func (nc *CacheManager) GetUpdateCacheInterval() time.Duration {
	return UpdateCacheInterval
}

// GetReportInterval 获取当前cache的更新间隔
func (nc *CacheManager) GetReportInterval() time.Duration {
	return ReportInterval
}

// Service 获取Service缓存信息
func (nc *CacheManager) Service() types.ServiceCache {
	return nc.caches[types.CacheService].(types.ServiceCache)
}

// Instance 获取Instance缓存信息
func (nc *CacheManager) Instance() types.InstanceCache {
	return nc.caches[types.CacheInstance].(types.InstanceCache)
}

// RoutingConfig 获取路由配置的缓存信息
func (nc *CacheManager) RoutingConfig() types.RoutingConfigCache {
	return nc.caches[types.CacheRoutingConfig].(types.RoutingConfigCache)
}

// CL5 获取l5缓存信息
func (nc *CacheManager) CL5() types.L5Cache {
	return nc.caches[types.CacheCL5].(types.L5Cache)
}

// RateLimit 获取限流规则缓存信息
func (nc *CacheManager) RateLimit() types.RateLimitCache {
	return nc.caches[types.CacheRateLimit].(types.RateLimitCache)
}

// CircuitBreaker 获取熔断规则缓存信息
func (nc *CacheManager) CircuitBreaker() types.CircuitBreakerCache {
	return nc.caches[types.CacheCircuitBreaker].(types.CircuitBreakerCache)
}

// FaultDetector 获取探测规则缓存信息
func (nc *CacheManager) FaultDetector() types.FaultDetectCache {
	return nc.caches[types.CacheFaultDetector].(types.FaultDetectCache)
}

// ServiceContract 获取服务契约缓存
func (nc *CacheManager) ServiceContract() types.ServiceContractCache {
	return nc.caches[types.CacheServiceContract].(types.ServiceContractCache)
}

// LaneRule 获取泳道规则缓存信息
func (nc *CacheManager) LaneRule() types.LaneCache {
	return nc.caches[types.CacheLaneRule].(types.LaneCache)
}

// User Get user information cache information
func (nc *CacheManager) User() types.UserCache {
	return nc.caches[types.CacheUser].(types.UserCache)
}

// AuthStrategy Get authentication cache information
func (nc *CacheManager) AuthStrategy() types.StrategyCache {
	return nc.caches[types.CacheAuthStrategy].(types.StrategyCache)
}

// Namespace Get namespace cache information
func (nc *CacheManager) Namespace() types.NamespaceCache {
	return nc.caches[types.CacheNamespace].(types.NamespaceCache)
}

// Client Get client cache information
func (nc *CacheManager) Client() types.ClientCache {
	return nc.caches[types.CacheClient].(types.ClientCache)
}

// ConfigFile get config file cache information
func (nc *CacheManager) ConfigFile() types.ConfigFileCache {
	return nc.caches[types.CacheConfigFile].(types.ConfigFileCache)
}

// ConfigGroup get config group cache information
func (nc *CacheManager) ConfigGroup() types.ConfigGroupCache {
	return nc.caches[types.CacheConfigGroup].(types.ConfigGroupCache)
}

// Gray get Gray cache information
func (nc *CacheManager) Gray() types.GrayCache {
	return nc.caches[types.CacheGray].(types.GrayCache)
}

// Role get Role cache information
func (nc *CacheManager) Role() types.RoleCache {
	return nc.caches[types.CacheRole].(types.RoleCache)
}

// GetCacher get types.Cache impl
func (nc *CacheManager) GetCacher(cacheIndex types.CacheIndex) types.Cache {
	return nc.caches[cacheIndex]
}

func (nc *CacheManager) RegisterCacher(cacheType types.CacheIndex, item types.Cache) {
	nc.caches[cacheType] = item
}

// GetStore get store
func (nc *CacheManager) GetStore() store.Store {
	return nc.storage
}

// RegisterCache 注册缓存资源
func RegisterCache(name string, index types.CacheIndex) {
	if _, exist := cacheSet[name]; exist {
		log.Warnf("existed cache resource: name = %s", name)
	}

	cacheSet[name] = int(index)
}
