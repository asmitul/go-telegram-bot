package handler

import (
	"context"
	"fmt"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Context 消息处理上下文
// 包含处理消息所需的所有信息
type Context struct {
	// 原始对象
	Ctx     context.Context
	Bot     *bot.Bot
	Update  *models.Update
	Message *models.Message

	// 聊天信息
	ChatType  string // "private", "group", "supergroup", "channel"
	ChatID    int64
	ChatTitle string

	// 用户信息
	UserID    int64
	Username  string
	FirstName string
	LastName  string
	User      *user.User // 数据库用户对象（由中间件注入）

	// 群组信息
	Group *group.Group // 数据库群组对象（由中间件注入）

	// 消息内容
	Text      string
	MessageID int

	// 回复消息
	ReplyTo *ReplyInfo

	// 上下文存储（用于处理器之间传递数据）
	values map[string]interface{}
}

// ReplyInfo 回复消息信息
type ReplyInfo struct {
	MessageID int
	UserID    int64
	Username  string
	Text      string
}

// IsPrivate 是否私聊
func (c *Context) IsPrivate() bool {
	return c.ChatType == "private"
}

// IsGroup 是否群组
func (c *Context) IsGroup() bool {
	return c.ChatType == "group" || c.ChatType == "supergroup"
}

// IsChannel 是否频道
func (c *Context) IsChannel() bool {
	return c.ChatType == "channel"
}

// Set 在上下文中存储值
func (c *Context) Set(key string, value interface{}) {
	if c.values == nil {
		c.values = make(map[string]interface{})
	}
	c.values[key] = value
}

// Get 从上下文中获取值
func (c *Context) Get(key string) (interface{}, bool) {
	if c.values == nil {
		return nil, false
	}
	val, ok := c.values[key]
	return val, ok
}

// Reply 回复消息（纯文本）
func (c *Context) Reply(text string) error {
	_, err := c.Bot.SendMessage(c.Ctx, &bot.SendMessageParams{
		ChatID: c.ChatID,
		Text:   text,
		ReplyParameters: &models.ReplyParameters{
			MessageID: c.MessageID,
		},
	})
	return err
}

// ReplyMarkdown 回复消息（Markdown 格式）
func (c *Context) ReplyMarkdown(text string) error {
	_, err := c.Bot.SendMessage(c.Ctx, &bot.SendMessageParams{
		ChatID:    c.ChatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
		ReplyParameters: &models.ReplyParameters{
			MessageID: c.MessageID,
		},
	})
	return err
}

// ReplyHTML 回复消息（HTML 格式）
func (c *Context) ReplyHTML(text string) error {
	_, err := c.Bot.SendMessage(c.Ctx, &bot.SendMessageParams{
		ChatID:    c.ChatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
		ReplyParameters: &models.ReplyParameters{
			MessageID: c.MessageID,
		},
	})
	return err
}

// Send 发送消息（不回复）
func (c *Context) Send(text string) error {
	_, err := c.Bot.SendMessage(c.Ctx, &bot.SendMessageParams{
		ChatID: c.ChatID,
		Text:   text,
	})
	return err
}

// SendMarkdown 发送消息（Markdown 格式，不回复）
func (c *Context) SendMarkdown(text string) error {
	_, err := c.Bot.SendMessage(c.Ctx, &bot.SendMessageParams{
		ChatID:    c.ChatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
	return err
}

// SendHTML 发送消息（HTML 格式，不回复）
func (c *Context) SendHTML(text string) error {
	_, err := c.Bot.SendMessage(c.Ctx, &bot.SendMessageParams{
		ChatID:    c.ChatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
	return err
}

// DeleteMessage 删除消息
func (c *Context) DeleteMessage() error {
	_, err := c.Bot.DeleteMessage(c.Ctx, &bot.DeleteMessageParams{
		ChatID:    c.ChatID,
		MessageID: c.MessageID,
	})
	return err
}

// HasPermission 检查用户是否有指定权限
func (c *Context) HasPermission(required user.Permission) bool {
	if c.User == nil {
		return false
	}

	// 私聊使用全局权限（groupID = 0），群组使用群组 ID
	groupID := c.ChatID
	if c.IsPrivate() {
		groupID = 0 // 全局权限
	}

	return c.User.HasPermission(groupID, required)
}

// RequirePermission 要求特定权限，如果不满足返回错误
func (c *Context) RequirePermission(required user.Permission) error {
	if !c.HasPermission(required) {
		currentPerm := user.PermissionUser
		if c.User != nil {
			groupID := c.ChatID
			if c.IsPrivate() {
				groupID = 0 // 全局权限
			}
			currentPerm = c.User.GetPermission(groupID)
		}

		return fmt.Errorf("❌ 权限不足！需要权限: %s，当前权限: %s",
			required.String(), currentPerm.String())
	}
	return nil
}
