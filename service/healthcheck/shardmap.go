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
	"sync/atomic"
)

// A Concurrent safe ShardMap for healthCheckInstances
// To avoid lock bottlenecks this map is dived to several (ShardSize) Concurrent map.
type ShardMap struct {
	shardSize uint32
	Shards    []*Shard
	Len       int32
}

type Shard struct {
	healthCheckInstances map[string]*InstanceWithChecker
	healthCheckMutex     *sync.RWMutex
}

// Creates a new ShardMap
func NewShardMap(size uint32) *ShardMap {
	m := &ShardMap{
		shardSize: size,
		Shards:    make([]*Shard, size),
		Len:       0,
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
	return m.Shards[fnv32(instanceId)%m.shardSize]
}

// Store stores healthCheckInstances under given instanceId.
func (m *ShardMap) Store(instanceId string, healthCheckInstance *InstanceWithChecker) {
	if len(instanceId) == 0 {
		return
	}
	shard := m.getShard(instanceId)
	shard.healthCheckMutex.Lock()
	_, ok := shard.healthCheckInstances[instanceId]
	if ok {
		shard.healthCheckInstances[instanceId] = healthCheckInstance
	} else {
		shard.healthCheckInstances[instanceId] = healthCheckInstance
		atomic.AddInt32(&m.Len, 1)
	}
	shard.healthCheckMutex.Unlock()
}

//PutIfAbsent to avoid storing twice when key is the same in the concurrent scenario。
func (m *ShardMap) PutIfAbsent(instanceId string, healthCheckInstance *InstanceWithChecker) (*InstanceWithChecker, bool) {
	if len(instanceId) == 0 {
		return nil, false
	}
	shard := m.getShard(instanceId)
	shard.healthCheckMutex.Lock()
	value, has := shard.healthCheckInstances[instanceId]
	if !has {
		shard.healthCheckInstances[instanceId] = healthCheckInstance
		shard.healthCheckMutex.Unlock()
		atomic.AddInt32(&m.Len, 1)
		return healthCheckInstance, true
	}
	shard.healthCheckMutex.Unlock()
	return value, false
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
	_, ok := shard.healthCheckInstances[instanceId]
	if ok {
		delete(shard.healthCheckInstances, instanceId)
		atomic.AddInt32(&m.Len, -1)
	}
	shard.healthCheckMutex.Unlock()
}

//DeleteIfExist to avoid deleting twice when key is the same in the concurrent scenario。
func (m *ShardMap) DeleteIfExist(instanceId string) bool {
	if len(instanceId) == 0 {
		return false
	}
	shard := m.getShard(instanceId)
	shard.healthCheckMutex.Lock()
	_, ok := shard.healthCheckInstances[instanceId]
	if ok {
		delete(shard.healthCheckInstances, instanceId)
		atomic.AddInt32(&m.Len, -1)
		shard.healthCheckMutex.Unlock()
		return true
	}
	shard.healthCheckMutex.Unlock()
	return false
}

// Range iterates over the ShardMap.
func (m *ShardMap) Range(fn func(instanceId string, healthCheckInstance *InstanceWithChecker)) {
	for _, shard := range m.Shards {
		shard.healthCheckMutex.RLock()
		for k, v := range shard.healthCheckInstances {
			fn(k, v)
		}
		shard.healthCheckMutex.RUnlock()
	}
}

// Count returns the number of elements within the map.
func (m *ShardMap) Count() int32 {
	return m.Len
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
