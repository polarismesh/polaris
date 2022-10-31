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

package timewheel

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// test timewheel task run
func TestTaskRun1(t *testing.T) {
	tw := New(time.Second, 5, "test tw")
	tw.Start()
	callback := func(data interface{}) {
		fmt.Println(data.(string))
	}

	t.Logf("add task time:%d", time.Now().Unix())
	for i := 0; i < 10; i++ {
		tw.AddTask(1000, "polaris 1s "+strconv.Itoa(i), callback)
	}
	t.Logf("add task time end:%d", time.Now().Unix())

	time.Sleep(2 * time.Second)
	t.Logf("add task time:%d", time.Now().Unix())
	for i := 0; i < 10; i++ {
		tw.AddTask(3000, "polaris 3s "+strconv.Itoa(i), callback)
	}
	t.Logf("add task time end:%d", time.Now().Unix())

	time.Sleep(5 * time.Second)
	t.Logf("add task time:%d", time.Now().Unix())
	for i := 0; i < 10; i++ {
		tw.AddTask(10000, "polaris 10s "+strconv.Itoa(i), callback)
	}
	t.Logf("add task time end:%d", time.Now().Unix())
	time.Sleep(15 * time.Second)

	tw.Stop()
}

// test timewheel task run
func TestTaskRun2(t *testing.T) {
	tw := New(time.Second, 5, "test tw")
	tw.Start()
	callback := func(data interface{}) {
		now := time.Now().Unix()
		if now != 3123124121 {
			_ = fmt.Sprintf("%s%+v", data.(string), time.Now())
		} else {
			_ = fmt.Sprintf("%s%+v", data.(string), time.Now())
		}
	}

	t.Logf("add task time:%d", time.Now().Unix())
	for i := 0; i < 50000; i++ {
		tw.AddTask(3000, "polaris 3s "+strconv.Itoa(i), callback)
	}
	t.Logf("add task time end:%d", time.Now().Unix())
	time.Sleep(8)

	tw.Stop()
}

// test timewheel task run
func TestTaskRunBoth(t *testing.T) {
	tw := New(time.Second, 5, "test tw")
	tw.Start()
	callback := func(data interface{}) {
		fmt.Println(data.(string))
	}

	for i := 0; i < 10; i++ {
		go tw.AddTask(1000, "polaris 1s_"+strconv.Itoa(i), callback)
		go tw.AddTask(3000, "polaris 3s_"+strconv.Itoa(i), callback)
		go tw.AddTask(7000, "polaris 10s_"+strconv.Itoa(i), callback)
	}
	time.Sleep(12 * time.Second)
	tw.Stop()
}

// timewheel task struct
type Info struct {
	id  string
	ttl int
	ms  int64
}

// bench-test timewheel task add
func BenchmarkAddTask1(t *testing.B) {
	tw := New(time.Second, 5, "test tw")
	info := &Info{
		"abcdefghijklmnopqrstuvwxyz",
		2,
		time.Now().Unix(),
	}

	callback := func(data interface{}) {
		dataInfo := data.(*Info)
		if dataInfo.ms < time.Now().Unix() {
			fmt.Println("overtime")
		}
	}

	// t.N = 100000
	t.SetParallelism(10000)
	t.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tw.AddTask(2000, info, callback)
		}
	})
}

// bench-test timewheel task add
// use 2 slot
func BenchmarkAddTask2(t *testing.B) {
	tw := New(time.Second, 5, "test tw")
	info := &Info{
		"abcdefghijklmnopqrstuvwxyz",
		2,
		time.Now().Unix(),
	}

	callback := func(data interface{}) {
		dataInfo := data.(*Info)
		if dataInfo.ms < time.Now().Unix() {
			fmt.Println("overtime")
		}
	}

	t.SetParallelism(10000)
	t.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tw.AddTask(2000, info, callback)
			tw.AddTask(3000, info, callback)
		}
	})
}

// bench-test timewheel task add
// use 2 timewheel
func BenchmarkAddTask3(t *testing.B) {
	tw := New(time.Second, 5, "test tw")
	tw2 := New(time.Second, 5, "test tw")

	info := &Info{
		"abcdefghijklmnopqrstuvwxyz",
		2,
		time.Now().Unix(),
	}

	callback := func(data interface{}) {
		dataInfo := data.(*Info)
		if dataInfo.ms < time.Now().Unix() {
			fmt.Println("overtime")
		}
	}

	t.SetParallelism(10000)
	t.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tw.AddTask(2000, info, callback)
			tw2.AddTask(2000, info, callback)
		}
	})
}

// result:select random get ch
func TestSelect(t *testing.T) {
	ch := make(chan int, 20)
	ch2 := make(chan int, 20)
	stopCh := make(chan bool)

	go func() {
		for i := 0; i < 10; i++ {
			ch <- i
			ch2 <- i + 20
		}
		time.Sleep(1 * time.Second)
		close(stopCh)
	}()

	for {
		select {
		case i := <-ch:
			fmt.Println(i)
			time.Sleep(time.Second)
		case i := <-ch2:
			fmt.Println(i)
			time.Sleep(time.Second)
		case <-stopCh:
			return
		}
	}
}

func TestRotationTask(t *testing.T) {
	tw := New(time.Second, 5, "")
	tw.Start()
	wg := &sync.WaitGroup{}
	wg.Add(5)
	t.Run("", func(t *testing.T) {
		rotationCallback(t, wg, tw, 1, 0, time.Now().UnixMilli())
	})
	t.Run("", func(t *testing.T) {
		rotationCallback(t, wg, tw, 3, 0, time.Now().UnixMilli())
	})
	t.Run("", func(t *testing.T) {
		rotationCallback(t, wg, tw, 5, 0, time.Now().UnixMilli())
	})
	t.Run("", func(t *testing.T) {
		rotationCallback(t, wg, tw, 10, 0, time.Now().UnixMilli())
	})
	t.Run("", func(t *testing.T) {
		rotationCallback(t, wg, tw, 12, 0, time.Now().UnixMilli())
	})
	wg.Wait()
	tw.Stop()
}

func rotationCallback(t *testing.T, wg *sync.WaitGroup, tw *TimeWheel, intervalSecond int64, runTimes int, lastTime int64) {
	tw.AddTask(uint32(intervalSecond*time.Second.Milliseconds()), nil, func(i interface{}) {
		//0.800-1.200
		fmt.Println(time.Now())
		interval := time.Now().UnixMilli() - lastTime
		if runTimes == 0 {
			//首次减去时间轮启动的刻度时间
			interval = interval - tw.interval.Milliseconds()
		}
		diff := math.Abs(float64(interval - intervalSecond*1000))
		assert.True(t, diff < 200)
		if runTimes > 3 {
			wg.Done()
			return
		}
		runTimes++
		rotationCallback(t, wg, tw, intervalSecond, runTimes, time.Now().UnixMilli())
	})
}

func TestForceCloseMode(t *testing.T) {
	a := 1
	tw := New(time.Second, 5, "force close", WithWaitTaskOnClose(false))
	tw.Start()
	tw.AddTask(uint32(2*time.Second.Milliseconds()), nil, func(i interface{}) {
		time.Sleep(5 * time.Second)
		a = 2
		fmt.Println("run end")
	})
	tw.AddTask(uint32(2*time.Second.Milliseconds()), nil, func(i interface{}) {
		time.Sleep(6 * time.Second)
		a = 2
		fmt.Println("task2 run end")
	})
	time.Sleep(4 * time.Second)
	tw.Stop()
	fmt.Println("tw is stop")
	assert.True(t, a == 1)
}

func TestWaitCloseMode(t *testing.T) {
	a := 1
	tw := New(time.Second, 5, "force close")
	tw.Start()
	tw.AddTask(uint32(2*time.Second.Milliseconds()), nil, func(i interface{}) {
		time.Sleep(5 * time.Second)
		a = 2
		fmt.Println("task1 run end")
	})

	tw.AddTask(uint32(2*time.Second.Milliseconds()), nil, func(i interface{}) {
		time.Sleep(6 * time.Second)
		a = 2
		fmt.Println("task2 run end")
	})
	time.Sleep(4 * time.Second)
	tw.Stop()
	fmt.Println("tw is stop")
	assert.True(t, a == 2)
}
