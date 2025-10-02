package command

import (
	"fmt"
	"strings"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

// MyPermHandler 查看自己权限命令处理器
type MyPermHandler struct {
	*BaseCommand
}

// NewMyPermHandler 创建查看自己权限命令处理器
func NewMyPermHandler(groupRepo GroupRepository) *MyPermHandler {
	return &MyPermHandler{
		BaseCommand: NewBaseCommand(
			"myperm",
			"查看自己的权限信息",
			user.PermissionUser, // 所有人可查看
			[]string{"group", "supergroup", "private"},
			groupRepo,
		),
	}
}

// Handle 处理命令
func (h *MyPermHandler) Handle(ctx *handler.Context) error {
	// 获取当前群组/私聊的 ID
	groupID := ctx.ChatID
	if ctx.IsPrivate() {
		groupID = ctx.UserID
	}

	perm := ctx.User.GetPermission(groupID)

	var sb strings.Builder
	sb.WriteString("👤 您的权限信息:\n\n")

	// 群组/私聊名称
	if ctx.IsPrivate() {
		sb.WriteString("环境: 私聊\n")
	} else {
		sb.WriteString(fmt.Sprintf("群组: %s\n", ctx.ChatTitle))
	}

	// 用户信息
	sb.WriteString(fmt.Sprintf("用户: %s\n", FormatUsername(ctx.User)))
	sb.WriteString(fmt.Sprintf("权限等级: %s %s\n\n", perm.String(), GetPermIcon(perm)))

	// 权限说明
	sb.WriteString("您可以:\n")

	switch perm {
	case user.PermissionOwner:
		sb.WriteString("✅ 所有权限（群主）\n")
		sb.WriteString("✅ 使用所有命令\n")
		sb.WriteString("✅ 提升/降低用户权限\n")
		sb.WriteString("✅ 直接设置任意用户权限\n")
		sb.WriteString("✅ 管理群组配置\n")
	case user.PermissionSuperAdmin:
		sb.WriteString("✅ 使用所有用户命令\n")
		sb.WriteString("✅ 使用管理员命令\n")
		sb.WriteString("✅ 提升/降低用户权限\n")
		sb.WriteString("✅ 管理群组配置\n")
	case user.PermissionAdmin:
		sb.WriteString("✅ 使用所有用户命令\n")
		sb.WriteString("✅ 使用管理员命令\n")
		sb.WriteString("✅ 查看群组统计\n")
	case user.PermissionUser:
		sb.WriteString("✅ 使用基础用户命令\n")
		sb.WriteString("✅ 查看公开信息\n")
	default:
		sb.WriteString("⚠️ 无权限\n")
	}

	return ctx.Reply(sb.String())
}
