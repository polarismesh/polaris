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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/protobuf/ptypes"
	"github.com/polarismesh/specification/source/go/api/v1/model"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
	testsuit "github.com/polarismesh/polaris/test/suit"
)

// 检查routingConfig前后是否一致
func checkSameRoutingConfig(t *testing.T, lhs *apitraffic.Routing, rhs *apitraffic.Routing) {
	if lhs.GetService().GetValue() != rhs.GetService().GetValue() ||
		lhs.GetNamespace().GetValue() != rhs.GetNamespace().GetValue() {
		t.Fatalf("error: (%s), (%s)", lhs, rhs)
	}

	checkFunc := func(labels string, in []*apitraffic.Route, out []*apitraffic.Route) bool {
		if len(in) == 0 && len(out) == 0 {
			return true
		}

		inStr, err := json.Marshal(in)
		assert.NoError(t, err)
		outStr, err := json.Marshal(out)
		assert.NoError(t, err)

		if in == nil || out == nil {
			t.Fatalf("%s error: empty (%s), (%s)", labels, string(inStr), string(outStr))
			return false
		}

		assert.Equalf(t, len(in), len(out), "%s len(in) != len(out)", labels)

		inRoutes := []*apitraffic.Route{}
		outRoutes := []*apitraffic.Route{}

		if err := json.Unmarshal(inStr, &inRoutes); err != nil {
			t.Fatal(err)
		}

		if err := json.Unmarshal(outStr, &outRoutes); err != nil {
			t.Fatal(err)
		}

		for i := range inRoutes {
			for j := range inRoutes[i].Destinations {
				inRoutes[i].Destinations[j].Name = nil
				inRoutes[i].Destinations[j].Isolate = nil
				outRoutes[i].Destinations[j].Name = nil
				outRoutes[i].Destinations[j].Isolate = nil
			}
		}

		if !reflect.DeepEqual(inRoutes, outRoutes) {
			t.Fatalf("%s error: (%s), (%s)", labels, string(inStr), string(outStr))
			return false
		}

		return true
	}

	checkFunc("Inbounds", lhs.Inbounds, rhs.Inbounds)
	checkFunc("Outbounds", lhs.Outbounds, rhs.Outbounds)
}

// 测试创建路由配置
func TestCreateRoutingConfig(t *testing.T) {
	t.Run("正常创建路由配置配置请求", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}

		defer discoverSuit.Destroy()
		_, serviceResp := discoverSuit.createCommonService(t, 200)
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		_, _ = discoverSuit.createCommonRoutingConfig(t, serviceResp, 3, 0)

		// 对写进去的数据进行查询
		_ = discoverSuit.CacheMgr().TestUpdate()
		out := discoverSuit.DiscoverServer().GetRoutingConfigWithCache(discoverSuit.DefaultCtx, serviceResp)
		defer discoverSuit.cleanCommonRoutingConfig(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}
	})

	t.Run("参数缺失，报错", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		_, serviceResp := discoverSuit.createCommonService(t, 20)
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		req := &apitraffic.Routing{}
		resp := discoverSuit.DiscoverServer().CreateRoutingConfigs(discoverSuit.DefaultCtx, []*apitraffic.Routing{req})
		assert.False(t, respSuccess(resp))
		t.Logf("%s", resp.GetInfo().GetValue())

		req.Service = serviceResp.Name
		resp = discoverSuit.DiscoverServer().CreateRoutingConfigs(discoverSuit.DefaultCtx, []*apitraffic.Routing{req})
		assert.False(t, respSuccess(resp))
		t.Logf("%s", resp.GetInfo().GetValue())

		req.Namespace = serviceResp.Namespace
		resp = discoverSuit.DiscoverServer().CreateRoutingConfigs(discoverSuit.DefaultCtx, []*apitraffic.Routing{req})
		defer discoverSuit.cleanCommonRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())
		assert.True(t, respSuccess(resp))
		t.Logf("%s", resp.GetInfo().GetValue())
	})

	t.Run("服务不存在，创建路由配置不报错", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		_, serviceResp := discoverSuit.createCommonService(t, 120)
		discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		_ = discoverSuit.CacheMgr().TestUpdate()
		req := &apitraffic.Routing{}
		req.Service = serviceResp.Name
		req.Namespace = serviceResp.Namespace
		req.ServiceToken = serviceResp.Token
		resp := discoverSuit.DiscoverServer().CreateRoutingConfigs(discoverSuit.DefaultCtx, []*apitraffic.Routing{req})
		assert.False(t, respSuccess(resp))
		t.Logf("%s", resp.GetInfo().GetValue())
	})
}

// 测试创建路由配置
func TestUpdateRoutingConfig(t *testing.T) {
	t.Run("更新V1路由规则, 成功转为V2规则", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}

		_, svc := discoverSuit.createCommonService(t, 200)
		v1Rule, _ := discoverSuit.createCommonRoutingConfigV1IntoOldStore(t, svc, 3, 0)
		t.Cleanup(func() {
			discoverSuit.cleanServiceName(svc.GetName().GetValue(), svc.GetNamespace().GetValue())
			discoverSuit.cleanCommonRoutingConfig(svc.GetName().GetValue(), svc.GetNamespace().GetValue())
			discoverSuit.truncateCommonRoutingConfigV2()
			discoverSuit.Destroy()
		})

		v1Rule.Outbounds = v1Rule.Inbounds
		uResp := discoverSuit.DiscoverServer().UpdateRoutingConfigs(discoverSuit.DefaultCtx, []*apitraffic.Routing{v1Rule})
		assert.True(t, respSuccess(uResp))

		// 等缓存层更新
		_ = discoverSuit.CacheMgr().TestUpdate()

		// 直接查询存储无法查询到 v1 的路由规则
		total, routingsV1, err := discoverSuit.Storage.GetRoutingConfigs(map[string]string{}, 0, 100)
		assert.NoError(t, err, err)
		assert.Equal(t, uint32(0), total, "v1 routing must delete and transfer to v1")
		assert.Equal(t, 0, len(routingsV1), "v1 routing ret len need zero")

		// 从缓存中查询应该查到 6 条 v2 的路由规则
		out := discoverSuit.DiscoverServer().QueryRoutingConfigsV2(discoverSuit.DefaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
		})
		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}
		assert.Equal(t, int(6), int(out.GetAmount().GetValue()), "query routing size")
		rulesV2, err := unmarshalRoutingV2toAnySlice(out.GetData())
		assert.NoError(t, err)
		for i := range rulesV2 {
			item := rulesV2[i]
			assert.True(t, item.Enable, "v1 to v2 need default open enable")
			msg := &apitraffic.RuleRoutingConfig{}
			err := ptypes.UnmarshalAny(item.GetRoutingConfig(), msg)
			assert.NoError(t, err)
			assert.True(t, len(msg.GetSources()) == 0, "RuleRoutingConfig.Sources len != 0")
			assert.True(t, len(msg.GetDestinations()) == 0, "RuleRoutingConfig.Destinations len != 0")
			assert.True(t, len(msg.GetRules()) != 0, "RuleRoutingConfig.Rules len == 0")
		}
	})
}

// 测试缓存获取路由配置
func TestGetRoutingConfigWithCache(t *testing.T) {

	t.Run("多个服务的，多个路由配置，都可以查询到", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		total := 20
		serviceResps := make([]*apiservice.Service, 0, total)
		routingResps := make([]*apitraffic.Routing, 0, total)
		for i := 0; i < total; i++ {
			_, resp := discoverSuit.createCommonService(t, i)
			defer discoverSuit.cleanServiceName(resp.GetName().GetValue(), resp.GetNamespace().GetValue())
			serviceResps = append(serviceResps, resp)

			_, routingResp := discoverSuit.createCommonRoutingConfig(t, resp, 2, 0)
			defer discoverSuit.cleanCommonRoutingConfig(resp.GetName().GetValue(), resp.GetNamespace().GetValue())
			routingResps = append(routingResps, routingResp)
		}

		_ = discoverSuit.CacheMgr().TestUpdate()
		for i := 0; i < total; i++ {
			t.Logf("service : name=%s namespace=%s", serviceResps[i].GetName().GetValue(), serviceResps[i].GetNamespace().GetValue())
			out := discoverSuit.DiscoverServer().GetRoutingConfigWithCache(discoverSuit.DefaultCtx, serviceResps[i])
			checkSameRoutingConfig(t, routingResps[i], out.GetRouting())
		}
	})

	t.Run("走v2接口创建路由规则，不启用查不到，启用可以查到", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		resp := discoverSuit.createCommonRoutingConfigV2(t, 1)
		assert.Equal(t, 1, len(resp))

		svcName := fmt.Sprintf("in-source-service-%d", 0)
		namespaceName := fmt.Sprintf("in-source-service-%d", 0)

		svcResp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{{
			Name:      utils.NewStringValue(svcName),
			Namespace: utils.NewStringValue(namespaceName),
		}})
		defer discoverSuit.cleanServiceName(svcName, namespaceName)
		if !respSuccess(svcResp) {
			t.Fatal(svcResp.Info)
		}

		_ = discoverSuit.CacheMgr().TestUpdate()
		t.Logf("service : name=%s namespace=%s", svcName, namespaceName)
		out := discoverSuit.DiscoverServer().GetRoutingConfigWithCache(discoverSuit.DefaultCtx, &apiservice.Service{
			Name:      utils.NewStringValue(svcName),
			Namespace: utils.NewStringValue(namespaceName),
		})

		assert.True(t, len(out.GetRouting().GetOutbounds()) == 0, "inBounds must be zero")

		_ = discoverSuit.CacheMgr().TestUpdate()

		enableResp := discoverSuit.DiscoverServer().EnableRoutings(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{
			{
				Id:     resp[0].Id,
				Enable: true,
			},
		})

		if !respSuccess(enableResp) {
			t.Fatal(enableResp.Info)
		}

		_ = discoverSuit.CacheMgr().TestUpdate()
		out = discoverSuit.DiscoverServer().GetRoutingConfigWithCache(discoverSuit.DefaultCtx, &apiservice.Service{
			Name:      utils.NewStringValue(svcName),
			Namespace: utils.NewStringValue(namespaceName),
		})

		assert.True(t, len(out.GetRouting().GetOutbounds()) == 1, "inBounds must be one")
	})

	t.Run("走v2接口创建路由规则，通配服务", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		rules := testsuit.MockRoutingV2(t, 1)
		ruleRoutings := &apitraffic.RuleRoutingConfig{
			Sources: []*apitraffic.SourceService{
				{
					Service:   "*",
					Namespace: "*",
					Arguments: []*apitraffic.SourceMatch{
						{
							Type: apitraffic.SourceMatch_CUSTOM,
							Key:  "key",
							Value: &apimodel.MatchString{
								Type: apimodel.MatchString_EXACT,
								Value: &wrapperspb.StringValue{
									Value: "123",
								},
								ValueType: apimodel.MatchString_TEXT,
							},
						},
					},
				},
			},
			Destinations: []*apitraffic.DestinationGroup{
				{
					Service:   "mock-servcie-test1",
					Namespace: "mock-namespace-test1",
					Labels: map[string]*apimodel.MatchString{
						"key": {
							Type: apimodel.MatchString_EXACT,
							Value: &wrapperspb.StringValue{
								Value: "value",
							},
							ValueType: apimodel.MatchString_TEXT,
						},
					},
					Priority: 0,
					Weight:   0,
					Transfer: "",
					Isolate:  false,
					Name:     "123",
				},
			},
		}

		any, err := ptypes.MarshalAny(ruleRoutings)
		if err != nil {
			t.Fatal(err)
			return
		}
		rules[0].RoutingPolicy = apitraffic.RoutingPolicy_RulePolicy
		rules[0].RoutingConfig = any

		resp := discoverSuit.createCommonRoutingConfigV2WithReq(t, rules)
		defer discoverSuit.truncateCommonRoutingConfigV2()
		assert.Equal(t, 1, len(resp))

		svcName := fmt.Sprintf("mock-source-service-%d", 0)
		namespaceName := fmt.Sprintf("mock-source-service-%d", 0)

		svcResp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{{
			Name:      utils.NewStringValue(svcName),
			Namespace: utils.NewStringValue(namespaceName),
		}})
		defer discoverSuit.cleanServiceName(svcName, namespaceName)
		if !respSuccess(svcResp) {
			t.Fatal(svcResp.Info)
		}

		_ = discoverSuit.CacheMgr().TestUpdate()
		t.Logf("service : name=%s namespace=%s", svcName, namespaceName)
		out := discoverSuit.DiscoverServer().GetRoutingConfigWithCache(discoverSuit.DefaultCtx, &apiservice.Service{
			Name:      utils.NewStringValue(svcName),
			Namespace: utils.NewStringValue(namespaceName),
		})

		assert.True(t, len(out.GetRouting().GetOutbounds()) == 0, "inBounds must be zero")
		_ = discoverSuit.CacheMgr().TestUpdate()
		enableResp := discoverSuit.DiscoverServer().EnableRoutings(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{
			{
				Id:     resp[0].Id,
				Enable: true,
			},
		})

		if !respSuccess(enableResp) {
			t.Fatal(enableResp.Info)
		}

		_ = discoverSuit.CacheMgr().TestUpdate()
		out = discoverSuit.DiscoverServer().GetRoutingConfigWithCache(discoverSuit.DefaultCtx, &apiservice.Service{
			Name:      utils.NewStringValue(svcName),
			Namespace: utils.NewStringValue(namespaceName),
		})

		assert.True(t, len(out.GetRouting().GetOutbounds()) == 0, "inBounds must be zero")
	})

	t.Run("服务路由数据不改变，传递了路由revision，不返回数据", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		_, serviceResp := discoverSuit.createCommonService(t, 10)
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		_, routingResp := discoverSuit.createCommonRoutingConfig(t, serviceResp, 2, 0)
		defer discoverSuit.cleanCommonRoutingConfig(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		_ = discoverSuit.CacheMgr().TestUpdate()
		firstResp := discoverSuit.DiscoverServer().GetRoutingConfigWithCache(discoverSuit.DefaultCtx, serviceResp)
		checkSameRoutingConfig(t, routingResp, firstResp.GetRouting())

		serviceResp.Revision = firstResp.Service.Revision
		secondResp := discoverSuit.DiscoverServer().GetRoutingConfigWithCache(discoverSuit.DefaultCtx, serviceResp)
		if secondResp.GetService().GetRevision().GetValue() != serviceResp.GetRevision().GetValue() {
			t.Fatalf("error")
		}
		assert.Equal(t, model.Code(secondResp.GetCode().GetValue()), apimodel.Code_DataNoChange)
	})
	t.Run("路由不存在，不会出异常", func(t *testing.T) {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		_, serviceResp := discoverSuit.createCommonService(t, 10)
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		_ = discoverSuit.CacheMgr().TestUpdate()
		if resp := discoverSuit.DiscoverServer().GetRoutingConfigWithCache(discoverSuit.DefaultCtx, serviceResp); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

// test对routing字段进行校验
func TestCheckRoutingFieldLen(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	req := &apitraffic.Routing{
		ServiceToken: utils.NewStringValue("test"),
		Service:      utils.NewStringValue("test"),
		Namespace:    utils.NewStringValue("default"),
	}

	t.Run("创建路由规则，服务名超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldName := req.Service
		req.Service = utils.NewStringValue(str)
		resp := discoverSuit.DiscoverServer().CreateRoutingConfigs(discoverSuit.DefaultCtx, []*apitraffic.Routing{req})
		req.Service = oldName
		if resp.Code.Value != api.InvalidServiceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("创建路由规则，命名空间超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldNamespace := req.Namespace
		req.Namespace = utils.NewStringValue(str)
		resp := discoverSuit.DiscoverServer().CreateRoutingConfigs(discoverSuit.DefaultCtx, []*apitraffic.Routing{req})
		req.Namespace = oldNamespace
		if resp.Code.Value != api.InvalidNamespaceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("创建路由规则，toeken超长", func(t *testing.T) {
		str := genSpecialStr(2049)
		oldServiceToken := req.ServiceToken
		req.ServiceToken = utils.NewStringValue(str)
		resp := discoverSuit.DiscoverServer().CreateRoutingConfigs(discoverSuit.DefaultCtx, []*apitraffic.Routing{req})
		req.ServiceToken = oldServiceToken
		if resp.Code.Value != api.InvalidServiceToken {
			t.Fatalf("%+v", resp)
		}
	})
}
