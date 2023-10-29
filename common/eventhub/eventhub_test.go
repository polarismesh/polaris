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
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventHub_Init(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "init eventhub",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eh := createEventhub()
			assert.NotNil(t, eh)
		})
	}
}

func TestEventHub_Publish(t *testing.T) {
	type args struct {
		topic    string
		name     string
		handler  Handler
		opts     []SubOption
		eventNum int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "publish event",
			args: args{
				topic:    "test1",
				name:     "subscribe1",
				handler:  &printEventHandler{},
				eventNum: 100,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eh := createEventhub()
			subCtx, err := eh.Subscribe(tt.args.topic, tt.args.handler, tt.args.opts...)
			assert.Nil(t, err)
			for i := 0; i < tt.args.eventNum; i++ {
				eh.Publish(tt.args.topic, i)
			}
			subCtx.Cancel()
			time.Sleep(5 * time.Second)
			fmt.Println("number of goroutines:", runtime.NumGoroutine())
			eh.shutdown()
		})
	}
}

type printEventHandler struct {
}

// PreProcess do preprocess logic for event
func (p *printEventHandler) PreProcess(ctx context.Context, value any) any {
	return value
}

// OnEvent event process logic
func (p *printEventHandler) OnEvent(ctx context.Context, any2 any) error {
	fmt.Printf("handle event=%v", any2)
	return nil
}

func TestEventHub_Subscribe(t *testing.T) {
	type args struct {
		topic   string
		name    string
		handler Handler
		opts    []SubOption
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "subscribe topic",
			args: args{
				topic:   "test2",
				name:    "subscribe2",
				handler: &printEventHandler{},
				opts:    []SubOption{WithQueueSize(100)},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eh := createEventhub()
			subCtx, err := eh.Subscribe(tt.args.topic, tt.args.handler, tt.args.opts...)
			assert.Equal(t, tt.wantErr, err)
			subCtx.Cancel()
		})
	}
}

func TestEventHub_Unsubscribe(t *testing.T) {
	type args struct {
		topic   string
		name    string
		handler Handler
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "unsubscribe topic",
			args: args{
				topic:   "test3",
				name:    "subscribe3",
				handler: &printEventHandler{},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eh := createEventhub()
			subCtx, err := eh.Subscribe(tt.args.topic, tt.args.handler)
			assert.Nil(t, err)
			subCtx.Cancel()
			time.Sleep(2 * time.Second)
		})
	}
}

func TestEventHub_Shutdown(t *testing.T) {
	tests := []struct {
		name         string
		wantTopicNum int
	}{
		{
			name:         "shutdown topic",
			wantTopicNum: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eh := createEventhub()
			eh.shutdown()
			assert.Equal(t, tt.wantTopicNum, len(eh.topics))
		})
	}
}

func TestEventHub_createTopic(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    *topic
		wantErr error
	}{
		{
			name: "create topic",
			args: args{
				name: "topic1",
			},
			want: &topic{
				name: "topic1",
				subs: make(map[string]*subscription),
			},
		},
	}
	eh := createEventhub()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eh.createTopic(tt.args.name, PublishOption{})
			assert.Equal(t, tt.want.name, got.name)
			got = eh.createTopic(tt.args.name, PublishOption{})
			assert.Equal(t, tt.want.name, got.name)
		})
	}
}

func TestEventHub_getTopic(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want *topic
	}{
		{
			name: "get topic exsit",
			args: args{
				name: "topic1",
			},
			want: &topic{
				name: "topic1",
				subs: make(map[string]*subscription),
			},
		},
		{
			name: "get topic not exsit",
			args: args{
				name: "topic2",
			},
			want: &topic{
				name: "topic2",
				subs: make(map[string]*subscription),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eh := createEventhub()
			got := eh.loadOrStoreTopic(tt.args.name)
			assert.Equal(t, tt.want.name, got.name)
		})
	}
}
