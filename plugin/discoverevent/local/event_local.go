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

package local

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	commonLog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

const (
	PluginName        = "discoverEventLocal"
	defaultBufferSize = 1024
)

var log = commonLog.RegisterScope(PluginName, "", 0)

func init() {
	d := &discoverEventLocal{}
	plugin.RegisterPlugin(d.Name(), d)
}

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
//
//	@return model.DiscoverEvent 元素
//	@return bool 是否还有下一个元素可以继续读取
func (holder *eventBufferHolder) Next() model.DiscoverEvent {

	event := holder.buffer[holder.readCursor]
	holder.readCursor++

	return event
}

// Size 当前所存储的有效元素的个数
func (holder *eventBufferHolder) Size() int {
	return holder.size
}

type discoverEventLocal struct {
	eventCh        chan model.DiscoverEvent
	bufferPool     sync.Pool
	curEventBuffer *eventBufferHolder
	cursor         int
	syncLock       sync.Mutex
}

// Name 插件名称
// @return string 返回插件名称
func (el *discoverEventLocal) Name() string {
	return PluginName
}

// Initialize 根据配置文件进行初始化插件 discoverEventLocal
// @param conf 配置文件内容
// @return error 初始化失败，返回 error 信息
func (el *discoverEventLocal) Initialize(conf *plugin.ConfigEntry) error {
	contentBytes, err := json.Marshal(conf.Option)
	if err != nil {
		return err
	}

	config := model.DefaultDiscoverEventConfig()
	if err := json.Unmarshal(contentBytes, config); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}

	el.eventCh = make(chan model.DiscoverEvent, config.QueueSize)

	el.bufferPool = sync.Pool{
		New: func() interface{} {
			return newEventBufferHolder(defaultBufferSize)
		},
	}

	el.switchEventBuffer()

	go el.Run()

	return nil
}

// Destroy 执行插件销毁
func (el *discoverEventLocal) Destroy() error {
	return nil
}

// PublishEvent 发布一个服务事件
func (el *discoverEventLocal) PublishEvent(event model.DiscoverEvent) {
	select {
	case el.eventCh <- event:
		return
	default:
		// do nothing
	}
}

// Run 执行主逻辑
func (el *discoverEventLocal) Run() {
	// 定时刷新事件到日志的定时器
	syncInterval := time.NewTicker(time.Duration(10) * time.Second)
	defer syncInterval.Stop()

	for {
		select {
		case event := <-el.eventCh:
			// 确保事件是顺序的
			event.CreateTime = time.Now()
			el.curEventBuffer.Put(event)

			// 触发持久化到 log 阈值
			if el.curEventBuffer.Size() == defaultBufferSize {
				go el.writeToFile(el.curEventBuffer)

				el.switchEventBuffer()
			}
		case <-syncInterval.C:
			go el.writeToFile(el.curEventBuffer)
			el.switchEventBuffer()
		}
	}
}

// switchEventBuffer 换一个新的 buffer 实例继续使用
func (el *discoverEventLocal) switchEventBuffer() {
	el.curEventBuffer = el.bufferPool.Get().(*eventBufferHolder)
}

// writeToFile 事件落盘
func (el *discoverEventLocal) writeToFile(eventHolder *eventBufferHolder) {
	el.syncLock.Lock()
	defer func() {
		el.syncLock.Unlock()
		eventHolder.Reset()
		el.bufferPool.Put(eventHolder)
	}()

	for eventHolder.HasNext() {
		event := eventHolder.Next()
		log.Info(fmt.Sprintf(
			"%s|%s|%s|%d|%s|%d|%s",
			event.Namespace,
			event.Service,
			event.Host,
			event.Port,
			event.EType,
			event.CreateTime.Unix(),
			utils.LocalHost))
	}
}
