package command

import (
	"fmt"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

// PromoteHandler 提升用户权限命令处理器
type PromoteHandler struct {
	*BaseCommand
	userRepo UserRepository
}

// NewPromoteHandler 创建提升权限命令处理器
func NewPromoteHandler(groupRepo GroupRepository, userRepo UserRepository) *PromoteHandler {
	return &PromoteHandler{
		BaseCommand: NewBaseCommand(
			"promote",
			"提升用户权限一级",
			user.PermissionSuperAdmin, // 需要 SuperAdmin 权限
			[]string{"group", "supergroup"},
			groupRepo,
		),
		userRepo: userRepo,
	}
}

// Handle 处理命令
func (h *PromoteHandler) Handle(ctx *handler.Context) error {
	// 1. 检查权限
	if err := h.CheckPermission(ctx); err != nil {
		return err
	}

	// 2. 获取目标用户
	targetUser, err := GetTargetUser(ctx, h.userRepo)
	if err != nil {
		return ctx.Reply(fmt.Sprintf("❌ %s", err.Error()))
	}

	// 3. 获取当前权限
	currentPerm := targetUser.GetPermission(ctx.ChatID)

	// 4. 计算新权限
	newPerm := currentPerm + 1
	if newPerm > user.PermissionOwner {
		return ctx.Reply(fmt.Sprintf("❌ 用户 %s 已是最高权限 %s",
			FormatUsername(targetUser), currentPerm.String()))
	}

	// 5. 权限保护：不能提升到比自己高的等级
	if !ctx.User.HasPermission(ctx.ChatID, newPerm) {
		return ctx.Reply(fmt.Sprintf("❌ 您无权提升用户到 %s 等级（您的权限: %s）",
			newPerm.String(), ctx.User.GetPermission(ctx.ChatID).String()))
	}

	// 6. 设置新权限
	targetUser.SetPermission(ctx.ChatID, newPerm)

	// 7. 保存到数据库
	if err := h.userRepo.Update(targetUser); err != nil {
		return ctx.Reply("❌ 权限更新失败，请稍后重试")
	}

	// 8. 成功反馈
	return ctx.Reply(fmt.Sprintf("✅ 用户 %s 权限已提升: %s → %s %s",
		FormatUsername(targetUser),
		currentPerm.String(),
		newPerm.String(),
		GetPermIcon(newPerm)))
}
