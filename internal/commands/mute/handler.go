package mute

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/user"
	"telegram-bot/pkg/validator"
	"time"

	"github.com/go-telegram/bot/models"
)

var (
	ErrInvalidDuration = errors.New("invalid duration format")
	ErrMissingUserID   = errors.New("missing user ID")
)

// TelegramAPI 定义与 Telegram API 交互的接口
type TelegramAPI interface {
	SendMessage(chatID int64, text string) error
	SendMessageWithReply(chatID int64, text string, replyToMessageID int) error
	RestrictChatMember(chatID, userID int64, permissions models.ChatPermissions) error
	RestrictChatMemberWithDuration(chatID, userID int64, permissions models.ChatPermissions, until time.Time) error
}

// UserRepository 用户仓储接口
type UserRepository interface {
	FindByID(userID int64, groupID int64) (*user.User, error)
}

// Handler 处理 mute 相关命令
type Handler struct {
	telegramAPI TelegramAPI
	userRepo    UserRepository
}

// NewHandler 创建新的 Handler
func NewHandler(telegramAPI TelegramAPI, userRepo UserRepository) *Handler {
	return &Handler{
		telegramAPI: telegramAPI,
		userRepo:    userRepo,
	}
}

// Name 返回命令名称
func (h *Handler) Name() string {
	return "mute"
}

// Description 返回命令描述
func (h *Handler) Description() string {
	return "禁言用户"
}

// RequiredPermission 返回所需权限
func (h *Handler) RequiredPermission() user.Permission {
	return user.PermissionAdmin
}

// Handle 处理命令
func (h *Handler) Handle(ctx *command.Context) error {
	// 检查是否是 unmute 命令
	if len(ctx.Args) > 0 && ctx.Args[0] == "unmute" || strings.HasPrefix(ctx.Text, "/unmute") {
		return h.handleUnmute(ctx)
	}

	// 如果没有参数且没有回复消息，显示帮助
	if len(ctx.Args) == 0 && ctx.ReplyToMessage == nil {
		return h.showHelp(ctx)
	}

	// 处理 mute 命令
	return h.handleMute(ctx)
}

// handleMute 处理禁言命令
func (h *Handler) handleMute(ctx *command.Context) error {
	var targetUserID int64
	var duration time.Duration
	var reason string
	var err error

	// 情况1：回复消息禁言
	if ctx.ReplyToMessage != nil {
		targetUserID = ctx.ReplyToMessage.UserID

		// 解析时长和原因
		if len(ctx.Args) > 0 {
			// 尝试解析第一个参数作为时长
			duration, err = parseDuration(ctx.Args[0])
			if err == nil {
				// 第一个参数是时长，后续是原因
				if len(ctx.Args) > 1 {
					reason = strings.Join(ctx.Args[1:], " ")
				}
			} else {
				// 第一个参数不是时长，全部作为原因
				reason = strings.Join(ctx.Args, " ")
			}
		}
	} else {
		// 情况2：指定用户 ID 禁言
		if len(ctx.Args) == 0 {
			return ErrMissingUserID
		}

		// 解析用户 ID
		targetUserID, err = strconv.ParseInt(ctx.Args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid user ID: %v", err)
		}

		// 解析时长和原因
		if len(ctx.Args) > 1 {
			// 尝试解析第二个参数作为时长
			duration, err = parseDuration(ctx.Args[1])
			if err == nil {
				// 第二个参数是时长，后续是原因
				if len(ctx.Args) > 2 {
					reason = strings.Join(ctx.Args[2:], " ")
				}
			} else {
				// 第二个参数不是时长，全部作为原因
				reason = strings.Join(ctx.Args[1:], " ")
			}
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

	// 获取目标用户信息
	targetUser, err := h.userRepo.FindByID(targetUserID, ctx.GroupID)
	if err != nil {
		return fmt.Errorf("failed to find user: %v", err)
	}

	// 构建禁言权限（禁止发送所有类型的消息）
	permissions := models.ChatPermissions{
		CanSendMessages:       false,
		CanSendPhotos:         false,
		CanSendVideos:         false,
		CanSendAudios:         false,
		CanSendDocuments:      false,
		CanSendVoiceNotes:     false,
		CanSendVideoNotes:     false,
		CanSendPolls:          false,
		CanSendOtherMessages:  false,
		CanAddWebPagePreviews: false,
	}

	// 执行禁言
	if duration > 0 {
		until := time.Now().Add(duration)
		if err := h.telegramAPI.RestrictChatMemberWithDuration(ctx.GroupID, targetUserID, permissions, until); err != nil {
			return fmt.Errorf("failed to mute user: %v", err)
		}
	} else {
		if err := h.telegramAPI.RestrictChatMember(ctx.GroupID, targetUserID, permissions); err != nil {
			return fmt.Errorf("failed to mute user: %v", err)
		}
	}

	// 构建响应消息
	message := h.formatMuteMessage(targetUser, duration, reason)

	// 发送响应
	if ctx.ReplyToMessage != nil {
		return h.telegramAPI.SendMessageWithReply(ctx.GroupID, message, ctx.ReplyToMessage.MessageID)
	}
	return h.telegramAPI.SendMessage(ctx.GroupID, message)
}

// handleUnmute 处理解除禁言命令
func (h *Handler) handleUnmute(ctx *command.Context) error {
	var targetUserID int64
	var err error

	// 处理 /unmute 命令
	args := ctx.Args
	if strings.HasPrefix(ctx.Text, "/unmute") {
		// 如果是 /unmute 命令，从 ctx.Args 中获取参数（第一个参数是用户ID）
		if len(args) == 0 && ctx.ReplyToMessage == nil {
			return ErrMissingUserID
		}
	} else if len(args) > 0 && args[0] == "unmute" {
		// 如果是 /mute unmute，跳过第一个参数
		args = args[1:]
	}

	// 情况1：回复消息解除禁言
	if ctx.ReplyToMessage != nil {
		targetUserID = ctx.ReplyToMessage.UserID
	} else {
		// 情况2：指定用户 ID 解除禁言
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

	// 获取目标用户信息
	targetUser, err := h.userRepo.FindByID(targetUserID, ctx.GroupID)
	if err != nil {
		return fmt.Errorf("failed to find user: %v", err)
	}

	// 构建正常权限（恢复发送消息的权限）
	permissions := models.ChatPermissions{
		CanSendMessages:       true,
		CanSendPhotos:         true,
		CanSendVideos:         true,
		CanSendAudios:         true,
		CanSendDocuments:      true,
		CanSendVoiceNotes:     true,
		CanSendVideoNotes:     true,
		CanSendPolls:          true,
		CanSendOtherMessages:  true,
		CanAddWebPagePreviews: true,
	}

	// 执行解除禁言
	if err := h.telegramAPI.RestrictChatMember(ctx.GroupID, targetUserID, permissions); err != nil {
		return fmt.Errorf("failed to unmute user: %v", err)
	}

	// 构建响应消息
	message := fmt.Sprintf("✅ 已解除对用户 %s (ID: %d) 的禁言", targetUser.Username, targetUser.ID)

	// 发送响应
	if ctx.ReplyToMessage != nil {
		return h.telegramAPI.SendMessageWithReply(ctx.GroupID, message, ctx.ReplyToMessage.MessageID)
	}
	return h.telegramAPI.SendMessage(ctx.GroupID, message)
}

// formatMuteMessage 格式化禁言消息
func (h *Handler) formatMuteMessage(user *user.User, duration time.Duration, reason string) string {
	message := fmt.Sprintf("🔇 用户 %s (ID: %d) 已被禁言", user.Username, user.ID)

	if duration > 0 {
		message += fmt.Sprintf("\n⏱ 时长: %s", formatDuration(duration))
	} else {
		message += "\n⏱ 时长: 永久"
	}

	if reason != "" {
		message += fmt.Sprintf("\n📝 原因: %s", reason)
	}

	return message
}

// formatDuration 格式化时长
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f秒", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0f分钟", d.Minutes())
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%d小时%d分钟", hours, minutes)
		}
		return fmt.Sprintf("%d小时", hours)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	if hours > 0 {
		return fmt.Sprintf("%d天%d小时", days, hours)
	}
	return fmt.Sprintf("%d天", days)
}

// parseDuration 解析时长字符串
func parseDuration(s string) (time.Duration, error) {
	// 先尝试标准的 Go 时长格式
	d, err := time.ParseDuration(s)
	if err == nil {
		return d, nil
	}

	// 支持 "天" 格式 (如 "7d")
	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return 0, ErrInvalidDuration
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	return 0, ErrInvalidDuration
}

// showHelp 显示帮助信息
func (h *Handler) showHelp(ctx *command.Context) error {
	help := "🔇 **禁言命令使用说明**\n\n" +
		"**禁言用户:**\n" +
		"• `/mute <用户ID>` - 永久禁言\n" +
		"• `/mute <用户ID> <时长>` - 临时禁言\n" +
		"• `/mute <用户ID> <时长> <原因>` - 带原因的临时禁言\n" +
		"• `/mute <用户ID> <原因>` - 带原因的永久禁言\n" +
		"• 回复消息 + `/mute` - 禁言该用户\n" +
		"• 回复消息 + `/mute <时长>` - 临时禁言该用户\n" +
		"• 回复消息 + `/mute <时长> <原因>` - 带原因的临时禁言\n\n" +
		"**解除禁言:**\n" +
		"• `/unmute <用户ID>` - 解除禁言\n" +
		"• 回复消息 + `/unmute` - 解除该用户的禁言\n\n" +
		"**时长格式:**\n" +
		"• `30m` - 30分钟\n" +
		"• `1h` - 1小时\n" +
		"• `2h30m` - 2小时30分钟\n" +
		"• `1d` - 1天\n" +
		"• `7d` - 7天\n\n" +
		"**示例:**\n" +
		"• `/mute 123456789` - 永久禁言用户\n" +
		"• `/mute 123456789 1h` - 禁言1小时\n" +
		"• `/mute 123456789 30m 刷屏` - 禁言30分钟，原因是刷屏\n" +
		"• `/unmute 123456789` - 解除禁言"

	return h.telegramAPI.SendMessage(ctx.GroupID, help)
}
