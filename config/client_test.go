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

package config_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
	testsuit "github.com/polarismesh/polaris/test/suit"
)

// TestClientSetupAndFileNotExisted 测试客户端启动时（version=0），并且配置不存在的情况下拉取配置
func TestClientSetupAndFileNotExisted(t *testing.T) {
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

	fileInfo := &apiconfig.ClientConfigFileInfo{
		Namespace: &wrapperspb.StringValue{Value: testNamespace},
		Group:     &wrapperspb.StringValue{Value: testGroup},
		FileName:  &wrapperspb.StringValue{Value: testFile},
		Version:   &wrapperspb.UInt64Value{Value: 0},
	}

	rsp := testSuit.ConfigServer().GetConfigFileWithCache(testSuit.DefaultCtx, fileInfo)
	assert.Equal(t, uint32(api.NotFoundResource), rsp.Code.GetValue(), "GetConfigFileWithCache must notfound")

	originSvr := testSuit.OriginConfigServer()
	rsp2, _ := originSvr.TestCheckClientConfigFile(testSuit.DefaultCtx, assembleDefaultClientConfigFile(0), config.TestCompareByVersion)
	assert.Equal(t, uint32(api.DataNoChange), rsp2.Code.GetValue(), "checkClientConfigFileByVersion must nochange")
	assert.Nil(t, rsp2.ConfigFile)
}

// TestClientSetupAndFileExisted 测试客户端启动时（version=0），并且配置存在的情况下拉取配置
func TestClientSetupAndFileExisted(t *testing.T) {
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

	// 创建并发布一个配置文件
	configFile := assembleConfigFile()
	rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue(), rsp.GetInfo().GetValue())

	rsp2 := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, assembleConfigFileRelease(configFile))
	assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue(), rsp2.GetInfo().GetValue())

	fileInfo := &apiconfig.ClientConfigFileInfo{
		Namespace: &wrapperspb.StringValue{Value: testNamespace},
		Group:     &wrapperspb.StringValue{Value: testGroup},
		FileName:  &wrapperspb.StringValue{Value: testFile},
		Version:   &wrapperspb.UInt64Value{Value: 0},
	}

	// 强制 Cache 层 load 到本地
	_ = testSuit.DiscoverServer().Cache().ConfigFile().Update()

	// 拉取配置接口
	rsp3 := testSuit.ConfigServer().GetConfigFileWithCache(testSuit.DefaultCtx, fileInfo)
	assert.Equalf(t, api.ExecuteSuccess, rsp3.Code.GetValue(), "GetConfigFileWithCache must success, acutal code : %d", rsp3.Code.GetValue())
	assert.NotNil(t, rsp3.ConfigFile)
	assert.Equal(t, uint64(1), rsp3.ConfigFile.Version.GetValue())
	assert.Equal(t, configFile.Content.GetValue(), rsp3.ConfigFile.Content.GetValue())

	// 比较客户端配置是否落后
	originSvr := testSuit.OriginConfigServer()
	rsp4, _ := originSvr.TestCheckClientConfigFile(testSuit.DefaultCtx, assembleDefaultClientConfigFile(0), config.TestCompareByVersion)
	assert.Equal(t, api.ExecuteSuccess, rsp4.Code.GetValue(), rsp4.GetInfo().GetValue())
	assert.NotNil(t, rsp4.ConfigFile)
}

// TestClientSetupAndCreateNewFile 测试客户端启动时（version=0），并且配置不存在的情况下创建新的配置
func TestClientSetupAndCreateNewFile(t *testing.T) {
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

	fileInfo := &apiconfig.ConfigFile{
		Namespace: &wrapperspb.StringValue{Value: testNamespace},
		Group:     &wrapperspb.StringValue{Value: testGroup},
		Name:      &wrapperspb.StringValue{Value: testFile},
		Content:   &wrapperspb.StringValue{Value: testContent},
	}

	rsp := testSuit.ConfigServer().CreateConfigFileFromClient(testSuit.DefaultCtx, fileInfo)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue(), rsp.GetInfo().GetValue())

	originSvr := testSuit.OriginConfigServer()
	rsp2, _ := originSvr.TestCheckClientConfigFile(testSuit.DefaultCtx, assembleDefaultClientConfigFile(0), config.TestCompareByVersion)
	assert.Equal(t, api.DataNoChange, rsp2.Code.GetValue(), rsp2.GetInfo().GetValue())
	assert.Nil(t, rsp2.ConfigFile)
}

// TestClientSetupAndCreateExistFile 测试客户端启动时（version=0），并且配置存在的情况下重复创建配置
func TestClientSetupAndCreateExistFile(t *testing.T) {
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

	fileInfo := &apiconfig.ConfigFile{
		Namespace: &wrapperspb.StringValue{Value: testNamespace},
		Group:     &wrapperspb.StringValue{Value: testGroup},
		Name:      &wrapperspb.StringValue{Value: testFile},
		Content:   &wrapperspb.StringValue{Value: testContent},
	}

	// 第一次创建
	rsp := testSuit.ConfigServer().CreateConfigFileFromClient(testSuit.DefaultCtx, fileInfo)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue(), "First CreateConfigFileFromClient must success")

	// 第二次创建
	rsp1 := testSuit.ConfigServer().CreateConfigFileFromClient(testSuit.DefaultCtx, fileInfo)
	assert.Equal(t, api.ExistedResource, rsp1.Code.GetValue(), "CreateConfigFileFromClient again must error")

	originSvr := testSuit.OriginConfigServer()
	rsp2, _ := originSvr.TestCheckClientConfigFile(testSuit.DefaultCtx, assembleDefaultClientConfigFile(0), config.TestCompareByVersion)
	assert.Equal(t, api.DataNoChange, rsp2.Code.GetValue(), "checkClientConfigFileByVersion must nochange")
	assert.Nil(t, rsp2.ConfigFile)
}

// TestClientSetupAndUpdateNewFile 测试客户端启动时（version=0），更新不存在的配置
func TestClientSetupAndUpdateNewFile(t *testing.T) {
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

	fileInfo := &apiconfig.ConfigFile{
		Namespace: &wrapperspb.StringValue{Value: testNamespace},
		Group:     &wrapperspb.StringValue{Value: testGroup},
		Name:      &wrapperspb.StringValue{Value: testFile},
		Content:   &wrapperspb.StringValue{Value: testContent},
	}

	// 直接更新
	rsp := testSuit.ConfigServer().UpdateConfigFileFromClient(testSuit.DefaultCtx, fileInfo)
	assert.Equal(t, api.NotFoundResource, rsp.Code.GetValue(), "UpdateConfigFileFromClient with no exist file must error")
}

// TestClientSetupAndUpdateExistFile 测试客户端启动时（version=0），更新存在的配置
func TestClientSetupAndUpdateExistFile(t *testing.T) {
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

	fileInfo := &apiconfig.ConfigFile{
		Namespace: &wrapperspb.StringValue{Value: testNamespace},
		Group:     &wrapperspb.StringValue{Value: testGroup},
		Name:      &wrapperspb.StringValue{Value: testFile},
		Content:   &wrapperspb.StringValue{Value: testContent},
	}

	// 先创建
	rsp := testSuit.ConfigServer().CreateConfigFileFromClient(testSuit.DefaultCtx, fileInfo)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue(), "CreateConfigFileFromClient must success")

	// 再更新
	fileInfo.Content = &wrapperspb.StringValue{Value: testContent + "1"}
	rsp1 := testSuit.ConfigServer().UpdateConfigFileFromClient(testSuit.DefaultCtx, fileInfo)
	assert.Equal(t, api.ExecuteSuccess, rsp1.Code.GetValue(), "UpdateConfigFileFromClient must success")

}

// TestClientSetupAndPublishNewFile 测试客户端启动时（version=0），发布不存在的配置
func TestClientSetupAndPublishNewFile(t *testing.T) {
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

	fileReleaseInfo := &apiconfig.ConfigFileRelease{
		Namespace: &wrapperspb.StringValue{Value: testNamespace},
		Group:     &wrapperspb.StringValue{Value: testGroup},
		FileName:  &wrapperspb.StringValue{Value: testFile},
		Content:   &wrapperspb.StringValue{Value: testContent},
	}

	// 直接发布
	rsp := testSuit.ConfigServer().PublishConfigFileFromClient(testSuit.DefaultCtx, fileReleaseInfo)
	assert.Equal(t, api.NotFoundNamespace, rsp.Code.GetValue(), "PublishConfigFileFromClient with no exist file must error")
}

// TestClientSetupAndPublishExistFile 测试客户端启动时（version=0），发布存在的配置
func TestClientSetupAndPublishExistFile(t *testing.T) {
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

	fileInfo := &apiconfig.ConfigFile{
		Namespace: &wrapperspb.StringValue{Value: testNamespace},
		Group:     &wrapperspb.StringValue{Value: testGroup},
		Name:      &wrapperspb.StringValue{Value: testFile},
		Content:   &wrapperspb.StringValue{Value: testContent},
	}

	// 先创建
	rsp := testSuit.ConfigServer().CreateConfigFileFromClient(testSuit.DefaultCtx, fileInfo)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue(), "CreateConfigFileFromClient must success")

	// 再发布
	fileReleaseInfo := &apiconfig.ConfigFileRelease{
		Namespace: &wrapperspb.StringValue{Value: testNamespace},
		Group:     &wrapperspb.StringValue{Value: testGroup},
		FileName:  &wrapperspb.StringValue{Value: testFile},
		Content:   &wrapperspb.StringValue{Value: testContent},
	}
	rsp1 := testSuit.ConfigServer().PublishConfigFileFromClient(testSuit.DefaultCtx, fileReleaseInfo)
	assert.Equal(t, api.ExecuteSuccess, rsp1.Code.GetValue(), "PublishConfigFileFromClient must success")

}

// TestClientVersionBehindServer 测试客户端版本落后服务端
func TestClientVersionBehindServer(t *testing.T) {
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

	// 创建并连续发布5次
	configFile := assembleConfigFile()
	rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

	for i := 0; i < 5; i++ {
		configFile.Content = utils.NewStringValue("content" + strconv.Itoa(i))
		// 更新
		rsp2 := testSuit.ConfigServer().UpdateConfigFile(testSuit.DefaultCtx, configFile)
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue(), rsp2.GetInfo().GetValue())
		// 发布
		rsp3 := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, assembleConfigFileRelease(configFile))
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue(), rsp3.GetInfo().GetValue())
	}

	// 客户端版本号为4， 服务端由于连续发布5次，所以版本号为5
	clientVersion := uint64(4)
	latestContent := "content4"

	fileInfo := &apiconfig.ClientConfigFileInfo{
		Namespace: &wrapperspb.StringValue{Value: configFile.GetNamespace().Value},
		Group:     &wrapperspb.StringValue{Value: configFile.GetGroup().Value},
		FileName:  &wrapperspb.StringValue{Value: configFile.GetName().Value},
		Version:   &wrapperspb.UInt64Value{Value: clientVersion},
	}

	_ = testSuit.DiscoverServer().Cache().ConfigFile().Update()

	// 拉取配置接口
	rsp4 := testSuit.ConfigServer().GetConfigFileWithCache(testSuit.DefaultCtx, fileInfo)
	assert.Equal(t, api.ExecuteSuccess, rsp4.Code.GetValue(), rsp4.GetInfo().GetValue())
	assert.NotNil(t, rsp4.ConfigFile)
	assert.Equal(t, uint64(5), rsp4.ConfigFile.Version.GetValue())
	assert.Equal(t, latestContent, rsp4.ConfigFile.Content.GetValue())

	svr := testSuit.OriginConfigServer()
	// 比较客户端配置是否落后
	rsp5, _ := svr.TestCheckClientConfigFile(testSuit.DefaultCtx, assembleDefaultClientConfigFile(clientVersion), config.TestCompareByVersion)
	assert.Equal(t, api.ExecuteSuccess, rsp5.Code.GetValue())
	assert.NotNil(t, rsp5.ConfigFile)
}

// TestWatchConfigFileAtFirstPublish 测试监听配置，并且第一次发布配置
func TestWatchConfigFileAtFirstPublish(t *testing.T) {
	testSuit := &ConfigCenterTest{}
	if err := testSuit.Initialize(func(cfg *testsuit.TestConfig) {
		for _, v := range cfg.Bootstrap.Logger {
			v.SetOutputLevel(commonlog.DebugLevel.Name())
		}
	}); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
		testSuit.Destroy()
	})

	// 创建并发布配置文件
	configFile := assembleConfigFile()

	t.Run("00_QuickCheck", func(t *testing.T) {
		curConfigFile := assembleConfigFile()
		curConfigFile.Namespace = utils.NewStringValue("QuickCheck")

		rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, curConfigFile)
		t.Log("create config file success")
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue(), rsp.GetInfo().GetValue())

		watchCenter := testSuit.OriginConfigServer().WatchCenter()

		t.Run("01_file_not_exist", func(t *testing.T) {
			_ = testSuit.DiscoverServer().Cache().ConfigFile().Update()

			tmpWatchCtx := config.BuildTimeoutWatchCtx(context.Background(),
				&apiconfig.ClientWatchConfigFileRequest{
					WatchFiles: []*apiconfig.ClientConfigFileInfo{
						{
							Namespace: curConfigFile.Namespace,
							Group:     curConfigFile.Group,
							FileName:  curConfigFile.Name,
						},
					},
				}, 0)(utils.NewUUID(), watchCenter.MatchBetaReleaseFile)

			rsp := watchCenter.CheckQuickResponseClient(tmpWatchCtx)
			assert.Nil(t, rsp, rsp)
		})

		t.Run("02_normal", func(t *testing.T) {
			// 发布一个正常的配置文件
			rsp2 := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, assembleConfigFileRelease(curConfigFile))
			t.Log("publish config file success")
			assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue(), rsp2.GetInfo().GetValue())

			_ = testSuit.DiscoverServer().Cache().ConfigFile().Update()

			req := &apiconfig.ClientWatchConfigFileRequest{
				WatchFiles: []*apiconfig.ClientConfigFileInfo{
					{
						Namespace: curConfigFile.Namespace,
						Group:     curConfigFile.Group,
						FileName:  curConfigFile.Name,
					},
				},
			}
			tmpWatchCtx := config.BuildTimeoutWatchCtx(context.Background(),
				req, 0)(utils.NewUUID(), watchCenter.MatchBetaReleaseFile)

			for i := range req.WatchFiles {
				tmpWatchCtx.AppendInterest(req.WatchFiles[i])
			}
			rsp := watchCenter.CheckQuickResponseClient(tmpWatchCtx)
			assert.NotNil(t, rsp, rsp)
			assert.True(t, api.IsSuccess(rsp), rsp.GetInfo().GetValue())
		})

		t.Run("03_gray", func(t *testing.T) {
			// 发布一个灰度配置文件
			curConfigFile.Content = utils.NewStringValue("gray polaris test")
			grayRelease := assembleConfigFileRelease(curConfigFile)
			grayRelease.ReleaseType = wrapperspb.String("gray")
			grayRelease.BetaLabels = []*apimodel.ClientLabel{
				&apimodel.ClientLabel{
					Key: "CLIENT_IP",
					Value: &apimodel.MatchString{
						Type:      apimodel.MatchString_EXACT,
						Value:     utils.NewStringValue("172.0.0.1"),
						ValueType: apimodel.MatchString_TEXT,
					},
				},
			}
			rsp2 := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, grayRelease)
			t.Log("publish config file success")
			assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue(), rsp2.GetInfo().GetValue())

			req := &apiconfig.ClientWatchConfigFileRequest{
				ClientIp: wrapperspb.String("172.0.0.1"),
				WatchFiles: []*apiconfig.ClientConfigFileInfo{
					{
						Namespace: curConfigFile.Namespace,
						Group:     curConfigFile.Group,
						FileName:  curConfigFile.Name,
					},
				},
			}
			tmpWatchCtx := config.BuildTimeoutWatchCtx(context.Background(),
				req, 0)(utils.NewUUID(), watchCenter.MatchBetaReleaseFile)

			for i := range req.WatchFiles {
				tmpWatchCtx.AppendInterest(req.WatchFiles[i])
			}
			rsp := watchCenter.CheckQuickResponseClient(tmpWatchCtx)
			assert.NotNil(t, rsp, rsp)
			assert.True(t, api.IsSuccess(rsp), rsp.GetInfo().GetValue())
		})
	})

	t.Run("01_first_watch", func(t *testing.T) {
		watchConfigFiles := assembleDefaultClientConfigFile(0)
		clientId := "TestWatchConfigFileAtFirstPublish-first"

		defer func() {
			testSuit.OriginConfigServer().WatchCenter().RemoveWatcher(clientId, watchConfigFiles)
		}()

		watchCtx := testSuit.OriginConfigServer().WatchCenter().AddWatcher(clientId, watchConfigFiles,
			config.BuildTimeoutWatchCtx(context.Background(), &apiconfig.ClientWatchConfigFileRequest{}, 30*time.Second))
		assert.NotNil(t, watchCtx)

		rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, configFile)
		t.Log("create config file success")
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue(), rsp.GetInfo().GetValue())

		rsp2 := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, assembleConfigFileRelease(configFile))
		t.Log("publish config file success")
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue(), rsp2.GetInfo().GetValue())

		saveData, err := testSuit.Storage.GetConfigFileActiveRelease(&model.ConfigFileKey{
			Name:      configFile.GetName().GetValue(),
			Namespace: configFile.GetNamespace().GetValue(),
			Group:     configFile.GetGroup().GetValue(),
		})
		assert.NoError(t, err)
		assert.Equal(t, uint64(1), saveData.Version)
		assert.Equal(t, configFile.GetContent().GetValue(), saveData.Content)

		notifyRsp, err := (watchCtx.(*config.LongPollWatchContext)).GetNotifieResultWithTime(10 * time.Second)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("clientId=[%s] receive config publish msg", clientId)
		receivedVersion := notifyRsp.GetConfigFile().GetVersion().GetValue()
		assert.Equal(t, uint64(1), receivedVersion)
	})

	t.Run("02_second_watch", func(t *testing.T) {
		// 版本号由于发布过一次，所以是1
		watchConfigFiles := assembleDefaultClientConfigFile(1)

		clientId := "TestWatchConfigFileAtFirstPublish-second"

		watchCtx := testSuit.OriginConfigServer().WatchCenter().AddWatcher(clientId, watchConfigFiles,
			config.BuildTimeoutWatchCtx(context.Background(), &apiconfig.ClientWatchConfigFileRequest{}, 30*time.Second))
		assert.NotNil(t, watchCtx)

		rsp3 := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, assembleConfigFileRelease(configFile))
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())

		// 等待回调
		notifyRsp, err := (watchCtx.(*config.LongPollWatchContext)).GetNotifieResultWithTime(10 * time.Second)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("clientId=[%s] receive config publish msg", clientId)
		receivedVersion := notifyRsp.ConfigFile.Version.GetValue()
		assert.Equal(t, uint64(2), receivedVersion)

		// 为了避免影响其它 case，删除订阅
		testSuit.OriginConfigServer().WatchCenter().RemoveWatcher(clientId, watchConfigFiles)
	})

	t.Run("03_clean_invalid_client", func(t *testing.T) {
		watchConfigFiles := assembleDefaultClientConfigFile(1)
		for i := range watchConfigFiles {
			watchConfigFiles[i].Namespace = utils.NewStringValue("03_clean_invalid_client")
		}

		// watchCtx 默认为 1s 超时
		watchCtx := testSuit.OriginConfigServer().WatchCenter().AddWatcher(utils.NewUUID(), watchConfigFiles,
			config.BuildTimeoutWatchCtx(context.Background(), &apiconfig.ClientWatchConfigFileRequest{}, time.Second))
		assert.NotNil(t, watchCtx)

		time.Sleep(10 * time.Second)

		//
		ret := watchCtx.(*config.LongPollWatchContext).GetNotifieResult()
		assert.Equal(t, uint32(apimodel.Code_DataNoChange), ret.GetCode().GetValue())
	})
}

// Test10000ClientWatchConfigFile 测试 10000 个客户端同时监听配置变更，配置发布所有客户端都收到通知
func TestManyClientWatchConfigFile(t *testing.T) {
	testSuit := newConfigCenterTestSuit(t)

	clientSize := 100
	received := utils.NewSyncMap[string, bool]()
	receivedVersion := utils.NewSyncMap[string, uint64]()
	watchConfigFiles := assembleDefaultClientConfigFile(0)

	for i := 0; i < clientSize; i++ {
		clientId := fmt.Sprintf("Test10000ClientWatchConfigFile-client-id=%d", i)
		received.Store(clientId, false)
		receivedVersion.Store(clientId, uint64(0))
		watchCtx := testSuit.OriginConfigServer().WatchCenter().AddWatcher(clientId, watchConfigFiles,
			config.BuildTimeoutWatchCtx(context.Background(), &apiconfig.ClientWatchConfigFileRequest{}, 30*time.Second))
		assert.NotNil(t, watchCtx)
		go func() {
			notifyRsp := (watchCtx.(*config.LongPollWatchContext)).GetNotifieResult()
			received.Store(clientId, true)
			receivedVersion.Store(clientId, notifyRsp.ConfigFile.Version.GetValue())
		}()
	}

	// 创建并发布配置文件
	configFile := assembleConfigFile()
	rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

	rsp2 := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, assembleConfigFileRelease(configFile))
	assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())

	// 等待回调
	time.Sleep(2000 * time.Millisecond)

	// 校验是否所有客户端都收到推送通知
	receivedCnt := 0
	received.ReadRange(func(key string, val bool) {
		if val {
			receivedCnt++
		}
	})
	assert.Equal(t, received.Len(), receivedCnt)

	activeQuery := assembleConfigFileRelease(configFile)
	activeQuery.Name = nil
	activeRsp := testSuit.ConfigServer().GetConfigFileRelease(testSuit.DefaultCtx, activeQuery)
	assert.Equal(t, apimodel.Code_ExecuteSuccess, apimodel.Code(activeRsp.Code.Value), activeRsp.Info.Value)

	receivedVerCnt := 0
	receivedVersion.ReadRange(func(key string, val uint64) {
		if val == activeRsp.ConfigFileRelease.Version.Value {
			receivedVerCnt++
		}
	})
	assert.Equal(t, receivedVersion.Len(), receivedVerCnt)

	// 为了避免影响其它case，删除订阅
	received.ReadRange(func(clientId string, val bool) {
		testSuit.OriginConfigServer().WatchCenter().RemoveWatcher(clientId, watchConfigFiles)
	})
}

// TestDeleteConfigFile 测试删除配置，删除配置会通知客户端，并且重新拉取配置会返回 NotFoundResourceConfigFile 状态码
func TestDeleteConfigFile(t *testing.T) {
	testSuit := newConfigCenterTestSuit(t)

	newMockNs := "TestDeleteConfigFile"

	// 创建并发布一个配置文件
	configFile := assembleConfigFile()
	configFile.Namespace = wrapperspb.String(newMockNs)

	rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

	rsp2 := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, assembleConfigFileRelease(configFile))
	assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
	_ = testSuit.CacheMgr().TestUpdate()

	activeRelease := testSuit.CacheMgr().ConfigFile().GetActiveRelease(configFile.Namespace.Value,
		configFile.Group.Value, configFile.Name.Value)
	assert.NotNil(t, activeRelease)

	// 客户端订阅
	watchConfigFiles := assembleDefaultClientConfigFile(activeRelease.Version)
	for i := range watchConfigFiles {
		watchConfigFiles[i].Namespace = wrapperspb.String(newMockNs)
	}

	// 删除配置文件
	t.Log("remove config file")
	rsp3 := testSuit.ConfigServer().BatchDeleteConfigFile(testSuit.DefaultCtx, []*apiconfig.ConfigFile{
		&apiconfig.ConfigFile{
			Namespace: utils.NewStringValue(newMockNs),
			Group:     utils.NewStringValue(testGroup),
			Name:      utils.NewStringValue(testFile),
		},
	})
	assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())
	_ = testSuit.CacheMgr().TestUpdate()

	fileInfo := &apiconfig.ClientConfigFileInfo{
		Namespace: &wrapperspb.StringValue{Value: newMockNs},
		Group:     &wrapperspb.StringValue{Value: testGroup},
		FileName:  &wrapperspb.StringValue{Value: testFile},
	}

	// 重新拉取配置，获取不到配置文件
	rsp4 := testSuit.ConfigServer().GetConfigFileWithCache(testSuit.DefaultCtx, fileInfo)
	assert.Equal(t, rsp4.Code.GetValue(), api.NotFoundResource, rsp4.GetInfo().GetValue())
}

// TestServer_GetConfigFileNamesWithCache
func TestServer_GetConfigFileNamesWithCache(t *testing.T) {
	testSuit := newConfigCenterTestSuit(t)

	mockFiles := make(map[string][]*apiconfig.ConfigFile)
	groupTotal := 2
	fileTotal := 10
	for i := 0; i < groupTotal; i++ {
		groupName := fmt.Sprintf("group-%d", i)
		mockFiles[groupName] = make([]*apiconfig.ConfigFile, 0, fileTotal)
		for j := 0; j < fileTotal; j++ {
			item := &apiconfig.ConfigFile{
				Namespace: wrapperspb.String(testNamespace),
				Group:     wrapperspb.String(groupName),
				Name:      wrapperspb.String(fmt.Sprintf("file-%d", j)),
				Content:   wrapperspb.String(fmt.Sprintf("%d-%d", i, j)),
			}
			mockFiles[groupName] = append(mockFiles[groupName], item)
			rsp := testSuit.ConfigServer().CreateConfigFile(testSuit.DefaultCtx, item)
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), rsp.Code.GetValue())
		}
	}
	t.Cleanup(func() {
		for k := range mockFiles {
			items := mockFiles[k]
			for _, item := range items {
				testSuit.ConfigServer().DeleteConfigFile(testSuit.DefaultCtx, item)
			}
		}
	})

	_ = testSuit.OriginConfigServer().CacheManager()
	_ = testSuit.OriginConfigServer().GroupCache()
	_ = testSuit.OriginConfigServer().FileCache()
	_ = testSuit.OriginConfigServer().CryptoManager()

	t.Run("bad-request", func(t *testing.T) {
		rsp := testSuit.ConfigServer().GetConfigFileNamesWithCache(testSuit.DefaultCtx, &apiconfig.ConfigFileGroupRequest{
			ConfigFileGroup: &apiconfig.ConfigFileGroup{
				Namespace: utils.NewStringValue(""),
				Name:      utils.NewStringValue("group-0"),
			},
		})
		assert.Equal(t, uint32(apimodel.Code_BadRequest), rsp.Code.GetValue())

		rsp = testSuit.ConfigServer().GetConfigFileNamesWithCache(testSuit.DefaultCtx, &apiconfig.ConfigFileGroupRequest{
			ConfigFileGroup: &apiconfig.ConfigFileGroup{
				Namespace: utils.NewStringValue(""),
				Name:      utils.NewStringValue(""),
			},
		})
		assert.Equal(t, uint32(apimodel.Code_BadRequest), rsp.Code.GetValue())

		rsp = testSuit.ConfigServer().GetConfigFileNamesWithCache(testSuit.DefaultCtx, &apiconfig.ConfigFileGroupRequest{
			ConfigFileGroup: &apiconfig.ConfigFileGroup{
				Namespace: utils.NewStringValue("mock-ns"),
				Name:      utils.NewStringValue(""),
			},
		})
		assert.Equal(t, uint32(apimodel.Code_BadRequest), rsp.Code.GetValue())
	})

	t.Run("no-publish-file", func(t *testing.T) {
		rsp := testSuit.ConfigServer().GetConfigFileNamesWithCache(testSuit.DefaultCtx, &apiconfig.ConfigFileGroupRequest{
			ConfigFileGroup: &apiconfig.ConfigFileGroup{
				Namespace: utils.NewStringValue(testNamespace),
				Name:      utils.NewStringValue("group-0"),
			},
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), rsp.Code.GetValue(), rsp.Info.Value)
		assert.True(t, len(rsp.ConfigFileInfos) == 0)
	})

	t.Run("publish-file", func(t *testing.T) {
		for _, item := range mockFiles["group-0"] {
			rsp := testSuit.ConfigServer().PublishConfigFile(testSuit.DefaultCtx, &apiconfig.ConfigFileRelease{
				Namespace: item.Namespace,
				Group:     item.Group,
				FileName:  item.Name,
			})
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), rsp.Code.GetValue())
		}

		_ = testSuit.CacheMgr().TestUpdate()

		t.Run("revision-fetch", func(t *testing.T) {
			rsp := testSuit.ConfigServer().GetConfigFileNamesWithCache(testSuit.DefaultCtx, &apiconfig.ConfigFileGroupRequest{
				ConfigFileGroup: &apiconfig.ConfigFileGroup{
					Namespace: utils.NewStringValue(testNamespace),
					Name:      utils.NewStringValue("group-0"),
				},
			})

			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), rsp.Code.GetValue(), rsp.String())
			assert.True(t, len(rsp.ConfigFileInfos) == fileTotal, rsp.String())

			secondRsp := testSuit.ConfigServer().GetConfigFileNamesWithCache(testSuit.DefaultCtx, &apiconfig.ConfigFileGroupRequest{
				Revision: wrapperspb.String(rsp.GetRevision().GetValue()),
				ConfigFileGroup: &apiconfig.ConfigFileGroup{
					Namespace: utils.NewStringValue(testNamespace),
					Name:      utils.NewStringValue("group-0"),
				},
			})

			assert.Equal(t, uint32(apimodel.Code_DataNoChange), secondRsp.Code.GetValue())
			assert.True(t, len(secondRsp.ConfigFileInfos) == 0)
		})

	})
}

// TestServer_GetConfigGroupsWithCache
func TestServer_GetConfigGroupsWithCache(t *testing.T) {
	testSuit := newConfigCenterTestSuit(t)

	mockFiles := make(map[string][]*apiconfig.ConfigFileGroup)
	nsTotal := 2
	groupTotal := 10
	for i := 0; i < nsTotal; i++ {
		nsName := fmt.Sprintf("ns-%d", i)
		mockFiles[nsName] = make([]*apiconfig.ConfigFileGroup, 0, groupTotal)
		for j := 0; j < groupTotal; j++ {
			item := &apiconfig.ConfigFileGroup{
				Namespace: wrapperspb.String(nsName),
				Name:      wrapperspb.String(fmt.Sprintf("group-%d", j)),
			}
			mockFiles[nsName] = append(mockFiles[nsName], item)
			rsp := testSuit.ConfigServer().CreateConfigFileGroup(testSuit.DefaultCtx, item)
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), rsp.Code.GetValue(), rsp.GetInfo().GetValue())
		}
	}
	t.Cleanup(func() {
		for k := range mockFiles {
			testSuit.NamespaceServer().DeleteNamespace(testSuit.DefaultCtx, &apimodel.Namespace{
				Name: wrapperspb.String(k),
			})
			items := mockFiles[k]
			for _, item := range items {
				testSuit.ConfigServer().DeleteConfigFileGroup(testSuit.DefaultCtx, item.GetNamespace().Value, item.GetName().Value)
			}
		}
	})

	_ = testSuit.CacheMgr().TestUpdate()

	t.Run("case-1", func(t *testing.T) {
		rsp := testSuit.ConfigServer().GetConfigGroupsWithCache(testSuit.DefaultCtx, &apiconfig.ClientConfigFileInfo{
			Namespace: wrapperspb.String("ns-0"),
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), rsp.Code, rsp.Info)
		assert.True(t, len(rsp.ConfigFileGroups) == groupTotal)

		// 同一个 revision 查询
		rsp = testSuit.ConfigServer().GetConfigGroupsWithCache(testSuit.DefaultCtx, &apiconfig.ClientConfigFileInfo{
			Namespace: wrapperspb.String("ns-0"),
			Md5:       wrapperspb.String(rsp.GetRevision()),
		})
		assert.Equal(t, uint32(apimodel.Code_DataNoChange), rsp.Code, rsp.Info)

		// 删除其中一个配置分组后查询
		groups := mockFiles["ns-0"]
		for i := 0; i < 2; i++ {
			delRsp := testSuit.ConfigServer().DeleteConfigFileGroup(testSuit.DefaultCtx, "ns-0", groups[i].Name.Value)
			assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), delRsp.Code.Value, delRsp.Info.Value)
		}

		_ = testSuit.CacheMgr().TestUpdate()

		rsp = testSuit.ConfigServer().GetConfigGroupsWithCache(testSuit.DefaultCtx, &apiconfig.ClientConfigFileInfo{
			Namespace: wrapperspb.String("ns-0"),
		})
		assert.Equal(t, uint32(apimodel.Code_ExecuteSuccess), rsp.Code, rsp.Info)
		assert.True(t, len(rsp.ConfigFileGroups) == groupTotal-2)
	})
}
