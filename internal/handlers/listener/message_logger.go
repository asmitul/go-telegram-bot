package listener

import (
	"telegram-bot/internal/handler"
	"telegram-bot/internal/middleware"
)

// MessageLoggerHandler 消息日志处理器
// 记录所有接收到的消息（用于审计和调试）
type MessageLoggerHandler struct {
	logger middleware.Logger
}

// NewMessageLoggerHandler 创建消息日志处理器
func NewMessageLoggerHandler(logger middleware.Logger) *MessageLoggerHandler {
	return &MessageLoggerHandler{
		logger: logger,
	}
}

// Match 匹配所有消息
func (h *MessageLoggerHandler) Match(ctx *handler.Context) bool {
	return true
}

// Handle 处理消息
func (h *MessageLoggerHandler) Handle(ctx *handler.Context) error {
	// 记录消息信息
	h.logger.Debug("message_logged",
		"chat_type", ctx.ChatType,
		"chat_id", ctx.ChatID,
		"chat_title", ctx.ChatTitle,
		"user_id", ctx.UserID,
		"username", ctx.Username,
		"first_name", ctx.FirstName,
		"text", ctx.Text,
		"message_id", ctx.MessageID,
	)

	return nil
}

// Priority 最低优先级
func (h *MessageLoggerHandler) Priority() int {
	return 900
}

// ContinueChain 总是继续
func (h *MessageLoggerHandler) ContinueChain() bool {
	return true
}
