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
	"strings"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

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
		hosts, hasHost                                  = filter["host"]
		protocol, hasProtocol                           = filter["protocol"]
		version, hasVersion                             = filter["version"]
		region, hasRegion                               = filter["cmdb_region"]
		zone, hasZone                                   = filter["cmdb_zone"]
		campus, hasIdc                                  = filter["cmdb_idc"]
		port, weight                                    uint32
		healthStatus, isolate                           bool
		hasPort, hasWeight, hasHealthStatus, hasIsolate bool
	)

	hostMap := map[string]struct{}{}
	hostItems := strings.Split(hosts, ",")
	for i := range hostItems {
		hostMap[strings.TrimSpace(hostItems[i])] = struct{}{}
	}

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
		if hasSvc && !utils.IsWildMatch(svc.Name, svcName) {
			return true, nil
		}
		if hasNamespace && !utils.IsWildMatch(svc.Namespace, namespace) {
			return true, nil
		}
		if hasId && !utils.IsWildMatch(value.Proto.GetId().GetValue(), id) {
			return true, nil
		}
		if hasHost {
			if _, ok := hostMap[value.Proto.GetHost().GetValue()]; !ok {
				return true, nil
			}
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
