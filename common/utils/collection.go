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

import (
	"sync"
)

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

// NewRefSyncSet returns a new Set
func NewRefSyncSet[K comparable]() *RefSyncSet[K] {
	return &RefSyncSet[K]{
		container: make(map[K]int),
	}
}

type RefSyncSet[K comparable] struct {
	container map[K]int
	lock      sync.RWMutex
}

// Add adds a string to the set
func (set *RefSyncSet[K]) Add(val K) {
	set.lock.Lock()
	defer set.lock.Unlock()

	ref, ok := set.container[val]
	if ok {
		ref++
	}
	set.container[val] = ref
}

// Remove removes a string from the set
func (set *RefSyncSet[K]) Remove(val K) {
	set.lock.Lock()
	defer set.lock.Unlock()
	ref, ok := set.container[val]
	if ok {
		ref--
	}
	if ref == 0 {
		delete(set.container, val)
	} else {
		set.container[val] = ref
	}
}

func (set *RefSyncSet[K]) ToSlice() []K {
	set.lock.RLock()
	defer set.lock.RUnlock()

	ret := make([]K, 0, len(set.container))
	for k := range set.container {
		ret = append(ret, k)
	}
	return ret
}

func (set *RefSyncSet[K]) Range(fn func(val K)) {
	set.lock.RLock()
	snapshot := map[K]struct{}{}
	for k := range set.container {
		snapshot[k] = struct{}{}
	}
	set.lock.RUnlock()

	for k := range snapshot {
		fn(k)
	}
}

func (set *RefSyncSet[K]) Len() int {
	set.lock.RLock()
	defer set.lock.RUnlock()

	return len(set.container)
}

// Contains contains target value
func (set *RefSyncSet[K]) Contains(val K) bool {
	set.lock.Lock()
	defer set.lock.Unlock()

	_, exist := set.container[val]
	return exist
}

func (set *RefSyncSet[K]) String() string {
	ret := set.ToSlice()
	return MustJson(ret)
}

// NewSyncSet returns a new Set
func NewSyncSet[K comparable]() *SyncSet[K] {
	return &SyncSet[K]{
		container: make(map[K]struct{}),
	}
}

type SyncSet[K comparable] struct {
	container map[K]struct{}
	lock      sync.RWMutex
}

// Add adds a string to the set
func (set *SyncSet[K]) Add(val K) {
	set.lock.Lock()
	defer set.lock.Unlock()

	set.container[val] = struct{}{}
}

// Add adds a string to the set
func (set *SyncSet[K]) AddAll(vals *SyncSet[K]) {
	vals.Range(func(val K) {
		set.lock.Lock()
		defer set.lock.Unlock()
		set.container[val] = struct{}{}
	})
}

// Remove removes a string from the set
func (set *SyncSet[K]) Remove(val K) {
	set.lock.Lock()
	defer set.lock.Unlock()

	delete(set.container, val)
}

func (set *SyncSet[K]) ToSlice() []K {
	set.lock.RLock()
	defer set.lock.RUnlock()

	ret := make([]K, 0, len(set.container))
	for k := range set.container {
		ret = append(ret, k)
	}
	return ret
}

func (set *SyncSet[K]) Range(fn func(val K)) {
	set.lock.RLock()
	snapshot := map[K]struct{}{}
	for k := range set.container {
		snapshot[k] = struct{}{}
	}
	set.lock.RUnlock()

	for k := range snapshot {
		fn(k)
	}
}

func (set *SyncSet[K]) Len() int {
	set.lock.RLock()
	defer set.lock.RUnlock()

	return len(set.container)
}

// Contains contains target value
func (set *SyncSet[K]) Contains(val K) bool {
	set.lock.Lock()
	defer set.lock.Unlock()

	_, exist := set.container[val]
	return exist
}

func (set *SyncSet[K]) String() string {
	ret := set.ToSlice()
	return MustJson(ret)
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
		m: make(map[K]V, 16),
	}
}

// SyncMap
type SyncMap[K comparable, V any] struct {
	lock sync.RWMutex
	m    map[K]V
}

// ComputeIfAbsent
func (s *SyncMap[K, V]) ComputeIfAbsent(k K, supplier func(k K) V) (V, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	actual, exist := s.m[k]
	if exist {
		return actual, false
	}
	val := supplier(k)
	s.m[k] = val
	return val, true
}

// Load
func (s *SyncMap[K, V]) Load(key K) (V, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	v, ok := s.m[key]
	if ok {
		return v, ok
	}
	var empty V
	return empty, false
}

// Store
func (s *SyncMap[K, V]) Store(key K, val V) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.m[key] = val
}

// Values
func (s *SyncMap[K, V]) Values() []V {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ret := make([]V, 0, len(s.m))
	for _, v := range s.m {
		ret = append(ret, v)
	}
	return ret
}

// Range
func (s *SyncMap[K, V]) Range(f func(key K, val V)) {
	s.lock.RLock()
	snapshot := map[K]V{}
	for k, v := range s.m {
		snapshot[k] = v
	}
	s.lock.RUnlock()

	for k, v := range snapshot {
		f(k, v)
	}
}

// ReadRange .
func (s *SyncMap[K, V]) ReadRange(f func(key K, val V)) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for k, v := range s.m {
		f(k, v)
	}
}

// Delete
func (s *SyncMap[K, V]) Delete(key K) (V, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	v, exist := s.m[key]
	delete(s.m, key)
	return v, exist
}

// Len
func (s *SyncMap[K, V]) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return len(s.m)
}

func (s *SyncMap[K, V]) ToMap() map[K]V {
	s.lock.RLock()
	defer s.lock.RUnlock()

	m := map[K]V{}
	for k, v := range s.m {
		m[k] = v
	}
	return m
}

// NewMap
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

// Values .
func (s *Map[K, V]) Values() []V {
	ret := make([]V, 0, s.Len())
	for _, v := range s.m {
		ret = append(ret, v)
	}
	return ret
}
