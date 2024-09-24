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
	"fmt"
	"reflect"
	"testing"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
)

var (
	randomGroupPrefix = "randomGroup"
	randomGroupSize   = uint32(7)
)

func Test_CheckCreateFileGroupParam(t *testing.T) {
	testSuit := newConfigCenterTestSuit(t)

	t.Run("invalid_nil_param", func(t *testing.T) {
		rsp := testSuit.ConfigServer().CreateConfigFileGroup(testSuit.DefaultCtx, nil)
		assert.Equal(t, uint32(apimodel.Code_InvalidParameter), rsp.Code.GetValue())
	})
	t.Run("invalid_name", func(t *testing.T) {
		mockGroup := assembleConfigFileGroup()
		mockGroup.Name = wrapperspb.String("")
		rsp := testSuit.ConfigServer().CreateConfigFileGroup(testSuit.DefaultCtx, mockGroup)
		assert.Equal(t, uint32(apimodel.Code_InvalidConfigFileGroupName), rsp.Code.GetValue())
	})
	t.Run("invalid_namespace", func(t *testing.T) {
		mockGroup := assembleConfigFileGroup()
		mockGroup.Namespace = wrapperspb.String("")
		rsp := testSuit.ConfigServer().CreateConfigFileGroup(testSuit.DefaultCtx, mockGroup)
		assert.Equal(t, uint32(apimodel.Code_InvalidNamespaceName), rsp.Code.GetValue())
	})
	t.Run("invalid_metadata_len", func(t *testing.T) {
		mockGroup := assembleConfigFileGroup()
		mockGroup.Metadata = map[string]string{}
		for i := 0; i < utils.MaxMetadataLength+10; i++ {
			mockGroup.Metadata[fmt.Sprintf("Key_%d", i)] = fmt.Sprintf("Value_%d", i)
		}
		rsp := testSuit.ConfigServer().CreateConfigFileGroup(testSuit.DefaultCtx, mockGroup)
		assert.Equal(t, uint32(apimodel.Code_InvalidMetadata), rsp.Code.GetValue())
	})
}

// TestConfigFileGroupCRUD 测试配置文件组增删改查
func TestConfigFileGroupCRUD(t *testing.T) {
	testSuit := newConfigCenterTestSuit(t)

	// 查询不存在的 group
	t.Run("step1-query-none", func(t *testing.T) {
		rsp := testSuit.ConfigServer().QueryConfigFileGroups(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"group":     testGroup,
			"offset":    "0",
			"limit":     "1",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, 0, len(rsp.ConfigFileGroups))
	})

	// 创建 group
	t.Run("step2-create", func(t *testing.T) {
		t.Run("normal", func(t *testing.T) {
			rsp := testSuit.ConfigServer().CreateConfigFileGroup(testSuit.DefaultCtx, assembleConfigFileGroup())
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

			rsp2 := testSuit.ConfigServer().CreateConfigFileGroup(testSuit.DefaultCtx, assembleConfigFileGroup())
			assert.Equal(t, uint32(api.ExistedResource), rsp2.Code.GetValue())
		})
	})

	t.Run("step2-update", func(t *testing.T) {
		t.Run("no_change", func(t *testing.T) {
			rsp := testSuit.ConfigServer().UpdateConfigFileGroup(testSuit.DefaultCtx, assembleConfigFileGroup())
			assert.Equal(t, api.NoNeedUpdate, rsp.Code.GetValue(), rsp.GetInfo().GetValue())
		})
		t.Run("change", func(t *testing.T) {
			mockGroup := assembleConfigFileGroup()
			mockGroup.Comment = wrapperspb.String("v string changed")
			rsp := testSuit.ConfigServer().UpdateConfigFileGroup(testSuit.DefaultCtx, mockGroup)
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		})
	})

	// 再次查询 group，能查询到上一步创建的 group
	t.Run("step3-query-existed", func(t *testing.T) {
		rsp := testSuit.ConfigServer().QueryConfigFileGroups(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"group":     testGroup,
			"offset":    "0",
			"limit":     "1",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, 1, len(rsp.ConfigFileGroups))
	})

	// 创建配置文件
	t.Run("create-config-files", func(t *testing.T) {
		rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, assembleConfigFile())
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

		rsp2 := testSuit.ConfigServer().SearchConfigFile(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"group":     testGroup,
			"offset":    "0",
			"limit":     "10",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		assert.Equal(t, uint32(1), rsp2.Total.GetValue())
	})

	// 删除 group
	t.Run("step4-delete", func(t *testing.T) {
		t.Run("delete-noexist", func(t *testing.T) {
			rsp := testSuit.ConfigServer().DeleteConfigFileGroup(testSuit.DefaultCtx, testNamespace, "testGroup_noExist")
			assert.Equal(t, uint32(apimodel.Code_NotFoundResource), rsp.Code.GetValue(), rsp.GetInfo().GetValue())
		})
		t.Run("exist-subresources", func(t *testing.T) {
			rsp := testSuit.ConfigServer().DeleteConfigFileGroup(testSuit.DefaultCtx, testNamespace, testGroup)
			assert.Equal(t, uint32(apimodel.Code_ExistedResource), rsp.Code.GetValue(), rsp.GetInfo().GetValue())
		})
		t.Run("normal", func(t *testing.T) {
			rsp := testSuit.ConfigServer().DeleteConfigFile(testSuit.DefaultCtx, assembleConfigFile())
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue(), rsp.GetInfo().GetValue())

			rsp = testSuit.ConfigServer().DeleteConfigFileGroup(testSuit.DefaultCtx, testNamespace, testGroup)
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue(), rsp.GetInfo().GetValue())

			rsp2 := testSuit.ConfigServer().SearchConfigFile(testSuit.DefaultCtx, map[string]string{
				"namespace": testNamespace,
				"group":     testGroup,
				"offset":    "0",
				"limit":     "10",
			})
			assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
			assert.Equal(t, uint32(0), rsp2.GetTotal().GetValue())
		})
	})

	// 再次查询group，由于被删除，所以查不到
	t.Run("step5-query-none", func(t *testing.T) {
		rsp := testSuit.ConfigServer().QueryConfigFileGroups(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"group":     testGroup,
			"offset":    "0",
			"limit":     "1",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, 0, len(rsp.ConfigFileGroups))
	})

	// 创建 7个随机 group 和一个固定的 group
	t.Run("step6-create", func(t *testing.T) {
		for i := 0; i < int(randomGroupSize); i++ {
			rsp := testSuit.ConfigServer().CreateConfigFileGroup(testSuit.DefaultCtx, assembleRandomConfigFileGroup())
			assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		}

		rsp2 := testSuit.ConfigServer().CreateConfigFileGroup(testSuit.DefaultCtx, assembleConfigFileGroup())
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue(), rsp2.GetInfo().GetValue())
	})

	// 模糊查询
	t.Run("step7-query-random", func(t *testing.T) {
		rsp := testSuit.ConfigServer().QueryConfigFileGroups(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"name":      randomGroupPrefix + "*",
			"offset":    "0",
			"limit":     "2",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, 2, len(rsp.ConfigFileGroups))
		assert.Equal(t, randomGroupSize, rsp.Total.GetValue())
	})

	// 测试翻页
	t.Run("step8-query-by-page", func(t *testing.T) {
		// 最后一页
		rsp := testSuit.ConfigServer().QueryConfigFileGroups(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"name":      randomGroupPrefix + "*",
			"offset":    "6",
			"limit":     "2",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.Equal(t, 1, len(rsp.ConfigFileGroups))
		assert.Equal(t, randomGroupSize, rsp.Total.GetValue())

		// 超出页范围
		rsp2 := testSuit.ConfigServer().QueryConfigFileGroups(testSuit.DefaultCtx, map[string]string{
			"namespace": testNamespace,
			"name":      randomGroupPrefix + "*",
			"offset":    "8",
			"limit":     "2",
		})
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		assert.Equal(t, 0, len(rsp2.ConfigFileGroups))
		assert.Equal(t, randomGroupSize, rsp2.Total.GetValue())
	})
}

func TestServer_UpdateGroupAttribute(t *testing.T) {
	type args struct {
		saveData   *model.ConfigFileGroup
		updateData *model.ConfigFileGroup
	}
	tests := []struct {
		name  string
		args  args
		want  *model.ConfigFileGroup
		want1 bool
	}{
		{
			name: "01",
			args: args{
				saveData: &model.ConfigFileGroup{
					Comment:    "test",
					Business:   "test",
					Department: "test",
				},
				updateData: &model.ConfigFileGroup{
					Comment: "test-1",
				},
			},
			want: &model.ConfigFileGroup{
				Comment: "test-1",
			},
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &config.Server{}
			got, got1 := s.UpdateGroupAttribute(tt.args.saveData, tt.args.updateData)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Server.UpdateGroupAttribute() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Server.UpdateGroupAttribute() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
