package middleware

import (
	"context"
	"fmt"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/handler"
)

// GroupMiddleware 群组中间件
// 负责加载群组信息并注入到上下文中
// 如果群组不存在，自动创建
type GroupMiddleware struct {
	groupRepo group.Repository
	logger    Logger // 用于记录错误
}

// NewGroupMiddleware 创建群组中间件
func NewGroupMiddleware(groupRepo group.Repository, logger Logger) *GroupMiddleware {
	return &GroupMiddleware{
		groupRepo: groupRepo,
		logger:    logger,
	}
}

// Middleware 返回中间件函数
func (m *GroupMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			// 只在群组/超级群组/频道中处理
			if !ctx.IsGroup() && !ctx.IsChannel() {
				// 私聊不需要加载群组信息
				return next(ctx)
			}

			// 创建 context（TODO: 从 handler.Context 传递）
			reqCtx := context.TODO()

			// 1. 尝试加载群组
			g, err := m.groupRepo.FindByID(reqCtx, ctx.ChatID)
			if err != nil {
				// 群组不存在，创建新群组
				g = group.NewGroup(
					ctx.ChatID,
					ctx.ChatTitle,
					ctx.ChatType,
				)

				if err := m.groupRepo.Save(reqCtx, g); err != nil {
					// 创建失败，记录错误并返回错误，不允许继续执行
					m.logger.Error("failed_to_create_group",
						"error", err.Error(),
						"chat_id", ctx.ChatID,
						"chat_title", ctx.ChatTitle,
						"chat_type", ctx.ChatType,
					)
					return fmt.Errorf("failed to create group: %w", err)
				}
			}

			// 2. 注入到上下文
			ctx.Group = g

			// 3. 执行下一个处理器
			return next(ctx)
		}
	}
}
