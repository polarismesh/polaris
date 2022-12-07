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

package log

import (
	"errors"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 创建日志打印器
func Logger(file string) *zap.Logger {
	encCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "name",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   file,
		MaxSize:    100, // MB
		MaxBackups: 10,
		MaxAge:     7, // days
		Compress:   true,
	})

	return zap.New(zapcore.NewCore(zapcore.NewConsoleEncoder(encCfg), w, zap.InfoLevel))
}

func SetLogOutputLevel(scopeName string, levelName string) error {
	scope := FindScope(scopeName)
	if scope == nil {
		return errors.New("invalid scope name")
	}

	l, exist := stringToLevel[levelName]
	if !exist {
		return errors.New("invalid log level")
	}

	lock.Lock()
	scope.SetOutputLevel(l)
	lock.Unlock()

	return nil
}
