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
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/duration"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

/**
 * @brief 测试创建限流规则
 */
func TestCreateRateLimit(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, serviceResp := discoverSuit.createCommonService(t, 0)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	defer discoverSuit.cleanRateLimitRevision(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("正常创建限流规则", func(t *testing.T) {
		_ = discoverSuit.server.Cache().Clear()

		time.Sleep(5 * time.Second)

		rateLimitReq, rateLimitResp := discoverSuit.createCommonRateLimit(t, serviceResp, 3)
		defer discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())

		// 等待缓存更新
		time.Sleep(discoverSuit.updateCacheInterval)
		resp := discoverSuit.server.GetRateLimitWithCache(context.Background(), serviceResp)
		checkRateLimit(t, rateLimitReq, resp.GetRateLimit().GetRules()[0])
	})

	t.Run("创建限流规则，删除，再创建，可以正常创建", func(t *testing.T) {
		_ = discoverSuit.server.Cache().Clear()
		time.Sleep(5 * time.Second)

		rateLimitReq, rateLimitResp := discoverSuit.createCommonRateLimit(t, serviceResp, 3)
		defer discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
		discoverSuit.deleteRateLimit(t, rateLimitResp)
		if resp := discoverSuit.server.CreateRateLimits(discoverSuit.defaultCtx, []*api.Rule{rateLimitReq}); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}

		// 等待缓存更新
		time.Sleep(discoverSuit.updateCacheInterval)
		resp := discoverSuit.server.GetRateLimitWithCache(context.Background(), serviceResp)
		checkRateLimit(t, rateLimitReq, resp.GetRateLimit().GetRules()[0])
		discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
	})

	t.Run("重复创建限流规则，返回成功", func(t *testing.T) {
		rateLimitReq, rateLimitResp := discoverSuit.createCommonRateLimit(t, serviceResp, 3)
		defer discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
		if resp := discoverSuit.server.CreateRateLimits(discoverSuit.defaultCtx, []*api.Rule{rateLimitReq}); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		} else {
			t.Log("pass")
		}
		discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
	})

	t.Run("创建限流规则时，没有传递token，返回失败", func(t *testing.T) {

		oldCtx := discoverSuit.defaultCtx

		discoverSuit.defaultCtx = context.Background()

		defer func() {
			discoverSuit.defaultCtx = oldCtx
		}()

		rateLimit := &api.Rule{
			Service:   serviceResp.GetName(),
			Namespace: serviceResp.GetNamespace(),
			Labels:    map[string]*api.MatchString{},
		}
		if resp := discoverSuit.server.CreateRateLimits(discoverSuit.defaultCtx, []*api.Rule{rateLimit}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	// t.Run("创建限流规则时，没有传递labels，返回失败", func(t *testing.T) {
	// 	rateLimit := &api.Rule{
	// 		Service:      serviceResp.GetName(),
	// 		Namespace:    serviceResp.GetNamespace(),
	// 		ServiceToken: serviceResp.GetToken(),
	// 	}
	// 	if resp := discoverSuit.server.CreateRateLimits(discoverSuit.defaultCtx, []*api.Rule{rateLimit}); !respSuccess(resp) {
	// 		t.Logf("pass: %s", resp.GetInfo().GetValue())
	// 	} else {
	// 		t.Fatalf("error")
	// 	}
	// })

	t.Run("创建限流规则时，amounts具有相同的duration，返回失败", func(t *testing.T) {
		rateLimit := &api.Rule{
			Service:   serviceResp.GetName(),
			Namespace: serviceResp.GetNamespace(),
			Labels:    map[string]*api.MatchString{},
			Amounts: []*api.Amount{
				{
					MaxAmount: utils.NewUInt32Value(1),
					ValidDuration: &duration.Duration{
						Seconds: 10,
						Nanos:   10,
					},
				},
				{
					MaxAmount: utils.NewUInt32Value(2),
					ValidDuration: &duration.Duration{
						Seconds: 10,
						Nanos:   10,
					},
				},
			},
			ServiceToken: serviceResp.GetToken(),
		}
		if resp := discoverSuit.server.CreateRateLimits(discoverSuit.defaultCtx, []*api.Rule{rateLimit}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error")
		}
	})

	t.Run("并发创建同一服务的限流规则，可以正常创建", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 1; i <= 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				_, rateLimitResp := discoverSuit.createCommonRateLimit(t, serviceResp, index)
				defer discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
			}(i)
		}
		wg.Wait()
		t.Log("pass")
	})

	t.Run("并发创建不同服务的限流规则，可以正常创建", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 1; i <= 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				_, serviceResp := discoverSuit.createCommonService(t, index)
				defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
				defer discoverSuit.cleanRateLimitRevision(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
				_, rateLimitResp := discoverSuit.createCommonRateLimit(t, serviceResp, 3)
				defer discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
			}(i)
		}
		wg.Wait()
		t.Log("pass")
	})

	t.Run("为不存在的服务创建限流规则，返回成功", func(t *testing.T) {
		_, serviceResp := discoverSuit.createCommonService(t, 2)
		discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		rateLimit := &api.Rule{
			Service:      serviceResp.GetName(),
			Namespace:    serviceResp.GetNamespace(),
			Labels:       map[string]*api.MatchString{},
			ServiceToken: serviceResp.GetToken(),
		}
		if resp := discoverSuit.server.CreateRateLimits(discoverSuit.defaultCtx, []*api.Rule{rateLimit}); respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error : %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 测试删除限流规则
 */
func TestDeleteRateLimit(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, serviceResp := discoverSuit.createCommonService(t, 0)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	defer discoverSuit.cleanRateLimitRevision(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	getRateLimits := func(t *testing.T, service *api.Service, expectNum uint32) []*api.Rule {
		filters := map[string]string{
			"service":   service.GetName().GetValue(),
			"namespace": service.GetNamespace().GetValue(),
		}
		resp := discoverSuit.server.GetRateLimits(discoverSuit.defaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error")
		}
		if resp.GetAmount().GetValue() != expectNum {
			t.Fatalf("error")
		}
		return resp.GetRateLimits()
	}

	t.Run("删除存在的限流规则，可以正常删除", func(t *testing.T) {
		_, rateLimitResp := discoverSuit.createCommonRateLimit(t, serviceResp, 3)
		defer discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
		discoverSuit.deleteRateLimit(t, rateLimitResp)
		getRateLimits(t, serviceResp, 0)
		t.Log("pass")
	})

	t.Run("删除不存在的限流规则，返回正常", func(t *testing.T) {
		_, rateLimitResp := discoverSuit.createCommonRateLimit(t, serviceResp, 3)
		discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
		discoverSuit.deleteRateLimit(t, rateLimitResp)
		getRateLimits(t, serviceResp, 0)
		t.Log("pass")
	})

	t.Run("删除限流规则时，没有传递token，返回失败", func(t *testing.T) {
		rateLimitReq, rateLimitResp := discoverSuit.createCommonRateLimit(t, serviceResp, 3)
		defer discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
		rateLimitReq.ServiceToken = utils.NewStringValue("")

		oldCtx := discoverSuit.defaultCtx

		discoverSuit.defaultCtx = context.Background()

		defer func() {
			discoverSuit.defaultCtx = oldCtx
		}()

		if resp := discoverSuit.server.DeleteRateLimits(discoverSuit.defaultCtx, []*api.Rule{rateLimitReq}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("并发删除限流规则，可以正常删除", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 1; i <= 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				_, serviceResp := discoverSuit.createCommonService(t, index)
				defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
				defer discoverSuit.cleanRateLimitRevision(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
				rateLimitReq, rateLimitResp := discoverSuit.createCommonRateLimit(t, serviceResp, 3)
				defer discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
				discoverSuit.deleteRateLimit(t, rateLimitReq)
			}(i)
		}
		wg.Wait()
		t.Log("pass")
	})
}

/**
 * @brief 测试更新限流规则
 */
func TestUpdateRateLimit(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, serviceResp := discoverSuit.createCommonService(t, 0)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	defer discoverSuit.cleanRateLimitRevision(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	_, rateLimitResp := discoverSuit.createCommonRateLimit(t, serviceResp, 1)
	defer discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())

	updateRateLimitContent(rateLimitResp, 2)

	t.Run("更新单个限流规则，可以正常更新", func(t *testing.T) {
		discoverSuit.updateRateLimit(t, rateLimitResp)
		filters := map[string]string{
			"service":   serviceResp.GetName().GetValue(),
			"namespace": serviceResp.GetNamespace().GetValue(),
		}
		resp := discoverSuit.server.GetRateLimits(discoverSuit.defaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error")
		}
		checkRateLimit(t, rateLimitResp, resp.GetRateLimits()[0])
	})

	t.Run("更新一个不存在的限流规则", func(t *testing.T) {
		discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
		if resp := discoverSuit.server.UpdateRateLimits(discoverSuit.defaultCtx, []*api.Rule{rateLimitResp}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error")
		}
	})

	t.Run("更新限流规则时，没有传递token，正常", func(t *testing.T) {

		oldCtx := discoverSuit.defaultCtx

		discoverSuit.defaultCtx = context.Background()

		defer func() {
			discoverSuit.defaultCtx = oldCtx
		}()

		rateLimitResp.ServiceToken = utils.NewStringValue("")
		if resp := discoverSuit.server.UpdateRateLimits(discoverSuit.defaultCtx, []*api.Rule{rateLimitResp}); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error")
		}
	})

	t.Run("并发更新限流规则时，可以正常更新", func(t *testing.T) {
		var wg sync.WaitGroup
		errs := make(chan error)
		for i := 1; i <= 50; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				_, serviceResp := discoverSuit.createCommonService(t, index)
				defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
				discoverSuit.cleanRateLimitRevision(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
				_, rateLimitResp := discoverSuit.createCommonRateLimit(t, serviceResp, index)
				defer discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
				updateRateLimitContent(rateLimitResp, index+1)
				discoverSuit.updateRateLimit(t, rateLimitResp)
				filters := map[string]string{
					"service":   serviceResp.GetName().GetValue(),
					"namespace": serviceResp.GetNamespace().GetValue(),
				}
				resp := discoverSuit.server.GetRateLimits(discoverSuit.defaultCtx, filters)
				if !respSuccess(resp) {
					errs <- fmt.Errorf("error : %v", resp)
				}
				checkRateLimit(t, rateLimitResp, resp.GetRateLimits()[0])
			}(i)
		}
		go func() {
			wg.Wait()
			close(errs)
		}()

		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}

		t.Log("pass")
	})

	t.Run("启用限流规则", func(t *testing.T) {

		oldCtx := discoverSuit.defaultCtx

		discoverSuit.defaultCtx = context.Background()

		defer func() {
			discoverSuit.defaultCtx = oldCtx
		}()

		ruleContents := []*api.Rule{
			{
				Id:      utils.NewStringValue(rateLimitResp.GetId().GetValue()),
				Disable: utils.NewBoolValue(true),
			},
		}
		if resp := discoverSuit.server.EnableRateLimits(discoverSuit.defaultCtx, ruleContents); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error")
		}
	})
}

/*
 * @brief 测试查询限流规则
 */
func TestGetRateLimit(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	serviceNum := 10
	rateLimitsNum := 30
	rateLimits := make([]*api.Rule, rateLimitsNum)
	serviceName := ""
	namespaceName := ""
	for i := 0; i < serviceNum; i++ {
		_, serviceResp := discoverSuit.createCommonService(t, i)
		if i == 5 {
			serviceName = serviceResp.GetName().GetValue()
			namespaceName = serviceResp.GetNamespace().GetValue()
		}
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		defer discoverSuit.cleanRateLimitRevision(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		for j := 0; j < rateLimitsNum/serviceNum; j++ {
			_, rateLimitResp := discoverSuit.createCommonRateLimit(t, serviceResp, j)
			defer discoverSuit.cleanRateLimit(rateLimitResp.GetId().GetValue())
			rateLimits = append(rateLimits, rateLimitResp)
		}
	}

	t.Run("查询限流规则，过滤条件为service", func(t *testing.T) {
		filters := map[string]string{
			"service": serviceName,
		}
		resp := discoverSuit.server.GetRateLimits(discoverSuit.defaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetSize().GetValue() != uint32(rateLimitsNum/serviceNum) {
			t.Fatalf("expect num is %d, actual num is %d", rateLimitsNum/serviceNum, resp.GetSize().GetValue())
		}
		t.Logf("pass: num is %d", resp.GetSize().GetValue())
	})

	t.Run("查询限流规则，过滤条件为namespace", func(t *testing.T) {
		filters := map[string]string{
			"namespace": namespaceName,
		}
		resp := discoverSuit.server.GetRateLimits(discoverSuit.defaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetSize().GetValue() != uint32(rateLimitsNum) {
			t.Fatalf("expect num is %d, actual num is %d", serviceNum, resp.GetSize().GetValue())
		}
		t.Logf("pass: num is %d", resp.GetSize().GetValue())
	})

	t.Run("查询限流规则，过滤条件为不存在的namespace", func(t *testing.T) {
		filters := map[string]string{
			"namespace": "Development",
		}
		resp := discoverSuit.server.GetRateLimits(discoverSuit.defaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetSize().GetValue() != 0 {
			t.Fatalf("expect num is 0, actual num is %d", resp.GetSize().GetValue())
		}
		t.Logf("pass: num is %d", resp.GetSize().GetValue())
	})

	t.Run("查询限流规则，过滤条件为namespace和service", func(t *testing.T) {
		filters := map[string]string{
			"service":   serviceName,
			"namespace": namespaceName,
		}
		resp := discoverSuit.server.GetRateLimits(discoverSuit.defaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetSize().GetValue() != uint32(rateLimitsNum/serviceNum) {
			t.Fatalf("expect num is %d, actual num is %d", rateLimitsNum/serviceNum, resp.GetSize().GetValue())
		}
		t.Logf("pass: num is %d", resp.GetSize().GetValue())
	})

	t.Run("查询限流规则，过滤条件为offset和limit", func(t *testing.T) {
		filters := map[string]string{
			"offset": "1",
			"limit":  "5",
		}
		resp := discoverSuit.server.GetRateLimits(discoverSuit.defaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetSize().GetValue() != 5 {
			t.Fatalf("expect num is 5, actual num is %d", resp.GetSize().GetValue())
		}
		t.Logf("pass: num is %d", resp.GetSize().GetValue())
	})

	t.Run("查询限流规则列表，过滤条件为name", func(t *testing.T) {
		filters := map[string]string{
			"name":  "rule_name_0",
			"brief": "true",
		}
		resp := discoverSuit.server.GetRateLimits(discoverSuit.defaultCtx, filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetSize().GetValue() != uint32(serviceNum) {
			t.Fatalf("expect num is %d, actual num is %d", serviceNum, resp.GetSize().GetValue())
		}
	})

	// t.Run("查询限流规则，过滤条件为arguments中的key", func(t *testing.T) {
	// 	filters := map[string]string{
	// 		"labels": labelsKey,
	// 	}
	// 	resp := discoverSuit.server.GetRateLimits(discoverSuit.defaultCtx, filters)
	// 	if !respSuccess(resp) {
	// 		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	// 	}
	// 	if resp.GetSize().GetValue() != uint32(serviceNum) {
	// 		t.Fatalf("expect num is %d, actual num is %d", serviceNum, resp.GetSize().GetValue())
	// 	}
	// })

	// t.Run("查询限流规则，过滤条件为arguments中的value", func(t *testing.T) {
	// 	filters := map[string]string{
	// 		"labels": labelsValue.GetValue().GetValue(),
	// 	}
	// 	resp := discoverSuit.server.GetRateLimits(discoverSuit.defaultCtx, filters)
	// 	if !respSuccess(resp) {
	// 		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	// 	}
	// 	if resp.GetSize().GetValue() != uint32(serviceNum) {
	// 		t.Fatalf("expect num is %d, actual num is %d", serviceNum, resp.GetSize().GetValue())
	// 	}
	// })

	// t.Run("查询限流规则，过滤条件为arguments中的key和value", func(t *testing.T) {
	// 	labelsString, err := json.Marshal(labels)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	filters := map[string]string{
	// 		"labels": string(labelsString),
	// 	}
	// 	resp := discoverSuit.server.GetRateLimits(discoverSuit.defaultCtx, filters)
	// 	if !respSuccess(resp) {
	// 		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	// 	}
	// 	if resp.GetSize().GetValue() != uint32(serviceNum) {
	// 		t.Fatalf("expect num is %d, actual num is %d", serviceNum, resp.GetSize().GetValue())
	// 	}
	// })

	t.Run("查询限流规则，offset为负数，返回错误", func(t *testing.T) {
		filters := map[string]string{
			"service":   serviceName,
			"namespace": namespaceName,
			"offset":    "-5",
		}
		resp := discoverSuit.server.GetRateLimits(discoverSuit.defaultCtx, filters)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error")
		}
	})

	t.Run("查询限流规则，limit为负数，返回错误", func(t *testing.T) {
		filters := map[string]string{
			"service":   serviceName,
			"namespace": namespaceName,
			"limit":     "-5",
		}
		resp := discoverSuit.server.GetRateLimits(discoverSuit.defaultCtx, filters)
		if !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error")
		}
	})
}

// test对ratelimit字段进行校验
func TestCheckRatelimitFieldLen(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	rateLimit := &api.Rule{
		Service:      utils.NewStringValue("test"),
		Namespace:    utils.NewStringValue("default"),
		Labels:       map[string]*api.MatchString{},
		ServiceToken: utils.NewStringValue("test"),
	}
	t.Run("创建限流规则，服务名超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldName := rateLimit.Service
		rateLimit.Service = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateRateLimits(discoverSuit.defaultCtx, []*api.Rule{rateLimit})
		rateLimit.Service = oldName
		if resp.Code.Value != api.InvalidServiceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("创建限流规则，命名空间超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldNamespace := rateLimit.Namespace
		rateLimit.Namespace = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateRateLimits(discoverSuit.defaultCtx, []*api.Rule{rateLimit})
		rateLimit.Namespace = oldNamespace
		if resp.Code.Value != api.InvalidNamespaceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("创建限流规则，名称超长", func(t *testing.T) {
		str := genSpecialStr(2049)
		oldName := rateLimit.Name
		rateLimit.Name = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateRateLimits(discoverSuit.defaultCtx, []*api.Rule{rateLimit})
		rateLimit.Name = oldName
		if resp.Code.Value != api.InvalidRateLimitName {
			t.Fatalf("%+v", resp)
		}
	})
}
