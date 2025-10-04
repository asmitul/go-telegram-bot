package telegram

import (
	"context"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// API Telegram API 适配器
// 提供常用的 Telegram Bot API 操作
type API struct {
	bot *bot.Bot
}

// NewAPI 创建 Telegram API 适配器
func NewAPI(b *bot.Bot) *API {
	return &API{bot: b}
}

// BanChatMember 永久封禁群组成员
func (a *API) BanChatMember(ctx context.Context, chatID, userID int64) error {
	_, err := a.bot.BanChatMember(ctx, &bot.BanChatMemberParams{
		ChatID: chatID,
		UserID: userID,
	})
	return err
}

// BanChatMemberWithDuration 临时封禁群组成员
func (a *API) BanChatMemberWithDuration(ctx context.Context, chatID, userID int64, until time.Time) error {
	_, err := a.bot.BanChatMember(ctx, &bot.BanChatMemberParams{
		ChatID:    chatID,
		UserID:    userID,
		UntilDate: int(until.Unix()),
	})
	return err
}

// RestrictChatMember 限制群组成员权限（禁言等）
func (a *API) RestrictChatMember(ctx context.Context, chatID, userID int64, permissions models.ChatPermissions) error {
	_, err := a.bot.RestrictChatMember(ctx, &bot.RestrictChatMemberParams{
		ChatID:      chatID,
		UserID:      userID,
		Permissions: &permissions,
	})
	return err
}

// RestrictChatMemberWithDuration 限制群组成员权限（禁言等）带时长
func (a *API) RestrictChatMemberWithDuration(ctx context.Context, chatID, userID int64, permissions models.ChatPermissions, until time.Time) error {
	_, err := a.bot.RestrictChatMember(ctx, &bot.RestrictChatMemberParams{
		ChatID:      chatID,
		UserID:      userID,
		Permissions: &permissions,
		UntilDate:   int(until.Unix()),
	})
	return err
}

// SendMessage 发送消息
func (a *API) SendMessage(ctx context.Context, chatID int64, text string) error {
	_, err := a.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	return err
}

// SendMessageWithReply 发送回复消息
func (a *API) SendMessageWithReply(ctx context.Context, chatID int64, text string, replyToMessageID int) error {
	_, err := a.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
		ReplyParameters: &models.ReplyParameters{
			MessageID: replyToMessageID,
		},
	})
	return err
}

// DeleteMessage 删除消息
func (a *API) DeleteMessage(ctx context.Context, chatID int64, messageID int) error {
	_, err := a.bot.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})
	return err
}

// GetChatMember 获取群组成员信息
func (a *API) GetChatMember(ctx context.Context, chatID, userID int64) (*models.ChatMember, error) {
	member, err := a.bot.GetChatMember(ctx, &bot.GetChatMemberParams{
		ChatID: chatID,
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	return member, nil
}
