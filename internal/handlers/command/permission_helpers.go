package command

import (
	"context"
	"fmt"
	"strings"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

// GetTargetUser 从参数或回复消息中获取目标用户
func GetTargetUser(reqCtx context.Context, ctx *handler.Context, userRepo UserRepository) (*user.User, error) {
	// 方式 1: 从参数获取 @username
	args := ParseArgs(ctx.Text)
	if len(args) > 0 {
		username := strings.TrimPrefix(args[0], "@")
		u, err := userRepo.FindByUsername(reqCtx, username)
		if err != nil {
			// 包装数据库错误，避免暴露内部细节
			if err == user.ErrUserNotFound {
				return nil, fmt.Errorf("用户 @%s 不存在或未使用过此机器人", username)
			}
			return nil, fmt.Errorf("查询用户失败，请稍后重试")
		}
		return u, nil
	}

	// 方式 2: 从回复消息获取
	if ctx.ReplyTo != nil {
		u, err := userRepo.FindByID(reqCtx, ctx.ReplyTo.UserID)
		if err != nil {
			// 包装数据库错误
			if err == user.ErrUserNotFound {
				return nil, fmt.Errorf("回复的用户不存在或未使用过此机器人")
			}
			return nil, fmt.Errorf("查询用户失败，请稍后重试")
		}
		return u, nil
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
