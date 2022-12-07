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

package logger

import (
	commonLog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

// 把操作记录记录到日志文件中
const (
	// PluginName plugin name
	PluginName = "HistoryLogger"
)

var log = commonLog.RegisterScope(PluginName, "", 0)

// init 初始化注册函数
func init() {
	plugin.RegisterPlugin(PluginName, &HistoryLogger{})
}

// HistoryLogger 历史记录logger
type HistoryLogger struct {
}

// Name 返回插件名字
func (h *HistoryLogger) Name() string {
	return PluginName
}

// Destroy 销毁插件
func (h *HistoryLogger) Destroy() error {
	return log.Sync()
}

// Initialize 插件初始化
func (h *HistoryLogger) Initialize(c *plugin.ConfigEntry) error {
	return nil
}

// Record 记录操作记录到日志中
func (h *HistoryLogger) Record(entry *model.RecordEntry) {
	entry.Server = utils.LocalHost
	log.Info(entry.String())
}
