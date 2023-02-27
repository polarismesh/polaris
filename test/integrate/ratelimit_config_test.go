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

	"github.com/polarismesh/polaris/test/integrate/http"
	"github.com/polarismesh/polaris/test/integrate/resource"
)

/**
 * @brief 测试增删改查限流规则
 */

/**
 * @brief 测试增删改查限流规则
 */
func TestRateLimit(t *testing.T) {
	t.Log("test rate limit interface")

	client := http.NewClient(httpserverAddress, httpserverVersion)

	namespaces := resource.CreateNamespaces()
	services := resource.CreateServices(namespaces[0])

	// 创建命名空间
	ret, err := client.CreateNamespaces(namespaces)
	if err != nil {
		t.Fatalf("create namespaces fail, err is %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		namespaces[index].Token = item.GetNamespace().GetToken()
	}
	t.Log("create namespaces success")

	// 创建服务
	ret, err = client.CreateServices(services)
	if err != nil {
		t.Fatalf("create services fail, err is %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		services[index].Token = item.GetService().GetToken()
	}
	t.Log("create services success")

	// -------------------------------------------------------

	rateLimits := resource.CreateRateLimits(services)

	// 创建限流规则
	ret, err = client.CreateRateLimits(rateLimits)
	if err != nil {
		t.Fatalf("create rate limits fail, err is %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		rateLimits[index].Id = item.GetRateLimit().GetId()
	}
	t.Log("create rate limits success")

	// 查询限流规则
	err = client.GetRateLimits(rateLimits)
	if err != nil {
		t.Fatalf("get rate limits fail, err is %s", err.Error())
	}
	t.Log("get rate limits success")

	// 更新限流规则
	resource.UpdateRateLimits(rateLimits)

	err = client.UpdateRateLimits(rateLimits)
	if err != nil {
		t.Fatalf("update rate limits fail, err is %s", err.Error())
	}
	t.Log("update rate limits success")

	// 查询限流规则
	err = client.GetRateLimits(rateLimits)
	if err != nil {
		t.Fatalf("get rate limits fail, err is %s", err.Error())
	}
	t.Log("get rate limits success")

	// 禁用限流规则
	err = client.EnableRateLimits(rateLimits)
	if err != nil {
		t.Fatalf("enable rate limits fail, err is %s", err.Error())
	}
	t.Log("enable rate limits success")

	// 删除限流规则
	err = client.DeleteRateLimits(rateLimits)
	if err != nil {
		t.Fatalf("delete rate limits fail, err is %s", err.Error())
	}
	t.Log("delete rate limits success")

	// -------------------------------------------------------

	// 删除服务
	err = client.DeleteServices(services)
	if err != nil {
		t.Fatalf("delete services fail, err is %s", err.Error())
	}
	t.Log("delete services success")

	// 删除命名空间
	err = client.DeleteNamespaces(namespaces)
	if err != nil {
		t.Fatalf("delete namespaces fail, err is %s", err.Error())
	}
	t.Log("delete namespaces success")
}
