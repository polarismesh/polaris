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
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/event"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	utils2 "github.com/polarismesh/polaris-server/config/utils"
	"go.uber.org/zap"
	"sync"
)

type watchContext struct {
	fileReleaseCb func(clientId string, rsp *api.ConfigClientResponse) bool
	ClientVersion uint64
}

type watchCenter struct {
	eventCenter        *event.Center
	configFileWatchers *sync.Map //fileId -> clientId -> watchContext
	lock               *sync.Mutex
}

// NewWatchCenter 创建一个订阅中心
func NewWatchCenter(eventCenter *event.Center) *watchCenter {
	wc := &watchCenter{
		eventCenter:        eventCenter,
		configFileWatchers: new(sync.Map),
		lock:               new(sync.Mutex),
	}
	eventCenter.WatchEvent(EventTypePublishConfigFile, func(event event.Event) bool {
		wc.notifyToWatchers(event.Message.(*model.ConfigFileRelease))
		return true
	})
	return wc
}

// AddWatcher 新增订阅者
func (wc *watchCenter) AddWatcher(clientId string, watchConfigFiles []*api.ClientConfigFileInfo,
	fileReleaseCb func(clientId string, rsp *api.ConfigClientResponse) bool) {
	if len(watchConfigFiles) == 0 {
		return
	}
	for _, file := range watchConfigFiles {
		watchFileId := utils.GenFileId(file.Namespace.GetValue(), file.Group.GetValue(), file.FileName.GetValue())
		watchers, ok := wc.configFileWatchers.Load(watchFileId)
		if !ok {
			wc.lock.Lock()
			//double check
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

func (wc *watchCenter) notifyToWatchers(publishConfigFile *model.ConfigFileRelease) {
	watchFileId := utils.GenFileId(publishConfigFile.Namespace, publishConfigFile.Group, publishConfigFile.FileName)

	log.GetConfigLogger().Info("[Config][Watcher] received config file publish event.", zap.String("file", watchFileId))

	watchers, ok := wc.configFileWatchers.Load(watchFileId)
	if !ok {
		return
	}

	response := utils2.GenConfigFileResponse(publishConfigFile.Namespace, publishConfigFile.Group,
		publishConfigFile.FileName, "", publishConfigFile.Version)

	watcherMap := watchers.(*sync.Map)
	watcherMap.Range(func(clientId, context interface{}) bool {
		log.GetConfigLogger().Info("[Config][Watcher] notify to client.",
			zap.String("file", watchFileId),
			zap.String("clientId", clientId.(string)))

		c := context.(*watchContext)
		if c.ClientVersion < publishConfigFile.Version {
			c.fileReleaseCb(clientId.(string), response)
		}
		return true
	})
}
