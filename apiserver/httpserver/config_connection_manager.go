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
	"context"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"sync"
	"time"
)

const (
	defaultLongPollingTimeout = 30000 * time.Millisecond
)

type connection struct {
	finishTime time.Time
	finishChan chan struct{}
	handler    *Handler
}

type connManager struct {
	conns          *sync.Map //client -> connection
	stopWorkerFunc context.CancelFunc
}

var cm *connManager

// initConnManager 初始化连接管理器，定时响应超时的请求
func initConnManager() {
	cm = &connManager{
		conns: new(sync.Map),
	}

	var ctx context.Context
	ctx, cm.stopWorkerFunc = context.WithCancel(context.Background())

	go startHandleTimeoutRequestWorker(ctx)
}

func (h *HTTPServer) addConn(clientId string, watchConfigFiles []*api.ClientConfigFileInfo,
	handler *Handler, finishChan chan struct{}) {
	cm.conns.Store(clientId, &connection{
		finishTime: time.Now().Add(defaultLongPollingTimeout),
		finishChan: finishChan,
		handler:    handler,
	})

	h.configServer.WatchCenter().AddWatcher(clientId, watchConfigFiles, func(clientId string, rsp *api.ConfigClientResponse) bool {
		conn, ok := cm.conns.Load(clientId)
		if ok {
			c := conn.(*connection)
			c.handler.WriteHeaderAndProto(rsp)
			c.finishChan <- struct{}{}
		}
		return true
	})
}

func (h *HTTPServer) removeConn(clientId string, watchConfigFiles []*api.ClientConfigFileInfo) {
	cm.conns.Delete(clientId)
	h.configServer.WatchCenter().RemoveWatcher(clientId, watchConfigFiles)
}

func startHandleTimeoutRequestWorker(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			cm.conns.Range(func(client, conn interface{}) bool {
				connCtx := conn.(*connection)
				if time.Now().After(connCtx.finishTime) {
					connCtx.finishChan <- struct{}{}
				}
				return true
			})
		}
	}
}

func stopHandleTimeoutRequestWorker() {
	cm.stopWorkerFunc()
}
