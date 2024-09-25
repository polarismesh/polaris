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

	"github.com/golang/protobuf/ptypes"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
)

// checkSameRoutingConfigV2 检查routingConfig前后是否一致
func checkSameRoutingConfigV2V2(t *testing.T, lhs []*apitraffic.RouteRule, rhs []*apitraffic.RouteRule) {
	if len(lhs) != len(rhs) {
		t.Fatal("error: len(lhs) != len(rhs)")
	}
}

// TestCreateRoutingConfigV2 测试创建路由配置
func TestCreateRoutingConfigV2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("正常创建路由配置配置请求", func(t *testing.T) {
		req := discoverSuit.createCommonRoutingConfigV2(t, 3)
		defer discoverSuit.truncateCommonRoutingConfigV2()

		// 对写进去的数据进行查询
		_ = discoverSuit.CacheMgr().TestUpdate()
		out := discoverSuit.DiscoverServer().QueryRoutingConfigsV2(discoverSuit.DefaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
		})
		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}

		ret, _ := unmarshalRoutingV2toAnySlice(out.GetData())
		t.Logf("query routing v2 : %#v", ret)

		// assert.Equal(t, int(3), int(out.Amount), "query routing size")

		// 按照名字查询

		out = discoverSuit.DiscoverServer().QueryRoutingConfigsV2(discoverSuit.DefaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
			"name":   req[0].Name,
		})
		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}
		rulesV2, err := unmarshalRoutingV2toAnySlice(out.GetData())
		assert.NoError(t, err)
		for i := range rulesV2 {
			item := rulesV2[i]
			msg := &apitraffic.RuleRoutingConfig{}
			err := ptypes.UnmarshalAny(item.GetRoutingConfig(), msg)
			assert.NoError(t, err)
			assert.True(t, len(msg.GetSources()) == 0, "RuleRoutingConfig.Sources len != 0")
			assert.True(t, len(msg.GetDestinations()) == 0, "RuleRoutingConfig.Destinations len != 0")
			assert.True(t, len(msg.GetRules()) != 0, "RuleRoutingConfig.Rules len == 0")
		}

		assert.Equal(t, int(1), int(out.Amount.GetValue()), "query routing size")

		item, err := service.Api2RoutingConfigV2(req[0])
		assert.NoError(t, err)
		expendItem, err := item.ToExpendRoutingConfig()
		assert.NoError(t, err)

		// 基于服务信息查询
		out = discoverSuit.DiscoverServer().QueryRoutingConfigsV2(discoverSuit.DefaultCtx, map[string]string{
			"limit":     "100",
			"offset":    "0",
			"namespace": expendItem.RuleRouting.RuleRouting.Rules[0].Sources[0].Namespace,
			"service":   expendItem.RuleRouting.RuleRouting.Rules[0].Sources[0].Service,
		})
		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}
		rulesV2, err = unmarshalRoutingV2toAnySlice(out.GetData())
		assert.NoError(t, err)
		for i := range rulesV2 {
			item := rulesV2[i]
			msg := &apitraffic.RuleRoutingConfig{}
			err := ptypes.UnmarshalAny(item.GetRoutingConfig(), msg)
			assert.NoError(t, err)
			assert.True(t, len(msg.GetSources()) == 0, "RuleRoutingConfig.Sources len != 0")
			assert.True(t, len(msg.GetDestinations()) == 0, "RuleRoutingConfig.Destinations len != 0")
			assert.True(t, len(msg.GetRules()) != 0, "RuleRoutingConfig.Rules len == 0")
		}
	})
}

// TestCompatibleRoutingConfigV2AndV1 测试V2版本的路由规则和V1版本的路由规则
func TestCompatibleRoutingConfigV2AndV1(t *testing.T) {

	svc := &apiservice.Service{
		Name:      utils.NewStringValue("compatible-routing"),
		Namespace: utils.NewStringValue("compatible"),
	}

	initSuitFunc := func(t *testing.T) *DiscoverTestSuit {
		discoverSuit := &DiscoverTestSuit{}
		if err := discoverSuit.Initialize(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			discoverSuit.Destroy()
		})

		createSvcResp := discoverSuit.DiscoverServer().CreateServices(discoverSuit.DefaultCtx, []*apiservice.Service{svc})
		if !respSuccess(createSvcResp) {
			t.Fatalf("error: %s", createSvcResp.GetInfo().GetValue())
		}

		_ = createSvcResp.Responses[0].GetService()
		t.Cleanup(func() {
			discoverSuit.cleanServices([]*apiservice.Service{svc})
		})
		return discoverSuit
	}

	t.Run("V1的存量规则-走V2接口可以查询到，ExtendInfo符合要求", func(t *testing.T) {
		discoverSuit := initSuitFunc(t)
		_, _ = discoverSuit.createCommonRoutingConfigV1IntoOldStore(t, svc, 3, 0)
		t.Cleanup(func() {
			discoverSuit.cleanCommonRoutingConfig(svc.GetName().GetValue(), svc.GetNamespace().GetValue())
			discoverSuit.truncateCommonRoutingConfigV2()
		})

		_ = discoverSuit.CacheMgr().TestUpdate()
		// 从缓存中查询应该查到 3+3 条 v2 的路由规则
		out := discoverSuit.DiscoverServer().QueryRoutingConfigsV2(discoverSuit.DefaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
		})
		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}
		assert.Equal(t, int(3), int(out.GetAmount().GetValue()), "query routing size")

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

	t.Run("V1的存量规则-走v2规则的启用可正常迁移v1规则", func(t *testing.T) {
		discoverSuit := initSuitFunc(t)
		_, _ = discoverSuit.createCommonRoutingConfigV1IntoOldStore(t, svc, 3, 0)
		t.Cleanup(func() {
			discoverSuit.cleanCommonRoutingConfig(svc.GetName().GetValue(), svc.GetNamespace().GetValue())
			discoverSuit.truncateCommonRoutingConfigV2()
		})

		_ = discoverSuit.CacheMgr().TestUpdate()
		// 从缓存中查询应该查到 3 条 v2 的路由规则
		out := discoverSuit.DiscoverServer().QueryRoutingConfigsV2(discoverSuit.DefaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
		})
		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}
		assert.Equal(t, int(3), int(out.GetAmount().GetValue()), "query routing size")
		rulesV2, err := unmarshalRoutingV2toAnySlice(out.GetData())
		assert.NoError(t, err)

		// 选择其中一条规则进行enable操作
		v2resp := discoverSuit.DiscoverServer().EnableRoutings(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{rulesV2[0]})
		if !respSuccess(v2resp) {
			t.Fatalf("error: %+v", v2resp)
		}
		// 直接查询存储无法查询到 v1 的路由规则
		total, routingsV1, err := discoverSuit.Storage.GetRoutingConfigs(map[string]string{}, 0, 100)
		assert.NoError(t, err, err)
		assert.Equal(t, uint32(0), total, "v1 routing must delete and transfer to v1")
		assert.Equal(t, 0, len(routingsV1), "v1 routing ret len need zero")

		// 从缓存中查询应该查到 3 条 v2 的路由规则
		out = discoverSuit.DiscoverServer().QueryRoutingConfigsV2(discoverSuit.DefaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
		})
		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}
		assert.Equal(t, int(3), int(out.GetAmount().GetValue()), "query routing size")
		rulesV2, err = unmarshalRoutingV2toAnySlice(out.GetData())
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

	t.Run("V1的存量规则-走v2规则的删除可正常迁移v1规则", func(t *testing.T) {
		discoverSuit := initSuitFunc(t)
		_, _ = discoverSuit.createCommonRoutingConfigV1IntoOldStore(t, svc, 3, 0)
		t.Cleanup(func() {
			discoverSuit.cleanCommonRoutingConfig(svc.GetName().GetValue(), svc.GetNamespace().GetValue())
			discoverSuit.truncateCommonRoutingConfigV2()
		})

		_ = discoverSuit.CacheMgr().TestUpdate()
		// 从缓存中查询应该查到 3+3 条 v2 的路由规则
		out := discoverSuit.DiscoverServer().QueryRoutingConfigsV2(discoverSuit.DefaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
		})
		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}
		assert.Equal(t, int(3), int(out.GetAmount().GetValue()), "query routing size")

		rulesV2, err := unmarshalRoutingV2toAnySlice(out.GetData())
		assert.NoError(t, err)

		// 选择其中一条规则进行删除操作
		v2resp := discoverSuit.DiscoverServer().DeleteRoutingConfigsV2(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{rulesV2[0]})
		if !respSuccess(v2resp) {
			t.Fatalf("error: %+v", v2resp)
		}
		// 直接查询存储无法查询到 v1 的路由规则
		total, routingsV1, err := discoverSuit.Storage.GetRoutingConfigs(map[string]string{}, 0, 100)
		assert.NoError(t, err, err)
		assert.Equal(t, uint32(0), total, "v1 routing must delete and transfer to v1")
		assert.Equal(t, 0, len(routingsV1), "v1 routing ret len need zero")

		// 查询对应的 v2 规则也查询不到
		ruleV2, err := discoverSuit.Storage.GetRoutingConfigV2WithID(rulesV2[0].Id)
		assert.NoError(t, err, err)
		assert.Nil(t, ruleV2, "v2 routing must delete")

		_ = discoverSuit.CacheMgr().TestUpdate()
		// 从缓存中查询应该查到 2 条 v2 的路由规则
		out = discoverSuit.DiscoverServer().QueryRoutingConfigsV2(discoverSuit.DefaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
		})
		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}
		assert.Equal(t, int(2), int(out.GetAmount().GetValue()), "query routing size")
		rulesV2, err = unmarshalRoutingV2toAnySlice(out.GetData())
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

	t.Run("V1的存量规则-走v2规则的编辑可正常迁移v1规则", func(t *testing.T) {
		discoverSuit := initSuitFunc(t)
		_, _ = discoverSuit.createCommonRoutingConfigV1IntoOldStore(t, svc, 3, 0)
		t.Cleanup(func() {
			discoverSuit.cleanCommonRoutingConfig(svc.GetName().GetValue(), svc.GetNamespace().GetValue())
			discoverSuit.truncateCommonRoutingConfigV2()
		})

		_ = discoverSuit.CacheMgr().TestUpdate()
		// 从缓存中查询应该查到 3+3 条 v2 的路由规则
		out := discoverSuit.DiscoverServer().QueryRoutingConfigsV2(discoverSuit.DefaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
		})
		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}
		assert.Equal(t, int(3), int(out.GetAmount().GetValue()), "query routing size")

		rulesV2, err := unmarshalRoutingV2toAnySlice(out.GetData())
		assert.NoError(t, err)

		// 需要将 v2 规则的 extendInfo 规则清理掉
		// 选择其中一条规则进行enable操作
		rulesV2[0].Description = "update v2 rule and transfer v1 to v2"
		v2resp := discoverSuit.DiscoverServer().UpdateRoutingConfigsV2(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{rulesV2[0]})
		if !respSuccess(v2resp) {
			t.Fatalf("error: %+v", v2resp)
		}
		// 直接查询存储无法查询到 v1 的路由规则
		total, routingsV1, err := discoverSuit.Storage.GetRoutingConfigs(map[string]string{}, 0, 100)
		assert.NoError(t, err, err)
		assert.Equal(t, uint32(0), total, "v1 routing must delete and transfer to v1")
		assert.Equal(t, 0, len(routingsV1), "v1 routing ret len need zero")

		// 查询对应的 v2 规则能够查询到
		ruleV2, err := discoverSuit.Storage.GetRoutingConfigV2WithID(rulesV2[0].Id)
		assert.NoError(t, err, err)
		assert.NotNil(t, ruleV2, "v2 routing must exist")
		assert.Equal(t, rulesV2[0].Description, ruleV2.Description)

		_ = discoverSuit.CacheMgr().TestUpdate()
		out = discoverSuit.DiscoverServer().QueryRoutingConfigsV2(discoverSuit.DefaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
		})
		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}
		assert.Equal(t, int(3), int(out.GetAmount().GetValue()), "query routing size")
		rulesV2, err = unmarshalRoutingV2toAnySlice(out.GetData())
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

// TestDeleteRoutingConfigV2 测试删除路由配置
func TestDeleteRoutingConfigV2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("可以正常删除路由配置", func(t *testing.T) {
		resp := discoverSuit.createCommonRoutingConfigV2(t, 1)
		discoverSuit.deleteCommonRoutingConfigV2(t, resp[0])
		defer discoverSuit.cleanCommonRoutingConfigV2(resp)

		serviceName := fmt.Sprintf("in-source-service-%d", 0)
		namespaceName := fmt.Sprintf("in-source-service-%d", 0)

		// 删除之后，数据不见
		_ = discoverSuit.CacheMgr().TestUpdate()
		out := discoverSuit.DiscoverServer().GetRoutingConfigWithCache(discoverSuit.DefaultCtx, &apiservice.Service{
			Name:      utils.NewStringValue(serviceName),
			Namespace: utils.NewStringValue(namespaceName),
		})

		noExist := out.GetRouting() == nil ||
			((len(out.GetRouting().Inbounds) == 0 && len(out.GetRouting().GetOutbounds()) == 0) ||
				len(out.Routing.GetRules()) == 0)
		assert.True(t, noExist)
	})
}

// TestUpdateRoutingConfigV2 测试更新路由配置
func TestUpdateRoutingConfigV2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	t.Run("可以正常更新路由配置", func(t *testing.T) {
		req := discoverSuit.createCommonRoutingConfigV2(t, 1)
		defer discoverSuit.cleanCommonRoutingConfigV2(req)
		// 对写进去的数据进行查询
		_ = discoverSuit.CacheMgr().TestUpdate()
		out := discoverSuit.DiscoverServer().QueryRoutingConfigsV2(discoverSuit.DefaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
		})
		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}

		assert.Equal(t, uint32(1), out.Size.GetValue(), "query routing size")

		ret, err := unmarshalRoutingV2toAnySlice(out.GetData())
		assert.NoError(t, err)
		routing := ret[0]

		updateName := "update routing second"
		routing.Name = updateName

		discoverSuit.DiscoverServer().UpdateRoutingConfigsV2(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{routing})
		_ = discoverSuit.CacheMgr().TestUpdate()
		out = discoverSuit.DiscoverServer().QueryRoutingConfigsV2(discoverSuit.DefaultCtx, map[string]string{
			"limit":  "100",
			"offset": "0",
			"id":     routing.Id,
		})

		if !respSuccess(out) {
			t.Fatalf("error: %+v", out)
		}

		assert.Equal(t, uint32(1), out.Size.GetValue(), "query routing size")
		ret, err = unmarshalRoutingV2toAnySlice(out.GetData())
		assert.NoError(t, err)
		assert.Equal(t, updateName, ret[0].Name)
	})
}

// test对routing字段进行校验
func TestCreateCheckRoutingFieldLenV2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	any, _ := ptypes.MarshalAny(&apitraffic.RuleRoutingConfig{})

	req := &apitraffic.RouteRule{
		Id:            "",
		Name:          "test-routing",
		Namespace:     "",
		Enable:        false,
		RoutingPolicy: apitraffic.RoutingPolicy_RulePolicy,
		RoutingConfig: any,
		Revision:      "",
		Ctime:         "",
		Mtime:         "",
		Etime:         "",
		Priority:      0,
		Description:   "",
	}

	t.Run("创建路由规则，规则名称超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldName := req.Name
		req.Name = str
		resp := discoverSuit.DiscoverServer().CreateRoutingConfigsV2(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{req})
		req.Name = oldName
		if resp.Code.GetValue() != uint32(apimodel.Code_InvalidRoutingName) {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("创建路由规则，路由规则类型不正确", func(t *testing.T) {
		oldPolicy := req.RoutingPolicy
		req.RoutingPolicy = apitraffic.RoutingPolicy(123)
		resp := discoverSuit.DiscoverServer().CreateRoutingConfigsV2(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{req})
		req.RoutingPolicy = oldPolicy
		if resp.Code.GetValue() != uint32(apimodel.Code_InvalidRoutingPolicy) {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("创建路由规则，路由规则类型不正确", func(t *testing.T) {
		oldPolicy := req.RoutingPolicy
		req.RoutingPolicy = apitraffic.RoutingPolicy_MetadataPolicy
		resp := discoverSuit.DiscoverServer().CreateRoutingConfigsV2(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{req})
		req.RoutingPolicy = oldPolicy
		if resp.Code.GetValue() != uint32(apimodel.Code_InvalidRoutingPolicy) {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("创建路由规则，路由优先级不正确", func(t *testing.T) {
		oldPriority := req.Priority
		req.Priority = 11
		resp := discoverSuit.DiscoverServer().CreateRoutingConfigsV2(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{req})
		req.Priority = oldPriority
		if resp.Code.GetValue() != uint32(apimodel.Code_InvalidRoutingPriority) {
			t.Fatalf("%+v", resp)
		}
	})
}

// TestUpdateCheckRoutingFieldLenV2 test对routing字段进行校验
func TestUpdateCheckRoutingFieldLenV2(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	any, _ := ptypes.MarshalAny(&apitraffic.RuleRoutingConfig{})

	req := &apitraffic.RouteRule{
		Id:            "12312312312312313",
		Name:          "test-routing",
		Namespace:     "",
		Enable:        false,
		RoutingPolicy: apitraffic.RoutingPolicy_RulePolicy,
		RoutingConfig: any,
		Revision:      "",
		Ctime:         "",
		Mtime:         "",
		Etime:         "",
		Priority:      0,
		Description:   "",
	}

	t.Run("更新路由规则，规则ID为空", func(t *testing.T) {
		oldId := req.Id
		req.Id = ""
		resp := discoverSuit.DiscoverServer().UpdateRoutingConfigsV2(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{req})
		req.Id = oldId
		if resp.Code.GetValue() != uint32(apimodel.Code_InvalidRoutingID) {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("更新路由规则，规则名称超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldName := req.Name
		req.Name = str
		resp := discoverSuit.DiscoverServer().UpdateRoutingConfigsV2(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{req})
		req.Name = oldName
		if resp.Code.GetValue() != uint32(apimodel.Code_InvalidRoutingName) {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("更新路由规则，路由规则类型不正确", func(t *testing.T) {
		oldPolicy := req.RoutingPolicy
		req.RoutingPolicy = apitraffic.RoutingPolicy(123)
		resp := discoverSuit.DiscoverServer().UpdateRoutingConfigsV2(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{req})
		req.RoutingPolicy = oldPolicy
		if resp.Code.GetValue() != uint32(apimodel.Code_InvalidRoutingPolicy) {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("更新路由规则，路由规则类型不正确", func(t *testing.T) {
		oldPolicy := req.RoutingPolicy
		req.RoutingPolicy = apitraffic.RoutingPolicy_MetadataPolicy
		resp := discoverSuit.DiscoverServer().UpdateRoutingConfigsV2(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{req})
		req.RoutingPolicy = oldPolicy
		if resp.Code.GetValue() != uint32(apimodel.Code_InvalidRoutingPolicy) {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("更新路由规则，路由优先级不正确", func(t *testing.T) {
		oldPriority := req.Priority
		req.Priority = 11
		resp := discoverSuit.DiscoverServer().UpdateRoutingConfigsV2(discoverSuit.DefaultCtx, []*apitraffic.RouteRule{req})
		req.Priority = oldPriority
		if resp.Code.GetValue() != uint32(apimodel.Code_InvalidRoutingPriority) {
			t.Fatalf("%+v", resp)
		}
	})
}

// marshalRoutingV2toAnySlice 转换为 []*apitraffic.RouteRule 数组
func unmarshalRoutingV2toAnySlice(routings []*anypb.Any) ([]*apitraffic.RouteRule, error) {
	ret := make([]*apitraffic.RouteRule, 0, len(routings))

	for i := range routings {
		entry := routings[i]

		msg := &apitraffic.RouteRule{}
		if err := ptypes.UnmarshalAny(entry, msg); err != nil {
			return nil, err
		}

		ret = append(ret, msg)
	}

	return ret, nil
}

func Test_PrintRouteRuleTypeUrl(t *testing.T) {
	any, _ := ptypes.MarshalAny(&apitraffic.RuleRoutingConfig{})
	t.Log(any.TypeUrl)
}
