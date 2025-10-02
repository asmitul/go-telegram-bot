package pattern

import (
	"fmt"
	"regexp"
	"telegram-bot/internal/handler"
)

// WeatherHandler 天气查询处理器
// 匹配 "天气 城市名" 格式的消息
type WeatherHandler struct {
	pattern   *regexp.Regexp
	chatTypes []string
}

// NewWeatherHandler 创建天气查询处理器
func NewWeatherHandler() *WeatherHandler {
	return &WeatherHandler{
		pattern:   regexp.MustCompile(`(?i)天气\s+(.+)`),
		chatTypes: []string{"private", "group", "supergroup"},
	}
}

// Match 判断是否匹配
func (h *WeatherHandler) Match(ctx *handler.Context) bool {
	// 检查聊天类型
	if !h.isSupportedChatType(ctx.ChatType) {
		return false
	}

	return h.pattern.MatchString(ctx.Text)
}

// Handle 处理消息
func (h *WeatherHandler) Handle(ctx *handler.Context) error {
	matches := h.pattern.FindStringSubmatch(ctx.Text)
	if len(matches) < 2 {
		return nil
	}

	city := matches[1]

	// TODO: 实际项目中应该调用天气 API
	// 这里只是示例
	response := fmt.Sprintf(
		"📍 城市: %s\n"+
			"🌡️ 温度: 25°C\n"+
			"☁️ 天气: 晴朗\n"+
			"💧 湿度: 60%%\n"+
			"💨 风速: 3m/s\n\n"+
			"（这是示例数据，实际项目请接入天气API）",
		city,
	)

	return ctx.Reply(response)
}

// Priority 优先级
func (h *WeatherHandler) Priority() int {
	return 300 // 正则处理器优先级为 300
}

// ContinueChain 停止执行后续处理器
func (h *WeatherHandler) ContinueChain() bool {
	return false
}

func (h *WeatherHandler) isSupportedChatType(chatType string) bool {
	for _, t := range h.chatTypes {
		if t == chatType {
			return true
		}
	}
	return false
}
