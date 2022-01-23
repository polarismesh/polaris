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

package testauthplugin

import (
	"context"
	"fmt"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
	"testing"
)

/**
 * @brief 测试使用平台Token操作路由规则
 */
func TestRoutingConfigAuthByPlatform(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), platformToken)

	// 创建服务
	serviceResp := createService(t, 3)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("使用平台Token创建路由规则，有权限", func(t *testing.T) {
		resp := createRoutingConfig(t, serviceResp, 1, ctx)
		defer cleanRoutingConfig(resp.GetService().GetValue(), serviceResp.GetNamespace().GetValue())
		t.Log("pass")
	})

	t.Run("使用平台Token修改路由规则，有权限", func(t *testing.T) {
		routingConfig := createRoutingConfig(t, serviceResp, 1, ctx)
		defer cleanRoutingConfig(routingConfig.GetService().GetValue(), routingConfig.GetNamespace().GetValue())

		routingConfig.Inbounds = []*api.Route{}
		resp := server.UpdateRoutingConfig(ctx, routingConfig)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("使用平台Token删除路由规则，有权限", func(t *testing.T) {
		routingConfig := createRoutingConfig(t, serviceResp, 1, ctx)
		defer cleanRoutingConfig(routingConfig.GetService().GetValue(), routingConfig.GetNamespace().GetValue())

		resp := server.DeleteRoutingConfig(ctx, routingConfig)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})
}

/**
 * @brief 测试使用服务Token创建路由规则
 */
func TestCreateRoutingConfigAuthByService(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), "test")

	correctCtx := context.Background()
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-id"), platformID)
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-token"), platformToken)

	// 创建服务
	serviceResp := createService(t, 5)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("平台信息为空，使用服务Token创建路由规则，有权限", func(t *testing.T) {
		req := &api.Routing{
			Service:      serviceResp.GetName(),
			Namespace:    serviceResp.GetNamespace(),
			ServiceToken: serviceResp.GetToken(),
		}

		resp := server.CreateRoutingConfig(defaultCtx, req)
		defer cleanRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())

		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token创建路由规则，有权限", func(t *testing.T) {
		req := &api.Routing{
			Service:      serviceResp.GetName(),
			Namespace:    serviceResp.GetNamespace(),
			ServiceToken: serviceResp.GetToken(),
		}

		resp := server.CreateRoutingConfig(ctx, req)
		defer cleanRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())

		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token创建路由规则，有权限", func(t *testing.T) {
		req := &api.Routing{
			Service:   serviceResp.GetName(),
			Namespace: serviceResp.GetNamespace(),
		}
		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		resp := server.CreateRoutingConfig(globalCtx, req)
		defer cleanRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())

		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，创建路由规则，返回错误", func(t *testing.T) {
		req := &api.Routing{
			Service:   serviceResp.GetName(),
			Namespace: serviceResp.GetNamespace(),
		}

		resp := server.CreateRoutingConfig(defaultCtx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Log("pass")
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，创建路由规则，返回错误", func(t *testing.T) {
		req := &api.Routing{
			Service:      serviceResp.GetName(),
			Namespace:    serviceResp.GetNamespace(),
			ServiceToken: utils.NewStringValue("test"),
		}

		resp := server.CreateRoutingConfig(defaultCtx, req)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Log("pass")
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 测试使用服务Token修改路由规则
 */
func TestUpdateRoutingConfigAuthByService(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), "test")

	correctCtx := context.Background()
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-id"), platformID)
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-token"), platformToken)

	// 创建服务
	serviceResp := createService(t, 5)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("平台信息为空，使用服务Token修改实例，有权限", func(t *testing.T) {
		req := createRoutingConfig(t, serviceResp, 22, correctCtx)
		defer cleanRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())

		req.ServiceToken = serviceResp.GetToken()
		resp := server.UpdateRoutingConfig(defaultCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token修改路由规则，有权限", func(t *testing.T) {
		req := createRoutingConfig(t, serviceResp, 33, correctCtx)
		defer cleanRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())

		req.ServiceToken = serviceResp.GetToken()
		resp := server.UpdateRoutingConfig(ctx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token修改路由规则，有权限", func(t *testing.T) {
		req := createRoutingConfig(t, serviceResp, 33, correctCtx)
		defer cleanRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())

		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		resp := server.UpdateRoutingConfig(globalCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，修改路由规则，返回错误", func(t *testing.T) {
		req := createRoutingConfig(t, serviceResp, 44, correctCtx)
		defer cleanRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())

		resp := server.UpdateRoutingConfig(defaultCtx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，修改路由规则，返回错误", func(t *testing.T) {
		req := createRoutingConfig(t, serviceResp, 55, correctCtx)
		defer cleanRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())

		req.ServiceToken = utils.NewStringValue("test")
		resp := server.UpdateRoutingConfig(ctx, req)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 使用服务Token删除路由规则
 */
func TestDeleteRoutingConfigAuthByService(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), "test")

	correctCtx := context.Background()
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-id"), platformID)
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-token"), platformToken)

	// 创建服务
	serviceResp := createService(t, 5)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("平台信息为空，使用服务Token删除路由规则，有权限", func(t *testing.T) {
		req := createRoutingConfig(t, serviceResp, 66, correctCtx)
		defer cleanRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())

		req.ServiceToken = serviceResp.GetToken()
		resp := server.DeleteRoutingConfig(defaultCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token删除路由规则，有权限", func(t *testing.T) {
		req := createRoutingConfig(t, serviceResp, 77, correctCtx)
		defer cleanRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())

		req.ServiceToken = serviceResp.GetToken()
		resp := server.DeleteRoutingConfig(ctx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token删除路由规则，有权限", func(t *testing.T) {
		req := createRoutingConfig(t, serviceResp, 77, correctCtx)
		defer cleanRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())

		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		resp := server.DeleteRoutingConfig(globalCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，删除路由规则，返回错误", func(t *testing.T) {
		req := createRoutingConfig(t, serviceResp, 88, correctCtx)
		defer cleanRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())

		resp := server.DeleteRoutingConfig(defaultCtx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，删除路由规则，返回错误", func(t *testing.T) {
		req := createRoutingConfig(t, serviceResp, 99, correctCtx)
		defer cleanRoutingConfig(req.GetService().GetValue(), req.GetNamespace().GetValue())

		req.ServiceToken = utils.NewStringValue("test")
		resp := server.DeleteRoutingConfig(ctx, req)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 创建路由规则
 */
func createRoutingConfig(t *testing.T, service *api.Service, id int, ctx context.Context) *api.Routing {
	source := &api.Source{
		Service:   utils.NewStringValue(fmt.Sprintf("in-source-service-%d", id)),
		Namespace: utils.NewStringValue(fmt.Sprintf("in-source-service-%d", id)),
	}
	req := &api.Routing{
		Service:   service.GetName(),
		Namespace: service.GetNamespace(),
		Inbounds: []*api.Route{
			{
				Sources: []*api.Source{source},
			},
		},
		Outbounds: []*api.Route{
			{
				Sources: []*api.Source{source},
			},
		},
	}

	resp := server.CreateRoutingConfig(ctx, req)
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}

	return resp.GetRouting()
}

/**
 * @brief 从数据库中删除路由规则
 */
func cleanRoutingConfig(service string, namespace string) {
	str := "delete from routing_config where id in (select id from service where name = ? and namespace = ?)"
	if _, err := db.Exec(str, service, namespace); err != nil {
		panic(err)
	}
}
