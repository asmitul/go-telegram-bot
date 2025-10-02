package middleware

import (
	"telegram-bot/internal/handler"
	"time"
)

// LoggingMiddleware 日志中间件
type LoggingMiddleware struct {
	logger Logger
}

// NewLoggingMiddleware 创建日志中间件
func NewLoggingMiddleware(logger Logger) *LoggingMiddleware {
	return &LoggingMiddleware{logger: logger}
}

// Middleware 返回中间件函数
func (m *LoggingMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			start := time.Now()

			m.logger.Info("message_received",
				"chat_type", ctx.ChatType,
				"chat_id", ctx.ChatID,
				"user_id", ctx.UserID,
				"username", ctx.Username,
				"text", ctx.Text,
			)

			err := next(ctx)

			duration := time.Since(start)

			if err != nil {
				m.logger.Error("handler_error",
					"error", err.Error(),
					"duration_ms", duration.Milliseconds(),
					"chat_id", ctx.ChatID,
					"user_id", ctx.UserID,
				)
			} else {
				m.logger.Info("handler_success",
					"duration_ms", duration.Milliseconds(),
					"chat_id", ctx.ChatID,
					"user_id", ctx.UserID,
				)
			}

			return err
		}
	}
}
