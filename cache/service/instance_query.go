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
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// InstanceSearchArgs .
type InstanceSearchArgs struct {
	SvcName      *string
	SvcNs        *string
	InstanceID   *string
	Hosts        map[string]struct{}
	Port         *uint32
	Protocol     *string
	Version      *string
	Region       *string
	Zone         *string
	Campus       *string
	Weight       *uint32
	HealthStatus *bool
	Isolate      *bool
	MetaFilter   map[string]string
}

func (args *InstanceSearchArgs) String() string {
	//nolint: errchkjson
	data, _ := json.Marshal(args)
	return string(data)
}

func parseInstanceSearchArgs(filter, metaFilter map[string]string) *InstanceSearchArgs {
	args := &InstanceSearchArgs{
		MetaFilter: metaFilter,
	}

	if searchSvcName, hasSvc := filter["name"]; hasSvc {
		args.SvcName = &searchSvcName
	}
	if searchNamespace, hasNamespace := filter["namespace"]; hasNamespace {
		args.SvcNs = &searchNamespace
	}
	if id, hasId := filter["id"]; hasId {
		args.InstanceID = &id
	}
	if protocol, hasProtocol := filter["protocol"]; hasProtocol {
		args.Protocol = &protocol
	}
	if version, hasVersion := filter["version"]; hasVersion {
		args.Version = &version
	}
	if region, hasRegion := filter["cmdb_region"]; hasRegion {
		args.Region = &region
	}
	if campus, hasIdc := filter["cmdb_idc"]; hasIdc {
		args.Campus = &campus
	}
	if zone, hasZone := filter["cmdb_zone"]; hasZone {
		args.Zone = &zone
	}

	if hosts, hasHosts := filter["host"]; hasHosts {
		hostMap := map[string]struct{}{}
		hostItems := strings.Split(hosts, ",")
		for i := range hostItems {
			hostVal := strings.TrimSpace(hostItems[i])
			if len(hostVal) == 0 {
				continue
			}
			hostMap[hostVal] = struct{}{}
		}
		args.Hosts = hostMap
	}

	if portStr, ok := filter["port"]; ok {
		if v, err := strconv.ParseUint(portStr, 10, 64); err == nil {
			port := uint32(v)
			args.Port = &port
		}
	}
	if weightStr, ok := filter["weight"]; ok {
		if v, err := strconv.ParseUint(weightStr, 10, 64); err == nil {
			weight := uint32(v)
			args.Weight = &weight
		}
	}
	if isolateStr, ok := filter["isolate"]; ok {
		if v, err := strconv.ParseBool(isolateStr); err == nil {
			isolate := v
			args.Isolate = &isolate
		}
	}
	if healthStatusStr, ok := filter["health_status"]; ok {
		if v, err := strconv.ParseBool(healthStatusStr); err == nil {
			healthStatus := v
			args.HealthStatus = &healthStatus
		}
	}
	if healthyStr, ok := filter["healthy"]; ok {
		if v, err := strconv.ParseBool(healthyStr); err == nil {
			healthStatus := v
			args.HealthStatus = &healthStatus
		}
	}
	return args
}

// forceQueryUpdate 为了确保读取的数据是最新的，这里需要做一个强制 update 的动作进行数据读取处理
func (ic *instanceCache) forceQueryUpdate() error {
	err, shared := ic.singleUpdate()
	// shared == true，表示当前已经有正在 update 执行的任务，这个任务不一定能够读取到最新的数据
	// 为了避免读取到脏数据，在发起一次 singleUpdate
	if shared {
		naminglog.Debug("[Server][Instances][Query] force query update from store")
		err, _ = ic.singleUpdate()
	}
	return err
}

func (ic *instanceCache) QueryInstances(filter, metaFilter map[string]string,
	offset, limit uint32) (uint32, []*model.Instance, error) {
	if err := ic.forceQueryUpdate(); err != nil {
		return 0, nil, err
	}
	var (
		tempInstances = make([]*model.Instance, 0, 32)
		args          = parseInstanceSearchArgs(filter, metaFilter)
	)
	naminglog.Info("[Server][Instances][Query] instances filter parameters", zap.String("args", args.String()))

	svcCache, _ := ic.BaseCache.CacheMgr.GetCacher(types.CacheService).(*serviceCache)
	_ = ic.IteratorInstances(func(key string, value *model.Instance) (bool, error) {
		svc := svcCache.GetOrLoadServiceByID(value.ServiceID)
		if svc == nil {
			return true, nil
		}
		if args.SvcName != nil && !utils.IsWildMatch(svc.Name, *args.SvcName) {
			return true, nil
		}
		if args.SvcNs != nil && !utils.IsWildMatch(svc.Namespace, *args.SvcNs) {
			return true, nil
		}
		if args.InstanceID != nil && !utils.IsWildMatch(value.Proto.GetId().GetValue(), *args.InstanceID) {
			return true, nil
		}
		if len(args.Hosts) != 0 {
			if _, ok := args.Hosts[value.Proto.GetHost().GetValue()]; !ok {
				return true, nil
			}
		}
		if args.Port != nil && value.Proto.GetPort().GetValue() != *args.Port {
			return true, nil
		}
		if args.Isolate != nil && value.Proto.GetIsolate().GetValue() != *args.Isolate {
			return true, nil
		}
		if args.HealthStatus != nil && value.Proto.GetHealthy().GetValue() != *args.HealthStatus {
			return true, nil
		}
		if args.Weight != nil && value.Proto.GetWeight().GetValue() != *args.Weight {
			return true, nil
		}
		if args.Region != nil && value.Proto.GetLocation().GetRegion().GetValue() != *args.Region {
			return true, nil
		}
		if args.Zone != nil && value.Proto.GetLocation().GetZone().GetValue() != *args.Zone {
			return true, nil
		}
		if args.Campus != nil && value.Proto.GetLocation().GetCampus().GetValue() != *args.Campus {
			return true, nil
		}
		if args.Protocol != nil && value.Proto.GetProtocol().GetValue() != *args.Protocol {
			return true, nil
		}
		if args.Version != nil && value.Proto.GetVersion().GetValue() != *args.Version {
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

	sortInstances(tempInstances)

	total, ret := ic.doPage(tempInstances, offset, limit)
	return total, ret, nil
}

func sortInstances(tempInstances []*model.Instance) {
	sort.Slice(tempInstances, func(i, j int) bool {
		aTime := tempInstances[i].ModifyTime
		bTime := tempInstances[j].ModifyTime
		if aTime.After(bTime) {
			return true
		}
		if aTime.Before(bTime) {
			return false
		}
		// 按照实例 ID 进行排序，确保排序结果的稳定性
		return strings.Compare(tempInstances[i].ID(), tempInstances[j].ID()) == 1
	})
}

func (ic *instanceCache) doPage(ins []*model.Instance, offset, limit uint32) (uint32, []*model.Instance) {
	total := uint32(len(ins))
	if offset > total {
		return total, []*model.Instance{}
	}
	if offset+limit > total {
		return total, ins[offset:]
	}
	return total, ins[offset : offset+limit]
}
