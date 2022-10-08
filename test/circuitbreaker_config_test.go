//go:build integration
// +build integration

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
	"testing"

	"github.com/polarismesh/polaris/test/http"
	"github.com/polarismesh/polaris/test/resource"
)

/**
 * @brief 测试熔断规则
 */
func TestCircuitBreaker(t *testing.T) {
	t.Log("test circuit breaker interface")

	client := http.NewClient(httpserverAddress, httpserverVersion)

	// -------------------------------------------------------
	namespaces := resource.CreateNamespaces()
	// 创建命名空间
	ret, err := client.CreateNamespaces(namespaces)
	if err != nil {
		t.Fatalf("create namespaces fail，error: %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		namespaces[index].Token = item.GetNamespace().GetToken()
	}
	t.Log("create namespaces success")

	circuitBreakers := resource.CreateCircuitBreakers(namespaces[0])

	// 创建熔断规则
	ret, err = client.CreateCircuitBreakers(circuitBreakers)
	if err != nil {
		t.Fatalf("create circuit breakers fail, err: %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		circuitBreakers[index].Id = item.GetCircuitBreaker().GetId()
		circuitBreakers[index].Version = item.GetCircuitBreaker().GetVersion()
		circuitBreakers[index].Token = item.GetCircuitBreaker().GetToken()
	}
	t.Log("create rate limits success")

	// 查询熔断规则:根据id和version
	if err = client.GetCircuitBreaker(circuitBreakers[0], circuitBreakers[0]); err != nil {
		t.Fatalf("get circuit breaker fail, err : %s", err.Error())
	}
	t.Log("get circuit breaker success")

	// 更新熔断规则
	resource.UpdateCircuitBreakers(circuitBreakers)

	if err := client.UpdateCircuitBreakers(circuitBreakers); err != nil {
		t.Fatalf("update circuit breakers fail, err: %s", err.Error())
	}
	t.Log("update circuit breaker success")

	// 查询熔断规则：根据id和version
	if err = client.GetCircuitBreaker(circuitBreakers[0], circuitBreakers[0]); err != nil {
		t.Fatalf("get circuit breaker fail, err : %s", err.Error())
	}
	t.Log("get circuit breaker success")

	// 创建熔断规则版本
	newCircuitBreakers := resource.CreateCircuitBreakerVersions(circuitBreakers)

	ret, err = client.CreateCircuitBreakerVersions(newCircuitBreakers)
	if err != nil {
		t.Fatalf("create circuit breaker versions fail, err: %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		newCircuitBreakers[index].Id = item.GetCircuitBreaker().GetId()
	}
	t.Log("create circuit breaker versions success")

	// 查询熔断规则：根据id和version
	if err = client.GetCircuitBreaker(circuitBreakers[0], newCircuitBreakers[0]); err != nil {
		t.Fatalf("get circuit breaker fail, err: %s", err.Error())
	}
	t.Log("get circuit breaker success")

	// 查询熔断规则的所有版本
	if err := client.GetCircuitBreakerVersions(newCircuitBreakers[0]); err != nil {
		t.Fatalf("get circuit breaker version fail, err: %s", err.Error())
	}

	// -------------------------------------------------------
	services := resource.CreateServices(namespaces[0])
	// 创建服务
	ret, err = client.CreateServices(services)
	if err != nil {
		t.Fatalf("create services fail，error: %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		services[index].Token = item.GetService().GetToken()
	}
	t.Log("create services success")
	// -------------------------------------------------------

	// 发布熔断规则
	configReleases := resource.CreateConfigRelease(services, newCircuitBreakers)
	if err := client.ReleaseCircuitBreakers(configReleases); err != nil {
		t.Fatalf("release config release fail，error: %s", err.Error())
	}
	t.Log("release circuit breakers success")

	// 查询熔断规则的已发布版本及绑定服务
	if err := client.GetCircuitBreakersRelease(newCircuitBreakers[0], services[0]); err != nil {
		t.Fatalf("get circuit breakers tag err: %s", err.Error())
	}
	t.Log("get circuit breakers tag success")

	// 查询服务绑定的熔断规则
	if err := client.GetCircuitBreakerByService(services[0], circuitBreakers[0], newCircuitBreakers[0]); err != nil {
		t.Fatalf("get circuit breaker by service err: %s", err.Error())
	}
	t.Log("get circuit breaker by service success")

	// 解绑熔断规则
	if err := client.UnbindCircuitBreakers(configReleases); err != nil {
		t.Fatalf("unbind config release fail，error: %s", err.Error())
	}
	t.Log("unbind circuit breakers success")

	// 删除熔断规则
	circuitBreakers = append(circuitBreakers, newCircuitBreakers...)
	err = client.DeleteCircuitBreakers(circuitBreakers)
	if err != nil {
		t.Fatalf("delete circuitbreaker fail，error: %s", err.Error())
	}
	t.Log("delete circuitbreaker success")

	// -------------------------------------------------------

	// 删除服务
	err = client.DeleteServices(services)
	if err != nil {
		t.Fatalf("delete services fail，error: %s", err.Error())
	}
	t.Log("delete services success")

	// 删除命名空间
	err = client.DeleteNamespaces(namespaces)
	if err != nil {
		t.Fatalf("delete namespaces fail，error: %s", err.Error())
	}
	t.Log("delete namespaces success")
}
