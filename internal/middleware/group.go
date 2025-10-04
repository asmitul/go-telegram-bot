package middleware

import (
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

			// 1. 尝试加载群组
			g, err := m.groupRepo.FindByID(ctx.ChatID)
			if err != nil {
				// 群组不存在，创建新群组
				g = group.NewGroup(
					ctx.ChatID,
					ctx.ChatTitle,
					ctx.ChatType,
				)

				if err := m.groupRepo.Save(g); err != nil {
					// 创建失败，记录错误并注入临时群组对象避免 NPE
					m.logger.Error("failed_to_create_group",
						"error", err.Error(),
						"chat_id", ctx.ChatID,
						"chat_title", ctx.ChatTitle,
						"chat_type", ctx.ChatType,
					)
					// 注入临时群组对象（内存对象），避免后续 NPE
					// 群组将拥有默认配置
				}
			}

			// 2. 注入到上下文
			ctx.Group = g

			// 3. 执行下一个处理器
			return next(ctx)
		}
	}
}
