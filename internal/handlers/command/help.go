package command

import (
	"fmt"
	"strings"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

// HelpHandler Help 命令处理器
type HelpHandler struct {
	*BaseCommand
	router *handler.Router // 用于获取所有命令
}

// NewHelpHandler 创建 Help 命令处理器
func NewHelpHandler(groupRepo GroupRepository, router *handler.Router) *HelpHandler {
	return &HelpHandler{
		BaseCommand: NewBaseCommand(
			"help",
			"显示帮助信息",
			user.PermissionUser,
			[]string{"private", "group", "supergroup"},
			groupRepo,
		),
		router: router,
	}
}

// Handle 处理命令
func (h *HelpHandler) Handle(ctx *handler.Context) error {
	// 权限检查
	if err := h.CheckPermission(ctx); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("📖 *可用命令列表*\n\n")

	// 遍历所有处理器，找出命令
	handlers := h.router.GetHandlers()
	for _, hdlr := range handlers {
		// 只显示命令类型的处理器
		if cmd, ok := hdlr.(*PingHandler); ok {
			sb.WriteString(h.formatCommand(cmd.GetName(), cmd.GetDescription(), cmd.GetPermission()))
		} else if cmd, ok := hdlr.(*HelpHandler); ok {
			sb.WriteString(h.formatCommand(cmd.GetName(), cmd.GetDescription(), cmd.GetPermission()))
		} else if cmd, ok := hdlr.(*StatsHandler); ok {
			sb.WriteString(h.formatCommand(cmd.GetName(), cmd.GetDescription(), cmd.GetPermission()))
		}
		// 可以继续添加其他命令类型...
	}

	sb.WriteString("\n💡 提示：使用 `/命令名` 执行命令")

	return ctx.ReplyMarkdown(sb.String())
}

func (h *HelpHandler) formatCommand(name, desc string, perm user.Permission) string {
	permIcon := h.getPermissionIcon(perm)
	return fmt.Sprintf("%s `/%s` - %s\n", permIcon, name, desc)
}

func (h *HelpHandler) getPermissionIcon(perm user.Permission) string {
	switch perm {
	case user.PermissionOwner:
		return "👑"
	case user.PermissionSuperAdmin:
		return "⭐"
	case user.PermissionAdmin:
		return "🔧"
	default:
		return "✅"
	}
}
