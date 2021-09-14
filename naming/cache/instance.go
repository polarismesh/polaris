/**
 * Tencent is pleased to support the open source community by making CL5 available.
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
	"go.uber.org/zap"
)

const (
	// InstanceName instance name
	InstanceName = "instance"
)

// InstanceIterProc instance iter proc func
type InstanceIterProc func(key string, value *model.Instance) (bool, error)

/**
 * InstanceCache 实例相关的缓存接口
 */
type InstanceCache interface {
	Cache
	GetInstance(instanceID string) *model.Instance
	// 根据服务名获取实例，先查找服务名对应的服务ID，再找实例列表
	GetInstancesByServiceID(serviceID string) []*model.Instance
	// 迭代
	IteratorInstances(iterProc InstanceIterProc) error
	// 根据服务ID进行迭代
	IteratorInstancesWithService(serviceID string, iterProc InstanceIterProc) error
	// 获取instance的个数
	GetInstancesCount() int
}

/**
 * @brief 实例缓存的类
 */
type instanceCache struct {
	storage         store.Store
	lastMtime       time.Time
	firstUpdate     bool
	ids             *sync.Map // id -> instance
	services        *sync.Map // service id -> [instances]
	revisionCh      chan *revisionNotify
	disableBusiness bool
	needMeta        bool
	systemServiceID []string
}

/**
 * @brief 自注册到缓存列表
 */
func init() {
	RegisterCache(InstanceName, CacheInstance)
}

// 新建一个instanceCache
func newInstanceCache(storage store.Store, ch chan *revisionNotify) *instanceCache {
	return &instanceCache{
		storage:    storage,
		revisionCh: ch,
	}
}

/**
 * @brief 初始化函数
 */
func (ic *instanceCache) initialize(opt map[string]interface{}) error {
	ic.ids = new(sync.Map)
	ic.services = new(sync.Map)
	ic.lastMtime = time.Unix(0, 0)
	ic.firstUpdate = true
	if opt == nil {
		return nil
	}
	ic.disableBusiness, _ = opt["disableBusiness"].(bool)
	ic.needMeta, _ = opt["needMeta"].(bool)
	// 只加载系统服务
	if ic.disableBusiness {
		services, err := ic.getSystemServices()
		if err != nil {
			return err
		}
		ic.systemServiceID = make([]string, 0, len(services))
		for _, service := range services {
			if service.IsAlias() {
				continue
			}
			ic.systemServiceID = append(ic.systemServiceID, service.ID)
		}
	}
	return nil
}

/**
 * @brief 更新缓存函数
 */
func (ic *instanceCache) update() error {
	// 拉取diff前的所有数据
	start := time.Now()
	instances, err := ic.storage.GetMoreInstances(ic.lastMtime.Add(DefaultTimeDiff),
		ic.firstUpdate, ic.needMeta, ic.systemServiceID)
	if err != nil {
		log.Errorf("[Cache][Instance] update get storage more err: %s", err.Error())
		return err
	}

	ic.firstUpdate = false
	update, del := ic.setInstances(instances)
	log.Info("[Cache][Instance] get more instances", zap.Int("update", update), zap.Int("delete", del),
		zap.Time("last", ic.lastMtime), zap.Duration("used", time.Now().Sub(start)))
	return nil
}

/**
 * @brief 清理内部缓存数据
 */
func (ic *instanceCache) clear() error {
	ic.ids = new(sync.Map)
	ic.services = new(sync.Map)
	ic.lastMtime = time.Unix(0, 0)
	return nil
}

/**
 * @brief 获取资源名称
 */
func (ic *instanceCache) name() string {
	return InstanceName
}

/**
 * @brief 获取系统服务ID
 */
func (ic *instanceCache) getSystemServices() ([]*model.Service, error) {
	services, err := ic.storage.GetSystemServices()
	if err != nil {
		log.Errorf("[Cache][Instance] get system services err: %s", err.Error())
		return nil, err
	}
	return services, nil
}

// 保存instance到内存中
// 返回：更新个数，删除个数
func (ic *instanceCache) setInstances(ins map[string]*model.Instance) (int, int) {
	if len(ins) == 0 {
		return 0, 0
	}

	lastMtime := ic.lastMtime.Unix()
	update := 0
	del := 0
	affect := make(map[string]bool)
	progress := 0
	for _, item := range ins {
		progress++
		if progress%50000 == 0 {
			log.Infof("[Cache][Instance] set instances progress: %d / %d", progress, len(ins))
		}
		modifyTime := item.ModifyTime.Unix()
		if lastMtime < modifyTime {
			lastMtime = modifyTime
		}
		affect[item.ServiceID] = true

		// 待删除的instance
		if !item.Valid {
			del++
			ic.ids.Delete(item.ID())
			value, ok := ic.services.Load(item.ServiceID)
			if !ok {
				continue
			}

			value.(*sync.Map).Delete(item.ID())
			continue
		}

		// 有修改或者新增的数据
		// 缓存的instance map增加一个version和protocol字段
		update++
		if item.Proto.Metadata == nil {
			item.Proto.Metadata = make(map[string]string)
		}
		item.Proto.Metadata["version"] = item.Version()
		item.Proto.Metadata["protocol"] = item.Protocol()
		ic.ids.Store(item.ID(), item)
		value, ok := ic.services.Load(item.ServiceID)
		if !ok {
			value = new(sync.Map)
			ic.services.Store(item.ServiceID, value)
		}
		value.(*sync.Map).Store(item.ID(), item)
	}

	if ic.lastMtime.Unix() < lastMtime {
		ic.lastMtime = time.Unix(lastMtime, 0)
	}

	progress = 0
	for serviceID := range affect {
		ic.revisionCh <- newRevisionNotify(serviceID, true)
		progress++
		if progress%10000 == 0 {
			log.Infof("[Cache][Instance] revision notify progress(%d / %d)", progress, len(affect))
		}
	}

	return update, del
}

/**
 * GetInstance 根据实例ID获取实例数据
 */
func (ic *instanceCache) GetInstance(instanceID string) *model.Instance {
	if instanceID == "" {
		return nil
	}

	value, ok := ic.ids.Load(instanceID)
	if !ok {
		return nil
	}

	return value.(*model.Instance)
}

/**
 * GetInstancesByServiceID 根据ServiceID获取实例数据
 */
func (ic *instanceCache) GetInstancesByServiceID(serviceID string) []*model.Instance {
	if serviceID == "" {
		return nil
	}

	value, ok := ic.services.Load(serviceID)
	if !ok {
		return nil
	}

	var out []*model.Instance
	value.(*sync.Map).Range(func(k interface{}, v interface{}) bool {
		out = append(out, v.(*model.Instance))
		return true
	})

	return out
}

/**
 * IteratorInstances 迭代所有的instance的函数
 */
func (ic *instanceCache) IteratorInstances(iterProc InstanceIterProc) error {
	return iteratorInstancesProc(ic.ids, iterProc)
}

// IteratorInstancesWithService 根据服务ID进行迭代回调
func (ic *instanceCache) IteratorInstancesWithService(serviceID string, iterProc InstanceIterProc) error {
	if serviceID == "" {
		return nil
	}
	value, ok := ic.services.Load(serviceID)
	if !ok {
		return nil
	}

	return iteratorInstancesProc(value.(*sync.Map), iterProc)
}

// GetInstancesCount 获取实例的个数
func (ic *instanceCache) GetInstancesCount() int {
	count := 0
	ic.ids.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	return count
}

// 迭代指定的instance数据，id->instance
func iteratorInstancesProc(data *sync.Map, iterProc InstanceIterProc) error {
	var cont bool
	var err error
	proc := func(k, v interface{}) bool {
		cont, err = iterProc(k.(string), v.(*model.Instance))
		if err != nil {
			return false
		}
		return cont
	}

	data.Range(proc)
	return err
}
