//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/handler"
	"telegram-bot/internal/handlers/command"
	"telegram-bot/internal/handlers/keyword"
	"telegram-bot/internal/handlers/listener"

	"github.com/stretchr/testify/assert"
)

// TestHandlerRouting 测试处理器路由
func TestHandlerRouting(t *testing.T) {
	// 创建路由器
	router := handler.NewRouter()

	// 创建 mock group repository
	mockGroupRepo := &MockGroupRepo{}

	// 注册处理器
	router.Register(command.NewPingHandler(mockGroupRepo))
	router.Register(keyword.NewGreetingHandler())

	// 验证处理器数量
	assert.Equal(t, 2, router.Count())

	// 获取所有处理器
	handlers := router.GetHandlers()
	assert.Len(t, handlers, 2)

	// 验证优先级排序
	assert.Equal(t, 100, handlers[0].Priority()) // 命令
	assert.Equal(t, 200, handlers[1].Priority()) // 关键词
}

// MockGroupRepo 模拟群组仓储
type MockGroupRepo struct{}

func (m *MockGroupRepo) FindByID(id int64) (*group.Group, error) {
	return group.NewGroup(id, "Test Group", "group"), nil
}

// TestCommandMatching 测试命令匹配
func TestCommandMatching(t *testing.T) {
	mockGroupRepo := &MockGroupRepo{}
	pingHandler := command.NewPingHandler(mockGroupRepo)

	testCases := []struct {
		name     string
		text     string
		chatType string
		expected bool
	}{
		{"匹配 ping 命令", "/ping", "group", true},
		{"匹配 ping 带参数", "/ping arg1", "group", true},
		{"匹配 ping 带 @botname", "/ping@testbot", "group", true},
		{"不匹配其他命令", "/help", "group", false},
		{"不匹配普通文本", "hello", "group", false},
		{"不匹配频道", "/ping", "channel", false},
		{"匹配私聊", "/ping", "private", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &handler.Context{
				Text:     tc.text,
				ChatType: tc.chatType,
				ChatID:   123,
			}

			result := pingHandler.Match(ctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestKeywordMatching 测试关键词匹配
func TestKeywordMatching(t *testing.T) {
	greetingHandler := keyword.NewGreetingHandler()

	testCases := []struct {
		name     string
		text     string
		chatType string
		expected bool
	}{
		{"匹配你好", "你好", "private", true},
		{"匹配 hello", "hello", "private", true},
		{"匹配 Hi", "Hi there!", "private", true},
		{"不匹配群组", "你好", "group", false},
		{"不匹配其他文本", "random text", "private", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &handler.Context{
				Text:     tc.text,
				ChatType: tc.chatType,
			}

			result := greetingHandler.Match(ctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestMiddlewareChain 测试中间件链
func TestMiddlewareChain(t *testing.T) {
	var executed []string

	middleware1 := func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			executed = append(executed, "mw1_before")
			err := next(ctx)
			executed = append(executed, "mw1_after")
			return err
		}
	}

	middleware2 := func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			executed = append(executed, "mw2_before")
			err := next(ctx)
			executed = append(executed, "mw2_after")
			return err
		}
	}

	finalHandler := func(ctx *handler.Context) error {
		executed = append(executed, "handler")
		return nil
	}

	// 组合中间件
	chain := handler.Chain(middleware1, middleware2)(finalHandler)

	// 执行
	ctx := &handler.Context{Ctx: context.Background()}
	err := chain(ctx)

	// 验证
	assert.NoError(t, err)
	assert.Equal(t, []string{
		"mw1_before",
		"mw2_before",
		"handler",
		"mw2_after",
		"mw1_after",
	}, executed)
}
