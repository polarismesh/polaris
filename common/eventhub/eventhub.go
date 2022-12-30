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

package eventhub

import (
	"context"
	"sync"
)

var (
	once sync.Once
	eh   *eventHub
)

// InitEventHub initialize event hub
func InitEventHub() {
	once.Do(func() {
		eh = &eventHub{
			topics: make(map[string]*topic),
		}
		eh.ctx, eh.cancel = context.WithCancel(context.Background())
	})
}

// Event evnt type
type Event interface{}

// eventHub event hub
type eventHub struct {
	ctx    context.Context
	cancel context.CancelFunc
	topics map[string]*topic
	mu     sync.RWMutex
}

// Publish pushlish event to topic
// @param topic Topic name
// @param event Event object
func Publish(topic string, event Event) {
	t := eh.getTopic(topic)
	t.publish(eh.ctx, event)
}

// Subscribe subscribe topic
// @param topic Topic name
// @param name Subscribe name
// @param handler Message handler
// @param opts Subscription options
// @return error Subscribe failed, return error
func Subscribe(topic string, name string, handler Handler, opts ...SubOption) error {
	t := eh.getTopic(topic)
	return t.subscribe(eh.ctx, name, handler, opts...)
}

// Unsubscribe unsubscribe topic
// @param topic Topic name
// @param name Subscribe name
func Unsubscribe(topic string, name string) {
	t := eh.getTopic(topic)
	t.unsubscribe(eh.ctx, name)
}

// Shutdown shutdown event hub
func Shutdown() {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	eh.cancel()

	for _, t := range eh.topics {
		t.close(eh.ctx)
		delete(eh.topics, t.name)
	}
}

func (e *eventHub) createTopic(name string) *topic {
	e.mu.Lock()
	defer e.mu.Unlock()
	if t, ok := e.topics[name]; ok {
		return t
	}
	t := newTopic(name)
	e.topics[name] = t
	go t.run(e.ctx)
	return t
}

func (e *eventHub) getTopic(name string) *topic {
	e.mu.RLock()
	if t, ok := e.topics[name]; ok {
		e.mu.RUnlock()
		return t
	}
	e.mu.RUnlock()
	return e.createTopic(name)
}
