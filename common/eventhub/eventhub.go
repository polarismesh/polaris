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
	"errors"
	"sync"
)

var (
	initOnce       sync.Once
	globalEventHub *eventHub
)

var (
	ErrorEventhubNotInitialize = errors.New("eventhub not initialize")
)

// InitEventHub initialize event hub
func InitEventHub() {
	initOnce.Do(func() {
		globalEventHub = createEventhub()
	})
}

func createEventhub() *eventHub {
	ctx, cancel := context.WithCancel(context.Background())
	return &eventHub{
		ctx:    ctx,
		cancel: cancel,
		topics: make(map[string]*topic),
	}
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

func RegisterPublisher(topic string, opt PublishOption) error {
	if globalEventHub == nil {
		return ErrorEventhubNotInitialize
	}
	return globalEventHub.RegisterPublisher(topic, opt)
}

func (eh *eventHub) RegisterPublisher(topic string, opt PublishOption) error {
	_ = eh.createTopic(topic, opt)
	return nil
}

// Publish pushlish event to topic
// @param topic Topic name
// @param event Event object
func Publish(topic string, event Event) error {
	if globalEventHub == nil {
		return ErrorEventhubNotInitialize
	}
	return globalEventHub.Publish(topic, event)
}

func (eh *eventHub) Publish(topic string, event Event) error {
	t := eh.loadOrStoreTopic(topic)
	t.publish(eh.ctx, event)
	return nil
}

// Subscribe subscribe topic
func Subscribe(topic string, handler Handler, opts ...SubOption) (*SubscribtionContext, error) {
	if globalEventHub == nil {
		return nil, ErrorEventhubNotInitialize
	}
	return globalEventHub.Subscribe(topic, handler, opts...)
}

// SubscribeWithFunc subscribe topic use func
func SubscribeWithFunc(topic string, handler HandlerFunc, opts ...SubOption) (*SubscribtionContext, error) {
	if globalEventHub == nil {
		return nil, ErrorEventhubNotInitialize
	}
	return globalEventHub.Subscribe(topic, &funcSubscriber{
		handlerFunc: handler,
	}, opts...)
}

func (e *eventHub) Subscribe(topic string, handler Handler,
	opts ...SubOption) (*SubscribtionContext, error) {
	t := e.loadOrStoreTopic(topic)
	return t.subscribe(e.ctx, handler, opts...)
}

// Shutdown shutdown event hub
func Shutdown() {
	if globalEventHub == nil {
		return
	}
	globalEventHub.shutdown()
}

func (e *eventHub) shutdown() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.cancel()
	for _, t := range e.topics {
		t.close(e.ctx)
		delete(e.topics, t.name)
	}
}

func (e *eventHub) createTopic(name string, opt PublishOption) *topic {
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

func (e *eventHub) loadOrStoreTopic(name string) *topic {
	e.mu.RLock()
	if t, ok := e.topics[name]; ok {
		e.mu.RUnlock()
		return t
	}
	e.mu.RUnlock()
	return e.createTopic(name, PublishOption{})
}
