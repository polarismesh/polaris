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

package sharedmap

import (
	"github.com/polarismesh/polaris-server/healthcheck"
	"sync"
)

type SharedMap struct {
	size   uint32
	Shared []*Shared
}

type Shared struct {
	healthCheckInstances map[string]*healthcheck.InstanceWithChecker
	healthCheckMutex     *sync.RWMutex
}

func (m *SharedMap) getShard(instanceId string) *Shared {
	return m.Shared[fnv32(instanceId)&(m.size-1)]
}

func newSharedMap(size uint32) *SharedMap {
	m := &SharedMap{
		size:   size,
		Shared: make([]*Shared, size),
	}
	for i := range m.Shared {
		m.Shared[i] = &Shared{
			healthCheckInstances: make(map[string]*healthcheck.InstanceWithChecker),
			healthCheckMutex:     &sync.RWMutex{},
		}
	}
	return m
}

func (m *SharedMap) store(instanceId string, healthCheckInstance *healthcheck.InstanceWithChecker) {
	if len(instanceId) == 0 {
		return
	}
	shard := m.getShard(instanceId)
	shard.healthCheckMutex.Lock()
	shard.healthCheckInstances[instanceId] = healthCheckInstance
	shard.healthCheckMutex.Unlock()
}

func (m *SharedMap) load(instanceId string) (healthCheckInstance *healthcheck.InstanceWithChecker, ok bool) {
	if len(instanceId) == 0 {
		return nil, false
	}
	shard := m.getShard(instanceId)
	shard.healthCheckMutex.RLock()
	healthCheckInstance, ok = shard.healthCheckInstances[instanceId]
	shard.healthCheckMutex.RUnlock()
	return healthCheckInstance, ok
}

func (m *SharedMap) delete(instanceId string) {
	if len(instanceId) == 0 {
		return
	}
	shard := m.getShard(instanceId)
	shard.healthCheckMutex.Lock()
	delete(shard.healthCheckInstances, instanceId)
	shard.healthCheckMutex.Unlock()
}

func (m *SharedMap) rangeMap(fn func(instanceId string, healthCheckInstance *healthcheck.InstanceWithChecker)) {
	for _, shard := range m.Shared {
		shard.healthCheckMutex.Lock()
		for k, v := range shard.healthCheckInstances {
			fn(k, v)
		}
		shard.healthCheckMutex.Unlock()
	}
}

// FNV hash
func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}
