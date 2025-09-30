package manage

import (
	"fmt"
	"sort"
	"strings"

	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
)

// Handler Manage commands handler
type Handler struct {
	groupRepo group.Repository
	registry  command.Registry
}

// NewHandler creates a new manage commands handler
func NewHandler(groupRepo group.Repository, registry command.Registry) *Handler {
	return &Handler{
		groupRepo: groupRepo,
		registry:  registry,
	}
}

// Name returns the command name
func (h *Handler) Name() string {
	return "manage"
}

// Description returns the command description
func (h *Handler) Description() string {
	return "管理群组命令"
}

// RequiredPermission returns the required permission
func (h *Handler) RequiredPermission() user.Permission {
	return user.PermissionAdmin
}

// IsEnabled checks if the command is enabled in the group
func (h *Handler) IsEnabled(groupID int64) bool {
	g, err := h.groupRepo.FindByID(groupID)
	if err != nil {
		return true
	}
	return g.IsCommandEnabled(h.Name())
}

// Handle processes the manage command
func (h *Handler) Handle(ctx *command.Context) error {
	if len(ctx.Args) == 0 {
		return h.showCommands(ctx)
	}

	subcommand := ctx.Args[0]
	switch subcommand {
	case "enable":
		return h.enableCommand(ctx)
	case "disable":
		return h.disableCommand(ctx)
	case "list":
		return h.showCommands(ctx)
	case "status":
		return h.showCommandStatus(ctx)
	default:
		return h.showHelp(ctx)
	}
}

// showCommands shows all commands and their status
func (h *Handler) showCommands(ctx *command.Context) error {
	g, err := h.groupRepo.FindByID(ctx.GroupID)
	if err != nil {
		return sendMessage(ctx, "⚠️ 无法获取群组配置")
	}

	allCommands := h.registry.GetAll()
	if len(allCommands) == 0 {
		return sendMessage(ctx, "⚠️ 没有可用的命令")
	}

	// Sort commands by name
	sort.Slice(allCommands, func(i, j int) bool {
		return allCommands[i].Name() < allCommands[j].Name()
	})

	response := "📋 *命令管理*\n\n"
	
	enabledCommands := []string{}
	disabledCommands := []string{}

	for _, cmd := range allCommands {
		cmdName := cmd.Name()
		status := "✅"
		if !g.IsCommandEnabled(cmdName) {
			status = "❌"
			disabledCommands = append(disabledCommands, cmdName)
		} else {
			enabledCommands = append(enabledCommands, cmdName)
		}
		
		permLabel := getPermissionLabel(cmd.RequiredPermission())
		response += fmt.Sprintf("%s `/%s` - %s (%s)\n", 
			status, cmdName, cmd.Description(), permLabel)
	}

	response += fmt.Sprintf("\n*统计*: %d 启用, %d 禁用\n\n",
		len(enabledCommands), len(disabledCommands))
	
	response += "*可用命令*:\n"
	response += "• `/manage enable <命令>` - 启用命令\n"
	response += "• `/manage disable <命令>` - 禁用命令\n"
	response += "• `/manage status <命令>` - 查看命令状态\n"
	response += "• `/manage list` - 查看所有命令"

	return sendMessage(ctx, response)
}

// enableCommand enables a command for the group
func (h *Handler) enableCommand(ctx *command.Context) error {
	if len(ctx.Args) < 2 {
		return sendMessage(ctx, "❌ 请指定要启用的命令\n\n用法: `/manage enable <命令>`")
	}

	commandName := strings.TrimPrefix(ctx.Args[1], "/")

	// Check if command exists
	if _, exists := h.registry.Get(commandName); !exists {
		return sendMessage(ctx, fmt.Sprintf("❌ 命令 `%s` 不存在", commandName))
	}

	// Get or create group
	g, err := h.groupRepo.FindByID(ctx.GroupID)
	if err != nil {
		g = group.NewGroup(ctx.GroupID, "Unknown", "supergroup")
	}

	// Enable command
	g.EnableCommand(commandName, ctx.UserID)
	if err := h.groupRepo.Save(g); err != nil {
		return sendMessage(ctx, fmt.Sprintf("❌ 启用命令 `%s` 失败", commandName))
	}

	return sendMessage(ctx, fmt.Sprintf("✅ 命令 `/%s` 已启用", commandName))
}

// disableCommand disables a command for the group
func (h *Handler) disableCommand(ctx *command.Context) error {
	if len(ctx.Args) < 2 {
		return sendMessage(ctx, "❌ 请指定要禁用的命令\n\n用法: `/manage disable <命令>`")
	}

	commandName := strings.TrimPrefix(ctx.Args[1], "/")

	// Check if command exists
	if _, exists := h.registry.Get(commandName); !exists {
		return sendMessage(ctx, fmt.Sprintf("❌ 命令 `%s` 不存在", commandName))
	}

	// Prevent disabling critical commands
	if commandName == "manage" {
		return sendMessage(ctx, "❌ 不能禁用 `manage` 命令")
	}

	// Get group
	g, err := h.groupRepo.FindByID(ctx.GroupID)
	if err != nil {
		return sendMessage(ctx, "⚠️ 无法获取群组配置")
	}

	// Disable command
	g.DisableCommand(commandName, ctx.UserID)
	if err := h.groupRepo.Save(g); err != nil {
		return sendMessage(ctx, fmt.Sprintf("❌ 禁用命令 `%s` 失败", commandName))
	}

	return sendMessage(ctx, fmt.Sprintf("✅ 命令 `/%s` 已禁用", commandName))
}

// showCommandStatus shows status of a specific command
func (h *Handler) showCommandStatus(ctx *command.Context) error {
	if len(ctx.Args) < 2 {
		return sendMessage(ctx, "❌ 请指定要查看的命令\n\n用法: `/manage status <命令>`")
	}

	commandName := strings.TrimPrefix(ctx.Args[1], "/")

	// Check if command exists
	handler, exists := h.registry.Get(commandName)
	if !exists {
		return sendMessage(ctx, fmt.Sprintf("❌ 命令 `%s` 不存在", commandName))
	}

	g, err := h.groupRepo.FindByID(ctx.GroupID)
	if err != nil {
		return sendMessage(ctx, "⚠️ 无法获取群组配置")
	}

	enabled := g.IsCommandEnabled(commandName)
	statusText := "❌ 已禁用"
	if enabled {
		statusText = "✅ 已启用"
	}

	config := g.GetCommandConfig(commandName)
	updatedBy := "系统"
	if config.UpdatedBy > 0 {
		updatedBy = fmt.Sprintf("用户 %d", config.UpdatedBy)
	}

	response := fmt.Sprintf("📊 *命令状态: %s*\n\n", commandName)
	response += fmt.Sprintf("*名称*: `/%s`\n", handler.Name())
	response += fmt.Sprintf("*描述*: %s\n", handler.Description())
	response += fmt.Sprintf("*所需权限*: %s\n", getPermissionLabel(handler.RequiredPermission()))
	response += fmt.Sprintf("*当前状态*: %s\n", statusText)
	response += fmt.Sprintf("*最后更新*: %s\n", config.UpdatedAt.Format("2006-01-02 15:04:05"))
	response += fmt.Sprintf("*更新者*: %s", updatedBy)

	return sendMessage(ctx, response)
}

// showHelp shows help for the manage command
func (h *Handler) showHelp(ctx *command.Context) error {
	response := "❌ 未知的子命令\n\n"
	response += "*可用命令*:\n"
	response += "• `/manage` 或 `/manage list` - 查看所有命令\n"
	response += "• `/manage enable <命令>` - 启用命令\n"
	response += "• `/manage disable <命令>` - 禁用命令\n"
	response += "• `/manage status <命令>` - 查看命令状态"

	return sendMessage(ctx, response)
}

// getPermissionLabel returns a human-readable permission label
func getPermissionLabel(perm user.Permission) string {
	switch perm {
	case user.PermissionNone:
		return "🚫 无权限"
	case user.PermissionUser:
		return "👤 普通用户"
	case user.PermissionAdmin:
		return "👮 管理员"
	case user.PermissionSuperAdmin:
		return "⭐ 超级管理员"
	case user.PermissionOwner:
		return "👑 群主"
	default:
		return "❓ 未知"
	}
}

// sendMessage helper function to send message
func sendMessage(ctx *command.Context, text string) error {
	// This will be called by Telegram API
	fmt.Printf("Send to group %d: %s\n", ctx.GroupID, text)
	return nil
}
