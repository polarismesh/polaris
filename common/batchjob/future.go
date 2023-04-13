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
	"sync/atomic"
)

type Task interface{}

type Future interface {
	TaskInfo() Task
	Done() (interface{}, error)
	Cancel()
	Reply(result interface{}, err error)
}

type future struct {
	task      Task
	err       error
	setsignal chan struct{}
	result    interface{}
	isCancel  int32
	replied   int32
	ctx       context.Context
	cancel    context.CancelFunc
}

func (f *future) TaskInfo() Task {
	return f.task
}

func (f *future) Done() (interface{}, error) {
	select {
	case <-f.ctx.Done():
		return nil, f.ctx.Err()
	case <-f.setsignal:
		return f.result, f.err
	}
}

func (f *future) Cancel() {
	f.cancel()
}

func (f *future) Reply(result interface{}, err error) {
	if !atomic.CompareAndSwapInt32(&f.replied, 0, 1) {
		return
	}
	f.result = result
	f.err = err
	f.setsignal <- struct{}{}
	close(f.setsignal)
}
