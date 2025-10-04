package telegram

import (
	"context"
	"telegram-bot/internal/handler"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// ConvertUpdate 将 Telegram Update 转换为 Handler Context
// 如果不是消息更新，返回 nil
func ConvertUpdate(ctx context.Context, b *bot.Bot, update *models.Update) *handler.Context {
	// 只处理消息更新
	if update.Message == nil {
		return nil
	}

	msg := update.Message

	// 某些消息（如频道消息）可能没有 From 字段，跳过处理
	if msg.From == nil {
		return nil
	}

	// 构建 handler.Context
	handlerCtx := &handler.Context{
		Ctx:     ctx,
		Bot:     b,
		Update:  update,
		Message: msg,

		// 聊天信息
		ChatType:  string(msg.Chat.Type),
		ChatID:    msg.Chat.ID,
		ChatTitle: msg.Chat.Title,

		// 用户信息
		UserID:    msg.From.ID,
		Username:  msg.From.Username,
		FirstName: msg.From.FirstName,
		LastName:  msg.From.LastName,

		// 消息内容
		Text:      msg.Text,
		MessageID: msg.ID,
	}

	// 处理回复消息
	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil {
		handlerCtx.ReplyTo = &handler.ReplyInfo{
			MessageID: msg.ReplyToMessage.ID,
			UserID:    msg.ReplyToMessage.From.ID,
			Username:  msg.ReplyToMessage.From.Username,
			Text:      msg.ReplyToMessage.Text,
		}
	}

	return handlerCtx
}
