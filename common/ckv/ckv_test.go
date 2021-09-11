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

package ckv

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkAdd 压测并发无锁加操作
func BenchmarkAdd(t *testing.B) {
	var a uint32

	// 共启动10000 * cpu数量个协程
	t.SetParallelism(10000)
	t.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			a++
		}
	})
}

// BenchmarkAtomicAdd 压测并发原子加操作
func BenchmarkAtomicAdd(t *testing.B) {
	var a uint32

	// 共启动10000 * cpu数量个协程
	t.SetParallelism(10000)
	t.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			b := atomic.AddUint32(&a, 1)
			_ = b % 500
		}
	})
}

// BenchmarkMutexAdd 压测并发有锁加操作
func BenchmarkMutexAdd(t *testing.B) {
	var a uint32
	var mu sync.Mutex

	// 共启动10000 * cpu数量个协程
	t.SetParallelism(10000)
	t.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			a++
			mu.Unlock()
		}
	})
}

// BenchmarkRand 压测并发生成随机数操作
func BenchmarkRand(t *testing.B) {
	rand.Seed(time.Now().Unix())

	// 共启动10000 * cpu数量个协程
	t.SetParallelism(10000)
	t.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = rand.Intn(500)
		}
	})
}

// BenchmarkTestAdd 压测直接加操作
func BenchmarkTestAdd(t *testing.B) {
	var a = ^uint32(0)
	t.Log(a)
	a++
	t.Log(a)
}

// TestExit 测试os.Exit()
func TestExit(t *testing.T) {
	go func() {
		os.Exit(1)
	}()

	time.Sleep(time.Second)
	fmt.Println("a")
}

// TestGoExit 测试runtime.Goexit()
func TestGoExit(t *testing.T) {
	go func() {
		defer fmt.Println("aaa")
		runtime.Goexit()
	}()

	time.Sleep(time.Second)
	fmt.Println("a")
}
