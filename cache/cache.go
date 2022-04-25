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
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
)

var (
	cacheSet = make(map[string]int)
)

const (
	// CacheNamespace int = iota
	// CacheBusiness
	CacheService int = iota
	CacheInstance
	CacheRoutingConfig
	CacheCL5
	CacheRateLimit
	CacheCircuitBreaker
	CacheUser
	CacheAuthStrategy
	CacheNamespace
	CacheClient

	CacheLast
)

const (
	// DefaultTimeDiff default time diff
	DefaultTimeDiff = -1 * time.Second * 10
)

// Cache 缓存接口
type Cache interface {

	// initialize
	// @param c
	// @return error
	initialize(c map[string]interface{}) error

	// update
	// @return error
	update() error

	// clear
	//  @return error
	clear() error

	// name
	//  @return string
	name() string
}

const (
	// UpdateCacheInterval 缓存更新时间间隔
	UpdateCacheInterval = 1 * time.Second
)

const (
	// RevisionConcurrenceCount Revision计算的并发线程数
	RevisionConcurrenceCount = 64
	// RevisionChanCount 存储revision计算的通知管道，可以稍微设置大一点
	RevisionChanCount = 102400
)

// 更新revision的结构体
type revisionNotify struct {
	serviceID string
	valid     bool
}

// create new revision notify
func newRevisionNotify(serviceID string, valid bool) *revisionNotify {
	return &revisionNotify{
		serviceID: serviceID,
		valid:     valid,
	}
}

// NamingCache 名字服务缓存
type NamingCache struct {
	storage store.Store
	caches  []Cache

	comRevisionCh chan *revisionNotify
	revisions     *sync.Map // service id -> reversion (所有instance reversion 的累计计算值)
}

// initialize 缓存对象初始化
func (nc *NamingCache) initialize() error {
	for _, obj := range nc.caches {
		var option map[string]interface{}
		for _, entry := range config.Resources {
			if obj.name() == entry.Name {
				option = entry.Option
				break
			}
		}
		if err := obj.initialize(option); err != nil {
			return err
		}
	}

	return nil
}

// update 缓存更新
func (nc *NamingCache) update() error {
	var wg sync.WaitGroup
	for _, entry := range config.Resources {
		index, exist := cacheSet[entry.Name]
		if !exist {
			return fmt.Errorf("cache resource %s not exists", entry.Name)
		}
		wg.Add(1)
		go func(c Cache) {
			defer wg.Done()
			_ = c.update()
		}(nc.caches[index])
	}

	wg.Wait()
	return nil
}

// clear 清除caches的所有缓存数据
func (nc *NamingCache) clear() error {
	for _, obj := range nc.caches {
		if err := obj.clear(); err != nil {
			return err
		}
	}

	return nil
}

// Start 缓存对象启动协程，定时更新缓存
func (nc *NamingCache) Start(ctx context.Context) error {
	log.CacheScope().Infof("[Cache] cache goroutine start")
	// 先启动revision计算协程
	go nc.revisionWorker(ctx)

	// 启动的时候，先更新一版缓存
	log.CacheScope().Infof("[Cache] cache update now first time")
	if err := nc.update(); err != nil {
		return err
	}
	log.CacheScope().Infof("[Cache] cache update done")

	// 启动协程，开始定时更新缓存数据
	go func() {
		ticker := time.NewTicker(nc.GetUpdateCacheInterval())
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				_ = nc.update()
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Clear 主动清除缓存数据
func (nc *NamingCache) Clear() error {
	nc.revisions = new(sync.Map)
	return nc.clear()
}

// revisionWorker Cache中计算服务实例revision的worker
func (nc *NamingCache) revisionWorker(ctx context.Context) {
	log.CacheScope().Infof("[Cache] compute revision worker start")
	defer log.CacheScope().Infof("[Cache] compute revision worker done")

	processFn := func() {
		for {
			select {
			case req := <-nc.comRevisionCh:
				if ok := nc.processRevisionWorker(req); !ok {
					continue
				}

				// 每个计算完，等待2ms
				time.Sleep(time.Millisecond * 2)
			case <-ctx.Done():
				return
			}
		}
	}

	// 启动多个协程来计算revision，后续可以通过启动参数控制
	for i := 0; i < RevisionConcurrenceCount; i++ {
		go processFn()
	}
}

// processRevisionWorker 处理revision计算的函数
func (nc *NamingCache) processRevisionWorker(req *revisionNotify) bool {
	if req == nil {
		log.CacheScope().Errorf("[Cache][Revision] get null revision request")
		return false
	}

	if req.serviceID == "" {
		log.CacheScope().Errorf("[Cache][Revision] get request service ID is empty")
		return false
	}

	if !req.valid {
		log.CacheScope().Infof("[Cache][Revision] service(%s) revision has all been removed", req.serviceID)
		nc.revisions.Delete(req.serviceID)
		return true
	}

	service := nc.Service().GetServiceByID(req.serviceID)
	if service == nil {
		// log.Errorf("[Cache][Revision] can not found service id(%s)", req.serviceID)
		return false
	}

	instances := nc.Instance().GetInstancesByServiceID(req.serviceID)
	revision, err := ComputeRevision(service.Revision, instances)
	if err != nil {
		log.CacheScope().Errorf(
			"[Cache] compute service id(%s) instances revision err: %s", req.serviceID, err.Error())
		return false
	}
	nc.revisions.Store(req.serviceID, revision) // string -> string
	return true
}

// GetUpdateCacheInterval 获取当前cache的更新间隔
func (nc *NamingCache) GetUpdateCacheInterval() time.Duration {
	return UpdateCacheInterval
}

// GetServiceInstanceRevision 获取服务实例计算之后的revision
func (nc *NamingCache) GetServiceInstanceRevision(serviceID string) string {
	value, ok := nc.revisions.Load(serviceID)
	if !ok {
		return ""
	}

	return value.(string)
}

// GetServiceRevisionCount 计算一下缓存中的revision的个数
func (nc *NamingCache) GetServiceRevisionCount() int {
	count := 0
	nc.revisions.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	return count
}

// Service 获取Service缓存信息
func (nc *NamingCache) Service() ServiceCache {
	return nc.caches[CacheService].(ServiceCache)
}

// Instance 获取Instance缓存信息
func (nc *NamingCache) Instance() InstanceCache {
	return nc.caches[CacheInstance].(InstanceCache)
}

// RoutingConfig 获取路由配置的缓存信息
func (nc *NamingCache) RoutingConfig() RoutingConfigCache {
	return nc.caches[CacheRoutingConfig].(RoutingConfigCache)
}

// CL5 获取l5缓存信息
func (nc *NamingCache) CL5() L5Cache {
	return nc.caches[CacheCL5].(L5Cache)
}

// RateLimit 获取限流规则缓存信息
func (nc *NamingCache) RateLimit() RateLimitCache {
	return nc.caches[CacheRateLimit].(RateLimitCache)
}

// CircuitBreaker 获取熔断规则缓存信息
func (nc *NamingCache) CircuitBreaker() CircuitBreakerCache {
	return nc.caches[CacheCircuitBreaker].(CircuitBreakerCache)
}

// User Get user information cache information
//  @receiver nc
//  @return UserCache
func (nc *NamingCache) User() UserCache {
	return nc.caches[CacheUser].(UserCache)
}

// AuthStrategy Get authentication cache information
//  @receiver nc
//  @return StrategyCache
func (nc *NamingCache) AuthStrategy() StrategyCache {
	return nc.caches[CacheAuthStrategy].(StrategyCache)
}

// Namespace Get namespace cache information
//  @receiver nc
//  @return NamespaceCache
func (nc *NamingCache) Namespace() NamespaceCache {
	return nc.caches[CacheNamespace].(NamespaceCache)
}

// GetStore get store
func (nc *NamingCache) GetStore() store.Store {
	return nc.storage
}

// Client Get client cache information
//  @receiver nc
//  @return ClientCache
func (nc *NamingCache) Client() ClientCache {
	return nc.caches[CacheClient].(ClientCache)
}

// ComputeRevision 计算唯一的版本标识
func ComputeRevision(serviceRevision string, instances []*model.Instance) (string, error) {
	h := sha1.New()
	if _, err := h.Write([]byte(serviceRevision)); err != nil {
		return "", err
	}

	var slice sort.StringSlice
	for _, item := range instances {
		slice = append(slice, item.Revision())
	}
	if len(slice) > 0 {
		slice.Sort()
	}
	for _, revision := range slice {
		if _, err := h.Write([]byte(revision)); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// RegisterCache 注册缓存资源
func RegisterCache(name string, index int) {
	if _, exist := cacheSet[name]; exist {
		panic(fmt.Sprintf("existed cache resource: name = %s", name))
	}

	cacheSet[name] = index
}

const mtimeLogIntervalSec = 120

// logLastMtime 定时打印mtime更新结果
func logLastMtime(lastMtimeLogged int64, lastMtime int64, prefix string) int64 {
	curTimeSec := time.Now().Unix()
	if lastMtimeLogged == 0 || curTimeSec-lastMtimeLogged >= mtimeLogIntervalSec {
		lastMtimeLogged = curTimeSec
		log.CacheScope().Infof("[Cache][%s] current lastMtime is %s", prefix, time.Unix(lastMtime, 0))
	}
	return lastMtimeLogged
}
