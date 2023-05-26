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
	"os"
	"testing"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/test/integrate/http"
	"github.com/polarismesh/polaris/test/integrate/resource"
)

func TestConfigCenter_ConfigFile(t *testing.T) {
	client := http.NewClient(httpserverAddress, httpserverVersion)

	ns := resource.CreateNamespaces()

	groups := resource.MockConfigGroups(ns[0])

	files := resource.MockConfigFiles(groups[0])

	defer func() {
		for i := range groups {
			if _, err := client.DeleteConfigGroup(groups[i]); err != nil {
				t.Log(err)
			}
		}
		client.DeleteNamespaces(ns)
		os.Remove("export.zip")
	}()

	t.Run("配置中心-创建配置文件", func(t *testing.T) {
		for _, file := range files {
			resp, err := client.CreateConfigFile(file)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, resp.GetCode().GetValue(), api.ExecuteSuccess, resp.GetInfo().GetValue())
		}
	})

	t.Run("配置中心-更新配置文件", func(t *testing.T) {
		for _, file := range files {
			file.Content = &wrapperspb.StringValue{
				Value: `name: polarismesh_test`,
			}
			resp, err := client.UpdateConfigFile(file)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, resp.GetCode().GetValue(), api.ExecuteSuccess, resp.GetInfo().GetValue())
		}
	})

	t.Run("配置中心-导出配置文件", func(t *testing.T) {
		req := &apiconfig.ConfigFileExportRequest{
			Namespace: ns[0].Name,
			Groups: []*wrapperspb.StringValue{
				groups[0].Name,
			},
		}
		err := client.ExportConfigFile(req)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("配置中心-导入配置文件", func(t *testing.T) {
		namespace := ns[1].Name.Value
		conflictHandling := utils.ConfigFileImportConflictSkip
		resp, err := client.ImportConfigFile(namespace, "", conflictHandling)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, resp.GetCode().GetValue(), api.ExecuteSuccess, resp.GetInfo().GetValue())
	})

	t.Run("配置中心-删除配置文件", func(t *testing.T) {
		for _, file := range files {
			resp, err := client.DeleteConfigFile(file)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, resp.GetCode().GetValue(), api.ExecuteSuccess, resp.GetInfo().GetValue())
		}
	})

	t.Run("配置中心-获取全部加密算法", func(t *testing.T) {
		resp, err := client.GetAllConfigEncryptAlgorithms()
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, resp.GetCode().GetValue(), api.ExecuteSuccess, resp.GetInfo().GetValue())
	})
}
