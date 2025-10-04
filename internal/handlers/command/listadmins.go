package command

import (
	"fmt"
	"strings"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

// ListAdminsHandler 查看管理员列表命令处理器
type ListAdminsHandler struct {
	*BaseCommand
	userRepo UserRepository
}

// NewListAdminsHandler 创建查看管理员列表命令处理器
func NewListAdminsHandler(groupRepo GroupRepository, userRepo UserRepository) *ListAdminsHandler {
	return &ListAdminsHandler{
		BaseCommand: NewBaseCommand(
			"listadmins",
			"查看当前群组管理员列表",
			user.PermissionUser, // 所有人可查看
			[]string{"group", "supergroup"},
			groupRepo,
		),
		userRepo: userRepo,
	}
}

// Handle 处理命令
func (h *ListAdminsHandler) Handle(ctx *handler.Context) error {
	// 1. 检查权限
	if err := h.CheckPermission(ctx); err != nil {
		return err
	}

	// 2. 查询所有管理员
	admins, err := h.userRepo.FindAdminsByGroup(ctx.ChatID)
	if err != nil {
		return ctx.Reply("❌ 查询管理员列表失败，请稍后重试")
	}

	// 3. 按权限等级分组
	owners := []string{}
	superAdmins := []string{}
	regularAdmins := []string{}

	for _, admin := range admins {
		perm := admin.GetPermission(ctx.ChatID)
		username := FormatUsername(admin)

		switch perm {
		case user.PermissionOwner:
			owners = append(owners, username)
		case user.PermissionSuperAdmin:
			superAdmins = append(superAdmins, username)
		case user.PermissionAdmin:
			regularAdmins = append(regularAdmins, username)
		}
	}

	// 4. 构建输出
	var sb strings.Builder
	sb.WriteString("👥 <b>当前群组管理员列表</b>\n\n")

	if len(owners) > 0 {
		sb.WriteString(fmt.Sprintf("👑 <b>Owner</b> (%d人):\n", len(owners)))
		for _, u := range owners {
			sb.WriteString(fmt.Sprintf("  • %s\n", u))
		}
		sb.WriteString("\n")
	}

	if len(superAdmins) > 0 {
		sb.WriteString(fmt.Sprintf("⭐ <b>SuperAdmin</b> (%d人):\n", len(superAdmins)))
		for _, u := range superAdmins {
			sb.WriteString(fmt.Sprintf("  • %s\n", u))
		}
		sb.WriteString("\n")
	}

	if len(regularAdmins) > 0 {
		sb.WriteString(fmt.Sprintf("🛡 <b>Admin</b> (%d人):\n", len(regularAdmins)))
		for _, u := range regularAdmins {
			sb.WriteString(fmt.Sprintf("  • %s\n", u))
		}
		sb.WriteString("\n")
	}

	total := len(owners) + len(superAdmins) + len(regularAdmins)
	if total == 0 {
		return ctx.Reply("👥 当前群组暂无管理员")
	}

	sb.WriteString(fmt.Sprintf("总计: <b>%d</b> 位管理员", total))

	return ctx.ReplyHTML(sb.String())
}
