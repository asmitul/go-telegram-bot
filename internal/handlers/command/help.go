package command

import (
	"fmt"
	"sort"
	"strings"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

// CommandInfo 命令信息接口
// 所有嵌入 BaseCommand 的命令处理器都实现了此接口
type CommandInfo interface {
	GetName() string
	GetDescription() string
	GetPermission() user.Permission
}

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
	sb.WriteString("📖 <b>可用命令列表</b>\n\n")

	// 获取所有命令信息
	commands := h.getCommands()

	// 按权限等级分组显示
	userCommands := []string{}
	adminCommands := []string{}
	superAdminCommands := []string{}
	ownerCommands := []string{}

	for _, cmd := range commands {
		formattedCmd := h.formatCommand(cmd.Name, cmd.Description, cmd.Permission)
		switch cmd.Permission {
		case user.PermissionOwner:
			ownerCommands = append(ownerCommands, formattedCmd)
		case user.PermissionSuperAdmin:
			superAdminCommands = append(superAdminCommands, formattedCmd)
		case user.PermissionAdmin:
			adminCommands = append(adminCommands, formattedCmd)
		default:
			userCommands = append(userCommands, formattedCmd)
		}
	}

	// 输出各级别命令
	if len(userCommands) > 0 {
		sb.WriteString("✅ <b>基础命令</b>\n")
		for _, cmd := range userCommands {
			sb.WriteString(cmd)
		}
		sb.WriteString("\n")
	}

	if len(adminCommands) > 0 {
		sb.WriteString("🔧 <b>管理员命令</b>\n")
		for _, cmd := range adminCommands {
			sb.WriteString(cmd)
		}
		sb.WriteString("\n")
	}

	if len(superAdminCommands) > 0 {
		sb.WriteString("⭐ <b>超级管理员命令</b>\n")
		for _, cmd := range superAdminCommands {
			sb.WriteString(cmd)
		}
		sb.WriteString("\n")
	}

	if len(ownerCommands) > 0 {
		sb.WriteString("👑 <b>群主命令</b>\n")
		for _, cmd := range ownerCommands {
			sb.WriteString(cmd)
		}
		sb.WriteString("\n")
	}

	// 自动功能说明
	sb.WriteString("🤖 <b>自动功能</b>\n")
	sb.WriteString("✅ <b>数学计算器</b> - 自动计算数学表达式\n")
	sb.WriteString("   • 支持：加减乘除 (+, -, *, /)，括号\n")
	sb.WriteString("   • 示例：<code>1+2</code>, <code>(10+5)*2</code>, <code>100/4</code>\n")
	sb.WriteString("   • 管理：<code>/togglecalc</code> 开启/关闭（需要 Admin 权限）\n")
	sb.WriteString("\n")

	sb.WriteString("💡 提示：使用 <code>/命令名</code> 执行命令")

	return ctx.ReplyHTML(sb.String())
}

// CommandData 命令数据
type CommandData struct {
	Name        string
	Description string
	Permission  user.Permission
}

// getCommands 获取所有命令信息
func (h *HelpHandler) getCommands() []CommandData {
	commands := []CommandData{}

	// 遍历所有处理器
	handlers := h.router.GetHandlers()
	for _, hdlr := range handlers {
		// 尝试类型断言为 CommandInfo 接口
		if cmdInfo, ok := hdlr.(CommandInfo); ok {
			commands = append(commands, CommandData{
				Name:        cmdInfo.GetName(),
				Description: cmdInfo.GetDescription(),
				Permission:  cmdInfo.GetPermission(),
			})
		}
	}

	// 按命令名排序
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Name < commands[j].Name
	})

	return commands
}

func (h *HelpHandler) formatCommand(name, desc string, perm user.Permission) string {
	permIcon := h.getPermissionIcon(perm)
	return fmt.Sprintf("%s <code>/%s</code> - %s\n", permIcon, name, desc)
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
