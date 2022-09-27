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

package discoverevent

import (
	"testing"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/stretchr/testify/assert"
)

func Test_newEventBufferHolder(t *testing.T) {
	type args struct {
		cap int
	}
	tests := []struct {
		name string
		args args
		want *eventBufferHolder
	}{
		{
			name: "new event buffer holder",
			args: args{
				cap: 10,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newEventBufferHolder(tt.args.cap)
			assert.NotNil(t, got)
		})
	}
}

func Test_eventBufferHolder_Reset(t *testing.T) {
	type fields struct {
		writeCursor int
		readCursor  int
		size        int
		buffer      []model.DiscoverEvent
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "reset event buffer holder",
			fields: fields{
				writeCursor: 10,
				readCursor:  5,
				size:        10,
				buffer:      make([]model.DiscoverEvent, 20),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			holder := &eventBufferHolder{
				writeCursor: tt.fields.writeCursor,
				readCursor:  tt.fields.readCursor,
				size:        tt.fields.size,
				buffer:      tt.fields.buffer,
			}
			holder.Reset()
			assert.Equal(t, 0, holder.writeCursor)
			assert.Equal(t, 0, holder.readCursor)
			assert.Equal(t, 0, holder.size)
		})
	}
}

func Test_eventBufferHolder_Put(t *testing.T) {
	type args struct {
		event model.DiscoverEvent
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "put event",
			args: args{
				event: model.DiscoverEvent{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			holder := newEventBufferHolder(10)
			holder.Put(tt.args.event)
			assert.Equal(t, 1, holder.writeCursor)
			assert.Equal(t, 1, holder.size)
		})
	}
}

func Test_eventBufferHolder_HasNext(t *testing.T) {
	type fields struct {
		writeCursor int
		readCursor  int
		size        int
		buffer      []model.DiscoverEvent
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "has next",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			holder := newEventBufferHolder(10)
			got := holder.HasNext()
			assert.Equal(t, false, got)
			holder.Put(model.DiscoverEvent{})
			got = holder.HasNext()
			assert.Equal(t, true, got)
		})
	}
}

func Test_eventBufferHolder_Next(t *testing.T) {
	type fields struct {
		writeCursor int
		readCursor  int
		size        int
		buffer      []model.DiscoverEvent
	}
	tests := []struct {
		name   string
		fields fields
		want   model.DiscoverEvent
	}{
		{
			name: "next",
			want: model.DiscoverEvent{
				Namespace: "test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			holder := newEventBufferHolder(10)
			holder.Put(model.DiscoverEvent{Namespace: "test"})
			got := holder.Next()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_eventBufferHolder_Size(t *testing.T) {
	type fields struct {
		writeCursor int
		readCursor  int
		size        int
		buffer      []model.DiscoverEvent
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "size",
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			holder := newEventBufferHolder(10)
			holder.Put(model.DiscoverEvent{})
			holder.Put(model.DiscoverEvent{})
			got := holder.Size()
			assert.Equal(t, tt.want, got)
		})
	}
}
