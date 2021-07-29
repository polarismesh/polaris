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
	"github.com/polarismesh/polaris-server/test/http"
	"github.com/polarismesh/polaris-server/test/resource"
	"testing"
)

/**
 * @brief 测试增删改查平台
 */
func TestPlatform(t *testing.T) {
	t.Log("test platform interface")

	client := http.NewClient(httpserverAddress, httpserverVersion)

	platforms := resource.CreatePlatforms()

	// 创建平台
	ret, err := client.CreatePlatforms(platforms)
	if err != nil {
		t.Fatalf("create platforms fail: %s", err.Error())
	}
	for index, item := range ret.GetResponses() {
		platforms[index].Token = item.GetPlatform().GetToken()
	}
	t.Log("create platforms success")

	// 查询平台
	err = client.GetPlatforms(platforms)
	if err != nil {
		t.Fatalf("get platforms fail: %s", err.Error())
	}
	t.Log("get platforms success")

	// 更新平台
	resource.UpdatePlatforms(platforms)

	err = client.UpdatePlatforms(platforms)
	if err != nil {
		t.Fatalf("update platforms fail: %s", err.Error())
	}
	t.Log("update platforms success")

	// 查询平台
	err = client.GetPlatforms(platforms)
	if err != nil {
		t.Fatalf("get platforms fail: %s", err.Error())
	}
	t.Log("get platforms success")

	// 删除平台
	err = client.DeletePlatforms(platforms)
	if err != nil {
		t.Fatalf("delete platforms fail: %s", err.Error())
	}
	t.Log("delete platforms success")
}
