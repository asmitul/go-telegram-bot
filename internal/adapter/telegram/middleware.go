package telegram

import (
	"context"
	"fmt"
	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Middleware 中间件接口
type Middleware func(next HandlerFunc) HandlerFunc

// HandlerFunc 处理函数
type HandlerFunc func(ctx *command.Context) error

// PermissionMiddleware 权限检查中间件
type PermissionMiddleware struct {
	userRepo  user.Repository
	groupRepo group.Repository
}

// NewPermissionMiddleware 创建权限中间件
func NewPermissionMiddleware(userRepo user.Repository, groupRepo group.Repository) *PermissionMiddleware {
	return &PermissionMiddleware{
		userRepo:  userRepo,
		groupRepo: groupRepo,
	}
}

// Check 检查权限
func (m *PermissionMiddleware) Check(handler command.Handler) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx *command.Context) error {
			// 1. 检查命令是否在群组中启用
			if !handler.IsEnabled(ctx.GroupID) {
				return fmt.Errorf("此命令在当前群组中未启用")
			}

			// 2. 获取用户信息
			u, err := m.userRepo.FindByID(ctx.UserID)
			if err != nil {
				// 用户不存在，创建新用户（权限为普通用户）
				u = user.NewUser(ctx.UserID, "", "", "")
				if err := m.userRepo.Save(u); err != nil {
					return fmt.Errorf("创建用户失败: %w", err)
				}
			}

			// 3. 检查权限
			requiredPerm := handler.RequiredPermission()
			if !u.HasPermission(ctx.GroupID, requiredPerm) {
				return fmt.Errorf("❌ 权限不足！需要权限: %s，当前权限: %s",
					requiredPerm.String(),
					u.GetPermission(ctx.GroupID).String())
			}

			// 4. 将用户信息注入上下文
			ctx.User = u

			// 5. 执行处理器
			return next(ctx)
		}
	}
}

// LoggingMiddleware 日志中间件
type LoggingMiddleware struct {
	logger Logger
}

type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

func NewLoggingMiddleware(logger Logger) *LoggingMiddleware {
	return &LoggingMiddleware{logger: logger}
}

func (m *LoggingMiddleware) Log() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx *command.Context) error {
			m.logger.Info("command_received",
				"command", ctx.Text,
				"user_id", ctx.UserID,
				"group_id", ctx.GroupID,
			)

			err := next(ctx)

			if err != nil {
				m.logger.Error("command_failed",
					"command", ctx.Text,
					"user_id", ctx.UserID,
					"group_id", ctx.GroupID,
					"error", err.Error(),
				)
			} else {
				m.logger.Info("command_success",
					"command", ctx.Text,
					"user_id", ctx.UserID,
					"group_id", ctx.GroupID,
				)
			}

			return err
		}
	}
}

// RateLimitMiddleware 限流中间件
type RateLimitMiddleware struct {
	limiter RateLimiter
}

type RateLimiter interface {
	Allow(userID int64) bool
}

func NewRateLimitMiddleware(limiter RateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{limiter: limiter}
}

func (m *RateLimitMiddleware) Limit() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx *command.Context) error {
			if !m.limiter.Allow(ctx.UserID) {
				return fmt.Errorf("⏱️ 操作过于频繁，请稍后再试")
			}
			return next(ctx)
		}
	}
}

// Chain 链式调用中间件
func Chain(middlewares ...Middleware) Middleware {
	return func(final HandlerFunc) HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// ConvertTelegramUpdate 将 Telegram Update 转换为 Command Context
func ConvertTelegramUpdate(update tgbotapi.Update) *command.Context {
	if update.Message == nil {
		return nil
	}

	msg := update.Message
	args := parseArgs(msg.CommandArguments())

	return &command.Context{
		Ctx:       context.Background(),
		UserID:    msg.From.ID,
		GroupID:   msg.Chat.ID,
		MessageID: msg.MessageID,
		Text:      msg.Text,
		Args:      args,
	}
}

// parseArgs 解析命令参数
func parseArgs(argsStr string) []string {
	if argsStr == "" {
		return []string{}
	}

	// 简单的空格分割
	// TODO: 支持引号包裹的参数
	args := []string{}
	current := ""

	for _, char := range argsStr {
		if char == ' ' {
			if current != "" {
				args = append(args, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		args = append(args, current)
	}

	return args
}
