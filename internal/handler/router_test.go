package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockHandler 模拟处理器
type MockHandler struct {
	priority      int
	shouldMatch   bool
	continueChain bool
	handleCalled  bool
}

func (m *MockHandler) Match(ctx *Context) bool {
	return m.shouldMatch
}

func (m *MockHandler) Handle(ctx *Context) error {
	m.handleCalled = true
	return nil
}

func (m *MockHandler) Priority() int {
	return m.priority
}

func (m *MockHandler) ContinueChain() bool {
	return m.continueChain
}

// TestRouter_Register 测试注册处理器
func TestRouter_Register(t *testing.T) {
	router := NewRouter()

	// 注册处理器
	handler1 := &MockHandler{priority: 100}
	handler2 := &MockHandler{priority: 200}
	handler3 := &MockHandler{priority: 50}

	router.Register(handler1)
	router.Register(handler2)
	router.Register(handler3)

	// 验证数量
	assert.Equal(t, 3, router.Count())

	// 验证优先级排序（数字越小越优先）
	handlers := router.GetHandlers()
	assert.Equal(t, 50, handlers[0].Priority())
	assert.Equal(t, 100, handlers[1].Priority())
	assert.Equal(t, 200, handlers[2].Priority())
}

// TestRouter_Route 测试路由
func TestRouter_Route(t *testing.T) {
	router := NewRouter()

	handler1 := &MockHandler{priority: 100, shouldMatch: true, continueChain: false}
	handler2 := &MockHandler{priority: 200, shouldMatch: true, continueChain: true}

	router.Register(handler1)
	router.Register(handler2)

	ctx := &Context{}

	err := router.Route(ctx)
	assert.NoError(t, err)

	// handler1 应该被执行
	assert.True(t, handler1.handleCalled)

	// handler2 不应该被执行（因为 handler1.ContinueChain() = false）
	assert.False(t, handler2.handleCalled)
}

// TestRouter_Route_ContinueChain 测试继续链
func TestRouter_Route_ContinueChain(t *testing.T) {
	router := NewRouter()

	handler1 := &MockHandler{priority: 100, shouldMatch: true, continueChain: true}
	handler2 := &MockHandler{priority: 200, shouldMatch: true, continueChain: false}

	router.Register(handler1)
	router.Register(handler2)

	ctx := &Context{}

	err := router.Route(ctx)
	assert.NoError(t, err)

	// 两个处理器都应该被执行
	assert.True(t, handler1.handleCalled)
	assert.True(t, handler2.handleCalled)
}

// TestRouter_Route_NoMatch 测试没有匹配的处理器
func TestRouter_Route_NoMatch(t *testing.T) {
	router := NewRouter()

	handler1 := &MockHandler{priority: 100, shouldMatch: false}
	handler2 := &MockHandler{priority: 200, shouldMatch: false}

	router.Register(handler1)
	router.Register(handler2)

	ctx := &Context{}

	err := router.Route(ctx)
	assert.NoError(t, err)

	// 没有处理器应该被执行
	assert.False(t, handler1.handleCalled)
	assert.False(t, handler2.handleCalled)
}

// TestMiddleware_Chain 测试中间件链
func TestMiddleware_Chain(t *testing.T) {
	var executed []string

	mw1 := func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) error {
			executed = append(executed, "mw1_before")
			err := next(ctx)
			executed = append(executed, "mw1_after")
			return err
		}
	}

	mw2 := func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) error {
			executed = append(executed, "mw2_before")
			err := next(ctx)
			executed = append(executed, "mw2_after")
			return err
		}
	}

	final := func(ctx *Context) error {
		executed = append(executed, "handler")
		return nil
	}

	// 组合中间件
	chain := Chain(mw1, mw2)(final)

	// 执行
	err := chain(&Context{})
	assert.NoError(t, err)

	// 验证执行顺序
	expected := []string{
		"mw1_before",
		"mw2_before",
		"handler",
		"mw2_after",
		"mw1_after",
	}
	assert.Equal(t, expected, executed)
}
