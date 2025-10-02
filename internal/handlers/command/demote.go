package command

import (
	"fmt"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

// DemoteHandler 降低用户权限命令处理器
type DemoteHandler struct {
	*BaseCommand
	userRepo UserRepository
}

// NewDemoteHandler 创建降低权限命令处理器
func NewDemoteHandler(groupRepo GroupRepository, userRepo UserRepository) *DemoteHandler {
	return &DemoteHandler{
		BaseCommand: NewBaseCommand(
			"demote",
			"降低用户权限一级",
			user.PermissionSuperAdmin, // 需要 SuperAdmin 权限
			[]string{"group", "supergroup"},
			groupRepo,
		),
		userRepo: userRepo,
	}
}

// Handle 处理命令
func (h *DemoteHandler) Handle(ctx *handler.Context) error {
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

	// 4. 权限保护：不能降低比自己高或相等的用户
	executorPerm := ctx.User.GetPermission(ctx.ChatID)
	if currentPerm >= executorPerm {
		return ctx.Reply(fmt.Sprintf("❌ 您无权降低 %s 的权限（目标权限: %s，您的权限: %s）",
			FormatUsername(targetUser),
			currentPerm.String(),
			executorPerm.String()))
	}

	// 5. 计算新权限
	newPerm := currentPerm - 1
	if newPerm < user.PermissionUser {
		return ctx.Reply(fmt.Sprintf("❌ 用户 %s 已是最低权限 %s",
			FormatUsername(targetUser), currentPerm.String()))
	}

	// 6. 设置新权限
	targetUser.SetPermission(ctx.ChatID, newPerm)

	// 7. 保存到数据库
	if err := h.userRepo.Update(targetUser); err != nil {
		return ctx.Reply("❌ 权限更新失败，请稍后重试")
	}

	// 8. 成功反馈
	return ctx.Reply(fmt.Sprintf("✅ 用户 %s 权限已降低: %s → %s %s",
		FormatUsername(targetUser),
		currentPerm.String(),
		newPerm.String(),
		GetPermIcon(newPerm)))
}
