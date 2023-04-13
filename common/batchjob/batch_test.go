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

package batchjob

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBatchController(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
	})

	total := 1000

	totalTasks := int32(0)
	testHandle := func(futures []Future) {
		atomic.AddInt32(&totalTasks, int32(len(futures)))
		for i := range futures {
			futures[i].Reply(nil, nil)
		}
	}

	ctrl := NewBatchController(ctx, CtrlConfig{
		QueueSize:     32,
		MaxBatchCount: 16,
		WaitTime:      32 * time.Millisecond,
		Concurrency:   8,
		Handler:       testHandle,
	})

	wg := &sync.WaitGroup{}

	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			future := ctrl.Submit(fmt.Sprintf("%d", i))
			_, _ = future.Done()
		}(i)
	}

	wg.Wait()
	assert.Equal(t, total, int(atomic.LoadInt32(&totalTasks)))
}

func TestNewBatchControllerTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
	})

	total := 1000

	totalTasks := int32(0)
	testHandle := func(futures []Future) {
		atomic.AddInt32(&totalTasks, int32(len(futures)))
	}

	ctrl := NewBatchController(ctx, CtrlConfig{
		QueueSize:     total * 2,
		MaxBatchCount: total * 2,
		WaitTime:      32 * time.Second,
		Concurrency:   8,
		Handler:       testHandle,
	})

	wg := &sync.WaitGroup{}

	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			future := ctrl.SubmitWithTimeout(fmt.Sprintf("%d", i), time.Second)
			_, err := future.Done()
			assert.True(t, errors.Is(err, context.DeadlineExceeded), err)
		}(i)
	}

	wg.Wait()
}
