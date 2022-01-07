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

package config

import (
	"github.com/polarismesh/polaris-server/common/log"
	"sync"
)

const (
	EventTypePublishConfigFile = "PublishConfigFile"
)

// Event 事件对象，包含类型和事件消息
type Event struct {
	EventType string
	Message   interface{}
}

// Center 事件中心
type Center struct {
	watchers *sync.Map
	lock     *sync.Mutex
}

// NewEventCenter 新建事件中心
func NewEventCenter() *Center {
	return &Center{
		watchers: new(sync.Map),
		lock:     new(sync.Mutex),
	}
}

// FireEvent 发布一个事件
func (c *Center) FireEvent(event Event) {
	log.GetConfigLogger().Infof("[Config][Event] fire event.")

	handlers, ok := c.watchers.Load(event.EventType)
	if !ok {
		return
	}

	handlerArr := handlers.([]func(event Event) bool)
	for _, handler := range handlerArr {
		h := handler
		go func() {
			ok := h(event)
			if !ok {
				log.GetConfigLogger().Errorf("[Config][Event] handler message error. event = %+v", event)
			}
		}()
	}
}

// WatchEvent 监听事件
func (c *Center) WatchEvent(eventType string, handler func(event Event) bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	handlers, ok := c.watchers.Load(eventType)
	if !ok {
		handlers = []func(event Event) bool{handler}
		c.watchers.Store(eventType, handlers)
	} else {
		handlerArr := handlers.([]func(event Event) bool)
		handlerArr = append(handlerArr, handler)
	}
}
