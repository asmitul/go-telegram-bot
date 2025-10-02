package command

import (
	"fmt"
	"strings"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

// GetTargetUser 从参数或回复消息中获取目标用户
func GetTargetUser(ctx *handler.Context, userRepo UserRepository) (*user.User, error) {
	// 方式 1: 从参数获取 @username
	args := ParseArgs(ctx.Text)
	if len(args) > 0 {
		username := strings.TrimPrefix(args[0], "@")
		return userRepo.FindByUsername(username)
	}

	// 方式 2: 从回复消息获取
	if ctx.ReplyTo != nil {
		return userRepo.FindByID(ctx.ReplyTo.UserID)
	}

	return nil, fmt.Errorf("未指定目标用户，请使用 @username 或回复用户消息")
}

// GetPermIcon 获取权限图标
func GetPermIcon(perm user.Permission) string {
	switch perm {
	case user.PermissionOwner:
		return "👑"
	case user.PermissionSuperAdmin:
		return "⭐"
	case user.PermissionAdmin:
		return "🛡"
	case user.PermissionUser:
		return "👤"
	default:
		return "❓"
	}
}

// FormatUsername 格式化用户名显示
func FormatUsername(u *user.User) string {
	if u.Username != "" {
		return "@" + u.Username
	}
	if u.FirstName != "" {
		return u.FirstName
	}
	return fmt.Sprintf("User#%d", u.ID)
}
