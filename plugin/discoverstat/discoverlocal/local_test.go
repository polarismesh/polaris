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

package discoverlocal

import (
	"fmt"
	"testing"
	"time"
)

// TestWriteFile 测试打印文件所需耗时
func TestWriteFile(t *testing.T) {
	dcs := &DiscoverCallStatis{
		statis: make(map[Service]time.Time),
		logger: newLogger("./log/discovercall_test1.log"),
	}
	namespace := "Test"
	totals := []int{25, 50, 100, 150}
	for _, num := range totals {
		count := 10000
		for i := 0; i <= num; i++ {
			for j := 0; j <= count; j++ {
				name := fmt.Sprintf("test-service-%d-%d", i, j)
				service := Service{
					name:      name,
					namespace: namespace,
				}
				dcs.statis[service] = time.Now()
			}
		}

		startTime := time.Now()
		dcs.log()
		endTime := time.Now()
		t.Logf("total num is %d, duration is %v", num*count, endTime.Sub(startTime))
	}

}

// TestDiscoverStatisWorker_AddDiscoverCall 测试写入chan的情况
func TestDiscoverStatisWorker_AddDiscoverCall(t *testing.T) {
	worker := &DiscoverStatisWorker{
		interval: 60 * time.Second,
		dcc:      make(chan *DiscoverCall, 1024),
		dcs: &DiscoverCallStatis{
			statis: make(map[Service]time.Time),
			logger: newLogger("./log/discovercall_test2.log"),
		},
	}

	workerStarted := make(chan struct{}, 1)
	go func() {
		workerStarted <- struct{}{}
		worker.Run()
	}()
	<-workerStarted // 等待 worker 启动

	namespace := "Test"
	timeout := time.After(time.Minute * 3)
	stop := make(chan struct{})
	for i := 0; i < 1000; i++ {
		go func(stop chan struct{}, index int) {
			trigger := time.NewTicker(time.Millisecond * 10)
			defer trigger.Stop()

			for {
				select {
				case <-trigger.C:
					name := fmt.Sprintf("test-service-%d", index)
					if err := worker.AddDiscoverCall(name, namespace, time.Now()); err != nil {
						t.Errorf("err: %s", err.Error())
					}
				case <-stop:
					return
				}
			}
		}(stop, i)
	}
	<-timeout
	stop <- struct{}{}
	t.Log("pass")
}
