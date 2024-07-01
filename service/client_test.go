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
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

// 测试discover instances
func TestDiscoverInstances(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("服务发现测试", func(t *testing.T) {
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
		t.Run("正常服务发现，返回的数据齐全", func(t *testing.T) {
			_ = discoverSuit.DiscoverServer().Cache().(*cache.CacheManager).TestUpdate()
			out := discoverSuit.DiscoverServer().ServiceInstancesCache(discoverSuit.DefaultCtx, &apiservice.DiscoverFilter{}, service)
			assert.True(t, respSuccess(out))
			assert.Equal(t, count, len(out.GetInstances()))
			assert.True(t, len(out.GetService().GetMetadata()) > 0)
			for _, resp := range out.GetInstances() {
				found := false
				for _, req := range reqInstances {
					if resp.GetHost().GetValue() == req.GetHost().GetValue() {
						instanceCheck(t, req, resp) // expect actual
						// 检查resp中必须包含额外的metadata
						assert.Equal(t, resp.Metadata["version"], req.GetVersion().GetValue())
						assert.Equal(t, resp.Metadata["protocol"], req.GetProtocol().GetValue())
						found = true
						t.Logf("%+v", resp)
						break
					}
				}
				assert.True(t, found)
			}
		})
		t.Run("service-metadata修改，revision会修改", func(t *testing.T) {
			out := discoverSuit.DiscoverServer().ServiceInstancesCache(discoverSuit.DefaultCtx, &apiservice.DiscoverFilter{}, service)
			assert.True(t, respSuccess(out))
			oldRevision := out.GetService().GetRevision().GetValue()

			service.Metadata = make(map[string]string)
			service.Metadata["new-metadata1"] = "1233"
			service.Metadata["new-metadata2"] = "2342"
			resp := discoverSuit.DiscoverServer().UpdateServices(discoverSuit.DefaultCtx, []*apiservice.Service{service})
			_ = discoverSuit.DiscoverServer().Cache().(*cache.CacheManager).TestUpdate()
			assert.True(t, respSuccess(resp))
			assert.NotEqual(t, resp.Responses[0].GetService().GetRevision().GetValue(), oldRevision)
			assert.Equal(t, resp.Responses[0].GetService().GetMetadata()["new-metadata1"], "1233")
			assert.Equal(t, resp.Responses[0].GetService().GetMetadata()["new-metadata2"], "2342")
			serviceCheck(t, service, resp.Responses[0].GetService())
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

	t.Run("熔断规则测试", func(t *testing.T) {
		rules, resp := createCircuitBreakerRules(discoverSuit, 5)
		defer cleanCircuitBreakerRules(discoverSuit, resp)
		service := &apiservice.Service{Name: utils.NewStringValue("testDestService"), Namespace: utils.NewStringValue("test")}
		t.Run("正常获取熔断规则", func(t *testing.T) {
			_ = discoverSuit.DiscoverServer().Cache().(*cache.CacheManager).TestUpdate()
			out := discoverSuit.DiscoverServer().GetCircuitBreakerWithCache(discoverSuit.DefaultCtx, service)
			assert.True(t, respSuccess(out))
			assert.Equal(t, len(out.GetCircuitBreaker().GetRules()), len(rules))
			t.Logf("pass: out is %+v", out)

			// 再次请求
			out = discoverSuit.DiscoverServer().GetCircuitBreakerWithCache(discoverSuit.DefaultCtx, out.GetService())
			assert.True(t, respSuccess(out))
			assert.Equal(t, out.GetCode().GetValue(), api.DataNoChange)
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

	t.Run("熔断规则异常测试", func(t *testing.T) {
		_, resp := createCircuitBreakerRules(discoverSuit, 1)
		defer cleanCircuitBreakerRules(discoverSuit, resp)
		service := &apiservice.Service{Name: utils.NewStringValue("testDestService"), Namespace: utils.NewStringValue("default")}
		t.Run("熔断规则不存在", func(t *testing.T) {
			_ = discoverSuit.DiscoverServer().Cache().(*cache.CacheManager).TestUpdate()
			out := discoverSuit.DiscoverServer().GetCircuitBreakerWithCache(discoverSuit.DefaultCtx, service)
			assert.True(t, respSuccess(out))
			assert.Equal(t, 0, len(out.GetCircuitBreaker().GetRules()))
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

	t.Run("服务测试", func(t *testing.T) {
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
		_ = discoverSuit.DiscoverServer().Cache().(*cache.CacheManager).TestUpdate()

		t.Run("正常获取服务", func(t *testing.T) {
			requestService := &apiservice.Service{
				Metadata: requestMeta,
			}
			out := discoverSuit.DiscoverServer().GetServiceWithCache(discoverSuit.DefaultCtx, requestService)
			assert.True(t, respSuccess(out))
			if len(out.GetServices()) == 2 {
				t.Logf("pass: out service is %+v", out.GetServices())
			} else {
				t.Logf("error: out is %+v", out)
			}
		})

		t.Run("元数据匹配到的服务为空", func(t *testing.T) {
			requestMeta := make(map[string]string)
			requestMeta["test"] = "test"
			requestService := &apiservice.Service{
				Metadata: requestMeta,
			}
			out := discoverSuit.DiscoverServer().GetServiceWithCache(discoverSuit.DefaultCtx, requestService)
			assert.True(t, respSuccess(out))
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

	t.Run("服务正常测试", func(t *testing.T) {
		t.Run("元数据不存在", func(t *testing.T) {
			service := &apiservice.Service{}
			out := discoverSuit.DiscoverServer().GetServiceWithCache(discoverSuit.DefaultCtx, service)
			assert.True(t, respSuccess(out))
			t.Logf("pass: out is %+v", out)
		})
		t.Run("元数据为空", func(t *testing.T) {
			service := &apiservice.Service{
				Metadata: make(map[string]string),
			}
			out := discoverSuit.DiscoverServer().GetServiceWithCache(discoverSuit.DefaultCtx, service)
			assert.True(t, respSuccess(out))
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

	t.Run("限流规则测试", func(t *testing.T) {
		_, service := discoverSuit.createCommonService(t, 1)
		defer discoverSuit.cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		_, rateLimitResp := discoverSuit.createCommonRateLimit(t, service, 1)
		defer discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
		defer discoverSuit.cleanRateLimitRevision(service.GetName().GetValue(), service.GetNamespace().GetValue())
		t.Run("正常获取限流规则", func(t *testing.T) {
			_ = discoverSuit.DiscoverServer().Cache().(*cache.CacheManager).TestUpdate()
			out := discoverSuit.DiscoverServer().GetRateLimitWithCache(discoverSuit.DefaultCtx, service)
			assert.True(t, respSuccess(out))
			assert.Equal(t, len(out.GetRateLimit().GetRules()), 1)
			checkRateLimit(t, rateLimitResp, out.GetRateLimit().GetRules()[0])
			t.Logf("pass: out is %+v", out)
			// 再次请求
			out = discoverSuit.DiscoverServer().GetRateLimitWithCache(discoverSuit.DefaultCtx, out.GetService())
			assert.True(t, respSuccess(out))
			assert.Equal(t, out.GetCode().GetValue(), api.DataNoChange)
			t.Logf("pass: out is %+v", out)
		})
		t.Run("限流规则已删除", func(t *testing.T) {
			discoverSuit.deleteRateLimit(t, rateLimitResp)
			_ = discoverSuit.DiscoverServer().Cache().(*cache.CacheManager).TestUpdate()
			out := discoverSuit.DiscoverServer().GetRateLimitWithCache(discoverSuit.DefaultCtx, service)
			assert.True(t, respSuccess(out))
			assert.Equal(t, len(out.GetRateLimit().GetRules()), 0)
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

	t.Run("限流规则异常测试", func(t *testing.T) {
		_, service := discoverSuit.createCommonService(t, 1)
		defer discoverSuit.cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		t.Run("限流规则不存在", func(t *testing.T) {
			_ = discoverSuit.DiscoverServer().Cache().(*cache.CacheManager).TestUpdate()
			out := discoverSuit.DiscoverServer().GetRateLimitWithCache(discoverSuit.DefaultCtx, service)
			assert.True(t, respSuccess(out))
			assert.Nil(t, out.GetRateLimit())
			t.Logf("pass: out is %+v", out)
		})
		t.Run("服务不存在", func(t *testing.T) {
			_ = discoverSuit.DiscoverServer().Cache().(*cache.CacheManager).TestUpdate()
			out := discoverSuit.DiscoverServer().GetRateLimitWithCache(discoverSuit.DefaultCtx, &apiservice.Service{
				Name:      utils.NewStringValue("not_exist_service"),
				Namespace: utils.NewStringValue("not_exist_namespace"),
			})
			assert.True(t, respSuccess(out))
			t.Logf("pass: out is %+v", out)
		})
	})
}

// TestServer_ReportServiceContract 测试上报服务合约
func TestServer_ReportServiceContract(t *testing.T) {
	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}

	req := &apiservice.ServiceContract{
		Namespace: "default",
		Service:   "test",
		Protocol:  "http",
		Name:      "test",
		Version:   "1.0.0",
		Content:   "test",
		Interfaces: []*apiservice.InterfaceDescriptor{
			{
				Name:   "test",
				Path:   "/test",
				Method: "GET",
			},
		},
	}

	t.Run("multi_report_same_contract", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			copyReq := proto.Clone(req).(*apiservice.ServiceContract)
			copyReq.Content = fmt.Sprintf("test%d", i)
			rsp := discoverSuit.DiscoverServer().ReportServiceContract(discoverSuit.DefaultCtx, copyReq)
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), rsp.GetCode().GetValue(), rsp.GetInfo().GetValue())
		}
	})

	t.Run("get_service_contract", func(t *testing.T) {
		err := discoverSuit.CacheMgr().TestUpdate()
		assert.NoError(t, err)
		rsp := discoverSuit.DiscoverServer().GetServiceContractWithCache(discoverSuit.DefaultCtx, req)
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), rsp.GetCode().GetValue(), rsp.GetInfo().GetValue())
		assert.Equal(t, 1, len(rsp.GetServiceContract().GetInterfaces()))
	})

}
