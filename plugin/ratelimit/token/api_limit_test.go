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
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

// newAPIRateLimitCheck 校验新建apiRate
func newAPIRateLimitCheck(t *testing.T, config *APILimitConfig) *apiRatelimit {
	apiLimit, err := newAPIRatelimit(config)
	if err != nil {
		t.Fatalf("error: %s", err.Error())
	}
	return apiLimit
}

// TestAPIRateLimitAllow 正常场景测试
func TestAPIRateLimitAllow(t *testing.T) {
	config := &APILimitConfig{
		Open: true,
		Rules: []*RateLimitRule{
			{Name: "rule-a", Limit: &BucketRatelimit{true, 10, 2}},
			{Name: "rule-b", Limit: &BucketRatelimit{true, 10, 1}},
			{Name: "rule-c", Limit: &BucketRatelimit{Open: false}},
		},
		Apis: []*APILimitInfo{
			{Name: "api1", Rule: "rule-a"},
			{Name: "api2", Rule: "rule-b"},
			{Name: "api3", Rule: "rule-c"},
		},
	}
	limiter := newAPIRateLimitCheck(t, config)
	Convey("正常请求，令牌桶限流可以生效", t, func() {
		allowCnt := 0
		for i := 0; i < limiter.rules["rule-a"].Bucket*2; i++ {
			if ok := limiter.allow("api1"); ok {
				allowCnt++
			}
		}
		So(allowCnt, ShouldEqual, limiter.rules["rule-a"].Bucket)
	})
	Convey("持续请求，不超过限制，可以一直请求下去", t, func() {
		for i := 0; i < 15; i++ {
			So(limiter.allow("api2"), ShouldEqual, true)
			time.Sleep(time.Millisecond*1 + time.Second)
		}
	})
	Convey("api不限制，可以随便请求", t, func() {
		for i := 0; i < limiter.rules["rule-c"].Bucket*2; i++ {
			So(limiter.allow("api3"), ShouldEqual, true)
		}
	})
	Convey("api不存在rule，不做限制", t, func() {
		cnt := 0
		for i := 0; i < 10000; i++ {
			cnt++
		}
		So(cnt, ShouldEqual, 10000)
	})
}

// TestAPILimitConfig 配置校验
func TestAPILimitConfig(t *testing.T) {
	Convey("api-limit配置为空，可以正常执行", t, func() {
		limiter := newAPIRateLimitCheck(t, nil)
		So(limiter, ShouldNotBeNil)
		So(limiter.isOpen(), ShouldEqual, false)
	})
	Convey("可以通过系统open开关，关闭api限流", t, func() {
		config := &APILimitConfig{
			Open: false,
			Rules: []*RateLimitRule{
				{Name: "rule-1",
					Limit: &BucketRatelimit{Open: true, Bucket: 10, Rate: 5}},
			},
			Apis: []*APILimitInfo{{Name: "api-1", Rule: "rule-1"}},
		}
		limiter := newAPIRateLimitCheck(t, config)
		So(limiter.isOpen(), ShouldEqual, false)
		for i := 0; i < 15; i++ {
			So(limiter.allow("api-1"), ShouldEqual, true)
		}
	})
	Convey("rules为空，报错", t, func() {
		config := &APILimitConfig{Open: true}
		limiter, err := newAPIRatelimit(config)
		So(err, ShouldNotBeNil)
		So(limiter, ShouldBeNil)
	})
	Convey("api为空，报错", t, func() {
		config := &APILimitConfig{
			Open: true,
			Rules: []*RateLimitRule{
				{Name: "rule-1",
					Limit: &BucketRatelimit{Open: true, Bucket: 10, Rate: 5}},
			},
			Apis: []*APILimitInfo{},
		}
		_, err := newAPIRatelimit(config)
		So(err, ShouldNotBeNil)
	})
}

// TestAPILimitConfigRule api-limit规则配置测试
func TestAPILimitConfigRule(t *testing.T) {
	config := &APILimitConfig{
		Open: true,
		Apis: []*APILimitInfo{{Name: "api-1", Rule: "rule-1"}},
	}
	Convey("rules内部参数，name不能为空", t, func() {
		config.Rules = []*RateLimitRule{
			{Name: "",
				Limit: &BucketRatelimit{Open: true, Bucket: 0, Rate: 5}},
		}
		_, err := newAPIRatelimit(config)
		So(err, ShouldNotBeNil)
	})
	Convey("rules内部参数，limit不能为空", t, func() {
		config.Rules = []*RateLimitRule{
			{Name: "rule-1", Limit: nil},
		}
		_, err := newAPIRatelimit(config)
		So(err, ShouldNotBeNil)
	})
	Convey("rules内部参数，open为false，bucket和rate可以是任意值", t, func() {
		config.Rules = []*RateLimitRule{
			{Name: "",
				Limit: &BucketRatelimit{Open: false}},
		}
		_, err := newAPIRatelimit(config)
		So(err, ShouldNotBeNil)
	})
	Convey("rules内部参数，open的规则，bucket和rate必须大于0", t, func() {
		config.Rules = []*RateLimitRule{
			{Name: "rule-1",
				Limit: &BucketRatelimit{Open: true, Bucket: 0, Rate: 5}},
		}
		_, err := newAPIRatelimit(config)
		So(err, ShouldNotBeNil)
		t.Logf("%s", err.Error())

		config.Rules[0].Limit.Bucket = 10
		config.Rules[0].Limit.Rate = 0
		_, err = newAPIRatelimit(config)
		So(err, ShouldNotBeNil)
		t.Logf("%s", err.Error())
	})
}

// TestAPILimitConfigApi apis配置测试
func TestAPILimitConfigApi(t *testing.T) {
	config := &APILimitConfig{
		Open: true,
		Rules: []*RateLimitRule{
			{Name: "rule-1",
				Limit: &BucketRatelimit{Open: true, Bucket: 10, Rate: 5}},
		},
	}
	Convey("apis内部参数，apis为空，返回错误", t, func() {
		_, err := newAPIRatelimit(config)
		So(err, ShouldNotBeNil)
		t.Logf("%s", err.Error())
	})
	Convey("apis内部参数，部分参数为空，返回错误", t, func() {
		config.Apis = []*APILimitInfo{{Name: "", Rule: ""}}
		_, err := newAPIRatelimit(config)
		So(err, ShouldNotBeNil)
		t.Logf("%s", err.Error())

		config.Apis[0].Name = "123"
		_, err = newAPIRatelimit(config)
		So(err, ShouldNotBeNil)
		t.Logf("%s", err.Error())

		config.Apis[0].Rule = "rule-1"
		newAPIRateLimitCheck(t, config)
	})
	Convey("api内部参数，rule不存在，返回错误", t, func() {
		config.Apis = []*APILimitInfo{{Name: "aaa", Rule: "bbb"}}
		_, err := newAPIRatelimit(config)
		So(err, ShouldNotBeNil)
		t.Logf("%s", err.Error())

		config.Apis[0].Rule = "rule-1"
		newAPIRateLimitCheck(t, config)
	})
}
