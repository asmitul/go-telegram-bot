# 中间件开发指南

## 📚 目录

- [概述](#概述)
- [核心概念](#核心概念)
- [快速开始](#快速开始)
- [内置中间件详解](#内置中间件详解)
- [完整代码示例](#完整代码示例)
- [中间件链执行原理](#中间件链执行原理)
- [实际场景示例](#实际场景示例)
- [最佳实践](#最佳实践)
- [测试方法](#测试方法)
- [常见问题](#常见问题)

---

## 概述

**中间件** (Middleware) 是包装在处理器外层的函数，用于在消息处理前后添加通用逻辑。中间件采用洋葱模型，从外到内逐层执行。

### 适用场景

- ✅ 错误恢复（捕获 panic）
- ✅ 日志记录（请求/响应日志）
- ✅ 权限验证（自动加载用户）
- ✅ 限流控制（防止滥用）
- ✅ 性能监控（记录执行时间）
- ✅ 请求追踪（分布式追踪）
- ✅ 数据转换（请求/响应格式化）
- ✅ 缓存控制

### 核心优势

- 🔄 **代码复用**：通用逻辑只写一次
- 🎯 **关注点分离**：业务逻辑与横切关注点分离
- 🔗 **灵活组合**：可以任意组合中间件
- 📊 **统一处理**：所有处理器自动应用

---

## 核心概念

### 中间件接口

```go
// Middleware 中间件函数类型
type Middleware func(next HandlerFunc) HandlerFunc

// HandlerFunc 处理器函数类型
type HandlerFunc func(ctx *Context) error
```

### 洋葱模型

```
Request
   ↓
┌──────────────────────────────────┐
│  Recovery Middleware              │
│  ┌────────────────────────────┐  │
│  │  Logging Middleware        │  │
│  │  ┌──────────────────────┐  │  │
│  │  │  Permission MW       │  │  │
│  │  │  ┌────────────────┐  │  │  │
│  │  │  │    Handler     │  │  │  │
│  │  │  └────────────────┘  │  │  │
│  │  │         ↓            │  │  │
│  │  └──────────────────────┘  │  │
│  │           ↓                │  │
│  └────────────────────────────┘  │
│                ↓                 │
└──────────────────────────────────┘
   ↓
Response
```

### 执行流程

```
1. RecoveryMiddleware (开始)
2. LoggingMiddleware (开始)
3. PermissionMiddleware (开始)
4. Handler (执行业务逻辑)
5. PermissionMiddleware (结束)
6. LoggingMiddleware (结束 - 记录日志)
7. RecoveryMiddleware (结束)
```

---

## 快速开始

### 步骤 1：创建中间件文件

在 `internal/middleware/` 目录下创建新文件，例如 `timing.go`：

```bash
touch internal/middleware/timing.go
```

### 步骤 2：实现中间件

```go
package middleware

import (
	"telegram-bot/internal/handler"
	"time"
)

// TimingMiddleware 计时中间件
type TimingMiddleware struct {
	logger Logger
}

func NewTimingMiddleware(logger Logger) *TimingMiddleware {
	return &TimingMiddleware{logger: logger}
}

func (m *TimingMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			// 前置处理：记录开始时间
			start := time.Now()

			// 执行下一个处理器
			err := next(ctx)

			// 后置处理：计算耗时
			duration := time.Since(start)
			m.logger.Info("handler_timing",
				"duration_ms", duration.Milliseconds(),
				"user_id", ctx.UserID,
			)

			return err
		}
	}
}
```

### 步骤 3：注册中间件

在 `cmd/bot/main.go` 中注册：

```go
// 6. 注册全局中间件（按执行顺序）
router.Use(middleware.NewRecoveryMiddleware(appLogger).Middleware())
router.Use(middleware.NewLoggingMiddleware(appLogger).Middleware())
router.Use(middleware.NewPermissionMiddleware(userRepo).Middleware())
router.Use(middleware.NewTimingMiddleware(appLogger).Middleware())  // 新增
```

### 步骤 4：测试

发送任意消息到机器人，查看日志中的 `handler_timing` 条目。

---

## 内置中间件详解

项目内置了 4 个核心中间件：

### 1. RecoveryMiddleware（错误恢复）

**作用**：捕获处理器中的 panic，防止程序崩溃。

**源码位置**：`internal/middleware/recovery.go`

**关键逻辑**：
```go
func (m *RecoveryMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					// 记录 panic 信息和堆栈
					m.logger.Error("panic_recovered",
						"panic", r,
						"stack", string(debug.Stack()),
					)

					// 转换为 error
					err = fmt.Errorf("internal error: %v", r)

					// 通知用户
					ctx.Reply("❌ 服务器内部错误，请稍后再试")
				}
			}()

			return next(ctx)
		}
	}
}
```

**使用场景**：
- ✅ 必须放在最外层（第一个注册）
- ✅ 捕获所有未处理的 panic
- ✅ 记录详细的堆栈信息
- ✅ 提供友好的错误提示

### 2. LoggingMiddleware（日志记录）

**作用**：记录所有消息的处理情况。

**源码位置**：`internal/middleware/logging.go`

**关键逻辑**：
```go
func (m *LoggingMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			start := time.Now()

			// 记录接收到的消息
			m.logger.Info("message_received",
				"chat_type", ctx.ChatType,
				"user_id", ctx.UserID,
				"text", ctx.Text,
			)

			err := next(ctx)

			duration := time.Since(start)

			// 记录处理结果
			if err != nil {
				m.logger.Error("handler_error", "error", err, "duration_ms", duration.Milliseconds())
			} else {
				m.logger.Info("handler_success", "duration_ms", duration.Milliseconds())
			}

			return err
		}
	}
}
```

**日志输出示例**：
```json
{"level":"info","msg":"message_received","chat_type":"private","user_id":123456789,"text":"/ping"}
{"level":"info","msg":"handler_success","duration_ms":5}
```

### 3. PermissionMiddleware（权限管理）

**作用**：自动加载用户信息并注入到上下文。

**源码位置**：`internal/middleware/permission.go`

**关键逻辑**：
```go
func (m *PermissionMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			// 1. 加载用户
			u, err := m.userRepo.FindByID(ctx.UserID)
			if err != nil {
				// 用户不存在，创建新用户
				u = user.NewUser(ctx.UserID, ctx.Username, ctx.FirstName, ctx.LastName)
				m.userRepo.Save(u)
			}

			// 2. 注入到上下文
			ctx.User = u

			// 3. 执行下一个处理器
			return next(ctx)
		}
	}
}
```

**特点**：
- ✅ 自动创建新用户
- ✅ 默认权限为普通用户
- ✅ 处理器可以直接使用 `ctx.User`
- ✅ 权限检查由处理器自己执行

### 4. RateLimitMiddleware（限流控制）

**作用**：防止用户频繁发送消息。

**源码位置**：`internal/middleware/ratelimit.go`

**关键逻辑**：
```go
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
```

**令牌桶算法**：
```go
limiter := middleware.NewSimpleRateLimiter(
	time.Second, // 每秒恢复 1 个令牌
	5,           // 令牌桶容量为 5（允许突发 5 条消息）
)
router.Use(middleware.NewRateLimitMiddleware(limiter).Middleware())
```

---

## 完整代码示例

### 示例 1：请求追踪中间件

```go
package middleware

import (
	"fmt"
	"math/rand"
	"telegram-bot/internal/handler"
)

// TracingMiddleware 请求追踪中间件
type TracingMiddleware struct {
	logger Logger
}

func NewTracingMiddleware(logger Logger) *TracingMiddleware {
	return &TracingMiddleware{logger: logger}
}

func (m *TracingMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			// 生成请求 ID
			requestID := fmt.Sprintf("%d-%d", ctx.UserID, rand.Int63())

			// 存储到上下文（供后续处理器使用）
			ctx.Set("request_id", requestID)

			m.logger.Info("request_start",
				"request_id", requestID,
				"user_id", ctx.UserID,
				"text", ctx.Text,
			)

			err := next(ctx)

			m.logger.Info("request_end",
				"request_id", requestID,
				"success", err == nil,
			)

			return err
		}
	}
}
```

### 示例 2：缓存中间件

```go
package middleware

import (
	"crypto/md5"
	"fmt"
	"sync"
	"telegram-bot/internal/handler"
	"time"
)

// CacheMiddleware 缓存中间件
type CacheMiddleware struct {
	cache map[string]cacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

func NewCacheMiddleware(ttl time.Duration) *CacheMiddleware {
	return &CacheMiddleware{
		cache: make(map[string]cacheEntry),
		ttl:   ttl,
	}
}

func (m *CacheMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			// 生成缓存键
			key := m.generateKey(ctx)

			// 检查缓存
			if cached, ok := m.get(key); ok {
				ctx.Reply(cached.(string))
				return nil
			}

			// 执行处理器
			err := next(ctx)

			// 缓存结果（这里简化处理，实际应该缓存响应内容）
			if err == nil {
				m.set(key, "cached_response")
			}

			return err
		}
	}
}

func (m *CacheMiddleware) generateKey(ctx *handler.Context) string {
	data := fmt.Sprintf("%d:%s", ctx.UserID, ctx.Text)
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}

func (m *CacheMiddleware) get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.cache[key]
	if !ok || time.Now().After(entry.expiration) {
		return nil, false
	}

	return entry.value, true
}

func (m *CacheMiddleware) set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache[key] = cacheEntry{
		value:      value,
		expiration: time.Now().Add(m.ttl),
	}
}
```

### 示例 3：认证中间件

```go
package middleware

import (
	"fmt"
	"telegram-bot/internal/handler"
)

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	allowedUsers map[int64]bool
}

func NewAuthMiddleware(allowedUsers []int64) *AuthMiddleware {
	allowed := make(map[int64]bool)
	for _, uid := range allowedUsers {
		allowed[uid] = true
	}

	return &AuthMiddleware{
		allowedUsers: allowed,
	}
}

func (m *AuthMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			// 检查用户是否在白名单中
			if !m.allowedUsers[ctx.UserID] {
				return fmt.Errorf("❌ 未授权访问")
			}

			return next(ctx)
		}
	}
}
```

---

## 中间件链执行原理

### 构建过程

```go
// Router 中的 buildChain 方法
func (r *Router) buildChain(h Handler) HandlerFunc {
	// 最终处理器
	final := func(ctx *Context) error {
		return h.Handle(ctx)
	}

	// 从后向前包装中间件
	wrapped := final
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		wrapped = r.middlewares[i](wrapped)
	}

	return wrapped
}
```

### 执行示例

假设注册了 3 个中间件：

```go
router.Use(mw1.Middleware())  // Recovery
router.Use(mw2.Middleware())  // Logging
router.Use(mw3.Middleware())  // Permission
```

**构建过程**：
```
final = h.Handle

step 1: wrapped = mw3(final)      // Permission(Handler)
step 2: wrapped = mw2(wrapped)    // Logging(Permission(Handler))
step 3: wrapped = mw1(wrapped)    // Recovery(Logging(Permission(Handler)))
```

**执行顺序**：
```
mw1 开始 → mw2 开始 → mw3 开始 → Handler → mw3 结束 → mw2 结束 → mw1 结束
```

---

## 实际场景示例

### 场景 1：IP 黑名单中间件

```go
package middleware

import (
	"fmt"
	"telegram-bot/internal/handler"
)

type IPBlacklistMiddleware struct {
	blacklist map[int64]bool
	logger    Logger
}

func NewIPBlacklistMiddleware(logger Logger) *IPBlacklistMiddleware {
	return &IPBlacklistMiddleware{
		blacklist: make(map[int64]bool),
		logger:    logger,
	}
}

func (m *IPBlacklistMiddleware) BlockUser(userID int64) {
	m.blacklist[userID] = true
	m.logger.Info("user_blocked", "user_id", userID)
}

func (m *IPBlacklistMiddleware) UnblockUser(userID int64) {
	delete(m.blacklist, userID)
	m.logger.Info("user_unblocked", "user_id", userID)
}

func (m *IPBlacklistMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			if m.blacklist[ctx.UserID] {
				m.logger.Warn("blocked_user_attempt", "user_id", ctx.UserID)
				return fmt.Errorf("🚫 你已被封禁")
			}

			return next(ctx)
		}
	}
}
```

### 场景 2：维护模式中间件

```go
package middleware

import (
	"fmt"
	"telegram-bot/internal/handler"
)

type MaintenanceMiddleware struct {
	enabled      bool
	allowedUsers map[int64]bool
	message      string
}

func NewMaintenanceMiddleware(allowedAdmins []int64) *MaintenanceMiddleware {
	allowed := make(map[int64]bool)
	for _, uid := range allowedAdmins {
		allowed[uid] = true
	}

	return &MaintenanceMiddleware{
		enabled:      false,
		allowedUsers: allowed,
		message:      "🔧 系统维护中，请稍后再试",
	}
}

func (m *MaintenanceMiddleware) Enable(message string) {
	m.enabled = true
	if message != "" {
		m.message = message
	}
}

func (m *MaintenanceMiddleware) Disable() {
	m.enabled = false
}

func (m *MaintenanceMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			// 维护模式下，只允许管理员使用
			if m.enabled && !m.allowedUsers[ctx.UserID] {
				return fmt.Errorf(m.message)
			}

			return next(ctx)
		}
	}
}
```

### 场景 3：语言切换中间件

```go
package middleware

import (
	"telegram-bot/internal/handler"
)

type I18nMiddleware struct {
	defaultLang string
}

func NewI18nMiddleware(defaultLang string) *I18nMiddleware {
	return &I18nMiddleware{defaultLang: defaultLang}
}

func (m *I18nMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			// 设置语言（优先使用用户语言，否则使用默认语言）
			lang := ctx.LanguageCode
			if lang == "" {
				lang = m.defaultLang
			}

			ctx.Set("lang", lang)

			return next(ctx)
		}
	}
}
```

---

## 最佳实践

### 1. 中间件顺序很重要

```go
// ✅ 推荐顺序
router.Use(NewRecoveryMiddleware(logger).Middleware())      // 1. 最外层：捕获 panic
router.Use(NewLoggingMiddleware(logger).Middleware())       // 2. 记录日志
router.Use(NewPermissionMiddleware(userRepo).Middleware())  // 3. 加载用户
router.Use(NewRateLimitMiddleware(limiter).Middleware())    // 4. 限流
router.Use(NewAuthMiddleware(admins).Middleware())          // 5. 认证

// ❌ 错误顺序
router.Use(NewPermissionMiddleware(userRepo).Middleware())  // Permission 在 Recovery 之前
router.Use(NewRecoveryMiddleware(logger).Middleware())      // 如果 Permission panic，无法捕获
```

### 2. 避免过度使用

```go
// ❌ 避免：中间件过多影响性能
router.Use(mw1.Middleware())
router.Use(mw2.Middleware())
router.Use(mw3.Middleware())
// ... 10+ 个中间件

// ✅ 推荐：合并相关功能
router.Use(NewCoreMiddleware(logger, userRepo).Middleware())
```

### 3. 使用 Context 存储数据

```go
func (m *MyMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			// ✅ 存储数据到 Context
			ctx.Set("start_time", time.Now())
			ctx.Set("request_id", generateID())

			err := next(ctx)

			// ✅ 从 Context 读取数据
			startTime, _ := ctx.Get("start_time")
			duration := time.Since(startTime.(time.Time))

			return err
		}
	}
}
```

### 4. 错误处理

```go
func (m *MyMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			// ✅ 捕获并处理错误
			err := next(ctx)

			if err != nil {
				m.logger.Error("middleware_error", "error", err)

				// 可以选择：
				// 1. 返回错误（传播）
				return err

				// 2. 转换错误
				// return fmt.Errorf("wrapped: %w", err)

				// 3. 吞掉错误（谨慎使用）
				// return nil
			}

			return nil
		}
	}
}
```

### 5. 线程安全

```go
type SafeMiddleware struct {
	counter int64  // ❌ 非线程安全
	mu      sync.Mutex
}

// ✅ 使用原子操作或互斥锁
func (m *SafeMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			atomic.AddInt64(&m.counter, 1)  // ✅ 原子操作

			// 或使用互斥锁
			m.mu.Lock()
			m.counter++
			m.mu.Unlock()

			return next(ctx)
		}
	}
}
```

---

## 测试方法

### 1. 单元测试

```go
package middleware

import (
	"errors"
	"testing"
	"telegram-bot/internal/handler"
	"github.com/stretchr/testify/assert"
)

func TestRecoveryMiddleware(t *testing.T) {
	logger := &MockLogger{}
	mw := NewRecoveryMiddleware(logger)

	// 模拟 panic 的处理器
	panicHandler := func(ctx *handler.Context) error {
		panic("test panic")
	}

	// 应用中间件
	wrapped := mw.Middleware()(panicHandler)

	ctx := &handler.Context{}
	err := wrapped(ctx)

	// 验证：panic 被捕获并转换为 error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal error")
}

func TestLoggingMiddleware(t *testing.T) {
	logger := &MockLogger{}
	mw := NewLoggingMiddleware(logger)

	// 成功的处理器
	successHandler := func(ctx *handler.Context) error {
		return nil
	}

	wrapped := mw.Middleware()(successHandler)

	ctx := &handler.Context{UserID: 123}
	err := wrapped(ctx)

	assert.NoError(t, err)
	// 验证日志被调用
	assert.True(t, logger.infoCalled)
}
```

### 2. 集成测试

```go
func TestMiddlewareChain(t *testing.T) {
	router := handler.NewRouter()

	// 注册中间件
	router.Use(NewRecoveryMiddleware(logger).Middleware())
	router.Use(NewLoggingMiddleware(logger).Middleware())

	// 注册处理器
	router.Register(&TestHandler{})

	// 执行路由
	ctx := &handler.Context{Text: "/test"}
	err := router.Route(ctx)

	assert.NoError(t, err)
}
```

---

## 常见问题

### Q1：中间件和处理器的区别？

| 特性 | 中间件 | 处理器 |
|------|-------|--------|
| **作用对象** | 所有处理器 | 单个消息 |
| **执行时机** | 每个处理器执行前后 | 消息匹配时 |
| **典型用途** | 日志、认证、限流 | 业务逻辑 |
| **注册方式** | `router.Use()` | `router.Register()` |

### Q2：中间件会影响性能吗？

轻量级中间件（如日志、权限）影响很小（通常 < 1ms）。避免在中间件中执行：
- ❌ 复杂的数据库查询
- ❌ 外部 API 调用
- ❌ 大量计算

### Q3：如何为特定处理器禁用中间件？

目前框架不支持，但可以通过条件判断实现：

```go
func (m *MyMiddleware) Middleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx *handler.Context) error {
			// 特定条件下跳过
			if ctx.Get("skip_middleware") == true {
				return next(ctx)
			}

			// 正常执行中间件逻辑
			return next(ctx)
		}
	}
}
```

### Q4：中间件可以修改 Context 吗？

可以，且推荐使用 `ctx.Set()` 存储数据：

```go
ctx.Set("user_role", "admin")
ctx.Set("request_id", "abc123")
```

### Q5：中间件执行顺序如何确定？

按照 `router.Use()` 的**注册顺序**执行（先注册先执行）。

---

## 附录

### 相关资源

- [Recovery 源码](../internal/middleware/recovery.go)
- [Logging 源码](../internal/middleware/logging.go)
- [Permission 源码](../internal/middleware/permission.go)
- [RateLimit 源码](../internal/middleware/ratelimit.go)
- [Router 源码](../internal/handler/router.go)

### 相关文档

- [项目快速入门](./getting-started.md)
- [命令处理器开发指南](./handlers/command-handler-guide.md)
- [架构总览](../CLAUDE.md)

---

**最后更新**: 2025-10-03
**文档版本**: v1.0
**维护者**: Telegram Bot Development Team
