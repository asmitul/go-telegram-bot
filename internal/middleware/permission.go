package middleware

import (
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

// PermissionMiddleware 权限中间件
// 负责加载用户信息并注入到上下文中
type PermissionMiddleware struct {
	userRepo user.Repository
	ownerIDs []int64 // 配置的Owner用户ID列表
}

// NewPermissionMiddleware 创建权限中间件
func NewPermissionMiddleware(userRepo user.Repository, ownerIDs []int64) *PermissionMiddleware {
	return &PermissionMiddleware{
		userRepo: userRepo,
		ownerIDs: ownerIDs,
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

				// 检查是否为配置的Owner
				if m.isConfiguredOwner(ctx.UserID) {
					// 设置为Owner权限（在所有群组/私聊中都有Owner权限）
					// 使用ChatID作为权限的groupID
					groupID := ctx.ChatID
					if ctx.IsPrivate() {
						groupID = ctx.UserID
					}
					u.SetPermission(groupID, user.PermissionOwner)
				}

				if err := m.userRepo.Save(u); err != nil {
					// 创建失败，继续执行但不注入用户
					return next(ctx)
				}
			} else {
				// 用户已存在，检查是否需要升级为Owner
				if m.isConfiguredOwner(ctx.UserID) {
					groupID := ctx.ChatID
					if ctx.IsPrivate() {
						groupID = ctx.UserID
					}

					currentPerm := u.GetPermission(groupID)
					if currentPerm < user.PermissionOwner {
						u.SetPermission(groupID, user.PermissionOwner)
						// 更新到数据库
						if err := m.userRepo.Update(u); err != nil {
							// 更新失败，继续执行
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
