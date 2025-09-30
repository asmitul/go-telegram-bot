package help

import (
	"fmt"
	"sort"
	"strings"

	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/user"
)

// Handler Help 命令处理器
type Handler struct {
	registry command.Registry
}

// NewHandler 创建 Help 命令处理器
func NewHandler(registry command.Registry) *Handler {
	return &Handler{
		registry: registry,
	}
}

// Name 命令名称
func (h *Handler) Name() string {
	return "help"
}

// Description 命令描述
func (h *Handler) Description() string {
	return "显示所有可用命令"
}

// RequiredPermission 所需权限
func (h *Handler) RequiredPermission() user.Permission {
	return user.PermissionUser // 所有用户都可以使用
}

// IsEnabled 检查命令是否在群组中启用
func (h *Handler) IsEnabled(groupID int64) bool {
	return true // help 命令始终启用
}

// Handle 处理命令
func (h *Handler) Handle(ctx *command.Context) error {
	// 如果指定了命令名称，显示该命令详情
	if len(ctx.Args) > 0 {
		return h.showCommandDetail(ctx, ctx.Args[0])
	}

	// 否则显示所有可用命令
	return h.showAllCommands(ctx)
}

// showAllCommands 显示所有可用命令
func (h *Handler) showAllCommands(ctx *command.Context) error {
	handlers := h.registry.GetAll()
	if len(handlers) == 0 {
		return sendMessage(ctx, "暂无可用命令")
	}

	// 获取用户权限
	userPerm := user.PermissionUser
	if ctx.User != nil {
		userPerm = ctx.User.GetPermission(ctx.GroupID)
	}

	// 按权限分组
	groups := make(map[user.Permission][]command.Handler)
	for _, handler := range handlers {
		if !handler.IsEnabled(ctx.GroupID) {
			continue // 跳过已禁用的命令
		}
		reqPerm := handler.RequiredPermission()
		groups[reqPerm] = append(groups[reqPerm], handler)
	}

	// 构建响应
	var sb strings.Builder
	sb.WriteString("📖 *可用命令列表*\n\n")

	// 按权限级别排序显示
	permissions := []user.Permission{
		user.PermissionUser,
		user.PermissionAdmin,
		user.PermissionSuperAdmin,
		user.PermissionOwner,
	}

	for _, perm := range permissions {
		cmds, exists := groups[perm]
		if !exists || len(cmds) == 0 {
			continue
		}

		// 只显示用户有权限的命令
		if userPerm < perm {
			continue
		}

		// 按命令名称排序
		sort.Slice(cmds, func(i, j int) bool {
			return cmds[i].Name() < cmds[j].Name()
		})

		// 分组标题
		sb.WriteString(fmt.Sprintf("*%s*\n", getPermissionLabel(perm)))

		// 列出命令
		for _, cmd := range cmds {
			sb.WriteString(fmt.Sprintf("• `/%s` - %s\n", cmd.Name(), cmd.Description()))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("💡 使用 `/help <命令>` 查看命令详情")

	return sendMessage(ctx, sb.String())
}

// showCommandDetail 显示单个命令详情
func (h *Handler) showCommandDetail(ctx *command.Context, cmdName string) error {
	// 移除命令前缀 /
	cmdName = strings.TrimPrefix(cmdName, "/")

	handler, exists := h.registry.Get(cmdName)
	if !exists {
		return sendMessage(ctx, fmt.Sprintf("❌ 命令 `/%s` 不存在", cmdName))
	}

	// 检查命令是否启用
	if !handler.IsEnabled(ctx.GroupID) {
		return sendMessage(ctx, fmt.Sprintf("⚠️ 命令 `/%s` 已在本群禁用", cmdName))
	}

	// 检查用户权限
	userPerm := user.PermissionUser
	if ctx.User != nil {
		userPerm = ctx.User.GetPermission(ctx.GroupID)
	}

	reqPerm := handler.RequiredPermission()
	if userPerm < reqPerm {
		return sendMessage(ctx, fmt.Sprintf("⚠️ 你没有权限使用命令 `/%s`\n需要权限: %s", 
			cmdName, getPermissionLabel(reqPerm)))
	}

	// 构建详情信息
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 *命令详情*\n\n"))
	sb.WriteString(fmt.Sprintf("*命令*: `/%s`\n", handler.Name()))
	sb.WriteString(fmt.Sprintf("*描述*: %s\n", handler.Description()))
	sb.WriteString(fmt.Sprintf("*所需权限*: %s\n", getPermissionLabel(reqPerm)))
	sb.WriteString(fmt.Sprintf("*状态*: %s\n", getStatusEmoji(handler.IsEnabled(ctx.GroupID))))

	return sendMessage(ctx, sb.String())
}

// getPermissionLabel 获取权限标签
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

// getStatusEmoji 获取状态图标
func getStatusEmoji(enabled bool) string {
	if enabled {
		return "✅ 已启用"
	}
	return "❌ 已禁用"
}

// sendMessage 发送消息的辅助函数
func sendMessage(ctx *command.Context, text string) error {
	// 这里实际会调用 Telegram API
	// 为了保持命令独立，这个函数会被注入
	fmt.Printf("Send to group %d: %s\n", ctx.GroupID, text)
	return nil
}
