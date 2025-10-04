package middleware

import (
	"fmt"
	"sync"
	"telegram-bot/internal/handler"
	"time"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(userID int64) bool
}

// RateLimitMiddleware 限流中间件
type RateLimitMiddleware struct {
	limiter RateLimiter
}

// NewRateLimitMiddleware 创建限流中间件
func NewRateLimitMiddleware(limiter RateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
	}
}

// Middleware 返回中间件函数
func (m *RateLimitMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			if !m.limiter.Allow(ctx.UserID) {
				return fmt.Errorf("⏱️ 操作过于频繁，请稍后再试")
			}
			return next(ctx)
		}
	}
}

// SimpleRateLimiter 简单的限流器实现（基于令牌桶）
type SimpleRateLimiter struct {
	rate     time.Duration // 每次请求的最小间隔
	capacity int           // 令牌桶容量
	tokens   map[int64]int // 用户的令牌数
	lastTime map[int64]time.Time
	mu       sync.Mutex
	stopChan chan struct{} // 停止清理 goroutine
	stopped  bool          // 是否已停止
}

// NewSimpleRateLimiter 创建简单限流器
// rate: 每次请求的最小间隔（如 1 秒）
// capacity: 令牌桶容量（允许突发请求数）
func NewSimpleRateLimiter(rate time.Duration, capacity int) *SimpleRateLimiter {
	limiter := &SimpleRateLimiter{
		rate:     rate,
		capacity: capacity,
		tokens:   make(map[int64]int),
		lastTime: make(map[int64]time.Time),
		stopChan: make(chan struct{}),
	}

	// 启动自动清理 goroutine
	// 每小时清理一次超过24小时未活动的用户数据
	go limiter.autoCleanup(1*time.Hour, 24*time.Hour)

	return limiter
}

// Allow 检查是否允许请求
func (l *SimpleRateLimiter) Allow(userID int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// 初始化
	if _, exists := l.tokens[userID]; !exists {
		l.tokens[userID] = l.capacity - 1
		l.lastTime[userID] = now
		return true
	}

	// 计算恢复的令牌数
	elapsed := now.Sub(l.lastTime[userID])
	recovered := int(elapsed / l.rate)

	if recovered > 0 {
		l.tokens[userID] += recovered
		if l.tokens[userID] > l.capacity {
			l.tokens[userID] = l.capacity
		}
		l.lastTime[userID] = now
	}

	// 检查是否有令牌
	if l.tokens[userID] > 0 {
		l.tokens[userID]--
		return true
	}

	return false
}

// Cleanup 清理长时间未使用的用户数据（防止内存泄漏）
func (l *SimpleRateLimiter) Cleanup(maxAge time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for userID, lastTime := range l.lastTime {
		if now.Sub(lastTime) > maxAge {
			delete(l.tokens, userID)
			delete(l.lastTime, userID)
		}
	}
}

// autoCleanup 自动清理 goroutine
func (l *SimpleRateLimiter) autoCleanup(interval, maxAge time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.Cleanup(maxAge)
		case <-l.stopChan:
			return
		}
	}
}

// Stop 停止自动清理 goroutine
func (l *SimpleRateLimiter) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.stopped {
		close(l.stopChan)
		l.stopped = true
	}
}
