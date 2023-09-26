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
	"time"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

const (
	defaultLongPollingTimeout = 30000 * time.Millisecond
	QueueSize                 = 10240
)

var (
	notModifiedResponse = &apiconfig.ConfigClientResponse{
		Code:       utils.NewUInt32Value(uint32(apimodel.Code_DataNoChange)),
		ConfigFile: nil,
	}
)

type FileReleaseCallback func(clientId string, rsp *apiconfig.ConfigClientResponse) bool

type watchContext struct {
	clientId      string
	fileVersions  map[string]uint64
	fileReleaseCb FileReleaseCallback
	//
	once             sync.Once
	finishTime       time.Time
	finishChan       chan *apiconfig.ConfigClientResponse
	watchConfigFiles []*apiconfig.ClientConfigFileInfo
}

func (c *watchContext) reply(rsp *apiconfig.ConfigClientResponse) {
	c.once.Do(func() {
		c.finishChan <- rsp
		close(c.finishChan)
	})
}

// watchCenter 处理客户端订阅配置请求，监听配置文件发布事件通知客户端
type watchCenter struct {
	subCtx *eventhub.SubscribtionContext
	lock   sync.Mutex
	// clientId -> watchContext
	clients *utils.SyncMap[string, *watchContext]
	// fileId -> []clientId
	watchers *utils.SyncMap[string, *utils.SyncSet[string]]
}

// NewWatchCenter 创建一个客户端监听配置发布的处理中心
func NewWatchCenter() (*watchCenter, error) {
	wc := &watchCenter{
		clients:  utils.NewSyncMap[string, *watchContext](),
		watchers: utils.NewSyncMap[string, *utils.SyncSet[string]](),
	}

	var err error
	wc.subCtx, err = eventhub.Subscribe(eventhub.ConfigFilePublishTopic, wc, eventhub.WithQueueSize(QueueSize))
	if err != nil {
		return nil, err
	}
	return wc, nil
}

// PreProcess do preprocess logic for event
func (wc *watchCenter) PreProcess(_ context.Context, e any) any {
	return e
}

// OnEvent event process logic
func (wc *watchCenter) OnEvent(ctx context.Context, arg any) error {
	event, ok := arg.(*eventhub.PublishConfigFileEvent)
	if !ok {
		log.Warn("[Config][Watcher] receive invalid event type")
		return nil
	}
	wc.notifyToWatchers(event.Message)
	return nil
}

// AddWatcher 新增订阅者
func (wc *watchCenter) AddWatcher(clientId string,
	watchFiles []*apiconfig.ClientConfigFileInfo) <-chan *apiconfig.ConfigClientResponse {
	if len(watchFiles) == 0 {
		return nil
	}

	watcheCtx, _ := wc.clients.ComputeIfAbsent(clientId,
		func(k string) *watchContext {
			return &watchContext{
				clientId: clientId,
				fileReleaseCb: func(clientId string, rsp *apiconfig.ConfigClientResponse) bool {
					if watchCtx, ok := wc.clients.Load(clientId); ok {
						watchCtx.reply(rsp)
					}
					return true
				},
				fileVersions: map[string]uint64{},
				finishChan:   make(chan *apiconfig.ConfigClientResponse),
			}
		})

	for _, file := range watchFiles {
		watchFileId := utils.GenFileId(file.Namespace.GetValue(), file.Group.GetValue(), file.FileName.GetValue())
		log.Info("[Config][Watcher] add watcher.", zap.Any("client-id", clientId),
			zap.String("watch-file-id", watchFileId), zap.Uint64("client-version", file.Version.GetValue()))
		watcheCtx.fileVersions[watchFileId] = file.GetVersion().GetValue()

		clientIds, _ := wc.watchers.ComputeIfAbsent(watchFileId, func(k string) *utils.SyncSet[string] {
			return utils.NewSyncSet[string]()
		})
		clientIds.Add(clientId)
	}

	return watcheCtx.finishChan
}

// RemoveWatcher 删除订阅者
func (wc *watchCenter) RemoveWatcher(clientId string, watchConfigFiles []*apiconfig.ClientConfigFileInfo) {
	wc.clients.Delete(clientId)
	if len(watchConfigFiles) == 0 {
		return
	}

	for _, file := range watchConfigFiles {
		watchFileId := utils.GenFileId(file.Namespace.GetValue(), file.Group.GetValue(), file.FileName.GetValue())
		watchers, ok := wc.watchers.Load(watchFileId)
		if !ok {
			continue
		}
		watchers.Remove(clientId)
	}
}

func (wc *watchCenter) notifyToWatchers(publishConfigFile *model.SimpleConfigFileRelease) {
	watchFileId := utils.GenFileId(publishConfigFile.Namespace, publishConfigFile.Group, publishConfigFile.FileName)

	clientIds, ok := wc.watchers.Load(watchFileId)
	if !ok {
		return
	}

	log.Info("[Config][Watcher] received config file publish message.", zap.String("file", watchFileId))

	response := GenConfigFileResponse(publishConfigFile.Namespace, publishConfigFile.Group,
		publishConfigFile.FileName, "", publishConfigFile.Md5, publishConfigFile.Version)

	clientIds.Range(func(clientId string) {
		watchCtx, ok := wc.clients.Load(clientId)
		if !ok {
			clientIds.Remove(clientId)
			return
		}
		clientVersion := watchCtx.fileVersions[watchFileId]
		if clientVersion < publishConfigFile.Version {
			watchCtx.fileReleaseCb(clientId, response)
			log.Info("[Config][Watcher] notify to client.",
				zap.String("file", watchFileId), zap.String("clientId", clientId),
				zap.Uint64("version", publishConfigFile.Version))
		} else {
			log.Info("[Config][Watcher] notify to client ignore.",
				zap.String("file", watchFileId), zap.String("clientId", clientId),
				zap.Uint64("client-version", clientVersion),
				zap.Uint64("version", publishConfigFile.Version))
		}
	})
}

func (wc *watchCenter) startHandleTimeoutRequestWorker(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if wc.clients.Len() == 0 {
				continue
			}
			tNow := time.Now()
			wc.clients.Range(func(client string, watchCtx *watchContext) {
				if tNow.After(watchCtx.finishTime) {
					watchCtx.reply(notModifiedResponse)
					wc.RemoveWatcher(client, watchCtx.watchConfigFiles)
				}
			})
		}
	}
}
