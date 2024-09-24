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

package service

import (
	"context"
	"crypto/sha1"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

// serviceCache Service data cache implementation class
type serviceCache struct {
	*types.BaseCache

	storage store.Store
	// service_id -> service
	ids *utils.SyncMap[string, *model.Service]
	// namespace -> [serviceName -> service]
	names *utils.SyncMap[string, *utils.SyncMap[string, *model.Service]]
	// 兼容Cl5，sid -> name
	cl5Sid2Name *utils.SyncMap[string, string]
	// 兼容Cl5，name -> service
	cl5Names        *utils.SyncMap[string, *model.Service]
	alias           *serviceAliasBucket
	serviceList     *serviceNamespaceBucket
	disableBusiness bool
	needMeta        bool
	singleFlight    *singleflight.Group
	instCache       types.InstanceCache

	plock sync.RWMutex
	// service-id -> struct{}{}
	pendingServices *utils.SyncMap[string, struct{}]
	countLock       sync.Mutex
	// namespace -> model.NamespaceServiceCount
	namespaceServiceCnt *utils.SyncMap[string, *model.NamespaceServiceCount]

	lastMtimeLogged int64

	serviceCount     int64
	lastCheckAllTime int64

	revisionWorker *ServiceRevisionWorker

	cancel context.CancelFunc

	// exportNamespace 某个命名空间下的所有服务的可见性
	exportNamespace *utils.SyncMap[string, *utils.SyncSet[string]]
	// exportServices 某个服务对部分命名空间全部可见 exportNamespace -> svcName -> model.Service
	exportServices *utils.SyncMap[string, *utils.SyncMap[string, *model.Service]]

	subCtx *eventhub.SubscribtionContext
}

// NewServiceCache 返回一个serviceCache
func NewServiceCache(storage store.Store, cacheMgr types.CacheManager) types.ServiceCache {
	return &serviceCache{
		BaseCache:   types.NewBaseCache(storage, cacheMgr),
		storage:     storage,
		alias:       newServiceAliasBucket(),
		serviceList: newServiceNamespaceBucket(),
	}
}

// initialize 缓存对象初始化
func (sc *serviceCache) Initialize(opt map[string]interface{}) error {
	sc.instCache = sc.BaseCache.CacheMgr.GetCacher(types.CacheInstance).(*instanceCache)
	sc.singleFlight = new(singleflight.Group)
	sc.ids = utils.NewSyncMap[string, *model.Service]()
	sc.names = utils.NewSyncMap[string, *utils.SyncMap[string, *model.Service]]()
	sc.cl5Sid2Name = utils.NewSyncMap[string, string]()
	sc.cl5Names = utils.NewSyncMap[string, *model.Service]()
	sc.pendingServices = utils.NewSyncMap[string, struct{}]()
	sc.namespaceServiceCnt = utils.NewSyncMap[string, *model.NamespaceServiceCount]()
	sc.exportNamespace = utils.NewSyncMap[string, *utils.SyncSet[string]]()
	sc.exportServices = utils.NewSyncMap[string, *utils.SyncMap[string, *model.Service]]()
	ctx, cancel := context.WithCancel(context.Background())
	sc.cancel = cancel
	sc.revisionWorker = newRevisionWorker(sc, sc.instCache.(*instanceCache), opt)
	// 先启动revision计算协程
	go sc.revisionWorker.revisionWorker(ctx)
	subCtx, err := eventhub.SubscribeWithFunc(eventhub.CacheNamespaceEventTopic, sc.handleNamespaceChange)
	if err != nil {
		return err
	}
	sc.subCtx = subCtx
	if opt == nil {
		return nil
	}
	sc.disableBusiness, _ = opt["disableBusiness"].(bool)
	sc.needMeta, _ = opt["needMeta"].(bool)
	return nil
}

// LastMtime 最后一次更新时间
func (sc *serviceCache) Close() error {
	if err := sc.BaseCache.Close(); err != nil {
		return err
	}
	if sc.subCtx != nil {
		sc.subCtx.Cancel()
	}
	if sc.cancel != nil {
		sc.cancel()
	}
	return nil
}

// LastMtime 最后一次更新时间
func (sc *serviceCache) LastMtime() time.Time {
	return sc.BaseCache.LastMtime(sc.Name())
}

// update Service缓存更新函数
// service + service_metadata作为一个整体获取
func (sc *serviceCache) Update() error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := sc.singleFlight.Do(sc.Name(), func() (interface{}, error) {
		defer func() {
			sc.lastMtimeLogged = types.LogLastMtime(sc.lastMtimeLogged, sc.LastMtime().Unix(), "Service")
			sc.checkAll()
		}()
		return nil, sc.DoCacheUpdate(sc.Name(), sc.realUpdate)
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
	sc.ResetLastMtime(sc.Name())
}

func (sc *serviceCache) realUpdate() (map[string]time.Time, int64, error) {
	// 获取几秒前的全部数据
	start := time.Now()
	services, err := sc.storage.GetMoreServices(sc.LastFetchTime(), sc.IsFirstUpdate(), sc.disableBusiness, sc.needMeta)
	if err != nil {
		log.Errorf("[Cache][Service] update services err: %s", err.Error())
		return nil, -1, err
	}

	lastMtimes, update, del := sc.setServices(services)
	costTime := time.Since(start)
	log.Info("[Cache][Service] get more services", zap.Int("update", update), zap.Int("delete", del),
		zap.Time("last", sc.LastMtime()), zap.Duration("used", costTime))
	return lastMtimes, int64(len(services)), err
}

// clear 清理内部缓存数据
func (sc *serviceCache) Clear() error {
	sc.BaseCache.Clear()
	sc.ids = utils.NewSyncMap[string, *model.Service]()
	sc.names = utils.NewSyncMap[string, *utils.SyncMap[string, *model.Service]]()
	sc.cl5Sid2Name = utils.NewSyncMap[string, string]()
	sc.cl5Names = utils.NewSyncMap[string, *model.Service]()
	sc.pendingServices = utils.NewSyncMap[string, struct{}]()
	sc.namespaceServiceCnt = utils.NewSyncMap[string, *model.NamespaceServiceCount]()
	sc.alias = newServiceAliasBucket()
	sc.serviceList = newServiceNamespaceBucket()
	sc.exportNamespace = utils.NewSyncMap[string, *utils.SyncSet[string]]()
	sc.exportServices = utils.NewSyncMap[string, *utils.SyncMap[string, *model.Service]]()
	return nil
}

// name 获取资源名称
func (sc *serviceCache) Name() string {
	return types.ServiceName
}

func (sc *serviceCache) GetAliasFor(name string, namespace string) *model.Service {
	svc := sc.GetServiceByName(name, namespace)
	if svc == nil {
		return nil
	}
	if svc.Reference == "" {
		return nil
	}
	return sc.GetServiceByID(svc.Reference)
}

// GetServiceByID 根据服务ID获取服务数据
func (sc *serviceCache) GetServiceByID(id string) *model.Service {
	if id == "" {
		return nil
	}
	svc, ok := sc.ids.Load(id)
	if !ok {
		return nil
	}
	sc.fillServicePorts(svc)
	return svc
}

// GetOrLoadServiceByID 先从缓存获取服务，如果没有的话，再从存储层获取，并设置到 Cache 中
func (sc *serviceCache) GetOrLoadServiceByID(id string) *model.Service {
	if id == "" {
		return nil
	}
	value, ok := sc.ids.Load(id)
	if !ok {
		_, _, _ = sc.singleFlight.Do(id, func() (interface{}, error) {
			svc, err := sc.storage.GetServiceByID(id)
			if err == nil && svc != nil {
				sc.ids.Store(svc.ID, svc)
			}
			return svc, err
		})

		value, ok = sc.ids.Load(id)
		if !ok {
			return nil
		}
	}
	svc := value
	sc.fillServicePorts(svc)
	return svc
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
	value, ok := spaces.Load(name)
	if !ok {
		return nil
	}
	svc := value
	sc.fillServicePorts(svc)
	return svc
}

func (sc *serviceCache) fillServicePorts(svc *model.Service) {
	if svc == nil {
		return
	}
	if svc.Ports != "" {
		return
	}
	if sc.instCache == nil {
		return
	}
	ports := sc.instCache.GetServicePorts(svc.ID)
	if len(ports) == 0 {
		return
	}
	item := make([]string, 0, len(ports))
	for i := range ports {
		item = append(item, strconv.FormatUint(uint64(ports[i].Port), 10))
	}
	svc.ServicePorts = ports
	svc.Ports = strings.Join(item, ",")
}

// CleanNamespace 清除Namespace对应的服务缓存
func (sc *serviceCache) CleanNamespace(namespace string) {
	sc.names.Delete(namespace)
}

// IteratorServices 对缓存中的服务进行迭代
func (sc *serviceCache) IteratorServices(iterProc types.ServiceIterProc) error {
	var err error
	proc := func(k string, svc *model.Service) {
		sc.fillServicePorts(svc)
		if _, err = iterProc(k, svc); err != nil {
			return
		}
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

	return *val
}

// GetServicesCount 获取缓存中服务的个数
func (sc *serviceCache) GetServicesCount() int {
	return sc.ids.Len()
}

// ListServices get service list and revision by namespace
func (sc *serviceCache) ListServices(ns string) (string, []*model.Service) {
	return sc.serviceList.ListServices(ns)
}

// ListAllServices get all service and revision
func (sc *serviceCache) ListAllServices() (string, []*model.Service) {
	return sc.serviceList.ListAllServices()
}

// ListServiceAlias get all service alias by target service
func (sc *serviceCache) ListServiceAlias(namespace, name string) []*model.Service {
	return sc.alias.getServiceAliases(&model.Service{
		Namespace: namespace,
		Name:      name,
	})
}

// GetServiceByCl5Name obtains the corresponding SID according to cl5Name
func (sc *serviceCache) GetServiceByCl5Name(cl5Name string) *model.Service {
	value, ok := sc.cl5Names.Load(genCl5Name(cl5Name))
	if !ok {
		return nil
	}

	return value
}

// removeServices Delete the service data from the cache
func (sc *serviceCache) removeServices(service *model.Service) {
	// Delete the index of serviceid
	sc.ids.Delete(service.ID)
	// delete service item from name list
	sc.serviceList.removeService(service)
	// delete service all link alias info
	sc.alias.cleanServiceAlias(service)
	// delete pending count service task
	sc.pendingServices.Delete(service.ID)

	// Delete the index of servicename
	spaceName := service.Namespace
	if spaces, ok := sc.names.Load(spaceName); ok {
		spaces.Delete(service.Name)
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
	changeNs := make(map[string]struct{})
	svcCount := sc.serviceCount

	aliases := make([]*model.Service, 0, 32)

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

		if service.IsAlias() {
			aliases = append(aliases, service)
		}
		oldVal, exist := sc.ids.Load(service.ID)
		if oldVal != nil {
			service.OldExportTo = oldVal.ExportTo
		}

		spaceName := service.Namespace
		changeNs[spaceName] = struct{}{}
		// 发现有删除操作
		if !service.Valid {
			sc.removeServices(service)
			sc.notifyRevisionWorker(service.ID, false)
			del++
			svcCount--
			continue
		}

		update++
		if !exist {
			svcCount++
		}

		sc.ids.Store(service.ID, service)
		sc.serviceList.addService(service)
		sc.notifyRevisionWorker(service.ID, true)

		spaces, ok := sc.names.Load(spaceName)
		if !ok {
			spaces = utils.NewSyncMap[string, *model.Service]()
			sc.names.Store(spaceName, spaces)
		}
		spaces.Store(service.Name, service)

		/******兼容cl5******/
		sc.updateCl5SidAndNames(service)
		/******兼容cl5******/
	}

	if sc.serviceCount != svcCount {
		log.Infof("[Cache][Service] service count update from %d to %d",
			sc.serviceCount, svcCount)
		sc.serviceCount = svcCount
	}

	sc.postProcessServiceAlias(aliases)
	sc.postProcessUpdatedServices(changeNs)
	sc.postProcessServiceExports(services)
	sc.serviceList.reloadRevision()
	return map[string]time.Time{
		sc.Name(): time.Unix(lastMtime, 0),
	}, update, del
}

func (sc *serviceCache) notifyServiceCountReload(svcIds map[string]bool) {
	sc.plock.RLock()
	for k := range svcIds {
		sc.pendingServices.Store(k, struct{}{})
	}
	sc.plock.RUnlock()
	sc.postProcessUpdatedServices(map[string]struct{}{})
}

// appendServiceCountChangeNamespace
// Two Case
// Case ONE:
//  1. T1, ServiceCache pulls all of the service information
//  2. T2 time, instanecache pulls and updates the instance count information, and notify ServiceCache to
//     count the namespace count Reload
//
// - In this case, the instancecache notifies the servicecache, ServiceCache is a fixed count update.
// Case TWO:
//  1. T1, instanecache pulls and updates the instance count information, and notify ServiceCache to
//     make a namespace count Reload
//  2. T2 moments, ServiceCache pulls all of the service information
//
// - This situation, ServiceCache does not update the count, because the corresponding service object
// has not been cached, you need to put it in a PendingService waiting
// - Because under this case, WatchCountChangech is the first RELOAD notification from Instanecache,
// handled the reload notification of ServiceCache.
// - Therefore, for the reload notification of instancecache, you need to record the non-existing SVCID
// record in the Pending list; wait for the servicecache's Reload notification. after arriving,
// need to handle the last legacy PENDING calculation task.
func (sc *serviceCache) appendServiceCountChangeNamespace(changeNs map[string]struct{}) map[string]struct{} {
	sc.plock.Lock()
	defer sc.plock.Unlock()
	waitDel := map[string]struct{}{}
	sc.pendingServices.ReadRange(func(svcId string, _ struct{}) {
		svc, ok := sc.ids.Load(svcId)
		if !ok {
			return
		}
		changeNs[svc.Namespace] = struct{}{}
		waitDel[svcId] = struct{}{}
	})
	for svcId := range waitDel {
		sc.pendingServices.Delete(svcId)
	}
	return changeNs
}

func (sc *serviceCache) postProcessServiceAlias(aliases []*model.Service) {
	for i := range aliases {
		alias := aliases[i]

		_, aliasExist := sc.ids.Load(alias.ID)
		aliasFor, aliasForExist := sc.ids.Load(alias.Reference)
		if !aliasForExist {
			continue
		}

		if aliasExist {
			sc.alias.addServiceAlias(alias, aliasFor)
		} else {
			sc.alias.delServiceAlias(alias, aliasFor)
		}
	}
}

func (sc *serviceCache) postProcessUpdatedServices(affect map[string]struct{}) {
	affect = sc.appendServiceCountChangeNamespace(affect)
	sc.countLock.Lock()
	defer sc.countLock.Unlock()
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

		count, _ := sc.namespaceServiceCnt.ComputeIfAbsent(namespace, func(_ string) *model.NamespaceServiceCount {
			return &model.NamespaceServiceCount{}
		})

		// For count information under the Namespace involved in the change, it is necessary to re-come over.
		count.ServiceCount = 0
		count.InstanceCnt = &model.InstanceCount{}

		value.ReadRange(func(key string, svc *model.Service) {
			count.ServiceCount++
			insCnt := sc.instCache.GetInstancesCountByServiceID(svc.ID)
			count.InstanceCnt.TotalInstanceCount += insCnt.TotalInstanceCount
			count.InstanceCnt.HealthyInstanceCount += insCnt.HealthyInstanceCount
		})
	}
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

// GetVisibleServicesInOtherNamespace 查询是否存在别的命名空间下存在名称相同且可见的服务
func (sc *serviceCache) GetVisibleServicesInOtherNamespace(svcName, namespace string) []*model.Service {
	ret := make(map[string]*model.Service)
	// 根据服务级别的可见性进行查询, 先查询精确匹配
	sc.exportServices.ReadRange(func(exportToNs string, services *utils.SyncMap[string, *model.Service]) {
		if exportToNs != namespace && exportToNs != types.AllMatched {
			return
		}
		services.ReadRange(func(_ string, svc *model.Service) {
			if svc.Name == svcName && svc.Namespace != namespace {
				ret[svc.ID] = svc
			}
		})
	})

	// 根据命名空间级别的可见性进行查询, 先看精确的
	sc.exportNamespace.ReadRange(func(exportNs string, viewerNs *utils.SyncSet[string]) {
		exactMatch := viewerNs.Contains(namespace)
		allMatch := viewerNs.Contains(types.AllMatched)
		if !exactMatch && !allMatch {
			return
		}
		svc := sc.GetServiceByName(svcName, exportNs)
		if svc == nil {
			return
		}
		ret[svc.ID] = svc
	})

	visibleServices := make([]*model.Service, 0, len(ret))
	for _, svc := range ret {
		if svc.IsAlias() {
			// 如果是别名，那就看下指向的别名是不是已经在待返回列表，存在，跳过
			if _, ok := ret[svc.Reference]; ok {
				continue
			}
			// 如果不存在，那就找真实服务信息，进行返回
			svc = sc.GetServiceByID(svc.Reference)
			if svc == nil {
				continue
			}
		}
		visibleServices = append(visibleServices, svc)
	}

	return visibleServices
}

func (sc *serviceCache) postProcessServiceExports(services map[string]*model.Service) {

	for i := range services {
		svc := services[i]
		for exportNs := range svc.OldExportTo {
			if _, ok := svc.ExportTo[exportNs]; ok {
				continue
			}
			// 取消可见性
			if services, ok := sc.exportServices.Load(exportNs); ok {
				services.Delete(svc.ID)
			}
		}

		for exportNs := range svc.ExportTo {
			services, _ := sc.exportServices.ComputeIfAbsent(exportNs, func(k string) *utils.SyncMap[string, *model.Service] {
				return utils.NewSyncMap[string, *model.Service]()
			})
			services.Store(svc.ID, svc)
		}
	}
}

func (sc *serviceCache) handleNamespaceChange(ctx context.Context, args interface{}) error {
	event, ok := args.(*eventhub.CacheNamespaceEvent)
	if !ok {
		return nil
	}

	switch event.EventType {
	case eventhub.EventUpdated, eventhub.EventCreated:
		exportTo := event.Item.ServiceExportTo
		if len(exportTo) == 0 {
			sc.exportNamespace.Delete(event.Item.Name)
			return nil
		}
		viewers := utils.NewSyncSet[string]()
		sc.exportNamespace.Store(event.Item.Name, viewers)
		for viewerNs := range exportTo {
			viewers.Add(viewerNs)
		}
	case eventhub.EventDeleted:
		sc.exportNamespace.Delete(event.Item.Name)
	}
	return nil
}

func (sc *serviceCache) notifyRevisionWorker(serviceID string, valid bool) {
	revisionWorker := sc.revisionWorker
	if revisionWorker == nil {
		return
	}
	revisionWorker.Notify(serviceID, valid)
}

// GetRevisionWorker
func (sc *serviceCache) GetRevisionWorker() types.ServiceRevisionWorker {
	return sc.revisionWorker
}

// genCl5Name 兼容cl5Name
// 部分cl5Name与已有服务名存在冲突，因此给cl5Name加上一个前缀
func genCl5Name(name string) string {
	return "cl5." + name
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
	return types.ComputeRevisionBySlice(h, slice)
}

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

func newRevisionWorker(svcCache *serviceCache, instCache *instanceCache, opt map[string]interface{}) *ServiceRevisionWorker {
	revisionWorkerCnt, _ := opt["revisionWorkerCnt"].(int)
	revisionTaskChain, _ := opt["revisionTaskChain"].(int)
	if revisionWorkerCnt == 0 {
		revisionWorkerCnt = RevisionConcurrenceCount
	}
	if revisionTaskChain == 0 {
		revisionTaskChain = RevisionChanCount
	}

	return &ServiceRevisionWorker{
		svcCache:      svcCache,
		instCache:     instCache,
		workerCnt:     revisionWorkerCnt,
		comRevisionCh: make(chan *revisionNotify, revisionTaskChain),
		revisions:     map[string]string{},
	}
}

type ServiceRevisionWorker struct {
	svcCache  *serviceCache
	instCache *instanceCache

	workerCnt     int
	comRevisionCh chan *revisionNotify
	revisions     map[string]string // service id -> reversion (所有instance reversion 的累计计算值)
	lock          sync.RWMutex      // for revisions rw lock
}

func (sc *ServiceRevisionWorker) Notify(serviceID string, valid bool) {
	sc.comRevisionCh <- newRevisionNotify(serviceID, valid)
}

// GetServiceInstanceRevision 获取服务实例计算之后的revision
func (sc *ServiceRevisionWorker) GetServiceInstanceRevision(serviceID string) string {
	value, ok := sc.readRevisions(serviceID)
	if !ok {
		return ""
	}
	return value
}

// GetServiceRevisionCount 计算一下缓存中的revision的个数
func (sc *ServiceRevisionWorker) GetServiceRevisionCount() int {
	sc.lock.RLock()
	defer sc.lock.RUnlock()

	return len(sc.revisions)
}

// revisionWorker Cache中计算服务实例revision的worker
func (sc *ServiceRevisionWorker) revisionWorker(ctx context.Context) {
	log.Infof("[Cache] compute revision worker start")
	defer log.Infof("[Cache] compute revision worker done")

	// 启动多个协程来计算revision，后续可以通过启动参数控制
	for i := 0; i < sc.workerCnt; i++ {
		go func() {
			for {
				select {
				case req := <-sc.comRevisionCh:
					if ok := sc.processRevisionWorker(req); !ok {
						continue
					}

					// 每个计算完，等待2ms
					time.Sleep(2 * time.Millisecond)
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

// processRevisionWorker 处理revision计算的函数
func (sc *ServiceRevisionWorker) processRevisionWorker(req *revisionNotify) bool {
	if req == nil {
		log.Errorf("[Cache][Revision] get null revision request")
		return false
	}

	if req.serviceID == "" {
		log.Errorf("[Cache][Revision] get request service ID is empty")
		return false
	}

	if !req.valid {
		log.Infof("[Cache][Revision] service(%s) revision has all been removed", req.serviceID)
		sc.deleteRevisions(req.serviceID)
		return true
	}

	service := sc.svcCache.GetServiceByID(req.serviceID)
	if service == nil {
		// log.Errorf("[Cache][Revision] can not found service id(%s)", req.serviceID)
		return false
	}

	instances := sc.instCache.GetInstancesByServiceID(req.serviceID)
	revision, err := ComputeRevision(service.Revision, instances)
	if err != nil {
		log.Errorf(
			"[Cache] compute service id(%s) instances revision err: %s", req.serviceID, err.Error())
		return false
	}

	sc.setRevisions(req.serviceID, revision) // string -> string
	log.Debugf("[Cache] compute service id(%s) instances revision : %s", req.serviceID, revision)
	return true
}

func (sc *ServiceRevisionWorker) deleteRevisions(id string) {
	sc.lock.Lock()
	delete(sc.revisions, id)
	sc.lock.Unlock()
}

func (sc *ServiceRevisionWorker) setRevisions(key string, val string) {
	sc.lock.Lock()
	sc.revisions[key] = val
	sc.lock.Unlock()
}

func (sc *ServiceRevisionWorker) readRevisions(key string) (string, bool) {
	sc.lock.RLock()
	defer sc.lock.RUnlock()

	id, ok := sc.revisions[key]
	return id, ok
}
