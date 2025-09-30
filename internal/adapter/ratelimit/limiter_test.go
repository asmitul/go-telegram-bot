package ratelimit

import (
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	tb := NewTokenBucket(3, 1) // 容量3，每秒生成1个令牌

	// 首次访问应该有满容量的令牌
	for i := 0; i < 3; i++ {
		if !tb.Allow("user1") {
			t.Errorf("expected allow on request %d", i+1)
		}
	}

	// 第4次应该被拒绝（令牌用完）
	if tb.Allow("user1") {
		t.Error("expected deny on 4th request")
	}

	// 等待1秒，应该生成1个新令牌
	time.Sleep(1100 * time.Millisecond)
	if !tb.Allow("user1") {
		t.Error("expected allow after refill")
	}
}

func TestTokenBucket_MultipleUsers(t *testing.T) {
	tb := NewTokenBucket(2, 1)

	// 用户1使用2个令牌
	if !tb.Allow("user1") {
		t.Error("user1 request 1 should be allowed")
	}
	if !tb.Allow("user1") {
		t.Error("user1 request 2 should be allowed")
	}

	// 用户2应该有独立的令牌桶
	if !tb.Allow("user2") {
		t.Error("user2 request 1 should be allowed")
	}
	if !tb.Allow("user2") {
		t.Error("user2 request 2 should be allowed")
	}

	// 两个用户的令牌都用完
	if tb.Allow("user1") {
		t.Error("user1 should be rate limited")
	}
	if tb.Allow("user2") {
		t.Error("user2 should be rate limited")
	}
}

func TestTokenBucket_Reset(t *testing.T) {
	tb := NewTokenBucket(1, 1)

	// 用完令牌
	tb.Allow("user1")
	if tb.Allow("user1") {
		t.Error("expected deny before reset")
	}

	// 重置
	tb.Reset("user1")

	// 应该恢复到满容量
	if !tb.Allow("user1") {
		t.Error("expected allow after reset")
	}
}

func TestTokenBucket_ResetAll(t *testing.T) {
	tb := NewTokenBucket(1, 1)

	// 多个用户用完令牌
	tb.Allow("user1")
	tb.Allow("user2")

	if tb.Allow("user1") || tb.Allow("user2") {
		t.Error("expected deny before reset")
	}

	// 全部重置
	tb.ResetAll()

	// 所有用户应该恢复
	if !tb.Allow("user1") || !tb.Allow("user2") {
		t.Error("expected allow after reset all")
	}
}

func TestSlidingWindow_Allow(t *testing.T) {
	sw := NewSlidingWindow(3, 1*time.Second) // 1秒内最多3个请求

	// 前3个请求应该通过
	for i := 0; i < 3; i++ {
		if !sw.Allow("user1") {
			t.Errorf("expected allow on request %d", i+1)
		}
	}

	// 第4个请求应该被拒绝
	if sw.Allow("user1") {
		t.Error("expected deny on 4th request")
	}

	// 等待窗口过期
	time.Sleep(1100 * time.Millisecond)

	// 应该可以再次请求
	if !sw.Allow("user1") {
		t.Error("expected allow after window expires")
	}
}

func TestSlidingWindow_MultipleUsers(t *testing.T) {
	sw := NewSlidingWindow(2, 1*time.Second)

	// 用户1使用2次
	sw.Allow("user1")
	sw.Allow("user1")

	// 用户2应该有独立的窗口
	if !sw.Allow("user2") {
		t.Error("user2 request should be allowed")
	}
}

func TestSlidingWindow_Reset(t *testing.T) {
	sw := NewSlidingWindow(1, 1*time.Second)

	// 用完配额
	sw.Allow("user1")
	if sw.Allow("user1") {
		t.Error("expected deny before reset")
	}

	// 重置
	sw.Reset("user1")

	// 应该可以再次请求
	if !sw.Allow("user1") {
		t.Error("expected allow after reset")
	}
}

func TestCompositeLimiter_Allow(t *testing.T) {
	tb := NewTokenBucket(3, 1)
	sw := NewSlidingWindow(2, 1*time.Second)
	cl := NewCompositeLimiter(tb, sw)

	// 第1个请求应该通过（两个限流器都通过）
	if !cl.Allow("user1") {
		t.Error("expected allow on 1st request")
	}

	// 第2个请求应该通过
	if !cl.Allow("user1") {
		t.Error("expected allow on 2nd request")
	}

	// 第3个请求应该被滑动窗口拒绝（窗口限制2个）
	if cl.Allow("user1") {
		t.Error("expected deny on 3rd request (sliding window limit)")
	}
}

func TestManager_AllowUser(t *testing.T) {
	config := RateLimitConfig{
		UserRequestsPerSecond: 1,
		UserBurstSize:         2,
		CommandLimit:          10,
		CommandWindow:         time.Minute,
	}
	m := NewManager(config)

	// 前2个请求应该通过（桶容量为2）
	if !m.AllowUser(123) {
		t.Error("expected allow on 1st request")
	}
	if !m.AllowUser(123) {
		t.Error("expected allow on 2nd request")
	}

	// 第3个请求应该被拒绝
	if m.AllowUser(123) {
		t.Error("expected deny on 3rd request")
	}

	// 不同用户应该独立限流
	if !m.AllowUser(456) {
		t.Error("expected allow for different user")
	}
}

func TestManager_AllowCommand(t *testing.T) {
	config := RateLimitConfig{
		UserRequestsPerSecond: 10,
		UserBurstSize:         10,
		CommandLimit:          2,
		CommandWindow:         time.Second,
	}
	m := NewManager(config)

	// 前2个命令应该通过
	if !m.AllowCommand(123, "test") {
		t.Error("expected allow on 1st command")
	}
	if !m.AllowCommand(123, "test") {
		t.Error("expected allow on 2nd command")
	}

	// 第3个命令应该被拒绝
	if m.AllowCommand(123, "test") {
		t.Error("expected deny on 3rd command")
	}

	// 不同命令应该独立限流
	if !m.AllowCommand(123, "other") {
		t.Error("expected allow for different command")
	}
}

func TestManager_Allow(t *testing.T) {
	config := RateLimitConfig{
		UserRequestsPerSecond: 1,
		UserBurstSize:         1,
		CommandLimit:          2,
		CommandWindow:         time.Second,
	}
	m := NewManager(config)

	// 第1个请求应该通过
	if !m.Allow(123, "test") {
		t.Error("expected allow on 1st request")
	}

	// 第2个请求应该被用户级别限流拒绝
	if m.Allow(123, "test") {
		t.Error("expected deny on 2nd request (user limit)")
	}
}

func TestManager_Reset(t *testing.T) {
	config := RateLimitConfig{
		UserRequestsPerSecond: 1,
		UserBurstSize:         1,
		CommandLimit:          1,
		CommandWindow:         time.Minute,
	}
	m := NewManager(config)

	// 用完配额
	m.Allow(123, "test")

	// 应该被拒绝
	if m.Allow(123, "test") {
		t.Error("expected deny before reset")
	}

	// 重置用户
	m.ResetUser(123)

	// 用户限流应该恢复，但命令限流仍然存在
	if m.Allow(123, "test") {
		t.Error("expected deny (command limit still active)")
	}

	// 重置用户和命令
	m.ResetUser(123)
	m.ResetCommand(123, "test")

	// 现在应该可以通过
	if !m.Allow(123, "test") {
		t.Error("expected allow after reset both")
	}
}

func TestManager_ResetAll(t *testing.T) {
	config := RateLimitConfig{
		UserRequestsPerSecond: 1,
		UserBurstSize:         1,
		CommandLimit:          1,
		CommandWindow:         time.Minute,
	}
	m := NewManager(config)

	// 多个用户用完配额
	m.Allow(123, "test")
	m.Allow(456, "test")

	// 全部重置
	m.ResetAll()

	// 所有用户应该恢复
	if !m.Allow(123, "test") {
		t.Error("expected allow for user 123 after reset all")
	}
	if !m.Allow(456, "test") {
		t.Error("expected allow for user 456 after reset all")
	}
}

func TestDefaultManager(t *testing.T) {
	m := NewDefaultManager()

	// 应该使用默认配置
	if m.config.UserRequestsPerSecond != DefaultConfig.UserRequestsPerSecond {
		t.Error("expected default user rate")
	}
	if m.config.CommandLimit != DefaultConfig.CommandLimit {
		t.Error("expected default command limit")
	}
}

func TestInt64ToString(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{123, "123"},
		{-456, "-456"},
		{1, "1"},
		{-1, "-1"},
		{9999999999, "9999999999"},
	}

	for _, tt := range tests {
		result := int64ToString(tt.input)
		if result != tt.expected {
			t.Errorf("int64ToString(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestFormatKeys(t *testing.T) {
	userKey := formatUserKey(123)
	if userKey != "user:123" {
		t.Errorf("expected 'user:123', got '%s'", userKey)
	}

	cmdKey := formatCommandKey(123, "test")
	if cmdKey != "cmd:123:test" {
		t.Errorf("expected 'cmd:123:test', got '%s'", cmdKey)
	}
}

func BenchmarkTokenBucket_Allow(b *testing.B) {
	tb := NewTokenBucket(1000, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.Allow("user1")
	}
}

func BenchmarkSlidingWindow_Allow(b *testing.B) {
	sw := NewSlidingWindow(1000, time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sw.Allow("user1")
	}
}

func BenchmarkManager_Allow(b *testing.B) {
	m := NewDefaultManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Allow(123, "test")
	}
}
