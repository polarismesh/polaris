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

// Package utils contains common utility functions
package utils

import "sync"

// NewSet returns a new Set
func NewSet[K comparable]() *Set[K] {
	return &Set[K]{
		container: make(map[K]struct{}),
	}
}

type Set[K comparable] struct {
	container map[K]struct{}
}

// Add adds a string to the set
func (set *Set[K]) Add(val K) {
	set.container[val] = struct{}{}
}

// Remove removes a string from the set
func (set *Set[K]) Remove(val K) {
	delete(set.container, val)
}

func (set *Set[K]) ToSlice() []K {
	ret := make([]K, 0, len(set.container))
	for k := range set.container {
		ret = append(ret, k)
	}
	return ret
}

func (set *Set[K]) Range(fn func(val K)) {
	for k := range set.container {
		fn(k)
	}
}

func NewSegmentMap[K comparable, V any](soltNum int, hashFunc func(k K) int) *SegmentMap[K, V] {
	locks := make([]*sync.RWMutex, 0, soltNum)
	solts := make([]map[K]V, 0, soltNum)
	for i := 0; i < int(soltNum); i++ {
		locks = append(locks, &sync.RWMutex{})
		solts = append(solts, map[K]V{})
	}
	return &SegmentMap[K, V]{
		soltNum:  soltNum,
		locks:    locks,
		solts:    solts,
		hashFunc: hashFunc,
	}
}

type SegmentMap[K comparable, V any] struct {
	soltNum  int
	locks    []*sync.RWMutex
	solts    []map[K]V
	hashFunc func(k K) int
}

func (s *SegmentMap[K, V]) Put(k K, v V) {
	lock, solt := s.caulIndex(k)
	lock.Lock()
	defer lock.Unlock()
	solt[k] = v
}

func (s *SegmentMap[K, V]) ComputeIfAbsent(k K, supplier func(k K) V) (V, bool) {
	lock, solt := s.caulIndex(k)
	lock.Lock()
	defer lock.Unlock()
	oldVal, ok := solt[k]
	if !ok {
		v := supplier(k)
		solt[k] = v
		return v, true
	}
	return oldVal, false
}

func (s *SegmentMap[K, V]) PutIfAbsent(k K, v V) (V, bool) {
	lock, solt := s.caulIndex(k)
	lock.Lock()
	defer lock.Unlock()
	oldVal, ok := solt[k]
	if !ok {
		solt[k] = v
		return oldVal, true
	}
	return oldVal, false
}

func (s *SegmentMap[K, V]) Get(k K) (V, bool) {
	lock, solt := s.caulIndex(k)
	lock.RLock()
	defer lock.RUnlock()

	v, ok := solt[k]
	return v, ok
}

func (s *SegmentMap[K, V]) Del(k K) bool {
	lock, solt := s.caulIndex(k)
	lock.Lock()
	defer lock.Unlock()

	_, ok := solt[k]
	delete(solt, k)
	return ok
}

func (s *SegmentMap[K, V]) Range(f func(k K, v V)) {
	for i := 0; i < s.soltNum; i++ {
		lock := s.locks[i]
		solt := s.solts[i]
		func() {
			lock.RLock()
			defer lock.RUnlock()
			for k, v := range solt {
				f(k, v)
			}
		}()
	}
}

func (s *SegmentMap[K, V]) Count() uint64 {
	count := uint64(0)
	for i := 0; i < s.soltNum; i++ {
		lock := s.locks[i]
		solt := s.solts[i]
		func() {
			lock.RLock()
			defer lock.RUnlock()
			count += uint64(len(solt))
		}()
	}
	return count
}

func (s *SegmentMap[K, V]) caulIndex(k K) (*sync.RWMutex, map[K]V) {
	index := s.hashFunc(k) % s.soltNum
	lock := s.locks[index]
	solt := s.solts[index]
	return lock, solt
}

// NewSyncMap
func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{
		m: &sync.Map{},
	}
}

// SyncMap
type SyncMap[K comparable, V any] struct {
	m *sync.Map
}

// Load
func (s *SyncMap[K, V]) Load(key K) (V, bool) {
	v, ok := s.m.Load(key)
	if ok {
		return v.(V), ok
	}
	var empty V
	return empty, false
}

// Store
func (s *SyncMap[K, V]) Store(key K, val V) {
	s.m.Store(key, val)
}

// Range
func (s *SyncMap[K, V]) Range(f func(key K, val V) bool) {
	s.m.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

// Delete
func (s *SyncMap[K, V]) Delete(key K) {
	s.m.Delete(key)
}

// LoadOrStore
func (s *SyncMap[K, V]) LoadOrStore(key K, val V) (V, bool) {
	actual, loaded := s.m.LoadOrStore(key, val)
	ret, _ := actual.(V)
	return ret, loaded
}

// NewSyncMap
func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		m: map[K]V{},
	}
}

// Map
type Map[K comparable, V any] struct {
	m map[K]V
}

// Load
func (s *Map[K, V]) Load(key K) (V, bool) {
	v, ok := s.m[key]
	return v, ok
}

// Store
func (s *Map[K, V]) Store(key K, val V) {
	s.m[key] = val
}

// Range
func (s *Map[K, V]) Range(f func(key K, val V)) {
	for k, v := range s.m {
		f(k, v)
	}
}

// Delete
func (s *Map[K, V]) Delete(key K) {
	delete(s.m, key)
}

// Len
func (s *Map[K, V]) Len() int {
	return len(s.m)
}
