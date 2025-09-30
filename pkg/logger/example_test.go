package logger_test

import (
	"telegram-bot/pkg/logger"
)

func ExampleNew_text() {
	// 创建文本格式的 Logger
	log := logger.New(logger.Config{
		Level:  logger.LevelInfo,
		Format: "text",
	})

	// 基本日志
	log.Info("application started")

	// 带字段的日志
	log.Info("user login", "user_id", 12345, "username", "john")

	// 不同级别
	log.Debug("debug info") // 不会输出，因为级别是 Info
	log.Warn("warning message")
	log.Error("error occurred", "error", "connection timeout")
}

func ExampleNew_json() {
	// 创建 JSON 格式的 Logger
	log := logger.New(logger.Config{
		Level:  logger.LevelInfo,
		Format: "json",
	})

	log.Info("user action", "user_id", 12345, "action", "login")
	// 输出: {"time":"2006-01-02T15:04:05Z","level":"INFO","msg":"user action","fields":{"user_id":12345,"action":"login"}}
}

func ExampleLogger_WithField() {
	log := logger.Default()

	// 创建带有持久字段的 logger
	userLogger := log.WithField("user_id", 12345)

	// 所有日志都会包含 user_id 字段
	userLogger.Info("login")
	userLogger.Info("logout")
}

func ExampleLogger_WithFields() {
	log := logger.Default()

	// 添加多个持久字段
	requestLogger := log.WithFields(map[string]interface{}{
		"request_id": "req-123",
		"user_id":    12345,
		"ip":         "192.168.1.1",
	})

	requestLogger.Info("processing request")
	requestLogger.Error("request failed")
}

func ExampleParseLevel() {
	// 从字符串解析日志级别
	level := logger.ParseLevel("debug")
	log := logger.NewWithLevel(level)

	log.Debug("this will be shown")
}