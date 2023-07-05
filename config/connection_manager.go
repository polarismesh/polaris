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

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/hash"
	"github.com/polarismesh/polaris/common/utils"
)

const (
	defaultLongPollingTimeout = 30000 * time.Millisecond
)

type connection struct {
	finishTime       time.Time
	finishChan       chan *apiconfig.ConfigClientResponse
	watchConfigFiles []*apiconfig.ClientConfigFileInfo
}

type connManager struct {
	watchCenter    *watchCenter
	conns          *utils.SegmentMap[string, *connection] // client -> connection
	stopWorkerFunc context.CancelFunc
}

var (
	notModifiedResponse = &apiconfig.ConfigClientResponse{
		Code:       utils.NewUInt32Value(api.DataNoChange),
		ConfigFile: nil,
	}

	cm *connManager
)

// NewConfigConnManager 初始化连接管理器，定时响应超时的请求
func NewConfigConnManager(ctx context.Context, watchCenter *watchCenter) *connManager {
	cm = &connManager{
		conns:       utils.NewSegmentMap[string, *connection](128, hash.Fnv32),
		watchCenter: watchCenter,
	}

	go cm.startHandleTimeoutRequestWorker(ctx)

	return cm
}

func (c *connManager) AddConn(
	clientId string, files []*apiconfig.ClientConfigFileInfo) chan *apiconfig.ConfigClientResponse {

	finishChan := make(chan *apiconfig.ConfigClientResponse)

	cm.conns.Put(clientId, &connection{
		finishTime:       time.Now().Add(defaultLongPollingTimeout),
		finishChan:       finishChan,
		watchConfigFiles: files,
	})

	c.watchCenter.AddWatcher(clientId, files, func(clientId string, rsp *apiconfig.ConfigClientResponse) bool {
		if conn, ok := cm.conns.Get(clientId); ok {
			conn.finishChan <- rsp
			close(conn.finishChan)
			c.removeConn(clientId)
		}
		return true
	})

	return finishChan
}

func (c *connManager) removeConn(clientId string) {
	conn, ok := cm.conns.Get(clientId)
	if !ok {
		return
	}
	c.watchCenter.RemoveWatcher(clientId, conn.watchConfigFiles)
	cm.conns.Del(clientId)
}

func (c *connManager) startHandleTimeoutRequestWorker(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if cm.conns == nil {
				continue
			}
			tNow := time.Now()
			cm.conns.Range(func(client string, conn *connection) {
				if tNow.After(conn.finishTime) {
					conn.finishChan <- notModifiedResponse
					c.removeConn(client)
				}
			})
		}
	}
}
