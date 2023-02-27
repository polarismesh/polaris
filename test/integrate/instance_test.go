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

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/test/integrate/http"
	"github.com/polarismesh/polaris/test/integrate/resource"
)

/**
 * @brief 测试增删改查服务实例
 */
func TestInstance(t *testing.T) {
	t.Log("test instance interface")

	client := http.NewClient(httpserverAddress, httpserverVersion)

	namespaces := resource.CreateNamespaces()
	services := resource.CreateServices(namespaces[0])

	// 创建命名空间
	ret, err := client.CreateNamespaces(namespaces)
	if err != nil {
		t.Fatalf("create namespaces fail: %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		namespaces[index].Token = item.GetNamespace().GetToken()
	}
	t.Log("create namespaces success")

	// 创建服务
	ret, err = client.CreateServices(services)
	if err != nil {
		t.Fatalf("create services fail: %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		services[index].Token = item.GetService().GetToken()
	}
	t.Log("create services success")

	// -------------------------------------------------------

	instances := resource.CreateInstances(services[0])

	// 创建实例
	ret, err = client.CreateInstances(instances)
	if err != nil {
		t.Fatalf("create instances fail: %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		instances[index].Id = item.GetInstance().GetId()
	}
	t.Log("create instances success")

	// 查询实例
	err = client.GetInstances(instances)
	if err != nil {
		t.Fatalf("get instances fail: %s", err.Error())
	}
	t.Log("get instances success")

	// 更新实例
	resource.UpdateInstances(instances)

	err = client.UpdateInstances(instances)
	if err != nil {
		t.Fatalf("update instances fail: %s", err.Error())
	}
	t.Log("update instances success")

	// 查询实例
	err = client.GetInstances(instances)
	if err != nil {
		t.Fatalf("get instances fail: %s", err.Error())
	}
	t.Log("get instances success")

	// 删除实例
	err = client.DeleteInstances(instances)
	if err != nil {
		t.Fatalf("delete instances fail: %s", err.Error())
	}
	t.Log("delete instances success")

	// -------------------------------------------------------

	// 删除服务
	err = client.DeleteServices(services)
	if err != nil {
		t.Fatalf("delete services fail: %s", err.Error())
	}
	t.Log("delete services success")

	// 删除命名空间
	err = client.DeleteNamespaces(namespaces)
	if err != nil {
		t.Fatalf("delete namespaces fail: %s", err.Error())
	}
	t.Log("delete namespaces success")
}

// TestInstanceWithoutService 测试注册实例的时候，没有创建服务时可以自动创建服务出来
func TestInstanceWithoutService(t *testing.T) {
	t.Log("test instance interface")

	client := http.NewClient(httpserverAddress, httpserverVersion)

	namespaces := resource.CreateNamespaces()
	services := resource.CreateServices(namespaces[0])

	for i := range services {
		services[i].Name = wrapperspb.String("WithoutService_" + services[i].Name.GetValue())
	}

	// 创建命名空间
	ret, err := client.CreateNamespaces(namespaces)
	if err != nil {
		t.Fatalf("create namespaces fail: %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		namespaces[index].Token = item.GetNamespace().GetToken()
	}
	t.Log("create namespaces success")

	// -------------------------------------------------------

	instances := resource.CreateInstances(services[0])

	// 创建实例
	ret, err = client.CreateInstances(instances)
	if err != nil {
		t.Fatalf("create instances fail: %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		instances[index].Id = item.GetInstance().GetId()
	}
	t.Log("create instances success")

	// 查询实例
	err = client.GetInstances(instances)
	if err != nil {
		t.Fatalf("get instances fail: %s", err.Error())
	}
	t.Log("get instances success")

	// 更新实例
	resource.UpdateInstances(instances)

	err = client.UpdateInstances(instances)
	if err != nil {
		t.Fatalf("update instances fail: %s", err.Error())
	}
	t.Log("update instances success")

	// 查询实例
	err = client.GetInstances(instances)
	if err != nil {
		t.Fatalf("get instances fail: %s", err.Error())
	}
	t.Log("get instances success")

	// 删除实例
	err = client.DeleteInstances(instances)
	if err != nil {
		t.Fatalf("delete instances fail: %s", err.Error())
	}
	t.Log("delete instances success")

	// -------------------------------------------------------

	// 删除服务
	err = client.DeleteServices(services)
	if err != nil {
		t.Fatalf("delete services fail: %s", err.Error())
	}
	t.Log("delete services success")

	// 删除命名空间
	err = client.DeleteNamespaces(namespaces)
	if err != nil {
		t.Fatalf("delete namespaces fail: %s", err.Error())
	}
	t.Log("delete namespaces success")
}
