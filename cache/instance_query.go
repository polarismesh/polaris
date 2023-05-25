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
	"strconv"

	"github.com/polarismesh/polaris/common/model"
)

/**
var (
	// InstanceFilterAttributes 查询实例支持的过滤字段
	InstanceFilterAttributes = map[string]bool{
		"id":            true, // 实例ID
		"service":       true, // 服务name
		"namespace":     true, // 服务namespace
		"host":          true,
		"port":          true,
		"keys":          true,
		"values":        true,
		"protocol":      true,
		"version":       true,
		"health_status": true,
		"healthy":       true, // health_status, healthy都有，以healthy为准
		"isolate":       true,
		"weight":        true,
		"logic_set":     true,
		"cmdb_region":   true,
		"cmdb_zone":     true,
		"cmdb_idc":      true,
		"priority":      true,
		"offset":        true,
		"limit":         true,
	}
	// InsFilter2toreAttr 查询字段转为存储层的属性值，映射表
	InsFilter2toreAttr = map[string]string{
		"service": "name",
		"healthy": "health_status",
	}
	// NotInsFilterAttr 不属于 instance 表属性的字段
	NotInsFilterAttr = map[string]bool{
		"keys":   true,
		"values": true,
	}
)
*/

// forceUpdate 更新配置
func (ic *instanceCache) forceUpdate() error {
	if err := ic.update(); err != nil {
		return err
	}
	return nil
}

func (ic *instanceCache) QueryInstances(filter, metaFilter map[string]string,
	offset, limit uint32) (uint32, []*model.Instance, error) {
	if err := ic.forceUpdate(); err != nil {
		return 0, nil, err
	}
	var (
		tempInstances = make([]*model.Instance, 0, 32)
	)

	var (
		svcName, hasSvc                                 = filter["service"]
		namespace, hasNamespace                         = filter["namespace"]
		id, hasId                                       = filter["id"]
		host, hasHost                                   = filter["host"]
		protocol, hasProtocol                           = filter["protocol"]
		version, hasVersion                             = filter["version"]
		region, hasRegion                               = filter["cmdb_region"]
		zone, hasZone                                   = filter["cmdb_zone"]
		campus, hasIdc                                  = filter["cmdb_idc"]
		port, weight                                    uint32
		healthStatus, isolate                           bool
		hasPort, hasWeight, hasHealthStatus, hasIsolate bool
	)

	if portStr, ok := filter["port"]; ok {
		if v, err := strconv.ParseUint(portStr, 10, 64); err == nil {
			port = uint32(v)
			hasPort = true
		}
	}
	if weightStr, ok := filter["weight"]; ok {
		if v, err := strconv.ParseUint(weightStr, 10, 64); err == nil {
			weight = uint32(v)
			hasWeight = true
		}
	}
	if isolateStr, ok := filter["isolate"]; ok {
		if v, err := strconv.ParseBool(isolateStr); err == nil {
			isolate = v
			hasIsolate = true
		}
	}
	if healthStatusStr, ok := filter["health_status"]; ok {
		if v, err := strconv.ParseBool(healthStatusStr); err == nil {
			healthStatus = v
			hasHealthStatus = true
		}
	}
	if healthyStr, ok := filter["healthy"]; ok {
		if v, err := strconv.ParseBool(healthyStr); err == nil {
			healthStatus = v
			hasHealthStatus = true
		}
	}

	svcCache := ic.cacheMgr.Service().(*serviceCache)
	_ = ic.IteratorInstances(func(key string, value *model.Instance) (bool, error) {
		svc := svcCache.GetOrLoadServiceByID(value.ServiceID)
		if svc == nil {
			return true, nil
		}
		if hasSvc && svc.Name != svcName {
			return true, nil
		}
		if hasNamespace && svc.Namespace != namespace {
			return true, nil
		}
		if hasId && value.Proto.GetId().GetValue() != id {
			return true, nil
		}
		if hasHost && value.Proto.GetHost().GetValue() != host {
			return true, nil
		}
		if hasPort && value.Proto.GetPort().GetValue() != port {
			return true, nil
		}
		if hasIsolate && value.Proto.GetIsolate().GetValue() != isolate {
			return true, nil
		}
		if hasHealthStatus && value.Proto.GetHealthy().GetValue() != healthStatus {
			return true, nil
		}
		if hasWeight && value.Proto.GetWeight().GetValue() != weight {
			return true, nil
		}
		if hasRegion && value.Proto.GetLocation().GetRegion().GetValue() != region {
			return true, nil
		}
		if hasZone && value.Proto.GetLocation().GetZone().GetValue() != zone {
			return true, nil
		}
		if hasIdc && value.Proto.GetLocation().GetCampus().GetValue() != campus {
			return true, nil
		}
		if hasProtocol && value.Proto.GetProtocol().GetValue() != protocol {
			return true, nil
		}
		if hasVersion && value.Proto.GetVersion().GetValue() != version {
			return true, nil
		}
		if len(metaFilter) > 0 {
			for k, v := range metaFilter {
				insV, ok := value.Proto.GetMetadata()[k]
				if !ok || insV != v {
					return true, nil
				}
			}
		}
		tempInstances = append(tempInstances, value)
		return true, nil
	})

	total, ret := ic.doPage(tempInstances, offset, limit)
	return total, ret, nil
}

func (ic *instanceCache) doPage(ins []*model.Instance, offset, limit uint32) (uint32, []*model.Instance) {
	total := uint32(len(ins))
	if offset > total {
		return total, []*model.Instance{}
	}
	if offset+limit > total {
		return total, ins[offset:]
	}

	sort.Slice(ins, func(i, j int) bool {
		return ins[i].ModifyTime.After(ins[j].ModifyTime)
	})

	return total, ins[offset : offset+limit]
}
