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

	"github.com/polarismesh/polaris-server/test/http"
	"github.com/polarismesh/polaris-server/test/resource"
)

/**
 * @brief 测试增删改查命名空间
 */
func TestNamespace(t *testing.T) {
	t.Log("test namepsace interface")

	client := http.NewClient(httpserverAddress, httpserverVersion)

	namespaces := resource.CreateNamespaces()

	// 创建命名空间
	ret, err := client.CreateNamespaces(namespaces)
	if err != nil {
		t.Fatalf("create namespaces fail")
	}
	for index, item := range ret.GetResponses() {
		namespaces[index].Token = item.GetNamespace().GetToken()
	}
	t.Log("create namepsaces success")

	// 查询命名空间
	err = client.GetNamespaces(namespaces)
	if err != nil {
		t.Fatalf("get namespaces fail")
	}
	t.Log("get namespaces success")

	// 更新命名空间
	resource.UpdateNamespaces(namespaces)

	err = client.UpdateNamesapces(namespaces)
	if err != nil {
		t.Fatalf("update namespaces fail")
	}
	t.Log("update namespaces success")

	// 查询命名空间
	err = client.GetNamespaces(namespaces)
	if err != nil {
		t.Fatalf("get namespaces fail")
	}
	t.Log("get namespaces success")

	// 删除命名空间
	err = client.DeleteNamespaces(namespaces)
	if err != nil {
		t.Fatalf("delete namespaces fail")
	}
	t.Log("delete namepsaces success")
}
