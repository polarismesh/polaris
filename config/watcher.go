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
	"sync"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	utils2 "github.com/polarismesh/polaris-server/config/utils"
	"go.uber.org/zap"
)

const (
	QueueSize = 10240
)

type FileReleaseCallback func(clientId string, rsp *api.ConfigClientResponse) bool

type watchContext struct {
	fileReleaseCb FileReleaseCallback
	ClientVersion uint64
}

// watchCenter 处理客户端订阅配置请求，监听配置文件发布事件通知客户端
type watchCenter struct {
	eventCenter         *Center
	configFileWatchers  *sync.Map // fileId -> clientId -> watchContext
	lock                *sync.Mutex
	releaseMessageQueue chan *model.ConfigFileRelease
}

// NewWatchCenter 创建一个客户端监听配置发布的处理中心
func NewWatchCenter(eventCenter *Center) *watchCenter {
	wc := &watchCenter{
		eventCenter:         eventCenter,
		configFileWatchers:  new(sync.Map),
		lock:                new(sync.Mutex),
		releaseMessageQueue: make(chan *model.ConfigFileRelease, QueueSize),
	}

	eventCenter.WatchEvent(eventTypePublishConfigFile, func(event Event) bool {
		wc.releaseMessageQueue <- event.Message.(*model.ConfigFileRelease)
		return true
	})

	wc.handleMessage()

	return wc
}

// AddWatcher 新增订阅者
func (wc *watchCenter) AddWatcher(clientId string, watchConfigFiles []*api.ClientConfigFileInfo,
	fileReleaseCb FileReleaseCallback) {
	if len(watchConfigFiles) == 0 {
		return
	}
	for _, file := range watchConfigFiles {
		watchFileId := utils.GenFileId(file.Namespace.GetValue(), file.Group.GetValue(), file.FileName.GetValue())

		log.ConfigScope().Info("[Config][Watcher] add watcher.", zap.Any("client-id", clientId),
			zap.String("watch-file-id", watchFileId), zap.Uint64("client-version", file.Version.GetValue()))

		watchers, ok := wc.configFileWatchers.Load(watchFileId)
		if !ok {
			wc.lock.Lock()
			// double check
			watchers, ok = wc.configFileWatchers.Load(watchFileId)
			if !ok {
				newWatchers := new(sync.Map)
				newWatchers.Store(clientId, &watchContext{
					fileReleaseCb: fileReleaseCb,
					ClientVersion: file.Version.GetValue(),
				})
				wc.configFileWatchers.Store(watchFileId, newWatchers)
			}
			wc.lock.Unlock()
			continue
		}

		watcherMap := watchers.(*sync.Map)

		watcherMap.Store(clientId, &watchContext{
			fileReleaseCb: fileReleaseCb,
			ClientVersion: file.Version.GetValue(),
		})
	}
}

// RemoveWatcher 删除订阅者
func (wc *watchCenter) RemoveWatcher(clientId string, watchConfigFiles []*api.ClientConfigFileInfo) {
	if len(watchConfigFiles) == 0 {
		return
	}

	for _, file := range watchConfigFiles {
		watchFileId := utils.GenFileId(file.Namespace.GetValue(), file.Group.GetValue(), file.FileName.GetValue())
		watchers, ok := wc.configFileWatchers.Load(watchFileId)
		if !ok {
			continue
		}
		watcherMap := watchers.(*sync.Map)
		watcherMap.Delete(clientId)
	}
}

func (wc *watchCenter) handleMessage() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.ConfigScope().Error("[Config][Watcher] handler config release message error.", zap.Any("error", err))
			}
		}()

		for message := range wc.releaseMessageQueue {
			wc.notifyToWatchers(message)
		}
	}()
}

func (wc *watchCenter) notifyToWatchers(publishConfigFile *model.ConfigFileRelease) {
	watchFileId := utils.GenFileId(publishConfigFile.Namespace, publishConfigFile.Group, publishConfigFile.FileName)

	log.ConfigScope().Info("[Config][Watcher] received config file publish message.", zap.String("file", watchFileId))

	watchers, ok := wc.configFileWatchers.Load(watchFileId)
	if !ok {
		return
	}

	response := utils2.GenConfigFileResponse(publishConfigFile.Namespace, publishConfigFile.Group,
		publishConfigFile.FileName, "", publishConfigFile.Md5, publishConfigFile.Version)

	watcherMap := watchers.(*sync.Map)
	watcherMap.Range(func(clientId, watchCtx interface{}) bool {

		c := watchCtx.(*watchContext)
		if c.ClientVersion < publishConfigFile.Version {
			log.ConfigScope().Info("[Config][Watcher] notify to client.",
				zap.String("file", watchFileId),
				zap.String("clientId", clientId.(string)),
				zap.Uint64("version", publishConfigFile.Version))
			c.fileReleaseCb(clientId.(string), response)
		} else {
			log.ConfigScope().Info("[Config][Watcher] notify to client ignore.",
				zap.String("file", watchFileId),
				zap.String("clientId", clientId.(string)),
				zap.Uint64("client-version", c.ClientVersion),
				zap.Uint64("version", publishConfigFile.Version))
		}
		return true
	})
}
