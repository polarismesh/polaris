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

	"github.com/stretchr/testify/assert"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/test/integrate/http"
	"github.com/polarismesh/polaris/test/integrate/resource"
)

func TestConfigCenter_ConfigFileGroup(t *testing.T) {

	client := http.NewClient(httpserverAddress, httpserverVersion)

	ns := resource.CreateNamespaces()
	groups := resource.MockConfigGroups(ns[0])

	defer func() {
		for i := range groups {
			if _, err := client.DeleteConfigGroup(groups[i]); err != nil {
				t.Fatal(err)
			}
		}
		client.DeleteNamespaces(ns)
	}()

	t.Run("配置中心-创建配置分组", func(t *testing.T) {

		for i := range groups {

			resp, err := client.CreateConfigGroup(groups[i])

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, resp.GetCode().GetValue(), api.ExecuteSuccess, resp.GetInfo().GetValue())
		}
	})

	t.Run("配置中心-更新配置分组", func(t *testing.T) {

		group := groups[0]

		newComment := utils.NewStringValue("update config_file_group " + utils.NewUUID())
		group.Comment = newComment

		resp, err := client.UpdateConfigGroup(group)

		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, resp.GetCode().GetValue(), api.ExecuteSuccess, resp.GetInfo().GetValue())

		queryResp, err := client.QueryConfigGroup(group, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, resp.GetCode().GetValue(), api.ExecuteSuccess, resp.GetInfo().GetValue())
		assert.Equal(t, 1, len(queryResp.ConfigFileGroups), resp.GetInfo().GetValue())

		queryGroup := queryResp.ConfigFileGroups[0]
		assert.NotNil(t, queryGroup, "query group nil")
		assert.Equal(t, group.Comment, queryGroup.Comment, "group comment is not equal")

	})

	t.Run("配置中心-删除配置分组", func(t *testing.T) {
	})
}
