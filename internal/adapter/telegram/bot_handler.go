package telegram

import (
	"context"
	"fmt"
	"strings"
	"telegram-bot/internal/domain/command"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// BotHandler Telegram Bot 处理器
type BotHandler struct {
	bot                 *tgbotapi.BotAPI
	registry            command.Registry
	permMiddleware      *PermissionMiddleware
	logMiddleware       *LoggingMiddleware
	rateLimitMiddleware *RateLimitMiddleware
}

// NewBotHandler 创建 Bot 处理器
func NewBotHandler(
	bot *tgbotapi.BotAPI,
	registry command.Registry,
	permMiddleware *PermissionMiddleware,
	logMiddleware *LoggingMiddleware,
) *BotHandler {
	return &BotHandler{
		bot:            bot,
		registry:       registry,
		permMiddleware: permMiddleware,
		logMiddleware:  logMiddleware,
	}
}

// Start 启动 Bot
func (h *BotHandler) Start(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := h.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			h.bot.StopReceivingUpdates()
			return ctx.Err()

		case update := <-updates:
			// 处理更新
			go h.handleUpdate(update)
		}
	}
}

// handleUpdate 处理单个更新
func (h *BotHandler) handleUpdate(update tgbotapi.Update) {
	// 只处理消息
	if update.Message == nil {
		return
	}

	msg := update.Message

	// 只处理群组消息和命令
	if !msg.Chat.IsGroup() && !msg.Chat.IsSuperGroup() {
		h.sendMessage(msg.Chat.ID, "此 Bot 仅在群组中工作")
		return
	}

	// 只处理命令
	if !msg.IsCommand() {
		return
	}

	// 解析命令
	commandName := msg.Command()

	// 获取命令处理器
	handler, exists := h.registry.Get(commandName)
	if !exists {
		h.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ 未知命令: /%s", commandName))
		return
	}

	// 创建命令上下文
	cmdCtx := &command.Context{
		Ctx:       context.Background(),
		UserID:    msg.From.ID,
		GroupID:   msg.Chat.ID,
		MessageID: msg.MessageID,
		Text:      msg.Text,
		Args:      parseArgs(msg.CommandArguments()),
	}

	// 构建中间件链
	middlewares := []Middleware{
		h.logMiddleware.Log(),
		h.permMiddleware.Check(handler),
	}

	// 如果配置了限流
	if h.rateLimitMiddleware != nil {
		middlewares = append([]Middleware{h.rateLimitMiddleware.Limit()}, middlewares...)
	}

	// 执行命令
	finalHandler := func(ctx *command.Context) error {
		return handler.Handle(ctx)
	}

	chainedHandler := Chain(middlewares...)(finalHandler)

	// 执行
	if err := chainedHandler(cmdCtx); err != nil {
		h.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ 错误: %v", err))
	}
}

// sendMessage 发送消息
func (h *BotHandler) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := h.bot.Send(msg)
	return err
}

// parseArgs 解析命令参数
func parseArgs(argsStr string) []string {
	if argsStr == "" {
		return []string{}
	}

	// 支持引号包裹的参数
	var args []string
	var current strings.Builder
	inQuotes := false

	for i, char := range argsStr {
		switch char {
		case '"':
			inQuotes = !inQuotes
		case ' ':
			if !inQuotes {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}

		// 最后一个字符
		if i == len(argsStr)-1 && current.Len() > 0 {
			args = append(args, current.String())
		}
	}

	return args
}

// API Telegram API 适配器
type API struct {
	bot *tgbotapi.BotAPI
}

// NewAPI 创建 Telegram API 适配器
func NewAPI(bot *tgbotapi.BotAPI) *API {
	return &API{bot: bot}
}

// BanChatMember 封禁群组成员
func (a *API) BanChatMember(chatID, userID int64) error {
	config := tgbotapi.BanChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: chatID,
			UserID: userID,
		},
	}
	_, err := a.bot.Request(config)
	return err
}

// SendMessage 发送消息
func (a *API) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := a.bot.Send(msg)
	return err
}

// SendMessageWithReply 发送回复消息
func (a *API) SendMessageWithReply(chatID int64, text string, replyToMessageID int) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = replyToMessageID
	_, err := a.bot.Send(msg)
	return err
}

// DeleteMessage 删除消息
func (a *API) DeleteMessage(chatID int64, messageID int) error {
	config := tgbotapi.DeleteMessageConfig{
		ChatID:    chatID,
		MessageID: messageID,
	}
	_, err := a.bot.Request(config)
	return err
}

// GetChatMember 获取群组成员信息
func (a *API) GetChatMember(chatID, userID int64) (*tgbotapi.ChatMember, error) {
	config := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: chatID,
			UserID: userID,
		},
	}

	member, err := a.bot.GetChatMember(config)
	if err != nil {
		return nil, err
	}

	return &member, nil
}

// UnbanChatMember 解封群组成员
func (a *API) UnbanChatMember(chatID, userID int64) error {
	config := tgbotapi.UnbanChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: chatID,
			UserID: userID,
		},
		OnlyIfBanned: true,
	}
	_, err := a.bot.Request(config)
	return err
}

// RestrictChatMember 限制群组成员权限（禁言等）
func (a *API) RestrictChatMember(chatID, userID int64, permissions tgbotapi.ChatPermissions) error {
	config := tgbotapi.RestrictChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: chatID,
			UserID: userID,
		},
		Permissions: &permissions,
	}
	_, err := a.bot.Request(config)
	return err
}
