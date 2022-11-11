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
	"container/list"
	"sync"
	"time"
)

// a simple routine-safe timewheel, can only add task
// not support update/delete

// TimeWheel 时间轮结构体
type TimeWheel struct {
	name       string
	interval   time.Duration
	ticker     *time.Ticker
	currentPos int
	slots      []*list.List
	locks      []sync.Mutex
	slotNum    int
	stopCh     chan struct{}
	wg         sync.WaitGroup
	opts       options
}

type options struct {
	waitTaskOnClose bool
}

type Option func(*options)

func WithWaitTaskOnClose(waitTaskOnClose bool) Option {
	return func(o *options) {
		o.waitTaskOnClose = waitTaskOnClose
	}
}

// Callback 时间轮回调函数定义
type Callback func(interface{})

// Task 时间轮任务结构体
type Task struct {
	delayTime time.Duration
	circle    int
	callback  Callback
	taskData  interface{}
}

// New 初始化时间轮
func New(interval time.Duration, slotNum int, name string, opts ...Option) *TimeWheel {
	if interval <= 0 || slotNum <= 0 {
		return nil
	}
	op := options{
		waitTaskOnClose: true,
	}

	for _, option := range opts {
		option(&op)
	}

	timeWheel := &TimeWheel{
		name:       name,
		interval:   interval,
		slots:      make([]*list.List, slotNum),
		locks:      make([]sync.Mutex, slotNum),
		currentPos: -1,
		slotNum:    slotNum,
		stopCh:     make(chan struct{}, 1),
		opts:       op,
	}

	for i := 0; i < slotNum; i++ {
		timeWheel.slots[i] = list.New()
	}

	return timeWheel
}

// Start 启动时间轮
func (tw *TimeWheel) Start() {
	tw.ticker = time.NewTicker(tw.interval)
	go tw.start()
}

// Stop 停止时间轮
func (tw *TimeWheel) Stop() {
	close(tw.stopCh)
	if tw.opts.waitTaskOnClose {
		tw.wg.Wait()
	}
}

// start 时间轮运转函数
func (tw *TimeWheel) start() {
	for {
		select {
		case <-tw.ticker.C:
			tw.taskRunner()
		case <-tw.stopCh:
			tw.ticker.Stop()
			return
		}
	}
}

// taskRunner 时间轮到期处理函数
func (tw *TimeWheel) taskRunner() {
	tw.currentPos++
	if tw.currentPos == tw.slotNum {
		tw.currentPos = 0
	}

	l := tw.slots[tw.currentPos]
	tw.locks[tw.currentPos].Lock()
	// execNum := tw.scanAddRunTask(l)
	_ = tw.scanAddRunTask(l)
	tw.locks[tw.currentPos].Unlock()

	// log.Debugf("%s task start time:%d, use time:%v, exec num:%d", tw.name, now.Unix(), time.Since(now), execNum)
}

// AddTask 新增时间轮任务
func (tw *TimeWheel) AddTask(delayMilli uint32, data interface{}, cb Callback) {
	delayTime := time.Duration(delayMilli) * time.Millisecond
	task := &Task{delayTime: delayTime, taskData: data, callback: cb}
	pos, circle := tw.getSlots(task.delayTime)
	task.circle = circle

	tw.locks[pos].Lock()
	tw.slots[pos].PushBack(task)
	tw.locks[pos].Unlock()
}

// scanAddRunTask 运行时间轮任务
func (tw *TimeWheel) scanAddRunTask(l *list.List) int {
	if l == nil || l.Len() == 0 {
		return 0
	}

	execNum := l.Len()
	for item := l.Front(); item != nil; {
		task := item.Value.(*Task)

		if task.circle > 0 {
			task.circle--
			item = item.Next()
			continue
		}

		go func() {
			if tw.opts.waitTaskOnClose {
				tw.wg.Add(1)
				defer tw.wg.Done()
			}
			task.callback(task.taskData)
		}()
		next := item.Next()
		l.Remove(item)
		item = next
	}

	return execNum
}

// getSlots 获取当前时间轮位置
func (tw *TimeWheel) getSlots(d time.Duration) (pos int, circle int) {
	delayTime := int(d.Seconds())
	interval := int(tw.interval.Seconds())
	circle = delayTime / interval / tw.slotNum
	pos = tw.currentPos
	if pos == -1 {
		pos = 0
	}
	pos = (pos + delayTime/interval) % tw.slotNum
	if pos == tw.currentPos {
		circle--
	}
	return
}
