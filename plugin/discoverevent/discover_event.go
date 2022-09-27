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
	"sync"
	"time"

	json "github.com/json-iterator/go"
	"github.com/pkg/errors"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/plugin"
)

//go:generate gotests -w -all discover_event.go

const (
	PluginName        string = "discoverEvent"
	defaultBufferSize int    = 1024
)

func init() {
	d := &discoverEvent{}
	plugin.RegisterPlugin(d.Name(), d)
}

type discoverEvent struct {
	eventCh        chan model.DiscoverEvent
	eventLoggers   []Logger
	bufferPool     *sync.Pool
	curEventBuffer *eventBufferHolder
	cursor         int
	syncLock       *sync.Mutex
}

// @return string 返回插件名称
func (d *discoverEvent) Name() string {
	return PluginName
}

// Initialize 根据配置文件进行初始化插件 discoverEventLocal
// @param conf 配置文件内容
// @return error 初始化失败，返回 error 信息
func (d *discoverEvent) Initialize(conf *plugin.ConfigEntry) error {
	confBytes, err := json.Marshal(conf.Option)
	if err != nil {
		return err
	}
	config := &model.DiscoverEventConfig{}
	if err := json.Unmarshal(confBytes, config); err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return err
	}

	d.eventCh = make(chan model.DiscoverEvent, config.QueueSize)

	for _, cfg := range config.LoggerConfigs {
		logger, err := d.newLogger(cfg)
		if err != nil {
			return err
		}
		d.eventLoggers = append(d.eventLoggers, logger)
	}

	d.syncLock = &sync.Mutex{}
	d.bufferPool = &sync.Pool{
		New: func() interface{} {
			return newEventBufferHolder(defaultBufferSize)
		},
	}

	d.switchEventBuffer()

	go d.Run()
	return nil
}

// Destroy 执行插件销毁
func (d *discoverEvent) Destroy() error {
	return nil
}

// PublishEvent 发布一个服务事件
func (d *discoverEvent) PublishEvent(event model.DiscoverEvent) {
	select {
	case d.eventCh <- event:
		return
	default:
		// do nothing
	}
}

// Run 执行主逻辑
func (d *discoverEvent) Run() {

	// 定时刷新事件到日志的定时器
	syncInterval := time.NewTicker(time.Duration(10) * time.Second)

	for {
		select {
		case event := <-d.eventCh:
			// 确保事件是顺序的
			now := time.Now()
			event.CreateTime = now
			// event.CreateTimeSec = now.Unix()
			d.curEventBuffer.Put(event)

			// 触发持久化到 log 阈值
			if d.curEventBuffer.Size() == defaultBufferSize {
				events := d.getEvents()
				for _, logger := range d.eventLoggers {
					go logger.Log(events)
				}
				d.switchEventBuffer()
			}
		case <-syncInterval.C:
			events := d.getEvents()
			if len(events) > 0 {
				for _, logger := range d.eventLoggers {
					go logger.Log(events)
				}
			}
			d.switchEventBuffer()
		}
	}
}

// switchEventBuffer 换一个新的 buffer 实例继续使用
func (d *discoverEvent) switchEventBuffer() {
	d.curEventBuffer = d.bufferPool.Get().(*eventBufferHolder)
}

// newLogger 根据配置新建事件 logger
func (d *discoverEvent) newLogger(cfg model.DiscoverEventLoggerConfig) (Logger, error) {
	switch cfg.Name {
	case LocalLoggerName:
		return newLocalLogger(cfg.Option)
	case LokiLoggerName:
		return newLokiLogger(cfg.Option)
	default:
		return nil, errors.Errorf("Invalid logger name %s", cfg.Name)
	}
}

// getEvents 获取 buffer 中的 discover events
func (d *discoverEvent) getEvents() []model.DiscoverEvent {
	defer func() {
		d.curEventBuffer.Reset()
		d.bufferPool.Put(d.curEventBuffer)
	}()

	var events []model.DiscoverEvent
	for d.curEventBuffer.HasNext() {
		events = append(events, d.curEventBuffer.Next())
	}
	return events
}
