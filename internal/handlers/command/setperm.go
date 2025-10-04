package command

import (
	"context"
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
	reqCtx := context.TODO()

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
	targetUser, err := h.userRepo.FindByUsername(reqCtx, username)
	if err != nil {
		// 包装错误，避免暴露数据库细节
		if err == user.ErrUserNotFound {
			return ctx.Reply(fmt.Sprintf("❌ 用户 @%s 不存在或未使用过此机器人", username))
		}
		return ctx.Reply("❌ 查询用户失败，请稍后重试")
	}

	// 5. 获取当前权限
	currentPerm := targetUser.GetPermission(ctx.ChatID)
	executorPerm := ctx.User.GetPermission(ctx.ChatID)

	// 5.1. 权限保护：不能修改自己的权限
	if targetUser.ID == ctx.UserID {
		return ctx.Reply("❌ 不能修改自己的权限")
	}

	// 5.2. 权限保护：不能修改同级或更高级别的用户
	if currentPerm >= executorPerm {
		return ctx.ReplyHTML(fmt.Sprintf("❌ 您无权修改 <b>%s</b> 的权限\n目标权限: <b>%s</b>，您的权限: <b>%s</b>",
			FormatUsername(targetUser),
			currentPerm.String(),
			executorPerm.String()))
	}

	// 6. 保存到数据库（使用细粒度更新避免并发冲突）
	if err := h.userRepo.UpdatePermission(reqCtx, targetUser.ID, ctx.ChatID, newPerm); err != nil {
		return ctx.Reply("❌ 权限更新失败，请稍后重试")
	}

	// 7. 更新本地对象（用于显示）
	targetUser.SetPermission(ctx.ChatID, newPerm)

	// 8. 成功反馈
	if currentPerm == newPerm {
		return ctx.ReplyHTML(fmt.Sprintf("✅ 用户 <b>%s</b> 权限保持不变:\n<b>%s</b> %s",
			FormatUsername(targetUser),
			newPerm.String(),
			GetPermIcon(newPerm)))
	}

	return ctx.ReplyHTML(fmt.Sprintf("✅ 用户 <b>%s</b> 权限已设置:\n<b>%s</b> → <b>%s</b> %s",
		FormatUsername(targetUser),
		currentPerm.String(),
		newPerm.String(),
		GetPermIcon(newPerm)))
}
