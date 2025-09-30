package welcome

import (
	"fmt"
	"strings"
	"time"

	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
)

const (
	DefaultWelcomeMessage = "欢迎 {user} 加入群组！"
	SettingKeyEnabled     = "welcome_enabled"
	SettingKeyMessage     = "welcome_message"
)

// Handler Welcome command handler
type Handler struct {
	groupRepo group.Repository
}

// NewHandler creates a new welcome command handler
func NewHandler(groupRepo group.Repository) *Handler {
	return &Handler{
		groupRepo: groupRepo,
	}
}

// Name returns the command name
func (h *Handler) Name() string {
	return "welcome"
}

// Description returns the command description
func (h *Handler) Description() string {
	return "设置欢迎消息"
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

// Handle processes the welcome command
func (h *Handler) Handle(ctx *command.Context) error {
	if len(ctx.Args) == 0 {
		return h.showWelcomeConfig(ctx)
	}

	subcommand := ctx.Args[0]
	switch subcommand {
	case "on":
		return h.enableWelcome(ctx)
	case "off":
		return h.disableWelcome(ctx)
	case "set":
		return h.setWelcomeMessage(ctx)
	case "reset":
		return h.resetWelcomeMessage(ctx)
	case "test":
		return h.testWelcomeMessage(ctx)
	default:
		return h.showHelp(ctx)
	}
}

// showWelcomeConfig shows current welcome configuration
func (h *Handler) showWelcomeConfig(ctx *command.Context) error {
	g, err := h.groupRepo.FindByID(ctx.GroupID)
	if err != nil {
		return sendMessage(ctx, "⚠️ 无法获取群组配置")
	}

	enabled := h.isWelcomeEnabled(g)
	message := h.getWelcomeMessage(g)

	status := "❌ 已禁用"
	if enabled {
		status = "✅ 已启用"
	}

	response := fmt.Sprintf("👋 *欢迎消息配置*\n\n")
	response += fmt.Sprintf("*状态*: %s\n", status)
	response += fmt.Sprintf("*消息*: %s\n\n", message)
	response += "*可用命令*:\n"
	response += "• `/welcome on` - 启用欢迎消息\n"
	response += "• `/welcome off` - 禁用欢迎消息\n"
	response += "• `/welcome set <消息>` - 设置欢迎消息\n"
	response += "• `/welcome reset` - 重置为默认消息\n"
	response += "• `/welcome test` - 测试欢迎消息\n\n"
	response += "*可用变量*:\n"
	response += "• `{user}` - 用户的显示名称\n"
	response += "• `{username}` - 用户名（@username）\n"
	response += "• `{userid}` - 用户ID\n"
	response += "• `{group}` - 群组名称\n"
	response += "• `{date}` - 当前日期\n"
	response += "• `{time}` - 当前时间"

	return sendMessage(ctx, response)
}

// enableWelcome enables welcome messages
func (h *Handler) enableWelcome(ctx *command.Context) error {
	g, err := h.groupRepo.FindByID(ctx.GroupID)
	if err != nil {
		g = group.NewGroup(ctx.GroupID, "Unknown", "supergroup")
	}

	g.SetSetting(SettingKeyEnabled, true)
	if err := h.groupRepo.Save(g); err != nil {
		return sendMessage(ctx, "❌ 启用欢迎消息失败")
	}

	return sendMessage(ctx, "✅ 欢迎消息已启用")
}

// disableWelcome disables welcome messages
func (h *Handler) disableWelcome(ctx *command.Context) error {
	g, err := h.groupRepo.FindByID(ctx.GroupID)
	if err != nil {
		return sendMessage(ctx, "⚠️ 无法获取群组配置")
	}

	g.SetSetting(SettingKeyEnabled, false)
	if err := h.groupRepo.Save(g); err != nil {
		return sendMessage(ctx, "❌ 禁用欢迎消息失败")
	}

	return sendMessage(ctx, "✅ 欢迎消息已禁用")
}

// setWelcomeMessage sets a custom welcome message
func (h *Handler) setWelcomeMessage(ctx *command.Context) error {
	if len(ctx.Args) < 2 {
		return sendMessage(ctx, "❌ 请提供欢迎消息内容\n\n用法: `/welcome set <消息>`")
	}

	message := strings.Join(ctx.Args[1:], " ")
	if len(message) > 500 {
		return sendMessage(ctx, "❌ 欢迎消息过长（最多500字符）")
	}

	g, err := h.groupRepo.FindByID(ctx.GroupID)
	if err != nil {
		g = group.NewGroup(ctx.GroupID, "Unknown", "supergroup")
	}

	g.SetSetting(SettingKeyMessage, message)
	if err := h.groupRepo.Save(g); err != nil {
		return sendMessage(ctx, "❌ 设置欢迎消息失败")
	}

	response := fmt.Sprintf("✅ 欢迎消息已设置为:\n\n%s", message)
	return sendMessage(ctx, response)
}

// resetWelcomeMessage resets welcome message to default
func (h *Handler) resetWelcomeMessage(ctx *command.Context) error {
	g, err := h.groupRepo.FindByID(ctx.GroupID)
	if err != nil {
		return sendMessage(ctx, "⚠️ 无法获取群组配置")
	}

	g.SetSetting(SettingKeyMessage, DefaultWelcomeMessage)
	if err := h.groupRepo.Save(g); err != nil {
		return sendMessage(ctx, "❌ 重置欢迎消息失败")
	}

	response := fmt.Sprintf("✅ 欢迎消息已重置为默认:\n\n%s", DefaultWelcomeMessage)
	return sendMessage(ctx, response)
}

// testWelcomeMessage tests the welcome message with current user
func (h *Handler) testWelcomeMessage(ctx *command.Context) error {
	g, err := h.groupRepo.FindByID(ctx.GroupID)
	if err != nil {
		return sendMessage(ctx, "⚠️ 无法获取群组配置")
	}

	message := h.getWelcomeMessage(g)
	testUser := &user.User{
		ID:        ctx.UserID,
		Username:  ctx.User.Username,
		FirstName: ctx.User.FirstName,
		LastName:  ctx.User.LastName,
	}

	welcomeMsg := h.formatWelcomeMessage(message, testUser, g)
	response := fmt.Sprintf("🧪 *测试欢迎消息*\n\n%s", welcomeMsg)
	return sendMessage(ctx, response)
}

// showHelp shows help for the welcome command
func (h *Handler) showHelp(ctx *command.Context) error {
	response := "❌ 未知的子命令\n\n"
	response += "*可用命令*:\n"
	response += "• `/welcome` - 查看当前配置\n"
	response += "• `/welcome on` - 启用欢迎消息\n"
	response += "• `/welcome off` - 禁用欢迎消息\n"
	response += "• `/welcome set <消息>` - 设置欢迎消息\n"
	response += "• `/welcome reset` - 重置为默认消息\n"
	response += "• `/welcome test` - 测试欢迎消息"

	return sendMessage(ctx, response)
}

// OnNewMember handles new member join event
func (h *Handler) OnNewMember(groupID int64, newUser *user.User) (string, bool) {
	g, err := h.groupRepo.FindByID(groupID)
	if err != nil {
		// Default behavior: send welcome message
		return h.formatWelcomeMessage(DefaultWelcomeMessage, newUser, nil), true
	}

	// Check if welcome is enabled
	if !h.isWelcomeEnabled(g) {
		return "", false
	}

	message := h.getWelcomeMessage(g)
	return h.formatWelcomeMessage(message, newUser, g), true
}

// isWelcomeEnabled checks if welcome message is enabled for the group
func (h *Handler) isWelcomeEnabled(g *group.Group) bool {
	if enabled, ok := g.GetSetting(SettingKeyEnabled); ok {
		if enabledBool, ok := enabled.(bool); ok {
			return enabledBool
		}
	}
	// Default: enabled
	return true
}

// getWelcomeMessage gets the welcome message for the group
func (h *Handler) getWelcomeMessage(g *group.Group) string {
	if message, ok := g.GetSetting(SettingKeyMessage); ok {
		if messageStr, ok := message.(string); ok {
			return messageStr
		}
	}
	return DefaultWelcomeMessage
}

// formatWelcomeMessage formats the welcome message with variables
func (h *Handler) formatWelcomeMessage(template string, u *user.User, g *group.Group) string {
	message := template

	// User display name (FirstName + LastName)
	displayName := u.FirstName
	if u.LastName != "" {
		displayName += " " + u.LastName
	}
	message = strings.ReplaceAll(message, "{user}", displayName)

	// Username
	username := u.Username
	if username != "" {
		username = "@" + username
	} else {
		username = displayName
	}
	message = strings.ReplaceAll(message, "{username}", username)

	// User ID
	message = strings.ReplaceAll(message, "{userid}", fmt.Sprintf("%d", u.ID))

	// Group name
	groupName := "群组"
	if g != nil {
		groupName = g.Title
	}
	message = strings.ReplaceAll(message, "{group}", groupName)

	// Date and time
	now := time.Now()
	message = strings.ReplaceAll(message, "{date}", now.Format("2006-01-02"))
	message = strings.ReplaceAll(message, "{time}", now.Format("15:04:05"))

	return message
}

// sendMessage helper function to send message
func sendMessage(ctx *command.Context, text string) error {
	// This will be called by Telegram API
	fmt.Printf("Send to group %d: %s\n", ctx.GroupID, text)
	return nil
}
