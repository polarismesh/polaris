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
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

const MapLen = 10000     //  Used to represent the amount of  InstanceWithChecker
const Parallelism = 1024 //  Used to represent the amount of  Concurrency

type MutexMap struct {
	sync.RWMutex
	m map[string]*InstanceWithChecker
}

func (m *MutexMap) Set(k string, v *InstanceWithChecker) {
	m.Lock()
	m.m[k] = v
	m.Unlock()
}

func (m *MutexMap) Get(k string) (v *InstanceWithChecker) {
	m.RLock()
	v = m.m[k]
	m.RUnlock()
	return
}

func (m *MutexMap) Del(k string) (v interface{}) {
	m.Lock()
	delete(m.m, k)
	m.Unlock()
	return
}

func BenchmarkSyncMap(b *testing.B) {
	syncMap := new(sync.Map)
	var key uint32
	b.ReportAllocs()
	b.StartTimer()
	b.SetParallelism(Parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := atomic.AddUint32(&key, 1)
			syncMap.Store(key, nil)
			syncMap.Load(key)
			syncMap.Delete(key)
		}
	})
}

func BenchmarkMutexMap(b *testing.B) {
	mutexMap := MutexMap{m: make(map[string]*InstanceWithChecker, MapLen)}
	var key uint32
	b.ReportAllocs()
	b.StartTimer()
	b.SetParallelism(Parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := atomic.AddUint32(&key, 1)
			mutexMap.Set(strconv.Itoa(int(key)), nil)
			mutexMap.Get(strconv.Itoa(int(key)))
			mutexMap.Del(strconv.Itoa(int(key)))
		}
	})
}

func BenchmarkSharedMap2(b *testing.B) {
	m := NewShardMap(2)
	var key int32
	var mod int32 = 1023
	b.ReportAllocs()
	b.StartTimer()
	b.SetParallelism(Parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := atomic.AddInt32(&key, 1) & mod
			m.Store(strconv.Itoa(int(key)), nil)
			m.Load(strconv.Itoa(int(key)))
			m.Delete(strconv.Itoa(int(key)))
		}
	})
}

func BenchmarkSharedMap4(b *testing.B) {
	m := NewShardMap(4)
	var key int32
	b.ReportAllocs()
	b.StartTimer()
	b.SetParallelism(Parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := atomic.AddInt32(&key, 1)
			m.Store(strconv.Itoa(int(key)), nil)
			m.Load(strconv.Itoa(int(key)))
			m.Delete(strconv.Itoa(int(key)))
		}
	})
}

func BenchmarkSharedMap8(b *testing.B) {
	m := NewShardMap(8)
	var key int32
	b.ReportAllocs()
	b.StartTimer()
	b.SetParallelism(Parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := atomic.AddInt32(&key, 1)
			m.Store(strconv.Itoa(int(key)), nil)
			m.Load(strconv.Itoa(int(key)))
			m.Delete(strconv.Itoa(int(key)))
		}
	})
}

func BenchmarkSharedMap16(b *testing.B) {
	m := NewShardMap(16)
	var key int32
	b.ReportAllocs()
	b.StartTimer()
	b.SetParallelism(Parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := atomic.AddInt32(&key, 1)
			m.Store(strconv.Itoa(int(key)), nil)
			m.Load(strconv.Itoa(int(key)))
			m.Delete(strconv.Itoa(int(key)))
		}
	})
}

func BenchmarkSharedMap32(b *testing.B) {
	m := NewShardMap(32)
	var key int32
	b.ReportAllocs()
	b.StartTimer()
	b.SetParallelism(Parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := atomic.AddInt32(&key, 1)
			m.Store(strconv.Itoa(int(key)), nil)
			m.Load(strconv.Itoa(int(key)))
			m.Delete(strconv.Itoa(int(key)))
		}
	})
}

func BenchmarkSharedMap64(b *testing.B) {
	m := NewShardMap(64)
	var key int32
	b.ReportAllocs()
	b.StartTimer()
	b.SetParallelism(Parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := atomic.AddInt32(&key, 1)
			m.Store(strconv.Itoa(int(key)), nil)
			m.Load(strconv.Itoa(int(key)))
			m.Delete(strconv.Itoa(int(key)))
		}
	})
}

func BenchmarkSharedMap128(b *testing.B) {
	m := NewShardMap(128)
	var key int32
	b.ReportAllocs()
	b.StartTimer()
	b.SetParallelism(Parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := atomic.AddInt32(&key, 1)
			m.Store(strconv.Itoa(int(key)), nil)
			m.Load(strconv.Itoa(int(key)))
			m.Delete(strconv.Itoa(int(key)))
		}
	})
}

func BenchmarkSharedMap256(b *testing.B) {
	m := NewShardMap(256)
	var key int32
	b.ReportAllocs()
	b.StartTimer()
	b.SetParallelism(Parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := atomic.AddInt32(&key, 1)
			m.Store(strconv.Itoa(int(key)), nil)
			m.Load(strconv.Itoa(int(key)))
			m.Delete(strconv.Itoa(int(key)))
		}
	})
}

func BenchmarkSharedMap512(b *testing.B) {
	m := NewShardMap(512)
	var key int32
	b.ReportAllocs()
	b.StartTimer()
	b.SetParallelism(Parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := atomic.AddInt32(&key, 1)
			m.Store(strconv.Itoa(int(key)), nil)
			m.Load(strconv.Itoa(int(key)))
			m.Delete(strconv.Itoa(int(key)))
		}
	})
}

func BenchmarkSharedMap1024(b *testing.B) {
	m := NewShardMap(1024)
	var key int32
	b.ReportAllocs()
	b.StartTimer()
	b.SetParallelism(Parallelism)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := atomic.AddInt32(&key, 1)
			m.Store(strconv.Itoa(int(key)), nil)
			m.Load(strconv.Itoa(int(key)))
			m.Delete(strconv.Itoa(int(key)))
		}
	})
}
