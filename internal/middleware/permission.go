package middleware

import (
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

// PermissionMiddleware 权限中间件
// 负责加载用户信息并注入到上下文中
type PermissionMiddleware struct {
	userRepo user.Repository
}

// NewPermissionMiddleware 创建权限中间件
func NewPermissionMiddleware(userRepo user.Repository) *PermissionMiddleware {
	return &PermissionMiddleware{
		userRepo: userRepo,
	}
}

// Middleware 返回中间件函数
func (m *PermissionMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			// 1. 加载用户
			u, err := m.userRepo.FindByID(ctx.UserID)
			if err != nil {
				// 用户不存在，创建新用户（默认权限为普通用户）
				u = user.NewUser(
					ctx.UserID,
					ctx.Username,
					ctx.FirstName,
					ctx.LastName,
				)
				if err := m.userRepo.Save(u); err != nil {
					// 创建失败，继续执行但不注入用户
					return next(ctx)
				}
			}

			// 2. 注入到上下文
			ctx.User = u

			// 3. 执行下一个处理器
			// 具体的权限检查由处理器自己在 Handle 中执行
			return next(ctx)
		}
	}
}
