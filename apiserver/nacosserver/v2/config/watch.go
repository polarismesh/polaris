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
	"time"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	"go.uber.org/zap"

	nacosmodel "github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacospb "github.com/polarismesh/polaris/apiserver/nacosserver/v2/pb"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/remote"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/model"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
	"github.com/polarismesh/polaris/plugin"
)

type ConnectionClientManager struct {
	configSvr *config.Server
	watchCtx  *eventhub.SubscribtionContext
}

func NewConnectionClientManager(configSvr *config.Server) (*ConnectionClientManager, error) {
	mgr := &ConnectionClientManager{
		configSvr: configSvr,
	}
	subCtx, err := eventhub.Subscribe(remote.ClientConnectionEvent, mgr)
	if err != nil {
		return nil, err
	}
	mgr.watchCtx = subCtx
	return mgr, nil
}

// PreProcess do preprocess logic for event
func (cm *ConnectionClientManager) PreProcess(_ context.Context, a any) any {
	return a
}

// OnEvent event process logic
func (c *ConnectionClientManager) OnEvent(ctx context.Context, a any) error {
	event, ok := a.(*remote.ConnectionEvent)
	if !ok {
		return nil
	}
	switch event.EventType {
	case remote.EventClientConnected:
		// do nothing
	case remote.EventClientDisConnected:
		c.configSvr.WatchCenter().RemoveAllWatcher(event.ConnID)
	}

	return nil
}

type StreamWatchContext struct {
	clientId         string
	labels           map[string]string
	connMgr          *remote.ConnectionManager
	watchConfigFiles *utils.SyncMap[string, *apiconfig.ClientConfigFileInfo]
	betaMatcher      config.BetaReleaseMatcher
}

func (c *StreamWatchContext) ClientLabels() map[string]string {
	return c.labels
}

// IsOnce
func (c *StreamWatchContext) IsOnce() bool {
	return false
}

func (c *StreamWatchContext) ShouldExpire(now time.Time) bool {
	return false
}

// ClientID .
func (c *StreamWatchContext) ClientID() string {
	return c.clientId
}

// ShouldNotify .
func (c *StreamWatchContext) ShouldNotify(event *model.SimpleConfigFileRelease) bool {
	if event.ReleaseType == model.ReleaseTypeGray && !c.betaMatcher(c.ClientLabels(), event) {
		return false
	}
	key := event.FileKey()
	watchFile, ok := c.watchConfigFiles.Load(key)
	if !ok {
		return false
	}
	// 删除操作，直接通知
	if !event.Valid {
		return true
	}
	isChange := watchFile.GetMd5().GetValue() != event.Md5
	return isChange
}

// ListWatchFiles .
func (c *StreamWatchContext) ListWatchFiles() []*apiconfig.ClientConfigFileInfo {
	return c.watchConfigFiles.Values()
}

// AppendInterest .
func (c *StreamWatchContext) AppendInterest(item *apiconfig.ClientConfigFileInfo) {
	key := model.BuildKeyForClientConfigFileInfo(item)
	c.watchConfigFiles.Store(key, item)
}

// RemoveInterest .
func (c *StreamWatchContext) RemoveInterest(item *apiconfig.ClientConfigFileInfo) {
	key := model.BuildKeyForClientConfigFileInfo(item)
	c.watchConfigFiles.Delete(key)
}

// Close .
func (c *StreamWatchContext) Close() error {
	return nil
}

// Reply .
func (c *StreamWatchContext) Reply(event *apiconfig.ConfigClientResponse) {
	viewConfig := event.GetConfigFile()
	notifyRequest := nacospb.NewConfigChangeNotifyRequest()
	notifyRequest.Tenant = nacosmodel.ToNacosConfigNamespace(viewConfig.GetNamespace().GetValue())
	notifyRequest.Group = viewConfig.GetGroup().GetValue()
	notifyRequest.DataId = viewConfig.GetFileName().GetValue()

	success := false
	startTime := commontime.CurrentMillisecond()
	defer func() {
		plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
			Action:    nacosmodel.ActionGrpcPushConfigFile,
			ClientIP:  c.ClientID(),
			Namespace: notifyRequest.Tenant,
			Resource:  metrics.ResourceOfConfigFile(notifyRequest.Group, notifyRequest.DataId),
			Timestamp: startTime,
			CostTime:  commontime.CurrentMillisecond() - startTime,
			Revision:  viewConfig.GetMd5().GetValue(),
			Success:   success,
		})
	}()

	remoteClient, ok := c.connMgr.GetClient(c.clientId)
	if !ok {
		nacoslog.Error("[NACOS-V2][Config][Push] send ConfigChangeNotifyRequest not found remoteClient",
			zap.String("clientId", c.ClientID()))
		return
	}
	stream, ok := remoteClient.LoadStream()
	if !ok {
		nacoslog.Error("[NACOS-V2][Config][Push] send ConfigChangeNotifyRequest not stream",
			zap.String("clientId", c.ClientID()))
		return
	}
	clientResp, err := remote.MarshalPayload(notifyRequest)
	if err != nil {
		nacoslog.Error("[NACOS-V2][Config][Push] send ConfigChangeNotifyRequest marshal payload",
			zap.String("clientId", c.ClientID()), zap.Error(err))
		return
	}
	if err := stream.SendMsg(clientResp); err != nil {
		nacoslog.Error("[NACOS-V2][Config][Push] send ConfigChangeNotifyRequest fail",
			zap.String("clientId", c.ClientID()), zap.Error(err))
	}
	success = true
}
