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

package testAuthPlugin

import (
	"context"
	"fmt"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/golang/protobuf/ptypes/duration"
	"testing"
)

/**
 * @brief 测试使用平台Token操作限流规则
 */
func TestRateLimitAuthByPlatform(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), platformToken)

	// 创建服务
	serviceResp := createService(t, 4)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("使用平台Token创建限流规则，有权限", func(t *testing.T) {
		rateLimit := createRateLimit(t, serviceResp, 1, ctx)
		defer cleanRateLimit(rateLimit.GetId().GetValue())
		t.Log("pass")
	})

	t.Run("使用平台Token修改限流规则，有权限", func(t *testing.T) {
		rateLimit := createRateLimit(t, serviceResp, 2, ctx)
		defer cleanRateLimit(rateLimit.GetId().GetValue())

		rateLimit.Labels = map[string]*api.MatchString{}
		resp := server.UpdateRateLimit(ctx, rateLimit)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("使用平台Token删除限流规则，有权限", func(t *testing.T) {
		rateLimit := createRateLimit(t, serviceResp, 3, ctx)
		defer cleanRateLimit(rateLimit.GetId().GetValue())

		resp := server.DeleteRateLimit(ctx, rateLimit)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})
}

/**
 * @brief 测试使用服务Token创建限流规则
 */
func TestCreateRateLimitAuthByService(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), "test")

	correctCtx := context.Background()
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-id"), platformID)
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-token"), platformToken)

	// 创建服务
	serviceResp := createService(t, 6)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("平台信息为空，使用服务Token创建限流规则，有权限", func(t *testing.T) {
		req := &api.Rule{
			Service:   serviceResp.GetName(),
			Namespace: serviceResp.GetNamespace(),
			Labels: map[string]*api.MatchString{
				"111": {
					Type:  0,
					Value: utils.NewStringValue("aaa"),
				},
			},
			ServiceToken: serviceResp.GetToken(),
		}

		resp := server.CreateRateLimit(defaultCtx, req)
		defer cleanRateLimit(resp.GetRateLimit().GetId().GetValue())

		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token创建限流规则，有权限", func(t *testing.T) {
		req := &api.Rule{
			Service:      serviceResp.GetName(),
			Namespace:    serviceResp.GetNamespace(),
			ServiceToken: serviceResp.GetToken(),
			Labels: map[string]*api.MatchString{
				"111": {
					Type:  0,
					Value: utils.NewStringValue("aaa"),
				},
			},
		}

		resp := server.CreateRateLimit(ctx, req)
		defer cleanRateLimit(resp.GetInstance().GetId().GetValue())

		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token创建限流规则，有权限", func(t *testing.T) {
		req := &api.Rule{
			Service:   serviceResp.GetName(),
			Namespace: serviceResp.GetNamespace(),
			Labels: map[string]*api.MatchString{
				"111": {
					Type:  0,
					Value: utils.NewStringValue("aaa"),
				},
			},
		}

		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		resp := server.CreateRateLimit(globalCtx, req)
		defer cleanRateLimit(resp.GetInstance().GetId().GetValue())

		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，创建限流规则，返回错误", func(t *testing.T) {
		req := &api.Rule{
			Service:   serviceResp.GetName(),
			Namespace: serviceResp.GetNamespace(),
			Labels: map[string]*api.MatchString{
				"111": {
					Type:  0,
					Value: utils.NewStringValue("aaa"),
				},
			},
		}

		resp := server.CreateRateLimit(defaultCtx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Log("pass")
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，创建限流规则，返回错误", func(t *testing.T) {
		req := &api.Rule{
			Service:   serviceResp.GetName(),
			Namespace: serviceResp.GetNamespace(),
			Labels: map[string]*api.MatchString{
				"111": {
					Type:  0,
					Value: utils.NewStringValue("aaa"),
				},
			},
			ServiceToken: utils.NewStringValue("test"),
		}

		resp := server.CreateRateLimit(defaultCtx, req)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Log("pass")
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 测试使用服务Token修改限流规则
 */
func TestUpdateRateLimitAuthByService(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), "test")

	correctCtx := context.Background()
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-id"), platformID)
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-token"), platformToken)

	// 创建服务
	serviceResp := createService(t, 6)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("平台信息为空，使用服务Token修改限流规则，有权限", func(t *testing.T) {
		req := createRateLimit(t, serviceResp, 22, correctCtx)
		defer cleanRateLimit(req.GetId().GetValue())

		req.RegexCombine = utils.NewBoolValue(true)
		req.ServiceToken = serviceResp.GetToken()
		resp := server.UpdateRateLimit(defaultCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token修改限流规则，有权限", func(t *testing.T) {
		req := createRateLimit(t, serviceResp, 33, correctCtx)
		defer cleanRateLimit(req.GetId().GetValue())

		req.RegexCombine = utils.NewBoolValue(true)
		req.ServiceToken = serviceResp.GetToken()
		resp := server.UpdateRateLimit(ctx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token修改限流规则，有权限", func(t *testing.T) {
		req := createRateLimit(t, serviceResp, 33, correctCtx)
		defer cleanRateLimit(req.GetId().GetValue())

		req.RegexCombine = utils.NewBoolValue(true)

		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		resp := server.UpdateRateLimit(globalCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，修改限流规则，返回错误", func(t *testing.T) {
		req := createRateLimit(t, serviceResp, 44, correctCtx)
		defer cleanRateLimit(req.GetId().GetValue())

		resp := server.UpdateRateLimit(defaultCtx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，修改限流规则，返回错误", func(t *testing.T) {
		req := createRateLimit(t, serviceResp, 55, correctCtx)
		defer cleanRateLimit(req.GetId().GetValue())

		req.ServiceToken = utils.NewStringValue("test")
		resp := server.UpdateRateLimit(ctx, req)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 测试使用服务Token删除限流规则
 */
func TestDeleteRateLimitAuthByService(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), "test")

	correctCtx := context.Background()
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-id"), platformID)
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-token"), platformToken)

	// 创建服务
	serviceResp := createService(t, 6)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("平台信息为空，使用服务Token删除限流规则，有权限", func(t *testing.T) {
		req := createRateLimit(t, serviceResp, 66, correctCtx)
		defer cleanRateLimit(req.GetId().GetValue())

		req.ServiceToken = serviceResp.GetToken()
		resp := server.DeleteRateLimit(defaultCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token删除限流规则，有权限", func(t *testing.T) {
		req := createRateLimit(t, serviceResp, 77, correctCtx)
		defer cleanRateLimit(req.GetId().GetValue())

		req.ServiceToken = serviceResp.GetToken()
		resp := server.DeleteRateLimit(ctx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token删除限流规则，有权限", func(t *testing.T) {
		req := createRateLimit(t, serviceResp, 77, correctCtx)
		defer cleanRateLimit(req.GetId().GetValue())

		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		resp := server.DeleteRateLimit(globalCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，删除限流规则，返回错误", func(t *testing.T) {
		req := createRateLimit(t, serviceResp, 88, correctCtx)
		defer cleanRateLimit(req.GetId().GetValue())

		resp := server.DeleteRateLimit(defaultCtx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，删除限流规则，返回错误", func(t *testing.T) {
		req := createRateLimit(t, serviceResp, 99, correctCtx)
		defer cleanRateLimit(req.GetId().GetValue())

		req.ServiceToken = utils.NewStringValue("test")
		resp := server.DeleteRateLimit(ctx, req)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 创建限流规则
 */
func createRateLimit(t *testing.T, service *api.Service, id int, ctx context.Context) *api.Rule {
	rateLimit := &api.Rule{
		Service:   service.GetName(),
		Namespace: service.GetNamespace(),
		Priority:  utils.NewUInt32Value(uint32(id)),
		Resource:  api.Rule_QPS,
		Type:      api.Rule_GLOBAL,
		Labels: map[string]*api.MatchString{
			fmt.Sprintf("name-%d", id): {
				Type:  api.MatchString_EXACT,
				Value: utils.NewStringValue(fmt.Sprintf("value-%d", id)),
			},
			fmt.Sprintf("name-%d", id+1): {
				Type:  api.MatchString_REGEX,
				Value: utils.NewStringValue(fmt.Sprintf("value-%d", id+1)),
			},
		},
		Amounts: []*api.Amount{
			{
				MaxAmount: utils.NewUInt32Value(uint32(10 * id)),
				ValidDuration: &duration.Duration{
					Seconds: int64(id),
					Nanos:   int32(id),
				},
			},
		},
		Action:  utils.NewStringValue(fmt.Sprintf("behavior-%d", id)),
		Disable: utils.NewBoolValue(false),
		Report: &api.Report{
			Interval: &duration.Duration{
				Seconds: int64(id),
			},
			AmountPercent: utils.NewUInt32Value(uint32(id)),
		},
	}

	resp := server.CreateRateLimit(ctx, rateLimit)
	if !respSuccess(resp) {
		t.Fatalf("error: %+v", resp)
	}

	return resp.GetRateLimit()
}

/**
 * @brief 从数据库中删除限流规则
 */
func cleanRateLimit(id string) {
	str := `delete from ratelimit_config where id = ?`
	if _, err := db.Exec(str, id); err != nil {
		panic(err)
	}
}
