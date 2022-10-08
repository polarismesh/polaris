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

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"

	api "github.com/polarismesh/polaris/common/api/v1"
	v2 "github.com/polarismesh/polaris/common/api/v2"
	"github.com/polarismesh/polaris/common/utils"
)

// 测试discover instances
func TestDiscoverInstances(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	Convey("服务发现测试", t, func() {
		_, service := discoverSuit.createCommonService(t, 5)
		defer discoverSuit.cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		count := 5
		var instances []*api.Instance
		var reqInstances []*api.Instance
		defer func() {
			for _, entry := range instances {
				discoverSuit.cleanInstance(entry.GetId().GetValue())
			}
		}()
		for i := 0; i < count; i++ {
			req, instance := discoverSuit.createCommonInstance(t, service, i)
			instances = append(instances, instance)
			reqInstances = append(reqInstances, req)
		}
		Convey("正常服务发现，返回的数据齐全", func() {
			time.Sleep(discoverSuit.updateCacheInterval)
			out := discoverSuit.server.ServiceInstancesCache(discoverSuit.defaultCtx, service)
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
			out := discoverSuit.server.ServiceInstancesCache(discoverSuit.defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			oldRevision := out.GetService().GetRevision().GetValue()

			service.Metadata = make(map[string]string)
			service.Metadata["new-metadata1"] = "1233"
			service.Metadata["new-metadata2"] = "2342"
			resp := discoverSuit.server.UpdateServices(discoverSuit.defaultCtx, []*api.Service{service})
			time.Sleep(discoverSuit.updateCacheInterval)
			So(respSuccess(resp), ShouldEqual, true)
			So(resp.Responses[0].GetService().GetRevision().GetValue(), ShouldNotEqual, oldRevision)
			So(resp.Responses[0].GetService().GetMetadata()["new-metadata1"], ShouldEqual, "1233")
			So(resp.Responses[0].GetService().GetMetadata()["new-metadata2"], ShouldEqual, "2342")
			serviceCheck(t, service, resp.Responses[0].GetService())
		})
	})
}

// 测试discover ratelimit
func TestDiscoverRateLimits(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	Convey("限流规则测试", t, func() {
		_, service := discoverSuit.createCommonService(t, 1)
		defer discoverSuit.cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		_, rateLimitResp := discoverSuit.createCommonRateLimit(t, service, 1)
		defer discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
		defer discoverSuit.cleanRateLimitRevision(service.GetName().GetValue(), service.GetNamespace().GetValue())
		Convey("正常获取限流规则", func() {
			time.Sleep(discoverSuit.updateCacheInterval)
			out := discoverSuit.server.GetRateLimitWithCache(discoverSuit.defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			So(len(out.GetRateLimit().GetRules()), ShouldEqual, 1)
			checkRateLimit(t, rateLimitResp, out.GetRateLimit().GetRules()[0])
			t.Logf("pass: out is %+v", out)
			// 再次请求
			out = discoverSuit.server.GetRateLimitWithCache(discoverSuit.defaultCtx, out.GetService())
			So(respSuccess(out), ShouldEqual, true)
			So(out.GetCode().GetValue(), ShouldEqual, api.DataNoChange)
			t.Logf("pass: out is %+v", out)
		})
		Convey("限流规则已删除", func() {
			discoverSuit.deleteRateLimit(t, rateLimitResp)
			time.Sleep(discoverSuit.updateCacheInterval)
			out := discoverSuit.server.GetRateLimitWithCache(discoverSuit.defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			So(len(out.GetRateLimit().GetRules()), ShouldEqual, 0)
			t.Logf("pass: out is %+v", out)
		})
	})
}

// 测试discover ratelimit
func TestDiscoverRateLimits2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	Convey("限流规则异常测试", t, func() {
		_, service := discoverSuit.createCommonService(t, 1)
		defer discoverSuit.cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		Convey("限流规则不存在", func() {
			time.Sleep(discoverSuit.updateCacheInterval)
			out := discoverSuit.server.GetRateLimitWithCache(discoverSuit.defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			So(out.GetRateLimit(), ShouldBeNil)
			t.Logf("pass: out is %+v", out)
		})
		Convey("服务不存在", func() {
			services := []*api.Service{service}
			discoverSuit.removeCommonServices(t, services)
			time.Sleep(discoverSuit.updateCacheInterval)
			out := discoverSuit.server.GetRateLimitWithCache(discoverSuit.defaultCtx, service)
			So(respSuccess(out), ShouldEqual, false)
			t.Logf("pass: out is %+v", out)
		})
	})
}

// 测试discover circuitbreaker
func TestDiscoverCircuitBreaker(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	Convey("熔断规则测试", t, func() {
		// 创建服务
		_, service := discoverSuit.createCommonService(t, 1)
		defer discoverSuit.cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		// 创建熔断规则
		_, cbResp := discoverSuit.createCommonCircuitBreaker(t, 1)
		defer discoverSuit.cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())
		// 创建熔断规则版本
		_, cbVersionResp := discoverSuit.createCommonCircuitBreakerVersion(t, cbResp, 1)
		defer discoverSuit.cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
		// 发布熔断规则
		discoverSuit.releaseCircuitBreaker(t, cbVersionResp, service)
		defer discoverSuit.cleanCircuitBreakerRelation(service.GetName().GetValue(), service.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		Convey("正常获取熔断规则", func() {
			time.Sleep(discoverSuit.updateCacheInterval)
			out := discoverSuit.server.GetCircuitBreakerWithCache(discoverSuit.defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			checkCircuitBreaker(t, cbVersionResp, cbResp, out.GetCircuitBreaker())
			t.Logf("pass: out is %+v", out)

			// 再次请求
			out = discoverSuit.server.GetCircuitBreakerWithCache(discoverSuit.defaultCtx, out.GetService())
			So(respSuccess(out), ShouldEqual, true)
			So(out.GetCode().GetValue(), ShouldEqual, api.DataNoChange)
			t.Logf("pass: out is %+v", out)
		})

		Convey("解绑熔断规则", func() {
			discoverSuit.unBindCircuitBreaker(t, cbVersionResp, service)
			time.Sleep(discoverSuit.updateCacheInterval)
			out := discoverSuit.server.GetCircuitBreakerWithCache(discoverSuit.defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			t.Logf("pass: out is %+v", out)
		})
	})
}

// 测试discover circuitbreaker
func TestDiscoverCircuitBreaker2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	Convey("熔断规则异常测试", t, func() {
		_, service := discoverSuit.createCommonService(t, 1)
		defer discoverSuit.cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		Convey("熔断规则不存在", func() {
			time.Sleep(discoverSuit.updateCacheInterval)
			out := discoverSuit.server.GetCircuitBreakerWithCache(discoverSuit.defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			So(out.GetCircuitBreaker(), ShouldBeNil)
			t.Logf("pass: out is %+v", out)
		})
		Convey("服务不存在", func() {
			services := []*api.Service{service}
			discoverSuit.removeCommonServices(t, services)
			time.Sleep(discoverSuit.updateCacheInterval)
			out := discoverSuit.server.GetCircuitBreakerWithCache(discoverSuit.defaultCtx, service)
			So(respSuccess(out), ShouldEqual, false)
			t.Logf("pass: out is %+v", out)
		})
	})
}

// 测试discover service
func TestDiscoverService(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	Convey("服务测试", t, func() {
		expectService1 := &api.Service{}
		expectService2 := &api.Service{}
		for id := 0; id < 5; id++ {
			_, service := discoverSuit.createCommonService(t, id)
			if id == 3 {
				expectService1 = service
			}
			if id == 4 {
				expectService2 = service
			}
			defer discoverSuit.cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
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
		_ = discoverSuit.server.UpdateServices(discoverSuit.defaultCtx, []*api.Service{expectService1})
		_ = discoverSuit.server.UpdateServices(discoverSuit.defaultCtx, []*api.Service{expectService1})
		time.Sleep(discoverSuit.updateCacheInterval)

		Convey("正常获取服务", func() {
			requestService := &api.Service{
				Metadata: requestMeta,
			}
			out := discoverSuit.server.GetServiceWithCache(discoverSuit.defaultCtx, requestService)
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
			out := discoverSuit.server.GetServiceWithCache(discoverSuit.defaultCtx, requestService)
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

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	Convey("服务正常测试", t, func() {
		Convey("元数据不存在", func() {
			service := &api.Service{}
			out := discoverSuit.server.GetServiceWithCache(discoverSuit.defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			t.Logf("pass: out is %+v", out)
		})
		Convey("元数据为空", func() {
			service := &api.Service{
				Metadata: make(map[string]string),
			}
			out := discoverSuit.server.GetServiceWithCache(discoverSuit.defaultCtx, service)
			So(respSuccess(out), ShouldEqual, true)
			t.Logf("pass: out is %+v", out)
		})
	})
}

func TestDiscoverServerV2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	svc := &api.Service{
		Name:      utils.NewStringValue("in-source-service-1"),
		Namespace: utils.NewStringValue("in-source-service-1"),
	}

	createSvcResp := discoverSuit.server.CreateServices(discoverSuit.defaultCtx, []*api.Service{svc})
	if !respSuccess(createSvcResp) {
		t.Fatalf("error: %s", createSvcResp.GetInfo().GetValue())
	}

	_ = discoverSuit.createCommonRoutingConfigV2(t, 3)
	defer discoverSuit.truncateCommonRoutingConfigV2()

	time.Sleep(discoverSuit.updateCacheInterval * 5)

	t.Run("空请求", func(t *testing.T) {
		resp := discoverSuit.server.GetRoutingConfigV2WithCache(context.Background(), nil)

		assert.Equal(t, api.EmptyRequest, resp.Code)
	})

	t.Run("没有带服务名", func(t *testing.T) {
		resp := discoverSuit.server.GetRoutingConfigV2WithCache(context.Background(), &v2.Service{
			Namespace: "string",
		})

		assert.Equal(t, api.InvalidServiceName, resp.Code)
	})

	t.Run("没有带命名空间", func(t *testing.T) {
		resp := discoverSuit.server.GetRoutingConfigV2WithCache(context.Background(), &v2.Service{
			Name: "string",
		})

		assert.Equal(t, api.InvalidNamespaceName, resp.Code)
	})

	t.Run("查询v2版本的路由规则", func(t *testing.T) {
		resp := discoverSuit.server.GetRoutingConfigV2WithCache(context.Background(), &v2.Service{
			Name:      "in-source-service-1",
			Namespace: "in-source-service-1",
		})

		if !respSuccessV2(resp) {
			t.Fatal(resp.GetInfo())
		}
	})
}
