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

// A concurrent safe shardMap for values
// To avoid lock bottlenecks this map is dived to several (shardSize) concurrent map.
type shardMap struct {
	shardSize uint32
	shards    []*shard
	size      int32
}

type shard struct {
	values map[string]ItemWithChecker
	mutex  *sync.RWMutex
}

// NewShardMap creates a new shardMap and init shardSize.
func NewShardMap(size uint32) *shardMap {
	m := &shardMap{
		shardSize: size,
		shards:    make([]*shard, size),
		size:      0,
	}
	for i := range m.shards {
		m.shards[i] = &shard{
			values: make(map[string]ItemWithChecker),
			mutex:  &sync.RWMutex{},
		}
	}
	return m
}

// getShard returns shard under given instanceId.
func (m *shardMap) getShard(instanceId string) *shard {
	return m.shards[fnv32(instanceId)%m.shardSize]
}

// Store stores values under given instanceId.
func (m *shardMap) Store(instanceId string, value ItemWithChecker) {
	shard := m.getShard(instanceId)
	shard.mutex.Lock()
	_, ok := shard.values[instanceId]
	if ok {
		shard.values[instanceId] = value
	} else {
		shard.values[instanceId] = value
		atomic.AddInt32(&m.size, 1)
	}
	shard.mutex.Unlock()
}

// PutIfAbsent to avoid storing twice when key is the same in the concurrent scenario.
func (m *shardMap) PutIfAbsent(instanceId string, value ItemWithChecker) (ItemWithChecker, bool) {
	shard := m.getShard(instanceId)
	shard.mutex.Lock()
	oldValue, has := shard.values[instanceId]
	if !has {
		shard.values[instanceId] = value
		shard.mutex.Unlock()
		atomic.AddInt32(&m.size, 1)
		return oldValue, true
	}
	shard.mutex.Unlock()
	return value, false
}

// Load loads the values under the instanceId.
func (m *shardMap) Load(instanceId string) (value ItemWithChecker, ok bool) {
	shard := m.getShard(instanceId)
	shard.mutex.RLock()
	exist, ok := shard.values[instanceId]
	shard.mutex.RUnlock()
	return exist, ok
}

// Delete deletes the values under the given instanceId.
func (m *shardMap) Delete(instanceId string) {
	shard := m.getShard(instanceId)
	shard.mutex.Lock()
	_, ok := shard.values[instanceId]
	if ok {
		delete(shard.values, instanceId)
		atomic.AddInt32(&m.size, -1)
	}
	shard.mutex.Unlock()
}

// DeleteIfExist to avoid deleting twice when key is the same in the concurrent scenario.
func (m *shardMap) DeleteIfExist(instanceId string) bool {
	if len(instanceId) == 0 {
		return false
	}
	shard := m.getShard(instanceId)
	shard.mutex.Lock()
	_, ok := shard.values[instanceId]
	if ok {
		delete(shard.values, instanceId)
		atomic.AddInt32(&m.size, -1)
		shard.mutex.Unlock()
		return true
	}
	shard.mutex.Unlock()
	return false
}

// Range iterates over the shardMap.
func (m *shardMap) Range(fn func(instanceId string, value ItemWithChecker)) {
	for _, shard := range m.shards {
		shard.mutex.RLock()
		for k, v := range shard.values {
			fn(k, v)
		}
		shard.mutex.RUnlock()
	}
}

// Count returns the number of elements within the map.
func (m *shardMap) Count() int32 {
	return atomic.LoadInt32(&m.size)
}

// FNV hash.
func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}
