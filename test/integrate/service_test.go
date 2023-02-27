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

// TestService 测试增删改查服务
func TestService(t *testing.T) {
	t.Log("test service interface")

	client := http.NewClient(httpserverAddress, httpserverVersion)

	namespaces := resource.CreateNamespaces()

	// 创建命名空间
	ret, err := client.CreateNamespaces(namespaces)
	if err != nil {
		t.Fatalf("create namespaces fail, err is %v", err)
	}
	for index, item := range ret.GetResponses() {
		namespaces[index].Token = item.GetNamespace().GetToken()
	}
	t.Log("create namespaces success")

	// -------------------------------------------------------

	services := resource.CreateServices(namespaces[0])

	// 创建服务
	ret, err = client.CreateServices(services)
	if err != nil {
		t.Fatalf("create services fail, err is %v", err)
	}
	for index, item := range ret.GetResponses() {
		services[index].Token = item.GetService().GetToken()
	}
	t.Log("create services success")

	// 查询服务
	err = client.GetServices(services)
	if err != nil {
		t.Fatalf("get services fail, err is %v", err)
	}
	t.Log("get services success")

	// 更新服务
	resource.UpdateServices(services)

	err = client.UpdateServices(services)
	if err != nil {
		t.Fatalf("update services fail, err is %v", err)
	}
	t.Log("update services success")

	// 查询服务
	err = client.GetServices(services)
	if err != nil {
		t.Fatalf("get services fail, err is %v", err)
	}
	t.Log("get services success")

	// 删除服务
	err = client.DeleteServices(services)
	if err != nil {
		t.Fatalf("delete services fail, err is %v", err)
	}
	t.Log("delete services success")

	// -------------------------------------------------------

	// 删除命名空间
	err = client.DeleteNamespaces(namespaces)
	if err != nil {
		t.Fatalf("delete namespaces fail, err is %v", err)
	}
	t.Log("delete namespaces success")
}
