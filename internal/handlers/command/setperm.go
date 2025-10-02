package command

import (
	"fmt"
	"strings"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

// SetPermHandler 设置用户权限命令处理器
type SetPermHandler struct {
	*BaseCommand
	userRepo UserRepository
}

// NewSetPermHandler 创建设置权限命令处理器
func NewSetPermHandler(groupRepo GroupRepository, userRepo UserRepository) *SetPermHandler {
	return &SetPermHandler{
		BaseCommand: NewBaseCommand(
			"setperm",
			"直接设置用户权限等级",
			user.PermissionOwner, // 需要 Owner 权限
			[]string{"group", "supergroup"},
			groupRepo,
		),
		userRepo: userRepo,
	}
}

// Handle 处理命令
func (h *SetPermHandler) Handle(ctx *handler.Context) error {
	// 1. 检查权限（必须是 Owner）
	if err := h.CheckPermission(ctx); err != nil {
		return err
	}

	// 2. 解析参数
	args := ParseArgs(ctx.Text)
	if len(args) < 2 {
		return ctx.Reply("❌ 用法: /setperm @username <user|admin|superadmin|owner>")
	}

	username := strings.TrimPrefix(args[0], "@")
	permStr := strings.ToLower(args[1])

	// 3. 解析权限等级
	var newPerm user.Permission
	switch permStr {
	case "user":
		newPerm = user.PermissionUser
	case "admin":
		newPerm = user.PermissionAdmin
	case "superadmin":
		newPerm = user.PermissionSuperAdmin
	case "owner":
		newPerm = user.PermissionOwner
	default:
		return ctx.Reply("❌ 无效的权限等级，可选: user, admin, superadmin, owner")
	}

	// 4. 查找用户
	targetUser, err := h.userRepo.FindByUsername(username)
	if err != nil {
		return ctx.Reply(fmt.Sprintf("❌ 用户 @%s 不存在", username))
	}

	// 5. 获取当前权限
	currentPerm := targetUser.GetPermission(ctx.ChatID)

	// 6. 设置新权限
	targetUser.SetPermission(ctx.ChatID, newPerm)

	// 7. 保存到数据库
	if err := h.userRepo.Update(targetUser); err != nil {
		return ctx.Reply("❌ 权限更新失败，请稍后重试")
	}

	// 8. 成功反馈
	if currentPerm == newPerm {
		return ctx.Reply(fmt.Sprintf("✅ 用户 %s 权限保持不变: %s %s",
			FormatUsername(targetUser),
			newPerm.String(),
			GetPermIcon(newPerm)))
	}

	return ctx.Reply(fmt.Sprintf("✅ 用户 %s 权限已设置: %s → %s %s",
		FormatUsername(targetUser),
		currentPerm.String(),
		newPerm.String(),
		GetPermIcon(newPerm)))
}
