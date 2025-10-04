package command

import (
	"fmt"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

// StatsHandler Stats 命令处理器
type StatsHandler struct {
	*BaseCommand
	userRepo  UserRepository
	groupRepo GroupRepository
}

// NewStatsHandler 创建 Stats 命令处理器
func NewStatsHandler(groupRepo GroupRepository, userRepo UserRepository) *StatsHandler {
	return &StatsHandler{
		BaseCommand: NewBaseCommand(
			"stats",
			"查看群组统计信息",
			user.PermissionAdmin, // 需要管理员权限
			[]string{"group", "supergroup"},
			groupRepo,
		),
		userRepo:  userRepo,
		groupRepo: groupRepo,
	}
}

// Handle 处理命令
func (h *StatsHandler) Handle(ctx *handler.Context) error {
	// 权限检查
	if err := h.CheckPermission(ctx); err != nil {
		return err
	}

	// 群组信息由 GroupMiddleware 自动注入
	if ctx.Group == nil {
		return fmt.Errorf("❌ 无法获取群组信息，请稍后重试")
	}

	// 构建统计信息
	response := fmt.Sprintf(
		"📊 <b>群组统计</b>\n\n"+
			"🏷️ 群组名称: <b>%s</b>\n"+
			"🆔 群组 ID: <code>%d</code>\n"+
			"📅 创建时间: %s\n",
		ctx.ChatTitle,
		ctx.ChatID,
		ctx.Group.CreatedAt.Format("2006-01-02 15:04:05"),
	)

	return ctx.ReplyHTML(response)
}
