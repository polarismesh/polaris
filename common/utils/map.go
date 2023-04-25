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

package utils

import "sync"

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
