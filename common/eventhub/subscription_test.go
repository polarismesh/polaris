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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type noopEventHandler struct {
}

// PreProcess do preprocess logic for event
func (p *noopEventHandler) PreProcess(ctx context.Context, value any) any {
	return value
}

// OnEvent event process logic
func (p *noopEventHandler) OnEvent(ctx context.Context, any2 any) error {
	return nil
}

func Test_newSubscription(t *testing.T) {
	type args struct {
		name    string
		handler Handler
		opts    []SubOption
	}
	tests := []struct {
		name string
		args args
		want *subscription
	}{
		{
			name: "new sub",
			args: args{
				name:    "sub1",
				handler: &noopEventHandler{},
				opts:    []SubOption{WithQueueSize(100)},
			},
			want: &subscription{
				name:    "sub1",
				queue:   make(chan Event, 100),
				closeCh: make(chan struct{}),
				handler: &noopEventHandler{},
				opts: &SubOptions{
					QueueSize: 100,
				},
			},
		},
		{
			name: "new sub no sub option",
			args: args{
				name:    "sub2",
				handler: &noopEventHandler{},
			},
			want: &subscription{
				name:    "sub2",
				queue:   make(chan Event, defaultQueueSize),
				closeCh: make(chan struct{}),
				handler: &noopEventHandler{},
				opts: &SubOptions{
					QueueSize: defaultQueueSize,
				},
			},
		},
		{
			name: "new sub with invalid sub option",
			args: args{
				name:    "sub3",
				handler: &noopEventHandler{},
				opts:    []SubOption{WithQueueSize(0)},
			},
			want: &subscription{
				name:    "sub3",
				queue:   make(chan Event, defaultQueueSize),
				closeCh: make(chan struct{}),
				handler: &noopEventHandler{},
				opts: &SubOptions{
					QueueSize: defaultQueueSize,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newSubscription(tt.args.name, tt.args.handler, tt.args.opts...)
			assert.Equal(t, tt.want.name, got.name)
			assert.Equal(t, tt.want.opts, got.opts)
		})
	}
}

func Test_subscription_send(t *testing.T) {
	type args struct {
		ctx context.Context
		num int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "send msg to subscription",
			args: args{
				ctx: context.Background(),
				num: 1000,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newSubscription("sub1", &noopEventHandler{})
			for i := 0; i < tt.args.num; i++ {
				s.send(tt.args.ctx, i)
			}
			assert.Equal(t, tt.args.num, len(s.queue))
		})
	}
}

type counterEventHandler struct {
	got int
}

// PreProcess do preprocess logic for event
func (p *counterEventHandler) PreProcess(ctx context.Context, value any) any {
	return value
}

// OnEvent event process logic
func (p *counterEventHandler) OnEvent(ctx context.Context, any2 any) error {
	p.got++
	return nil
}

func Test_subscription_receive(t *testing.T) {
	type args struct {
		ctx context.Context
		num int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "subscription receive msg",
			args: args{
				ctx: context.Background(),
				num: 1000,
			},
			want: 1000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &counterEventHandler{}
			s := newSubscription("sub1", handler)
			go func() {
				for i := 0; i < tt.args.num; i++ {
					s.send(tt.args.ctx, i)
				}
			}()
			go s.receive(tt.args.ctx)
			time.Sleep(2 * time.Second)
			s.close()
			assert.Equal(t, tt.want, handler.got)
		})
	}
}
