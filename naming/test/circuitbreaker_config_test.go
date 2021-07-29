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
	"fmt"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/naming"
	"sync"
	"testing"
	"time"
)

/**
 * @brief 测试创建熔断规则
 */
func TestCreateCircuitBreaker(t *testing.T) {
	t.Run("正常创建熔断规则，返回成功", func(t *testing.T) {
		circuitBreakerReq, circuitBreakerResp := createCommonCircuitBreaker(t, 0)
		defer cleanCircuitBreaker(circuitBreakerResp.GetId().GetValue(), circuitBreakerResp.GetVersion().GetValue())
		checkCircuitBreaker(t, circuitBreakerReq, circuitBreakerReq, circuitBreakerResp)
	})

	t.Run("重复创建熔断规则，返回错误", func(t *testing.T) {
		_, circuitBreakerResp := createCommonCircuitBreaker(t, 0)
		defer cleanCircuitBreaker(circuitBreakerResp.GetId().GetValue(), circuitBreakerResp.GetVersion().GetValue())

		if resp := server.CreateCircuitBreaker(defaultCtx, circuitBreakerResp); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("创建熔断规则，删除，再创建，返回成功", func(t *testing.T) {
		_, circuitBreakerResp := createCommonCircuitBreaker(t, 0)
		defer cleanCircuitBreaker(circuitBreakerResp.GetId().GetValue(), circuitBreakerResp.GetVersion().GetValue())
		deleteCircuitBreaker(t, circuitBreakerResp)

		newCircuitBreakerReq, newCircuitBreakerResp := createCommonCircuitBreaker(t, 0)
		checkCircuitBreaker(t, newCircuitBreakerReq, newCircuitBreakerReq, newCircuitBreakerResp)
		cleanCircuitBreaker(newCircuitBreakerResp.GetId().GetValue(), newCircuitBreakerResp.GetVersion().GetValue())
	})

	t.Run("创建熔断规则时，没有传递负责人，返回错误", func(t *testing.T) {
		circuitBreaker := &api.CircuitBreaker{}
		if resp := server.CreateCircuitBreaker(defaultCtx, circuitBreaker); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("创建熔断规则时，没有传递规则名，返回错误", func(t *testing.T) {
		circuitBreaker := &api.CircuitBreaker{
			Namespace: utils.NewStringValue(naming.DefaultNamespace),
			Owners:    utils.NewStringValue("test"),
		}
		if resp := server.CreateCircuitBreaker(defaultCtx, circuitBreaker); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("创建熔断规则时，没有传递命名空间，返回错误", func(t *testing.T) {
		circuitBreaker := &api.CircuitBreaker{
			Name:   utils.NewStringValue("name-test-1"),
			Owners: utils.NewStringValue("test"),
		}
		if resp := server.CreateCircuitBreaker(defaultCtx, circuitBreaker); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("并发创建熔断规则，返回成功", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 500; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				_, circuitBreakerResp := createCommonCircuitBreaker(t, index)
				cleanCircuitBreaker(circuitBreakerResp.GetId().GetValue(), circuitBreakerResp.GetVersion().GetValue())
			}(i)
		}
		wg.Wait()
	})
}

/**
 * @brief 测试创建熔断规则版本
 */
func TestCreateCircuitBreakerVersion(t *testing.T) {
	_, cbResp := createCommonCircuitBreaker(t, 0)
	defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

	t.Run("正常创建熔断规则版本", func(t *testing.T) {
		cbVersionReq, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 0)
		defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
		checkCircuitBreaker(t, cbVersionReq, cbVersionReq, cbVersionResp)
	})

	t.Run("传递id，正常创建熔断规则版本", func(t *testing.T) {
		cbVersionReq := &api.CircuitBreaker{
			Id:      cbResp.GetId(),
			Version: utils.NewStringValue("test"),
			Token:   cbResp.GetToken(),
		}

		resp := server.CreateCircuitBreakerVersion(defaultCtx, cbVersionReq)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		cbVersionResp := resp.GetCircuitBreaker()

		defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		checkCircuitBreaker(t, cbVersionReq, cbVersionReq, cbVersionResp)
	})

	t.Run("传递name和namespace，正常创建熔断规则版本", func(t *testing.T) {
		cbVersionReq := &api.CircuitBreaker{
			Version:   utils.NewStringValue("test"),
			Name:      cbResp.GetName(),
			Namespace: cbResp.GetNamespace(),
			Token:     cbResp.GetToken(),
		}

		resp := server.CreateCircuitBreakerVersion(defaultCtx, cbVersionReq)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		cbVersionResp := resp.GetCircuitBreaker()

		defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		checkCircuitBreaker(t, cbVersionReq, cbVersionReq, cbVersionResp)
	})

	t.Run("创建熔断规则版本，删除，再创建，返回成功", func(t *testing.T) {
		cbVersionReq, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 0)
		defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		deleteCircuitBreaker(t, cbVersionResp)
		cbVersionReq, cbVersionResp = createCommonCircuitBreakerVersion(t, cbResp, 0)
		checkCircuitBreaker(t, cbVersionReq, cbVersionReq, cbVersionResp)
	})

	t.Run("为不存在的熔断规则创建版本，返回错误", func(t *testing.T) {
		_, cbResp := createCommonCircuitBreaker(t, 1)
		cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		version := &api.CircuitBreaker{
			Id:      cbResp.GetId(),
			Version: utils.NewStringValue("test"),
			Token:   cbResp.GetToken(),
			Owners:  cbResp.GetOwners(),
		}

		if resp := server.CreateCircuitBreakerVersion(defaultCtx, version); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("创建master版本的熔断规则，返回错误", func(t *testing.T) {
		if resp := server.CreateCircuitBreakerVersion(defaultCtx, cbResp); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("创建熔断规则版本时，没有传递version，返回错误", func(t *testing.T) {
		version := &api.CircuitBreaker{
			Id:     cbResp.GetId(),
			Token:  cbResp.GetToken(),
			Owners: cbResp.GetOwners(),
		}
		if resp := server.CreateCircuitBreakerVersion(defaultCtx, version); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("创建熔断规则版本时，没有传递token，返回错误", func(t *testing.T) {
		version := &api.CircuitBreaker{
			Id:      cbResp.GetId(),
			Version: cbResp.GetVersion(),
			Owners:  cbResp.GetOwners(),
		}
		if resp := server.CreateCircuitBreakerVersion(defaultCtx, version); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("创建熔断规则版本时，没有传递name，返回错误", func(t *testing.T) {
		version := &api.CircuitBreaker{
			Version:   cbResp.GetVersion(),
			Token:     cbResp.GetToken(),
			Namespace: cbResp.GetNamespace(),
		}
		if resp := server.CreateCircuitBreakerVersion(defaultCtx, version); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("创建熔断规则版本时，没有传递namespace，返回错误", func(t *testing.T) {
		version := &api.CircuitBreaker{
			Version: cbResp.GetVersion(),
			Token:   cbResp.GetToken(),
			Name:    cbResp.GetName(),
		}
		if resp := server.CreateCircuitBreakerVersion(defaultCtx, version); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("并发创建同一个规则的多个版本，返回成功", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i <= 500; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				cbVersionReq, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, index)
				checkCircuitBreaker(t, cbVersionReq, cbVersionReq, cbVersionResp)
				defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
			}(i)
		}
		wg.Wait()
		t.Log("pass")
	})
}

/**
 * @brief 删除熔断规则
 */
func TestDeleteCircuitBreaker(t *testing.T) {
	getCircuitBreakerVersions := func(t *testing.T, id string, expectNum uint32) {
		filters := map[string]string{
			"id": id,
		}
		resp := server.GetCircuitBreakerVersions(filters)
		if !respSuccess(resp) {
			t.Fatal("error")
		}
		if resp.GetAmount().GetValue() != expectNum {
			t.Fatalf("error, actual num is %d, expect num is %d", resp.GetAmount().GetValue(), expectNum)
		} else {
			t.Log("pass")
		}
	}

	t.Run("根据name和namespace删除master版本的熔断规则", func(t *testing.T) {
		// 创建熔断规则
		_, cbResp := createCommonCircuitBreaker(t, 0)
		defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		// 创建熔断规则版本
		for i := 1; i <= 10; i++ {
			_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, i)
			defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
		}

		rule := &api.CircuitBreaker{
			Version:   cbResp.GetVersion(),
			Name:      cbResp.GetName(),
			Namespace: cbResp.GetNamespace(),
			Token:     cbResp.GetToken(),
		}

		deleteCircuitBreaker(t, rule)
		getCircuitBreakerVersions(t, cbResp.GetId().GetValue(), 0)
	})

	t.Run("删除master版本的熔断规则", func(t *testing.T) {
		// 创建熔断规则
		_, cbResp := createCommonCircuitBreaker(t, 0)
		defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		// 创建熔断规则版本
		for i := 1; i <= 10; i++ {
			_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, i)
			defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
		}

		deleteCircuitBreaker(t, cbResp)
		getCircuitBreakerVersions(t, cbResp.GetId().GetValue(), 0)
	})

	t.Run("删除非master版本的熔断规则", func(t *testing.T) {
		// 创建熔断规则
		_, cbResp := createCommonCircuitBreaker(t, 0)
		defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		// 创建熔断规则版本
		_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 0)
		defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// 创建熔断规则版本
		for i := 1; i <= 10; i++ {
			_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, i)
			defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
		}

		// 删除特定版本的熔断规则
		deleteCircuitBreaker(t, cbVersionResp)

		getCircuitBreakerVersions(t, cbResp.GetId().GetValue(), 1+10)
	})

	t.Run("根据name和namespace删除非master版本的熔断规则", func(t *testing.T) {
		// 创建熔断规则
		_, cbResp := createCommonCircuitBreaker(t, 0)
		defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		// 创建熔断规则版本
		_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 0)
		defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// 创建熔断规则版本
		for i := 1; i <= 10; i++ {
			_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, i)
			defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
		}

		// 删除特定版本的熔断规则
		rule := &api.CircuitBreaker{
			Version:   cbVersionResp.GetVersion(),
			Name:      cbVersionResp.GetName(),
			Namespace: cbVersionResp.GetNamespace(),
			Token:     cbVersionResp.GetToken(),
		}
		deleteCircuitBreaker(t, rule)

		getCircuitBreakerVersions(t, cbResp.GetId().GetValue(), 1+10)
	})

	t.Run("删除不存在的熔断规则，返回成功", func(t *testing.T) {
		// 创建熔断规则
		_, cbResp := createCommonCircuitBreaker(t, 0)
		cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		deleteCircuitBreaker(t, cbResp)
		getCircuitBreakerVersions(t, cbResp.GetId().GetValue(), 0)
	})

	t.Run("删除熔断规则时，没有传递token，返回错误", func(t *testing.T) {
		// 创建熔断规则
		_, cbResp := createCommonCircuitBreaker(t, 0)
		defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		rule := &api.CircuitBreaker{
			Id:      cbResp.GetId(),
			Version: cbResp.GetVersion(),
		}

		if resp := server.DeleteCircuitBreaker(defaultCtx, rule); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("删除熔断规则时，没有传递name和id，返回错误", func(t *testing.T) {
		// 创建熔断规则
		_, cbResp := createCommonCircuitBreaker(t, 0)
		defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		rule := &api.CircuitBreaker{
			Version:   cbResp.GetVersion(),
			Namespace: cbResp.GetNamespace(),
			Token:     cbResp.GetToken(),
		}

		if resp := server.DeleteCircuitBreaker(defaultCtx, rule); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("删除已发布的规则，返回错误", func(t *testing.T) {
		// 创建服务
		_, serviceResp := createCommonService(t, 0)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		// 创建熔断规则
		_, cbResp := createCommonCircuitBreaker(t, 0)
		defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		// 创建熔断规则版本
		_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 0)
		defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// 发布熔断规则
		releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// 删除master版本
		if resp := server.DeleteCircuitBreaker(defaultCtx, cbResp); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}

		// 删除其他版本
		if resp := server.DeleteCircuitBreaker(defaultCtx, cbVersionResp); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("创建多个版本的规则，并发布其中一个规则，删除未发布规则，可以正常删除", func(t *testing.T) {
		// 创建服务
		_, serviceResp := createCommonService(t, 0)
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		// 创建熔断规则
		_, cbResp := createCommonCircuitBreaker(t, 0)
		defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

		// 创建熔断规则版本
		_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 0)
		defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// 创建熔断规则版本
		_, newCbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 1)
		defer cleanCircuitBreaker(newCbVersionResp.GetId().GetValue(), newCbVersionResp.GetVersion().GetValue())

		// 发布熔断规则
		releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		deleteCircuitBreaker(t, newCbVersionResp)
		getCircuitBreakerVersions(t, cbResp.GetId().GetValue(), 1+1)
	})

	t.Run("并发删除熔断规则，可以正常删除", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 1; i <= 500; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				_, cbResp := createCommonCircuitBreaker(t, index)
				defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())
				deleteCircuitBreaker(t, cbResp)
			}(i)
		}
		wg.Wait()
		t.Log("pass")
	})
}

/**
 * @brief 测试更新熔断规则
 */
func TestUpdateCircuitBreaker(t *testing.T) {
	// 创建熔断规则
	_, cbResp := createCommonCircuitBreaker(t, 0)
	defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

	t.Run("更新master版本的熔断规则，返回成功", func(t *testing.T) {
		cbResp.Inbounds = []*api.CbRule{}
		updateCircuitBreaker(t, cbResp)

		filters := map[string]string{
			"id":      cbResp.GetId().GetValue(),
			"version": cbResp.GetVersion().GetValue(),
		}

		resp := server.GetCircuitBreaker(filters)
		if !respSuccess(resp) {
			t.Fatal("error")
		}
		checkCircuitBreaker(t, cbResp, cbResp, resp.GetConfigWithServices()[0].GetCircuitBreaker())
	})

	t.Run("没有更新任何字段，返回不需要更新", func(t *testing.T) {
		if resp := server.UpdateCircuitBreaker(defaultCtx, cbResp); respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("没有传递任何可更新的字段，返回不需要更新", func(t *testing.T) {
		rule := &api.CircuitBreaker{
			Id:      cbResp.GetId(),
			Version: cbResp.GetVersion(),
			Token:   cbResp.GetToken(),
		}
		if resp := server.UpdateCircuitBreaker(defaultCtx, rule); respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("负责人为空，返回错误", func(t *testing.T) {
		rule := &api.CircuitBreaker{
			Id:      cbResp.GetId(),
			Version: cbResp.GetVersion(),
			Token:   cbResp.GetToken(),
			Owners:  utils.NewStringValue(""),
		}
		if resp := server.UpdateCircuitBreaker(defaultCtx, rule); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("更新其他版本的熔断规则，返回错误", func(t *testing.T) {
		// 创建熔断规则版本
		_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 0)
		defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		if resp := server.UpdateCircuitBreaker(defaultCtx, cbVersionResp); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("更新不存在的熔断规则，返回错误", func(t *testing.T) {
		cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())
		if resp := server.UpdateCircuitBreaker(defaultCtx, cbResp); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("更新熔断规则时，没有传递token，返回错误", func(t *testing.T) {
		rule := &api.CircuitBreaker{
			Id:      cbResp.GetId(),
			Version: cbResp.GetVersion(),
		}
		if resp := server.UpdateCircuitBreaker(defaultCtx, rule); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("并发更新熔断规则时,可以正常更新", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 1; i <= 500; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				// 创建熔断规则
				_, cbResp := createCommonCircuitBreaker(t, index)
				defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

				cbResp.Owners = utils.NewStringValue(fmt.Sprintf("test-owner-%d", index))

				updateCircuitBreaker(t, cbResp)

				filters := map[string]string{
					"id":      cbResp.GetId().GetValue(),
					"version": cbResp.GetVersion().GetValue(),
				}
				resp := server.GetCircuitBreaker(filters)
				if !respSuccess(resp) {
					t.Fatal("error")
				}
				checkCircuitBreaker(t, cbResp, cbResp, resp.GetConfigWithServices()[0].GetCircuitBreaker())
			}(i)
		}
		wg.Wait()
	})
}

/**
 * @brief 测试发布熔断规则
 */
func TestReleaseCircuitBreaker(t *testing.T) {
	// 创建服务
	_, serviceResp := createCommonService(t, 0)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	// 创建熔断规则
	_, cbResp := createCommonCircuitBreaker(t, 0)
	defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

	// 创建熔断规则的版本
	_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 0)
	defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

	t.Run("正常发布熔断规则", func(t *testing.T) {
		_ = server.Cache().Clear()

		releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// 等待缓存更新
		time.Sleep(updateCacheInterval)

		resp := server.GetCircuitBreakerWithCache(defaultCtx, serviceResp)
		checkCircuitBreaker(t, cbVersionResp, cbResp, resp.GetCircuitBreaker())
	})

	t.Run("根据name和namespace发布熔断规则", func(t *testing.T) {
		_ = server.Cache().Clear()

		rule := &api.CircuitBreaker{
			Version:   cbVersionResp.GetVersion(),
			Name:      cbVersionResp.GetName(),
			Namespace: cbVersionResp.GetNamespace(),
		}
		releaseCircuitBreaker(t, rule, serviceResp)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// 等待缓存更新
		time.Sleep(updateCacheInterval)

		resp := server.GetCircuitBreakerWithCache(defaultCtx, serviceResp)
		checkCircuitBreaker(t, cbVersionResp, cbResp, resp.GetCircuitBreaker())
	})

	t.Run("为同一个服务发布多条不同熔断规则", func(t *testing.T) {
		_ = server.Cache().Clear()

		releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// 创建熔断规则的版本
		_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 1)
		defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// 再次发布熔断规则
		releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// 等待缓存更新
		time.Sleep(updateCacheInterval)

		resp := server.GetCircuitBreakerWithCache(defaultCtx, serviceResp)
		checkCircuitBreaker(t, cbVersionResp, cbResp, resp.GetCircuitBreaker())
	})

	t.Run("为不同服务发布相同熔断规则，返回成功", func(t *testing.T) {
		_ = server.Cache().Clear()

		releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// 创建服务
		_, serviceResp2 := createCommonService(t, 1)
		defer cleanServiceName(serviceResp2.GetName().GetValue(), serviceResp2.GetNamespace().GetValue())

		releaseCircuitBreaker(t, cbVersionResp, serviceResp2)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// 等待缓存更新
		time.Sleep(updateCacheInterval)

		resp := server.GetCircuitBreakerWithCache(defaultCtx, serviceResp)
		checkCircuitBreaker(t, cbVersionResp, cbResp, resp.GetCircuitBreaker())

		resp = server.GetCircuitBreakerWithCache(defaultCtx, serviceResp2)
		checkCircuitBreaker(t, cbVersionResp, cbResp, resp.GetCircuitBreaker())
	})

	t.Run("规则命名空间与服务命名空间不一致，返回错误", func(t *testing.T) {
		release := &api.ConfigRelease{
			Service: &api.Service{
				Name:      serviceResp.GetName(),
				Namespace: utils.NewStringValue("Test"),
				Token:     serviceResp.GetToken(),
			},
			CircuitBreaker: cbVersionResp,
		}

		if resp := server.ReleaseCircuitBreaker(defaultCtx, release); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("为同一个服务发布多条相同熔断规则，返回错误", func(t *testing.T) {
		releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		release := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: cbVersionResp,
		}

		if resp := server.ReleaseCircuitBreaker(defaultCtx, release); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("发布熔断规则时，没有传递token，返回错误", func(t *testing.T) {
		release := &api.ConfigRelease{
			Service: &api.Service{
				Name:      serviceResp.GetName(),
				Namespace: serviceResp.GetNamespace(),
			},
			CircuitBreaker: cbVersionResp,
		}
		if resp := server.ReleaseCircuitBreaker(defaultCtx, release); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("发布服务不存在的熔断规则，返回错误", func(t *testing.T) {
		_, serviceResp := createCommonService(t, 1)
		cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		release := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: cbVersionResp,
		}
		if resp := server.ReleaseCircuitBreaker(defaultCtx, release); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("发布的熔断规则为master版本，返回错误", func(t *testing.T) {
		release := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: cbResp,
		}
		if resp := server.ReleaseCircuitBreaker(defaultCtx, release); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("发布不存在的熔断规则，返回错误", func(t *testing.T) {
		_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 1)
		cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		release := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: cbVersionResp,
		}
		if resp := server.ReleaseCircuitBreaker(defaultCtx, release); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("并发发布同一个服务的熔断规则", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 1; i <= 500; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, index)
				defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

				releaseCircuitBreaker(t, cbVersionResp, serviceResp)
				defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
					cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())
			}(i)
		}
		wg.Wait()
		t.Log("pass")
	})
}

/**
 * @brief 测试解绑熔断规则
 */
func TestUnBindCircuitBreaker(t *testing.T) {
	// 创建服务
	_, serviceResp := createCommonService(t, 0)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	// 创建熔断规则
	_, cbResp := createCommonCircuitBreaker(t, 0)
	defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

	// 创建熔断规则的版本
	_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 0)
	defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

	t.Run("正常解绑熔断规则", func(t *testing.T) {
		_ = server.Cache().Clear()

		// 发布熔断规则
		releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		unBindCircuitBreaker(t, cbVersionResp, serviceResp)

		// 等待缓存更新
		time.Sleep(updateCacheInterval)

		resp := server.GetCircuitBreakerWithCache(defaultCtx, serviceResp)
		if resp != nil && resp.GetCircuitBreaker() == nil {
			t.Log("pass")
		} else {
			t.Fatalf("err is %+v", resp)
		}
	})

	t.Run("解绑关系不存在的熔断规则, 返回成功", func(t *testing.T) {
		_ = server.Cache().Clear()

		// 发布熔断规则
		releaseCircuitBreaker(t, cbVersionResp, serviceResp)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		// 创建熔断规则的版本
		_, newCbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 1)
		defer cleanCircuitBreaker(newCbVersionResp.GetId().GetValue(), newCbVersionResp.GetVersion().GetValue())

		unBindCircuitBreaker(t, newCbVersionResp, serviceResp)

		// 等待缓存更新
		time.Sleep(updateCacheInterval)

		resp := server.GetCircuitBreakerWithCache(defaultCtx, serviceResp)
		checkCircuitBreaker(t, cbVersionResp, cbResp, resp.GetCircuitBreaker())
	})

	t.Run("解绑规则时没有传递token，返回错误", func(t *testing.T) {
		unbind := &api.ConfigRelease{
			Service: &api.Service{
				Name:      serviceResp.GetName(),
				Namespace: serviceResp.GetNamespace(),
			},
			CircuitBreaker: cbVersionResp,
		}

		if resp := server.UnBindCircuitBreaker(defaultCtx, unbind); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("解绑服务不存在的熔断规则，返回错误", func(t *testing.T) {
		_, serviceResp := createCommonService(t, 1)
		cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		unbind := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: cbVersionResp,
		}

		if resp := server.UnBindCircuitBreaker(defaultCtx, unbind); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("解绑规则不存在的熔断规则，返回错误", func(t *testing.T) {
		// 创建熔断规则的版本
		_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, 1)
		cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		unbind := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: cbVersionResp,
		}

		if resp := server.UnBindCircuitBreaker(defaultCtx, unbind); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("解绑master版本的熔断规则，返回错误", func(t *testing.T) {
		unbind := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: cbResp,
		}

		if resp := server.UnBindCircuitBreaker(defaultCtx, unbind); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("解绑熔断规则时没有传递name，返回错误", func(t *testing.T) {
		unbind := &api.ConfigRelease{
			Service: serviceResp,
			CircuitBreaker: &api.CircuitBreaker{
				Version:   cbVersionResp.GetVersion(),
				Namespace: cbVersionResp.GetNamespace(),
			},
		}

		if resp := server.UnBindCircuitBreaker(defaultCtx, unbind); !respSuccess(resp) {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatal("error")
		}
	})

	t.Run("并发解绑熔断规则", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 1; i <= 500; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				// 创建服务
				_, serviceResp := createCommonService(t, index)
				defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

				// 发布熔断规则
				releaseCircuitBreaker(t, cbVersionResp, serviceResp)
				defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
					cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

				unBindCircuitBreaker(t, cbVersionResp, serviceResp)
			}(i)
		}
		wg.Wait()
		t.Log("pass")
	})
}

/**
 * @brief 测试查询熔断规则
 */
func TestGetCircuitBreaker(t *testing.T) {
	versionNum := 10
	serviceNum := 2
	releaseVersion := &api.CircuitBreaker{}
	deleteVersion := &api.CircuitBreaker{}
	service := &api.Service{}

	// 创建熔断规则
	_, cbResp := createCommonCircuitBreaker(t, 0)
	defer cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

	// 创建熔断规则版本
	for i := 1; i <= versionNum; i++ {
		// 创建熔断规则的版本
		_, cbVersionResp := createCommonCircuitBreakerVersion(t, cbResp, i)
		defer cleanCircuitBreaker(cbVersionResp.GetId().GetValue(), cbVersionResp.GetVersion().GetValue())

		if i == 5 {
			releaseVersion = cbVersionResp
		}

		if i == versionNum {
			deleteVersion = cbVersionResp
		}
	}

	// 删除一个版本的熔断规则
	deleteCircuitBreaker(t, deleteVersion)

	// 发布熔断规则
	for i := 1; i <= serviceNum; i++ {
		_, serviceResp := createCommonService(t, i)
		if i == 1 {
			service = serviceResp
		}
		defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

		releaseCircuitBreaker(t, releaseVersion, serviceResp)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			releaseVersion.GetId().GetValue(), releaseVersion.GetVersion().GetValue())
	}

	t.Run("测试获取熔断规则的所有版本", func(t *testing.T) {
		filters := map[string]string{
			"id": cbResp.GetId().GetValue(),
		}

		resp := server.GetCircuitBreakerVersions(filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetAmount().GetValue() != resp.GetSize().GetValue() ||
			resp.GetSize().GetValue() != uint32(versionNum) || len(resp.GetConfigWithServices()) != versionNum {
			t.Fatalf("amount is %d, size is %d, num is %d, expect num is %d", resp.GetAmount().GetValue(),
				resp.GetSize().GetValue(), len(resp.GetConfigWithServices()), versionNum)
		}
		t.Logf("pass: num is %d", resp.GetSize().GetValue())
	})

	t.Run("测试获取熔断规则创建过的版本", func(t *testing.T) {
		filters := map[string]string{
			"id": cbResp.GetId().GetValue(),
		}

		resp := server.GetReleaseCircuitBreakers(filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetAmount().GetValue() != resp.GetSize().GetValue() ||
			resp.GetSize().GetValue() != uint32(serviceNum) {
			t.Fatalf("amount is %d, size is %d, expect num is %d", resp.GetAmount().GetValue(),
				resp.GetSize().GetValue(), versionNum)
		}
		t.Logf("pass: num is %d", resp.GetSize().GetValue())
	})

	t.Run("测试获取指定版本的熔断规则", func(t *testing.T) {
		filters := map[string]string{
			"id":      releaseVersion.GetId().GetValue(),
			"version": releaseVersion.GetVersion().GetValue(),
		}

		resp := server.GetCircuitBreaker(filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		checkCircuitBreaker(t, releaseVersion, cbResp, resp.GetConfigWithServices()[0].GetCircuitBreaker())
	})

	t.Run("根据服务获取绑定的熔断规则", func(t *testing.T) {
		filters := map[string]string{
			"service":   service.GetName().GetValue(),
			"namespace": service.GetNamespace().GetValue(),
		}

		resp := server.GetCircuitBreakerByService(filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		checkCircuitBreaker(t, releaseVersion, cbResp, resp.GetConfigWithServices()[0].GetCircuitBreaker())
	})
}

/**
 * @brief 测试查询熔断规则
 */
func TestGetCircuitBreaker2(t *testing.T) {
	// 创建服务
	_, serviceResp := createCommonService(t, 0)
	defer cleanServiceName(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	// 创建熔断规则
	_, cbResp := createCommonCircuitBreaker(t, 0)
	cleanCircuitBreaker(cbResp.GetId().GetValue(), cbResp.GetVersion().GetValue())

	t.Run("熔断规则不存在，测试获取所有版本", func(t *testing.T) {
		filters := map[string]string{
			"id": cbResp.GetId().GetValue(),
		}

		resp := server.GetCircuitBreakerVersions(filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetAmount().GetValue() != 0 || resp.GetSize().GetValue() != 0 ||
			len(resp.GetConfigWithServices()) != 0 {
			t.Fatalf("amount is %d, size is %d, num is %d", resp.GetAmount().GetValue(),
				resp.GetSize().GetValue(), len(resp.GetConfigWithServices()))
		}
		t.Logf("pass: resp is %+v, configServices is %+v", resp, resp.GetConfigWithServices())
	})

	t.Run("熔断规则不存在，测试获取所有创建过的版本", func(t *testing.T) {
		filters := map[string]string{
			"id": cbResp.GetId().GetValue(),
		}

		resp := server.GetReleaseCircuitBreakers(filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetAmount().GetValue() != 0 || resp.GetSize().GetValue() != 0 ||
			len(resp.GetConfigWithServices()) != 0 {
			t.Fatalf("amount is %d, size is %d, num is %d", resp.GetAmount().GetValue(),
				resp.GetSize().GetValue(), len(resp.GetConfigWithServices()))
		}
		t.Logf("pass: resp is %+v, configServices is %+v", resp, resp.GetConfigWithServices())
	})

	t.Run("熔断规则不存在，测试获取指定版本的熔断规则", func(t *testing.T) {
		filters := map[string]string{
			"id":      cbResp.GetId().GetValue(),
			"version": cbResp.GetVersion().GetValue(),
		}

		resp := server.GetCircuitBreaker(filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetAmount().GetValue() != 0 || resp.GetSize().GetValue() != 0 ||
			len(resp.GetConfigWithServices()) != 0 {
			t.Fatalf("amount is %d, size is %d, num is %d", resp.GetAmount().GetValue(),
				resp.GetSize().GetValue(), len(resp.GetConfigWithServices()))
		}
		t.Logf("pass: resp is %+v, configServices is %+v", resp, resp.GetConfigWithServices())
	})

	t.Run("服务未绑定熔断规则，获取熔断规则", func(t *testing.T) {
		filters := map[string]string{
			"service":   serviceResp.GetName().GetValue(),
			"namespace": serviceResp.GetNamespace().GetValue(),
		}

		resp := server.GetCircuitBreakerByService(filters)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		if resp.GetAmount().GetValue() != 0 || resp.GetSize().GetValue() != 0 ||
			len(resp.GetConfigWithServices()) != 0 {
			t.Fatalf("amount is %d, size is %d, num is %d", resp.GetAmount().GetValue(),
				resp.GetSize().GetValue(), len(resp.GetConfigWithServices()))
		}
		t.Logf("pass: resp is %+v, configServices is %+v", resp, resp.GetConfigWithServices())
	})
}

// test对CircuitBreaker字段进行校验
func TestCheckCircuitBreakerFieldLen(t *testing.T) {
	circuitBreaker := &api.CircuitBreaker{
		Name:       utils.NewStringValue("name-test-123"),
		Namespace:  utils.NewStringValue(naming.DefaultNamespace),
		Owners:     utils.NewStringValue("owner-test"),
		Comment:    utils.NewStringValue("comment-test"),
		Department: utils.NewStringValue("department-test"),
		Business:   utils.NewStringValue("business-test"),
	}
	t.Run("熔断名超长", func(t *testing.T) {
		str := genSpecialStr(33)
		oldName := circuitBreaker.Name
		circuitBreaker.Name = utils.NewStringValue(str)
		resp := server.CreateCircuitBreaker(defaultCtx, circuitBreaker)
		circuitBreaker.Name = oldName
		if resp.Code.Value != api.InvalidCircuitBreakerName {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("熔断命名空间超长", func(t *testing.T) {
		str := genSpecialStr(65)
		oldNamespace := circuitBreaker.Namespace
		circuitBreaker.Namespace = utils.NewStringValue(str)
		resp := server.CreateCircuitBreaker(defaultCtx, circuitBreaker)
		circuitBreaker.Namespace = oldNamespace
		if resp.Code.Value != api.InvalidCircuitBreakerNamespace {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("熔断business超长", func(t *testing.T) {
		str := genSpecialStr(65)
		oldBusiness := circuitBreaker.Business
		circuitBreaker.Business = utils.NewStringValue(str)
		resp := server.CreateCircuitBreaker(defaultCtx, circuitBreaker)
		circuitBreaker.Business = oldBusiness
		if resp.Code.Value != api.InvalidCircuitBreakerBusiness {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("熔断部门超长", func(t *testing.T) {
		str := genSpecialStr(1025)
		oldDepartment := circuitBreaker.Department
		circuitBreaker.Department = utils.NewStringValue(str)
		resp := server.CreateCircuitBreaker(defaultCtx, circuitBreaker)
		circuitBreaker.Department = oldDepartment
		if resp.Code.Value != api.InvalidCircuitBreakerDepartment {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("熔断comment超长", func(t *testing.T) {
		str := genSpecialStr(1025)
		oldComment := circuitBreaker.Comment
		circuitBreaker.Comment = utils.NewStringValue(str)
		resp := server.CreateCircuitBreaker(defaultCtx, circuitBreaker)
		circuitBreaker.Comment = oldComment
		if resp.Code.Value != api.InvalidCircuitBreakerComment {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("熔断owner超长", func(t *testing.T) {
		str := genSpecialStr(1025)
		oldOwners := circuitBreaker.Owners
		circuitBreaker.Owners = utils.NewStringValue(str)
		resp := server.CreateCircuitBreaker(defaultCtx, circuitBreaker)
		circuitBreaker.Owners = oldOwners
		if resp.Code.Value != api.InvalidCircuitBreakerOwners {
			t.Fatalf("%+v", resp)
		}
	})
	t.Run("发布熔断规则超长", func(t *testing.T) {
		release := &api.ConfigRelease{
			Service: &api.Service{
				Name:      utils.NewStringValue("test"),
				Namespace: utils.NewStringValue("default"),
				Token:     utils.NewStringValue("test"),
			},
			CircuitBreaker: &api.CircuitBreaker{
				Name:      utils.NewStringValue("test"),
				Namespace: utils.NewStringValue("default"),
				Version:   utils.NewStringValue("1.0"),
			},
		}
		t.Run("发布熔断规则服务名超长", func(t *testing.T) {
			str := genSpecialStr(1025)
			oldName := release.Service.Name
			release.Service.Name = utils.NewStringValue(str)
			resp := server.ReleaseCircuitBreaker(defaultCtx, release)
			release.Service.Name = oldName
			if resp.Code.Value != api.InvalidServiceName {
				t.Fatalf("%+v", resp)
			}
		})
		t.Run("发布熔断规则服务命名空间超长", func(t *testing.T) {
			str := genSpecialStr(1025)
			oldNamespace := release.Service.Namespace
			release.Service.Namespace = utils.NewStringValue(str)
			resp := server.ReleaseCircuitBreaker(defaultCtx, release)
			release.Service.Namespace = oldNamespace
			if resp.Code.Value != api.InvalidNamespaceName {
				t.Fatalf("%+v", resp)
			}
		})
		t.Run("发布熔断规则服务token超长", func(t *testing.T) {
			str := genSpecialStr(2049)
			oldToken := release.Service.Token
			release.Service.Token = utils.NewStringValue(str)
			resp := server.ReleaseCircuitBreaker(defaultCtx, release)
			release.Service.Token = oldToken
			if resp.Code.Value != api.InvalidServiceToken {
				t.Fatalf("%+v", resp)
			}
		})
		t.Run("发布熔断规则熔断名超长", func(t *testing.T) {
			str := genSpecialStr(1025)
			oldName := release.CircuitBreaker.Name
			release.CircuitBreaker.Name = utils.NewStringValue(str)
			resp := server.ReleaseCircuitBreaker(defaultCtx, release)
			release.CircuitBreaker.Name = oldName
			if resp.Code.Value != api.InvalidCircuitBreakerName {
				t.Fatalf("%+v", resp)
			}
		})
		t.Run("发布熔断规则熔断命名空间超长", func(t *testing.T) {
			str := genSpecialStr(1025)
			oldNamespace := release.CircuitBreaker.Namespace
			release.CircuitBreaker.Namespace = utils.NewStringValue(str)
			resp := server.ReleaseCircuitBreaker(defaultCtx, release)
			release.CircuitBreaker.Namespace = oldNamespace
			if resp.Code.Value != api.InvalidCircuitBreakerNamespace {
				t.Fatalf("%+v", resp)
			}
		})
		t.Run("发布熔断规则熔断version超长", func(t *testing.T) {
			str := genSpecialStr(1025)
			oldVersion := release.CircuitBreaker.Version
			release.CircuitBreaker.Version = utils.NewStringValue(str)
			resp := server.ReleaseCircuitBreaker(defaultCtx, release)
			release.CircuitBreaker.Version = oldVersion
			if resp.Code.Value != api.InvalidCircuitBreakerVersion {
				t.Fatalf("%+v", resp)
			}
		})
	})

}
