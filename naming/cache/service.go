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

	// 根据ID查询服务信息
	GetServiceByID(id string) *model.Service

	// 根据服务名查询服务信息
	GetServiceByName(name string, namespace string) *model.Service

	// 迭代缓存的服务信息
	IteratorServices(iterProc ServiceIterProc) error

	// 获取缓存中服务的个数
	GetServicesCount() int

	// 根据cl5Name获取对应的sid
	GetServiceByCl5Name(cl5Name string) *model.Service
}

/**
 * @brief 服务数据缓存实现类
 */
type serviceCache struct {
	storage         store.Store
	lastMtime       time.Time
	firstUpdate     bool
	ids             *sync.Map
	names           *sync.Map
	cl5Sid2Name     *sync.Map // 兼容Cl5，sid -> name
	cl5Names        *sync.Map // 兼容Cl5，name -> service
	revisionCh      chan *revisionNotify
	disableBusiness bool
	needMeta        bool
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
func newServiceCache(storage store.Store, ch chan *revisionNotify) *serviceCache {
	return &serviceCache{
		storage:    storage,
		revisionCh: ch,
	}
}

/**
 * @brief 缓存对象初始化
 */
func (sc *serviceCache) initialize(opt map[string]interface{}) error {
	sc.lastMtime = time.Unix(0, 0)
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

/**
 * @brief Service缓存更新函数
 *
 * @note  service + service_metadata作为一个整体获取
 */
func (sc *serviceCache) update() error {
	// 获取几秒前的全部数据
	start := time.Now()
	services, err := sc.storage.GetMoreServices(sc.lastMtime.Add(DefaultTimeDiff),
		sc.firstUpdate, sc.disableBusiness, sc.needMeta)
	if err != nil {
		log.Errorf("[Cache][Service] update services err: %s", err.Error())
		return err
	}

	sc.firstUpdate = false
	update, del := sc.setServices(services)
	log.Debug("[Cache][Service] get more services", zap.Int("update", update), zap.Int("delete", del),
		zap.Time("last", sc.lastMtime), zap.Duration("used", time.Now().Sub(start)))
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
	sc.lastMtime = time.Unix(0, 0)
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
	lastMtime := sc.lastMtime.Unix()
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

	if sc.lastMtime.Unix() < lastMtime {
		sc.lastMtime = time.Unix(lastMtime, 0)
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
