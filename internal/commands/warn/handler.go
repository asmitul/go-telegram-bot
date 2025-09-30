package warn

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/user"
	"telegram-bot/pkg/validator"
	"time"
)

const (
	// MaxWarnings 最大警告次数，超过则自动踢出
	MaxWarnings = 3
)

var (
	ErrMissingReason   = errors.New("missing warning reason")
	ErrMissingUserID   = errors.New("missing user ID")
	ErrMaxWarnings     = errors.New("user reached maximum warnings and will be kicked")
)

// TelegramAPI 定义与 Telegram API 交互的接口
type TelegramAPI interface {
	SendMessage(chatID int64, text string) error
	SendMessageWithReply(chatID int64, text string, replyToMessageID int) error
	BanChatMember(chatID, userID int64) error
}

// UserRepository 用户仓储接口
type UserRepository interface {
	FindByID(userID int64, groupID int64) (*user.User, error)
}

// Handler 处理 warn 相关命令
type Handler struct {
	telegramAPI   TelegramAPI
	userRepo      UserRepository
	warningRepo   user.WarningRepository
}

// NewHandler 创建新的 Handler
func NewHandler(telegramAPI TelegramAPI, userRepo UserRepository, warningRepo user.WarningRepository) *Handler {
	return &Handler{
		telegramAPI: telegramAPI,
		userRepo:    userRepo,
		warningRepo: warningRepo,
	}
}

// Name 返回命令名称
func (h *Handler) Name() string {
	return "warn"
}

// Description 返回命令描述
func (h *Handler) Description() string {
	return "警告用户"
}

// RequiredPermission 返回所需权限
func (h *Handler) RequiredPermission() user.Permission {
	return user.PermissionAdmin
}

// Handle 处理命令
func (h *Handler) Handle(ctx *command.Context) error {
	// 检查子命令
	if len(ctx.Args) > 0 {
		switch ctx.Args[0] {
		case "warnings", "list":
			return h.handleWarnings(ctx)
		case "clear", "clearwarn":
			return h.handleClearWarn(ctx)
		}
	}

	// 默认是警告命令
	return h.handleWarn(ctx)
}

// handleWarn 处理警告命令
func (h *Handler) handleWarn(ctx *command.Context) error {
	var targetUserID int64
	var reason string
	var err error

	// 情况1：回复消息警告
	if ctx.ReplyToMessage != nil {
		targetUserID = ctx.ReplyToMessage.UserID

		// 原因是所有参数
		if len(ctx.Args) == 0 {
			return ErrMissingReason
		}
		reason = strings.Join(ctx.Args, " ")
	} else {
		// 情况2：指定用户 ID 警告
		if len(ctx.Args) < 2 {
			return h.showHelp(ctx)
		}

		// 解析用户 ID
		targetUserID, err = strconv.ParseInt(ctx.Args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid user ID: %v", err)
		}

		// 原因是后续参数
		reason = strings.Join(ctx.Args[1:], " ")
	}

	// 验证用户 ID
	if err := validator.UserID(targetUserID); err != nil {
		return err
	}

	// 验证群组 ID
	if err := validator.GroupID(ctx.GroupID); err != nil {
		return err
	}

	// 验证原因
	if strings.TrimSpace(reason) == "" {
		return ErrMissingReason
	}

	// 获取目标用户信息
	targetUser, err := h.userRepo.FindByID(targetUserID, ctx.GroupID)
	if err != nil {
		return fmt.Errorf("failed to find user: %v", err)
	}

	// 创建警告记录
	warning := user.NewWarning(targetUserID, ctx.GroupID, reason, ctx.UserID)
	if err := h.warningRepo.Save(warning); err != nil {
		return fmt.Errorf("failed to save warning: %v", err)
	}

	// 统计当前警告数量
	count, err := h.warningRepo.CountActiveWarnings(targetUserID, ctx.GroupID)
	if err != nil {
		return fmt.Errorf("failed to count warnings: %v", err)
	}

	// 构建响应消息
	message := h.formatWarnMessage(targetUser, reason, count)

	// 如果达到最大警告次数，自动踢出
	if count >= MaxWarnings {
		if err := h.telegramAPI.BanChatMember(ctx.GroupID, targetUserID); err != nil {
			return fmt.Errorf("failed to kick user: %v", err)
		}
		message += fmt.Sprintf("\n⚠️ 用户已达到 %d 次警告上限，已被踢出群组", MaxWarnings)
	}

	// 发送响应
	if ctx.ReplyToMessage != nil {
		return h.telegramAPI.SendMessageWithReply(ctx.GroupID, message, ctx.ReplyToMessage.MessageID)
	}
	return h.telegramAPI.SendMessage(ctx.GroupID, message)
}

// handleWarnings 处理查看警告记录命令
func (h *Handler) handleWarnings(ctx *command.Context) error {
	var targetUserID int64
	var err error

	// 解析参数
	args := ctx.Args
	if len(args) > 0 && (args[0] == "warnings" || args[0] == "list") {
		args = args[1:]
	}

	// 情况1：回复消息查看
	if ctx.ReplyToMessage != nil {
		targetUserID = ctx.ReplyToMessage.UserID
	} else {
		// 情况2：指定用户 ID
		if len(args) == 0 {
			return ErrMissingUserID
		}

		targetUserID, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid user ID: %v", err)
		}
	}

	// 验证用户 ID
	if err := validator.UserID(targetUserID); err != nil {
		return err
	}

	// 验证群组 ID
	if err := validator.GroupID(ctx.GroupID); err != nil {
		return err
	}

	// 获取用户信息
	targetUser, err := h.userRepo.FindByID(targetUserID, ctx.GroupID)
	if err != nil {
		return fmt.Errorf("failed to find user: %v", err)
	}

	// 获取警告记录
	warnings, err := h.warningRepo.FindByUserAndGroup(targetUserID, ctx.GroupID)
	if err != nil {
		return fmt.Errorf("failed to get warnings: %v", err)
	}

	// 构建响应消息
	message := h.formatWarningsMessage(targetUser, warnings)

	// 发送响应
	if ctx.ReplyToMessage != nil {
		return h.telegramAPI.SendMessageWithReply(ctx.GroupID, message, ctx.ReplyToMessage.MessageID)
	}
	return h.telegramAPI.SendMessage(ctx.GroupID, message)
}

// handleClearWarn 处理清除警告命令
func (h *Handler) handleClearWarn(ctx *command.Context) error {
	var targetUserID int64
	var err error

	// 解析参数
	args := ctx.Args
	if len(args) > 0 && (args[0] == "clear" || args[0] == "clearwarn") {
		args = args[1:]
	}

	// 情况1：回复消息清除
	if ctx.ReplyToMessage != nil {
		targetUserID = ctx.ReplyToMessage.UserID
	} else {
		// 情况2：指定用户 ID
		if len(args) == 0 {
			return ErrMissingUserID
		}

		targetUserID, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid user ID: %v", err)
		}
	}

	// 验证用户 ID
	if err := validator.UserID(targetUserID); err != nil {
		return err
	}

	// 验证群组 ID
	if err := validator.GroupID(ctx.GroupID); err != nil {
		return err
	}

	// 获取用户信息
	targetUser, err := h.userRepo.FindByID(targetUserID, ctx.GroupID)
	if err != nil {
		return fmt.Errorf("failed to find user: %v", err)
	}

	// 获取当前警告数量
	count, err := h.warningRepo.CountActiveWarnings(targetUserID, ctx.GroupID)
	if err != nil {
		return fmt.Errorf("failed to count warnings: %v", err)
	}

	if count == 0 {
		message := fmt.Sprintf("ℹ️ 用户 %s (ID: %d) 当前没有警告记录", targetUser.Username, targetUser.ID)
		if ctx.ReplyToMessage != nil {
			return h.telegramAPI.SendMessageWithReply(ctx.GroupID, message, ctx.ReplyToMessage.MessageID)
		}
		return h.telegramAPI.SendMessage(ctx.GroupID, message)
	}

	// 清除警告
	if err := h.warningRepo.ClearWarnings(targetUserID, ctx.GroupID); err != nil {
		return fmt.Errorf("failed to clear warnings: %v", err)
	}

	// 构建响应消息
	message := fmt.Sprintf("✅ 已清除用户 %s (ID: %d) 的 %d 条警告记录", targetUser.Username, targetUser.ID, count)

	// 发送响应
	if ctx.ReplyToMessage != nil {
		return h.telegramAPI.SendMessageWithReply(ctx.GroupID, message, ctx.ReplyToMessage.MessageID)
	}
	return h.telegramAPI.SendMessage(ctx.GroupID, message)
}

// formatWarnMessage 格式化警告消息
func (h *Handler) formatWarnMessage(user *user.User, reason string, count int) string {
	message := fmt.Sprintf("⚠️ 警告 %s (ID: %d)\n📝 原因: %s\n🔢 当前警告数: %d/%d",
		user.Username, user.ID, reason, count, MaxWarnings)

	if count >= MaxWarnings {
		message += "\n❗ 已达到警告上限"
	}

	return message
}

// formatWarningsMessage 格式化警告记录消息
func (h *Handler) formatWarningsMessage(user *user.User, warnings []*user.Warning) string {
	if len(warnings) == 0 {
		return fmt.Sprintf("ℹ️ 用户 %s (ID: %d) 没有警告记录", user.Username, user.ID)
	}

	// 统计有效警告
	activeCount := 0
	for _, w := range warnings {
		if !w.IsCleared {
			activeCount++
		}
	}

	message := fmt.Sprintf("📋 用户 %s (ID: %d) 的警告记录\n", user.Username, user.ID)
	message += fmt.Sprintf("🔢 有效警告: %d/%d\n", activeCount, MaxWarnings)
	message += fmt.Sprintf("📊 总计: %d 条\n\n", len(warnings))

	// 只显示有效警告
	count := 0
	for _, w := range warnings {
		if !w.IsCleared {
			count++
			message += fmt.Sprintf("%d. 📝 %s\n", count, w.Reason)
			message += fmt.Sprintf("   ⏰ %s\n", formatTime(w.IssuedAt))
		}
	}

	if activeCount == 0 {
		message += "✅ 所有警告已清除"
	}

	return message
}

// formatTime 格式化时间
func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// showHelp 显示帮助信息
func (h *Handler) showHelp(ctx *command.Context) error {
	help := "⚠️ **警告系统使用说明**\n\n" +
		"**警告用户:**\n" +
		"• `/warn <用户ID> <原因>` - 警告用户\n" +
		"• 回复消息 + `/warn <原因>` - 警告该用户\n\n" +
		"**查看警告记录:**\n" +
		"• `/warn warnings <用户ID>` - 查看用户警告\n" +
		"• `/warn list <用户ID>` - 查看用户警告\n" +
		"• 回复消息 + `/warn warnings` - 查看该用户警告\n\n" +
		"**清除警告:**\n" +
		"• `/warn clear <用户ID>` - 清除用户警告\n" +
		"• `/warn clearwarn <用户ID>` - 清除用户警告\n" +
		"• 回复消息 + `/warn clear` - 清除该用户警告\n\n" +
		fmt.Sprintf("**规则:**\n") +
		fmt.Sprintf("• 警告上限: %d 次\n", MaxWarnings) +
		"• 达到上限将自动踢出群组\n" +
		"• 管理员可清除警告记录\n\n" +
		"**示例:**\n" +
		"• `/warn 123456789 发送广告` - 警告用户\n" +
		"• `/warn warnings 123456789` - 查看警告\n" +
		"• `/warn clear 123456789` - 清除警告"

	return h.telegramAPI.SendMessage(ctx.GroupID, help)
}
