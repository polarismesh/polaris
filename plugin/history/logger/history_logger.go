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
	"fmt"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/polarismesh/polaris-server/common/model"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/plugin"
)

// 把操作记录记录到日志文件中
const (
	// PluginName plugin name
	PluginName = "HistoryLogger"
)

// init 初始化注册函数
func init() {
	plugin.RegisterPlugin(PluginName, &HistoryLogger{})
}

// HistoryLogger 历史记录logger
type HistoryLogger struct {
	logger *zap.Logger
}

// Name 返回插件名字
func (h *HistoryLogger) Name() string {
	return PluginName
}

// Destroy 销毁插件
func (h *HistoryLogger) Destroy() error {
	return h.logger.Sync()
}

// Initialize 插件初始化
func (h *HistoryLogger) Initialize(c *plugin.ConfigEntry) error {
	// 日志的encode
	encCfg := zapcore.EncoderConfig{
		TimeKey: "time",
		// LevelKey:       "level",
		NameKey:        "scope",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stack",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		// EncodeTime:     TimeEncoder,
	}

	// 同步到文件中的配置 TODO，参数来自于外部配置文件
	log := &lumberjack.Logger{
		Filename:   "./log/polaris-history.log", // TODO
		MaxSize:    100,                         // megabytes TODO
		MaxBackups: 50,
		MaxAge:     15, // days TODO
		LocalTime:  true,
	}
	go func() {
		duration := 24 * time.Hour
		ticker := time.NewTicker(duration)
		for {
			<-ticker.C
			log.Rotate()
		}
	}()
	w := zapcore.AddSync(log)
	// multiSync := zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), w)

	// 日志
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(encCfg), w, zap.DebugLevel)
	logger := zap.New(core)
	h.logger = logger

	return nil
}

// Record 记录操作记录到日志中
func (h *HistoryLogger) Record(entry *model.RecordEntry) {
	var str string
	switch model.GetResourceType(entry.ResourceType) {
	case model.ServiceType:
		str = fmt.Sprintf("resource_type=%s;operation_type=%s;namespace=%s;service=%s;context=%s;operator=%s;ctime=%s",
			string(entry.ResourceType), string(entry.OperationType), entry.Namespace, entry.Service,
			entry.Context, entry.Operator, commontime.Time2String(entry.CreateTime))
	case model.MeshType:
		str = fmt.Sprintf(
			"resource_type=%s;operation_type=%s;mesh_id=%s;revision=%s;context=%s;operator=%s;ctime=%s",
			string(entry.ResourceType), string(entry.OperationType), entry.MeshID, entry.Revision,
			entry.Context, entry.Operator, commontime.Time2String(entry.CreateTime))
	}
	h.logger.Info(str)
}
