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

// 测试缓存获取路由配置
func TestGetRoutingConfigWithCache(t *testing.T) {
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

// Test_RouteRule_V1_Server
func Test_RouteRule_V1_Server(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("Create", func(t *testing.T) {
		rsp := discoverSuit.DiscoverServer().CreateRoutingConfigs(discoverSuit.DefaultCtx, []*apitraffic.Routing{
			&apitraffic.Routing{},
		})
		assert.False(t, api.IsSuccess(rsp), rsp.GetInfo().GetValue())
	})

	t.Run("Update", func(t *testing.T) {
		rsp := discoverSuit.DiscoverServer().UpdateRoutingConfigs(discoverSuit.DefaultCtx, []*apitraffic.Routing{
			&apitraffic.Routing{},
		})
		assert.False(t, api.IsSuccess(rsp), rsp.GetInfo().GetValue())
	})

	t.Run("Delete", func(t *testing.T) {
		rsp := discoverSuit.DiscoverServer().UpdateRoutingConfigs(discoverSuit.DefaultCtx, []*apitraffic.Routing{
			&apitraffic.Routing{},
		})
		assert.False(t, api.IsSuccess(rsp), rsp.GetInfo().GetValue())
	})

	t.Run("Get", func(t *testing.T) {
		rsp := discoverSuit.DiscoverServer().GetRoutingConfigs(discoverSuit.DefaultCtx, map[string]string{})
		assert.False(t, api.IsSuccess(rsp), rsp.GetInfo().GetValue())
	})
}
