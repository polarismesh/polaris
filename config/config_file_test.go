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

package config

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/wrappers"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin/crypto/aes"
	storemock "github.com/polarismesh/polaris/store/mock"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var (
	testNamespace = "testNamespace123qwe"
	testGroup     = "testGroup"
	testFile      = "testFile"
	testContent   = "testContent"
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

	t.Run("step5-find-by-group", func(t *testing.T) {
		group := "group12"
		for i := 0; i < size; i++ {
			rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, assembleConfigFileWithFixedGroupAndRandomFileName(group))
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		}

		// 第一页
		rsp2 := testSuit.testService.QueryConfigFilesByGroup(testSuit.defaultCtx, testNamespace, group, 0, 3)
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		assert.Equal(t, uint32(size), rsp2.Total.GetValue())
		assert.Equal(t, 3, len(rsp2.ConfigFiles))

		// 最后一页
		rsp3 := testSuit.testService.QueryConfigFilesByGroup(testSuit.defaultCtx, testNamespace, group, 6, 3)
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())
		assert.Equal(t, uint32(size), rsp3.Total.GetValue())
		assert.Equal(t, 1, len(rsp3.ConfigFiles))

		// group为空
		rsp4 := testSuit.testService.QueryConfigFilesByGroup(testSuit.defaultCtx, testNamespace, "", 0, 3)
		assert.Equal(t, api.InvalidConfigFileGroupName, rsp4.Code.GetValue())

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
		assert.Equal(t, uint32(size*3), rsp4.Total.GetValue()) // 总数为随机 group 和随机 fileName 总和
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
		assert.Equal(t, uint32(size*3), rsp.Total.GetValue())
		assert.Equal(t, 3, len(rsp.ConfigFiles))
	})

	t.Run("step8-export", func(t *testing.T) {
		namespace := "namespace_0"
		for i := 0; i < 3; i++ {
			group := fmt.Sprintf("group_%d", i)
			for j := 0; j < 3; j++ {
				name := fmt.Sprintf("file_%d", j)
				configFile := assembleConfigFileWithNamespaceAndGroupAndName(namespace, group, name)
				rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
				assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
			}
		}
		// 导出 group
		configFileExport := &apiconfig.ConfigFileExportRequest{
			Namespace: utils.NewStringValue("namespace_0"),
			Groups: []*wrappers.StringValue{
				utils.NewStringValue("group_0"),
				utils.NewStringValue("group_1"),
			},
		}
		rsp := testSuit.testService.ExportConfigFile(testSuit.defaultCtx, configFileExport)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		// 导出 file
		configFileExport = &apiconfig.ConfigFileExportRequest{
			Namespace: utils.NewStringValue("namespace_0"),
			Groups: []*wrappers.StringValue{
				utils.NewStringValue("group_0"),
			},
			Names: []*wrappers.StringValue{
				utils.NewStringValue("file_0"),
				utils.NewStringValue("file_1"),
				utils.NewStringValue("file_2"),
			},
		}
		rsp = testSuit.testService.ExportConfigFile(testSuit.defaultCtx, configFileExport)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		// 导出参数错误：无效的命名空间
		configFileExport = &apiconfig.ConfigFileExportRequest{
			Namespace: utils.NewStringValue(""),
		}
		rsp = testSuit.testServer.ExportConfigFile(testSuit.defaultCtx, configFileExport)
		assert.Equal(t, api.InvalidNamespaceName, rsp.Code.GetValue())
		// 导出参数错误：无效的组和文件
		configFileExport = &apiconfig.ConfigFileExportRequest{
			Namespace: utils.NewStringValue("namespace_0"),
			Groups: []*wrappers.StringValue{
				utils.NewStringValue("group_0"),
				utils.NewStringValue("group_1"),
			},
			Names: []*wrappers.StringValue{
				utils.NewStringValue("file_0"),
			},
		}
		rsp = testSuit.testServer.ExportConfigFile(testSuit.defaultCtx, configFileExport)
		assert.Equal(t, api.InvalidParameter, rsp.Code.GetValue())
		// 导出配置不存在
		configFileExport = &apiconfig.ConfigFileExportRequest{
			Namespace: utils.NewStringValue("namespace_0"),
			Groups: []*wrappers.StringValue{
				utils.NewStringValue("group_10"),
			},
		}
		rsp = testSuit.testServer.ExportConfigFile(testSuit.defaultCtx, configFileExport)
		assert.Equal(t, api.NotFoundResourceConfigFile, rsp.Code.GetValue())
	})

	t.Run("step9-import", func(t *testing.T) {
		// 导入配置文件错误
		namespace := "namespace_0"
		var configFiles []*apiconfig.ConfigFile
		for i := 0; i < 2; i++ {
			group := fmt.Sprintf("group_%d", i)
			for j := 1; j < 4; j++ {
				name := ""
				configFile := assembleConfigFileWithNamespaceAndGroupAndName(namespace, group, name)
				configFiles = append(configFiles, configFile)
			}
		}
		rsp := testSuit.testService.ImportConfigFile(testSuit.defaultCtx, configFiles, utils.ConfigFileImportConflictSkip)
		assert.Equal(t, api.InvalidConfigFileName, rsp.Code.GetValue())
	})
	t.Run("step10-import-conflict-skip", func(t *testing.T) {
		namespace := "namespace_import_skip"
		group := fmt.Sprintf("group_0")
		for j := 0; j < 3; j++ {
			name := fmt.Sprintf("file_%d", j)
			configFile := assembleConfigFileWithNamespaceAndGroupAndName(namespace, group, name)
			rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		}

		var configFiles []*apiconfig.ConfigFile
		for i := 0; i < 2; i++ {
			group := fmt.Sprintf("group_%d", i)
			for j := 1; j < 4; j++ {
				name := fmt.Sprintf("file_%d", j)
				configFile := assembleConfigFileWithNamespaceAndGroupAndName(namespace, group, name)
				configFiles = append(configFiles, configFile)
			}
		}
		rsp := testSuit.testService.ImportConfigFile(testSuit.defaultCtx, configFiles, utils.ConfigFileImportConflictSkip)
		t.Log(rsp.Code.GetValue())
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, 4, len(rsp.CreateConfigFiles))
		assert.Equal(t, 2, len(rsp.SkipConfigFiles))
		assert.Equal(t, 0, len(rsp.OverwriteConfigFiles))
	})

	t.Run("step11-import-conflict-overwrite", func(t *testing.T) {
		namespace := "namespace_import_overwrite"
		group := fmt.Sprintf("group_0")
		for j := 0; j < 3; j++ {
			name := fmt.Sprintf("file_%d", j)
			configFile := assembleConfigFileWithNamespaceAndGroupAndName(namespace, group, name)
			rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		}

		var configFiles []*apiconfig.ConfigFile
		for i := 0; i < 2; i++ {
			group := fmt.Sprintf("group_%d", i)
			for j := 1; j < 4; j++ {
				name := fmt.Sprintf("file_%d", j)
				configFile := assembleConfigFileWithNamespaceAndGroupAndName(namespace, group, name)
				configFiles = append(configFiles, configFile)
			}
		}
		rsp := testSuit.testService.ImportConfigFile(testSuit.defaultCtx, configFiles, utils.ConfigFileImportConflictOverwrite)
		t.Log(rsp.Code.GetValue())
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, 4, len(rsp.CreateConfigFiles))
		assert.Equal(t, 0, len(rsp.SkipConfigFiles))
		assert.Equal(t, 2, len(rsp.OverwriteConfigFiles))
	})

	t.Run("step12-create-entrypted", func(t *testing.T) {
		configFile := assembleConfigFile()
		configFile.Encrypted = utils.NewBoolValue(true)
		configFile.EncryptAlgo = utils.NewStringValue("AES")
		rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, testNamespace, rsp.ConfigFile.Namespace.GetValue())
		assert.Equal(t, testGroup, rsp.ConfigFile.Group.GetValue())
		assert.Equal(t, testFile, rsp.ConfigFile.Name.GetValue())
		assert.NotEqual(t, configFile.Content.GetValue(), rsp.ConfigFile.Content.GetValue())
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
		assert.NotEqual(t, configFile.Content.GetValue(), rsp.ConfigFile.Content.GetValue())
		assert.Equal(t, configFile.Format.GetValue(), rsp.ConfigFile.Format.GetValue())
		assert.Equal(t, operator, rsp.ConfigFile.CreateBy.GetValue())
		assert.Equal(t, operator, rsp.ConfigFile.ModifyBy.GetValue())
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

func TestServer_encryptConfigFile(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()
	type args struct {
		ctx        context.Context
		algorithm  string
		configFile *apiconfig.ConfigFile
		dataKey    string
	}
	dataKey, _ := hex.DecodeString("777b162a185673cb1b72b467a78221cd")
	fmt.Println(base64.StdEncoding.EncodeToString(dataKey))

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			name: "encrypt config file",
			args: args{
				ctx:       context.Background(),
				algorithm: "AES",
				configFile: &apiconfig.ConfigFile{
					Content: utils.NewStringValue("polaris"),
				},
				dataKey: "",
			},
			wantErr: nil,
		},
		{
			name: "encrypt config file with dataKey",
			args: args{
				ctx:       context.Background(),
				algorithm: "AES",
				configFile: &apiconfig.ConfigFile{
					Content: utils.NewStringValue("polaris"),
				},
				dataKey: base64.StdEncoding.EncodeToString(dataKey),
			},
			want:    "YnLZ0SYuujFBHjYHAZVN5A==",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testSuit.testServer
			err := s.encryptConfigFile(tt.args.ctx, tt.args.configFile, tt.args.algorithm, tt.args.dataKey)
			assert.Equal(t, tt.wantErr, err)
			if tt.want != "" {
				assert.Equal(t, tt.want, tt.args.configFile.Content.GetValue())
			}
			hasDataKeyTag := false
			hasAlgoTag := false
			for _, tag := range tt.args.configFile.Tags {
				if tag.Key.GetValue() == utils.ConfigFileTagKeyDataKey {
					hasDataKeyTag = true
					if tt.args.dataKey != "" {
						assert.Equal(t, tt.args.dataKey, tag.Value.GetValue())
					}
				}
				if tag.Key.GetValue() == utils.ConfigFileTagKeyEncryptAlgo {
					hasAlgoTag = true
					assert.Equal(t, tt.args.algorithm, tag.Value.GetValue())
				}
			}
			assert.True(t, hasDataKeyTag)
			assert.True(t, hasAlgoTag)
		})
	}
}

func TestServer_decryptConfigFile(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()
	type args struct {
		ctx        context.Context
		configFile *apiconfig.ConfigFile
	}

	dataKey, _ := hex.DecodeString("777b162a185673cb1b72b467a78221cd")

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			name: "decrypt config file",
			args: args{
				ctx: context.WithValue(context.Background(), utils.ContextUserNameKey, "polaris"),
				configFile: &apiconfig.ConfigFile{
					Content: utils.NewStringValue("YnLZ0SYuujFBHjYHAZVN5A=="),
					Tags: []*apiconfig.ConfigFileTag{
						{
							Key:   utils.NewStringValue(utils.ConfigFileTagKeyDataKey),
							Value: utils.NewStringValue(base64.StdEncoding.EncodeToString(dataKey)),
						},
						{
							Key:   utils.NewStringValue(utils.ConfigFileTagKeyEncryptAlgo),
							Value: utils.NewStringValue("AES"),
						},
					},
					CreateBy: utils.NewStringValue("polaris"),
				},
			},
			want:    "polaris",
			wantErr: nil,
		},
		{
			name: "non creator don't decrypt config file",
			args: args{
				ctx: context.WithValue(context.Background(), utils.ContextUserNameKey, "test"),
				configFile: &apiconfig.ConfigFile{
					Content: utils.NewStringValue("YnLZ0SYuujFBHjYHAZVN5A=="),
					Tags: []*apiconfig.ConfigFileTag{
						{
							Key:   utils.NewStringValue(utils.ConfigFileTagKeyDataKey),
							Value: utils.NewStringValue(base64.StdEncoding.EncodeToString(dataKey)),
						},
						{
							Key:   utils.NewStringValue(utils.ConfigFileTagKeyEncryptAlgo),
							Value: utils.NewStringValue("AES"),
						},
					},
					CreateBy: utils.NewStringValue("polaris"),
				},
			},
			want:    "YnLZ0SYuujFBHjYHAZVN5A==",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testSuit.testServer
			err := s.decryptConfigFile(tt.args.ctx, tt.args.configFile)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, tt.args.configFile.Content.GetValue())
			for _, tag := range tt.args.configFile.Tags {
				if tag.Key.GetValue() == utils.ConfigFileTagKeyDataKey {
					t.Fatal("config tags has data key")
				}
			}
		})
	}
}

func TestServer_GetConfigEncryptAlgorithm(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()
	tests := []struct {
		name string
		want []*wrapperspb.StringValue
	}{
		{
			name: "get config encrypt algorithm",
			want: []*wrapperspb.StringValue{
				utils.NewStringValue("AES"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsp := testSuit.testService.GetAllConfigEncryptAlgorithms(testSuit.defaultCtx)
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
			assert.Equal(t, tt.want, rsp.GetAlgorithms())
		})
	}
}

func TestServer_CreateConfigFile(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Convey("创建配置文件", t, func() {
		Convey("加密配置文件-返回error\n", func() {
			crypto := &aes.AESCrypto{}
			encryptFunc := ApplyMethod(reflect.TypeOf(crypto), "Encrypt", func(_ *aes.AESCrypto, plaintext string, key []byte) (string, error) {
				return "", errors.New("mock encrypt error")
			})
			defer encryptFunc.Reset()

			configFile := assembleEncryptConfigFile()
			got := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
			So(apimodel.Code_EncryptConfigFileException, ShouldEqual, apimodel.Code(got.GetCode().GetValue()))
		})
		Convey("解密配置文件-返回error", func() {
			crypto := &aes.AESCrypto{}
			encryptFunc := ApplyMethod(reflect.TypeOf(crypto), "Decrypt", func(_ *aes.AESCrypto, plaintext string, key []byte) (string, error) {
				return "", errors.New("mock encrypt error")
			})
			defer encryptFunc.Reset()

			configFile := assembleEncryptConfigFile()
			testSuit.defaultCtx = context.WithValue(testSuit.defaultCtx, utils.ContextUserNameKey, configFile.CreateBy.GetValue())
			got := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
			So(apimodel.Code_DecryptConfigFileException, ShouldEqual, apimodel.Code(got.GetCode().GetValue()))
		})

		Convey("存储层-查询配置文件-返回error", func() {
			storage := storemock.NewMockStore(ctrl)
			storage.EXPECT().GetConfigFileGroup(gomock.Any(), gomock.Any()).AnyTimes().Return(&model.ConfigFileGroup{}, nil)
			storage.EXPECT().GetConfigFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, errors.New("mock storage error"))
			testSuit.testServer.storage = storage

			configFile := assembleConfigFile()

			got := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
			So(apimodel.Code_StoreLayerException, ShouldEqual, apimodel.Code(got.GetCode().GetValue()))
		})

		Convey("存储层-创建配置文件-返回error", func() {
			storage := storemock.NewMockStore(ctrl)
			storage.EXPECT().GetConfigFileGroup(gomock.Any(), gomock.Any()).AnyTimes().Return(&model.ConfigFileGroup{}, nil)
			storage.EXPECT().GetConfigFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
			storage.EXPECT().CreateConfigFile(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, errors.New("mock storage error"))
			testSuit.testServer.storage = storage

			configFile := assembleConfigFile()

			got := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
			So(apimodel.Code_StoreLayerException, ShouldEqual, apimodel.Code(got.GetCode().GetValue()))
		})
	})
}

func TestServer_GetConfigFileBaseInfo(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Convey("获取配置文件基本信息", t, func() {
		Convey("存储层-查询配置文件-返回error", func() {
			storage := storemock.NewMockStore(ctrl)
			storage.EXPECT().GetConfigFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, errors.New("mock storage error"))
			testSuit.testServer.storage = storage

			configFile := assembleConfigFile()
			got := testSuit.testService.GetConfigFileBaseInfo(testSuit.defaultCtx, configFile.Namespace.Value, configFile.Group.Value, configFile.Name.Value)
			So(apimodel.Code_StoreLayerException, ShouldEqual, apimodel.Code(got.GetCode().GetValue()))
		})

		Convey("解密配置文件-返回error", func() {
			storage := storemock.NewMockStore(ctrl)
			storage.EXPECT().GetConfigFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&model.ConfigFile{
				CreateBy: operator,
			}, nil)
			storage.EXPECT().QueryTagByConfigFile(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]*model.ConfigFileTag{{Key: utils.ConfigFileTagKeyDataKey, Value: "abc"}}, nil)
			testSuit.testServer.storage = storage

			crypto := &aes.AESCrypto{}
			encryptFunc := ApplyMethod(reflect.TypeOf(crypto), "Decrypt", func(_ *aes.AESCrypto, plaintext string, key []byte) (string, error) {
				return "", errors.New("mock encrypt error")
			})
			defer encryptFunc.Reset()

			configFile := assembleConfigFile()
			got := testSuit.testService.GetConfigFileBaseInfo(testSuit.defaultCtx, configFile.Namespace.Value, configFile.Group.Value, configFile.Name.Value)
			So(apimodel.Code_DecryptConfigFileException, ShouldEqual, apimodel.Code(got.GetCode().GetValue()))
		})

	})
}
