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
		return ctx.ReplyHTML(fmt.Sprintf("❌ 您无权降低 <b>%s</b> 的权限\n目标权限: <b>%s</b>，您的权限: <b>%s</b>",
			FormatUsername(targetUser),
			currentPerm.String(),
			executorPerm.String()))
	}

	// 5. 计算新权限
	newPerm := currentPerm - 1
	if newPerm < user.PermissionUser {
		return ctx.ReplyHTML(fmt.Sprintf("❌ 用户 <b>%s</b> 已是最低权限 <b>%s</b>",
			FormatUsername(targetUser), currentPerm.String()))
	}

	// 6. 保存到数据库（使用细粒度更新避免并发冲突）
	if err := h.userRepo.UpdatePermission(targetUser.ID, ctx.ChatID, newPerm); err != nil {
		return ctx.Reply("❌ 权限更新失败，请稍后重试")
	}

	// 7. 更新本地对象（用于显示）
	targetUser.SetPermission(ctx.ChatID, newPerm)

	// 8. 成功反馈
	return ctx.ReplyHTML(fmt.Sprintf("✅ 用户 <b>%s</b> 权限已降低:\n<b>%s</b> → <b>%s</b> %s",
		FormatUsername(targetUser),
		currentPerm.String(),
		newPerm.String(),
		GetPermIcon(newPerm)))
}
