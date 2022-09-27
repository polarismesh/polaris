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
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/natefinch/lumberjack"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

//go:generate gotests -w -all local_logger.go

const LocalLoggerName = "local"

type LocalLoggerConfig struct {
	OutputPath         string `json:"outputPath"`
	RotationMaxSize    int    `json:"rotationMaxSize"`
	RotationMaxAge     int    `json:"rotationMaxAge"`
	RotationMaxBackups int    `json:"rotationMaxBackups"`
}

// Validate 检查配置是否正确配置
func (c *LocalLoggerConfig) Validate() error {
	if c.OutputPath == "" {
		return errors.New("OutputPath is empty")
	}
	if c.RotationMaxSize <= 0 {
		return errors.New("RotationMaxSize is <= 0")
	}
	if c.RotationMaxAge <= 0 {
		return errors.New("RotationMaxAge is <= 0")
	}
	if c.RotationMaxBackups <= 0 {
		return errors.New("RotationMaxBackups is <= 0")
	}
	return nil
}

func defaultLocalLoggerConfig() *LocalLoggerConfig {
	return &LocalLoggerConfig{
		OutputPath:         "./discover-event",
		RotationMaxSize:    50,
		RotationMaxAge:     7,
		RotationMaxBackups: 100,
	}
}

type LocalLogger struct {
	logger *zap.Logger
}

func newLocalLogger(opt map[string]interface{}) (Logger, error) {
	data, err := json.Marshal(opt)
	if err != nil {
		return nil, err
	}
	conf := defaultLocalLoggerConfig()
	if err := json.Unmarshal(data, conf); err != nil {
		return nil, err
	}
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	localLogger := &LocalLogger{}
	localLogger.logger = newLogger(
		filepath.Join(conf.OutputPath, "discoverevent.log"),
		conf.RotationMaxSize,
		conf.RotationMaxBackups,
		conf.RotationMaxAge,
	)
	return localLogger, nil
}

func (l *LocalLogger) Log(events []model.DiscoverEvent) {
	for _, event := range events {
		l.logger.Info(fmt.Sprintf(
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

// newLogger 创建日志打印器
func newLogger(file string, maxSizeMB, maxBackups, maxAge int) *zap.Logger {
	encCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		TimeKey:        "time",
		NameKey:        "name",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   file,
		MaxSize:    maxSizeMB, // MB
		MaxBackups: maxBackups,
		MaxAge:     maxAge, // days
	})

	return zap.New(zapcore.NewCore(zapcore.NewConsoleEncoder(encCfg), w, zap.InfoLevel))
}
