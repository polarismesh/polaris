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
	"github.com/polarismesh/polaris-server/service"
	"testing"
)

/**
 * @brief 测试使用平台Token操作服务
 */
func TestServiceAuthByPlatform(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), platformToken)

	t.Run("使用平台Token修改服务，有权限", func(t *testing.T) {
		serviceResp := createService(t, 1)
		defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		req := getCommonService(serviceResp)
		req.Owners = utils.NewStringValue("test-owner")

		resp := server.UpdateService(ctx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("使用平台Token删除服务，有权限", func(t *testing.T) {
		serviceResp := createService(t, 2)
		defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		req := getCommonService(serviceResp)

		resp := server.DeleteService(ctx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})
}

/**
 * @brief 测试使用服务Token操作服务
 */
func TestServiceAuthByService(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), "test")

	t.Run("平台信息为空，使用服务Token修改服务，有权限", func(t *testing.T) {
		serviceResp := createService(t, 1)
		defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		serviceResp.Owners = utils.NewStringValue("test-owner")

		resp := server.UpdateService(defaultCtx, serviceResp)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token修改服务，有权限", func(t *testing.T) {
		serviceResp := createService(t, 3)
		defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		serviceResp.Owners = utils.NewStringValue("test-owner")

		resp := server.UpdateService(ctx, serviceResp)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token修改服务，有权限", func(t *testing.T) {
		serviceResp := createService(t, 4)
		defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		req := getCommonService(serviceResp)
		req.Owners = utils.NewStringValue("test-owner")

		resp := server.UpdateService(globalCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，修改服务，返回错误", func(t *testing.T) {
		serviceResp := createService(t, 5)
		defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		req := getCommonService(serviceResp)
		req.Owners = utils.NewStringValue("test-owner")

		resp := server.UpdateService(defaultCtx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，修改服务，返回错误", func(t *testing.T) {
		serviceResp := createService(t, 5)
		defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		req := getCommonService(serviceResp)
		req.Owners = utils.NewStringValue("test-owner")
		req.Token = utils.NewStringValue("test")

		resp := server.UpdateService(defaultCtx, req)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("平台信息为空，使用服务Token删除服务，有权限", func(t *testing.T) {
		serviceResp := createService(t, 2)
		defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		resp := server.DeleteService(defaultCtx, serviceResp)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token删除服务，有权限", func(t *testing.T) {
		serviceResp := createService(t, 4)
		defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		resp := server.DeleteService(ctx, serviceResp)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token删除服务，有权限", func(t *testing.T) {
		serviceResp := createService(t, 4)
		defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		req := getCommonService(serviceResp)
		resp := server.DeleteService(globalCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，删除服务，返回错误", func(t *testing.T) {
		serviceResp := createService(t, 6)
		defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		req := getCommonService(serviceResp)
		resp := server.DeleteService(ctx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，删除服务，返回错误", func(t *testing.T) {
		serviceResp := createService(t, 7)
		defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		serviceResp.Token = utils.NewStringValue("test")
		resp := server.DeleteService(ctx, serviceResp)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 创建服务
 */
func createService(t *testing.T, id int) *api.Service {
	req := &api.Service{
		Name:       utils.NewStringValue(fmt.Sprintf("test-service-%d", id)),
		Namespace:  utils.NewStringValue(service.DefaultNamespace),
		PlatformId: utils.NewStringValue(platformID),
		Owners:     utils.NewStringValue(fmt.Sprintf("service-owner-%d", id)),
	}

	cleanService(req.GetName().GetValue(), req.GetNamespace().GetValue())

	resp := server.CreateService(defaultCtx, req)
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}

	return resp.GetService()
}

/**
 * @brief 删除服务
 */
func cleanService(name string, namespace string) {
	str := `delete from service where name = ? and namespace = ?`
	if _, err := db.Exec(str, name, namespace); err != nil {
		panic(err)
	}

	str = `delete from owner_service_map where service=? and namespace=?`
	if _, err := db.Exec(str, name, namespace); err != nil {
		panic(err)
	}
}

/**
 * @brief 生成服务信息
 */
func getCommonService(req *api.Service) *api.Service {
	service := &api.Service{
		Name:      req.GetName(),
		Namespace: req.GetNamespace(),
	}
	return service
}
