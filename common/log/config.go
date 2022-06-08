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

// Package log
// Once configured, this package intercepts the output of the standard golang "log" package as well as anything
// sent to the global zap logger (zap.L()).
package log

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc/grpclog"
)

// none is used to disable logging output as well as to disable stack tracing.
const none zapcore.Level = 100

var levelToZap = map[Level]zapcore.Level{
	DebugLevel: zapcore.DebugLevel,
	InfoLevel:  zapcore.InfoLevel,
	WarnLevel:  zapcore.WarnLevel,
	ErrorLevel: zapcore.ErrorLevel,
	FatalLevel: zapcore.FatalLevel,
	NoneLevel:  none,
}

// functions that can be replaced in a test setting
type patchTable struct {
	write       func(ent zapcore.Entry, fields []zapcore.Field) error
	sync        func() error
	exitProcess func(code int)
	errorSink   zapcore.WriteSyncer
}

func init() {
	// use our defaults for starters so that logging works even before everything is fully configured
	_ = Configure(DefaultOptions())
}

// prepZap is a utility function used by the Configure function.
func prepZap(options *Options) ([]zapcore.Core, zapcore.Core, zapcore.WriteSyncer, error) {
	encCfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "scope",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stack",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeTime:     formatDate,
	}

	var enc zapcore.Encoder
	if options.JSONEncoding {
		enc = zapcore.NewJSONEncoder(encCfg)
	} else {
		enc = zapcore.NewConsoleEncoder(encCfg)
	}

	var rotateSink zapcore.WriteSyncer
	if len(options.RotateOutputPath) > 0 {
		rotateSink = zapcore.AddSync(&lumberjack.Logger{
			Filename:   options.RotateOutputPath,
			MaxSize:    options.RotationMaxSize,
			MaxBackups: options.RotationMaxBackups,
			MaxAge:     options.RotationMaxAge,
		})
	}

	err := createPathIfNotExist(options.ErrorOutputPaths...)
	if err != nil {
		return nil, nil, nil, err
	}
	errSink, closeErrorSink, err := zap.Open(options.ErrorOutputPaths...)
	if err != nil {
		return nil, nil, nil, err
	}

	var outputSink zapcore.WriteSyncer
	if len(options.OutputPaths) > 0 {
		err := createPathIfNotExist(options.OutputPaths...)
		if err != nil {
			return nil, nil, nil, err
		}
		outputSink, _, err = zap.Open(options.OutputPaths...)
		if err != nil {
			closeErrorSink()
			return nil, nil, nil, err
		}
	}

	var sink zapcore.WriteSyncer
	if rotateSink != nil && outputSink != nil {
		sink = zapcore.NewMultiWriteSyncer(outputSink, rotateSink)
	} else if rotateSink != nil {
		sink = rotateSink
	} else if outputSink != nil {
		sink = outputSink
	} else {
		sink = zapcore.AddSync(os.Stdout)
	}

	var enabler zap.LevelEnablerFunc = func(lvl zapcore.Level) bool {
		switch lvl {
		case zapcore.FatalLevel:
			return defaultScope.FatalEnabled()
		case zapcore.ErrorLevel:
			return defaultScope.ErrorEnabled()
		case zapcore.WarnLevel:
			return defaultScope.WarnEnabled()
		case zapcore.InfoLevel:
			return defaultScope.InfoEnabled()
		}
		return defaultScope.DebugEnabled()
	}

	var errCore zapcore.Core
	if len(options.ErrorRotateOutputPath) > 0 {
		errRotateSink := zapcore.AddSync(&lumberjack.Logger{
			Filename:   options.ErrorRotateOutputPath,
			MaxSize:    options.RotationMaxSize,
			MaxBackups: options.RotationMaxBackups,
			MaxAge:     options.RotationMaxAge,
		})
		errCore = zapcore.NewCore(enc, errRotateSink, zap.NewAtomicLevelAt(zapcore.ErrorLevel))
	}

	cores := make([]zapcore.Core, 0)
	cores = append(cores, zapcore.NewCore(enc, sink, zap.NewAtomicLevelAt(zapcore.DebugLevel)))
	if errCore != nil {
		cores = append(cores, errCore)
	}
	return cores, zapcore.NewCore(enc, sink, enabler), errSink, nil
}

func formatDate(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	// t = t.UTC() 不用utc时间
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	micros := t.Nanosecond() / 1000

	buf := make([]byte, 27)

	buf[0] = byte((year/1000)%10) + '0'
	buf[1] = byte((year/100)%10) + '0'
	buf[2] = byte((year/10)%10) + '0'
	buf[3] = byte(year%10) + '0'
	buf[4] = '-'
	buf[5] = byte((month)/10) + '0'
	buf[6] = byte((month)%10) + '0'
	buf[7] = '-'
	buf[8] = byte((day)/10) + '0'
	buf[9] = byte((day)%10) + '0'
	buf[10] = 'T'
	buf[11] = byte((hour)/10) + '0'
	buf[12] = byte((hour)%10) + '0'
	buf[13] = ':'
	buf[14] = byte((minute)/10) + '0'
	buf[15] = byte((minute)%10) + '0'
	buf[16] = ':'
	buf[17] = byte((second)/10) + '0'
	buf[18] = byte((second)%10) + '0'
	buf[19] = '.'
	buf[20] = byte((micros/100000)%10) + '0'
	buf[21] = byte((micros/10000)%10) + '0'
	buf[22] = byte((micros/1000)%10) + '0'
	buf[23] = byte((micros/100)%10) + '0'
	buf[24] = byte((micros/10)%10) + '0'
	buf[25] = byte((micros)%10) + '0'
	buf[26] = 'Z'

	enc.AppendString(string(buf))
}

func updateScopes(typeName string, options *Options, cores []zapcore.Core, errSink zapcore.WriteSyncer) error {
	scope := FindScope(typeName)
	if scope == nil {
		return fmt.Errorf("unknown logger name '%s' specified", typeName)
	}

	// update the output levels of all listed scopes
	outPutLevel, ok := stringToLevel[options.OutputLevel]
	if !ok {
		return fmt.Errorf("unknown outPutLevel '%s' specified", options.OutputLevel)
	}
	scope.SetOutputLevel(outPutLevel)

	// update the stack tracing levels of all listed scopes
	stackTraceLevel, ok := stringToLevel[options.StackTraceLevel]
	if !ok {
		return fmt.Errorf("unknown stackTraceLevel '%s' specified", options.StackTraceLevel)
	}
	scope.SetStackTraceLevel(stackTraceLevel)

	// update patchTable
	pt := patchTable{
		write: func(ent zapcore.Entry, fields []zapcore.Field) error {
			var errs error
			for _, core := range cores {
				if core.Enabled(ent.Level) {
					if err := core.Write(ent, fields); err != nil {
						errs = multierror.Append(errs, err)
					}
				}
			}
			if ent.Level == zapcore.FatalLevel {
				scope.getPathTable().exitProcess(1)
			}

			return errs
		},
		sync: func() error {
			var errs error
			for _, core := range cores {
				if err := core.Sync(); err != nil {
					errs = multierror.Append(errs, err)
				}
			}
			return errs
		},
		exitProcess: os.Exit,
		errorSink:   errSink,
	}
	scope.pt.Store(&pt)

	// update the caller location setting of all listed scopes
	scope.SetLogCallers(options.LogCaller)

	return nil
}

// Configure .
// nolint: staticcheck
// You typically call this once at process startup.
// Configure Once this call returns, the logging system is ready to accept data.
func Configure(optionsMap map[string]*Options) error {
	for typeName, options := range optionsMap {
		setDefaultOption(options)
		cores, captureCore, errSink, err := prepZap(options)
		if err != nil {
			return err
		}

		if err = updateScopes(typeName, options, cores, errSink); err != nil {
			return err
		}

		if typeName == DefaultLoggerName {
			opts := []zap.Option{
				zap.ErrorOutput(errSink),
				zap.AddCallerSkip(1),
			}

			if defaultScope.GetLogCallers() {
				opts = append(opts, zap.AddCaller())
			}

			l := defaultScope.GetStackTraceLevel()
			if l != NoneLevel {
				opts = append(opts, zap.AddStacktrace(levelToZap[l]))
			}

			captureLogger := zap.New(captureCore, opts...)

			// capture global zap logging and force it through our logger
			_ = zap.ReplaceGlobals(captureLogger)

			// capture standard golang "log" package output and force it through our logger
			_ = zap.RedirectStdLog(captureLogger)

			// capture gRPC logging
			if options.LogGrpc {
				grpclog.SetLogger(zapgrpc.NewLogger(captureLogger.WithOptions(zap.AddCallerSkip(2))))
			}
		}
	}
	return nil
}

// setDefaultOption 设置日志配置的默认值
func setDefaultOption(options *Options) {
	if options.RotationMaxSize == 0 {
		options.RotationMaxSize = defaultRotationMaxSize
	}
	if options.RotationMaxAge == 0 {
		options.RotationMaxAge = defaultRotationMaxAge
	}
	if options.RotationMaxBackups == 0 {
		options.RotationMaxBackups = defaultRotationMaxBackups
	}
	if options.OutputLevel == "" {
		options.OutputLevel = levelToString[defaultOutputLevel]
	}
	if options.StackTraceLevel == "" {
		options.StackTraceLevel = levelToString[defaultStackTraceLevel]
	}
	// 默认打开
	options.LogCaller = true
}

// Sync flushes any buffered log entries.
// Processes should normally take care to call Sync before exiting.
func Sync() error {
	return defaultScope.getPathTable().sync()
}

// createPathIfNotExist 如果判断为本地文件，检查目录是否存在，不存在创建父级目录
func createPathIfNotExist(paths ...string) error {
	for _, path := range paths {
		u, err := url.Parse(path)
		if err != nil {
			return fmt.Errorf("can't parse %q as a URL: %v", path, err)
		}
		if (u.Scheme == "" || u.Scheme == "file") && u.Path != "stdout" && u.Path != "stderr" {
			dir := filepath.Dir(u.Path)
			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return fmt.Errorf("can't create %q directory: %v", dir, err)
			}
		}
	}
	return nil
}
