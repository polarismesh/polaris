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
	"container/list"
	"sync"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
)

const (
	// L5Name l5 name
	L5Name = "l5"
)

// L5Cache L5的cache接口
type L5Cache interface {
	Cache

	// GetRouteByIP 根据IP获取访问关系
	GetRouteByIP(ip uint32) []*model.Route

	// CheckRouteExisted 检查IP对应的SID是否存在访问关系
	CheckRouteExisted(ip uint32, modID uint32, cmdID uint32) bool

	// GetPolicy 获取有状态路由信息policy
	GetPolicy(modID uint32) *model.Policy

	// GetSection 获取有状态路由信息policy
	GetSection(modeID uint32) []*model.Section

	// GetIPConfig 获取IpConfig
	GetIPConfig(ip uint32) *model.IPConfig
}

// l5Cache L5的cache对象
type l5Cache struct {
	*baseCache

	storage store.Store

	lastRouteFlow    uint32
	lastPolicyFlow   uint32
	lastSectionFlow  uint32
	lastIPConfigFlow uint32

	routeList    *sync.Map // <IP, <sidStr, setID> >
	policyList   *sync.Map // <modID, Policy>
	sectionList  *sync.Map // <modID, []*Section (list)>
	ipConfigList *sync.Map // <IP, IPConfig>

	// instances的信息
	ic *instanceCache
	sc *serviceCache
}

func newL5Cache(s store.Store) *l5Cache {
	return &l5Cache{
		baseCache: newBaseCache(),
		storage:   s,
	}
}

// init 自注册到缓存列表
func init() {
	RegisterCache(L5Name, CacheCL5)
}

// initialize 初始化函数
func (lc *l5Cache) initialize(opt map[string]interface{}) error {
	lc.lastRouteFlow = 0
	lc.lastPolicyFlow = 0
	lc.lastSectionFlow = 0
	lc.lastIPConfigFlow = 0
	lc.routeList = new(sync.Map)
	lc.policyList = new(sync.Map)
	lc.sectionList = new(sync.Map)
	lc.ipConfigList = new(sync.Map)
	if opt == nil {
		return nil
	}
	return nil
}

func (lc *l5Cache) update(storeRollbackSec time.Duration) error {
	err := lc.updateCL5Route()
	if err != nil {
		log.CacheScope().Errorf("[Cache][CL5] update l5 route cache err: %s", err.Error())
	}
	err = lc.updateCL5Policy()
	if err != nil {
		log.CacheScope().Errorf("[Cache][CL5] update l5 policy cache err: %s", err.Error())
	}
	err = lc.updateCL5Section()
	if err != nil {
		log.CacheScope().Errorf("[Cache][CL5] update l5 section cache err: %s", err.Error())
	}
	return err
}

// clear 清理内部缓存数据
func (lc *l5Cache) clear() error {
	lc.routeList = new(sync.Map)
	lc.policyList = new(sync.Map)
	lc.sectionList = new(sync.Map)
	lc.ipConfigList = new(sync.Map)
	lc.lastRouteFlow = 0
	lc.lastPolicyFlow = 0
	lc.lastSectionFlow = 0
	lc.lastIPConfigFlow = 0
	return nil
}

// name 获取资源名称
func (lc *l5Cache) name() string {
	return L5Name
}

// GetRouteByIP 根据Ip获取访问关系
func (lc *l5Cache) GetRouteByIP(ip uint32) []*model.Route {
	out := make([]*model.Route, 0)
	value, ok := lc.routeList.Load(ip)
	if !ok {
		// 该ip不存在访问关系，则返回一个空数组
		return out
	}

	entry := value.(*sync.Map)
	entry.Range(func(key, value interface{}) bool {
		// sidStr -> setID
		sid, err := utils.UnmarshalSid(key.(string))
		if err != nil {
			return true
		}

		item := &model.Route{
			IP:    ip,
			ModID: sid.ModID,
			CmdID: sid.CmdID,
			SetID: value.(string),
		}
		out = append(out, item)
		return true
	})

	return out
}

// CheckRouteExisted 检查访问关系是否存在
func (lc *l5Cache) CheckRouteExisted(ip uint32, modID uint32, cmdID uint32) bool {
	value, ok := lc.routeList.Load(ip)
	if !ok {
		return false
	}

	entry := value.(*sync.Map)
	found := false
	entry.Range(func(key, value interface{}) bool {
		sid, err := utils.UnmarshalSid(key.(string))
		if err != nil {
			// continue range
			return true
		}

		if modID == sid.ModID && cmdID == sid.CmdID {
			found = true
			// break range
			return false
		}
		return true
	})

	return found
}

// GetPolicy 根据modID获取policy信息
func (lc *l5Cache) GetPolicy(modID uint32) *model.Policy {
	value, ok := lc.policyList.Load(modID)
	if !ok {
		return nil
	}

	return value.(*model.Policy)
}

// GetSection 根据modID获取section信息
func (lc *l5Cache) GetSection(modeID uint32) []*model.Section {
	value, ok := lc.sectionList.Load(modeID)
	if !ok {
		return nil
	}

	obj := value.(*list.List)
	out := make([]*model.Section, 0, obj.Len())
	for e := obj.Front(); e != nil; e = e.Next() {
		out = append(out, e.Value.(*model.Section))
	}

	return out
}

// GetIPConfig 根据IP获取ipConfig
func (lc *l5Cache) GetIPConfig(ip uint32) *model.IPConfig {
	value, ok := lc.ipConfigList.Load(ip)
	if !ok {
		return nil
	}

	return value.(*model.IPConfig)
}

// updateCL5Route 更新l5的route缓存数据
func (lc *l5Cache) updateCL5Route() error {
	routes, err := lc.storage.GetMoreL5Routes(lc.lastRouteFlow)
	if err != nil {
		log.CacheScope().Errorf("[Cache][CL5] get l5 route from storage err: %s", err.Error())
		return err
	}

	return lc.setCL5Route(routes)
}

// updateCL5Policy更新l5的policy缓存数据
func (lc *l5Cache) updateCL5Policy() error {
	policies, err := lc.storage.GetMoreL5Policies(lc.lastPolicyFlow)
	if err != nil {
		log.CacheScope().Errorf("[Cache][CL5] get l5 policy from storage err: %s", err.Error())
		return err
	}

	return lc.setCL5Policy(policies)
}

// updateCL5Section 更新l5的section缓存数据
func (lc *l5Cache) updateCL5Section() error {
	sections, err := lc.storage.GetMoreL5Sections(lc.lastSectionFlow)
	if err != nil {
		log.CacheScope().Errorf("[Cache][CL5] get l5 section from storage err: %s", err.Error())
		return err
	}

	return lc.setCL5Section(sections)
}

// updateCL5IPConfig 更新l5的ip config缓存数据
func (lc *l5Cache) updateCL5IPConfig() error {
	ipConfigs, err := lc.storage.GetMoreL5IPConfigs(lc.lastIPConfigFlow)
	if err != nil {
		log.CacheScope().Errorf("[Cache][CL5] get l5 ip config from storage err: %s", err.Error())
		return err
	}

	return lc.setCL5IPConfig(ipConfigs)
}

// setCL5Route 更新l5 route的本地缓存
func (lc *l5Cache) setCL5Route(routes []*model.Route) error {
	if len(routes) == 0 {
		return nil
	}

	lastRouteFlow := lc.lastRouteFlow
	for _, item := range routes {
		if item.Flow > lastRouteFlow {
			lastRouteFlow = item.Flow
		}

		sidStr := utils.MarshalModCmd(item.ModID, item.CmdID)

		// 待删除的route
		if !item.Valid {
			value, ok := lc.routeList.Load(item.IP)
			if !ok {
				continue
			}

			value.(*sync.Map).Delete(sidStr)
			continue
		}

		value, ok := lc.routeList.Load(item.IP)
		if !ok {
			value = new(sync.Map)
			lc.routeList.Store(item.IP, value)
		}
		value.(*sync.Map).Store(sidStr, item.SetID)
	}

	if lc.lastRouteFlow < lastRouteFlow {
		lc.lastRouteFlow = lastRouteFlow
	}

	return nil
}

// setCL5Policy 更新l5 policy的本地缓存
func (lc *l5Cache) setCL5Policy(policies []*model.Policy) error {
	if len(policies) == 0 {
		return nil
	}

	lastPolicyFlow := lc.lastPolicyFlow
	for _, item := range policies {
		if item.Flow > lastPolicyFlow {
			lastPolicyFlow = item.Flow
		}

		// 待删除的policy
		if !item.Valid {
			lc.policyList.Delete(item.ModID)
			continue
		}

		lc.policyList.Store(item.ModID, item)
	}

	if lc.lastPolicyFlow < lastPolicyFlow {
		lc.lastPolicyFlow = lastPolicyFlow
	}

	return nil
}

// setCL5Section 更新l5 section的本地缓存
func (lc *l5Cache) setCL5Section(sections []*model.Section) error {
	if len(sections) == 0 {
		return nil
	}

	lastSectionFlow := lc.lastSectionFlow
	for _, item := range sections {
		if item.Flow > lastSectionFlow {
			lastSectionFlow = item.Flow
		}

		// 无论数据是否要删除，都执行删除老数据操作
		var listObj *list.List
		if value, ok := lc.sectionList.Load(item.ModID); ok {
			listObj = value.(*list.List)
		} else {
			listObj = list.New()
		}

		for ele := listObj.Front(); ele != nil; ele = ele.Next() {
			entry := ele.Value.(*model.Section)
			if entry.From == item.From && entry.To == item.To {
				listObj.Remove(ele)
				break
			}
		}
		// 上面已经删除了，这里直接继续迭代
		if !item.Valid {
			continue
		}

		// 存储有效的数据
		listObj.PushBack(item)
		lc.sectionList.Store(item.ModID, listObj)
	}

	if lc.lastSectionFlow < lastSectionFlow {
		lc.lastSectionFlow = lastSectionFlow
	}

	return nil
}

// setCL5IPConfig 更新l5 ipConfig的本地缓存
func (lc *l5Cache) setCL5IPConfig(ipConfigs []*model.IPConfig) error {
	if len(ipConfigs) == 0 {
		return nil
	}

	lastIPConfigFlow := lc.lastIPConfigFlow
	for _, item := range ipConfigs {
		if item.Flow > lastIPConfigFlow {
			lastIPConfigFlow = item.Flow
		}

		// 待删除的ip config
		if !item.Valid {
			lc.ipConfigList.Delete(item.IP)
			continue
		}

		lc.ipConfigList.Store(item.IP, item)
	}

	if lc.lastIPConfigFlow < lastIPConfigFlow {
		lc.lastIPConfigFlow = lastIPConfigFlow
	}

	return nil
}
