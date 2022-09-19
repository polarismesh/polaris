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

// import (
// 	"testing"
// 	"time"

// 	. "github.com/smartystreets/goconvey/convey"

// 	api "github.com/polarismesh/polaris-server/common/api/v1"
// 	apiv2 "github.com/polarismesh/polaris-server/common/api/v2"
// 	"github.com/polarismesh/polaris-server/common/utils"
// )

// // checkSameRoutingConfigV2 检查routingConfig前后是否一致
// func checkSameRoutingConfigV2V2(t *testing.T, lhs []*apiv2.Routing, rhs []*apiv2.Routing) {
// 	if len(lhs) != len(rhs) {
// 		t.Fatal("error: len(lhs) != len(rhs)")
// 	}
// }

// // TestCreateRoutingConfigV2 测试创建路由配置
// func TestCreateRoutingConfigV2(t *testing.T) {

// 	discoverSuit := &DiscoverTestSuit{}
// 	if err := discoverSuit.initialize(); err != nil {
// 		t.Fatal(err)
// 	}
// 	defer discoverSuit.Destroy()

// 	Convey("正常创建路由配置配置请求", t, func() {
// 		req := discoverSuit.createCommonRoutingConfigV2(t, 3)

// 		// 对写进去的数据进行查询
// 		time.Sleep(discoverSuit.updateCacheInterval)
// 		out := discoverSuit.server.GetRoutingConfigsV2(discoverSuit.defaultCtx, map[string]string{
// 			"limit":  "100",
// 			"offset": "0",
// 		})
// 		defer discoverSuit.cleanCommonRoutingConfigV2(req)
// 		if !respSuccessV2(out) {
// 			t.Fatalf("error: %+v", out)
// 		}
// 	})

// 	Convey("同一个服务重复创建路由配置，报错", t, func() {
// 		_, serviceResp := discoverSuit.createCommonService(t, 10)
// 		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

// 		req, _ := discoverSuit.createCommonRoutingConfig(t, serviceResp, 1, 0)
// 		defer discoverSuit.cleanCommonRoutingConfig(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

// 		resp := discoverSuit.server.CreateRoutingConfigs(discoverSuit.defaultCtx, []*api.Routing{req})
// 		So(respSuccess(resp), ShouldEqual, false)
// 		t.Logf("%s", resp.GetInfo().GetValue())
// 	})
// }

// // TestCreateRoutingConfig2V2 测试创建路由配置
// func TestCreateRoutingConfig2V2(t *testing.T) {

// 	discoverSuit := &DiscoverTestSuit{}
// 	if err := discoverSuit.initialize(); err != nil {
// 		t.Fatal(err)
// 	}
// 	defer discoverSuit.Destroy()

// 	Convey("参数缺失，报错", t, func() {
// 		_, serviceResp := discoverSuit.createCommonService(t, 20)
// 		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

// 		req := &api.Routing{}
// 		resp := discoverSuit.server.CreateRoutingConfigs(discoverSuit.defaultCtx, []*api.Routing{req})
// 		So(respSuccess(resp), ShouldEqual, false)
// 		t.Logf("%s", resp.GetInfo().GetValue())

// 		req.Service = serviceResp.Name
// 		resp = discoverSuit.server.CreateRoutingConfigs(discoverSuit.defaultCtx, []*api.Routing{req})
// 		So(respSuccess(resp), ShouldEqual, false)
// 		t.Logf("%s", resp.GetInfo().GetValue())

// 		req.Namespace = serviceResp.Namespace
// 		resp = discoverSuit.server.CreateRoutingConfigs(discoverSuit.defaultCtx, []*api.Routing{req})
// 		defer discoverSuit.cleanCommonRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())
// 		So(respSuccess(resp), ShouldEqual, true)
// 		t.Logf("%s", resp.GetInfo().GetValue())
// 	})

// 	Convey("服务不存在，创建路由配置，报错", t, func() {
// 		_, serviceResp := discoverSuit.createCommonService(t, 120)
// 		discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

// 		req := &api.Routing{}
// 		req.Service = serviceResp.Name
// 		req.Namespace = serviceResp.Namespace
// 		req.ServiceToken = serviceResp.Token
// 		resp := discoverSuit.server.CreateRoutingConfigs(discoverSuit.defaultCtx, []*api.Routing{req})
// 		So(respSuccess(resp), ShouldEqual, false)
// 		t.Logf("%s", resp.GetInfo().GetValue())
// 	})
// }

// // TestDeleteRoutingConfigV2 测试删除路由配置
// func TestDeleteRoutingConfigV2(t *testing.T) {

// 	discoverSuit := &DiscoverTestSuit{}
// 	if err := discoverSuit.initialize(); err != nil {
// 		t.Fatal(err)
// 	}
// 	defer discoverSuit.Destroy()

// 	Convey("可以正常删除路由配置", t, func() {
// 		_, serviceResp := discoverSuit.createCommonService(t, 100)
// 		defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

// 		_, resp := discoverSuit.createCommonRoutingConfig(t, serviceResp, 3, 0)
// 		discoverSuit.deleteCommonRoutingConfig(t, resp)
// 		defer discoverSuit.cleanCommonRoutingConfig(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

// 		// 删除之后，数据不见
// 		time.Sleep(discoverSuit.updateCacheInterval)
// 		out := discoverSuit.server.GetRoutingConfigWithCache(discoverSuit.defaultCtx, serviceResp)
// 		So(out.GetRouting(), ShouldBeNil)
// 	})
// }

// // TestUpdateRoutingConfigV2 测试更新路由配置
// func TestUpdateRoutingConfigV2(t *testing.T) {

// 	discoverSuit := &DiscoverTestSuit{}
// 	if err := discoverSuit.initialize(); err != nil {
// 		t.Fatal(err)
// 	}
// 	defer discoverSuit.Destroy()

// 	Convey("可以正常更新路由配置", t, func() {
// 		_, serviceResp := discoverSuit.createCommonService(t, 50)
// 		serviceName := serviceResp.GetName().GetValue()
// 		namespace := serviceResp.GetNamespace().GetValue()
// 		defer discoverSuit.cleanServiceName(serviceName, namespace)

// 		_, resp := discoverSuit.createCommonRoutingConfig(t, serviceResp, 2, 0)
// 		defer discoverSuit.cleanCommonRoutingConfig(serviceName, namespace)
// 		resp.ServiceToken = utils.NewStringValue(serviceResp.GetToken().GetValue())

// 		resp.Outbounds = resp.Inbounds
// 		resp.Inbounds = make([]*api.Route, 0)
// 		discoverSuit.updateCommonRoutingConfig(t, resp)

// 		time.Sleep(discoverSuit.updateCacheInterval)
// 		out := discoverSuit.server.GetRoutingConfigWithCache(discoverSuit.defaultCtx, serviceResp)
// 		checkSameRoutingConfig(t, resp, out.GetRouting())
// 	})
// }

// // test对routing字段进行校验
// func TestCheckRoutingFieldLenV2(t *testing.T) {

// 	discoverSuit := &DiscoverTestSuit{}
// 	if err := discoverSuit.initialize(); err != nil {
// 		t.Fatal(err)
// 	}
// 	defer discoverSuit.Destroy()

// 	req := &api.Routing{
// 		ServiceToken: utils.NewStringValue("test"),
// 		Service:      utils.NewStringValue("test"),
// 		Namespace:    utils.NewStringValue("default"),
// 	}

// 	t.Run("创建路由规则，服务名超长", func(t *testing.T) {
// 		str := genSpecialStr(129)
// 		oldName := req.Service
// 		req.Service = utils.NewStringValue(str)
// 		resp := discoverSuit.server.CreateRoutingConfigsV2(discoverSuit.defaultCtx, []*api.Routing{req})
// 		req.Service = oldName
// 		if resp.Code.Value != api.InvalidServiceName {
// 			t.Fatalf("%+v", resp)
// 		}
// 	})
// 	t.Run("创建路由规则，命名空间超长", func(t *testing.T) {
// 		str := genSpecialStr(129)
// 		oldNamespace := req.Namespace
// 		req.Namespace = utils.NewStringValue(str)
// 		resp := discoverSuit.server.CreateRoutingConfigsV2(discoverSuit.defaultCtx, []*api.Routing{req})
// 		req.Namespace = oldNamespace
// 		if resp.Code.Value != api.InvalidNamespaceName {
// 			t.Fatalf("%+v", resp)
// 		}
// 	})
// }
