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
	"time"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	v1 "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/test/integrate/http"
	"github.com/polarismesh/polaris/test/integrate/resource"
)

/**
 * @brief 测试增删改查命名空间
 */
func TestNamespace(t *testing.T) {
	t.Log("test namespace interface")

	client := http.NewClient(httpserverAddress, httpserverVersion)

	namespaces := resource.CreateNamespaces()

	// 创建命名空间
	ret, err := client.CreateNamespaces(namespaces)
	if err != nil {
		t.Fatalf("create namespaces fail: %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		namespaces[index].Token = item.GetNamespace().GetToken()
	}
	t.Log("create namespaces success")

	// 查询命名空间
	_, err = client.GetNamespaces(namespaces)
	if err != nil {
		t.Fatalf("get namespaces fail: %s", err.Error())
	}
	t.Log("get namespaces success")

	// 更新命名空间
	resource.UpdateNamespaces(namespaces)

	err = client.UpdateNamesapces(namespaces)
	if err != nil {
		t.Fatalf("update namespaces fail: %s", err.Error())
	}
	t.Log("update namespaces success")

	// 查询命名空间
	_, err = client.GetNamespaces(namespaces)
	if err != nil {
		t.Fatalf("get namespaces fail: %s", err.Error())
	}
	t.Log("get namespaces success")

	// 删除命名空间
	err = client.DeleteNamespaces(namespaces)
	if err != nil {
		t.Fatalf("delete namespaces fail: %s", err.Error())
	}
	t.Log("delete namespaces success")
}

// TestCountNamespaceService 统计命名空间下的服务数以及实例数
func TestCountNamespaceService(t *testing.T) {
	t.Log("test namepsace interface")
	client := http.NewClient(httpserverAddress, httpserverVersion)

	namespaces := resource.CreateNamespaces()

	// 创建命名空间
	ret, err := client.CreateNamespaces(namespaces)
	if err != nil {
		t.Fatalf("create namespaces fail: %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		namespaces[index].Token = item.GetNamespace().GetToken()
	}
	t.Log("create namepsaces success")

	expectRes := make(map[string]model.NamespaceServiceCount)

	for _, namespace := range ret.Responses {
		createServiceAndInstance(t, &expectRes, client, namespace.Namespace)
	}

	//
	time.Sleep(time.Duration(5) * time.Second)

	// 获取namespace info 列表

	resp, err := client.GetNamespaces(namespaces)

	for _, namespace := range resp {
		expectVal := expectRes[namespace.GetName().GetValue()]

		if expectVal.ServiceCount == namespace.TotalServiceCount.GetValue() && expectVal.InstanceCnt.TotalInstanceCount == namespace.TotalInstanceCount.Value {
			continue
		} else {
			t.Fatalf("namespace %s cnt info not expect", namespace.Name.GetValue())
		}
	}

	t.Logf("TestNamespaceServiceCnt success")

	// 开始清理所有的数据
}

// TestDeleteNamespaceWhenHaveService 当命名空间下还有服务时进行删除命名空间的操作
func TestDeleteNamespaceWhenHaveService(t *testing.T) {
	t.Log("test namepsace delete when has service")
	client := http.NewClient(httpserverAddress, httpserverVersion)

	namespaces := resource.CreateNamespaces()

	// 创建命名空间
	_, err := client.CreateNamespaces(namespaces)
	if err != nil {
		t.Fatalf("create namespaces fail: %s", err.Error())
	}
	t.Log("create namepsaces success")

	services := resource.CreateServices(namespaces[0])
	if _, err = client.CreateServices(services); err != nil {
		t.Fatal(err)
	}

	resp, err := client.DeleteNamespacesGetResp(namespaces)
	if resp != nil {
		if resp.GetCode().GetValue() != v1.NamespaceExistedServices {
			t.Fatalf("delete namespace need return code:NamespaceExistedServices, actual : %d, %s", int(resp.GetCode().GetValue()), resp.GetInfo().GetValue())
		}
	}
	if err != nil && resp == nil {
		t.Fatalf("delete namespaces fail: %s", err.Error())
	}

	// 删除 service
	if err := client.DeleteServices(services); err != nil {
		t.Fatalf("delete service fail: %+v", err)
	}

	resp, err = client.DeleteNamespacesGetResp(namespaces)
	if resp != nil {
		if resp.GetCode().GetValue() != v1.ExecuteSuccess {
			t.Fatalf("delete namespace need return code:ExecuteSuccess, actual : %d, %s", int(resp.GetCode().GetValue()), resp.GetInfo().GetValue())
		}
	}
	if err != nil && resp == nil {
		t.Fatalf("delete namespaces fail: %s", err.Error())
	}
}

// TestDeleteNamespaceWhenHaveConfigGroup 当命名空间下还有配置时进行删除命名空间的操作
func TestDeleteNamespaceWhenHaveConfigGroup(t *testing.T) {
	t.Log("test namepsace interface")
	client := http.NewClient(httpserverAddress, httpserverVersion)

	namespaces := resource.CreateNamespaces()

	// 创建命名空间
	_, err := client.CreateNamespaces(namespaces)
	if err != nil {
		t.Fatalf("create namespaces fail: %s", err.Error())
	}
	t.Log("create namepsaces success")

	groups := resource.MockConfigGroups(namespaces[0])
	if _, err = client.CreateConfigGroup(groups[0]); err != nil {
		t.Fatal(err)
	}

	resp, err := client.DeleteNamespacesGetResp(namespaces)
	if resp != nil {
		if resp.GetCode().GetValue() != v1.NamespaceExistedConfigGroups {
			t.Fatalf("delete namespace need return code:NamespaceExistedConfigGroups, actual : %d, %s", int(resp.GetCode().GetValue()), resp.GetInfo().GetValue())
		}
	}
	if err != nil && resp == nil {
		t.Fatalf("delete namespaces fail: %s", err.Error())
	}

	// 删除配置分组
	if _, err := client.DeleteConfigGroup(groups[0]); err != nil {
		t.Fatalf("delete service fail: %+v", err)
	}

	resp, err = client.DeleteNamespacesGetResp(namespaces)
	if resp != nil {
		if resp.GetCode().GetValue() != v1.ExecuteSuccess {
			t.Fatalf("delete namespace need return code:ExecuteSuccess, actual : %d, %s", int(resp.GetCode().GetValue()), resp.GetInfo().GetValue())
		}
	}
	if err != nil && resp == nil {
		t.Fatalf("delete namespaces fail: %s", err.Error())
	}
}

func createServiceAndInstance(t *testing.T, expectRes *map[string]model.NamespaceServiceCount, client *http.Client, namespace *apimodel.Namespace) ([]*apiservice.Service, []*apiservice.Instance) {
	services := resource.CreateServices(namespace)

	_, err := client.CreateServices(services)

	if err != nil {
		t.Fatal(err)
	}

	cntVal := &model.NamespaceServiceCount{
		ServiceCount: uint32(len(services)),
		InstanceCnt:  &model.InstanceCount{},
	}

	finalInstances := make([]*apiservice.Instance, 0)

	for _, service := range services {
		instances := resource.CreateInstances(service)
		if _, err := client.CreateInstances(instances); err != nil {
			t.Fatal(err)
		}

		finalInstances = append(finalInstances, instances...)
		cntVal.InstanceCnt.TotalInstanceCount += uint32(len(instances))
	}

	(*expectRes)[namespace.GetName().GetValue()] = *cntVal

	return services, finalInstances
}
