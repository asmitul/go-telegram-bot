package ban

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
)

var (
	ErrInvalidArguments = errors.New("invalid arguments")
	ErrCannotBanAdmin   = errors.New("cannot ban admin")
	ErrInvalidDuration  = errors.New("invalid duration")
)

// TelegramAPI Telegram API 接口（依赖注入）
type TelegramAPI interface {
	BanChatMember(chatID, userID int64) error
	BanChatMemberWithDuration(chatID, userID int64, until time.Time) error
	SendMessage(chatID int64, text string) error
}

// Handler Ban 命令处理器
type Handler struct {
	groupRepo   group.Repository
	userRepo    user.Repository
	telegramAPI TelegramAPI
}

// NewHandler 创建 Ban 命令处理器
func NewHandler(groupRepo group.Repository, userRepo user.Repository, api TelegramAPI) *Handler {
	return &Handler{
		groupRepo:   groupRepo,
		userRepo:    userRepo,
		telegramAPI: api,
	}
}

// Name 命令名称
func (h *Handler) Name() string {
	return "ban"
}

// Description 命令描述
func (h *Handler) Description() string {
	return "封禁用户"
}

// RequiredPermission 所需权限
func (h *Handler) RequiredPermission() user.Permission {
	return user.PermissionAdmin // 需要管理员权限
}

// IsEnabled 检查命令是否在群组中启用
func (h *Handler) IsEnabled(groupID int64) bool {
	g, err := h.groupRepo.FindByID(groupID)
	if err != nil {
		return true
	}
	return g.IsCommandEnabled(h.Name())
}

// Handle 处理命令
func (h *Handler) Handle(ctx *command.Context) error {
	// 解析目标用户 ID、时长和原因
	targetUserID, duration, reason, err := h.parseArguments(ctx)
	if err != nil {
		return h.sendError(ctx, h.getUsageMessage())
	}

	// 检查目标用户是否为管理员
	targetUser, err := h.userRepo.FindByID(targetUserID)
	if err == nil && targetUser.IsAdmin(ctx.GroupID) {
		return h.sendError(ctx, "❌ 无法封禁管理员！")
	}

	// 执行封禁
	var banErr error
	if duration > 0 {
		until := time.Now().Add(duration)
		banErr = h.telegramAPI.BanChatMemberWithDuration(ctx.GroupID, targetUserID, until)
	} else {
		banErr = h.telegramAPI.BanChatMember(ctx.GroupID, targetUserID)
	}

	if banErr != nil {
		return h.sendError(ctx, fmt.Sprintf("❌ 封禁失败: %v", banErr))
	}

	// 构建成功消息
	message := h.buildSuccessMessage(targetUserID, duration, reason)
	return h.telegramAPI.SendMessage(ctx.GroupID, message)
}

// parseArguments 解析命令参数
func (h *Handler) parseArguments(ctx *command.Context) (userID int64, duration time.Duration, reason string, err error) {
	// 优先从回复消息获取用户 ID
	if ctx.ReplyToMessage != nil {
		userID = ctx.ReplyToMessage.UserID
		// 解析时长和原因
		duration, reason = h.parseDurationAndReason(ctx.Args)
		return userID, duration, reason, nil
	}

	// 从参数获取用户 ID
	if len(ctx.Args) == 0 {
		return 0, 0, "", ErrInvalidArguments
	}

	userID, err = strconv.ParseInt(ctx.Args[0], 10, 64)
	if err != nil {
		return 0, 0, "", ErrInvalidArguments
	}

	// 解析时长和原因
	duration, reason = h.parseDurationAndReason(ctx.Args[1:])
	return userID, duration, reason, nil
}

// parseDurationAndReason 解析时长和原因
func (h *Handler) parseDurationAndReason(args []string) (duration time.Duration, reason string) {
	if len(args) == 0 {
		return 0, ""
	}

	// 尝试解析第一个参数为时长
	if d, err := parseDuration(args[0]); err == nil {
		duration = d
		// 剩余参数作为原因
		if len(args) > 1 {
			reason = strings.Join(args[1:], " ")
		}
	} else {
		// 所有参数都作为原因
		reason = strings.Join(args, " ")
	}

	return duration, reason
}

// parseDuration 解析时长字符串
// 支持格式: 1h, 30m, 1d, 2h30m
func parseDuration(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, ErrInvalidDuration
	}

	// 尝试使用 time.ParseDuration
	d, err := time.ParseDuration(s)
	if err == nil {
		return d, nil
	}

	// 支持天数 (d)
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, ErrInvalidDuration
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	return 0, ErrInvalidDuration
}

// buildSuccessMessage 构建成功消息
func (h *Handler) buildSuccessMessage(userID int64, duration time.Duration, reason string) string {
	var message string

	if duration > 0 {
		message = fmt.Sprintf("✅ 用户 `%d` 已被临时封禁 %s", userID, formatDuration(duration))
	} else {
		message = fmt.Sprintf("✅ 用户 `%d` 已被永久封禁", userID)
	}

	if reason != "" {
		message += fmt.Sprintf("\n📝 原因: %s", reason)
	}

	return message
}

// formatDuration 格式化时长
func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d 分钟", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%d 小时 %d 分钟", hours, minutes)
		}
		return fmt.Sprintf("%d 小时", hours)
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	if hours > 0 {
		return fmt.Sprintf("%d 天 %d 小时", days, hours)
	}
	return fmt.Sprintf("%d 天", days)
}

// getUsageMessage 获取使用说明
func (h *Handler) getUsageMessage() string {
	return `❌ 用法错误！

*使用方法*:
• ` + "`/ban <用户ID>`" + ` - 永久封禁用户
• ` + "`/ban <用户ID> <时长>`" + ` - 临时封禁
• ` + "`/ban <用户ID> <原因>`" + ` - 永久封禁并说明原因
• ` + "`/ban <用户ID> <时长> <原因>`" + ` - 临时封禁并说明原因
• 回复消息 + ` + "`/ban`" + ` - 封禁被回复的用户
• 回复消息 + ` + "`/ban <时长>`" + ` - 临时封禁被回复的用户
• 回复消息 + ` + "`/ban <原因>`" + ` - 封禁被回复的用户并说明原因

*时长格式*:
• 30m - 30分钟
• 1h - 1小时
• 2h30m - 2小时30分钟
• 1d - 1天
• 7d - 7天

*示例*:
• ` + "`/ban 123456`" + ` - 永久封禁用户123456
• ` + "`/ban 123456 1h`" + ` - 封禁用户123456 1小时
• ` + "`/ban 123456 spam`" + ` - 永久封禁用户123456，原因：spam
• ` + "`/ban 123456 1d spam`" + ` - 封禁用户123456 1天，原因：spam`
}

// sendError 发送错误消息
func (h *Handler) sendError(ctx *command.Context, text string) error {
	return h.telegramAPI.SendMessage(ctx.GroupID, text)
}
