package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"
	"telegram-bot/internal/domain/command"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// HandleUpdate 处理 Telegram 更新的全局函数
func HandleUpdate(
	ctx context.Context,
	b *bot.Bot,
	update *models.Update,
	registry command.Registry,
	permMiddleware *PermissionMiddleware,
	logMiddleware *LoggingMiddleware,
) {
	// 只处理消息
	if update.Message == nil {
		return
	}

	msg := update.Message

	// 只处理群组消息
	if msg.Chat.Type != "group" && msg.Chat.Type != "supergroup" {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "此 Bot 仅在群组中工作",
		})
		return
	}

	// 只处理命令（以 / 开头）
	if len(msg.Text) == 0 || msg.Text[0] != '/' {
		return
	}

	// 解析命令
	commandName, args := parseCommand(msg.Text)

	// 获取命令处理器
	handler, exists := registry.Get(commandName)
	if !exists {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   fmt.Sprintf("❌ 未知命令: /%s", commandName),
		})
		return
	}

	// 创建命令上下文
	cmdCtx := &command.Context{
		Ctx:       ctx,
		UserID:    msg.From.ID,
		GroupID:   msg.Chat.ID,
		MessageID: msg.ID,
		Text:      msg.Text,
		Args:      args,
	}

	// 构建中间件链
	middlewares := []Middleware{
		logMiddleware.Log(),
		permMiddleware.Check(handler),
	}

	// 执行命令
	finalHandler := func(ctx *command.Context) error {
		return handler.Handle(ctx)
	}

	chainedHandler := Chain(middlewares...)(finalHandler)

	// 执行
	if err := chainedHandler(cmdCtx); err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   fmt.Sprintf("❌ 错误: %v", err),
		})
	}
}

// parseCommand 解析命令和参数
func parseCommand(text string) (string, []string) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return "", nil
	}

	// 移除开头的 /
	commandName := strings.TrimPrefix(parts[0], "/")

	// 如果命令包含 @botname，移除它
	if idx := strings.Index(commandName, "@"); idx != -1 {
		commandName = commandName[:idx]
	}

	// 返回命令名和参数
	if len(parts) > 1 {
		return commandName, parts[1:]
	}
	return commandName, nil
}

// parseArgs 解析命令参数（支持引号）
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
	bot *bot.Bot
}

// NewAPI 创建 Telegram API 适配器
func NewAPI(b *bot.Bot) *API {
	return &API{bot: b}
}

// BanChatMember 封禁群组成员
func (a *API) BanChatMember(chatID, userID int64) error {
	_, err := a.bot.BanChatMember(context.Background(), &bot.BanChatMemberParams{
		ChatID: chatID,
		UserID: userID,
	})
	return err
}

// BanChatMemberWithDuration 临时封禁群组成员
func (a *API) BanChatMemberWithDuration(chatID, userID int64, until time.Time) error {
	_, err := a.bot.BanChatMember(context.Background(), &bot.BanChatMemberParams{
		ChatID:         chatID,
		UserID:         userID,
		UntilDate:      int(until.Unix()),
	})
	return err
}

// SendMessage 发送消息
func (a *API) SendMessage(chatID int64, text string) error {
	_, err := a.bot.SendMessage(context.Background(), &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	return err
}

// SendMessageWithReply 发送回复消息
func (a *API) SendMessageWithReply(chatID int64, text string, replyToMessageID int) error {
	_, err := a.bot.SendMessage(context.Background(), &bot.SendMessageParams{
		ChatID:           chatID,
		Text:             text,
		ReplyParameters: &models.ReplyParameters{
			MessageID: replyToMessageID,
		},
	})
	return err
}

// DeleteMessage 删除消息
func (a *API) DeleteMessage(chatID int64, messageID int) error {
	_, err := a.bot.DeleteMessage(context.Background(), &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})
	return err
}

// GetChatMember 获取群组成员信息
func (a *API) GetChatMember(chatID, userID int64) (*models.ChatMember, error) {
	member, err := a.bot.GetChatMember(context.Background(), &bot.GetChatMemberParams{
		ChatID: chatID,
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	return member, nil
}

// UnbanChatMember 解封群组成员
func (a *API) UnbanChatMember(chatID, userID int64) error {
	_, err := a.bot.UnbanChatMember(context.Background(), &bot.UnbanChatMemberParams{
		ChatID:       chatID,
		UserID:       userID,
		OnlyIfBanned: true,
	})
	return err
}

// RestrictChatMember 限制群组成员权限（禁言等）
func (a *API) RestrictChatMember(chatID, userID int64, permissions models.ChatPermissions) error {
	_, err := a.bot.RestrictChatMember(context.Background(), &bot.RestrictChatMemberParams{
		ChatID:      chatID,
		UserID:      userID,
		Permissions: &permissions,
	})
	return err
}