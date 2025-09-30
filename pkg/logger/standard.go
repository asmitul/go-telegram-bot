package logger

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// StandardLogger 标准文本格式的日志实现
type StandardLogger struct {
	mu         sync.Mutex
	level      Level
	output     io.Writer
	fields     map[string]interface{}
	timeFormat string
	addSource  bool
}

// NewStandardLogger 创建标准 Logger
func NewStandardLogger(cfg Config) *StandardLogger {
	return &StandardLogger{
		level:      cfg.Level,
		output:     cfg.Output,
		fields:     make(map[string]interface{}),
		timeFormat: cfg.TimeFormat,
		addSource:  cfg.AddSource,
	}
}

// log 内部日志方法
func (l *StandardLogger) log(level Level, msg string, fields ...interface{}) {
	if level < l.level {
		return // 级别不够，不输出
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 构建日志消息
	timestamp := time.Now().Format(l.timeFormat)
	levelStr := level.String()

	// 格式: [2006-01-02 15:04:05] [INFO] message key=value key2=value2
	output := fmt.Sprintf("[%s] [%s] %s", timestamp, levelStr, msg)

	// 添加实例字段
	if len(l.fields) > 0 {
		for k, v := range l.fields {
			output += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	// 添加临时字段
	if len(fields) > 0 {
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				key := fields[i]
				val := fields[i+1]
				output += fmt.Sprintf(" %s=%v", key, val)
			}
		}
	}

	output += "\n"
	l.output.Write([]byte(output))
}

// Debug 输出 Debug 级别日志
func (l *StandardLogger) Debug(msg string, fields ...interface{}) {
	l.log(LevelDebug, msg, fields...)
}

// Info 输出 Info 级别日志
func (l *StandardLogger) Info(msg string, fields ...interface{}) {
	l.log(LevelInfo, msg, fields...)
}

// Warn 输出 Warn 级别日志
func (l *StandardLogger) Warn(msg string, fields ...interface{}) {
	l.log(LevelWarn, msg, fields...)
}

// Error 输出 Error 级别日志
func (l *StandardLogger) Error(msg string, fields ...interface{}) {
	l.log(LevelError, msg, fields...)
}

// WithField 添加单个字段
func (l *StandardLogger) WithField(key string, value interface{}) Logger {
	newFields := make(map[string]interface{}, len(l.fields)+1)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &StandardLogger{
		level:      l.level,
		output:     l.output,
		fields:     newFields,
		timeFormat: l.timeFormat,
		addSource:  l.addSource,
	}
}

// WithFields 添加多个字段
func (l *StandardLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{}, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &StandardLogger{
		level:      l.level,
		output:     l.output,
		fields:     newFields,
		timeFormat: l.timeFormat,
		addSource:  l.addSource,
	}
}

// SetLevel 设置日志级别
func (l *StandardLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}