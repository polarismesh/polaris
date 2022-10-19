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

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

const (
	defaultLongPollingTimeout = 30000 * time.Millisecond
)

type connection struct {
	finishTime       time.Time
	finishChan       chan *api.ConfigClientResponse
	watchConfigFiles []*api.ClientConfigFileInfo
}

type connManager struct {
	watchCenter    *watchCenter
	conns          *sync.Map // client -> connection
	stopWorkerFunc context.CancelFunc
}

var (
	notModifiedResponse = &api.ConfigClientResponse{
		Code:       utils.NewUInt32Value(api.DataNoChange),
		ConfigFile: nil,
	}

	cm *connManager
)

// NewConfigConnManager 初始化连接管理器，定时响应超时的请求
func NewConfigConnManager(ctx context.Context, watchCenter *watchCenter) *connManager {
	cm = &connManager{
		conns:       new(sync.Map),
		watchCenter: watchCenter,
	}

	go cm.startHandleTimeoutRequestWorker(ctx)

	return cm
}

func (c *connManager) AddConn(clientId string, files []*api.ClientConfigFileInfo) chan *api.ConfigClientResponse {

	finishChan := make(chan *api.ConfigClientResponse)

	cm.conns.Store(clientId, &connection{
		finishTime:       time.Now().Add(defaultLongPollingTimeout),
		finishChan:       finishChan,
		watchConfigFiles: files,
	})

	c.watchCenter.AddWatcher(clientId, files, func(clientId string, rsp *api.ConfigClientResponse) bool {
		connObj, ok := cm.conns.Load(clientId)
		if ok {
			conn := connObj.(*connection)
			conn.finishChan <- rsp
			close(conn.finishChan)
			c.removeConn(clientId)
		}
		return true
	})

	return finishChan
}

func (c *connManager) removeConn(clientId string) {
	conn, ok := cm.conns.Load(clientId)
	if !ok {
		return
	}
	connObj := conn.(*connection)

	c.watchCenter.RemoveWatcher(clientId, connObj.watchConfigFiles)

	cm.conns.Delete(clientId)
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
			cm.conns.Range(func(client, conn interface{}) bool {
				connCtx := conn.(*connection)
				if time.Now().After(connCtx.finishTime) {
					connCtx.finishChan <- notModifiedResponse

					c.removeConn(client.(string))
				}
				return true
			})
		}
	}
}
