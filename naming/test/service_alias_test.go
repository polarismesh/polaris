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

package test

import (
	"context"
	"fmt"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/naming"
	. "github.com/smartystreets/goconvey/convey"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// 创建一个服务别名
func createCommonAlias(service *api.Service, alias string, typ api.AliasType) *api.Response {
	req := &api.ServiceAlias{
		Service:      service.Name,
		Namespace:    service.Namespace,
		Alias:        utils.NewStringValue(alias),
		Type:         typ,
		ServiceToken: service.Token,
	}
	return server.CreateServiceAlias(defaultCtx, req)
}

// 创建一个服务别名（不需要鉴权）
func createCommonAliasNoAuth(service *api.Service, alias string, typ api.AliasType) *api.Response {
	req := &api.ServiceAlias{
		Service:   service.Name,
		Namespace: service.Namespace,
		Alias:     utils.NewStringValue(alias),
		Type:      typ,
		Owners:    utils.NewStringValue("alias-owner"),
		Comment:   utils.NewStringValue("comment"),
	}
	return server.CreateServiceAliasNoAuth(defaultCtx, req)
}

// 创建别名，并检查
func createCommonAliasCheck(t *testing.T, service *api.Service, alias string,
	typ api.AliasType) *api.Response {
	resp := createCommonAlias(service, alias, typ)
	if !respSuccess(resp) {
		t.Fatalf("error")
	}
	return resp
}

// 检查一个服务别名是否是sid
func isSid(alias string) bool {
	items := strings.Split(alias, ":")
	if len(items) != 2 {
		return false
	}

	for _, it := range items {
		if ok, _ := regexp.MatchString("^[0-9]+$", it); !ok {
			return false
		}
	}

	return true
}

// 正常场景测试
func TestCreateServiceAlias(t *testing.T) {
	_, serviceResp := createCommonService(t, 123)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	Convey("正常创建非Sid的别名", t, func() {
		alias := fmt.Sprintf("alias.%d", time.Now().Unix())
		resp := createCommonAlias(serviceResp, alias, api.AliasType_DEFAULT)
		defer cleanServiceName(alias, serviceResp.GetNamespace().GetValue())
		So(respSuccess(resp), ShouldEqual, true)
		So(resp.Alias.Alias.Value, ShouldEqual, alias)
	})

	Convey("正常创建Sid别名", t, func() {
		resp := createCommonAlias(serviceResp, "", api.AliasType_CL5SID)
		So(respSuccess(resp), ShouldEqual, true)
		defer cleanServiceName(resp.Alias.Alias.Value, serviceResp.GetNamespace().GetValue())
		So(isSid(resp.Alias.Alias.Value), ShouldEqual, true)
		t.Logf("alias sid: %s", resp.Alias.Alias.Value)
	})

	Convey("使用ctx带上的token可以创建成功", t, func() {
		req := &api.ServiceAlias{
			Service:   serviceResp.Name,
			Namespace: serviceResp.Namespace,
			Type:      api.AliasType_CL5SID,
		}
		ctx := context.WithValue(defaultCtx, utils.StringContext("polaris-token"),
			serviceResp.GetToken().GetValue())
		resp := server.CreateServiceAlias(ctx, req)
		So(respSuccess(resp), ShouldEqual, true)
		cleanServiceName(resp.Alias.Alias.Value, serviceResp.GetNamespace().GetValue())

		// 带上系统token，也可以成功
		ctx = context.WithValue(defaultCtx, utils.StringContext("polaris-token"),
			"polaris@12345678")
		resp = server.CreateServiceAlias(ctx, req)
		So(respSuccess(resp), ShouldEqual, true)
		cleanServiceName(resp.Alias.Alias.Value, serviceResp.GetNamespace().GetValue())
	})
	Convey("不允许为别名创建别名", t, func() {
		resp := createCommonAliasCheck(t, serviceResp, "", api.AliasType_CL5SID)
		defer cleanServiceName(resp.Alias.Alias.Value, serviceResp.Namespace.Value)

		service := &api.Service{
			Name:      resp.Alias.Alias,
			Namespace: serviceResp.Namespace,
			Token:     serviceResp.Token,
		}
		repeatedResp := createCommonAlias(service, "", api.AliasType_CL5SID)
		if respSuccess(repeatedResp) {
			t.Fatalf("error: %+v", repeatedResp)
		}
		t.Logf("%+v", repeatedResp)
	})
}

// 测试不需要鉴权的创建服务别名场景
func TestCreateServiceAliasNoAuth(t *testing.T) {
	_, serviceResp := createCommonService(t, 1)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	Convey("正常创建非Sid的别名（新接口）", t, func() {
		alias := fmt.Sprintf("alias.%d", time.Now().Unix())
		resp := createCommonAliasNoAuth(serviceResp, alias, api.AliasType_DEFAULT)
		defer cleanServiceName(alias, serviceResp.GetNamespace().GetValue())
		So(respSuccess(resp), ShouldEqual, true)
		So(resp.GetAlias().GetAlias().GetValue(), ShouldEqual, alias)
	})

	Convey("正常创建Sid的别名（新接口）", t, func() {
		resp := createCommonAliasNoAuth(serviceResp, "", api.AliasType_CL5SID)
		defer cleanServiceName(resp.GetAlias().GetAlias().GetValue(), serviceResp.GetNamespace().GetValue())
		So(respSuccess(resp), ShouldEqual, true)
		So(isSid(resp.Alias.Alias.Value), ShouldEqual, true)
		t.Logf("alias sid: %s", resp.Alias.Alias.Value)
	})

	Convey("不允许为别名创建别名（新接口）", t, func() {
		resp := createCommonAliasCheck(t, serviceResp, "", api.AliasType_CL5SID)
		defer cleanServiceName(resp.Alias.Alias.Value, serviceResp.Namespace.Value)

		service := &api.Service{
			Name:      resp.Alias.Alias,
			Namespace: serviceResp.Namespace,
		}
		repeatedResp := createCommonAliasNoAuth(service, "", api.AliasType_CL5SID)
		if respSuccess(repeatedResp) {
			t.Fatalf("error: %+v", repeatedResp)
		}
		t.Logf("%+v", repeatedResp)
	})
}

// 重点测试创建sid别名的场景
// 注意：该测试函数出错的情况，会遗留一些测试数据无法清理 TODO
func TestCreateSid(t *testing.T) {
	Convey("创建不同命名空间的sid，可以返回符合规范的sid", t, func() {
		for namespace, layout := range naming.Namespace2SidLayoutID {
			service := &api.Service{
				Name:      utils.NewStringValue("sid-test-xxx"),
				Namespace: utils.NewStringValue(namespace),
				Revision:  utils.NewStringValue("revision111"),
				Owners:    utils.NewStringValue("owners111"),
			}
			cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
			serviceResp := server.CreateService(defaultCtx, service)
			So(respSuccess(serviceResp), ShouldEqual, true)

			aliasResp := createCommonAlias(serviceResp.Service, "", api.AliasType_CL5SID)
			So(respSuccess(aliasResp), ShouldEqual, true)
			modID, cmdID := parseStr2Sid(aliasResp.GetAlias().GetAlias().GetValue())
			So(modID, ShouldNotEqual, uint32(0))
			So(cmdID, ShouldNotEqual, uint32(0))
			So(modID>>6, ShouldBeGreaterThanOrEqualTo, 3000001) // module
			So(modID&63, ShouldEqual, layout)                   // 根据保留字段标识服务名
			So(aliasResp.GetAlias().GetNamespace().GetValue(), ShouldEqual, namespace)
			cleanServiceName(aliasResp.GetAlias().GetAlias().GetValue(), namespace)
			cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		}
	})
	Convey("非默认的5个命名空间，不允许创建sid别名", t, func() {
		namespace := &api.Namespace{
			Name:   utils.NewStringValue("other-namespace-xxx"),
			Owners: utils.NewStringValue("aaa"),
		}
		So(respSuccess(server.CreateNamespace(defaultCtx, namespace)), ShouldEqual, true)
		defer cleanNamespace(namespace.Name.Value)

		service := &api.Service{
			Name:      utils.NewStringValue("sid-test-xxx"),
			Namespace: utils.NewStringValue(namespace.Name.Value),
			Revision:  utils.NewStringValue("revision111"),
			Owners:    utils.NewStringValue("owners111"),
		}
		serviceResp := server.CreateService(defaultCtx, service)
		So(respSuccess(serviceResp), ShouldEqual, true)
		defer cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		aliasResp := createCommonAlias(serviceResp.Service, "", api.AliasType_CL5SID)
		So(respSuccess(aliasResp), ShouldEqual, false)
		t.Logf("%s", aliasResp.GetInfo().GetValue())
	})
}

// 并发测试
func TestConcurrencyCreateSid(t *testing.T) {
	_, serviceResp := createCommonService(t, 234)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	Convey("并发创建sid别名，sid不会重复", t, func() {
		c := 200
		var wg sync.WaitGroup
		resultCh := make(chan *api.Response, 1)
		results := make([]*api.Response, 0, 200)
		shutdown := make(chan struct{})

		go func() {
			for {
				select {
				case result := <-resultCh:
					results = append(results, result)
				case <-shutdown:
					t.Log("[Alias] concurrency function exit")
					return
				}
			}
		}()

		for i := 0; i < c; i++ {
			wg.Add(1)
			go func(index int) {
				defer func() {
					t.Logf("[Alias] finish creating alias sid func index(%d)", index)
					wg.Done()
				}()
				resp := createCommonAlias(serviceResp, "", api.AliasType_CL5SID)
				resultCh <- resp
			}(i)
		}

		wg.Wait()
		time.Sleep(time.Second)
		close(shutdown)

		repeated := make(map[string]bool)
		for i := 0; i < c; i++ {
			resp := results[i]
			So(respSuccess(resp), ShouldEqual, true)
			defer cleanServiceName(resp.Alias.Alias.Value, serviceResp.GetNamespace().GetValue())
			So(isSid(resp.Alias.Alias.Value), ShouldEqual, true)
			repeated[resp.Alias.Alias.Value] = true
		}
		// 检查是否重复，必须是200个
		So(len(repeated), ShouldEqual, c)
	})
}

// 异常测试
func TestExceptCreateAlias(t *testing.T) {
	_, serviceResp := createCommonService(t, 345)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	Convey("参数缺失，报错", t, func() {
		noService := &api.Service{}
		resp := createCommonAlias(noService, "x1.x2.x3", api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, false)

		noService.Name = utils.NewStringValue("123")
		resp = createCommonAlias(noService, "x1.x2.x3", api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, false)

		noService.Namespace = utils.NewStringValue("456")
		resp = createCommonAlias(noService, "x1.x2.x3", api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, false)

		noService.Token = utils.NewStringValue("567")
		resp = createCommonAlias(noService, "", api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, false)
		t.Logf("return code: %d", resp.Code.Value)
	})

	Convey("不存在的源服务，报错", t, func() {
		noService := &api.Service{
			Name:      utils.NewStringValue("my.service.2020.02.19"),
			Namespace: utils.NewStringValue("123123"),
			Token:     utils.NewStringValue("aaa"),
		}
		resp := createCommonAlias(noService, "x1.x2.x3", api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, false)
		t.Logf("return code: %d", resp.Code.Value)
		So(resp.Code.Value, ShouldEqual, api.NotFoundService)
	})

	Convey("同名alias，报错", t, func() {
		resp := createCommonAlias(serviceResp, "x1.x2.x3", api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, true)
		defer cleanServiceName(resp.Alias.Alias.Value, serviceResp.GetNamespace().GetValue())

		resp = createCommonAlias(serviceResp, "x1.x2.x3", api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, false)
		t.Logf("same alias return code: %d", resp.Code.Value)
	})

	Convey("鉴权失败，报错", t, func() {
		service := &api.Service{
			Name:      serviceResp.Name,
			Namespace: serviceResp.Namespace,
			Token:     utils.NewStringValue("123123123"),
		}
		resp := createCommonAlias(service, "x1.x2.x3", api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, false)
		t.Logf("error token, return code: %d", resp.Code.Value)
	})

	Convey("指向的服务不存在（新接口）", t, func() {
		_, serviceResp2 := createCommonService(t, 2)
		cleanServiceName(serviceResp2.GetName().GetValue(), serviceResp2.GetNamespace().GetValue())
		resp := createCommonAliasNoAuth(serviceResp2, "", api.AliasType_CL5SID)
		if respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		t.Logf("%+v", resp)
	})

	Convey("请求参数没有负责人（新接口）", t, func() {
		req := &api.ServiceAlias{
			Service:   serviceResp.GetName(),
			Namespace: serviceResp.GetNamespace(),
			Type:      api.AliasType_CL5SID,
		}
		resp := server.CreateServiceAliasNoAuth(defaultCtx, req)
		if respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		t.Logf("%+v", resp)
	})
}

// 别名修改的测试
func TestUpdateServiceAlias(t *testing.T) {
	_, serviceResp := createCommonService(t, 3)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	Convey("修改别名负责人", t, func() {
		resp := createCommonAliasNoAuth(serviceResp, "", api.AliasType_CL5SID)
		So(respSuccess(resp), ShouldEqual, true)
		defer cleanServiceName(resp.GetAlias().GetAlias().GetValue(), serviceResp.GetNamespace().GetValue())

		// 修改别名负责人
		req := &api.ServiceAlias{
			Service:      resp.GetAlias().GetService(),
			Namespace:    resp.GetAlias().GetNamespace(),
			Alias:        resp.GetAlias().GetAlias(),
			Owners:       utils.NewStringValue("alias-owner-new"),
			ServiceToken: resp.GetAlias().GetServiceToken(),
		}

		repeatedResp := server.UpdateServiceAlias(defaultCtx, req)
		So(respSuccess(repeatedResp), ShouldEqual, true)

		query := map[string]string{
			"alias":     req.GetAlias().GetValue(),
			"namespace": req.GetNamespace().GetValue(),
		}
		aliasResponse := server.GetServiceAliases(query)
		// 判断负责人是否一致
		So(aliasResponse.GetAliases()[0].GetOwners().GetValue(), ShouldEqual, "alias-owner-new")
		t.Logf("pass, owner is %v", aliasResponse.GetAliases()[0].GetOwners().GetValue())
	})

	Convey("修改指向服务", t, func() {
		resp := createCommonAliasNoAuth(serviceResp, "", api.AliasType_CL5SID)
		So(respSuccess(resp), ShouldEqual, true)
		defer cleanServiceName(resp.GetAlias().GetAlias().GetValue(), serviceResp.GetNamespace().GetValue())

		// 创建新的服务
		_, serviceResp2 := createCommonService(t, 4)
		defer cleanServiceName(serviceResp2.GetName().GetValue(), serviceResp2.GetNamespace().GetValue())

		// 修改别名指向
		req := &api.ServiceAlias{
			Service:      serviceResp2.GetName(),
			Namespace:    serviceResp2.GetNamespace(),
			Alias:        resp.GetAlias().GetAlias(),
			Owners:       resp.GetAlias().GetOwners(),
			Comment:      resp.GetAlias().GetComment(),
			ServiceToken: resp.GetAlias().GetServiceToken(),
		}

		repeatedResp := server.UpdateServiceAlias(defaultCtx, req)
		So(respSuccess(repeatedResp), ShouldEqual, true)

		query := map[string]string{
			"alias":     req.GetAlias().GetValue(),
			"namespace": req.GetNamespace().GetValue(),
		}
		aliasResponse := server.GetServiceAliases(query)
		// 判断指向服务是否一致
		So(aliasResponse.GetAliases()[0].GetService().GetValue(), ShouldEqual, serviceResp2.GetName().GetValue())
		t.Logf("pass, service is %v", aliasResponse.GetAliases()[0].GetService().GetValue())
	})

	Convey("要指向的服务不存在", t, func() {
		resp := createCommonAliasNoAuth(serviceResp, "", api.AliasType_CL5SID)
		So(respSuccess(resp), ShouldEqual, true)
		defer cleanServiceName(resp.GetAlias().GetAlias().GetValue(), serviceResp.GetNamespace().GetValue())

		// 创建新的服务并删除
		_, serviceResp2 := createCommonService(t, 4)
		cleanServiceName(serviceResp2.GetName().GetValue(), serviceResp2.GetNamespace().GetValue())

		// 修改别名指向
		req := &api.ServiceAlias{
			Service:      serviceResp2.GetName(),
			Namespace:    serviceResp2.GetNamespace(),
			Alias:        resp.GetAlias().GetAlias(),
			Owners:       resp.GetAlias().GetOwners(),
			Comment:      resp.GetAlias().GetComment(),
			ServiceToken: resp.GetAlias().GetServiceToken(),
		}
		repeatedResp := server.UpdateServiceAlias(defaultCtx, req)
		if respSuccess(repeatedResp) {
			t.Fatalf("error: %+v", repeatedResp)
		}
		t.Logf("%+v", repeatedResp)
	})

	Convey("鉴权失败", t, func() {
		resp := createCommonAliasNoAuth(serviceResp, "", api.AliasType_CL5SID)
		So(respSuccess(resp), ShouldEqual, true)
		defer cleanServiceName(resp.GetAlias().GetAlias().GetValue(), serviceResp.GetNamespace().GetValue())

		// 修改service token
		req := resp.GetAlias()
		req.ServiceToken = utils.NewStringValue("")
		repeatedResp := server.UpdateServiceAlias(defaultCtx, req)
		if respSuccess(repeatedResp) {
			t.Fatalf("error: %+v", repeatedResp)
		}
		t.Logf("%+v", repeatedResp)
	})
}

// 别名删除
func TestDeleteServiceAlias(t *testing.T) {
	_, serviceResp := createCommonService(t, 201)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	Convey("通过服务删除接口可以直接删除别名", t, func() {
		resp := createCommonAlias(serviceResp, "", api.AliasType_CL5SID)
		So(respSuccess(resp), ShouldEqual, true)
		defer cleanServiceName(resp.Alias.Alias.Value, serviceResp.Namespace.Value)

		service := &api.Service{
			Name:      resp.Alias.Alias,
			Namespace: serviceResp.Namespace,
			Token:     serviceResp.Token,
		}
		removeCommonServices(t, []*api.Service{service})

		query := map[string]string{"name": resp.Alias.Alias.Value}
		queryResp := server.GetServices(query)
		So(respSuccess(queryResp), ShouldEqual, true)
		So(len(queryResp.Services), ShouldEqual, 0)
	})

	Convey("通过ctx带上token，可以删除别名成功", t, func() {
		resp := createCommonAlias(serviceResp, "", api.AliasType_CL5SID)
		So(respSuccess(resp), ShouldEqual, true)
		defer cleanServiceName(resp.Alias.Alias.Value, serviceResp.Namespace.Value)

		service := &api.Service{Name: resp.Alias.Alias, Namespace: serviceResp.Namespace}
		ctx := context.WithValue(defaultCtx, utils.StringContext("polaris-token"),
			"polaris@12345678")
		So(respSuccess(server.DeleteService(ctx, service)), ShouldEqual, true)
	})
}

// 服务实例与服务路由关联测试
func TestServiceAliasRelated(t *testing.T) {
	_, serviceResp := createCommonService(t, 202)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	resp := createCommonAlias(serviceResp, "", api.AliasType_CL5SID)
	if !respSuccess(resp) {
		t.Fatalf("errror")
	}
	defer cleanServiceName(resp.Alias.Alias.Value, serviceResp.Namespace.Value)
	Convey("实例新建，不允许为别名新建实例", t, func() {
		instance := &api.Instance{
			Service:      resp.Alias.Alias,
			Namespace:    serviceResp.Namespace,
			ServiceToken: serviceResp.Token,
			Host:         utils.NewStringValue("1.12.123.132"),
			Port:         utils.NewUInt32Value(8080),
		}
		instanceResp := server.CreateInstance(defaultCtx, instance)
		So(respSuccess(instanceResp), ShouldEqual, false)
		t.Logf("alias create instance ret code(%d), msg(%s)",
			instanceResp.Code.Value, instanceResp.Info.Value)
	})
	Convey("实例Discover，别名查询实例，返回源服务的实例信息", t, func() {
		_, instanceResp := createCommonInstance(t, serviceResp, 123)
		defer cleanInstance(instanceResp.GetId().GetValue())

		time.Sleep(updateCacheInterval)
		service := &api.Service{Name: resp.Alias.Alias, Namespace: resp.Alias.Namespace}
		disResp := server.ServiceInstancesCache(defaultCtx, service)
		So(respSuccess(disResp), ShouldEqual, true)
		So(len(disResp.Instances), ShouldEqual, 1)
	})
	Convey("路由新建，不允许为别名新建路由", t, func() {
		routing := &api.Routing{
			Service:      resp.Alias.Alias,
			Namespace:    resp.Alias.Namespace,
			ServiceToken: serviceResp.Token,
			Inbounds:     make([]*api.Route, 0),
		}
		routingResp := server.CreateRoutingConfig(defaultCtx, routing)
		So(respSuccess(routingResp), ShouldEqual, false)
		t.Logf("create routing ret code(%d), info(%s)", routingResp.Code.Value, routingResp.Info.Value)
	})
	Convey("路由Discover，别名查询路由，返回源服务的路由信息", t, func() {
		createCommonRoutingConfig(t, serviceResp, 1, 0) // in=1, out=0
		defer cleanCommonRoutingConfig(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		time.Sleep(updateCacheInterval)
		service := &api.Service{Name: resp.Alias.Alias, Namespace: resp.Alias.Namespace}
		disResp := server.GetRoutingConfigWithCache(defaultCtx, service)
		So(respSuccess(disResp), ShouldEqual, true)
		So(len(disResp.Routing.Inbounds), ShouldEqual, 1)
		So(len(disResp.Routing.Outbounds), ShouldEqual, 0)
	})
}

// 测试获取别名列表
func TestGetServiceAliases(t *testing.T) {
	_, serviceResp := createCommonService(t, 203)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	var aliases []*api.Response
	count := 5
	for i := 0; i < count; i++ {
		resp := createCommonAlias(serviceResp, "", api.AliasType_CL5SID)
		if !respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		defer cleanServiceName(resp.Alias.Alias.Value, serviceResp.Namespace.Value)
		aliases = append(aliases, resp)
	}

	Convey("可以查询到全量别名", t, func() {
		resp := server.GetServiceAliases(nil)
		So(respSuccess(resp), ShouldEqual, true)
		So(len(resp.Aliases), ShouldBeGreaterThanOrEqualTo, count)
		So(resp.Amount.Value, ShouldBeGreaterThanOrEqualTo, count)
	})
	Convey("offset,limit测试", t, func() {
		query := map[string]string{"offset": "0", "limit": "100"}
		resp := server.GetServiceAliases(query)
		So(respSuccess(resp), ShouldEqual, true)
		So(len(resp.Aliases), ShouldBeGreaterThanOrEqualTo, count)
		So(resp.Amount.Value, ShouldBeGreaterThanOrEqualTo, count)

		query["limit"] = "0"
		resp = server.GetServiceAliases(query)
		So(respSuccess(resp), ShouldEqual, true)
		So(len(resp.Aliases), ShouldEqual, 0)
		So(resp.Amount.Value, ShouldBeGreaterThanOrEqualTo, count)
	})
	Convey("不合法的过滤条件", t, func() {
		query := map[string]string{"xxx": "1", "limit": "100"}
		resp := server.GetServiceAliases(query)
		So(respSuccess(resp), ShouldEqual, false)
	})
	Convey("过滤条件可以生效", t, func() {
		query := map[string]string{
			"alias":     aliases[2].Alias.Alias.Value,
			"service":   serviceResp.Name.Value,
			"namespace": serviceResp.Namespace.Value,
		}
		resp := server.GetServiceAliases(query)
		So(respSuccess(resp), ShouldEqual, true)
		So(len(resp.Aliases), ShouldEqual, 1)
		So(resp.Amount.Value, ShouldEqual, 1)
	})
	Convey("找不到别名", t, func() {
		query := map[string]string{"alias": "x1.1.x2.x3"}
		resp := server.GetServiceAliases(query)
		So(respSuccess(resp), ShouldEqual, true)
		So(len(resp.Aliases), ShouldEqual, 0)
		So(resp.Amount.Value, ShouldEqual, 0)
	})
	Convey("支持owner过滤", t, func() {
		query := map[string]string{"owner": "service-owner-203"}
		resp := server.GetServiceAliases(query)
		So(respSuccess(resp), ShouldEqual, true)
		So(len(resp.Aliases), ShouldEqual, count)
		So(resp.Amount.Value, ShouldEqual, count)
	})
}

// test对serviceAlias字段进行校验
func TestCheckServiceAliasFieldLen(t *testing.T) {
	serviceAlias := &api.ServiceAlias{
		Service:   utils.NewStringValue("test-123"),
		Namespace: utils.NewStringValue("default"),
		Alias:     utils.NewStringValue("0"),
		Type:      api.AliasType_DEFAULT,
		Owners:    utils.NewStringValue("alias-owner"),
		Comment:   utils.NewStringValue("comment"),
	}
	t.Run("服务名超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldService := serviceAlias.Service
		serviceAlias.Service = utils.NewStringValue(str)
		resp := server.CreateServiceAliasNoAuth(defaultCtx, serviceAlias)
		serviceAlias.Service = oldService
		if resp.Code.Value != api.InvalidServiceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("命名空间超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldNamespace := serviceAlias.Namespace
		serviceAlias.Namespace = utils.NewStringValue(str)
		resp := server.CreateServiceAliasNoAuth(defaultCtx, serviceAlias)
		serviceAlias.Namespace = oldNamespace
		if resp.Code.Value != api.InvalidNamespaceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("别名超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldAlias := serviceAlias.Alias
		serviceAlias.Alias = utils.NewStringValue(str)
		resp := server.CreateServiceAliasNoAuth(defaultCtx, serviceAlias)
		serviceAlias.Alias = oldAlias
		if resp.Code.Value != api.InvalidServiceAlias {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("服务别名comment超长", func(t *testing.T) {
		str := genSpecialStr( 1025)
		oldComment := serviceAlias.Comment
		serviceAlias.Comment = utils.NewStringValue(str)
		resp := server.CreateServiceAliasNoAuth(defaultCtx, serviceAlias)
		serviceAlias.Comment = oldComment
		if resp.Code.Value != api.InvalidServiceAliasComment {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("服务owner超长", func(t *testing.T) {
		str := genSpecialStr( 1025)
		oldOwner := serviceAlias.Owners
		serviceAlias.Owners = utils.NewStringValue(str)
		resp := server.CreateServiceAliasNoAuth(defaultCtx, serviceAlias)
		serviceAlias.Owners = oldOwner
		if resp.Code.Value != api.InvalidServiceAliasOwners {
			t.Fatalf("%+v", resp)
		}
	})
}