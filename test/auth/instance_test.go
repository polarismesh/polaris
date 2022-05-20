//go:build integrationauth
// +build integrationauth

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
	"sync"
	"testing"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
)

/**
 * @brief 测试使用平台Token操作实例
 */
func TestInstanceAuthByPlatform(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), platformToken)

	// 创建服务
	serviceResp := createService(t, 2)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("使用平台Token创建服务实例，有权限", func(t *testing.T) {
		resp := createInstance(t, serviceResp, 1, ctx)
		defer cleanInstance(resp.GetId().GetValue())
		t.Log("pass")
	})

	t.Run("使用平台Token并发创建服务实例，有权限", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 100; i <= 500; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				resp := createInstance(t, serviceResp, index, ctx)
				cleanInstance(resp.GetId().GetValue())
			}(i)
		}
		wg.Wait()
	})

	t.Run("使用平台Token修改服务实例，有权限", func(t *testing.T) {
		instance := createInstance(t, serviceResp, 2, ctx)
		defer cleanInstance(instance.GetId().GetValue())

		instance.Isolate = utils.NewBoolValue(false)
		resp := server.UpdateInstance(ctx, instance)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("使用平台Token删除服务实例，有权限", func(t *testing.T) {
		instance := createInstance(t, serviceResp, 3, ctx)
		defer cleanInstance(instance.GetId().GetValue())

		resp := server.DeleteInstance(ctx, instance)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("使用平台Token，根据host修改服务实例，有权限", func(t *testing.T) {
		instance := createInstance(t, serviceResp, 4, ctx)
		defer cleanInstance(instance.GetId().GetValue())

		instance.Isolate = utils.NewBoolValue(true)
		resp := server.UpdateInstanceIsolate(ctx, instance)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("使用平台Token，根据host删除服务实例，有权限", func(t *testing.T) {
		instance := createInstance(t, serviceResp, 5, ctx)
		defer cleanInstance(instance.GetId().GetValue())

		resp := server.DeleteInstanceByHost(ctx, instance)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

}

/**
 * @brief 测试使用服务Token创建实例
 */
func TestCreateInstanceAuthByService(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), "test")

	correctCtx := context.Background()
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-id"), platformID)
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-token"), platformToken)

	// 创建服务
	serviceResp := createService(t, 3)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("平台信息为空，使用服务Token创建实例，有权限", func(t *testing.T) {
		req := &api.Instance{
			Service:      serviceResp.GetName(),
			Namespace:    serviceResp.GetNamespace(),
			Host:         utils.NewStringValue("test-host"),
			Port:         utils.NewUInt32Value(11),
			ServiceToken: serviceResp.GetToken(),
		}

		resp := server.CreateInstance(defaultCtx, req)
		defer cleanInstance(resp.GetInstance().GetId().GetValue())

		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token创建实例，有权限", func(t *testing.T) {
		req := &api.Instance{
			Service:      serviceResp.GetName(),
			Namespace:    serviceResp.GetNamespace(),
			Host:         utils.NewStringValue("test-host"),
			Port:         utils.NewUInt32Value(22),
			ServiceToken: serviceResp.GetToken(),
		}

		resp := server.CreateInstance(ctx, req)
		defer cleanInstance(resp.GetInstance().GetId().GetValue())

		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token创建实例，有权限", func(t *testing.T) {
		req := &api.Instance{
			Service:   serviceResp.GetName(),
			Namespace: serviceResp.GetNamespace(),
			Host:      utils.NewStringValue("test-host"),
			Port:      utils.NewUInt32Value(33),
		}
		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		resp := server.CreateInstance(globalCtx, req)
		defer cleanInstance(resp.GetInstance().GetId().GetValue())

		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，创建实例，返回错误", func(t *testing.T) {
		req := &api.Instance{
			Service:   serviceResp.GetName(),
			Namespace: serviceResp.GetNamespace(),
			Host:      utils.NewStringValue("test-host"),
			Port:      utils.NewUInt32Value(44),
		}

		resp := server.CreateInstance(defaultCtx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Log("pass")
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，创建实例，返回错误", func(t *testing.T) {
		req := &api.Instance{
			Service:      serviceResp.GetName(),
			Namespace:    serviceResp.GetNamespace(),
			Host:         utils.NewStringValue("test-host"),
			Port:         utils.NewUInt32Value(55),
			ServiceToken: utils.NewStringValue("test"),
		}

		resp := server.CreateInstance(defaultCtx, req)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Log("pass")
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 测试使用服务Token修改实例
 */
func TestUpdateInstanceAuthByService(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), "test")

	correctCtx := context.Background()
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-id"), platformID)
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-token"), platformToken)

	// 创建服务
	serviceResp := createService(t, 3)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("平台信息为空，使用服务Token修改实例，有权限", func(t *testing.T) {
		req := createInstance(t, serviceResp, 66, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		req.Isolate = utils.NewBoolValue(true)
		req.ServiceToken = serviceResp.GetToken()
		resp := server.UpdateInstance(defaultCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token修改实例，有权限", func(t *testing.T) {
		req := createInstance(t, serviceResp, 77, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		req.Isolate = utils.NewBoolValue(true)
		req.ServiceToken = serviceResp.GetToken()
		resp := server.UpdateInstance(ctx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token修改实例，有权限", func(t *testing.T) {
		req := createInstance(t, serviceResp, 88, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		req.Isolate = utils.NewBoolValue(true)
		resp := server.UpdateInstance(globalCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，修改实例，返回错误", func(t *testing.T) {
		req := createInstance(t, serviceResp, 99, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		resp := server.UpdateInstance(defaultCtx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，修改实例，返回错误", func(t *testing.T) {
		req := createInstance(t, serviceResp, 10, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		req.ServiceToken = utils.NewStringValue("test")
		resp := server.UpdateInstance(ctx, req)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 测试使用服务Token根据host修改实例
 */
func TestUpdateIsolateAuthByService(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), "test")

	correctCtx := context.Background()
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-id"), platformID)
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-token"), platformToken)

	// 创建服务
	serviceResp := createService(t, 3)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("平台信息为空，使用服务Token修改实例，有权限", func(t *testing.T) {
		req := createInstance(t, serviceResp, 1111, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		req.Isolate = utils.NewBoolValue(true)
		req.ServiceToken = serviceResp.GetToken()
		resp := server.UpdateInstanceIsolate(defaultCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token修改实例，有权限", func(t *testing.T) {
		req := createInstance(t, serviceResp, 2222, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		req.Isolate = utils.NewBoolValue(true)
		req.ServiceToken = serviceResp.GetToken()
		resp := server.UpdateInstanceIsolate(ctx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token修改实例，有权限", func(t *testing.T) {
		req := createInstance(t, serviceResp, 3333, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		req.Isolate = utils.NewBoolValue(true)
		resp := server.UpdateInstanceIsolate(globalCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，修改实例，返回错误", func(t *testing.T) {
		req := createInstance(t, serviceResp, 4444, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		req.Isolate = utils.NewBoolValue(true)
		resp := server.UpdateInstanceIsolate(defaultCtx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，修改实例，返回错误", func(t *testing.T) {
		req := createInstance(t, serviceResp, 5555, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		req.Isolate = utils.NewBoolValue(true)
		req.ServiceToken = utils.NewStringValue("test")
		resp := server.UpdateInstanceIsolate(ctx, req)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 测试使用服务Token删除实例
 */
func TestDeleteInstanceAuthByService(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), "test")

	correctCtx := context.Background()
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-id"), platformID)
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-token"), platformToken)

	// 创建服务
	serviceResp := createService(t, 3)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("平台信息为空，使用服务Token删除实例，有权限", func(t *testing.T) {
		req := createInstance(t, serviceResp, 13, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		req.ServiceToken = serviceResp.GetToken()
		resp := server.DeleteInstance(defaultCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token删除实例，有权限", func(t *testing.T) {
		req := createInstance(t, serviceResp, 14, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		req.ServiceToken = serviceResp.GetToken()
		resp := server.DeleteInstance(ctx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token删除实例，有权限", func(t *testing.T) {
		req := createInstance(t, serviceResp, 15, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		resp := server.DeleteInstance(globalCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，删除实例，返回错误", func(t *testing.T) {
		req := createInstance(t, serviceResp, 17, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		resp := server.DeleteInstance(defaultCtx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，删除实例，返回错误", func(t *testing.T) {
		req := createInstance(t, serviceResp, 19, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		req.ServiceToken = utils.NewStringValue("test")
		resp := server.DeleteInstance(ctx, req)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 测试使用服务Token根据host删除实例
 */
func TestDeleteHostAuthByService(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), "test")

	correctCtx := context.Background()
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-id"), platformID)
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-token"), platformToken)

	// 创建服务
	serviceResp := createService(t, 3)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("平台信息为空，使用服务Token删除实例，有权限", func(t *testing.T) {
		req := createInstance(t, serviceResp, 101, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		req.ServiceToken = serviceResp.GetToken()
		resp := server.DeleteInstanceByHost(defaultCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token删除实例，有权限", func(t *testing.T) {
		req := createInstance(t, serviceResp, 102, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		req.ServiceToken = serviceResp.GetToken()
		resp := server.DeleteInstanceByHost(ctx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token删除实例，有权限", func(t *testing.T) {
		req := createInstance(t, serviceResp, 103, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		resp := server.DeleteInstanceByHost(globalCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，删除实例，返回错误", func(t *testing.T) {
		req := createInstance(t, serviceResp, 104, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		resp := server.DeleteInstanceByHost(defaultCtx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，删除实例，返回错误", func(t *testing.T) {
		req := createInstance(t, serviceResp, 105, correctCtx)
		defer cleanInstance(req.GetId().GetValue())

		req.ServiceToken = utils.NewStringValue("test")
		resp := server.DeleteInstanceByHost(ctx, req)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 创建服务实例
 */
func createInstance(t *testing.T, service *api.Service, id int, ctx context.Context) *api.Instance {
	req := &api.Instance{
		Service:   service.GetName(),
		Namespace: service.GetNamespace(),
		Host:      utils.NewStringValue(fmt.Sprintf("%d.%d.%d.%d", id, id, id, id)),
		Port:      utils.NewUInt32Value(uint32(id)),
	}

	resp := server.CreateInstance(ctx, req)
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}

	return resp.GetInstance()
}

/**
 * @brief 从数据库中删除服务实例
 */
func cleanInstance(id string) {
	log.Infof("clean instance: %s", id)
	str := `delete from instance where id = ?`
	if _, err := db.Exec(str, id); err != nil {
		panic(err)
	}
}
