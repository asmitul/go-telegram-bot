package logger

import (
	"context"
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

	// 从 context 中提取字段
	WithContext(ctx context.Context) Logger

	// 设置日志级别
	SetLevel(level Level)
}

// Config 日志配置
type Config struct {
	Level         Level  // 日志级别
	Format        string // 输出格式: "text" 或 "json"
	Output        io.Writer
	AddSource     bool   // 是否添加调用位置信息
	TimeFormat    string
	EnableSanitize bool  // 是否启用敏感信息脱敏
	// 文件输出配置
	FileOutput     string // 日志文件路径（为空则不输出到文件）
	RotationConfig *RotationConfig // 日志轮转配置
}

// New 创建新的 Logger
func New(cfg Config) Logger {
	// 设置默认输出
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	// 如果配置了文件输出
	if cfg.FileOutput != "" {
		var fileWriter io.Writer

		// 如果配置了轮转
		if cfg.RotationConfig != nil {
			cfg.RotationConfig.Filename = cfg.FileOutput
			rotatingWriter, err := NewRotatingWriter(*cfg.RotationConfig)
			if err == nil {
				fileWriter = rotatingWriter
			}
		} else {
			// 简单文件输出
			file, err := os.OpenFile(cfg.FileOutput, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err == nil {
				fileWriter = file
			}
		}

		// 同时输出到控制台和文件
		if fileWriter != nil {
			cfg.Output = NewMultiWriter(cfg.Output, fileWriter)
		}
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