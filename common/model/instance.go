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

package model

import (
	"strconv"
	"sync"
)

type ServiceInstances struct {
	lock               sync.RWMutex
	instances          map[string]*Instance
	healthyInstances   map[string]*Instance
	unhealthyInstances map[string]*Instance
	protectInstances   map[string]*Instance
	protectThreshold   float32
}

func NewServiceInstances(protectThreshold float32) *ServiceInstances {
	return &ServiceInstances{
		instances:          make(map[string]*Instance, 128),
		healthyInstances:   make(map[string]*Instance, 128),
		unhealthyInstances: make(map[string]*Instance, 128),
		protectInstances:   make(map[string]*Instance, 128),
	}
}

func (si *ServiceInstances) TotalCount() int {
	si.lock.RLock()
	defer si.lock.RUnlock()

	return len(si.instances)
}

func (si *ServiceInstances) UpdateProtectThreshold(protectThreshold float32) {
	si.lock.Lock()
	defer si.lock.Unlock()

	si.protectThreshold = protectThreshold
}

func (si *ServiceInstances) UpsertInstance(ins *Instance) {
	si.lock.Lock()
	defer si.lock.Unlock()

	si.instances[ins.ID()] = ins
	if ins.Healthy() {
		si.healthyInstances[ins.ID()] = ins
	} else {
		si.unhealthyInstances[ins.ID()] = ins
	}
}

func (si *ServiceInstances) RemoveInstance(ins *Instance) {
	si.lock.Lock()
	defer si.lock.Unlock()

	delete(si.instances, ins.ID())
	delete(si.healthyInstances, ins.ID())
	delete(si.unhealthyInstances, ins.ID())
	delete(si.protectInstances, ins.ID())
}

func (si *ServiceInstances) Range(iterator func(id string, ins *Instance)) {
	si.lock.RLock()
	defer si.lock.RUnlock()

	for k, v := range si.instances {
		iterator(k, v)
	}
}

func (si *ServiceInstances) GetInstances(onlyHealthy bool) []*Instance {
	si.lock.RLock()
	defer si.lock.RUnlock()

	ret := make([]*Instance, 0, len(si.healthyInstances)+len(si.protectInstances))
	if !onlyHealthy {
		for k, v := range si.instances {
			protectIns, ok := si.protectInstances[k]
			if ok {
				ret = append(ret, protectIns)
			} else {
				ret = append(ret, v)
			}
		}
	} else {
		for _, v := range si.healthyInstances {
			ret = append(ret, v)
		}
		for _, v := range si.protectInstances {
			ret = append(ret, v)
		}
	}
	return ret
}

func (si *ServiceInstances) ReachHealthyProtect() bool {
	si.lock.RLock()
	defer si.lock.RUnlock()

	return len(si.protectInstances) > 0
}

func (si *ServiceInstances) RunHealthyProtect() {
	si.lock.Lock()
	defer si.lock.Unlock()

	lastBeat := int64(-1)

	curProportion := float32(len(si.healthyInstances)) / float32(len(si.instances))
	if curProportion > si.protectThreshold {
		// 不会触发, 并且清空当前保护状态的实例
		si.protectInstances = make(map[string]*Instance, 128)
		return
	}
	instanceLastBeatTimes := map[string]int64{}
	instances := si.unhealthyInstances
	for i := range instances {
		ins := instances[i]
		metadata := ins.Metadata()
		if len(metadata) == 0 {
			continue
		}
		val, ok := metadata[MetadataInstanceLastHeartbeatTime]
		if !ok {
			continue
		}
		beatTime, _ := strconv.ParseInt(val, 10, 64)
		if beatTime >= lastBeat {
			lastBeat = beatTime
		}
		instanceLastBeatTimes[ins.ID()] = beatTime
	}
	if lastBeat == -1 {
		return
	}
	for i := range instances {
		ins := instances[i]
		beatTime, ok := instanceLastBeatTimes[ins.ID()]
		if !ok {
			continue
		}
		needProtect := needZeroProtect(lastBeat, beatTime, int64(ins.HealthCheck().GetHeartbeat().GetTtl().GetValue()))
		if !needProtect {
			continue
		}
		si.protectInstances[ins.ID()] = ins
	}
}

// needZeroProtect .
func needZeroProtect(lastBeat, beatTime, ttl int64) bool {
	return lastBeat-3*ttl > beatTime
}
