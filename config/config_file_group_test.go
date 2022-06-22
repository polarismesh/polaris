/*
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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

var (
	randomGroupPrefix = "randomGroup"
	randomGroupSize   = uint32(7)
)

// TestConfigFileGroupCRUD 测试配置文件组增删改查
func TestConfigFileGroupCRUD(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()

	// 查询不存在的 group
	t.Run("step1-query-none", func(t *testing.T) {
		rsp := testSuit.testService.QueryConfigFileGroups(testSuit.defaultCtx, testNamespace, testGroup, "", 0, 1)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, 0, len(rsp.ConfigFileGroups))
	})

	// 创建 group
	t.Run("step2-create", func(t *testing.T) {
		rsp := testSuit.testService.CreateConfigFileGroup(testSuit.defaultCtx, assembleConfigFileGroup())
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, testNamespace, rsp.ConfigFileGroup.Namespace.GetValue())
		assert.Equal(t, testGroup, rsp.ConfigFileGroup.Name.GetValue())

		rsp2 := testSuit.testService.CreateConfigFileGroup(testSuit.defaultCtx, assembleConfigFileGroup())
		assert.Equal(t, uint32(api.ExistedResource), rsp2.Code.GetValue())
	})

	// 再次查询 group，能查询到上一步创建的 group
	t.Run("step3-query-existed", func(t *testing.T) {
		rsp := testSuit.testService.QueryConfigFileGroups(testSuit.defaultCtx, testNamespace, testGroup, "", 0, 1)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, 1, len(rsp.ConfigFileGroups))
	})

	// 创建配置文件
	t.Run("create-config-files", func(t *testing.T) {
		rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, assembleConfigFile())
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

		rsp2 := testSuit.testService.SearchConfigFile(testSuit.defaultCtx, testNamespace, testGroup, "", "", uint32(0), uint32(10))
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		assert.Equal(t, uint32(1), rsp2.Total.GetValue())
	})

	// 删除 group
	t.Run("step4-delete", func(t *testing.T) {
		rsp := testSuit.testService.DeleteConfigFileGroup(testSuit.defaultCtx, testNamespace, testGroup)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

		rsp2 := testSuit.testService.SearchConfigFile(testSuit.defaultCtx, testNamespace, testGroup, "", "", uint32(0), uint32(10))
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		assert.Equal(t, uint32(0), rsp2.Total.GetValue())
	})

	// 再次查询group，由于被删除，所以查不到
	t.Run("step5-query-none", func(t *testing.T) {
		rsp := testSuit.testService.QueryConfigFileGroups(testSuit.defaultCtx, testNamespace, testGroup, "", 0, 1)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, 0, len(rsp.ConfigFileGroups))
	})

	// 创建 7个随机 group 和一个固定的 group
	t.Run("step6-create", func(t *testing.T) {
		for i := 0; i < int(randomGroupSize); i++ {
			rsp := testSuit.testService.CreateConfigFileGroup(testSuit.defaultCtx, assembleRandomConfigFileGroup())
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
			assert.Equal(t, testNamespace, rsp.ConfigFileGroup.Namespace.GetValue())
		}

		rsp2 := testSuit.testService.CreateConfigFileGroup(testSuit.defaultCtx, assembleConfigFileGroup())
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		assert.Equal(t, testNamespace, rsp2.ConfigFileGroup.Namespace.GetValue())
		assert.Equal(t, testGroup, rsp2.ConfigFileGroup.Name.GetValue())
	})

	// 模糊查询
	t.Run("step7-query-random", func(t *testing.T) {
		rsp := testSuit.testService.QueryConfigFileGroups(testSuit.defaultCtx, testNamespace, randomGroupPrefix, "", 0, 2)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, 2, len(rsp.ConfigFileGroups))
		assert.Equal(t, randomGroupSize, rsp.Total.GetValue())
	})

	// 测试翻页
	t.Run("step8-query-by-page", func(t *testing.T) {
		// 最后一页
		rsp := testSuit.testService.QueryConfigFileGroups(testSuit.defaultCtx, testNamespace, randomGroupPrefix, "", 6, 2)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, 1, len(rsp.ConfigFileGroups))
		assert.Equal(t, randomGroupSize, rsp.Total.GetValue())

		// 超出页范围
		rsp2 := testSuit.testService.QueryConfigFileGroups(testSuit.defaultCtx, testNamespace, randomGroupPrefix, "", 8, 2)
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		assert.Equal(t, 0, len(rsp2.ConfigFileGroups))
		assert.Equal(t, randomGroupSize, rsp2.Total.GetValue())
	})
}
