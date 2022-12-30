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

func TestInitEventHub(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "init eventhub",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitEventHub()
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
				topic: "test1",
				name:  "subscribe1",
				handler: func(ctx context.Context, i interface{}) error {
					fmt.Printf("handle event=%v\n", i)
					return nil
				},
				eventNum: 100,
			},
		},
	}
	for _, tt := range tests {
		InitEventHub()
		t.Run(tt.name, func(t *testing.T) {
			err := Subscribe(tt.args.topic, tt.args.name, tt.args.handler, tt.args.opts...)
			assert.Nil(t, err)
			for i := 0; i < tt.args.eventNum; i++ {
				Publish(tt.args.topic, i)
			}
			time.Sleep(5 * time.Second)
			fmt.Println("number of goroutines:", runtime.NumGoroutine())
			Shutdown()
		})
	}
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
				topic: "test2",
				name:  "subscribe2",
				handler: func(ctx context.Context, i interface{}) error {
					fmt.Printf("handle event=%v", i)
					return nil
				},
				opts: []SubOption{WithQueueSize(100)},
			},
			wantErr: nil,
		},
	}
	InitEventHub()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Subscribe(tt.args.topic, tt.args.name, tt.args.handler, tt.args.opts...)
			assert.Equal(t, tt.wantErr, err)
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
				topic: "test3",
				name:  "subscribe3",
				handler: func(ctx context.Context, i interface{}) error {
					fmt.Printf("handle event=%v\n", i)
					return nil
				},
			},
			wantErr: nil,
		},
	}
	InitEventHub()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Subscribe(tt.args.topic, tt.args.name, tt.args.handler)
			assert.Nil(t, err)
			Unsubscribe(tt.args.topic, tt.args.name)
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
	InitEventHub()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Shutdown()
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
	InitEventHub()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eh.createTopic(tt.args.name)
			assert.Equal(t, tt.want.name, got.name)
			got = eh.createTopic(tt.args.name)
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
	InitEventHub()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eh.getTopic(tt.args.name)
			assert.Equal(t, tt.want.name, got.name)
		})
	}
}
