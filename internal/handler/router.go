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
			if !h.ContinueChain() {
				// 命令类处理器：错误是用户级的，需要立即返回
				// 例如：权限不足、参数错误等，这些应该反馈给用户
				return err
			} else {
				// 监听器类处理器：错误是系统级的，记录但不中断
				// 例如：日志记录失败、统计更新失败等
				// 这些错误已经在 LoggingMiddleware 中记录
				// 保存最后一个错误，调用方可以决定如何处理
				lastErr = err
			}
		}

		// 检查是否继续链
		if !h.ContinueChain() {
			break
		}
	}

	// 返回最后一个监听器的错误（如果有）
	// 如果没有处理器匹配或所有处理器都成功，返回 nil
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
