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

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/test/integrate/http"
	"github.com/polarismesh/polaris/test/integrate/resource"
)

const (
	httpserverVersion = "v1"
	httpserverAddress = "127.0.0.1:8090"
	grpcServerAddress = "127.0.0.1"
)

type (
	DiscoveryRunner func(t *testing.T, clientHttp *http.Client, namespaces []*apimodel.Namespace, services []*apiservice.Service)

	ConfigRunner func(t *testing.T, clientHttp *http.Client, namespaces []*apimodel.Namespace, configGroups []*apiconfig.ConfigFileGroup)
)

func DiscoveryRunAndInitResource(t *testing.T, runner DiscoveryRunner) {
	clientHTTP := http.NewClient(httpserverAddress, httpserverVersion)

	namespaces := resource.CreateNamespaces()
	services := resource.CreateServices(namespaces[0])

	// 创建命名空间
	ret, err := clientHTTP.CreateNamespaces(namespaces)
	if err != nil {
		t.Fatalf("create namespaces fail")
	}
	for index, item := range ret.GetResponses() {
		namespaces[index].Token = item.GetNamespace().GetToken()
	}
	t.Log("create namespaces success")

	// 创建服务
	ret, err = clientHTTP.CreateServices(services)
	if err != nil {
		t.Fatalf("create services fail")
	}
	for index, item := range ret.GetResponses() {
		services[index].Token = item.GetService().GetToken()
	}
	t.Log("create services success")

	defer func() {
		// 删除服务
		err = clientHTTP.DeleteServices(services)
		if err != nil {
			t.Fatalf("delete services fail")
		}
		t.Log("delete services success")

		// 删除命名空间
		err = clientHTTP.DeleteNamespaces(namespaces)
		if err != nil {
			t.Fatalf("delete namespaces fail")
		}
		t.Log("delete namespaces success")
	}()

	runner(t, clientHTTP, namespaces, services)
}

func ConfigCenterRunAndInitResource(t *testing.T, runner ConfigRunner) {
	clientHTTP := http.NewClient(httpserverAddress, httpserverVersion)

	namespaces := resource.CreateNamespaces()
	groups := resource.MockConfigGroups(namespaces[0])

	// 创建命名空间
	ret, err := clientHTTP.CreateNamespaces(namespaces)
	if err != nil {
		t.Fatalf("create namespaces fail")
	}
	for index, item := range ret.GetResponses() {
		namespaces[index].Token = item.GetNamespace().GetToken()
	}
	t.Log("create namespaces success")

	// 创建服务
	_, err = clientHTTP.CreateConfigGroup(groups[0])
	if err != nil {
		t.Fatalf("create config group fail")
	}
	t.Log("create config group success")

	defer func() {
		// 删除配置分组
		_, err := clientHTTP.DeleteConfigGroup(groups[0])
		if err != nil {
			t.Fatalf("delete config group fail")
		}
		t.Log("delete config group success")

		// 删除命名空间
		err = clientHTTP.DeleteNamespaces(namespaces)
		if err != nil {
			t.Fatalf("delete namespaces fail")
		}
		t.Log("delete namespaces success")
	}()

	runner(t, clientHTTP, namespaces, groups)
}
