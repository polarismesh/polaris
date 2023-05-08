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

const (
	shutdownNow      = 1
	shutdownGraceful = 2
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
		handler: func(tasks []Future) {
			defer func() {
				if err := recover(); err != nil {
					log.Errorf("[Batch] %s trigger consumer panic : %+v", conf.Label, err)
				}
			}()
			conf.Handler(tasks)
		},
	}
	bc.runWorkers(ctx)
	bc.mainLoop(ctx)
	return bc
}

// Submit 提交执行任务参数
func (bc *BatchController) Submit(task Param) Future {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	if bc.isStop() {
		return &errorFuture{task: task, err: ErrorBatchControllerStopped}
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

func (bc *BatchController) SubmitWithTimeout(task Param, timeout time.Duration) Future {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	if bc.isStop() {
		return &errorFuture{task: task, err: ErrorBatchControllerStopped}
	}

	ctx, cancel := context.WithCancel(context.Background())
	f := &future{
		task:      task,
		ctx:       ctx,
		cancel:    cancel,
		setsignal: make(chan struct{}),
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-timer.C:
		f.Cancel()
		return &errorFuture{task: task, err: context.DeadlineExceeded}
	case bc.tasksChan <- f:
		return f
	}
}

func (bc *BatchController) runWorkers(ctx context.Context) {
	wait := sync.WaitGroup{}
	wait.Add(int(bc.conf.Concurrency))

	workerLoop := func(index int) {
		stopFunc := func() {
			defer wait.Done()
			switch atomic.LoadInt32(&bc.stop) {
			case shutdownGraceful:
				replied := 0
				for futures := range bc.workers[index] {
					replied += len(futures)
					bc.handler(futures)
					bc.idleSignal <- int(index)
				}
				log.Infof("[Batch] %s worker(%d) exit, handle future count: %d", bc.label, index, replied)
			case shutdownNow:
				stopped := 0
				for futures := range bc.workers[index] {
					replyStoppedFutures(futures...)
					stopped += len(futures)
				}
				log.Infof("[Batch] %s worker(%d) exit, reply stop msg to future count: %d", bc.label, index, stopped)
			}
		}

		for {
			select {
			case <-ctx.Done():
				stopFunc()
				return
			case futures := <-bc.workers[index]:
				bc.handler(futures)
				bc.idleSignal <- int(index)
			}
		}
	}

	for i := uint32(0); i < bc.conf.Concurrency; i++ {
		index := i
		bc.workers = append(bc.workers, make(chan []Future))
		log.Infof("[Batch] %s worker(%d) running in main loop", bc.label, index)
		bc.idleSignal <- int(index)
		go func(index uint32) {
			workerLoop(int(index))
		}(index)
	}

	go func() {
		wait.Wait()
		log.Infof("[Batch] %s close idle worker signal", bc.label)
		close(bc.idleSignal)
	}()

}

func (bc *BatchController) mainLoop(ctx context.Context) {
	futures := make([]Future, 0, bc.conf.MaxBatchCount)
	idx := 0
	triggerConsume := func(data []Future) {
		if idx == 0 {
			return
		}
		idleIndex := <-bc.idleSignal
		bc.workers[idleIndex] <- data
		futures = make([]Future, 0, bc.conf.MaxBatchCount)
		idx = 0
	}

	stopFunc := func() {
		close(bc.tasksChan)
		switch atomic.LoadInt32(&bc.stop) {
		case shutdownGraceful:
			triggerConsume(futures[0:idx])
			for future := range bc.tasksChan {
				futures = append(futures, future)
				idx++
				if idx == int(bc.conf.MaxBatchCount) {
					triggerConsume(futures[0:idx])
				}
			}
			for i := range bc.workers {
				close(bc.workers[i])
			}
		case shutdownNow:
			log.Infof("[Batch] %s begin close worker loop", bc.label)
			for i := range bc.workers {
				close(bc.workers[i])
			}
			stopped := len(futures)
			replyStoppedFutures(futures...)
			for future := range bc.tasksChan {
				replyStoppedFutures(future)
				stopped++
			}
			log.Infof("[Batch] %s do reply stop msg to future count: %d", bc.label, stopped)
		}
		log.Infof("[Batch] %s main loop exited", bc.label)
	}

	go func() {
		log.Infof("[Batch] %s running main loop", bc.label)
		ticker := time.NewTicker(bc.conf.WaitTime)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				stopFunc()
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

// Stop 关闭批任务执行器
func (bc *BatchController) Stop() {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	log.Infof("[Batch] %s begin do stop", bc.label)
	atomic.StoreInt32(&bc.stop, shutdownNow)
	bc.cancel()
}

// Stop 关闭批任务执行器
func (bc *BatchController) GracefulStop() {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	atomic.StoreInt32(&bc.stop, shutdownGraceful)
	bc.cancel()
}

func (bc *BatchController) isStop() bool {
	return atomic.LoadInt32(&bc.stop) == 1 || atomic.LoadInt32(&bc.stop) == shutdownGraceful
}

func replyStoppedFutures(futures ...Future) {
	for i := range futures {
		futures[i].Reply(nil, ErrorBatchControllerStopped)
	}
}
