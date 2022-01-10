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

package httpserver

import (
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/config"
	"go.uber.org/zap"
	"sync"
)

type watchContext struct {
	FinishChan    chan interface{}
	handler       *Handler
	ClientVersion uint64
}

var (
	//fileId -> clientId -> watchContext
	configFileWatchers *sync.Map
	lock               *sync.Mutex
)

func (h *HTTPServer) initWatchCenter() {
	configFileWatchers = new(sync.Map)
	lock = new(sync.Mutex)

	//监听配置发布事件
	h.configServer.WatchPublishConfigEvent(func(event config.Event) bool {
		h.notifyToClients(event.Message.(*model.ConfigFileRelease))
		return true
	})
}

func (h *HTTPServer) addWatcher(clientId string, watchConfigFiles []*api.ClientConfigFileInfo,
	handler *Handler, finishChan chan interface{}) {
	if len(watchConfigFiles) == 0 {
		return
	}
	for _, file := range watchConfigFiles {
		watchFileId := cache.GenFileId(file.Namespace.GetValue(), file.Group.GetValue(), file.FileName.GetValue())
		watchers, ok := configFileWatchers.Load(watchFileId)
		if !ok {
			lock.Lock()
			//double check
			watchers, ok = configFileWatchers.Load(watchFileId)
			if !ok {
				w := new(sync.Map)
				w.Store(clientId, &watchContext{
					FinishChan:    finishChan,
					handler:       handler,
					ClientVersion: file.Version.GetValue(),
				})
				configFileWatchers.Store(watchFileId, w)
			}
			lock.Unlock()
			continue
		}

		w := watchers.(*sync.Map)
		w.Store(clientId, &watchContext{
			FinishChan:    finishChan,
			handler:       handler,
			ClientVersion: file.Version.GetValue(),
		})
	}
}

func (h *HTTPServer) removeWatcher(clientId string, watchConfigFiles []*api.ClientConfigFileInfo) {
	if len(watchConfigFiles) == 0 {
		return
	}

	for _, file := range watchConfigFiles {
		watchFileId := cache.GenFileId(file.Namespace.GetValue(), file.Group.GetValue(), file.FileName.GetValue())
		watchers, ok := configFileWatchers.Load(watchFileId)
		if !ok {
			continue
		}
		w := watchers.(*sync.Map)
		w.Delete(clientId)
	}
}

func (h *HTTPServer) notifyToClients(publishConfigFile *model.ConfigFileRelease) {
	watchFileId := cache.GenFileId(publishConfigFile.Namespace, publishConfigFile.Group, publishConfigFile.FileName)

	log.GetConfigLogger().Info("[Config][HttpServer] received config file publish event.", zap.String("file", watchFileId))

	watchers, ok := configFileWatchers.Load(watchFileId)
	if !ok {
		return
	}

	response := genConfigFileResponse(publishConfigFile.Namespace, publishConfigFile.Group,
		publishConfigFile.FileName, "", publishConfigFile.Version)

	w := watchers.(*sync.Map)
	w.Range(func(clientId, context interface{}) bool {
		log.GetConfigLogger().Info("[Config][HttpServer] notify to client.",
			zap.String("file", watchFileId),
			zap.String("clientId", clientId.(string)))

		c := context.(*watchContext)
		if c.ClientVersion < publishConfigFile.Version {
			c.handler.WriteHeaderAndProto(response)
			c.FinishChan <- new(interface{})
		}
		return true
	})
}
