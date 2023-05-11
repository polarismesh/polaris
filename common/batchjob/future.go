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
	"sync/atomic"
	"time"
)

type Param interface{}

type Future interface {
	Param() Param
	Done() (interface{}, error)
	DoneTimeout(timeout time.Duration) (interface{}, error)
	Cancel()
	Reply(result interface{}, err error) error
}

type errorFuture struct {
	task Param
	err  error
}

func (f *errorFuture) Param() Param {
	return f.task
}

func (f *errorFuture) Done() (interface{}, error) {
	return nil, f.err
}

func (f *errorFuture) DoneTimeout(timeout time.Duration) (interface{}, error) {
	return nil, f.err
}

func (f *errorFuture) Cancel() {
}

func (f *errorFuture) Reply(result interface{}, err error) error {
	return nil
}

type future struct {
	task      Param
	setsignal chan struct{}
	err       error
	result    interface{}
	replied   int32
	closed    int32
	ctx       context.Context
	cancel    context.CancelFunc
}

func (f *future) Param() Param {
	return f.task
}

func (f *future) Done() (interface{}, error) {
	defer func() {
		if atomic.CompareAndSwapInt32(&f.closed, 0, 1) {
			close(f.setsignal)
		}
		f.cancel()
	}()
	select {
	case <-f.ctx.Done():
		return nil, f.ctx.Err()
	case <-f.setsignal:
		return f.result, f.err
	}
}

func (f *future) DoneTimeout(timeout time.Duration) (interface{}, error) {
	timer := time.NewTimer(timeout)
	defer func() {
		if atomic.CompareAndSwapInt32(&f.closed, 0, 1) {
			close(f.setsignal)
		}
		timer.Stop()
		f.cancel()
	}()
	select {
	case <-timer.C:
		return nil, context.DeadlineExceeded
	case <-f.ctx.Done():
		return nil, f.ctx.Err()
	case <-f.setsignal:
		return f.result, f.err
	}
}

func (f *future) Cancel() {
	if !atomic.CompareAndSwapInt32(&f.replied, 0, 1) {
		return
	}
	f.cancel()
}

var (
	ErrorReplyOnlyOnce = errors.New("reply only call once")
	ErrorReplyCanceled = errors.New("reply canceled")
)

func (f *future) Reply(result interface{}, err error) error {
	if !atomic.CompareAndSwapInt32(&f.replied, 0, 1) {
		return ErrorReplyOnlyOnce
	}
	f.result = result
	f.err = err
	f.setsignal <- struct{}{}
	return nil
}
