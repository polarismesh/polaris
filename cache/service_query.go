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
	"sort"
	"strings"
	"sync"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
)

// ServiceArgs 服务查询条件
type ServiceArgs struct {
	// Filter 普通服务字段条件
	Filter map[string]string
	// Metadata 元数据条件
	Metadata map[string]string
	// SvcIds 是否按照服务的ID进行等值查询
	SvcIds map[string]struct{}
	// FuzzyName 是否进行名字的模糊匹配
	FuzzyName bool
	// FuzzyBusiness 是否进行业务的模糊匹配
	FuzzyBusiness bool
	// Namespace 条件中的命名空间
	Namespace string
	// Name 条件中的服务名
	Name string
	// EmptyCondition 是否是空条件，即只需要从所有服务或者某个命名空间下面的服务，进行不需要匹配的遍历，返回前面的服务即可
	EmptyCondition bool
}

// Update 更新配置
func (sc *serviceCache) Update() error {
	var err error
	if err = sc.update(0); err != nil {
		return err
	}
	if err = sc.instCache.update(0); err != nil {
		return err
	}
	return nil
}

// GetServicesByFilter 通过filter在缓存中进行服务过滤
func (sc *serviceCache) GetServicesByFilter(serviceFilters *ServiceArgs,
	instanceFilters *store.InstanceArgs, offset, limit uint32) (uint32, []*model.EnhancedService, error) {
	var amount uint32
	var err error
	var services []*model.Service
	// 如果具有名字条件，并且不是模糊查询，直接获取对应命名空间下面的服务，并检查是否匹配所有条件
	if serviceFilters.Name != "" && !serviceFilters.FuzzyName {
		amount, services, err = sc.getServicesFromCacheByName(serviceFilters, instanceFilters, offset, limit)
	} else {
		amount, services, err = sc.getServicesByIteratingCache(serviceFilters, instanceFilters, offset, limit)
	}
	var enhancedServices []*model.EnhancedService
	if amount > 0 {
		enhancedServices = make([]*model.EnhancedService, 0, len(services))
		for _, service := range services {
			count := sc.instCache.GetInstancesCountByServiceID(service.ID)
			enhancedService := &model.EnhancedService{
				Service:              service,
				TotalInstanceCount:   count.TotalInstanceCount,
				HealthyInstanceCount: count.HealthyInstanceCount,
			}
			enhancedServices = append(enhancedServices, enhancedService)
		}
	}
	return amount, enhancedServices, err
}

func hasInstanceFilter(instanceFilters *store.InstanceArgs) bool {
	if instanceFilters == nil || (len(instanceFilters.Hosts) == 0 && len(instanceFilters.Ports) == 0) {
		return false
	}
	return true
}

func (sc *serviceCache) matchInstances(instances []*model.Instance, instanceFilters *store.InstanceArgs) bool {
	if len(instances) == 0 {
		return false
	}
	var matchedHost bool
	if len(instanceFilters.Hosts) > 0 {
		var hosts = make(map[string]bool, len(instanceFilters.Hosts))
		for _, host := range instanceFilters.Hosts {
			hosts[host] = true
		}
		for _, instance := range instances {
			if _, ok := hosts[instance.Proto.GetHost().GetValue()]; ok {
				matchedHost = true
				break
			}
		}
	} else {
		matchedHost = true
	}

	var matchedPort bool
	if len(instanceFilters.Ports) > 0 {
		var ports = make(map[uint32]bool, len(instanceFilters.Ports))
		for _, port := range instanceFilters.Ports {
			ports[port] = true
		}
		for _, instance := range instances {
			if _, ok := ports[instance.Proto.GetPort().GetValue()]; ok {
				matchedPort = true
				break
			}
		}
	} else {
		matchedPort = true
	}
	return matchedHost && matchedPort
}

// GetAllNamespaces 返回所有的命名空间
func (sc *serviceCache) GetAllNamespaces() []string {
	var res []string
	sc.names.Range(func(k, v interface{}) bool {
		res = append(res, k.(string))
		return true
	})
	return res
}

// 通过具体的名字来进行查询服务
func (sc *serviceCache) getServicesFromCacheByName(svcArgs *ServiceArgs, instArgs *store.InstanceArgs,
	offset, limit uint32) (uint32, []*model.Service, error) {
	var res []*model.Service
	if svcArgs.Namespace != "" {
		svc := sc.GetServiceByName(svcArgs.Name, svcArgs.Namespace)
		if svc != nil && !svc.IsAlias() && matchService(svc, svcArgs.Filter, svcArgs.Metadata, false) &&
			sc.matchInstance(svc, instArgs) {
			res = append(res, svc)
		}
	} else {
		for _, namespace := range sc.GetAllNamespaces() {
			svc := sc.GetServiceByName(svcArgs.Name, namespace)
			if svc != nil && !svc.IsAlias() && matchService(svc, svcArgs.Filter, svcArgs.Metadata, false) &&
				sc.matchInstance(svc, instArgs) {
				res = append(res, svc)
			}
		}
	}
	amount, services := sortBeforeTrim(res, offset, limit)
	return amount, services, nil
}

func sortBeforeTrim(services []*model.Service, offset, limit uint32) (uint32, []*model.Service) {
	// 所有符合条件的服务数量
	amount := uint32(len(services))
	// 判断 offset 和 limit 是否允许返回对应的服务
	if offset >= amount || limit == 0 {
		return amount, nil
	}
	// 将服务按照修改时间和 id 进行排序
	sort.Slice(services, func(i, j int) bool {
		if services[i].Mtime > services[j].Mtime {
			return true
		} else if services[i].Mtime < services[j].Mtime {
			return false
		} else {
			return strings.Compare(services[i].ID, services[j].ID) < 0
		}
	})
	endIdx := offset + limit
	if endIdx > amount {
		endIdx = amount
	}
	return amount, services[offset:endIdx]
}

// matchService 根据查询条件比较一个服务是否符合条件
func matchService(svc *model.Service, svcFilter map[string]string, metaFilter map[string]string, matchName bool) bool {
	if !matchServiceFilter(svc, svcFilter, matchName) {
		return false
	}
	return matchMetadata(svc, metaFilter)
}

// matchServiceFilter 查询一个服务是否满足服务相关字段的条件
func matchServiceFilter(svc *model.Service, svcFilter map[string]string, matchName bool) bool {
	var value string
	var exist bool
	if matchName {
		// 走到这一步，一定是模糊匹配
		if value, exist = svcFilter["name"]; exist {
			searchVal := value[0 : len(value)-1]
			if !strings.Contains(strings.ToLower(svc.Name), strings.ToLower(searchVal)) {
				return false
			}
		}
	}
	if value, exist = svcFilter["business"]; exist &&
		!strings.Contains(strings.ToLower(svc.Business), strings.ToLower(value)) {
		return false
	}
	if value, exist = svcFilter["department"]; exist && svc.Department != value {
		return false
	}
	if value, exist = svcFilter["cmdb_mod1"]; exist && svc.CmdbMod1 != value {
		return false
	}
	if value, exist = svcFilter["cmdb_mod2"]; exist && svc.CmdbMod2 != value {
		return false
	}
	if value, exist = svcFilter["cmdb_mod3"]; exist && svc.CmdbMod3 != value {
		return false
	}
	if value, exist = svcFilter["platform_id"]; exist && svc.PlatformID != value {
		return false
	}
	if value, exist = svcFilter["owner"]; exist && !strings.Contains(svc.Owner, value) {
		return false
	}
	return true
}

// matchMetadata 检查一个服务是否包含有相关的元数据
func matchMetadata(svc *model.Service, metaFilter map[string]string) bool {
	for k, v := range metaFilter {
		value, ok := svc.Meta[k]
		if !ok || value != v {
			return false
		}
	}
	return true
}

func (sc *serviceCache) matchInstance(svc *model.Service, instArgs *store.InstanceArgs) bool {
	if hasInstanceFilter(instArgs) {
		instances := sc.instCache.GetInstancesByServiceID(svc.ID)
		if !sc.matchInstances(instances, instArgs) {
			return false
		}
	}
	return true
}

// getServicesByIteratingCache 通过遍历缓存中的服务
func (sc *serviceCache) getServicesByIteratingCache(
	svcArgs *ServiceArgs, instArgs *store.InstanceArgs, offset, limit uint32) (uint32, []*model.Service, error) {
	var res []*model.Service
	var process = func(svc *model.Service) {
		// 如果是别名，直接略过
		if svc.IsAlias() {
			return
		}
		if !svcArgs.EmptyCondition {
			if !matchService(svc, svcArgs.Filter, svcArgs.Metadata, true) {
				return
			}
		}
		if !sc.matchInstance(svc, instArgs) {
			return
		}
		res = append(res, svc)
	}
	if len(svcArgs.Namespace) > 0 {
		// 从命名空间来找
		spaces, ok := sc.names.Load(svcArgs.Namespace)
		if !ok {
			return 0, nil, nil
		}
		spaces.(*sync.Map).Range(func(key, value interface{}) bool {
			process(value.(*model.Service))
			return true
		})
	} else {
		// 直接名字匹配
		_ = sc.IteratorServices(func(key string, svc *model.Service) (bool, error) {
			process(svc)
			return true, nil
		})
	}
	amount, services := sortBeforeTrim(res, offset, limit)
	return amount, services, nil
}
