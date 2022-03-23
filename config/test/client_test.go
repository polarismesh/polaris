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

package test

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
	utils2 "github.com/polarismesh/polaris-server/config/utils"
)

// TestClientSetupAndFileNotExisted 测试客户端启动时（version=0），并且配置不存在的情况下拉取配置
func TestClientSetupAndFileNotExisted(t *testing.T) {
	if err := clearTestData(); err != nil {
		t.FailNow()
	}
	rsp := configService.Service().GetConfigFileForClient(defaultCtx, testNamespace, testGroup, testFile, 0)
	assert.Equal(t, uint32(api.NotFoundResource), rsp.Code.GetValue())

	rsp2 := configService.Service().CheckClientConfigFileByVersion(defaultCtx, assembleDefaultClientConfigFile(0))
	assert.Equal(t, uint32(api.DataNoChange), rsp2.Code.GetValue())
	assert.Nil(t, rsp2.ConfigFile)

	rsp3 := configService.Service().CheckClientConfigFileByMd5(defaultCtx, assembleDefaultClientConfigFile(0))
	assert.Equal(t, uint32(api.DataNoChange), rsp3.Code.GetValue())
	assert.Nil(t, rsp3.ConfigFile)
}

// TestClientSetupAndFileExisted 测试客户端启动时（version=0），并且配置存在的情况下拉取配置
func TestClientSetupAndFileExisted(t *testing.T) {
	if err := clearTestData(); err != nil {
		t.FailNow()
	}
	// 创建并发布一个配置文件
	configFile := assembleConfigFile()
	rsp := configService.Service().CreateConfigFile(defaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

	rsp2 := configService.Service().PublishConfigFile(defaultCtx, assembleConfigFileRelease(configFile))
	assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())

	// 拉取配置接口
	rsp3 := configService.Service().GetConfigFileForClient(defaultCtx, testNamespace, testGroup, testFile, 0)
	assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())
	assert.NotNil(t, rsp3.ConfigFile)
	assert.Equal(t, uint64(1), rsp3.ConfigFile.Version.GetValue())
	assert.Equal(t, configFile.Content.GetValue(), rsp3.ConfigFile.Content.GetValue())
	assert.Equal(t, utils2.CalMd5(configFile.Content.GetValue()), rsp3.ConfigFile.Md5.GetValue())

	// 比较客户端配置是否落后
	rsp4 := configService.Service().CheckClientConfigFileByVersion(defaultCtx, assembleDefaultClientConfigFile(0))
	assert.Equal(t, api.ExecuteSuccess, rsp4.Code.GetValue())
	assert.NotNil(t, rsp4.ConfigFile)
	assert.Equal(t, utils2.CalMd5(configFile.Content.GetValue()), rsp4.ConfigFile.Md5.GetValue())

	rsp5 := configService.Service().CheckClientConfigFileByMd5(defaultCtx, assembleDefaultClientConfigFile(0))
	assert.Equal(t, api.ExecuteSuccess, rsp5.Code.GetValue())
	assert.NotNil(t, rsp5.ConfigFile)
	assert.Equal(t, uint64(1), rsp5.ConfigFile.Version.GetValue())
	assert.Equal(t, utils2.CalMd5(configFile.Content.GetValue()), rsp5.ConfigFile.Md5.GetValue())
}

// TestClientVersionBehindServer 测试客户端版本落后服务端
func TestClientVersionBehindServer(t *testing.T) {
	if err := clearTestData(); err != nil {
		t.FailNow()
	}

	// 创建并连续发布5次
	configFile := assembleConfigFile()
	rsp := configService.Service().CreateConfigFile(defaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

	for i := 0; i < 5; i++ {
		configFile.Content = utils.NewStringValue("content" + strconv.Itoa(i))
		// 更新
		rsp2 := configService.Service().UpdateConfigFile(defaultCtx, configFile)
		assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())
		// 发布
		rsp3 := configService.Service().PublishConfigFile(defaultCtx, assembleConfigFileRelease(configFile))
		assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())
	}

	// 客户端版本号为4， 服务端由于连续发布5次，所以版本号为5
	clientVersion := uint64(4)
	latestContent := "content4"
	// 拉取配置接口
	rsp4 := configService.Service().GetConfigFileForClient(defaultCtx, testNamespace, testGroup, testFile, clientVersion)
	assert.Equal(t, api.ExecuteSuccess, rsp4.Code.GetValue())
	assert.NotNil(t, rsp4.ConfigFile)
	assert.Equal(t, uint64(5), rsp4.ConfigFile.Version.GetValue())
	assert.Equal(t, latestContent, rsp4.ConfigFile.Content.GetValue())
	assert.Equal(t, utils2.CalMd5(latestContent), rsp4.ConfigFile.Md5.GetValue())

	// 比较客户端配置是否落后
	rsp5 := configService.Service().CheckClientConfigFileByVersion(defaultCtx, assembleDefaultClientConfigFile(clientVersion))
	assert.Equal(t, api.ExecuteSuccess, rsp5.Code.GetValue())
	assert.NotNil(t, rsp5.ConfigFile)
	assert.Equal(t, utils2.CalMd5(latestContent), rsp5.ConfigFile.Md5.GetValue())

	rsp6 := configService.Service().CheckClientConfigFileByMd5(defaultCtx, assembleDefaultClientConfigFile(clientVersion))
	assert.Equal(t, api.ExecuteSuccess, rsp6.Code.GetValue())
	assert.NotNil(t, rsp6.ConfigFile)
	assert.Equal(t, uint64(5), rsp6.ConfigFile.Version.GetValue())
	assert.Equal(t, utils2.CalMd5(latestContent), rsp6.ConfigFile.Md5.GetValue())
}

// TestWatchConfigFileAtFirstPublish 测试监听配置，并且第一次发布配置
func TestWatchConfigFileAtFirstPublish(t *testing.T) {
	if err := clearTestData(); err != nil {
		t.FailNow()
	}

	clientId := randomStr()
	watchConfigFiles := assembleDefaultClientConfigFile(0)
	var received bool
	var receivedVersion uint64
	configService.WatchCenter().AddWatcher(clientId, watchConfigFiles, func(clientId string, rsp *api.ConfigClientResponse) bool {
		received = true
		receivedVersion = rsp.ConfigFile.Version.GetValue()
		return true
	})

	// 创建并发布配置文件
	configFile := assembleConfigFile()
	rsp := configService.Service().CreateConfigFile(defaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

	rsp2 := configService.Service().PublishConfigFile(defaultCtx, assembleConfigFileRelease(configFile))
	assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())

	// 等待回调
	time.Sleep(1200 * time.Millisecond)

	assert.True(t, received)
	assert.Equal(t, uint64(1), receivedVersion)

	// 第二次订阅发布
	configService.WatchCenter().RemoveWatcher(clientId, watchConfigFiles)

	// 版本号由于发布过一次，所以是1
	watchConfigFiles = assembleDefaultClientConfigFile(1)
	received = false
	configService.WatchCenter().AddWatcher(clientId, watchConfigFiles, func(clientId string, rsp *api.ConfigClientResponse) bool {
		received = true
		receivedVersion = rsp.ConfigFile.Version.GetValue()
		return true
	})

	rsp3 := configService.Service().PublishConfigFile(defaultCtx, assembleConfigFileRelease(configFile))
	assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())

	// 等待回调
	time.Sleep(1200 * time.Millisecond)

	assert.True(t, received)
	assert.Equal(t, uint64(2), receivedVersion)

	// 为了避免影响其它 case，删除订阅
	configService.WatchCenter().RemoveWatcher(clientId, watchConfigFiles)
}

// Test10000ClientWatchConfigFile 测试 10000 个客户端同时监听配置变更，配置发布所有客户端都收到通知
func Test10000ClientWatchConfigFile(t *testing.T) {
	if err := clearTestData(); err != nil {
		t.FailNow()
	}

	clientSize := 10000
	received := make(map[string]bool)
	receivedVersion := make(map[string]uint64)
	watchConfigFiles := assembleDefaultClientConfigFile(0)
	for i := 0; i < clientSize; i++ {
		clientId := randomStr()
		received[clientId] = false
		receivedVersion[clientId] = uint64(0)
		configService.WatchCenter().AddWatcher(clientId, watchConfigFiles, func(clientId string, rsp *api.ConfigClientResponse) bool {
			received[clientId] = true
			receivedVersion[clientId] = rsp.ConfigFile.Version.GetValue()
			return true
		})
	}

	// 创建并发布配置文件
	configFile := assembleConfigFile()
	rsp := configService.Service().CreateConfigFile(defaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

	rsp2 := configService.Service().PublishConfigFile(defaultCtx, assembleConfigFileRelease(configFile))
	assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())

	// 等待回调
	time.Sleep(2000 * time.Millisecond)

	// 校验是否所有客户端都收到推送通知
	for _, v := range received {
		assert.True(t, v)
	}

	for _, v := range receivedVersion {
		assert.Equal(t, uint64(1), v)
	}

	// 为了避免影响其它case，删除订阅
	for clientId, _ := range received {
		configService.WatchCenter().RemoveWatcher(clientId, watchConfigFiles)
	}
}

// TestDeleteConfigFile 测试删除配置，删除配置会通知客户端，并且重新拉取配置会返回 NotFoundResourceConfigFile 状态码
func TestDeleteConfigFile(t *testing.T) {
	if err := clearTestData(); err != nil {
		t.FailNow()
	}
	// 创建并发布一个配置文件
	configFile := assembleConfigFile()
	rsp := configService.Service().CreateConfigFile(defaultCtx, configFile)
	assert.Equal(t, api.ExecuteSuccess, rsp.Code.GetValue())

	rsp2 := configService.Service().PublishConfigFile(defaultCtx, assembleConfigFileRelease(configFile))
	assert.Equal(t, api.ExecuteSuccess, rsp2.Code.GetValue())

	time.Sleep(1200 * time.Millisecond)

	// 客户端订阅
	clientId := randomStr()
	var received bool
	var receivedVersion uint64
	watchConfigFiles := assembleDefaultClientConfigFile(0)
	configService.WatchCenter().AddWatcher(clientId, watchConfigFiles, func(clientId string, rsp *api.ConfigClientResponse) bool {
		received = true
		receivedVersion = rsp.ConfigFile.Version.GetValue()
		return true
	})

	// 删除配置文件
	rsp3 := configService.Service().DeleteConfigFile(defaultCtx, testNamespace, testGroup, testFile, operator)
	assert.Equal(t, api.ExecuteSuccess, rsp3.Code.GetValue())

	time.Sleep(1200 * time.Millisecond)

	// 客户端收到推送通知
	assert.True(t, received)
	assert.Equal(t, uint64(2), receivedVersion)

	// 重新拉取配置，获取不到配置文件
	rsp4 := configService.Service().GetConfigFileForClient(defaultCtx, testNamespace, testGroup, testFile, 2)
	assert.Equal(t, uint32(api.NotFoundResource), rsp4.Code.GetValue())
}
