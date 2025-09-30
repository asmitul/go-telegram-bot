package ratelimit

import (
	"sync"
	"time"
)

// Limiter 限流器接口
type Limiter interface {
	// Allow 检查是否允许请求
	Allow(key string) bool
	// Reset 重置指定 key 的限流状态
	Reset(key string)
	// ResetAll 重置所有限流状态
	ResetAll()
}

// TokenBucket 令牌桶实现
type TokenBucket struct {
	capacity   int           // 桶容量
	rate       int           // 每秒生成的令牌数
	tokens     map[string]int // 当前令牌数
	lastRefill map[string]time.Time // 上次填充时间
	mu         sync.RWMutex
}

// NewTokenBucket 创建新的令牌桶限流器
// capacity: 桶容量（最大令牌数）
// rate: 每秒生成的令牌数
func NewTokenBucket(capacity, rate int) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		rate:       rate,
		tokens:     make(map[string]int),
		lastRefill: make(map[string]time.Time),
	}
}

// Allow 检查是否允许请求
func (tb *TokenBucket) Allow(key string) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// 填充令牌
	tb.refill(key)

	// 检查令牌
	if tb.tokens[key] > 0 {
		tb.tokens[key]--
		return true
	}

	return false
}

// refill 填充令牌（内部方法，调用前需加锁）
func (tb *TokenBucket) refill(key string) {
	now := time.Now()

	// 首次访问，初始化为满容量
	if _, exists := tb.lastRefill[key]; !exists {
		tb.tokens[key] = tb.capacity
		tb.lastRefill[key] = now
		return
	}

	// 计算应该生成的令牌数
	elapsed := now.Sub(tb.lastRefill[key])
	tokensToAdd := int(elapsed.Seconds() * float64(tb.rate))

	if tokensToAdd > 0 {
		tb.tokens[key] += tokensToAdd
		if tb.tokens[key] > tb.capacity {
			tb.tokens[key] = tb.capacity
		}
		tb.lastRefill[key] = now
	}
}

// Reset 重置指定 key 的限流状态
func (tb *TokenBucket) Reset(key string) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	delete(tb.tokens, key)
	delete(tb.lastRefill, key)
}

// ResetAll 重置所有限流状态
func (tb *TokenBucket) ResetAll() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.tokens = make(map[string]int)
	tb.lastRefill = make(map[string]time.Time)
}

// SlidingWindow 滑动窗口限流器
type SlidingWindow struct {
	limit    int                      // 时间窗口内的最大请求数
	window   time.Duration            // 时间窗口大小
	requests map[string][]time.Time   // 请求时间记录
	mu       sync.RWMutex
}

// NewSlidingWindow 创建新的滑动窗口限流器
// limit: 时间窗口内的最大请求数
// window: 时间窗口大小
func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
	return &SlidingWindow{
		limit:    limit,
		window:   window,
		requests: make(map[string][]time.Time),
	}
}

// Allow 检查是否允许请求
func (sw *SlidingWindow) Allow(key string) bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-sw.window)

	// 清理过期请求
	if reqs, exists := sw.requests[key]; exists {
		validReqs := make([]time.Time, 0)
		for _, t := range reqs {
			if t.After(windowStart) {
				validReqs = append(validReqs, t)
			}
		}
		sw.requests[key] = validReqs
	}

	// 检查是否超过限制
	if len(sw.requests[key]) >= sw.limit {
		return false
	}

	// 记录请求
	sw.requests[key] = append(sw.requests[key], now)
	return true
}

// Reset 重置指定 key 的限流状态
func (sw *SlidingWindow) Reset(key string) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	delete(sw.requests, key)
}

// ResetAll 重置所有限流状态
func (sw *SlidingWindow) ResetAll() {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.requests = make(map[string][]time.Time)
}

// Compositelimiter 组合限流器，可以同时应用多个限流策略
type CompositeLimiter struct {
	limiters []Limiter
}

// NewCompositeLimiter 创建组合限流器
func NewCompositeLimiter(limiters ...Limiter) *CompositeLimiter {
	return &CompositeLimiter{
		limiters: limiters,
	}
}

// Allow 检查是否允许请求（所有限流器都通过才允许）
func (cl *CompositeLimiter) Allow(key string) bool {
	for _, limiter := range cl.limiters {
		if !limiter.Allow(key) {
			return false
		}
	}
	return true
}

// Reset 重置指定 key 的限流状态
func (cl *CompositeLimiter) Reset(key string) {
	for _, limiter := range cl.limiters {
		limiter.Reset(key)
	}
}

// ResetAll 重置所有限流状态
func (cl *CompositeLimiter) ResetAll() {
	for _, limiter := range cl.limiters {
		limiter.ResetAll()
	}
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// 用户级别限流：每秒最多 N 个请求
	UserRequestsPerSecond int
	// 用户级别限流：桶容量（突发请求数）
	UserBurstSize int
	// 命令级别限流：时间窗口内最多 N 个请求
	CommandLimit int
	// 命令级别限流：时间窗口大小
	CommandWindow time.Duration
}

// DefaultConfig 默认限流配置
var DefaultConfig = RateLimitConfig{
	UserRequestsPerSecond: 2,      // 每秒2个请求
	UserBurstSize:         5,      // 允许突发5个请求
	CommandLimit:          10,     // 1分钟内最多10个命令
	CommandWindow:         time.Minute,
}

// Manager 限流管理器
type Manager struct {
	userLimiter    Limiter
	commandLimiter Limiter
	config         RateLimitConfig
}

// NewManager 创建限流管理器
func NewManager(config RateLimitConfig) *Manager {
	return &Manager{
		userLimiter:    NewTokenBucket(config.UserBurstSize, config.UserRequestsPerSecond),
		commandLimiter: NewSlidingWindow(config.CommandLimit, config.CommandWindow),
		config:         config,
	}
}

// NewDefaultManager 创建使用默认配置的限流管理器
func NewDefaultManager() *Manager {
	return NewManager(DefaultConfig)
}

// AllowUser 检查用户是否允许发送请求
func (m *Manager) AllowUser(userID int64) bool {
	key := formatUserKey(userID)
	return m.userLimiter.Allow(key)
}

// AllowCommand 检查命令是否允许执行
func (m *Manager) AllowCommand(userID int64, commandName string) bool {
	key := formatCommandKey(userID, commandName)
	return m.commandLimiter.Allow(key)
}

// Allow 综合检查（用户级别 + 命令级别）
func (m *Manager) Allow(userID int64, commandName string) bool {
	return m.AllowUser(userID) && m.AllowCommand(userID, commandName)
}

// ResetUser 重置用户的限流状态
func (m *Manager) ResetUser(userID int64) {
	key := formatUserKey(userID)
	m.userLimiter.Reset(key)
}

// ResetCommand 重置命令的限流状态
func (m *Manager) ResetCommand(userID int64, commandName string) {
	key := formatCommandKey(userID, commandName)
	m.commandLimiter.Reset(key)
}

// ResetAll 重置所有限流状态
func (m *Manager) ResetAll() {
	m.userLimiter.ResetAll()
	m.commandLimiter.ResetAll()
}

// formatUserKey 格式化用户 key
func formatUserKey(userID int64) string {
	return "user:" + int64ToString(userID)
}

// formatCommandKey 格式化命令 key
func formatCommandKey(userID int64, commandName string) string {
	return "cmd:" + int64ToString(userID) + ":" + commandName
}

// int64ToString 将 int64 转换为字符串（简单实现）
func int64ToString(n int64) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}
