package keyword

import (
	"fmt"
	"strings"
	"telegram-bot/internal/handler"
)

// GreetingHandler 问候处理器
// 当用户发送问候语时自动回复
type GreetingHandler struct {
	keywords  []string
	chatTypes []string // 支持的聊天类型
}

// NewGreetingHandler 创建问候处理器
func NewGreetingHandler() *GreetingHandler {
	return &GreetingHandler{
		keywords: []string{
			"你好", "您好", "hello", "hi", "嗨",
			"早上好", "晚上好", "下午好",
		},
		chatTypes: []string{"private"}, // 只在私聊中响应
	}
}

// Match 判断是否匹配
func (h *GreetingHandler) Match(ctx *handler.Context) bool {
	// 检查聊天类型
	if !h.isSupportedChatType(ctx.ChatType) {
		return false
	}

	// 检查是否包含关键词
	text := strings.ToLower(strings.TrimSpace(ctx.Text))
	for _, keyword := range h.keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// Handle 处理消息
func (h *GreetingHandler) Handle(ctx *handler.Context) error {
	name := ctx.FirstName
	if name == "" {
		name = "朋友"
	}

	response := fmt.Sprintf("你好，%s！👋\n\n有什么可以帮你的吗？\n输入 /help 查看可用命令。", name)
	return ctx.Reply(response)
}

// Priority 优先级
func (h *GreetingHandler) Priority() int {
	return 200 // 关键词处理器优先级为 200
}

// ContinueChain 继续执行后续处理器
func (h *GreetingHandler) ContinueChain() bool {
	return true
}

func (h *GreetingHandler) isSupportedChatType(chatType string) bool {
	for _, t := range h.chatTypes {
		if t == chatType {
			return true
		}
	}
	return false
}
