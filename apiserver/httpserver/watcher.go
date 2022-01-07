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
	"github.com/emicklei/go-restful"
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/config"
	"go.uber.org/zap"
	"sync"
)

type watchContext struct {
	FinishChan chan interface{}
	Response   *restful.Response
	ClientMd5  string
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
	rsp *restful.Response, finishChan chan interface{}) {
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
					FinishChan: finishChan,
					Response:   rsp,
					ClientMd5:  file.Md5.GetValue(),
				})
				configFileWatchers.Store(watchFileId, w)
			}
			lock.Unlock()
			continue
		}

		w := watchers.(*sync.Map)
		w.Store(clientId, &watchContext{
			FinishChan: finishChan,
			Response:   rsp,
			ClientMd5:  file.Md5.GetValue(),
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

	log.GetConfigLogger().Info("[Nacos] received config file publish event.", zap.String("file", watchFileId))

	watchers, ok := configFileWatchers.Load(watchFileId)
	if !ok {
		return
	}

	response := genConfigFileChangeResponse(publishConfigFile.Namespace, publishConfigFile.Group,
		publishConfigFile.FileName, publishConfigFile.Md5)

	w := watchers.(*sync.Map)
	w.Range(func(clientId, context interface{}) bool {
		log.GetConfigLogger().Info("[Nacos] notify to client.",
			zap.String("file", watchFileId),
			zap.String("clientId", clientId.(string)))

		c := context.(*watchContext)
		if c.ClientMd5 != publishConfigFile.Md5 {
			err := writeResponse(response, c.Response)
			if err != nil {
				//请求响应的时候，可能客户端已经断开链接，属于可预见的情况，所以打印 warn 日志
				log.GetConfigLogger().Warn("[Nacos] response long polling error.",
					zap.String("file", publishConfigFile.FileName),
					zap.String("clientId", clientId.(string)),
					zap.Error(err))
			}
			c.FinishChan <- new(interface{})
		}
		return true
	})
}
