package listener

import (
	"sync"
	"telegram-bot/internal/handler"
	"time"
)

// AnalyticsHandler 数据分析处理器
// 统计消息数量、活跃用户等信息
type AnalyticsHandler struct {
	stats *Stats
	mu    sync.RWMutex
}

// Stats 统计数据
type Stats struct {
	TotalMessages    int64
	MessagesByChat   map[int64]int64
	ActiveUsers      map[int64]time.Time
	MessagesByHour   map[int]int64
	LastUpdated      time.Time
}

// NewAnalyticsHandler 创建数据分析处理器
func NewAnalyticsHandler() *AnalyticsHandler {
	return &AnalyticsHandler{
		stats: &Stats{
			MessagesByChat: make(map[int64]int64),
			ActiveUsers:    make(map[int64]time.Time),
			MessagesByHour: make(map[int]int64),
			LastUpdated:    time.Now(),
		},
	}
}

// Match 匹配所有消息
func (h *AnalyticsHandler) Match(ctx *handler.Context) bool {
	return true
}

// Handle 处理消息
func (h *AnalyticsHandler) Handle(ctx *handler.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()

	// 总消息数
	h.stats.TotalMessages++

	// 按聊天统计
	h.stats.MessagesByChat[ctx.ChatID]++

	// 活跃用户
	h.stats.ActiveUsers[ctx.UserID] = now

	// 按小时统计
	hour := now.Hour()
	h.stats.MessagesByHour[hour]++

	h.stats.LastUpdated = now

	// 定期清理（每1000条消息清理一次）
	if h.stats.TotalMessages%1000 == 0 {
		h.cleanupOldUsers(now)
	}

	return nil
}

// cleanupOldUsers 清理超过 30 天未活跃的用户（内部方法，调用前需加锁）
func (h *AnalyticsHandler) cleanupOldUsers(now time.Time) {
	cutoff := now.Add(-30 * 24 * time.Hour) // 30天

	for userID, lastSeen := range h.stats.ActiveUsers {
		if lastSeen.Before(cutoff) {
			delete(h.stats.ActiveUsers, userID)
		}
	}
}

// Priority 优先级
func (h *AnalyticsHandler) Priority() int {
	return 950
}

// ContinueChain 总是继续
func (h *AnalyticsHandler) ContinueChain() bool {
	return true
}

// GetStats 获取统计数据
func (h *AnalyticsHandler) GetStats() Stats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 返回副本
	stats := Stats{
		TotalMessages:  h.stats.TotalMessages,
		MessagesByChat: make(map[int64]int64),
		ActiveUsers:    make(map[int64]time.Time),
		MessagesByHour: make(map[int]int64),
		LastUpdated:    h.stats.LastUpdated,
	}

	for k, v := range h.stats.MessagesByChat {
		stats.MessagesByChat[k] = v
	}
	for k, v := range h.stats.ActiveUsers {
		stats.ActiveUsers[k] = v
	}
	for k, v := range h.stats.MessagesByHour {
		stats.MessagesByHour[k] = v
	}

	return stats
}

// GetActiveUserCount 获取活跃用户数（最近 N 分钟）
func (h *AnalyticsHandler) GetActiveUserCount(minutes int) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	count := 0

	for _, lastSeen := range h.stats.ActiveUsers {
		if lastSeen.After(cutoff) {
			count++
		}
	}

	return count
}
