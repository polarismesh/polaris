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
	"time"

	"github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
)

// Test_PublishConfigFile 测试配置文件发布
func Test_PublishConfigFile_Check(t *testing.T) {
	testSuit := newConfigCenterTestSuit(t)

	var (
		mockNamespace   = "mock_namespace"
		mockGroup       = "mock_group"
		mockFileName    = "mock_filename"
		mockReleaseName = "mock_release"
	)

	t.Run("参数检查", func(t *testing.T) {
		testSuit.NamespaceServer().CreateNamespace(testSuit.DefaultCtx, &apimodel.Namespace{
			Name: utils.NewStringValue(mockNamespace),
		})

		t.Run("invalid_file_name", func(t *testing.T) {
			pubResp := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
				Name:      utils.NewStringValue(mockReleaseName),
				Namespace: utils.NewStringValue(mockNamespace),
				Group:     utils.NewStringValue(mockGroup),
				// FileName:  utils.NewStringValue(mockFileName),
			})
			// 发布失败
			assert.Equal(t, uint32(apimodel.Code_InvalidConfigFileName), pubResp.GetCode().GetValue(), pubResp.GetInfo().GetValue())
		})
		t.Run("invalid_namespace", func(t *testing.T) {
			pubResp := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
				Name: utils.NewStringValue(mockReleaseName),
				// Namespace: utils.NewStringValue(mockNamespace),
				Group:    utils.NewStringValue(mockGroup),
				FileName: utils.NewStringValue(mockFileName),
			})
			// 发布失败
			assert.Equal(t, uint32(apimodel.Code_InvalidNamespaceName), pubResp.GetCode().GetValue(), pubResp.GetInfo().GetValue())
		})
		t.Run("invalid_group", func(t *testing.T) {
			pubResp := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
				Name:      utils.NewStringValue(mockReleaseName),
				Namespace: utils.NewStringValue(mockNamespace),
				// Group:     utils.NewStringValue(mockGroup),
				FileName: utils.NewStringValue(mockFileName),
			})
			// 发布失败
			assert.Equal(t, uint32(apimodel.Code_InvalidConfigFileGroupName), pubResp.GetCode().GetValue(), pubResp.GetInfo().GetValue())
		})
		t.Run("invalid_gray_publish", func(t *testing.T) {
			pubResp := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
				Name:        utils.NewStringValue(mockReleaseName),
				Namespace:   utils.NewStringValue(mockNamespace),
				Group:       utils.NewStringValue(mockGroup),
				FileName:    utils.NewStringValue(mockFileName),
				ReleaseType: wrapperspb.String(model.ReleaseTypeGray),
			})
			// 发布失败
			assert.Equal(t, uint32(apimodel.Code_InvalidMatchRule), pubResp.GetCode().GetValue(), pubResp.GetInfo().GetValue())
		})
	})
}

// Test_PublishConfigFile 测试配置文件发布
func Test_PublishConfigFile(t *testing.T) {
	testSuit := newConfigCenterTestSuit(t)

	var (
		mockNamespace   = "mock_namespace_pub"
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

	// 重新发布
	t.Run("normal_republish", func(t *testing.T) {
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

	// 创建一个 v1 的配置发布
	// 删除 v1 配置发布
	// 再创建一个 v1 的配置发布
	// 客户端可以正常读取到数据
	t.Run("create_delete_recreate_same", func(t *testing.T) {
		pubResp := testSuit.ConfigServer().UpsertAndReleaseConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFilePublishInfo{
			ReleaseName:        utils.NewStringValue(mockReleaseName + "same-v1"),
			Namespace:          utils.NewStringValue(mockNamespace + "same-v1"),
			Group:              utils.NewStringValue(mockGroup + "same-v1"),
			FileName:           utils.NewStringValue(mockFileName + "same-v1"),
			Content:            utils.NewStringValue(mockContent + "same-v1"),
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

		delResp := testSuit.ConfigServer().DeleteConfigFileReleases(testSuit.DefaultCtx, []*config_manage.ConfigFileRelease{
			{
				Name:      utils.NewStringValue(mockReleaseName + "same-v1"),
				Namespace: utils.NewStringValue(mockNamespace + "same-v1"),
				Group:     utils.NewStringValue(mockGroup + "same-v1"),
				FileName:  utils.NewStringValue(mockFileName + "same-v1"),
			},
		})
		// 删除成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), delResp.GetCode().GetValue(), delResp.GetInfo().GetValue())

		// 再次重新发布
		pubResp = testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
			Name:      utils.NewStringValue(mockReleaseName + "same-v1"),
			Namespace: utils.NewStringValue(mockNamespace + "same-v1"),
			Group:     utils.NewStringValue(mockGroup + "same-v1"),
			FileName:  utils.NewStringValue(mockFileName + "same-v1"),
		})

		// 再次重新发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), pubResp.GetCode().GetValue(), pubResp.GetInfo().GetValue())

		// 客户端读取数据正常
		_ = testSuit.CacheMgr().TestUpdate()

		cacheData := testSuit.CacheMgr().ConfigFile().GetRelease(model.ConfigFileReleaseKey{
			Namespace:   mockNamespace + "same-v1",
			Group:       mockGroup + "same-v1",
			FileName:    mockFileName + "same-v1",
			Name:        mockReleaseName + "same-v1",
			ReleaseType: model.ReleaseTypeFull,
		})
		assert.NotNil(t, cacheData)
		assert.Equal(t, mockContent+"same-v1", cacheData.Content)

		clientRsp := testSuit.ConfigServer().GetConfigFileWithCache(testSuit.DefaultCtx, &config_manage.ClientConfigFileInfo{
			Namespace: utils.NewStringValue(mockNamespace + "same-v1"),
			Group:     utils.NewStringValue(mockGroup + "same-v1"),
			FileName:  utils.NewStringValue(mockFileName + "same-v1"),
		})
		// 正常读取到数据
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), clientRsp.GetCode().GetValue(), clientRsp.GetInfo().GetValue())
	})

	t.Run("list_release_version", func(t *testing.T) {
		t.Run("invalid_namespace", func(t *testing.T) {
			queryRsp := testSuit.ConfigServer().GetConfigFileReleaseVersions(testSuit.DefaultCtx, map[string]string{
				"group":     mockGroup,
				"file_name": mockFileName,
			})
			assert.Equal(t, uint32(apimodel.Code_BadRequest), queryRsp.GetCode().GetValue())
			assert.Equal(t, "invalid namespace", queryRsp.GetInfo().GetValue())
		})
		t.Run("invalid_group", func(t *testing.T) {
			queryRsp := testSuit.ConfigServer().GetConfigFileReleaseVersions(testSuit.DefaultCtx, map[string]string{
				"namespace": mockNamespace,
				"file_name": mockFileName,
			})
			assert.Equal(t, uint32(apimodel.Code_BadRequest), queryRsp.GetCode().GetValue())
			assert.Equal(t, "invalid config group", queryRsp.GetInfo().GetValue())
		})
		t.Run("invalid_file_name", func(t *testing.T) {
			queryRsp := testSuit.ConfigServer().GetConfigFileReleaseVersions(testSuit.DefaultCtx, map[string]string{
				"group":     mockGroup,
				"namespace": mockNamespace,
			})
			assert.Equal(t, uint32(apimodel.Code_BadRequest), queryRsp.GetCode().GetValue())
			assert.Equal(t, "invalid config file name", queryRsp.GetInfo().GetValue())
		})
		t.Run("normal", func(t *testing.T) {
			queryRsp := testSuit.ConfigServer().GetConfigFileReleaseVersions(testSuit.DefaultCtx, map[string]string{
				"namespace": mockNamespace,
				"group":     mockGroup,
				"file_name": mockFileName,
			})
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), queryRsp.GetInfo().GetValue())
			assert.Equal(t, 1, len(queryRsp.ConfigFileReleases))
			assert.Equal(t, 1, int(queryRsp.GetTotal().GetValue()))
		})
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
		_ = testSuit.CacheMgr().TestUpdate()
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
		_ = testSuit.CacheMgr().TestUpdate()
		// 客户端获取符合预期, 这里强制触发一次缓存数据同步
		clientResp := testSuit.ConfigServer().GetConfigFileWithCache(testSuit.DefaultCtx, &config_manage.ClientConfigFileInfo{
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
		})

		// 获取配置发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), clientResp.GetCode().GetValue(), clientResp.GetInfo().GetValue())
		assert.Equal(t, mockContent+"Second", clientResp.GetConfigFile().GetContent().GetValue())
	})

	t.Run("normal_publish_fordelete", func(t *testing.T) {
		releaseName := mockReleaseName + "_delete"
		pubResp := testSuit.ConfigServer().UpsertAndReleaseConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFilePublishInfo{
			ReleaseName:        utils.NewStringValue(releaseName),
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

		delResp := testSuit.ConfigServer().DeleteConfigFileReleases(testSuit.DefaultCtx, []*config_manage.ConfigFileRelease{
			{
				Name:      utils.NewStringValue(releaseName),
				Namespace: utils.NewStringValue(mockNamespace),
				Group:     utils.NewStringValue(mockGroup),
				FileName:  utils.NewStringValue(mockFileName),
			},
		})
		// 删除成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), delResp.GetCode().GetValue(), delResp.GetInfo().GetValue())

		// 查询不到
		queryRsp := testSuit.ConfigServer().GetConfigFileReleases(testSuit.DefaultCtx, map[string]string{
			"name": releaseName,
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), queryRsp.GetInfo().GetValue())
		assert.Equal(t, 0, len(queryRsp.ConfigFileReleases))
		assert.Equal(t, 0, int(queryRsp.GetTotal().GetValue()))
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
		clientResp := testSuit.ConfigServer().GetConfigFileWithCache(testSuit.DefaultCtx, &config_manage.ClientConfigFileInfo{
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

// Test_GrayConfigFileRelease 测试配置灰度发布
func Test_GrayConfigFileRelease(t *testing.T) {
	testSuit := newConfigCenterTestSuit(t)

	var (
		mockNamespace       = "gray_mock_namespace"
		mockGroup           = "gray_mock_group"
		mockFileName        = "gray_mock_filename"
		mockReleaseName     = "gray_mock_release"
		mockContent         = "gray_mock_content"
		mockBetaReleaseName = "gray_mock_beta_release"
		mockNewContent      = "gray_mock_content_v2"
		mockClientIP        = "1.1.1.1"
	)

	t.Run("01-first-publish", func(t *testing.T) {
		resp := testSuit.ConfigServer().UpsertAndReleaseConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFilePublishInfo{
			Namespace:   utils.NewStringValue(mockNamespace),
			Group:       utils.NewStringValue(mockGroup),
			FileName:    utils.NewStringValue(mockFileName),
			ReleaseName: utils.NewStringValue(mockReleaseName),
			Content:     utils.NewStringValue(mockContent),
		})
		// 正常发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())

		resp = testSuit.ConfigServer().GetConfigFileRelease(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
			Name:      utils.NewStringValue(mockReleaseName),
		})
		// 正常发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
		// 正常发布成功
		assert.Equal(t, mockContent, resp.GetConfigFileRelease().GetContent().GetValue())
	})

	t.Run("02-gray_publish", func(t *testing.T) {
		resp := testSuit.ConfigServer().UpdateConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFile{
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			Name:      utils.NewStringValue(mockFileName),
			Content:   utils.NewStringValue(mockNewContent),
		})
		// 正常更新配置文件
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())

		// 发布灰度配置
		resp = testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
			Namespace:   utils.NewStringValue(mockNamespace),
			Group:       utils.NewStringValue(mockGroup),
			FileName:    utils.NewStringValue(mockFileName),
			Content:     utils.NewStringValue(mockNewContent),
			Name:        utils.NewStringValue(mockBetaReleaseName),
			ReleaseType: wrapperspb.String(model.ReleaseTypeGray),
			BetaLabels: []*apimodel.ClientLabel{
				{
					Key: model.ClientLabel_IP,
					Value: &apimodel.MatchString{
						Type:      apimodel.MatchString_EXACT,
						Value:     wrapperspb.String(mockClientIP),
						ValueType: apimodel.MatchString_TEXT,
					},
				},
			},
		})
		// 正常发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
		resp = testSuit.ConfigServer().GetConfigFileRelease(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
			Name:      utils.NewStringValue(mockBetaReleaseName),
		})
		// 正常发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.String())
		// 正常发布成功
		assert.Equal(t, mockNewContent, resp.GetConfigFileRelease().GetContent().GetValue())

		_ = testSuit.CacheMgr().TestUpdate()

		// 不带配置标签查询, 查不到处于灰度发布的配置
		clientRsp := testSuit.ConfigServer().GetConfigFileWithCache(testSuit.DefaultCtx, &config_manage.ClientConfigFileInfo{
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
		assert.Equal(t, mockContent, clientRsp.GetConfigFile().GetContent().GetValue())

		// 携带正确配置标签查询, 查到处于灰度发布的配置
		clientRsp = testSuit.ConfigServer().GetConfigFileWithCache(testSuit.DefaultCtx, &config_manage.ClientConfigFileInfo{
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
			Tags: []*config_manage.ConfigFileTag{
				{
					Key:   utils.NewStringValue(model.ClientLabel_IP),
					Value: utils.NewStringValue(mockClientIP),
				},
			},
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
		assert.Equal(t, mockNewContent, clientRsp.GetConfigFile().GetContent().GetValue())

		// 携带不正确配置标签查询, 查不到处于灰度发布的配置
		clientRsp = testSuit.ConfigServer().GetConfigFileWithCache(testSuit.DefaultCtx, &config_manage.ClientConfigFileInfo{
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
			Tags: []*config_manage.ConfigFileTag{
				{
					Key:   utils.NewStringValue(model.ClientLabel_IP),
					Value: utils.NewStringValue(mockClientIP + "2"),
				},
			},
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
		assert.Equal(t, mockContent, clientRsp.GetConfigFile().GetContent().GetValue())
	})

	// 测试存在灰度发布配置时, 不得发布新的配置文件
	t.Run("03-normal_publish_when_exist_gray", func(t *testing.T) {
		resp := testSuit.ConfigServer().UpsertAndReleaseConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFilePublishInfo{
			Namespace:   utils.NewStringValue(mockNamespace),
			Group:       utils.NewStringValue(mockGroup),
			FileName:    utils.NewStringValue(mockFileName),
			ReleaseName: utils.NewStringValue(mockReleaseName),
			Content:     utils.NewStringValue(mockContent),
		})
		// 正常发布成功
		assert.Equal(t, uint32(apimodel.Code_DataConflict), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
	})

	// 删除已发布的灰度配置，获取不到
	t.Run("04-delete_gray_release", func(t *testing.T) {
		resp := testSuit.ConfigServer().StopGrayConfigFileReleases(testSuit.DefaultCtx, []*config_manage.ConfigFileRelease{
			{
				Namespace: utils.NewStringValue(mockNamespace),
				Group:     utils.NewStringValue(mockGroup),
				FileName:  utils.NewStringValue(mockFileName),
			},
		})
		// 正常发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())

		_ = testSuit.CacheMgr().TestUpdate()

		// 不带配置标签查询, 查不到处于灰度发布的配置
		clientRsp := testSuit.ConfigServer().GetConfigFileWithCache(testSuit.DefaultCtx, &config_manage.ClientConfigFileInfo{
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
		assert.Equal(t, mockContent, clientRsp.GetConfigFile().GetContent().GetValue())

		// 携带正确配置标签查询, 查不到处于灰度发布的配置
		clientRsp = testSuit.ConfigServer().GetConfigFileWithCache(testSuit.DefaultCtx, &config_manage.ClientConfigFileInfo{
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
			Tags: []*config_manage.ConfigFileTag{
				{
					Key:   utils.NewStringValue(model.ClientLabel_IP),
					Value: utils.NewStringValue(mockClientIP),
				},
			},
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), resp.GetCode().GetValue(), resp.GetInfo().GetValue())
		assert.Equal(t, mockContent, clientRsp.GetConfigFile().GetContent().GetValue())

		// 配置发布成功
		pubResp := testSuit.ConfigServer().UpsertAndReleaseConfigFile(testSuit.DefaultCtx, &config_manage.ConfigFilePublishInfo{
			Namespace:   utils.NewStringValue(mockNamespace),
			Group:       utils.NewStringValue(mockGroup),
			FileName:    utils.NewStringValue(mockFileName),
			ReleaseName: utils.NewStringValue(mockReleaseName),
			Content:     utils.NewStringValue(mockContent),
		})
		// 正常发布成功
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), pubResp.GetCode().GetValue(), pubResp.GetInfo().GetValue())
	})
}

func TestServer_CasUpsertAndReleaseConfigFile(t *testing.T) {
	testSuit := newConfigCenterTestSuit(t)
	_ = testSuit

	var (
		mockNamespace   = "mock_namespace_cas"
		mockGroup       = "mock_group"
		mockFileName    = "mock_filename"
		mockReleaseName = "mock_release"
		mockContent     = "mock_content"
	)

	nsRsp := testSuit.NamespaceServer().CreateNamespace(testSuit.DefaultCtx, &apimodel.Namespace{
		Name: utils.NewStringValue(mockNamespace),
	})
	assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), nsRsp.GetCode().GetValue(), nsRsp.GetInfo().GetValue())

	t.Run("param_check", func(t *testing.T) {
		t.Run("invalid_namespace", func(t *testing.T) {
			queryRsp := testSuit.ConfigServer().GetConfigFileReleaseVersions(testSuit.DefaultCtx, map[string]string{
				"group":     mockGroup,
				"file_name": mockFileName,
			})
			assert.Equal(t, uint32(apimodel.Code_BadRequest), queryRsp.GetCode().GetValue())
			assert.Equal(t, "invalid namespace", queryRsp.GetInfo().GetValue())
		})
		t.Run("invalid_group", func(t *testing.T) {
			queryRsp := testSuit.ConfigServer().GetConfigFileReleaseVersions(testSuit.DefaultCtx, map[string]string{
				"namespace": mockNamespace,
				"file_name": mockFileName,
			})
			assert.Equal(t, uint32(apimodel.Code_BadRequest), queryRsp.GetCode().GetValue())
			assert.Equal(t, "invalid config group", queryRsp.GetInfo().GetValue())
		})
		t.Run("invalid_file_name", func(t *testing.T) {
			queryRsp := testSuit.ConfigServer().GetConfigFileReleaseVersions(testSuit.DefaultCtx, map[string]string{
				"group":     mockGroup,
				"namespace": mockNamespace,
			})
			assert.Equal(t, uint32(apimodel.Code_BadRequest), queryRsp.GetCode().GetValue())
			assert.Equal(t, "invalid config file name", queryRsp.GetInfo().GetValue())
		})
	})

	t.Run("publish_cas", func(t *testing.T) {
		// 第一次配置发布，就算带了 MD5，也是可以发布成功
		pubResp := testSuit.ConfigServer().CasUpsertAndReleaseConfigFileFromClient(testSuit.DefaultCtx, &config_manage.ConfigFilePublishInfo{
			Namespace:   utils.NewStringValue(mockNamespace),
			Group:       utils.NewStringValue(mockGroup),
			FileName:    utils.NewStringValue(mockFileName),
			ReleaseName: utils.NewStringValue(mockReleaseName),
			Content:     utils.NewStringValue(mockContent),
			Md5:         wrapperspb.String(config.CalMd5(mockContent)),
		})
		// 正常发布失败，数据冲突无法处理
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), pubResp.GetCode().GetValue(), pubResp.GetInfo().GetValue())

		// MD5 不一致，直接发布失败
		pubResp = testSuit.ConfigServer().CasUpsertAndReleaseConfigFileFromClient(testSuit.DefaultCtx, &config_manage.ConfigFilePublishInfo{
			Namespace:   utils.NewStringValue(mockNamespace),
			Group:       utils.NewStringValue(mockGroup),
			FileName:    utils.NewStringValue(mockFileName),
			ReleaseName: utils.NewStringValue(mockReleaseName),
			Content:     utils.NewStringValue(mockContent),
			Md5:         wrapperspb.String(config.CalMd5(time.Now().UTC().GoString())),
		})
		// 正常发布失败，数据冲突无法处理
		assert.Equal(t, uint32(apimodel.Code_DataConflict), pubResp.GetCode().GetValue(), pubResp.GetInfo().GetValue())

		// 获取下当前配置的 Release
		queryRsp := testSuit.ConfigServer().GetConfigFileRelease(testSuit.DefaultCtx, &config_manage.ConfigFileRelease{
			Namespace: utils.NewStringValue(mockNamespace),
			Group:     utils.NewStringValue(mockGroup),
			FileName:  utils.NewStringValue(mockFileName),
			Name:      wrapperspb.String(mockReleaseName),
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), queryRsp.GetCode().GetValue(), queryRsp.GetInfo().GetValue())
		assert.NotNil(t, queryRsp.GetConfigFileRelease())
		assert.Equal(t, config.CalMd5(mockContent), queryRsp.GetConfigFileRelease().GetMd5().GetValue())

		t.Run("md5_不匹配", func(t *testing.T) {
			pubResp := testSuit.ConfigServer().CasUpsertAndReleaseConfigFileFromClient(testSuit.DefaultCtx, &config_manage.ConfigFilePublishInfo{
				Namespace:   utils.NewStringValue(mockNamespace),
				Group:       utils.NewStringValue(mockGroup),
				FileName:    utils.NewStringValue(mockFileName),
				ReleaseName: utils.NewStringValue(mockReleaseName),
				Content:     utils.NewStringValue(mockContent),
				Md5:         wrapperspb.String(utils.NewUUID()),
			})
			// 正常发布失败，数据冲突无法处理
			assert.Equal(t, uint32(apimodel.Code_DataConflict), pubResp.GetCode().GetValue(), pubResp.GetInfo().GetValue())
		})

		t.Run("md5_匹配", func(t *testing.T) {
			pubResp := testSuit.ConfigServer().CasUpsertAndReleaseConfigFileFromClient(testSuit.DefaultCtx, &config_manage.ConfigFilePublishInfo{
				Namespace: utils.NewStringValue(mockNamespace),
				Group:     utils.NewStringValue(mockGroup),
				FileName:  utils.NewStringValue(mockFileName),
				Content:   utils.NewStringValue(mockContent),
				Md5:       wrapperspb.String(queryRsp.GetConfigFileRelease().GetMd5().GetValue()),
			})
			// 正常发布失败，数据冲突无法处理
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), pubResp.GetCode().GetValue(), pubResp.GetInfo().GetValue())
		})
	})
}
