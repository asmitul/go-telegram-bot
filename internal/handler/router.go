package handler

import (
	"sort"
	"sync"
)

// Router 消息路由器
// 负责将消息分发到匹配的处理器
type Router struct {
	handlers    []Handler
	middlewares []Middleware
	mu          sync.RWMutex
}

// NewRouter 创建新的路由器
func NewRouter() *Router {
	return &Router{
		handlers:    make([]Handler, 0),
		middlewares: make([]Middleware, 0),
	}
}

// Register 注册处理器
func (r *Router) Register(h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers = append(r.handlers, h)

	// 按优先级排序（数字越小越优先）
	sort.Slice(r.handlers, func(i, j int) bool {
		return r.handlers[i].Priority() < r.handlers[j].Priority()
	})
}

// Use 注册全局中间件
// 中间件会应用到所有匹配的处理器
func (r *Router) Use(mw Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.middlewares = append(r.middlewares, mw)
}

// Route 路由消息到匹配的处理器
// 返回 error 表示处理过程中出现错误
func (r *Router) Route(ctx *Context) error {
	r.mu.RLock()
	handlers := r.handlers
	r.mu.RUnlock()

	var lastErr error
	matchedCount := 0

	// 遍历所有处理器，执行匹配的
	for _, h := range handlers {
		// 匹配检查
		if !h.Match(ctx) {
			continue
		}

		matchedCount++

		// 构建中间件链
		handler := r.buildChain(h)

		// 执行处理器
		if err := handler(ctx); err != nil {
			lastErr = err

			// 如果处理器不继续链，立即返回错误
			// 如果处理器继续链（如监听器），错误已在日志中间件记录
			if !h.ContinueChain() {
				return err
			}
			// 否则继续执行下一个处理器（错误已被记录）
		}

		// 检查是否继续链
		if !h.ContinueChain() {
			break
		}
	}

	// 如果没有处理器匹配，返回 nil（不是错误）
	return lastErr
}

// buildChain 构建中间件链
func (r *Router) buildChain(h Handler) HandlerFunc {
	// 最终处理器
	final := func(ctx *Context) error {
		return h.Handle(ctx)
	}

	// 从后往前包装中间件
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		final = r.middlewares[i](final)
	}

	return final
}

// Count 返回已注册的处理器数量
func (r *Router) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.handlers)
}

// GetHandlers 获取所有处理器（用于调试）
func (r *Router) GetHandlers() []Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 返回副本
	handlers := make([]Handler, len(r.handlers))
	copy(handlers, r.handlers)
	return handlers
}
