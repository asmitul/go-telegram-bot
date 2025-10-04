# Telegram Bot 架构设计文档

## 目录

- [系统概述](#系统概述)
- [架构原则](#架构原则)
- [核心架构](#核心架构)
- [目录结构](#目录结构)
- [核心组件](#核心组件)
- [消息处理流程](#消息处理流程)
- [数据流图](#数据流图)
- [技术栈](#技术栈)
- [扩展指南](#扩展指南)
- [最佳实践](#最佳实践)
- [部署架构](#部署架构)

---

## 系统概述

本项目是一个基于 **Handler 架构** 设计的生产级 Telegram 群组管理机器人，采用 Go 1.25+ 开发。

### 核心特性

- ✅ **统一消息处理**: Handler 接口统一所有消息处理逻辑
- ✅ **灵活的路由系统**: 基于优先级的消息路由
- ✅ **完善的中间件**: 错误恢复、日志、权限、限流
- ✅ **插件式处理器**: 命令、关键词、正则、监听器
- ✅ **权限系统**: 4 级权限，按群组隔离
- ✅ **可测试性**: 接口驱动，易于模拟和测试

### 系统目标

1. **简单易用**: 清晰的接口和文档，快速上手
2. **灵活扩展**: 插件式架构，轻松添加新功能
3. **生产可用**: 错误处理、日志、监控完善
4. **高性能**: 优化的 MongoDB 索引，连接池

---

## 架构原则

### 1. Interface-Driven Design（接口驱动）

所有处理器都实现统一的 `Handler` 接口：

```go
type Handler interface {
    Match(ctx *Context) bool      // 匹配逻辑
    Handle(ctx *Context) error    // 处理逻辑
    Priority() int                // 优先级
    ContinueChain() bool          // 是否继续链
}
```

### 2. 关注点分离

- **Router**: 负责消息路由和分发
- **Handler**: 负责具体的业务逻辑
- **Middleware**: 负责横切关注点（日志、权限等）
- **Domain**: 负责业务规则和实体
- **Adapter**: 负责外部系统集成

### 3. 组合优于继承

使用 BaseCommand 作为可组合的基础组件，而非强制继承：

```go
type MyHandler struct {
    *BaseCommand  // 组合 BaseCommand
    // 自定义字段
}
```

---

## 核心架构

### 整体架构图

```
┌─────────────────────────────────────────────────────────┐
│                    Telegram Update                       │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│           Converter (Update → Context)                   │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│                   Router.Route()                         │
│  • 获取所有处理器                                           │
│  • 按优先级排序                                             │
│  • 执行匹配的处理器                                          │
└──────────────────────┬──────────────────────────────────┘
                       │
           ┌───────────┴───────────┐
           │                       │
           ▼                       ▼
      Match(ctx)?            ContinueChain()?
           │                       │
           ├─ Yes                 Yes → 下一个 Handler
           │                      No  → 停止
           ▼
    ┌──────────────┐
    │  Middleware  │
    │   Recovery   │
    │   Logging    │
    │  Permission  │
    └──────┬───────┘
           │
           ▼
      Handle(ctx)
```

### 三层架构

```
┌─────────────────────────────────────────┐
│         Framework Layer                  │  框架层
│  - handler/                              │  - Handler 接口
│  - Router                                │  - Router 路由器
│  - Middleware                            │  - 中间件系统
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│        Handlers Layer                    │  处理器层
│  - handlers/command/                     │  - 命令处理器
│  - handlers/keyword/                     │  - 关键词处理器
│  - handlers/pattern/                     │  - 正则处理器
│  - handlers/listener/                    │  - 监听器
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│      Infrastructure Layer                │  基础设施层
│  - domain/           (业务实体)           │  - User, Group 等
│  - adapter/          (外部集成)           │  - MongoDB, Telegram
│  - middleware/       (横切关注点)          │  - 日志、权限等
└─────────────────────────────────────────┘
```

---

## 目录结构

```
telegram-bot/
├── cmd/
│   └── bot/
│       └── main.go              # 应用入口
│
├── internal/
│   ├── handler/                 # 核心框架层
│   │   ├── handler.go           # Handler 接口定义
│   │   ├── context.go           # 消息上下文
│   │   ├── router.go            # 消息路由器
│   │   └── middleware.go        # 中间件基础
│   │
│   ├── handlers/                # 处理器实现层
│   │   ├── command/             # 命令处理器 (Priority: 100-199)
│   │   │   ├── base.go          # BaseCommand 基类
│   │   │   ├── ping.go          # /ping 命令
│   │   │   ├── help.go          # /help 命令
│   │   │   ├── stats.go         # /stats 命令
│   │   │   ├── promote.go       # /promote 提升权限
│   │   │   ├── demote.go        # /demote 降低权限
│   │   │   ├── setperm.go       # /setperm 设置权限
│   │   │   ├── listadmins.go    # /listadmins 管理员列表
│   │   │   └── myperm.go        # /myperm 查看权限
│   │   │
│   │   ├── keyword/             # 关键词处理器 (Priority: 200-299)
│   │   │   └── greeting.go      # 问候语处理
│   │   │
│   │   ├── pattern/             # 正则处理器 (Priority: 300-399)
│   │   │   └── weather.go       # 天气查询（正则匹配）
│   │   │
│   │   └── listener/            # 监听器 (Priority: 900-999)
│   │       ├── message_logger.go # 消息日志
│   │       └── analytics.go      # 数据分析
│   │
│   ├── middleware/              # 中间件层
│   │   ├── recovery.go          # 错误恢复（捕获 panic）
│   │   ├── logging.go           # 日志记录
│   │   ├── permission.go        # 权限管理（自动加载用户）
│   │   └── ratelimit.go         # 限流控制（令牌桶）
│   │
│   ├── domain/                  # 领域层（业务实体）
│   │   ├── user/                # 用户聚合根
│   │   │   ├── user.go          # User 实体
│   │   │   └── permission.go    # 权限枚举
│   │   └── group/               # 群组聚合根
│   │       ├── group.go         # Group 实体
│   │       └── command_config.go # 命令配置
│   │
│   ├── adapter/                 # 适配器层（外部集成）
│   │   ├── telegram/            # Telegram API 适配
│   │   │   ├── converter.go     # Update → Context 转换
│   │   │   └── api.go           # Telegram API 封装
│   │   └── repository/          # 数据持久化
│   │       └── mongodb/         # MongoDB 实现
│   │           ├── user_repository.go
│   │           ├── group_repository.go
│   │           └── indexes.go   # 索引管理
│   │
│   ├── config/                  # 配置管理
│   │   └── config.go
│   │
│   └── scheduler/               # 定时任务
│       ├── scheduler.go
│       └── jobs.go
│
├── pkg/                         # 公共包
│   ├── logger/                  # 结构化日志
│   ├── errors/                  # 错误处理
│   └── validator/               # 数据验证
│
├── test/                        # 测试
│   ├── mocks/                   # Mock 对象（gomock）
│   └── integration/             # 集成测试
│
├── docs/                        # 文档
│   ├── architecture.md          # 架构文档（本文件）
│   ├── getting-started.md       # 快速入门
│   ├── developer-api.md         # API 参考
│   ├── handlers/                # 处理器开发指南
│   │   ├── command-handler-guide.md
│   │   ├── keyword-handler-guide.md
│   │   ├── pattern-handler-guide.md
│   │   └── listener-handler-guide.md
│   └── middleware-guide.md      # 中间件开发
│
└── deployments/                 # 部署相关
    └── docker/
        ├── Dockerfile
        └── docker-compose.yml
```

---

## 核心组件

### 1. Handler Interface（处理器接口）

所有消息处理器的核心接口：

```go
type Handler interface {
    // Match 判断是否应该处理这条消息
    Match(ctx *Context) bool

    // Handle 处理消息
    Handle(ctx *Context) error

    // Priority 优先级（0-999，数字越小越优先）
    Priority() int

    // ContinueChain 处理后是否继续执行后续处理器
    ContinueChain() bool
}
```

**职责**：
- ✅ 定义统一的消息处理规范
- ✅ 支持优先级排序
- ✅ 控制处理链的执行流程

### 2. Context（消息上下文）

封装所有消息处理所需的信息：

```go
type Context struct {
    // 原始对象
    Ctx     context.Context
    Bot     *bot.Bot
    Update  *models.Update
    Message *models.Message

    // 聊天信息
    ChatType  string  // "private", "group", "supergroup", "channel"
    ChatID    int64
    ChatTitle string

    // 用户信息
    UserID    int64
    Username  string
    FirstName string
    LastName  string
    User      *user.User  // 数据库用户对象

    // 消息内容
    Text      string
    MessageID int

    // 回复消息
    ReplyTo *ReplyInfo

    // 上下文存储
    values map[string]interface{}
}
```

**提供的方法**：
- `IsPrivate()`, `IsGroup()`, `IsChannel()` - 聊天类型判断
- `Reply()`, `ReplyHTML()`, `Send()` - 消息发送
- `HasPermission()`, `RequirePermission()` - 权限检查
- `Set()`, `Get()` - 上下文数据存储

### 3. Router（路由器）

负责消息的路由和分发：

**核心方法**：
```go
func (r *Router) Register(h Handler)          // 注册处理器
func (r *Router) Use(mw Middleware)            // 注册中间件
func (r *Router) Route(ctx *Context) error     // 路由消息
```

**执行流程**：
1. 获取所有已注册的处理器
2. 按优先级排序（数字越小越优先）
3. 遍历处理器，调用 `Match()` 检查是否匹配
4. 匹配成功时，构建中间件链并执行 `Handle()`
5. 检查 `ContinueChain()`，决定是否继续执行下一个处理器

### 4. Middleware（中间件）

横切关注点的实现：

```go
type Middleware func(HandlerFunc) HandlerFunc
type HandlerFunc func(*Context) error
```

**内置中间件**：
- **RecoveryMiddleware**: 捕获 panic，防止程序崩溃
- **LoggingMiddleware**: 记录消息处理日志
- **PermissionMiddleware**: 自动加载用户信息
- **RateLimitMiddleware**: 令牌桶限流

**执行顺序**（洋葱模型）：
```
Request
  → Recovery (开始)
    → Logging (开始)
      → Permission (开始)
        → Handler (执行)
      ← Permission (结束)
    ← Logging (结束，记录日志)
  ← Recovery (结束)
Response
```

### 5. Handler Types（处理器类型）

#### 命令处理器 (Priority: 100-199)

处理以 `/` 开头的命令：

```go
type MyCommandHandler struct {
    *BaseCommand
}

func NewMyCommandHandler(groupRepo GroupRepository) *MyCommandHandler {
    return &MyCommandHandler{
        BaseCommand: NewBaseCommand(
            "mycommand",              // 命令名
            "命令描述",                // 描述
            user.PermissionUser,      // 所需权限
            []string{"private"},      // 支持的聊天类型
            groupRepo,
        ),
    }
}

func (h *MyCommandHandler) Handle(ctx *handler.Context) error {
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }
    return ctx.Reply("执行成功")
}
```

**特点**：
- ✅ BaseCommand 自动处理命令匹配、聊天类型过滤、群组启用检查
- ✅ 内置权限检查支持
- ✅ 支持 `@botname` 后缀
- ✅ 默认 `ContinueChain() = false`

#### 关键词处理器 (Priority: 200-299)

匹配包含特定关键词的消息：

```go
type KeywordHandler struct {
    keywords []string
}

func (h *KeywordHandler) Match(ctx *handler.Context) bool {
    text := strings.ToLower(ctx.Text)
    for _, kw := range h.keywords {
        if strings.Contains(text, kw) {
            return true
        }
    }
    return false
}

func (h *KeywordHandler) Handle(ctx *handler.Context) error {
    return ctx.Reply("检测到关键词")
}

func (h *KeywordHandler) Priority() int { return 200 }
func (h *KeywordHandler) ContinueChain() bool { return true }
```

#### 正则处理器 (Priority: 300-399)

使用正则表达式匹配复杂模式：

```go
type PatternHandler struct {
    pattern *regexp.Regexp
}

func (h *PatternHandler) Match(ctx *handler.Context) bool {
    return h.pattern.MatchString(ctx.Text)
}

func (h *PatternHandler) Handle(ctx *handler.Context) error {
    matches := h.pattern.FindStringSubmatch(ctx.Text)
    // 处理匹配结果
    return ctx.Reply("匹配成功")
}

func (h *PatternHandler) Priority() int { return 300 }
func (h *PatternHandler) ContinueChain() bool { return false }
```

#### 监听器 (Priority: 900-999)

监听所有消息，用于日志、分析等：

```go
type ListenerHandler struct {
    logger Logger
}

func (h *ListenerHandler) Match(ctx *handler.Context) bool {
    return true  // 匹配所有消息
}

func (h *ListenerHandler) Handle(ctx *handler.Context) error {
    h.logger.Info("message_received", "text", ctx.Text)
    return nil
}

func (h *ListenerHandler) Priority() int { return 900 }
func (h *ListenerHandler) ContinueChain() bool { return true }
```

---

## 消息处理流程

### 完整流程图

```
┌─────────────┐
│  Telegram   │
│   Update    │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│  ConvertUpdate  │  telegram.ConvertUpdate()
│  (converter.go) │  创建 Handler Context
└──────┬──────────┘
       │
       ▼
┌─────────────────┐
│ Router.Route()  │  路由消息
└──────┬──────────┘
       │
       ▼
┌──────────────────────────────────┐
│ 遍历所有 Handler (按优先级)        │
└──────┬───────────────────────────┘
       │
       ▼
┌─────────────────┐
│  h.Match(ctx)?  │──No─→ 下一个 Handler
└──────┬──────────┘
       │ Yes
       ▼
┌─────────────────┐
│ 构建中间件链     │
│ Recovery        │
│ Logging         │
│ Permission      │
└──────┬──────────┘
       │
       ▼
┌─────────────────┐
│  h.Handle(ctx)  │  执行处理逻辑
└──────┬──────────┘
       │
       ▼
┌──────────────────┐
│ ContinueChain()? │──No─→ 停止
└──────┬───────────┘
       │ Yes
       ▼
    下一个 Handler
```

### 示例执行序列

假设注册了以下处理器：
1. PingHandler (Priority: 100, ContinueChain: false)
2. GreetingHandler (Priority: 200, ContinueChain: true)
3. MessageLogger (Priority: 900, ContinueChain: true)

**场景 1: 用户发送 `/ping`**

```
1. PingHandler.Match("/ping") → true
2. 执行中间件链 → PingHandler.Handle() → 发送 "Pong!"
3. PingHandler.ContinueChain() → false
4. 停止执行，不继续检查后续处理器
```

**场景 2: 用户发送 "你好"**

```
1. PingHandler.Match("你好") → false，跳过
2. GreetingHandler.Match("你好") → true
3. 执行中间件链 → GreetingHandler.Handle() → 发送 "你好！"
4. GreetingHandler.ContinueChain() → true，继续
5. MessageLogger.Match("你好") → true
6. 执行中间件链 → MessageLogger.Handle() → 记录日志
7. MessageLogger.ContinueChain() → true，但已无后续处理器
8. 结束
```

---

## 数据流图

### 权限检查流程

```
┌──────────────┐
│ User Request │
└──────┬───────┘
       │
       ▼
┌────────────────────┐       No      ┌──────────┐
│ PermissionMW       │───────────────→│  创建新   │
│ 用户是否存在？      │                │  用户     │
└──────┬─────────────┘                └────┬─────┘
       │ Yes                                │
       │←───────────────────────────────────┘
       ▼
┌────────────────────┐
│ ctx.User 注入完成  │
└──────┬─────────────┘
       │
       ▼
┌────────────────────┐       No      ┌──────────┐
│ Handler 执行        │               │  返回    │
│ CheckPermission()  │───────────────→│  错误    │
└──────┬─────────────┘                └──────────┘
       │ Pass
       ▼
┌────────────────────┐
│ 执行业务逻辑        │
└────────────────────┘
```

### 群组命令启用检查

```
┌────────────────────┐
│ 收到命令 /mycommand│
└──────┬─────────────┘
       │
       ▼
┌────────────────────┐       No      ┌──────────┐
│ 是否在群组中？      │───────────────→│  跳过    │
└──────┬─────────────┘                │  检查    │
       │ Yes                           └────┬─────┘
       │                                    │
       ▼                                    │
┌────────────────────┐       No      ┌─────▼─────┐
│ 查询群组配置        │───────────────→│  命令     │
│ IsCommandEnabled() │                │  已禁用   │
└──────┬─────────────┘                └───────────┘
       │ Enabled
       ▼
┌────────────────────┐
│ 执行命令处理器      │
└────────────────────┘
```

---

## 技术栈

### 核心框架

| 组件 | 技术 | 版本 | 用途 |
|-----|------|-----|------|
| 语言 | Go | 1.25+ | 主开发语言 |
| Bot SDK | go-telegram/bot | latest | Telegram Bot API 客户端 |
| 数据库 | MongoDB Atlas | 云数据库 | 数据持久化（支持免费套餐）|

### 关键依赖

```go
require (
    github.com/go-telegram/bot v1.17.0       // Telegram Bot API
    go.mongodb.org/mongo-driver v1.13.1      // MongoDB 驱动
    github.com/joho/godotenv v1.5.1          // 环境变量加载
    github.com/stretchr/testify v1.11.1      // 测试框架
    go.uber.org/mock v0.6.0                  // Mock 生成
)
```

### 工具链

- **构建**: `make` (Makefile 提供各种命令)
- **测试**: `go test` + testify + gomock
- **部署**: Docker + Docker Compose
- **CI/CD**: GitHub Actions

---

## 扩展指南

### 1. 添加新命令

**步骤**:

1. 创建命令文件 `internal/handlers/command/mycommand.go`
2. 实现处理器
3. 在 `cmd/bot/main.go` 中注册

**示例**:

```go
// internal/handlers/command/version.go
package command

import (
    "telegram-bot/internal/domain/user"
    "telegram-bot/internal/handler"
)

type VersionHandler struct {
    *BaseCommand
}

func NewVersionHandler(groupRepo GroupRepository) *VersionHandler {
    return &VersionHandler{
        BaseCommand: NewBaseCommand(
            "version",
            "查看机器人版本",
            user.PermissionUser,
            nil,  // 支持所有聊天类型
            groupRepo,
        ),
    }
}

func (h *VersionHandler) Handle(ctx *handler.Context) error {
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }
    return ctx.Reply("Bot Version: v2.0.0")
}

// cmd/bot/main.go
router.Register(command.NewVersionHandler(groupRepo))
```

### 2. 添加新中间件

**步骤**:

1. 创建中间件文件 `internal/middleware/myMiddleware.go`
2. 实现 `Middleware func(HandlerFunc) HandlerFunc`
3. 在 `cmd/bot/main.go` 中使用 `router.Use()` 注册

**示例**:

```go
// internal/middleware/timing.go
package middleware

import (
    "telegram-bot/internal/handler"
    "time"
)

type TimingMiddleware struct {
    logger Logger
}

func NewTimingMiddleware(logger Logger) *TimingMiddleware {
    return &TimingMiddleware{logger: logger}
}

func (m *TimingMiddleware) Middleware() handler.Middleware {
    return func(next handler.HandlerFunc) handler.HandlerFunc {
        return func(ctx *handler.Context) error {
            start := time.Now()
            err := next(ctx)
            duration := time.Since(start)
            m.logger.Info("handler_duration", "ms", duration.Milliseconds())
            return err
        }
    }
}

// cmd/bot/main.go
router.Use(middleware.NewTimingMiddleware(appLogger).Middleware())
```

### 3. 添加关键词处理器

```go
// internal/handlers/keyword/thanks.go
package keyword

import (
    "strings"
    "telegram-bot/internal/handler"
)

type ThanksHandler struct {
    keywords []string
}

func NewThanksHandler() *ThanksHandler {
    return &ThanksHandler{
        keywords: []string{"谢谢", "thanks"},
    }
}

func (h *ThanksHandler) Match(ctx *handler.Context) bool {
    text := strings.ToLower(ctx.Text)
    for _, kw := range h.keywords {
        if strings.Contains(text, kw) {
            return true
        }
    }
    return false
}

func (h *ThanksHandler) Handle(ctx *handler.Context) error {
    return ctx.Reply("不客气！😊")
}

func (h *ThanksHandler) Priority() int { return 200 }
func (h *ThanksHandler) ContinueChain() bool { return true }

// cmd/bot/main.go
router.Register(keyword.NewThanksHandler())
```

### 4. 添加正则处理器

```go
// internal/handlers/pattern/email.go
package pattern

import (
    "regexp"
    "telegram-bot/internal/handler"
)

type EmailHandler struct {
    pattern *regexp.Regexp
}

func NewEmailHandler() *EmailHandler {
    return &EmailHandler{
        pattern: regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
    }
}

func (h *EmailHandler) Match(ctx *handler.Context) bool {
    return h.pattern.MatchString(ctx.Text)
}

func (h *EmailHandler) Handle(ctx *handler.Context) error {
    emails := h.pattern.FindAllString(ctx.Text, -1)
    return ctx.Reply(fmt.Sprintf("检测到 %d 个邮箱地址", len(emails)))
}

func (h *EmailHandler) Priority() int { return 300 }
func (h *EmailHandler) ContinueChain() bool { return false }

// cmd/bot/main.go
router.Register(pattern.NewEmailHandler())
```

---

## 最佳实践

### 1. 处理器设计

✅ **DO**:
- 保持处理器无状态（stateless）
- 使用依赖注入传递仓储、服务等依赖
- 在 `Handle()` 开头显式调用 `CheckPermission()`
- 合理设置 `ContinueChain()` 的返回值

❌ **DON'T**:
- 在处理器中存储可变状态
- 在 `Match()` 中执行耗时操作（如数据库查询）
- 忘记检查权限
- 在监听器中返回 `ContinueChain() = false`

### 2. 优先级分配

```go
// 系统级（紧急操作）
const PriorityUrgent = 0

// 命令处理器
const PriorityCommand = 100

// 关键词处理器
const PriorityKeyword = 200

// 正则处理器
const PriorityPattern = 300

// 监听器（日志、分析）
const PriorityListener = 900
```

### 3. 错误处理

```go
func (h *MyHandler) Handle(ctx *handler.Context) error {
    // 1. 参数验证
    args := command.ParseArgs(ctx.Text)
    if len(args) < 1 {
        return ctx.Reply("❌ 参数不足")  // 用户友好的错误
    }

    // 2. 权限检查
    if err := h.CheckPermission(ctx); err != nil {
        return err  // 框架会自动回复权限错误
    }

    // 3. 业务逻辑
    result, err := h.doSomething(args[0])
    if err != nil {
        h.logger.Error("business_error", "error", err)  // 记录详细错误
        return ctx.Reply("❌ 操作失败，请稍后再试")        // 用户友好的错误
    }

    // 4. 成功响应
    return ctx.Reply("✅ " + result)
}
```

### 4. 测试

```go
// 单元测试
func TestMyHandler(t *testing.T) {
    // 创建 mock 依赖
    mockRepo := &MockGroupRepo{}

    // 创建处理器
    h := NewMyHandler(mockRepo)

    // 创建测试上下文
    ctx := &handler.Context{
        Text:     "/mycommand arg1",
        ChatType: "private",
        User:     user.NewUser(123, "test", "Test", "User"),
    }

    // 测试匹配
    assert.True(t, h.Match(ctx))

    // 测试处理
    err := h.Handle(ctx)
    assert.NoError(t, err)
}
```

### 5. 性能优化

- ✅ 使用 MongoDB 索引（`internal/adapter/repository/mongodb/indexes.go`）
- ✅ 合理设置连接池大小
- ✅ 避免在 `Match()` 中执行数据库查询
- ✅ 使用中间件缓存用户信息（`PermissionMiddleware`）

---

## 部署架构

### 生产环境

```
┌─────────────┐
│   Telegram  │
│   Servers   │
└──────┬──────┘
       │ HTTPS
       ▼
┌─────────────────────┐
│  Bot Instance       │
│  (Docker Container) │
│                     │
│  ┌───────────────┐  │
│  │  Router       │  │
│  │  Handlers     │  │
│  │  Middleware   │  │
│  └───────┬───────┘  │
│          │          │
└──────────┼──────────┘
           │
           ▼
┌─────────────────────┐
│  MongoDB Atlas      │
│  (Cloud Database)   │
│                     │
│  - M0 Free Tier     │
│  - Auto Backup      │
│  - Global Access    │
└─────────────────────┘
```

### Docker 部署

```yaml
# docker-compose.yml
version: '3.8'

services:
  bot:
    build: .
    environment:
      - TELEGRAM_TOKEN=${TELEGRAM_TOKEN}
      - MONGO_URI=${MONGO_URI}
      - LOG_LEVEL=info
    restart: unless-stopped
    networks:
      - bot-network

networks:
  bot-network:
    driver: bridge
```

### 健康检查

系统提供了健康检查端点（如果启用 HTTP 服务器）：

- `/health` - 基本健康状态
- `/health/db` - 数据库连接状态

---

## 总结

本架构设计的核心优势：

1. **简洁清晰**: Handler 接口统一所有处理逻辑
2. **易于扩展**: 添加新功能只需实现 Handler 接口
3. **职责分离**: Router、Handler、Middleware 各司其职
4. **生产可用**: 完善的错误处理、日志、监控
5. **高可测试**: 接口驱动，易于模拟和测试

通过这种架构，开发者可以快速构建功能丰富、稳定可靠的 Telegram 机器人应用。

---

## 参考资料

- [Handler 接口源码](../internal/handler/handler.go)
- [Router 源码](../internal/handler/router.go)
- [BaseCommand 源码](../internal/handlers/command/base.go)
- [快速入门指南](./getting-started.md)
- [开发者 API 文档](./developer-api.md)

---

**文档版本**: v2.0.0
**最后更新**: 2025-10-04
**维护者**: Telegram Bot Development Team
