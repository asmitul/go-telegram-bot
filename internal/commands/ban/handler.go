package ban

import (
	"errors"
	"fmt"
	"strconv"
	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
)

var (
	ErrInvalidArguments = errors.New("invalid arguments")
	ErrCannotBanAdmin   = errors.New("cannot ban admin")
)

// TelegramAPI Telegram API 接口（依赖注入）
type TelegramAPI interface {
	BanChatMember(chatID, userID int64) error
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
	return "封禁用户 - 用法: /ban <user_id> 或回复消息"
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
	// 解析目标用户 ID
	targetUserID, err := h.parseTargetUser(ctx)
	if err != nil {
		return h.sendError(ctx, "❌ 用法错误！请使用: /ban <user_id> 或回复要封禁的消息")
	}

	// 检查目标用户是否为管理员
	targetUser, err := h.userRepo.FindByID(targetUserID)
	if err == nil && targetUser.IsAdmin(ctx.GroupID) {
		return h.sendError(ctx, "❌ 无法封禁管理员！")
	}

	// 执行封禁
	if err := h.telegramAPI.BanChatMember(ctx.GroupID, targetUserID); err != nil {
		return h.sendError(ctx, fmt.Sprintf("❌ 封禁失败: %v", err))
	}

	// 发送成功消息
	message := fmt.Sprintf("✅ 用户 %d 已被封禁", targetUserID)
	return h.telegramAPI.SendMessage(ctx.GroupID, message)
}

// parseTargetUser 解析目标用户 ID
func (h *Handler) parseTargetUser(ctx *command.Context) (int64, error) {
	// 如果有参数，解析参数
	if len(ctx.Args) > 0 {
		userID, err := strconv.ParseInt(ctx.Args[0], 10, 64)
		if err != nil {
			return 0, ErrInvalidArguments
		}
		return userID, nil
	}

	// TODO: 如果是回复消息，从回复中获取用户 ID
	// 这需要在 Context 中添加 ReplyToMessage 字段

	return 0, ErrInvalidArguments
}

// sendError 发送错误消息
func (h *Handler) sendError(ctx *command.Context, text string) error {
	return h.telegramAPI.SendMessage(ctx.GroupID, text)
}
