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
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

// TestRetry 测试retry
func TestRetry(t *testing.T) {
	Convey("重试可以成功", t, func() {
		var err error
		Retry("retry", func() error {
			err = errors.New("retry error")
			return err
		})
		So(err, ShouldNotBeNil)

		start := time.Now()
		count := 0
		Retry("retry", func() error {
			count++
			if count <= 10 {
				err = errors.New("invalid connection")
				return err
			}
			err = nil
			return nil
		})
		sub := time.Since(start)
		So(err, ShouldBeNil)
		So(sub, ShouldBeGreaterThan, time.Millisecond*100)
	})
	Convey("只捕获固定的错误", t, func() {
		for _, msg := range errMsg {
			var err error
			start := time.Now()
			Retry(fmt.Sprintf("retry-%s", msg), func() error {
				err = fmt.Errorf("my-error: %s", msg)
				return err
			})
			So(err, ShouldNotBeNil)
			So(time.Since(start), ShouldBeGreaterThan, time.Millisecond*100)
		}
	})
}

// TestRetryTransaction 测试retryTransaction
func TestRetryTransaction(t *testing.T) {
	Convey("handle错误可以正常捕获", t, func() {
		err := RetryTransaction("test-handle", func() error {
			t.Logf("handle ok")
			return nil
		})
		So(err, ShouldBeNil)

		start := time.Now()
		err = RetryTransaction("test-handle", func() error {
			return errors.New("Deadlock")
		})
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "Deadlock")
		sub := time.Since(start)
		t.Logf("%v", sub)
		So(sub, ShouldBeGreaterThan, time.Millisecond*100)

		start = time.Now()
		err = RetryTransaction("test-handle", func() error {
			return errors.New("other error")
		})
		So(err, ShouldNotBeNil)
		sub = time.Since(start)
		So(sub, ShouldBeLessThan, time.Millisecond*5)
	})
}

// TestBatchOperation 测试BatchOperation
func TestBatchOperation(t *testing.T) {
	Convey("data为nil", t, func() {
		err := BatchOperation("data为nil", nil, func(objects []interface{}) error {
			return nil
		})
		So(err, ShouldBeNil)
	})
	Convey("data大小为1", t, func() {
		data := make([]interface{}, 1)
		num := 0
		err := BatchOperation("data为1", data, func(objects []interface{}) error {
			num++
			return nil
		})
		So(err, ShouldBeNil)
		So(num, ShouldEqual, 1)
	})
	Convey("data大小为101", t, func() {
		data := make([]interface{}, 101)
		num := 0
		err := BatchOperation("data为101", data, func(objects []interface{}) error {
			num++
			return nil
		})
		So(err, ShouldBeNil)
		So(num, ShouldEqual, 2)
	})

	Convey("data大小为100", t, func() {
		data := make([]interface{}, 100)
		num := 0
		err := BatchOperation("data为100", data, func(objects []interface{}) error {
			num++
			return nil
		})
		So(err, ShouldBeNil)
		So(num, ShouldEqual, 1)
	})

	Convey("data大小为0", t, func() {
		data := make([]interface{}, 0)
		num := 0
		err := BatchOperation("data为100", data, func(objects []interface{}) error {
			num++
			return nil
		})
		So(err, ShouldBeNil)
		So(num, ShouldEqual, 0)
	})
}
