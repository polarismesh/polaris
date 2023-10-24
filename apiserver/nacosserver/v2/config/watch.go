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
	"time"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"

	nacosmodel "github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacospb "github.com/polarismesh/polaris/apiserver/nacosserver/v2/pb"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/remote"
	"github.com/polarismesh/polaris/common/utils"
)

type StreamWatchContext struct {
	clientId          string
	connectionManager *remote.ConnectionManager
	watchConfigFiles  *utils.SyncMap[string, *apiconfig.ClientConfigFileInfo]
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

func (c *StreamWatchContext) ShouldNotify(resp *apiconfig.ClientConfigFileInfo) bool {
	key := resp.GetNamespace().GetValue() + "@" +
		resp.GetGroup().GetValue() + "@" + resp.GetName().GetValue()
	watchFile, ok := c.watchConfigFiles.Load(key)
	if !ok {
		return false
	}
	return watchFile.GetMd5().GetValue() != resp.GetMd5().GetValue()
}

func (c *StreamWatchContext) ListWatchFiles() []*apiconfig.ClientConfigFileInfo {
	return c.watchConfigFiles.Values()
}

// AppendInterest .
func (c *StreamWatchContext) AppendInterest(item *apiconfig.ClientConfigFileInfo) {
	key := item.GetNamespace().GetValue() + "@" +
		item.GetGroup().GetValue() + "@" + item.GetFileName().GetValue()
	c.watchConfigFiles.Store(key, item)
}

// RemoveInterest .
func (c *StreamWatchContext) RemoveInterest(item *apiconfig.ClientConfigFileInfo) {
	key := item.GetNamespace().GetValue() + "@" +
		item.GetGroup().GetValue() + "@" + item.GetFileName().GetValue()
	c.watchConfigFiles.Delete(key)
}

// Close .
func (c *StreamWatchContext) Close() error {
	return nil
}

func (c *StreamWatchContext) Reply(event *apiconfig.ConfigClientResponse) {
	viewConfig := event.GetConfigFile()
	notifyRequest := nacospb.NewConfigChangeNotifyRequest()
	notifyRequest.Tenant = nacosmodel.ToNacosNamespace(viewConfig.GetNamespace().GetValue())
	notifyRequest.Group = viewConfig.GetGroup().GetValue()
	notifyRequest.DataId = viewConfig.GetFileName().GetValue()

	watchClient, ok := c.connectionManager.GetClient(c.clientId)
	if !ok {
		return
	}
	stream, ok := watchClient.LoadStream()
	if !ok {
		return
	}
	if err := stream.SendMsg(notifyRequest); err != nil {
		// TODO need print log
	}
}
