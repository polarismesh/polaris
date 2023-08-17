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

	"github.com/google/uuid"

	"github.com/polarismesh/polaris/common/log"
)

//go:generate gotests -w -all topic.go

type topic struct {
	name    string
	queue   chan Event
	closeCh chan struct{}
	subs    map[string]*subscription
	mu      sync.RWMutex
}

func newTopic(name string) *topic {
	t := &topic{
		name:    name,
		queue:   make(chan Event, defaultQueueSize),
		closeCh: make(chan struct{}),
		subs:    make(map[string]*subscription),
	}
	return t
}

// publish publish msg to topic
func (t *topic) publish(ctx context.Context, event Event) {
	if log.DebugEnabled() {
		log.Debugf("[EventHub] publish topic:%s, event:%v", t.name, event)
	}
	t.queue <- event
}

// subscribe subscribe msg from topic
func (t *topic) subscribe(ctx context.Context, handler Handler,
	opts ...SubOption) (*SubscribtionContext, error) {

	subID := uuid.NewString()
	sub := newSubscription(subID, handler, opts...)

	t.mu.Lock()
	defer t.mu.Unlock()
	t.subs[subID] = sub

	newCtx, cancel := context.WithCancel(ctx)
	subscribtionCtx := &SubscribtionContext{
		subID: subID,
		cancel: func() {
			cancel()
			t.unsubscribe(subID)
		},
	}

	go sub.receive(newCtx)
	return subscribtionCtx, nil
}

// unsubscribe unsubscrib msg from topic
func (t *topic) unsubscribe(name string) {
	sub, ok := t.subs[name]
	if !ok {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.subs, sub.name)
}

// close close topic
func (t *topic) close(ctx context.Context) {
	t.mu.Lock()
	defer t.mu.Unlock()
	close(t.closeCh)
	for _, sub := range t.subs {
		sub.close()
		delete(t.subs, sub.name)
	}
}

// run read msg from topic queue and send to all subscription
func (t *topic) run(ctx context.Context) {
	log.Infof("[EventHub] topic:%s run dispatch", t.name)
	for {
		select {
		case msg := <-t.queue:
			func() {
				subs := t.listSubscribers()
				for i := range subs {
					sub := subs[i]
					go sub.send(ctx, msg)
				}
			}()
		case <-t.closeCh:
			log.Infof("[EventHub] topic:%s run stop", t.name)
			return
		case <-ctx.Done():
			log.Infof("[EventHub] topic:%s run stop by context cancel", t.name)
			return
		}
	}
}

func (t *topic) listSubscribers() []*subscription {
	t.mu.RLock()
	defer t.mu.RUnlock()
	ret := make([]*subscription, 0, len(t.subs))
	for _, sub := range t.subs {
		ret = append(ret, sub)
	}
	return ret
}
