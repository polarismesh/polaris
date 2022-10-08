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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	apiv2 "github.com/polarismesh/polaris/common/api/v2"
	"github.com/polarismesh/polaris/common/utils"
)

// 检查routingConfig前后是否一致
func checkSameRoutingConfig(t *testing.T, lhs *api.Routing, rhs *api.Routing) {
	if lhs.GetService().GetValue() != rhs.GetService().GetValue() ||
		lhs.GetNamespace().GetValue() != rhs.GetNamespace().GetValue() {
		t.Fatalf("error: (%s), (%s)", lhs, rhs)
	}

	checkFunc := func(in []*api.Route, out []*api.Route) bool {
		if len(in) == 0 && len(out) == 0 {
			return true
		}

		inStr, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("error: %s", err.Error())
			return false
		}

		outStr, err := json.Marshal(out)
		if err != nil {
			t.Fatalf("error: %s", err.Error())
			return false
		}

		if in == nil || out == nil {
			t.Fatalf("error: empty (%s), (%s)", string(inStr), string(outStr))
			return false
		}

		if len(in) != len(out) {
			t.Fatalf("error: %d, %d", len(in), len(out))
			return false
		}

		inRoutes := []*api.Route{}
		outRoutes := []*api.Route{}

		if err := json.Unmarshal(inStr, &inRoutes); err != nil {
			t.Fatal(err)
		}

		if err := json.Unmarshal(outStr, &outRoutes); err != nil {
			t.Fatal(err)
		}

		for i := range inRoutes {
			for j := range inRoutes[i].Destinations {
				inRoutes[i].Destinations[j].Isolate = nil
			}
		}

		for i := range outRoutes {
			for j := range outRoutes[i].Destinations {
				outRoutes[i].Destinations[j].Isolate = nil
			}
		}

		if !reflect.DeepEqual(inRoutes, outRoutes) {
			t.Fatalf("error: (%s), (%s)", string(inStr), string(outStr))
			return false
		}

		return true
	}

	checkFunc(lhs.Inbounds, rhs.Inbounds)
	checkFunc(lhs.Outbounds, rhs.Outbounds)
}

// 测试创建路由配置
func TestCreateRoutingConfig(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	Convey("正常创建路由配置配置请求", t, func() {
		_, serviceResp := discoverSuit.createCommonService(t, 200)
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		_, _ = discoverSuit.createCommonRoutingConfig(t, serviceResp, 3, 0)

		// 对写进去的数据进行查询
		time.Sleep(discoverSuit.updateCacheInterval)
		out := discoverSuit.server.GetRoutingConfigWithCache(discoverSuit.defaultCtx, serviceResp)
		defer discoverSuit.cleanCommonRoutingConfig(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}
	})
}

// 测试创建路由配置
func TestCreateRoutingConfig2(t *testing.T) {

	Convey("参数缺失，报错", t, func() {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		_, serviceResp := discoverSuit.createCommonService(t, 20)
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		req := &api.Routing{}
		resp := discoverSuit.server.CreateRoutingConfigs(discoverSuit.defaultCtx, []*api.Routing{req})
		So(respSuccess(resp), ShouldEqual, false)
		t.Logf("%s", resp.GetInfo().GetValue())

		req.Service = serviceResp.Name
		resp = discoverSuit.server.CreateRoutingConfigs(discoverSuit.defaultCtx, []*api.Routing{req})
		So(respSuccess(resp), ShouldEqual, false)
		t.Logf("%s", resp.GetInfo().GetValue())

		req.Namespace = serviceResp.Namespace
		resp = discoverSuit.server.CreateRoutingConfigs(discoverSuit.defaultCtx, []*api.Routing{req})
		defer discoverSuit.cleanCommonRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())
		So(respSuccess(resp), ShouldEqual, true)
		t.Logf("%s", resp.GetInfo().GetValue())
	})

	Convey("服务不存在，创建路由配置，报错", t, func() {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		_, serviceResp := discoverSuit.createCommonService(t, 120)
		discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		req := &api.Routing{}
		req.Service = serviceResp.Name
		req.Namespace = serviceResp.Namespace
		req.ServiceToken = serviceResp.Token
		resp := discoverSuit.server.CreateRoutingConfigs(discoverSuit.defaultCtx, []*api.Routing{req})
		So(respSuccess(resp), ShouldEqual, false)
		t.Logf("%s", resp.GetInfo().GetValue())
	})
}

// 测试缓存获取路由配置
func TestGetRoutingConfigWithCache(t *testing.T) {

	Convey("多个服务的，多个路由配置，都可以查询到", t, func() {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		total := 20
		serviceResps := make([]*api.Service, 0, total)
		routingResps := make([]*api.Routing, 0, total)
		for i := 0; i < total; i++ {
			_, resp := discoverSuit.createCommonService(t, i)
			defer discoverSuit.cleanServiceName(resp.GetName().GetValue(), resp.GetNamespace().GetValue())
			serviceResps = append(serviceResps, resp)

			_, routingResp := discoverSuit.createCommonRoutingConfig(t, resp, 2, 0)
			defer discoverSuit.cleanCommonRoutingConfig(resp.GetName().GetValue(), resp.GetNamespace().GetValue())
			routingResps = append(routingResps, routingResp)
		}

		time.Sleep(discoverSuit.updateCacheInterval)
		for i := 0; i < total; i++ {
			t.Logf("service : name=%s namespace=%s", serviceResps[i].GetName().GetValue(), serviceResps[i].GetNamespace().GetValue())
			out := discoverSuit.server.GetRoutingConfigWithCache(discoverSuit.defaultCtx, serviceResps[i])
			checkSameRoutingConfig(t, routingResps[i], out.GetRouting())
		}
	})

	Convey("走v2接口创建路由规则，不启用查不到，启用可以查到", t, func() {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		resp := discoverSuit.createCommonRoutingConfigV2(t, 1)
		assert.Equal(t, 1, len(resp))

		svcName := fmt.Sprintf("in-source-service-%d", 0)
		namespaceName := fmt.Sprintf("in-source-service-%d", 0)

		svcResp := discoverSuit.server.CreateServices(discoverSuit.defaultCtx, []*api.Service{{
			Name:      utils.NewStringValue(svcName),
			Namespace: utils.NewStringValue(namespaceName),
		}})
		defer discoverSuit.cleanServiceName(svcName, namespaceName)
		if !respSuccess(svcResp) {
			t.Fatal(svcResp.Info)
		}

		time.Sleep(discoverSuit.updateCacheInterval)
		t.Logf("service : name=%s namespace=%s", svcName, namespaceName)
		out := discoverSuit.server.GetRoutingConfigWithCache(discoverSuit.defaultCtx, &api.Service{
			Name:      utils.NewStringValue(svcName),
			Namespace: utils.NewStringValue(namespaceName),
		})

		assert.True(t, len(out.GetRouting().GetOutbounds()) == 0, "inBounds must be zero")

		time.Sleep(discoverSuit.updateCacheInterval)

		enableResp := discoverSuit.server.EnableRoutings(discoverSuit.defaultCtx, []*apiv2.Routing{
			{
				Id:     resp[0].Id,
				Enable: true,
			},
		})

		if !respSuccessV2(enableResp) {
			t.Fatal(enableResp.Info)
		}

		time.Sleep(discoverSuit.updateCacheInterval)
		out = discoverSuit.server.GetRoutingConfigWithCache(discoverSuit.defaultCtx, &api.Service{
			Name:      utils.NewStringValue(svcName),
			Namespace: utils.NewStringValue(namespaceName),
		})

		assert.True(t, len(out.GetRouting().GetOutbounds()) == 1, "inBounds must be one")
	})

	Convey("走v2接口创建路由规则，通配服务", t, func() {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		rules := mockRoutingV2(t, 1)
		ruleRoutings := &apiv2.RuleRoutingConfig{
			Sources: []*apiv2.Source{
				{
					Service:   "*",
					Namespace: "*",
					Arguments: []*apiv2.SourceMatch{
						{
							Type: apiv2.SourceMatch_CUSTOM,
							Key:  "key",
							Value: &apiv2.MatchString{
								Type: apiv2.MatchString_EXACT,
								Value: &wrapperspb.StringValue{
									Value: "123",
								},
								ValueType: apiv2.MatchString_TEXT,
							},
						},
					},
				},
			},
			Destinations: []*apiv2.Destination{
				{
					Service:   "mock-servcie-test1",
					Namespace: "mock-namespace-test1",
					Labels: map[string]*apiv2.MatchString{
						"key": {
							Type: apiv2.MatchString_EXACT,
							Value: &wrapperspb.StringValue{
								Value: "value",
							},
							ValueType: apiv2.MatchString_TEXT,
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
		rules[0].RoutingPolicy = apiv2.RoutingPolicy_RulePolicy
		rules[0].RoutingConfig = any

		resp := discoverSuit.createCommonRoutingConfigV2WithReq(t, rules)
		defer discoverSuit.truncateCommonRoutingConfigV2()
		assert.Equal(t, 1, len(resp))

		svcName := fmt.Sprintf("mock-source-service-%d", 0)
		namespaceName := fmt.Sprintf("mock-source-service-%d", 0)

		svcResp := discoverSuit.server.CreateServices(discoverSuit.defaultCtx, []*api.Service{{
			Name:      utils.NewStringValue(svcName),
			Namespace: utils.NewStringValue(namespaceName),
		}})
		defer discoverSuit.cleanServiceName(svcName, namespaceName)
		if !respSuccess(svcResp) {
			t.Fatal(svcResp.Info)
		}

		time.Sleep(discoverSuit.updateCacheInterval)
		t.Logf("service : name=%s namespace=%s", svcName, namespaceName)
		out := discoverSuit.server.GetRoutingConfigWithCache(discoverSuit.defaultCtx, &api.Service{
			Name:      utils.NewStringValue(svcName),
			Namespace: utils.NewStringValue(namespaceName),
		})

		assert.True(t, len(out.GetRouting().GetOutbounds()) == 0, "inBounds must be zero")
		time.Sleep(discoverSuit.updateCacheInterval)
		enableResp := discoverSuit.server.EnableRoutings(discoverSuit.defaultCtx, []*apiv2.Routing{
			{
				Id:     resp[0].Id,
				Enable: true,
			},
		})

		if !respSuccessV2(enableResp) {
			t.Fatal(enableResp.Info)
		}

		time.Sleep(discoverSuit.updateCacheInterval)
		out = discoverSuit.server.GetRoutingConfigWithCache(discoverSuit.defaultCtx, &api.Service{
			Name:      utils.NewStringValue(svcName),
			Namespace: utils.NewStringValue(namespaceName),
		})

		assert.True(t, len(out.GetRouting().GetOutbounds()) == 1, "inBounds must be one")
	})

	Convey("服务路由数据不改变，传递了路由revision，不返回数据", t, func() {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		_, serviceResp := discoverSuit.createCommonService(t, 10)
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		_, routingResp := discoverSuit.createCommonRoutingConfig(t, serviceResp, 2, 0)
		defer discoverSuit.cleanCommonRoutingConfig(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		time.Sleep(discoverSuit.updateCacheInterval * 10)
		firstResp := discoverSuit.server.GetRoutingConfigWithCache(discoverSuit.defaultCtx, serviceResp)
		checkSameRoutingConfig(t, routingResp, firstResp.GetRouting())

		serviceResp.Revision = firstResp.Service.Revision
		secondResp := discoverSuit.server.GetRoutingConfigWithCache(discoverSuit.defaultCtx, serviceResp)
		if secondResp.GetService().GetRevision().GetValue() != serviceResp.GetRevision().GetValue() {
			t.Fatalf("error")
		}
		if secondResp.GetRouting() != nil {
			t.Fatalf("error: %+v", secondResp.GetRouting())
		}
		t.Logf("%+v", secondResp)
	})
	Convey("路由不存在，不会出异常", t, func() {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.initialize(); err != nil {
			t.Fatal(err)
		}
		defer discoverSuit.Destroy()

		_, serviceResp := discoverSuit.createCommonService(t, 10)
		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
		time.Sleep(discoverSuit.updateCacheInterval)
		if resp := discoverSuit.server.GetRoutingConfigWithCache(discoverSuit.defaultCtx, serviceResp); !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

// test对routing字段进行校验
func TestCheckRoutingFieldLen(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	req := &api.Routing{
		ServiceToken: utils.NewStringValue("test"),
		Service:      utils.NewStringValue("test"),
		Namespace:    utils.NewStringValue("default"),
	}

	t.Run("创建路由规则，服务名超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldName := req.Service
		req.Service = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateRoutingConfigs(discoverSuit.defaultCtx, []*api.Routing{req})
		req.Service = oldName
		if resp.Code.Value != api.InvalidServiceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("创建路由规则，命名空间超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldNamespace := req.Namespace
		req.Namespace = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateRoutingConfigs(discoverSuit.defaultCtx, []*api.Routing{req})
		req.Namespace = oldNamespace
		if resp.Code.Value != api.InvalidNamespaceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("创建路由规则，toeken超长", func(t *testing.T) {
		str := genSpecialStr(2049)
		oldServiceToken := req.ServiceToken
		req.ServiceToken = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateRoutingConfigs(discoverSuit.defaultCtx, []*api.Routing{req})
		req.ServiceToken = oldServiceToken
		if resp.Code.Value != api.InvalidServiceToken {
			t.Fatalf("%+v", resp)
		}
	})
}
