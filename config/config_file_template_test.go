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
	"testing"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

const (
	templateName1 = "t1"
	templateName2 = "t2"
)

// TestConfigFileTemplateCRUD the base test for config file template
func TestConfigFileTemplateCRUD(t *testing.T) {
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

	t.Run("first-query", func(t *testing.T) {
		queryRsp := testSuit.ConfigServer().GetConfigFileTemplate(testSuit.DefaultCtx, templateName1)
		assert.Equal(t, api.NotFoundResource, queryRsp.Code.Value)
	})

	template1 := assembleConfigFileTemplate(templateName1)
	t.Run("first-create", func(t *testing.T) {
		createRsp := testSuit.ConfigServer().CreateConfigFileTemplate(testSuit.DefaultCtx, template1)
		assert.Equal(t, api.ExecuteSuccess, createRsp.Code.GetValue())
		//repeat create
		createRsp = testSuit.ConfigServer().CreateConfigFileTemplate(testSuit.DefaultCtx, template1)
		assert.Equal(t, api.ExistedResource, createRsp.Code.GetValue(), createRsp.GetInfo().GetValue())
	})

	t.Run("second-query", func(t *testing.T) {
		queryRsp := testSuit.ConfigServer().GetConfigFileTemplate(testSuit.DefaultCtx, templateName1)
		assert.Equal(t, api.ExecuteSuccess, queryRsp.Code.Value)
		assert.Equal(t, template1.Name.GetValue(), queryRsp.ConfigFileTemplate.Name.GetValue())
		assert.Equal(t, template1.Content.GetValue(), queryRsp.ConfigFileTemplate.Content.GetValue())
		assert.Equal(t, template1.Comment.GetValue(), queryRsp.ConfigFileTemplate.Comment.GetValue())
		assert.Equal(t, template1.Format.GetValue(), queryRsp.ConfigFileTemplate.Format.GetValue())
	})

	template2 := assembleConfigFileTemplate(templateName2)
	t.Run("second-create", func(t *testing.T) {
		createRsp := testSuit.ConfigServer().CreateConfigFileTemplate(testSuit.DefaultCtx, template2)
		assert.Equal(t, api.ExecuteSuccess, createRsp.Code.GetValue())
	})

	t.Run("query-all", func(t *testing.T) {
		rsp := testSuit.ConfigServer().GetAllConfigFileTemplates(testSuit.DefaultCtx)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.True(t, 2 <= len(rsp.ConfigFileTemplates))
	})
}

func assembleConfigFileTemplate(name string) *apiconfig.ConfigFileTemplate {
	return &apiconfig.ConfigFileTemplate{
		Name:     utils.NewStringValue(name),
		Content:  utils.NewStringValue("some content"),
		Comment:  utils.NewStringValue("comment"),
		Format:   utils.NewStringValue("json"),
		CreateBy: utils.NewStringValue("testUser"),
		ModifyBy: utils.NewStringValue("testUser"),
	}
}

func TestServer_CreateConfigFileTemplateParam(t *testing.T) {
	testSuit := newConfigCenterTestSuit(t)
	_ = testSuit

	var (
		mockTemplName = "mock_templname"
		mockContent   = "mock_content"
	)

	t.Run("invalid_tpl_name", func(t *testing.T) {
		rsp := testSuit.ConfigServer().CreateConfigFileTemplate(testSuit.DefaultCtx, &apiconfig.ConfigFileTemplate{
			Content: wrapperspb.String(mockContent),
		})
		assert.Equal(t, uint32(apimodel.Code_InvalidConfigFileTemplateName), rsp.Code.GetValue())
	})

	t.Run("content_too_long", func(t *testing.T) {
		mockContentL := "mock_content"
		for {
			if len(mockContentL) > int(testSuit.GetBootstrapConfig().Config.ContentMaxLength) {
				break
			}
			mockContentL += mockContentL
		}
		rsp := testSuit.ConfigServer().CreateConfigFileTemplate(testSuit.DefaultCtx, &apiconfig.ConfigFileTemplate{
			Name:    wrapperspb.String(mockTemplName),
			Content: wrapperspb.String(mockContentL),
		})
		assert.Equal(t, uint32(apimodel.Code_InvalidConfigFileContentLength), rsp.Code.GetValue(), rsp.GetInfo().GetValue())
	})

	t.Run("no_content", func(t *testing.T) {
		rsp := testSuit.ConfigServer().CreateConfigFileTemplate(testSuit.DefaultCtx, &apiconfig.ConfigFileTemplate{
			Name:    wrapperspb.String(mockTemplName),
			Content: wrapperspb.String(""),
		})
		assert.Equal(t, uint32(apimodel.Code_BadRequest), rsp.Code.GetValue())
	})
}
