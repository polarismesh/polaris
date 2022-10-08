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
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

const defaultAliasNs = "Production"

// 创建一个服务别名
func (d *DiscoverTestSuit) createCommonAlias(service *api.Service, alias string, aliasNamespace string, typ api.AliasType) *api.Response {
	req := &api.ServiceAlias{
		Service:        service.Name,
		Namespace:      service.Namespace,
		Alias:          utils.NewStringValue(alias),
		AliasNamespace: utils.NewStringValue(aliasNamespace),
		Type:           typ,
		Owners:         utils.NewStringValue("polaris"),
	}
	return d.server.CreateServiceAlias(d.defaultCtx, req)
}

// 创建别名，并检查
func (d *DiscoverTestSuit) createCommonAliasCheck(
	t *testing.T, service *api.Service, alias string, aliasNamespace string, typ api.AliasType) *api.Response {
	resp := d.createCommonAlias(service, alias, aliasNamespace, typ)
	if !respSuccess(resp) {
		t.Fatalf("error : %s", resp.GetInfo().GetValue())
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

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, serviceResp := discoverSuit.createCommonService(t, 123)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	Convey("正常创建非Sid的别名", t, func() {
		alias := fmt.Sprintf("alias.%d", time.Now().Unix())
		resp := discoverSuit.createCommonAlias(serviceResp, alias, serviceResp.GetNamespace().GetValue(), api.AliasType_DEFAULT)
		defer discoverSuit.cleanServiceName(alias, serviceResp.GetNamespace().GetValue())
		So(respSuccess(resp), ShouldEqual, true)
		So(resp.Alias.Alias.Value, ShouldEqual, alias)
	})

	Convey("正常创建Sid别名", t, func() {
		resp := discoverSuit.createCommonAlias(serviceResp, "", serviceResp.GetNamespace().GetValue(), api.AliasType_CL5SID)
		So(respSuccess(resp), ShouldEqual, true)
		defer discoverSuit.cleanServiceName(resp.Alias.Alias.Value, serviceResp.GetNamespace().GetValue())
		So(isSid(resp.Alias.Alias.Value), ShouldEqual, true)
		t.Logf("alias sid: %s", resp.Alias.Alias.Value)
	})

	Convey("使用ctx带上的token可以创建成功", t, func() {
		req := &api.ServiceAlias{
			Service:        serviceResp.Name,
			Namespace:      serviceResp.Namespace,
			AliasNamespace: serviceResp.Namespace,
			Type:           api.AliasType_CL5SID,
		}
		ctx := context.WithValue(discoverSuit.defaultCtx, utils.StringContext("polaris-token"),
			serviceResp.GetToken().GetValue())
		resp := discoverSuit.server.CreateServiceAlias(ctx, req)
		So(respSuccess(resp), ShouldEqual, true)
		discoverSuit.cleanServiceName(resp.Alias.Alias.Value, serviceResp.GetNamespace().GetValue())

		// 带上系统token，也可以成功
		ctx = context.WithValue(discoverSuit.defaultCtx, utils.StringContext("polaris-token"),
			"polaris@12345678")
		resp = discoverSuit.server.CreateServiceAlias(ctx, req)
		So(respSuccess(resp), ShouldEqual, true)
		discoverSuit.cleanServiceName(resp.Alias.Alias.Value, serviceResp.GetNamespace().GetValue())
	})
	Convey("不允许为别名创建别名", t, func() {
		resp := discoverSuit.namespaceSvr.CreateNamespace(discoverSuit.defaultCtx, &api.Namespace{
			Name: &wrapperspb.StringValue{Value: defaultAliasNs},
		})
		if !respSuccess(resp) {
			t.Fatalf("error : %s", resp.GetInfo().GetValue())
		}

		resp = discoverSuit.createCommonAliasCheck(t, serviceResp, "", defaultAliasNs, api.AliasType_CL5SID)
		defer discoverSuit.cleanServiceName(resp.Alias.Alias.Value, serviceResp.Namespace.Value)

		service := &api.Service{
			Name:      resp.Alias.Alias,
			Namespace: serviceResp.Namespace,
			Token:     serviceResp.Token,
		}
		repeatedResp := discoverSuit.createCommonAlias(service, "", defaultAliasNs, api.AliasType_CL5SID)
		if respSuccess(repeatedResp) {
			t.Fatalf("error: %+v", repeatedResp)
		}
		t.Logf("%+v", repeatedResp)
	})
}

// 重点测试创建sid别名的场景
// 注意：该测试函数出错的情况，会遗留一些测试数据无法清理 TODO
func TestCreateSid(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	Convey("创建不同命名空间的sid，可以返回符合规范的sid", t, func() {
		for namespace, layout := range Namespace2SidLayoutID {
			service := &api.Service{
				Name:      utils.NewStringValue("sid-test-xxx"),
				Namespace: utils.NewStringValue(namespace),
				Revision:  utils.NewStringValue("revision111"),
				Owners:    utils.NewStringValue("owners111"),
			}
			discoverSuit.cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
			serviceResp := discoverSuit.server.CreateServices(discoverSuit.defaultCtx, []*api.Service{service})
			t.Logf("resp : %s", serviceResp.GetInfo().GetValue())
			So(respSuccess(serviceResp), ShouldEqual, true)

			aliasResp := discoverSuit.createCommonAlias(serviceResp.Responses[0].Service, "", namespace, api.AliasType_CL5SID)
			So(respSuccess(aliasResp), ShouldEqual, true)
			modID, cmdID := parseStr2Sid(aliasResp.GetAlias().GetAlias().GetValue())
			So(modID, ShouldNotEqual, uint32(0))
			So(cmdID, ShouldNotEqual, uint32(0))
			So(modID>>6, ShouldBeGreaterThanOrEqualTo, 3000001) // module
			So(modID&63, ShouldEqual, layout)                   // 根据保留字段标识服务名
			So(aliasResp.GetAlias().GetNamespace().GetValue(), ShouldEqual, namespace)
			discoverSuit.cleanServiceName(aliasResp.GetAlias().GetAlias().GetValue(), namespace)
			discoverSuit.cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		}
	})
	Convey("非默认的5个命名空间，不允许创建sid别名", t, func() {
		namespace := &api.Namespace{
			Name:   utils.NewStringValue("other-namespace-xxx"),
			Owners: utils.NewStringValue("aaa"),
		}
		So(respSuccess(discoverSuit.namespaceSvr.CreateNamespace(discoverSuit.defaultCtx, namespace)), ShouldEqual, true)
		defer discoverSuit.cleanNamespace(namespace.Name.Value)

		service := &api.Service{
			Name:      utils.NewStringValue("sid-test-xxx"),
			Namespace: utils.NewStringValue(namespace.Name.Value),
			Revision:  utils.NewStringValue("revision111"),
			Owners:    utils.NewStringValue("owners111"),
		}
		serviceResp := discoverSuit.server.CreateServices(discoverSuit.defaultCtx, []*api.Service{service})
		So(respSuccess(serviceResp), ShouldEqual, true)
		defer discoverSuit.cleanServiceName(service.GetName().GetValue(), service.GetNamespace().GetValue())
		aliasResp := discoverSuit.createCommonAlias(serviceResp.Responses[0].Service, "", namespace.Name.Value, api.AliasType_CL5SID)
		So(respSuccess(aliasResp), ShouldEqual, false)
		t.Logf("%s", aliasResp.GetInfo().GetValue())
	})
}

// 并发测试
func TestConcurrencyCreateSid(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, serviceResp := discoverSuit.createCommonService(t, 234)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	Convey("并发创建sid别名，sid不会重复", t, func() {
		c := 20
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
				resp := discoverSuit.createCommonAlias(
					serviceResp, "", serviceResp.GetNamespace().GetValue(), api.AliasType_CL5SID)
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
			defer discoverSuit.cleanServiceName(resp.Alias.Alias.Value, serviceResp.GetNamespace().GetValue())
			So(isSid(resp.Alias.Alias.Value), ShouldEqual, true)
			repeated[resp.Alias.Alias.Value] = true
		}
		// 检查是否重复，必须是200个
		So(len(repeated), ShouldEqual, c)
	})
}

// 异常测试
func TestExceptCreateAlias(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, serviceResp := discoverSuit.createCommonService(t, 345)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	Convey("参数缺失，报错", t, func() {
		noService := &api.Service{}
		resp := discoverSuit.createCommonAlias(
			noService, "x1.x2.x3", serviceResp.GetNamespace().GetValue(), api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, false)

		noService.Name = utils.NewStringValue("123")
		resp = discoverSuit.createCommonAlias(
			noService, "x1.x2.x3", serviceResp.GetNamespace().GetValue(), api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, false)

		noService.Namespace = utils.NewStringValue("456")
		resp = discoverSuit.createCommonAlias(
			noService, "x1.x2.x3", serviceResp.GetNamespace().GetValue(), api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, false)

		noService.Token = utils.NewStringValue("567")
		resp = discoverSuit.createCommonAlias(noService, "", serviceResp.GetNamespace().GetValue(), api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, false)
		t.Logf("return code: %d", resp.Code.Value)
	})

	Convey("不存在的源服务，报错", t, func() {
		noService := &api.Service{
			Name:      utils.NewStringValue("my.service.2020.02.19"),
			Namespace: utils.NewStringValue("123123"),
			Token:     utils.NewStringValue("aaa"),
		}
		resp := discoverSuit.createCommonAlias(noService, "x1.x2.x3", noService.Namespace.GetValue(), api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, false)
		t.Logf("return code: %d", resp.Code.Value)
		So(resp.Code.Value, ShouldEqual, api.NotFoundService)
	})

	Convey("同名alias，报错", t, func() {
		resp := discoverSuit.createCommonAlias(
			serviceResp, "x1.x2.x3", serviceResp.GetNamespace().GetValue(), api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, true)
		defer discoverSuit.cleanServiceName(resp.Alias.Alias.Value, serviceResp.GetNamespace().GetValue())

		resp = discoverSuit.createCommonAlias(
			serviceResp, "x1.x2.x3", serviceResp.GetNamespace().GetValue(), api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, false)
		t.Logf("same alias return code: %d", resp.Code.Value)
	})

	Convey("鉴权失败，报错", t, func() {
		service := &api.Service{
			Name:      serviceResp.Name,
			Namespace: serviceResp.Namespace,
			Token:     utils.NewStringValue("123123123"),
		}

		oldCtx := discoverSuit.defaultCtx

		discoverSuit.defaultCtx = context.Background()

		defer func() {
			discoverSuit.defaultCtx = oldCtx
		}()

		resp := discoverSuit.createCommonAlias(service, "x1.x2.x3", service.Namespace.GetValue(), api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, false)
		t.Logf("error token, return code: %d", resp.Code.Value)
	})

	Convey("指向的服务不存在（新接口）", t, func() {
		_, serviceResp2 := discoverSuit.createCommonService(t, 2)
		discoverSuit.cleanServiceName(serviceResp2.GetName().GetValue(), serviceResp2.GetNamespace().GetValue())
		resp := discoverSuit.createCommonAlias(serviceResp2, "", serviceResp2.GetNamespace().GetValue(), api.AliasType_CL5SID)
		if respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		t.Logf("%+v", resp)
	})
}

// 别名修改的测试
func TestUpdateServiceAlias(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, serviceResp := discoverSuit.createCommonService(t, 3)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	Convey("修改别名负责人", t, func() {
		resp := discoverSuit.createCommonAlias(serviceResp, "", serviceResp.GetNamespace().GetValue(), api.AliasType_CL5SID)
		So(respSuccess(resp), ShouldEqual, true)
		defer discoverSuit.cleanServiceName(resp.GetAlias().GetAlias().GetValue(), serviceResp.GetNamespace().GetValue())

		// 修改别名负责人
		req := &api.ServiceAlias{
			Service:        resp.GetAlias().GetService(),
			Namespace:      resp.GetAlias().GetNamespace(),
			Alias:          resp.GetAlias().GetAlias(),
			AliasNamespace: resp.GetAlias().GetNamespace(),
			Owners:         utils.NewStringValue("alias-owner-new"),
			ServiceToken:   resp.GetAlias().GetServiceToken(),
		}

		repeatedResp := discoverSuit.server.UpdateServiceAlias(discoverSuit.defaultCtx, req)
		So(respSuccess(repeatedResp), ShouldEqual, true)

		query := map[string]string{
			"alias":     req.GetAlias().GetValue(),
			"namespace": req.GetNamespace().GetValue(),
		}
		aliasResponse := discoverSuit.server.GetServiceAliases(discoverSuit.defaultCtx, query)
		// 判断负责人是否一致
		So(aliasResponse.GetAliases()[0].GetOwners().GetValue(), ShouldEqual, "alias-owner-new")
		t.Logf("pass, owner is %v", aliasResponse.GetAliases()[0].GetOwners().GetValue())
	})

	Convey("修改指向服务", t, func() {
		resp := discoverSuit.createCommonAlias(serviceResp, "", serviceResp.GetNamespace().GetValue(), api.AliasType_CL5SID)
		So(respSuccess(resp), ShouldEqual, true)
		defer discoverSuit.cleanServiceName(resp.GetAlias().GetAlias().GetValue(), serviceResp.GetNamespace().GetValue())

		// 创建新的服务
		_, serviceResp2 := discoverSuit.createCommonService(t, 4)
		defer discoverSuit.cleanServiceName(serviceResp2.GetName().GetValue(), serviceResp2.GetNamespace().GetValue())

		// 修改别名指向
		req := &api.ServiceAlias{
			Service:        serviceResp2.GetName(),
			Namespace:      serviceResp2.GetNamespace(),
			Alias:          resp.GetAlias().GetAlias(),
			AliasNamespace: serviceResp2.GetNamespace(),
			Owners:         resp.GetAlias().GetOwners(),
			Comment:        resp.GetAlias().GetComment(),
			ServiceToken:   resp.GetAlias().GetServiceToken(),
		}

		repeatedResp := discoverSuit.server.UpdateServiceAlias(discoverSuit.defaultCtx, req)
		So(respSuccess(repeatedResp), ShouldEqual, true)

		query := map[string]string{
			"alias":     req.GetAlias().GetValue(),
			"namespace": req.GetNamespace().GetValue(),
		}
		aliasResponse := discoverSuit.server.GetServiceAliases(discoverSuit.defaultCtx, query)
		// 判断指向服务是否一致
		So(aliasResponse.GetAliases()[0].GetService().GetValue(), ShouldEqual, serviceResp2.GetName().GetValue())
		t.Logf("pass, service is %v", aliasResponse.GetAliases()[0].GetService().GetValue())
	})

	Convey("要指向的服务不存在", t, func() {
		resp := discoverSuit.createCommonAlias(serviceResp, "", serviceResp.GetNamespace().GetValue(), api.AliasType_CL5SID)
		So(respSuccess(resp), ShouldEqual, true)
		defer discoverSuit.cleanServiceName(resp.GetAlias().GetAlias().GetValue(), serviceResp.GetNamespace().GetValue())

		// 创建新的服务并删除
		_, serviceResp2 := discoverSuit.createCommonService(t, 4)
		discoverSuit.cleanServiceName(serviceResp2.GetName().GetValue(), serviceResp2.GetNamespace().GetValue())

		// 修改别名指向
		req := &api.ServiceAlias{
			Service:      serviceResp2.GetName(),
			Namespace:    serviceResp2.GetNamespace(),
			Alias:        resp.GetAlias().GetAlias(),
			Owners:       resp.GetAlias().GetOwners(),
			Comment:      resp.GetAlias().GetComment(),
			ServiceToken: resp.GetAlias().GetServiceToken(),
		}
		repeatedResp := discoverSuit.server.UpdateServiceAlias(discoverSuit.defaultCtx, req)
		if respSuccess(repeatedResp) {
			t.Fatalf("error: %+v", repeatedResp)
		}
		t.Logf("%+v", repeatedResp)
	})

	Convey("鉴权失败", t, func() {
		resp := discoverSuit.createCommonAlias(serviceResp, "", serviceResp.GetNamespace().GetValue(), api.AliasType_CL5SID)
		So(respSuccess(resp), ShouldEqual, true)
		defer discoverSuit.cleanServiceName(resp.GetAlias().GetAlias().GetValue(), serviceResp.GetNamespace().GetValue())
		// 修改service token
		req := resp.GetAlias()
		req.ServiceToken = utils.NewStringValue("")

		repeatedResp := discoverSuit.server.UpdateServiceAlias(context.Background(), req)

		if respSuccess(repeatedResp) {
			t.Fatalf("error: %+v", repeatedResp)
		}
		t.Logf("%+v", repeatedResp)
	})
}

// 别名删除
func TestDeleteServiceAlias(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, serviceResp := discoverSuit.createCommonService(t, 201)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	Convey("通过服务别名删除接口可以直接删除别名", t, func() {
		resp := discoverSuit.createCommonAlias(serviceResp, serviceResp.Name.GetValue()+"_alias", serviceResp.GetNamespace().GetValue(), api.AliasType_DEFAULT)
		So(respSuccess(resp), ShouldEqual, true)
		defer discoverSuit.cleanServiceName(resp.Alias.Alias.Value, resp.Alias.AliasNamespace.Value)
		discoverSuit.removeCommonServiceAliases(t, []*api.ServiceAlias{resp.Alias})

		query := map[string]string{"name": resp.Alias.Alias.Value}
		queryResp := discoverSuit.server.GetServices(discoverSuit.defaultCtx, query)
		So(respSuccess(queryResp), ShouldEqual, true)
		So(len(queryResp.Services), ShouldEqual, 0)
	})

	Convey("通过ctx带上token，可以删除别名成功", t, func() {
		resp := discoverSuit.createCommonAlias(serviceResp, "", serviceResp.GetNamespace().GetValue(), api.AliasType_CL5SID)
		So(respSuccess(resp), ShouldEqual, true)
		defer discoverSuit.cleanServiceName(resp.Alias.Alias.Value, serviceResp.Namespace.Value)

		ctx := context.WithValue(discoverSuit.defaultCtx, utils.StringContext("polaris-token"),
			"polaris@12345678")
		So(respSuccess(discoverSuit.server.DeleteServiceAliases(ctx, []*api.ServiceAlias{resp.Alias})), ShouldEqual, true)
	})

}

// 服务实例与服务路由关联测试
func TestServiceAliasRelated(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, serviceResp := discoverSuit.createCommonService(t, 202)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	resp := discoverSuit.createCommonAlias(serviceResp, "", serviceResp.GetNamespace().GetValue(), api.AliasType_CL5SID)
	if !respSuccess(resp) {
		t.Fatalf("errror")
	}
	defer discoverSuit.cleanServiceName(resp.Alias.Alias.Value, serviceResp.Namespace.Value)
	Convey("实例新建，不允许为别名新建实例", t, func() {
		instance := &api.Instance{
			Service:      resp.Alias.Alias,
			Namespace:    serviceResp.Namespace,
			ServiceToken: serviceResp.Token,
			Host:         utils.NewStringValue("1.12.123.132"),
			Port:         utils.NewUInt32Value(8080),
		}
		instanceResp := discoverSuit.server.CreateInstances(discoverSuit.defaultCtx, []*api.Instance{instance})
		So(respSuccess(instanceResp), ShouldEqual, false)
		t.Logf("alias create instance ret code(%d), msg(%s)",
			instanceResp.Code.Value, instanceResp.Info.Value)
	})
	Convey("实例Discover，别名查询实例，返回源服务的实例信息", t, func() {
		_, instanceResp := discoverSuit.createCommonInstance(t, serviceResp, 123)
		defer discoverSuit.cleanInstance(instanceResp.GetId().GetValue())

		time.Sleep(discoverSuit.updateCacheInterval)
		service := &api.Service{Name: resp.Alias.Alias, Namespace: resp.Alias.Namespace}
		disResp := discoverSuit.server.ServiceInstancesCache(discoverSuit.defaultCtx, service)
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
		routingResp := discoverSuit.server.CreateRoutingConfigs(discoverSuit.defaultCtx, []*api.Routing{routing})
		So(respSuccess(routingResp), ShouldEqual, false)
		t.Logf("create routing ret code(%d), info(%s)", routingResp.Code.Value, routingResp.Info.Value)
	})
	// Convey("路由Discover，别名查询路由，返回源服务的路由信息", t, func() {
	// 	discoverSuit.createCommonRoutingConfig(t, serviceResp, 1, 0) // in=1, out=0
	// 	defer discoverSuit.cleanCommonRoutingConfig(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	// 	time.Sleep(discoverSuit.updateCacheInterval)
	// 	service := &api.Service{Name: resp.Alias.Alias, Namespace: resp.Alias.Namespace}
	// 	disResp := discoverSuit.server.GetRoutingConfigWithCache(discoverSuit.defaultCtx, service)
	// 	So(respSuccess(disResp), ShouldEqual, true)
	// 	So(len(disResp.Routing.Inbounds), ShouldEqual, 1)
	// 	So(len(disResp.Routing.Outbounds), ShouldEqual, 0)
	// })
}

// 测试获取别名列表
func TestGetServiceAliases(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, serviceResp := discoverSuit.createCommonService(t, 203)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	var aliases []*api.Response
	count := 5
	for i := 0; i < count; i++ {
		resp := discoverSuit.createCommonAlias(serviceResp, "", serviceResp.GetNamespace().GetValue(), api.AliasType_CL5SID)
		if !respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		defer discoverSuit.cleanServiceName(resp.Alias.Alias.Value, serviceResp.Namespace.Value)
		aliases = append(aliases, resp)
	}

	Convey("可以查询到全量别名", t, func() {
		resp := discoverSuit.server.GetServiceAliases(discoverSuit.defaultCtx, nil)
		So(respSuccess(resp), ShouldEqual, true)
		So(len(resp.Aliases), ShouldBeGreaterThanOrEqualTo, count)
		So(resp.Amount.Value, ShouldBeGreaterThanOrEqualTo, count)
	})
	Convey("offset,limit测试", t, func() {
		query := map[string]string{"offset": "0", "limit": "100"}
		resp := discoverSuit.server.GetServiceAliases(discoverSuit.defaultCtx, query)
		So(respSuccess(resp), ShouldEqual, true)
		So(len(resp.Aliases), ShouldBeGreaterThanOrEqualTo, count)
		So(resp.Amount.Value, ShouldBeGreaterThanOrEqualTo, count)

		query["limit"] = "0"
		resp = discoverSuit.server.GetServiceAliases(discoverSuit.defaultCtx, query)
		So(respSuccess(resp), ShouldEqual, true)
		So(len(resp.Aliases), ShouldEqual, 0)
		So(resp.Amount.Value, ShouldBeGreaterThanOrEqualTo, count)
	})
	Convey("不合法的过滤条件", t, func() {
		query := map[string]string{"xxx": "1", "limit": "100"}
		resp := discoverSuit.server.GetServiceAliases(discoverSuit.defaultCtx, query)
		So(respSuccess(resp), ShouldEqual, false)
	})
	Convey("过滤条件可以生效", t, func() {
		query := map[string]string{
			"alias":     aliases[2].Alias.Alias.Value,
			"service":   serviceResp.Name.Value,
			"namespace": serviceResp.Namespace.Value,
		}
		resp := discoverSuit.server.GetServiceAliases(discoverSuit.defaultCtx, query)
		So(respSuccess(resp), ShouldEqual, true)
		So(len(resp.Aliases), ShouldEqual, 1)
		So(resp.Amount.Value, ShouldEqual, 1)
	})
	Convey("找不到别名", t, func() {
		query := map[string]string{"alias": "x1.1.x2.x3"}
		resp := discoverSuit.server.GetServiceAliases(discoverSuit.defaultCtx, query)
		So(respSuccess(resp), ShouldEqual, true)
		So(len(resp.Aliases), ShouldEqual, 0)
		So(resp.Amount.Value, ShouldEqual, 0)
	})
	// Convey("支持owner过滤", t, func() {
	// 	query := map[string]string{"owner": "service-owner-203"}
	// 	resp := discoverSuit.server.GetServiceAliases(discoverSuit.defaultCtx, query)
	// 	So(respSuccess(resp), ShouldEqual, true)
	// 	So(len(resp.Aliases), ShouldEqual, count)
	// 	So(resp.Amount.Value, ShouldEqual, count)
	// })
}

// test对serviceAlias字段进行校验
func TestCheckServiceAliasFieldLen(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	serviceAlias := &api.ServiceAlias{
		Service:        utils.NewStringValue("test-123"),
		Namespace:      utils.NewStringValue("Production"),
		Alias:          utils.NewStringValue("0"),
		AliasNamespace: utils.NewStringValue("Production"),
		Type:           api.AliasType_DEFAULT,
		Owners:         utils.NewStringValue("alias-owner"),
		Comment:        utils.NewStringValue("comment"),
	}
	t.Run("服务名超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldService := serviceAlias.Service
		serviceAlias.Service = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateServiceAlias(discoverSuit.defaultCtx, serviceAlias)
		serviceAlias.Service = oldService
		if resp.Code.Value != api.InvalidServiceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("命名空间超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldNamespace := serviceAlias.Namespace
		serviceAlias.Namespace = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateServiceAlias(discoverSuit.defaultCtx, serviceAlias)
		serviceAlias.Namespace = oldNamespace
		if resp.Code.Value != api.InvalidNamespaceName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("别名超长", func(t *testing.T) {
		str := genSpecialStr(129)
		oldAlias := serviceAlias.Alias
		serviceAlias.Alias = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateServiceAlias(discoverSuit.defaultCtx, serviceAlias)
		serviceAlias.Alias = oldAlias
		if resp.Code.Value != api.InvalidServiceAlias {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("服务别名comment超长", func(t *testing.T) {
		str := genSpecialStr(1025)
		oldComment := serviceAlias.Comment
		serviceAlias.Comment = utils.NewStringValue(str)
		resp := discoverSuit.server.CreateServiceAlias(discoverSuit.defaultCtx, serviceAlias)
		serviceAlias.Comment = oldComment
		if resp.Code.Value != api.InvalidServiceAliasComment {
			t.Fatalf("%+v", resp)
		}
	})
	// t.Run("服务owner超长", func(t *testing.T) {
	// 	str := genSpecialStr(1025)
	// 	oldOwner := serviceAlias.Owners
	// 	serviceAlias.Owners = utils.NewStringValue(str)
	// 	resp := discoverSuit.server.CreateServiceAlias(discoverSuit.defaultCtx, serviceAlias)
	// 	serviceAlias.Owners = oldOwner
	// 	if resp.Code.Value != api.InvalidServiceAliasOwners {
	// 		t.Fatalf("%+v", resp)
	// 	}
	// })
}

// test测试别名的命名空间与服务名不一样
func TestServiceAliasDifferentNamespace(t *testing.T) {

	discoverSuit := &DiscoverTestSuit{}
	if err := discoverSuit.initialize(); err != nil {
		t.Fatal(err)
	}
	defer discoverSuit.Destroy()

	_, serviceResp := discoverSuit.createCommonService(t, 203)
	defer discoverSuit.cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())
	Convey("正常创建不一样命名空间的非Sid的别名", t, func() {
		alias := fmt.Sprintf("alias.%d", time.Now().Unix())
		resp := discoverSuit.createCommonAlias(serviceResp, alias, defaultAliasNs, api.AliasType_DEFAULT)
		defer discoverSuit.cleanServiceName(alias, defaultAliasNs)
		So(respSuccess(resp), ShouldEqual, true)
		So(resp.Alias.Alias.Value, ShouldEqual, alias)
	})
}
