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

package healthcheck

import (
	"sync"
)

// A Concurrent safe ShardMap for healthCheckInstances
// To avoid lock bottlenecks this map is dived to several (ShardSize) Concurrent map.
type ShardMap struct {
	size   uint32
	Shards []*Shard
}

type Shard struct {
	healthCheckInstances map[string]*InstanceWithChecker
	healthCheckMutex     *sync.RWMutex
}

// Creates a new ShardMap
func NewShardMap(size uint32) *ShardMap {
	m := &ShardMap{
		size:   size,
		Shards: make([]*Shard, size),
	}
	for i := range m.Shards {
		m.Shards[i] = &Shard{
			healthCheckInstances: make(map[string]*InstanceWithChecker),
			healthCheckMutex:     &sync.RWMutex{},
		}
	}
	return m
}

// getShard returns shard under given instanceId
func (m *ShardMap) getShard(instanceId string) *Shard {
	return m.Shards[fnv32(instanceId)%m.size]
}

// Store stores healthCheckInstances under given instanceId.
func (m *ShardMap) Store(instanceId string, healthCheckInstance *InstanceWithChecker) {
	if len(instanceId) == 0 {
		return
	}
	shard := m.getShard(instanceId)
	shard.healthCheckMutex.Lock()
	shard.healthCheckInstances[instanceId] = healthCheckInstance
	shard.healthCheckMutex.Unlock()
}

// Load loads the healthCheckInstances under the instanceId.
func (m *ShardMap) Load(instanceId string) (healthCheckInstance *InstanceWithChecker, ok bool) {
	if len(instanceId) == 0 {
		return nil, false
	}
	shard := m.getShard(instanceId)
	shard.healthCheckMutex.RLock()
	healthCheckInstance, ok = shard.healthCheckInstances[instanceId]
	shard.healthCheckMutex.RUnlock()
	return healthCheckInstance, ok
}

// Delete deletes the healthCheckInstances under the given instanceId.
func (m *ShardMap) Delete(instanceId string) {
	if len(instanceId) == 0 {
		return
	}
	shard := m.getShard(instanceId)
	shard.healthCheckMutex.Lock()
	delete(shard.healthCheckInstances, instanceId)
	shard.healthCheckMutex.Unlock()
}

// RangeMap iterates over the ShardMap.
func (m *ShardMap) RangeMap(fn func(instanceId string, healthCheckInstance *InstanceWithChecker)) {
	for _, shard := range m.Shards {
		shard.healthCheckMutex.Lock()
		for k, v := range shard.healthCheckInstances {
			fn(k, v)
		}
		shard.healthCheckMutex.Unlock()
	}
}

// Count returns the number of elements within the map.
func (m *ShardMap) Count() int {
	count := 0
	for i := 0; i < int(m.size); i++ {
		shard := m.Shards[i]
		shard.healthCheckMutex.RLock()
		count += len(shard.healthCheckInstances)
		shard.healthCheckMutex.RUnlock()
	}
	return count
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
