package logger

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

// JSONLogger JSON 格式的日志实现
type JSONLogger struct {
	mu        sync.Mutex
	level     Level
	output    io.Writer
	fields    map[string]interface{}
	addSource bool
}

// NewJSONLogger 创建 JSON Logger
func NewJSONLogger(cfg Config) *JSONLogger {
	return &JSONLogger{
		level:     cfg.Level,
		output:    cfg.Output,
		fields:    make(map[string]interface{}),
		addSource: cfg.AddSource,
	}
}

// logEntry JSON 日志条目
type logEntry struct {
	Time    string                 `json:"time"`
	Level   string                 `json:"level"`
	Message string                 `json:"msg"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

// log 内部日志方法
func (l *JSONLogger) log(level Level, msg string, fields ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 构建字段 map
	allFields := make(map[string]interface{})

	// 添加实例字段
	for k, v := range l.fields {
		allFields[k] = v
	}

	// 添加临时字段
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key, ok := fields[i].(string)
			if ok {
				allFields[key] = fields[i+1]
			}
		}
	}

	// 构建日志条目
	entry := logEntry{
		Time:    time.Now().Format(time.RFC3339),
		Level:   level.String(),
		Message: msg,
	}

	if len(allFields) > 0 {
		entry.Fields = allFields
	}

	// 序列化为 JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return // 忽略序列化错误
	}

	data = append(data, '\n')
	l.output.Write(data)
}

// Debug 输出 Debug 级别日志
func (l *JSONLogger) Debug(msg string, fields ...interface{}) {
	l.log(LevelDebug, msg, fields...)
}

// Info 输出 Info 级别日志
func (l *JSONLogger) Info(msg string, fields ...interface{}) {
	l.log(LevelInfo, msg, fields...)
}

// Warn 输出 Warn 级别日志
func (l *JSONLogger) Warn(msg string, fields ...interface{}) {
	l.log(LevelWarn, msg, fields...)
}

// Error 输出 Error 级别日志
func (l *JSONLogger) Error(msg string, fields ...interface{}) {
	l.log(LevelError, msg, fields...)
}

// WithField 添加单个字段
func (l *JSONLogger) WithField(key string, value interface{}) Logger {
	newFields := make(map[string]interface{}, len(l.fields)+1)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &JSONLogger{
		level:     l.level,
		output:    l.output,
		fields:    newFields,
		addSource: l.addSource,
	}
}

// WithFields 添加多个字段
func (l *JSONLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{}, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &JSONLogger{
		level:     l.level,
		output:    l.output,
		fields:    newFields,
		addSource: l.addSource,
	}
}

// SetLevel 设置日志级别
func (l *JSONLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}