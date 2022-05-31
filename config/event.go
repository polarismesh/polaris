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
	"sync"

	"github.com/polarismesh/polaris-server/common/log"
	"go.uber.org/zap"
)

// Event 事件对象，包含类型和事件消息
type Event struct {
	EventType string
	Message   interface{}
}

// Callback 事件回调
type Callback func(event Event) bool

// Center 事件中心
type Center struct {
	watchers map[string]callbackBucket
	lock     sync.RWMutex
}

type callbackBucket struct {
	cbs  []Callback
	lock *sync.RWMutex // this lock cannot be copied
}

func (c *callbackBucket) add(cb Callback) {
	c.lock.Lock()
	c.cbs = append(c.cbs, cb)
	c.lock.Unlock()
}

func (c *callbackBucket) getCbs() []Callback {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.cbs
}

// NewEventCenter 新建事件中心
func NewEventCenter() *Center {
	center := &Center{
		watchers: make(map[string]callbackBucket),
	}

	return center
}

// WatchEvent 监听事件
func (c *Center) WatchEvent(eventType string, cb Callback) {
	c.lock.Lock()
	defer c.lock.Unlock()

	callback, ok := c.watchers[eventType]
	if !ok {
		callback = callbackBucket{
			cbs:  make([]Callback, 0, 6),
			lock: &sync.RWMutex{},
		}
	}

	callback.add(cb)
	c.watchers[eventType] = callback
}

func (c *Center) handleEvent(e Event) {
	defer c.recovery()

	// get map value
	c.lock.RLock()
	callback, ok := c.watchers[e.EventType]
	if !ok {
		c.lock.RUnlock()
		return
	}

	c.lock.RUnlock()

	for _, cb := range callback.getCbs() {
		if !cb(e) {
			log.ConfigScope().Errorf("[Common][Event] cb message error. event = %+v", e)
		}
	}
}

func (c *Center) recovery() {
	if err := recover(); err != nil {
		log.ConfigScope().Error("[Common][Event] handler event error.", zap.Any("error", err))
	}
}
