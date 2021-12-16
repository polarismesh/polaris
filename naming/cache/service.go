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
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"sync"
	"time"
)

const (
	ServiceName = "service"
)

/**
 * ServiceIterProc 迭代回调函数
 */
type ServiceIterProc func(key string, value *model.Service) (bool, error)

/**
 * ServiceCache 服务数据缓存接口
 */
type ServiceCache interface {
	Cache

	// GetAllNamespaces 返回所有命名空间
	GetAllNamespaces() []string

	// GetServiceByID 根据ID查询服务信息
	GetServiceByID(id string) *model.Service

	// GetServiceByName 根据服务名查询服务信息
	GetServiceByName(name string, namespace string) *model.Service

	// IteratorServices 迭代缓存的服务信息
	IteratorServices(iterProc ServiceIterProc) error

	// GetServicesNames 获取所有服务缓存
	GetServicesCache() *sync.Map

	// GetServicesCount 获取缓存中服务的个数
	GetServicesCount() int

	// GetServiceByCl5Name 根据cl5Name获取对应的sid
	GetServiceByCl5Name(cl5Name string) *model.Service

	// GetServiceByFilter 通过filter在缓存中进行服务过滤
	GetServicesByFilter(serviceFilters *ServiceArgs,
		instanceFilters *store.InstanceArgs, offset, limit uint32) (uint32, []*model.EnhancedService, error)

	// Update 查询触发更新接口
	Update() error
}

/**
 * @brief 服务数据缓存实现类
 */
type serviceCache struct {
	storage         store.Store
	lastMtime       int64
	lastMtimeLogged int64
	firstUpdate     bool
	ids             *sync.Map // id -> service
	names           *sync.Map // space -> [serviceName -> service]
	cl5Sid2Name     *sync.Map // 兼容Cl5，sid -> name
	cl5Names        *sync.Map // 兼容Cl5，name -> service
	revisionCh      chan *revisionNotify
	disableBusiness bool
	needMeta        bool
	singleFlight    *singleflight.Group
	instCache       InstanceCache
}

/**
 * @brief 自注册到缓存列表
 */
func init() {
	RegisterCache(ServiceName, CacheService)
}

/**
 * @brief 返回一个serviceCache
 */
func newServiceCache(storage store.Store, ch chan *revisionNotify, instCache InstanceCache) *serviceCache {
	return &serviceCache{
		storage:    storage,
		revisionCh: ch,
		instCache:  instCache,
	}
}

/**
 * @brief 缓存对象初始化
 */
func (sc *serviceCache) initialize(opt map[string]interface{}) error {
	sc.singleFlight = new(singleflight.Group)
	sc.lastMtime = 0
	sc.ids = new(sync.Map)
	sc.names = new(sync.Map)
	sc.cl5Sid2Name = new(sync.Map)
	sc.cl5Names = new(sync.Map)
	sc.firstUpdate = true
	if opt == nil {
		return nil
	}
	sc.disableBusiness, _ = opt["disableBusiness"].(bool)
	sc.needMeta, _ = opt["needMeta"].(bool)
	return nil
}

// LastMtime 最后一次更新时间
func (sc *serviceCache) LastMtime() time.Time {
	return time.Unix(sc.lastMtime, 0)
}

/**
 * @brief Service缓存更新函数
 *
 * @note  service + service_metadata作为一个整体获取
 */
func (sc *serviceCache) update() error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := sc.singleFlight.Do(ServiceName, func() (interface{}, error) {
		defer func() {
			sc.lastMtimeLogged = logLastMtime(sc.lastMtimeLogged, sc.lastMtime, "Service")
		}()
		return nil, sc.realUpdate()
	})
	return err
}

func (sc *serviceCache) realUpdate() error {
	// 获取几秒前的全部数据
	start := time.Now()
	lastMtime := sc.LastMtime()
	services, err := sc.storage.GetMoreServices(lastMtime.Add(DefaultTimeDiff),
		sc.firstUpdate, sc.disableBusiness, sc.needMeta)
	if err != nil {
		log.Errorf("[Cache][Service] update services err: %s", err.Error())
		return err
	}

	sc.firstUpdate = false
	update, del := sc.setServices(services)
	log.Debug("[Cache][Service] get more services", zap.Int("update", update), zap.Int("delete", del),
		zap.Time("last", lastMtime), zap.Duration("used", time.Now().Sub(start)))
	return nil
}

/**
 * @brief 清理内部缓存数据
 */
func (sc *serviceCache) clear() error {
	sc.ids = new(sync.Map)
	sc.names = new(sync.Map)
	sc.cl5Sid2Name = new(sync.Map)
	sc.cl5Names = new(sync.Map)
	sc.lastMtime = 0
	return nil
}

/**
 * @brief 获取资源名称
 */
func (sc *serviceCache) name() string {
	return ServiceName
}

/**
 * GetServiceByID 根据服务ID获取服务数据
 */
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

/**
 * GetServiceByName 根据服务名获取服务数据
 */
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

/**
 * GetServicesCache 获取所有服务的缓存
 */
func (sc *serviceCache) GetServicesCache() *sync.Map {

	return sc.names
}

/**
 * IteratorServices 对缓存中的服务进行迭代
 */
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

// GetServicesCount 获取缓存中服务的个数
func (sc *serviceCache) GetServicesCount() int {
	count := 0
	sc.ids.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	return count
}

// GetServiceByCl5Name 根据cl5Name获取对应的sid
func (sc *serviceCache) GetServiceByCl5Name(cl5Name string) *model.Service {
	value, ok := sc.cl5Names.Load(genCl5Name(cl5Name))
	if !ok {
		return nil
	}

	return value.(*model.Service)
}

// 从缓存中删除service数据
func (sc *serviceCache) removeServices(service *model.Service) {
	// 删除serviceID的索引
	sc.ids.Delete(service.ID)

	// 删除serviceName的索引
	spaceName := service.Namespace
	if spaces, ok := sc.names.Load(spaceName); ok {
		spaces.(*sync.Map).Delete(service.Name)
	}

	/******兼容cl5******/
	if cl5Name, ok := sc.cl5Sid2Name.Load(service.Name); ok {
		sc.cl5Sid2Name.Delete(service.Name)
		sc.cl5Names.Delete(cl5Name)
	}
	/******兼容cl5******/
}

// 服务缓存更新
// 返回：更新数量，删除数量
func (sc *serviceCache) setServices(services map[string]*model.Service) (int, int) {
	if len(services) == 0 {
		return 0, 0
	}

	progress := 0
	update := 0
	del := 0
	lastMtime := sc.lastMtime
	for _, service := range services {
		progress++
		if progress%20000 == 0 {
			log.Infof("[Cache][Service] update service item progress(%d / %d)", progress, len(services))
		}
		serviceMtime := service.ModifyTime.Unix()
		if lastMtime < serviceMtime {
			lastMtime = serviceMtime
		}
		spaceName := service.Namespace
		// 发现有删除操作
		if !service.Valid {
			sc.removeServices(service)
			sc.revisionCh <- newRevisionNotify(service.ID, false)
			del++
			continue
		}

		update++
		sc.ids.Store(service.ID, service)
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

	if sc.lastMtime < lastMtime {
		sc.lastMtime = lastMtime
	}

	return update, del
}

// 更新cl5的服务数据
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
	return
}

// 兼容cl5Name
// 部分cl5Name与已有服务名存在冲突，因此给cl5Name加上一个前缀
func genCl5Name(name string) string {
	return "cl5." + name
}
