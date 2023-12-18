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

package config_test

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/wrappers"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/plugin/crypto/aes"
	storemock "github.com/polarismesh/polaris/store/mock"
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
	testSuit := &ConfigCenterTest{}
	if err := testSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
		testSuit.Destroy()
	})

	t.Run("step1-query", func(t *testing.T) {
		rsp := testSuit.ConfigServer().GetConfigFileRichInfo(testSuit.DefaultCtx, &apiconfig.ConfigFile{
			Namespace: utils.NewStringValue(testNamespace),
			Group:     utils.NewStringValue(testGroup),
			Name:      utils.NewStringValue(testFile),
		})
		assert.Equal(t, uint32(api.NotFoundResource), rsp.Code.GetValue())
		assert.Nil(t, rsp.ConfigFile)
	})

	t.Run("step2-create", func(t *testing.T) {
		configFile := assembleConfigFile()
		rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, configFile)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

		// 重复创建
		rsp2 := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, configFile)
		assert.Equal(t, uint32(api.ExistedResource), rsp2.Code.GetValue())

		// 创建完之后再查询
		rsp3 := testSuit.ConfigServer().GetConfigFileRichInfo(testSuit.DefaultCtx, &apiconfig.ConfigFile{
			Namespace: utils.NewStringValue(testNamespace),
			Group:     utils.NewStringValue(testGroup),
			Name:      utils.NewStringValue(testFile),
		})
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())
		assert.NotNil(t, rsp3.ConfigFile)
		assert.Equal(t, testNamespace, rsp3.ConfigFile.Namespace.GetValue())
		assert.Equal(t, testGroup, rsp3.ConfigFile.Group.GetValue())
		assert.Equal(t, testFile, rsp3.ConfigFile.Name.GetValue())
		assert.Equal(t, configFile.Content.GetValue(), rsp3.ConfigFile.Content.GetValue())
		assert.Equal(t, configFile.Format.GetValue(), rsp3.ConfigFile.Format.GetValue())
	})

	t.Run("step3-update", func(t *testing.T) {
		testSuit.DefaultCtx = context.WithValue(testSuit.DefaultCtx, utils.ContextUserNameKey, "polaris")

		rsp := testSuit.ConfigServer().GetConfigFileRichInfo(testSuit.DefaultCtx, &apiconfig.ConfigFile{
			Namespace: utils.NewStringValue(testNamespace),
			Group:     utils.NewStringValue(testGroup),
			Name:      utils.NewStringValue(testFile),
		})
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

		configFile := rsp.ConfigFile
		newContent := "k1=v1"
		modifyBy := "polaris"
		configFile.Content = utils.NewStringValue(newContent)
		configFile.ModifyBy = utils.NewStringValue(modifyBy)

		rsp2 := testSuit.ConfigServer().UpdateConfigFile(testSuit.DefaultCtx, configFile)
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		assert.Equal(t, newContent, rsp2.ConfigFile.Content.GetValue())

		// 更新完之后再查询
		rsp3 := testSuit.ConfigServer().GetConfigFileRichInfo(testSuit.DefaultCtx, &apiconfig.ConfigFile{
			Namespace: utils.NewStringValue(testNamespace),
			Group:     utils.NewStringValue(testGroup),
			Name:      utils.NewStringValue(testFile),
		})
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
		rsp := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, configFileRelease)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

		rsp2 := testSuit.ConfigServer().DeleteConfigFile(testSuit.DefaultCtx, &apiconfig.ConfigFile{
			Namespace: utils.NewStringValue(testNamespace),
			Group:     utils.NewStringValue(testGroup),
			Name:      utils.NewStringValue(testFile),
		})
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())

		// 删除后，查询不到
		rsp3 := testSuit.ConfigServer().GetConfigFileRichInfo(testSuit.DefaultCtx, &apiconfig.ConfigFile{
			Namespace: utils.NewStringValue(testNamespace),
			Group:     utils.NewStringValue(testGroup),
			Name:      utils.NewStringValue(testFile),
		})
		assert.Equal(t, uint32(api.NotFoundResource), rsp3.Code.GetValue())
		assert.Nil(t, rsp2.ConfigFile)
	})

	t.Run("step5-search-by-group", func(t *testing.T) {
		group := "group11"
		for i := 0; i < size; i++ {
			rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, assembleConfigFileWithFixedGroupAndRandomFileName(group))
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		}

		// 第一页
		rsp2 := testSuit.ConfigServer().SearchConfigFile(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"group":     group,
			"offset":    "0",
			"limit":     "3",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		assert.Equal(t, uint32(size), rsp2.Total.GetValue())
		assert.Equal(t, 3, len(rsp2.ConfigFiles))

		// 最后一页
		rsp3 := testSuit.ConfigServer().SearchConfigFile(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"group":     group,
			"offset":    "6",
			"limit":     "3",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())
		assert.Equal(t, uint32(size), rsp3.Total.GetValue())
		assert.Equal(t, 1, len(rsp3.ConfigFiles))

		// group为空
		rsp4 := testSuit.ConfigServer().SearchConfigFile(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"group":     "",
			"offset":    "0",
			"limit":     "3",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp4.Code.GetValue())
		assert.Equal(t, uint32(size), rsp4.Total.GetValue())
		assert.Equal(t, 3, len(rsp4.ConfigFiles))

		// group 模糊搜索
		rsp5 := testSuit.ConfigServer().SearchConfigFile(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"group":     "group1*",
			"offset":    "0",
			"limit":     "3",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp5.Code.GetValue())
		assert.Equal(t, uint32(size), rsp5.Total.GetValue())
		assert.Equal(t, 3, len(rsp5.ConfigFiles))
	})

	t.Run("step5-find-by-group", func(t *testing.T) {
		group := "group12"
		for i := 0; i < size; i++ {
			rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, assembleConfigFileWithFixedGroupAndRandomFileName(group))
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		}

		// 第一页
		rsp2 := testSuit.ConfigServer().SearchConfigFile(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"group":     group,
			"offset":    "0",
			"limit":     "3",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		assert.Equal(t, uint32(size), rsp2.Total.GetValue())
		assert.Equal(t, 3, len(rsp2.ConfigFiles))

		// 最后一页
		rsp3 := testSuit.ConfigServer().SearchConfigFile(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"group":     group,
			"offset":    "6",
			"limit":     "3",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())
		assert.Equal(t, uint32(size), rsp3.Total.GetValue())
		assert.Equal(t, 1, len(rsp3.ConfigFiles))
	})

	t.Run("step6-search-by-file", func(t *testing.T) {
		file := "file1.txt"
		for i := 0; i < size; i++ {
			rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, assembleConfigFileWithRandomGroupAndFixedFileName(file))
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		}

		// 第一页
		rsp2 := testSuit.ConfigServer().SearchConfigFile(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"name":      file,
			"offset":    "0",
			"limit":     "3",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		assert.Equal(t, uint32(size), rsp2.Total.GetValue())
		assert.Equal(t, 3, len(rsp2.ConfigFiles))

		// 最后一页
		rsp3 := testSuit.ConfigServer().SearchConfigFile(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"name":      file,
			"offset":    "6",
			"limit":     "3",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())
		assert.Equal(t, uint32(size), rsp3.Total.GetValue())
		assert.Equal(t, 1, len(rsp3.ConfigFiles))

		// group,name都为空
		rsp4 := testSuit.ConfigServer().SearchConfigFile(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"offset":    "0",
			"limit":     "3",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp4.Code.GetValue())
		assert.Equal(t, uint32(size*3), rsp4.Total.GetValue()) // 总数为随机 group 和随机 fileName 总和
		assert.Equal(t, 3, len(rsp4.ConfigFiles))

		// fileName 模糊搜索
		rsp5 := testSuit.ConfigServer().SearchConfigFile(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"name":      "file1*",
			"offset":    "0",
			"limit":     "3",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp5.Code.GetValue())
		assert.Equal(t, uint32(size), rsp5.Total.GetValue())
		assert.Equal(t, 3, len(rsp5.ConfigFiles))
	})

	t.Run("step7-search-by-tag", func(t *testing.T) {
		t.Skip()
		// 按 tag k1=v1 搜索
		rsp := testSuit.ConfigServer().SearchConfigFile(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"tags":      "k1,v1",
			"offset":    "0",
			"limit":     "3",
		})
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
				rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, configFile)
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
		rsp := testSuit.ConfigServer().ExportConfigFile(testSuit.DefaultCtx, configFileExport)
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
		rsp = testSuit.ConfigServer().ExportConfigFile(testSuit.DefaultCtx, configFileExport)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		// 导出参数错误：无效的命名空间
		configFileExport = &apiconfig.ConfigFileExportRequest{
			Namespace: utils.NewStringValue(""),
		}
		rsp = testSuit.ConfigServer().ExportConfigFile(testSuit.DefaultCtx, configFileExport)
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
		rsp = testSuit.ConfigServer().ExportConfigFile(testSuit.DefaultCtx, configFileExport)
		assert.Equal(t, api.InvalidParameter, rsp.Code.GetValue())
		// 导出配置不存在
		configFileExport = &apiconfig.ConfigFileExportRequest{
			Namespace: utils.NewStringValue("namespace_0"),
			Groups: []*wrappers.StringValue{
				utils.NewStringValue("group_10"),
			},
		}
		rsp = testSuit.ConfigServer().ExportConfigFile(testSuit.DefaultCtx, configFileExport)
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
		rsp := testSuit.ConfigServer().ImportConfigFile(testSuit.DefaultCtx, configFiles, utils.ConfigFileImportConflictSkip)
		assert.Equal(t, api.InvalidConfigFileName, rsp.Code.GetValue(), rsp.GetInfo().GetValue())
	})
	t.Run("step10-import-conflict-skip", func(t *testing.T) {
		namespace := "namespace_import_skip"
		group := fmt.Sprintf("group_0")
		for j := 0; j < 3; j++ {
			name := fmt.Sprintf("file_%d", j)
			configFile := assembleConfigFileWithNamespaceAndGroupAndName(namespace, group, name)
			rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, configFile)
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
		rsp := testSuit.ConfigServer().ImportConfigFile(testSuit.DefaultCtx, configFiles, utils.ConfigFileImportConflictSkip)
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
			rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, configFile)
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
		rsp := testSuit.ConfigServer().ImportConfigFile(testSuit.DefaultCtx, configFiles, utils.ConfigFileImportConflictOverwrite)
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
		rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, configFile)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue(), rsp.GetInfo().GetValue())

		// 重复创建
		rsp2 := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, configFile)
		assert.Equal(t, uint32(api.ExistedResource), rsp2.Code.GetValue(), rsp2.GetInfo().GetValue())

		// 创建完之后再查询
		rsp3 := testSuit.ConfigServer().GetConfigFileRichInfo(testSuit.DefaultCtx, &apiconfig.ConfigFile{
			Namespace: utils.NewStringValue(testNamespace),
			Group:     utils.NewStringValue(testGroup),
			Name:      utils.NewStringValue(testFile),
		})
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue(), rsp3.GetInfo().GetValue())
		assert.NotNil(t, rsp3.ConfigFile)
		assert.Equal(t, testNamespace, rsp3.ConfigFile.Namespace.GetValue())
		assert.Equal(t, testGroup, rsp3.ConfigFile.Group.GetValue())
		assert.Equal(t, testFile, rsp3.ConfigFile.Name.GetValue())
		assert.Equal(t, configFile.Content.GetValue(), rsp3.ConfigFile.Content.GetValue())
		assert.Equal(t, configFile.Format.GetValue(), rsp3.ConfigFile.Format.GetValue())
	})
}

// TestPublishConfigFile 测试配置文件发布相关的用例
func TestPublishConfigFile(t *testing.T) {
	testSuit := &ConfigCenterTest{}
	if err := testSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
		testSuit.Destroy()
	})

	configFile := assembleConfigFile()
	rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

	configFileRelease := assembleConfigFileRelease(configFile)
	rsp2 := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, configFileRelease)
	assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())

	rsp3 := testSuit.ConfigServer().GetConfigFileRelease(testSuit.DefaultCtx, &apiconfig.ConfigFileRelease{
		Namespace: utils.NewStringValue(testNamespace),
		Group:     utils.NewStringValue(testGroup),
		FileName:  utils.NewStringValue(testFile),
	})
	assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue(), rsp3.GetInfo().GetValue())
	assert.Equal(t, uint64(1), rsp3.ConfigFileRelease.Version.GetValue())
	assert.Equal(t, configFileRelease.Name.GetValue(), rsp3.ConfigFileRelease.Name.GetValue())
	assert.Equal(t, configFileRelease.CreateBy.GetValue(), rsp3.ConfigFileRelease.CreateBy.GetValue())
	assert.Equal(t, configFileRelease.CreateBy.GetValue(), rsp3.ConfigFileRelease.ModifyBy.GetValue())
	assert.Equal(t, configFile.Content.GetValue(), rsp3.ConfigFileRelease.Content.GetValue())

	rsp5 := testSuit.ConfigServer().GetConfigFileRichInfo(testSuit.DefaultCtx, &apiconfig.ConfigFile{
		Namespace: utils.NewStringValue(testNamespace),
		Group:     utils.NewStringValue(testGroup),
		Name:      utils.NewStringValue(testFile),
	})
	assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), rsp5.GetCode().GetValue())

	// 第二次修改发布
	secondReleaseContent := "k3=v3"
	secondReleaseFormat := utils.FileFormatHtml
	configFile.Content = utils.NewStringValue(secondReleaseContent)
	configFile.Format = utils.NewStringValue(secondReleaseFormat)

	rsp6 := testSuit.ConfigServer().UpdateConfigFile(testSuit.DefaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp6.Code.GetValue())

	configFileRelease.Name = utils.NewStringValue("")
	configFileRelease.CreateBy = utils.NewStringValue("ledou3")
	rsp7 := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, configFileRelease)
	assert.Equal(t, api.ExecuteSuccess, rsp7.Code.GetValue(), rsp7.GetInfo().GetValue())

	rsp9 := testSuit.ConfigServer().GetConfigFileReleaseHistories(testSuit.DefaultCtx, map[string]string{
		"namespace": testNamespace,
		"group":     testGroup,
		"name":      testFile,
		"offset":    "0",
		"limit":     "10",
		"endId":     "0",
	})
	assert.Equal(t, api.ExecuteSuccess, rsp9.Code.GetValue())
	assert.Equal(t, 2, len(rsp9.ConfigFileReleaseHistories))

}

func Test_encryptConfigFile(t *testing.T) {
	testSuit := &ConfigCenterTest{}
	if err := testSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testSuit.Destroy()
	})

	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()
	type args struct {
		ctx        context.Context
		algorithm  string
		configFile *model.ConfigFile
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
				configFile: &model.ConfigFile{
					Content: "polaris",
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
				configFile: &model.ConfigFile{
					Content: "polaris",
				},
				dataKey: base64.StdEncoding.EncodeToString(dataKey),
			},
			want:    "YnLZ0SYuujFBHjYHAZVN5A==",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testSuit.OriginConfigServer()
			err := s.TestEncryptConfigFile(tt.args.ctx, tt.args.configFile, tt.args.algorithm, tt.args.dataKey)
			assert.Equal(t, tt.wantErr, err)
			if tt.want != "" {
				assert.Equal(t, tt.want, tt.args.configFile.Content)
			}
			hasDataKeyTag := false
			hasAlgoTag := false
			for tagKey, tagVal := range tt.args.configFile.Metadata {
				if tagKey == model.MetaKeyConfigFileDataKey {
					hasDataKeyTag = true
					if tt.args.dataKey != "" {
						assert.Equal(t, tt.args.dataKey, tagVal)
					}
				}
				if tagKey == model.MetaKeyConfigFileEncryptAlgo {
					hasAlgoTag = true
					assert.Equal(t, tt.args.algorithm, tagVal)
				}
			}
			assert.True(t, hasDataKeyTag)
			assert.True(t, hasAlgoTag)
		})
	}
}

func Test_decryptConfigFile(t *testing.T) {
	testSuit := &ConfigCenterTest{}
	if err := testSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testSuit.Destroy()
	})
	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()
	type args struct {
		ctx        context.Context
		configFile *model.ConfigFile
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
				configFile: &model.ConfigFile{
					Content: "YnLZ0SYuujFBHjYHAZVN5A==",
					Metadata: map[string]string{
						model.MetaKeyConfigFileDataKey:     base64.StdEncoding.EncodeToString(dataKey),
						model.MetaKeyConfigFileEncryptAlgo: "AES",
					},
					CreateBy: "polaris",
				},
			},
			want:    "polaris",
			wantErr: nil,
		},
		{
			name: "non creator can decrypt config file",
			args: args{
				ctx: context.WithValue(context.Background(), utils.ContextUserNameKey, "test"),
				configFile: &model.ConfigFile{
					Content: "YnLZ0SYuujFBHjYHAZVN5A==",
					Metadata: map[string]string{
						model.MetaKeyConfigFileDataKey:     base64.StdEncoding.EncodeToString(dataKey),
						model.MetaKeyConfigFileEncryptAlgo: "AES",
					},
					CreateBy: "polaris",
				},
			},
			want:    "polaris",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testSuit.OriginConfigServer()
			err := s.TestDecryptConfigFile(tt.args.ctx, tt.args.configFile)
			assert.Equal(t, tt.wantErr, err, tt.name)
			assert.Equal(t, tt.want, tt.args.configFile.Content, tt.name)
			for tagKey := range tt.args.configFile.Metadata {
				if tagKey == model.MetaKeyConfigFileDataKey {
					t.Fatal("config tags has data key")
				}
			}
		})
	}
}

func Test_GetConfigEncryptAlgorithm(t *testing.T) {
	testSuit := &ConfigCenterTest{}
	if err := testSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testSuit.Destroy()
	})
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
			rsp := testSuit.OriginConfigServer().GetAllConfigEncryptAlgorithms(testSuit.DefaultCtx)
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
			assert.Equal(t, tt.want, rsp.GetAlgorithms())
		})
	}
}

func Test_CreateConfigFile(t *testing.T) {
	testSuit := &ConfigCenterTest{}
	if err := testSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
		testSuit.Destroy()
	})
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("加密配置文件-返回error", func(t *testing.T) {
		crypto := &MockCrypto{}
		testSuit.OriginConfigServer().TestMockCryptoManager(&MockCryptoManager{
			repos: map[string]plugin.Crypto{
				crypto.Name(): crypto,
			},
		})

		configFile := assembleEncryptConfigFile()
		got := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, configFile)
		assert.Equal(t, apimodel.Code_EncryptConfigFileException, apimodel.Code(got.GetCode().GetValue()))
	})
}

func Test_GetConfigFileRichInfo(t *testing.T) {
	testSuit := &ConfigCenterTest{}
	if err := testSuit.Initialize(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
		testSuit.Destroy()
	})
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("获取配置文件基本信息-解密配置文件-返回error", func(t *testing.T) {
		configFile := assembleConfigFile()

		storage := storemock.NewMockStore(ctrl)
		storage.EXPECT().GetConfigFile(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&model.ConfigFile{
			Namespace: configFile.Namespace.Value,
			Group:     configFile.Group.Value,
			Name:      configFile.Name.Value,
			CreateBy:  operator,
		}, nil)

		svr := testSuit.OriginConfigServer()
		svr.TestMockStore(storage)
		svr.TestMockCryptoManager(&MockCryptoManager{
			repos: map[string]plugin.Crypto{
				(&aes.AESCrypto{}).Name(): &MockCrypto{
					mockDecrypt: func(cryptotext string, key []byte) (string, error) {
						return "", errors.New("mock encrypt error")
					},
				},
			},
		})
		got := testSuit.ConfigServer().GetConfigFileRichInfo(testSuit.DefaultCtx, configFile)
		assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(got.GetCode().GetValue()), got.GetInfo().GetValue())
	})
}

type MockCryptoManager struct {
	repos map[string]plugin.Crypto
}

func (m *MockCryptoManager) Name() string {
	return ""
}

func (m *MockCryptoManager) Initialize() error {
	return nil
}

func (m *MockCryptoManager) Destroy() error {
	return nil
}

func (m *MockCryptoManager) GetCryptoAlgoNames() []string {
	return []string{}
}

func (m *MockCryptoManager) GetCrypto(algo string) (plugin.Crypto, error) {
	val, ok := m.repos[algo]
	if !ok {
		return nil, errors.New("Not Exist")
	}
	return val, nil
}

type MockCrypto struct {
	mockDecrypt func(cryptotext string, key []byte) (string, error)
}

func (m *MockCrypto) Name() string {
	return (&aes.AESCrypto{}).Name()
}

func (m *MockCrypto) Initialize(c *plugin.ConfigEntry) error {
	return nil
}

func (m *MockCrypto) Destroy() error {
	return nil
}

func (m *MockCrypto) GenerateKey() ([]byte, error) {
	return nil, errors.New("Not Support")
}

func (m *MockCrypto) Encrypt(plaintext string, key []byte) (string, error) {
	return "", errors.New("Not Support")
}

func (m *MockCrypto) Decrypt(cryptotext string, key []byte) (string, error) {
	return "", errors.New("Not Support")
}
