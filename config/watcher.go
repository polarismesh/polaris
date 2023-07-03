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
	"context"
	"sync"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/hash"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

const (
	QueueSize = 10240
)

// Event 事件对象，包含类型和事件消息
type Event struct {
	EventType string
	Message   interface{}
}

type FileReleaseCallback func(clientId string, rsp *apiconfig.ConfigClientResponse) bool

type watchContext struct {
	fileReleaseCb FileReleaseCallback
	ClientVersion uint64
}

// watchCenter 处理客户端订阅配置请求，监听配置文件发布事件通知客户端
type watchCenter struct {
	// fileId -> clientId -> watchContext
	configFileWatchers  *utils.SegmentMap[string, *utils.SegmentMap[string, *watchContext]]
	lock                *sync.Mutex
	releaseMessageQueue chan *model.ConfigFileRelease
}

// NewWatchCenter 创建一个客户端监听配置发布的处理中心
func NewWatchCenter() *watchCenter {
	wc := &watchCenter{
		configFileWatchers:  utils.NewSegmentMap[string, *utils.SegmentMap[string, *watchContext]](128, hash.Fnv32),
		lock:                new(sync.Mutex),
		releaseMessageQueue: make(chan *model.ConfigFileRelease, QueueSize),
	}

	eventhub.Subscribe(eventTypePublishConfigFile, utils.NewUUID(), wc)
	go wc.handleMessage()
	return wc
}

// PreProcess do preprocess logic for event
func (wc *watchCenter) PreProcess(_ context.Context, e any) any {
	return e
}

// OnEvent event process logic
func (wc *watchCenter) OnEvent(ctx context.Context, arg any) error {
	event, ok := arg.(*Event)
	if !ok {
		return nil
	}
	wc.releaseMessageQueue <- event.Message.(*model.ConfigFileRelease)
	return nil
}

// AddWatcher 新增订阅者
func (wc *watchCenter) AddWatcher(clientId string, watchConfigFiles []*apiconfig.ClientConfigFileInfo,
	fileReleaseCb FileReleaseCallback) {
	if len(watchConfigFiles) == 0 {
		return
	}
	for _, file := range watchConfigFiles {
		watchFileId := utils.GenFileId(file.Namespace.GetValue(), file.Group.GetValue(), file.FileName.GetValue())
		log.Info("[Config][Watcher] add watcher.", zap.Any("client-id", clientId),
			zap.String("watch-file-id", watchFileId), zap.Uint64("client-version", file.Version.GetValue()))

		watchers, _ := wc.configFileWatchers.ComputeIfAbsent(watchFileId,
			func(k string) *utils.SegmentMap[string, *watchContext] {
				newWatchers := utils.NewSegmentMap[string, *watchContext](128, hash.Fnv32)
				return newWatchers
			})
		watchers.Put(clientId, &watchContext{
			fileReleaseCb: fileReleaseCb,
			ClientVersion: file.Version.GetValue(),
		})
	}
}

// RemoveWatcher 删除订阅者
func (wc *watchCenter) RemoveWatcher(clientId string, watchConfigFiles []*apiconfig.ClientConfigFileInfo) {
	if len(watchConfigFiles) == 0 {
		return
	}

	for _, file := range watchConfigFiles {
		watchFileId := utils.GenFileId(file.Namespace.GetValue(), file.Group.GetValue(), file.FileName.GetValue())
		watchers, ok := wc.configFileWatchers.Get(watchFileId)
		if !ok {
			continue
		}
		watchers.Del(clientId)
	}
}

func (wc *watchCenter) handleMessage() {
	defer func() {
		if err := recover(); err != nil {
			log.Error("[Config][Watcher] handler config release message error.", zap.Any("err", err))
		}
	}()

	for message := range wc.releaseMessageQueue {
		wc.notifyToWatchers(message)
	}
}

func (wc *watchCenter) notifyToWatchers(publishConfigFile *model.ConfigFileRelease) {
	watchFileId := utils.GenFileId(publishConfigFile.Namespace, publishConfigFile.Group, publishConfigFile.FileName)

	log.Info("[Config][Watcher] received config file publish message.", zap.String("file", watchFileId))

	watchers, ok := wc.configFileWatchers.Get(watchFileId)
	if !ok {
		return
	}

	response := GenConfigFileResponse(publishConfigFile.Namespace, publishConfigFile.Group,
		publishConfigFile.FileName, "", publishConfigFile.Md5, publishConfigFile.Version)

	watchers.Range(func(clientId string, watchCtx *watchContext) {
		if watchCtx.ClientVersion < publishConfigFile.Version {
			log.Info("[Config][Watcher] notify to client.",
				zap.String("file", watchFileId), zap.String("clientId", clientId),
				zap.Uint64("version", publishConfigFile.Version))
			watchCtx.fileReleaseCb(clientId, response)
		} else {
			log.Info("[Config][Watcher] notify to client ignore.",
				zap.String("file", watchFileId), zap.String("clientId", clientId),
				zap.Uint64("client-version", watchCtx.ClientVersion),
				zap.Uint64("version", publishConfigFile.Version))
		}
	})
}
