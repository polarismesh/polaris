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

package cache

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store/mock"
)

var (
	testNamespace = "testNamespace"
	testGroup     = "testGroup"
	testFile      = "testFile"
)

// TestGetAndRemoveAndReloadConfigFile 组合测试获取、删除、重载缓存
func TestGetAndRemoveAndReloadConfigFile(t *testing.T) {
	control, mockedStorage, fileCache := newConfigFileMockedCache(t)
	fileCache.Clear()
	defer control.Finish()

	// Mock 数据
	configFile := assembleConfigFile()
	configFileRelease := assembleConfigFileRelease(configFile)

	// 前三次调用返回一个值，第四次调用返回另外一个值，默认更新
	mockedStorage.EXPECT().GetConfigFileRelease(nil, testNamespace, testGroup, testFile).Return(configFileRelease, nil).Times(3)

	for i := 0; i < 100; i++ {
		go func() {
			entry, _ := fileCache.GetOrLoadIfAbsent(testNamespace, testGroup, testFile)
			assert.True(t, !entry.Empty)
		}()
	}

	time.Sleep(100 * time.Millisecond)

	// 删除缓存
	fileCache.Remove(testNamespace, testGroup, testFile)

	for i := 0; i < 100; i++ {
		go func() {
			entry, _ := fileCache.GetOrLoadIfAbsent(testNamespace, testGroup, testFile)
			assert.True(t, !entry.Empty)
		}()
	}

	time.Sleep(100 * time.Millisecond)

	// 重新加载缓存
	reloadEntry, _ := fileCache.ReLoad(testNamespace, testGroup, testFile)
	assert.True(t, !reloadEntry.Empty)

	for i := 0; i < 100; i++ {
		go func() {
			entry, _ := fileCache.GetOrLoadIfAbsent(testNamespace, testGroup, testFile)
			assert.True(t, !entry.Empty)
		}()
	}

	time.Sleep(100 * time.Millisecond)
}

// TestConcurrentGetConfigFile 测试 1000 个并发拉取配置
func TestConcurrentGetConfigFile(t *testing.T) {
	control, mockedStorage, fileCache := newConfigFileMockedCache(t)
	defer control.Finish()
	defer fileCache.Clear()

	// Mock 数据
	configFile := assembleConfigFile()
	configFileRelease := assembleConfigFileRelease(configFile)

	// 一共调用三次，
	mockedStorage.EXPECT().GetConfigFileRelease(nil, testNamespace, testGroup, testFile).Return(configFileRelease, nil).Times(1)

	for i := 0; i < 1000; i++ {
		go func() {
			entry, _ := fileCache.GetOrLoadIfAbsent(testNamespace, testGroup, testFile)
			assert.True(t, !entry.Empty)
		}()
	}

	time.Sleep(500 * time.Millisecond)
}

// TestUpdateCache 测试配置发布时，更新缓存
func TestUpdateCache(t *testing.T) {
	control, mockedStorage, fileCache := newConfigFileMockedCache(t)
	fileCache.Clear()
	defer control.Finish()

	// Mock 数据
	// 第一次调用返会值
	firstValue := "firstValue"
	firstVersion := uint64(10)
	configFile := assembleConfigFile()
	configFileRelease := assembleConfigFileRelease(configFile)
	configFileRelease.Content = firstValue
	configFileRelease.Version = firstVersion
	first := mockedStorage.EXPECT().GetConfigFileRelease(nil, testNamespace, testGroup, testFile).Return(configFileRelease, nil).Times(1)

	// 第二次调用返会值
	secondValue := "secondValue"
	secondVersion := uint64(11)
	configFile2 := assembleConfigFile()
	configFileRelease2 := assembleConfigFileRelease(configFile2)
	configFileRelease2.Content = secondValue
	configFileRelease2.Version = secondVersion
	second := mockedStorage.EXPECT().GetConfigFileRelease(nil, testNamespace, testGroup, testFile).Return(configFileRelease2, nil).Times(1)

	gomock.InOrder(first, second)

	for i := 0; i < 100; i++ {
		go func() {
			entry, _ := fileCache.GetOrLoadIfAbsent(testNamespace, testGroup, testFile)
			assert.True(t, !entry.Empty)
			assert.Equal(t, firstValue, entry.Content)
			assert.Equal(t, firstVersion, entry.Version)
		}()
	}

	time.Sleep(100 * time.Millisecond)

	// 删除缓存
	fileCache.Remove(testNamespace, testGroup, testFile)

	for i := 0; i < 100; i++ {
		go func() {
			entry, _ := fileCache.GetOrLoadIfAbsent(testNamespace, testGroup, testFile)
			assert.True(t, !entry.Empty)
			assert.Equal(t, secondValue, entry.Content)
			assert.Equal(t, secondVersion, entry.Version)
		}()
	}

	time.Sleep(100 * time.Millisecond)
}

func newConfigFileMockedCache(t *testing.T) (*gomock.Controller, *mock.MockStore, *FileCache) {
	control := gomock.NewController(t)
	mockedStorage := mock.NewMockStore(control)
	fileCache := NewFileCache(context.Background(), mockedStorage, FileCacheParam{
		ExpireTimeAfterWrite: 60000,
	})

	return control, mockedStorage, fileCache
}

func assembleConfigFile() *model.ConfigFile {
	return &model.ConfigFile{
		Namespace: testNamespace,
		Group:     testGroup,
		Name:      testFile,
		Format:    utils.FileFormatText,
		Content:   "k1=v1,k2=v2",
	}
}

func assembleConfigFileRelease(configFile *model.ConfigFile) *model.ConfigFileRelease {
	return &model.ConfigFileRelease{
		Name:      "release-name",
		Namespace: configFile.Namespace,
		Group:     configFile.Group,
		FileName:  configFile.Name,
		CreateBy:  "polaris",
		Content:   configFile.Content,
		Version:   uint64(10),
	}
}
