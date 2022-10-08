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

package token

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/polarismesh/polaris/plugin"
)

// TestNewResourceRatelimit 测试新建
func TestNewResourceRatelimit(t *testing.T) {
	Convey("测试新建一个资源限制器", t, func() {
		Convey("config为空", func() {
			limiter, err := newResourceRatelimit(plugin.InstanceRatelimit, nil)
			So(limiter, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})
		Convey("不开启限制器", func() {
			limiter, err := newResourceRatelimit(plugin.InstanceRatelimit, &ResourceLimitConfig{
				Open: false,
			})
			So(limiter, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(limiter.allow("11111"), ShouldBeTrue)
		})
		Convey("开启了限制器，global为空", func() {
			limiter, err := newResourceRatelimit(plugin.InstanceRatelimit, &ResourceLimitConfig{
				Open: true,
			})
			So(limiter, ShouldBeNil)
			So(err, ShouldNotBeNil)
			t.Logf("%s", err.Error())
		})
		Convey("开启了限制器，global其他参数不合法", func() {
			limiter, err := newResourceRatelimit(plugin.InstanceRatelimit, &ResourceLimitConfig{
				Open:   true,
				Global: &BucketRatelimit{},
			})
			So(limiter, ShouldBeNil)
			So(err, ShouldNotBeNil)

			limiter, err = newResourceRatelimit(plugin.InstanceRatelimit, &ResourceLimitConfig{
				Open:   true,
				Global: &BucketRatelimit{true, 10, 10},
			})
			So(limiter, ShouldBeNil)
			So(err, ShouldNotBeNil)

			limiter, err = newResourceRatelimit(plugin.InstanceRatelimit, &ResourceLimitConfig{
				Open:                   true,
				Global:                 &BucketRatelimit{true, 10, 10},
				MaxResourceCacheAmount: -1,
			})
			So(limiter, ShouldBeNil)
			So(err, ShouldNotBeNil)
		})
		Convey("正常新建限制器", func() {
			limiter, err := newResourceRatelimit(plugin.InstanceRatelimit, &ResourceLimitConfig{
				Open:                   true,
				Global:                 &BucketRatelimit{true, 10, 5},
				MaxResourceCacheAmount: 10,
			})
			So(limiter, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})
		Convey("白名单正常解析", func() {
			limiter, err := newResourceRatelimit(plugin.InstanceRatelimit, &ResourceLimitConfig{
				Open:                   true,
				Global:                 &BucketRatelimit{true, 10, 5},
				MaxResourceCacheAmount: 10,
				WhiteList:              []string{"1", "2", "3"},
			})
			So(limiter, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(len(limiter.whiteList), ShouldEqual, 3)
		})
	})
}

// TestResourceAllow 测试allow
func TestResourceAllow(t *testing.T) {
	Convey("测试allow", t, func() {
		Convey("正常限流", func() {
			limiter, err := newResourceRatelimit(plugin.InstanceRatelimit, &ResourceLimitConfig{
				Open:                   true,
				Global:                 &BucketRatelimit{true, 5, 5},
				MaxResourceCacheAmount: 2,
			})
			So(err, ShouldBeNil)
			cnt := 0
			for i := 0; i <= limiter.config.Global.Rate*2; i++ {
				if ok := limiter.allow("12345"); ok {
					cnt++
				}
			}
			So(cnt, ShouldEqual, limiter.config.Global.Rate)
			// 其他key，可以通过
			So(limiter.allow("67890"), ShouldBeTrue)

			// 1秒之后，12345又可以通过
			time.Sleep(time.Second + time.Millisecond*10)
			So(limiter.allow("12345"), ShouldBeTrue)
			So(limiter.allow("67890"), ShouldBeTrue)
			So(limiter.allow("13579"), ShouldBeTrue)
		})
		Convey("max-resource测试", func() {
			limiter, err := newResourceRatelimit(plugin.InstanceRatelimit, &ResourceLimitConfig{
				Open:                   true,
				Global:                 &BucketRatelimit{true, 5, 5},
				MaxResourceCacheAmount: 2,
			})
			So(err, ShouldBeNil)
			cnt := 0
			for i := 0; i < limiter.config.Global.Rate*20; i++ {
				if ok := limiter.allow(fmt.Sprintf("key-%d", i)); ok {
					cnt++
				}
			}
			// 不同key，全部通过
			So(cnt, ShouldEqual, limiter.config.Global.Rate*20)
		})
		Convey("白名单测试", func() {
			limiter, err := newResourceRatelimit(plugin.InstanceRatelimit, &ResourceLimitConfig{
				Open:                   true,
				Global:                 &BucketRatelimit{true, 5, 5},
				MaxResourceCacheAmount: 1024,
				WhiteList:              []string{"1000", "1001", "1002"},
			})
			So(err, ShouldBeNil)

			cnt := 0
			for i := 0; i < limiter.config.Global.Rate*3; i++ {
				if ok := limiter.allow("1003"); ok {
					cnt++
				}
			}
			So(cnt, ShouldEqual, limiter.config.Global.Rate)

			cnt = 0
			for i := 0; i < limiter.config.Global.Rate*30; i++ {
				if ok := limiter.allow("1002"); ok {
					cnt++
				}
			}
			So(cnt, ShouldEqual, limiter.config.Global.Rate*30)
		})
	})
}
