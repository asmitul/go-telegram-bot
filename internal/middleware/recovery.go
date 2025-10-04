package middleware

import (
	"fmt"
	"runtime/debug"
	"telegram-bot/internal/handler"
)

// RecoveryMiddleware 错误恢复中间件
// 捕获 panic 并转换为 error，防止程序崩溃
type RecoveryMiddleware struct {
	logger Logger
}

// NewRecoveryMiddleware 创建错误恢复中间件
func NewRecoveryMiddleware(logger Logger) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		logger: logger,
	}
}

// Middleware 返回中间件函数
func (m *RecoveryMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					// 记录 panic 信息和堆栈
					m.logger.Error("panic_recovered",
						"panic", r,
						"stack", string(debug.Stack()),
						"chat_id", ctx.ChatID,
						"user_id", ctx.UserID,
					)

					// 转换为 error，保留原始类型信息
					switch v := r.(type) {
					case error:
						// 如果 panic 的值本身就是 error，包装它
						err = fmt.Errorf("panic recovered: %w", v)
					default:
						// 否则创建新的 error
						err = fmt.Errorf("panic recovered: %v (type: %T)", r, r)
					}

					// 尝试通知用户
					ctx.Reply("❌ 服务器内部错误，请稍后再试")
				}
			}()

			return next(ctx)
		}
	}
}
