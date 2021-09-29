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

package sqldb

import (
	"errors"
	"github.com/polarismesh/polaris-server/common/log"
	"sync"
	"time"

	"context"
)

// 用户处理函数
type Handler func(item interface{}) error

// 任务传输结构体
type Future struct {
	item    interface{}     // 需要处理的参数
	handler Handler         // 处理函数
	resp    *ResponseFuture // 任务返回future
}

// 处理任务返回的结构体
type ResponseFuture struct {
	finishCh  chan struct{} // 任务执行成功的反馈chan
	errCh     chan error    // 任务执行失败的反馈chan
	total     int           // 本次任务总数
	label     string        // 任务的标签
	doneCnt   int           // 记录收到多少个task回复的
	notifyCnt int           // 记录分发了多少个task
}

// 等待任务执行
func (r *ResponseFuture) wait() error {
	//defer log.Infof("[TaskManager][%s] finish all task(%d)", r.label, r.total)

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case err := <-r.errCh:
			// 无论成功还失败，收到回复都要增加doneCnt
			r.doneCnt++
			return err
		case <-r.finishCh:
			r.doneCnt++
			if r.doneCnt == r.total {
				return nil
			}
		case <-ticker.C:
			log.Infof("[TaskManager][%s] wait for task count(%d) response, finish progress(%d / %d)",
				r.label, r.total, r.doneCnt, r.notifyCnt)
		}
	}
}

// 返回函数
func (r *ResponseFuture) reply(err error) {
	if err != nil {
		r.errCh <- err
	} else {
		r.finishCh <- struct{}{}
	}
}

// 每个任务集需要释放资源
func (r *ResponseFuture) release() {
	// 如果收到回复的个数==任务分发的个数，那么可以退出
	if r.doneCnt >= r.notifyCnt {
		return
	}

	waitCnt := 0
	ticker := time.NewTicker(time.Second * 5) // TODO
	defer ticker.Stop()

	for {
		select {
		case <-r.errCh:
			waitCnt++
		case <-r.finishCh:
			waitCnt++
		case <-ticker.C:
			log.Infof("[TaskManager][%s] response release progress(%d / %d)",
				r.label, waitCnt, r.notifyCnt-r.doneCnt)
		}

		if waitCnt+r.doneCnt == r.notifyCnt {
			close(r.errCh)
			close(r.finishCh)
			return
		}
	}
}

// 任务管理器，全局可以有多个，不过尽量全局只有一个
type TaskManager struct {
	recvCh      chan *Future
	concurrence int
	exitCh      chan struct{}
}

// 新建任务管理器
func NewTaskManager(concurrence int) (*TaskManager, error) {
	if concurrence <= 0 {
		return nil, errors.New("max channel count is error")
	}

	t := &TaskManager{
		recvCh:      make(chan *Future, concurrence),
		concurrence: concurrence,
		exitCh:      make(chan struct{}),
	}

	return t, nil
}

// 执行任务集
func (t *TaskManager) Do(label string, data []interface{}, handler Handler) error {
	if len(data) == 0 {
		return nil
	}

	count := len(data)
	maxCount := 64
	if count > maxCount {
		count = maxCount
	}

	resp := &ResponseFuture{
		finishCh:  make(chan struct{}, count),
		errCh:     make(chan error, count),
		total:     len(data),
		label:     label,
		doneCnt:   0,
		notifyCnt: 0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Add(1)
	go func(recvCh chan *Future) {
		defer wg.Done()
		for i := range data {
			select {
			case <-ctx.Done():
				return
			default:
			}

			resp.notifyCnt++
			future := &Future{
				item:    data[i],
				handler: handler,
				resp:    resp,
			}
			recvCh <- future

			if resp.notifyCnt%10000 == 0 {
				log.Infof("[TaskManager][%s] task notify progress(%d / %d)", label, resp.notifyCnt, len(data))
			}
		}
	}(t.recvCh)

	// 先到等待任务执行，有可能执行失败，有可能全部执行完
	err := resp.wait()

	// 触发分发协程退出
	cancel()

	// 等待分发协程彻底退出
	wg.Wait()

	// 回收资源
	resp.release()

	return err
}

// 开启运行任务管理器
func (t *TaskManager) Start() {
	log.Infof("[TaskManager] goroutine count(%d) will start", t.concurrence)
	for i := 0; i < t.concurrence; i++ {
		go t.worker(i)
	}
}

// 回收任务管理的资源，销毁任务管理器
func (t *TaskManager) Release() {
	close(t.exitCh)
}

// 任务管理器的工作协程
func (t *TaskManager) worker(index int) {
	//log.Infof("[TaskManager] reading metadata worker(%d) running", index)
	defer log.Infof("[TaskManager] reading metadata worker(%d) exit", index)

	for {
		select {
		case future := <-t.recvCh:
			// 收到一个任务，执行任务设置的handler函数
			err := future.handler(future.item)
			// 处理回复，保证每个回复都要发送
			future.resp.reply(err)
		case <-t.exitCh:
			return
		}
	}
}
