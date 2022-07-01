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
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
	utils2 "github.com/polarismesh/polaris-server/config/utils"
)

// TestClientSetupAndFileNotExisted 测试客户端启动时（version=0），并且配置不存在的情况下拉取配置
func TestClientSetupAndFileNotExisted(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()

	fileInfo := &api.ClientConfigFileInfo{
		Namespace: &wrapperspb.StringValue{Value: testNamespace},
		Group:     &wrapperspb.StringValue{Value: testGroup},
		FileName:  &wrapperspb.StringValue{Value: testFile},
		Version:   &wrapperspb.UInt64Value{Value: 0},
	}

	rsp := testSuit.testService.GetConfigFileForClient(testSuit.defaultCtx, fileInfo)
	assert.Equal(t, uint32(api.NotFoundResource), rsp.Code.GetValue(), "GetConfigFileForClient must notfound")

	rsp2 := testSuit.testServer.CheckClientConfigFileByVersion(testSuit.defaultCtx, assembleDefaultClientConfigFile(0))
	assert.Equal(t, uint32(api.DataNoChange), rsp2.Code.GetValue(), "CheckClientConfigFileByVersion must nochange")
	assert.Nil(t, rsp2.ConfigFile)

	rsp3 := testSuit.testServer.CheckClientConfigFileByMd5(testSuit.defaultCtx, assembleDefaultClientConfigFile(0))
	assert.Equal(t, uint32(api.DataNoChange), rsp3.Code.GetValue())
	assert.Nil(t, rsp3.ConfigFile)
}

// TestClientSetupAndFileExisted 测试客户端启动时（version=0），并且配置存在的情况下拉取配置
func TestClientSetupAndFileExisted(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()
	// 创建并发布一个配置文件
	configFile := assembleConfigFile()
	rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

	rsp2 := testSuit.testService.PublishConfigFile(testSuit.defaultCtx, assembleConfigFileRelease(configFile))
	assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())

	fileInfo := &api.ClientConfigFileInfo{
		Namespace: &wrapperspb.StringValue{Value: testNamespace},
		Group:     &wrapperspb.StringValue{Value: testGroup},
		FileName:  &wrapperspb.StringValue{Value: testFile},
		Version:   &wrapperspb.UInt64Value{Value: 0},
	}

	// 拉取配置接口
	rsp3 := testSuit.testService.GetConfigFileForClient(testSuit.defaultCtx, fileInfo)
	assert.Equalf(t, api.ExecuteSuccess, rsp3.Code.GetValue(), "GetConfigFileForClient must success, acutal code : %d", rsp3.Code.GetValue())
	assert.NotNil(t, rsp3.ConfigFile)
	assert.Equal(t, uint64(1), rsp3.ConfigFile.Version.GetValue())
	assert.Equal(t, configFile.Content.GetValue(), rsp3.ConfigFile.Content.GetValue())
	assert.Equal(t, utils2.CalMd5(configFile.Content.GetValue()), rsp3.ConfigFile.Md5.GetValue())

	// 比较客户端配置是否落后
	rsp4 := testSuit.testServer.CheckClientConfigFileByVersion(testSuit.defaultCtx, assembleDefaultClientConfigFile(0))
	assert.Equal(t, api.ExecuteSuccess, rsp4.Code.GetValue())
	assert.NotNil(t, rsp4.ConfigFile)
	assert.Equal(t, utils2.CalMd5(configFile.Content.GetValue()), rsp4.ConfigFile.Md5.GetValue())

	rsp5 := testSuit.testServer.CheckClientConfigFileByMd5(testSuit.defaultCtx, assembleDefaultClientConfigFile(0))
	assert.Equal(t, api.ExecuteSuccess, rsp5.Code.GetValue())
	assert.NotNil(t, rsp5.ConfigFile)
	assert.Equal(t, uint64(1), rsp5.ConfigFile.Version.GetValue())
	assert.Equal(t, utils2.CalMd5(configFile.Content.GetValue()), rsp5.ConfigFile.Md5.GetValue())
}

// TestClientVersionBehindServer 测试客户端版本落后服务端
func TestClientVersionBehindServer(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()

	// 创建并连续发布5次
	configFile := assembleConfigFile()
	rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

	for i := 0; i < 5; i++ {
		configFile.Content = utils.NewStringValue("content" + strconv.Itoa(i))
		// 更新
		rsp2 := testSuit.testService.UpdateConfigFile(testSuit.defaultCtx, configFile)
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		// 发布
		rsp3 := testSuit.testService.PublishConfigFile(testSuit.defaultCtx, assembleConfigFileRelease(configFile))
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())
	}

	// 客户端版本号为4， 服务端由于连续发布5次，所以版本号为5
	clientVersion := uint64(4)
	latestContent := "content4"

	fileInfo := &api.ClientConfigFileInfo{
		Namespace: &wrapperspb.StringValue{Value: testNamespace},
		Group:     &wrapperspb.StringValue{Value: testGroup},
		FileName:  &wrapperspb.StringValue{Value: testFile},
		Version:   &wrapperspb.UInt64Value{Value: clientVersion},
	}

	// 拉取配置接口
	rsp4 := testSuit.testService.GetConfigFileForClient(testSuit.defaultCtx, fileInfo)
	assert.Equal(t, api.ExecuteSuccess, rsp4.Code.GetValue())
	assert.NotNil(t, rsp4.ConfigFile)
	assert.Equal(t, uint64(5), rsp4.ConfigFile.Version.GetValue())
	assert.Equal(t, latestContent, rsp4.ConfigFile.Content.GetValue())
	assert.Equal(t, utils2.CalMd5(latestContent), rsp4.ConfigFile.Md5.GetValue())

	// 比较客户端配置是否落后
	rsp5 := testSuit.testServer.CheckClientConfigFileByVersion(testSuit.defaultCtx, assembleDefaultClientConfigFile(clientVersion))
	assert.Equal(t, api.ExecuteSuccess, rsp5.Code.GetValue())
	assert.NotNil(t, rsp5.ConfigFile)
	assert.Equal(t, utils2.CalMd5(latestContent), rsp5.ConfigFile.Md5.GetValue())

	rsp6 := testSuit.testServer.CheckClientConfigFileByMd5(testSuit.defaultCtx, assembleDefaultClientConfigFile(clientVersion))
	assert.Equal(t, api.ExecuteSuccess, rsp6.Code.GetValue())
	assert.NotNil(t, rsp6.ConfigFile)
	assert.Equal(t, uint64(5), rsp6.ConfigFile.Version.GetValue())
	assert.Equal(t, utils2.CalMd5(latestContent), rsp6.ConfigFile.Md5.GetValue())
}

// TestWatchConfigFileAtFirstPublish 测试监听配置，并且第一次发布配置
func TestWatchConfigFileAtFirstPublish(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()

	// 创建并发布配置文件
	configFile := assembleConfigFile()

	t.Run("第一次订阅发布", func(t *testing.T) {

		received := make(chan uint64)

		watchConfigFiles := assembleDefaultClientConfigFile(0)
		clientId := "TestWatchConfigFileAtFirstPublish-first"

		defer func() {
			testSuit.testServer.WatchCenter().RemoveWatcher(clientId, watchConfigFiles)
		}()

		testSuit.testServer.WatchCenter().AddWatcher(clientId, watchConfigFiles, func(clientId string, rsp *api.ConfigClientResponse) bool {
			t.Logf("clientId=[%s] receive config publish msg", clientId)
			received <- rsp.ConfigFile.Version.GetValue()
			return true
		})

		rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
		assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

		rsp2 := testSuit.testService.PublishConfigFile(testSuit.defaultCtx, assembleConfigFileRelease(configFile))
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())

		receivedVersion := <-received
		assert.Equal(t, uint64(1), receivedVersion)
	})

	t.Run("第二次订阅发布", func(t *testing.T) {

		received := make(chan uint64)

		// 版本号由于发布过一次，所以是1
		watchConfigFiles := assembleDefaultClientConfigFile(1)

		clientId := "TestWatchConfigFileAtFirstPublish-second"

		testSuit.testServer.WatchCenter().AddWatcher(clientId, watchConfigFiles, func(clientId string, rsp *api.ConfigClientResponse) bool {
			t.Logf("clientId=[%s] receive config publish msg", clientId)
			received <- rsp.ConfigFile.Version.GetValue()
			return true
		})

		rsp3 := testSuit.testService.PublishConfigFile(testSuit.defaultCtx, assembleConfigFileRelease(configFile))
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())

		// 等待回调
		receivedVersion := <-received
		assert.Equal(t, uint64(2), receivedVersion)

		// 为了避免影响其它 case，删除订阅
		testSuit.testServer.WatchCenter().RemoveWatcher(clientId, watchConfigFiles)
	})
}

// Test10000ClientWatchConfigFile 测试 10000 个客户端同时监听配置变更，配置发布所有客户端都收到通知
func Test10000ClientWatchConfigFile(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()

	clientSize := 10000
	received := make(map[string]bool)
	receivedVersion := make(map[string]uint64)
	watchConfigFiles := assembleDefaultClientConfigFile(0)
	for i := 0; i < clientSize; i++ {
		clientId := fmt.Sprintf("Test10000ClientWatchConfigFile-client-id=%d", i)
		received[clientId] = false
		receivedVersion[clientId] = uint64(0)
		testSuit.testServer.WatchCenter().AddWatcher(clientId, watchConfigFiles, func(clientId string, rsp *api.ConfigClientResponse) bool {
			received[clientId] = true
			receivedVersion[clientId] = rsp.ConfigFile.Version.GetValue()
			return true
		})
	}

	// 创建并发布配置文件
	configFile := assembleConfigFile()
	rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

	rsp2 := testSuit.testService.PublishConfigFile(testSuit.defaultCtx, assembleConfigFileRelease(configFile))
	assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())

	// 等待回调
	time.Sleep(2000 * time.Millisecond)

	// 校验是否所有客户端都收到推送通知
	receivedCnt := 0
	for _, v := range received {
		if v {
			receivedCnt++
		}
	}
	assert.Equal(t, len(received), receivedCnt)

	receivedVerCnt := uint64(0)
	for _, v := range receivedVersion {
		receivedVerCnt += v
	}
	assert.Equal(t, uint64(len(receivedVersion)), uint64(receivedVerCnt))

	// 为了避免影响其它case，删除订阅
	for clientId := range received {
		testSuit.testServer.WatchCenter().RemoveWatcher(clientId, watchConfigFiles)
	}
}

// TestDeleteConfigFile 测试删除配置，删除配置会通知客户端，并且重新拉取配置会返回 NotFoundResourceConfigFile 状态码
func TestDeleteConfigFile(t *testing.T) {
	testSuit, err := newConfigCenterTest(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := testSuit.clearTestData(); err != nil {
			t.Fatal(err)
		}
	}()

	// 创建并发布一个配置文件
	configFile := assembleConfigFile()
	rsp := testSuit.testService.CreateConfigFile(testSuit.defaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

	rsp2 := testSuit.testService.PublishConfigFile(testSuit.defaultCtx, assembleConfigFileRelease(configFile))
	assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())

	time.Sleep(1200 * time.Millisecond)

	// 客户端订阅
	clientId := randomStr()
	received := make(chan uint64)
	watchConfigFiles := assembleDefaultClientConfigFile(0)

	t.Log("add config watcher")

	testSuit.testServer.WatchCenter().AddWatcher(clientId, watchConfigFiles, func(clientId string, rsp *api.ConfigClientResponse) bool {
		received <- rsp.ConfigFile.Version.GetValue()
		return true
	})

	// 删除配置文件
	t.Log("remove config file")
	rsp3 := testSuit.testService.DeleteConfigFile(testSuit.defaultCtx, testNamespace, testGroup, testFile, operator)
	assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())

	// 客户端收到推送通知
	t.Log("wait receive config change msg")
	receivedVersion := <-received
	assert.Equal(t, uint64(2), receivedVersion)

	fileInfo := &api.ClientConfigFileInfo{
		Namespace: &wrapperspb.StringValue{Value: testNamespace},
		Group:     &wrapperspb.StringValue{Value: testGroup},
		FileName:  &wrapperspb.StringValue{Value: testFile},
		Version:   &wrapperspb.UInt64Value{Value: 2},
	}

	// 重新拉取配置，获取不到配置文件
	rsp4 := testSuit.testService.GetConfigFileForClient(testSuit.defaultCtx, fileInfo)
	assert.Equal(t, uint32(api.NotFoundResource), rsp4.Code.GetValue())
}
