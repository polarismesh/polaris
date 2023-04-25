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
	"sync"
	"sync/atomic"
	"time"

	"github.com/polarismesh/polaris/common/log"
)

var (
	ErrorBatchControllerStopped = errors.New("batch controller is stopped")
)

// BatchController 通用的批任务处理框架
type BatchController struct {
	lock       sync.RWMutex
	stop       int32
	label      string
	conf       CtrlConfig
	handler    func(tasks []Future)
	tasksChan  chan Future
	idleSignal chan int
	workers    []chan []Future
	cancel     context.CancelFunc
}

// NewBatchController 创建一个批任务处理
func NewBatchController(ctx context.Context, conf CtrlConfig) *BatchController {
	ctx, cancel := context.WithCancel(ctx)
	bc := &BatchController{
		label:      conf.Label,
		conf:       conf,
		cancel:     cancel,
		tasksChan:  make(chan Future, conf.QueueSize),
		workers:    make([]chan []Future, 0, conf.Concurrency),
		idleSignal: make(chan int, conf.Concurrency),
		handler:    conf.Handler,
	}
	bc.runWorkers(ctx)
	bc.mainLoop(ctx)
	return bc
}

// Stop 关闭批任务执行器
func (bc *BatchController) Stop() {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	atomic.StoreInt32(&bc.stop, 1)
	bc.cancel()
	if bc.tasksChan != nil {
		close(bc.tasksChan)
	}
	if bc.idleSignal != nil {
		close(bc.idleSignal)
	}
	for i := range bc.workers {
		item := bc.workers[i]
		if item != nil {
			close(item)
		}
	}
}

func (bc *BatchController) isStop() bool {
	return atomic.LoadInt32(&bc.stop) == 1
}

// Submit 提交执行任务参数
func (bc *BatchController) Submit(task Task) Future {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	if bc.isStop() {
		return &errorFuture{task: task}
	}

	ctx, cancel := context.WithCancel(context.Background())
	f := &future{
		task:      task,
		ctx:       ctx,
		cancel:    cancel,
		setsignal: make(chan struct{}),
	}
	bc.tasksChan <- f
	return f
}

func (bc *BatchController) SubmitWithTimeout(task Task, timeout time.Duration) Future {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	if bc.isStop() {
		return &errorFuture{task: task}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	f := &future{
		task:      task,
		ctx:       ctx,
		cancel:    cancel,
		setsignal: make(chan struct{}),
	}
	bc.tasksChan <- f
	return f
}

func (bc *BatchController) runWorkers(ctx context.Context) {
	for i := uint32(0); i < bc.conf.Concurrency; i++ {
		index := i
		bc.workers = append(bc.workers, make(chan []Future))
		log.Infof("[Batch] %s worker(%d) running in main loop", bc.label, index)
		bc.idleSignal <- int(index)
		go func() {
			for {
				select {
				case <-ctx.Done():
					log.Infof("[Batch] %s worker(%d) exited", bc.label, index)
					return
				case futures := <-bc.workers[index]:
					bc.handler(futures)

					bc.lock.RLock()
					defer bc.lock.RUnlock()
					if bc.isStop() {
						return
					}
					bc.idleSignal <- int(index)
				}
			}
		}()
	}
}

func (bc *BatchController) mainLoop(ctx context.Context) {
	futures := make([]Future, 0, bc.conf.MaxBatchCount)
	idx := 0
	triggerConsume := func(data []Future) {
		bc.lock.RLock()
		defer bc.lock.RUnlock()
		if bc.isStop() {
			return
		}
		if idx == 0 {
			return
		}
		idleIndex := <-bc.idleSignal
		bc.workers[idleIndex] <- data
		futures = make([]Future, 0, bc.conf.MaxBatchCount)
		idx = 0
	}
	go func() {
		ticker := time.NewTicker(bc.conf.WaitTime)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Infof("[Batch] %s main loop exited", bc.label)
				return
			case <-ticker.C:
				triggerConsume(futures[0:idx])
			case future := <-bc.tasksChan:
				futures = append(futures, future)
				idx++
				if idx == int(bc.conf.MaxBatchCount) {
					triggerConsume(futures[0:idx])
				}
			}
		}
	}()
}
