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

	. "github.com/smartystreets/goconvey/convey"

	"github.com/polarismesh/polaris/plugin"
)

// baseConfigOption 返回一个基础的正常option配置
func baseConfigOption() map[string]interface{} {
	return map[string]interface{}{
		"enable": true,
		"ip-limit": &ResourceLimitConfig{
			Open:                   true,
			Global:                 &BucketRatelimit{true, 10, 2},
			MaxResourceCacheAmount: 100,
		},
		"api-limit": &APILimitConfig{
			Open: true,
			Rules: []*RateLimitRule{{
				Name:  "rule-1",
				Limit: &BucketRatelimit{true, 5, 1},
			}},
			Apis: []*APILimitInfo{{Name: "api-1", Rule: "rule-1"}},
		},
	}
}

// TestTokenBucket_Name 对插件名字接口实现的测试
func TestTokenBucket_Name(t *testing.T) {
	tb := &tokenBucket{}
	Convey("返回插件名服务预期", t, func() {
		So(tb.Name(), ShouldEqual, PluginName)
	})
}

// TestTokenBucket_Initialize 测试初始化函数
func TestTokenBucket_Initialize(t *testing.T) {
	configEntry := &plugin.ConfigEntry{Name: PluginName}
	tb := &tokenBucket{}
	Convey("配置option为空，返回失败", t, func() {
		So(tb.Initialize(configEntry), ShouldNotBeNil)
	})
	Convey("配置字段不对，不影响解析，配置不生效", t, func() {
		configEntry.Option = map[string]interface{}{"aaa": 123}
		So(tb.Initialize(configEntry), ShouldBeNil)
	})
	Convey("无效ip-limit配置，返回失败", t, func() {
		configEntry.Option = map[string]interface{}{
			"ip-limit": &ResourceLimitConfig{
				Open:                   true,
				Global:                 nil,
				MaxResourceCacheAmount: 100,
			},
			"enable": true,
		}
		So(tb.Initialize(configEntry), ShouldNotBeNil)
	})
	Convey("无效api-limit配置，返回失败", t, func() {
		configEntry.Option = map[string]interface{}{
			"enable":    true,
			"api-limit": &APILimitConfig{Open: true},
		}
		So(tb.Initialize(configEntry), ShouldNotBeNil)
	})
	Convey("配置有效，可以正常初始化", t, func() {
		configEntry.Option = baseConfigOption()
		So(tb.Initialize(configEntry), ShouldBeNil)
		So(tb.limiters[plugin.IPRatelimit], ShouldNotBeNil)
		So(tb.limiters[plugin.APIRatelimit], ShouldNotBeNil)
	})
}

// TestTokenBucket_Allow 测试Allow函数
func TestTokenBucket_Allow(t *testing.T) {
	configEntry := &plugin.ConfigEntry{Name: PluginName}
	configEntry.Option = baseConfigOption()
	tb := &tokenBucket{}
	if err := tb.Initialize(configEntry); err != nil {
		t.Fatalf("error: %s", err.Error())
	}
	ipLimiter := tb.limiters[plugin.IPRatelimit].(*resourceRatelimit)
	apiLimiter := tb.limiters[plugin.APIRatelimit].(*apiRatelimit)
	Convey("IP正常限流", t, func() {
		cnt := 0
		for i := 0; i < ipLimiter.config.Global.Bucket*2; i++ {
			if ok := tb.Allow(plugin.IPRatelimit, "1.2.3.4"); ok {
				cnt++
			}
		}
		So(cnt, ShouldEqual, ipLimiter.config.Global.Bucket)
		// 其他IP可以正常通过
		So(tb.Allow(plugin.IPRatelimit, "2.3.4.5"), ShouldEqual, true)
	})
	Convey("api正常限流", t, func() {
		cnt := 0
		for i := 0; i < apiLimiter.rules["rule-1"].Bucket*2; i++ {
			if ok := tb.Allow(plugin.APIRatelimit, "api-1"); ok {
				cnt++
			}
		}
		So(cnt, ShouldEqual, apiLimiter.rules["rule-1"].Bucket)
		// 其他接口没有限流的可以通过
		So(tb.Allow(plugin.APIRatelimit, "api-2"), ShouldEqual, true)
	})
	Convey("空的key，正常限流", t, func() {
		So(tb.Allow(plugin.APIRatelimit, ""), ShouldEqual, true)
	})
	Convey("非法的限制类型，直接通过", t, func() {
		So(tb.Allow(plugin.RatelimitType(100), "123"), ShouldEqual, true)
	})
}
