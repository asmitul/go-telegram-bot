# Telegram Bot 架构设计文档

## 目录

- [系统概述](#系统概述)
- [架构原则](#架构原则)
- [分层架构](#分层架构)
- [核心组件](#核心组件)
- [数据流图](#数据流图)
- [序列图](#序列图)
- [技术栈](#技术栈)
- [扩展指南](#扩展指南)
- [最佳实践](#最佳实践)

---

## 系统概述

本项目是一个基于 **Clean Architecture（整洁架构）** 设计的 Telegram 群组管理机器人，采用 Go 语言开发。

### 核心特性

- ✅ **分层架构**: Domain → UseCase → Adapter → Commands
- ✅ **依赖倒置**: 高层模块不依赖低层模块，都依赖抽象
- ✅ **可测试性**: 单元测试覆盖率 > 80%
- ✅ **可扩展性**: 插件式命令系统
- ✅ **高性能**: 缓存、连接池、并发优化
- ✅ **可维护性**: 清晰的代码结构和文档

### 系统目标

1. **功能完整**: 提供群组管理所需的所有功能
2. **性能优良**: 高并发、低延迟响应
3. **易于扩展**: 快速添加新命令和功能
4. **运维友好**: 完善的监控、日志和健康检查

---

## 架构原则

### 1. Clean Architecture

遵循 Robert C. Martin 的整洁架构原则：

```
                    ┌─────────────────┐
                    │   Frameworks    │
                    │   & Drivers     │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  Interface      │
                    │  Adapters       │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  Application    │
                    │  Business Rules │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │   Enterprise    │
                    │  Business Rules │
                    └─────────────────┘
```

### 2. 依赖规则

- **内层不依赖外层**: Domain 不依赖 Adapter
- **外层依赖内层**: Adapter 依赖 UseCase，UseCase 依赖 Domain
- **依赖抽象**: 通过接口实现依赖倒置

### 3. 单一职责

每个模块只负责一个功能领域：
- **Domain**: 业务实体和规则
- **UseCase**: 应用业务逻辑
- **Adapter**: 外部接口适配
- **Commands**: 具体命令实现

---

## 分层架构

### 目录结构

```
telegram-bot/
├── cmd/
│   └── bot/
│       └── main.go              # 应用入口
├── internal/
│   ├── domain/                  # 领域层（最内层）
│   │   ├── user/               # 用户领域
│   │   ├── group/              # 群组领域
│   │   └── command/            # 命令领域
│   ├── usecase/                # 用例层
│   │   ├── user/               # 用户用例
│   │   └── group/              # 群组用例
│   ├── adapter/                # 适配器层
│   │   ├── repository/         # 数据仓储
│   │   │   ├── mongodb/       # MongoDB 实现
│   │   │   └── memory/        # 内存实现（测试用）
│   │   ├── telegram/          # Telegram API 适配
│   │   ├── cache/             # 缓存适配
│   │   ├── health/            # 健康检查
│   │   ├── metrics/           # 监控指标
│   │   └── ratelimit/         # 限流
│   ├── commands/               # 命令层
│   │   ├── ping/              # Ping 命令
│   │   ├── help/              # Help 命令
│   │   ├── ban/               # Ban 命令
│   │   ├── mute/              # Mute 命令
│   │   ├── warn/              # Warn 命令
│   │   ├── stats/             # Stats 命令
│   │   ├── admin/             # Admin 命令
│   │   ├── manage/            # Manage 命令
│   │   └── welcome/           # Welcome 命令
│   └── config/                 # 配置
├── pkg/                        # 公共包
│   ├── logger/                # 日志
│   ├── errors/                # 错误处理
│   └── validator/             # 验证器
├── test/                       # 测试
│   ├── mocks/                 # Mock 对象
│   └── integration/           # 集成测试
└── docs/                       # 文档
    ├── api.md                 # API 文档
    └── architecture.md        # 架构文档
```

### 层次说明

#### 1️⃣ Domain Layer (领域层)

**职责**: 定义业务实体和核心业务规则

**组件**:
- `user.User`: 用户实体
- `group.Group`: 群组实体
- `user.Permission`: 权限枚举
- `command.Handler`: 命令处理器接口

**特点**:
- ✅ 不依赖任何外部框架
- ✅ 包含业务逻辑
- ✅ 可独立测试

**示例**:
```go
// internal/domain/user/user.go
type User struct {
    ID          int64
    Username    string
    Permissions map[int64]Permission
}

func (u *User) HasPermission(groupID int64, required Permission) bool {
    return u.GetPermission(groupID) >= required
}
```

#### 2️⃣ UseCase Layer (用例层)

**职责**: 协调领域对象完成具体业务流程

**组件**:
- `user.CreateUserUseCase`: 创建用户用例
- `group.UpdateGroupSettingsUseCase`: 更新群组设置用例

**特点**:
- ✅ 编排业务流程
- ✅ 依赖 Domain 接口
- ✅ 不关心数据存储细节

**示例**:
```go
// internal/usecase/user/create_user.go
type CreateUserUseCase struct {
    userRepo user.Repository
}

func (uc *CreateUserUseCase) Execute(userID int64, username string) error {
    u := user.NewUser(userID, username, "", "")
    return uc.userRepo.Save(u)
}
```

#### 3️⃣ Adapter Layer (适配器层)

**职责**: 实现外部接口，连接外部系统

**组件**:
- **Repository**: MongoDB/Memory 数据存储
- **Telegram**: Telegram API 封装
- **Cache**: Redis/Memory 缓存
- **Health**: 健康检查
- **Metrics**: Prometheus 监控
- **RateLimit**: 限流器

**特点**:
- ✅ 实现 Domain 定义的接口
- ✅ 处理外部系统细节
- ✅ 可替换实现

**示例**:
```go
// internal/adapter/repository/mongodb/user_repository.go
type UserRepository struct {
    collection *mongo.Collection
}

func (r *UserRepository) FindByID(id int64) (*user.User, error) {
    // MongoDB 实现细节
}
```

#### 4️⃣ Commands Layer (命令层)

**职责**: 实现具体的 Telegram 命令

**组件**:
- 每个命令一个包 (ping, ban, mute, etc.)
- 实现 `command.Handler` 接口

**特点**:
- ✅ 处理用户输入
- ✅ 调用 UseCase
- ✅ 返回响应

**示例**:
```go
// internal/commands/ban/handler.go
type Handler struct {
    groupRepo group.Repository
    userRepo  user.Repository
    api       *telegram.API
}

func (h *Handler) Handle(ctx *command.Context) error {
    // 1. 解析参数
    // 2. 权限检查
    // 3. 执行封禁
    // 4. 返回结果
}
```

---

## 核心组件

### 1. 命令注册系统

**Registry Pattern**: 集中管理所有命令

```go
type Registry interface {
    Register(handler Handler)
    Get(name string) (Handler, bool)
    GetAll() map[string]Handler
}

// 使用示例
registry := command.NewRegistry()
registry.Register(ping.NewHandler(groupRepo))
registry.Register(ban.NewHandler(groupRepo, userRepo, api))
```

### 2. 中间件系统

**责任链模式**: 请求依次通过多个中间件

```
Request → Logging → RateLimit → Permission → Handler → Response
```

**中间件类型**:
- `LoggingMiddleware`: 记录命令执行日志
- `PermissionMiddleware`: 检查权限和命令启用状态
- `RateLimitMiddleware`: 防止滥用

**实现**:
```go
type Middleware func(HandlerFunc) HandlerFunc

func Chain(middlewares ...Middleware) Middleware {
    return func(next HandlerFunc) HandlerFunc {
        for i := len(middlewares) - 1; i >= 0; i-- {
            next = middlewares[i](next)
        }
        return next
    }
}
```

### 3. 仓储模式

**Repository Pattern**: 抽象数据访问

```go
type Repository interface {
    FindByID(id int64) (*User, error)
    Save(user *User) error
    Delete(id int64) error
}

// 实现可以是 MongoDB, PostgreSQL, Memory 等
```

### 4. 缓存系统

**Cache-Aside Pattern**: 旁路缓存

```
1. 查询缓存
2. 缓存命中 → 返回
3. 缓存未命中 → 查询数据库
4. 更新缓存
5. 返回结果
```

**实现**:
- `RedisCache`: 生产环境
- `MemoryCache`: 测试环境
- `UserCache`, `GroupCache`: 业务缓存

### 5. 重试机制

**Exponential Backoff**: 指数退避重试

```go
type Retrier interface {
    Do(ctx context.Context, fn RetryableFunc) error
}

// 配置
config := &RetryConfig{
    MaxRetries:   3,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     10 * time.Second,
    Multiplier:   2.0,
}
```

---

## 数据流图

### 1. 命令处理流程

```
┌─────────┐
│ Telegram│
│  Update │
└────┬────┘
     │
     ▼
┌─────────────────┐
│  Bot Handler    │
│  (main.go)      │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│ HandleUpdate    │
│ (bot_handler)   │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│ Parse Command   │
│ & Get Handler   │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│ Middleware      │
│ Chain           │
├─────────────────┤
│ 1. Logging      │
│ 2. RateLimit    │
│ 3. Permission   │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│ Command Handler │
│ (Handle)        │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│ UseCase         │
│ (Business Logic)│
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│ Repository      │
│ (Data Access)   │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│ MongoDB / Cache │
└─────────────────┘
```

### 2. 缓存数据流

```
┌──────────┐
│ Request  │
└────┬─────┘
     │
     ▼
┌──────────────┐      Yes     ┌──────────┐
│ Check Cache  │─────────────→│  Return  │
└──────┬───────┘               └──────────┘
       │ No
       ▼
┌──────────────┐
│ Query DB     │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Update Cache │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│    Return    │
└──────────────┘
```

### 3. 权限检查流程

```
┌──────────────┐
│ User Request │
└──────┬───────┘
       │
       ▼
┌──────────────────┐       No      ┌──────────┐
│ Command Enabled? │───────────────→│  Reject  │
└──────┬───────────┘                └──────────┘
       │ Yes
       ▼
┌──────────────────┐       No      ┌──────────┐
│ Get User Info    │───────────────→│  Create  │
└──────┬───────────┘                │   User   │
       │ Found                       └──────────┘
       ▼
┌──────────────────┐       No      ┌──────────┐
│ Check Permission │───────────────→│  Reject  │
└──────┬───────────┘                └──────────┘
       │ Pass
       ▼
┌──────────────────┐
│ Execute Command  │
└──────────────────┘
```

---

## 序列图

### 1. 用户封禁流程

```
User        Bot         Handler      UseCase     Repository    Telegram API
 │           │            │            │              │              │
 │─/ban──────→│           │            │              │              │
 │           │            │            │              │              │
 │           │─Parse─────→│            │              │              │
 │           │            │            │              │              │
 │           │←Command────│            │              │              │
 │           │            │            │              │              │
 │           │─Middleware→│            │              │              │
 │           │  Chain     │            │              │              │
 │           │            │            │              │              │
 │           │←Permission─│            │              │              │
 │           │  Check     │            │              │              │
 │           │            │            │              │              │
 │           │            │─Handle────→│              │              │
 │           │            │            │              │              │
 │           │            │            │─Get User────→│              │
 │           │            │            │              │              │
 │           │            │            │←User─────────│              │
 │           │            │            │              │              │
 │           │            │            │─Ban API──────────────────→  │
 │           │            │            │              │              │
 │           │            │            │←Success──────────────────── │
 │           │            │            │              │              │
 │           │            │            │─Save Record─→│              │
 │           │            │            │              │              │
 │           │            │←Success────│              │              │
 │           │            │            │              │              │
 │           │←Response───│            │              │              │
 │           │            │            │              │              │
 │←Message───│            │            │              │              │
 │           │            │            │              │              │
```

### 2. 缓存查询流程

```
Handler    UserCache    Cache       Repository    MongoDB
  │           │           │              │            │
  │─Get User─→│           │              │            │
  │           │           │              │            │
  │           │─Get──────→│              │            │
  │           │           │              │            │
  │           │←Hit───────│              │            │
  │           │           │              │            │
  │←User──────│           │              │            │
  │           │           │              │            │

  (Cache Miss Scenario)
  │           │           │              │            │
  │─Get User─→│           │              │            │
  │           │           │              │            │
  │           │─Get──────→│              │            │
  │           │           │              │            │
  │           │←Miss──────│              │            │
  │           │           │              │            │
  │           │─Find─────────────────────→│           │
  │           │           │              │            │
  │           │           │              │─Query─────→│
  │           │           │              │            │
  │           │           │              │←Data───────│
  │           │           │              │            │
  │           │←User──────────────────────│           │
  │           │           │              │            │
  │           │─Set──────→│              │            │
  │           │           │              │            │
  │           │←OK────────│              │            │
  │           │           │              │            │
  │←User──────│           │              │            │
  │           │           │              │            │
```

### 3. 新成员欢迎流程

```
Telegram    Bot      Handler    GroupRepo   Cache
   │         │          │           │          │
   │─New─────→│         │           │          │
   │ Member  │          │           │          │
   │         │          │           │          │
   │         │─Handle──→│           │          │
   │         │          │           │          │
   │         │          │─Get───────→│         │
   │         │          │  Config    │         │
   │         │          │           │          │
   │         │          │           │─Check───→│
   │         │          │           │  Cache   │
   │         │          │           │          │
   │         │          │           │←Miss─────│
   │         │          │           │          │
   │         │          │           │─Query───→│
   │         │          │           │  DB      │
   │         │          │           │          │
   │         │          │←Config────│          │
   │         │          │           │          │
   │         │          │─Format────┐          │
   │         │          │  Message  │          │
   │         │          │           │          │
   │         │          │←Formatted─┘          │
   │         │          │           │          │
   │         │←Message──│           │          │
   │         │          │           │          │
   │←Send────│          │           │          │
   │ Welcome │          │           │          │
   │         │          │           │          │
```

---

## 技术栈

### 核心框架

| 组件 | 技术 | 版本 | 用途 |
|-----|------|-----|------|
| 语言 | Go | 1.21+ | 主要开发语言 |
| Bot SDK | go-telegram/bot | latest | Telegram Bot API |
| 数据库 | MongoDB | 4.4+ | 数据持久化 |
| 缓存 | Redis | 6.0+ | 数据缓存 |

### 基础设施

| 组件 | 技术 | 用途 |
|-----|------|-----|
| 监控 | Prometheus | 指标采集 |
| 日志 | 结构化日志 | 日志记录 |
| 测试 | testify, gomock | 单元测试 |

### 第三方库

```go
require (
    github.com/go-telegram/bot v1.x.x
    go.mongodb.org/mongo-driver v1.x.x
    github.com/redis/go-redis/v9 v9.x.x
    github.com/prometheus/client_golang v1.x.x
    github.com/stretchr/testify v1.x.x
    go.uber.org/mock v0.x.x
)
```

---

## 扩展指南

### 1. 添加新命令

**步骤**:

1. **创建命令包**
```bash
mkdir internal/commands/mycommand
```

2. **实现 Handler 接口**
```go
// internal/commands/mycommand/handler.go
package mycommand

import "telegram-bot/internal/domain/command"

type Handler struct {
    // 依赖注入
}

func NewHandler(deps...) *Handler {
    return &Handler{...}
}

func (h *Handler) Name() string {
    return "mycommand"
}

func (h *Handler) Description() string {
    return "My command description"
}

func (h *Handler) Usage() string {
    return "/mycommand [args]"
}

func (h *Handler) RequiredPermission() user.Permission {
    return user.PermissionUser
}

func (h *Handler) IsEnabled(groupID int64) bool {
    // 检查命令是否在群组中启用
    return true
}

func (h *Handler) Handle(ctx *command.Context) error {
    // 实现命令逻辑
    return nil
}
```

3. **注册命令**
```go
// cmd/bot/main.go
func registerCommands(registry command.Registry, ...) {
    // ... 其他命令
    registry.Register(mycommand.NewHandler(...))
}
```

4. **编写测试**
```go
// internal/commands/mycommand/handler_test.go
func TestHandler_Handle(t *testing.T) {
    // 测试用例
}
```

### 2. 添加新仓储实现

**步骤**:

1. **实现 Repository 接口**
```go
// internal/adapter/repository/postgres/user_repository.go
package postgres

import "telegram-bot/internal/domain/user"

type UserRepository struct {
    db *sql.DB
}

func (r *UserRepository) FindByID(id int64) (*user.User, error) {
    // PostgreSQL 实现
}

func (r *UserRepository) Save(u *user.User) error {
    // PostgreSQL 实现
}
```

2. **依赖注入**
```go
// cmd/bot/main.go
userRepo := postgres.NewUserRepository(db)
```

### 3. 添加新中间件

**步骤**:

1. **实现中间件函数**
```go
// internal/adapter/telegram/my_middleware.go
package telegram

type MyMiddleware struct {
    // 配置
}

func NewMyMiddleware() *MyMiddleware {
    return &MyMiddleware{}
}

func (m *MyMiddleware) Process() Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx *command.Context) error {
            // 前置处理

            err := next(ctx)

            // 后置处理

            return err
        }
    }
}
```

2. **添加到中间件链**
```go
// internal/adapter/telegram/bot_handler.go
middlewares := []Middleware{
    logMiddleware.Log(),
    myMiddleware.Process(),  // 新中间件
    permMiddleware.Check(handler),
}
```

### 4. 添加新用例

**步骤**:

1. **定义用例接口**
```go
// internal/usecase/my_usecase.go
package usecase

type MyUseCase interface {
    Execute(params) error
}
```

2. **实现用例**
```go
type myUseCaseImpl struct {
    repo Repository
}

func NewMyUseCase(repo Repository) MyUseCase {
    return &myUseCaseImpl{repo: repo}
}

func (uc *myUseCaseImpl) Execute(params) error {
    // 业务逻辑
}
```

3. **在命令中使用**
```go
func (h *Handler) Handle(ctx *command.Context) error {
    return h.useCase.Execute(params)
}
```

### 5. 扩展领域模型

**步骤**:

1. **添加新实体**
```go
// internal/domain/myentity/entity.go
package myentity

type MyEntity struct {
    ID        int64
    Name      string
    CreatedAt time.Time
}

func NewMyEntity(id int64, name string) *MyEntity {
    return &MyEntity{
        ID:        id,
        Name:      name,
        CreatedAt: time.Now(),
    }
}
```

2. **定义仓储接口**
```go
type Repository interface {
    FindByID(id int64) (*MyEntity, error)
    Save(entity *MyEntity) error
}
```

3. **实现仓储**
```go
// internal/adapter/repository/mongodb/myentity_repository.go
type MyEntityRepository struct {
    collection *mongo.Collection
}
```

---

## 最佳实践

### 1. 代码组织

- ✅ **包命名**: 使用单数名词 (user, group)
- ✅ **接口定义**: 在使用方定义接口（依赖倒置）
- ✅ **错误处理**: 使用 `pkg/errors` 包装错误
- ✅ **配置管理**: 集中在 `internal/config`

### 2. 测试策略

- ✅ **单元测试**: 每个 Handler 都有测试
- ✅ **Mock 对象**: 使用 gomock 生成 mock
- ✅ **表驱动测试**: 使用 table-driven tests
- ✅ **覆盖率**: 保持 > 80%

### 3. 性能优化

- ✅ **缓存策略**: 用户权限、群组配置
- ✅ **批量操作**: 使用 GetMulti/SetMulti
- ✅ **连接池**: MongoDB、Redis 连接池
- ✅ **并发控制**: WaitGroup、Context

### 4. 监控运维

- ✅ **日志级别**: Debug/Info/Warn/Error
- ✅ **结构化日志**: 使用键值对
- ✅ **指标收集**: Prometheus metrics
- ✅ **健康检查**: HTTP endpoints

### 5. 错误处理

- ✅ **错误分类**: 权限错误、参数错误、业务错误
- ✅ **错误包装**: 使用 fmt.Errorf 包装
- ✅ **用户友好**: 返回清晰的错误消息
- ✅ **日志记录**: 记录详细错误信息

### 6. 安全性

- ✅ **权限检查**: 每个命令都检查权限
- ✅ **限流保护**: RateLimitMiddleware
- ✅ **输入验证**: 验证所有用户输入
- ✅ **SQL 注入防护**: 使用参数化查询

---

## 部署架构

### 生产环境

```
┌─────────────┐
│   Telegram  │
│   Servers   │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│  Load Balancer      │
│  (Optional)         │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│  Bot Instance 1     │◄──┐
├─────────────────────┤   │
│  - Command Handler  │   │
│  - Middleware       │   │
│  - Cache Client     │   │
└──────┬──────────────┘   │
       │                  │
       ▼                  │
┌─────────────────────┐   │
│  Bot Instance N     │   │  Horizontal
├─────────────────────┤   │  Scaling
│  - Command Handler  │   │
│  - Middleware       │   │
│  - Cache Client     │◄──┘
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│  Data Layer         │
├─────────────────────┤
│  - MongoDB Cluster  │
│  - Redis Cluster    │
└─────────────────────┘
```

### 监控架构

```
┌─────────────┐
│   Bot App   │
└──────┬──────┘
       │
       │ Metrics
       ▼
┌─────────────┐
│ Prometheus  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Grafana    │
│  Dashboard  │
└─────────────┘
```

---

## 总结

本架构设计遵循以下核心原则：

1. **分层清晰**: Domain → UseCase → Adapter → Commands
2. **依赖倒置**: 高层不依赖低层，都依赖抽象
3. **可测试性**: 通过接口和依赖注入实现高可测试性
4. **可扩展性**: 插件式命令系统，易于添加新功能
5. **高性能**: 缓存、连接池、并发优化
6. **可维护性**: 清晰的代码结构和完善的文档

通过这种架构，我们实现了一个**可靠、高效、易于维护和扩展**的 Telegram Bot 系统。

---

## 参考资料

- [Clean Architecture (Robert C. Martin)](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Go Telegram Bot API](https://github.com/go-telegram/bot)
- [MongoDB Go Driver](https://www.mongodb.com/docs/drivers/go/current/)
- [Go Redis Client](https://redis.uptrace.dev/)
- [Prometheus Go Client](https://prometheus.io/docs/guides/go-application/)

---

**文档版本**: v1.0.0
**最后更新**: 2025-10-01
**维护者**: Development Team
