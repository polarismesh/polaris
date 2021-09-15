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
	"fmt"
	"strings"
)

const (
	DefaultScopeName          = "default"
	OverrideScopeName         = "all"
	defaultOutputLevel        = InfoLevel
	defaultStackTraceLevel    = NoneLevel
	defaultOutputPath         = "stdout"
	defaultErrorOutputPath    = "stderr"
	defaultRotationMaxAge     = 30
	defaultRotationMaxSize    = 100 * 1024 * 1024
	defaultRotationMaxBackups = 1000
)

// Level is an enumeration of all supported log levels.
type Level int

const (
	// NoneLevel disables logging
	NoneLevel Level = iota
	// FatalLevel enables fatal level logging
	FatalLevel
	// ErrorLevel enables error level logging
	ErrorLevel
	// WarnLevel enables warn level logging
	WarnLevel
	// InfoLevel enables info level logging
	InfoLevel
	// DebugLevel enables debug level logging
	DebugLevel
)

var levelToString = map[Level]string{
	DebugLevel: "debug",
	InfoLevel:  "info",
	WarnLevel:  "warn",
	ErrorLevel: "error",
	FatalLevel: "fatal",
	NoneLevel:  "none",
}

var stringToLevel = map[string]Level{
	"debug": DebugLevel,
	"info":  InfoLevel,
	"warn":  WarnLevel,
	"error": ErrorLevel,
	"fatal": FatalLevel,
	"none":  NoneLevel,
}

// Options defines the set of options supported by Istio's component logging package.
type Options struct {
	// OutputPaths is a list of file system paths to write the log data to.
	// The special values stdout and stderr can be used to output to the
	// standard I/O streams. This defaults to stdout.
	OutputPaths []string `yaml:"OutputPaths"`

	// ErrorOutputPaths is a list of file system paths to write logger errors to.
	// The special values stdout and stderr can be used to output to the
	// standard I/O streams. This defaults to stderr.
	ErrorOutputPaths []string `yaml:"ErrorOutputPaths"`

	// RotateOutputPath is the path to a rotating log file. This file should
	// be automatically rotated over time, based on the rotation parameters such
	// as RotationMaxSize and RotationMaxAge. The default is to not rotate.
	//
	// This path is used as a foundational path. This is where log output is normally
	// saved. When a rotation needs to take place because the file got too big or too
	// old, then the file is renamed by appending a timestamp to the name. Such renamed
	// files are called backups. Once a backup has been created,
	// output resumes to this path.
	RotateOutputPath string `yaml:"RotateOutputPath"`

	// RotationMaxSize is the maximum size in megabytes of a log file before it gets
	// rotated. It defaults to 100 megabytes.
	RotationMaxSize int `yaml:"RotationMaxSize"`

	// RotationMaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename. Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is to remove log files
	// older than 30 days.
	RotationMaxAge int `yaml:"RotationMaxAge"`

	// RotationMaxBackups is the maximum number of old log files to retain.  The default
	// is to retain at most 1000 logs.
	RotationMaxBackups int `yaml:"RotationMaxBackups"`

	// JSONEncoding controls whether the log is formatted as JSON.
	JSONEncoding bool

	// LogGrpc indicates that Grpc logs should be captured. The default is true.
	// This is not exposed through the command-line flags, as this flag is mainly useful for testing: Grpc
	// stack will hold on to the logger even though it gets closed. This causes data races.
	LogGrpc bool

	Level string

	outputLevels     string
	logCallers       string
	stackTraceLevels string
}

// DefaultOptions returns a new set of options, initialized to the defaults
func DefaultOptions() *Options {
	return &Options{
		OutputPaths:        []string{defaultOutputPath},
		ErrorOutputPaths:   []string{defaultErrorOutputPath},
		RotationMaxSize:    defaultRotationMaxSize,
		RotationMaxAge:     defaultRotationMaxAge,
		RotationMaxBackups: defaultRotationMaxBackups,
		outputLevels:       DefaultScopeName + ":" + levelToString[defaultOutputLevel],
		stackTraceLevels:   DefaultScopeName + ":" + levelToString[defaultStackTraceLevel],
		LogGrpc:            false,
	}
}

// SetOutputLevel sets the minimum log output level for a given scope.
func (o *Options) SetOutputLevel(scope, level string) error {
	_, exist := stringToLevel[level]
	if !exist {
		return errors.New("invalid log level")
	}
	sl := scope + ":" + level

	levels := strings.Split(o.outputLevels, ",")

	if scope == DefaultScopeName {
		// see if we have an entry without a scope prefix (which represents the default scope)
		for i, ol := range levels {
			if !strings.Contains(ol, ":") {
				levels[i] = sl
				o.outputLevels = strings.Join(levels, ",")
				return nil
			}
		}
	}

	prefix := scope + ":"
	for i, ol := range levels {
		if strings.HasPrefix(ol, prefix) {
			levels[i] = sl
			o.outputLevels = strings.Join(levels, ",")
			return nil
		}
	}

	levels = append(levels, sl)
	o.outputLevels = strings.Join(levels, ",")

	return nil
}

// GetOutputLevel returns the minimum log output level for a given scope.
func (o *Options) GetOutputLevel(scope string) (Level, error) {
	return o.GetStackTraceLevel(scope)
}

// SetStackTraceLevel sets the minimum stack tracing level for a given scope.
func (o *Options) SetStackTraceLevel(scope, level string) {
	sl := scope + ":" + level

	levels := strings.Split(o.stackTraceLevels, ",")

	if scope == DefaultScopeName {
		// see if we have an entry without a scope prefix (which represents the default scope)
		for i, ol := range levels {
			if !strings.Contains(ol, ":") {
				levels[i] = sl
				o.stackTraceLevels = strings.Join(levels, ",")
				return
			}
		}
	}

	prefix := scope + ":"
	for i, ol := range levels {
		if strings.HasPrefix(ol, prefix) {
			levels[i] = sl
			o.stackTraceLevels = strings.Join(levels, ",")
			return
		}
	}

	levels = append(levels, sl)
	o.stackTraceLevels = strings.Join(levels, ",")
}

// GetStackTraceLevel returns the minimum stack tracing level for a given scope.
func (o *Options) GetStackTraceLevel(scope string) (Level, error) {
	levels := strings.Split(o.stackTraceLevels, ",")

	if scope == DefaultScopeName {
		// see if we have an entry without a scope prefix (which represents the default scope)
		for _, ol := range levels {
			if !strings.Contains(ol, ":") {
				_, l, err := convertScopedLevel(ol)
				return l, err
			}
		}
	}

	prefix := scope + ":"
	for _, ol := range levels {
		if strings.HasPrefix(ol, prefix) {
			_, l, err := convertScopedLevel(ol)
			return l, err
		}
	}

	return NoneLevel, fmt.Errorf("no level defined for scope '%s'", scope)
}

// SetLogCallers sets whether to output the caller's source code location for a given scope.
func (o *Options) SetLogCallers(scope string, include bool) {
	scopes := strings.Split(o.logCallers, ",")

	// remove any occurrence of the scope
	for i, s := range scopes {
		if s == scope {
			scopes[i] = ""
		}
	}

	if include {
		// find a free slot if there is one
		for i, s := range scopes {
			if s == "" {
				scopes[i] = scope
				o.logCallers = strings.Join(scopes, ",")
				return
			}
		}

		scopes = append(scopes, scope)
	}

	o.logCallers = strings.Join(scopes, ",")
}

// GetLogCallers returns whether the caller's source code location is output for a given scope.
func (o *Options) GetLogCallers(scope string) bool {
	scopes := strings.Split(o.logCallers, ",")

	for _, s := range scopes {
		if s == scope {
			return true
		}
	}

	return false
}

func convertScopedLevel(sl string) (string, Level, error) {
	var s string
	var l string

	pieces := strings.Split(sl, ":")
	if len(pieces) == 1 {
		s = DefaultScopeName
		l = pieces[0]
	} else if len(pieces) == 2 {
		s = pieces[0]
		l = pieces[1]
	} else {
		return "", NoneLevel, fmt.Errorf("invalid output level format '%s'", sl)
	}

	level, ok := stringToLevel[l]
	if !ok {
		return "", NoneLevel, fmt.Errorf("invalid output level '%s'", sl)
	}

	return s, level, nil
}
