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

	"github.com/polarismesh/polaris/common/log"
)

const (
	defaultQueueSize = 16384
)

type funcSubscriber struct {
	handlerFunc HandlerFunc
}

// PreProcess do preprocess logic for event
func (s *funcSubscriber) PreProcess(_ context.Context, a any) any {
	return a
}

// OnEvent event process logic
func (s *funcSubscriber) OnEvent(ctx context.Context, event any) error {
	return s.handlerFunc(ctx, event)
}

type HandlerFunc func(ctx context.Context, any2 any) error

// Handler event handler
type Handler interface {
	// PreProcess do preprocess logic for event
	PreProcess(context.Context, any) any
	// OnEvent event process logic
	OnEvent(ctx context.Context, any2 any) error
}

type SubscribtionContext struct {
	subID  string
	cancel context.CancelFunc
}

func (s *SubscribtionContext) Cancel() {
	s.cancel()
}

// Subscription subscription info
type subscription struct {
	name    string
	queue   chan Event
	closeCh chan struct{}
	handler Handler
	opts    *SubOptions
}

func newSubscription(name string, handler Handler, opts ...SubOption) *subscription {
	subOpts := &SubOptions{
		QueueSize: defaultQueueSize,
	}
	for _, o := range opts {
		o(subOpts)
	}
	if subOpts.QueueSize == 0 {
		subOpts.QueueSize = defaultQueueSize
	}
	sub := &subscription{
		name:    name,
		queue:   make(chan Event, subOpts.QueueSize),
		closeCh: make(chan struct{}),
		handler: handler,
		opts:    subOpts,
	}
	return sub
}

func (s *subscription) send(ctx context.Context, event Event) {
	select {
	case s.queue <- event:
		if log.DebugEnabled() {
			log.Debugf("[EventHub] subscription:%s send event:%v", s.name, event)
		}
	case <-s.closeCh:
		log.Infof("[EventHub] subscription:%s send close", s.name)
		return
	case <-ctx.Done():
		log.Infof("[EventHub] subscription:%s send close by context cancel", s.name)
		return
	}
	return
}

func (s *subscription) receive(ctx context.Context) {
	for {
		select {
		case event := <-s.queue:
			if log.DebugEnabled() {
				log.Debugf("[EventHub] subscription:%s receive event:%v", s.name, event)
			}
			event = s.handler.PreProcess(ctx, event)
			if err := s.handler.OnEvent(ctx, event); err != nil {
				log.Errorf("[EventHub] subscriptions:%s handler event error:%s", s.name, err.Error())
			}
		case <-s.closeCh:
			log.Infof("[EventHub] subscription:%s receive close", s.name)
			return
		case <-ctx.Done():
			log.Infof("[EventHub] subscription:%s receive close by context cancel", s.name)
			return
		}
	}
}

func (s *subscription) close() {
	close(s.closeCh)
}

// SubOption subscription option func
type SubOption func(s *SubOptions)

// SubOptions subscripion options
type SubOptions struct {
	QueueSize int
}

// WithQueueSize set event queue size
func WithQueueSize(size int) SubOption {
	return func(s *SubOptions) {
		s.QueueSize = size
	}
}

// PublishOption .
type PublishOption struct {
	WaitHaveSub bool
}
