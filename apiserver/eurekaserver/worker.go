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

package eurekaserver

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	"github.com/polarismesh/polaris-server/service"
	"github.com/polarismesh/polaris-server/service/healthcheck"
)

func sha1s(bytes []byte) string {
	r := sha1.Sum(bytes)
	return hex.EncodeToString(r[:])
}

// ApplicationsWorker 应用缓存协程
type ApplicationsWorker struct {
	mutex *sync.Mutex

	started uint32

	waitCtx context.Context

	workerCancel context.CancelFunc

	interval time.Duration

	deltaExpireInterval time.Duration

	unhealthyExpireInterval time.Duration
	// 全量服务的缓存，数据结构为ApplicationsRespCache
	appsCache *atomic.Value
	// 增量数据缓存，数据结构为ApplicationsRespCache
	deltaCache *atomic.Value
	// vip缓存同步
	vipCacheMutex *sync.RWMutex
	// vip数据缓存，数据格式为VipCacheKey:ApplicationsRespCache
	vipCache map[VipCacheKey]*ApplicationsRespCache

	appBuilder *ApplicationsBuilder

	healthCheckServer *healthcheck.Server

	// 上一次清理增量缓存的时间
	deltaExpireTimesMilli int64
}

// NewApplicationsWorker 构造函数
func NewApplicationsWorker(interval time.Duration,
	deltaExpireInterval time.Duration, enableSelfPreservation bool,
	namingServer service.DiscoverServer, healthCheckServer *healthcheck.Server, namespace string) *ApplicationsWorker {
	appBuilder := &ApplicationsBuilder{
		namingServer:           namingServer,
		namespace:              namespace,
		enableSelfPreservation: enableSelfPreservation,
	}
	return &ApplicationsWorker{
		mutex:               &sync.Mutex{},
		interval:            interval,
		deltaExpireInterval: deltaExpireInterval,
		appsCache:           &atomic.Value{},
		deltaCache:          &atomic.Value{},
		vipCacheMutex:       &sync.RWMutex{},
		vipCache:            make(map[VipCacheKey]*ApplicationsRespCache),
		healthCheckServer:   healthCheckServer,
		appBuilder:          appBuilder,
	}
}

// IsStarted 是否已经启动
func (a *ApplicationsWorker) IsStarted() bool {
	return atomic.LoadUint32(&a.started) > 0
}

// getCachedApps 从缓存获取全量服务数据
func (a *ApplicationsWorker) getCachedApps() *ApplicationsRespCache {
	appsValue := a.appsCache.Load()
	if appsValue != nil {
		return appsValue.(*ApplicationsRespCache)
	}
	return nil
}

// GetCachedAppsWithLoad 从缓存中获取全量服务信息，如果不存在就读取
func (a *ApplicationsWorker) GetCachedAppsWithLoad() *ApplicationsRespCache {
	appsRespCache := a.getCachedApps()
	if appsRespCache == nil {
		ctx := a.StartWorker()
		if ctx != nil {
			<-ctx.Done()
		}
		appsRespCache = a.getCachedApps()
	}
	return appsRespCache
}

// GetDeltaApps 从缓存获取增量服务数据
func (a *ApplicationsWorker) GetDeltaApps() *ApplicationsRespCache {
	appsValue := a.deltaCache.Load()
	if appsValue != nil {
		return appsValue.(*ApplicationsRespCache)
	}
	return nil
}

// GetVipApps 从缓存中读取VIP资源
func (a *ApplicationsWorker) GetVipApps(key VipCacheKey) *ApplicationsRespCache {
	a.vipCacheMutex.RLock()
	res, ok := a.vipCache[key]
	a.vipCacheMutex.RUnlock()
	if ok {
		return res
	}
	cachedApps := a.GetCachedAppsWithLoad()
	a.vipCacheMutex.Lock()
	defer a.vipCacheMutex.Unlock()
	res, ok = a.vipCache[key]
	if ok {
		return res
	}
	res = BuildApplicationsForVip(&key, cachedApps)
	a.vipCache[key] = res
	return res
}

func (a *ApplicationsWorker) timingReloadAppsCache(workerCtx context.Context) {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()
	for {
		select {
		case <-workerCtx.Done():
			return
		case <-ticker.C:
			oldApps := a.getCachedApps()
			newApps := a.appBuilder.BuildApplications(oldApps)
			newDeltaApps := a.appBuilder.buildDeltaApps(oldApps, newApps, a.getLatestDeltaAppsCache())
			a.appsCache.Store(newApps)
			a.deltaCache.Store(newDeltaApps)
			a.clearExpiredVipResources()
		}
	}
}

func (a *ApplicationsWorker) clearExpiredVipResources() {
	expireIntervalSec := int64(a.interval / time.Second)
	a.vipCacheMutex.Lock()
	defer a.vipCacheMutex.Unlock()
	for key, respCache := range a.vipCache {
		curTimeSec := time.Now().Unix()
		if curTimeSec-respCache.createTimeSec >= expireIntervalSec {
			delete(a.vipCache, key)
		}
	}
}

func (a *ApplicationsWorker) getLatestDeltaAppsCache() *ApplicationsRespCache {
	var oldDeltaAppsCache *ApplicationsRespCache
	curTimeMs := time.Now().UnixNano() / 1e6
	diffTimeMs := curTimeMs - a.deltaExpireTimesMilli
	if diffTimeMs > 0 && diffTimeMs < a.deltaExpireInterval.Milliseconds() {
		oldDeltaAppsCache = a.GetDeltaApps()
	} else {
		a.deltaExpireTimesMilli = curTimeMs
	}
	return oldDeltaAppsCache
}

func diffApplication(oldApplication *Application, newApplication *Application) *Application {
	oldRevision := oldApplication.Revision
	newRevision := newApplication.Revision
	if len(oldRevision) > 0 && len(newRevision) > 0 && oldRevision == newRevision {
		// 完全相同，没有变更
		return nil
	}
	diffApplication := &Application{
		Name: newApplication.Name,
	}
	// 获取新增和修改
	newInstances := newApplication.Instance
	if len(newInstances) > 0 {
		for _, instance := range newInstances {
			oldInstance := oldApplication.GetInstance(instance.InstanceId)
			if oldInstance == nil {
				// 新增实例
				diffApplication.Instance = append(diffApplication.Instance, instance)
				continue
			}
			// 比较实际的实例是否发生了变更
			if oldInstance.Equals(instance) {
				continue
			}
			// 新创建一个instance
			diffApplication.Instance = append(diffApplication.Instance, instance.Clone(ActionModified))
		}
	}
	// 获取删除
	oldInstances := oldApplication.Instance
	if len(oldInstances) > 0 {
		for _, instance := range oldInstances {
			newInstance := newApplication.GetInstance(instance.InstanceId)
			if newInstance == nil {
				// 被删除了
				// 新创建一个instance
				diffApplication.Instance = append(diffApplication.Instance, instance.Clone(ActionDeleted))
			}
		}
	}
	if len(diffApplication.Instance) > 0 {
		return diffApplication
	}
	return nil
}

// StartWorker 启动缓存构建器
func (a *ApplicationsWorker) StartWorker() context.Context {
	if a.getCachedApps() != nil {
		return nil
	}
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if !atomic.CompareAndSwapUint32(&a.started, 0, 1) {
		return a.waitCtx
	}
	var waitCancel context.CancelFunc
	// 进行首次缓存构建
	a.waitCtx, waitCancel = context.WithCancel(context.Background())
	defer waitCancel()
	apps := a.appBuilder.BuildApplications(nil)
	a.appsCache.Store(apps)
	a.deltaCache.Store(apps)
	a.deltaExpireTimesMilli = time.Now().UnixNano() / 1e6
	// 开启定时任务构建
	var workerCtx context.Context
	workerCtx, a.workerCancel = context.WithCancel(context.Background())
	go a.timingReloadAppsCache(workerCtx)
	return nil
}

// Stop 结束任务
func (a *ApplicationsWorker) Stop() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if atomic.CompareAndSwapUint32(&a.started, 1, 0) {
		a.workerCancel()
	}
}
