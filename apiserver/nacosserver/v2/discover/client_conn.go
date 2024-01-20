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

package discover

import (
	"context"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/remote"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/model"
)

type (
	// ConnectionClientManager
	ConnectionClientManager struct {
		lock        sync.RWMutex
		clients     map[string]*ConnectionClient // ConnID => ConnectionClient
		inteceptors []ClientConnectionInterceptor

		watchCtx *eventhub.SubscribtionContext
	}

	// ClientConnectionInterceptor
	ClientConnectionInterceptor interface {
		// HandleClientConnect .
		HandleClientConnect(ctx context.Context, client *remote.Client)
		// HandleClientDisConnect .
		HandleClientDisConnect(ctx context.Context, client *remote.Client)
	}

	// ConnectionClient .
	ConnectionClient struct {
		// ConnID 物理连接唯一ID标识
		ConnID string
		lock   sync.RWMutex
		// PublishInstances 这个连接上发布的实例信息
		PublishInstances map[model.ServiceKey]map[string]struct{}
		destroy          int32
	}
)

func NewConnectionClientManager(inteceptors []ClientConnectionInterceptor) (*ConnectionClientManager, error) {
	mgr := &ConnectionClientManager{
		clients:     map[string]*ConnectionClient{},
		inteceptors: inteceptors,
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
		nacoslog.Info("[NACOS-V2][Naming] receive client connected event", zap.String("clientId", event.ConnID))
		for i := range c.inteceptors {
			c.inteceptors[i].HandleClientConnect(ctx, event.Client)
		}
	case remote.EventClientDisConnected:
		nacoslog.Info("[NACOS-V2][Naming] receive client disconnected event", zap.String("clientId", event.ConnID))
		for i := range c.inteceptors {
			c.inteceptors[i].HandleClientDisConnect(ctx, event.Client)
		}
	}

	return nil
}

func (c *ConnectionClientManager) getClient(connID string) (*ConnectionClient, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	client, ok := c.clients[connID]
	return client, ok
}

func (c *ConnectionClientManager) delClient(connID string) (*ConnectionClient, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	client, ok := c.clients[connID]
	delete(c.clients, connID)
	return client, ok
}

func (c *ConnectionClientManager) addServiceInstance(connID string, svc model.ServiceKey, instanceIDS ...string) {
	c.addConnectionClientIfAbsent(connID)
	c.lock.RLock()
	defer c.lock.RUnlock()
	client := c.clients[connID]
	client.addServiceInstance(svc, instanceIDS...)
}

func (c *ConnectionClientManager) delServiceInstance(connID string, svc model.ServiceKey, instanceIDS ...string) {
	c.addConnectionClientIfAbsent(connID)
	c.lock.RLock()
	defer c.lock.RUnlock()
	client := c.clients[connID]
	client.delServiceInstance(svc, instanceIDS...)
}

func (c *ConnectionClientManager) addConnectionClientIfAbsent(connID string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, ok := c.clients[connID]; !ok {
		client := &ConnectionClient{
			ConnID:           connID,
			PublishInstances: make(map[model.ServiceKey]map[string]struct{}),
		}
		c.clients[connID] = client
	}
}

func (c *ConnectionClient) RangePublishInstance(f func(svc model.ServiceKey, ids []string)) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	for svc, ids := range c.PublishInstances {
		ret := make([]string, 0, 16)
		for i := range ids {
			ret = append(ret, i)
		}
		f(svc, ret)
	}
}

func (c *ConnectionClient) addServiceInstance(svc model.ServiceKey, instanceIDS ...string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, ok := c.PublishInstances[svc]; !ok {
		c.PublishInstances[svc] = map[string]struct{}{}
	}
	publishInfos := c.PublishInstances[svc]

	for i := range instanceIDS {
		publishInfos[instanceIDS[i]] = struct{}{}
	}
	c.PublishInstances[svc] = publishInfos
}

func (c *ConnectionClient) delServiceInstance(svc model.ServiceKey, instanceIDS ...string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, ok := c.PublishInstances[svc]; !ok {
		c.PublishInstances[svc] = map[string]struct{}{}
	}
	publishInfos := c.PublishInstances[svc]

	for i := range instanceIDS {
		delete(publishInfos, instanceIDS[i])
	}
	c.PublishInstances[svc] = publishInfos
}

func (c *ConnectionClient) Destroy() {
	atomic.StoreInt32(&c.destroy, 1)
}

func (c *ConnectionClient) isDestroy() bool {
	return atomic.LoadInt32(&c.destroy) == 1
}
