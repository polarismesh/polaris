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

	"github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/common/utils"
)

// Test_PublishConfigFile 测试配置文件发布
func Test_PublishConfigFile(t *testing.T) {
	testSuit := newConfigCenterTestSuit(t)

	var (
		mockNamespace   = "mock_namespace"
		mockGroup       = "mock_group"
		mockFileName    = "mock_filename"
		mockReleaseName = "mock_release"
		mockContent     = "mock_content"
	)

	t.Run("pubslish_file_noexist", func(t *testing.T) {
		t.Run("namespace_not_exist", func(t *testing.T) {
			pubResp := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
				Name:      utils.NewStringValue(mockReleaseName),
				Namespace: utils.NewStringValue(mockNamespace),
				Group:     utils.NewStringValue(mockGroup),
				FileName:  utils.NewStringValue(mockFileName),
			})
			// 发布失败
			assert.Equal(t, uint32(apimodel.Code_NotFoundNamespace), pubResp.GetCode().GetValue(), pubResp.GetInfo().GetValue())
		})

		t.Run("file_not_exist", func(t *testing.T) {
			testSuit.NamespaceServer().CreateNamespace(testSuit.DefaultCtx, &apimodel.Namespace{
				Name: utils.NewStringValue(mockNamespace),
			})

			pubResp := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
				Name:      utils.NewStringValue(mockReleaseName),
				Namespace: utils.NewStringValue(mockNamespace),
				Group:     utils.NewStringValue(mockGroup),
				FileName:  utils.NewStringValue(mockFileName),
			})
			// 发布失败
			assert.Equal(t, uint32(apimodel.Code_NotFoundResource), pubResp.GetCode().GetValue(), pubResp.GetInfo().GetValue())
		})
	})

	t.Run("normal_publish", func(t *testing.T) {
		pubResp := testSuit.ConfigServer().UpsertAndReleaseConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFilePublishInfo{
			ReleaseName:        utils.NewStringValue(mockReleaseName),
			Namespace:          utils.NewStringValue(mockNamespace),
			Group:              utils.NewStringValue(mockGroup),
			FileName:           utils.NewStringValue(mockFileName),
			Content:            utils.NewStringValue(mockContent),
			Comment:            utils.NewStringValue("mock_comment"),
			Format:             utils.NewStringValue("yaml"),
			ReleaseDescription: utils.NewStringValue("mock_releaseDescription"),
			Tags: []*config_manage.ConfigFileTag{
				{
					Key:   utils.NewStringValue("mock_key"),
					Value: utils.NewStringValue("mock_value"),
				},
			},
		})

		// 正常发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), pubResp.GetCode().GetValue(), pubResp.GetInfo().GetValue())
	})

	t.Run("get_config_file_release", func(t *testing.T) {
		resp := testSuit.ConfigServer().GetConfigFileRelease(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
			Name:      utils.NewStringValue(mockReleaseName),
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
		})
		// 获取配置发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
		// 配置内容符合预期
		assert.Equal(t, mockContent, resp.GetConfigFileRelease().GetContent().GetValue(), resp.GetInfo().GetValue())
		// 必须是处于使用状态
		assert.True(t, resp.GetConfigFileRelease().GetActive().GetValue(), resp.GetInfo().GetValue())
	})

	t.Run("republish_config_file", func(t *testing.T) {
		// 再次发布
		resp := testSuit.ConfigServer().UpsertAndReleaseConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFilePublishInfo{
			ReleaseName: utils.NewStringValue(mockReleaseName + "Second"),
			Namespace:   utils.NewStringValue(mockNamespace),
			Group:       utils.NewStringValue(mockGroup),
			FileName:    utils.NewStringValue(mockFileName),
			Content:     utils.NewStringValue(mockContent + "Second"),
		})
		// 获取配置发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
	})

	t.Run("reget_config_file_release", func(t *testing.T) {
		secondResp := testSuit.ConfigServer().GetConfigFileRelease(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
			Name:      utils.NewStringValue(mockReleaseName + "Second"),
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
		})
		// 获取配置发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), secondResp.GetCode().GetValue(), secondResp.GetInfo().GetValue())
		// 配置内容符合预期
		assert.Equal(t, mockContent+"Second", secondResp.GetConfigFileRelease().GetContent().GetValue(), secondResp.GetInfo().GetValue())
		// 必须是处于使用状态
		assert.True(t, secondResp.GetConfigFileRelease().GetActive().GetValue(), secondResp.GetInfo().GetValue())

		firstResp := testSuit.ConfigServer().GetConfigFileRelease(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
			Name:      utils.NewStringValue(mockReleaseName),
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
		})
		// 获取配置发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), firstResp.GetCode().GetValue(), firstResp.GetInfo().GetValue())
		// 必须是处于非使用状态
		assert.False(t, firstResp.GetConfigFileRelease().GetActive().GetValue(), firstResp.GetInfo().GetValue())

		// 后一次的发布要比前面一次的发布来的大
		assert.True(t, secondResp.GetConfigFileRelease().GetVersion().GetValue() > firstResp.GetConfigFileRelease().GetVersion().GetValue())
	})

	t.Run("client_get_configfile", func(t *testing.T) {
		// 客户端获取符合预期, 这里强制触发一次缓存数据同步
		_ = testSuit.CacheMgr().TestUpdate()
		clientResp := testSuit.ConfigServer().GetConfigFileForClient(testSuit.DefaultCtx, &config_manage.ClientConfigFileInfo{
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
		})

		// 获取配置发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), clientResp.GetCode().GetValue(), clientResp.GetInfo().GetValue())
		assert.Equal(t, mockContent+"Second", clientResp.GetConfigFile().GetContent().GetValue())
	})
}

// Test_RollbackConfigFileRelease 测试配置发布回滚
func Test_RollbackConfigFileRelease(t *testing.T) {
	testSuit := newConfigCenterTestSuit(t)

	var (
		mockNamespace   = "mock_namespace"
		mockGroup       = "mock_group"
		mockFileName    = "mock_filename"
		mockReleaseName = "mock_release"
		mockContent     = "mock_content"
	)

	t.Run("first_publish", func(t *testing.T) {
		resp := testSuit.ConfigServer().UpsertAndReleaseConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFilePublishInfo{
			ReleaseName: utils.NewStringValue(mockReleaseName),
			Namespace:   utils.NewStringValue(mockNamespace),
			Group:       utils.NewStringValue(mockGroup),
			FileName:    utils.NewStringValue(mockFileName),
			Content:     utils.NewStringValue(mockContent),
		})
		// 正常发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
	})

	t.Run("republish_config_file", func(t *testing.T) {
		// 再次发布
		resp := testSuit.ConfigServer().UpsertAndReleaseConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFilePublishInfo{
			ReleaseName: utils.NewStringValue(mockReleaseName + "Second"),
			Namespace:   utils.NewStringValue(mockNamespace),
			Group:       utils.NewStringValue(mockGroup),
			FileName:    utils.NewStringValue(mockFileName),
			Content:     utils.NewStringValue(mockContent + "Second"),
		})
		// 正常发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())

		secondResp := testSuit.ConfigServer().GetConfigFileRelease(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
			Name:      utils.NewStringValue(mockReleaseName + "Second"),
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
		})
		// 获取配置发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), secondResp.GetCode().GetValue(), secondResp.GetInfo().GetValue())
		// 配置内容符合预期
		assert.Equal(t, mockContent+"Second", secondResp.GetConfigFileRelease().GetContent().GetValue(), secondResp.GetInfo().GetValue())
		// 必须是处于使用状态
		assert.True(t, secondResp.GetConfigFileRelease().GetActive().GetValue(), secondResp.GetInfo().GetValue())
	})

	// 回滚某个配置版本
	t.Run("rollback_config_release", func(t *testing.T) {
		resp := testSuit.ConfigServer().RollbackConfigFileReleases(testSuit.DefaultCtx, []*config_manage.ConfigFileRelease{
			{
				Name:      utils.NewStringValue(mockReleaseName),
				Namespace: utils.NewStringValue(mockNamespace),
				Group:     utils.NewStringValue(mockGroup),
				FileName:  utils.NewStringValue(mockFileName),
			},
		})

		// 正常回滚成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
		secondResp := testSuit.ConfigServer().GetConfigFileRelease(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
			Name:      utils.NewStringValue(mockReleaseName + "Second"),
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
		})
		// 获取配置发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), secondResp.GetCode().GetValue(), secondResp.GetInfo().GetValue())
		// 必须是处于非使用状态
		assert.False(t, secondResp.GetConfigFileRelease().GetActive().GetValue(), secondResp.GetInfo().GetValue())

		firstResp := testSuit.ConfigServer().GetConfigFileRelease(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
			Name:      utils.NewStringValue(mockReleaseName),
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
		})
		// 获取配置发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
		// 必须是处于使用状态
		assert.True(t, firstResp.GetConfigFileRelease().GetActive().GetValue(), resp.GetInfo().GetValue())

		// 客户端获取符合预期, 这里强制触发一次缓存数据同步
		_ = testSuit.CacheMgr().TestUpdate()
		clientResp := testSuit.ConfigServer().GetConfigFileForClient(testSuit.DefaultCtx, &config_manage.ClientConfigFileInfo{
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
		})

		// 获取配置发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), clientResp.GetCode().GetValue(), clientResp.GetInfo().GetValue())
		assert.Equal(t, mockContent, clientResp.GetConfigFile().GetContent().GetValue())
		assert.Equal(t, firstResp.GetConfigFileRelease().GetVersion().GetValue(), clientResp.GetConfigFile().GetVersion().GetValue())
	})

	// 回滚不存在的目标版本
	t.Run("rollback_notexist_release", func(t *testing.T) {
		resp := testSuit.ConfigServer().RollbackConfigFileReleases(testSuit.DefaultCtx, []*config_manage.ConfigFileRelease{
			{
				Name:      utils.NewStringValue(mockReleaseName + "_NotExist"),
				Namespace: utils.NewStringValue(mockNamespace),
				Group:     utils.NewStringValue(mockGroup),
				FileName:  utils.NewStringValue(mockFileName),
			},
		})

		// 回滚失败成功
		assert.Equal(t, uint32(apimodel.Code_NotFoundResource), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
	})
}

// Test_DeleteConfigFileRelease 测试删除配置发布
func Test_DeleteConfigFileRelease(t *testing.T) {

}
