package middleware

import (
	"context"
	"fmt"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

// PermissionMiddleware 权限中间件
// 负责加载用户信息并注入到上下文中
type PermissionMiddleware struct {
	userRepo user.Repository
	ownerIDs []int64 // 配置的Owner用户ID列表
	logger   Logger  // 用于记录错误
}

// NewPermissionMiddleware 创建权限中间件
func NewPermissionMiddleware(userRepo user.Repository, ownerIDs []int64, logger Logger) *PermissionMiddleware {
	return &PermissionMiddleware{
		userRepo: userRepo,
		ownerIDs: ownerIDs,
		logger:   logger,
	}
}

// Middleware 返回中间件函数
func (m *PermissionMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			// 创建 context（TODO: 从 handler.Context 传递）
			reqCtx := context.TODO()

			// 1. 加载用户
			u, err := m.userRepo.FindByID(reqCtx, ctx.UserID)
			if err != nil {
				// 用户不存在，创建新用户（默认权限为普通用户）
				u = user.NewUser(
					ctx.UserID,
					ctx.Username,
					ctx.FirstName,
					ctx.LastName,
				)

				// 检查是否为配置的Owner
				if m.isConfiguredOwner(ctx.UserID) {
					// 设置为全局Owner权限（groupID = 0 表示全局）
					// 这样Owner在所有群组/私聊中都有Owner权限
					u.SetPermission(0, user.PermissionOwner)
				}

				if err := m.userRepo.Save(reqCtx, u); err != nil {
					// 创建失败，记录错误并返回错误，不允许继续执行
					m.logger.Error("failed_to_create_user",
						"error", err.Error(),
						"user_id", ctx.UserID,
						"username", ctx.Username,
					)
					return fmt.Errorf("failed to create user: %w", err)
				}
			} else {
				// 用户已存在，检查是否需要升级为Owner
				if m.isConfiguredOwner(ctx.UserID) {
					currentPerm := u.GetPermission(0)
					if currentPerm < user.PermissionOwner {
						// 使用细粒度更新避免并发冲突
						if err := m.userRepo.UpdatePermission(reqCtx, ctx.UserID, 0, user.PermissionOwner); err != nil {
							// 更新失败，记录错误但继续执行
							m.logger.Warn("failed_to_upgrade_owner_permission",
								"error", err.Error(),
								"user_id", ctx.UserID,
								"username", ctx.Username,
							)
						} else {
							// 更新本地对象（用于后续使用）
							u.SetPermission(0, user.PermissionOwner)
						}
					}
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

// isConfiguredOwner 检查用户ID是否在配置的Owner列表中
func (m *PermissionMiddleware) isConfiguredOwner(userID int64) bool {
	for _, id := range m.ownerIDs {
		if id == userID {
			return true
		}
	}
	return false
}
