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

	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/healthcheck"
)

func sha1s(bytes []byte) string {
	r := sha1.Sum(bytes)
	return hex.EncodeToString(r[:])
}

type Lease struct {
	instance          *InstanceInfo
	lastUpdateTimeSec int64
}

// Expired check lease expired
func (l *Lease) Expired(curTimeSec int64, deltaExpireInterval time.Duration) bool {
	return curTimeSec-l.lastUpdateTimeSec >= deltaExpireInterval.Milliseconds()/1000
}

type ApplicationsWorkers struct {
	interval               time.Duration
	deltaExpireInterval    time.Duration
	enableSelfPreservation bool
	namingServer           service.DiscoverServer
	healthCheckServer      *healthcheck.Server
	workers                map[string]*ApplicationsWorker
	rwMutex                *sync.RWMutex
}

func NewApplicationsWorkers(interval time.Duration,
	deltaExpireInterval time.Duration, enableSelfPreservation bool,
	namingServer service.DiscoverServer, healthCheckServer *healthcheck.Server,
	namespaces ...string) *ApplicationsWorkers {
	workers := make(map[string]*ApplicationsWorker)
	for _, namespace := range namespaces {
		work := NewApplicationsWorker(interval, deltaExpireInterval, enableSelfPreservation,
			namingServer, healthCheckServer, namespace)
		workers[namespace] = work
	}
	return &ApplicationsWorkers{
		interval:               interval,
		deltaExpireInterval:    deltaExpireInterval,
		enableSelfPreservation: enableSelfPreservation,
		namingServer:           namingServer,
		healthCheckServer:      healthCheckServer,
		workers:                workers,
		rwMutex:                &sync.RWMutex{},
	}
}

func (a *ApplicationsWorkers) Get(namespace string) *ApplicationsWorker {
	a.rwMutex.RLock()
	work, exist := a.workers[namespace]
	a.rwMutex.RUnlock()
	if exist {
		return work
	}
	a.rwMutex.Lock()
	defer a.rwMutex.Unlock()

	work, exist = a.workers[namespace]
	if exist {
		return work
	}

	work = NewApplicationsWorker(a.interval, a.deltaExpireInterval, a.enableSelfPreservation,
		a.namingServer, a.healthCheckServer, namespace)
	a.workers[namespace] = work
	return work

}

func (a *ApplicationsWorkers) Stop() {
	for _, v := range a.workers {
		v.Stop()
	}
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

	// 增量缓存
	leases []*Lease
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
		leases:              make([]*Lease, 0),
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

func (a *ApplicationsWorker) cleanupExpiredLeases() {
	curTimeSec := commontime.CurrentMillisecond() / 1000
	var startIndex = -1
	for i, lease := range a.leases {
		if !lease.Expired(curTimeSec, a.deltaExpireInterval) {
			startIndex = i
			break
		}
		eurekalog.Infof("[Eureka]lease %s(%s) has expired, lastUpdateTime %d, curTimeSec %d",
			lease.instance.InstanceId, lease.instance.ActionType, lease.lastUpdateTimeSec, curTimeSec)
	}
	if startIndex == -1 && len(a.leases) > 0 {
		// all expired
		a.leases = make([]*Lease, 0)
	} else if startIndex > -1 {
		a.leases = a.leases[startIndex:]
	}
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
			newDeltaApps := a.buildDeltaApps(oldApps, newApps)
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

func diffApplicationInstances(curTimeSec int64, oldApplication *Application, newApplication *Application) []*Lease {
	var out []*Lease
	oldRevision := oldApplication.Revision
	newRevision := newApplication.Revision
	if len(oldRevision) > 0 && len(newRevision) > 0 && oldRevision == newRevision {
		// 完全相同，没有变更
		return out
	}
	// 获取新增和修改
	newInstances := newApplication.Instance
	if len(newInstances) > 0 {
		for _, instance := range newInstances {
			oldInstance := oldApplication.GetInstance(instance.InstanceId)
			if oldInstance == nil {
				// 新增实例
				out = addLease(out, &Lease{instance: instance.Clone(ActionAdded), lastUpdateTimeSec: curTimeSec})
				continue
			}
			// 比较实际的实例是否发生了变更
			if oldInstance.Equals(instance) {
				continue
			}
			// 新创建一个instance
			out = addLease(out, &Lease{instance: instance.Clone(ActionModified), lastUpdateTimeSec: curTimeSec})
		}
	}
	// 获取删除
	oldInstances := oldApplication.Instance
	if len(oldInstances) > 0 {
		for _, instance := range oldInstances {
			newInstance := newApplication.GetInstance(instance.InstanceId)
			if newInstance == nil {
				// 被删除了
				out = addLease(out, &Lease{instance: instance.Clone(ActionDeleted), lastUpdateTimeSec: curTimeSec})
			}
		}
	}
	return out
}

func addLease(out []*Lease, lease *Lease) []*Lease {
	eurekalog.Infof("[EUREKA] add delta instance %s(%s)", lease.instance.InstanceId, lease.instance.ActionType)
	out = append(out, lease)
	return out
}

func calculateDeltaInstances(oldAppsCache *ApplicationsRespCache, newAppsCache *ApplicationsRespCache) []*Lease {
	var out []*Lease
	newApps := newAppsCache.AppsResp.Applications
	curTimeSec := commontime.CurrentMillisecond() / 1000
	// 1. 处理服务新增场景
	if nil == oldAppsCache {
		applications := newApps.Application
		for _, app := range applications {
			for _, instance := range app.Instance {
				out = addLease(out, &Lease{instance: instance.Clone(ActionAdded), lastUpdateTimeSec: curTimeSec})
			}
		}
		return out
	}
	// 2. 处理服务变更场景
	if oldAppsCache.Revision != newAppsCache.Revision {
		oldApps := oldAppsCache.AppsResp.Applications
		applications := newApps.Application
		for _, application := range applications {
			var oldApplication = oldApps.GetApplication(application.Name)
			if oldApplication == nil {
				// 新增，全部加入
				for _, instance := range application.Instance {
					out = addLease(out, &Lease{instance: instance.Clone(ActionAdded), lastUpdateTimeSec: curTimeSec})
				}
				continue
			}
			// 修改，需要比较实例的变更
			leases := diffApplicationInstances(curTimeSec, oldApplication, application)
			if len(leases) > 0 {
				out = append(out, leases...)
			}
		}
		// 3. 处理服务删除场景
		oldApplications := oldApps.Application
		if len(oldApplications) > 0 {
			for _, application := range oldApplications {
				var newApplication = newApps.GetApplication(application.Name)
				if newApplication == nil {
					// 删除
					for _, instance := range application.Instance {
						out = addLease(out, &Lease{instance: instance.Clone(ActionDeleted), lastUpdateTimeSec: curTimeSec})
					}
				}
			}
		}
	}
	return out
}

func (a *ApplicationsWorker) buildDeltaApps(
	oldAppsCache *ApplicationsRespCache, newAppsCache *ApplicationsRespCache) *ApplicationsRespCache {
	// 1. 清理过期的增量缓存
	a.cleanupExpiredLeases()
	// 2. 构建新增的增量缓存
	leases := calculateDeltaInstances(oldAppsCache, newAppsCache)
	a.leases = append(a.leases, leases...)
	// 3. 创建新的delta对象
	var instCount int
	newApps := newAppsCache.AppsResp.Applications
	newDeltaApps := &Applications{
		VersionsDelta:  newApps.VersionsDelta,
		AppsHashCode:   newApps.AppsHashCode,
		Application:    make([]*Application, 0),
		ApplicationMap: make(map[string]*Application, 0),
	}
	// 4. 拷贝lease对象，对同一实例的事件去重，最新事件会覆盖之前的事件
	leaseMap := make(map[string]*Lease, len(a.leases))
	for _, lease := range a.leases {
		leaseMap[lease.instance.AppName+lease.instance.InstanceId] = lease
	}
	// 5.两次遍历，将delete事件放到最后，避免客户端hash code与服务端不一致
	for _, lease := range leaseMap {
		instance := lease.instance
		appName := instance.AppName
		var app *Application
		var ok bool
		if app, ok = newDeltaApps.ApplicationMap[appName]; !ok {
			app = &Application{
				Name: appName,
			}
			newDeltaApps.Application = append(newDeltaApps.Application, app)
			newDeltaApps.ApplicationMap[appName] = app
		}
		if instance.ActionType != ActionDeleted {
			app.Instance = append(app.Instance, instance)
			instCount++
		}
	}
	for _, lease := range leaseMap {
		instance := lease.instance
		appName := instance.AppName
		app := newDeltaApps.ApplicationMap[appName]
		if instance.ActionType == ActionDeleted {
			app.Instance = append(app.Instance, instance)
			instCount++
		}
	}
	return constructResponseCache(newDeltaApps, instCount, true)
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
	a.deltaCache.Store(a.buildDeltaApps(nil, apps))
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
