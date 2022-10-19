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
	"testing"

	"github.com/stretchr/testify/assert"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

// TestConfigFileTemplateCRUD the base test for config file template
func TestConfigFileTemplateCRUD(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()

	templateName := "t1"
	t.Run("first-query", func(t *testing.T) {
		queryRsp := testSuit.testService.GetConfigFileTemplate(testSuit.defaultCtx, templateName)
		assert.Equal(t, api.NotFoundResource, queryRsp.Code.Value)
	})

	template1 := assembleConfigFileTemplate(templateName)
	t.Run("first-create", func(t *testing.T) {
		createRsp := testSuit.testService.CreateConfigFileTemplate(testSuit.defaultCtx, template1)
		assert.Equal(t, api.ExecuteSuccess, createRsp.Code.GetValue())
		//repeat create
		createRsp = testSuit.testService.CreateConfigFileTemplate(testSuit.defaultCtx, template1)
		assert.Equal(t, api.BadRequest, createRsp.Code.GetValue())
	})

	t.Run("second-query", func(t *testing.T) {
		queryRsp := testSuit.testService.GetConfigFileTemplate(testSuit.defaultCtx, templateName)
		assert.Equal(t, api.ExecuteSuccess, queryRsp.Code.Value)
		assert.Equal(t, template1.Name.GetValue(), queryRsp.ConfigFileTemplate.Name.GetValue())
		assert.Equal(t, template1.Content.GetValue(), queryRsp.ConfigFileTemplate.Content.GetValue())
		assert.Equal(t, template1.Comment.GetValue(), queryRsp.ConfigFileTemplate.Comment.GetValue())
		assert.Equal(t, template1.Format.GetValue(), queryRsp.ConfigFileTemplate.Format.GetValue())
	})

	template2 := assembleConfigFileTemplate("t2")
	t.Run("second-create", func(t *testing.T) {
		createRsp := testSuit.testService.CreateConfigFileTemplate(testSuit.defaultCtx, template2)
		assert.Equal(t, api.ExecuteSuccess, createRsp.Code.GetValue())
	})

	t.Run("query-all", func(t *testing.T) {
		rsp := testSuit.testService.GetAllConfigFileTemplates(testSuit.defaultCtx)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())
		assert.True(t, 2 <= len(rsp.ConfigFileTemplates))
	})
}

func assembleConfigFileTemplate(name string) *api.ConfigFileTemplate {
	return &api.ConfigFileTemplate{
		Name:     utils.NewStringValue(name),
		Content:  utils.NewStringValue("some content"),
		Comment:  utils.NewStringValue("comment"),
		Format:   utils.NewStringValue("json"),
		CreateBy: utils.NewStringValue("testUser"),
		ModifyBy: utils.NewStringValue("testUser"),
	}
}
