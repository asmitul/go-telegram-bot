package ratelimit

import (
	"telegram-bot/internal/domain/command"
)

// Middleware 限流中间件
type Middleware struct {
	manager *Manager
}

// NewMiddleware 创建限流中间件
func NewMiddleware(manager *Manager) *Middleware {
	return &Middleware{
		manager: manager,
	}
}

// Limit 限流中间件函数
func (m *Middleware) Limit(handler command.Handler) command.HandlerFunc {
	return func(ctx *command.Context) error {
		// 检查用户级别限流
		if !m.manager.AllowUser(ctx.UserID) {
			return command.ErrRateLimitExceeded
		}

		// 检查命令级别限流
		commandName := handler.Name()
		if !m.manager.AllowCommand(ctx.UserID, commandName) {
			return command.ErrRateLimitExceeded
		}

		// 继续执行
		return handler.Handle(ctx)
	}
}
