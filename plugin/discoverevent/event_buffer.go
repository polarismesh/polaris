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

import "github.com/polarismesh/polaris-server/common/model"

//go:generate gotests -w -all event_buffer.go

type eventBufferHolder struct {
	writeCursor int
	readCursor  int
	size        int
	buffer      []model.DiscoverEvent
}

func newEventBufferHolder(cap int) *eventBufferHolder {
	return &eventBufferHolder{
		writeCursor: 0,
		readCursor:  0,
		size:        0,
		buffer:      make([]model.DiscoverEvent, cap),
	}
}

// Reset 重置 eventBufferHolder，使之可以复用
func (holder *eventBufferHolder) Reset() {
	holder.writeCursor = 0
	holder.readCursor = 0
	holder.size = 0
}

// Put 放入一个 model.DiscoverEvent
func (holder *eventBufferHolder) Put(event model.DiscoverEvent) {
	holder.buffer[holder.writeCursor] = event
	holder.size++
	holder.writeCursor++
}

// HasNext 判断是否还有下一个元素
func (holder *eventBufferHolder) HasNext() bool {
	return holder.readCursor < holder.size
}

// Next 返回下一个元素
//  @return model.DiscoverEvent 元素
//  @return bool 是否还有下一个元素可以继续读取
func (holder *eventBufferHolder) Next() model.DiscoverEvent {

	event := holder.buffer[holder.readCursor]
	holder.readCursor++

	return event
}

// Size 当前所存储的有效元素的个数
func (holder *eventBufferHolder) Size() int {
	return holder.size
}
