# 监听器开发指南

## 📚 目录

- [概述](#概述)
- [核心概念](#核心概念)
- [快速开始](#快速开始)
- [完整代码示例](#完整代码示例)
- [数据统计与分析](#数据统计与分析)
- [注册流程](#注册流程)
- [测试方法](#测试方法)
- [实际场景示例](#实际场景示例)
- [常见问题](#常见问题)

---

## 概述

**监听器** (Listener) 是本机器人框架的特殊处理器类型，用于监控和记录**所有消息**，而不对消息内容做任何过滤。适合日志记录、数据统计、监控告警等场景。

### 适用场景

- ✅ 消息日志记录（审计、调试）
- ✅ 数据统计分析（活跃用户、消息量、热点时段）
- ✅ 性能监控（响应时间、错误率）
- ✅ 行为分析（用户使用习惯、功能热度）
- ✅ 安全监控（异常行为检测、频率限制）
- ✅ 业务指标收集（转化率、留存率）

### 不适用场景

- ❌ 响应特定命令 → 使用 **命令处理器**
- ❌ 匹配关键词 → 使用 **关键词处理器**
- ❌ 复杂模式匹配 → 使用 **正则匹配处理器**

---

## 核心概念

### 处理器接口

所有监听器必须实现 `handler.Handler` 接口：

```go
type Handler interface {
    Match(ctx *Context) bool      // 始终返回 true（匹配所有消息）
    Handle(ctx *Context) error    // 处理消息（记录、统计）
    Priority() int                // 优先级（900-999）
    ContinueChain() bool          // 始终返回 true（不阻断）
}
```

### 核心特征

1. **`Match()` 始终返回 `true`**
   ```go
   func (h *Listener) Match(ctx *handler.Context) bool {
       return true // 匹配所有消息
   }
   ```

2. **`Priority()` 在 900-999 范围**
   - 最低优先级（最后执行）
   - 确保不影响业务处理器

3. **`ContinueChain()` 始终返回 `true`**
   ```go
   func (h *Listener) ContinueChain() bool {
       return true // 不阻断后续处理器
   }
   ```

4. **`Handle()` 不应返回错误**
   - 监听器的错误不应影响业务流程
   - 内部捕获并记录错误

### 优先级规则

- **优先级范围**：`900-999`
- **数值越小，优先级越高**
- **推荐分配**：
  - `900-929`：日志记录
  - `930-959`：数据统计
  - `960-989`：性能监控
  - `990-999`：兜底监听器

### 执行顺序

```
命令处理器（100）
    ↓
关键词处理器（200）
    ↓
正则处理器（300）
    ↓
监听器（900+）← 最后执行
```

---

## 快速开始

### 步骤 1：创建监听器文件

在 `internal/handlers/listener/` 目录下创建新文件，例如 `audit.go`：

```bash
touch internal/handlers/listener/audit.go
```

### 步骤 2：编写监听器代码

```go
package listener

import (
    "telegram-bot/internal/handler"
)

type AuditListener struct {
    logger Logger
}

type Logger interface {
    Info(msg string, fields ...interface{})
}

func NewAuditListener(logger Logger) *AuditListener {
    return &AuditListener{
        logger: logger,
    }
}

func (h *AuditListener) Match(ctx *handler.Context) bool {
    return true // 匹配所有消息
}

func (h *AuditListener) Handle(ctx *handler.Context) error {
    h.logger.Info("message_audit",
        "user_id", ctx.UserID,
        "chat_id", ctx.ChatID,
        "text", ctx.Text,
    )
    return nil
}

func (h *AuditListener) Priority() int {
    return 900
}

func (h *AuditListener) ContinueChain() bool {
    return true
}
```

### 步骤 3：注册监听器

在 `cmd/bot/main.go` 的 `registerHandlers()` 函数中添加：

```go
// 4. 监听器（优先级 900+）
router.Register(listener.NewMessageLoggerHandler(appLogger))
router.Register(listener.NewAnalyticsHandler())
router.Register(listener.NewAuditListener(appLogger)) // 新增
```

### 步骤 4：测试

启动机器人后，发送任意消息，检查日志输出。

---

## 完整代码示例

### 示例 1：消息日志记录（项目内置）

```go
package listener

import (
    "telegram-bot/internal/handler"
    "telegram-bot/internal/middleware"
)

type MessageLoggerHandler struct {
    logger middleware.Logger
}

func NewMessageLoggerHandler(logger middleware.Logger) *MessageLoggerHandler {
    return &MessageLoggerHandler{
        logger: logger,
    }
}

func (h *MessageLoggerHandler) Match(ctx *handler.Context) bool {
    return true // 匹配所有消息
}

func (h *MessageLoggerHandler) Handle(ctx *handler.Context) error {
    // 记录详细的消息信息
    h.logger.Debug("message_logged",
        "chat_type", ctx.ChatType,
        "chat_id", ctx.ChatID,
        "chat_title", ctx.ChatTitle,
        "user_id", ctx.UserID,
        "username", ctx.Username,
        "first_name", ctx.FirstName,
        "text", ctx.Text,
        "message_id", ctx.MessageID,
    )

    return nil
}

func (h *MessageLoggerHandler) Priority() int {
    return 900
}

func (h *MessageLoggerHandler) ContinueChain() bool {
    return true
}
```

### 示例 2：数据分析统计（项目内置）

```go
package listener

import (
    "sync"
    "telegram-bot/internal/handler"
    "time"
)

type AnalyticsHandler struct {
    stats *Stats
    mu    sync.RWMutex
}

type Stats struct {
    TotalMessages  int64
    MessagesByChat map[int64]int64    // 每个聊天的消息数
    ActiveUsers    map[int64]time.Time // 用户ID -> 最后活跃时间
    MessagesByHour map[int]int64       // 每小时的消息数
    LastUpdated    time.Time
}

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

func (h *AnalyticsHandler) Match(ctx *handler.Context) bool {
    return true
}

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

    return nil
}

func (h *AnalyticsHandler) Priority() int {
    return 950
}

func (h *AnalyticsHandler) ContinueChain() bool {
    return true
}

// GetStats 获取统计数据（线程安全）
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
```

### 示例 3：性能监控

```go
package listener

import (
    "sync"
    "telegram-bot/internal/handler"
    "time"
)

type PerformanceMonitor struct {
    metrics *Metrics
    mu      sync.RWMutex
}

type Metrics struct {
    TotalRequests    int64
    SuccessCount     int64
    ErrorCount       int64
    TotalDuration    time.Duration
    MinDuration      time.Duration
    MaxDuration      time.Duration
    SlowRequestCount int64 // 超过阈值的请求数
    LastReset        time.Time
}

func NewPerformanceMonitor() *PerformanceMonitor {
    return &PerformanceMonitor{
        metrics: &Metrics{
            LastReset:   time.Now(),
            MinDuration: time.Hour, // 初始值设为很大
        },
    }
}

func (h *PerformanceMonitor) Match(ctx *handler.Context) bool {
    return true
}

func (h *PerformanceMonitor) Handle(ctx *handler.Context) error {
    // 获取请求开始时间（需要在 middleware 中设置）
    startTime, ok := ctx.Get("request_start_time")
    if !ok {
        startTime = time.Now()
    }

    duration := time.Since(startTime.(time.Time))

    h.mu.Lock()
    defer h.mu.Unlock()

    h.metrics.TotalRequests++
    h.metrics.TotalDuration += duration

    // 更新最小/最大耗时
    if duration < h.metrics.MinDuration {
        h.metrics.MinDuration = duration
    }
    if duration > h.metrics.MaxDuration {
        h.metrics.MaxDuration = duration
    }

    // 慢请求检测（超过 1 秒）
    if duration > time.Second {
        h.metrics.SlowRequestCount++
    }

    // 检查是否有错误（需要在 middleware 中设置）
    if hasError, _ := ctx.Get("has_error"); hasError == true {
        h.metrics.ErrorCount++
    } else {
        h.metrics.SuccessCount++
    }

    return nil
}

func (h *PerformanceMonitor) Priority() int {
    return 980 // 较高优先级的监听器
}

func (h *PerformanceMonitor) ContinueChain() bool {
    return true
}

// GetMetrics 获取性能指标
func (h *PerformanceMonitor) GetMetrics() Metrics {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return *h.metrics
}

// GetAverageDuration 获取平均响应时间
func (h *PerformanceMonitor) GetAverageDuration() time.Duration {
    h.mu.RLock()
    defer h.mu.RUnlock()

    if h.metrics.TotalRequests == 0 {
        return 0
    }

    return h.metrics.TotalDuration / time.Duration(h.metrics.TotalRequests)
}

// Reset 重置指标
func (h *PerformanceMonitor) Reset() {
    h.mu.Lock()
    defer h.mu.Unlock()

    h.metrics = &Metrics{
        LastReset:   time.Now(),
        MinDuration: time.Hour,
    }
}
```

### 示例 4：安全监控

```go
package listener

import (
    "sync"
    "telegram-bot/internal/handler"
    "time"
)

type SecurityMonitor struct {
    rateLimits map[int64]*RateLimit // 用户ID -> 频率限制
    mu         sync.RWMutex
    logger     Logger
}

type RateLimit struct {
    MessageCount int
    FirstMessage time.Time
    LastMessage  time.Time
    IsBlocked    bool
}

type Logger interface {
    Warn(msg string, fields ...interface{})
    Error(msg string, fields ...interface{})
}

func NewSecurityMonitor(logger Logger) *SecurityMonitor {
    monitor := &SecurityMonitor{
        rateLimits: make(map[int64]*RateLimit),
        logger:     logger,
    }

    // 启动清理协程
    go monitor.cleanupRoutine()

    return monitor
}

func (h *SecurityMonitor) Match(ctx *handler.Context) bool {
    return true
}

func (h *SecurityMonitor) Handle(ctx *handler.Context) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    userID := ctx.UserID
    now := time.Now()

    // 获取或创建用户的频率限制记录
    limit, exists := h.rateLimits[userID]
    if !exists {
        limit = &RateLimit{
            FirstMessage: now,
        }
        h.rateLimits[userID] = limit
    }

    // 更新统计
    limit.MessageCount++
    limit.LastMessage = now

    // 检测异常频率（1分钟内超过30条消息）
    if now.Sub(limit.FirstMessage) < time.Minute {
        if limit.MessageCount > 30 {
            if !limit.IsBlocked {
                limit.IsBlocked = true
                h.logger.Warn("rate_limit_exceeded",
                    "user_id", userID,
                    "message_count", limit.MessageCount,
                    "duration", now.Sub(limit.FirstMessage),
                )

                // 可选：自动封禁或通知管理员
                // ctx.Reply("⚠️ 消息发送过快，请稍后再试")
            }
        }
    } else {
        // 重置计数器
        limit.MessageCount = 1
        limit.FirstMessage = now
        limit.IsBlocked = false
    }

    // 检测可疑行为
    if h.isSuspicious(ctx) {
        h.logger.Error("suspicious_activity",
            "user_id", userID,
            "chat_id", ctx.ChatID,
            "text", ctx.Text,
        )
    }

    return nil
}

func (h *SecurityMonitor) isSuspicious(ctx *handler.Context) bool {
    // 示例：检测超长消息
    if len(ctx.Text) > 4000 {
        return true
    }

    // 示例：检测大量链接
    // linkCount := countLinks(ctx.Text)
    // if linkCount > 5 {
    //     return true
    // }

    return false
}

func (h *SecurityMonitor) cleanupRoutine() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        h.mu.Lock()

        now := time.Now()
        for userID, limit := range h.rateLimits {
            // 清理 10 分钟前的记录
            if now.Sub(limit.LastMessage) > 10*time.Minute {
                delete(h.rateLimits, userID)
            }
        }

        h.mu.Unlock()
    }
}

func (h *SecurityMonitor) Priority() int {
    return 920 // 高优先级监听器
}

func (h *SecurityMonitor) ContinueChain() bool {
    return true
}
```

---

## 数据统计与分析

### 1. 实时统计

```go
type RealtimeStats struct {
    stats map[string]int64
    mu    sync.RWMutex
}

func (h *RealtimeStats) Handle(ctx *handler.Context) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    // 按聊天类型统计
    key := "chat_type:" + ctx.ChatType
    h.stats[key]++

    // 按小时统计
    hour := time.Now().Hour()
    hourKey := fmt.Sprintf("hour:%02d", hour)
    h.stats[hourKey]++

    // 按命令统计（如果是命令）
    if strings.HasPrefix(ctx.Text, "/") {
        cmd := strings.Fields(ctx.Text)[0]
        cmdKey := "command:" + cmd
        h.stats[cmdKey]++
    }

    return nil
}
```

### 2. 持久化统计

```go
type PersistentStats struct {
    db     *mongo.Database
    logger Logger
}

func (h *PersistentStats) Handle(ctx *handler.Context) error {
    // 异步保存到数据库
    go func() {
        record := bson.M{
            "user_id":    ctx.UserID,
            "chat_id":    ctx.ChatID,
            "chat_type":  ctx.ChatType,
            "text":       ctx.Text,
            "created_at": time.Now(),
        }

        collection := h.db.Collection("message_logs")
        _, err := collection.InsertOne(context.Background(), record)
        if err != nil {
            h.logger.Error("failed_to_save_log", "error", err)
        }
    }()

    return nil
}
```

### 3. 聚合统计

```go
type AggregatedStats struct {
    daily   map[string]*DayStats
    mu      sync.RWMutex
}

type DayStats struct {
    Date         string
    TotalUsers   int
    TotalChats   int
    TotalMessages int
    UniqueUsers  map[int64]bool
    UniqueChats  map[int64]bool
}

func (h *AggregatedStats) Handle(ctx *handler.Context) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    today := time.Now().Format("2006-01-02")

    stats, exists := h.daily[today]
    if !exists {
        stats = &DayStats{
            Date:        today,
            UniqueUsers: make(map[int64]bool),
            UniqueChats: make(map[int64]bool),
        }
        h.daily[today] = stats
    }

    stats.TotalMessages++
    stats.UniqueUsers[ctx.UserID] = true
    stats.UniqueChats[ctx.ChatID] = true
    stats.TotalUsers = len(stats.UniqueUsers)
    stats.TotalChats = len(stats.UniqueChats)

    return nil
}
```

---

## 注册流程

### 1. 基本注册

在 `cmd/bot/main.go` 的 `registerHandlers()` 函数中：

```go
func registerHandlers(
    router *handler.Router,
    groupRepo *mongodb.GroupRepository,
    userRepo *mongodb.UserRepository,
    appLogger logger.Logger,
) {
    // 1. 命令处理器（优先级 100）
    router.Register(command.NewPingHandler(groupRepo))

    // 2. 关键词处理器（优先级 200）
    router.Register(keyword.NewGreetingHandler())

    // 3. 正则处理器（优先级 300）
    router.Register(pattern.NewWeatherHandler())

    // 4. 监听器（优先级 900+）
    router.Register(listener.NewMessageLoggerHandler(appLogger))      // 900
    router.Register(listener.NewSecurityMonitor(appLogger))           // 920
    router.Register(listener.NewAnalyticsHandler())                   // 950
    router.Register(listener.NewPerformanceMonitor())                 // 980

    appLogger.Info("Registered handlers breakdown",
        "commands", 1,
        "keywords", 1,
        "patterns", 1,
        "listeners", 4, // 更新数量
    )
}
```

### 2. 按优先级顺序注册

```go
// 高优先级监听器（900-929）- 日志记录
router.Register(listener.NewMessageLoggerHandler(appLogger))    // 900
router.Register(listener.NewAuditLogger(appLogger))             // 910
router.Register(listener.NewSecurityMonitor(appLogger))         // 920

// 中优先级监听器（930-959）- 数据统计
router.Register(listener.NewAnalyticsHandler())                 // 950
router.Register(listener.NewUserBehaviorTracker())              // 955

// 低优先级监听器（960-999）- 性能监控
router.Register(listener.NewPerformanceMonitor())               // 980
router.Register(listener.NewHealthCheck())                      // 990
```

---

## 测试方法

### 1. 单元测试

创建 `internal/handlers/listener/analytics_test.go`：

```go
package listener

import (
    "testing"
    "telegram-bot/internal/handler"
    "github.com/stretchr/testify/assert"
)

func TestAnalyticsHandler_Match(t *testing.T) {
    h := NewAnalyticsHandler()

    ctx := &handler.Context{
        Text:     "任意消息",
        ChatType: "private",
    }

    // 监听器应该匹配所有消息
    assert.True(t, h.Match(ctx))
}

func TestAnalyticsHandler_Handle(t *testing.T) {
    h := NewAnalyticsHandler()

    ctx := &handler.Context{
        UserID:   123456789,
        ChatID:   -1001234567890,
        Text:     "test message",
        ChatType: "group",
    }

    err := h.Handle(ctx)
    assert.NoError(t, err)

    // 验证统计数据
    stats := h.GetStats()
    assert.Equal(t, int64(1), stats.TotalMessages)
    assert.Equal(t, int64(1), stats.MessagesByChat[ctx.ChatID])
    assert.Contains(t, stats.ActiveUsers, ctx.UserID)
}

func TestAnalyticsHandler_Priority(t *testing.T) {
    h := NewAnalyticsHandler()
    assert.Equal(t, 950, h.Priority())
}

func TestAnalyticsHandler_ContinueChain(t *testing.T) {
    h := NewAnalyticsHandler()
    assert.True(t, h.ContinueChain())
}

func TestAnalyticsHandler_GetActiveUserCount(t *testing.T) {
    h := NewAnalyticsHandler()

    // 模拟 3 个用户发送消息
    for i := 1; i <= 3; i++ {
        ctx := &handler.Context{
            UserID: int64(i),
            ChatID: -1001234567890,
        }
        h.Handle(ctx)
    }

    // 验证活跃用户数
    count := h.GetActiveUserCount(5) // 最近 5 分钟
    assert.Equal(t, 3, count)
}
```

运行测试：

```bash
go test ./internal/handlers/listener/... -v
```

### 2. 集成测试

```go
func TestListener_Integration(t *testing.T) {
    // 创建完整的处理器链
    router := handler.NewRouter()

    // 注册监听器
    analytics := listener.NewAnalyticsHandler()
    router.Register(analytics)

    // 模拟消息
    ctx := &handler.Context{
        UserID:   123456789,
        ChatID:   -1001234567890,
        Text:     "test",
        ChatType: "group",
    }

    // 执行路由
    err := router.Route(ctx)
    assert.NoError(t, err)

    // 验证统计
    stats := analytics.GetStats()
    assert.Equal(t, int64(1), stats.TotalMessages)
}
```

### 3. 性能测试

```go
func BenchmarkAnalyticsHandler(b *testing.B) {
    h := NewAnalyticsHandler()

    ctx := &handler.Context{
        UserID:   123456789,
        ChatID:   -1001234567890,
        Text:     "benchmark test",
        ChatType: "group",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        h.Handle(ctx)
    }
}
```

运行性能测试：

```bash
go test ./internal/handlers/listener/... -bench=. -benchmem
```

---

## 实际场景示例

### 场景 1：用户行为追踪

```go
package listener

import (
    "sync"
    "telegram-bot/internal/handler"
    "time"
)

type UserBehaviorTracker struct {
    sessions map[int64]*UserSession
    mu       sync.RWMutex
}

type UserSession struct {
    UserID        int64
    FirstSeen     time.Time
    LastSeen      time.Time
    MessageCount  int
    CommandsUsed  map[string]int
    ChatsVisited  map[int64]bool
}

func NewUserBehaviorTracker() *UserBehaviorTracker {
    return &UserBehaviorTracker{
        sessions: make(map[int64]*UserSession),
    }
}

func (h *UserBehaviorTracker) Match(ctx *handler.Context) bool {
    return true
}

func (h *UserBehaviorTracker) Handle(ctx *handler.Context) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    session, exists := h.sessions[ctx.UserID]
    if !exists {
        session = &UserSession{
            UserID:       ctx.UserID,
            FirstSeen:    time.Now(),
            CommandsUsed: make(map[string]int),
            ChatsVisited: make(map[int64]bool),
        }
        h.sessions[ctx.UserID] = session
    }

    session.LastSeen = time.Now()
    session.MessageCount++
    session.ChatsVisited[ctx.ChatID] = true

    // 记录命令使用
    if strings.HasPrefix(ctx.Text, "/") {
        cmd := strings.Fields(ctx.Text)[0]
        session.CommandsUsed[cmd]++
    }

    return nil
}

func (h *UserBehaviorTracker) Priority() int {
    return 955
}

func (h *UserBehaviorTracker) ContinueChain() bool {
    return true
}

// GetUserSession 获取用户会话
func (h *UserBehaviorTracker) GetUserSession(userID int64) *UserSession {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return h.sessions[userID]
}
```

### 场景 2：错误日志收集

```go
package listener

import (
    "telegram-bot/internal/handler"
)

type ErrorCollector struct {
    logger Logger
}

func NewErrorCollector(logger Logger) *ErrorCollector {
    return &ErrorCollector{
        logger: logger,
    }
}

func (h *ErrorCollector) Match(ctx *handler.Context) bool {
    return true
}

func (h *ErrorCollector) Handle(ctx *handler.Context) error {
    // 检查上下文中是否有错误标记
    if err, exists := ctx.Get("handler_error"); exists && err != nil {
        h.logger.Error("handler_error_occurred",
            "error", err,
            "user_id", ctx.UserID,
            "chat_id", ctx.ChatID,
            "text", ctx.Text,
            "handler", ctx.Get("current_handler"),
        )

        // 可选：发送错误通知到管理员
        // notifyAdmin(err)
    }

    return nil
}

func (h *ErrorCollector) Priority() int {
    return 990 // 最低优先级，确保捕获所有错误
}

func (h *ErrorCollector) ContinueChain() bool {
    return true
}
```

---

## 常见问题

### Q1：监听器会影响性能吗？

**影响很小**，但需要注意：

✅ **推荐做法**：
- 使用异步处理（goroutine）
- 避免在 `Handle()` 中执行耗时操作
- 使用 `sync.RWMutex` 保护共享数据
- 定期清理过期数据

❌ **避免**：
- 在 `Handle()` 中同步写入数据库
- 执行网络请求
- 复杂的计算

### Q2：监听器应该返回错误吗？

**不应该**。监听器的错误不应影响业务流程。

```go
func (h *Listener) Handle(ctx *handler.Context) error {
    defer func() {
        if r := recover(); r != nil {
            h.logger.Error("listener_panic", "panic", r)
        }
    }()

    // 内部捕获错误
    if err := h.doSomething(); err != nil {
        h.logger.Error("listener_error", "error", err)
        return nil // 不向上传播
    }

    return nil
}
```

### Q3：如何保证监听器的线程安全？

使用 `sync.RWMutex`：

```go
type SafeListener struct {
    data map[string]int
    mu   sync.RWMutex
}

func (h *SafeListener) Handle(ctx *handler.Context) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    h.data["key"]++

    return nil
}

// 读取时使用读锁
func (h *SafeListener) GetData(key string) int {
    h.mu.RLock()
    defer h.mu.RUnlock()

    return h.data[key]
}
```

### Q4：监听器如何持久化数据？

**推荐异步持久化**：

```go
func (h *Listener) Handle(ctx *handler.Context) error {
    // 异步保存
    go func() {
        if err := h.saveToDatabase(ctx); err != nil {
            h.logger.Error("save_failed", "error", err)
        }
    }()

    return nil
}
```

**批量持久化**（更高效）：

```go
type BatchListener struct {
    buffer chan *handler.Context
}

func NewBatchListener() *BatchListener {
    l := &BatchListener{
        buffer: make(chan *handler.Context, 100),
    }

    // 启动批量处理协程
    go l.batchProcessor()

    return l
}

func (h *BatchListener) Handle(ctx *handler.Context) error {
    select {
    case h.buffer <- ctx:
    default:
        // 缓冲区满，丢弃或记录
    }
    return nil
}

func (h *BatchListener) batchProcessor() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    batch := make([]*handler.Context, 0, 100)

    for {
        select {
        case ctx := <-h.buffer:
            batch = append(batch, ctx)

            if len(batch) >= 100 {
                h.saveBatch(batch)
                batch = batch[:0]
            }

        case <-ticker.C:
            if len(batch) > 0 {
                h.saveBatch(batch)
                batch = batch[:0]
            }
        }
    }
}
```

### Q5：如何导出监听器收集的数据？

**方式 1：HTTP API**

```go
func (h *AnalyticsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    stats := h.GetStats()
    json.NewEncoder(w).Encode(stats)
}

// 在 main.go 中注册
http.HandleFunc("/stats", analyticsHandler.ServeHTTP)
```

**方式 2：命令查询**

```go
type StatsCommand struct {
    analytics *AnalyticsHandler
}

func (h *StatsCommand) Handle(ctx *handler.Context) error {
    stats := h.analytics.GetStats()

    response := fmt.Sprintf(
        "📊 统计数据\n\n"+
            "总消息数: %d\n"+
            "活跃用户: %d\n"+
            "最后更新: %s",
        stats.TotalMessages,
        len(stats.ActiveUsers),
        stats.LastUpdated.Format("2006-01-02 15:04:05"),
    )

    return ctx.Reply(response)
}
```

### Q6：监听器的优先级如何选择？

| 优先级范围 | 用途 | 示例 |
|-----------|------|------|
| 900-909 | 关键日志 | MessageLogger |
| 910-919 | 安全监控 | SecurityMonitor |
| 920-929 | 审计日志 | AuditLogger |
| 930-949 | 业务统计 | UserBehavior |
| 950-969 | 数据分析 | Analytics |
| 970-989 | 性能监控 | PerformanceMonitor |
| 990-999 | 兜底处理 | ErrorCollector |

**原则**：重要的先执行（数字小）

---

## 附录

### 相关资源

- [项目内置示例 - MessageLogger](../../internal/handlers/listener/message_logger.go)
- [项目内置示例 - Analytics](../../internal/handlers/listener/analytics.go)
- [Go sync 包文档](https://pkg.go.dev/sync)

### 相关文档

- [命令处理器开发指南](./command-handler-guide.md)
- [关键词处理器开发指南](./keyword-handler-guide.md)
- [正则匹配处理器开发指南](./pattern-handler-guide.md)
- [架构总览](../../CLAUDE.md)

---

**编写日期**: 2025-10-02
**文档版本**: v1.0
**维护者**: Telegram Bot Development Team
