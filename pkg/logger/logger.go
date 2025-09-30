package logger

import (
	"io"
	"os"
)

// Logger 日志接口
type Logger interface {
	// 四个日志级别方法
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})

	// 添加字段的方法（返回新的 Logger 实例）
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger

	// 设置日志级别
	SetLevel(level Level)
}

// Config 日志配置
type Config struct {
	Level      Level  // 日志级别
	Format     string // 输出格式: "text" 或 "json"
	Output     io.Writer
	AddSource  bool // 是否添加调用位置信息
	TimeFormat string
}

// New 创建新的 Logger
func New(cfg Config) Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	if cfg.TimeFormat == "" {
		cfg.TimeFormat = "2006-01-02 15:04:05"
	}

	switch cfg.Format {
	case "json":
		return NewJSONLogger(cfg)
	default:
		return NewStandardLogger(cfg)
	}
}

// Default 创建默认的 Logger（Text 格式，Info 级别）
func Default() Logger {
	return New(Config{
		Level:      LevelInfo,
		Format:     "text",
		Output:     os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
	})
}

// NewWithLevel 创建指定级别的 Logger
func NewWithLevel(level Level) Logger {
	return New(Config{
		Level:  level,
		Format: "text",
	})
}