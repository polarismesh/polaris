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

package service_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/smartystreets/goconvey/convey"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
)

// 测试discover instances
func TestDiscoverInstances(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	convey.Convey("服务发现测试", t, func() {
		_, service := discoverSuit.createCommonService(t, 5)
		defer discoverSuit.cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		count := 5
		var instances []*apiservice.Instance
		var reqInstances []*apiservice.Instance
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
		convey.Convey("正常服务发现，返回的数据齐全", func() {
			time.Sleep(discoverSuit.UpdateCacheInterval())
			out := discoverSuit.DiscoverServer().ServiceInstancesCache(discoverSuit.DefaultCtx, service)
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			convey.So(len(out.GetInstances()), convey.ShouldEqual, count)
			for _, resp := range out.GetInstances() {
				found := false
				for _, req := range reqInstances {
					if resp.GetHost().GetValue() == req.GetHost().GetValue() {
						instanceCheck(t, req, resp) // expect actual
						// 检查resp中必须包含额外的metadata
						convey.So(resp.Metadata["version"], convey.ShouldEqual, req.GetVersion().GetValue())
						convey.So(resp.Metadata["protocol"], convey.ShouldEqual, req.GetProtocol().GetValue())
						found = true
						t.Logf("%+v", resp)
						break
					}
				}
				convey.So(found, convey.ShouldEqual, true)
			}
		})
		convey.Convey("service-metadata修改，revision会修改", func() {
			out := discoverSuit.DiscoverServer().ServiceInstancesCache(discoverSuit.DefaultCtx, service)
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			oldRevision := out.GetService().GetRevision().GetValue()

			service.Metadata = make(map[string]string)
			service.Metadata["new-metadata1"] = "1233"
			service.Metadata["new-metadata2"] = "2342"
			resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
			time.Sleep(discoverSuit.UpdateCacheInterval())
			convey.So(respSuccess(resp), convey.ShouldEqual, true)
			convey.So(resp.Responses[0].GetService().GetRevision().GetValue(), convey.ShouldNotEqual, oldRevision)
			convey.So(resp.Responses[0].GetService().GetMetadata()["new-metadata1"], convey.ShouldEqual, "1233")
			convey.So(resp.Responses[0].GetService().GetMetadata()["new-metadata2"], convey.ShouldEqual, "2342")
			serviceCheck(t, service, resp.Responses[0].GetService())
		})
	})
}

// 测试discover instances
func TestDiscoverInstancesById(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	convey.Convey("服务发现Id测试", t, func() {
		_, svc := discoverSuit.createCommonService(t, 1)
		defer discoverSuit.cleanServiceName(svc.GetName().GetValue(), svc.GetNamespace().GetValue())
		var instances []*apiservice.Instance
		var reqInstances []*apiservice.Instance
		defer func() {
			for _, entry := range instances {
				discoverSuit.cleanInstance(entry.GetId().GetValue())
			}
		}()

		idPrefix := "prefix-"
		prefixCount := 5
		idSuffix := "-suffix"
		suffixCount := 3
		for i := 0; i < prefixCount; i++ {
			req, instance := discoverSuit.createCommonInstanceById(
				t, svc, i, fmt.Sprintf("%s%d", idPrefix, i))
			instances = append(instances, instance)
			reqInstances = append(reqInstances, req)
		}
		for i := 0; i < suffixCount; i++ {
			req, instance := discoverSuit.createCommonInstanceById(
				t, svc, i, fmt.Sprintf("%d%s", i, idSuffix))
			instances = append(instances, instance)
			reqInstances = append(reqInstances, req)
		}
		time.Sleep(discoverSuit.UpdateCacheInterval())
		convey.Convey("根据精准匹配ID进行获取实例", func() {
			instId := fmt.Sprintf("%s%d", idPrefix, 0)
			ctx := context.WithValue(discoverSuit.DefaultCtx, service.ContextDiscoverParam,
				map[string]string{service.ParamKeyInstanceId: instId})
			out := discoverSuit.DiscoverServer().ServiceInstancesCache(ctx, nil)
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			convey.So(len(out.GetInstances()), convey.ShouldEqual, 1)
			instance := out.GetInstances()[0]
			convey.So(instance.GetId().GetValue(), convey.ShouldEqual, instId)
			convey.So(instance.GetNamespace().GetValue(), convey.ShouldEqual, svc.GetNamespace().GetValue())
			convey.So(instance.GetService().GetValue(), convey.ShouldEqual, svc.GetName().GetValue())
		})
		convey.Convey("根据前缀匹配ID进行获取实例", func() {
			instId := fmt.Sprintf("%s%s", idPrefix, "*")
			ctx := context.WithValue(discoverSuit.DefaultCtx, service.ContextDiscoverParam,
				map[string]string{service.ParamKeyInstanceId: instId})
			out := discoverSuit.DiscoverServer().ServiceInstancesCache(ctx, nil)
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			convey.So(len(out.GetInstances()), convey.ShouldEqual, prefixCount)
			for _, instance := range out.GetInstances() {
				convey.So(strings.HasPrefix(instance.GetId().GetValue(), idPrefix), convey.ShouldEqual, true)
				convey.So(instance.GetNamespace().GetValue(), convey.ShouldEqual, svc.GetNamespace().GetValue())
				convey.So(instance.GetService().GetValue(), convey.ShouldEqual, svc.GetName().GetValue())
			}
		})
		convey.Convey("根据后缀匹配ID进行获取实例", func() {
			instId := fmt.Sprintf("%s%s", "*", idSuffix)
			ctx := context.WithValue(discoverSuit.DefaultCtx, service.ContextDiscoverParam,
				map[string]string{service.ParamKeyInstanceId: instId})
			out := discoverSuit.DiscoverServer().ServiceInstancesCache(ctx, nil)
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			convey.So(len(out.GetInstances()), convey.ShouldEqual, suffixCount)
			for _, instance := range out.GetInstances() {
				convey.So(strings.HasSuffix(instance.GetId().GetValue(), idSuffix), convey.ShouldEqual, true)
				convey.So(instance.GetNamespace().GetValue(), convey.ShouldEqual, svc.GetNamespace().GetValue())
				convey.So(instance.GetService().GetValue(), convey.ShouldEqual, svc.GetName().GetValue())
			}
		})
	})
}

// 测试discover circuitbreaker
func TestDiscoverCircuitBreaker(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	convey.Convey("熔断规则测试", t, func() {
		rules, resp := createCircuitBreakerRules(discoverSuit, 5)
		defer cleanCircuitBreakerRules(discoverSuit, resp)
		service := &apiservice.Service{Name: utils.NewStringValue("testDestService"), Namespace: utils.NewStringValue("test")}
		convey.Convey("正常获取熔断规则", func() {
			time.Sleep(discoverSuit.UpdateCacheInterval())
			out := discoverSuit.DiscoverServer().GetCircuitBreakerWithCache(discoverSuit.DefaultCtx, service)
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			convey.So(len(out.GetCircuitBreaker().GetRules()), convey.ShouldEqual, len(rules))
			t.Logf("pass: out is %+v", out)

			// 再次请求
			out = discoverSuit.DiscoverServer().GetCircuitBreakerWithCache(discoverSuit.DefaultCtx, out.GetService())
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			convey.So(out.GetCode().GetValue(), convey.ShouldEqual, api.DataNoChange)
			t.Logf("pass: out is %+v", out)
		})
	})
}

// 测试discover circuitbreaker
func TestDiscoverCircuitBreaker2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	convey.Convey("熔断规则异常测试", t, func() {
		_, resp := createCircuitBreakerRules(discoverSuit, 1)
		defer cleanCircuitBreakerRules(discoverSuit, resp)
		service := &apiservice.Service{Name: utils.NewStringValue("testDestService"), Namespace: utils.NewStringValue("default")}
		convey.Convey("熔断规则不存在", func() {
			time.Sleep(discoverSuit.UpdateCacheInterval())
			out := discoverSuit.DiscoverServer().GetCircuitBreakerWithCache(discoverSuit.DefaultCtx, service)
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			convey.So(len(out.GetCircuitBreaker().GetRules()), convey.ShouldEqual, 0)
			t.Logf("pass: out is %+v", out)
		})
	})
}

// 测试discover service
func TestDiscoverService(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	convey.Convey("服务测试", t, func() {
		expectService1 := &apiservice.Service{}
		expectService2 := &apiservice.Service{}
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
		_ = discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{expectService1})
		_ = discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{expectService1})
		time.Sleep(discoverSuit.UpdateCacheInterval())

		convey.Convey("正常获取服务", func() {
			requestService := &apiservice.Service{
				Metadata: requestMeta,
			}
			out := discoverSuit.DiscoverServer().GetServiceWithCache(discoverSuit.DefaultCtx, requestService)
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			if len(out.GetServices()) == 2 {
				t.Logf("pass: out service is %+v", out.GetServices())
			} else {
				t.Logf("error: out is %+v", out)
			}
		})

		convey.Convey("元数据匹配到的服务为空", func() {
			requestMeta := make(map[string]string)
			requestMeta["test"] = "test"
			requestService := &apiservice.Service{
				Metadata: requestMeta,
			}
			out := discoverSuit.DiscoverServer().GetServiceWithCache(discoverSuit.DefaultCtx, requestService)
			convey.So(respSuccess(out), convey.ShouldEqual, true)
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
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	convey.Convey("服务正常测试", t, func() {
		convey.Convey("元数据不存在", func() {
			service := &apiservice.Service{}
			out := discoverSuit.DiscoverServer().GetServiceWithCache(discoverSuit.DefaultCtx, service)
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			t.Logf("pass: out is %+v", out)
		})
		convey.Convey("元数据为空", func() {
			service := &apiservice.Service{
				Metadata: make(map[string]string),
			}
			out := discoverSuit.DiscoverServer().GetServiceWithCache(discoverSuit.DefaultCtx, service)
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			t.Logf("pass: out is %+v", out)
		})
	})
}

// 测试discover ratelimit
func TestDiscoverRateLimits(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	convey.Convey("限流规则测试", t, func() {
		_, service := discoverSuit.createCommonService(t, 1)
		defer discoverSuit.cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		_, rateLimitResp := discoverSuit.createCommonRateLimit(t, service, 1)
		defer discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
		defer discoverSuit.cleanRateLimitRevision(service.GetName().GetValue(), service.GetNamespace().GetValue())
		convey.Convey("正常获取限流规则", func() {
			time.Sleep(discoverSuit.UpdateCacheInterval())
			out := discoverSuit.DiscoverServer().GetRateLimitWithCache(discoverSuit.DefaultCtx, service)
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			convey.So(len(out.GetRateLimit().GetRules()), convey.ShouldEqual, 1)
			checkRateLimit(t, rateLimitResp, out.GetRateLimit().GetRules()[0])
			t.Logf("pass: out is %+v", out)
			// 再次请求
			out = discoverSuit.DiscoverServer().GetRateLimitWithCache(discoverSuit.DefaultCtx, out.GetService())
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			convey.So(out.GetCode().GetValue(), convey.ShouldEqual, api.DataNoChange)
			t.Logf("pass: out is %+v", out)
		})
		convey.Convey("限流规则已删除", func() {
			discoverSuit.deleteRateLimit(t, rateLimitResp)
			time.Sleep(discoverSuit.UpdateCacheInterval())
			out := discoverSuit.DiscoverServer().GetRateLimitWithCache(discoverSuit.DefaultCtx, service)
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			convey.So(len(out.GetRateLimit().GetRules()), convey.ShouldEqual, 0)
			t.Logf("pass: out is %+v", out)
		})
	})
}

// 测试discover ratelimit
func TestDiscoverRateLimits2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	convey.Convey("限流规则异常测试", t, func() {
		_, service := discoverSuit.createCommonService(t, 1)
		defer discoverSuit.cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		convey.Convey("限流规则不存在", func() {
			time.Sleep(discoverSuit.UpdateCacheInterval())
			out := discoverSuit.DiscoverServer().GetRateLimitWithCache(discoverSuit.DefaultCtx, service)
			convey.So(respSuccess(out), convey.ShouldEqual, true)
			convey.So(out.GetRateLimit(), convey.ShouldBeNil)
			t.Logf("pass: out is %+v", out)
		})
		convey.Convey("服务不存在", func() {
			services := []*apiservice.Service{service}
			discoverSuit.removeCommonServices(t, services)
			time.Sleep(discoverSuit.UpdateCacheInterval())
			out := discoverSuit.DiscoverServer().GetRateLimitWithCache(discoverSuit.DefaultCtx, service)
			convey.So(respSuccess(out), convey.ShouldEqual, false)
			t.Logf("pass: out is %+v", out)
		})
	})
}
