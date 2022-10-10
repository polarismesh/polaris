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

package loki

//go:generate gotests -w -all history_loki.go

import (
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/plugin"
)

// 把操作记录记录到Loki
const (
	// PluginName plugin name
	PluginName       string = "HistoryLoki"
	defaultBatchSize int    = 512
	defaultQueueSize int    = 1024
)

func init() {
	h := &HistoryLoki{}
	plugin.RegisterPlugin(h.Name(), h)
}

type HistoryLoki struct {
	entryCh chan *model.RecordEntry
	stopCh  chan struct{}
	logger  *LokiLogger
}

// Name 插件名称
// @return string 返回插件名称
func (h *HistoryLoki) Name() string {
	return PluginName
}

// Initialize 根据配置文件进行初始化插件 HistoryLoki
// @param conf 配置文件内容
// @return error 初始化失败，返回 error 信息
func (h *HistoryLoki) Initialize(conf *plugin.ConfigEntry) error {
	var queueSize = defaultQueueSize
	if val, ok := conf.Option["queueSize"]; ok {
		queueSize, _ = val.(int)
	}
	lokiLogger, err := newLokiLogger(conf.Option)
	if err != nil {
		return err
	}
	h.logger = lokiLogger
	h.entryCh = make(chan *model.RecordEntry, queueSize)
	h.stopCh = make(chan struct{})
	go h.Run()
	return nil
}

// Destroy 执行插件销毁
func (h *HistoryLoki) Destroy() error {
	close(h.stopCh)
	return nil
}

// Record 记录操作记录
func (h *HistoryLoki) Record(entry *model.RecordEntry) {
	select {
	case h.entryCh <- entry:
		return
	default:
		// do nothing
	}
}

// Run 执行主逻辑
func (h *HistoryLoki) Run() {
	// 定时刷新事件到日志的定时器
	syncInterval := time.NewTicker(time.Duration(10) * time.Second)
	defer syncInterval.Stop()

	batch := make([]*model.RecordEntry, 0, defaultBatchSize)
	batchSize := 0

	for {
		select {
		case entry := <-h.entryCh:
			batch = append(batch, entry)
			batchSize++
			// 触发批量生产发送 log 阈值
			if batchSize == defaultBatchSize {
				h.logger.Log(batch[:batchSize])
				batch = make([]*model.RecordEntry, 0, defaultBatchSize)
				batchSize = 0
			}
		case <-syncInterval.C:
			if batchSize > 0 {
				h.logger.Log(batch[:batchSize])
				batch = make([]*model.RecordEntry, 0, defaultBatchSize)
				batchSize = 0
			}
		case <-h.stopCh:
			if batchSize > 0 {
				h.logger.Log(batch[:batchSize])
			}
			return
		}
	}
}
