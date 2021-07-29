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
	"github.com/polarismesh/polaris-server/naming"
	"testing"
)

/**
 * @brief 测试使用平台Token绑定/解绑熔断规则
 */
func TestCircuitBreakerAuthByPlatform(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), platformToken)

	// 创建熔断规则
	circuitBreaker := createCircuitBreaker(t, 1)
	defer cleanCircuitBreaker(circuitBreaker.GetId().GetValue(), circuitBreaker.GetVersion().GetValue())

	// 创建熔断规则版本
	resp := createCircuitBreakerVersion(t, circuitBreaker)
	defer cleanCircuitBreaker(resp.GetId().GetValue(), resp.GetVersion().GetValue())

	// 创建服务
	serviceResp := createService(t, 334)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("使用平台Token发布熔断规则，有权限", func(t *testing.T) {
		releaseCircuitBreaker(t, circuitBreaker, serviceResp, ctx)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			circuitBreaker.GetId().GetValue(), circuitBreaker.GetVersion().GetValue())

		t.Log("pass")
	})

	t.Run("使用平台Token解绑熔断规则, 有权限", func(t *testing.T) {
		releaseCircuitBreaker(t, circuitBreaker, serviceResp, ctx)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			circuitBreaker.GetId().GetValue(), circuitBreaker.GetVersion().GetValue())

		unbind := &api.ConfigRelease{
			Service: &api.Service{
				Name:      serviceResp.GetName(),
				Namespace: serviceResp.GetNamespace(),
			},
			CircuitBreaker: circuitBreaker,
		}

		resp := server.UnBindCircuitBreaker(ctx, unbind)
		if !respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		t.Log("pass")
	})
}

/**
 * @brief 测试使用服务Token绑定熔断规则
 */
func TestReleaseCircuitBreakerAuthByService(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), "test")

	correctCtx := context.Background()
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-id"), platformID)
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-token"), platformToken)

	// 创建熔断规则
	circuitBreaker := createCircuitBreaker(t, 2)
	defer cleanCircuitBreaker(circuitBreaker.GetId().GetValue(), circuitBreaker.GetVersion().GetValue())

	// 创建熔断规则版本
	resp := createCircuitBreakerVersion(t, circuitBreaker)
	defer cleanCircuitBreaker(resp.GetId().GetValue(), resp.GetVersion().GetValue())

	// 创建服务
	serviceResp := createService(t, 134)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("平台信息为空，使用服务Token发布熔断规则，有权限", func(t *testing.T) {
		req := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: circuitBreaker,
		}

		resp := server.ReleaseCircuitBreaker(defaultCtx, req)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			circuitBreaker.GetId().GetValue(), circuitBreaker.GetVersion().GetValue())

		if !respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token发布熔断规则，有权限", func(t *testing.T) {
		req := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: circuitBreaker,
		}

		resp := server.ReleaseCircuitBreaker(ctx, req)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			circuitBreaker.GetId().GetValue(), circuitBreaker.GetVersion().GetValue())

		if !respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token发布熔断规则，有权限", func(t *testing.T) {
		req := &api.ConfigRelease{
			Service: &api.Service{
				Name:      serviceResp.GetName(),
				Namespace: serviceResp.GetNamespace(),
			},
			CircuitBreaker: circuitBreaker,
		}

		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		resp := server.ReleaseCircuitBreaker(globalCtx, req)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			circuitBreaker.GetId().GetValue(), circuitBreaker.GetVersion().GetValue())

		if !respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，发布熔断规则，返回错误", func(t *testing.T) {
		req := &api.ConfigRelease{
			Service: &api.Service{
				Name:      serviceResp.GetName(),
				Namespace: serviceResp.GetNamespace(),
			},
			CircuitBreaker: circuitBreaker,
		}

		resp := server.ReleaseCircuitBreaker(defaultCtx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Log("pass")
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，发布熔断规则，返回错误", func(t *testing.T) {
		req := &api.ConfigRelease{
			Service: &api.Service{
				Name:      serviceResp.GetName(),
				Namespace: serviceResp.GetNamespace(),
				Token:     utils.NewStringValue("test"),
			},
			CircuitBreaker: circuitBreaker,
		}

		resp := server.ReleaseCircuitBreaker(defaultCtx, req)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Log("pass")
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 测试使用服务Token解绑熔断规则
 */
func TestUnbindCircuitBreakerAuthByService(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), "test")

	correctCtx := context.Background()
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-id"), platformID)
	correctCtx = context.WithValue(correctCtx, utils.StringContext("platform-token"), platformToken)

	// 创建熔断规则
	circuitBreaker := createCircuitBreaker(t, 2)
	defer cleanCircuitBreaker(circuitBreaker.GetId().GetValue(), circuitBreaker.GetVersion().GetValue())

	// 创建熔断规则版本
	resp := createCircuitBreakerVersion(t, circuitBreaker)
	defer cleanCircuitBreaker(resp.GetId().GetValue(), resp.GetVersion().GetValue())

	// 创建服务
	serviceResp := createService(t, 134)
	defer cleanService(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue())

	t.Run("平台信息为空，使用服务Token解绑熔断规则，有权限", func(t *testing.T) {
		releaseCircuitBreaker(t, circuitBreaker, serviceResp, correctCtx)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			circuitBreaker.GetId().GetValue(), circuitBreaker.GetVersion().GetValue())

		req := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: circuitBreaker,
		}

		resp := server.UnBindCircuitBreaker(defaultCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用服务Token解绑熔断规则，有权限", func(t *testing.T) {
		releaseCircuitBreaker(t, circuitBreaker, serviceResp, correctCtx)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			circuitBreaker.GetId().GetValue(), circuitBreaker.GetVersion().GetValue())

		req := &api.ConfigRelease{
			Service:        serviceResp,
			CircuitBreaker: circuitBreaker,
		}

		resp := server.UnBindCircuitBreaker(ctx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		t.Log("pass")
	})

	t.Run("平台信息不正确，使用系统Token解绑熔断规则，有权限", func(t *testing.T) {
		releaseCircuitBreaker(t, circuitBreaker, serviceResp, correctCtx)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			circuitBreaker.GetId().GetValue(), circuitBreaker.GetVersion().GetValue())

		req := &api.ConfigRelease{
			Service: &api.Service{
				Name:      serviceResp.GetName(),
				Namespace: serviceResp.GetNamespace(),
			},
			CircuitBreaker: circuitBreaker,
		}

		globalCtx := context.WithValue(ctx, utils.StringContext("polaris-token"), "polaris@12345678")

		resp := server.UnBindCircuitBreaker(globalCtx, req)
		if !respSuccess(resp) {
			t.Fatalf("error: %+v", resp)
		}
		t.Log("pass")
	})

	t.Run("无服务Token和平台信息，解绑熔断规则，返回错误", func(t *testing.T) {
		releaseCircuitBreaker(t, circuitBreaker, serviceResp, correctCtx)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			circuitBreaker.GetId().GetValue(), circuitBreaker.GetVersion().GetValue())

		req := &api.ConfigRelease{
			Service: &api.Service{
				Name:      serviceResp.GetName(),
				Namespace: serviceResp.GetNamespace(),
			},
			CircuitBreaker: circuitBreaker,
		}

		resp := server.UnBindCircuitBreaker(defaultCtx, req)
		if resp.GetCode().GetValue() == api.InvalidServiceToken {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})

	t.Run("服务Token错误，解绑熔断规则，返回错误", func(t *testing.T) {
		releaseCircuitBreaker(t, circuitBreaker, serviceResp, correctCtx)
		defer cleanCircuitBreakerRelation(serviceResp.GetName().GetValue(), serviceResp.GetNamespace().GetValue(),
			circuitBreaker.GetId().GetValue(), circuitBreaker.GetVersion().GetValue())

		req := &api.ConfigRelease{
			Service: &api.Service{
				Name:      serviceResp.GetName(),
				Namespace: serviceResp.GetNamespace(),
				Token:     utils.NewStringValue("test"),
			},
			CircuitBreaker: circuitBreaker,
		}

		resp := server.UnBindCircuitBreaker(defaultCtx, req)
		if resp.GetCode().GetValue() == api.Unauthorized {
			t.Logf("pass: %s", resp.GetInfo().GetValue())
		} else {
			t.Fatalf("error: %s", resp.GetInfo().GetValue())
		}
	})
}

/**
 * @brief 创建熔断规则
 */
func createCircuitBreaker(t *testing.T, id int) *api.CircuitBreaker {
	circuitBreaker := &api.CircuitBreaker{
		Name:      utils.NewStringValue(fmt.Sprintf("name-test-%d", id)),
		Namespace: utils.NewStringValue(naming.DefaultNamespace),
		Owners:    utils.NewStringValue("owner-test"),
	}
	ruleNum := 1
	// 填充source规则
	sources := make([]*api.SourceMatcher, 0, ruleNum)
	for i := 0; i < ruleNum; i++ {
		source := &api.SourceMatcher{
			Service:   utils.NewStringValue(fmt.Sprintf("service-test-%d", i)),
			Namespace: utils.NewStringValue(fmt.Sprintf("namespace-test-%d", i)),
			Labels: map[string]*api.MatchString{
				fmt.Sprintf("name-%d", i): {
					Type:  api.MatchString_EXACT,
					Value: utils.NewStringValue(fmt.Sprintf("value-%d", i)),
				},
			},
		}
		sources = append(sources, source)
	}

	// 填充inbound规则
	inbounds := make([]*api.CbRule, 0, ruleNum)
	for i := 0; i < ruleNum; i++ {
		inbound := &api.CbRule{
			Sources: sources,
		}
		inbounds = append(inbounds, inbound)
	}
	circuitBreaker.Inbounds = inbounds

	resp := server.CreateCircuitBreaker(defaultCtx, circuitBreaker)
	if !respSuccess(resp) {
		t.Fatalf("error: %+v", resp)
	}

	return resp.GetCircuitBreaker()
}

/**
 * @brief 创建熔断规则版本
 */
func createCircuitBreakerVersion(t *testing.T, circuitBreaker *api.CircuitBreaker) *api.CircuitBreaker {
	circuitBreaker.Version = utils.NewStringValue("test-version")
	resp := server.CreateCircuitBreakerVersion(defaultCtx, circuitBreaker)
	if !respSuccess(resp) {
		t.Fatalf("error: %s", resp.GetInfo().GetValue())
	}

	return resp.GetCircuitBreaker()
}

/**
 * @brief 发布熔断规则
 */
func releaseCircuitBreaker(t *testing.T, circuitBreaker *api.CircuitBreaker, service *api.Service,
	ctx context.Context) *api.ConfigRelease {
	release := &api.ConfigRelease{
		Service: &api.Service{
			Name:      service.GetName(),
			Namespace: service.GetNamespace(),
		},
		CircuitBreaker: circuitBreaker,
	}

	resp := server.ReleaseCircuitBreaker(ctx, release)
	if !respSuccess(resp) {
		t.Fatalf("error: %+v", resp)
	}

	return resp.GetConfigRelease()
}

/**
 * @brief 彻底删除熔断规则
 */
func cleanCircuitBreaker(id, version string) {
	str := `delete from circuitbreaker_rule where id = ? and version = ?`
	if _, err := db.Exec(str, id, version); err != nil {
		panic(err)
	}
}

/**
 * @brief 彻底删除熔断规则发布记录
 */
func cleanCircuitBreakerRelation(name, namespace, ruleID, ruleVersion string) {
	str := `delete from circuitbreaker_rule_relation using circuitbreaker_rule_relation, service where 
			service_id = service.id and name = ? and namespace = ? and rule_id = ? and rule_version = ?`
	if _, err := db.Exec(str, name, namespace, ruleID, ruleVersion); err != nil {
		panic(err)
	}
}
