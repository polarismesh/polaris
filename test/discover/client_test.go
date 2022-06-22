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

package discover

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
)

// 测试client版本上报
func TestReportClient(t *testing.T) {
	Convey("可以进行正常的client上报", t, func() {
		req := &api.Client{
			Host:    utils.NewStringValue("127.0.0.1"),
			Type:    api.Client_SDK,
			Version: utils.NewStringValue("v1.0.0"),
		}
		resp := server.ReportClient(defaultCtx, req)
		So(respSuccess(resp), ShouldEqual, true)
	})
}

// 测试discover instances
func TestDiscoverInstances(t *testing.T) {
	Convey("服务发现测试", t, func() {
		_, service := createCommonService(t, 5)
		defer cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		count := 5
		var instances []*api.Instance
		var reqInstances []*api.Instance
		defer func() {
			for _, entry := range instances {
				cleanInstance(entry.GetId().GetValue())
			}
		}()
		for i := 0; i < count; i++ {
			req, instance := createCommonInstance(t, service, i)
			instances = append(instances, instance)
			reqInstances = append(reqInstances, req)
		}
		Convey("正常服务发现，返回的数据齐全", func() {
			time.Sleep(updateCacheInterval)
			out := server.ServiceInstancesCache(defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			So(len(out.GetInstances()), ShouldEqual, count)
			for _, resp := range out.GetInstances() {
				found := false
				for _, req := range reqInstances {
					if resp.GetHost().GetValue() == req.GetHost().GetValue() {
						instanceCheck(t, req, resp) // expect actual
						// 检查resp中必须包含额外的metadata
						So(resp.Metadata["version"], ShouldEqual, req.GetVersion().GetValue())
						So(resp.Metadata["protocol"], ShouldEqual, req.GetProtocol().GetValue())
						found = true
						t.Logf("%+v", resp)
						break
					}
				}
				So(found, ShouldEqual, true)
			}
		})
		Convey("service-metadata修改，revision会修改", func() {
			out := server.ServiceInstancesCache(defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			oldRevision := out.GetService().GetRevision().GetValue()

			service.Metadata = make(map[string]string)
			service.Metadata["new-metadata1"] = "1233"
			service.Metadata["new-metadata2"] = "2342"
			resp := server.UpdateService(defaultCtx, service)
			time.Sleep(updateCacheInterval)
			So(respSuccess(resp), ShouldEqual, true)
			So(resp.GetService().GetRevision().GetValue(), ShouldNotEqual, oldRevision)
			So(resp.GetService().GetMetadata()["new-metadata1"], ShouldEqual, "1233")
			So(resp.GetService().GetMetadata()["new-metadata2"], ShouldEqual, "2342")
			serviceCheck(t, service, resp.GetService())
		})
	})
}

// 测试discover ratelimit
func TestDiscoverRateLimits(t *testing.T) {
	Convey("限流规则测试", t, func() {
		_, service := createCommonService(t, 1)
		defer cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		_, rateLimitResp := createCommonRateLimit(t, service, 1)
		defer cleanRateLimit(rateLimitResp.GetId().GetValue())
		defer cleanRateLimitRevision(service.GetName().GetValue(), service.GetNamespace().GetValue())
		Convey("正常获取限流规则", func() {
			time.Sleep(updateCacheInterval)
			out := server.GetRateLimitWithCache(defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			So(len(out.GetRateLimit().GetRules()), ShouldEqual, 1)
			checkRateLimit(t, rateLimitResp, out.GetRateLimit().GetRules()[0])
			t.Logf("pass: out is %+v", out)
			// 再次请求
			out = server.GetRateLimitWithCache(defaultCtx, out.GetService())
			So(respSuccess(out), ShouldEqual, true)
			So(out.GetCode().GetValue(), ShouldEqual, api.DataNoChange)
			t.Logf("pass: out is %+v", out)
		})
		Convey("限流规则已删除", func() {
			deleteRateLimit(t, rateLimitResp)
			time.Sleep(updateCacheInterval)
			out := server.GetRateLimitWithCache(defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			So(len(out.GetRateLimit().GetRules()), ShouldEqual, 0)
			t.Logf("pass: out is %+v", out)
		})
	})
}

// 测试discover ratelimit
func TestDiscoverRateLimits2(t *testing.T) {
	Convey("限流规则异常测试", t, func() {
		_, service := createCommonService(t, 1)
		defer cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		Convey("限流规则不存在", func() {
			time.Sleep(updateCacheInterval)
			out := server.GetRateLimitWithCache(defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			So(out.GetRateLimit(), ShouldBeNil)
			t.Logf("pass: out is %+v", out)
		})
		Convey("服务不存在", func() {
			services := []*api.Service{service}
			removeCommonServices(t, services)
			time.Sleep(updateCacheInterval)
			out := server.GetRateLimitWithCache(defaultCtx, service)
			So(respSuccess(out), ShouldEqual, false)
			t.Logf("pass: out is %+v", out)
		})
	})
}

// 测试discover circuitbreaker
func TestDiscoverCircuitBreaker(t *testing.T) {
	Convey("熔断规则测试", t, func() {
		// 创建服务
		_, service := createCommonService(t, 1)
		defer cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		// 创建熔断规则
		_, cbResp := createCommonCircuitBreaker(t, 1)
		defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())
		// 创建熔断规则版本
		_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 1)
		defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
		// 发布熔断规则
		releaseCircuitBreaker(t, cbVersionResp, service)
		defer cleanCircuitBreakerRelation(service.GetName().GetValue(), service.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		Convey("正常获取熔断规则", func() {
			time.Sleep(updateCacheInterval)
			out := server.GetCircuitBreakerWithCache(defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			checkCircuitBreaker(t, cbVersionResp, cbResp, out.GetCircuitBreaker())
			t.Logf("pass: out is %+v", out)

			// 再次请求
			out = server.GetCircuitBreakerWithCache(defaultCtx, out.GetService())
			So(respSuccess(out), ShouldEqual, true)
			So(out.GetCode().GetValue(), ShouldEqual, api.DataNoChange)
			t.Logf("pass: out is %+v", out)
		})

		Convey("解绑熔断规则", func() {
			unBindCircuitBreaker(t, cbVersionResp, service)
			time.Sleep(updateCacheInterval)
			out := server.GetCircuitBreakerWithCache(defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			t.Logf("pass: out is %+v", out)
		})
	})
}

// 测试discover circuitbreaker
func TestDiscoverCircuitBreaker2(t *testing.T) {
	Convey("熔断规则异常测试", t, func() {
		_, service := createCommonService(t, 1)
		defer cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		Convey("熔断规则不存在", func() {
			time.Sleep(updateCacheInterval)
			out := server.GetCircuitBreakerWithCache(defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			So(out.GetCircuitBreaker(), ShouldBeNil)
			t.Logf("pass: out is %+v", out)
		})
		Convey("服务不存在", func() {
			services := []*api.Service{service}
			removeCommonServices(t, services)
			time.Sleep(updateCacheInterval)
			out := server.GetCircuitBreakerWithCache(defaultCtx, service)
			So(respSuccess(out), ShouldEqual, false)
			t.Logf("pass: out is %+v", out)
		})
	})
}

// 测试discover service
func TestDiscoverService(t *testing.T) {
	Convey("服务测试", t, func() {
		expectService1 := &api.Service{}
		expectService2 := &api.Service{}
		for id := 0; id < 5; id++ {
			_, service := createCommonService(t, id)
			if id == 3 {
				expectService1 = service
			}
			if id == 4 {
				expectService2 = service
			}
			defer cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		}

		meta := make(map[string]string)
		requestMeta := make(map[string]string)
		for i := 0; i < 5; i++ {
			k := fmt.Sprintf("key-%d", i)
			v := fmt.Sprintf("value-%d", i)
			if i == 0 || i == 1 {
				requestMeta[k] = v
			}
			meta[k] = v
		}

		expectService1.Metadata = meta
		expectService2.Metadata = meta
		_ = server.UpdateService(defaultCtx, expectService1)
		_ = server.UpdateService(defaultCtx, expectService2)
		time.Sleep(updateCacheInterval)

		Convey("正常获取服务", func() {
			requestService := &api.Service{
				Metadata: requestMeta,
			}
			out := server.GetServiceWithCache(defaultCtx, requestService)
			So(respSuccess(out), ShouldEqual, true)
			if len(out.GetServices()) == 2 {
				t.Logf("pass: out service is %+v", out.GetServices())
			} else {
				t.Logf("error: out is %+v", out)
			}
		})

		Convey("元数据匹配到的服务为空", func() {
			requestMeta := make(map[string]string)
			requestMeta["test"] = "test"
			requestService := &api.Service{
				Metadata: requestMeta,
			}
			out := server.GetServiceWithCache(defaultCtx, requestService)
			So(respSuccess(out), ShouldEqual, true)
			if len(out.GetServices()) == 0 {
				t.Logf("pass: out service is %+v", out.GetServices())
			} else {
				t.Logf("error: out is %+v", out)
			}
		})
	})
}

// 测试discover service
func TestDiscoverService2(t *testing.T) {
	Convey("服务异常测试", t, func() {
		Convey("元数据不存在", func() {
			service := &api.Service{}
			out := server.GetServiceWithCache(defaultCtx, service)
			So(respSuccess(out), ShouldEqual, false)
			t.Logf("pass: out is %+v", out)
		})
		Convey("元数据为空", func() {
			service := &api.Service{
				Metadata: make(map[string]string),
			}
			out := server.GetServiceWithCache(defaultCtx, service)
			So(respSuccess(out), ShouldEqual, false)
			t.Logf("pass: out is %+v", out)
		})
	})
}
