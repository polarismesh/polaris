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

	"github.com/stretchr/testify/assert"
)

func Test_newTopic(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want *topic
	}{
		{
			name: "new topic",
			args: args{
				name: "topic1",
			},
			want: &topic{
				name:    "topic1",
				queue:   make(chan Event, defaultQueueSize),
				closeCh: make(chan struct{}),
				subs:    make(map[string]*subscription),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newTopic(tt.args.name)
			assert.Equal(t, tt.want.name, got.name)
		})
	}
}

func Test_topic_publish(t *testing.T) {
	type args struct {
		ctx context.Context
		num int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "publish msg to topic",
			args: args{
				ctx: context.Background(),
				num: 100,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := newTopic("topic1")
			for i := 0; i < tt.args.num; i++ {
				tp.publish(tt.args.ctx, i)
			}
			assert.Equal(t, tt.args.num, len(tp.queue))
		})
	}
}

func Test_topic_subscribe(t *testing.T) {
	type args struct {
		ctx     context.Context
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
			name: "topic subscribe",
			args: args{
				ctx:     context.Background(),
				handler: &printEventHandler{},
				opts:    []SubOption{WithQueueSize(100)},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := newTopic("topic2")
			subCtx, err := tr.subscribe(tt.args.ctx, tt.args.handler, tt.args.opts...)
			assert.Equal(t, tt.wantErr, err)
			subCtx.Cancel()
		})
	}
}

func Test_topic_unsubscribe(t *testing.T) {
	type args struct {
		ctx     context.Context
		name    string
		handler Handler
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "topic unsubscribe",
			args: args{
				ctx:     context.Background(),
				handler: &printEventHandler{},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := newTopic("topic3")
			subCtx, err := tr.subscribe(tt.args.ctx, tt.args.handler)
			assert.NoError(t, err)
			_, ok := tr.subs[subCtx.subID]
			assert.True(t, ok)
			subCtx.Cancel()
			_, ok = tr.subs[subCtx.subID]
			assert.False(t, ok)
		})
	}
}

func Test_topic_run(t *testing.T) {
	type args struct {
		ctx     context.Context
		name    string
		handler Handler
		opts    []SubOption
		num     int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "topic run",
			args: args{
				ctx:     context.Background(),
				name:    "sub1",
				handler: &printEventHandler{},
				opts:    []SubOption{WithQueueSize(100)},
				num:     100,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := newTopic("topic4")
			subCtx, err := tr.subscribe(tt.args.ctx, tt.args.handler, tt.args.opts...)
			assert.Nil(t, err)
			go tr.run(tt.args.ctx)
			for i := 0; i < tt.args.num; i++ {
				tr.publish(tt.args.ctx, i)
			}
			subCtx.Cancel()
			tr.close(tt.args.ctx)
			assert.Zero(t, len(tr.subs))
		})
	}
}
