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

package local

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/common/model"
)

func Test_discoverEventLocal_Run(t *testing.T) {

	totalCnt := 10

	wait := sync.WaitGroup{}
	wait.Add(totalCnt)

	testFn := func(eb *eventBufferHolder) {
		for eb.HasNext() {
			wait.Done()
			et := eb.Next()
			t.Logf("%v", et)
			_, ok := subscribeEvents[et.EType]
			assert.True(t, ok)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	l := &discoverEventLocal{
		eventCh: make(chan model.InstanceEvent, 32),
		bufferPool: sync.Pool{
			New: func() interface{} { return newEventBufferHolder(defaultBufferSize) },
		},
		cursor:       0,
		syncLock:     sync.Mutex{},
		eventHandler: testFn,
		cancel:       cancel,
	}
	l.switchEventBuffer()

	t.Cleanup(func() {
		l.Destroy()
	})

	go l.Run(ctx)

	for i := 0; i < totalCnt; i++ {
		l.PublishEvent(model.InstanceEvent{
			Id:        "123456",
			Namespace: "DemoNamespace",
			Service:   "DemoService",
			Instance: &apiservice.Instance{
				Host: &wrappers.StringValue{Value: "127.0.0.1"},
				Port: &wrappers.UInt32Value{Value: 8080},
			},
			EType:      model.EventInstanceCloseIsolate,
			CreateTime: time.Time{},
		})

		l.PublishEvent(model.InstanceEvent{
			Id:        "111111",
			Namespace: "DemoNamespace",
			Service:   "DemoService",
			Instance: &apiservice.Instance{
				Host: &wrappers.StringValue{Value: "127.0.0.1"},
				Port: &wrappers.UInt32Value{Value: 8080},
			},
			EType:      model.EventInstanceSendHeartbeat,
			CreateTime: time.Time{},
		})

		l.PublishEvent(model.InstanceEvent{
			Id:        "111111",
			Namespace: "DemoNamespace",
			Service:   "DemoService",
			Instance: &apiservice.Instance{
				Host: &wrappers.StringValue{Value: "127.0.0.1"},
				Port: &wrappers.UInt32Value{Value: 8080},
			},
			EType:      model.EventInstanceUpdate,
			CreateTime: time.Time{},
		})
	}

	wait.Wait()
}
