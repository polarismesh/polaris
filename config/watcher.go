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

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
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

type (
	BetaReleaseMatcher func(clientLabels map[string]string, event *model.SimpleConfigFileRelease) bool

	FileReleaseCallback func(clientId string, rsp *apiconfig.ConfigClientResponse) bool

	WatchContextFactory func(clientId string, matcher BetaReleaseMatcher) WatchContext

	WatchContext interface {
		// ClientID 客户端发起的
		ClientID() string
		// ClientLabels 客户端的标识，用于灰度发布要做标签的匹配判断
		ClientLabels() map[string]string
		// AppendInterest 客户端增加订阅列表
		AppendInterest(item *apiconfig.ClientConfigFileInfo)
		// RemoveInterest 客户端删除订阅列表
		RemoveInterest(item *apiconfig.ClientConfigFileInfo)
		// ShouldNotify 判断是不是需要通知客户端某个配置变动了
		ShouldNotify(event *model.SimpleConfigFileRelease) bool
		// Reply 真正的通知逻辑
		Reply(rsp *apiconfig.ConfigClientResponse)
		// Close .
		Close() error
		// ShouldExpire 是不是存在有效时间
		ShouldExpire(now time.Time) bool
		// ListWatchFiles 列举出当前订阅的所有配置文件
		ListWatchFiles() []*apiconfig.ClientConfigFileInfo
		// IsOnce 是不是只能被通知一次
		IsOnce() bool
	}
)

type LongPollWatchContext struct {
	clientId         string
	labels           map[string]string
	once             sync.Once
	finishTime       time.Time
	finishChan       chan *apiconfig.ConfigClientResponse
	watchConfigFiles map[string]*apiconfig.ClientConfigFileInfo
	betaMatcher      BetaReleaseMatcher
}

func (c *LongPollWatchContext) ClientLabels() map[string]string {
	return c.labels
}

// IsOnce
func (c *LongPollWatchContext) IsOnce() bool {
	return true
}

func (c *LongPollWatchContext) GetNotifieResult() *apiconfig.ConfigClientResponse {
	return <-c.finishChan
}

func (c *LongPollWatchContext) GetNotifieResultWithTime(timeout time.Duration) (*apiconfig.ConfigClientResponse, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case ret := <-c.finishChan:
		return ret, nil
	case <-timer.C:
		return nil, context.DeadlineExceeded
	}
}

func (c *LongPollWatchContext) ShouldExpire(now time.Time) bool {
	return now.After(c.finishTime)
}

// ClientID .
func (c *LongPollWatchContext) ClientID() string {
	return c.clientId
}

func (c *LongPollWatchContext) ShouldNotify(event *model.SimpleConfigFileRelease) bool {
	if event.ReleaseType == model.ReleaseTypeGray && !c.betaMatcher(c.ClientLabels(), event) {
		return false
	}

	key := event.FileKey()
	watchFile, ok := c.watchConfigFiles[key]
	if !ok {
		return false
	}
	return watchFile.GetVersion().GetValue() < event.Version
}

func (c *LongPollWatchContext) ListWatchFiles() []*apiconfig.ClientConfigFileInfo {
	ret := make([]*apiconfig.ClientConfigFileInfo, 0, len(c.watchConfigFiles))
	for _, v := range c.watchConfigFiles {
		ret = append(ret, v)
	}
	return ret
}

// AppendInterest .
func (c *LongPollWatchContext) AppendInterest(item *apiconfig.ClientConfigFileInfo) {
	key := model.BuildKeyForClientConfigFileInfo(item)
	c.watchConfigFiles[key] = item
}

// RemoveInterest .
func (c *LongPollWatchContext) RemoveInterest(item *apiconfig.ClientConfigFileInfo) {
	key := model.BuildKeyForClientConfigFileInfo(item)
	delete(c.watchConfigFiles, key)
}

// Close .
func (c *LongPollWatchContext) Close() error {
	return nil
}

func (c *LongPollWatchContext) Reply(rsp *apiconfig.ConfigClientResponse) {
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
	clients *utils.SyncMap[string, WatchContext]
	// fileId -> []clientId
	watchers *utils.SyncMap[string, *utils.SyncSet[string]]
	// fileCache
	fileCache cachetypes.ConfigFileCache
	cacheMgr  cachetypes.CacheManager
	cancel    context.CancelFunc
}

// NewWatchCenter 创建一个客户端监听配置发布的处理中心
func NewWatchCenter(cacheMgr cachetypes.CacheManager) (*watchCenter, error) {
	ctx, cancel := context.WithCancel(context.Background())

	wc := &watchCenter{
		clients:   utils.NewSyncMap[string, WatchContext](),
		watchers:  utils.NewSyncMap[string, *utils.SyncSet[string]](),
		fileCache: cacheMgr.ConfigFile(),
		cacheMgr:  cacheMgr,
		cancel:    cancel,
	}

	var err error
	wc.subCtx, err = eventhub.Subscribe(eventhub.ConfigFilePublishTopic, wc, eventhub.WithQueueSize(QueueSize))
	if err != nil {
		return nil, err
	}
	go wc.startHandleTimeoutRequestWorker(ctx)
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

func (wc *watchCenter) CheckQuickResponseClient(watchCtx WatchContext) *apiconfig.ConfigClientResponse {
	buildRet := func(release *model.ConfigFileRelease) *apiconfig.ConfigClientResponse {
		ret := &apiconfig.ClientConfigFileInfo{
			Namespace: utils.NewStringValue(release.Namespace),
			Group:     utils.NewStringValue(release.Group),
			FileName:  utils.NewStringValue(release.FileName),
			Version:   utils.NewUInt64Value(release.Version),
			Md5:       utils.NewStringValue(release.Md5),
			Name:      utils.NewStringValue(release.Name),
		}
		return api.NewConfigClientResponse(apimodel.Code_ExecuteSuccess, ret)
	}

	for _, configFile := range watchCtx.ListWatchFiles() {
		namespace := configFile.GetNamespace().GetValue()
		group := configFile.GetGroup().GetValue()
		fileName := configFile.GetFileName().GetValue()
		// 从缓存中获取灰度文件
		if len(watchCtx.ClientLabels()) > 0 {
			if release := wc.fileCache.GetActiveGrayRelease(namespace, group, fileName); release != nil {
				if watchCtx.ShouldNotify(release.SimpleConfigFileRelease) {
					return buildRet(release)
				}
			}
		}
		release := wc.fileCache.GetActiveRelease(namespace, group, fileName)
		// 从缓存中获取最新的配置文件信息
		if release != nil && watchCtx.ShouldNotify(release.SimpleConfigFileRelease) {
			return buildRet(release)
		}
	}
	return nil
}

// GetWatchContext .
func (wc *watchCenter) GetWatchContext(clientId string) (WatchContext, bool) {
	return wc.clients.Load(clientId)
}

// DelWatchContext .
func (wc *watchCenter) DelWatchContext(clientId string) (WatchContext, bool) {
	return wc.clients.Delete(clientId)
}

// AddWatcher 新增订阅者
func (wc *watchCenter) AddWatcher(clientId string,
	watchFiles []*apiconfig.ClientConfigFileInfo, factory WatchContextFactory) WatchContext {
	watchCtx, _ := wc.clients.ComputeIfAbsent(clientId, func(k string) WatchContext {
		return factory(clientId, wc.MatchBetaReleaseFile)
	})

	for _, file := range watchFiles {
		fileKey := utils.GenFileId(file.GetNamespace().GetValue(), file.GetGroup().GetValue(), file.GetFileName().GetValue())

		watchCtx.AppendInterest(file)
		clientIds, _ := wc.watchers.ComputeIfAbsent(fileKey, func(k string) *utils.SyncSet[string] {
			return utils.NewSyncSet[string]()
		})
		clientIds.Add(clientId)
	}
	return watchCtx
}

// RemoveAllWatcher 删除订阅者
func (wc *watchCenter) RemoveAllWatcher(clientId string) {
	oldVal, exist := wc.clients.Delete(clientId)
	if !exist {
		return
	}
	_ = oldVal.Close()
	for _, file := range oldVal.ListWatchFiles() {
		watchFileId := utils.GenFileId(file.Namespace.GetValue(), file.Group.GetValue(), file.FileName.GetValue())
		watchers, ok := wc.watchers.Load(watchFileId)
		if !ok {
			continue
		}
		watchers.Remove(clientId)
	}
	wc.clients.Delete(clientId)
}

// RemoveWatcher 删除订阅者
func (wc *watchCenter) RemoveWatcher(clientId string, watchConfigFiles []*apiconfig.ClientConfigFileInfo) {
	oldVal, exist := wc.clients.Delete(clientId)
	if exist {
		_ = oldVal.Close()
	}
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
	changeNotifyRequest := publishConfigFile.ToSpecNotifyClientRequest()
	response := api.NewConfigClientResponse(apimodel.Code_ExecuteSuccess, changeNotifyRequest)

	notifyCnt := 0
	clientIds.Range(func(clientId string) {
		watchCtx, ok := wc.clients.Load(clientId)
		if !ok {
			log.Warn("[Config][Watcher] not found client when do notify.", zap.String("clientId", clientId),
				zap.String("file", watchFileId))
			return
		}

		if watchCtx.ShouldNotify(publishConfigFile) {
			watchCtx.Reply(response)
			notifyCnt++
			// 只能用一次，通知完就要立马清理掉这个 WatchContext
			if watchCtx.IsOnce() {
				wc.RemoveAllWatcher(watchCtx.ClientID())
			}
		}
	})

	log.Info("[Config][Watcher] received config file release event.", zap.String("file", watchFileId),
		zap.Uint64("version", publishConfigFile.Version), zap.Int("clients", clientIds.Len()),
		zap.Int("notify", notifyCnt))
}

func (wc *watchCenter) MatchBetaReleaseFile(clientLabels map[string]string, event *model.SimpleConfigFileRelease) bool {
	return wc.cacheMgr.Gray().HitGrayRule(model.GetGrayConfigRealseKey(event), clientLabels)
}

func (wc *watchCenter) Close() {
	wc.cancel()
	wc.subCtx.Cancel()
}

func (wc *watchCenter) startHandleTimeoutRequestWorker(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			tNow := time.Now()
			waitRemove := make([]WatchContext, 0, 32)
			wc.clients.Range(func(client string, watchCtx WatchContext) {
				if watchCtx.ShouldExpire(tNow) {
					waitRemove = append(waitRemove, watchCtx)
				}
			})
			if len(waitRemove) > 0 {
				log.Info("remove expire watch context", zap.Any("client-ids", waitRemove))
			}

			for i := range waitRemove {
				watchCtx := waitRemove[i]
				watchCtx.Reply(notModifiedResponse)
				wc.RemoveAllWatcher(watchCtx.ClientID())
			}
		}
	}
}
