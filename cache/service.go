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
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

const (
	ServiceName = "service"
)

// ServiceIterProc 迭代回调函数
type ServiceIterProc func(key string, value *model.Service) (bool, error)

// ServiceCache 服务数据缓存接口
type ServiceCache interface {
	Cache

	// GetNamespaceCntInfo Return to the service statistics according to the namespace,
	// 	the count statistics and health instance statistics
	GetNamespaceCntInfo(namespace string) model.NamespaceServiceCount
	// GetAllNamespaces Return all namespaces
	GetAllNamespaces() []string
	// GetServiceByID According to ID query service information
	GetServiceByID(id string) *model.Service
	// GetServiceByName Inquiry service information according to service name
	GetServiceByName(name string, namespace string) *model.Service
	// IteratorServices Iterative Cache Service Information
	IteratorServices(iterProc ServiceIterProc) error
	// CleanNamespace Clear the cache of NameSpace
	CleanNamespace(namespace string)
	// GetServicesCount Get the number of services in the cache
	GetServicesCount() int
	// GetServiceByCl5Name Get the corresponding SID according to CL5name
	GetServiceByCl5Name(cl5Name string) *model.Service
	// GetServicesByFilter Serving the service filtering in the cache through Filter
	GetServicesByFilter(serviceFilters *ServiceArgs,
		instanceFilters *store.InstanceArgs, offset, limit uint32) (uint32, []*model.EnhancedService, error)
	// ListServices get service list and revision by namespace
	ListServices(ns string) (string, []*model.Service)
	// ListAllServices get all service and revision
	ListAllServices() (string, []*model.Service)
	// Update Query trigger update interface
	Update() error
}

// serviceCache Service data cache implementation class
type serviceCache struct {
	*baseCache

	storage             store.Store
	ids                 *sync.Map // service_id -> service
	names               *sync.Map // namespace -> [serviceName -> service]
	cl5Sid2Name         *sync.Map // 兼容Cl5，sid -> name
	cl5Names            *sync.Map // 兼容Cl5，name -> service
	serviceList         *serviceNamespaceBucket
	revisionCh          chan *revisionNotify
	disableBusiness     bool
	needMeta            bool
	singleFlight        *singleflight.Group
	instCache           InstanceCache
	countChangeCh       chan map[string]bool // Counting information requires a change event channel
	pendingServices     map[string]int8
	namespaceServiceCnt *sync.Map // namespace -> model.NamespaceServiceCount
	cancel              context.CancelFunc

	lastMtimeLogged int64

	serviceCount     int64
	lastCheckAllTime int64
}

// init 自注册到缓存列表
func init() {
	RegisterCache(ServiceName, CacheService)
}

// newServiceCache 返回一个serviceCache
func newServiceCache(storage store.Store, ch chan *revisionNotify, instCache InstanceCache) *serviceCache {
	return &serviceCache{
		baseCache:   newBaseCache(storage),
		storage:     storage,
		revisionCh:  ch,
		instCache:   instCache,
		serviceList: newServiceNamespaceBucket(),
	}
}

// initialize 缓存对象初始化
func (sc *serviceCache) initialize(opt map[string]interface{}) error {
	sc.singleFlight = new(singleflight.Group)
	sc.ids = new(sync.Map)
	sc.names = new(sync.Map)
	sc.cl5Sid2Name = new(sync.Map)
	sc.cl5Names = new(sync.Map)

	sc.countChangeCh = make(chan map[string]bool, 1024)
	sc.namespaceServiceCnt = new(sync.Map)

	ctx, cancel := context.WithCancel(context.Background())
	sc.cancel = cancel
	go sc.watchCountChangeCh(ctx)

	if opt == nil {
		return nil
	}

	sc.disableBusiness, _ = opt["disableBusiness"].(bool)
	sc.needMeta, _ = opt["needMeta"].(bool)
	return nil
}

// LastMtime 最后一次更新时间
func (sc *serviceCache) LastMtime() time.Time {
	return sc.baseCache.LastMtime(sc.name())
}

// update Service缓存更新函数
// service + service_metadata作为一个整体获取
func (sc *serviceCache) update() error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := sc.singleFlight.Do(sc.name(), func() (interface{}, error) {
		defer func() {
			sc.lastMtimeLogged = logLastMtime(sc.lastMtimeLogged, sc.LastMtime().Unix(), "Service")
			sc.checkAll()
		}()
		return nil, sc.doCacheUpdate(sc.name(), sc.realUpdate)
	})
	return err
}

func (sc *serviceCache) checkAll() {
	curTimeSec := time.Now().Unix()
	if curTimeSec-sc.lastCheckAllTime < checkAllIntervalSec {
		return
	}
	defer func() {
		sc.lastCheckAllTime = curTimeSec
	}()
	count, err := sc.storage.GetServicesCount()
	if err != nil {
		log.Errorf("[Cache][Service] get service count from storage err: %s", err.Error())
		return
	}
	if sc.serviceCount == int64(count) {
		return
	}
	log.Infof(
		"[Cache][Service] service count not match, expect %d, actual %d, fallback to load all",
		count, sc.serviceCount)
	sc.resetLastMtime(sc.name())
}

func (sc *serviceCache) realUpdate() (map[string]time.Time, int64, error) {
	// 获取几秒前的全部数据
	start := time.Now()
	services, err := sc.storage.GetMoreServices(sc.LastFetchTime(), sc.isFirstUpdate(), sc.disableBusiness, sc.needMeta)
	if err != nil {
		log.Errorf("[Cache][Service] update services err: %s", err.Error())
		return nil, -1, err
	}

	lastMtimes, update, del := sc.setServices(services)
	costTime := time.Since(start)
	if costTime > time.Second {
		log.Info(
			"[Cache][Service] get more services", zap.Int("update", update), zap.Int("delete", del),
			zap.Time("last", sc.LastMtime()), zap.Duration("used", costTime))
	}
	return lastMtimes, int64(len(services)), err
}

// clear 清理内部缓存数据
func (sc *serviceCache) clear() error {
	sc.baseCache.clear()
	sc.ids = new(sync.Map)
	sc.names = new(sync.Map)
	sc.cl5Sid2Name = new(sync.Map)
	sc.cl5Names = new(sync.Map)
	sc.namespaceServiceCnt = new(sync.Map)
	sc.pendingServices = make(map[string]int8)
	sc.serviceList = newServiceNamespaceBucket()
	return nil
}

// name 获取资源名称
func (sc *serviceCache) name() string {
	return ServiceName
}

// GetServiceByID 根据服务ID获取服务数据
func (sc *serviceCache) GetServiceByID(id string) *model.Service {
	if id == "" {
		return nil
	}

	value, ok := sc.ids.Load(id)
	if !ok {
		return nil
	}

	return value.(*model.Service)
}

// GetServiceByName 根据服务名获取服务数据
func (sc *serviceCache) GetServiceByName(name string, namespace string) *model.Service {
	if name == "" || namespace == "" {
		return nil
	}

	spaces, ok := sc.names.Load(namespace)
	if !ok {
		return nil
	}
	value, ok := spaces.(*sync.Map).Load(name)
	if !ok {
		return nil
	}

	return value.(*model.Service)
}

// CleanNamespace 清除Namespace对应的服务缓存
func (sc *serviceCache) CleanNamespace(namespace string) {
	sc.names.Delete(namespace)
}

// IteratorServices 对缓存中的服务进行迭代
func (sc *serviceCache) IteratorServices(iterProc ServiceIterProc) error {
	var (
		cont bool
		err  error
	)

	proc := func(k interface{}, v interface{}) bool {
		cont, err = iterProc(k.(string), v.(*model.Service))
		if err != nil {
			return false
		}
		return cont
	}
	sc.ids.Range(proc)
	return err
}

// GetNamespaceCntInfo Return to the service statistics according to the namespace,
//
//	the count statistics and health instance statistics
func (sc *serviceCache) GetNamespaceCntInfo(namespace string) model.NamespaceServiceCount {
	val, _ := sc.namespaceServiceCnt.Load(namespace)
	if val == nil {
		return model.NamespaceServiceCount{
			InstanceCnt: &model.InstanceCount{},
		}
	}

	return *val.(*model.NamespaceServiceCount)
}

// GetServicesCount 获取缓存中服务的个数
func (sc *serviceCache) GetServicesCount() int {
	count := 0
	sc.ids.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	return count
}

// ListServices get service list and revision by namespace
func (sc *serviceCache) ListServices(ns string) (string, []*model.Service) {
	return sc.serviceList.ListServices(ns)
}

// ListAllServices get all service and revision
func (sc *serviceCache) ListAllServices() (string, []*model.Service) {
	return sc.serviceList.ListAllServices()
}

// GetServiceByCl5Name obtains the corresponding SID according to cl5Name
func (sc *serviceCache) GetServiceByCl5Name(cl5Name string) *model.Service {
	value, ok := sc.cl5Names.Load(genCl5Name(cl5Name))
	if !ok {
		return nil
	}

	return value.(*model.Service)
}

// removeServices Delete the service data from the cache
func (sc *serviceCache) removeServices(service *model.Service) {
	// Delete the index of serviceid
	sc.ids.Delete(service.ID)

	// delete service item from name list
	sc.serviceList.removeService(service)

	// Delete the index of servicename
	spaceName := service.Namespace
	if spaces, ok := sc.names.Load(spaceName); ok {
		spaces.(*sync.Map).Delete(service.Name)
	}

	/******Compatible CL5******/
	if cl5Name, ok := sc.cl5Sid2Name.Load(service.Name); ok {
		sc.cl5Sid2Name.Delete(service.Name)
		sc.cl5Names.Delete(cl5Name)
	}
	/******Compatible CL5******/
}

// setServices 服务缓存更新
// 返回：更新数量，删除数量
func (sc *serviceCache) setServices(services map[string]*model.Service) (map[string]time.Time, int, int) {
	if len(services) == 0 {
		return nil, 0, 0
	}

	lastMtime := sc.LastMtime().Unix()

	progress := 0
	update := 0
	del := 0

	// 这里要记录 ns 的变动情况，避免由于 svc delete 之后，命名空间的服务计数无法更新
	changeNs := make(map[string]bool)
	svcCount := sc.serviceCount

	for _, service := range services {
		progress++
		if progress%20000 == 0 {
			log.Infof(
				"[Cache][Service] update service item progress(%d / %d)", progress, len(services))
		}
		serviceMtime := service.ModifyTime.Unix()
		if lastMtime < serviceMtime {
			lastMtime = serviceMtime
		}

		spaceName := service.Namespace
		changeNs[spaceName] = true
		// 发现有删除操作
		if !service.Valid {
			sc.removeServices(service)
			sc.revisionCh <- newRevisionNotify(service.ID, false)
			del++
			svcCount--
			continue
		}

		update++
		_, exist := sc.ids.Load(service.ID)
		if !exist {
			svcCount++
		}

		sc.ids.Store(service.ID, service)
		sc.serviceList.addService(service)
		sc.revisionCh <- newRevisionNotify(service.ID, true)

		spaces, ok := sc.names.Load(spaceName)
		if !ok {
			spaces = new(sync.Map)
			sc.names.Store(spaceName, spaces)
		}
		spaces.(*sync.Map).Store(service.Name, service)

		/******兼容cl5******/
		sc.updateCl5SidAndNames(service)
		/******兼容cl5******/
	}

	if sc.serviceCount != svcCount {
		log.Infof("[Cache][Service] service count update from %d to %d",
			sc.serviceCount, svcCount)
		sc.serviceCount = svcCount
	}

	sc.postProcessUpdatedServices(changeNs)
	return map[string]time.Time{
		sc.name(): time.Unix(lastMtime, 0),
	}, update, del
}

func (sc *serviceCache) notifyServiceCountReload(svcIds map[string]bool) {
	sc.countChangeCh <- svcIds
}

// watchCountChangeCh
// Two Case
// Case ONE:
//  1. T1, ServiceCache pulls all of the service information
//  2. T2 time, instanecache pulls and updates the instance count information, and notify ServiceCache to
//     count the namespace count Reload

// - In this case, the instancecache notifies the servicecache, ServiceCache is a fixed count update.

// Case TWO:

//  1. T1, instanecache pulls and updates the instance count information, and notify ServiceCache to
//     make a namespace count Reload

//  2. T2 moments, ServiceCache pulls all of the service information

// - This situation, ServiceCache does not update the count, because the corresponding service object
// has not been cached, you need to put it in a PendingService waiting
// - Because under this case, WatchCountChangech is the first RELOAD notification from Instanecache,
// handled the reload notification of ServiceCache.
// - Therefore, for the reload notification of instancecache, you need to record the non-existing SVCID
// record in the Pending list; wait for the servicecache's Reload notification. after arriving,
// need to handle the last legacy PENDING calculation task.
func (sc *serviceCache) watchCountChangeCh(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-sc.countChangeCh:
			affect := make(map[string]bool)

			if len(sc.pendingServices) != 0 {
				for svcId := range sc.pendingServices {
					svc, ok := sc.ids.Load(svcId)
					if !ok {
						log.Debugf("[Cache][Service] id : %s no found when reload namespace count", svcId)
						continue
					}
					affect[svc.(*model.Service).Namespace] = true
				}
			}

			newPendingServices := make(map[string]int8)
			for svcId := range event {
				svc, ok := sc.ids.Load(svcId)
				if !ok {
					newPendingServices[svcId] = 0
					continue
				}
				affect[svc.(*model.Service).Namespace] = true
			}

			sc.postProcessUpdatedServices(affect)
			sc.pendingServices = newPendingServices
		}
	}
}

func (sc *serviceCache) postProcessUpdatedServices(affect map[string]bool) {
	progress := 0
	for namespace := range affect {
		progress++
		if progress%10000 == 0 {
			log.Infof("[Cache][Service] namespace service detail count progress(%d / %d)", progress, len(affect))
		}
		// Construction of service quantity statistics
		value, ok := sc.names.Load(namespace)
		if !ok {
			sc.namespaceServiceCnt.Delete(namespace)
			continue
		}

		newVal, _ := sc.namespaceServiceCnt.LoadOrStore(namespace, &model.NamespaceServiceCount{})
		count := newVal.(*model.NamespaceServiceCount)

		// For count information under the Namespace involved in the change, it is necessary to re-come over.
		count.ServiceCount = 0
		count.InstanceCnt = &model.InstanceCount{}

		value.(*sync.Map).Range(func(key, item interface{}) bool {
			count.ServiceCount++
			service := item.(*model.Service)
			insCnt := sc.instCache.GetInstancesCountByServiceID(service.ID)
			count.InstanceCnt.TotalInstanceCount += insCnt.TotalInstanceCount
			count.InstanceCnt.HealthyInstanceCount += insCnt.HealthyInstanceCount
			return true
		})
	}

	sc.serviceList.reloadRevision()
}

// updateCl5SidAndNames 更新cl5的服务数据
func (sc *serviceCache) updateCl5SidAndNames(service *model.Service) {
	// 不是cl5服务的，不需要更新
	if _, ok := service.Meta["internal-cl5-sid"]; !ok {
		return
	}

	// service更新
	// service中不存在cl5Name，可以认为是该sid删除了cl5Name，删除缓存
	// service中存在cl5Name，则更新缓存
	cl5NameMeta, ok := service.Meta["internal-cl5-name"]
	sid := service.Name
	if !ok {
		if oldCl5Name, exist := sc.cl5Sid2Name.Load(sid); exist {
			sc.cl5Sid2Name.Delete(sid)
			sc.cl5Names.Delete(oldCl5Name)
		}
		return
	}

	// 更新的service，有cl5Name
	cl5Name := genCl5Name(cl5NameMeta)
	sc.cl5Sid2Name.Store(sid, cl5Name)
	sc.cl5Names.Store(cl5Name, service)
}

// genCl5Name 兼容cl5Name
// 部分cl5Name与已有服务名存在冲突，因此给cl5Name加上一个前缀
func genCl5Name(name string) string {
	return "cl5." + name
}

// WatchInstanceReload Listener 的一个简单实现
type WatchInstanceReload struct {
	// 实际的处理方法
	Handler func(val interface{})
}

// OnCreated callback when cache value created
func (fc *WatchInstanceReload) OnCreated(value interface{}) {

}

// OnUpdated callback when cache value updated
func (fc *WatchInstanceReload) OnUpdated(value interface{}) {

}

// OnDeleted callback when cache value deleted
func (fc *WatchInstanceReload) OnDeleted(value interface{}) {

}

// OnBatchCreated callback when cache value created
func (fc *WatchInstanceReload) OnBatchCreated(value interface{}) {

}

// OnBatchUpdated callback when cache value updated
func (fc *WatchInstanceReload) OnBatchUpdated(value interface{}) {
	fc.Handler(value)
}

// OnBatchDeleted callback when cache value deleted
func (fc *WatchInstanceReload) OnBatchDeleted(value interface{}) {

}
