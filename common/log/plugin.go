package log

import (
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/go-hclog"
	pluginLog "github.com/polaris-contrib/polaris-server-remote-plugin-common/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// defaultPluginLogName default log name
	defaultPluginLogName = "polaris-server-plugin"
)

func init() {
	pluginLog.DefaultLogger = newPluginLogger(defaultPluginLogName)
}

// pluginLogger polaris plugin log implements
type pluginLogger struct {
	name string
}

// NewPluginLogger return s new pluginLogger.
func newPluginLogger(name string) *pluginLogger {
	return &pluginLogger{name: name}
}

// Log Emit a message and key/value pairs at a provided log level
func (h *pluginLogger) Log(level hclog.Level, msg string, args ...interface{}) {
	switch level {
	case hclog.NoLevel:
		return
	case hclog.Trace:
		h.Trace(msg, args)
	case hclog.Debug:
		h.Debug(msg, args)
	case hclog.Info:
		h.Info(msg, args)
	case hclog.Warn:
		h.Warn(msg, args)
	case hclog.Error:
		h.Error(msg, args)
	case hclog.Off:
		return
	}
}

// Trace emit a message and key/value pairs at the TRACE level
func (h *pluginLogger) Trace(msg string, args ...interface{}) {
	defaultScope.Debug(msg, anythingsToZapFields(args...)...)
}

// Debug emit a message and key/value pairs at the DEBUG level
func (h *pluginLogger) Debug(msg string, args ...interface{}) {
	defaultScope.Debug(msg, anythingsToZapFields(args...)...)
}

// Info emit a message and key/value pairs at the INFO level
func (h *pluginLogger) Info(msg string, args ...interface{}) {
	defaultScope.Info(msg, anythingsToZapFields(args...)...)
}

// Warn emit a message and key/value pairs at the WARN level
func (h *pluginLogger) Warn(msg string, args ...interface{}) {
	defaultScope.Warn(msg, anythingsToZapFields(args...)...)
}

// Error emit a message and key/value pairs at the ERROR level
func (h *pluginLogger) Error(msg string, args ...interface{}) {
	defaultScope.Error(msg, anythingsToZapFields(args...)...)
}

// Fatal emit a message and key/value pairs at the Fatal level
func (h *pluginLogger) Fatal(msg string, args ...interface{}) {
	defaultScope.Fatal(msg, anythingsToZapFields(args...)...)
}

// IsTrace indicate if TRACE logs would be emitted.
func (h *pluginLogger) IsTrace() bool {
	return defaultScope.DebugEnabled()
}

// IsDebug indicate if DEBUG logs would be emitted.
func (h *pluginLogger) IsDebug() bool {
	return defaultScope.DebugEnabled()
}

// IsInfo indicate if INFO logs would be emitted
func (h *pluginLogger) IsInfo() bool {
	return defaultScope.InfoEnabled()
}

// IsWarn indicate if WARN logs would be emitted.
func (h *pluginLogger) IsWarn() bool {
	return defaultScope.WarnEnabled()
}

// IsError indicate if ERROR logs would be emitted.
func (h *pluginLogger) IsError() bool {
	return defaultScope.ErrorEnabled()
}

// ImpliedArgs returns With key/value pairs
func (h *pluginLogger) ImpliedArgs() []interface{} {
	return []interface{}{}
}

// With creates a su blogger that will always have the given key/value pairs
//
// not support.
func (h *pluginLogger) With(_ ...interface{}) hclog.Logger {
	return h
}

// Name returns the Name of the logger
func (h *pluginLogger) Name() string {
	return h.name
}

// Named create a logger that will prepend the name string on the front of all messages.
func (h *pluginLogger) Named(name string) hclog.Logger {
	newLogger := &pluginLogger{name: name}
	return newLogger
}

// ResetNamed reset log name
func (h *pluginLogger) ResetNamed(name string) hclog.Logger {
	h.name = name
	return h
}

// SetLevel updates the level.
func (h *pluginLogger) SetLevel(level hclog.Level) {
	defaultScope.outputLevel = convertHCLogLevel(level)
}

// StandardLogger return a value that conforms to the stdlib log.Logger interface
func (h *pluginLogger) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return log.New(h.StandardWriter(opts), "", log.LstdFlags)
}

// StandardWriter Return a value that conforms to io.Writer, which can be passed into log.SetOutput()
func (h *pluginLogger) StandardWriter(_ *hclog.StandardLoggerOptions) io.Writer {
	return hclog.DefaultOutput
}

// convertHCLogLevel convert log level.
func convertHCLogLevel(hll hclog.Level) Level {
	logLevelMap := map[hclog.Level]Level{
		hclog.NoLevel: NoneLevel,
		hclog.Debug:   DebugLevel,
		hclog.Trace:   DebugLevel,
		hclog.Info:    InfoLevel,
		hclog.Warn:    WarnLevel,
		hclog.Error:   ErrorLevel,
	}
	if level, ok := logLevelMap[hll]; ok {
		return level
	}
	return InfoLevel
}

func anythingsToZapFields(args ...interface{}) []zap.Field {
	var fields []zapcore.Field
	for i := len(args); i > 0; i -= 2 {
		left := i - 2
		if left < 0 {
			left = 0
		}

		items := args[left:i]

		switch l := len(items); l {
		case 2:
			k, ok := items[0].(string)
			// only support string value as zap field's key
			if ok {
				fields = append(fields, zap.Any(k, items[1]))
			} else {
				fields = append(fields, zap.Any(fmt.Sprintf("field-%d", i-1), items[1]))
				fields = append(fields, zap.Any(fmt.Sprintf("field-%d", left), items[0]))
			}
		case 1:
			fields = append(fields, zap.Any(fmt.Sprintf("arg%d", left), items[0]))
		}
	}

	return fields
}
