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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
)

var (
	testNamespace = "testNamespace123qwe"
	testGroup     = "testGroup"
	testFile      = "testFile"
	operator      = "polaris"
	size          = 7
)

// TestConfigFileCRUD 测试配置文件增删改查
func TestConfigFileCRUD(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()

	t.Run("step1-query", func(t *testing.T) {
		rsp := testSuit.testService.GetConfigFileBaseInfo(testSuit.defaultCtx, testNamespace, testGroup, testFile)
		assert.Equal(t, uint32(api.NotFoundResource), rsp.Code.GetValue())
		assert.Nil(t, rsp.ConfigFile)
	})

	t.Run("step2-create", func(t *testing.T) {
		configFile := assembleConfigFile()
		rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, testNamespace, rsp.ConfigFile.Namespace.GetValue())
		assert.Equal(t, testGroup, rsp.ConfigFile.Group.GetValue())
		assert.Equal(t, testFile, rsp.ConfigFile.Name.GetValue())
		assert.Equal(t, configFile.Content.GetValue(), rsp.ConfigFile.Content.GetValue())
		assert.Equal(t, configFile.Format.GetValue(), rsp.ConfigFile.Format.GetValue())
		assert.Equal(t, operator, rsp.ConfigFile.CreateBy.GetValue())
		assert.Equal(t, operator, rsp.ConfigFile.ModifyBy.GetValue())

		// 重复创建
		rsp2 := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
		assert.Equal(t, uint32(api.ExistedResource), rsp2.Code.GetValue())

		// 创建完之后再查询
		rsp3 := testSuit.testService.GetConfigFileBaseInfo(testSuit.defaultCtx, testNamespace, testGroup, testFile)
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())
		assert.NotNil(t, rsp.ConfigFile)
		assert.Equal(t, testNamespace, rsp.ConfigFile.Namespace.GetValue())
		assert.Equal(t, testGroup, rsp.ConfigFile.Group.GetValue())
		assert.Equal(t, testFile, rsp.ConfigFile.Name.GetValue())
		assert.Equal(t, configFile.Content.GetValue(), rsp.ConfigFile.Content.GetValue())
		assert.Equal(t, configFile.Format.GetValue(), rsp.ConfigFile.Format.GetValue())
		assert.Equal(t, operator, rsp.ConfigFile.CreateBy.GetValue())
		assert.Equal(t, operator, rsp.ConfigFile.ModifyBy.GetValue())
	})

	t.Run("step3-update", func(t *testing.T) {
		testSuit.defaultCtx = context.WithValue(testSuit.defaultCtx, utils.ContextUserNameKey, "polaris")

		rsp := testSuit.testService.GetConfigFileBaseInfo(testSuit.defaultCtx, testNamespace, testGroup, testFile)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

		configFile := rsp.ConfigFile
		newContent := "k1=v1"
		modifyBy := "polaris"
		configFile.Content = utils.NewStringValue(newContent)
		configFile.ModifyBy = utils.NewStringValue(modifyBy)

		rsp2 := testSuit.testService.UpdateConfigFile(testSuit.defaultCtx, configFile)
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		assert.Equal(t, newContent, rsp2.ConfigFile.Content.GetValue())

		// 更新完之后再查询
		rsp3 := testSuit.testService.GetConfigFileBaseInfo(testSuit.defaultCtx, testNamespace, testGroup, testFile)
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())
		assert.NotNil(t, rsp.ConfigFile)
		assert.Equal(t, testNamespace, rsp.ConfigFile.Namespace.GetValue())
		assert.Equal(t, testGroup, rsp.ConfigFile.Group.GetValue())
		assert.Equal(t, testFile, rsp.ConfigFile.Name.GetValue())
		assert.Equal(t, configFile.Content.GetValue(), rsp.ConfigFile.Content.GetValue())
		assert.Equal(t, configFile.Format.GetValue(), rsp.ConfigFile.Format.GetValue())
		assert.Equal(t, operator, rsp.ConfigFile.CreateBy.GetValue())
		assert.Equal(t, modifyBy, rsp.ConfigFile.ModifyBy.GetValue())
	})

	t.Run("step4-delete", func(t *testing.T) {
		// 删除前先发布一次
		configFile := assembleConfigFile()
		configFileRelease := assembleConfigFileRelease(configFile)
		rsp := testSuit.testService.PublishConfigFile(testSuit.defaultCtx, configFileRelease)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

		deleteBy := "polaris"
		rsp2 := testSuit.testService.DeleteConfigFile(testSuit.defaultCtx, testNamespace, testGroup, testFile, deleteBy)
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())

		// 删除后，查询不到
		rsp3 := testSuit.testService.GetConfigFileBaseInfo(testSuit.defaultCtx, testNamespace, testGroup, testFile)
		assert.Equal(t, uint32(api.NotFoundResource), rsp3.Code.GetValue())
		assert.Nil(t, rsp2.ConfigFile)

		// 删除会创建一条删除的历史记录
		rsp4 := testSuit.testService.GetConfigFileReleaseHistory(testSuit.defaultCtx, testNamespace, testGroup, testFile, 0, 2, 0)
		assert.Equal(t, api.ExecuteSuccess, rsp4.Code.GetValue())
		assert.Equal(t, uint32(2), rsp4.Total.GetValue())
		assert.Equal(t, utils.ReleaseTypeDelete, rsp4.ConfigFileReleaseHistories[0].Type.GetValue())
		assert.Equal(t, utils.ReleaseStatusSuccess, rsp4.ConfigFileReleaseHistories[0].Status.GetValue())
		assert.Equal(t, deleteBy, rsp4.ConfigFileReleaseHistories[0].CreateBy.GetValue())
		assert.Equal(t, deleteBy, rsp4.ConfigFileReleaseHistories[0].ModifyBy.GetValue())
		assert.Equal(t, "", rsp4.ConfigFileReleaseHistories[0].Content.GetValue())
	})

	t.Run("step5-search-by-group", func(t *testing.T) {
		group := "group11"
		for i := 0; i < size; i++ {
			rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, assembleConfigFileWithFixedGroupAndRandomFileName(group))
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		}

		// 第一页
		rsp2 := testSuit.testService.SearchConfigFile(testSuit.defaultCtx, testNamespace, group, "", "", 0, 3)
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		assert.Equal(t, uint32(size), rsp2.Total.GetValue())
		assert.Equal(t, 3, len(rsp2.ConfigFiles))

		// 最后一页
		rsp3 := testSuit.testService.SearchConfigFile(testSuit.defaultCtx, testNamespace, group, "", "", 6, 3)
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())
		assert.Equal(t, uint32(size), rsp3.Total.GetValue())
		assert.Equal(t, 1, len(rsp3.ConfigFiles))

		// group为空
		rsp4 := testSuit.testService.SearchConfigFile(testSuit.defaultCtx, testNamespace, "", "", "", 0, 3)
		assert.Equal(t, api.ExecuteSuccess, rsp4.Code.GetValue())
		assert.Equal(t, uint32(size), rsp4.Total.GetValue())
		assert.Equal(t, 3, len(rsp4.ConfigFiles))

		// group 模糊搜索
		rsp5 := testSuit.testService.SearchConfigFile(testSuit.defaultCtx, testNamespace, "group1", "", "", 0, 3)
		assert.Equal(t, api.ExecuteSuccess, rsp5.Code.GetValue())
		assert.Equal(t, uint32(size), rsp5.Total.GetValue())
		assert.Equal(t, 3, len(rsp5.ConfigFiles))
	})

	t.Run("step6-search-by-file", func(t *testing.T) {
		file := "file1.txt"
		for i := 0; i < size; i++ {
			rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, assembleConfigFileWithRandomGroupAndFixedFileName(file))
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		}

		// 第一页
		rsp2 := testSuit.testService.SearchConfigFile(testSuit.defaultCtx, testNamespace, "", file, "", 0, 3)
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		assert.Equal(t, uint32(size), rsp2.Total.GetValue())
		assert.Equal(t, 3, len(rsp2.ConfigFiles))

		// 最后一页
		rsp3 := testSuit.testService.SearchConfigFile(testSuit.defaultCtx, testNamespace, "", file, "", 6, 3)
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())
		assert.Equal(t, uint32(size), rsp3.Total.GetValue())
		assert.Equal(t, 1, len(rsp3.ConfigFiles))

		// group,name都为空
		rsp4 := testSuit.testService.SearchConfigFile(testSuit.defaultCtx, testNamespace, "", "", "", 0, 3)
		assert.Equal(t, api.ExecuteSuccess, rsp4.Code.GetValue())
		assert.Equal(t, uint32(size*2), rsp4.Total.GetValue()) // 总数为随机 group 和随机 fileName 总和
		assert.Equal(t, 3, len(rsp4.ConfigFiles))

		// fileName 模糊搜索
		rsp5 := testSuit.testService.SearchConfigFile(testSuit.defaultCtx, testNamespace, "", "file1", "", 0, 3)
		assert.Equal(t, api.ExecuteSuccess, rsp5.Code.GetValue())
		assert.Equal(t, uint32(size), rsp5.Total.GetValue())
		assert.Equal(t, 3, len(rsp5.ConfigFiles))
	})

	t.Run("step7-search-by-tag", func(t *testing.T) {
		// 按 tag k1=v1 搜索
		rsp := testSuit.testService.SearchConfigFile(testSuit.defaultCtx, testNamespace, "", "", "k1,v1", 0, 3)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, uint32(size*2), rsp.Total.GetValue())
		assert.Equal(t, 3, len(rsp.ConfigFiles))
	})

}

// TestPublishConfigFile 测试配置文件发布相关的用例
func TestPublishConfigFile(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()

	configFile := assembleConfigFile()
	rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

	configFileRelease := assembleConfigFileRelease(configFile)
	rsp2 := testSuit.testService.PublishConfigFile(testSuit.defaultCtx, configFileRelease)
	assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
	assert.Equal(t, uint64(1), rsp2.ConfigFileRelease.Version.GetValue())
	assert.Equal(t, configFileRelease.Name.GetValue(), rsp2.ConfigFileRelease.Name.GetValue())
	assert.Equal(t, configFileRelease.CreateBy.GetValue(), rsp2.ConfigFileRelease.CreateBy.GetValue())
	assert.Equal(t, configFileRelease.CreateBy.GetValue(), rsp2.ConfigFileRelease.ModifyBy.GetValue())
	assert.Equal(t, configFile.Content.GetValue(), rsp2.ConfigFileRelease.Content.GetValue())

	rsp3 := testSuit.testService.GetConfigFileRelease(testSuit.defaultCtx, testNamespace, testGroup, testFile)
	assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())
	assert.Equal(t, uint64(1), rsp3.ConfigFileRelease.Version.GetValue())
	assert.Equal(t, configFileRelease.Name.GetValue(), rsp3.ConfigFileRelease.Name.GetValue())
	assert.Equal(t, configFileRelease.CreateBy.GetValue(), rsp3.ConfigFileRelease.CreateBy.GetValue())
	assert.Equal(t, configFileRelease.CreateBy.GetValue(), rsp3.ConfigFileRelease.ModifyBy.GetValue())
	assert.Equal(t, configFile.Content.GetValue(), rsp3.ConfigFileRelease.Content.GetValue())

	rsp4 := testSuit.testService.GetConfigFileLatestReleaseHistory(testSuit.defaultCtx, testNamespace, testGroup, testFile)
	assert.Equal(t, api.ExecuteSuccess, rsp4.Code.GetValue())
	assert.Equal(t, configFileRelease.Name.GetValue(), rsp4.ConfigFileReleaseHistory.Name.GetValue())
	assert.Equal(t, configFileRelease.CreateBy.GetValue(), rsp4.ConfigFileReleaseHistory.CreateBy.GetValue())
	assert.Equal(t, configFileRelease.CreateBy.GetValue(), rsp4.ConfigFileReleaseHistory.ModifyBy.GetValue())
	assert.Equal(t, configFile.Content.GetValue(), rsp4.ConfigFileReleaseHistory.Content.GetValue())
	assert.Equal(t, configFile.Format.GetValue(), rsp4.ConfigFileReleaseHistory.Format.GetValue())
	assert.Equal(t, 3, len(rsp4.ConfigFileReleaseHistory.Tags))
	assert.Equal(t, utils.ReleaseTypeNormal, rsp4.ConfigFileReleaseHistory.Type.GetValue())
	assert.Equal(t, utils.ReleaseStatusSuccess, rsp4.ConfigFileReleaseHistory.Status.GetValue())

	rsp5 := testSuit.testService.GetConfigFileRichInfo(testSuit.defaultCtx, testNamespace, testGroup, testFile)
	assert.Equal(t, api.ExecuteSuccess, rsp4.Code.GetValue())
	assert.Equal(t, rsp4.ConfigFileReleaseHistory.CreateTime.GetValue(), rsp5.ConfigFile.ReleaseTime.GetValue())
	assert.Equal(t, rsp3.ConfigFileRelease.ModifyTime.GetValue(), rsp5.ConfigFile.ReleaseTime.GetValue())
	assert.Equal(t, rsp3.ConfigFileRelease.ModifyBy.GetValue(), rsp5.ConfigFile.ReleaseBy.GetValue())

	// 第二次修改发布
	secondReleaseContent := "k3=v3"
	secondReleaseFormat := utils.FileFormatHtml
	configFile.Content = utils.NewStringValue(secondReleaseContent)
	configFile.Format = utils.NewStringValue(secondReleaseFormat)

	rsp6 := testSuit.testService.UpdateConfigFile(testSuit.defaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp6.Code.GetValue())

	configFileRelease.CreateBy = utils.NewStringValue("ledou3")
	rsp7 := testSuit.testService.PublishConfigFile(testSuit.defaultCtx, configFileRelease)
	assert.Equal(t, api.ExecuteSuccess, rsp7.Code.GetValue())
	assert.Equal(t, configFile.Content.GetValue(), rsp7.ConfigFileRelease.Content.GetValue())
	assert.Equal(t, configFileRelease.CreateBy.GetValue(), rsp7.ConfigFileRelease.ModifyBy.GetValue())
	assert.Equal(t, uint64(2), rsp7.ConfigFileRelease.Version.GetValue())

	rsp8 := testSuit.testService.GetConfigFileLatestReleaseHistory(testSuit.defaultCtx, testNamespace, testGroup, testFile)
	assert.Equal(t, api.ExecuteSuccess, rsp8.Code.GetValue())
	assert.Equal(t, configFile.Content.GetValue(), rsp8.ConfigFileReleaseHistory.Content.GetValue())
	assert.Equal(t, configFile.Format.GetValue(), rsp8.ConfigFileReleaseHistory.Format.GetValue())
	assert.Equal(t, configFileRelease.CreateBy.GetValue(), rsp8.ConfigFileReleaseHistory.ModifyBy.GetValue())
	assert.Equal(t, configFileRelease.CreateBy.GetValue(), rsp8.ConfigFileReleaseHistory.CreateBy.GetValue())

	rsp9 := testSuit.testService.GetConfigFileReleaseHistory(testSuit.defaultCtx, testNamespace, testGroup, testFile, 0, 10, 0)
	assert.Equal(t, api.ExecuteSuccess, rsp9.Code.GetValue())
	assert.Equal(t, 2, len(rsp9.ConfigFileReleaseHistories))

}
